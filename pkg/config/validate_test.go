package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateRejectsIncompleteControllers covers the configurations that used
// to reach startup and panic on a nil block, or fail late once the OPA SDK was
// already running.
func TestValidateRejectsIncompleteControllers(t *testing.T) {
	tt := []struct {
		name      string
		mutate    func(*Config)
		expectErr string
	}{
		{
			name: "opa validator without opa_rule",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "opa", Name: "x"}}
			},
			expectErr: `validator "x" of type "opa" requires a opa_rule block`,
		},
		{
			name: "opa validator without filename",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "opa", Name: "x", OpaRule: &OpaRule{Query: "q"}}}
			},
			expectErr: `validator "x" requires opa_rule.filename`,
		},
		{
			name: "webhook mutator without webhook block",
			mutate: func(c *Config) {
				c.Mutators = []Mutator{{Type: "json_patch_webhook", Name: "x"}}
			},
			expectErr: `mutator "x" of type "json_patch_webhook" requires a webhook block`,
		},
		{
			name: "notation validator without notation block",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "notation", Name: "x"}}
			},
			expectErr: `validator "x" of type "notation" requires a notation block`,
		},
		{
			name: "unknown validator type",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "opa_sdk", Name: "x"}}
			},
			expectErr: "unknown validator type opa_sdk",
		},
		{
			name: "bundle validator without bundle_rule",
			mutate: func(c *Config) {
				c.OpaBundles = []OpaBundle{{Id: "platform", ConfigPath: "/opa.yml"}}
				c.Validators = []Validator{{Type: "opa_bundle", Name: "x"}}
			},
			expectErr: `validator "x" of type "opa_bundle" requires a bundle_rule block`,
		},
		{
			name: "bundle validator without any configured bundle",
			mutate: func(c *Config) {
				c.Validators = []Validator{{Type: "opa_bundle", Name: "x", BundleRule: &BundleRule{Path: "/p"}}}
			},
			expectErr: `validator "x" references a bundle but no opa_bundle block is configured`,
		},
		{
			name: "bundle validator must name a source when several exist",
			mutate: func(c *Config) {
				c.OpaBundles = []OpaBundle{
					{Id: "platform", ConfigPath: "/a.yml"},
					{Id: "team", ConfigPath: "/b.yml"},
				}
				c.Validators = []Validator{{Type: "opa_bundle", Name: "x", BundleRule: &BundleRule{Path: "/p"}}}
			},
			expectErr: `validator "x" must set bundle_rule.source, configured bundles: platform, team`,
		},
		{
			name: "bundle validator with unknown source",
			mutate: func(c *Config) {
				c.OpaBundles = []OpaBundle{{Id: "platform", ConfigPath: "/a.yml"}}
				c.Validators = []Validator{{Type: "opa_bundle", Name: "x", BundleRule: &BundleRule{Source: "nope", Path: "/p"}}}
			},
			expectErr: `validator "x" references unknown bundle_rule.source "nope", configured bundles: platform`,
		},
		{
			name: "duplicate bundle id",
			mutate: func(c *Config) {
				c.OpaBundles = []OpaBundle{
					{Id: "platform", ConfigPath: "/a.yml"},
					{Id: "platform", ConfigPath: "/b.yml"},
				}
			},
			expectErr: `duplicate opa_bundle "platform"`,
		},
		{
			name: "bundle without config_path",
			mutate: func(c *Config) {
				c.OpaBundles = []OpaBundle{{Id: "platform"}}
			},
			expectErr: `opa_bundle "platform" requires config_path`,
		},
		{
			name: "bundle with unparsable timeout",
			mutate: func(c *Config) {
				c.OpaBundles = []OpaBundle{{Id: "platform", ConfigPath: "/a.yml", ReadyTimeout: Ptr("soon")}}
			},
			expectErr: `opa_bundle "platform": invalid ready_timeout "soon"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c := DefaultConfig()
			tc.mutate(c)
			assert.ErrorContains(t, c.Validate(), tc.expectErr)
		})
	}
}

func TestValidateAcceptsSingleBundleWithoutSource(t *testing.T) {
	c := DefaultConfig()
	c.OpaBundles = []OpaBundle{{Id: "platform", ConfigPath: "/a.yml"}}
	c.Validators = []Validator{{Type: "opa_bundle", Name: "x", BundleRule: &BundleRule{Path: "/p"}}}

	assert.NoError(t, c.Validate())
}

func TestBundleTimeoutDefaults(t *testing.T) {
	b := OpaBundle{Id: "platform", ConfigPath: "/a.yml"}

	ready, err := b.ResolvedReadyTimeout()
	require.NoError(t, err)
	assert.Equal(t, DefaultBundleReadyTimeout, ready)

	decision, err := b.ResolvedDecisionTimeout()
	require.NoError(t, err)
	assert.Equal(t, DefaultBundleDecisionTimeout, decision)

	// An explicit zero opts out of the per-decision deadline.
	b.DecisionTimeout = Ptr("0s")
	decision, err = b.ResolvedDecisionTimeout()
	require.NoError(t, err)
	assert.Zero(t, decision)
}
