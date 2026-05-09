package reasoners

import (
	"context"
	"testing"

	"github.com/cardiofit/shared/v2_substrate/ethics/erm"
)

func TestVisibilityReasoner_HoldsBelowReidentificationFloor(t *testing.T) {
	r := NewVisibilityReasoner(5)
	dp := erm.DecisionPoint{
		DecisionType:   erm.DecisionTypeVisibilityAggregate,
		ProposedOutput: AggregationProposal{PharmacistCount: 3, GateSatisfied: true},
	}
	out, _ := r.Review(context.Background(), dp)
	if out != erm.OutcomeHold {
		t.Errorf("expected hold for re-identification risk")
	}
}

func TestVisibilityReasoner_HoldsGateUnsatisfied(t *testing.T) {
	r := NewVisibilityReasoner(5)
	dp := erm.DecisionPoint{
		DecisionType:   erm.DecisionTypeVisibilityAggregate,
		ProposedOutput: AggregationProposal{PharmacistCount: 20, GateSatisfied: false},
	}
	out, _ := r.Review(context.Background(), dp)
	if out != erm.OutcomeHold {
		t.Errorf("expected hold when gate not satisfied")
	}
}

func TestVisibilityReasoner_ApproveHappy(t *testing.T) {
	r := NewVisibilityReasoner(5)
	dp := erm.DecisionPoint{
		DecisionType:   erm.DecisionTypeVisibilityAggregate,
		ProposedOutput: AggregationProposal{PharmacistCount: 30, GateSatisfied: true, SurveillanceFlag: false},
	}
	out, _ := r.Review(context.Background(), dp)
	if out != erm.OutcomeApprove {
		t.Errorf("expected approve, got %v", out)
	}
}

// TestVisibilityReasoner_MalformedProposalHolds verifies that a nil or
// wrong-type ProposedOutput returns Hold with a P6 concern.
func TestVisibilityReasoner_MalformedProposalHolds(t *testing.T) {
	r := NewVisibilityReasoner(5)

	t.Run("nil_proposal", func(t *testing.T) {
		dp := erm.DecisionPoint{
			DecisionType:   erm.DecisionTypeVisibilityAggregate,
			ProposedOutput: nil,
		}
		out, concerns := r.Review(context.Background(), dp)
		if out != erm.OutcomeHold {
			t.Errorf("expected Hold for nil proposal, got %v", out)
		}
		if len(concerns) == 0 || concerns[0].Principle != "P6" {
			t.Errorf("expected P6 concern, got %v", concerns)
		}
	})

	t.Run("wrong_type_proposal", func(t *testing.T) {
		dp := erm.DecisionPoint{
			DecisionType:   erm.DecisionTypeVisibilityAggregate,
			ProposedOutput: "not-an-aggregation-proposal",
		}
		out, concerns := r.Review(context.Background(), dp)
		if out != erm.OutcomeHold {
			t.Errorf("expected Hold for wrong-type proposal, got %v", out)
		}
		if len(concerns) == 0 || concerns[0].Principle != "P6" {
			t.Errorf("expected P6 concern, got %v", concerns)
		}
	})
}

// TestVisibilityReasoner_HoldsSurveillanceFlag verifies that a SurveillanceFlag
// triggers Hold (P5) even when all other gates pass.
func TestVisibilityReasoner_HoldsSurveillanceFlag(t *testing.T) {
	r := NewVisibilityReasoner(5)
	dp := erm.DecisionPoint{
		DecisionType: erm.DecisionTypeVisibilityAggregate,
		ProposedOutput: AggregationProposal{
			PharmacistCount:  20,
			GateSatisfied:    true,
			SurveillanceFlag: true,
		},
	}
	out, concerns := r.Review(context.Background(), dp)
	if out != erm.OutcomeHold {
		t.Errorf("expected Hold when surveillance flag is set, got %v", out)
	}
	if len(concerns) == 0 || concerns[0].Principle != "P5" {
		t.Errorf("expected P5 concern, got %v", concerns)
	}
}
