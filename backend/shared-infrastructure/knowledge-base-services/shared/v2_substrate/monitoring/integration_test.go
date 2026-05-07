package monitoring

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// TestIntegration_ThresholdCrossClosesOutcomeLoop is the executable
// definition of "Plan 0.3 closes the v2 §3 line 136 outcome loop."
//
// Scenario walks through the full chain:
//  1. Seed an active monitoring plan with a potassium obligation
//     (threshold: value > 5.5)
//  2. An Observation lands with value 5.8 (crosses threshold)
//  3. ThresholdEvaluator marks obligation fulfilled + threshold-crossed
//     AND emits an Event of type monitoring_threshold_crossed
//     AND transitions plan to escalated via Lifecycle (which emits one
//     EvidenceEdge from active → escalated with ActorClassAlgorithmic)
//
// Asserts every link of the chain.
func TestIntegration_ThresholdCrossClosesOutcomeLoop(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	ctx := context.Background()

	// Real components
	store := NewPostgresStore(db)
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	emitter := &fakeEventEmitter{}
	frozen := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	evaluator := NewThresholdEvaluator(store, lc, emitter)
	evaluator.now = func() time.Time { return frozen }

	// Seed: active monitoring plan with one potassium obligation
	resident := uuid.New()
	rec := uuid.New()
	plan := models.MonitoringPlan{
		ID:               uuid.New(),
		RecommendationID: rec,
		ResidentID:       resident,
		State:            models.MonitoringPlanStateActive,
		Obligations: []models.MonitoringObligation{
			{
				Type:            models.MonitoringObligationTypeObservation,
				ObservationCode: "potassium",
				FrequencyHours:  24,
				DueAt:           frozen.Add(2 * time.Hour),
				ThresholdSpec:   "value > 5.5",
			},
		},
		StartedAt:           frozen.Add(-7 * 24 * time.Hour),
		ExpectedEndAt:       frozen.Add(7 * 24 * time.Hour),
		EscalateAfterMissed: 2,
		CreatedAt:           frozen.Add(-7 * 24 * time.Hour),
		UpdatedAt:           frozen.Add(-7 * 24 * time.Hour),
	}
	if err := store.Create(ctx, &plan); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM monitoring_plans WHERE id = $1", plan.ID)
	})

	// An Observation lands that crosses the threshold (5.8 > 5.5)
	value := 5.8
	obsID := uuid.New()
	obs := models.Observation{
		ID:         obsID,
		ResidentID: resident,
		LOINCCode:  "potassium",
		Value:      &value,
		ObservedAt: frozen,
	}
	if err := evaluator.Evaluate(ctx, obs); err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	// 1. Plan state transitions to escalated
	got, err := store.Get(ctx, plan.ID)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}
	if got.State != models.MonitoringPlanStateEscalated {
		t.Errorf("plan state = %q, want escalated", got.State)
	}

	// 2. Obligation marked fulfilled by the observation
	if got.Obligations[0].FulfilledAt == nil {
		t.Errorf("FulfilledAt nil after threshold-cross observation")
	}
	if got.Obligations[0].FulfilledByObsID == nil ||
		*got.Obligations[0].FulfilledByObsID != obsID {
		t.Errorf("FulfilledByObsID not propagated: %v",
			got.Obligations[0].FulfilledByObsID)
	}

	// 3. Threshold marked crossed
	if got.Obligations[0].ThresholdCrossedAt == nil {
		t.Errorf("ThresholdCrossedAt nil after value crossed threshold")
	}

	// 4. Exactly one Event emitted of type monitoring_threshold_crossed
	if len(emitter.events) != 1 {
		t.Fatalf("expected 1 Event emitted; got %d", len(emitter.events))
	}
	ev := emitter.events[0]
	if ev.EventType != models.EventTypeMonitoringThresholdCrossed {
		t.Errorf("Event type = %q, want monitoring_threshold_crossed",
			ev.EventType)
	}
	if ev.ResidentID != resident {
		t.Errorf("Event ResidentID wrong")
	}
	// 4b. Event payload includes structured details about the cross
	if len(ev.DescriptionStructured) == 0 {
		t.Errorf("Event DescriptionStructured empty")
	}
	var payload map[string]any
	if err := json.Unmarshal(ev.DescriptionStructured, &payload); err != nil {
		t.Fatalf("Event DescriptionStructured invalid JSON: %v", err)
	}
	if payload["threshold_spec"] != "value > 5.5" {
		t.Errorf("payload threshold_spec = %v, want 'value > 5.5'", payload["threshold_spec"])
	}

	// 5. Exactly one EvidenceEdge emitted by the Lifecycle transition
	if len(edges.emitted) != 1 {
		t.Fatalf("expected 1 EvidenceEdge; got %d", len(edges.emitted))
	}
	edge := edges.emitted[0]
	if edge.FromState != models.MonitoringPlanStateActive ||
		edge.ToState != models.MonitoringPlanStateEscalated {
		t.Errorf("edge states wrong: %s -> %s", edge.FromState, edge.ToState)
	}
	if edge.ActorClass != ActorClassAlgorithmic {
		t.Errorf("edge ActorClass = %v, want algorithmic", edge.ActorClass)
	}
	if edge.PlanID != plan.ID {
		t.Errorf("edge PlanID wrong")
	}
}

// TestIntegration_EscalatorMissedObligationsClosesOutcomeLoop is a parallel
// integration test covering the escalator path: when observations FAIL TO
// LAND, the escalator emits an Event and escalates.
//
// Scenario:
//  1. Active plan with 2 obligations both past-due-plus-grace, neither
//     fulfilled
//  2. Escalator.RunOnce sweeps
//  3. Plan escalated; one Event of type monitoring_obligation_missed
//     emitted; one EvidenceEdge from active → escalated
func TestIntegration_EscalatorMissedObligationsClosesOutcomeLoop(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	ctx := context.Background()

	store := NewPostgresStore(db)
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	emitter := &fakeEventEmitter{}
	frozen := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	graceWindow := 1 * time.Hour
	esc := NewEscalator(store, lc, emitter, graceWindow)
	esc.now = func() time.Time { return frozen }

	resident := uuid.New()
	plan := models.MonitoringPlan{
		ID:               uuid.New(),
		RecommendationID: uuid.New(),
		ResidentID:       resident,
		State:            models.MonitoringPlanStateActive,
		Obligations: []models.MonitoringObligation{
			{
				Type:            models.MonitoringObligationTypeObservation,
				ObservationCode: "potassium",
				DueAt:           frozen.Add(-3 * time.Hour),
				ThresholdSpec:   "value > 5.5",
			},
			{
				Type:            models.MonitoringObligationTypeObservation,
				ObservationCode: "creatinine",
				DueAt:           frozen.Add(-2 * time.Hour),
				ThresholdSpec:   "value > 200",
			},
		},
		StartedAt:           frozen.Add(-7 * 24 * time.Hour),
		ExpectedEndAt:       frozen.Add(-1 * time.Hour),
		EscalateAfterMissed: 2,
		CreatedAt:           frozen.Add(-7 * 24 * time.Hour),
		UpdatedAt:           frozen.Add(-7 * 24 * time.Hour),
	}
	if err := store.Create(ctx, &plan); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM monitoring_plans WHERE id = $1", plan.ID)
	})

	if err := esc.RunOnce(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}

	got, _ := store.Get(ctx, plan.ID)
	if got.State != models.MonitoringPlanStateEscalated {
		t.Errorf("plan state = %q, want escalated", got.State)
	}
	if len(emitter.events) != 1 {
		t.Fatalf("expected 1 Event; got %d", len(emitter.events))
	}
	if emitter.events[0].EventType != models.EventTypeMonitoringObligationMissed {
		t.Errorf("Event type = %q, want monitoring_obligation_missed",
			emitter.events[0].EventType)
	}
	if len(edges.emitted) != 1 {
		t.Fatalf("expected 1 EvidenceEdge; got %d", len(edges.emitted))
	}
	if edges.emitted[0].ActorClass != ActorClassAlgorithmic {
		t.Errorf("edge ActorClass = %v, want algorithmic",
			edges.emitted[0].ActorClass)
	}
}
