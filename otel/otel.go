package otel

import (
	"bytes"
	"context"
	"errors"
	"io"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	logApi "go.opentelemetry.io/otel/log"
	metricApi "go.opentelemetry.io/otel/metric"
	traceApi "go.opentelemetry.io/otel/trace"

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

func SetupOTelSDK(ctx context.Context, logging, metrics, tracing bool) (shutdown func(context.Context) error, err error) {

	shutdown, handleErr, shutdownFnAppender := cleanupConfig(ctx)

	var tp traceApi.TracerProvider

	if tracing {

		tracerExporter, err := otlptracehttp.New(ctx)

		if err != nil {
			err = handleErr(err)
			return nil, err
		}
		tracerProvider := newTracerProvider(tracerExporter)
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

		meterProvider := newMeterProvider(metricExporter) // Updated to remove error handling

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

		loggerProvider := newLoggerProvider(loggerExporter)

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
