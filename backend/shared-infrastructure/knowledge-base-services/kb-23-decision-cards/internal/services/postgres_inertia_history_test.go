package services

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-23-decision-cards/internal/models"
)

func setupInertiaHistoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Raw DDL to avoid Postgres gen_random_uuid() default issue.
	err = db.Exec(`
		CREATE TABLE inertia_verdict_history (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			week_start_date DATETIME NOT NULL,
			verdicts_json TEXT NOT NULL,
			dual_domain_detected INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			UNIQUE(patient_id, week_start_date)
		)
	`).Error
	if err != nil {
		t.Fatalf("create inertia_verdict_history: %v", err)
	}
	return db
}

// TestPostgresInertiaHistory_SaveAndFetchLatest verifies the basic
// write + read round-trip: a verdict saved for Week 1 can be fetched
// back with the correct patient_id, week_start, and verdicts JSON.
func TestPostgresInertiaHistory_SaveAndFetchLatest(t *testing.T) {
	db := setupInertiaHistoryTestDB(t)
	history := NewPostgresInertiaHistory(db, zap.NewNop())

	week1 := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC) // Monday
	report := models.PatientInertiaReport{
		PatientID: "p-test",
		Verdicts: []models.InertiaVerdict{
			{Domain: models.DomainGlycaemic, Detected: true, Severity: models.SeverityModerate},
		},
		HasAnyInertia: true,
	}

	if err := history.SaveVerdict("p-test", week1, report); err != nil {
		t.Fatalf("SaveVerdict: %v", err)
	}

	got, weekStart, ok := history.FetchLatest("p-test")
	if !ok {
		t.Fatal("expected FetchLatest to return ok=true")
	}
	if !weekStart.Equal(week1) {
		t.Errorf("weekStart = %v, want %v", weekStart, week1)
	}
	if len(got.Verdicts) != 1 {
		t.Fatalf("len(Verdicts) = %d, want 1", len(got.Verdicts))
	}
	if got.Verdicts[0].Domain != models.DomainGlycaemic {
		t.Errorf("Domain = %q, want GLYCAEMIC", got.Verdicts[0].Domain)
	}
}

// TestPostgresInertiaHistory_Upsert verifies that a second
// SaveVerdict for the same (patient_id, week_start_date) updates
// the existing row rather than creating a duplicate.
func TestPostgresInertiaHistory_Upsert(t *testing.T) {
	db := setupInertiaHistoryTestDB(t)
	history := NewPostgresInertiaHistory(db, zap.NewNop())

	week := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	r1 := models.PatientInertiaReport{Verdicts: []models.InertiaVerdict{
		{Domain: models.DomainGlycaemic, Detected: true},
	}}
	r2 := models.PatientInertiaReport{Verdicts: []models.InertiaVerdict{
		{Domain: models.DomainHemodynamic, Detected: true},
	}}

	if err := history.SaveVerdict("p-upsert", week, r1); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := history.SaveVerdict("p-upsert", week, r2); err != nil {
		t.Fatalf("second save (upsert): %v", err)
	}

	// Only one row should exist.
	var count int64
	db.Table("inertia_verdict_history").Where("patient_id = ?", "p-upsert").Count(&count)
	if count != 1 {
		t.Errorf("row count = %d, want 1 (upsert, not duplicate)", count)
	}

	// Latest should be the second save's data.
	got, _, ok := history.FetchLatest("p-upsert")
	if !ok {
		t.Fatal("expected ok=true after upsert")
	}
	if len(got.Verdicts) != 1 || got.Verdicts[0].Domain != models.DomainHemodynamic {
		t.Errorf("expected hemodynamic verdict after upsert, got %+v", got.Verdicts)
	}
}

// TestPostgresInertiaHistory_FetchLatest_ReturnsNewestWeek verifies
// that FetchLatest returns the most recent week's verdict when
// multiple weeks have been saved.
func TestPostgresInertiaHistory_FetchLatest_ReturnsNewestWeek(t *testing.T) {
	db := setupInertiaHistoryTestDB(t)
	history := NewPostgresInertiaHistory(db, zap.NewNop())

	week1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	week2 := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)

	_ = history.SaveVerdict("p-multi", week1, models.PatientInertiaReport{
		Verdicts: []models.InertiaVerdict{{Domain: models.DomainGlycaemic}},
	})
	_ = history.SaveVerdict("p-multi", week2, models.PatientInertiaReport{
		Verdicts: []models.InertiaVerdict{{Domain: models.DomainHemodynamic}},
	})

	got, weekStart, ok := history.FetchLatest("p-multi")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !weekStart.Equal(week2) {
		t.Errorf("weekStart = %v, want week2 %v", weekStart, week2)
	}
	if got.Verdicts[0].Domain != models.DomainHemodynamic {
		t.Errorf("expected newest week's verdict (hemodynamic), got %+v", got.Verdicts)
	}
}

// TestPostgresInertiaHistory_FetchLatest_NoHistory verifies that a
// patient with no saved verdicts returns (zero, zero, false).
func TestPostgresInertiaHistory_FetchLatest_NoHistory(t *testing.T) {
	db := setupInertiaHistoryTestDB(t)
	history := NewPostgresInertiaHistory(db, zap.NewNop())

	_, _, ok := history.FetchLatest("p-new")
	if ok {
		t.Error("expected ok=false for patient with no history")
	}
}
