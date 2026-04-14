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
			raw_phenotype TEXT,
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

// setupTwinStateTable creates the twin_states table in the given SQLite DB using
// manual DDL. AutoMigrate is deliberately avoided because TwinState uses
// gen_random_uuid() (PostgreSQL-specific) and jsonb column types that SQLite
// cannot handle.
func setupTwinStateTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, _ := db.DB()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS twin_states (
			id            TEXT PRIMARY KEY,
			patient_id    TEXT NOT NULL,
			state_version INTEGER NOT NULL,
			update_source TEXT NOT NULL,
			updated_at    DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_twin_state_updated_at ON twin_states(updated_at DESC)`,
	}
	for _, stmt := range stmts {
		if _, err := sqlDB.Exec(stmt); err != nil {
			t.Fatalf("DDL exec: %v", err)
		}
	}
}

func TestBPContextRepository_ListActivePatientIDs(t *testing.T) {
	db := setupBPContextTestDB(t)
	setupTwinStateTable(t, db)

	now := time.Now().UTC()
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	type twinRow struct {
		ID           string
		PatientID    string
		StateVersion int
		UpdateSource string
		UpdatedAt    time.Time
	}
	seed := func(patientID uuid.UUID, daysAgo int) {
		row := twinRow{
			ID:           uuid.New().String(),
			PatientID:    patientID.String(),
			StateVersion: 1,
			UpdateSource: "TEST",
			UpdatedAt:    now.AddDate(0, 0, -daysAgo),
		}
		if err := db.Table("twin_states").Create(&row).Error; err != nil {
			t.Fatalf("seed patient %s: %v", patientID, err)
		}
	}

	seed(id1, 5)  // active: 5 days ago
	seed(id2, 1)  // active: 1 day ago
	seed(id3, 60) // inactive: 60 days ago

	repo := NewBPContextRepository(db)
	active, err := repo.ListActivePatientIDs(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("ListActivePatientIDs: %v", err)
	}

	if len(active) != 2 {
		t.Errorf("expected 2 active patients, got %d", len(active))
	}

	got := map[string]bool{}
	for _, p := range active {
		got[p] = true
	}
	if !got[id1.String()] || !got[id2.String()] {
		t.Errorf("expected active set to contain %s and %s, got %v", id1, id2, got)
	}
	if got[id3.String()] {
		t.Errorf("inactive patient %s should not be in result", id3)
	}
}

func TestBPContextRepository_FetchHistorySince_FiltersByTime(t *testing.T) {
	db := setupBPContextTestDB(t)
	repo := NewBPContextRepository(db)

	now := time.Now().UTC()

	// 3 snapshots: 40 days ago, 20 days ago, 5 days ago
	for i, daysAgo := range []int{40, 20, 5} {
		_ = i
		snapshot := &models.BPContextHistory{
			ID:           uuid.New().String(),
			PatientID:    "p1",
			SnapshotDate: now.AddDate(0, 0, -daysAgo),
			Phenotype:    models.PhenotypeSustainedHTN,
			Confidence:   "HIGH",
		}
		if err := repo.SaveSnapshot(snapshot); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	// Query last 30 days — should return 2 snapshots (the 20-day and 5-day rows)
	history, err := repo.FetchHistorySince("p1", now.AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("FetchHistorySince: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("expected 2 snapshots in last 30 days, got %d", len(history))
	}
	// Oldest first — first row should be the 20-day snapshot
	if len(history) >= 2 {
		diff := history[1].SnapshotDate.Sub(history[0].SnapshotDate)
		if diff < 0 {
			t.Errorf("expected oldest first, got newest first")
		}
	}
}

func TestBPContextRepository_ListActivePatientIDs_DeduplicatesMultipleSnapshots(t *testing.T) {
	db := setupBPContextTestDB(t)
	setupTwinStateTable(t, db)

	now := time.Now().UTC()
	patientID := uuid.New()

	type twinRow struct {
		ID           string
		PatientID    string
		StateVersion int
		UpdateSource string
		UpdatedAt    time.Time
	}

	// Three snapshots for the same patient — query must return one row.
	for i := 0; i < 3; i++ {
		row := twinRow{
			ID:           uuid.New().String(),
			PatientID:    patientID.String(),
			StateVersion: i + 1,
			UpdateSource: "TEST",
			UpdatedAt:    now.AddDate(0, 0, -i),
		}
		if err := db.Table("twin_states").Create(&row).Error; err != nil {
			t.Fatalf("seed snapshot %d: %v", i, err)
		}
	}

	repo := NewBPContextRepository(db)
	active, err := repo.ListActivePatientIDs(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("ListActivePatientIDs: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("expected 1 deduplicated patient, got %d", len(active))
	}
}
