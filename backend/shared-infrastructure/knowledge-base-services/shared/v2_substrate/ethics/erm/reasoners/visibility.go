package reasoners

import (
	"context"

	"github.com/cardiofit/shared/v2_substrate/ethics/erm"
)

// AggregationProposal is the expected ProposedOutput type for
// DecisionTypeVisibilityAggregate decision points. It represents a proposed
// employer-facing aggregated visibility result.
type AggregationProposal struct {
	// PharmacistCount is the number of distinct pharmacists in the aggregated
	// result. Must be >= reidentificationFloor (default 5, per Guidelines
	// §11.1 Risk 11) to prevent re-identification.
	PharmacistCount int

	// GateSatisfied is true when the PFA (Pharmacist Facility Access)
	// aggregation gate conditions are all met.
	GateSatisfied bool

	// SurveillanceFlag is raised by the employer query pattern analyser
	// (§9.7) when the query matches a surveillance heuristic.
	SurveillanceFlag bool
}

// VisibilityReasoner implements erm.Reasoner for
// DecisionTypeVisibilityAggregate. It enforces three gates (all Hold with
// Principle 5):
//  1. PFA aggregation gate must be satisfied.
//  2. PharmacistCount >= reidentificationFloor.
//  3. SurveillanceFlag must be false.
//
// A nil or wrong-type ProposedOutput returns Hold with Principle 6
// (audit-trail incomplete).
type VisibilityReasoner struct {
	reidentificationFloor int
}

// NewVisibilityReasoner returns a VisibilityReasoner with the given
// re-identification floor. The recommended floor is 5 (Guidelines §11.1).
func NewVisibilityReasoner(floor int) *VisibilityReasoner {
	return &VisibilityReasoner{reidentificationFloor: floor}
}

// Review evaluates dp.ProposedOutput as an AggregationProposal and applies
// the three visibility gates.
func (r *VisibilityReasoner) Review(_ context.Context, dp erm.DecisionPoint) (erm.Outcome, []erm.Concern) {
	prop, ok := dp.ProposedOutput.(AggregationProposal)
	if !ok {
		return erm.OutcomeHold, []erm.Concern{{
			Principle:    "P6",
			ConcernLevel: 2,
			Reasoning:    "malformed aggregation: expected AggregationProposal",
		}}
	}

	// Gate 1: PFA aggregation gate (Principle 5 — employer data access
	// subject to explicit consent gate).
	if !prop.GateSatisfied {
		return erm.OutcomeHold, []erm.Concern{{
			Principle:    "P5",
			ConcernLevel: 4,
			Reasoning:    "PFA aggregation gate not satisfied",
		}}
	}

	// Gate 2: re-identification floor (Principle 5 — individual
	// pharmacists must not be re-identifiable from aggregated results).
	if prop.PharmacistCount < r.reidentificationFloor {
		return erm.OutcomeHold, []erm.Concern{{
			Principle:    "P5",
			ConcernLevel: 5,
			Reasoning:    "re-identification risk: subset below minimum",
		}}
	}

	// Gate 3: surveillance heuristic (Principle 5 — employer query patterns
	// must not enable workforce surveillance, §9.7).
	if prop.SurveillanceFlag {
		return erm.OutcomeHold, []erm.Concern{{
			Principle:    "P5",
			ConcernLevel: 4,
			Reasoning:    "query pattern matches surveillance heuristic",
		}}
	}

	return erm.OutcomeApprove, nil
}
