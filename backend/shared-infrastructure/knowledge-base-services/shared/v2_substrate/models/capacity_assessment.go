// Package models — CapacityAssessment is the per-Resident, per-domain
// capacity entity introduced by Wave 2.5 of the Layer 2 substrate plan
// (Layer 2 doc §2.5).
//
// Capacity is dynamic, domain-specific, and date-stamped. The substrate
// captures it as a separate object (not a single Resident attribute)
// because:
//
//   - It changes (capacity can be lost permanently or temporarily).
//   - It's domain-specific: medical decisions, financial decisions,
//     accommodation decisions, restrictive-practice decisions, and
//     medication decisions all have different capacity standards. A
//     resident may have intact medical capacity but impaired financial
//     capacity, or vice versa.
//   - It's date-stamped and has an assessor (Role).
//   - It interacts with Consent state: a resident with capacity gives
//     consent themselves; without capacity, the SDM does. The Consent
//     state machine consumes capacity_change Events emitted when the
//     medical_decisions domain transitions to impaired.
//
// Canonical storage: kb-20-patient-profile (capacity_assessments table,
// migration 017). The history is append-only — never UPDATE rows. The
// latest row by AssessedAt per (ResidentRef, Domain) is the current
// assessment for that domain (queried via the capacity_current view).
//
// FHIR boundary: maps to FHIR Observation with category=assessment and a
// Vaidshala-defined CodeSystem (capacity-assessment) keyed by Domain.
// See shared/v2_substrate/fhir/capacity_assessment_mapper.go.
package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CapacityDomain values per Layer 2 doc §2.5. The domain set is closed:
// validators reject unknown domains at the model boundary; the
// capacity_assessments.domain CHECK constraint is the storage backstop.
const (
	CapacityDomainMedical             = "medical_decisions"
	CapacityDomainFinancial           = "financial"
	CapacityDomainAccommodation       = "accommodation"
	CapacityDomainRestrictivePractice = "restrictive_practice"
	CapacityDomainMedicationDecisions = "medication_decisions"
)

// IsValidCapacityDomain reports whether s is one of the five recognised
// CapacityDomain* values. Empty string is rejected.
func IsValidCapacityDomain(s string) bool {
	switch s {
	case CapacityDomainMedical, CapacityDomainFinancial,
		CapacityDomainAccommodation, CapacityDomainRestrictivePractice,
		CapacityDomainMedicationDecisions:
		return true
	}
	return false
}

// CapacityOutcome values per Layer 2 doc §2.5.
const (
	CapacityOutcomeIntact         = "intact"
	CapacityOutcomeImpaired       = "impaired"
	CapacityOutcomeUnableToAssess = "unable_to_assess"
)

// IsValidCapacityOutcome reports whether s is one of the three recognised
// CapacityOutcome* values.
func IsValidCapacityOutcome(s string) bool {
	switch s {
	case CapacityOutcomeIntact, CapacityOutcomeImpaired, CapacityOutcomeUnableToAssess:
		return true
	}
	return false
}

// CapacityDuration values per Layer 2 doc §2.5.
const (
	CapacityDurationPermanent         = "permanent"
	CapacityDurationTemporary         = "temporary"
	CapacityDurationUnableToDetermine = "unable_to_determine"
)

// IsValidCapacityDuration reports whether s is one of the three recognised
// CapacityDuration* values.
func IsValidCapacityDuration(s string) bool {
	switch s {
	case CapacityDurationPermanent, CapacityDurationTemporary, CapacityDurationUnableToDetermine:
		return true
	}
	return false
}

// CapacityInstrument values per Layer 2 doc §2.5. The instrument set is
// open in the doc ("MMSE | MoCA | clinical_judgement | other") — these
// constants are the convenience set; validators allow any non-empty
// string when an instrument is supplied. The constants exist so callers
// can reference them by symbol rather than by free-text literal.
const (
	CapacityInstrumentMMSE              = "MMSE"
	CapacityInstrumentMoCA              = "MoCA"
	CapacityInstrumentClinicalJudgement = "clinical_judgement"
	CapacityInstrumentOther             = "other"
)

// CapacityAssessment is one append-only assessment row for a (resident,
// domain) pair. The latest by AssessedAt per (ResidentRef, Domain) is
// the current assessment for that domain.
//
// Cross-field invariants enforced by validation.ValidateCapacityAssessment:
//
//   - If Outcome=intact, Duration MUST be permanent (clinical sanity:
//     intact capacity is not a temporary state).
//   - If Duration=temporary, ExpectedReviewDate MUST be set and after
//     AssessedAt.
//   - If Score is set, Instrument MUST also be set (a numeric score
//     without an instrument is incoherent).
type CapacityAssessment struct {
	ID                  uuid.UUID       `json:"id"`
	ResidentRef         uuid.UUID       `json:"resident_ref"`
	AssessedAt          time.Time       `json:"assessed_at"`
	AssessorRoleRef     uuid.UUID       `json:"assessor_role_ref"`
	Domain              string          `json:"domain"`               // see CapacityDomain* constants
	Instrument          string          `json:"instrument,omitempty"` // see CapacityInstrument* constants
	Score               *float64        `json:"score,omitempty"`      // pointer-nullable for optional numeric instruments
	Outcome             string          `json:"outcome"`              // see CapacityOutcome* constants
	Duration            string          `json:"duration"`             // see CapacityDuration* constants
	ExpectedReviewDate  *time.Time      `json:"expected_review_date,omitempty"`
	RationaleStructured json.RawMessage `json:"rationale_structured,omitempty"`
	RationaleFreeText   string          `json:"rationale_free_text,omitempty"`
	SupersedesRef       *uuid.UUID      `json:"supersedes_ref,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
}
