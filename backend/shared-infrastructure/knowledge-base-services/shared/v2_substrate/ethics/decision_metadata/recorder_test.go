// Package decision_metadata — recorder tests.
// VisibilityClass: AD (audit-defensible)
package decision_metadata

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // future Postgres-backed tests; side-effect import only
)

// ---------------------------------------------------------------------------
// Plan verbatim tests (TestRecorder_RecordRoundTrip, TestRecorder_QueryBySubject)
// ---------------------------------------------------------------------------

func TestRecorder_RecordRoundTrip(t *testing.T) {
	store := NewInMemoryStore()
	rec := NewRecorder(store)

	id := uuid.New()
	traceRef := uuid.New()
	recID := uuid.New()
	subjectID := uuid.New().String()
	if err := rec.Record(context.Background(), Metadata{
		DecisionID:           id,
		Component:            "kb-30",
		DecisionType:         "recommendation_draft",
		AffectedSubjectID:    subjectID,
		AffectedSubjectClass: "resident",
		PrinciplesImplicated: []string{"P2", "P3"},
		ERMReviewed:          true,
		ERMOutcome:           ptr("approve_with_monitoring"),
		ContestationEnabled:  true,
		AuditTraceRef:        traceRef,
		Timestamp:            time.Now().UTC(),
		RecommendationID:     recID,
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	got, err := store.Get(context.Background(), id)
	if err != nil || got == nil {
		t.Fatalf("get: err=%v got=%v", err, got)
	}
	if got.AffectedSubjectClass != "resident" {
		t.Errorf("subject class roundtrip fail")
	}
	if len(got.PrinciplesImplicated) != 2 {
		t.Errorf("principles roundtrip fail: %v", got.PrinciplesImplicated)
	}
	if got.RecommendationID != recID {
		t.Errorf("RecommendationID roundtrip fail: got %s want %s", got.RecommendationID, recID)
	}
}

// TestRecorder_RecommendationIDZeroValueWhenAbsent verifies the documented
// sentinel semantics: when a caller omits RecommendationID, it round-trips
// as uuid.Nil (the "no associated recommendation" sentinel the /v1/explain
// reader keys off).
func TestRecorder_RecommendationIDZeroValueWhenAbsent(t *testing.T) {
	store := NewInMemoryStore()
	rec := NewRecorder(store)
	id := uuid.New()
	if err := rec.Record(context.Background(), Metadata{
		DecisionID:           id,
		Component:            "kb-30",
		DecisionType:         "non_recommendation_decision",
		AffectedSubjectID:    uuid.New().String(),
		AffectedSubjectClass: "resident",
		Timestamp:            time.Now().UTC(),
		// RecommendationID intentionally omitted — this decision is not a kb-32 recommendation
	}); err != nil {
		t.Fatalf("record: %v", err)
	}
	got, err := store.Get(context.Background(), id)
	if err != nil || got == nil {
		t.Fatalf("get: err=%v got=%v", err, got)
	}
	if got.RecommendationID != uuid.Nil {
		t.Errorf("expected RecommendationID == uuid.Nil sentinel, got %s", got.RecommendationID)
	}
}

func TestRecorder_QueryBySubject(t *testing.T) {
	store := NewInMemoryStore()
	rec := NewRecorder(store)
	subj := uuid.New().String()
	for i := 0; i < 3; i++ {
		_ = rec.Record(context.Background(), Metadata{
			DecisionID: uuid.New(), Component: "kb-30",
			AffectedSubjectID: subj, AffectedSubjectClass: "resident",
			Timestamp: time.Now().UTC(),
		})
	}
	list, _ := store.QueryBySubject(context.Background(), subj)
	if len(list) != 3 {
		t.Errorf("got %d entries for subject", len(list))
	}
}

// ---------------------------------------------------------------------------
// Augmentation test 1: IsValidPrinciple accepts P1..P7 only
// ---------------------------------------------------------------------------

func TestIsValidPrinciple(t *testing.T) {
	valid := []string{"P1", "P2", "P3", "P4", "P5", "P6", "P7"}
	for _, p := range valid {
		if !IsValidPrinciple(p) {
			t.Errorf("expected %q to be valid", p)
		}
	}
	invalid := []string{"P0", "P8", "p1", "P10", "", "P", "PRINCIPLE1"}
	for _, p := range invalid {
		if IsValidPrinciple(p) {
			t.Errorf("expected %q to be invalid", p)
		}
	}
}

// ---------------------------------------------------------------------------
// Augmentation test 2: Record sets Timestamp when zero
// ---------------------------------------------------------------------------

func TestRecorder_DefaultsTimestampWhenZero(t *testing.T) {
	store := NewInMemoryStore()
	rec := NewRecorder(store)
	id := uuid.New()

	before := time.Now().UTC()
	if err := rec.Record(context.Background(), Metadata{
		DecisionID:           id,
		Component:            "kb-30",
		DecisionType:         "recommendation_draft",
		AffectedSubjectID:    uuid.New().String(),
		AffectedSubjectClass: "resident",
		// Timestamp intentionally zero
	}); err != nil {
		t.Fatalf("record: %v", err)
	}
	after := time.Now().UTC()

	got, err := store.Get(context.Background(), id)
	if err != nil || got == nil {
		t.Fatalf("get: err=%v got=%v", err, got)
	}
	if got.Timestamp.IsZero() {
		t.Error("expected Timestamp to be set by Record(), got zero value")
	}
	if got.Timestamp.Before(before) || got.Timestamp.After(after) {
		t.Errorf("Timestamp %v not in expected window [%v, %v]", got.Timestamp, before, after)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func ptr(s string) *string { return &s }
