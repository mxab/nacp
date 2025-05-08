package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/nomad/api"
)

type Webhook struct {
	Endpoint string `hcl:"endpoint"`
	Method   string `hcl:"method"`
}
type OpaRule struct {
	Query    string                  `hcl:"query"`
	Filename string                  `hcl:"filename"`
	Notation *NotationVerifierConfig `hcl:"notation,block"`
}

type Validator struct {
	Type         string   `hcl:"type,label"`
	Name         string   `hcl:"name,label"`
	OpaRule      *OpaRule `hcl:"opa_rule,block"`
	Webhook      *Webhook `hcl:"webhook,block"`
	ResolveToken bool     `hcl:"resolve_token,optional"`

	Notation *NotationVerifierConfig `hcl:"notation,block"`
}
type Mutator struct {
	Type         string   `hcl:"type,label"`
	Name         string   `hcl:"name,label"`
	OpaRule      *OpaRule `hcl:"opa_rule,block"`
	Webhook      *Webhook `hcl:"webhook,block"`
	ResolveToken bool     `hcl:"resolve_token,optional"`
}

type RequestContext struct {
	ClientIP     string        `json:"clientIP"`
	AccessorID   string        `json:"accessorID"`
	ResolveToken bool          `json:"resolveToken"`
	TokenInfo    *api.ACLToken `json:"tokenInfo,omitempty"`
}

type NomadServerTLS struct {
	CaFile             string `hcl:"ca_file"`
	CertFile           string `hcl:"cert_file"`
	KeyFile            string `hcl:"key_file"`
	InsecureSkipVerify bool   `hcl:"insecure_skip_verify,optional"`
}
type NomadServer struct {
	Address string          `hcl:"address"`
	TLS     *NomadServerTLS `hcl:"tls,block"`
}
type ProxyTLS struct {
	CertFile     string `hcl:"cert_file"`
	KeyFile      string `hcl:"key_file"`
	CaFile       string `hcl:"ca_file"`
	NoClientCert bool   `hcl:"no_client_cert,optional"`
}
type NotationVerifierConfig struct {
	TrustPolicyFile     string `hcl:"trust_policy_file"`
	TrustStoreDir       string `hcl:"trust_store_dir"`
	RepoPlainHTTP       bool   `hcl:"repo_plain_http,optional"`
	MaxSigAttempts      int    `hcl:"max_sig_attempts,optional"`
	CredentialStoreFile string `hcl:"credential_store_file,optional"`
}

type SlogLogging struct {
	Handler string `hcl:"handler,optional"` // "json" or "text"
}
type Logging struct {
	Level       string       `hcl:"level,optional"`
	Type        string       `hcl:"type,optional"` // "slog" or "otel"
	SlogLogging *SlogLogging `hcl:"slog,block"`
}
type Metrics struct {
	Enabled bool `hcl:"enabled,optional"`
	// only otel for now
}
type Tracing struct {
	Enabled bool `hcl:"enabled,optional"`
	// only otel for now
}
type Telemetry struct {
	Logging *Logging `hcl:"logging,block"`
	Metrics *Metrics `hcl:"metrics,block"`
	Tracing *Tracing `hcl:"tracing,block"`
}
type Config struct {
	Port int    `hcl:"port,optional"`
	Bind string `hcl:"bind,optional"`

	LogLevel string    `hcl:"log_level,optional"`
	Tls      *ProxyTLS `hcl:"tls,block"`

	Nomad      *NomadServer `hcl:"nomad,block"`
	Validators []Validator  `hcl:"validator,block"`
	Mutators   []Mutator    `hcl:"mutator,block"`

	Telemetry *Telemetry `hcl:"telemetry,block"`
}

func DefaultConfig() *Config {
	c := &Config{
		Port: 6464,
		Bind: "0.0.0.0",
		Nomad: &NomadServer{
			Address: "http://localhost:4646",
		},
		LogLevel:   "info",
		Validators: []Validator{},
		Mutators:   []Mutator{},
		Telemetry: &Telemetry{
			Logging: &Logging{
				Level: "info",
				Type:  "slog",
				SlogLogging: &SlogLogging{
					Handler: "text",
				},
			},
		},
	}
	return c
}
func LoadConfig(name string) (*Config, error) {

	c := DefaultConfig()

	evalContext := &hcl.EvalContext{}
	err := hclsimple.DecodeFile(name, evalContext, c)
	if err != nil {
		return nil, err
	}

	// set default on all Notation Verifiers, is there a better way to do this?
	for _, v := range c.Validators {
		if v.Notation != nil && v.Notation.MaxSigAttempts == 0 {
			v.Notation.MaxSigAttempts = 50

		}
	}

	return c, nil
}
