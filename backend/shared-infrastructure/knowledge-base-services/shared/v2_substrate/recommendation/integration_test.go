package recommendation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// TestIntegration_FullLifecycleEndToEnd is the executable definition of
// "Recommendation entity + lifecycle is shipped" per Plan 0.1 Task 9.
//
// It exercises the full chain end-to-end against real Postgres:
//   Create → drafted → submitted → viewed → decided → implemented
//   → monitoring-active (5 transitions; Option B)
//
// We stop at monitoring-active rather than walking all the way to closed
// because RIR_SEMANTICS.md documents that the substrate-table RIR query
// excludes `closed` rows from the actioned numerator (state-based, no
// history). The closed-state under-count is already covered by
// TestComputeRIR_OnlyImplementedCounts in rir_test.go; this end-to-end
// test asserts the RIR happy-path (Actioned == Submitted == 1).
//
// Verifies:
//   - All 5 lifecycle transitions succeed without error
//   - Final state is monitoring-active
//   - submitted_at and decided_at columns auto-populated by SQL triggers
//   - closed_at remains nil (we never reached closed)
//   - 5 EvidenceTraceNodes captured by the adapter (one per transition),
//     each with the expected StateChangeType
//   - RIR computation reports Submitted=1, Actioned=1, RatePercent=100
func TestIntegration_FullLifecycleEndToEnd(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	ctx := context.Background()

	store := NewPostgresStore(db)
	nodeWriter := &fakeNodeWriter{}
	edges := NewEvidenceTraceAdapter(nodeWriter)
	lc := NewLifecycle(store, edges, AlwaysPassConsentChecker{})

	author := uuid.New()
	rec := models.Recommendation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		AuthorID:   author,
		State:      models.RecommendationStateDrafted,
		Type:       models.RecommendationTypeStop,
		Urgency:    models.RecommendationUrgencyAmber,
		Title:      "Cease oxybutynin",
		ClinicalContent: models.ClinicalContent{
			Issue:           "ACB",
			ClinicalContext: "87yo, eGFR 32, recent fall",
			Rationale:       "DBI 0.8 attributable",
			EvidenceRefs:    []string{"ADG-2025"},
			ProposedPlan:    "cease oxybutynin 5mg BD",
			MonitoringPlan:  "voiding diary 14 days",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.Create(ctx, &rec); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM recommendations WHERE id = $1", rec.ID)
	})

	steps := []string{
		models.RecommendationStateSubmitted,
		models.RecommendationStateViewed,
		models.RecommendationStateDecided,
		models.RecommendationStateImplemented,
		models.RecommendationStateMonitoringActive,
	}
	for _, s := range steps {
		err := lc.Transition(ctx, TransitionRequest{
			RecommendationID: rec.ID,
			ToState:          s,
			ActorID:          uuid.New(),
			ActorClass:       ActorClassHuman,
			ReasoningSummary: "test step " + s,
		})
		if err != nil {
			t.Fatalf("transition to %s: %v", s, err)
		}
	}

	// Final state checks
	got, err := store.Get(ctx, rec.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.State != models.RecommendationStateMonitoringActive {
		t.Errorf("final state = %q want monitoring-active", got.State)
	}
	if got.SubmittedAt == nil {
		t.Errorf("submitted_at must be populated after passing through submitted")
	}
	if got.DecidedAt == nil {
		t.Errorf("decided_at must be populated after passing through decided (RIR invariant)")
	}
	if got.ClosedAt != nil {
		t.Errorf("closed_at must remain nil; we never reached closed (got %v)", got.ClosedAt)
	}

	// EvidenceTrace capture: one node per transition
	if len(nodeWriter.nodes) != len(steps) {
		t.Errorf("expected %d evidence nodes; got %d", len(steps), len(nodeWriter.nodes))
	}
	expectedTransitions := []string{
		"drafted -> submitted",
		"submitted -> viewed",
		"viewed -> decided",
		"decided -> implemented",
		"implemented -> monitoring-active",
	}
	for i, expectedSCT := range expectedTransitions {
		if i >= len(nodeWriter.nodes) {
			break
		}
		if nodeWriter.nodes[i].StateChangeType != expectedSCT {
			t.Errorf("node[%d] state_change_type = %q want %q",
				i, nodeWriter.nodes[i].StateChangeType, expectedSCT)
		}
		if nodeWriter.nodes[i].ReasoningSummary == nil ||
			!strings.Contains(nodeWriter.nodes[i].ReasoningSummary.Text, "actor_class=human") {
			t.Errorf("node[%d] missing actor_class in reasoning: %+v",
				i, nodeWriter.nodes[i].ReasoningSummary)
		}
	}

	// RIR check: 1 submitted in window, 1 actioned (final state =
	// monitoring-active is counted), 100%.
	rir, err := ComputeRIR(ctx, db, author, 28*24*time.Hour)
	if err != nil {
		t.Fatalf("rir: %v", err)
	}
	if rir.Submitted != 1 {
		t.Errorf("RIR Submitted = %d want 1", rir.Submitted)
	}
	if rir.Actioned != 1 {
		t.Errorf("RIR Actioned = %d want 1", rir.Actioned)
	}
	if rir.RatePercent < 99.9 || rir.RatePercent > 100.1 {
		t.Errorf("RIR RatePercent = %v want ~100", rir.RatePercent)
	}
}
