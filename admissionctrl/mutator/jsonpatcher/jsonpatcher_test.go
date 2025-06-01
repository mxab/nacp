package jsonpatcher

import (
	"reflect"
	"testing"

	"github.com/hashicorp/nomad/api"
)

func TestPatchJob(t *testing.T) {
	type args struct {
		job       *api.Job
		patchdata []interface{}
	}
	tests := []struct {
		name        string
		args        args
		want        *api.Job
		wantMutated bool
		wantErr     bool
	}{
		{
			name: "no patches",
			args: args{
				job:       &api.Job{},
				patchdata: []interface{}{},
			},
			want:        &api.Job{},
			wantMutated: false,
		},
		{
			name: "simple patch",
			args: args{
				job: &api.Job{},
				patchdata: []interface{}{

					map[string]interface{}{
						"op":    "add",
						"path":  "/Meta",
						"value": map[string]string{"description": "This is a test job"},
					},
				},
			},
			want: &api.Job{
				Meta: map[string]string{
					"description": "This is a test job",
				},
			},
			wantMutated: true,
			wantErr:     false,
		},
		{
			name: "invalid patch",
			args: args{
				job: &api.Job{
					Meta: map[string]string{
						"description": "This is a test job",
					},
				},
				patchdata: []interface{}{
					map[string]interface{}{
						"op":    "invalid",
						"path":  "/Meta",
						"value": map[string]string{"description": "This is a test job"},
					},
				},
			},
			want:        nil,
			wantMutated: false,
			wantErr:     true,
		},
		{
			name: "invalid patch data",
			args: args{

				job: &api.Job{},

				patchdata: []interface{}{
					func() {}, // Invalid patch data
				},
			},
			want:        nil,
			wantMutated: false,
			wantErr:     true,
		},
		{
			name: "unmarshable payload",
			args: args{
				job: &api.Job{

					TaskGroups: []*api.TaskGroup{
						{
							Tasks: []*api.Task{
								{
									Name: "test-task",
									Config: map[string]interface{}{
										"oh no": func() {

										},
									},
								},
							},
						},
					},
				},
				patchdata: []interface{}{
					map[string]interface{}{
						"op":    "add",
						"path":  "/Meta",
						"value": "This is a test job",
					},
				},
			},
			want:        nil,
			wantMutated: false,
			wantErr:     true,
		},
		{
			name: "patch fails if fields dont match api",
			args: args{
				job: &api.Job{
					Meta: map[string]string{
						"description": "This is a test job",
					},
				},
				patchdata: []interface{}{
					map[string]interface{}{
						"op":    "replace",
						"path":  "/Meta",
						"value": 123, // Invalid type for description
					},
				},
			},
			want:        nil,
			wantMutated: false,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, mutated, err := PatchJob(tt.args.job, tt.args.patchdata)
			if (err != nil) != tt.wantErr {
				t.Errorf("PatchJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PatchJob() = %v, want %v", got, tt.want)
			}
			if mutated != tt.wantMutated {
				t.Errorf("PatchJob() mutated = %v, want %v", mutated, tt.wantMutated)
			}
		})
	}
}
