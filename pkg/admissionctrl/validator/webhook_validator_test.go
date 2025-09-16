package validator

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mxab/nacp/pkg/admissionctrl/types"
	"github.com/mxab/nacp/pkg/config"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WebhookValidator is a validator that uses a webhook to validate a job.

func TestWebhookValidator(t *testing.T) {
	tt := []struct {
		name          string
		endpointPath  string
		method        string
		context       *config.RequestContext
		response      string
		wantErr       error
		wantWarnings  []error
		wantedHeaders map[string]string
	}{
		{
			name:         "empty response",
			endpointPath: "/validate",
			method:       "POST",

			response:     `{}`,
			wantErr:      nil,
			wantWarnings: nil,
		},
		{
			name:         "empty response fields",
			endpointPath: "/validate",

			method:       "POST",
			response:     `{"errors": [], "warnings": []}`,
			wantErr:      nil,
			wantWarnings: nil,
		},
		{
			name:         "errors",
			endpointPath: "/validate",

			method:       "POST",
			response:     `{"errors": ["error1", "error2"], "warnings": []}`,
			wantErr:      multierror.Append(fmt.Errorf("error1"), fmt.Errorf("error2")),
			wantWarnings: nil,
		},
		{
			name:         "warnings",
			endpointPath: "/validate",

			method:       "POST",
			response:     `{"errors": [], "warnings": ["warning1", "warning2"]}`,
			wantErr:      nil,
			wantWarnings: []error{fmt.Errorf("warning1"), fmt.Errorf("warning2")},
		},
		{
			name:         "with AccessorID context",
			endpointPath: "/validate",
			method:       "POST",
			context: &config.RequestContext{
				AccessorID: "accessor-id",
			},
			response:     `{"errors": [], "warnings": []}`,
			wantErr:      nil,
			wantWarnings: nil,
			wantedHeaders: map[string]string{
				"NACP-Accessor-ID": "accessor-id",
			},
		},
		{
			name:         "with ClientIP context",
			endpointPath: "/validate",
			method:       "POST",
			context: &config.RequestContext{
				ClientIP: "client-ip",
			},
			response:     `{"errors": [], "warnings": []}`,
			wantErr:      nil,
			wantWarnings: nil,
			wantedHeaders: map[string]string{
				"NACP-Client-IP":  "client-ip",
				"X-Forwarded-For": "client-ip",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			webhookCalled := false
			//Setup Mock Server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				webhookCalled = true

				var payload types.Payload
				jsonData, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				err = json.Unmarshal(jsonData, &payload)
				require.NoError(t, err)

				expectedJob := &api.Job{ID: &tc.name}
				assert.Equal(t, expectedJob.ID, payload.Job.ID)
				assert.Equal(t, tc.endpointPath, r.URL.Path)
				assert.Equal(t, tc.method, r.Method)
				for key, value := range tc.wantedHeaders {
					assert.Equal(t, value, r.Header.Get(key), "Header %s does not match", key)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tc.response))
			}))
			defer server.Close()

			validator, err := NewWebhookValidator("test", server.URL+tc.endpointPath, tc.method, slog.New(slog.DiscardHandler))
			require.NoError(t, err)

			payload := &types.Payload{Job: &api.Job{ID: &tc.name}, Context: tc.context}
			warnings, err := validator.Validate(t.Context(), payload)

			require.True(t, webhookCalled, "webhook was not called")
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantWarnings, warnings)

		})
	}
}
