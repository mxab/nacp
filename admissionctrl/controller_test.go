package admissionctrl

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMutator struct {
	mock.Mock
}

func (m *MockMutator) Mutate(job *structs.Job) (out *structs.Job, warnings []error, err error) {
	args := m.Called(job)
	return args.Get(0).(*structs.Job), args.Get(1).([]error), args.Error(2)
}
func (m *MockMutator) Name() string {
	return "mock-mutator"
}

type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) Validate(job *structs.Job) (warnings []error, err error) {
	args := m.Called(job)
	return args.Get(0).([]error), args.Error(1)
}
func (m *MockValidator) Name() string {
	return "mock-validator"
}
func TestJobHandler_ApplyAdmissionControllers(t *testing.T) {

	type fields struct {
		mutator   JobMutator
		validator JobValidator
		logger    hclog.Logger
	}
	type args struct {
		job *structs.Job
	}
	job := &structs.Job{} // testutil.ReadJob(t)
	mutator := new(MockMutator)
	mutator.On("Mutate", job).Return(job, []error{}, nil)

	validator := new(MockValidator)
	validator.On("Validate", job).Return([]error{}, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *structs.Job
		want1   []error
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				mutator:   mutator,
				validator: validator,
				logger:    hclog.NewNullLogger(),
			},
			args: args{
				job: job,
			},
			want:    job,
			want1:   nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &JobHandler{
				mutators:   []JobMutator{tt.fields.mutator},
				validators: []JobValidator{tt.fields.validator},
				logger:     tt.fields.logger,
			}
			_, warnings, err := j.ApplyAdmissionControllers(tt.args.job)
			assert.Empty(t, warnings, "No Warnings")

			if (err != nil) != tt.wantErr {
				t.Errorf("JobHandler.ApplyAdmissionControllers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, tt.args.job, "Jobs should be equal")
				assert.Equal(t, tt.want1, warnings, "Warnings should be equal")
			} else {
				assert.NotEqual(t, tt.want, tt.args.job, "Jobs should not be equal")
			}

			mutator.AssertExpectations(t)
			validator.AssertExpectations(t)

		})
	}
}
