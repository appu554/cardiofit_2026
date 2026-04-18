package services

import (
	"math"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"

	"go.uber.org/zap"
)

// Use the ValidationState constants from models package.
var (
	ValidationConfirmed            = models.ValidationConfirmed
	ValidationUnconfirmed          = models.ValidationUnconfirmed
	ValidationAwaitingConfirmation = models.ValidationAwaitingConfirmation
	ValidationUnconfirmedCritical  = models.ValidationUnconfirmedCritical
	ValidationRefuted              = models.ValidationRefuted
)

// ValidationManager provides pure-function validation logic for weight deviation
// readings. It has no database dependency — callers persist results.
type ValidationManager struct {
	log *zap.Logger
}

// NewValidationManager creates a ValidationManager.
func NewValidationManager(log *zap.Logger) *ValidationManager {
	return &ValidationManager{log: log}
}

// CheckWeightValidation determines the validation state for a weight deviation.
//
// Parameters:
//   - currentValue:      the new weight reading (kg)
//   - baselineMedian:    patient's rolling baseline median (kg)
//   - deviation:         absolute deviation from baseline (kg)
//   - severity:          MODERATE, HIGH, or CRITICAL
//   - measurementHour:   hour of day (0-23) of the new reading
//   - usualHour:         patient's usual measurement hour (median)
//   - hasPriorDeviation: whether a deviating reading exists in the last 48h
//   - isCriticalHF:      CKM 4c patient with deviation exceeding CRITICAL threshold
func (m *ValidationManager) CheckWeightValidation(
	currentValue float64,
	baselineMedian float64,
	deviation float64,
	severity string,
	measurementHour int,
	usualHour int,
	hasPriorDeviation bool,
	isCriticalHF bool,
) (models.ValidationState, string) {
	// Rule 1: Time-of-day consistency — flag readings taken at unusual times.
	hourDiff := abs(measurementHour - usualHour)
	if hourDiff > 12 {
		hourDiff = 24 - hourDiff // wrap around midnight
	}
	if hourDiff > 2 {
		m.log.Info("weight validation: time-of-day inconsistent",
			zap.Int("measurement_hour", measurementHour),
			zap.Int("usual_hour", usualHour),
			zap.Int("hour_diff", hourDiff),
		)
		return models.ValidationUnconfirmed, "TIME_OF_DAY_INCONSISTENT"
	}

	// Rule 2: CRITICAL in heart-failure bypasses waiting — escalate immediately
	// but note the reading is unconfirmed.
	if isCriticalHF {
		m.log.Warn("weight validation: critical HF bypass — escalating unconfirmed",
			zap.Float64("deviation_kg", deviation),
			zap.String("severity", severity),
		)
		return models.ValidationUnconfirmedCritical, "CRITICAL_HF_BYPASS"
	}

	// Rule 3: First deviation without prior confirmation — require a second reading.
	if !hasPriorDeviation && (severity == "HIGH" || severity == "MODERATE") {
		m.log.Info("weight validation: first deviation, awaiting confirmation",
			zap.Float64("deviation_kg", deviation),
			zap.String("severity", severity),
		)
		return models.ValidationAwaitingConfirmation, "FIRST_DEVIATION_NEEDS_CONFIRMATION"
	}

	return models.ValidationConfirmed, ""
}

// ProcessConfirmation checks whether a confirmation reading confirms or refutes
// the original pending deviation.
//
//   - <=20% difference: CONFIRMED (readings agree)
//   - >50% difference:  REFUTED (likely artefact)
//   - 20-50%:           AWAITING_CONFIRMATION (still uncertain)
func (m *ValidationManager) ProcessConfirmation(
	originalDeviation float64,
	confirmationDeviation float64,
) models.ValidationState {
	if originalDeviation == 0 {
		return models.ValidationRefuted
	}
	diffPct := math.Abs(confirmationDeviation-originalDeviation) / math.Abs(originalDeviation) * 100

	if diffPct <= 20 {
		return models.ValidationConfirmed
	}
	if diffPct > 50 {
		return models.ValidationRefuted
	}
	return models.ValidationAwaitingConfirmation // between 20-50% — still uncertain
}

// IsExpired checks if a pending validation has expired (>24h without confirmation).
func (m *ValidationManager) IsExpired(expiresAt time.Time, now time.Time) bool {
	return now.After(expiresAt)
}

// CreatePendingValidation builds a PendingValidation record. The caller is
// responsible for persisting it to the database.
func (m *ValidationManager) CreatePendingValidation(
	patientID, vitalType string,
	value, deviation float64,
	readingTime time.Time,
) models.PendingValidation {
	return models.PendingValidation{
		PatientID:           patientID,
		VitalSignType:       vitalType,
		OriginalValue:       value,
		OriginalDeviation:   deviation,
		OriginalReadingTime: readingTime,
		ExpiresAt:           readingTime.Add(24 * time.Hour),
	}
}

// abs returns the absolute value of an integer.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
