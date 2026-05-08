package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
)

// testRedis returns a connection to a local Redis for integration tests.
// Tests skip if KB30_TEST_REDIS_ADDR is unset, preserving CI without Redis.
func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("KB30_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("KB30_TEST_REDIS_ADDR unset; skipping Redis integration test")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("ping redis at %s: %v", addr, err)
	}
	return rdb
}

// uniqueKey returns a key prefixed with this test's identity so tests
// don't pollute each other when run in parallel.
func uniqueKey(prefix string) string {
	return prefix + ":" + uuid.New().String()
}

func newResult() *evaluator.Result {
	return &evaluator.Result{
		Decision:    dsl.DecisionGranted,
		Reason:      "test",
		EvaluatedAt: time.Now().UTC(),
	}
}

func TestRedisCache_SetGetRoundTrip(t *testing.T) {
	rdb := testRedis(t)
	defer rdb.Close()
	c := NewRedisFromClient(rdb)
	ctx := context.Background()

	key := uniqueKey("rt")
	r := newResult()
	if err := c.Set(ctx, key, r, 30*time.Second); err != nil {
		t.Fatalf("set: %v", err)
	}
	t.Cleanup(func() { rdb.Del(context.Background(), key) })

	got, hit, err := c.Get(ctx, key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !hit {
		t.Errorf("expected hit; got miss")
	}
	if got == nil || got.Decision != r.Decision || got.Reason != r.Reason {
		t.Errorf("round-trip lost data: got %+v want %+v", got, r)
	}
}

func TestRedisCache_GetMiss(t *testing.T) {
	rdb := testRedis(t)
	defer rdb.Close()
	c := NewRedisFromClient(rdb)
	got, hit, err := c.Get(context.Background(), uniqueKey("miss"))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if hit || got != nil {
		t.Errorf("expected miss; got hit=%v val=%+v", hit, got)
	}
}

func TestRedisCache_TTLExpiry(t *testing.T) {
	rdb := testRedis(t)
	defer rdb.Close()
	c := NewRedisFromClient(rdb)
	ctx := context.Background()

	key := uniqueKey("ttl")
	if err := c.Set(ctx, key, newResult(), 500*time.Millisecond); err != nil {
		t.Fatalf("set: %v", err)
	}
	t.Cleanup(func() { rdb.Del(context.Background(), key) })

	// Hit immediately
	_, hit, _ := c.Get(ctx, key)
	if !hit {
		t.Errorf("expected hit before TTL")
	}

	time.Sleep(700 * time.Millisecond)

	// Miss after TTL
	_, hit, _ = c.Get(ctx, key)
	if hit {
		t.Errorf("expected miss after TTL expiry")
	}
}

func TestRedisCache_InvalidateGlob(t *testing.T) {
	rdb := testRedis(t)
	defer rdb.Close()
	c := NewRedisFromClient(rdb)
	ctx := context.Background()

	prefix := "globtest:" + uuid.New().String()
	keys := []string{
		prefix + ":a",
		prefix + ":b",
		prefix + ":c",
	}
	for _, k := range keys {
		if err := c.Set(ctx, k, newResult(), time.Minute); err != nil {
			t.Fatalf("set %s: %v", k, err)
		}
	}
	t.Cleanup(func() {
		for _, k := range keys {
			rdb.Del(context.Background(), k)
		}
	})

	if err := c.Invalidate(ctx, prefix+":*"); err != nil {
		t.Fatalf("invalidate: %v", err)
	}

	for _, k := range keys {
		_, hit, _ := c.Get(ctx, k)
		if hit {
			t.Errorf("key %s should be invalidated; still in cache", k)
		}
	}
}

func TestRedisCache_InvalidateExactKey(t *testing.T) {
	rdb := testRedis(t)
	defer rdb.Close()
	c := NewRedisFromClient(rdb)
	ctx := context.Background()

	key := uniqueKey("exact")
	if err := c.Set(ctx, key, newResult(), time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	t.Cleanup(func() { rdb.Del(context.Background(), key) })

	if err := c.Invalidate(ctx, key); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	_, hit, _ := c.Get(ctx, key)
	if hit {
		t.Errorf("expected miss after exact-key invalidate")
	}
}

func TestRedisCache_Size(t *testing.T) {
	rdb := testRedis(t)
	defer rdb.Close()
	c := NewRedisFromClient(rdb)
	ctx := context.Background()

	prefix := "sizetest:" + uuid.New().String()
	for i := 0; i < 3; i++ {
		_ = c.Set(ctx, prefix+":"+uuid.New().String(), newResult(), time.Minute)
	}
	t.Cleanup(func() { _ = c.Invalidate(context.Background(), prefix+":*") })

	// Size returns total Redis DB size or -1 if unsupported. Just verify it
	// doesn't panic and returns a non-negative integer.
	if got := c.Size(); got < 0 {
		t.Errorf("Size returned negative: %d", got)
	}
}
