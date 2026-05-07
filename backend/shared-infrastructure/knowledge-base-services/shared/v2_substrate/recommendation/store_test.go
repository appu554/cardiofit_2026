package recommendation

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// testDB returns a connection to the local Docker Postgres for integration
// tests. Tests skip if VAIDSHALA_TEST_DSN is unset, so `go test ./...` still
// passes in CI environments without a database.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN unset; skipping DB integration test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return db
}

func TestPostgresStore_CreateAndGet(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	medUse1 := uuid.New()
	medUse2 := uuid.New()
	rec := models.Recommendation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		AuthorID:   uuid.New(),
		State:      models.RecommendationStateDrafted,
		Type:       models.RecommendationTypeStop,
		Urgency:    models.RecommendationUrgencyAmber,
		Title:      "Cease oxybutynin",
		ClinicalContent: models.ClinicalContent{
			Issue:           "anticholinergic burden",
			ClinicalContext: "87yo eGFR 32",
			Rationale:       "DBI 0.8 attributable",
			EvidenceRefs:    []string{"AMH-2024", "ADG-2025-Rec-42"},
			ProposedPlan:    "cease oxybutynin 5mg BD",
			MonitoringPlan:  "voiding diary 14d",
		},
		MedicineUseRefs: []uuid.UUID{medUse1, medUse2},
		ConsentRequired: true,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	if err := store.Create(ctx, &rec); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM recommendations WHERE id = $1", rec.ID)
	})

	got, err := store.Get(ctx, rec.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != rec.ID {
		t.Errorf("ID mismatch: got %v want %v", got.ID, rec.ID)
	}
	if got.State != rec.State || got.Type != rec.Type || got.Urgency != rec.Urgency {
		t.Errorf("state/type/urgency mismatch: got %+v", got)
	}
	if got.Title != rec.Title {
		t.Errorf("title mismatch: %q vs %q", got.Title, rec.Title)
	}
	if got.ClinicalContent.Issue != rec.ClinicalContent.Issue ||
		got.ClinicalContent.ProposedPlan != rec.ClinicalContent.ProposedPlan ||
		got.ClinicalContent.MonitoringPlan != rec.ClinicalContent.MonitoringPlan {
		t.Errorf("clinical content lost: %+v", got.ClinicalContent)
	}
	if len(got.ClinicalContent.EvidenceRefs) != 2 {
		t.Errorf("evidence refs lost: %v", got.ClinicalContent.EvidenceRefs)
	}
	if len(got.MedicineUseRefs) != 2 {
		t.Fatalf("medicine_use_refs round-trip failed: got %v", got.MedicineUseRefs)
	}
	if got.MedicineUseRefs[0] != medUse1 || got.MedicineUseRefs[1] != medUse2 {
		t.Errorf("medicine_use_refs values mismatch: got %v want [%v %v]",
			got.MedicineUseRefs, medUse1, medUse2)
	}
	if !got.ConsentRequired {
		t.Errorf("consent_required not preserved")
	}
}

func TestPostgresStore_CreateWithNilMedicineUseRefs(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	rec := models.Recommendation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		AuthorID:   uuid.New(),
		State:      models.RecommendationStateDetected,
		Type:       models.RecommendationTypeMonitor,
		Urgency:    models.RecommendationUrgencyGreen,
		Title:      "monitor only",
		ClinicalContent: models.ClinicalContent{Issue: "x", ProposedPlan: "y"},
		// MedicineUseRefs intentionally nil to exercise the nil-coercion path
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.Create(ctx, &rec); err != nil {
		t.Fatalf("create with nil MedicineUseRefs: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM recommendations WHERE id = $1", rec.ID)
	})

	got, err := store.Get(ctx, rec.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.MedicineUseRefs == nil {
		t.Errorf("expected empty slice (nil-coerced to '{}'), got nil")
	}
	if len(got.MedicineUseRefs) != 0 {
		t.Errorf("expected empty array; got %v", got.MedicineUseRefs)
	}
}
