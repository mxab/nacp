package opa_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mxab/nacp/pkg/admissionctrl/types"

	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/pkg/admissionctrl/notation"
	. "github.com/mxab/nacp/pkg/admissionctrl/opa"
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

	`, ctx, nil)
	require.NoError(t, err, "No error creating query")
	assert.NotNil(t, query, "Query is not nil")

	job := &api.Job{}
	payload := &types.Payload{Job: job}
	result, err := query.Query(ctx, payload)
	assert.Nil(t, err, "No error executing query")
	assert.NotNil(t, result, "Result is not nil")

	decision := result.Decision()
	assert.Equal(t, []string{"This is a warning message"}, errorMessages(decision.Warnings), "Warnings are correct")
	assert.Equal(t, []string{"This is a error message"}, errorMessages(decision.Errors), "Errors are correct")
	assert.Equal(t, []interface{}{
		map[string]interface{}{
			"op":    "add",
			"path":  "/Meta",
			"value": map[string]interface{}{"foo": "bar"},
		},
	}, decision.Patch, "Patch is correct")
}

func errorMessages(errs []error) []string {
	out := make([]string, 0, len(errs))
	for _, err := range errs {
		out = append(out, err.Error())
	}
	return out
}
func TestFailOnEmptyResultSet(t *testing.T) {
	ctx := context.Background()

	path := testutil.Filepath(t, "opa/test.rego")
	query, err := CreateQuery(path, `
		errors = data.opatest.notexisting


	`, ctx, nil)
	require.Nil(t, err, "No error creating query")
	assert.NotNil(t, query, "Query is not nil")

	job := &api.Job{}
	payload := &types.Payload{Job: job}
	result, err := query.Query(ctx, payload)
	assert.Error(t, err, "Error executing query")
	assert.Nil(t, result, "Result is nil")

}
func TestReturnsEmptyIfNotExisting(t *testing.T) {
	ctx := context.Background()

	path := testutil.Filepath(t, "opa/test.rego")
	query, err := CreateQuery(path, `
		notimportant = data.opatest.errors


	`, ctx, nil)
	require.Nil(t, err, "No error creating query")
	assert.NotNil(t, query, "Query is not nil")
	job := &api.Job{}
	payload := &types.Payload{Job: job}
	result, err := query.Query(ctx, payload)
	assert.Nil(t, err, "No error executing query")
	assert.NotNil(t, result, "Result is not nil")

	decision := result.Decision()
	assert.Empty(t, decision.Warnings, "Warnings are correct")
	assert.Empty(t, decision.Errors, "Errors are correct")
	assert.Equal(t, []interface{}{}, decision.Patch, "Patch is correct")
	assert.False(t, decision.HasPatch, "No patch was produced")

}

type DummyVerifier struct {
}

func (m *DummyVerifier) VerifyImage(ctx context.Context, imageReference string) error {

	if imageReference == "invalidimage:latest" {
		return errors.New("invalid image")
	}
	if imageReference == "validimage:latest" {
		return nil
	}

	panic("invalid image reference")
}
func TestNotationImageValidation(t *testing.T) {

	tt := []struct {
		name           string
		image          string
		verifier       notation.ImageVerifier
		expectedErrors []string
	}{
		{
			name:           "valid image",
			image:          "validimage:latest",
			verifier:       new(DummyVerifier),
			expectedErrors: []string{},
		},
		{
			name:     "invalid image",
			image:    "invalidimage:latest",
			verifier: new(DummyVerifier),
			expectedErrors: []string{
				"Image is not in valid",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			ctx := context.Background()

			path := testutil.Filepath(t, "opa/test_notation.rego")

			query, err := CreateQuery(path,
				"errors = data.opatest.errors",
				ctx,
				tc.verifier,
			)
			job := &api.Job{
				TaskGroups: []*api.TaskGroup{
					{
						Tasks: []*api.Task{
							{
								Driver: "docker",
								Config: map[string]interface{}{
									"image": tc.image,
								},
							},
						},
					},
				},
			}
			require.NoError(t, err, "No error creating query")
			payload := &types.Payload{Job: job}
			result, err := query.Query(ctx, payload)
			require.NoError(t, err, "No error executing query")
			require.NotNil(t, result, "Result is not nil")

			assert.Equal(t, tc.expectedErrors, errorMessages(result.Decision().Errors), "Errors are correct")
		})
	}
}

func TestCreateQueryIfNotationFnIsUsedButVerifierIsNil(t *testing.T) {

	ctx := context.Background()

	path := testutil.Filepath(t, "opa/test_notation.rego")

	_, err := CreateQuery(path,
		"errors = data.opatest.errors",
		ctx,
		nil,
	)
	assert.Error(t, err, "Error creating query")

}
