package services

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// prometheusMetricsService implements MetricsService using Prometheus
type prometheusMetricsService struct {
	registry *prometheus.Registry

	// Flow 2 metrics
	flow2ExecutionDuration    *prometheus.HistogramVec
	flow2ExecutionTotal       *prometheus.CounterVec
	medicationIntelligence    *prometheus.HistogramVec
	doseOptimization         *prometheus.HistogramVec
	safetyValidation         *prometheus.HistogramVec
	flow2Errors              prometheus.Counter

	// HTTP metrics
	httpRequestDuration *prometheus.HistogramVec
	httpRequestTotal    *prometheus.CounterVec

	// Cache metrics
	cacheHits   *prometheus.CounterVec
	cacheMisses *prometheus.CounterVec

	// Rust engine metrics
	rustEngineFailures prometheus.Counter
	rustEngineLatency  prometheus.Histogram
}

// NewMetricsService creates a new metrics service
func NewMetricsService() MetricsService {
	registry := prometheus.NewRegistry()

	service := &prometheusMetricsService{
		registry: registry,

		// Flow 2 metrics
		flow2ExecutionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "flow2_execution_duration_seconds",
				Help:    "Duration of Flow 2 execution",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"status", "recipes_executed"},
		),

		flow2ExecutionTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "flow2_execution_total",
				Help: "Total number of Flow 2 executions",
			},
			[]string{"status"},
		),

		medicationIntelligence: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "medication_intelligence_duration_seconds",
				Help:    "Duration of medication intelligence execution",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"intelligence_score_range"},
		),

		doseOptimization: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "dose_optimization_duration_seconds",
				Help:    "Duration of dose optimization execution",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"optimization_score_range"},
		),

		safetyValidation: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "safety_validation_duration_seconds",
				Help:    "Duration of safety validation execution",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"safety_status"},
		),

		flow2Errors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "flow2_errors_total",
				Help: "Total number of Flow 2 errors",
			},
		),

		// HTTP metrics
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status_code"},
		),

		httpRequestTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status_code"},
		),

		// Cache metrics
		cacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_level"},
		),

		cacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_level"},
		),

		// Rust engine metrics
		rustEngineFailures: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "rust_engine_failures_total",
				Help: "Total number of Rust engine failures",
			},
		),

		rustEngineLatency: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "rust_engine_latency_seconds",
				Help:    "Latency of Rust engine calls",
				Buckets: prometheus.DefBuckets,
			},
		),
	}

	// Register all metrics
	registry.MustRegister(
		service.flow2ExecutionDuration,
		service.flow2ExecutionTotal,
		service.medicationIntelligence,
		service.doseOptimization,
		service.safetyValidation,
		service.flow2Errors,
		service.httpRequestDuration,
		service.httpRequestTotal,
		service.cacheHits,
		service.cacheMisses,
		service.rustEngineFailures,
		service.rustEngineLatency,
	)

	return service
}

// RecordFlow2Execution records Flow 2 execution metrics
func (p *prometheusMetricsService) RecordFlow2Execution(duration time.Duration, status string, recipesExecuted int) {
	p.flow2ExecutionDuration.WithLabelValues(status, strconv.Itoa(recipesExecuted)).Observe(duration.Seconds())
	p.flow2ExecutionTotal.WithLabelValues(status).Inc()
}

// RecordMedicationIntelligence records medication intelligence metrics
func (p *prometheusMetricsService) RecordMedicationIntelligence(duration time.Duration, intelligenceScore float64) {
	scoreRange := getScoreRange(intelligenceScore)
	p.medicationIntelligence.WithLabelValues(scoreRange).Observe(duration.Seconds())
}

// RecordDoseOptimization records dose optimization metrics
func (p *prometheusMetricsService) RecordDoseOptimization(duration time.Duration, optimizationScore float64) {
	scoreRange := getScoreRange(optimizationScore)
	p.doseOptimization.WithLabelValues(scoreRange).Observe(duration.Seconds())
}

// RecordSafetyValidation records safety validation metrics
func (p *prometheusMetricsService) RecordSafetyValidation(duration time.Duration, safetyStatus string) {
	p.safetyValidation.WithLabelValues(safetyStatus).Observe(duration.Seconds())
}

// IncrementFlow2Errors increments Flow 2 error counter
func (p *prometheusMetricsService) IncrementFlow2Errors() {
	p.flow2Errors.Inc()
}

// RecordHTTPRequest records HTTP request metrics
func (p *prometheusMetricsService) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	statusCodeStr := strconv.Itoa(statusCode)
	p.httpRequestDuration.WithLabelValues(method, path, statusCodeStr).Observe(duration.Seconds())
	p.httpRequestTotal.WithLabelValues(method, path, statusCodeStr).Inc()
}

// IncrementCacheHits increments cache hit counter
func (p *prometheusMetricsService) IncrementCacheHits(cacheLevel string) {
	p.cacheHits.WithLabelValues(cacheLevel).Inc()
}

// IncrementCacheMisses increments cache miss counter
func (p *prometheusMetricsService) IncrementCacheMisses(cacheLevel string) {
	p.cacheMisses.WithLabelValues(cacheLevel).Inc()
}

// IncrementRustEngineFailures increments Rust engine failure counter
func (p *prometheusMetricsService) IncrementRustEngineFailures() {
	p.rustEngineFailures.Inc()
}

// RecordRustEngineLatency records Rust engine latency
func (p *prometheusMetricsService) RecordRustEngineLatency(duration time.Duration) {
	p.rustEngineLatency.Observe(duration.Seconds())
}

// PrometheusHandler returns the Prometheus metrics handler
func (p *prometheusMetricsService) PrometheusHandler(c *gin.Context) {
	promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{}).ServeHTTP(c.Writer, c.Request)
}

// GetRegistry returns the Prometheus registry
func (p *prometheusMetricsService) GetRegistry() *prometheus.Registry {
	return p.registry
}

// Helper function to categorize scores
func getScoreRange(score float64) string {
	switch {
	case score >= 0.9:
		return "excellent"
	case score >= 0.8:
		return "good"
	case score >= 0.7:
		return "fair"
	case score >= 0.6:
		return "poor"
	default:
		return "very_poor"
	}
}
