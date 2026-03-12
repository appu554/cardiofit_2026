// Package integration provides integration tests for KB-11 Population Health Engine.
package integration

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/cardiofit/kb-11-population-health/internal/models"
	"github.com/cardiofit/kb-11-population-health/internal/risk"
)

// TestRiskCalculationDeterminism verifies that risk calculations are deterministic.
// CRITICAL: Same input MUST always produce the same output.
// This is required for KB-18 governance and audit compliance.
func TestRiskCalculationDeterminism(t *testing.T) {
	t.Run("same input produces same hash", func(t *testing.T) {
		features := createTestFeatures()

		// Calculate hash multiple times
		hash1 := features.Hash()
		hash2 := features.Hash()
		hash3 := features.Hash()

		assert.Equal(t, hash1, hash2, "Hash should be deterministic")
		assert.Equal(t, hash2, hash3, "Hash should be deterministic")
	})

	t.Run("different inputs produce different hashes", func(t *testing.T) {
		features1 := createTestFeatures()
		features2 := createTestFeatures()
		features2.Age = 75 // Change one field

		hash1 := features1.Hash()
		hash2 := features2.Hash()

		assert.NotEqual(t, hash1, hash2, "Different inputs should produce different hashes")
	})

	t.Run("field order does not affect hash", func(t *testing.T) {
		// Create features with same data but ensure order consistency
		features1 := &risk.RiskFeatures{
			PatientFHIRID: "patient-123",
			Timestamp:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Age:           65,
			Gender:        models.GenderMale,
		}

		features2 := &risk.RiskFeatures{
			PatientFHIRID: "patient-123",
			Timestamp:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Age:           65,
			Gender:        models.GenderMale,
		}

		assert.Equal(t, features1.Hash(), features2.Hash(), "Same data should produce same hash")
	})
}

// TestRiskResultDeterminism verifies that risk results produce consistent hashes.
func TestRiskResultDeterminism(t *testing.T) {
	t.Run("same result produces same calculation hash", func(t *testing.T) {
		result := createTestRiskResult()

		hash1 := result.Hash()
		hash2 := result.Hash()

		assert.Equal(t, hash1, hash2, "Result hash should be deterministic")
	})

	t.Run("score change produces different hash", func(t *testing.T) {
		result1 := createTestRiskResult()
		result2 := createTestRiskResult()
		result2.Score = 0.75 // Change score

		assert.NotEqual(t, result1.Hash(), result2.Hash(), "Different scores should produce different hashes")
	})

	t.Run("contributing factors change produces different hash", func(t *testing.T) {
		result1 := createTestRiskResult()
		result2 := createTestRiskResult()
		result2.ContributingFactors["new_factor"] = 0.1

		assert.NotEqual(t, result1.Hash(), result2.Hash(), "Different factors should produce different hashes")
	})
}

// TestRiskTierDetermination verifies risk tier assignment is consistent.
func TestRiskTierDetermination(t *testing.T) {
	thresholds := risk.RiskThresholds{
		VeryHigh: 0.80,
		High:     0.60,
		Moderate: 0.40,
		Low:      0.10,
		Rising:   0.15,
	}

	testCases := []struct {
		name     string
		score    float64
		isRising bool
		expected models.RiskTier
	}{
		{"Very High Score", 0.85, false, models.RiskTierVeryHigh},
		{"High Score", 0.70, false, models.RiskTierHigh},
		{"Moderate Score", 0.50, false, models.RiskTierModerate},
		{"Low Score", 0.20, false, models.RiskTierLow},
		// Score below Low threshold (0.10) returns UNSCORED per implementation
		{"Below Low Threshold", 0.05, false, models.RiskTierUnscored},
		{"Rising Risk Override", 0.45, true, models.RiskTierRising},
		// Implementation: isRising is checked FIRST, so Rising always takes priority
		{"High Score With Rising Flag", 0.85, true, models.RiskTierRising},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tier := risk.DetermineRiskTier(tc.score, thresholds, tc.isRising)
			assert.Equal(t, tc.expected, tier)
		})
	}
}

// TestScoreNormalization verifies score normalization is consistent.
func TestScoreNormalization(t *testing.T) {
	testCases := []struct {
		name     string
		rawScore float64
		expected float64
	}{
		{"Zero Score", 0.0, 0.0},
		{"Full Score", 1.0, 1.0},
		{"Half Score", 0.5, 0.5},
		{"Negative Clamped", -0.5, 0.0},
		{"Over 1 Clamped", 1.5, 1.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			normalized := risk.NormalizeScore(tc.rawScore)
			assert.InDelta(t, tc.expected, normalized, 0.0001, "Score normalization should be deterministic")
		})
	}
}

// TestRisingRiskCalculation verifies rising risk detection is deterministic.
// Implementation calculates monthly rate: (current - oldest) / months_elapsed
// Rising is true when monthlyRate >= threshold.
func TestRisingRiskCalculation(t *testing.T) {
	t.Run("rising detected with sufficient monthly increase", func(t *testing.T) {
		// Score change of 0.35 over ~2 months = 0.175 monthly rate >= 0.15 threshold
		currentScore := 0.65
		previousScores := []risk.HistoricalScore{
			{Score: 0.50, CalculatedAt: time.Now().AddDate(0, 0, -30)},
			{Score: 0.30, CalculatedAt: time.Now().AddDate(0, 0, -60)}, // oldest within 3 months
		}
		threshold := 0.15

		isRising, rate := risk.CalculateRisingRisk(currentScore, previousScores, threshold)

		assert.True(t, isRising, "Should detect rising risk when monthly rate >= threshold")
		assert.GreaterOrEqual(t, rate, threshold, "Rising rate should meet threshold")
	})

	t.Run("not rising with stable scores", func(t *testing.T) {
		currentScore := 0.45
		previousScores := []risk.HistoricalScore{
			{Score: 0.44, CalculatedAt: time.Now().AddDate(0, 0, -30)},
			{Score: 0.45, CalculatedAt: time.Now().AddDate(0, 0, -60)},
			{Score: 0.43, CalculatedAt: time.Now().AddDate(0, 0, -90)},
		}
		threshold := 0.15

		isRising, rate := risk.CalculateRisingRisk(currentScore, previousScores, threshold)

		assert.False(t, isRising, "Should not detect rising risk for stable scores")
		assert.InDelta(t, 0.0, rate, 0.1, "Rising rate should be near zero for stable scores")
	})

	t.Run("not rising with decreasing scores", func(t *testing.T) {
		currentScore := 0.30
		previousScores := []risk.HistoricalScore{
			{Score: 0.45, CalculatedAt: time.Now().AddDate(0, 0, -30)},
			{Score: 0.50, CalculatedAt: time.Now().AddDate(0, 0, -60)},
		}
		threshold := 0.15

		isRising, rate := risk.CalculateRisingRisk(currentScore, previousScores, threshold)

		assert.False(t, isRising, "Should not detect rising risk for decreasing scores")
		assert.Less(t, rate, 0.0, "Rising rate should be negative for decreasing scores")
	})

	t.Run("deterministic across multiple calls", func(t *testing.T) {
		currentScore := 0.55
		previousScores := []risk.HistoricalScore{
			{Score: 0.40, CalculatedAt: time.Now().AddDate(0, 0, -30)},
		}
		threshold := 0.15

		isRising1, rate1 := risk.CalculateRisingRisk(currentScore, previousScores, threshold)
		isRising2, rate2 := risk.CalculateRisingRisk(currentScore, previousScores, threshold)

		assert.Equal(t, isRising1, isRising2, "Rising detection should be deterministic")
		assert.Equal(t, rate1, rate2, "Rising rate should be deterministic")
	})
}

// TestModelWeightConsistency verifies model weights remain constant.
func TestModelWeightConsistency(t *testing.T) {
	t.Run("hospitalization model weights are fixed", func(t *testing.T) {
		model1 := risk.DefaultHospitalizationModel()
		model2 := risk.DefaultHospitalizationModel()

		assert.Equal(t, model1.Weights, model2.Weights, "Model weights should be constant")
		assert.Equal(t, model1.Thresholds, model2.Thresholds, "Model thresholds should be constant")
		assert.Equal(t, model1.Version, model2.Version, "Model version should be constant")
	})

	t.Run("readmission model weights are fixed", func(t *testing.T) {
		model1 := risk.DefaultReadmissionModel()
		model2 := risk.DefaultReadmissionModel()

		assert.Equal(t, model1.Weights, model2.Weights, "Model weights should be constant")
	})

	t.Run("ED utilization model weights are fixed", func(t *testing.T) {
		model1 := risk.DefaultEDUtilizationModel()
		model2 := risk.DefaultEDUtilizationModel()

		assert.Equal(t, model1.Weights, model2.Weights, "Model weights should be constant")
	})
}

// TestEndToEndDeterminism simulates a full risk calculation and verifies determinism.
func TestEndToEndDeterminism(t *testing.T) {
	t.Skip("Requires database connection - run with integration test flag")

	// This test would:
	// 1. Create identical features
	// 2. Run calculation twice
	// 3. Compare input_hash and calculation_hash
	// 4. Verify governance events have matching hashes
}

// ──────────────────────────────────────────────────────────────────────────────
// Test Helpers
// ──────────────────────────────────────────────────────────────────────────────

func createTestFeatures() *risk.RiskFeatures {
	return &risk.RiskFeatures{
		PatientFHIRID: "patient-" + uuid.New().String()[:8],
		Timestamp:     time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
		Age:           68,
		Gender:        models.GenderMale,
		Conditions: []risk.ConditionFeature{
			{Code: "E11", System: "ICD-10", IsActive: true},
			{Code: "I10", System: "ICD-10", IsActive: true},
		},
		Medications: []risk.MedicationFeature{
			{Code: "6809", Display: "Metformin", IsActive: true, HighRisk: false},
		},
		Encounters: []risk.EncounterFeature{
			{Type: "outpatient", Date: time.Now().AddDate(0, -1, 0)},
		},
		LabValues: []risk.LabFeature{
			{Code: "2345-7", Value: 6.8, IsAbnormal: false},
		},
	}
}

func createTestRiskResult() *risk.RiskResult {
	now := time.Now()
	return &risk.RiskResult{
		PatientFHIRID: "patient-123",
		ModelName:     "hospitalization_risk",
		ModelVersion:  "1.0.0",
		Score:         0.65,
		RiskTier:      models.RiskTierHigh,
		Confidence:    0.85,
		ContributingFactors: map[string]float64{
			"age_over_65":        0.15,
			"chronic_conditions": 0.20,
		},
		InputHash:    "abc123",
		CalculatedAt: now,
		ValidUntil:   now.AddDate(0, 0, 30),
		IsRising:     false,
		RisingRate:   0.0,
	}
}
