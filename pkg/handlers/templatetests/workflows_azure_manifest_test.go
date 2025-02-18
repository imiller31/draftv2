package templatetests

import (
	"testing"

	"github.com/Azure/draft/pkg/templatewriter/writers"
)

func TestAzureWorkflowManifestTemplates(t *testing.T) {
	tests := []TestInput{
		{
			Name:            "valid azpipeline manifests deployment",
			TemplateName:    "azure-pipeline-manifests",
			FixturesBaseDir: "../../fixtures/workflows/azurepipelines/manifests",
			Version:         "0.0.1",
			Dest:            ".",
			TemplateWriter:  &writers.FileMapWriter{},
			VarMap: map[string]string{
				"ARMSERVICECONNECTION":   "testserviceconnection",
				"AZURECONTAINERREGISTRY": "myacr.acr.io",
				"CONTAINERNAME":          "myapp",
				"CLUSTERRESOURCEGROUP":   "myrg",
				"ACRRESOURCEGROUP":       "myrg",
				"CLUSTERNAME":            "testcluster",
				"RESOURCETYPE":           "Microsoft.ContainerService/fleets",
			},
		},
	}

	for _, test := range tests {
		RunTemplateTest(t, test)
	}
}
