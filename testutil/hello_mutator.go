package testutil

import (
	"context"

	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/admissionctrl/types"
)

type HelloMutator struct {
	MutatorName string
}

func (h *HelloMutator) Mutate(ctx context.Context, payload *types.Payload) (out *api.Job, mutated bool, warnings []error, err error) {

	if payload.Job.Meta == nil {
		payload.Job.Meta = make(map[string]string)
	}

	payload.Job.Meta["hello"] = "world"

	return payload.Job, true, nil, nil
}

func (h *HelloMutator) Name() string {
	return h.MutatorName
}
