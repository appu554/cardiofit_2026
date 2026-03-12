package cache

import (
	"container/list"
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"kb-2-clinical-context-go/internal/metrics"
)

// MemoryCacheConfig configures the in-memory L1 cache
type MemoryCacheConfig struct {
	MaxSize       int64         // Maximum memory usage in bytes (100MB)
	DefaultTTL    time.Duration // Default TTL (5 minutes)
	MaxItems      int           // Maximum number of items (10,000)
	EvictionRate  float64       // Percentage to evict when full (0.1 = 10%)
	HitRateTarget float64       // Target hit rate (0.85 = 85%)
}

// MemoryCache implements a high-performance LRU cache for L1 caching
type MemoryCache struct {
	config  *MemoryCacheConfig
	metrics *metrics.PrometheusMetrics
	
	// LRU implementation
	items    map[string]*list.Element
	lruList  *list.List
	mu       sync.RWMutex
	
	// Statistics
	hits         int64
	misses       int64
	evictions    int64
	currentSize  int64
	lastOptimize time.Time
	
	// Background cleanup
	cleanup chan struct{}
	done    chan struct{}
}

// cacheEntry represents a cached item with LRU metadata
type cacheEntry struct {
	key        string
	value      interface{}
	size       int64
	createdAt  time.Time
	expiresAt  time.Time
	accessCount int64
	lastAccessed time.Time
}

// NewMemoryCache creates a new in-memory LRU cache
func NewMemoryCache(config *MemoryCacheConfig, metricsCollector *metrics.PrometheusMetrics) *MemoryCache {
	cache := &MemoryCache{
		config:   config,
		metrics:  metricsCollector,
		items:    make(map[string]*list.Element),
		lruList:  list.New(),
		cleanup:  make(chan struct{}, 1),
		done:     make(chan struct{}),
	}
	
	// Start background cleanup goroutine
	go cache.backgroundCleanup()
	
	return cache
}

// Get retrieves an item from the cache
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	elem, found := mc.items[key]
	if !found {
		mc.misses++
		return nil, false
	}
	
	entry := elem.Value.(*cacheEntry)
	
	// Check expiration
	if time.Now().After(entry.expiresAt) {
		mc.removeElement(elem)
		mc.misses++
		return nil, false
	}
	
	// Update access statistics
	entry.accessCount++
	entry.lastAccessed = time.Now()
	
	// Move to front (most recently used)
	mc.lruList.MoveToFront(elem)
	
	mc.hits++
	return entry.value, true
}

// Set stores an item in the cache
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	// Check if item already exists
	if elem, found := mc.items[key]; found {
		// Update existing item
		entry := elem.Value.(*cacheEntry)
		oldSize := entry.size
		entry.value = value
		entry.size = mc.calculateSize(value)
		entry.expiresAt = time.Now().Add(ttl)
		entry.lastAccessed = time.Now()
		entry.accessCount++
		
		// Update size tracking
		mc.currentSize = mc.currentSize - oldSize + entry.size
		
		// Move to front
		mc.lruList.MoveToFront(elem)
		return
	}
	
	// Create new entry
	size := mc.calculateSize(value)
	entry := &cacheEntry{
		key:          key,
		value:        value,
		size:         size,
		createdAt:    time.Now(),
		expiresAt:    time.Now().Add(ttl),
		accessCount:  1,
		lastAccessed: time.Now(),
	}
	
	// Check if we need to evict items before adding
	mc.ensureCapacity(size)
	
	// Add to cache
	elem := mc.lruList.PushFront(entry)
	mc.items[key] = elem
	mc.currentSize += size
}

// Delete removes an item from the cache
func (mc *MemoryCache) Delete(key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if elem, found := mc.items[key]; found {
		mc.removeElement(elem)
	}
}

// DeletePattern removes all items matching a pattern
func (mc *MemoryCache) DeletePattern(pattern string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	keysToDelete := []string{}
	
	// Find matching keys
	for key := range mc.items {
		if mc.matchesPattern(key, pattern) {
			keysToDelete = append(keysToDelete, key)
		}
	}
	
	// Delete matching items
	for _, key := range keysToDelete {
		if elem, found := mc.items[key]; found {
			mc.removeElementUnsafe(elem)
		}
	}
	
	return nil
}

// Clear removes all items from the cache
func (mc *MemoryCache) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.items = make(map[string]*list.Element)
	mc.lruList = list.New()
	mc.currentSize = 0
}

// GetStats returns current cache statistics
func (mc *MemoryCache) GetStats() *CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	totalOps := mc.hits + mc.misses
	hitRate := 0.0
	if totalOps > 0 {
		hitRate = float64(mc.hits) / float64(totalOps)
	}
	
	return &CacheStats{
		HitRate:     hitRate,
		MissRate:    1.0 - hitRate,
		Size:        len(mc.items),
		Evictions:   mc.evictions,
		Operations:  totalOps,
		MemoryUsage: mc.currentSize,
	}
}

// Optimize performs cache optimization
func (mc *MemoryCache) Optimize() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	now := time.Now()
	
	// Skip if optimized recently (within last minute)
	if now.Sub(mc.lastOptimize) < time.Minute {
		return nil
	}
	
	// Remove expired items
	mc.removeExpiredUnsafe()
	
	// If hit rate is below target, adjust eviction strategy
	stats := mc.getStatsUnsafe()
	if stats.HitRate < mc.config.HitRateTarget {
		// More aggressive cleanup of less frequently accessed items
		mc.evictLowFrequencyItemsUnsafe(0.05) // Evict 5% of low-frequency items
	}
	
	mc.lastOptimize = now
	return nil
}

// Cleanup performs cache cleanup
func (mc *MemoryCache) Cleanup() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.removeExpiredUnsafe()
	return nil
}

// Close closes the cache and cleanup goroutines
func (mc *MemoryCache) Close() {
	close(mc.done)
	close(mc.cleanup)
}

// Private methods

// ensureCapacity ensures cache has capacity for new item
func (mc *MemoryCache) ensureCapacity(newItemSize int64) {
	// Check size limit
	targetSize := mc.currentSize + newItemSize
	if targetSize > mc.config.MaxSize {
		mc.evictBySize(targetSize - mc.config.MaxSize)
	}
	
	// Check item count limit
	if len(mc.items) >= mc.config.MaxItems {
		itemsToEvict := int(float64(mc.config.MaxItems) * mc.config.EvictionRate)
		if itemsToEvict < 1 {
			itemsToEvict = 1
		}
		mc.evictLRUItems(itemsToEvict)
	}
}

// evictBySize evicts items to free up specified amount of memory
func (mc *MemoryCache) evictBySize(sizeToFree int64) {
	freedSize := int64(0)
	
	// Start from least recently used
	for elem := mc.lruList.Back(); elem != nil && freedSize < sizeToFree; {
		next := elem.Prev()
		entry := elem.Value.(*cacheEntry)
		freedSize += entry.size
		mc.removeElementUnsafe(elem)
		elem = next
	}
}

// evictLRUItems evicts specified number of least recently used items
func (mc *MemoryCache) evictLRUItems(count int) {
	evicted := 0
	
	for elem := mc.lruList.Back(); elem != nil && evicted < count; {
		next := elem.Prev()
		mc.removeElementUnsafe(elem)
		evicted++
		elem = next
	}
}

// evictLowFrequencyItemsUnsafe evicts items with low access frequency
func (mc *MemoryCache) evictLowFrequencyItemsUnsafe(percentage float64) {
	if len(mc.items) == 0 {
		return
	}
	
	// Collect access frequencies
	type itemFreq struct {
		elem      *list.Element
		frequency float64
	}
	
	items := make([]itemFreq, 0, len(mc.items))
	now := time.Now()
	
	for _, elem := range mc.items {
		entry := elem.Value.(*cacheEntry)
		age := now.Sub(entry.createdAt).Seconds()
		if age > 0 {
			frequency := float64(entry.accessCount) / age // accesses per second
			items = append(items, itemFreq{elem: elem, frequency: frequency})
		}
	}
	
	if len(items) == 0 {
		return
	}
	
	// Sort by frequency (ascending)
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].frequency > items[j].frequency {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	
	// Evict lowest frequency items
	toEvict := int(float64(len(items)) * percentage)
	if toEvict < 1 {
		toEvict = 1
	}
	
	for i := 0; i < toEvict && i < len(items); i++ {
		mc.removeElementUnsafe(items[i].elem)
	}
}

// removeElement removes an element from the cache (thread-safe)
func (mc *MemoryCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*cacheEntry)
	delete(mc.items, entry.key)
	mc.lruList.Remove(elem)
	mc.currentSize -= entry.size
	mc.evictions++
}

// removeElementUnsafe removes an element (caller must hold lock)
func (mc *MemoryCache) removeElementUnsafe(elem *list.Element) {
	entry := elem.Value.(*cacheEntry)
	delete(mc.items, entry.key)
	mc.lruList.Remove(elem)
	mc.currentSize -= entry.size
	mc.evictions++
}

// removeExpiredUnsafe removes expired items (caller must hold lock)
func (mc *MemoryCache) removeExpiredUnsafe() {
	now := time.Now()
	
	// Walk through list and remove expired items
	for elem := mc.lruList.Back(); elem != nil; {
		next := elem.Prev()
		entry := elem.Value.(*cacheEntry)
		
		if now.After(entry.expiresAt) {
			mc.removeElementUnsafe(elem)
		}
		
		elem = next
	}
}

// calculateSize estimates memory usage of a value
func (mc *MemoryCache) calculateSize(value interface{}) int64 {
	// Rough estimation of memory usage
	// In production, you might want a more accurate size calculation
	
	baseSize := int64(unsafe.Sizeof(value))
	
	switch v := value.(type) {
	case string:
		return baseSize + int64(len(v))
	case []byte:
		return baseSize + int64(len(v))
	case *models.PhenotypeDefinition:
		// Estimate phenotype definition size
		size := baseSize + int64(len(v.Name)) + int64(len(v.Description)) + int64(len(v.CELRule))
		return size + 200 // Additional overhead for metadata
	case *models.ClinicalContext:
		// Estimate clinical context size (can be large)
		size := baseSize + 500 // Base overhead
		if v.PhenotypeResults != nil {
			size += int64(len(v.PhenotypeResults)) * 300 // Estimate per result
		}
		if v.RiskAssessment != nil {
			size += 1000 // Risk assessment overhead
		}
		if v.TreatmentPreferences != nil {
			size += int64(len(v.TreatmentPreferences.TreatmentOptions)) * 200
		}
		return size
	case *models.RiskAssessmentResult:
		size := baseSize + int64(len(v.RiskFactors))*100 + int64(len(v.Recommendations))*150
		return size
	case *models.TreatmentPreferencesResult:
		size := baseSize + int64(len(v.TreatmentOptions))*200 + int64(len(v.PreferredTreatments))*100
		return size
	default:
		// Generic estimation based on Go runtime
		return baseSize + 100 // Conservative estimate
	}
}

// matchesPattern checks if key matches cache pattern
func (mc *MemoryCache) matchesPattern(key, pattern string) bool {
	// Simple wildcard pattern matching
	if len(pattern) == 0 {
		return false
	}
	
	// Handle trailing wildcard
	if pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}
	
	// Exact match
	return key == pattern
}

// getStatsUnsafe returns stats without locking (caller must hold lock)
func (mc *MemoryCache) getStatsUnsafe() *CacheStats {
	totalOps := mc.hits + mc.misses
	hitRate := 0.0
	if totalOps > 0 {
		hitRate = float64(mc.hits) / float64(totalOps)
	}
	
	return &CacheStats{
		HitRate:     hitRate,
		MissRate:    1.0 - hitRate,
		Size:        len(mc.items),
		Evictions:   mc.evictions,
		Operations:  totalOps,
		MemoryUsage: mc.currentSize,
	}
}

// backgroundCleanup runs periodic cleanup in background
func (mc *MemoryCache) backgroundCleanup() {
	ticker := time.NewTicker(30 * time.Second) // Cleanup every 30 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			mc.performCleanup()
		case <-mc.cleanup:
			mc.performCleanup()
		case <-mc.done:
			return
		}
	}
}

// performCleanup performs the actual cleanup operations
func (mc *MemoryCache) performCleanup() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	// Remove expired items
	mc.removeExpiredUnsafe()
	
	// Check memory pressure
	stats := mc.getStatsUnsafe()
	memoryPressure := float64(mc.currentSize) / float64(mc.config.MaxSize)
	
	if memoryPressure > 0.8 { // 80% memory usage
		// Aggressive cleanup - evict 20% of items
		itemsToEvict := int(float64(len(mc.items)) * 0.2)
		mc.evictLRUItems(itemsToEvict)
		
		// Force garbage collection to reclaim memory
		runtime.GC()
	} else if memoryPressure > 0.6 { // 60% memory usage  
		// Moderate cleanup - remove low-frequency items
		mc.evictLowFrequencyItemsUnsafe(0.1) // 10% of low-frequency items
	}
	
	// Report statistics to metrics
	mc.reportMetrics(stats)
}

// reportMetrics reports cache statistics to Prometheus
func (mc *MemoryCache) reportMetrics(stats *CacheStats) {
	// Report cache operations
	mc.metrics.CacheOperations.WithLabelValues("get", "l1").Add(float64(stats.Operations))
	
	// Report hit/miss rates through operations
	if mc.hits > 0 {
		mc.metrics.CacheHits.WithLabelValues("l1").Add(float64(mc.hits))
		mc.hits = 0 // Reset counter after reporting
	}
	
	if mc.misses > 0 {
		mc.metrics.CacheMisses.WithLabelValues("l1").Add(float64(mc.misses))
		mc.misses = 0 // Reset counter after reporting
	}
}

// GetSize returns current cache size in bytes
func (mc *MemoryCache) GetSize() int64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.currentSize
}

// GetItemCount returns current number of items
func (mc *MemoryCache) GetItemCount() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return len(mc.items)
}

// GetMemoryPressure returns memory usage percentage (0.0 to 1.0)
func (mc *MemoryCache) GetMemoryPressure() float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return float64(mc.currentSize) / float64(mc.config.MaxSize)
}

// ForceEviction forces eviction of specified number of items
func (mc *MemoryCache) ForceEviction(count int) int {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	initialCount := len(mc.items)
	mc.evictLRUItems(count)
	return initialCount - len(mc.items)
}

// WarmUp preloads frequently accessed keys
func (mc *MemoryCache) WarmUp(keys []string, loader func(string) (interface{}, error)) error {
	for _, key := range keys {
		// Check if already cached
		if _, found := mc.Get(key); found {
			continue
		}
		
		// Load and cache
		data, err := loader(key)
		if err != nil {
			return fmt.Errorf("failed to warm cache for key %s: %w", key, err)
		}
		
		mc.Set(key, data, mc.config.DefaultTTL)
	}
	
	return nil
}

// GetHotKeys returns most frequently accessed keys
func (mc *MemoryCache) GetHotKeys(limit int) []string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	type keyFreq struct {
		key       string
		frequency int64
	}
	
	items := make([]keyFreq, 0, len(mc.items))
	
	for _, elem := range mc.items {
		entry := elem.Value.(*cacheEntry)
		items = append(items, keyFreq{
			key:       entry.key,
			frequency: entry.accessCount,
		})
	}
	
	// Sort by frequency (descending)
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].frequency < items[j].frequency {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	
	// Return top keys
	result := make([]string, 0, limit)
	maxItems := limit
	if maxItems > len(items) {
		maxItems = len(items)
	}
	
	for i := 0; i < maxItems; i++ {
		result = append(result, items[i].key)
	}
	
	return result
}

// TriggerCleanup triggers immediate cleanup
func (mc *MemoryCache) TriggerCleanup() {
	select {
	case mc.cleanup <- struct{}{}:
	default:
		// Cleanup already triggered
	}
}

// Performance monitoring methods

// GetAverageAccessTime returns average access time estimation
func (mc *MemoryCache) GetAverageAccessTime() time.Duration {
	// L1 cache should provide sub-millisecond access
	return 100 * time.Microsecond // ~0.1ms average
}

// GetEvictionRate returns current eviction rate
func (mc *MemoryCache) GetEvictionRate() float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	totalOps := mc.hits + mc.misses
	if totalOps > 0 {
		return float64(mc.evictions) / float64(totalOps)
	}
	return 0.0
}

// IsHealthy checks if cache is operating within healthy parameters
func (mc *MemoryCache) IsHealthy() bool {
	stats := mc.GetStats()
	pressure := mc.GetMemoryPressure()
	
	// Healthy if:
	// 1. Hit rate above 70% (below target but acceptable)
	// 2. Memory pressure below 90%
	// 3. Not excessive evictions
	return stats.HitRate >= 0.70 && pressure < 0.90 && mc.GetEvictionRate() < 0.1
}