package bundle

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/pkg/admissionctrl/types"
	"github.com/mxab/nacp/pkg/config"
	"github.com/mxab/nacp/pkg/logutil"
	sdktest "github.com/open-policy-agent/opa/v1/sdk/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPayload() *types.Payload {
	id := "test-job"
	return &types.Payload{Job: &api.Job{ID: &id}}
}

func discardFactory(t *testing.T) *logutil.LoggerFactory {
	t.Helper()
	factory, _ := logutil.NewLoggerFactory(io.Discard, io.Discard, false)
	return factory
}

// writeConfig starts a mock bundle server serving policy and writes an OPA
// configuration pointing at it.
func writeConfig(t *testing.T, name, policy string) string {
	t.Helper()

	server, err := sdktest.NewServer(sdktest.MockBundle("/bundles/bundle.tar.gz", map[string]string{
		"example.rego": policy,
	}))
	require.NoError(t, err)
	t.Cleanup(server.Stop)

	return writeConfigFile(t, name, fmt.Sprintf(`{
		"services": {%q: {"url": %q}},
		"bundles": {%q: {"service": %q, "resource": "/bundles/bundle.tar.gz"}}
	}`, name, server.URL(), name, name))
}

func writeConfigFile(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name+".json")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

func TestSetupResolvesEachSourceToItsOwnBundle(t *testing.T) {
	configs := []config.OpaBundle{
		{Id: "platform", ConfigPath: writeConfig(t, "platform", `package platformpolicy
		errors = ["platform says no"]`)},
		{Id: "team", ConfigPath: writeConfig(t, "team", `package teampolicy
		errors = ["team says no"]`)},
	}

	registry, stop, err := Setup(t.Context(), discardFactory(t), configs)
	require.NoError(t, err)
	t.Cleanup(stop)

	platform, err := registry.Get("platform")
	require.NoError(t, err)
	decision, err := platform.Decide(t.Context(), "/platformpolicy", testPayload())
	require.NoError(t, err)
	require.Len(t, decision.Errors, 1)
	assert.EqualError(t, decision.Errors[0], "platform says no")

	team, err := registry.Get("team")
	require.NoError(t, err)
	decision, err = team.Decide(t.Context(), "/teampolicy", testPayload())
	require.NoError(t, err)
	require.Len(t, decision.Errors, 1)
	assert.EqualError(t, decision.Errors[0], "team says no")

	// Each bundle only knows its own policy.
	_, err = platform.Decide(t.Context(), "/teampolicy", testPayload())
	assert.ErrorContains(t, err, "is undefined in the active bundle")
}

func TestGetSourceResolution(t *testing.T) {
	single, stop, err := Setup(t.Context(), discardFactory(t), []config.OpaBundle{
		{Id: "only", ConfigPath: writeConfig(t, "only", "package p")},
	})
	require.NoError(t, err)
	t.Cleanup(stop)

	t.Run("empty source resolves to the only bundle", func(t *testing.T) {
		instance, err := single.Get("")
		require.NoError(t, err)
		assert.Equal(t, "only", instance.ID())
	})

	t.Run("unknown source lists the valid ids", func(t *testing.T) {
		_, err := single.Get("nope")
		assert.ErrorContains(t, err, `unknown bundle_rule.source "nope", configured bundles: only`)
	})

	multi, stop, err := Setup(t.Context(), discardFactory(t), []config.OpaBundle{
		{Id: "a", ConfigPath: writeConfig(t, "a", "package p")},
		{Id: "b", ConfigPath: writeConfig(t, "b", "package p")},
	})
	require.NoError(t, err)
	t.Cleanup(stop)

	t.Run("empty source is ambiguous with several bundles", func(t *testing.T) {
		_, err := multi.Get("")
		assert.ErrorContains(t, err, "bundle_rule.source is required, configured bundles: a, b")
	})

	t.Run("no bundles at all", func(t *testing.T) {
		empty, stop, err := Setup(t.Context(), discardFactory(t), nil)
		require.NoError(t, err)
		t.Cleanup(stop)

		_, err = empty.Get("")
		assert.ErrorContains(t, err, "no opa_bundle is configured")
	})
}

func TestSetupErrors(t *testing.T) {
	t.Run("missing config file", func(t *testing.T) {
		_, _, err := Setup(t.Context(), discardFactory(t), []config.OpaBundle{
			{Id: "missing", ConfigPath: filepath.Join(t.TempDir(), "nope.json")},
		})
		assert.ErrorContains(t, err, `opa_bundle "missing": failed to read config_path`)
	})

	t.Run("unparsable config", func(t *testing.T) {
		_, _, err := Setup(t.Context(), discardFactory(t), []config.OpaBundle{
			{Id: "bad", ConfigPath: writeConfigFile(t, "bad", "... not json ...")},
		})
		assert.ErrorContains(t, err, `opa_bundle "bad": failed to create OPA SDK`)
	})

	t.Run("readiness timeout", func(t *testing.T) {
		path := writeConfigFile(t, "unreachable", `{
			"services": {"test": {"url": "http://127.0.0.1:1"}},
			"bundles": {"test": {"service": "test", "resource": "/bundle.tar.gz"}}
		}`)

		_, _, err := Setup(t.Context(), discardFactory(t), []config.OpaBundle{
			{Id: "unreachable", ConfigPath: path, ReadyTimeout: config.Ptr("1ms")},
		})
		assert.ErrorContains(t, err, `opa_bundle "unreachable" did not become ready in time`)
	})

	t.Run("one failure fails the whole setup", func(t *testing.T) {
		_, _, err := Setup(t.Context(), discardFactory(t), []config.OpaBundle{
			{Id: "good", ConfigPath: writeConfig(t, "good", "package p")},
			{Id: "bad", ConfigPath: filepath.Join(t.TempDir(), "nope.json")},
		})
		assert.ErrorContains(t, err, `opa_bundle "bad"`)
	})
}

// TestSetupLogsBundleFailures is the regression for OPA's default logger being
// a buffered logger that is discarded once plugins start: an instance built
// without an explicit logger swallows every bundle download failure.
func TestSetupLogsBundleFailures(t *testing.T) {
	var buf lockedBuffer
	factory, _ := logutil.NewLoggerFactory(io.Discard, &buf, false)

	path := writeConfigFile(t, "unreachable", `{
		"services": {"test": {"url": "http://127.0.0.1:1"}},
		"bundles": {"test": {"service": "test", "resource": "/bundle.tar.gz"}}
	}`)

	_, _, err := Setup(t.Context(), factory, []config.OpaBundle{
		{Id: "unreachable", ConfigPath: path, ReadyTimeout: config.Ptr("2s")},
	})
	require.Error(t, err)

	assert.Contains(t, buf.String(), "connection refused",
		"the bundle download failure must reach NACP's logger, not OPA's discarded default")
}

func TestReloadKeepsPreviousConfigOnFailure(t *testing.T) {
	path := writeConfig(t, "platform", `package platformpolicy
	errors = ["still here"]`)

	registry, stop, err := Setup(t.Context(), discardFactory(t), []config.OpaBundle{
		{Id: "platform", ConfigPath: path},
	})
	require.NoError(t, err)
	t.Cleanup(stop)

	// A config file that no longer parses must not take the policy down.
	require.NoError(t, os.WriteFile(path, []byte("... not json ..."), 0o600))

	err = registry.Reload(t.Context(), []config.OpaBundle{{Id: "platform", ConfigPath: path}})
	assert.ErrorContains(t, err, `opa_bundle "platform": failed to reconfigure`)

	instance, err := registry.Get("platform")
	require.NoError(t, err)
	decision, err := instance.Decide(t.Context(), "/platformpolicy", testPayload())
	require.NoError(t, err)
	require.Len(t, decision.Errors, 1)
	assert.EqualError(t, decision.Errors[0], "still here")
}

func TestReloadUnknownBundle(t *testing.T) {
	registry, stop, err := Setup(t.Context(), discardFactory(t), []config.OpaBundle{
		{Id: "platform", ConfigPath: writeConfig(t, "platform", "package p")},
	})
	require.NoError(t, err)
	t.Cleanup(stop)

	err = registry.Reload(t.Context(), []config.OpaBundle{{Id: "other", ConfigPath: "/nope"}})
	assert.ErrorContains(t, err, `opa_bundle "other" is not running`)
}

func TestStatusReportsActiveRevision(t *testing.T) {
	registry, stop, err := Setup(t.Context(), discardFactory(t), []config.OpaBundle{
		{Id: "platform", ConfigPath: writeConfig(t, "platform", "package p")},
	})
	require.NoError(t, err)
	t.Cleanup(stop)

	instance, err := registry.Get("platform")
	require.NoError(t, err)

	// The status listener fires asynchronously from bundle activation.
	require.Eventually(t, func() bool {
		for _, status := range instance.Status() {
			if status.Activated() {
				return true
			}
		}
		return false
	}, 5*time.Second, 20*time.Millisecond, "bundle should report a successful activation")
}

func TestDecisionTimeout(t *testing.T) {
	registry, stop, err := Setup(t.Context(), discardFactory(t), []config.OpaBundle{
		{
			Id:              "explicit",
			ConfigPath:      writeConfig(t, "explicit", "package p"),
			DecisionTimeout: config.Ptr("250ms"),
		},
		{
			Id:         "default",
			ConfigPath: writeConfig(t, "default", "package p"),
		},
		{
			Id:              "unbounded",
			ConfigPath:      writeConfig(t, "unbounded", "package p"),
			DecisionTimeout: config.Ptr("0s"),
		},
	})
	require.NoError(t, err)
	t.Cleanup(stop)

	explicit, err := registry.Get("explicit")
	require.NoError(t, err)
	assert.Equal(t, 250*time.Millisecond, explicit.decisionTimeout)

	byDefault, err := registry.Get("default")
	require.NoError(t, err)
	assert.Equal(t, config.DefaultBundleDecisionTimeout, byDefault.decisionTimeout)

	// An explicit zero opts out, leaving decisions bounded only by the request.
	unbounded, err := registry.Get("unbounded")
	require.NoError(t, err)
	assert.Zero(t, unbounded.decisionTimeout)
}

func TestSlogAdapterLevels(t *testing.T) {
	var buf lockedBuffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	adapter := newSlogAdapter(logger)

	adapter.Debug("dropped %s", "debug")
	adapter.Info("kept %s", "info")
	adapter.WithFields(map[string]any{"bundle": "platform"}).Warn("with fields")

	out := buf.String()
	assert.NotContains(t, out, "dropped debug")
	assert.Contains(t, out, "kept info")
	assert.Contains(t, out, "bundle=platform")
}

// lockedBuffer collects log output written from OPA's background goroutines.
type lockedBuffer struct {
	mtx sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.buf.String()
}

func TestVerifySigningConfigured(t *testing.T) {
	tt := []struct {
		name      string
		config    string
		expectErr string
	}{
		{
			name: "signed bundle",
			config: `{
				"keys": {"global_key": {"algorithm": "RS256", "key": "-----BEGIN PUBLIC KEY-----"}},
				"bundles": {"demo": {"service": "s", "signing": {"keyid": "global_key"}}}
			}`,
		},
		{
			name: "yaml form is accepted",
			config: strings.Join([]string{
				"keys:",
				"  global_key:",
				"    algorithm: RS256",
				"bundles:",
				"  demo:",
				"    signing:",
				"      keyid: global_key",
			}, "\n"),
		},
		{
			name:      "no bundles at all",
			config:    `{"services": {"s": {"url": "http://example.com"}}}`,
			expectErr: "declares no bundles",
		},
		{
			name:      "unsigned bundle",
			config:    `{"bundles": {"demo": {"service": "s"}}}`,
			expectErr: `bundle "demo" has no signing block`,
		},
		{
			name:      "signing without a key id",
			config:    `{"bundles": {"demo": {"signing": {"scope": "write"}}}}`,
			expectErr: `bundle "demo" has no signing.keyid`,
		},
		{
			name:      "key id not declared",
			config:    `{"keys": {"other": {}}, "bundles": {"demo": {"signing": {"keyid": "global_key"}}}}`,
			expectErr: `references signing key "global_key" which is not declared in keys`,
		},
		{
			name:      "unparsable config",
			config:    "... not yaml or json ...",
			expectErr: "could not be parsed",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := verifySigningConfigured([]byte(tc.config))
			if tc.expectErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tc.expectErr)
		})
	}
}

func TestSetupRequireSigningRejectsUnsignedBundle(t *testing.T) {
	path := writeConfigFile(t, "unsigned", `{"bundles": {"demo": {"service": "s"}}}`)

	_, _, err := Setup(t.Context(), discardFactory(t), []config.OpaBundle{
		{Id: "platform", ConfigPath: path, RequireSigning: true},
	})
	assert.ErrorContains(t, err, `opa_bundle "platform": require_signing is set but bundle "demo" has no signing block`)
}
