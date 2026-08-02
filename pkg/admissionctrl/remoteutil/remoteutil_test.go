package remoteutil

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mxab/nacp/pkg/admissionctrl/types"
	"github.com/mxab/nacp/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func TestApplyContextHeders(t *testing.T) {
	type args struct {
		req     *http.Request
		payload *types.Payload
	}
	tests := []struct {
		name          string
		args          args
		wantedHeaders map[string]string
	}{
		{
			name: "with context client ip",
			args: args{
				req: &http.Request{
					Header: http.Header{},
				},
				payload: &types.Payload{
					Context: &config.RequestContext{
						ClientIP: "127.0.0.1",
					},
				},
			},
			wantedHeaders: map[string]string{
				"X-Forwarded-For": "127.0.0.1",
				"NACP-Client-IP":  "127.0.0.1"},
		},
		{
			name: "with context accessor id",
			args: args{
				req: &http.Request{
					Header: http.Header{},
				},
				payload: &types.Payload{
					Context: &config.RequestContext{
						AccessorID: "accessor-id",
					},
				},
			},
			wantedHeaders: map[string]string{
				"NACP-Accessor-ID": "accessor-id",
			},
		},
		{
			name: "without context",
			args: args{
				req: &http.Request{
					Header: http.Header{},
				},
				payload: &types.Payload{
					Context: nil,
				},
			},
			wantedHeaders: map[string]string{},
		},
		{
			name: "with empty context",
			args: args{
				req: &http.Request{
					Header: http.Header{},
				},
				payload: &types.Payload{
					Context: &config.RequestContext{},
				},
			},
			wantedHeaders: map[string]string{},
		},

		{
			name: "without context",
			args: args{
				req: &http.Request{
					Header: http.Header{},
				},
				payload: &types.Payload{
					Context: nil,
				},
			},
			wantedHeaders: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ApplyContextHeaders(tt.args.req, tt.args.payload)
			for header, expectedValue := range tt.wantedHeaders {
				assert.Equal(t, expectedValue, tt.args.req.Header.Get(header), "header %s", header)
			}
		})
	}
}

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantHost string
		wantErr  string
	}{
		{
			name:     "http endpoint",
			endpoint: "http://localhost:8080/validate",
			wantHost: "localhost:8080",
		},
		{
			name:     "https endpoint",
			endpoint: "https://example.com/validate",
			wantHost: "example.com",
		},
		{
			name:     "relative url",
			endpoint: "/validate",
			wantErr:  "webhook endpoint must be an absolute HTTP(S) URL",
		},
		{
			name:     "non http scheme",
			endpoint: "ftp://example.com/validate",
			wantErr:  "webhook endpoint must be an absolute HTTP(S) URL",
		},
		{
			name:     "unparsable url",
			endpoint: "http://exa mple.com",
			wantErr:  "invalid character",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := ParseEndpoint(tt.endpoint)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, u)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantHost, u.Host)
		})
	}
}

func TestDecodeJSONResponse(t *testing.T) {
	type target struct {
		Name string `json:"name"`
	}

	// One byte past the limit, so the size check trips on the smallest possible body.
	oversized := strings.Repeat("a", MaxResponseBodyBytes+1)

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantName   string
		wantErr    string
	}{
		{
			name:       "decodes the response",
			statusCode: http.StatusOK,
			body:       `{"name":"my-job"}`,
			wantName:   "my-job",
		},
		{
			name:       "response too large",
			statusCode: http.StatusOK,
			body:       oversized,
			wantErr:    "webhook response exceeds",
		},
		{
			name:       "unexpected status",
			statusCode: http.StatusInternalServerError,
			body:       `{"name":"my-job"}`,
			wantErr:    "webhook returned unexpected HTTP status",
		},
		{
			name:       "malformed json",
			statusCode: http.StatusOK,
			body:       `{"name":`,
			wantErr:    "failed to decode webhook response",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Status:     http.StatusText(tt.statusCode),
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}
			var got target
			err := DecodeJSONResponse(resp, &got)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, got.Name)
		})
	}
}

func TestDecodeJSONResponseReadError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
		Body:       io.NopCloser(errReader{}),
	}
	err := DecodeJSONResponse(resp, &struct{}{})
	assert.ErrorContains(t, err, "failed to read webhook response")
}

// errReader fails on the first read so the body-read error path is exercised.
type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestNewInstrumentedClient(t *testing.T) {
	client := NewInstrumentedClient()
	require.NotNil(t, client)
	assert.IsType(t, &otelhttp.Transport{}, client.Transport)
}
