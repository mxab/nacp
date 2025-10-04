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

func NewOpaBundleValidator(name string, path string, logger *slog.Logger, sdk *sdk.OPA) (*OpaBundleValidator, error) {
	return &OpaBundleValidator{
		name:   name,
		path:   path,
		logger: logger,
		opa:    sdk,
	}, nil
}

func (v *OpaBundleValidator) Validate(ctx context.Context, payload *types.Payload) (warnings []error, err error) {

	result, err := v.opa.Decision(ctx, sdk.DecisionOptions{
		Input: payload,
		Path:  v.path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to perform policy decision: %w", err)
	}

	v.logger.DebugContext(ctx, "OPA decision", slog.Any("result", result))

	if rmap, ok := result.Result.(map[string]interface{}); ok {
		if errs, found := rmap["errors"]; found {
			if errlist, ok := errs.([]interface{}); ok {

				for _, e := range errlist {
					if emsg, ok := e.(string); ok {
						err = multierror.Append(err, errors.New(emsg))
					} else {
						err = multierror.Append(err, fmt.Errorf("policy yielded an invalid error value: %v", e))
					}
				}
				if err != nil {
					return
				}
			} else if errs != nil {
				err = fmt.Errorf("policy yielded an invalid errors value: %v", errs)
				return
			}
		}
		if warns, found := rmap["warnings"]; found {
			if warnlist, ok := warns.([]interface{}); ok {
				for _, w := range warnlist {
					if wmsg, ok := w.(string); ok {
						warnings = append(warnings, errors.New(wmsg))
					} else {
						warnings = append(warnings, fmt.Errorf("policy yielded an invalid warning value: %v", w))
					}
				}
			} else if warns != nil {
				warnings = append(warnings, fmt.Errorf("policy yielded an invalid warnings value: %v", warns))
			}
		}
	}

	return
}

func (v *OpaBundleValidator) Name() string {
	return v.name
}
