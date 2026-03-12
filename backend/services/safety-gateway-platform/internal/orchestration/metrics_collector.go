package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// ComprehensiveMetricsCollector provides advanced metrics collection and analysis
type ComprehensiveMetricsCollector struct {
	config              *config.OrchestrationMetricsConfig
	logger              *logger.Logger
	
	// Metrics storage
	performanceMetrics  *PerformanceMetrics
	loadMetrics         *LoadMetrics
	routingMetrics      *RoutingMetrics
	batchMetrics        *BatchMetrics
	snapshotMetrics     *SnapshotMetrics
	
	// Historical data
	metricsHistory      []HistoricalSnapshot
	maxHistorySize      int
	
	// Export and reporting
	exportTicker        *time.Ticker
	reportingActive     bool
	
	// Synchronization
	mu                  sync.RWMutex
	historyMu           sync.RWMutex
}

// PerformanceMetrics tracks orchestration performance
type PerformanceMetrics struct {
	RequestsPerSecond     float64           `json:"requests_per_second"`
	AverageResponseTime   time.Duration     `json:"average_response_time"`
	P50ResponseTime       time.Duration     `json:"p50_response_time"`
	P95ResponseTime       time.Duration     `json:"p95_response_time"`
	P99ResponseTime       time.Duration     `json:"p99_response_time"`
	TotalRequests         int64             `json:"total_requests"`
	SuccessfulRequests    int64             `json:"successful_requests"`
	FailedRequests        int64             `json:"failed_requests"`
	ErrorRate             float64           `json:"error_rate"`
	ResponseTimeHistory   []time.Duration   `json:"-"` // Not exported to JSON
	LastUpdated           time.Time         `json:"last_updated"`
	mu                    sync.RWMutex
}

// LoadMetrics tracks system load and resource utilization
type LoadMetrics struct {
	CPUUtilization        float64           `json:"cpu_utilization"`
	MemoryUtilization     float64           `json:"memory_utilization"`
	GoroutineCount        int               `json:"goroutine_count"`
	ActiveConnections     int               `json:"active_connections"`
	QueueDepth            int               `json:"queue_depth"`
	ConcurrentRequests    int               `json:"concurrent_requests"`
	MaxConcurrentRequests int               `json:"max_concurrent_requests"`
	LoadScore             float64           `json:"load_score"`
	ResourcePressure      map[string]float64 `json:"resource_pressure"`
	LastUpdated           time.Time         `json:"last_updated"`
	mu                    sync.RWMutex
}

// RoutingMetrics tracks routing decisions and effectiveness
type RoutingMetrics struct {
	TotalRoutingDecisions    int64                     `json:"total_routing_decisions"`
	RoutingRuleHits          map[string]int64          `json:"routing_rule_hits"`
	TierUtilization          map[string]int64          `json:"tier_utilization"`
	AverageRoutingTime       time.Duration             `json:"average_routing_time"`
	FailedRoutings           int64                     `json:"failed_routings"`
	FallbacksActivated       int64                     `json:"fallbacks_activated"`
	RoutingEfficiency        float64                   `json:"routing_efficiency"`
	EngineSelectionMetrics   map[string]*EngineMetrics `json:"engine_selection_metrics"`
	LastUpdated              time.Time                 `json:"last_updated"`
	mu                       sync.RWMutex
}

// BatchMetrics tracks batch processing performance
type BatchMetrics struct {
	TotalBatches             int64             `json:"total_batches"`
	AverageBatchSize         float64           `json:"average_batch_size"`
	BatchThroughput          float64           `json:"batch_throughput"`
	AverageBatchProcessingTime time.Duration   `json:"average_batch_processing_time"`
	BatchSuccessRate         float64           `json:"batch_success_rate"`
	ParallelismEfficiency    float64           `json:"parallelism_efficiency"`
	CacheHitRatioInBatches   float64           `json:"cache_hit_ratio_in_batches"`
	StrategyUsage            map[string]int64  `json:"strategy_usage"`
	LastUpdated              time.Time         `json:"last_updated"`
	mu                       sync.RWMutex
}

// SnapshotMetrics tracks snapshot-specific performance
type SnapshotMetrics struct {
	SnapshotRetrievals       int64             `json:"snapshot_retrievals"`
	CacheHits                int64             `json:"cache_hits"`
	CacheMisses              int64             `json:"cache_misses"`
	CacheHitRatio            float64           `json:"cache_hit_ratio"`
	AverageRetrievalTime     time.Duration     `json:"average_retrieval_time"`
	ValidationTime           time.Duration     `json:"average_validation_time"`
	SnapshotSizeDistribution map[string]int64  `json:"snapshot_size_distribution"`
	ExpirationEvents         int64             `json:"expiration_events"`
	ValidationFailures       int64             `json:"validation_failures"`
	LastUpdated              time.Time         `json:"last_updated"`
	mu                       sync.RWMutex
}

// HistoricalSnapshot represents a point-in-time metrics snapshot
type HistoricalSnapshot struct {
	Timestamp           time.Time           `json:"timestamp"`
	PerformanceMetrics  *PerformanceMetrics `json:"performance_metrics"`
	LoadMetrics         *LoadMetrics        `json:"load_metrics"`
	RoutingMetrics      *RoutingMetrics     `json:"routing_metrics"`
	BatchMetrics        *BatchMetrics       `json:"batch_metrics"`
	SnapshotMetrics     *SnapshotMetrics    `json:"snapshot_metrics"`
}

// MetricsReport provides a comprehensive metrics report
type MetricsReport struct {
	GeneratedAt         time.Time                    `json:"generated_at"`
	ReportPeriod        time.Duration                `json:"report_period"`
	Summary             *MetricsSummary              `json:"summary"`
	Performance         *PerformanceMetrics          `json:"performance"`
	Load                *LoadMetrics                 `json:"load"`
	Routing             *RoutingMetrics              `json:"routing"`
	Batch               *BatchMetrics                `json:"batch"`
	Snapshot            *SnapshotMetrics             `json:"snapshot"`
	Trends              *MetricsTrends               `json:"trends"`
	Recommendations     []string                     `json:"recommendations"`
	Alerts              []MetricsAlert               `json:"alerts"`
}

// MetricsSummary provides high-level system health metrics
type MetricsSummary struct {
	OverallHealth       string    `json:"overall_health"` // healthy, degraded, critical
	TotalRequests       int64     `json:"total_requests"`
	SuccessRate         float64   `json:"success_rate"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	SystemLoad          float64   `json:"system_load"`
	CacheEfficiency     float64   `json:"cache_efficiency"`
	UptimeSeconds       int64     `json:"uptime_seconds"`
}

// MetricsTrends provides trend analysis
type MetricsTrends struct {
	ResponseTimeTrend   string  `json:"response_time_trend"` // improving, stable, degrading
	ThroughputTrend     string  `json:"throughput_trend"`
	ErrorRateTrend      string  `json:"error_rate_trend"`
	LoadTrend           string  `json:"load_trend"`
	CacheHitRateTrend   string  `json:"cache_hit_rate_trend"`
}

// MetricsAlert represents a metrics-based alert
type MetricsAlert struct {
	Level       string    `json:"level"`       // warning, critical
	Component   string    `json:"component"`   // performance, load, routing, etc.
	Message     string    `json:"message"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewComprehensiveMetricsCollector creates a new metrics collector
func NewComprehensiveMetricsCollector(
	cfg *config.OrchestrationMetricsConfig,
	logger *logger.Logger,
) *ComprehensiveMetricsCollector {
	collector := &ComprehensiveMetricsCollector{
		config:           cfg,
		logger:           logger,
		maxHistorySize:   1000, // Keep last 1000 snapshots
		metricsHistory:   make([]HistoricalSnapshot, 0, 1000),
		
		performanceMetrics: &PerformanceMetrics{
			ResponseTimeHistory: make([]time.Duration, 0, 1000),
			LastUpdated:        time.Now(),
		},
		loadMetrics: &LoadMetrics{
			ResourcePressure: make(map[string]float64),
			LastUpdated:     time.Now(),
		},
		routingMetrics: &RoutingMetrics{
			RoutingRuleHits:        make(map[string]int64),
			TierUtilization:        make(map[string]int64),
			EngineSelectionMetrics: make(map[string]*EngineMetrics),
			LastUpdated:           time.Now(),
		},
		batchMetrics: &BatchMetrics{
			StrategyUsage: make(map[string]int64),
			LastUpdated:   time.Now(),
		},
		snapshotMetrics: &SnapshotMetrics{
			SnapshotSizeDistribution: make(map[string]int64),
			LastUpdated:             time.Now(),
		},
	}

	// Start periodic metrics collection and export
	if cfg.EnableMetrics {
		collector.startPeriodicCollection()
	}

	return collector
}

// RecordRequest records metrics for a processed request
func (c *ComprehensiveMetricsCollector) RecordRequest(
	request *types.SafetyRequest,
	response *types.SafetyResponse,
	processingTime time.Duration,
	engineResults []types.EngineResult,
) {
	c.recordPerformanceMetrics(response, processingTime)
	c.recordRoutingMetrics(request, engineResults)
	c.recordSnapshotMetrics(response)
}

// RecordBatchRequest records metrics for a batch request
func (c *ComprehensiveMetricsCollector) RecordBatchRequest(
	batch *BatchRequest,
	response *BatchResponse,
) {
	c.recordBatchMetrics(batch, response)
}

// recordPerformanceMetrics updates performance-related metrics
func (c *ComprehensiveMetricsCollector) recordPerformanceMetrics(
	response *types.SafetyResponse,
	processingTime time.Duration,
) {
	c.performanceMetrics.mu.Lock()
	defer c.performanceMetrics.mu.Unlock()

	c.performanceMetrics.TotalRequests++
	
	if response.Status != types.SafetyStatusError {
		c.performanceMetrics.SuccessfulRequests++
	} else {
		c.performanceMetrics.FailedRequests++
	}

	// Update error rate
	if c.performanceMetrics.TotalRequests > 0 {
		c.performanceMetrics.ErrorRate = float64(c.performanceMetrics.FailedRequests) / 
			float64(c.performanceMetrics.TotalRequests)
	}

	// Update response time metrics
	c.performanceMetrics.ResponseTimeHistory = append(c.performanceMetrics.ResponseTimeHistory, processingTime)
	
	// Keep only last 1000 response times
	if len(c.performanceMetrics.ResponseTimeHistory) > 1000 {
		c.performanceMetrics.ResponseTimeHistory = c.performanceMetrics.ResponseTimeHistory[1:]
	}

	// Calculate percentiles
	c.calculateResponseTimePercentiles()
	
	// Update average response time (running average)
	if c.performanceMetrics.TotalRequests == 1 {
		c.performanceMetrics.AverageResponseTime = processingTime
	} else {
		totalTime := int64(c.performanceMetrics.AverageResponseTime) * c.performanceMetrics.TotalRequests
		c.performanceMetrics.AverageResponseTime = time.Duration(
			(totalTime + int64(processingTime)) / c.performanceMetrics.TotalRequests,
		)
	}

	c.performanceMetrics.LastUpdated = time.Now()
}

// recordRoutingMetrics updates routing-related metrics
func (c *ComprehensiveMetricsCollector) recordRoutingMetrics(
	request *types.SafetyRequest,
	engineResults []types.EngineResult,
) {
	c.routingMetrics.mu.Lock()
	defer c.routingMetrics.mu.Unlock()

	c.routingMetrics.TotalRoutingDecisions++

	// Track engine usage
	for _, result := range engineResults {
		if metrics, exists := c.routingMetrics.EngineSelectionMetrics[result.EngineID]; exists {
			metrics.RequestCount++
			if result.Error != "" {
				metrics.ErrorCount++
			}
			
			// Update average latency
			if metrics.RequestCount == 1 {
				metrics.AverageLatency = result.Duration
			} else {
				totalTime := int64(metrics.AverageLatency) * metrics.RequestCount
				metrics.AverageLatency = time.Duration(
					(totalTime + int64(result.Duration)) / metrics.RequestCount,
				)
			}
		} else {
			// New engine
			c.routingMetrics.EngineSelectionMetrics[result.EngineID] = &EngineMetrics{
				RequestCount:   1,
				ErrorCount:     0,
				AverageLatency: result.Duration,
				LastUpdated:    time.Now(),
			}
			
			if result.Error != "" {
				c.routingMetrics.EngineSelectionMetrics[result.EngineID].ErrorCount = 1
			}
		}

		// Track tier utilization
		tierKey := string(result.Tier)
		c.routingMetrics.TierUtilization[tierKey]++
	}

	c.routingMetrics.LastUpdated = time.Now()
}

// recordSnapshotMetrics updates snapshot-related metrics
func (c *ComprehensiveMetricsCollector) recordSnapshotMetrics(response *types.SafetyResponse) {
	c.snapshotMetrics.mu.Lock()
	defer c.snapshotMetrics.mu.Unlock()

	// Check if this was a snapshot-based request
	if processingMode, exists := response.Metadata["processing_mode"]; exists {
		if processingMode == "snapshot_based" {
			c.snapshotMetrics.SnapshotRetrievals++
			c.snapshotMetrics.CacheHits++
		} else {
			c.snapshotMetrics.CacheMisses++
		}
	} else {
		c.snapshotMetrics.CacheMisses++
	}

	// Update cache hit ratio
	total := c.snapshotMetrics.CacheHits + c.snapshotMetrics.CacheMisses
	if total > 0 {
		c.snapshotMetrics.CacheHitRatio = float64(c.snapshotMetrics.CacheHits) / float64(total)
	}

	c.snapshotMetrics.LastUpdated = time.Now()
}

// recordBatchMetrics updates batch processing metrics
func (c *ComprehensiveMetricsCollector) recordBatchMetrics(
	batch *BatchRequest,
	response *BatchResponse,
) {
	c.batchMetrics.mu.Lock()
	defer c.batchMetrics.mu.Unlock()

	c.batchMetrics.TotalBatches++

	// Update average batch size
	if c.batchMetrics.TotalBatches == 1 {
		c.batchMetrics.AverageBatchSize = float64(len(batch.Requests))
	} else {
		totalSize := c.batchMetrics.AverageBatchSize * float64(c.batchMetrics.TotalBatches-1)
		c.batchMetrics.AverageBatchSize = (totalSize + float64(len(batch.Requests))) / float64(c.batchMetrics.TotalBatches)
	}

	// Update average processing time
	if c.batchMetrics.TotalBatches == 1 {
		c.batchMetrics.AverageBatchProcessingTime = response.TotalDuration
	} else {
		totalTime := int64(c.batchMetrics.AverageBatchProcessingTime) * c.batchMetrics.TotalBatches
		c.batchMetrics.AverageBatchProcessingTime = time.Duration(
			(totalTime + int64(response.TotalDuration)) / c.batchMetrics.TotalBatches,
		)
	}

	// Record strategy usage
	if strategy, exists := response.Metadata["processing_strategy"]; exists {
		if strategyStr, ok := strategy.(string); ok {
			c.batchMetrics.StrategyUsage[strategyStr]++
		}
	}

	// Calculate success rate
	if len(response.Responses) > 0 {
		successCount := float64(response.Summary.SuccessfulResults + response.Summary.WarningResults)
		c.batchMetrics.BatchSuccessRate = successCount / float64(len(response.Responses))
	}

	c.batchMetrics.LastUpdated = time.Now()
}

// GenerateReport generates a comprehensive metrics report
func (c *ComprehensiveMetricsCollector) GenerateReport() *MetricsReport {
	c.mu.RLock()
	defer c.mu.RUnlock()

	report := &MetricsReport{
		GeneratedAt:  time.Now(),
		ReportPeriod: c.config.MetricsInterval,
		Performance:  c.copyPerformanceMetrics(),
		Load:         c.copyLoadMetrics(),
		Routing:      c.copyRoutingMetrics(),
		Batch:        c.copyBatchMetrics(),
		Snapshot:     c.copySnapshotMetrics(),
	}

	// Generate summary
	report.Summary = c.generateSummary()

	// Analyze trends
	report.Trends = c.analyzeTrends()

	// Generate alerts
	report.Alerts = c.generateAlerts()

	// Generate recommendations
	report.Recommendations = c.generateRecommendations()

	return report
}

// Helper methods

func (c *ComprehensiveMetricsCollector) calculateResponseTimePercentiles() {
	if len(c.performanceMetrics.ResponseTimeHistory) == 0 {
		return
	}

	// Simple percentile calculation (could be optimized with more sophisticated algorithms)
	history := make([]time.Duration, len(c.performanceMetrics.ResponseTimeHistory))
	copy(history, c.performanceMetrics.ResponseTimeHistory)

	// Sort for percentile calculation
	for i := 0; i < len(history)-1; i++ {
		for j := i + 1; j < len(history); j++ {
			if history[i] > history[j] {
				history[i], history[j] = history[j], history[i]
			}
		}
	}

	n := len(history)
	c.performanceMetrics.P50ResponseTime = history[n*50/100]
	c.performanceMetrics.P95ResponseTime = history[n*95/100]
	c.performanceMetrics.P99ResponseTime = history[n*99/100]
}

func (c *ComprehensiveMetricsCollector) copyPerformanceMetrics() *PerformanceMetrics {
	c.performanceMetrics.mu.RLock()
	defer c.performanceMetrics.mu.RUnlock()

	return &PerformanceMetrics{
		RequestsPerSecond:   c.performanceMetrics.RequestsPerSecond,
		AverageResponseTime: c.performanceMetrics.AverageResponseTime,
		P50ResponseTime:     c.performanceMetrics.P50ResponseTime,
		P95ResponseTime:     c.performanceMetrics.P95ResponseTime,
		P99ResponseTime:     c.performanceMetrics.P99ResponseTime,
		TotalRequests:       c.performanceMetrics.TotalRequests,
		SuccessfulRequests:  c.performanceMetrics.SuccessfulRequests,
		FailedRequests:      c.performanceMetrics.FailedRequests,
		ErrorRate:           c.performanceMetrics.ErrorRate,
		LastUpdated:         c.performanceMetrics.LastUpdated,
	}
}

func (c *ComprehensiveMetricsCollector) copyLoadMetrics() *LoadMetrics {
	c.loadMetrics.mu.RLock()
	defer c.loadMetrics.mu.RUnlock()

	resourcePressure := make(map[string]float64)
	for k, v := range c.loadMetrics.ResourcePressure {
		resourcePressure[k] = v
	}

	return &LoadMetrics{
		CPUUtilization:        c.loadMetrics.CPUUtilization,
		MemoryUtilization:     c.loadMetrics.MemoryUtilization,
		GoroutineCount:        c.loadMetrics.GoroutineCount,
		ActiveConnections:     c.loadMetrics.ActiveConnections,
		QueueDepth:            c.loadMetrics.QueueDepth,
		ConcurrentRequests:    c.loadMetrics.ConcurrentRequests,
		MaxConcurrentRequests: c.loadMetrics.MaxConcurrentRequests,
		LoadScore:             c.loadMetrics.LoadScore,
		ResourcePressure:      resourcePressure,
		LastUpdated:           c.loadMetrics.LastUpdated,
	}
}

func (c *ComprehensiveMetricsCollector) copyRoutingMetrics() *RoutingMetrics {
	c.routingMetrics.mu.RLock()
	defer c.routingMetrics.mu.RUnlock()

	routingRuleHits := make(map[string]int64)
	for k, v := range c.routingMetrics.RoutingRuleHits {
		routingRuleHits[k] = v
	}

	tierUtilization := make(map[string]int64)
	for k, v := range c.routingMetrics.TierUtilization {
		tierUtilization[k] = v
	}

	engineMetrics := make(map[string]*EngineMetrics)
	for k, v := range c.routingMetrics.EngineSelectionMetrics {
		engineMetrics[k] = &EngineMetrics{
			RequestCount:   v.RequestCount,
			ErrorCount:     v.ErrorCount,
			AverageLatency: v.AverageLatency,
			LastUpdated:    v.LastUpdated,
		}
	}

	return &RoutingMetrics{
		TotalRoutingDecisions:  c.routingMetrics.TotalRoutingDecisions,
		RoutingRuleHits:        routingRuleHits,
		TierUtilization:        tierUtilization,
		AverageRoutingTime:     c.routingMetrics.AverageRoutingTime,
		FailedRoutings:         c.routingMetrics.FailedRoutings,
		FallbacksActivated:     c.routingMetrics.FallbacksActivated,
		RoutingEfficiency:      c.routingMetrics.RoutingEfficiency,
		EngineSelectionMetrics: engineMetrics,
		LastUpdated:            c.routingMetrics.LastUpdated,
	}
}

func (c *ComprehensiveMetricsCollector) copyBatchMetrics() *BatchMetrics {
	c.batchMetrics.mu.RLock()
	defer c.batchMetrics.mu.RUnlock()

	strategyUsage := make(map[string]int64)
	for k, v := range c.batchMetrics.StrategyUsage {
		strategyUsage[k] = v
	}

	return &BatchMetrics{
		TotalBatches:              c.batchMetrics.TotalBatches,
		AverageBatchSize:          c.batchMetrics.AverageBatchSize,
		BatchThroughput:           c.batchMetrics.BatchThroughput,
		AverageBatchProcessingTime: c.batchMetrics.AverageBatchProcessingTime,
		BatchSuccessRate:          c.batchMetrics.BatchSuccessRate,
		ParallelismEfficiency:     c.batchMetrics.ParallelismEfficiency,
		CacheHitRatioInBatches:    c.batchMetrics.CacheHitRatioInBatches,
		StrategyUsage:             strategyUsage,
		LastUpdated:               c.batchMetrics.LastUpdated,
	}
}

func (c *ComprehensiveMetricsCollector) copySnapshotMetrics() *SnapshotMetrics {
	c.snapshotMetrics.mu.RLock()
	defer c.snapshotMetrics.mu.RUnlock()

	sizeDistribution := make(map[string]int64)
	for k, v := range c.snapshotMetrics.SnapshotSizeDistribution {
		sizeDistribution[k] = v
	}

	return &SnapshotMetrics{
		SnapshotRetrievals:       c.snapshotMetrics.SnapshotRetrievals,
		CacheHits:                c.snapshotMetrics.CacheHits,
		CacheMisses:              c.snapshotMetrics.CacheMisses,
		CacheHitRatio:            c.snapshotMetrics.CacheHitRatio,
		AverageRetrievalTime:     c.snapshotMetrics.AverageRetrievalTime,
		ValidationTime:           c.snapshotMetrics.ValidationTime,
		SnapshotSizeDistribution: sizeDistribution,
		ExpirationEvents:         c.snapshotMetrics.ExpirationEvents,
		ValidationFailures:       c.snapshotMetrics.ValidationFailures,
		LastUpdated:              c.snapshotMetrics.LastUpdated,
	}
}

func (c *ComprehensiveMetricsCollector) generateSummary() *MetricsSummary {
	performance := c.copyPerformanceMetrics()
	load := c.copyLoadMetrics()
	snapshot := c.copySnapshotMetrics()

	// Determine overall health
	health := "healthy"
	if performance.ErrorRate > 0.05 || load.LoadScore > 0.9 {
		health = "degraded"
	}
	if performance.ErrorRate > 0.15 || load.LoadScore > 0.95 {
		health = "critical"
	}

	return &MetricsSummary{
		OverallHealth:       health,
		TotalRequests:       performance.TotalRequests,
		SuccessRate:         1.0 - performance.ErrorRate,
		AverageResponseTime: performance.AverageResponseTime,
		SystemLoad:          load.LoadScore,
		CacheEfficiency:     snapshot.CacheHitRatio,
		UptimeSeconds:       int64(time.Since(time.Now().Add(-time.Hour)).Seconds()), // Placeholder
	}
}

func (c *ComprehensiveMetricsCollector) analyzeTrends() *MetricsTrends {
	// Simplified trend analysis - in production, this would use more sophisticated algorithms
	return &MetricsTrends{
		ResponseTimeTrend: "stable",
		ThroughputTrend:   "stable", 
		ErrorRateTrend:    "stable",
		LoadTrend:         "stable",
		CacheHitRateTrend: "stable",
	}
}

func (c *ComprehensiveMetricsCollector) generateAlerts() []MetricsAlert {
	var alerts []MetricsAlert
	performance := c.copyPerformanceMetrics()
	load := c.copyLoadMetrics()

	// Error rate alert
	if performance.ErrorRate > 0.05 {
		level := "warning"
		if performance.ErrorRate > 0.15 {
			level = "critical"
		}
		
		alerts = append(alerts, MetricsAlert{
			Level:     level,
			Component: "performance",
			Message:   fmt.Sprintf("High error rate detected: %.2f%%", performance.ErrorRate*100),
			Value:     performance.ErrorRate,
			Threshold: 0.05,
			Timestamp: time.Now(),
		})
	}

	// High load alert
	if load.LoadScore > 0.8 {
		level := "warning"
		if load.LoadScore > 0.95 {
			level = "critical"
		}
		
		alerts = append(alerts, MetricsAlert{
			Level:     level,
			Component: "load",
			Message:   fmt.Sprintf("High system load detected: %.2f", load.LoadScore),
			Value:     load.LoadScore,
			Threshold: 0.8,
			Timestamp: time.Now(),
		})
	}

	return alerts
}

func (c *ComprehensiveMetricsCollector) generateRecommendations() []string {
	var recommendations []string
	performance := c.copyPerformanceMetrics()
	load := c.copyLoadMetrics()
	snapshot := c.copySnapshotMetrics()

	if performance.ErrorRate > 0.05 {
		recommendations = append(recommendations, "Consider implementing circuit breakers to prevent cascade failures")
	}

	if load.LoadScore > 0.8 {
		recommendations = append(recommendations, "Scale up system resources or implement load balancing")
	}

	if snapshot.CacheHitRatio < 0.7 {
		recommendations = append(recommendations, "Optimize cache configuration to improve hit ratio")
	}

	if performance.P95ResponseTime > 1*time.Second {
		recommendations = append(recommendations, "Investigate performance bottlenecks in slow requests")
	}

	return recommendations
}

func (c *ComprehensiveMetricsCollector) startPeriodicCollection() {
	if c.config.ExportJSON {
		c.exportTicker = time.NewTicker(c.config.MetricsInterval)
		c.reportingActive = true

		go func() {
			for {
				select {
				case <-c.exportTicker.C:
					if err := c.exportMetricsToJSON(); err != nil {
						c.logger.Error("Failed to export metrics to JSON", zap.Error(err))
					}
				}
			}
		}()
	}
}

func (c *ComprehensiveMetricsCollector) exportMetricsToJSON() error {
	report := c.GenerateReport()
	
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics report: %w", err)
	}

	if err := os.WriteFile(c.config.JSONExportPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write metrics file: %w", err)
	}

	return nil
}

// Cleanup stops the metrics collector and cleans up resources
func (c *ComprehensiveMetricsCollector) Cleanup() {
	if c.exportTicker != nil {
		c.exportTicker.Stop()
	}
	c.reportingActive = false
}