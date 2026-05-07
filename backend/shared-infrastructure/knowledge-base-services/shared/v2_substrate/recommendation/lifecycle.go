package recommendation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// Sentinel errors. The HTTP layer maps these to status codes; the lifecycle
// engine never returns raw SQL errors for guard violations.
var (
	ErrInvalidTransition = errors.New("invalid recommendation transition")
	ErrReviewDueRequired = errors.New("review_due_at required for deferred state")
	ErrConsentRequired   = errors.New("recommendation requires active consent before submission")
)

// ActorClass distinguishes algorithmic from human actors in the
// EvidenceTrace, satisfying v3 §9 Principle 4.
type ActorClass string

const (
	ActorClassHuman                      ActorClass = "human"
	ActorClassAlgorithmic                ActorClass = "algorithmic"
	ActorClassHumanWithAlgorithmic       ActorClass = "human-with-algorithmic-suggestion"
	ActorClassHumanOverridingAlgorithmic ActorClass = "human-overriding-algorithmic"
)

// EvidenceEdge is the substrate-local record of one lifecycle transition.
// EdgeStore.EmitEdge persists this into the EvidenceTrace graph.
type EvidenceEdge struct {
	RecommendationID uuid.UUID
	FromState        string
	ToState          string
	ActorID          uuid.UUID
	ActorClass       ActorClass
	OccurredAt       time.Time
	ReasoningSummary string
	InputRefs        []uuid.UUID
}

// EdgeStore is the EvidenceTrace persistence boundary. Real implementation
// (Task 6 EvidenceTraceAdapter) wraps the substrate's evidence_trace
// package; tests use a fake.
type EdgeStore interface {
	EmitEdge(ctx context.Context, e EvidenceEdge) error
}

// ConsentChecker is the substrate gate ensuring restrictive-practice and
// other consent-required recommendation classes have an active matching
// Consent before they advance from drafted → submitted.
//
// Plan 0.2 ships a real Postgres-backed checker; tests use AlwaysPassConsentChecker
// or fakes.
type ConsentChecker interface {
	ConsentActive(ctx context.Context, residentID uuid.UUID, recType string) (bool, error)
}

// TransitionRequest is the input contract for Lifecycle.Transition.
type TransitionRequest struct {
	RecommendationID uuid.UUID
	ToState          string
	ActorID          uuid.UUID
	ActorClass       ActorClass
	OccurredAt       time.Time // optional; defaults to time.Now() UTC
	ReasoningSummary string
	InputRefs        []uuid.UUID
	ReviewDueAt      *time.Time // required when ToState == deferred
}

// Lifecycle is the only sanctioned mutator of recommendation state.
type Lifecycle struct {
	store   Store
	edges   EdgeStore
	consent ConsentChecker
	now     func() time.Time
}

// NewLifecycle constructs a Lifecycle wired to the supplied collaborators.
func NewLifecycle(store Store, edges EdgeStore, consent ConsentChecker) *Lifecycle {
	return &Lifecycle{
		store:   store,
		edges:   edges,
		consent: consent,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// Transition advances a recommendation through one DAG edge. Returns one of
// the sentinel errors on guard violation; otherwise returns the underlying
// store/edge error verbatim.
//
// Note: state change is committed BEFORE the EvidenceTrace edge emit. If
// emit fails, the error surfaces operationally but state is not rolled back —
// rollback would risk losing a legitimate transition merely because the
// trace store hiccupped. Emit failures are loud and surfaced to the caller.
func (l *Lifecycle) Transition(ctx context.Context, req TransitionRequest) error {
	rec, err := l.store.Get(ctx, req.RecommendationID)
	if err != nil {
		return fmt.Errorf("load recommendation: %w", err)
	}

	if !models.IsValidTransition(rec.State, req.ToState) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, rec.State, req.ToState)
	}

	if req.ToState == models.RecommendationStateDeferred && req.ReviewDueAt == nil {
		return ErrReviewDueRequired
	}

	if rec.State == models.RecommendationStateDrafted &&
		req.ToState == models.RecommendationStateSubmitted &&
		rec.ConsentRequired {
		ok, err := l.consent.ConsentActive(ctx, rec.ResidentID, rec.Type)
		if err != nil {
			return fmt.Errorf("consent check: %w", err)
		}
		if !ok {
			return ErrConsentRequired
		}
	}

	occurred := req.OccurredAt
	if occurred.IsZero() {
		occurred = l.now()
	}

	// Capture fromState BEFORE UpdateState — Get may return a pointer that
	// the store mutates in place (the test fake does), which would corrupt
	// the EvidenceTrace edge's FromState if we read rec.State after.
	fromState := rec.State

	if err := l.store.UpdateState(ctx, rec.ID, req.ToState, req.ReviewDueAt); err != nil {
		return fmt.Errorf("update state: %w", err)
	}

	edge := EvidenceEdge{
		RecommendationID: rec.ID,
		FromState:        fromState,
		ToState:          req.ToState,
		ActorID:          req.ActorID,
		ActorClass:       req.ActorClass,
		OccurredAt:       occurred,
		ReasoningSummary: req.ReasoningSummary,
		InputRefs:        req.InputRefs,
	}
	if err := l.edges.EmitEdge(ctx, edge); err != nil {
		return fmt.Errorf("emit evidence edge: %w", err)
	}
	return nil
}

// AlwaysPassConsentChecker is a test/dev double. Production deployments
// (Plan 0.4 kb-30 wiring) pass Plan 0.2's PostgresConsentChecker.
type AlwaysPassConsentChecker struct{}

func (AlwaysPassConsentChecker) ConsentActive(_ context.Context,
	_ uuid.UUID, _ string) (bool, error) {
	return true, nil
}
