// Package citations — postgres_registry_test.go contains integration tests
// for PostgresRegistry against migration 043 (source_versions +
// recommendation_citations).
//
// All tests skip cleanly when VAIDSHALA_TEST_DSN is unset so the unit-test
// suite stays green in environments without a Postgres instance.
//
// Each test is self-contained: it generates unique IDs (via uuid.New() or
// t.Name()-prefixed strings) and registers a t.Cleanup callback that deletes
// the rows it created. Tests can therefore run in any order against the same
// database.
//
// VisibilityClass: AD — verifies audit-defensible Postgres persistence
package citations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// openTestDB returns an open *sql.DB connected to VAIDSHALA_TEST_DSN, or skips
// the test if the env var is not set. It also pings the DB so a misconfigured
// DSN fails the test loudly rather than producing confusing scan errors.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN not set; skipping Postgres integration test")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("db.Ping: %v", err)
	}
	return db
}

// uniqueSourceID returns a per-test source identifier so concurrent tests
// against the same database do not collide on the source_versions PK.
func uniqueSourceID(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("kb32-test-%s-%s", t.Name(), uuid.NewString())
}

// cleanupSource registers a best-effort teardown that removes every row this
// test inserted for sourceID (citations first, then versions — order matters
// because of the FK).
func cleanupSource(t *testing.T, db *sql.DB, sourceIDs ...string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, sid := range sourceIDs {
			_, _ = db.ExecContext(ctx, `DELETE FROM recommendation_citations WHERE source_id = $1`, sid)
			_, _ = db.ExecContext(ctx, `DELETE FROM source_versions WHERE source_id = $1`, sid)
		}
	})
}

// ---------------------------------------------------------------------------
// Register / Get
// ---------------------------------------------------------------------------

func TestPostgresRegistry_RegisterAndGet(t *testing.T) {
	db := openTestDB(t)
	reg := NewPostgresRegistry(db)
	ctx := context.Background()

	sourceID := uniqueSourceID(t)
	cleanupSource(t, db, sourceID)

	sv := SourceVersion{
		SourceID:      sourceID,
		Version:       "1",
		EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectiveTo:   nil,
		ContentHash:   "hash-1",
		Status:        StatusActive,
	}
	if err := reg.Register(ctx, sv); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := reg.Get(ctx, sourceID, "1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SourceID != sv.SourceID || got.Version != sv.Version {
		t.Errorf("Get returned wrong identity: got (%s, %s)", got.SourceID, got.Version)
	}
	if got.ContentHash != "hash-1" {
		t.Errorf("ContentHash = %q; want %q", got.ContentHash, "hash-1")
	}
	if got.Status != StatusActive {
		t.Errorf("Status = %q; want %q", got.Status, StatusActive)
	}
	if !got.EffectiveFrom.Equal(sv.EffectiveFrom) {
		t.Errorf("EffectiveFrom = %v; want %v", got.EffectiveFrom, sv.EffectiveFrom)
	}
	if got.EffectiveTo != nil {
		t.Errorf("EffectiveTo = %v; want nil", got.EffectiveTo)
	}
}

func TestPostgresRegistry_RegisterDuplicate_ReturnsErrVersionExists(t *testing.T) {
	db := openTestDB(t)
	reg := NewPostgresRegistry(db)
	ctx := context.Background()

	sourceID := uniqueSourceID(t)
	cleanupSource(t, db, sourceID)

	sv := SourceVersion{
		SourceID:      sourceID,
		Version:       "1",
		EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ContentHash:   "hash",
		Status:        StatusActive,
	}
	if err := reg.Register(ctx, sv); err != nil {
		t.Fatalf("first Register: %v", err)
	}

	err := reg.Register(ctx, sv)
	if !errors.Is(err, ErrVersionExists) {
		t.Fatalf("second Register: got %v; want ErrVersionExists", err)
	}
}

func TestPostgresRegistry_GetMissing_ReturnsErrVersionNotFound(t *testing.T) {
	db := openTestDB(t)
	reg := NewPostgresRegistry(db)
	ctx := context.Background()

	_, err := reg.Get(ctx, uniqueSourceID(t), "does-not-exist")
	if !errors.Is(err, ErrVersionNotFound) {
		t.Fatalf("Get on missing row: got %v; want ErrVersionNotFound", err)
	}
}

// ---------------------------------------------------------------------------
// ListVersions
// ---------------------------------------------------------------------------

func TestPostgresRegistry_ListVersions_OrderedByEffectiveFrom(t *testing.T) {
	db := openTestDB(t)
	reg := NewPostgresRegistry(db)
	ctx := context.Background()

	sourceID := uniqueSourceID(t)
	cleanupSource(t, db, sourceID)

	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	// Insert in deliberately scrambled order — registry should still return
	// them sorted by effective_from ASC.
	for _, sv := range []SourceVersion{
		{SourceID: sourceID, Version: "3", EffectiveFrom: t3, ContentHash: "c3", Status: StatusActive},
		{SourceID: sourceID, Version: "1", EffectiveFrom: t1, EffectiveTo: &t2, ContentHash: "c1", Status: StatusAmended},
		{SourceID: sourceID, Version: "2", EffectiveFrom: t2, EffectiveTo: &t3, ContentHash: "c2", Status: StatusAmended},
	} {
		if err := reg.Register(ctx, sv); err != nil {
			t.Fatalf("Register %s: %v", sv.Version, err)
		}
	}

	got, err := reg.ListVersions(ctx, sourceID)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("ListVersions len = %d; want 3", len(got))
	}
	wantOrder := []string{"1", "2", "3"}
	for i, v := range got {
		if v.Version != wantOrder[i] {
			t.Errorf("ListVersions[%d].Version = %q; want %q", i, v.Version, wantOrder[i])
		}
	}
}

// ---------------------------------------------------------------------------
// ActiveVersion — half-open interval semantics
// ---------------------------------------------------------------------------

func TestPostgresRegistry_ActiveVersion_TimeWindowSemantics(t *testing.T) {
	db := openTestDB(t)
	reg := NewPostgresRegistry(db)
	ctx := context.Background()

	sourceID := uniqueSourceID(t)
	cleanupSource(t, db, sourceID)

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	if err := reg.Register(ctx, SourceVersion{
		SourceID:      sourceID,
		Version:       "1",
		EffectiveFrom: from,
		EffectiveTo:   &to,
		ContentHash:   "h",
		Status:        StatusAmended,
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// At EffectiveFrom → active (inclusive lower bound).
	if v, err := reg.ActiveVersion(ctx, sourceID, from); err != nil || v == nil || v.Version != "1" {
		t.Errorf("ActiveVersion(from): got (%v, %v); want version=1, nil error", v, err)
	}

	// Mid-window → active.
	mid := from.Add(48 * time.Hour)
	if v, err := reg.ActiveVersion(ctx, sourceID, mid); err != nil || v == nil || v.Version != "1" {
		t.Errorf("ActiveVersion(mid): got (%v, %v); want version=1, nil error", v, err)
	}

	// At exactly EffectiveTo → NOT active (exclusive upper bound).
	_, err := reg.ActiveVersion(ctx, sourceID, to)
	if !errors.Is(err, ErrNoActiveVersion) {
		t.Errorf("ActiveVersion(to): got %v; want ErrNoActiveVersion (exclusive upper bound)", err)
	}

	// Past EffectiveTo → not active.
	_, err = reg.ActiveVersion(ctx, sourceID, to.Add(time.Hour))
	if !errors.Is(err, ErrNoActiveVersion) {
		t.Errorf("ActiveVersion(after to): got %v; want ErrNoActiveVersion", err)
	}

	// Before EffectiveFrom → not active.
	_, err = reg.ActiveVersion(ctx, sourceID, from.Add(-time.Hour))
	if !errors.Is(err, ErrNoActiveVersion) {
		t.Errorf("ActiveVersion(before from): got %v; want ErrNoActiveVersion", err)
	}
}

func TestPostgresRegistry_ActiveVersion_NoneActive_ReturnsErrNoActiveVersion(t *testing.T) {
	db := openTestDB(t)
	reg := NewPostgresRegistry(db)
	ctx := context.Background()

	_, err := reg.ActiveVersion(ctx, uniqueSourceID(t), time.Now().UTC())
	if !errors.Is(err, ErrNoActiveVersion) {
		t.Fatalf("ActiveVersion on empty source: got %v; want ErrNoActiveVersion", err)
	}
}

// ---------------------------------------------------------------------------
// Amend
// ---------------------------------------------------------------------------

func TestPostgresRegistry_Amend_ClosesOldAndOpensNew_InTransaction(t *testing.T) {
	db := openTestDB(t)
	reg := NewPostgresRegistry(db)
	ctx := context.Background()

	sourceID := uniqueSourceID(t)
	cleanupSource(t, db, sourceID)

	from1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	from2 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	if err := reg.Register(ctx, SourceVersion{
		SourceID:      sourceID,
		Version:       "1",
		EffectiveFrom: from1,
		ContentHash:   "v1",
		Status:        StatusActive,
	}); err != nil {
		t.Fatalf("Register v1: %v", err)
	}

	if err := reg.Amend(ctx, sourceID, "2", "v2", from2); err != nil {
		t.Fatalf("Amend: %v", err)
	}

	old, err := reg.Get(ctx, sourceID, "1")
	if err != nil {
		t.Fatalf("Get v1 after amend: %v", err)
	}
	if old.Status != StatusAmended {
		t.Errorf("old.Status = %q; want %q", old.Status, StatusAmended)
	}
	if old.EffectiveTo == nil || !old.EffectiveTo.Equal(from2) {
		t.Errorf("old.EffectiveTo = %v; want %v", old.EffectiveTo, from2)
	}

	nv, err := reg.Get(ctx, sourceID, "2")
	if err != nil {
		t.Fatalf("Get v2 after amend: %v", err)
	}
	if nv.Status != StatusActive {
		t.Errorf("new.Status = %q; want %q", nv.Status, StatusActive)
	}
	if nv.EffectiveTo != nil {
		t.Errorf("new.EffectiveTo = %v; want nil", nv.EffectiveTo)
	}
	if !nv.EffectiveFrom.Equal(from2) {
		t.Errorf("new.EffectiveFrom = %v; want %v", nv.EffectiveFrom, from2)
	}
	if nv.ContentHash != "v2" {
		t.Errorf("new.ContentHash = %q; want %q", nv.ContentHash, "v2")
	}

	// ActiveVersion at from2 should now return v2.
	got, err := reg.ActiveVersion(ctx, sourceID, from2)
	if err != nil || got == nil || got.Version != "2" {
		t.Errorf("ActiveVersion(from2) after amend: got (%v, %v); want v2", got, err)
	}

	// NOTE: transaction-failure injection (mid-amend abort → partial state)
	// would require monkey-patching tx.ExecContext or a chaos hook in the
	// Registry. Left as TODO — the Amend implementation uses BeginTx +
	// defer Rollback + Commit, so structurally it is atomic.
}

// ---------------------------------------------------------------------------
// Retract
// ---------------------------------------------------------------------------

func TestPostgresRegistry_Retract_FlipsAllVersionsStatus(t *testing.T) {
	db := openTestDB(t)
	reg := NewPostgresRegistry(db)
	ctx := context.Background()

	sourceID := uniqueSourceID(t)
	cleanupSource(t, db, sourceID)

	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	for _, sv := range []SourceVersion{
		{SourceID: sourceID, Version: "1", EffectiveFrom: t1, EffectiveTo: &t2, ContentHash: "v1", Status: StatusAmended},
		{SourceID: sourceID, Version: "2", EffectiveFrom: t2, ContentHash: "v2", Status: StatusActive},
	} {
		if err := reg.Register(ctx, sv); err != nil {
			t.Fatalf("Register %s: %v", sv.Version, err)
		}
	}

	if err := reg.Retract(ctx, sourceID, "withdrawn by publisher"); err != nil {
		t.Fatalf("Retract: %v", err)
	}

	all, err := reg.ListVersions(ctx, sourceID)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	for _, sv := range all {
		if sv.Status != StatusRetracted {
			t.Errorf("version %s Status = %q; want %q", sv.Version, sv.Status, StatusRetracted)
		}
	}
}

// ---------------------------------------------------------------------------
// Supersede
// ---------------------------------------------------------------------------

func TestPostgresRegistry_Supersede_OldRetiredNewActive(t *testing.T) {
	db := openTestDB(t)
	reg := NewPostgresRegistry(db)
	ctx := context.Background()

	oldID := uniqueSourceID(t) + "-old"
	newID := uniqueSourceID(t) + "-new"
	cleanupSource(t, db, oldID, newID)

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	supersedeAt := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	if err := reg.Register(ctx, SourceVersion{
		SourceID:      oldID,
		Version:       "1",
		EffectiveFrom: from,
		ContentHash:   "old",
		Status:        StatusActive,
	}); err != nil {
		t.Fatalf("Register old: %v", err)
	}

	if err := reg.Supersede(ctx, oldID, newID, supersedeAt); err != nil {
		t.Fatalf("Supersede: %v", err)
	}

	old, err := reg.Get(ctx, oldID, "1")
	if err != nil {
		t.Fatalf("Get old: %v", err)
	}
	if old.Status != StatusSuperseded {
		t.Errorf("old.Status = %q; want %q", old.Status, StatusSuperseded)
	}
	if old.EffectiveTo == nil || !old.EffectiveTo.Equal(supersedeAt) {
		t.Errorf("old.EffectiveTo = %v; want %v", old.EffectiveTo, supersedeAt)
	}

	nv, err := reg.Get(ctx, newID, "1")
	if err != nil {
		t.Fatalf("Get new: %v", err)
	}
	if nv.Status != StatusActive {
		t.Errorf("new.Status = %q; want %q", nv.Status, StatusActive)
	}
	if !nv.EffectiveFrom.Equal(supersedeAt) {
		t.Errorf("new.EffectiveFrom = %v; want %v", nv.EffectiveFrom, supersedeAt)
	}
	if nv.EffectiveTo != nil {
		t.Errorf("new.EffectiveTo = %v; want nil", nv.EffectiveTo)
	}
}

// ---------------------------------------------------------------------------
// Citations: Save / Get / List
// ---------------------------------------------------------------------------

func TestPostgresRegistry_SaveCitation_GetCitation_RoundTrip(t *testing.T) {
	db := openTestDB(t)
	reg := NewPostgresRegistry(db)
	ctx := context.Background()

	sourceID := uniqueSourceID(t)
	cleanupSource(t, db, sourceID)

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := reg.Register(ctx, SourceVersion{
		SourceID:      sourceID,
		Version:       "1",
		EffectiveFrom: from,
		ContentHash:   "h",
		Status:        StatusActive,
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	recID := uuid.NewString()
	pinnedAt := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	c := RecommendationCitation{
		RecommendationID: recID,
		SourceID:         sourceID,
		Version:          "1",
		PinnedAt:         pinnedAt,
	}
	if err := reg.SaveCitation(ctx, c); err != nil {
		t.Fatalf("SaveCitation: %v", err)
	}

	got, err := reg.GetCitation(ctx, recID, sourceID, "1")
	if err != nil {
		t.Fatalf("GetCitation: %v", err)
	}
	if got.RecommendationID != recID || got.SourceID != sourceID || got.Version != "1" {
		t.Errorf("GetCitation returned wrong identity: %+v", got)
	}
	if !got.PinnedAt.Equal(pinnedAt) {
		t.Errorf("PinnedAt = %v; want %v", got.PinnedAt, pinnedAt)
	}
}

func TestPostgresRegistry_GetCitation_Missing_ReturnsErrCitationNotFound(t *testing.T) {
	db := openTestDB(t)
	reg := NewPostgresRegistry(db)
	ctx := context.Background()

	_, err := reg.GetCitation(ctx, uuid.NewString(), uniqueSourceID(t), "1")
	if !errors.Is(err, ErrCitationNotFound) {
		t.Fatalf("GetCitation on missing row: got %v; want ErrCitationNotFound", err)
	}
}

func TestPostgresRegistry_ListCitations_FiltersByRecID(t *testing.T) {
	db := openTestDB(t)
	reg := NewPostgresRegistry(db)
	ctx := context.Background()

	sourceA := uniqueSourceID(t) + "-A"
	sourceB := uniqueSourceID(t) + "-B"
	cleanupSource(t, db, sourceA, sourceB)

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, sid := range []string{sourceA, sourceB} {
		if err := reg.Register(ctx, SourceVersion{
			SourceID:      sid,
			Version:       "1",
			EffectiveFrom: from,
			ContentHash:   "h",
			Status:        StatusActive,
		}); err != nil {
			t.Fatalf("Register %s: %v", sid, err)
		}
	}

	recID := uuid.NewString()
	otherRecID := uuid.NewString()
	pinnedAt := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)

	// Two citations for recID (across two sources), one for otherRecID.
	cites := []RecommendationCitation{
		{RecommendationID: recID, SourceID: sourceA, Version: "1", PinnedAt: pinnedAt},
		{RecommendationID: recID, SourceID: sourceB, Version: "1", PinnedAt: pinnedAt},
		{RecommendationID: otherRecID, SourceID: sourceA, Version: "1", PinnedAt: pinnedAt},
	}
	for _, c := range cites {
		if err := reg.SaveCitation(ctx, c); err != nil {
			t.Fatalf("SaveCitation: %v", err)
		}
	}

	got, err := reg.ListCitations(ctx, recID)
	if err != nil {
		t.Fatalf("ListCitations: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListCitations len = %d; want 2", len(got))
	}
	for _, c := range got {
		if c.RecommendationID != recID {
			t.Errorf("ListCitations returned wrong rec: %s; want %s", c.RecommendationID, recID)
		}
	}
}
