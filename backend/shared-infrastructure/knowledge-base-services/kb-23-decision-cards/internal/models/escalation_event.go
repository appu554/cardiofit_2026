package models

import (
	"time"

	"github.com/google/uuid"
)

// EscalationTier classifies the urgency of an escalation.
type EscalationTier string

const (
	TierSafety        EscalationTier = "SAFETY"        // 30 min — stop the harm now
	TierImmediate     EscalationTier = "IMMEDIATE"      // 4 hours — needs attention within hours
	TierUrgent        EscalationTier = "URGENT"         // 24 hours — needs attention today
	TierRoutine       EscalationTier = "ROUTINE"        // 7 days — needs attention this week
	TierInformational EscalationTier = "INFORMATIONAL"  // passive — FYI only
)

// EscalationState tracks the lifecycle of an escalation event.
type EscalationState string

const (
	StatePending      EscalationState = "PENDING"
	StateDelivered    EscalationState = "DELIVERED"
	StateAcknowledged EscalationState = "ACKNOWLEDGED"
	StateActed        EscalationState = "ACTED"
	StateEscalated    EscalationState = "ESCALATED"
	StateResolved     EscalationState = "RESOLVED"
	StateCancelled    EscalationState = "CANCELLED"
	StateExpired      EscalationState = "EXPIRED"
)

// TriggerType identifies what caused the escalation.
type TriggerType string

const (
	TriggerCardGenerated  TriggerType = "CARD_GENERATED"
	TriggerPAIChange      TriggerType = "PAI_CHANGE"
	TriggerSafetyAlert    TriggerType = "SAFETY_ALERT"
	TriggerBatchDetection TriggerType = "BATCH_DETECTION"
)

// EscalationEvent is the core entity for every escalation lifecycle.
type EscalationEvent struct {
	ID                    uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID             string     `gorm:"size:100;index:idx_esc_patient_state,priority:1;not null" json:"patient_id"`
	CardID                *uuid.UUID `gorm:"type:uuid" json:"card_id,omitempty"`
	TriggerType           string     `gorm:"size:30;not null" json:"trigger_type"`
	EscalationTier        string     `gorm:"size:20;not null" json:"escalation_tier"`
	CurrentState          string     `gorm:"size:20;index:idx_esc_patient_state,priority:2;not null;default:'PENDING'" json:"current_state"`
	AssignedClinicianID   string     `gorm:"size:100" json:"assigned_clinician_id,omitempty"`
	AssignedClinicianRole string     `gorm:"size:30" json:"assigned_clinician_role,omitempty"`
	Channels              string     `gorm:"type:text" json:"channels,omitempty"`
	DeliveryAttempts      int        `gorm:"default:0" json:"delivery_attempts"`
	CreatedAt             time.Time  `gorm:"not null;autoCreateTime" json:"created_at"`
	DeliveredAt           *time.Time `json:"delivered_at,omitempty"`
	AcknowledgedAt        *time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy        string     `gorm:"size:100" json:"acknowledged_by,omitempty"`
	ActedAt               *time.Time `json:"acted_at,omitempty"`
	ActionType            string     `gorm:"size:60" json:"action_type,omitempty"`
	ActionDetail          string     `gorm:"type:text" json:"action_detail,omitempty"`
	ResolvedAt            *time.Time `json:"resolved_at,omitempty"`
	ResolutionReason      string     `gorm:"size:100" json:"resolution_reason,omitempty"`
	EscalatedAt           *time.Time `json:"escalated_at,omitempty"`
	EscalationLevel       int        `gorm:"not null;default:1" json:"escalation_level"`
	TimeoutAt             *time.Time `gorm:"index:idx_esc_timeout" json:"timeout_at,omitempty"`
	PreviousEventID       *uuid.UUID `gorm:"type:uuid" json:"previous_event_id,omitempty"`
	PAIScoreAtTrigger     float64    `json:"pai_score_at_trigger"`
	PAITierAtTrigger      string     `gorm:"size:20" json:"pai_tier_at_trigger,omitempty"`
	PrimaryReason         string     `gorm:"size:200" json:"primary_reason,omitempty"`
	SuggestedAction       string     `gorm:"size:200" json:"suggested_action,omitempty"`
	SuggestedTimeframe    string     `gorm:"size:50" json:"suggested_timeframe,omitempty"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the database table name for EscalationEvent.
func (EscalationEvent) TableName() string { return "escalation_events" }

// ClinicianPreferences stores per-clinician notification settings.
type ClinicianPreferences struct {
	ClinicianID             string    `gorm:"size:100;primaryKey" json:"clinician_id"`
	PreferredChannels       string    `gorm:"type:text" json:"preferred_channels"`
	QuietHoursStart         string    `gorm:"size:5;default:'22:00'" json:"quiet_hours_start"`
	QuietHoursEnd           string    `gorm:"size:5;default:'06:00'" json:"quiet_hours_end"`
	Timezone                string    `gorm:"size:50;default:'UTC'" json:"timezone"`
	MaxNotificationsPerHour int       `gorm:"default:10" json:"max_notifications_per_hour"`
	CreatedAt               time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the database table name for ClinicianPreferences.
func (ClinicianPreferences) TableName() string { return "clinician_preferences" }
