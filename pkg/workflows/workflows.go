package workflows

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"

	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/kubernetes/scheme"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/draft/pkg/config"
	"github.com/Azure/draft/pkg/embedutils"
	"github.com/Azure/draft/pkg/osutil"
	"github.com/Azure/draft/pkg/templatewriter"
)

type Workflows struct {
	workflows         map[string]fs.DirEntry
	configs           map[string]*config.DraftConfig
	Dest              string
	workflowTemplates fs.FS
}

const (
	parentDirName         = "workflows"
	configFileName        = "/draft.yaml"
	emptyDefaultFlagValue = ""
)

func UpdateProductionDeployments(deployType, dest string, draftConfig *config.DraftConfig, templateWriter templatewriter.TemplateWriter) error {
	acr, err := draftConfig.GetVariable("AZURECONTAINERREGISTRY")
	if err != nil {
		return fmt.Errorf("failed to get variable: %w", err)
	}

	repositoryName, err := draftConfig.GetVariable("ACRREPOSITORYNAME")
	if err != nil {
		return fmt.Errorf("failed to get variable: %w", err)
	}

	productionImage := fmt.Sprintf("%s.azurecr.io/%s", acr.Value, repositoryName.Value)
	switch deployType {
	case "helm":
		return setHelmContainerImage(dest+"/charts/production.yaml", productionImage, templateWriter)
	case "kustomize":
		return setDeploymentContainerImage(dest+"/overlays/production/deployment.yaml", productionImage)
	case "manifests":
		return setDeploymentContainerImage(dest+"/manifests/deployment.yaml", productionImage)
	}
	return nil
}

func setDeploymentContainerImage(filePath, productionImage string) error {

	decode := scheme.Codecs.UniversalDeserializer().Decode
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	k8sObj, _, err := decode(file, nil, nil)
	if err != nil {
		return err
	}
	deploy, ok := k8sObj.(*appsv1.Deployment)
	if !ok {
		return errors.New("could not decode kubernetes deployment")
	}

	if len(deploy.Spec.Template.Spec.Containers) != 1 {
		return errors.New("unsupported number of containers defined in the deployment spec")
	}

	deploy.Spec.Template.Spec.Containers[0].Image = productionImage

	printer := printers.YAMLPrinter{}

	out, err := os.OpenFile(filePath, os.O_RDWR, 0755)
	if err != nil {
		return nil
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Errorf("error closing file: %v", err)
		}
	}()

	return printer.PrintObj(deploy, out)
}

func setHelmContainerImage(filePath, productionImage string, templateWriter templatewriter.TemplateWriter) error {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	var deploy HelmProductionYaml
	err = yaml.Unmarshal(file, &deploy)
	if err != nil {
		return err
	}

	deploy.Image.Repository = productionImage

	out, err := yaml.Marshal(deploy)
	if err != nil {
		return err
	}

	return templateWriter.WriteFile(filePath, out)
}

func (w *Workflows) loadConfig(deployType string) (*config.DraftConfig, error) {
	val, ok := w.workflows[deployType]
	if !ok {
		return nil, fmt.Errorf("deploy type %s unsupported", deployType)
	}

	configPath := path.Join(parentDirName, val.Name(), configFileName)
	configBytes, err := fs.ReadFile(w.workflowTemplates, configPath)
	if err != nil {
		return nil, err
	}

	var draftConfig config.DraftConfig
	if err = yaml.Unmarshal(configBytes, &draftConfig); err != nil {
		return nil, err
	}

	return &draftConfig, nil
}

func (w *Workflows) GetConfig(deployType string) (*config.DraftConfig, error) {
	val, ok := w.configs[deployType]
	if !ok {
		return nil, fmt.Errorf("deploy type %s unsupported", deployType)
	}
	return val, nil
}

func CreateWorkflowsFromEmbedFS(workflowTemplates embed.FS, dest string) *Workflows {
	deployMap, err := embedutils.EmbedFStoMap(workflowTemplates, parentDirName)
	if err != nil {
		log.Fatal(err)
	}

	w := &Workflows{
		workflows:         deployMap,
		Dest:              dest,
		configs:           make(map[string]*config.DraftConfig),
		workflowTemplates: workflowTemplates,
	}
	w.populateConfigs()

	return w
}

func (w *Workflows) populateConfigs() {
	for deployType := range w.workflows {
		draftConfig, err := w.loadConfig(deployType)
		if err != nil {
			log.Debugf("no draftConfig found for workflow of deploy type %s", deployType)
			draftConfig = &config.DraftConfig{}
		}
		w.configs[deployType] = draftConfig
	}
}

func (w *Workflows) CreateWorkflowFiles(deployType string, draftConfig *config.DraftConfig, templateWriter templatewriter.TemplateWriter) error {
	val, ok := w.workflows[deployType]
	if !ok {
		return fmt.Errorf("deployment type: %s is not currently supported", deployType)
	}

	srcDir := path.Join(parentDirName, val.Name())
	log.Debugf("source directory for workflow template: %s", srcDir)

	if err := draftConfig.ApplyDefaultVariables(); err != nil {
		return fmt.Errorf("create workflow files: %w", err)
	}

	if err := osutil.CopyDir(w.workflowTemplates, srcDir, w.Dest, draftConfig, templateWriter); err != nil {
		return err
	}

	return nil
}
