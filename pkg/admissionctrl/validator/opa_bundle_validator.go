package validator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hashicorp/go-multierror"
	"github.com/mxab/nacp/pkg/admissionctrl"
	"github.com/mxab/nacp/pkg/admissionctrl/types"
	"github.com/open-policy-agent/opa/v1/sdk"
)

type OpaBundleValidator struct {
	name   string
	path   string
	logger *slog.Logger
	opa    *sdk.OPA
}

var _ admissionctrl.JobValidator = (*OpaBundleValidator)(nil) // Verify that *T implements I.

func NewOpaBundleValidator(name string, path string, logger *slog.Logger, opaSDK *sdk.OPA) (*OpaBundleValidator, error) {
	if opaSDK == nil {
		return nil, errors.New("OPA SDK is required")
	}
	if path == "" {
		return nil, errors.New("OPA decision path is required")
	}
	return &OpaBundleValidator{
		name:   name,
		path:   path,
		logger: logger,
		opa:    opaSDK,
	}, nil
}

func (v *OpaBundleValidator) Validate(ctx context.Context, payload *types.Payload) ([]error, error) {
	decision, err := v.opa.Decision(ctx, sdk.DecisionOptions{
		Input: payload,
		Path:  v.path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to perform policy decision: %w", err)
	}

	v.logger.DebugContext(ctx, "OPA decision", slog.Any("result", decision))

	result, ok := decision.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("policy yielded an invalid decision value: %v", decision.Result)
	}

	warnings, err := parseBundleMessages(result["warnings"], "warning")
	if err != nil {
		return nil, err
	}

	policyErrors, err := parseBundleMessages(result["errors"], "error")
	if err != nil {
		return warnings, err
	}

	var validationErr error
	for _, policyErr := range policyErrors {
		validationErr = multierror.Append(validationErr, policyErr)
	}
	return warnings, validationErr
}

func parseBundleMessages(raw interface{}, kind string) ([]error, error) {
	if raw == nil {
		return nil, nil
	}

	entries, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("policy yielded an invalid %ss value: %v", kind, raw)
	}

	messages := make([]error, 0, len(entries))
	for _, entry := range entries {
		message, ok := entry.(string)
		if !ok {
			return nil, fmt.Errorf("policy yielded an invalid %s value: %v", kind, entry)
		}
		messages = append(messages, errors.New(message))
	}
	return messages, nil
}

func (v *OpaBundleValidator) Name() string {
	return v.name
}
