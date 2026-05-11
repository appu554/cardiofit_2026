package aggregation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

func mkPacket(rid uuid.UUID, t, urgency string) substrate_types.RecommendationPacket {
	return substrate_types.RecommendationPacket{
		RecommendationID: uuid.New(),
		AuthorID:         uuid.New(),
		Type:             t,
		Sections: map[string]string{
			"layer_1": t + " layer1 body",
			"layer_2": t + " layer2 body",
			"layer_3": t + " layer3 body",
		},
		AppliedRule: substrate_types.AppliedRule{RuleID: "rule-" + t, Type: t, Urgency: urgency},
		SnapshotRef: rid,
	}
}

func TestBuildPendingRecommendationCards_EmptyStateNotNil(t *testing.T) {
	// v1.0 Part 6.5: empty state must be a non-nil empty slice so
	// callers can distinguish "no data fetched" from "fetched, zero".
	rid := uuid.New()
	client := NewInMemorySubstrateClient()
	cards, err := BuildPendingRecommendationCards(context.Background(), client, rid, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cards == nil {
		t.Fatal("empty state returned nil; want non-nil empty slice per v1.0 Part 6.5")
	}
	if len(cards) != 0 {
		t.Fatalf("expected 0 cards, got %d", len(cards))
	}
}

func TestBuildPendingRecommendationCards_SortOrder(t *testing.T) {
	// Sort priority per kb-32 ordering.Order: STOP > MONITOR > DOSE_CHANGE > ADD;
	// within type, red > amber > green.
	rid := uuid.New()
	add := mkPacket(rid, "ADD", "amber")
	stopRed := mkPacket(rid, "STOP", "red")
	stopAmber := mkPacket(rid, "STOP", "amber")
	monitor := mkPacket(rid, "MONITOR", "amber")
	doseChange := mkPacket(rid, "DOSE_CHANGE", "green")

	client := NewInMemorySubstrateClient().WithPackets(add, stopAmber, monitor, doseChange, stopRed)

	cards, err := BuildPendingRecommendationCards(context.Background(), client, rid, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"STOP", "STOP", "MONITOR", "DOSE_CHANGE", "ADD"}
	got := make([]string, len(cards))
	for i, c := range cards {
		got[i] = c.Type
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("position %d: got %s, want %s (full order: %v)", i, got[i], want[i], got)
		}
	}
	// Within STOP, red must precede amber.
	if cards[0].Urgency != "red" || cards[1].Urgency != "amber" {
		t.Errorf("STOP urgency ordering wrong: %s %s; want red amber", cards[0].Urgency, cards[1].Urgency)
	}
}

func TestBuildPendingRecommendationCards_ConfidenceFromAssessment(t *testing.T) {
	rid := uuid.New()
	pkt := mkPacket(rid, "STOP", "red")
	client := NewInMemorySubstrateClient().
		WithPackets(pkt).
		WithAssessment(pkt.RecommendationID, substrate_types.AssessmentScores{
			ClinicalWarrant:        5,
			EvidenceSolidity:       4,
			AlternativesConsidered: 4,
			RestraintConsidered:    3,
			GoalsOfCareAlignment:   5,
		})

	cards, err := BuildPendingRecommendationCards(context.Background(), client, rid, time.Now())
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	c := cards[0]
	if c.Confidence.SubstrateConfidence != 4.0/5.0 {
		t.Errorf("SubstrateConfidence = %v, want %v", c.Confidence.SubstrateConfidence, 4.0/5.0)
	}
	wantClinical := float64(5+4+3+5) / (4.0 * 5.0)
	if c.Confidence.ClinicalConfidence != wantClinical {
		t.Errorf("ClinicalConfidence = %v, want %v", c.Confidence.ClinicalConfidence, wantClinical)
	}
	if c.HoldReason != "" {
		t.Errorf("HoldReason should be empty when all dims ≥3; got %q", c.HoldReason)
	}
}

func TestBuildPendingRecommendationCards_HoldReason(t *testing.T) {
	rid := uuid.New()
	pkt := mkPacket(rid, "STOP", "red")
	client := NewInMemorySubstrateClient().
		WithPackets(pkt).
		WithAssessment(pkt.RecommendationID, substrate_types.AssessmentScores{
			ClinicalWarrant:        5,
			EvidenceSolidity:       2, // hold trigger
			AlternativesConsidered: 4,
			RestraintConsidered:    4,
			GoalsOfCareAlignment:   5,
		})
	cards, _ := BuildPendingRecommendationCards(context.Background(), client, rid, time.Now())
	if cards[0].HoldReason == "" {
		t.Error("expected non-empty HoldReason for EvidenceSolidity=2")
	}
}

func TestBuildPendingRecommendationCards_PairsRestraintSignal(t *testing.T) {
	// v1.0 Part 6.4: when a paired restraint signal is active, the card
	// surfaces it via PairedRestraintSignal.
	rid := uuid.New()
	pkt := mkPacket(rid, "STOP", "red")
	signal := substrate_types.RestraintSignal{
		SignalID:               uuid.New(),
		Type:                   "care_intensity_transition_recent",
		Severity:               2,
		PairedRecommendationID: pkt.RecommendationID,
		TriggeredAt:            time.Now().Add(-8 * 24 * time.Hour),
		SubstrateID:            uuid.New(),
		SubstrateSource:        "kb-32-restraint",
	}
	client := NewInMemorySubstrateClient().
		WithPackets(pkt).
		WithRestraintSignals(rid, signal)

	cards, err := BuildPendingRecommendationCards(context.Background(), client, rid, time.Now())
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if cards[0].PairedRestraintSignal == nil {
		t.Fatal("expected paired restraint signal on card; got nil")
	}
	if cards[0].PairedRestraintSignal.SignalID != signal.SignalID {
		t.Errorf("paired signal ID mismatch: got %s want %s", cards[0].PairedRestraintSignal.SignalID, signal.SignalID)
	}
}

func TestBuildPendingRecommendationCards_CitationsAndOverridesAttached(t *testing.T) {
	rid := uuid.New()
	pkt := mkPacket(rid, "STOP", "red")
	cit := substrate_types.Citation{
		RecommendationID: pkt.RecommendationID.String(),
		SourceID:         "AMH-2025",
		Version:          "1.2.3",
		PinnedAt:         time.Now(),
	}
	or := substrate_types.OverrideReason{
		ID:                  "or-1",
		RecommendationID:    pkt.RecommendationID.String(),
		ReasonCode:          "patient_preference",
		ReasonCodeShort:     "PPF",
		AppropriatenessFlag: "appropriate_override",
		Reasoning:           "family declined",
		CapturedAt:          time.Now(),
		CapturedBy:          "pharm-1",
	}
	client := NewInMemorySubstrateClient().
		WithPackets(pkt).
		WithCitations(pkt.RecommendationID, cit).
		WithOverrides(pkt.RecommendationID, or)

	cards, _ := BuildPendingRecommendationCards(context.Background(), client, rid, time.Now())
	if len(cards[0].Citations) != 1 {
		t.Errorf("expected 1 citation; got %d", len(cards[0].Citations))
	}
	if len(cards[0].OverrideHistory) != 1 {
		t.Errorf("expected 1 override; got %d", len(cards[0].OverrideHistory))
	}
}

func TestBuildPendingRecommendationCards_NilClient(t *testing.T) {
	if _, err := BuildPendingRecommendationCards(context.Background(), nil, uuid.New(), time.Now()); err == nil {
		t.Fatal("expected error from nil client")
	}
}
