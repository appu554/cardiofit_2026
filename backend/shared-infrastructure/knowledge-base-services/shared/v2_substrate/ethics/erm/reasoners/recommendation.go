// Package reasoners provides concrete ERM Reasoner implementations that plug
// into the Ethical Reasoning Module scaffold.
//
// All reasoners in this package are non-autonomous: they surface concerns for
// human review and MUST NOT be configured to auto-approve after a Hold or
// Reject verdict (Guidelines §4.6).
//
// VisibilityClass: AD
package reasoners

import (
	"context"

	"github.com/cardiofit/shared/v2_substrate/ethics/erm"
)

// RecommendationProposal is the expected ProposedOutput type for
// DecisionTypeRecommendationDraft decision points.
type RecommendationProposal struct {
	// AppropriatenessScore is the craft engine's appropriateness rating
	// (0–5 scale). Must be >= the configured minimum to pass.
	AppropriatenessScore float64

	// RuleID is the identifier of the clinical rule driving this
	// recommendation. Checked against the 30-day divergence list.
	RuleID string

	// RestraintOverridden is true when the craft engine overrode a patient
	// restraint signal. Override is only permissible when RestraintReasoning
	// is non-empty (Principle 3).
	RestraintOverridden bool

	// RestraintReasoning documents the clinical justification for the
	// restraint override. Required when RestraintOverridden is true.
	RestraintReasoning string
}

// DivergenceSource checks whether a clinical rule is currently on the 30-day
// acceptance-appropriateness divergence watchlist (craft engine §9).
type DivergenceSource interface {
	IsDivergent(ctx context.Context, ruleID string) (bool, error)
}

// RecommendationReasoner implements erm.Reasoner for
// DecisionTypeRecommendationDraft. It enforces:
//   - Appropriateness >= minAppropriateness threshold → else Hold (P2)
//   - Restraint signals not silently overridden (must have reasoning) → else Hold (P3)
//   - Rule not on divergence list → else Hold (P2)
//   - Divergence source available → else Hold (P6, audit-trail incomplete)
type RecommendationReasoner struct {
	minAppropriateness float64
	divergence         DivergenceSource
}

// NewRecommendationReasoner returns a RecommendationReasoner. minAppr is the
// minimum acceptable appropriateness score (e.g. 3.0). div is required; pass
// a no-op implementation if divergence checking is not yet wired.
func NewRecommendationReasoner(minAppr float64, div DivergenceSource) *RecommendationReasoner {
	return &RecommendationReasoner{minAppropriateness: minAppr, divergence: div}
}

// Review evaluates dp.ProposedOutput as a RecommendationProposal and applies
// the three ethical gates described above.
func (r *RecommendationReasoner) Review(ctx context.Context, dp erm.DecisionPoint) (erm.Outcome, []erm.Concern) {
	prop, ok := dp.ProposedOutput.(RecommendationProposal)
	if !ok {
		return erm.OutcomeHold, []erm.Concern{{
			Principle:    "P6",
			ConcernLevel: 2,
			Reasoning:    "malformed proposal: expected RecommendationProposal",
		}}
	}

	// Gate 1: appropriateness threshold (Principle 2 — acceptance follows
	// appropriateness).
	if prop.AppropriatenessScore < r.minAppropriateness {
		return erm.OutcomeHold, []erm.Concern{{
			Principle:    "P2",
			ConcernLevel: 4,
			Reasoning:    "appropriateness below minimum threshold",
		}}
	}

	// Gate 2: restraint signals must not be silently overridden (Principle 3).
	if prop.RestraintOverridden && prop.RestraintReasoning == "" {
		return erm.OutcomeHold, []erm.Concern{{
			Principle:    "P3",
			ConcernLevel: 3,
			Reasoning:    "restraint override without documented reasoning",
		}}
	}

	// Gate 3: divergence list check (Principle 2). A DivergenceSource error
	// triggers a P6 Hold (audit-trail incomplete) — fail-closed rather than
	// silently approving.
	if r.divergence != nil {
		divergent, err := r.divergence.IsDivergent(ctx, prop.RuleID)
		if err != nil {
			return erm.OutcomeHold, []erm.Concern{{
				Principle:    "P6",
				ConcernLevel: 4,
				Reasoning:    "divergence source unavailable: audit-trail incomplete",
			}}
		}
		if divergent {
			return erm.OutcomeHold, []erm.Concern{{
				Principle:    "P2",
				ConcernLevel: 4,
				Reasoning:    "rule on 30-day acceptance-appropriateness divergence list",
			}}
		}
	}

	return erm.OutcomeApprove, nil
}
