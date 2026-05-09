package erm

import (
	"context"
	"testing"
)

func TestERM_RoutesToRegisteredReasoner(t *testing.T) {
	called := false
	r := ReasonerFunc(func(_ context.Context, dp DecisionPoint) (Outcome, []Concern) {
		called = true
		return OutcomeApprove, nil
	})
	m := NewModule()
	m.Register(DecisionTypeRecommendationDraft, r)

	out, _, err := m.Review(context.Background(), DecisionPoint{
		Component: "kb-30", DecisionType: DecisionTypeRecommendationDraft,
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !called {
		t.Errorf("registered reasoner was not invoked")
	}
	if out != OutcomeApprove {
		t.Errorf("outcome = %v, want approve", out)
	}
}

func TestERM_RejectsUnknownDecisionType(t *testing.T) {
	m := NewModule()
	_, _, err := m.Review(context.Background(), DecisionPoint{DecisionType: "bogus"})
	if err == nil {
		t.Errorf("unknown decision type should error")
	}
}

// TestModule_RegisterReplacesExisting verifies that registering the same
// DecisionType twice replaces rather than appends the reasoner.
func TestModule_RegisterReplacesExisting(t *testing.T) {
	calls := 0
	first := ReasonerFunc(func(_ context.Context, dp DecisionPoint) (Outcome, []Concern) {
		calls++
		return OutcomeReject, nil
	})
	second := ReasonerFunc(func(_ context.Context, dp DecisionPoint) (Outcome, []Concern) {
		calls += 10
		return OutcomeApprove, nil
	})

	m := NewModule()
	m.Register(DecisionTypeRecommendationDraft, first)
	m.Register(DecisionTypeRecommendationDraft, second) // replaces first

	out, _, err := m.Review(context.Background(), DecisionPoint{
		DecisionType: DecisionTypeRecommendationDraft,
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	// Only the second reasoner should have been called (calls += 10).
	if calls != 10 {
		t.Errorf("expected second reasoner only (calls=10), got calls=%d", calls)
	}
	if out != OutcomeApprove {
		t.Errorf("expected Approve from second reasoner, got %v", out)
	}
}

// TestModule_ReviewSurfacesReasonerOutcome confirms all four Outcome values
// flow through Review without being mutated.
func TestModule_ReviewSurfacesReasonerOutcome(t *testing.T) {
	outcomes := []Outcome{
		OutcomeApprove,
		OutcomeApproveWithMonitoring,
		OutcomeHold,
		OutcomeReject,
	}
	for _, want := range outcomes {
		want := want
		t.Run(string(want), func(t *testing.T) {
			m := NewModule()
			m.Register(DecisionTypeAuthorisation, ReasonerFunc(func(_ context.Context, _ DecisionPoint) (Outcome, []Concern) {
				return want, nil
			}))
			got, _, err := m.Review(context.Background(), DecisionPoint{
				DecisionType: DecisionTypeAuthorisation,
			})
			if err != nil {
				t.Fatalf("review: %v", err)
			}
			if got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestIsValidOutcome(t *testing.T) {
	valid := []string{"approve", "approve_with_monitoring", "hold", "reject"}
	for _, s := range valid {
		if !IsValidOutcome(s) {
			t.Errorf("IsValidOutcome(%q) = false, want true", s)
		}
	}
	if IsValidOutcome("bogus") {
		t.Errorf("IsValidOutcome(bogus) = true, want false")
	}
	if IsValidOutcome("") {
		t.Errorf("IsValidOutcome('') = true, want false")
	}
}
