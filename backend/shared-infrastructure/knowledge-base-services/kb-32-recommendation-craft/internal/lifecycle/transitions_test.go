package lifecycle

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/appropriateness"
)

// stubSource is a test AppropriatenessSource that returns a fixed Assessment.
type stubSource struct {
	assessment appropriateness.Assessment
	err        error
}

func (s stubSource) AssessFor(_ context.Context, _ uuid.UUID) (appropriateness.Assessment, error) {
	return s.assessment, s.err
}

func TestGate_PassingAssessment(t *testing.T) {
	// All dimensions at 3 → above HoldThreshold(2) → gate passes.
	src := stubSource{
		assessment: appropriateness.Assessment{
			ClinicalWarrant:        3,
			EvidenceSolidity:       3,
			AlternativesConsidered: 3,
			RestraintConsidered:    3,
			GoalsOfCareAlignment:   3,
		},
	}
	gate := NewGate(src)
	if err := gate.AdvanceDetectedToDrafted(context.Background(), uuid.New()); err != nil {
		t.Fatalf("expected nil (gate passes); got %v", err)
	}
}

func TestGate_FailingAssessment_ReturnsErrTransitionHeld(t *testing.T) {
	// One dimension at HoldThreshold → gate holds.
	src := stubSource{
		assessment: appropriateness.Assessment{
			ClinicalWarrant:        2, // at threshold → holds
			EvidenceSolidity:       3,
			AlternativesConsidered: 3,
			RestraintConsidered:    3,
			GoalsOfCareAlignment:   3,
		},
	}
	gate := NewGate(src)
	err := gate.AdvanceDetectedToDrafted(context.Background(), uuid.New())
	if !errors.Is(err, ErrTransitionHeld) {
		t.Fatalf("expected ErrTransitionHeld; got %v", err)
	}
}

func TestGate_SourceError_Propagates(t *testing.T) {
	sentinel := errors.New("substrate unavailable")
	src := stubSource{err: sentinel}
	gate := NewGate(src)
	err := gate.AdvanceDetectedToDrafted(context.Background(), uuid.New())
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected source error to propagate; got %v", err)
	}
}

func TestDefaultScorer_AlwaysPasses(t *testing.T) {
	scorer := DefaultScorer{}
	a, err := scorer.AssessFor(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("DefaultScorer.AssessFor error: %v", err)
	}
	if err := appropriateness.Check(a); err != nil {
		t.Fatalf("DefaultScorer assessment should pass gate; got %v", err)
	}
	// All dimensions must be > HoldThreshold (2).
	for _, score := range []int{
		a.ClinicalWarrant,
		a.EvidenceSolidity,
		a.AlternativesConsidered,
		a.RestraintConsidered,
		a.GoalsOfCareAlignment,
	} {
		if score <= appropriateness.HoldThreshold {
			t.Errorf("DefaultScorer dimension score %d ≤ HoldThreshold %d",
				score, appropriateness.HoldThreshold)
		}
	}
}

func TestGate_AllDimensionsAtOne_Held(t *testing.T) {
	src := stubSource{
		assessment: appropriateness.Assessment{
			ClinicalWarrant:        1,
			EvidenceSolidity:       1,
			AlternativesConsidered: 1,
			RestraintConsidered:    1,
			GoalsOfCareAlignment:   1,
		},
	}
	gate := NewGate(src)
	err := gate.AdvanceDetectedToDrafted(context.Background(), uuid.New())
	if !errors.Is(err, ErrTransitionHeld) {
		t.Fatalf("expected ErrTransitionHeld for all-1 assessment; got %v", err)
	}
}

func TestGate_MaxScores_Passes(t *testing.T) {
	src := stubSource{
		assessment: appropriateness.Assessment{
			ClinicalWarrant:        5,
			EvidenceSolidity:       5,
			AlternativesConsidered: 5,
			RestraintConsidered:    5,
			GoalsOfCareAlignment:   5,
		},
	}
	gate := NewGate(src)
	if err := gate.AdvanceDetectedToDrafted(context.Background(), uuid.New()); err != nil {
		t.Fatalf("expected nil for all-5 assessment; got %v", err)
	}
}
