package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// JSONB is a json.RawMessage wrapper that implements driver.Valuer and
// sql.Scanner so GORM can read/write PostgreSQL jsonb columns.
// ---------------------------------------------------------------------------

type JSONB json.RawMessage

// Value implements driver.Valuer for database writes.
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

// Scan implements sql.Scanner for database reads.
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = make(JSONB, len(v))
		copy(*j, v)
		return nil
	case string:
		*j = JSONB(v)
		return nil
	default:
		return fmt.Errorf("JSONB.Scan: unsupported type %T", value)
	}
}

// MarshalJSON returns the raw JSON bytes.
func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

// UnmarshalJSON sets the raw JSON bytes.
func (j *JSONB) UnmarshalJSON(data []byte) error {
	if data == nil {
		*j = nil
		return nil
	}
	*j = make(JSONB, len(data))
	copy(*j, data)
	return nil
}

// ---------------------------------------------------------------------------
// DecisionCard is the core entity produced by the KB-23 engine.
// ---------------------------------------------------------------------------

type DecisionCard struct {
	CardID                    uuid.UUID              `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"card_id"`
	PatientID                 uuid.UUID              `gorm:"type:uuid;index;not null" json:"patient_id"`
	SessionID                 *uuid.UUID             `gorm:"type:uuid;index" json:"session_id,omitempty"`
	SnapshotID                *uuid.UUID             `gorm:"type:uuid" json:"snapshot_id,omitempty"`
	TemplateID                string                 `gorm:"index;not null" json:"template_id"`
	NodeID                    string                 `gorm:"index;not null" json:"node_id"`
	PrimaryDifferentialID     string                 `json:"primary_differential_id"`
	PrimaryPosterior          float64                `json:"primary_posterior"`
	DiagnosticConfidenceTier  ConfidenceTier         `gorm:"type:varchar(20);not null" json:"diagnostic_confidence_tier"`
	ConfidenceTierDecayed     bool                   `gorm:"default:false" json:"confidence_tier_decayed"`
	ConfidenceTierDecayReason *string                `json:"confidence_tier_decay_reason,omitempty"`
	MCUGate                   MCUGate                `gorm:"type:varchar(10);not null" json:"mcu_gate"`
	MCUGateRationale          string                 `gorm:"type:text" json:"mcu_gate_rationale"`
	DoseAdjustmentNotes       *string                `gorm:"type:text" json:"dose_adjustment_notes,omitempty"`
	ObservationReliability    ObservationReliability  `gorm:"type:varchar(20);default:'HIGH'" json:"observation_reliability"`
	SecondaryDifferentials    JSONB                  `gorm:"type:jsonb" json:"secondary_differentials,omitempty"`
	ClinicianSummary          string                 `gorm:"type:text" json:"clinician_summary"`
	PatientSummaryEn          string                 `gorm:"type:text" json:"patient_summary_en"`
	PatientSummaryHi          string                 `gorm:"type:text" json:"patient_summary_hi"`
	PatientSummaryLocal       *string                `gorm:"type:text" json:"patient_summary_local,omitempty"`
	PatientSafetyInstructions JSONB                  `gorm:"type:jsonb" json:"patient_safety_instructions,omitempty"`

	// CTL Panel 1: Structured patient state from KB-20
	PatientStateSnapshot JSONB `gorm:"type:jsonb" json:"patient_state_snapshot,omitempty"`

	// CTL Panel 2: Overall guideline condition status
	GuidelineConditionStatus *ConditionStatus `gorm:"type:varchar(20)" json:"guideline_condition_status,omitempty"`

	// CTL Panel 3: Safety check summary assembled from gate evaluation
	SafetyCheckSummary JSONB `gorm:"type:jsonb" json:"safety_check_summary,omitempty"`

	// CTL Panel 4: Reasoning chain from KB-22 Bayesian engine
	ReasoningChain JSONB `gorm:"type:jsonb" json:"reasoning_chain,omitempty"`

	LocaleCode                *string                `gorm:"type:varchar(10)" json:"locale_code,omitempty"`
	SafetyTier                SafetyTier             `gorm:"type:varchar(20);not null" json:"safety_tier"`
	RecurrenceCount           int                    `gorm:"default:0" json:"recurrence_count"`
	CardSource                CardSource             `gorm:"type:varchar(30);not null" json:"card_source"`
	Status                    CardStatus             `gorm:"type:varchar(30);not null;default:'ACTIVE'" json:"status"`
	PendingReaffirmation      bool                   `gorm:"default:false" json:"pending_reaffirmation"`
	ReEntryProtocol           bool                   `gorm:"default:false" json:"re_entry_protocol"`
	SLADeadline               *time.Time             `gorm:"column:sla_deadline" json:"sla_deadline,omitempty"`
	SLABreached               bool                   `gorm:"default:false" json:"sla_breached"`
	SLABreachedAt             *time.Time             `gorm:"column:sla_breached_at" json:"sla_breached_at,omitempty"`
	EscalatedTo               string                 `gorm:"column:escalated_to" json:"escalated_to,omitempty"`
	CreatedAt                 time.Time              `json:"created_at"`
	UpdatedAt                 time.Time              `json:"updated_at"`
	SupersededAt              *time.Time             `json:"superseded_at,omitempty"`
	SupersededBy              *uuid.UUID             `gorm:"type:uuid" json:"superseded_by,omitempty"`

	// ---------------------------------------------------------------------------
	// AD-04: Antihypertensive Deprescribing State Machine Fields
	// ---------------------------------------------------------------------------

	// DeprescribingPhase tracks the current phase of a dose-halving card:
	// DOSE_REDUCTION | MONITORING | REMOVAL | FAILED
	DeprescribingPhase string `gorm:"type:varchar(30)" json:"deprescribing_phase,omitempty"`

	// DeprescribingDrugClass is the antihypertensive class being stepped down
	// (e.g. THIAZIDE, CCB, BETA_BLOCKER, ACE_INHIBITOR, ARB).
	DeprescribingDrugClass string `gorm:"type:varchar(30)" json:"deprescribing_drug_class,omitempty"`

	// PreStepDownDose is the dose (mg) before the step-down began.
	PreStepDownDose *float64 `gorm:"type:decimal(10,2)" json:"pre_step_down_dose,omitempty"`

	// CurrentStepDownDose is the dose (mg) currently active during step-down.
	CurrentStepDownDose *float64 `gorm:"type:decimal(10,2)" json:"current_step_down_dose,omitempty"`

	// MonitoringWindowWeeks is the class-specific monitoring duration:
	// 4 weeks for thiazide/CCB, 6 weeks for beta-blocker/ACEi-ARB.
	MonitoringWindowWeeks int `gorm:"default:0" json:"monitoring_window_weeks,omitempty"`

	// MonitoringStartDate marks when the post-dose-halving monitoring began.
	MonitoringStartDate *time.Time `json:"monitoring_start_date,omitempty"`

	// PreStepDownACRCategory stores the ACR category (A1/A2/A3) before an
	// ACEi/ARB step-down so AD-10 can detect worsening.
	PreStepDownACRCategory string `gorm:"type:varchar(5)" json:"pre_step_down_acr_category,omitempty"`

	// Transient fields (not persisted to DB)
	AdherenceGainFactor float64 `gorm:"-" json:"adherence_gain_factor,omitempty"`

	// Relations
	Recommendations []CardRecommendation `gorm:"foreignKey:CardID" json:"recommendations,omitempty"`
}

// TableName sets the PostgreSQL table name.
func (DecisionCard) TableName() string { return "decision_cards" }

// BeforeCreate generates a UUID primary key if not already set.
func (d *DecisionCard) BeforeCreate(tx *gorm.DB) error {
	if d.CardID == uuid.Nil {
		d.CardID = uuid.New()
	}
	return nil
}
