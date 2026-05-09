// Package appropriateness implements Stage 4 of the six-stage rendering pipeline:
// the clinical-safety gate that enforces a five-dimension appropriateness rubric
// before a recommendation may advance from the `detected` state to `drafted`.
//
// VisibilityClass: AD (audit-defensible) — clinical-safety gate per Guidelines §9
//
// The hold contract is non-negotiable: if any single dimension score is at or
// below HoldThreshold the recommendation MUST remain in `detected` state. The
// caller is contractually required to honour the hold — it must NOT advance the
// recommendation to `drafted` when Check returns ErrAppropriatenessHold.
//
// All five dimensions are scored on a 1–5 integer scale. Scores outside [1,5]
// indicate a caller bug and are rejected by ValidateScores before Check is run.
package appropriateness

import (
	"errors"
	"fmt"
)

// HoldThreshold is the inclusive boundary below which a dimension score triggers
// a clinical hold. Any dimension scored ≤ HoldThreshold holds the recommendation.
// This value is a non-negotiable clinical-safety commitment per Guidelines §9.
const HoldThreshold = 2

// ErrAppropriatenessHold is returned by Check when any dimension score is at or
// below HoldThreshold. The recommendation must remain in `detected` state; it
// MUST NOT be advanced to `drafted` while this error is active.
var ErrAppropriatenessHold = errors.New("appropriateness: dimension below hold threshold; recommendation held in detected")

// ErrInvalidScore is returned by ValidateScores when any dimension contains a
// score outside the valid range [1,5]. This guards against caller bugs that
// produce nonsense assessments before the hold gate is applied.
var ErrInvalidScore = errors.New("appropriateness: dimension score outside valid range [1,5]")

// Assessment scores each of five clinical dimensions on a 1–5 integer scale.
// All five scores must be in [1,5]; use ValidateScores to enforce this before
// calling Check or PassesGate. Each dimension maps to a distinct clinical concern
// as described in Recommendation Craft Guidelines Part 9.
type Assessment struct {
	// ClinicalWarrant asks: is the intervention clinically warranted for this patient?
	ClinicalWarrant int

	// EvidenceSolidity asks: how strong is the supporting evidence?
	EvidenceSolidity int

	// AlternativesConsidered asks: were relevant alternatives adequately weighed?
	AlternativesConsidered int

	// RestraintConsidered asks: was non-action (watchful waiting) evaluated as an option?
	RestraintConsidered int

	// GoalsOfCareAlignment asks: does the recommendation align with the documented goals of care?
	GoalsOfCareAlignment int
}

// IsValidScore reports whether n is a legal dimension score. Valid scores are
// integers in the inclusive range [1,5].
func IsValidScore(n int) bool {
	return n >= 1 && n <= 5
}

// ValidateScores returns ErrInvalidScore if any dimension score is outside [1,5].
// Callers should call this before Check to distinguish a nonsense assessment from
// a legitimate clinical hold.
func (a Assessment) ValidateScores() error {
	dims := []struct {
		name  string
		score int
	}{
		{"clinical_warrant", a.ClinicalWarrant},
		{"evidence_solidity", a.EvidenceSolidity},
		{"alternatives_considered", a.AlternativesConsidered},
		{"restraint_considered", a.RestraintConsidered},
		{"goals_of_care_alignment", a.GoalsOfCareAlignment},
	}
	for _, d := range dims {
		if !IsValidScore(d.score) {
			return fmt.Errorf("%w: %s has score %d", ErrInvalidScore, d.name, d.score)
		}
	}
	return nil
}

// Check returns nil when all five dimension scores are strictly above HoldThreshold,
// meaning the recommendation may safely advance to the `drafted` state.
// It returns ErrAppropriatenessHold when any single dimension score is ≤ HoldThreshold.
//
// Callers MUST honour the hold: a non-nil error from Check means the recommendation
// must remain in `detected` state and must not advance to `drafted`.
func Check(a Assessment) error {
	for _, score := range []int{
		a.ClinicalWarrant,
		a.EvidenceSolidity,
		a.AlternativesConsidered,
		a.RestraintConsidered,
		a.GoalsOfCareAlignment,
	} {
		if score <= HoldThreshold {
			return ErrAppropriatenessHold
		}
	}
	return nil
}

// PassesGate is a convenience method for lifecycle transition guards.
// It returns true when all five dimensions exceed HoldThreshold.
// Equivalent to Check(a) == nil.
func (a Assessment) PassesGate() bool { return Check(a) == nil }

// LowestDimension returns the name of the worst-scoring dimension and its score.
// When multiple dimensions share the minimum score the earliest in the canonical
// ordering (ClinicalWarrant → EvidenceSolidity → AlternativesConsidered →
// RestraintConsidered → GoalsOfCareAlignment) is returned.
//
// This method is intended for EthicsLog and audit-trail integration (Task 13):
// when ERM holds a recommendation the log entry must record WHICH dimension
// failed to preserve full audit detail.
func (a Assessment) LowestDimension() (string, int) {
	dims := []struct {
		name  string
		score int
	}{
		{"clinical_warrant", a.ClinicalWarrant},
		{"evidence_solidity", a.EvidenceSolidity},
		{"alternatives_considered", a.AlternativesConsidered},
		{"restraint_considered", a.RestraintConsidered},
		{"goals_of_care_alignment", a.GoalsOfCareAlignment},
	}
	worst := dims[0]
	for _, d := range dims[1:] {
		if d.score < worst.score {
			worst = d
		}
	}
	return worst.name, worst.score
}
