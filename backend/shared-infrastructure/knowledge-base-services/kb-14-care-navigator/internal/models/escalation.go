// Package models contains domain models for KB-14 Care Navigator
package models

import (
	"time"

	"github.com/google/uuid"
)

// EscalationStatus represents the status of an escalation
type EscalationStatus string

const (
	EscalationStatusPending      EscalationStatus = "PENDING"
	EscalationStatusAcknowledged EscalationStatus = "ACKNOWLEDGED"
	EscalationStatusResolved     EscalationStatus = "RESOLVED"
)

// EscalationLevel represents the severity level of an escalation
type EscalationLevel int

const (
	EscalationNone      EscalationLevel = 0
	EscalationWarning   EscalationLevel = 1 // 50% SLA elapsed (standard) or 25% (critical)
	EscalationUrgent    EscalationLevel = 2 // 75% SLA elapsed (standard) or 50% (critical)
	EscalationCritical  EscalationLevel = 3 // 100% SLA elapsed (standard) or 75% (critical)
	EscalationExecutive EscalationLevel = 4 // 125% SLA elapsed (standard) or 100% (critical)
)

// String returns the string representation of the escalation level
func (e EscalationLevel) String() string {
	switch e {
	case EscalationWarning:
		return "WARNING"
	case EscalationUrgent:
		return "URGENT"
	case EscalationCritical:
		return "CRITICAL"
	case EscalationExecutive:
		return "EXECUTIVE"
	default:
		return "NONE"
	}
}

// GetNotificationChannels returns the notification channels for this escalation level
func (e EscalationLevel) GetNotificationChannels() []NotificationChannel {
	switch e {
	case EscalationWarning:
		return []NotificationChannel{NotificationChannelInApp, NotificationChannelEmail}
	case EscalationUrgent:
		return []NotificationChannel{NotificationChannelPush, NotificationChannelInApp, NotificationChannelEmail}
	case EscalationCritical:
		return []NotificationChannel{NotificationChannelPush, NotificationChannelSMS, NotificationChannelInApp, NotificationChannelEmail}
	case EscalationExecutive:
		return []NotificationChannel{NotificationChannelPager, NotificationChannelSMS, NotificationChannelPush, NotificationChannelInApp, NotificationChannelEmail}
	default:
		return []NotificationChannel{NotificationChannelInApp}
	}
}

// EscalationThresholds defines the SLA percentage thresholds for escalation
type EscalationThresholds struct {
	Warning   float64 // Percentage of SLA elapsed for WARNING level
	Urgent    float64 // Percentage of SLA elapsed for URGENT level
	Critical  float64 // Percentage of SLA elapsed for CRITICAL level
	Executive float64 // Percentage of SLA elapsed for EXECUTIVE level
}

// GetStandardThresholds returns standard escalation thresholds
func GetStandardThresholds() EscalationThresholds {
	return EscalationThresholds{
		Warning:   0.50, // 50%
		Urgent:    0.75, // 75%
		Critical:  1.00, // 100%
		Executive: 1.25, // 125%
	}
}

// GetCriticalThresholds returns faster escalation thresholds for critical tasks
func GetCriticalThresholds() EscalationThresholds {
	return EscalationThresholds{
		Warning:   0.25, // 25%
		Urgent:    0.50, // 50%
		Critical:  0.75, // 75%
		Executive: 1.00, // 100%
	}
}

// GetThresholdsForPriority returns the appropriate thresholds based on task priority
func GetThresholdsForPriority(priority TaskPriority) EscalationThresholds {
	if priority == TaskPriorityCritical {
		return GetCriticalThresholds()
	}
	return GetStandardThresholds()
}

// Escalation represents an escalation event for a task
type Escalation struct {
	ID              uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TaskID          uuid.UUID        `gorm:"type:uuid;not null;index" json:"task_id"`
	Level           EscalationLevel  `gorm:"not null" json:"level"`
	Status          EscalationStatus `gorm:"size:30;not null;default:PENDING" json:"status"`
	Reason          string           `gorm:"size:500" json:"reason"`

	// Who was notified
	EscalatedTo     *uuid.UUID `gorm:"type:uuid" json:"escalated_to,omitempty"`
	EscalatedToRole string     `gorm:"size:50" json:"escalated_to_role,omitempty"`

	// Notification tracking
	NotificationSent    bool       `gorm:"default:false" json:"notification_sent"`
	NotificationChannel string     `gorm:"size:20" json:"notification_channel,omitempty"`
	NotificationSentAt  *time.Time `json:"notification_sent_at,omitempty"`

	// Acknowledgment
	Acknowledged   bool       `gorm:"default:false" json:"acknowledged"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy *uuid.UUID `gorm:"type:uuid" json:"acknowledged_by,omitempty"`

	// Resolution
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy *uuid.UUID `gorm:"type:uuid" json:"resolved_by,omitempty"`
	Resolution string     `gorm:"size:500" json:"resolution,omitempty"`

	// SLA information at time of escalation
	SLAElapsedPercent float64 `gorm:"column:sla_elapsed_percent" json:"sla_elapsed_percent"`
	TimeOverdue       int     `gorm:"column:time_overdue_minutes" json:"time_overdue_minutes"` // Minutes past SLA (can be negative if before SLA)

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationship
	Task *Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
}

// TableName returns the table name for Escalation
func (Escalation) TableName() string {
	return "escalations"
}

// EscalationRequest represents a request to escalate a task
type EscalationRequest struct {
	Level   EscalationLevel `json:"level" binding:"required"`
	Reason  string          `json:"reason,omitempty"`
	EscalateTo *uuid.UUID   `json:"escalate_to,omitempty"`
}

// EscalationResponse wraps an escalation for API responses
type EscalationResponse struct {
	Success bool        `json:"success"`
	Data    *Escalation `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// EscalationListResponse wraps a list of escalations for API responses
type EscalationListResponse struct {
	Success bool         `json:"success"`
	Data    []Escalation `json:"data,omitempty"`
	Total   int64        `json:"total"`
	Error   string       `json:"error,omitempty"`
}

// EscalationSummary provides a summary of escalations
type EscalationSummary struct {
	TotalEscalations     int64 `json:"total_escalations"`
	WarningCount         int64 `json:"warning_count"`
	UrgentCount          int64 `json:"urgent_count"`
	CriticalCount        int64 `json:"critical_count"`
	ExecutiveCount       int64 `json:"executive_count"`
	UnacknowledgedCount  int64 `json:"unacknowledged_count"`
	AverageResponseTime  int   `json:"average_response_time_minutes"`
}

// CalculateEscalationLevel determines the appropriate escalation level based on SLA elapsed
func CalculateEscalationLevel(slaElapsedPercent float64, priority TaskPriority) EscalationLevel {
	thresholds := GetThresholdsForPriority(priority)

	switch {
	case slaElapsedPercent >= thresholds.Executive:
		return EscalationExecutive
	case slaElapsedPercent >= thresholds.Critical:
		return EscalationCritical
	case slaElapsedPercent >= thresholds.Urgent:
		return EscalationUrgent
	case slaElapsedPercent >= thresholds.Warning:
		return EscalationWarning
	default:
		return EscalationNone
	}
}

// GetEscalationReason generates a human-readable reason for the escalation
func GetEscalationReason(level EscalationLevel, taskType TaskType, slaElapsedPercent float64) string {
	percentStr := ""
	switch {
	case slaElapsedPercent >= 1.25:
		percentStr = "125%+ of SLA elapsed"
	case slaElapsedPercent >= 1.0:
		percentStr = "SLA deadline reached"
	case slaElapsedPercent >= 0.75:
		percentStr = "75% of SLA elapsed"
	case slaElapsedPercent >= 0.50:
		percentStr = "50% of SLA elapsed"
	case slaElapsedPercent >= 0.25:
		percentStr = "25% of SLA elapsed"
	default:
		percentStr = "approaching deadline"
	}

	switch level {
	case EscalationWarning:
		return "Task " + string(taskType) + " - " + percentStr + " - supervisor notification"
	case EscalationUrgent:
		return "Task " + string(taskType) + " - " + percentStr + " - manager escalation"
	case EscalationCritical:
		return "Task " + string(taskType) + " - " + percentStr + " - deadline reached"
	case EscalationExecutive:
		return "Task " + string(taskType) + " - " + percentStr + " - executive escalation"
	default:
		return ""
	}
}
