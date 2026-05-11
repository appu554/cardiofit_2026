// restraint_signal_composition_test.go — v1.0 Part 17 Category 4
// (restraint signal rendering). Composition tests for the Phase 1
// advisory-only contract + the acknowledgment workflow.
package structural

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// TestRestraint_Phase1_AdvisoryOnly_NotSuppressive — even when
// TransitionCriteriaSatisfied=true, the signal still appears (per
// Task 4 contract + v1.0 Part 7.3).
//
// The aggregator-side contract is that TransitionCriteriaSatisfied is
// INFORMATIONAL ONLY and BuildRestraintSignalCards does not filter on
// it. We verify the contract by asserting the card always appears
// regardless of any "transition" semantics; the aggregator does not
// suppress.
func TestRestraint_Phase1_AdvisoryOnly_NotSuppressive(t *testing.T) {
	rid := uuid.New()
	client := aggregation.NewInMemorySubstrateClient().WithRestraintSignals(rid,
		substrate_types.RestraintSignal{
			SignalID:        uuid.New(),
			Type:            "care_intensity_transition_recent",
			Severity:        3,
			TriggeredAt:     time.Now().AddDate(0, 0, -1),
			SubstrateID:     uuid.New(),
			SubstrateSource: "kb-32-restraint",
		},
	)
	cards, err := aggregation.BuildRestraintSignalCards(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("BuildRestraintSignalCards: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected 1 card; got %d", len(cards))
	}
	// Phase 1 commitment: TransitionCriteriaSatisfied is informational —
	// the aggregator sets it to false and the renderer MUST NOT auto-
	// suppress. We assert the field is false (default) and the card
	// rendered anyway.
	if cards[0].TransitionCriteriaSatisfied {
		t.Error("Phase 1: TransitionCriteriaSatisfied must default to false in BuildRestraintSignalCards output (informational-only)")
	}
}

// TestRestraint_AcknowledgmentRequest_BypassReasoning_Mandatory —
// Decision=invoke_safety_critical_bypass + empty BypassReasoning →
// validation error.
func TestRestraint_AcknowledgmentRequest_BypassReasoning_Mandatory(t *testing.T) {
	req := aggregation.RestraintAcknowledgmentRequest{
		SignalID:        uuid.New(),
		PharmacistID:    uuid.New(),
		Decision:        substrate_types.RestraintDecisionSafetyCriticalBypass,
		BypassReasoning: "", // missing — must reject
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected ValidationError when bypass reasoning is empty (v1.0 Part 7.4)")
	}
}

// TestRestraint_AcknowledgmentRequest_BypassReasoning_Whitespace —
// whitespace-only reasoning does not satisfy the mandatory contract.
func TestRestraint_AcknowledgmentRequest_BypassReasoning_Whitespace(t *testing.T) {
	req := aggregation.RestraintAcknowledgmentRequest{
		SignalID:        uuid.New(),
		PharmacistID:    uuid.New(),
		Decision:        substrate_types.RestraintDecisionSafetyCriticalBypass,
		BypassReasoning: "   \t\n   ",
	}
	if err := req.Validate(); err == nil {
		t.Fatal("expected ValidationError for whitespace-only bypass reasoning")
	}
}

// TestRestraint_AcknowledgmentRequest_Advisory_ReasoningOptional —
// Decision=acknowledge_advisory + empty reasoning → no error
// (reasoning is optional for advisory acknowledgments).
func TestRestraint_AcknowledgmentRequest_Advisory_ReasoningOptional(t *testing.T) {
	req := aggregation.RestraintAcknowledgmentRequest{
		SignalID:     uuid.New(),
		PharmacistID: uuid.New(),
		Decision:     substrate_types.RestraintDecisionAcknowledgeAdvisory,
		// BypassReasoning omitted — must be accepted
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("advisory acknowledgment with empty reasoning should be accepted; got %v", err)
	}
}

// TestRestraint_AcknowledgmentRequest_UnknownDecision_Rejected —
// closed-set Decision vocabulary: unknown values are rejected even if
// reasoning is supplied.
func TestRestraint_AcknowledgmentRequest_UnknownDecision_Rejected(t *testing.T) {
	req := aggregation.RestraintAcknowledgmentRequest{
		SignalID:        uuid.New(),
		PharmacistID:    uuid.New(),
		Decision:        "ignore_signal", // not in closed set
		BypassReasoning: "I know better",
	}
	if err := req.Validate(); err == nil {
		t.Fatal("expected ValidationError for unknown decision value")
	}
}

// TestRestraint_NilIDs_Rejected — SignalID/PharmacistID = uuid.Nil are
// rejected to prevent accidental writes attributed to no-one.
func TestRestraint_NilIDs_Rejected(t *testing.T) {
	cases := []struct {
		name string
		req  aggregation.RestraintAcknowledgmentRequest
	}{
		{"nil_signal_id", aggregation.RestraintAcknowledgmentRequest{
			SignalID: uuid.Nil, PharmacistID: uuid.New(),
			Decision: substrate_types.RestraintDecisionAcknowledgeAdvisory,
		}},
		{"nil_pharmacist_id", aggregation.RestraintAcknowledgmentRequest{
			SignalID: uuid.New(), PharmacistID: uuid.Nil,
			Decision: substrate_types.RestraintDecisionAcknowledgeAdvisory,
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.req.Validate(); err == nil {
				t.Error("expected validation error for nil id")
			}
		})
	}
}
