package recommendation

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
	rec  *models.Recommendation
	last struct {
		newState  string
		reviewDue *time.Time
		updateErr error
	}
}

func (f *fakeStore) Create(_ context.Context, r *models.Recommendation) error {
	f.rec = r
	return nil
}
func (f *fakeStore) Get(_ context.Context, _ uuid.UUID) (*models.Recommendation, error) {
	return f.rec, nil
}
func (f *fakeStore) UpdateState(_ context.Context, _ uuid.UUID, newState string,
	reviewDue *time.Time) error {
	f.last.newState = newState
	f.last.reviewDue = reviewDue
	if f.rec != nil {
		f.rec.State = newState
		f.rec.ReviewDueAt = reviewDue
	}
	return f.last.updateErr
}
func (f *fakeStore) ListDeferredOverdue(_ context.Context, _ time.Time) (
	[]models.Recommendation, error) {
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
	store := &fakeStore{rec: &models.Recommendation{
		ID:    uuid.New(),
		State: models.RecommendationStateDrafted,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges, AlwaysPassConsentChecker{})

	err := lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateSubmitted,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
		ReasoningSummary: "pharmacist completed draft",
	})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if store.last.newState != models.RecommendationStateSubmitted {
		t.Errorf("state = %q want submitted", store.last.newState)
	}
	if len(edges.emitted) != 1 {
		t.Fatalf("expected 1 EvidenceTrace edge; got %d", len(edges.emitted))
	}
	got := edges.emitted[0]
	if got.FromState != models.RecommendationStateDrafted ||
		got.ToState != models.RecommendationStateSubmitted {
		t.Errorf("edge states wrong: %+v", got)
	}
	if got.ActorClass != ActorClassHuman {
		t.Errorf("actor class wrong: %v", got.ActorClass)
	}
}

func TestLifecycle_TransitionForbidden(t *testing.T) {
	store := &fakeStore{rec: &models.Recommendation{
		ID:    uuid.New(),
		State: models.RecommendationStateDrafted,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges, AlwaysPassConsentChecker{})

	err := lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateImplemented, // skips submitted/viewed/decided
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
	})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition; got %v", err)
	}
	if len(edges.emitted) != 0 {
		t.Errorf("expected no edges emitted on rejected transition; got %d",
			len(edges.emitted))
	}
}

func TestLifecycle_DeferredRequiresReviewDue(t *testing.T) {
	store := &fakeStore{rec: &models.Recommendation{
		ID:    uuid.New(),
		State: models.RecommendationStateSubmitted,
	}}
	lc := NewLifecycle(store, &fakeEdgeStore{}, AlwaysPassConsentChecker{})

	err := lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateDeferred,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
		// ReviewDueAt deliberately omitted
	})
	if !errors.Is(err, ErrReviewDueRequired) {
		t.Fatalf("expected ErrReviewDueRequired; got %v", err)
	}

	due := time.Now().Add(72 * time.Hour)
	err = lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateDeferred,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
		ReviewDueAt:      &due,
	})
	if err != nil {
		t.Fatalf("expected success with ReviewDueAt; got %v", err)
	}
	if store.last.reviewDue == nil || !store.last.reviewDue.Equal(due) {
		t.Errorf("reviewDue not propagated: %v", store.last.reviewDue)
	}
}

// TestLifecycle_DecidedAtPopulated_RIRInvariant verifies the RIR-supporting
// invariant flagged by the Task 3 review: when a recommendation transitions
// through 'decided' on its way to 'implemented', decided_at MUST be set so
// the matview's COALESCE(decided_at, closed_at) doesn't fall through to NULL.
//
// This is an integration test against real Postgres.
func TestLifecycle_DecidedAtPopulated_RIRInvariant(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges, AlwaysPassConsentChecker{})
	ctx := context.Background()

	rec := models.Recommendation{
		ID: uuid.New(), ResidentID: uuid.New(), AuthorID: uuid.New(),
		State:           models.RecommendationStateDrafted,
		Type:            models.RecommendationTypeStop,
		Urgency:         models.RecommendationUrgencyAmber,
		Title:           "RIR-invariant test",
		ClinicalContent: models.ClinicalContent{Issue: "test"},
		CreatedAt:       time.Now().UTC(), UpdatedAt: time.Now().UTC(),
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
	}
	for _, s := range steps {
		err := lc.Transition(ctx, TransitionRequest{
			RecommendationID: rec.ID,
			ToState:          s,
			ActorID:          uuid.New(),
			ActorClass:       ActorClassHuman,
		})
		if err != nil {
			t.Fatalf("transition to %s: %v", s, err)
		}
	}

	got, err := store.Get(ctx, rec.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.DecidedAt == nil {
		t.Errorf("RIR invariant violated: decided_at must be set after passing through decided state")
	}
	if got.SubmittedAt == nil {
		t.Errorf("submitted_at must be set after passing through submitted state")
	}
	if got.State != models.RecommendationStateImplemented {
		t.Errorf("final state = %q want implemented", got.State)
	}
}

// TestLifecycle_ConsentRequiredBlocksSubmit exercises the v2 §3 line 140
// consent gate: drafted → submitted is blocked when ConsentRequired=true
// and the ConsentChecker reports no active consent.
func TestLifecycle_ConsentRequiredBlocksSubmit(t *testing.T) {
	store := &fakeStore{rec: &models.Recommendation{
		ID:              uuid.New(),
		ResidentID:      uuid.New(),
		State:           models.RecommendationStateDrafted,
		Type:            models.RecommendationTypeStop,
		ConsentRequired: true,
	}}
	checker := &fakeConsentChecker{active: false}
	lc := NewLifecycle(store, &fakeEdgeStore{}, checker)

	err := lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateSubmitted,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
	})
	if !errors.Is(err, ErrConsentRequired) {
		t.Fatalf("expected ErrConsentRequired; got %v", err)
	}

	checker.active = true
	err = lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateSubmitted,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
	})
	if err != nil {
		t.Fatalf("expected success once consent active; got %v", err)
	}
}

type fakeConsentChecker struct{ active bool }

func (f *fakeConsentChecker) ConsentActive(_ context.Context, _ uuid.UUID,
	_ string) (bool, error) {
	return f.active, nil
}
