package ethics_log

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestLogger_AppendAndQuery is the verbatim plan test (Task 2 §Step 1-3).
func TestLogger_AppendAndQuery(t *testing.T) {
	store := NewInMemoryStore()
	l := NewLogger(store)
	q := NewQuerier(store)

	decisionID := uuid.New()
	if err := l.Append(context.Background(), Entry{
		DecisionID:  decisionID,
		EntryType:   EntryTypePatternDetected,
		Severity:    3,
		Description: "acceptance-appropriateness divergence",
		Status:      StatusOpen,
	}); err != nil {
		t.Fatalf("append: %v", err)
	}

	list, _ := q.ByDecision(context.Background(), decisionID)
	if len(list) != 1 || list[0].Severity != 3 {
		t.Errorf("query roundtrip fail: %v", list)
	}

	openSev3, _ := q.OpenAtSeverity(context.Background(), 3)
	if len(openSev3) != 1 {
		t.Errorf("open-at-severity-3 query: %d", len(openSev3))
	}
	_ = time.Now()
}

// TestLogger_DefaultsTimestampsAndIDAndStatus verifies that Append fills in zero
// values for ID, CreatedAt, UpdatedAt, and Status before writing to the store.
func TestLogger_DefaultsTimestampsAndIDAndStatus(t *testing.T) {
	store := NewInMemoryStore()
	l := NewLogger(store)

	before := time.Now().UTC()
	err := l.Append(context.Background(), Entry{
		DecisionID:  uuid.New(),
		EntryType:   EntryTypeConcernFlagged,
		Severity:    2,
		Description: "zero-value defaults test",
		// ID, CreatedAt, UpdatedAt, Status intentionally left zero
	})
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	entries, _ := store.List(context.Background())
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]

	if e.ID == uuid.Nil {
		t.Error("ID should not be nil after Append")
	}
	if e.Status != StatusOpen {
		t.Errorf("Status = %q, want %q", e.Status, StatusOpen)
	}
	if e.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if e.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
	if e.CreatedAt.Before(before) || e.CreatedAt.After(after) {
		t.Errorf("CreatedAt %v not in [%v, %v]", e.CreatedAt, before, after)
	}
}

// TestIsValidEntryType verifies the IsValidEntryType helper covers all five
// canonical values and rejects unknown strings.
func TestIsValidEntryType(t *testing.T) {
	valid := []string{"decision", "concern_flagged", "review_requested", "pattern_detected", "incident"}
	for _, v := range valid {
		if !IsValidEntryType(v) {
			t.Errorf("IsValidEntryType(%q) = false, want true", v)
		}
	}
	invalid := []string{"", "unknown", "DECISION", "Incident"}
	for _, v := range invalid {
		if IsValidEntryType(v) {
			t.Errorf("IsValidEntryType(%q) = true, want false", v)
		}
	}
}

// TestIsValidStatus verifies the IsValidStatus helper covers all five canonical
// values and rejects unknown strings.
func TestIsValidStatus(t *testing.T) {
	valid := []string{"open", "investigating", "remediated", "verified", "closed"}
	for _, v := range valid {
		if !IsValidStatus(v) {
			t.Errorf("IsValidStatus(%q) = false, want true", v)
		}
	}
	invalid := []string{"", "unknown", "Open", "CLOSED"}
	for _, v := range invalid {
		if IsValidStatus(v) {
			t.Errorf("IsValidStatus(%q) = true, want false", v)
		}
	}
}

// TestInMemoryStore_RaceCondition verifies that InMemoryStore is safe for concurrent
// Append and List calls (requires -race flag).
func TestInMemoryStore_RaceCondition(t *testing.T) {
	store := NewInMemoryStore()
	l := NewLogger(store)
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 50; i++ {
			_ = l.Append(ctx, Entry{
				DecisionID:  uuid.New(),
				EntryType:   EntryTypeDecision,
				Severity:    1,
				Description: "concurrent write",
			})
		}
	}()

	for i := 0; i < 50; i++ {
		_, _ = store.List(ctx)
	}
	<-done
}

// TestLogger_MultipleEntriesSameDecision verifies that multiple entries linked
// to the same DecisionID are all returned by ByDecision.
func TestLogger_MultipleEntriesSameDecision(t *testing.T) {
	store := NewInMemoryStore()
	l := NewLogger(store)
	q := NewQuerier(store)
	ctx := context.Background()

	decisionID := uuid.New()
	otherID := uuid.New()

	for _, et := range []EntryType{EntryTypeDecision, EntryTypeConcernFlagged, EntryTypeReviewRequested} {
		_ = l.Append(ctx, Entry{DecisionID: decisionID, EntryType: et, Severity: 1, Description: "linked"})
	}
	_ = l.Append(ctx, Entry{DecisionID: otherID, EntryType: EntryTypeIncident, Severity: 5, Description: "other"})

	results, err := q.ByDecision(ctx, decisionID)
	if err != nil {
		t.Fatalf("ByDecision: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("got %d entries for decisionID, want 3", len(results))
	}
}
