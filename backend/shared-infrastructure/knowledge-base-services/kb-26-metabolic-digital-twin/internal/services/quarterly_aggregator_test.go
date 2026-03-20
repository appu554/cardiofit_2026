package services

import (
	"testing"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestQuarterlyAggregator_Compute(t *testing.T) {
	db := setupTestDB(t)

	// Create mri_scores table and the unique index needed for UPSERT.
	sqlDB, _ := db.DB()
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS mri_scores (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			score REAL NOT NULL,
			category TEXT NOT NULL,
			trend TEXT,
			top_driver TEXT,
			glucose_domain REAL,
			body_comp_domain REAL,
			cardio_domain REAL,
			behavioral_domain REAL,
			signal_z_scores TEXT,
			twin_state_id TEXT,
			computed_at DATETIME NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_qs_unique ON quarterly_summaries(patient_id, year, quarter)`,
	} {
		if _, err := sqlDB.Exec(stmt); err != nil {
			t.Fatalf("exec DDL: %v", err)
		}
	}

	agg := NewQuarterlyAggregator(db, zap.NewNop())

	pid := uuid.New()
	now := time.Now().UTC()

	// Seed 3 MRI scores in the current quarter.
	for _, score := range []float64{45.0, 50.0, 55.0} {
		db.Create(&models.MRIScore{
			ID: uuid.New(), PatientID: pid, Score: score, Category: "MILD_DYSREGULATION",
			Trend: "STABLE", ComputedAt: now,
		})
	}

	year := now.Year()
	q := (int(now.Month())-1)/3 + 1

	err := agg.ComputeQuarter(pid, year, q)
	if err != nil {
		t.Fatalf("ComputeQuarter: %v", err)
	}

	var summary models.QuarterlySummary
	db.Where("patient_id = ? AND year = ? AND quarter = ?", pid, year, q).First(&summary)

	if summary.MeanMRI != 50.0 {
		t.Errorf("mean MRI = %.1f, want 50.0", summary.MeanMRI)
	}
	if summary.MinMRI != 45.0 {
		t.Errorf("min MRI = %.1f, want 45.0", summary.MinMRI)
	}
	if summary.MaxMRI != 55.0 {
		t.Errorf("max MRI = %.1f, want 55.0", summary.MaxMRI)
	}
	if summary.MRICount != 3 {
		t.Errorf("MRI count = %d, want 3", summary.MRICount)
	}
}

func TestQuarterlyAggregator_Compute_Upsert(t *testing.T) {
	db := setupTestDB(t)

	sqlDB, _ := db.DB()
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS mri_scores (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			score REAL NOT NULL,
			category TEXT NOT NULL,
			trend TEXT,
			top_driver TEXT,
			glucose_domain REAL,
			body_comp_domain REAL,
			cardio_domain REAL,
			behavioral_domain REAL,
			signal_z_scores TEXT,
			twin_state_id TEXT,
			computed_at DATETIME NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_qs_unique ON quarterly_summaries(patient_id, year, quarter)`,
	} {
		if _, err := sqlDB.Exec(stmt); err != nil {
			t.Fatalf("exec DDL: %v", err)
		}
	}

	agg := NewQuarterlyAggregator(db, zap.NewNop())

	pid := uuid.New()
	now := time.Now().UTC()
	year := now.Year()
	q := (int(now.Month())-1)/3 + 1

	// Seed 2 scores, compute quarter.
	for _, score := range []float64{40.0, 60.0} {
		db.Create(&models.MRIScore{
			ID: uuid.New(), PatientID: pid, Score: score, Category: "MILD_DYSREGULATION",
			Trend: "STABLE", ComputedAt: now,
		})
	}
	if err := agg.ComputeQuarter(pid, year, q); err != nil {
		t.Fatalf("first ComputeQuarter: %v", err)
	}

	// Add another score and re-compute (upsert).
	db.Create(&models.MRIScore{
		ID: uuid.New(), PatientID: pid, Score: 50.0, Category: "MILD_DYSREGULATION",
		Trend: "STABLE", ComputedAt: now,
	})
	if err := agg.ComputeQuarter(pid, year, q); err != nil {
		t.Fatalf("second ComputeQuarter (upsert): %v", err)
	}

	var summaries []models.QuarterlySummary
	db.Where("patient_id = ? AND year = ? AND quarter = ?", pid, year, q).Find(&summaries)
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary after upsert, got %d", len(summaries))
	}
	if summaries[0].MRICount != 3 {
		t.Errorf("MRI count = %d after upsert, want 3", summaries[0].MRICount)
	}
	if summaries[0].MeanMRI != 50.0 {
		t.Errorf("mean MRI = %.1f after upsert, want 50.0", summaries[0].MeanMRI)
	}
}

func TestQuarterlyAggregator_NoData_NoSummary(t *testing.T) {
	db := setupTestDB(t)

	sqlDB, _ := db.DB()
	sqlDB.Exec(`CREATE TABLE IF NOT EXISTS mri_scores (
		id TEXT PRIMARY KEY, patient_id TEXT NOT NULL, score REAL NOT NULL,
		category TEXT NOT NULL, trend TEXT, top_driver TEXT, glucose_domain REAL,
		body_comp_domain REAL, cardio_domain REAL, behavioral_domain REAL,
		signal_z_scores TEXT, twin_state_id TEXT, computed_at DATETIME NOT NULL
	)`)

	agg := NewQuarterlyAggregator(db, zap.NewNop())
	pid := uuid.New()

	err := agg.ComputeQuarter(pid, 2026, 1)
	if err != nil {
		t.Fatalf("ComputeQuarter with no data should return nil, got: %v", err)
	}

	var count int64
	db.Model(&models.QuarterlySummary{}).Where("patient_id = ?", pid).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 summaries when no MRI data, got %d", count)
	}
}
