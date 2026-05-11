package structural

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// TestEveryFailedInterventionCardHasSubstrateRef enforces the v1.0 Part
// 17 verification-not-belief invariant for Panel F: every populated
// FailedInterventionCard must carry ≥1 SubstrateRef.
func TestEveryFailedInterventionCardHasSubstrateRef(t *testing.T) {
	rid := uuid.New()
	now := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	recs := []substrate_types.FailedInterventionRecord{
		{
			ResidentID:        rid,
			InterventionType:  "antipsychotic_deprescribing",
			AttemptDate:       now.AddDate(0, -2, 0),
			Outcome:           substrate_types.OutcomeReversedDueToBPSDRecurrence,
			RetryEligibleDate: now.AddDate(0, 10, 0),
			DocumentedBy:      uuid.New(),
		},
		{
			ResidentID:        rid,
			InterventionType:  "benzodiazepine_deprescribing",
			AttemptDate:       now.AddDate(0, -6, 0),
			Outcome:           substrate_types.OutcomeReversedDueToFamilyRequest,
			RetryEligibleDate: now.AddDate(0, 6, 0),
			DocumentedBy:      uuid.New(),
		},
	}
	client := aggregation.NewInMemorySubstrateClient().WithFailedInterventions(rid, recs...)
	panel, err := aggregation.BuildFailedInterventionPanel(context.Background(), client, rid, now)
	if err != nil {
		t.Fatalf("BuildFailedInterventionPanel error: %v", err)
	}
	if len(panel.Cards) == 0 {
		t.Fatal("expected non-empty FIR card set")
	}
	for _, c := range panel.Cards {
		if len(c.SubstrateRefs) == 0 {
			t.Errorf(
				"FailedInterventionCard (intervention=%s attempted=%s) has no SubstrateRef — violates verification-not-belief (v1.0 Part 17)",
				c.Record.InterventionType, c.Record.AttemptDate.Format("2006-01-02"),
			)
		}
	}
}

// TestGoalsOfCarePanel_CurrentHasSubstrateRef enforces v-n-b for Panel G.
func TestGoalsOfCarePanel_CurrentHasSubstrateRef(t *testing.T) {
	rid := uuid.New()
	client := aggregation.NewInMemorySubstrateClient().WithGoalsOfCare(rid,
		substrate_types.GoalsOfCareEntry{
			State:         substrate_types.GoCStateComfortFocused,
			EffectiveFrom: time.Now().AddDate(0, -1, 0),
			DocumentedBy:  uuid.New(),
			SubstrateID:   uuid.New(),
		},
	)
	panel, err := aggregation.BuildGoalsOfCarePanel(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("BuildGoalsOfCarePanel error: %v", err)
	}
	if panel.Current == nil {
		t.Fatal("expected non-nil Current")
	}
	if len(panel.SubstrateRefs) == 0 {
		t.Error("GoalsOfCarePanel.Current populated but SubstrateRefs empty — violates verification-not-belief")
	}
}

// TestCareIntensityPanel_CurrentHasSubstrateRef enforces v-n-b for Panel I.
func TestCareIntensityPanel_CurrentHasSubstrateRef(t *testing.T) {
	rid := uuid.New()
	client := aggregation.NewInMemorySubstrateClient().WithCareIntensity(rid,
		substrate_types.CareIntensityEntry{
			Tag:           substrate_types.CareIntensityTagComfortFocused,
			EffectiveDate: time.Now().AddDate(0, -1, 0),
			DocumentedBy:  uuid.New(),
			SubstrateID:   uuid.New(),
		},
	)
	panel, err := aggregation.BuildCareIntensityPanel(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("BuildCareIntensityPanel error: %v", err)
	}
	if panel.Current == nil {
		t.Fatal("expected non-nil Current")
	}
	if len(panel.SubstrateRefs) == 0 {
		t.Error("CareIntensityPanel.Current populated but SubstrateRefs empty — violates verification-not-belief")
	}
}

// TestGoalsConflict_HasSubstrateRefs enforces v-n-b for conflict
// surfacing — every emitted GoalsConflict must carry SubstrateRefs
// (per v1.0 Part 9.4 + Part 17).
func TestGoalsConflict_HasSubstrateRefs(t *testing.T) {
	goc := &substrate_types.GoalsOfCareEntry{
		State:       substrate_types.GoCStatePalliative,
		SubstrateID: uuid.New(),
	}
	addCard := aggregation.PendingRecommendationCard{
		RecommendationID: uuid.New(),
		Type:             "ADD",
		SubstrateRefs: []aggregation.SubstrateRef{{
			Source:      "kb-32",
			Description: "ADD recommendation (rule ADD_VITD)",
		}},
	}
	conflicts := aggregation.DetectGoalsConflicts([]aggregation.PendingRecommendationCard{addCard}, goc)
	if len(conflicts) == 0 {
		t.Fatal("expected at least one GoalsConflict for ADD on palliative")
	}
	for _, c := range conflicts {
		if len(c.SubstrateRefs) == 0 {
			t.Errorf(
				"GoalsConflict for rec %s has no SubstrateRefs — violates verification-not-belief",
				c.RecommendationID,
			)
		}
	}
}
