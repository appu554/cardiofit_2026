package monitoring

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ErrInvalidTransition is returned when the monitoring lifecycle DAG
// forbids the requested transition.
var ErrInvalidTransition = errors.New("invalid monitoring transition")

// ActorClass distinguishes algorithmic from human actors in the
// EvidenceTrace, satisfying v3 §9 Principle 4. Monitoring transitions
// are frequently algorithmic (escalator, threshold evaluator), so the
// algorithmic-vs-human distinction is more important here than a
// free-form role string would be.
type ActorClass string

const (
	ActorClassHuman                      ActorClass = "human"
	ActorClassAlgorithmic                ActorClass = "algorithmic"
	ActorClassHumanWithAlgorithmic       ActorClass = "human-with-algorithmic-suggestion"
	ActorClassHumanOverridingAlgorithmic ActorClass = "human-overriding-algorithmic"
)

// IsValidActorClass reports whether s is a valid monitoring lifecycle
// actor class.
func IsValidActorClass(s string) bool {
	switch ActorClass(s) {
	case ActorClassHuman, ActorClassAlgorithmic,
		ActorClassHumanWithAlgorithmic, ActorClassHumanOverridingAlgorithmic:
		return true
	}
	return false
}

// EvidenceEdge is the substrate-local record of one monitoring lifecycle
// transition. EdgeStore.EmitEdge persists this; the actual node-writing
// adapter (parallel to Plan 0.1 Task 6 EvidenceTraceAdapter) is deferred
// to a follow-up — when the Monitoring surface layer needs longitudinal
// audit, an adapter translates this into a models.EvidenceTraceNode.
type EvidenceEdge struct {
	PlanID     uuid.UUID
	FromState  string
	ToState    string
	ActorID    uuid.UUID
	ActorClass ActorClass
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
	PlanID     uuid.UUID
	ToState    string
	ActorID    uuid.UUID
	ActorClass ActorClass
	OccurredAt time.Time // optional; defaults to time.Now() UTC
	Notes      string
}

// Lifecycle is the only sanctioned mutator of monitoring plan state.
// Direct callers of Store.UpdateState bypass the transition matrix; this
// engine owns the layered-trust contract.
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

// Transition advances a monitoring plan through one DAG edge. Returns
// ErrInvalidTransition (wrapped with state context) on guard violation;
// otherwise returns the underlying store/edge error verbatim.
//
// Order: load → capture fromState → validate matrix → validate actor class
// → default OccurredAt → UpdateState → emit edge. State commits BEFORE
// emit; an emit failure surfaces but does NOT roll back state.
func (l *Lifecycle) Transition(ctx context.Context, r TransitionRequest) error {
	p, err := l.store.Get(ctx, r.PlanID)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}

	// Capture from-state BEFORE UpdateState in case the Store's Get returns
	// a pointer that the UpdateState path mutates in place. (Plan 0.1 Task 5
	// found this trap with the recommendation fakeStore.)
	fromState := p.State

	if !models.IsValidMonitoringTransition(fromState, r.ToState) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, fromState, r.ToState)
	}

	if !IsValidActorClass(string(r.ActorClass)) {
		return fmt.Errorf("invalid actor class: %q", r.ActorClass)
	}

	occurred := r.OccurredAt
	if occurred.IsZero() {
		occurred = l.now()
	}

	if err := l.store.UpdateState(ctx, p.ID, r.ToState); err != nil {
		return fmt.Errorf("update state: %w", err)
	}

	edge := EvidenceEdge{
		PlanID:     p.ID,
		FromState:  fromState,
		ToState:    r.ToState,
		ActorID:    r.ActorID,
		ActorClass: r.ActorClass,
		OccurredAt: occurred,
		Notes:      r.Notes,
	}
	if err := l.edges.EmitEdge(ctx, edge); err != nil {
		return fmt.Errorf("emit evidence edge: %w", err)
	}
	return nil
}
