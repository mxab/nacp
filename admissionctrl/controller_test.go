package admissionctrl

import (
	"github.com/mxab/nacp/admissionctrl/types"
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
	}
	type args struct {
		job *api.Job
	}
	job := &api.Job{} // testutil.ReadJob(t)
	payload := &types.Payload{Job: job}
	mutator := new(testutil.MockMutator)
	mutator.On("Mutate", payload).Return(payload.Job, []error{}, nil)

	validator := new(testutil.MockValidator)
	validator.On("Validate", payload).Return([]error{}, nil)

	tests := []struct {
		name         string
		fields       fields
		args         args
		want         *api.Job
		want1        []error
		wantErr      bool
		resolveToken bool
	}{
		{
			name: "test",
			fields: fields{
				mutator:   mutator,
				validator: validator,
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
			j := NewJobHandler([]JobMutator{tt.fields.mutator}, []JobValidator{tt.fields.validator}, hclog.NewNullLogger(), tt.resolveToken)
			payload := &types.Payload{Job: tt.args.job}
			_, warnings, err := j.ApplyAdmissionControllers(payload)
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
