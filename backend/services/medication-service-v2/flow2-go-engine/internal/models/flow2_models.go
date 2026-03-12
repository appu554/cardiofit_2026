package models

import (
	"time"
)

// Flow2Request represents a Flow 2 execution request
type Flow2Request struct {
	RequestID         string                 `json:"request_id"`
	PatientID         string                 `json:"patient_id" binding:"required"`
	ActionType        string                 `json:"action_type" binding:"required"`
	MedicationData    map[string]interface{} `json:"medication_data"`
	PatientData       map[string]interface{} `json:"patient_data"`
	ClinicalContext   map[string]interface{} `json:"clinical_context"`
	ProcessingHints   map[string]interface{} `json:"processing_hints"`
	Priority          string                 `json:"priority"`
	EnableMLInference bool                   `json:"enable_ml_inference"`
	Timeout           time.Duration          `json:"timeout"`
	Timestamp         time.Time              `json:"timestamp"`
}

// Flow2Response represents a Flow 2 execution response
type Flow2Response struct {
	RequestID                string                 `json:"request_id"`
	PatientID                string                 `json:"patient_id"`
	OverallStatus            string                 `json:"overall_status"`
	ExecutionSummary         Flow2ExecutionSummary  `json:"execution_summary"`
	RecipeResults            []RecipeResult         `json:"recipe_results"`
	ClinicalDecisionSupport  map[string]interface{} `json:"clinical_decision_support"`
	SafetyAlerts             []SafetyAlert          `json:"safety_alerts"`
	Recommendations          []Recommendation       `json:"recommendations"`
	Analytics                map[string]interface{} `json:"analytics"`
	ExecutionTimeMs          int64                  `json:"execution_time_ms"`
	EngineUsed               string                 `json:"engine_used"`
	Timestamp                time.Time              `json:"timestamp"`
	ProcessingMetadata       ProcessingMetadata     `json:"processing_metadata"`
}

// Flow2ExecutionSummary provides a summary of the Flow 2 execution
type Flow2ExecutionSummary struct {
	TotalRecipesExecuted int    `json:"total_recipes_executed"`
	SuccessfulRecipes    int    `json:"successful_recipes"`
	FailedRecipes        int    `json:"failed_recipes"`
	Warnings             int    `json:"warnings"`
	Errors               int    `json:"errors"`
	Engine               string `json:"engine"`
	CacheHitRate         float64 `json:"cache_hit_rate"`
}

// RecipeResult represents the result of a single recipe execution
type RecipeResult struct {
	RecipeID                string                 `json:"recipe_id"`
	RecipeName              string                 `json:"recipe_name"`
	OverallStatus           string                 `json:"overall_status"`
	ExecutionTimeMs         int64                  `json:"execution_time_ms"`
	Validations             []RecipeValidation     `json:"validations"`
	ClinicalDecisionSupport map[string]interface{} `json:"clinical_decision_support"`
	Recommendations         []string               `json:"recommendations"`
	Warnings                []string               `json:"warnings"`
	Errors                  []string               `json:"errors"`
	Metadata                map[string]interface{} `json:"metadata"`
}

// RecipeValidation represents a validation result from a recipe
type RecipeValidation struct {
	Passed      bool   `json:"passed"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Explanation string `json:"explanation"`
	Code        string `json:"code"`
}

// SafetyAlert represents a safety alert
type SafetyAlert struct {
	AlertID     string `json:"alert_id"`
	Severity    string `json:"severity"`
	Type        string `json:"type"`
	Message     string `json:"message"`
	Description string `json:"description"`
	ActionRequired bool `json:"action_required"`
}

// Recommendation represents a clinical recommendation
type Recommendation struct {
	RecommendationID string `json:"recommendation_id"`
	Type             string `json:"type"`
	Priority         string `json:"priority"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	Rationale        string `json:"rationale"`
	ActionItems      []string `json:"action_items"`
}

// ProcessingMetadata contains metadata about the processing
type ProcessingMetadata struct {
	FallbackUsed     bool   `json:"fallback_used"`
	CacheUsed        bool   `json:"cache_used"`
	ContextSources   []string `json:"context_sources"`
	ProcessingStages []ProcessingStage `json:"processing_stages"`
}

// ProcessingStage represents a stage in the processing pipeline
type ProcessingStage struct {
	StageName       string        `json:"stage_name"`
	ExecutionTimeMs int64         `json:"execution_time_ms"`
	Status          string        `json:"status"`
	Details         map[string]interface{} `json:"details"`
}

// MedicationIntelligenceRequest represents a medication intelligence request
type MedicationIntelligenceRequest struct {
	RequestID            string                 `json:"request_id"`
	PatientID            string                 `json:"patient_id" binding:"required"`
	Medications          []Medication           `json:"medications" binding:"required"`
	IntelligenceType     string                 `json:"intelligence_type"`
	AnalysisDepth        string                 `json:"analysis_depth"`
	IncludePredictions   bool                   `json:"include_predictions"`
	IncludeAlternatives  bool                   `json:"include_alternatives"`
	ClinicalContext      map[string]interface{} `json:"clinical_context"`
}

// MedicationIntelligenceResponse represents a medication intelligence response
type MedicationIntelligenceResponse struct {
	RequestID                   string                 `json:"request_id"`
	IntelligenceScore           float64                `json:"intelligence_score"`
	MedicationAnalysis          map[string]interface{} `json:"medication_analysis"`
	InteractionAnalysis         map[string]interface{} `json:"interaction_analysis"`
	OutcomePredictions          map[string]interface{} `json:"outcome_predictions"`
	AlternativeRecommendations  []AlternativeMedication `json:"alternative_recommendations"`
	ClinicalInsights            []ClinicalInsight      `json:"clinical_insights"`
	ExecutionTimeMs             int64                  `json:"execution_time_ms"`
}

// DoseOptimizationRequest represents a dose optimization request
type DoseOptimizationRequest struct {
	RequestID          string                 `json:"request_id"`
	PatientID          string                 `json:"patient_id" binding:"required"`
	MedicationCode     string                 `json:"medication_code" binding:"required"`
	ClinicalParameters map[string]interface{} `json:"clinical_parameters"`
	OptimizationGoals  []string               `json:"optimization_goals"`
}

// DoseOptimizationResponse represents a dose optimization response
type DoseOptimizationResponse struct {
	RequestID                    string                 `json:"request_id"`
	OptimizedDose                float64                `json:"optimized_dose"`
	OptimizationScore            float64                `json:"optimization_score"`
	ConfidenceInterval           ConfidenceInterval     `json:"confidence_interval"`
	PharmacokineticPredictions   map[string]interface{} `json:"pharmacokinetic_predictions"`
	MonitoringRecommendations    []MonitoringRecommendation `json:"monitoring_recommendations"`
	ClinicalRationale            string                 `json:"clinical_rationale"`
	ExecutionTimeMs              int64                  `json:"execution_time_ms"`
}

// SafetyValidationRequest represents a safety validation request
type SafetyValidationRequest struct {
	RequestID       string                 `json:"request_id"`
	PatientID       string                 `json:"patient_id" binding:"required"`
	Medications     []Medication           `json:"medications" binding:"required"`
	ClinicalContext map[string]interface{} `json:"clinical_context"`
	ValidationLevel string                 `json:"validation_level"`
}

// SafetyValidationResponse represents a safety validation response
type SafetyValidationResponse struct {
	RequestID                 string                 `json:"request_id"`
	OverallSafetyStatus       string                 `json:"overall_safety_status"`
	DrugInteractions          []DrugInteraction      `json:"drug_interactions"`
	AllergyAlerts             []AllergyAlert         `json:"allergy_alerts"`
	ContraindicationAlerts    []ContraindicationAlert `json:"contraindication_alerts"`
	DosingAlerts              []DosingAlert          `json:"dosing_alerts"`
	MonitoringRequirements    []MonitoringRequirement `json:"monitoring_requirements"`
	SafetyScore               float64                `json:"safety_score"`
	ExecutionTimeMs           int64                  `json:"execution_time_ms"`
}

// Supporting types
type Medication struct {
	Code        string                 `json:"code"`
	Name        string                 `json:"name"`
	Dose        float64                `json:"dose"`
	Unit        string                 `json:"unit"`
	Frequency   string                 `json:"frequency"`
	Route       string                 `json:"route"`
	Duration    string                 `json:"duration"`
	Indication  string                 `json:"indication"`
	Properties  map[string]interface{} `json:"properties"`
}

type AlternativeMedication struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Rationale      string  `json:"rationale"`
	AdvantageScore float64 `json:"advantage_score"`
	CostComparison string  `json:"cost_comparison"`
}

type ClinicalInsight struct {
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Evidence    string `json:"evidence"`
}

type ConfidenceInterval struct {
	Lower      float64 `json:"lower"`
	Upper      float64 `json:"upper"`
	Confidence float64 `json:"confidence"`
}

type MonitoringRecommendation struct {
	Parameter   string `json:"parameter"`
	Frequency   string `json:"frequency"`
	Target      string `json:"target"`
	Rationale   string `json:"rationale"`
}

type DrugInteraction struct {
	Drug1       string `json:"drug1"`
	Drug2       string `json:"drug2"`
	Severity    string `json:"severity"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Management  string `json:"management"`
}

type AllergyAlert struct {
	Allergen    string `json:"allergen"`
	Medication  string `json:"medication"`
	Severity    string `json:"severity"`
	Reaction    string `json:"reaction"`
	Confidence  float64 `json:"confidence"`
}

type ContraindicationAlert struct {
	Medication    string `json:"medication"`
	Condition     string `json:"condition"`
	Severity      string `json:"severity"`
	Rationale     string `json:"rationale"`
	Alternatives  []string `json:"alternatives"`
}

type DosingAlert struct {
	Medication      string `json:"medication"`
	AlertType       string `json:"alert_type"`
	CurrentDose     float64 `json:"current_dose"`
	RecommendedDose float64 `json:"recommended_dose"`
	Rationale       string `json:"rationale"`
}

type MonitoringRequirement struct {
	Parameter     string `json:"parameter"`
	Frequency     string `json:"frequency"`
	Target        string `json:"target"`
	Rationale     string `json:"rationale"`
	Priority      string `json:"priority"`
}

// ClinicalIntelligenceRequest represents a clinical intelligence request
type ClinicalIntelligenceRequest struct {
	RequestID   string `json:"request_id"`
	PatientID   string `json:"patient_id" binding:"required"`
	RequestType string `json:"request_type"`
}

// ==================== ENHANCED PROPOSAL GENERATION MODELS ====================

// EnhancedProposedOrder represents the comprehensive clinical recommendation structure
type EnhancedProposedOrder struct {
	ProposalID      string                    `json:"proposalId"`
	ProposalVersion string                    `json:"proposalVersion"`
	Timestamp       time.Time                 `json:"timestamp"`
	ExpiresAt       time.Time                 `json:"expiresAt"`
	Metadata        ProposalMetadata          `json:"metadata"`
	CalculatedOrder CalculatedOrder           `json:"calculatedOrder"`
	MonitoringPlan  EnhancedMonitoringPlan    `json:"monitoringPlan"`
	TherapeuticAlternatives TherapeuticAlternatives `json:"therapeuticAlternatives"`
	ClinicalRationale ClinicalRationale       `json:"clinicalRationale"`
	ProposalMetadata ProposalMetadataSection  `json:"proposalMetadata"`
}

// ProposalMetadata contains core proposal metadata
type ProposalMetadata struct {
	PatientID            string  `json:"patientId"`
	EncounterID          string  `json:"encounterId"`
	PrescriberID         string  `json:"prescriberId"`
	Status               string  `json:"status"`
	Urgency              string  `json:"urgency"`
	ProposalType         string  `json:"proposalType"`
	RecipeUsed           string  `json:"recipeUsed"`
	ContextCompleteness  float64 `json:"contextCompleteness"`
	ConfidenceScore      float64 `json:"confidenceScore"`
}

// CalculatedOrder contains the enhanced calculated medication order
type CalculatedOrder struct {
	Medication        MedicationDetail        `json:"medication"`
	Dosing            DosingDetail            `json:"dosing"`
	CalculationDetails CalculationDetails     `json:"calculationDetails"`
	Formulation       FormulationDetail       `json:"formulation"`
}

// MedicationDetail contains comprehensive medication information
type MedicationDetail struct {
	PrimaryIdentifier    Identifier   `json:"primaryIdentifier"`
	AlternateIdentifiers []Identifier `json:"alternateIdentifiers"`
	BrandName            *string      `json:"brandName"`
	GenericName          string       `json:"genericName"`
	TherapeuticClass     string       `json:"therapeuticClass"`
	IsHighAlert          bool         `json:"isHighAlert"`
	IsControlled         bool         `json:"isControlled"`
}

// Identifier represents a medication identifier
type Identifier struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display"`
}

// DosingDetail contains comprehensive dosing information
type DosingDetail struct {
	Dose         DoseInfo         `json:"dose"`
	Route        RouteInfo        `json:"route"`
	Frequency    FrequencyInfo    `json:"frequency"`
	Duration     DurationInfo     `json:"duration"`
	Instructions InstructionInfo  `json:"instructions"`
}

// DoseInfo contains dose details
type DoseInfo struct {
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
	PerDose bool    `json:"perDose"`
}

// RouteInfo contains route information
type RouteInfo struct {
	Code    string `json:"code"`
	Display string `json:"display"`
}

// FrequencyInfo contains frequency details
type FrequencyInfo struct {
	Code          string   `json:"code"`
	Display       string   `json:"display"`
	TimesPerDay   int      `json:"timesPerDay"`
	SpecificTimes []string `json:"specificTimes"`
}

// DurationInfo contains duration details
type DurationInfo struct {
	Value   int    `json:"value"`
	Unit    string `json:"unit"`
	Refills int    `json:"refills"`
}

// InstructionInfo contains patient and pharmacy instructions
type InstructionInfo struct {
	PatientInstructions     string   `json:"patientInstructions"`
	PharmacyInstructions    string   `json:"pharmacyInstructions"`
	AdditionalInstructions  []string `json:"additionalInstructions"`
}

// CalculationDetails contains dose calculation methodology and factors
type CalculationDetails struct {
	Method           string                 `json:"method"`
	Factors          CalculationFactors     `json:"factors"`
	Adjustments      []string               `json:"adjustments"`
	RoundingApplied  bool                   `json:"roundingApplied"`
	MaximumDoseCheck MaximumDoseCheck       `json:"maximumDoseCheck"`
}

// CalculationFactors contains patient factors used in calculation
type CalculationFactors struct {
	PatientWeight  float64      `json:"patientWeight"`
	PatientAge     int          `json:"patientAge"`
	RenalFunction  RenalFunction `json:"renalFunction"`
}

// RenalFunction contains renal function details
type RenalFunction struct {
	EGFR     float64 `json:"eGFR"`
	Category string  `json:"category"`
}

// MaximumDoseCheck contains dose limit validation
type MaximumDoseCheck struct {
	Daily        float64 `json:"daily"`
	Maximum      float64 `json:"maximum"`
	WithinLimits bool    `json:"withinLimits"`
}

// FormulationDetail contains formulation information and alternatives
type FormulationDetail struct {
	SelectedForm             string                    `json:"selectedForm"`
	AvailableStrengths       []float64                 `json:"availableStrengths"`
	Splittable               bool                      `json:"splittable"`
	Crushable                bool                      `json:"crushable"`
	AlternativeFormulations  []AlternativeFormulation `json:"alternativeFormulations"`
}

// AlternativeFormulation represents alternative drug formulations
type AlternativeFormulation struct {
	Form         string    `json:"form"`
	Strengths    []float64 `json:"strengths"`
	ClinicalNote string    `json:"clinicalNote"`
}

// EnhancedMonitoringPlan contains comprehensive monitoring requirements
type EnhancedMonitoringPlan struct {
	RiskStratification RiskStratification    `json:"riskStratification"`
	Baseline          []BaselineMonitoring  `json:"baseline"`
	Ongoing           []OngoingMonitoring   `json:"ongoing"`
	SymptomMonitoring []SymptomMonitoring   `json:"symptomMonitoring"`
}

// RiskStratification contains patient risk assessment
type RiskStratification struct {
	OverallRisk string       `json:"overallRisk"`
	Factors     []RiskFactor `json:"factors"`
}

// RiskFactor represents individual risk factors
type RiskFactor struct {
	Factor  string `json:"factor"`
	Present bool   `json:"present"`
	Impact  string `json:"impact"`
}

// BaselineMonitoring contains baseline monitoring requirements
type BaselineMonitoring struct {
	Parameter      string         `json:"parameter"`
	LOINC          string         `json:"loinc"`
	Timing         string         `json:"timing"`
	Priority       string         `json:"priority"`
	Rationale      string         `json:"rationale"`
	CriticalValues CriticalValues `json:"criticalValues"`
}

// CriticalValues contains critical value thresholds
type CriticalValues struct {
	Contraindicated string `json:"contraindicated"`
	CautionRequired string `json:"cautionRequired"`
	Normal          string `json:"normal"`
}

// OngoingMonitoring contains ongoing monitoring requirements
type OngoingMonitoring struct {
	Parameter        string             `json:"parameter"`
	Frequency        MonitoringFrequency `json:"frequency"`
	Rationale        string             `json:"rationale"`
	ActionThresholds []ActionThreshold  `json:"actionThresholds"`
	TargetRange      *TargetRange       `json:"targetRange,omitempty"`
}

// MonitoringFrequency contains monitoring frequency details
type MonitoringFrequency struct {
	Interval   int                      `json:"interval"`
	Unit       string                   `json:"unit"`
	Conditions []FrequencyCondition     `json:"conditions"`
}

// FrequencyCondition contains conditional frequency modifications
type FrequencyCondition struct {
	Condition         string              `json:"condition"`
	ModifiedFrequency MonitoringFrequency `json:"modifiedFrequency"`
}

// ActionThreshold contains action thresholds for monitoring
type ActionThreshold struct {
	Value   string `json:"value"`
	Action  string `json:"action"`
	Urgency string `json:"urgency"`
}

// TargetRange contains target value ranges
type TargetRange struct {
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Unit string  `json:"unit"`
}

// SymptomMonitoring contains symptom monitoring requirements
type SymptomMonitoring struct {
	Symptom           string   `json:"symptom"`
	Frequency         string   `json:"frequency"`
	EducationProvided string   `json:"educationProvided"`
	RedFlags          []string `json:"redFlags,omitempty"`
}

// TherapeuticAlternatives contains alternative medication options
type TherapeuticAlternatives struct {
	PrimaryReason        string                    `json:"primaryReason"`
	Alternatives         []TherapeuticAlternative  `json:"alternatives"`
	NonPharmAlternatives []NonPharmAlternative     `json:"nonPharmAlternatives"`
}

// TherapeuticAlternative represents an alternative medication
type TherapeuticAlternative struct {
	Medication              AlternativeMedicationDetail `json:"medication"`
	Category                string                      `json:"category"`
	FormularyStatus         FormularyStatus             `json:"formularyStatus"`
	CostComparison          CostComparison              `json:"costComparison"`
	ClinicalConsiderations  ClinicalConsiderations      `json:"clinicalConsiderations"`
	SwitchingInstructions   string                      `json:"switchingInstructions"`
	Evidence                *AlternativeEvidence        `json:"evidence,omitempty"`
}

// AlternativeMedicationDetail contains alternative medication details
type AlternativeMedicationDetail struct {
	Name     string  `json:"name"`
	Code     string  `json:"code"`
	Strength float64 `json:"strength"`
	Unit     string  `json:"unit"`
}

// FormularyStatus contains formulary information
type FormularyStatus struct {
	Tier              int     `json:"tier"`
	PriorAuthRequired bool    `json:"priorAuthRequired"`
	QuantityLimits    *string `json:"quantityLimits"`
}

// CostComparison contains cost comparison information
type CostComparison struct {
	RelativeCost         string  `json:"relativeCost"`
	EstimatedMonthlyCost float64 `json:"estimatedMonthlyCost"`
	PatientCopay         float64 `json:"patientCopay"`
}

// ClinicalConsiderations contains clinical pros and cons
type ClinicalConsiderations struct {
	Advantages        []string `json:"advantages"`
	Disadvantages     []string `json:"disadvantages"`
	Contraindications []string `json:"contraindications,omitempty"`
}

// AlternativeEvidence contains evidence for alternatives
type AlternativeEvidence struct {
	ComparativeEffectiveness string   `json:"comparativeEffectiveness"`
	GuidelinePosition        string   `json:"guidelinePosition"`
	References               []string `json:"references"`
}

// NonPharmAlternative represents non-pharmacological alternatives
type NonPharmAlternative struct {
	Intervention   string   `json:"intervention"`
	Components     []string `json:"components"`
	Effectiveness  string   `json:"effectiveness"`
	Recommendation string   `json:"recommendation"`
}

// ClinicalRationale contains comprehensive clinical reasoning
type ClinicalRationale struct {
	Summary               RationaleSummary        `json:"summary"`
	IndicationAssessment  IndicationAssessment    `json:"indicationAssessment"`
	DosingRationale       DosingRationale         `json:"dosingRationale"`
	FormularyRationale    FormularyRationale      `json:"formularyRationale"`
	PatientFactors        PatientFactors          `json:"patientFactors"`
	QualityMeasures       QualityMeasures         `json:"qualityMeasures"`
}

// RationaleSummary contains high-level decision summary
type RationaleSummary struct {
	Decision   string `json:"decision"`
	Confidence string `json:"confidence"`
	Complexity string `json:"complexity"`
}

// IndicationAssessment contains indication validation
type IndicationAssessment struct {
	PrimaryIndication string             `json:"primaryIndication"`
	ICDCode           string             `json:"icdCode"`
	ClinicalCriteria  []ClinicalCriterion `json:"clinicalCriteria"`
	Appropriateness   string             `json:"appropriateness"`
}

// ClinicalCriterion represents clinical criteria assessment
type ClinicalCriterion struct {
	Criterion string  `json:"criterion"`
	Met       bool    `json:"met"`
	Value     *string `json:"value,omitempty"`
}

// DosingRationale contains dosing decision rationale
type DosingRationale struct {
	Strategy     string         `json:"strategy"`
	Explanation  string         `json:"explanation"`
	TitrationPlan TitrationPlan  `json:"titrationPlan"`
	EvidenceBase EvidenceBase   `json:"evidenceBase"`
}

// TitrationPlan contains titration schedule
type TitrationPlan struct {
	Week2   string `json:"week2"`
	Week4   string `json:"week4"`
	MaxDose string `json:"maxDose"`
}

// EvidenceBase contains evidence information
type EvidenceBase struct {
	Source                 string `json:"source"`
	RecommendationStrength string `json:"recommendationStrength"`
	EvidenceQuality        string `json:"evidenceQuality"`
}

// FormularyRationale contains formulary decision rationale
type FormularyRationale struct {
	FormularyDecision   string            `json:"formularyDecision"`
	CostEffectiveness   string            `json:"costEffectiveness"`
	InsuranceCoverage   InsuranceCoverage `json:"insuranceCoverage"`
}

// InsuranceCoverage contains insurance coverage details
type InsuranceCoverage struct {
	Covered           bool    `json:"covered"`
	Tier              int     `json:"tier"`
	Copay             float64 `json:"copay"`
	DeductibleApplies bool    `json:"deductibleApplies"`
}

// PatientFactors contains patient-specific considerations
type PatientFactors struct {
	PositiveFactors      []string              `json:"positiveFactors"`
	Considerations       []string              `json:"considerations"`
	SharedDecisionMaking SharedDecisionMaking  `json:"sharedDecisionMaking"`
}

// SharedDecisionMaking contains shared decision making details
type SharedDecisionMaking struct {
	Discussed         []string `json:"discussed"`
	PatientPreference string   `json:"patientPreference"`
}

// QualityMeasures contains quality measure alignment
type QualityMeasures struct {
	AlignedMeasures []QualityMeasure `json:"alignedMeasures"`
}

// QualityMeasure represents a quality measure
type QualityMeasure struct {
	Measure   string `json:"measure"`
	NQFNumber string `json:"nqfNumber"`
	Impact    string `json:"impact"`
}

// ProposalMetadataSection contains additional proposal metadata
type ProposalMetadataSection struct {
	ClinicalReferences []ClinicalReference `json:"clinicalReferences"`
	AuditTrail         AuditTrail          `json:"auditTrail"`
	NextSteps          []NextStep          `json:"nextSteps"`
}

// ClinicalReference contains clinical reference information
type ClinicalReference struct {
	Type     string `json:"type"`
	Citation string `json:"citation"`
	URL      string `json:"url"`
}

// AuditTrail contains processing audit information
type AuditTrail struct {
	CalculationTime    int64            `json:"calculationTime"`
	ContextFetchTime   int64            `json:"contextFetchTime"`
	TotalProcessingTime int64           `json:"totalProcessingTime"`
	CacheUtilization   CacheUtilization `json:"cacheUtilization"`
}

// CacheUtilization contains cache hit/miss information
type CacheUtilization struct {
	FormularyCache        string `json:"formularyCache"`
	DoseCalculationCache  string `json:"doseCalculationCache"`
	MonitoringCache       string `json:"monitoringCache"`
}

// NextStep contains workflow next steps
type NextStep struct {
	Step           string   `json:"step"`
	Service        string   `json:"service"`
	Optional       bool     `json:"optional"`
	Reason         string   `json:"reason"`
	RequiredChecks []string `json:"requiredChecks,omitempty"`
}
