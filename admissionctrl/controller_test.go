package admissionctrl

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/mxab/nacp/admissionctrl/types"

	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
)

func TestJobHandler_ApplyAdmissionControllers(t *testing.T) {
	type fields struct {
		mutator   func() *testutil.MockMutator
		validator func() *testutil.MockValidator
	}
	type args struct {
		job *api.Job
	}
	job := &api.Job{} // testutil.ReadJob(t)
	payload := &types.Payload{Job: job}

	defaultMutator := func() *testutil.MockMutator {
		mutator := new(testutil.MockMutator)
		mutator.On("Mutate", payload).Return(payload.Job, true, []error{}, nil)
		return mutator
	}
	defaultValidator := func() *testutil.MockValidator {

		validator := new(testutil.MockValidator)
		validator.On("Validate", payload).Return([]error{}, nil)
		return validator
	}
	tests := []struct {
		name                  string
		fields                fields
		args                  args
		want                  *api.Job
		wantWarnings          []error
		wantErr               bool
		resolveToken          bool
		wantedCalledMutator   bool
		wantedCalledValidator bool
	}{
		{
			name: "test",
			fields: fields{
				mutator:   defaultMutator,
				validator: defaultValidator,
			},
			args: args{
				job: job,
			},
			want:         job,
			wantWarnings: nil,
			wantErr:      false,

			wantedCalledMutator:   true,
			wantedCalledValidator: true,
		},
		{
			name: "test validator error",
			fields: fields{
				mutator: defaultMutator,
				validator: func() *testutil.MockValidator {
					return testutil.MockValidatorReturningError("Validator error")
				},
			},
			args: args{
				job: job,
			},
			want:         job,
			wantWarnings: nil,
			wantErr:      true,

			wantedCalledMutator:   true, // happened before validation
			wantedCalledValidator: true,
		},
		{
			name: "test mutator error",
			fields: fields{
				mutator: func() *testutil.MockMutator {
					return testutil.MockMutatorReturningError("Mutator error")
				},
				validator: defaultValidator,
			},
			args: args{
				job: job,
			},
			want:         job,
			wantWarnings: nil,
			wantErr:      true,

			wantedCalledMutator: true,
		},
		{
			name: "test validator warnings",
			fields: fields{
				mutator: defaultMutator,
				validator: func() *testutil.MockValidator {
					return testutil.MockValidatorReturningWarnings("Validator warning")
				},
			},
			args: args{
				job: job,
			},
			want:         job,
			wantWarnings: []error{fmt.Errorf("Validator warning")},
			wantErr:      false,

			wantedCalledMutator:   true, // happened before validation
			wantedCalledValidator: true,
		},
		{
			name: "test mutator warnings",
			fields: fields{
				mutator: func() *testutil.MockMutator {
					return testutil.MockMutatorReturningWarnings("Mutator warning")
				},

				validator: defaultValidator,
			},
			args: args{
				job: job,
			},
			want:                  job,
			wantWarnings:          []error{fmt.Errorf("Mutator warning")},
			wantErr:               false,
			wantedCalledMutator:   true,
			wantedCalledValidator: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator := tt.fields.mutator()
			validator := tt.fields.validator()
			j := NewJobHandler([]JobMutator{mutator}, []JobValidator{validator}, slog.New(slog.DiscardHandler), tt.resolveToken)
			payload := &types.Payload{Job: tt.args.job}
			_, warnings, err := j.ApplyAdmissionControllers(payload)

			assert.Equal(t, tt.wantWarnings, warnings, "Warnings should be equal")

			if (err != nil) != tt.wantErr {
				t.Errorf("JobHandler.ApplyAdmissionControllers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, tt.args.job, "Jobs should be equal")
				assert.Equal(t, tt.wantWarnings, warnings, "Warnings should be equal")

			}
			if tt.wantedCalledMutator {
				mutator.AssertExpectations(t)
			}
			if tt.wantedCalledValidator {
				validator.AssertExpectations(t)
			}
		})
	}
}
