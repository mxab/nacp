package opa

import (
	"context"
	"os"

	"github.com/open-policy-agent/opa/rego"
)

func CreateQuery(filename string, query string, ctx context.Context) (*rego.PreparedEvalQuery, error) {

	module, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	preparedQuery, err := rego.New(
		rego.Query(query),
		rego.Module(filename, string(module)),
	).PrepareForEval(ctx)

	if err != nil {
		return nil, err
	}

	return &preparedQuery, nil
}

// func (r *OpaRuleSet) Eval(ctx context.Context, job *api.Job) (rego.ResultSet, error) {

// 	results, err := r.prepared.Eval(ctx, rego.EvalInput(job))
// 	return results, err
// }
