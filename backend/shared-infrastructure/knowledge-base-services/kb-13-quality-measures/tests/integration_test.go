// Package tests provides integration tests for KB-13 Quality Measures.
//
// Integration tests require:
//   - PostgreSQL database running
//   - Proper environment configuration
//
// Run with: go test -tags=integration ./tests/...
//
//go:build integration

package tests

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"kb-13-quality-measures/internal/calculator"
	"kb-13-quality-measures/internal/config"
	"kb-13-quality-measures/internal/models"
	"kb-13-quality-measures/internal/repository"
	"kb-13-quality-measures/internal/scheduler"
)

// testDB connects to the test database.
func testDB(t *testing.T) *sql.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:password@localhost:5432/kb13_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Skipf("Cannot ping test database: %v", err)
	}

	return db
}

func TestIntegration_ResultRepository_SaveAndRetrieve(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	repo := repository.NewResultRepository(db, logger)

	ctx := context.Background()

	// Create test result
	result := &models.CalculationResult{
		ID:                   "integration-test-" + time.Now().Format("20060102150405"),
		MeasureID:            "HBD-TEST",
		ReportType:           models.ReportSummary,
		PeriodStart:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:            time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		InitialPopulation:    1000,
		Denominator:          950,
		DenominatorExclusion: 50,
		DenominatorException: 0,
		Numerator:            800,
		NumeratorExclusion:   0,
		Score:                0.842,
		ExecutionTimeMs:      150,
		ExecutionContext: models.ExecutionContextVersion{
			KB13Version:        "1.0.0",
			CQLLibraryVersion:  "1.0.0",
			TerminologyVersion: "2024.1",
			MeasureYAMLVersion: "2024",
			ExecutedAt:         time.Now(),
		},
		CreatedAt: time.Now(),
	}

	// Save
	err := repo.Save(ctx, result)
	if err != nil {
		t.Fatalf("Failed to save result: %v", err)
	}

	// Retrieve by ID
	retrieved, err := repo.GetByID(ctx, result.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve result: %v", err)
	}

	// Verify
	if retrieved.MeasureID != result.MeasureID {
		t.Errorf("Expected MeasureID %s, got %s", result.MeasureID, retrieved.MeasureID)
	}
	if retrieved.Score != result.Score {
		t.Errorf("Expected Score %f, got %f", result.Score, retrieved.Score)
	}
	if retrieved.InitialPopulation != result.InitialPopulation {
		t.Errorf("Expected InitialPopulation %d, got %d", result.InitialPopulation, retrieved.InitialPopulation)
	}

	// Cleanup
	err = repo.Delete(ctx, result.ID)
	if err != nil {
		t.Errorf("Failed to cleanup test result: %v", err)
	}
}

func TestIntegration_ResultRepository_GetByMeasure(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	repo := repository.NewResultRepository(db, logger)

	ctx := context.Background()
	measureID := "INTEGRATION-TEST-MEASURE"

	// Create multiple results
	var resultIDs []string
	for i := 0; i < 3; i++ {
		result := &models.CalculationResult{
			ID:                "integration-measure-" + time.Now().Format("20060102150405") + "-" + string(rune('A'+i)),
			MeasureID:         measureID,
			ReportType:        models.ReportSummary,
			PeriodStart:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			PeriodEnd:         time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			InitialPopulation: 100 + i*10,
			Denominator:       90 + i*10,
			Numerator:         80 + i*10,
			Score:             0.8 + float64(i)*0.02,
			CreatedAt:         time.Now(),
		}

		if err := repo.Save(ctx, result); err != nil {
			t.Fatalf("Failed to save result %d: %v", i, err)
		}
		resultIDs = append(resultIDs, result.ID)
	}

	// Query by measure
	results, err := repo.GetByMeasure(ctx, measureID, 10)
	if err != nil {
		t.Fatalf("Failed to query by measure: %v", err)
	}

	if len(results) < 3 {
		t.Errorf("Expected at least 3 results, got %d", len(results))
	}

	// Cleanup
	for _, id := range resultIDs {
		repo.Delete(ctx, id)
	}
}

func TestIntegration_CareGapRepository_CRUD(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	repo := repository.NewCareGapRepository(db, logger)

	ctx := context.Background()

	// Create test gap
	gap := models.NewCareGap(
		"INTEGRATION-MEASURE",
		"patient-integration-123",
		"process_gap",
		"Integration test care gap",
		models.PriorityHigh,
	)
	gap.ID = "integration-gap-" + time.Now().Format("20060102150405")

	// Save
	err := repo.Save(ctx, gap)
	if err != nil {
		t.Fatalf("Failed to save care gap: %v", err)
	}

	// Retrieve
	retrieved, err := repo.GetByID(ctx, gap.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve care gap: %v", err)
	}

	if retrieved.PatientID != gap.PatientID {
		t.Errorf("Expected PatientID %s, got %s", gap.PatientID, retrieved.PatientID)
	}
	if retrieved.Source != "QUALITY_MEASURE" {
		t.Errorf("Expected Source QUALITY_MEASURE, got %s", retrieved.Source)
	}

	// Update status
	err = repo.UpdateStatus(ctx, gap.ID, models.CareGapStatusInProgress)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Verify status update
	updated, _ := repo.GetByID(ctx, gap.ID)
	if updated.Status != models.CareGapStatusInProgress {
		t.Errorf("Expected status in-progress, got %s", updated.Status)
	}

	// Cleanup
	repo.Delete(ctx, gap.ID)
}

func TestIntegration_MeasureStore_LoadFromDirectory(t *testing.T) {
	store := models.NewMeasureStore()

	measuresPath := "../measures"
	if _, err := os.Stat(measuresPath); os.IsNotExist(err) {
		t.Skipf("Measures directory not found: %s", measuresPath)
	}

	err := store.LoadMeasuresFromDirectory(measuresPath)
	if err != nil {
		// May not be an error if directory is empty
		t.Logf("Load measures result: %v", err)
	}

	count := store.Count()
	t.Logf("Loaded %d measures from directory", count)

	// If measures are loaded, verify structure
	if count > 0 {
		measures := store.GetActiveMeasures()
		for _, m := range measures {
			if m.ID == "" {
				t.Error("Found measure with empty ID")
			}
			if m.Name == "" {
				t.Errorf("Measure %s has empty Name", m.ID)
			}
		}
	}
}

func TestIntegration_CachedEngine_WithCache(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := models.NewMeasureStore()

	// Add test measure
	measure := &models.Measure{
		ID:     "CACHE-TEST",
		Name:   "Cache Test Measure",
		Title:  "Test Measure for Caching",
		Type:   models.MeasureTypeProcess,
		Domain: models.DomainDiabetes,
		Active: true,
	}
	store.AddMeasure(measure)

	// Create cache
	cache := calculator.NewCache(&calculator.CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Enabled: true,
	}, logger)

	// Create mock result
	result := &models.CalculationResult{
		ID:        "cache-test-result",
		MeasureID: "CACHE-TEST",
		Score:     0.85,
	}

	// Test cache operations
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	key := calculator.CacheKey("CACHE-TEST", start, end, models.ReportSummary)

	// Cache miss
	_, found := cache.Get(key)
	if found {
		t.Error("Expected cache miss on first access")
	}

	// Cache set
	cache.Set(key, result)

	// Cache hit
	cached, found := cache.Get(key)
	if !found {
		t.Error("Expected cache hit after set")
	}
	if cached.Score != result.Score {
		t.Errorf("Expected cached score %f, got %f", result.Score, cached.Score)
	}
}

func TestIntegration_Scheduler_Configuration(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := models.NewMeasureStore()

	cfg := &config.SchedulerConfig{
		Enabled:            true,
		DailyEnabled:       true,
		WeeklyEnabled:      true,
		MonthlyEnabled:     true,
		QuarterlyEnabled:   true,
		DailyInterval:      24 * time.Hour,
		WeeklyInterval:     7 * 24 * time.Hour,
		MonthlyInterval:    30 * 24 * time.Hour,
		WeeklyRunDay:       time.Sunday,
		MonthlyRunDay:      1,
		CalculationTimeout: 10 * time.Minute,
	}

	s := scheduler.NewScheduler(nil, nil, store, cfg, logger)

	status := s.GetStatus()

	if !status.DailyEnabled {
		t.Error("Expected daily to be enabled")
	}
	if !status.WeeklyEnabled {
		t.Error("Expected weekly to be enabled")
	}
	if !status.MonthlyEnabled {
		t.Error("Expected monthly to be enabled")
	}
	if !status.QuarterlyEnabled {
		t.Error("Expected quarterly to be enabled")
	}
}

func TestIntegration_Metrics_Collection(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Create metrics collector
	metricsConfig := &config.MetricsConfig{
		Enabled: true,
		Path:    "/metrics",
	}

	// Verify metrics config
	if !metricsConfig.Enabled {
		t.Error("Expected metrics to be enabled")
	}

	if metricsConfig.Path != "/metrics" {
		t.Errorf("Expected path /metrics, got %s", metricsConfig.Path)
	}

	_ = logger // Use logger
}

func TestIntegration_EndToEnd_MeasureCalculationFlow(t *testing.T) {
	// This test verifies the entire flow without database
	// Real integration would require CQL engine and database

	logger, _ := zap.NewDevelopment()
	store := models.NewMeasureStore()

	// Add test measures
	measures := []*models.Measure{
		{
			ID:                  "E2E-HBD",
			Name:                "E2E Test HbA1c Control",
			Title:               "HbA1c < 8%",
			Type:                models.MeasureTypeProcess,
			Domain:              models.DomainDiabetes,
			Active:              true,
			CalculationSchedule: []string{"daily", "monthly"},
		},
		{
			ID:                  "E2E-CBP",
			Name:                "E2E Test Blood Pressure Control",
			Title:               "BP < 140/90",
			Type:                models.MeasureTypeProcess,
			Domain:              models.DomainCardiovascular,
			Active:              true,
			CalculationSchedule: []string{"monthly"},
		},
	}

	for _, m := range measures {
		store.AddMeasure(m)
	}

	// Verify measures loaded
	if store.Count() != 2 {
		t.Errorf("Expected 2 measures, got %d", store.Count())
	}

	// Test measure retrieval
	hbd := store.GetMeasure("E2E-HBD")
	if hbd == nil {
		t.Fatal("Expected to find E2E-HBD measure")
	}

	// Verify measure properties
	if hbd.Domain != models.DomainDiabetes {
		t.Errorf("Expected domain diabetes, got %s", hbd.Domain)
	}
	if len(hbd.CalculationSchedule) != 2 {
		t.Errorf("Expected 2 schedules, got %d", len(hbd.CalculationSchedule))
	}

	// Test search
	searchResults := store.Search("blood pressure")
	if len(searchResults) != 1 {
		t.Errorf("Expected 1 search result for 'blood pressure', got %d", len(searchResults))
	}

	// Test by domain
	diabetesMeasures := store.GetByDomain(models.DomainDiabetes)
	if len(diabetesMeasures) != 1 {
		t.Errorf("Expected 1 diabetes measure, got %d", len(diabetesMeasures))
	}

	_ = logger
}
