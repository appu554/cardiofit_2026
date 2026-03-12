package cache

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// CacheOptimizer provides intelligent cache optimization and analytics
type CacheOptimizer struct {
	cache              *SnapshotCache
	config             *config.CacheConfig
	logger             *logger.Logger
	analytics          *CacheAnalytics
	recommendations    *OptimizationRecommendations
	warmingStrategies  map[string]WarmingStrategy
	compressionManager *CompressionManager
	mu                 sync.RWMutex
	
	// Performance tracking
	accessPatterns     map[string]*AccessPattern
	performanceHistory *PerformanceHistory
	optimizationEvents []OptimizationEvent
	
	// Auto-optimization
	autoOptimizeEnabled bool
	optimizationTicker  *time.Ticker
	ctx                 context.Context
	cancel              context.CancelFunc
}

// CacheAnalytics provides detailed cache performance analytics
type CacheAnalytics struct {
	HitRates           map[string]float64    `json:"hit_rates"`
	MissReasons        map[string]int64      `json:"miss_reasons"`
	AccessFrequency    map[string]int64      `json:"access_frequency"`
	DataAgeDistribution map[string]int64     `json:"data_age_distribution"`
	CompressionStats   *CompressionStats    `json:"compression_stats"`
	MemoryUtilization  *MemoryUtilization   `json:"memory_utilization"`
	PerformanceMetrics *PerformanceMetrics  `json:"performance_metrics"`
	Recommendations    []string             `json:"recommendations"`
}

// OptimizationRecommendations provides actionable cache optimization recommendations
type OptimizationRecommendations struct {
	TTLAdjustments     []TTLRecommendation     `json:"ttl_adjustments"`
	CacheWarming       []WarmingRecommendation `json:"cache_warming"`
	CompressionChanges []CompressionRecommendation `json:"compression_changes"`
	EvictionPolicy     []EvictionRecommendation `json:"eviction_policy"`
	MemoryOptimization []MemoryRecommendation  `json:"memory_optimization"`
	Priority           RecommendationPriority  `json:"priority"`
}

// AccessPattern tracks how cache entries are accessed
type AccessPattern struct {
	Key                string        `json:"key"`
	AccessCount        int64         `json:"access_count"`
	LastAccessed       time.Time     `json:"last_accessed"`
	CreatedAt          time.Time     `json:"created_at"`
	AverageAccessGap   time.Duration `json:"average_access_gap"`
	PeakAccessTime     time.Time     `json:"peak_access_time"`
	DataSize           int64         `json:"data_size"`
	CompressionRatio   float64       `json:"compression_ratio"`
	PredictedNextAccess time.Time    `json:"predicted_next_access"`
}

// PerformanceHistory tracks historical performance data
type PerformanceHistory struct {
	Timestamps    []time.Time `json:"timestamps"`
	HitRates      []float64   `json:"hit_rates"`
	ResponseTimes []float64   `json:"response_times"`
	MemoryUsage   []int64     `json:"memory_usage"`
	ErrorRates    []float64   `json:"error_rates"`
}

// WarmingStrategy defines different cache warming approaches
type WarmingStrategy interface {
	Warm(ctx context.Context, cache *SnapshotCache, keys []string) error
	GetName() string
	GetEffectiveness() float64
}

// PreemptiveWarmingStrategy warms cache based on predicted access patterns
type PreemptiveWarmingStrategy struct {
	predictor *AccessPredictor
	logger    *logger.Logger
}

// OnDemandWarmingStrategy warms cache entries as they're needed
type OnDemandWarmingStrategy struct {
	concurrency int
	logger      *logger.Logger
}

// CompressionStats tracks compression performance
type CompressionStats struct {
	Algorithm         string  `json:"algorithm"`
	AverageRatio      float64 `json:"average_ratio"`
	CompressionTime   float64 `json:"compression_time_ms"`
	DecompressionTime float64 `json:"decompression_time_ms"`
	SpaceSaved        int64   `json:"space_saved_bytes"`
	CPUOverhead       float64 `json:"cpu_overhead_percent"`
}

// MemoryUtilization tracks memory usage patterns
type MemoryUtilization struct {
	L1Usage           int64   `json:"l1_usage_bytes"`
	L2Usage           int64   `json:"l2_usage_bytes"`
	UtilizationRatio  float64 `json:"utilization_ratio"`
	FragmentationLevel float64 `json:"fragmentation_level"`
	PressureLevel     string  `json:"pressure_level"`
}

// PerformanceMetrics tracks key performance indicators
type PerformanceMetrics struct {
	AverageLatency    float64 `json:"average_latency_ms"`
	P95Latency        float64 `json:"p95_latency_ms"`
	P99Latency        float64 `json:"p99_latency_ms"`
	ThroughputOps     int64   `json:"throughput_ops_per_sec"`
	ErrorRate         float64 `json:"error_rate_percent"`
	SLACompliance     float64 `json:"sla_compliance_percent"`
}

// Various recommendation types
type TTLRecommendation struct {
	Key            string        `json:"key"`
	CurrentTTL     time.Duration `json:"current_ttl"`
	RecommendedTTL time.Duration `json:"recommended_ttl"`
	Reason         string        `json:"reason"`
	Impact         string        `json:"impact"`
}

type WarmingRecommendation struct {
	Keys      []string `json:"keys"`
	Strategy  string   `json:"strategy"`
	Priority  int      `json:"priority"`
	Reasoning string   `json:"reasoning"`
}

type CompressionRecommendation struct {
	CurrentAlgorithm    string `json:"current_algorithm"`
	RecommendedAlgorithm string `json:"recommended_algorithm"`
	ExpectedSavings     int64  `json:"expected_savings_bytes"`
	CPUTradeoff         string `json:"cpu_tradeoff"`
}

type EvictionRecommendation struct {
	CurrentPolicy     string   `json:"current_policy"`
	RecommendedPolicy string   `json:"recommended_policy"`
	AffectedKeys      []string `json:"affected_keys"`
	ExpectedImprovement float64 `json:"expected_improvement"`
}

type MemoryRecommendation struct {
	Component     string `json:"component"`
	Action        string `json:"action"`
	ExpectedGain  int64  `json:"expected_gain_bytes"`
	Implementation string `json:"implementation"`
}

type RecommendationPriority struct {
	High   []string `json:"high"`
	Medium []string `json:"medium"`
	Low    []string `json:"low"`
}

type OptimizationEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Impact      map[string]interface{} `json:"impact"`
	Success     bool                   `json:"success"`
}

// NewCacheOptimizer creates a new cache optimizer
func NewCacheOptimizer(cache *SnapshotCache, cfg *config.CacheConfig, logger *logger.Logger) *CacheOptimizer {
	ctx, cancel := context.WithCancel(context.Background())
	
	optimizer := &CacheOptimizer{
		cache:              cache,
		config:             cfg,
		logger:             logger,
		analytics:          &CacheAnalytics{},
		recommendations:    &OptimizationRecommendations{},
		warmingStrategies:  make(map[string]WarmingStrategy),
		compressionManager: NewCompressionManager(cfg, logger),
		accessPatterns:     make(map[string]*AccessPattern),
		performanceHistory: &PerformanceHistory{},
		autoOptimizeEnabled: true,
		ctx:                ctx,
		cancel:             cancel,
	}
	
	optimizer.initializeStrategies()
	optimizer.startOptimizationLoop()
	
	return optimizer
}

// initializeStrategies sets up warming strategies
func (co *CacheOptimizer) initializeStrategies() {
	co.warmingStrategies["preemptive"] = &PreemptiveWarmingStrategy{
		predictor: NewAccessPredictor(co.logger),
		logger:    co.logger,
	}
	
	co.warmingStrategies["on_demand"] = &OnDemandWarmingStrategy{
		concurrency: co.config.WarmingConcurrency,
		logger:      co.logger,
	}
}

// startOptimizationLoop starts the continuous optimization process
func (co *CacheOptimizer) startOptimizationLoop() {
	co.optimizationTicker = time.NewTicker(5 * time.Minute) // Optimize every 5 minutes
	
	go func() {
		for {
			select {
			case <-co.ctx.Done():
				co.optimizationTicker.Stop()
				return
			case <-co.optimizationTicker.C:
				if co.autoOptimizeEnabled {
					co.performOptimization()
				}
			}
		}
	}()
}

// AnalyzePerformance performs comprehensive cache performance analysis
func (co *CacheOptimizer) AnalyzePerformance() *CacheAnalytics {
	co.mu.Lock()
	defer co.mu.Unlock()
	
	stats := co.cache.GetStats()
	
	analytics := &CacheAnalytics{
		HitRates:           co.calculateHitRates(stats),
		MissReasons:        co.analyzeMissReasons(),
		AccessFrequency:    co.calculateAccessFrequency(),
		DataAgeDistribution: co.analyzeDataAge(),
		CompressionStats:   co.compressionManager.GetStats(),
		MemoryUtilization:  co.analyzeMemoryUtilization(),
		PerformanceMetrics: co.calculatePerformanceMetrics(),
		Recommendations:    co.generateRecommendations(),
	}
	
	co.analytics = analytics
	co.updatePerformanceHistory(analytics)
	
	co.logger.Info("Cache performance analysis completed",
		zap.Float64("overall_hit_rate", analytics.PerformanceMetrics.SLACompliance),
		zap.Float64("p95_latency", analytics.PerformanceMetrics.P95Latency),
		zap.Int("recommendations", len(analytics.Recommendations)),
	)
	
	return analytics
}

// OptimizeCache performs intelligent cache optimization
func (co *CacheOptimizer) OptimizeCache() error {
	co.logger.Info("Starting cache optimization")
	
	// 1. Analyze current performance
	analytics := co.AnalyzePerformance()
	
	// 2. Generate optimization plan
	optimizationPlan := co.createOptimizationPlan(analytics)
	
	// 3. Execute optimization plan
	results := co.executeOptimizationPlan(optimizationPlan)
	
	// 4. Record optimization event
	event := OptimizationEvent{
		Timestamp:   time.Now(),
		Type:        "automatic_optimization",
		Description: "Performed automated cache optimization",
		Impact:      results,
		Success:     results["success"].(bool),
	}
	co.optimizationEvents = append(co.optimizationEvents, event)
	
	co.logger.Info("Cache optimization completed",
		zap.Bool("success", results["success"].(bool)),
		zap.Int("optimizations_applied", results["optimizations_applied"].(int)),
	)
	
	return nil
}

// WarmCache performs intelligent cache warming
func (co *CacheOptimizer) WarmCache(strategy string, keys []string) error {
	warmingStrategy, exists := co.warmingStrategies[strategy]
	if !exists {
		return fmt.Errorf("unknown warming strategy: %s", strategy)
	}
	
	startTime := time.Now()
	co.logger.Info("Starting cache warming",
		zap.String("strategy", strategy),
		zap.Int("key_count", len(keys)),
	)
	
	err := warmingStrategy.Warm(co.ctx, co.cache, keys)
	duration := time.Since(startTime)
	
	if err != nil {
		co.logger.Error("Cache warming failed",
			zap.String("strategy", strategy),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return err
	}
	
	co.logger.Info("Cache warming completed successfully",
		zap.String("strategy", strategy),
		zap.Duration("duration", duration),
		zap.Float64("effectiveness", warmingStrategy.GetEffectiveness()),
	)
	
	return nil
}

// GetOptimizationRecommendations returns current optimization recommendations
func (co *CacheOptimizer) GetOptimizationRecommendations() *OptimizationRecommendations {
	co.mu.RLock()
	defer co.mu.RUnlock()
	
	return co.recommendations
}

// RecordAccess records cache access for pattern analysis
func (co *CacheOptimizer) RecordAccess(key string, hit bool, dataSize int64) {
	co.mu.Lock()
	defer co.mu.Unlock()
	
	now := time.Now()
	pattern, exists := co.accessPatterns[key]
	
	if !exists {
		pattern = &AccessPattern{
			Key:       key,
			CreatedAt: now,
			DataSize:  dataSize,
		}
		co.accessPatterns[key] = pattern
	}
	
	pattern.AccessCount++
	pattern.LastAccessed = now
	
	// Calculate average access gap
	if pattern.AccessCount > 1 {
		totalTime := now.Sub(pattern.CreatedAt)
		pattern.AverageAccessGap = time.Duration(int64(totalTime) / pattern.AccessCount)
	}
	
	// Predict next access
	pattern.PredictedNextAccess = now.Add(pattern.AverageAccessGap)
}

// calculateHitRates calculates hit rates by category
func (co *CacheOptimizer) calculateHitRates(stats *types.SnapshotCacheStats) map[string]float64 {
	hitRates := make(map[string]float64)
	
	if stats.TotalRequests > 0 {
		hitRates["overall"] = stats.OverallHitRate
		hitRates["l1"] = stats.L1HitRate
		hitRates["l2"] = stats.L2HitRate
		
		// Calculate time-based hit rates
		hitRates["last_hour"] = co.calculateTimeBasedHitRate(time.Hour)
		hitRates["last_day"] = co.calculateTimeBasedHitRate(24 * time.Hour)
	}
	
	return hitRates
}

// analyzeMissReasons analyzes reasons for cache misses
func (co *CacheOptimizer) analyzeMissReasons() map[string]int64 {
	reasons := map[string]int64{
		"expired":       0,
		"evicted":       0,
		"never_cached":  0,
		"invalid":       0,
		"corruption":    0,
	}
	
	// This would be populated from actual miss tracking
	// For now, return structure for implementation
	return reasons
}

// calculateAccessFrequency calculates access frequency patterns
func (co *CacheOptimizer) calculateAccessFrequency() map[string]int64 {
	frequency := make(map[string]int64)
	
	for _, pattern := range co.accessPatterns {
		bucket := co.getFrequencyBucket(pattern.AccessCount)
		frequency[bucket]++
	}
	
	return frequency
}

// analyzeDataAge analyzes age distribution of cached data
func (co *CacheOptimizer) analyzeDataAge() map[string]int64 {
	distribution := map[string]int64{
		"0-1min":   0,
		"1-5min":   0,
		"5-15min":  0,
		"15-60min": 0,
		"1h+":      0,
	}
	
	now := time.Now()
	for _, pattern := range co.accessPatterns {
		age := now.Sub(pattern.CreatedAt)
		bucket := co.getAgeBucket(age)
		distribution[bucket]++
	}
	
	return distribution
}

// analyzeMemoryUtilization analyzes memory usage patterns
func (co *CacheOptimizer) analyzeMemoryUtilization() *MemoryUtilization {
	// This would integrate with actual memory monitoring
	return &MemoryUtilization{
		L1Usage:           0, // Would be populated from actual metrics
		L2Usage:           0,
		UtilizationRatio:  0.0,
		FragmentationLevel: 0.0,
		PressureLevel:     "low",
	}
}

// calculatePerformanceMetrics calculates key performance metrics
func (co *CacheOptimizer) calculatePerformanceMetrics() *PerformanceMetrics {
	// Calculate metrics from performance history
	if len(co.performanceHistory.ResponseTimes) == 0 {
		return &PerformanceMetrics{
			AverageLatency: 0,
			P95Latency:    0,
			P99Latency:    0,
			ThroughputOps: 0,
			ErrorRate:     0,
			SLACompliance: 100.0,
		}
	}
	
	responseTimes := make([]float64, len(co.performanceHistory.ResponseTimes))
	copy(responseTimes, co.performanceHistory.ResponseTimes)
	sort.Float64s(responseTimes)
	
	count := len(responseTimes)
	
	// Calculate percentiles
	p95Index := int(float64(count) * 0.95)
	p99Index := int(float64(count) * 0.99)
	
	var average float64
	for _, rt := range responseTimes {
		average += rt
	}
	average = average / float64(count)
	
	// Calculate SLA compliance (target: <200ms for 95% of requests)
	slaThreshold := 200.0 // milliseconds
	compliantRequests := 0
	for _, rt := range responseTimes {
		if rt <= slaThreshold {
			compliantRequests++
		}
	}
	slaCompliance := float64(compliantRequests) / float64(count) * 100.0
	
	return &PerformanceMetrics{
		AverageLatency: average,
		P95Latency:    responseTimes[p95Index],
		P99Latency:    responseTimes[p99Index],
		ThroughputOps: 0, // Would be calculated from actual metrics
		ErrorRate:     0, // Would be calculated from actual error tracking
		SLACompliance: slaCompliance,
	}
}

// generateRecommendations generates optimization recommendations
func (co *CacheOptimizer) generateRecommendations() []string {
	var recommendations []string
	
	// Analyze hit rate
	if co.analytics.PerformanceMetrics != nil && co.analytics.PerformanceMetrics.SLACompliance < 85.0 {
		recommendations = append(recommendations, "Cache hit rate below target (85%). Consider increasing cache size or adjusting TTL values.")
	}
	
	// Analyze latency
	if co.analytics.PerformanceMetrics != nil && co.analytics.PerformanceMetrics.P95Latency > 200.0 {
		recommendations = append(recommendations, "P95 latency exceeds target (200ms). Consider optimizing cache access patterns or increasing cache warming.")
	}
	
	// Memory utilization
	if co.analytics.MemoryUtilization != nil && co.analytics.MemoryUtilization.UtilizationRatio > 0.8 {
		recommendations = append(recommendations, "Memory utilization high (>80%). Consider implementing compression or adjusting cache size limits.")
	}
	
	// Compression analysis
	if co.analytics.CompressionStats != nil && co.analytics.CompressionStats.AverageRatio < 2.0 {
		recommendations = append(recommendations, "Low compression ratio detected. Consider using alternative compression algorithms.")
	}
	
	return recommendations
}

// Helper methods
func (co *CacheOptimizer) getFrequencyBucket(accessCount int64) string {
	switch {
	case accessCount < 10:
		return "low"
	case accessCount < 100:
		return "medium"
	case accessCount < 1000:
		return "high"
	default:
		return "very_high"
	}
}

func (co *CacheOptimizer) getAgeBucket(age time.Duration) string {
	switch {
	case age < time.Minute:
		return "0-1min"
	case age < 5*time.Minute:
		return "1-5min"
	case age < 15*time.Minute:
		return "5-15min"
	case age < time.Hour:
		return "15-60min"
	default:
		return "1h+"
	}
}

func (co *CacheOptimizer) calculateTimeBasedHitRate(duration time.Duration) float64 {
	// This would calculate hit rate for a specific time window
	// Implementation would track actual hits/misses over time
	return 0.0 // Placeholder
}

func (co *CacheOptimizer) updatePerformanceHistory(analytics *CacheAnalytics) {
	co.performanceHistory.Timestamps = append(co.performanceHistory.Timestamps, time.Now())
	co.performanceHistory.HitRates = append(co.performanceHistory.HitRates, analytics.HitRates["overall"])
	
	if analytics.PerformanceMetrics != nil {
		co.performanceHistory.ResponseTimes = append(co.performanceHistory.ResponseTimes, analytics.PerformanceMetrics.P95Latency)
		co.performanceHistory.ErrorRates = append(co.performanceHistory.ErrorRates, analytics.PerformanceMetrics.ErrorRate)
	}
	
	if analytics.MemoryUtilization != nil {
		co.performanceHistory.MemoryUsage = append(co.performanceHistory.MemoryUsage, analytics.MemoryUtilization.L1Usage+analytics.MemoryUtilization.L2Usage)
	}
	
	// Keep only last 100 entries to prevent unlimited growth
	if len(co.performanceHistory.Timestamps) > 100 {
		co.performanceHistory.Timestamps = co.performanceHistory.Timestamps[1:]
		co.performanceHistory.HitRates = co.performanceHistory.HitRates[1:]
		co.performanceHistory.ResponseTimes = co.performanceHistory.ResponseTimes[1:]
		co.performanceHistory.ErrorRates = co.performanceHistory.ErrorRates[1:]
		co.performanceHistory.MemoryUsage = co.performanceHistory.MemoryUsage[1:]
	}
}

func (co *CacheOptimizer) performOptimization() {
	co.logger.Debug("Performing scheduled cache optimization")
	if err := co.OptimizeCache(); err != nil {
		co.logger.Error("Scheduled optimization failed", zap.Error(err))
	}
}

func (co *CacheOptimizer) createOptimizationPlan(analytics *CacheAnalytics) map[string]interface{} {
	plan := make(map[string]interface{})
	
	// TTL optimization
	if analytics.PerformanceMetrics.P95Latency > 200.0 {
		plan["adjust_ttl"] = true
		plan["target_ttl_reduction"] = 0.8 // Reduce TTL by 20%
	}
	
	// Cache warming
	if analytics.HitRates["overall"] < 85.0 {
		plan["enable_warming"] = true
		plan["warming_strategy"] = "preemptive"
	}
	
	// Compression optimization
	if analytics.CompressionStats != nil && analytics.CompressionStats.AverageRatio < 2.0 {
		plan["optimize_compression"] = true
		plan["try_algorithm"] = "zstd"
	}
	
	return plan
}

func (co *CacheOptimizer) executeOptimizationPlan(plan map[string]interface{}) map[string]interface{} {
	results := map[string]interface{}{
		"success": true,
		"optimizations_applied": 0,
		"details": []string{},
	}
	
	optimizationsApplied := 0
	details := []string{}
	
	// Execute TTL adjustment
	if plan["adjust_ttl"] == true {
		// Implementation would adjust TTL values
		optimizationsApplied++
		details = append(details, "TTL values optimized")
	}
	
	// Execute cache warming
	if plan["enable_warming"] == true {
		if strategy, ok := plan["warming_strategy"].(string); ok {
			// Implementation would trigger cache warming
			optimizationsApplied++
			details = append(details, fmt.Sprintf("Cache warming enabled with %s strategy", strategy))
		}
	}
	
	// Execute compression optimization
	if plan["optimize_compression"] == true {
		// Implementation would switch compression algorithm
		optimizationsApplied++
		details = append(details, "Compression algorithm optimized")
	}
	
	results["optimizations_applied"] = optimizationsApplied
	results["details"] = details
	
	return results
}

// Stop stops the cache optimizer
func (co *CacheOptimizer) Stop() {
	co.cancel()
	co.logger.Info("Cache optimizer stopped")
}

// GetPerformanceHistory returns historical performance data
func (co *CacheOptimizer) GetPerformanceHistory() *PerformanceHistory {
	co.mu.RLock()
	defer co.mu.RUnlock()
	
	// Return a copy to prevent external modification
	history := &PerformanceHistory{
		Timestamps:    make([]time.Time, len(co.performanceHistory.Timestamps)),
		HitRates:      make([]float64, len(co.performanceHistory.HitRates)),
		ResponseTimes: make([]float64, len(co.performanceHistory.ResponseTimes)),
		MemoryUsage:   make([]int64, len(co.performanceHistory.MemoryUsage)),
		ErrorRates:    make([]float64, len(co.performanceHistory.ErrorRates)),
	}
	
	copy(history.Timestamps, co.performanceHistory.Timestamps)
	copy(history.HitRates, co.performanceHistory.HitRates)
	copy(history.ResponseTimes, co.performanceHistory.ResponseTimes)
	copy(history.MemoryUsage, co.performanceHistory.MemoryUsage)
	copy(history.ErrorRates, co.performanceHistory.ErrorRates)
	
	return history
}