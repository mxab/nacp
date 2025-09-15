package validator

import (
	"log/slog"
	"testing"

	"github.com/mxab/nacp/config"
	"github.com/mxab/nacp/pkg/admissionctrl/types"

	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpaValidator(t *testing.T) {
	// create a context with a timeout

	// create a new OPA object
	opaValidator, err := NewOpaValidator("testopavalidator", testutil.Filepath(t, "opa/validators/prefixed_policies/prefixed_policies.rego"),
		"errors = data.prefixed_policies.errors", slog.New(slog.DiscardHandler), nil)

	require.NoError(t, err)

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
			payload := &types.Payload{Job: job}
			_, err := opaValidator.Validate(t.Context(), payload)
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

			opaValidator, err := NewOpaValidator("testopavalidator", testutil.Filepath(t, "opa/errors.rego"),
				tt.query, slog.New(slog.DiscardHandler), nil)
			require.NoError(t, err)
			payload := &types.Payload{Job: dummyJob}
			warnings, err := opaValidator.Validate(t.Context(), payload)
			require.Equal(t, tt.wantErr, err != nil, "OpaValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			assert.Len(t, warnings, tt.wantWarnings, "OpaValidator.Validate() warnings = %v, wantWarnings %v", warnings, tt.wantWarnings)
		})
	}

}

func TestOpaValidatorContext(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		payload      *types.Payload
		wantErr      bool
		wantWarnings int
	}{
		{
			name:  "reject job from specific IP",
			query: `errors = data.context.errors`,
			payload: &types.Payload{
				Job: &api.Job{},
				Context: &config.RequestContext{
					ClientIP: "192.168.1.10",
				},
			},
			wantErr:      true,
			wantWarnings: 0,
		},
		{
			name:  "allow job from other IP",
			query: `errors = data.context.errors`,
			payload: &types.Payload{
				Job: &api.Job{},
				Context: &config.RequestContext{
					ClientIP: "192.168.1.20",
				},
			},
			wantErr:      false,
			wantWarnings: 0,
		},
		{
			name:  "reject job with forbidden policy",
			query: `errors = data.context.errors`,
			payload: &types.Payload{
				Job: &api.Job{},
				Context: &config.RequestContext{
					TokenInfo: &api.ACLToken{
						Policies: []string{"nomad_reject", "other_policy"},
					},
				},
			},
			wantErr:      true,
			wantWarnings: 0,
		},
		{
			name:  "warn job with warn policy",
			query: `warnings = data.context.warnings`,
			payload: &types.Payload{
				Job: &api.Job{},
				Context: &config.RequestContext{
					TokenInfo: &api.ACLToken{
						Policies: []string{"nomad_warn", "other_policy"},
					},
				},
			},
			wantErr:      false,
			wantWarnings: 1,
		},
		{
			name:  "allow job with normal policies",
			query: `errors = data.context.errors`,
			payload: &types.Payload{
				Job: &api.Job{},
				Context: &config.RequestContext{
					TokenInfo: &api.ACLToken{
						Policies: []string{"normal_policy"},
					},
				},
			},
			wantErr:      false,
			wantWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewOpaValidator(
				"test_context_validator",
				testutil.Filepath(t, "opa/validators/context/context.rego"),
				tt.query,
				slog.New(slog.DiscardHandler),
				nil,
			)
			require.NoError(t, err)

			warnings, err := validator.Validate(t.Context(), tt.payload)
			assert.Equal(t, tt.wantErr, err != nil, "OpaValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			assert.Len(t, warnings, tt.wantWarnings, "OpaValidator.Validate() warnings = %v, wantWarnings %v", warnings, tt.wantWarnings)
		})
	}
}
