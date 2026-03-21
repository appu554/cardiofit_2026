package services

import (
	"math"
	"testing"

	"go.uber.org/zap"
)

func TestStepThreshold(t *testing.T) {
	tests := []struct {
		age      int
		expected float64
	}{
		{30, 4000},
		{64, 4000},
		{65, 2500},
		{70, 2500},
		{75, 2500},
		{76, 1500},
		{85, 1500},
	}
	for _, tt := range tests {
		got := StepThreshold(tt.age)
		if got != tt.expected {
			t.Errorf("StepThreshold(%d) = %f, want %f", tt.age, got, tt.expected)
		}
	}
}

func TestActivityScorer_ScoreDaily(t *testing.T) {
	scorer := NewActivityScorer(zap.NewNop())

	// 2000 steps for age 50 (threshold 4000) → 0.5
	score := scorer.ScoreDaily(2000, 50)
	if math.Abs(score-0.5) > 0.001 {
		t.Errorf("ScoreDaily(2000, 50) = %f, want 0.5", score)
	}

	// 5000 steps for age 50 → clamped to 1.0
	score = scorer.ScoreDaily(5000, 50)
	if score != 1.0 {
		t.Errorf("ScoreDaily(5000, 50) = %f, want 1.0", score)
	}

	// 0 steps → 0.0
	score = scorer.ScoreDaily(0, 50)
	if score != 0.0 {
		t.Errorf("ScoreDaily(0, 50) = %f, want 0.0", score)
	}

	// 2500 steps for age 70 (threshold 2500) → 1.0
	score = scorer.ScoreDaily(2500, 70)
	if score != 1.0 {
		t.Errorf("ScoreDaily(2500, 70) = %f, want 1.0", score)
	}
}

func TestActivityScorer_ScoreRolling7d(t *testing.T) {
	scorer := NewActivityScorer(zap.NewNop())

	// 7 days of data, age 50 (threshold 4000)
	steps := []float64{3000, 4000, 5000, 2000, 4000, 3000, 3000}
	// avg = 24000/7 ≈ 3428.57, score = 3428.57/4000 ≈ 0.857
	score := scorer.ScoreRolling7d(steps, 50)
	if score < 0.85 || score > 0.87 {
		t.Errorf("ScoreRolling7d = %f, want ~0.857", score)
	}

	// Empty slice → 0
	score = scorer.ScoreRolling7d([]float64{}, 50)
	if score != 0 {
		t.Errorf("ScoreRolling7d(empty) = %f, want 0", score)
	}

	// More than 7 entries → uses first 7 only
	longSteps := []float64{4000, 4000, 4000, 4000, 4000, 4000, 4000, 10000, 10000}
	score = scorer.ScoreRolling7d(longSteps, 50)
	if score != 1.0 {
		t.Errorf("ScoreRolling7d(7x4000 + extras) = %f, want 1.0", score)
	}
}
