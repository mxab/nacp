package mutator

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/mxab/nacp/pkg/admissionctrl/types"

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
		wantMutated  bool
		wantWarnings []error
		wantErr      bool
	}{

		{
			name: "hello world",
			j: newMutator(t, testutil.Filepath(t, "opa/mutators/opajsonpatchtesting/opajsonpatchtesting.rego"),
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
			wantMutated:  true,
			wantWarnings: []error{},
			wantErr:      false,
		},
		{
			name: "warning",
			j: newMutator(t, testutil.Filepath(t, "opa/mutators/opajsonpatchtesting/opajsonpatchtesting.rego"),
				`warnings = data.opajsonpatchtesting.warnings`,
			),

			args: args{
				job: &api.Job{},
			},
			wantOut:      &api.Job{},
			wantMutated:  false,
			wantWarnings: []error{fmt.Errorf("This is a warning message (%s)", "testopavalidator")},
			wantErr:      false,
		},
		{
			name: "warning and hello world",
			j: newMutator(t, testutil.Filepath(t, "opa/mutators/opajsonpatchtesting/opajsonpatchtesting.rego"),
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
			wantMutated:  true,
			wantWarnings: []error{fmt.Errorf("This is a warning message (%s)", "testopavalidator")},
			wantErr:      false,
		},
		{
			name: "error",
			j: newMutator(t, testutil.Filepath(t, "opa/mutators/opajsonpatchtesting/opajsonpatchtesting.rego"),
				`errors = data.opajsonpatchtesting.errors`,
			),

			args: args{
				job: &api.Job{},
			},
			wantMutated:  false,
			wantOut:      nil,
			wantWarnings: nil,
			wantErr:      true,
		},
		{
			name: "error when elements are not queryable",
			//faultypatch has only a patch, no errors or warnings
			j: newMutator(t, testutil.Filepath(t, "opa/mutators/faultypatch/faultypatch.rego"),
				`errors = data.faultypatch.errors`,
			),

			args: args{
				job: &api.Job{},
			},
			wantMutated:  false,
			wantOut:      nil,
			wantWarnings: nil,
			wantErr:      true,
		},
		{
			name: "error when result is not a patch",
			//faultypatch has only a patch does not contain valid patch data
			j: newMutator(t, testutil.Filepath(t, "opa/mutators/faultypatch/faultypatch.rego"),
				`patch = data.faultypatch.patch`,
			),

			args: args{
				job: &api.Job{},
			},
			wantMutated:  false,
			wantOut:      nil,
			wantWarnings: nil,
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := &types.Payload{Job: tt.args.job}
			gotOut, mutated, gotWarnings, err := tt.j.Mutate(t.Context(), payload)
			require.Equal(t, tt.wantErr, err != nil, "JSONPatcher.Mutate() error = %v, wantErr %v", err, tt.wantErr)

			assert.Equal(t, tt.wantWarnings, gotWarnings, "JSONPatcher.Mutate() gotWarnings = %v, want %v", gotWarnings, tt.wantWarnings)
			assert.Equal(t, tt.wantOut, gotOut)
			assert.Equal(t, tt.wantMutated, mutated, "JSONPatcher.Mutate() mutated = %v, want %v", mutated, tt.wantMutated)

		})
	}
}

func newMutator(t *testing.T, filename, query string) *OpaJsonPatchMutator {
	t.Helper()
	m, err := NewOpaJsonPatchMutator("testopavalidator", filename, query, slog.New(slog.DiscardHandler), nil)
	require.NoError(t, err)
	return m
}
