package consent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ErrInvalidTransition is returned when the consent lifecycle DAG forbids
// the requested transition. The HTTP layer maps this to 400 / 409; the
// engine never returns raw SQL errors for guard violations.
var ErrInvalidTransition = errors.New("invalid consent transition")

// EvidenceEdge is the substrate-local record of one consent lifecycle
// transition. EdgeStore.EmitEdge persists this; the actual node-writing
// adapter (parallel to Plan 0.1 Task 6 EvidenceTraceAdapter for
// Recommendations) is deferred to a follow-up — when the Consent
// surface layer needs longitudinal audit, an adapter translates this
// into a models.EvidenceTraceNode and upserts via NodeWriter.
type EvidenceEdge struct {
	ConsentID  uuid.UUID
	FromState  string
	ToState    string
	ActorID    uuid.UUID
	ActorRole  string
	OccurredAt time.Time
	Notes      string
}

// EdgeStore is the EvidenceTrace persistence boundary. Real implementation
// (follow-up Adapter) wraps the substrate's evidence_trace package; tests
// use a fake.
type EdgeStore interface {
	EmitEdge(ctx context.Context, e EvidenceEdge) error
}

// TransitionRequest is the input contract for Lifecycle.Transition.
type TransitionRequest struct {
	ConsentID  uuid.UUID
	ToState    string
	ActorID    uuid.UUID
	ActorRole  string    // role at the time of action (e.g. "substitute_decision_maker")
	OccurredAt time.Time // optional; defaults to time.Now() UTC
	Notes      string
}

// Lifecycle is the only sanctioned mutator of consent state. Direct
// callers of Store.UpdateState bypass the transition matrix; this engine
// owns the layered-trust contract.
type Lifecycle struct {
	store Store
	edges EdgeStore
	now   func() time.Time
}

// NewLifecycle constructs a Lifecycle wired to the supplied collaborators.
func NewLifecycle(store Store, edges EdgeStore) *Lifecycle {
	return &Lifecycle{
		store: store, edges: edges,
		now: func() time.Time { return time.Now().UTC() },
	}
}

// Transition advances a consent through one DAG edge. Returns
// ErrInvalidTransition (wrapped with state context) on guard violation;
// otherwise returns the underlying store/edge error verbatim.
//
// Order: load → validate matrix → update store → emit edge. State change
// is committed BEFORE emit; an emit failure surfaces but does NOT roll
// back state.
func (l *Lifecycle) Transition(ctx context.Context, r TransitionRequest) error {
	c, err := l.store.Get(ctx, r.ConsentID)
	if err != nil {
		return fmt.Errorf("load consent: %w", err)
	}

	// Capture from-state BEFORE UpdateState in case the Store's Get returns
	// a pointer that the UpdateState path mutates in place. (Plan 0.1 Task 5
	// found this trap with the recommendation fakeStore.)
	fromState := c.State

	if !models.IsValidConsentTransition(fromState, r.ToState) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, fromState, r.ToState)
	}

	occurred := r.OccurredAt
	if occurred.IsZero() {
		occurred = l.now()
	}

	if err := l.store.UpdateState(ctx, c.ID, r.ToState); err != nil {
		return fmt.Errorf("update state: %w", err)
	}

	edge := EvidenceEdge{
		ConsentID:  c.ID,
		FromState:  fromState,
		ToState:    r.ToState,
		ActorID:    r.ActorID,
		ActorRole:  r.ActorRole,
		OccurredAt: occurred,
		Notes:      r.Notes,
	}
	if err := l.edges.EmitEdge(ctx, edge); err != nil {
		return fmt.Errorf("emit evidence edge: %w", err)
	}
	return nil
}
