package opa

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/admissionctrl/notation"
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
func TestFailOnEmptyResultSet(t *testing.T) {
	ctx := context.Background()

	path := testutil.Filepath(t, "opa/test.rego")
	query, err := CreateQuery(path, `
		errors = data.opatest.notexisting


	`, ctx, nil)
	require.Nil(t, err, "No error creating query")
	assert.NotNil(t, query, "Query is not nil")

	job := &api.Job{}
	result, err := query.Query(ctx, job)
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
	result, err := query.Query(ctx, job)
	assert.Nil(t, err, "No error executing query")
	assert.NotNil(t, result, "Result is not nil")

	warnings := result.GetWarnings()
	assert.Equal(t, []interface{}{}, warnings, "Warnings are correct")
	errors := result.GetErrors()
	assert.Equal(t, []interface{}{}, errors, "Errors are correct")
	patch := result.GetPatch()
	assert.Equal(t, []interface{}{}, patch, "Patch is correct")

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
		expectedErrors []interface{}
	}{
		{
			name:           "valid image",
			image:          "validimage:latest",
			verifier:       new(DummyVerifier),
			expectedErrors: []interface{}{},
		},
		{
			name:     "invalid image",
			image:    "invalidimage:latest",
			verifier: new(DummyVerifier),
			expectedErrors: []interface{}{
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
			result, err := query.Query(ctx, job)
			require.NoError(t, err, "No error executing query")
			require.NotNil(t, result, "Result is not nil")

			errors := result.GetErrors()
			assert.Equal(t, tc.expectedErrors, errors, "Errors are correct")
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
