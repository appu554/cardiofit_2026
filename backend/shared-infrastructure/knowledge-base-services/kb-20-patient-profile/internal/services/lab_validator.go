package services

import (
	"fmt"

	"kb-patient-profile/internal/models"
)

// LabValidator implements Finding F-05 (RED): physiological plausibility
// ranges for all tracked lab values. Values outside plausibility are REJECTED.
// Values at physiological extremes but within range are FLAGGED.
type LabValidator struct{}

// NewLabValidator creates a lab validator.
func NewLabValidator() *LabValidator {
	return &LabValidator{}
}

// PlausibilityRange defines the acceptable and flagging boundaries for a lab type.
type PlausibilityRange struct {
	Min         float64
	Max         float64
	FlagLow     float64
	FlagHigh    float64
	Unit        string
}

// plausibilityRanges defines physiological plausibility for each lab type.
// Values from KB-20 spec review Finding F-05.
var plausibilityRanges = map[string]PlausibilityRange{
	models.LabTypeCreatinine: {
		Min: 0.2, Max: 20.0,
		FlagLow: 0.3, FlagHigh: 10.0,
		Unit: "mg/dL",
	},
	models.LabTypeEGFR: {
		Min: 0, Max: 200,
		FlagLow: 5, FlagHigh: 150,
		Unit: "mL/min/1.73m²",
	},
	models.LabTypeFBG: {
		Min: 30, Max: 600,
		FlagLow: 40, FlagHigh: 500,
		Unit: "mg/dL",
	},
	models.LabTypeHbA1c: {
		Min: 3.0, Max: 18.0,
		FlagLow: 3.5, FlagHigh: 15.0,
		Unit: "%",
	},
	models.LabTypeSBP: {
		Min: 60, Max: 280,
		FlagLow: 70, FlagHigh: 250,
		Unit: "mmHg",
	},
	models.LabTypeDBP: {
		Min: 30, Max: 180,
		FlagLow: 40, FlagHigh: 150,
		Unit: "mmHg",
	},
	models.LabTypePotassium: {
		Min: 1.5, Max: 9.0,
		FlagLow: 2.5, FlagHigh: 6.5,
		Unit: "mEq/L",
	},
	models.LabTypeTotalCholesterol: {
		Min: 50, Max: 600,
		FlagLow: 80, FlagHigh: 400,
		Unit: "mg/dL",
	},
}

// ValidationResult holds the outcome of lab plausibility validation.
type ValidationResult struct {
	Status     string // ACCEPTED, FLAGGED, REJECTED
	FlagReason string
}

// Validate checks a lab value against plausibility ranges.
func (lv *LabValidator) Validate(labType string, value float64) ValidationResult {
	r, exists := plausibilityRanges[labType]
	if !exists {
		// Unknown lab types are accepted without validation
		return ValidationResult{Status: models.ValidationAccepted}
	}

	// Outside plausibility → REJECTED
	if value < r.Min || value > r.Max {
		return ValidationResult{
			Status:     models.ValidationRejected,
			FlagReason: fmt.Sprintf("IMPLAUSIBLE_VALUE: %s %.2f %s outside range [%.1f–%.1f]", labType, value, r.Unit, r.Min, r.Max),
		}
	}

	// At physiological extremes → FLAGGED
	if value < r.FlagLow {
		return ValidationResult{
			Status:     models.ValidationFlagged,
			FlagReason: fmt.Sprintf("EXTREME_LOW: %s %.2f %s is critically low", labType, value, r.Unit),
		}
	}
	if value > r.FlagHigh {
		return ValidationResult{
			Status:     models.ValidationFlagged,
			FlagReason: fmt.Sprintf("EXTREME_HIGH: %s %.2f %s is critically high", labType, value, r.Unit),
		}
	}

	return ValidationResult{Status: models.ValidationAccepted}
}

// ValidateBPPair cross-validates that SBP > DBP.
func (lv *LabValidator) ValidateBPPair(sbp, dbp float64) *ValidationResult {
	if sbp <= dbp {
		return &ValidationResult{
			Status:     models.ValidationRejected,
			FlagReason: fmt.Sprintf("INVALID_BP: SBP (%.0f) must be > DBP (%.0f)", sbp, dbp),
		}
	}
	return nil
}
