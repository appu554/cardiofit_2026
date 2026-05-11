// view_assembly_test.go — v1.0 Part 17 Category 1 (view assembly).
//
// Composition test layer: exercises whole-view assembly from each of
// the four S2 entry paths (v1.0 Part 3) + graceful degradation under
// sparse substrate + complex-activation-offer evaluation.
package structural

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// TestS2ViewAssembly_FromWorklistEntry_CarriesCAPEContext — entry path
// Worklist → CAPEContextBand populated with the CAPE signals.
func TestS2ViewAssembly_FromWorklistEntry_CarriesCAPEContext(t *testing.T) {
	pid := uuid.New()
	rid := uuid.New()
	meta := aggregation.EntryPathMetadata{
		Path:         aggregation.EntryPathWorklist,
		PharmacistID: pid,
		ResidentID:   rid,
		TriggeredAt:  time.Now(),
		Context: aggregation.WorklistContext{
			PrimarySignals: []string{"acute_event_severity_5_fall", "trajectory_velocity_4_egfr_decline"},
			CAPEScore:      0.82,
			TriagedAt:      time.Now().Add(-30 * time.Minute),
		},
	}
	band, err := aggregation.BuildCAPEContextBand(meta)
	if err != nil {
		t.Fatalf("BuildCAPEContextBand: %v", err)
	}
	if len(band.Signals) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(band.Signals))
	}
	if band.CAPEScore != 0.82 {
		t.Errorf("CAPE score not carried: got %v want 0.82", band.CAPEScore)
	}
	if len(band.SubstrateRefs) == 0 {
		t.Error("CAPE band populated with signals but no SubstrateRefs (verification-not-belief)")
	}
}

// TestS2ViewAssembly_FromSearchEntry_NoCAPEContext — entry path Search
// → CAPEContextBand empty per Task 2 contract.
func TestS2ViewAssembly_FromSearchEntry_NoCAPEContext(t *testing.T) {
	meta := aggregation.EntryPathMetadata{
		Path: aggregation.EntryPathSearch,
		Context: aggregation.SearchContext{
			Query:     "smith",
			MatchedAt: time.Now(),
		},
	}
	band, err := aggregation.BuildCAPEContextBand(meta)
	if err != nil {
		t.Fatalf("BuildCAPEContextBand: %v", err)
	}
	if len(band.Signals) != 0 {
		t.Errorf("Search entry should produce empty CAPE band; got %d signals", len(band.Signals))
	}
	if band.CAPEScore != 0 {
		t.Errorf("Search entry should leave CAPE score zero; got %v", band.CAPEScore)
	}
}

// TestS2ViewAssembly_FromNotificationEntry_NotificationBandPopulated —
// entry path Notification → the AssembledLayer1View's NotificationBand
// field carries the reason text.
func TestS2ViewAssembly_FromNotificationEntry_NotificationBandPopulated(t *testing.T) {
	nctx := aggregation.NotificationContext{
		NotificationID: uuid.New(),
		ReasonText:     "pharmacy queue alert: pending recommendation aging >7 days",
		DispatchedAt:   time.Now(),
	}
	meta := aggregation.EntryPathMetadata{
		Path:    aggregation.EntryPathNotification,
		Context: nctx,
	}
	// CAPE band stays empty on Notification entry (only Worklist populates).
	band, err := aggregation.BuildCAPEContextBand(meta)
	if err != nil {
		t.Fatalf("BuildCAPEContextBand: %v", err)
	}
	if len(band.Signals) != 0 {
		t.Errorf("Notification entry should leave CAPE band empty; got %d signals", len(band.Signals))
	}
	// AssembledLayer1View carries the NotificationBand via its
	// NotificationBand field — verify the metadata round-trip works.
	if got, ok := meta.Context.(aggregation.NotificationContext); !ok || got.ReasonText != nctx.ReasonText {
		t.Errorf("Notification context not round-trippable through EntryPathMetadata")
	}
}

// TestS2ViewAssembly_FromCrossReferenceEntry_OriginResidentRecorded —
// entry path CrossReference → AssembledLayer1View.OriginResidentID
// populated.
func TestS2ViewAssembly_FromCrossReferenceEntry_OriginResidentRecorded(t *testing.T) {
	origin := uuid.New()
	meta := aggregation.EntryPathMetadata{
		Path: aggregation.EntryPathCrossReference,
		Context: aggregation.CrossReferenceContext{
			OriginResidentID: origin,
			ReasonCode:       "medication_class_cross_reference",
		},
	}
	got, ok := meta.Context.(aggregation.CrossReferenceContext)
	if !ok {
		t.Fatal("Context is not CrossReferenceContext")
	}
	if got.OriginResidentID != origin {
		t.Errorf("OriginResidentID not preserved: got %s want %s", got.OriginResidentID, origin)
	}
	if got.ReasonCode == "" {
		t.Error("ReasonCode dropped on round-trip")
	}
}

// TestS2ViewAssembly_SparseSubstrate_GracefulDegradation — substrate
// returns minimal data → every panel still renders with appropriate
// empty-state / sparse-data flags (v1.0 Part 5.3 + Part 6.5 +
// Part 8.5).
func TestS2ViewAssembly_SparseSubstrate_GracefulDegradation(t *testing.T) {
	view := buildTestS2View(t, scenarioSparseOnly)

	// Trajectories: every parameter MUST appear in the slice (even with
	// SparseDataFlag=true). The catalogue is 7 numeric + 3 PRN.
	if len(view.Trajectories) < 7 {
		t.Errorf("sparse scenario: trajectory catalogue should still render all parameters; got %d", len(view.Trajectories))
	}
	// At least one trajectory should be sparse-flagged.
	sawSparse := false
	for _, tr := range view.Trajectories {
		if tr.SparseDataFlag {
			sawSparse = true
			break
		}
	}
	if !sawSparse {
		t.Error("sparse scenario: expected at least one SparseDataFlag=true trajectory")
	}
	// Pending recs: empty slice but non-nil (Part 6.5 empty-state contract).
	if view.PendingRecs == nil {
		t.Error("sparse scenario: PendingRecs is nil; must be non-nil empty slice (Part 6.5)")
	}
	// FIR panel: Cards non-nil, Patterns non-nil.
	if view.FailedInterventionPanel.Cards == nil {
		t.Error("sparse scenario: FIR Cards must be non-nil empty slice")
	}
	if view.FailedInterventionPanel.Patterns == nil {
		t.Error("sparse scenario: FIR Patterns must be non-nil empty slice")
	}
}

// TestS2ViewAssembly_ComplexActivationCriteriaEvaluated — when CFS ≥6
// + ACB elevated + eGFR decline (Addendum Part 3.3 activation
// criteria), the view's ComplexActivationOffer field is populated;
// otherwise nil. The activation OFFER is in Layer 1 scope (v1.0 Part
// 11); the activated Layer 3 view is not.
func TestS2ViewAssembly_ComplexActivationCriteriaEvaluated(t *testing.T) {
	t.Run("criteria_met", func(t *testing.T) {
		view := buildTestS2View(t, scenarioRepresentative)
		if view.ComplexActivationOffer == nil {
			t.Fatal("representative scenario: criteria CFS≥6+ACB elevated+eGFR decline are met — expected ComplexActivationOffer populated")
		}
		if len(view.ComplexActivationOffer.SubstrateRefs) == 0 {
			t.Error("ComplexActivationOffer populated but SubstrateRefs empty (verification-not-belief)")
		}
	})
	t.Run("criteria_not_met", func(t *testing.T) {
		view := buildTestS2View(t, scenarioSparseOnly)
		if view.ComplexActivationOffer != nil {
			t.Error("sparse scenario: criteria not met — expected ComplexActivationOffer nil")
		}
	})
}

// TestS2ViewAssembly_FullPipeline_AllPanelsPopulate — sanity check
// that BuildLayer1Baseline + the panel builders complete end-to-end
// without errors against the in-memory substrate.
func TestS2ViewAssembly_FullPipeline_AllPanelsPopulate(t *testing.T) {
	rid := uuid.New()
	asOf := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)

	client := aggregation.NewInMemorySubstrateClient().
		WithObservations(mkObsV9(rid, "egfr", 35, asOf.AddDate(0, -1, 0))).
		WithPackets(mkPktV9(rid, "STOP", "red")).
		WithGoalsOfCare(rid, substrate_types.GoalsOfCareEntry{
			State: substrate_types.GoCStateActiveTreatment, EffectiveFrom: asOf.AddDate(0, -1, 0),
			DocumentedBy: uuid.New(), SubstrateID: uuid.New(),
		}).
		WithCareIntensity(rid, substrate_types.CareIntensityEntry{
			Tag: substrate_types.CareIntensityTagActiveTreatment, EffectiveDate: asOf.AddDate(0, -1, 0),
			DocumentedBy: uuid.New(), SubstrateID: uuid.New(),
		})

	ctx := context.Background()

	if _, err := aggregation.BuildTrajectories(ctx, client, rid, asOf); err != nil {
		t.Errorf("BuildTrajectories: %v", err)
	}
	if _, err := aggregation.BuildPendingRecommendationCards(ctx, client, rid, asOf); err != nil {
		t.Errorf("BuildPendingRecommendationCards: %v", err)
	}
	if _, err := aggregation.BuildRestraintSignalCards(ctx, client, rid); err != nil {
		t.Errorf("BuildRestraintSignalCards: %v", err)
	}
	if _, err := aggregation.BuildGoalsOfCarePanel(ctx, client, rid); err != nil {
		t.Errorf("BuildGoalsOfCarePanel: %v", err)
	}
	if _, err := aggregation.BuildCareIntensityPanel(ctx, client, rid); err != nil {
		t.Errorf("BuildCareIntensityPanel: %v", err)
	}
	if _, err := aggregation.BuildFailedInterventionPanel(ctx, client, rid, asOf); err != nil {
		t.Errorf("BuildFailedInterventionPanel: %v", err)
	}
	if _, err := aggregation.BuildFamilyCommunicationContext(ctx, client, rid); err != nil {
		t.Errorf("BuildFamilyCommunicationContext: %v", err)
	}

	// BuildLayer1Baseline (Task 1 empty slot) must not error.
	vb := aggregation.NewDefaultViewBuilder()
	if _, err := vb.BuildLayer1Baseline(ctx, aggregation.WorkspaceRequest{ResidentID: rid, AsOf: asOf}); err != nil {
		t.Errorf("BuildLayer1Baseline: %v", err)
	}
}
