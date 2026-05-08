package algorithmic_distinction

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Class identifies which epistemic category an AlgorithmicObservation belongs to.
// Per Self-Visibility Guidelines §6, every surface element carries a class marker so
// the pharmacist can distinguish substrate facts, platform suggestions, own reflections,
// and hybrid observations (suggestion confirmed by pharmacist).
type Class string

const (
	// ClassSubstrateFact — computed from EvidenceTrace; no algorithmic inference.
	ClassSubstrateFact Class = "substrate_fact"
	// ClassPlatformSuggestion — algorithmic pattern detection; not yet pharmacist-confirmed.
	ClassPlatformSuggestion Class = "platform_suggestion"
	// ClassPharmacistReflection — authored by the pharmacist; POA visibility class.
	ClassPharmacistReflection Class = "pharmacist_reflection"
	// ClassHybrid — platform suggestion that the pharmacist has explicitly confirmed.
	ClassHybrid Class = "hybrid"
)

// Valid reports whether c is one of the four recognised observation classes.
func (c Class) Valid() bool {
	switch c {
	case ClassSubstrateFact, ClassPlatformSuggestion, ClassPharmacistReflection, ClassHybrid:
		return true
	}
	return false
}

// IsValidClass is a package-level convenience that mirrors the Valid() pattern used
// across Phase 1a/1b (VisibilityClass.Valid, IsValidPurpose).
func IsValidClass(s string) bool {
	return Class(s).Valid()
}

// ErrCannotConfirm is returned when Confirm is called on an observation whose class
// is not ClassPlatformSuggestion.
var ErrCannotConfirm = errors.New("algorithmic_distinction: only platform_suggestion observations can be confirmed")

// Observation is a single surface element shown on the pharmacist self-visibility
// dashboard. It carries a Class marker so the pharmacist always knows the epistemic
// provenance of what they are reading.
type Observation struct {
	ID                uuid.UUID
	Class             Class
	PharmacistID      uuid.UUID  // subject of the observation
	Body              string
	AlgorithmicOrigin *string    // pattern detector / rule ID; for suggestion + hybrid
	ConfirmedBy       *uuid.UUID // for hybrid only
	ConfirmedAt       *time.Time // for hybrid only
	CreatedAt         time.Time
}

// Confirm transitions a PlatformSuggestion to Hybrid when the pharmacist confirms
// the observation (e.g., writes a reflective entry aligning with it).
// Returns ErrCannotConfirm if the observation is not a platform suggestion.
func (o Observation) Confirm(by uuid.UUID, at time.Time) (Observation, error) {
	if o.Class != ClassPlatformSuggestion {
		return o, ErrCannotConfirm
	}
	o.Class = ClassHybrid
	o.ConfirmedBy = &by
	at = at.UTC()
	o.ConfirmedAt = &at
	return o, nil
}

// IsConfirmed reports whether the observation has been pharmacist-confirmed:
// it must be ClassHybrid with both ConfirmedBy and ConfirmedAt set.
func (o Observation) IsConfirmed() bool {
	return o.Class == ClassHybrid && o.ConfirmedBy != nil && o.ConfirmedAt != nil
}
