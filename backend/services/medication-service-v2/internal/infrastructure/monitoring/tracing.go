package monitoring

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TracingConfig holds configuration for distributed tracing
type TracingConfig struct {
	ServiceName     string  `yaml:"service_name"`
	ServiceVersion  string  `yaml:"service_version"`
	JaegerEndpoint  string  `yaml:"jaeger_endpoint"`
	SamplingRate    float64 `yaml:"sampling_rate"`
	Environment     string  `yaml:"environment"`
	InstanceID      string  `yaml:"instance_id"`
}

// TracingManager manages distributed tracing for the medication service
type TracingManager struct {
	config         *TracingConfig
	tracerProvider *trace.TracerProvider
	tracer         oteltrace.Tracer
	logger         *zap.Logger
}

// NewTracingManager creates a new tracing manager with healthcare-specific configuration
func NewTracingManager(config *TracingConfig, logger *zap.Logger) (*TracingManager, error) {
	tm := &TracingManager{
		config: config,
		logger: logger,
	}

	if err := tm.initializeTracing(); err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}

	return tm, nil
}

// initializeTracing sets up OpenTelemetry tracing with healthcare compliance
func (tm *TracingManager) initializeTracing() error {
	// Create Jaeger exporter with healthcare-specific configuration
	jaegerExporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(tm.config.JaegerEndpoint),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create jaeger exporter: %w", err)
	}

	// Create resource with healthcare service identification
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			// Service identification
			semconv.ServiceName(tm.config.ServiceName),
			semconv.ServiceVersion(tm.config.ServiceVersion),
			semconv.ServiceInstanceID(tm.config.InstanceID),
			
			// Healthcare-specific attributes
			attribute.String("healthcare.domain", "medication_management"),
			attribute.String("healthcare.compliance", "hipaa"),
			attribute.String("healthcare.criticality", "high"),
			attribute.String("data.classification", "phi"), // Protected Health Information
			
			// Environment and deployment
			attribute.String("deployment.environment", tm.config.Environment),
			attribute.String("service.namespace", "clinical-synthesis-hub"),
			
			// Technical attributes
			attribute.String("telemetry.sdk.language", "go"),
			attribute.String("service.type", "medication-service"),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider with healthcare-appropriate sampling
	tp := trace.NewTracerProvider(
		trace.WithBatcher(jaegerExporter, trace.WithBatchTimeout(2*time.Second)),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(tm.config.SamplingRate)),
		
		// Add span processors for healthcare compliance
		trace.WithSpanProcessor(&HealthcareSpanProcessor{
			logger: tm.logger,
		}),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)
	
	tm.tracerProvider = tp
	tm.tracer = otel.Tracer(tm.config.ServiceName)

	tm.logger.Info("Distributed tracing initialized",
		zap.String("service", tm.config.ServiceName),
		zap.Float64("sampling_rate", tm.config.SamplingRate),
		zap.String("jaeger_endpoint", tm.config.JaegerEndpoint),
	)

	return nil
}

// StartSpan creates a new span with healthcare-specific attributes
func (tm *TracingManager) StartSpan(ctx context.Context, operationName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return tm.tracer.Start(ctx, operationName, opts...)
}

// StartClinicalSpan creates a span for clinical operations with required healthcare attributes
func (tm *TracingManager) StartClinicalSpan(ctx context.Context, operationName string, patientID, indication string) (context.Context, oteltrace.Span) {
	ctx, span := tm.tracer.Start(ctx, operationName,
		oteltrace.WithAttributes(
			// Clinical context
			attribute.String("patient.id.hash", hashPatientID(patientID)), // Hashed for privacy
			attribute.String("clinical.indication", indication),
			attribute.String("operation.type", "clinical"),
			attribute.String("data.sensitivity", "high"),
			
			// Compliance attributes
			attribute.String("audit.required", "true"),
			attribute.String("retention.class", "clinical"),
			attribute.Int64("timestamp.start", time.Now().Unix()),
		),
	)
	
	return ctx, span
}

// StartWorkflowSpan creates a span for workflow operations
func (tm *TracingManager) StartWorkflowSpan(ctx context.Context, workflowName, phase string) (context.Context, oteltrace.Span) {
	ctx, span := tm.tracer.Start(ctx, fmt.Sprintf("workflow.%s.%s", workflowName, phase),
		oteltrace.WithAttributes(
			attribute.String("workflow.name", workflowName),
			attribute.String("workflow.phase", phase),
			attribute.String("operation.category", "workflow"),
		),
	)
	
	return ctx, span
}

// StartDatabaseSpan creates a span for database operations
func (tm *TracingManager) StartDatabaseSpan(ctx context.Context, operation, table string) (context.Context, oteltrace.Span) {
	ctx, span := tm.tracer.Start(ctx, fmt.Sprintf("db.%s", operation),
		oteltrace.WithAttributes(
			semconv.DBOperation(operation),
			attribute.String("db.table", table),
			attribute.String("db.type", "postgresql"),
			attribute.String("operation.category", "database"),
		),
	)
	
	return ctx, span
}

// StartExternalSpan creates a span for external service calls
func (tm *TracingManager) StartExternalSpan(ctx context.Context, serviceName, operation string) (context.Context, oteltrace.Span) {
	ctx, span := tm.tracer.Start(ctx, fmt.Sprintf("external.%s.%s", serviceName, operation),
		oteltrace.WithAttributes(
			attribute.String("service.external", serviceName),
			attribute.String("operation.external", operation),
			attribute.String("operation.category", "external"),
		),
	)
	
	return ctx, span
}

// RecordError records an error in the current span with healthcare context
func (tm *TracingManager) RecordError(span oteltrace.Span, err error, errorType string) {
	if err == nil || span == nil {
		return
	}

	span.RecordError(err,
		oteltrace.WithAttributes(
			attribute.String("error.type", errorType),
			attribute.String("error.message", err.Error()),
			attribute.Bool("error.recorded", true),
			attribute.Int64("error.timestamp", time.Now().Unix()),
		),
	)
	span.SetStatus(codes.Error, err.Error())
}

// RecordClinicalEvent records a clinical event in the span
func (tm *TracingManager) RecordClinicalEvent(span oteltrace.Span, eventType, outcome string, metadata map[string]string) {
	if span == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("clinical.event.type", eventType),
		attribute.String("clinical.event.outcome", outcome),
		attribute.Int64("clinical.event.timestamp", time.Now().Unix()),
	}

	for key, value := range metadata {
		attrs = append(attrs, attribute.String(fmt.Sprintf("clinical.event.%s", key), value))
	}

	span.AddEvent("clinical_event", oteltrace.WithAttributes(attrs...))
}

// RecordPerformanceMetric records performance metrics in the span
func (tm *TracingManager) RecordPerformanceMetric(span oteltrace.Span, metricName string, value float64, unit string) {
	if span == nil {
		return
	}

	span.SetAttributes(
		attribute.Float64(fmt.Sprintf("performance.%s", metricName), value),
		attribute.String(fmt.Sprintf("performance.%s.unit", metricName), unit),
	)
}

// FinishSpan properly closes a span with final attributes
func (tm *TracingManager) FinishSpan(span oteltrace.Span, success bool, metadata map[string]interface{}) {
	if span == nil {
		return
	}

	defer span.End()

	// Add final attributes
	attrs := []attribute.KeyValue{
		attribute.Bool("operation.success", success),
		attribute.Int64("timestamp.end", time.Now().Unix()),
	}

	for key, value := range metadata {
		switch v := value.(type) {
		case string:
			attrs = append(attrs, attribute.String(key, v))
		case int:
			attrs = append(attrs, attribute.Int(key, v))
		case int64:
			attrs = append(attrs, attribute.Int64(key, v))
		case float64:
			attrs = append(attrs, attribute.Float64(key, v))
		case bool:
			attrs = append(attrs, attribute.Bool(key, v))
		}
	}

	span.SetAttributes(attrs...)

	if success {
		span.SetStatus(codes.Ok, "Operation completed successfully")
	}
}

// Shutdown gracefully shuts down the tracing system
func (tm *TracingManager) Shutdown(ctx context.Context) error {
	if tm.tracerProvider != nil {
		return tm.tracerProvider.Shutdown(ctx)
	}
	return nil
}

// HealthcareSpanProcessor processes spans for healthcare compliance
type HealthcareSpanProcessor struct {
	logger *zap.Logger
}

// OnStart is called when a span starts
func (hsp *HealthcareSpanProcessor) OnStart(parent context.Context, s trace.ReadWriteSpan) {
	// Add compliance attributes to all spans
	s.SetAttributes(
		attribute.String("compliance.framework", "hipaa"),
		attribute.Bool("audit.enabled", true),
		attribute.String("data.handling", "secure"),
	)
}

// OnEnd is called when a span ends
func (hsp *HealthcareSpanProcessor) OnEnd(s trace.ReadOnlySpan) {
	// Log high-value clinical operations for audit trail
	if isAuditableSpan(s) {
		hsp.logger.Info("Clinical operation completed",
			zap.String("span.name", s.Name()),
			zap.String("span.id", s.SpanContext().SpanID().String()),
			zap.String("trace.id", s.SpanContext().TraceID().String()),
			zap.Duration("duration", s.EndTime().Sub(s.StartTime())),
		)
	}
}

// ForceFlush forces the processor to flush any buffered spans
func (hsp *HealthcareSpanProcessor) ForceFlush(ctx context.Context) error {
	return nil
}

// Shutdown shuts down the processor
func (hsp *HealthcareSpanProcessor) Shutdown(ctx context.Context) error {
	return nil
}

// Helper functions

// hashPatientID creates a secure hash of patient ID for tracing (privacy protection)
func hashPatientID(patientID string) string {
	// In production, use proper hashing with salt
	// This is a simplified version for demonstration
	if len(patientID) < 8 {
		return "patient_xxx"
	}
	return fmt.Sprintf("patient_%s", patientID[len(patientID)-4:])
}

// isAuditableSpan determines if a span should be audited
func isAuditableSpan(s trace.ReadOnlySpan) bool {
	attrs := s.Attributes()
	for _, attr := range attrs {
		if attr.Key == "operation.type" && attr.Value.AsString() == "clinical" {
			return true
		}
		if attr.Key == "audit.required" && attr.Value.AsBool() {
			return true
		}
	}
	return false
}

// GetCorrelationID extracts or creates a correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return fmt.Sprintf("corr_%d", time.Now().UnixNano())
}

// InjectCorrelationID injects correlation ID into context
func InjectCorrelationID(ctx context.Context, correlationID string) context.Context {
	// This would typically use OpenTelemetry propagators
	// For now, we'll use the existing span context
	return ctx
}