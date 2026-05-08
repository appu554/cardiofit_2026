package permissions

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// testDB returns a connection to the local Docker Postgres for integration
// tests. Tests skip if KB_TEST_DATABASE_URL is unset, so `go test ./...`
// still passes in CI environments without a database.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("KB_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB_TEST_DATABASE_URL unset; skipping DB integration test")
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

// ---------------------------------------------------------------------------
// Helpers — fixture factories
// ---------------------------------------------------------------------------

func newSimpleScope() Scope {
	return Scope{
		Class:         WO,
		ViewType:      ViewTypePharmacist,
		ResourceTypes: []string{"active_recommendations"},
	}
}

func newPFAScope() Scope {
	return Scope{
		Class:         PFA,
		ViewType:      ViewTypeEmployer,
		ResourceTypes: []string{"rir_summary"},
		Gate: &AggregationGate{
			MinObservations:  30,
			TimeWindow:       90 * 24 * time.Hour,
			DelayWindow:      30 * 24 * time.Hour,
			ContractualBasis: "enterprise tier §4.2",
			ExplicitNotice:   true,
		},
	}
}

func newViewPermission() ViewPermission {
	return ViewPermission{
		ID:           uuid.New(),
		SubjectID:    uuid.New(),
		ViewerRoleID: uuid.New(),
		Scope:        newSimpleScope(),
		GrantedAt:    time.Now().UTC(),
		GrantedByID:  uuid.New(),
	}
}

func newDataConsent(pharmacistID uuid.UUID, dataElement, target string, expiresIn time.Duration) DataAggregationConsent {
	return DataAggregationConsent{
		ID:                uuid.New(),
		PharmacistID:      pharmacistID,
		DataElement:       dataElement,
		AggregationTarget: target,
		Purpose:           PurposeWorkforcePlanning,
		GrantedAt:         time.Now().UTC(),
		ExpiresAt:         time.Now().UTC().Add(expiresIn),
	}
}

// ===========================================================================
// InMemoryStore tests — no DB required, run in CI
// ===========================================================================

func TestInMemoryStore_CreateAndGet(t *testing.T) {
	store := &InMemoryStore{}
	ctx := context.Background()

	p := newViewPermission()
	got, err := store.Create(ctx, p)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("Create returned wrong ID: got %v want %v", got.ID, p.ID)
	}

	retrieved, err := store.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if retrieved.ID != p.ID {
		t.Errorf("Get returned wrong ID")
	}
	if retrieved.Scope.Class != p.Scope.Class {
		t.Errorf("Scope.Class mismatch: got %v want %v", retrieved.Scope.Class, p.Scope.Class)
	}
}

func TestInMemoryStore_GetNotFound(t *testing.T) {
	store := &InMemoryStore{}
	_, err := store.Get(context.Background(), uuid.New())
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound; got %v", err)
	}
}

func TestInMemoryStore_FindForSubjectAndViewer_ReturnsActive(t *testing.T) {
	store := &InMemoryStore{}
	ctx := context.Background()

	subjectID := uuid.New()
	viewerID := uuid.New()

	p := newViewPermission()
	p.SubjectID = subjectID
	p.ViewerRoleID = viewerID
	if _, err := store.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	found, err := store.FindForSubjectAndViewer(ctx, subjectID, viewerID)
	if err != nil {
		t.Fatalf("FindForSubjectAndViewer: %v", err)
	}
	if found == nil {
		t.Fatal("expected non-nil result for active permission")
	}
	if found.ID != p.ID {
		t.Errorf("wrong permission returned: got %v want %v", found.ID, p.ID)
	}
}

func TestInMemoryStore_FindForSubjectAndViewer_RevocationReturnsNil(t *testing.T) {
	store := &InMemoryStore{}
	ctx := context.Background()

	subjectID := uuid.New()
	viewerID := uuid.New()

	p := newViewPermission()
	p.SubjectID = subjectID
	p.ViewerRoleID = viewerID
	if _, err := store.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// revoke it
	if err := store.Revoke(ctx, p.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	found, err := store.FindForSubjectAndViewer(ctx, subjectID, viewerID)
	if err != nil {
		t.Fatalf("FindForSubjectAndViewer after revoke: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil after revoke; got %+v", found)
	}
}

func TestInMemoryStore_FindForSubjectAndViewer_NoneExists(t *testing.T) {
	store := &InMemoryStore{}
	found, err := store.FindForSubjectAndViewer(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for unknown pair; got %+v", found)
	}
}

func TestInMemoryStore_ListBySubject_IncludesRevoked(t *testing.T) {
	store := &InMemoryStore{}
	ctx := context.Background()

	subjectID := uuid.New()

	p1 := newViewPermission()
	p1.SubjectID = subjectID
	p2 := newViewPermission()
	p2.SubjectID = subjectID

	for _, p := range []ViewPermission{p1, p2} {
		if _, err := store.Create(ctx, p); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}
	// revoke one
	if err := store.Revoke(ctx, p1.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	all, err := store.ListBySubject(ctx, subjectID)
	if err != nil {
		t.Fatalf("ListBySubject: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("ListBySubject should return all (including revoked): got %d", len(all))
	}
}

func TestInMemoryStore_Revoke_UnknownID(t *testing.T) {
	store := &InMemoryStore{}
	err := store.Revoke(context.Background(), uuid.New())
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows for unknown id; got %v", err)
	}
}

func TestInMemoryStore_PFAScope_RoundTrip(t *testing.T) {
	store := &InMemoryStore{}
	ctx := context.Background()

	p := newViewPermission()
	p.Scope = newPFAScope()

	created, err := store.Create(ctx, p)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := store.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Scope.Class != PFA {
		t.Errorf("Scope.Class: got %v want PFA", got.Scope.Class)
	}
	if got.Scope.Gate == nil {
		t.Fatal("Scope.Gate must be preserved for PFA scope")
	}
	if got.Scope.Gate.MinObservations != 30 {
		t.Errorf("Gate.MinObservations: got %d want 30", got.Scope.Gate.MinObservations)
	}
}

// ===========================================================================
// InMemoryDataConsentStore tests — no DB required
// ===========================================================================

func TestInMemoryDataConsentStore_CreateAndFind(t *testing.T) {
	store := &InMemoryDataConsentStore{}
	ctx := context.Background()
	pharmacistID := uuid.New()

	c := newDataConsent(pharmacistID, "rir_class_specific", "employer_xyz", 365*24*time.Hour)
	if _, err := store.CreateConsent(ctx, c); err != nil {
		t.Fatalf("CreateConsent: %v", err)
	}

	found, err := store.FindActiveConsent(ctx, pharmacistID, "rir_class_specific", "employer_xyz", time.Now())
	if err != nil {
		t.Fatalf("FindActiveConsent: %v", err)
	}
	if found == nil {
		t.Fatal("expected non-nil active consent")
	}
	if found.ID != c.ID {
		t.Errorf("wrong consent: got %v want %v", found.ID, c.ID)
	}
}

func TestInMemoryDataConsentStore_FindActive_ExpiredReturnsNil(t *testing.T) {
	store := &InMemoryDataConsentStore{}
	ctx := context.Background()
	pharmacistID := uuid.New()

	// Create a consent that already expired
	c := newDataConsent(pharmacistID, "rir_class_specific", "employer_xyz", -1*time.Hour)
	if _, err := store.CreateConsent(ctx, c); err != nil {
		t.Fatalf("CreateConsent: %v", err)
	}

	found, err := store.FindActiveConsent(ctx, pharmacistID, "rir_class_specific", "employer_xyz", time.Now())
	if err != nil {
		t.Fatalf("FindActiveConsent: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for expired consent; got %+v", found)
	}
}

func TestInMemoryDataConsentStore_FindActive_RevokedReturnsNil(t *testing.T) {
	store := &InMemoryDataConsentStore{}
	ctx := context.Background()
	pharmacistID := uuid.New()

	c := newDataConsent(pharmacistID, "rir_class_specific", "employer_xyz", 365*24*time.Hour)
	if _, err := store.CreateConsent(ctx, c); err != nil {
		t.Fatalf("CreateConsent: %v", err)
	}
	if err := store.RevokeConsent(ctx, c.ID, "pharmacist withdrew consent"); err != nil {
		t.Fatalf("RevokeConsent: %v", err)
	}

	found, err := store.FindActiveConsent(ctx, pharmacistID, "rir_class_specific", "employer_xyz", time.Now())
	if err != nil {
		t.Fatalf("FindActiveConsent: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for revoked consent; got %+v", found)
	}
}

func TestInMemoryDataConsentStore_FindActive_WrongTargetReturnsNil(t *testing.T) {
	store := &InMemoryDataConsentStore{}
	ctx := context.Background()
	pharmacistID := uuid.New()

	c := newDataConsent(pharmacistID, "rir_class_specific", "employer_xyz", 365*24*time.Hour)
	if _, err := store.CreateConsent(ctx, c); err != nil {
		t.Fatalf("CreateConsent: %v", err)
	}

	found, err := store.FindActiveConsent(ctx, pharmacistID, "rir_class_specific", "different_employer", time.Now())
	if err != nil {
		t.Fatalf("FindActiveConsent: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for wrong target; got %+v", found)
	}
}

func TestInMemoryDataConsentStore_FindActive_EmptyTargetMatchesAny(t *testing.T) {
	store := &InMemoryDataConsentStore{}
	ctx := context.Background()
	pharmacistID := uuid.New()

	c := newDataConsent(pharmacistID, "rir_class_specific", "employer_xyz", 365*24*time.Hour)
	if _, err := store.CreateConsent(ctx, c); err != nil {
		t.Fatalf("CreateConsent: %v", err)
	}

	// Empty aggregationTarget should match any target
	found, err := store.FindActiveConsent(ctx, pharmacistID, "rir_class_specific", "", time.Now())
	if err != nil {
		t.Fatalf("FindActiveConsent (empty target): %v", err)
	}
	if found == nil {
		t.Fatal("empty target should match any consent for that element; got nil")
	}
}

func TestInMemoryDataConsentStore_ListByPharmacist(t *testing.T) {
	store := &InMemoryDataConsentStore{}
	ctx := context.Background()
	pharmacistID := uuid.New()

	c1 := newDataConsent(pharmacistID, "rir_class_specific", "employer_a", 365*24*time.Hour)
	c2 := newDataConsent(pharmacistID, "attendance_pattern", "employer_b", 365*24*time.Hour)
	other := newDataConsent(uuid.New(), "rir_class_specific", "employer_a", 365*24*time.Hour)

	for _, c := range []DataAggregationConsent{c1, c2, other} {
		if _, err := store.CreateConsent(ctx, c); err != nil {
			t.Fatalf("CreateConsent: %v", err)
		}
	}

	list, err := store.ListByPharmacist(ctx, pharmacistID)
	if err != nil {
		t.Fatalf("ListByPharmacist: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListByPharmacist: got %d records; want 2", len(list))
	}
}

func TestInMemoryDataConsentStore_Revoke_UnknownID(t *testing.T) {
	store := &InMemoryDataConsentStore{}
	err := store.RevokeConsent(context.Background(), uuid.New(), "no reason")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows for unknown id; got %v", err)
	}
}

// ===========================================================================
// PostgresStore tests — skip if KB_TEST_DATABASE_URL unset
// ===========================================================================

func TestPostgresStore_ViewPermission_CreateAndGet(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	p := newViewPermission()
	p.Scope = newPFAScope()

	created, err := store.Create(ctx, p)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM view_permissions WHERE id = $1", p.ID)
	})

	got, err := store.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("ID mismatch: got %v want %v", got.ID, p.ID)
	}
	if got.SubjectID != p.SubjectID {
		t.Errorf("SubjectID mismatch")
	}
	if got.Scope.Class != PFA {
		t.Errorf("Scope.Class: got %v want PFA", got.Scope.Class)
	}
	if got.Scope.Gate == nil {
		t.Fatal("Scope.Gate must survive JSONB round-trip")
	}
	if got.Scope.Gate.MinObservations != 30 {
		t.Errorf("Gate.MinObservations: got %d want 30", got.Scope.Gate.MinObservations)
	}
	if len(got.Scope.ResourceTypes) != 1 || got.Scope.ResourceTypes[0] != "rir_summary" {
		t.Errorf("ResourceTypes mismatch: %v", got.Scope.ResourceTypes)
	}
}

func TestPostgresStore_ViewPermission_GetNotFound(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	_, err := store.Get(context.Background(), uuid.New())
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound; got %v", err)
	}
}

func TestPostgresStore_ViewPermission_FindForSubjectAndViewer_ActiveOnly(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	subjectID := uuid.New()
	viewerID := uuid.New()

	active := newViewPermission()
	active.SubjectID = subjectID
	active.ViewerRoleID = viewerID

	if _, err := store.Create(ctx, active); err != nil {
		t.Fatalf("Create active: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM view_permissions WHERE subject_id = $1", subjectID)
	})

	found, err := store.FindForSubjectAndViewer(ctx, subjectID, viewerID)
	if err != nil {
		t.Fatalf("FindForSubjectAndViewer: %v", err)
	}
	if found == nil {
		t.Fatal("expected active permission; got nil")
	}
	if found.ID != active.ID {
		t.Errorf("wrong permission: got %v want %v", found.ID, active.ID)
	}
}

func TestPostgresStore_ViewPermission_FindForSubjectAndViewer_RevokedReturnsNil(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	subjectID := uuid.New()
	viewerID := uuid.New()

	p := newViewPermission()
	p.SubjectID = subjectID
	p.ViewerRoleID = viewerID

	if _, err := store.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM view_permissions WHERE subject_id = $1", subjectID)
	})

	if err := store.Revoke(ctx, p.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	found, err := store.FindForSubjectAndViewer(ctx, subjectID, viewerID)
	if err != nil {
		t.Fatalf("FindForSubjectAndViewer after revoke: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil after revoke; got %+v", found)
	}
}

func TestPostgresStore_ViewPermission_ListBySubjectIncludesRevoked(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	subjectID := uuid.New()

	p1 := newViewPermission()
	p1.SubjectID = subjectID
	p2 := newViewPermission()
	p2.SubjectID = subjectID

	for _, p := range []ViewPermission{p1, p2} {
		if _, err := store.Create(ctx, p); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM view_permissions WHERE subject_id = $1", subjectID)
	})

	// revoke one
	if err := store.Revoke(ctx, p1.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	all, err := store.ListBySubject(ctx, subjectID)
	if err != nil {
		t.Fatalf("ListBySubject: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("ListBySubject must include revoked; got %d records want 2", len(all))
	}
}

func TestPostgresStore_ViewPermission_Revoke_UnknownIDReturnsErrNoRows(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	err := store.Revoke(context.Background(), uuid.New())
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows for unknown id; got %v", err)
	}
}

// ===========================================================================
// PostgresDataConsentStore tests
// ===========================================================================

func TestPostgresDataConsentStore_CreateAndFind(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresDataConsentStore(db)
	ctx := context.Background()
	pharmacistID := uuid.New()

	c := newDataConsent(pharmacistID, "rir_class_specific", "employer_xyz", 365*24*time.Hour)
	if _, err := store.CreateConsent(ctx, c); err != nil {
		t.Fatalf("CreateConsent: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM data_aggregation_consents WHERE id = $1", c.ID)
	})

	found, err := store.FindActiveConsent(ctx, pharmacistID, "rir_class_specific", "employer_xyz", time.Now())
	if err != nil {
		t.Fatalf("FindActiveConsent: %v", err)
	}
	if found == nil {
		t.Fatal("expected non-nil active consent")
	}
	if found.ID != c.ID {
		t.Errorf("wrong consent: got %v want %v", found.ID, c.ID)
	}
	if found.DataElement != "rir_class_specific" {
		t.Errorf("DataElement mismatch: %q", found.DataElement)
	}
	if found.Purpose != PurposeWorkforcePlanning {
		t.Errorf("Purpose mismatch: %q", found.Purpose)
	}
}

func TestPostgresDataConsentStore_FindActive_ExpiredReturnsNil(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresDataConsentStore(db)
	ctx := context.Background()
	pharmacistID := uuid.New()

	// expired in the past
	c := newDataConsent(pharmacistID, "rir_class_specific", "employer_xyz", -1*time.Hour)
	if _, err := store.CreateConsent(ctx, c); err != nil {
		t.Fatalf("CreateConsent: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM data_aggregation_consents WHERE id = $1", c.ID)
	})

	found, err := store.FindActiveConsent(ctx, pharmacistID, "rir_class_specific", "employer_xyz", time.Now())
	if err != nil {
		t.Fatalf("FindActiveConsent: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for expired consent; got %+v", found)
	}
}

func TestPostgresDataConsentStore_FindActive_RevokedReturnsNil(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresDataConsentStore(db)
	ctx := context.Background()
	pharmacistID := uuid.New()

	c := newDataConsent(pharmacistID, "rir_class_specific", "employer_xyz", 365*24*time.Hour)
	if _, err := store.CreateConsent(ctx, c); err != nil {
		t.Fatalf("CreateConsent: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM data_aggregation_consents WHERE id = $1", c.ID)
	})

	if err := store.RevokeConsent(ctx, c.ID, "withdrew"); err != nil {
		t.Fatalf("RevokeConsent: %v", err)
	}

	found, err := store.FindActiveConsent(ctx, pharmacistID, "rir_class_specific", "employer_xyz", time.Now())
	if err != nil {
		t.Fatalf("FindActiveConsent: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for revoked consent; got %+v", found)
	}
}

func TestPostgresDataConsentStore_FindActive_WrongTargetReturnsNil(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresDataConsentStore(db)
	ctx := context.Background()
	pharmacistID := uuid.New()

	c := newDataConsent(pharmacistID, "rir_class_specific", "employer_xyz", 365*24*time.Hour)
	if _, err := store.CreateConsent(ctx, c); err != nil {
		t.Fatalf("CreateConsent: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM data_aggregation_consents WHERE id = $1", c.ID)
	})

	found, err := store.FindActiveConsent(ctx, pharmacistID, "rir_class_specific", "different_employer", time.Now())
	if err != nil {
		t.Fatalf("FindActiveConsent: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for wrong aggregation_target; got %+v", found)
	}
}

func TestPostgresDataConsentStore_FindActive_EmptyTargetMatchesAny(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresDataConsentStore(db)
	ctx := context.Background()
	pharmacistID := uuid.New()

	c := newDataConsent(pharmacistID, "rir_class_specific", "employer_xyz", 365*24*time.Hour)
	if _, err := store.CreateConsent(ctx, c); err != nil {
		t.Fatalf("CreateConsent: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM data_aggregation_consents WHERE id = $1", c.ID)
	})

	// Empty target should match the record regardless of stored target value
	found, err := store.FindActiveConsent(ctx, pharmacistID, "rir_class_specific", "", time.Now())
	if err != nil {
		t.Fatalf("FindActiveConsent (empty target): %v", err)
	}
	if found == nil {
		t.Fatal("empty aggregationTarget must match any active consent for that element")
	}
}

func TestPostgresDataConsentStore_ListByPharmacist(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresDataConsentStore(db)
	ctx := context.Background()
	pharmacistID := uuid.New()

	c1 := newDataConsent(pharmacistID, "rir_class_specific", "employer_a", 365*24*time.Hour)
	c2 := newDataConsent(pharmacistID, "attendance_pattern", "employer_b", 365*24*time.Hour)

	for _, c := range []DataAggregationConsent{c1, c2} {
		if _, err := store.CreateConsent(ctx, c); err != nil {
			t.Fatalf("CreateConsent: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM data_aggregation_consents WHERE pharmacist_id = $1", pharmacistID)
	})

	// Revoke c2 so we can confirm list includes it
	if err := store.RevokeConsent(ctx, c2.ID, "test"); err != nil {
		t.Fatalf("RevokeConsent: %v", err)
	}

	list, err := store.ListByPharmacist(ctx, pharmacistID)
	if err != nil {
		t.Fatalf("ListByPharmacist: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListByPharmacist must include revoked; got %d want 2", len(list))
	}

	// verify revoked record has revocation reason populated
	var revokedRecord *DataAggregationConsent
	for i := range list {
		if list[i].ID == c2.ID {
			revokedRecord = &list[i]
		}
	}
	if revokedRecord == nil {
		t.Fatal("c2 not found in list")
	}
	if revokedRecord.RevokedAt == nil {
		t.Errorf("revoked_at should be set on revoked record")
	}
	if revokedRecord.RevocationReason == nil || *revokedRecord.RevocationReason != "test" {
		t.Errorf("revocation_reason should be 'test'; got %v", revokedRecord.RevocationReason)
	}
}

func TestPostgresDataConsentStore_RevokeConsent_UnknownIDReturnsErrNoRows(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresDataConsentStore(db)
	err := store.RevokeConsent(context.Background(), uuid.New(), "no-op")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows for unknown id; got %v", err)
	}
}
