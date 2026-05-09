// Package incident_response — see classifier.go for package-level documentation.
package incident_response

import (
	"context"

	"github.com/google/uuid"
)

// Incident describes a confirmed ethical incident that requires corrective
// action. It is passed to every HoldHandler registered on an Orchestrator.
//
// VisibilityClass: AD
type Incident struct {
	// ID is the unique identifier for this incident. If zero when passed to
	// Orchestrator.Trigger, it is not modified — callers may generate their own.
	ID uuid.UUID

	// Severity is the numerical severity level (1..4) derived via Classify.
	// Severity 1 is most severe (clinical_safety), 4 is least (procedural).
	Severity int

	// Kind is the incident kind string, one of the four canonical values
	// recognised by Classify.
	Kind string

	// AffectedComponents lists the service/sub-system identifiers that are
	// involved in the incident.
	AffectedComponents []string

	// HoldActive records whether an active hold has been placed on the affected
	// components as a result of this incident.
	HoldActive bool

	// Description is a free-form human-readable account of the incident.
	Description string
}

// HoldHandler is the interface corrective subsystems implement to act when an
// incident hold is triggered. Implementations MUST be idempotent where possible.
type HoldHandler interface {
	OnHold(ctx context.Context, inc Incident) error
}

// HoldHandlerFunc is a function adapter for HoldHandler.
type HoldHandlerFunc func(ctx context.Context, inc Incident) error

// OnHold implements HoldHandler.
func (f HoldHandlerFunc) OnHold(ctx context.Context, inc Incident) error {
	return f(ctx, inc)
}

// Orchestrator routes Incidents to registered HoldHandlers. A hold is triggered
// only when the incident Severity is ≤ 2, corresponding to "clinical_safety"
// (sev 1) and "trust_violation" (sev 2) — both of which require immediate
// corrective action per Ethical Architecture Guidelines §11.2.
//
// Handlers are invoked in registration order. If any handler returns an error,
// Trigger returns that error immediately without calling subsequent handlers.
//
// Orchestrator is not goroutine-safe; callers must synchronise externally if
// concurrent use is required.
type Orchestrator struct {
	handlers []HoldHandler
}

// NewOrchestrator returns an Orchestrator with an empty handler list.
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{}
}

// Register appends h to the handler list. h is called in the order registered.
func (o *Orchestrator) Register(h HoldHandler) {
	o.handlers = append(o.handlers, h)
}

// Trigger evaluates inc and, when inc.Severity ≤ 2, calls every registered
// HoldHandler in order. The first handler error is returned immediately.
// When inc.Severity > 2, Trigger is a no-op and returns nil.
func (o *Orchestrator) Trigger(ctx context.Context, inc Incident) error {
	if inc.Severity > 2 {
		return nil
	}
	for _, h := range o.handlers {
		if err := h.OnHold(ctx, inc); err != nil {
			return err
		}
	}
	return nil
}
