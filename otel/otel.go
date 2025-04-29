package otel

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/mxab/nacp/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

func cleanupConfig(ctx context.Context) (
	shutdown func(context.Context) error,
	handleErr func(inErr error) error,
	shutdownFnAppender func(fn func(context.Context) error)) {

	var shutdownFuncs []func(context.Context) error

	shutdownFnAppender = func(fn func(context.Context) error) {
		shutdownFuncs = append(shutdownFuncs, fn)
	}
	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {

		var err error

		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}
	handleErr = func(inErr error) error {
		return errors.Join(inErr, shutdown(ctx))
	}
	return
}

func SetupOTelSDKWithInmemoryOutput(ctx context.Context) (lr, mr, tr io.Reader, shutdown func(context.Context) error, err error) {

	logOut := bytes.NewBufferString("")
	metricsOut := bytes.NewBufferString("")
	traceOut := bytes.NewBufferString("")
	shutdown, handleErr, shutdownFnAppender := cleanupConfig(ctx)

	logExporter, err := stdoutlog.New(stdoutlog.WithWriter(logOut))
	if err != nil {
		handleErr(err)
		return
	}
	metricExporter, err := stdoutmetric.New(stdoutmetric.WithWriter(metricsOut))
	if err != nil {
		handleErr(err)
		return
	}
	traceExporter, err := stdouttrace.New(stdouttrace.WithWriter(traceOut))
	if err != nil {
		handleErr(err)
		return
	}

	tracerProvider := newTracerProvider(traceExporter)

	shutdownFnAppender(tracerProvider.Shutdown)

	meterProvider := newMeterProvider(metricExporter) // Updated to remove error handling

	shutdownFnAppender(meterProvider.Shutdown)

	loggerProvider := newLoggerProvider(logExporter)

	shutdownFnAppender(loggerProvider.Shutdown)

	// Set up propagator.
	ApplyProviders(tracerProvider, meterProvider, loggerProvider)

	return logOut, metricsOut, traceOut, shutdown, nil
}
func SetupOTelSDK(ctx context.Context, otelConfig config.OtelConfig) (shutdown func(context.Context) error, err error) {

	shutdown, handleErr, shutdownFnAppender, forceFlushFnAppender := cleanupConfig(ctx)

	tracerExporter, err := traceExporter(ctx, otelConfig.Tracing)
	if err != nil {
		handleErr(err)
		return
	}
	metricExporter, err := metricExporter(ctx, otelConfig.Metrics)
	if err != nil {
		handleErr(err)
		return
	}
	loggerExporter, err := logExporter(ctx, otelConfig.Logging)
	if err != nil {
		handleErr(err)
		return
	}

	// Set up trace provider.
	tracerProvider := newTracerProvider(tracerExporter)

	shutdownFnAppender(tracerProvider.Shutdown)
	forceFlushFnAppender(tracerProvider.ForceFlush)

	meterProvider := newMeterProvider(metricExporter) // Updated to remove error handling

	shutdownFnAppender(meterProvider.Shutdown)
	forceFlushFnAppender(meterProvider.ForceFlush)

	loggerProvider := newLoggerProvider(loggerExporter)

	shutdownFnAppender(loggerProvider.Shutdown)
	forceFlushFnAppender(loggerProvider.ForceFlush)

	// Set up propagator.
	ApplyProviders(tracerProvider, meterProvider, loggerProvider)

	return shutdown, nil
}

func ApplyProviders(tracerProvider *trace.TracerProvider, meterProvider *metric.MeterProvider, loggerProvider *log.LoggerProvider) {
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	if tracerProvider != nil {
		otel.SetTracerProvider(tracerProvider)
	}
	if meterProvider != nil {
		otel.SetMeterProvider(meterProvider)
	}
	if loggerProvider != nil {
		global.SetLoggerProvider(loggerProvider)
	}

}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTracerProvider(exporter trace.SpanExporter) *trace.TracerProvider {

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
	)
	return tracerProvider
}

func newMeterProvider(exporter metric.Exporter) *metric.MeterProvider {

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
	)

	return meterProvider
}

func newLoggerProvider(exporter log.Exporter) *log.LoggerProvider {

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter)),
	)
	return loggerProvider
}

func logExporter(ctx context.Context, cfg *config.LoggingConfig) (log.Exporter, error) {
	exporterType := cfg.Exporter
	switch exporterType {
	case "stdout":
		return stdoutlog.New()
	case "otlp":
		return otlploghttp.New(ctx)
	default:
		return nil, errors.New("unknown exporter type")
	}
}
func metricExporter(ctx context.Context, cfg *config.MetricsConfig) (metric.Exporter, error) {
	exporterType := cfg.Exporter
	switch exporterType {
	case "stdout":
		return stdoutmetric.New()
	case "otlp":
		return otlpmetrichttp.New(ctx)
	default:
		return nil, errors.New("unknown exporter type")
	}
}
func traceExporter(ctx context.Context, cfg *config.TracingConfig) (trace.SpanExporter, error) {
	exporterType := cfg.Exporter
	switch exporterType {
	case "stdout":
		return stdouttrace.New()
	case "otlp":
		return otlptracehttp.New(ctx)
	default:
		return nil, errors.New("unknown exporter type")
	}
}
