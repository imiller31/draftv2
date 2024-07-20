package workflows

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"testing"
	"testing/fstest"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/stretchr/testify/assert"

	"github.com/Azure/draft/pkg/config"
	"github.com/Azure/draft/pkg/embedutils"
	"github.com/Azure/draft/pkg/templatewriter/writers"
	"github.com/Azure/draft/template"
)

func TestCreateWorkflows(t *testing.T) {
	dest := "."
	templatewriter := &writers.LocalFSWriter{}
	draftConfig := &config.DraftConfig{
		Variables: []*config.BuilderVar{
			{
				Name:  "WORKFLOWNAME",
				Value: "testWorkflow",
			},
			{
				Name:  "BRANCHNAME",
				Value: "testBranch",
			},
			{
				Name:  "ACRRESOURCEGROUP",
				Value: "testAcrRG",
			},
			{
				Name:  "AZURECONTAINERREGISTRY",
				Value: "testAcr",
			},
			{
				Name:  "CONTAINERNAME",
				Value: "testContainer",
			},
			{
				Name:  "CLUSTERRESOURCEGROUP",
				Value: "testClusterRG",
			},
			{
				Name:  "CLUSTERNAME",
				Value: "testCluster",
			},
			{
				Name:  "KUSTOMIZEPATH",
				Value: "./overlays/production",
			},
			{
				Name:  "DEPLOYMENTMANIFESTPATH",
				Value: "./manifests",
			},
			{
				Name:  "DOCKERFILE",
				Value: "./Dockerfile",
			},
			{
				Name:  "BUILDCONTEXTPATH",
				Value: ".",
			},
			{
				Name:  "CHARTPATH",
				Value: "testPath",
			},
			{
				Name:  "CHARTOVERRIDEPATH",
				Value: "testOverridePath",
			},
			{
				Name:  "CHARTOVERRIDES",
				Value: "replicas:2",
			},
			{
				Name:  "NAMESPACE",
				Value: "default",
			},
		},
	}
	draftConfigNoRoot := &config.DraftConfig{
		Variables: []*config.BuilderVar{
			{
				Name:  "WORKFLOWNAME",
				Value: "testWorkflow",
			},
			{
				Name:  "BRANCHNAME",
				Value: "testBranch",
			},
			{
				Name:  "ACRRESOURCEGROUP",
				Value: "testAcrRG",
			},
			{
				Name:  "AZURECONTAINERREGISTRY",
				Value: "testAcr",
			},
			{
				Name:  "CONTAINERNAME",
				Value: "testContainer",
			},
			{
				Name:  "CLUSTERRESOURCEGROUP",
				Value: "testClusterRG",
			},
			{
				Name:  "CLUSTERNAME",
				Value: "testCluster",
			},
			{
				Name:  "KUSTOMIZEPATH",
				Value: "./overlays/production",
			},
			{
				Name:  "DEPLOYMENTMANIFESTPATH",
				Value: "./manifests",
			},
			{
				Name:  "DOCKERFILE",
				Value: "./Dockerfile",
			},
			{
				Name:  "BUILDCONTEXTPATH",
				Value: "test",
			},
			{
				Name:  "CHARTPATH",
				Value: "testPath",
			},
			{
				Name:  "CHARTOVERRIDEPATH",
				Value: "testOverridePath",
			},
			{
				Name:  "CHARTOVERRIDES",
				Value: "replicas:2",
			},
			{
				Name:  "NAMESPACE",
				Value: "default",
			},
		},
	}

	tests := []struct {
		name         string
		deployType   string
		shouldError  bool
		tempDirPath  string
		tempFileName string
		tempPath     string
		cleanUp      func()
	}{
		{
			name:         "helm",
			deployType:   "helm",
			shouldError:  false,
			tempDirPath:  "charts",
			tempFileName: "charts/production.yaml",
			tempPath:     "../../test/templates/helm/charts/production.yaml",
			cleanUp: func() {
				os.Remove(".charts")
				os.Remove(".github")
			},
		},
		{
			name:         "kustomize",
			deployType:   "kustomize",
			shouldError:  false,
			tempDirPath:  "overlays/production",
			tempFileName: "overlays/production/deployment.yaml",
			tempPath:     "../../test/templates/kustomize/overlays/production/deployment.yaml",
			cleanUp: func() {
				os.Remove(".overlays")
				os.Remove(".github")
			},
		},
		{
			name:         "manifests",
			deployType:   "manifests",
			shouldError:  false,
			tempDirPath:  "manifests",
			tempFileName: "manifests/deployment.yaml",
			tempPath:     "../../test/templates/manifests/manifests/deployment.yaml",
			cleanUp: func() {
				os.Remove(".manifests")
				os.Remove(".github")
			},
		},
		{
			name:         "missing manifest",
			deployType:   "manifests",
			shouldError:  true,
			tempDirPath:  "manifests",
			tempFileName: "manifests/deployment.yaml",
			tempPath:     "../../test/templates/manifests/manifests/deployment.yaml",
			cleanUp: func() {
				os.Remove(".manifests")
				os.Remove(".github")
			},
		},
		{
			name:         "invalid deploy type",
			deployType:   "invalid",
			shouldError:  true,
			tempDirPath:  "manifests",
			tempFileName: "manifests/deployment.yaml",
			tempPath:     "../../test/templates/manifests/manifests/deployment.yaml",
			cleanUp: func() {
			},
		},
	}

	for _, tt := range tests {

		err := createTempDeploymentFile("charts", "charts/production.yaml", "../../test/templates/helm/charts/production.yaml")
		assert.Nil(t, err)

		workflows := CreateWorkflowsFromEmbedFS(template.Workflows, dest)
		err = workflows.CreateWorkflowFiles(tt.deployType, draftConfig, templatewriter)
		if err != nil && tt.shouldError == false {
			t.Errorf("Default Build Context CreateWorkflows() error = %v, wantErr %v", err, tt.shouldError)
		}
		err = workflows.CreateWorkflowFiles(tt.deployType, draftConfigNoRoot, templatewriter)
		if err != nil && tt.shouldError == false {
			t.Errorf("Custom Build Context CreateWorkflows() error = %v, wantErr %v", err, tt.shouldError)
		}

		tt.cleanUp()
	}
}

func TestUpdateProductionDeploymentsValid(t *testing.T) {
	testTemplateWriter := &writers.LocalFSWriter{}

	//test for valid helm deployment file
	helmFileName, _ := createTempManifest("../../test/templates/helm_prod_values.yaml")
	defer os.Remove(helmFileName)

	assert.Nil(t, setHelmContainerImage(helmFileName, "testImage", testTemplateWriter))

	helmDeploy := &HelmProductionYaml{}
	assert.Nil(t, helmDeploy.LoadFromFile(helmFileName))
	assert.Equal(t, "testImage", helmDeploy.Image.Repository)

	//test for valid deployment file
	deploymentFileName, _ := createTempManifest("../../test/templates/deployment.yaml")
	defer os.Remove(deploymentFileName)

	assert.Nil(t, setDeploymentContainerImage(deploymentFileName, "testImage"))
	decode := scheme.Codecs.UniversalDeserializer().Decode
	file, err := ioutil.ReadFile(deploymentFileName)
	assert.Nil(t, err)

	k8sObj, _, err := decode(file, nil, nil)
	assert.Nil(t, err)

	deploy, ok := k8sObj.(*appsv1.Deployment)
	assert.True(t, ok)
	assert.Equal(t, "testImage", deploy.Spec.Template.Spec.Containers[0].Image)
}

func TestUpdateProductionDeploymentsInvalid(t *testing.T) {
	testTemplateWriter := &writers.LocalFSWriter{}

	//test for invalid helm deployment file
	tempFile, err := ioutil.TempFile("", "*.yaml")
	assert.Nil(t, err)
	defer os.Remove(tempFile.Name())
	yamlData := []byte(`not a valid yaml`)
	_, err = tempFile.Write(yamlData)
	assert.Nil(t, err)
	err = tempFile.Close()
	assert.Nil(t, err)
	assert.NotNil(t, setHelmContainerImage(tempFile.Name(), "testImage", testTemplateWriter))

	//test for invalid deployment file
	assert.NotNil(t, setDeploymentContainerImage(tempFile.Name(), "testImage"))

	//test for invalid k8sObj
	invalidDeploymentFile, _ := createTempManifest("../../test/templates/invalid_deployment.yaml")
	assert.Equal(t, errors.New("could not decode kubernetes deployment"), setDeploymentContainerImage(invalidDeploymentFile, "testImage"))

	//test for unsupported number of containers in the deployment spec
	invalidDeploymentFile, _ = createTempManifest("../../test/templates/unsupported_no_of_containers.yaml")
	defer os.Remove(invalidDeploymentFile)
	assert.Equal(t, errors.New("unsupported number of containers defined in the deployment spec"), setDeploymentContainerImage(invalidDeploymentFile, "testImage"))
}

func TestUpdateProductionDeploymentsMissing(t *testing.T) {
	draftConfig := &config.DraftConfig{
		Variables: []*config.BuilderVar{
			{
				Name:  "AZURECONTAINERREGISTRY",
				Value: "testRegistry",
			},
			{
				Name:  "CONTAINERNAME",
				Value: "testContainer",
			},
		},
	}
	testTemplateWriter := &writers.LocalFSWriter{}
	//test for missing deploy type
	assert.Nil(t, UpdateProductionDeployments("", ".", draftConfig, testTemplateWriter))

	//test for missing helm deployment file
	assert.NotNil(t, setHelmContainerImage("", "testImage", testTemplateWriter))

	//test for missing deployment file
	assert.NotNil(t, setDeploymentContainerImage("", "testImage"))
}

func TestLoadConfig(t *testing.T) {
	fakeFS, err := createMockWorkflowTemplatesFS()
	assert.Nil(t, err)
	w, err := createMockWorkflow("workflows", fakeFS)
	assert.Nil(t, err)

	cases := []loadConfTestCase{
		{"helm", true},
		{"kustomize", true},
		{"manifests", true},
		{"fake", false},
		{"emptyDir", false},
		{"corrupted", false},
	}

	for _, c := range cases {
		if c.isNil {
			_, err = w.loadConfig(c.deployType)
			assert.Nil(t, err)
		} else {
			_, err = w.loadConfig(c.deployType)
			assert.NotNil(t, err)
		}
	}
}

func TestPopulateConfigs(t *testing.T) {
	fakeFS, err := createMockWorkflowTemplatesFS()
	assert.Nil(t, err)

	w, err := createMockWorkflow("workflows", fakeFS)
	assert.Nil(t, err)

	w.populateConfigs()
	assert.Equal(t, 5, len(w.configs)) // includes emptyDir and corrupted so 2 additional configs

	w, err = createTestWorkflowEmbed("workflows")
	assert.Nil(t, err)

	w.populateConfigs()
	assert.Equal(t, 3, len(w.configs))
}

func TestCreateWorkflowFiles(t *testing.T) {
	templatewriter := &writers.LocalFSWriter{}
	draftConfig := &config.DraftConfig{
		Variables: []*config.BuilderVar{
			{
				Name:  "WORKFLOWNAME",
				Value: "testWorkflow",
			},
			{
				Name:  "BRANCHNAME",
				Value: "testBranch",
			},
			{
				Name:  "ACRRESOURCEGROUP",
				Value: "testAcrRG",
			},
			{
				Name:  "AZURECONTAINERREGISTRY",
				Value: "testAcr",
			},
			{
				Name:  "CONTAINERNAME",
				Value: "testContainer",
			},
			{
				Name:  "CLUSTERRESOURCEGROUP",
				Value: "testClusterRG",
			},
			{
				Name:  "CLUSTERNAME",
				Value: "testCluster",
			},
			{
				Name:  "DOCKERFILE",
				Value: "./Dockerfile",
			},
			{
				Name:  "BUILDCONTEXTPATH",
				Value: ".",
			},
			{
				Name:  "CHARTPATH",
				Value: "testPath",
			},
			{
				Name:  "CHARTOVERRIDEPATH",
				Value: "testOverridePath",
			},
			{
				Name:  "CHARTOVERRIDES",
				Value: "replicas:2",
			},
			{
				Name:  "NAMESPACE",
				Value: "default",
			},
		},
	}
	draftConfigNoRoot := &config.DraftConfig{
		Variables: []*config.BuilderVar{
			{
				Name:  "WORKFLOWNAME",
				Value: "testWorkflow",
			},
			{
				Name:  "BRANCHNAME",
				Value: "testBranch",
			},
			{
				Name:  "ACRRESOURCEGROUP",
				Value: "testAcrRG",
			},
			{
				Name:  "AZURECONTAINERREGISTRY",
				Value: "testAcr",
			},
			{
				Name:  "CONTAINERNAME",
				Value: "testContainer",
			},
			{
				Name:  "CLUSTERRESOURCEGROUP",
				Value: "testClusterRG",
			},
			{
				Name:  "CLUSTERNAME",
				Value: "testCluster",
			},
			{
				Name:  "DOCKERFILE",
				Value: "./Dockerfile",
			},
			{
				Name:  "BUILDCONTEXTPATH",
				Value: "test",
			},
			{
				Name:  "CHARTPATH",
				Value: "testPath",
			},
			{
				Name:  "CHARTOVERRIDEPATH",
				Value: "testOverridePath",
			},
			{
				Name:  "CHARTOVERRIDES",
				Value: "replicas:2",
			},
			{
				Name:  "NAMESPACE",
				Value: "default",
			},
		},
	}
	badDraftConfig := &config.DraftConfig{}

	workflowTemplate, err := createMockWorkflowTemplatesFS()
	assert.Nil(t, err)

	mockWF, err := createMockWorkflow("workflows", workflowTemplate)
	assert.Nil(t, err)

	mockWF.populateConfigs()

	err = mockWF.CreateWorkflowFiles("fakeDeployType", draftConfig, templatewriter)
	assert.NotNil(t, err)

	err = mockWF.CreateWorkflowFiles("helm", draftConfig, templatewriter)
	assert.Nil(t, err)
	os.RemoveAll(".github")

	err = mockWF.CreateWorkflowFiles("helm", draftConfigNoRoot, templatewriter)
	assert.Nil(t, err)
	os.RemoveAll(".github")

	err = mockWF.CreateWorkflowFiles("helm", badDraftConfig, templatewriter)
	assert.NotNil(t, err)
	os.RemoveAll(".github")
}

type loadConfTestCase struct {
	deployType string
	isNil      bool
}

func createTempManifest(path string) (string, error) {
	file, err := ioutil.TempFile("", "*.yaml")
	if err != nil {
		return "", err
	}
	defer file.Close()

	var source *os.File
	source, err = os.Open(path)
	if err != nil {
		return "", err
	}
	defer source.Close()

	_, err = io.Copy(file, source)
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}

func createTempDeploymentFile(dirPath, fileName, path string) error {
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return err
	}
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	var source *os.File
	source, err = os.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()

	_, err = io.Copy(file, source)
	if err != nil {
		return err
	}
	return nil
}

// Creates a copy of the embeded files in memory and returns a passable fs to use for testing
func createMockWorkflowTemplatesFS() (fs.FS, error) {
	rootPath := "workflows/"
	embedFiles, err := embedutils.EmbedFStoMapWithFiles(template.Workflows, "workflows")
	if err != nil {
		return nil, fmt.Errorf("failed to readDir: %w in embeded files", err)
	}

	mockFS := fstest.MapFS{}

	for path, file := range embedFiles {
		if file.IsDir() {
			mockFS[path] = &fstest.MapFile{Mode: fs.ModeDir}
		} else {
			bytes, err := template.Workflows.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read file: %w", err)
			}
			mockFS[path] = &fstest.MapFile{Data: bytes}
		}
	}

	mockFS[rootPath+"emptyDir"] = &fstest.MapFile{Mode: fs.ModeDir}
	mockFS[rootPath+"corrupted"] = &fstest.MapFile{Mode: fs.ModeDir}
	mockFS[rootPath+"corrupted/draft.yaml"] = &fstest.MapFile{Data: []byte("fake yaml data")}

	return mockFS, nil
}

func createMockWorkflow(dirPath string, mockWorkflowTemplates fs.FS) (*Workflows, error) {
	dest := "."

	deployMap, err := fsToMap(mockWorkflowTemplates, dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed fsToMap: %w", err)
	}

	w := &Workflows{
		workflows:         deployMap,
		Dest:              dest,
		configs:           make(map[string]*config.DraftConfig),
		workflowTemplates: mockWorkflowTemplates,
	}

	return w, nil
}

func createTestWorkflowEmbed(dirPath string) (*Workflows, error) {
	dest := "."

	deployMap, err := embedutils.EmbedFStoMap(template.Workflows, "workflows")
	if err != nil {
		return nil, fmt.Errorf("failed to create deployMap: %w", err)
	}

	w := &Workflows{
		workflows:         deployMap,
		Dest:              dest,
		configs:           make(map[string]*config.DraftConfig),
		workflowTemplates: template.Workflows,
	}

	return w, nil
}

func fsToMap(fsFS fs.FS, path string) (map[string]fs.DirEntry, error) {
	files, err := fs.ReadDir(fsFS, path)
	if err != nil {
		return nil, fmt.Errorf("failed to ReadDir: %w", err)
	}

	mapping := make(map[string]fs.DirEntry)

	for _, f := range files {
		if f.IsDir() {
			mapping[f.Name()] = f
		}
	}

	return mapping, nil
}
