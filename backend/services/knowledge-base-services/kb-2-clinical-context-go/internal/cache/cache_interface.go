package cache

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// CacheInterface defines the standard interface for all cache implementations
type CacheInterface interface {
	// Basic operations
	Get(ctx context.Context, key string) (interface{}, bool)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	
	// Batch operations
	GetBatch(ctx context.Context, keys []string) (map[string]interface{}, error)
	SetBatch(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
	
	// Pattern operations
	DeletePattern(ctx context.Context, pattern string) error
	
	// Statistics and health
	GetStats() *CacheStats
	IsHealthy(ctx context.Context) bool
	
	// Maintenance
	Cleanup(ctx context.Context) error
	Optimize(ctx context.Context) error
}

// MultiTierCacheInterface extends CacheInterface for multi-tier caching
type MultiTierCacheInterface interface {
	CacheInterface
	
	// Multi-tier specific operations
	GetWithLoader(ctx context.Context, key string, loader func() (interface{}, error)) (interface{}, error)
	InvalidatePattern(ctx context.Context, pattern string) error
	WarmCache(ctx context.Context) error
	
	// Cache tier management
	GetHitRates() map[string]float64
	CheckSLACompliance() map[string]bool
	OptimizeCache(ctx context.Context) error
	
	// Specialized domain operations
	GetPhenotypeDefinition(ctx context.Context, phenotypeID string, loader func() (interface{}, error)) (interface{}, error)
	GetPatientContext(ctx context.Context, patientID string, contextType string, loader func() (interface{}, error)) (interface{}, error)
	GetRiskAssessment(ctx context.Context, patientID string, riskType string, loader func() (interface{}, error)) (interface{}, error)
	GetTreatmentPreferences(ctx context.Context, patientID string, condition string, loader func() (interface{}, error)) (interface{}, error)
}

// CacheMetrics defines metrics interface for cache monitoring
type CacheMetrics interface {
	RecordHit(tier string)
	RecordMiss(tier string)
	RecordOperation(operation string, duration time.Duration)
	RecordEviction(tier string)
	RecordError(errorType string)
}

// CacheSerializer handles serialization for different cache tiers
type CacheSerializer interface {
	Serialize(value interface{}) ([]byte, error)
	Deserialize(data []byte, targetType interface{}) error
	GetContentType(value interface{}) string
}

// DefaultCacheSerializer implements basic JSON serialization
type DefaultCacheSerializer struct{}

func (dcs *DefaultCacheSerializer) Serialize(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (dcs *DefaultCacheSerializer) Deserialize(data []byte, targetType interface{}) error {
	return json.Unmarshal(data, targetType)
}

func (dcs *DefaultCacheSerializer) GetContentType(value interface{}) string {
	return "application/json"
}

// CacheEvictionPolicy defines eviction strategies
type CacheEvictionPolicy interface {
	ShouldEvict(item *CacheItem, cacheSize int64, maxSize int64) bool
	SelectEvictionCandidates(items map[string]*CacheItem, count int) []string
}

// LRUEvictionPolicy implements Least Recently Used eviction
type LRUEvictionPolicy struct{}

func (lru *LRUEvictionPolicy) ShouldEvict(item *CacheItem, cacheSize int64, maxSize int64) bool {
	// Evict if cache is over size limit
	return cacheSize > maxSize
}

func (lru *LRUEvictionPolicy) SelectEvictionCandidates(items map[string]*CacheItem, count int) []string {
	if len(items) == 0 || count <= 0 {
		return []string{}
	}
	
	// Sort by last accessed time (ascending - oldest first)
	type itemAccess struct {
		key          string
		lastAccessed time.Time
	}
	
	itemList := make([]itemAccess, 0, len(items))
	for key, item := range items {
		itemList = append(itemList, itemAccess{
			key:          key,
			lastAccessed: item.LastAccessed,
		})
	}
	
	// Simple sort by last accessed time
	for i := 0; i < len(itemList)-1; i++ {
		for j := i + 1; j < len(itemList); j++ {
			if itemList[i].lastAccessed.After(itemList[j].lastAccessed) {
				itemList[i], itemList[j] = itemList[j], itemList[i]
			}
		}
	}
	
	// Return oldest items up to count
	candidates := make([]string, 0, count)
	maxCandidates := count
	if maxCandidates > len(itemList) {
		maxCandidates = len(itemList)
	}
	
	for i := 0; i < maxCandidates; i++ {
		candidates = append(candidates, itemList[i].key)
	}
	
	return candidates
}

// TTLEvictionPolicy implements Time To Live eviction
type TTLEvictionPolicy struct{}

func (ttl *TTLEvictionPolicy) ShouldEvict(item *CacheItem, cacheSize int64, maxSize int64) bool {
	// Evict if item has expired
	return time.Since(item.CreatedAt) > item.TTL
}

func (ttl *TTLEvictionPolicy) SelectEvictionCandidates(items map[string]*CacheItem, count int) []string {
	candidates := make([]string, 0, count)
	now := time.Now()
	
	for key, item := range items {
		if now.Sub(item.CreatedAt) > item.TTL {
			candidates = append(candidates, key)
			if len(candidates) >= count {
				break
			}
		}
	}
	
	return candidates
}

// CacheConfiguration defines configuration interface
type CacheConfiguration interface {
	GetMaxSize() int64
	GetDefaultTTL() time.Duration
	GetEvictionPolicy() CacheEvictionPolicy
	GetCompressionThreshold() int
	IsCompressionEnabled() bool
}

// CacheKeyBuilder builds standardized cache keys
type CacheKeyBuilder struct {
	namespace string
	version   string
}

func NewCacheKeyBuilder(namespace, version string) *CacheKeyBuilder {
	return &CacheKeyBuilder{
		namespace: namespace,
		version:   version,
	}
}

// BuildKey builds a standardized cache key
func (ckb *CacheKeyBuilder) BuildKey(keyType, identifier string, metadata ...string) string {
	parts := []string{ckb.namespace, keyType, identifier}
	
	if len(metadata) > 0 {
		parts = append(parts, metadata...)
	}
	
	if ckb.version != "" {
		parts = append(parts, "v"+ckb.version)
	}
	
	return strings.Join(parts, ":")
}

// BuildPhenotypeKey builds key for phenotype definitions
func (ckb *CacheKeyBuilder) BuildPhenotypeKey(phenotypeID string) string {
	return ckb.BuildKey("phenotype_definition", phenotypeID)
}

// BuildPatientContextKey builds key for patient context
func (ckb *CacheKeyBuilder) BuildPatientContextKey(patientID, contextType string) string {
	return ckb.BuildKey("patient_context", patientID, contextType)
}

// BuildRiskAssessmentKey builds key for risk assessments
func (ckb *CacheKeyBuilder) BuildRiskAssessmentKey(patientID, riskType string) string {
	return ckb.BuildKey("risk_assessment", patientID, riskType)
}

// BuildTreatmentPreferencesKey builds key for treatment preferences
func (ckb *CacheKeyBuilder) BuildTreatmentPreferencesKey(patientID, condition string) string {
	return ckb.BuildKey("treatment_preferences", patientID, condition)
}

// BuildStaticKey builds key for static content
func (ckb *CacheKeyBuilder) BuildStaticKey(contentType, identifier string) string {
	return ckb.BuildKey("static", contentType, identifier)
}

// ParseKey parses a cache key into its components
func (ckb *CacheKeyBuilder) ParseKey(key string) (namespace, keyType, identifier string, metadata []string) {
	parts := strings.Split(key, ":")
	
	if len(parts) < 3 {
		return "", "", "", nil
	}
	
	namespace = parts[0]
	keyType = parts[1]
	identifier = parts[2]
	
	if len(parts) > 3 {
		metadata = parts[3:]
	}
	
	return
}

// CacheHealthChecker monitors cache health
type CacheHealthChecker struct {
	cache           CacheInterface
	healthThreshold HealthThreshold
}

type HealthThreshold struct {
	MinHitRate          float64
	MaxErrorRate        float64
	MaxMemoryPressure   float64
	MaxResponseTime     time.Duration
	MaxEvictionRate     float64
}

func NewCacheHealthChecker(cache CacheInterface) *CacheHealthChecker {
	return &CacheHealthChecker{
		cache: cache,
		healthThreshold: HealthThreshold{
			MinHitRate:        0.7,  // 70% minimum hit rate
			MaxErrorRate:      0.05, // 5% maximum error rate
			MaxMemoryPressure: 0.9,  // 90% maximum memory usage
			MaxResponseTime:   100 * time.Millisecond,
			MaxEvictionRate:   0.1,  // 10% maximum eviction rate
		},
	}
}

// CheckHealth performs comprehensive health check
func (chc *CacheHealthChecker) CheckHealth(ctx context.Context) HealthReport {
	stats := chc.cache.GetStats()
	
	report := HealthReport{
		Timestamp: time.Now(),
		Status:    "healthy",
		Checks:    make(map[string]HealthCheck),
	}
	
	// Check hit rate
	hitRateCheck := HealthCheck{
		Name:   "hit_rate",
		Status: "pass",
		Value:  stats.HitRate,
	}
	if stats.HitRate < chc.healthThreshold.MinHitRate {
		hitRateCheck.Status = "fail"
		hitRateCheck.Message = "Hit rate below threshold"
		report.Status = "degraded"
	}
	report.Checks["hit_rate"] = hitRateCheck
	
	// Check memory usage
	memoryCheck := HealthCheck{
		Name:   "memory_usage",
		Status: "pass",
		Value:  float64(stats.MemoryUsage),
	}
	// Memory check would need maximum memory limit to calculate pressure
	report.Checks["memory_usage"] = memoryCheck
	
	// Check eviction rate
	evictionCheck := HealthCheck{
		Name:   "eviction_rate", 
		Status: "pass",
		Value:  float64(stats.Evictions),
	}
	if stats.Operations > 0 {
		evictionRate := float64(stats.Evictions) / float64(stats.Operations)
		evictionCheck.Value = evictionRate
		if evictionRate > chc.healthThreshold.MaxEvictionRate {
			evictionCheck.Status = "fail"
			evictionCheck.Message = "High eviction rate"
			report.Status = "degraded"
		}
	}
	report.Checks["eviction_rate"] = evictionCheck
	
	// Overall connectivity check
	connectivityCheck := HealthCheck{
		Name:   "connectivity",
		Status: "pass",
	}
	if !chc.cache.IsHealthy(ctx) {
		connectivityCheck.Status = "fail"
		connectivityCheck.Message = "Cache connectivity issues"
		report.Status = "unhealthy"
	}
	report.Checks["connectivity"] = connectivityCheck
	
	return report
}

// HealthReport contains cache health information
type HealthReport struct {
	Timestamp time.Time            `json:"timestamp"`
	Status    string               `json:"status"` // healthy, degraded, unhealthy
	Checks    map[string]HealthCheck `json:"checks"`
}

// HealthCheck represents individual health check result
type HealthCheck struct {
	Name    string  `json:"name"`
	Status  string  `json:"status"` // pass, fail
	Value   float64 `json:"value,omitempty"`
	Message string  `json:"message,omitempty"`
}

// CacheMetricsCollector collects detailed cache metrics
type CacheMetricsCollector struct {
	hitsByTier    map[string]int64
	missByTier    map[string]int64
	operationsByType map[string]int64
	errorsByType  map[string]int64
}

func NewCacheMetricsCollector() *CacheMetricsCollector {
	return &CacheMetricsCollector{
		hitsByTier:       make(map[string]int64),
		missByTier:       make(map[string]int64),
		operationsByType: make(map[string]int64),
		errorsByType:     make(map[string]int64),
	}
}

// RecordHit records cache hit by tier
func (cmc *CacheMetricsCollector) RecordHit(tier string) {
	cmc.hitsByTier[tier]++
}

// RecordMiss records cache miss by tier
func (cmc *CacheMetricsCollector) RecordMiss(tier string) {
	cmc.missByTier[tier]++
}

// RecordOperation records cache operation by type
func (cmc *CacheMetricsCollector) RecordOperation(operation string, duration time.Duration) {
	cmc.operationsByType[operation]++
}

// RecordError records cache error by type
func (cmc *CacheMetricsCollector) RecordError(errorType string) {
	cmc.errorsByType[errorType]++
}

// GetMetrics returns collected metrics
func (cmc *CacheMetricsCollector) GetMetrics() CacheMetrics {
	return CacheMetrics{
		HitsByTier:       cmc.copyIntMap(cmc.hitsByTier),
		MissByTier:       cmc.copyIntMap(cmc.missByTier),
		OperationsByType: cmc.copyIntMap(cmc.operationsByType),
		ErrorsByType:     cmc.copyIntMap(cmc.errorsByType),
	}
}

func (cmc *CacheMetricsCollector) copyIntMap(source map[string]int64) map[string]int64 {
	copy := make(map[string]int64)
	for k, v := range source {
		copy[k] = v
	}
	return copy
}

// CacheMetrics represents collected cache metrics
type CacheMetrics struct {
	HitsByTier       map[string]int64 `json:"hits_by_tier"`
	MissByTier       map[string]int64 `json:"miss_by_tier"`
	OperationsByType map[string]int64 `json:"operations_by_type"`
	ErrorsByType     map[string]int64 `json:"errors_by_type"`
}

// CacheWarmingScheduler manages cache warming schedules
type CacheWarmingScheduler struct {
	warmer    *CacheWarmer
	schedules map[string]WarmingSchedule
}

type WarmingSchedule struct {
	Name        string        `json:"name"`
	Interval    time.Duration `json:"interval"`
	Strategies  []string      `json:"strategies"`
	Enabled     bool          `json:"enabled"`
	LastRun     time.Time     `json:"last_run"`
	NextRun     time.Time     `json:"next_run"`
}

func NewCacheWarmingScheduler(warmer *CacheWarmer) *CacheWarmingScheduler {
	return &CacheWarmingScheduler{
		warmer:    warmer,
		schedules: make(map[string]WarmingSchedule),
	}
}

// AddSchedule adds a warming schedule
func (cws *CacheWarmingScheduler) AddSchedule(schedule WarmingSchedule) {
	schedule.NextRun = time.Now().Add(schedule.Interval)
	cws.schedules[schedule.Name] = schedule
}

// RemoveSchedule removes a warming schedule
func (cws *CacheWarmingScheduler) RemoveSchedule(name string) {
	delete(cws.schedules, name)
}

// GetSchedules returns all warming schedules
func (cws *CacheWarmingScheduler) GetSchedules() map[string]WarmingSchedule {
	schedules := make(map[string]WarmingSchedule)
	for k, v := range cws.schedules {
		schedules[k] = v
	}
	return schedules
}

// RunDueSchedules runs warming schedules that are due
func (cws *CacheWarmingScheduler) RunDueSchedules(ctx context.Context) error {
	now := time.Now()
	
	for name, schedule := range cws.schedules {
		if !schedule.Enabled || now.Before(schedule.NextRun) {
			continue
		}
		
		// Run warming
		err := cws.warmer.WarmCache(ctx)
		if err != nil {
			return fmt.Errorf("warming schedule %s failed: %w", name, err)
		}
		
		// Update schedule
		schedule.LastRun = now
		schedule.NextRun = now.Add(schedule.Interval)
		cws.schedules[name] = schedule
	}
	
	return nil
}

// CacheKeyPattern represents patterns for cache key matching
type CacheKeyPattern struct {
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
	TTL         time.Duration `json:"ttl"`
	CacheTiers  []string `json:"cache_tiers"`
}

// MatchesKey checks if pattern matches a cache key
func (ckp *CacheKeyPattern) MatchesKey(key string) bool {
	// Simple wildcard pattern matching
	return matchesWildcard(key, ckp.Pattern)
}

// matchesWildcard performs simple wildcard matching
func matchesWildcard(text, pattern string) bool {
	if pattern == "*" {
		return true
	}
	
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(text, prefix)
	}
	
	if strings.HasPrefix(pattern, "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(text, suffix)
	}
	
	return text == pattern
}

// CachePerformanceOptimizer optimizes cache performance based on usage patterns
type CachePerformanceOptimizer struct {
	cache          MultiTierCacheInterface
	patterns       []CacheKeyPattern
	optimizations  map[string]OptimizationRule
}

type OptimizationRule struct {
	Name        string        `json:"name"`
	Condition   func(*CacheStats) bool
	Action      func(context.Context, MultiTierCacheInterface) error
	Description string        `json:"description"`
}

func NewCachePerformanceOptimizer(cache MultiTierCacheInterface) *CachePerformanceOptimizer {
	optimizer := &CachePerformanceOptimizer{
		cache:         cache,
		patterns:      []CacheKeyPattern{},
		optimizations: make(map[string]OptimizationRule),
	}
	
	optimizer.initializeOptimizationRules()
	return optimizer
}

// initializeOptimizationRules sets up default optimization rules
func (cpo *CachePerformanceOptimizer) initializeOptimizationRules() {
	// Rule 1: Low hit rate optimization
	cpo.optimizations["low_hit_rate"] = OptimizationRule{
		Name: "low_hit_rate",
		Condition: func(stats *CacheStats) bool {
			return stats.HitRate < 0.8
		},
		Action: func(ctx context.Context, cache MultiTierCacheInterface) error {
			return cache.WarmCache(ctx)
		},
		Description: "Warm cache when hit rate is below 80%",
	}
	
	// Rule 2: High memory pressure optimization
	cpo.optimizations["high_memory_pressure"] = OptimizationRule{
		Name: "high_memory_pressure",
		Condition: func(stats *CacheStats) bool {
			// Assuming we have a way to calculate memory pressure
			return stats.MemoryUsage > 800*1024*1024 // 800MB
		},
		Action: func(ctx context.Context, cache MultiTierCacheInterface) error {
			return cache.Cleanup(ctx)
		},
		Description: "Clean up cache when memory pressure is high",
	}
}

// OptimizePerformance runs performance optimization
func (cpo *CachePerformanceOptimizer) OptimizePerformance(ctx context.Context) error {
	stats := cpo.cache.GetStats()
	
	// Check combined stats for overall optimization decisions
	if combinedStats, exists := stats["combined"]; exists {
		for _, rule := range cpo.optimizations {
			if rule.Condition(combinedStats) {
				if err := rule.Action(ctx, cpo.cache); err != nil {
					return fmt.Errorf("optimization rule %s failed: %w", rule.Name, err)
				}
			}
		}
	}
	
	return nil
}

// AddOptimizationRule adds a custom optimization rule
func (cpo *CachePerformanceOptimizer) AddOptimizationRule(name string, rule OptimizationRule) {
	cpo.optimizations[name] = rule
}

// GetOptimizationRules returns all optimization rules
func (cpo *CachePerformanceOptimizer) GetOptimizationRules() map[string]OptimizationRule {
	rules := make(map[string]OptimizationRule)
	for k, v := range cpo.optimizations {
		rules[k] = v
	}
	return rules
}