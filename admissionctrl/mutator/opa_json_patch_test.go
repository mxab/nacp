package mutator

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONPatcher_Mutate(t *testing.T) {
	type args struct {
		job *api.Job
	}
	tests := []struct {
		name         string
		j            *OpaJsonPatchMutator
		args         args
		wantOut      *api.Job
		wantWarnings []error
		wantErr      bool
	}{

		{
			name: "hello world",
			j: newMutator(t, testutil.Filepath(t, "opa/mutators/opajsonpatchtesting.rego"),
				"patch = data.opajsonpatchtesting.patch",
			),

			args: args{
				job: &api.Job{},
			},
			wantOut: &api.Job{
				Meta: map[string]string{
					"hello": "world",
				},
			},
			wantWarnings: []error{},
			wantErr:      false,
		},
		{
			name: "warning",
			j: newMutator(t, testutil.Filepath(t, "opa/mutators/opajsonpatchtesting.rego"),
				`warnings = data.opajsonpatchtesting.warnings`,
			),

			args: args{
				job: &api.Job{},
			},
			wantOut:      &api.Job{},
			wantWarnings: []error{fmt.Errorf("This is a warning message (%s)", "testopavalidator")},
			wantErr:      false,
		},
		{
			name: "warning and hello world",
			j: newMutator(t, testutil.Filepath(t, "opa/mutators/opajsonpatchtesting.rego"),
				`
				patch = data.opajsonpatchtesting.patch
				warnings = data.opajsonpatchtesting.warnings
				`,
			),

			args: args{
				job: &api.Job{},
			},
			wantOut: &api.Job{
				Meta: map[string]string{
					"hello": "world",
				},
			},
			wantWarnings: []error{fmt.Errorf("This is a warning message (%s)", "testopavalidator")},
			wantErr:      false,
		},
		{
			name: "error",
			j: newMutator(t, testutil.Filepath(t, "opa/mutators/opajsonpatchtesting.rego"),
				`errors = data.opajsonpatchtesting.errors`,
			),

			args: args{
				job: &api.Job{},
			},
			wantOut:      nil,
			wantWarnings: nil,
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, gotWarnings, err := tt.j.Mutate(tt.args.job)
			require.Equal(t, tt.wantErr, err != nil, "JSONPatcher.Mutate() error = %v, wantErr %v", err, tt.wantErr)

			assert.Equal(t, tt.wantWarnings, gotWarnings, "JSONPatcher.Mutate() gotWarnings = %v, want %v", gotWarnings, tt.wantWarnings)
			assert.Equal(t, tt.wantOut, gotOut)

		})
	}
}

func newMutator(t *testing.T, filename, query string) *OpaJsonPatchMutator {
	t.Helper()
	m, err := NewOpaJsonPatchMutator("testopavalidator", filename, query, hclog.NewNullLogger())
	require.NoError(t, err)
	return m
}
