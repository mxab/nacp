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
