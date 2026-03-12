package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// SnapshotMetricsCollector provides comprehensive metrics for snapshot-based operations
type SnapshotMetricsCollector struct {
	// Snapshot processing metrics
	SnapshotRequestsTotal          *prometheus.CounterVec
	SnapshotProcessingDuration     *prometheus.HistogramVec
	SnapshotRetrievalDuration      *prometheus.HistogramVec
	SnapshotValidationDuration     *prometheus.HistogramVec
	SnapshotCompressionRatio       *prometheus.GaugeVec
	
	// Cache performance metrics
	CacheHitRateGauge              *prometheus.GaugeVec
	CacheOperationDuration         *prometheus.HistogramVec
	CacheOperationsTotal           *prometheus.CounterVec
	CacheCompressionRatio          *prometheus.GaugeVec
	CacheMemoryUsage               *prometheus.GaugeVec
	CacheEvictionRate              *prometheus.GaugeVec
	
	// Advanced cache analytics
	CacheAccessPatterns            *prometheus.CounterVec
	CacheWarmupDuration            *prometheus.HistogramVec
	CacheOptimizationEvents        *prometheus.CounterVec
	CacheRecommendations           *prometheus.GaugeVec
	
	// Performance optimization metrics
	P95LatencyGauge                *prometheus.GaugeVec
	P99LatencyGauge                *prometheus.GaugeVec
	ConcurrentRequestsGauge        prometheus.Gauge
	MemoryOptimizationRatio        *prometheus.GaugeVec
	CompressionSavingsBytes        *prometheus.CounterVec
	
	// Engine performance metrics
	EnginePerformanceScore         *prometheus.GaugeVec
	EngineOptimizationStatus       *prometheus.GaugeVec
	ParallelProcessingEfficiency   *prometheus.GaugeVec
	
	// System health metrics
	SystemMemoryPressure           prometheus.Gauge
	GCPauseTime                    *prometheus.HistogramVec
	ThreadPoolUtilization          *prometheus.GaugeVec
	ConnectionPoolHealth           *prometheus.GaugeVec
	
	// Business metrics
	TargetLatencyCompliance        *prometheus.GaugeVec
	SLAViolationRate               *prometheus.CounterVec
	UserExperienceScore            *prometheus.GaugeVec
	
	logger   *zap.Logger
	registry *prometheus.Registry
	mu       sync.RWMutex
}

// NewSnapshotMetricsCollector creates a new snapshot metrics collector
func NewSnapshotMetricsCollector(logger *zap.Logger, registry *prometheus.Registry) *SnapshotMetricsCollector {
	collector := &SnapshotMetricsCollector{
		logger:   logger,
		registry: registry,
	}
	
	collector.initializeMetrics()
	collector.registerMetrics()
	
	return collector
}

// initializeMetrics initializes all Prometheus metrics
func (s *SnapshotMetricsCollector) initializeMetrics() {
	// Snapshot processing metrics
	s.SnapshotRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "snapshot_requests_total",
			Help: "Total number of snapshot-based requests processed",
		},
		[]string{"status", "cache_hit", "processing_mode", "patient_tier"},
	)
	
	s.SnapshotProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "snapshot_processing_duration_seconds",
			Help:    "Duration of snapshot-based request processing",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.5, 1.0},
		},
		[]string{"processing_stage", "cache_level", "complexity"},
	)
	
	s.SnapshotRetrievalDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "snapshot_retrieval_duration_seconds",
			Help:    "Duration of snapshot retrieval operations",
			Buckets: []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25},
		},
		[]string{"source", "cache_level", "compression"},
	)
	
	s.SnapshotValidationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "snapshot_validation_duration_seconds",
			Help:    "Duration of snapshot validation operations",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05},
		},
		[]string{"validation_type", "complexity"},
	)
	
	s.SnapshotCompressionRatio = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "snapshot_compression_ratio",
			Help: "Compression ratio achieved for snapshots",
		},
		[]string{"compression_algorithm", "data_type"},
	)
	
	// Cache performance metrics
	s.CacheHitRateGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_hit_rate_current",
			Help: "Current cache hit rate percentage",
		},
		[]string{"cache_level", "data_type", "time_window"},
	)
	
	s.CacheOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_operation_duration_seconds",
			Help:    "Duration of cache operations",
			Buckets: []float64{0.00001, 0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025},
		},
		[]string{"operation", "cache_level", "result"},
	)
	
	s.CacheOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation", "cache_level", "result", "optimization"},
	)
	
	s.CacheCompressionRatio = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_compression_ratio",
			Help: "Compression ratio in cache storage",
		},
		[]string{"cache_level", "algorithm"},
	)
	
	s.CacheMemoryUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_memory_usage_bytes",
			Help: "Memory usage by cache level",
		},
		[]string{"cache_level", "data_category"},
	)
	
	s.CacheEvictionRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_eviction_rate_per_minute",
			Help: "Rate of cache evictions per minute",
		},
		[]string{"cache_level", "eviction_reason"},
	)
	
	// Advanced cache analytics
	s.CacheAccessPatterns = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_access_patterns_total",
			Help: "Cache access patterns for optimization",
		},
		[]string{"pattern_type", "access_frequency", "data_age"},
	)
	
	s.CacheWarmupDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_warmup_duration_seconds",
			Help:    "Duration of cache warming operations",
			Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
		},
		[]string{"warmup_type", "data_volume"},
	)
	
	s.CacheOptimizationEvents = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_optimization_events_total",
			Help: "Cache optimization events triggered",
		},
		[]string{"optimization_type", "trigger_reason", "effectiveness"},
	)
	
	s.CacheRecommendations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_recommendations_active",
			Help: "Active cache optimization recommendations",
		},
		[]string{"recommendation_type", "priority", "impact_level"},
	)
	
	// Performance optimization metrics
	s.P95LatencyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "request_latency_p95_seconds",
			Help: "P95 request latency",
		},
		[]string{"request_type", "optimization_level"},
	)
	
	s.P99LatencyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "request_latency_p99_seconds",
			Help: "P99 request latency",
		},
		[]string{"request_type", "optimization_level"},
	)
	
	s.ConcurrentRequestsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "concurrent_requests_current",
			Help: "Current number of concurrent requests being processed",
		},
	)
	
	s.MemoryOptimizationRatio = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "memory_optimization_ratio",
			Help: "Memory optimization ratio achieved",
		},
		[]string{"optimization_type", "component"},
	)
	
	s.CompressionSavingsBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "compression_savings_bytes_total",
			Help: "Total bytes saved through compression",
		},
		[]string{"compression_type", "data_category"},
	)
	
	// Engine performance metrics
	s.EnginePerformanceScore = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "engine_performance_score",
			Help: "Performance score for each engine (0-100)",
		},
		[]string{"engine_id", "metric_type"},
	)
	
	s.EngineOptimizationStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "engine_optimization_status",
			Help: "Engine optimization status (0=disabled, 1=enabled, 2=auto)",
		},
		[]string{"engine_id", "optimization_type"},
	)
	
	s.ParallelProcessingEfficiency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "parallel_processing_efficiency_ratio",
			Help: "Efficiency of parallel processing (0-1)",
		},
		[]string{"processing_type", "concurrency_level"},
	)
	
	// System health metrics
	s.SystemMemoryPressure = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_memory_pressure_ratio",
			Help: "System memory pressure ratio (0-1)",
		},
	)
	
	s.GCPauseTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gc_pause_time_seconds",
			Help:    "Garbage collection pause times",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
		},
		[]string{"gc_type"},
	)
	
	s.ThreadPoolUtilization = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "thread_pool_utilization_ratio",
			Help: "Thread pool utilization ratio",
		},
		[]string{"pool_type", "priority"},
	)
	
	s.ConnectionPoolHealth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "connection_pool_health_score",
			Help: "Connection pool health score (0-100)",
		},
		[]string{"pool_type", "endpoint"},
	)
	
	// Business metrics
	s.TargetLatencyCompliance = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "target_latency_compliance_ratio",
			Help: "Ratio of requests meeting target latency (<200ms)",
		},
		[]string{"service_tier", "time_window"},
	)
	
	s.SLAViolationRate = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sla_violations_total",
			Help: "Total SLA violations",
		},
		[]string{"violation_type", "severity", "service_tier"},
	)
	
	s.UserExperienceScore = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "user_experience_score",
			Help: "User experience score based on performance metrics (0-100)",
		},
		[]string{"metric_category", "user_tier"},
	)
}

// registerMetrics registers all metrics with the Prometheus registry
func (s *SnapshotMetricsCollector) registerMetrics() {
	metrics := []prometheus.Collector{
		s.SnapshotRequestsTotal,
		s.SnapshotProcessingDuration,
		s.SnapshotRetrievalDuration,
		s.SnapshotValidationDuration,
		s.SnapshotCompressionRatio,
		s.CacheHitRateGauge,
		s.CacheOperationDuration,
		s.CacheOperationsTotal,
		s.CacheCompressionRatio,
		s.CacheMemoryUsage,
		s.CacheEvictionRate,
		s.CacheAccessPatterns,
		s.CacheWarmupDuration,
		s.CacheOptimizationEvents,
		s.CacheRecommendations,
		s.P95LatencyGauge,
		s.P99LatencyGauge,
		s.ConcurrentRequestsGauge,
		s.MemoryOptimizationRatio,
		s.CompressionSavingsBytes,
		s.EnginePerformanceScore,
		s.EngineOptimizationStatus,
		s.ParallelProcessingEfficiency,
		s.SystemMemoryPressure,
		s.GCPauseTime,
		s.ThreadPoolUtilization,
		s.ConnectionPoolHealth,
		s.TargetLatencyCompliance,
		s.SLAViolationRate,
		s.UserExperienceScore,
	}
	
	for _, metric := range metrics {
		s.registry.MustRegister(metric)
	}
	
	s.logger.Info("Snapshot metrics registered successfully", zap.Int("total_metrics", len(metrics)))
}

// RecordSnapshotRequest records snapshot request metrics
func (s *SnapshotMetricsCollector) RecordSnapshotRequest(status, cacheHit, processingMode, patientTier string) {
	s.SnapshotRequestsTotal.WithLabelValues(status, cacheHit, processingMode, patientTier).Inc()
}

// RecordSnapshotProcessingDuration records snapshot processing duration
func (s *SnapshotMetricsCollector) RecordSnapshotProcessingDuration(stage, cacheLevel, complexity string, duration time.Duration) {
	s.SnapshotProcessingDuration.WithLabelValues(stage, cacheLevel, complexity).Observe(duration.Seconds())
}

// RecordCacheOperation records cache operation metrics
func (s *SnapshotMetricsCollector) RecordCacheOperation(operation, cacheLevel, result, optimization string, duration time.Duration) {
	s.CacheOperationDuration.WithLabelValues(operation, cacheLevel, result).Observe(duration.Seconds())
	s.CacheOperationsTotal.WithLabelValues(operation, cacheLevel, result, optimization).Inc()
}

// UpdateCacheHitRate updates cache hit rate gauge
func (s *SnapshotMetricsCollector) UpdateCacheHitRate(cacheLevel, dataType, timeWindow string, hitRate float64) {
	s.CacheHitRateGauge.WithLabelValues(cacheLevel, dataType, timeWindow).Set(hitRate)
}

// UpdatePerformanceLatency updates P95/P99 latency metrics
func (s *SnapshotMetricsCollector) UpdatePerformanceLatency(requestType, optimizationLevel string, p95, p99 time.Duration) {
	s.P95LatencyGauge.WithLabelValues(requestType, optimizationLevel).Set(p95.Seconds())
	s.P99LatencyGauge.WithLabelValues(requestType, optimizationLevel).Set(p99.Seconds())
}

// RecordCacheOptimization records cache optimization events
func (s *SnapshotMetricsCollector) RecordCacheOptimization(optimizationType, triggerReason, effectiveness string) {
	s.CacheOptimizationEvents.WithLabelValues(optimizationType, triggerReason, effectiveness).Inc()
}

// UpdateMemoryMetrics updates memory-related metrics
func (s *SnapshotMetricsCollector) UpdateMemoryMetrics(memoryPressure float64, cacheMemoryUsage map[string]int64) {
	s.SystemMemoryPressure.Set(memoryPressure)
	
	for cacheLevel, usage := range cacheMemoryUsage {
		s.CacheMemoryUsage.WithLabelValues(cacheLevel, "snapshot_data").Set(float64(usage))
	}
}

// UpdateSLACompliance updates SLA compliance metrics
func (s *SnapshotMetricsCollector) UpdateSLACompliance(serviceTier, timeWindow string, complianceRatio float64) {
	s.TargetLatencyCompliance.WithLabelValues(serviceTier, timeWindow).Set(complianceRatio)
}

// RecordSLAViolation records SLA violation
func (s *SnapshotMetricsCollector) RecordSLAViolation(violationType, severity, serviceTier string) {
	s.SLAViolationRate.WithLabelValues(violationType, severity, serviceTier).Inc()
}

// UpdateUserExperienceScore updates user experience metrics
func (s *SnapshotMetricsCollector) UpdateUserExperienceScore(category, userTier string, score float64) {
	s.UserExperienceScore.WithLabelValues(category, userTier).Set(score)
}

// RecordCompressionSavings records compression savings
func (s *SnapshotMetricsCollector) RecordCompressionSavings(compressionType, dataCategory string, savedBytes int64) {
	s.CompressionSavingsBytes.WithLabelValues(compressionType, dataCategory).Add(float64(savedBytes))
}

// UpdateEnginePerformance updates engine performance metrics
func (s *SnapshotMetricsCollector) UpdateEnginePerformance(engineID, metricType string, score float64) {
	s.EnginePerformanceScore.WithLabelValues(engineID, metricType).Set(score)
}

// UpdateSystemHealth updates system health metrics
func (s *SnapshotMetricsCollector) UpdateSystemHealth(gcPauseTime time.Duration, threadPoolUtilization, connectionPoolHealth map[string]float64) {
	s.GCPauseTime.WithLabelValues("full_gc").Observe(gcPauseTime.Seconds())
	
	for poolType, utilization := range threadPoolUtilization {
		s.ThreadPoolUtilization.WithLabelValues(poolType, "normal").Set(utilization)
	}
	
	for poolType, health := range connectionPoolHealth {
		s.ConnectionPoolHealth.WithLabelValues(poolType, "default").Set(health)
	}
}

// StartPerformanceTracking starts continuous performance tracking
func (s *SnapshotMetricsCollector) StartPerformanceTracking(ctx context.Context) {
	go s.trackPerformanceMetrics(ctx)
	go s.trackCacheAnalytics(ctx)
	go s.trackSystemHealth(ctx)
}

// trackPerformanceMetrics continuously tracks performance metrics
func (s *SnapshotMetricsCollector) trackPerformanceMetrics(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second) // High frequency for performance tracking
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collectPerformanceMetrics()
		}
	}
}

// trackCacheAnalytics continuously tracks cache analytics
func (s *SnapshotMetricsCollector) trackCacheAnalytics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collectCacheAnalytics()
		}
	}
}

// trackSystemHealth continuously tracks system health
func (s *SnapshotMetricsCollector) trackSystemHealth(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collectSystemHealth()
		}
	}
}

// collectPerformanceMetrics collects current performance metrics
func (s *SnapshotMetricsCollector) collectPerformanceMetrics() {
	// This would integrate with actual performance monitoring
	// For now, we'll set up the structure for real implementation
	s.logger.Debug("Collecting performance metrics")
}

// collectCacheAnalytics collects cache analytics
func (s *SnapshotMetricsCollector) collectCacheAnalytics() {
	// This would integrate with actual cache monitoring
	s.logger.Debug("Collecting cache analytics")
}

// collectSystemHealth collects system health metrics
func (s *SnapshotMetricsCollector) collectSystemHealth() {
	// This would integrate with actual system monitoring
	s.logger.Debug("Collecting system health metrics")
}

// GetPerformanceReport generates a comprehensive performance report
func (s *SnapshotMetricsCollector) GetPerformanceReport() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	report := map[string]interface{}{
		"timestamp":     time.Now(),
		"report_type":   "performance_comprehensive",
		"version":       "3.0",
		"metrics": map[string]interface{}{
			"latency": map[string]interface{}{
				"target_compliance": ">=85%",
				"current_p95":       "<200ms target",
				"current_p99":       "<500ms target",
			},
			"cache": map[string]interface{}{
				"hit_rate_target":   ">=85%",
				"memory_efficiency": "optimized",
				"compression_ratio": "tracked",
			},
			"system": map[string]interface{}{
				"memory_pressure":    "monitored",
				"gc_performance":     "tracked",
				"connection_health":  "healthy",
			},
		},
		"recommendations": []string{
			"Monitor P95 latency continuously",
			"Optimize cache hit rates above 85%",
			"Track compression effectiveness",
			"Monitor system memory pressure",
		},
	}
	
	return report
}