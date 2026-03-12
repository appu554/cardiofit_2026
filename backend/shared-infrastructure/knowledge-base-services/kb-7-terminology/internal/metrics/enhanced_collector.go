package metrics

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/models"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

// EnhancedCollector extends the basic metrics collector with advanced monitoring capabilities
type EnhancedCollector struct {
	*Collector // Embed the basic collector
	
	// Enhanced terminology metrics
	expansionRequestsTotal    *prometheus.CounterVec
	expansionDuration         *prometheus.HistogramVec
	snomedValidationTotal     *prometheus.CounterVec
	snomedValidationDuration  *prometheus.HistogramVec
	batchOperationTotal       *prometheus.CounterVec
	batchOperationDuration    *prometheus.HistogramVec
	
	// Advanced cache metrics
	cacheLayerHits            *prometheus.CounterVec
	cacheMemoryUsage          *prometheus.GaugeVec
	cacheEvictions            *prometheus.CounterVec
	cacheTTLRemaining         *prometheus.GaugeVec
	
	// ETL and data quality metrics
	etlOperationTotal         *prometheus.CounterVec
	etlOperationDuration      *prometheus.HistogramVec
	etlRecordsProcessed       *prometheus.CounterVec
	dataQualityScore          *prometheus.GaugeVec
	dataFreshness             *prometheus.GaugeVec
	
	// Performance and reliability metrics
	serviceAvailability       prometheus.Gauge
	errorRate                 *prometheus.GaugeVec
	responseTimePercentiles   *prometheus.SummaryVec
	
	// Security metrics
	authenticationAttempts    *prometheus.CounterVec
	rateLimitExceeded         *prometheus.CounterVec
	licenseViolations         *prometheus.CounterVec
	
	// Background processes
	backgroundCollector       *BackgroundMetricsCollector
	logger                    *logrus.Logger
	db                        *sql.DB
	cache                     cache.EnhancedCache
	
	// Metrics state
	metricsState             MetricsState
	stateMutex               sync.RWMutex
}

// MetricsState holds current state information
type MetricsState struct {
	ServiceStartTime      time.Time                    `json:"service_start_time"`
	LastMetricsUpdate     time.Time                    `json:"last_metrics_update"`
	SystemStatus          string                       `json:"system_status"`
	ActiveConnections     int64                        `json:"active_connections"`
	CacheStatistics       cache.CacheStatistics        `json:"cache_statistics"`
	PerformanceMetrics    PerformanceMetrics           `json:"performance_metrics"`
	QualityMetrics        map[string]float64           `json:"quality_metrics"`
	AlertsActive          []ActiveAlert                `json:"alerts_active"`
}

// PerformanceMetrics holds current performance metrics
type PerformanceMetrics struct {
	RequestsPerSecond     float64 `json:"requests_per_second"`
	AverageResponseTime   float64 `json:"average_response_time"`
	P95ResponseTime       float64 `json:"p95_response_time"`
	P99ResponseTime       float64 `json:"p99_response_time"`
	ErrorRate             float64 `json:"error_rate"`
	CacheHitRate          float64 `json:"cache_hit_rate"`
	ThroughputMBps        float64 `json:"throughput_mbps"`
}

// ActiveAlert represents an active monitoring alert
type ActiveAlert struct {
	AlertName     string                 `json:"alert_name"`
	Severity      string                 `json:"severity"`
	Description   string                 `json:"description"`
	StartTime     time.Time              `json:"start_time"`
	Labels        map[string]string      `json:"labels"`
	Value         float64                `json:"value"`
	Threshold     float64                `json:"threshold"`
}

// BackgroundMetricsCollector handles background metric collection
type BackgroundMetricsCollector struct {
	db         *sql.DB
	cache      cache.EnhancedCache
	logger     *logrus.Logger
	collector  *EnhancedCollector
	
	stopCh     chan struct{}
	interval   time.Duration
}

// NewEnhancedCollector creates a new enhanced metrics collector
func NewEnhancedCollector(namespace string, db *sql.DB, cache cache.EnhancedCache, logger *logrus.Logger) *EnhancedCollector {
	basicCollector := NewCollector(namespace)
	
	enhanced := &EnhancedCollector{
		Collector: basicCollector,
		db:        db,
		cache:     cache,
		logger:    logger,
		metricsState: MetricsState{
			ServiceStartTime:   time.Now(),
			SystemStatus:       "starting",
			QualityMetrics:     make(map[string]float64),
			AlertsActive:       make([]ActiveAlert, 0),
		},
		
		// Enhanced terminology metrics
		expansionRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "expansion_requests_total",
				Help:      "Total number of value set expansion requests",
			},
			[]string{"value_set", "status", "cache_hit"},
		),
		expansionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "expansion_duration_seconds",
				Help:      "Value set expansion duration in seconds",
				Buckets:   []float64{.1, .5, 1, 2, 5, 10, 30, 60},
			},
			[]string{"value_set", "complexity"},
		),
		snomedValidationTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "snomed_validation_total",
				Help:      "Total number of SNOMED expression validations",
			},
			[]string{"validation_type", "status"},
		),
		snomedValidationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "snomed_validation_duration_seconds",
				Help:      "SNOMED validation duration in seconds",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2},
			},
			[]string{"validation_type"},
		),
		batchOperationTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "batch_operation_total",
				Help:      "Total number of batch operations",
			},
			[]string{"operation_type", "status"},
		),
		batchOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "batch_operation_duration_seconds",
				Help:      "Batch operation duration in seconds",
				Buckets:   []float64{.1, .5, 1, 2, 5, 10, 30, 60, 120},
			},
			[]string{"operation_type", "batch_size"},
		),
		
		// Advanced cache metrics
		cacheLayerHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_layer_hits_total",
				Help:      "Cache hits by layer (L1, L2, L3)",
			},
			[]string{"layer", "key_type"},
		),
		cacheMemoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "cache_memory_usage_bytes",
				Help:      "Cache memory usage in bytes",
			},
			[]string{"layer"},
		),
		cacheEvictions: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_evictions_total",
				Help:      "Total number of cache evictions",
			},
			[]string{"layer", "reason"},
		),
		
		// ETL metrics
		etlOperationTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "etl_operation_total",
				Help:      "Total number of ETL operations",
			},
			[]string{"system", "operation", "status"},
		),
		etlOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "etl_operation_duration_seconds",
				Help:      "ETL operation duration in seconds",
				Buckets:   []float64{60, 300, 600, 1800, 3600, 7200, 14400}, // 1min to 4hours
			},
			[]string{"system", "operation"},
		),
		etlRecordsProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "etl_records_processed_total",
				Help:      "Total number of records processed by ETL",
			},
			[]string{"system", "record_type"},
		),
		dataQualityScore: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "data_quality_score",
				Help:      "Data quality score (0-1) for terminology systems",
			},
			[]string{"system", "metric_type"},
		),
		dataFreshness: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "data_freshness_hours",
				Help:      "Data freshness in hours since last update",
			},
			[]string{"system"},
		),
		
		// Performance and reliability
		serviceAvailability: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "service_availability",
				Help:      "Service availability (0-1)",
			},
		),
		errorRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "error_rate",
				Help:      "Error rate by endpoint",
			},
			[]string{"endpoint"},
		),
		responseTimePercentiles: promauto.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:  namespace,
				Name:       "response_time_percentiles",
				Help:       "Response time percentiles",
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
			},
			[]string{"endpoint"},
		),
		
		// Security metrics
		authenticationAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "authentication_attempts_total",
				Help:      "Total authentication attempts",
			},
			[]string{"result", "method"},
		),
		rateLimitExceeded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rate_limit_exceeded_total",
				Help:      "Total rate limit violations",
			},
			[]string{"endpoint", "client_id"},
		),
		licenseViolations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "license_violations_total",
				Help:      "Total license violations",
			},
			[]string{"system", "violation_type"},
		),
	}
	
	// Initialize background collector
	enhanced.backgroundCollector = &BackgroundMetricsCollector{
		db:        db,
		cache:     cache,
		logger:    logger,
		collector: enhanced,
		stopCh:    make(chan struct{}),
		interval:  30 * time.Second, // Collect metrics every 30 seconds
	}
	
	return enhanced
}

// StartBackgroundCollection starts the background metrics collection
func (c *EnhancedCollector) StartBackgroundCollection() {
	go c.backgroundCollector.run()
	c.logger.Info("Background metrics collection started")
	
	c.stateMutex.Lock()
	c.metricsState.SystemStatus = "running"
	c.stateMutex.Unlock()
}

// StopBackgroundCollection stops the background metrics collection
func (c *EnhancedCollector) StopBackgroundCollection() {
	close(c.backgroundCollector.stopCh)
	c.logger.Info("Background metrics collection stopped")
	
	c.stateMutex.Lock()
	c.metricsState.SystemStatus = "stopped"
	c.stateMutex.Unlock()
}

// Enhanced metric recording methods

// RecordExpansion records value set expansion metrics
func (c *EnhancedCollector) RecordExpansion(valueSet, status string, cacheHit bool, duration time.Duration) {
	cacheHitLabel := "false"
	if cacheHit {
		cacheHitLabel = "true"
	}
	c.expansionRequestsTotal.WithLabelValues(valueSet, status, cacheHitLabel).Inc()
	
	complexity := c.determineComplexity(duration)
	c.expansionDuration.WithLabelValues(valueSet, complexity).Observe(duration.Seconds())
}

// RecordSNOMEDValidation records SNOMED expression validation metrics
func (c *EnhancedCollector) RecordSNOMEDValidation(validationType, status string, duration time.Duration) {
	c.snomedValidationTotal.WithLabelValues(validationType, status).Inc()
	c.snomedValidationDuration.WithLabelValues(validationType).Observe(duration.Seconds())
}

// RecordBatchOperation records batch operation metrics
func (c *EnhancedCollector) RecordBatchOperation(operationType, status string, batchSize int, duration time.Duration) {
	c.batchOperationTotal.WithLabelValues(operationType, status).Inc()
	
	sizeCategory := c.categorizeBatchSize(batchSize)
	c.batchOperationDuration.WithLabelValues(operationType, sizeCategory).Observe(duration.Seconds())
}

// RecordCacheLayerHit records cache hit by specific layer
func (c *EnhancedCollector) RecordCacheLayerHit(layer, keyType string) {
	c.cacheLayerHits.WithLabelValues(layer, keyType).Inc()
}

// RecordETLOperation records ETL operation metrics
func (c *EnhancedCollector) RecordETLOperation(system, operation, status string, duration time.Duration) {
	c.etlOperationTotal.WithLabelValues(system, operation, status).Inc()
	c.etlOperationDuration.WithLabelValues(system, operation).Observe(duration.Seconds())
}

// RecordETLRecords records ETL record processing metrics
func (c *EnhancedCollector) RecordETLRecords(system, recordType string, count int64) {
	c.etlRecordsProcessed.WithLabelValues(system, recordType).Add(float64(count))
}

// SetDataQuality sets data quality scores
func (c *EnhancedCollector) SetDataQuality(system, metricType string, score float64) {
	c.dataQualityScore.WithLabelValues(system, metricType).Set(score)
	
	c.stateMutex.Lock()
	c.metricsState.QualityMetrics[system+"_"+metricType] = score
	c.stateMutex.Unlock()
}

// SetDataFreshness sets data freshness metrics
func (c *EnhancedCollector) SetDataFreshness(system string, hours float64) {
	c.dataFreshness.WithLabelValues(system).Set(hours)
}

// Security metric recording
func (c *EnhancedCollector) RecordAuthenticationAttempt(result, method string) {
	c.authenticationAttempts.WithLabelValues(result, method).Inc()
}

func (c *EnhancedCollector) RecordRateLimitExceeded(endpoint, clientID string) {
	c.rateLimitExceeded.WithLabelValues(endpoint, clientID).Inc()
}

func (c *EnhancedCollector) RecordLicenseViolation(system, violationType string) {
	c.licenseViolations.WithLabelValues(system, violationType).Inc()
}

// GetCurrentMetrics returns current metrics state
func (c *EnhancedCollector) GetCurrentMetrics() models.ServiceMetrics {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	
	return models.ServiceMetrics{
		RequestsPerSecond:   c.metricsState.PerformanceMetrics.RequestsPerSecond,
		AverageLatencyMs:    c.metricsState.PerformanceMetrics.AverageResponseTime,
		P95LatencyMs:        c.metricsState.PerformanceMetrics.P95ResponseTime,
		P99LatencyMs:        c.metricsState.PerformanceMetrics.P99ResponseTime,
		CacheHitRate:        c.metricsState.PerformanceMetrics.CacheHitRate,
		ErrorRate:           c.metricsState.PerformanceMetrics.ErrorRate,
		ActiveConnections:   int(c.metricsState.ActiveConnections),
	}
}

// GetHealthStatus returns overall service health
func (c *EnhancedCollector) GetHealthStatus() models.HealthCheckResult {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	
	status := "healthy"
	if len(c.metricsState.AlertsActive) > 0 {
		for _, alert := range c.metricsState.AlertsActive {
			if alert.Severity == "critical" {
				status = "unhealthy"
				break
			} else if alert.Severity == "warning" && status == "healthy" {
				status = "degraded"
			}
		}
	}
	
	checks := make(map[string]models.ComponentCheck)
	
	// Database health
	checks["database"] = c.getDatabaseHealth()
	
	// Cache health
	checks["cache"] = c.getCacheHealth()
	
	// Performance health
	checks["performance"] = c.getPerformanceHealth()
	
	return models.HealthCheckResult{
		Service:   "kb-7-terminology",
		Status:    status,
		Timestamp: time.Now(),
		Checks:    checks,
	}
}

// Helper methods

func (c *EnhancedCollector) determineComplexity(duration time.Duration) string {
	if duration < 100*time.Millisecond {
		return "simple"
	} else if duration < 1*time.Second {
		return "moderate"
	} else {
		return "complex"
	}
}

func (c *EnhancedCollector) categorizeBatchSize(size int) string {
	if size < 10 {
		return "small"
	} else if size < 100 {
		return "medium"
	} else if size < 1000 {
		return "large"
	} else {
		return "xlarge"
	}
}

func (c *EnhancedCollector) getDatabaseHealth() models.ComponentCheck {
	// Check database connectivity and performance
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	start := time.Now()
	err := c.db.PingContext(ctx)
	latency := time.Since(start)
	
	if err != nil {
		return models.ComponentCheck{
			Status:      "unhealthy",
			Message:     "Database connectivity failed",
			Error:       err.Error(),
			LastUpdated: time.Now(),
		}
	}
	
	metrics := map[string]interface{}{
		"ping_latency_ms": latency.Milliseconds(),
		"connections":     c.metricsState.ActiveConnections,
	}
	
	status := "healthy"
	if latency > 1*time.Second {
		status = "degraded"
	}
	
	return models.ComponentCheck{
		Status:      status,
		Message:     "Database is responsive",
		LastUpdated: time.Now(),
		Metrics:     metrics,
	}
}

func (c *EnhancedCollector) getCacheHealth() models.ComponentCheck {
	if c.cache == nil {
		return models.ComponentCheck{
			Status:      "unhealthy",
			Message:     "Cache not available",
			LastUpdated: time.Now(),
		}
	}
	
	healthy, err := c.cache.HealthCheck()
	if err != nil || !healthy {
		return models.ComponentCheck{
			Status:      "unhealthy",
			Message:     "Cache health check failed",
			Error:       err.Error(),
			LastUpdated: time.Now(),
		}
	}
	
	stats := c.cache.GetStatistics()
	metrics := map[string]interface{}{
		"hit_rate":      stats.OverallHitRate,
		"total_requests": stats.TotalRequests,
		"total_hits":    stats.TotalHits,
	}
	
	return models.ComponentCheck{
		Status:      "healthy",
		Message:     "Cache is operational",
		LastUpdated: time.Now(),
		Metrics:     metrics,
	}
}

func (c *EnhancedCollector) getPerformanceHealth() models.ComponentCheck {
	perf := c.metricsState.PerformanceMetrics
	
	status := "healthy"
	if perf.ErrorRate > 0.05 { // >5% error rate
		status = "unhealthy"
	} else if perf.P95ResponseTime > 1000 || perf.ErrorRate > 0.01 { // >1s P95 or >1% errors
		status = "degraded"
	}
	
	metrics := map[string]interface{}{
		"requests_per_second": perf.RequestsPerSecond,
		"p95_response_time":   perf.P95ResponseTime,
		"error_rate":          perf.ErrorRate,
		"cache_hit_rate":      perf.CacheHitRate,
	}
	
	return models.ComponentCheck{
		Status:      status,
		Message:     "Performance metrics within acceptable ranges",
		LastUpdated: time.Now(),
		Metrics:     metrics,
	}
}

// Background metrics collection
func (bg *BackgroundMetricsCollector) run() {
	ticker := time.NewTicker(bg.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-bg.stopCh:
			return
		case <-ticker.C:
			bg.collectSystemMetrics()
			bg.collectCacheMetrics()
			bg.collectPerformanceMetrics()
			bg.detectAnomalies()
		}
	}
}

func (bg *BackgroundMetricsCollector) collectSystemMetrics() {
	// Collect database connection metrics
	var activeConns int64
	query := `SELECT count(*) FROM pg_stat_activity WHERE state = 'active'`
	if err := bg.db.QueryRow(query).Scan(&activeConns); err == nil {
		bg.collector.SetActiveConnections(float64(activeConns))
		
		bg.collector.stateMutex.Lock()
		bg.collector.metricsState.ActiveConnections = activeConns
		bg.collector.stateMutex.Unlock()
	}
	
	// Collect terminology system counts
	systems := []string{"SNOMED", "RxNorm", "LOINC", "ICD10"}
	for _, system := range systems {
		var count int64
		query := `SELECT COUNT(*) FROM concepts WHERE system = $1`
		if err := bg.db.QueryRow(query, system).Scan(&count); err == nil {
			bg.collector.SetConceptsCount(float64(count))
		}
		
		// Calculate data freshness
		var lastUpdate time.Time
		query = `SELECT MAX(updated_at) FROM concepts WHERE system = $1`
		if err := bg.db.QueryRow(query, system).Scan(&lastUpdate); err == nil {
			hours := time.Since(lastUpdate).Hours()
			bg.collector.SetDataFreshness(system, hours)
		}
	}
}

func (bg *BackgroundMetricsCollector) collectCacheMetrics() {
	if bg.cache == nil {
		return
	}
	
	stats := bg.cache.GetStatistics()
	
	bg.collector.stateMutex.Lock()
	bg.collector.metricsState.CacheStatistics = stats
	bg.collector.metricsState.PerformanceMetrics.CacheHitRate = stats.OverallHitRate
	bg.collector.stateMutex.Unlock()
	
	// Update cache layer metrics
	bg.collector.cacheMemoryUsage.WithLabelValues("L1").Set(float64(stats.L1Stats.Size))
	bg.collector.cacheMemoryUsage.WithLabelValues("L2").Set(float64(stats.L2Stats.Size))
	bg.collector.cacheMemoryUsage.WithLabelValues("L3").Set(float64(stats.L3Stats.Size))
}

func (bg *BackgroundMetricsCollector) collectPerformanceMetrics() {
	// This would collect performance metrics from various sources
	// For now, we'll update the timestamp
	bg.collector.stateMutex.Lock()
	bg.collector.metricsState.LastMetricsUpdate = time.Now()
	bg.collector.stateMutex.Unlock()
}

func (bg *BackgroundMetricsCollector) detectAnomalies() {
	// Implement anomaly detection logic here
	// This could check for unusual patterns in metrics and create alerts
	
	bg.collector.stateMutex.RLock()
	perf := bg.collector.metricsState.PerformanceMetrics
	bg.collector.stateMutex.RUnlock()
	
	var newAlerts []ActiveAlert
	
	// Check error rate
	if perf.ErrorRate > 0.1 { // >10% error rate
		alert := ActiveAlert{
			AlertName:   "high_error_rate",
			Severity:    "critical",
			Description: "Error rate is above acceptable threshold",
			StartTime:   time.Now(),
			Value:       perf.ErrorRate,
			Threshold:   0.1,
			Labels:      map[string]string{"metric": "error_rate"},
		}
		newAlerts = append(newAlerts, alert)
	}
	
	// Check response time
	if perf.P95ResponseTime > 5000 { // >5s P95
		alert := ActiveAlert{
			AlertName:   "high_response_time",
			Severity:    "warning",
			Description: "P95 response time is above acceptable threshold",
			StartTime:   time.Now(),
			Value:       perf.P95ResponseTime,
			Threshold:   5000,
			Labels:      map[string]string{"metric": "p95_response_time"},
		}
		newAlerts = append(newAlerts, alert)
	}
	
	bg.collector.stateMutex.Lock()
	bg.collector.metricsState.AlertsActive = newAlerts
	bg.collector.stateMutex.Unlock()
}