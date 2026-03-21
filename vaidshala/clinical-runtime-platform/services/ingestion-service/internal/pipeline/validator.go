package pipeline

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// clinicalRange defines the valid, critical, and implausible ranges for an observation.
type clinicalRange struct {
	// Plausible range -- values outside are flagged IMPLAUSIBLE
	PlausibleMin float64
	PlausibleMax float64
	// Critical range -- values in critical zone flagged CRITICAL_VALUE
	CriticalLow  float64 // value <= CriticalLow is critical (0 = no low critical)
	CriticalHigh float64 // value >= CriticalHigh is critical (0 = no high critical)
}

// clinicalRanges maps LOINC codes to their clinical range definitions.
var clinicalRanges = map[string]clinicalRange{
	// Fasting glucose (mg/dL)
	"1558-6": {PlausibleMin: 10, PlausibleMax: 600, CriticalLow: 40, CriticalHigh: 400},
	// Random glucose (mg/dL)
	"2345-7": {PlausibleMin: 10, PlausibleMax: 600, CriticalLow: 40, CriticalHigh: 400},
	// Blood glucose (mg/dL)
	"2339-0": {PlausibleMin: 10, PlausibleMax: 600, CriticalLow: 40, CriticalHigh: 400},
	// HbA1c (%)
	"4548-4": {PlausibleMin: 2.0, PlausibleMax: 20.0, CriticalHigh: 14.0},
	// eGFR (mL/min/1.73m2)
	"33914-3": {PlausibleMin: 0, PlausibleMax: 200, CriticalLow: 15},
	// Creatinine (mg/dL)
	"2160-0": {PlausibleMin: 0.1, PlausibleMax: 30.0, CriticalHigh: 10.0},
	// Potassium (mEq/L)
	"2823-3": {PlausibleMin: 1.0, PlausibleMax: 10.0, CriticalLow: 3.0, CriticalHigh: 6.0},
	// Sodium (mEq/L)
	"2951-2": {PlausibleMin: 100, PlausibleMax: 180, CriticalLow: 120, CriticalHigh: 160},
	// Total cholesterol (mg/dL)
	"2093-3": {PlausibleMin: 50, PlausibleMax: 500, CriticalHigh: 400},
	// HDL (mg/dL)
	"2085-9": {PlausibleMin: 5, PlausibleMax: 150},
	// LDL (mg/dL)
	"2089-1": {PlausibleMin: 10, PlausibleMax: 400, CriticalHigh: 300},
	// Triglycerides (mg/dL)
	"2571-8": {PlausibleMin: 10, PlausibleMax: 2000, CriticalHigh: 500},
	// Systolic BP (mmHg)
	"8480-6": {PlausibleMin: 40, PlausibleMax: 300, CriticalLow: 70, CriticalHigh: 180},
	// Diastolic BP (mmHg)
	"8462-4": {PlausibleMin: 20, PlausibleMax: 200, CriticalLow: 40, CriticalHigh: 120},
	// Heart rate (bpm)
	"8867-4": {PlausibleMin: 20, PlausibleMax: 250, CriticalLow: 40, CriticalHigh: 150},
	// SpO2 (%)
	"2708-6": {PlausibleMin: 50, PlausibleMax: 100, CriticalLow: 90},
	// Body temperature (degC)
	"8310-5": {PlausibleMin: 30, PlausibleMax: 45, CriticalLow: 35, CriticalHigh: 40},
	// Body weight (kg)
	"29463-7": {PlausibleMin: 1, PlausibleMax: 500},
	// Body height (cm)
	"8302-2": {PlausibleMin: 30, PlausibleMax: 300},
	// BMI
	"39156-5": {PlausibleMin: 5, PlausibleMax: 80},
	// TSH (mIU/L)
	"3016-3": {PlausibleMin: 0.01, PlausibleMax: 100, CriticalHigh: 50},
	// ALT (U/L)
	"1742-6": {PlausibleMin: 0, PlausibleMax: 5000, CriticalHigh: 1000},
	// AST (U/L)
	"1920-8": {PlausibleMin: 0, PlausibleMax: 5000, CriticalHigh: 1000},
	// Hemoglobin (g/dL)
	"718-7": {PlausibleMin: 2, PlausibleMax: 25, CriticalLow: 7, CriticalHigh: 20},
	// Uric acid (mg/dL)
	"3084-1": {PlausibleMin: 0.5, PlausibleMax: 20, CriticalHigh: 12},
}

// sourceQualityBase assigns a base quality score by source type.
var sourceQualityBase = map[canonical.SourceType]float64{
	canonical.SourceLab:             0.95,
	canonical.SourceEHR:             0.90,
	canonical.SourceABDM:            0.85,
	canonical.SourceDevice:          0.90,
	canonical.SourceWearable:        0.80,
	canonical.SourcePatientReported: 0.70,
	canonical.SourceHPI:             0.75,
}

// DefaultValidator checks clinical ranges, flags critical/implausible values,
// and computes a quality score (0.0-1.0).
type DefaultValidator struct {
	logger *zap.Logger
}

// NewValidator creates a new DefaultValidator.
func NewValidator(logger *zap.Logger) *DefaultValidator {
	return &DefaultValidator{logger: logger}
}

// Validate checks clinical ranges and computes quality score.
// Modifies the observation in place. Returns error only for structural
// issues (missing required fields). Clinical flags are set on the observation,
// not returned as errors.
func (v *DefaultValidator) Validate(ctx context.Context, obs *canonical.CanonicalObservation) error {
	// Structural validation -- required fields
	if obs.PatientID == uuid.Nil {
		return fmt.Errorf("observation missing patient_id")
	}
	if obs.Timestamp.IsZero() {
		return fmt.Errorf("observation missing timestamp")
	}

	// Start with base quality score from source type
	baseQuality, ok := sourceQualityBase[obs.SourceType]
	if !ok {
		baseQuality = 0.70
	}
	obs.QualityScore = baseQuality

	// Clinical range check
	if obs.LOINCCode != "" {
		r, found := clinicalRanges[obs.LOINCCode]
		if found {
			v.applyRangeChecks(obs, r)
		}
	}

	// Deductions for existing flags (from normalizer)
	for _, f := range obs.Flags {
		switch f {
		case canonical.FlagStale:
			obs.QualityScore -= 0.10
		case canonical.FlagUnmappedCode:
			obs.QualityScore -= 0.15
		case canonical.FlagManualEntry:
			obs.QualityScore -= 0.05
		}
	}

	// Clamp quality score to [0.0, 1.0]
	if obs.QualityScore < 0.0 {
		obs.QualityScore = 0.0
	}
	if obs.QualityScore > 1.0 {
		obs.QualityScore = 1.0
	}

	return nil
}

// applyRangeChecks checks the observation value against clinical ranges.
func (v *DefaultValidator) applyRangeChecks(obs *canonical.CanonicalObservation, r clinicalRange) {
	val := obs.Value

	// Check implausible range first (superset of critical)
	if val < r.PlausibleMin || val > r.PlausibleMax {
		obs.Flags = append(obs.Flags, canonical.FlagImplausible)
		obs.QualityScore = 0.10 // Very low quality for implausible
		v.logger.Warn("implausible observation value",
			zap.String("loinc", obs.LOINCCode),
			zap.Float64("value", val),
			zap.Float64("plausible_min", r.PlausibleMin),
			zap.Float64("plausible_max", r.PlausibleMax),
		)
		return
	}

	// Check critical ranges
	isCritical := false
	if r.CriticalLow > 0 && val <= r.CriticalLow {
		isCritical = true
	}
	if r.CriticalHigh > 0 && val >= r.CriticalHigh {
		isCritical = true
	}

	if isCritical {
		obs.Flags = append(obs.Flags, canonical.FlagCriticalValue)
		v.logger.Warn("critical observation value detected",
			zap.String("loinc", obs.LOINCCode),
			zap.Float64("value", val),
			zap.String("patient_id", obs.PatientID.String()),
		)
	}
}
