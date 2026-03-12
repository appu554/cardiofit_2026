// Package models defines data structures for KB-12 Order Sets & Care Plans
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CPOEDraftStatus represents the status of a CPOE draft session
type CPOEDraftStatus string

const (
	CPOEDraftStatusDraft     CPOEDraftStatus = "draft"
	CPOEDraftStatusSubmitted CPOEDraftStatus = "submitted"
	CPOEDraftStatusCancelled CPOEDraftStatus = "cancelled"
	CPOEDraftStatusExpired   CPOEDraftStatus = "expired"
)

// CPOEDraftSession represents a CPOE draft session for order review
type CPOEDraftSession struct {
	ID           uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID    string           `gorm:"uniqueIndex;size:50;not null" json:"session_id"`
	PatientID    string           `gorm:"size:50;not null;index" json:"patient_id"`
	ProviderID   string           `gorm:"size:100;not null;index" json:"provider_id"`
	EncounterID  string           `gorm:"size:50;index" json:"encounter_id,omitempty"`
	Orders       OrderSlice       `gorm:"type:jsonb;not null;default:'[]'" json:"orders"`
	SafetyAlerts SafetyAlertSlice `gorm:"type:jsonb" json:"safety_alerts,omitempty"`
	Status       CPOEDraftStatus  `gorm:"size:20;not null;index" json:"status"`
	CreatedAt    time.Time        `gorm:"autoCreateTime" json:"created_at"`
	ExpiresAt    time.Time        `json:"expires_at"`
	SubmittedAt  *time.Time       `json:"submitted_at,omitempty"`
}

// TableName specifies the table name for CPOEDraftSession
func (CPOEDraftSession) TableName() string {
	return "cpoe_draft_sessions"
}

// SubmittedOrder represents a submitted order
type SubmittedOrder struct {
	ID             uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID        string      `gorm:"uniqueIndex;size:50;not null" json:"order_id"`
	PatientID      string      `gorm:"size:50;not null;index" json:"patient_id"`
	EncounterID    string      `gorm:"size:50;not null;index" json:"encounter_id"`
	ProviderID     string      `gorm:"size:100;not null;index" json:"provider_id"`
	OrderType      OrderType   `gorm:"size:50;not null;index" json:"order_type"`
	OrderData      JSONMap     `gorm:"type:jsonb;not null" json:"order_data"`
	Status         OrderStatus `gorm:"size:20;not null;index" json:"status"`
	SourceTemplate string      `gorm:"size:50" json:"source_template,omitempty"`
	SourceInstance string      `gorm:"size:50" json:"source_instance,omitempty"`
	CreatedAt      time.Time   `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time   `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for SubmittedOrder
func (SubmittedOrder) TableName() string {
	return "submitted_orders"
}

// OrderSetActivationRequest represents a request to activate an order set
type OrderSetActivationRequest struct {
	TemplateID   string            `json:"template_id" binding:"required"`
	PatientID    string            `json:"patient_id" binding:"required"`
	EncounterID  string            `json:"encounter_id" binding:"required"`
	ActivatedBy  string            `json:"activated_by" binding:"required"`
	SelectedOrders []string        `json:"selected_orders,omitempty"` // Order IDs to include
	Customizations map[string]interface{} `json:"customizations,omitempty"`
	PatientContext PatientContext  `json:"patient_context,omitempty"`
}

// PatientContext represents patient context for order customization
type PatientContext struct {
	Weight        float64           `json:"weight_kg,omitempty"`
	Height        float64           `json:"height_cm,omitempty"`
	Age           int               `json:"age_years,omitempty"`
	Gender        string            `json:"gender,omitempty"`
	BSA           float64           `json:"bsa_m2,omitempty"`
	CrCl          float64           `json:"crcl_ml_min,omitempty"`
	RenalFunction string            `json:"renal_function,omitempty"` // normal, mild, moderate, severe, esrd
	HepaticFunction string          `json:"hepatic_function,omitempty"`
	Allergies     []string          `json:"allergies,omitempty"`
	Diagnoses     []CodeReference   `json:"diagnoses,omitempty"`
	ActiveMeds    []string          `json:"active_medications,omitempty"`
	LabValues     map[string]float64 `json:"lab_values,omitempty"`
}

// OrderSetActivationResponse represents the response from activating an order set
type OrderSetActivationResponse struct {
	Success       bool                `json:"success"`
	InstanceID    string              `json:"instance_id"`
	TemplateID    string              `json:"template_id"`
	TemplateName  string              `json:"template_name"`
	PatientID     string              `json:"patient_id"`
	EncounterID   string              `json:"encounter_id"`
	Orders        []Order             `json:"orders"`
	Constraints   []ConstraintStatus  `json:"constraints,omitempty"`
	SafetyAlerts  []SafetyAlert       `json:"safety_alerts,omitempty"`
	Warnings      []string            `json:"warnings,omitempty"`
	ActivatedAt   time.Time           `json:"activated_at"`
	ErrorMessage  string              `json:"error_message,omitempty"`
}

// SafetyAlert represents a clinical safety alert
type SafetyAlert struct {
	AlertID       string   `json:"alert_id"`
	AlertType     string   `json:"alert_type"` // allergy, interaction, duplicate, dose_range, contraindication
	Severity      string   `json:"severity"`   // high, medium, low
	Message       string   `json:"message"`
	Details       string   `json:"details,omitempty"`
	SourceOrder   string   `json:"source_order,omitempty"`
	AffectedOrders []string `json:"affected_orders,omitempty"`
	Overridable   bool     `json:"overridable"`
	OverrideReason string  `json:"override_reason,omitempty"`
	Reference     string   `json:"reference,omitempty"`
}

// CarePlanActivationRequest represents a request to activate a care plan
type CarePlanActivationRequest struct {
	PlanID       string            `json:"plan_id" binding:"required"`
	PatientID    string            `json:"patient_id" binding:"required"`
	StartDate    time.Time         `json:"start_date"`
	EndDate      *time.Time        `json:"end_date,omitempty"`
	Customizations map[string]interface{} `json:"customizations,omitempty"`
	PatientContext PatientContext  `json:"patient_context,omitempty"`
}

// CarePlanActivationResponse represents the response from activating a care plan
type CarePlanActivationResponse struct {
	Success       bool              `json:"success"`
	InstanceID    string            `json:"instance_id"`
	PlanID        string            `json:"plan_id"`
	PlanName      string            `json:"plan_name"`
	PatientID     string            `json:"patient_id"`
	Status        CarePlanStatus    `json:"status"`
	Goals         []Goal            `json:"goals"`
	Activities    []Activity        `json:"activities"`
	MonitoringItems []MonitoringItem `json:"monitoring_items,omitempty"`
	StartDate     time.Time         `json:"start_date"`
	EndDate       *time.Time        `json:"end_date,omitempty"`
	ErrorMessage  string            `json:"error_message,omitempty"`
}

// CPOESubmitRequest represents a request to submit orders through CPOE
type CPOESubmitRequest struct {
	SessionID    string   `json:"session_id,omitempty"` // Existing draft session
	PatientID    string   `json:"patient_id" binding:"required"`
	EncounterID  string   `json:"encounter_id" binding:"required"`
	ProviderID   string   `json:"provider_id" binding:"required"`
	Orders       []Order  `json:"orders" binding:"required"`
	Overrides    []SafetyOverride `json:"overrides,omitempty"`
}

// SafetyOverride represents an override for a safety alert
type SafetyOverride struct {
	AlertID     string `json:"alert_id"`
	Reason      string `json:"reason"`
	OverriddenBy string `json:"overridden_by"`
	Acknowledged bool   `json:"acknowledged"`
}

// CPOESubmitResponse represents the response from submitting orders
type CPOESubmitResponse struct {
	Success       bool          `json:"success"`
	SubmittedOrders []SubmittedOrderRef `json:"submitted_orders"`
	SafetyAlerts  []SafetyAlert `json:"safety_alerts,omitempty"`
	Blocked       bool          `json:"blocked"`
	BlockReason   string        `json:"block_reason,omitempty"`
	ErrorMessage  string        `json:"error_message,omitempty"`
}

// SubmittedOrderRef represents a reference to a submitted order
type SubmittedOrderRef struct {
	OrderID   string    `json:"order_id"`
	OrderType OrderType `json:"order_type"`
	Name      string    `json:"name"`
	Status    OrderStatus `json:"status"`
}

// BeforeCreate generates UUID and session ID before creating
func (c *CPOEDraftSession) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.SessionID == "" {
		c.SessionID = "CPOE-" + uuid.New().String()[:8]
	}
	if c.ExpiresAt.IsZero() {
		c.ExpiresAt = time.Now().Add(24 * time.Hour)
	}
	return nil
}

// BeforeCreate generates UUID and order ID before creating
func (s *SubmittedOrder) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.OrderID == "" {
		s.OrderID = "ORD-" + uuid.New().String()[:8]
	}
	return nil
}

// IsExpired returns true if the draft session has expired
func (c *CPOEDraftSession) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// IsDraft returns true if the session is still in draft status
func (c *CPOEDraftSession) IsDraft() bool {
	return c.Status == CPOEDraftStatusDraft && !c.IsExpired()
}
