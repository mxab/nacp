package opa

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDecisionStrict(t *testing.T) {
	tt := []struct {
		name         string
		raw          interface{}
		expectErr    string
		expectErrs   []string
		expectWarns  []string
		expectPatch  []interface{}
		expectHasPch bool
	}{
		{
			name: "empty document",
			raw:  map[string]interface{}{},
		},
		{
			name:       "errors",
			raw:        map[string]interface{}{"errors": []interface{}{"boom", "bang"}},
			expectErrs: []string{"boom", "bang"},
		},
		{
			name:        "warnings",
			raw:         map[string]interface{}{"warnings": []interface{}{"careful"}},
			expectWarns: []string{"careful"},
		},
		{
			name:         "patch",
			raw:          map[string]interface{}{"patch": []interface{}{map[string]interface{}{"op": "remove", "path": "/Meta"}}},
			expectPatch:  []interface{}{map[string]interface{}{"op": "remove", "path": "/Meta"}},
			expectHasPch: true,
		},
		{
			name:         "empty patch list is still a patch",
			raw:          map[string]interface{}{"patch": []interface{}{}},
			expectPatch:  []interface{}{},
			expectHasPch: true,
		},
		{
			name: "null patch is not a patch",
			raw:  map[string]interface{}{"patch": nil},
		},
		{
			name:      "undefined decision",
			raw:       nil,
			expectErr: "policy decision is undefined",
		},
		{
			name:      "scalar decision does not mean allow",
			raw:       true,
			expectErr: "policy yielded an invalid decision value",
		},
		{
			name:      "list decision does not mean allow",
			raw:       []interface{}{"nope"},
			expectErr: "policy yielded an invalid decision value",
		},
		{
			name:      "errors not a list",
			raw:       map[string]interface{}{"errors": 5},
			expectErr: "policy yielded an invalid errors value",
		},
		{
			name:      "error entry not a string",
			raw:       map[string]interface{}{"errors": []interface{}{"fine", 5}},
			expectErr: "policy yielded an invalid error entry value",
		},
		{
			name:      "warnings not a list",
			raw:       map[string]interface{}{"warnings": 5},
			expectErr: "policy yielded an invalid warnings value",
		},
		{
			name:      "warning entry not a string",
			raw:       map[string]interface{}{"warnings": []interface{}{5}},
			expectErr: "policy yielded an invalid warning entry value",
		},
		{
			name:      "patch not a list",
			raw:       map[string]interface{}{"patch": 5},
			expectErr: "policy yielded an invalid patch value",
		},
		{
			name:      "patch entry not an object",
			raw:       map[string]interface{}{"patch": []interface{}{5}},
			expectErr: "policy yielded an invalid patch entry value",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := ParseDecision(tc.raw, Strict)

			if tc.expectErr != "" {
				assert.ErrorContains(t, err, tc.expectErr)
				assert.Nil(t, decision)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectErrs, messages(decision.Errors))
			assert.Equal(t, tc.expectWarns, messages(decision.Warnings))
			assert.Equal(t, tc.expectHasPch, decision.HasPatch)
			if tc.expectPatch != nil {
				assert.Equal(t, tc.expectPatch, decision.Patch)
			}
		})
	}
}

// TestParseDecisionLenient pins the tolerance the embedded adapters shipped
// with: anything that does not match the shape reports nothing rather than
// failing the request.
func TestParseDecisionLenient(t *testing.T) {
	tt := []struct {
		name        string
		raw         interface{}
		expectErrs  []string
		expectWarns []string
		expectPatch []interface{}
	}{
		{name: "undefined decision", raw: nil, expectPatch: []interface{}{}},
		{name: "scalar decision", raw: true, expectPatch: []interface{}{}},
		{name: "missing bindings", raw: map[string]interface{}{}, expectPatch: []interface{}{}},
		{
			name:        "errors not a list",
			raw:         map[string]interface{}{"errors": 5},
			expectPatch: []interface{}{},
		},
		{
			name:        "warnings not a list",
			raw:         map[string]interface{}{"warnings": 5},
			expectPatch: []interface{}{},
		},
		{
			name:        "patch not a list",
			raw:         map[string]interface{}{"patch": 5},
			expectPatch: []interface{}{},
		},
		{
			name:        "non string entries are rendered",
			raw:         map[string]interface{}{"errors": []interface{}{5}, "warnings": []interface{}{true}},
			expectErrs:  []string{"5"},
			expectWarns: []string{"true"},
			expectPatch: []interface{}{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := ParseDecision(tc.raw, Lenient)

			require.NoError(t, err)
			assert.Equal(t, tc.expectErrs, messages(decision.Errors))
			assert.Equal(t, tc.expectWarns, messages(decision.Warnings))
			assert.Equal(t, tc.expectPatch, decision.Patch)
			assert.False(t, decision.HasPatch)
		})
	}
}

func messages(errs []error) []string {
	if len(errs) == 0 {
		return nil
	}
	out := make([]string, 0, len(errs))
	for _, err := range errs {
		out = append(out, err.Error())
	}
	return out
}
