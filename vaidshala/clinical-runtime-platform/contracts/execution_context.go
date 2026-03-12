// Package contracts defines the FROZEN clinical execution contract.
//
// GOVERNANCE: This file is FROZEN. Changes require CTO + CMO sign-off.
// All engines MUST consume this contract. No exceptions.
//
// The ClinicalExecutionContext is the ONE contract that all engines see.
// It represents a complete, pre-assembled snapshot of:
//   - Patient clinical context (KB-2A assembly + KB-2B intelligence)
//   - Pre-computed knowledge answers (KB-7, KB-4, KB-5, KB-6, KB-8, KB-11)
//   - Runtime metadata (who, when, where)
//
// CRITICAL DESIGN PRINCIPLES:
//   1. Engines see ANSWERS, not questions - no KB calls at execution time
//   2. Snapshot is IMMUTABLE - built once, read-only thereafter
//   3. Pure PostgreSQL at runtime - Neo4j only at build/materialize time
//   4. All engines are stateless - they receive context, return results
package contracts

import (
	"time"
)

// ============================================================================
// FROZEN CONTRACT: ClinicalExecutionContext
// ============================================================================

// ClinicalExecutionContext is the SINGLE contract all engines receive.
// It contains everything needed for deterministic clinical decision making.
//
// This struct is IMMUTABLE after construction. Engines MUST NOT modify it.
// Instead, engines return their results which are collected by the orchestrator.
type ClinicalExecutionContext struct {
	// Patient contains assembled patient data (KB-2A) enriched with
	// clinical intelligence (KB-2B risk scores, summaries, flags)
	Patient PatientContext `json:"patient"`

	// Knowledge contains pre-answered results from all Knowledge Bases.
	// Built ONCE by BuildKnowledgeSnapshot(), read-only thereafter.
	// Engines query this snapshot - they NEVER call KBs directly.
	Knowledge KnowledgeSnapshot `json:"knowledge"`

	// Runtime contains execution metadata: who requested, when, where.
	// Used for audit trails and evidence envelopes.
	Runtime ExecutionMetadata `json:"runtime"`
}

// ============================================================================
// PATIENT CONTEXT (KB-2A Assembly + KB-2B Intelligence)
// ============================================================================

// PatientContext contains the assembled patient clinical picture.
// KB-2A provides raw assembly, KB-2B adds intelligence.
type PatientContext struct {
	// ============ KB-2A: Raw Patient Assembly (NO intelligence) ============

	// Demographics from FHIR Patient resource
	Demographics PatientDemographics `json:"demographics"`

	// ActiveConditions normalized from FHIR Condition resources
	// Only clinically-active conditions (not resolved, not refuted)
	ActiveConditions []ClinicalCondition `json:"activeConditions"`

	// ActiveMedications from FHIR MedicationRequest/MedicationStatement
	// Only active prescriptions (not stopped, not cancelled)
	ActiveMedications []Medication `json:"activeMedications"`

	// RecentLabResults from FHIR Observation (laboratory category)
	// Typically last 90 days, configurable per use case
	RecentLabResults []LabResult `json:"recentLabResults"`

	// RecentVitalSigns from FHIR Observation (vital-signs category)
	// Typically last 30 days, configurable per use case
	RecentVitalSigns []VitalSign `json:"recentVitalSigns"`

	// RecentEncounters from FHIR Encounter resources
	// Used for measure period calculations
	RecentEncounters []Encounter `json:"recentEncounters"`

	// Allergies from FHIR AllergyIntolerance resources
	Allergies []Allergy `json:"allergies"`

	// ============ KB-2B: Clinical Intelligence (enriched data) ============

	// RiskProfile contains KB-2B computed risk scores and flags
	RiskProfile RiskProfile `json:"riskProfile"`

	// ClinicalSummary contains KB-2B generated narrative summaries
	ClinicalSummary ClinicalSummary `json:"clinicalSummary"`

	// CQLExportBundle contains KB-2B formatted data for CQL execution
	// This is the exact format the CQL executor expects
	CQLExportBundle *CQLExportBundle `json:"cqlExportBundle,omitempty"`
}

// PatientDemographics contains core patient identity information.
type PatientDemographics struct {
	// PatientID is the FHIR Patient.id (anonymized for audit)
	PatientID string `json:"patientId"`

	// BirthDate for age calculations
	BirthDate *time.Time `json:"birthDate,omitempty"`

	// Gender as FHIR administrative gender code
	Gender string `json:"gender,omitempty"`

	// Region determines which regional adapter to apply (IN, AU, etc.)
	Region string `json:"region"`
}

// ClinicalCondition represents a normalized active condition.
type ClinicalCondition struct {
	// Code is the primary clinical code (SNOMED, ICD-10)
	Code ClinicalCode `json:"code"`

	// OnsetDate when the condition started
	OnsetDate *time.Time `json:"onsetDate,omitempty"`

	// ClinicalStatus must be "active" (filtered by KB-2A)
	ClinicalStatus string `json:"clinicalStatus"`

	// VerificationStatus (confirmed, provisional, differential)
	VerificationStatus string `json:"verificationStatus,omitempty"`

	// Severity if coded (mild, moderate, severe)
	Severity string `json:"severity,omitempty"`

	// SourceReference to original FHIR resource
	SourceReference string `json:"sourceReference"`
}

// Medication represents an active medication order.
type Medication struct {
	// Code is the medication code (RxNorm, AMT, NLEM)
	Code ClinicalCode `json:"code"`

	// Dosage information
	Dosage *Dosage `json:"dosage,omitempty"`

	// Status must be "active" (filtered by KB-2A)
	Status string `json:"status"`

	// AuthoredOn when the prescription was written
	AuthoredOn *time.Time `json:"authoredOn,omitempty"`

	// SourceReference to original FHIR resource
	SourceReference string `json:"sourceReference"`
}

// Dosage contains medication dosage details.
type Dosage struct {
	// Text is the human-readable dosage instruction
	Text string `json:"text,omitempty"`

	// DoseQuantity with value and unit
	DoseQuantity *Quantity `json:"doseQuantity,omitempty"`

	// Frequency (e.g., "once daily", "BID")
	Frequency string `json:"frequency,omitempty"`

	// Route of administration
	Route string `json:"route,omitempty"`
}

// LabResult represents a laboratory observation result.
type LabResult struct {
	// Code is the lab test code (LOINC)
	Code ClinicalCode `json:"code"`

	// Value is the result value
	Value *Quantity `json:"value,omitempty"`

	// ValueString for non-numeric results
	ValueString string `json:"valueString,omitempty"`

	// ReferenceRange for interpretation
	ReferenceRange *ReferenceRange `json:"referenceRange,omitempty"`

	// Interpretation (normal, high, low, critical)
	Interpretation string `json:"interpretation,omitempty"`

	// EffectiveDateTime when the specimen was collected
	EffectiveDateTime *time.Time `json:"effectiveDateTime,omitempty"`

	// SourceReference to original FHIR resource
	SourceReference string `json:"sourceReference"`
}

// VitalSign represents a vital sign observation.
type VitalSign struct {
	// Code is the vital sign code (LOINC)
	Code ClinicalCode `json:"code"`

	// Value is the measurement value
	Value *Quantity `json:"value,omitempty"`

	// ComponentValues for composite vitals (e.g., BP systolic/diastolic)
	ComponentValues []ComponentValue `json:"componentValues,omitempty"`

	// EffectiveDateTime when measured
	EffectiveDateTime *time.Time `json:"effectiveDateTime,omitempty"`

	// SourceReference to original FHIR resource
	SourceReference string `json:"sourceReference"`
}

// ComponentValue represents a component of a composite observation.
type ComponentValue struct {
	Code  ClinicalCode `json:"code"`
	Value *Quantity    `json:"value,omitempty"`
}

// Encounter represents a clinical encounter.
type Encounter struct {
	// EncounterID is the FHIR Encounter.id
	EncounterID string `json:"encounterId"`

	// Type codes for the encounter
	Type []ClinicalCode `json:"type,omitempty"`

	// Class (ambulatory, inpatient, emergency)
	Class string `json:"class,omitempty"`

	// Period of the encounter
	Period *Period `json:"period,omitempty"`

	// Status (finished, in-progress)
	Status string `json:"status"`

	// SourceReference to original FHIR resource
	SourceReference string `json:"sourceReference"`
}

// Allergy represents an allergy or intolerance.
type Allergy struct {
	// Code of the allergen
	Code ClinicalCode `json:"code"`

	// Category (food, medication, environment)
	Category string `json:"category,omitempty"`

	// Criticality (low, high, unable-to-assess)
	Criticality string `json:"criticality,omitempty"`

	// ClinicalStatus (active, inactive, resolved)
	ClinicalStatus string `json:"clinicalStatus"`

	// SourceReference to original FHIR resource
	SourceReference string `json:"sourceReference"`
}

// ============================================================================
// KB-2B: CLINICAL INTELLIGENCE
// ============================================================================

// RiskProfile contains KB-2B computed risk assessments.
type RiskProfile struct {
	// ComputedRisks are named risk scores computed by KB-2B
	ComputedRisks map[string]RiskScore `json:"computedRisks,omitempty"`

	// ClinicalFlags are boolean alerts (e.g., "is_diabetic", "is_sepsis_risk")
	ClinicalFlags map[string]bool `json:"clinicalFlags,omitempty"`

	// ComputedAt when the risk profile was generated
	ComputedAt time.Time `json:"computedAt"`
}

// RiskScore represents a single computed risk score.
type RiskScore struct {
	// Name of the risk score (e.g., "ASCVD", "SOFA", "qSOFA")
	Name string `json:"name"`

	// Value is the numeric score
	Value float64 `json:"value"`

	// Category interpretation (low, moderate, high, critical)
	Category string `json:"category,omitempty"`

	// Confidence level if applicable
	Confidence float64 `json:"confidence,omitempty"`

	// ComponentScores if the score is composite
	ComponentScores map[string]float64 `json:"componentScores,omitempty"`
}

// ClinicalSummary contains KB-2B generated summaries.
type ClinicalSummary struct {
	// ProblemList is a prioritized list of active problems
	ProblemList []string `json:"problemList,omitempty"`

	// MedicationSummary is a narrative medication reconciliation
	MedicationSummary string `json:"medicationSummary,omitempty"`

	// CareGaps are identified gaps in recommended care
	CareGaps []CareGap `json:"careGaps,omitempty"`

	// GeneratedAt when the summary was produced
	GeneratedAt time.Time `json:"generatedAt"`
}

// CareGap represents an identified gap in patient care.
type CareGap struct {
	// MeasureID the gap relates to (e.g., "CMS122")
	MeasureID string `json:"measureId"`

	// Description of the care gap
	Description string `json:"description"`

	// Priority (high, medium, low)
	Priority string `json:"priority,omitempty"`

	// RecommendedAction to close the gap
	RecommendedAction string `json:"recommendedAction,omitempty"`
}

// CQLExportBundle contains data formatted for CQL execution.
// This matches the FHIR Bundle structure expected by HAPI CQL.
type CQLExportBundle struct {
	// ResourceType is always "Bundle"
	ResourceType string `json:"resourceType"`

	// Type is "collection" for CQL execution
	Type string `json:"type"`

	// Entry contains FHIR resources for CQL
	Entry []interface{} `json:"entry"`
}

// ============================================================================
// KNOWLEDGE SNAPSHOT (Pre-answered KB results)
// ============================================================================

// KnowledgeSnapshot contains all pre-computed knowledge base answers.
// Built ONCE by BuildKnowledgeSnapshot(), engines read from here.
// NO engine should call any KB directly - all answers are in this snapshot.
//
// STRUCTURE (per CTO/CMO spec):
// - Terminology (KB-7): Pre-resolved codes and ValueSet memberships
// - Calculators (KB-8): Pre-computed clinical calculations
// - Safety (KB-4): Allergies, contraindications, pregnancy status
// - Interactions (KB-5): Drug-drug interactions (current and potential)
// - Formulary (KB-6): Formulary status, prior auth, alternatives
// - Dosing (KB-1): Renal/hepatic/weight-based dose adjustments
// - CDI (KB-11): Clinical documentation intelligence
// - LabInterpretation (KB-16): Reference ranges and lab interpretations
type KnowledgeSnapshot struct {
	// Terminology contains KB-7 pre-resolved terminology
	Terminology TerminologySnapshot `json:"terminology"`

	// Calculators contains KB-8 pre-computed clinical calculations
	Calculators CalculatorSnapshot `json:"calculators"`

	// Safety contains KB-4 patient safety information
	Safety SafetySnapshot `json:"safety"`

	// Interactions contains KB-5 drug interaction analysis
	Interactions InteractionSnapshot `json:"interactions"`

	// Formulary contains KB-6 formulary status and alternatives
	Formulary FormularySnapshot `json:"formulary"`

	// Dosing contains KB-1 dose adjustment information
	Dosing DosingSnapshot `json:"dosing"`

	// CDI contains KB-11 clinical documentation facts
	CDI CDIFacts `json:"cdi"`

	// LabInterpretation contains KB-16 lab reference ranges and interpretations
	LabInterpretation LabInterpretationSnapshot `json:"labInterpretation"`

	// SnapshotTimestamp when the snapshot was constructed
	SnapshotTimestamp time.Time `json:"snapshotTimestamp"`

	// SnapshotVersion version identifier for the snapshot
	SnapshotVersion string `json:"snapshotVersion"`

	// KBVersions tracks which KB versions were used
	KBVersions map[string]string `json:"kbVersions"`
}

// ============================================================================
// KB-7: TERMINOLOGY SNAPSHOT
// ============================================================================

// TerminologySnapshot contains KB-7 pre-resolved terminology (per CTO/CMO spec).
type TerminologySnapshot struct {
	// PatientConditionCodes are SNOMED codes with display names for patient conditions
	PatientConditionCodes []ResolvedCode `json:"patientConditionCodes,omitempty"`

	// PatientMedicationCodes are RxNorm codes with display names for patient medications
	PatientMedicationCodes []ResolvedCode `json:"patientMedicationCodes,omitempty"`

	// ValueSetMemberships answers "Is patient diabetic?" → true
	// Key: ValueSet name (e.g., "Diabetes", "Essential Hypertension")
	// Value: true if patient has condition in that ValueSet
	ValueSetMemberships map[string]bool `json:"valueSetMemberships"`

	// ExpandedValueSets contains full ValueSet expansions for CQL
	// Key: ValueSet name
	// Value: All codes in that ValueSet
	ExpandedValueSets map[string][]ClinicalCode `json:"expandedValueSets,omitempty"`

	// CodeMemberships maps code -> ValueSet memberships for lookups
	// Key: "system|code"
	// Value: List of ValueSet names containing this code
	CodeMemberships map[string][]string `json:"codeMemberships,omitempty"`
}

// ResolvedCode is a clinical code with resolved display name.
type ResolvedCode struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display"`
}

// ============================================================================
// KB-8: CALCULATOR SNAPSHOT
// ============================================================================

// CalculatorSnapshot contains KB-8 pre-computed calculations (per CTO/CMO spec).
type CalculatorSnapshot struct {
	// eGFR - Estimated Glomerular Filtration Rate
	EGFR *CalculationResult `json:"eGFR,omitempty"`

	// ASCVD10Year - 10-year Atherosclerotic Cardiovascular Disease Risk
	ASCVD10Year *CalculationResult `json:"ascvd10Year,omitempty"`

	// CHA2DS2VASc - Stroke risk in atrial fibrillation
	CHA2DS2VASc *CalculationResult `json:"cha2ds2Vasc,omitempty"`

	// HASBLED - Bleeding risk score
	HASBLED *CalculationResult `json:"hasBled,omitempty"`

	// BMI - Body Mass Index (with Asian interpretation if applicable)
	BMI *CalculationResult `json:"bmi,omitempty"`

	// ChildPugh - Liver function classification (nil if no liver disease)
	ChildPugh *CalculationResult `json:"childPugh,omitempty"`

	// MELD - Model for End-Stage Liver Disease (nil if no liver disease)
	MELD *CalculationResult `json:"meld,omitempty"`

	// SOFA - Sequential Organ Failure Assessment (for sepsis)
	SOFA *CalculationResult `json:"sofa,omitempty"`

	// qSOFA - Quick SOFA Score
	QSOFA *CalculationResult `json:"qsofa,omitempty"`

	// AdditionalCalculations for any other calculators
	AdditionalCalculations map[string]CalculationResult `json:"additionalCalculations,omitempty"`
}

// ============================================================================
// KB-4: SAFETY SNAPSHOT
// ============================================================================

// SafetySnapshot contains KB-4 patient safety information (per CTO/CMO spec).
// Enhanced to include comprehensive safety data from KB-4's full API:
// - Black box warnings (FDA/TGA/EMA)
// - High-alert medications (ISMP)
// - Pregnancy/lactation safety
// - Beers criteria (geriatric)
// - Anticholinergic burden (ACB)
// - Lab monitoring requirements
// - Dose limits
type SafetySnapshot struct {
	// ============================================================================
	// ORIGINAL FIELDS (Backward Compatible)
	// ============================================================================

	// ActiveAllergies from AllergyIntolerance resources
	ActiveAllergies []AllergyInfo `json:"activeAllergies,omitempty"`

	// Contraindications detected drug-condition contraindications
	Contraindications []ContraindicationInfo `json:"contraindications,omitempty"`

	// PregnancyStatus if applicable (nil if not pregnant or unknown)
	PregnancyStatus *PregnancyInfo `json:"pregnancyStatus,omitempty"`

	// RenalDoseAdjustmentNeeded based on eGFR
	RenalDoseAdjustmentNeeded bool `json:"renalDoseAdjustmentNeeded"`

	// HepaticDoseAdjustmentNeeded based on liver function
	HepaticDoseAdjustmentNeeded bool `json:"hepaticDoseAdjustmentNeeded"`

	// SafetyAlerts high-priority warnings
	SafetyAlerts []SafetyAlert `json:"safetyAlerts,omitempty"`

	// ============================================================================
	// ENHANCED FIELDS (KB-4 Full API Integration)
	// ============================================================================

	// BlackBoxWarnings contains FDA/TGA black box warnings for patient's medications.
	// Map key is RxNorm code. These are the most serious safety warnings.
	BlackBoxWarnings map[string]*BlackBoxWarningInfo `json:"blackBoxWarnings,omitempty"`

	// HighAlertMedications contains ISMP high-alert medication flags.
	// Map key is RxNorm code. These require extra safety precautions.
	HighAlertMedications map[string]*HighAlertInfo `json:"highAlertMedications,omitempty"`

	// PregnancySafetyInfo contains enhanced pregnancy safety data.
	// Map key is RxNorm code. Includes FDA PLLR and TGA categories.
	PregnancySafetyInfo map[string]*PregnancySafetyInfo `json:"pregnancySafetyInfo,omitempty"`

	// LactationSafetyInfo contains LactMed lactation safety data.
	// Map key is RxNorm code. Includes milk excretion and infant effects.
	LactationSafetyInfo map[string]*LactationSafetyInfo `json:"lactationSafetyInfo,omitempty"`

	// BeersCriteria contains AGS Beers Criteria entries for geriatric patients.
	// Map key is RxNorm code. Only populated if patient age >= 65.
	BeersCriteria map[string]*BeersEntryInfo `json:"beersCriteria,omitempty"`

	// AnticholinergicBurden contains ACB scores for patient's medications.
	// Map key is RxNorm code.
	AnticholinergicBurden map[string]*AnticholinergicInfo `json:"anticholinergicBurden,omitempty"`

	// TotalACBScore is the cumulative anticholinergic burden score.
	// Risk levels: Low (1-2), Moderate (3-4), High (5+)
	TotalACBScore int `json:"totalAcbScore,omitempty"`

	// ACBRiskLevel is the interpreted risk level from TotalACBScore.
	// Values: "Low", "Moderate", "High", "Very High"
	ACBRiskLevel string `json:"acbRiskLevel,omitempty"`

	// LabRequirements contains required lab monitoring for medications.
	// Map key is RxNorm code. Used to generate monitoring recommendations.
	LabRequirements map[string]*LabRequirementInfo `json:"labRequirements,omitempty"`

	// DoseLimits contains dose limit information for medications.
	// Map key is RxNorm code. Used for dose validation.
	DoseLimits map[string]*DoseLimitInfo `json:"doseLimits,omitempty"`

	// AgeLimits contains age restriction information for medications.
	// Map key is RxNorm code. Used for pediatric/geriatric safety checks.
	AgeLimits map[string]*AgeLimitInfo `json:"ageLimits,omitempty"`

	// HasBlackBoxWarnings quick check flag for any black box warnings
	HasBlackBoxWarnings bool `json:"hasBlackBoxWarnings"`

	// HasHighAlertDrugs quick check flag for any ISMP high-alert medications
	HasHighAlertDrugs bool `json:"hasHighAlertDrugs"`

	// HasBeersWarnings quick check flag for Beers criteria warnings (geriatric)
	HasBeersWarnings bool `json:"hasBeersWarnings"`
}

// ============================================================================
// ENHANCED SAFETY TYPES (KB-4 Full API)
// ============================================================================

// BlackBoxWarningInfo contains FDA/TGA black box warning details.
type BlackBoxWarningInfo struct {
	RxNormCode       string   `json:"rxnormCode"`
	DrugName         string   `json:"drugName"`
	RiskCategories   []string `json:"riskCategories"`
	WarningText      string   `json:"warningText"`
	Severity         string   `json:"severity"` // CRITICAL, HIGH
	HasREMS          bool     `json:"hasRems"`
	REMSProgram      string   `json:"remsProgram,omitempty"`
	REMSRequirements []string `json:"remsRequirements,omitempty"`
	SourceAuthority  string   `json:"sourceAuthority"` // FDA, TGA, EMA
	SourceDocument   string   `json:"sourceDocument,omitempty"`
}

// HighAlertInfo contains ISMP high-alert medication details.
type HighAlertInfo struct {
	RxNormCode             string   `json:"rxnormCode"`
	DrugName               string   `json:"drugName"`
	Category               string   `json:"category"` // ANTICOAGULANTS, INSULIN, OPIOIDS, etc.
	TallManName            string   `json:"tallManName,omitempty"`
	Requirements           []string `json:"requirements"`
	Safeguards             []string `json:"safeguards"`
	DoubleCheckRequired    bool     `json:"doubleCheckRequired"`
	SmartPumpRequired      bool     `json:"smartPumpRequired"`
	IndependentDoubleCheck bool     `json:"independentDoubleCheck,omitempty"`
}

// PregnancySafetyInfo contains comprehensive pregnancy safety details.
type PregnancySafetyInfo struct {
	RxNormCode             string            `json:"rxnormCode"`
	DrugName               string            `json:"drugName"`
	Category               string            `json:"category"` // A, B, C, D, X, N
	PLLRRiskSummary        string            `json:"pllrRiskSummary,omitempty"`
	Teratogenic            bool              `json:"teratogenic"`
	TeratogenicEffects     []string          `json:"teratogenicEffects,omitempty"`
	TrimesterRisks         map[string]string `json:"trimesterRisks,omitempty"`
	Recommendation         string            `json:"recommendation"`
	AlternativeDrugs       []string          `json:"alternativeDrugs,omitempty"`
	MonitoringRequired     []string          `json:"monitoringRequired,omitempty"`
	SourceAuthority        string            `json:"sourceAuthority"` // FDA, TGA
}

// LactationSafetyInfo contains LactMed lactation safety details.
type LactationSafetyInfo struct {
	RxNormCode        string   `json:"rxnormCode"`
	DrugName          string   `json:"drugName"`
	Risk              string   `json:"risk"` // COMPATIBLE, PROBABLY_COMPATIBLE, USE_WITH_CAUTION, CONTRAINDICATED, UNKNOWN
	RiskSummary       string   `json:"riskSummary,omitempty"`
	ExcretedInMilk    bool     `json:"excretedInMilk"`
	MilkPlasmaRatio   string   `json:"milkPlasmaRatio,omitempty"`
	InfantDosePercent float64  `json:"infantDosePercent,omitempty"` // Relative Infant Dose (RID)
	HalfLifeHours     float64  `json:"halfLifeHours"`
	InfantEffects     []string `json:"infantEffects,omitempty"`
	InfantMonitoring  []string `json:"infantMonitoring,omitempty"`
	Recommendation    string   `json:"recommendation"`
	AlternativeDrugs  []string `json:"alternativeDrugs,omitempty"`
	TimingAdvice      string   `json:"timingAdvice,omitempty"`
}

// BeersEntryInfo contains AGS Beers Criteria entry details.
type BeersEntryInfo struct {
	RxNormCode               string   `json:"rxnormCode"`
	DrugName                 string   `json:"drugName"`
	DrugClass                string   `json:"drugClass,omitempty"`
	Recommendation           string   `json:"recommendation"` // AVOID, AVOID_IN_CONDITION, USE_WITH_CAUTION
	Rationale                string   `json:"rationale"`
	QualityOfEvidence        string   `json:"qualityOfEvidence"`        // High, Moderate, Low
	StrengthOfRecommendation string   `json:"strengthOfRecommendation"` // Strong, Weak
	Conditions               []string `json:"conditions,omitempty"`
	ACBScore                 int      `json:"acbScore,omitempty"`
	AlternativeDrugs         []string `json:"alternativeDrugs,omitempty"`
	NonPharmacologic         []string `json:"nonPharmacologic,omitempty"`
	AgeThreshold             int      `json:"ageThreshold,omitempty"` // Default 65
}

// AnticholinergicInfo contains ACB score details.
type AnticholinergicInfo struct {
	RxNormCode        string   `json:"rxnormCode"`
	DrugName          string   `json:"drugName"`
	ACBScore          int      `json:"acbScore"` // 1-3 scale
	RiskLevel         string   `json:"riskLevel"` // Low, Moderate, High
	Effects           []string `json:"effects,omitempty"`
	CognitiveRisk     string   `json:"cognitiveRisk,omitempty"`
	PeripheralEffects []string `json:"peripheralEffects,omitempty"`
}

// LabRequirementInfo contains lab monitoring requirement details.
type LabRequirementInfo struct {
	RxNormCode         string            `json:"rxnormCode"`
	DrugName           string            `json:"drugName"`
	MonitoringRequired bool              `json:"monitoringRequired"`
	CriticalMonitoring bool              `json:"criticalMonitoring,omitempty"`
	REMSProgram        string            `json:"remsProgram,omitempty"`
	RequiredLabs       []string          `json:"requiredLabs,omitempty"`
	LabCodes           []string          `json:"labCodes,omitempty"` // LOINC codes
	Frequency          string            `json:"frequency,omitempty"`
	BaselineRequired   bool              `json:"baselineRequired,omitempty"`
	InitialMonitoring  string            `json:"initialMonitoring,omitempty"`
	OngoingMonitoring  string            `json:"ongoingMonitoring,omitempty"`
	CriticalValues     map[string]string `json:"criticalValues,omitempty"`
	ActionRequired     string            `json:"actionRequired,omitempty"`
}

// DoseLimitInfo contains dose limit details.
type DoseLimitInfo struct {
	RxNormCode         string             `json:"rxnormCode"`
	DrugName           string             `json:"drugName"`
	MaxSingleDose      float64            `json:"maxSingleDose"`
	MaxSingleDoseUnit  string             `json:"maxSingleDoseUnit"`
	MaxDailyDose       float64            `json:"maxDailyDose"`
	MaxDailyDoseUnit   string             `json:"maxDailyDoseUnit"`
	MaxCumulativeDose  float64            `json:"maxCumulativeDose,omitempty"`
	GeriatricMaxDose   float64            `json:"geriatricMaxDose,omitempty"`
	PediatricMaxDose   float64            `json:"pediatricMaxDose,omitempty"`
	RenalAdjustment    string             `json:"renalAdjustment,omitempty"`
	HepaticAdjustment  string             `json:"hepaticAdjustment,omitempty"`
	RenalDoseByEGFR    map[string]float64 `json:"renalDoseByEgfr,omitempty"`
	HepaticDoseByClass map[string]float64 `json:"hepaticDoseByClass,omitempty"`
}

// AgeLimitInfo contains age restriction details.
type AgeLimitInfo struct {
	RxNormCode  string `json:"rxnormCode"`
	DrugName    string `json:"drugName"`
	MinAgeYears int    `json:"minAgeYears,omitempty"`
	MaxAgeYears int    `json:"maxAgeYears,omitempty"`
	Rationale   string `json:"rationale"`
	Severity    string `json:"severity"` // CRITICAL, HIGH, MODERATE, LOW
}

// AllergyInfo represents an active allergy.
type AllergyInfo struct {
	Allergen    ClinicalCode `json:"allergen"`
	Category    string       `json:"category"`    // food, medication, environment
	Criticality string       `json:"criticality"` // low, high, unable-to-assess
	Reactions   []string     `json:"reactions,omitempty"`
}

// ContraindicationInfo represents a drug-condition contraindication.
type ContraindicationInfo struct {
	Medication     ClinicalCode `json:"medication"`
	Condition      ClinicalCode `json:"condition"`
	Severity       string       `json:"severity"` // absolute, relative
	Description    string       `json:"description"`
	Recommendation string       `json:"recommendation,omitempty"`
	Evidence       string       `json:"evidence,omitempty"`
}

// PregnancyInfo contains pregnancy status.
type PregnancyInfo struct {
	IsPregnant      bool   `json:"isPregnant"`
	Trimester       int    `json:"trimester,omitempty"`       // 1, 2, 3
	EstimatedWeeks  int    `json:"estimatedWeeks,omitempty"`  // gestational weeks
	LactationStatus bool   `json:"lactationStatus,omitempty"` // breastfeeding
}

// SafetyAlert represents a high-priority safety warning.
type SafetyAlert struct {
	AlertID     string    `json:"alertId"`
	Type        string    `json:"type"` // allergy, interaction, contraindication, duplicate
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ============================================================================
// KB-5: INTERACTION SNAPSHOT
// ============================================================================

// InteractionSnapshot contains KB-5 drug interaction analysis (per CTO/CMO spec).
type InteractionSnapshot struct {
	// CurrentDDIs are interactions between patient's active medications
	CurrentDDIs []DrugInteraction `json:"currentDDIs,omitempty"`

	// PotentialDDIs are interactions if adding common drugs
	PotentialDDIs []DrugInteraction `json:"potentialDDIs,omitempty"`

	// SeverityMax is the highest severity among all interactions
	// Values: "none", "low", "medium", "high", "critical"
	SeverityMax string `json:"severityMax"`

	// HasCriticalInteraction quick check flag
	HasCriticalInteraction bool `json:"hasCriticalInteraction"`
}

// DrugInteraction represents a detected drug-drug interaction.
type DrugInteraction struct {
	Drug1          ClinicalCode `json:"drug1"`
	Drug2          ClinicalCode `json:"drug2"`
	Severity       string       `json:"severity"` // mild, moderate, severe, critical
	Description    string       `json:"description"`
	Recommendation string       `json:"recommendation,omitempty"`
	Evidence       string       `json:"evidence,omitempty"`
}

// ============================================================================
// KB-6: FORMULARY SNAPSHOT
// ============================================================================

// FormularySnapshot contains KB-6 formulary status (per CTO/CMO spec).
type FormularySnapshot struct {
	// MedicationStatus maps RxNorm code to formulary status
	// Key: "system|code"
	MedicationStatus map[string]FormularyStatus `json:"medicationStatus"`

	// PriorAuthRequired lists medications needing prior authorization
	PriorAuthRequired []ClinicalCode `json:"priorAuthRequired,omitempty"`

	// GenericAlternatives maps brand name to generic alternatives
	// Key: "system|code" of brand drug
	GenericAlternatives map[string][]ClinicalCode `json:"genericAlternatives,omitempty"`

	// NLEMAvailability for India National List of Essential Medicines
	// Key: "system|code"
	// Value: true if on NLEM
	NLEMAvailability map[string]bool `json:"nlemAvailability,omitempty"`

	// PBSAvailability for Australia Pharmaceutical Benefits Scheme
	// Key: "system|code"
	// Value: true if on PBS
	PBSAvailability map[string]bool `json:"pbsAvailability,omitempty"`
}

// FormularyStatus represents a medication's formulary status.
type FormularyStatus struct {
	Code          ClinicalCode `json:"code"`
	Status        string       `json:"status"` // preferred, non-preferred, excluded, not-listed
	FormularyName string       `json:"formularyName,omitempty"`
	Tier          int          `json:"tier,omitempty"`
	Restrictions  []string     `json:"restrictions,omitempty"`
	CopayAmount   float64      `json:"copayAmount,omitempty"`
}

// ============================================================================
// KB-1: DOSING SNAPSHOT
// ============================================================================

// DosingSnapshot contains KB-1 dose adjustment information (per CTO/CMO spec).
type DosingSnapshot struct {
	// RenalAdjustments maps medication code to renal dose adjustment
	// Key: "system|code"
	RenalAdjustments map[string]DoseAdjustment `json:"renalAdjustments,omitempty"`

	// HepaticAdjustments maps medication code to hepatic dose adjustment
	// Key: "system|code"
	HepaticAdjustments map[string]DoseAdjustment `json:"hepaticAdjustments,omitempty"`

	// WeightBasedDoses maps medication code to weight-based dosing
	// Key: "system|code"
	WeightBasedDoses map[string]DoseCalculation `json:"weightBasedDoses,omitempty"`

	// AgeBasedAdjustments for pediatric/geriatric dosing
	// Key: "system|code"
	AgeBasedAdjustments map[string]DoseAdjustment `json:"ageBasedAdjustments,omitempty"`
}

// DoseAdjustment represents a recommended dose modification.
type DoseAdjustment struct {
	Medication        ClinicalCode `json:"medication"`
	Reason            string       `json:"reason"` // renal, hepatic, age, weight
	CurrentDose       *Quantity    `json:"currentDose,omitempty"`
	RecommendedDose   *Quantity    `json:"recommendedDose,omitempty"`
	AdjustmentPercent float64      `json:"adjustmentPercent,omitempty"`
	ThresholdEGFR     float64      `json:"thresholdEGFR,omitempty"` // eGFR threshold for adjustment
	Guidance          string       `json:"guidance,omitempty"`
	Evidence          string       `json:"evidence,omitempty"`
}

// DoseCalculation represents a weight/BSA-based dose calculation.
type DoseCalculation struct {
	Medication       ClinicalCode `json:"medication"`
	DosePerKg        float64      `json:"dosePerKg,omitempty"`        // mg/kg
	DosePerM2        float64      `json:"dosePerM2,omitempty"`        // mg/m² (BSA)
	MaxDose          *Quantity    `json:"maxDose,omitempty"`          // Maximum single dose
	CalculatedDose   *Quantity    `json:"calculatedDose,omitempty"`   // Calculated for this patient
	PatientWeight    float64      `json:"patientWeight,omitempty"`    // kg
	PatientBSA       float64      `json:"patientBSA,omitempty"`       // m²
	RoundingStrategy string       `json:"roundingStrategy,omitempty"` // nearest5, nearest10, etc.
}

// DrugInfo contains detailed drug information.
type DrugInfo struct {
	Code              ClinicalCode `json:"code"`
	GenericName       string       `json:"genericName"`
	BrandNames        []string     `json:"brandNames,omitempty"`
	DrugClass         string       `json:"drugClass,omitempty"`
	MechanismOfAction string       `json:"mechanismOfAction,omitempty"`
	RenalDosing       bool         `json:"renalDosing"`
	HepaticDosing     bool         `json:"hepaticDosing"`
	HighAlert         bool         `json:"highAlert"`
	Pregnancy         string       `json:"pregnancy,omitempty"` // A, B, C, D, X
	Lactation         string       `json:"lactation,omitempty"` // safe, caution, avoid
}

// CDIFacts contains KB-11 clinical documentation intelligence facts.
type CDIFacts struct {
	// ExtractedFacts from clinical notes
	ExtractedFacts []CDIFact `json:"extractedFacts,omitempty"`

	// CodingOpportunities identified by CDI
	CodingOpportunities []CodingOpportunity `json:"codingOpportunities,omitempty"`

	// QueryOpportunities for documentation improvement
	QueryOpportunities []QueryOpportunity `json:"queryOpportunities,omitempty"`
}

// CDIFact represents an extracted clinical documentation fact.
type CDIFact struct {
	FactType    string       `json:"factType"` // diagnosis, procedure, symptom
	Code        ClinicalCode `json:"code,omitempty"`
	Description string       `json:"description"`
	Confidence  float64      `json:"confidence"`
	SourceText  string       `json:"sourceText,omitempty"`
}

// CodingOpportunity represents a potential coding improvement.
type CodingOpportunity struct {
	CurrentCode     ClinicalCode `json:"currentCode,omitempty"`
	SuggestedCode   ClinicalCode `json:"suggestedCode"`
	Reason          string       `json:"reason"`
	ImpactEstimate  string       `json:"impactEstimate,omitempty"`
}

// QueryOpportunity represents a documentation clarification query.
type QueryOpportunity struct {
	QueryType   string `json:"queryType"`
	Question    string `json:"question"`
	Context     string `json:"context,omitempty"`
	Priority    string `json:"priority"`
}

// ============================================================================
// KB-16: LAB INTERPRETATION SNAPSHOT
// ============================================================================

// LabInterpretationSnapshot contains KB-16 pre-computed lab interpretations (per CTO/CMO spec).
// Reference ranges and critical values are patient-specific based on demographics.
type LabInterpretationSnapshot struct {
	// ReferenceRanges maps LOINC code to patient-specific reference range
	// Key: LOINC code (e.g., "2345-7" for Glucose)
	ReferenceRanges map[string]LabReferenceRange `json:"referenceRanges,omitempty"`

	// CriticalValues maps LOINC code to critical/panic thresholds
	// Key: LOINC code
	CriticalValues map[string]LabCriticalValue `json:"criticalValues,omitempty"`

	// LabInterpretations contains pre-computed interpretations for patient's recent labs
	// Keyed by source reference or LOINC|timestamp
	LabInterpretations map[string]LabInterpretationResult `json:"labInterpretations,omitempty"`

	// PanelInterpretations contains holistic panel assessments
	// Key: panel type (e.g., "BMP", "CMP", "CBC", "LipidPanel")
	PanelInterpretations map[string]LabPanelResult `json:"panelInterpretations,omitempty"`

	// HasCriticalValue indicates if any lab has a critical/panic value
	HasCriticalValue bool `json:"hasCriticalValue"`

	// CriticalLabCodes lists LOINC codes with critical values
	CriticalLabCodes []string `json:"criticalLabCodes,omitempty"`
}

// LabReferenceRange contains the normal range for a lab value.
type LabReferenceRange struct {
	// LOINCCode for the lab test
	LOINCCode string `json:"loincCode"`

	// DisplayName of the lab test
	DisplayName string `json:"displayName,omitempty"`

	// LowNormal end of normal range
	LowNormal float64 `json:"lowNormal"`

	// HighNormal end of normal range
	HighNormal float64 `json:"highNormal"`

	// Unit of measurement (e.g., "mg/dL", "mmol/L")
	Unit string `json:"unit"`

	// CriticalLow threshold (panic value)
	CriticalLow *float64 `json:"criticalLow,omitempty"`

	// CriticalHigh threshold (panic value)
	CriticalHigh *float64 `json:"criticalHigh,omitempty"`

	// AgeMin this range applies to
	AgeMin int `json:"ageMin,omitempty"`

	// AgeMax this range applies to
	AgeMax int `json:"ageMax,omitempty"`

	// Sex this range applies to (empty = both)
	Sex string `json:"sex,omitempty"`

	// IsPregnancyRange indicates pregnancy-specific range
	IsPregnancyRange bool `json:"isPregnancyRange,omitempty"`

	// Source of the reference range (e.g., "AACC", "WHO", "RACGP")
	Source string `json:"source,omitempty"`
}

// LabCriticalValue contains critical/panic value thresholds.
type LabCriticalValue struct {
	// LOINCCode for the lab test
	LOINCCode string `json:"loincCode"`

	// DisplayName of the lab test
	DisplayName string `json:"displayName,omitempty"`

	// CriticalLow threshold
	CriticalLow *float64 `json:"criticalLow,omitempty"`

	// CriticalHigh threshold
	CriticalHigh *float64 `json:"criticalHigh,omitempty"`

	// Unit of measurement
	Unit string `json:"unit"`

	// RequiredAction for critical values
	RequiredAction string `json:"requiredAction,omitempty"`

	// NotifyWithinMinutes time to notify for critical values
	NotifyWithinMinutes int `json:"notifyWithinMinutes,omitempty"`

	// Source/Guideline reference
	Source string `json:"source,omitempty"`
}

// LabInterpretationResult provides clinical interpretation of a lab result.
type LabInterpretationResult struct {
	// LOINCCode for the lab test
	LOINCCode string `json:"loincCode"`

	// DisplayName of the lab test
	DisplayName string `json:"displayName,omitempty"`

	// Value of the result
	Value float64 `json:"value"`

	// Unit of measurement
	Unit string `json:"unit"`

	// AbnormalityLevel: "normal", "low", "high", "critical_low", "critical_high"
	AbnormalityLevel string `json:"abnormalityLevel"`

	// Flag for display: "", "L", "H", "LL", "HH"
	Flag string `json:"flag"`

	// IsCritical indicates if this is a critical/panic value
	IsCritical bool `json:"isCritical"`

	// ClinicalSignificance: "none", "mild", "moderate", "severe"
	ClinicalSignificance string `json:"clinicalSignificance"`

	// PossibleCauses for abnormality
	PossibleCauses []string `json:"possibleCauses,omitempty"`

	// SuggestedActions for follow-up
	SuggestedActions []string `json:"suggestedActions,omitempty"`

	// Trend direction: "stable", "improving", "worsening", "unknown"
	Trend string `json:"trend,omitempty"`

	// Narrative interpretation
	Narrative string `json:"narrative,omitempty"`

	// EffectiveDateTime when the lab was collected
	EffectiveDateTime *time.Time `json:"effectiveDateTime,omitempty"`
}

// LabPanelResult provides holistic interpretation of a lab panel.
type LabPanelResult struct {
	// PanelType (e.g., "BMP", "CMP", "CBC", "LipidPanel")
	PanelType string `json:"panelType"`

	// DisplayName of the panel
	DisplayName string `json:"displayName,omitempty"`

	// OverallAssessment: "normal", "abnormal", "critical"
	OverallAssessment string `json:"overallAssessment"`

	// AbnormalCount number of abnormal results in panel
	AbnormalCount int `json:"abnormalCount"`

	// CriticalCount number of critical results in panel
	CriticalCount int `json:"criticalCount"`

	// PanelFindings from analyzing multiple labs together
	PanelFindings []LabPanelFinding `json:"panelFindings,omitempty"`

	// DifferentialDiagnoses suggested by the panel pattern
	DifferentialDiagnoses []string `json:"differentialDiagnoses,omitempty"`

	// RecommendedFollowUp tests
	RecommendedFollowUp []string `json:"recommendedFollowUp,omitempty"`

	// Narrative interpretation
	Narrative string `json:"narrative,omitempty"`
}

// LabPanelFinding represents a finding from analyzing multiple labs together.
type LabPanelFinding struct {
	// Type: "pattern", "ratio", "correlation"
	Type string `json:"type"`

	// Description of the finding
	Description string `json:"description"`

	// InvolvedLabs LOINC codes involved in this finding
	InvolvedLabs []string `json:"involvedLabs"`

	// Significance: "informational", "notable", "concerning"
	Significance string `json:"significance"`

	// Interpretation of the finding
	Interpretation string `json:"interpretation,omitempty"`
}

// CalculationResult represents a clinical calculation result.
type CalculationResult struct {
	// Name of the calculator
	Name string `json:"name"`

	// Value is the primary result
	Value float64 `json:"value"`

	// Unit of the result
	Unit string `json:"unit,omitempty"`

	// Category interpretation
	Category string `json:"category,omitempty"`

	// Formula used (for transparency)
	Formula string `json:"formula,omitempty"`

	// Inputs used for the calculation
	Inputs map[string]interface{} `json:"inputs,omitempty"`

	// CalculatedAt timestamp
	CalculatedAt time.Time `json:"calculatedAt"`

	// Warnings or caveats
	Warnings []string `json:"warnings,omitempty"`
}

// ============================================================================
// RUNTIME METADATA
// ============================================================================

// ExecutionMetadata contains context about the execution request.
type ExecutionMetadata struct {
	// RequestID unique identifier for this execution
	RequestID string `json:"requestId"`

	// RequestedBy user/system that initiated the request
	RequestedBy string `json:"requestedBy"`

	// RequestedAt timestamp of the request
	RequestedAt time.Time `json:"requestedAt"`

	// Region where execution is happening (IN, AU)
	Region string `json:"region"`

	// TenantID for multi-tenant deployments
	TenantID string `json:"tenantId,omitempty"`

	// MeasurementPeriod for quality measure calculations
	MeasurementPeriod *Period `json:"measurementPeriod,omitempty"`

	// RequestedEngines list of engines to execute
	RequestedEngines []string `json:"requestedEngines,omitempty"`

	// ExecutionMode (sync, async, batch)
	ExecutionMode string `json:"executionMode,omitempty"`
}

// ============================================================================
// COMMON TYPES
// ============================================================================

// ClinicalCode represents a coded clinical concept.
type ClinicalCode struct {
	// System is the code system URI (e.g., "http://snomed.info/sct")
	System string `json:"system"`

	// Code is the actual code value
	Code string `json:"code"`

	// Display is human-readable description
	Display string `json:"display,omitempty"`
}

// Quantity represents a measured quantity with value and unit.
type Quantity struct {
	// Value is the numeric value
	Value float64 `json:"value"`

	// Unit is the unit of measure
	Unit string `json:"unit,omitempty"`

	// System is the unit system (e.g., UCUM)
	System string `json:"system,omitempty"`

	// Code is the coded unit
	Code string `json:"code,omitempty"`
}

// Period represents a time period.
type Period struct {
	// Start of the period
	Start *time.Time `json:"start,omitempty"`

	// End of the period
	End *time.Time `json:"end,omitempty"`
}

// ReferenceRange represents lab result reference ranges.
type ReferenceRange struct {
	// Low bound
	Low *Quantity `json:"low,omitempty"`

	// High bound
	High *Quantity `json:"high,omitempty"`

	// Text description
	Text string `json:"text,omitempty"`
}

// ============================================================================
// ENGINE RESULT CONTRACT
// ============================================================================

// EngineResult is what every engine returns after processing ClinicalExecutionContext.
// All engines MUST return this structure.
//
// Per CTO/CMO Architecture:
//   - CQL Engine produces ClinicalFacts (truths) but NOT MeasureResults
//   - Measure Engine consumes ClinicalFacts and produces MeasureResults (care gaps)
//   - Other engines (Medication, etc.) may produce both
type EngineResult struct {
	// EngineName identifies which engine produced this result
	EngineName string `json:"engineName"`

	// Success indicates if the engine executed without error
	Success bool `json:"success"`

	// ═══════════════════════════════════════════════════════════════════════════
	// CQL ENGINE OUTPUT: Clinical Facts (Truths)
	// ═══════════════════════════════════════════════════════════════════════════

	// ClinicalFacts are truth statements produced by CQL Engine
	// Example: "HbA1cPoorControl = true", "BloodPressureControlled = false"
	// These facts flow to Measure Engine for care gap determination
	ClinicalFacts []ClinicalFact `json:"clinicalFacts,omitempty"`

	// ═══════════════════════════════════════════════════════════════════════════
	// MEASURE ENGINE OUTPUT: Care Judgments
	// ═══════════════════════════════════════════════════════════════════════════

	// MeasureResults for quality measure calculations (care gaps)
	// Produced by Measure Engine after consuming ClinicalFacts
	MeasureResults []MeasureResult `json:"measureResults,omitempty"`

	// ═══════════════════════════════════════════════════════════════════════════
	// GENERAL ENGINE OUTPUTS
	// ═══════════════════════════════════════════════════════════════════════════

	// Recommendations produced by the engine
	Recommendations []Recommendation `json:"recommendations,omitempty"`

	// Alerts generated by the engine
	Alerts []Alert `json:"alerts,omitempty"`

	// ExecutionTime how long the engine took
	ExecutionTimeMs int64 `json:"executionTimeMs"`

	// Error if Success is false
	Error string `json:"error,omitempty"`

	// EvidenceLinks for audit trail
	EvidenceLinks []string `json:"evidenceLinks,omitempty"`
}

// Recommendation represents a clinical recommendation from an engine.
type Recommendation struct {
	// ID unique identifier for this recommendation
	ID string `json:"id"`

	// Type (medication, lab, referral, counseling, etc.)
	Type string `json:"type"`

	// Title short description
	Title string `json:"title"`

	// Description detailed explanation
	Description string `json:"description,omitempty"`

	// Priority (high, medium, low)
	Priority string `json:"priority"`

	// Source guideline or measure that generated this
	Source string `json:"source,omitempty"`

	// Actions suggested clinical actions
	Actions []SuggestedAction `json:"actions,omitempty"`
}

// SuggestedAction represents a specific action to take.
type SuggestedAction struct {
	// Type of action (order, prescribe, refer, schedule)
	Type string `json:"type"`

	// Description of the action
	Description string `json:"description"`

	// Resource FHIR resource type to create
	Resource string `json:"resource,omitempty"`

	// Draft resource content
	DraftContent interface{} `json:"draftContent,omitempty"`
}

// Alert represents a clinical alert from an engine.
type Alert struct {
	// ID unique identifier
	ID string `json:"id"`

	// Severity (critical, high, moderate, low, info)
	Severity string `json:"severity"`

	// Category (safety, quality, documentation, etc.)
	Category string `json:"category"`

	// Title short description
	Title string `json:"title"`

	// Description detailed explanation
	Description string `json:"description,omitempty"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"createdAt"`
}

// ============================================================================
// CLINICAL FACTS (CQL Engine Output → Measure Engine Input)
// ============================================================================

// ClinicalFact represents a clinical truth determined by CQL Engine.
// These are binary or structured clinical truths, NOT care judgments.
//
// Per CTO/CMO Architecture:
//   - CQL Engine produces facts: "Is HbA1c > 9%?" → true
//   - Measure Engine consumes facts to determine care gaps
//   - Facts are immutable, deterministic, and auditable
type ClinicalFact struct {
	// FactID unique identifier (e.g., "HbA1cPoorControl", "BloodPressureControlled")
	FactID string `json:"factId"`

	// Value is the boolean truth value
	Value bool `json:"value"`

	// NumericValue for facts with numeric answers (optional)
	NumericValue *float64 `json:"numericValue,omitempty"`

	// Evidence explains WHY this fact is true/false
	Evidence string `json:"evidence,omitempty"`

	// SourceData references the clinical data that determined this fact
	SourceData []string `json:"sourceData,omitempty"`

	// EvaluatedAt timestamp of fact determination
	EvaluatedAt time.Time `json:"evaluatedAt,omitempty"`

	// FactCategory groups related facts (e.g., "glycemic", "cardiovascular", "screening")
	FactCategory string `json:"factCategory,omitempty"`
}

// StandardClinicalFacts defines the canonical fact IDs produced by CQL Engine.
// These are the "clinical truths" that Measure Engine consumes.
const (
	// Diabetes-related facts
	FactHbA1cPoorControl       = "HbA1cPoorControl"       // HbA1c > 9%
	FactHbA1cModerateControl   = "HbA1cModerateControl"   // HbA1c 7-9%
	FactHbA1cGoodControl       = "HbA1cGoodControl"       // HbA1c < 7%
	FactHasDiabetes            = "HasDiabetes"            // Patient has diabetes diagnosis

	// Blood pressure facts
	FactBloodPressureControlled   = "BloodPressureControlled"   // BP < 140/90
	FactBloodPressureUncontrolled = "BloodPressureUncontrolled" // BP >= 140/90
	FactHasHypertension           = "HasHypertension"           // Patient has HTN diagnosis

	// Kidney health facts
	FactKidneyScreeningComplete = "KidneyScreeningComplete" // uACR or eGFR test in period
	FactHasACEorARB             = "HasACEorARB"             // On ACE inhibitor or ARB
	FactHasCKD                  = "HasCKD"                  // Patient has CKD

	// Depression screening facts
	FactDepressionScreeningComplete = "DepressionScreeningComplete" // PHQ-9 in period
	FactPositiveDepressionScreen    = "PositiveDepressionScreen"    // PHQ-9 >= 10
	FactFollowUpPlanDocumented      = "FollowUpPlanDocumented"      // Follow-up documented

	// General facts
	FactHasOutpatientEncounter = "HasOutpatientEncounter" // Qualifying encounter in period
	FactIsAdult                = "IsAdult"                // Age >= 18
	FactIsEligibleAge          = "IsEligibleAge"          // Within measure age range
)

// MeasureResult represents a quality measure calculation result.
type MeasureResult struct {
	// MeasureID (e.g., "CMS122")
	MeasureID string `json:"measureId"`

	// MeasureName human-readable name
	MeasureName string `json:"measureName,omitempty"`

	// InInitialPopulation eligibility check
	InInitialPopulation bool `json:"inInitialPopulation"`

	// InDenominator (if applicable)
	InDenominator bool `json:"inDenominator"`

	// InNumerator (if applicable)
	InNumerator bool `json:"inNumerator"`

	// InDenominatorExclusion (if applicable)
	InDenominatorExclusion bool `json:"inDenominatorExclusion"`

	// InDenominatorException (if applicable)
	InDenominatorException bool `json:"inDenominatorException"`

	// EvaluatedResources FHIR resources used in calculation
	EvaluatedResources []string `json:"evaluatedResources,omitempty"`

	// CareGapIdentified if patient has a gap in this measure
	CareGapIdentified bool `json:"careGapIdentified"`

	// ═══════════════════════════════════════════════════════════════════════════
	// AUDIT METADATA (CTO/CMO Requirement)
	// These fields provide regulatory defensibility and version traceability
	// ═══════════════════════════════════════════════════════════════════════════

	// MeasureVersion CMS-published version (e.g., "2024.0.0")
	MeasureVersion string `json:"measureVersion,omitempty"`

	// LogicVersion internal implementation version (e.g., "1.0.0")
	// Tracks changes to our Go-based rule implementation
	LogicVersion string `json:"logicVersion,omitempty"`

	// ELMCorrespondence maps to the CQL/ELM artifact this implements
	// Format: "{library_name}:{version}" (e.g., "DiabetesGlycemicStatusAssessmentGreaterThan9PercentFHIR:0.1.002")
	ELMCorrespondence string `json:"elmCorrespondence,omitempty"`

	// EvaluatedAt timestamp of evaluation
	EvaluatedAt time.Time `json:"evaluatedAt,omitempty"`

	// Rationale explains why patient is in/out of population
	Rationale string `json:"rationale,omitempty"`
}

// ============================================================================
// KB-10 RULES ENGINE TYPES (RUNTIME)
// ============================================================================

// RuleEvaluationResult contains the outcome of rule set evaluation.
// Produced by KB-10 Rules Engine during runtime.
type RuleEvaluationResult struct {
	// RuleSetID that was evaluated
	RuleSetID string `json:"ruleSetId"`

	// EvaluatedAt timestamp of evaluation
	EvaluatedAt time.Time `json:"evaluatedAt"`

	// TriggeredRules that matched the facts
	TriggeredRules []TriggeredRule `json:"triggeredRules"`

	// GeneratedAlerts from triggered rules
	GeneratedAlerts []ClinicalAlert `json:"generatedAlerts"`

	// TotalRulesRun count of rules evaluated
	TotalRulesRun int `json:"totalRulesRun"`

	// ExecutionTimeMs processing time
	ExecutionTimeMs int64 `json:"executionTimeMs"`
}

// MultiRuleEvaluationResult contains results from evaluating multiple rule sets.
type MultiRuleEvaluationResult struct {
	// RuleSetIDs that were evaluated
	RuleSetIDs []string `json:"ruleSetIds"`

	// EvaluatedAt timestamp of evaluation
	EvaluatedAt time.Time `json:"evaluatedAt"`

	// TriggeredRules from all rule sets
	TriggeredRules []TriggeredRule `json:"triggeredRules"`

	// GeneratedAlerts from all triggered rules
	GeneratedAlerts []ClinicalAlert `json:"generatedAlerts"`

	// TotalRulesRun count of rules evaluated across all sets
	TotalRulesRun int `json:"totalRulesRun"`

	// ExecutionTimeMs total processing time
	ExecutionTimeMs int64 `json:"executionTimeMs"`
}

// TriggeredRule represents a rule that matched during evaluation.
type TriggeredRule struct {
	// RuleID unique identifier
	RuleID string `json:"ruleId"`

	// RuleName human-readable name
	RuleName string `json:"ruleName"`

	// Severity of the triggered rule (critical, high, moderate, low)
	Severity string `json:"severity"`

	// Condition that matched (description)
	Condition string `json:"condition"`

	// MatchedFacts that triggered the rule
	MatchedFacts map[string]interface{} `json:"matchedFacts,omitempty"`

	// Recommendation from the rule
	Recommendation string `json:"recommendation,omitempty"`

	// EvidenceLevel (e.g., "A", "B", "C", "D")
	EvidenceLevel string `json:"evidenceLevel,omitempty"`

	// GuidelineSource (e.g., "AHA 2024", "CMS-122")
	GuidelineSource string `json:"guidelineSource,omitempty"`
}

// ClinicalAlert represents an alert generated by the rules engine.
type ClinicalAlert struct {
	// AlertID unique identifier
	AlertID string `json:"alertId"`

	// PatientID the alert relates to
	PatientID string `json:"patientId"`

	// AlertType categorizes the alert (drug_interaction, critical_lab, protocol_deviation)
	AlertType string `json:"alertType"`

	// Severity of the alert (critical, high, moderate, low, info)
	Severity string `json:"severity"`

	// Title short description
	Title string `json:"title"`

	// Description detailed explanation
	Description string `json:"description,omitempty"`

	// SourceRule that generated this alert
	SourceRule string `json:"sourceRule,omitempty"`

	// GeneratedAt timestamp
	GeneratedAt time.Time `json:"generatedAt"`

	// ExpiresAt when the alert expires (optional)
	ExpiresAt time.Time `json:"expiresAt,omitempty"`

	// Acknowledged if the alert has been seen/handled
	Acknowledged bool `json:"acknowledged"`

	// AcknowledgedBy user who acknowledged
	AcknowledgedBy string `json:"acknowledgedBy,omitempty"`

	// AcknowledgedAt when acknowledged
	AcknowledgedAt time.Time `json:"acknowledgedAt,omitempty"`

	// ActionItems suggested actions to take
	ActionItems []string `json:"actionItems,omitempty"`
}

// RuleSetDefinition describes a set of clinical rules.
type RuleSetDefinition struct {
	// RuleSetID unique identifier
	RuleSetID string `json:"ruleSetId"`

	// Name human-readable name
	Name string `json:"name"`

	// Description of the rule set purpose
	Description string `json:"description,omitempty"`

	// Version of the rule set
	Version string `json:"version,omitempty"`

	// Category (sepsis, aki, medication, etc.)
	Category string `json:"category,omitempty"`

	// Enabled if the rule set is active
	Enabled bool `json:"enabled"`

	// Rules in this set
	Rules []RuleDefinition `json:"rules,omitempty"`
}

// RuleDefinition describes a single clinical rule.
type RuleDefinition struct {
	// RuleID unique identifier
	RuleID string `json:"ruleId"`

	// Name human-readable name
	Name string `json:"name"`

	// Description of what the rule detects
	Description string `json:"description,omitempty"`

	// Condition expression or description
	Condition string `json:"condition"`

	// Severity when triggered (critical, high, moderate, low)
	Severity string `json:"severity"`

	// Enabled if the rule is active
	Enabled bool `json:"enabled"`

	// EvidenceLevel (e.g., "A", "B", "C", "D")
	EvidenceLevel string `json:"evidenceLevel,omitempty"`

	// GuidelineSource (e.g., "AHA 2024", "CMS-122")
	GuidelineSource string `json:"guidelineSource,omitempty"`
}

// ============================================================================
// KB-15 EVIDENCE ENGINE TYPES (RUNTIME)
// ============================================================================

// EvidenceQuery represents search criteria for clinical evidence.
// Uses PICO framework (Population, Intervention, Comparison, Outcome).
type EvidenceQuery struct {
	// ConditionCodes SNOMED/ICD codes for conditions (Population)
	ConditionCodes []string `json:"conditionCodes,omitempty"`

	// InterventionCodes RxNorm/SNOMED codes for interventions
	InterventionCodes []string `json:"interventionCodes,omitempty"`

	// OutcomeCodes SNOMED/LOINC codes for outcomes
	OutcomeCodes []string `json:"outcomeCodes,omitempty"`

	// StudyTypes to include (RCT, meta-analysis, cohort, etc.)
	StudyTypes []string `json:"studyTypes,omitempty"`

	// MinGrade minimum GRADE level to include (High, Moderate, Low)
	MinGrade string `json:"minGrade,omitempty"`

	// MaxResults to return
	MaxResults int `json:"maxResults,omitempty"`

	// DateRange for publication date filter
	DateRange *Period `json:"dateRange,omitempty"`
}

// EvidenceResult contains search results from the evidence engine.
type EvidenceResult struct {
	// Query that was executed
	Query EvidenceQuery `json:"query"`

	// TotalMatches found
	TotalMatches int `json:"totalMatches"`

	// Items returned (may be limited by MaxResults)
	Items []EvidenceItem `json:"items"`

	// SearchTime in milliseconds
	SearchTime int64 `json:"searchTime"`
}

// EvidenceItem represents a single evidence item (study, guideline, etc.).
type EvidenceItem struct {
	// EvidenceID unique identifier
	EvidenceID string `json:"evidenceId"`

	// Title of the study/article
	Title string `json:"title"`

	// Authors list
	Authors []string `json:"authors,omitempty"`

	// PublicationDate of the study
	PublicationDate time.Time `json:"publicationDate,omitempty"`

	// Journal name
	Journal string `json:"journal,omitempty"`

	// StudyType (RCT, meta-analysis, cohort, case-control, etc.)
	StudyType string `json:"studyType,omitempty"`

	// SampleSize of the study
	SampleSize int `json:"sampleSize,omitempty"`

	// Summary abstract or key findings
	Summary string `json:"summary,omitempty"`

	// GRADELevel quality assessment (High, Moderate, Low, Very Low)
	GRADELevel string `json:"gradeLevel,omitempty"`

	// DOI digital object identifier
	DOI string `json:"doi,omitempty"`

	// PMID PubMed ID
	PMID string `json:"pmid,omitempty"`

	// Relevance score (0.0-1.0)
	Relevance float64 `json:"relevance,omitempty"`
}

// GRADEAssessment represents a GRADE quality assessment for evidence.
type GRADEAssessment struct {
	// EvidenceID the assessment is for
	EvidenceID string `json:"evidenceId"`

	// OverallGrade (High, Moderate, Low, Very Low)
	OverallGrade string `json:"overallGrade"`

	// Certainty description
	Certainty string `json:"certainty,omitempty"`

	// Domains individual GRADE domain assessments
	Domains []GRADEDomain `json:"domains,omitempty"`

	// Recommendation derived from the evidence
	Recommendation string `json:"recommendation,omitempty"`

	// StrengthOfRec (Strong, Weak/Conditional)
	StrengthOfRec string `json:"strengthOfRec,omitempty"`

	// LastAssessedAt timestamp
	LastAssessedAt time.Time `json:"lastAssessedAt,omitempty"`

	// AssessedBy who performed the assessment
	AssessedBy string `json:"assessedBy,omitempty"`

	// GRADEVersion version of GRADE methodology used
	GRADEVersion string `json:"gradeVersion,omitempty"`
}

// GRADEDomain represents assessment of a single GRADE domain.
type GRADEDomain struct {
	// Name of the domain (risk_of_bias, inconsistency, indirectness, imprecision, publication_bias)
	Name string `json:"name"`

	// Rating for this domain (no_serious_concern, serious_concern, very_serious_concern)
	Rating string `json:"rating"`

	// Concern level (none, serious, very_serious)
	Concern string `json:"concern,omitempty"`

	// Explanation of the rating
	Explanation string `json:"explanation,omitempty"`
}

// EvidenceEnvelope contains pre-packaged evidence for a protocol decision node.
type EvidenceEnvelope struct {
	// ProtocolID the envelope is for
	ProtocolID string `json:"protocolId"`

	// DecisionNodeID specific decision point
	DecisionNodeID string `json:"decisionNodeId"`

	// Items evidence items relevant to this decision
	Items []EvidenceItem `json:"items,omitempty"`

	// SummaryStatement narrative summary of the evidence
	SummaryStatement string `json:"summaryStatement,omitempty"`

	// Summary alternative summary field
	Summary string `json:"summary,omitempty"`

	// OverallGrade evidence quality grade
	OverallGrade string `json:"overallGrade,omitempty"`

	// StrengthOfRec recommendation strength
	StrengthOfRec string `json:"strengthOfRec,omitempty"`

	// RecommendationStrength from the evidence
	RecommendationStrength string `json:"recommendationStrength,omitempty"`

	// CertaintyOfEvidence overall certainty
	CertaintyOfEvidence string `json:"certaintyOfEvidence,omitempty"`

	// LastUpdated when envelope was last refreshed
	LastUpdated time.Time `json:"lastUpdated,omitempty"`

	// LastUpdatedAt alternative timestamp field
	LastUpdatedAt *time.Time `json:"lastUpdatedAt,omitempty"`

	// GuidelineSources source guidelines
	GuidelineSources []string `json:"guidelineSources,omitempty"`

	// CuratedBy who curated this envelope
	CuratedBy string `json:"curatedBy,omitempty"`
}

// SystematicReview represents a systematic review or meta-analysis.
type SystematicReview struct {
	// ReviewID unique identifier
	ReviewID string `json:"reviewId"`

	// Title of the review
	Title string `json:"title"`

	// Authors list
	Authors []string `json:"authors,omitempty"`

	// PublicationDate when published
	PublicationDate time.Time `json:"publicationDate,omitempty"`

	// CochraneID if Cochrane review
	CochraneID string `json:"cochraneId,omitempty"`

	// StudiesIncluded count of studies included
	StudiesIncluded int `json:"studiesIncluded,omitempty"`

	// IncludedStudies alternative field name
	IncludedStudies int `json:"includedStudies,omitempty"`

	// Participants total participants
	Participants int `json:"participants,omitempty"`

	// TotalParticipants across all studies
	TotalParticipants int `json:"totalParticipants,omitempty"`

	// Summary brief summary
	Summary string `json:"summary,omitempty"`

	// MainFindings key findings
	MainFindings string `json:"mainFindings,omitempty"`

	// PooledEffect summary effect if meta-analysis
	PooledEffect string `json:"pooledEffect,omitempty"`

	// Heterogeneity I² value
	Heterogeneity string `json:"heterogeneity,omitempty"`

	// AuthorConclusion author's conclusion
	AuthorConclusion string `json:"authorConclusion,omitempty"`

	// Conclusions main conclusions
	Conclusions string `json:"conclusions,omitempty"`

	// GRADELevel quality grade
	GRADELevel string `json:"gradeLevel,omitempty"`

	// GRADEAssessment quality assessment
	GRADEAssessment *GRADEAssessment `json:"gradeAssessment,omitempty"`

	// LastUpdated when last updated
	LastUpdated time.Time `json:"lastUpdated,omitempty"`

	// DOI digital object identifier
	DOI string `json:"doi,omitempty"`

	// PMID PubMed ID
	PMID string `json:"pmid,omitempty"`
}

// GuidelineEvidence represents evidence from a clinical guideline.
type GuidelineEvidence struct {
	// GuidelineID unique identifier
	GuidelineID string `json:"guidelineId"`

	// RecommendationID unique identifier for the specific recommendation
	RecommendationID string `json:"recommendationId,omitempty"`

	// Title of the guideline
	Title string `json:"title"`

	// Organization that published (e.g., AHA, ACC, ESC)
	Organization string `json:"organization,omitempty"`

	// GuidelineOrganization alternative field for organization name
	GuidelineOrganization string `json:"guidelineOrganization,omitempty"`

	// PublicationDate when published
	PublicationDate *time.Time `json:"publicationDate,omitempty"`

	// PublicationYear year of publication
	PublicationYear int `json:"publicationYear,omitempty"`

	// Version of the guideline
	Version string `json:"version,omitempty"`

	// RecommendationClass (I, IIa, IIb, III)
	RecommendationClass string `json:"recommendationClass,omitempty"`

	// RecommendationGrade (A, B, C, etc.)
	RecommendationGrade string `json:"recommendationGrade,omitempty"`

	// LevelOfEvidence (A, B-R, B-NR, C-LD, C-EO)
	LevelOfEvidence string `json:"levelOfEvidence,omitempty"`

	// StrengthOfRecommendation (Strong, Weak/Conditional)
	StrengthOfRecommendation string `json:"strengthOfRecommendation,omitempty"`

	// RecommendationText the actual recommendation
	RecommendationText string `json:"recommendationText,omitempty"`

	// SupportingEvidence references (legacy string array)
	SupportingEvidence []string `json:"supportingEvidence,omitempty"`

	// SupportingItems structured evidence items
	SupportingItems []EvidenceItem `json:"supportingItems,omitempty"`

	// EvidenceSummary summary of the supporting evidence
	EvidenceSummary string `json:"evidenceSummary,omitempty"`

	// TargetPopulation who the recommendation applies to
	TargetPopulation string `json:"targetPopulation,omitempty"`

	// URL link to guideline
	URL string `json:"url,omitempty"`
}

// ============================================================================
// ICU INTELLIGENCE & SAFETY TYPES (RUNTIME)
// ============================================================================

// SafetyFacts contains safety-relevant clinical facts for ICU veto evaluation.
// These facts come from CQL evaluation of SafetyCommon.cql.
type SafetyFacts struct {
	// PatientID the facts relate to
	PatientID string `json:"patientId"`

	// EncounterID current encounter
	EncounterID string `json:"encounterId,omitempty"`

	// VitalSigns current vital measurements
	VitalSigns SafetyVitalSigns `json:"vitalSigns,omitempty"`

	// LabValues current lab values
	LabValues SafetyLabValues `json:"labValues,omitempty"`

	// ActiveConditions relevant conditions
	ActiveConditions []string `json:"activeConditions,omitempty"`

	// ActiveMedications current medications (RxNorm codes)
	ActiveMedications []string `json:"activeMedications,omitempty"`

	// Allergies known allergies
	Allergies []string `json:"allergies,omitempty"`

	// RiskScores calculated risk scores
	RiskScores map[string]float64 `json:"riskScores,omitempty"`

	// Flags safety-relevant flags
	Flags SafetyFlags `json:"flags,omitempty"`

	// EvaluatedAt timestamp when facts were gathered
	EvaluatedAt time.Time `json:"evaluatedAt,omitempty"`
}

// SafetyVitalSigns contains vital signs relevant for safety checks.
type SafetyVitalSigns struct {
	// SystolicBP in mmHg
	SystolicBP *float64 `json:"systolicBP,omitempty"`

	// DiastolicBP in mmHg
	DiastolicBP *float64 `json:"diastolicBP,omitempty"`

	// HeartRate in bpm
	HeartRate *float64 `json:"heartRate,omitempty"`

	// RespiratoryRate in breaths/min
	RespiratoryRate *float64 `json:"respiratoryRate,omitempty"`

	// SpO2 oxygen saturation (%)
	SpO2 *float64 `json:"spO2,omitempty"`

	// Temperature in Celsius
	Temperature *float64 `json:"temperature,omitempty"`

	// GCS Glasgow Coma Scale score
	GCS *int `json:"gcs,omitempty"`
}

// SafetyLabValues contains lab values relevant for safety checks.
type SafetyLabValues struct {
	// Lactate in mmol/L
	Lactate *float64 `json:"lactate,omitempty"`

	// Creatinine in mg/dL
	Creatinine *float64 `json:"creatinine,omitempty"`

	// eGFR in mL/min/1.73m²
	EGFR *float64 `json:"eGFR,omitempty"`

	// Potassium in mEq/L
	Potassium *float64 `json:"potassium,omitempty"`

	// Hemoglobin in g/dL
	Hemoglobin *float64 `json:"hemoglobin,omitempty"`

	// Platelets in K/uL
	Platelets *float64 `json:"platelets,omitempty"`

	// INR International Normalized Ratio
	INR *float64 `json:"inr,omitempty"`

	// Troponin in ng/mL
	Troponin *float64 `json:"troponin,omitempty"`
}

// SafetyFlags contains boolean safety flags.
type SafetyFlags struct {
	// IsPregnant pregnancy status
	IsPregnant bool `json:"isPregnant"`

	// IsBreastfeeding lactation status
	IsBreastfeeding bool `json:"isBreastfeeding"`

	// HasActiveBleed active bleeding
	HasActiveBleed bool `json:"hasActiveBleed"`

	// IsOnAnticoagulation anticoagulation therapy
	IsOnAnticoagulation bool `json:"isOnAnticoagulation"`

	// HasSevereRenalImpairment eGFR < 30
	HasSevereRenalImpairment bool `json:"hasSevereRenalImpairment"`

	// HasSevereHepaticImpairment liver failure
	HasSevereHepaticImpairment bool `json:"hasSevereHepaticImpairment"`

	// IsHemodynamicallyUnstable shock state
	IsHemodynamicallyUnstable bool `json:"isHemodynamicallyUnstable"`

	// HasAlteredMentalStatus GCS < 15 or delirium
	HasAlteredMentalStatus bool `json:"hasAlteredMentalStatus"`

	// IsOnVasopressors vasopressor support
	IsOnVasopressors bool `json:"isOnVasopressors"`

	// IsOnMechanicalVentilation ventilator support
	IsOnMechanicalVentilation bool `json:"isOnMechanicalVentilation"`
}

// ICUDominanceState represents the current ICU dominance state for a patient.
type ICUDominanceState struct {
	// PatientID the state relates to
	PatientID string `json:"patientId"`

	// EncounterID current encounter
	EncounterID string `json:"encounterId,omitempty"`

	// CurrentState active dominance state (NONE, SHOCK, HYPOXIA, ACTIVE_BLEED, etc.)
	CurrentState DominanceState `json:"currentState"`

	// Severity of the current state (0-10)
	Severity int `json:"severity"`

	// ConfidenceScore of the classification (0.0-1.0)
	ConfidenceScore float64 `json:"confidenceScore"`

	// TriggeringFactors that caused this state
	TriggeringFactors []string `json:"triggeringFactors,omitempty"`

	// ActiveSafetyFlags from safety evaluation
	ActiveSafetyFlags SafetyFlags `json:"activeSafetyFlags,omitempty"`

	// StateStartedAt when the current state began
	StateStartedAt *time.Time `json:"stateStartedAt,omitempty"`

	// LastEvaluatedAt when state was last checked
	LastEvaluatedAt time.Time `json:"lastEvaluatedAt"`

	// UnderICUOversight if ICU is actively monitoring
	UnderICUOversight bool `json:"underIcuOversight"`

	// CanPreemptWorkflows if ICU can interrupt current workflows
	CanPreemptWorkflows bool `json:"canPreemptWorkflows"`

	// VetoedActionTypes actions that are currently blocked
	VetoedActionTypes []string `json:"vetoedActionTypes,omitempty"`
}

// ============================================================================
// KB-18 GOVERNANCE TYPES (RUNTIME)
// ============================================================================

// ClinicalAction represents a clinical action requiring governance classification.
type ClinicalAction struct {
	// ActionID unique identifier
	ActionID string `json:"actionId"`

	// ActionType categorizes the action (medication_order, procedure_order, etc.)
	ActionType string `json:"actionType"`

	// PatientID the action relates to
	PatientID string `json:"patientId"`

	// EncounterID current encounter
	EncounterID string `json:"encounterId,omitempty"`

	// RequestedBy user/system requesting the action
	RequestedBy string `json:"requestedBy"`

	// RequestedAt timestamp
	RequestedAt time.Time `json:"requestedAt"`

	// Details action-specific data
	Details map[string]interface{} `json:"details,omitempty"`

	// Description human-readable description
	Description string `json:"description,omitempty"`
}

// GovernanceClassification represents CQL-generated governance classification.
type GovernanceClassification struct {
	// Level of governance required (routine, elevated, high, critical)
	Level string `json:"level"`

	// Reason for the classification
	Reason string `json:"reason,omitempty"`

	// RiskScore numeric risk assessment (0.0-1.0)
	RiskScore float64 `json:"riskScore"`

	// SafetyFlags that influenced classification
	SafetyFlags []string `json:"safetyFlags,omitempty"`

	// TriggeredRules that led to this classification
	TriggeredRules []string `json:"triggeredRules,omitempty"`

	// EscalationPath ordered list of roles/steps if escalation is needed
	EscalationPath []string `json:"escalationPath,omitempty"`

	// ClassifiedAt timestamp
	ClassifiedAt time.Time `json:"classifiedAt,omitempty"`

	// ClassifiedBy system/user that performed classification
	ClassifiedBy string `json:"classifiedBy,omitempty"`
}

// ApprovalRequirement specifies approval needs for a clinical action.
type ApprovalRequirement struct {
	// ApprovalLevel (none, single, dual, committee)
	ApprovalLevel string `json:"approvalLevel"`

	// ApproverRole required role for approval
	ApproverRole string `json:"approverRole"`

	// TimeoutMinutes before auto-escalation
	TimeoutMinutes int `json:"timeoutMinutes,omitempty"`

	// EscalationPath ordered list of roles/steps if timeout occurs
	EscalationPath []string `json:"escalationPath,omitempty"`

	// RequiresReason if approver must provide reason
	RequiresReason bool `json:"requiresReason"`

	// AllowDelegation if approver can delegate
	AllowDelegation bool `json:"allowDelegation"`

	// PolicyID that determined this requirement
	PolicyID string `json:"policyId,omitempty"`

	// PolicyVersion of the policy
	PolicyVersion string `json:"policyVersion,omitempty"`

	// AutoApprove if action can be auto-approved
	AutoApprove bool `json:"autoApprove"`
}

// ApprovalRequest represents a request for governance approval.
type ApprovalRequest struct {
	// ActionID unique identifier for the action requiring approval
	ActionID string `json:"actionId"`

	// PatientID the action relates to
	PatientID string `json:"patientId"`

	// EncounterID current encounter
	EncounterID string `json:"encounterId,omitempty"`

	// ActionType categorizes the action (medication_order, procedure_order, etc.)
	ActionType string `json:"actionType"`

	// ActionDescription human-readable description
	ActionDescription string `json:"actionDescription,omitempty"`

	// RequestedBy user/system requesting approval
	RequestedBy string `json:"requestedBy"`

	// RequestedAt timestamp
	RequestedAt time.Time `json:"requestedAt"`

	// ApprovalLevel required (none, single, dual, committee)
	ApprovalLevel string `json:"approvalLevel,omitempty"`

	// ApproverRole required for approval
	ApproverRole string `json:"approverRole,omitempty"`

	// ActionDetails specific details about the action
	ActionDetails map[string]interface{} `json:"actionDetails,omitempty"`

	// Classification governance classification
	Classification string `json:"classification,omitempty"`

	// Justification for the action
	Justification string `json:"justification,omitempty"`

	// UrgencyLevel (routine, urgent, emergent)
	UrgencyLevel string `json:"urgencyLevel,omitempty"`

	// Urgency level (0=routine, 10=immediate)
	Urgency int `json:"urgency"`

	// RiskLevel of the action (0=minimal, 10=critical)
	RiskLevel int `json:"riskLevel"`

	// SupportingEvidence for the request
	SupportingEvidence []string `json:"supportingEvidence,omitempty"`

	// Metadata action-specific data
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ApprovalSubmission represents the result of submitting for approval.
type ApprovalSubmission struct {
	// SubmissionID unique identifier for this submission
	SubmissionID string `json:"submissionId"`

	// ActionID that was submitted
	ActionID string `json:"actionId"`

	// Status of the submission (pending, approved, rejected, escalated)
	Status string `json:"status"`

	// WorkflowInstanceID if a workflow was triggered
	WorkflowInstanceID string `json:"workflowInstanceId,omitempty"`

	// SubmittedAt timestamp
	SubmittedAt time.Time `json:"submittedAt"`

	// ExpiresAt when approval window closes
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	// AssignedTo user assigned to approve
	AssignedTo string `json:"assignedTo,omitempty"`

	// AssignedApprovers list of approvers
	AssignedApprovers []string `json:"assignedApprovers,omitempty"`

	// EscalationTime when to escalate if no decision
	EscalationTime *time.Time `json:"escalationTime,omitempty"`

	// AuditTrailID for tracking
	AuditTrailID string `json:"auditTrailId,omitempty"`

	// RequiredApprovals count needed
	RequiredApprovals int `json:"requiredApprovals"`

	// CurrentApprovals count received
	CurrentApprovals int `json:"currentApprovals"`

	// NextAction suggested next step
	NextAction string `json:"nextAction,omitempty"`
}

// PendingApproval represents an approval awaiting decision.
type PendingApproval struct {
	// SubmissionID unique identifier
	SubmissionID string `json:"submissionId"`

	// ActionID the action requiring approval
	ActionID string `json:"actionId"`

	// PatientID the action relates to
	PatientID string `json:"patientId"`

	// PatientName human-readable patient name for display
	PatientName string `json:"patientName,omitempty"`

	// ActionType categorizes the action
	ActionType string `json:"actionType"`

	// ActionDescription human-readable description
	ActionDescription string `json:"actionDescription,omitempty"`

	// ActionSummary brief summary of the action
	ActionSummary string `json:"actionSummary,omitempty"`

	// RequestedBy who made the request
	RequestedBy string `json:"requestedBy"`

	// RequestedAt timestamp
	RequestedAt time.Time `json:"requestedAt"`

	// ApprovalLevel required level (none, single, dual, committee)
	ApprovalLevel string `json:"approvalLevel,omitempty"`

	// UrgencyLevel (routine, urgent, emergent)
	UrgencyLevel string `json:"urgencyLevel,omitempty"`

	// Classification governance classification level
	Classification string `json:"classification,omitempty"`

	// ExpiresAt when approval window closes
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	// EscalationTime when escalation will occur
	EscalationTime *time.Time `json:"escalationTime,omitempty"`

	// RiskScore numeric risk assessment (0.0-1.0)
	RiskScore float64 `json:"riskScore,omitempty"`

	// SafetyFlags that influenced classification
	SafetyFlags []string `json:"safetyFlags,omitempty"`

	// AssignedTo current approver
	AssignedTo string `json:"assignedTo,omitempty"`

	// Priority calculated priority
	Priority int `json:"priority"`
}

// ApprovalDecision represents an approver's decision on an approval request.
type ApprovalDecision struct {
	// Decision outcome (approved, denied, escalated)
	Decision string `json:"decision"`

	// DecisionBy who made the decision
	DecisionBy string `json:"decisionBy"`

	// DecisionAt timestamp
	DecisionAt time.Time `json:"decisionAt,omitempty"`

	// Rationale for the decision
	Rationale string `json:"rationale,omitempty"`

	// Conditions any conditions attached to approval
	Conditions []string `json:"conditions,omitempty"`

	// DelegatedFrom if this was a delegated decision
	DelegatedFrom string `json:"delegatedFrom,omitempty"`
}

// AuditRecord represents an immutable audit trail entry.
type AuditRecord struct {
	// AuditID unique identifier
	AuditID string `json:"auditId"`

	// SubmissionID if related to an approval
	SubmissionID string `json:"submissionId,omitempty"`

	// ActionID the action being audited
	ActionID string `json:"actionId"`

	// PatientID related patient
	PatientID string `json:"patientId"`

	// EventType categorizes the audit event
	EventType string `json:"eventType"`

	// EventTime when the event occurred
	EventTime time.Time `json:"eventTime"`

	// ActorID who performed the action
	ActorID string `json:"actorId"`

	// ActorRole role of the actor
	ActorRole string `json:"actorRole,omitempty"`

	// Decision if this was a decision event
	Decision string `json:"decision,omitempty"`

	// Rationale for the decision
	Rationale string `json:"rationale,omitempty"`

	// Hash for integrity verification
	Hash string `json:"hash,omitempty"`

	// PreviousHash links to prior record
	PreviousHash string `json:"previousHash,omitempty"`

	// Immutable flag (always true for audit records)
	Immutable bool `json:"immutable"`
}

// GovernancePolicy defines approval rules and escalation paths.
type GovernancePolicy struct {
	// PolicyID unique identifier
	PolicyID string `json:"policyId"`

	// Version of the policy
	Version string `json:"version"`

	// Name human-readable name
	Name string `json:"name"`

	// Description of the policy purpose
	Description string `json:"description,omitempty"`

	// Levels approval level definitions
	Levels []ApprovalLevelDefinition `json:"levels,omitempty"`

	// DefaultTimeout in minutes for approval decisions
	DefaultTimeout int `json:"defaultTimeout,omitempty"`

	// EscalationEnabled if auto-escalation is active
	EscalationEnabled bool `json:"escalationEnabled"`

	// AuditRetention days to retain audit records
	AuditRetention int `json:"auditRetention,omitempty"`

	// EffectiveFrom when policy becomes active
	EffectiveFrom *time.Time `json:"effectiveFrom,omitempty"`

	// EffectiveTo when policy expires
	EffectiveTo *time.Time `json:"effectiveTo,omitempty"`
}

// ApprovalLevelDefinition defines an approval level in a governance policy.
type ApprovalLevelDefinition struct {
	// Level identifier (e.g., "routine", "elevated", "high", "critical")
	Level string `json:"level"`

	// Name human-readable name
	Name string `json:"name"`

	// Description of when this level applies
	Description string `json:"description,omitempty"`

	// ApproverRoles roles that can approve at this level
	ApproverRoles []string `json:"approverRoles,omitempty"`

	// TimeoutMins before escalation
	TimeoutMins int `json:"timeoutMins,omitempty"`

	// EscalatesTo next level if timeout
	EscalatesTo string `json:"escalatesTo,omitempty"`
}
