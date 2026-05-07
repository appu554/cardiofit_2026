package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// fakeStore implements Store for unit tests, with no DB.
type fakeStore struct {
	rec *models.MonitoringPlan
}

func (f *fakeStore) Create(_ context.Context, p *models.MonitoringPlan) error {
	f.rec = p
	return nil
}
func (f *fakeStore) Get(_ context.Context, _ uuid.UUID) (*models.MonitoringPlan, error) {
	return f.rec, nil
}
func (f *fakeStore) UpdateState(_ context.Context, _ uuid.UUID, newState string) error {
	if f.rec != nil {
		f.rec.State = newState
	}
	return nil
}
func (f *fakeStore) MarkObligationFulfilled(_ context.Context, _ uuid.UUID,
	_ int, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (f *fakeStore) MarkThresholdCrossed(_ context.Context, _ uuid.UUID,
	_ int, _ time.Time) error {
	return nil
}
func (f *fakeStore) ListActiveOverdue(_ context.Context, _ time.Time) ([]models.MonitoringPlan, error) {
	return nil, nil
}
func (f *fakeStore) ListByRecommendation(_ context.Context, _ uuid.UUID) ([]models.MonitoringPlan, error) {
	return nil, nil
}

// fakeEdgeStore captures EmitEdge calls for assertion.
type fakeEdgeStore struct {
	emitted []EvidenceEdge
}

func (f *fakeEdgeStore) EmitEdge(_ context.Context, e EvidenceEdge) error {
	f.emitted = append(f.emitted, e)
	return nil
}

func TestLifecycle_TransitionHappyPath(t *testing.T) {
	store := &fakeStore{rec: &models.MonitoringPlan{
		ID:    uuid.New(),
		State: models.MonitoringPlanStatePending,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)

	err := lc.Transition(context.Background(), TransitionRequest{
		PlanID:     store.rec.ID,
		ToState:    models.MonitoringPlanStateActive,
		ActorID:    uuid.New(),
		ActorClass: ActorClassHuman,
	})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if len(edges.emitted) != 1 {
		t.Fatalf("expected 1 edge; got %d", len(edges.emitted))
	}
	got := edges.emitted[0]
	if got.FromState != models.MonitoringPlanStatePending ||
		got.ToState != models.MonitoringPlanStateActive {
		t.Errorf("edge states wrong: %+v", got)
	}
	if got.ActorClass != ActorClassHuman {
		t.Errorf("actor class wrong: %v", got.ActorClass)
	}
}

func TestLifecycle_TransitionForbidden(t *testing.T) {
	store := &fakeStore{rec: &models.MonitoringPlan{
		ID:    uuid.New(),
		State: models.MonitoringPlanStateCompleted, // terminal
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)

	err := lc.Transition(context.Background(), TransitionRequest{
		PlanID:     store.rec.ID,
		ToState:    models.MonitoringPlanStateActive,
		ActorID:    uuid.New(),
		ActorClass: ActorClassHuman,
	})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition; got %v", err)
	}
	if len(edges.emitted) != 0 {
		t.Errorf("no edges should be emitted on rejected transition; got %d",
			len(edges.emitted))
	}
}

func TestLifecycle_OccurredAtDefaultsToNow(t *testing.T) {
	store := &fakeStore{rec: &models.MonitoringPlan{
		ID:    uuid.New(),
		State: models.MonitoringPlanStatePending,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	frozen := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	lc.now = func() time.Time { return frozen }

	err := lc.Transition(context.Background(), TransitionRequest{
		PlanID:     store.rec.ID,
		ToState:    models.MonitoringPlanStateActive,
		ActorID:    uuid.New(),
		ActorClass: ActorClassAlgorithmic,
		// OccurredAt deliberately zero
	})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if !edges.emitted[0].OccurredAt.Equal(frozen) {
		t.Errorf("OccurredAt = %v, want frozen %v",
			edges.emitted[0].OccurredAt, frozen)
	}
}

func TestLifecycle_AlgorithmicEscalationCarriesActorClass(t *testing.T) {
	// Verifies that an escalator-driven transition (algorithmic actor)
	// records ActorClassAlgorithmic in the EvidenceEdge — supports the
	// v3 §9 Principle 4 audit guarantee for monitoring plans.
	store := &fakeStore{rec: &models.MonitoringPlan{
		ID:    uuid.New(),
		State: models.MonitoringPlanStateActive,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)

	err := lc.Transition(context.Background(), TransitionRequest{
		PlanID:     store.rec.ID,
		ToState:    models.MonitoringPlanStateEscalated,
		ActorID:    uuid.Nil, // system actor
		ActorClass: ActorClassAlgorithmic,
		Notes:      "threshold crossed",
	})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	got := edges.emitted[0]
	if got.ActorClass != ActorClassAlgorithmic {
		t.Errorf("ActorClass = %v want algorithmic", got.ActorClass)
	}
	if got.ActorID != uuid.Nil {
		t.Errorf("ActorID should be uuid.Nil for system actors; got %v", got.ActorID)
	}
	if got.Notes != "threshold crossed" {
		t.Errorf("Notes not propagated: %q", got.Notes)
	}
}

func TestLifecycle_InvalidActorClassRejected(t *testing.T) {
	store := &fakeStore{rec: &models.MonitoringPlan{
		ID:    uuid.New(),
		State: models.MonitoringPlanStatePending,
	}}
	lc := NewLifecycle(store, &fakeEdgeStore{})

	err := lc.Transition(context.Background(), TransitionRequest{
		PlanID:     store.rec.ID,
		ToState:    models.MonitoringPlanStateActive,
		ActorID:    uuid.New(),
		ActorClass: "bogus_class",
	})
	if err == nil || !contains(err.Error(), "actor class") {
		t.Errorf("expected actor class error; got %v", err)
	}
}

// contains is a tiny helper to avoid importing "strings" just for one call.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsMid(s, substr))))
}
func containsMid(s, substr string) bool {
	for i := 1; i+len(substr) < len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
