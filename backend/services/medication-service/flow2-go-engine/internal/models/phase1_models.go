package models

import (
	"time"
)

// Phase 1 Data Structures - Exact compliance with GO Orchestrator specification

// MedicationRequest represents an incoming medication request (Phase 1 input)
type MedicationRequest struct {
	RequestID      string                 `json:"request_id"`
	PatientID      string                 `json:"patient_id"`
	EncounterID    string                 `json:"encounter_id"`
	
	// Clinical context
	Indication       string                 `json:"indication"`        // e.g., "hypertension_stage2_ckd"
	ClinicalContext  ClinicalContextInput   `json:"clinical_context"`
	Urgency          UrgencyLevel           `json:"urgency"`           // ROUTINE, URGENT, STAT
	
	// Provider context
	Provider         ProviderContext        `json:"provider"`
	CareSettings     CareSettings           `json:"care_settings"`
	
	// Special considerations
	Preferences      PatientPreferences     `json:"preferences,omitempty"`
	Constraints      []ClinicalConstraint   `json:"constraints,omitempty"`
}

// IntentManifest represents the output from Phase 1 (ORB + Recipe Resolution)
type IntentManifest struct {
	ManifestID       string                 `json:"manifest_id"`
	RequestID        string                 `json:"request_id"`
	GeneratedAt      time.Time              `json:"generated_at"`
	
	// Classification results
	PrimaryIntent    ClinicalIntent         `json:"primary_intent"`
	SecondaryIntents []ClinicalIntent       `json:"secondary_intents,omitempty"`
	
	// Protocol selection
	ProtocolID       string                 `json:"protocol_id"`
	ProtocolVersion  string                 `json:"protocol_version"`
	EvidenceGrade    string                 `json:"evidence_grade"`
	
	// Recipe references
	ContextRecipeID  string                 `json:"context_recipe_id"`
	ClinicalRecipeID string                 `json:"clinical_recipe_id"`
	
	// Computed requirements
	RequiredFields   []FieldRequirement     `json:"required_fields"`
	OptionalFields   []FieldRequirement     `json:"optional_fields"`
	
	// Freshness requirements
	DataFreshness    FreshnessRequirements  `json:"data_freshness"`
	SnapshotTTL      int                    `json:"snapshot_ttl_seconds"`
	
	// Therapy options determined by ORB
	TherapyOptions   []TherapyCandidate     `json:"therapy_options"`
	
	// Provenance
	ORBVersion       string                 `json:"orb_version"`
	RulesApplied     []AppliedRule          `json:"rules_applied"`
}

// ClinicalIntent represents the classified clinical intent
type ClinicalIntent struct {
	Category         string                 `json:"category"`     // TREATMENT, PROPHYLAXIS, SYMPTOM_CONTROL
	Condition        string                 `json:"condition"`    // Coded condition (SNOMED/ICD)
	Severity         string                 `json:"severity"`     // MILD, MODERATE, SEVERE, CRITICAL
	Phenotype        string                 `json:"phenotype"`    // Patient phenotype classification
	TimeHorizon      string                 `json:"time_horizon"` // ACUTE, CHRONIC, MAINTENANCE
}

// TherapyCandidate represents a therapy option from ORB evaluation
type TherapyCandidate struct {
	TherapyClass     string                 `json:"therapy_class"`    // ACE_INHIBITOR, ARB, etc.
	PreferenceOrder  int                    `json:"preference_order"`
	Rationale        string                 `json:"rationale"`
	GuidelineSource  string                 `json:"guideline_source"`
}

// FieldRequirement represents a required or optional data field
type FieldRequirement struct {
	FieldName        string                 `json:"field_name"`
	FieldType        string                 `json:"field_type"`       // LAB, VITAL, MEDICATION, etc.
	Required         bool                   `json:"required"`
	MaxAgeHours      int                    `json:"max_age_hours"`
	Source           string                 `json:"source"`           // EHR, DEVICE, MANUAL, etc.
	ClinicalReason   string                 `json:"clinical_reason"`
}

// FreshnessRequirements defines data freshness requirements
type FreshnessRequirements struct {
	MaxAge           time.Duration          `json:"max_age"`
	CriticalFields   []string               `json:"critical_fields"`
	PreferredSources []string               `json:"preferred_sources"`
}

// AppliedRule represents a rule that was applied in ORB evaluation
type AppliedRule struct {
	RuleID           string                 `json:"rule_id"`
	RuleName         string                 `json:"rule_name"`
	Confidence       float64                `json:"confidence"`
	AppliedAt        time.Time              `json:"applied_at"`
	EvidenceLevel    string                 `json:"evidence_level"`
}

// Clinical Context Input Types

// ClinicalContextInput represents the clinical context provided in the request
type ClinicalContextInput struct {
	// Patient demographics
	Age              float64                `json:"age,omitempty"`
	Sex              string                 `json:"sex,omitempty"`
	Weight           float64                `json:"weight,omitempty"`
	
	// Clinical conditions
	Comorbidities    []string               `json:"comorbidities,omitempty"`
	ActiveProblems   []string               `json:"active_problems,omitempty"`
	
	// Current medications
	CurrentMeds      []CurrentMedication    `json:"current_medications,omitempty"`
	Allergies        []AllergyInfo          `json:"allergies,omitempty"`
	
	// Recent labs/vitals
	RecentLabs       map[string]LabValue    `json:"recent_labs,omitempty"`
	VitalSigns       map[string]VitalSign   `json:"vital_signs,omitempty"`
	
	// Clinical notes
	ProviderNotes    string                 `json:"provider_notes,omitempty"`
}

// Supporting Types

// UrgencyLevel represents the urgency of the medication request
type UrgencyLevel string

const (
	UrgencyRoutine UrgencyLevel = "ROUTINE"
	UrgencyUrgent  UrgencyLevel = "URGENT"
	UrgencyStat    UrgencyLevel = "STAT"
)

// ProviderContext represents the provider making the request
type ProviderContext struct {
	ProviderID       string                 `json:"provider_id"`
	ProviderName     string                 `json:"provider_name"`
	Specialty        string                 `json:"specialty"`
	Institution      string                 `json:"institution"`
}

// CareSettings represents the care settings
type CareSettings struct {
	Setting          string                 `json:"setting"`          // INPATIENT, OUTPATIENT, ED, etc.
	Unit             string                 `json:"unit,omitempty"`   // ICU, WARD, CLINIC, etc.
	AcuityLevel      string                 `json:"acuity_level,omitempty"`
}

// PatientPreferences represents patient preferences and constraints
type PatientPreferences struct {
	RoutePreferences []string               `json:"route_preferences,omitempty"`
	FrequencyPref    string                 `json:"frequency_preference,omitempty"`
	CostConstraints  string                 `json:"cost_constraints,omitempty"`
	Lifestyle        map[string]interface{} `json:"lifestyle,omitempty"`
}

// ClinicalConstraint represents clinical constraints
type ClinicalConstraint struct {
	Type             string                 `json:"type"`             // FORMULARY, COST, INTERACTION, etc.
	Value            string                 `json:"value"`
	Severity         string                 `json:"severity"`
	Source           string                 `json:"source"`
}

// CurrentMedication represents a current medication
type CurrentMedication struct {
	MedicationCode   string                 `json:"medication_code"`
	MedicationName   string                 `json:"medication_name"`
	Dose             string                 `json:"dose"`
	Frequency        string                 `json:"frequency"`
	Route            string                 `json:"route"`
	StartDate        time.Time              `json:"start_date"`
	Indication       string                 `json:"indication,omitempty"`
}

// AllergyInfo represents allergy information
type AllergyInfo struct {
	Allergen         string                 `json:"allergen"`
	AllergenType     string                 `json:"allergen_type"`    // DRUG, FOOD, ENVIRONMENTAL
	Reaction         string                 `json:"reaction"`
	Severity         string                 `json:"severity"`         // MILD, MODERATE, SEVERE
	VerifiedBy       string                 `json:"verified_by,omitempty"`
}

// LabValue represents a laboratory value
type LabValue struct {
	Value            float64                `json:"value"`
	Unit             string                 `json:"unit"`
	ReferenceRange   string                 `json:"reference_range,omitempty"`
	Timestamp        time.Time              `json:"timestamp"`
	Status           string                 `json:"status"`           // NORMAL, HIGH, LOW, CRITICAL
}

// VitalSign represents a vital sign measurement
type VitalSign struct {
	Value            float64                `json:"value"`
	Unit             string                 `json:"unit"`
	Timestamp        time.Time              `json:"timestamp"`
	Method           string                 `json:"method,omitempty"`
}

// Recipe Resolution Types (Phase 1 Internal)

// ContextRecipe represents a context recipe for data requirements
type ContextRecipe struct {
	ID               string                 `json:"id"`
	ProtocolID       string                 `json:"protocol_id"`
	Version          string                 `json:"version"`
	
	// Core required fields for all patients
	CoreFields       []FieldSpec            `json:"core_fields"`
	
	// Conditional fields based on patient characteristics
	ConditionalRules []ConditionalFieldRule `json:"conditional_rules"`
	
	// Freshness requirements per field category
	FreshnessRules   map[string]FreshnessRule `json:"freshness_rules"`
}

// ClinicalRecipe represents a clinical recipe for therapy protocols
type ClinicalRecipe struct {
	ID                string                 `json:"id"`
	ProtocolID        string                 `json:"protocol_id"`
	Version           string                 `json:"version"`
	
	// Therapy selection rules
	TherapySelectionRules []TherapyRule      `json:"therapy_selection_rules"`
	
	// Dosing strategy
	DosingStrategy    DosingStrategy         `json:"dosing_strategy"`
	
	// Safety requirements
	SafetyChecks      []SafetyCheckRequirement `json:"safety_checks"`
	
	// Monitoring requirements
	MonitoringPlan    MonitoringRequirements `json:"monitoring_plan"`
}

// FieldSpec represents a field specification
type FieldSpec struct {
	Name             string                 `json:"name"`
	Type             string                 `json:"type"`
	Required         bool                   `json:"required"`
	MaxAgeHours      int                    `json:"max_age_hours"`
	ClinicalContext  string                 `json:"clinical_context"`
}

// ConditionalFieldRule represents conditional field requirements
type ConditionalFieldRule struct {
	Condition        string                 `json:"condition"`        // Expression to evaluate
	RequiredFields   []FieldSpec            `json:"required_fields"`
	Rationale        string                 `json:"rationale"`
}

// FreshnessRule represents data freshness rules
type FreshnessRule struct {
	MaxAge           time.Duration          `json:"max_age"`
	CriticalThreshold time.Duration         `json:"critical_threshold"`
	PreferredSources []string               `json:"preferred_sources"`
}

// TherapyRule represents therapy selection rules
type TherapyRule struct {
	Priority         int                    `json:"priority"`
	DrugClass        string                 `json:"drug_class"`
	Conditions       []string               `json:"conditions"`
	Contraindications []string              `json:"contraindications"`
	EvidenceLevel    string                 `json:"evidence_level"`
}

// DosingStrategy represents dosing strategy
type DosingStrategy struct {
	Approach         string                 `json:"approach"`         // STANDARD, INDIVIDUALIZED, etc.
	AdjustmentFactors []string              `json:"adjustment_factors"`
	StartingDose     *DoseRecommendation    `json:"starting_dose,omitempty"`
	TitrationPlan    *TitrationPlan         `json:"titration_plan,omitempty"`
}

// SafetyCheckRequirement represents safety check requirements
type SafetyCheckRequirement struct {
	CheckType        string                 `json:"check_type"`       // INTERACTION, ALLERGY, RENAL, etc.
	Severity         string                 `json:"severity"`
	Mandatory        bool                   `json:"mandatory"`
	Parameters       map[string]interface{} `json:"parameters,omitempty"`
}

// MonitoringRequirements represents monitoring requirements
type MonitoringRequirements struct {
	Required         []MonitoringParameter  `json:"required"`
	Optional         []MonitoringParameter  `json:"optional"`
	Duration         string                 `json:"duration"`
	EscalationPlan   string                 `json:"escalation_plan,omitempty"`
}

// MonitoringParameter represents a monitoring parameter
type MonitoringParameter struct {
	Parameter        string                 `json:"parameter"`
	Frequency        string                 `json:"frequency"`
	Duration         string                 `json:"duration"`
	ThresholdAlerts  []ThresholdAlert       `json:"threshold_alerts,omitempty"`
}

// DoseRecommendation represents a dose recommendation
type DoseRecommendation struct {
	Amount           float64                `json:"amount"`
	Unit             string                 `json:"unit"`
	Frequency        string                 `json:"frequency"`
	Route            string                 `json:"route"`
	Rationale        string                 `json:"rationale"`
}

// TitrationPlan represents a titration plan
type TitrationPlan struct {
	InitialDose      DoseRecommendation     `json:"initial_dose"`
	TitrationSteps   []TitrationStep        `json:"titration_steps"`
	MaxDose          DoseRecommendation     `json:"max_dose"`
	MonitoringPoints []string               `json:"monitoring_points"`
}

// TitrationStep represents a titration step
type TitrationStep struct {
	StepNumber       int                    `json:"step_number"`
	Dose             DoseRecommendation     `json:"dose"`
	Criteria         string                 `json:"criteria"`
	MinDuration      string                 `json:"min_duration"`
}

// ThresholdAlert represents a threshold alert
type ThresholdAlert struct {
	Parameter        string                 `json:"parameter"`
	Threshold        float64                `json:"threshold"`
	Direction        string                 `json:"direction"`       // ABOVE, BELOW
	Severity         string                 `json:"severity"`
	Action           string                 `json:"action"`
}