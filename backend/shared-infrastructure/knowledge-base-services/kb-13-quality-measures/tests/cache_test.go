// Package tests provides unit and integration tests for KB-13 Quality Measures.
package tests

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-13-quality-measures/internal/calculator"
	"kb-13-quality-measures/internal/models"
)

func TestCache_SetAndGet(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cache := calculator.NewCache(&calculator.CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Enabled: true,
	}, logger)

	// Create test result
	result := &models.CalculationResult{
		ID:                   "test-result-1",
		MeasureID:            "HBD",
		InitialPopulation:    100,
		Denominator:          90,
		Numerator:            75,
		Score:                0.833,
	}

	key := calculator.CacheKey("HBD", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), models.ReportSummary)

	// Test Set
	cache.Set(key, result)

	// Test Get
	retrieved, found := cache.Get(key)
	if !found {
		t.Fatal("Expected to find cached result")
	}
	if retrieved.ID != result.ID {
		t.Errorf("Expected ID %s, got %s", result.ID, retrieved.ID)
	}
	if retrieved.Score != result.Score {
		t.Errorf("Expected Score %f, got %f", result.Score, retrieved.Score)
	}
}

func TestCache_NotFound(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cache := calculator.NewCache(&calculator.CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Enabled: true,
	}, logger)

	_, found := cache.Get("non-existent-key")
	if found {
		t.Error("Expected not to find non-existent key")
	}
}

func TestCache_Delete(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cache := calculator.NewCache(&calculator.CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Enabled: true,
	}, logger)

	result := &models.CalculationResult{ID: "test-1"}
	key := "test-key"

	cache.Set(key, result)
	cache.Delete(key)

	_, found := cache.Get(key)
	if found {
		t.Error("Expected key to be deleted")
	}
}

func TestCache_Clear(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cache := calculator.NewCache(&calculator.CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Enabled: true,
	}, logger)

	// Add multiple entries
	for i := 0; i < 10; i++ {
		cache.Set("key-"+string(rune('0'+i)), &models.CalculationResult{ID: "test"})
	}

	if cache.Size() != 10 {
		t.Errorf("Expected size 10, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", cache.Size())
	}
}

func TestCache_MaxSize(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cache := calculator.NewCache(&calculator.CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 5,
		Enabled: true,
	}, logger)

	// Add more than max entries
	for i := 0; i < 10; i++ {
		cache.Set("key-"+string(rune('A'+i)), &models.CalculationResult{ID: "test"})
	}

	// Size should not exceed max
	if cache.Size() > 5 {
		t.Errorf("Cache size %d exceeds max size 5", cache.Size())
	}
}

func TestCache_Stats(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cache := calculator.NewCache(&calculator.CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Enabled: true,
	}, logger)

	cache.Set("key-1", &models.CalculationResult{ID: "test-1"})
	cache.Set("key-2", &models.CalculationResult{ID: "test-2"})

	stats := cache.Stats()
	if stats.Size != 2 {
		t.Errorf("Expected stats size 2, got %d", stats.Size)
	}
	if stats.MaxSize != 100 {
		t.Errorf("Expected max size 100, got %d", stats.MaxSize)
	}
}

func TestCacheKey_Format(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	key := calculator.CacheKey("HBD", start, end, models.ReportSummary)
	expected := "HBD|2024-01-01|2024-12-31|summary"

	if key != expected {
		t.Errorf("Expected key %s, got %s", expected, key)
	}
}
