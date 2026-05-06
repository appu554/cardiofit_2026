package validation

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ValidateCareIntensity reports any structural problem with c. Per Layer 2
// doc §2.4 (Wave 2.4 of the Layer 2 substrate plan).
//
// Required fields:
//   - ResidentRef         (uuid.Nil rejected)
//   - Tag                 (must be one of models.CareIntensityTag* constants)
//   - EffectiveDate       (zero time rejected)
//   - DocumentedByRoleRef (uuid.Nil rejected)
//
// Conditional rules:
//   - If ReviewDueDate is set, it must be >= EffectiveDate.
//   - SupersedesRef may be nil for a resident's first CareIntensity row;
//     when set it must not equal ID (a row cannot supersede itself).
func ValidateCareIntensity(c models.CareIntensity) error {
	if c.ResidentRef == uuid.Nil {
		return errors.New("resident_ref is required")
	}
	if c.Tag == "" {
		return errors.New("tag is required")
	}
	if !models.IsValidCareIntensityTag(c.Tag) {
		return fmt.Errorf("invalid tag %q", c.Tag)
	}
	if c.EffectiveDate.IsZero() {
		return errors.New("effective_date is required")
	}
	if c.DocumentedByRoleRef == uuid.Nil {
		return errors.New("documented_by_role_ref is required")
	}
	if c.ReviewDueDate != nil && c.ReviewDueDate.Before(c.EffectiveDate) {
		return errors.New("review_due_date must be >= effective_date")
	}
	if c.SupersedesRef != nil && c.ID != uuid.Nil && *c.SupersedesRef == c.ID {
		return errors.New("supersedes_ref must not equal id (self-loop)")
	}
	return nil
}

// ValidateCareIntensityTransition reports whether moving from `fromTag` to
// `toTag` is permitted. Used by the engine + handler before producing the
// transition Event so the substrate never records a transition into an
// unknown tag.
//
// `fromTag` may be empty (first row for the resident); `toTag` must be a
// valid models.CareIntensityTag* value.
func ValidateCareIntensityTransition(fromTag, toTag string) error {
	if toTag == "" {
		return errors.New("target tag is required")
	}
	if !models.IsValidCareIntensityTag(toTag) {
		return fmt.Errorf("invalid target tag %q", toTag)
	}
	if fromTag != "" && !models.IsValidCareIntensityTag(fromTag) {
		return fmt.Errorf("invalid source tag %q", fromTag)
	}
	if !models.IsValidCareIntensityTransition(fromTag, toTag) {
		return fmt.Errorf("illegal care-intensity transition %s → %s", fromTag, toTag)
	}
	return nil
}
