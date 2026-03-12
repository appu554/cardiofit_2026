package cache

import "errors"

// Cache-related errors
var (
	ErrKeyNotFound    = errors.New("key not found in cache")
	ErrCacheDisabled  = errors.New("cache is disabled")
	ErrCacheFull      = errors.New("cache is full")
	ErrInvalidKey     = errors.New("invalid cache key")
	ErrSerializationFailed = errors.New("failed to serialize value")
	ErrDeserializationFailed = errors.New("failed to deserialize value")
	ErrCacheUnavailable = errors.New("cache service unavailable")
	ErrTTLExpired     = errors.New("cache entry has expired")
	ErrInvalidationFailed = errors.New("cache invalidation failed")
)