package tracing

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Init configures W3C trace context propagation and, when OTLP endpoint env vars are set, an OTLP/HTTP trace exporter.
// Returns a shutdown function that should be called on process exit.
func Init(ctx context.Context) (shutdown func(context.Context) error) {
	noop := func(context.Context) error { return nil }

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if !otlpConfigured() {
		slog.Info("tracing: OTLP disabled (set OTEL_EXPORTER_OTLP_ENDPOINT or OTEL_EXPORTER_OTLP_TRACES_ENDPOINT to enable)")
		return noop
	}

	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		slog.Error("tracing: OTLP exporter failed, continuing without trace export", "err", err)
		return noop
	}

	res, err := resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(attribute.String("service.name", serviceName())),
		resource.WithFromEnv(),
	)
	if err != nil {
		slog.Error("tracing: resource detection failed, continuing without trace export", "err", err)
		return noop
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	slog.Info("tracing: OTLP HTTP exporter enabled")
	return tp.Shutdown
}

func otlpConfigured() bool {
	ep := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"))
	if ep != "" {
		return true
	}
	ep = strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	return ep != ""
}

func serviceName() string {
	if s := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")); s != "" {
		return s
	}
	return "gess-backend"
}
