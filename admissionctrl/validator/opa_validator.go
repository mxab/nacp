package validator

import (
	"context"
	"fmt"

	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/open-policy-agent/opa/rego"
)

type OpaRule struct {
	name    string
	query   string
	module  string
	binding string
}

type opaRuleSet struct {
	rule     OpaRule
	prepared *rego.PreparedEvalQuery
}

func NewOpaValidator(rules []OpaRule) (*OpaValidator, error) {
	ruleSets := make(map[string]*opaRuleSet)

	ctx := context.TODO()

	for _, rule := range rules {

		query, err := rego.New(
			rego.Query(rule.query),
			rego.Module(rule.name, rule.module),
		).PrepareForEval(ctx)

		if err != nil {
			return nil, err
		}
		ruleSets[rule.name] = &opaRuleSet{
			rule:     rule,
			prepared: &query,
		}
	}
	return &OpaValidator{
		ruleSets: ruleSets,
	}, nil

}

type OpaValidator struct {
	ruleSets map[string]*opaRuleSet
}

func (v *OpaValidator) Validate(job *structs.Job) (*structs.Job, []error, error) {

	ctx := context.TODO()
	//iterate over rulesets and evaluate
	for _, ruleSet := range v.ruleSets {
		// evaluate the query
		results, err := ruleSet.prepared.Eval(ctx, rego.EvalInput(job))
		if err != nil {
			return nil, nil, err
		}
		result := results[0].Bindings[ruleSet.rule.binding]
		//check if result is true
		if result != true {
			//if true return job
			return nil, nil, fmt.Errorf("opa validation failed for rule %s (result: %s)", ruleSet.rule.name, result)
		}
	}

	return job, nil, nil
}

// Name
func (v *OpaValidator) Name() string {
	return "opa"
}
