package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/Azure/draft/pkg/config"
	dryrunpkg "github.com/Azure/draft/pkg/dryrun"
	"github.com/Azure/draft/pkg/filematches"
	"github.com/Azure/draft/pkg/handlers"
	"github.com/Azure/draft/pkg/linguist"
	"github.com/Azure/draft/pkg/prompts"
	"github.com/Azure/draft/pkg/reporeader"
	"github.com/Azure/draft/pkg/reporeader/readers"
	"github.com/Azure/draft/pkg/templatewriter"
	"github.com/Azure/draft/pkg/templatewriter/writers"
	"github.com/Azure/draft/template"
)

// ErrNoLanguageDetected is raised when `draft create` does not detect source
// code for linguist to classify, or if there are no packs available for the detected languages.
var ErrNoLanguageDetected = errors.New("no supported languages were detected")
var flagVariablesMap = make(map[string]string)

const LANGUAGE_VARIABLE = "LANGUAGE"
const TWO_SPACES = "  "

// Flag defaults
const emptyDefaultFlagValue = ""
const currentDirDefaultFlagValue = "."

const DOCKERFILES_DIR = "dockerfiles"

func listSupportedLanguages() ([]string, error) {
	var supportedLanguages []string
	entries, err := template.Templates.ReadDir(DOCKERFILES_DIR)
	if err != nil {
		return supportedLanguages, fmt.Errorf("reading supported languages from embedded fs: %w", err)
	}
	for _, d := range entries {
		if d.IsDir() {
			supportedLanguages = append(supportedLanguages, d.Name())
		}
	}
	return supportedLanguages, nil
}

type createCmd struct {
	lang       string
	dest       string
	deployType string

	dockerfileOnly    bool
	deploymentOnly    bool
	skipFileDetection bool
	flagVariables     []string

	createConfigPath string
	createConfig     *CreateConfig

	templateWriter           templatewriter.TemplateWriter
	templateVariableRecorder config.TemplateVariableRecorder
	repoReader               reporeader.RepoReader
}

func newCreateCmd() *cobra.Command {
	cc := &createCmd{}

	cmd := &cobra.Command{
		Use:   "create [flags]",
		Short: "Add minimum required files to the directory",
		Long:  "This command will add the minimum required files to the local directory for your Kubernetes deployment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cc.initConfig(); err != nil {
				return err
			}
			return cc.run()
		},
	}

	f := cmd.Flags()

	f.StringVarP(&cc.createConfigPath, "create-config", "c", emptyDefaultFlagValue, "specify the path to the configuration file")
	f.StringVarP(&cc.lang, "language", "l", emptyDefaultFlagValue, "specify the language used to create the Kubernetes deployment")
	f.StringVarP(&cc.dest, "destination", "d", currentDirDefaultFlagValue, "specify the path to the project directory")
	f.StringVarP(&cc.deployType, "deploy-type", "", emptyDefaultFlagValue, "specify deployment type (eg. helm, kustomize, manifests)")
	f.BoolVar(&cc.dockerfileOnly, "dockerfile-only", false, "only create Dockerfile in the project directory")
	f.BoolVar(&cc.deploymentOnly, "deployment-only", false, "only create deployment files in the project directory")
	f.BoolVar(&cc.skipFileDetection, "skip-file-detection", false, "skip file detection step")
	f.StringArrayVarP(&cc.flagVariables, "variable", "", []string{}, "pass template variables (e.g. --variable PORT=8080 --variable APPNAME=test)")

	return cmd
}

func (cc *createCmd) initConfig() error {
	if cc.createConfigPath != "" {
		log.Debug("loading config")
		configBytes, err := os.ReadFile(cc.createConfigPath)
		if err != nil {
			return err
		}

		var cfg CreateConfig
		if err = yaml.Unmarshal(configBytes, &cfg); err != nil {
			return err
		}
		cc.createConfig = &cfg
		return nil
	}

	//TODO: create a config for the user and save it for subsequent uses
	cc.createConfig = &CreateConfig{}

	return nil
}

func (cc *createCmd) run() error {
	log.Debugf("config: %s", cc.createConfigPath)

	flagVariablesMap = flagVariablesToMap(cc.flagVariables)

	var dryRunRecorder *dryrunpkg.DryRunRecorder
	if dryRun {
		dryRunRecorder = dryrunpkg.NewDryRunRecorder()
		cc.templateVariableRecorder = dryRunRecorder
		cc.templateWriter = dryRunRecorder
	} else {
		cc.templateWriter = &writers.LocalFSWriter{}
	}
	cc.repoReader = &readers.LocalFSReader{}

	detectedLangDraftConfig, languageName, err := cc.detectLanguage()
	if err != nil {
		return err
	}

	err = cc.createFiles(detectedLangDraftConfig, languageName)
	if dryRun {
		cc.templateVariableRecorder.Record(LANGUAGE_VARIABLE, languageName)
		dryRunText, err := json.MarshalIndent(dryRunRecorder.DryRunInfo, "", TWO_SPACES)
		if err != nil {
			return err
		}
		fmt.Println(string(dryRunText))
		if dryRunFile != "" {
			log.Printf("writing dry run info to file %s", dryRunFile)
			err = os.WriteFile(dryRunFile, dryRunText, 0644)
			if err != nil {
				return err
			}
		}
	}
	return err
}

// detectLanguage detects the language used in a project destination directory
// It returns the DraftConfig for that language and the name of the language
func (cc *createCmd) detectLanguage() (*handlers.Template, string, error) {
	hasGo := false
	hasGoMod := false
	var langs []*linguist.Language
	var err error
	supportedLanguages, err := listSupportedLanguages()
	if err != nil {
		log.Errorf("loading supported languages: %s", err.Error())
	}
	log.Debugf("loaded supported languages: %v", supportedLanguages)
	if cc.createConfig.LanguageType == "" {
		if cc.lang != "" {
			cc.createConfig.LanguageType = cc.lang
		} else {
			log.Info("--- Detecting Language ---")
			langs, err = linguist.ProcessDir(cc.dest)
			log.Debugf("linguist.ProcessDir(%v) result:\n\nError: %v", cc.dest, err)
			if err != nil {
				return nil, "", fmt.Errorf("there was an error detecting the language: %s", err)
			}
			if len(langs) == 0 {
				langs, err = promptLanguageSelection(supportedLanguages)
				if err != nil {
					return nil, "", fmt.Errorf("prompting for language: %w", err)
				}
			}
			for _, lang := range langs {
				log.Debugf("%s:\t%f (%s)", lang.Language, lang.Percent, lang.Color)
				// For now let's check here for weird stuff like go module support
				if lang.Language == "Go" {
					hasGo = true

					selection := &promptui.Select{
						Label: "Linguist detected Go, do you use Go Modules?",
						Items: []string{"yes", "no"},
					}

					_, selectResponse, err := selection.Run()
					if err != nil {
						return nil, "", err
					}

					hasGoMod = strings.EqualFold(selectResponse, "yes")
				}

				if lang.Language == "Java" {

					selection := &promptui.Select{
						Label: "Linguist detected Java, are you using maven or gradle?",
						Items: []string{"gradle", "maven", "gradlew"},
					}

					_, selectResponse, err := selection.Run()
					if err != nil {
						return nil, "", err
					}

					if selectResponse == "gradle" {
						lang.Language = "Gradle"
					} else if selectResponse == "gradlew" {
						lang.Language = "Gradlew"
					}
				}
			}

			log.Debugf("detected %d langs", len(langs))

			if len(langs) == 0 {
				return nil, "", ErrNoLanguageDetected
			}
		}
	}

	if cc.createConfig.LanguageType != "" {
		log.Debug("using configuration language")
		lowerLang := strings.ToLower(cc.createConfig.LanguageType)
		langDockerfileTemplate, err := handlers.GetTemplate(fmt.Sprintf("dockerfile-%s", lowerLang), "", cc.dest, cc.templateWriter)
		if err != nil {
			return nil, "", err
		}
		if langDockerfileTemplate == nil {
			return nil, "", fmt.Errorf("could not find a template for %s", cc.createConfig.LanguageType)
		}

		return langDockerfileTemplate, lowerLang, nil
	}

	for _, lang := range langs {
		detectedLang := linguist.Alias(lang)
		log.Infof("--> Draft detected %s (%f%%)\n", detectedLang.Language, detectedLang.Percent)
		lowerLang := strings.ToLower(detectedLang.Language)
		if handlers.IsValidTemplate(fmt.Sprintf("dockerfile-%s", lowerLang)) {
			if lowerLang == "go" && hasGo && hasGoMod {
				log.Debug("detected go and go module")
				lowerLang = "gomodule"
			}
			langDockerfileTemplate, err := handlers.GetTemplate(fmt.Sprintf("dockerfile-%s", lowerLang), "", cc.dest, cc.templateWriter)
			if err != nil {
				return nil, "", err
			}
			if langDockerfileTemplate == nil {
				return nil, "", fmt.Errorf("could not find a template for detected language %s", detectedLang.Language)
			}
			return langDockerfileTemplate, lowerLang, nil
		}
		log.Infof("--> Could not find a pack for %s. Trying to find the next likely language match...", detectedLang.Language)
	}
	return nil, "", ErrNoLanguageDetected
}

func (cc *createCmd) generateDockerfile(dockerfileTemplate *handlers.Template, lowerLang string) error {
	log.Info("--- Dockerfile Creation ---")

	// Extract language-specific defaults from repo
	extractedValues, err := dockerfileTemplate.ExtractDefaults(lowerLang, cc.repoReader)
	if err != nil {
		return err
	}

	// Check for existing duplicate defaults
	for k, v := range extractedValues {
		variableExists := false
		for i, variable := range dockerfileTemplate.Config.Variables {
			if k == variable.Name {
				variableExists = true
				dockerfileTemplate.Config.Variables[i].Default.Value = v
				break
			}
		}
		if !variableExists {
			dockerfileTemplate.Config.Variables = append(dockerfileTemplate.Config.Variables, &config.BuilderVar{
				Name: k,
				Default: config.BuilderVarDefault{
					Value: v,
				},
			})
		}
	}

	if cc.createConfig.LanguageVariables == nil {
		dockerfileTemplate.Config.VariableMapToDraftConfig(flagVariablesMap)

		if err = prompts.RunPromptsFromConfigWithSkips(dockerfileTemplate.Config); err != nil {
			return err
		}
	} else {
		err = validateConfigInputsToPrompts(dockerfileTemplate.Config, cc.createConfig.LanguageVariables)
		if err != nil {
			return err
		}
	}

	if cc.templateVariableRecorder != nil {
		for _, variable := range dockerfileTemplate.Config.Variables {
			cc.templateVariableRecorder.Record(variable.Name, variable.Value)
		}
	}

	if err = dockerfileTemplate.Generate(); err != nil {
		return fmt.Errorf("there was an error when creating the Dockerfile for language %s: %w", cc.createConfig.LanguageType, err)
	}

	log.Info("--> Creating Dockerfile...\n")
	return nil
}

func (cc *createCmd) createDeployment() error {
	log.Info("--- Deployment File Creation ---")
	var deployType string
	var deployTemplate *handlers.Template
	var err error

	if cc.createConfig.DeployType != "" {
		deployType = strings.ToLower(cc.createConfig.DeployType)
		deployTemplate, err = handlers.GetTemplate(fmt.Sprintf("deployment-%s", deployType), "", cc.dest, cc.templateWriter)
		if err != nil {
			return err
		}
		if deployTemplate == nil || deployTemplate.Config == nil {
			return errors.New("invalid deployment type")
		}
		err = validateConfigInputsToPrompts(deployTemplate.Config, cc.createConfig.DeployVariables)
		if err != nil {
			return err
		}
	} else {
		if cc.deployType == "" {
			selection := &promptui.Select{
				Label: "Select k8s Deployment Type",
				Items: []string{"helm", "kustomize", "manifests"},
			}

			_, deployType, err = selection.Run()
			if err != nil {
				return err
			}
		} else {
			deployType = cc.deployType
		}

		deployTemplate, err = handlers.GetTemplate(fmt.Sprintf("deployment-%s", deployType), "", cc.dest, cc.templateWriter)
		if err != nil {
			return err
		}

		if deployTemplate == nil || deployTemplate.Config == nil {
			return errors.New("invalid deployment type")
		}

		deployTemplate.Config.VariableMapToDraftConfig(flagVariablesMap)

		err = prompts.RunPromptsFromConfigWithSkips(deployTemplate.Config)
		if err != nil {
			return err
		}
	}

	if cc.templateVariableRecorder != nil {
		for _, variable := range deployTemplate.Config.Variables {
			cc.templateVariableRecorder.Record(variable.Name, variable.Value)
		}
	}

	log.Infof("--> Creating %s Kubernetes resources...\n", deployType)
	return deployTemplate.Generate()
}

func (cc *createCmd) createFiles(detectedLangTempalte *handlers.Template, lowerLang string) error {
	// does no further checks without file detection

	if cc.dockerfileOnly && cc.deploymentOnly {
		return errors.New("can only pass in one of --dockerfile-only and --deployment-only")
	}

	if cc.skipFileDetection {
		if !cc.deploymentOnly {
			err := cc.generateDockerfile(detectedLangTempalte, lowerLang)
			if err != nil {
				return err
			}
		}
		if !cc.dockerfileOnly {
			err := cc.createDeployment()
			if err != nil {
				return err
			}
		}
		return nil
	}

	// check if the local directory has dockerfile or charts
	hasDockerFile, hasDeploymentFiles, err := filematches.SearchDirectory(cc.dest)
	if err != nil {
		return err
	}

	// prompts user for dockerfile re-creation
	if hasDockerFile && !cc.deploymentOnly {
		selection := &promptui.Select{
			Label: "We found Dockerfile in the directory, would you like to recreate the Dockerfile?",
			Items: []string{"yes", "no"},
		}

		_, selectResponse, err := selection.Run()
		if err != nil {
			return err
		}

		hasDockerFile = strings.EqualFold(selectResponse, "no")
	}

	if cc.deploymentOnly {
		log.Info("--> --deployment-only=true, skipping Dockerfile creation...")
	} else if hasDockerFile {
		log.Info("--> Found Dockerfile in local directory, skipping Dockerfile creation...")
	} else if !cc.deploymentOnly {
		err := cc.generateDockerfile(detectedLangTempalte, lowerLang)
		if err != nil {
			return err
		}
	}

	// prompts user for deployment re-creation
	if hasDeploymentFiles && !cc.dockerfileOnly {
		selection := &promptui.Select{
			Label: "We found deployment files in the directory, would you like to create new deployment files?",
			Items: []string{"yes", "no"},
		}

		_, selectResponse, err := selection.Run()
		if err != nil {
			return err
		}

		hasDeploymentFiles = strings.EqualFold(selectResponse, "no")
	}

	if cc.dockerfileOnly {
		log.Info("--> --dockerfile-only=true, skipping deployment file creation...")
	} else if hasDeploymentFiles {
		log.Info("--> Found deployment directory in local directory, skipping deployment file creation...")
	} else if !cc.dockerfileOnly {
		err := cc.createDeployment()
		if err != nil {
			return err
		}
	}

	log.Info("Draft has successfully created deployment resources for your project 😃")
	log.Info("Use 'draft setup-gh' to set up Github OIDC.")

	return nil
}

func init() {
	rootCmd.AddCommand(newCreateCmd())
}

func validateConfigInputsToPrompts(draftConfig *config.DraftConfig, provided []UserInputs) error {
	// set inputs to provided values
	for _, providedVar := range provided {
		draftConfig.SetVariable(providedVar.Name, providedVar.Value)
	}

	return nil
}

func promptLanguageSelection(supportedLanguages []string) ([]*linguist.Language, error) {
	selection := &promptui.Select{
		Label: "Unable to detect a supported language, please select one:",
		Items: supportedLanguages,
	}
	_, selectResponse, err := selection.Run()
	if err != nil {
		return nil, fmt.Errorf("manually selecting language: %w", err)
	}
	langs := []*linguist.Language{{Language: selectResponse}}
	return langs, nil
}
