package mutator

import (
	"testing"

	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/stretchr/testify/assert"
)

func TestJSONPatcher_Mutate(t *testing.T) {
	type args struct {
		job *structs.Job
	}
	tests := []struct {
		name         string
		j            *JSONPatcher
		args         args
		wantOut      *structs.Job
		wantWarnings []error
		wantErr      bool
	}{
		// {
		// 	name: "test",
		// 	j:    &JSONPatcher{},
		// 	args: args{
		// 		job: testutil.ReadJob(t, "job.json"),
		// 	},
		// 	wantOut:      testutil.ReadJob(t, "job.json"),
		// 	wantWarnings: []error{},
		// 	wantErr:      false,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &JSONPatcher{}
			gotOut, gotWarnings, err := j.Mutate(tt.args.job)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONPatcher.Mutate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Empty(t, gotWarnings)
			assert.Equal(t, tt.wantOut, gotOut)

		})
	}
}

func TestJSONPatcher_Name(t *testing.T) {
	tests := []struct {
		name string
		j    *JSONPatcher
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &JSONPatcher{}
			if got := j.Name(); got != tt.want {
				t.Errorf("JSONPatcher.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}
