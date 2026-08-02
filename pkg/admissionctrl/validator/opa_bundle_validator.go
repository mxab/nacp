package validator

import (
	"context"
	"errors"

	"github.com/hashicorp/go-multierror"
	"github.com/mxab/nacp/pkg/admissionctrl"
	"github.com/mxab/nacp/pkg/admissionctrl/opa/bundle"
	"github.com/mxab/nacp/pkg/admissionctrl/types"
)

type OpaBundleValidator struct {
	name   string
	path   string
	bundle *bundle.Instance
}

var _ admissionctrl.JobValidator = (*OpaBundleValidator)(nil) // Verify that *T implements I.

func NewOpaBundleValidator(name string, path string, instance *bundle.Instance) (*OpaBundleValidator, error) {
	if instance == nil {
		return nil, errors.New("OPA bundle is required")
	}
	if path == "" {
		return nil, errors.New("OPA decision path is required")
	}
	return &OpaBundleValidator{
		name:   name,
		path:   path,
		bundle: instance,
	}, nil
}

func (v *OpaBundleValidator) Validate(ctx context.Context, payload *types.Payload) ([]error, error) {
	// Decide parses the result strictly: a decision that is not the documented
	// {errors, warnings} document fails the admission instead of being read as
	// "the policy found nothing".
	decision, err := v.bundle.Decide(ctx, v.path, payload)
	if err != nil {
		return nil, err
	}

	if len(decision.Errors) > 0 {
		return decision.Warnings, multierror.Append(nil, decision.Errors...)
	}
	return decision.Warnings, nil
}

func (v *OpaBundleValidator) Name() string {
	return v.name
}
