package structural

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// TestEveryPendingRecCardHasSubstrateRef enforces the v1.0 Part 17
// verification-not-belief invariant at the pending-recommendations layer:
// every PendingRecommendationCard built from a representative substrate
// must carry at least one SubstrateRef.
func TestEveryPendingRecCardHasSubstrateRef(t *testing.T) {
	rid := uuid.New()
	pkts := []substrate_types.RecommendationPacket{
		mkPkt(rid, "STOP", "red"),
		mkPkt(rid, "MONITOR", "amber"),
		mkPkt(rid, "ADD", "green"),
	}
	client := aggregation.NewInMemorySubstrateClient().WithPackets(pkts...)
	// Seed at least one citation for the STOP packet to exercise that ref path.
	client.WithCitations(pkts[0].RecommendationID, substrate_types.Citation{
		RecommendationID: pkts[0].RecommendationID.String(),
		SourceID:         "AMH-2025",
		Version:          "1.0.0",
		PinnedAt:         time.Now(),
	})

	cards, err := aggregation.BuildPendingRecommendationCards(context.Background(), client, rid, time.Now())
	if err != nil {
		t.Fatalf("BuildPendingRecommendationCards error: %v", err)
	}
	if len(cards) == 0 {
		t.Fatal("expected non-empty card set")
	}
	for _, c := range cards {
		if len(c.SubstrateRefs) == 0 {
			t.Errorf(
				"PendingRecommendationCard %s (%s) has no SubstrateRef — violates verification-not-belief (v1.0 Principle 2; Part 17 critical test)",
				c.RecommendationID, c.Type,
			)
		}
	}
}

// TestEveryRestraintSignalCardHasSubstrateRef enforces the same invariant
// for the restraint-signals panel.
func TestEveryRestraintSignalCardHasSubstrateRef(t *testing.T) {
	rid := uuid.New()
	sigs := []substrate_types.RestraintSignal{
		{
			SignalID:        uuid.New(),
			Type:            "care_intensity_transition_recent",
			Severity:        2,
			TriggeredAt:     time.Now().Add(-12 * 24 * time.Hour),
			SubstrateID:     uuid.New(),
			SubstrateSource: "kb-32-restraint",
		},
		{
			SignalID:        uuid.New(),
			Type:            "recent_pathology_collection_attempt",
			Severity:        2,
			TriggeredAt:     time.Now().Add(-3 * 24 * time.Hour),
			SubstrateID:     uuid.New(),
			SubstrateSource: "kb-32-restraint",
		},
	}
	client := aggregation.NewInMemorySubstrateClient().WithRestraintSignals(rid, sigs...)
	cards, err := aggregation.BuildRestraintSignalCards(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("BuildRestraintSignalCards error: %v", err)
	}
	if len(cards) == 0 {
		t.Fatal("expected non-empty restraint signal set")
	}
	for _, c := range cards {
		if len(c.SubstrateRefs) == 0 {
			t.Errorf(
				"RestraintSignalCard %s (%s) has no SubstrateRef — violates verification-not-belief",
				c.SignalID, c.Type,
			)
		}
	}
}

func mkPkt(rid uuid.UUID, t, urgency string) substrate_types.RecommendationPacket {
	return substrate_types.RecommendationPacket{
		RecommendationID: uuid.New(),
		AuthorID:         uuid.New(),
		Type:             t,
		Sections:         map[string]string{"layer_1": "body"},
		AppliedRule:      substrate_types.AppliedRule{RuleID: "r-" + t, Type: t, Urgency: urgency},
		SnapshotRef:      rid,
	}
}
