package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/cardiofit/shared/v2_substrate/identity"
	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
)

// helperFreshIdentityStore opens a fresh V2SubstrateStore + IdentityStore
// pair against KB20_TEST_DATABASE_URL. Skips when unset so unit-test
// CI without DB stays green. Mirrors the gating pattern used by
// TestUpsertResidentRoundTrip in v2_substrate_store_test.go.
//
// In production main.go passes the same *sql.DB into both stores; here
// we open a parallel handle (the v2 store doesn't expose its db field)
// because the IdentityStore needs raw SQL access and an *sql.DB the
// V2SubstrateStore can serve EvidenceTrace writes through.
func helperFreshIdentityStore(t *testing.T) (*IdentityStore, *V2SubstrateStore, func()) {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping identity-store integration test")
	}
	v2, err := NewV2SubstrateStore(dsn)
	if err != nil {
		t.Fatalf("NewV2SubstrateStore: %v", err)
	}
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		_ = v2.Close()
		t.Fatalf("sql.Open: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		_ = v2.Close()
		t.Fatalf("ping: %v", err)
	}
	idStore := NewIdentityStore(sqlDB, v2)
	cleanup := func() {
		_ = sqlDB.Close()
		_ = v2.Close()
	}
	return idStore, v2, cleanup
}

func TestIdentityStore_InsertAndListMappings(t *testing.T) {
	store, _, cleanup := helperFreshIdentityStore(t)
	defer cleanup()
	ctx := context.Background()

	residentID := uuid.New()
	m := interfaces.IdentityMapping{
		IdentifierKind:  "ihi",
		IdentifierValue: "8003608000099001",
		ResidentRef:     residentID,
		Confidence:      "high",
		MatchPath:       "ihi",
		Source:          "kb20-test",
	}
	first, err := store.InsertIdentityMapping(ctx, m)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if first.ID == uuid.Nil {
		t.Errorf("ID not assigned")
	}

	// Idempotent UPSERT on (kind, value, resident_ref).
	second, err := store.InsertIdentityMapping(ctx, m)
	if err != nil {
		t.Fatalf("Insert (re-write): %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("UPSERT should preserve id; got %s vs %s", second.ID, first.ID)
	}

	mappings, err := store.ListIdentityMappingsByResident(ctx, residentID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(mappings) != 1 {
		t.Errorf("expected 1 mapping, got %d", len(mappings))
	}
}

func TestIdentityStore_ReassignMappingsSince(t *testing.T) {
	store, _, cleanup := helperFreshIdentityStore(t)
	defer cleanup()
	ctx := context.Background()

	wrong := uuid.New()
	right := uuid.New()
	since := time.Now().UTC().Add(-time.Hour)

	if _, err := store.InsertIdentityMapping(ctx, interfaces.IdentityMapping{
		IdentifierKind: "medicare", IdentifierValue: "9001-" + wrong.String()[:8], ResidentRef: wrong,
		Confidence: "low", MatchPath: "name+dob+facility", Source: "test",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.InsertIdentityMapping(ctx, interfaces.IdentityMapping{
		IdentifierKind: "hospital_mrn", IdentifierValue: "MRN-" + wrong.String()[:8], ResidentRef: wrong,
		Confidence: "low", MatchPath: "name+dob+facility", Source: "test",
	}); err != nil {
		t.Fatal(err)
	}

	n, err := store.ReassignIdentityMappingsByResidentSince(ctx, wrong, right, since)
	if err != nil {
		t.Fatalf("Reassign: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 rows reassigned, got %d", n)
	}

	left, err := store.ListIdentityMappingsByResident(ctx, wrong)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Errorf("wrong resident should have 0 mappings after reassign, got %d", len(left))
	}
	moved, err := store.ListIdentityMappingsByResident(ctx, right)
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 2 {
		t.Errorf("right resident should have 2 mappings after reassign, got %d", len(moved))
	}
}

func TestIdentityStore_ReviewQueueRoundTrip(t *testing.T) {
	store, _, cleanup := helperFreshIdentityStore(t)
	defer cleanup()
	ctx := context.Background()

	cand := uuid.New()
	dist := 2
	blob, _ := json.Marshal(identity.IncomingIdentifier{
		GivenName: "Robert", FamilyName: "OConnor",
	})
	in := interfaces.IdentityReviewQueueEntry{
		IncomingIdentifier:    blob,
		CandidateResidentRefs: []uuid.UUID{cand},
		BestCandidate:         &cand,
		BestDistance:          &dist,
		MatchPath:             "name+dob+facility",
		Confidence:            "low",
		Source:                "test",
	}
	written, err := store.InsertIdentityReviewQueueEntry(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if written.Status != "pending" {
		t.Errorf("default Status: got %s want pending", written.Status)
	}

	got, err := store.GetIdentityReviewQueueEntry(ctx, written.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.BestCandidate == nil || *got.BestCandidate != cand {
		t.Errorf("BestCandidate round-trip failed")
	}

	list, err := store.ListIdentityReviewQueue(ctx, "pending", 100, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, e := range list {
		if e.ID == written.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("inserted entry not in pending list")
	}

	resolvedBy := uuid.New()
	resolvedRef := uuid.New()
	updated, err := store.UpdateIdentityReviewQueueEntryResolution(ctx, written.ID, "resolved", &resolvedRef, resolvedBy, "verified by clinical reviewer")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Status != "resolved" {
		t.Errorf("Status after update: got %s want resolved", updated.Status)
	}
	if updated.ResolvedResidentRef == nil || *updated.ResolvedResidentRef != resolvedRef {
		t.Errorf("ResolvedResidentRef mismatch")
	}
}

// TestIdentityStore_MatchAndPersist_HighIHI verifies the auto-accept
// path: HIGH IHI exact -> mapping written, EvidenceTrace node written,
// no review-queue entry.
func TestIdentityStore_MatchAndPersist_HighIHI(t *testing.T) {
	store, v2, cleanup := helperFreshIdentityStore(t)
	defer cleanup()
	ctx := context.Background()

	ihi := "8003608000077001"
	residentID := uuid.New()
	if _, err := v2.UpsertResident(ctx, models.Resident{
		ID: residentID, IHI: ihi,
		GivenName: "Ada", FamilyName: "Lovelace",
		DOB:        time.Date(1815, 12, 10, 0, 0, 0, 0, time.UTC),
		Sex:        "female",
		FacilityID: uuid.New(),
		Status:     models.ResidentStatusActive,
	}); err != nil {
		t.Fatalf("seed resident: %v", err)
	}

	res, err := store.MatchAndPersist(ctx, identity.IncomingIdentifier{
		IHI:    ihi,
		Source: "kb20-test",
	})
	if err != nil {
		t.Fatalf("MatchAndPersist: %v", err)
	}
	if res.Match.Confidence != identity.ConfidenceHigh {
		t.Errorf("Confidence: got %s want HIGH", res.Match.Confidence)
	}
	if res.ReviewQueueEntryID != nil {
		t.Errorf("HIGH match should not enqueue review")
	}
	if res.EvidenceTraceNodeRef == uuid.Nil {
		t.Errorf("EvidenceTrace node id should be set")
	}
	node, err := v2.GetEvidenceTraceNode(ctx, res.EvidenceTraceNodeRef)
	if err != nil {
		t.Fatalf("read trace node: %v", err)
	}
	if node.StateChangeType != "identity_match" {
		t.Errorf("StateChangeType: got %s want identity_match", node.StateChangeType)
	}
	mappings, err := store.ListIdentityMappingsByResident(ctx, residentID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range mappings {
		if m.IdentifierKind == "ihi" && m.IdentifierValue == ihi {
			found = true
			if m.Confidence != "high" {
				t.Errorf("confidence: got %s want high", m.Confidence)
			}
		}
	}
	if !found {
		t.Errorf("ihi mapping not written")
	}
}

func TestIdentityStore_MatchAndPersist_NoneEnqueues(t *testing.T) {
	store, _, cleanup := helperFreshIdentityStore(t)
	defer cleanup()
	ctx := context.Background()

	res, err := store.MatchAndPersist(ctx, identity.IncomingIdentifier{
		Source: "kb20-test",
	})
	if err != nil {
		t.Fatalf("MatchAndPersist: %v", err)
	}
	if res.Match.Confidence != identity.ConfidenceNone {
		t.Errorf("Confidence: got %s want NONE", res.Match.Confidence)
	}
	if res.ReviewQueueEntryID == nil {
		t.Errorf("NONE must enqueue review entry")
	}
	if res.EvidenceTraceNodeRef == uuid.Nil {
		t.Errorf("EvidenceTrace node must be written even for NONE")
	}
}

// TestIdentityStore_ResolveReview_Reroutes verifies the manual-override
// re-route requirement at Layer 2 §3.3 (the headline acceptance
// criterion): a low-confidence match auto-accepted onto the wrong
// resident, with subsequent mappings, gets corrected — and the
// subsequent mappings are repointed at the resolved resident.
func TestIdentityStore_ResolveReview_Reroutes(t *testing.T) {
	store, v2, cleanup := helperFreshIdentityStore(t)
	defer cleanup()
	ctx := context.Background()

	fac := uuid.New()
	dob := time.Date(1942, 7, 30, 0, 0, 0, 0, time.UTC)
	wrong := uuid.New()
	right := uuid.New()

	for _, r := range []models.Resident{
		{ID: wrong, GivenName: "Helen", FamilyName: "Yang", DOB: dob, Sex: "female",
			FacilityID: fac, Status: models.ResidentStatusActive},
		{ID: right, GivenName: "Helena", FamilyName: "Yang", DOB: dob, Sex: "female",
			FacilityID: fac, Status: models.ResidentStatusActive},
	} {
		if _, err := v2.UpsertResident(ctx, r); err != nil {
			t.Fatalf("seed resident: %v", err)
		}
	}

	res, err := store.MatchAndPersist(ctx, identity.IncomingIdentifier{
		GivenName:  "Helen",
		FamilyName: "Yang",
		DOB:        dob,
		FacilityID: &fac,
		Source:     "test",
	})
	if err != nil {
		t.Fatalf("MatchAndPersist (LOW): %v", err)
	}
	if res.Match.Confidence != identity.ConfidenceLow {
		t.Fatalf("Confidence: got %s want LOW", res.Match.Confidence)
	}
	if res.ReviewQueueEntryID == nil {
		t.Fatal("LOW must enqueue")
	}

	// Simulate a subsequent ingest event that wrote a mapping against
	// the (wrong) auto-chosen resident while the queue entry was still
	// pending.
	if _, err := store.InsertIdentityMapping(ctx, interfaces.IdentityMapping{
		IdentifierKind: "hospital_mrn", IdentifierValue: "MRN-555-" + uuid.New().String()[:8],
		ResidentRef: *res.Match.ResidentRef, Confidence: "high", MatchPath: "ihi", Source: "test",
	}); err != nil {
		t.Fatalf("post-queue mapping: %v", err)
	}

	reviewer := uuid.New()
	updated, rerouted, err := store.ResolveReview(ctx, *res.ReviewQueueEntryID, right, reviewer, "correct match is Helena, not Helen")
	if err != nil {
		t.Fatalf("ResolveReview: %v", err)
	}
	if updated.Status != "resolved" {
		t.Errorf("Status: got %s want resolved", updated.Status)
	}
	if rerouted < 1 {
		t.Errorf("expected at least 1 mapping rerouted, got %d", rerouted)
	}

	moved, err := store.ListIdentityMappingsByResident(ctx, right)
	if err != nil {
		t.Fatal(err)
	}
	foundMRN := false
	for _, m := range moved {
		if m.IdentifierKind == "hospital_mrn" {
			foundMRN = true
		}
	}
	if !foundMRN {
		t.Errorf("MRN mapping should have been re-routed to right")
	}
}
