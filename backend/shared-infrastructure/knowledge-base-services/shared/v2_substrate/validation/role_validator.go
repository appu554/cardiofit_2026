package validation

import (
	"errors"
	"fmt"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ValidateRole reports any structural problem with r.
func ValidateRole(r models.Role) error {
	if !models.IsValidRoleKind(r.Kind) {
		return fmt.Errorf("invalid kind %q (see RoleKind constants)", r.Kind)
	}
	if r.ValidTo != nil && r.ValidTo.Before(r.ValidFrom) {
		return errors.New("valid_to must be on or after valid_from")
	}
	return nil
}
