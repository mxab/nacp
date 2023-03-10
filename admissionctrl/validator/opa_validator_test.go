package validator

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpaValidator(t *testing.T) {
	// create a context with a timeout

	// create a new OPA object
	opa, err := NewOpaValidator(testutil.Filepath(t, "opa/validators/prefixed_policies.rego"),
		"errors = data.prefixed_policies.errors", hclog.NewNullLogger())

	require.Equal(t, nil, err)

	tests := []struct {
		name    string
		jobFile string
		wantErr bool
	}{
		{
			name:    "valid job",
			jobFile: "job.json",
			wantErr: false,
		},
		{
			name:    "invalid job",
			jobFile: "job_invalid_policy.json",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := testutil.ReadJob(t, tt.jobFile)
			_, err := opa.Validate(job)
			require.Equal(t, tt.wantErr, err != nil, "OpaValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)

		})
	}

}
func TestOpaValidatorSimple(t *testing.T) {

	// table test of errors and warnings
	dummyJob := &api.Job{}

	tests := []struct {
		name         string
		query        string
		wantErr      bool
		wantWarnings int
	}{
		{
			name:         "error",
			query:        "errors = data.dummy.errors",
			wantErr:      true,
			wantWarnings: 0,
		},
		{
			name:         "warning",
			query:        "warnings = data.dummy.warnings",
			wantErr:      false,
			wantWarnings: 1,
		},
		{
			name: "warning_and_error",
			query: `
			errors = data.dummy.errors
			warnings = data.dummy.warnings
			`,
			wantErr:      true,
			wantWarnings: 1,
		},
		{
			name: "none",
			query: `
			ignore_errors = data.dummy.errors
			ignore_warnings = data.dummy.warnings
			`,
			wantErr:      false,
			wantWarnings: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			opa, err := NewOpaValidator(testutil.Filepath(t, "opa/errors.rego"),
				tt.query, hclog.NewNullLogger())
			require.NoError(t, err)
			warnings, err := opa.Validate(dummyJob)
			require.Equal(t, tt.wantErr, err != nil, "OpaValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			assert.Len(t, warnings, tt.wantWarnings, "OpaValidator.Validate() warnings = %v, wantWarnings %v", warnings, tt.wantWarnings)
		})
	}

}
