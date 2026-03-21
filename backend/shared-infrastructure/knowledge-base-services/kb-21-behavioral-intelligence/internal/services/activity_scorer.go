package services

import (
	"go.uber.org/zap"
)

const (
	stepsThresholdUnder65 = 4000.0
	stepsThreshold65to75  = 2500.0
	stepsThresholdOver75  = 1500.0
)

// ActivityScorer computes exercise compliance from daily step counts
// using age-adjusted thresholds per M3-VFRP.
type ActivityScorer struct {
	logger *zap.Logger
}

func NewActivityScorer(logger *zap.Logger) *ActivityScorer {
	return &ActivityScorer{logger: logger}
}

// StepThreshold returns the daily step threshold for a given age.
func StepThreshold(age int) float64 {
	switch {
	case age > 75:
		return stepsThresholdOver75
	case age >= 65:
		return stepsThreshold65to75
	default:
		return stepsThresholdUnder65
	}
}

// ScoreDaily computes a compliance score (0.0-1.0) for a single day's steps.
func (a *ActivityScorer) ScoreDaily(steps float64, age int) float64 {
	if steps < 0 {
		a.logger.Warn("negative step count received, clamping to 0", zap.Float64("steps", steps))
		return 0
	}
	threshold := StepThreshold(age)
	if threshold <= 0 {
		return 0
	}
	score := steps / threshold
	if score > 1.0 {
		return 1.0
	}
	if score < 0 {
		return 0
	}
	return score
}

// ScoreRolling7d computes the 7-day rolling compliance score.
// Takes up to 7 daily step values (most recent first).
func (a *ActivityScorer) ScoreRolling7d(dailySteps []float64, age int) float64 {
	if len(dailySteps) == 0 {
		return 0
	}
	// Use at most 7 days
	n := len(dailySteps)
	if n > 7 {
		n = 7
	}
	var sum float64
	for i := 0; i < n; i++ {
		sum += dailySteps[i]
	}
	avg := sum / float64(n)
	return a.ScoreDaily(avg, age)
}
