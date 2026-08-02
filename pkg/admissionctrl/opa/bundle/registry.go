// Package bundle owns the OPA SDK instances that evaluate remote policy
// bundles. Each configured opa_bundle block becomes one Instance with its own
// OPA configuration, and therefore its own bundle services, signing keys and
// refresh schedule.
package bundle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mxab/nacp/pkg/config"
	"github.com/mxab/nacp/pkg/logutil"
	"github.com/open-policy-agent/opa/v1/sdk"
)

// Instance is a single OPA SDK instance plus the NACP settings that govern how
// decisions are taken against it.
type Instance struct {
	id              string
	opa             *sdk.OPA
	decisionTimeout time.Duration
	logger          *slog.Logger

	status *statusTracker
}

// ID returns the opa_bundle label this instance was built from.
func (i *Instance) ID() string { return i.id }

// Registry holds every configured bundle instance, keyed by id.
type Registry struct {
	instances map[string]*Instance
	ids       []string
}

// Setup builds every configured bundle instance and waits for each to activate
// its first bundle. It returns a stop function that shuts all of them down. A
// configuration without opa_bundle blocks yields an empty registry, which is a
// valid state: NACP simply has no bundle-backed controllers.
func Setup(ctx context.Context, loggerFactory *logutil.LoggerFactory, bundles []config.OpaBundle) (*Registry, func(), error) {
	registry := &Registry{instances: make(map[string]*Instance, len(bundles))}

	stop := func() {
		for _, instance := range registry.instances {
			instance.stop()
		}
	}

	type result struct {
		instance *Instance
		err      error
	}
	results := make([]result, len(bundles))

	var wg sync.WaitGroup
	for idx, bundleConfig := range bundles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			instance, err := newInstance(ctx, loggerFactory, bundleConfig)
			results[idx] = result{instance: instance, err: err}
		}()
	}
	wg.Wait()

	var errs []error
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		registry.instances[r.instance.id] = r.instance
		registry.ids = append(registry.ids, r.instance.id)
	}
	sort.Strings(registry.ids)

	if len(errs) > 0 {
		// Instances that did come up still hold goroutines and HTTP clients.
		stop()
		return nil, nil, errors.Join(errs...)
	}

	return registry, stop, nil
}

// Get resolves the bundle a controller is bound to. An empty source resolves to
// the only configured bundle, which keeps single-bundle configurations from
// having to repeat the id on every rule.
func (r *Registry) Get(source string) (*Instance, error) {
	if r == nil || len(r.instances) == 0 {
		return nil, errors.New("no opa_bundle is configured")
	}
	if source == "" {
		if len(r.ids) > 1 {
			return nil, fmt.Errorf("bundle_rule.source is required, configured bundles: %s", strings.Join(r.ids, ", "))
		}
		return r.instances[r.ids[0]], nil
	}
	instance, ok := r.instances[source]
	if !ok {
		return nil, fmt.Errorf("unknown bundle_rule.source %q, configured bundles: %s", source, strings.Join(r.ids, ", "))
	}
	return instance, nil
}

// Reload re-reads each bundle's OPA configuration file and reconfigures the
// running instance with it. An instance whose reload fails keeps its previous
// configuration, so a bad edit degrades to "no change" rather than to an
// admission controller with no policy.
func (r *Registry) Reload(ctx context.Context, bundleConfigs []config.OpaBundle) error {
	var errs []error
	for _, bundleConfig := range bundleConfigs {
		instance, ok := r.instances[bundleConfig.Id]
		if !ok {
			errs = append(errs, fmt.Errorf("opa_bundle %q is not running", bundleConfig.Id))
			continue
		}
		if err := instance.reload(ctx, bundleConfig); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (i *Instance) reload(ctx context.Context, bundleConfig config.OpaBundle) error {
	raw, err := os.ReadFile(bundleConfig.ConfigPath)
	if err != nil {
		return fmt.Errorf("opa_bundle %q: failed to read config_path: %w", bundleConfig.Id, err)
	}
	if bundleConfig.RequireSigning {
		if err := verifySigningConfigured(raw); err != nil {
			return fmt.Errorf("opa_bundle %q: %w", bundleConfig.Id, err)
		}
	}

	readyTimeout, err := bundleConfig.ResolvedReadyTimeout()
	if err != nil {
		return fmt.Errorf("opa_bundle %q: %w", bundleConfig.Id, err)
	}
	readyCtx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()

	if err := i.opa.Configure(readyCtx, sdk.ConfigOptions{Config: bytes.NewReader(raw)}); err != nil {
		return fmt.Errorf("opa_bundle %q: failed to reconfigure: %w", bundleConfig.Id, err)
	}
	i.logger.Info("Reconfigured OPA bundle instance", "id", bundleConfig.Id, "config_path", bundleConfig.ConfigPath)
	return nil
}

// Instances returns every configured instance, ordered by id.
func (r *Registry) Instances() []*Instance {
	if r == nil {
		return nil
	}
	instances := make([]*Instance, 0, len(r.ids))
	for _, id := range r.ids {
		instances = append(instances, r.instances[id])
	}
	return instances
}

func newInstance(ctx context.Context, loggerFactory *logutil.LoggerFactory, bundleConfig config.OpaBundle) (*Instance, error) {
	logger := loggerFactory.GetLogger("opa_bundle/" + bundleConfig.Id)

	readyTimeout, err := bundleConfig.ResolvedReadyTimeout()
	if err != nil {
		return nil, fmt.Errorf("opa_bundle %q: %w", bundleConfig.Id, err)
	}
	decisionTimeout, err := bundleConfig.ResolvedDecisionTimeout()
	if err != nil {
		return nil, fmt.Errorf("opa_bundle %q: %w", bundleConfig.Id, err)
	}

	// Read once: the same bytes are inspected for signing requirements and
	// replayed on reload, so the file is never re-read behind OPA's back.
	raw, err := os.ReadFile(bundleConfig.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("opa_bundle %q: failed to read config_path: %w", bundleConfig.Id, err)
	}
	if bundleConfig.RequireSigning {
		if err := verifySigningConfigured(raw); err != nil {
			return nil, fmt.Errorf("opa_bundle %q: %w", bundleConfig.Id, err)
		}
	}

	logger.Info("Starting OPA bundle instance",
		"id", bundleConfig.Id,
		"config_path", bundleConfig.ConfigPath,
		"ready_timeout", readyTimeout,
		"decision_timeout", decisionTimeout,
		"require_signing", bundleConfig.RequireSigning,
	)

	instance := &Instance{
		id:              bundleConfig.Id,
		decisionTimeout: decisionTimeout,
		logger:          logger,
		status:          newStatusTracker(logger),
	}

	ready := make(chan struct{})
	readyCtx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()

	// Without an explicit logger OPA buffers its own logs and then discards
	// them, which hides bundle download and signature failures entirely.
	opaLogger := newSlogAdapter(logger)
	opaSDK, err := sdk.New(ctx, sdk.Options{
		ID:            bundleConfig.Id,
		Config:        bytes.NewReader(raw),
		Ready:         ready,
		Logger:        opaLogger,
		ConsoleLogger: opaLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("opa_bundle %q: failed to create OPA SDK: %w", bundleConfig.Id, err)
	}
	instance.opa = opaSDK
	instance.status.watch(opaSDK)

	logger.Info("Waiting for OPA bundle to become ready", "id", bundleConfig.Id)
	select {
	case <-ready:
		logger.Info("OPA bundle is ready", "id", bundleConfig.Id)
		return instance, nil
	case <-readyCtx.Done():
		instance.stop()
		logger.Error("OPA bundle did not become ready in time", "id", bundleConfig.Id)
		return nil, fmt.Errorf("opa_bundle %q did not become ready in time: %w", bundleConfig.Id, readyCtx.Err())
	}
}

func (i *Instance) stop() {
	if i == nil || i.opa == nil {
		return
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	i.opa.Stop(shutdownCtx)
}
