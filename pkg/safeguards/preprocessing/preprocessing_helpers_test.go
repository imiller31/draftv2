package preprocessing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

const (
	helmTempDir             = "testdata/helm" // Rendered files are stored here before they are read for comparison
	chartPath               = "../tests/testmanifests/validchart"
	invalidChartPath        = "../tests/testmanifests/invalidchart"
	invalidValuesChart      = "../tests/testmanifests/invalidvalues"
	invalidDeploymentsChart = "../tests/testmanifests/invaliddeployment"
	invalidDeploymentSyntax = "../tests/testmanifests/invaliddeployment-syntax"
	invalidDeploymentValues = "../tests/testmanifests/invaliddeployment-values"
	folderwithHelpersTmpl   = "../tests/testmanifests/different-structure"
	multipleTemplateDirs    = "../tests/testmanifests/multiple-templates"
	multipleValuesFile      = "../tests/testmanifests/multiple-values-files"

	subcharts                  = "../tests/testmanifests/multiple-charts"
	subchartDir                = "../tests/testmanifests/multiple-charts/charts/subchart2"
	directPath_ToSubchartYaml  = "../tests/testmanifests/multiple-charts/charts/subchart1/Chart.yaml"
	directPath_ToMainChartYaml = "../tests/testmanifests/multiple-charts/Chart.yaml"

	directPath_ToValidChart   = "../tests/testmanifests/validchart/Chart.yaml"
	directPath_ToInvalidChart = "../tests/testmanifests/invalidchart/Chart.yaml"
	kustomizeTempDir          = "testdata/kustomize"
	kustomizePath             = "../tests/testmanifests/kustomize"
)

func makeTempDir(t *testing.T, dir string) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create temporary output directory: %s", err)
	}
}

// Test rendering a valid Helm chart with no subcharts and three templates
func TestRenderHelmChart_Valid(t *testing.T) {
	makeTempDir(t, helmTempDir)
	t.Cleanup(func() { cleanupDir(t, helmTempDir) })

	manifestFiles, err := RenderHelmChart(false, chartPath, helmTempDir)
	assert.Nil(t, err)

	// Check that the output directory exists and contains expected files
	expectedFiles := make(map[string]string)
	expectedFiles["deployment.yaml"] = getManifestAsString(t, "../tests/testmanifests/expecteddeployment.yaml")
	expectedFiles["service.yaml"] = getManifestAsString(t, "../tests/testmanifests/expectedservice.yaml")
	expectedFiles["ingress.yaml"] = getManifestAsString(t, "../tests/testmanifests/expectedingress.yaml")

	for _, writtenFile := range manifestFiles {
		expectedYaml := expectedFiles[writtenFile.Name]
		writtenYaml := parseYAML(t, getManifestAsString(t, writtenFile.Path))
		assert.Equal(t, writtenYaml, parseYAML(t, expectedYaml))
	}

	cleanupDir(t, helmTempDir)
	makeTempDir(t, helmTempDir)

	// Test by giving file directly
	manifestFiles, err = RenderHelmChart(true, directPath_ToValidChart, helmTempDir)
	assert.Nil(t, err)

	for _, writtenFile := range manifestFiles {
		expectedYaml := expectedFiles[writtenFile.Name]
		writtenYaml := parseYAML(t, getManifestAsString(t, writtenFile.Path))
		assert.Equal(t, writtenYaml, parseYAML(t, expectedYaml))
	}
}

// Should successfully render a Helm chart with sub charts and be able to render subchart separately within a helm chart
func TestSubCharts(t *testing.T) {
	makeTempDir(t, helmTempDir)
	t.Cleanup(func() { cleanupDir(t, helmTempDir) })

	manifestFiles, err := RenderHelmChart(false, subcharts, helmTempDir)
	assert.Nil(t, err)

	// Assert that 3 files were created in temp dir: 1 from main chart, 2 from subcharts
	files, _ := os.ReadDir(helmTempDir)
	assert.Equal(t, len(files), 3)

	expectedFiles := make(map[string]string)
	expectedFiles["maindeployment.yaml"] = getManifestAsString(t, "../tests/testmanifests/expected-mainchart.yaml")
	expectedFiles["deployment1.yaml"] = getManifestAsString(t, "../tests/testmanifests/expected-subchart1.yaml")
	expectedFiles["deployment2.yaml"] = getManifestAsString(t, "../tests/testmanifests/expected-subchart2.yaml")

	for _, writtenFile := range manifestFiles {
		expectedYaml := expectedFiles[writtenFile.Name]
		writtenYaml := parseYAML(t, getManifestAsString(t, writtenFile.Path))
		assert.Equal(t, writtenYaml, parseYAML(t, expectedYaml))
	}

	cleanupDir(t, helmTempDir)
	makeTempDir(t, helmTempDir)

	// Given a sub-chart dir, that specific sub chart only should be evaluated and rendered
	_, err = RenderHelmChart(false, subchartDir, helmTempDir)
	assert.Nil(t, err)

	cleanupDir(t, helmTempDir)
	makeTempDir(t, helmTempDir)

	// Given a Chart.yaml in the main directory, main chart and subcharts should be evaluated
	_, err = RenderHelmChart(true, directPath_ToMainChartYaml, helmTempDir)
	assert.Nil(t, err)

	cleanupDir(t, helmTempDir)
	makeTempDir(t, helmTempDir)

	// Given path to a sub-Chart.yaml with a dependency on another subchart, should render both subcharts, but not the main chart
	manifestFiles, err = RenderHelmChart(true, directPath_ToSubchartYaml, helmTempDir)
	assert.Nil(t, err)

	expectedFiles = make(map[string]string)
	expectedFiles["deployment1.yaml"] = getManifestAsString(t, "../tests/testmanifests/expected-subchart1.yaml")
	expectedFiles["deployment2.yaml"] = getManifestAsString(t, "../tests/testmanifests/expected-subchart2.yaml")

	for _, writtenFile := range manifestFiles {
		expectedYaml := expectedFiles[writtenFile.Name]
		writtenYaml := parseYAML(t, getManifestAsString(t, writtenFile.Path))
		assert.Equal(t, writtenYaml, parseYAML(t, expectedYaml))
	}

	//expect mainchart.yaml to not exist
	outputFilePath := filepath.Join(helmTempDir, "maindeployment.yaml")
	assert.NoFileExists(t, outputFilePath, "Unexpected file was created: %s", outputFilePath)
}

/**
* Testing user errors
 */

// Should fail if the Chart.yaml is invalid
func TestInvalidChartAndValues(t *testing.T) {
	makeTempDir(t, helmTempDir)
	t.Cleanup(func() { cleanupDir(t, helmTempDir) })

	_, err := RenderHelmChart(false, invalidChartPath, helmTempDir)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to load main chart: validation: chart.metadata.name is required")

	_, err = RenderHelmChart(true, directPath_ToValidChart, helmTempDir)
	assert.Nil(t, err)

	// Should fail if values.yaml doesn't contain all values necessary for templating
	cleanupDir(t, helmTempDir)
	makeTempDir(t, helmTempDir)

	_, err = RenderHelmChart(false, invalidValuesChart, helmTempDir)
	assert.NotNil(t, err)
}

func TestInvalidDeployments(t *testing.T) {
	makeTempDir(t, helmTempDir)
	t.Cleanup(func() { cleanupDir(t, helmTempDir) })

	_, err := RenderHelmChart(false, invalidDeploymentSyntax, helmTempDir)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "parse error")
	assert.Contains(t, err.Error(), "function \"selector\" not defined")

	_, err = RenderHelmChart(false, invalidDeploymentValues, helmTempDir)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "map has no entry for key")
}

// Test different helm folder structures
func TestDifferentFolderStructures(t *testing.T) {
	makeTempDir(t, helmTempDir)
	t.Cleanup(func() { cleanupDir(t, helmTempDir) })

	manifestFiles, err := RenderHelmChart(false, folderwithHelpersTmpl, helmTempDir) // includes _helpers.tpl
	assert.Nil(t, err)

	expectedFiles := make(map[string]string)
	expectedFiles["deployment.yaml"] = getManifestAsString(t, "../tests/testmanifests/expected-helpers-deployment.yaml")
	expectedFiles["service.yaml"] = getManifestAsString(t, "../tests/testmanifests/expected-helpers-service.yaml")
	for _, writtenFile := range manifestFiles {
		expectedYaml := expectedFiles[writtenFile.Name]
		writtenYaml := parseYAML(t, getManifestAsString(t, writtenFile.Path))
		assert.Equal(t, writtenYaml, parseYAML(t, expectedYaml))
	}
	cleanupDir(t, helmTempDir)
	makeTempDir(t, helmTempDir)

	manifestFiles, err = RenderHelmChart(false, multipleTemplateDirs, helmTempDir) // all manifests defined in one file
	assert.Nil(t, err)

	expectedFiles = make(map[string]string)
	expectedFiles["resources.yaml"] = getManifestAsString(t, "../tests/testmanifests/expected-resources.yaml")
	expectedFiles["service-1.yaml"] = getManifestAsString(t, "../tests/testmanifests/expectedservice.yaml")
	expectedFiles["service-2.yaml"] = getManifestAsString(t, "../tests/testmanifests/expectedservice2.yaml")
	for _, writtenFile := range manifestFiles {
		expectedYaml := expectedFiles[writtenFile.Name]
		writtenYaml := parseYAML(t, getManifestAsString(t, writtenFile.Path))
		assert.Equal(t, writtenYaml, parseYAML(t, expectedYaml))
	}
}

func cleanupDir(t *testing.T, dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("Failed to clean directory: %s", err)
	}
}

func parseYAML(t *testing.T, content string) map[string]interface{} {
	var result map[string]interface{}
	err := yaml.Unmarshal([]byte(content), &result)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %s", err)
	}
	return result
}

func getManifestAsString(t *testing.T, filePath string) string {
	yamlFileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read YAML file: %s", err)
	}

	yamlContentString := string(yamlFileContent)
	return yamlContentString
}

// Test rendering a valid kustomize yaml
func TestRenderKustomizeManifest_Valid(t *testing.T) {
	makeTempDir(t, kustomizeTempDir)
	t.Cleanup(func() { cleanupDir(t, kustomizeTempDir) })

	_, err := RenderKustomizeManifest(kustomizePath, kustomizeTempDir)
	assert.Nil(t, err)

	//// Check that the output directory exists and contains expected files
	//expectedFiles := make(map[string]string)
	//expectedFiles["deployment.yaml"] = getManifestAsString(t, "../tests/testmanifests/expecteddeployment.yaml")
	//expectedFiles["service.yaml"] = getManifestAsString(t, "../tests/testmanifests/expectedservice.yaml")
	//expectedFiles["ingress.yaml"] = getManifestAsString(t, "../tests/testmanifests/expectedingress.yaml")
	//
	//for _, writtenFile := range manifestFiles {
	//	expectedYaml := expectedFiles[writtenFile.Name]
	//	writtenYaml := parseYAML(t, getManifestAsString(t, writtenFile.Path))
	//	assert.Equal(t, writtenYaml, parseYAML(t, expectedYaml))
	//}
	//
	//cleanupDir(t, helmTempDir)
	//makeTempDir(t)
	//
	//// Test by giving file directly
	//manifestFiles, err = RenderHelmChart(true, directPath_ToValidChart, helmTempDir)
	//assert.Nil(t, err)
	//
	//for _, writtenFile := range manifestFiles {
	//	expectedYaml := expectedFiles[writtenFile.Name]
	//	writtenYaml := parseYAML(t, getManifestAsString(t, writtenFile.Path))
	//	assert.Equal(t, writtenYaml, parseYAML(t, expectedYaml))
	//}
}
