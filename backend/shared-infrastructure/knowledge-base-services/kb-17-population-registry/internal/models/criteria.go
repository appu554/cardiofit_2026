// Package models contains domain models for KB-17 Population Registry
package models

import (
	"time"
)

// CriteriaType represents the type of criteria being evaluated
type CriteriaType string

const (
	CriteriaTypeDiagnosis   CriteriaType = "DIAGNOSIS"
	CriteriaTypeLabResult   CriteriaType = "LAB_RESULT"
	CriteriaTypeMedication  CriteriaType = "MEDICATION"
	CriteriaTypeProblemList CriteriaType = "PROBLEM_LIST"
	CriteriaTypeAge         CriteriaType = "AGE"
	CriteriaTypeGender      CriteriaType = "GENDER"
	CriteriaTypeVitalSign   CriteriaType = "VITAL_SIGN"
	CriteriaTypeRiskScore   CriteriaType = "RISK_SCORE"
)

// IsValid checks if the criteria type is valid
func (t CriteriaType) IsValid() bool {
	switch t {
	case CriteriaTypeDiagnosis, CriteriaTypeLabResult, CriteriaTypeMedication,
		CriteriaTypeProblemList, CriteriaTypeAge, CriteriaTypeGender,
		CriteriaTypeVitalSign, CriteriaTypeRiskScore:
		return true
	}
	return false
}

// CriteriaOperator represents comparison operators for criteria evaluation
type CriteriaOperator string

const (
	OperatorEquals         CriteriaOperator = "EQUALS"
	OperatorNotEquals      CriteriaOperator = "NOT_EQUALS"
	OperatorStartsWith     CriteriaOperator = "STARTS_WITH"
	OperatorEndsWith       CriteriaOperator = "ENDS_WITH"
	OperatorContains       CriteriaOperator = "CONTAINS"
	OperatorIn             CriteriaOperator = "IN"
	OperatorNotIn          CriteriaOperator = "NOT_IN"
	OperatorGreaterThan    CriteriaOperator = "GREATER_THAN"
	OperatorGreaterOrEqual CriteriaOperator = "GREATER_OR_EQUAL"
	OperatorLessThan       CriteriaOperator = "LESS_THAN"
	OperatorLessOrEqual    CriteriaOperator = "LESS_OR_EQUAL"
	OperatorBetween        CriteriaOperator = "BETWEEN"
	OperatorExists         CriteriaOperator = "EXISTS"
	OperatorNotExists      CriteriaOperator = "NOT_EXISTS"
)

// IsValid checks if the operator is valid
func (o CriteriaOperator) IsValid() bool {
	switch o {
	case OperatorEquals, OperatorNotEquals, OperatorStartsWith, OperatorEndsWith,
		OperatorContains, OperatorIn, OperatorNotIn, OperatorGreaterThan,
		OperatorGreaterOrEqual, OperatorLessThan, OperatorLessOrEqual,
		OperatorBetween, OperatorExists, OperatorNotExists:
		return true
	}
	return false
}

// LogicalOperator represents AND/OR logic for criteria groups
type LogicalOperator string

const (
	LogicalAnd LogicalOperator = "AND"
	LogicalOr  LogicalOperator = "OR"
)

// CodeSystem represents the code system for terminology
type CodeSystem string

const (
	CodeSystemICD10   CodeSystem = "ICD-10"
	CodeSystemICD10CM CodeSystem = "ICD-10-CM"
	CodeSystemSNOMED  CodeSystem = "SNOMED-CT"
	CodeSystemLOINC   CodeSystem = "LOINC"
	CodeSystemRxNorm  CodeSystem = "RxNorm"
	CodeSystemCPT     CodeSystem = "CPT"
	CodeSystemHCPCS   CodeSystem = "HCPCS"
)

// Criterion represents a single evaluation criterion
type Criterion struct {
	ID          string           `json:"id,omitempty"`
	Type        CriteriaType     `json:"type"`
	Field       string           `json:"field"`           // e.g., "code", "value", "status"
	Operator    CriteriaOperator `json:"operator"`
	Value       interface{}      `json:"value"`           // single value
	Values      []interface{}    `json:"values,omitempty"` // for IN, BETWEEN operators
	CodeSystem  CodeSystem       `json:"code_system,omitempty"`
	Unit        string           `json:"unit,omitempty"`  // for lab values
	TimeWindow  *TimeWindow      `json:"time_window,omitempty"`
	Description string           `json:"description,omitempty"`
}

// TimeWindow defines a time-based filter for criteria
type TimeWindow struct {
	Within     string     `json:"within,omitempty"`      // e.g., "30d", "1y"
	After      *time.Time `json:"after,omitempty"`
	Before     *time.Time `json:"before,omitempty"`
	MostRecent bool       `json:"most_recent,omitempty"` // use most recent value
}

// CriteriaGroup represents a group of criteria with logical operators
type CriteriaGroup struct {
	ID          string          `json:"id"`
	Operator    LogicalOperator `json:"operator"` // AND/OR between criteria in this group
	Criteria    []Criterion     `json:"criteria"`
	Description string          `json:"description,omitempty"`
}

// CriteriaEvaluationResult represents the result of evaluating criteria for a patient
type CriteriaEvaluationResult struct {
	PatientID         string           `json:"patient_id"`
	RegistryCode      RegistryCode     `json:"registry_code"`
	MeetsInclusion    bool             `json:"meets_inclusion"`
	MeetsExclusion    bool             `json:"meets_exclusion"`
	Eligible          bool             `json:"eligible"`
	SuggestedRiskTier RiskTier         `json:"suggested_risk_tier"`
	MatchedCriteria   []MatchedCriterion `json:"matched_criteria,omitempty"`
	ExcludedCriteria  []MatchedCriterion `json:"excluded_criteria,omitempty"`
	RiskFactors       []RiskFactor     `json:"risk_factors,omitempty"`
	EvaluatedAt       time.Time        `json:"evaluated_at"`
	EvaluationDetails map[string]interface{} `json:"evaluation_details,omitempty"`
}

// MatchedCriterion represents a criterion that was matched during evaluation
type MatchedCriterion struct {
	CriterionID   string      `json:"criterion_id"`
	CriteriaGroup string      `json:"criteria_group,omitempty"`
	Type          CriteriaType `json:"type"`
	Field         string      `json:"field"`
	MatchedValue  interface{} `json:"matched_value"`
	Description   string      `json:"description,omitempty"`
}

// RiskFactor represents a factor contributing to risk tier
type RiskFactor struct {
	Name        string      `json:"name"`
	Value       interface{} `json:"value"`
	Impact      string      `json:"impact"`  // "LOW", "MODERATE", "HIGH"
	Source      string      `json:"source"`
	Description string      `json:"description,omitempty"`
}

// EvaluateRequest represents a request to evaluate a patient's eligibility
type EvaluateRequest struct {
	PatientID     string                 `json:"patient_id" binding:"required"`
	RegistryCode  RegistryCode           `json:"registry_code,omitempty"` // if empty, evaluate all registries
	PatientData   *PatientClinicalData   `json:"patient_data,omitempty"`   // optional pre-loaded data
	IncludeDetails bool                  `json:"include_details,omitempty"`
}

// PatientClinicalData represents clinical data for a patient used in evaluation
type PatientClinicalData struct {
	PatientID    string           `json:"patient_id"`
	Demographics *Demographics    `json:"demographics,omitempty"`
	Diagnoses    []Diagnosis      `json:"diagnoses,omitempty"`
	LabResults   []LabResult      `json:"lab_results,omitempty"`
	Medications  []Medication     `json:"medications,omitempty"`
	Problems     []Problem        `json:"problems,omitempty"`
	VitalSigns   []VitalSign      `json:"vital_signs,omitempty"`
	RiskScores   []RiskScoreData  `json:"risk_scores,omitempty"`
}

// Demographics holds patient demographic information
type Demographics struct {
	BirthDate  *time.Time `json:"birth_date,omitempty"`
	Age        int        `json:"age,omitempty"`
	Gender     string     `json:"gender,omitempty"`
	Ethnicity  string     `json:"ethnicity,omitempty"`
	Race       string     `json:"race,omitempty"`
}

// Diagnosis represents a clinical diagnosis
type Diagnosis struct {
	Code        string     `json:"code"`
	CodeSystem  CodeSystem `json:"code_system"`
	Display     string     `json:"display,omitempty"`
	Status      string     `json:"status,omitempty"` // active, resolved, etc.
	OnsetDate   *time.Time `json:"onset_date,omitempty"`
	RecordedAt  time.Time  `json:"recorded_at"`
}

// LabResult represents a laboratory result
type LabResult struct {
	Code        string      `json:"code"`        // LOINC code
	CodeSystem  CodeSystem  `json:"code_system"`
	Display     string      `json:"display,omitempty"`
	Value       interface{} `json:"value"`
	Unit        string      `json:"unit,omitempty"`
	ReferenceRange *ReferenceRange `json:"reference_range,omitempty"`
	Status      string      `json:"status,omitempty"`
	EffectiveAt time.Time   `json:"effective_at"`
}

// ReferenceRange represents lab result reference ranges
type ReferenceRange struct {
	Low  float64 `json:"low,omitempty"`
	High float64 `json:"high,omitempty"`
	Text string  `json:"text,omitempty"`
}

// Medication represents an active medication
type Medication struct {
	Code        string     `json:"code"`        // RxNorm code
	CodeSystem  CodeSystem `json:"code_system"`
	Display     string     `json:"display,omitempty"`
	Status      string     `json:"status,omitempty"` // active, completed, etc.
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}

// Problem represents a problem list entry
type Problem struct {
	Code        string     `json:"code"`
	CodeSystem  CodeSystem `json:"code_system"`
	Display     string     `json:"display,omitempty"`
	Status      string     `json:"status,omitempty"` // active, inactive, resolved
	OnsetDate   *time.Time `json:"onset_date,omitempty"`
	RecordedAt  time.Time  `json:"recorded_at"`
}

// VitalSign represents a vital sign measurement
type VitalSign struct {
	Type        string      `json:"type"` // BP, HR, Temp, SpO2, etc.
	Code        string      `json:"code,omitempty"`
	CodeSystem  CodeSystem  `json:"code_system,omitempty"`
	Value       interface{} `json:"value"`
	Unit        string      `json:"unit,omitempty"`
	EffectiveAt time.Time   `json:"effective_at"`
}

// RiskScoreData represents a calculated risk score
type RiskScoreData struct {
	ScoreType   string    `json:"score_type"` // e.g., "HAS-BLED", "ASCVD", "eGFR"
	Value       float64   `json:"value"`
	Category    string    `json:"category,omitempty"` // e.g., "LOW", "MODERATE", "HIGH"
	CalculatedAt time.Time `json:"calculated_at"`
}

// EvaluateResponse represents the response for evaluation requests
type EvaluateResponse struct {
	Success bool                       `json:"success"`
	Data    []CriteriaEvaluationResult `json:"data,omitempty"`
	Error   string                     `json:"error,omitempty"`
}
