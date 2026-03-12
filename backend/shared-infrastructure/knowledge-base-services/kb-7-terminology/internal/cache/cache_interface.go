package cache

import (
	"time"
)

// Cache interface defines the contract for caching operations
type Cache interface {
	Get(key string) (interface{}, error)
	GetWithLoader(key string, loader func() (interface{}, error)) (interface{}, error)
	Set(key string, value interface{}, ttl time.Duration) error
	Delete(key string) error
	Exists(key string) (bool, error)
	Close() error
}

// EnhancedCache extends the basic cache interface with advanced features
type EnhancedCache interface {
	Cache
	
	// Multi-layer operations
	SetAllLayers(key string, value interface{}) error
	
	// Invalidation
	Invalidate(pattern InvalidationPattern) error
	DeletePattern(pattern string) error
	
	// Cache warming
	WarmCache(data map[string]interface{}) error
	
	// Statistics
	GetStatistics() CacheStatistics
	
	// Health check
	HealthCheck() (bool, error)
}

// CacheManager provides unified access to different cache implementations
type CacheManager struct {
	primary   EnhancedCache
	fallback  Cache
	enabled   bool
}

// NewCacheManager creates a new cache manager
func NewCacheManager(primary EnhancedCache, fallback Cache) *CacheManager {
	return &CacheManager{
		primary:  primary,
		fallback: fallback,
		enabled:  true,
	}
}

// Get retrieves a value from cache with fallback support
func (cm *CacheManager) Get(key string) (interface{}, error) {
	if !cm.enabled {
		return nil, ErrCacheDisabled
	}
	
	// Try primary cache first
	if cm.primary != nil {
		if value, err := cm.primary.Get(key); err == nil {
			return value, nil
		}
	}
	
	// Fall back to secondary cache
	if cm.fallback != nil {
		return cm.fallback.Get(key)
	}
	
	return nil, ErrKeyNotFound
}

// GetWithLoader retrieves or loads a value
func (cm *CacheManager) GetWithLoader(key string, loader func() (interface{}, error)) (interface{}, error) {
	if !cm.enabled {
		return loader()
	}
	
	// Try primary cache with loader
	if cm.primary != nil {
		return cm.primary.GetWithLoader(key, loader)
	}
	
	// Fall back to basic get/load pattern
	if cm.fallback != nil {
		if value, err := cm.fallback.Get(key); err == nil {
			return value, nil
		}
	}
	
	// Load and cache the value
	value, err := loader()
	if err != nil {
		return nil, err
	}
	
	if cm.fallback != nil {
		cm.fallback.Set(key, value, time.Hour) // Default TTL
	}
	
	return value, nil
}

// Set stores a value in cache
func (cm *CacheManager) Set(key string, value interface{}, ttl time.Duration) error {
	if !cm.enabled {
		return nil
	}
	
	var errs []error
	
	// Store in primary cache
	if cm.primary != nil {
		if err := cm.primary.Set(key, value, ttl); err != nil {
			errs = append(errs, err)
		}
	}
	
	// Store in fallback cache
	if cm.fallback != nil {
		if err := cm.fallback.Set(key, value, ttl); err != nil {
			errs = append(errs, err)
		}
	}
	
	if len(errs) > 0 {
		return errs[0] // Return first error
	}
	
	return nil
}

// SetAllLayers stores in all available cache layers
func (cm *CacheManager) SetAllLayers(key string, value interface{}) error {
	if !cm.enabled {
		return nil
	}
	
	if cm.primary != nil {
		return cm.primary.SetAllLayers(key, value)
	}
	
	if cm.fallback != nil {
		return cm.fallback.Set(key, value, time.Hour)
	}
	
	return nil
}

// Delete removes a key from cache
func (cm *CacheManager) Delete(key string) error {
	if !cm.enabled {
		return nil
	}
	
	// Delete from both caches
	if cm.primary != nil {
		cm.primary.Delete(key)
	}
	
	if cm.fallback != nil {
		cm.fallback.Delete(key)
	}
	
	return nil
}

// Invalidate removes keys based on pattern
func (cm *CacheManager) Invalidate(pattern InvalidationPattern) error {
	if !cm.enabled {
		return nil
	}
	
	if cm.primary != nil {
		return cm.primary.Invalidate(pattern)
	}
	
	return nil
}

// WarmCache preloads frequently accessed data
func (cm *CacheManager) WarmCache(data map[string]interface{}) error {
	if !cm.enabled {
		return nil
	}
	
	if cm.primary != nil {
		return cm.primary.WarmCache(data)
	}
	
	// Fallback warming
	if cm.fallback != nil {
		for key, value := range data {
			cm.fallback.Set(key, value, 24*time.Hour)
		}
	}
	
	return nil
}

// GetStatistics returns cache performance metrics
func (cm *CacheManager) GetStatistics() CacheStatistics {
	if cm.primary != nil {
		return cm.primary.GetStatistics()
	}
	
	return CacheStatistics{}
}

// HealthCheck verifies cache connectivity
func (cm *CacheManager) HealthCheck() (bool, error) {
	if !cm.enabled {
		return false, ErrCacheDisabled
	}
	
	if cm.primary != nil {
		return cm.primary.HealthCheck()
	}
	
	if cm.fallback != nil {
		// Basic health check for fallback cache
		testKey := "health_check_" + time.Now().Format("20060102150405")
		if err := cm.fallback.Set(testKey, "ok", time.Minute); err != nil {
			return false, err
		}
		cm.fallback.Delete(testKey)
	}
	
	return true, nil
}

// Exists checks if a key exists in cache
func (cm *CacheManager) Exists(key string) (bool, error) {
	if !cm.enabled {
		return false, nil
	}
	
	if cm.primary != nil {
		if exists, err := cm.primary.Exists(key); err == nil && exists {
			return true, nil
		}
	}
	
	if cm.fallback != nil {
		return cm.fallback.Exists(key)
	}
	
	return false, nil
}

// Close shuts down the cache manager
func (cm *CacheManager) Close() error {
	if cm.primary != nil {
		cm.primary.Close()
	}
	
	if cm.fallback != nil {
		cm.fallback.Close()
	}
	
	return nil
}

// Enable/disable cache operations
func (cm *CacheManager) SetEnabled(enabled bool) {
	cm.enabled = enabled
}

func (cm *CacheManager) IsEnabled() bool {
	return cm.enabled
}