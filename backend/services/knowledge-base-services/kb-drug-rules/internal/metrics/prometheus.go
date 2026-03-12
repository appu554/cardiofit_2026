package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector defines the metrics collection interface
type Collector interface {
	// Core methods
	IncrementCounter(name string, labels map[string]string)
	RecordHistogram(name string, value float64, labels map[string]string)
	RecordGauge(name string, value float64, labels map[string]string)
	RecordHTTPRequest(method, path string, statusCode int, duration time.Duration)
	Handler() func(http.ResponseWriter, *http.Request)
	// KB-1 enhanced methods
	IncrCounter(name string, value float64, labels map[string]string)
	RecordDuration(name string, duration time.Duration, labels map[string]string)
	GetMetrics() map[string]interface{}
}

// PrometheusCollector implements metrics collection using Prometheus
type PrometheusCollector struct {
	registry *prometheus.Registry
	
	// HTTP metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	
	// Business metrics
	drugRulesRequests    *prometheus.CounterVec
	validationRequests   *prometheus.CounterVec
	validationSuccess    prometheus.Counter
	validationFailure    prometheus.Counter
	hotloadRequests      *prometheus.CounterVec
	hotloadSuccess       *prometheus.CounterVec
	
	// Cache metrics
	cacheHits            *prometheus.CounterVec
	cacheMisses          *prometheus.CounterVec
	
	// Signature metrics
	signatureValidations prometheus.Counter
	signatureFailures    prometheus.Counter
	
	// Governance metrics
	governanceApprovals  prometheus.Counter
	governanceRejections prometheus.Counter
	
	// System metrics
	activeVersions       prometheus.Gauge
	databaseConnections  prometheus.Gauge
}

// NewCollector creates a new Prometheus metrics collector
func NewCollector() Collector {
	registry := prometheus.NewRegistry()
	
	collector := &PrometheusCollector{
		registry: registry,
		
		// HTTP metrics
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		
		// Business metrics
		drugRulesRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_drug_rules_requests_total",
				Help: "Total number of drug rules requests",
			},
			[]string{"drug_id", "region"},
		),
		
		validationRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_validation_requests_total",
				Help: "Total number of validation requests",
			},
			[]string{"type"},
		),
		
		validationSuccess: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "kb_validation_success_total",
				Help: "Total number of successful validations",
			},
		),
		
		validationFailure: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "kb_validation_failure_total",
				Help: "Total number of failed validations",
			},
		),
		
		hotloadRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_hotload_requests_total",
				Help: "Total number of hotload requests",
			},
			[]string{"drug_id"},
		),
		
		hotloadSuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_hotload_success_total",
				Help: "Total number of successful hotloads",
			},
			[]string{"drug_id"},
		),
		
		// Cache metrics
		cacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"type"},
		),
		
		cacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"type"},
		),
		
		// Signature metrics
		signatureValidations: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "kb_signature_validations_total",
				Help: "Total number of signature validations",
			},
		),
		
		signatureFailures: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "kb_signature_failures_total",
				Help: "Total number of signature validation failures",
			},
		),
		
		// Governance metrics
		governanceApprovals: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "kb_governance_approvals_total",
				Help: "Total number of governance approvals",
			},
		),
		
		governanceRejections: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "kb_governance_rejections_total",
				Help: "Total number of governance rejections",
			},
		),
		
		// System metrics
		activeVersions: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kb_active_versions",
				Help: "Number of active rule versions",
			},
		),
		
		databaseConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kb_database_connections",
				Help: "Number of active database connections",
			},
		),
	}
	
	// Register all metrics
	registry.MustRegister(
		collector.httpRequestsTotal,
		collector.httpRequestDuration,
		collector.drugRulesRequests,
		collector.validationRequests,
		collector.validationSuccess,
		collector.validationFailure,
		collector.hotloadRequests,
		collector.hotloadSuccess,
		collector.cacheHits,
		collector.cacheMisses,
		collector.signatureValidations,
		collector.signatureFailures,
		collector.governanceApprovals,
		collector.governanceRejections,
		collector.activeVersions,
		collector.databaseConnections,
	)
	
	return collector
}

// IncrementCounter increments a counter metric
func (p *PrometheusCollector) IncrementCounter(name string, labels map[string]string) {
	switch name {
	case "drug_rules_requests_total":
		p.drugRulesRequests.With(prometheus.Labels(labels)).Inc()
	case "validation_requests_total":
		p.validationRequests.With(prometheus.Labels(labels)).Inc()
	case "validation_success_total":
		p.validationSuccess.Inc()
	case "validation_failure_total":
		p.validationFailure.Inc()
	case "hotload_requests_total":
		p.hotloadRequests.With(prometheus.Labels(labels)).Inc()
	case "hotload_success_total":
		p.hotloadSuccess.With(prometheus.Labels(labels)).Inc()
	case "cache_hits_total":
		p.cacheHits.With(prometheus.Labels(labels)).Inc()
	case "cache_misses_total":
		p.cacheMisses.With(prometheus.Labels(labels)).Inc()
	case "signature_validations_total":
		p.signatureValidations.Inc()
	case "signature_failures_total":
		p.signatureFailures.Inc()
	case "governance_approvals_total":
		p.governanceApprovals.Inc()
	case "governance_rejections_total":
		p.governanceRejections.Inc()
	}
}

// RecordHistogram records a histogram metric
func (p *PrometheusCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	// Add histogram metrics as needed
}

// RecordGauge records a gauge metric
func (p *PrometheusCollector) RecordGauge(name string, value float64, labels map[string]string) {
	switch name {
	case "active_versions":
		p.activeVersions.Set(value)
	case "database_connections":
		p.databaseConnections.Set(value)
	}
}

// RecordHTTPRequest records HTTP request metrics
func (p *PrometheusCollector) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)

	p.httpRequestsTotal.WithLabelValues(method, path, status).Inc()
	p.httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// Handler returns the Prometheus metrics HTTP handler
func (p *PrometheusCollector) Handler() func(http.ResponseWriter, *http.Request) {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{}).ServeHTTP
}

// IncrCounter increments a counter by a specific value (KB-1 enhanced method)
func (p *PrometheusCollector) IncrCounter(name string, value float64, labels map[string]string) {
	switch name {
	case "drug_rules_requests_total":
		p.drugRulesRequests.With(prometheus.Labels(labels)).Add(value)
	case "validation_requests_total":
		p.validationRequests.With(prometheus.Labels(labels)).Add(value)
	case "hotload_requests_total":
		p.hotloadRequests.With(prometheus.Labels(labels)).Add(value)
	case "hotload_success_total":
		p.hotloadSuccess.With(prometheus.Labels(labels)).Add(value)
	case "cache_hits_total":
		p.cacheHits.With(prometheus.Labels(labels)).Add(value)
	case "cache_misses_total":
		p.cacheMisses.With(prometheus.Labels(labels)).Add(value)
	}
}

// RecordDuration records a duration metric (KB-1 enhanced method)
func (p *PrometheusCollector) RecordDuration(name string, duration time.Duration, labels map[string]string) {
	switch name {
	case "http_request_duration":
		if method, ok := labels["method"]; ok {
			if path, ok := labels["path"]; ok {
				p.httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
			}
		}
	}
}

// GetMetrics returns all current metric values (KB-1 enhanced method for health endpoints)
func (p *PrometheusCollector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"collector_type": "prometheus",
		"registry":       "active",
		"metrics_count":  16, // Number of registered metrics
	}
}

// NoOpCollector is a no-op implementation for testing
type NoOpCollector struct{}

// NewNoOpCollector creates a new no-op metrics collector
func NewNoOpCollector() Collector {
	return &NoOpCollector{}
}

func (n *NoOpCollector) IncrementCounter(name string, labels map[string]string)                        {}
func (n *NoOpCollector) RecordHistogram(name string, value float64, labels map[string]string)         {}
func (n *NoOpCollector) RecordGauge(name string, value float64, labels map[string]string)             {}
func (n *NoOpCollector) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {}
func (n *NoOpCollector) Handler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# No metrics available\n"))
	}
}

// KB-1 enhanced methods for NoOpCollector
func (n *NoOpCollector) IncrCounter(name string, value float64, labels map[string]string)         {}
func (n *NoOpCollector) RecordDuration(name string, duration time.Duration, labels map[string]string) {}
func (n *NoOpCollector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"collector_type": "noop",
		"status":         "disabled",
	}
}

// TOMLMetrics provides TOML-specific metrics collection
type TOMLMetrics struct {
	collector      Collector
	activeRequests int64
}

// NewTOMLMetrics creates a new TOML metrics instance
func NewTOMLMetrics(collector Collector) *TOMLMetrics {
	return &TOMLMetrics{
		collector:      collector,
		activeRequests: 0,
	}
}

// RecordError records a TOML-related error
func (m *TOMLMetrics) RecordError(errorType, method string) {
	m.collector.IncrementCounter("toml_errors_total", map[string]string{
		"error_type": errorType,
		"method":     method,
	})
}

// IncrementActiveRequests increments active request counter
func (m *TOMLMetrics) IncrementActiveRequests() {
	m.activeRequests++
	m.collector.RecordGauge("toml_active_requests", float64(m.activeRequests), nil)
}

// DecrementActiveRequests decrements active request counter
func (m *TOMLMetrics) DecrementActiveRequests() {
	m.activeRequests--
	if m.activeRequests < 0 {
		m.activeRequests = 0
	}
	m.collector.RecordGauge("toml_active_requests", float64(m.activeRequests), nil)
}

// RecordResponseTime records response time for TOML operations
func (m *TOMLMetrics) RecordResponseTime(duration time.Duration) {
	m.collector.RecordHistogram("toml_response_time_seconds", duration.Seconds(), nil)
}

// RecordValidation records TOML validation metrics
func (m *TOMLMetrics) RecordValidation(duration time.Duration, success bool, score float64) {
	status := "failure"
	if success {
		status = "success"
	}
	m.collector.IncrementCounter("toml_validations_total", map[string]string{
		"status": status,
	})
	m.collector.RecordHistogram("toml_validation_duration_seconds", duration.Seconds(), nil)
}

// RecordConversion records TOML conversion metrics
func (m *TOMLMetrics) RecordConversion(duration time.Duration, success bool, inputSize, outputSize int) {
	status := "failure"
	if success {
		status = "success"
	}
	m.collector.IncrementCounter("toml_conversions_total", map[string]string{
		"status": status,
	})
	m.collector.RecordHistogram("toml_conversion_duration_seconds", duration.Seconds(), nil)
}

// RecordHotload records TOML hotload metrics
func (m *TOMLMetrics) RecordHotload(duration time.Duration, success bool) {
	status := "failure"
	if success {
		status = "success"
	}
	m.collector.IncrementCounter("toml_hotloads_total", map[string]string{
		"status": status,
	})
	m.collector.RecordHistogram("toml_hotload_duration_seconds", duration.Seconds(), nil)
}

// RecordBatchLoad records TOML batch load metrics
func (m *TOMLMetrics) RecordBatchLoad(duration time.Duration, success bool, itemCount int) {
	status := "failure"
	if success {
		status = "success"
	}
	m.collector.IncrementCounter("toml_batch_loads_total", map[string]string{
		"status": status,
	})
	m.collector.RecordHistogram("toml_batch_load_duration_seconds", duration.Seconds(), nil)
	m.collector.RecordGauge("toml_batch_load_items", float64(itemCount), nil)
}
