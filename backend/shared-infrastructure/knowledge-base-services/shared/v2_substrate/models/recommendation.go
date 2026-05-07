package models

import (
	"time"

	"github.com/google/uuid"
)

// Recommendation is a v2/v3 substrate entity capturing a proposed clinical
// action through its lifecycle. It is the keystone entity for the v3 product
// thesis: without persistence of recommendations, RIR cannot be computed,
// rationale survival cannot be measured, and the craft engine has no
// substrate to render against.
//
// State transitions are governed by recommendation.Lifecycle, which writes
// an EvidenceTrace edge per transition. Direct State mutation outside the
// Lifecycle engine is a contract violation.
//
// Canonical storage: migrations/023_recommendation_lifecycle.sql
// (table: recommendations).
type Recommendation struct {
	ID         uuid.UUID `json:"id"`
	ResidentID uuid.UUID `json:"resident_id"`
	AuthorID   uuid.UUID `json:"author_id"` // Person.id (typically the ACOP pharmacist)

	State   string `json:"state"`   // see RecommendationState* constants
	Type    string `json:"type"`    // see RecommendationType* constants
	Urgency string `json:"urgency"` // see RecommendationUrgency* constants

	Title           string          `json:"title"`
	ClinicalContent ClinicalContent `json:"clinical_content"` // v3 §7: invariant across framings

	// MedicineUseRefs links to MedicineUse entities this recommendation
	// targets (cease X, dose-change Y, add Z).
	MedicineUseRefs []uuid.UUID `json:"medicine_use_refs"`

	// ConsentRequired is set true at draft time when the recommendation class
	// requires a matching active Consent (e.g. psychotropic, restrictive
	// practice). The Lifecycle.Submit guard enforces this.
	ConsentRequired bool `json:"consent_required"`

	// ReviewDueAt is the forced review timestamp when in deferred state.
	// Nil for non-deferred states. The deferred escalator sweeps this column.
	ReviewDueAt *time.Time `json:"review_due_at,omitempty"`

	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	DecidedAt   *time.Time `json:"decided_at,omitempty"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ClinicalContent is the audience-invariant clinical substance of a
// recommendation. v3 §7 line 416 mandates that this is recorded separately
// from any per-audience framing so a regulator audit query can verify
// content invariance across framings.
type ClinicalContent struct {
	Issue           string   `json:"issue"`
	ClinicalContext string   `json:"clinical_context"`
	Rationale       string   `json:"rationale"`
	EvidenceRefs    []string `json:"evidence_refs"`
	ProposedPlan    string   `json:"proposed_plan"`
	MonitoringPlan  string   `json:"monitoring_plan"`
}

// validTransitions encodes the recommendation lifecycle DAG. A pair (from, to)
// is in the map iff the transition is permitted. Direct mutation outside
// recommendation.Lifecycle is a contract violation; this function exists so
// the Lifecycle engine and storage layer share one source of truth.
var validTransitions = map[string]map[string]bool{
	RecommendationStateDetected: {
		RecommendationStateDrafted: true,
		RecommendationStateClosed:  true,
	},
	RecommendationStateDrafted: {
		RecommendationStateSubmitted: true,
		RecommendationStateClosed:    true,
	},
	RecommendationStateSubmitted: {
		RecommendationStateViewed:   true,
		RecommendationStateDeferred: true,
		RecommendationStateClosed:   true,
	},
	RecommendationStateViewed: {
		RecommendationStateDecided:  true,
		RecommendationStateDeferred: true,
		RecommendationStateClosed:   true,
	},
	RecommendationStateDeferred: {
		RecommendationStateSubmitted: true, // re-surfaced
		RecommendationStateClosed:    true, // expired without action
	},
	RecommendationStateDecided: {
		RecommendationStateImplemented: true,
		RecommendationStateClosed:      true, // decided-no-action
	},
	RecommendationStateImplemented: {
		RecommendationStateMonitoringActive: true,
		RecommendationStateOutcomeRecorded:  true, // skip monitoring if not warranted
	},
	RecommendationStateMonitoringActive: {
		RecommendationStateOutcomeRecorded: true,
	},
	RecommendationStateOutcomeRecorded: {
		RecommendationStateClosed: true,
	},
	// RecommendationStateClosed is terminal; no entry.
}

// IsValidTransition reports whether the lifecycle DAG permits from → to.
func IsValidTransition(from, to string) bool {
	if !IsValidRecommendationState(from) || !IsValidRecommendationState(to) {
		return false
	}
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}
