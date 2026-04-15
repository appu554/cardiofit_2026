package services

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/models"
)

// setupCGMPeriodReportTestDB creates an in-memory sqlite DB with the
// cgm_period_reports schema needed by the repository tests.
func setupCGMPeriodReportTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Use GORM AutoMigrate so the test schema tracks the production
	// model exactly — any field addition propagates without a raw
	// DDL update here.
	if err := db.AutoMigrate(&models.CGMPeriodReport{}); err != nil {
		t.Fatalf("auto-migrate CGMPeriodReport: %v", err)
	}
	return db
}

// TestEventToPeriodReport_MapsAllCoreFields pins the wire-to-model
// field translation. A mismatch here would silently lose data between
// the Flink sink and the GORM row. Phase 7 P7-E Milestone 2.
func TestEventToPeriodReport_MapsAllCoreFields(t *testing.T) {
	windowEndMs := int64(1776249000000) // 2026-04-15T10:30:00Z
	evt := CGMAnalyticsEventPayload{
		PatientID:                   "p-map",
		WindowEndMs:                 windowEndMs,
		WindowDays:                  14,
		CoveragePct:                 100.0,
		SufficientData:              true,
		ConfidenceLvl:               "HIGH",
		MeanGlucose:                 140.0,
		SDGlucose:                   15.2,
		CVPct:                       10.9,
		GlucoseStable:               true,
		TIRPct:                      92.5,
		TBRL1Pct:                    2.0,
		TBRL2Pct:                    0.5,
		TARL1Pct:                    3.5,
		TARL2Pct:                    1.5,
		GMI:                         6.8,
		GRI:                         12.5,
		GRIZone:                     "A",
		SustainedHypoDetected:       true,
		SustainedSevereHypoDetected: false,
		SustainedHyperDetected:      true,
		NocturnalHypoDetected:       false,
	}

	report := eventToPeriodReport(evt)
	if report == nil {
		t.Fatal("expected non-nil report")
	}

	if report.PatientID != "p-map" {
		t.Errorf("PatientID = %q, want p-map", report.PatientID)
	}
	// PeriodEnd should be exactly windowEndMs.
	if report.PeriodEnd.UnixMilli() != windowEndMs {
		t.Errorf("PeriodEnd = %d ms, want %d ms", report.PeriodEnd.UnixMilli(), windowEndMs)
	}
	// PeriodStart should be 14 days before PeriodEnd.
	delta := report.PeriodEnd.Sub(report.PeriodStart)
	if delta != 14*24*time.Hour {
		t.Errorf("period delta = %v, want 14 days", delta)
	}
	if report.TIRPct != 92.5 {
		t.Errorf("TIRPct = %f, want 92.5", report.TIRPct)
	}
	if report.MeanGlucose != 140.0 {
		t.Errorf("MeanGlucose = %f, want 140.0", report.MeanGlucose)
	}
	if report.GRIZone != "A" {
		t.Errorf("GRIZone = %q, want A", report.GRIZone)
	}
	if report.HypoEvents != 1 {
		t.Errorf("HypoEvents = %d, want 1 (sustained hypo detected)", report.HypoEvents)
	}
	if report.HyperEvents != 1 {
		t.Errorf("HyperEvents = %d, want 1 (sustained hyper detected)", report.HyperEvents)
	}
	if report.SevereHypoEvents != 0 {
		t.Errorf("SevereHypoEvents = %d, want 0", report.SevereHypoEvents)
	}
	if report.NocturnalHypos != 0 {
		t.Errorf("NocturnalHypos = %d, want 0", report.NocturnalHypos)
	}
}

// TestCGMPeriodReportRepository_SaveAndFetch verifies the repository
// can write a report and then fetch the latest for that patient.
func TestCGMPeriodReportRepository_SaveAndFetch(t *testing.T) {
	db := setupCGMPeriodReportTestDB(t)
	repo := NewCGMPeriodReportRepository(db, zap.NewNop())

	// Older report
	oldReport := &models.CGMPeriodReport{
		PatientID:  "p-1",
		PeriodEnd:  time.Now().UTC().AddDate(0, 0, -7),
		TIRPct:     70.0,
		GRIZone:    "B",
	}
	if err := repo.SavePeriodReport(oldReport); err != nil {
		t.Fatalf("SavePeriodReport (old): %v", err)
	}
	// Newer report
	newReport := &models.CGMPeriodReport{
		PatientID:  "p-1",
		PeriodEnd:  time.Now().UTC(),
		TIRPct:     80.0,
		GRIZone:    "A",
	}
	if err := repo.SavePeriodReport(newReport); err != nil {
		t.Fatalf("SavePeriodReport (new): %v", err)
	}

	latest, err := repo.FetchLatestPeriodReport("p-1")
	if err != nil {
		t.Fatalf("FetchLatestPeriodReport: %v", err)
	}
	if latest == nil {
		t.Fatal("expected non-nil latest report")
	}
	if latest.TIRPct != 80.0 {
		t.Errorf("latest TIRPct = %f, want 80.0 (newer row should win)", latest.TIRPct)
	}
	if latest.GRIZone != "A" {
		t.Errorf("latest GRIZone = %q, want A", latest.GRIZone)
	}
}

// TestCGMPeriodReportRepository_FetchLatest_NoRows returns (nil, nil)
// when the patient has no reports.
func TestCGMPeriodReportRepository_FetchLatest_NoRows(t *testing.T) {
	db := setupCGMPeriodReportTestDB(t)
	repo := NewCGMPeriodReportRepository(db, zap.NewNop())

	latest, err := repo.FetchLatestPeriodReport("p-nonexistent")
	if err != nil {
		t.Errorf("expected nil error for missing patient, got %v", err)
	}
	if latest != nil {
		t.Errorf("expected nil report for missing patient, got %+v", latest)
	}
}

// TestPersistingCGMAnalyticsHandler_WritesRow verifies the
// Milestone 2 handler converts a wire event into a database row
// end-to-end.
func TestPersistingCGMAnalyticsHandler_WritesRow(t *testing.T) {
	db := setupCGMPeriodReportTestDB(t)
	repo := NewCGMPeriodReportRepository(db, zap.NewNop())
	handler := PersistingCGMAnalyticsHandler(repo, zap.NewNop())

	evt := CGMAnalyticsEventPayload{
		PatientID:   "p-handler",
		WindowEndMs: time.Now().UnixMilli(),
		WindowDays:  14,
		TIRPct:      85.0,
		MeanGlucose: 135.0,
		GRIZone:     "A",
	}
	if err := handler(context.Background(), evt); err != nil {
		t.Fatalf("handler: %v", err)
	}

	latest, err := repo.FetchLatestPeriodReport("p-handler")
	if err != nil {
		t.Fatalf("FetchLatest: %v", err)
	}
	if latest == nil {
		t.Fatal("expected persisted row, got nil")
	}
	if latest.TIRPct != 85.0 {
		t.Errorf("persisted TIRPct = %f, want 85.0", latest.TIRPct)
	}
}
