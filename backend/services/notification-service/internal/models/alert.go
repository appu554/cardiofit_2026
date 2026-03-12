package models

import "time"

// ClinicalAlert represents a clinical alert to be notified
type ClinicalAlert struct {
	ID          string                 `json:"id"`
	PatientID   string                 `json:"patient_id"`
	Priority    string                 `json:"priority"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata"`
	Recipients  []string               `json:"recipients"`
	Timestamp   time.Time              `json:"timestamp"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	RequiresAck bool                   `json:"requires_ack"`
}

// NotificationPreference represents user notification preferences (legacy, see UserPreferences in models.go)
type NotificationPreference struct {
	UserID            string   `json:"user_id"`
	PreferredChannels []string `json:"preferred_channels"`
	QuietHoursStart   string   `json:"quiet_hours_start"`
	QuietHoursEnd     string   `json:"quiet_hours_end"`
	EnabledAlertTypes []string `json:"enabled_alert_types"`
	Priority          string   `json:"priority"`
}
