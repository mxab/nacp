package admissionctrl

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/mitchellh/copystructure"
	"github.com/mxab/nacp/admissionctrl/types"

	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
)

type AddMetaMutator struct {
	Field string
}

func (m *AddMetaMutator) Mutate(payload *types.Payload) (*api.Job, bool, []error, error) {
	copy, err := copystructure.Copy(payload.Job)

	if err != nil {
		return nil, false, nil, fmt.Errorf("failed to copy job: %w", err)
	}
	job := copy.(*api.Job)
	// Simulate a mutation by adding a field to the job's Meta
	if job.Meta == nil {
		job.Meta = make(map[string]string)
	}
	job.Meta[m.Field] = "applied"
	return job, true, nil, nil
}
func (m *AddMetaMutator) Name() string {
	return m.Field
}
func TestJobHandler_ApplyAdmissionControllers(t *testing.T) {
	type fields struct {
		mutators  func() []JobMutator
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
	defaultMutators := func() []JobMutator {
		return []JobMutator{defaultMutator()}
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
				mutators:  defaultMutators,
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
				mutators: defaultMutators,
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
				mutators: func() []JobMutator {
					return []JobMutator{
						testutil.MockMutatorReturningError("Mutator error"),
					}
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
				mutators: defaultMutators,
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
				mutators: func() []JobMutator {
					return []JobMutator{
						testutil.MockMutatorReturningWarnings("Mutator warning"),
					}
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
		{
			name: "two mutators with changes are applied",
			fields: fields{
				mutators: func() []JobMutator {

					return []JobMutator{&AddMetaMutator{Field: "mutator1"}, &AddMetaMutator{Field: "mutator2"}}
				},
				validator: defaultValidator,
			},
			args: args{
				job: job,
			},
			want: &api.Job{
				Meta: map[string]string{
					"mutator1": "applied",
					"mutator2": "applied",
				},
			},
			wantWarnings:          nil,
			wantErr:               false,
			wantedCalledMutator:   true,
			wantedCalledValidator: true,
		},
		{
			name: "mutator returns nil job results in error",
			fields: fields{
				mutators: func() []JobMutator {
					mutator := new(testutil.MockMutator)
					mutator.On("Mutate", payload).Return(nil, false, []error{}, nil)
					return []JobMutator{mutator}
				},
				validator: defaultValidator,
			},
			args: args{
				job: job,
			},
			want:                  nil,
			wantWarnings:          nil,
			wantErr:               true,
			wantedCalledMutator:   true,
			wantedCalledValidator: false, // validator should not be called if mutator returns nil job
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutators := tt.fields.mutators()

			validator := tt.fields.validator()
			j := NewJobHandler(mutators, []JobValidator{validator}, slog.New(slog.DiscardHandler), tt.resolveToken)
			payload := &types.Payload{Job: tt.args.job}
			job, warnings, err := j.ApplyAdmissionControllers(payload)

			assert.Equal(t, tt.wantWarnings, warnings, "Warnings should be equal")

			if (err != nil) != tt.wantErr {
				t.Errorf("JobHandler.ApplyAdmissionControllers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, job, "Jobs should be equal")
				assert.Equal(t, tt.wantWarnings, warnings, "Warnings should be equal")

			}
			if tt.wantedCalledMutator {
				for _, mutator := range mutators {
					if mutator, ok := mutator.(*testutil.MockMutator); ok {
						// Assert that the mutator was called
						mutator.AssertExpectations(t)
					}
				}
			}
			if tt.wantedCalledValidator {
				validator.AssertExpectations(t)
			}
		})
	}
}
