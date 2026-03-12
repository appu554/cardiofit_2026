// Package models contains domain models for KB-14 Care Navigator
package models

import (
	"time"

	"github.com/google/uuid"
)

// NotificationChannel represents a notification delivery channel
type NotificationChannel string

const (
	NotificationChannelInApp  NotificationChannel = "in_app"
	NotificationChannelEmail  NotificationChannel = "email"
	NotificationChannelSMS    NotificationChannel = "sms"
	NotificationChannelPush   NotificationChannel = "push"
	NotificationChannelPager  NotificationChannel = "pager"
)

// NotificationPriority represents the priority of a notification
type NotificationPriority string

const (
	NotificationPriorityCritical NotificationPriority = "critical"
	NotificationPriorityHigh     NotificationPriority = "high"
	NotificationPriorityNormal   NotificationPriority = "normal"
	NotificationPriorityLow      NotificationPriority = "low"
)

// GetChannelsForPriority returns the notification channels for a given priority
func GetChannelsForPriority(priority TaskPriority) []NotificationChannel {
	switch priority {
	case TaskPriorityCritical:
		return []NotificationChannel{
			NotificationChannelPager,
			NotificationChannelSMS,
			NotificationChannelPush,
			NotificationChannelInApp,
			NotificationChannelEmail,
		}
	case TaskPriorityHigh:
		return []NotificationChannel{
			NotificationChannelPush,
			NotificationChannelInApp,
			NotificationChannelEmail,
		}
	case TaskPriorityMedium:
		return []NotificationChannel{
			NotificationChannelInApp,
			NotificationChannelEmail,
		}
	default: // LOW
		return []NotificationChannel{
			NotificationChannelInApp,
		}
	}
}

// NotificationRequest represents a request to send a notification
type NotificationRequest struct {
	// Recipient
	RecipientID   string              `json:"recipient_id"`
	RecipientType string              `json:"recipient_type"` // user, team, role
	RecipientName string              `json:"recipient_name,omitempty"`

	// Content
	Title     string `json:"title"`
	Message   string `json:"message"`
	ActionURL string `json:"action_url,omitempty"`

	// Priority and channel
	Priority NotificationPriority  `json:"priority"`
	Channels []NotificationChannel `json:"channels"`

	// Context
	TaskID     *uuid.UUID `json:"task_id,omitempty"`
	PatientID  string     `json:"patient_id,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationResult represents the result of sending a notification
type NotificationResult struct {
	Success   bool                `json:"success"`
	Channel   NotificationChannel `json:"channel"`
	SentAt    *time.Time          `json:"sent_at,omitempty"`
	MessageID string              `json:"message_id,omitempty"`
	Error     string              `json:"error,omitempty"`
}

// NotificationResponse wraps notification results for API responses
type NotificationResponse struct {
	Success bool                 `json:"success"`
	Results []NotificationResult `json:"results,omitempty"`
	Error   string               `json:"error,omitempty"`
}

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"` // task_assigned, task_escalated, task_due_soon, etc.
	Subject     string `json:"subject"`
	BodyHTML    string `json:"body_html"`
	BodyText    string `json:"body_text"`
	PushTitle   string `json:"push_title"`
	PushBody    string `json:"push_body"`
	SMSTemplate string `json:"sms_template"`
}

// Common notification types
const (
	NotificationTypeTaskAssigned    = "task_assigned"
	NotificationTypeTaskEscalated   = "task_escalated"
	NotificationTypeTaskDueSoon     = "task_due_soon"
	NotificationTypeTaskOverdue     = "task_overdue"
	NotificationTypeTaskCompleted   = "task_completed"
	NotificationTypeTaskNote        = "task_note_added"
	NotificationTypeCareGap         = "care_gap_detected"
	NotificationTypeProtocolDeadline = "protocol_deadline"
)

// BuildTaskAssignedNotification creates a notification for task assignment
func BuildTaskAssignedNotification(task *Task, assigneeName string) NotificationRequest {
	channels := GetChannelsForPriority(task.Priority)

	return NotificationRequest{
		RecipientID:   task.AssignedTo.String(),
		RecipientType: "user",
		RecipientName: assigneeName,
		Title:         "New Task Assigned: " + task.Title,
		Message:       "You have been assigned a new " + string(task.Type) + " task for patient " + task.PatientID,
		ActionURL:     "/tasks/" + task.ID.String(),
		Priority:      NotificationPriority(task.Priority),
		Channels:      channels,
		TaskID:        &task.ID,
		PatientID:     task.PatientID,
		Metadata: map[string]interface{}{
			"task_type": task.Type,
			"due_date":  task.DueDate,
		},
	}
}

// BuildEscalationNotification creates a notification for task escalation
func BuildEscalationNotification(task *Task, escalation *Escalation, recipientID string, recipientName string) NotificationRequest {
	priority := NotificationPriorityHigh
	if escalation.Level >= EscalationCritical {
		priority = NotificationPriorityCritical
	}

	return NotificationRequest{
		RecipientID:   recipientID,
		RecipientType: "user",
		RecipientName: recipientName,
		Title:         "Task Escalation: " + escalation.Level.String(),
		Message:       escalation.Reason,
		ActionURL:     "/tasks/" + task.ID.String(),
		Priority:      priority,
		Channels:      escalation.Level.GetNotificationChannels(),
		TaskID:        &task.ID,
		PatientID:     task.PatientID,
		Metadata: map[string]interface{}{
			"escalation_level": escalation.Level,
			"task_type":        task.Type,
		},
	}
}

// BuildDueSoonNotification creates a notification for tasks due soon
func BuildDueSoonNotification(task *Task, minutesRemaining int) NotificationRequest {
	return NotificationRequest{
		RecipientID:   task.AssignedTo.String(),
		RecipientType: "user",
		Title:         "Task Due Soon: " + task.Title,
		Message:       "Task is due in " + formatDuration(minutesRemaining),
		ActionURL:     "/tasks/" + task.ID.String(),
		Priority:      NotificationPriorityHigh,
		Channels:      GetChannelsForPriority(task.Priority),
		TaskID:        &task.ID,
		PatientID:     task.PatientID,
		Metadata: map[string]interface{}{
			"minutes_remaining": minutesRemaining,
			"task_type":         task.Type,
		},
	}
}

// formatDuration formats minutes into a human-readable string
func formatDuration(minutes int) string {
	if minutes < 60 {
		return string(rune(minutes)) + " minutes"
	}
	hours := minutes / 60
	if hours < 24 {
		return string(rune(hours)) + " hours"
	}
	days := hours / 24
	return string(rune(days)) + " days"
}
