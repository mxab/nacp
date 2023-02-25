package opa

import (
	"context"
	"os"

	"github.com/hashicorp/nomad/api"
	"github.com/open-policy-agent/opa/rego"
)

type OpaQueryAndModule struct {
	Filename string
	Query    string
}

type OpaRuleSet struct {
	rule     OpaQueryAndModule
	prepared *rego.PreparedEvalQuery
}

func CreateOpaRuleSet(rules []OpaQueryAndModule, ctx context.Context) ([]*OpaRuleSet, error) {
	var ruleSets []*OpaRuleSet

	for _, rule := range rules {

		module, err := os.ReadFile(rule.Filename)
		if err != nil {
			return nil, err
		}

		query, err := rego.New(
			rego.Query(rule.Query),
			rego.Module(rule.Filename, string(module)),
		).PrepareForEval(ctx)

		if err != nil {
			return nil, err
		}
		ruleSets = append(ruleSets, &OpaRuleSet{
			rule:     rule,
			prepared: &query,
		},
		)

	}
	return ruleSets, nil
}

func (r *OpaRuleSet) Eval(ctx context.Context, job *api.Job) (rego.ResultSet, error) {

	results, err := r.prepared.Eval(ctx, rego.EvalInput(job))
	return results, err
}
func (r *OpaRuleSet) Name() string {
	return r.rule.Filename
}
