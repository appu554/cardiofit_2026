package tests

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-13-quality-measures/internal/calculator"
	"kb-13-quality-measures/internal/models"
	"kb-13-quality-measures/internal/period"
)

func TestCareGapDetector_DetectCareGaps(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	detector := calculator.NewCareGapDetector(nil, logger)

	// Create test measure
	measure := &models.Measure{
		ID:          "HBD",
		Name:        "Hemoglobin A1c Control",
		Title:       "HbA1c < 8%",
		Type:        models.MeasureTypeProcess,
		Domain:      models.DomainDiabetes,
		Active:      true,
	}

	// Create test period
	mp := &period.MeasurementPeriod{
		Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	}

	// Test detection request with patients in denominator but not numerator
	req := &calculator.DetectionRequest{
		MeasureID: "HBD",
		Measure:   measure,
		Period:    mp,
		DenominatorPatientIDs: []string{"patient-1", "patient-2", "patient-3", "patient-4", "patient-5"},
		NumeratorPatientIDs:   []string{"patient-1", "patient-3", "patient-5"},
	}

	gaps, err := detector.DetectCareGaps(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should find 2 gaps (patient-2 and patient-4)
	if len(gaps) != 2 {
		t.Errorf("Expected 2 care gaps, got %d", len(gaps))
	}

	// Verify gap properties
	for _, gap := range gaps {
		// Check source is QUALITY_MEASURE (KB-13 derived)
		if gap.Source != "QUALITY_MEASURE" {
			t.Errorf("Expected source QUALITY_MEASURE, got %s", gap.Source)
		}

		// Check IsAuthoritative is false
		if gap.IsAuthoritative {
			t.Error("Expected IsAuthoritative to be false for KB-13 derived gaps")
		}

		// Check measure ID
		if gap.MeasureID != "HBD" {
			t.Errorf("Expected measure ID HBD, got %s", gap.MeasureID)
		}

		// Check priority is high for diabetes domain
		if gap.Priority != models.PriorityHigh {
			t.Errorf("Expected priority high for diabetes measure, got %s", gap.Priority)
		}
	}
}

func TestCareGapDetector_NoGaps(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	detector := calculator.NewCareGapDetector(nil, logger)

	measure := &models.Measure{
		ID:     "TEST",
		Domain: models.DomainPreventive,
	}

	mp := &period.MeasurementPeriod{
		Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	}

	// All denominator patients are also in numerator
	req := &calculator.DetectionRequest{
		MeasureID:             "TEST",
		Measure:               measure,
		Period:                mp,
		DenominatorPatientIDs: []string{"patient-1", "patient-2"},
		NumeratorPatientIDs:   []string{"patient-1", "patient-2"},
	}

	gaps, err := detector.DetectCareGaps(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(gaps) != 0 {
		t.Errorf("Expected 0 care gaps, got %d", len(gaps))
	}
}

func TestCareGapDetector_BulkDetect(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	detector := calculator.NewCareGapDetector(nil, logger)

	mp := &period.MeasurementPeriod{
		Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	}

	requests := []*calculator.DetectionRequest{
		{
			MeasureID:             "HBD",
			Measure:               &models.Measure{ID: "HBD", Domain: models.DomainDiabetes},
			Period:                mp,
			DenominatorPatientIDs: []string{"p1", "p2"},
			NumeratorPatientIDs:   []string{"p1"},
		},
		{
			MeasureID:             "CBP",
			Measure:               &models.Measure{ID: "CBP", Domain: models.DomainCardiovascular},
			Period:                mp,
			DenominatorPatientIDs: []string{"p1", "p2", "p3"},
			NumeratorPatientIDs:   []string{"p2"},
		},
	}

	results, err := detector.BulkDetectCareGaps(context.Background(), requests)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check HBD gaps
	if len(results["HBD"]) != 1 {
		t.Errorf("Expected 1 HBD gap, got %d", len(results["HBD"]))
	}

	// Check CBP gaps
	if len(results["CBP"]) != 2 {
		t.Errorf("Expected 2 CBP gaps, got %d", len(results["CBP"]))
	}
}

func TestCareGapDetector_SummarizeCareGaps(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	detector := calculator.NewCareGapDetector(nil, logger)

	now := time.Now()
	gaps := []*models.CareGap{
		{
			ID:        "gap-1",
			MeasureID: "HBD",
			Priority:  models.PriorityHigh,
			Status:    models.CareGapStatusOpen,
			CreatedAt: now.Add(-48 * time.Hour),
		},
		{
			ID:        "gap-2",
			MeasureID: "HBD",
			Priority:  models.PriorityMedium,
			Status:    models.CareGapStatusOpen,
			CreatedAt: now.Add(-24 * time.Hour),
		},
		{
			ID:        "gap-3",
			MeasureID: "HBD",
			Priority:  models.PriorityHigh,
			Status:    models.CareGapStatusInProgress,
			CreatedAt: now,
		},
	}

	summary := detector.SummarizeCareGaps(gaps)

	if summary.TotalGaps != 3 {
		t.Errorf("Expected 3 total gaps, got %d", summary.TotalGaps)
	}

	if summary.ByPriority["high"] != 2 {
		t.Errorf("Expected 2 high priority gaps, got %d", summary.ByPriority["high"])
	}

	if summary.ByStatus["open"] != 2 {
		t.Errorf("Expected 2 open gaps, got %d", summary.ByStatus["open"])
	}
}
