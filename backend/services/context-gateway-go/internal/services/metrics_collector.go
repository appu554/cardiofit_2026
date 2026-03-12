// Package services provides metrics collection for clinical context operations
package services

import (
	"sync"
	"time"
)

// MetricsCollector collects performance and operational metrics
type MetricsCollector struct {
	mu sync.RWMutex
	
	// Snapshot metrics
	snapshotsCreated      int64
	snapshotsAccessed     int64
	snapshotsInvalidated  int64
	snapshotCreationTime  []time.Duration
	
	// Recipe metrics
	recipeUsage           map[string]int64
	recipePerformance     map[string][]time.Duration
	
	// Live fetch metrics
	liveFetchCount        int64
	liveFetchByService    map[string]int64
	liveFetchFieldCounts  []int
	
	// Data source metrics
	dataSourceCalls       map[string]int64
	dataSourceErrors      map[string]int64
	dataSourceLatency     map[string][]time.Duration
	
	// Quality metrics
	completenessScores    []float64
	dataQualityIssues     int64
	
	// Error metrics
	totalErrors           int64
	errorsByType          map[string]int64
	
	// Cache metrics
	cacheHits             int64
	cacheMisses           int64
	cacheEvictions        int64
	
	// Service startup time
	serviceStartTime      time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		recipeUsage:        make(map[string]int64),
		recipePerformance:  make(map[string][]time.Duration),
		liveFetchByService: make(map[string]int64),
		dataSourceCalls:    make(map[string]int64),
		dataSourceErrors:   make(map[string]int64),
		dataSourceLatency:  make(map[string][]time.Duration),
		errorsByType:       make(map[string]int64),
		serviceStartTime:   time.Now(),
	}
}

// RecordSnapshotCreated records a snapshot creation event
func (mc *MetricsCollector) RecordSnapshotCreated(recipeID string, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.snapshotsCreated++
	mc.snapshotCreationTime = append(mc.snapshotCreationTime, duration)
	mc.recipeUsage[recipeID]++
	
	if mc.recipePerformance[recipeID] == nil {
		mc.recipePerformance[recipeID] = make([]time.Duration, 0)
	}
	mc.recipePerformance[recipeID] = append(mc.recipePerformance[recipeID], duration)
	
	// Keep only last 1000 measurements to prevent memory growth
	if len(mc.snapshotCreationTime) > 1000 {
		mc.snapshotCreationTime = mc.snapshotCreationTime[len(mc.snapshotCreationTime)-1000:]
	}
	if len(mc.recipePerformance[recipeID]) > 1000 {
		mc.recipePerformance[recipeID] = mc.recipePerformance[recipeID][len(mc.recipePerformance[recipeID])-1000:]
	}
}

// RecordSnapshotAccessed records a snapshot access event
func (mc *MetricsCollector) RecordSnapshotAccessed() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.snapshotsAccessed++
}

// RecordSnapshotInvalidated records a snapshot invalidation event
func (mc *MetricsCollector) RecordSnapshotInvalidated() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.snapshotsInvalidated++
}

// RecordLiveFetch records a live data fetch event
func (mc *MetricsCollector) RecordLiveFetch(requestingService string, fieldCount int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.liveFetchCount++
	mc.liveFetchByService[requestingService]++
	mc.liveFetchFieldCounts = append(mc.liveFetchFieldCounts, fieldCount)
	
	// Keep only last 1000 measurements
	if len(mc.liveFetchFieldCounts) > 1000 {
		mc.liveFetchFieldCounts = mc.liveFetchFieldCounts[len(mc.liveFetchFieldCounts)-1000:]
	}
}

// RecordDataSourceCall records a data source call
func (mc *MetricsCollector) RecordDataSourceCall(source string, duration time.Duration, success bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.dataSourceCalls[source]++
	
	if mc.dataSourceLatency[source] == nil {
		mc.dataSourceLatency[source] = make([]time.Duration, 0)
	}
	mc.dataSourceLatency[source] = append(mc.dataSourceLatency[source], duration)
	
	if !success {
		mc.dataSourceErrors[source]++
	}
	
	// Keep only last 1000 measurements per source
	if len(mc.dataSourceLatency[source]) > 1000 {
		mc.dataSourceLatency[source] = mc.dataSourceLatency[source][len(mc.dataSourceLatency[source])-1000:]
	}
}

// RecordCompleteness records a data completeness score
func (mc *MetricsCollector) RecordCompleteness(score float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.completenessScores = append(mc.completenessScores, score)
	
	// Keep only last 1000 measurements
	if len(mc.completenessScores) > 1000 {
		mc.completenessScores = mc.completenessScores[len(mc.completenessScores)-1000:]
	}
}

// RecordDataQualityIssue records a data quality issue
func (mc *MetricsCollector) RecordDataQualityIssue() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.dataQualityIssues++
}

// RecordError records an error by type
func (mc *MetricsCollector) RecordError(errorType string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.totalErrors++
	mc.errorsByType[errorType]++
}

// RecordCacheHit records a cache hit
func (mc *MetricsCollector) RecordCacheHit() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.cacheHits++
}

// RecordCacheMiss records a cache miss
func (mc *MetricsCollector) RecordCacheMiss() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.cacheMisses++
}

// RecordCacheEviction records a cache eviction
func (mc *MetricsCollector) RecordCacheEviction() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.cacheEvictions++
}

// GetMetrics returns comprehensive metrics
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	now := time.Now()
	uptime := now.Sub(mc.serviceStartTime)
	
	// Calculate averages and rates
	avgSnapshotCreationTime := mc.calculateAverageDuration(mc.snapshotCreationTime)
	avgCompleteness := mc.calculateAverageFloat64(mc.completenessScores)
	
	// Calculate cache hit ratio
	totalCacheOperations := mc.cacheHits + mc.cacheMisses
	var cacheHitRatio float64
	if totalCacheOperations > 0 {
		cacheHitRatio = float64(mc.cacheHits) / float64(totalCacheOperations)
	}
	
	// Calculate rates (per hour)
	hoursUptime := uptime.Hours()
	var snapshotCreationRate float64
	var liveFetchRate float64
	if hoursUptime > 0 {
		snapshotCreationRate = float64(mc.snapshotsCreated) / hoursUptime
		liveFetchRate = float64(mc.liveFetchCount) / hoursUptime
	}
	
	// Top recipes by usage
	topRecipes := mc.getTopRecipes(5)
	
	// Data source performance
	dataSourcePerf := mc.getDataSourcePerformance()
	
	return map[string]interface{}{
		"service_info": map[string]interface{}{
			"uptime_seconds":         uptime.Seconds(),
			"start_time":             mc.serviceStartTime.Format(time.RFC3339),
			"current_time":           now.Format(time.RFC3339),
		},
		"snapshot_metrics": map[string]interface{}{
			"total_created":          mc.snapshotsCreated,
			"total_accessed":         mc.snapshotsAccessed,
			"total_invalidated":      mc.snapshotsInvalidated,
			"creation_rate_per_hour": snapshotCreationRate,
			"avg_creation_time_ms":   avgSnapshotCreationTime,
		},
		"recipe_metrics": map[string]interface{}{
			"total_recipes_used":     len(mc.recipeUsage),
			"top_recipes":            topRecipes,
			"recipe_usage":           mc.recipeUsage,
		},
		"live_fetch_metrics": map[string]interface{}{
			"total_live_fetches":     mc.liveFetchCount,
			"live_fetch_rate_per_hour": liveFetchRate,
			"live_fetch_by_service":  mc.liveFetchByService,
			"avg_fields_per_fetch":   mc.calculateAverageInt(mc.liveFetchFieldCounts),
		},
		"data_source_metrics": map[string]interface{}{
			"data_source_calls":      mc.dataSourceCalls,
			"data_source_errors":     mc.dataSourceErrors,
			"data_source_performance": dataSourcePerf,
		},
		"quality_metrics": map[string]interface{}{
			"avg_completeness_score": avgCompleteness,
			"data_quality_issues":    mc.dataQualityIssues,
		},
		"cache_metrics": map[string]interface{}{
			"cache_hits":             mc.cacheHits,
			"cache_misses":           mc.cacheMisses,
			"cache_evictions":        mc.cacheEvictions,
			"cache_hit_ratio":        cacheHitRatio,
		},
		"error_metrics": map[string]interface{}{
			"total_errors":           mc.totalErrors,
			"errors_by_type":         mc.errorsByType,
		},
		"timestamp": now.Format(time.RFC3339),
	}
}

// Helper methods for calculations

func (mc *MetricsCollector) calculateAverageDuration(durations []time.Duration) float64 {
	if len(durations) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	
	return float64(total.Milliseconds()) / float64(len(durations))
}

func (mc *MetricsCollector) calculateAverageFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	var sum float64
	for _, v := range values {
		sum += v
	}
	
	return sum / float64(len(values))
}

func (mc *MetricsCollector) calculateAverageInt(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	
	var sum int
	for _, v := range values {
		sum += v
	}
	
	return float64(sum) / float64(len(values))
}

func (mc *MetricsCollector) getTopRecipes(limit int) []map[string]interface{} {
	type recipeUsage struct {
		RecipeID string
		Count    int64
	}
	
	var recipes []recipeUsage
	for recipeID, count := range mc.recipeUsage {
		recipes = append(recipes, recipeUsage{RecipeID: recipeID, Count: count})
	}
	
	// Simple bubble sort for small datasets
	for i := 0; i < len(recipes)-1; i++ {
		for j := 0; j < len(recipes)-i-1; j++ {
			if recipes[j].Count < recipes[j+1].Count {
				recipes[j], recipes[j+1] = recipes[j+1], recipes[j]
			}
		}
	}
	
	// Return top recipes up to limit
	var result []map[string]interface{}
	maxRecipes := limit
	if len(recipes) < maxRecipes {
		maxRecipes = len(recipes)
	}
	
	for i := 0; i < maxRecipes; i++ {
		recipe := recipes[i]
		avgPerf := mc.calculateAverageDuration(mc.recipePerformance[recipe.RecipeID])
		
		result = append(result, map[string]interface{}{
			"recipe_id":           recipe.RecipeID,
			"usage_count":         recipe.Count,
			"avg_performance_ms":  avgPerf,
		})
	}
	
	return result
}

func (mc *MetricsCollector) getDataSourcePerformance() map[string]interface{} {
	performance := make(map[string]interface{})
	
	for source, latencies := range mc.dataSourceLatency {
		calls := mc.dataSourceCalls[source]
		errors := mc.dataSourceErrors[source]
		avgLatency := mc.calculateAverageDuration(latencies)
		
		var errorRate float64
		if calls > 0 {
			errorRate = float64(errors) / float64(calls)
		}
		
		performance[source] = map[string]interface{}{
			"total_calls":       calls,
			"total_errors":      errors,
			"error_rate":        errorRate,
			"avg_latency_ms":    avgLatency,
		}
	}
	
	return performance
}

// Reset resets all metrics (useful for testing or periodic resets)
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.snapshotsCreated = 0
	mc.snapshotsAccessed = 0
	mc.snapshotsInvalidated = 0
	mc.snapshotCreationTime = nil
	
	mc.recipeUsage = make(map[string]int64)
	mc.recipePerformance = make(map[string][]time.Duration)
	
	mc.liveFetchCount = 0
	mc.liveFetchByService = make(map[string]int64)
	mc.liveFetchFieldCounts = nil
	
	mc.dataSourceCalls = make(map[string]int64)
	mc.dataSourceErrors = make(map[string]int64)
	mc.dataSourceLatency = make(map[string][]time.Duration)
	
	mc.completenessScores = nil
	mc.dataQualityIssues = 0
	
	mc.totalErrors = 0
	mc.errorsByType = make(map[string]int64)
	
	mc.cacheHits = 0
	mc.cacheMisses = 0
	mc.cacheEvictions = 0
	
	mc.serviceStartTime = time.Now()
}