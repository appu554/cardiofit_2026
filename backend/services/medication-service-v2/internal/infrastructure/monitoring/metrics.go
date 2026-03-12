package monitoring

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Metrics holds all Prometheus metrics for the medication service
type Metrics struct {
	// HTTP metrics
	httpRequestsTotal        *prometheus.CounterVec
	httpRequestDuration      *prometheus.HistogramVec
	httpRequestsInFlight     prometheus.Gauge
	
	// Business metrics
	medicationProposalsTotal    *prometheus.CounterVec
	medicationProposalDuration  *prometheus.HistogramVec
	recipeResolutionDuration    *prometheus.HistogramVec
	snapshotCreationDuration    *prometheus.HistogramVec
	clinicalCalculationDuration *prometheus.HistogramVec
	
	// Safety and compliance metrics
	safetyViolationsTotal       *prometheus.CounterVec
	validationFailuresTotal     *prometheus.CounterVec
	complianceChecksTotal       *prometheus.CounterVec
	
	// Performance metrics
	cacheHitRatio              *prometheus.GaugeVec
	databaseConnectionsActive  prometheus.Gauge
	redisConnectionsActive     prometheus.Gauge
	
	// Error metrics
	errorsTotal                *prometheus.CounterVec
	
	// Healthcare-specific metrics
	patientContextAge          *prometheus.HistogramVec
	clinicalDataFreshness      *prometheus.GaugeVec
	dosageCalculationAccuracy  *prometheus.GaugeVec
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	return &Metrics{
		// HTTP metrics
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "medication_service_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "medication_service_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "endpoint"},
		),
		httpRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "medication_service_http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),
		
		// Business metrics
		medicationProposalsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "medication_service_proposals_total",
				Help: "Total number of medication proposals created",
			},
			[]string{"indication", "status", "protocol"},
		),
		medicationProposalDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "medication_service_proposal_processing_duration_seconds",
				Help:    "Time taken to process medication proposals",
				Buckets: []float64{0.01, 0.05, 0.1, 0.15, 0.2, 0.25, 0.5, 1, 2, 5},
			},
			[]string{"indication", "protocol"},
		),
		recipeResolutionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "medication_service_recipe_resolution_duration_seconds",
				Help:    "Time taken to resolve recipes",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
			},
			[]string{"protocol_id"},
		),
		snapshotCreationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "medication_service_snapshot_creation_duration_seconds",
				Help:    "Time taken to create clinical snapshots",
				Buckets: []float64{0.01, 0.025, 0.05, 0.075, 0.1, 0.15, 0.2, 0.5, 1},
			},
			[]string{"snapshot_type"},
		),
		clinicalCalculationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "medication_service_clinical_calculation_duration_seconds",
				Help:    "Time taken for clinical calculations",
				Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.15, 0.2, 0.5},
			},
			[]string{"calculation_type"},
		),
		
		// Safety and compliance metrics
		safetyViolationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "medication_service_safety_violations_total",
				Help: "Total number of safety violations detected",
			},
			[]string{"violation_type", "severity"},
		),
		validationFailuresTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "medication_service_validation_failures_total",
				Help: "Total number of validation failures",
			},
			[]string{"validation_type", "reason"},
		),
		complianceChecksTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "medication_service_compliance_checks_total",
				Help: "Total number of compliance checks performed",
			},
			[]string{"regulation", "status"},
		),
		
		// Performance metrics
		cacheHitRatio: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "medication_service_cache_hit_ratio",
				Help: "Cache hit ratio for different cache types",
			},
			[]string{"cache_type"},
		),
		databaseConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "medication_service_database_connections_active",
				Help: "Number of active database connections",
			},
		),
		redisConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "medication_service_redis_connections_active",
				Help: "Number of active Redis connections",
			},
		),
		
		// Error metrics
		errorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "medication_service_errors_total",
				Help: "Total number of errors by type",
			},
			[]string{"error_type", "component"},
		),
		
		// Healthcare-specific metrics
		patientContextAge: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "medication_service_patient_context_age_seconds",
				Help:    "Age of patient clinical context data",
				Buckets: []float64{60, 300, 900, 1800, 3600, 7200, 14400, 28800, 86400},
			},
			[]string{"data_type"},
		),
		clinicalDataFreshness: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "medication_service_clinical_data_freshness_score",
				Help: "Freshness score of clinical data (0-1)",
			},
			[]string{"data_source", "patient_id"},
		),
		dosageCalculationAccuracy: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "medication_service_dosage_calculation_accuracy",
				Help: "Accuracy score of dosage calculations (0-1)",
			},
			[]string{"calculation_method", "drug_class"},
		),
	}
}

// HTTP metrics methods
func (m *Metrics) RecordHTTPRequest(method, endpoint, status string, duration time.Duration) {
	m.httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	m.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

func (m *Metrics) IncRequestsInFlight() {
	m.httpRequestsInFlight.Inc()
}

func (m *Metrics) DecRequestsInFlight() {
	m.httpRequestsInFlight.Dec()
}

// Business metrics methods
func (m *Metrics) RecordMedicationProposal(indication, status, protocol string) {
	m.medicationProposalsTotal.WithLabelValues(indication, status, protocol).Inc()
}

func (m *Metrics) RecordDuration(metricType string, duration time.Duration, labels ...string) {
	switch metricType {
	case "medication_proposal":
		if len(labels) >= 2 {
			m.medicationProposalDuration.WithLabelValues(labels[0], labels[1]).Observe(duration.Seconds())
		}
	case "recipe_resolution":
		if len(labels) >= 1 {
			m.recipeResolutionDuration.WithLabelValues(labels[0]).Observe(duration.Seconds())
		}
	case "snapshot_creation":
		if len(labels) >= 1 {
			m.snapshotCreationDuration.WithLabelValues(labels[0]).Observe(duration.Seconds())
		}
	case "clinical_calculation":
		if len(labels) >= 1 {
			m.clinicalCalculationDuration.WithLabelValues(labels[0]).Observe(duration.Seconds())
		}
	}
}

// Safety and compliance metrics
func (m *Metrics) RecordSafetyViolation(violationType, severity string) {
	m.safetyViolationsTotal.WithLabelValues(violationType, severity).Inc()
}

func (m *Metrics) RecordValidationFailure(validationType, reason string) {
	m.validationFailuresTotal.WithLabelValues(validationType, reason).Inc()
}

func (m *Metrics) RecordComplianceCheck(regulation, status string) {
	m.complianceChecksTotal.WithLabelValues(regulation, status).Inc()
}

// Performance metrics
func (m *Metrics) SetCacheHitRatio(cacheType string, ratio float64) {
	m.cacheHitRatio.WithLabelValues(cacheType).Set(ratio)
}

func (m *Metrics) SetActiveConnections(connectionType string, count float64) {
	switch connectionType {
	case "database":
		m.databaseConnectionsActive.Set(count)
	case "redis":
		m.redisConnectionsActive.Set(count)
	}
}

// Error metrics
func (m *Metrics) RecordError(errorType, component string) {
	m.errorsTotal.WithLabelValues(errorType, component).Inc()
}

// Healthcare-specific metrics
func (m *Metrics) RecordPatientContextAge(dataType string, age time.Duration) {
	m.patientContextAge.WithLabelValues(dataType).Observe(age.Seconds())
}

func (m *Metrics) SetClinicalDataFreshness(dataSource, patientID string, score float64) {
	m.clinicalDataFreshness.WithLabelValues(dataSource, patientID).Set(score)
}

func (m *Metrics) SetDosageCalculationAccuracy(calculationMethod, drugClass string, accuracy float64) {
	m.dosageCalculationAccuracy.WithLabelValues(calculationMethod, drugClass).Set(accuracy)
}

// Generic methods for convenience
func (m *Metrics) RecordCounter(name string, value float64, labels map[string]string) {
	// This would need to be implemented based on dynamic metric creation
	// For now, using the existing specific methods is recommended
}

func (m *Metrics) RecordGauge(name string, value float64, labels map[string]string) {
	// This would need to be implemented based on dynamic metric creation
	// For now, using the existing specific methods is recommended
}

// OpenTelemetry tracing setup
func InitTracing(serviceName, version string) (*trace.TracerProvider, error) {
	// Create Jaeger exporter
	jaegerExporter, err := jaeger.New(jaeger.WithCollectorEndpoint())
	if err != nil {
		return nil, err
	}

	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
			semconv.ServiceInstanceID("medication-service-v2-instance"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create tracer provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(jaegerExporter),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(0.1)), // Sample 10% of traces
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	return tp, nil
}

// GetTracer returns a tracer for the medication service
func GetTracer() oteltrace.Tracer {
	return otel.Tracer("medication-service-v2")
}

// HealthMetrics provides health-specific metrics
type HealthMetrics struct {
	DatabaseResponseTime time.Duration `json:"database_response_time"`
	RedisResponseTime    time.Duration `json:"redis_response_time"`
	CacheHitRate        float64       `json:"cache_hit_rate"`
	ActiveSessions      int           `json:"active_sessions"`
	ProcessingLatencyP95 time.Duration `json:"processing_latency_p95"`
	ErrorRate           float64       `json:"error_rate"`
}

// GetHealthMetrics returns current health metrics
func (m *Metrics) GetHealthMetrics() *HealthMetrics {
	// This would query the actual metrics registry
	// For now, returning a placeholder structure
	return &HealthMetrics{
		DatabaseResponseTime: 5 * time.Millisecond,
		RedisResponseTime:    1 * time.Millisecond,
		CacheHitRate:        0.85,
		ActiveSessions:      25,
		ProcessingLatencyP95: 150 * time.Millisecond,
		ErrorRate:           0.001,
	}
}