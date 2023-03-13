package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	type args struct {
		name string
	}
	port := 6464
	bind := "0.0.0.0"
	nomadAddr := "http://localhost:4646"
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "default config",
			args: args{name: "testdata/simple.hcl"},
			want: &Config{
				Port: port,
				Bind: bind,
				Nomad: &NomadServer{
					Address: nomadAddr,
				},
				Validators: []Validator{},
				Mutators:   []Mutator{},
			},
		},
		{
			name:    "fail if no config file",
			args:    args{name: "testdata/doesnotexist.hcl"},
			want:    nil,
			wantErr: true,
		},

		{
			name: "with admission controllers",
			args: args{name: "testdata/with_admission.hcl"},
			want: &Config{
				Port: port,
				Bind: bind,
				Nomad: &NomadServer{
					Address: nomadAddr,
				},
				Validators: []Validator{
					{
						Type: "opa",
						Name: "some_opa_validator",
						OpaRule: &OpaRule{

							Query:    "errors = data.costcenter_meta.errors",
							Filename: "testdata/opa/validators/costcenter_meta.rego",
						},
					},
				},
				Mutators: []Mutator{
					{
						Type: "opa_json_patch",
						Name: "some_opa_mutator",
						OpaRule: &OpaRule{

							Query:    "patch = data.hello_world_meta.patch",
							Filename: "testdata/opa/mutators/hello_world_meta.rego",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadConfig(tt.args.name)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, config)
			}

		})
	}
}
