package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/nomad/api"
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
							Filename: "testdata/opa/validators/costcenter_meta/costcenter_meta.rego",
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
		{
			name: "with opa sdk",
			args: args{name: "testdata/with_opa_sdk.hcl"},
			want: &Config{
				Port: port,
				Bind: bind,

				Nomad: &NomadServer{
					Address: nomadAddr,
				},
				Validators: []Validator{
					{
						Type: "opa_bundle",
						Name: "some_validator",
						OpaSdkRule: &OpaSdkRule{
							Path: "/my/validation/policy",
						},
					},
				},
				Mutators: []Mutator{
					{
						Type: "opa_bundle_json_patch",
						Name: "some_mutator",
						OpaSdkRule: &OpaSdkRule{
							Path: "/my/mutation/policy",
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
				OpaSdk: &OpaSdk{

					Id:         "example",
					ConfigPath: "/my/path/to/config.json",
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

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name: "invalid port",
			mutate: func(c *Config) {
				c.Port = 0
			},
			wantErr: "port must be between",
		},
		{
			name: "missing webhook block",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "webhook", Name: "remote"}}
			},
			wantErr: "requires a webhook block",
		},
		{
			name: "bundle controller without SDK",
			mutate: func(c *Config) {
				c.Mutators = []Mutator{{Type: "opa_bundle_json_patch", Name: "bundle", OpaSdkRule: &OpaSdkRule{Path: "/patch"}}}
			},
			wantErr: "requires an opa_sdk block",
		},
		{
			name: "relative webhook URL",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "webhook", Name: "remote", Webhook: &Webhook{Endpoint: "/validate", Method: "POST"}}}
			},
			wantErr: "absolute HTTP(S) URL",
		},
		{
			name: "blank bind address",
			mutate: func(c *Config) {
				c.Bind = "  "
			},
			wantErr: "bind address is required",
		},
		{
			name: "missing nomad block",
			mutate: func(c *Config) {
				c.Nomad = nil
			},
			wantErr: "nomad block is required",
		},
		{
			name: "relative nomad address",
			mutate: func(c *Config) {
				c.Nomad.Address = "localhost:4646"
			},
			wantErr: "nomad address must be an absolute HTTP(S) URL",
		},
		{
			name: "listener TLS without key file",
			mutate: func(c *Config) {
				c.Tls = &ProxyTLS{CertFile: "cert.pem"}
			},
			wantErr: "listener TLS requires cert_file and key_file",
		},
		{
			name: "nomad TLS with only a cert file",
			mutate: func(c *Config) {
				c.Nomad.TLS = &NomadServerTLS{CertFile: "cert.pem"}
			},
			wantErr: "nomad TLS cert_file and key_file must be configured together",
		},
		{
			name: "opa_sdk without config path",
			mutate: func(c *Config) {
				c.OpaSdk = &OpaSdk{Id: "bundles"}
			},
			wantErr: "opa_sdk requires a non-empty id and config_path",
		},
		{
			name: "unknown mutator type",
			mutate: func(c *Config) {
				c.Mutators = []Mutator{{Type: "wat", Name: "unknown"}}
			},
			wantErr: `unknown mutator type "wat"`,
		},
		{
			name: "unknown validator type",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "wat", Name: "unknown"}}
			},
			wantErr: `unknown validator type "wat"`,
		},
		{
			name: "mutator without a name",
			mutate: func(c *Config) {
				c.Mutators = []Mutator{{Type: "opa_json_patch", Name: " "}}
			},
			wantErr: "mutator name is required",
		},
		{
			name: "validator without a name",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "opa", Name: ""}}
			},
			wantErr: "validator name is required",
		},
		{
			name: "duplicate mutator name",
			mutate: func(c *Config) {
				mutator := Mutator{Type: "opa_json_patch", Name: "dupe", OpaRule: &OpaRule{Filename: "rule.rego", Query: "patch"}}
				c.Mutators = []Mutator{mutator, mutator}
			},
			wantErr: `duplicate mutator name "dupe"`,
		},
		{
			name: "duplicate validator name",
			mutate: func(c *Config) {
				validator := Validator{Type: "opa", Name: "dupe", OpaRule: &OpaRule{Filename: "rule.rego", Query: "errors"}}
				c.Validators = []Validator{validator, validator}
			},
			wantErr: `duplicate validator name "dupe"`,
		},
		{
			name: "opa rule without a query",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "opa", Name: "policy", OpaRule: &OpaRule{Filename: "rule.rego"}}}
			},
			wantErr: "requires an opa_rule block with filename and query",
		},
		{
			name: "opa_bundle rule without a path",
			mutate: func(c *Config) {
				c.OpaSdk = &OpaSdk{Id: "bundles", ConfigPath: "opa.yml"}
				c.Validators = []Validator{{Type: "opa_bundle", Name: "bundle"}}
			},
			wantErr: "requires an opa_sdk_rule block with path",
		},
		{
			name: "webhook without a method",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "webhook", Name: "remote", Webhook: &Webhook{Endpoint: "http://localhost:8080/validate"}}}
			},
			wantErr: "requires a webhook method",
		},
		{
			name: "webhook with an invalid method",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "webhook", Name: "remote", Webhook: &Webhook{Endpoint: "http://localhost:8080/validate", Method: "BAD METHOD"}}}
			},
			wantErr: "has an invalid webhook method",
		},
		{
			name: "notation without a trust store dir",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "notation", Name: "signed", Notation: &NotationVerifierConfig{TrustPolicyFile: "policy.json", MaxSigAttempts: 1}}}
			},
			wantErr: "requires notation trust_policy_file and trust_store_dir",
		},
		{
			name: "notation with non positive max_sig_attempts",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "notation", Name: "signed", Notation: &NotationVerifierConfig{
					TrustPolicyFile: "policy.json",
					TrustStoreDir:   "truststore",
					MaxSigAttempts:  0,
				}}}
			},
			wantErr: "max_sig_attempts must be positive",
		},
		{
			name: "notation attached to an opa rule",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "opa", Name: "policy", OpaRule: &OpaRule{
					Filename: "rule.rego",
					Query:    "errors",
					Notation: &NotationVerifierConfig{TrustPolicyFile: "policy.json"},
				}}}
			},
			wantErr: "requires notation trust_policy_file and trust_store_dir",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := DefaultConfig()
			tc.mutate(c)
			assert.ErrorContains(t, c.Validate(), tc.wantErr)
		})
	}

	t.Run("default config is valid", func(t *testing.T) {
		assert.NoError(t, DefaultConfig().Validate())
	})

	t.Run("nil config", func(t *testing.T) {
		var c *Config
		assert.ErrorContains(t, c.Validate(), "config is nil")
	})
}

func TestSanitizeACLTokenExcludesSecretID(t *testing.T) {
	token := &api.ACLToken{
		AccessorID: "accessor",
		SecretID:   "must-not-leak",
		Policies:   []string{"read"},
	}

	sanitized := SanitizeACLToken(token)
	data, err := json.Marshal(sanitized)
	require.NoError(t, err)
	assert.Equal(t, "accessor", sanitized.AccessorID)
	assert.Equal(t, []string{"read"}, sanitized.Policies)
	assert.False(t, strings.Contains(string(data), token.SecretID))
}

func TestLoadConfigDefaults(t *testing.T) {

	defaultConfig := &Config{
		Port: 6464,
		Bind: "0.0.0.0",

		Nomad: &NomadServer{
			Address: "http://localhost:4646",
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
	}

	tt := []struct {
		name         string
		configAsText string
		want         *Config
	}{
		{
			name:         "empty",
			configAsText: ``,
			want:         defaultConfig,
		},
		{
			name:         "just telemetry empty",
			configAsText: `telemetry {}`,
			want:         defaultConfig,
		},
		{
			name: "just telemetry logging empty",
			configAsText: `telemetry {
			logging {
			}
			}`,
			want: defaultConfig,
		},
		{
			name: "just telemetry logging slog empty",
			configAsText: `telemetry {
			logging {
				slog {
				}
			}
			}`,
			want: defaultConfig,
		},
		{
			name: "just telemetry logging otel empty",
			configAsText: `telemetry {
			logging {
				otel {
				}
			}
			}`,
			want: defaultConfig,
		},
		{
			name: "just telemetry metric empty",
			configAsText: `telemetry {
			metrics {
			}
			}`,
			want: defaultConfig,
		},
		{
			name: "just telemetry tracing empty",
			configAsText: `telemetry {
			tracing {
			}
			}`,
			want: defaultConfig,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			dir := t.TempDir()
			configFile := filepath.Join(dir, "config.hcl")
			os.WriteFile(configFile, []byte(tc.configAsText), 0644)
			config, err := LoadConfig(configFile)

			require.NoError(t, err)
			assert.EqualValues(t, tc.want, config)

		})
	}
}
