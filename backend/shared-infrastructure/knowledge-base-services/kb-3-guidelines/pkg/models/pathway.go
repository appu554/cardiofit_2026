package models

import "time"

// PathwayStatus represents the status of a pathway instance
type PathwayStatus string

const (
	PathwayActive    PathwayStatus = "active"
	PathwayCompleted PathwayStatus = "completed"
	PathwaySuspended PathwayStatus = "suspended"
	PathwayCancelled PathwayStatus = "cancelled"
)

// ActionType represents the type of clinical action
type ActionType string

const (
	ActionMedication   ActionType = "medication"
	ActionLab          ActionType = "lab"
	ActionProcedure    ActionType = "procedure"
	ActionAssessment   ActionType = "assessment"
	ActionConsult      ActionType = "consult"
	ActionNotification ActionType = "notification"
)

// ConstraintStatus per README specification
type ConstraintStatus string

const (
	StatusPending       ConstraintStatus = "PENDING"
	StatusMet           ConstraintStatus = "MET"
	StatusApproaching   ConstraintStatus = "APPROACHING"
	StatusOverdue       ConstraintStatus = "OVERDUE"
	StatusMissed        ConstraintStatus = "MISSED"
	StatusNotApplicable ConstraintStatus = "NOT_APPLICABLE"
)

// Action status aliases for semantic clarity when referring to pathway actions
const (
	ActionPending     ConstraintStatus = StatusPending
	ActionMet         ConstraintStatus = StatusMet
	ActionApproaching ConstraintStatus = StatusApproaching
	ActionOverdue     ConstraintStatus = StatusOverdue
	ActionMissed      ConstraintStatus = StatusMissed
)

// PathwayInstance represents an active pathway for a patient
type PathwayInstance struct {
	InstanceID   string                 `json:"instance_id"`
	PathwayID    string                 `json:"pathway_id"`
	PatientID    string                 `json:"patient_id"`
	CurrentStage string                 `json:"current_stage"`
	Status       PathwayStatus          `json:"status"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	SuspendedAt  *time.Time             `json:"suspended_at,omitempty"`
	Context      map[string]interface{} `json:"context"`
	Actions      []PathwayAction        `json:"actions"`
	AuditLog     []AuditEntry           `json:"audit_log"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// PathwayAction represents an action within a pathway
type PathwayAction struct {
	ActionID       string           `json:"action_id"`
	InstanceID     string           `json:"instance_id"`
	Name           string           `json:"name"`
	Type           ActionType       `json:"type"`
	Status         ConstraintStatus `json:"status"`
	Deadline       time.Time        `json:"deadline"`
	GracePeriod    time.Duration    `json:"grace_period"`
	AlertThreshold time.Duration    `json:"alert_threshold"`
	Required       bool             `json:"required"`
	StageID        string           `json:"stage_id"`
	Description    string           `json:"description,omitempty"`
	CompletedAt    *time.Time       `json:"completed_at,omitempty"`
	CompletedBy    string           `json:"completed_by,omitempty"`
	Notes          string           `json:"notes,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
}

// ConstraintEvaluation result from evaluating pathway constraints
type ConstraintEvaluation struct {
	ActionID      string           `json:"action_id"`
	ActionName    string           `json:"action_name"`
	ActionType    ActionType       `json:"action_type"`
	Status        ConstraintStatus `json:"status"`
	Deadline      time.Time        `json:"deadline"`
	TimeRemaining *time.Duration   `json:"time_remaining,omitempty"`
	TimeOverdue   *time.Duration   `json:"time_overdue,omitempty"`
	CompletedAt   *time.Time       `json:"completed_at,omitempty"`
	AlertLevel    string           `json:"alert_level,omitempty"`
}

// StartPathwayRequest for API
type StartPathwayRequest struct {
	PathwayID string                 `json:"pathway_id" binding:"required"`
	PatientID string                 `json:"patient_id" binding:"required"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// CompleteActionRequest for API
type CompleteActionRequest struct {
	ActionID    string `json:"action_id" binding:"required"`
	CompletedBy string `json:"completed_by" binding:"required"`
	Notes       string `json:"notes,omitempty"`
}

// PathwayStatusResponse for API
type PathwayStatusResponse struct {
	Instance    *PathwayInstance       `json:"instance"`
	Pending     []PathwayAction        `json:"pending_actions"`
	Overdue     []PathwayAction        `json:"overdue_actions"`
	Constraints []ConstraintEvaluation `json:"constraint_evaluations"`
}

// StartProtocolRequest for starting a protocol for a patient
type StartProtocolRequest struct {
	ProtocolID   string                 `json:"protocol_id" binding:"required"`
	ProtocolType string                 `json:"protocol_type" binding:"required"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// BatchStartRequest for starting multiple protocols
type BatchStartRequest struct {
	Protocols []StartProtocolRequest `json:"protocols" binding:"required"`
	PatientID string                 `json:"patient_id" binding:"required"`
}
