// Package ethics_test provides end-to-end integration tests for the Phase 1c
// ethical architecture substrate. These tests exercise the assembled chain:
//
//	metadata recorder → ERM (with recommendation reasoner) → ethics log → hold orchestrator
//
// All implementations are in-memory — no HTTP, no database. This validates the
// wiring contracts between all Phase 1c components without external dependencies.
//
// VisibilityClass: AD
package ethics_test

import (
	"context"
	"testing"

	"github.com/cardiofit/shared/v2_substrate/ethics/decision_metadata"
	"github.com/cardiofit/shared/v2_substrate/ethics/erm"
	"github.com/cardiofit/shared/v2_substrate/ethics/erm/reasoners"
	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
	"github.com/cardiofit/shared/v2_substrate/ethics/incident_response"
	"github.com/google/uuid"
)

// fakeDivergence is a DivergenceSource that always returns a fixed value.
type fakeDivergence struct{ divergent bool }

func (f *fakeDivergence) IsDivergent(_ context.Context, _ string) (bool, error) {
	return f.divergent, nil
}

// TestE2E_LowAppropriatenessHeldByERMAndLogged verifies the rejection path:
// a recommendation with appropriateness < threshold results in OutcomeHold,
// a P2 concern is raised, metadata records ERMReviewed=true, and an ethics log
// entry of severity 3 and StatusOpen is created.
func TestE2E_LowAppropriatenessHeldByERMAndLogged(t *testing.T) {
	ctx := context.Background()
	decisionID := uuid.New()

	// Wire up all in-memory stores.
	logStore := ethics_log.NewInMemoryStore()
	logger := ethics_log.NewLogger(logStore)
	querier := ethics_log.NewQuerier(logStore)

	metaStore := decision_metadata.NewInMemoryStore()
	recorder := decision_metadata.NewRecorder(metaStore)

	// ERM with recommendation reasoner; fake divergence always returns false.
	module := erm.NewModule()
	module.Register(
		erm.DecisionTypeRecommendationDraft,
		reasoners.NewRecommendationReasoner(3.0, &fakeDivergence{divergent: false}),
	)

	// Build decision point: appropriateness=2.0 is below the 3.0 threshold.
	dp := erm.DecisionPoint{
		DecisionID:   decisionID,
		Component:    "craft-engine",
		DecisionType: erm.DecisionTypeRecommendationDraft,
		ProposedOutput: reasoners.RecommendationProposal{
			AppropriatenessScore: 2.0,
			RuleID:               "rule-metformin-001",
		},
	}

	outcome, concerns, err := module.Review(ctx, dp)
	if err != nil {
		t.Fatalf("ERM Review: %v", err)
	}

	// Assert outcome is Hold.
	if outcome != erm.OutcomeHold {
		t.Errorf("outcome = %v, want Hold", outcome)
	}

	// Assert at least one P2 concern.
	foundP2 := false
	for _, c := range concerns {
		if c.Principle == "P2" {
			foundP2 = true
			break
		}
	}
	if !foundP2 {
		t.Errorf("expected at least one P2 concern, got %+v", concerns)
	}

	// Record metadata: ERMReviewed=true, outcome=hold.
	holdStr := string(outcome)
	meta := decision_metadata.Metadata{
		DecisionID:           decisionID,
		Component:            "craft-engine",
		DecisionType:         string(erm.DecisionTypeRecommendationDraft),
		AffectedSubjectID:    "patient-001",
		AffectedSubjectClass: "resident",
		PrinciplesImplicated: []string{"P2"},
		ERMReviewed:          true,
		ERMOutcome:           &holdStr,
		ContestationEnabled:  true,
	}
	if err := recorder.Record(ctx, meta); err != nil {
		t.Fatalf("Record metadata: %v", err)
	}

	// Write ethics log entry: severity 3, StatusOpen, EntryTypeConcernFlagged.
	if err := logger.Append(ctx, ethics_log.Entry{
		DecisionID:  decisionID,
		EntryType:   ethics_log.EntryTypeConcernFlagged,
		Severity:    3,
		Description: "ERM held recommendation: appropriateness below threshold (P2)",
		Status:      ethics_log.StatusOpen,
	}); err != nil {
		t.Fatalf("Append ethics log: %v", err)
	}

	// Assert metadata is stored with ERMReviewed=true.
	storedMeta, err := metaStore.Get(ctx, decisionID)
	if err != nil {
		t.Fatalf("Get metadata: %v", err)
	}
	if storedMeta == nil {
		t.Fatalf("metadata not found for decisionID %v", decisionID)
	}
	if !storedMeta.ERMReviewed {
		t.Errorf("ERMReviewed = false, want true")
	}

	// Assert ethics log has exactly 1 entry for this decision, severity 3, open.
	entries, err := querier.ByDecision(ctx, decisionID)
	if err != nil {
		t.Fatalf("ByDecision: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 ethics log entry, got %d", len(entries))
	}
	if entries[0].Severity != 3 {
		t.Errorf("entry severity = %d, want 3", entries[0].Severity)
	}
	if entries[0].Status != ethics_log.StatusOpen {
		t.Errorf("entry status = %v, want open", entries[0].Status)
	}
}

// TestE2E_HighAppropriatenessApprovedAndLogged verifies the happy path:
// a recommendation with appropriateness=4.5 (above the 3.0 threshold) results
// in OutcomeApprove, and an EntryTypeDecision entry of severity 1 is logged.
func TestE2E_HighAppropriatenessApprovedAndLogged(t *testing.T) {
	ctx := context.Background()
	decisionID := uuid.New()

	logStore := ethics_log.NewInMemoryStore()
	logger := ethics_log.NewLogger(logStore)
	querier := ethics_log.NewQuerier(logStore)

	metaStore := decision_metadata.NewInMemoryStore()
	recorder := decision_metadata.NewRecorder(metaStore)

	module := erm.NewModule()
	module.Register(
		erm.DecisionTypeRecommendationDraft,
		reasoners.NewRecommendationReasoner(3.0, &fakeDivergence{divergent: false}),
	)

	dp := erm.DecisionPoint{
		DecisionID:   decisionID,
		Component:    "craft-engine",
		DecisionType: erm.DecisionTypeRecommendationDraft,
		ProposedOutput: reasoners.RecommendationProposal{
			AppropriatenessScore: 4.5,
			RuleID:               "rule-metformin-002",
		},
	}

	outcome, concerns, err := module.Review(ctx, dp)
	if err != nil {
		t.Fatalf("ERM Review: %v", err)
	}

	// Assert outcome is Approve.
	if outcome != erm.OutcomeApprove {
		t.Errorf("outcome = %v, want Approve", outcome)
	}
	if len(concerns) != 0 {
		t.Errorf("expected 0 concerns on approval, got %d", len(concerns))
	}

	// Record metadata.
	approveStr := string(outcome)
	if err := recorder.Record(ctx, decision_metadata.Metadata{
		DecisionID:           decisionID,
		Component:            "craft-engine",
		DecisionType:         string(erm.DecisionTypeRecommendationDraft),
		AffectedSubjectID:    "patient-002",
		AffectedSubjectClass: "resident",
		ERMReviewed:          true,
		ERMOutcome:           &approveStr,
		ContestationEnabled:  true,
	}); err != nil {
		t.Fatalf("Record metadata: %v", err)
	}

	// Write approval entry: EntryTypeDecision, severity 1.
	if err := logger.Append(ctx, ethics_log.Entry{
		DecisionID:  decisionID,
		EntryType:   ethics_log.EntryTypeDecision,
		Severity:    1,
		Description: "ERM approved recommendation: appropriateness meets threshold",
		Status:      ethics_log.StatusOpen,
	}); err != nil {
		t.Fatalf("Append ethics log: %v", err)
	}

	// Assert metadata stored with ERMReviewed=true.
	storedMeta, err := metaStore.Get(ctx, decisionID)
	if err != nil {
		t.Fatalf("Get metadata: %v", err)
	}
	if storedMeta == nil || !storedMeta.ERMReviewed {
		t.Errorf("ERMReviewed not set in metadata")
	}

	// Assert ethics log: 1 entry, EntryTypeDecision, severity 1, open.
	entries, err := querier.ByDecision(ctx, decisionID)
	if err != nil {
		t.Fatalf("ByDecision: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 ethics log entry, got %d", len(entries))
	}
	e := entries[0]
	if e.EntryType != ethics_log.EntryTypeDecision {
		t.Errorf("EntryType = %v, want EntryTypeDecision", e.EntryType)
	}
	if e.Severity != 1 {
		t.Errorf("Severity = %d, want 1", e.Severity)
	}
	if e.Status != ethics_log.StatusOpen {
		t.Errorf("Status = %v, want open", e.Status)
	}
}

// TestE2E_HoldOrchestratorTriggersOnHoldOutcome wires the full chain:
// metadata recorder → ERM → ethics log → hold orchestrator.
// The incident kind is "trust_violation" (severity 2) which satisfies the
// hold threshold (≤ 2), so the orchestrator handler MUST be called.
func TestE2E_HoldOrchestratorTriggersOnHoldOutcome(t *testing.T) {
	ctx := context.Background()
	decisionID := uuid.New()

	logStore := ethics_log.NewInMemoryStore()
	logger := ethics_log.NewLogger(logStore)
	querier := ethics_log.NewQuerier(logStore)

	metaStore := decision_metadata.NewInMemoryStore()
	recorder := decision_metadata.NewRecorder(metaStore)

	module := erm.NewModule()
	module.Register(
		erm.DecisionTypeRecommendationDraft,
		reasoners.NewRecommendationReasoner(3.0, &fakeDivergence{divergent: false}),
	)

	// Force a Hold outcome via low appropriateness.
	dp := erm.DecisionPoint{
		DecisionID:   decisionID,
		Component:    "craft-engine",
		DecisionType: erm.DecisionTypeRecommendationDraft,
		ProposedOutput: reasoners.RecommendationProposal{
			AppropriatenessScore: 1.5,
			RuleID:               "rule-metformin-003",
		},
	}

	outcome, _, err := module.Review(ctx, dp)
	if err != nil {
		t.Fatalf("ERM Review: %v", err)
	}
	if outcome != erm.OutcomeHold {
		t.Fatalf("expected Hold, got %v", outcome)
	}

	// Record metadata.
	holdStr := string(outcome)
	if err := recorder.Record(ctx, decision_metadata.Metadata{
		DecisionID:           decisionID,
		Component:            "craft-engine",
		DecisionType:         string(erm.DecisionTypeRecommendationDraft),
		AffectedSubjectID:    "patient-003",
		AffectedSubjectClass: "resident",
		PrinciplesImplicated: []string{"P2"},
		ERMReviewed:          true,
		ERMOutcome:           &holdStr,
		ContestationEnabled:  true,
	}); err != nil {
		t.Fatalf("Record metadata: %v", err)
	}

	// Append ethics log entry.
	if err := logger.Append(ctx, ethics_log.Entry{
		DecisionID:  decisionID,
		EntryType:   ethics_log.EntryTypeConcernFlagged,
		Severity:    3,
		Description: "ERM hold due to low appropriateness; escalating via trust_violation hold",
		Status:      ethics_log.StatusOpen,
	}); err != nil {
		t.Fatalf("Append ethics log: %v", err)
	}

	// Wire hold orchestrator: kind=trust_violation → severity 2 → triggers.
	holdCalled := false
	orchestrator := incident_response.NewOrchestrator()
	orchestrator.Register(incident_response.HoldHandlerFunc(func(_ context.Context, inc incident_response.Incident) error {
		holdCalled = true
		return nil
	}))

	inc := incident_response.Incident{
		ID:                 uuid.New(),
		Severity:           incident_response.Classify("trust_violation"), // → 2
		Kind:               "trust_violation",
		AffectedComponents: []string{"craft-engine"},
		HoldActive:         true,
		Description:        "ERM-triggered hold on recommendation decision",
	}
	if err := orchestrator.Trigger(ctx, inc); err != nil {
		t.Fatalf("orchestrator.Trigger: %v", err)
	}

	// Assert hold handler was called.
	if !holdCalled {
		t.Errorf("hold orchestrator handler was NOT called for trust_violation (severity 2)")
	}

	// Assert ethics log still has the entry.
	entries, err := querier.ByDecision(ctx, decisionID)
	if err != nil {
		t.Fatalf("ByDecision: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 ethics log entry, got %d", len(entries))
	}

	// Assert metadata stored correctly.
	storedMeta, err := metaStore.Get(ctx, decisionID)
	if err != nil || storedMeta == nil {
		t.Fatalf("metadata lookup failed: err=%v meta=%v", err, storedMeta)
	}
	if !storedMeta.ERMReviewed {
		t.Errorf("ERMReviewed = false, want true")
	}
}
