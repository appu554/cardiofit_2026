package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	// Per-second limits
	RequestsPerSecond int `json:"requests_per_second"`
	BurstSize         int `json:"burst_size"`
	
	// Per-minute limits
	RequestsPerMinute int `json:"requests_per_minute"`
	
	// Per-hour limits
	RequestsPerHour int `json:"requests_per_hour"`
	
	// Per-day limits
	RequestsPerDay int `json:"requests_per_day"`
	
	// Window duration for sliding window
	WindowDuration time.Duration `json:"window_duration"`
	
	// Block duration when limit is exceeded
	BlockDuration time.Duration `json:"block_duration"`
}

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed       bool          `json:"allowed"`
	Remaining     int           `json:"remaining"`
	ResetTime     time.Time     `json:"reset_time"`
	RetryAfter    time.Duration `json:"retry_after,omitempty"`
	LimitType     string        `json:"limit_type,omitempty"`
	WindowSize    time.Duration `json:"window_size"`
	RequestsUsed  int           `json:"requests_used"`
}

// TokenBucket implements the token bucket algorithm
type TokenBucket struct {
	capacity     int
	tokens       int
	refillRate   int // tokens per second
	lastRefill   time.Time
	mu           sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity, refillRate int) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	
	// Refill tokens based on elapsed time
	tokensToAdd := int(elapsed * float64(tb.refillRate))
	tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
	tb.lastRefill = now

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// GetStatus returns the current status of the bucket
func (tb *TokenBucket) GetStatus() (int, int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.tokens, tb.capacity
}

// RateLimiter manages rate limiting for the terminology service
type RateLimiter struct {
	redis         *redis.Client
	logger        *zap.Logger
	buckets       sync.Map // map[string]*TokenBucket
	configs       sync.Map // map[string]*RateLimitConfig
	defaultConfig *RateLimitConfig
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client, logger *zap.Logger) *RateLimiter {
	defaultConfig := &RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         200,
		RequestsPerMinute: 3000,
		RequestsPerHour:   50000,
		RequestsPerDay:    500000,
		WindowDuration:    time.Minute,
		BlockDuration:     time.Minute * 5,
	}

	rl := &RateLimiter{
		redis:         redisClient,
		logger:        logger,
		defaultConfig: defaultConfig,
	}

	// Initialize default configurations for different operations
	rl.initializeDefaultConfigs()

	return rl
}

// initializeDefaultConfigs sets up default rate limiting configurations
func (rl *RateLimiter) initializeDefaultConfigs() {
	configs := map[string]*RateLimitConfig{
		"lookup": {
			RequestsPerSecond: 100,
			BurstSize:         200,
			RequestsPerMinute: 5000,
			RequestsPerHour:   100000,
			RequestsPerDay:    1000000,
			WindowDuration:    time.Minute,
			BlockDuration:     time.Minute * 2,
		},
		"search": {
			RequestsPerSecond: 50,
			BurstSize:         100,
			RequestsPerMinute: 2000,
			RequestsPerHour:   50000,
			RequestsPerDay:    500000,
			WindowDuration:    time.Minute,
			BlockDuration:     time.Minute * 5,
		},
		"expand": {
			RequestsPerSecond: 20,
			BurstSize:         40,
			RequestsPerMinute: 1000,
			RequestsPerHour:   20000,
			RequestsPerDay:    200000,
			WindowDuration:    time.Minute,
			BlockDuration:     time.Minute * 10,
		},
		"validate": {
			RequestsPerSecond: 200,
			BurstSize:         400,
			RequestsPerMinute: 10000,
			RequestsPerHour:   200000,
			RequestsPerDay:    2000000,
			WindowDuration:    time.Minute,
			BlockDuration:     time.Minute,
		},
		"batch": {
			RequestsPerSecond: 10,
			BurstSize:         20,
			RequestsPerMinute: 500,
			RequestsPerHour:   5000,
			RequestsPerDay:    50000,
			WindowDuration:    time.Minute,
			BlockDuration:     time.Minute * 15,
		},
	}

	for operation, config := range configs {
		rl.configs.Store(operation, config)
	}
}

// CheckLimit checks if a request should be allowed based on rate limits
func (rl *RateLimiter) CheckLimit(ctx context.Context, key, operation string) (*RateLimitResult, error) {
	config := rl.getConfig(operation)
	
	// Check if user is currently blocked
	if blocked, blockExpiry := rl.isBlocked(ctx, key, operation); blocked {
		return &RateLimitResult{
			Allowed:    false,
			RetryAfter: time.Until(blockExpiry),
			LimitType:  "block",
		}, nil
	}

	// Check token bucket (burst protection)
	bucketKey := fmt.Sprintf("bucket:%s:%s", key, operation)
	bucket := rl.getOrCreateBucket(bucketKey, config)
	
	if !bucket.Allow() {
		tokens, capacity := bucket.GetStatus()
		resetTime := time.Now().Add(time.Second)
		
		return &RateLimitResult{
			Allowed:      false,
			Remaining:    tokens,
			ResetTime:    resetTime,
			LimitType:    "burst",
			RequestsUsed: capacity - tokens,
		}, nil
	}

	// Check sliding window limits
	windowResult, err := rl.checkSlidingWindow(ctx, key, operation, config)
	if err != nil {
		rl.logger.Error("Failed to check sliding window", zap.Error(err))
		// Allow request if Redis is down (fail open)
		return &RateLimitResult{Allowed: true}, nil
	}

	if !windowResult.Allowed {
		// Block user if they've exceeded limits multiple times
		rl.recordViolation(ctx, key, operation)
		return windowResult, nil
	}

	// Record successful request
	rl.recordRequest(ctx, key, operation)

	return windowResult, nil
}

// checkSlidingWindow checks sliding window rate limits using Redis
func (rl *RateLimiter) checkSlidingWindow(ctx context.Context, key, operation string, config *RateLimitConfig) (*RateLimitResult, error) {
	now := time.Now()
	
	// Check different time windows
	windows := []struct {
		name     string
		duration time.Duration
		limit    int
	}{
		{"minute", time.Minute, config.RequestsPerMinute},
		{"hour", time.Hour, config.RequestsPerHour},
		{"day", 24 * time.Hour, config.RequestsPerDay},
	}

	for _, window := range windows {
		windowKey := fmt.Sprintf("rl:%s:%s:%s", key, operation, window.name)
		
		// Use Redis pipeline for efficiency
		pipe := rl.redis.Pipeline()
		
		// Count requests in current window
		pipe.ZCount(ctx, windowKey, fmt.Sprintf("(%d", now.Add(-window.duration).UnixNano()), "+inf")
		
		// Add current request
		pipe.ZAdd(ctx, windowKey, &redis.Z{
			Score:  float64(now.UnixNano()),
			Member: fmt.Sprintf("%d-%d", now.UnixNano(), time.Now().Nanosecond()),
		})
		
		// Set expiry
		pipe.Expire(ctx, windowKey, window.duration)
		
		// Remove old entries
		pipe.ZRemRangeByScore(ctx, windowKey, "-inf", fmt.Sprintf("(%d", now.Add(-window.duration).UnixNano()))
		
		results, err := pipe.Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to execute Redis pipeline: %w", err)
		}

		count := results[0].(*redis.IntCmd).Val()
		
		if int(count) >= window.limit {
			resetTime := now.Add(window.duration)
			
			return &RateLimitResult{
				Allowed:      false,
				Remaining:    window.limit - int(count),
				ResetTime:    resetTime,
				RetryAfter:   time.Until(resetTime),
				LimitType:    window.name,
				WindowSize:   window.duration,
				RequestsUsed: int(count),
			}, nil
		}
	}

	// All checks passed
	return &RateLimitResult{
		Allowed:   true,
		Remaining: config.RequestsPerMinute - 1, // Approximate remaining for minute window
		ResetTime: now.Add(time.Minute),
	}, nil
}

// isBlocked checks if a key is currently blocked
func (rl *RateLimiter) isBlocked(ctx context.Context, key, operation string) (bool, time.Time) {
	blockKey := fmt.Sprintf("block:%s:%s", key, operation)
	
	result := rl.redis.Get(ctx, blockKey)
	if result.Err() == redis.Nil {
		return false, time.Time{}
	}
	
	if result.Err() != nil {
		rl.logger.Error("Failed to check block status", zap.Error(result.Err()))
		return false, time.Time{}
	}

	// Parse block expiry time
	expiryStr := result.Val()
	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		rl.logger.Error("Failed to parse block expiry", zap.Error(err))
		return false, time.Time{}
	}

	if time.Now().Before(expiry) {
		return true, expiry
	}

	// Block has expired, clean it up
	rl.redis.Del(ctx, blockKey)
	return false, time.Time{}
}

// recordViolation records a rate limit violation and potentially blocks the user
func (rl *RateLimiter) recordViolation(ctx context.Context, key, operation string) {
	violationKey := fmt.Sprintf("violations:%s:%s", key, operation)
	config := rl.getConfig(operation)

	// Increment violation counter
	pipe := rl.redis.Pipeline()
	pipe.Incr(ctx, violationKey)
	pipe.Expire(ctx, violationKey, time.Hour) // Reset violation count after 1 hour

	results, err := pipe.Exec(ctx)
	if err != nil {
		rl.logger.Error("Failed to record violation", zap.Error(err))
		return
	}

	violations := results[0].(*redis.IntCmd).Val()

	// Block user if they have too many violations
	if violations >= 5 {
		rl.blockUser(ctx, key, operation, config.BlockDuration)
		
		// Reset violation counter
		rl.redis.Del(ctx, violationKey)
		
		rl.logger.Warn("User blocked due to rate limit violations",
			zap.String("key", key),
			zap.String("operation", operation),
			zap.Int64("violations", violations),
			zap.Duration("block_duration", config.BlockDuration))
	}
}

// blockUser blocks a user for a specified duration
func (rl *RateLimiter) blockUser(ctx context.Context, key, operation string, duration time.Duration) {
	blockKey := fmt.Sprintf("block:%s:%s", key, operation)
	expiry := time.Now().Add(duration)
	
	rl.redis.Set(ctx, blockKey, expiry.Format(time.RFC3339), duration)
}

// recordRequest records a successful request for metrics
func (rl *RateLimiter) recordRequest(ctx context.Context, key, operation string) {
	metricsKey := fmt.Sprintf("metrics:%s:%s", key, operation)
	
	pipe := rl.redis.Pipeline()
	pipe.Incr(ctx, metricsKey)
	pipe.Expire(ctx, metricsKey, 24*time.Hour)
	pipe.Exec(ctx)
}

// getConfig gets rate limit configuration for an operation
func (rl *RateLimiter) getConfig(operation string) *RateLimitConfig {
	if config, ok := rl.configs.Load(operation); ok {
		return config.(*RateLimitConfig)
	}
	return rl.defaultConfig
}

// getOrCreateBucket gets or creates a token bucket for a key
func (rl *RateLimiter) getOrCreateBucket(key string, config *RateLimitConfig) *TokenBucket {
	if bucket, ok := rl.buckets.Load(key); ok {
		return bucket.(*TokenBucket)
	}

	bucket := NewTokenBucket(config.BurstSize, config.RequestsPerSecond)
	rl.buckets.Store(key, bucket)
	
	// Clean up old buckets periodically
	go rl.cleanupBucket(key, time.Minute*10)
	
	return bucket
}

// cleanupBucket removes a bucket after a delay
func (rl *RateLimiter) cleanupBucket(key string, delay time.Duration) {
	time.Sleep(delay)
	rl.buckets.Delete(key)
}

// GetStats returns rate limiting statistics for a key
func (rl *RateLimiter) GetStats(ctx context.Context, key string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Get metrics for all operations
	operations := []string{"lookup", "search", "expand", "validate", "batch"}
	
	for _, operation := range operations {
		metricsKey := fmt.Sprintf("metrics:%s:%s", key, operation)
		
		count := rl.redis.Get(ctx, metricsKey)
		if count.Err() == nil {
			stats[operation] = count.Val()
		} else {
			stats[operation] = 0
		}
		
		// Check if blocked
		blocked, expiry := rl.isBlocked(ctx, key, operation)
		if blocked {
			stats[fmt.Sprintf("%s_blocked_until", operation)] = expiry
		}
	}

	return stats, nil
}

// UpdateConfig updates rate limiting configuration for an operation
func (rl *RateLimiter) UpdateConfig(operation string, config *RateLimitConfig) {
	rl.configs.Store(operation, config)
	rl.logger.Info("Updated rate limit configuration",
		zap.String("operation", operation),
		zap.Int("requests_per_second", config.RequestsPerSecond),
		zap.Int("burst_size", config.BurstSize))
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}