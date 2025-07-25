package validator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hashicorp/go-multierror"
	"github.com/mxab/nacp/admissionctrl/notation"
	"github.com/mxab/nacp/admissionctrl/opa"
	"github.com/mxab/nacp/admissionctrl/types"
)

type OpaValidator struct {
	query  *opa.OpaQuery
	logger *slog.Logger
	name   string
}

func (v *OpaValidator) Validate(ctx context.Context, payload *types.Payload) ([]error, error) {

	//iterate over rulesets and evaluate
	allErrs := &multierror.Error{}
	allWarnings := make([]error, 0)

	v.logger.Debug("Validating job", "job", payload.Job.ID)

	// evaluate the query
	results, err := v.query.Query(ctx, payload)

	if err != nil {
		return nil, err
	}

	// aggregate warnings
	warnings := results.GetWarnings()

	if len(warnings) > 0 {
		v.logger.Debug("Got warnings from rule", "rule", v.Name(), "warnings", warnings, "job", payload.Job.ID)
		for _, warn := range warnings {
			allWarnings = append(allWarnings, fmt.Errorf("%s (%s)", warn, v.Name()))
		}
	}

	errors := results.GetErrors()

	if len(errors) > 0 { // no errors is ok
		v.logger.Debug("Got errors from rule", "rule", v.Name(), "errors", errors, "job", payload.Job.ID)
		errsForRule := &multierror.Error{}
		for _, err := range errors {
			errsForRule = multierror.Append(errsForRule, fmt.Errorf("%s (%s)", err, v.Name()))
		}
		allErrs = multierror.Append(allErrs, errsForRule)
	}

	if len(allErrs.Errors) > 0 {
		return allWarnings, allErrs
	}
	return allWarnings, nil
}

// Name
func (v *OpaValidator) Name() string {
	return v.name
}

func NewOpaValidator(name, filename, query string, logger *slog.Logger, imageVerifier notation.ImageVerifier) (*OpaValidator, error) {

	ctx := context.TODO()

	// read the policy file
	preparedEvalQuery, err := opa.CreateQuery(filename, query, ctx, imageVerifier)
	if err != nil {
		return nil, err
	}
	return &OpaValidator{
		query:  preparedEvalQuery,
		logger: logger,
		name:   name,
	}, nil

}
