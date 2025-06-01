package jsonpatcher

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/hashicorp/nomad/api"
)

func PatchJob(job *api.Job, patchOps []interface{}) (*api.Job, bool, error) {

	jobJson, err := json.Marshal(job)
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal job: %w", err)
	}

	patchOpsData, err := json.Marshal(patchOps)
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal patch data: %w", err)
	}
	patch, err := jsonpatch.DecodePatch(patchOpsData)
	if err != nil {
		return nil, false, err
	}
	patchedJobJson, err := patch.Apply(jobJson)

	if err != nil {
		return nil, false, fmt.Errorf("failed to apply patch: %w", err)
	}
	var patchedJob api.Job
	err = json.Unmarshal(patchedJobJson, &patchedJob)
	if err != nil {
		return nil, false, err
	}
	mutated := len(patch) > 0
	return &patchedJob, mutated, nil
}
