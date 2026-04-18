package models

import (
	"time"

	"github.com/google/uuid"
)

// RiskTier represents the categorization of predicted risk.
type RiskTier string

const (
	RiskTierHigh     RiskTier = "HIGH"
	RiskTierModerate RiskTier = "MODERATE"
	RiskTierLow      RiskTier = "LOW"
)

// PredictedRisk is the output of the risk predictor for one patient.
type PredictedRisk struct {
	ID                      uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID               string       `gorm:"size:100;index;not null" json:"patient_id"`
	PredictionType          string       `gorm:"size:30;not null;default:'DETERIORATION_30D'" json:"prediction_type"`
	RiskScore               float64      `json:"risk_score"`
	RiskTier                string       `gorm:"size:10" json:"risk_tier"`
	PrimaryDrivers          []RiskFactor `gorm:"-" json:"primary_drivers"`
	ModifiableDrivers       []RiskFactor `gorm:"-" json:"modifiable_drivers"`
	RiskSummary             string       `gorm:"type:text" json:"risk_summary"`
	RecommendedAction       string       `gorm:"type:text" json:"recommended_action"`
	CounterfactualReduction float64      `json:"counterfactual_reduction"`
	PredictionWindowDays    int          `gorm:"default:30" json:"prediction_window_days"`
	ModelType               string       `gorm:"size:20;default:'HEURISTIC'" json:"model_type"`
	ComputedAt              time.Time    `gorm:"not null" json:"computed_at"`
	ExpiresAt               time.Time    `json:"expires_at"`
	CreatedAt               time.Time    `gorm:"autoCreateTime" json:"created_at"`
}

// TableName returns the database table name for PredictedRisk.
func (PredictedRisk) TableName() string { return "predicted_risks" }

// RiskFactor is one contributing factor to a predicted risk score.
type RiskFactor struct {
	FactorName        string  `json:"factor_name"`
	FactorValue       float64 `json:"factor_value"`
	Contribution      float64 `json:"contribution"`
	Direction         string  `json:"direction"`
	Modifiable        bool    `json:"modifiable"`
	Interpretation    string  `json:"interpretation"`
	RecommendedAction string  `json:"recommended_action,omitempty"`
}

// PredictedRiskInput carries all inputs for risk prediction.
type PredictedRiskInput struct {
	PatientID             string
	CompositeSlope30d     *float64
	WorstDomainSlope30d   *float64
	SecondDerivative      *string
	DomainsDeterioring    int
	PAIScore              float64
	PAITrend30d           *float64
	PAICriticalCount90d   int
	EngagementComposite   *float64
	EngagementTrend30d    *float64
	MeasurementFreqDrop   float64
	IsPostDischarge       bool
	DaysSinceDischarge    int
	MedicationChanges30d  int
	PolypharmacyCount     int
	CKMStage              string
	Age                   int
	ActiveConfounderScore float64
	SeasonalWindow        bool
}
