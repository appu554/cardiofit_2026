// Package unit provides unit tests for KB-9 Care Gaps Service components.
package unit

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-9-care-gaps/internal/cache"
)

// TestCacheInitialization tests cache initialization without Redis.
func TestCacheInitialization(t *testing.T) {
	logger := zap.NewNop()

	// Test with cache disabled
	t.Run("disabled_cache", func(t *testing.T) {
		cfg := cache.Config{
			RedisURL: "",
			TTL:      5 * time.Minute,
			Enabled:  false,
			Prefix:   "test:",
		}

		c, err := cache.NewCache(cfg, logger)
		if err != nil {
			t.Fatalf("Failed to create disabled cache: %v", err)
		}
		defer c.Close()

		stats := c.GetStats()
		if stats.Enabled {
			t.Error("Expected cache to be disabled")
		}
	})

	// Test with invalid Redis URL (should gracefully fallback)
	t.Run("invalid_redis_url", func(t *testing.T) {
		cfg := cache.Config{
			RedisURL: "redis://invalid-host:9999",
			TTL:      5 * time.Minute,
			Enabled:  true,
			Prefix:   "test:",
		}

		c, err := cache.NewCache(cfg, logger)
		if err != nil {
			t.Fatalf("Cache should not fail on invalid Redis: %v", err)
		}
		defer c.Close()

		// Should fall back to in-memory only
		stats := c.GetStats()
		if stats.Enabled {
			t.Log("Cache connected to Redis (unexpected in CI)")
		}
	})
}

// TestCQLLibraryCaching tests CQL library cache operations.
func TestCQLLibraryCaching(t *testing.T) {
	logger := zap.NewNop()

	cfg := cache.Config{
		RedisURL: "",
		TTL:      5 * time.Minute,
		Enabled:  false,
		Prefix:   "test:",
	}

	c, err := cache.NewCache(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Test cache miss
	t.Run("cache_miss", func(t *testing.T) {
		lib, err := c.GetCQLLibrary(ctx, "CMS122", "1.0.0")
		if err != nil {
			t.Fatalf("GetCQLLibrary failed: %v", err)
		}
		if lib != nil {
			t.Error("Expected nil on cache miss")
		}
	})

	// Test cache set and get
	t.Run("cache_hit", func(t *testing.T) {
		lib := &cache.CachedLibrary{
			LibraryID:   "CMS122-DiabetesHbA1c",
			Version:     "11.0.0",
			Content:     "library CMS122 version '11.0.0'",
			CompiledELM: `{"library": {"identifier": {"id": "CMS122"}}}`,
		}

		err := c.SetCQLLibrary(ctx, lib)
		if err != nil {
			t.Fatalf("SetCQLLibrary failed: %v", err)
		}

		retrieved, err := c.GetCQLLibrary(ctx, "CMS122-DiabetesHbA1c", "11.0.0")
		if err != nil {
			t.Fatalf("GetCQLLibrary failed: %v", err)
		}

		if retrieved == nil {
			t.Fatal("Expected cached library, got nil")
		}

		if retrieved.LibraryID != lib.LibraryID {
			t.Errorf("Library ID mismatch: got %s, want %s", retrieved.LibraryID, lib.LibraryID)
		}

		if retrieved.Content != lib.Content {
			t.Error("Library content mismatch")
		}
	})
}

// TestMeasureCaching tests measure definition cache operations.
func TestMeasureCaching(t *testing.T) {
	logger := zap.NewNop()

	cfg := cache.Config{
		RedisURL: "",
		TTL:      5 * time.Minute,
		Enabled:  false,
		Prefix:   "test:",
	}

	c, err := cache.NewCache(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Test cache set and get for measures
	t.Run("measure_cache", func(t *testing.T) {
		measure := &cache.CachedMeasure{
			MeasureID: "CMS122",
			Version:   "11.0.0",
			Definition: map[string]interface{}{
				"id":          "CMS122",
				"name":        "Diabetes HbA1c Poor Control",
				"description": "Patients with diabetes and HbA1c > 9%",
			},
		}

		err := c.SetMeasure(ctx, measure)
		if err != nil {
			t.Fatalf("SetMeasure failed: %v", err)
		}

		retrieved, err := c.GetMeasure(ctx, "CMS122")
		if err != nil {
			t.Fatalf("GetMeasure failed: %v", err)
		}

		if retrieved == nil {
			t.Fatal("Expected cached measure, got nil")
		}

		if retrieved.MeasureID != measure.MeasureID {
			t.Errorf("Measure ID mismatch: got %s, want %s", retrieved.MeasureID, measure.MeasureID)
		}
	})
}

// TestEvaluationCaching tests evaluation result cache operations.
func TestEvaluationCaching(t *testing.T) {
	logger := zap.NewNop()

	cfg := cache.Config{
		RedisURL: "",
		TTL:      5 * time.Minute,
		Enabled:  false,
		Prefix:   "test:",
	}

	c, err := cache.NewCache(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	t.Run("evaluation_cache", func(t *testing.T) {
		eval := &cache.CachedEvaluation{
			PatientID: "patient-123",
			MeasureID: "CMS122",
			PeriodKey: "2024-01-01",
			Result: map[string]interface{}{
				"inDenominator": true,
				"inNumerator":   false,
				"gapIdentified": true,
			},
		}

		err := c.SetEvaluation(ctx, eval)
		if err != nil {
			t.Fatalf("SetEvaluation failed: %v", err)
		}

		// Note: GetEvaluation uses a composite key, so this tests the cache key logic
		retrieved, err := c.GetEvaluation(ctx, "patient-123", "CMS122", "2024-01-01", "")
		if err != nil {
			t.Fatalf("GetEvaluation failed: %v", err)
		}

		if retrieved == nil {
			t.Fatal("Expected cached evaluation, got nil")
		}

		if retrieved.PatientID != eval.PatientID {
			t.Errorf("Patient ID mismatch: got %s, want %s", retrieved.PatientID, eval.PatientID)
		}
	})
}

// TestCacheInvalidation tests cache invalidation operations.
func TestCacheInvalidation(t *testing.T) {
	logger := zap.NewNop()

	cfg := cache.Config{
		RedisURL: "",
		TTL:      5 * time.Minute,
		Enabled:  false,
		Prefix:   "test:",
	}

	c, err := cache.NewCache(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Add some cached items
	lib := &cache.CachedLibrary{
		LibraryID: "CMS165",
		Version:   "11.0.0",
		Content:   "library CMS165",
	}
	c.SetCQLLibrary(ctx, lib)

	// Test clear
	t.Run("clear_cache", func(t *testing.T) {
		err := c.Clear(ctx)
		if err != nil {
			t.Fatalf("Clear failed: %v", err)
		}

		stats := c.GetStats()
		if stats.MemoryItems != 0 {
			t.Errorf("Expected 0 memory items after clear, got %d", stats.MemoryItems)
		}
	})

	// Test patient invalidation
	t.Run("invalidate_patient", func(t *testing.T) {
		// Add patient-specific evaluation
		eval := &cache.CachedEvaluation{
			PatientID: "patient-invalidate-test",
			MeasureID: "CMS122",
			PeriodKey: "2024-01-01",
			Result:    map[string]interface{}{"test": true},
		}
		c.SetEvaluation(ctx, eval)

		err := c.InvalidatePatient(ctx, "patient-invalidate-test")
		if err != nil {
			t.Fatalf("InvalidatePatient failed: %v", err)
		}

		// Verify invalidation
		retrieved, _ := c.GetEvaluation(ctx, "patient-invalidate-test", "CMS122", "2024-01-01", "")
		if retrieved != nil {
			t.Error("Expected nil after patient invalidation")
		}
	})
}

// TestCacheStats tests cache statistics.
func TestCacheStats(t *testing.T) {
	logger := zap.NewNop()

	cfg := cache.Config{
		RedisURL: "",
		TTL:      10 * time.Minute,
		Enabled:  false,
		Prefix:   "stats-test:",
	}

	c, err := cache.NewCache(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Add items
	for i := 0; i < 5; i++ {
		lib := &cache.CachedLibrary{
			LibraryID: "Lib" + string(rune('A'+i)),
			Version:   "1.0.0",
			Content:   "content",
		}
		c.SetCQLLibrary(ctx, lib)
	}

	stats := c.GetStats()

	if stats.MemoryItems != 5 {
		t.Errorf("Expected 5 memory items, got %d", stats.MemoryItems)
	}

	if stats.TTL != "10m0s" {
		t.Errorf("Expected TTL '10m0s', got %s", stats.TTL)
	}

	t.Logf("Cache stats: %+v", stats)
}
