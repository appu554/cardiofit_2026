package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// EVENT/SIGNAL TYPES
// =============================================================================

// EventType defines the type of cross-service signal
type EventType string

const (
	// Prior Authorization Events
	EventPARequired      EventType = "PA_REQUIRED"
	EventPAApproved      EventType = "PA_APPROVED"
	EventPADenied        EventType = "PA_DENIED"
	EventPAPending       EventType = "PA_PENDING"
	EventPAExpired       EventType = "PA_EXPIRED"
	EventPANeedInfo      EventType = "PA_NEED_INFO"

	// Step Therapy Events
	EventSTRequired      EventType = "ST_REQUIRED"
	EventSTNonCompliant  EventType = "ST_NON_COMPLIANT"
	EventSTCompliant     EventType = "ST_COMPLIANT"
	EventSTExemption     EventType = "ST_EXEMPTION"

	// Quantity Limit Events
	EventQLViolation     EventType = "QL_VIOLATION"
	EventQLCompliant     EventType = "QL_COMPLIANT"
	EventQLExceeded      EventType = "QL_EXCEEDED"

	// Override Events
	EventOverrideRequested EventType = "OVERRIDE_REQUESTED"
	EventOverrideApproved  EventType = "OVERRIDE_APPROVED"
	EventOverrideDenied    EventType = "OVERRIDE_DENIED"
	EventOverrideExpired   EventType = "OVERRIDE_EXPIRED"

	// Coverage Events
	EventCoverageNotFound     EventType = "COVERAGE_NOT_FOUND"
	EventCoverageTierChanged  EventType = "COVERAGE_TIER_CHANGED"
	EventFormularyExclusion   EventType = "FORMULARY_EXCLUSION"
	EventGenericAvailable     EventType = "GENERIC_AVAILABLE"

	// Policy/Governance Events
	EventPolicyViolation     EventType = "POLICY_VIOLATION"
	EventGovernanceBreach    EventType = "GOVERNANCE_BREACH"
	EventComplianceWarning   EventType = "COMPLIANCE_WARNING"
)

// EventSeverity defines the severity level of the event
type EventSeverity string

const (
	EventSeverityCritical EventSeverity = "CRITICAL"
	EventSeverityHigh     EventSeverity = "HIGH"
	EventSeverityMedium   EventSeverity = "MEDIUM"
	EventSeverityLow      EventSeverity = "LOW"
	EventSeverityInfo     EventSeverity = "INFO"
)

// EventCategory groups related event types
type EventCategory string

const (
	EventCategoryPA         EventCategory = "PRIOR_AUTHORIZATION"
	EventCategoryST         EventCategory = "STEP_THERAPY"
	EventCategoryQL         EventCategory = "QUANTITY_LIMIT"
	EventCategoryOverride   EventCategory = "OVERRIDE"
	EventCategoryCoverage   EventCategory = "COVERAGE"
	EventCategoryGovernance EventCategory = "GOVERNANCE"
)

// EventTargetService defines which service should consume the event
type EventTargetService string

const (
	TargetKB3TemporalEngine    EventTargetService = "KB3_TEMPORAL_ENGINE"
	TargetKB5DrugInteractions  EventTargetService = "KB5_DRUG_INTERACTIONS"
	TargetKB7Terminology       EventTargetService = "KB7_TERMINOLOGY"
	TargetKB14CareNavigator    EventTargetService = "KB14_CARE_NAVIGATOR"
	TargetFlowOrchestrator     EventTargetService = "FLOW_ORCHESTRATOR"
	TargetSafetyGateway        EventTargetService = "SAFETY_GATEWAY"
	TargetAuditService         EventTargetService = "AUDIT_SERVICE"
	TargetNotificationService  EventTargetService = "NOTIFICATION_SERVICE"
	TargetCDSSEngine           EventTargetService = "CDSS_ENGINE"
)

// =============================================================================
// EVENT MODELS
// =============================================================================

// DrugContext provides drug-specific context for events
type DrugContext struct {
	RxNormCode  string  `json:"rxnorm_code"`
	DrugName    string  `json:"drug_name"`
	NDC         string  `json:"ndc,omitempty"`
	GenericName string  `json:"generic_name,omitempty"`
	DrugClass   string  `json:"drug_class,omitempty"`
	FormRoute   string  `json:"form_route,omitempty"` // e.g., "oral tablet", "injection"
}

// PatientContext provides patient-specific context for events
type PatientContext struct {
	PatientID     string   `json:"patient_id"`
	MemberID      string   `json:"member_id,omitempty"`
	Age           *int     `json:"age,omitempty"`
	Diagnoses     []string `json:"diagnoses,omitempty"`     // Active ICD-10 codes
	RiskFactors   []string `json:"risk_factors,omitempty"`  // Identified risk factors
}

// ProviderContext provides provider-specific context for events
type ProviderContext struct {
	ProviderID   string `json:"provider_id"`
	ProviderNPI  string `json:"provider_npi,omitempty"`
	ProviderType string `json:"provider_type,omitempty"`
	FacilityID   string `json:"facility_id,omitempty"`
}

// PayerContext provides payer/plan-specific context for events
type PayerEventContext struct {
	PayerID      string `json:"payer_id,omitempty"`
	PayerName    string `json:"payer_name,omitempty"`
	PlanID       string `json:"plan_id,omitempty"`
	PlanName     string `json:"plan_name,omitempty"`
	ProgramType  string `json:"program_type,omitempty"`  // Medicare, Medicaid, Commercial
	BenefitPhase string `json:"benefit_phase,omitempty"` // For Medicare Part D
}

// ActionRecommendation provides recommended actions for event consumers
type ActionRecommendation struct {
	Action       string `json:"action"`                 // Recommended action
	Priority     string `json:"priority"`               // urgent, high, normal, low
	Description  string `json:"description"`            // Human-readable description
	RequiredBy   string `json:"required_by,omitempty"`  // Who needs to take action
	DeadlineHrs  *int   `json:"deadline_hours,omitempty"` // Time to act
}

// FormularyEvent represents a cross-service event/signal from KB-6
type FormularyEvent struct {
	// Identity
	ID            uuid.UUID    `json:"id"`
	EventType     EventType    `json:"event_type"`
	Category      EventCategory `json:"category"`
	Severity      EventSeverity `json:"severity"`

	// Source
	SourceService string       `json:"source_service"`       // Always "KB6_FORMULARY"
	SourceVersion string       `json:"source_version"`
	CorrelationID string       `json:"correlation_id"`       // For tracing across services
	RequestID     string       `json:"request_id,omitempty"` // Original request ID

	// Context
	DrugContext     *DrugContext        `json:"drug_context,omitempty"`
	PatientContext  *PatientContext     `json:"patient_context,omitempty"`
	ProviderContext *ProviderContext    `json:"provider_context,omitempty"`
	PayerContext    *PayerEventContext  `json:"payer_context,omitempty"`

	// Policy Binding (Enhancement #1 integration)
	PolicyBinding   *PolicyBinding      `json:"policy_binding,omitempty"`

	// Event Details
	Reason          string              `json:"reason"`                  // Why this event occurred
	Details         interface{}         `json:"details,omitempty"`       // Type-specific details
	Recommendations []ActionRecommendation `json:"recommendations,omitempty"`

	// Target Services
	TargetServices []EventTargetService `json:"target_services,omitempty"` // Who should consume this

	// Metadata
	Timestamp       time.Time           `json:"timestamp"`
	ExpiresAt       *time.Time          `json:"expires_at,omitempty"`     // When event becomes stale
	Acknowledged    bool                `json:"acknowledged"`
	AcknowledgedAt  *time.Time          `json:"acknowledged_at,omitempty"`
	AcknowledgedBy  *string             `json:"acknowledged_by,omitempty"`
}

// =============================================================================
// SPECIFIC EVENT DETAIL STRUCTURES
// =============================================================================

// PAEventDetails provides PA-specific event details
type PAEventDetails struct {
	SubmissionID     *uuid.UUID     `json:"submission_id,omitempty"`
	RequirementID    *uuid.UUID     `json:"requirement_id,omitempty"`
	CriteriaMet      []string       `json:"criteria_met,omitempty"`
	CriteriaMissing  []string       `json:"criteria_missing,omitempty"`
	ApprovalDuration *int           `json:"approval_duration_days,omitempty"`
	ExpiresAt        *time.Time     `json:"expires_at,omitempty"`
	DenialReason     string         `json:"denial_reason,omitempty"`
}

// STEventDetails provides Step Therapy-specific event details
type STEventDetails struct {
	RuleID          *uuid.UUID `json:"rule_id,omitempty"`
	TotalSteps      int        `json:"total_steps"`
	CurrentStep     int        `json:"current_step"`
	StepsSatisfied  []int      `json:"steps_satisfied,omitempty"`
	NextStep        *Step      `json:"next_step,omitempty"`
	RequiredDrugs   []string   `json:"required_drugs,omitempty"`
	RequiredDays    int        `json:"required_days,omitempty"`
}

// QLEventDetails provides Quantity Limit-specific event details
type QLEventDetails struct {
	RequestedQty     int                    `json:"requested_quantity"`
	RequestedDays    int                    `json:"requested_days_supply"`
	LimitQty         int                    `json:"limit_quantity"`
	LimitDays        int                    `json:"limit_days_supply"`
	Violations       []QLViolation          `json:"violations,omitempty"`
	SuggestedQty     *int                   `json:"suggested_quantity,omitempty"`
	SuggestedDays    *int                   `json:"suggested_days_supply,omitempty"`
}

// OverrideEventDetails provides Override-specific event details
type OverrideEventDetails struct {
	OverrideID       uuid.UUID   `json:"override_id"`
	OverrideType     string      `json:"override_type"`    // PA, ST, QL
	Reason           string      `json:"reason"`
	Justification    string      `json:"justification,omitempty"`
	ApprovedBy       string      `json:"approved_by,omitempty"`
	DeniedBy         string      `json:"denied_by,omitempty"`
	DenialReason     string      `json:"denial_reason,omitempty"`
	ValidUntil       *time.Time  `json:"valid_until,omitempty"`
}

// GovernanceEventDetails provides Governance/Policy-specific event details
type GovernanceEventDetails struct {
	PolicyID         string      `json:"policy_id"`
	PolicyName       string      `json:"policy_name"`
	PolicyVersion    string      `json:"policy_version"`
	ViolationType    string      `json:"violation_type"`
	Jurisdiction     string      `json:"jurisdiction"`
	BindingLevel     string      `json:"binding_level"`
	EnforcementMode  string      `json:"enforcement_mode"`
	ComplianceAction string      `json:"compliance_action,omitempty"`
}

// =============================================================================
// EVENT FACTORY FUNCTIONS
// =============================================================================

// NewFormularyEvent creates a new event with standard fields populated
func NewFormularyEvent(eventType EventType, correlationID string) FormularyEvent {
	return FormularyEvent{
		ID:            uuid.New(),
		EventType:     eventType,
		Category:      GetEventCategory(eventType),
		Severity:      GetDefaultSeverity(eventType),
		SourceService: "KB6_FORMULARY",
		SourceVersion: "1.0.0",
		CorrelationID: correlationID,
		Timestamp:     time.Now().UTC(),
		Acknowledged:  false,
	}
}

// GetEventCategory returns the category for an event type
func GetEventCategory(eventType EventType) EventCategory {
	switch eventType {
	case EventPARequired, EventPAApproved, EventPADenied, EventPAPending, EventPAExpired, EventPANeedInfo:
		return EventCategoryPA
	case EventSTRequired, EventSTNonCompliant, EventSTCompliant, EventSTExemption:
		return EventCategoryST
	case EventQLViolation, EventQLCompliant, EventQLExceeded:
		return EventCategoryQL
	case EventOverrideRequested, EventOverrideApproved, EventOverrideDenied, EventOverrideExpired:
		return EventCategoryOverride
	case EventCoverageNotFound, EventCoverageTierChanged, EventFormularyExclusion, EventGenericAvailable:
		return EventCategoryCoverage
	case EventPolicyViolation, EventGovernanceBreach, EventComplianceWarning:
		return EventCategoryGovernance
	default:
		return EventCategoryCoverage
	}
}

// GetDefaultSeverity returns the default severity for an event type
func GetDefaultSeverity(eventType EventType) EventSeverity {
	switch eventType {
	case EventPADenied, EventGovernanceBreach, EventQLExceeded:
		return EventSeverityHigh
	case EventPARequired, EventSTNonCompliant, EventQLViolation, EventPolicyViolation:
		return EventSeverityMedium
	case EventPAApproved, EventSTCompliant, EventQLCompliant, EventOverrideApproved:
		return EventSeverityLow
	case EventComplianceWarning, EventGenericAvailable:
		return EventSeverityInfo
	default:
		return EventSeverityMedium
	}
}

// GetDefaultTargetServices returns the default target services for an event type
func GetDefaultTargetServices(eventType EventType) []EventTargetService {
	baseTargets := []EventTargetService{
		TargetAuditService, // Always audit
	}

	switch eventType {
	case EventPARequired, EventPADenied, EventPAPending:
		return append(baseTargets,
			TargetKB14CareNavigator,
			TargetFlowOrchestrator,
			TargetNotificationService,
		)
	case EventPAApproved:
		return append(baseTargets,
			TargetKB14CareNavigator,
			TargetFlowOrchestrator,
		)
	case EventSTNonCompliant:
		return append(baseTargets,
			TargetKB14CareNavigator,
			TargetCDSSEngine,
			TargetNotificationService,
		)
	case EventQLViolation, EventQLExceeded:
		return append(baseTargets,
			TargetSafetyGateway,
			TargetNotificationService,
		)
	case EventGovernanceBreach, EventPolicyViolation:
		return append(baseTargets,
			TargetSafetyGateway,
			TargetNotificationService,
			TargetKB14CareNavigator,
		)
	case EventOverrideApproved, EventOverrideDenied:
		return append(baseTargets,
			TargetFlowOrchestrator,
			TargetKB3TemporalEngine,
		)
	default:
		return baseTargets
	}
}

// =============================================================================
// JSON MARSHALING
// =============================================================================

// ToJSON serializes the event to JSON
func (e *FormularyEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON deserializes an event from JSON
func FromJSON(data []byte) (*FormularyEvent, error) {
	var event FormularyEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
