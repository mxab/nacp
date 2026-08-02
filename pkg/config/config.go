package config

import (
	"fmt"
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

// BundleRule selects the decision to evaluate inside a bundle. Source names the
// opa_bundle block to evaluate it against and may be omitted when exactly one
// bundle is configured.
type BundleRule struct {
	Source string `hcl:"source,optional"`
	Path   string `hcl:"path"`
}

type Validator struct {
	Type         string      `hcl:"type,label"`
	Name         string      `hcl:"name,label"`
	OpaRule      *OpaRule    `hcl:"opa_rule,block"`
	BundleRule   *BundleRule `hcl:"bundle_rule,block"`
	Webhook      *Webhook    `hcl:"webhook,block"`
	ResolveToken bool        `hcl:"resolve_token,optional"`

	Notation *NotationVerifierConfig `hcl:"notation,block"`
}
type Mutator struct {
	Type         string      `hcl:"type,label"`
	Name         string      `hcl:"name,label"`
	OpaRule      *OpaRule    `hcl:"opa_rule,block"`
	BundleRule   *BundleRule `hcl:"bundle_rule,block"`
	Webhook      *Webhook    `hcl:"webhook,block"`
	ResolveToken bool        `hcl:"resolve_token,optional"`
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

	OpaBundles []OpaBundle `hcl:"opa_bundle,block"`
}

// OpaBundle is one OPA SDK instance fed by its own OPA configuration file, and
// therefore its own bundle services, signing keys and refresh settings.
type OpaBundle struct {
	Id         string `hcl:"id,label"`
	ConfigPath string `hcl:"config_path"`

	// ReadyTimeout bounds how long startup waits for the first bundle
	// activation. Defaults to DefaultBundleReadyTimeout.
	ReadyTimeout *string `hcl:"ready_timeout,optional"`
	// DecisionTimeout bounds a single policy evaluation. Defaults to
	// DefaultBundleDecisionTimeout; "0" inherits the request deadline.
	DecisionTimeout *string `hcl:"decision_timeout,optional"`
	// RequireSigning refuses to start unless every bundle in the OPA
	// configuration is configured for signature verification.
	RequireSigning bool `hcl:"require_signing,optional"`
}

const (
	DefaultBundleReadyTimeout    = 30 * time.Second
	DefaultBundleDecisionTimeout = 5 * time.Second
)

// ResolvedReadyTimeout parses ready_timeout, falling back to the default.
func (b OpaBundle) ResolvedReadyTimeout() (time.Duration, error) {
	return parseOptionalDuration(b.ReadyTimeout, DefaultBundleReadyTimeout, "ready_timeout")
}

// ResolvedDecisionTimeout parses decision_timeout, falling back to the default.
// A zero duration means decisions are only bounded by the request context.
func (b OpaBundle) ResolvedDecisionTimeout() (time.Duration, error) {
	return parseOptionalDuration(b.DecisionTimeout, DefaultBundleDecisionTimeout, "decision_timeout")
}

func parseOptionalDuration(raw *string, fallback time.Duration, field string) (time.Duration, error) {
	if raw == nil || *raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(*raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", field, *raw, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("invalid %s %q: must not be negative", field, *raw)
	}
	return d, nil
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

	// set default on all Notation Verifiers, is there a better way to do this?
	for _, v := range c.Validators {
		if v.Notation != nil && v.Notation.MaxSigAttempts == 0 {
			v.Notation.MaxSigAttempts = 50

		}
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Validate rejects a configuration before anything is built from it, so that a
// missing block surfaces as a config error rather than a nil dereference or a
// half-started process.
func (c *Config) Validate() error {
	// verify json/text out
	var validOuts = []string{"stdout", "stderr"}
	if !slices.Contains(validOuts, *c.Telemetry.Logging.SlogLogging.TextOut) {
		return fmt.Errorf("invalid slog text output: %s", *c.Telemetry.Logging.SlogLogging.TextOut)
	}
	if !slices.Contains(validOuts, *c.Telemetry.Logging.SlogLogging.JsonOut) {
		return fmt.Errorf("invalid slog json output: %s", *c.Telemetry.Logging.SlogLogging.JsonOut)
	}

	bundleIds, err := c.validateBundles()
	if err != nil {
		return err
	}

	for _, v := range c.Validators {
		if err := validateController("validator", v.Type, v.Name, controllerBlocks{
			opaRule:    v.OpaRule,
			bundleRule: v.BundleRule,
			webhook:    v.Webhook,
			notation:   v.Notation,
		}, bundleIds); err != nil {
			return err
		}
	}
	for _, m := range c.Mutators {
		if err := validateController("mutator", m.Type, m.Name, controllerBlocks{
			opaRule:    m.OpaRule,
			bundleRule: m.BundleRule,
			webhook:    m.Webhook,
		}, bundleIds); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) validateBundles() ([]string, error) {
	ids := make([]string, 0, len(c.OpaBundles))
	for _, b := range c.OpaBundles {
		if b.Id == "" {
			return nil, fmt.Errorf("opa_bundle block requires a non-empty id label")
		}
		if slices.Contains(ids, b.Id) {
			return nil, fmt.Errorf("duplicate opa_bundle %q", b.Id)
		}
		if b.ConfigPath == "" {
			return nil, fmt.Errorf("opa_bundle %q requires config_path", b.Id)
		}
		if _, err := b.ResolvedReadyTimeout(); err != nil {
			return nil, fmt.Errorf("opa_bundle %q: %w", b.Id, err)
		}
		if _, err := b.ResolvedDecisionTimeout(); err != nil {
			return nil, fmt.Errorf("opa_bundle %q: %w", b.Id, err)
		}
		ids = append(ids, b.Id)
	}
	return ids, nil
}

type controllerBlocks struct {
	opaRule    *OpaRule
	bundleRule *BundleRule
	webhook    *Webhook
	notation   *NotationVerifierConfig
}

func validateController(role, typ, name string, blocks controllerBlocks, bundleIds []string) error {
	required := func(present bool, block string) error {
		if present {
			return nil
		}
		return fmt.Errorf("%s %q of type %q requires a %s block", role, name, typ, block)
	}

	switch typ {
	case "opa", "opa_json_patch":
		if err := required(blocks.opaRule != nil, "opa_rule"); err != nil {
			return err
		}
		if blocks.opaRule.Filename == "" {
			return fmt.Errorf("%s %q requires opa_rule.filename", role, name)
		}
		if blocks.opaRule.Query == "" {
			return fmt.Errorf("%s %q requires opa_rule.query", role, name)
		}
	case "opa_bundle", "opa_bundle_json_patch":
		if err := required(blocks.bundleRule != nil, "bundle_rule"); err != nil {
			return err
		}
		if blocks.bundleRule.Path == "" {
			return fmt.Errorf("%s %q requires bundle_rule.path", role, name)
		}
		if err := validateBundleSource(role, name, blocks.bundleRule.Source, bundleIds); err != nil {
			return err
		}
	case "webhook", "json_patch_webhook":
		if err := required(blocks.webhook != nil, "webhook"); err != nil {
			return err
		}
		if blocks.webhook.Endpoint == "" {
			return fmt.Errorf("%s %q requires webhook.endpoint", role, name)
		}
	case "notation":
		if err := required(blocks.notation != nil, "notation"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown %s type %s", role, typ)
	}
	return nil
}

func validateBundleSource(role, name, source string, bundleIds []string) error {
	switch {
	case len(bundleIds) == 0:
		return fmt.Errorf("%s %q references a bundle but no opa_bundle block is configured", role, name)
	case source == "" && len(bundleIds) > 1:
		return fmt.Errorf("%s %q must set bundle_rule.source, configured bundles: %s",
			role, name, strings.Join(bundleIds, ", "))
	case source != "" && !slices.Contains(bundleIds, source):
		return fmt.Errorf("%s %q references unknown bundle_rule.source %q, configured bundles: %s",
			role, name, source, strings.Join(bundleIds, ", "))
	}
	return nil
}
