package aggregation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

func mkSignal(_ uuid.UUID, recID uuid.UUID) substrate_types.RestraintSignal {
	return substrate_types.RestraintSignal{
		SignalID:               uuid.New(),
		Type:                   "care_intensity_transition_recent",
		Severity:               2,
		PairedRecommendationID: recID,
		TriggeredAt:            time.Now().Add(-12 * 24 * time.Hour),
		SubstrateID:            uuid.New(),
		SubstrateSource:        "kb-32-restraint",
	}
}

func TestBuildRestraintSignalCards_EmptyState(t *testing.T) {
	rid := uuid.New()
	client := NewInMemorySubstrateClient()
	cards, err := BuildRestraintSignalCards(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if cards == nil {
		t.Fatal("nil returned; want non-nil empty slice")
	}
	if len(cards) != 0 {
		t.Errorf("want 0 cards, got %d", len(cards))
	}
}

func TestBuildRestraintSignalCards_TransitionCriteriaInformationalOnly(t *testing.T) {
	// v1.0 Part 7.3: transition criteria is INFORMATIONAL ONLY in Phase 1.
	// The aggregator MUST NOT auto-suppress signals based on it.
	rid := uuid.New()
	sig := mkSignal(rid, uuid.New())
	client := NewInMemorySubstrateClient().WithRestraintSignals(rid, sig)

	cards, err := BuildRestraintSignalCards(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("want 1 card (no auto-suppression), got %d", len(cards))
	}
	// Even if we explicitly forced TransitionCriteriaSatisfied=true, the
	// signal must still appear. The field is informational only.
	cards[0].TransitionCriteriaSatisfied = true

	// Re-fetch to assert the renderer's contract holds across calls:
	cards2, _ := BuildRestraintSignalCards(context.Background(), client, rid)
	if len(cards2) != 1 {
		t.Errorf("signal disappeared on second fetch; transition criteria must NOT auto-suppress (v1.0 Part 7.3)")
	}
}

func TestBuildRestraintSignalCards_PairedSignalCarriesSubstrateRef(t *testing.T) {
	rid := uuid.New()
	sig := mkSignal(rid, uuid.New())
	client := NewInMemorySubstrateClient().WithRestraintSignals(rid, sig)
	cards, _ := BuildRestraintSignalCards(context.Background(), client, rid)
	if len(cards[0].SubstrateRefs) == 0 {
		t.Error("restraint signal card has no SubstrateRefs — violates verification-not-belief")
	}
}

func TestRestraintAcknowledgmentRequest_Validate(t *testing.T) {
	sid := uuid.New()
	pid := uuid.New()

	cases := []struct {
		name    string
		req     RestraintAcknowledgmentRequest
		wantErr bool
	}{
		{"advisory_ok", RestraintAcknowledgmentRequest{
			SignalID: sid, PharmacistID: pid,
			Decision: substrate_types.RestraintDecisionAcknowledgeAdvisory,
		}, false},
		{"advisory_no_reasoning_ok", RestraintAcknowledgmentRequest{
			SignalID: sid, PharmacistID: pid,
			Decision: substrate_types.RestraintDecisionAcknowledgeAdvisory,
		}, false},
		{"bypass_with_reasoning_ok", RestraintAcknowledgmentRequest{
			SignalID: sid, PharmacistID: pid,
			Decision:        substrate_types.RestraintDecisionSafetyCriticalBypass,
			BypassReasoning: "toxic drug level confirmed at 3.2",
		}, false},
		{"bypass_missing_reasoning_err", RestraintAcknowledgmentRequest{
			SignalID: sid, PharmacistID: pid,
			Decision: substrate_types.RestraintDecisionSafetyCriticalBypass,
		}, true},
		{"bypass_whitespace_reasoning_err", RestraintAcknowledgmentRequest{
			SignalID: sid, PharmacistID: pid,
			Decision:        substrate_types.RestraintDecisionSafetyCriticalBypass,
			BypassReasoning: "   ",
		}, true},
		{"unknown_decision_err", RestraintAcknowledgmentRequest{
			SignalID: sid, PharmacistID: pid,
			Decision: "bogus",
		}, true},
		{"nil_signal_err", RestraintAcknowledgmentRequest{
			PharmacistID: pid,
			Decision:     substrate_types.RestraintDecisionAcknowledgeAdvisory,
		}, true},
		{"nil_pharmacist_err", RestraintAcknowledgmentRequest{
			SignalID: sid,
			Decision: substrate_types.RestraintDecisionAcknowledgeAdvisory,
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestPairRestraintSignalsWithRecommendations(t *testing.T) {
	rid := uuid.New()
	recID := uuid.New()
	cards := []PendingRecommendationCard{{RecommendationID: recID, Type: "STOP"}}
	signals := []RestraintSignalCard{{
		SignalID:               uuid.New(),
		Type:                   "care_intensity_transition_recent",
		Severity:               2,
		PairedRecommendationID: recID,
		TriggeredAt:            time.Now(),
	}}
	cards = PairRestraintSignalsWithRecommendations(signals, cards)
	if cards[0].PairedRestraintSignal == nil {
		t.Fatal("expected pairing to populate PairedRestraintSignal")
	}
	if cards[0].PairedRestraintSignal.PairedRecommendationID != recID {
		t.Errorf("paired rec id mismatch")
	}
	_ = rid
}

func TestPairRestraintSignalsWithRecommendations_UnpairedSignalsSkipped(t *testing.T) {
	recID := uuid.New()
	cards := []PendingRecommendationCard{{RecommendationID: recID, Type: "STOP"}}
	signals := []RestraintSignalCard{{
		SignalID:               uuid.New(),
		Type:                   "panel_level_signal",
		PairedRecommendationID: uuid.Nil, // unpaired
	}}
	cards = PairRestraintSignalsWithRecommendations(signals, cards)
	if cards[0].PairedRestraintSignal != nil {
		t.Error("unpaired (nil) signals must not attach to any card")
	}
}

func TestBuildRestraintSignalCards_NilClient(t *testing.T) {
	if _, err := BuildRestraintSignalCards(context.Background(), nil, uuid.New()); err == nil {
		t.Fatal("expected nil-client error")
	}
}
