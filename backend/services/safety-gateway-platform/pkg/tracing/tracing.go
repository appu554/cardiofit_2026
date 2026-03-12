package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"safety-gateway-platform/pkg/types"
)

// TracingConfig represents configuration for distributed tracing
type TracingConfig struct {
	Enabled     bool   `yaml:"enabled"`
	ServiceName string `yaml:"service_name"`
	Environment string `yaml:"environment"`
	Version     string `yaml:"version"`
	
	// Jaeger configuration
	JaegerEndpoint string  `yaml:"jaeger_endpoint"`
	SampleRate     float64 `yaml:"sample_rate"`
	
	// Trace attributes
	IncludeRequestData  bool `yaml:"include_request_data"`
	IncludeResponseData bool `yaml:"include_response_data"`
	IncludeEngineData   bool `yaml:"include_engine_data"`
}

// TracingManager manages distributed tracing for the Safety Gateway Platform
type TracingManager struct {
	config   TracingConfig
	tracer   oteltrace.Tracer
	logger   *zap.Logger
	provider *trace.TracerProvider
}

// NewTracingManager creates a new tracing manager
func NewTracingManager(config TracingConfig, logger *zap.Logger) (*TracingManager, error) {
	if !config.Enabled {
		return &TracingManager{
			config: config,
			logger: logger,
		}, nil
	}
	
	// Create Jaeger exporter
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}
	
	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.Version),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	
	// Create trace provider
	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(config.SampleRate)),
	)
	
	// Set global trace provider
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	
	// Create tracer
	tracer := provider.Tracer(config.ServiceName)
	
	logger.Info("Distributed tracing initialized",
		zap.String("service", config.ServiceName),
		zap.String("endpoint", config.JaegerEndpoint),
		zap.Float64("sample_rate", config.SampleRate),
	)
	
	return &TracingManager{
		config:   config,
		tracer:   tracer,
		logger:   logger,
		provider: provider,
	}, nil
}

// StartSpan starts a new trace span
func (tm *TracingManager) StartSpan(ctx context.Context, operationName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	if !tm.config.Enabled || tm.tracer == nil {
		return ctx, oteltrace.SpanFromContext(ctx)
	}
	
	return tm.tracer.Start(ctx, operationName, opts...)
}

// TraceRequest traces a safety request
func (tm *TracingManager) TraceRequest(ctx context.Context, req *types.SafetyRequest) (context.Context, oteltrace.Span) {
	ctx, span := tm.StartSpan(ctx, "safety_gateway.process_request",
		oteltrace.WithAttributes(
			attribute.String("request.id", req.RequestID),
			attribute.String("patient.id", req.PatientID),
			attribute.String("clinician.id", req.ClinicianID),
			attribute.String("action.type", req.ActionType),
			attribute.String("priority", req.Priority),
			attribute.String("source", req.Source),
			attribute.Int("medication.count", len(req.MedicationIDs)),
			attribute.Int("condition.count", len(req.ConditionIDs)),
			attribute.Int("allergy.count", len(req.AllergyIDs)),
		),
	)
	
	if tm.config.IncludeRequestData {
		span.SetAttributes(
			attribute.StringSlice("medication.ids", req.MedicationIDs),
			attribute.StringSlice("condition.ids", req.ConditionIDs),
			attribute.StringSlice("allergy.ids", req.AllergyIDs),
		)
	}
	
	return ctx, span
}

// TraceEngineExecution traces engine execution
func (tm *TracingManager) TraceEngineExecution(ctx context.Context, engineID, engineName string, req *types.SafetyRequest) (context.Context, oteltrace.Span) {
	ctx, span := tm.StartSpan(ctx, fmt.Sprintf("engine.%s.execute", engineID),
		oteltrace.WithAttributes(
			attribute.String("engine.id", engineID),
			attribute.String("engine.name", engineName),
			attribute.String("request.id", req.RequestID),
			attribute.String("patient.id", req.PatientID),
			attribute.String("action.type", req.ActionType),
		),
	)
	
	return ctx, span
}

// TraceCAERequest traces CAE service requests
func (tm *TracingManager) TraceCAERequest(ctx context.Context, method string, req *types.SafetyRequest) (context.Context, oteltrace.Span) {
	ctx, span := tm.StartSpan(ctx, fmt.Sprintf("cae.%s", method),
		oteltrace.WithAttributes(
			attribute.String("cae.method", method),
			attribute.String("request.id", req.RequestID),
			attribute.String("patient.id", req.PatientID),
			attribute.String("grpc.service", "clinical_reasoning"),
		),
	)
	
	return ctx, span
}

// FinishEngineSpan finishes an engine execution span with results
func (tm *TracingManager) FinishEngineSpan(span oteltrace.Span, result *types.EngineResult, err error) {
	if !tm.config.Enabled || span == nil {
		return
	}
	
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(
			attribute.String("engine.status", "error"),
			attribute.String("engine.error", err.Error()),
		)
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(
			attribute.String("engine.status", string(result.Status)),
			attribute.Float64("engine.risk_score", result.RiskScore),
			attribute.Float64("engine.confidence", result.Confidence),
			attribute.String("engine.tier", string(result.Tier)),
			attribute.Int64("engine.duration_ms", result.Duration.Milliseconds()),
			attribute.Int("engine.violations", len(result.Violations)),
			attribute.Int("engine.warnings", len(result.Warnings)),
		)
		
		if tm.config.IncludeEngineData {
			span.SetAttributes(
				attribute.StringSlice("engine.violations", result.Violations),
				attribute.StringSlice("engine.warnings", result.Warnings),
			)
		}
	}
	
	span.End()
}

// FinishCAESpan finishes a CAE request span
func (tm *TracingManager) FinishCAESpan(span oteltrace.Span, status string, duration time.Duration, err error) {
	if !tm.config.Enabled || span == nil {
		return
	}
	
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(
			attribute.String("cae.status", "error"),
			attribute.String("cae.error", err.Error()),
		)
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(
			attribute.String("cae.status", status),
		)
	}
	
	span.SetAttributes(
		attribute.Int64("cae.duration_ms", duration.Milliseconds()),
	)
	
	span.End()
}

// FinishRequestSpan finishes a request span with response
func (tm *TracingManager) FinishRequestSpan(span oteltrace.Span, resp *types.SafetyResponse, err error) {
	if !tm.config.Enabled || span == nil {
		return
	}
	
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(
			attribute.String("response.status", "error"),
			attribute.String("response.error", err.Error()),
		)
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(
			attribute.String("response.status", string(resp.Status)),
			attribute.Float64("response.risk_score", resp.RiskScore),
			attribute.Int64("response.processing_time_ms", resp.ProcessingTime.Milliseconds()),
			attribute.Int("response.engine_count", len(resp.EngineResults)),
			attribute.Int("response.violations", len(resp.Violations)),
			attribute.Int("response.warnings", len(resp.Warnings)),
		)
		
		if resp.OverrideToken != "" {
			span.SetAttributes(
				attribute.String("response.override_token", resp.OverrideToken),
				attribute.Bool("response.override_used", true),
			)
		}
		
		if tm.config.IncludeResponseData {
			span.SetAttributes(
				attribute.StringSlice("response.violations", resp.Violations),
				attribute.StringSlice("response.warnings", resp.Warnings),
			)
		}
	}
	
	span.End()
}

// AddEvent adds an event to the current span
func (tm *TracingManager) AddEvent(ctx context.Context, name string, attributes ...attribute.KeyValue) {
	if !tm.config.Enabled {
		return
	}
	
	span := oteltrace.SpanFromContext(ctx)
	if span != nil {
		span.AddEvent(name, oteltrace.WithAttributes(attributes...))
	}
}

// SetAttribute sets an attribute on the current span
func (tm *TracingManager) SetAttribute(ctx context.Context, key string, value interface{}) {
	if !tm.config.Enabled {
		return
	}
	
	span := oteltrace.SpanFromContext(ctx)
	if span != nil {
		var attr attribute.KeyValue
		switch v := value.(type) {
		case string:
			attr = attribute.String(key, v)
		case int:
			attr = attribute.Int(key, v)
		case int64:
			attr = attribute.Int64(key, v)
		case float64:
			attr = attribute.Float64(key, v)
		case bool:
			attr = attribute.Bool(key, v)
		default:
			attr = attribute.String(key, fmt.Sprintf("%v", v))
		}
		span.SetAttributes(attr)
	}
}

// RecordError records an error on the current span
func (tm *TracingManager) RecordError(ctx context.Context, err error) {
	if !tm.config.Enabled {
		return
	}
	
	span := oteltrace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// InjectHeaders injects trace context into HTTP headers
func (tm *TracingManager) InjectHeaders(ctx context.Context, headers map[string]string) {
	if !tm.config.Enabled {
		return
	}
	
	carrier := propagation.MapCarrier(headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// ExtractHeaders extracts trace context from HTTP headers
func (tm *TracingManager) ExtractHeaders(ctx context.Context, headers map[string]string) context.Context {
	if !tm.config.Enabled {
		return ctx
	}
	
	carrier := propagation.MapCarrier(headers)
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// GetTraceID returns the trace ID from the current context
func (tm *TracingManager) GetTraceID(ctx context.Context) string {
	if !tm.config.Enabled {
		return ""
	}
	
	span := oteltrace.SpanFromContext(ctx)
	if span != nil {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID returns the span ID from the current context
func (tm *TracingManager) GetSpanID(ctx context.Context) string {
	if !tm.config.Enabled {
		return ""
	}
	
	span := oteltrace.SpanFromContext(ctx)
	if span != nil {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// Shutdown shuts down the tracing provider
func (tm *TracingManager) Shutdown(ctx context.Context) error {
	if tm.provider != nil {
		return tm.provider.Shutdown(ctx)
	}
	return nil
}

// TraceMiddleware provides HTTP middleware for tracing
func (tm *TracingManager) TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !tm.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		
		// Extract trace context from headers
		ctx := tm.ExtractHeaders(r.Context(), map[string]string{
			"traceparent": r.Header.Get("traceparent"),
			"tracestate":  r.Header.Get("tracestate"),
		})
		
		// Start span for HTTP request
		ctx, span := tm.StartSpan(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			oteltrace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.scheme", r.URL.Scheme),
				attribute.String("http.host", r.Host),
				attribute.String("http.user_agent", r.UserAgent()),
			),
		)
		defer span.End()
		
		// Inject trace context into response headers
		headers := make(map[string]string)
		tm.InjectHeaders(ctx, headers)
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		
		// Continue with request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
