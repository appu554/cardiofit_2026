package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Collector struct {
	// Request metrics
	requestsTotal     *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	requestsInFlight  prometheus.Gauge

	// Terminology specific metrics
	conceptLookupsTotal    *prometheus.CounterVec
	conceptLookupDuration  *prometheus.HistogramVec
	searchRequestsTotal    *prometheus.CounterVec
	searchDuration         *prometheus.HistogramVec
	validationRequestsTotal *prometheus.CounterVec
	validationDuration     *prometheus.HistogramVec

	// Cache metrics
	cacheHitsTotal   *prometheus.CounterVec
	cacheMissesTotal *prometheus.CounterVec

	// Database metrics
	dbConnectionsActive prometheus.Gauge
	dbQueriesTotal     *prometheus.CounterVec
	dbQueryDuration    *prometheus.HistogramVec

	// System metrics
	terminologySystemsTotal prometheus.Gauge
	conceptsTotal           prometheus.Gauge
	valueSetsTotal          prometheus.Gauge
	mappingsTotal           prometheus.Gauge
}

func NewCollector(namespace string) *Collector {
	return &Collector{
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "endpoint"},
		),
		requestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "requests_in_flight",
				Help:      "Number of HTTP requests currently being processed",
			},
		),

		conceptLookupsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "concept_lookups_total",
				Help:      "Total number of concept lookups",
			},
			[]string{"system", "status"},
		),
		conceptLookupDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "concept_lookup_duration_seconds",
				Help:      "Concept lookup duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"system"},
		),
		searchRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "search_requests_total",
				Help:      "Total number of terminology search requests",
			},
			[]string{"system", "status"},
		),
		searchDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "search_duration_seconds",
				Help:      "Terminology search duration in seconds",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"system"},
		),
		validationRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "validation_requests_total",
				Help:      "Total number of terminology validation requests",
			},
			[]string{"system", "status"},
		),
		validationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "validation_duration_seconds",
				Help:      "Terminology validation duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25},
			},
			[]string{"system"},
		),

		cacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"operation", "key_type"},
		),
		cacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"operation", "key_type"},
		),

		dbConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "database_connections_active",
				Help:      "Number of active database connections",
			},
		),
		dbQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "database_queries_total",
				Help:      "Total number of database queries",
			},
			[]string{"operation", "status"},
		),
		dbQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "database_query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"operation"},
		),

		terminologySystemsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "terminology_systems_total",
				Help:      "Total number of terminology systems",
			},
		),
		conceptsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "concepts_total",
				Help:      "Total number of concepts",
			},
		),
		valueSetsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "value_sets_total",
				Help:      "Total number of value sets",
			},
		),
		mappingsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "mappings_total",
				Help:      "Total number of concept mappings",
			},
		),
	}
}

// Request metrics
func (c *Collector) RecordRequest(method, endpoint, statusCode string, duration time.Duration) {
	c.requestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
	c.requestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

func (c *Collector) IncRequestsInFlight() {
	c.requestsInFlight.Inc()
}

func (c *Collector) DecRequestsInFlight() {
	c.requestsInFlight.Dec()
}

// Concept lookup metrics
func (c *Collector) RecordConceptLookup(system, status string, duration time.Duration) {
	c.conceptLookupsTotal.WithLabelValues(system, status).Inc()
	c.conceptLookupDuration.WithLabelValues(system).Observe(duration.Seconds())
}

// Search metrics
func (c *Collector) RecordSearch(system, status string, duration time.Duration) {
	c.searchRequestsTotal.WithLabelValues(system, status).Inc()
	c.searchDuration.WithLabelValues(system).Observe(duration.Seconds())
}

// Validation metrics
func (c *Collector) RecordValidation(system, status string, duration time.Duration) {
	c.validationRequestsTotal.WithLabelValues(system, status).Inc()
	c.validationDuration.WithLabelValues(system).Observe(duration.Seconds())
}

// Translation metrics 
func (c *Collector) RecordTranslation(sourceSystem, targetSystem, status string, duration time.Duration) {
	// For now, use validation metrics as a placeholder
	// In production, you'd want dedicated translation metrics
	c.validationRequestsTotal.WithLabelValues(sourceSystem+"->"+targetSystem, status).Inc()
	c.validationDuration.WithLabelValues(sourceSystem+"->"+targetSystem).Observe(duration.Seconds())
}

// Cache metrics
func (c *Collector) RecordCacheHit(operation, keyType string) {
	c.cacheHitsTotal.WithLabelValues(operation, keyType).Inc()
}

func (c *Collector) RecordCacheMiss(operation, keyType string) {
	c.cacheMissesTotal.WithLabelValues(operation, keyType).Inc()
}

// Database metrics
func (c *Collector) SetActiveConnections(count float64) {
	c.dbConnectionsActive.Set(count)
}

func (c *Collector) RecordDBQuery(operation, status string, duration time.Duration) {
	c.dbQueriesTotal.WithLabelValues(operation, status).Inc()
	c.dbQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// System metrics
func (c *Collector) SetTerminologySystemsCount(count float64) {
	c.terminologySystemsTotal.Set(count)
}

func (c *Collector) SetConceptsCount(count float64) {
	c.conceptsTotal.Set(count)
}

func (c *Collector) SetValueSetsCount(count float64) {
	c.valueSetsTotal.Set(count)
}

func (c *Collector) SetMappingsCount(count float64) {
	c.mappingsTotal.Set(count)
}

// Batch operation metrics
func (c *Collector) RecordBatchOperation(operation, status string, duration time.Duration, batchSize int) {
	c.dbQueriesTotal.WithLabelValues(operation, status).Add(float64(batchSize))
	c.dbQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// Expansion metrics
func (c *Collector) RecordExpansion(system, status string, duration time.Duration, resultCount int) {
	c.searchRequestsTotal.WithLabelValues(system, status).Inc()
	c.searchDuration.WithLabelValues(system).Observe(duration.Seconds())
}

// Autocomplete metrics
func (c *Collector) RecordAutocompleteMetric(metricType, value string) {
	c.searchRequestsTotal.WithLabelValues("autocomplete_"+metricType, value).Inc()
}

// Generic counter with labels
func (c *Collector) IncrementCounterWithLabels(counterName string, labels map[string]string) {
	// Use request counter as a generic counter
	method := labels["method"]
	endpoint := labels["endpoint"]
	status := labels["status"]
	c.requestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

// Analysis metrics
func (c *Collector) RecordAnalysisMetric(metricType, value string) {
	c.searchRequestsTotal.WithLabelValues("analysis_"+metricType, value).Inc()
}

// Search metrics
func (c *Collector) RecordSearchMetric(metricType, value string) {
	c.searchRequestsTotal.WithLabelValues("search_"+metricType, value).Inc()
}

// SNOMED validation metrics
func (c *Collector) RecordSNOMEDValidation(code, status string, duration time.Duration) {
	c.validationRequestsTotal.WithLabelValues("SNOMED", status).Inc()
	c.validationDuration.WithLabelValues("SNOMED").Observe(duration.Seconds())
}

// API metrics
func (c *Collector) RecordAPIMetric(metricType, value string) {
	c.requestsTotal.WithLabelValues("api_"+metricType, value, "200").Inc()
}