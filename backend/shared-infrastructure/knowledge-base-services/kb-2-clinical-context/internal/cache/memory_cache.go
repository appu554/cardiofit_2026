package cache

import (
	"container/list"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MemoryCache implements L1 in-memory LRU cache
type MemoryCache struct {
	maxSize    int
	defaultTTL time.Duration
	items      map[string]*list.Element
	lruList    *list.List
	mutex      sync.RWMutex
	logger     *zap.Logger
	metrics    *MemoryCacheMetrics
}

type memoryCacheItem struct {
	key       string
	value     interface{}
	expiresAt time.Time
	createdAt time.Time
	accessCount int64
}

type MemoryCacheMetrics struct {
	Hits        int64
	Misses      int64
	Evictions   int64
	Expirations int64
	TotalSets   int64
	TotalGets   int64
	MemoryUsage int64
	mutex       sync.RWMutex
}

// NewMemoryCache creates a new in-memory LRU cache
func NewMemoryCache(maxSize int, defaultTTL time.Duration, logger *zap.Logger) *MemoryCache {
	if maxSize <= 0 {
		maxSize = 10000 // Default size
	}
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute // Default TTL
	}

	mc := &MemoryCache{
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		items:      make(map[string]*list.Element),
		lruList:    list.New(),
		logger:     logger,
		metrics:    &MemoryCacheMetrics{},
	}

	// Start cleanup goroutine
	go mc.cleanupExpired()

	return mc
}

// Get retrieves a value from the cache
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.metrics.mutex.Lock()
	mc.metrics.TotalGets++
	mc.metrics.mutex.Unlock()

	element, exists := mc.items[key]
	if !exists {
		mc.recordMiss()
		return nil, false
	}

	item := element.Value.(*memoryCacheItem)

	// Check expiration
	if time.Now().After(item.expiresAt) {
		mc.removeElement(element)
		mc.recordExpiration()
		mc.recordMiss()
		return nil, false
	}

	// Move to front (LRU)
	mc.lruList.MoveToFront(element)
	item.accessCount++

	mc.recordHit()
	
	mc.logger.Debug("Memory cache hit", 
		zap.String("key", key),
		zap.Int64("access_count", item.accessCount))

	return item.value, true
}

// Set stores a value in the cache
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.metrics.mutex.Lock()
	mc.metrics.TotalSets++
	mc.metrics.mutex.Unlock()

	if ttl <= 0 {
		ttl = mc.defaultTTL
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	// Check if key already exists
	if element, exists := mc.items[key]; exists {
		// Update existing item
		item := element.Value.(*memoryCacheItem)
		item.value = value
		item.expiresAt = expiresAt
		item.accessCount++
		
		// Move to front
		mc.lruList.MoveToFront(element)
		
		mc.logger.Debug("Memory cache update", zap.String("key", key))
		return
	}

	// Create new item
	item := &memoryCacheItem{
		key:         key,
		value:       value,
		expiresAt:   expiresAt,
		createdAt:   now,
		accessCount: 1,
	}

	// Add to front of LRU list
	element := mc.lruList.PushFront(item)
	mc.items[key] = element

	// Check size limit and evict if necessary
	if mc.lruList.Len() > mc.maxSize {
		mc.evictOldest()
	}

	// Update memory usage estimate
	mc.updateMemoryUsage()

	mc.logger.Debug("Memory cache set", 
		zap.String("key", key),
		zap.Duration("ttl", ttl),
		zap.Int("cache_size", mc.lruList.Len()))
}

// Delete removes a key from the cache
func (mc *MemoryCache) Delete(key string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if element, exists := mc.items[key]; exists {
		mc.removeElement(element)
		mc.logger.Debug("Memory cache delete", zap.String("key", key))
	}
}

// InvalidatePattern removes all keys matching a pattern
func (mc *MemoryCache) InvalidatePattern(pattern string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	keysToDelete := make([]string, 0)
	
	for key := range mc.items {
		// Simple pattern matching - contains check
		if strings.Contains(key, pattern) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		if element, exists := mc.items[key]; exists {
			mc.removeElement(element)
		}
	}

	mc.logger.Info("Memory cache pattern invalidation", 
		zap.String("pattern", pattern),
		zap.Int("invalidated_keys", len(keysToDelete)))
}

// Clear removes all items from the cache
func (mc *MemoryCache) Clear() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	itemCount := mc.lruList.Len()
	mc.items = make(map[string]*list.Element)
	mc.lruList.Init()

	mc.logger.Info("Memory cache cleared", zap.Int("items_removed", itemCount))
}

// ItemCount returns the number of items in the cache
func (mc *MemoryCache) ItemCount() int {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.lruList.Len()
}

// GetMetrics returns cache performance metrics
func (mc *MemoryCache) GetMetrics() map[string]interface{} {
	mc.metrics.mutex.RLock()
	defer mc.metrics.mutex.RUnlock()

	totalRequests := mc.metrics.Hits + mc.metrics.Misses
	hitRate := float64(0)
	if totalRequests > 0 {
		hitRate = float64(mc.metrics.Hits) / float64(totalRequests) * 100
	}

	return map[string]interface{}{
		"hits":         mc.metrics.Hits,
		"misses":       mc.metrics.Misses,
		"evictions":    mc.metrics.Evictions,
		"expirations":  mc.metrics.Expirations,
		"total_sets":   mc.metrics.TotalSets,
		"total_gets":   mc.metrics.TotalGets,
		"hit_rate_pct": hitRate,
		"item_count":   mc.ItemCount(),
		"memory_usage": mc.metrics.MemoryUsage,
		"max_size":     mc.maxSize,
	}
}

// Private helper methods

func (mc *MemoryCache) removeElement(element *list.Element) {
	item := element.Value.(*memoryCacheItem)
	delete(mc.items, item.key)
	mc.lruList.Remove(element)
}

func (mc *MemoryCache) evictOldest() {
	element := mc.lruList.Back()
	if element != nil {
		mc.removeElement(element)
		mc.recordEviction()
		mc.logger.Debug("Memory cache eviction", zap.Int("cache_size", mc.lruList.Len()))
	}
}

func (mc *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.performCleanup()
		}
	}
}

func (mc *MemoryCache) performCleanup() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	now := time.Now()
	var expiredKeys []string

	// Find expired items
	for element := mc.lruList.Back(); element != nil; element = element.Prev() {
		item := element.Value.(*memoryCacheItem)
		if now.After(item.expiresAt) {
			expiredKeys = append(expiredKeys, item.key)
		} else {
			// Since we're going from back to front and items are sorted by access time,
			// we can break early if we find a non-expired item
			break
		}
	}

	// Remove expired items
	for _, key := range expiredKeys {
		if element, exists := mc.items[key]; exists {
			mc.removeElement(element)
			mc.recordExpiration()
		}
	}

	if len(expiredKeys) > 0 {
		mc.logger.Debug("Memory cache cleanup", 
			zap.Int("expired_items", len(expiredKeys)),
			zap.Int("remaining_items", mc.lruList.Len()))
	}

	// Update memory usage
	mc.updateMemoryUsage()
}

func (mc *MemoryCache) updateMemoryUsage() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// Estimate cache memory usage (rough approximation)
	itemCount := mc.lruList.Len()
	estimatedUsage := int64(itemCount * 1024) // Rough estimate: 1KB per item

	mc.metrics.mutex.Lock()
	mc.metrics.MemoryUsage = estimatedUsage
	mc.metrics.mutex.Unlock()
}

// Metrics recording methods
func (mc *MemoryCache) recordHit() {
	mc.metrics.mutex.Lock()
	mc.metrics.Hits++
	mc.metrics.mutex.Unlock()
}

func (mc *MemoryCache) recordMiss() {
	mc.metrics.mutex.Lock()
	mc.metrics.Misses++
	mc.metrics.mutex.Unlock()
}

func (mc *MemoryCache) recordEviction() {
	mc.metrics.mutex.Lock()
	mc.metrics.Evictions++
	mc.metrics.mutex.Unlock()
}

func (mc *MemoryCache) recordExpiration() {
	mc.metrics.mutex.Lock()
	mc.metrics.Expirations++
	mc.metrics.mutex.Unlock()
}

// Close cleans up the memory cache
func (mc *MemoryCache) Close() error {
	mc.Clear()
	
	mc.logger.Info("Memory cache closed",
		zap.Int64("total_hits", mc.metrics.Hits),
		zap.Int64("total_misses", mc.metrics.Misses),
		zap.Int64("total_evictions", mc.metrics.Evictions))

	return nil
}

// Advanced cache operations

// GetWithMetadata returns value along with cache metadata
func (mc *MemoryCache) GetWithMetadata(key string) (interface{}, map[string]interface{}, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	element, exists := mc.items[key]
	if !exists {
		return nil, nil, false
	}

	item := element.Value.(*memoryCacheItem)

	// Check expiration
	if time.Now().After(item.expiresAt) {
		return nil, nil, false
	}

	metadata := map[string]interface{}{
		"created_at":    item.createdAt,
		"expires_at":    item.expiresAt,
		"access_count":  item.accessCount,
		"ttl_remaining": time.Until(item.expiresAt),
	}

	return item.value, metadata, true
}

// SetWithCustomTTL allows setting items with custom TTL
func (mc *MemoryCache) SetWithCustomTTL(key string, value interface{}, ttl time.Duration) {
	mc.Set(key, value, ttl)
}

// GetKeys returns all current cache keys (for debugging)
func (mc *MemoryCache) GetKeys() []string {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	keys := make([]string, 0, len(mc.items))
	for key := range mc.items {
		keys = append(keys, key)
	}
	return keys
}

// GetExpiredKeys returns keys that have expired but not yet cleaned up
func (mc *MemoryCache) GetExpiredKeys() []string {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	now := time.Now()
	var expiredKeys []string

	for key, element := range mc.items {
		item := element.Value.(*memoryCacheItem)
		if now.After(item.expiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	return expiredKeys
}