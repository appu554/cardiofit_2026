package validation

import (
	"encoding/json"
	"fmt"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ValidateTarget reports any structural problem with t. The validator
// dispatches to per-Kind validators based on Target.Kind.
func ValidateTarget(t models.Target) error {
	if !models.IsValidTargetKind(t.Kind) {
		return fmt.Errorf("invalid target.kind %q", t.Kind)
	}
	switch t.Kind {
	case models.TargetKindBPThreshold:
		return validateTargetBPThreshold(t.Spec)
	case models.TargetKindCompletionDate:
		return validateTargetCompletionDate(t.Spec)
	case models.TargetKindSymptomResolution:
		return validateTargetSymptomResolution(t.Spec)
	case models.TargetKindHbA1cBand:
		return validateTargetHbA1cBand(t.Spec)
	case models.TargetKindOpen:
		return validateTargetOpen(t.Spec)
	}
	return fmt.Errorf("unhandled target.kind %q (validator not implemented)", t.Kind)
}

func validateTargetBPThreshold(raw json.RawMessage) error {
	var s models.TargetBPThresholdSpec
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("BP_threshold spec unmarshal: %w", err)
	}
	if s.SystolicMax <= 0 || s.SystolicMax > 300 {
		return fmt.Errorf("BP_threshold systolic_max %d out of physiological range (1-300)", s.SystolicMax)
	}
	if s.DiastolicMax <= 0 || s.DiastolicMax > 200 {
		return fmt.Errorf("BP_threshold diastolic_max %d out of physiological range (1-200)", s.DiastolicMax)
	}
	if s.SystolicMax < s.DiastolicMax {
		return fmt.Errorf("BP_threshold systolic_max (%d) must be >= diastolic_max (%d)", s.SystolicMax, s.DiastolicMax)
	}
	return nil
}

func validateTargetCompletionDate(raw json.RawMessage) error {
	var s models.TargetCompletionDateSpec
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("completion_date spec unmarshal: %w", err)
	}
	if s.EndDate.IsZero() {
		return fmt.Errorf("completion_date end_date is required")
	}
	if s.DurationDays < 0 {
		return fmt.Errorf("completion_date duration_days must be >= 0")
	}
	return nil
}

func validateTargetSymptomResolution(raw json.RawMessage) error {
	var s models.TargetSymptomResolutionSpec
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("symptom_resolution spec unmarshal: %w", err)
	}
	if s.TargetSymptom == "" {
		return fmt.Errorf("symptom_resolution target_symptom is required")
	}
	if s.MonitoringWindowDays < 0 {
		return fmt.Errorf("symptom_resolution monitoring_window_days must be >= 0")
	}
	return nil
}

func validateTargetHbA1cBand(raw json.RawMessage) error {
	var s models.TargetHbA1cBandSpec
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("HbA1c_band spec unmarshal: %w", err)
	}
	if s.Min <= 0 || s.Min > 20 {
		return fmt.Errorf("HbA1c_band min %.2f out of physiological range (0-20%%)", s.Min)
	}
	if s.Max <= 0 || s.Max > 20 {
		return fmt.Errorf("HbA1c_band max %.2f out of physiological range (0-20%%)", s.Max)
	}
	if s.Min >= s.Max {
		return fmt.Errorf("HbA1c_band min (%.2f) must be < max (%.2f)", s.Min, s.Max)
	}
	return nil
}

func validateTargetOpen(raw json.RawMessage) error {
	// Open spec is structurally permissive; rationale is optional.
	var s models.TargetOpenSpec
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("open spec unmarshal: %w", err)
	}
	_ = s
	return nil
}
