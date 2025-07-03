package remoteutil

import (
	"context"
	"net/http"
	"net/http/httptrace"

	"github.com/mxab/nacp/admissionctrl/types"
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

func NewInstrumentedClient() *http.Client {
	return &http.Client{
		Transport: InstrumentedTransport(http.DefaultTransport.(*http.Transport)),
	}
}
