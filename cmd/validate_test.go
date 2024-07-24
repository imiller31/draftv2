package cmd

import (
	"context"
	"path/filepath"

	"testing"

	"github.com/Azure/draft/pkg/safeguards"
	"github.com/Azure/draft/pkg/safeguards/preprocessing"
	"github.com/stretchr/testify/assert"

	h "github.com/Azure/draft/pkg/safeguards/types"
)

// TestRunValidate tests the run command for `draft validate` for proper returns
func TestRunValidate(t *testing.T) {
	ctx := context.TODO()
	manifestFilesEmpty := []h.ManifestFile{}
	manifestPathDirectorySuccess, _ := filepath.Abs("../pkg/safeguards/tests/all/success")
	manifestPathDirectoryError, _ := filepath.Abs("../pkg/safeguards/tests/all/error")
	manifestPathFileSuccess, _ := filepath.Abs("../pkg/safeguards/tests/all/success/all-success-manifest-1.yaml")
	manifestPathFileError, _ := filepath.Abs("../pkg/safeguards/tests/all/error/all-error-manifest-1.yaml")
	var manifestFiles []h.ManifestFile

	// Scenario 1: empty manifest path should error
	_, err := safeguards.GetManifestResults(ctx, manifestFilesEmpty)
	assert.NotNil(t, err)

	// Scenario 2a: manifest path leads to a directory of manifestFiles - expect success
	manifestFiles, err = preprocessing.GetManifestFiles(manifestPathDirectorySuccess)
	assert.Nil(t, err)
	v, err := safeguards.GetManifestResults(ctx, manifestFiles)
	assert.Nil(t, err)
	numViolations := countTestViolations(v)
	assert.Equal(t, numViolations, 0)

	// Scenario 2b: manifest path leads to a directory of manifestFiles - expect failure
	manifestFiles, err = preprocessing.GetManifestFiles(manifestPathDirectoryError)
	assert.Nil(t, err)
	v, err = safeguards.GetManifestResults(ctx, manifestFiles)
	assert.Nil(t, err)
	numViolations = countTestViolations(v)
	assert.Greater(t, numViolations, 0)

	// Scenario 3a: manifest path leads to one manifest file - expect success
	manifestFiles, err = preprocessing.GetManifestFiles(manifestPathFileSuccess)
	assert.Nil(t, err)
	v, err = safeguards.GetManifestResults(ctx, manifestFiles)
	assert.Nil(t, err)
	numViolations = countTestViolations(v)
	assert.Equal(t, numViolations, 0)

	// Scenario 3b: manifest path leads to one manifest file - expect failure
	manifestFiles, err = preprocessing.GetManifestFiles(manifestPathFileError)
	assert.Nil(t, err)
	v, err = safeguards.GetManifestResults(ctx, manifestFiles)
	assert.Nil(t, err)
	numViolations = countTestViolations(v)
	assert.Greater(t, numViolations, 0)
}

// TestRunValidate_Kustomize tests the run command for `draft validate` for proper returns when given a kustomize project
func TestRunValidate_Kustomize(t *testing.T) {
	ctx := context.TODO()
	kustomizationPath, _ := filepath.Abs("../pkg/safeguards/tests/kustomize/overlays/production")
	kustomizationFilePath, _ := filepath.Abs("../pkg/safeguards/tests/kustomize/overlays/production/kustomization.yaml")

	makeTempDir(t)
	t.Cleanup(func() { cleanupDir(t, tempDir) })

	var manifestFiles []h.ManifestFile
	var err error

	// Scenario 1a: kustomizationPath leads to a directory containing kustomization.yaml - expect success
	manifestFiles, err = preprocessing.RenderKustomizeManifest(kustomizationPath, tempDir)
	assert.Nil(t, err)
	v, err := safeguards.GetManifestResults(ctx, manifestFiles)
	assert.Nil(t, err)
	numViolations := countTestViolations(v)
	assert.Equal(t, numViolations, 1)

	// Scenario 1b: kustomizationFilePath path leads to a specific kustomization.yaml - expect success
	manifestFiles, err = preprocessing.RenderKustomizeManifest(kustomizationFilePath, tempDir)
	assert.Nil(t, err)
	v, err = safeguards.GetManifestResults(ctx, manifestFiles)
	assert.Nil(t, err)
	numViolations = countTestViolations(v)
	assert.Equal(t, numViolations, 1)
}
