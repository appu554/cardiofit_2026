package models

import "time"

// NotificationChannel represents the delivery channel for notifications
type NotificationChannel string

const (
	ChannelSMS    NotificationChannel = "SMS"
	ChannelEmail  NotificationChannel = "EMAIL"
	ChannelPush   NotificationChannel = "PUSH"
	ChannelPager  NotificationChannel = "PAGER"
	ChannelVoice  NotificationChannel = "VOICE"
	ChannelInApp  NotificationChannel = "IN_APP"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "CRITICAL"
	SeverityHigh     AlertSeverity = "HIGH"
	SeverityModerate AlertSeverity = "MODERATE"
	SeverityLow      AlertSeverity = "LOW"
	SeverityMLAlert  AlertSeverity = "ML_ALERT"
)

// AlertType represents the type of clinical alert
type AlertType string

const (
	AlertTypeSepsis              AlertType = "SEPSIS_ALERT"
	AlertTypeMortalityRisk       AlertType = "MORTALITY_RISK"
	AlertTypeDeterioration       AlertType = "DETERIORATION"
	AlertTypeReadmissionRisk     AlertType = "READMISSION_RISK"
	AlertTypeVitalSignAnomaly    AlertType = "VITAL_SIGN_ANOMALY"
	AlertTypeTrendDeterioration  AlertType = "TREND_DETERIORATION"
	AlertTypeThresholdViolation  AlertType = "THRESHOLD_VIOLATION"
)

// NotificationStatus represents the delivery status
type NotificationStatus string

const (
	StatusPending       NotificationStatus = "PENDING"
	StatusSending       NotificationStatus = "SENDING"
	StatusSent          NotificationStatus = "SENT"
	StatusDelivered     NotificationStatus = "DELIVERED"
	StatusFailed        NotificationStatus = "FAILED"
	StatusAcknowledged  NotificationStatus = "ACKNOWLEDGED"
)

// PatientLocation contains patient room and bed information
type PatientLocation struct {
	Room string `json:"room"`
	Bed  string `json:"bed"`
}

// VitalSigns contains current vital sign measurements
type VitalSigns struct {
	HeartRate             int     `json:"heart_rate"`
	BloodPressureSystolic int     `json:"blood_pressure_systolic"`
	BloodPressureDiastolic int    `json:"blood_pressure_diastolic,omitempty"`
	Temperature           float64 `json:"temperature"`
	RespiratoryRate       int     `json:"respiratory_rate,omitempty"`
	OxygenSaturation      int     `json:"oxygen_saturation,omitempty"`
}

// AlertMetadata contains alert source and processing information
type AlertMetadata struct {
	SourceModule       string                 `json:"source_module"`
	ModelVersion       string                 `json:"model_version,omitempty"`
	RequiresEscalation bool                   `json:"requires_escalation"`
	FeatureImportance  map[string]interface{} `json:"feature_importance,omitempty"`
	DetectionAlgorithm string                 `json:"detection_algorithm,omitempty"`
	WindowSizeMinutes  int                    `json:"window_size_minutes,omitempty"`
}

// Alert represents a clinical alert event from Kafka
type Alert struct {
	AlertID           string           `json:"alert_id"`
	PatientID         string           `json:"patient_id"`
	HospitalID        string           `json:"hospital_id"`
	DepartmentID      string           `json:"department_id"`
	AlertType         AlertType        `json:"alert_type"`
	Severity          AlertSeverity    `json:"severity"`
	Confidence        float64          `json:"confidence"`
	RiskScore         float64          `json:"risk_score,omitempty"`
	Message           string           `json:"message"`
	Recommendations   []string         `json:"recommendations,omitempty"`
	PatientLocation   PatientLocation  `json:"patient_location"`
	VitalSigns        *VitalSigns      `json:"vital_signs,omitempty"`
	Timestamp         int64            `json:"timestamp"`
	Metadata          AlertMetadata    `json:"metadata"`

	// Pattern-specific fields
	PatternID          string   `json:"pattern_id,omitempty"`
	PatternType        string   `json:"pattern_type,omitempty"`
	AffectedParameters []string `json:"affected_parameters,omitempty"`
}

// User represents a healthcare user with contact information
type User struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Email        string              `json:"email"`
	PhoneNumber  string              `json:"phone_number"`
	PagerNumber  string              `json:"pager_number,omitempty"`
	FCMToken     string              `json:"fcm_token,omitempty"`
	Role         string              `json:"role"` // ATTENDING, CHARGE_NURSE, PRIMARY_NURSE, RESIDENT, etc.
	DepartmentID string              `json:"department_id"`
	Preferences  *UserPreferences    `json:"preferences,omitempty"`
}

// UserPreferences stores user notification preferences
type UserPreferences struct {
	UserID               string                         `json:"user_id"`
	ChannelPreferences   map[NotificationChannel]bool   `json:"channel_preferences"`
	SeverityChannels     map[AlertSeverity][]NotificationChannel `json:"severity_channels"`
	QuietHoursEnabled    bool                           `json:"quiet_hours_enabled"`
	QuietHoursStart      int                            `json:"quiet_hours_start"` // 0-23
	QuietHoursEnd        int                            `json:"quiet_hours_end"`   // 0-23
	MaxAlertsPerHour     int                            `json:"max_alerts_per_hour"`
	UpdatedAt            time.Time                      `json:"updated_at"`
}

// Notification represents a notification to be delivered
type Notification struct {
	ID                string              `json:"id"`
	AlertID           string              `json:"alert_id"`
	UserID            string              `json:"user_id"`
	User              *User               `json:"user"`
	Alert             *Alert              `json:"alert"`
	Channel           NotificationChannel `json:"channel"`
	Priority          int                 `json:"priority"` // 1 (highest) to 5 (lowest)
	Message           string              `json:"message"`
	Status            NotificationStatus  `json:"status"`
	RetryCount        int                 `json:"retry_count"`
	ExternalID        string              `json:"external_id,omitempty"`
	CreatedAt         time.Time           `json:"created_at"`
	SentAt            *time.Time          `json:"sent_at,omitempty"`
	DeliveredAt       *time.Time          `json:"delivered_at,omitempty"`
	AcknowledgedAt    *time.Time          `json:"acknowledged_at,omitempty"`
	ErrorMessage      string              `json:"error_message,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// AlertRecord stores alert history for fatigue tracking
type AlertRecord struct {
	AlertID   string        `json:"alert_id"`
	PatientID string        `json:"patient_id"`
	Type      AlertType     `json:"type"`
	Severity  AlertSeverity `json:"severity"`
	Timestamp time.Time     `json:"timestamp"`
}

// DeliveryResult represents the result of a notification delivery attempt
type DeliveryResult struct {
	Success    bool   `json:"success"`
	ExternalID string `json:"external_id,omitempty"`
	ErrorCode  string `json:"error_code,omitempty"`
	ErrorMsg   string `json:"error_msg,omitempty"`
}

// RoutingDecision contains the routing decision for an alert
type RoutingDecision struct {
	Alert              *Alert                              `json:"alert"`
	TargetUsers        []*User                             `json:"target_users"`
	UserChannels       map[string][]NotificationChannel    `json:"user_channels"`
	SuppressedUsers    map[string]string                   `json:"suppressed_users"` // userID -> reason
	RequiresEscalation bool                                `json:"requires_escalation"`
	EscalationTimeout  time.Duration                       `json:"escalation_timeout"`
}

// Default channel configurations by severity
var DefaultSeverityChannels = map[AlertSeverity][]NotificationChannel{
	SeverityCritical: {ChannelPager, ChannelSMS, ChannelVoice},
	SeverityHigh:     {ChannelSMS, ChannelPush},
	SeverityModerate: {ChannelPush, ChannelInApp},
	SeverityLow:      {ChannelInApp},
	SeverityMLAlert:  {ChannelEmail, ChannelPush},
}

// Default escalation timeouts by severity
var DefaultEscalationTimeouts = map[AlertSeverity]time.Duration{
	SeverityCritical: 5 * time.Minute,
	SeverityHigh:     15 * time.Minute,
	SeverityModerate: 30 * time.Minute,
	SeverityLow:      0, // No escalation
	SeverityMLAlert:  0, // No escalation
}
