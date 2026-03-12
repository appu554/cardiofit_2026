// Package clients provides HTTP clients for KB services.
//
// KB12HTTPClient implements the KB12Client interface for KB-12 OrderSets & CarePlans Service.
// It provides access to order set templates, care plan templates, and CPOE integration.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// - KB-12 provides ORDER EXECUTION through templates and care plans
// - Templates are pre-built clinical order sets (admission, procedure, emergency)
// - Care Plans are long-term management plans for chronic conditions
// - Integrates with KB-1 (dosing), KB-3 (temporal), KB-6 (formulary), KB-7 (terminology)
// - CPOE integration provides safety checks before order submission
//
// Key Endpoints:
//   - /api/v1/templates - Order Set Template management
//   - /api/v1/activate - Activate order sets for patients
//   - /api/v1/instances - Active order set instances
//   - /api/v1/careplans - Care Plan Template management
//   - /api/v1/careplan-instances - Active care plan instances
//   - /api/v1/cpoe - CPOE integration and safety checks
//   - /api/v1/workflow - Task and deadline management
//   - /api/v1/fhir - FHIR resource generation
//   - /cds-services - CDS Hooks integration
//
// Connects to: http://localhost:8092 (Docker: kb12-ordersets-careplans)
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// ORDER SET TYPES
// ============================================================================

// OrderSetCategory represents order set categories.
type OrderSetCategory string

const (
	CategoryAdmission OrderSetCategory = "admission"
	CategoryProcedure OrderSetCategory = "procedure"
	CategoryEmergency OrderSetCategory = "emergency"
	CategoryDischarge OrderSetCategory = "discharge"
)

// OrderStatus represents the status of an order.
type OrderStatus string

const (
	OrderStatusDraft     OrderStatus = "draft"
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusActive    OrderStatus = "active"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusOnHold    OrderStatus = "on-hold"
)

// OrderType represents types of orders.
type OrderType string

const (
	OrderTypeMedication OrderType = "medication"
	OrderTypeLab        OrderType = "lab"
	OrderTypeImaging    OrderType = "imaging"
	OrderTypeProcedure  OrderType = "procedure"
	OrderTypeNursing    OrderType = "nursing"
	OrderTypeConsult    OrderType = "consult"
	OrderTypeDiet       OrderType = "diet"
	OrderTypeActivity   OrderType = "activity"
	OrderTypeMonitoring OrderType = "monitoring"
	OrderTypeIVFluids   OrderType = "iv_fluids"
	OrderTypeVTE        OrderType = "vte_prophylaxis"
	OrderTypeEducation  OrderType = "education"
)

// Priority represents order priority levels.
type Priority string

const (
	PrioritySTAT    Priority = "stat"
	PriorityUrgent  Priority = "urgent"
	PriorityRoutine Priority = "routine"
	PriorityPRN     Priority = "prn"
)

// OrderSetTemplate represents a template for an order set.
type OrderSetTemplate struct {
	// ID is the unique identifier
	ID string `json:"id"`

	// TemplateID is an alternative identifier
	TemplateID string `json:"template_id,omitempty"`

	// Category of the order set
	Category OrderSetCategory `json:"category"`

	// Subcategory within the category
	Subcategory string `json:"subcategory,omitempty"`

	// Specialty the order set belongs to
	Specialty string `json:"specialty,omitempty"`

	// Name is the display name
	Name string `json:"name"`

	// Version of the order set
	Version string `json:"version"`

	// Status (active, draft, retired)
	Status string `json:"status,omitempty"`

	// GuidelineSource the order set is based on
	GuidelineSource string `json:"guideline_source,omitempty"`

	// Description of the order set
	Description string `json:"description,omitempty"`

	// EvidenceLevel for the order set
	EvidenceLevel string `json:"evidence_level,omitempty"`

	// Author who created the template
	Author string `json:"author,omitempty"`

	// Approver who approved the template
	Approver string `json:"approver,omitempty"`

	// References to supporting literature
	References []string `json:"references,omitempty"`

	// ICDCodes for billing/diagnosis
	ICDCodes []string `json:"icd_codes,omitempty"`

	// SNOMEDCodes for clinical coding
	SNOMEDCodes []string `json:"snomed_codes,omitempty"`

	// Orders in this order set
	Orders []Order `json:"orders"`

	// Sections organizing the orders
	Sections []OrderSection `json:"sections,omitempty"`

	// TimeConstraints for the order set
	TimeConstraints []TimeConstraint `json:"time_constraints,omitempty"`

	// Metadata additional information
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// EffectiveDate when the template becomes active
	EffectiveDate *time.Time `json:"effective_date,omitempty"`

	// ExpirationDate when the template expires
	ExpirationDate *time.Time `json:"expiration_date,omitempty"`

	// Active flag
	Active bool `json:"active"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt timestamp
	UpdatedAt time.Time `json:"updated_at"`
}

// Order represents a single order within an order set.
type Order struct {
	// ID of the order
	ID string `json:"id,omitempty"`

	// OrderID alternative identifier
	OrderID string `json:"order_id,omitempty"`

	// Type of the order
	Type string `json:"type,omitempty"`

	// OrderType enum
	OrderType OrderType `json:"order_type,omitempty"`

	// Category of the order
	Category string `json:"category,omitempty"`

	// Name is the display name
	Name string `json:"name"`

	// Description of the order
	Description string `json:"description,omitempty"`

	// RxNormCode for medications
	RxNormCode string `json:"rxnorm_code,omitempty"`

	// LoincCode for labs
	LoincCode string `json:"loinc_code,omitempty"`

	// SNOMEDCode for procedures
	SNOMEDCode string `json:"snomed_code,omitempty"`

	// Dose for medications
	Dose float64 `json:"dose,omitempty"`

	// DoseUnit for medications
	DoseUnit string `json:"dose_unit,omitempty"`

	// Route for medications
	Route string `json:"route,omitempty"`

	// Frequency of the order
	Frequency string `json:"frequency,omitempty"`

	// Duration of the order
	Duration string `json:"duration,omitempty"`

	// Priority of the order
	Priority Priority `json:"priority,omitempty"`

	// Status of the order
	Status OrderStatus `json:"status,omitempty"`

	// IsRequired if this order is mandatory
	IsRequired bool `json:"is_required,omitempty"`

	// DefaultSelected if pre-selected
	DefaultSelected bool `json:"default_selected,omitempty"`

	// Instructions for the order
	Instructions string `json:"instructions,omitempty"`

	// Indication for the order
	Indication string `json:"indication,omitempty"`

	// Contraindications to this order
	Contraindications []string `json:"contraindications,omitempty"`

	// TimeConstraint for this order
	TimeConstraint *TimeConstraint `json:"time_constraint,omitempty"`
}

// OrderSection groups related orders.
type OrderSection struct {
	// SectionID identifier
	SectionID string `json:"section_id"`

	// Name of the section
	Name string `json:"name"`

	// Description of the section
	Description string `json:"description,omitempty"`

	// OrderIDs in this section
	OrderIDs []string `json:"order_ids"`

	// Order of the section
	Order int `json:"order,omitempty"`
}

// TimeConstraint defines temporal constraints on orders.
type TimeConstraint struct {
	// ConstraintID identifier
	ConstraintID string `json:"constraint_id"`

	// OrderID the constraint applies to
	OrderID string `json:"order_id,omitempty"`

	// Type of constraint (deadline, interval, dependency)
	Type string `json:"type"`

	// WindowMinutes allowed window
	WindowMinutes int `json:"window_minutes,omitempty"`

	// DeadlineMinutes from activation
	DeadlineMinutes int `json:"deadline_minutes,omitempty"`

	// IsCritical if this is a critical constraint
	IsCritical bool `json:"is_critical,omitempty"`

	// Description of the constraint
	Description string `json:"description,omitempty"`
}

// OrderSetInstance represents an activated order set for a patient.
type OrderSetInstance struct {
	// ID is the UUID
	ID string `json:"id"`

	// InstanceID is a human-readable ID
	InstanceID string `json:"instance_id"`

	// TemplateID of the source template
	TemplateID string `json:"template_id"`

	// PatientID the instance is for
	PatientID string `json:"patient_id"`

	// EncounterID the instance is associated with
	EncounterID string `json:"encounter_id"`

	// ActivatedBy user who activated
	ActivatedBy string `json:"activated_by"`

	// Status of the instance
	Status OrderStatus `json:"status"`

	// Orders in this instance
	Orders []Order `json:"orders"`

	// ConstraintStatus for each constraint
	ConstraintStatus []ConstraintStatusItem `json:"constraint_status,omitempty"`

	// ActivatedAt timestamp
	ActivatedAt time.Time `json:"activated_at"`

	// CompletedAt timestamp if completed
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt timestamp
	UpdatedAt time.Time `json:"updated_at"`
}

// ConstraintStatusItem tracks status of a time constraint.
type ConstraintStatusItem struct {
	// ConstraintID being tracked
	ConstraintID string `json:"constraint_id"`

	// Status (pending, met, overdue, waived)
	Status string `json:"status"`

	// DueAt deadline time
	DueAt *time.Time `json:"due_at,omitempty"`

	// MetAt when constraint was met
	MetAt *time.Time `json:"met_at,omitempty"`

	// WaivedBy if constraint was waived
	WaivedBy string `json:"waived_by,omitempty"`

	// WaivedReason reason for waiving
	WaivedReason string `json:"waived_reason,omitempty"`
}

// ============================================================================
// CARE PLAN TYPES
// ============================================================================

// CarePlanStatus represents the status of a care plan.
type CarePlanStatus string

const (
	CarePlanStatusDraft     CarePlanStatus = "draft"
	CarePlanStatusActive    CarePlanStatus = "active"
	CarePlanStatusOnHold    CarePlanStatus = "on-hold"
	CarePlanStatusCompleted CarePlanStatus = "completed"
	CarePlanStatusCancelled CarePlanStatus = "cancelled"
	CarePlanStatusRevoked   CarePlanStatus = "revoked"
)

// GoalStatus represents the status of a goal.
type GoalStatus string

const (
	GoalStatusProposed    GoalStatus = "proposed"
	GoalStatusAccepted    GoalStatus = "accepted"
	GoalStatusInProgress  GoalStatus = "in-progress"
	GoalStatusAchieved    GoalStatus = "achieved"
	GoalStatusNotAchieved GoalStatus = "not-achieved"
	GoalStatusCancelled   GoalStatus = "cancelled"
)

// ActivityStatus represents the status of an activity.
type ActivityStatus string

const (
	ActivityStatusScheduled  ActivityStatus = "scheduled"
	ActivityStatusInProgress ActivityStatus = "in-progress"
	ActivityStatusCompleted  ActivityStatus = "completed"
	ActivityStatusNotDone    ActivityStatus = "not-done"
	ActivityStatusCancelled  ActivityStatus = "cancelled"
)

// CarePlanTemplate represents a template for a chronic care plan.
type CarePlanTemplate struct {
	// ID is the unique identifier
	ID string `json:"id"`

	// PlanID alternative identifier
	PlanID string `json:"plan_id,omitempty"`

	// TemplateID alias
	TemplateID string `json:"template_id,omitempty"`

	// Condition the plan addresses
	Condition string `json:"condition"`

	// ConditionRef structured condition reference
	ConditionRef *contracts.ClinicalCode `json:"condition_ref,omitempty"`

	// Category of the care plan
	Category string `json:"category,omitempty"`

	// Subcategory within the category
	Subcategory string `json:"subcategory,omitempty"`

	// Name is the display name
	Name string `json:"name"`

	// Description of the care plan
	Description string `json:"description,omitempty"`

	// GuidelineSource the plan is based on
	GuidelineSource string `json:"guideline_source,omitempty"`

	// Guidelines referenced
	Guidelines []GuidelineRef `json:"guidelines,omitempty"`

	// Version of the plan
	Version string `json:"version,omitempty"`

	// Status (active, draft, retired)
	Status string `json:"status,omitempty"`

	// Duration of the plan (e.g., "ongoing", "6 months")
	Duration string `json:"duration,omitempty"`

	// ReviewPeriod when to review (e.g., "3-6 months")
	ReviewPeriod string `json:"review_period,omitempty"`

	// Goals in this care plan
	Goals []CarePlanGoal `json:"goals"`

	// Activities in this care plan
	Activities []CarePlanActivity `json:"activities"`

	// MonitoringItems for ongoing monitoring
	MonitoringItems []CarePlanMonitoringItem `json:"monitoring_items,omitempty"`

	// Active flag
	Active bool `json:"active"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt timestamp
	UpdatedAt time.Time `json:"updated_at"`
}

// GuidelineRef references a clinical guideline.
type GuidelineRef struct {
	// GuidelineID identifier
	GuidelineID string `json:"guideline_id,omitempty"`

	// Name of the guideline
	Name string `json:"name"`

	// Source organization
	Source string `json:"source,omitempty"`

	// URL to the guideline
	URL string `json:"url,omitempty"`

	// Year of publication
	Year int `json:"year,omitempty"`
}

// CarePlanGoal represents a goal in a care plan.
type CarePlanGoal struct {
	// GoalID identifier
	GoalID string `json:"goal_id,omitempty"`

	// ID alternative identifier
	ID string `json:"id,omitempty"`

	// Description of the goal
	Description string `json:"description"`

	// Target value if measurable
	Target *GoalTarget `json:"target,omitempty"`

	// Category of the goal
	Category string `json:"category,omitempty"`

	// Priority of the goal
	Priority string `json:"priority,omitempty"`

	// Timeframe for achieving
	Timeframe string `json:"timeframe,omitempty"`

	// Status of the goal
	Status GoalStatus `json:"status,omitempty"`
}

// GoalTarget defines a measurable target for a goal.
type GoalTarget struct {
	// Measure being tracked (e.g., HbA1c, BP)
	Measure string `json:"measure"`

	// Operator (lt, lte, gt, gte, eq, between)
	Operator string `json:"operator"`

	// Value target value
	Value float64 `json:"value"`

	// UpperValue for range targets
	UpperValue float64 `json:"upper_value,omitempty"`

	// Unit of measurement
	Unit string `json:"unit,omitempty"`
}

// CarePlanActivity represents an activity in a care plan.
type CarePlanActivity struct {
	// ActivityID identifier
	ActivityID string `json:"activity_id,omitempty"`

	// ID alternative identifier
	ID string `json:"id,omitempty"`

	// Description of the activity
	Description string `json:"description"`

	// Category of the activity
	Category string `json:"category,omitempty"`

	// Type of activity (medication, lab, visit, education)
	Type string `json:"type,omitempty"`

	// Recurrence schedule
	Recurrence *ActivityRecurrence `json:"recurrence,omitempty"`

	// Status of the activity
	Status ActivityStatus `json:"status,omitempty"`

	// ReferenceCode (LOINC, SNOMED, RxNorm)
	ReferenceCode string `json:"reference_code,omitempty"`
}

// ActivityRecurrence defines the schedule for an activity.
type ActivityRecurrence struct {
	// Frequency (daily, weekly, monthly, quarterly, annually)
	Frequency string `json:"frequency"`

	// Interval (e.g., 1 for every month, 3 for every 3 months)
	Interval int `json:"interval"`

	// DaysFromBaseline for one-time follow-ups
	DaysFromBaseline int `json:"days_from_baseline,omitempty"`
}

// CarePlanMonitoringItem represents a monitoring item in a care plan.
type CarePlanMonitoringItem struct {
	// ItemID identifier
	ItemID string `json:"item_id"`

	// Name of the monitoring item
	Name string `json:"name"`

	// Type of monitoring (lab, vital, screening)
	Type string `json:"type"`

	// Code (LOINC, SNOMED)
	Code string `json:"code,omitempty"`

	// Recurrence schedule
	Recurrence *ActivityRecurrence `json:"recurrence"`

	// Priority of this item
	Priority string `json:"priority,omitempty"`

	// Rationale for monitoring
	Rationale string `json:"rationale,omitempty"`
}

// CarePlanInstance represents an activated care plan for a patient.
type CarePlanInstance struct {
	// ID is the UUID
	ID string `json:"id"`

	// InstanceID is a human-readable ID
	InstanceID string `json:"instance_id"`

	// TemplateID of the source template
	TemplateID string `json:"template_id"`

	// PatientID the instance is for
	PatientID string `json:"patient_id"`

	// Status of the instance
	Status CarePlanStatus `json:"status"`

	// StartDate when the plan started
	StartDate time.Time `json:"start_date"`

	// EndDate when the plan ended (if applicable)
	EndDate *time.Time `json:"end_date,omitempty"`

	// GoalsProgress tracking goal progress
	GoalsProgress []GoalProgress `json:"goals_progress,omitempty"`

	// ActivitiesCompleted tracking activity completion
	ActivitiesCompleted []ActivityCompletion `json:"activities_completed,omitempty"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt timestamp
	UpdatedAt time.Time `json:"updated_at"`
}

// GoalProgress tracks progress toward a goal.
type GoalProgress struct {
	// GoalID being tracked
	GoalID string `json:"goal_id"`

	// Status of the goal
	Status GoalStatus `json:"status"`

	// CurrentValue current measured value
	CurrentValue float64 `json:"current_value,omitempty"`

	// TargetValue target value
	TargetValue float64 `json:"target_value,omitempty"`

	// Unit of measurement
	Unit string `json:"unit,omitempty"`

	// LastUpdated when progress was last updated
	LastUpdated *time.Time `json:"last_updated,omitempty"`

	// Notes about progress
	Notes string `json:"notes,omitempty"`
}

// ActivityCompletion tracks completion of an activity.
type ActivityCompletion struct {
	// ActivityID being tracked
	ActivityID string `json:"activity_id"`

	// Status of the activity
	Status ActivityStatus `json:"status"`

	// CompletedAt when completed
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// CompletedBy who completed it
	CompletedBy string `json:"completed_by,omitempty"`

	// Notes about completion
	Notes string `json:"notes,omitempty"`
}

// ============================================================================
// CPOE TYPES
// ============================================================================

// CPOEDraftSession represents a draft CPOE session.
type CPOEDraftSession struct {
	// ID of the draft session
	ID string `json:"id"`

	// PatientID the session is for
	PatientID string `json:"patient_id"`

	// EncounterID the session is associated with
	EncounterID string `json:"encounter_id"`

	// CreatedBy user who created the session
	CreatedBy string `json:"created_by"`

	// Orders in the draft
	Orders []Order `json:"orders"`

	// SafetyCheckResults from safety validation
	SafetyCheckResults *SafetyCheckResult `json:"safety_check_results,omitempty"`

	// Status (draft, submitted, cancelled)
	Status string `json:"status"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt timestamp
	UpdatedAt time.Time `json:"updated_at"`
}

// SafetyCheckResult contains results of CPOE safety check.
type SafetyCheckResult struct {
	// Passed if all safety checks passed
	Passed bool `json:"passed"`

	// Alerts generated by safety checks
	Alerts []SafetyAlert `json:"alerts,omitempty"`

	// InteractionWarnings drug-drug interactions
	InteractionWarnings []InteractionWarning `json:"interaction_warnings,omitempty"`

	// AllergyWarnings allergy matches
	AllergyWarnings []AllergyWarning `json:"allergy_warnings,omitempty"`

	// DoseWarnings dosing issues
	DoseWarnings []DoseWarning `json:"dose_warnings,omitempty"`

	// CheckedAt when safety check was performed
	CheckedAt time.Time `json:"checked_at"`
}

// SafetyAlert represents a general safety alert.
type SafetyAlert struct {
	// AlertID identifier
	AlertID string `json:"alert_id"`

	// Type of alert
	Type string `json:"type"`

	// Severity (critical, high, moderate, low)
	Severity string `json:"severity"`

	// Message describing the alert
	Message string `json:"message"`

	// OrderID the alert relates to
	OrderID string `json:"order_id,omitempty"`

	// Recommendation to address
	Recommendation string `json:"recommendation,omitempty"`

	// RequiresOverride if must be overridden
	RequiresOverride bool `json:"requires_override"`
}

// InteractionWarning represents a drug-drug interaction warning.
type InteractionWarning struct {
	// Drug1 first drug
	Drug1 string `json:"drug1"`

	// Drug2 second drug
	Drug2 string `json:"drug2"`

	// Severity of interaction
	Severity string `json:"severity"`

	// Description of interaction
	Description string `json:"description"`

	// Mechanism of interaction
	Mechanism string `json:"mechanism,omitempty"`

	// Management recommendation
	Management string `json:"management,omitempty"`
}

// AllergyWarning represents an allergy match warning.
type AllergyWarning struct {
	// DrugName triggering the warning
	DrugName string `json:"drug_name"`

	// Allergen matched
	Allergen string `json:"allergen"`

	// MatchType (exact, class, cross-reactive)
	MatchType string `json:"match_type"`

	// Severity of reaction
	Severity string `json:"severity"`

	// Reaction expected
	Reaction string `json:"reaction,omitempty"`
}

// DoseWarning represents a dosing issue warning.
type DoseWarning struct {
	// DrugName with the issue
	DrugName string `json:"drug_name"`

	// Issue description
	Issue string `json:"issue"`

	// RecommendedDose if applicable
	RecommendedDose string `json:"recommended_dose,omitempty"`

	// Reason for the warning
	Reason string `json:"reason,omitempty"`
}

// ============================================================================
// WORKFLOW TYPES
// ============================================================================

// KB12WorkflowTask represents a task in the order set/care plan workflow.
// Named KB12WorkflowTask to distinguish from KB-14's WorkflowTask (care navigation).
type KB12WorkflowTask struct {
	// TaskID identifier
	TaskID string `json:"task_id"`

	// Type of task
	Type string `json:"type"`

	// Description of the task
	Description string `json:"description"`

	// PatientID the task relates to
	PatientID string `json:"patient_id"`

	// InstanceID (order set or care plan)
	InstanceID string `json:"instance_id,omitempty"`

	// OrderID if related to an order
	OrderID string `json:"order_id,omitempty"`

	// Status of the task
	Status string `json:"status"`

	// Priority of the task
	Priority Priority `json:"priority"`

	// AssignedTo user assigned
	AssignedTo string `json:"assigned_to,omitempty"`

	// DueAt deadline
	DueAt *time.Time `json:"due_at,omitempty"`

	// IsCritical if time-sensitive
	IsCritical bool `json:"is_critical"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt timestamp
	UpdatedAt time.Time `json:"updated_at"`
}

// ============================================================================
// KB-12 HTTP CLIENT
// ============================================================================

// KB12HTTPClient implements KB12Client by calling the KB-12 OrderSets & CarePlans Service REST API.
type KB12HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB12HTTPClient creates a new KB-12 HTTP client.
func NewKB12HTTPClient(baseURL string) *KB12HTTPClient {
	return &KB12HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB12HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB12HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB12HTTPClient {
	return &KB12HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// ORDER SET TEMPLATE METHODS
// ============================================================================

// ListTemplates returns all order set templates.
// Calls KB-12 GET /api/v1/templates endpoint.
func (c *KB12HTTPClient) ListTemplates(ctx context.Context) ([]OrderSetTemplate, error) {
	resp, err := c.doGet(ctx, "/api/v1/templates")
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	var result kb12TemplatesResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return result.Templates, nil
}

// GetTemplate returns a specific order set template.
// Calls KB-12 GET /api/v1/templates/{id} endpoint.
func (c *KB12HTTPClient) GetTemplate(ctx context.Context, templateID string) (*OrderSetTemplate, error) {
	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/templates/%s", url.PathEscape(templateID)))
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	var template OrderSetTemplate
	if err := json.Unmarshal(resp, &template); err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &template, nil
}

// SearchTemplates searches for order set templates.
// Calls KB-12 GET /api/v1/templates/search endpoint.
func (c *KB12HTTPClient) SearchTemplates(
	ctx context.Context,
	query string,
	category OrderSetCategory,
	specialty string,
) ([]OrderSetTemplate, error) {

	reqURL := fmt.Sprintf("/api/v1/templates/search?q=%s", url.QueryEscape(query))
	if category != "" {
		reqURL += "&category=" + url.QueryEscape(string(category))
	}
	if specialty != "" {
		reqURL += "&specialty=" + url.QueryEscape(specialty)
	}

	resp, err := c.doGet(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to search templates: %w", err)
	}

	var result kb12TemplatesResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	return result.Templates, nil
}

// GetTemplatesByCategory returns templates for a category.
// Calls KB-12 GET /api/v1/templates/category/{category} endpoint.
func (c *KB12HTTPClient) GetTemplatesByCategory(
	ctx context.Context,
	category OrderSetCategory,
) ([]OrderSetTemplate, error) {

	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/templates/category/%s", url.PathEscape(string(category))))
	if err != nil {
		return nil, fmt.Errorf("failed to get templates by category: %w", err)
	}

	var result kb12TemplatesResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return result.Templates, nil
}

// ============================================================================
// ORDER SET ACTIVATION METHODS
// ============================================================================

// ActivateOrderSet activates an order set for a patient.
// Calls KB-12 POST /api/v1/activate endpoint.
func (c *KB12HTTPClient) ActivateOrderSet(
	ctx context.Context,
	templateID string,
	patientID string,
	encounterID string,
	activatedBy string,
	customizations map[string]interface{},
) (*OrderSetInstance, error) {

	req := kb12ActivateRequest{
		TemplateID:     templateID,
		PatientID:      patientID,
		EncounterID:    encounterID,
		ActivatedBy:    activatedBy,
		Customizations: customizations,
	}

	resp, err := c.callKB12(ctx, "/api/v1/activate", req)
	if err != nil {
		return nil, fmt.Errorf("failed to activate order set: %w", err)
	}

	var instance OrderSetInstance
	if err := json.Unmarshal(resp, &instance); err != nil {
		return nil, fmt.Errorf("failed to parse activation result: %w", err)
	}

	return &instance, nil
}

// ============================================================================
// ORDER SET INSTANCE METHODS
// ============================================================================

// ListInstances returns active order set instances.
// Calls KB-12 GET /api/v1/instances endpoint.
func (c *KB12HTTPClient) ListInstances(ctx context.Context, patientID string) ([]OrderSetInstance, error) {
	reqURL := "/api/v1/instances"
	if patientID != "" {
		reqURL += "?patient_id=" + url.QueryEscape(patientID)
	}

	resp, err := c.doGet(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	var result kb12InstancesResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse instances: %w", err)
	}

	return result.Instances, nil
}

// GetInstance returns a specific order set instance.
// Calls KB-12 GET /api/v1/instances/{id} endpoint.
func (c *KB12HTTPClient) GetInstance(ctx context.Context, instanceID string) (*OrderSetInstance, error) {
	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/instances/%s", url.PathEscape(instanceID)))
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	var instance OrderSetInstance
	if err := json.Unmarshal(resp, &instance); err != nil {
		return nil, fmt.Errorf("failed to parse instance: %w", err)
	}

	return &instance, nil
}

// UpdateInstanceStatus updates the status of an order set instance.
// Calls KB-12 PUT /api/v1/instances/{id}/status endpoint.
func (c *KB12HTTPClient) UpdateInstanceStatus(
	ctx context.Context,
	instanceID string,
	status OrderStatus,
	updatedBy string,
) error {

	req := kb12StatusUpdateRequest{
		Status:    string(status),
		UpdatedBy: updatedBy,
	}

	_, err := c.callKB12(ctx, fmt.Sprintf("/api/v1/instances/%s/status", instanceID), req)
	if err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	return nil
}

// UpdateOrderStatus updates the status of an order within an instance.
// Calls KB-12 PUT /api/v1/instances/{id}/order/{order_id} endpoint.
func (c *KB12HTTPClient) UpdateOrderStatus(
	ctx context.Context,
	instanceID string,
	orderID string,
	status OrderStatus,
	updatedBy string,
) error {

	req := kb12StatusUpdateRequest{
		Status:    string(status),
		UpdatedBy: updatedBy,
	}

	_, err := c.callKB12(ctx, fmt.Sprintf("/api/v1/instances/%s/order/%s", instanceID, orderID), req)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

// ============================================================================
// CARE PLAN METHODS
// ============================================================================

// ListCarePlanTemplates returns all care plan templates.
// Calls KB-12 GET /api/v1/careplans endpoint.
func (c *KB12HTTPClient) ListCarePlanTemplates(ctx context.Context) ([]CarePlanTemplate, error) {
	resp, err := c.doGet(ctx, "/api/v1/careplans")
	if err != nil {
		return nil, fmt.Errorf("failed to list care plan templates: %w", err)
	}

	var result kb12CarePlansResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse care plan templates: %w", err)
	}

	return result.CarePlans, nil
}

// GetCarePlanTemplate returns a specific care plan template.
// Calls KB-12 GET /api/v1/careplans/{id} endpoint.
func (c *KB12HTTPClient) GetCarePlanTemplate(ctx context.Context, templateID string) (*CarePlanTemplate, error) {
	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/careplans/%s", url.PathEscape(templateID)))
	if err != nil {
		return nil, fmt.Errorf("failed to get care plan template: %w", err)
	}

	var template CarePlanTemplate
	if err := json.Unmarshal(resp, &template); err != nil {
		return nil, fmt.Errorf("failed to parse care plan template: %w", err)
	}

	return &template, nil
}

// ActivateCarePlan activates a care plan for a patient.
// Calls KB-12 POST /api/v1/careplans endpoint.
func (c *KB12HTTPClient) ActivateCarePlan(
	ctx context.Context,
	templateID string,
	patientID string,
	activatedBy string,
	customizations map[string]interface{},
) (*CarePlanInstance, error) {

	req := kb12ActivateCarePlanRequest{
		TemplateID:     templateID,
		PatientID:      patientID,
		ActivatedBy:    activatedBy,
		Customizations: customizations,
	}

	resp, err := c.callKB12(ctx, "/api/v1/careplans", req)
	if err != nil {
		return nil, fmt.Errorf("failed to activate care plan: %w", err)
	}

	var instance CarePlanInstance
	if err := json.Unmarshal(resp, &instance); err != nil {
		return nil, fmt.Errorf("failed to parse care plan activation result: %w", err)
	}

	return &instance, nil
}

// ============================================================================
// CARE PLAN INSTANCE METHODS
// ============================================================================

// ListCarePlanInstances returns active care plan instances.
// Calls KB-12 GET /api/v1/careplan-instances endpoint.
func (c *KB12HTTPClient) ListCarePlanInstances(ctx context.Context, patientID string) ([]CarePlanInstance, error) {
	reqURL := "/api/v1/careplan-instances"
	if patientID != "" {
		reqURL += "?patient_id=" + url.QueryEscape(patientID)
	}

	resp, err := c.doGet(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list care plan instances: %w", err)
	}

	var result kb12CarePlanInstancesResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse care plan instances: %w", err)
	}

	return result.Instances, nil
}

// GetCarePlanInstance returns a specific care plan instance.
// Calls KB-12 GET /api/v1/careplan-instances/{id} endpoint.
func (c *KB12HTTPClient) GetCarePlanInstance(ctx context.Context, instanceID string) (*CarePlanInstance, error) {
	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/careplan-instances/%s", url.PathEscape(instanceID)))
	if err != nil {
		return nil, fmt.Errorf("failed to get care plan instance: %w", err)
	}

	var instance CarePlanInstance
	if err := json.Unmarshal(resp, &instance); err != nil {
		return nil, fmt.Errorf("failed to parse care plan instance: %w", err)
	}

	return &instance, nil
}

// UpdateCarePlanStatus updates the status of a care plan instance.
// Calls KB-12 PUT /api/v1/careplan-instances/{id}/status endpoint.
func (c *KB12HTTPClient) UpdateCarePlanStatus(
	ctx context.Context,
	instanceID string,
	status CarePlanStatus,
	updatedBy string,
) error {

	req := kb12StatusUpdateRequest{
		Status:    string(status),
		UpdatedBy: updatedBy,
	}

	_, err := c.callKB12(ctx, fmt.Sprintf("/api/v1/careplan-instances/%s/status", instanceID), req)
	if err != nil {
		return fmt.Errorf("failed to update care plan status: %w", err)
	}

	return nil
}

// UpdateGoalProgress updates goal progress in a care plan.
// Calls KB-12 PUT /api/v1/careplan-instances/{id}/goals/{goal_id} endpoint.
func (c *KB12HTTPClient) UpdateGoalProgress(
	ctx context.Context,
	instanceID string,
	goalID string,
	progress GoalProgress,
) error {

	_, err := c.callKB12(ctx, fmt.Sprintf("/api/v1/careplan-instances/%s/goals/%s", instanceID, goalID), progress)
	if err != nil {
		return fmt.Errorf("failed to update goal progress: %w", err)
	}

	return nil
}

// UpdateActivityStatus updates activity status in a care plan.
// Calls KB-12 PUT /api/v1/careplan-instances/{id}/activities/{activity_id} endpoint.
func (c *KB12HTTPClient) UpdateActivityStatus(
	ctx context.Context,
	instanceID string,
	activityID string,
	completion ActivityCompletion,
) error {

	_, err := c.callKB12(ctx, fmt.Sprintf("/api/v1/careplan-instances/%s/activities/%s", instanceID, activityID), completion)
	if err != nil {
		return fmt.Errorf("failed to update activity status: %w", err)
	}

	return nil
}

// ============================================================================
// CPOE METHODS
// ============================================================================

// CreateDraftSession creates a new CPOE draft session.
// Calls KB-12 POST /api/v1/cpoe/drafts endpoint.
func (c *KB12HTTPClient) CreateDraftSession(
	ctx context.Context,
	patientID string,
	encounterID string,
	createdBy string,
) (*CPOEDraftSession, error) {

	req := kb12CreateDraftRequest{
		PatientID:   patientID,
		EncounterID: encounterID,
		CreatedBy:   createdBy,
	}

	resp, err := c.callKB12(ctx, "/api/v1/cpoe/drafts", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create draft session: %w", err)
	}

	var session CPOEDraftSession
	if err := json.Unmarshal(resp, &session); err != nil {
		return nil, fmt.Errorf("failed to parse draft session: %w", err)
	}

	return &session, nil
}

// GetDraftSession returns a CPOE draft session.
// Calls KB-12 GET /api/v1/cpoe/drafts/{id} endpoint.
func (c *KB12HTTPClient) GetDraftSession(ctx context.Context, sessionID string) (*CPOEDraftSession, error) {
	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/cpoe/drafts/%s", url.PathEscape(sessionID)))
	if err != nil {
		return nil, fmt.Errorf("failed to get draft session: %w", err)
	}

	var session CPOEDraftSession
	if err := json.Unmarshal(resp, &session); err != nil {
		return nil, fmt.Errorf("failed to parse draft session: %w", err)
	}

	return &session, nil
}

// PerformSafetyCheck performs safety checks on orders.
// Calls KB-12 POST /api/v1/cpoe/safety-check endpoint.
func (c *KB12HTTPClient) PerformSafetyCheck(
	ctx context.Context,
	patientID string,
	orders []Order,
) (*SafetyCheckResult, error) {

	req := kb12SafetyCheckRequest{
		PatientID: patientID,
		Orders:    orders,
	}

	resp, err := c.callKB12(ctx, "/api/v1/cpoe/safety-check", req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform safety check: %w", err)
	}

	var result SafetyCheckResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse safety check result: %w", err)
	}

	return &result, nil
}

// SubmitOrders submits orders for processing.
// Calls KB-12 POST /api/v1/cpoe/submit endpoint.
func (c *KB12HTTPClient) SubmitOrders(
	ctx context.Context,
	patientID string,
	encounterID string,
	orders []Order,
	submittedBy string,
	overrides []string,
) ([]Order, error) {

	req := kb12SubmitOrdersRequest{
		PatientID:   patientID,
		EncounterID: encounterID,
		Orders:      orders,
		SubmittedBy: submittedBy,
		Overrides:   overrides,
	}

	resp, err := c.callKB12(ctx, "/api/v1/cpoe/submit", req)
	if err != nil {
		return nil, fmt.Errorf("failed to submit orders: %w", err)
	}

	var result kb12OrdersResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse submitted orders: %w", err)
	}

	return result.Orders, nil
}

// ============================================================================
// WORKFLOW METHODS
// ============================================================================

// ListTasks returns workflow tasks.
// Calls KB-12 GET /api/v1/workflow/tasks endpoint.
func (c *KB12HTTPClient) ListTasks(ctx context.Context, patientID string, status string) ([]KB12WorkflowTask, error) {
	reqURL := "/api/v1/workflow/tasks"
	params := url.Values{}
	if patientID != "" {
		params.Set("patient_id", patientID)
	}
	if status != "" {
		params.Set("status", status)
	}
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	resp, err := c.doGet(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	var result kb12TasksResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tasks: %w", err)
	}

	return result.Tasks, nil
}

// GetOverdueTasks returns overdue workflow tasks.
// Calls KB-12 GET /api/v1/workflow/overdue endpoint.
func (c *KB12HTTPClient) GetOverdueTasks(ctx context.Context) ([]KB12WorkflowTask, error) {
	resp, err := c.doGet(ctx, "/api/v1/workflow/overdue")
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue tasks: %w", err)
	}

	var result kb12TasksResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse overdue tasks: %w", err)
	}

	return result.Tasks, nil
}

// UpdateTaskStatus updates the status of a workflow task.
// Calls KB-12 PUT /api/v1/workflow/tasks/{id}/status endpoint.
func (c *KB12HTTPClient) UpdateTaskStatus(
	ctx context.Context,
	taskID string,
	status string,
	updatedBy string,
) error {

	req := kb12StatusUpdateRequest{
		Status:    status,
		UpdatedBy: updatedBy,
	}

	_, err := c.callKB12(ctx, fmt.Sprintf("/api/v1/workflow/tasks/%s/status", taskID), req)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

// ============================================================================
// FHIR METHODS
// ============================================================================

// GetFHIRBundle returns a FHIR Bundle for an order set instance.
// Calls KB-12 GET /api/v1/fhir/bundle/{instance_id} endpoint.
func (c *KB12HTTPClient) GetFHIRBundle(ctx context.Context, instanceID string) (map[string]interface{}, error) {
	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/fhir/bundle/%s", url.PathEscape(instanceID)))
	if err != nil {
		return nil, fmt.Errorf("failed to get FHIR bundle: %w", err)
	}

	var bundle map[string]interface{}
	if err := json.Unmarshal(resp, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse FHIR bundle: %w", err)
	}

	return bundle, nil
}

// GetFHIRCarePlan returns a FHIR CarePlan resource.
// Calls KB-12 GET /api/v1/fhir/careplan/{instance_id} endpoint.
func (c *KB12HTTPClient) GetFHIRCarePlan(ctx context.Context, instanceID string) (map[string]interface{}, error) {
	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/fhir/careplan/%s", url.PathEscape(instanceID)))
	if err != nil {
		return nil, fmt.Errorf("failed to get FHIR CarePlan: %w", err)
	}

	var carePlan map[string]interface{}
	if err := json.Unmarshal(resp, &carePlan); err != nil {
		return nil, fmt.Errorf("failed to parse FHIR CarePlan: %w", err)
	}

	return carePlan, nil
}

// ============================================================================
// PATIENT-SPECIFIC METHODS
// ============================================================================

// GetPatientOrderSets returns order sets for a patient.
// Calls KB-12 GET /api/v1/patient/{patient_id}/ordersets endpoint.
func (c *KB12HTTPClient) GetPatientOrderSets(ctx context.Context, patientID string) ([]OrderSetInstance, error) {
	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/patient/%s/ordersets", url.PathEscape(patientID)))
	if err != nil {
		return nil, fmt.Errorf("failed to get patient order sets: %w", err)
	}

	var result kb12InstancesResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse patient order sets: %w", err)
	}

	return result.Instances, nil
}

// GetPatientCarePlans returns care plans for a patient.
// Calls KB-12 GET /api/v1/patient/{patient_id}/careplans endpoint.
func (c *KB12HTTPClient) GetPatientCarePlans(ctx context.Context, patientID string) ([]CarePlanInstance, error) {
	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/patient/%s/careplans", url.PathEscape(patientID)))
	if err != nil {
		return nil, fmt.Errorf("failed to get patient care plans: %w", err)
	}

	var result kb12CarePlanInstancesResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse patient care plans: %w", err)
	}

	return result.Instances, nil
}

// ============================================================================
// HEALTH CHECK
// ============================================================================

// HealthCheck verifies KB-12 service availability.
func (c *KB12HTTPClient) HealthCheck(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/health", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("KB-12 health check failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-12 unhealthy: status %d", httpResp.StatusCode)
	}

	return nil
}

// ============================================================================
// PRIVATE HELPER METHODS
// ============================================================================

// doGet makes a GET request to KB-12 service.
func (c *KB12HTTPClient) doGet(ctx context.Context, endpoint string) ([]byte, error) {
	reqURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("KB-12 returned status %d: %s", httpResp.StatusCode, string(body))
	}

	return body, nil
}

// callKB12 makes a POST request to KB-12 service.
func (c *KB12HTTPClient) callKB12(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	reqURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("KB-12 returned status %d: %s", httpResp.StatusCode, string(body))
	}

	return body, nil
}

// ============================================================================
// REQUEST/RESPONSE TYPES (PRIVATE)
// ============================================================================

type kb12TemplatesResult struct {
	Templates []OrderSetTemplate `json:"templates"`
}

type kb12InstancesResult struct {
	Instances []OrderSetInstance `json:"instances"`
}

type kb12CarePlansResult struct {
	CarePlans []CarePlanTemplate `json:"careplans"`
}

type kb12CarePlanInstancesResult struct {
	Instances []CarePlanInstance `json:"instances"`
}

type kb12TasksResult struct {
	Tasks []KB12WorkflowTask `json:"tasks"`
}

type kb12OrdersResult struct {
	Orders []Order `json:"orders"`
}

type kb12ActivateRequest struct {
	TemplateID     string                 `json:"template_id"`
	PatientID      string                 `json:"patient_id"`
	EncounterID    string                 `json:"encounter_id"`
	ActivatedBy    string                 `json:"activated_by"`
	Customizations map[string]interface{} `json:"customizations,omitempty"`
}

type kb12ActivateCarePlanRequest struct {
	TemplateID     string                 `json:"template_id"`
	PatientID      string                 `json:"patient_id"`
	ActivatedBy    string                 `json:"activated_by"`
	Customizations map[string]interface{} `json:"customizations,omitempty"`
}

type kb12StatusUpdateRequest struct {
	Status    string `json:"status"`
	UpdatedBy string `json:"updated_by"`
}

type kb12CreateDraftRequest struct {
	PatientID   string `json:"patient_id"`
	EncounterID string `json:"encounter_id"`
	CreatedBy   string `json:"created_by"`
}

type kb12SafetyCheckRequest struct {
	PatientID string  `json:"patient_id"`
	Orders    []Order `json:"orders"`
}

type kb12SubmitOrdersRequest struct {
	PatientID   string   `json:"patient_id"`
	EncounterID string   `json:"encounter_id"`
	Orders      []Order  `json:"orders"`
	SubmittedBy string   `json:"submitted_by"`
	Overrides   []string `json:"overrides,omitempty"`
}

// ============================================================================
// INTERFACE COMPLIANCE DOCUMENTATION
// ============================================================================
//
// KB12HTTPClient implements the following interface methods:
//
// Order Set Template Methods:
//   - ListTemplates(ctx) → []OrderSetTemplate
//   - GetTemplate(ctx, templateID) → *OrderSetTemplate
//   - SearchTemplates(ctx, query, category, specialty) → []OrderSetTemplate
//   - GetTemplatesByCategory(ctx, category) → []OrderSetTemplate
//
// Order Set Activation Methods:
//   - ActivateOrderSet(ctx, templateID, patientID, encounterID, activatedBy, customizations) → *OrderSetInstance
//
// Order Set Instance Methods:
//   - ListInstances(ctx, patientID) → []OrderSetInstance
//   - GetInstance(ctx, instanceID) → *OrderSetInstance
//   - UpdateInstanceStatus(ctx, instanceID, status, updatedBy) → error
//   - UpdateOrderStatus(ctx, instanceID, orderID, status, updatedBy) → error
//
// Care Plan Template Methods:
//   - ListCarePlanTemplates(ctx) → []CarePlanTemplate
//   - GetCarePlanTemplate(ctx, templateID) → *CarePlanTemplate
//   - ActivateCarePlan(ctx, templateID, patientID, activatedBy, customizations) → *CarePlanInstance
//
// Care Plan Instance Methods:
//   - ListCarePlanInstances(ctx, patientID) → []CarePlanInstance
//   - GetCarePlanInstance(ctx, instanceID) → *CarePlanInstance
//   - UpdateCarePlanStatus(ctx, instanceID, status, updatedBy) → error
//   - UpdateGoalProgress(ctx, instanceID, goalID, progress) → error
//   - UpdateActivityStatus(ctx, instanceID, activityID, completion) → error
//
// CPOE Methods:
//   - CreateDraftSession(ctx, patientID, encounterID, createdBy) → *CPOEDraftSession
//   - GetDraftSession(ctx, sessionID) → *CPOEDraftSession
//   - PerformSafetyCheck(ctx, patientID, orders) → *SafetyCheckResult
//   - SubmitOrders(ctx, patientID, encounterID, orders, submittedBy, overrides) → []Order
//
// Workflow Methods:
//   - ListTasks(ctx, patientID, status) → []KB12WorkflowTask
//   - GetOverdueTasks(ctx) → []KB12WorkflowTask
//   - UpdateTaskStatus(ctx, taskID, status, updatedBy) → error
//
// FHIR Methods:
//   - GetFHIRBundle(ctx, instanceID) → map[string]interface{}
//   - GetFHIRCarePlan(ctx, instanceID) → map[string]interface{}
//
// Patient Methods:
//   - GetPatientOrderSets(ctx, patientID) → []OrderSetInstance
//   - GetPatientCarePlans(ctx, patientID) → []CarePlanInstance
//
// Health:
//   - HealthCheck(ctx) → error
// ============================================================================
