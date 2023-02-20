package admissionctrl

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
)

func TestJobHandler_ApplyAdmissionControllers(t *testing.T) {

	type fields struct {
		mutator   JobMutator
		validator JobValidator
		logger    hclog.Logger
	}
	type args struct {
		job *api.Job
	}
	job := &api.Job{} // testutil.ReadJob(t)
	mutator := new(testutil.MockMutator)
	mutator.On("Mutate", job).Return(job, []error{}, nil)

	validator := new(testutil.MockValidator)
	validator.On("Validate", job).Return([]error{}, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.Job
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
