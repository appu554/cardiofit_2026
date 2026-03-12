package context

import (
	stdcontext "context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// CacheManager implements multi-level caching (L1: in-memory, L2: Redis, L3: database)
type CacheManager struct {
	l1Cache     *sync.Map          // In-memory cache
	l2Client    *redis.Client      // Redis client
	config      config.CachingConfig
	logger      *logger.Logger
	l1Stats     *CacheStats
	l2Stats     *CacheStats
	mutex       sync.RWMutex
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits        int64 `json:"hits"`
	Misses      int64 `json:"misses"`
	Sets        int64 `json:"sets"`
	Deletes     int64 `json:"deletes"`
	Errors      int64 `json:"errors"`
	LastAccess  time.Time `json:"last_access"`
}

// CacheEntry represents a cached entry with metadata
type CacheEntry struct {
	Data      *types.ClinicalContext `json:"data"`
	ExpiresAt time.Time              `json:"expires_at"`
	CreatedAt time.Time              `json:"created_at"`
	Version   string                 `json:"version"`
}

// NewCacheManager creates a new cache manager
func NewCacheManager(cfg config.CachingConfig, logger *logger.Logger) *CacheManager {
	cm := &CacheManager{
		l1Cache: &sync.Map{},
		config:  cfg,
		logger:  logger,
		l1Stats: &CacheStats{},
		l2Stats: &CacheStats{},
	}

	// Initialize Redis client if enabled
	if cfg.Redis.Enabled {
		cm.l2Client = redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Address,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
			PoolSize: cfg.Redis.PoolSize,
		})

		// Test Redis connection
		ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 5*time.Second)
		defer cancel()

		if err := cm.l2Client.Ping(ctx).Err(); err != nil {
			logger.Warn("Redis connection failed, L2 cache disabled", zap.Error(err))
			cm.l2Client = nil
		} else {
			logger.Info("Redis L2 cache initialized", zap.String("address", cfg.Redis.Address))
		}
	}

	return cm
}

// Get retrieves a clinical context from cache
func (cm *CacheManager) Get(patientID string) *types.ClinicalContext {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// L1 Cache (in-memory) - fastest
	if entry, ok := cm.l1Cache.Load(patientID); ok {
		cacheEntry := entry.(*CacheEntry)
		
		// Check expiration
		if time.Now().Before(cacheEntry.ExpiresAt) {
			cm.l1Stats.Hits++
			cm.l1Stats.LastAccess = time.Now()
			
			cm.logger.Debug("L1 cache hit", zap.String("patient_id", patientID))
			return cacheEntry.Data
		} else {
			// Expired entry, remove it
			cm.l1Cache.Delete(patientID)
		}
	}
	cm.l1Stats.Misses++

	// L2 Cache (Redis) - fast
	if cm.l2Client != nil {
		ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 100*time.Millisecond)
		defer cancel()

		data, err := cm.l2Client.Get(ctx, cm.getRedisKey(patientID)).Result()
		if err == nil {
			var cacheEntry CacheEntry
			if err := json.Unmarshal([]byte(data), &cacheEntry); err == nil {
				// Check expiration
				if time.Now().Before(cacheEntry.ExpiresAt) {
					cm.l2Stats.Hits++
					cm.l2Stats.LastAccess = time.Now()
					
					// Promote to L1 cache
					cm.l1Cache.Store(patientID, &cacheEntry)
					
					cm.logger.Debug("L2 cache hit, promoted to L1", zap.String("patient_id", patientID))
					return cacheEntry.Data
				}
			}
		} else if err != redis.Nil {
			cm.l2Stats.Errors++
			cm.logger.Warn("L2 cache error", zap.String("patient_id", patientID), zap.Error(err))
		}
	}
	cm.l2Stats.Misses++

	// Cache miss at all levels
	cm.logger.Debug("Cache miss at all levels", zap.String("patient_id", patientID))
	return nil
}

// Set stores a clinical context in cache
func (cm *CacheManager) Set(patientID string, context *types.ClinicalContext, ttl time.Duration) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	expiresAt := time.Now().Add(ttl)
	cacheEntry := &CacheEntry{
		Data:      context,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		Version:   context.ContextVersion,
	}

	// Store in L1 cache (in-memory)
	cm.l1Cache.Store(patientID, cacheEntry)
	cm.l1Stats.Sets++

	// Store in L2 cache (Redis)
	if cm.l2Client != nil {
		go func() {
			ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 200*time.Millisecond)
			defer cancel()

			data, err := json.Marshal(cacheEntry)
			if err != nil {
				cm.l2Stats.Errors++
				cm.logger.Warn("Failed to marshal cache entry for L2", zap.String("patient_id", patientID), zap.Error(err))
				return
			}

			err = cm.l2Client.Set(ctx, cm.getRedisKey(patientID), data, ttl).Err()
			if err != nil {
				cm.l2Stats.Errors++
				cm.logger.Warn("Failed to set L2 cache", zap.String("patient_id", patientID), zap.Error(err))
			} else {
				cm.l2Stats.Sets++
				cm.logger.Debug("Context cached in L2", zap.String("patient_id", patientID), zap.Duration("ttl", ttl))
			}
		}()
	}

	cm.logger.Debug("Context cached",
		zap.String("patient_id", patientID),
		zap.Duration("ttl", ttl),
		zap.String("context_version", context.ContextVersion),
	)
}

// Delete removes a clinical context from cache
func (cm *CacheManager) Delete(patientID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Delete from L1 cache
	cm.l1Cache.Delete(patientID)
	cm.l1Stats.Deletes++

	// Delete from L2 cache
	if cm.l2Client != nil {
		go func() {
			ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 100*time.Millisecond)
			defer cancel()

			err := cm.l2Client.Del(ctx, cm.getRedisKey(patientID)).Err()
			if err != nil {
				cm.l2Stats.Errors++
				cm.logger.Warn("Failed to delete from L2 cache", zap.String("patient_id", patientID), zap.Error(err))
			} else {
				cm.l2Stats.Deletes++
			}
		}()
	}

	cm.logger.Debug("Context cache deleted", zap.String("patient_id", patientID))
}

// Clear clears all cache entries
func (cm *CacheManager) Clear() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Clear L1 cache
	cm.l1Cache = &sync.Map{}

	// Clear L2 cache (Redis) - be careful with this in production
	if cm.l2Client != nil {
		go func() {
			ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 5*time.Second)
			defer cancel()

			// Only clear keys with our prefix
			keys, err := cm.l2Client.Keys(ctx, cm.getRedisKey("*")).Result()
			if err != nil {
				cm.logger.Warn("Failed to get Redis keys for clearing", zap.Error(err))
				return
			}

			if len(keys) > 0 {
				err = cm.l2Client.Del(ctx, keys...).Err()
				if err != nil {
					cm.logger.Warn("Failed to clear L2 cache", zap.Error(err))
				} else {
					cm.logger.Info("L2 cache cleared", zap.Int("keys_deleted", len(keys)))
				}
			}
		}()
	}

	cm.logger.Info("Cache cleared")
}

// GetStats returns cache statistics
func (cm *CacheManager) GetStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Count L1 cache entries
	l1Count := 0
	cm.l1Cache.Range(func(key, value interface{}) bool {
		l1Count++
		return true
	})

	stats := map[string]interface{}{
		"l1_cache": map[string]interface{}{
			"enabled":     true,
			"entries":     l1Count,
			"hits":        cm.l1Stats.Hits,
			"misses":      cm.l1Stats.Misses,
			"sets":        cm.l1Stats.Sets,
			"deletes":     cm.l1Stats.Deletes,
			"hit_rate":    cm.calculateHitRate(cm.l1Stats.Hits, cm.l1Stats.Misses),
			"last_access": cm.l1Stats.LastAccess,
		},
		"l2_cache": map[string]interface{}{
			"enabled": cm.l2Client != nil,
		},
	}

	if cm.l2Client != nil {
		stats["l2_cache"] = map[string]interface{}{
			"enabled":     true,
			"hits":        cm.l2Stats.Hits,
			"misses":      cm.l2Stats.Misses,
			"sets":        cm.l2Stats.Sets,
			"deletes":     cm.l2Stats.Deletes,
			"errors":      cm.l2Stats.Errors,
			"hit_rate":    cm.calculateHitRate(cm.l2Stats.Hits, cm.l2Stats.Misses),
			"last_access": cm.l2Stats.LastAccess,
		}
	}

	return stats
}

// calculateHitRate calculates cache hit rate
func (cm *CacheManager) calculateHitRate(hits, misses int64) float64 {
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total)
}

// getRedisKey generates a Redis key for a patient
func (cm *CacheManager) getRedisKey(patientID string) string {
	return fmt.Sprintf("sgp:context:%s", patientID)
}

// HealthCheck performs a health check on the cache system
func (cm *CacheManager) HealthCheck() error {
	// Test L1 cache
	testKey := "health_check_test"
	testContext := &types.ClinicalContext{
		PatientID:      testKey,
		ContextVersion: "test",
		AssemblyTime:   time.Now(),
	}

	// Test L1 cache operations
	cm.Set(testKey, testContext, 1*time.Minute)
	retrieved := cm.Get(testKey)
	if retrieved == nil {
		return fmt.Errorf("L1 cache health check failed")
	}
	cm.Delete(testKey)

	// Test L2 cache if enabled
	if cm.l2Client != nil {
		ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 2*time.Second)
		defer cancel()

		if err := cm.l2Client.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("L2 cache (Redis) health check failed: %w", err)
		}
	}

	return nil
}

// Shutdown shuts down the cache manager
func (cm *CacheManager) Shutdown() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Close Redis connection
	if cm.l2Client != nil {
		if err := cm.l2Client.Close(); err != nil {
			cm.logger.Warn("Error closing Redis connection", zap.Error(err))
		}
	}

	// Clear L1 cache
	cm.l1Cache = &sync.Map{}

	cm.logger.Info("Cache manager shut down")
}
