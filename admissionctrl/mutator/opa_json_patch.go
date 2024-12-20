package mutator

import (
	"context"
	"encoding/json"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/admissionctrl/notation"
	"github.com/mxab/nacp/admissionctrl/opa"
	"github.com/mxab/nacp/admissionctrl/types"
)

type OpaJsonPatchMutator struct {
	query  *opa.OpaQuery
	logger hclog.Logger
	name   string
}

func (j *OpaJsonPatchMutator) Mutate(payload *types.Payload) (*api.Job, []error, error) {
	allWarnings := make([]error, 0)
	ctx := context.TODO()

	results, err := j.query.Query(ctx, payload)
	if err != nil {
		return nil, nil, err
	}

	errors := results.GetErrors()

	if len(errors) > 0 {
		j.logger.Debug("Got errors from rule", "rule", j.Name(), "errors", errors, "job", payload.Job.ID)
		allErrors := multierror.Append(nil)
		for _, warn := range errors {
			allErrors = multierror.Append(allErrors, fmt.Errorf("%s (%s)", warn, j.Name()))
		}
		return nil, nil, allErrors
	}

	warnings := results.GetWarnings()

	if len(warnings) > 0 {
		j.logger.Debug("Got warnings from rule", "rule", j.Name(), "warnings", warnings, "job", payload.Job.ID)
		for _, warn := range warnings {
			allWarnings = append(allWarnings, fmt.Errorf("%s (%s)", warn, j.Name()))
		}
	}
	patchData := results.GetPatch()
	patchJSON, err := json.Marshal(patchData)
	if err != nil {
		return nil, nil, err
	}

	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		return nil, nil, err
	}
	j.logger.Debug("Got patch fom rule", "rule", j.Name(), "patch", string(patchJSON), "job", payload.Job.ID)
	jobJson, err := json.Marshal(payload.Job)
	if err != nil {
		return nil, nil, err
	}

	patched, err := patch.Apply(jobJson)
	if err != nil {
		return nil, nil, err
	}
	var patchedJob api.Job
	err = json.Unmarshal(patched, &patchedJob)
	if err != nil {
		return nil, nil, err
	}
	payload.Job = &patchedJob

	return payload.Job, allWarnings, nil
}
func (j *OpaJsonPatchMutator) Name() string {
	return j.name
}

func NewOpaJsonPatchMutator(name, filename, query string, logger hclog.Logger, ImageVerifier notation.ImageVerifier) (*OpaJsonPatchMutator, error) {

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
