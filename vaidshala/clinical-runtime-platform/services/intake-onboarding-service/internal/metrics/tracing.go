package metrics

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// IntakeServiceName is the OpenTelemetry service name.
	IntakeServiceName = "intake-onboarding-service"
	// IntakeServiceVersion is the current service version.
	IntakeServiceVersion = "0.5.0"
)

// InitTracer initialises an OTLP HTTP trace exporter and returns the TracerProvider.
// The caller must call TracerProvider.Shutdown on graceful exit.
func InitTracer(ctx context.Context, otlpEndpoint string) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otlpEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(IntakeServiceName),
			semconv.ServiceVersion(IntakeServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.10))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// TracingMiddleware returns a Gin middleware that extracts the incoming trace
// context and creates a server span for each request.
func TracingMiddleware() gin.HandlerFunc {
	tracer := otel.Tracer(IntakeServiceName)
	return func(c *gin.Context) {
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		span.SetAttributes(
			semconv.HTTPMethod(c.Request.Method),
			semconv.HTTPTarget(c.Request.URL.Path),
		)

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		span.SetAttributes(semconv.HTTPStatusCode(c.Writer.Status()))
	}
}

// TraceIDFromContext extracts the trace ID from the current span context.
// Returns an empty string if no active span exists.
func TraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.HasTraceID() {
		return sc.TraceID().String()
	}
	return ""
}
