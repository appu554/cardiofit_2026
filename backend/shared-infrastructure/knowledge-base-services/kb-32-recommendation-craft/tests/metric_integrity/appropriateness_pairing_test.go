// Package metric_integrity_test validates forward-compatible metric shapes for
// recommendation appropriateness pairing.
//
// Recommendation Craft Guidelines Part 13 — metric integrity test category.
//
// This test documents the expected shape of the paired metric that Phase 2
// completion (or Phase 3) will produce. It asserts that:
//   - An appropriateness.Assessment and an override flag can be co-located in
//     a single struct without field collisions.
//   - The Assessment.PassesGate() and overrides.IsValidFlag() behaviours are
//     consistent for the co-located values.
//
// The actual paired-storage implementation (writing both the Assessment scores
// and the acceptance flag into the same metrics table row) is deferred to a
// Phase 2-completion or Phase 3 task. This test exists to document the
// expected interface so that implementation can proceed without ambiguity.
//
// VisibilityClass: AD — appropriateness gate per Guidelines §9;
// override capture per Guidelines §5
package metric_integrity_test

import (
	"testing"

	"github.com/cardiofit/kb32/internal/appropriateness"
	"github.com/cardiofit/kb32/internal/overrides"
)

// PairedMetric is the forward-compatible shape for paired appropriateness and
// acceptance metrics. This is the struct that the Phase 2-completion storage
// implementation must produce and persist.
//
// The exact persistence mechanism (dedicated table, JSONB column, or
// time-series metric row) is deferred. What this test validates is that the
// Go types are composable without ambiguity.
type PairedMetric struct {
	// Assessment is the five-dimension appropriateness score captured at the
	// moment the recommendation was generated (Stage 4 output).
	Assessment appropriateness.Assessment

	// OverrideFlag is the override taxonomy appropriateness classification
	// captured when the clinician accepted or overrode the recommendation.
	// One of: "appropriate_override", "inappropriate_override", "mixed".
	// Empty string means the recommendation was accepted without override.
	OverrideFlag string
}

// TestAppropriatenessPairing_DocumentsExpectedShape asserts that:
//  1. A PairedMetric struct can be constructed with a passing Assessment and a
//     valid override flag without compilation errors or type conflicts.
//  2. Assessment.PassesGate() returns true for the synthetic passing assessment.
//  3. overrides.IsValidFlag() returns true for the synthetic override flag.
//
// If either assertion fails it indicates a contract regression in the underlying
// packages that would break the paired-metric implementation.
func TestAppropriatenessPairing_DocumentsExpectedShape(t *testing.T) {
	t.Parallel()

	p := PairedMetric{
		Assessment: appropriateness.Assessment{
			ClinicalWarrant:        4,
			EvidenceSolidity:       4,
			AlternativesConsidered: 4,
			RestraintConsidered:    4,
			GoalsOfCareAlignment:   4,
		},
		OverrideFlag: "appropriate_override",
	}

	if !p.Assessment.PassesGate() {
		t.Errorf(
			"synthetic paired Assessment (all-4) should pass gate; "+
				"this indicates a regression in appropriateness.Assessment.PassesGate()",
		)
	}

	if !overrides.IsValidFlag(p.OverrideFlag) {
		t.Errorf(
			"OverrideFlag %q is not valid; "+
				"this indicates a regression in overrides.IsValidFlag()",
			p.OverrideFlag,
		)
	}
}

// TestAppropriatenessPairing_AllThreeFlagsAreValid documents that all three
// canonical override flag values can be set on PairedMetric without issue.
func TestAppropriatenessPairing_AllThreeFlagsAreValid(t *testing.T) {
	t.Parallel()

	canonicalFlags := []string{
		"appropriate_override",
		"inappropriate_override",
		"mixed",
	}

	passingAssessment := appropriateness.Assessment{
		ClinicalWarrant:        3,
		EvidenceSolidity:       3,
		AlternativesConsidered: 3,
		RestraintConsidered:    3,
		GoalsOfCareAlignment:   3,
	}

	for _, flag := range canonicalFlags {
		flag := flag // capture loop var
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			p := PairedMetric{
				Assessment:   passingAssessment,
				OverrideFlag: flag,
			}
			if !overrides.IsValidFlag(p.OverrideFlag) {
				t.Errorf("canonical flag %q failed IsValidFlag(); contract regression detected", flag)
			}
		})
	}
}

// TestAppropriatenessPairing_HeldAssessmentPairsWithInappropriateFlag
// documents that a held Assessment (below gate) can be paired with an
// inappropriate_override flag — this is the expected pairing when a clinician
// overrides a recommendation that was held at Stage 4.
//
// The test confirms the types compose cleanly even for this edge case.
func TestAppropriatenessPairing_HeldAssessmentPairsWithInappropriateFlag(t *testing.T) {
	t.Parallel()

	p := PairedMetric{
		Assessment: appropriateness.Assessment{
			ClinicalWarrant:        2, // held
			EvidenceSolidity:       5,
			AlternativesConsidered: 5,
			RestraintConsidered:    5,
			GoalsOfCareAlignment:   5,
		},
		OverrideFlag: "inappropriate_override",
	}

	// The held assessment must not pass the gate.
	if p.Assessment.PassesGate() {
		t.Error("held assessment (clinical_warrant=2) should not pass gate")
	}

	// The override flag must still be structurally valid for audit purposes.
	if !overrides.IsValidFlag(p.OverrideFlag) {
		t.Errorf("flag %q should be valid even when assessment is held", p.OverrideFlag)
	}
}
