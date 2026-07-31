package remoteutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

	"github.com/mxab/nacp/pkg/admissionctrl/types"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func ApplyContextHeaders(req *http.Request, payload *types.Payload) {
	if payload.Context != nil {
		// Add standard headers for backward compatibility
		if payload.Context.ClientIP != "" {
			req.Header.Set("X-Forwarded-For", payload.Context.ClientIP) // Standard proxy header
			req.Header.Set("NACP-Client-IP", payload.Context.ClientIP)  // NACP specific
		}
		if payload.Context.AccessorID != "" {
			req.Header.Set("NACP-Accessor-ID", payload.Context.AccessorID)
		}
	}
}

// https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/net/http/httptrace/otelhttptrace/example/client/client.go

func InstrumentedTransport(transport http.RoundTripper) *otelhttp.Transport {
	return otelhttp.NewTransport(
		transport,
		// By setting the otelhttptrace client in this transport, it can be
		// injected into the context after the span is started, which makes the
		// httptrace spans children of the transport one.
		otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
			return otelhttptrace.NewClientTrace(ctx)
		}),
	)
}

const (
	DefaultRequestTimeout = 30 * time.Second
	MaxResponseBodyBytes  = 10 << 20
)

var defaultInstrumentedTransport = InstrumentedTransport(http.DefaultTransport)

func NewInstrumentedClient() *http.Client {
	return &http.Client{
		Transport: defaultInstrumentedTransport,
		Timeout:   DefaultRequestTimeout,
	}
}

func ParseEndpoint(endpoint string) (*url.URL, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("webhook endpoint must be an absolute HTTP(S) URL: %q", endpoint)
	}
	return u, nil
}

func DecodeJSONResponse(resp *http.Response, target interface{}) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBodyBytes+1))
	if err != nil {
		return fmt.Errorf("failed to read webhook response: %w", err)
	}
	if len(body) > MaxResponseBodyBytes {
		return fmt.Errorf("webhook response exceeds %d bytes", MaxResponseBodyBytes)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("webhook returned unexpected HTTP status %s", resp.Status)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("failed to decode webhook response: %w", err)
	}
	return nil
}
