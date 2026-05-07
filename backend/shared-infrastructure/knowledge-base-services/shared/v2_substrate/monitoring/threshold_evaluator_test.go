package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// fakeEventEmitter captures Events the evaluator would persist.
type fakeEventEmitter struct{ events []models.Event }

func (f *fakeEventEmitter) Emit(_ context.Context, e models.Event) error {
	f.events = append(f.events, e)
	return nil
}

func TestCrossThreshold(t *testing.T) {
	cases := []struct {
		spec  string
		value float64
		want  bool
	}{
		{"value > 5.5", 5.8, true},
		{"value > 5.5", 5.5, false},
		{"value < 3.0", 2.9, true},
		{"value < 3.0", 3.0, false},
		{"value > 5.5 OR value < 3.0", 2.5, true},
		{"value > 5.5 OR value < 3.0", 4.0, false},
		{"value >= 100", 100, true},
		{"value <= 100", 100, true},
		{"value == 5.5", 5.5, true},
		{"value == 5.5", 5.6, false},
		{"bogus spec", 1, false},
		{"", 1, false},
	}
	for _, c := range cases {
		if got := crossThreshold(c.spec, c.value); got != c.want {
			t.Errorf("crossThreshold(%q, %v) = %v want %v",
				c.spec, c.value, got, c.want)
		}
	}
}

func TestThresholdEvaluator_CrossingEmitsEventAndEscalates(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	emitter := &fakeEventEmitter{}
	evaluator := NewThresholdEvaluator(store, lc, emitter)
	ctx := context.Background()

	resident := uuid.New()
	due := time.Now().Add(2 * time.Hour).UTC()
	plan := models.MonitoringPlan{
		ID:               uuid.New(),
		RecommendationID: uuid.New(),
		ResidentID:       resident,
		State:            models.MonitoringPlanStateActive,
		Obligations: []models.MonitoringObligation{
			{
				Type:            models.MonitoringObligationTypeObservation,
				ObservationCode: "potassium",
				FrequencyHours:  24,
				DueAt:           due,
				ThresholdSpec:   "value > 5.5",
			},
		},
		StartedAt:           time.Now().UTC(),
		ExpectedEndAt:       time.Now().Add(7 * 24 * time.Hour).UTC(),
		EscalateAfterMissed: 2,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}
	if err := store.Create(ctx, &plan); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM monitoring_plans WHERE id = $1", plan.ID)
	})

	value := 5.8
	obs := models.Observation{
		ID:         uuid.New(),
		ResidentID: resident,
		LOINCCode:  "potassium", // matches obligation.ObservationCode
		Value:      &value,
		ObservedAt: time.Now().UTC(),
	}
	if err := evaluator.Evaluate(ctx, obs); err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	// Plan should be escalated
	got, err := store.Get(ctx, plan.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.State != models.MonitoringPlanStateEscalated {
		t.Errorf("expected escalated; got %q", got.State)
	}

	// Obligation should be marked fulfilled and threshold-crossed
	if got.Obligations[0].FulfilledAt == nil {
		t.Errorf("obligation FulfilledAt nil after threshold-crossing observation")
	}
	if got.Obligations[0].ThresholdCrossedAt == nil {
		t.Errorf("obligation ThresholdCrossedAt nil after crossing")
	}

	// One Event should be emitted
	if len(emitter.events) != 1 {
		t.Fatalf("expected 1 event; got %d", len(emitter.events))
	}
	ev := emitter.events[0]
	if ev.EventType != models.EventTypeMonitoringThresholdCrossed {
		t.Errorf("event type = %q want monitoring_threshold_crossed", ev.EventType)
	}
	if ev.ResidentID != resident {
		t.Errorf("event ResidentID wrong: %v", ev.ResidentID)
	}

	// One EvidenceEdge should be emitted by the lifecycle (active -> escalated)
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

func TestThresholdEvaluator_NonMatchingObservationIgnored(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	emitter := &fakeEventEmitter{}
	evaluator := NewThresholdEvaluator(store, lc, emitter)
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
				FrequencyHours:  24,
				DueAt:           time.Now().Add(2 * time.Hour).UTC(),
				ThresholdSpec:   "value > 5.5",
			},
		},
		StartedAt:           time.Now().UTC(),
		ExpectedEndAt:       time.Now().Add(7 * 24 * time.Hour).UTC(),
		EscalateAfterMissed: 2,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}
	if err := store.Create(ctx, &plan); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM monitoring_plans WHERE id = $1", plan.ID)
	})

	value := 7.0
	obs := models.Observation{
		ID:         uuid.New(),
		ResidentID: resident,
		LOINCCode:  "sodium", // does NOT match obligation.ObservationCode "potassium"
		Value:      &value,
		ObservedAt: time.Now().UTC(),
	}
	if err := evaluator.Evaluate(ctx, obs); err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	got, _ := store.Get(ctx, plan.ID)
	if got.State != models.MonitoringPlanStateActive {
		t.Errorf("plan state should remain active; got %q", got.State)
	}
	if got.Obligations[0].FulfilledAt != nil {
		t.Errorf("obligation should remain unfulfilled when observation code doesn't match")
	}
	if len(emitter.events) != 0 {
		t.Errorf("no events should be emitted on non-matching observation; got %d",
			len(emitter.events))
	}
}

func TestThresholdEvaluator_ObservationBelowThresholdMarksFulfilledOnly(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	emitter := &fakeEventEmitter{}
	evaluator := NewThresholdEvaluator(store, lc, emitter)
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
				FrequencyHours:  24,
				DueAt:           time.Now().Add(2 * time.Hour).UTC(),
				ThresholdSpec:   "value > 5.5",
			},
		},
		StartedAt:           time.Now().UTC(),
		ExpectedEndAt:       time.Now().Add(7 * 24 * time.Hour).UTC(),
		EscalateAfterMissed: 2,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}
	if err := store.Create(ctx, &plan); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM monitoring_plans WHERE id = $1", plan.ID)
	})

	value := 4.5
	obs := models.Observation{
		ID:         uuid.New(),
		ResidentID: resident,
		LOINCCode:  "potassium",
		Value:      &value, // below threshold (5.5)
		ObservedAt: time.Now().UTC(),
	}
	if err := evaluator.Evaluate(ctx, obs); err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	got, _ := store.Get(ctx, plan.ID)
	// Plan stays active (no escalation), but obligation IS marked fulfilled
	if got.State != models.MonitoringPlanStateActive {
		t.Errorf("plan should remain active when observation is in-range; got %q", got.State)
	}
	if got.Obligations[0].FulfilledAt == nil {
		t.Errorf("obligation should be marked fulfilled even when threshold not crossed")
	}
	if got.Obligations[0].ThresholdCrossedAt != nil {
		t.Errorf("threshold should NOT be marked crossed for in-range value")
	}
	if len(emitter.events) != 0 {
		t.Errorf("no event should be emitted for in-range observation; got %d", len(emitter.events))
	}
}

func TestThresholdEvaluator_NilObservationValueIgnored(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	emitter := &fakeEventEmitter{}
	evaluator := NewThresholdEvaluator(store, lc, emitter)
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
				DueAt:           time.Now().Add(2 * time.Hour).UTC(),
				ThresholdSpec:   "value > 5.5",
			},
		},
		StartedAt:           time.Now().UTC(),
		ExpectedEndAt:       time.Now().Add(7 * 24 * time.Hour).UTC(),
		EscalateAfterMissed: 2,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}
	if err := store.Create(ctx, &plan); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM monitoring_plans WHERE id = $1", plan.ID)
	})

	// Textual observation: Value is nil, only ValueText set
	obs := models.Observation{
		ID:         uuid.New(),
		ResidentID: resident,
		LOINCCode:  "potassium",
		Value:      nil,
		ValueText:  "haemolysed sample — unmeasurable",
		ObservedAt: time.Now().UTC(),
	}
	if err := evaluator.Evaluate(ctx, obs); err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	got, _ := store.Get(ctx, plan.ID)
	if got.State != models.MonitoringPlanStateActive {
		t.Errorf("nil-value observation should not change plan state; got %q", got.State)
	}
	if got.Obligations[0].FulfilledAt != nil {
		t.Errorf("nil-value observation should not mark obligation fulfilled")
	}
	if len(emitter.events) != 0 {
		t.Errorf("no event for nil-value observation; got %d", len(emitter.events))
	}
}
