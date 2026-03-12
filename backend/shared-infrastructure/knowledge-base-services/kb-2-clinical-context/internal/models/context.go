package models

import (
	"time"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PhenotypeDefinition represents a clinical phenotype definition (from context.go)
type PhenotypeDefinition struct {
	ID          primitive.ObjectID    `json:"id" bson:"_id,omitempty"`
	PhenotypeID string               `json:"phenotype_id" bson:"phenotype_id"`
	Name        string               `json:"name" bson:"name"`
	Version     string               `json:"version" bson:"version"`
	Description string               `json:"description" bson:"description"`
	Criteria    PhenotypeCriteria    `json:"criteria" bson:"criteria"`
	Clinical    ClinicalSignificance `json:"clinical_significance" bson:"clinical_significance"`
	Status      string               `json:"status" bson:"status"`
	CreatedAt   time.Time            `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at" bson:"updated_at"`
}

// PhenotypeCriteria defines the criteria for detecting a phenotype (from context.go)
type PhenotypeCriteria struct {
	RequiredConditions []ConditionCriteria `json:"required_conditions" bson:"required_conditions"`
	RequiredLabs       []LabCriteria       `json:"required_labs" bson:"required_labs"`
	RequiredMeds       []MedicationCriteria `json:"required_medications" bson:"required_medications"`
	ExclusionCriteria  []string            `json:"exclusion_criteria" bson:"exclusion_criteria"`
}

// ConditionCriteria represents condition-based criteria
type ConditionCriteria struct {
	Type           string `json:"type" bson:"type"`
	Codes          []string `json:"codes" bson:"codes"`
	TimeWindow     string `json:"time_window" bson:"time_window"`
	MinOccurrences int    `json:"min_occurrences" bson:"min_occurrences"`
}

// LabCriteria represents laboratory-based criteria
type LabCriteria struct {
	LOINCCode  string  `json:"loinc_code" bson:"loinc_code"`
	Operator   string  `json:"operator" bson:"operator"`
	Value      float64 `json:"value" bson:"value"`
	Unit       string  `json:"unit" bson:"unit"`
	TimeWindow string  `json:"time_window" bson:"time_window"`
}

// MedicationCriteria represents medication-based criteria
type MedicationCriteria struct {
	RxNormCodes  []string `json:"rxnorm_codes" bson:"rxnorm_codes"`
	DurationDays int      `json:"duration_days" bson:"duration_days"`
}

// ClinicalSignificance defines the clinical implications
type ClinicalSignificance struct {
	RiskImplications        []string `json:"risk_implications" bson:"risk_implications"`
	TreatmentModifications  []string `json:"treatment_modifications" bson:"treatment_modifications"`
	MonitoringRequirements  []string `json:"monitoring_requirements" bson:"monitoring_requirements"`
}

// PatientContext represents the clinical context for a patient
type PatientContext struct {
	ID                primitive.ObjectID    `json:"id" bson:"_id,omitempty"`
	PatientID         string               `json:"patient_id" bson:"patient_id"`
	ContextID         string               `json:"context_id" bson:"context_id"`
	Timestamp         time.Time            `json:"timestamp" bson:"timestamp"`
	Demographics      Demographics         `json:"demographics" bson:"demographics"`
	ActiveConditions  []Condition          `json:"active_conditions" bson:"active_conditions"`
	RecentLabs        []LabResult          `json:"recent_labs" bson:"recent_labs"`
	CurrentMeds       []Medication         `json:"current_medications" bson:"current_medications"`
	DetectedPhenotypes []DetectedPhenotype  `json:"detected_phenotypes" bson:"detected_phenotypes"`
	RiskFactors       map[string]interface{} `json:"risk_factors" bson:"risk_factors"`
	CareGaps          []string             `json:"care_gaps" bson:"care_gaps"`
	TTL               time.Time            `json:"ttl" bson:"ttl"`
}

// Demographics represents patient demographic information (from context.go)
type Demographics struct {
	AgeYears  int    `json:"age_years" bson:"age_years"`
	Sex       string `json:"sex" bson:"sex"`
	Race      string `json:"race" bson:"race"`
	Ethnicity string `json:"ethnicity" bson:"ethnicity"`
}

// Condition represents a medical condition
type Condition struct {
	Code        string    `json:"code" bson:"code"`
	System      string    `json:"system" bson:"system"`
	Name        string    `json:"name" bson:"name"`
	OnsetDate   time.Time `json:"onset_date" bson:"onset_date"`
	Severity    string    `json:"severity" bson:"severity"`
}

// LabResult represents a laboratory result
type LabResult struct {
	LOINCCode    string    `json:"loinc_code" bson:"loinc_code"`
	Value        float64   `json:"value" bson:"value"`
	Unit         string    `json:"unit" bson:"unit"`
	ResultDate   time.Time `json:"result_date" bson:"result_date"`
	AbnormalFlag string    `json:"abnormal_flag" bson:"abnormal_flag"`
}

// Medication represents a current medication
type Medication struct {
	RxNormCode string    `json:"rxnorm_code" bson:"rxnorm_code"`
	Name       string    `json:"name" bson:"name"`
	Dose       string    `json:"dose" bson:"dose"`
	Frequency  string    `json:"frequency" bson:"frequency"`
	StartDate  time.Time `json:"start_date" bson:"start_date"`
}

// DetectedPhenotype represents a detected clinical phenotype
type DetectedPhenotype struct {
	PhenotypeID        string                 `json:"phenotype_id" bson:"phenotype_id"`
	Confidence         float64                `json:"confidence" bson:"confidence"`
	DetectedAt         time.Time              `json:"detected_at" bson:"detected_at"`
	SupportingEvidence []map[string]interface{} `json:"supporting_evidence" bson:"supporting_evidence"`
}

// Request/Response Models

// BuildContextRequest represents a request to build patient context
type BuildContextRequest struct {
	PatientID    string                 `json:"patient_id" binding:"required"`
	Patient      map[string]interface{} `json:"patient" binding:"required"`
	TransactionID string                `json:"transaction_id,omitempty"`
}

// BuildContextResponse represents the response with built context
type BuildContextResponse struct {
	Context     PatientContext         `json:"context"`
	Phenotypes  []string               `json:"phenotypes"`
	RiskScores  map[string]float64     `json:"risk_scores"`
	CacheHit    bool                   `json:"cache_hit"`
	ProcessedAt time.Time              `json:"processed_at"`
}

// PhenotypeDetectionRequest represents a request for phenotype detection
type PhenotypeDetectionRequest struct {
	PatientID    string                 `json:"patient_id" binding:"required"`
	PatientData  map[string]interface{} `json:"patient_data" binding:"required"`
	PhenotypeIDs []string               `json:"phenotype_ids,omitempty"`
}

// PhenotypeDetectionResponse represents phenotype detection results
type PhenotypeDetectionResponse struct {
	PatientID          string              `json:"patient_id"`
	DetectedPhenotypes []DetectedPhenotype `json:"detected_phenotypes"`
	TotalPhenotypes    int                 `json:"total_phenotypes"`
	ProcessingTime     int64               `json:"processing_time_ms"`
	Timestamp          time.Time           `json:"timestamp"`
}

// RiskAssessmentRequest represents a risk assessment request
type RiskAssessmentRequest struct {
	PatientID   string                 `json:"patient_id" binding:"required"`
	RiskTypes   []string               `json:"risk_types,omitempty"`
	PatientData map[string]interface{} `json:"patient_data,omitempty"`
}

// RiskAssessmentResponse represents risk assessment results
type RiskAssessmentResponse struct {
	PatientID           string             `json:"patient_id"`
	RiskScores          map[string]float64 `json:"risk_scores"`
	RiskFactors         map[string]interface{} `json:"risk_factors"`
	Recommendations     []string           `json:"recommendations"`
	ConfidenceScore     float64            `json:"confidence_score"`
	AssessmentTimestamp time.Time          `json:"assessment_timestamp"`
}

// CareGapsRequest represents a care gaps identification request
type CareGapsRequest struct {
	PatientID        string `json:"patient_id" binding:"required"`
	IncludeResolved  bool   `json:"include_resolved,omitempty"`
	TimeframeDays    int    `json:"timeframe_days,omitempty"`
}

// CareGapsResponse represents care gaps identification results
type CareGapsResponse struct {
	PatientID     string    `json:"patient_id"`
	CareGaps      []CareGap `json:"care_gaps"`
	TotalGaps     int       `json:"total_gaps"`
	Priority      string    `json:"priority"`
	NextReview    time.Time `json:"next_review"`
}

// CareGap represents an identified care gap
type CareGap struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	DueDays     int       `json:"due_days"`
	Actions     []string  `json:"actions"`
}

// Additional models needed for GraphQL federation

// Patient represents a federated patient entity for Apollo Federation
type Patient struct {
	ID string `json:"id"`
}

// SystemHealth represents system health status
type SystemHealth struct {
	Status    string      `json:"status"`
	Timestamp time.Time   `json:"timestamp"`
	Checks    interface{} `json:"checks"`
}

// ContextStatistics represents context processing statistics
type ContextStatistics struct {
	TotalContexts           int     `json:"total_contexts"`
	CacheHitRate           float64 `json:"cache_hit_rate"`
	AverageProcessingTime  float64 `json:"average_processing_time"`
	PhenotypeDetectionRate float64 `json:"phenotype_detection_rate"`
}