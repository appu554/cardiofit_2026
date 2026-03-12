package clients

import (
	"time"
)

// Knowledge Base Types - based on the 7 KB microservices architecture

// DrugRules represents comprehensive drug dosing and safety rules
type DrugRules struct {
	DrugID          string             `json:"drugId"`
	Version         string             `json:"version"`
	ContentSha      string             `json:"contentSha"`
	CreatedAt       time.Time          `json:"createdAt"`
	SignedBy        string             `json:"signedBy"`
	SignatureValid  bool               `json:"signatureValid"`
	ClinicalReviewer string            `json:"clinicalReviewer"`
	ClinicalReviewDate time.Time       `json:"clinicalReviewDate"`
	Regions         []string           `json:"regions"`
	SelectedRegion  *string            `json:"selectedRegion"`
	Content         DrugRuleContent    `json:"content"`
	CacheControl    string             `json:"cacheControl"`
	Etag            string             `json:"etag"`
}

type DrugRuleContent struct {
	Meta                RuleMetadata              `json:"meta"`
	DoseCalculation     DoseCalculation           `json:"doseCalculation"`
	SafetyVerification  SafetyVerification        `json:"safetyVerification"`
	MonitoringRequirements []MonitoringRequirement `json:"monitoringRequirements"`
	RegionalVariations  map[string]RegionalOverride `json:"regionalVariations"`
}

type RuleMetadata struct {
	Name           string    `json:"name"`
	Version        string    `json:"version"`
	Description    string    `json:"description"`
	EffectiveDate  time.Time `json:"effectiveDate"`
	ExpirationDate *time.Time `json:"expirationDate"`
	Author         string    `json:"author"`
	ReviewedBy     string    `json:"reviewedBy"`
}

type DoseCalculation struct {
	StandardDose       float64              `json:"standardDose"`
	MaxDailyDose      float64              `json:"maxDailyDose"`
	MinDailyDose      float64              `json:"minDailyDose"`
	RenalAdjustment   []RenalAdjustment    `json:"renalAdjustment"`
	HepaticAdjustment []HepaticAdjustment  `json:"hepaticAdjustment"`
	AgeAdjustment     []AgeAdjustment      `json:"ageAdjustment"`
	WeightBased       bool                 `json:"weightBased"`
	Formula           string               `json:"formula"`
}

type RenalAdjustment struct {
	GFRThreshold float64 `json:"gfrThreshold"`
	Adjustment   float64 `json:"adjustment"`
	Method       string  `json:"method"` // "multiply", "fixed", "avoid"
}

type HepaticAdjustment struct {
	Severity   string  `json:"severity"` // "mild", "moderate", "severe"
	Adjustment float64 `json:"adjustment"`
	Method     string  `json:"method"`
}

type AgeAdjustment struct {
	MinAge     *int    `json:"minAge"`
	MaxAge     *int    `json:"maxAge"`
	Adjustment float64 `json:"adjustment"`
	Method     string  `json:"method"`
}

type SafetyVerification struct {
	Contraindications []string            `json:"contraindications"`
	Warnings          []string            `json:"warnings"`
	Precautions       []string            `json:"precautions"`
	BlackBoxWarnings  []string            `json:"blackBoxWarnings"`
	Monitoring        []MonitoringRule    `json:"monitoring"`
}

type MonitoringRule struct {
	Parameter  string             `json:"parameter"`
	Frequency  string             `json:"frequency"`
	Thresholds MonitoringThreshold `json:"thresholds"`
}

type MonitoringThreshold struct {
	Normal   Range   `json:"normal"`
	Warning  Range   `json:"warning"`
	Critical Range   `json:"critical"`
}

type Range struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type MonitoringRequirement struct {
	Parameter     string  `json:"parameter"`
	Frequency     string  `json:"frequency"`
	NormalRange   Range   `json:"normalRange"`
	CriticalRange Range   `json:"criticalRange"`
	Units         string  `json:"units"`
	Rationale     string  `json:"rationale"`
}

type RegionalOverride struct {
	Region     string                 `json:"region"`
	Overrides  map[string]interface{} `json:"overrides"`
	Effective  time.Time              `json:"effective"`
	Rationale  string                 `json:"rationale"`
}

// Drug Interaction Types
type InteractionCheckRequest struct {
	ActiveMedications []ActiveMedication `json:"activeMedications"`
	CandidateDrug    string             `json:"candidateDrug"`
}

type ActiveMedication struct {
	DrugID      string `json:"drugId"`
	Dose        *float64 `json:"dose,omitempty"`
	Frequency   *string  `json:"frequency,omitempty"`
	StartDate   *time.Time `json:"startDate,omitempty"`
}

type InteractionCheckResponse struct {
	CandidateDrug    string              `json:"candidateDrug"`
	Interactions     []DrugInteraction   `json:"interactions"`
	OverallAction    OverallAction       `json:"overallAction"`
	ClinicalSummary  string              `json:"clinicalSummary"`
	ProcessingTime   string              `json:"processingTime"`
	SLACompliant     bool                `json:"slaCompliant"`
}

type DrugInteraction struct {
	Substrate      string                `json:"substrate"`
	Perpetrator    string                `json:"perpetrator"`
	Severity       InteractionSeverity   `json:"severity"`
	Mechanism      string                `json:"mechanism"`
	ClinicalEffect string                `json:"clinicalEffect"`
	Management     ManagementStrategy    `json:"management"`
	EvidenceLevel  EvidenceLevel         `json:"evidenceLevel"`
	References     []Reference           `json:"references"`
	Onset          InteractionOnset      `json:"onset"`
	Probability    float64               `json:"probability"`
}

type InteractionSeverity string

const (
	SeverityContraindicated InteractionSeverity = "CONTRAINDICATED"
	SeverityMajor          InteractionSeverity = "MAJOR"
	SeverityModerate       InteractionSeverity = "MODERATE"
	SeverityMinor          InteractionSeverity = "MINOR"
)

type ManagementStrategy struct {
	Action         ManagementAction  `json:"action"`
	DoseAdjustment *DoseAdjustment   `json:"doseAdjustment,omitempty"`
	Monitoring     []MonitoringParameter `json:"monitoring"`
	Alternatives   []string          `json:"alternatives"`
	Instructions   string            `json:"instructions"`
}

type ManagementAction string

const (
	ActionAvoid                ManagementAction = "AVOID"
	ActionAdjustDose          ManagementAction = "ADJUST_DOSE"
	ActionMonitorClosely      ManagementAction = "MONITOR_CLOSELY"
	ActionSeparateAdministration ManagementAction = "SEPARATE_ADMINISTRATION"
	ActionNoActionNeeded      ManagementAction = "NO_ACTION_NEEDED"
)

type DoseAdjustment struct {
	Type   string  `json:"type"`   // "reduce", "increase", "fixed"
	Factor float64 `json:"factor"` // multiplier or fixed amount
	Units  string  `json:"units"`
}

type MonitoringParameter struct {
	Parameter string `json:"parameter"`
	Frequency string `json:"frequency"`
	Rationale string `json:"rationale"`
}

type EvidenceLevel string

const (
	EvidenceLevelHigh     EvidenceLevel = "HIGH"
	EvidenceLevelModerate EvidenceLevel = "MODERATE"
	EvidenceLevelLow      EvidenceLevel = "LOW"
	EvidenceLevelTheoretical EvidenceLevel = "THEORETICAL"
)

type Reference struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Authors     []string `json:"authors"`
	Journal     string `json:"journal"`
	Year        int    `json:"year"`
	PMID        *string `json:"pmid,omitempty"`
	DOI         *string `json:"doi,omitempty"`
}

type InteractionOnset string

const (
	OnsetImmediate InteractionOnset = "IMMEDIATE"
	OnsetRapid     InteractionOnset = "RAPID"
	OnsetDelayed   InteractionOnset = "DELAYED"
)

type OverallAction string

const (
	OverallActionBlock         OverallAction = "BLOCK"
	OverallActionRequireOverride OverallAction = "REQUIRE_OVERRIDE"
	OverallActionProceed       OverallAction = "PROCEED"
)

// Patient Safety Profile Types
type PatientSafetyProfile struct {
	PatientID             string                  `json:"patientId"`
	SafetyFlags           []SafetyFlag            `json:"safetyFlags"`
	ContraindicationCodes []ContraindicationCode  `json:"contraindicationCodes"`
	RiskScores            map[string]RiskScore    `json:"riskScores"`
	Phenotypes            []ClinicalPhenotype     `json:"phenotypes"`
	GeneratedAt           time.Time               `json:"generatedAt"`
	ExpiresAt             *time.Time              `json:"expiresAt,omitempty"`
	Version               int                     `json:"version"`
}

type SafetyFlag struct {
	FlagType     SafetyFlagType `json:"flagType"`
	Value        bool           `json:"value"`
	Confidence   float64        `json:"confidence"`
	Source       DataSource     `json:"source"`
	LastVerified time.Time      `json:"lastVerified"`
}

type SafetyFlagType struct {
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes"`
}

type DataSource struct {
	System    string    `json:"system"`
	Reference string    `json:"reference"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ContraindicationCode struct {
	Code        string                      `json:"code"`
	System      string                      `json:"system"`
	Description string                      `json:"description"`
	Severity    ContraindicationSeverity    `json:"severity"`
}

type ContraindicationSeverity string

const (
	ContraindicationAbsolute ContraindicationSeverity = "ABSOLUTE"
	ContraindicationRelative ContraindicationSeverity = "RELATIVE"
	ContraindicationCaution  ContraindicationSeverity = "CAUTION"
)

type RiskScore struct {
	Category   string    `json:"category"`
	Score      float64   `json:"score"`
	Percentile *float64  `json:"percentile,omitempty"`
	Level      RiskLevel `json:"level"`
	LastCalculated time.Time `json:"lastCalculated"`
}

type RiskLevel string

const (
	RiskLow      RiskLevel = "LOW"
	RiskModerate RiskLevel = "MODERATE"
	RiskHigh     RiskLevel = "HIGH"
	RiskVeryHigh RiskLevel = "VERY_HIGH"
	RiskExtreme  RiskLevel = "EXTREME"
)

// Clinical Phenotype Types (from KB-2 Clinical Context)
type ClinicalPhenotype struct {
	ID              string                      `json:"id"`
	Name            string                      `json:"name"`
	Category        string                      `json:"category"`
	Domain          string                      `json:"domain"`
	Priority        int                         `json:"priority"`
	Matched         bool                        `json:"matched"`
	Confidence      float64                     `json:"confidence"`
	CELRule         string                      `json:"celRule"`
	Implications    []ClinicalImplication       `json:"implications"`
	EvaluationDetails *PhenotypeEvaluationDetails `json:"evaluationDetails,omitempty"`
	LastEvaluated   time.Time                   `json:"lastEvaluated"`
}

type ClinicalImplication struct {
	Type              string              `json:"type"`
	Severity          ImplicationSeverity `json:"severity"`
	Description       string              `json:"description"`
	Recommendations   []string            `json:"recommendations"`
	ClinicalEvidence  *string             `json:"clinicalEvidence,omitempty"`
}

type ImplicationSeverity string

const (
	ImplicationInformational ImplicationSeverity = "INFORMATIONAL"
	ImplicationLow          ImplicationSeverity = "LOW"
	ImplicationModerate     ImplicationSeverity = "MODERATE"
	ImplicationHigh         ImplicationSeverity = "HIGH"
	ImplicationCritical     ImplicationSeverity = "CRITICAL"
)

type PhenotypeEvaluationDetails struct {
	EvaluationPath     []string           `json:"evaluationPath"`
	FactorsConsidered  []EvaluationFactor `json:"factorsConsidered"`
	CELExpression      string             `json:"celExpression"`
	ExecutionTime      string             `json:"executionTime"`
}

type EvaluationFactor struct {
	Name         string   `json:"name"`
	Value        string   `json:"value"`
	Weight       *float64 `json:"weight,omitempty"`
	Contribution string   `json:"contribution"`
}

// Patient Clinical Data Types
type PatientClinicalData struct {
	ID            string              `json:"id"`
	Age           int                 `json:"age"`
	Gender        string              `json:"gender"`
	Conditions    []string            `json:"conditions"`
	Medications   []string            `json:"medications"`
	Labs          []LabValue          `json:"labs"`
	Vitals        []VitalSign         `json:"vitals"`
	Procedures    []string            `json:"procedures"`
	Allergies     []string            `json:"allergies"`
	FamilyHistory []string            `json:"familyHistory"`
	SocialHistory *SocialHistory      `json:"socialHistory,omitempty"`
}

type LabValue struct {
	Name           string     `json:"name"`
	Value          float64    `json:"value"`
	Unit           string     `json:"unit"`
	ReferenceRange *string    `json:"referenceRange,omitempty"`
	TestDate       *time.Time `json:"testDate,omitempty"`
}

type VitalSign struct {
	Name            string     `json:"name"`
	Value           float64    `json:"value"`
	Unit            string     `json:"unit"`
	MeasurementDate *time.Time `json:"measurementDate,omitempty"`
}

type SocialHistory struct {
	SmokingStatus      *string `json:"smokingStatus,omitempty"`
	AlcoholUse         *string `json:"alcoholUse,omitempty"`
	ExerciseFrequency  *string `json:"exerciseFrequency,omitempty"`
	DietaryPatterns    *string `json:"dietaryPatterns,omitempty"`
}

// Clinical Pathway Types
type ClinicalPathway struct {
	PathwayID        string               `json:"pathwayId"`
	Version          string               `json:"version"`
	Condition        string               `json:"condition"`
	PatientCriteria  PatientCriteria      `json:"patientCriteria"`
	Steps            []PathwayStep        `json:"steps"`
	DecisionPoints   []DecisionPoint      `json:"decisionPoints"`
	Outcomes         []ExpectedOutcome    `json:"outcomes"`
	EvidenceBase     []Evidence           `json:"evidenceBase"`
}

type PatientCriteria struct {
	InclusionCriteria []string `json:"inclusionCriteria"`
	ExclusionCriteria []string `json:"exclusionCriteria"`
	AgeRange          *Range   `json:"ageRange,omitempty"`
}

type PathwayStep struct {
	StepID         string           `json:"stepId"`
	StepType       StepType         `json:"stepType"`
	Actions        []ClinicalAction `json:"actions"`
	Timing         Timing           `json:"timing"`
	Prerequisites  []string         `json:"prerequisites"`
	ExitCriteria   []Criterion      `json:"exitCriteria"`
}

type StepType string

const (
	StepDiagnostic   StepType = "DIAGNOSTIC"
	StepTherapeutic  StepType = "THERAPEUTIC"
	StepMonitoring   StepType = "MONITORING"
	StepReassessment StepType = "REASSESSMENT"
	StepDischarge    StepType = "DISCHARGE"
)

type ClinicalAction struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type Timing struct {
	Duration string `json:"duration"`
	Frequency string `json:"frequency"`
	StartCondition string `json:"startCondition"`
}

type Criterion struct {
	Condition string `json:"condition"`
	Operator  string `json:"operator"`
	Value     string `json:"value"`
}

type DecisionPoint struct {
	ID          string              `json:"id"`
	Condition   string              `json:"condition"`
	Branches    []DecisionBranch    `json:"branches"`
}

type DecisionBranch struct {
	Condition   string `json:"condition"`
	NextStepID  string `json:"nextStepId"`
	Probability float64 `json:"probability"`
}

type ExpectedOutcome struct {
	Outcome     string  `json:"outcome"`
	Probability float64 `json:"probability"`
	Timeframe   string  `json:"timeframe"`
}

type Evidence struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Type         string `json:"type"`
	Quality      string `json:"quality"`
	Relevance    float64 `json:"relevance"`
}

// Formulary Coverage Types
type CoverageResponse struct {
	Covered      bool                     `json:"covered"`
	Tier         FormularyTier            `json:"tier"`
	PatientCost  float64                  `json:"patientCost"`
	Restrictions []Restriction            `json:"restrictions"`
	Alternatives []FormularyAlternative   `json:"alternatives"`
}

type FormularyTier string

const (
	TierGeneric      FormularyTier = "TIER1_GENERIC"
	TierPreferred    FormularyTier = "TIER2_PREFERRED"
	TierNonPreferred FormularyTier = "TIER3_NON_PREFERRED"
	TierSpecialty    FormularyTier = "TIER4_SPECIALTY"
	TierNotCovered   FormularyTier = "NOT_COVERED"
)

type Restriction struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Requirements []string `json:"requirements"`
}

type FormularyAlternative struct {
	DrugID       string        `json:"drugId"`
	DrugName     string        `json:"drugName"`
	Tier         FormularyTier `json:"tier"`
	PatientCost  float64       `json:"patientCost"`
	Rationale    string        `json:"rationale"`
}

// Drug Master Types
type DrugMasterEntry struct {
	DrugID               string                          `json:"drugId"`
	RxNormID             string                          `json:"rxNormId"`
	GenericName          string                          `json:"genericName"`
	BrandNames           []string                        `json:"brandNames"`
	TherapeuticClass     []string                        `json:"therapeuticClass"`
	PharmacologicClass   string                          `json:"pharmacologicClass"`
	Routes               []Route                         `json:"routes"`
	DoseForms            []DoseForm                      `json:"doseForms"`
	AvailableStrengths   []Strength                      `json:"availableStrengths"`
	PKProperties         PharmacokineticProperties       `json:"pkProperties"`
	SpecialPopulations   SpecialPopulationConsiderations `json:"specialPopulations"`
	BoxedWarnings        []BoxedWarning                  `json:"boxedWarnings"`
}

type Route struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type DoseForm struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type Strength struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type PharmacokineticProperties struct {
	HalfLifeHours       Range                `json:"halfLifeHours"`
	ProteinBindingPercent float64            `json:"proteinBindingPercent"`
	VolumeDistribution  float64              `json:"volumeDistribution"`
	Clearance           float64              `json:"clearance"`
	Bioavailability     float64              `json:"bioavailability"`
	Metabolism          []MetabolicPathway   `json:"metabolism"`
	Elimination         []EliminationRoute   `json:"elimination"`
	NarrowTherapeuticIndex bool              `json:"narrowTherapeuticIndex"`
}

type MetabolicPathway struct {
	Enzyme      string  `json:"enzyme"`
	Percentage  float64 `json:"percentage"`
	Description string  `json:"description"`
}

type EliminationRoute struct {
	Route      string  `json:"route"`
	Percentage float64 `json:"percentage"`
}

type SpecialPopulationConsiderations struct {
	Pregnancy  PregnancyConsiderations `json:"pregnancy"`
	Pediatric  PediatricConsiderations `json:"pediatric"`
	Geriatric  GeriatricConsiderations `json:"geriatric"`
	Renal      RenalConsiderations     `json:"renal"`
	Hepatic    HepaticConsiderations   `json:"hepatic"`
}

type PregnancyConsiderations struct {
	Category    string `json:"category"`
	Trimester   string `json:"trimester"`
	Lactation   string `json:"lactation"`
	Rationale   string `json:"rationale"`
}

type PediatricConsiderations struct {
	MinAge      int    `json:"minAge"`
	DoseAdjustment string `json:"doseAdjustment"`
	SafetyConcerns []string `json:"safetyConcerns"`
}

type GeriatricConsiderations struct {
	BeersListCriteria bool     `json:"beersListCriteria"`
	DoseAdjustment   string   `json:"doseAdjustment"`
	SafetyConcerns   []string `json:"safetyConcerns"`
}

type RenalConsiderations struct {
	RequiresAdjustment bool                `json:"requiresAdjustment"`
	GFRThresholds      []RenalAdjustment   `json:"gfrThresholds"`
}

type HepaticConsiderations struct {
	RequiresAdjustment bool                  `json:"requiresAdjustment"`
	Adjustments        []HepaticAdjustment   `json:"adjustments"`
}

type BoxedWarning struct {
	Warning     string    `json:"warning"`
	Rationale   string    `json:"rationale"`
	EffectiveDate time.Time `json:"effectiveDate"`
}

// Terminology Mapping Types
type TerminologyMapping struct {
	SourceSystem string       `json:"sourceSystem"`
	SourceCode   string       `json:"sourceCode"`
	TargetSystem string       `json:"targetSystem"`
	TargetCodes  []TargetCode `json:"targetCodes"`
	MappingType  MappingType  `json:"mappingType"`
	Validity     Validity     `json:"validity"`
}

type TargetCode struct {
	Code        string  `json:"code"`
	Display     string  `json:"display"`
	Equivalence string  `json:"equivalence"`
	Comments    *string `json:"comments,omitempty"`
}

type MappingType string

const (
	MappingExact       MappingType = "EXACT"
	MappingApproximate MappingType = "APPROXIMATE"
	MappingBroader     MappingType = "BROADER"
	MappingNarrower    MappingType = "NARROWER"
	MappingInexact     MappingType = "INEXACT"
)

type Validity struct {
	ValidFrom time.Time  `json:"validFrom"`
	ValidTo   *time.Time `json:"validTo,omitempty"`
}

// Response Types for Complex Operations

type PhenotypeEvaluationResponse struct {
	Results        []PatientPhenotypeResult `json:"results"`
	ProcessingTime string                   `json:"processingTime"`
	BatchSize      int                      `json:"batchSize"`
	SLACompliant   bool                     `json:"slaCompliant"`
	Metadata       ProcessingMetadata       `json:"metadata"`
}

type PatientPhenotypeResult struct {
	PatientID         string               `json:"patientId"`
	Phenotypes        []ClinicalPhenotype  `json:"phenotypes"`
	EvaluationSummary EvaluationSummary    `json:"evaluationSummary"`
}

type EvaluationSummary struct {
	TotalPhenotypes       int     `json:"totalPhenotypes"`
	MatchedPhenotypes     int     `json:"matchedPhenotypes"`
	HighConfidenceMatches int     `json:"highConfidenceMatches"`
	AverageConfidence     float64 `json:"averageConfidence"`
	ProcessingTime        string  `json:"processingTime"`
}

type ProcessingMetadata struct {
	CacheHitRate           float64  `json:"cacheHitRate"`
	AverageProcessingTime  string   `json:"averageProcessingTime"`
	ComponentsProcessed    []string `json:"componentsProcessed"`
	ErrorCount             int      `json:"errorCount"`
}

type RiskAssessmentResponse struct {
	PatientID          string              `json:"patientId"`
	RiskAssessments    []RiskAssessment    `json:"riskAssessments"`
	OverallRiskProfile RiskProfile         `json:"overallRiskProfile"`
	ProcessingTime     string              `json:"processingTime"`
	SLACompliant       bool                `json:"slaCompliant"`
}

type RiskAssessment struct {
	ID                 string                `json:"id"`
	Model              string                `json:"model"`
	Category           RiskCategory          `json:"category"`
	Score              float64               `json:"score"`
	Percentile         *float64              `json:"percentile,omitempty"`
	CategoryResult     RiskLevel             `json:"categoryResult"`
	Recommendations    []RiskRecommendation  `json:"recommendations"`
	RiskFactors        []RiskFactor          `json:"riskFactors"`
	CalculationMethod  string                `json:"calculationMethod"`
	ValidUntil         *time.Time            `json:"validUntil,omitempty"`
	LastCalculated     time.Time             `json:"lastCalculated"`
}

type RiskCategory string

const (
	CategoryCardiovascular RiskCategory = "CARDIOVASCULAR"
	CategoryDiabetes       RiskCategory = "DIABETES"
	CategoryMedication     RiskCategory = "MEDICATION"
	CategoryFall           RiskCategory = "FALL"
	CategoryBleeding       RiskCategory = "BLEEDING"
	CategoryKidney         RiskCategory = "KIDNEY"
	CategoryLiver          RiskCategory = "LIVER"
	CategoryRespiratory    RiskCategory = "RESPIRATORY"
	CategoryCognitive      RiskCategory = "COGNITIVE"
	CategoryGeneral        RiskCategory = "GENERAL"
)

type RiskRecommendation struct {
	Priority         int     `json:"priority"`
	Action           string  `json:"action"`
	Rationale        string  `json:"rationale"`
	Urgency          string  `json:"urgency"`
	ClinicalEvidence *string `json:"clinicalEvidence,omitempty"`
}

type RiskFactor struct {
	Name         string              `json:"name"`
	Value        string              `json:"value"`
	Contribution float64             `json:"contribution"`
	Modifiable   bool                `json:"modifiable"`
	Severity     RiskFactorSeverity  `json:"severity"`
}

type RiskFactorSeverity string

const (
	RiskFactorMinor    RiskFactorSeverity = "MINOR"
	RiskFactorModerate RiskFactorSeverity = "MODERATE"
	RiskFactorMajor    RiskFactorSeverity = "MAJOR"
	RiskFactorCritical RiskFactorSeverity = "CRITICAL"
)

type RiskProfile struct {
	OverallRisk        RiskLevel           `json:"overallRisk"`
	PrimaryConcerns    []string            `json:"primaryConcerns"`
	RiskDistribution   []RiskCategoryScore `json:"riskDistribution"`
	RecommendedActions []string            `json:"recommendedActions"`
}

type RiskCategoryScore struct {
	Category RiskCategory `json:"category"`
	Score    float64      `json:"score"`
	Level    RiskLevel    `json:"level"`
	Trend    *string      `json:"trend,omitempty"`
}

type TreatmentPreferencesResponse struct {
	PatientID           string                  `json:"patientId"`
	Condition           string                  `json:"condition"`
	Preferences         TreatmentPreference     `json:"preferences"`
	AlternativeOptions  []TreatmentOption       `json:"alternativeOptions"`
	ConflictResolution  []ConflictResolution    `json:"conflictResolution"`
	ProcessingTime      string                  `json:"processingTime"`
}

type TreatmentPreference struct {
	ID               string                  `json:"id"`
	Condition        string                  `json:"condition"`
	FirstLine        []MedicationPreference  `json:"firstLine"`
	Alternatives     []MedicationPreference  `json:"alternatives"`
	Avoid            []MedicationConstraint  `json:"avoid"`
	Rationale        string                  `json:"rationale"`
	GuidelineSource  string                  `json:"guidelineSource"`
	ConfidenceLevel  float64                 `json:"confidenceLevel"`
	LastUpdated      time.Time               `json:"lastUpdated"`
}

type MedicationPreference struct {
	Medication       MedicationReference `json:"medication"`
	PreferenceScore  float64             `json:"preferenceScore"`
	DosageForm       *string             `json:"dosageForm,omitempty"`
	Frequency        *string             `json:"frequency,omitempty"`
	CostTier         *int                `json:"costTier,omitempty"`
	Reasons          []string            `json:"reasons"`
}

type MedicationReference struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	GenericName  string   `json:"genericName"`
	BrandNames   []string `json:"brandNames"`
	DrugClass    string   `json:"drugClass"`
	Mechanism    *string  `json:"mechanism,omitempty"`
}

type MedicationConstraint struct {
	Medication       MedicationReference   `json:"medication"`
	ConstraintType   ConstraintType        `json:"constraintType"`
	Severity         ConstraintSeverity    `json:"severity"`
	Reason           string                `json:"reason"`
	Alternatives     []MedicationReference `json:"alternatives"`
}

type ConstraintType string

const (
	ConstraintContraindication ConstraintType = "CONTRAINDICATION"
	ConstraintCaution         ConstraintType = "CAUTION"
	ConstraintPreference      ConstraintType = "PREFERENCE"
	ConstraintAllergy         ConstraintType = "ALLERGY"
	ConstraintInteraction     ConstraintType = "INTERACTION"
)

type ConstraintSeverity string

const (
	ConstraintMinor    ConstraintSeverity = "MINOR"
	ConstraintModerate ConstraintSeverity = "MODERATE"
	ConstraintMajor    ConstraintSeverity = "MAJOR"
	ConstraintAbsolute ConstraintSeverity = "ABSOLUTE"
)

type TreatmentOption struct {
	Medication       MedicationReference `json:"medication"`
	SuitabilityScore float64             `json:"suitabilityScore"`
	Rationale        string              `json:"rationale"`
	Considerations   []string            `json:"considerations"`
}

type ConflictResolution struct {
	ConflictType string `json:"conflictType"`
	Resolution   string `json:"resolution"`
	Priority     int    `json:"priority"`
	Reasoning    string `json:"reasoning"`
}

type ClinicalContextResponse struct {
	PatientID       string                    `json:"patientId"`
	Context         ClinicalContext           `json:"context"`
	Warnings        []ContextWarning          `json:"warnings"`
	Recommendations []ContextRecommendation   `json:"recommendations"`
	ProcessingTime  string                    `json:"processingTime"`
	SLACompliant    bool                      `json:"slaCompliant"`
}

type ClinicalContext struct {
	PatientID            string                  `json:"patientId"`
	Phenotypes           []ClinicalPhenotype     `json:"phenotypes"`
	RiskAssessments      []RiskAssessment        `json:"riskAssessments"`
	TreatmentPreferences []TreatmentPreference   `json:"treatmentPreferences"`
	ContextMetadata      ContextMetadata         `json:"contextMetadata"`
	AssemblyTime         time.Time               `json:"assemblyTime"`
	DetailLevel          string                  `json:"detailLevel"`
}

type ContextMetadata struct {
	ProcessingTime       string   `json:"processingTime"`
	SLACompliant         bool     `json:"slaCompliant"`
	DataCompleteness     float64  `json:"dataCompleteness"`
	ConfidenceScore      float64  `json:"confidenceScore"`
	ComponentsEvaluated  []string `json:"componentsEvaluated"`
	CacheHit             bool     `json:"cacheHit"`
}

type ContextWarning struct {
	Severity       WarningSeverity `json:"severity"`
	Category       string          `json:"category"`
	Message        string          `json:"message"`
	ActionRequired *string         `json:"actionRequired,omitempty"`
}

type WarningSeverity string

const (
	WarningInfo     WarningSeverity = "INFO"
	WarningWarning  WarningSeverity = "WARNING"
	WarningError    WarningSeverity = "ERROR"
	WarningCritical WarningSeverity = "CRITICAL"
)

type ContextRecommendation struct {
	Priority       int     `json:"priority"`
	Category       string  `json:"category"`
	Recommendation string  `json:"recommendation"`
	Rationale      string  `json:"rationale"`
	Timeframe      *string `json:"timeframe,omitempty"`
}

type ContextDetailLevel string

const (
	DetailSummary       ContextDetailLevel = "SUMMARY"
	DetailStandard      ContextDetailLevel = "STANDARD"
	DetailComprehensive ContextDetailLevel = "COMPREHENSIVE"
	DetailDetailed      ContextDetailLevel = "DETAILED"
)

// Federation Metrics
type FederationMetrics struct {
	RequestsTotal       int64   `json:"requestsTotal"`
	CacheHitRate        float64 `json:"cacheHitRate"`
	AverageLatencyMs    float64 `json:"averageLatencyMs"`
	ErrorRate           float64 `json:"errorRate"`
	SLACompliance       float64 `json:"slaCompliance"`
	ActiveConnections   int     `json:"activeConnections"`
	QueriesPerSecond    float64 `json:"queriesPerSecond"`
}