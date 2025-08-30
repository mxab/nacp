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
				Telemetry: &Telemetry{
					Logging: &Logging{
						Level: "info",
						SlogLogging: &SlogLogging{
							Text:    Ptr(true),
							TextOut: Ptr("stdout"),
							Json:    Ptr(false),
							JsonOut: Ptr("stdout"),
						},
						OtelLogging: &OtelLogging{
							Enabled: Ptr(false),
						},
					},
					Metrics: &Metrics{
						Enabled: false,
					},
					Tracing: &Tracing{
						Enabled: false,
					},
				},
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
					{
						Type: "notation",
						Name: "some_notation_validator",

						Notation: &NotationVerifierConfig{
							TrustPolicyFile: "testdata/notation/validators/trust_policy.json",
							TrustStoreDir:   "testdata/notation/validators/trust_store",
							RepoPlainHTTP:   false,
							MaxSigAttempts:  50,
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

				Telemetry: &Telemetry{
					Logging: &Logging{
						Level: "info",
						SlogLogging: &SlogLogging{
							Text:    Ptr(true),
							TextOut: Ptr("stdout"),
							Json:    Ptr(false),
							JsonOut: Ptr("stdout"),
						},
						OtelLogging: &OtelLogging{
							Enabled: Ptr(false),
						},
					},
					Metrics: &Metrics{
						Enabled: false,
					},
					Tracing: &Tracing{
						Enabled: false,
					},
				},
			},
		},
		{
			name: "with slog and json logging",
			args: args{name: "testdata/loggingjson.hcl"},
			want: &Config{
				Port: port,
				Bind: bind,

				Nomad: &NomadServer{
					Address: nomadAddr,
				},
				Validators: []Validator{},
				Mutators:   []Mutator{},
				Telemetry: &Telemetry{
					Logging: &Logging{
						Level: "info",
						SlogLogging: &SlogLogging{
							Json:    Ptr(true),
							Text:    Ptr(false),
							JsonOut: Ptr("stdout"),
							TextOut: Ptr("stdout"),
						},
						OtelLogging: &OtelLogging{
							Enabled: Ptr(false),
						},
					},
					Metrics: &Metrics{
						Enabled: false,
					},
					Tracing: &Tracing{
						Enabled: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "with otel logging",
			args: args{name: "testdata/otelconfig.hcl"},
			want: &Config{
				Port: port,
				Bind: bind,

				Nomad: &NomadServer{
					Address: nomadAddr,
				},
				Validators: []Validator{},
				Mutators:   []Mutator{},
				Telemetry: &Telemetry{
					Logging: &Logging{
						Level: "info",
						SlogLogging: &SlogLogging{ //just default part
							Text:    Ptr(true),
							TextOut: Ptr("stdout"),
							Json:    Ptr(false),
							JsonOut: Ptr("stdout"),
						},
						OtelLogging: &OtelLogging{
							Enabled: Ptr(true),
						},
					},
					Metrics: &Metrics{
						Enabled: true,
					},
					Tracing: &Tracing{
						Enabled: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "fail if slog text_out is not valid",
			args:    args{name: "testdata/not_valid_text_out.hcl"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "fail if slog json_out is not valid",
			args:    args{name: "testdata/not_valid_json_out.hcl"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "log level is default info",
			args: args{name: "testdata/emptylogging.hcl"},
			want: &Config{
				Port: port,
				Bind: bind,

				Nomad: &NomadServer{
					Address: nomadAddr,
				},
				Validators: []Validator{},
				Mutators:   []Mutator{},
				Telemetry: &Telemetry{
					Logging: &Logging{
						Level: "info",
						SlogLogging: &SlogLogging{
							Text:    Ptr(true),
							TextOut: Ptr("stdout"),
							Json:    Ptr(false),
							JsonOut: Ptr("stdout"),
						},
						OtelLogging: &OtelLogging{
							Enabled: Ptr(false),
						},
					},
					Metrics: &Metrics{
						Enabled: false,
					},
					Tracing: &Tracing{
						Enabled: false,
					},
				},
			},

			wantErr: false,
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
