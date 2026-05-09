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
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		State:      models.RecommendationStateDrafted,
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
	if got.ResidentID != store.rec.ResidentID {
		t.Errorf("ResidentID not propagated to edge: got %v want %v",
			got.ResidentID, store.rec.ResidentID)
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

// TestLifecycle_ConsentGateScopedToSubmit verifies the consent gate fires
// ONLY for drafted→submitted transitions. Other transitions (like
// drafted→closed) must not invoke the ConsentChecker even when
// rec.ConsentRequired=true. Pins the lifecycle.go guard predicate against
// future refactors that might inadvertently broaden the gate.
func TestLifecycle_ConsentGateScopedToSubmit(t *testing.T) {
	store := &fakeStore{rec: &models.Recommendation{
		ID:              uuid.New(),
		ResidentID:      uuid.New(),
		State:           models.RecommendationStateDrafted,
		Type:            models.RecommendationTypeStop,
		ConsentRequired: true,
	}}
	checker := &consentCheckCounter{} // tracks invocations
	lc := NewLifecycle(store, &fakeEdgeStore{}, checker)

	// drafted → closed should NOT invoke the consent checker even with
	// ConsentRequired=true, because the gate is scoped to drafted→submitted.
	err := lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateClosed,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
	})
	if err != nil {
		t.Fatalf("drafted->closed should succeed regardless of consent; got %v", err)
	}
	if checker.calls != 0 {
		t.Errorf("consent checker invoked on non-submit transition: %d calls", checker.calls)
	}
}

// consentCheckCounter is a ConsentChecker that counts invocations. Returns
// true (active) so it never blocks; the test asserts on the call count.
type consentCheckCounter struct{ calls int }

func (c *consentCheckCounter) ConsentActive(_ context.Context, _ uuid.UUID,
	_ string) (bool, error) {
	c.calls++
	return true, nil
}

// ---------------------------------------------------------------------------
// CraftEngineGate tests (Task 13)
// ---------------------------------------------------------------------------

// fakeCraftGate is a test double for the optional CraftEngineGate.
// passThrough=true → nil error (allow transition).
// passThrough=false → sentinelGateErr (block transition).
type fakeCraftGate struct {
	passThrough bool
	calls       int
}

var sentinelGateErr = errors.New("craft gate: held")

func (g *fakeCraftGate) AdvanceDetectedToDrafted(_ context.Context, _ uuid.UUID) error {
	g.calls++
	if g.passThrough {
		return nil
	}
	return sentinelGateErr
}

// TestLifecycle_CraftEngineGate_AllowsDetectedToDrafted verifies that when
// the gate returns nil, the detected → drafted transition succeeds and an
// EvidenceTrace edge is emitted.
func TestLifecycle_CraftEngineGate_AllowsDetectedToDrafted(t *testing.T) {
	store := &fakeStore{rec: &models.Recommendation{
		ID:    uuid.New(),
		State: models.RecommendationStateDetected,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges, AlwaysPassConsentChecker{})

	gate := &fakeCraftGate{passThrough: true}
	lc.SetCraftEngineGate(gate)

	err := lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateDrafted,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassAlgorithmic,
	})
	if err != nil {
		t.Fatalf("expected transition to succeed; got %v", err)
	}
	if gate.calls != 1 {
		t.Errorf("gate invoked %d times; want 1", gate.calls)
	}
	if len(edges.emitted) != 1 {
		t.Fatalf("want 1 evidence edge; got %d", len(edges.emitted))
	}
}

// TestLifecycle_CraftEngineGate_BlocksDetectedToDrafted verifies that when
// the gate returns an error, the detected → drafted transition is aborted and
// no state change is committed.
func TestLifecycle_CraftEngineGate_BlocksDetectedToDrafted(t *testing.T) {
	store := &fakeStore{rec: &models.Recommendation{
		ID:    uuid.New(),
		State: models.RecommendationStateDetected,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges, AlwaysPassConsentChecker{})

	gate := &fakeCraftGate{passThrough: false}
	lc.SetCraftEngineGate(gate)

	err := lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateDrafted,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassAlgorithmic,
	})
	if err == nil {
		t.Fatal("expected transition to fail (gate holds); got nil")
	}
	if !errors.Is(err, sentinelGateErr) {
		t.Errorf("expected sentinelGateErr in error chain; got %v", err)
	}
	if gate.calls != 1 {
		t.Errorf("gate invoked %d times; want 1", gate.calls)
	}
	// State must not have changed.
	if store.last.newState != "" {
		t.Errorf("state was mutated despite gate hold: %q", store.last.newState)
	}
	if len(edges.emitted) != 0 {
		t.Errorf("no evidence edge should be emitted on blocked transition; got %d", len(edges.emitted))
	}
}

// TestLifecycle_CraftEngineGate_NotInvokedForOtherTransitions verifies that
// the CraftEngineGate is ONLY called for the detected → drafted edge. All
// other transitions must bypass the gate even when one is configured.
func TestLifecycle_CraftEngineGate_NotInvokedForOtherTransitions(t *testing.T) {
	// drafted → submitted: gate must NOT be invoked.
	store := &fakeStore{rec: &models.Recommendation{
		ID:    uuid.New(),
		State: models.RecommendationStateDrafted,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges, AlwaysPassConsentChecker{})

	gate := &fakeCraftGate{passThrough: false} // would block if called
	lc.SetCraftEngineGate(gate)

	err := lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateSubmitted,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
	})
	if err != nil {
		t.Fatalf("drafted→submitted should succeed without gate; got %v", err)
	}
	if gate.calls != 0 {
		t.Errorf("gate should not be invoked for drafted→submitted; got %d calls", gate.calls)
	}
}

// TestLifecycle_NoCraftEngineGate_DetectedToDraftedSucceeds verifies that
// existing Lifecycle instances without a configured gate behave identically
// to pre-Task-13 behaviour (no gate invocation, transition succeeds).
func TestLifecycle_NoCraftEngineGate_DetectedToDraftedSucceeds(t *testing.T) {
	store := &fakeStore{rec: &models.Recommendation{
		ID:    uuid.New(),
		State: models.RecommendationStateDetected,
	}}
	edges := &fakeEdgeStore{}
	// No SetCraftEngineGate call — gate is nil.
	lc := NewLifecycle(store, edges, AlwaysPassConsentChecker{})

	err := lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateDrafted,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassAlgorithmic,
	})
	if err != nil {
		t.Fatalf("expected detected→drafted to succeed with no gate configured; got %v", err)
	}
	if len(edges.emitted) != 1 {
		t.Fatalf("want 1 evidence edge; got %d", len(edges.emitted))
	}
}

// TestLifecycle_OccurredAtDefaultsToNow verifies that an unset OccurredAt
// in the request is replaced with the lifecycle's `now` value before being
// recorded into the EvidenceEdge.
func TestLifecycle_OccurredAtDefaultsToNow(t *testing.T) {
	store := &fakeStore{rec: &models.Recommendation{
		ID:    uuid.New(),
		State: models.RecommendationStateDrafted,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges, AlwaysPassConsentChecker{})

	// Inject a deterministic now so we can assert exact value.
	frozen := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	lc.now = func() time.Time { return frozen }

	err := lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateSubmitted,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
		// OccurredAt deliberately zero
	})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if len(edges.emitted) != 1 {
		t.Fatalf("want 1 edge; got %d", len(edges.emitted))
	}
	if !edges.emitted[0].OccurredAt.Equal(frozen) {
		t.Errorf("OccurredAt = %v, want %v (frozen now)",
			edges.emitted[0].OccurredAt, frozen)
	}
}
