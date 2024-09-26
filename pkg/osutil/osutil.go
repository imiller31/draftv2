package osutil

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"text/template"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/draft/pkg/config"
	"github.com/Azure/draft/pkg/templatewriter"
)

// A draft variable is defined as a string of non-whitespace characters wrapped in double curly braces.
var draftVariableRegex = regexp.MustCompile("{{[^\\s.]+\\S*}}")

const configFileName = "draft.yaml"

// Exists returns whether the given file or directory exists or not.
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// SymlinkWithFallback attempts to symlink a file or directory, but falls back to a move operation
// in the event of the user not having the required privileges to create the symlink.
func SymlinkWithFallback(oldname, newname string) (err error) {
	err = os.Symlink(oldname, newname)
	if runtime.GOOS == "windows" {
		// If creating the symlink fails on Windows because the user
		// does not have the required privileges, ignore the error and
		// fall back to renaming the file.
		//
		// ERROR_PRIVILEGE_NOT_HELD is 0x522:
		// https://msdn.microsoft.com/en-us/library/windows/desktop/ms681385(v=vs.85).aspx
		if lerr, ok := err.(*os.LinkError); ok && lerr.Err == syscall.Errno(0x522) {
			err = os.Rename(oldname, newname)
		}
	}
	return
}

// EnsureDirectory checks if a directory exists and creates it if it doesn't
func EnsureDirectory(dir string) error {
	if fi, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("could not create %s: %s", dir, err)
		}
	} else if !fi.IsDir() {
		return fmt.Errorf("%s must be a directory", dir)
	}

	return nil
}

// EnsureFile checks if a file exists and creates it if it doesn't
func EnsureFile(file string) error {
	fi, err := os.Stat(file)
	if err != nil {
		f, err := os.Create(file)
		if err != nil {
			return fmt.Errorf("could not create %s: %s", file, err)
		}
		defer f.Close()
	} else if fi.IsDir() {
		return fmt.Errorf("%s must not be a directory", file)
	}

	return nil
}

func CopyDir(
	fileSys fs.FS,
	src, dest string,
	draftConfig *config.DraftConfig,
	templateWriter templatewriter.TemplateWriter) error {
	files, err := fs.ReadDir(fileSys, src)
	if err != nil {
		return err
	}

	for _, f := range files {

		if f.Name() == configFileName {
			continue
		}

		fileName := f.Name()
		if overrideName, ok := draftConfig.FileNameOverrideMap[f.Name()]; ok {
			fileName = overrideName
		}
		srcPath := path.Join(src, f.Name())
		destPath := path.Join(dest, fileName)
		log.Debugf("Source path: %s Dest path: %s", srcPath, destPath)

		if f.IsDir() {
			if err = templateWriter.EnsureDirectory(destPath); err != nil {
				return err
			}
			if err = CopyDir(fileSys, srcPath, destPath, draftConfig, templateWriter); err != nil {
				return err
			}
		} else {
			fileContent, err := replaceTemplateVariables(fileSys, srcPath, draftConfig)
			if err != nil {
				return err
			}

			if err = checkAllVariablesSubstituted(string(fileContent)); err != nil {
				return fmt.Errorf("error substituting file %s: %w", srcPath, err)
			}

			if err = templateWriter.WriteFile(destPath, fileContent); err != nil {
				return err
			}
		}
	}
	return nil
}

/*
	checkAllVariablesSubstituted checks that all draft variables have been substituted.

If any draft variables are found, an error is returned.
Draft variables are defined as a string of non-whitespace characters starting with a non-period character wrapped in double curly braces.
The non-period first character constraint is used to avoid matching helm template functions.
*/
func checkAllVariablesSubstituted(fileContent string) error {
	if unsubstitutedVars := draftVariableRegex.FindAllString(fileContent, -1); len(unsubstitutedVars) > 0 {
		unsubstitutedVarsString := strings.Join(unsubstitutedVars, ", ")
		return fmt.Errorf("unsubstituted variable: %s", unsubstitutedVarsString)
	}
	return nil
}

func replaceTemplateVariables(fileSys fs.FS, srcPath string, draftConfig *config.DraftConfig) ([]byte, error) {
	file, err := fs.ReadFile(fileSys, srcPath)
	if err != nil {
		return nil, err
	}

	fileString := string(file)
	for _, variable := range draftConfig.Variables {
		log.Debugf("replacing %s with %s", variable.Name, variable.Value)
		fileString = strings.ReplaceAll(fileString, "{{"+variable.Name+"}}", variable.Value)
	}
	return []byte(fileString), nil
}

// CopyDirWithTemplates - Handles Gotemplate processing and writing
func CopyDirWithTemplates(
	fileSys fs.FS,
	src, dest string,
	draftConfig *config.DraftConfig,
	templateWriter templatewriter.TemplateWriter) error {

	files, err := fs.ReadDir(fileSys, src)
	if err != nil {
		return err
	}

	for _, f := range files {

		if f.Name() == configFileName {
			continue
		}

		fileName := f.Name()
		if overrideName, ok := draftConfig.FileNameOverrideMap[f.Name()]; ok {
			fileName = overrideName
		}

		srcPath := path.Join(src, f.Name())
		destPath := path.Join(dest, fileName)
		log.Debugf("Source path: %s Dest path: %s", srcPath, destPath)

		variableMap := draftConfig.GetVariableMap()
		if len(variableMap) == 0 {
			return fmt.Errorf("variable map is empty, unable to replace template variables")
		}

		if f.IsDir() {
			if err = templateWriter.EnsureDirectory(destPath); err != nil {
				return err
			}
			if err = CopyDirWithTemplates(fileSys, srcPath, destPath, draftConfig, templateWriter); err != nil {
				return err
			}
		} else {
			fileContent, err := replaceGoTemplateVariables(fileSys, srcPath, variableMap)
			if err != nil {
				return err
			}

			if err = templateWriter.WriteFile(destPath, fileContent); err != nil {
				return err
			}
		}
	}
	return nil
}

func replaceGoTemplateVariables(fileSys fs.FS, srcPath string, variableMap map[string]interface{}) ([]byte, error) {
	file, err := fs.ReadFile(fileSys, srcPath)
	if err != nil {
		return nil, err
	}

	// Parse the template file, missingkey=error ensures an error will be returned if any variable is missing during template execution.
	tmpl, err := template.New("template").Option("missingkey=error").Parse(string(file))
	if err != nil {
		return nil, err
	}

	// Execute the template with variableMap
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, variableMap)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
