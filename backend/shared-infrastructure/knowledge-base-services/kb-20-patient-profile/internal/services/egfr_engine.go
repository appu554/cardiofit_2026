package services

import (
	"fmt"
	"math"
	"sort"

	"kb-patient-profile/internal/models"
)

// EGFREngine computes eGFR using CKD-EPI 2021, determines CKD staging,
// and classifies eGFR trajectory from serial measurements.
type EGFREngine struct{}

// NewEGFREngine creates an eGFR computation engine.
func NewEGFREngine() *EGFREngine {
	return &EGFREngine{}
}

// ComputeEGFR calculates eGFR using the CKD-EPI 2021 (race-free) equation.
// eGFR = 142 × min(Scr/κ, 1)^α × max(Scr/κ, 1)^-1.200 × 0.9938^age [× 1.012 if female]
func (e *EGFREngine) ComputeEGFR(creatinine float64, age int, sex string) float64 {
	var kappa, alpha float64
	var sexMultiplier float64

	if sex == "F" {
		kappa = 0.7
		alpha = -0.241
		sexMultiplier = 1.012
	} else {
		kappa = 0.9
		alpha = -0.302
		sexMultiplier = 1.0
	}

	scrOverKappa := creatinine / kappa

	minTerm := math.Min(scrOverKappa, 1.0)
	maxTerm := math.Max(scrOverKappa, 1.0)

	egfr := 142.0 *
		math.Pow(minTerm, alpha) *
		math.Pow(maxTerm, -1.200) *
		math.Pow(0.9938, float64(age)) *
		sexMultiplier

	return math.Round(egfr*100) / 100
}

// CKDStageFromEGFR returns the CKD stage based on eGFR value.
func (e *EGFREngine) CKDStageFromEGFR(egfr float64) string {
	switch {
	case egfr >= 90:
		return models.CKDG1
	case egfr >= 60:
		return models.CKDG2
	case egfr >= 45:
		return models.CKDG3a
	case egfr >= 30:
		return models.CKDG3b
	case egfr >= 15:
		return models.CKDG4
	default:
		return models.CKDG5
	}
}

// IsCKDConfirmed checks whether CKD can be auto-derived from two eGFR measurements
// ≥90 days apart, both below 60 (KDIGO criteria).
func (e *EGFREngine) IsCKDConfirmed(egfrEntries []models.LabEntry) (bool, bool) {
	var ckdReadings []models.LabEntry
	for _, entry := range egfrEntries {
		val, _ := entry.Value.Float64()
		if val < 60 && entry.ValidationStatus == models.ValidationAccepted {
			ckdReadings = append(ckdReadings, entry)
		}
	}

	if len(ckdReadings) < 1 {
		return false, false // No CKD evidence
	}

	if len(ckdReadings) < 2 {
		return true, false // SUSPECTED — single reading < 60
	}

	// Sort by date
	sort.Slice(ckdReadings, func(i, j int) bool {
		return ckdReadings[i].MeasuredAt.Before(ckdReadings[j].MeasuredAt)
	})

	// Check if earliest and latest are ≥90 days apart
	earliest := ckdReadings[0].MeasuredAt
	latest := ckdReadings[len(ckdReadings)-1].MeasuredAt
	daysBetween := latest.Sub(earliest).Hours() / 24

	if daysBetween >= 90 {
		return true, true // CONFIRMED — two readings ≥90 days apart
	}

	return true, false // SUSPECTED — readings too close together
}

// Trajectory trend constants
const (
	TrendStable           = "STABLE"
	TrendSlowDecline      = "SLOW_DECLINE"
	TrendRapidDecline     = "RAPID_DECLINE"
	TrendInsufficientData = "INSUFFICIENT_DATA"
	TrendImproving        = "IMPROVING"
)

// ClassifyTrajectory determines the eGFR trend from serial measurements.
// Requires ≥3 data points for regression; otherwise returns INSUFFICIENT_DATA.
func (e *EGFREngine) ClassifyTrajectory(points []models.EGFRTrajectoryPoint) (string, *float64) {
	if len(points) < 3 {
		return TrendInsufficientData, nil
	}

	// Sort by date
	sort.Slice(points, func(i, j int) bool {
		return points[i].MeasuredAt.Before(points[j].MeasuredAt)
	})

	// Simple OLS linear regression: eGFR vs time-in-years
	baseTime := points[0].MeasuredAt
	n := float64(len(points))
	var sumX, sumY, sumXY, sumX2 float64

	for _, p := range points {
		x := p.MeasuredAt.Sub(baseTime).Hours() / (24 * 365.25)
		y := p.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return TrendStable, nil
	}

	slope := (n*sumXY - sumX*sumY) / denominator
	annualChange := math.Round(slope*100) / 100

	var trend string
	switch {
	case slope > 1:
		trend = TrendImproving
	case slope >= -3:
		trend = TrendStable
	case slope >= -5:
		trend = TrendSlowDecline
	default:
		trend = TrendRapidDecline
	}

	return trend, &annualChange
}

// DetectThresholdCrossings checks which medication-relevant eGFR boundaries
// were crossed between oldEGFR and newEGFR (F-03 RED).
func (e *EGFREngine) DetectThresholdCrossings(oldEGFR, newEGFR float64) []models.MedicationThreshold {
	var crossed []models.MedicationThreshold
	for _, t := range models.MedicationThresholds {
		oldAbove := oldEGFR >= t.EGFRBoundary
		newBelow := newEGFR < t.EGFRBoundary
		newAbove := newEGFR >= t.EGFRBoundary
		oldBelow := oldEGFR < t.EGFRBoundary

		if (oldAbove && newBelow) || (oldBelow && newAbove) {
			crossed = append(crossed, t)
		}
	}
	return crossed
}

// CheckMedicationAlerts returns safety overrides for medications based on current eGFR.
func (e *EGFREngine) CheckMedicationAlerts(egfr float64, medications []models.MedicationState) []models.SafetyOverride {
	var overrides []models.SafetyOverride
	stage := e.CKDStageFromEGFR(egfr)

	for _, med := range medications {
		for _, drugClass := range med.EffectiveDrugClasses() {
			for _, t := range models.MedicationThresholds {
				if t.AffectedDrugClass == drugClass && egfr < t.EGFRBoundary {
					override := models.SafetyOverride{
						DrugClass:      drugClass,
						AlertType:      "RENAL_DOSE_ADJUSTMENT",
						Severity:       severityForStage(stage),
						Message:        fmt.Sprintf("%s: %s at eGFR %.0f (%s)", drugClass, t.RequiredAction, egfr, stage),
						RequiredAction: t.RequiredAction,
					}

					// Check if current dose exceeds max
					if t.MaxDoseMg != nil {
						doseMg, _ := med.DoseMg.Float64()
						dailyDose := estimateDailyDose(doseMg, med.Frequency)
						if dailyDose > *t.MaxDoseMg {
							override.Severity = "RED"
							override.Message = fmt.Sprintf("%s dose %.0fmg/day exceeds %s maximum %.0fmg at eGFR %.0f (%s)",
								drugClass, dailyDose, stage, *t.MaxDoseMg, egfr, stage)
						}
					}

					overrides = append(overrides, override)
				}
			}
		}
	}
	return overrides
}

func severityForStage(stage string) string {
	switch stage {
	case models.CKDG4, models.CKDG5:
		return "RED"
	case models.CKDG3b:
		return "AMBER"
	default:
		return "YELLOW"
	}
}

func estimateDailyDose(doseMg float64, frequency string) float64 {
	switch frequency {
	case "BD", "BID", "twice daily":
		return doseMg * 2
	case "TDS", "TID", "three times daily":
		return doseMg * 3
	case "QID", "four times daily":
		return doseMg * 4
	default:
		return doseMg // assume OD
	}
}
