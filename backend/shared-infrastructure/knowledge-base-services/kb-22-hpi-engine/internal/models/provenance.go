package models

import (
	"time"

	"github.com/google/uuid"
)

// BAY-12: ProvenanceRecord captures a single Bayesian update step for audit trail
// and session replay. Each answer submission creates one record, enabling full
// reconstruction of the posterior evolution from session initialisation to completion.
//
// Fields:
//   - OldLogOdds: log-odds state BEFORE the LR update
//   - NewLogOdds: log-odds state AFTER the LR update
//   - LRDelta: the actual log(LR) applied per differential
//   - InformationGain: H_before - H_after (entropy reduction)
//
// Session replay: GET /api/v1/sessions/:id/provenance returns the ordered chain,
// allowing complete step-by-step reconstruction of Bayesian reasoning.
type ProvenanceRecord struct {
	RecordID  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"record_id"`
	SessionID uuid.UUID `gorm:"type:uuid;index:idx_provenance_session;not null" json:"session_id"`

	// Ordering: monotonically increasing per session
	StepNumber int `gorm:"type:int;not null" json:"step_number"`

	// What triggered this update
	StepType    string `gorm:"type:varchar(32);not null" json:"step_type"` // ANSWER, CM_APPLICATION, SAFETY_FLOOR, SEX_MODIFIER, INIT
	QuestionID  string `gorm:"type:varchar(64)" json:"question_id,omitempty"`
	AnswerValue string `gorm:"type:varchar(32)" json:"answer_value,omitempty"`

	// Log-odds state vectors (JSONB: map[string]float64)
	OldLogOdds JSONB `gorm:"type:jsonb;not null" json:"old_log_odds"`
	NewLogOdds JSONB `gorm:"type:jsonb;not null" json:"new_log_odds"`

	// Per-differential deltas applied in this step
	LRDelta JSONB `gorm:"type:jsonb;default:'{}'" json:"lr_delta"`

	// Entropy reduction
	InformationGain float64 `gorm:"type:float8;default:0" json:"information_gain"`

	// Context
	StratumLabel        string  `gorm:"type:varchar(32)" json:"stratum_label,omitempty"`
	ReliabilityModifier float64 `gorm:"type:float8;default:1.0" json:"reliability_modifier"`
	DampeningFactor     float64 `gorm:"type:float8;default:1.0" json:"dampening_factor"`

	// Metadata
	CreatedAt time.Time `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
}

func (ProvenanceRecord) TableName() string { return "session_provenance" }

// ProvenanceStepType constants for StepType field.
const (
	ProvenanceStepInit         = "INIT"
	ProvenanceStepSexModifier  = "SEX_MODIFIER"
	ProvenanceStepCMApply      = "CM_APPLICATION"
	ProvenanceStepAnswer       = "ANSWER"
	ProvenanceStepSafetyFloor  = "SAFETY_FLOOR"
	ProvenanceStepAcuity       = "ACUITY"
)
