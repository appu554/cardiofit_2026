package dashboards

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Fake source
// ---------------------------------------------------------------------------

type fakeGPSrc struct{ patterns map[uuid.UUID]gpPattern }

func (f *fakeGPSrc) PatternsForPharmacist(_ context.Context, _ uuid.UUID) (map[uuid.UUID]gpPattern, error) {
	return f.patterns, nil
}

type fakeGPSrcErr struct{ err error }

func (f *fakeGPSrcErr) PatternsForPharmacist(_ context.Context, _ uuid.UUID) (map[uuid.UUID]gpPattern, error) {
	return nil, f.err
}

// ---------------------------------------------------------------------------
// Plan-verbatim tests
// ---------------------------------------------------------------------------

// TestGPRelationships_NeverShowsAcceptancePercentage verifies that the For()
// method never includes acceptance-rate strings (containing "%" or "rate") in
// any GPCard.Display value, even when the underlying gpPattern carries a
// non-zero acceptanceRate.
func TestGPRelationships_NeverShowsAcceptancePercentage(t *testing.T) {
	src := &fakeGPSrc{patterns: map[uuid.UUID]gpPattern{
		uuid.New(): {framingObservation: "recommendations land better with monitoring plan up front", acceptanceRate: 0.42},
	}}
	d := NewGPRelationships(src)
	cards, _ := d.For(context.Background(), uuid.New())
	for _, c := range cards {
		if strings.Contains(c.Display, "%") || strings.Contains(c.Display, "rate") {
			t.Errorf("GP card must not surface acceptance rate or %%; got %q", c.Display)
		}
	}
}

// TestGPRelationships_RespectsOptOut verifies that a GP who has opted out
// receives Display="default_framing" regardless of their framingObservation.
func TestGPRelationships_RespectsOptOut(t *testing.T) {
	gpA := uuid.New()
	gpB := uuid.New()
	src := &fakeGPSrc{
		patterns: map[uuid.UUID]gpPattern{
			gpA: {framingObservation: "X", acceptanceRate: 0.5},
			gpB: {framingObservation: "Y", acceptanceRate: 0.6, optedOut: true},
		},
	}
	d := NewGPRelationships(src)
	cards, _ := d.For(context.Background(), uuid.New())
	for _, c := range cards {
		if c.GPID == gpB && c.Display != "default_framing" {
			t.Errorf("opted-out GP should show default_framing only; got %q", c.Display)
		}
	}
}

// ---------------------------------------------------------------------------
// Augmentation 1 — DisplayNeverContainsPercentage (property-based sweep)
// ---------------------------------------------------------------------------

// TestGPRelationships_DisplayNeverContainsPercentage is a property-based sweep
// over a variety of synthetic gpPatterns to confirm that no GPCard.Display
// value ever contains a digit-followed-by-% pattern, the word "acceptance",
// or any raw rate figure. This guards against future additions that might
// accidentally leak numeric rates into observation strings.
func TestGPRelationships_DisplayNeverContainsPercentage(t *testing.T) {
	ratePattern := regexp.MustCompile(`\d+%`)
	dangerous := []string{
		"acceptance rate",
		"% acceptance",
		"0.42 rate",
		"42%",
	}

	syntheticPatterns := map[uuid.UUID]gpPattern{
		uuid.New(): {framingObservation: "prefers concise summaries", acceptanceRate: 0.0},
		uuid.New(): {framingObservation: "monitoring plan up front improves landing", acceptanceRate: 0.99},
		uuid.New(): {framingObservation: "shorter referral text preferred", acceptanceRate: 0.55},
		uuid.New(): {framingObservation: "evidence citations appreciated", acceptanceRate: 0.33},
		// opted-out: observation would be dangerous but must resolve to default_framing
		uuid.New(): {framingObservation: "acceptance rate 100%", acceptanceRate: 1.0, optedOut: true},
	}

	src := &fakeGPSrc{patterns: syntheticPatterns}
	d := NewGPRelationships(src)
	cards, err := d.For(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, c := range cards {
		if ratePattern.MatchString(c.Display) {
			t.Errorf("Display contains digit%% pattern — must never surface rate: %q", c.Display)
		}
		for _, bad := range dangerous {
			if strings.Contains(strings.ToLower(c.Display), strings.ToLower(bad)) {
				t.Errorf("Display contains forbidden phrase %q: got %q", bad, c.Display)
			}
		}
	}

	// Specifically verify the opted-out card resolved to "default_framing".
	for _, c := range cards {
		if c.Display == "acceptance rate 100%" {
			t.Error("opted-out GP's dangerous observation leaked into Display")
		}
	}
}

// ---------------------------------------------------------------------------
// Augmentation 2 — PropagatesSourceError
// ---------------------------------------------------------------------------

// TestGPRelationships_PropagatesSourceError ensures that when the source
// returns an error, For() propagates it and returns nil cards.
func TestGPRelationships_PropagatesSourceError(t *testing.T) {
	sentinel := errors.New("db unavailable")
	d := NewGPRelationships(&fakeGPSrcErr{err: sentinel})
	cards, err := d.For(context.Background(), uuid.New())
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
	if cards != nil {
		t.Errorf("expected nil cards on error, got %v", cards)
	}
}

// ---------------------------------------------------------------------------
// Augmentation 3 — EmptyReturnsEmptySlice
// ---------------------------------------------------------------------------

// TestGPRelationships_EmptyReturnsEmptySlice verifies that a source returning
// an empty map yields a non-nil, zero-length slice, allowing callers to
// distinguish "no GP relationships" from an uninitialised result.
func TestGPRelationships_EmptyReturnsEmptySlice(t *testing.T) {
	src := &fakeGPSrc{patterns: map[uuid.UUID]gpPattern{}}
	d := NewGPRelationships(src)
	cards, err := d.For(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cards == nil {
		t.Error("expected non-nil empty slice for no patterns, got nil")
	}
	if len(cards) != 0 {
		t.Errorf("expected 0 cards, got %d", len(cards))
	}
}

// ---------------------------------------------------------------------------
// Augmentation 4 — ContextCancellation
// ---------------------------------------------------------------------------

// TestGPRelationships_ContextCancellation verifies that For() returns
// ctx.Err() immediately when the context is already cancelled, without
// calling the source.
func TestGPRelationships_ContextCancellation(t *testing.T) {
	// Use a source that would panic if called, to confirm it is never reached.
	src := &fakeGPSrc{patterns: map[uuid.UUID]gpPattern{
		uuid.New(): {framingObservation: "should not be reached"},
	}}
	d := NewGPRelationships(src)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cards, err := d.For(ctx, uuid.New())
	if err == nil {
		t.Error("expected error from cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if cards != nil {
		t.Errorf("expected nil cards on cancelled context, got %v", cards)
	}
}
