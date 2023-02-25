package validator

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/admissionctrl/opa"
)

type OpaValidator struct {
	ruleSets []*opa.OpaRuleSet
	logger   hclog.Logger
}

func (v *OpaValidator) Validate(job *api.Job) ([]error, error) {

	ctx := context.TODO()
	//iterate over rulesets and evaluate
	allErrs := &multierror.Error{}
	allWarnings := make([]error, 0)

	v.logger.Debug("Validating job", "job", job.ID)
	for _, ruleSet := range v.ruleSets {
		// evaluate the query

		results, err := ruleSet.Eval(ctx, job)

		if err != nil {
			return nil, err
		}

		// aggregate warnings
		warnings, ok := results[0].Bindings["warnings"].([]interface{})
		v.logger.Trace("Warnings", "warnings", warnings, "ok", ok)
		if ok && len(warnings) > 0 {

			for _, warn := range warnings {
				allWarnings = append(allWarnings, fmt.Errorf("%s (%s)", warn, ruleSet.Name()))
			}
		}

		errors, ok := results[0].Bindings["errors"].([]interface{})
		v.logger.Trace("Errors", "errors", errors, "ok", ok)
		if ok || len(errors) > 0 { // no errors is ok
			errsForRule := &multierror.Error{}
			for _, err := range errors {
				errsForRule = multierror.Append(errsForRule, fmt.Errorf("%s (%s)", err, ruleSet.Name()))
			}
			allErrs = multierror.Append(allErrs, errsForRule)
		}

	}
	if len(allErrs.Errors) > 0 {
		return allWarnings, allErrs
	}
	return allWarnings, nil
}

// Name
func (v *OpaValidator) Name() string {
	return "opa"
}

func NewOpaValidator(rules []opa.OpaQueryAndModule, logger hclog.Logger) (*OpaValidator, error) {

	ctx := context.TODO()
	// read the policy file
	ruleSets, err := opa.CreateOpaRuleSet(rules, ctx)
	if err != nil {
		return nil, err
	}
	return &OpaValidator{
		ruleSets: ruleSets,
		logger:   logger,
	}, nil

}
