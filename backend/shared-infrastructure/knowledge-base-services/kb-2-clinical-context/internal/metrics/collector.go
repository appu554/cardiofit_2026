package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	// Request metrics
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	responseSize     *prometheus.HistogramVec

	// Context-specific metrics
	contextBuildsTotal      *prometheus.CounterVec
	phenotypeDetectionsTotal *prometheus.CounterVec
	riskAssessmentsTotal     *prometheus.CounterVec
	careGapsTotal           *prometheus.CounterVec

	// Performance metrics
	contextBuildDuration    *prometheus.HistogramVec
	phenotypeDetectDuration *prometheus.HistogramVec
	riskAssessmentDuration  *prometheus.HistogramVec

	// Cache metrics
	cacheHitsTotal   *prometheus.CounterVec
	cacheMissesTotal *prometheus.CounterVec

	// Database metrics
	mongoOperationsTotal *prometheus.CounterVec
	mongoOperationDuration *prometheus.HistogramVec
}

func NewCollector() *Collector {
	c := &Collector{
		// Request metrics
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_context_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb_context_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		responseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb_context_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"endpoint"},
		),

		// Context-specific metrics
		contextBuildsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_context_builds_total",
				Help: "Total number of context builds",
			},
			[]string{"status"},
		),
		phenotypeDetectionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_context_phenotype_detections_total",
				Help: "Total number of phenotype detections",
			},
			[]string{"phenotype_id"},
		),
		riskAssessmentsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_context_risk_assessments_total",
				Help: "Total number of risk assessments",
			},
			[]string{"risk_type", "status"},
		),
		careGapsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_context_care_gaps_total",
				Help: "Total number of care gaps identified",
			},
			[]string{"gap_type"},
		),

		// Performance metrics
		contextBuildDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb_context_build_duration_seconds",
				Help:    "Context build duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"phenotypes_count"},
		),
		phenotypeDetectDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb_context_phenotype_detect_duration_seconds",
				Help:    "Phenotype detection duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
			},
			[]string{"phenotype_count"},
		),
		riskAssessmentDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb_context_risk_assessment_duration_seconds",
				Help:    "Risk assessment duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
			},
			[]string{"risk_type"},
		),

		// Cache metrics
		cacheHitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_context_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type"},
		),
		cacheMissesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_context_cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type"},
		),

		// Database metrics
		mongoOperationsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb_context_mongo_operations_total",
				Help: "Total number of MongoDB operations",
			},
			[]string{"operation", "collection", "status"},
		),
		mongoOperationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb_context_mongo_operation_duration_seconds",
				Help:    "MongoDB operation duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
			[]string{"operation", "collection"},
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		c.requestsTotal,
		c.requestDuration,
		c.responseSize,
		c.contextBuildsTotal,
		c.phenotypeDetectionsTotal,
		c.riskAssessmentsTotal,
		c.careGapsTotal,
		c.contextBuildDuration,
		c.phenotypeDetectDuration,
		c.riskAssessmentDuration,
		c.cacheHitsTotal,
		c.cacheMissesTotal,
		c.mongoOperationsTotal,
		c.mongoOperationDuration,
	)

	return c
}

// Request metrics
func (c *Collector) RecordRequest(method, endpoint string, statusCode int, duration time.Duration) {
	status := "success"
	if statusCode >= 400 {
		status = "error"
	}

	c.requestsTotal.WithLabelValues(method, endpoint, status).Inc()
	c.requestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

func (c *Collector) RecordResponseSize(endpoint string, size int) {
	c.responseSize.WithLabelValues(endpoint).Observe(float64(size))
}

// Context metrics
func (c *Collector) RecordContextBuild(success bool) {
	status := "success"
	if !success {
		status = "error"
	}
	c.contextBuildsTotal.WithLabelValues(status).Inc()
}

func (c *Collector) RecordContextBuildDuration(phenotypeCount int, duration time.Duration) {
	c.contextBuildDuration.WithLabelValues(string(rune(phenotypeCount))).Observe(duration.Seconds())
}

func (c *Collector) RecordPhenotypeDetection(phenotypeID string) {
	c.phenotypeDetectionsTotal.WithLabelValues(phenotypeID).Inc()
}

func (c *Collector) RecordPhenotypeDetectionDuration(phenotypeCount int, duration time.Duration) {
	c.phenotypeDetectDuration.WithLabelValues(string(rune(phenotypeCount))).Observe(duration.Seconds())
}

func (c *Collector) RecordRiskAssessment(riskType string, success bool) {
	status := "success"
	if !success {
		status = "error"
	}
	c.riskAssessmentsTotal.WithLabelValues(riskType, status).Inc()
}

func (c *Collector) RecordRiskAssessmentDuration(riskType string, duration time.Duration) {
	c.riskAssessmentDuration.WithLabelValues(riskType).Observe(duration.Seconds())
}

func (c *Collector) RecordCareGap(gapType string) {
	c.careGapsTotal.WithLabelValues(gapType).Inc()
}

// Cache metrics
func (c *Collector) RecordCacheHit(cacheType string) {
	c.cacheHitsTotal.WithLabelValues(cacheType).Inc()
}

func (c *Collector) RecordCacheMiss(cacheType string) {
	c.cacheMissesTotal.WithLabelValues(cacheType).Inc()
}

// Database metrics
func (c *Collector) RecordMongoOperation(operation, collection string, success bool, duration time.Duration) {
	status := "success"
	if !success {
		status = "error"
	}

	c.mongoOperationsTotal.WithLabelValues(operation, collection, status).Inc()
	c.mongoOperationDuration.WithLabelValues(operation, collection).Observe(duration.Seconds())
}