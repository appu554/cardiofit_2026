package appropriateness_test

import (
	"errors"
	"testing"

	"github.com/cardiofit/kb32/internal/appropriateness"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// allPass returns an Assessment where every dimension is set to score, which
// should pass the gate as long as score > HoldThreshold.
func allPass(score int) appropriateness.Assessment {
	return appropriateness.Assessment{
		ClinicalWarrant:        score,
		EvidenceSolidity:       score,
		AlternativesConsidered: score,
		RestraintConsidered:    score,
		GoalsOfCareAlignment:   score,
	}
}

// ---------------------------------------------------------------------------
// TestCheck_HoldPath — each dimension individually at boundary (score == 2)
// ---------------------------------------------------------------------------

func TestCheck_HoldPath_ClinicalWarrant_Score2(t *testing.T) {
	a := allPass(3)
	a.ClinicalWarrant = 2
	err := appropriateness.Check(a)
	if !errors.Is(err, appropriateness.ErrAppropriatenessHold) {
		t.Fatalf("expected ErrAppropriatenessHold for ClinicalWarrant=2, got %v", err)
	}
}

func TestCheck_HoldPath_EvidenceSolidity_Score2(t *testing.T) {
	a := allPass(3)
	a.EvidenceSolidity = 2
	err := appropriateness.Check(a)
	if !errors.Is(err, appropriateness.ErrAppropriatenessHold) {
		t.Fatalf("expected ErrAppropriatenessHold for EvidenceSolidity=2, got %v", err)
	}
}

func TestCheck_HoldPath_AlternativesConsidered_Score2(t *testing.T) {
	a := allPass(3)
	a.AlternativesConsidered = 2
	err := appropriateness.Check(a)
	if !errors.Is(err, appropriateness.ErrAppropriatenessHold) {
		t.Fatalf("expected ErrAppropriatenessHold for AlternativesConsidered=2, got %v", err)
	}
}

func TestCheck_HoldPath_RestraintConsidered_Score2(t *testing.T) {
	a := allPass(3)
	a.RestraintConsidered = 2
	err := appropriateness.Check(a)
	if !errors.Is(err, appropriateness.ErrAppropriatenessHold) {
		t.Fatalf("expected ErrAppropriatenessHold for RestraintConsidered=2, got %v", err)
	}
}

func TestCheck_HoldPath_GoalsOfCareAlignment_Score2(t *testing.T) {
	a := allPass(3)
	a.GoalsOfCareAlignment = 2
	err := appropriateness.Check(a)
	if !errors.Is(err, appropriateness.ErrAppropriatenessHold) {
		t.Fatalf("expected ErrAppropriatenessHold for GoalsOfCareAlignment=2, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestCheck_HoldPath — each dimension at score 1 (well below threshold)
// ---------------------------------------------------------------------------

func TestCheck_HoldPath_ClinicalWarrant_Score1(t *testing.T) {
	a := allPass(3)
	a.ClinicalWarrant = 1
	if err := appropriateness.Check(a); !errors.Is(err, appropriateness.ErrAppropriatenessHold) {
		t.Fatalf("expected ErrAppropriatenessHold for ClinicalWarrant=1, got %v", err)
	}
}

func TestCheck_HoldPath_EvidenceSolidity_Score1(t *testing.T) {
	a := allPass(3)
	a.EvidenceSolidity = 1
	if err := appropriateness.Check(a); !errors.Is(err, appropriateness.ErrAppropriatenessHold) {
		t.Fatalf("expected ErrAppropriatenessHold for EvidenceSolidity=1, got %v", err)
	}
}

func TestCheck_HoldPath_AlternativesConsidered_Score1(t *testing.T) {
	a := allPass(3)
	a.AlternativesConsidered = 1
	if err := appropriateness.Check(a); !errors.Is(err, appropriateness.ErrAppropriatenessHold) {
		t.Fatalf("expected ErrAppropriatenessHold for AlternativesConsidered=1, got %v", err)
	}
}

func TestCheck_HoldPath_RestraintConsidered_Score1(t *testing.T) {
	a := allPass(3)
	a.RestraintConsidered = 1
	if err := appropriateness.Check(a); !errors.Is(err, appropriateness.ErrAppropriatenessHold) {
		t.Fatalf("expected ErrAppropriatenessHold for RestraintConsidered=1, got %v", err)
	}
}

func TestCheck_HoldPath_GoalsOfCareAlignment_Score1(t *testing.T) {
	a := allPass(3)
	a.GoalsOfCareAlignment = 1
	if err := appropriateness.Check(a); !errors.Is(err, appropriateness.ErrAppropriatenessHold) {
		t.Fatalf("expected ErrAppropriatenessHold for GoalsOfCareAlignment=1, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestCheck_AllPass — all dimensions at 3 (first score above threshold)
// ---------------------------------------------------------------------------

func TestCheck_AllPass_Score3(t *testing.T) {
	a := allPass(3)
	if err := appropriateness.Check(a); err != nil {
		t.Fatalf("expected nil for all dims=3, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestCheck_Boundary — exact boundary between hold and pass
// ---------------------------------------------------------------------------

// Score 2 must hold; score 3 must pass. Both tested simultaneously.
func TestCheck_Boundary_Score2Holds_Score3Passes(t *testing.T) {
	hold := allPass(3)
	hold.ClinicalWarrant = 2
	if err := appropriateness.Check(hold); !errors.Is(err, appropriateness.ErrAppropriatenessHold) {
		t.Fatalf("score 2 boundary: expected ErrAppropriatenessHold, got %v", err)
	}

	pass := allPass(3) // all at exactly 3
	if err := appropriateness.Check(pass); err != nil {
		t.Fatalf("score 3 boundary: expected nil, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestPassesGate — sugar method mirrors Check
// ---------------------------------------------------------------------------

func TestPassesGate_HappyPath(t *testing.T) {
	a := allPass(4)
	if !a.PassesGate() {
		t.Fatal("expected PassesGate() == true for all dims=4")
	}
}

func TestPassesGate_HoldPath(t *testing.T) {
	a := allPass(3)
	a.EvidenceSolidity = 2
	if a.PassesGate() {
		t.Fatal("expected PassesGate() == false when EvidenceSolidity=2")
	}
}

// ---------------------------------------------------------------------------
// TestValidateScores — invalid score on each dimension
// ---------------------------------------------------------------------------

func TestValidateScores_Valid(t *testing.T) {
	a := allPass(3)
	if err := a.ValidateScores(); err != nil {
		t.Fatalf("expected nil for valid assessment, got %v", err)
	}
}

func TestValidateScores_InvalidClinicalWarrant(t *testing.T) {
	a := allPass(3)
	a.ClinicalWarrant = 0
	if err := a.ValidateScores(); !errors.Is(err, appropriateness.ErrInvalidScore) {
		t.Fatalf("expected ErrInvalidScore for ClinicalWarrant=0, got %v", err)
	}
}

func TestValidateScores_InvalidEvidenceSolidity(t *testing.T) {
	a := allPass(3)
	a.EvidenceSolidity = 6
	if err := a.ValidateScores(); !errors.Is(err, appropriateness.ErrInvalidScore) {
		t.Fatalf("expected ErrInvalidScore for EvidenceSolidity=6, got %v", err)
	}
}

func TestValidateScores_InvalidAlternativesConsidered(t *testing.T) {
	a := allPass(3)
	a.AlternativesConsidered = -1
	if err := a.ValidateScores(); !errors.Is(err, appropriateness.ErrInvalidScore) {
		t.Fatalf("expected ErrInvalidScore for AlternativesConsidered=-1, got %v", err)
	}
}

func TestValidateScores_InvalidRestraintConsidered(t *testing.T) {
	a := allPass(3)
	a.RestraintConsidered = 0
	if err := a.ValidateScores(); !errors.Is(err, appropriateness.ErrInvalidScore) {
		t.Fatalf("expected ErrInvalidScore for RestraintConsidered=0, got %v", err)
	}
}

func TestValidateScores_InvalidGoalsOfCareAlignment(t *testing.T) {
	a := allPass(3)
	a.GoalsOfCareAlignment = 10
	if err := a.ValidateScores(); !errors.Is(err, appropriateness.ErrInvalidScore) {
		t.Fatalf("expected ErrInvalidScore for GoalsOfCareAlignment=10, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestLowestDimension
// ---------------------------------------------------------------------------

func TestLowestDimension_IdentifiesLowest(t *testing.T) {
	a := appropriateness.Assessment{
		ClinicalWarrant:        5,
		EvidenceSolidity:       1,
		AlternativesConsidered: 4,
		RestraintConsidered:    3,
		GoalsOfCareAlignment:   5,
	}
	name, score := a.LowestDimension()
	if name != "evidence_solidity" {
		t.Errorf("expected dimension name 'evidence_solidity', got %q", name)
	}
	if score != 1 {
		t.Errorf("expected lowest score 1, got %d", score)
	}
}

func TestLowestDimension_TieBreak_ReturnsFirst(t *testing.T) {
	// ClinicalWarrant and RestraintConsidered both at 2; ClinicalWarrant is first.
	a := appropriateness.Assessment{
		ClinicalWarrant:        2,
		EvidenceSolidity:       4,
		AlternativesConsidered: 4,
		RestraintConsidered:    2,
		GoalsOfCareAlignment:   4,
	}
	name, score := a.LowestDimension()
	if name != "clinical_warrant" {
		t.Errorf("expected tie to return first dim 'clinical_warrant', got %q", name)
	}
	if score != 2 {
		t.Errorf("expected score 2, got %d", score)
	}
}

func TestLowestDimension_AllSameScore(t *testing.T) {
	a := allPass(3)
	name, score := a.LowestDimension()
	if name != "clinical_warrant" {
		t.Errorf("expected first dim 'clinical_warrant' when all equal, got %q", name)
	}
	if score != 3 {
		t.Errorf("expected score 3, got %d", score)
	}
}

// ---------------------------------------------------------------------------
// TestIsValidScore — boundary tests
// ---------------------------------------------------------------------------

func TestIsValidScore_Boundaries(t *testing.T) {
	cases := []struct {
		n       int
		wantOK  bool
	}{
		{0, false},
		{1, true},
		{2, true},
		{3, true},
		{4, true},
		{5, true},
		{6, false},
		{-1, false},
		{100, false},
	}
	for _, tc := range cases {
		got := appropriateness.IsValidScore(tc.n)
		if got != tc.wantOK {
			t.Errorf("IsValidScore(%d) = %v, want %v", tc.n, got, tc.wantOK)
		}
	}
}
