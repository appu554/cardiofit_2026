package monitoring

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics contains all Prometheus metrics for the workflow engine
type Metrics struct {
	// Workflow orchestration metrics
	WorkflowsTotal                 *prometheus.CounterVec
	WorkflowDuration              *prometheus.HistogramVec
	WorkflowExecutionPhase        *prometheus.HistogramVec
	WorkflowStatus                *prometheus.CounterVec
	WorkflowErrors                *prometheus.CounterVec
	
	// Performance tracking metrics
	CalculatePhaseDuration        *prometheus.HistogramVec
	ValidatePhaseDuration         *prometheus.HistogramVec
	CommitPhaseDuration           *prometheus.HistogramVec
	PerformanceTargetsExceeded    *prometheus.CounterVec
	
	// External service metrics
	ExternalServiceCalls          *prometheus.CounterVec
	ExternalServiceDuration       *prometheus.HistogramVec
	ExternalServiceErrors         *prometheus.CounterVec
	ExternalServiceHealthStatus   *prometheus.GaugeVec
	
	// Database metrics
	DatabaseQueries               *prometheus.CounterVec
	DatabaseQueryDuration         *prometheus.HistogramVec
	DatabaseConnections           *prometheus.GaugeVec
	DatabaseErrors                *prometheus.CounterVec
	
	// API metrics
	HTTPRequestsTotal             *prometheus.CounterVec
	HTTPRequestDuration           *prometheus.HistogramVec
	HTTPRequestSize               *prometheus.HistogramVec
	HTTPResponseSize              *prometheus.HistogramVec
	
	// GraphQL metrics
	GraphQLOperations             *prometheus.CounterVec
	GraphQLResolverDuration       *prometheus.HistogramVec
	GraphQLErrors                 *prometheus.CounterVec
	
	// Clinical safety metrics
	SafetyValidationResults       *prometheus.CounterVec
	SafetyFindingsSeverity        *prometheus.CounterVec
	ClinicalOverrides             *prometheus.CounterVec
	MedicationOrdersCommitted     *prometheus.CounterVec
	
	// System resource metrics
	ActiveWorkflows               *prometheus.GaugeVec
	MemoryUsage                   *prometheus.GaugeVec
	CPUUsage                      *prometheus.GaugeVec
	GoroutineCount                prometheus.Gauge
	
	// Authentication and security metrics
	AuthenticationAttempts        *prometheus.CounterVec
	AuthorizationFailures         *prometheus.CounterVec
	RateLimitExceeded             *prometheus.CounterVec
	SecurityViolations            *prometheus.CounterVec
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	return &Metrics{
		// Workflow orchestration metrics
		WorkflowsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_workflows_total",
				Help: "Total number of workflow executions",
			},
			[]string{"type", "patient_id", "provider_id"},
		),
		
		WorkflowDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "workflow_engine_workflow_duration_seconds",
				Help:    "Duration of workflow executions",
				Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
			},
			[]string{"type", "status", "phase"},
		),
		
		WorkflowExecutionPhase: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "workflow_engine_phase_duration_seconds",
				Help:    "Duration of individual workflow phases",
				Buckets: []float64{0.05, 0.1, 0.175, 0.25, 0.5, 1.0, 2.0},
			},
			[]string{"phase", "status"},
		),
		
		WorkflowStatus: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_workflow_status_total",
				Help: "Total workflow executions by final status",
			},
			[]string{"status", "type"},
		),
		
		WorkflowErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_workflow_errors_total",
				Help: "Total workflow execution errors by type",
			},
			[]string{"error_type", "phase", "component"},
		),
		
		// Performance tracking metrics
		CalculatePhaseDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "workflow_engine_calculate_phase_duration_seconds",
				Help:    "Duration of calculate phase (target: 175ms)",
				Buckets: []float64{0.05, 0.1, 0.175, 0.25, 0.5, 1.0},
			},
			[]string{"execution_mode", "success"},
		),
		
		ValidatePhaseDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "workflow_engine_validate_phase_duration_seconds",
				Help:    "Duration of validate phase (target: 100ms)",
				Buckets: []float64{0.025, 0.05, 0.1, 0.15, 0.25, 0.5},
			},
			[]string{"validation_level", "verdict"},
		),
		
		CommitPhaseDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "workflow_engine_commit_phase_duration_seconds",
				Help:    "Duration of commit phase (target: 50ms)",
				Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.2, 0.5},
			},
			[]string{"commit_mode", "success"},
		),
		
		PerformanceTargetsExceeded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_performance_targets_exceeded_total",
				Help: "Number of times performance targets were exceeded",
			},
			[]string{"phase", "target_ms"},
		),
		
		// External service metrics
		ExternalServiceCalls: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_external_service_calls_total",
				Help: "Total calls to external services",
			},
			[]string{"service", "method", "status_code"},
		),
		
		ExternalServiceDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "workflow_engine_external_service_duration_seconds",
				Help:    "Duration of external service calls",
				Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 5.0},
			},
			[]string{"service", "method"},
		),
		
		ExternalServiceErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_external_service_errors_total",
				Help: "Total errors from external services",
			},
			[]string{"service", "error_type"},
		),
		
		ExternalServiceHealthStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "workflow_engine_external_service_health",
				Help: "Health status of external services (1=healthy, 0=unhealthy)",
			},
			[]string{"service", "endpoint"},
		),
		
		// Database metrics
		DatabaseQueries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_database_queries_total",
				Help: "Total database queries",
			},
			[]string{"operation", "table", "status"},
		),
		
		DatabaseQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "workflow_engine_database_query_duration_seconds",
				Help:    "Duration of database queries",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25},
			},
			[]string{"operation", "table"},
		),
		
		DatabaseConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "workflow_engine_database_connections",
				Help: "Current database connections",
			},
			[]string{"state"}, // active, idle
		),
		
		DatabaseErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_database_errors_total",
				Help: "Total database errors",
			},
			[]string{"operation", "error_type"},
		),
		
		// API metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_http_requests_total",
				Help: "Total HTTP requests",
			},
			[]string{"method", "path", "status_code"},
		),
		
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "workflow_engine_http_request_duration_seconds",
				Help:    "Duration of HTTP requests",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		
		HTTPRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "workflow_engine_http_request_size_bytes",
				Help:    "Size of HTTP requests",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
			},
			[]string{"method", "path"},
		),
		
		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "workflow_engine_http_response_size_bytes",
				Help:    "Size of HTTP responses",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
			},
			[]string{"method", "path"},
		),
		
		// GraphQL metrics
		GraphQLOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_graphql_operations_total",
				Help: "Total GraphQL operations",
			},
			[]string{"operation_type", "operation_name", "status"},
		),
		
		GraphQLResolverDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "workflow_engine_graphql_resolver_duration_seconds",
				Help:    "Duration of GraphQL resolvers",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
			},
			[]string{"resolver", "operation"},
		),
		
		GraphQLErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_graphql_errors_total",
				Help: "Total GraphQL errors",
			},
			[]string{"error_type", "resolver"},
		),
		
		// Clinical safety metrics
		SafetyValidationResults: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_safety_validation_results_total",
				Help: "Total safety validation results by verdict",
			},
			[]string{"verdict", "validation_level"},
		),
		
		SafetyFindingsSeverity: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_safety_findings_severity_total",
				Help: "Total safety findings by severity level",
			},
			[]string{"severity", "category"},
		),
		
		ClinicalOverrides: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_clinical_overrides_total",
				Help: "Total clinical provider overrides",
			},
			[]string{"override_type", "provider_id", "specialty"},
		),
		
		MedicationOrdersCommitted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_medication_orders_committed_total",
				Help: "Total medication orders successfully committed",
			},
			[]string{"commit_mode", "validation_verdict"},
		),
		
		// System resource metrics
		ActiveWorkflows: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "workflow_engine_active_workflows",
				Help: "Current number of active workflows",
			},
			[]string{"type", "phase"},
		),
		
		MemoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "workflow_engine_memory_usage_bytes",
				Help: "Current memory usage",
			},
			[]string{"type"}, // heap, stack, sys
		),
		
		CPUUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "workflow_engine_cpu_usage_percent",
				Help: "Current CPU usage percentage",
			},
			[]string{"core"},
		),
		
		GoroutineCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "workflow_engine_goroutines_count",
				Help: "Current number of goroutines",
			},
		),
		
		// Authentication and security metrics
		AuthenticationAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_authentication_attempts_total",
				Help: "Total authentication attempts",
			},
			[]string{"result", "method"},
		),
		
		AuthorizationFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_authorization_failures_total",
				Help: "Total authorization failures",
			},
			[]string{"reason", "resource"},
		),
		
		RateLimitExceeded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_rate_limit_exceeded_total",
				Help: "Total rate limit violations",
			},
			[]string{"ip", "endpoint"},
		),
		
		SecurityViolations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_engine_security_violations_total",
				Help: "Total security violations detected",
			},
			[]string{"violation_type", "severity"},
		),
	}
}

// RecordWorkflowExecution records workflow execution metrics
func (m *Metrics) RecordWorkflowExecution(workflowType, patientID, providerID string, duration time.Duration, status string) {
	m.WorkflowsTotal.WithLabelValues(workflowType, patientID, providerID).Inc()
	m.WorkflowDuration.WithLabelValues(workflowType, status, "total").Observe(duration.Seconds())
	m.WorkflowStatus.WithLabelValues(status, workflowType).Inc()
}

// RecordPhaseExecution records individual phase metrics
func (m *Metrics) RecordPhaseExecution(phase string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.WorkflowExecutionPhase.WithLabelValues(phase, status).Observe(duration.Seconds())
	
	// Check performance targets and record violations
	var targetMS float64
	switch phase {
	case "calculate":
		targetMS = 175
		m.CalculatePhaseDuration.WithLabelValues("advanced", status).Observe(duration.Seconds())
	case "validate":
		targetMS = 100
		m.ValidatePhaseDuration.WithLabelValues("comprehensive", status).Observe(duration.Seconds())
	case "commit":
		targetMS = 50
		m.CommitPhaseDuration.WithLabelValues("conditional", status).Observe(duration.Seconds())
	}
	
	if duration.Milliseconds() > int64(targetMS) {
		m.PerformanceTargetsExceeded.WithLabelValues(phase, fmt.Sprintf("%.0f", targetMS)).Inc()
	}
}

// RecordExternalServiceCall records external service call metrics
func (m *Metrics) RecordExternalServiceCall(service, method string, duration time.Duration, statusCode int, err error) {
	status := fmt.Sprintf("%d", statusCode)
	m.ExternalServiceCalls.WithLabelValues(service, method, status).Inc()
	m.ExternalServiceDuration.WithLabelValues(service, method).Observe(duration.Seconds())
	
	if err != nil {
		errorType := "unknown"
		if statusCode >= 400 && statusCode < 500 {
			errorType = "client_error"
		} else if statusCode >= 500 {
			errorType = "server_error"
		}
		m.ExternalServiceErrors.WithLabelValues(service, errorType).Inc()
	}
}

// RecordDatabaseQuery records database operation metrics
func (m *Metrics) RecordDatabaseQuery(operation, table string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	
	m.DatabaseQueries.WithLabelValues(operation, table, status).Inc()
	m.DatabaseQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
	
	if err != nil {
		errorType := "query_error" // Could be more specific based on error type
		m.DatabaseErrors.WithLabelValues(operation, errorType).Inc()
	}
}

// RecordHTTPRequest records HTTP request metrics
func (m *Metrics) RecordHTTPRequest(method, path string, duration time.Duration, statusCode int, requestSize, responseSize int64) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, fmt.Sprintf("%d", statusCode)).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
	m.HTTPRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
	m.HTTPResponseSize.WithLabelValues(method, path).Observe(float64(responseSize))
}

// RecordSafetyValidation records safety validation metrics
func (m *Metrics) RecordSafetyValidation(verdict, validationLevel string, findings map[string]int) {
	m.SafetyValidationResults.WithLabelValues(verdict, validationLevel).Inc()
	
	for severity, count := range findings {
		for i := 0; i < count; i++ {
			m.SafetyFindingsSeverity.WithLabelValues(severity, "general").Inc()
		}
	}
}

// UpdateActiveWorkflows updates the active workflows gauge
func (m *Metrics) UpdateActiveWorkflows(workflowType, phase string, count float64) {
	m.ActiveWorkflows.WithLabelValues(workflowType, phase).Set(count)
}

// UpdateExternalServiceHealth updates external service health status
func (m *Metrics) UpdateExternalServiceHealth(service, endpoint string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.ExternalServiceHealthStatus.WithLabelValues(service, endpoint).Set(value)
}