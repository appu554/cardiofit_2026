package monitoring

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TracingProvider wraps OpenTelemetry tracing functionality
type TracingProvider struct {
	tracer   oteltrace.Tracer
	provider *trace.TracerProvider
	logger   *zap.Logger
}

// NewTracingProvider creates a new tracing provider
func NewTracingProvider(serviceName, jaegerEndpoint string, logger *zap.Logger) (*TracingProvider, error) {
	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("service.namespace", "clinical-workflow"),
			attribute.String("service.instance.id", "workflow-engine-go"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	provider := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()), // In production, use a more sophisticated sampler
	)

	// Set global tracer provider
	otel.SetTracerProvider(provider)

	// Create tracer
	tracer := provider.Tracer(serviceName)

	return &TracingProvider{
		tracer:   tracer,
		provider: provider,
		logger:   logger,
	}, nil
}

// Close shuts down the tracing provider
func (t *TracingProvider) Close(ctx context.Context) error {
	return t.provider.Shutdown(ctx)
}

// StartWorkflowSpan starts a new span for workflow orchestration
func (t *TracingProvider) StartWorkflowSpan(ctx context.Context, workflowType, correlationID, patientID string) (context.Context, oteltrace.Span) {
	ctx, span := t.tracer.Start(ctx, "workflow.orchestrate",
		oteltrace.WithAttributes(
			attribute.String("workflow.type", workflowType),
			attribute.String("workflow.correlation_id", correlationID),
			attribute.String("patient.id", patientID),
			attribute.String("component", "strategic_orchestrator"),
		),
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
	)

	return ctx, span
}

// StartPhaseSpan starts a new span for a workflow phase
func (t *TracingProvider) StartPhaseSpan(ctx context.Context, phase string) (context.Context, oteltrace.Span) {
	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("workflow.phase.%s", phase),
		oteltrace.WithAttributes(
			attribute.String("workflow.phase", phase),
			attribute.String("component", "strategic_orchestrator"),
		),
	)

	return ctx, span
}

// StartExternalServiceSpan starts a new span for external service calls
func (t *TracingProvider) StartExternalServiceSpan(ctx context.Context, serviceName, operation string) (context.Context, oteltrace.Span) {
	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("external.%s.%s", serviceName, operation),
		oteltrace.WithAttributes(
			attribute.String("external.service", serviceName),
			attribute.String("external.operation", operation),
			attribute.String("component", "external_client"),
		),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
	)

	return ctx, span
}

// StartDatabaseSpan starts a new span for database operations
func (t *TracingProvider) StartDatabaseSpan(ctx context.Context, operation, table string) (context.Context, oteltrace.Span) {
	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("db.%s.%s", operation, table),
		oteltrace.WithAttributes(
			attribute.String("db.operation", operation),
			attribute.String("db.table", table),
			attribute.String("db.system", "postgresql"),
			attribute.String("component", "database"),
		),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
	)

	return ctx, span
}

// StartHTTPSpan starts a new span for HTTP requests
func (t *TracingProvider) StartHTTPSpan(ctx context.Context, method, path string) (context.Context, oteltrace.Span) {
	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("http.%s %s", method, path),
		oteltrace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.route", path),
			attribute.String("component", "http_handler"),
		),
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
	)

	return ctx, span
}

// StartGraphQLSpan starts a new span for GraphQL operations
func (t *TracingProvider) StartGraphQLSpan(ctx context.Context, operationType, operationName string) (context.Context, oteltrace.Span) {
	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("graphql.%s.%s", operationType, operationName),
		oteltrace.WithAttributes(
			attribute.String("graphql.operation.type", operationType),
			attribute.String("graphql.operation.name", operationName),
			attribute.String("component", "graphql_resolver"),
		),
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
	)

	return ctx, span
}

// AddWorkflowAttributes adds workflow-specific attributes to a span
func (t *TracingProvider) AddWorkflowAttributes(span oteltrace.Span, attributes map[string]interface{}) {
	for key, value := range attributes {
		switch v := value.(type) {
		case string:
			span.SetAttributes(attribute.String(key, v))
		case int:
			span.SetAttributes(attribute.Int(key, v))
		case int64:
			span.SetAttributes(attribute.Int64(key, v))
		case float64:
			span.SetAttributes(attribute.Float64(key, v))
		case bool:
			span.SetAttributes(attribute.Bool(key, v))
		default:
			span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
		}
	}
}

// AddClinicalAttributes adds clinical workflow specific attributes
func (t *TracingProvider) AddClinicalAttributes(span oteltrace.Span, patientID, providerID, encounterID string) {
	if patientID != "" {
		span.SetAttributes(attribute.String("patient.id", patientID))
	}
	if providerID != "" {
		span.SetAttributes(attribute.String("provider.id", providerID))
	}
	if encounterID != "" {
		span.SetAttributes(attribute.String("encounter.id", encounterID))
	}
}

// AddValidationAttributes adds safety validation attributes
func (t *TracingProvider) AddValidationAttributes(span oteltrace.Span, validationID, verdict string, riskScore float64, findingsCount int) {
	span.SetAttributes(
		attribute.String("validation.id", validationID),
		attribute.String("validation.verdict", verdict),
		attribute.Float64("validation.risk_score", riskScore),
		attribute.Int("validation.findings_count", findingsCount),
	)
}

// AddMedicationAttributes adds medication-specific attributes
func (t *TracingProvider) AddMedicationAttributes(span oteltrace.Span, proposalSetID, snapshotID, medicationOrderID string) {
	if proposalSetID != "" {
		span.SetAttributes(attribute.String("medication.proposal_set_id", proposalSetID))
	}
	if snapshotID != "" {
		span.SetAttributes(attribute.String("medication.snapshot_id", snapshotID))
	}
	if medicationOrderID != "" {
		span.SetAttributes(attribute.String("medication.order_id", medicationOrderID))
	}
}

// AddErrorAttributes adds error information to a span
func (t *TracingProvider) AddErrorAttributes(span oteltrace.Span, err error, errorType string) {
	span.RecordError(err)
	span.SetAttributes(
		attribute.Bool("error", true),
		attribute.String("error.type", errorType),
		attribute.String("error.message", err.Error()),
	)
}

// AddPerformanceAttributes adds performance metrics to a span
func (t *TracingProvider) AddPerformanceAttributes(span oteltrace.Span, phase string, duration float64, targetMS float64) {
	span.SetAttributes(
		attribute.String("performance.phase", phase),
		attribute.Float64("performance.duration_ms", duration),
		attribute.Float64("performance.target_ms", targetMS),
		attribute.Bool("performance.target_exceeded", duration > targetMS),
	)
}

// RecordEvent records a structured event in the current span
func (t *TracingProvider) RecordEvent(span oteltrace.Span, eventName string, attributes map[string]interface{}) {
	var eventAttrs []attribute.KeyValue
	for key, value := range attributes {
		switch v := value.(type) {
		case string:
			eventAttrs = append(eventAttrs, attribute.String(key, v))
		case int:
			eventAttrs = append(eventAttrs, attribute.Int(key, v))
		case int64:
			eventAttrs = append(eventAttrs, attribute.Int64(key, v))
		case float64:
			eventAttrs = append(eventAttrs, attribute.Float64(key, v))
		case bool:
			eventAttrs = append(eventAttrs, attribute.Bool(key, v))
		default:
			eventAttrs = append(eventAttrs, attribute.String(key, fmt.Sprintf("%v", v)))
		}
	}
	
	span.AddEvent(eventName, oteltrace.WithAttributes(eventAttrs...))
}

// GetTraceID extracts trace ID from context for correlation
func (t *TracingProvider) GetTraceID(ctx context.Context) string {
	spanContext := oteltrace.SpanFromContext(ctx).SpanContext()
	if spanContext.IsValid() {
		return spanContext.TraceID().String()
	}
	return ""
}

// GetSpanID extracts span ID from context
func (t *TracingProvider) GetSpanID(ctx context.Context) string {
	spanContext := oteltrace.SpanFromContext(ctx).SpanContext()
	if spanContext.IsValid() {
		return spanContext.SpanID().String()
	}
	return ""
}

// InjectTraceContext injects trace context into headers for propagation
func (t *TracingProvider) InjectTraceContext(ctx context.Context) map[string]string {
	// This would typically use otel propagators to inject trace context
	// into HTTP headers for distributed tracing
	headers := make(map[string]string)
	
	traceID := t.GetTraceID(ctx)
	spanID := t.GetSpanID(ctx)
	
	if traceID != "" && spanID != "" {
		// Simplified trace context injection
		headers["X-Trace-ID"] = traceID
		headers["X-Span-ID"] = spanID
	}
	
	return headers
}

// LogWithTraceContext adds trace information to log entries
func (t *TracingProvider) LogWithTraceContext(ctx context.Context, logger *zap.Logger) *zap.Logger {
	traceID := t.GetTraceID(ctx)
	spanID := t.GetSpanID(ctx)
	
	if traceID != "" || spanID != "" {
		return logger.With(
			zap.String("trace_id", traceID),
			zap.String("span_id", spanID),
		)
	}
	
	return logger
}

// Common span finishing patterns
func FinishSpanWithSuccess(span oteltrace.Span) {
	span.SetAttributes(attribute.Bool("success", true))
	span.End()
}

func FinishSpanWithError(span oteltrace.Span, err error, errorType string) {
	span.RecordError(err)
	span.SetAttributes(
		attribute.Bool("success", false),
		attribute.Bool("error", true),
		attribute.String("error.type", errorType),
	)
	span.End()
}

func FinishSpanWithStatus(span oteltrace.Span, success bool, statusCode int) {
	span.SetAttributes(
		attribute.Bool("success", success),
		attribute.Int("status_code", statusCode),
	)
	span.End()
}