package config

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/nomad/api"
)

func Ptr[T any](v T) *T {
	return &v
}

type Webhook struct {
	Endpoint string `hcl:"endpoint"`
	Method   string `hcl:"method"`
}
type OpaRule struct {
	Query    string                  `hcl:"query"`
	Filename string                  `hcl:"filename"`
	Notation *NotationVerifierConfig `hcl:"notation,block"`
}
type OpaSdkRule struct {
	Path string `hcl:"path"`
}

type Validator struct {
	Type         string      `hcl:"type,label"`
	Name         string      `hcl:"name,label"`
	OpaRule      *OpaRule    `hcl:"opa_rule,block"`
	OpaSdkRule   *OpaSdkRule `hcl:"opa_sdk_rule,block"`
	Webhook      *Webhook    `hcl:"webhook,block"`
	ResolveToken bool        `hcl:"resolve_token,optional"`

	Notation *NotationVerifierConfig `hcl:"notation,block"`
}
type Mutator struct {
	Type         string      `hcl:"type,label"`
	Name         string      `hcl:"name,label"`
	OpaRule      *OpaRule    `hcl:"opa_rule,block"`
	OpaSdkRule   *OpaSdkRule `hcl:"opa_sdk_rule,block"`
	Webhook      *Webhook    `hcl:"webhook,block"`
	ResolveToken bool        `hcl:"resolve_token,optional"`
}

type RequestContext struct {
	ClientIP     string           `json:"clientIP"`
	AccessorID   string           `json:"accessorID"`
	ResolveToken bool             `json:"resolveToken"`
	TokenInfo    *ACLTokenContext `json:"tokenInfo,omitempty"`
}

type ACLTokenContext struct {
	AccessorID     string
	Name           string
	Type           string
	Policies       []string
	Roles          []*api.ACLTokenRoleLink
	Global         bool
	CreateTime     time.Time
	ExpirationTime *time.Time `json:",omitempty"`
	CreateIndex    uint64
	ModifyIndex    uint64
}

func SanitizeACLToken(token *api.ACLToken) *ACLTokenContext {
	if token == nil {
		return nil
	}
	return &ACLTokenContext{
		AccessorID:     token.AccessorID,
		Name:           token.Name,
		Type:           token.Type,
		Policies:       slices.Clone(token.Policies),
		Roles:          slices.Clone(token.Roles),
		Global:         token.Global,
		CreateTime:     token.CreateTime,
		ExpirationTime: token.ExpirationTime,
		CreateIndex:    token.CreateIndex,
		ModifyIndex:    token.ModifyIndex,
	}
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
	Text    *bool   `hcl:"text,optional"`
	TextOut *string `hcl:"text_out,optional"`

	Json    *bool   `hcl:"json,optional"`
	JsonOut *string `hcl:"json_out,optional"`
}
type OtelLogging struct {
	Enabled *bool `hcl:"enabled,optional"`
}
type Logging struct {
	Level       string       `hcl:"level,optional"`
	SlogLogging *SlogLogging `hcl:"slog,block"`
	OtelLogging *OtelLogging `hcl:"otel,block"`
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

	Tls *ProxyTLS `hcl:"tls,block"`

	Nomad      *NomadServer `hcl:"nomad,block"`
	Validators []Validator  `hcl:"validator,block"`
	Mutators   []Mutator    `hcl:"mutator,block"`

	Telemetry *Telemetry `hcl:"telemetry,block"`

	OpaSdk *OpaSdk `hcl:"opa_sdk,block"`
}
type OpaSdk struct {
	Id         string `hcl:"id,label"`
	ConfigPath string `hcl:"config_path"`
}

func DefaultConfig() *Config {
	c := &Config{
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
	return c
}
func LoadConfig(name string) (*Config, error) {

	c := DefaultConfig()

	evalContext := &hcl.EvalContext{}
	err := hclsimple.DecodeFile(name, evalContext, c)
	if err != nil {
		return nil, err
	}

	for i := range c.Validators {
		setNotationDefaults(c.Validators[i].Notation)
		if c.Validators[i].OpaRule != nil {
			setNotationDefaults(c.Validators[i].OpaRule.Notation)
		}
	}
	for i := range c.Mutators {
		if c.Mutators[i].OpaRule != nil {
			setNotationDefaults(c.Mutators[i].OpaRule.Notation)
		}
	}

	// verify json/text out
	var validOuts = []string{"stdout", "stderr"}
	if !slices.Contains(validOuts, *c.Telemetry.Logging.SlogLogging.TextOut) {
		return nil, fmt.Errorf("invalid slog text output: %s", *c.Telemetry.Logging.SlogLogging.TextOut)
	}
	if !slices.Contains(validOuts, *c.Telemetry.Logging.SlogLogging.JsonOut) {
		return nil, fmt.Errorf("invalid slog json output: %s", *c.Telemetry.Logging.SlogLogging.JsonOut)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func setNotationDefaults(notation *NotationVerifierConfig) {
	if notation != nil && notation.MaxSigAttempts == 0 {
		notation.MaxSigAttempts = 50
	}
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if err := validateListener(c); err != nil {
		return err
	}
	if err := validateNomadConfig(c); err != nil {
		return err
	}
	if c.OpaSdk != nil && (strings.TrimSpace(c.OpaSdk.Id) == "" || strings.TrimSpace(c.OpaSdk.ConfigPath) == "") {
		return fmt.Errorf("opa_sdk requires a non-empty id and config_path")
	}
	return validateControllers(c)
}

func validateListener(c *Config) error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if strings.TrimSpace(c.Bind) == "" {
		return fmt.Errorf("bind address is required")
	}
	if c.Tls != nil && (c.Tls.CertFile == "" || c.Tls.KeyFile == "") {
		return fmt.Errorf("listener TLS requires cert_file and key_file")
	}
	return nil
}

func validateNomadConfig(c *Config) error {
	if c.Nomad == nil {
		return fmt.Errorf("nomad block is required")
	}
	if err := validateHTTPURL("nomad address", c.Nomad.Address); err != nil {
		return err
	}
	if c.Nomad.TLS != nil && ((c.Nomad.TLS.CertFile == "") != (c.Nomad.TLS.KeyFile == "")) {
		return fmt.Errorf("nomad TLS cert_file and key_file must be configured together")
	}
	return nil
}

// validateControllers checks every mutator and validator and rejects duplicate
// names within each kind.
func validateControllers(c *Config) error {
	hasOpaSDK := c.OpaSdk != nil
	seen := make(map[string]struct{}, len(c.Mutators)+len(c.Validators))
	for _, mutator := range c.Mutators {
		if err := validateMutator(mutator, hasOpaSDK); err != nil {
			return err
		}
		key := "mutator:" + mutator.Name
		if _, found := seen[key]; found {
			return fmt.Errorf("duplicate mutator name %q", mutator.Name)
		}
		seen[key] = struct{}{}
	}
	for _, validator := range c.Validators {
		if err := validateValidator(validator, hasOpaSDK); err != nil {
			return err
		}
		key := "validator:" + validator.Name
		if _, found := seen[key]; found {
			return fmt.Errorf("duplicate validator name %q", validator.Name)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateMutator(mutator Mutator, hasOpaSDK bool) error {
	if strings.TrimSpace(mutator.Name) == "" {
		return fmt.Errorf("mutator name is required")
	}
	switch mutator.Type {
	case "opa_json_patch":
		return validateOpaRule("mutator", mutator.Name, mutator.OpaRule)
	case "json_patch_webhook":
		return validateWebhook("mutator", mutator.Name, mutator.Webhook)
	case "opa_bundle_json_patch":
		return validateOpaSDKRule("mutator", mutator.Name, mutator.OpaSdkRule, hasOpaSDK)
	default:
		return fmt.Errorf("unknown mutator type %q", mutator.Type)
	}
}

func validateValidator(validator Validator, hasOpaSDK bool) error {
	if strings.TrimSpace(validator.Name) == "" {
		return fmt.Errorf("validator name is required")
	}
	switch validator.Type {
	case "opa":
		if err := validateOpaRule("validator", validator.Name, validator.OpaRule); err != nil {
			return err
		}
		if validator.Notation != nil {
			return validateNotation("validator", validator.Name, validator.Notation)
		}
		return nil
	case "opa_bundle":
		return validateOpaSDKRule("validator", validator.Name, validator.OpaSdkRule, hasOpaSDK)
	case "webhook":
		return validateWebhook("validator", validator.Name, validator.Webhook)
	case "notation":
		return validateNotation("validator", validator.Name, validator.Notation)
	default:
		return fmt.Errorf("unknown validator type %q", validator.Type)
	}
}

func validateOpaRule(kind, name string, rule *OpaRule) error {
	if rule == nil || strings.TrimSpace(rule.Filename) == "" || strings.TrimSpace(rule.Query) == "" {
		return fmt.Errorf("%s %q requires an opa_rule block with filename and query", kind, name)
	}
	if rule.Notation != nil {
		return validateNotation(kind, name, rule.Notation)
	}
	return nil
}

func validateOpaSDKRule(kind, name string, rule *OpaSdkRule, hasOpaSDK bool) error {
	if !hasOpaSDK {
		return fmt.Errorf("%s %q requires an opa_sdk block", kind, name)
	}
	if rule == nil || strings.TrimSpace(rule.Path) == "" {
		return fmt.Errorf("%s %q requires an opa_sdk_rule block with path", kind, name)
	}
	return nil
}

func validateWebhook(kind, name string, webhook *Webhook) error {
	if webhook == nil {
		return fmt.Errorf("%s %q requires a webhook block", kind, name)
	}
	if err := validateHTTPURL("webhook endpoint", webhook.Endpoint); err != nil {
		return fmt.Errorf("%s %q: %w", kind, name, err)
	}
	if strings.TrimSpace(webhook.Method) == "" {
		return fmt.Errorf("%s %q requires a webhook method", kind, name)
	}
	if _, err := http.NewRequest(webhook.Method, webhook.Endpoint, nil); err != nil {
		return fmt.Errorf("%s %q has an invalid webhook method: %w", kind, name, err)
	}
	return nil
}

func validateNotation(kind, name string, notation *NotationVerifierConfig) error {
	if notation == nil || notation.TrustPolicyFile == "" || notation.TrustStoreDir == "" {
		return fmt.Errorf("%s %q requires notation trust_policy_file and trust_store_dir", kind, name)
	}
	if notation.MaxSigAttempts < 1 {
		return fmt.Errorf("%s %q max_sig_attempts must be positive", kind, name)
	}
	return nil
}

func validateHTTPURL(name, value string) error {
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", name, err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%s must be an absolute HTTP(S) URL", name)
	}
	return nil
}
