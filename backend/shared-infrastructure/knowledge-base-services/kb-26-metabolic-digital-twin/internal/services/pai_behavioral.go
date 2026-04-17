package services

import (
	"math"

	"kb-26-metabolic-digital-twin/internal/models"
)

// ComputeBehavioralScore computes the behavioral dimension of the Patient
// Acuity Index from patient engagement and measurement frequency signals.
//
// Two sub-components are computed independently, then combined via a
// compound rule:
//
//	A. Engagement score — derived from EngagementComposite (0-1 scale)
//	B. Measurement frequency score — cessation and frequency drop signals
//	C. Compound rule — if both are high, return compound ceiling; else max
func ComputeBehavioralScore(input models.PAIDimensionInput, cfg *PAIConfig) float64 {
	engScore := engagementScore(input)
	measScore := measurementScore(input, cfg)

	// Compound rule: both engagement and measurement signals firing
	if engScore >= 80 && measScore >= 70 {
		return cfg.BehavioralCompoundBoth
	}

	return math.Max(engScore, measScore)
}

// engagementScore maps the EngagementComposite (0-1) to a 0-100 acuity score.
// Lower engagement → higher acuity. Nil composite → 0 (assume engaged).
func engagementScore(input models.PAIDimensionInput) float64 {
	if input.EngagementComposite == nil {
		return 0
	}
	c := *input.EngagementComposite

	switch {
	case c < 0.3:
		// Disengaged
		return 80
	case c < 0.5:
		// Declining
		return scaleLinear(c, 0.3, 0.5, 80, 50)
	case c < 0.7:
		// Active
		return scaleLinear(c, 0.5, 0.7, 50, 20)
	default:
		// Engaged
		return scaleLinear(c, 0.7, 1.0, 20, 0)
	}
}

// measurementScore detects cessation and frequency drops in self-monitoring.
func measurementScore(input models.PAIDimensionInput, cfg *PAIConfig) float64 {
	// Cessation: no reading for N+ days
	if input.DaysSinceLastBPReading >= cfg.BehavioralCessationDays {
		return 70
	}

	// Frequency drop thresholds
	if input.MeasurementFreqDrop > cfg.BehavioralReducedThreshold {
		return 50
	}
	if input.MeasurementFreqDrop >= cfg.BehavioralSlightlyReduced {
		return 25
	}

	return 0
}
