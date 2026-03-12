package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"
)

// Collector holds all metrics for KB-5 Drug Interactions service
type Collector struct {
	// HTTP metrics
	RequestsTotal     *prometheus.CounterVec
	RequestDuration   *prometheus.HistogramVec
	ResponseSize      *prometheus.HistogramVec
	
	// Interaction check metrics
	InteractionChecksTotal      *prometheus.CounterVec
	InteractionCheckDuration    *prometheus.HistogramVec
	InteractionsFoundTotal      *prometheus.CounterVec
	InteractionSeverityGauge    *prometheus.GaugeVec
	
	// Cache metrics
	CacheHits             *prometheus.CounterVec
	CacheMisses           *prometheus.CounterVec
	CacheOperations       *prometheus.CounterVec
	CacheSize             *prometheus.GaugeVec
	
	// Database metrics
	DatabaseQueries       *prometheus.CounterVec
	DatabaseErrors        *prometheus.CounterVec
	DatabaseDuration      *prometheus.HistogramVec
	DatabaseConnections   prometheus.Gauge
	
	// Alert metrics
	AlertsCreatedTotal    *prometheus.CounterVec
	AlertsResolvedTotal   *prometheus.CounterVec
	AlertOverridesTotal   *prometheus.CounterVec
	ActiveAlertsGauge     *prometheus.GaugeVec
	
	// Drug synonym metrics
	SynonymResolutionsTotal  *prometheus.CounterVec
	SynonymResolutionTime    *prometheus.HistogramVec
	SynonymMappingAccuracy   *prometheus.GaugeVec
	
	// Rule evaluation metrics
	RuleEvaluationsTotal     *prometheus.CounterVec
	RuleExecutionTime        *prometheus.HistogramVec
	ActiveRulesGauge         prometheus.Gauge
	RuleMatchesTotal         *prometheus.CounterVec
	
	// Clinical decision support metrics
	CDSRecommendationsTotal  *prometheus.CounterVec
	CDSOverridesTotal        *prometheus.CounterVec
	CDSComplianceRate        *prometheus.GaugeVec
	
	// Analytics metrics
	InteractionFrequency     *prometheus.HistogramVec
	ClinicalOutcomesTotal    *prometheus.CounterVec
	AdverseEventsTotal       *prometheus.CounterVec
	
	// Performance metrics
	BatchProcessingTime      *prometheus.HistogramVec
	ConcurrentChecksGauge    prometheus.Gauge
	ErrorRateGauge           *prometheus.GaugeVec
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		// HTTP metrics
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb5",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
			},
			[]string{"method", "endpoint"},
		),
		
		ResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb5",
				Name:      "response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 6),
			},
			[]string{"endpoint"},
		),
		
		// Interaction check metrics
		InteractionChecksTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "interaction_checks_total",
				Help:      "Total number of drug interaction checks performed",
			},
			[]string{"check_type", "status"},
		),
		
		InteractionCheckDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb5",
				Name:      "interaction_check_duration_seconds",
				Help:      "Time taken to perform interaction checks",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 5.0},
			},
			[]string{"check_type"},
		),
		
		InteractionsFoundTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "interactions_found_total",
				Help:      "Total number of drug interactions found",
			},
			[]string{"severity", "interaction_type", "evidence_level"},
		),
		
		InteractionSeverityGauge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "kb5",
				Name:      "interaction_severity_distribution",
				Help:      "Distribution of interaction severities",
			},
			[]string{"severity"},
		),
		
		// Cache metrics
		CacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache_type", "operation"},
		),
		
		CacheMisses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache_type", "operation"},
		),
		
		CacheOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "cache_operations_total",
				Help:      "Total number of cache operations",
			},
			[]string{"operation", "cache_type", "status"},
		),
		
		CacheSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "kb5",
				Name:      "cache_size_bytes",
				Help:      "Current cache size in bytes",
			},
			[]string{"cache_type"},
		),
		
		// Database metrics
		DatabaseQueries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "database_queries_total",
				Help:      "Total number of database queries",
			},
			[]string{"operation", "table", "status"},
		),
		
		DatabaseErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "database_errors_total",
				Help:      "Total number of database errors",
			},
			[]string{"operation", "error_type"},
		),
		
		DatabaseDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb5",
				Name:      "database_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0},
			},
			[]string{"operation", "table"},
		),
		
		DatabaseConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "kb5",
				Name:      "database_connections",
				Help:      "Number of active database connections",
			},
		),
		
		// Alert metrics
		AlertsCreatedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "alerts_created_total",
				Help:      "Total number of interaction alerts created",
			},
			[]string{"alert_type", "severity", "source"},
		),
		
		AlertsResolvedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "alerts_resolved_total",
				Help:      "Total number of alerts resolved",
			},
			[]string{"alert_type", "resolution_method"},
		),
		
		AlertOverridesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "alert_overrides_total",
				Help:      "Total number of alert overrides",
			},
			[]string{"alert_type", "override_reason_category"},
		),
		
		ActiveAlertsGauge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "kb5",
				Name:      "active_alerts",
				Help:      "Number of currently active interaction alerts",
			},
			[]string{"severity", "alert_type"},
		),
		
		// Drug synonym metrics
		SynonymResolutionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "synonym_resolutions_total",
				Help:      "Total number of drug synonym resolutions performed",
			},
			[]string{"synonym_type", "status"},
		),
		
		SynonymResolutionTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb5",
				Name:      "synonym_resolution_duration_seconds",
				Help:      "Time taken to resolve drug synonyms",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
			},
			[]string{"synonym_type"},
		),
		
		SynonymMappingAccuracy: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "kb5",
				Name:      "synonym_mapping_accuracy",
				Help:      "Accuracy of drug synonym mappings",
			},
			[]string{"synonym_type"},
		),
		
		// Rule evaluation metrics
		RuleEvaluationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "rule_evaluations_total",
				Help:      "Total number of interaction rule evaluations",
			},
			[]string{"rule_type", "result"},
		),
		
		RuleExecutionTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb5",
				Name:      "rule_execution_duration_seconds",
				Help:      "Time taken to execute interaction rules",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
			},
			[]string{"rule_type"},
		),
		
		ActiveRulesGauge: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "kb5",
				Name:      "active_rules",
				Help:      "Number of currently active interaction rules",
			},
		),
		
		RuleMatchesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "rule_matches_total",
				Help:      "Total number of rule matches",
			},
			[]string{"rule_type", "match_severity"},
		),
		
		// Clinical decision support metrics
		CDSRecommendationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "cds_recommendations_total",
				Help:      "Total number of CDS recommendations generated",
			},
			[]string{"recommendation_type", "severity"},
		),
		
		CDSOverridesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "cds_overrides_total",
				Help:      "Total number of CDS recommendation overrides",
			},
			[]string{"recommendation_type", "override_category"},
		),
		
		CDSComplianceRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "kb5",
				Name:      "cds_compliance_rate",
				Help:      "Compliance rate with CDS recommendations",
			},
			[]string{"recommendation_type"},
		),
		
		// Analytics metrics
		InteractionFrequency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb5",
				Name:      "interaction_frequency",
				Help:      "Frequency of specific drug interactions",
				Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
			},
			[]string{"interaction_id", "severity"},
		),
		
		ClinicalOutcomesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "clinical_outcomes_total",
				Help:      "Total number of recorded clinical outcomes",
			},
			[]string{"outcome_type", "severity"},
		),
		
		AdverseEventsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kb5",
				Name:      "adverse_events_total",
				Help:      "Total number of adverse events related to drug interactions",
			},
			[]string{"event_type", "interaction_severity"},
		),
		
		// Performance metrics
		BatchProcessingTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kb5",
				Name:      "batch_processing_duration_seconds",
				Help:      "Time taken to process batch requests",
				Buckets:   []float64{0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
			},
			[]string{"batch_type"},
		),
		
		ConcurrentChecksGauge: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "kb5",
				Name:      "concurrent_checks",
				Help:      "Number of concurrent interaction checks being processed",
			},
		),
		
		ErrorRateGauge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "kb5",
				Name:      "error_rate",
				Help:      "Current error rate for different operations",
			},
			[]string{"operation_type"},
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

// Interaction Check Metrics Methods

func (c *Collector) RecordInteractionCheck(checkType string, duration time.Duration) {
	c.InteractionChecksTotal.WithLabelValues(checkType, "success").Inc()
	c.InteractionCheckDuration.WithLabelValues(checkType).Observe(duration.Seconds())
}

func (c *Collector) RecordInteractionCheckError(checkType string) {
	c.InteractionChecksTotal.WithLabelValues(checkType, "error").Inc()
}

func (c *Collector) RecordInteractionFound(severity, interactionType, evidenceLevel string) {
	c.InteractionsFoundTotal.WithLabelValues(severity, interactionType, evidenceLevel).Inc()
}

func (c *Collector) SetInteractionSeverityDistribution(severity string, count float64) {
	c.InteractionSeverityGauge.WithLabelValues(severity).Set(count)
}

// Cache Metrics Methods

func (c *Collector) RecordCacheHit(cacheType, operation string) {
	c.CacheHits.WithLabelValues(cacheType, operation).Inc()
}

func (c *Collector) RecordCacheMiss(cacheType, operation string) {
	c.CacheMisses.WithLabelValues(cacheType, operation).Inc()
}

func (c *Collector) RecordCacheOperation(operation, cacheType, status string) {
	c.CacheOperations.WithLabelValues(operation, cacheType, status).Inc()
}

func (c *Collector) SetCacheSize(cacheType string, size float64) {
	c.CacheSize.WithLabelValues(cacheType).Set(size)
}

// Database Metrics Methods

func (c *Collector) RecordDatabaseQuery(operation, table, status string, duration time.Duration) {
	c.DatabaseQueries.WithLabelValues(operation, table, status).Inc()
	c.DatabaseDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

func (c *Collector) RecordDatabaseError(operation, errorType string) {
	c.DatabaseErrors.WithLabelValues(operation, errorType).Inc()
}

func (c *Collector) SetDatabaseConnections(count float64) {
	c.DatabaseConnections.Set(count)
}

// Alert Metrics Methods

func (c *Collector) RecordAlertCreated(alertType, severity, source string) {
	c.AlertsCreatedTotal.WithLabelValues(alertType, severity, source).Inc()
}

func (c *Collector) RecordAlertResolved(alertType, resolutionMethod string) {
	c.AlertsResolvedTotal.WithLabelValues(alertType, resolutionMethod).Inc()
}

func (c *Collector) RecordAlertOverride(alertType, overrideReason string) {
	c.AlertOverridesTotal.WithLabelValues(alertType, overrideReason).Inc()
}

func (c *Collector) SetActiveAlerts(severity, alertType string, count float64) {
	c.ActiveAlertsGauge.WithLabelValues(severity, alertType).Set(count)
}

// Synonym Metrics Methods

func (c *Collector) RecordSynonymResolution(synonymType, status string, duration time.Duration) {
	c.SynonymResolutionsTotal.WithLabelValues(synonymType, status).Inc()
	c.SynonymResolutionTime.WithLabelValues(synonymType).Observe(duration.Seconds())
}

func (c *Collector) SetSynonymMappingAccuracy(synonymType string, accuracy float64) {
	c.SynonymMappingAccuracy.WithLabelValues(synonymType).Set(accuracy)
}

// Rule Evaluation Metrics Methods

func (c *Collector) RecordRuleEvaluation(ruleType, result string, duration time.Duration) {
	c.RuleEvaluationsTotal.WithLabelValues(ruleType, result).Inc()
	c.RuleExecutionTime.WithLabelValues(ruleType).Observe(duration.Seconds())
}

func (c *Collector) SetActiveRules(count float64) {
	c.ActiveRulesGauge.Set(count)
}

func (c *Collector) RecordRuleMatch(ruleType, severity string) {
	c.RuleMatchesTotal.WithLabelValues(ruleType, severity).Inc()
}

// CDS Metrics Methods

func (c *Collector) RecordCDSRecommendation(recommendationType, severity string) {
	c.CDSRecommendationsTotal.WithLabelValues(recommendationType, severity).Inc()
}

func (c *Collector) RecordCDSOverride(recommendationType, overrideCategory string) {
	c.CDSOverridesTotal.WithLabelValues(recommendationType, overrideCategory).Inc()
}

func (c *Collector) SetCDSComplianceRate(recommendationType string, rate float64) {
	c.CDSComplianceRate.WithLabelValues(recommendationType).Set(rate)
}

// Analytics Metrics Methods

func (c *Collector) RecordInteractionFrequency(interactionID, severity string, frequency float64) {
	c.InteractionFrequency.WithLabelValues(interactionID, severity).Observe(frequency)
}

func (c *Collector) RecordClinicalOutcome(outcomeType, severity string) {
	c.ClinicalOutcomesTotal.WithLabelValues(outcomeType, severity).Inc()
}

func (c *Collector) RecordAdverseEvent(eventType, interactionSeverity string) {
	c.AdverseEventsTotal.WithLabelValues(eventType, interactionSeverity).Inc()
}

// Performance Metrics Methods

func (c *Collector) RecordBatchProcessing(batchType string, duration time.Duration) {
	c.BatchProcessingTime.WithLabelValues(batchType).Observe(duration.Seconds())
}

func (c *Collector) SetConcurrentChecks(count float64) {
	c.ConcurrentChecksGauge.Set(count)
}

func (c *Collector) SetErrorRate(operationType string, rate float64) {
	c.ErrorRateGauge.WithLabelValues(operationType).Set(rate)
}

// Utility Methods

// GetCacheHitRate calculates cache hit rate for a specific cache type
func (c *Collector) GetCacheHitRate(cacheType string) float64 {
	hitMetric := c.CacheHits.WithLabelValues(cacheType, "get")
	missMetric := c.CacheMisses.WithLabelValues(cacheType, "get")
	
	hits := getCounterValue(hitMetric)
	misses := getCounterValue(missMetric)
	
	total := hits + misses
	if total == 0 {
		return 0
	}
	
	return hits / total
}

// Helper function to extract counter value
func getCounterValue(counter prometheus.Counter) float64 {
	metric := &dto.Metric{}
	if err := counter.Write(metric); err != nil {
		return 0
	}
	return metric.GetCounter().GetValue()
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

// Background metrics updater for interaction statistics
func (c *Collector) UpdateInteractionMetrics(
	severityCounts map[string]float64,
	totalActiveAlerts map[string]map[string]float64, // map[severity][alert_type] = count
	totalActiveRules float64,
) {
	// Update severity distribution
	for severity, count := range severityCounts {
		c.SetInteractionSeverityDistribution(severity, count)
	}
	
	// Update active alerts by severity and type
	for severity, alertTypes := range totalActiveAlerts {
		for alertType, count := range alertTypes {
			c.SetActiveAlerts(severity, alertType, count)
		}
	}
	
	c.SetActiveRules(totalActiveRules)
}

// CDS compliance metrics updater
func (c *Collector) UpdateCDSMetrics(complianceRates map[string]float64) {
	for recommendationType, rate := range complianceRates {
		c.SetCDSComplianceRate(recommendationType, rate)
	}
}

// Database performance metrics updater
func (c *Collector) UpdateDatabaseMetrics(connections float64) {
	c.SetDatabaseConnections(connections)
}

// System performance metrics updater
func (c *Collector) UpdatePerformanceMetrics(
	concurrentChecks float64,
	errorRates map[string]float64,
) {
	c.SetConcurrentChecks(concurrentChecks)

	for operationType, rate := range errorRates {
		c.SetErrorRate(operationType, rate)
	}
}

// Drug Class Interaction Metrics Methods

func (c *Collector) RecordClassInteractionCheck(checkType string, duration time.Duration) {
	c.InteractionCheckDuration.WithLabelValues("class_"+checkType).Observe(duration.Seconds())
	c.InteractionChecksTotal.WithLabelValues("class_interaction", "success").Inc()
}

func (c *Collector) RecordClassInteractionsFound(count int, interactionType string) {
	c.InteractionsFoundTotal.WithLabelValues("class", interactionType, "B").Add(float64(count))
}

func (c *Collector) RecordTripleWhammyDetection(status string) {
	c.InteractionsFoundTotal.WithLabelValues("moderate", "triple_whammy", "B").Inc()
	c.RuleMatchesTotal.WithLabelValues("triple_whammy", status).Inc()
}

func (c *Collector) RecordClassInteraction(drugClass1, drugClass2, severity string) {
	c.InteractionsFoundTotal.WithLabelValues(severity, "class_to_class", "B").Inc()
	c.RuleMatchesTotal.WithLabelValues("class_interaction", severity).Inc()
}

// Matrix Load Metrics

func (c *Collector) RecordMatrixLoad(loadType string, duration time.Duration) {
	c.DatabaseDuration.WithLabelValues("matrix_load", "ddi_interaction_matrix").Observe(duration.Seconds())
	c.DatabaseQueries.WithLabelValues("matrix_load", "ddi_interaction_matrix", "success").Inc()
}
// PGX Interaction Metrics

func (c *Collector) RecordPGXEvaluation(duration time.Duration, variantCount int) {
	c.DatabaseDuration.WithLabelValues("pgx_eval", "pgx_markers").Observe(duration.Seconds())
	c.DatabaseQueries.WithLabelValues("pgx_eval", "pgx_markers", "success").Inc()
}

func (c *Collector) RecordPGXInteraction(markerType, severity string) {
	c.InteractionsFoundTotal.WithLabelValues(severity, "pgx", "A").Inc()
	c.RuleMatchesTotal.WithLabelValues("pgx_"+markerType, severity).Inc()
}

// RecordGRPCRequest records metrics for a gRPC request
func (c *Collector) RecordGRPCRequest(method, status string, duration time.Duration) {
	c.RequestDuration.WithLabelValues("grpc", method).Observe(duration.Seconds())
	c.RequestsTotal.WithLabelValues("grpc", method, status).Inc()
}

// Drug-Disease Contraindication Metrics

// RecordDrugDiseaseCheck records metrics for a drug-disease contraindication check
func (c *Collector) RecordDrugDiseaseCheck(duration time.Duration, pairCount int) {
	c.InteractionCheckDuration.WithLabelValues("drug_disease").Observe(duration.Seconds())
	c.InteractionChecksTotal.WithLabelValues("drug_disease", "success").Inc()
}

// RecordDrugDiseaseInteraction records a detected drug-disease contraindication
func (c *Collector) RecordDrugDiseaseInteraction(drugCode, diseaseCode, severity string) {
	c.InteractionsFoundTotal.WithLabelValues(severity, "drug_disease", "B").Inc()
	c.RuleMatchesTotal.WithLabelValues("drug_disease", severity).Inc()
}

// Allergy Cross-Reactivity Metrics

// RecordAllergyCheck records metrics for an allergy cross-reactivity check
func (c *Collector) RecordAllergyCheck(duration time.Duration, checkCount int) {
	c.InteractionCheckDuration.WithLabelValues("allergy").Observe(duration.Seconds())
	c.InteractionChecksTotal.WithLabelValues("allergy", "success").Inc()
}

// RecordAllergyInteraction records a detected allergy cross-reactivity
func (c *Collector) RecordAllergyInteraction(allergenCode, drugCode, severity string) {
	c.InteractionsFoundTotal.WithLabelValues(severity, "allergy_cross_reactivity", "B").Inc()
	c.RuleMatchesTotal.WithLabelValues("allergy", severity).Inc()
}

// Duplicate Therapy Metrics

// RecordDuplicateTherapyCheck records metrics for a duplicate therapy check
func (c *Collector) RecordDuplicateTherapyCheck(duration time.Duration, drugCount int) {
	c.InteractionCheckDuration.WithLabelValues("duplicate_therapy").Observe(duration.Seconds())
	c.InteractionChecksTotal.WithLabelValues("duplicate_therapy", "success").Inc()
}

// RecordDuplicateTherapy records a detected duplicate therapy
func (c *Collector) RecordDuplicateTherapy(atcCode string, drugCount int) {
	c.InteractionsFoundTotal.WithLabelValues("moderate", "duplicate_therapy", "C").Inc()
	c.RuleMatchesTotal.WithLabelValues("duplicate_"+atcCode, "moderate").Inc()
}
