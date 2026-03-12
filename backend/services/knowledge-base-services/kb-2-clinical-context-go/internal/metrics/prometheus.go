package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusMetrics holds all Prometheus metrics for KB-2 service
type PrometheusMetrics struct {
	// Request metrics
	RequestsTotal     prometheus.CounterVec
	RequestDuration   prometheus.HistogramVec
	RequestsInFlight  prometheus.Gauge
	
	// Phenotype evaluation metrics
	PhenotypeEvaluationsTotal    prometheus.CounterVec
	PhenotypeEvaluationDuration  prometheus.HistogramVec
	PhenotypeRulesEvaluated      prometheus.CounterVec
	
	// Risk assessment metrics
	RiskAssessmentsTotal    prometheus.CounterVec
	RiskAssessmentDuration  prometheus.HistogramVec
	RiskScoresGenerated     prometheus.CounterVec
	
	// Treatment preference metrics
	TreatmentPreferencesTotal    prometheus.CounterVec
	TreatmentPreferenceDuration  prometheus.HistogramVec
	TreatmentOptionsGenerated    prometheus.CounterVec
	
	// Context assembly metrics
	ContextAssembliesTotal    prometheus.CounterVec
	ContextAssemblyDuration   prometheus.HistogramVec
	
	// Enhanced Cache metrics for 3-tier caching
	CacheHits         prometheus.CounterVec
	CacheMisses       prometheus.CounterVec
	CacheOperations   prometheus.CounterVec
	CacheHitRate      prometheus.GaugeVec     // Real-time hit rate by tier
	CacheMemoryUsage  prometheus.GaugeVec     // Memory usage by tier
	CacheEvictions    prometheus.CounterVec   // Evictions by tier
	CacheLatency      prometheus.HistogramVec // Cache access latency by tier
	CacheSize         prometheus.GaugeVec     // Current cache size by tier
	CacheWarming      prometheus.CounterVec   // Cache warming operations
	
	// Multi-tier cache specific metrics
	CacheTierHits        prometheus.CounterVec // Hits by specific tier (L1/L2/L3)
	CacheTierMisses      prometheus.CounterVec // Misses by specific tier
	CachePromotions      prometheus.CounterVec // Data promotions between tiers
	CacheInvalidations   prometheus.CounterVec // Cache invalidations by pattern
	CacheBatchOperations prometheus.CounterVec // Batch cache operations
	
	// Performance SLA metrics
	SLACompliance        prometheus.GaugeVec   // SLA compliance by metric
	PerformanceScore     prometheus.Gauge     // Overall performance score
	ThroughputActual     prometheus.Gauge     // Actual throughput (RPS)
	LatencyPercentiles   prometheus.GaugeVec  // Latency percentiles (P50, P95, P99)
	
	// CEL engine metrics
	CELEvaluationsTotal    prometheus.CounterVec
	CELEvaluationDuration  prometheus.HistogramVec
	CELErrors             prometheus.CounterVec
	
	// Database metrics
	DatabaseOperationsTotal    prometheus.CounterVec
	DatabaseOperationDuration  prometheus.HistogramVec
	DatabaseConnections        prometheus.Gauge
	
	// Performance metrics
	BatchSizeHistogram      prometheus.Histogram
	SLAViolations          prometheus.CounterVec
	ConcurrentRequests     prometheus.Gauge
	
	// Error metrics
	ErrorsTotal           prometheus.CounterVec
	ValidationErrors      prometheus.CounterVec
	TimeoutErrors         prometheus.CounterVec
}

// NewPrometheusMetrics creates and registers all Prometheus metrics
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		// Request metrics
		RequestsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		RequestDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb2_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.15, 0.2, 0.5, 1.0}, // High-precision buckets for target latencies
			},
			[]string{"method", "endpoint"},
		),
		RequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "kb2_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),
		
		// Phenotype evaluation metrics
		PhenotypeEvaluationsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_phenotype_evaluations_total",
				Help: "Total number of phenotype evaluations",
			},
			[]string{"phenotype_category", "status"},
		),
		PhenotypeEvaluationDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb2_phenotype_evaluation_duration_seconds",
				Help:    "Phenotype evaluation duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.15, 0.2}, // Sub-100ms precision
			},
			[]string{"phenotype_category"},
		),
		PhenotypeRulesEvaluated: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_phenotype_rules_evaluated_total",
				Help: "Total number of phenotype rules evaluated",
			},
			[]string{"rule_type"},
		),
		
		// Risk assessment metrics
		RiskAssessmentsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_risk_assessments_total",
				Help: "Total number of risk assessments",
			},
			[]string{"risk_category", "status"},
		),
		RiskAssessmentDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb2_risk_assessment_duration_seconds",
				Help:    "Risk assessment duration in seconds",
				Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.15, 0.2, 0.3, 0.5}, // Target <200ms
			},
			[]string{"risk_category"},
		),
		RiskScoresGenerated: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_risk_scores_generated_total",
				Help: "Total number of risk scores generated",
			},
			[]string{"score_level"},
		),
		
		// Treatment preference metrics
		TreatmentPreferencesTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_treatment_preferences_total",
				Help: "Total number of treatment preference evaluations",
			},
			[]string{"condition", "status"},
		),
		TreatmentPreferenceDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb2_treatment_preference_duration_seconds",
				Help:    "Treatment preference evaluation duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.15}, // Target <50ms
			},
			[]string{"condition"},
		),
		TreatmentOptionsGenerated: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_treatment_options_generated_total",
				Help: "Total number of treatment options generated",
			},
			[]string{"treatment_category"},
		),
		
		// Context assembly metrics
		ContextAssembliesTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_context_assemblies_total",
				Help: "Total number of context assemblies",
			},
			[]string{"detail_level", "status"},
		),
		ContextAssemblyDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb2_context_assembly_duration_seconds",
				Help:    "Context assembly duration in seconds",
				Buckets: []float64{0.01, 0.05, 0.1, 0.15, 0.2, 0.3, 0.5, 1.0}, // Target <200ms
			},
			[]string{"detail_level"},
		),
		
		// Enhanced Cache metrics for 3-tier caching
		CacheHits: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_cache_hits_total",
				Help: "Total number of cache hits by tier",
			},
			[]string{"tier"}, // l1, l2, l3, combined
		),
		CacheMisses: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_cache_misses_total",
				Help: "Total number of cache misses by tier",
			},
			[]string{"tier"},
		),
		CacheOperations: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_cache_operations_total",
				Help: "Total number of cache operations",
			},
			[]string{"operation", "tier"}, // get, set, delete, invalidate
		),
		CacheHitRate: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kb2_cache_hit_rate",
				Help: "Current cache hit rate by tier (0.0 to 1.0)",
			},
			[]string{"tier"},
		),
		CacheMemoryUsage: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kb2_cache_memory_usage_bytes",
				Help: "Current cache memory usage in bytes by tier",
			},
			[]string{"tier"},
		),
		CacheEvictions: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_cache_evictions_total",
				Help: "Total number of cache evictions by tier",
			},
			[]string{"tier", "reason"}, // size, ttl, manual
		),
		CacheLatency: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb2_cache_access_duration_seconds",
				Help:    "Cache access latency by tier",
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05}, // Sub-ms to ms precision
			},
			[]string{"tier", "operation"},
		),
		CacheSize: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kb2_cache_size_items",
				Help: "Current number of items in cache by tier",
			},
			[]string{"tier"},
		),
		CacheWarming: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_cache_warming_operations_total",
				Help: "Total number of cache warming operations",
			},
			[]string{"strategy", "status"}, // strategy: phenotype_definitions, frequent_patients, etc.
		),
		
		// Multi-tier cache specific metrics
		CacheTierHits: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_cache_tier_hits_total",
				Help: "Cache hits by specific tier with operation type",
			},
			[]string{"tier", "operation_type"}, // phenotype, risk, treatment, context
		),
		CacheTierMisses: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_cache_tier_misses_total",
				Help: "Cache misses by specific tier with operation type",
			},
			[]string{"tier", "operation_type"},
		),
		CachePromotions: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_cache_promotions_total",
				Help: "Cache data promotions between tiers",
			},
			[]string{"from_tier", "to_tier"}, // l3_to_l2, l2_to_l1, etc.
		),
		CacheInvalidations: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_cache_invalidations_total",
				Help: "Cache invalidations by pattern and reason",
			},
			[]string{"pattern_type", "reason"}, // patient_context, phenotype, version_update, manual
		),
		CacheBatchOperations: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_cache_batch_operations_total",
				Help: "Batch cache operations with size buckets",
			},
			[]string{"operation", "size_bucket"}, // get, set | small, medium, large
		),
		
		// Performance SLA metrics
		SLACompliance: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kb2_sla_compliance",
				Help: "SLA compliance status (1.0 = compliant, 0.0 = non-compliant)",
			},
			[]string{"metric_type"}, // latency_p50, latency_p95, latency_p99, throughput, hit_rate_l1, hit_rate_l2
		),
		PerformanceScore: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "kb2_performance_score",
				Help: "Overall performance score (0.0 to 1.0)",
			},
		),
		ThroughputActual: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "kb2_throughput_rps",
				Help: "Current actual throughput in requests per second",
			},
		),
		LatencyPercentiles: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kb2_latency_percentiles_seconds",
				Help: "Latency percentiles for performance monitoring",
			},
			[]string{"percentile"}, // p50, p95, p99
		),
		
		// CEL engine metrics
		CELEvaluationsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_cel_evaluations_total",
				Help: "Total number of CEL evaluations",
			},
			[]string{"expression_type", "status"},
		),
		CELEvaluationDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb2_cel_evaluation_duration_seconds",
				Help:    "CEL evaluation duration in seconds",
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1}, // Sub-ms precision for CEL
			},
			[]string{"expression_type"},
		),
		CELErrors: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_cel_errors_total",
				Help: "Total number of CEL evaluation errors",
			},
			[]string{"error_type"},
		),
		
		// Database metrics
		DatabaseOperationsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_database_operations_total",
				Help: "Total number of database operations",
			},
			[]string{"operation", "collection", "status"},
		),
		DatabaseOperationDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb2_database_operation_duration_seconds",
				Help:    "Database operation duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5}, // DB access timing
			},
			[]string{"operation", "collection"},
		),
		DatabaseConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "kb2_database_connections",
				Help: "Current number of database connections",
			},
		),
		
		// Performance metrics
		BatchSizeHistogram: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "kb2_batch_size",
				Help:    "Size of processing batches",
				Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000}, // Up to 1000 patient batches
			},
		),
		SLAViolations: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_sla_violations_total",
				Help: "Total number of SLA violations",
			},
			[]string{"endpoint", "sla_type"},
		),
		ConcurrentRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "kb2_concurrent_requests",
				Help: "Current number of concurrent requests being processed",
			},
		),
		
		// Error metrics
		ErrorsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_errors_total",
				Help: "Total number of errors",
			},
			[]string{"error_type", "component"},
		),
		ValidationErrors: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_validation_errors_total",
				Help: "Total number of validation errors",
			},
			[]string{"validation_type"},
		),
		TimeoutErrors: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb2_timeout_errors_total",
				Help: "Total number of timeout errors",
			},
			[]string{"operation"},
		),
	}
}

// Timer helper for measuring request duration
func (pm *PrometheusMetrics) StartTimer(method, endpoint string) *prometheus.Timer {
	return prometheus.NewTimer(pm.RequestDuration.WithLabelValues(method, endpoint))
}

// Cache-specific timer for measuring cache access latency
func (pm *PrometheusMetrics) StartCacheTimer(tier, operation string) *prometheus.Timer {
	return prometheus.NewTimer(pm.CacheLatency.WithLabelValues(tier, operation))
}

// RecordRequest records a completed HTTP request
func (pm *PrometheusMetrics) RecordRequest(method, endpoint, status string) {
	pm.RequestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

// RecordPhenotypeEvaluation records a phenotype evaluation
func (pm *PrometheusMetrics) RecordPhenotypeEvaluation(category, status string, duration time.Duration) {
	pm.PhenotypeEvaluationsTotal.WithLabelValues(category, status).Inc()
	pm.PhenotypeEvaluationDuration.WithLabelValues(category).Observe(duration.Seconds())
}

// RecordRiskAssessment records a risk assessment
func (pm *PrometheusMetrics) RecordRiskAssessment(category, status string, duration time.Duration) {
	pm.RiskAssessmentsTotal.WithLabelValues(category, status).Inc()
	pm.RiskAssessmentDuration.WithLabelValues(category).Observe(duration.Seconds())
}

// RecordTreatmentPreference records a treatment preference evaluation
func (pm *PrometheusMetrics) RecordTreatmentPreference(condition, status string, duration time.Duration) {
	pm.TreatmentPreferencesTotal.WithLabelValues(condition, status).Inc()
	pm.TreatmentPreferenceDuration.WithLabelValues(condition).Observe(duration.Seconds())
}

// RecordContextAssembly records a context assembly operation
func (pm *PrometheusMetrics) RecordContextAssembly(detailLevel, status string, duration time.Duration) {
	pm.ContextAssembliesTotal.WithLabelValues(detailLevel, status).Inc()
	pm.ContextAssemblyDuration.WithLabelValues(detailLevel).Observe(duration.Seconds())
}

// Enhanced cache metrics methods

// RecordCacheHit records a cache hit by tier
func (pm *PrometheusMetrics) RecordCacheHit(tier string) {
	pm.CacheHits.WithLabelValues(tier).Inc()
}

// RecordCacheMiss records a cache miss by tier
func (pm *PrometheusMetrics) RecordCacheMiss(tier string) {
	pm.CacheMisses.WithLabelValues(tier).Inc()
}

// RecordCacheOperation records a cache operation with timing
func (pm *PrometheusMetrics) RecordCacheOperation(operation, tier string, duration time.Duration) {
	pm.CacheOperations.WithLabelValues(operation, tier).Inc()
	pm.CacheLatency.WithLabelValues(tier, operation).Observe(duration.Seconds())
}

// UpdateCacheHitRate updates real-time hit rate
func (pm *PrometheusMetrics) UpdateCacheHitRate(tier string, hitRate float64) {
	pm.CacheHitRate.WithLabelValues(tier).Set(hitRate)
}

// UpdateCacheMemoryUsage updates memory usage by tier
func (pm *PrometheusMetrics) UpdateCacheMemoryUsage(tier string, bytes int64) {
	pm.CacheMemoryUsage.WithLabelValues(tier).Set(float64(bytes))
}

// RecordCacheEviction records cache eviction events
func (pm *PrometheusMetrics) RecordCacheEviction(tier, reason string) {
	pm.CacheEvictions.WithLabelValues(tier, reason).Inc()
}

// UpdateCacheSize updates current cache size
func (pm *PrometheusMetrics) UpdateCacheSize(tier string, itemCount int) {
	pm.CacheSize.WithLabelValues(tier).Set(float64(itemCount))
}

// RecordCacheWarming records cache warming operations
func (pm *PrometheusMetrics) RecordCacheWarming(strategy, status string) {
	pm.CacheWarming.WithLabelValues(strategy, status).Inc()
}

// Multi-tier specific metrics

// RecordCacheTierHit records hit with operation type context
func (pm *PrometheusMetrics) RecordCacheTierHit(tier, operationType string) {
	pm.CacheTierHits.WithLabelValues(tier, operationType).Inc()
}

// RecordCacheTierMiss records miss with operation type context
func (pm *PrometheusMetrics) RecordCacheTierMiss(tier, operationType string) {
	pm.CacheTierMisses.WithLabelValues(tier, operationType).Inc()
}

// RecordCachePromotion records data promotion between tiers
func (pm *PrometheusMetrics) RecordCachePromotion(fromTier, toTier string) {
	pm.CachePromotions.WithLabelValues(fromTier, toTier).Inc()
}

// RecordCacheInvalidation records cache invalidation events
func (pm *PrometheusMetrics) RecordCacheInvalidation(patternType, reason string) {
	pm.CacheInvalidations.WithLabelValues(patternType, reason).Inc()
}

// RecordCacheBatchOperation records batch operations with size classification
func (pm *PrometheusMetrics) RecordCacheBatchOperation(operation string, batchSize int) {
	sizeBucket := "small"
	if batchSize > 100 {
		sizeBucket = "large"
	} else if batchSize > 10 {
		sizeBucket = "medium"
	}
	
	pm.CacheBatchOperations.WithLabelValues(operation, sizeBucket).Inc()
}

// Performance and SLA metrics

// UpdateSLACompliance updates SLA compliance status
func (pm *PrometheusMetrics) UpdateSLACompliance(metricType string, isCompliant bool) {
	value := 0.0
	if isCompliant {
		value = 1.0
	}
	pm.SLACompliance.WithLabelValues(metricType).Set(value)
}

// UpdatePerformanceScore updates overall performance score
func (pm *PrometheusMetrics) UpdatePerformanceScore(score float64) {
	pm.PerformanceScore.Set(score)
}

// UpdateThroughput updates actual throughput measurement
func (pm *PrometheusMetrics) UpdateThroughput(rps float64) {
	pm.ThroughputActual.Set(rps)
}

// UpdateLatencyPercentile updates latency percentile measurements
func (pm *PrometheusMetrics) UpdateLatencyPercentile(percentile string, latencySeconds float64) {
	pm.LatencyPercentiles.WithLabelValues(percentile).Set(latencySeconds)
}

// RecordCELEvaluation records a CEL evaluation
func (pm *PrometheusMetrics) RecordCELEvaluation(expressionType, status string, duration time.Duration) {
	pm.CELEvaluationsTotal.WithLabelValues(expressionType, status).Inc()
	pm.CELEvaluationDuration.WithLabelValues(expressionType).Observe(duration.Seconds())
}

// RecordDatabaseOperation records a database operation
func (pm *PrometheusMetrics) RecordDatabaseOperation(operation, collection, status string, duration time.Duration) {
	pm.DatabaseOperationsTotal.WithLabelValues(operation, collection, status).Inc()
	pm.DatabaseOperationDuration.WithLabelValues(operation, collection).Observe(duration.Seconds())
}

// RecordSLAViolation records an SLA violation
func (pm *PrometheusMetrics) RecordSLAViolation(endpoint, slaType string) {
	pm.SLAViolations.WithLabelValues(endpoint, slaType).Inc()
}

// RecordError records an error
func (pm *PrometheusMetrics) RecordError(errorType, component string) {
	pm.ErrorsTotal.WithLabelValues(errorType, component).Inc()
}

// RecordValidationError records a validation error
func (pm *PrometheusMetrics) RecordValidationError(validationType string) {
	pm.ValidationErrors.WithLabelValues(validationType).Inc()
}

// IncrementConcurrentRequests increments the concurrent requests gauge
func (pm *PrometheusMetrics) IncrementConcurrentRequests() {
	pm.ConcurrentRequests.Inc()
}

// DecrementConcurrentRequests decrements the concurrent requests gauge
func (pm *PrometheusMetrics) DecrementConcurrentRequests() {
	pm.ConcurrentRequests.Dec()
}

// RecordBatchSize records the size of a processing batch
func (pm *PrometheusMetrics) RecordBatchSize(size float64) {
	pm.BatchSizeHistogram.Observe(size)
}

// Advanced metrics for performance analysis

// RecordEndToEndLatency records complete request latency with cache breakdown
func (pm *PrometheusMetrics) RecordEndToEndLatency(endpoint string, totalDuration time.Duration, cacheTime time.Duration, processingTime time.Duration) {
	// Record total duration
	pm.RequestDuration.WithLabelValues("POST", endpoint).Observe(totalDuration.Seconds())
	
	// Record cache efficiency ratio
	if totalDuration > 0 {
		cacheRatio := float64(cacheTime) / float64(totalDuration)
		// You could add a cache efficiency gauge here
		_ = cacheRatio
	}
}

// UpdateCacheEfficiency updates cache efficiency metrics
func (pm *PrometheusMetrics) UpdateCacheEfficiency(tier string, efficiency float64) {
	// efficiency = (cache_hits / total_requests) for the tier
	// This could be added as a separate gauge if needed
}

// RecordBatchPerformance records batch operation performance
func (pm *PrometheusMetrics) RecordBatchPerformance(batchSize int, duration time.Duration, cacheHitCount int) {
	// Record batch size
	pm.RecordBatchSize(float64(batchSize))
	
	// Calculate cache hit ratio for batch
	if batchSize > 0 {
		hitRatio := float64(cacheHitCount) / float64(batchSize)
		// Record batch hit ratio - could add specific gauge for this
		_ = hitRatio
	}
	
	// Record if batch meets SLA (1000 patients < 1s)
	if batchSize >= 1000 {
		isCompliant := duration < time.Second
		pm.UpdateSLACompliance("batch_1000_patients", isCompliant)
	}
}