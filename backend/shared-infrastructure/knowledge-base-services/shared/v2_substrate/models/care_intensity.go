package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CareIntensity is a per-Resident first-class entity that captures the
// resident's overall care plan posture as described in Layer 2 doc §2.4.
// Care intensity is the single most important context variable shaping
// recommendations: the same recommendation has different framing depending
// on whether the resident is in active_treatment, rehabilitation,
// comfort_focused, or palliative care.
//
// The previous β.1 design encoded care intensity as a single denormalised
// string field on Resident (models.Resident.CareIntensity). Wave 2.4 of
// the Layer 2 substrate plan promotes care intensity to its own
// append-only history entity so transitions are first-class events that
// propagate through the substrate (active concerns may resolve, existing
// recommendations may be re-evaluated, monitoring plans may be revised).
//
// Canonical storage: kb-20-patient-profile (care_intensity_history table,
// migration 016). The latest row by EffectiveDate per ResidentRef is the
// current tag; persisted via the care_intensity_current view.
//
// FHIR boundary: not currently mapped — care intensity is Vaidshala-internal
// per Layer 2 doc §2.4 (the doc does not specify a FHIR mapping; a CodeSystem
// mapping can be added in Layer 3 if regulator-facing surfaces need it).
type CareIntensity struct {
	ID                  uuid.UUID       `json:"id"`
	ResidentRef         uuid.UUID       `json:"resident_ref"`
	Tag                 string          `json:"tag"` // see CareIntensityTag* constants
	EffectiveDate       time.Time       `json:"effective_date"`
	DocumentedByRoleRef uuid.UUID       `json:"documented_by_role_ref"`
	ReviewDueDate       *time.Time      `json:"review_due_date,omitempty"`
	RationaleStructured json.RawMessage `json:"rationale_structured,omitempty"` // SNOMED + ICD codes capturing prognostic findings
	RationaleFreeText   string          `json:"rationale_free_text,omitempty"`
	SupersedesRef       *uuid.UUID      `json:"supersedes_ref,omitempty"` // points at prior CareIntensity row this one transitions from
	CreatedAt           time.Time       `json:"created_at"`
}

// CareIntensityTag values per Layer 2 doc §2.4. The tag set is closed:
// validators reject unknown tags at the model boundary; the
// care_intensity_history.tag CHECK constraint is the storage-level backstop.
//
// These constants live in addition to the legacy CareIntensity* values in
// enums.go (active, comfort, rehabilitation, palliative). The legacy set
// is retained for the denormalised Resident.CareIntensity field; the
// CareIntensityTag* set is the canonical vocabulary for the v2.4 entity.
//
// Layer 2 doc §2.4 wording is the source of truth: "active_treatment" and
// "comfort_focused" are the doc's preferred forms (vs. the legacy short
// "active" / "comfort"). Wave 2.4 records transitions in the doc's
// vocabulary; downstream consumers map to the legacy short forms via
// LegacyCareIntensityForTag if they still read Resident.CareIntensity.
const (
	CareIntensityTagActiveTreatment = "active_treatment"
	CareIntensityTagRehabilitation  = "rehabilitation"
	CareIntensityTagComfortFocused  = "comfort_focused"
	CareIntensityTagPalliative      = "palliative"
)

// IsValidCareIntensityTag reports whether s is one of the four recognised
// CareIntensityTag* values. Empty string is rejected (use the validator's
// dedicated empty-tag error rather than relying on this).
func IsValidCareIntensityTag(s string) bool {
	switch s {
	case CareIntensityTagActiveTreatment, CareIntensityTagRehabilitation,
		CareIntensityTagComfortFocused, CareIntensityTagPalliative:
		return true
	}
	return false
}

// IsValidCareIntensityTransition reports whether moving from `from` to
// `to` is clinically defensible. All transitions between valid tags are
// permitted in MVP (clinical flexibility — a resident may step back from
// palliative to comfort_focused if their condition improves); the engine
// uses the direction information to emit appropriate cascade hints
// (review preventive medications, refresh consent, etc.).
//
// `from` may be empty string for a resident's very first CareIntensity row
// (no prior tag); `to` must be a valid tag.
func IsValidCareIntensityTransition(from, to string) bool {
	if !IsValidCareIntensityTag(to) {
		return false
	}
	if from == "" {
		return true
	}
	return IsValidCareIntensityTag(from)
}

// LegacyCareIntensityForTag maps a Wave 2.4 CareIntensityTag* value to the
// legacy CareIntensity* short form used by Resident.CareIntensity. The
// mapping is one-way (Wave 2.4 vocabulary → legacy enum) so consumers
// that still read the denormalised Resident field receive the closest
// equivalent value.
//
// Returns "" if t is not a recognised CareIntensityTag* value.
func LegacyCareIntensityForTag(t string) string {
	switch t {
	case CareIntensityTagActiveTreatment:
		return CareIntensityActive
	case CareIntensityTagRehabilitation:
		return CareIntensityRehabilitation
	case CareIntensityTagComfortFocused:
		return CareIntensityComfort
	case CareIntensityTagPalliative:
		return CareIntensityPalliative
	}
	return ""
}
