package opa

import (
	"context"
	"os"

	"github.com/hashicorp/nomad/api"
	"github.com/open-policy-agent/opa/rego"
)

type OpaQuery struct {
	query *rego.PreparedEvalQuery
}
type OpaQueryResult struct {
	resultSet *rego.ResultSet
}

func CreateQuery(filename string, query string, ctx context.Context) (*OpaQuery, error) {

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

	return &OpaQuery{
		query: &preparedQuery,
	}, nil
}

func (q *OpaQuery) Query(ctx context.Context, input *api.Job) (*OpaQueryResult, error) {
	resultSet, err := q.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return nil, err
	}
	return &OpaQueryResult{&resultSet}, nil
}

func (result *OpaQueryResult) GetWarnings() []interface{} {

	rs := *result.resultSet

	warnings, ok := rs[0].Bindings["warnings"].([]interface{})
	if !ok {
		return make([]interface{}, 0)
	}
	return warnings
}
func (result *OpaQueryResult) GetErrors() []interface{} {

	rs := *result.resultSet
	errors, ok := rs[0].Bindings["errors"].([]interface{})
	if !ok {
		return make([]interface{}, 0)
	}
	return errors
}
func (result *OpaQueryResult) GetPatch() []interface{} {

	rs := *result.resultSet
	patch, ok := rs[0].Bindings["patch"].([]interface{})
	if !ok {
		return make([]interface{}, 0)
	}
	return patch
}
