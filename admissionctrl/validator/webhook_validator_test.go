package validator

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WebhookValidator is a validator that uses a webhook to validate a job.

func TestWebhookValidator(t *testing.T) {
	//calls and endpoint with a job, returns a json response with result.errors and result.warnings fields
	//if result.errors is not empty, return an error
	//if result.warnings is not empty, return a all warnings

	//Setup Table test, with different responses
	tt := []struct {
		name         string
		endpointPath string
		method       string

		response     string
		wantErr      error
		wantWarnings []error
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
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			webhookCalled := false
			//Setup Mock Server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				webhookCalled = true

				expectedJobJsonData, err := json.Marshal(&api.Job{ID: &tc.name})
				require.NoError(t, err)

				assert.Equal(t, tc.endpointPath, r.URL.Path)
				assert.Equal(t, tc.method, r.Method)
				data, err := io.ReadAll(r.Body)

				require.NoError(t, err)
				assert.JSONEq(t, string(expectedJobJsonData), string(data))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tc.response))

			}))
			defer server.Close()
			//Test
			endpoint, err := url.Parse(server.URL + tc.endpointPath)
			require.NoError(t, err)
			validator := WebhookValidator{
				endpoint: endpoint,
				name:     "test",
				method:   tc.method,
			}
			warnings, err := validator.Validate(&api.Job{ID: &tc.name})

			require.True(t, webhookCalled, "webhook was not called")
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantWarnings, warnings)
		})
	}

}
