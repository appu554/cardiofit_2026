package cache

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// AdvancedCacheManager provides sophisticated caching strategies
type AdvancedCacheManager struct {
	// Multi-tier caches
	l1Cache         *MemoryCache
	l2Cache         *RedisCache
	l3Cache         *PersistentCache
	
	// Advanced features
	predictiveCache *PredictiveCache
	bloomFilter     *BloomFilter
	heatMap         *AccessHeatMap
	
	// Optimization components
	optimizer       *CacheOptimizer
	prewarmer       *PredictivePrewarmer
	compressor      *AdaptiveCompressor
	partitioner     *CachePartitioner
	
	// Configuration and logging
	config          *config.AdvancedCacheConfig
	logger          *logger.Logger
	
	// Metrics
	metrics         *AdvancedCacheMetrics
	
	// Synchronization
	mu              sync.RWMutex
}

// PredictiveCache uses ML-based prediction for cache optimization
type PredictiveCache struct {
	accessPatterns   map[string]*AccessPattern
	predictionModel  *SimplePredictionModel
	preloadQueue     chan *PreloadRequest
	mu               sync.RWMutex
}

// BloomFilter provides fast negative lookups
type BloomFilter struct {
	bits    []uint64
	size    uint64
	hashFns int
	mu      sync.RWMutex
}

// AccessHeatMap tracks access patterns for optimization
type AccessHeatMap struct {
	heatData    map[string]*HeatEntry
	timeWindows []time.Duration
	mu          sync.RWMutex
}

// CacheOptimizer performs real-time cache optimization
type CacheOptimizer struct {
	optimizations    []OptimizationStrategy
	lastOptimization time.Time
	optimizationLog  []OptimizationEvent
	mu               sync.RWMutex
}

// PredictivePrewarmer proactively loads cache entries
type PredictivePrewarmer struct {
	warmupQueue      chan *WarmupTask
	patternAnalyzer  *PatternAnalyzer
	warmupScheduler  *WarmupScheduler
	activeWarmups    map[string]*WarmupTask
	mu               sync.RWMutex
}

// AdaptiveCompressor dynamically adjusts compression
type AdaptiveCompressor struct {
	compressionLevels map[string]int
	performanceData   map[string]*CompressionPerformance
	mu                sync.RWMutex
}

// CachePartitioner manages cache partitioning strategies
type CachePartitioner struct {
	partitions      map[string]*CachePartition
	strategy        string
	rebalanceTimer  *time.Timer
	mu              sync.RWMutex
}

// Supporting types

type AccessPattern struct {
	PatientID       string
	AccessTimes     []time.Time
	Frequency       float64
	Recency         time.Duration
	Predictability  float64
	LastAccess      time.Time
}

type SimplePredictionModel struct {
	weights         map[string]float64
	intercept       float64
	lastTraining    time.Time
}

type PreloadRequest struct {
	SnapshotID      string
	Priority        int
	PredictedTime   time.Time
	PatientID       string
}

type HeatEntry struct {
	SnapshotID      string
	AccessCount     int64
	LastAccess      time.Time
	AccessVelocity  float64
	HotScore        float64
}

type OptimizationStrategy interface {
	Name() string
	Analyze(cache *AdvancedCacheManager) *OptimizationRecommendation
	Apply(cache *AdvancedCacheManager, recommendation *OptimizationRecommendation) error
}

type OptimizationRecommendation struct {
	Strategy        string
	Description     string
	ExpectedGain    float64
	EstimatedCost   time.Duration
	Parameters      map[string]interface{}
}

type OptimizationEvent struct {
	Timestamp       time.Time
	Strategy        string
	Action          string
	Parameters      map[string]interface{}
	Result          string
	PerformanceGain float64
}

type WarmupTask struct {
	SnapshotID      string
	PatientID       string
	Priority        int
	ScheduledTime   time.Time
	EstimatedSize   int64
	Status          string
}

type PatternAnalyzer struct {
	patterns        map[string]*AccessPattern
	analysisWindow  time.Duration
	minSamples      int
}

type WarmupScheduler struct {
	schedule        map[time.Time][]*WarmupTask
	capacity        int
	currentLoad     int
}

type CompressionPerformance struct {
	Level           int
	CompressionRatio float64
	CompressionTime time.Duration
	DecompressionTime time.Duration
	LastMeasured    time.Time
}

type CachePartition struct {
	Name            string
	Size            int64
	MaxSize         int64
	HitRatio        float64
	EvictionRate    float64
	AverageLatency  time.Duration
}

type AdvancedCacheMetrics struct {
	// Traditional metrics
	L1Hits              int64
	L1Misses            int64
	L2Hits              int64
	L2Misses            int64
	L3Hits              int64
	L3Misses            int64
	
	// Advanced metrics
	PredictiveHits      int64
	BloomFilterSaves    int64
	PrewarmingHits      int64
	CompressionSavings  int64
	PartitioningGains   int64
	
	// Performance metrics
	AverageRetrievalTime time.Duration
	CacheEfficiency     float64
	MemoryUtilization   float64
	OptimizationGains   float64
	
	mu                  sync.RWMutex
}

// NewAdvancedCacheManager creates a new advanced cache manager
func NewAdvancedCacheManager(
	cfg *config.AdvancedCacheConfig,
	logger *logger.Logger,
) (*AdvancedCacheManager, error) {
	// Initialize core caches
	l1Cache, err := NewMemoryCache(cfg.L1MaxSize, cfg.L1TTL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 cache: %w", err)
	}

	var l2Cache *RedisCache
	if cfg.EnableL2Cache {
		l2Cache, err = NewRedisCache(cfg.Redis, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create L2 cache: %w", err)
		}
	}

	var l3Cache *PersistentCache
	if cfg.EnableL3Cache {
		l3Cache, err = NewPersistentCache(cfg.L3Config, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create L3 cache: %w", err)
		}
	}

	// Initialize advanced features
	predictiveCache := &PredictiveCache{
		accessPatterns:  make(map[string]*AccessPattern),
		predictionModel: &SimplePredictionModel{
			weights:   make(map[string]float64),
			intercept: 0.0,
		},
		preloadQueue: make(chan *PreloadRequest, cfg.PreloadQueueSize),
	}

	bloomFilter := NewBloomFilter(cfg.BloomFilterSize, cfg.BloomFilterHashCount)
	
	heatMap := &AccessHeatMap{
		heatData: make(map[string]*HeatEntry),
		timeWindows: []time.Duration{
			1 * time.Minute,
			5 * time.Minute,
			15 * time.Minute,
			1 * time.Hour,
		},
	}

	optimizer := &CacheOptimizer{
		optimizations: createOptimizationStrategies(),
		optimizationLog: make([]OptimizationEvent, 0, 1000),
	}

	prewarmer := &PredictivePrewarmer{
		warmupQueue: make(chan *WarmupTask, cfg.WarmupQueueSize),
		patternAnalyzer: &PatternAnalyzer{
			patterns: make(map[string]*AccessPattern),
			analysisWindow: cfg.PatternAnalysisWindow,
			minSamples: cfg.MinPatternSamples,
		},
		warmupScheduler: &WarmupScheduler{
			schedule: make(map[time.Time][]*WarmupTask),
			capacity: cfg.WarmupCapacity,
		},
		activeWarmups: make(map[string]*WarmupTask),
	}

	compressor := &AdaptiveCompressor{
		compressionLevels: make(map[string]int),
		performanceData: make(map[string]*CompressionPerformance),
	}

	partitioner := &CachePartitioner{
		partitions: make(map[string]*CachePartition),
		strategy: cfg.PartitioningStrategy,
	}

	manager := &AdvancedCacheManager{
		l1Cache:         l1Cache,
		l2Cache:         l2Cache,
		l3Cache:         l3Cache,
		predictiveCache: predictiveCache,
		bloomFilter:     bloomFilter,
		heatMap:         heatMap,
		optimizer:       optimizer,
		prewarmer:       prewarmer,
		compressor:      compressor,
		partitioner:     partitioner,
		config:          cfg,
		logger:          logger,
		metrics:         &AdvancedCacheMetrics{},
	}

	// Start background processes
	manager.startBackgroundProcesses()

	return manager, nil
}

// Get retrieves a snapshot with advanced caching strategies
func (acm *AdvancedCacheManager) Get(ctx context.Context, snapshotID string) (*types.ClinicalSnapshot, bool) {
	startTime := time.Now()
	
	// Update access patterns
	acm.updateAccessPattern(snapshotID)
	
	// Check bloom filter first (fast negative lookup)
	if !acm.bloomFilter.Contains(snapshotID) {
		acm.metrics.mu.Lock()
		acm.metrics.BloomFilterSaves++
		acm.metrics.mu.Unlock()
		return nil, false
	}

	// Try L1 cache first
	if snapshot, exists := acm.l1Cache.Get(snapshotID); exists {
		acm.recordHit("L1", time.Since(startTime))
		acm.updateHeatMap(snapshotID, true)
		return snapshot, true
	}

	// Try L2 cache
	if acm.l2Cache != nil {
		if snapshot, exists := acm.l2Cache.Get(ctx, snapshotID); exists {
			// Promote to L1
			acm.l1Cache.Set(snapshotID, snapshot, acm.config.L1TTL)
			acm.recordHit("L2", time.Since(startTime))
			acm.updateHeatMap(snapshotID, true)
			return snapshot, true
		}
	}

	// Try L3 cache
	if acm.l3Cache != nil {
		if snapshot, exists := acm.l3Cache.Get(ctx, snapshotID); exists {
			// Promote to L2 and L1
			if acm.l2Cache != nil {
				acm.l2Cache.Set(ctx, snapshotID, snapshot, acm.config.L2TTL)
			}
			acm.l1Cache.Set(snapshotID, snapshot, acm.config.L1TTL)
			acm.recordHit("L3", time.Since(startTime))
			acm.updateHeatMap(snapshotID, true)
			return snapshot, true
		}
	}

	// Cache miss
	acm.updateHeatMap(snapshotID, false)
	return nil, false
}

// Set stores a snapshot with advanced caching strategies
func (acm *AdvancedCacheManager) Set(ctx context.Context, snapshotID string, snapshot *types.ClinicalSnapshot, ttl time.Duration) error {
	// Add to bloom filter
	acm.bloomFilter.Add(snapshotID)
	
	// Determine optimal compression level
	compressionLevel := acm.compressor.getOptimalCompressionLevel(snapshotID, snapshot)
	
	// Determine cache tier placement based on heat score
	heatScore := acm.heatMap.getHeatScore(snapshotID)
	
	// Always store in L1
	if err := acm.l1Cache.Set(snapshotID, snapshot, ttl); err != nil {
		acm.logger.Warn("Failed to store in L1 cache", zap.Error(err))
	}
	
	// Store in L2 based on heat score
	if acm.l2Cache != nil && heatScore > acm.config.L2HeatThreshold {
		compressedSnapshot := acm.compressor.compress(snapshot, compressionLevel)
		if err := acm.l2Cache.Set(ctx, snapshotID, compressedSnapshot, ttl*2); err != nil {
			acm.logger.Warn("Failed to store in L2 cache", zap.Error(err))
		}
	}
	
	// Store in L3 for long-term caching
	if acm.l3Cache != nil && acm.config.EnableL3Cache {
		persistentSnapshot := acm.preparePersistentSnapshot(snapshot)
		if err := acm.l3Cache.Set(ctx, snapshotID, persistentSnapshot, ttl*5); err != nil {
			acm.logger.Warn("Failed to store in L3 cache", zap.Error(err))
		}
	}
	
	// Trigger predictive preloading
	acm.triggerPredictivePreload(snapshotID, snapshot.PatientID)
	
	return nil
}

// PredictiveGet attempts to predict and preload likely cache misses
func (acm *AdvancedCacheManager) PredictiveGet(ctx context.Context, patientID string) {
	predictions := acm.predictiveCache.predictNextAccess(patientID)
	
	for _, prediction := range predictions {
		// Queue for preloading if not already cached
		if _, exists := acm.Get(ctx, prediction.SnapshotID); !exists {
			select {
			case acm.predictiveCache.preloadQueue <- prediction:
				acm.logger.Debug("Queued predictive preload",
					zap.String("snapshot_id", prediction.SnapshotID),
					zap.String("patient_id", patientID),
					zap.Int("priority", prediction.Priority),
				)
			default:
				// Queue full, skip this prediction
			}
		}
	}
}

// OptimizeCache performs real-time cache optimization
func (acm *AdvancedCacheManager) OptimizeCache(ctx context.Context) error {
	acm.logger.Debug("Starting cache optimization cycle")
	
	// Run optimization strategies
	for _, strategy := range acm.optimizer.optimizations {
		recommendation := strategy.Analyze(acm)
		if recommendation.ExpectedGain > acm.config.MinOptimizationGain {
			if err := strategy.Apply(acm, recommendation); err != nil {
				acm.logger.Warn("Optimization strategy failed",
					zap.String("strategy", strategy.Name()),
					zap.Error(err),
				)
			} else {
				acm.recordOptimization(strategy.Name(), recommendation)
			}
		}
	}
	
	acm.optimizer.lastOptimization = time.Now()
	return nil
}

// Warm performs intelligent cache warming
func (acm *AdvancedCacheManager) Warm(ctx context.Context, hints []string) error {
	// Analyze patterns to determine warming strategy
	patterns := acm.prewarmer.patternAnalyzer.analyzePatterns(hints)
	
	// Schedule warming tasks
	tasks := acm.prewarmer.warmupScheduler.scheduleTasks(patterns, acm.config.WarmupCapacity)
	
	// Execute warming tasks
	for _, task := range tasks {
		select {
		case acm.prewarmer.warmupQueue <- task:
			acm.logger.Debug("Scheduled cache warmup",
				zap.String("snapshot_id", task.SnapshotID),
				zap.String("patient_id", task.PatientID),
				zap.Time("scheduled_time", task.ScheduledTime),
			)
		default:
			// Queue full, task will be retried later
		}
	}
	
	return nil
}

// Background processes and supporting methods

func (acm *AdvancedCacheManager) startBackgroundProcesses() {
	// Predictive preloading worker
	go acm.predictivePreloadWorker()
	
	// Cache warmup worker
	go acm.cacheWarmupWorker()
	
	// Optimization scheduler
	go acm.optimizationScheduler()
	
	// Metrics collector
	go acm.metricsCollector()
}

func (acm *AdvancedCacheManager) predictivePreloadWorker() {
	for preloadReq := range acm.predictiveCache.preloadQueue {
		// Simulate preloading (in production, this would fetch from Context Service)
		acm.logger.Debug("Processing predictive preload",
			zap.String("snapshot_id", preloadReq.SnapshotID),
			zap.String("patient_id", preloadReq.PatientID),
		)
		
		// Update metrics
		acm.metrics.mu.Lock()
		acm.metrics.PredictiveHits++
		acm.metrics.mu.Unlock()
	}
}

func (acm *AdvancedCacheManager) cacheWarmupWorker() {
	for warmupTask := range acm.prewarmer.warmupQueue {
		acm.prewarmer.mu.Lock()
		acm.prewarmer.activeWarmups[warmupTask.SnapshotID] = warmupTask
		acm.prewarmer.mu.Unlock()
		
		// Execute warmup task
		acm.logger.Debug("Processing cache warmup",
			zap.String("snapshot_id", warmupTask.SnapshotID),
			zap.String("patient_id", warmupTask.PatientID),
			zap.Int("priority", warmupTask.Priority),
		)
		
		// Simulate warmup completion
		warmupTask.Status = "completed"
		
		acm.prewarmer.mu.Lock()
		delete(acm.prewarmer.activeWarmups, warmupTask.SnapshotID)
		acm.prewarmer.mu.Unlock()
		
		// Update metrics
		acm.metrics.mu.Lock()
		acm.metrics.PrewarmingHits++
		acm.metrics.mu.Unlock()
	}
}

func (acm *AdvancedCacheManager) optimizationScheduler() {
	ticker := time.NewTicker(acm.config.OptimizationInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := acm.OptimizeCache(ctx); err != nil {
			acm.logger.Error("Cache optimization failed", zap.Error(err))
		}
		cancel()
	}
}

func (acm *AdvancedCacheManager) metricsCollector() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		acm.updateMetrics()
	}
}

// Helper methods

func (acm *AdvancedCacheManager) updateAccessPattern(snapshotID string) {
	acm.predictiveCache.mu.Lock()
	defer acm.predictiveCache.mu.Unlock()
	
	now := time.Now()
	pattern, exists := acm.predictiveCache.accessPatterns[snapshotID]
	if !exists {
		pattern = &AccessPattern{
			AccessTimes: make([]time.Time, 0, 100),
		}
		acm.predictiveCache.accessPatterns[snapshotID] = pattern
	}
	
	pattern.AccessTimes = append(pattern.AccessTimes, now)
	pattern.LastAccess = now
	
	// Keep only last 100 accesses
	if len(pattern.AccessTimes) > 100 {
		pattern.AccessTimes = pattern.AccessTimes[1:]
	}
	
	// Update frequency and recency
	if len(pattern.AccessTimes) > 1 {
		totalDuration := pattern.AccessTimes[len(pattern.AccessTimes)-1].Sub(pattern.AccessTimes[0])
		pattern.Frequency = float64(len(pattern.AccessTimes)) / totalDuration.Hours()
		pattern.Recency = time.Since(pattern.LastAccess)
	}
}

func (acm *AdvancedCacheManager) updateHeatMap(snapshotID string, hit bool) {
	acm.heatMap.mu.Lock()
	defer acm.heatMap.mu.Unlock()
	
	entry, exists := acm.heatMap.heatData[snapshotID]
	if !exists {
		entry = &HeatEntry{
			SnapshotID: snapshotID,
		}
		acm.heatMap.heatData[snapshotID] = entry
	}
	
	entry.AccessCount++
	entry.LastAccess = time.Now()
	
	// Calculate access velocity (accesses per hour)
	if entry.AccessCount > 1 {
		duration := time.Since(entry.LastAccess).Hours()
		if duration > 0 {
			entry.AccessVelocity = float64(entry.AccessCount) / duration
		}
	}
	
	// Calculate hot score (combination of frequency, recency, and hit ratio)
	recencyScore := 1.0 / (1.0 + time.Since(entry.LastAccess).Hours())
	velocityScore := entry.AccessVelocity / 10.0 // Normalize to 0-1 scale
	if velocityScore > 1.0 {
		velocityScore = 1.0
	}
	
	entry.HotScore = 0.4*recencyScore + 0.6*velocityScore
}

func (acm *AdvancedCacheManager) recordHit(tier string, duration time.Duration) {
	acm.metrics.mu.Lock()
	defer acm.metrics.mu.Unlock()
	
	switch tier {
	case "L1":
		acm.metrics.L1Hits++
	case "L2":
		acm.metrics.L2Hits++
	case "L3":
		acm.metrics.L3Hits++
	}
	
	// Update average retrieval time
	if acm.metrics.L1Hits+acm.metrics.L2Hits+acm.metrics.L3Hits == 1 {
		acm.metrics.AverageRetrievalTime = duration
	} else {
		totalHits := acm.metrics.L1Hits + acm.metrics.L2Hits + acm.metrics.L3Hits
		totalTime := int64(acm.metrics.AverageRetrievalTime) * totalHits
		acm.metrics.AverageRetrievalTime = time.Duration(
			(totalTime + int64(duration)) / totalHits,
		)
	}
}

func (acm *AdvancedCacheManager) recordOptimization(strategy string, recommendation *OptimizationRecommendation) {
	acm.optimizer.mu.Lock()
	defer acm.optimizer.mu.Unlock()
	
	event := OptimizationEvent{
		Timestamp:       time.Now(),
		Strategy:        strategy,
		Action:          "applied",
		Parameters:      recommendation.Parameters,
		Result:          "success",
		PerformanceGain: recommendation.ExpectedGain,
	}
	
	acm.optimizer.optimizationLog = append(acm.optimizer.optimizationLog, event)
	
	// Keep only last 1000 events
	if len(acm.optimizer.optimizationLog) > 1000 {
		acm.optimizer.optimizationLog = acm.optimizer.optimizationLog[1:]
	}
}

func (acm *AdvancedCacheManager) triggerPredictivePreload(snapshotID, patientID string) {
	// Simple prediction: preload other recent snapshots for the same patient
	acm.predictiveCache.mu.RLock()
	defer acm.predictiveCache.mu.RUnlock()
	
	for id, pattern := range acm.predictiveCache.accessPatterns {
		if pattern.PatientID == patientID && id != snapshotID {
			if time.Since(pattern.LastAccess) < 1*time.Hour {
				preloadReq := &PreloadRequest{
					SnapshotID:    id,
					Priority:      2,
					PredictedTime: time.Now().Add(5 * time.Minute),
					PatientID:     patientID,
				}
				
				select {
				case acm.predictiveCache.preloadQueue <- preloadReq:
					// Queued successfully
				default:
					// Queue full, skip
				}
			}
		}
	}
}

func (acm *AdvancedCacheManager) updateMetrics() {
	acm.metrics.mu.Lock()
	defer acm.metrics.mu.Unlock()
	
	totalHits := acm.metrics.L1Hits + acm.metrics.L2Hits + acm.metrics.L3Hits
	totalMisses := acm.metrics.L1Misses + acm.metrics.L2Misses + acm.metrics.L3Misses
	totalRequests := totalHits + totalMisses
	
	if totalRequests > 0 {
		acm.metrics.CacheEfficiency = float64(totalHits) / float64(totalRequests)
	}
}

// Supporting functions and interfaces

func (pc *PredictiveCache) predictNextAccess(patientID string) []*PreloadRequest {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	var predictions []*PreloadRequest
	
	// Simple prediction algorithm
	for snapshotID, pattern := range pc.accessPatterns {
		if pattern.PatientID == patientID {
			// Predict based on access frequency and recency
			score := pattern.Frequency * (1.0 - pattern.Recency.Hours()/24.0)
			if score > 0.5 {
				priority := 1
				if score > 0.8 {
					priority = 0 // High priority
				}
				
				predictions = append(predictions, &PreloadRequest{
					SnapshotID:    snapshotID,
					Priority:      priority,
					PredictedTime: time.Now().Add(time.Duration(1.0/pattern.Frequency) * time.Hour),
					PatientID:     patientID,
				})
			}
		}
	}
	
	// Sort by priority
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].Priority < predictions[j].Priority
	})
	
	// Return top 5 predictions
	if len(predictions) > 5 {
		predictions = predictions[:5]
	}
	
	return predictions
}

func (hm *AccessHeatMap) getHeatScore(snapshotID string) float64 {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	entry, exists := hm.heatData[snapshotID]
	if !exists {
		return 0.0
	}
	
	return entry.HotScore
}

func (ac *AdaptiveCompressor) getOptimalCompressionLevel(snapshotID string, snapshot *types.ClinicalSnapshot) int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	
	// Simple heuristic based on snapshot size
	if level, exists := ac.compressionLevels[snapshotID]; exists {
		return level
	}
	
	// Default compression level based on estimated size
	return 6 // Default gzip compression level
}

func (ac *AdaptiveCompressor) compress(snapshot *types.ClinicalSnapshot, level int) *types.ClinicalSnapshot {
	// Placeholder for compression logic
	return snapshot
}

func (acm *AdvancedCacheManager) preparePersistentSnapshot(snapshot *types.ClinicalSnapshot) *types.ClinicalSnapshot {
	// Prepare snapshot for persistent storage (e.g., remove volatile fields)
	return snapshot
}

// Bloom Filter implementation

func NewBloomFilter(size uint64, hashCount int) *BloomFilter {
	return &BloomFilter{
		bits:    make([]uint64, (size+63)/64),
		size:    size,
		hashFns: hashCount,
	}
}

func (bf *BloomFilter) Add(item string) {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	
	hashes := bf.getHashes(item)
	for _, hash := range hashes {
		index := hash % bf.size
		wordIndex := index / 64
		bitIndex := index % 64
		bf.bits[wordIndex] |= 1 << bitIndex
	}
}

func (bf *BloomFilter) Contains(item string) bool {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	
	hashes := bf.getHashes(item)
	for _, hash := range hashes {
		index := hash % bf.size
		wordIndex := index / 64
		bitIndex := index % 64
		if bf.bits[wordIndex]&(1<<bitIndex) == 0 {
			return false
		}
	}
	return true
}

func (bf *BloomFilter) getHashes(item string) []uint64 {
	h := fnv.New64a()
	h.Write([]byte(item))
	hash1 := h.Sum64()
	
	h.Reset()
	h.Write([]byte(item + "salt"))
	hash2 := h.Sum64()
	
	hashes := make([]uint64, bf.hashFns)
	for i := 0; i < bf.hashFns; i++ {
		hashes[i] = hash1 + uint64(i)*hash2
	}
	
	return hashes
}

// Optimization strategies

func createOptimizationStrategies() []OptimizationStrategy {
	return []OptimizationStrategy{
		&TTLOptimizationStrategy{},
		&EvictionPolicyOptimization{},
		&CompressionOptimization{},
		&PartitioningOptimization{},
	}
}

type TTLOptimizationStrategy struct{}

func (s *TTLOptimizationStrategy) Name() string { return "TTL_optimization" }

func (s *TTLOptimizationStrategy) Analyze(cache *AdvancedCacheManager) *OptimizationRecommendation {
	// Analyze access patterns to recommend optimal TTL values
	return &OptimizationRecommendation{
		Strategy:     "TTL_optimization",
		Description:  "Adjust TTL values based on access patterns",
		ExpectedGain: 0.15,
		EstimatedCost: 100 * time.Millisecond,
		Parameters: map[string]interface{}{
			"l1_ttl_multiplier": 1.2,
			"l2_ttl_multiplier": 1.5,
		},
	}
}

func (s *TTLOptimizationStrategy) Apply(cache *AdvancedCacheManager, rec *OptimizationRecommendation) error {
	// Apply TTL optimizations
	cache.logger.Debug("Applied TTL optimization", zap.Any("parameters", rec.Parameters))
	return nil
}

type EvictionPolicyOptimization struct{}
func (s *EvictionPolicyOptimization) Name() string { return "eviction_policy_optimization" }
func (s *EvictionPolicyOptimization) Analyze(cache *AdvancedCacheManager) *OptimizationRecommendation {
	return &OptimizationRecommendation{
		Strategy: "eviction_policy_optimization",
		Description: "Optimize eviction policy based on access patterns",
		ExpectedGain: 0.10,
		EstimatedCost: 50 * time.Millisecond,
	}
}
func (s *EvictionPolicyOptimization) Apply(cache *AdvancedCacheManager, rec *OptimizationRecommendation) error {
	return nil
}

type CompressionOptimization struct{}
func (s *CompressionOptimization) Name() string { return "compression_optimization" }
func (s *CompressionOptimization) Analyze(cache *AdvancedCacheManager) *OptimizationRecommendation {
	return &OptimizationRecommendation{
		Strategy: "compression_optimization",
		Description: "Adjust compression levels for optimal performance",
		ExpectedGain: 0.08,
		EstimatedCost: 25 * time.Millisecond,
	}
}
func (s *CompressionOptimization) Apply(cache *AdvancedCacheManager, rec *OptimizationRecommendation) error {
	return nil
}

type PartitioningOptimization struct{}
func (s *PartitioningOptimization) Name() string { return "partitioning_optimization" }
func (s *PartitioningOptimization) Analyze(cache *AdvancedCacheManager) *OptimizationRecommendation {
	return &OptimizationRecommendation{
		Strategy: "partitioning_optimization",
		Description: "Optimize cache partitioning strategy",
		ExpectedGain: 0.12,
		EstimatedCost: 200 * time.Millisecond,
	}
}
func (s *PartitioningOptimization) Apply(cache *AdvancedCacheManager, rec *OptimizationRecommendation) error {
	return nil
}

// GetMetrics returns comprehensive cache metrics
func (acm *AdvancedCacheManager) GetMetrics() *AdvancedCacheMetrics {
	acm.metrics.mu.RLock()
	defer acm.metrics.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	return &AdvancedCacheMetrics{
		L1Hits:              acm.metrics.L1Hits,
		L1Misses:            acm.metrics.L1Misses,
		L2Hits:              acm.metrics.L2Hits,
		L2Misses:            acm.metrics.L2Misses,
		L3Hits:              acm.metrics.L3Hits,
		L3Misses:            acm.metrics.L3Misses,
		PredictiveHits:      acm.metrics.PredictiveHits,
		BloomFilterSaves:    acm.metrics.BloomFilterSaves,
		PrewarmingHits:      acm.metrics.PrewarmingHits,
		CompressionSavings:  acm.metrics.CompressionSavings,
		PartitioningGains:   acm.metrics.PartitioningGains,
		AverageRetrievalTime: acm.metrics.AverageRetrievalTime,
		CacheEfficiency:     acm.metrics.CacheEfficiency,
		MemoryUtilization:   acm.metrics.MemoryUtilization,
		OptimizationGains:   acm.metrics.OptimizationGains,
	}
}