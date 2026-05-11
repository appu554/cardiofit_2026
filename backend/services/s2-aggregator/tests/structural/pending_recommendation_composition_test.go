// pending_recommendation_composition_test.go — v1.0 Part 17 Category 3
// (pending recommendation composition). Tests the interaction edges
// between pending recs and other panels: restraint pairing,
// goals-of-care conflict, FIR veto, override history, empty state,
// sort order.
package structural

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// TestPendingRec_PairsWithRestraintSignal — when a restraint signal +
// recommendation share a recommendation_id, pairing populates the
// card's PairedRestraintSignal field.
func TestPendingRec_PairsWithRestraintSignal(t *testing.T) {
	rid := uuid.New()
	asOf := time.Now()
	pkt := mkPktV9(rid, "STOP", "red")

	client := aggregation.NewInMemorySubstrateClient().
		WithPackets(pkt).
		WithRestraintSignals(rid, substrate_types.RestraintSignal{
			SignalID:               uuid.New(),
			Type:                   "recent_pathology_collection_attempt",
			Severity:               2,
			PairedRecommendationID: pkt.RecommendationID,
			TriggeredAt:            asOf.AddDate(0, 0, -3),
			SubstrateID:            uuid.New(),
			SubstrateSource:        "kb-32-restraint",
		})

	cards, err := aggregation.BuildPendingRecommendationCards(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("BuildPendingRecommendationCards: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected 1 card; got %d", len(cards))
	}
	if cards[0].PairedRestraintSignal == nil {
		t.Fatal("expected PairedRestraintSignal to be populated when signal shares RecommendationID")
	}
	if cards[0].PairedRestraintSignal.Type != "recent_pathology_collection_attempt" {
		t.Errorf("paired signal type mismatch: got %q", cards[0].PairedRestraintSignal.Type)
	}
}

// TestPendingRec_ConflictWithGoalsOfCare — ADD on palliative →
// GoalsConflict emitted via DetectGoalsConflicts.
func TestPendingRec_ConflictWithGoalsOfCare(t *testing.T) {
	rid := uuid.New()
	asOf := time.Now()

	addPkt := mkPktV9(rid, "ADD", "green")
	client := aggregation.NewInMemorySubstrateClient().
		WithPackets(addPkt).
		WithGoalsOfCare(rid, substrate_types.GoalsOfCareEntry{
			State: substrate_types.GoCStatePalliative, EffectiveFrom: asOf.AddDate(0, -1, 0),
			DocumentedBy: uuid.New(), SubstrateID: uuid.New(),
		})

	cards, err := aggregation.BuildPendingRecommendationCards(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("BuildPendingRecommendationCards: %v", err)
	}
	goc, err := aggregation.BuildGoalsOfCarePanel(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("BuildGoalsOfCarePanel: %v", err)
	}
	conflicts := aggregation.DetectGoalsConflicts(cards, goc.Current)
	if len(conflicts) == 0 {
		t.Fatal("expected at least one GoalsConflict for ADD on palliative")
	}
	if conflicts[0].RecommendationID != addPkt.RecommendationID {
		t.Errorf("conflict references wrong recommendation: got %s want %s",
			conflicts[0].RecommendationID, addPkt.RecommendationID)
	}
	if len(conflicts[0].SubstrateRefs) == 0 {
		t.Error("GoalsConflict carries no SubstrateRefs (verification-not-belief)")
	}
}

// TestPendingRec_FIRVeto_InlineBadge — active FIR record matching the
// recommendation's intervention type links the FIR card's
// LinkedRecommendationIDs to the recommendation.
//
// We exercise the linkage from the FIR panel side (panel builder owns
// the link index) since BuildPendingRecommendationCards does not
// itself perform the FIR×rec pairing — that is FIR-panel-owned per
// Task 5 design.
func TestPendingRec_FIRVeto_InlineBadge(t *testing.T) {
	rid := uuid.New()
	asOf := time.Now()
	stopPkt := mkPktV9(rid, "STOP", "red")
	stopPkt.AppliedRule.RuleID = "STOP_ANTIPSYCHOTIC_DEPRESCRIBING"

	client := aggregation.NewInMemorySubstrateClient().
		WithPackets(stopPkt).
		WithFailedInterventions(rid, substrate_types.FailedInterventionRecord{
			ResidentID:        rid,
			InterventionType:  "antipsychotic_deprescribing",
			AttemptDate:       asOf.AddDate(0, -3, 0),
			Outcome:           substrate_types.OutcomeReversedDueToBPSDRecurrence,
			RetryEligibleDate: asOf.AddDate(0, 9, 0),
			DocumentedBy:      uuid.New(),
		})

	panel, err := aggregation.BuildFailedInterventionPanel(
		context.Background(), client, rid, asOf,
		aggregation.WithClassifier(prefixClassifier{prefix: "STOP_ANTIPSYCHOTIC_", typ: "antipsychotic_deprescribing"}),
	)
	if err != nil {
		t.Fatalf("BuildFailedInterventionPanel: %v", err)
	}
	if len(panel.Cards) != 1 {
		t.Fatalf("expected 1 FIR card; got %d", len(panel.Cards))
	}
	if len(panel.Cards[0].LinkedRecommendationIDs) != 1 {
		t.Errorf("expected FIR card linked to 1 recommendation; got %d",
			len(panel.Cards[0].LinkedRecommendationIDs))
	}
	if !panel.Cards[0].IsActiveVeto {
		t.Error("expected IsActiveVeto=true (RetryEligibleDate is in the future)")
	}
}

// prefixClassifier is an inline aggregation.ClassifierAdapter — when
// the RuleID starts with `prefix`, returns `typ` + true.
type prefixClassifier struct {
	prefix string
	typ    string
}

func (c prefixClassifier) ClassifyInterventionType(ruleID string) (string, bool) {
	if len(ruleID) >= len(c.prefix) && ruleID[:len(c.prefix)] == c.prefix {
		return c.typ, true
	}
	return "", false
}

// TestPendingRec_OverrideHistory_Populated — kb-32 prior overrides for
// the same recommendation render in the OverrideHistory slice.
func TestPendingRec_OverrideHistory_Populated(t *testing.T) {
	rid := uuid.New()
	asOf := time.Now()
	pkt := mkPktV9(rid, "MONITOR", "amber")

	or1 := substrate_types.OverrideReason{
		ID: uuid.New().String(), RecommendationID: pkt.RecommendationID.String(),
		ReasonCode: "clinical_judgment", ReasonCodeShort: "CJG",
		AppropriatenessFlag: "appropriate_override",
		Reasoning:           "GP reviewed last visit, monitoring not required",
		CapturedAt:          asOf.AddDate(0, -1, 0),
	}
	or2 := substrate_types.OverrideReason{
		ID: uuid.New().String(), RecommendationID: pkt.RecommendationID.String(),
		ReasonCode: "monitoring_in_place", ReasonCodeShort: "MIP",
		AppropriatenessFlag: "appropriate_override",
		Reasoning:           "pathology already scheduled",
		CapturedAt:          asOf.AddDate(0, 0, -10),
	}

	client := aggregation.NewInMemorySubstrateClient().
		WithPackets(pkt).
		WithOverrides(pkt.RecommendationID, or1, or2)

	cards, err := aggregation.BuildPendingRecommendationCards(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("BuildPendingRecommendationCards: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected 1 card; got %d", len(cards))
	}
	if len(cards[0].OverrideHistory) != 2 {
		t.Errorf("OverrideHistory should have 2 entries; got %d", len(cards[0].OverrideHistory))
	}
	// Chronological (oldest first): or1 (1mo ago) comes before or2 (10d ago).
	if cards[0].OverrideHistory[0].CapturedAt.After(cards[0].OverrideHistory[1].CapturedAt) {
		t.Error("OverrideHistory should be chronological (oldest first)")
	}
}

// TestPendingRec_EmptyState_HasSubstrateRef — empty-state contract: an
// empty slice is returned (Part 6.5), but the cross-cutting Task 9
// test synthesises the substrate anchor for the "zero rows" claim.
// Here we assert the panel-level contract: BuildPendingRecommendationCards
// must return a NON-NIL empty slice on no packets.
func TestPendingRec_EmptyState_HasSubstrateRef(t *testing.T) {
	rid := uuid.New()
	client := aggregation.NewInMemorySubstrateClient() // no packets
	cards, err := aggregation.BuildPendingRecommendationCards(context.Background(), client, rid, time.Now())
	if err != nil {
		t.Fatalf("BuildPendingRecommendationCards: %v", err)
	}
	if cards == nil {
		t.Fatal("empty-state must return non-nil empty slice (v1.0 Part 6.5)")
	}
	if len(cards) != 0 {
		t.Errorf("expected empty slice; got %d cards", len(cards))
	}
	// The cross-cutting verification-not-belief test
	// (TestEveryClaimHasSubstrateReference scenario=empty_pending_recs)
	// asserts that the empty-state CLAIM still carries a SubstrateRef.
}

// TestPendingRec_SortOrder_StopBeforeMonitorBeforeDoseChangeBeforeAdd —
// sort priority per kb-32 ordering rules + v1.0 Part 6.1.
func TestPendingRec_SortOrder_StopBeforeMonitorBeforeDoseChangeBeforeAdd(t *testing.T) {
	rid := uuid.New()
	asOf := time.Now()
	client := aggregation.NewInMemorySubstrateClient().WithPackets(
		mkPktV9(rid, "ADD", "red"),
		mkPktV9(rid, "MONITOR", "amber"),
		mkPktV9(rid, "DOSE_CHANGE", "amber"),
		mkPktV9(rid, "STOP", "green"), // green STOP still beats red ADD
	)
	cards, err := aggregation.BuildPendingRecommendationCards(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("BuildPendingRecommendationCards: %v", err)
	}
	if len(cards) != 4 {
		t.Fatalf("expected 4 cards; got %d", len(cards))
	}
	want := []string{"STOP", "MONITOR", "DOSE_CHANGE", "ADD"}
	for i, c := range cards {
		if c.Type != want[i] {
			t.Errorf("sort order [%d]: got %q want %q (full sequence: %v)",
				i, c.Type, want[i], typesOf(cards))
		}
	}
}

func typesOf(cards []aggregation.PendingRecommendationCard) []string {
	out := make([]string, len(cards))
	for i, c := range cards {
		out[i] = c.Type
	}
	return out
}
