package entities

import (
	"time"
	"github.com/google/uuid"
)

// Recipe represents a clinical recipe for medication workflow
type Recipe struct {
	ID                     uuid.UUID               `json:"id" db:"id"`
	ProtocolID            string                  `json:"protocol_id" db:"protocol_id"`
	Name                  string                  `json:"name" db:"name"`
	Version               string                  `json:"version" db:"version"`
	Description           string                  `json:"description" db:"description"`
	Indication            string                  `json:"indication" db:"indication"`
	ContextRequirements   ContextRequirements     `json:"context_requirements" db:"context_requirements"`
	CalculationRules      []CalculationRule       `json:"calculation_rules" db:"calculation_rules"`
	SafetyRules          []SafetyRule            `json:"safety_rules" db:"safety_rules"`
	MonitoringRules      []MonitoringRule        `json:"monitoring_rules" db:"monitoring_rules"`
	CreatedAt            time.Time               `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at" db:"updated_at"`
	CreatedBy            string                  `json:"created_by" db:"created_by"`
	Status               RecipeStatus            `json:"status" db:"status"`
	TTL                  time.Duration           `json:"ttl" db:"ttl"`
	ClinicalEvidence     *ClinicalEvidence       `json:"clinical_evidence,omitempty" db:"clinical_evidence"`
	ApprovalMetadata     *ApprovalMetadata       `json:"approval_metadata,omitempty" db:"approval_metadata"`
}

// RecipeStatus represents the status of a recipe
type RecipeStatus string

const (
	RecipeStatusDraft      RecipeStatus = "draft"
	RecipeStatusReview     RecipeStatus = "review"
	RecipeStatusApproved   RecipeStatus = "approved"
	RecipeStatusActive     RecipeStatus = "active"
	RecipeStatusDeprecated RecipeStatus = "deprecated"
	RecipeStatusArchived   RecipeStatus = "archived"
)

// ContextRequirements defines what clinical context is needed
type ContextRequirements struct {
	CalculationFields   []ContextField      `json:"calculation_fields"`
	SafetyFields        []ContextField      `json:"safety_fields"`
	MonitoringFields    []ContextField      `json:"monitoring_fields"`
	FreshnessRequirements map[string]time.Duration `json:"freshness_requirements"`
	OptionalFields      []ContextField      `json:"optional_fields,omitempty"`
}

// ContextField represents a required context field
type ContextField struct {
	Name         string    `json:"name"`
	Type         FieldType `json:"type"`
	Required     bool      `json:"required"`
	Unit         string    `json:"unit,omitempty"`
	ValidRange   *Range    `json:"valid_range,omitempty"`
	Description  string    `json:"description,omitempty"`
}

// FieldType represents the type of context field
type FieldType string

const (
	FieldTypeNumber   FieldType = "number"
	FieldTypeString   FieldType = "string"
	FieldTypeBoolean  FieldType = "boolean"
	FieldTypeDate     FieldType = "date"
	FieldTypeArray    FieldType = "array"
	FieldTypeObject   FieldType = "object"
)

// Range represents a valid range for numeric fields
type Range struct {
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`
}

// CalculationRule defines how to calculate medication dosages
type CalculationRule struct {
	ID              uuid.UUID              `json:"id"`
	Name            string                 `json:"name"`
	Priority        int                    `json:"priority"`
	Condition       *RuleCondition         `json:"condition,omitempty"`
	CalculationType CalculationMethod      `json:"calculation_type"`
	Formula         string                 `json:"formula"`
	Parameters      map[string]interface{} `json:"parameters"`
	OutputUnit      string                 `json:"output_unit"`
	RoundingRule    RoundingRule           `json:"rounding_rule"`
	Adjustments     []DoseAdjustment       `json:"adjustments,omitempty"`
	Notes           string                 `json:"notes,omitempty"`
}

// RuleCondition defines when a rule should be applied
type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // ==, !=, >, <, >=, <=, in, not_in
	Value    interface{} `json:"value"`
	LogicalOperator string `json:"logical_operator,omitempty"` // and, or
	SubConditions []RuleCondition `json:"sub_conditions,omitempty"`
}

// RoundingRule defines how to round calculated doses
type RoundingRule struct {
	Type      RoundingType `json:"type"`
	Precision int          `json:"precision"`
	Direction RoundingDirection `json:"direction"`
	MinimumDose *float64   `json:"minimum_dose,omitempty"`
	MaximumDose *float64   `json:"maximum_dose,omitempty"`
}

// RoundingType represents different rounding strategies
type RoundingType string

const (
	RoundingStandard    RoundingType = "standard"
	RoundingTabletSize  RoundingType = "tablet_size"
	RoundingVialSize    RoundingType = "vial_size"
	RoundingPractical   RoundingType = "practical"
)

// RoundingDirection represents rounding direction
type RoundingDirection string

const (
	RoundingNearest RoundingDirection = "nearest"
	RoundingUp      RoundingDirection = "up"
	RoundingDown    RoundingDirection = "down"
)

// DoseAdjustment represents dose adjustments based on patient factors
type DoseAdjustment struct {
	ID         uuid.UUID      `json:"id"`
	Name       string         `json:"name"`
	Condition  RuleCondition  `json:"condition"`
	Type       AdjustmentType `json:"type"`
	Value      float64        `json:"value"`
	Unit       string         `json:"unit,omitempty"`
	Reason     string         `json:"reason"`
	Evidence   string         `json:"evidence,omitempty"`
}

// AdjustmentType represents different types of dose adjustments
type AdjustmentType string

const (
	AdjustmentMultiplier  AdjustmentType = "multiplier"
	AdjustmentAdditive    AdjustmentType = "additive"
	AdjustmentSubtractive AdjustmentType = "subtractive"
	AdjustmentReplacement AdjustmentType = "replacement"
)

// SafetyRule defines safety constraints and checks
type SafetyRule struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	Priority    int               `json:"priority"`
	Type        SafetyRuleType    `json:"type"`
	Condition   RuleCondition     `json:"condition"`
	Action      SafetyAction      `json:"action"`
	Severity    ConstraintSeverity `json:"severity"`
	Message     string            `json:"message"`
	Mitigation  string            `json:"mitigation,omitempty"`
	Evidence    string            `json:"evidence,omitempty"`
	Exceptions  []SafetyException `json:"exceptions,omitempty"`
}

// SafetyRuleType represents different types of safety rules
type SafetyRuleType string

const (
	SafetyRuleDoseLimit        SafetyRuleType = "dose_limit"
	SafetyRuleFrequencyLimit   SafetyRuleType = "frequency_limit"
	SafetyRuleDurationLimit    SafetyRuleType = "duration_limit"
	SafetyRuleLabCheck         SafetyRuleType = "lab_check"
	SafetyRuleInteraction      SafetyRuleType = "drug_interaction"
	SafetyRuleAllergy          SafetyRuleType = "allergy_check"
	SafetyRuleContraindication SafetyRuleType = "contraindication"
	SafetyRuleAge              SafetyRuleType = "age_check"
	SafetyRulePregnancy        SafetyRuleType = "pregnancy_check"
	SafetyRuleRenalFunction    SafetyRuleType = "renal_function"
	SafetyRuleHepaticFunction  SafetyRuleType = "hepatic_function"
)

// SafetyAction represents actions to take when safety rule triggers
type SafetyAction string

const (
	ActionWarn     SafetyAction = "warn"
	ActionBlock    SafetyAction = "block"
	ActionAdjust   SafetyAction = "adjust"
	ActionMonitor  SafetyAction = "monitor"
	ActionConsult  SafetyAction = "consult"
)

// SafetyException represents exceptions to safety rules
type SafetyException struct {
	ID         uuid.UUID     `json:"id"`
	Condition  RuleCondition `json:"condition"`
	Reason     string        `json:"reason"`
	RequiredBy string        `json:"required_by"`
}

// MonitoringRule defines required clinical monitoring
type MonitoringRule struct {
	ID          uuid.UUID           `json:"id"`
	Name        string              `json:"name"`
	Parameter   string              `json:"parameter"`
	Condition   *RuleCondition      `json:"condition,omitempty"`
	Frequency   MonitoringFrequency `json:"frequency"`
	Duration    *time.Duration      `json:"duration,omitempty"`
	TargetRange *string             `json:"target_range,omitempty"`
	AlertRules  []AlertRule         `json:"alert_rules,omitempty"`
	Instructions string             `json:"instructions,omitempty"`
}

// AlertRule defines when to generate alerts for monitoring
type AlertRule struct {
	ID           uuid.UUID    `json:"id"`
	Condition    RuleCondition `json:"condition"`
	AlertLevel   AlertLevel   `json:"alert_level"`
	Message      string       `json:"message"`
	Action       string       `json:"action"`
	NotifyRoles  []string     `json:"notify_roles,omitempty"`
}

// AlertLevel represents different alert severity levels
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
	AlertLevelUrgent   AlertLevel = "urgent"
)

// ClinicalEvidence contains supporting clinical evidence for the recipe
type ClinicalEvidence struct {
	Guidelines   []GuidelineReference `json:"guidelines,omitempty"`
	Studies      []StudyReference     `json:"studies,omitempty"`
	Expert       *ExpertOpinion       `json:"expert_opinion,omitempty"`
	LastUpdated  time.Time            `json:"last_updated"`
	EvidenceLevel EvidenceLevel       `json:"evidence_level"`
}

// GuidelineReference references clinical guidelines
type GuidelineReference struct {
	Organization string `json:"organization"`
	Title        string `json:"title"`
	Version      string `json:"version"`
	URL          string `json:"url,omitempty"`
	Relevance    string `json:"relevance"`
}

// StudyReference references clinical studies
type StudyReference struct {
	Title       string `json:"title"`
	Authors     string `json:"authors"`
	Journal     string `json:"journal"`
	Year        int    `json:"year"`
	DOI         string `json:"doi,omitempty"`
	PMID        string `json:"pmid,omitempty"`
	StudyType   string `json:"study_type"`
	Relevance   string `json:"relevance"`
}

// ExpertOpinion represents expert clinical opinion
type ExpertOpinion struct {
	Expert      string    `json:"expert"`
	Institution string    `json:"institution"`
	Opinion     string    `json:"opinion"`
	Date        time.Time `json:"date"`
}

// EvidenceLevel represents the level of clinical evidence
type EvidenceLevel string

const (
	EvidenceLevelA EvidenceLevel = "A" // High quality evidence
	EvidenceLevelB EvidenceLevel = "B" // Moderate quality evidence
	EvidenceLevelC EvidenceLevel = "C" // Low quality evidence
	EvidenceLevelD EvidenceLevel = "D" // Very low quality evidence
	EvidenceExpert EvidenceLevel = "Expert" // Expert opinion
)

// ApprovalMetadata contains approval and audit information
type ApprovalMetadata struct {
	ApprovedBy      string    `json:"approved_by"`
	ApprovedAt      time.Time `json:"approved_at"`
	ApprovalNotes   string    `json:"approval_notes,omitempty"`
	ReviewedBy      []string  `json:"reviewed_by,omitempty"`
	ReviewComments  string    `json:"review_comments,omitempty"`
	ComplianceCheck bool      `json:"compliance_check"`
	RiskAssessment  string    `json:"risk_assessment,omitempty"`
}

// RecipeResolution represents the result of resolving a recipe
type RecipeResolution struct {
	RecipeID        uuid.UUID               `json:"recipe_id"`
	ContextSnapshot map[string]interface{}  `json:"context_snapshot"`
	CalculatedDoses []CalculatedDose        `json:"calculated_doses"`
	SafetyViolations []SafetyViolation      `json:"safety_violations,omitempty"`
	MonitoringPlan  []MonitoringInstruction `json:"monitoring_plan,omitempty"`
	ResolutionTime  time.Time               `json:"resolution_time"`
	ProcessingTimeMs int64                  `json:"processing_time_ms"`
	ConfidenceScore float64                 `json:"confidence_score"`
	Warnings        []string                `json:"warnings,omitempty"`
}

// CalculatedDose represents a calculated medication dose
type CalculatedDose struct {
	RuleID          uuid.UUID `json:"rule_id"`
	RuleName        string    `json:"rule_name"`
	CalculatedValue float64   `json:"calculated_value"`
	Unit            string    `json:"unit"`
	RoundedValue    float64   `json:"rounded_value"`
	Formula         string    `json:"formula"`
	InputValues     map[string]interface{} `json:"input_values"`
	Adjustments     []AppliedAdjustment    `json:"adjustments,omitempty"`
}

// AppliedAdjustment represents an adjustment that was applied
type AppliedAdjustment struct {
	AdjustmentID   uuid.UUID `json:"adjustment_id"`
	AdjustmentName string    `json:"adjustment_name"`
	OriginalValue  float64   `json:"original_value"`
	AdjustedValue  float64   `json:"adjusted_value"`
	Reason         string    `json:"reason"`
}

// SafetyViolation represents a safety rule violation
type SafetyViolation struct {
	RuleID      uuid.UUID         `json:"rule_id"`
	RuleName    string            `json:"rule_name"`
	Type        SafetyRuleType    `json:"type"`
	Severity    ConstraintSeverity `json:"severity"`
	Message     string            `json:"message"`
	Value       interface{}       `json:"value"`
	Threshold   interface{}       `json:"threshold"`
	Action      SafetyAction      `json:"action"`
	Mitigation  string            `json:"mitigation,omitempty"`
	CanOverride bool              `json:"can_override"`
}

// MonitoringInstruction represents required monitoring
type MonitoringInstruction struct {
	RuleID       uuid.UUID           `json:"rule_id"`
	Parameter    string              `json:"parameter"`
	Frequency    MonitoringFrequency `json:"frequency"`
	Duration     *time.Duration      `json:"duration,omitempty"`
	TargetRange  *string             `json:"target_range,omitempty"`
	Instructions string              `json:"instructions"`
	Priority     int                 `json:"priority"`
}

// Validate validates the recipe structure
func (r *Recipe) Validate() error {
	if r.ProtocolID == "" {
		return NewValidationError("protocol_id is required")
	}

	if r.Name == "" {
		return NewValidationError("name is required")
	}

	if r.Indication == "" {
		return NewValidationError("indication is required")
	}

	if len(r.CalculationRules) == 0 {
		return NewValidationError("at least one calculation rule is required")
	}

	// Validate calculation rules
	for _, rule := range r.CalculationRules {
		if err := rule.Validate(); err != nil {
			return err
		}
	}

	// Validate safety rules
	for _, rule := range r.SafetyRules {
		if err := rule.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// IsActive checks if the recipe is active and valid
func (r *Recipe) IsActive() bool {
	return r.Status == RecipeStatusActive
}

// IsExpired checks if the recipe has expired based on TTL
func (r *Recipe) IsExpired() bool {
	if r.TTL == 0 {
		return false // No expiry
	}
	expiryTime := r.UpdatedAt.Add(r.TTL)
	return time.Now().After(expiryTime)
}

// Validate validates a calculation rule
func (cr *CalculationRule) Validate() error {
	if cr.Name == "" {
		return NewValidationError("calculation rule name is required")
	}

	if cr.Formula == "" {
		return NewValidationError("calculation rule formula is required")
	}

	if cr.OutputUnit == "" {
		return NewValidationError("calculation rule output unit is required")
	}

	return nil
}

// Validate validates a safety rule
func (sr *SafetyRule) Validate() error {
	if sr.Name == "" {
		return NewValidationError("safety rule name is required")
	}

	if sr.Message == "" {
		return NewValidationError("safety rule message is required")
	}

	return nil
}