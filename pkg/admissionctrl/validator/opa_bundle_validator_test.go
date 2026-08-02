package validator

import (
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
			expectErrParts: []string{"policy yielded an invalid error entry value"},
		},
		{
			name: "handle invalid warnings value",
			policy: `package mypolicy
			warnings = 5`,
			path:           "/mypolicy",
			expectErrParts: []string{"policy yielded an invalid warnings value"},
		},
		{
			name: "handle invalid warning entry value",
			policy: `package mypolicy
			warnings = [5]`,
			path:           "/mypolicy",
			expectErrParts: []string{"policy yielded an invalid warning entry value"},
		},
		{
			// A decision path that resolves to something other than the
			// documented {errors, warnings, patch} document must fail the
			// admission, not be read as "the policy found nothing".
			name: "reject non-object decision",
			policy: `package mypolicy
			allow := true`,
			path:           "/mypolicy/allow",
			expectErrParts: []string{"policy yielded an invalid decision value"},
		},
		{
			name: "reject list decision",
			policy: `package mypolicy
			findings := ["nope"]`,
			path:           "/mypolicy/findings",
			expectErrParts: []string{"policy yielded an invalid decision value"},
		},
		{
			name: "test invalid policy path",
			policy: `package mypolicy
			errors = ["an error message"]`,
			path:           "/invalidpath",
			expectErrParts: []string{"is undefined in the active bundle"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			job := testutil.BaseJob()

			opa := testutil.SetupOpa(t, tc.policy)
			validator, err := NewOpaBundleValidator("testopabundlevalidator", tc.path, opa)

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
	validator, err := NewOpaBundleValidator("testopabundlevalidator", "/mypolicy", opa)

	require.NoError(t, err, "No error creating validator")
	assert.Equal(t, "testopabundlevalidator", validator.Name(), "Validator name")
}

func TestNewOpaBundleValidatorValidation(t *testing.T) {
	opa := testutil.SetupOpa(t, "package mypolicy")

	validator, err := NewOpaBundleValidator("test", "/mypolicy", nil)
	assert.ErrorContains(t, err, "OPA bundle is required")
	assert.Nil(t, validator)

	validator, err = NewOpaBundleValidator("test", "", opa)
	assert.ErrorContains(t, err, "OPA decision path is required")
	assert.Nil(t, validator)
}
