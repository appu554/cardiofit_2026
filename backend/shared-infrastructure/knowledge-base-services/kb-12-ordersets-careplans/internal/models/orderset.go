// Package models defines data structures for KB-12 Order Sets & Care Plans
package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderSetCategory represents order set categories
type OrderSetCategory string

const (
	CategoryAdmission  OrderSetCategory = "admission"
	CategoryProcedure  OrderSetCategory = "procedure"
	CategoryEmergency  OrderSetCategory = "emergency"
	CategoryDischarge  OrderSetCategory = "discharge"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusDraft     OrderStatus = "draft"
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusActive    OrderStatus = "active"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusOnHold    OrderStatus = "on-hold"
)

// OrderType represents types of orders
type OrderType string

const (
	OrderTypeMedication  OrderType = "medication"
	OrderTypeLab         OrderType = "lab"
	OrderTypeImaging     OrderType = "imaging"
	OrderTypeProcedure   OrderType = "procedure"
	OrderTypeNursing     OrderType = "nursing"
	OrderTypeConsult     OrderType = "consult"
	OrderTypeDiet        OrderType = "diet"
	OrderTypeActivity    OrderType = "activity"
	OrderTypeMonitoring  OrderType = "monitoring"
	OrderTypeIVFluids    OrderType = "iv_fluids"
	OrderTypeVTE         OrderType = "vte_prophylaxis"
	OrderTypeEducation   OrderType = "education"
)

// Priority represents order priority levels
type Priority string

const (
	PrioritySTAT    Priority = "stat"
	PriorityUrgent  Priority = "urgent"
	PriorityRoutine Priority = "routine"
	PriorityPRN     Priority = "prn"
)

// OrderSetTemplate represents a template for an order set
// Uses Go native types with GORM JSONB serialization for clean API usage
type OrderSetTemplate struct {
	ID              string           `gorm:"type:varchar(50);primaryKey" json:"id"`
	TemplateID      string           `gorm:"uniqueIndex;size:50" json:"template_id,omitempty"`
	Category        OrderSetCategory `gorm:"size:30;not null;index" json:"category"`
	Subcategory     string           `gorm:"size:50" json:"subcategory,omitempty"`
	Specialty       string           `gorm:"size:50;index" json:"specialty,omitempty"`
	Name            string           `gorm:"size:200;not null" json:"name"`
	Version         string           `gorm:"size:20;not null" json:"version"`
	Status          string           `gorm:"size:20;default:'active'" json:"status,omitempty"`
	GuidelineSource string           `gorm:"size:200" json:"guideline_source,omitempty"`
	Description     string           `gorm:"type:text" json:"description,omitempty"`
	EvidenceLevel   string           `gorm:"size:50" json:"evidence_level,omitempty"`
	Author          string           `gorm:"size:100" json:"author,omitempty"`
	Approver        string           `gorm:"size:100" json:"approver,omitempty"`

	// JSON serialized fields - uses Go types for clean API, JSONB for storage
	References      StringSlice      `gorm:"type:jsonb" json:"references,omitempty"`
	ICDCodes        StringSlice      `gorm:"type:jsonb" json:"icd_codes,omitempty"`
	SNOMEDCodes     StringSlice      `gorm:"type:jsonb" json:"snomed_codes,omitempty"`
	Orders          OrderSlice       `gorm:"type:jsonb;not null" json:"orders"`
	Sections        OrderGroupSlice  `gorm:"type:jsonb" json:"sections,omitempty"`
	TimeConstraints ConstraintSlice  `gorm:"type:jsonb" json:"time_constraints,omitempty"`
	Metadata        JSONMap          `gorm:"type:jsonb" json:"metadata,omitempty"`

	// Temporal fields
	EffectiveDate   time.Time        `json:"effective_date,omitempty"`
	ExpirationDate  time.Time        `json:"expiration_date,omitempty"`
	Active          bool             `gorm:"default:true" json:"active"`
	CreatedAt       time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for OrderSetTemplate
func (OrderSetTemplate) TableName() string {
	return "order_set_templates"
}

// BeforeCreate generates ID if not set
func (o *OrderSetTemplate) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	return nil
}

// OrderSetInstance represents an activated order set for a patient
type OrderSetInstance struct {
	ID               uuid.UUID         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID       string            `gorm:"uniqueIndex;size:50;not null" json:"instance_id"`
	TemplateID       string            `gorm:"size:50;not null;index" json:"template_id"`
	PatientID        string            `gorm:"size:50;not null;index" json:"patient_id"`
	EncounterID      string            `gorm:"size:50;not null;index" json:"encounter_id"`
	ActivatedBy      string            `gorm:"size:100;not null" json:"activated_by"`
	Status           OrderStatus       `gorm:"size:20;not null;index" json:"status"`
	Orders           OrderSlice        `gorm:"type:jsonb;not null" json:"orders"`
	ConstraintStatus StatusSlice       `gorm:"type:jsonb" json:"constraint_status,omitempty"`
	ActivatedAt      time.Time         `gorm:"autoCreateTime" json:"activated_at"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	CreatedAt        time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for OrderSetInstance
func (OrderSetInstance) TableName() string {
	return "order_set_instances"
}

// BeforeCreate generates UUID and instance ID before creating
func (o *OrderSetInstance) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	if o.InstanceID == "" {
		o.InstanceID = "OSI-" + uuid.New().String()[:8]
	}
	return nil
}

// Order represents a single order within an order set
type Order struct {
	// Core identifiers
	ID              string            `json:"id,omitempty"`
	OrderID         string            `json:"order_id,omitempty"`
	Type            string            `json:"type,omitempty"`
	OrderType       OrderType         `json:"order_type,omitempty"`
	Category        string            `json:"category,omitempty"`
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	Instructions    string            `json:"instructions,omitempty"`
	Priority        Priority          `json:"priority,omitempty"`
	Status          string            `json:"status,omitempty"`
	Required        bool              `json:"required,omitempty"`
	IsRequired      bool              `json:"is_required,omitempty"`
	Selected        bool              `json:"selected,omitempty"`
	Sequence        int               `json:"sequence,omitempty"`

	// Medication-specific fields
	DrugCode        string            `json:"drug_code,omitempty"`
	DrugName        string            `json:"drug_name,omitempty"`
	RxNormCode      string            `json:"rxnorm_code,omitempty"`
	Dose            string            `json:"dose,omitempty"`
	DoseValue       float64           `json:"dose_value,omitempty"`
	DoseUnit        string            `json:"dose_unit,omitempty"`
	Route           string            `json:"route,omitempty"`
	Frequency       string            `json:"frequency,omitempty"`
	Duration        string            `json:"duration,omitempty"`
	PRN             bool              `json:"prn,omitempty"`
	PRNReason       string            `json:"prn_reason,omitempty"`
	MaxDailyDose    string            `json:"max_daily_dose,omitempty"`
	MaxDose         string            `json:"max_dose,omitempty"`
	WeightBased     bool              `json:"weight_based,omitempty"`

	// Lab-specific fields
	LabCode         string            `json:"lab_code,omitempty"`
	LOINCCode       string            `json:"loinc_code,omitempty"`
	LabPanel        string            `json:"lab_panel,omitempty"`
	Specimen        string            `json:"specimen,omitempty"`
	Timing          string            `json:"timing,omitempty"`

	// Imaging-specific fields
	ImagingCode     string            `json:"imaging_code,omitempty"`
	CPTCode         string            `json:"cpt_code,omitempty"`
	Modality        string            `json:"modality,omitempty"`
	BodySite        string            `json:"body_site,omitempty"`
	Contrast        bool              `json:"contrast,omitempty"`

	// Consult-specific fields
	Specialty       string            `json:"specialty,omitempty"`
	Reason          string            `json:"reason,omitempty"`
	Urgency         string            `json:"urgency,omitempty"`

	// Additional metadata
	Codes           []CodeReference   `json:"codes,omitempty"`
	Conditions      []OrderCondition  `json:"conditions,omitempty"`
	Alternatives    []string          `json:"alternatives,omitempty"`
	References      []string          `json:"references,omitempty"`
	Notes           string            `json:"notes,omitempty"`

	// Workflow fields
	DependsOn       []string          `json:"depends_on,omitempty"`
	TimeConstraint  *TimeConstraint   `json:"time_constraint,omitempty"`
}

// CodeReference represents a clinical code reference (SNOMED, ICD, LOINC, etc.)
type CodeReference struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display"`
}

// OrderCondition represents a condition for an order
type OrderCondition struct {
	ConditionID string `json:"condition_id,omitempty"`
	Type        string `json:"type,omitempty"`
	Field       string `json:"field,omitempty"`
	Parameter   string `json:"parameter,omitempty"`
	Operator    string `json:"operator,omitempty"`
	Value       string `json:"value,omitempty"`
	Unit        string `json:"unit,omitempty"`
	Action      string `json:"action,omitempty"`
}

// TimeConstraint represents a time-critical constraint for an order set
type TimeConstraint struct {
	// Core identifiers
	ID             string        `json:"id,omitempty"`
	ConstraintID   string        `json:"constraint_id,omitempty"`
	Name           string        `json:"name,omitempty"`
	Type           string        `json:"type,omitempty"`
	Action         string        `json:"action,omitempty"`
	Description    string        `json:"description,omitempty"`

	// Time specification
	Deadline       time.Duration `json:"deadline,omitempty"`
	AlertThreshold time.Duration `json:"alert_threshold,omitempty"`
	ReferenceEvent string        `json:"reference_event,omitempty"`
	OffsetMinutes  int           `json:"offset_minutes,omitempty"`
	DeadlineHours  float64       `json:"deadline_hours,omitempty"`

	// Priority and severity
	Priority       string        `json:"priority,omitempty"`
	Severity       string        `json:"severity,omitempty"`

	// Clinical rationale
	Reference      string        `json:"reference,omitempty"`
	Rationale      string        `json:"rationale,omitempty"`
	Regulatory     bool          `json:"regulatory,omitempty"`
	MetricsCode    string        `json:"metrics_code,omitempty"`
}

// ConstraintStatus represents the current status of a time constraint
type ConstraintStatus struct {
	ConstraintID    string        `json:"constraint_id"`
	Action          string        `json:"action"`
	Status          string        `json:"status"`
	StartTime       time.Time     `json:"start_time"`
	Deadline        time.Time     `json:"deadline"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`
	TimeRemaining   time.Duration `json:"time_remaining,omitempty"`
	TimeElapsed     time.Duration `json:"time_elapsed,omitempty"`
	PercentComplete float64       `json:"percent_complete"`
	Severity        string        `json:"severity"`
}

// OrderGroup represents a group of related orders (sections)
type OrderGroup struct {
	GroupID     string  `json:"group_id,omitempty"`
	ID          string  `json:"id,omitempty"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Sequence    int     `json:"sequence,omitempty"`
	Orders      []Order `json:"orders,omitempty"`
	Items       []Order `json:"items,omitempty"` // Alias for Orders in sections
}

// ==================== Custom GORM Types for JSONB ====================

// StringSlice is a []string that serializes to JSONB
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// OrderSlice is a []Order that serializes to JSONB
type OrderSlice []Order

func (s OrderSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *OrderSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// OrderGroupSlice is a []OrderGroup that serializes to JSONB
type OrderGroupSlice []OrderGroup

func (s OrderGroupSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *OrderGroupSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// ConstraintSlice is a []TimeConstraint that serializes to JSONB
type ConstraintSlice []TimeConstraint

func (s ConstraintSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *ConstraintSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// StatusSlice is a []ConstraintStatus that serializes to JSONB
type StatusSlice []ConstraintStatus

func (s StatusSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *StatusSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// JSONMap is a map[string]interface{} that serializes to JSONB
type JSONMap map[string]interface{}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, m)
}

// SafetyAlertSlice is a []SafetyAlert that serializes to JSONB
// SafetyAlert is defined in activation.go
type SafetyAlertSlice []SafetyAlert

func (s SafetyAlertSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *SafetyAlertSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// IsTimeCritical returns true if the order set has time constraints
func (o *OrderSetTemplate) IsTimeCritical() bool {
	return len(o.TimeConstraints) > 0
}

// HasCriticalConstraints returns true if any constraint is critical severity
func (o *OrderSetTemplate) HasCriticalConstraints() bool {
	for _, c := range o.TimeConstraints {
		if c.Severity == "critical" || c.Priority == "high" {
			return true
		}
	}
	return false
}

// GetOrders returns the orders directly (compatibility method)
func (o *OrderSetTemplate) GetOrders() ([]Order, error) {
	return o.Orders, nil
}

// SetOrders sets the orders directly (compatibility method)
func (o *OrderSetTemplate) SetOrders(orders []Order) error {
	o.Orders = orders
	return nil
}

// GetTimeConstraints returns the time constraints directly (compatibility method)
func (o *OrderSetTemplate) GetTimeConstraints() ([]TimeConstraint, error) {
	return o.TimeConstraints, nil
}

// SetTimeConstraints sets the time constraints directly (compatibility method)
func (o *OrderSetTemplate) SetTimeConstraints(constraints []TimeConstraint) error {
	o.TimeConstraints = constraints
	return nil
}

// GetOrders returns the instance orders directly (compatibility method)
func (o *OrderSetInstance) GetOrders() ([]Order, error) {
	return o.Orders, nil
}

// SetOrders sets the instance orders directly (compatibility method)
func (o *OrderSetInstance) SetOrders(orders []Order) error {
	o.Orders = orders
	return nil
}

// GetConstraintStatus returns the constraint status directly (compatibility method)
func (o *OrderSetInstance) GetConstraintStatus() ([]ConstraintStatus, error) {
	return o.ConstraintStatus, nil
}

// SetConstraintStatus sets the constraint status directly (compatibility method)
func (o *OrderSetInstance) SetConstraintStatus(status []ConstraintStatus) error {
	o.ConstraintStatus = status
	return nil
}
