package validator

import (
	"log/slog"
	"testing"

	"github.com/mxab/nacp/pkg/admissionctrl/types"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpaBundleValidator(t *testing.T) {

	// https://www.openpolicyagent.org/docs/integration#integrating-with-the-go-sdk

	tt := []struct {
		name           string
		policy         string
		path           string
		expectErrParts []string
		expectWarns    []string
	}{
		{
			name:   "no issues",
			policy: `package mypolicy`,
			path:   "/mypolicy",
		},
		{
			name: "error",
			policy: `package mypolicy
			errors = ["an error message"]`,
			path:           "/mypolicy",
			expectErrParts: []string{"an error message"},
		},
		{
			name: "multiple errors",
			policy: `package mypolicy
			errors = ["an error message", "another error message"]`,
			path:           "/mypolicy",
			expectErrParts: []string{"an error message", "another error message"},
		},
		{
			name: "warning",
			policy: `package mypolicy
			warnings = ["a warning message"]`,
			path:        "/mypolicy",
			expectWarns: []string{"a warning message"},
		},
		{
			name: "handle invalid errors value",
			policy: `package mypolicy
			errors = 5`,
			path:           "/mypolicy",
			expectErrParts: []string{"policy yielded an invalid errors value"},
		},
		{
			name: "handle invalid error value",
			policy: `package mypolicy
			errors = [5]`,
			path:           "/mypolicy",
			expectErrParts: []string{"policy yielded an invalid error value"},
		},
		{
			name: "handle invalid warnings value",
			policy: `package mypolicy
			warnings = 5`,
			path:        "/mypolicy",
			expectWarns: []string{"policy yielded an invalid warnings value"},
		},
		{
			name: "handle invalid warnings value",
			policy: `package mypolicy
			warnings = [5]`,
			path:        "/mypolicy",
			expectWarns: []string{"policy yielded an invalid warning value"},
		},
		{
			name: "test invalid policy path",
			policy: `package mypolicy
			errors = ["an error message"]`,
			path:           "/invalidpath",
			expectErrParts: []string{"failed to perform policy decision"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			job := testutil.BaseJob()

			opa := testutil.SetupOpa(t, tc.policy)
			validator, err := NewOpaBundleValidator("testopabundlevalidator", tc.path, slog.New(slog.DiscardHandler), opa)

			require.NoError(t, err, "No error creating validator")

			warnings, err := validator.Validate(t.Context(), &types.Payload{Job: job})

			if len(tc.expectErrParts) > 0 {
				for _, expectErrPart := range tc.expectErrParts {
					assert.ErrorContains(t, err, expectErrPart, "Error from validator")
				}
			} else {
				assert.NoError(t, err, "No error from validator")
			}

			// check warnings
			require.Len(t, warnings, len(tc.expectWarns), "Number of warnings from validator")
			for i, expectWarn := range tc.expectWarns {
				assert.ErrorContains(t, warnings[i], expectWarn, "Warning from validator")
			}

		})
	}
}

func TestBundleValidatorName(t *testing.T) {
	opa := testutil.SetupOpa(t, "package mypolicy")
	validator, err := NewOpaBundleValidator("testopabundlevalidator", "/mypolicy", slog.New(slog.DiscardHandler), opa)

	require.NoError(t, err, "No error creating validator")

	assert.Equal(t, "testopabundlevalidator", validator.Name(), "Validator name")
}
