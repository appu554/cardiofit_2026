package models

import (
	"time"

	"github.com/google/uuid"
)

// LifecycleState tracks progression of a detection through the response pipeline.
type LifecycleState string

const (
	LifecyclePendingNotification LifecycleState = "PENDING_NOTIFICATION"
	LifecycleNotified            LifecycleState = "NOTIFIED"
	LifecycleAcknowledged        LifecycleState = "ACKNOWLEDGED"
	LifecycleActioned            LifecycleState = "ACTIONED"
	LifecycleResolved            LifecycleState = "RESOLVED"
	LifecycleTimedOut            LifecycleState = "TIMED_OUT"
	LifecycleCancelled           LifecycleState = "CANCELLED"
)

// DetectionLifecycle tracks the T0->T4 lifecycle of a single detection event.
type DetectionLifecycle struct {
	ID                      uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DetectionType           string     `gorm:"size:40;index;not null" json:"detection_type"`
	DetectionSubtype        string     `gorm:"size:60" json:"detection_subtype,omitempty"`
	PatientID               string     `gorm:"size:100;index;not null" json:"patient_id"`
	AssignedClinicianID     string     `gorm:"size:100;index" json:"assigned_clinician_id,omitempty"`
	CurrentState            string     `gorm:"size:30;not null;default:'PENDING_NOTIFICATION'" json:"current_state"`
	TierAtDetection         string     `gorm:"size:20" json:"tier_at_detection"`
	DetectedAt              time.Time  `gorm:"not null" json:"detected_at"`
	DeliveredAt             *time.Time `json:"delivered_at,omitempty"`
	AcknowledgedAt          *time.Time `json:"acknowledged_at,omitempty"`
	ActionedAt              *time.Time `json:"actioned_at,omitempty"`
	ResolvedAt              *time.Time `json:"resolved_at,omitempty"`
	DeliveryLatencyMs       *int64     `json:"delivery_latency_ms,omitempty"`
	AcknowledgmentLatencyMs *int64     `json:"acknowledgment_latency_ms,omitempty"`
	ActionLatencyMs         *int64     `json:"action_latency_ms,omitempty"`
	OutcomeLatencyMs        *int64     `json:"outcome_latency_ms,omitempty"`
	TotalLatencyMs          *int64     `json:"total_latency_ms,omitempty"`
	ActionType              string     `gorm:"size:60" json:"action_type,omitempty"`
	ActionDetail            string     `gorm:"type:text" json:"action_detail,omitempty"`
	OutcomeDescription      string     `gorm:"type:text" json:"outcome_description,omitempty"`
	CardID                  *uuid.UUID `gorm:"type:uuid" json:"card_id,omitempty"`
	EscalationID            *uuid.UUID `gorm:"type:uuid" json:"escalation_id,omitempty"`
	SourceService           string     `gorm:"size:30" json:"source_service"`
	CreatedAt               time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName sets the database table name for DetectionLifecycle.
func (DetectionLifecycle) TableName() string { return "detection_lifecycles" }

// ClinicianResponseMetrics holds computed metrics for one clinician.
type ClinicianResponseMetrics struct {
	ClinicianID            string `json:"clinician_id"`
	WindowDays             int    `json:"window_days"`
	TotalDetections        int    `json:"total_detections"`
	MedianDeliveryMs       *int64 `json:"median_delivery_ms,omitempty"`
	MedianAcknowledgmentMs *int64 `json:"median_acknowledgment_ms,omitempty"`
	MedianActionMs         *int64 `json:"median_action_ms,omitempty"`
	ActionCompletionRate   float64 `json:"action_completion_rate"`
	OutcomeRate            float64 `json:"outcome_rate"`
	TeamMedianAckMs        *int64 `json:"team_median_ack_ms,omitempty"`
}

// SystemResponseMetrics holds aggregate metrics.
type SystemResponseMetrics struct {
	WindowDays           int                    `json:"window_days"`
	TotalDetections      int                    `json:"total_detections"`
	MedianT0toT2Ms       *int64                 `json:"median_t0_to_t2_ms,omitempty"`
	MedianT0toT3Ms       *int64                 `json:"median_t0_to_t3_ms,omitempty"`
	ActionCompletionRate float64                `json:"action_completion_rate"`
	OutcomeRate          float64                `json:"outcome_rate"`
	TimeoutRate          float64                `json:"timeout_rate"`
	ByTier               map[string]TierMetrics `json:"by_tier,omitempty"`
}

// TierMetrics holds metrics for a specific escalation tier.
type TierMetrics struct {
	Count                int     `json:"count"`
	MedianAckMs          *int64  `json:"median_ack_ms,omitempty"`
	MedianActionMs       *int64  `json:"median_action_ms,omitempty"`
	ActionCompletionRate float64 `json:"action_completion_rate"`
}

// PilotMetrics holds HCF CHF pilot-specific KPIs.
type PilotMetrics struct {
	TotalDetections             int     `json:"total_detections"`
	DetectionsAcknowledgedInTime int    `json:"detections_acknowledged_in_time"`
	DetectionsWithAction        int     `json:"detections_with_action"`
	MedicationChanges           int     `json:"medication_changes"`
	OutreachCalls               int     `json:"outreach_calls"`
	AppointmentsScheduled       int     `json:"appointments_scheduled"`
	MedianDetectionToActionHrs  float64 `json:"median_detection_to_action_hrs"`
	PatientsWithTimelyAction    int     `json:"patients_with_timely_action"`
	PatientsWithoutTimelyAction int     `json:"patients_without_timely_action"`
}
