// Package aggregation contains the s2-aggregator's layer-aware view-building
// types and the S2ViewBuilder interface.
//
// The five-layer cognitive escalation framework comes from the S2 Adaptive
// Cognition Architectural Commitment Addendum (Parts 3 and 8.1). Layer 1 is
// the immediate Phase 1 implementation target; Layers 2–5 contain content
// that is deferred to senior consultant pharmacist authoring per Addendum
// Part 6 ("Authoring discipline for deferred content layers").
//
// In Task 1 of the S2 Layer 1 build plan, the Layer*View types are scaffolded
// as empty structs. Task 3 onward populates Layer1View per S2 v1.0 Parts 4–13;
// Layers 2–5 stay empty per the Addendum's content-deferral discipline.
package aggregation

import (
	"time"

	"github.com/google/uuid"
)

// EntryPath identifies which of the four S2 entry paths (S2 v1.0 Part 3)
// surfaced this resident to the pharmacist. Task 2 of the build plan
// implements the entry-path handlers; for Task 1 it is a typed string slot
// on WorkspaceRequest so the field is named and stable.
type EntryPath string

const (
	EntryPathWorklist       EntryPath = "worklist"        // CAPE worklist entry (v1.0 §3.1)
	EntryPathSearch         EntryPath = "search"          // search entry (v1.0 §3.2)
	EntryPathNotification   EntryPath = "notification"    // notification entry (v1.0 §3.3)
	EntryPathCrossReference EntryPath = "cross_reference" // cross-reference entry (v1.0 §3.4)
)

// WorkspaceRequest is the per-call input to every S2ViewBuilder method.
// It carries enough identity to (a) look up the resident, (b) attribute
// pharmacist actions, and (c) cohere events into a single session for audit.
//
// AsOf supports time-travel queries; default is time.Now() at call site.
//
// EntryMetadata carries the entry-path context produced by the handlers
// in internal/entry_paths (Task 2 of the build plan). It is required for
// view assembly so that the CAPE context band, notification context band,
// and comparative mode can render per v1.0 Part 4.1. When EntryMetadata
// is the zero value, view assembly treats the request as if it arrived
// via EntryPathSearch with no context to surface.
type WorkspaceRequest struct {
	ResidentID    uuid.UUID
	EntryPath     EntryPath
	PharmacistID  uuid.UUID
	SessionID     uuid.UUID
	AsOf          time.Time
	EntryMetadata EntryPathMetadata
}

// EntryContext is the polymorphic per-path context carried in
// EntryPathMetadata. Each of the four entry paths supplies a concrete
// implementation that names the path it belongs to via Kind().
//
// Concrete implementations live in this package so they can be referenced
// by both the entry-path handlers (internal/entry_paths) and the CAPE
// context band renderer without import cycles.
type EntryContext interface {
	// Kind returns the EntryPath this context corresponds to.
	Kind() EntryPath
}

// EntryPathMetadata is the v1.0 Part 3.5 audit record produced by each
// entry-path handler. It carries entry context into workspace assembly
// so that downstream view-builders can surface the appropriate top-band.
type EntryPathMetadata struct {
	TriggeredAt  time.Time
	PharmacistID uuid.UUID
	ResidentID   uuid.UUID
	Path         EntryPath
	Context      EntryContext
}

// WorklistContext carries CAPE prioritisation signals per v1.0 Part 4.1
// Component 2 (CAPE context band).
//
// TODO(kb-33 Step 5 integration): replace stub fields with actual CAPE
// outputs from kb-33-triage-engine. The real shape — including dimension
// score breakdowns, instability chronology link, and substrate references
// — lives in kb-33, which is not yet built.
type WorklistContext struct {
	PrimarySignals []string
	CAPEScore      float64
	TriagedAt      time.Time
}

// Kind implements EntryContext.
func (WorklistContext) Kind() EntryPath { return EntryPathWorklist }

// SearchContext carries the search query that brought the pharmacist to
// this resident per v1.0 Part 3.2.
type SearchContext struct {
	Query     string
	MatchedAt time.Time
}

// Kind implements EntryContext.
func (SearchContext) Kind() EntryPath { return EntryPathSearch }

// NotificationContext carries the in-app or email notification that
// dispatched the pharmacist to S2 per v1.0 Part 3.3.
type NotificationContext struct {
	NotificationID uuid.UUID
	ReasonText     string
	DispatchedAt   time.Time
}

// Kind implements EntryContext.
func (NotificationContext) Kind() EntryPath { return EntryPathNotification }

// CrossReferenceContext carries the origin resident and reason for a
// cross-reference entry per v1.0 Part 3.4.
type CrossReferenceContext struct {
	OriginResidentID uuid.UUID
	ReasonCode       string
}

// Kind implements EntryContext.
func (CrossReferenceContext) Kind() EntryPath { return EntryPathCrossReference }

// SubstrateRef is the verification-not-belief anchor per v1.0 Part 10:
// every claim rendered in S2 carries at least one SubstrateRef back to
// the underlying observation, recommendation, or audit row.
//
// Source names the substrate origin (e.g., "kb-20", "kb-32", "kb-33").
// ID is the substrate row identifier. Description is a short
// human-readable label for the substrate object.
type SubstrateRef struct {
	Source      string
	ID          uuid.UUID
	Description string
}

// View is the marker interface implemented by all five Layer*View types so
// that EscalateToLayer can return a polymorphic view value. The single
// Layer() method exposes which layer the concrete view represents.
type View interface {
	Layer() int
}

// Layer1View is the baseline rendering view (Addendum §3.1). It is the
// pharmacist's standard S2 workspace and is the only layer populated in
// Phase 1.
//
// Content fields are added in Tasks 3–7 of the build plan per S2 v1.0
// Parts 4–13. Task 1 establishes the architectural slot only.
type Layer1View struct{}

// Layer2View is the escalated-context rendering view (Addendum §3.2).
// Content is deferred to senior consultant pharmacist authoring + pilot
// evidence on actual pharmacist behaviour (Addendum §6.1). Empty in Phase 1.
type Layer2View struct{}

// Layer3View is the complex-cognition rendering view (Addendum §3.3).
// Multi-domain concern vectors and the "what experts typically check"
// memory aid are deferred to senior consultant pharmacist authoring
// (Addendum §6.1). Empty in Phase 1.
type Layer3View struct{}

// Layer4View is the situation-board view (Addendum §3.4). Section
// composition, default order, and freshness indicators are deferred to
// senior consultant pharmacist + pilot evidence + clinical informatics
// validation (Addendum §6.1). Empty in Phase 1.
type Layer4View struct{}

// Layer5View is the deep-investigation view (Addendum §3.5). Investigation
// patterns (observation lineage, negative-evidence audit, reasoning replay)
// are deferred until Phase 1 pilot evidence indicates what investigation
// patterns pharmacists actually need (Addendum §6.1). Empty in Phase 1.
type Layer5View struct{}

// Layer returns the layer number for each view type — single-method
// implementations satisfy the View interface.

func (Layer1View) Layer() int { return 1 }
func (Layer2View) Layer() int { return 2 }
func (Layer3View) Layer() int { return 3 }
func (Layer4View) Layer() int { return 4 }
func (Layer5View) Layer() int { return 5 }

// EscalationTrigger classifies how a layer transition was initiated.
// Addendum §5.1 distinguishes automatic (substrate-signal-driven) from
// pharmacist-initiated escalations; both are logged for audit only in
// Phase 1 per §5.5.
type EscalationTrigger int

const (
	// TriggerAutomatic indicates the platform proposed the escalation
	// based on substrate signals.
	TriggerAutomatic EscalationTrigger = iota
	// TriggerPharmacistInitiated indicates the pharmacist explicitly
	// requested the escalation.
	TriggerPharmacistInitiated
)

// String returns the canonical lower_snake_case rendering used in audit
// records so log readers do not have to map integers back to semantics.
func (t EscalationTrigger) String() string {
	switch t {
	case TriggerAutomatic:
		return "automatic"
	case TriggerPharmacistInitiated:
		return "pharmacist_initiated"
	default:
		return "unknown"
	}
}

// EscalationEvent is the audit record emitted whenever the pharmacist (or
// the platform on the pharmacist's behalf) crosses a layer boundary in S2.
//
// Schema per Addendum §8.1 lines 411–422. Phase 1 deployment logs these
// for audit only; no platform behaviour is driven by them per §5.5.
type EscalationEvent struct {
	PharmacistID uuid.UUID
	ResidentID   uuid.UUID
	SessionID    uuid.UUID
	FromLayer    int
	ToLayer      int
	TriggeredBy  EscalationTrigger
	Timestamp    time.Time
	AuditTraceID uuid.UUID
}
