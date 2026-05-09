// VisibilityClass: AD
package reasoners

import (
	"context"
	"errors"
	"testing"

	"github.com/cardiofit/shared/v2_substrate/ethics/erm"
)

func TestRecommendationReasoner_HoldsLowAppropriateness(t *testing.T) {
	r := NewRecommendationReasoner(3.0, &fakeDivergence{})
	dp := erm.DecisionPoint{
		DecisionType:   erm.DecisionTypeRecommendationDraft,
		ProposedOutput: RecommendationProposal{AppropriatenessScore: 2.4, RuleID: "R1"},
	}
	out, concerns := r.Review(context.Background(), dp)
	if out != erm.OutcomeHold {
		t.Errorf("expected hold for low appropriateness, got %v", out)
	}
	if len(concerns) == 0 || concerns[0].Principle != "P2" {
		t.Errorf("expected P2 concern, got %v", concerns)
	}
}

func TestRecommendationReasoner_ApprovesHighAppropriateness(t *testing.T) {
	r := NewRecommendationReasoner(3.0, &fakeDivergence{})
	dp := erm.DecisionPoint{
		DecisionType:   erm.DecisionTypeRecommendationDraft,
		ProposedOutput: RecommendationProposal{AppropriatenessScore: 4.2, RuleID: "R1"},
	}
	out, _ := r.Review(context.Background(), dp)
	if out != erm.OutcomeApprove {
		t.Errorf("expected approve, got %v", out)
	}
}

func TestRecommendationReasoner_HoldsDivergentRule(t *testing.T) {
	r := NewRecommendationReasoner(3.0, &fakeDivergence{divergent: map[string]bool{"R1": true}})
	dp := erm.DecisionPoint{
		DecisionType:   erm.DecisionTypeRecommendationDraft,
		ProposedOutput: RecommendationProposal{AppropriatenessScore: 4.5, RuleID: "R1"},
	}
	out, _ := r.Review(context.Background(), dp)
	if out != erm.OutcomeHold {
		t.Errorf("expected hold for divergent rule, got %v", out)
	}
}

// TestRecommendationReasoner_DivergenceSourceErrorPropagates verifies that
// when the DivergenceSource returns an error, Review returns Hold with a P6
// concern (audit-trail incomplete) rather than silently approving.
func TestRecommendationReasoner_DivergenceSourceErrorPropagates(t *testing.T) {
	r := NewRecommendationReasoner(3.0, &fakeDivergence{err: errors.New("db unavailable")})
	dp := erm.DecisionPoint{
		DecisionType: erm.DecisionTypeRecommendationDraft,
		ProposedOutput: RecommendationProposal{
			AppropriatenessScore: 4.5,
			RuleID:               "R-ok",
		},
	}
	out, concerns := r.Review(context.Background(), dp)
	if out != erm.OutcomeHold {
		t.Errorf("expected hold when divergence source errors, got %v", out)
	}
	if len(concerns) == 0 || concerns[0].Principle != "P6" {
		t.Errorf("expected P6 concern for audit-trail incomplete, got %v", concerns)
	}
}

// TestRecommendationReasoner_RestraintOverrideWithoutReasoning verifies that
// a RestraintOverridden=true with no RestraintReasoning produces Hold + P3.
func TestRecommendationReasoner_RestraintOverrideWithoutReasoning(t *testing.T) {
	r := NewRecommendationReasoner(3.0, &fakeDivergence{})
	dp := erm.DecisionPoint{
		DecisionType: erm.DecisionTypeRecommendationDraft,
		ProposedOutput: RecommendationProposal{
			AppropriatenessScore: 4.0,
			RuleID:               "R1",
			RestraintOverridden:  true,
			RestraintReasoning:   "",
		},
	}
	out, concerns := r.Review(context.Background(), dp)
	if out != erm.OutcomeHold {
		t.Errorf("expected hold for restraint override without reasoning, got %v", out)
	}
	if len(concerns) == 0 || concerns[0].Principle != "P3" {
		t.Errorf("expected P3 concern, got %v", concerns)
	}
}

// ── test helpers ────────────────────────────────────────────────────────────

type fakeDivergence struct {
	divergent map[string]bool
	err       error
}

func (f *fakeDivergence) IsDivergent(_ context.Context, ruleID string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.divergent[ruleID], nil
}
