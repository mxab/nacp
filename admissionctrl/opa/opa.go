package opa

import (
	"context"
	"errors"
	"os"

	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/admissionctrl/notation"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/types"
)

type OpaQuery struct {
	query *rego.PreparedEvalQuery
}
type OpaQueryResult struct {
	resultSet *rego.ResultSet
}

func CreateQuery(filename string, query string, ctx context.Context, verifier notation.ImageVerifier) (*OpaQuery, error) {

	module, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	preparedQuery, err := rego.New(
		rego.Query(query),
		rego.Function1(
			&rego.Function{
				Name: "valid_notation_image",
				Decl: types.NewFunction(types.Args(types.S), types.B),
			},
			func(bctx rego.BuiltinContext, a *ast.Term) (*ast.Term, error) {
				if str, ok := a.Value.(ast.String); ok {
					ctx := bctx.Context
					err := verifier.VerifyImage(ctx, string(str))
					valid := err == nil
					return ast.BooleanTerm(valid), nil

				}
				return ast.BooleanTerm(false), nil
			}),
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
	if len(resultSet) == 0 {
		return nil, errors.New("no result set returned, maybe the query is wrong?")
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
