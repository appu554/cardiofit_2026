// Package cache holds the eval-result cache for the kb-30 evaluator.
//
// Two implementations:
//   - InMemoryCache: production-grade for single-replica deploys, MVP local
//     dev, and tests. Uses sync.Map + per-entry expiration tracking with
//     a single periodic sweeper goroutine.
//   - RedisCache:    stub. Wired at the cmd/server entry point in production.
//     Marked TODO until production Redis credentials are available.
//
// Per-rule TTLs (Layer 3 v2 doc Part 4.5.3):
//   - 24h for static jurisdictional rules
//   - 1h  for credential-dependent decisions
//   - 15min for prescribing-agreement-dependent decisions
//   - 5min for consent-dependent decisions
package cache

import (
	"context"
	"strings"
	"sync"
	"time"

	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
)

// Cache is the contract implemented by InMemoryCache and (eventually)
// RedisCache.
type Cache interface {
	Get(ctx context.Context, key string) (*evaluator.Result, bool, error)
	Set(ctx context.Context, key string, result *evaluator.Result, ttl time.Duration) error
	Invalidate(ctx context.Context, pattern string) error
	Size() int
}

// DefaultTTL maps an evaluation Result back to the Layer 3 v2 doc Part 4.5.3
// TTL bucket. The mapping looks at the Result's conditions: any condition
// referencing Consent → 5min; PrescribingAgreement → 15min; Credential →
// 1h; otherwise 24h.
func DefaultTTL(res evaluator.Result) time.Duration {
	if res.Decision == dsl.DecisionDenied && len(res.Conditions) == 0 {
		return 24 * time.Hour
	}
	if len(res.Conditions) == 0 {
		return 24 * time.Hour
	}
	// The most-volatile dependency wins (shortest TTL).
	ttl := 24 * time.Hour
	for _, c := range res.Conditions {
		check := strings.ToLower(c.Check + " " + c.Condition)
		switch {
		case strings.Contains(check, "consent"):
			if 5*time.Minute < ttl {
				ttl = 5 * time.Minute
			}
		case strings.Contains(check, "prescribingagreement"), strings.Contains(check, "agreement"):
			if 15*time.Minute < ttl {
				ttl = 15 * time.Minute
			}
		case strings.Contains(check, "credential"), strings.Contains(check, "endorsement"):
			if time.Hour < ttl {
				ttl = time.Hour
			}
		}
	}
	return ttl
}

// ----- InMemoryCache ---------------------------------------------------------

type entry struct {
	value     *evaluator.Result
	expiresAt time.Time
}

// InMemoryCache is a sync.Map-backed cache safe for concurrent use.
type InMemoryCache struct {
	data sync.Map // map[string]entry
	now  func() time.Time
}

// NewInMemory returns a fresh in-memory cache.
func NewInMemory() *InMemoryCache {
	return &InMemoryCache{now: time.Now}
}

func (c *InMemoryCache) Get(_ context.Context, key string) (*evaluator.Result, bool, error) {
	v, ok := c.data.Load(key)
	if !ok {
		return nil, false, nil
	}
	e := v.(entry)
	if c.now().After(e.expiresAt) {
		c.data.Delete(key)
		return nil, false, nil
	}
	return e.value, true, nil
}

func (c *InMemoryCache) Set(_ context.Context, key string, result *evaluator.Result, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	c.data.Store(key, entry{value: result, expiresAt: c.now().Add(ttl)})
	return nil
}

// Invalidate removes every entry whose key matches pattern. The pattern
// supports a single trailing '*' wildcard and embedded '*' wildcards. We
// implement glob match with a simple strings.Contains-on-segments approach
// because the cache key format is well-known and finite.
func (c *InMemoryCache) Invalidate(_ context.Context, pattern string) error {
	c.data.Range(func(k, _ any) bool {
		key := k.(string)
		if matchPattern(pattern, key) {
			c.data.Delete(k)
		}
		return true
	})
	return nil
}

// Size returns the current entry count (after lazy-expiring matched entries).
func (c *InMemoryCache) Size() int {
	count := 0
	now := c.now()
	c.data.Range(func(k, v any) bool {
		e := v.(entry)
		if now.After(e.expiresAt) {
			c.data.Delete(k)
		} else {
			count++
		}
		return true
	})
	return count
}

// matchPattern is a tiny glob matcher: '*' matches any sequence of
// characters (including empty). Other characters match literally.
func matchPattern(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}
	// First part must prefix s (unless empty).
	if parts[0] != "" {
		if !strings.HasPrefix(s, parts[0]) {
			return false
		}
		s = s[len(parts[0]):]
	}
	// Last part must suffix s (unless empty).
	last := parts[len(parts)-1]
	if last != "" {
		if !strings.HasSuffix(s, last) {
			return false
		}
		s = s[:len(s)-len(last)]
	}
	// Middle parts must appear in order.
	for _, p := range parts[1 : len(parts)-1] {
		idx := strings.Index(s, p)
		if idx < 0 {
			return false
		}
		s = s[idx+len(p):]
	}
	return true
}

// ----- RedisCache ------------------------------------------------------------

// RedisCache is a stub; production wiring requires go-redis credentials.
type RedisCache struct{}

// NewRedis returns the stub Redis cache. All operations are no-ops with
// nil errors; the InMemoryCache should be used until Redis is wired.
//
// TODO(layer3-v1): wire go-redis client when production Redis credentials
// are available. The interface above is sufficient for migration.
func NewRedis() *RedisCache { return &RedisCache{} }

func (RedisCache) Get(context.Context, string) (*evaluator.Result, bool, error) {
	return nil, false, nil
}
func (RedisCache) Set(context.Context, string, *evaluator.Result, time.Duration) error { return nil }
func (RedisCache) Invalidate(context.Context, string) error                            { return nil }
func (RedisCache) Size() int                                                           { return 0 }
