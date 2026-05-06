package validation

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ValidateCapacityAssessment reports any structural problem with c. Per
// Layer 2 doc §2.5 (Wave 2.5 of the Layer 2 substrate plan).
//
// Required fields:
//   - ResidentRef     (uuid.Nil rejected)
//   - AssessedAt      (zero time rejected)
//   - AssessorRoleRef (uuid.Nil rejected)
//   - Domain          (must be one of models.CapacityDomain* constants)
//   - Outcome         (must be one of models.CapacityOutcome* constants)
//   - Duration        (must be one of models.CapacityDuration* constants)
//
// Cross-field invariants (clinical correctness — see Layer 2 doc §2.5):
//   - Outcome=intact MUST pair with Duration=permanent. Intact capacity
//     is not a temporary state — a "temporarily intact" capacity is
//     incoherent.
//   - Duration=temporary MUST set ExpectedReviewDate, and that date MUST
//     be strictly after AssessedAt. A temporary capacity finding without
//     a review date provides no plan for re-evaluation.
//   - If Score is set, Instrument MUST also be set. A numeric score
//     without naming the instrument that produced it is uninterpretable.
//   - SupersedesRef may be nil for the first assessment for a (resident,
//     domain) pair; when set it MUST NOT equal ID (no self-loop).
func ValidateCapacityAssessment(c models.CapacityAssessment) error {
	if c.ResidentRef == uuid.Nil {
		return errors.New("resident_ref is required")
	}
	if c.AssessedAt.IsZero() {
		return errors.New("assessed_at is required")
	}
	if c.AssessorRoleRef == uuid.Nil {
		return errors.New("assessor_role_ref is required")
	}
	if c.Domain == "" {
		return errors.New("domain is required")
	}
	if !models.IsValidCapacityDomain(c.Domain) {
		return fmt.Errorf("invalid domain %q", c.Domain)
	}
	if c.Outcome == "" {
		return errors.New("outcome is required")
	}
	if !models.IsValidCapacityOutcome(c.Outcome) {
		return fmt.Errorf("invalid outcome %q", c.Outcome)
	}
	if c.Duration == "" {
		return errors.New("duration is required")
	}
	if !models.IsValidCapacityDuration(c.Duration) {
		return fmt.Errorf("invalid duration %q", c.Duration)
	}

	// Cross-field rule 1: intact ⇒ permanent.
	if c.Outcome == models.CapacityOutcomeIntact && c.Duration != models.CapacityDurationPermanent {
		return fmt.Errorf("outcome=intact requires duration=permanent (got duration=%q)", c.Duration)
	}

	// Cross-field rule 2: temporary ⇒ ExpectedReviewDate set and strictly after AssessedAt.
	if c.Duration == models.CapacityDurationTemporary {
		if c.ExpectedReviewDate == nil {
			return errors.New("duration=temporary requires expected_review_date")
		}
		if !c.ExpectedReviewDate.After(c.AssessedAt) {
			return errors.New("expected_review_date must be after assessed_at")
		}
	}

	// Cross-field rule 3: Score requires Instrument.
	if c.Score != nil && c.Instrument == "" {
		return errors.New("score requires instrument")
	}

	// SupersedesRef self-loop guard.
	if c.SupersedesRef != nil && c.ID != uuid.Nil && *c.SupersedesRef == c.ID {
		return errors.New("supersedes_ref must not equal id (self-loop)")
	}
	return nil
}
