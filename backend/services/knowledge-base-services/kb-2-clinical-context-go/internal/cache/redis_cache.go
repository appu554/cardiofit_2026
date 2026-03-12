package cache

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"kb-2-clinical-context-go/internal/metrics"
)

// RedisCacheConfig configures the Redis L2 cache
type RedisCacheConfig struct {
	Client        *redis.Client
	DefaultTTL    time.Duration // Default TTL (1 hour)
	MaxMemory     int64         // Maximum memory per node (1GB)
	KeyPrefix     string        // Key prefix for namespacing
	Compression   bool          // Enable compression for large objects
	HitRateTarget float64       // Target hit rate (0.95 = 95%)
}

// RedisCache implements distributed L2 caching with Redis
type RedisCache struct {
	config  *RedisCacheConfig
	client  *redis.Client
	metrics *metrics.PrometheusMetrics
	
	// Statistics (atomic counters for thread safety)
	hits         int64
	misses       int64
	operations   int64
	errors       int64
	compressions int64
	
	// Lua scripts for atomic operations
	getWithStatsScript *redis.Script
	setWithStatsScript *redis.Script
	cleanupScript      *redis.Script
}

// RedisItem represents a cached item in Redis with metadata
type RedisItem struct {
	Data         []byte    `json:"data"`
	ContentType  string    `json:"content_type"`
	CreatedAt    time.Time `json:"created_at"`
	TTL          int64     `json:"ttl_seconds"`
	Version      string    `json:"version"`
	Compressed   bool      `json:"compressed"`
	OriginalSize int       `json:"original_size"`
	AccessCount  int64     `json:"access_count"`
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(config *RedisCacheConfig, metricsCollector *metrics.PrometheusMetrics) *RedisCache {
	cache := &RedisCache{
		config:  config,
		client:  config.Client,
		metrics: metricsCollector,
	}
	
	// Initialize Lua scripts for atomic operations
	cache.initializeLuaScripts()
	
	return cache
}

// Get retrieves an item from Redis cache
func (rc *RedisCache) Get(ctx context.Context, key string) (interface{}, bool) {
	atomic.AddInt64(&rc.operations, 1)
	
	fullKey := rc.getFullKey(key)
	
	// Get item with statistics update using Lua script
	result, err := rc.getWithStatsScript.Run(ctx, rc.client, []string{fullKey}).Result()
	if err != nil {
		if err == redis.Nil {
			atomic.AddInt64(&rc.misses, 1)
			return nil, false
		}
		atomic.AddInt64(&rc.errors, 1)
		log.Printf("Redis cache get error for key %s: %v", key, err)
		return nil, false
	}
	
	// Parse result
	itemData, ok := result.(string)
	if !ok {
		atomic.AddInt64(&rc.errors, 1)
		return nil, false
	}
	
	var item RedisItem
	if err := json.Unmarshal([]byte(itemData), &item); err != nil {
		atomic.AddInt64(&rc.errors, 1)
		log.Printf("Redis cache item unmarshal error: %v", err)
		return nil, false
	}
	
	// Check TTL
	if time.Since(item.CreatedAt) > time.Duration(item.TTL)*time.Second {
		// Item expired, remove it
		go func() {
			rc.client.Del(context.Background(), fullKey)
		}()
		atomic.AddInt64(&rc.misses, 1)
		return nil, false
	}
	
	// Decompress if needed
	data := item.Data
	if item.Compressed {
		decompressed, err := rc.decompress(data)
		if err != nil {
			atomic.AddInt64(&rc.errors, 1)
			log.Printf("Redis cache decompression error: %v", err)
			return nil, false
		}
		data = decompressed
	}
	
	// Deserialize based on content type
	value, err := rc.deserialize(data, item.ContentType)
	if err != nil {
		atomic.AddInt64(&rc.errors, 1)
		log.Printf("Redis cache deserialization error: %v", err)
		return nil, false
	}
	
	atomic.AddInt64(&rc.hits, 1)
	return value, true
}

// Set stores an item in Redis cache
func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	atomic.AddInt64(&rc.operations, 1)
	
	fullKey := rc.getFullKey(key)
	
	// Serialize value
	data, contentType, err := rc.serialize(value)
	if err != nil {
		atomic.AddInt64(&rc.errors, 1)
		return fmt.Errorf("serialization failed: %w", err)
	}
	
	originalSize := len(data)
	compressed := false
	
	// Compress if enabled and data is large enough (>1KB)
	if rc.config.Compression && len(data) > 1024 {
		compressedData, err := rc.compress(data)
		if err == nil && len(compressedData) < len(data) {
			data = compressedData
			compressed = true
			atomic.AddInt64(&rc.compressions, 1)
		}
	}
	
	// Create Redis item
	item := RedisItem{
		Data:         data,
		ContentType:  contentType,
		CreatedAt:    time.Now(),
		TTL:          int64(ttl.Seconds()),
		Version:      "1.0",
		Compressed:   compressed,
		OriginalSize: originalSize,
		AccessCount:  0,
	}
	
	// Serialize item
	itemData, err := json.Marshal(item)
	if err != nil {
		atomic.AddInt64(&rc.errors, 1)
		return fmt.Errorf("item marshaling failed: %w", err)
	}
	
	// Store in Redis using Lua script for atomic operation
	_, err = rc.setWithStatsScript.Run(ctx, rc.client, []string{fullKey}, string(itemData), int64(ttl.Seconds())).Result()
	if err != nil {
		atomic.AddInt64(&rc.errors, 1)
		return fmt.Errorf("Redis set failed: %w", err)
	}
	
	return nil
}

// Delete removes an item from Redis cache
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := rc.getFullKey(key)
	
	err := rc.client.Del(ctx, fullKey).Err()
	if err != nil {
		atomic.AddInt64(&rc.errors, 1)
		return fmt.Errorf("Redis delete failed: %w", err)
	}
	
	return nil
}

// DeletePattern removes all keys matching a pattern
func (rc *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	fullPattern := rc.getFullKey(pattern)
	
	// Use SCAN to find matching keys
	keys, err := rc.scanKeys(ctx, fullPattern)
	if err != nil {
		return fmt.Errorf("pattern scan failed: %w", err)
	}
	
	if len(keys) == 0 {
		return nil // Nothing to delete
	}
	
	// Delete in batches to avoid blocking Redis
	batchSize := 100
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		
		batch := keys[i:end]
		if err := rc.client.Del(ctx, batch...).Err(); err != nil {
			log.Printf("Redis batch delete error: %v", err)
		}
	}
	
	return nil
}

// GetBatch retrieves multiple items efficiently using Redis pipeline
func (rc *RedisCache) GetBatch(ctx context.Context, keys []string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}
	
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = rc.getFullKey(key)
	}
	
	// Use pipeline for batch get
	pipe := rc.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(fullKeys))
	
	for i, fullKey := range fullKeys {
		cmds[i] = pipe.Get(ctx, fullKey)
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		atomic.AddInt64(&rc.errors, 1)
		return nil, fmt.Errorf("Redis pipeline exec failed: %w", err)
	}
	
	// Process results
	results := make(map[string]interface{})
	
	for i, cmd := range cmds {
		val, err := cmd.Result()
		if err == redis.Nil {
			atomic.AddInt64(&rc.misses, 1)
			continue
		}
		if err != nil {
			atomic.AddInt64(&rc.errors, 1)
			log.Printf("Redis batch get error for key %s: %v", keys[i], err)
			continue
		}
		
		// Deserialize item
		var item RedisItem
		if err := json.Unmarshal([]byte(val), &item); err != nil {
			log.Printf("Redis item unmarshal error: %v", err)
			continue
		}
		
		// Check expiration
		if time.Since(item.CreatedAt) > time.Duration(item.TTL)*time.Second {
			// Item expired
			go rc.client.Del(context.Background(), fullKeys[i])
			atomic.AddInt64(&rc.misses, 1)
			continue
		}
		
		// Decompress and deserialize
		data := item.Data
		if item.Compressed {
			if decompressed, err := rc.decompress(data); err == nil {
				data = decompressed
			}
		}
		
		if value, err := rc.deserialize(data, item.ContentType); err == nil {
			results[keys[i]] = value
			atomic.AddInt64(&rc.hits, 1)
		}
	}
	
	return results, nil
}

// SetBatch stores multiple items efficiently using Redis pipeline
func (rc *RedisCache) SetBatch(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}
	
	pipe := rc.client.Pipeline()
	
	for key, value := range items {
		fullKey := rc.getFullKey(key)
		
		// Serialize value
		data, contentType, err := rc.serialize(value)
		if err != nil {
			log.Printf("Batch serialization error for key %s: %v", key, err)
			continue
		}
		
		// Compress if applicable
		compressed := false
		if rc.config.Compression && len(data) > 1024 {
			if compressedData, err := rc.compress(data); err == nil && len(compressedData) < len(data) {
				data = compressedData
				compressed = true
			}
		}
		
		// Create item
		item := RedisItem{
			Data:         data,
			ContentType:  contentType,
			CreatedAt:    time.Now(),
			TTL:          int64(ttl.Seconds()),
			Compressed:   compressed,
			OriginalSize: len(data),
		}
		
		if itemData, err := json.Marshal(item); err == nil {
			pipe.Set(ctx, fullKey, string(itemData), ttl)
		}
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		atomic.AddInt64(&rc.errors, 1)
		return fmt.Errorf("Redis batch set failed: %w", err)
	}
	
	return nil
}

// GetStats returns current Redis cache statistics
func (rc *RedisCache) GetStats() *CacheStats {
	totalOps := atomic.LoadInt64(&rc.operations)
	hits := atomic.LoadInt64(&rc.hits)
	misses := atomic.LoadInt64(&rc.misses)
	
	hitRate := 0.0
	if totalOps > 0 {
		hitRate = float64(hits) / float64(totalOps)
	}
	
	// Get memory usage from Redis INFO
	memoryUsage := rc.getRedisMemoryUsage()
	
	return &CacheStats{
		HitRate:     hitRate,
		MissRate:    1.0 - hitRate,
		Operations:  totalOps,
		MemoryUsage: memoryUsage,
	}
}

// Optimize performs Redis cache optimization
func (rc *RedisCache) Optimize(ctx context.Context) error {
	// Remove expired keys using cleanup script
	_, err := rc.cleanupScript.Run(ctx, rc.client, []string{rc.config.KeyPrefix + "*"}).Result()
	if err != nil {
		return fmt.Errorf("Redis cleanup script failed: %w", err)
	}
	
	// Check memory pressure
	memoryInfo := rc.getRedisMemoryInfo(ctx)
	if memoryPressure := rc.calculateMemoryPressure(memoryInfo); memoryPressure > 0.8 {
		// High memory pressure - expire some keys early
		if err := rc.expireOldKeys(ctx, 0.1); err != nil {
			log.Printf("Redis memory pressure relief failed: %v", err)
		}
	}
	
	return nil
}

// Cleanup performs cache cleanup
func (rc *RedisCache) Cleanup(ctx context.Context) error {
	return rc.Optimize(ctx)
}

// Private methods

// getFullKey returns the full Redis key with prefix
func (rc *RedisCache) getFullKey(key string) string {
	return rc.config.KeyPrefix + key
}

// serialize converts a value to bytes with content type
func (rc *RedisCache) serialize(value interface{}) ([]byte, string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, "", fmt.Errorf("JSON marshaling failed: %w", err)
	}
	
	return data, "application/json", nil
}

// deserialize converts bytes back to original type
func (rc *RedisCache) deserialize(data []byte, contentType string) (interface{}, error) {
	if contentType != "application/json" {
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
	
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("JSON unmarshaling failed: %w", err)
	}
	
	return value, nil
}

// compress compresses data using gzip
func (rc *RedisCache) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	
	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, fmt.Errorf("compression write failed: %w", err)
	}
	
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("compression close failed: %w", err)
	}
	
	return buf.Bytes(), nil
}

// decompress decompresses gzip data
func (rc *RedisCache) decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decompression reader failed: %w", err)
	}
	defer reader.Close()
	
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("decompression read failed: %w", err)
	}
	
	return decompressed, nil
}

// scanKeys scans Redis for keys matching pattern
func (rc *RedisCache) scanKeys(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	var cursor uint64
	
	for {
		scanResult, err := rc.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("Redis scan failed: %w", err)
		}
		
		keys = append(keys, scanResult...)
		cursor = scanResult[1].(uint64)
		
		if cursor == 0 {
			break
		}
	}
	
	return keys, nil
}

// getRedisMemoryUsage gets current memory usage from Redis INFO
func (rc *RedisCache) getRedisMemoryUsage() int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	info, err := rc.client.Info(ctx, "memory").Result()
	if err != nil {
		return 0
	}
	
	// Parse used_memory from INFO output
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "used_memory:") {
			var memory int64
			if _, err := fmt.Sscanf(line, "used_memory:%d", &memory); err == nil {
				return memory
			}
		}
	}
	
	return 0
}

// getRedisMemoryInfo gets detailed memory information
func (rc *RedisCache) getRedisMemoryInfo(ctx context.Context) map[string]string {
	info, err := rc.client.Info(ctx, "memory").Result()
	if err != nil {
		return make(map[string]string)
	}
	
	memInfo := make(map[string]string)
	lines := strings.Split(info, "\n")
	
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			memInfo[parts[0]] = strings.TrimSpace(parts[1])
		}
	}
	
	return memInfo
}

// calculateMemoryPressure calculates memory pressure from Redis info
func (rc *RedisCache) calculateMemoryPressure(memInfo map[string]string) float64 {
	var usedMemory, maxMemory int64
	
	if used, exists := memInfo["used_memory"]; exists {
		fmt.Sscanf(used, "%d", &usedMemory)
	}
	
	if max, exists := memInfo["maxmemory"]; exists {
		fmt.Sscanf(max, "%d", &maxMemory)
	} else {
		maxMemory = rc.config.MaxMemory // Use configured max if Redis doesn't have limit
	}
	
	if maxMemory > 0 {
		return float64(usedMemory) / float64(maxMemory)
	}
	
	return 0.0
}

// expireOldKeys expires a percentage of old keys to relieve memory pressure
func (rc *RedisCache) expireOldKeys(ctx context.Context, percentage float64) error {
	// Get all keys with our prefix
	pattern := rc.config.KeyPrefix + "*"
	keys, err := rc.scanKeys(ctx, pattern)
	if err != nil {
		return fmt.Errorf("key scan failed: %w", err)
	}
	
	if len(keys) == 0 {
		return nil
	}
	
	// Expire a percentage of keys
	toExpire := int(float64(len(keys)) * percentage)
	if toExpire < 1 {
		toExpire = 1
	}
	
	// Expire keys using batch operation
	if toExpire < len(keys) {
		keysToExpire := keys[:toExpire]
		
		pipe := rc.client.Pipeline()
		for _, key := range keysToExpire {
			pipe.Expire(ctx, key, 1*time.Second) // Short expiration
		}
		
		_, err = pipe.Exec(ctx)
		if err != nil {
			return fmt.Errorf("batch expire failed: %w", err)
		}
	}
	
	return nil
}

// initializeLuaScripts initializes Lua scripts for atomic operations
func (rc *RedisCache) initializeLuaScripts() {
	// Script for atomic get with access count increment
	rc.getWithStatsScript = redis.NewScript(`
		local key = KEYS[1]
		local item = redis.call('GET', key)
		if item then
			-- Increment access count in the item metadata
			-- For simplicity, we'll just return the item
			-- In a production system, you might want to track access count
			return item
		end
		return nil
	`)
	
	// Script for atomic set with statistics
	rc.setWithStatsScript = redis.NewScript(`
		local key = KEYS[1]
		local value = ARGV[1]
		local ttl = tonumber(ARGV[2])
		
		redis.call('SET', key, value)
		if ttl > 0 then
			redis.call('EXPIRE', key, ttl)
		end
		
		return 'OK'
	`)
	
	// Script for cleanup of expired keys
	rc.cleanupScript = redis.NewScript(`
		local pattern = KEYS[1]
		local keys = redis.call('KEYS', pattern)
		local expired = 0
		
		for i=1,#keys do
			local ttl = redis.call('TTL', keys[i])
			if ttl == -2 then  -- Key doesn't exist or expired
				redis.call('DEL', keys[i])
				expired = expired + 1
			elseif ttl == -1 then  -- Key exists but no expiration
				-- Optionally set a default expiration
				redis.call('EXPIRE', keys[i], 3600)  -- 1 hour default
			end
		end
		
		return expired
	`)
}

// Health check methods

// IsHealthy checks if Redis cache is healthy
func (rc *RedisCache) IsHealthy(ctx context.Context) bool {
	// Test Redis connectivity
	if err := rc.client.Ping(ctx).Err(); err != nil {
		return false
	}
	
	// Check hit rate
	stats := rc.GetStats()
	if stats.HitRate < 0.7 { // Below 70% is concerning
		return false
	}
	
	// Check memory pressure
	memInfo := rc.getRedisMemoryInfo(ctx)
	if pressure := rc.calculateMemoryPressure(memInfo); pressure > 0.9 {
		return false
	}
	
	return true
}

// GetConnectionInfo returns Redis connection information
func (rc *RedisCache) GetConnectionInfo() map[string]interface{} {
	return map[string]interface{}{
		"redis_addr":     rc.client.Options().Addr,
		"redis_db":       rc.client.Options().DB,
		"key_prefix":     rc.config.KeyPrefix,
		"compression":    rc.config.Compression,
		"default_ttl":    rc.config.DefaultTTL.String(),
		"hit_rate_target": rc.config.HitRateTarget,
	}
}

// Performance optimization methods

// GetCompressionRatio returns compression efficiency statistics
func (rc *RedisCache) GetCompressionRatio() float64 {
	totalOps := atomic.LoadInt64(&rc.operations)
	compressions := atomic.LoadInt64(&rc.compressions)
	
	if totalOps > 0 {
		return float64(compressions) / float64(totalOps)
	}
	return 0.0
}

// OptimizeCompression adjusts compression strategy based on performance
func (rc *RedisCache) OptimizeCompression(ctx context.Context) error {
	ratio := rc.GetCompressionRatio()
	
	// If compression ratio is low, consider disabling for small items
	if ratio < 0.1 {
		log.Printf("Low compression ratio (%.2f%%), consider adjusting compression threshold", ratio*100)
	}
	
	return nil
}

// ResetStats resets cache statistics counters
func (rc *RedisCache) ResetStats() {
	atomic.StoreInt64(&rc.hits, 0)
	atomic.StoreInt64(&rc.misses, 0)
	atomic.StoreInt64(&rc.operations, 0)
	atomic.StoreInt64(&rc.errors, 0)
	atomic.StoreInt64(&rc.compressions, 0)
}