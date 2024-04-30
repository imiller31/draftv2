package cmd

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"testing"

	"github.com/Azure/draft/pkg/safeguards"
	"github.com/stretchr/testify/assert"
)

// TestIsDirectory tests the isDirectory function for proper returns
func TestIsDirectory(t *testing.T) {
	testWd, _ := os.Getwd()
	pathTrue := testWd
	pathFalse := path.Join(testWd, "validate.go")
	pathError := ""

	isDir, err := isDirectory(pathTrue)
	assert.True(t, isDir)
	assert.Nil(t, err)

	isDir, err = isDirectory(pathFalse)
	assert.False(t, isDir)
	assert.Nil(t, err)

	isDir, err = isDirectory(pathError)
	assert.False(t, isDir)
	assert.NotNil(t, err)
}

// TestIsYAML tests the isYAML function for proper returns
func TestIsYAML(t *testing.T) {
	dirNotYaml, _ := filepath.Abs("../pkg/safeguards/tests/not-yaml")
	dirYaml, _ := filepath.Abs("../pkg/safeguards/tests/all/success")
	fileNotYaml, _ := filepath.Abs("../pkg/safeguards/tests/not-yaml/readme.md")
	fileYaml, _ := filepath.Abs("../pkg/safeguards/tests/all/success/all-success-manifest-1.yaml")

	assert.False(t, isYAML(fileNotYaml))
	assert.True(t, isYAML(fileYaml))

	manifestFiles, err := GetManifestFiles(dirNotYaml)
	assert.Nil(t, manifestFiles)
	assert.NotNil(t, err)

	manifestFiles, err = GetManifestFiles(dirYaml)
	assert.NotNil(t, manifestFiles)
	assert.Nil(t, err)
}

// TestRunValidate tests the run command for `draft validate` for proper returns
func TestRunValidate(t *testing.T) {
	ctx := context.TODO()
	manifestFilesEmpty := []safeguards.ManifestFile{}
	manifestPathDirectorySuccess, _ := filepath.Abs("../pkg/safeguards/tests/all/success")
	manifestPathDirectoryError, _ := filepath.Abs("../pkg/safeguards/tests/all/error")
	manifestPathFileSuccess, _ := filepath.Abs("../pkg/safeguards/tests/all/success/all-success-manifest-1.yaml")
	manifestPathFileError, _ := filepath.Abs("../pkg/safeguards/tests/all/error/all-error-manifest-1.yaml")
	var manifestFiles []safeguards.ManifestFile

	// Scenario 1: empty manifest path should error
	_, err := safeguards.GetManifestResults(ctx, manifestFilesEmpty)
	assert.NotNil(t, err)

	// Scenario 2a: manifest path leads to a directory of manifestFiles - expect success
	manifestFiles, err = GetManifestFiles(manifestPathDirectorySuccess)
	assert.Nil(t, err)
	v, err := safeguards.GetManifestResults(ctx, manifestFiles)
	assert.Nil(t, err)
	numViolations := countTestViolations(v)
	assert.Equal(t, numViolations, 0)

	// Scenario 2b: manifest path leads to a directory of manifestFiles - expect failure
	manifestFiles, err = GetManifestFiles(manifestPathDirectoryError)
	assert.Nil(t, err)
	v, err = safeguards.GetManifestResults(ctx, manifestFiles)
	assert.Nil(t, err)
	numViolations = countTestViolations(v)
	assert.Greater(t, numViolations, 0)

	// Scenario 3a: manifest path leads to one manifest file - expect success
	manifestFiles, err = GetManifestFiles(manifestPathFileSuccess)
	v, err = safeguards.GetManifestResults(ctx, manifestFiles)
	assert.Nil(t, err)
	numViolations = countTestViolations(v)
	assert.Equal(t, numViolations, 0)

	// Scenario 3b: manifest path leads to one manifest file - expect failure
	manifestFiles, err = GetManifestFiles(manifestPathFileError)
	v, err = safeguards.GetManifestResults(ctx, manifestFiles)
	assert.Nil(t, err)
	numViolations = countTestViolations(v)
	assert.Greater(t, numViolations, 0)
}
