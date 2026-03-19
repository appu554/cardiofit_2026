package services

import (
	"testing"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// Create tables manually because the model default:gen_random_uuid() is
	// PostgreSQL-specific and SQLite's AutoMigrate chokes on it.
	sqlDB, _ := db.DB()
	stmts := []string{
		`CREATE TABLE mri_nadirs (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			nadir_score REAL NOT NULL,
			nadir_date DATETIME NOT NULL,
			hb_a1c_nadir REAL,
			hb_a1c_nadir_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE UNIQUE INDEX idx_mri_nadirs_patient_id ON mri_nadirs(patient_id)`,
		`CREATE TABLE relapse_events (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			trigger_type TEXT NOT NULL,
			trigger_value REAL NOT NULL,
			nadir_value REAL NOT NULL,
			current_value REAL NOT NULL,
			action_taken TEXT,
			detected_at DATETIME NOT NULL,
			created_at DATETIME
		)`,
		`CREATE INDEX idx_relapse_events_patient_id ON relapse_events(patient_id)`,
		`CREATE INDEX idx_relapse_events_detected_at ON relapse_events(detected_at)`,
		`CREATE TABLE quarterly_summaries (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			year INTEGER NOT NULL,
			quarter INTEGER NOT NULL,
			mean_mri REAL,
			min_mri REAL,
			max_mri REAL,
			mri_count INTEGER,
			latest_hb_a1c REAL,
			computed_at DATETIME NOT NULL,
			created_at DATETIME
		)`,
		`CREATE INDEX idx_quarterly_summaries_patient_id ON quarterly_summaries(patient_id)`,
	}
	for _, stmt := range stmts {
		if _, err := sqlDB.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt[:40], err)
		}
	}
	return db
}

func newTestRelapseDetector(t *testing.T) *RelapseDetector {
	t.Helper()
	db := setupTestDB(t)
	return NewRelapseDetector(db, zap.NewNop())
}

func TestRelapseDetector_UpdateNadir_SetsInitialNadir(t *testing.T) {
	det := newTestRelapseDetector(t)
	pid := uuid.New()

	err := det.UpdateNadir(pid, 45.0, nil)
	if err != nil {
		t.Fatalf("UpdateNadir: %v", err)
	}

	nadir, err := det.GetNadir(pid)
	if err != nil {
		t.Fatalf("GetNadir: %v", err)
	}
	if nadir.NadirScore != 45.0 {
		t.Errorf("nadir = %.1f, want 45.0", nadir.NadirScore)
	}
}

func TestRelapseDetector_UpdateNadir_OnlyLowersNeverRaises(t *testing.T) {
	det := newTestRelapseDetector(t)
	pid := uuid.New()

	det.UpdateNadir(pid, 45.0, nil)
	det.UpdateNadir(pid, 50.0, nil) // higher — should NOT update nadir
	det.UpdateNadir(pid, 38.0, nil) // lower — SHOULD update

	nadir, _ := det.GetNadir(pid)
	if nadir.NadirScore != 38.0 {
		t.Errorf("nadir = %.1f, want 38.0 (lowest seen)", nadir.NadirScore)
	}
}

func TestRelapseDetector_UpdateNadir_TracksHbA1c(t *testing.T) {
	det := newTestRelapseDetector(t)
	pid := uuid.New()

	hba1c1 := 7.0
	hba1c2 := 6.5
	hba1c3 := 7.5

	det.UpdateNadir(pid, 45.0, &hba1c1)
	det.UpdateNadir(pid, 44.0, &hba1c2) // lower HbA1c — should update
	det.UpdateNadir(pid, 43.0, &hba1c3) // higher HbA1c — should NOT update HbA1c nadir

	nadir, _ := det.GetNadir(pid)
	if nadir.HbA1cNadir == nil || *nadir.HbA1cNadir != 6.5 {
		t.Errorf("HbA1c nadir = %v, want 6.5", nadir.HbA1cNadir)
	}
}

func TestRelapseDetector_CheckRelapse_MRIRise_2ConsecutiveQuarters(t *testing.T) {
	det := newTestRelapseDetector(t)
	pid := uuid.New()

	det.UpdateNadir(pid, 35.0, nil)

	now := time.Now()
	det.db.Create(&models.QuarterlySummary{
		ID: uuid.New(), PatientID: pid, Year: now.Year(), Quarter: quarterOf(now.AddDate(0, -3, 0)),
		MeanMRI: 52.0, MinMRI: 50.0, MaxMRI: 55.0, MRICount: 3, ComputedAt: now.AddDate(0, -3, 0),
	})
	det.db.Create(&models.QuarterlySummary{
		ID: uuid.New(), PatientID: pid, Year: now.Year(), Quarter: quarterOf(now),
		MeanMRI: 54.0, MinMRI: 51.0, MaxMRI: 56.0, MRICount: 3, ComputedAt: now,
	})

	event, err := det.CheckRelapse(pid)
	if err != nil {
		t.Fatalf("CheckRelapse: %v", err)
	}
	if event == nil {
		t.Fatal("expected relapse event, got nil")
	}
	if event.TriggerType != "MRI_RISE" {
		t.Errorf("trigger = %q, want MRI_RISE", event.TriggerType)
	}
	if event.NadirValue != 35.0 {
		t.Errorf("nadir_value = %.1f, want 35.0", event.NadirValue)
	}
}

func TestRelapseDetector_CheckRelapse_OnlyOneQuarter_NoRelapse(t *testing.T) {
	det := newTestRelapseDetector(t)
	pid := uuid.New()

	det.UpdateNadir(pid, 35.0, nil)

	now := time.Now()
	det.db.Create(&models.QuarterlySummary{
		ID: uuid.New(), PatientID: pid, Year: now.Year(), Quarter: quarterOf(now),
		MeanMRI: 55.0, MinMRI: 53.0, MaxMRI: 57.0, MRICount: 3, ComputedAt: now,
	})

	event, _ := det.CheckRelapse(pid)
	if event != nil {
		t.Error("expected no relapse with only 1 quarter of data")
	}
}

func TestRelapseDetector_CheckRelapse_HbA1cRise(t *testing.T) {
	det := newTestRelapseDetector(t)
	pid := uuid.New()

	hba1c := 6.5
	det.UpdateNadir(pid, 35.0, &hba1c)

	// Use explicit year/quarter to guarantee ordering (most recent = Q2 2026)
	latestHbA1c1 := 7.0
	latestHbA1c2 := 7.2
	now := time.Now()
	det.db.Create(&models.QuarterlySummary{
		ID: uuid.New(), PatientID: pid, Year: 2026, Quarter: 1,
		MeanMRI: 40.0, MRICount: 3, LatestHbA1c: &latestHbA1c1, ComputedAt: now.AddDate(0, -3, 0),
	})
	det.db.Create(&models.QuarterlySummary{
		ID: uuid.New(), PatientID: pid, Year: 2026, Quarter: 2,
		MeanMRI: 42.0, MRICount: 3, LatestHbA1c: &latestHbA1c2, ComputedAt: now,
	})

	event, _ := det.CheckRelapse(pid)
	if event == nil {
		t.Fatal("expected HbA1c relapse event, got nil")
	}
	if event.TriggerType != "HBA1C_RISE" {
		t.Errorf("trigger = %q, want HBA1C_RISE", event.TriggerType)
	}
}

func TestRelapseDetector_CheckRelapse_BelowThreshold_NoRelapse(t *testing.T) {
	det := newTestRelapseDetector(t)
	pid := uuid.New()

	det.UpdateNadir(pid, 35.0, nil)

	now := time.Now()
	// MRI rise of only 10 (below 15 threshold)
	det.db.Create(&models.QuarterlySummary{
		ID: uuid.New(), PatientID: pid, Year: now.Year(), Quarter: quarterOf(now.AddDate(0, -3, 0)),
		MeanMRI: 44.0, MRICount: 3, ComputedAt: now.AddDate(0, -3, 0),
	})
	det.db.Create(&models.QuarterlySummary{
		ID: uuid.New(), PatientID: pid, Year: now.Year(), Quarter: quarterOf(now),
		MeanMRI: 45.0, MRICount: 3, ComputedAt: now,
	})

	event, _ := det.CheckRelapse(pid)
	if event != nil {
		t.Error("expected no relapse when MRI rise is below threshold")
	}
}

func TestRelapseDetector_GetRelapseHistory(t *testing.T) {
	det := newTestRelapseDetector(t)
	pid := uuid.New()

	// Seed two relapse events directly
	now := time.Now()
	det.db.Create(&models.RelapseEvent{
		ID: uuid.New(), PatientID: pid, TriggerType: "MRI_RISE",
		TriggerValue: 20.0, NadirValue: 30.0, CurrentValue: 50.0, DetectedAt: now.AddDate(0, -1, 0),
	})
	det.db.Create(&models.RelapseEvent{
		ID: uuid.New(), PatientID: pid, TriggerType: "HBA1C_RISE",
		TriggerValue: 0.8, NadirValue: 6.5, CurrentValue: 7.3, DetectedAt: now,
	})

	events, err := det.GetRelapseHistory(pid)
	if err != nil {
		t.Fatalf("GetRelapseHistory: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("got %d events, want 2", len(events))
	}
	// Most recent first
	if len(events) >= 2 && events[0].TriggerType != "HBA1C_RISE" {
		t.Errorf("first event = %q, want HBA1C_RISE (most recent)", events[0].TriggerType)
	}
}

func TestRelapseDetector_DaysSinceLastRelapse_NoEvents(t *testing.T) {
	det := newTestRelapseDetector(t)
	pid := uuid.New()

	days := det.DaysSinceLastRelapse(pid)
	if days != -1 {
		t.Errorf("days = %d, want -1 (no events)", days)
	}
}

func TestRelapseDetector_DaysSinceLastRelapse_WithEvent(t *testing.T) {
	det := newTestRelapseDetector(t)
	pid := uuid.New()

	det.db.Create(&models.RelapseEvent{
		ID: uuid.New(), PatientID: pid, TriggerType: "MRI_RISE",
		TriggerValue: 20.0, NadirValue: 30.0, CurrentValue: 50.0,
		DetectedAt: time.Now().AddDate(0, 0, -10),
	})

	days := det.DaysSinceLastRelapse(pid)
	if days < 9 || days > 11 {
		t.Errorf("days = %d, want ~10", days)
	}
}

func quarterOf(t time.Time) int {
	return (int(t.Month())-1)/3 + 1
}
