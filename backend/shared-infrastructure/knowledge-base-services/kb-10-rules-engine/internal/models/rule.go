// Package models defines the domain models for the Clinical Rules Engine
package models

import (
	"encoding/json"
	"time"
)

// Rule Types
const (
	RuleTypeAlert          = "ALERT"
	RuleTypeInference      = "INFERENCE"
	RuleTypeValidation     = "VALIDATION"
	RuleTypeEscalation     = "ESCALATION"
	RuleTypeSuppression    = "SUPPRESSION"
	RuleTypeDerivation     = "DERIVATION"
	RuleTypeRecommendation = "RECOMMENDATION"
	RuleTypeConflict       = "CONFLICT"
)

// Rule Severities
const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityModerate = "MODERATE"
	SeverityLow      = "LOW"
	SeverityInfo     = "INFO"
)

// Rule Status
const (
	StatusActive   = "ACTIVE"
	StatusInactive = "INACTIVE"
	StatusDraft    = "DRAFT"
	StatusRetired  = "RETIRED"
)

// Condition Operators - 20+ operators for comprehensive rule evaluation
const (
	OperatorEQ          = "EQ"          // Equals
	OperatorNEQ         = "NEQ"         // Not equals
	OperatorGT          = "GT"          // Greater than
	OperatorGTE         = "GTE"         // Greater than or equal
	OperatorLT          = "LT"          // Less than
	OperatorLTE         = "LTE"         // Less than or equal
	OperatorCONTAINS    = "CONTAINS"    // String contains
	OperatorNOTCONTAINS = "NOT_CONTAINS" // String does not contain
	OperatorIN          = "IN"          // Value in list
	OperatorNOTIN       = "NOT_IN"      // Value not in list
	OperatorBETWEEN     = "BETWEEN"     // Value between min and max
	OperatorEXISTS      = "EXISTS"      // Field exists
	OperatorNOTEXISTS   = "NOT_EXISTS"  // Field does not exist
	OperatorISNULL      = "IS_NULL"     // Field is null
	OperatorISNOTNULL   = "IS_NOT_NULL" // Field is not null
	OperatorMATCHES     = "MATCHES"     // Regex pattern match
	OperatorSTARTSWITH  = "STARTS_WITH" // String starts with
	OperatorENDSWITH    = "ENDS_WITH"   // String ends with
	OperatorAGEGT       = "AGE_GT"      // Age greater than
	OperatorAGELT       = "AGE_LT"      // Age less than
	OperatorAGEBETWEEN  = "AGE_BETWEEN" // Age between
	OperatorWITHINDAYS  = "WITHIN_DAYS" // Date within N days
	OperatorBEFOREDAYS  = "BEFORE_DAYS" // Date before N days ago
	OperatorAFTERDAYS   = "AFTER_DAYS"  // Date after N days ago
)

// Condition Logic
const (
	LogicAND = "AND"
	LogicOR  = "OR"
)

// Action Types
const (
	ActionTypeAlert         = "ALERT"
	ActionTypeEscalate      = "ESCALATE"
	ActionTypeNotify        = "NOTIFY"
	ActionTypeRecommend     = "RECOMMEND"
	ActionTypeInference     = "INFERENCE"
	ActionTypeDerivation    = "DERIVATION"
	ActionTypeSuppress      = "SUPPRESS"
	ActionTypeLog           = "LOG"
	ActionTypeWebhook       = "WEBHOOK"
	ActionTypeCQLExpression = "CQL_EXPRESSION"
)

// Rule represents a clinical decision rule
type Rule struct {
	ID             string      `yaml:"id" json:"id"`
	Name           string      `yaml:"name" json:"name"`
	Description    string      `yaml:"description" json:"description"`
	Type           string      `yaml:"type" json:"type"`
	Category       string      `yaml:"category" json:"category"`
	Severity       string      `yaml:"severity" json:"severity"`
	Status         string      `yaml:"status" json:"status"`
	Priority       int         `yaml:"priority" json:"priority"`
	Version        string      `yaml:"version" json:"version"`
	Conditions     []Condition `yaml:"conditions" json:"conditions"`
	ConditionLogic string      `yaml:"condition_logic" json:"condition_logic"`
	Actions        []Action    `yaml:"actions" json:"actions"`
	Evidence       Evidence    `yaml:"evidence" json:"evidence"`
	Tags           []string    `yaml:"tags" json:"tags"`
	Metadata       Metadata    `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

// Condition represents a rule condition to evaluate
type Condition struct {
	ID          string      `yaml:"id,omitempty" json:"id,omitempty"`
	Field       string      `yaml:"field" json:"field"`
	Operator    string      `yaml:"operator" json:"operator"`
	Value       interface{} `yaml:"value" json:"value"`
	Unit        string      `yaml:"unit,omitempty" json:"unit,omitempty"`
	CQLExpr     string      `yaml:"cql_expression,omitempty" json:"cql_expression,omitempty"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
}

// Action represents an action to execute when a rule triggers
type Action struct {
	Type       string            `yaml:"type" json:"type"`
	Message    string            `yaml:"message,omitempty" json:"message,omitempty"`
	Priority   string            `yaml:"priority,omitempty" json:"priority,omitempty"`
	Parameters map[string]string `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Recipients []string          `yaml:"recipients,omitempty" json:"recipients,omitempty"`
	Channel    string            `yaml:"channel,omitempty" json:"channel,omitempty"`
	Template   string            `yaml:"template,omitempty" json:"template,omitempty"`
}

// Evidence represents the clinical evidence supporting a rule
type Evidence struct {
	Level       string   `yaml:"level" json:"level"`
	Source      string   `yaml:"source" json:"source"`
	References  []string `yaml:"references,omitempty" json:"references,omitempty"`
	LastReviewed string  `yaml:"last_reviewed,omitempty" json:"last_reviewed,omitempty"`
	Reviewers   []string `yaml:"reviewers,omitempty" json:"reviewers,omitempty"`
}

// Metadata contains additional rule metadata
type Metadata struct {
	Author       string            `yaml:"author,omitempty" json:"author,omitempty"`
	Organization string            `yaml:"organization,omitempty" json:"organization,omitempty"`
	Department   string            `yaml:"department,omitempty" json:"department,omitempty"`
	EffectiveDate string           `yaml:"effective_date,omitempty" json:"effective_date,omitempty"`
	ExpiryDate   string            `yaml:"expiry_date,omitempty" json:"expiry_date,omitempty"`
	CustomFields map[string]string `yaml:"custom_fields,omitempty" json:"custom_fields,omitempty"`
}

// RuleFile represents a YAML file containing multiple rules
type RuleFile struct {
	Type        string `yaml:"type"`
	Version     string `yaml:"version"`
	Description string `yaml:"description,omitempty"`
	Rules       []Rule `yaml:"rules"`
}

// EvaluationContext contains the patient context for rule evaluation
type EvaluationContext struct {
	PatientID    string                 `json:"patient_id"`
	EncounterID  string                 `json:"encounter_id,omitempty"`
	Labs         map[string]LabValue    `json:"labs,omitempty"`
	Vitals       map[string]VitalSign   `json:"vitals,omitempty"`
	Medications  []MedicationContext    `json:"medications,omitempty"`
	Conditions   []ConditionContext     `json:"conditions,omitempty"`
	Allergies    []AllergyContext       `json:"allergies,omitempty"`
	Patient      PatientContext         `json:"patient,omitempty"`
	Encounter    EncounterContext       `json:"encounter,omitempty"`
	CustomData   map[string]interface{} `json:"custom_data,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	RequestID    string                 `json:"request_id,omitempty"`
}

// LabValue represents a laboratory result
type LabValue struct {
	Value         float64   `json:"value"`
	Unit          string    `json:"unit,omitempty"`
	ReferenceMin  float64   `json:"reference_min,omitempty"`
	ReferenceMax  float64   `json:"reference_max,omitempty"`
	Status        string    `json:"status,omitempty"`
	Date          time.Time `json:"date,omitempty"`
	LoincCode     string    `json:"loinc_code,omitempty"`
	Interpretation string   `json:"interpretation,omitempty"`
}

// VitalSign represents a vital sign measurement
type VitalSign struct {
	Value  float64   `json:"value"`
	Unit   string    `json:"unit,omitempty"`
	Date   time.Time `json:"date,omitempty"`
	Method string    `json:"method,omitempty"`
}

// MedicationContext represents medication information
type MedicationContext struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	RxNormCode  string    `json:"rxnorm_code,omitempty"`
	Dose        string    `json:"dose,omitempty"`
	Unit        string    `json:"unit,omitempty"`
	Route       string    `json:"route,omitempty"`
	Frequency   string    `json:"frequency,omitempty"`
	Status      string    `json:"status,omitempty"`
	StartDate   time.Time `json:"start_date,omitempty"`
	EndDate     time.Time `json:"end_date,omitempty"`
	DrugClass   string    `json:"drug_class,omitempty"`
}

// ConditionContext represents a clinical condition/diagnosis
type ConditionContext struct {
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	ICD10Code  string    `json:"icd10_code,omitempty"`
	SnomedCode string    `json:"snomed_code,omitempty"`
	Status     string    `json:"status,omitempty"`
	OnsetDate  time.Time `json:"onset_date,omitempty"`
	Severity   string    `json:"severity,omitempty"`
	Category   string    `json:"category,omitempty"`
}

// AllergyContext represents allergy information
type AllergyContext struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Severity    string `json:"severity,omitempty"`
	Reaction    string `json:"reaction,omitempty"`
	Status      string `json:"status,omitempty"`
}

// PatientContext represents patient demographic information
type PatientContext struct {
	DateOfBirth time.Time `json:"date_of_birth,omitempty"`
	Age         int       `json:"age,omitempty"`
	Gender      string    `json:"gender,omitempty"`
	Weight      float64   `json:"weight,omitempty"`
	Height      float64   `json:"height,omitempty"`
	BSA         float64   `json:"bsa,omitempty"`
	Pregnant    bool      `json:"pregnant,omitempty"`
	Lactating   bool      `json:"lactating,omitempty"`
}

// EncounterContext represents encounter information
type EncounterContext struct {
	Type        string    `json:"type,omitempty"`
	Class       string    `json:"class,omitempty"`
	Status      string    `json:"status,omitempty"`
	StartDate   time.Time `json:"start_date,omitempty"`
	Location    string    `json:"location,omitempty"`
	Department  string    `json:"department,omitempty"`
	Provider    string    `json:"provider,omitempty"`
}

// EvaluationResult represents the result of evaluating a rule
type EvaluationResult struct {
	RuleID         string         `json:"rule_id"`
	RuleName       string         `json:"rule_name"`
	RuleType       string         `json:"rule_type"`
	Triggered      bool           `json:"triggered"`
	Severity       string         `json:"severity,omitempty"`
	Category       string         `json:"category,omitempty"`
	Actions        []ActionResult `json:"actions,omitempty"`
	Message        string         `json:"message,omitempty"`
	Evidence       Evidence       `json:"evidence,omitempty"`
	ConditionsMet  []string       `json:"conditions_met,omitempty"`
	ConditionsFailed []string     `json:"conditions_failed,omitempty"`
	ExecutedAt     time.Time      `json:"executed_at"`
	ExecutionTimeMs float64       `json:"execution_time_ms"`
	CacheHit       bool           `json:"cache_hit"`
	Error          string         `json:"error,omitempty"`
}

// ActionResult represents the result of executing an action
type ActionResult struct {
	Type       string                 `json:"type"`
	Success    bool                   `json:"success"`
	Message    string                 `json:"message,omitempty"`
	AlertID    string                 `json:"alert_id,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// Alert represents a clinical alert generated by a rule
type Alert struct {
	ID             string                 `json:"id"`
	RuleID         string                 `json:"rule_id"`
	RuleName       string                 `json:"rule_name"`
	PatientID      string                 `json:"patient_id"`
	EncounterID    string                 `json:"encounter_id,omitempty"`
	Severity       string                 `json:"severity"`
	Category       string                 `json:"category"`
	Message        string                 `json:"message"`
	Details        string                 `json:"details,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
	Status         string                 `json:"status"`
	Priority       string                 `json:"priority,omitempty"`
	AcknowledgedBy string                 `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty"`
	ResolvedBy     string                 `json:"resolved_by,omitempty"`
	ResolvedAt     *time.Time             `json:"resolved_at,omitempty"`
	Resolution     string                 `json:"resolution,omitempty"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// Alert Status constants
const (
	AlertStatusActive       = "active"
	AlertStatusAcknowledged = "acknowledged"
	AlertStatusResolved     = "resolved"
	AlertStatusExpired      = "expired"
	AlertStatusSuppressed   = "suppressed"
)

// RuleExecution represents an audit record of a rule execution
type RuleExecution struct {
	ID              string                 `json:"id"`
	RuleID          string                 `json:"rule_id"`
	RuleName        string                 `json:"rule_name"`
	PatientID       string                 `json:"patient_id"`
	EncounterID     string                 `json:"encounter_id,omitempty"`
	Triggered       bool                   `json:"triggered"`
	Context         map[string]interface{} `json:"context,omitempty"`
	Result          json.RawMessage        `json:"result,omitempty"`
	ExecutionTimeMs float64                `json:"execution_time_ms"`
	CacheHit        bool                   `json:"cache_hit"`
	Error           string                 `json:"error,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
}

// StoreStats represents statistics about the rule store
type StoreStats struct {
	TotalRules      int            `json:"total_rules"`
	ActiveRules     int            `json:"active_rules"`
	RulesByType     map[string]int `json:"rules_by_type"`
	RulesByCategory map[string]int `json:"rules_by_category"`
	RulesBySeverity map[string]int `json:"rules_by_severity"`
	LastReloadAt    time.Time      `json:"last_reload_at"`
	LoadDurationMs  float64        `json:"load_duration_ms"`
}

// ExecutionStats represents statistics for rule executions
type ExecutionStats struct {
	RuleID           string    `json:"rule_id"`
	TotalExecutions  int64     `json:"total_executions"`
	TriggerCount     int64     `json:"trigger_count"`
	TriggerRate      float64   `json:"trigger_rate"`
	AvgExecutionMs   float64   `json:"avg_execution_ms"`
	CacheHitRate     float64   `json:"cache_hit_rate"`
	LastExecutedAt   time.Time `json:"last_executed_at"`
	LastTriggeredAt  time.Time `json:"last_triggered_at"`
}

// Validate validates a rule's structure
func (r *Rule) Validate() error {
	if r.ID == "" {
		return ErrRuleIDRequired
	}
	if r.Name == "" {
		return ErrRuleNameRequired
	}
	if r.Type == "" {
		return ErrRuleTypeRequired
	}
	if !isValidRuleType(r.Type) {
		return ErrInvalidRuleType
	}
	if len(r.Conditions) == 0 && r.Type != RuleTypeSuppression {
		return ErrConditionsRequired
	}
	if len(r.Actions) == 0 {
		return ErrActionsRequired
	}
	return nil
}

func isValidRuleType(t string) bool {
	switch t {
	case RuleTypeAlert, RuleTypeInference, RuleTypeValidation,
		RuleTypeEscalation, RuleTypeSuppression, RuleTypeDerivation,
		RuleTypeRecommendation, RuleTypeConflict:
		return true
	}
	return false
}

// Rule validation errors
var (
	ErrRuleIDRequired     = &ValidationError{Field: "id", Message: "rule ID is required"}
	ErrRuleNameRequired   = &ValidationError{Field: "name", Message: "rule name is required"}
	ErrRuleTypeRequired   = &ValidationError{Field: "type", Message: "rule type is required"}
	ErrInvalidRuleType    = &ValidationError{Field: "type", Message: "invalid rule type"}
	ErrConditionsRequired = &ValidationError{Field: "conditions", Message: "at least one condition is required"}
	ErrActionsRequired    = &ValidationError{Field: "actions", Message: "at least one action is required"}
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
