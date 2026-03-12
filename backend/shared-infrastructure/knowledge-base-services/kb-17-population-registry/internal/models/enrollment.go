// Package models contains domain models for KB-17 Population Registry
package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// EnrollmentStatus represents the lifecycle status of an enrollment
type EnrollmentStatus string

const (
	EnrollmentStatusPending     EnrollmentStatus = "PENDING"
	EnrollmentStatusActive      EnrollmentStatus = "ACTIVE"
	EnrollmentStatusSuspended   EnrollmentStatus = "SUSPENDED"
	EnrollmentStatusDisenrolled EnrollmentStatus = "DISENROLLED"
)

// IsValid checks if the enrollment status is valid
func (s EnrollmentStatus) IsValid() bool {
	switch s {
	case EnrollmentStatusPending, EnrollmentStatusActive,
		EnrollmentStatusSuspended, EnrollmentStatusDisenrolled:
		return true
	}
	return false
}

// IsActive returns true if status represents an active enrollment
func (s EnrollmentStatus) IsActive() bool {
	return s == EnrollmentStatusActive || s == EnrollmentStatusPending
}

// EnrollmentSource represents how the patient was enrolled
type EnrollmentSource string

const (
	EnrollmentSourceDiagnosis   EnrollmentSource = "DIAGNOSIS"
	EnrollmentSourceLabResult   EnrollmentSource = "LAB_RESULT"
	EnrollmentSourceMedication  EnrollmentSource = "MEDICATION"
	EnrollmentSourceProblemList EnrollmentSource = "PROBLEM_LIST"
	EnrollmentSourceManual      EnrollmentSource = "MANUAL"
	EnrollmentSourceBulk        EnrollmentSource = "BULK"
	EnrollmentSourceMigration   EnrollmentSource = "MIGRATION"
)

// IsValid checks if the enrollment source is valid
func (s EnrollmentSource) IsValid() bool {
	switch s {
	case EnrollmentSourceDiagnosis, EnrollmentSourceLabResult,
		EnrollmentSourceMedication, EnrollmentSourceProblemList,
		EnrollmentSourceManual, EnrollmentSourceBulk, EnrollmentSourceMigration:
		return true
	}
	return false
}

// RegistryPatient represents a patient's enrollment in a registry
type RegistryPatient struct {
	ID               uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RegistryCode     RegistryCode     `gorm:"size:50;not null;index" json:"registry_code"`
	PatientID        string           `gorm:"size:50;not null;index" json:"patient_id"`
	Status           EnrollmentStatus `gorm:"size:30;not null;index;default:PENDING" json:"status"`
	EnrollmentSource EnrollmentSource `gorm:"size:30;not null" json:"enrollment_source"`
	SourceEventID    string           `gorm:"size:100" json:"source_event_id,omitempty"`
	RiskTier         RiskTier         `gorm:"size:20;not null;default:MODERATE;index" json:"risk_tier"`

	// Clinical Metrics (stored as JSONB for flexibility)
	Metrics          MetricMapSlice   `gorm:"type:jsonb;default:'{}'" json:"metrics,omitempty"`

	// Care Gaps linked to KB-9
	CareGaps         StringSlice      `gorm:"type:jsonb;default:'[]'" json:"care_gaps,omitempty"`

	// Enrollment Timeline
	EnrolledAt       time.Time        `gorm:"not null;default:now()" json:"enrolled_at"`
	LastEvaluatedAt  *time.Time       `json:"last_evaluated_at,omitempty"`
	DisenrolledAt    *time.Time       `json:"disenrolled_at,omitempty"`
	DisenrollReason  string           `gorm:"type:text" json:"disenroll_reason,omitempty"`
	DisenrolledBy    string           `gorm:"size:50" json:"disenrolled_by,omitempty"`

	// Additional Context
	EnrolledBy       string           `gorm:"size:50" json:"enrolled_by,omitempty"`
	Notes            string           `gorm:"type:text" json:"notes,omitempty"`
	Metadata         JSONMap          `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt        time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for RegistryPatient
func (RegistryPatient) TableName() string {
	return "registry_patients"
}

// MetricValue represents a clinical metric value with context
type MetricValue struct {
	Value       interface{} `json:"value"`
	Unit        string      `json:"unit,omitempty"`
	EffectiveAt time.Time   `json:"effective_at"`
	Source      string      `json:"source,omitempty"`
	SourceID    string      `json:"source_id,omitempty"`
}

// MetricMapSlice is a custom type for JSONB map of metric values
type MetricMapSlice map[string]*MetricValue

// Value implements the driver.Valuer interface
func (m MetricMapSlice) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface
func (m *MetricMapSlice) Scan(value interface{}) error {
	if value == nil {
		*m = MetricMapSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("MetricMapSlice.Scan: unsupported type")
	}
	return json.Unmarshal(bytes, m)
}

// EnrollmentHistory tracks changes to enrollments
type EnrollmentHistory struct {
	ID           uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	EnrollmentID uuid.UUID        `gorm:"type:uuid;not null;index" json:"enrollment_id"`
	Action       string           `gorm:"size:30;not null" json:"action"`
	OldStatus    EnrollmentStatus `gorm:"size:30" json:"old_status,omitempty"`
	NewStatus    EnrollmentStatus `gorm:"size:30" json:"new_status,omitempty"`
	OldRiskTier  RiskTier         `gorm:"size:20" json:"old_risk_tier,omitempty"`
	NewRiskTier  RiskTier         `gorm:"size:20" json:"new_risk_tier,omitempty"`
	Reason       string           `gorm:"type:text" json:"reason,omitempty"`
	ActorID      string           `gorm:"size:50" json:"actor_id,omitempty"`
	Metadata     JSONMap          `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
	CreatedAt    time.Time        `gorm:"autoCreateTime" json:"created_at"`
}

// TableName returns the table name for EnrollmentHistory
func (EnrollmentHistory) TableName() string {
	return "enrollment_history"
}

// History action constants
const (
	HistoryActionEnrolled     = "ENROLLED"
	HistoryActionDisenrolled  = "DISENROLLED"
	HistoryActionSuspended    = "SUSPENDED"
	HistoryActionReactivated  = "REACTIVATED"
	HistoryActionRiskChanged  = "RISK_CHANGED"
	HistoryActionMetricUpdate = "METRIC_UPDATE"
	HistoryActionCareGapAdded = "CARE_GAP_ADDED"
	HistoryActionCareGapClosed = "CARE_GAP_CLOSED"
)

// EnrollRequest represents the request body for enrolling a patient
type EnrollRequest struct {
	RegistryCode     RegistryCode     `json:"registry_code" binding:"required"`
	PatientID        string           `json:"patient_id" binding:"required"`
	EnrollmentSource EnrollmentSource `json:"enrollment_source" binding:"required"`
	SourceEventID    string           `json:"source_event_id,omitempty"`
	RiskTier         RiskTier         `json:"risk_tier,omitempty"`
	Metrics          map[string]*MetricValue `json:"metrics,omitempty"`
	Notes            string           `json:"notes,omitempty"`
	EnrolledBy       string           `json:"enrolled_by,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// BulkEnrollRequest represents the request body for bulk enrollment
type BulkEnrollRequest struct {
	RegistryCode     RegistryCode     `json:"registry_code" binding:"required"`
	PatientIDs       []string         `json:"patient_ids" binding:"required"`
	EnrollmentSource EnrollmentSource `json:"enrollment_source" binding:"required"`
	RiskTier         RiskTier         `json:"risk_tier,omitempty"`
	EnrolledBy       string           `json:"enrolled_by,omitempty"`
}

// UpdateEnrollmentRequest represents the request body for updating an enrollment
type UpdateEnrollmentRequest struct {
	Status      *EnrollmentStatus       `json:"status,omitempty"`
	RiskTier    *RiskTier               `json:"risk_tier,omitempty"`
	Metrics     map[string]*MetricValue `json:"metrics,omitempty"`
	CareGaps    []string                `json:"care_gaps,omitempty"`
	Notes       *string                 `json:"notes,omitempty"`
	Metadata    map[string]interface{}  `json:"metadata,omitempty"`
}

// DisenrollRequest represents the request body for disenrolling a patient
type DisenrollRequest struct {
	Reason       string `json:"reason" binding:"required"`
	DisenrolledBy string `json:"disenrolled_by,omitempty"`
}

// EnrollmentResponse wraps an enrollment for API responses
type EnrollmentResponse struct {
	Success bool             `json:"success"`
	Data    *RegistryPatient `json:"data,omitempty"`
	Error   string           `json:"error,omitempty"`
}

// EnrollmentListResponse wraps a list of enrollments for API responses
type EnrollmentListResponse struct {
	Success bool              `json:"success"`
	Data    []RegistryPatient `json:"data,omitempty"`
	Total   int64             `json:"total"`
	Error   string            `json:"error,omitempty"`
}

// BulkEnrollmentResult represents the result of a bulk enrollment
type BulkEnrollmentResult struct {
	Success   int      `json:"success_count"`
	Failed    int      `json:"failed_count"`
	Errors    []string `json:"errors,omitempty"`
	Enrolled  []string `json:"enrolled_patient_ids"`
	Skipped   []string `json:"skipped_patient_ids,omitempty"`
}

// BulkEnrollmentResponse wraps bulk enrollment results
type BulkEnrollmentResponse struct {
	Success bool                  `json:"success"`
	Data    *BulkEnrollmentResult `json:"data,omitempty"`
	Error   string                `json:"error,omitempty"`
}

// EnrollmentQuery represents query parameters for listing enrollments
type EnrollmentQuery struct {
	RegistryCode RegistryCode     `form:"registry_code"`
	PatientID    string           `form:"patient_id"`
	Status       EnrollmentStatus `form:"status"`
	RiskTier     RiskTier         `form:"risk_tier"`
	HasCareGaps  *bool            `form:"has_care_gaps"`
	Limit        int              `form:"limit,default=50"`
	Offset       int              `form:"offset,default=0"`
	SortBy       string           `form:"sort_by,default=enrolled_at"`
	SortOrder    string           `form:"sort_order,default=desc"`
}

// IsHighRisk returns true if the patient is high or critical risk
func (rp *RegistryPatient) IsHighRisk() bool {
	return rp.RiskTier == RiskTierHigh || rp.RiskTier == RiskTierCritical
}

// HasCareGaps returns true if the patient has any care gaps
func (rp *RegistryPatient) HasCareGaps() bool {
	return len(rp.CareGaps) > 0
}

// GetMetricValue retrieves a specific metric value
func (rp *RegistryPatient) GetMetricValue(key string) *MetricValue {
	if rp.Metrics == nil {
		return nil
	}
	return rp.Metrics[key]
}

// SetMetricValue sets a metric value
func (rp *RegistryPatient) SetMetricValue(key string, value *MetricValue) {
	if rp.Metrics == nil {
		rp.Metrics = make(MetricMapSlice)
	}
	rp.Metrics[key] = value
}
