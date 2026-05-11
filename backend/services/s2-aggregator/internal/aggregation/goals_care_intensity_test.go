package aggregation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

func mkGoC(state string, effectiveFrom time.Time) substrate_types.GoalsOfCareEntry {
	return substrate_types.GoalsOfCareEntry{
		State:         state,
		EffectiveFrom: effectiveFrom,
		DocumentedBy:  uuid.New(),
		SubstrateID:   uuid.New(),
	}
}

func mkCI(tag string, effective time.Time) substrate_types.CareIntensityEntry {
	return substrate_types.CareIntensityEntry{
		Tag:           tag,
		EffectiveDate: effective,
		DocumentedBy:  uuid.New(),
		SubstrateID:   uuid.New(),
	}
}

func TestBuildGoalsOfCarePanel_EmptyState(t *testing.T) {
	rid := uuid.New()
	client := NewInMemorySubstrateClient()
	panel, err := BuildGoalsOfCarePanel(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if panel.Current != nil {
		t.Error("Current must be nil when no GoC documented")
	}
	if panel.History == nil {
		t.Error("History must be non-nil empty slice")
	}
	if panel.FreshnessFlag {
		t.Error("FreshnessFlag must be false when no documentation exists")
	}
}

func TestBuildGoalsOfCarePanel_FreshnessSoftAndStrong(t *testing.T) {
	rid := uuid.New()

	// Soft: 8 months old → soft flag
	soft := mkGoC(substrate_types.GoCStateComfortFocused, time.Now().Add(-8*30*24*time.Hour))
	client := NewInMemorySubstrateClient().WithGoalsOfCare(rid, soft)
	panel, err := BuildGoalsOfCarePanel(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !panel.FreshnessFlag {
		t.Error("expected FreshnessFlag=true for 8mo-old GoC")
	}
	if panel.FreshnessReason != GoCFreshnessReasonSoft {
		t.Errorf("FreshnessReason = %q, want %q", panel.FreshnessReason, GoCFreshnessReasonSoft)
	}

	// Strong: 14 months old → strong flag
	rid2 := uuid.New()
	strong := mkGoC(substrate_types.GoCStatePalliative, time.Now().Add(-14*30*24*time.Hour))
	client2 := NewInMemorySubstrateClient().WithGoalsOfCare(rid2, strong)
	panel2, err := BuildGoalsOfCarePanel(context.Background(), client2, rid2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !panel2.FreshnessFlag {
		t.Error("expected FreshnessFlag=true for 14mo-old GoC")
	}
	if panel2.FreshnessReason != GoCFreshnessReasonStrong {
		t.Errorf("FreshnessReason = %q, want %q", panel2.FreshnessReason, GoCFreshnessReasonStrong)
	}

	// Fresh: 2 months old → no flag
	rid3 := uuid.New()
	fresh := mkGoC(substrate_types.GoCStateActiveTreatment, time.Now().Add(-60*24*time.Hour))
	client3 := NewInMemorySubstrateClient().WithGoalsOfCare(rid3, fresh)
	panel3, err := BuildGoalsOfCarePanel(context.Background(), client3, rid3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if panel3.FreshnessFlag {
		t.Error("expected FreshnessFlag=false for 2mo-old GoC")
	}
}

func TestBuildGoalsOfCarePanel_HasSubstrateRefForCurrent(t *testing.T) {
	rid := uuid.New()
	client := NewInMemorySubstrateClient().WithGoalsOfCare(rid,
		mkGoC(substrate_types.GoCStateComfortFocused, time.Now()),
	)
	panel, err := BuildGoalsOfCarePanel(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(panel.SubstrateRefs) == 0 {
		t.Error("expected ≥1 SubstrateRef when Current is non-nil (verification-not-belief)")
	}
}

func TestBuildCareIntensityPanel_SparseDataFlag(t *testing.T) {
	rid := uuid.New()
	// One entry → sparse
	client := NewInMemorySubstrateClient().WithCareIntensity(rid,
		mkCI(substrate_types.CareIntensityTagActiveTreatment, time.Now()),
	)
	panel, err := BuildCareIntensityPanel(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !panel.SparseDataFlag {
		t.Error("expected SparseDataFlag=true with single entry")
	}
	if panel.Current == nil {
		t.Error("expected Current to be populated")
	}

	// Two entries → not sparse
	rid2 := uuid.New()
	client2 := NewInMemorySubstrateClient().WithCareIntensity(rid2,
		mkCI(substrate_types.CareIntensityTagActiveTreatment, time.Now().AddDate(0, -3, 0)),
		mkCI(substrate_types.CareIntensityTagRehabilitation, time.Now()),
	)
	panel2, err := BuildCareIntensityPanel(context.Background(), client2, rid2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if panel2.SparseDataFlag {
		t.Error("expected SparseDataFlag=false with 2 entries")
	}
}

func TestBuildCareIntensityPanel_EmptyState(t *testing.T) {
	rid := uuid.New()
	client := NewInMemorySubstrateClient()
	panel, err := BuildCareIntensityPanel(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if panel.Current != nil {
		t.Error("Current must be nil")
	}
	if panel.History == nil {
		t.Error("History must be non-nil empty slice")
	}
	if !panel.SparseDataFlag {
		t.Error("empty history (<2) must set SparseDataFlag=true")
	}
}

func TestDetectGoalsConflicts_AddOnPalliative(t *testing.T) {
	gocID := uuid.New()
	goc := &substrate_types.GoalsOfCareEntry{
		State:         substrate_types.GoCStatePalliative,
		EffectiveFrom: time.Now().AddDate(0, -1, 0),
		SubstrateID:   gocID,
	}
	addCard := PendingRecommendationCard{
		RecommendationID: uuid.New(),
		Type:             "ADD",
		SubstrateRefs:    []SubstrateRef{{Source: "kb-32", Description: "ADD recommendation (rule ADD_VITD)"}},
	}
	stopCard := PendingRecommendationCard{
		RecommendationID: uuid.New(),
		Type:             "STOP",
		SubstrateRefs:    []SubstrateRef{{Source: "kb-32", Description: "STOP recommendation (rule STOP_STATIN)"}},
	}
	conflicts := DetectGoalsConflicts([]PendingRecommendationCard{addCard, stopCard}, goc)
	if got := len(conflicts); got != 1 {
		t.Fatalf("expected 1 conflict (ADD on palliative), got %d", got)
	}
	if conflicts[0].RecommendationID != addCard.RecommendationID {
		t.Errorf("expected conflict on ADD card, got %s", conflicts[0].RecommendationID)
	}
	if conflicts[0].CurrentGoCState != substrate_types.GoCStatePalliative {
		t.Errorf("CurrentGoCState = %q, want %q", conflicts[0].CurrentGoCState, substrate_types.GoCStatePalliative)
	}
	if len(conflicts[0].SubstrateRefs) < 2 {
		t.Errorf("expected ≥2 SubstrateRefs (GoC + rec); got %d", len(conflicts[0].SubstrateRefs))
	}
}

func TestDetectGoalsConflicts_AddOnComfortFocused_LegacyEquivalence(t *testing.T) {
	// Verify the legacy short form "comfort" normalizes to comfort_focused.
	gocLegacy := &substrate_types.GoalsOfCareEntry{
		State:       "comfort", // legacy short form
		SubstrateID: uuid.New(),
	}
	addCard := PendingRecommendationCard{
		RecommendationID: uuid.New(),
		Type:             "ADD",
		SubstrateRefs:    []SubstrateRef{{Source: "kb-32", Description: "ADD recommendation (rule ADD_VITD)"}},
	}
	conflicts := DetectGoalsConflicts([]PendingRecommendationCard{addCard}, gocLegacy)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict (ADD on comfort/comfort_focused), got %d", len(conflicts))
	}
}

func TestDetectGoalsConflicts_StopPsychotropicOnActive_Informational(t *testing.T) {
	goc := &substrate_types.GoalsOfCareEntry{
		State:       substrate_types.GoCStateActiveTreatment,
		SubstrateID: uuid.New(),
	}
	stopPsych := PendingRecommendationCard{
		RecommendationID: uuid.New(),
		Type:             "STOP",
		SubstrateRefs:    []SubstrateRef{{Source: "kb-32", Description: "STOP recommendation (rule STOP_PSYCH_HALDOL)"}},
	}
	stopStatin := PendingRecommendationCard{
		RecommendationID: uuid.New(),
		Type:             "STOP",
		SubstrateRefs:    []SubstrateRef{{Source: "kb-32", Description: "STOP recommendation (rule STOP_STATIN)"}},
	}
	conflicts := DetectGoalsConflicts([]PendingRecommendationCard{stopPsych, stopStatin}, goc)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 informational conflict (STOP psychotropic on active), got %d", len(conflicts))
	}
	if conflicts[0].RecommendationID != stopPsych.RecommendationID {
		t.Error("expected the psychotropic STOP to be flagged, not the statin STOP")
	}
}

func TestDetectGoalsConflicts_EmptyWhenGoCNil(t *testing.T) {
	got := DetectGoalsConflicts([]PendingRecommendationCard{
		{RecommendationID: uuid.New(), Type: "ADD"},
	}, nil)
	if got == nil {
		t.Fatal("expected non-nil empty slice when GoC is nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty conflicts when GoC is nil, got %d", len(got))
	}
}

func TestDetectGoalsConflicts_AddOnActive_NoConflict(t *testing.T) {
	goc := &substrate_types.GoalsOfCareEntry{
		State:       substrate_types.GoCStateActiveTreatment,
		SubstrateID: uuid.New(),
	}
	addCard := PendingRecommendationCard{
		RecommendationID: uuid.New(),
		Type:             "ADD",
		SubstrateRefs:    []SubstrateRef{{Source: "kb-32", Description: "ADD recommendation (rule ADD_VITD)"}},
	}
	if got := DetectGoalsConflicts([]PendingRecommendationCard{addCard}, goc); len(got) != 0 {
		t.Errorf("ADD on active_treatment should not conflict; got %d", len(got))
	}
}
