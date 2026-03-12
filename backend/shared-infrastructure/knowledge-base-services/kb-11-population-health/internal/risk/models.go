// Package risk provides risk stratification engine for KB-11 Population Health.
//
// GOVERNANCE REQUIREMENT: All risk calculations MUST:
// 1. Emit governance events to KB-18
// 2. Include input_hash for determinism verification
// 3. Include calculation_hash for audit trail
// 4. Be reproducible: same input → same output (100% guarantee)
package risk

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// ModelConfig represents the configuration for a risk model.
type ModelConfig struct {
	Name        models.RiskModelType `json:"name"`
	Version     string               `json:"version"`
	Description string               `json:"description"`
	Weights     map[string]float64   `json:"weights"`
	Thresholds  RiskThresholds       `json:"thresholds"`
	ValidDays   int                  `json:"valid_days"` // How long the score is valid
}

// RiskThresholds defines the cutoff scores for each risk tier.
type RiskThresholds struct {
	VeryHigh float64 `json:"very_high"` // >= this = VERY_HIGH
	High     float64 `json:"high"`      // >= this = HIGH
	Moderate float64 `json:"moderate"`  // >= this = MODERATE
	Rising   float64 `json:"rising"`    // Rising risk threshold (score increase rate)
	Low      float64 `json:"low"`       // >= this = LOW (below = UNSCORED)
}

// RiskFeatures represents the input features for risk calculation.
type RiskFeatures struct {
	PatientFHIRID string    `json:"patient_fhir_id"`
	Timestamp     time.Time `json:"timestamp"`

	// Demographics
	Age    int            `json:"age"`
	Gender models.Gender  `json:"gender"`

	// Clinical data
	Conditions       []ConditionFeature  `json:"conditions"`
	Medications      []MedicationFeature `json:"medications"`
	LabValues        []LabFeature        `json:"lab_values"`
	VitalSigns       []VitalFeature      `json:"vital_signs"`
	Encounters       []EncounterFeature  `json:"encounters"`

	// Historical scores (for rising risk detection)
	PreviousScores []HistoricalScore `json:"previous_scores,omitempty"`
}

// ConditionFeature represents a clinical condition.
type ConditionFeature struct {
	Code        string    `json:"code"`        // SNOMED or ICD-10 code
	System      string    `json:"system"`      // Code system (snomed, icd10)
	Display     string    `json:"display"`     // Human readable name
	OnsetDate   time.Time `json:"onset_date"`
	IsActive    bool      `json:"is_active"`
	Severity    string    `json:"severity,omitempty"` // mild, moderate, severe
}

// MedicationFeature represents a medication.
type MedicationFeature struct {
	Code      string    `json:"code"`       // RxNorm code
	Display   string    `json:"display"`    // Drug name
	StartDate time.Time `json:"start_date"`
	IsActive  bool      `json:"is_active"`
	HighRisk  bool      `json:"high_risk"` // High-alert medication flag
}

// LabFeature represents a laboratory result.
type LabFeature struct {
	Code         string    `json:"code"`     // LOINC code
	Display      string    `json:"display"`
	Value        float64   `json:"value"`
	Unit         string    `json:"unit"`
	Date         time.Time `json:"date"`
	IsAbnormal   bool      `json:"is_abnormal"`
	ReferenceMin float64   `json:"reference_min,omitempty"`
	ReferenceMax float64   `json:"reference_max,omitempty"`
}

// VitalFeature represents a vital sign measurement.
type VitalFeature struct {
	Code    string    `json:"code"` // LOINC code
	Display string    `json:"display"`
	Value   float64   `json:"value"`
	Unit    string    `json:"unit"`
	Date    time.Time `json:"date"`
}

// EncounterFeature represents a healthcare encounter.
type EncounterFeature struct {
	Type        string    `json:"type"` // emergency, inpatient, outpatient
	Date        time.Time `json:"date"`
	Facility    string    `json:"facility,omitempty"`
	LengthDays  int       `json:"length_days,omitempty"`
	WasPlanned  bool      `json:"was_planned"`
}

// HistoricalScore represents a previous risk score.
type HistoricalScore struct {
	Score        float64   `json:"score"`
	CalculatedAt time.Time `json:"calculated_at"`
	ModelVersion string    `json:"model_version"`
}

// RiskResult represents the output of a risk calculation.
type RiskResult struct {
	PatientFHIRID       string             `json:"patient_fhir_id"`
	ModelName           string             `json:"model_name"`
	ModelVersion        string             `json:"model_version"`
	Score               float64            `json:"score"`
	RiskTier            models.RiskTier    `json:"risk_tier"`
	Confidence          float64            `json:"confidence"`
	ContributingFactors map[string]float64 `json:"contributing_factors"`
	InputHash           string             `json:"input_hash"`
	CalculationHash     string             `json:"calculation_hash"`
	CalculatedAt        time.Time          `json:"calculated_at"`
	ValidUntil          time.Time          `json:"valid_until"`
	IsRising            bool               `json:"is_rising"`
	RisingRate          float64            `json:"rising_rate,omitempty"` // Score increase per month
}

// Hash generates a deterministic SHA-256 hash of the features.
func (f *RiskFeatures) Hash() string {
	// Sort and normalize data for deterministic hashing
	data, _ := json.Marshal(f)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// Hash generates a deterministic SHA-256 hash of the result.
func (r *RiskResult) Hash() string {
	// Create a hashable subset (exclude timestamps and hashes themselves)
	hashable := struct {
		Score               float64            `json:"score"`
		RiskTier            models.RiskTier    `json:"risk_tier"`
		ContributingFactors map[string]float64 `json:"contributing_factors"`
	}{
		Score:               r.Score,
		RiskTier:            r.RiskTier,
		ContributingFactors: r.ContributingFactors,
	}
	data, _ := json.Marshal(hashable)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// ──────────────────────────────────────────────────────────────────────────────
// Default Model Configurations
// ──────────────────────────────────────────────────────────────────────────────

// DefaultHospitalizationModel returns the default hospitalization risk model config.
func DefaultHospitalizationModel() *ModelConfig {
	return &ModelConfig{
		Name:        models.RiskModelHospitalization,
		Version:     "1.0.0",
		Description: "30-day hospitalization risk prediction",
		Weights: map[string]float64{
			"age_over_65":           0.15,
			"age_over_80":           0.10,
			"chronic_conditions":    0.20,
			"recent_hospitalization": 0.25,
			"high_risk_medications": 0.10,
			"abnormal_labs":         0.10,
			"ed_visits_90d":         0.10,
		},
		Thresholds: RiskThresholds{
			VeryHigh: 0.80,
			High:     0.60,
			Moderate: 0.40,
			Rising:   0.15, // 15% increase per month
			Low:      0.10,
		},
		ValidDays: 30,
	}
}

// DefaultReadmissionModel returns the default readmission risk model config.
func DefaultReadmissionModel() *ModelConfig {
	return &ModelConfig{
		Name:        models.RiskModelReadmission,
		Version:     "1.0.0",
		Description: "30-day readmission risk prediction",
		Weights: map[string]float64{
			"prior_admission_30d":    0.30,
			"length_of_stay":         0.15,
			"discharge_disposition":  0.10,
			"chronic_conditions":     0.15,
			"medication_complexity":  0.10,
			"social_support":         0.10,
			"follow_up_scheduled":    0.10,
		},
		Thresholds: RiskThresholds{
			VeryHigh: 0.75,
			High:     0.55,
			Moderate: 0.35,
			Rising:   0.20,
			Low:      0.10,
		},
		ValidDays: 30,
	}
}

// DefaultEDUtilizationModel returns the default ED utilization risk model config.
func DefaultEDUtilizationModel() *ModelConfig {
	return &ModelConfig{
		Name:        models.RiskModelEDUtilization,
		Version:     "1.0.0",
		Description: "90-day ED utilization risk prediction",
		Weights: map[string]float64{
			"ed_visits_12m":          0.30,
			"chronic_conditions":     0.20,
			"mental_health":          0.15,
			"substance_use":          0.10,
			"social_determinants":    0.10,
			"no_primary_care":        0.15,
		},
		Thresholds: RiskThresholds{
			VeryHigh: 0.70,
			High:     0.50,
			Moderate: 0.30,
			Rising:   0.15,
			Low:      0.10,
		},
		ValidDays: 90,
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper Functions
// ──────────────────────────────────────────────────────────────────────────────

// DetermineRiskTier determines the risk tier based on score and thresholds.
func DetermineRiskTier(score float64, thresholds RiskThresholds, isRising bool) models.RiskTier {
	if isRising {
		return models.RiskTierRising
	}
	if score >= thresholds.VeryHigh {
		return models.RiskTierVeryHigh
	}
	if score >= thresholds.High {
		return models.RiskTierHigh
	}
	if score >= thresholds.Moderate {
		return models.RiskTierModerate
	}
	if score >= thresholds.Low {
		return models.RiskTierLow
	}
	return models.RiskTierUnscored
}

// CalculateRisingRisk determines if the patient has rising risk.
func CalculateRisingRisk(currentScore float64, history []HistoricalScore, threshold float64) (bool, float64) {
	if len(history) < 2 {
		return false, 0
	}

	// Get the oldest score within the last 90 days
	var oldestScore *HistoricalScore
	cutoff := time.Now().AddDate(0, -3, 0)

	for i := len(history) - 1; i >= 0; i-- {
		if history[i].CalculatedAt.After(cutoff) {
			oldestScore = &history[i]
			break
		}
	}

	if oldestScore == nil {
		return false, 0
	}

	// Calculate monthly rate of change
	monthsElapsed := time.Since(oldestScore.CalculatedAt).Hours() / (24 * 30)
	if monthsElapsed < 0.5 {
		return false, 0
	}

	scoreChange := currentScore - oldestScore.Score
	monthlyRate := scoreChange / monthsElapsed

	return monthlyRate >= threshold, monthlyRate
}

// NormalizeScore ensures score is between 0 and 1.
func NormalizeScore(score float64) float64 {
	return math.Max(0, math.Min(1, score))
}
