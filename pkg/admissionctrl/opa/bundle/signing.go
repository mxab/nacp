package bundle

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// signingView is the minimal slice of an OPA configuration NACP needs in order
// to tell whether bundle signature verification is switched on. Everything else
// in the file, including the parts NACP knows nothing about, is handed to OPA
// verbatim — this only reads, it never rewrites.
type signingView struct {
	Keys    map[string]struct{} `yaml:"keys"`
	Bundles map[string]struct {
		Signing *struct {
			KeyID  string `yaml:"keyid"`
			Scope  string `yaml:"scope"`
			KeyIDs string `yaml:"keyids"`
		} `yaml:"signing"`
	} `yaml:"bundles"`
}

// verifySigningConfigured refuses a configuration whose bundles are downloaded
// without signature verification. Bundle policy runs with full authority over
// every job that passes through the proxy, so an unsigned bundle means whoever
// can answer the bundle URL can rewrite admission control.
//
// JSON is a subset of YAML, so this parses both forms OPA accepts.
func verifySigningConfigured(raw []byte) error {
	var view signingView
	if err := yaml.Unmarshal(raw, &view); err != nil {
		return fmt.Errorf("require_signing is set but the OPA configuration could not be parsed: %w", err)
	}

	if len(view.Bundles) == 0 {
		return fmt.Errorf("require_signing is set but the OPA configuration declares no bundles")
	}

	names := make([]string, 0, len(view.Bundles))
	for name := range view.Bundles {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		signing := view.Bundles[name].Signing
		if signing == nil {
			return fmt.Errorf("require_signing is set but bundle %q has no signing block", name)
		}
		keyID := signing.KeyID
		if keyID == "" {
			keyID = signing.KeyIDs
		}
		if keyID == "" {
			return fmt.Errorf("require_signing is set but bundle %q has no signing.keyid", name)
		}
		if _, ok := view.Keys[keyID]; !ok {
			return fmt.Errorf("require_signing is set but bundle %q references signing key %q which is not declared in keys", name, keyID)
		}
	}

	return nil
}
