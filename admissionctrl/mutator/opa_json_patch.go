package mutator

import (
	"context"
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/admissionctrl/opa"
)

type OpaJsonPatchMutator struct {
	ruleSets []*opa.OpaRuleSet
	logger   hclog.Logger
}

func (j *OpaJsonPatchMutator) Mutate(job *api.Job) (out *api.Job, warnings []error, err error) {
	allWarnings := make([]error, 0)
	ctx := context.TODO()
	for _, ruleSet := range j.ruleSets {
		results, err := ruleSet.Eval(ctx, job)
		if err != nil {
			return nil, nil, err
		}

		errors, ok := results[0].Bindings["errors"].([]interface{})

		if ok && len(errors) > 0 {
			allErrors := multierror.Append(nil)
			for _, warn := range errors {
				allErrors = multierror.Append(allErrors, fmt.Errorf("%s (%s)", warn, ruleSet.Name()))
			}
			return nil, nil, allErrors
		}

		warnings, ok := results[0].Bindings["warnings"].([]interface{})

		if ok && len(warnings) > 0 {
			for _, warn := range warnings {
				allWarnings = append(allWarnings, fmt.Errorf("%s (%s)", warn, ruleSet.Name()))
			}
		}
		patchData, ok := results[0].Bindings["patch"].([]interface{})
		patchJSON, err := json.Marshal(patchData)
		if err != nil {
			return nil, nil, err
		}

		if ok {
			patch, err := jsonpatch.DecodePatch(patchJSON)
			if err != nil {
				return nil, nil, err
			}
			j.logger.Debug("Got patch fom rule", "rule", ruleSet.Name(), "patch", string(patchJSON))
			jobJson, err := json.Marshal(job)
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
			job = &patchedJob

		}

	}

	return job, allWarnings, nil
}
func (j *OpaJsonPatchMutator) Name() string {
	return "jsonpatch"
}

func NewOpaJsonPatchMutator(rules []opa.OpaQueryAndModule, logger hclog.Logger) (*OpaJsonPatchMutator, error) {

	ctx := context.TODO()
	// read the policy file
	ruleSets, err := opa.CreateOpaRuleSet(rules, ctx)
	if err != nil {
		return nil, err
	}
	return &OpaJsonPatchMutator{
		ruleSets: ruleSets,
		logger:   logger,
	}, nil

}
