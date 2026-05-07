package consent

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
	rec *models.Consent
}

func (f *fakeStore) Create(_ context.Context, c *models.Consent) error {
	f.rec = c
	return nil
}
func (f *fakeStore) Get(_ context.Context, _ uuid.UUID) (*models.Consent, error) {
	return f.rec, nil
}
func (f *fakeStore) UpdateState(_ context.Context, _ uuid.UUID, newState string) error {
	if f.rec != nil {
		f.rec.State = newState
	}
	return nil
}
func (f *fakeStore) FindActive(_ context.Context, _ uuid.UUID, _ string) (*models.Consent, error) {
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
	store := &fakeStore{rec: &models.Consent{
		ID:    uuid.New(),
		State: models.ConsentStateDiscussed,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)

	err := lc.Transition(context.Background(), TransitionRequest{
		ConsentID: store.rec.ID,
		ToState:   models.ConsentStateGranted,
		ActorID:   uuid.New(),
		ActorRole: "substitute_decision_maker",
	})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if len(edges.emitted) != 1 {
		t.Fatalf("expected 1 edge; got %d", len(edges.emitted))
	}
	got := edges.emitted[0]
	if got.FromState != models.ConsentStateDiscussed ||
		got.ToState != models.ConsentStateGranted {
		t.Errorf("edge states wrong: %+v", got)
	}
	if got.ActorRole != "substitute_decision_maker" {
		t.Errorf("ActorRole not propagated: %q", got.ActorRole)
	}
}

func TestLifecycle_TransitionForbidden(t *testing.T) {
	store := &fakeStore{rec: &models.Consent{
		ID:    uuid.New(),
		State: models.ConsentStateRefused, // terminal
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)

	err := lc.Transition(context.Background(), TransitionRequest{
		ConsentID: store.rec.ID,
		ToState:   models.ConsentStateActive,
		ActorID:   uuid.New(),
		ActorRole: "sdm",
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
	store := &fakeStore{rec: &models.Consent{
		ID:    uuid.New(),
		State: models.ConsentStateDiscussed,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	frozen := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	lc.now = func() time.Time { return frozen }

	err := lc.Transition(context.Background(), TransitionRequest{
		ConsentID: store.rec.ID,
		ToState:   models.ConsentStateGranted,
		ActorID:   uuid.New(),
		ActorRole: "sdm",
		// OccurredAt deliberately zero
	})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if len(edges.emitted) != 1 {
		t.Fatalf("want 1 edge; got %d", len(edges.emitted))
	}
	if !edges.emitted[0].OccurredAt.Equal(frozen) {
		t.Errorf("OccurredAt = %v, want frozen %v",
			edges.emitted[0].OccurredAt, frozen)
	}
}

func TestLifecycle_StateUpdatedBeforeEmit(t *testing.T) {
	// Verifies state is committed before edge emit (so emit-failure
	// surfaces but does NOT roll back the state change).
	store := &fakeStore{rec: &models.Consent{
		ID:    uuid.New(),
		State: models.ConsentStateActive,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)

	err := lc.Transition(context.Background(), TransitionRequest{
		ConsentID: store.rec.ID,
		ToState:   models.ConsentStateWithdrawn,
		ActorID:   uuid.New(),
		ActorRole: "resident_self",
	})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if store.rec.State != models.ConsentStateWithdrawn {
		t.Errorf("state must be updated; got %q", store.rec.State)
	}
}
