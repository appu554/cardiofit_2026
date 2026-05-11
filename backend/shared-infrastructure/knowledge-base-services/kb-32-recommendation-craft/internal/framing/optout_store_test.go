package framing

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Compile-time conformance — caught at test-discovery time.
var (
	_ OptOutStore = (*InMemoryOptOutStore)(nil)
	_ OptOutStore = (*PostgresOptOutStore)(nil)
)

// ---------------------------------------------------------------------------
// InMemoryOptOutStore tests
// ---------------------------------------------------------------------------

func TestInMemoryOptOutStore_RegisterThenIsOptedOut(t *testing.T) {
	t.Parallel()
	s := NewInMemoryOptOutStore()
	ctx := context.Background()
	gp := uuid.New()

	if err := s.RegisterOptOut(ctx, gp, "patient prefers no profiling"); err != nil {
		t.Fatalf("RegisterOptOut: %v", err)
	}
	got, err := s.IsOptedOut(ctx, gp)
	if err != nil {
		t.Fatalf("IsOptedOut: %v", err)
	}
	if !got {
		t.Errorf("expected IsOptedOut=true after Register, got false")
	}
}

func TestInMemoryOptOutStore_RevokeFlipsOff(t *testing.T) {
	t.Parallel()
	s := NewInMemoryOptOutStore()
	ctx := context.Background()
	gp := uuid.New()

	if err := s.RegisterOptOut(ctx, gp, ""); err != nil {
		t.Fatalf("RegisterOptOut: %v", err)
	}
	if err := s.RevokeOptOut(ctx, gp); err != nil {
		t.Fatalf("RevokeOptOut: %v", err)
	}
	got, err := s.IsOptedOut(ctx, gp)
	if err != nil {
		t.Fatalf("IsOptedOut: %v", err)
	}
	if got {
		t.Errorf("expected IsOptedOut=false after Revoke, got true")
	}
}

func TestInMemoryOptOutStore_RegisterTwiceIdempotent(t *testing.T) {
	t.Parallel()
	s := NewInMemoryOptOutStore()
	ctx := context.Background()
	gp := uuid.New()

	if err := s.RegisterOptOut(ctx, gp, "first"); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := s.RegisterOptOut(ctx, gp, "second"); err != nil {
		t.Fatalf("second Register: %v", err)
	}
	got, _ := s.IsOptedOut(ctx, gp)
	if !got {
		t.Errorf("expected still opted-out after double Register, got false")
	}
	// One logical record (PK on gp_id semantic — InMemory uses map).
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.records) != 1 {
		t.Errorf("expected exactly 1 stored record, got %d", len(s.records))
	}
}

func TestInMemoryOptOutStore_RevokeWhenNotOptedOut_IsNoOp(t *testing.T) {
	t.Parallel()
	s := NewInMemoryOptOutStore()
	ctx := context.Background()
	// Document the chosen semantic: revoke on a GP that never opted-out
	// returns nil (no error). The application layer relies on this so the
	// DELETE handler can return 204 unconditionally.
	if err := s.RevokeOptOut(ctx, uuid.New()); err != nil {
		t.Errorf("expected nil error on revoke-without-prior-register, got %v", err)
	}
}

func TestInMemoryOptOutStore_RegisterAfterRevokeReactivates(t *testing.T) {
	t.Parallel()
	s := NewInMemoryOptOutStore()
	ctx := context.Background()
	gp := uuid.New()

	if err := s.RegisterOptOut(ctx, gp, ""); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := s.RevokeOptOut(ctx, gp); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if err := s.RegisterOptOut(ctx, gp, "changed mind"); err != nil {
		t.Fatalf("re-Register: %v", err)
	}
	got, _ := s.IsOptedOut(ctx, gp)
	if !got {
		t.Errorf("expected re-Register to reactivate opt-out, got IsOptedOut=false")
	}
}

func TestInMemoryOptOutStore_ConcurrentSafety(t *testing.T) {
	t.Parallel()
	s := NewInMemoryOptOutStore()
	ctx := context.Background()

	const goroutines = 32
	gps := make([]uuid.UUID, goroutines)
	for i := range gps {
		gps[i] = uuid.New()
	}

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			_ = s.RegisterOptOut(ctx, gps[i], "")
		}()
		go func() {
			defer wg.Done()
			_ = s.RevokeOptOut(ctx, gps[i])
		}()
		go func() {
			defer wg.Done()
			_, _ = s.IsOptedOut(ctx, gps[i])
		}()
	}
	wg.Wait()
	// No assertion on final state — the test would fail under -race if the
	// store were not safe for concurrent use. State is intentionally
	// indeterminate after racing Register/Revoke goroutines.
}

// ---------------------------------------------------------------------------
// PostgresOptOutStore tests
// ---------------------------------------------------------------------------

// TestPostgresOptOutStore_InterfaceConformance is a unit-level guard that
// does not require a live database.
func TestPostgresOptOutStore_InterfaceConformance(t *testing.T) {
	t.Parallel()
	store := NewPostgresOptOutStore(nil)
	if store == nil {
		t.Fatal("NewPostgresOptOutStore(nil) returned nil")
	}
	var _ OptOutStore = store
}

// TestPostgresOptOutStore_RoundTrip exercises register → revoke → re-register
// against a live Postgres with migration 047 applied. Skipped when
// VAIDSHALA_TEST_DSN is unset so the suite stays green without a DB.
func TestPostgresOptOutStore_RoundTrip(t *testing.T) {
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN not set; skipping Postgres integration test")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("db.Ping: %v", err)
	}

	store := NewPostgresOptOutStore(db)
	gp := uuid.New()

	// Best-effort cleanup so a re-run is idempotent.
	cleanup := func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM prescriber_framing_optout WHERE gp_id = $1`, gp)
	}
	cleanup()
	defer cleanup()

	// Register → IsOptedOut true.
	if err := store.RegisterOptOut(ctx, gp, "phase-2-completion test"); err != nil {
		t.Fatalf("RegisterOptOut: %v", err)
	}
	got, err := store.IsOptedOut(ctx, gp)
	if err != nil {
		t.Fatalf("IsOptedOut: %v", err)
	}
	if !got {
		t.Errorf("expected IsOptedOut=true after Register, got false")
	}

	// Register again → idempotent, still true, exactly one row.
	if err := store.RegisterOptOut(ctx, gp, "second call"); err != nil {
		t.Fatalf("second Register: %v", err)
	}
	var rowCount int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM prescriber_framing_optout WHERE gp_id = $1`, gp,
	).Scan(&rowCount); err != nil {
		t.Fatalf("row count: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("expected exactly 1 row after double Register, got %d", rowCount)
	}

	// Revoke → IsOptedOut false.
	if err := store.RevokeOptOut(ctx, gp); err != nil {
		t.Fatalf("RevokeOptOut: %v", err)
	}
	got, _ = store.IsOptedOut(ctx, gp)
	if got {
		t.Errorf("expected IsOptedOut=false after Revoke, got true")
	}

	// Revoke again (already revoked) → no error.
	if err := store.RevokeOptOut(ctx, gp); err != nil {
		t.Errorf("expected nil on double Revoke, got %v", err)
	}

	// Register after revoke → reactivates (revoked_at back to NULL).
	if err := store.RegisterOptOut(ctx, gp, "reactivated"); err != nil {
		t.Fatalf("re-Register: %v", err)
	}
	got, _ = store.IsOptedOut(ctx, gp)
	if !got {
		t.Errorf("expected re-Register to reactivate opt-out, got IsOptedOut=false")
	}
}

func TestPostgresOptOutStore_RevokeWhenAbsent(t *testing.T) {
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN not set; skipping Postgres integration test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store := NewPostgresOptOutStore(db)
	gp := uuid.New()
	// No prior register. Revoke should return nil (no-op).
	if err := store.RevokeOptOut(ctx, gp); err != nil {
		t.Errorf("expected nil on revoke-without-register, got %v", err)
	}
}
