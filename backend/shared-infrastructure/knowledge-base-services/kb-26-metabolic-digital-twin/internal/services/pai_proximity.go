package services

import (
	"math"

	"kb-26-metabolic-digital-twin/internal/models"
)

// proximityMetric defines a single lab/vital proximity dimension with
// danger threshold, warning start, direction, and score mapping.
type proximityMetric struct {
	DangerThreshold float64
	WarningStart    float64
	ScoreAtDanger   float64
	ScoreAtWarning  float64
	Direction       string // "below" = lower is dangerous, "above" = higher is dangerous
}

// ComputeProximityScore evaluates how close the patient's current values
// are to clinically dangerous thresholds. Uses exponential scaling so
// that scores rise steeply as values approach danger boundaries.
//
// Multi-metric compounding: final = max(scores) + 20% * sum(others), cap 100.
func ComputeProximityScore(input models.PAIDimensionInput, cfg *PAIConfig) float64 {
	exponent := cfg.ProximityExponent
	if exponent <= 0 {
		exponent = 2.0
	}

	var scores []float64

	// ── Standard lab/vital metrics ──────────────────────────────────────

	metrics := []struct {
		value  *float64
		metric proximityMetric
	}{
		{input.CurrentEGFR, proximityMetric{30, 45, 100, 50, "below"}},
		{input.CurrentHbA1c, proximityMetric{10.0, 8.0, 80, 40, "above"}},
		{input.CurrentSBP, proximityMetric{180, 160, 100, 60, "above"}},
		{input.CurrentPotassium, proximityMetric{6.0, 5.5, 100, 50, "above"}},
		{input.CurrentTBRL2Pct, proximityMetric{5.0, 2.0, 90, 40, "above"}},
	}

	for _, m := range metrics {
		if m.value == nil {
			continue
		}
		s := scoreMetric(*m.value, m.metric, exponent)
		if s > 0 {
			scores = append(scores, s)
		}
	}

	// ── Acute weight gain (HF patients only) ────────────────────────────

	if input.CurrentWeight != nil && input.PreviousWeight72h != nil && input.CKMStage == "4c" {
		gain := *input.CurrentWeight - *input.PreviousWeight72h
		if gain > 0 {
			wm := proximityMetric{3.0, 2.0, 85, 40, "above"}
			s := scoreMetric(gain, wm, exponent)
			if s > 0 {
				scores = append(scores, s)
			}
		}
	}

	// ── Compound scoring ────────────────────────────────────────────────

	if len(scores) == 0 {
		return 0
	}

	// Find max and sum of secondary contributions
	maxScore := 0.0
	sumSecondary := 0.0
	for _, s := range scores {
		if s > maxScore {
			sumSecondary += maxScore // demote old max to secondary
			maxScore = s
		} else {
			sumSecondary += s
		}
	}

	final := maxScore + 0.20*sumSecondary
	return math.Min(100, math.Max(0, final))
}

// scoreMetric computes the proximity score for a single metric using
// exponential scaling within the warning-to-danger zone.
func scoreMetric(value float64, m proximityMetric, exponent float64) float64 {
	if m.Direction == "below" {
		// Lower is more dangerous (e.g., eGFR)
		if value <= m.DangerThreshold {
			return m.ScoreAtDanger
		}
		if value >= m.WarningStart {
			return 0
		}
		// In warning-to-danger zone: fraction of distance covered
		fraction := (m.WarningStart - value) / (m.WarningStart - m.DangerThreshold)
		scaled := math.Pow(fraction, exponent)
		return m.ScoreAtWarning + scaled*(m.ScoreAtDanger-m.ScoreAtWarning)
	}

	// Direction == "above": higher is more dangerous
	if value >= m.DangerThreshold {
		return m.ScoreAtDanger
	}
	if value <= m.WarningStart {
		return 0
	}
	fraction := (value - m.WarningStart) / (m.DangerThreshold - m.WarningStart)
	scaled := math.Pow(fraction, exponent)
	return m.ScoreAtWarning + scaled*(m.ScoreAtDanger-m.ScoreAtWarning)
}
