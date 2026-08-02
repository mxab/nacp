package main

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/mxab/nacp/pkg/admissionctrl/opa/bundle"
)

type healthResponse struct {
	Status  string              `json:"status"`
	Bundles []healthBundleEntry `json:"bundles,omitempty"`
}

type healthBundleEntry struct {
	Source string              `json:"source"`
	Bundle bundle.BundleStatus `json:"bundle"`
}

// newHealthHandler reports whether every configured bundle has activated a
// policy. NACP keeps serving the last activated bundle when refreshes start
// failing, so this is what makes that state visible to an operator or an
// orchestrator health check instead of silently enforcing stale policy.
func newHealthHandler(bundles *bundle.Registry, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := healthResponse{Status: "ok"}

		for _, instance := range bundles.Instances() {
			statuses := instance.Status()
			if len(statuses) == 0 {
				// The instance is up but OPA has not reported on any bundle yet.
				response.Status = "unavailable"
				continue
			}
			for _, status := range statuses {
				response.Bundles = append(response.Bundles, healthBundleEntry{
					Source: instance.ID(),
					Bundle: status,
				})
				if !status.Activated() {
					response.Status = "unavailable"
				}
			}
		}

		code := http.StatusOK
		if response.Status != "ok" {
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.WarnContext(r.Context(), "Writing health response failed", "error", err)
		}
	})
}
