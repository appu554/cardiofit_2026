package models

import (
	"time"

	"github.com/google/uuid"
)

// TreatmentStrategy is the TTE-protocol treatment label inferred from the clinician response.
// Derived from DetectionLifecycle.ActionType (Gap 19) and the override/intervention choice.
type TreatmentStrategy string

const (
	TreatmentInterventionTaken TreatmentStrategy = "INTERVENTION_TAKEN"
	TreatmentOverrideReason    TreatmentStrategy = "OVERRIDE_WITH_REASON"
	TreatmentNoResponse        TreatmentStrategy = "NO_RESPONSE"
	TreatmentAlreadyAddressed  TreatmentStrategy = "ALREADY_ADDRESSED"
)

// ConsolidatedAlertRecord is the TTE-ready per-alert record combining:
// - pre-alert prediction snapshot (Gap 20 PredictedRisk)
// - lifecycle timestamps (Gap 19 DetectionLifecycle)
// - treatment strategy (clinician response)
// - outcome (Task 1 OutcomeRecord)
// - time-zero (TTE protocol anchor)
type ConsolidatedAlertRecord struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	LifecycleID       uuid.UUID  `gorm:"type:uuid;index;not null" json:"lifecycle_id"`
	PatientID         string     `gorm:"size:100;index;not null" json:"patient_id"`
	CohortID          string     `gorm:"size:60;index" json:"cohort_id,omitempty"`

	// Pre-alert snapshot
	PreAlertRiskScore float64    `json:"pre_alert_risk_score"`
	PreAlertRiskTier  string     `gorm:"size:10" json:"pre_alert_risk_tier"`
	PredictionModelID string     `gorm:"size:60" json:"prediction_model_id,omitempty"`

	// Lifecycle anchors
	DetectedAt        time.Time  `gorm:"not null" json:"detected_at"`
	DeliveredAt       *time.Time `json:"delivered_at,omitempty"`
	AcknowledgedAt    *time.Time `json:"acknowledged_at,omitempty"`
	ActionedAt        *time.Time `json:"actioned_at,omitempty"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`

	// Causal annotation
	TimeZero          time.Time  `gorm:"not null" json:"time_zero"`
	TreatmentStrategy string     `gorm:"size:40;index;not null" json:"treatment_strategy"`
	ActionType        string     `gorm:"size:60" json:"action_type,omitempty"`
	OverrideReason    string     `gorm:"size:60" json:"override_reason,omitempty"`

	// Outcome
	OutcomeRecordID   *uuid.UUID `gorm:"type:uuid;index" json:"outcome_record_id,omitempty"`
	OutcomeOccurred   *bool      `json:"outcome_occurred,omitempty"`
	OutcomeType       string     `gorm:"size:60" json:"outcome_type,omitempty"`
	HorizonDays       int        `json:"horizon_days"`

	BuiltAt           time.Time  `gorm:"autoCreateTime" json:"built_at"`
}

func (ConsolidatedAlertRecord) TableName() string { return "consolidated_alert_records" }
