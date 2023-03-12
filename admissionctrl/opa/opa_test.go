package opa

import (
	"context"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpa(t *testing.T) {
	ctx := context.Background()

	path := testutil.Filepath(t, "opa/test.rego")
	query, err := CreateQuery(path, `
		errors = data.opatest.errors
		warnings = data.opatest.warnings
		patch = data.opatest.patch

	`, ctx)
	require.Nil(t, err, "No error creating query")
	assert.NotNil(t, query, "Query is not nil")

	job := &api.Job{}
	result, err := query.Query(ctx, job)
	assert.Nil(t, err, "No error executing query")
	assert.NotNil(t, result, "Result is not nil")

	warnings := result.GetWarnings()
	assert.Equal(t, []interface{}{"This is a warning message"}, warnings, "Warnings are correct")

	errors := result.GetErrors()
	assert.Equal(t, []interface{}{"This is a error message"}, errors, "Errors are correct")

	patch := result.GetPatch()
	assert.Equal(t, []interface{}{
		map[string]interface{}{
			"op":    "add",
			"path":  "/Meta",
			"value": map[string]interface{}{"foo": "bar"},
		},
	}, patch, "Patch is correct")
}
