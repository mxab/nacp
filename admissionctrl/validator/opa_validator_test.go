package validator

import (
	"context"
	"testing"

	"github.com/mxab/nacp/testutil"
	"github.com/open-policy-agent/opa/rego"
	"github.com/stretchr/testify/assert"
)

func TestOpaValidator(t *testing.T) {
	// create a context with a timeout

	ctx := context.TODO()

	// create a new OPA object
	// opa := NewOpaValidator()

	// opa.Validate(nil)
	// create a new query object
	query, err := rego.New(
		rego.Query("policies_allowed = data.nacp.vault_policy_prefix.allow"),
		//rego.Query("data.nacp.vault_policy_prefix.policy_prefix"),
		rego.Module("nacp.vault_policy_prefix", `
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
		`),
	).PrepareForEval(ctx)

	// prepare the query for evaluation

	if err != nil {
		panic(err)
	}
	results, err := query.Eval(ctx, rego.EvalInput(testutil.ReadJob(t)))

	if err != nil {
		panic(err)
	}
	assert.Equal(t, true, results[0].Bindings["policies_allowed"])

}
