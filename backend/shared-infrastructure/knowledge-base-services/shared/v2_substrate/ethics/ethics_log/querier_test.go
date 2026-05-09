package ethics_log

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestQuerier_ByTimeWindow verifies that ByTimeWindow filters entries correctly
// based on CreatedAt within [since, until].
func TestQuerier_ByTimeWindow(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Manually insert entries with known CreatedAt values (bypassing Logger defaults
	// so we control timestamps precisely).
	base := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)

	entries := []Entry{
		{ID: uuid.New(), DecisionID: uuid.New(), EntryType: EntryTypeDecision, Severity: 1,
			Description: "early", Status: StatusOpen,
			CreatedAt: base.Add(-2 * time.Hour), UpdatedAt: base.Add(-2 * time.Hour)},
		{ID: uuid.New(), DecisionID: uuid.New(), EntryType: EntryTypeConcernFlagged, Severity: 2,
			Description: "in-window-1", Status: StatusOpen,
			CreatedAt: base, UpdatedAt: base},
		{ID: uuid.New(), DecisionID: uuid.New(), EntryType: EntryTypeReviewRequested, Severity: 3,
			Description: "in-window-2", Status: StatusInvestigating,
			CreatedAt: base.Add(30 * time.Minute), UpdatedAt: base.Add(30 * time.Minute)},
		{ID: uuid.New(), DecisionID: uuid.New(), EntryType: EntryTypeIncident, Severity: 5,
			Description: "late", Status: StatusOpen,
			CreatedAt: base.Add(4 * time.Hour), UpdatedAt: base.Add(4 * time.Hour)},
	}
	for _, e := range entries {
		if err := store.Append(ctx, e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	q := NewQuerier(store)
	since := base
	until := base.Add(time.Hour)

	results, err := q.ByTimeWindow(ctx, since, until)
	if err != nil {
		t.Fatalf("ByTimeWindow: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results in window, want 2", len(results))
	}
	for _, r := range results {
		if r.CreatedAt.Before(since) || r.CreatedAt.After(until) {
			t.Errorf("entry %v has CreatedAt %v outside window [%v, %v]",
				r.ID, r.CreatedAt, since, until)
		}
	}
}

// TestQuerier_ByEntryType verifies that ByEntryType filters entries correctly
// based on EntryType.
func TestQuerier_ByEntryType(t *testing.T) {
	store := NewInMemoryStore()
	l := NewLogger(store)
	q := NewQuerier(store)
	ctx := context.Background()

	types := []EntryType{
		EntryTypeDecision,
		EntryTypeDecision,
		EntryTypeConcernFlagged,
		EntryTypePatternDetected,
		EntryTypeIncident,
	}
	for _, et := range types {
		_ = l.Append(ctx, Entry{
			DecisionID:  uuid.New(),
			EntryType:   et,
			Severity:    1,
			Description: "type filter test",
		})
	}

	decisions, err := q.ByEntryType(ctx, EntryTypeDecision)
	if err != nil {
		t.Fatalf("ByEntryType(decision): %v", err)
	}
	if len(decisions) != 2 {
		t.Errorf("got %d decision entries, want 2", len(decisions))
	}

	incidents, err := q.ByEntryType(ctx, EntryTypeIncident)
	if err != nil {
		t.Fatalf("ByEntryType(incident): %v", err)
	}
	if len(incidents) != 1 {
		t.Errorf("got %d incident entries, want 1", len(incidents))
	}

	reviews, err := q.ByEntryType(ctx, EntryTypeReviewRequested)
	if err != nil {
		t.Fatalf("ByEntryType(review_requested): %v", err)
	}
	if len(reviews) != 0 {
		t.Errorf("got %d review_requested entries, want 0", len(reviews))
	}
}

// TestQuerier_OpenAtSeverity_MultipleStatuses verifies that OpenAtSeverity only
// returns StatusOpen entries, excluding investigating/remediated/etc.
func TestQuerier_OpenAtSeverity_MultipleStatuses(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	statuses := []Status{StatusOpen, StatusInvestigating, StatusRemediated, StatusVerified, StatusClosed}
	for _, s := range statuses {
		_ = store.Append(ctx, Entry{
			ID:          uuid.New(),
			DecisionID:  uuid.New(),
			EntryType:   EntryTypeConcernFlagged,
			Severity:    4,
			Description: "status filter",
			Status:      s,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		})
	}

	q := NewQuerier(store)
	results, err := q.OpenAtSeverity(ctx, 4)
	if err != nil {
		t.Fatalf("OpenAtSeverity: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1 (only StatusOpen)", len(results))
	}
	if len(results) == 1 && results[0].Status != StatusOpen {
		t.Errorf("result status = %q, want %q", results[0].Status, StatusOpen)
	}
}
