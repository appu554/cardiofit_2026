// Package models contains governance and audit models for KB-14 Care Navigator
package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// AUDIT EVENT MODELS
// =============================================================================

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	// Task Lifecycle Events
	AuditEventCreated    AuditEventType = "CREATED"
	AuditEventAssigned   AuditEventType = "ASSIGNED"
	AuditEventReassigned AuditEventType = "REASSIGNED"
	AuditEventStarted    AuditEventType = "STARTED"
	AuditEventPaused     AuditEventType = "PAUSED"
	AuditEventResumed    AuditEventType = "RESUMED"
	AuditEventCompleted  AuditEventType = "COMPLETED"
	AuditEventVerified   AuditEventType = "VERIFIED"
	AuditEventDeclined   AuditEventType = "DECLINED"
	AuditEventCancelled  AuditEventType = "CANCELLED"

	// Escalation Events
	AuditEventEscalated            AuditEventType = "ESCALATED"
	AuditEventEscalationAcknowledged AuditEventType = "ESCALATION_ACKNOWLEDGED"
	AuditEventEscalationResolved   AuditEventType = "ESCALATION_RESOLVED"

	// Modification Events
	AuditEventPriorityChanged AuditEventType = "PRIORITY_CHANGED"
	AuditEventDueDateChanged  AuditEventType = "DUE_DATE_CHANGED"
	AuditEventNoteAdded       AuditEventType = "NOTE_ADDED"
	AuditEventActionCompleted AuditEventType = "ACTION_COMPLETED"

	// Governance Events
	AuditEventSLAWarning    AuditEventType = "SLA_WARNING"
	AuditEventSLABreach     AuditEventType = "SLA_BREACH"
	AuditEventComplianceCheck AuditEventType = "COMPLIANCE_CHECK"
)

// AuditEventCategory represents the category of audit event
type AuditEventCategory string

const (
	AuditCategoryLifecycle   AuditEventCategory = "LIFECYCLE"
	AuditCategoryAssignment  AuditEventCategory = "ASSIGNMENT"
	AuditCategoryEscalation  AuditEventCategory = "ESCALATION"
	AuditCategoryModification AuditEventCategory = "MODIFICATION"
	AuditCategoryGovernance  AuditEventCategory = "GOVERNANCE"
)

// GetEventCategory returns the category for an event type
func (e AuditEventType) GetEventCategory() AuditEventCategory {
	switch e {
	case AuditEventCreated, AuditEventStarted, AuditEventPaused, AuditEventResumed,
		AuditEventCompleted, AuditEventVerified, AuditEventDeclined, AuditEventCancelled:
		return AuditCategoryLifecycle
	case AuditEventAssigned, AuditEventReassigned:
		return AuditCategoryAssignment
	case AuditEventEscalated, AuditEventEscalationAcknowledged, AuditEventEscalationResolved:
		return AuditCategoryEscalation
	case AuditEventPriorityChanged, AuditEventDueDateChanged, AuditEventNoteAdded, AuditEventActionCompleted:
		return AuditCategoryModification
	case AuditEventSLAWarning, AuditEventSLABreach, AuditEventComplianceCheck:
		return AuditCategoryGovernance
	default:
		return AuditCategoryModification
	}
}

// ActorType represents the type of actor performing an action
type ActorType string

const (
	ActorTypeUser        ActorType = "USER"
	ActorTypeSystem      ActorType = "SYSTEM"
	ActorTypeWorker      ActorType = "WORKER"
	ActorTypeIntegration ActorType = "INTEGRATION"
)

// TaskAuditLog represents an immutable audit record
type TaskAuditLog struct {
	ID             uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SequenceNumber int64              `gorm:"autoIncrement" json:"sequence_number"`

	// Task Reference
	TaskID     uuid.UUID `gorm:"type:uuid;not null;index" json:"task_id"`
	TaskNumber string    `gorm:"size:50;not null" json:"task_number"`

	// Event Details
	EventType     AuditEventType     `gorm:"size:50;not null;index" json:"event_type"`
	EventCategory AuditEventCategory `gorm:"size:30;not null" json:"event_category"`

	// State Change
	PreviousStatus *TaskStatus `gorm:"size:30" json:"previous_status,omitempty"`
	NewStatus      *TaskStatus `gorm:"size:30" json:"new_status,omitempty"`
	PreviousValue  JSONMap     `gorm:"type:jsonb;default:'{}'" json:"previous_value,omitempty"`
	NewValue       JSONMap     `gorm:"type:jsonb;default:'{}'" json:"new_value,omitempty"`

	// Actor Information
	ActorID   *uuid.UUID `gorm:"type:uuid" json:"actor_id,omitempty"`
	ActorType ActorType  `gorm:"size:30;not null" json:"actor_type"`
	ActorName string     `gorm:"size:100" json:"actor_name,omitempty"`
	ActorRole string     `gorm:"size:50" json:"actor_role,omitempty"`

	// Clinical Context
	PatientID   string `gorm:"size:50;not null;index" json:"patient_id"`
	EncounterID string `gorm:"size:50" json:"encounter_id,omitempty"`

	// Source Information
	SourceService string `gorm:"size:50" json:"source_service,omitempty"`
	SourceEventID string `gorm:"size:100" json:"source_event_id,omitempty"`

	// Governance Fields
	ReasonCode            string `gorm:"size:50" json:"reason_code,omitempty"`
	ReasonText            string `gorm:"type:text" json:"reason_text,omitempty"`
	ClinicalJustification string `gorm:"type:text" json:"clinical_justification,omitempty"`

	// Evidence Snapshot
	EvidenceSnapshot JSONMap `gorm:"type:jsonb;default:'{}'" json:"evidence_snapshot,omitempty"`

	// Hash Chain
	PreviousHash string `gorm:"size:64" json:"previous_hash,omitempty"`
	RecordHash   string `gorm:"size:64;not null" json:"record_hash"`

	// Timestamps
	EventTimestamp time.Time `gorm:"not null;default:NOW()" json:"event_timestamp"`
	RecordedAt     time.Time `gorm:"autoCreateTime" json:"recorded_at"`

	// Request Metadata
	IPAddress string  `gorm:"size:45" json:"ip_address,omitempty"`
	UserAgent string  `gorm:"type:text" json:"user_agent,omitempty"`
	SessionID string  `gorm:"size:100" json:"session_id,omitempty"`
	Metadata  JSONMap `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
}

// TableName returns the table name for TaskAuditLog
func (TaskAuditLog) TableName() string {
	return "task_audit_log"
}

// CalculateHash calculates the SHA-256 hash for this record
func (a *TaskAuditLog) CalculateHash() string {
	// Always use UTC and truncate to microseconds for consistent hashing
	// PostgreSQL stores with microsecond precision, and GORM may return in local timezone
	timestamp := a.EventTimestamp.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)

	content := fmt.Sprintf("%s|%s|%v|%v|%v|%s|%s",
		a.TaskID.String(),
		string(a.EventType),
		a.ActorID,
		a.PreviousStatus,
		a.NewStatus,
		timestamp,
		a.PreviousHash,
	)

	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// =============================================================================
// GOVERNANCE EVENT MODELS
// =============================================================================

// GovernanceEventType represents the type of governance event
type GovernanceEventType string

const (
	GovernanceEventComplianceCheck GovernanceEventType = "COMPLIANCE_CHECK"
	GovernanceEventAuditRequired   GovernanceEventType = "AUDIT_REQUIRED"
	GovernanceEventPolicyViolation GovernanceEventType = "POLICY_VIOLATION"
	GovernanceEventSLABreach       GovernanceEventType = "SLA_BREACH"
	GovernanceEventEscalationAlert GovernanceEventType = "ESCALATION_ALERT"
	GovernanceEventIntelligenceGap GovernanceEventType = "INTELLIGENCE_GAP"
)

// GovernanceSeverity represents the severity level
type GovernanceSeverity string

const (
	GovernanceSeverityInfo     GovernanceSeverity = "INFO"
	GovernanceSeverityWarning  GovernanceSeverity = "WARNING"
	GovernanceSeverityCritical GovernanceSeverity = "CRITICAL"
	GovernanceSeverityAlert    GovernanceSeverity = "ALERT"
)

// GovernanceEvent represents a governance event for Tier-7 compliance
type GovernanceEvent struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// Event Classification
	EventType GovernanceEventType `gorm:"size:50;not null;index" json:"event_type"`
	Severity  GovernanceSeverity  `gorm:"size:20;not null;index" json:"severity"`

	// Context
	TaskID         *uuid.UUID `gorm:"type:uuid;index" json:"task_id,omitempty"`
	PatientID      string     `gorm:"size:50;index" json:"patient_id,omitempty"`
	OrganizationID string     `gorm:"size:50" json:"organization_id,omitempty"`

	// Event Details
	Title       string `gorm:"size:200;not null" json:"title"`
	Description string `gorm:"type:text" json:"description,omitempty"`

	// Governance Metrics
	ComplianceScore *float64 `gorm:"type:decimal(5,2)" json:"compliance_score,omitempty"`
	RiskScore       *float64 `gorm:"type:decimal(5,2)" json:"risk_score,omitempty"`

	// Resolution
	RequiresAction  bool       `gorm:"default:false" json:"requires_action"`
	ActionDeadline  *time.Time `json:"action_deadline,omitempty"`
	Resolved        bool       `gorm:"default:false;index" json:"resolved"`
	ResolvedBy      *uuid.UUID `gorm:"type:uuid" json:"resolved_by,omitempty"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	ResolutionNotes string     `gorm:"type:text" json:"resolution_notes,omitempty"`

	// Audit Trail
	TriggeredBy   string `gorm:"size:50;not null" json:"triggered_by"`
	TriggeredByID string `gorm:"size:100" json:"triggered_by_id,omitempty"`

	// Evidence
	Evidence JSONMap `gorm:"type:jsonb;default:'{}'" json:"evidence,omitempty"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for GovernanceEvent
func (GovernanceEvent) TableName() string {
	return "governance_events"
}

// =============================================================================
// REASON CODE MODELS
// =============================================================================

// ReasonCodeCategory represents the category of reason code
type ReasonCodeCategory string

const (
	ReasonCategoryAcceptance   ReasonCodeCategory = "ACCEPTANCE"
	ReasonCategoryRejection    ReasonCodeCategory = "REJECTION"
	ReasonCategoryEscalation   ReasonCodeCategory = "ESCALATION"
	ReasonCategoryCompletion   ReasonCodeCategory = "COMPLETION"
	ReasonCategoryCancellation ReasonCodeCategory = "CANCELLATION"
)

// ReasonCode represents a standardized reason code
type ReasonCode struct {
	Code                       string             `gorm:"primaryKey;size:50" json:"code"`
	Category                   ReasonCodeCategory `gorm:"size:30;not null" json:"category"`
	DisplayName                string             `gorm:"size:100;not null" json:"display_name"`
	Description                string             `gorm:"type:text" json:"description,omitempty"`
	RequiresJustification      bool               `gorm:"default:false" json:"requires_justification"`
	RequiresSupervisorApproval bool               `gorm:"default:false" json:"requires_supervisor_approval"`
	IsActive                   bool               `gorm:"default:true" json:"is_active"`
	SortOrder                  int                `gorm:"default:0" json:"sort_order"`
	CreatedAt                  time.Time          `gorm:"autoCreateTime" json:"created_at"`
}

// TableName returns the table name for ReasonCode
func (ReasonCode) TableName() string {
	return "reason_codes"
}

// =============================================================================
// INTELLIGENCE TRACKING MODELS
// =============================================================================

// IntelligenceStatus represents the processing status
type IntelligenceStatus string

const (
	IntelligenceStatusReceived    IntelligenceStatus = "RECEIVED"
	IntelligenceStatusProcessed   IntelligenceStatus = "PROCESSED"
	IntelligenceStatusTaskCreated IntelligenceStatus = "TASK_CREATED"
	IntelligenceStatusDeclined    IntelligenceStatus = "DECLINED"
	IntelligenceStatusError       IntelligenceStatus = "ERROR"
)

// IntelligenceSourceType represents the type of intelligence source
type IntelligenceSourceType string

const (
	IntelligenceSourceTemporalAlert   IntelligenceSourceType = "TEMPORAL_ALERT"
	IntelligenceSourceCareGap         IntelligenceSourceType = "CARE_GAP"
	IntelligenceSourceCarePlanActivity IntelligenceSourceType = "CARE_PLAN_ACTIVITY"
	IntelligenceSourceProtocolStep    IntelligenceSourceType = "PROTOCOL_STEP"
)

// IntelligenceTracking tracks all incoming intelligence
type IntelligenceTracking struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// Source Intelligence
	SourceService string                 `gorm:"size:50;not null;index" json:"source_service"`
	SourceID      string                 `gorm:"size:100;not null" json:"source_id"`
	SourceType    IntelligenceSourceType `gorm:"size:50;not null" json:"source_type"`

	// Patient Context
	PatientID string `gorm:"size:50;not null;index" json:"patient_id"`

	// Processing Status
	Status IntelligenceStatus `gorm:"size:30;not null;index" json:"status"`

	// Task Linkage
	TaskID *uuid.UUID `gorm:"type:uuid;index" json:"task_id,omitempty"`

	// Disposition (if not creating task)
	DispositionCode   string     `gorm:"size:50" json:"disposition_code,omitempty"`
	DispositionReason string     `gorm:"type:text" json:"disposition_reason,omitempty"`
	DispositionBy     *uuid.UUID `gorm:"type:uuid" json:"disposition_by,omitempty"`
	DispositionAt     *time.Time `json:"disposition_at,omitempty"`

	// Evidence
	IntelligenceSnapshot JSONMap `gorm:"type:jsonb;not null" json:"intelligence_snapshot"`

	// Timestamps
	ReceivedAt  time.Time  `gorm:"autoCreateTime" json:"received_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

// TableName returns the table name for IntelligenceTracking
func (IntelligenceTracking) TableName() string {
	return "intelligence_tracking"
}

// =============================================================================
// AUDIT REQUEST/RESPONSE MODELS
// =============================================================================

// CreateAuditLogRequest represents a request to create an audit log entry
type CreateAuditLogRequest struct {
	TaskID                uuid.UUID          `json:"task_id" binding:"required"`
	TaskNumber            string             `json:"task_number" binding:"required"`
	EventType             AuditEventType     `json:"event_type" binding:"required"`
	PreviousStatus        *TaskStatus        `json:"previous_status,omitempty"`
	NewStatus             *TaskStatus        `json:"new_status,omitempty"`
	PreviousValue         map[string]interface{} `json:"previous_value,omitempty"`
	NewValue              map[string]interface{} `json:"new_value,omitempty"`
	ActorID               *uuid.UUID         `json:"actor_id,omitempty"`
	ActorType             ActorType          `json:"actor_type" binding:"required"`
	ActorName             string             `json:"actor_name,omitempty"`
	ActorRole             string             `json:"actor_role,omitempty"`
	PatientID             string             `json:"patient_id" binding:"required"`
	EncounterID           string             `json:"encounter_id,omitempty"`
	SourceService         string             `json:"source_service,omitempty"`
	SourceEventID         string             `json:"source_event_id,omitempty"`
	ReasonCode            string             `json:"reason_code,omitempty"`
	ReasonText            string             `json:"reason_text,omitempty"`
	ClinicalJustification string             `json:"clinical_justification,omitempty"`
	EvidenceSnapshot      map[string]interface{} `json:"evidence_snapshot,omitempty"`
	IPAddress             string             `json:"ip_address,omitempty"`
	UserAgent             string             `json:"user_agent,omitempty"`
	SessionID             string             `json:"session_id,omitempty"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
}

// AuditLogQuery represents query parameters for audit log search
type AuditLogQuery struct {
	TaskID         *uuid.UUID         `json:"task_id,omitempty"`
	PatientID      string             `json:"patient_id,omitempty"`
	ActorID        *uuid.UUID         `json:"actor_id,omitempty"`
	EventType      *AuditEventType    `json:"event_type,omitempty"`
	EventCategory  *AuditEventCategory `json:"event_category,omitempty"`
	StartDate      *time.Time         `json:"start_date,omitempty"`
	EndDate        *time.Time         `json:"end_date,omitempty"`
	Limit          int                `json:"limit,omitempty"`
	Offset         int                `json:"offset,omitempty"`
}

// AuditLogResponse wraps audit log entries for API responses
type AuditLogResponse struct {
	Success bool           `json:"success"`
	Data    []TaskAuditLog `json:"data,omitempty"`
	Total   int64          `json:"total"`
	Error   string         `json:"error,omitempty"`
}

// GovernanceEventQuery represents query parameters for governance events
type GovernanceEventQuery struct {
	EventType    *GovernanceEventType `json:"event_type,omitempty"`
	Severity     *GovernanceSeverity  `json:"severity,omitempty"`
	TaskID       *uuid.UUID           `json:"task_id,omitempty"`
	PatientID    string               `json:"patient_id,omitempty"`
	Resolved     *bool                `json:"resolved,omitempty"`
	RequiresAction *bool              `json:"requires_action,omitempty"`
	StartDate    *time.Time           `json:"start_date,omitempty"`
	EndDate      *time.Time           `json:"end_date,omitempty"`
	Limit        int                  `json:"limit,omitempty"`
	Offset       int                  `json:"offset,omitempty"`
}

// GovernanceEventResponse wraps governance events for API responses
type GovernanceEventResponse struct {
	Success bool              `json:"success"`
	Data    []GovernanceEvent `json:"data,omitempty"`
	Total   int64             `json:"total"`
	Error   string            `json:"error,omitempty"`
}

// =============================================================================
// SUMMARY MODELS
// =============================================================================

// AuditSummary provides summary statistics for audit logs
type AuditSummary struct {
	TaskID         uuid.UUID           `json:"task_id"`
	TaskNumber     string              `json:"task_number"`
	PatientID      string              `json:"patient_id"`
	TaskType       TaskType            `json:"task_type"`
	CurrentStatus  TaskStatus          `json:"current_status"`
	TotalEvents    int64               `json:"total_events"`
	FirstEvent     time.Time           `json:"first_event"`
	LastEvent      time.Time           `json:"last_event"`
	UniqueActors   int                 `json:"unique_actors"`
	EventTypes     []AuditEventType    `json:"event_types"`
	HasReasonCodes bool                `json:"has_reason_codes"`
}

// GovernanceDashboard provides dashboard statistics
type GovernanceDashboard struct {
	Date               string             `json:"date"`
	EventType          GovernanceEventType `json:"event_type"`
	Severity           GovernanceSeverity `json:"severity"`
	EventCount         int64              `json:"event_count"`
	ResolvedCount      int64              `json:"resolved_count"`
	PendingActionCount int64              `json:"pending_action_count"`
	AvgComplianceScore *float64           `json:"avg_compliance_score,omitempty"`
	AvgRiskScore       *float64           `json:"avg_risk_score,omitempty"`
}

// IntelligenceAccountability provides intelligence tracking statistics
type IntelligenceAccountability struct {
	SourceService string             `json:"source_service"`
	SourceType    IntelligenceSourceType `json:"source_type"`
	Status        IntelligenceStatus `json:"status"`
	Count         int64              `json:"count"`
	TasksCreated  int64              `json:"tasks_created"`
	Dispositioned int64              `json:"dispositioned"`
	Pending       int64              `json:"pending"`
}
