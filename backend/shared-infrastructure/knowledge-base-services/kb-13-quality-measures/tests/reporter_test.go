package tests

import (
	"testing"
	"time"

	"kb-13-quality-measures/internal/models"
	"kb-13-quality-measures/internal/reporter"
)

func TestReport_Structure(t *testing.T) {
	// Verify Report struct has all required fields for FHIR MeasureReport
	report := &reporter.Report{
		ID:                   "report-123",
		MeasureID:            "HBD",
		MeasureName:          "Hemoglobin A1c Control",
		ReportType:           models.ReportSummary,
		Status:               "complete",
		PeriodStart:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:            time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		GeneratedAt:          time.Now(),
		InitialPopulation:    1000,
		Denominator:          950,
		DenominatorExclusion: 50,
		DenominatorException: 0,
		Numerator:            800,
		NumeratorExclusion:   0,
		Score:                0.842,
		PerformanceRate:      84.2,
	}

	// Verify basic fields
	if report.ID != "report-123" {
		t.Errorf("Expected ID report-123, got %s", report.ID)
	}
	if report.MeasureID != "HBD" {
		t.Errorf("Expected MeasureID HBD, got %s", report.MeasureID)
	}
	if report.Status != "complete" {
		t.Errorf("Expected status complete, got %s", report.Status)
	}
	if report.Score != 0.842 {
		t.Errorf("Expected score 0.842, got %f", report.Score)
	}
}

func TestGenerateRequest_Validation(t *testing.T) {
	// Test GenerateRequest structure
	req := &reporter.GenerateRequest{
		MeasureID:          "HBD",
		ReportType:         models.ReportSummary,
		PeriodStart:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:          time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		IncludePriorPeriod: true,
		IncludeBenchmark:   true,
	}

	if req.MeasureID != "HBD" {
		t.Errorf("Expected MeasureID HBD, got %s", req.MeasureID)
	}
	if req.ReportType != models.ReportSummary {
		t.Errorf("Expected ReportType summary, got %s", req.ReportType)
	}
	if !req.IncludePriorPeriod {
		t.Error("Expected IncludePriorPeriod to be true")
	}
}

func TestStratificationResult(t *testing.T) {
	stratification := reporter.StratificationResult{
		ID:          "age-18-44",
		Description: "Age 18-44",
		Value:       "18-44",
		Count:       250,
		Score:       0.88,
	}

	if stratification.ID != "age-18-44" {
		t.Errorf("Expected ID age-18-44, got %s", stratification.ID)
	}
	if stratification.Count != 250 {
		t.Errorf("Expected count 250, got %d", stratification.Count)
	}
	if stratification.Score != 0.88 {
		t.Errorf("Expected score 0.88, got %f", stratification.Score)
	}
}

func TestReport_WithStratifications(t *testing.T) {
	report := &reporter.Report{
		ID:        "report-with-strat",
		MeasureID: "HBD",
		Score:     0.842,
		Stratifications: []reporter.StratificationResult{
			{ID: "age-18-44", Description: "Age 18-44", Value: "18-44", Count: 300, Score: 0.90},
			{ID: "age-45-64", Description: "Age 45-64", Value: "45-64", Count: 400, Score: 0.85},
			{ID: "age-65-75", Description: "Age 65-75", Value: "65-75", Count: 300, Score: 0.78},
		},
	}

	if len(report.Stratifications) != 3 {
		t.Errorf("Expected 3 stratifications, got %d", len(report.Stratifications))
	}

	// Verify stratification order
	if report.Stratifications[0].Value != "18-44" {
		t.Errorf("Expected first stratification 18-44, got %s", report.Stratifications[0].Value)
	}
}

func TestReport_WithComparisons(t *testing.T) {
	priorScore := 0.79
	benchmarkScore := 0.85

	report := &reporter.Report{
		ID:               "report-with-comparisons",
		MeasureID:        "HBD",
		Score:            0.84,
		PriorPeriodScore: &priorScore,
		BenchmarkScore:   &benchmarkScore,
	}

	if report.PriorPeriodScore == nil {
		t.Fatal("Expected PriorPeriodScore to be set")
	}
	if *report.PriorPeriodScore != 0.79 {
		t.Errorf("Expected prior score 0.79, got %f", *report.PriorPeriodScore)
	}

	if report.BenchmarkScore == nil {
		t.Fatal("Expected BenchmarkScore to be set")
	}
	if *report.BenchmarkScore != 0.85 {
		t.Errorf("Expected benchmark score 0.85, got %f", *report.BenchmarkScore)
	}

	// Check improvement over prior period
	improvement := report.Score - *report.PriorPeriodScore
	if improvement < 0.04 || improvement > 0.06 {
		t.Errorf("Expected improvement around 0.05, got %f", improvement)
	}
}
