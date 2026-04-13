package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-patient-profile/internal/models"
)

func setupBPReadingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// LabEntry has PostgreSQL-specific defaults; use raw DDL for tests.
	err = db.Exec(`
		CREATE TABLE lab_entries (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			lab_type TEXT NOT NULL,
			value TEXT NOT NULL,
			unit TEXT NOT NULL,
			measured_at DATETIME NOT NULL,
			source TEXT,
			is_derived INTEGER DEFAULT 0,
			validation_status TEXT NOT NULL DEFAULT 'ACCEPTED',
			flag_reason TEXT,
			loinc_code TEXT,
			fhir_observation_id TEXT,
			created_at DATETIME
		)
	`).Error
	if err != nil {
		t.Fatalf("create lab_entries table: %v", err)
	}
	return db
}

func seedLabEntry(t *testing.T, db *gorm.DB, patientID, labType string, value float64, measuredAt time.Time, source string) {
	t.Helper()
	entry := map[string]interface{}{
		"id":                uuid.New().String(),
		"patient_id":        patientID,
		"lab_type":          labType,
		"value":             decimal.NewFromFloat(value).String(),
		"unit":              "mmHg",
		"measured_at":       measuredAt,
		"source":            source,
		"validation_status": "ACCEPTED",
	}
	if err := db.Table("lab_entries").Create(entry).Error; err != nil {
		t.Fatalf("seed lab_entry %s: %v", labType, err)
	}
}

func TestBPReadingQuery_PairsSBPAndDBPWithinFiveMinutes(t *testing.T) {
	db := setupBPReadingTestDB(t)
	query := NewBPReadingQuery(db)

	now := time.Date(2026, 4, 11, 9, 0, 0, 0, time.UTC)
	seedLabEntry(t, db, "p1", models.LabTypeSBP, 142, now, "HOME_CUFF")
	seedLabEntry(t, db, "p1", models.LabTypeDBP, 88, now.Add(30*time.Second), "HOME_CUFF")

	readings, err := query.FetchSince("p1", now.AddDate(0, 0, -1))
	if err != nil {
		t.Fatalf("FetchSince: %v", err)
	}
	if len(readings) != 1 {
		t.Fatalf("expected 1 paired reading, got %d", len(readings))
	}
	if readings[0].SBP != 142 || readings[0].DBP != 88 {
		t.Errorf("expected 142/88, got %.0f/%.0f", readings[0].SBP, readings[0].DBP)
	}
	if readings[0].Source != "HOME_CUFF" {
		t.Errorf("expected HOME_CUFF source, got %s", readings[0].Source)
	}
}

func TestBPReadingQuery_DropsUnpairedSBPWithoutDBPWithinWindow(t *testing.T) {
	db := setupBPReadingTestDB(t)
	query := NewBPReadingQuery(db)

	now := time.Date(2026, 4, 11, 9, 0, 0, 0, time.UTC)
	seedLabEntry(t, db, "p1", models.LabTypeSBP, 142, now, "HOME_CUFF")
	// DBP is 10 minutes later — outside the 5-min pairing window
	seedLabEntry(t, db, "p1", models.LabTypeDBP, 88, now.Add(10*time.Minute), "HOME_CUFF")

	readings, err := query.FetchSince("p1", now.AddDate(0, 0, -1))
	if err != nil {
		t.Fatalf("FetchSince: %v", err)
	}
	if len(readings) != 0 {
		t.Errorf("expected 0 paired readings (outside window), got %d", len(readings))
	}
}

func TestBPReadingQuery_DoesNotCrossPairDifferentSources(t *testing.T) {
	db := setupBPReadingTestDB(t)
	query := NewBPReadingQuery(db)

	now := time.Date(2026, 4, 11, 9, 0, 0, 0, time.UTC)
	seedLabEntry(t, db, "p1", models.LabTypeSBP, 142, now, "HOME_CUFF")
	seedLabEntry(t, db, "p1", models.LabTypeDBP, 88, now.Add(30*time.Second), "CLINIC")

	readings, err := query.FetchSince("p1", now.AddDate(0, 0, -1))
	if err != nil {
		t.Fatalf("FetchSince: %v", err)
	}
	if len(readings) != 0 {
		t.Errorf("expected 0 paired readings (different sources), got %d", len(readings))
	}
}

func TestBPReadingQuery_FiltersByPatientID(t *testing.T) {
	db := setupBPReadingTestDB(t)
	query := NewBPReadingQuery(db)

	now := time.Date(2026, 4, 11, 9, 0, 0, 0, time.UTC)
	seedLabEntry(t, db, "p1", models.LabTypeSBP, 142, now, "HOME_CUFF")
	seedLabEntry(t, db, "p1", models.LabTypeDBP, 88, now.Add(30*time.Second), "HOME_CUFF")
	seedLabEntry(t, db, "p2", models.LabTypeSBP, 150, now, "HOME_CUFF")
	seedLabEntry(t, db, "p2", models.LabTypeDBP, 95, now.Add(30*time.Second), "HOME_CUFF")

	readings, err := query.FetchSince("p1", now.AddDate(0, 0, -1))
	if err != nil {
		t.Fatalf("FetchSince: %v", err)
	}
	if len(readings) != 1 {
		t.Fatalf("expected 1 reading for p1, got %d", len(readings))
	}
	if readings[0].PatientID != "p1" {
		t.Errorf("expected patient p1, got %s", readings[0].PatientID)
	}
}

func TestBPReadingQuery_FiltersByTimeWindow(t *testing.T) {
	db := setupBPReadingTestDB(t)
	query := NewBPReadingQuery(db)

	now := time.Date(2026, 4, 11, 9, 0, 0, 0, time.UTC)
	oneWeekAgo := now.AddDate(0, 0, -7)
	twoMonthsAgo := now.AddDate(0, -2, 0)

	seedLabEntry(t, db, "p1", models.LabTypeSBP, 142, oneWeekAgo, "HOME_CUFF")
	seedLabEntry(t, db, "p1", models.LabTypeDBP, 88, oneWeekAgo.Add(30*time.Second), "HOME_CUFF")
	seedLabEntry(t, db, "p1", models.LabTypeSBP, 150, twoMonthsAgo, "HOME_CUFF")
	seedLabEntry(t, db, "p1", models.LabTypeDBP, 95, twoMonthsAgo.Add(30*time.Second), "HOME_CUFF")

	// Query last 30 days — should return only the recent paired reading
	readings, err := query.FetchSince("p1", now.AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("FetchSince: %v", err)
	}
	if len(readings) != 1 {
		t.Fatalf("expected 1 reading in last 30 days, got %d", len(readings))
	}
}

func TestBPReadingQuery_EmptyPatient_ReturnsEmptySlice(t *testing.T) {
	db := setupBPReadingTestDB(t)
	query := NewBPReadingQuery(db)

	readings, err := query.FetchSince("unknown", time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Errorf("unknown patient should return empty slice, not error: %v", err)
	}
	if len(readings) != 0 {
		t.Errorf("expected empty slice, got %d readings", len(readings))
	}
}
