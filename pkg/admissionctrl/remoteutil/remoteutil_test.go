package remoteutil

import (
	"net/http"
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

func TestNewInstrumentedClient(t *testing.T) {
	client := NewInstrumentedClient()
	if client == nil {
		t.Fatal("Expected NewInstrumentedClient to return a non-nil client")
	}

	if _, ok := client.Transport.(*otelhttp.Transport); !ok {
		t.Fatal("Expected the client's transport to be of type InstrumentedTransport")
	}
}
