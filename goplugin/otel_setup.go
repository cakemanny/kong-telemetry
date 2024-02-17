package main

import (
	"context"
	"errors"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var otelConfig struct {
	ExporterOTLPEndpoint string            `json:"otel_exporter_otlp_endpoint"`
	ExporterOTLPHeaders  map[string]string `json:"otel_exporter_otlp_headers"`
	Environment          string            `json:"deployment_environment"`
}

func init() {
	// TODO: load from a config file?
	otelConfig.ExporterOTLPEndpoint = "http://apm-server:8200"
	otelConfig.ExporterOTLPHeaders = map[string]string{
		"Authorization": "Bearer db6496b7e310a16f798a6f990f1a48be636755289d67ecbefccf9a634387e4e5",
	}
	otelConfig.Environment = "production"
}

// setupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func setupOTelSDK(ctx context.Context) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

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

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider.
	tracerProvider, err := newTraceProvider(ctx)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider.
	//meterProvider, err := newMeterProvider(ctx)
	//if err != nil {
	//	handleErr(err)
	//	return
	//}
	//shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	//otel.SetMeterProvider(meterProvider)

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTraceProvider(ctx context.Context) (*trace.TracerProvider, error) {
	var traceExporter trace.SpanExporter
	var err error
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		traceExporter, err = otlptracehttp.New(ctx)
	} else {
		traceExporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithEndpointURL(otelConfig.ExporterOTLPEndpoint),
			otlptracehttp.WithHeaders(otelConfig.ExporterOTLPHeaders),
		)
	}
	if err != nil {
		return nil, err
	}

	// Kong hides env vars from plugins, so we have to configure in code
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(pluginName),
		semconv.ServiceVersion(pluginVersion),
		semconv.DeploymentEnvironment(otelConfig.Environment),
	)
	res, err = resource.Merge(resource.Default(), res)
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			trace.WithBatchTimeout(5*time.Second)),
		trace.WithResource(res),
	)
	return traceProvider, nil
}

func newMeterProvider(ctx context.Context) (*metric.MeterProvider, error) {
	var metricExporter metric.Exporter
	var err error
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		metricExporter, err = otlpmetrichttp.New(ctx)
	} else {
		metricExporter, err = otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpointURL(otelConfig.ExporterOTLPEndpoint),
			otlpmetrichttp.WithHeaders(otelConfig.ExporterOTLPHeaders),
		)
	}
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			metric.WithInterval(1*time.Minute))),
	)
	return meterProvider, nil
}
