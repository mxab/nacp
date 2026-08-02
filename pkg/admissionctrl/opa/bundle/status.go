package bundle

import (
	"log/slog"
	"sync"
	"time"

	"github.com/open-policy-agent/opa/v1/plugins/bundle"
	"github.com/open-policy-agent/opa/v1/sdk"
)

// BundleStatus is the part of OPA's bundle status NACP reports on.
type BundleStatus struct {
	Name                     string    `json:"name"`
	ActiveRevision           string    `json:"active_revision,omitempty"`
	LastSuccessfulActivation time.Time `json:"last_successful_activation,omitzero"`
	Error                    string    `json:"error,omitempty"`
}

// Activated reports whether this bundle has ever been successfully activated.
// A bundle that has activated once keeps serving its last good policy even if
// the bundle server later disappears, so this is a liveness signal for startup
// rather than for staleness; staleness surfaces through Error.
func (s BundleStatus) Activated() bool {
	return !s.LastSuccessfulActivation.IsZero()
}

// statusTracker caches the latest status OPA reports for each bundle and warns
// whenever a refresh fails. Without it a bundle server that dies leaves NACP
// silently enforcing whatever policy it last managed to download.
type statusTracker struct {
	logger *slog.Logger

	mtx      sync.RWMutex
	statuses map[string]BundleStatus
}

func newStatusTracker(logger *slog.Logger) *statusTracker {
	return &statusTracker{logger: logger, statuses: map[string]BundleStatus{}}
}

func (t *statusTracker) watch(opaSDK *sdk.OPA) {
	plugin, ok := opaSDK.Plugin(bundle.Name).(*bundle.Plugin)
	if !ok || plugin == nil {
		// No bundles configured for this instance (for example a purely
		// inline-policy config); nothing to track.
		return
	}
	plugin.RegisterBulkListener("nacp", t.update)
}

func (t *statusTracker) update(statuses map[string]*bundle.Status) {
	snapshot := make(map[string]BundleStatus, len(statuses))
	for name, status := range statuses {
		if status == nil {
			continue
		}
		entry := BundleStatus{
			Name:                     name,
			ActiveRevision:           status.ActiveRevision,
			LastSuccessfulActivation: status.LastSuccessfulActivation,
		}
		if status.Message != "" {
			entry.Error = status.Message
		}
		if entry.Error != "" || status.Code != "" {
			t.logger.Warn("OPA bundle refresh failed, continuing with the last activated policy",
				"bundle", name,
				"code", status.Code,
				"message", status.Message,
				"active_revision", status.ActiveRevision,
				"last_successful_activation", status.LastSuccessfulActivation,
			)
		}
		snapshot[name] = entry
	}

	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.statuses = snapshot
}

func (t *statusTracker) snapshot() []BundleStatus {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	out := make([]BundleStatus, 0, len(t.statuses))
	for _, status := range t.statuses {
		out = append(out, status)
	}
	return out
}

// Status returns the latest known status of every bundle in this instance.
func (i *Instance) Status() []BundleStatus {
	if i == nil || i.status == nil {
		return nil
	}
	return i.status.snapshot()
}
