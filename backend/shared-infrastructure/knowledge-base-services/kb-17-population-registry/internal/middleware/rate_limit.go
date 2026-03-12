// Package middleware provides HTTP middleware for KB-17 Population Registry
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled         bool
	RequestsPerMin  int
	BurstSize       int
	CleanupInterval time.Duration
	SkipPaths       []string
	UseRedis        bool
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled:         true,
		RequestsPerMin:  100, // 100 requests per minute per client
		BurstSize:       20,  // Allow burst of 20 requests
		CleanupInterval: 5 * time.Minute,
		SkipPaths:       []string{"/health", "/ready"},
		UseRedis:        false,
	}
}

// RateLimiter interface for different implementations
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, int, time.Duration)
}

// InMemoryRateLimiter implements in-memory token bucket rate limiting
type InMemoryRateLimiter struct {
	config  *RateLimitConfig
	buckets map[string]*tokenBucket
	mu      sync.RWMutex
	stopCh  chan struct{}
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter
func NewInMemoryRateLimiter(config *RateLimitConfig) *InMemoryRateLimiter {
	// Ensure CleanupInterval is positive to avoid ticker panic
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 5 * time.Minute // Use default
	}

	limiter := &InMemoryRateLimiter{
		config:  config,
		buckets: make(map[string]*tokenBucket),
		stopCh:  make(chan struct{}),
	}

	// Start cleanup goroutine
	go limiter.cleanupLoop()

	return limiter
}

func (l *InMemoryRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.stopCh:
			return
		}
	}
}

func (l *InMemoryRateLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	threshold := time.Now().Add(-l.config.CleanupInterval)
	for key, bucket := range l.buckets {
		if bucket.lastRefill.Before(threshold) {
			delete(l.buckets, key)
		}
	}
}

// Stop stops the rate limiter cleanup loop
func (l *InMemoryRateLimiter) Stop() {
	close(l.stopCh)
}

// Allow checks if a request should be allowed
func (l *InMemoryRateLimiter) Allow(ctx context.Context, key string) (bool, int, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	refillRate := float64(l.config.RequestsPerMin) / 60.0 // tokens per second

	bucket, exists := l.buckets[key]
	if !exists {
		bucket = &tokenBucket{
			tokens:     float64(l.config.BurstSize),
			lastRefill: now,
		}
		l.buckets[key] = bucket
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens = min(float64(l.config.BurstSize), bucket.tokens+(elapsed*refillRate))
	bucket.lastRefill = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		remaining := int(bucket.tokens)
		return true, remaining, 0
	}

	// Calculate retry after
	retryAfter := time.Duration((1 - bucket.tokens) / refillRate * float64(time.Second))
	return false, 0, retryAfter
}

// RedisRateLimiter implements Redis-based rate limiting using sliding window
type RedisRateLimiter struct {
	client *redis.Client
	config *RateLimitConfig
	logger *logrus.Entry
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(client *redis.Client, config *RateLimitConfig, logger *logrus.Entry) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		config: config,
		logger: logger,
	}
}

// Allow checks if a request should be allowed using Redis
func (l *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, int, time.Duration) {
	now := time.Now()
	windowStart := now.Add(-time.Minute)

	redisKey := fmt.Sprintf("ratelimit:%s", key)

	// Use Redis sorted set for sliding window
	pipe := l.client.Pipeline()

	// Remove old entries
	pipe.ZRemRangeByScore(ctx, redisKey, "0", strconv.FormatInt(windowStart.UnixMilli(), 10))

	// Count current entries
	countCmd := pipe.ZCard(ctx, redisKey)

	// Add current request
	pipe.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: now.UnixNano(),
	})

	// Set expiry
	pipe.Expire(ctx, redisKey, 2*time.Minute)

	_, err := pipe.Exec(ctx)
	if err != nil {
		l.logger.WithError(err).Warn("Redis rate limit check failed, allowing request")
		return true, l.config.RequestsPerMin, 0
	}

	count := countCmd.Val()
	remaining := l.config.RequestsPerMin - int(count)

	if int(count) < l.config.RequestsPerMin {
		return true, remaining, 0
	}

	// Calculate retry after
	retryAfter := time.Minute / time.Duration(l.config.RequestsPerMin)
	return false, 0, retryAfter
}

// RateLimitMiddleware creates rate limiting middleware
func RateLimitMiddleware(limiter RateLimiter, config *RateLimitConfig, logger *logrus.Entry) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if disabled
		if !config.Enabled {
			c.Next()
			return
		}

		// Skip for excluded paths
		path := c.Request.URL.Path
		for _, skipPath := range config.SkipPaths {
			if path == skipPath {
				c.Next()
				return
			}
		}

		// Determine rate limit key (IP-based by default)
		key := c.ClientIP()

		// Use authenticated user/service if available
		if serviceName := c.GetString("service_name"); serviceName != "" {
			key = fmt.Sprintf("service:%s", serviceName)
		} else if userID := c.GetString("user_id"); userID != "" {
			key = fmt.Sprintf("user:%s", userID)
		}

		allowed, remaining, retryAfter := limiter.Allow(c.Request.Context(), key)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerMin))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if !allowed {
			logger.WithFields(logrus.Fields{
				"key":         key,
				"path":        path,
				"retry_after": retryAfter.Seconds(),
			}).Warn("Rate limit exceeded")

			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too Many Requests",
				"message":     "Rate limit exceeded",
				"retry_after": retryAfter.Seconds(),
			})
			return
		}

		c.Next()
	}
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// AdaptiveRateLimitConfig holds adaptive rate limiting configuration
type AdaptiveRateLimitConfig struct {
	BaseRequestsPerMin int
	MaxRequestsPerMin  int
	MinRequestsPerMin  int
	AdjustmentFactor   float64
}

// AdaptiveRateLimiter adjusts rate limits based on system load
type AdaptiveRateLimiter struct {
	*InMemoryRateLimiter
	adaptiveConfig *AdaptiveRateLimitConfig
	currentLimit   int
	mu             sync.RWMutex
}

// NewAdaptiveRateLimiter creates an adaptive rate limiter
func NewAdaptiveRateLimiter(config *RateLimitConfig, adaptiveConfig *AdaptiveRateLimitConfig) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		InMemoryRateLimiter: NewInMemoryRateLimiter(config),
		adaptiveConfig:      adaptiveConfig,
		currentLimit:        adaptiveConfig.BaseRequestsPerMin,
	}
}

// AdjustLimit adjusts the rate limit based on system load
func (l *AdaptiveRateLimiter) AdjustLimit(loadFactor float64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if loadFactor > 0.8 {
		// High load - reduce limits
		newLimit := int(float64(l.currentLimit) * (1 - l.adaptiveConfig.AdjustmentFactor))
		l.currentLimit = max(newLimit, l.adaptiveConfig.MinRequestsPerMin)
	} else if loadFactor < 0.5 {
		// Low load - increase limits
		newLimit := int(float64(l.currentLimit) * (1 + l.adaptiveConfig.AdjustmentFactor))
		l.currentLimit = min64(newLimit, l.adaptiveConfig.MaxRequestsPerMin)
	}

	l.config.RequestsPerMin = l.currentLimit
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min64(a, b int) int {
	if a < b {
		return a
	}
	return b
}
