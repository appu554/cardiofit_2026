package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MCUGateHistory records every MCU gate transition for audit and analytics.
type MCUGateHistory struct {
	HistoryID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"history_id"`
	PatientID             uuid.UUID  `gorm:"type:uuid;index;not null" json:"patient_id"`
	CardID                uuid.UUID  `gorm:"type:uuid" json:"card_id"`
	GateValue             MCUGate    `gorm:"type:varchar(10);not null" json:"gate_value"`
	PreviousGate          *MCUGate   `gorm:"type:varchar(10)" json:"previous_gate,omitempty"`
	SessionID             *uuid.UUID `gorm:"type:uuid" json:"session_id,omitempty"`
	TransitionReason      string     `gorm:"type:text" json:"transition_reason"`
	ClinicianResumeBy     *string    `json:"clinician_resume_by,omitempty"`
	ClinicianResumeReason *string    `gorm:"type:text" json:"clinician_resume_reason,omitempty"`
	ReEntryProtocol       bool       `gorm:"default:false" json:"re_entry_protocol"`
	HaltDurationHours     *float64   `json:"halt_duration_hours,omitempty"`
	AcknowledgedAt        *time.Time `json:"acknowledged_at,omitempty"`
	CreatedAt             time.Time  `gorm:"index" json:"created_at"`
}

// TableName sets the PostgreSQL table name.
func (MCUGateHistory) TableName() string { return "mcu_gate_history" }

// BeforeCreate generates a UUID primary key if not already set.
func (h *MCUGateHistory) BeforeCreate(tx *gorm.DB) error {
	if h.HistoryID == uuid.Nil {
		h.HistoryID = uuid.New()
	}
	return nil
}

// EnrichedMCUGateResponse is the Redis-cached response that V-MCU reads
// on every titration cycle.
type EnrichedMCUGateResponse struct {
	MCUGate                 MCUGate                `json:"mcu_gate"`
	DoseAdjustmentNotes     string                 `json:"dose_adjustment_notes,omitempty"`
	AdherenceGainFactor     float64                `json:"adherence_gain_factor"`
	AdherenceScoreSource    string                 `json:"adherence_score_source,omitempty"`
	ObservationReliability  ObservationReliability  `json:"observation_reliability"`
	ActivePerturbationCount int                    `json:"active_perturbation_count"`
	ReEntryProtocol         bool                   `json:"re_entry_protocol"`
	GateCardID              uuid.UUID              `json:"gate_card_id"`
}
