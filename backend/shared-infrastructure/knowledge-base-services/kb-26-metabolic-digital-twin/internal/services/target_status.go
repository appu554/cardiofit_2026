package services

import (
	"math"
	"time"
)

// TargetStatusInput provides glycaemic measurements for target evaluation.
// CGM TIR is preferred when available and sufficient; HbA1c is the fallback.
type TargetStatusInput struct {
	HbA1c        *float64
	PrevHbA1c    *float64
	HbA1cDate    *time.Time
	PrevHbA1cDate *time.Time
	HbA1cTarget  float64

	CGMAvailable      bool
	CGMSufficientData bool
	CGMTIR            *float64
	CGMReportDate     *time.Time
	TIRTarget         float64
}

// BPTargetStatusInput provides blood-pressure measurements for target evaluation.
type BPTargetStatusInput struct {
	MeanSBP7d *float64
	SBPTarget float64
}

// DomainTargetStatusResult reports whether a clinical domain is at target
// and provides context for the therapeutic-inertia engine.
type DomainTargetStatusResult struct {
	Domain              string
	AtTarget            bool
	CurrentValue        float64
	TargetValue         float64
	FirstUncontrolledAt *time.Time
	DaysUncontrolled    int
	ConsecutiveReadings int
	DataSource          string
	Confidence          string
}

// ComputeGlycaemicTargetStatus evaluates glycaemic control.
// Prefers CGM TIR (HIGH confidence) when available and sufficient;
// falls back to HbA1c (MODERATE confidence).
func ComputeGlycaemicTargetStatus(input TargetStatusInput) DomainTargetStatusResult {
	result := DomainTargetStatusResult{
		Domain: "GLYCAEMIC",
	}

	// Prefer CGM when available and sufficient.
	if input.CGMAvailable && input.CGMSufficientData && input.CGMTIR != nil {
		tir := *input.CGMTIR
		result.CurrentValue = tir
		result.TargetValue = input.TIRTarget
		result.AtTarget = tir >= input.TIRTarget
		result.DataSource = "CGM_TIR"
		result.Confidence = "HIGH"
		result.ConsecutiveReadings = 1
		if input.CGMReportDate != nil && !result.AtTarget {
			t := *input.CGMReportDate
			result.FirstUncontrolledAt = &t
			result.DaysUncontrolled = daysSince(t)
		}
		return result
	}

	// Fallback: HbA1c.
	if input.HbA1c != nil {
		a1c := *input.HbA1c
		result.CurrentValue = a1c
		result.TargetValue = input.HbA1cTarget
		result.AtTarget = a1c <= input.HbA1cTarget
		result.DataSource = "HBA1C"
		result.Confidence = "MODERATE"

		consecutive := 0
		var firstExceedance *time.Time

		// Check previous reading first (chronologically earlier).
		if input.PrevHbA1c != nil && *input.PrevHbA1c > input.HbA1cTarget {
			consecutive++
			if input.PrevHbA1cDate != nil {
				t := *input.PrevHbA1cDate
				firstExceedance = &t
			}
		}
		// Current reading.
		if a1c > input.HbA1cTarget {
			consecutive++
			if firstExceedance == nil && input.HbA1cDate != nil {
				t := *input.HbA1cDate
				firstExceedance = &t
			}
		}

		result.ConsecutiveReadings = consecutive
		if firstExceedance != nil {
			result.FirstUncontrolledAt = firstExceedance
			result.DaysUncontrolled = daysSince(*firstExceedance)
		}
	}

	return result
}

// ComputeHemodynamicTargetStatus evaluates blood-pressure control from
// home BP data (Module 7 metrics).
func ComputeHemodynamicTargetStatus(input BPTargetStatusInput) DomainTargetStatusResult {
	result := DomainTargetStatusResult{
		Domain:     "HEMODYNAMIC",
		TargetValue: input.SBPTarget,
		DataSource: "HOME_BP",
		Confidence: "MODERATE",
	}

	if input.MeanSBP7d != nil {
		sbp := *input.MeanSBP7d
		result.CurrentValue = sbp
		result.AtTarget = sbp <= input.SBPTarget
		result.ConsecutiveReadings = 1
	}

	return result
}

// daysSince returns the number of days between t and now, floored.
func daysSince(t time.Time) int {
	d := time.Since(t).Hours() / 24
	return int(math.Floor(d))
}
