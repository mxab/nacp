package mutator

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/pkg/admissionctrl"
	"github.com/mxab/nacp/pkg/admissionctrl/mutator/jsonpatcher"
	"github.com/mxab/nacp/pkg/admissionctrl/opa/bundle"
	"github.com/mxab/nacp/pkg/admissionctrl/types"
)

type OpaBundleMutator struct {
	name   string
	path   string
	bundle *bundle.Instance
}

var _ admissionctrl.JobMutator = (*OpaBundleMutator)(nil)

func NewOpaBundleMutator(name string, path string, instance *bundle.Instance) (*OpaBundleMutator, error) {
	if instance == nil {
		return nil, errors.New("OPA bundle is required")
	}
	if path == "" {
		return nil, errors.New("OPA decision path is required")
	}
	return &OpaBundleMutator{
		name:   name,
		path:   path,
		bundle: instance,
	}, nil
}

func (m *OpaBundleMutator) Mutate(ctx context.Context, payload *types.Payload) (*api.Job, bool, []error, error) {
	decision, err := m.bundle.Decide(ctx, m.path, payload)
	if err != nil {
		return nil, false, nil, err
	}

	if len(decision.Errors) > 0 {
		return nil, false, nil, multierror.Append(nil, decision.Errors...)
	}

	// A policy that produced no patch is a no-op, not an empty patch.
	if !decision.HasPatch {
		return payload.Job, false, decision.Warnings, nil
	}

	job, mutated, err := jsonpatcher.PatchJob(payload.Job, decision.Patch)
	if err != nil {
		return nil, false, nil, fmt.Errorf("policy yielded patch failed: %w", err)
	}
	return job, mutated, decision.Warnings, nil
}

func (m *OpaBundleMutator) Name() string {
	return m.name
}
