// Package clinical_safety_test validates that STOP recommendations always carry
// a non-NA evidence anchor after GenerateWithNegativeEvidence is called with
// an absence-confirmed Querier.
//
// Recommendation Craft Guidelines Part 13 — clinical safety test category.
// VisibilityClass: AD — negative-evidence audit per Guidelines §7
package clinical_safety_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/generator"
	"github.com/cardiofit/kb32/internal/negative_evidence"
	"github.com/cardiofit/kb32/internal/reasoning"
	"github.com/cardiofit/kb32/internal/template"
	kb32ctx "github.com/cardiofit/kb32/internal/context"
)

// absenceConfirmedQuerier always confirms absence (no presence detected).
// This is equivalent to negative_evidence.NewInMemoryQuerier(nil).
type absenceConfirmedQuerier struct{}

func (a *absenceConfirmedQuerier) QueryAbsence(ctx context.Context, q negative_evidence.AbsenceQuery) (negative_evidence.AbsenceResult, error) {
	result, err := negative_evidence.NewInMemoryQuerier(nil).QueryAbsence(ctx, q)
	return result, err
}

var _ negative_evidence.Querier = (*absenceConfirmedQuerier)(nil)

// minimalSnapshot returns a minimal ClinicalSnapshot sufficient for Generate.
func minimalSnapshot(residentID uuid.UUID) kb32ctx.ClinicalSnapshot {
	return kb32ctx.ClinicalSnapshot{
		ResidentID: residentID,
	}
}

// stopRule builds an ApplicableRule with the given ruleID and type STOP.
func stopRule(ruleID string) reasoning.ApplicableRule {
	return reasoning.ApplicableRule{
		RuleID: ruleID,
		Type:   "STOP",
	}
}

// TestNegativeEvidenceCompleteness_STOPRulesHaveAnchor asserts that for each
// common STOP rule (PostFall, PPI, BenzodiazepineLongTerm, and an unknown
// fallback), calling GenerateWithNegativeEvidence with an absence-confirmed
// Querier always sets the "evidence" section to a non-NA value.
//
// This test documents the invariant: STOP recommendations must never surface
// to a clinical audience with an NA evidence section when absence is confirmed.
func TestNegativeEvidenceCompleteness_STOPRulesHaveAnchor(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	authorID := uuid.New()
	querier := &absenceConfirmedQuerier{}
	attacher := negative_evidence.NewQuerierAttacher(querier)

	stopRules := []struct {
		name   string
		ruleID string
	}{
		{"PostFall", "PostFall_Rule_001"},
		{"PPI", "PPI_DeprescribingRule"},
		{"BenzodiazepineLongTerm", "BenzodiazepineLongTerm_Rule"},
		{"unknown-fallback", "UnknownSTOPRule_XYZ"},
	}

	for _, tc := range stopRules {
		tc := tc // capture loop var
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			residentID := uuid.New()
			snap := minimalSnapshot(residentID)
			rules := []reasoning.ApplicableRule{stopRule(tc.ruleID)}

			pkt, err := generator.GenerateWithNegativeEvidence(ctx, snap, rules, authorID, attacher)
			if err != nil {
				t.Fatalf("GenerateWithNegativeEvidence failed for %s: %v", tc.name, err)
			}

			evidence, ok := pkt.Sections["evidence"]
			if !ok {
				t.Fatalf("packet has no 'evidence' section for rule %s", tc.ruleID)
			}

			// The evidence section must be non-NA after a confirmed absence.
			// NA is the template placeholder; a real absence statement replaces it.
			if evidence == template.NA || evidence == "" {
				t.Errorf(
					"rule %s: evidence section is %q after confirmed absence; expected a non-NA anchor text",
					tc.ruleID, evidence,
				)
			}
		})
	}
}
