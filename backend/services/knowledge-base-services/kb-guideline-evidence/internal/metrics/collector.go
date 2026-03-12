package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector holds all metrics for KB-3 service
type Collector struct {
	// HTTP metrics
	RequestsTotal     *prometheus.CounterVec
	RequestDuration   *prometheus.HistogramVec
	ResponseSize      *prometheus.HistogramVec
	
	// Business metrics
	GuidelinesTotal   prometheus.Gauge
	RecommendationsTotal prometheus.Gauge
	ActiveGuidelines  prometheus.Gauge
	
	// Cache metrics
	CacheHits         *prometheus.CounterVec
	CacheMisses       *prometheus.CounterVec
	CacheOperations   *prometheus.CounterVec
	
	// Database metrics
	DatabaseQueries   *prometheus.CounterVec
	DatabaseErrors    *prometheus.CounterVec
	DatabaseDuration  *prometheus.HistogramVec
	
	// Cross-KB validation metrics
	CrossKBValidations *prometheus.CounterVec
	CrossKBErrors     *prometheus.CounterVec
	CrossKBDuration   *prometheus.HistogramVec
	
	// Regional metrics
	RegionalRequests  *prometheus.CounterVec
	RegionalGuidelines *prometheus.GaugeVec
	
	// Search metrics
	SearchRequests    *prometheus.CounterVec
	SearchDuration    *prometheus.HistogramVec
	SearchResults     *prometheus.HistogramVec
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		// HTTP metrics
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb3",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb3",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
			},
			[]string{"method", "endpoint"},
		),
		
		ResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb3",
				Name:      "response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 6), // 100B to 100MB
			},
			[]string{"endpoint"},
		),
		
		// Business metrics
		GuidelinesTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "kb3",
				Name:      "guidelines_total",
				Help:      "Total number of guidelines in the system",
			},
		),
		
		RecommendationsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "kb3",
				Name:      "recommendations_total",
				Help:      "Total number of recommendations in the system",
			},
		),
		
		ActiveGuidelines: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "kb3",
				Name:      "active_guidelines",
				Help:      "Number of currently active guidelines",
			},
		),
		
		// Cache metrics
		CacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb3",
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache_type"},
		),
		
		CacheMisses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb3",
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache_type"},
		),
		
		CacheOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb3",
				Name:      "cache_operations_total",
				Help:      "Total number of cache operations",
			},
			[]string{"operation", "cache_type"},
		),
		
		// Database metrics
		DatabaseQueries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb3",
				Name:      "database_queries_total",
				Help:      "Total number of database queries",
			},
			[]string{"operation", "table"},
		),
		
		DatabaseErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb3",
				Name:      "database_errors_total",
				Help:      "Total number of database errors",
			},
			[]string{"operation", "error_type"},
		),
		
		DatabaseDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb3",
				Name:      "database_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
			},
			[]string{"operation", "table"},
		),
		
		// Cross-KB validation metrics
		CrossKBValidations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb3",
				Name:      "cross_kb_validations_total",
				Help:      "Total number of cross-KB validations",
			},
			[]string{"kb_name", "status"},
		),
		
		CrossKBErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb3",
				Name:      "cross_kb_errors_total",
				Help:      "Total number of cross-KB validation errors",
			},
			[]string{"kb_name", "error_type"},
		),
		
		CrossKBDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb3",
				Name:      "cross_kb_duration_seconds",
				Help:      "Cross-KB validation duration in seconds",
				Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 5.0},
			},
			[]string{"kb_name"},
		),
		
		// Regional metrics
		RegionalRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb3",
				Name:      "regional_requests_total",
				Help:      "Total number of requests by region",
			},
			[]string{"region"},
		),
		
		RegionalGuidelines: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "kb3",
				Name:      "regional_guidelines",
				Help:      "Number of guidelines by region",
			},
			[]string{"region", "organization"},
		),
		
		// Search metrics
		SearchRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb3",
				Name:      "search_requests_total",
				Help:      "Total number of search requests",
			},
			[]string{"search_type"},
		),
		
		SearchDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb3",
				Name:      "search_duration_seconds",
				Help:      "Search request duration in seconds",
				Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0},
			},
			[]string{"search_type"},
		),
		
		SearchResults: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb3",
				Name:      "search_results",
				Help:      "Number of search results returned",
				Buckets:   []float64{0, 1, 5, 10, 25, 50, 100, 250},
			},
			[]string{"search_type"},
		),
	}
}

// HTTP Metrics Methods

func (c *Collector) RecordRequest(method, endpoint string, statusCode int, duration time.Duration) {
	c.RequestsTotal.WithLabelValues(method, endpoint, strconv.Itoa(statusCode)).Inc()
	c.RequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

func (c *Collector) RecordResponseSize(endpoint string, size int) {
	c.ResponseSize.WithLabelValues(endpoint).Observe(float64(size))
}

// Business Metrics Methods

func (c *Collector) SetGuidelinesTotal(count float64) {
	c.GuidelinesTotal.Set(count)
}

func (c *Collector) SetRecommendationsTotal(count float64) {
	c.RecommendationsTotal.Set(count)
}

func (c *Collector) SetActiveGuidelines(count float64) {
	c.ActiveGuidelines.Set(count)
}

// Cache Metrics Methods

func (c *Collector) RecordCacheHit(cacheType string) {
	c.CacheHits.WithLabelValues(cacheType).Inc()
}

func (c *Collector) RecordCacheMiss(cacheType string) {
	c.CacheMisses.WithLabelValues(cacheType).Inc()
}

func (c *Collector) RecordCacheOperation(operation, cacheType string) {
	c.CacheOperations.WithLabelValues(operation, cacheType).Inc()
}

// Database Metrics Methods

func (c *Collector) RecordDatabaseQuery(operation, table string, duration time.Duration) {
	c.DatabaseQueries.WithLabelValues(operation, table).Inc()
	c.DatabaseDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

func (c *Collector) RecordDatabaseError(operation, errorType string) {
	c.DatabaseErrors.WithLabelValues(operation, errorType).Inc()
}

// Cross-KB Validation Metrics Methods

func (c *Collector) RecordCrossKBValidation(kbName, status string, duration time.Duration) {
	c.CrossKBValidations.WithLabelValues(kbName, status).Inc()
	c.CrossKBDuration.WithLabelValues(kbName).Observe(duration.Seconds())
}

func (c *Collector) RecordCrossKBError(kbName, errorType string) {
	c.CrossKBErrors.WithLabelValues(kbName, errorType).Inc()
}

// Regional Metrics Methods

func (c *Collector) RecordRegionalRequest(region string) {
	c.RegionalRequests.WithLabelValues(region).Inc()
}

func (c *Collector) SetRegionalGuidelines(region, organization string, count float64) {
	c.RegionalGuidelines.WithLabelValues(region, organization).Set(count)
}

// Search Metrics Methods

func (c *Collector) RecordSearchRequest(searchType string, duration time.Duration, resultCount int) {
	c.SearchRequests.WithLabelValues(searchType).Inc()
	c.SearchDuration.WithLabelValues(searchType).Observe(duration.Seconds())
	c.SearchResults.WithLabelValues(searchType).Observe(float64(resultCount))
}

// Utility Methods

// GetCacheHitRate calculates cache hit rate for a specific cache type
func (c *Collector) GetCacheHitRate(cacheType string) float64 {
	hitMetric := c.CacheHits.WithLabelValues(cacheType)
	missMetric := c.CacheMisses.WithLabelValues(cacheType)
	
	// Get metric values (this is a simplified example)
	// In practice, you'd need to use a metric gatherer
	hits := getCounterValue(hitMetric)
	misses := getCounterValue(missMetric)
	
	total := hits + misses
	if total == 0 {
		return 0
	}
	
	return hits / total
}

// Helper function to extract counter value (simplified)
func getCounterValue(counter prometheus.Counter) float64 {
	dto := &prometheus.Metric{}
	if err := counter.Write(dto); err != nil {
		return 0
	}
	return dto.GetCounter().GetValue()
}

// Middleware function for automatic HTTP metrics collection
func (c *Collector) HTTPMiddleware() func(next func()) func() {
	return func(next func()) func() {
		return func() {
			start := time.Now()
			
			// Execute the handler
			next()
			
			// Record metrics after handler completes
			duration := time.Since(start)
			// Note: In real implementation, you'd extract method, endpoint, and status from context
			c.RecordRequest("GET", "/api/v1/guidelines", 200, duration)
		}
	}
}

// Background metrics updater for business metrics
func (c *Collector) UpdateBusinessMetrics(
	totalGuidelines int,
	totalRecommendations int,
	activeGuidelines int,
) {
	c.SetGuidelinesTotal(float64(totalGuidelines))
	c.SetRecommendationsTotal(float64(totalRecommendations))
	c.SetActiveGuidelines(float64(activeGuidelines))
}

// Regional metrics updater
func (c *Collector) UpdateRegionalMetrics(regionalCounts map[string]map[string]int) {
	for region, orgCounts := range regionalCounts {
		for organization, count := range orgCounts {
			c.SetRegionalGuidelines(region, organization, float64(count))
		}
	}
}

// Timer utility for measuring durations
type Timer struct {
	start time.Time
}

func StartTimer() *Timer {
	return &Timer{start: time.Now()}
}

func (t *Timer) Observe(histogram prometheus.Observer) {
	histogram.Observe(time.Since(t.start).Seconds())
}

func (t *Timer) ObserveWithLabels(histogram *prometheus.HistogramVec, labels ...string) {
	histogram.WithLabelValues(labels...).Observe(time.Since(t.start).Seconds())
}

func (t *Timer) Duration() time.Duration {
	return time.Since(t.start)
}