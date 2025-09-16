package mutator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/pkg/admissionctrl/mutator/jsonpatcher"
	"github.com/mxab/nacp/pkg/admissionctrl/notation"
	"github.com/mxab/nacp/pkg/admissionctrl/opa"
	"github.com/mxab/nacp/pkg/admissionctrl/types"
)

type OpaJsonPatchMutator struct {
	query  *opa.OpaQuery
	logger *slog.Logger
	name   string
}

func (j *OpaJsonPatchMutator) Mutate(ctx context.Context, payload *types.Payload) (*api.Job, bool, []error, error) {
	allWarnings := make([]error, 0)

	results, err := j.query.Query(ctx, payload)
	if err != nil {
		return nil, false, nil, err
	}

	errors := results.GetErrors()

	if len(errors) > 0 {
		j.logger.Debug("Got errors from rule", "rule", j.Name(), "errors", errors, "job", payload.Job.ID)
		allErrors := multierror.Append(nil)
		for _, warn := range errors {
			allErrors = multierror.Append(allErrors, fmt.Errorf("%s (%s)", warn, j.Name()))
		}
		return nil, false, nil, allErrors
	}

	warnings := results.GetWarnings()

	if len(warnings) > 0 {
		j.logger.Debug("Got warnings from rule", "rule", j.Name(), "warnings", warnings, "job", payload.Job.ID)
		for _, warn := range warnings {
			allWarnings = append(allWarnings, fmt.Errorf("%s (%s)", warn, j.Name()))
		}
	}
	patchData := results.GetPatch()
	patchedJob, mutated, err := jsonpatcher.PatchJob(payload.Job, patchData)
	if err != nil {
		return nil, false, nil, err
	}

	return patchedJob, mutated, allWarnings, nil
}
func (j *OpaJsonPatchMutator) Name() string {
	return j.name
}

func NewOpaJsonPatchMutator(name, filename, query string, logger *slog.Logger, ImageVerifier notation.ImageVerifier) (*OpaJsonPatchMutator, error) {

	ctx := context.TODO()
	// read the policy file
	preparedQuery, err := opa.CreateQuery(filename, query, ctx, ImageVerifier)
	if err != nil {
		return nil, err
	}
	return &OpaJsonPatchMutator{
		query:  preparedQuery,
		logger: logger,
		name:   name,
	}, nil

}
