// Package models contains domain models for KB-14 Care Navigator
package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the lifecycle status of a task
type TaskStatus string

const (
	TaskStatusCreated    TaskStatus = "CREATED"
	TaskStatusAssigned   TaskStatus = "ASSIGNED"
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusCompleted  TaskStatus = "COMPLETED"
	TaskStatusVerified   TaskStatus = "VERIFIED"
	TaskStatusDeclined   TaskStatus = "DECLINED"
	TaskStatusBlocked    TaskStatus = "BLOCKED"
	TaskStatusEscalated  TaskStatus = "ESCALATED"
	TaskStatusCancelled  TaskStatus = "CANCELLED"
)

// IsValid checks if the status is valid
func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusCreated, TaskStatusAssigned, TaskStatusInProgress,
		TaskStatusCompleted, TaskStatusVerified, TaskStatusDeclined,
		TaskStatusBlocked, TaskStatusEscalated, TaskStatusCancelled:
		return true
	}
	return false
}

// TaskType represents the type/category of task
type TaskType string

const (
	// Clinical Tasks (Licensed Clinician Required)
	TaskTypeCriticalLabReview     TaskType = "CRITICAL_LAB_REVIEW"
	TaskTypeMedicationReview      TaskType = "MEDICATION_REVIEW"
	TaskTypeAbnormalResult        TaskType = "ABNORMAL_RESULT"
	TaskTypeTherapeuticChange     TaskType = "THERAPEUTIC_CHANGE"
	TaskTypeCarePlanReview        TaskType = "CARE_PLAN_REVIEW"
	TaskTypeAcuteProtocolDeadline TaskType = "ACUTE_PROTOCOL_DEADLINE"

	// Care Coordination Tasks
	TaskTypeCareGapClosure     TaskType = "CARE_GAP_CLOSURE"
	TaskTypeMonitoringOverdue  TaskType = "MONITORING_OVERDUE"
	TaskTypeTransitionFollowup TaskType = "TRANSITION_FOLLOWUP"
	TaskTypeAnnualWellness     TaskType = "ANNUAL_WELLNESS"
	TaskTypeChronicCareMgmt    TaskType = "CHRONIC_CARE_MGMT"

	// Patient Outreach Tasks
	TaskTypeAppointmentRemind  TaskType = "APPOINTMENT_REMIND"
	TaskTypeMissedAppointment  TaskType = "MISSED_APPOINTMENT"
	TaskTypeScreeningOutreach  TaskType = "SCREENING_OUTREACH"
	TaskTypeMedicationRefill   TaskType = "MEDICATION_REFILL"

	// Administrative Tasks
	TaskTypePriorAuthNeeded    TaskType = "PRIOR_AUTH_NEEDED"
	TaskTypeReferralProcessing TaskType = "REFERRAL_PROCESSING"
)

// GetDefaultRole returns the default assignee role for this task type
func (t TaskType) GetDefaultRole() string {
	switch t {
	case TaskTypeCriticalLabReview, TaskTypeTherapeuticChange:
		return "Physician"
	case TaskTypeMedicationReview:
		return "Pharmacist"
	case TaskTypeAbnormalResult:
		return "Ordering MD"
	case TaskTypeCarePlanReview:
		return "PCP"
	case TaskTypeAcuteProtocolDeadline:
		return "Attending"
	case TaskTypeCareGapClosure, TaskTypeMonitoringOverdue:
		return "Care Coordinator"
	case TaskTypeTransitionFollowup:
		return "Transition Coordinator"
	case TaskTypeAnnualWellness:
		return "Nurse"
	case TaskTypeChronicCareMgmt:
		return "Care Manager"
	case TaskTypeAppointmentRemind:
		return "Scheduler"
	case TaskTypeMissedAppointment, TaskTypeScreeningOutreach, TaskTypeMedicationRefill:
		return "Outreach Specialist"
	case TaskTypePriorAuthNeeded:
		return "Auth Specialist"
	case TaskTypeReferralProcessing:
		return "Referral Coordinator"
	default:
		return "Care Coordinator"
	}
}

// IsValid checks if the task type is valid
func (t TaskType) IsValid() bool {
	switch t {
	case TaskTypeCriticalLabReview, TaskTypeMedicationReview, TaskTypeAbnormalResult,
		TaskTypeTherapeuticChange, TaskTypeCarePlanReview, TaskTypeAcuteProtocolDeadline,
		TaskTypeCareGapClosure, TaskTypeMonitoringOverdue, TaskTypeTransitionFollowup,
		TaskTypeAnnualWellness, TaskTypeChronicCareMgmt, TaskTypeAppointmentRemind,
		TaskTypeMissedAppointment, TaskTypeScreeningOutreach, TaskTypeMedicationRefill,
		TaskTypePriorAuthNeeded, TaskTypeReferralProcessing:
		return true
	}
	return false
}

// GetDefaultSLAMinutes returns the default SLA in minutes for this task type
func (t TaskType) GetDefaultSLAMinutes() int {
	switch t {
	case TaskTypeCriticalLabReview, TaskTypeAcuteProtocolDeadline:
		return 60 // 1 hour
	case TaskTypeMedicationReview:
		return 240 // 4 hours
	case TaskTypeAbnormalResult:
		return 1440 // 24 hours
	case TaskTypeTherapeuticChange, TaskTypeMissedAppointment:
		return 2880 // 48 hours
	case TaskTypeMonitoringOverdue, TaskTypeAppointmentRemind:
		return 4320 // 3 days
	case TaskTypeReferralProcessing:
		return 7200 // 5 days
	case TaskTypeCarePlanReview, TaskTypeTransitionFollowup, TaskTypeMedicationRefill:
		return 10080 // 7 days
	case TaskTypeScreeningOutreach:
		return 20160 // 14 days
	case TaskTypeCareGapClosure, TaskTypeAnnualWellness, TaskTypeChronicCareMgmt:
		return 43200 // 30 days
	case TaskTypePriorAuthNeeded:
		return 4320 // 3 days
	default:
		return 10080 // 7 days default
	}
}

// TaskPriority represents the priority level of a task
type TaskPriority string

const (
	TaskPriorityCritical TaskPriority = "CRITICAL"
	TaskPriorityHigh     TaskPriority = "HIGH"
	TaskPriorityMedium   TaskPriority = "MEDIUM"
	TaskPriorityLow      TaskPriority = "LOW"
)

// GetDefaultPriority returns the default priority for this task type
func (t TaskType) GetDefaultPriority() TaskPriority {
	switch t {
	case TaskTypeCriticalLabReview, TaskTypeAcuteProtocolDeadline:
		return TaskPriorityCritical
	case TaskTypeMedicationReview, TaskTypeAbnormalResult, TaskTypeMonitoringOverdue,
		TaskTypeTransitionFollowup, TaskTypePriorAuthNeeded:
		return TaskPriorityHigh
	case TaskTypeTherapeuticChange, TaskTypeCarePlanReview, TaskTypeCareGapClosure,
		TaskTypeChronicCareMgmt, TaskTypeMissedAppointment, TaskTypeScreeningOutreach,
		TaskTypeReferralProcessing:
		return TaskPriorityMedium
	default:
		return TaskPriorityLow
	}
}

// TaskSource represents the source of the task
type TaskSource string

const (
	TaskSourceKB3    TaskSource = "KB3_TEMPORAL"
	TaskSourceKB9    TaskSource = "KB9_CARE_GAPS"
	TaskSourceKB12   TaskSource = "KB12_ORDER_SETS"
	TaskSourceManual TaskSource = "MANUAL"
)

// Task represents a clinical task in KB-14
type Task struct {
	ID       uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TaskID   string     `gorm:"uniqueIndex;size:50;not null" json:"task_id"`
	Type     TaskType   `gorm:"size:50;not null;index" json:"type"`
	Status   TaskStatus `gorm:"size:30;not null;index;default:CREATED" json:"status"`
	Priority TaskPriority `gorm:"size:20;not null;index;default:MEDIUM" json:"priority"`
	Source   TaskSource `gorm:"size:30;not null;index" json:"source"`
	SourceID string     `gorm:"size:100;index" json:"source_id,omitempty"`

	// Patient Context
	PatientID   string `gorm:"size:50;not null;index" json:"patient_id"`
	EncounterID string `gorm:"size:50;index" json:"encounter_id,omitempty"`

	// Task Details
	Title        string `gorm:"size:200;not null" json:"title"`
	Description  string `gorm:"type:text" json:"description,omitempty"`
	Instructions string `gorm:"type:text" json:"instructions,omitempty"`
	ClinicalNote string `gorm:"type:text" json:"clinical_note,omitempty"`

	// Assignment
	AssignedTo   *uuid.UUID `gorm:"type:uuid;index" json:"assigned_to,omitempty"`
	AssignedRole string     `gorm:"size:50" json:"assigned_role,omitempty"`
	TeamID       *uuid.UUID `gorm:"type:uuid;index" json:"team_id,omitempty"`

	// SLA & Timing
	DueDate         *time.Time `gorm:"index" json:"due_date,omitempty"`
	SLAMinutes      int        `gorm:"default:0" json:"sla_minutes"`
	EscalationLevel int        `gorm:"default:0" json:"escalation_level"`

	// Completion
	CompletedBy *uuid.UUID `gorm:"type:uuid" json:"completed_by,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	VerifiedBy  *uuid.UUID `gorm:"type:uuid" json:"verified_by,omitempty"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	Outcome     string     `gorm:"size:50" json:"outcome,omitempty"`

	// Governance Fields (Tier-7 Compliance)
	ReasonCode            string     `gorm:"size:50" json:"reason_code,omitempty"`
	ReasonText            string     `gorm:"type:text" json:"reason_text,omitempty"`
	ClinicalJustification string     `gorm:"type:text" json:"clinical_justification,omitempty"`
	IntelligenceID        *uuid.UUID `gorm:"type:uuid;index" json:"intelligence_id,omitempty"`
	LastAuditAt           *time.Time `json:"last_audit_at,omitempty"`

	// Actions & Notes (JSONB)
	Actions  ActionSlice `gorm:"type:jsonb;default:'[]'" json:"actions,omitempty"`
	Notes    NoteSlice   `gorm:"type:jsonb;default:'[]'" json:"notes,omitempty"`
	Metadata JSONMap     `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt  time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	AssignedAt *time.Time `json:"assigned_at,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
}

// TableName returns the table name for Task
func (Task) TableName() string {
	return "tasks"
}

// TaskAction represents an action item within a task
type TaskAction struct {
	ActionID    string     `json:"action_id"`
	Type        string     `json:"type"`
	Description string     `json:"description"`
	Required    bool       `json:"required"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CompletedBy string     `json:"completed_by,omitempty"`
}

// TaskNote represents a note attached to a task
type TaskNote struct {
	NoteID    string    `json:"note_id"`
	Author    string    `json:"author"`
	AuthorID  string    `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// ActionSlice is a custom type for JSONB array of TaskAction
type ActionSlice []TaskAction

// Value implements the driver.Valuer interface
func (a ActionSlice) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	return json.Marshal(a)
}

// Scan implements the sql.Scanner interface
func (a *ActionSlice) Scan(value interface{}) error {
	if value == nil {
		*a = ActionSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("ActionSlice.Scan: unsupported type")
	}
	return json.Unmarshal(bytes, a)
}

// NoteSlice is a custom type for JSONB array of TaskNote
type NoteSlice []TaskNote

// Value implements the driver.Valuer interface
func (n NoteSlice) Value() (driver.Value, error) {
	if n == nil {
		return "[]", nil
	}
	return json.Marshal(n)
}

// Scan implements the sql.Scanner interface
func (n *NoteSlice) Scan(value interface{}) error {
	if value == nil {
		*n = NoteSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("NoteSlice.Scan: unsupported type")
	}
	return json.Unmarshal(bytes, n)
}

// JSONMap is a custom type for JSONB map
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface
func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = JSONMap{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("JSONMap.Scan: unsupported type")
	}
	return json.Unmarshal(bytes, m)
}

// CreateTaskRequest represents the request body for creating a task
type CreateTaskRequest struct {
	Type         TaskType              `json:"type" binding:"required"`
	Priority     TaskPriority          `json:"priority,omitempty"`
	Source       TaskSource            `json:"source" binding:"required"`
	SourceID     string                `json:"source_id,omitempty"`
	PatientID    string                `json:"patient_id" binding:"required"`
	EncounterID  string                `json:"encounter_id,omitempty"`
	Title        string                `json:"title" binding:"required"`
	Description  string                `json:"description,omitempty"`
	Instructions string                `json:"instructions,omitempty"`
	ClinicalNote string                `json:"clinical_note,omitempty"`
	DueDate      *time.Time            `json:"due_date,omitempty"`
	SLAMinutes   int                   `json:"sla_minutes,omitempty"`
	TeamID       *uuid.UUID            `json:"team_id,omitempty"`
	AssignedTo   *uuid.UUID            `json:"assigned_to,omitempty"`
	AssignedRole string                `json:"assigned_role,omitempty"`
	Actions      []TaskAction          `json:"actions,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateTaskRequest represents the request body for updating a task
type UpdateTaskRequest struct {
	Status                *TaskStatus            `json:"status,omitempty"`
	Priority              *TaskPriority          `json:"priority,omitempty"`
	Title                 *string                `json:"title,omitempty"`
	Description           *string                `json:"description,omitempty"`
	Instructions          *string                `json:"instructions,omitempty"`
	ClinicalNote          *string                `json:"clinical_note,omitempty"`
	DueDate               *time.Time             `json:"due_date,omitempty"`
	ReasonCode            string                 `json:"reason_code,omitempty"`
	ReasonText            string                 `json:"reason_text,omitempty"`
	ClinicalJustification string                 `json:"clinical_justification,omitempty"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
}

// AssignTaskRequest represents the request body for assigning a task
type AssignTaskRequest struct {
	AssigneeID uuid.UUID  `json:"assignee_id" binding:"required"`
	Role       string     `json:"role,omitempty"`
	TeamID     *uuid.UUID `json:"team_id,omitempty"` // Optional team override
}

// CompleteTaskRequest represents the request body for completing a task
type CompleteTaskRequest struct {
	Outcome               string `json:"outcome,omitempty"`
	Notes                 string `json:"notes,omitempty"`
	ReasonCode            string `json:"reason_code,omitempty"`
	ReasonText            string `json:"reason_text,omitempty"`
	ClinicalJustification string `json:"clinical_justification,omitempty"`
}

// DeclineTaskRequest represents the request body for declining a task
type DeclineTaskRequest struct {
	ReasonCode            string `json:"reason_code" binding:"required"`
	ReasonText            string `json:"reason_text,omitempty"`
	ClinicalJustification string `json:"clinical_justification,omitempty"`
}

// CancelTaskRequest represents the request body for cancelling a task
type CancelTaskRequest struct {
	ReasonCode            string `json:"reason_code" binding:"required"`
	ReasonText            string `json:"reason_text,omitempty"`
	ClinicalJustification string `json:"clinical_justification,omitempty"`
}

// AddNoteRequest represents the request body for adding a note
type AddNoteRequest struct {
	Content  string `json:"content" binding:"required"`
	AuthorID string `json:"author_id" binding:"required"`
	Author   string `json:"author" binding:"required"`
}

// TaskResponse wraps a task for API responses
type TaskResponse struct {
	Success bool   `json:"success"`
	Data    *Task  `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// TaskListResponse wraps a list of tasks for API responses
type TaskListResponse struct {
	Success bool    `json:"success"`
	Data    []Task  `json:"data,omitempty"`
	Total   int64   `json:"total"`
	Error   string  `json:"error,omitempty"`
}

// IsOverdue checks if the task is past its due date
func (t *Task) IsOverdue() bool {
	if t.DueDate == nil {
		return false
	}
	return t.DueDate.Before(time.Now().UTC())
}

// IsDueSoon checks if the task is due within the specified hours
func (t *Task) IsDueSoon(hoursAhead int) bool {
	if t.DueDate == nil {
		return false
	}
	deadline := time.Now().UTC().Add(time.Duration(hoursAhead) * time.Hour)
	return t.DueDate.Before(deadline) && !t.IsOverdue()
}

// GetSLAElapsedPercent calculates the percentage of SLA time elapsed
// Note: This can return values > 100% for overdue tasks, which is needed
// for escalation level calculation (e.g., 125% triggers Executive escalation)
func (t *Task) GetSLAElapsedPercent() float64 {
	if t.SLAMinutes <= 0 {
		return 0
	}
	elapsed := time.Since(t.CreatedAt).Minutes()
	percent := (elapsed / float64(t.SLAMinutes)) * 100
	return percent
}
