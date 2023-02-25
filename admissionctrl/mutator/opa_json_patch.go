package mutator

import (
	"context"
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/admissionctrl/opa"
)

type OpaJsonPatchMutator struct {
	ruleSets []*opa.OpaRuleSet
	logger   hclog.Logger
}

func (j *OpaJsonPatchMutator) Mutate(job *api.Job) (out *api.Job, warnings []error, err error) {

	ctx := context.TODO()
	for _, ruleSet := range j.ruleSets {
		result, err := ruleSet.Eval(ctx, job)
		if err != nil {
			return nil, nil, err
		}
		patchData, ok := result[0].Bindings["patch"].([]interface{})
		patchJSON, err := json.Marshal(patchData)
		if err != nil {
			return nil, nil, err
		}

		if ok {
			patch, err := jsonpatch.DecodePatch(patchJSON)
			if err != nil {
				return nil, nil, err
			}
			j.logger.Debug("Got patch fom ruleset %s, patch: %v", ruleSet.Name(), patchJSON)
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

	return job, nil, nil
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
