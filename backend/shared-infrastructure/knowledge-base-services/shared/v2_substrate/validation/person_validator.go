package validation

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/cardiofit/shared/v2_substrate/models"
)

var hpiiPattern = regexp.MustCompile(`^\d{16}$`)

// ValidatePerson reports any structural problem with p.
func ValidatePerson(p models.Person) error {
	if p.GivenName == "" {
		return errors.New("given_name is required")
	}
	if p.FamilyName == "" {
		return errors.New("family_name is required")
	}
	if p.HPII != "" && !hpiiPattern.MatchString(p.HPII) {
		return fmt.Errorf("hpii must be 16 digits, got %q", p.HPII)
	}
	return nil
}
