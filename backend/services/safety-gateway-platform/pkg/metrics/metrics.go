package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// MetricsCollector handles all metrics collection for the Safety Gateway Platform
type MetricsCollector struct {
	// Request metrics
	RequestsTotal        *prometheus.CounterVec
	RequestDuration      *prometheus.HistogramVec
	RequestsInFlight     prometheus.Gauge
	
	// Engine metrics
	EngineRequestsTotal  *prometheus.CounterVec
	EngineRequestDuration *prometheus.HistogramVec
	EngineHealthStatus   *prometheus.GaugeVec
	EngineErrors         *prometheus.CounterVec
	
	// Safety decision metrics
	SafetyDecisionsTotal *prometheus.CounterVec
	RiskScoreDistribution *prometheus.HistogramVec
	OverrideTokensUsed   *prometheus.CounterVec
	
	// System metrics
	CacheHitRate         *prometheus.GaugeVec
	CircuitBreakerStatus *prometheus.GaugeVec
	ActiveConnections    prometheus.Gauge
	
	// CAE integration metrics
	CAERequestsTotal     *prometheus.CounterVec
	CAERequestDuration   *prometheus.HistogramVec
	CAEConnectionStatus  prometheus.Gauge
	
	logger *zap.Logger
	registry *prometheus.Registry
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger *zap.Logger) *MetricsCollector {
	registry := prometheus.NewRegistry()
	
	mc := &MetricsCollector{
		logger:   logger,
		registry: registry,
	}
	
	mc.initializeMetrics()
	mc.registerMetrics()
	
	return mc
}

// initializeMetrics initializes all Prometheus metrics
func (mc *MetricsCollector) initializeMetrics() {
	// Request metrics
	mc.RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "safety_gateway_requests_total",
			Help: "Total number of requests processed by the Safety Gateway",
		},
		[]string{"method", "endpoint", "status_code"},
	)
	
	mc.RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "safety_gateway_request_duration_seconds",
			Help:    "Duration of requests processed by the Safety Gateway",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"method", "endpoint"},
	)
	
	mc.RequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "safety_gateway_requests_in_flight",
			Help: "Number of requests currently being processed",
		},
	)
	
	// Engine metrics
	mc.EngineRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "safety_gateway_engine_requests_total",
			Help: "Total number of requests sent to safety engines",
		},
		[]string{"engine_id", "engine_name", "status"},
	)
	
	mc.EngineRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "safety_gateway_engine_request_duration_seconds",
			Help:    "Duration of requests to safety engines",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		},
		[]string{"engine_id", "engine_name"},
	)
	
	mc.EngineHealthStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "safety_gateway_engine_health_status",
			Help: "Health status of safety engines (1=healthy, 0=unhealthy)",
		},
		[]string{"engine_id", "engine_name"},
	)
	
	mc.EngineErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "safety_gateway_engine_errors_total",
			Help: "Total number of errors from safety engines",
		},
		[]string{"engine_id", "engine_name", "error_type"},
	)
	
	// Safety decision metrics
	mc.SafetyDecisionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "safety_gateway_decisions_total",
			Help: "Total number of safety decisions made",
		},
		[]string{"status", "tier", "patient_id"},
	)
	
	mc.RiskScoreDistribution = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "safety_gateway_risk_score_distribution",
			Help:    "Distribution of risk scores in safety decisions",
			Buckets: []float64{0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		},
		[]string{"status", "tier"},
	)
	
	mc.OverrideTokensUsed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "safety_gateway_override_tokens_used_total",
			Help: "Total number of override tokens used for unsafe decisions",
		},
		[]string{"clinician_id", "reason"},
	)
	
	// System metrics
	mc.CacheHitRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "safety_gateway_cache_hit_rate",
			Help: "Cache hit rate for different cache levels",
		},
		[]string{"cache_level", "cache_type"},
	)
	
	mc.CircuitBreakerStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "safety_gateway_circuit_breaker_status",
			Help: "Circuit breaker status (0=closed, 1=open, 2=half-open)",
		},
		[]string{"service", "endpoint"},
	)
	
	mc.ActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "safety_gateway_active_connections",
			Help: "Number of active gRPC connections",
		},
	)
	
	// CAE integration metrics
	mc.CAERequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "safety_gateway_cae_requests_total",
			Help: "Total number of requests sent to CAE service",
		},
		[]string{"method", "status"},
	)
	
	mc.CAERequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "safety_gateway_cae_request_duration_seconds",
			Help:    "Duration of requests to CAE service",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		},
		[]string{"method"},
	)
	
	mc.CAEConnectionStatus = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "safety_gateway_cae_connection_status",
			Help: "CAE service connection status (1=connected, 0=disconnected)",
		},
	)
}

// registerMetrics registers all metrics with the Prometheus registry
func (mc *MetricsCollector) registerMetrics() {
	metrics := []prometheus.Collector{
		mc.RequestsTotal,
		mc.RequestDuration,
		mc.RequestsInFlight,
		mc.EngineRequestsTotal,
		mc.EngineRequestDuration,
		mc.EngineHealthStatus,
		mc.EngineErrors,
		mc.SafetyDecisionsTotal,
		mc.RiskScoreDistribution,
		mc.OverrideTokensUsed,
		mc.CacheHitRate,
		mc.CircuitBreakerStatus,
		mc.ActiveConnections,
		mc.CAERequestsTotal,
		mc.CAERequestDuration,
		mc.CAEConnectionStatus,
	}
	
	for _, metric := range metrics {
		mc.registry.MustRegister(metric)
	}
	
	mc.logger.Info("Metrics registered successfully", zap.Int("total_metrics", len(metrics)))
}

// RecordRequest records metrics for an incoming request
func (mc *MetricsCollector) RecordRequest(method, endpoint string, statusCode int, duration time.Duration) {
	mc.RequestsTotal.WithLabelValues(method, endpoint, strconv.Itoa(statusCode)).Inc()
	mc.RequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordRequestStart records the start of a request
func (mc *MetricsCollector) RecordRequestStart() {
	mc.RequestsInFlight.Inc()
}

// RecordRequestEnd records the end of a request
func (mc *MetricsCollector) RecordRequestEnd() {
	mc.RequestsInFlight.Dec()
}

// RecordEngineRequest records metrics for an engine request
func (mc *MetricsCollector) RecordEngineRequest(engineID, engineName, status string, duration time.Duration) {
	mc.EngineRequestsTotal.WithLabelValues(engineID, engineName, status).Inc()
	mc.EngineRequestDuration.WithLabelValues(engineID, engineName).Observe(duration.Seconds())
}

// RecordEngineHealth records engine health status
func (mc *MetricsCollector) RecordEngineHealth(engineID, engineName string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	mc.EngineHealthStatus.WithLabelValues(engineID, engineName).Set(value)
}

// RecordEngineError records an engine error
func (mc *MetricsCollector) RecordEngineError(engineID, engineName, errorType string) {
	mc.EngineErrors.WithLabelValues(engineID, engineName, errorType).Inc()
}

// RecordSafetyDecision records a safety decision
func (mc *MetricsCollector) RecordSafetyDecision(status, tier, patientID string, riskScore float64) {
	mc.SafetyDecisionsTotal.WithLabelValues(status, tier, patientID).Inc()
	mc.RiskScoreDistribution.WithLabelValues(status, tier).Observe(riskScore)
}

// RecordOverrideToken records the use of an override token
func (mc *MetricsCollector) RecordOverrideToken(clinicianID, reason string) {
	mc.OverrideTokensUsed.WithLabelValues(clinicianID, reason).Inc()
}

// RecordCacheHitRate records cache hit rate
func (mc *MetricsCollector) RecordCacheHitRate(cacheLevel, cacheType string, hitRate float64) {
	mc.CacheHitRate.WithLabelValues(cacheLevel, cacheType).Set(hitRate)
}

// RecordCircuitBreakerStatus records circuit breaker status
func (mc *MetricsCollector) RecordCircuitBreakerStatus(service, endpoint string, status int) {
	mc.CircuitBreakerStatus.WithLabelValues(service, endpoint).Set(float64(status))
}

// RecordActiveConnections records the number of active connections
func (mc *MetricsCollector) RecordActiveConnections(count int) {
	mc.ActiveConnections.Set(float64(count))
}

// RecordCAERequest records metrics for a CAE request
func (mc *MetricsCollector) RecordCAERequest(method, status string, duration time.Duration) {
	mc.CAERequestsTotal.WithLabelValues(method, status).Inc()
	mc.CAERequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// RecordCAEConnectionStatus records CAE connection status
func (mc *MetricsCollector) RecordCAEConnectionStatus(connected bool) {
	value := 0.0
	if connected {
		value = 1.0
	}
	mc.CAEConnectionStatus.Set(value)
}

// StartMetricsServer starts the Prometheus metrics HTTP server
func (mc *MetricsCollector) StartMetricsServer(port string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(mc.registry, promhttp.HandlerOpts{}))
	
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	
	mc.logger.Info("Starting metrics server", zap.String("port", port))
	
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			mc.logger.Error("Metrics server failed", zap.Error(err))
		}
	}()
	
	return nil
}

// GetRegistry returns the Prometheus registry for custom metrics
func (mc *MetricsCollector) GetRegistry() *prometheus.Registry {
	return mc.registry
}
