package services

import (
	"go.uber.org/zap"
)

const (
	defaultProteinMinGKg = 0.8
	defaultProteinMaxGKg = 1.2
)

// ProteinTracker computes 7-day rolling protein adequacy against M3-PRP targets.
type ProteinTracker struct {
	minTargetGKg float64
	maxTargetGKg float64
	logger       *zap.Logger
}

func NewProteinTracker(logger *zap.Logger) *ProteinTracker {
	return &ProteinTracker{
		minTargetGKg: defaultProteinMinGKg,
		maxTargetGKg: defaultProteinMaxGKg,
		logger:       logger,
	}
}

// NewProteinTrackerWithTargets creates a tracker with custom target range.
func NewProteinTrackerWithTargets(minGKg, maxGKg float64, logger *zap.Logger) *ProteinTracker {
	return &ProteinTracker{
		minTargetGKg: minGKg,
		maxTargetGKg: maxGKg,
		logger:       logger,
	}
}

// TargetMid returns the midpoint of the protein target range in g/kg/day.
func (p *ProteinTracker) TargetMid() float64 {
	return (p.minTargetGKg + p.maxTargetGKg) / 2.0
}

// ComputeAdequacy computes protein adequacy (0.0-1.0) from a slice of
// daily protein intakes (grams) and the patient's weight (kg).
// Uses up to 7 most recent entries for the rolling average.
func (p *ProteinTracker) ComputeAdequacy(dailyProteinG []float64, weightKg float64) float64 {
	if len(dailyProteinG) == 0 || weightKg <= 0 {
		return 0
	}

	n := len(dailyProteinG)
	if n > 7 {
		n = 7
	}

	var sum float64
	for i := 0; i < n; i++ {
		v := dailyProteinG[i]
		if v < 0 {
			p.logger.Warn("negative protein intake received, treating as 0", zap.Float64("protein_g", v))
			continue
		}
		sum += v
	}
	avgIntake := sum / float64(n)

	targetDaily := p.TargetMid() * weightKg // g/day
	if targetDaily <= 0 {
		return 0
	}

	adequacy := avgIntake / targetDaily
	if adequacy > 1.0 {
		return 1.0
	}
	if adequacy < 0 {
		return 0
	}
	return adequacy
}
