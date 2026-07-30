package mutator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/pkg/admissionctrl"
	"github.com/mxab/nacp/pkg/admissionctrl/mutator/jsonpatcher"
	"github.com/mxab/nacp/pkg/admissionctrl/types"
	"github.com/open-policy-agent/opa/v1/sdk"
)

type OpaBundleMutator struct {
	name   string
	path   string
	logger *slog.Logger
	opa    *sdk.OPA
}

var _ admissionctrl.JobMutator = (*OpaBundleMutator)(nil)

func (m *OpaBundleMutator) Mutate(ctx context.Context, payload *types.Payload) (*api.Job, bool, []error, error) {
	decision, err := m.opa.Decision(ctx, sdk.DecisionOptions{
		Input: payload,
		Path:  m.path,
	})
	if err != nil {
		return nil, false, nil, fmt.Errorf("failed to perform policy decision: %w", err)
	}
	m.logger.DebugContext(ctx, "OPA decision", slog.Any("result", decision))

	result, ok := decision.Result.(map[string]interface{})
	if !ok {
		return nil, false, nil, fmt.Errorf("policy yielded an invalid decision value: %v", decision.Result)
	}

	if err := parseDecisionErrors(result["errors"]); err != nil {
		return nil, false, nil, err
	}

	warnings, err := parseDecisionWarnings(result["warnings"])
	if err != nil {
		return nil, false, warnings, err
	}

	patch, found := result["patch"]
	if !found || patch == nil {
		return payload.Job, false, warnings, nil
	}

	job, mutated, err := applyDecisionPatch(payload.Job, patch)
	return job, mutated, warnings, err
}

func parseDecisionErrors(raw interface{}) error {
	if raw == nil {
		return nil
	}

	entries, ok := raw.([]interface{})
	if !ok {
		return fmt.Errorf("policy yielded an invalid errors value: %v", raw)
	}

	var result error
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		message, ok := entry.(string)
		if !ok {
			return fmt.Errorf("policy yielded an invalid error entry value: %v", entry)
		}
		result = multierror.Append(result, errors.New(message))
	}
	return result
}

func parseDecisionWarnings(raw interface{}) ([]error, error) {
	warnings := []error{}
	if raw == nil {
		return warnings, nil
	}

	entries, ok := raw.([]interface{})
	if !ok {
		return warnings, fmt.Errorf("policy yielded an invalid warnings value: %v", raw)
	}

	for _, entry := range entries {
		if entry == nil {
			continue
		}
		message, ok := entry.(string)
		if !ok {
			return warnings, fmt.Errorf("policy yielded an invalid warning entry value: %v", entry)
		}
		warnings = append(warnings, errors.New(message))
	}
	return warnings, nil
}

func applyDecisionPatch(job *api.Job, raw interface{}) (*api.Job, bool, error) {
	operations, ok := raw.([]interface{})
	if !ok {
		return nil, false, fmt.Errorf("policy yielded an invalid patch value: %v", raw)
	}

	result, mutated, err := jsonpatcher.PatchJob(job, operations)
	if err != nil {
		return nil, false, fmt.Errorf("policy yielded patch failed: %w", err)
	}
	return result, mutated, nil
}

func NewOpaBundleMutator(name string, path string, logger *slog.Logger, opaSDK *sdk.OPA) (*OpaBundleMutator, error) {
	if opaSDK == nil {
		return nil, errors.New("OPA SDK is required")
	}
	if path == "" {
		return nil, errors.New("OPA decision path is required")
	}
	return &OpaBundleMutator{
		name:   name,
		path:   path,
		logger: logger,
		opa:    opaSDK,
	}, nil
}

func (m *OpaBundleMutator) Name() string {
	return m.name
}
