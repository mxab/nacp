package opa

import (
	"context"
	"errors"
	"os"

	types2 "github.com/mxab/nacp/pkg/admissionctrl/types"

	"github.com/mxab/nacp/pkg/admissionctrl/notation"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/types"
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
	options := []func(*rego.Rego){
		rego.Query(query),
		rego.Module(filename, string(module)),
	}
	if verifier != nil {
		options = append(options, rego.Function1(

			&rego.Function{
				Name: "notation_verify_image",
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
		)
	}

	preparedQuery, err := rego.New(options...).PrepareForEval(ctx)

	if err != nil {
		return nil, err
	}

	return &OpaQuery{
		query: &preparedQuery,
	}, nil
}

func (q *OpaQuery) Query(ctx context.Context, payload *types2.Payload) (*OpaQueryResult, error) {
	resultSet, err := q.query.Eval(ctx, rego.EvalInput(payload))
	if err != nil {
		return nil, err
	}
	if len(resultSet) == 0 {
		return nil, errors.New("no result set returned, maybe the query is wrong?")
	}
	return &OpaQueryResult{&resultSet}, nil
}

// Decision reads the query bindings through the same decision contract the
// bundle adapters use. Lenient mode keeps the tolerance the embedded adapters
// shipped with: a missing or wrongly typed binding reports nothing.
func (result *OpaQueryResult) Decision() *Decision {
	rs := *result.resultSet

	decision, err := ParseDecision(map[string]interface{}(rs[0].Bindings), Lenient)
	if err != nil {
		// Lenient parsing never fails, but do not let a future change here turn
		// into a silent allow.
		return &Decision{Errors: []error{err}, Patch: []interface{}{}}
	}
	return decision
}
