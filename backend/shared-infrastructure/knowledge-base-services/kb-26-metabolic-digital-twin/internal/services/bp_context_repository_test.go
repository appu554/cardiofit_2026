package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/models"
)

func setupBPContextTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// AutoMigrate would fail because BPContextHistory uses gen_random_uuid() which
	// is PostgreSQL-specific. Create the table manually, matching the established
	// pattern used by relapse_detector_test.go and quarterly_aggregator_test.go.
	sqlDB, _ := db.DB()
	stmts := []string{
		`CREATE TABLE bp_context_histories (
			id            TEXT PRIMARY KEY,
			patient_id    TEXT NOT NULL,
			snapshot_date DATETIME NOT NULL,
			phenotype     TEXT NOT NULL,
			clinic_sbp_mean REAL,
			home_sbp_mean   REAL,
			gap_sbp         REAL,
			confidence    TEXT,
			created_at    DATETIME
		)`,
		// Unique index is required for the ON CONFLICT upsert in SaveSnapshot.
		`CREATE UNIQUE INDEX idx_bp_ctx_patient_date ON bp_context_histories(patient_id, snapshot_date)`,
	}
	for _, stmt := range stmts {
		if _, err := sqlDB.Exec(stmt); err != nil {
			t.Fatalf("DDL exec: %v", err)
		}
	}
	return db
}

func TestBPContextRepository_SaveAndFetchLatest(t *testing.T) {
	db := setupBPContextTestDB(t)
	repo := NewBPContextRepository(db)

	snapshot := &models.BPContextHistory{
		ID:            uuid.New().String(),
		PatientID:     "p1",
		SnapshotDate:  time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC),
		Phenotype:     models.PhenotypeMaskedHTN,
		ClinicSBPMean: 128,
		HomeSBPMean:   148,
		GapSBP:        -20,
		Confidence:    "HIGH",
	}

	if err := repo.SaveSnapshot(snapshot); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	latest, err := repo.FetchLatest("p1")
	if err != nil {
		t.Fatalf("FetchLatest failed: %v", err)
	}
	if latest == nil {
		t.Fatal("expected non-nil latest snapshot")
	}
	if latest.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN, got %s", latest.Phenotype)
	}
}

func TestBPContextRepository_SaveSnapshot_UpsertOnSameDay(t *testing.T) {
	db := setupBPContextTestDB(t)
	repo := NewBPContextRepository(db)

	day := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)

	first := &models.BPContextHistory{
		ID:           uuid.New().String(),
		PatientID:    "p1",
		SnapshotDate: day,
		Phenotype:    models.PhenotypeMaskedHTN,
		Confidence:   "HIGH",
	}
	if err := repo.SaveSnapshot(first); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	// Reclassification on the same day should upsert, not duplicate.
	second := &models.BPContextHistory{
		ID:           uuid.New().String(),
		PatientID:    "p1",
		SnapshotDate: day,
		Phenotype:    models.PhenotypeMaskedUncontrolled,
		Confidence:   "HIGH",
	}
	if err := repo.SaveSnapshot(second); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	all, err := repo.FetchHistory("p1", 10)
	if err != nil {
		t.Fatalf("FetchHistory failed: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 snapshot after upsert, got %d", len(all))
	}
	if all[0].Phenotype != models.PhenotypeMaskedUncontrolled {
		t.Errorf("expected upserted phenotype MASKED_UNCONTROLLED, got %s", all[0].Phenotype)
	}
}

func TestBPContextRepository_FetchLatest_NotFound(t *testing.T) {
	db := setupBPContextTestDB(t)
	repo := NewBPContextRepository(db)

	latest, err := repo.FetchLatest("unknown")
	if err != nil {
		t.Fatalf("FetchLatest should return nil snapshot, no error; got err=%v", err)
	}
	if latest != nil {
		t.Errorf("expected nil for unknown patient, got %+v", latest)
	}
}

func TestBPContextRepository_FetchHistory_OrderedDesc(t *testing.T) {
	db := setupBPContextTestDB(t)
	repo := NewBPContextRepository(db)

	dates := []time.Time{
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC),
	}
	for _, d := range dates {
		if err := repo.SaveSnapshot(&models.BPContextHistory{
			ID:           uuid.New().String(),
			PatientID:    "p1",
			SnapshotDate: d,
			Phenotype:    models.PhenotypeSustainedHTN,
			Confidence:   "HIGH",
		}); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	history, err := repo.FetchHistory("p1", 10)
	if err != nil {
		t.Fatalf("FetchHistory failed: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 history rows, got %d", len(history))
	}
	if !history[0].SnapshotDate.Equal(dates[2]) {
		t.Errorf("expected newest first; got %v", history[0].SnapshotDate)
	}
}
