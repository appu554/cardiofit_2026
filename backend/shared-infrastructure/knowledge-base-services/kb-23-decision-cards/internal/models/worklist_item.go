package models

import "time"

// WorklistItem is one patient's most important clinical finding.
type WorklistItem struct {
	PatientID          string         `json:"patient_id"`
	PatientName        string         `json:"patient_name"`
	PatientAge         int            `json:"patient_age"`
	CareSetting        string         `json:"care_setting"`
	UrgencyTier        string         `json:"urgency_tier"`
	PAIScore           float64        `json:"pai_score"`
	PAITrend           string         `json:"pai_trend"`
	EscalationTier     string         `json:"escalation_tier"`
	TriggeringSource   string         `json:"triggering_source"`
	PrimaryReason      string         `json:"primary_reason"`
	SuggestedAction    string         `json:"suggested_action"`
	SuggestedTimeframe string         `json:"suggested_timeframe"`
	ActionButtons      []ActionButton `json:"action_buttons"`
	ContextTags        []string       `json:"context_tags"`
	DominantDimension  string         `json:"dominant_dimension,omitempty"`
	LastClinicianDays  int            `json:"last_clinician_days"`
	UnderlyingCardIDs  []string       `json:"underlying_card_ids,omitempty"`
	ResolutionState    string         `json:"resolution_state"`
	ComputedAt         time.Time      `json:"computed_at"`
}

// ActionButton is a one-tap action on a worklist item.
type ActionButton struct {
	ActionCode   string `json:"action_code"`
	DisplayLabel string `json:"display_label"`
	Primary      bool   `json:"primary"`
}

// WorklistView is a complete worklist for one clinician.
type WorklistView struct {
	ClinicianID   string         `json:"clinician_id"`
	PersonaType   string         `json:"persona_type"`
	Items         []WorklistItem `json:"items"`
	TotalCount    int            `json:"total_count"`
	CriticalCount int            `json:"critical_count"`
	HighCount     int            `json:"high_count"`
	ModerateCount int            `json:"moderate_count"`
	LowCount      int            `json:"low_count"`
	LastRefreshed time.Time      `json:"last_refreshed"`
}

// WorklistActionRequest is a clinician action on a worklist item.
type WorklistActionRequest struct {
	PatientID   string `json:"patient_id" binding:"required"`
	ClinicianID string `json:"clinician_id" binding:"required"`
	ActionCode  string `json:"action_code" binding:"required"`
	Notes       string `json:"notes,omitempty"`
	DeferHours  int    `json:"defer_hours,omitempty"`
}

// WorklistFeedback is clinician feedback for trust calibration.
type WorklistFeedback struct {
	PatientID    string    `json:"patient_id"`
	ClinicianID  string    `json:"clinician_id"`
	FeedbackType string    `json:"feedback_type"`
	Reason       string    `json:"reason,omitempty"`
	SubmittedAt  time.Time `json:"submitted_at"`
}

// Urgency tier constants for worklist display grouping.
const (
	WorklistTierCritical = "CRITICAL"
	WorklistTierHigh     = "HIGH"
	WorklistTierModerate = "MODERATE"
	WorklistTierLow      = "LOW"
)

// Worklist triggering source constants.
const (
	WorklistTriggerEscalation          = "ESCALATION"
	WorklistTriggerPAIChange           = "PAI_CHANGE"
	WorklistTriggerAcuteEvent          = "ACUTE_EVENT"
	WorklistTriggerTransitionMilestone = "TRANSITION_MILESTONE"
	WorklistTriggerCard                = "CARD"
	WorklistTriggerAttentionGap        = "ATTENTION_GAP"
)

// Resolution state constants.
const (
	ResolutionPending    = "PENDING"
	ResolutionInProgress = "IN_PROGRESS"
	ResolutionResolved   = "RESOLVED"
	ResolutionDeferred   = "DEFERRED"
)
