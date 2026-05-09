// Package dashboards — test suite for Surface 2: My Recommendations.
//
// VisibilityClass: PDP (Pharmacist-Default-Private)
package dashboards

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Plan verbatim tests (Task 5, Step 1)
// ---------------------------------------------------------------------------

func TestMyRecommendations_FilterByAuthor(t *testing.T) {
	author := uuid.New()
	src := &fakeRecSource{
		recs: []RecRow{
			{AuthorID: author, State: "drafted", ID: uuid.New()},
			{AuthorID: author, State: "implemented", ID: uuid.New()},
			{AuthorID: uuid.New(), State: "drafted", ID: uuid.New()}, // someone else's
		},
	}
	d := NewMyRecommendations(src)
	got, _ := d.For(context.Background(), author)
	if len(got) != 2 {
		t.Errorf("expected 2 own recs, got %d", len(got))
	}
}

func TestMyRecommendations_RejectedFramedAsLearning(t *testing.T) {
	author := uuid.New()
	src := &fakeRecSource{
		recs: []RecRow{{AuthorID: author, State: "rejected", ID: uuid.New(), RejectionReason: "GP preferred alternative"}},
	}
	d := NewMyRecommendations(src)
	got, _ := d.For(context.Background(), author)
	if got[0].Framing != "learning_opportunity" {
		t.Errorf("rejected rec should carry framing=learning_opportunity, got %q", got[0].Framing)
	}
}

// ---------------------------------------------------------------------------
// Augmentation 3: source error propagation
// ---------------------------------------------------------------------------

func TestMyRecommendations_PropagatesSourceError(t *testing.T) {
	sentinel := errors.New("source unavailable")
	src := &errRecSource{err: sentinel}
	d := NewMyRecommendations(src)
	_, err := d.For(context.Background(), uuid.New())
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error; got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Augmentation 4: empty source returns empty (not nil) slice
// ---------------------------------------------------------------------------

func TestMyRecommendations_EmptyResultIsEmptySlice(t *testing.T) {
	src := &fakeRecSource{recs: []RecRow{}}
	d := NewMyRecommendations(src)
	got, err := d.For(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(got))
	}
}

// ---------------------------------------------------------------------------
// Augmentation 5: context cancellation
// ---------------------------------------------------------------------------

func TestMyRecommendations_ContextCancellation(t *testing.T) {
	author := uuid.New()
	src := &fakeRecSource{
		recs: []RecRow{
			{AuthorID: author, State: "drafted", ID: uuid.New()},
		},
	}
	d := NewMyRecommendations(src)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before For() is called

	_, err := d.For(ctx, author)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled; got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test helper types
// ---------------------------------------------------------------------------

// fakeRecSource filters by AuthorID, simulating permission middleware from Phase 1a.
type fakeRecSource struct{ recs []RecRow }

func (f *fakeRecSource) ListByAuthor(_ context.Context, author uuid.UUID) ([]RecRow, error) {
	out := []RecRow{}
	for _, r := range f.recs {
		if r.AuthorID == author {
			out = append(out, r)
		}
	}
	return out, nil
}

// errRecSource always returns an error from ListByAuthor.
type errRecSource struct{ err error }

func (e *errRecSource) ListByAuthor(_ context.Context, _ uuid.UUID) ([]RecRow, error) {
	return nil, e.err
}
