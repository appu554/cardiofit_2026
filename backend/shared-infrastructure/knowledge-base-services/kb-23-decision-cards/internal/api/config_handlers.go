package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// Risk-Scoring Configuration (static, served to Flink)
// ---------------------------------------------------------------------------

// DailyRiskWeights holds the weight factors for daily risk score computation.
type DailyRiskWeights struct {
	VitalStability       float64 `json:"vital_stability"`
	LabAbnormality       float64 `json:"lab_abnormality"`
	MedicationComplexity float64 `json:"medication_complexity"`
}

// RiskLevel describes a single risk band with its threshold range and action.
type RiskLevel struct {
	Name   string `json:"name"`
	Min    int    `json:"min"`
	Max    int    `json:"max"`
	Action string `json:"action"`
}

// PatientVulnerability captures age, comorbidity, and condition risk factors.
type PatientVulnerability struct {
	Age75Plus              int      `json:"age_75_plus"`
	Age65Plus              int      `json:"age_65_plus"`
	ChronicConditions3Plus int      `json:"chronic_conditions_3_plus"`
	ChronicConditions1Plus int      `json:"chronic_conditions_1_plus"`
	HighRiskConditions     []string `json:"high_risk_conditions"`
	HighRiskConditionBonus int      `json:"high_risk_condition_bonus"`
	NEWS2GTE5Baseline      int      `json:"news2_gte_5_baseline"`
}

// RiskScoringConfig is the complete configuration returned by the endpoint.
type RiskScoringConfig struct {
	DailyRiskWeights      DailyRiskWeights   `json:"daily_risk_weights"`
	RiskLevels            []RiskLevel         `json:"risk_levels"`
	AlertSeverityScores   map[string]int      `json:"alert_severity_scores"`
	TimeSensitivityScores map[string]int      `json:"time_sensitivity_scores"`
	PatientVulnerability  PatientVulnerability `json:"patient_vulnerability"`
	Version               string              `json:"version"`
}

// riskScoringConfig is the singleton configuration value.
var riskScoringConfig = RiskScoringConfig{
	DailyRiskWeights: DailyRiskWeights{
		VitalStability:       0.40,
		LabAbnormality:       0.35,
		MedicationComplexity: 0.25,
	},
	RiskLevels: []RiskLevel{
		{Name: "LOW", Min: 0, Max: 24, Action: "routine monitoring"},
		{Name: "MODERATE", Min: 25, Max: 49, Action: "enhanced monitoring"},
		{Name: "HIGH", Min: 50, Max: 74, Action: "frequent assessment"},
		{Name: "CRITICAL", Min: 75, Max: 100, Action: "ICU-level monitoring"},
	},
	AlertSeverityScores: map[string]int{
		"CARDIAC_ARREST":                  10,
		"RESPIRATORY_FAILURE":             10,
		"SEVERE_SEPTIC_SHOCK":             10,
		"SEPSIS_LIKELY":                   9,
		"RESPIRATORY_DISTRESS":            9,
		"AKI_STAGE3":                      8,
		"SEVERE_HYPOTENSION":              8,
		"SPO2_CRITICAL":                   8,
		"VITAL_THRESHOLD_BREACH_HIGH":     7,
		"VITAL_THRESHOLD_BREACH_WARNING":  5,
		"MEDICATION_ALERT":                3,
	},
	TimeSensitivityScores: map[string]int{
		"CARDIAC_ARREST":              5,
		"SEVERE_RESPIRATORY_DISTRESS": 5,
		"SEPSIS_LIKELY":               4,
		"NEWS2_GTE_7":                 3,
		"SEPSIS_PATTERN":              3,
		"RESPIRATORY_DISTRESS":        3,
		"WARNING_SEVERITY":            2,
		"LAB_ABNORMALITY":             2,
		"MEDICATION_ALERT":            1,
	},
	PatientVulnerability: PatientVulnerability{
		Age75Plus:              2,
		Age65Plus:              1,
		ChronicConditions3Plus: 2,
		ChronicConditions1Plus: 1,
		HighRiskConditions:     []string{"diabetes", "heart_failure", "ckd"},
		HighRiskConditionBonus: 1,
		NEWS2GTE5Baseline:      1,
	},
	Version: "2026-03-23T00:00:00Z",
}

// getRiskScoringConfig handles GET /api/v1/config/risk-scoring.
// Returns the static risk-scoring configuration consumed by the Flink
// stream-processing layer for daily risk score computation and alert routing.
func (s *Server) getRiskScoringConfig(c *gin.Context) {
	c.JSON(http.StatusOK, riskScoringConfig)
}
