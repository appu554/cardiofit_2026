package contestation

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// testDB returns a live Postgres connection. Tests skip if VAIDSHALA_TEST_DSN
// is unset, so `go test ./...` passes without a database (CI environments).
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

// newContestation builds a valid Contestation fixture.
func newContestation(pharmacistID, employerID uuid.UUID) Contestation {
	return Contestation{
		ID:                 uuid.New(),
		PharmacistID:       pharmacistID,
		EmployerID:         employerID,
		KPIType:            "dispensing_accuracy",
		KPISnapshot:        map[string]any{"rate": 0.94, "period": "2026-Q1"},
		PharmacistArgument: "Near-miss was not a true dispensing error per clinical review.",
		Status:             StatusOpen,
		FiledAt:            time.Now().UTC(),
	}
}

// ===========================================================================
// InMemoryStore tests — no DB required, run in CI
// ===========================================================================

func TestInMemoryStore_CreateAndGet(t *testing.T) {
	store := &InMemoryStore{}
	ctx := context.Background()

	pharmID := uuid.New()
	empID := uuid.New()
	c := newContestation(pharmID, empID)

	got, err := store.Create(ctx, c)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != c.ID {
		t.Errorf("Create returned wrong ID: got %v want %v", got.ID, c.ID)
	}

	retrieved, err := store.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if retrieved.PharmacistArgument != c.PharmacistArgument {
		t.Errorf("PharmacistArgument mismatch: got %q want %q",
			retrieved.PharmacistArgument, c.PharmacistArgument)
	}
}

func TestInMemoryStore_ListByPharmacist_Filter(t *testing.T) {
	store := &InMemoryStore{}
	ctx := context.Background()

	pharmID := uuid.New()
	otherID := uuid.New()
	empID := uuid.New()

	c1 := newContestation(pharmID, empID)
	c2 := newContestation(pharmID, empID)
	c2.ID = uuid.New()
	c2.KPIType = "counselling_rate"
	c3 := newContestation(otherID, empID) // different pharmacist

	for _, c := range []Contestation{c1, c2, c3} {
		if _, err := store.Create(ctx, c); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	list, err := store.ListByPharmacist(ctx, pharmID)
	if err != nil {
		t.Fatalf("ListByPharmacist: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 records for pharmID, got %d", len(list))
	}

	// verify other pharmacist's record is absent
	for _, rec := range list {
		if rec.PharmacistID != pharmID {
			t.Errorf("got record for wrong pharmacist: %v", rec.PharmacistID)
		}
	}
}

func TestInMemoryStore_UpdateStatus_MovesToResolved(t *testing.T) {
	store := &InMemoryStore{}
	ctx := context.Background()

	pharmID := uuid.New()
	empID := uuid.New()
	c := newContestation(pharmID, empID)

	if _, err := store.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}

	response := "After review, the error classification was corrected."
	if err := store.UpdateStatus(ctx, c.ID, StatusResolved, response); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	updated, err := store.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get after UpdateStatus: %v", err)
	}
	if updated.Status != StatusResolved {
		t.Errorf("expected status %q, got %q", StatusResolved, updated.Status)
	}
	if updated.EmployerResponse != response {
		t.Errorf("expected response %q, got %q", response, updated.EmployerResponse)
	}
	if updated.ResolvedAt == nil {
		t.Error("expected ResolvedAt to be set, got nil")
	}
}

// ===========================================================================
// PostgresStore tests — skip without DSN
// ===========================================================================

func TestPostgresStore_CreateAndGet(t *testing.T) {
	db := testDB(t)
	t.Cleanup(func() { db.Close() })
	store := NewPostgresStore(db)
	ctx := context.Background()

	pharmID := uuid.New()
	empID := uuid.New()
	c := newContestation(pharmID, empID)

	got, err := store.Create(ctx, c)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != c.ID {
		t.Errorf("Create returned wrong ID: got %v want %v", got.ID, c.ID)
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM contestations WHERE id = $1", c.ID)
	})

	retrieved, err := store.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if retrieved.PharmacistArgument != c.PharmacistArgument {
		t.Errorf("PharmacistArgument mismatch: got %q want %q",
			retrieved.PharmacistArgument, c.PharmacistArgument)
	}
}

func TestPostgresStore_ListByPharmacist_Filter(t *testing.T) {
	db := testDB(t)
	t.Cleanup(func() { db.Close() })
	store := NewPostgresStore(db)
	ctx := context.Background()

	pharmID := uuid.New()
	otherID := uuid.New()
	empID := uuid.New()

	c1 := newContestation(pharmID, empID)
	c2 := newContestation(pharmID, empID)
	c2.ID = uuid.New()
	c2.KPIType = "counselling_rate"
	c3 := newContestation(otherID, empID)

	for _, c := range []Contestation{c1, c2, c3} {
		if _, err := store.Create(ctx, c); err != nil {
			t.Fatalf("Create: %v", err)
		}
		cCopy := c
		t.Cleanup(func() { db.Exec("DELETE FROM contestations WHERE id = $1", cCopy.ID) })
	}

	list, err := store.ListByPharmacist(ctx, pharmID)
	if err != nil {
		t.Fatalf("ListByPharmacist: %v", err)
	}
	if len(list) < 2 {
		t.Errorf("expected at least 2 records for pharmID, got %d", len(list))
	}
	for _, rec := range list {
		if rec.PharmacistID != pharmID {
			t.Errorf("got record for wrong pharmacist: %v", rec.PharmacistID)
		}
	}
}

func TestPostgresStore_UpdateStatus_MovesToResolved(t *testing.T) {
	db := testDB(t)
	t.Cleanup(func() { db.Close() })
	store := NewPostgresStore(db)
	ctx := context.Background()

	pharmID := uuid.New()
	empID := uuid.New()
	c := newContestation(pharmID, empID)

	if _, err := store.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { db.Exec("DELETE FROM contestations WHERE id = $1", c.ID) })

	response := "Error classification corrected after pharmacist review."
	if err := store.UpdateStatus(ctx, c.ID, StatusResolved, response); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	updated, err := store.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get after UpdateStatus: %v", err)
	}
	if updated.Status != StatusResolved {
		t.Errorf("expected status %q, got %q", StatusResolved, updated.Status)
	}
	if updated.ResolvedAt == nil {
		t.Error("expected ResolvedAt to be set, got nil")
	}
}
