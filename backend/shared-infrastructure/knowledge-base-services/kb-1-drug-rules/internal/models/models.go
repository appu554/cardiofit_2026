// Package models defines data structures matching the OpenAPI specification.
package models

import "time"

// ============================================================================
// SAFETY STATUS - Critical for fail-safe clinical decisions
// ============================================================================

// SafetyStatus indicates the governance status of a dose calculation result.
// CRITICAL: This field determines whether the dose can be used clinically.
// The principle is: ABSENCE OF DATA = BLOCK, not "allow with default".
type SafetyStatus string

const (
	// SafetyStatusGoverned - Rule found, data complete, safe to use for clinical decisions
	SafetyStatusGoverned SafetyStatus = "GOVERNED"

	// SafetyStatusDataIncomplete - Drug found but specific adjustment data is MISSING
	// CRITICAL: This MUST block prescribing - missing data is NOT permission
	// Example: Metformin found but no renal adjustment tiers → BLOCK
	SafetyStatusDataIncomplete SafetyStatus = "DATA_INCOMPLETE"

	// SafetyStatusNotFound - Drug not in governed formulary at all
	// CRITICAL: This MUST block prescribing - ungoverned drugs cannot be dosed
	SafetyStatusNotFound SafetyStatus = "NOT_FOUND"

	// SafetyStatusDraft - Rule exists but is DRAFT status, not approved for clinical use
	SafetyStatusDraft SafetyStatus = "DRAFT"
)

// IsBlockRequired returns true if this safety status requires blocking the prescription
func (s SafetyStatus) IsBlockRequired() bool {
	return s != SafetyStatusGoverned
}

// ============================================================================
// COMMON MODELS
// ============================================================================

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Service   string    `json:"service"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message,omitempty"`
	Code      string `json:"code,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Success   bool   `json:"success"`
}

// PatientParameters contains patient demographic and clinical data.
type PatientParameters struct {
	Age             int     `json:"age" binding:"required,min=0,max=150"`
	Gender          string  `json:"gender" binding:"required,oneof=M F male female"`
	WeightKg        float64 `json:"weight_kg" binding:"required,min=0.1"`
	HeightCm        float64 `json:"height_cm" binding:"required,min=10"`
	SerumCreatinine float64 `json:"serum_creatinine,omitempty"`
	EGFR            float64 `json:"egfr,omitempty"`
	ChildPughScore  int     `json:"child_pugh_score,omitempty"`
	ChildPughClass  string  `json:"child_pugh_class,omitempty"`
}

// ============================================================================
// FIXED-DOSE COMBINATION (FDC) MODELS
// ============================================================================

// FDCComponent represents one component of a fixed-dose combination.
type FDCComponent struct {
	DrugName  string  `json:"drug_name"`
	DrugClass string  `json:"drug_class"` // ACE_INHIBITOR, ARB, CCB, BETA_BLOCKER, THIAZIDE, MRA, ALPHA_BLOCKER
	DoseMg    float64 `json:"dose_mg"`
}

// FDCMapping maps an FDC product to its constituent components.
// A patient taking one FDC pill is adherent to ALL constituent drug classes.
type FDCMapping struct {
	FDCName    string         `json:"fdc_name"`
	Components []FDCComponent `json:"components"`
	IsHTN      bool           `json:"is_htn"` // true if all components are antihypertensive
}

// ============================================================================
// OPTIMISED DOSE MODELS
// ============================================================================

// OptimisedDose defines the maximum recommended dose for an antihypertensive.
// Used by resistant HTN detection to determine if a drug is at optimised dose.
type OptimisedDose struct {
	DrugName     string  `json:"drug_name"`
	DrugClass    string  `json:"drug_class"`
	MaxDoseMg    float64 `json:"max_dose_mg"`      // maximum recommended antihypertensive dose
	StandardDose float64 `json:"standard_dose_mg"`  // typical maintenance dose
}

// OptimisedDoseCheckResponse is the API response for optimised dose check.
type OptimisedDoseCheckResponse struct {
	DrugName    string  `json:"drug_name"`
	DrugClass   string  `json:"drug_class"`
	CurrentDose float64 `json:"current_dose_mg"`
	MaxDose     float64 `json:"max_dose_mg"`
	IsOptimised bool    `json:"is_optimised"`
}

// ============================================================================
// DOSE CALCULATION MODELS
// ============================================================================

// DoseCalculationRequest is the request for general dose calculation.
type DoseCalculationRequest struct {
	RxNormCode string            `json:"rxnorm_code" binding:"required"`
	Patient    PatientParameters `json:"patient" binding:"required"`
	Indication string            `json:"indication,omitempty"`
}

// DoseCalculationResult is the response for dose calculation.
type DoseCalculationResult struct {
	Success              bool                   `json:"success"`
	DrugName             string                 `json:"drug_name"`
	RxNormCode           string                 `json:"rxnorm_code"`
	RecommendedDose      float64                `json:"recommended_dose"`
	Unit                 string                 `json:"unit"`
	Frequency            string                 `json:"frequency"`
	Route                string                 `json:"route"`
	DosingMethod         string                 `json:"dosing_method"`
	DoseRange            *DoseRange             `json:"dose_range,omitempty"`
	RenalAdjustment      *AdjustmentInfo        `json:"renal_adjustment,omitempty"`
	HepaticAdjustment    *AdjustmentInfo        `json:"hepatic_adjustment,omitempty"`
	AgeAdjustment        *AdjustmentInfo        `json:"age_adjustment,omitempty"`
	CalculatedParameters *CalculatedParams      `json:"calculated_parameters,omitempty"`
	Alerts               []SafetyAlert          `json:"alerts,omitempty"`
	Monitoring           []string               `json:"monitoring,omitempty"`
	Error                string                 `json:"error,omitempty"`
	ErrorCode            string                 `json:"error_code,omitempty"`
	Source               *DoseSourceAttribution `json:"source,omitempty"`
	// KB-4 Patient Safety verdict - populated when KB-4 integration is enabled
	SafetyVerdict        *KB4SafetyVerdict      `json:"safety_verdict,omitempty"`
}

// KB4SafetyVerdict represents the safety evaluation from KB-4 Patient Safety Service.
// This is the clinical brain's final safety assessment before allowing a prescription.
type KB4SafetyVerdict struct {
	Safe             bool              `json:"safe"`               // Overall safety verdict
	BlockPrescribing bool              `json:"block_prescribing"`  // Hard stop - cannot prescribe
	RequiresAction   bool              `json:"requires_action"`    // Clinician acknowledgment required
	IsHighAlertDrug  bool              `json:"is_high_alert_drug"` // ISMP high-alert medication
	TotalAlerts      int               `json:"total_alerts"`
	CriticalAlerts   int               `json:"critical_alerts"`
	HighAlerts       int               `json:"high_alerts"`
	Alerts           []KB4SafetyAlert  `json:"alerts,omitempty"`
	CheckedAt        time.Time         `json:"checked_at"`
	KB4RequestID     string            `json:"kb4_request_id,omitempty"`
}

// KB4SafetyAlert represents an individual safety alert from KB-4.
type KB4SafetyAlert struct {
	Type                   string   `json:"type"`                     // BLACK_BOX_WARNING, CONTRAINDICATION, etc.
	Severity               string   `json:"severity"`                 // CRITICAL, HIGH, MODERATE, LOW
	Title                  string   `json:"title"`
	Message                string   `json:"message"`
	RequiresAcknowledgment bool     `json:"requires_acknowledgment"`
	CanOverride            bool     `json:"can_override"`
	ClinicalRationale      string   `json:"clinical_rationale,omitempty"`
	Recommendations        []string `json:"recommendations,omitempty"`
	References             []string `json:"references,omitempty"`
}

// DoseRange represents min/max dose range.
type DoseRange struct {
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Unit string  `json:"unit"`
}

// AdjustmentInfo contains dose adjustment details.
type AdjustmentInfo struct {
	Applied         bool    `json:"applied"`
	Reason          string  `json:"reason,omitempty"`
	Factor          float64 `json:"factor,omitempty"`
	OriginalDose    float64 `json:"original_dose,omitempty"`
	AdjustedDose    float64 `json:"adjusted_dose,omitempty"`
	Contraindicated bool    `json:"contraindicated,omitempty"`
}

// CalculatedParams contains derived patient parameters.
type CalculatedParams struct {
	BSA         float64 `json:"bsa,omitempty"`
	IBW         float64 `json:"ibw,omitempty"`
	AdjBW       float64 `json:"adj_bw,omitempty"`
	BMI         float64 `json:"bmi,omitempty"`
	CrCl        float64 `json:"crcl,omitempty"`
	EGFR        float64 `json:"egfr,omitempty"`
	CKDStage    string  `json:"ckd_stage,omitempty"`
	IsPediatric bool    `json:"is_pediatric"`
	IsGeriatric bool    `json:"is_geriatric"`
	IsObese     bool    `json:"is_obese"`
}

// SafetyAlert represents a clinical safety warning.
type SafetyAlert struct {
	AlertType      string `json:"alert_type"`
	Severity       string `json:"severity"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation,omitempty"`
}

// ============================================================================
// WEIGHT-BASED DOSING
// ============================================================================

// WeightBasedRequest is the request for weight-based dose calculation.
type WeightBasedRequest struct {
	RxNormCode string            `json:"rxnorm_code,omitempty"`
	Patient    PatientParameters `json:"patient" binding:"required"`
	DosePerKg  float64           `json:"dose_per_kg,omitempty"`
}

// ============================================================================
// BSA-BASED DOSING
// ============================================================================

// BSABasedRequest is the request for BSA-based dose calculation.
type BSABasedRequest struct {
	Patient   PatientParameters `json:"patient" binding:"required"`
	DosePerM2 float64           `json:"dose_per_m2" binding:"required"`
}

// BSADoseResult is the response for BSA-based calculation.
type BSADoseResult struct {
	Success        bool    `json:"success"`
	BSA            float64 `json:"bsa"`
	DosePerM2      float64 `json:"dose_per_m2"`
	CalculatedDose float64 `json:"calculated_dose"`
	Unit           string  `json:"unit"`
	FormulaUsed    string  `json:"formula_used"`
}

// ============================================================================
// PEDIATRIC DOSING
// ============================================================================

// PediatricRequest is the request for pediatric dose calculation.
type PediatricRequest struct {
	RxNormCode string            `json:"rxnorm_code" binding:"required"`
	Patient    PatientParameters `json:"patient" binding:"required"`
}

// PediatricDoseResult is the response for pediatric calculation.
// CRITICAL: Always check SafetyStatus before using this result clinically.
// SafetyStatus != GOVERNED means this calculation MUST NOT be used for prescribing.
type PediatricDoseResult struct {
	Success          bool                   `json:"success"`
	SafetyStatus     SafetyStatus           `json:"safety_status"`     // GOVERNED, DATA_INCOMPLETE, NOT_FOUND
	BlockPrescribing bool                   `json:"block_prescribing"` // True if prescribing should be blocked
	AgeCategory      string                 `json:"age_category"`
	DrugName         string                 `json:"drug_name"`
	RecommendedDose  float64                `json:"recommended_dose"`
	Unit             string                 `json:"unit"`
	DosePerKg        float64                `json:"dose_per_kg,omitempty"`
	MaxDose          float64                `json:"max_dose,omitempty"`
	Contraindicated  bool                   `json:"contraindicated,omitempty"` // True if drug is contraindicated
	Warnings         []string               `json:"warnings,omitempty"`
	Error            string                 `json:"error,omitempty"`
	Source           *DoseSourceAttribution `json:"source,omitempty"`
}

// ============================================================================
// RENAL-ADJUSTED DOSING
// ============================================================================

// RenalAdjustedRequest is the request for renal-adjusted dose calculation.
type RenalAdjustedRequest struct {
	RxNormCode string            `json:"rxnorm_code" binding:"required"`
	Patient    PatientParameters `json:"patient" binding:"required"`
}

// RenalDoseResult is the response for renal-adjusted calculation.
// CRITICAL: Always check SafetyStatus before using this result clinically.
// SafetyStatus != GOVERNED means this calculation MUST NOT be used for prescribing.
type RenalDoseResult struct {
	Success                bool                   `json:"success"`
	SafetyStatus           SafetyStatus           `json:"safety_status"`           // GOVERNED, DATA_INCOMPLETE, NOT_FOUND
	BlockPrescribing       bool                   `json:"block_prescribing"`       // True if prescribing should be blocked
	DrugName               string                 `json:"drug_name"`
	RxNormCode             string                 `json:"rxnorm_code"`
	OriginalDose           float64                `json:"original_dose"`
	AdjustedDose           float64                `json:"adjusted_dose"`
	Unit                   string                 `json:"unit"`
	EGFR                   float64                `json:"egfr"`
	CrCl                   float64                `json:"crcl,omitempty"`
	CKDStage               string                 `json:"ckd_stage"`
	CKDDescription         string                 `json:"ckd_description,omitempty"`
	AdjustmentFactor       float64                `json:"adjustment_factor"`
	Recommendation         string                 `json:"recommendation,omitempty"`
	Contraindicated        bool                   `json:"contraindicated"`
	ContraindicationReason string                 `json:"contraindication_reason,omitempty"`
	Frequency              string                 `json:"frequency,omitempty"`
	Error                  string                 `json:"error,omitempty"`
	Source                 *DoseSourceAttribution `json:"source,omitempty"`
}

// ============================================================================
// HEPATIC-ADJUSTED DOSING
// ============================================================================

// HepaticAdjustedRequest is the request for hepatic-adjusted dose calculation.
type HepaticAdjustedRequest struct {
	RxNormCode string            `json:"rxnorm_code" binding:"required"`
	Patient    PatientParameters `json:"patient" binding:"required"`
}

// HepaticDoseResult is the response for hepatic-adjusted calculation.
// CRITICAL: Always check SafetyStatus before using this result clinically.
// SafetyStatus != GOVERNED means this calculation MUST NOT be used for prescribing.
type HepaticDoseResult struct {
	Success          bool                   `json:"success"`
	SafetyStatus     SafetyStatus           `json:"safety_status"`     // GOVERNED, DATA_INCOMPLETE, NOT_FOUND
	BlockPrescribing bool                   `json:"block_prescribing"` // True if prescribing should be blocked
	DrugName         string                 `json:"drug_name"`
	RxNormCode       string                 `json:"rxnorm_code"`
	OriginalDose     float64                `json:"original_dose"`
	AdjustedDose     float64                `json:"adjusted_dose"`
	Unit             string                 `json:"unit"`
	ChildPughClass   string                 `json:"child_pugh_class"`
	AdjustmentFactor float64                `json:"adjustment_factor"`
	Recommendation   string                 `json:"recommendation,omitempty"`
	Contraindicated  bool                   `json:"contraindicated"`
	Error            string                 `json:"error,omitempty"`
	Source           *DoseSourceAttribution `json:"source,omitempty"`
}

// ============================================================================
// GERIATRIC DOSING
// ============================================================================

// GeriatricRequest is the request for geriatric dose calculation.
type GeriatricRequest struct {
	RxNormCode string            `json:"rxnorm_code" binding:"required"`
	Patient    PatientParameters `json:"patient" binding:"required"`
}

// GeriatricDoseResult is the response for geriatric calculation.
// CRITICAL: Always check SafetyStatus before using this result clinically.
// SafetyStatus != GOVERNED means this calculation MUST NOT be used for prescribing.
type GeriatricDoseResult struct {
	Success          bool                   `json:"success"`
	SafetyStatus     SafetyStatus           `json:"safety_status"`     // GOVERNED, DATA_INCOMPLETE, NOT_FOUND
	BlockPrescribing bool                   `json:"block_prescribing"` // True if prescribing should be blocked
	DrugName         string                 `json:"drug_name"`
	RecommendedDose  float64                `json:"recommended_dose"`
	Unit             string                 `json:"unit"`
	AdjustmentNotes  string                 `json:"adjustment_notes,omitempty"`
	BeersWarning     string                 `json:"beers_warning,omitempty"`
	Contraindicated  bool                   `json:"contraindicated,omitempty"` // True if drug should be avoided
	Warnings         []string               `json:"warnings,omitempty"`
	Error            string                 `json:"error,omitempty"`
	Source           *DoseSourceAttribution `json:"source,omitempty"`
}

// ============================================================================
// PATIENT PARAMETER CALCULATIONS
// ============================================================================

// BSARequest is the request for BSA calculation.
type BSARequest struct {
	HeightCm float64 `json:"height_cm" binding:"required,min=10"`
	WeightKg float64 `json:"weight_kg" binding:"required,min=0.1"`
}

// BSAResponse is the response for BSA calculation.
type BSAResponse struct {
	BSA      float64 `json:"bsa"`
	Formula  string  `json:"formula"`
	HeightCm float64 `json:"height_cm"`
	WeightKg float64 `json:"weight_kg"`
}

// IBWRequest is the request for IBW calculation.
type IBWRequest struct {
	HeightCm float64 `json:"height_cm" binding:"required,min=10"`
	Gender   string  `json:"gender" binding:"required,oneof=M F"`
}

// IBWResponse is the response for IBW calculation.
type IBWResponse struct {
	IBWKg    float64 `json:"ibw_kg"`
	Formula  string  `json:"formula"`
	HeightCm float64 `json:"height_cm"`
	Gender   string  `json:"gender"`
}

// CrClRequest is the request for CrCl calculation.
type CrClRequest struct {
	Age             int     `json:"age" binding:"required,min=0"`
	WeightKg        float64 `json:"weight_kg" binding:"required,min=0.1"`
	SerumCreatinine float64 `json:"serum_creatinine" binding:"required,min=0.1"`
	Gender          string  `json:"gender" binding:"required,oneof=M F"`
}

// CrClResponse is the response for CrCl calculation.
type CrClResponse struct {
	CrCl           float64 `json:"crcl"`
	Formula        string  `json:"formula"`
	CKDStage       string  `json:"ckd_stage"`
	Interpretation string  `json:"interpretation"`
}

// EGFRRequest is the request for eGFR calculation.
type EGFRRequest struct {
	Age             int     `json:"age" binding:"required,min=0"`
	SerumCreatinine float64 `json:"serum_creatinine" binding:"required,min=0.1"`
	Gender          string  `json:"gender" binding:"required,oneof=M F"`
}

// EGFRResponse is the response for eGFR calculation.
type EGFRResponse struct {
	EGFR           float64 `json:"egfr"`
	Formula        string  `json:"formula"`
	CKDStage       string  `json:"ckd_stage"`
	CKDDescription string  `json:"ckd_description"`
	Interpretation string  `json:"interpretation"`
}

// ============================================================================
// DOSE VALIDATION
// ============================================================================

// DoseValidationRequest is the request for dose validation.
type DoseValidationRequest struct {
	RxNormCode   string            `json:"rxnorm_code" binding:"required"`
	ProposedDose float64           `json:"proposed_dose" binding:"required"`
	Unit         string            `json:"unit"`
	Frequency    string            `json:"frequency,omitempty"`
	Patient      PatientParameters `json:"patient" binding:"required"`
}

// DoseValidationResult is the response for dose validation.
type DoseValidationResult struct {
	Valid            bool                   `json:"valid"`
	DrugName         string                 `json:"drug_name"`
	ProposedDose     float64                `json:"proposed_dose"`
	RecommendedDose  float64                `json:"recommended_dose"`
	MaxSingleDose    float64                `json:"max_single_dose"`
	MaxDailyDose     float64                `json:"max_daily_dose"`
	ValidationStatus string                 `json:"validation_status"`
	Alerts           []SafetyAlert          `json:"alerts,omitempty"`
	Reasons          []string               `json:"reasons,omitempty"`
	Source           *DoseSourceAttribution `json:"source,omitempty"`
}

// MaxDoseResponse is the response for max dose query.
type MaxDoseResponse struct {
	RxNormCode    string                 `json:"rxnorm_code"`
	DrugName      string                 `json:"drug_name"`
	MaxSingleDose float64                `json:"max_single_dose"`
	MaxDailyDose  float64                `json:"max_daily_dose"`
	Unit          string                 `json:"unit"`
	Notes         string                 `json:"notes,omitempty"`
	Source        *DoseSourceAttribution `json:"source,omitempty"`
}

// ============================================================================
// DRUG RULES
// ============================================================================

// DrugRule represents a complete drug dosing rule.
type DrugRule struct {
	RxNormCode          string             `json:"rxnorm_code"`
	DrugName            string             `json:"drug_name"`
	DrugClass           string             `json:"drug_class"`
	Category            string             `json:"category"`
	DosingMethod        string             `json:"dosing_method"`
	DefaultDose         float64            `json:"default_dose"`
	MinDose             float64            `json:"min_dose"`
	MaxDose             float64            `json:"max_dose"`
	MaxDailyDose        float64            `json:"max_daily_dose"`
	DoseUnit            string             `json:"dose_unit"`
	Frequency           string             `json:"frequency"`
	Route               string             `json:"route"`
	DosePerKg           float64            `json:"dose_per_kg,omitempty"`
	IsHighAlert         bool               `json:"is_high_alert"`
	IsNarrowTI          bool               `json:"is_narrow_ti"`
	HasBlackBoxWarning  bool               `json:"has_black_box_warning"`
	BlackBoxWarning     string             `json:"black_box_warning,omitempty"`
	IsBeersList         bool               `json:"is_beers_list"`
	BeersCriteria       string             `json:"beers_criteria,omitempty"`
	RenalAdjustments    []RenalAdjustment  `json:"renal_adjustments,omitempty"`
	HepaticAdjustments  []HepaticAdjustment `json:"hepatic_adjustments,omitempty"`
	AgeAdjustments      []AgeAdjustment    `json:"age_adjustments,omitempty"`
	Monitoring          []string           `json:"monitoring_parameters,omitempty"`
	Contraindications   []string           `json:"contraindications,omitempty"`
}

// RenalAdjustment represents renal dose adjustment rules.
type RenalAdjustment struct {
	EGFRMin          float64 `json:"egfr_min"`
	EGFRMax          float64 `json:"egfr_max"`
	AdjustmentFactor float64 `json:"adjustment_factor"`
	MaxDose          float64 `json:"max_dose,omitempty"`
	Recommendation   string  `json:"recommendation"`
	Contraindicated  bool    `json:"contraindicated"`
}

// HepaticAdjustment represents hepatic dose adjustment rules.
type HepaticAdjustment struct {
	ChildPughClass   string  `json:"child_pugh_class"`
	AdjustmentFactor float64 `json:"adjustment_factor"`
	MaxDose          float64 `json:"max_dose,omitempty"`
	Recommendation   string  `json:"recommendation"`
	Contraindicated  bool    `json:"contraindicated"`
}

// AgeAdjustment represents age-based dose adjustment rules.
type AgeAdjustment struct {
	MinAge           int     `json:"min_age"`
	MaxAge           int     `json:"max_age"`
	AdjustmentFactor float64 `json:"adjustment_factor"`
	MaxDose          float64 `json:"max_dose,omitempty"`
	Recommendation   string  `json:"recommendation"`
	Contraindicated  bool    `json:"contraindicated"`
}

// Note: DrugRuleSummary is defined in governed_models.go with fields:
// RxNormCode, DrugName, GenericName, DrugClass, Jurisdiction,
// IsHighAlert, IsNarrowTI, HasBlackBox, Authority, Version

// RulesListResponse is the response for listing all rules.
type RulesListResponse struct {
	Count int                `json:"count"`
	Rules []*DrugRuleSummary `json:"rules"`
}

// RulesSearchResponse is the response for searching rules.
type RulesSearchResponse struct {
	Query   string             `json:"query"`
	Count   int                `json:"count"`
	Results []*DrugRuleSummary `json:"results"`
}

// ============================================================================
// ADJUSTMENT INFO ENDPOINTS
// ============================================================================

// AdjustmentInfoResponse is the response for adjustment info queries.
type AdjustmentInfoResponse struct {
	RxNormCode  string      `json:"rxnorm_code"`
	DrugName    string      `json:"drug_name"`
	Type        string      `json:"type"`
	Adjustments interface{} `json:"adjustments"`
}

// HighAlertCheckResponse is the response for high-alert check.
type HighAlertCheckResponse struct {
	RxNormCode         string                 `json:"rxnorm_code"`
	DrugName           string                 `json:"drug_name"`
	IsHighAlert        bool                   `json:"is_high_alert"`
	IsNarrowTI         bool                   `json:"is_narrow_ti"`
	HasBlackBoxWarning bool                   `json:"has_black_box_warning"`
	BlackBoxWarning    string                 `json:"black_box_warning,omitempty"`
	IsBeersList        bool                   `json:"is_beers_list"`
	BeersCriteria      string                 `json:"beers_criteria,omitempty"`
	Source             *DoseSourceAttribution `json:"source,omitempty"`
}
