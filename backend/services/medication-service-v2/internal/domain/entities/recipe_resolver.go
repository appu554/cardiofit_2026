package entities

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RecipeResolver represents the core recipe resolution engine
type RecipeResolver struct {
	ID             uuid.UUID             `json:"id"`
	RecipeID       uuid.UUID             `json:"recipe_id"`
	RequestID      uuid.UUID             `json:"request_id"`
	PatientContext PatientContext        `json:"patient_context"`
	ProtocolID     string                `json:"protocol_id"`
	ResolvedFields map[string]interface{} `json:"resolved_fields"`
	Metadata       ResolutionMetadata    `json:"metadata"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

// PatientContext contains patient-specific information for resolution
type PatientContext struct {
	PatientID        string            `json:"patient_id"`
	Age              int               `json:"age"`
	Weight           float64           `json:"weight"`
	Height           float64           `json:"height"`
	Gender           string            `json:"gender"`
	PregnancyStatus  bool              `json:"pregnancy_status"`
	RenalFunction    *RenalFunction    `json:"renal_function,omitempty"`
	HepaticFunction  *HepaticFunction  `json:"hepatic_function,omitempty"`
	Allergies        []Allergy         `json:"allergies,omitempty"`
	Conditions       []Condition       `json:"conditions,omitempty"`
	CurrentMedications []CurrentMedication `json:"current_medications,omitempty"`
	LabResults       map[string]LabValue `json:"lab_results,omitempty"`
	Demographics     Demographics      `json:"demographics"`
	EncounterContext EncounterContext  `json:"encounter_context"`
}

// RenalFunction represents kidney function parameters
type RenalFunction struct {
	CreatinineClearance float64   `json:"creatinine_clearance"`
	SerumCreatinine     float64   `json:"serum_creatinine"`
	eGFR               float64   `json:"egfr"`
	Stage              string    `json:"stage"`
	LastUpdated        time.Time `json:"last_updated"`
}

// HepaticFunction represents liver function parameters
type HepaticFunction struct {
	ALT         float64   `json:"alt"`
	AST         float64   `json:"ast"`
	Bilirubin   float64   `json:"bilirubin"`
	Albumin     float64   `json:"albumin"`
	ChildPugh   string    `json:"child_pugh_class"`
	LastUpdated time.Time `json:"last_updated"`
}

// Allergy represents patient allergies
type Allergy struct {
	Allergen    string    `json:"allergen"`
	Reaction    string    `json:"reaction"`
	Severity    string    `json:"severity"`
	Type        string    `json:"type"`
	OnsetDate   time.Time `json:"onset_date"`
	Verified    bool      `json:"verified"`
}

// Condition represents patient conditions/diagnoses
type Condition struct {
	Code        string    `json:"code"`
	System      string    `json:"system"`
	Display     string    `json:"display"`
	Status      string    `json:"status"`
	OnsetDate   time.Time `json:"onset_date"`
	Severity    string    `json:"severity"`
	IsPrimary   bool      `json:"is_primary"`
}

// Removed duplicate CurrentMedication and LabValue types - defined in medication.go

// Demographics contains demographic information
type Demographics struct {
	Race          string `json:"race"`
	Ethnicity     string `json:"ethnicity"`
	Language      string `json:"language"`
	MaritalStatus string `json:"marital_status"`
	Insurance     string `json:"insurance"`
}

// EncounterContext contains encounter-specific information
type EncounterContext struct {
	EncounterID   string    `json:"encounter_id"`
	ProviderID    string    `json:"provider_id"`
	Specialty     string    `json:"specialty"`
	EncounterType string    `json:"encounter_type"`
	FacilityID    string    `json:"facility_id"`
	Date          time.Time `json:"date"`
	Urgency       string    `json:"urgency"`
}

// ResolutionMetadata contains metadata about the resolution process
type ResolutionMetadata struct {
	ProcessingTimeMs    int64                    `json:"processing_time_ms"`
	CacheHit           bool                     `json:"cache_hit"`
	CacheKey           string                   `json:"cache_key,omitempty"`
	RulesEvaluated     []RuleEvaluation         `json:"rules_evaluated"`
	FieldsResolved     int                      `json:"fields_resolved"`
	ConditionalRules   int                      `json:"conditional_rules"`
	ProtocolSpecific   bool                     `json:"protocol_specific"`
	FreshnessChecks    []FreshnessCheck         `json:"freshness_checks"`
	DataSources        map[string]string        `json:"data_sources"`
	QualityScore       float64                  `json:"quality_score"`
}

// RuleEvaluation represents evaluation of a specific rule
type RuleEvaluation struct {
	RuleID       uuid.UUID              `json:"rule_id"`
	RuleName     string                 `json:"rule_name"`
	Type         string                 `json:"type"`
	Condition    string                 `json:"condition"`
	Result       bool                   `json:"result"`
	Value        interface{}            `json:"value"`
	ProcessingMs int64                  `json:"processing_ms"`
	CacheUsed    bool                   `json:"cache_used"`
}

// Removed duplicate FreshnessCheck type - defined in snapshot.go

// FieldMergeStrategy defines how fields should be merged
type FieldMergeStrategy string

const (
	MergeStrategyReplace     FieldMergeStrategy = "replace"
	MergeStrategyAppend      FieldMergeStrategy = "append"
	MergeStrategyMerge       FieldMergeStrategy = "merge"
	MergeStrategyPrioritize  FieldMergeStrategy = "prioritize"
	MergeStrategyValidate    FieldMergeStrategy = "validate"
)

// FieldResolutionPhase defines different phases of field resolution
type FieldResolutionPhase string

const (
	PhaseCalculation FieldResolutionPhase = "calculation"
	PhaseSafety      FieldResolutionPhase = "safety"
	PhaseAudit       FieldResolutionPhase = "audit"
	PhaseConditional FieldResolutionPhase = "conditional"
	PhaseValidation  FieldResolutionPhase = "validation"
)

// ResolvedField represents a field that has been resolved
type ResolvedField struct {
	Name           string                `json:"name"`
	Value          interface{}           `json:"value"`
	Source         string                `json:"source"`
	Phase          FieldResolutionPhase  `json:"phase"`
	MergeStrategy  FieldMergeStrategy    `json:"merge_strategy"`
	Priority       int                   `json:"priority"`
	LastUpdated    time.Time             `json:"last_updated"`
	ExpiresAt      *time.Time            `json:"expires_at,omitempty"`
	Confidence     float64               `json:"confidence"`
	ValidationStatus string              `json:"validation_status"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ConditionalRule represents a rule that applies under specific conditions
type ConditionalRule struct {
	ID              uuid.UUID               `json:"id"`
	Name            string                  `json:"name"`
	Description     string                  `json:"description"`
	Protocol        string                  `json:"protocol"`
	Condition       *RuleCondition          `json:"condition"`
	Fields          []FieldRequirement      `json:"fields"`
	Priority        int                     `json:"priority"`
	CacheEnabled    bool                    `json:"cache_enabled"`
	CacheTTL        time.Duration           `json:"cache_ttl"`
	ValidationLevel ValidationLevel         `json:"validation_level"`
}

// FieldRequirement defines a required field for resolution
type FieldRequirement struct {
	Name          string              `json:"name"`
	Type          FieldType           `json:"type"`
	Required      bool                `json:"required"`
	DefaultValue  interface{}         `json:"default_value,omitempty"`
	ValidRange    *Range              `json:"valid_range,omitempty"`
	ValidValues   []string            `json:"valid_values,omitempty"`
	FreshnessReq  time.Duration       `json:"freshness_requirement"`
	Source        string              `json:"source"`
	MergeStrategy FieldMergeStrategy  `json:"merge_strategy"`
	Priority      int                 `json:"priority"`
}

// ValidationLevel represents different levels of field validation
type ValidationLevel string

const (
	ValidationLevelNone     ValidationLevel = "none"
	ValidationLevelBasic    ValidationLevel = "basic"
	ValidationLevelStrict   ValidationLevel = "strict"
	ValidationLevelCritical ValidationLevel = "critical"
)

// ProtocolResolver interface defines protocol-specific resolution methods
type ProtocolResolver interface {
	ResolveFields(ctx context.Context, patientContext PatientContext, recipe *Recipe) (*RecipeResolution, error)
	GetRequiredFields(ctx context.Context, protocolID string) ([]FieldRequirement, error)
	ValidateConditions(ctx context.Context, conditions []RuleCondition, patientContext PatientContext) (bool, error)
	GetCacheKey(patientContext PatientContext, recipe *Recipe) string
}

// RecipeResolverService interface defines the main resolver service contract
type RecipeResolverService interface {
	ResolveRecipe(ctx context.Context, request RecipeResolutionRequest) (*RecipeResolution, error)
	ResolveFields(ctx context.Context, recipe *Recipe, patientContext PatientContext) (map[string]*ResolvedField, error)
	MergeFields(ctx context.Context, fields map[FieldResolutionPhase]map[string]*ResolvedField) (map[string]*ResolvedField, error)
	ValidateFreshness(ctx context.Context, fields map[string]*ResolvedField, requirements map[string]time.Duration) error
	GetProtocolResolver(protocolID string) (ProtocolResolver, error)
	ClearCache(ctx context.Context, cacheKey string) error
}

// RecipeResolutionRequest represents a request to resolve a recipe
type RecipeResolutionRequest struct {
	RecipeID       uuid.UUID      `json:"recipe_id"`
	PatientContext PatientContext `json:"patient_context"`
	Options        ResolutionOptions `json:"options"`
	CorrelationID  string         `json:"correlation_id"`
}

// ResolutionOptions contains options for recipe resolution
type ResolutionOptions struct {
	UseCache           bool          `json:"use_cache"`
	CacheTTL           time.Duration `json:"cache_ttl"`
	SkipFreshnessCheck bool          `json:"skip_freshness_check"`
	ValidationLevel    ValidationLevel `json:"validation_level"`
	IncludeMetadata    bool          `json:"include_metadata"`
	ParallelProcessing bool          `json:"parallel_processing"`
	TimeoutMs          int64         `json:"timeout_ms"`
}

// RecipeResolutionResponse represents the result of recipe resolution
type RecipeResolutionResponse struct {
	Resolution    *RecipeResolution     `json:"resolution"`
	CacheUsed     bool                  `json:"cache_used"`
	ProcessingTimeMs int64              `json:"processing_time_ms"`
	Errors        []ResolutionError     `json:"errors,omitempty"`
	Warnings      []ResolutionWarning   `json:"warnings,omitempty"`
}

// ResolutionError represents an error during resolution
type ResolutionError struct {
	Code        string      `json:"code"`
	Message     string      `json:"message"`
	Field       string      `json:"field,omitempty"`
	Phase       string      `json:"phase,omitempty"`
	Severity    string      `json:"severity"`
	Recoverable bool        `json:"recoverable"`
	Details     interface{} `json:"details,omitempty"`
}

// ResolutionWarning represents a warning during resolution
type ResolutionWarning struct {
	Code     string      `json:"code"`
	Message  string      `json:"message"`
	Field    string      `json:"field,omitempty"`
	Phase    string      `json:"phase,omitempty"`
	Severity string      `json:"severity"`
	Details  interface{} `json:"details,omitempty"`
}

// Validate validates the recipe resolver
func (rr *RecipeResolver) Validate() error {
	if rr.RecipeID == uuid.Nil {
		return NewValidationError("recipe_id is required")
	}

	if rr.RequestID == uuid.Nil {
		return NewValidationError("request_id is required")
	}

	if rr.PatientContext.PatientID == "" {
		return NewValidationError("patient_id is required")
	}

	if rr.ProtocolID == "" {
		return NewValidationError("protocol_id is required")
	}

	return nil
}

// IsExpired checks if any resolved fields have expired
func (rr *RecipeResolver) IsExpired() bool {
	now := time.Now()
	for _, fieldData := range rr.ResolvedFields {
		if field, ok := fieldData.(*ResolvedField); ok {
			if field.ExpiresAt != nil && field.ExpiresAt.Before(now) {
				return true
			}
		}
	}
	return false
}

// GetCacheKey generates a cache key for the resolution
func (rr *RecipeResolver) GetCacheKey() string {
	return fmt.Sprintf("recipe_resolver:%s:%s:%s", 
		rr.RecipeID.String(), 
		rr.PatientContext.PatientID, 
		rr.ProtocolID)
}

// GetQualityScore calculates the overall quality score of the resolution
func (rr *RecipeResolver) GetQualityScore() float64 {
	if len(rr.ResolvedFields) == 0 {
		return 0.0
	}

	totalScore := 0.0
	count := 0

	for _, fieldData := range rr.ResolvedFields {
		if field, ok := fieldData.(*ResolvedField); ok {
			totalScore += field.Confidence
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return totalScore / float64(count)
}

// Validate validates a conditional rule
func (cr *ConditionalRule) Validate() error {
	if cr.Name == "" {
		return NewValidationError("conditional rule name is required")
	}

	if cr.Protocol == "" {
		return NewValidationError("conditional rule protocol is required")
	}

	if len(cr.Fields) == 0 {
		return NewValidationError("conditional rule must have at least one field requirement")
	}

	// Validate field requirements
	for _, field := range cr.Fields {
		if err := field.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// EvaluateCondition evaluates if the condition is met for given patient context
func (cr *ConditionalRule) EvaluateCondition(ctx context.Context, patientContext PatientContext) (bool, error) {
	if cr.Condition == nil {
		return true, nil // No condition means always applies
	}

	return evaluateRuleCondition(*cr.Condition, patientContext)
}

// Validate validates a field requirement
func (fr *FieldRequirement) Validate() error {
	if fr.Name == "" {
		return NewValidationError("field requirement name is required")
	}

	if fr.Type == "" {
		return NewValidationError("field requirement type is required")
	}

	if fr.Source == "" {
		return NewValidationError("field requirement source is required")
	}

	return nil
}

// evaluateRuleCondition evaluates a rule condition against patient context
func evaluateRuleCondition(condition RuleCondition, patientContext PatientContext) (bool, error) {
	switch condition.Field {
	case "age":
		return evaluateNumericCondition(float64(patientContext.Age), condition)
	case "weight":
		return evaluateNumericCondition(patientContext.Weight, condition)
	case "pregnancy_status":
		return evaluateBooleanCondition(patientContext.PregnancyStatus, condition)
	case "gender":
		return evaluateStringCondition(patientContext.Gender, condition)
	case "renal_function.egfr":
		if patientContext.RenalFunction != nil {
			return evaluateNumericCondition(patientContext.RenalFunction.eGFR, condition)
		}
		return false, nil
	default:
		// Handle lab values and other dynamic fields
		if labValue, exists := patientContext.LabResults[condition.Field]; exists {
			return evaluateNumericCondition(labValue.Value, condition)
		}
		return false, fmt.Errorf("unknown field: %s", condition.Field)
	}
}

// evaluateNumericCondition evaluates numeric conditions
func evaluateNumericCondition(value float64, condition RuleCondition) (bool, error) {
	conditionValue, ok := condition.Value.(float64)
	if !ok {
		if intValue, ok := condition.Value.(int); ok {
			conditionValue = float64(intValue)
		} else {
			return false, fmt.Errorf("invalid numeric condition value: %v", condition.Value)
		}
	}

	switch condition.Operator {
	case "==":
		return value == conditionValue, nil
	case "!=":
		return value != conditionValue, nil
	case ">":
		return value > conditionValue, nil
	case "<":
		return value < conditionValue, nil
	case ">=":
		return value >= conditionValue, nil
	case "<=":
		return value <= conditionValue, nil
	default:
		return false, fmt.Errorf("unsupported numeric operator: %s", condition.Operator)
	}
}

// evaluateBooleanCondition evaluates boolean conditions
func evaluateBooleanCondition(value bool, condition RuleCondition) (bool, error) {
	conditionValue, ok := condition.Value.(bool)
	if !ok {
		return false, fmt.Errorf("invalid boolean condition value: %v", condition.Value)
	}

	switch condition.Operator {
	case "==":
		return value == conditionValue, nil
	case "!=":
		return value != conditionValue, nil
	default:
		return false, fmt.Errorf("unsupported boolean operator: %s", condition.Operator)
	}
}

// evaluateStringCondition evaluates string conditions
func evaluateStringCondition(value string, condition RuleCondition) (bool, error) {
	switch condition.Operator {
	case "==":
		conditionValue, ok := condition.Value.(string)
		if !ok {
			return false, fmt.Errorf("invalid string condition value: %v", condition.Value)
		}
		return value == conditionValue, nil
	case "!=":
		conditionValue, ok := condition.Value.(string)
		if !ok {
			return false, fmt.Errorf("invalid string condition value: %v", condition.Value)
		}
		return value != conditionValue, nil
	case "in":
		values, ok := condition.Value.([]string)
		if !ok {
			return false, fmt.Errorf("invalid string array condition value: %v", condition.Value)
		}
		for _, v := range values {
			if value == v {
				return true, nil
			}
		}
		return false, nil
	case "not_in":
		values, ok := condition.Value.([]string)
		if !ok {
			return false, fmt.Errorf("invalid string array condition value: %v", condition.Value)
		}
		for _, v := range values {
			if value == v {
				return false, nil
			}
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported string operator: %s", condition.Operator)
	}
}