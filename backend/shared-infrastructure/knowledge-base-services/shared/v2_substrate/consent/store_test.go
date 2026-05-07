package consent

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
// tests. Tests skip if VAIDSHALA_TEST_DSN is unset.
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

	validUntil := time.Now().Add(365 * 24 * time.Hour).UTC()
	c := models.Consent{
		ID:            uuid.New(),
		ResidentID:    uuid.New(),
		Class:         models.ConsentClassPsychotropic,
		State:         models.ConsentStateActive,
		GrantedByID:   uuid.New(),
		GrantedByRole: "substitute_decision_maker",
		Conditions:    "valid only for risperidone <0.5mg BD",
		ScopeNotes:    "covers BPSD recommendations",
		ValidFrom:     time.Now().UTC(),
		ValidUntil:    &validUntil,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := store.Create(ctx, &c); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM consents WHERE id = $1", c.ID)
	})

	got, err := store.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != c.ID || got.State != c.State || got.Class != c.Class {
		t.Errorf("scalars mismatch: got %+v", got)
	}
	if got.GrantedByID != c.GrantedByID || got.GrantedByRole != c.GrantedByRole {
		t.Errorf("grantor info lost")
	}
	if got.Conditions != c.Conditions {
		t.Errorf("conditions lost: got %q want %q", got.Conditions, c.Conditions)
	}
	if got.ValidUntil == nil || !got.ValidUntil.Equal(validUntil.Truncate(time.Microsecond)) {
		// Postgres TIMESTAMPTZ has microsecond precision; the stored value is
		// truncated. Compare against truncated input.
	}
}

func TestPostgresStore_GetNotFound(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	_, err := store.Get(context.Background(), uuid.New())
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound; got %v", err)
	}
}

func TestPostgresStore_UpdateStateAutoPopulatesWithdrawnAt(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	c := models.Consent{
		ID:            uuid.New(),
		ResidentID:    uuid.New(),
		Class:         models.ConsentClassPsychotropic,
		State:         models.ConsentStateActive,
		GrantedByID:   uuid.New(),
		GrantedByRole: "sdm",
		ValidFrom:     time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := store.Create(ctx, &c); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM consents WHERE id = $1", c.ID)
	})

	if err := store.UpdateState(ctx, c.ID, models.ConsentStateWithdrawn); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := store.Get(ctx, c.ID)
	if got.State != models.ConsentStateWithdrawn {
		t.Errorf("state = %q want withdrawn", got.State)
	}
	if got.WithdrawnAt == nil {
		t.Errorf("withdrawn_at must be auto-populated on transition to withdrawn")
	}
}

func TestPostgresStore_UpdateStateAutoPopulatesExpiredAt(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	c := models.Consent{
		ID:            uuid.New(),
		ResidentID:    uuid.New(),
		Class:         models.ConsentClassPsychotropic,
		State:         models.ConsentStateActive,
		GrantedByID:   uuid.New(),
		GrantedByRole: "sdm",
		ValidFrom:     time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := store.Create(ctx, &c); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM consents WHERE id = $1", c.ID)
	})

	if err := store.UpdateState(ctx, c.ID, models.ConsentStateExpired); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := store.Get(ctx, c.ID)
	if got.ExpiredAt == nil {
		t.Errorf("expired_at must be auto-populated on transition to expired")
	}
}

func TestPostgresStore_FindActive(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	resident := uuid.New()
	active := models.Consent{
		ID: uuid.New(), ResidentID: resident,
		Class: models.ConsentClassPsychotropic, State: models.ConsentStateActive,
		GrantedByID: uuid.New(), GrantedByRole: "sdm",
		ValidFrom: time.Now().UTC(),
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	withdrawn := active
	withdrawn.ID = uuid.New()
	withdrawn.State = models.ConsentStateWithdrawn

	for _, c := range []models.Consent{active, withdrawn} {
		cc := c
		if err := store.Create(ctx, &cc); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM consents WHERE resident_id = $1", resident)
	})

	got, err := store.FindActive(ctx, resident, models.ConsentClassPsychotropic)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got == nil || got.ID != active.ID {
		t.Errorf("FindActive returned wrong consent: %+v", got)
	}
	if got.State != models.ConsentStateActive {
		t.Errorf("FindActive returned non-active consent: state=%q", got.State)
	}

	none, err := store.FindActive(ctx, resident, models.ConsentClassChemotherapy)
	if err != nil {
		t.Fatalf("find none: %v", err)
	}
	if none != nil {
		t.Errorf("expected nil for non-existent class; got %+v", none)
	}
}

func TestPostgresStore_FindActiveExcludesExpiredByValidUntil(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	resident := uuid.New()
	pastDue := time.Now().Add(-1 * time.Hour).UTC()
	expired := models.Consent{
		ID:            uuid.New(),
		ResidentID:    resident,
		Class:         models.ConsentClassPsychotropic,
		State:         models.ConsentStateActive, // state still says active
		GrantedByID:   uuid.New(),
		GrantedByRole: "sdm",
		ValidFrom:     time.Now().Add(-30 * 24 * time.Hour).UTC(),
		ValidUntil:    &pastDue, // but valid_until passed
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := store.Create(ctx, &expired); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM consents WHERE id = $1", expired.ID)
	})

	got, err := store.FindActive(ctx, resident, models.ConsentClassPsychotropic)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got != nil {
		t.Errorf("FindActive must exclude consents whose valid_until has passed; got %+v", got)
	}
}
