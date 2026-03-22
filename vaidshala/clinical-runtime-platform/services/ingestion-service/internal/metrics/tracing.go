package metrics

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// IngestionServiceName is the OpenTelemetry service name.
	IngestionServiceName = "ingestion-service"
	// IngestionServiceVersion is the current service version.
	IngestionServiceVersion = "0.5.0"
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
			semconv.ServiceName(IngestionServiceName),
			semconv.ServiceVersion(IngestionServiceVersion),
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
func TracingMiddleware(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(serviceName)
	return func(c *gin.Context) {
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		span.SetAttributes(
			semconv.HTTPMethod(c.Request.Method),
			semconv.HTTPTarget(c.Request.URL.Path),
			attribute.String("http.route", c.FullPath()),
		)

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		span.SetAttributes(semconv.HTTPStatusCode(c.Writer.Status()))
	}
}

// SpanFromContext creates a new child span from the given context.
func SpanFromContext(ctx context.Context, serviceName, operationName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer(serviceName)
	ctx, span := tracer.Start(ctx, operationName)
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	return ctx, span
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
