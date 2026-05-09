// Package consent_extension extends the Plan 0.2 Consent state machine for
// restrictive practices per Guidelines §6.3. It provides the
// RestrictivePracticeConsent entity, which gates ERM recommendations
// involving psychotropic medication, physical restraint, environmental
// restraint, or seclusion on an active consent for the specific practice type.
//
// All transitions in this package are traced; records are audit-defensible
// under the Aged Care Quality Standards 2026 and the Restrictive Practice
// Regulations 2019.
//
// VisibilityClass: AD — extends Plan 0.2 Consent state machine
package consent_extension

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// PracticeType identifies the category of restrictive practice for which
// consent is being recorded.
type PracticeType string

const (
	// PracticeChemicalRestraint covers psychotropic medications used to
	// influence behaviour or movement. MaxDuration MUST NOT exceed 12 weeks
	// (see Validate and Guidelines §6.3).
	PracticeChemicalRestraint PracticeType = "chemical_restraint"

	// PracticePhysicalRestraint covers mechanical or manual restriction of
	// movement.
	PracticePhysicalRestraint PracticeType = "physical_restraint"

	// PracticeEnvironmentalRestraint covers restriction of movement by
	// environmental means (e.g. locked units, bed rails).
	PracticeEnvironmentalRestraint PracticeType = "environmental_restraint"

	// PracticeSeclusion covers involuntary confinement of a resident alone in
	// a room or area they are not free to leave.
	PracticeSeclusion PracticeType = "seclusion"
)

// maxChemicalRestraintDuration is the regulatory cap on chemical-restraint
// consent duration per Guidelines §6.3.
const maxChemicalRestraintDuration = 12 * 7 * 24 * time.Hour // 12 weeks

// IsValidPracticeType returns true when s is one of the four canonical
// PracticeType string values.
func IsValidPracticeType(s string) bool {
	switch PracticeType(s) {
	case PracticeChemicalRestraint, PracticePhysicalRestraint,
		PracticeEnvironmentalRestraint, PracticeSeclusion:
		return true
	default:
		return false
	}
}

// RestrictivePracticeConsent records the authorisation for a specific
// restrictive practice for a specific resident. It extends the Plan 0.2
// Consent state machine: ConsentID is a foreign key to the consents table
// (migration 024).
//
// ERM gates recommendations involving a practice type on Allows returning true
// for the corresponding RestrictivePracticeConsent.
type RestrictivePracticeConsent struct {
	// ID uniquely identifies this restrictive-practice consent record.
	ID uuid.UUID

	// ConsentID is the FK to the Plan 0.2 consents table (migration 024).
	ConsentID uuid.UUID

	// PracticeType is the category of restrictive practice being authorised.
	PracticeType PracticeType

	// Status is the lifecycle state of the consent record.
	// Valid values: "requested" / "discussed" / "active" / "expired" / "withdrawn".
	Status string

	// LessRestrictiveAlternativesDocumented must be true before Allows returns
	// true. Guidelines §6.3 mandates that less-restrictive alternatives have
	// been considered and documented before a restrictive practice may proceed.
	LessRestrictiveAlternativesDocumented bool

	// BehaviourSupportPlanRef is an optional reference to the behaviour support
	// plan associated with this consent.
	BehaviourSupportPlanRef *uuid.UUID

	// SDMConsentRecordRef is an optional reference to the substitute
	// decision-maker consent record.
	SDMConsentRecordRef *uuid.UUID

	// GrantedAt is the UTC timestamp at which the consent was granted.
	GrantedAt time.Time

	// MaxDuration is the maximum authorised duration. For chemical restraint,
	// this MUST NOT exceed 12 weeks (enforced by Validate).
	MaxDuration time.Duration

	// DesignatedPractitionerID identifies the clinician responsible for
	// overseeing this restrictive practice.
	DesignatedPractitionerID uuid.UUID

	// MandatoryReviewDueAt is the timestamp by which the consent must be
	// reviewed by the designated practitioner.
	MandatoryReviewDueAt time.Time
}

// Allows returns true when all three conditions are satisfied as of asOf:
//
//  1. Status is "active".
//  2. LessRestrictiveAlternativesDocumented is true.
//  3. The consent has not expired (GrantedAt + MaxDuration > asOf).
//
// A false return MUST be treated by callers as a hard gate — the practice
// MUST NOT proceed without human review and re-authorisation.
func (c RestrictivePracticeConsent) Allows(asOf time.Time) bool {
	if c.Status != "active" {
		return false
	}
	if !c.LessRestrictiveAlternativesDocumented {
		return false
	}
	expiry := c.GrantedAt.Add(c.MaxDuration)
	if asOf.After(expiry) {
		return false
	}
	return true
}

// Validate checks invariants that cannot be expressed as type constraints.
// Currently enforces:
//   - For PracticeChemicalRestraint, MaxDuration MUST be ≤ 12 weeks
//     (Guidelines §6.3).
//
// Returns nil when all invariants are satisfied.
func (c RestrictivePracticeConsent) Validate() error {
	if c.PracticeType == PracticeChemicalRestraint && c.MaxDuration > maxChemicalRestraintDuration {
		return errors.New("consent_extension: chemical restraint MaxDuration exceeds 12-week regulatory cap (Guidelines §6.3)")
	}
	return nil
}
