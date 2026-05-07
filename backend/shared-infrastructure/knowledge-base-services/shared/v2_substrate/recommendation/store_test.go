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

	rec := models.Recommendation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		AuthorID:   uuid.New(),
		State:      models.RecommendationStateDrafted,
		Type:       models.RecommendationTypeStop,
		Urgency:    models.RecommendationUrgencyAmber,
		Title:      "Cease oxybutynin",
		ClinicalContent: models.ClinicalContent{
			Issue: "test", Rationale: "test", ProposedPlan: "test",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.Create(ctx, &rec); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := store.Get(ctx, rec.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != rec.ID || got.State != rec.State {
		t.Errorf("got %+v want %+v", got, rec)
	}
	if got.ClinicalContent.Issue != "test" {
		t.Errorf("clinical content not persisted: %+v", got.ClinicalContent)
	}

	// cleanup
	_, _ = db.ExecContext(ctx, "DELETE FROM recommendations WHERE id = $1", rec.ID)
}
