// Package validation provides cross-field validation rules for v2 substrate
// entities. Validators are pure functions: they read an entity and return
// an error or nil.
package validation

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/cardiofit/shared/v2_substrate/models"
)

var ihiPattern = regexp.MustCompile(`^\d{16}$`)

// ValidateResident reports any structural problem with r. It does not
// check referential integrity (e.g. FacilityID exists) — that is the
// caller's responsibility at write time.
func ValidateResident(r models.Resident) error {
	if r.GivenName == "" {
		return errors.New("given_name is required")
	}
	if r.FamilyName == "" {
		return errors.New("family_name is required")
	}
	if !models.IsValidCareIntensity(r.CareIntensity) {
		return fmt.Errorf("invalid care_intensity %q", r.CareIntensity)
	}
	if !models.IsValidResidentStatus(r.Status) {
		return fmt.Errorf("invalid status %q", r.Status)
	}
	if r.IHI != "" && !ihiPattern.MatchString(r.IHI) {
		return fmt.Errorf("ihi must be 16 digits, got %q", r.IHI)
	}
	return nil
}
