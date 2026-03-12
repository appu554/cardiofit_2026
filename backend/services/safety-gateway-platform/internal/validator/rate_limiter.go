package validator

import (
	"fmt"
	"sync"
	"time"

	"safety-gateway-platform/internal/config"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	enabled           bool
	requestsPerMinute int
	burstSize         int
	buckets           map[string]*TokenBucket
	mutex             sync.RWMutex
	cleanupTicker     *time.Ticker
	stopCleanup       chan struct{}
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens       int
	maxTokens    int
	refillRate   time.Duration
	lastRefill   time.Time
	mutex        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cfg config.RateLimitingConfig) (*RateLimiter, error) {
	if !cfg.Enabled {
		return &RateLimiter{enabled: false}, nil
	}

	if cfg.RequestsPerMinute <= 0 {
		return nil, fmt.Errorf("requests per minute must be positive")
	}

	if cfg.BurstSize <= 0 {
		return nil, fmt.Errorf("burst size must be positive")
	}

	rl := &RateLimiter{
		enabled:           true,
		requestsPerMinute: cfg.RequestsPerMinute,
		burstSize:         cfg.BurstSize,
		buckets:           make(map[string]*TokenBucket),
		stopCleanup:       make(chan struct{}),
	}

	// Start cleanup goroutine to remove old buckets
	rl.cleanupTicker = time.NewTicker(5 * time.Minute)
	go rl.cleanupRoutine()

	return rl, nil
}

// Allow checks if a request is allowed for the given client ID
func (rl *RateLimiter) Allow(clientID string) error {
	if !rl.enabled {
		return nil
	}

	rl.mutex.Lock()
	bucket, exists := rl.buckets[clientID]
	if !exists {
		bucket = rl.createBucket()
		rl.buckets[clientID] = bucket
	}
	rl.mutex.Unlock()

	if !bucket.consume() {
		return fmt.Errorf("rate limit exceeded for client %s", clientID)
	}

	return nil
}

// createBucket creates a new token bucket
func (rl *RateLimiter) createBucket() *TokenBucket {
	refillRate := time.Minute / time.Duration(rl.requestsPerMinute)
	
	return &TokenBucket{
		tokens:     rl.burstSize,
		maxTokens:  rl.burstSize,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// consume attempts to consume a token from the bucket
func (tb *TokenBucket) consume() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int(elapsed / tb.refillRate)

	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.maxTokens {
			tb.tokens = tb.maxTokens
		}
		tb.lastRefill = now
	}

	// Try to consume a token
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// cleanupRoutine removes old, unused buckets
func (rl *RateLimiter) cleanupRoutine() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.cleanup()
		case <-rl.stopCleanup:
			return
		}
	}
}

// cleanup removes buckets that haven't been used recently
func (rl *RateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	
	for clientID, bucket := range rl.buckets {
		bucket.mutex.Lock()
		if bucket.lastRefill.Before(cutoff) {
			delete(rl.buckets, clientID)
		}
		bucket.mutex.Unlock()
	}
}

// Stop stops the rate limiter and cleanup routine
func (rl *RateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
	}
	
	select {
	case rl.stopCleanup <- struct{}{}:
	default:
	}
}

// GetStats returns rate limiting statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	if !rl.enabled {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	return map[string]interface{}{
		"enabled":             true,
		"requests_per_minute": rl.requestsPerMinute,
		"burst_size":          rl.burstSize,
		"active_clients":      len(rl.buckets),
	}
}
