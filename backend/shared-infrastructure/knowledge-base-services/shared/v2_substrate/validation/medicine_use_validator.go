package validation

import (
	"errors"
	"fmt"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ValidateMedicineUse reports any structural problem with m. Includes
// validation of nested Intent.Category, Target (delegated to ValidateTarget),
// and StopCriteria.Triggers.
func ValidateMedicineUse(m models.MedicineUse) error {
	if m.DisplayName == "" {
		return errors.New("display_name is required")
	}
	if !models.IsValidMedicineUseStatus(m.Status) {
		return fmt.Errorf("invalid status %q", m.Status)
	}
	if !models.IsValidIntentCategory(m.Intent.Category) {
		return fmt.Errorf("invalid intent.category %q", m.Intent.Category)
	}
	if m.Intent.Indication == "" {
		return errors.New("intent.indication is required")
	}
	if err := ValidateTarget(m.Target); err != nil {
		return fmt.Errorf("target invalid: %w", err)
	}
	for i, trig := range m.StopCriteria.Triggers {
		if !models.IsValidStopTrigger(trig) {
			return fmt.Errorf("invalid stop_criteria.triggers[%d] %q", i, trig)
		}
	}
	if m.EndedAt != nil && m.EndedAt.Before(m.StartedAt) {
		return errors.New("ended_at must be on or after started_at")
	}
	return nil
}
