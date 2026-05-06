package validation

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ValidateObservation reports any structural problem with o.
//
// Rules (mirrors DB observations_value_or_text CHECK + spec §3.6 + §9.10):
//   - ResidentID must be non-zero
//   - Kind must be one of the ObservationKind* enum values
//   - ObservedAt must be non-zero
//   - At least one of Value or ValueText must be set
//   - Per-kind sanity ranges on Value when present:
//       vital   — systolic BP (LOINC 8480-6) 1-300; diastolic BP (LOINC 8462-4) 1-200;
//                 default vital range 0-1000 (broad guard against absurd inputs)
//       lab     — value > 0 (labs are never <= 0; rejected values surface as ValidationStatus elsewhere)
//       weight  — value > 0
//       mobility — value >= 0
//       behavioural — Value is allowed but ValueText is the canonical carrier
func ValidateObservation(o models.Observation) error {
	if o.ResidentID == uuid.Nil {
		return errors.New("resident_id is required")
	}
	if !models.IsValidObservationKind(o.Kind) {
		return fmt.Errorf("invalid kind %q", o.Kind)
	}
	if o.ObservedAt.IsZero() {
		return errors.New("observed_at is required")
	}
	if o.Value == nil && o.ValueText == "" {
		return errors.New("one of value or value_text must be provided")
	}

	if o.Value != nil {
		v := *o.Value
		switch o.Kind {
		case models.ObservationKindVital:
			switch o.LOINCCode {
			case "8480-6": // systolic
				if v < 1 || v > 300 {
					return fmt.Errorf("systolic BP %v out of range [1,300]", v)
				}
			case "8462-4": // diastolic
				if v < 1 || v > 200 {
					return fmt.Errorf("diastolic BP %v out of range [1,200]", v)
				}
			default:
				if v < 0 || v > 1000 {
					return fmt.Errorf("vital value %v out of broad range [0,1000]", v)
				}
			}
		case models.ObservationKindLab:
			if v <= 0 {
				return fmt.Errorf("lab value must be > 0, got %v", v)
			}
		case models.ObservationKindWeight:
			if v <= 0 {
				return fmt.Errorf("weight must be > 0, got %v", v)
			}
		case models.ObservationKindMobility:
			if v < 0 {
				return fmt.Errorf("mobility score must be >= 0, got %v", v)
			}
		}
	}
	return nil
}
