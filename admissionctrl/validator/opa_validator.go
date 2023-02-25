package validator

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/admissionctrl/opa"
)

type OpaValidator struct {
	ruleSets []*opa.OpaRuleSet
}

func (v *OpaValidator) Validate(job *api.Job) (*api.Job, []error, error) {

	ctx := context.TODO()
	//iterate over rulesets and evaluate
	allErrs := &multierror.Error{}
	allWarnings := make([]error, 0)
	for _, ruleSet := range v.ruleSets {
		// evaluate the query

		results, err := ruleSet.Eval(ctx, job)

		if err != nil {
			return nil, nil, err
		}

		// aggregate warnings
		warnings, ok := results[0].Bindings["warnings"].([]interface{})

		if ok && len(warnings) > 0 {
			for _, warn := range warnings {
				allWarnings = append(allWarnings, fmt.Errorf("%s (%s)", warn, ruleSet.Name()))
			}
		}

		errors, ok := results[0].Bindings["errors"].([]interface{})
		if ok || len(errors) > 0 { // no errors is ok
			errsForRule := &multierror.Error{}
			for _, err := range errors {
				errsForRule = multierror.Append(errsForRule, fmt.Errorf("%s (%s)", err, ruleSet.Name()))
			}
			allErrs = multierror.Append(allErrs, errsForRule)
		}

	}
	if len(allErrs.Errors) > 0 {
		return job, allWarnings, allErrs
	}
	return job, allWarnings, nil
}

// Name
func (v *OpaValidator) Name() string {
	return "opa"
}

func NewOpaValidator(rules []opa.OpaQueryAndModule) (*OpaValidator, error) {

	ctx := context.TODO()
	// read the policy file
	ruleSets, err := opa.CreateOpaRuleSet(rules, ctx)
	if err != nil {
		return nil, err
	}
	return &OpaValidator{
		ruleSets: ruleSets,
	}, nil

}
