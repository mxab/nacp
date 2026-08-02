package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthReportsActivatedBundles(t *testing.T) {
	bundles := testutil.SetupOpaRegistry(t, map[string]string{"platform": "package p"})
	handler := newHealthHandler(bundles, slog.New(slog.DiscardHandler))

	// The bundle status listener fires asynchronously from activation, so the
	// endpoint reports unavailable until OPA has told us about a revision.
	var body healthResponse
	require.Eventually(t, func() bool {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/-/health", nil))
		if rec.Code != http.StatusOK {
			return false
		}
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		return true
	}, 5*time.Second, 20*time.Millisecond, "health should become ok once the bundle activates")

	assert.Equal(t, "ok", body.Status)
	require.Len(t, body.Bundles, 1)
	assert.Equal(t, "platform", body.Bundles[0].Source)
	assert.True(t, body.Bundles[0].Bundle.Activated())
}

func TestHealthWithoutBundlesIsOk(t *testing.T) {
	bundles := testutil.SetupOpaRegistry(t, nil)
	handler := newHealthHandler(bundles, slog.New(slog.DiscardHandler))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/-/health", nil))

	assert.Equal(t, http.StatusOK, rec.Code)

	var body healthResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "ok", body.Status)
	assert.Empty(t, body.Bundles)
}
