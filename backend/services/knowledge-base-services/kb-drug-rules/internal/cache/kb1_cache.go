package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// KB1Cache implements KB-1 specification compliant caching
// Key pattern: dose:v2:{drug_code}:{context_hash}
// TTL: 1 hour with immediate invalidation on rule updates
type KB1Cache struct {
	client *redis.Client
	logger *logrus.Logger
	ctx    context.Context
}

// KB1CacheInterface defines KB-1 specific caching operations
type KB1CacheInterface interface {
	// Core operations with KB-1 key patterns
	GetDosingRule(drugCode, contextHash string) ([]byte, error)
	SetDosingRule(drugCode, contextHash string, data []byte) error
	InvalidateDrugCode(drugCode string) error

	// Context-aware operations (for gRPC and enhanced API)
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttlSeconds int) error

	// Context hash generation for KB-1 specification
	GenerateContextHash(patientContext map[string]interface{}) string

	// Cache prewarming for top medications
	PrewarmTopMedications(drugCodes []string, getRuleFunc func(string) ([]byte, error)) error

	// Cache invalidation listener
	ListenForInvalidationEvents() error

	// Performance monitoring
	GetKB1Stats() (*KB1CacheStats, error)

	// Health and lifecycle
	Ping() error
	Close() error

	// Legacy compatibility (explicit - NOT embedding Client to avoid conflicts)
	Delete(key string) error
	InvalidatePattern(pattern string) error
	GetStats() (map[string]interface{}, error)
}

// KB1CacheStats represents KB-1 specific cache performance metrics
type KB1CacheStats struct {
	HitCount       int64   `json:"hit_count"`
	MissCount      int64   `json:"miss_count"`
	HitRate        float64 `json:"hit_rate"`
	KeyCount       int64   `json:"key_count"`
	DosingKeyCount int64   `json:"dosing_key_count"` // Keys matching dose:v2:* pattern
	MemoryUsage    int64   `json:"memory_usage_bytes"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	SLOCompliance  float64 `json:"slo_compliance_percent"` // % of requests < 5ms
}

// NewKB1Cache creates a KB-1 specification compliant Redis cache
func NewKB1Cache(redisURL string, logger *logrus.Logger) (*KB1Cache, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Optimize Redis connection for KB-1 SLO requirements (p95 < 60ms overall, cache < 5ms)
	opts.PoolSize = 50              // High concurrency support
	opts.MinIdleConns = 10          // Keep connections warm
	opts.MaxIdleConns = 20          // Prevent connection exhaustion
	opts.DialTimeout = 2 * time.Second
	opts.ReadTimeout = 5 * time.Millisecond   // Aggressive timeout for sub-5ms cache ops
	opts.WriteTimeout = 5 * time.Millisecond
	opts.PoolTimeout = 10 * time.Second

	client := redis.NewClient(opts)
	
	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &KB1Cache{
		client: client,
		logger: logger,
		ctx:    ctx,
	}, nil
}

// GetDosingRule retrieves dosing rule using KB-1 key pattern
func (k *KB1Cache) GetDosingRule(drugCode, contextHash string) ([]byte, error) {
	start := time.Now()
	key := fmt.Sprintf("dose:v2:%s:%s", drugCode, contextHash)
	
	defer func() {
		duration := time.Since(start)
		k.logger.WithFields(logrus.Fields{
			"operation":   "kb1_cache_get",
			"drug_code":   drugCode,
			"key":         key,
			"duration_ms": duration.Milliseconds(),
		}).Debug("KB-1 cache GET operation")
		
		// Alert if cache operation is slow (should be < 5ms for KB-1 SLO)
		if duration > 5*time.Millisecond {
			k.logger.WithFields(logrus.Fields{
				"drug_code":   drugCode,
				"duration_ms": duration.Milliseconds(),
				"slo_breach":  true,
			}).Warn("KB-1 cache GET operation exceeded 5ms SLO")
		}
	}()

	val, err := k.client.Get(k.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("KB-1 cache GET failed for key %s: %w", key, err)
	}

	return []byte(val), nil
}

// SetDosingRule stores dosing rule using KB-1 key pattern with 1-hour TTL
func (k *KB1Cache) SetDosingRule(drugCode, contextHash string, data []byte) error {
	start := time.Now()
	key := fmt.Sprintf("dose:v2:%s:%s", drugCode, contextHash)
	
	defer func() {
		duration := time.Since(start)
		k.logger.WithFields(logrus.Fields{
			"operation":   "kb1_cache_set",
			"drug_code":   drugCode,
			"key":         key,
			"size_bytes":  len(data),
			"duration_ms": duration.Milliseconds(),
		}).Debug("KB-1 cache SET operation")
		
		if duration > 5*time.Millisecond {
			k.logger.WithFields(logrus.Fields{
				"drug_code":   drugCode,
				"duration_ms": duration.Milliseconds(),
				"slo_breach":  true,
			}).Warn("KB-1 cache SET operation exceeded 5ms SLO")
		}
	}()

	// KB-1 specification: 1 hour TTL
	return k.client.Set(k.ctx, key, data, time.Hour).Err()
}

// InvalidateDrugCode invalidates all cache entries for a drug code
// Implements KB-1 specification: immediate invalidation on rule updates
func (k *KB1Cache) InvalidateDrugCode(drugCode string) error {
	start := time.Now()
	pattern := fmt.Sprintf("dose:v2:%s:*", drugCode)
	
	defer func() {
		k.logger.WithFields(logrus.Fields{
			"operation":   "kb1_cache_invalidate",
			"drug_code":   drugCode,
			"pattern":     pattern,
			"duration_ms": time.Since(start).Milliseconds(),
		}).Info("KB-1 cache invalidation for drug code")
	}()

	// Use SCAN for production safety (avoids blocking Redis)
	var cursor uint64
	var deletedCount int
	
	for {
		var keys []string
		var err error
		
		keys, cursor, err = k.client.Scan(k.ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan KB-1 cache keys for drug %s: %w", drugCode, err)
		}
		
		if len(keys) > 0 {
			// Delete in pipeline for efficiency
			pipeline := k.client.Pipeline()
			for _, key := range keys {
				pipeline.Del(k.ctx, key)
			}
			
			if _, err := pipeline.Exec(k.ctx); err != nil {
				k.logger.WithError(err).Error("Failed to execute cache invalidation pipeline")
			} else {
				deletedCount += len(keys)
			}
		}
		
		if cursor == 0 {
			break
		}
	}
	
	k.logger.WithFields(logrus.Fields{
		"drug_code":     drugCode,
		"deleted_count": deletedCount,
	}).Info("KB-1 cache invalidation completed")

	return nil
}

// GenerateContextHash creates deterministic hash from patient context
// Implements KB-1 specification: only relevant context variance creates cache keys
func (k *KB1Cache) GenerateContextHash(patientContext map[string]interface{}) string {
	// Only include fields that actually affect dosing calculations
	relevantFields := []string{
		"weight_kg", "egfr", "age_years", "sex", "pregnant", 
		"creatinine_clearance", "dialysis_type", "child_pugh_class",
		"alt", "ast", "bilirubin", "albumin",
	}
	
	// Create normalized context with only relevant fields
	normalized := make(map[string]interface{})
	for _, field := range relevantFields {
		if val, exists := patientContext[field]; exists && val != nil {
			normalized[field] = val
		}
	}
	
	// Add extra numeric fields that might be relevant
	if extraNumeric, ok := patientContext["extra_numeric"].(map[string]interface{}); ok {
		for key, val := range extraNumeric {
			if val != nil {
				normalized[fmt.Sprintf("extra_%s", key)] = val
			}
		}
	}
	
	// Create deterministic sorted representation
	var keys []string
	for k := range normalized {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	var parts []string
	for _, k := range keys {
		val := normalized[k]
		parts = append(parts, fmt.Sprintf("%s=%v", k, val))
	}
	
	// Generate SHA256 hash and return first 16 chars for readable cache keys
	hashInput := strings.Join(parts, "&")
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:8])
}

// PrewarmTopMedications implements KB-1 prewarming for top 100-200 prescribed medications
func (k *KB1Cache) PrewarmTopMedications(drugCodes []string, getRuleFunc func(string) ([]byte, error)) error {
	start := time.Now()
	k.logger.WithField("drug_count", len(drugCodes)).Info("Starting KB-1 cache prewarming")
	
	successCount := 0
	failureCount := 0
	
	// Process in batches for efficiency
	batchSize := 10
	for i := 0; i < len(drugCodes); i += batchSize {
		end := i + batchSize
		if end > len(drugCodes) {
			end = len(drugCodes)
		}
		
		batch := drugCodes[i:end]
		pipeline := k.client.Pipeline()
		
		for _, drugCode := range batch {
			ruleData, err := getRuleFunc(drugCode)
			if err != nil {
				k.logger.WithError(err).WithField("drug_code", drugCode).Warn("Failed to get rule for prewarming")
				failureCount++
				continue
			}
			
			// Generate base cache key (common patient context)
			baseContextHash := k.GenerateContextHash(map[string]interface{}{
				"weight_kg": 70.0,  // Average adult weight
				"age_years": 45,    // Average adult age
				"egfr":      90.0,  // Normal kidney function
				"sex":       "M",   // Male
			})
			
			cacheKey := fmt.Sprintf("dose:v2:%s:%s", drugCode, baseContextHash)
			pipeline.Set(k.ctx, cacheKey, ruleData, time.Hour)
			successCount++
		}
		
		// Execute batch
		if _, err := pipeline.Exec(k.ctx); err != nil {
			k.logger.WithError(err).Error("Pipeline execution failed during prewarming")
		}
	}
	
	duration := time.Since(start)
	k.logger.WithFields(logrus.Fields{
		"total_drugs":   len(drugCodes),
		"success_count": successCount,
		"failure_count": failureCount,
		"duration_ms":   duration.Milliseconds(),
	}).Info("KB-1 cache prewarming completed")

	return nil
}

// GetKB1Stats retrieves KB-1 specific cache performance statistics
func (k *KB1Cache) GetKB1Stats() (*KB1CacheStats, error) {
	info, err := k.client.Info(k.ctx, "stats", "memory").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis stats: %w", err)
	}

	stats := &KB1CacheStats{}
	
	// Parse Redis INFO output
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		switch key {
		case "keyspace_hits":
			if hits, err := parseRedisInt64(value); err == nil {
				stats.HitCount = hits
			}
		case "keyspace_misses":
			if misses, err := parseRedisInt64(value); err == nil {
				stats.MissCount = misses
			}
		case "used_memory":
			if memory, err := parseRedisInt64(value); err == nil {
				stats.MemoryUsage = memory
			}
		}
	}
	
	// Calculate hit rate
	total := stats.HitCount + stats.MissCount
	if total > 0 {
		stats.HitRate = float64(stats.HitCount) / float64(total) * 100
	}
	
	// Get total key count
	if dbSize, err := k.client.DBSize(k.ctx).Result(); err == nil {
		stats.KeyCount = dbSize
	}
	
	// Count KB-1 dosing keys specifically
	dosingKeyCount, _ := k.countDosingKeys()
	stats.DosingKeyCount = dosingKeyCount
	
	// Calculate SLO compliance (placeholder - would track actual latencies)
	stats.SLOCompliance = 95.0 // TODO: Implement actual SLO tracking
	
	return stats, nil
}

// countDosingKeys counts keys matching the KB-1 dose pattern
func (k *KB1Cache) countDosingKeys() (int64, error) {
	var cursor uint64
	var count int64
	pattern := "dose:v2:*"
	
	for {
		keys, nextCursor, err := k.client.Scan(k.ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return 0, err
		}
		
		count += int64(len(keys))
		cursor = nextCursor
		
		if cursor == 0 {
			break
		}
	}
	
	return count, nil
}

// ListenForInvalidationEvents listens for PostgreSQL notifications to invalidate cache
func (k *KB1Cache) ListenForInvalidationEvents() error {
	// Subscribe to PostgreSQL notifications for rule updates
	pubsub := k.client.Subscribe(k.ctx, "rule_cache_invalidate")
	defer pubsub.Close()
	
	k.logger.Info("KB-1 cache started listening for invalidation events")
	
	for msg := range pubsub.Channel() {
		var notification struct {
			DrugCode        string `json:"drug_code"`
			SemanticVersion string `json:"semantic_version"`
			Operation       string `json:"operation"`
		}
		
		if err := json.Unmarshal([]byte(msg.Payload), &notification); err != nil {
			k.logger.WithError(err).Error("Failed to parse cache invalidation notification")
			continue
		}
		
		// Invalidate all cache entries for the affected drug
		if err := k.InvalidateDrugCode(notification.DrugCode); err != nil {
			k.logger.WithError(err).WithField("drug_code", notification.DrugCode).Error("Failed to invalidate cache for drug")
		} else {
			k.logger.WithFields(logrus.Fields{
				"drug_code": notification.DrugCode,
				"version":   notification.SemanticVersion,
				"operation": notification.Operation,
			}).Info("Cache invalidated due to rule update")
		}
	}
	
	return nil
}

// Implement KB1CacheInterface methods

// Get retrieves a value from cache (legacy interface)
func (k *KB1Cache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := k.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return []byte(val), nil
}

// Set stores a value in cache with TTL in seconds (context-aware)
func (k *KB1Cache) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	return k.client.Set(ctx, key, value, time.Duration(ttlSeconds)*time.Second).Err()
}

// GetLegacy retrieves a value from cache (legacy interface without context)
func (k *KB1Cache) GetLegacy(key string) ([]byte, error) {
	val, err := k.client.Get(k.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return []byte(val), nil
}

// SetLegacy stores a value in cache with TTL (legacy interface without context)
func (k *KB1Cache) SetLegacy(key string, value []byte, ttl time.Duration) error {
	return k.client.Set(k.ctx, key, value, ttl).Err()
}

func (k *KB1Cache) Delete(key string) error {
	return k.client.Del(k.ctx, key).Err()
}

func (k *KB1Cache) InvalidatePattern(pattern string) error {
	return k.InvalidateDrugCode(strings.TrimSuffix(strings.TrimPrefix(pattern, "dose:v2:"), ":*"))
}

func (k *KB1Cache) Ping() error {
	return k.client.Ping(k.ctx).Err()
}

func (k *KB1Cache) Close() error {
	return k.client.Close()
}

func (k *KB1Cache) GetStats() (map[string]interface{}, error) {
	kb1Stats, err := k.GetKB1Stats()
	if err != nil {
		return nil, err
	}
	
	// Convert to generic map for backward compatibility
	return map[string]interface{}{
		"hit_count":         kb1Stats.HitCount,
		"miss_count":        kb1Stats.MissCount,
		"hit_rate":          kb1Stats.HitRate,
		"key_count":         kb1Stats.KeyCount,
		"dosing_key_count":  kb1Stats.DosingKeyCount,
		"memory_usage":      kb1Stats.MemoryUsage,
		"slo_compliance":    kb1Stats.SLOCompliance,
	}, nil
}

// Helper functions

func parseRedisInt64(value string) (int64, error) {
	cleaned := strings.TrimSpace(value)
	if idx := strings.Index(cleaned, " "); idx != -1 {
		cleaned = cleaned[:idx]
	}
	
	var result int64
	if _, err := fmt.Sscanf(cleaned, "%d", &result); err != nil {
		return 0, err
	}
	return result, nil
}

// Interface is an alias for KB1CacheInterface for legacy compatibility
type Interface = KB1CacheInterface

// CacheKeyBuilder helps build KB-1 compliant cache keys
type CacheKeyBuilder struct{}

// BuildDosingKey creates a KB-1 compliant dosing cache key
func (ckb *CacheKeyBuilder) BuildDosingKey(drugCode, contextHash string) string {
	return fmt.Sprintf("dose:v2:%s:%s", drugCode, contextHash)
}

// BuildMetadataKey creates a cache key for rule metadata
func (ckb *CacheKeyBuilder) BuildMetadataKey(drugCode, version string) string {
	return fmt.Sprintf("meta:v2:%s:%s", drugCode, version)
}

// BuildAvailabilityKey creates a cache key for rule availability checks
func (ckb *CacheKeyBuilder) BuildAvailabilityKey(drugCode, region string) string {
	return fmt.Sprintf("avail:v2:%s:%s", drugCode, region)
}

// ParseDosingKey extracts drug code and context hash from a dosing cache key
func (ckb *CacheKeyBuilder) ParseDosingKey(key string) (drugCode, contextHash string, err error) {
	parts := strings.Split(key, ":")
	if len(parts) != 4 || parts[0] != "dose" || parts[1] != "v2" {
		return "", "", fmt.Errorf("invalid dosing cache key format: %s", key)
	}
	return parts[2], parts[3], nil
}

// KB1ClientAdapter adapts KB1CacheInterface to the cache.Client interface
// This allows KB-1 enhanced cache to work with services that expect the legacy Client interface
type KB1ClientAdapter struct {
	kb1Cache KB1CacheInterface
	ctx      context.Context
}

// NewKB1ClientAdapter creates a new adapter that bridges KB1CacheInterface to Client
func NewKB1ClientAdapter(kb1Cache KB1CacheInterface) Client {
	return &KB1ClientAdapter{
		kb1Cache: kb1Cache,
		ctx:      context.Background(),
	}
}

// Get implements Client.Get by delegating to KB1CacheInterface
func (a *KB1ClientAdapter) Get(key string) ([]byte, error) {
	return a.kb1Cache.Get(a.ctx, key)
}

// Set implements Client.Set by delegating to KB1CacheInterface
func (a *KB1ClientAdapter) Set(key string, value []byte, ttl time.Duration) error {
	ttlSeconds := int(ttl.Seconds())
	return a.kb1Cache.Set(a.ctx, key, value, ttlSeconds)
}

// Delete implements Client.Delete
func (a *KB1ClientAdapter) Delete(key string) error {
	return a.kb1Cache.Delete(key)
}

// InvalidatePattern implements Client.InvalidatePattern
func (a *KB1ClientAdapter) InvalidatePattern(pattern string) error {
	return a.kb1Cache.InvalidatePattern(pattern)
}

// Ping implements Client.Ping
func (a *KB1ClientAdapter) Ping() error {
	return a.kb1Cache.Ping()
}

// Close implements Client.Close
func (a *KB1ClientAdapter) Close() error {
	return a.kb1Cache.Close()
}

// GetStats implements Client.GetStats
func (a *KB1ClientAdapter) GetStats() (map[string]interface{}, error) {
	return a.kb1Cache.GetStats()
}