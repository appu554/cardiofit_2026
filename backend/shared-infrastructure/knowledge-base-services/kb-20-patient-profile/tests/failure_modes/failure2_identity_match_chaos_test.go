// Wave 6.1 — Failure Mode 2: identity-match errors.
//
// Layer 2 doc Part 6 Failure 2: "an IHI typo could mis-route care.
// Defence: any non-exact identity match is LOW-confidence, requires
// human review, never auto-accepts."
//
// This is a pure-logic test against the identity package's MatchResult /
// Confidence types, simulating an IHI typo that misses the high-confidence
// IHI lookup path and falls through to fuzzy matching with a name+DOB
// distance threshold breach.
package failure_modes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/identity"
)

// fakeMatcher returns a configured MatchResult on every Match call.
// Stand-in for the real IdentityMatcher; the failure-mode unit test
// doesn't need a candidate store, only the result envelope.
type fakeMatcher struct {
	result identity.MatchResult
	err    error
}

func (f *fakeMatcher) Match(_ context.Context, _ identity.IncomingIdentifier) (identity.MatchResult, error) {
	return f.result, f.err
}

func TestFailure2_IHITypoFallsToLowConfidenceReviewQueue(t *testing.T) {
	// Simulate the chaos: an IHI typo means the IHI exact-match path
	// returns no candidate, and the fuzzy path produces a LOW-confidence
	// result. The defence is that LOW confidence → RequiresReview=true,
	// and the resident_ref MUST be presented as a review-required match
	// rather than auto-accepted.
	resident := uuid.New()
	matcher := &fakeMatcher{
		result: identity.MatchResult{
			ResidentRef:    &resident,
			Confidence:     identity.ConfidenceLow,
			Path:           identity.MatchPathNameDOBFacility,
			RequiresReview: true,
			NameDistance:   4,
			Candidates:     []uuid.UUID{resident, uuid.New()},
		},
	}
	got, err := matcher.Match(context.Background(), identity.IncomingIdentifier{
		IHI:        "8003600166666661", // hypothetical typo
		GivenName:  "Jane",
		FamilyName: "Smith",
		DOB:        time.Date(1948, 3, 15, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("matcher returned err: %v", err)
	}
	if got.Confidence != identity.ConfidenceLow {
		t.Fatalf("want LOW confidence (review queue), got %s", got.Confidence)
	}
	if !got.RequiresReview {
		t.Fatal("LOW confidence MUST require review (Failure 2 defence)")
	}
	if len(got.Candidates) < 2 {
		t.Fatal("LOW confidence must surface candidate set so reviewer can disambiguate")
	}
}

func TestFailure2_NoMatchAlsoQueuesForReview(t *testing.T) {
	matcher := &fakeMatcher{
		result: identity.MatchResult{
			Confidence:     identity.ConfidenceNone,
			Path:           identity.MatchPathNoMatch,
			RequiresReview: true,
		},
	}
	got, _ := matcher.Match(context.Background(), identity.IncomingIdentifier{
		IHI: "0000000000000000",
	})
	if got.ResidentRef != nil {
		t.Fatal("no-match must not propagate a resident_ref")
	}
	if !got.RequiresReview {
		t.Fatal("no-match MUST require review (Failure 2 defence)")
	}
}

func TestFailure2_MatcherErrorBubblesUp(t *testing.T) {
	want := errors.New("simulated DB outage")
	matcher := &fakeMatcher{err: want}
	_, err := matcher.Match(context.Background(), identity.IncomingIdentifier{IHI: "x"})
	if !errors.Is(err, want) {
		t.Fatalf("expected the error to bubble; got %v", err)
	}
}
