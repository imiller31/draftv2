package safeguards

import (
	"context"
	"fmt"
	"os"

	api "github.com/open-policy-agent/gatekeeper/v3/apis"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// Globals
var s = runtime.NewScheme()
var wd, _ = os.Getwd()

// TODO: for each constraint/constraint template -> make a new embedded FS with directive
// could get away
var f = os.DirFS(wd)
var fc FileCrawler

// primes the scheme to be able to interpret beta templates
func init() {
	_ = clientgoscheme.AddToScheme(s)
	_ = api.AddToScheme(s)

	selectedVersion = getLatestSafeguardsVersion()
	updateSafeguardPaths()

	fc = FileCrawler{
		Safeguards: safeguards,
	}
}

// ValidateManifest is what will be called by `draft validate` to validate the user's manifest
// against each safeguards constraint
func ValidateManifest(ctx context.Context, manifestPathOrFile []string) error {
	// constraint client instantiation
	c, err := getConstraintClient()
	if err != nil {
		return err
	}

	// retrieval of templates, constraints, and deployment
	constraintTemplates, err := fc.ReadConstraintTemplates()
	if err != nil {
		return err
	}
	constraints, err := fc.ReadConstraints()
	if err != nil {
		return err
	}

	// loading of templates, constraints into constraint client
	err = loadConstraintTemplates(ctx, c, constraintTemplates)
	if err != nil {
		return err
	}
	err = loadConstraints(ctx, c, constraints)
	if err != nil {
		return err
	}

	// TODO: for loop
	for _, m := range manifestPathOrFile {
		manifest, err := fc.ReadManifest(m)
		if err != nil {
			return err
		}

		// validation of deployment manifest with constraints, templates loaded
		// TODO: for loop here
		var violations []string
		err = validateManifest(ctx, c, manifest)
		if err != nil {
			violations = append(violations, err.Error())
		}

		if len(violations) > 0 {
			return fmt.Errorf("violations have occurred: %s", violations)
		}
	}

	return nil
}
