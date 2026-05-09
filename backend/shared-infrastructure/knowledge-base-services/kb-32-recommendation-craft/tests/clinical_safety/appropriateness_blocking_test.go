// Package clinical_safety_test validates the clinical-safety gate (Stage 4)
// for the appropriateness dimension blocking contract.
//
// Recommendation Craft Guidelines Part 13 — clinical safety test category.
// VisibilityClass: AD — clinical-safety gate per Guidelines §9
package clinical_safety_test

import (
	"errors"
	"testing"

	"github.com/cardiofit/kb32/internal/appropriateness"
)

// TestAppropriatenessBlocking_EachDimensionHolds asserts that a score of 2
// on any single one of the five appropriateness dimensions causes Check to
// return ErrAppropriatenessHold, while every other dimension is set to 5
// (safely above the hold threshold).
//
// Guidelines §13 hard cap: any dimension ≤ HoldThreshold (2) must hold the
// recommendation in `detected` state; it must NOT advance to `drafted`.
func TestAppropriatenessBlocking_EachDimensionHolds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    appropriateness.Assessment
	}{
		{
			"clinical_warrant=2",
			appropriateness.Assessment{
				ClinicalWarrant: 2, EvidenceSolidity: 5,
				AlternativesConsidered: 5, RestraintConsidered: 5, GoalsOfCareAlignment: 5,
			},
		},
		{
			"evidence_solidity=2",
			appropriateness.Assessment{
				ClinicalWarrant: 5, EvidenceSolidity: 2,
				AlternativesConsidered: 5, RestraintConsidered: 5, GoalsOfCareAlignment: 5,
			},
		},
		{
			"alternatives_considered=2",
			appropriateness.Assessment{
				ClinicalWarrant: 5, EvidenceSolidity: 5,
				AlternativesConsidered: 2, RestraintConsidered: 5, GoalsOfCareAlignment: 5,
			},
		},
		{
			"restraint_considered=2",
			appropriateness.Assessment{
				ClinicalWarrant: 5, EvidenceSolidity: 5,
				AlternativesConsidered: 5, RestraintConsidered: 2, GoalsOfCareAlignment: 5,
			},
		},
		{
			"goals_of_care_alignment=2",
			appropriateness.Assessment{
				ClinicalWarrant: 5, EvidenceSolidity: 5,
				AlternativesConsidered: 5, RestraintConsidered: 5, GoalsOfCareAlignment: 2,
			},
		},
	}

	for _, c := range cases {
		c := c // capture loop var
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if err := appropriateness.Check(c.a); !errors.Is(err, appropriateness.ErrAppropriatenessHold) {
				t.Errorf("expected ErrAppropriatenessHold for %s; got %v", c.name, err)
			}
		})
	}
}

// TestAppropriatenessBlocking_AllPassAdvances asserts that a score of 3 on
// all five dimensions passes the gate (Check returns nil) and the recommendation
// is permitted to advance from `detected` to `drafted`.
//
// Guidelines §13 hard cap: all dimensions strictly above HoldThreshold (2)
// must allow the recommendation to proceed.
func TestAppropriatenessBlocking_AllPassAdvances(t *testing.T) {
	t.Parallel()

	a := appropriateness.Assessment{
		ClinicalWarrant:        3,
		EvidenceSolidity:       3,
		AlternativesConsidered: 3,
		RestraintConsidered:    3,
		GoalsOfCareAlignment:   3,
	}
	if err := appropriateness.Check(a); err != nil {
		t.Errorf("all-3 assessment should pass gate; got %v", err)
	}
	if !a.PassesGate() {
		t.Error("PassesGate() should return true for all-3 assessment")
	}
}

// TestAppropriatenessBlocking_BoundaryAtThreshold asserts that scores exactly
// equal to HoldThreshold (2) trigger a hold, while scores at HoldThreshold+1
// (3) pass, confirming the inclusive boundary semantics.
func TestAppropriatenessBlocking_BoundaryAtThreshold(t *testing.T) {
	t.Parallel()

	// Score 2 == HoldThreshold must hold.
	atThreshold := appropriateness.Assessment{
		ClinicalWarrant: 2, EvidenceSolidity: 5,
		AlternativesConsidered: 5, RestraintConsidered: 5, GoalsOfCareAlignment: 5,
	}
	if err := appropriateness.Check(atThreshold); !errors.Is(err, appropriateness.ErrAppropriatenessHold) {
		t.Errorf("score==HoldThreshold(%d) must hold; got %v", appropriateness.HoldThreshold, err)
	}

	// Score 3 == HoldThreshold+1 must pass.
	aboveThreshold := appropriateness.Assessment{
		ClinicalWarrant: 3, EvidenceSolidity: 5,
		AlternativesConsidered: 5, RestraintConsidered: 5, GoalsOfCareAlignment: 5,
	}
	if err := appropriateness.Check(aboveThreshold); err != nil {
		t.Errorf("score==HoldThreshold+1(%d) must pass; got %v", appropriateness.HoldThreshold+1, err)
	}
}
