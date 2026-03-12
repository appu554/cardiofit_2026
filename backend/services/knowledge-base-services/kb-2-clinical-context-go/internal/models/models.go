package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Patient represents patient data for context evaluation
type Patient struct {
	ID         string                 `json:"id" bson:"_id,omitempty"`
	Age        int                   `json:"age" bson:"age"`
	Gender     string                `json:"gender" bson:"gender"`
	Conditions []string              `json:"conditions" bson:"conditions"`
	Medications []string             `json:"medications" bson:"medications"`
	Labs       map[string]LabValue   `json:"labs" bson:"labs"`
	Vitals     map[string]VitalValue `json:"vitals" bson:"vitals"`
	Allergies  []string              `json:"allergies" bson:"allergies"`
	Metadata   map[string]interface{} `json:"metadata" bson:"metadata"`
}

type LabValue struct {
	Value     float64   `json:"value" bson:"value"`
	Unit      string    `json:"unit" bson:"unit"`
	Date      time.Time `json:"date" bson:"date"`
	Reference Range     `json:"reference" bson:"reference"`
}

type VitalValue struct {
	Value float64   `json:"value" bson:"value"`
	Unit  string    `json:"unit" bson:"unit"`
	Date  time.Time `json:"date" bson:"date"`
}

type Range struct {
	Min float64 `json:"min" bson:"min"`
	Max float64 `json:"max" bson:"max"`
}

// PhenotypeDefinition represents a clinical phenotype with CEL evaluation
type PhenotypeDefinition struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string            `json:"name" bson:"name"`
	Description string            `json:"description" bson:"description"`
	Category    string            `json:"category" bson:"category"`
	CELRule     string            `json:"cel_rule" bson:"cel_rule"`
	Priority    int               `json:"priority" bson:"priority"`
	Metadata    map[string]interface{} `json:"metadata" bson:"metadata"`
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" bson:"updated_at"`
	Version     string            `json:"version" bson:"version"`
}

// PhenotypeEvaluationRequest for batch phenotype evaluation
type PhenotypeEvaluationRequest struct {
	Patients      []Patient `json:"patients"`
	PhenotypeIDs  []string  `json:"phenotype_ids,omitempty"`
	IncludeExplanation bool `json:"include_explanation,omitempty"`
}

// PhenotypeEvaluationResult represents evaluation results
type PhenotypeEvaluationResult struct {
	PatientID     string                      `json:"patient_id"`
	Phenotypes    []DetectedPhenotype         `json:"phenotypes"`
	Explanation   *PhenotypeExplanation       `json:"explanation,omitempty"`
	ProcessingTime time.Duration              `json:"processing_time"`
	Errors        []string                    `json:"errors,omitempty"`
}

type DetectedPhenotype struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	Detected    bool              `json:"detected"`
	Confidence  float64           `json:"confidence"`
	Evidence    []EvidenceItem    `json:"evidence"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type EvidenceItem struct {
	Type        string      `json:"type"`
	Value       interface{} `json:"value"`
	Description string      `json:"description"`
	Weight      float64     `json:"weight"`
}

// PhenotypeExplanation provides reasoning chains
type PhenotypeExplanation struct {
	PatientID       string            `json:"patient_id"`
	ReasoningChains []ReasoningChain  `json:"reasoning_chains"`
	GeneratedAt     time.Time         `json:"generated_at"`
}

type ReasoningChain struct {
	PhenotypeID  string         `json:"phenotype_id"`
	PhenotypeName string        `json:"phenotype_name"`
	Steps        []ReasoningStep `json:"steps"`
	Conclusion   string         `json:"conclusion"`
}

type ReasoningStep struct {
	Rule        string      `json:"rule"`
	Evaluation  string      `json:"evaluation"`
	Result      interface{} `json:"result"`
	Explanation string      `json:"explanation"`
}

// RiskAssessmentRequest for enhanced risk calculation
type RiskAssessmentRequest struct {
	PatientID        string                 `json:"patient_id"`
	PatientData      Patient                `json:"patient_data"`
	RiskCategories   []string               `json:"risk_categories,omitempty"`
	TimeHorizon      string                 `json:"time_horizon,omitempty"`
	IncludeFactors   bool                   `json:"include_factors,omitempty"`
	CustomParameters map[string]interface{} `json:"custom_parameters,omitempty"`
}

type RiskAssessmentResult struct {
	PatientID      string             `json:"patient_id"`
	OverallRisk    RiskScore          `json:"overall_risk"`
	CategoryRisks  map[string]RiskScore `json:"category_risks"`
	RiskFactors    []RiskFactor       `json:"risk_factors"`
	Recommendations []Recommendation   `json:"recommendations"`
	ProcessingTime time.Duration      `json:"processing_time"`
	GeneratedAt    time.Time          `json:"generated_at"`
}

type RiskScore struct {
	Score       float64 `json:"score"`
	Level       string  `json:"level"`
	Percentile  float64 `json:"percentile"`
	Confidence  float64 `json:"confidence"`
	Description string  `json:"description"`
}

type RiskFactor struct {
	Factor      string  `json:"factor"`
	Impact      float64 `json:"impact"`
	Evidence    string  `json:"evidence"`
	Modifiable  bool    `json:"modifiable"`
}

type Recommendation struct {
	Category    string `json:"category"`
	Action      string `json:"action"`
	Priority    string `json:"priority"`
	Evidence    string `json:"evidence"`
	ExpectedBenefit float64 `json:"expected_benefit"`
}

// TreatmentPreferencesRequest for treatment recommendations
type TreatmentPreferencesRequest struct {
	PatientID           string                 `json:"patient_id"`
	PatientData         Patient                `json:"patient_data"`
	Condition           string                 `json:"condition"`
	TreatmentCategories []string               `json:"treatment_categories,omitempty"`
	InstitutionalRules  []string               `json:"institutional_rules,omitempty"`
	PreferenceProfile   map[string]interface{} `json:"preference_profile,omitempty"`
}

type TreatmentPreferencesResult struct {
	PatientID          string                    `json:"patient_id"`
	Condition          string                    `json:"condition"`
	TreatmentOptions   []TreatmentOption         `json:"treatment_options"`
	PreferredTreatments []PreferredTreatment     `json:"preferred_treatments"`
	ConflictResolution []ConflictResolution      `json:"conflict_resolution,omitempty"`
	ProcessingTime     time.Duration             `json:"processing_time"`
	GeneratedAt        time.Time                 `json:"generated_at"`
}

type TreatmentOption struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Category        string            `json:"category"`
	Suitability     float64           `json:"suitability"`
	Contraindications []string        `json:"contraindications,omitempty"`
	Preferences     []PreferenceMatch `json:"preferences"`
	Evidence        EvidenceLevel     `json:"evidence"`
	Cost            CostProfile       `json:"cost,omitempty"`
}

type PreferredTreatment struct {
	TreatmentID     string          `json:"treatment_id"`
	Rank            int             `json:"rank"`
	OverallScore    float64         `json:"overall_score"`
	Rationale       string          `json:"rationale"`
	AlternativeRanks map[string]int `json:"alternative_ranks,omitempty"`
}

type ConflictResolution struct {
	ConflictType string            `json:"conflict_type"`
	Resolution   string            `json:"resolution"`
	Priority     string            `json:"priority"`
	Rationale    string            `json:"rationale"`
	AppliedRules []string          `json:"applied_rules"`
}

type PreferenceMatch struct {
	Preference string  `json:"preference"`
	Match      bool    `json:"match"`
	Weight     float64 `json:"weight"`
}

type EvidenceLevel struct {
	Grade       string  `json:"grade"`
	Level       int     `json:"level"`
	Description string  `json:"description"`
	References  []string `json:"references,omitempty"`
}

type CostProfile struct {
	Category      string  `json:"category"`
	EstimatedCost float64 `json:"estimated_cost"`
	Currency      string  `json:"currency"`
}

// ContextAssemblyRequest for complete context assembly
type ContextAssemblyRequest struct {
	PatientID          string                 `json:"patient_id"`
	PatientData        Patient                `json:"patient_data"`
	ContextComponents  []string               `json:"context_components"`
	IncludePhenotypes  bool                   `json:"include_phenotypes,omitempty"`
	IncludeRisks       bool                   `json:"include_risks,omitempty"`
	IncludeTreatments  bool                   `json:"include_treatments,omitempty"`
	DetailLevel        string                 `json:"detail_level,omitempty"` // basic, standard, comprehensive
	CustomParameters   map[string]interface{} `json:"custom_parameters,omitempty"`
}

type ClinicalContext struct {
	PatientID           string                      `json:"patient_id"`
	PhenotypeResults    []PhenotypeEvaluationResult `json:"phenotype_results,omitempty"`
	RiskAssessment      *RiskAssessmentResult       `json:"risk_assessment,omitempty"`
	TreatmentPreferences *TreatmentPreferencesResult `json:"treatment_preferences,omitempty"`
	ContextSummary      ContextSummary              `json:"context_summary"`
	ProcessingMetrics   ProcessingMetrics           `json:"processing_metrics"`
	GeneratedAt         time.Time                   `json:"generated_at"`
}

type ContextSummary struct {
	KeyFindings        []string          `json:"key_findings"`
	RiskHighlights     []string          `json:"risk_highlights"`
	TreatmentSummary   []string          `json:"treatment_summary"`
	ClinicalAlerts     []ClinicalAlert   `json:"clinical_alerts"`
	Recommendations    []string          `json:"recommendations"`
}

type ClinicalAlert struct {
	Level       string `json:"level"`
	Type        string `json:"type"`
	Message     string `json:"message"`
	Source      string `json:"source"`
	Action      string `json:"action,omitempty"`
}

type ProcessingMetrics struct {
	TotalProcessingTime    time.Duration `json:"total_processing_time"`
	ComponentProcessingTimes map[string]time.Duration `json:"component_processing_times"`
	CacheHits              int           `json:"cache_hits"`
	CacheMisses            int           `json:"cache_misses"`
	RulesEvaluated         int           `json:"rules_evaluated"`
	PhenotypesEvaluated    int           `json:"phenotypes_evaluated"`
}

// Error types
type APIError struct {
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Status    int                    `json:"status"`
	Detail    string                 `json:"detail"`
	Instance  string                 `json:"instance,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func (e *APIError) Error() string {
	return e.Detail
}

// Health check model
type HealthStatus struct {
	Status      string            `json:"status"`
	Timestamp   time.Time         `json:"timestamp"`
	Version     string            `json:"version"`
	Environment string            `json:"environment"`
	Checks      map[string]Check  `json:"checks"`
}

type Check struct {
	Status      string        `json:"status"`
	ResponseTime time.Duration `json:"response_time,omitempty"`
	Message     string        `json:"message,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}