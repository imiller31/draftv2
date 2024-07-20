package example

import (
	"fmt"
	"testing"

	"github.com/Azure/draft/pkg/config"
	"github.com/Azure/draft/pkg/templatewriter/writers"
)

func TestWriteDockerfile(t *testing.T) {
	templateWriter := writers.FileMapWriter{}
	outputPath := "test/path"

	testCases := []struct {
		name               string
		langConfig         *config.DraftConfig
		generationLanguage string
		expectError        bool
	}{

		{
			name: "Test Valid Go Dockerfile Generation",
			langConfig: &config.DraftConfig{
				Variables: []*config.BuilderVar{
					{
						Name: "PORT",
						Default: &config.BuilderVarDefault{
							Value: "80",
						},
						Description: "the port exposed in the application",
						Type:        "int",
						Value:       "8080",
					},
					{
						Name: "VERSION",
						Default: &config.BuilderVarDefault{
							Value: "1.18",
						},
						Description:   "the version of go used by the application",
						ExampleValues: []string{"1.16", "1.17", "1.18", "1.19"},
						Value:         "1.20",
					},
				},
			},
			generationLanguage: "go",
			expectError:        false,
		},
		{
			name: "Test Valid Go Dockerfile Generation with default",
			langConfig: &config.DraftConfig{
				Variables: []*config.BuilderVar{
					{
						Name: "PORT",
						Default: &config.BuilderVarDefault{
							Value: "80",
						},
						Description: "the port exposed in the application",
						Type:        "int",
						Value:       "8080",
					},
					{
						Name: "VERSION",
						Default: &config.BuilderVarDefault{
							Value: "1.18",
						},
						Description:   "the version of go used by the application",
						ExampleValues: []string{"1.16", "1.17", "1.18", "1.19"},
					},
				},
			},
			generationLanguage: "go",
			expectError:        false,
		},
		{
			name: "Test Invalid GenerationLanguage",
			langConfig: &config.DraftConfig{
				Variables: []*config.BuilderVar{
					{
						Name: "PORT",
						Default: &config.BuilderVarDefault{
							Value: "80",
						},
						Description: "the port exposed in the application",
						Type:        "int",
						Value:       "8080",
					},
					{
						Name: "VERSION",
						Default: &config.BuilderVarDefault{
							Value: "1.18",
						},
						Description:   "the version of go used by the application",
						ExampleValues: []string{"1.16", "1.17", "1.18", "1.19"},
						Value:         "1.20",
					},
				},
			},
			generationLanguage: "invalid",
			expectError:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := WriteDockerfile(&templateWriter, outputPath, tc.langConfig, tc.generationLanguage)
			errored := err != nil
			if err != nil {
				fmt.Printf("WriteDockerfile failed: %e\n", err)
			}
			if errored != tc.expectError {
				t.Errorf("WriteDockerfile failed: expected error %t, got %t", tc.expectError, errored)
				t.Fail()
			}
		})
	}
}

func TestWriteDockerfileExample(t *testing.T) {
	err := WriteDockerfileExample()
	if err != nil {
		t.Errorf("WriteDockerfileExample failed: %e", err)
	}
}
