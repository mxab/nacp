package validator

import (
	"testing"

	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/require"
)

func TestOpaValidator(t *testing.T) {
	// create a context with a timeout

	// create a new OPA object
	opa, err := NewOpaValidator([]OpaRule{
		{
			name:  "nacp.vault_policy_prefix",
			query: "policies_allowed = data.nacp.vault_policy_prefix.allow",
			module: `
			package nacp.vault_policy_prefix

		import future.keywords.every
		import future.keywords.if
		import future.keywords.contains

		default allow :=false
		policies contains name if {
			name := input.TaskGroups[_].Tasks[_].Vault.Policies[_]
		}
		policy_prefix := concat("-", [input.ID, ""])
		allow if {

			every p in policies {
				startswith(p, policy_prefix)
			}
		}
			`,
			binding: "policies_allowed",
		},
	})

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
			_, _, err := opa.Validate(job)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpaValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}
