package models

import (
	"time"
)

// ============================================================================
// HCC (Hierarchical Condition Category) Models
// Used for CMS Risk Adjustment Factor calculation
// ============================================================================

// HCCCategory represents a single HCC category
type HCCCategory struct {
	ID          string  `json:"id" db:"id"`
	Code        string  `json:"code" db:"code"`                 // e.g., "HCC17", "HCC18"
	Version     string  `json:"version" db:"version"`           // e.g., "V24", "V28"
	Description string  `json:"description" db:"description"`
	Coefficient float64 `json:"coefficient" db:"coefficient"`   // RAF coefficient for this HCC

	// Category metadata
	CategoryType string `json:"category_type" db:"category_type"` // disease, interaction, demographic
	ClinicalArea string `json:"clinical_area" db:"clinical_area"` // diabetes, chf, ckd, etc.

	// Hierarchy information
	SupersededBy []string `json:"superseded_by" db:"superseded_by"` // Higher priority HCCs that trump this one
	Supersedes   []string `json:"supersedes" db:"supersedes"`       // Lower priority HCCs this one trumps

	// Model applicability
	ModelType    string `json:"model_type" db:"model_type"`       // CNA, CND, CFA, CFD, CPA, CPD, INS
	AgeGroup     string `json:"age_group" db:"age_group"`         // adult, child, esrd

	// Timestamps
	EffectiveFrom time.Time  `json:"effective_from" db:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to" db:"effective_to"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// ICD10ToHCCMapping represents a mapping from ICD-10-CM code to HCC category
type ICD10ToHCCMapping struct {
	ID         string `json:"id" db:"id"`
	ICD10Code  string `json:"icd10_code" db:"icd10_code"`   // e.g., "E11.21"
	HCCCode    string `json:"hcc_code" db:"hcc_code"`       // e.g., "HCC18"
	HCCVersion string `json:"hcc_version" db:"hcc_version"` // e.g., "V24"

	// Mapping metadata
	MappingType   string `json:"mapping_type" db:"mapping_type"`     // primary, secondary
	ConditionType string `json:"condition_type" db:"condition_type"` // chronic, acute

	// Validity period
	EffectiveFrom time.Time  `json:"effective_from" db:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to" db:"effective_to"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// HCCHierarchy represents the hierarchy rules between HCC categories
type HCCHierarchy struct {
	ID            string `json:"id" db:"id"`
	HigherHCC     string `json:"higher_hcc" db:"higher_hcc"`         // The trumping HCC
	LowerHCC      string `json:"lower_hcc" db:"lower_hcc"`           // The trumped HCC
	HierarchyType string `json:"hierarchy_type" db:"hierarchy_type"` // severity, specificity
	Version       string `json:"version" db:"version"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// RAFCalculationRequest represents a request to calculate RAF score
type RAFCalculationRequest struct {
	PatientID     string             `json:"patient_id" binding:"required"`
	DiagnosisCodes []string          `json:"diagnosis_codes" binding:"required"` // ICD-10-CM codes
	Demographics  PatientDemographics `json:"demographics" binding:"required"`
	ModelType     string             `json:"model_type"`     // CNA, CND, CFA, CFD, CPA, CPD, INS (default: CNA)
	PaymentYear   int                `json:"payment_year"`   // Year for coefficient lookup
	Version       string             `json:"version"`        // HCC model version (default: V24)
}

// PatientDemographics represents patient demographic information for RAF calculation
type PatientDemographics struct {
	Age               int    `json:"age" binding:"required"`
	Gender            string `json:"gender" binding:"required"` // M, F
	OriginallyDisabled bool  `json:"originally_disabled"`
	DualEligible      bool   `json:"dual_eligible"`
	InstitutionStatus string `json:"institution_status"` // community, institutional, ltc
	ESRD              bool   `json:"esrd"`               // End-Stage Renal Disease
	Medicaid          bool   `json:"medicaid"`
}

// RAFCalculationResult represents the result of a RAF calculation
type RAFCalculationResult struct {
	PatientID       string             `json:"patient_id"`
	TotalRAF        float64            `json:"total_raf"`
	DemographicRAF  float64            `json:"demographic_raf"`
	DiseaseRAF      float64            `json:"disease_raf"`
	InteractionRAF  float64            `json:"interaction_raf"`

	// Detailed breakdown
	Demographics    DemographicFactors `json:"demographics"`
	HCCCategories   []HCCResult        `json:"hcc_categories"`
	Interactions    []InteractionResult `json:"interactions,omitempty"`

	// Hierarchy application
	DroppedHCCs     []DroppedHCC       `json:"dropped_hccs,omitempty"`

	// Metadata
	ModelType       string             `json:"model_type"`
	ModelVersion    string             `json:"model_version"`
	PaymentYear     int                `json:"payment_year"`
	CalculatedAt    time.Time          `json:"calculated_at"`
}

// DemographicFactors represents the demographic components of RAF
type DemographicFactors struct {
	AgeGenderCoefficient float64 `json:"age_gender_coefficient"`
	DisabilityCoefficient float64 `json:"disability_coefficient,omitempty"`
	MedicaidCoefficient   float64 `json:"medicaid_coefficient,omitempty"`
	InstitutionalCoefficient float64 `json:"institutional_coefficient,omitempty"`

	// Demographics used
	AgeGroup string `json:"age_group"`
	Gender   string `json:"gender"`
}

// HCCResult represents a single HCC category result
type HCCResult struct {
	HCCCode       string   `json:"hcc_code"`
	Description   string   `json:"description"`
	Coefficient   float64  `json:"coefficient"`
	SourceCodes   []string `json:"source_codes"` // ICD-10 codes that mapped to this HCC
	ClinicalArea  string   `json:"clinical_area"`
	Applied       bool     `json:"applied"`      // Whether this HCC was used in final calculation
}

// InteractionResult represents an HCC interaction result
type InteractionResult struct {
	InteractionName string   `json:"interaction_name"`
	HCCCodes        []string `json:"hcc_codes"`
	Coefficient     float64  `json:"coefficient"`
	Description     string   `json:"description"`
}

// DroppedHCC represents an HCC that was dropped due to hierarchy rules
type DroppedHCC struct {
	DroppedCode   string `json:"dropped_code"`
	TrumpedByCode string `json:"trumped_by_code"`
	Reason        string `json:"reason"`
}

// HCCBatchRequest represents a batch RAF calculation request
type HCCBatchRequest struct {
	Patients []RAFCalculationRequest `json:"patients" binding:"required"`
}

// HCCBatchResult represents batch RAF calculation results
type HCCBatchResult struct {
	Results      []RAFCalculationResult `json:"results"`
	TotalCount   int                    `json:"total_count"`
	SuccessCount int                    `json:"success_count"`
	ErrorCount   int                    `json:"error_count"`
	Errors       []HCCError             `json:"errors,omitempty"`
}

// HCCError represents an error in HCC processing
type HCCError struct {
	PatientID string `json:"patient_id"`
	Error     string `json:"error"`
	Code      string `json:"code,omitempty"` // The problematic code if applicable
}

// HCCLookupResult represents the result of looking up HCCs for diagnosis codes
type HCCLookupResult struct {
	DiagnosisCode string       `json:"diagnosis_code"`
	Display       string       `json:"display,omitempty"`
	HCCMappings   []HCCMapping `json:"hcc_mappings"`
	Valid         bool         `json:"valid"`
	Message       string       `json:"message,omitempty"`
}

// HCCMapping represents a single HCC mapping result
type HCCMapping struct {
	HCCCode       string  `json:"hcc_code"`
	Description   string  `json:"description"`
	Coefficient   float64 `json:"coefficient"`
	ClinicalArea  string  `json:"clinical_area"`
	Version       string  `json:"version"`
}

// ============================================================================
// HCC Interaction Definitions
// ============================================================================

// HCCInteraction represents disease interaction factors
type HCCInteraction struct {
	ID              string   `json:"id" db:"id"`
	InteractionName string   `json:"interaction_name" db:"interaction_name"`
	RequiredHCCs    []string `json:"required_hccs" db:"required_hccs"` // All must be present
	Coefficient     float64  `json:"coefficient" db:"coefficient"`
	Description     string   `json:"description" db:"description"`
	Version         string   `json:"version" db:"version"`
	ModelType       string   `json:"model_type" db:"model_type"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ============================================================================
// Standard CMS HCC Hierarchies (V24/V28)
// These define which HCCs trump others
// ============================================================================

// HCCHierarchyRule defines a hierarchy rule
type HCCHierarchyRule struct {
	ClinicalArea string   `json:"clinical_area"`
	Hierarchy    []string `json:"hierarchy"` // Ordered from highest to lowest severity
}

// StandardHierarchies returns the standard CMS HCC hierarchies
func StandardHierarchies() []HCCHierarchyRule {
	return []HCCHierarchyRule{
		// Diabetes
		{ClinicalArea: "diabetes", Hierarchy: []string{"HCC17", "HCC18", "HCC19"}},
		// Chronic Kidney Disease
		{ClinicalArea: "ckd", Hierarchy: []string{"HCC136", "HCC137", "HCC138"}},
		// Congestive Heart Failure
		{ClinicalArea: "chf", Hierarchy: []string{"HCC85", "HCC86"}},
		// Depression
		{ClinicalArea: "depression", Hierarchy: []string{"HCC59", "HCC60"}},
		// COPD
		{ClinicalArea: "copd", Hierarchy: []string{"HCC111", "HCC112"}},
		// Vascular Disease
		{ClinicalArea: "vascular", Hierarchy: []string{"HCC107", "HCC108"}},
		// Seizure Disorders
		{ClinicalArea: "seizure", Hierarchy: []string{"HCC79", "HCC80"}},
		// Stroke
		{ClinicalArea: "stroke", Hierarchy: []string{"HCC99", "HCC100"}},
		// Cancer
		{ClinicalArea: "cancer", Hierarchy: []string{"HCC8", "HCC9", "HCC10", "HCC11", "HCC12"}},
		// Substance Abuse
		{ClinicalArea: "substance", Hierarchy: []string{"HCC54", "HCC55", "HCC56"}},
		// Schizophrenia
		{ClinicalArea: "schizophrenia", Hierarchy: []string{"HCC57", "HCC58"}},
	}
}
