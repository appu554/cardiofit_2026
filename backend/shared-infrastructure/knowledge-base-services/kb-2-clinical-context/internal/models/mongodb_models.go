package models

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ClinicalContext represents a patient's clinical context
type ClinicalContext struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	PatientID         string             `bson:"patient_id" json:"patient_id"`
	ContextType       string             `bson:"context_type" json:"context_type"` // admission, visit, episode
	ContextID         string             `bson:"context_id" json:"context_id"`
	ClinicalIndicators ClinicalIndicators `bson:"clinical_indicators" json:"clinical_indicators"`
	Demographics      MongoDBDemographics       `bson:"demographics" json:"demographics"`
	RiskFactors       []RiskFactor       `bson:"risk_factors" json:"risk_factors"`
	Phenotypes        []PhenotypeMatch   `bson:"phenotypes" json:"phenotypes"`
	ContextualInsights []ContextualInsight `bson:"contextual_insights" json:"contextual_insights"`
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
	Version           int                `bson:"version" json:"version"`
}

// ClinicalIndicators holds various clinical measurements and indicators
type ClinicalIndicators struct {
	VitalSigns       VitalSigns         `bson:"vital_signs" json:"vital_signs"`
	LabValues        []LabValue         `bson:"lab_values" json:"lab_values"`
	Medications      []MedicationInfo   `bson:"medications" json:"medications"`
	ConditionCodes   []ConditionCode    `bson:"condition_codes" json:"condition_codes"`
	Procedures       []ProcedureCode    `bson:"procedures" json:"procedures"`
	Allergies        []AllergyInfo      `bson:"allergies" json:"allergies"`
	SocialHistory    SocialHistory      `bson:"social_history" json:"social_history"`
}

// MongoDBDemographics holds patient demographic information
type MongoDBDemographics struct {
	AgeRange      AgeRange `bson:"age_range" json:"age_range"`         // Instead of exact age for privacy
	Gender        string   `bson:"gender" json:"gender"`
	Ethnicity     string   `bson:"ethnicity,omitempty" json:"ethnicity,omitempty"`
	Race          string   `bson:"race,omitempty" json:"race,omitempty"`
	GeographicRegion string `bson:"geographic_region,omitempty" json:"geographic_region,omitempty"`
}

// AgeRange represents age brackets for privacy protection
type AgeRange struct {
	Min int    `bson:"min" json:"min"`
	Max int    `bson:"max" json:"max"`
	Category string `bson:"category" json:"category"` // pediatric, adult, geriatric
}

// VitalSigns holds patient vital signs
type VitalSigns struct {
	SystolicBP      *float64 `bson:"systolic_bp,omitempty" json:"systolic_bp,omitempty"`
	DiastolicBP     *float64 `bson:"diastolic_bp,omitempty" json:"diastolic_bp,omitempty"`
	HeartRate       *int     `bson:"heart_rate,omitempty" json:"heart_rate,omitempty"`
	Temperature     *float64 `bson:"temperature,omitempty" json:"temperature,omitempty"`
	RespiratoryRate *int     `bson:"respiratory_rate,omitempty" json:"respiratory_rate,omitempty"`
	OxygenSat       *float64 `bson:"oxygen_sat,omitempty" json:"oxygen_sat,omitempty"`
	BMI             *float64 `bson:"bmi,omitempty" json:"bmi,omitempty"`
	Weight          *float64 `bson:"weight,omitempty" json:"weight,omitempty"`
	Height          *float64 `bson:"height,omitempty" json:"height,omitempty"`
}

// LabValue represents laboratory test results
type LabValue struct {
	TestCode      string    `bson:"test_code" json:"test_code"`
	TestName      string    `bson:"test_name" json:"test_name"`
	Value         float64   `bson:"value" json:"value"`
	Unit          string    `bson:"unit" json:"unit"`
	ReferenceRange string   `bson:"reference_range,omitempty" json:"reference_range,omitempty"`
	Status        string    `bson:"status" json:"status"` // normal, abnormal, critical
	CollectedAt   time.Time `bson:"collected_at" json:"collected_at"`
	LOINCCode     string    `bson:"loinc_code,omitempty" json:"loinc_code,omitempty"`
}

// MedicationInfo represents current/recent medications
type MedicationInfo struct {
	MedicationID   string    `bson:"medication_id" json:"medication_id"`
	MedicationName string    `bson:"medication_name" json:"medication_name"`
	RxNormCode     string    `bson:"rxnorm_code,omitempty" json:"rxnorm_code,omitempty"`
	Dosage         string    `bson:"dosage,omitempty" json:"dosage,omitempty"`
	Frequency      string    `bson:"frequency,omitempty" json:"frequency,omitempty"`
	StartDate      time.Time `bson:"start_date" json:"start_date"`
	EndDate        *time.Time `bson:"end_date,omitempty" json:"end_date,omitempty"`
	Status         string    `bson:"status" json:"status"` // active, discontinued, held
	Indication     string    `bson:"indication,omitempty" json:"indication,omitempty"`
}

// ConditionCode represents patient conditions/diagnoses
type ConditionCode struct {
	Code            string    `bson:"code" json:"code"`
	CodeSystem      string    `bson:"code_system" json:"code_system"` // ICD10, SNOMED
	Description     string    `bson:"description" json:"description"`
	Severity        string    `bson:"severity,omitempty" json:"severity,omitempty"`
	Status          string    `bson:"status" json:"status"` // active, resolved, inactive
	OnsetDate       *time.Time `bson:"onset_date,omitempty" json:"onset_date,omitempty"`
	DiagnosedAt     time.Time `bson:"diagnosed_at" json:"diagnosed_at"`
	IsPrimary       bool      `bson:"is_primary" json:"is_primary"`
}

// ProcedureCode represents medical procedures
type ProcedureCode struct {
	Code         string    `bson:"code" json:"code"`
	CodeSystem   string    `bson:"code_system" json:"code_system"` // CPT, ICD10-PCS
	Description  string    `bson:"description" json:"description"`
	PerformedAt  time.Time `bson:"performed_at" json:"performed_at"`
	Status       string    `bson:"status" json:"status"`
}

// AllergyInfo represents patient allergies
type AllergyInfo struct {
	Allergen      string    `bson:"allergen" json:"allergen"`
	AllergenType  string    `bson:"allergen_type" json:"allergen_type"` // drug, food, environmental
	Severity      string    `bson:"severity" json:"severity"`           // mild, moderate, severe
	Reaction      []string  `bson:"reaction" json:"reaction"`
	OnsetDate     *time.Time `bson:"onset_date,omitempty" json:"onset_date,omitempty"`
	Status        string    `bson:"status" json:"status"`               // active, inactive, resolved
}

// SocialHistory captures relevant social determinants
type SocialHistory struct {
	SmokingStatus   string `bson:"smoking_status,omitempty" json:"smoking_status,omitempty"`
	AlcoholUse      string `bson:"alcohol_use,omitempty" json:"alcohol_use,omitempty"`
	DrugUse         string `bson:"drug_use,omitempty" json:"drug_use,omitempty"`
	MaritalStatus   string `bson:"marital_status,omitempty" json:"marital_status,omitempty"`
	EmploymentStatus string `bson:"employment_status,omitempty" json:"employment_status,omitempty"`
	InsuranceType   string `bson:"insurance_type,omitempty" json:"insurance_type,omitempty"`
}

// RiskFactor represents clinical risk factors
type RiskFactor struct {
	FactorType      string    `bson:"factor_type" json:"factor_type"`
	FactorName      string    `bson:"factor_name" json:"factor_name"`
	RiskScore       float64   `bson:"risk_score" json:"risk_score"`       // 0.0 to 1.0
	RiskCategory    string    `bson:"risk_category" json:"risk_category"` // low, moderate, high, very_high
	Evidence        []string  `bson:"evidence" json:"evidence"`
	AssessedAt      time.Time `bson:"assessed_at" json:"assessed_at"`
	ValidUntil      *time.Time `bson:"valid_until,omitempty" json:"valid_until,omitempty"`
}

// MongoDBPhenotypeDefinition represents a clinical phenotype
type MongoDBPhenotypeDefinition struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty" json:"id,omitempty"`
	PhenotypeID    string             `bson:"phenotype_id" json:"phenotype_id"`
	Name           string             `bson:"name" json:"name"`
	Description    string             `bson:"description" json:"description"`
	Category       string             `bson:"category" json:"category"` // cardiovascular, endocrine, etc.
	Severity       string             `bson:"severity" json:"severity"` // mild, moderate, severe
	Criteria       MongoDBPhenotypeCriteria  `bson:"criteria" json:"criteria"`
	ICD10Codes     []string           `bson:"icd10_codes" json:"icd10_codes"`
	SNOMEDCodes    []string           `bson:"snomed_codes" json:"snomed_codes"`
	AlgorithmType  string             `bson:"algorithm_type" json:"algorithm_type"` // rule_based, ml_model, hybrid
	Algorithm      PhenotypeAlgorithm `bson:"algorithm" json:"algorithm"`
	ValidationData ValidationData     `bson:"validation_data" json:"validation_data"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
	Version        string             `bson:"version" json:"version"`
	Status         string             `bson:"status" json:"status"` // active, deprecated, testing
}

// MongoDBPhenotypeCriteria defines the criteria for phenotype identification
type MongoDBPhenotypeCriteria struct {
	RequiredConditions []string              `bson:"required_conditions" json:"required_conditions"`
	ExclusionConditions []string             `bson:"exclusion_conditions" json:"exclusion_conditions"`
	LabValueRules      []LabValueRule       `bson:"lab_value_rules" json:"lab_value_rules"`
	MedicationRules    []MedicationRule     `bson:"medication_rules" json:"medication_rules"`
	AgeRestrictions    *AgeRestriction      `bson:"age_restrictions,omitempty" json:"age_restrictions,omitempty"`
	GenderRestrictions []string             `bson:"gender_restrictions,omitempty" json:"gender_restrictions,omitempty"`
	TimeWindows        *TimeWindow          `bson:"time_windows,omitempty" json:"time_windows,omitempty"`
}

// LabValueRule defines rules for lab value evaluation
type LabValueRule struct {
	LabCode      string  `bson:"lab_code" json:"lab_code"`
	LOINCCode    string  `bson:"loinc_code,omitempty" json:"loinc_code,omitempty"`
	Operator     string  `bson:"operator" json:"operator"` // gt, lt, eq, gte, lte, between
	Value        float64 `bson:"value" json:"value"`
	Unit         string  `bson:"unit" json:"unit"`
	Required     bool    `bson:"required" json:"required"`
	Weight       float64 `bson:"weight" json:"weight"` // Contribution weight to phenotype score
}

// MedicationRule defines rules for medication evaluation
type MedicationRule struct {
	MedicationClass []string `bson:"medication_class" json:"medication_class"`
	RxNormCodes     []string `bson:"rxnorm_codes,omitempty" json:"rxnorm_codes,omitempty"`
	Required        bool     `bson:"required" json:"required"`
	MinDuration     *int     `bson:"min_duration_days,omitempty" json:"min_duration_days,omitempty"` // Minimum days on medication
	Weight          float64  `bson:"weight" json:"weight"`
}

// AgeRestriction defines age-based restrictions
type AgeRestriction struct {
	MinAge *int `bson:"min_age,omitempty" json:"min_age,omitempty"`
	MaxAge *int `bson:"max_age,omitempty" json:"max_age,omitempty"`
}

// TimeWindow defines temporal constraints for phenotype detection
type TimeWindow struct {
	LookbackDays   int `bson:"lookback_days" json:"lookback_days"`     // How far back to look for evidence
	RequiredWithin int `bson:"required_within" json:"required_within"` // All criteria must be met within this window
}

// PhenotypeAlgorithm contains the algorithm details
type PhenotypeAlgorithm struct {
	Type           string                 `bson:"type" json:"type"` // rule_based, ml_model, hybrid
	RuleLogic      string                 `bson:"rule_logic,omitempty" json:"rule_logic,omitempty"` // AND, OR, custom expression
	ModelDetails   *MLModelDetails        `bson:"model_details,omitempty" json:"model_details,omitempty"`
	Thresholds     map[string]float64     `bson:"thresholds" json:"thresholds"`
	Parameters     map[string]interface{} `bson:"parameters,omitempty" json:"parameters,omitempty"`
}

// MLModelDetails for machine learning phenotype models
type MLModelDetails struct {
	ModelType    string   `bson:"model_type" json:"model_type"` // logistic_regression, random_forest, etc.
	Features     []string `bson:"features" json:"features"`
	ModelPath    string   `bson:"model_path,omitempty" json:"model_path,omitempty"`
	Version      string   `bson:"version" json:"version"`
	Accuracy     float64  `bson:"accuracy" json:"accuracy"`
	Sensitivity  float64  `bson:"sensitivity" json:"sensitivity"`
	Specificity  float64  `bson:"specificity" json:"specificity"`
}

// ValidationData holds validation metrics for the phenotype
type ValidationData struct {
	ValidationDataset string    `bson:"validation_dataset" json:"validation_dataset"`
	PPV               float64   `bson:"ppv" json:"ppv"`               // Positive Predictive Value
	NPV               float64   `bson:"npv" json:"npv"`               // Negative Predictive Value
	Sensitivity       float64   `bson:"sensitivity" json:"sensitivity"`
	Specificity       float64   `bson:"specificity" json:"specificity"`
	F1Score           float64   `bson:"f1_score" json:"f1_score"`
	AUC               float64   `bson:"auc,omitempty" json:"auc,omitempty"`
	ValidatedAt       time.Time `bson:"validated_at" json:"validated_at"`
	ValidatedBy       string    `bson:"validated_by" json:"validated_by"`
}

// PhenotypeMatch represents a matched phenotype for a patient
type PhenotypeMatch struct {
	PhenotypeID      string                 `bson:"phenotype_id" json:"phenotype_id"`
	PhenotypeName    string                 `bson:"phenotype_name" json:"phenotype_name"`
	MatchScore       float64                `bson:"match_score" json:"match_score"`       // 0.0 to 1.0
	Confidence       float64                `bson:"confidence" json:"confidence"`         // 0.0 to 1.0
	MatchedCriteria  []string               `bson:"matched_criteria" json:"matched_criteria"`
	Evidence         []EvidenceItem         `bson:"evidence" json:"evidence"`
	MatchedAt        time.Time              `bson:"matched_at" json:"matched_at"`
	AlgorithmVersion string                 `bson:"algorithm_version" json:"algorithm_version"`
	Metadata         map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// EvidenceItem represents evidence supporting a phenotype match
type EvidenceItem struct {
	EvidenceType   string    `bson:"evidence_type" json:"evidence_type"` // condition, lab, medication, etc.
	Description    string    `bson:"description" json:"description"`
	Value          string    `bson:"value,omitempty" json:"value,omitempty"`
	Timestamp      time.Time `bson:"timestamp" json:"timestamp"`
	Weight         float64   `bson:"weight" json:"weight"`
	Confidence     float64   `bson:"confidence" json:"confidence"`
}

// ContextualInsight represents generated insights about a patient's context
type ContextualInsight struct {
	InsightID       string                 `bson:"insight_id" json:"insight_id"`
	InsightType     string                 `bson:"insight_type" json:"insight_type"` // risk_alert, care_opportunity, etc.
	Title           string                 `bson:"title" json:"title"`
	Description     string                 `bson:"description" json:"description"`
	Priority        string                 `bson:"priority" json:"priority"`         // low, medium, high, critical
	ConfidenceScore float64                `bson:"confidence_score" json:"confidence_score"`
	Evidence        []EvidenceItem         `bson:"evidence" json:"evidence"`
	Recommendations []string               `bson:"recommendations" json:"recommendations"`
	GeneratedAt     time.Time              `bson:"generated_at" json:"generated_at"`
	ExpiresAt       *time.Time             `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	Status          string                 `bson:"status" json:"status"` // active, acknowledged, resolved
	Metadata        map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// PatientProfile aggregates patient clinical context data
type PatientProfile struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"id,omitempty"`
	PatientID    string              `bson:"patient_id" json:"patient_id"`
	Demographics MongoDBDemographics        `bson:"demographics" json:"demographics"`
	Phenotypes   []PhenotypeMatch    `bson:"phenotypes" json:"phenotypes"`
	RiskProfile  RiskProfile         `bson:"risk_profile" json:"risk_profile"`
	CohortMemberships []CohortMembership `bson:"cohort_memberships" json:"cohort_memberships"`
	LastUpdated  time.Time           `bson:"last_updated" json:"last_updated"`
	DataVersion  int                 `bson:"data_version" json:"data_version"`
}

// RiskProfile aggregates risk information for a patient
type RiskProfile struct {
	OverallRiskScore    float64                `bson:"overall_risk_score" json:"overall_risk_score"`
	RiskCategory        string                 `bson:"risk_category" json:"risk_category"`
	DomainRiskScores    map[string]float64     `bson:"domain_risk_scores" json:"domain_risk_scores"`
	ActiveRiskFactors   []RiskFactor           `bson:"active_risk_factors" json:"active_risk_factors"`
	RiskTrends          []RiskTrendPoint       `bson:"risk_trends,omitempty" json:"risk_trends,omitempty"`
	NextAssessmentDue   *time.Time             `bson:"next_assessment_due,omitempty" json:"next_assessment_due,omitempty"`
}

// RiskTrendPoint represents a point in risk trend analysis
type RiskTrendPoint struct {
	Date       time.Time `bson:"date" json:"date"`
	RiskScore  float64   `bson:"risk_score" json:"risk_score"`
	Category   string    `bson:"category" json:"category"`
	Contributors []string `bson:"contributors,omitempty" json:"contributors,omitempty"`
}

// CohortMembership represents membership in population cohorts
type CohortMembership struct {
	CohortID     string    `bson:"cohort_id" json:"cohort_id"`
	CohortName   string    `bson:"cohort_name" json:"cohort_name"`
	JoinedAt     time.Time `bson:"joined_at" json:"joined_at"`
	MatchScore   float64   `bson:"match_score" json:"match_score"`
	Status       string    `bson:"status" json:"status"`
}

// PopulationCohort defines a population cohort
type PopulationCohort struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CohortID    string             `bson:"cohort_id" json:"cohort_id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Criteria    CohortCriteria     `bson:"criteria" json:"criteria"`
	Statistics  CohortStatistics   `bson:"statistics" json:"statistics"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
	Status      string             `bson:"status" json:"status"`
}

// CohortCriteria defines inclusion/exclusion criteria for cohorts
type CohortCriteria struct {
	InclusionCriteria []CriteriaRule `bson:"inclusion_criteria" json:"inclusion_criteria"`
	ExclusionCriteria []CriteriaRule `bson:"exclusion_criteria" json:"exclusion_criteria"`
	Phenotypes        []string       `bson:"phenotypes" json:"phenotypes"`
	AgeRange          *AgeRange      `bson:"age_range,omitempty" json:"age_range,omitempty"`
	Gender            []string       `bson:"gender,omitempty" json:"gender,omitempty"`
	GeographicRegions []string       `bson:"geographic_regions,omitempty" json:"geographic_regions,omitempty"`
}

// CriteriaRule represents a single cohort criteria rule
type CriteriaRule struct {
	RuleType    string                 `bson:"rule_type" json:"rule_type"`
	Field       string                 `bson:"field" json:"field"`
	Operator    string                 `bson:"operator" json:"operator"`
	Value       interface{}            `bson:"value" json:"value"`
	Weight      float64                `bson:"weight" json:"weight"`
	Required    bool                   `bson:"required" json:"required"`
	Parameters  map[string]interface{} `bson:"parameters,omitempty" json:"parameters,omitempty"`
}

// CohortStatistics holds statistics about the cohort
type CohortStatistics struct {
	MemberCount     int                    `bson:"member_count" json:"member_count"`
	Demographics    DemographicDistribution `bson:"demographics" json:"demographics"`
	TopPhenotypes   []PhenotypeFrequency   `bson:"top_phenotypes" json:"top_phenotypes"`
	AvgRiskScore    float64                `bson:"avg_risk_score" json:"avg_risk_score"`
	LastComputed    time.Time              `bson:"last_computed" json:"last_computed"`
}

// DemographicDistribution shows demographic breakdown
type DemographicDistribution struct {
	AgeGroups     map[string]int `bson:"age_groups" json:"age_groups"`
	GenderSplit   map[string]int `bson:"gender_split" json:"gender_split"`
	EthnicitySplit map[string]int `bson:"ethnicity_split,omitempty" json:"ethnicity_split,omitempty"`
}

// PhenotypeFrequency represents phenotype frequency in cohorts
type PhenotypeFrequency struct {
	PhenotypeID   string  `bson:"phenotype_id" json:"phenotype_id"`
	PhenotypeName string  `bson:"phenotype_name" json:"phenotype_name"`
	Count         int     `bson:"count" json:"count"`
	Frequency     float64 `bson:"frequency" json:"frequency"`
}

// ContextCache represents cached context computation results
type ContextCache struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CacheKey  string             `bson:"cache_key" json:"cache_key"`
	Data      interface{}        `bson:"data" json:"data"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}