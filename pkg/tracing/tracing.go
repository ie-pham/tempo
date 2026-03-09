// Package tracing provides OpenTelemetry tracing setup for the application.
package tracing

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/prometheus/common/version"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

func InstallOpenTelemetryTracer(appName, target string) (func(), error) {
	level.Info(log.Logger).Log("msg", "initialising OpenTelemetry tracer")

	exp, err := autoexport.NewSpanExporter(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create OTEL exporter: %w", err)
	}

	// service.version is read explicitly from OTEL_RESOURCE_ATTRIBUTES so that
	// the test harness can inject a commit identifier (e.g. a synthetic PR tag)
	// without rebuilding the binary.  Falls back to the binary's embedded
	// version/revision when the env var is absent or does not contain service.version.
	//
	// Note: resource.New() gives earlier options higher merge priority, so
	// resource.WithFromEnv() must not be relied on to override resource.WithAttributes().
	serviceVersion := serviceVersionFromEnv()
	if serviceVersion == "" {
		serviceVersion = fmt.Sprintf("%s-%s", version.Version, version.Revision)
	}

	resources, err := resource.New(context.Background(),
		resource.WithFromEnv(), // other OTEL_RESOURCE_ATTRIBUTES pass through
		resource.WithHost(),
		resource.WithTelemetrySDK(),
		// WithAttributes is last so service.name is always the component-specific
		// name and service.version is the explicitly resolved value above.
		resource.WithAttributes(
			semconv.ServiceNameKey.String(fmt.Sprintf("%s-%s", appName, target)),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise trace resources: %w", err)
	}

	otel.SetErrorHandler(otelErrorHandlerFunc(func(err error) {
		level.Error(log.Logger).Log("msg", "OpenTelemetry.ErrorHandler", "err", err)
	}))

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resources),
	)
	otel.SetTracerProvider(tp)

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			level.Error(log.Logger).Log("msg", "OpenTelemetry trace provider failed to shutdown", "err", err)
			os.Exit(1)
		}
	}

	propagator := propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})
	otel.SetTextMapPropagator(propagator)

	return shutdown, nil
}

// serviceVersionFromEnv parses OTEL_RESOURCE_ATTRIBUTES and returns the value
// of the "service.version" key, or an empty string if it is not present.
func serviceVersionFromEnv() string {
	for _, kv := range strings.Split(os.Getenv("OTEL_RESOURCE_ATTRIBUTES"), ",") {
		kv = strings.TrimSpace(kv)
		if after, ok := strings.CutPrefix(kv, "service.version="); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

type otelErrorHandlerFunc func(error)

// Handle implements otel.ErrorHandler
func (f otelErrorHandlerFunc) Handle(err error) {
	f(err)
}
