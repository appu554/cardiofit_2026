package models

import "time"

// RustIntelligenceRequest represents a request to the Rust intelligence engine
type RustIntelligenceRequest struct {
	RequestID        string                 `json:"request_id"`
	PatientID        string                 `json:"patient_id"`
	Medications      []Medication           `json:"medications"`
	IntelligenceType string                 `json:"intelligence_type"`
	AnalysisDepth    string                 `json:"analysis_depth"`
	ClinicalContext  *ClinicalContext       `json:"clinical_context"`
	ProcessingHints  map[string]interface{} `json:"processing_hints"`
}

// RustDoseOptimizationRequest represents a dose optimization request to Rust engine
type RustDoseOptimizationRequest struct {
	RequestID          string                 `json:"request_id"`
	PatientID          string                 `json:"patient_id"`
	MedicationCode     string                 `json:"medication_code"`
	ClinicalParameters map[string]interface{} `json:"clinical_parameters"`
	OptimizationType   string                 `json:"optimization_type"`
	ClinicalContext    *ClinicalContext       `json:"clinical_context"`
	ProcessingHints    map[string]interface{} `json:"processing_hints"`
}

// RustSafetyValidationRequest represents a safety validation request to Rust engine
type RustSafetyValidationRequest struct {
	RequestID       string                 `json:"request_id"`
	PatientID       string                 `json:"patient_id"`
	Medications     []Medication           `json:"medications"`
	ClinicalContext *ClinicalContext       `json:"clinical_context"`
	ValidationLevel string                 `json:"validation_level"`
	ProcessingHints map[string]interface{} `json:"processing_hints"`
}

// ClinicalContext represents comprehensive clinical context for a patient
type ClinicalContext struct {
	PatientDemographics *PatientDemographics `json:"patient_demographics,omitempty"`
	CurrentMedications  []Medication         `json:"current_medications,omitempty"`
	Allergies           []Allergy            `json:"allergies,omitempty"`
	Conditions          []Condition          `json:"conditions,omitempty"`
	LabResults          []LabResult          `json:"lab_results,omitempty"`
	VitalSigns          []VitalSign          `json:"vital_signs,omitempty"`
	Encounters          []Encounter          `json:"encounters,omitempty"`
	SocialHistory       *SocialHistory       `json:"social_history,omitempty"`
	FamilyHistory       []FamilyHistory      `json:"family_history,omitempty"`
	Procedures          []Procedure          `json:"procedures,omitempty"`
	Observations        []Observation        `json:"observations,omitempty"`
	ClinicalNotes       []ClinicalNote       `json:"clinical_notes,omitempty"`
	RiskFactors         []RiskFactor         `json:"risk_factors,omitempty"`
	Preferences         *PatientPreferences  `json:"preferences,omitempty"`
	Insurance           *InsuranceInfo       `json:"insurance,omitempty"`
	Formulary           *FormularyInfo       `json:"formulary,omitempty"`
}

// PatientDemographics represents patient demographic information
type PatientDemographics struct {
	Age       *float64 `json:"age_years,omitempty"`
	Weight    *float64 `json:"weight_kg,omitempty"`
	Height    *float64 `json:"height_cm,omitempty"`
	BMI       *float64 `json:"bmi,omitempty"`
	BSA       *float64 `json:"bsa_m2,omitempty"`
	Gender    string   `json:"gender,omitempty"`
	Race      string   `json:"race,omitempty"`
	Ethnicity string   `json:"ethnicity,omitempty"`
}

// Allergy represents a patient allergy
type Allergy struct {
	Allergen    string    `json:"allergen"`
	AllergenType string   `json:"allergen_type"` // "DRUG", "FOOD", "ENVIRONMENTAL"
	Reaction    string    `json:"reaction"`
	Severity    string    `json:"severity"` // "MILD", "MODERATE", "SEVERE", "LIFE_THREATENING"
	OnsetDate   *time.Time `json:"onset_date,omitempty"`
	Status      string    `json:"status"` // "ACTIVE", "INACTIVE", "RESOLVED"
	Confidence  float64   `json:"confidence"`
}

// Condition represents a patient condition/diagnosis
type Condition struct {
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	Category     string     `json:"category"`
	Severity     string     `json:"severity"`
	Status       string     `json:"status"` // "ACTIVE", "INACTIVE", "RESOLVED"
	OnsetDate    *time.Time `json:"onset_date,omitempty"`
	ResolvedDate *time.Time `json:"resolved_date,omitempty"`
	IsPrimary    bool       `json:"is_primary"`
}

// LabResult represents a laboratory test result
type LabResult struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	ReferenceRange string `json:"reference_range"`
	Status      string    `json:"status"` // "NORMAL", "HIGH", "LOW", "CRITICAL"
	Timestamp   time.Time `json:"timestamp"`
	OrderedBy   string    `json:"ordered_by,omitempty"`
}

// VitalSign represents a vital sign measurement
type VitalSign struct {
	Type      string    `json:"type"` // "BLOOD_PRESSURE", "HEART_RATE", "TEMPERATURE", etc.
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
	Method    string    `json:"method,omitempty"`
}

// Encounter represents a healthcare encounter
type Encounter struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "INPATIENT", "OUTPATIENT", "EMERGENCY"
	Status      string    `json:"status"`
	StartDate   time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	Provider    string    `json:"provider,omitempty"`
	Department  string    `json:"department,omitempty"`
	Diagnosis   []string  `json:"diagnosis,omitempty"`
}

// SocialHistory represents patient social history
type SocialHistory struct {
	SmokingStatus   string  `json:"smoking_status,omitempty"`
	AlcoholUse      string  `json:"alcohol_use,omitempty"`
	DrugUse         string  `json:"drug_use,omitempty"`
	Occupation      string  `json:"occupation,omitempty"`
	MaritalStatus   string  `json:"marital_status,omitempty"`
	LivingSituation string  `json:"living_situation,omitempty"`
	ExerciseLevel   string  `json:"exercise_level,omitempty"`
	DietType        string  `json:"diet_type,omitempty"`
}

// FamilyHistory represents family medical history
type FamilyHistory struct {
	Relationship string `json:"relationship"`
	Condition    string `json:"condition"`
	AgeAtOnset   *int   `json:"age_at_onset,omitempty"`
	Status       string `json:"status"` // "ACTIVE", "DECEASED"
}

// Procedure represents a medical procedure
type Procedure struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Date        time.Time `json:"date"`
	Provider    string    `json:"provider,omitempty"`
	Outcome     string    `json:"outcome,omitempty"`
	Complications []string `json:"complications,omitempty"`
}

// Observation represents a clinical observation
type Observation struct {
	Code      string                 `json:"code"`
	Name      string                 `json:"name"`
	Value     interface{}            `json:"value"`
	Unit      string                 `json:"unit,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Method    string                 `json:"method,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ClinicalNote represents a clinical note
type ClinicalNote struct {
	Type      string    `json:"type"` // "PROGRESS", "DISCHARGE", "CONSULTATION"
	Content   string    `json:"content"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
	Keywords  []string  `json:"keywords,omitempty"`
}

// RiskFactor represents a clinical risk factor
type RiskFactor struct {
	Factor      string  `json:"factor"`
	Category    string  `json:"category"`
	Severity    string  `json:"severity"`
	Probability float64 `json:"probability"`
	Impact      string  `json:"impact"`
	Modifiable  bool    `json:"modifiable"`
}

// PatientPreferences represents patient preferences
type PatientPreferences struct {
	PreferredLanguage    string   `json:"preferred_language,omitempty"`
	CommunicationMethod  string   `json:"communication_method,omitempty"`
	MedicationPreferences []string `json:"medication_preferences,omitempty"`
	RoutePreferences     []string `json:"route_preferences,omitempty"`
	FrequencyPreferences []string `json:"frequency_preferences,omitempty"`
	CostSensitivity      string   `json:"cost_sensitivity,omitempty"`
	AdherenceHistory     string   `json:"adherence_history,omitempty"`
}

// InsuranceInfo represents insurance information
type InsuranceInfo struct {
	PlanName     string   `json:"plan_name,omitempty"`
	PlanType     string   `json:"plan_type,omitempty"`
	Copay        *float64 `json:"copay,omitempty"`
	Deductible   *float64 `json:"deductible,omitempty"`
	Coverage     string   `json:"coverage,omitempty"`
	Formulary    string   `json:"formulary,omitempty"`
	Restrictions []string `json:"restrictions,omitempty"`
}

// FormularyInfo represents formulary information
type FormularyInfo struct {
	Name         string                 `json:"name,omitempty"`
	Type         string                 `json:"type,omitempty"`
	Tier         string                 `json:"tier,omitempty"`
	Covered      bool                   `json:"covered"`
	Copay        *float64               `json:"copay,omitempty"`
	Restrictions []string               `json:"restrictions,omitempty"`
	Alternatives []AlternativeMedication `json:"alternatives,omitempty"`
}
