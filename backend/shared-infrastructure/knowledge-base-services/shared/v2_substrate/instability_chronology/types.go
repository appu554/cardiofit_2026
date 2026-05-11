// Package instability_chronology defines the Instability Chronology
// substrate — CAPE's cross-parameter temporal narrative primitive that
// composes events across multiple clinical parameters into a single
// audience-neutral chronology with audience-adapted renderings.
//
// Spec: docs/superpowers/plans/CAPE_v1_1_Architectural_Commitment_Addendum.md
// §3 (lines 230–345). This package ships TYPES ONLY. The CAPE engine
// (kb-33, Roadmap Step 5) is the first writer; surface services
// (pharmacist, RACH operator, governance, family communication, audit
// defensibility) are the first readers. No computation, no persistence,
// no I/O lives here.
//
// Phase 1 vocabulary expansion for InstabilityPrimitive and the
// TemporalPattern library is explicitly gated on senior consultant
// pharmacist pattern authoring (Addendum line 341); do not extend the
// canonical primitive set in this package without that gate.
//
// VisibilityClass: PDP (Pharmacist-Default-Private) — resident clinical
// narrative derived from underlying substrate.
package instability_chronology

import (
	"time"

	"github.com/google/uuid"
)

// InstabilityChronology is a temporal narrative composed of multi-parameter
// events that together describe a resident's destabilisation. Computed by
// the CAPE engine (kb-33, Roadmap Step 5) — this package ships types only.
// Spec: CAPE_v1_1_Architectural_Commitment_Addendum.md lines 232–260.
type InstabilityChronology struct {
	ResidentID          uuid.UUID                              `json:"resident_id"`
	TimeWindow          TimeWindow                             `json:"time_window"`
	Events              []ChronologyEvent                      `json:"events"`
	Patterns            []TemporalPattern                      `json:"patterns"`
	Severity            Severity                               `json:"severity"`
	AudienceAdaptations map[AudienceClass]ChronologyRendering `json:"audience_adaptations"`
}

// ChronologyEvent is one moment in the chronology — a substrate-grounded,
// audience-neutral fact. Description is factual; rendering happens via
// AudienceAdaptations on the parent chronology.
type ChronologyEvent struct {
	EventID         uuid.UUID            `json:"event_id"`
	Timestamp       time.Time            `json:"timestamp"`
	EventType       string               `json:"event_type"`     // free-form taxonomy keyword
	PrimitiveType   InstabilityPrimitive `json:"primitive_type"` // canonical vocabulary (constants below)
	Severity        Severity             `json:"severity"`
	Description     string               `json:"description"` // factual, audience-neutral
	SubstrateRefs   []SubstrateReference `json:"substrate_refs"`
	SuspectedCauses []string             `json:"suspected_causes"`
	RelatedEvents   []uuid.UUID          `json:"related_events"`
}

// TimeWindow is a half-open interval [Start, End): Start is inclusive, End
// is exclusive. This matches the convention established in prn_velocity
// (closed-recent / open-baseline pairing) and is the project-wide default.
type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Contains reports whether t falls inside the window — Start ≤ t < End.
// Left-inclusive, right-exclusive (half-open).
func (w TimeWindow) Contains(t time.Time) bool {
	if t.Before(w.Start) {
		return false
	}
	if !t.Before(w.End) { // t >= End
		return false
	}
	return true
}

// Duration returns End - Start. Zero or negative durations are allowed
// (callers may use them as sentinel values, e.g., an unset window) but
// should be uncommon in practice. No normalisation is performed.
func (w TimeWindow) Duration() time.Duration {
	return w.End.Sub(w.Start)
}

// Severity is a 1-5 scale matching the project-wide convention
// (PRN velocity, scoring layers, etc.). 1 = quiescent, 5 = critical.
type Severity int

// InstabilityPrimitive is the canonical vocabulary for ChronologyEvent.PrimitiveType.
// Addendum §3 lines 308–316 list these examples; this package canonicalises
// the Phase 1 set and reserves expansion for the senior consultant
// pharmacist pattern-authoring step (Addendum line 341) not yet engaged.
type InstabilityPrimitive string

// Canonical Phase 1 instability primitives. Do not extend without senior
// consultant pharmacist authoring per CAPE Addendum line 341.
const (
	// PrimitiveMedicationChange — dose or regimen change (start, stop, titrate).
	PrimitiveMedicationChange InstabilityPrimitive = "medication_change"

	// PrimitiveIntakeDecline — reduced oral intake (food, fluids).
	PrimitiveIntakeDecline InstabilityPrimitive = "intake_decline"

	// PrimitiveFall — fall or near-fall event.
	PrimitiveFall InstabilityPrimitive = "fall"

	// PrimitiveConfusionOnset — new or worsened cognitive change (e.g., 4AT rise).
	PrimitiveConfusionOnset InstabilityPrimitive = "confusion_onset"

	// PrimitiveOrthostaticInstability — documented orthostatic BP drop or
	// associated symptoms.
	PrimitiveOrthostaticInstability InstabilityPrimitive = "orthostatic_instability"

	// PrimitiveSedation — daytime somnolence or sedation-related decline.
	PrimitiveSedation InstabilityPrimitive = "sedation"

	// Reserved for kb-33 to extend after senior consultant pharmacist pattern authoring.
)

// ValidInstabilityPrimitives is the exported ordered list of canonical Phase 1
// primitives. Mirrors the failed_interventions / overrides ValidXxx pattern.
var ValidInstabilityPrimitives = []InstabilityPrimitive{
	PrimitiveMedicationChange,
	PrimitiveIntakeDecline,
	PrimitiveFall,
	PrimitiveConfusionOnset,
	PrimitiveOrthostaticInstability,
	PrimitiveSedation,
}

// IsValidInstabilityPrimitive reports whether s is one of the canonical Phase 1
// primitives. The empty string is not valid.
func IsValidInstabilityPrimitive(s string) bool {
	if s == "" {
		return false
	}
	for _, p := range ValidInstabilityPrimitives {
		if string(p) == s {
			return true
		}
	}
	return false
}

// TemporalPattern is a named cross-event pattern (e.g., "volume-contraction
// cascade"). The Patterns slice on InstabilityChronology lists recognised
// pattern matches; pattern authoring is reserved for senior consultant
// pharmacists (CAPE Addendum line 341).
type TemporalPattern struct {
	PatternID     string      `json:"pattern_id"`     // canonical name (e.g., "volume_contraction_cascade")
	EventSequence []uuid.UUID `json:"event_sequence"` // ChronologyEvent IDs participating in the pattern
	Reasoning     string      `json:"reasoning"`      // why the engine matched this pattern
	Confidence    float64     `json:"confidence"`     // 0.0-1.0
}

// AudienceClass identifies the consumer surface for which a chronology
// rendering has been adapted. Addendum lines 318–327 enumerate these surfaces.
type AudienceClass string

const (
	// AudiencePharmacist — pharmacist surface (S1, recommendation craft).
	AudiencePharmacist AudienceClass = "pharmacist"

	// AudienceRACHOperator — Residential Aged Care Home operational view.
	AudienceRACHOperator AudienceClass = "rach_operator"

	// AudienceGovernance — governance / quality-improvement retrospective.
	AudienceGovernance AudienceClass = "governance"

	// AudienceFamilyCommunication — family update narrative framing.
	AudienceFamilyCommunication AudienceClass = "family_communication"

	// AudienceAuditDefensibility — ACQSC / regulator audit response framing.
	AudienceAuditDefensibility AudienceClass = "audit_defensibility"
)

// ValidAudienceClasses is the exported ordered list of canonical audience classes.
var ValidAudienceClasses = []AudienceClass{
	AudiencePharmacist,
	AudienceRACHOperator,
	AudienceGovernance,
	AudienceFamilyCommunication,
	AudienceAuditDefensibility,
}

// IsValidAudienceClass reports whether s is one of the canonical audience
// classes. The empty string is not valid.
func IsValidAudienceClass(s string) bool {
	if s == "" {
		return false
	}
	for _, a := range ValidAudienceClasses {
		if string(a) == s {
			return true
		}
	}
	return false
}

// ChronologyRendering is the per-audience presentation envelope around the
// audience-neutral substrate. Concrete shapes are TODO until kb-33 + the
// surface services (S1, RACHOperationalView, etc.) author them.
type ChronologyRendering struct {
	Audience          AudienceClass `json:"audience"`
	Headline          string        `json:"headline"`           // one-line summary
	Narrative         string        `json:"narrative"`          // full prose
	HighlightedEvents []uuid.UUID   `json:"highlighted_events"` // EventIDs to emphasise in this rendering
}

// SubstrateReference points to an underlying datum (PRN administration,
// lab result, observation) that anchors a ChronologyEvent in verifiable
// data. Concrete substrate types live in their own packages (prn_velocity,
// failed_interventions, etc.); SubstrateReference is the polymorphic
// pointer used by chronology consumers.
type SubstrateReference struct {
	SubstrateType string    `json:"substrate_type"` // e.g., "prn_administration", "lab_result", "observation"
	ReferenceID   uuid.UUID `json:"reference_id"`   // the primary key in the referenced table
	Description   string    `json:"description"`    // short human-readable label
}
