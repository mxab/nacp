package remoteutil

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mxab/nacp/pkg/admissionctrl/types"
	"github.com/mxab/nacp/pkg/config"
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
				if tt.args.req.Header.Get(header) != expectedValue {
					t.Errorf("Expected header %s to be %s, got %s", header, expectedValue, tt.args.req.Header.Get(header))
				}
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
				if err == nil {
					t.Fatalf("Expected an error containing %q, got none", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				if u != nil {
					t.Errorf("Expected a nil url on error, got %v", u)
				}
				return
			}
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if u.Host != tt.wantHost {
				t.Errorf("Expected host %s, got %s", tt.wantHost, u.Host)
			}
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
				if err == nil {
					t.Fatalf("Expected an error containing %q, got none", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if got.Name != tt.wantName {
				t.Errorf("Expected name %s, got %s", tt.wantName, got.Name)
			}
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
	if err == nil {
		t.Fatal("Expected an error, got none")
	}
	if !strings.Contains(err.Error(), "failed to read webhook response") {
		t.Errorf("Expected a read failure, got %q", err.Error())
	}
}

// errReader fails on the first read so the body-read error path is exercised.
type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestNewInstrumentedClient(t *testing.T) {
	client := NewInstrumentedClient()
	if client == nil {
		t.Fatal("Expected NewInstrumentedClient to return a non-nil client")
	}

	if _, ok := client.Transport.(*otelhttp.Transport); !ok {
		t.Fatal("Expected the client's transport to be of type InstrumentedTransport")
	}
}
