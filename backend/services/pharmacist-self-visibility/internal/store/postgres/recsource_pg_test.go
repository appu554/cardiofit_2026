package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/cardiofit/pharmacist-self-visibility/internal/dashboards"
	"github.com/cardiofit/pharmacist-self-visibility/internal/store/postgres"
)

// ---------------------------------------------------------------------------
// Test 1: InMemoryRecSource — always runs (no DB required).
// ---------------------------------------------------------------------------

func TestInMemoryRecSource_FilterByAuthor(t *testing.T) {
	authorA := uuid.New()
	authorB := uuid.New()

	rows := []dashboards.RecRow{
		{ID: uuid.New(), AuthorID: authorA, State: "drafted"},
		{ID: uuid.New(), AuthorID: authorA, State: "submitted"},
		{ID: uuid.New(), AuthorID: authorB, State: "implemented"},
	}

	src := postgres.NewInMemoryRecSource(rows)

	got, err := src.ListByAuthor(context.Background(), authorA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 rows for authorA, got %d", len(got))
	}
	for _, r := range got {
		if r.AuthorID != authorA {
			t.Errorf("row authorID mismatch: got %v, want %v", r.AuthorID, authorA)
		}
	}

	gotB, err := src.ListByAuthor(context.Background(), authorB)
	if err != nil {
		t.Fatalf("unexpected error for authorB: %v", err)
	}
	if len(gotB) != 1 {
		t.Errorf("expected 1 row for authorB, got %d", len(gotB))
	}
}

// ---------------------------------------------------------------------------
// Test 2: PostgresRecSource round-trip — skips if VAIDSHALA_TEST_DSN unset.
// ---------------------------------------------------------------------------

func TestPostgresRecSource_RoundTrip(t *testing.T) {
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN not set; skipping Postgres integration test")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping: %v", err)
	}

	authorA := uuid.New()
	authorB := uuid.New()
	residentID := uuid.New()

	idA1 := uuid.New()
	idA2 := uuid.New()
	idB1 := uuid.New()

	// Insert fixtures.
	insertRec := func(id, resID, authorID uuid.UUID, state string) {
		t.Helper()
		_, err := db.Exec(`
			INSERT INTO recommendations
				(id, resident_id, author_id, state, type, urgency, title, clinical_content)
			VALUES ($1, $2, $3, $4, 'monitor', 'green', 'test', '{}')
		`, id, resID, authorID, state)
		if err != nil {
			t.Fatalf("insert fixture %v: %v", id, err)
		}
	}

	insertRec(idA1, residentID, authorA, "drafted")
	insertRec(idA2, residentID, authorA, "submitted")
	insertRec(idB1, residentID, authorB, "implemented")

	t.Cleanup(func() {
		for _, id := range []uuid.UUID{idA1, idA2, idB1} {
			_, _ = db.Exec(`DELETE FROM recommendations WHERE id = $1`, id)
		}
	})

	src := postgres.NewPostgresRecSource(db)
	got, err := src.ListByAuthor(context.Background(), authorA)
	if err != nil {
		t.Fatalf("ListByAuthor: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 rows for authorA, got %d", len(got))
	}
	for _, r := range got {
		if r.AuthorID != authorA {
			t.Errorf("row authorID mismatch: got %v", r.AuthorID)
		}
	}
}

// ---------------------------------------------------------------------------
// Test 3: PostgresRecSource ordering — skips if VAIDSHALA_TEST_DSN unset.
// ---------------------------------------------------------------------------

func TestPostgresRecSource_OrderedByRecency(t *testing.T) {
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN not set; skipping Postgres integration test")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping: %v", err)
	}

	author := uuid.New()
	residentID := uuid.New()
	idOlder := uuid.New()
	idNewer := uuid.New()

	tOlder := time.Now().Add(-1 * time.Hour).UTC()
	tNewer := time.Now().UTC()

	_, err = db.Exec(`
		INSERT INTO recommendations
			(id, resident_id, author_id, state, type, urgency, title, clinical_content, created_at, updated_at)
		VALUES ($1, $2, $3, 'drafted', 'monitor', 'green', 'older', '{}', $4, $4)
	`, idOlder, residentID, author, tOlder)
	if err != nil {
		t.Fatalf("insert older: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO recommendations
			(id, resident_id, author_id, state, type, urgency, title, clinical_content, created_at, updated_at)
		VALUES ($1, $2, $3, 'submitted', 'monitor', 'green', 'newer', '{}', $4, $4)
	`, idNewer, residentID, author, tNewer)
	if err != nil {
		t.Fatalf("insert newer: %v", err)
	}

	t.Cleanup(func() {
		for _, id := range []uuid.UUID{idOlder, idNewer} {
			_, _ = db.Exec(`DELETE FROM recommendations WHERE id = $1`, id)
		}
	})

	src := postgres.NewPostgresRecSource(db)
	got, err := src.ListByAuthor(context.Background(), author)
	if err != nil {
		t.Fatalf("ListByAuthor: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
	// Most recent first.
	if got[0].ID != idNewer {
		t.Errorf("expected newest row first; got ID %v", got[0].ID)
	}
	if got[1].ID != idOlder {
		t.Errorf("expected older row second; got ID %v", got[1].ID)
	}
}
