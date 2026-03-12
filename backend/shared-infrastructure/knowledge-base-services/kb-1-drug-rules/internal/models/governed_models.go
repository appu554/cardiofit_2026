package models

import "time"

// =============================================================================
// GOVERNED DRUG RULE - Primary Structure
// =============================================================================

// GovernedDrugRule represents a fully governed drug dosing rule
// This is the authoritative structure for all drug rules in KB-1
// Risk Level: CRITICAL - These rules compute doses that get administered
type GovernedDrugRule struct {
	Drug       DrugIdentification `json:"drug" yaml:"drug"`
	Dosing     DosingRules        `json:"dosing" yaml:"dosing"`
	Safety     SafetyInfo         `json:"safety" yaml:"safety"`
	Governance GovernanceMetadata `json:"governance" yaml:"governance"`
}

// =============================================================================
// DRUG IDENTIFICATION
// =============================================================================

// DrugIdentification contains drug identification codes from multiple terminology systems
type DrugIdentification struct {
	RxNormCode  string `json:"rxnorm_code" yaml:"rxnormCode"`
	Name        string `json:"name" yaml:"name"`
	GenericName string `json:"generic_name" yaml:"genericName"`
	DrugClass   string `json:"drug_class,omitempty" yaml:"drugClass,omitempty"`
	ATCCode     string `json:"atc_code,omitempty" yaml:"atcCode,omitempty"`
	SNOMEDCode  string `json:"snomed_code,omitempty" yaml:"snomedCode,omitempty"`
	AMTCode     string `json:"amt_code,omitempty" yaml:"amtCode,omitempty"` // Australian Medicines Terminology
	NDC         string `json:"ndc,omitempty" yaml:"ndc,omitempty"`          // National Drug Code (FDA)
}

// =============================================================================
// DOSING RULES
// =============================================================================

// DosingRules contains all dosing information for a drug
type DosingRules struct {
	PrimaryMethod string             `json:"primary_method" yaml:"primaryMethod"` // FIXED, WEIGHT_BASED, BSA_BASED
	Adult         *AdultDosing       `json:"adult,omitempty" yaml:"adult,omitempty"`
	WeightBased   *WeightBasedDosing `json:"weight_based,omitempty" yaml:"weightBased,omitempty"`
	BSABased      *BSABasedDosing    `json:"bsa_based,omitempty" yaml:"bsaBased,omitempty"`
	Pediatric     *PediatricDosing   `json:"pediatric,omitempty" yaml:"pediatric,omitempty"`
	Geriatric     *GeriatricDosing   `json:"geriatric,omitempty" yaml:"geriatric,omitempty"`
	Renal         *RenalDosing       `json:"renal,omitempty" yaml:"renal,omitempty"`
	Hepatic       *HepaticDosing     `json:"hepatic,omitempty" yaml:"hepatic,omitempty"`
	Titration     *TitrationSchedule `json:"titration,omitempty" yaml:"titration,omitempty"`
}

// AdultDosing contains standard adult dosing information
type AdultDosing struct {
	Standard  []StandardDose `json:"standard,omitempty" yaml:"standard,omitempty"`
	MaxDaily  float64        `json:"max_daily,omitempty" yaml:"maxDaily,omitempty"`
	MaxSingle float64        `json:"max_single,omitempty" yaml:"maxSingle,omitempty"`
	MaxUnit   string         `json:"max_unit,omitempty" yaml:"maxUnit,omitempty"`
}

// StandardDose represents a single dosing recommendation
type StandardDose struct {
	Indication        string  `json:"indication" yaml:"indication"`
	Route             string  `json:"route,omitempty" yaml:"route,omitempty"`
	Dose              float64 `json:"dose,omitempty" yaml:"dose,omitempty"`
	DoseMin           float64 `json:"dose_min,omitempty" yaml:"doseMin,omitempty"`
	DoseMax           float64 `json:"dose_max,omitempty" yaml:"doseMax,omitempty"`
	Unit              string  `json:"unit" yaml:"unit"`
	Frequency         string  `json:"frequency" yaml:"frequency"`
	IsStartingDose    bool    `json:"is_starting_dose,omitempty" yaml:"isStartingDose,omitempty"`
	IsMaintenanceDose bool    `json:"is_maintenance_dose,omitempty" yaml:"isMaintenanceDose,omitempty"`
	Duration          string  `json:"duration,omitempty" yaml:"duration,omitempty"`
	Notes             string  `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// WeightBasedDosing for mg/kg calculations
type WeightBasedDosing struct {
	DosePerKg         float64 `json:"dose_per_kg" yaml:"dosePerKg"`
	Unit              string  `json:"unit" yaml:"unit"`
	Frequency         string  `json:"frequency" yaml:"frequency"`
	MinDose           float64 `json:"min_dose,omitempty" yaml:"minDose,omitempty"`
	MaxDose           float64 `json:"max_dose,omitempty" yaml:"maxDose,omitempty"`
	UseIdealWeight    bool    `json:"use_ideal_weight,omitempty" yaml:"useIdealWeight,omitempty"`
	UseAdjustedWeight bool    `json:"use_adjusted_weight,omitempty" yaml:"useAdjustedWeight,omitempty"`
	AdjustmentFactor  float64 `json:"adjustment_factor,omitempty" yaml:"adjustmentFactor,omitempty"` // For adjusted body weight
	Notes             string  `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// BSABasedDosing for mg/m2 calculations (oncology, etc.)
type BSABasedDosing struct {
	DosePerM2       float64 `json:"dose_per_m2" yaml:"dosePerM2"`
	Unit            string  `json:"unit" yaml:"unit"`
	Frequency       string  `json:"frequency,omitempty" yaml:"frequency,omitempty"`
	MaxAbsoluteDose float64 `json:"max_absolute_dose,omitempty" yaml:"maxAbsoluteDose,omitempty"`
	CappedAtBSA     float64 `json:"capped_at_bsa,omitempty" yaml:"cappedAtBSA,omitempty"` // Cap BSA calculation at this value
	Notes           string  `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// PediatricDosing contains age-specific pediatric dosing
type PediatricDosing struct {
	AgeRanges       []PediatricAgeRange `json:"age_ranges,omitempty" yaml:"ageRanges,omitempty"`
	UseWeight       bool                `json:"use_weight,omitempty" yaml:"useWeight,omitempty"`
	MinAgeMonths    int                 `json:"min_age_months,omitempty" yaml:"minAgeMonths,omitempty"`
	MaxAgeMonths    int                 `json:"max_age_months,omitempty" yaml:"maxAgeMonths,omitempty"`
	Contraindicated bool                `json:"contraindicated,omitempty" yaml:"contraindicated,omitempty"`
	Notes           string              `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// PediatricAgeRange defines dosing for specific age groups
type PediatricAgeRange struct {
	AgeGroup     string  `json:"age_group" yaml:"ageGroup"`
	MinAgeMonths int     `json:"min_age_months" yaml:"minAgeMonths"`
	MaxAgeMonths int     `json:"max_age_months" yaml:"maxAgeMonths"`
	DosePerKg    float64 `json:"dose_per_kg" yaml:"dosePerKg"`
	Unit         string  `json:"unit" yaml:"unit"`
	Frequency    string  `json:"frequency" yaml:"frequency"`
	MinDose      float64 `json:"min_dose,omitempty" yaml:"minDose,omitempty"`
	MaxDose      float64 `json:"max_dose,omitempty" yaml:"maxDose,omitempty"`
	Notes        string  `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// GeriatricDosing contains elderly-specific adjustments
type GeriatricDosing struct {
	StartLow        bool    `json:"start_low,omitempty" yaml:"startLow,omitempty"`
	DoseReduction   float64 `json:"dose_reduction,omitempty" yaml:"doseReduction,omitempty"` // Percentage reduction
	MaxDose         float64 `json:"max_dose,omitempty" yaml:"maxDose,omitempty"`
	AvoidInElderly  bool    `json:"avoid_in_elderly,omitempty" yaml:"avoidInElderly,omitempty"`
	BeersListStatus string  `json:"beers_list_status,omitempty" yaml:"beersListStatus,omitempty"` // AVOID, USE_WITH_CAUTION, CONDITIONAL
	BeersRationale  string  `json:"beers_rationale,omitempty" yaml:"beersRationale,omitempty"`
	Notes           string  `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// RenalDosing contains renal adjustment tiers
type RenalDosing struct {
	AdjustmentBasis string                `json:"adjustment_basis" yaml:"adjustmentBasis"` // eGFR, CrCl
	Adjustments     []RenalAdjustmentTier `json:"adjustments" yaml:"adjustments"`
	Notes           string                `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// RenalAdjustmentTier defines a single renal adjustment tier
type RenalAdjustmentTier struct {
	MinGFR         float64 `json:"min_gfr" yaml:"minGFR"`
	MaxGFR         float64 `json:"max_gfr" yaml:"maxGFR"`
	DosePercent    float64 `json:"dose_percent,omitempty" yaml:"dosePercent,omitempty"` // 100 = no change, 50 = half dose
	FixedDose      float64 `json:"fixed_dose,omitempty" yaml:"fixedDose,omitempty"`
	Frequency      string  `json:"frequency,omitempty" yaml:"frequency,omitempty"`
	Avoid          bool    `json:"avoid,omitempty" yaml:"avoid,omitempty"`
	Dialyzable     bool    `json:"dialyzable,omitempty" yaml:"dialyzable,omitempty"`
	SupplementDose float64 `json:"supplement_dose,omitempty" yaml:"supplementDose,omitempty"` // Post-dialysis supplement
	Notes          string  `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// HepaticDosing contains hepatic adjustment by Child-Pugh class
type HepaticDosing struct {
	ChildPughA *HepaticAdjustmentTier `json:"child_pugh_a,omitempty" yaml:"childPughA,omitempty"`
	ChildPughB *HepaticAdjustmentTier `json:"child_pugh_b,omitempty" yaml:"childPughB,omitempty"`
	ChildPughC *HepaticAdjustmentTier `json:"child_pugh_c,omitempty" yaml:"childPughC,omitempty"`
	Notes      string                 `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// HepaticAdjustmentTier defines hepatic adjustment for a Child-Pugh class
type HepaticAdjustmentTier struct {
	DosePercent float64 `json:"dose_percent,omitempty" yaml:"dosePercent,omitempty"`
	FixedDose   float64 `json:"fixed_dose,omitempty" yaml:"fixedDose,omitempty"`
	MaxDose     float64 `json:"max_dose,omitempty" yaml:"maxDose,omitempty"`
	Avoid       bool    `json:"avoid,omitempty" yaml:"avoid,omitempty"`
	Notes       string  `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// TitrationSchedule for drugs requiring gradual dose adjustment
type TitrationSchedule struct {
	Steps   []TitrationStep `json:"steps" yaml:"steps"`
	Target  string          `json:"target,omitempty" yaml:"target,omitempty"`
	MaxDose float64         `json:"max_dose,omitempty" yaml:"maxDose,omitempty"`
	Notes   string          `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// TitrationStep defines a single titration step
type TitrationStep struct {
	StepNumber   int     `json:"step_number" yaml:"stepNumber"`
	Dose         float64 `json:"dose" yaml:"dose"`
	Unit         string  `json:"unit" yaml:"unit"`
	Frequency    string  `json:"frequency" yaml:"frequency"`
	DurationDays int     `json:"duration_days,omitempty" yaml:"durationDays,omitempty"`
	Criteria     string  `json:"criteria,omitempty" yaml:"criteria,omitempty"` // Criteria to advance to next step
	Notes        string  `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// =============================================================================
// SAFETY INFORMATION
// =============================================================================

// SafetyInfo contains drug safety information
type SafetyInfo struct {
	HighAlertDrug          bool     `json:"high_alert_drug,omitempty" yaml:"highAlertDrug,omitempty"`
	NarrowTherapeuticIndex bool     `json:"narrow_therapeutic_index,omitempty" yaml:"narrowTherapeuticIndex,omitempty"`
	BlackBoxWarning        bool     `json:"black_box_warning,omitempty" yaml:"blackBoxWarning,omitempty"`
	BlackBoxText           string   `json:"black_box_text,omitempty" yaml:"blackBoxText,omitempty"`
	Monitoring             []string `json:"monitoring,omitempty" yaml:"monitoring,omitempty"`
	Contraindications      []string `json:"contraindications,omitempty" yaml:"contraindications,omitempty"`
	MajorInteractions      []string `json:"major_interactions,omitempty" yaml:"majorInteractions,omitempty"`
	Precautions            []string `json:"precautions,omitempty" yaml:"precautions,omitempty"`
	PregnancyCategory      string   `json:"pregnancy_category,omitempty" yaml:"pregnancyCategory,omitempty"` // A, B, C, D, X or lactation risk
	ClinicalNotes          string   `json:"clinical_notes,omitempty" yaml:"clinicalNotes,omitempty"`
}

// =============================================================================
// APPROVAL WORKFLOW TYPES
// =============================================================================
//
// NOTE: These types are maintained for backward compatibility.
// For enterprise-wide governance, use KB-0 Unified Governance Platform:
//   - kb-0-governance-platform/internal/models.ItemState (replaces ApprovalStatus)
//   - kb-0-governance-platform/internal/models.RiskLevel (same enum values)
//   - kb-0-governance-platform/pkg/client for workflow operations
//
// Migration Path:
//   Phase 1: KB-0 runs alongside KB-1's internal governance (current)
//   Phase 2: KB-1 uses KB-0 client for new items
//   Phase 3: Migrate existing KB-1 items to KB-0
//   Phase 4: Remove KB-1 internal governance code
// =============================================================================

// ApprovalStatus represents the lifecycle state of a drug rule
// CRITICAL: Only ACTIVE rules should be used for clinical dosing
//
// Deprecated: Use kb-0-governance-platform/internal/models.ItemState for new implementations.
// KB-0 provides a more comprehensive state machine with dual-review support.
type ApprovalStatus string

const (
	// ApprovalStatusDraft - Newly ingested, not yet reviewed
	ApprovalStatusDraft ApprovalStatus = "DRAFT"
	// ApprovalStatusReviewed - Pharmacist has reviewed, awaiting final approval
	ApprovalStatusReviewed ApprovalStatus = "REVIEWED"
	// ApprovalStatusApproved - Approved but not yet activated
	ApprovalStatusApproved ApprovalStatus = "APPROVED"
	// ApprovalStatusActive - In production, can be used for dosing
	ApprovalStatusActive ApprovalStatus = "ACTIVE"
	// ApprovalStatusRetired - No longer in use (superseded or rejected)
	ApprovalStatusRetired ApprovalStatus = "RETIRED"
)

// RiskLevel represents clinical risk classification for a drug
//
// Deprecated: Use kb-0-governance-platform/internal/models.RiskLevel for new implementations.
// Values are compatible: CRITICAL, HIGH, STANDARD, LOW
type RiskLevel string

const (
	// RiskLevelCritical - Anticoagulants, insulin, chemotherapy, opioids
	// Requires CMO + Pharmacist sign-off
	RiskLevelCritical RiskLevel = "CRITICAL"
	// RiskLevelHigh - Narrow therapeutic index, black box warning
	// Requires Pharmacist sign-off
	RiskLevelHigh RiskLevel = "HIGH"
	// RiskLevelStandard - Most drugs, standard review process
	RiskLevelStandard RiskLevel = "STANDARD"
	// RiskLevelLow - Low risk, can auto-approve with high confidence extraction
	RiskLevelLow RiskLevel = "LOW"
)

// =============================================================================
// GOVERNANCE METADATA
// =============================================================================
//
// NOTE: This struct is KB-1 specific with drug rule ingestion fields
// (ExtractionConfidence, ExtractionWarnings, RiskFactors).
//
// For enterprise governance operations (workflow, audit, cross-KB metrics),
// use KB-0 Unified Governance Platform:
//   - kb-0-governance-platform/internal/models.GovernanceTrail
//   - kb-0-governance-platform/internal/audit.Logger
//   - kb-0-governance-platform/pkg/client.Client
//
// See README at kb-0-governance-platform/ for integration guide.
// =============================================================================

// GovernanceMetadata contains full provenance tracking for regulatory compliance
// This is KB-1's drug-specific governance with ingestion quality metrics.
type GovernanceMetadata struct {
	// Primary Source
	Authority     string `json:"authority" yaml:"authority"`         // FDA, TGA, CDSCO, WHO
	Document      string `json:"document" yaml:"document"`           // DailyMed SPL, TGA PI, etc.
	Section       string `json:"section,omitempty" yaml:"section,omitempty"`
	URL           string `json:"url,omitempty" yaml:"url,omitempty"` // Direct link to source
	Jurisdiction  string `json:"jurisdiction" yaml:"jurisdiction"`   // US, AU, IN, GLOBAL
	EvidenceLevel string `json:"evidence_level,omitempty" yaml:"evidenceLevel,omitempty"` // LABEL, GUIDELINE, CONSENSUS

	// Temporal
	EffectiveDate  string `json:"effective_date,omitempty" yaml:"effectiveDate,omitempty"`
	ExpirationDate string `json:"expiration_date,omitempty" yaml:"expirationDate,omitempty"`

	// Version Control
	Version        string     `json:"version" yaml:"version"`
	ApprovedBy     string     `json:"approved_by" yaml:"approvedBy"`
	ApprovedAt     time.Time  `json:"approved_at" yaml:"approvedAt"`
	LastReviewedAt *time.Time `json:"last_reviewed_at,omitempty" yaml:"lastReviewedAt,omitempty"`
	LastReviewedBy string     `json:"last_reviewed_by,omitempty" yaml:"lastReviewedBy,omitempty"`
	NextReviewDue  string     `json:"next_review_due,omitempty" yaml:"nextReviewDue,omitempty"`

	// Approval Workflow (CRITICAL for clinical safety)
	ApprovalStatus       ApprovalStatus `json:"approval_status,omitempty" yaml:"approvalStatus,omitempty"`
	RiskLevel            RiskLevel      `json:"risk_level,omitempty" yaml:"riskLevel,omitempty"`
	RiskFactors          []string       `json:"risk_factors,omitempty" yaml:"riskFactors,omitempty"`
	RequiresManualReview bool           `json:"requires_manual_review,omitempty" yaml:"requiresManualReview,omitempty"`
	ReviewedBy           string         `json:"reviewed_by,omitempty" yaml:"reviewedBy,omitempty"`
	ReviewedAt           *time.Time     `json:"reviewed_at,omitempty" yaml:"reviewedAt,omitempty"`
	ReviewNotes          string         `json:"review_notes,omitempty" yaml:"reviewNotes,omitempty"`

	// Extraction Quality Metrics
	ExtractionConfidence int      `json:"extraction_confidence,omitempty" yaml:"extractionConfidence,omitempty"`
	ExtractionWarnings   []string `json:"extraction_warnings,omitempty" yaml:"extractionWarnings,omitempty"`

	// Audit Trail
	ChangeLog []ChangeLogEntry `json:"change_log,omitempty" yaml:"changeLog,omitempty"`

	// Secondary Sources (when primary label lacks detail)
	SecondarySources []SecondarySource `json:"secondary_sources,omitempty" yaml:"secondarySources,omitempty"`

	// Ingestion Tracking
	SourceSetID string    `json:"source_set_id,omitempty" yaml:"sourceSetId,omitempty"` // FDA SetID, TGA PI ID
	SourceHash  string    `json:"source_hash,omitempty" yaml:"sourceHash,omitempty"`    // SHA-256 of source document
	IngestedAt  time.Time `json:"ingested_at,omitempty" yaml:"ingestedAt,omitempty"`
}

// ChangeLogEntry tracks rule changes
// Also available in KB-0 at kb-0-governance-platform/internal/models.ChangeLogEntry
type ChangeLogEntry struct {
	Date     string `json:"date" yaml:"date"`
	Change   string `json:"change" yaml:"change"`
	Reviewer string `json:"reviewer" yaml:"reviewer"`
	Reason   string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// SecondarySource for when primary label lacks detail
// Also available in KB-0 at kb-0-governance-platform/internal/models.SecondarySource
type SecondarySource struct {
	Authority string `json:"authority" yaml:"authority"`
	Document  string `json:"document" yaml:"document"`
	Section   string `json:"section,omitempty" yaml:"section,omitempty"`
	Reason    string `json:"reason,omitempty" yaml:"reason,omitempty"`
	URL       string `json:"url,omitempty" yaml:"url,omitempty"`
}

// =============================================================================
// API RESPONSE TYPES
// =============================================================================

// DoseSourceAttribution for API responses - provides provenance with every dose calculation
type DoseSourceAttribution struct {
	Authority    string `json:"authority"`
	Document     string `json:"document"`
	Section      string `json:"section,omitempty"`
	URL          string `json:"url,omitempty"`
	Jurisdiction string `json:"jurisdiction"`
	Version      string `json:"version"`
	ApprovedBy   string `json:"approved_by"`
	ApprovedAt   string `json:"approved_at"`
	SourceSetID  string `json:"source_set_id,omitempty"`
	IngestedAt   string `json:"ingested_at,omitempty"`
}

// DrugRuleSummary for search results
type DrugRuleSummary struct {
	RxNormCode   string `json:"rxnorm_code"`
	DrugName     string `json:"drug_name"`
	GenericName  string `json:"generic_name"`
	DrugClass    string `json:"drug_class"`
	Jurisdiction string `json:"jurisdiction"`
	IsHighAlert  bool   `json:"is_high_alert"`
	IsNarrowTI   bool   `json:"is_narrow_ti"`
	HasBlackBox  bool   `json:"has_black_box"`
	Authority    string `json:"authority"`
	Version      string `json:"version"`
}

// GovernedDoseResponse wraps dose calculation with provenance
type GovernedDoseResponse struct {
	// Calculated dose
	RecommendedDose float64 `json:"recommended_dose"`
	Unit            string  `json:"unit"`
	Frequency       string  `json:"frequency"`
	Route           string  `json:"route,omitempty"`

	// Adjustments applied
	Adjustments []DoseAdjustment `json:"adjustments,omitempty"`

	// Safety alerts
	Alerts []SafetyAlert `json:"alerts,omitempty"`

	// Full provenance
	Source DoseSourceAttribution `json:"source"`

	// Original rule for audit
	RuleVersion string `json:"rule_version"`
}

// DoseAdjustment records an adjustment made to the base dose
type DoseAdjustment struct {
	Type        string  `json:"type"` // RENAL, HEPATIC, GERIATRIC, WEIGHT
	Reason      string  `json:"reason"`
	Adjustment  float64 `json:"adjustment"` // Multiplier (e.g., 0.5 for 50% reduction)
	PatientData string  `json:"patient_data,omitempty"`
}

// Note: SafetyAlert is defined in models.go with fields:
// AlertType, Severity, Message, Recommendation
// This avoids duplicate definitions in the same package

// =============================================================================
// HELPER METHODS
// =============================================================================

// ToSourceAttribution converts GovernanceMetadata to API response format
func (g *GovernanceMetadata) ToSourceAttribution() DoseSourceAttribution {
	return DoseSourceAttribution{
		Authority:    g.Authority,
		Document:     g.Document,
		Section:      g.Section,
		URL:          g.URL,
		Jurisdiction: g.Jurisdiction,
		Version:      g.Version,
		ApprovedBy:   g.ApprovedBy,
		ApprovedAt:   g.ApprovedAt.Format(time.RFC3339),
		SourceSetID:  g.SourceSetID,
		IngestedAt:   g.IngestedAt.Format(time.RFC3339),
	}
}

// IsHighRisk returns true if the drug is high-alert or narrow therapeutic index
func (s *SafetyInfo) IsHighRisk() bool {
	return s.HighAlertDrug || s.NarrowTherapeuticIndex || s.BlackBoxWarning
}

// RequiresRenalAdjustment checks if renal dosing adjustments are defined
func (d *DosingRules) RequiresRenalAdjustment() bool {
	return d.Renal != nil && len(d.Renal.Adjustments) > 0
}

// RequiresHepaticAdjustment checks if hepatic dosing adjustments are defined
func (d *DosingRules) RequiresHepaticAdjustment() bool {
	return d.Hepatic != nil && (d.Hepatic.ChildPughA != nil || d.Hepatic.ChildPughB != nil || d.Hepatic.ChildPughC != nil)
}

// GetRenalAdjustmentForGFR returns the appropriate renal adjustment tier for a given GFR
func (d *DosingRules) GetRenalAdjustmentForGFR(gfr float64) *RenalAdjustmentTier {
	if d.Renal == nil {
		return nil
	}
	for i := range d.Renal.Adjustments {
		adj := &d.Renal.Adjustments[i]
		if gfr >= adj.MinGFR && gfr < adj.MaxGFR {
			return adj
		}
	}
	return nil
}
