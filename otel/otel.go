package otel

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	logApi "go.opentelemetry.io/otel/log"
	metricApi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	traceApi "go.opentelemetry.io/otel/trace"

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

func SetupOTelSDKWith(ctx context.Context, loggerProvider logApi.LoggerProvider, metricReader metric.Reader, traceExporter trace.SpanExporter) (shutdown func(context.Context) error, flush func(context.Context) error, err error) {

	shutdown, _, shutdownFnAppender := cleanupConfig(ctx)

	tracerProvider := newTracerProvider(traceExporter, nil)

	shutdownFnAppender(tracerProvider.Shutdown)

	meterProvider := newMeterProvider(metricReader, nil)

	shutdownFnAppender(meterProvider.Shutdown)
	shutdownFnAppender(traceExporter.Shutdown)

	// Set up propagator.
	ApplyProviders(tracerProvider, meterProvider, loggerProvider)

	forceFlushes := make([]func(context.Context) error, 0)
	forceFlushes = append(forceFlushes, tracerProvider.ForceFlush)
	forceFlushes = append(forceFlushes, meterProvider.ForceFlush)

	flush = func(ctx context.Context) error {
		var err error
		for _, fn := range forceFlushes {
			err = errors.Join(err, fn(ctx))
		}
		return err
	}
	return shutdown, flush, nil
}

func SetupOTelSDK(ctx context.Context, logging, metrics, tracing bool, versionKey string) (shutdown func(context.Context) error, err error) {

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("nacp"),
		semconv.ServiceVersionKey.String(versionKey),
	)
	shutdown, handleErr, shutdownFnAppender := cleanupConfig(ctx)

	var tp traceApi.TracerProvider

	if tracing {

		tracerExporter, err := otlptracehttp.New(ctx)

		if err != nil {
			err = handleErr(err)
			return nil, err
		}
		tracerProvider := newTracerProvider(tracerExporter, res)
		shutdownFnAppender(tracerProvider.Shutdown)
		tp = tracerProvider
	}

	var mp metricApi.MeterProvider
	if metrics {
		metricExporter, err := otlpmetrichttp.New(ctx)
		if err != nil {
			err = handleErr(err)
			return nil, err
		}

		meterProvider := newMeterProvider(metric.NewPeriodicReader(metricExporter), res) // Updated to remove error handling

		shutdownFnAppender(meterProvider.Shutdown)
		mp = meterProvider
	}

	var lp logApi.LoggerProvider

	if logging {
		loggerExporter, err := otlploghttp.New(ctx)
		if err != nil {
			err = handleErr(err)
			return nil, err
		}

		loggerProvider := newLoggerProvider(loggerExporter, res)

		shutdownFnAppender(loggerProvider.Shutdown)
		lp = loggerProvider
	}
	// Set up propagator.
	ApplyProviders(tp, mp, lp)

	return shutdown, nil
}

func ApplyProviders(tracerProvider traceApi.TracerProvider, meterProvider metricApi.MeterProvider, loggerProvider logApi.LoggerProvider) {
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

func newTracerProvider(exporter trace.SpanExporter, res *resource.Resource) *trace.TracerProvider {

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)
	return tracerProvider
}

func newMeterProvider(reader metric.Reader, res *resource.Resource) *metric.MeterProvider {

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(res),
	)

	return meterProvider
}

func newLoggerProvider(exporter log.Exporter, res *resource.Resource) *log.LoggerProvider {

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter)),
		log.WithResource(res),
	)
	return loggerProvider
}
