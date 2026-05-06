package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/scoring"
)

func openTestScoringStore(t *testing.T) (*ScoringStore, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping DB-gated scoring store test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	v2 := NewV2SubstrateStoreWithDB(db)
	return NewScoringStore(db, v2), db
}

func validCFSScoreFixture(rid uuid.UUID, score int) models.CFSScore {
	return models.CFSScore{
		ResidentRef:       rid,
		AssessedAt:        time.Now().UTC().Truncate(time.Second),
		AssessorRoleRef:   uuid.New(),
		InstrumentVersion: scoring.CFSInstrumentVersionCurrent,
		Score:             score,
	}
}

func validAKPSScoreFixture(rid uuid.UUID, score int) models.AKPSScore {
	return models.AKPSScore{
		ResidentRef:       rid,
		AssessedAt:        time.Now().UTC().Truncate(time.Second),
		AssessorRoleRef:   uuid.New(),
		InstrumentVersion: scoring.AKPSInstrumentVersionCurrent,
		Score:             score,
	}
}

func TestScoringStore_CreateCFSPersistsAndCurrentReflects(t *testing.T) {
	s, db := openTestScoringStore(t)
	defer db.Close()
	rid := uuid.New()
	out, err := s.CreateCFSScore(context.Background(), validCFSScoreFixture(rid, 5))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.CFSScore == nil || out.CFSScore.Score != 5 {
		t.Fatalf("CFSScore drift: %+v", out.CFSScore)
	}
	if out.CareIntensityHint != nil {
		t.Errorf("expected no hint for CFS=5; got %+v", out.CareIntensityHint)
	}
	if out.EvidenceTraceNodeRef == uuid.Nil {
		t.Errorf("expected EvidenceTraceNodeRef set")
	}
	cur, err := s.GetCurrentCFSScore(context.Background(), rid)
	if err != nil {
		t.Fatalf("GetCurrent: %v", err)
	}
	if cur.ID != out.CFSScore.ID {
		t.Errorf("current ID drift")
	}
}

func TestScoringStore_CFSHighScoreEmitsHintWithoutCareIntensityRow(t *testing.T) {
	s, db := openTestScoringStore(t)
	defer db.Close()
	rid := uuid.New()

	out, err := s.CreateCFSScore(context.Background(), validCFSScoreFixture(rid, 7))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.CareIntensityHint == nil {
		t.Fatalf("expected hint for CFS=7")
	}
	if out.CareIntensityHint.Instrument != "CFS" {
		t.Errorf("hint instrument drift: %s", out.CareIntensityHint.Instrument)
	}
	if out.CareIntensityHint.Score != 7 {
		t.Errorf("hint score drift: %d", out.CareIntensityHint.Score)
	}

	// Critical safety acceptance: substrate must NOT have written a
	// care_intensity_history row.
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM care_intensity_history WHERE resident_ref = $1`, rid).Scan(&count); err != nil {
		t.Fatalf("count care_intensity: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 care_intensity_history rows for CFS hint; got %d (substrate must NEVER auto-transition)", count)
	}
}

func TestScoringStore_CFSScoreSixDoesNotHint(t *testing.T) {
	s, db := openTestScoringStore(t)
	defer db.Close()
	rid := uuid.New()
	out, err := s.CreateCFSScore(context.Background(), validCFSScoreFixture(rid, 6))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.CareIntensityHint != nil {
		t.Errorf("expected no hint for CFS=6 (boundary just below threshold); got %+v", out.CareIntensityHint)
	}
}

func TestScoringStore_AKPSLowScoreEmitsHint(t *testing.T) {
	s, db := openTestScoringStore(t)
	defer db.Close()
	rid := uuid.New()
	out, err := s.CreateAKPSScore(context.Background(), validAKPSScoreFixture(rid, 40))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.CareIntensityHint == nil {
		t.Fatalf("expected hint for AKPS=40")
	}
	if out.CareIntensityHint.Instrument != "AKPS" {
		t.Errorf("hint instrument drift: %s", out.CareIntensityHint.Instrument)
	}
}

func TestScoringStore_AKPSScoreFiftyDoesNotHint(t *testing.T) {
	s, db := openTestScoringStore(t)
	defer db.Close()
	rid := uuid.New()
	out, err := s.CreateAKPSScore(context.Background(), validAKPSScoreFixture(rid, 50))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.CareIntensityHint != nil {
		t.Errorf("expected no hint for AKPS=50; got %+v", out.CareIntensityHint)
	}
}

func TestScoringStore_HistoryDescending(t *testing.T) {
	s, db := openTestScoringStore(t)
	defer db.Close()
	rid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	for _, at := range []time.Time{now.Add(-72 * time.Hour), now.Add(-24 * time.Hour), now} {
		c := validCFSScoreFixture(rid, 3)
		c.AssessedAt = at
		if _, err := s.CreateCFSScore(context.Background(), c); err != nil {
			t.Fatalf("seed at %v: %v", at, err)
		}
	}
	hist, err := s.ListCFSHistory(context.Background(), rid)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(hist) != 3 {
		t.Fatalf("expected 3 rows; got %d", len(hist))
	}
	if !hist[0].AssessedAt.Equal(now) {
		t.Errorf("expected newest first; got %v", hist[0].AssessedAt)
	}
}

func TestScoringStore_GetCurrentNotFound(t *testing.T) {
	s, db := openTestScoringStore(t)
	defer db.Close()
	_, err := s.GetCurrentCFSScore(context.Background(), uuid.New())
	if !errors.Is(err, interfaces.ErrNotFound) {
		t.Errorf("expected ErrNotFound; got %v", err)
	}
}

func TestScoringStore_RejectsInvalidCFS(t *testing.T) {
	s, db := openTestScoringStore(t)
	defer db.Close()
	bad := validCFSScoreFixture(uuid.New(), 0) // out of range
	if _, err := s.CreateCFSScore(context.Background(), bad); err == nil {
		t.Errorf("expected error for score=0")
	}
}

func TestScoringStore_RejectsInvalidAKPS(t *testing.T) {
	s, db := openTestScoringStore(t)
	defer db.Close()
	bad := validAKPSScoreFixture(uuid.New(), 35) // not multiple of 10
	if _, err := s.CreateAKPSScore(context.Background(), bad); err == nil {
		t.Errorf("expected error for score=35 (not multiple of 10)")
	}
}

// staticLookupForRecompute returns a predictable lookup for the
// recompute integration test that doesn't depend on the live seed table.
func staticLookupForRecompute() scoring.DrugWeightLookup {
	return scoring.NewStaticDrugWeightLookup(map[string]scoring.DrugWeight{
		"amitriptyline": {DrugName: "amitriptyline", AnticholinergicWeight: 0.5, SedativeWeight: 0.5, ACBWeight: 3},
		"temazepam":     {DrugName: "temazepam", AnticholinergicWeight: 0.0, SedativeWeight: 0.5, ACBWeight: 1},
	})
}

func TestScoringStore_RecomputeDrugBurdenWritesNewRows(t *testing.T) {
	s, db := openTestScoringStore(t)
	defer db.Close()
	s = s.WithDrugWeightLookup(staticLookupForRecompute())
	rid := uuid.New()

	// Seed two active medications via V2SubstrateStore.UpsertMedicineUse
	// — the same write path the REST handler uses.
	med1 := models.MedicineUse{
		ID:          uuid.New(),
		ResidentID:  rid,
		DisplayName: "Amitriptyline 25mg",
		Status:      models.MedicineUseStatusActive,
		StartedAt:   time.Now().UTC(),
		Intent:      models.Intent{Category: models.IntentTherapeutic},
		Target:      models.Target{Kind: models.TargetKindOpen},
	}
	med2 := models.MedicineUse{
		ID:          uuid.New(),
		ResidentID:  rid,
		DisplayName: "Temazepam 10mg",
		Status:      models.MedicineUseStatusActive,
		StartedAt:   time.Now().UTC(),
		Intent:      models.Intent{Category: models.IntentSymptomatic},
		Target:      models.Target{Kind: models.TargetKindOpen},
	}
	if _, err := s.v2.UpsertMedicineUse(context.Background(), med1); err != nil {
		t.Fatalf("seed med1: %v", err)
	}
	if _, err := s.v2.UpsertMedicineUse(context.Background(), med2); err != nil {
		t.Fatalf("seed med2: %v", err)
	}

	out, err := s.RecomputeDrugBurden(context.Background(), rid)
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if out.DBIScore == nil {
		t.Fatalf("expected DBIScore set")
	}
	// amitriptyline (0.5+0.5) + temazepam (0+0.5) = 1.5
	if out.DBIScore.Score != 1.5 {
		t.Errorf("DBI Score: got %v want 1.5", out.DBIScore.Score)
	}
	if out.DBIScore.AnticholinergicComponent != 0.5 {
		t.Errorf("DBI ach: got %v want 0.5", out.DBIScore.AnticholinergicComponent)
	}
	if out.DBIScore.SedativeComponent != 1.0 {
		t.Errorf("DBI sed: got %v want 1.0", out.DBIScore.SedativeComponent)
	}
	if len(out.DBIScore.ComputationInputs) != 2 {
		t.Errorf("DBI inputs: got %d want 2", len(out.DBIScore.ComputationInputs))
	}
	// amitriptyline=3 + temazepam=1 = 4
	if out.ACBScore == nil {
		t.Fatalf("expected ACBScore set")
	}
	if out.ACBScore.Score != 4 {
		t.Errorf("ACB Score: got %d want 4", out.ACBScore.Score)
	}

	// Re-running the recompute writes a NEW row in each table (history
	// is append-only).
	if _, err := s.RecomputeDrugBurden(context.Background(), rid); err != nil {
		t.Fatalf("Recompute again: %v", err)
	}
	hist, err := s.ListDBIHistory(context.Background(), rid)
	if err != nil {
		t.Fatalf("ListDBIHistory: %v", err)
	}
	if len(hist) != 2 {
		t.Errorf("expected 2 dbi_scores rows after 2 recomputes; got %d", len(hist))
	}
}

func TestScoringStore_CurrentScoresAggregatesPresentAndAbsent(t *testing.T) {
	s, db := openTestScoringStore(t)
	defer db.Close()
	rid := uuid.New()
	// Only CFS recorded — others should come back nil.
	if _, err := s.CreateCFSScore(context.Background(), validCFSScoreFixture(rid, 4)); err != nil {
		t.Fatalf("Create CFS: %v", err)
	}
	out, err := s.CurrentScoresByResident(context.Background(), rid)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if out.CFS == nil {
		t.Errorf("expected CFS set")
	}
	if out.AKPS != nil {
		t.Errorf("expected AKPS nil")
	}
	if out.DBI != nil {
		t.Errorf("expected DBI nil")
	}
	if out.ACB != nil {
		t.Errorf("expected ACB nil")
	}
}

func TestSeedDrugWeightLookup_KnownDrugReturnsWeights(t *testing.T) {
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping DB-gated lookup test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	l := NewSeedDrugWeightLookup(db)
	// Spot check 3 known drugs from the migration 018 seed.
	for _, name := range []string{"Amitriptyline 25mg", "Temazepam 10mg", "Quetiapine 25mg"} {
		w, found, err := l.Lookup(context.Background(), name)
		if err != nil {
			t.Fatalf("Lookup(%q): %v", name, err)
		}
		if !found {
			t.Errorf("Lookup(%q): expected match in seed table", name)
		}
		if w.DrugName == "" {
			t.Errorf("Lookup(%q): empty DrugName", name)
		}
	}
	// And one unknown drug.
	_, found, err := l.Lookup(context.Background(), "ParacetamolNonExistent")
	if err != nil {
		t.Fatalf("Lookup unknown: %v", err)
	}
	if found {
		t.Errorf("expected no match for unknown drug")
	}
}

func TestSeedDrugWeightLookup_SeedTableRowCount(t *testing.T) {
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping DB-gated count test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	var dbiCount, acbCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM dbi_drug_weights`).Scan(&dbiCount); err != nil {
		t.Fatalf("count dbi: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM acb_drug_weights`).Scan(&acbCount); err != nil {
		t.Fatalf("count acb: %v", err)
	}
	if dbiCount < 20 {
		t.Errorf("dbi_drug_weights: expected >=20 seed rows; got %d", dbiCount)
	}
	if acbCount < 20 {
		t.Errorf("acb_drug_weights: expected >=20 seed rows; got %d", acbCount)
	}
}
