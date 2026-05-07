package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestEscalator_EscalatesWhenMissedExceedsThreshold(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	emitter := &fakeEventEmitter{}
	frozen := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	graceWindow := 1 * time.Hour
	esc := NewEscalator(store, lc, emitter, graceWindow)
	esc.now = func() time.Time { return frozen }
	ctx := context.Background()

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

	got, err := store.Get(ctx, plan.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.State != models.MonitoringPlanStateEscalated {
		t.Errorf("expected escalated; got %q", got.State)
	}

	if len(emitter.events) != 1 {
		t.Fatalf("expected 1 event; got %d", len(emitter.events))
	}
	if emitter.events[0].EventType != models.EventTypeMonitoringObligationMissed {
		t.Errorf("event type = %q want monitoring_obligation_missed",
			emitter.events[0].EventType)
	}
	if emitter.events[0].ResidentID != resident {
		t.Errorf("event ResidentID wrong: %v", emitter.events[0].ResidentID)
	}

	if len(edges.emitted) != 1 {
		t.Fatalf("expected 1 evidence edge; got %d", len(edges.emitted))
	}
	if edges.emitted[0].ToState != models.MonitoringPlanStateEscalated {
		t.Errorf("edge ToState wrong: %q", edges.emitted[0].ToState)
	}
	if edges.emitted[0].ActorClass != ActorClassAlgorithmic {
		t.Errorf("edge ActorClass = %v want algorithmic", edges.emitted[0].ActorClass)
	}
}

func TestEscalator_BelowThresholdLeavesAlone(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	emitter := &fakeEventEmitter{}
	frozen := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	graceWindow := 1 * time.Hour
	esc := NewEscalator(store, lc, emitter, graceWindow)
	esc.now = func() time.Time { return frozen }
	ctx := context.Background()

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
				DueAt:           frozen.Add(+2 * time.Hour),
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
	if got.State != models.MonitoringPlanStateActive {
		t.Errorf("plan should remain active (1 missed < threshold 2); got %q", got.State)
	}
	if len(emitter.events) != 0 {
		t.Errorf("expected 0 events; got %d", len(emitter.events))
	}
}

func TestEscalator_FulfilledObligationsDoNotCount(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	emitter := &fakeEventEmitter{}
	frozen := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	graceWindow := 1 * time.Hour
	esc := NewEscalator(store, lc, emitter, graceWindow)
	esc.now = func() time.Time { return frozen }
	ctx := context.Background()

	resident := uuid.New()
	fulfilled := frozen.Add(-2 * time.Hour)
	obsID := uuid.New()
	plan := models.MonitoringPlan{
		ID:               uuid.New(),
		RecommendationID: uuid.New(),
		ResidentID:       resident,
		State:            models.MonitoringPlanStateActive,
		Obligations: []models.MonitoringObligation{
			{
				Type:             models.MonitoringObligationTypeObservation,
				ObservationCode:  "potassium",
				DueAt:            frozen.Add(-3 * time.Hour),
				ThresholdSpec:    "value > 5.5",
				FulfilledAt:      &fulfilled,
				FulfilledByObsID: &obsID,
			},
			{
				Type:            models.MonitoringObligationTypeObservation,
				ObservationCode: "creatinine",
				DueAt:           frozen.Add(-3 * time.Hour),
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
	if got.State != models.MonitoringPlanStateActive {
		t.Errorf("plan should remain active (only 1 missed; the other was fulfilled); got %q", got.State)
	}
}

func TestEscalator_NoOverduePlansNoOp(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	emitter := &fakeEventEmitter{}
	frozen := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	esc := NewEscalator(store, lc, emitter, 1*time.Hour)
	esc.now = func() time.Time { return frozen }
	ctx := context.Background()

	if err := esc.RunOnce(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(emitter.events) != 0 || len(edges.emitted) != 0 {
		t.Errorf("expected no work for empty active-overdue list")
	}
}
