// Package safety provides comprehensive medication safety checking
// including black box warnings, contraindications, dose limits,
// pregnancy/lactation safety, and geriatric-specific evaluations.
//
// This package implements clinically governed medication safety knowledge
// with full provenance tracking, jurisdiction awareness, and regulatory
// authority citations for defensible clinical decision support.
package safety

import "time"

// =============================================================================
// CLINICAL GOVERNANCE METADATA
// =============================================================================
// These types ensure every piece of clinical knowledge is traceable to its
// authoritative source, making KB-4 regulator-defensible and auditable.

// Jurisdiction represents regulatory jurisdictions for drug safety rules
type Jurisdiction string

const (
	JurisdictionUS     Jurisdiction = "US"     // FDA, ISMP, AGS
	JurisdictionEU     Jurisdiction = "EU"     // EMA, EC
	JurisdictionAU     Jurisdiction = "AU"     // TGA, ACSQHC, PBS
	JurisdictionIN     Jurisdiction = "IN"     // CDSCO, PvPI, IPC
	JurisdictionUK     Jurisdiction = "UK"     // MHRA, NICE
	JurisdictionGlobal Jurisdiction = "GLOBAL" // WHO, ICH
)

// SourceAuthority represents the regulatory/clinical authority for knowledge
type SourceAuthority string

const (
	// US Authorities
	AuthorityFDA       SourceAuthority = "FDA"       // US Food and Drug Administration
	AuthorityISMP      SourceAuthority = "ISMP"      // Institute for Safe Medication Practices
	AuthorityAGS       SourceAuthority = "AGS"       // American Geriatrics Society (Beers)
	AuthorityCDC       SourceAuthority = "CDC"       // Centers for Disease Control
	AuthorityNLM       SourceAuthority = "NLM"       // National Library of Medicine (LactMed)

	// EU Authorities
	AuthorityEMA       SourceAuthority = "EMA"       // European Medicines Agency
	AuthorityEC        SourceAuthority = "EC"        // European Commission

	// Australia Authorities
	AuthorityTGA       SourceAuthority = "TGA"       // Therapeutic Goods Administration
	AuthorityACSQHC    SourceAuthority = "ACSQHC"    // Australian Commission on Safety and Quality
	AuthorityPBS       SourceAuthority = "PBS"       // Pharmaceutical Benefits Scheme
	AuthorityAMH       SourceAuthority = "AMH"       // Australian Medicines Handbook

	// India Authorities
	AuthorityCDSCO     SourceAuthority = "CDSCO"     // Central Drugs Standard Control Organization
	AuthorityPvPI      SourceAuthority = "PvPI"      // Pharmacovigilance Programme of India
	AuthorityIPC       SourceAuthority = "IPC"       // Indian Pharmacopoeia Commission
	AuthorityNFI       SourceAuthority = "NFI"       // National Formulary of India

	// UK Authorities
	AuthorityMHRA      SourceAuthority = "MHRA"      // Medicines and Healthcare products Regulatory Agency
	AuthorityNICE      SourceAuthority = "NICE"      // National Institute for Health and Care Excellence
	AuthoritySPS       SourceAuthority = "SPS"       // Specialist Pharmacy Service

	// International
	AuthorityWHO       SourceAuthority = "WHO"       // World Health Organization
	AuthorityICH       SourceAuthority = "ICH"       // International Council for Harmonisation
	AuthorityEUGMS     SourceAuthority = "EUGMS"     // European Union Geriatric Medicine Society (STOPP/START)
)

// EvidenceLevel represents the strength of clinical evidence
type EvidenceLevel string

const (
	EvidenceLevelA      EvidenceLevel = "A"      // High-quality RCTs or meta-analyses
	EvidenceLevelB      EvidenceLevel = "B"      // Moderate-quality RCTs or systematic reviews
	EvidenceLevelC      EvidenceLevel = "C"      // Observational studies
	EvidenceLevelD      EvidenceLevel = "D"      // Expert opinion, case reports
	EvidenceLevelExpert EvidenceLevel = "EXPERT" // Expert consensus
)

// ApprovalStatus represents the governance approval status
type ApprovalStatus string

const (
	ApprovalStatusDraft    ApprovalStatus = "DRAFT"    // Not yet reviewed
	ApprovalStatusReviewed ApprovalStatus = "REVIEWED" // Clinically reviewed
	ApprovalStatusApproved ApprovalStatus = "APPROVED" // CMO approved
	ApprovalStatusActive   ApprovalStatus = "ACTIVE"   // Active in production
	ApprovalStatusRetired  ApprovalStatus = "RETIRED"  // No longer in use
)

// ClinicalGovernance contains mandatory provenance and audit metadata
// for all clinical knowledge entries. This ensures every safety rule
// can answer: "Who says so? Which guideline? Which year? Which country?"
type ClinicalGovernance struct {
	// Source Identification
	SourceAuthority   SourceAuthority `json:"sourceAuthority" yaml:"sourceAuthority"`     // Primary authority (FDA, TGA, etc.)
	SourceDocument    string          `json:"sourceDocument" yaml:"sourceDocument"`       // Document name (e.g., "FDA Label")
	SourceSection     string          `json:"sourceSection,omitempty" yaml:"sourceSection,omitempty"` // Section (e.g., "4.3 Contraindications")
	SourceURL         string          `json:"sourceUrl,omitempty" yaml:"sourceUrl,omitempty"`         // Direct link if available
	SourceVersion     string          `json:"sourceVersion,omitempty" yaml:"sourceVersion,omitempty"` // Source document version

	// Jurisdiction
	Jurisdiction       Jurisdiction   `json:"jurisdiction" yaml:"jurisdiction"`                     // Primary jurisdiction (US, AU, etc.)
	AdditionalJurisdictions []Jurisdiction `json:"additionalJurisdictions,omitempty" yaml:"additionalJurisdictions,omitempty"`

	// Evidence Quality
	EvidenceLevel     EvidenceLevel  `json:"evidenceLevel" yaml:"evidenceLevel"`                     // A, B, C, D, EXPERT

	// Temporal Validity
	EffectiveDate     string         `json:"effectiveDate" yaml:"effectiveDate"`                     // YYYY-MM-DD
	ReviewDate        string         `json:"reviewDate,omitempty" yaml:"reviewDate,omitempty"`       // Next review due
	ExpiryDate        string         `json:"expiryDate,omitempty" yaml:"expiryDate,omitempty"`       // When rule expires

	// Versioning & Approval
	KnowledgeVersion  string         `json:"knowledgeVersion" yaml:"knowledgeVersion"`               // KB-4 internal version (e.g., "2025.1")
	ApprovalStatus    ApprovalStatus `json:"approvalStatus" yaml:"approvalStatus"`                   // DRAFT, APPROVED, ACTIVE, etc.
	ApprovedBy        string         `json:"approvedBy,omitempty" yaml:"approvedBy,omitempty"`       // CMO or approver ID
	ApprovedAt        string         `json:"approvedAt,omitempty" yaml:"approvedAt,omitempty"`       // Approval timestamp

	// Audit Trail
	CreatedBy         string         `json:"createdBy,omitempty" yaml:"createdBy,omitempty"`
	CreatedAt         string         `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	LastModifiedBy    string         `json:"lastModifiedBy,omitempty" yaml:"lastModifiedBy,omitempty"`
	LastModifiedAt    string         `json:"lastModifiedAt,omitempty" yaml:"lastModifiedAt,omitempty"`

	// Clinical Notes
	ClinicalNotes     string         `json:"clinicalNotes,omitempty" yaml:"clinicalNotes,omitempty"` // Additional clinical context
	ChangeSummary     string         `json:"changeSummary,omitempty" yaml:"changeSummary,omitempty"` // Why this was updated
}

// JurisdictionConfig represents jurisdiction-specific configuration
type JurisdictionConfig struct {
	PreferredAuthorities []SourceAuthority `json:"preferredAuthorities" yaml:"preferredAuthorities"`
	FallbackAuthorities  []SourceAuthority `json:"fallbackAuthorities" yaml:"fallbackAuthorities"`
	PregnancySystem      string            `json:"pregnancySystem" yaml:"pregnancySystem"` // "PLLR", "TGA", "CDSCO"
	TerminologyPreference string           `json:"terminologyPreference" yaml:"terminologyPreference"` // "RxNorm", "AMT", "ATC"
}

// GetDefaultJurisdictionConfig returns the authority preference for a jurisdiction
func GetDefaultJurisdictionConfig(j Jurisdiction) JurisdictionConfig {
	switch j {
	case JurisdictionUS:
		return JurisdictionConfig{
			PreferredAuthorities:  []SourceAuthority{AuthorityFDA, AuthorityISMP, AuthorityAGS},
			FallbackAuthorities:   []SourceAuthority{AuthorityWHO},
			PregnancySystem:       "PLLR",
			TerminologyPreference: "RxNorm",
		}
	case JurisdictionAU:
		return JurisdictionConfig{
			PreferredAuthorities:  []SourceAuthority{AuthorityTGA, AuthorityACSQHC, AuthorityAMH},
			FallbackAuthorities:   []SourceAuthority{AuthorityFDA, AuthorityWHO},
			PregnancySystem:       "TGA",
			TerminologyPreference: "AMT",
		}
	case JurisdictionIN:
		return JurisdictionConfig{
			PreferredAuthorities:  []SourceAuthority{AuthorityCDSCO, AuthorityPvPI, AuthorityNFI},
			FallbackAuthorities:   []SourceAuthority{AuthorityFDA, AuthorityWHO},
			PregnancySystem:       "CDSCO",
			TerminologyPreference: "ATC",
		}
	case JurisdictionEU:
		return JurisdictionConfig{
			PreferredAuthorities:  []SourceAuthority{AuthorityEMA, AuthorityEC},
			FallbackAuthorities:   []SourceAuthority{AuthorityWHO},
			PregnancySystem:       "EMA",
			TerminologyPreference: "ATC",
		}
	case JurisdictionUK:
		return JurisdictionConfig{
			PreferredAuthorities:  []SourceAuthority{AuthorityMHRA, AuthorityNICE, AuthoritySPS},
			FallbackAuthorities:   []SourceAuthority{AuthorityEMA, AuthorityWHO},
			PregnancySystem:       "MHRA",
			TerminologyPreference: "ATC",
		}
	default:
		return JurisdictionConfig{
			PreferredAuthorities:  []SourceAuthority{AuthorityWHO, AuthorityFDA},
			FallbackAuthorities:   []SourceAuthority{},
			PregnancySystem:       "PLLR",
			TerminologyPreference: "RxNorm",
		}
	}
}

// Severity levels for safety alerts
type Severity string

const (
	SeverityCritical Severity = "CRITICAL" // Block prescribing
	SeverityHigh     Severity = "HIGH"     // Require acknowledgment
	SeverityModerate Severity = "MODERATE" // Caution advised
	SeverityLow      Severity = "LOW"      // Informational
)

// AlertType represents the category of safety check
type AlertType string

const (
	AlertTypeBlackBox         AlertType = "BLACK_BOX_WARNING"
	AlertTypeContraindication AlertType = "CONTRAINDICATION"
	AlertTypeAgeLimit         AlertType = "AGE_LIMIT"
	AlertTypeDoseLimit        AlertType = "DOSE_LIMIT"
	AlertTypePregnancy        AlertType = "PREGNANCY"
	AlertTypeLactation        AlertType = "LACTATION"
	AlertTypeHighAlert        AlertType = "HIGH_ALERT"
	AlertTypeBeers            AlertType = "BEERS_CRITERIA"
	AlertTypeAnticholinergic  AlertType = "ANTICHOLINERGIC"
	AlertTypeLabRequired      AlertType = "LAB_REQUIRED"
)

// PregnancyCategory represents FDA pregnancy categories
type PregnancyCategory string

const (
	PregnancyCategoryA PregnancyCategory = "A" // Controlled studies show no risk
	PregnancyCategoryB PregnancyCategory = "B" // No evidence of risk in humans
	PregnancyCategoryC PregnancyCategory = "C" // Risk cannot be ruled out
	PregnancyCategoryD PregnancyCategory = "D" // Positive evidence of risk
	PregnancyCategoryX PregnancyCategory = "X" // Contraindicated in pregnancy
	PregnancyCategoryN PregnancyCategory = "N" // Not classified
)

// LactationRisk represents lactation risk categories
type LactationRisk string

const (
	LactationCompatible         LactationRisk = "COMPATIBLE"
	LactationProbablyCompatible LactationRisk = "PROBABLY_COMPATIBLE"
	LactationUseWithCaution     LactationRisk = "USE_WITH_CAUTION"
	LactationContraindicated    LactationRisk = "CONTRAINDICATED"
	LactationUnknown            LactationRisk = "UNKNOWN"
)

// BeersRecommendation represents Beers Criteria recommendations
type BeersRecommendation string

const (
	BeersAvoid            BeersRecommendation = "AVOID"
	BeersAvoidInCondition BeersRecommendation = "AVOID_IN_CONDITION"
	BeersUseWithCaution   BeersRecommendation = "USE_WITH_CAUTION"
)

// HighAlertCategory represents ISMP high-alert medication categories
type HighAlertCategory string

const (
	HighAlertAnticoagulants     HighAlertCategory = "ANTICOAGULANTS"
	HighAlertInsulin            HighAlertCategory = "INSULIN"
	HighAlertOpioids            HighAlertCategory = "OPIOIDS"
	HighAlertNeuromuscular      HighAlertCategory = "NEUROMUSCULAR_BLOCKERS"
	HighAlertChemotherapy       HighAlertCategory = "CHEMOTHERAPY"
	HighAlertElectrolytes       HighAlertCategory = "ELECTROLYTES"
	HighAlertAnesthetics        HighAlertCategory = "ANESTHETICS"
	HighAlertCardioactive       HighAlertCategory = "CARDIOACTIVE"
	HighAlertAntidiabetics      HighAlertCategory = "ANTIDIABETICS"
	HighAlertImmunosuppressants HighAlertCategory = "IMMUNOSUPPRESSANTS"
)

// DrugInfo represents basic drug identification
type DrugInfo struct {
	RxNormCode string `json:"rxnormCode"`
	DrugName   string `json:"drugName"`
	NDC        string `json:"ndc,omitempty"`
	DrugClass  string `json:"drugClass,omitempty"`
}

// BlackBoxWarning represents FDA/TGA/EMA black box (boxed) warning data
// with full clinical governance metadata for regulatory defensibility
type BlackBoxWarning struct {
	// Drug Identification - supports both rxnorm and rxnormCode YAML keys
	RxNormCode     string   `json:"rxnormCode" yaml:"rxnorm"`
	DrugName       string   `json:"drugName" yaml:"drugName"`
	ATCCode        string   `json:"atcCode,omitempty" yaml:"atcCode,omitempty"`           // WHO ATC classification
	AMTCode        string   `json:"amtCode,omitempty" yaml:"amtCode,omitempty"`           // Australian Medicines Terminology
	DrugClass      string   `json:"drugClass,omitempty" yaml:"drugClass,omitempty"`

	// Warning Content
	RiskCategories []string `json:"riskCategories" yaml:"riskCategories"`
	WarningText    string   `json:"warningText" yaml:"warningText"`
	Severity       Severity `json:"severity" yaml:"severity"`

	// REMS (Risk Evaluation and Mitigation Strategy) - FDA specific
	HasREMS        bool     `json:"hasRems" yaml:"hasRems"`
	REMSProgram    string   `json:"remsProgram,omitempty" yaml:"remsProgram,omitempty"`
	REMSRequirements []string `json:"remsRequirements,omitempty" yaml:"remsRequirements,omitempty"`

	// Governance (MANDATORY for clinical defensibility)
	Governance     ClinicalGovernance `json:"governance" yaml:"governance"`
}

// Contraindication represents drug contraindication data
// Sourced from FDA Label Section 4.0, EMA SmPC Section 4.3, TGA PI
type Contraindication struct {
	// Drug Identification - supports both rxnorm and rxnormCode YAML keys
	RxNormCode              string   `json:"rxnormCode" yaml:"rxnorm"`
	DrugName                string   `json:"drugName" yaml:"drugName"`
	ATCCode                 string   `json:"atcCode,omitempty" yaml:"atcCode,omitempty"`

	// Contraindication Details
	ConditionCodes          []string `json:"conditionCodes" yaml:"conditionCodes"`              // ICD-10 codes
	ConditionDescriptions   []string `json:"conditionDescriptions" yaml:"conditionDescriptions"` // Human-readable
	SNOMEDCodes             []string `json:"snomedCodes,omitempty" yaml:"snomedCodes,omitempty"` // SNOMED CT codes
	Type                    string   `json:"type" yaml:"type"`                                   // "absolute" or "relative"
	Severity                Severity `json:"severity" yaml:"severity"`
	ClinicalRationale       string   `json:"clinicalRationale" yaml:"clinicalRationale"`
	AlternativeConsiderations string `json:"alternativeConsiderations,omitempty" yaml:"alternativeConsiderations,omitempty"`

	// Governance (MANDATORY)
	Governance              ClinicalGovernance `json:"governance" yaml:"governance"`
}

// DoseLimit represents maximum dose information
// Sourced from FDA Label Dosage Section, manufacturer guidelines
type DoseLimit struct {
	// Drug Identification - supports both rxnorm and rxnormCode YAML keys
	RxNormCode        string   `json:"rxnormCode" yaml:"rxnorm"`
	DrugName          string   `json:"drugName" yaml:"drugName"`
	ATCCode           string   `json:"atcCode,omitempty" yaml:"atcCode,omitempty"`

	// Dose Limits
	MaxSingleDose     float64  `json:"maxSingleDose" yaml:"maxSingleDose"`
	MaxSingleDoseUnit string   `json:"maxSingleDoseUnit" yaml:"maxSingleDoseUnit"`
	MaxDailyDose      float64  `json:"maxDailyDose" yaml:"maxDailyDose"`
	MaxDailyDoseUnit  string   `json:"maxDailyDoseUnit" yaml:"maxDailyDoseUnit"`
	MaxCumulativeDose float64  `json:"maxCumulativeDose,omitempty" yaml:"maxCumulativeDose,omitempty"`

	// Population-Specific Limits
	GeriatricMaxDose  float64  `json:"geriatricMaxDose,omitempty" yaml:"geriatricMaxDose,omitempty"`
	PediatricMaxDose  float64  `json:"pediatricMaxDose,omitempty" yaml:"pediatricMaxDose,omitempty"`

	// Organ Impairment Adjustments
	RenalAdjustment   string   `json:"renalAdjustment,omitempty" yaml:"renalAdjustment,omitempty"`
	HepaticAdjustment string   `json:"hepaticAdjustment,omitempty" yaml:"hepaticAdjustment,omitempty"`
	RenalDoseByEGFR   map[string]float64 `json:"renalDoseByEgfr,omitempty" yaml:"renalDoseByEgfr,omitempty"` // eGFR range -> max dose
	HepaticDoseByClass map[string]float64 `json:"hepaticDoseByClass,omitempty" yaml:"hepaticDoseByClass,omitempty"` // Child-Pugh class -> max dose

	// Governance (MANDATORY)
	Governance        ClinicalGovernance `json:"governance" yaml:"governance"`
}

// AgeLimit represents age restriction data
type AgeLimit struct {
	// Drug Identification - supports both rxnorm and rxnormCode YAML keys
	RxNormCode  string   `json:"rxnormCode" yaml:"rxnorm"`
	DrugName    string   `json:"drugName" yaml:"drugName"`
	MinAgeYears float64  `json:"minAgeYears,omitempty" yaml:"minAgeYears,omitempty"`
	MaxAgeYears float64  `json:"maxAgeYears,omitempty" yaml:"maxAgeYears,omitempty"`
	Rationale   string   `json:"rationale" yaml:"rationale"`
	Severity    Severity `json:"severity" yaml:"severity"`

	// Governance (MANDATORY)
	Governance  ClinicalGovernance `json:"governance" yaml:"governance"`
}

// TeratogenicEffect represents detailed teratogenic effect information
// for complex YAML structures like those in pregnancy safety files
type TeratogenicEffect struct {
	Category   string   `json:"category,omitempty" yaml:"category,omitempty"`
	Effects    []string `json:"effects,omitempty" yaml:"effects,omitempty"`
	RiskPeriod string   `json:"riskPeriod,omitempty" yaml:"riskPeriod,omitempty"`
	Note       string   `json:"note,omitempty" yaml:"note,omitempty"`
}

// REMSProgram represents REMS program details for pregnancy safety
type REMSProgram struct {
	Name         string   `json:"name,omitempty" yaml:"name,omitempty"`
	Requirements []string `json:"requirements,omitempty" yaml:"requirements,omitempty"`
}

// PregnancySafety represents pregnancy safety information
// Note: FDA transitioned from A/B/C/D/X to PLLR narrative format in 2015
// TGA (Australia) still uses A/B1/B2/B3/C/D/X categories
// This struct supports both systems for jurisdiction flexibility
type PregnancySafety struct {
	// Drug Identification - supports both rxnorm and rxnormCode YAML keys
	RxNormCode        string            `json:"rxnormCode" yaml:"rxnorm"`
	DrugName          string            `json:"drugName" yaml:"drugName"`
	GenericName       string            `json:"genericName,omitempty" yaml:"genericName,omitempty"`
	BrandNames        []string          `json:"brandNames,omitempty" yaml:"brandNames,omitempty"`
	ATCCode           string            `json:"atcCode,omitempty" yaml:"atcCode,omitempty"`

	// Legacy Category (TGA, historical FDA)
	Category          PregnancyCategory `json:"category" yaml:"category"`
	RiskCategory      string            `json:"riskCategory,omitempty" yaml:"riskCategory,omitempty"` // X, D, C etc as string

	// FDA PLLR Format (current US standard)
	PLLRRiskSummary   string            `json:"pllrRiskSummary,omitempty" yaml:"pllrRiskSummary,omitempty"`
	PLLRClinicalConsiderations string   `json:"pllrClinicalConsiderations,omitempty" yaml:"pllrClinicalConsiderations,omitempty"`
	PLLRDataSummary   string            `json:"pllrDataSummary,omitempty" yaml:"pllrDataSummary,omitempty"`
	RiskSummary       string            `json:"riskSummary,omitempty" yaml:"riskSummary,omitempty"` // Alternative field

	// Teratogenicity - supports both simple []string and complex []TeratogenicEffect
	Teratogenic         bool                `json:"teratogenic" yaml:"teratogenic"`
	TeratogenicEffects  interface{}         `json:"teratogenicEffects,omitempty" yaml:"teratogenicEffects,omitempty"` // Can be []string or []TeratogenicEffect
	TrimesterRisks      interface{}         `json:"trimesterRisks,omitempty" yaml:"trimesterRisks,omitempty"` // Can be map[string]string or complex

	// REMS Program
	REMSProgramData     *REMSProgram        `json:"remsProgram,omitempty" yaml:"remsProgram,omitempty"`

	// Dose Limits
	DoseLimit           interface{}         `json:"doseLimit,omitempty" yaml:"doseLimit,omitempty"` // Complex nested structure

	// Recommendations
	Recommendation       string             `json:"recommendation" yaml:"recommendation"`
	AlternativeDrugs     []string           `json:"alternativeDrugs,omitempty" yaml:"alternativeDrugs,omitempty"`
	Alternatives         []string           `json:"alternatives,omitempty" yaml:"alternatives,omitempty"` // Alternative field name
	MonitoringRequired   []string           `json:"monitoringRequired,omitempty" yaml:"monitoringRequired,omitempty"`
	MonitoringInPregnancy []string          `json:"monitoringInPregnancy,omitempty" yaml:"monitoringInPregnancy,omitempty"` // Alternative field

	// Exceptions and additional info - supports complex YAML structures
	Exceptions           interface{}        `json:"exceptions,omitempty" yaml:"exceptions,omitempty"` // Can be []string or []map

	// Governance (MANDATORY)
	Governance           ClinicalGovernance `json:"governance" yaml:"governance"`
}

// LactationSafety represents lactation safety information
// Primary source: NIH LactMed database (National Library of Medicine)
// Also: FDA PLLR Section 8.2, WHO breastfeeding guidance
type LactationSafety struct {
	// Drug Identification - supports both rxnorm and rxnormCode YAML keys
	RxNormCode       string        `json:"rxnormCode" yaml:"rxnorm"`
	DrugName         string        `json:"drugName" yaml:"drugName"`
	GenericName      string        `json:"genericName,omitempty" yaml:"genericName,omitempty"`
	ATCCode          string        `json:"atcCode,omitempty" yaml:"atcCode,omitempty"`

	// Risk Assessment - supports both risk and lactationRisk YAML keys
	Risk             LactationRisk `json:"risk" yaml:"lactationRisk"`
	RiskSummary      string        `json:"riskSummary,omitempty" yaml:"riskSummary,omitempty"`
	ExcretedInMilk   bool          `json:"excretedInMilk" yaml:"excretedInMilk"`
	MilkConcentration string       `json:"milkConcentration,omitempty" yaml:"milkConcentration,omitempty"`
	MilkPlasmaRatio  string        `json:"milkPlasmaRatio,omitempty" yaml:"milkPlasmaRatio,omitempty"`
	MilkToPlasmaRatio string       `json:"milkToPlasmaRatio,omitempty" yaml:"milkToPlasmaRatio,omitempty"` // Alternative field
	InfantDosePercent float64      `json:"infantDosePercent,omitempty" yaml:"infantDosePercent,omitempty"` // Relative Infant Dose (RID)
	RelativeInfantDose string      `json:"relativeInfantDose,omitempty" yaml:"relativeInfantDose,omitempty"` // String version
	HalfLifeHours    float64       `json:"halfLifeHours" yaml:"halfLifeHours"`

	// Infant Considerations - supports both infantEffects and infantRisk YAML keys
	InfantEffects    []string      `json:"infantEffects,omitempty" yaml:"infantRisk"` // YAML uses infantRisk
	InfantMonitoring []string      `json:"infantMonitoring,omitempty" yaml:"infantMonitoring,omitempty"`
	MonitoringIfUsed []string      `json:"monitoringIfUsed,omitempty" yaml:"monitoringIfUsed,omitempty"` // Alternative field
	MonitoringIfExposed []string   `json:"monitoringIfExposed,omitempty" yaml:"monitoringIfExposed,omitempty"` // Alternative field

	// Risk Factors
	RiskFactors      []string      `json:"riskFactors,omitempty" yaml:"riskFactors,omitempty"`
	MaternalEffect   string        `json:"maternalEffect,omitempty" yaml:"maternalEffect,omitempty"`

	// Pharmacogenetic considerations
	PharmacogeneticRisk string     `json:"pharmacogeneticRisk,omitempty" yaml:"pharmacogeneticRisk,omitempty"`
	InfantFatalities string        `json:"infantFatalities,omitempty" yaml:"infantFatalities,omitempty"`
	FDAWarning       string        `json:"fdaWarning,omitempty" yaml:"fdaWarning,omitempty"`

	// Recommendations
	Recommendation   string        `json:"recommendation" yaml:"recommendation"`
	AlternativeDrugs []string      `json:"alternativeDrugs,omitempty" yaml:"alternativeDrugs,omitempty"`
	Alternatives     []string      `json:"alternatives,omitempty" yaml:"alternatives,omitempty"` // Alternative field name
	AlternativesPreferred []string `json:"alternativesPreferred,omitempty" yaml:"alternativesPreferred,omitempty"` // Preferred alternatives
	TimingAdvice     string        `json:"timingAdvice,omitempty" yaml:"timingAdvice,omitempty"`
	WaitAfterDose    string        `json:"waitAfterDose,omitempty" yaml:"waitAfterDose,omitempty"`
	WaitPeriodAfterLastDose string `json:"waitPeriodAfterLastDose,omitempty" yaml:"waitPeriodAfterLastDose,omitempty"`

	// Additional fields from YAML
	Monitoring       []string      `json:"monitoring,omitempty" yaml:"monitoring,omitempty"`
	Note             string        `json:"note,omitempty" yaml:"note,omitempty"`
	InfantSerumLevels string       `json:"infantSerumLevels,omitempty" yaml:"infantSerumLevels,omitempty"`

	// Governance (MANDATORY)
	Governance       ClinicalGovernance `json:"governance" yaml:"governance"`
}

// HighAlertMedication represents ISMP high-alert medication data
// Primary source: ISMP High-Alert Medications Lists (Acute Care & Community)
// Updated annually by Institute for Safe Medication Practices
type HighAlertMedication struct {
	// Drug Identification - supports both rxnorm and rxnormCode YAML keys
	RxNormCode    string            `json:"rxnormCode" yaml:"rxnorm"`
	DrugName      string            `json:"drugName" yaml:"drugName"`
	ATCCode       string            `json:"atcCode,omitempty" yaml:"atcCode,omitempty"`
	TallManName   string            `json:"tallManName,omitempty" yaml:"tallManName,omitempty"` // e.g., "DOPamine" vs "DOBUTamine"

	// ISMP Classification
	Category      HighAlertCategory `json:"category" yaml:"category"`
	ISMPListType  string            `json:"ismpListType,omitempty" yaml:"ismpListType,omitempty"` // "acute_care" or "community"

	// Safety Requirements
	Requirements  []string          `json:"requirements" yaml:"requirements"`
	Safeguards    []string          `json:"safeguards" yaml:"safeguards"`
	DoubleCheck   bool              `json:"doubleCheck" yaml:"doubleCheck"`
	SmartPump     bool              `json:"smartPump" yaml:"smartPump"`

	// Additional ISMP Requirements
	IndependentDoubleCheck bool     `json:"independentDoubleCheck,omitempty" yaml:"independentDoubleCheck,omitempty"`
	MaxConcentration float64        `json:"maxConcentration,omitempty" yaml:"maxConcentration,omitempty"` // For IV meds
	WeightBasedDosing bool          `json:"weightBasedDosing,omitempty" yaml:"weightBasedDosing,omitempty"`

	// Governance (MANDATORY)
	Governance    ClinicalGovernance `json:"governance" yaml:"governance"`
}

// ConditionToAvoid represents a condition to avoid for Beers Criteria entries
type ConditionToAvoid struct {
	Code    string `json:"code,omitempty" yaml:"code,omitempty"`
	Display string `json:"display,omitempty" yaml:"display,omitempty"`
}

// DoseValue represents a dose with value, unit, and optional note
type DoseValue struct {
	Value float64 `json:"value,omitempty" yaml:"value,omitempty"`
	Unit  string  `json:"unit,omitempty" yaml:"unit,omitempty"`
	Note  string  `json:"note,omitempty" yaml:"note,omitempty"`
}

// TargetRange represents a target range for serum levels
type TargetRange struct {
	Min  float64 `json:"min,omitempty" yaml:"min,omitempty"`
	Max  float64 `json:"max,omitempty" yaml:"max,omitempty"`
	Unit string  `json:"unit,omitempty" yaml:"unit,omitempty"`
}

// BeersEntry represents Beers Criteria information for geriatric patients
// Primary source: American Geriatrics Society (AGS) Beers Criteria 2023
// The Beers Criteria® is a registered trademark of the AGS
type BeersEntry struct {
	// Drug Identification - supports both rxnorm and rxnormCode YAML keys
	RxNormCode        string              `json:"rxnormCode" yaml:"rxnorm"`
	DrugName          string              `json:"drugName" yaml:"drugName"`
	ATCCode           string              `json:"atcCode,omitempty" yaml:"atcCode,omitempty"`
	DrugClass         string              `json:"drugClass,omitempty" yaml:"drugClass,omitempty"`

	// AGS Beers Criteria Classification
	Recommendation    BeersRecommendation `json:"recommendation" yaml:"recommendation"`
	BeersTable        string              `json:"beersTable,omitempty" yaml:"beersTable,omitempty"` // Table 1, 2, 3, etc.
	Rationale         string              `json:"rationale" yaml:"rationale"`
	QualityOfEvidence string              `json:"qualityOfEvidence" yaml:"qualityOfEvidence"` // High, Moderate, Low
	StrengthOfRecommendation string       `json:"strengthOfRecommendation" yaml:"strengthOfRecommendation"` // Strong, Weak

	// Disease/Syndrome Interactions (Beers Table 3) - supports complex YAML structure
	Conditions          []string            `json:"conditions,omitempty" yaml:"conditions,omitempty"` // Simple string conditions
	ConditionsToAvoid   interface{}         `json:"conditionsToAvoid,omitempty" yaml:"conditionsToAvoid,omitempty"` // Complex: []ConditionToAvoid or []string
	ConditionCodes      []string            `json:"conditionCodes,omitempty" yaml:"conditionCodes,omitempty"` // ICD-10 codes

	// Dose Limits - supports complex YAML structure
	MaxSafeDose         interface{}         `json:"maxSafeDose,omitempty" yaml:"maxSafeDose,omitempty"` // DoseValue or simple value
	TargetSerumLevel    interface{}         `json:"targetSerumLevel,omitempty" yaml:"targetSerumLevel,omitempty"` // TargetRange

	// Anticholinergic Burden (often included in Beers)
	ACBScore            int                 `json:"acbScore,omitempty" yaml:"acbScore,omitempty"`

	// Alternatives - supports both field names
	AlternativeDrugs    []string            `json:"alternativeDrugs,omitempty" yaml:"alternativeDrugs,omitempty"`
	Alternatives        []string            `json:"alternatives,omitempty" yaml:"alternatives,omitempty"` // Alternative field name
	NonPharmacologic    []string            `json:"nonPharmacologic,omitempty" yaml:"nonPharmacologic,omitempty"`
	NonPharmacologicOptions []string        `json:"nonPharmacologicOptions,omitempty" yaml:"nonPharmacologicOptions,omitempty"` // Alternative field

	// Age Threshold
	AgeThreshold        int                 `json:"ageThreshold,omitempty" yaml:"ageThreshold,omitempty"` // Default 65, but some entries differ

	// Additional YAML fields
	Exceptions          []string            `json:"exceptions,omitempty" yaml:"exceptions,omitempty"`
	RiskFactors         []string            `json:"riskFactors,omitempty" yaml:"riskFactors,omitempty"`
	LongTermIndications []string            `json:"longTermIndications,omitempty" yaml:"longTermIndications,omitempty"`
	ToxicMetabolite     string              `json:"toxicMetabolite,omitempty" yaml:"toxicMetabolite,omitempty"`
	Toxicities          []string            `json:"toxicities,omitempty" yaml:"toxicities,omitempty"`
	Risks               []string            `json:"risks,omitempty" yaml:"risks,omitempty"`
	Contraindications   []string            `json:"contraindications,omitempty" yaml:"contraindications,omitempty"`
	HalfLifeHours       float64             `json:"halfLifeHours,omitempty" yaml:"halfLifeHours,omitempty"`
	HasActiveMetabolites bool               `json:"hasActiveMetabolites,omitempty" yaml:"hasActiveMetabolites,omitempty"`
	MaxDuration         string              `json:"maxDuration,omitempty" yaml:"maxDuration,omitempty"`

	// Governance (MANDATORY)
	Governance          ClinicalGovernance  `json:"governance" yaml:"governance"`
}

// AnticholinergicBurden represents anticholinergic burden score data
// Primary source: Anticholinergic Cognitive Burden (ACB) Scale
// Also: Anticholinergic Risk Scale (ARS), Drug Burden Index
type AnticholinergicBurden struct {
	// Drug Identification - supports both rxnorm and rxnormCode YAML keys
	RxNormCode  string `json:"rxnormCode" yaml:"rxnorm"`
	DrugName    string `json:"drugName" yaml:"drugName"`
	ATCCode     string `json:"atcCode,omitempty" yaml:"atcCode,omitempty"`

	// ACB Scoring
	ACBScore    int    `json:"acbScore" yaml:"acbScore"` // 1-3 scale
	RiskLevel   string `json:"riskLevel" yaml:"riskLevel"` // Low (1), Moderate (2), High (3)
	ScaleUsed   string `json:"scaleUsed,omitempty" yaml:"scaleUsed,omitempty"` // "ACB", "ARS", "DBI"

	// Clinical Effects
	Effects     []string `json:"effects,omitempty" yaml:"effects,omitempty"`
	CognitiveRisk string `json:"cognitiveRisk,omitempty" yaml:"cognitiveRisk,omitempty"` // Delirium, dementia risk
	PeripheralEffects []string `json:"peripheralEffects,omitempty" yaml:"peripheralEffects,omitempty"` // Dry mouth, constipation, etc.

	// Population Considerations
	GeriatricRisk string `json:"geriatricRisk,omitempty" yaml:"geriatricRisk,omitempty"`
	DementiaRisk  string `json:"dementiaRisk,omitempty" yaml:"dementiaRisk,omitempty"`

	// Governance (MANDATORY)
	Governance  ClinicalGovernance `json:"governance" yaml:"governance"`
}

// LabMonitoringEntry represents a single lab monitoring requirement with complex structure
type LabMonitoringEntry struct {
	LabName        string                 `json:"labName,omitempty" yaml:"labName,omitempty"`
	LOINCCode      string                 `json:"loincCode,omitempty" yaml:"loincCode,omitempty"`
	Purpose        string                 `json:"purpose,omitempty" yaml:"purpose,omitempty"`
	TargetRange    interface{}            `json:"targetRange,omitempty" yaml:"targetRange,omitempty"` // Complex nested structure
	Frequency      interface{}            `json:"frequency,omitempty" yaml:"frequency,omitempty"` // Can be string or map
	Timing         string                 `json:"timing,omitempty" yaml:"timing,omitempty"`
	CriticalValues interface{}            `json:"criticalValues,omitempty" yaml:"criticalValues,omitempty"` // Complex nested
	ActionRequired interface{}            `json:"actionRequired,omitempty" yaml:"actionRequired,omitempty"` // Can be string or map
	Note           string                 `json:"note,omitempty" yaml:"note,omitempty"`
}

// LabRequirement represents required laboratory monitoring
// Primary sources: FDA Label Monitoring Section, NICE Guidelines, SPS Monitoring Guidelines
type LabRequirement struct {
	// Drug Identification - supports both rxnorm and rxnormCode YAML keys
	RxNormCode        string   `json:"rxnormCode" yaml:"rxnorm"`
	DrugName          string   `json:"drugName" yaml:"drugName"`
	GenericName       string   `json:"genericName,omitempty" yaml:"genericName,omitempty"`
	ATCCode           string   `json:"atcCode,omitempty" yaml:"atcCode,omitempty"`
	DrugClass         string   `json:"drugClass,omitempty" yaml:"drugClass,omitempty"`

	// Monitoring flags
	MonitoringRequired bool    `json:"monitoringRequired,omitempty" yaml:"monitoringRequired,omitempty"`
	CriticalMonitoring bool    `json:"criticalMonitoring,omitempty" yaml:"criticalMonitoring,omitempty"`

	// REMS Program (for drugs like Clozapine)
	REMSProgram       string   `json:"remsProgram,omitempty" yaml:"remsProgram,omitempty"`

	// Laboratory Tests - supports complex YAML structure
	Labs              []LabMonitoringEntry `json:"labs,omitempty" yaml:"labs,omitempty"` // Complex nested lab entries
	RequiredLabs      []string             `json:"requiredLabs,omitempty" yaml:"requiredLabs,omitempty"` // Simple list
	LabCodes          []string             `json:"labCodes,omitempty" yaml:"labCodes,omitempty"` // LOINC codes
	LabDescriptions   []string             `json:"labDescriptions,omitempty" yaml:"labDescriptions,omitempty"`

	// Monitoring Schedule (legacy simple fields)
	Frequency         string   `json:"frequency,omitempty" yaml:"frequency,omitempty"`
	BaselineRequired  bool     `json:"baselineRequired,omitempty" yaml:"baselineRequired,omitempty"`
	InitialMonitoring string   `json:"initialMonitoring,omitempty" yaml:"initialMonitoring,omitempty"`
	OngoingMonitoring string   `json:"ongoingMonitoring,omitempty" yaml:"ongoingMonitoring,omitempty"`

	// Thresholds and Actions (legacy simple fields)
	CriticalValues    map[string]string `json:"criticalValues,omitempty" yaml:"criticalValues,omitempty"`
	ActionRequired    string            `json:"actionRequired,omitempty" yaml:"actionRequired,omitempty"`

	// Clinical Context
	Rationale            string   `json:"rationale,omitempty" yaml:"rationale,omitempty"`
	ConsequenceOfMissing string   `json:"consequenceOfMissing,omitempty" yaml:"consequenceOfMissing,omitempty"`

	// Additional YAML fields
	RiskFactors           []string `json:"riskFactors,omitempty" yaml:"riskFactors,omitempty"`
	DrugInteractions      []string `json:"drugInteractions,omitempty" yaml:"drugInteractions,omitempty"`
	FolateSupplementation string   `json:"folateSupplementation,omitempty" yaml:"folateSupplementation,omitempty"`
	PharmacogeneticTesting string  `json:"pharmacogeneticTesting,omitempty" yaml:"pharmacogeneticTesting,omitempty"`
	Audiometry            string   `json:"audiometry,omitempty" yaml:"audiometry,omitempty"`

	// Governance (MANDATORY)
	Governance        ClinicalGovernance `json:"governance" yaml:"governance"`
}

// PatientContext represents patient information for safety evaluation
type PatientContext struct {
	PatientID    string    `json:"patientId,omitempty"`
	AgeYears     float64   `json:"ageYears"`
	AgeMonths    float64   `json:"ageMonths,omitempty"`
	Gender       string    `json:"gender"` // M, F, U
	WeightKg     float64   `json:"weightKg,omitempty"`
	HeightCm     float64   `json:"heightCm,omitempty"`
	IsPregnant   bool      `json:"isPregnant"`
	IsLactating  bool      `json:"isLactating"`
	Trimester    int       `json:"trimester,omitempty"` // 1, 2, 3
	Diagnoses    []Diagnosis `json:"diagnoses,omitempty"`
	Allergies    []Allergy   `json:"allergies,omitempty"`
	RenalFunction *RenalFunction `json:"renalFunction,omitempty"`
	HepaticFunction *HepaticFunction `json:"hepaticFunction,omitempty"`
	CurrentMedications []DrugInfo `json:"currentMedications,omitempty"`
}

// Diagnosis represents a patient diagnosis
type Diagnosis struct {
	Code    string `json:"code"`    // ICD-10
	Display string `json:"display"`
	System  string `json:"system,omitempty"`
}

// Allergy represents a patient allergy
type Allergy struct {
	Substance    string `json:"substance"`
	Code         string `json:"code,omitempty"`
	ReactionType string `json:"reactionType,omitempty"`
	Severity     string `json:"severity,omitempty"`
}

// RenalFunction represents kidney function metrics
type RenalFunction struct {
	Creatinine     float64 `json:"creatinine,omitempty"`
	BUN            float64 `json:"bun,omitempty"`
	EGFR           float64 `json:"egfr,omitempty"`
	CrCl           float64 `json:"crcl,omitempty"` // Creatinine clearance
	Stage          string  `json:"stage,omitempty"` // CKD stage
}

// HepaticFunction represents liver function metrics
type HepaticFunction struct {
	AST          float64 `json:"ast,omitempty"`
	ALT          float64 `json:"alt,omitempty"`
	Bilirubin    float64 `json:"bilirubin,omitempty"`
	Albumin      float64 `json:"albumin,omitempty"`
	INR          float64 `json:"inr,omitempty"`
	ChildPughScore int   `json:"childPughScore,omitempty"`
	ChildPughClass string `json:"childPughClass,omitempty"` // A, B, C
}

// SafetyCheckRequest represents a request for safety evaluation
type SafetyCheckRequest struct {
	Drug         DrugInfo       `json:"drug"`
	ProposedDose float64        `json:"proposedDose,omitempty"`
	DoseUnit     string         `json:"doseUnit,omitempty"`
	Frequency    string         `json:"frequency,omitempty"`
	Route        string         `json:"route,omitempty"`
	Patient      PatientContext `json:"patient"`
	CheckTypes   []AlertType    `json:"checkTypes,omitempty"` // Empty = all checks
}

// SafetyAlert represents an individual safety finding
type SafetyAlert struct {
	ID                     string    `json:"id,omitempty"`
	Type                   AlertType `json:"type"`
	Severity               Severity  `json:"severity"`
	Title                  string    `json:"title"`
	Message                string    `json:"message"`
	RequiresAcknowledgment bool      `json:"requiresAcknowledgment"`
	CanOverride            bool      `json:"canOverride"`
	ClinicalRationale      string    `json:"clinicalRationale,omitempty"`
	Recommendations        []string  `json:"recommendations,omitempty"`
	References             []string  `json:"references,omitempty"`
	DrugInfo               *DrugInfo `json:"drugInfo,omitempty"`
	CreatedAt              time.Time `json:"createdAt,omitempty"`
}

// SafetyCheckResponse represents the result of a safety evaluation
type SafetyCheckResponse struct {
	Safe              bool           `json:"safe"`
	RequiresAction    bool           `json:"requiresAction"`
	BlockPrescribing  bool           `json:"blockPrescribing"`
	CriticalAlerts    int            `json:"criticalAlerts"`
	HighAlerts        int            `json:"highAlerts"`
	ModerateAlerts    int            `json:"moderateAlerts"`
	LowAlerts         int            `json:"lowAlerts"`
	TotalAlerts       int            `json:"totalAlerts"`
	Alerts            []SafetyAlert  `json:"alerts"`
	IsHighAlertDrug   bool           `json:"isHighAlertDrug"`
	AnticholinergicBurdenTotal int   `json:"anticholinergicBurdenTotal,omitempty"`
	CheckedAt         time.Time      `json:"checkedAt"`
	RequestID         string         `json:"requestId,omitempty"`
}

// DoseLimitValidation represents dose validation request/response
type DoseLimitValidation struct {
	Drug          DrugInfo `json:"drug"`
	ProposedDose  float64  `json:"proposedDose"`
	DoseUnit      string   `json:"doseUnit"`
	Patient       PatientContext `json:"patient,omitempty"`
	IsValid       bool     `json:"isValid"`
	ExceedsSingle bool     `json:"exceedsSingle"`
	ExceedsDaily  bool     `json:"exceedsDaily"`
	MaxAllowed    float64  `json:"maxAllowed,omitempty"`
	Message       string   `json:"message,omitempty"`
}

// AnticholinergicBurdenCalculation represents ACB calculation result
type AnticholinergicBurdenCalculation struct {
	TotalScore     int                     `json:"totalScore"`
	RiskLevel      string                  `json:"riskLevel"` // Low, Moderate, High, Very High
	Medications    []AnticholinergicBurden `json:"medications"`
	Recommendation string                  `json:"recommendation"`
	CognitiveRisk  string                  `json:"cognitiveRisk"`
}

// =============================================================================
// STOPP/START CRITERIA (European Geriatric Prescribing)
// =============================================================================
// Source: O'Mahony D, et al. STOPP/START criteria for potentially inappropriate
// prescribing in older people: version 3. Age and Ageing, 2023
// DOI: 10.1093/ageing/afad042
// Applicability: Adults ≥65 years (unless end-of-life or symptom control priority)

// StoppEntry represents a STOPP (Screening Tool of Older Persons' Prescriptions) criterion
// STOPP criteria identify potentially inappropriate prescribing in older adults
type StoppEntry struct {
	// Criterion Identification
	ID            string   `json:"id" yaml:"id"`                     // e.g., "A1", "B2", "K3"
	Section       string   `json:"section" yaml:"section"`           // e.g., "A - Indication of Medication"
	SectionName   string   `json:"sectionName" yaml:"sectionName"`   // e.g., "General Prescribing Principles"

	// Drug/Class Identification
	DrugClass     string   `json:"drugClass,omitempty" yaml:"drugClass,omitempty"`   // Drug class if applicable
	RxNormCodes   []string `json:"rxnormCodes,omitempty" yaml:"rxnormCodes,omitempty"` // Specific RxNorm codes
	ATCCodes      []string `json:"atcCodes,omitempty" yaml:"atcCodes,omitempty"`     // ATC classification codes

	// Condition Context (when STOPP applies to specific conditions)
	Condition       string   `json:"condition,omitempty" yaml:"condition,omitempty"`
	ConditionICD10  []string `json:"conditionICD10,omitempty" yaml:"conditionICD10,omitempty"`
	SNOMEDCodes     []string `json:"snomedCodes,omitempty" yaml:"snomedCodes,omitempty"`

	// Clinical Criteria
	Criteria        string   `json:"criteria" yaml:"criteria"`             // The actual STOPP criterion text
	Rationale       string   `json:"rationale" yaml:"rationale"`           // Clinical rationale for stopping
	EvidenceLevel   string   `json:"evidenceLevel" yaml:"evidenceLevel"`   // Evidence level (I, II, III)
	Exceptions      string   `json:"exceptions,omitempty" yaml:"exceptions,omitempty"` // When criterion doesn't apply

	// Alternative Recommendations
	Alternatives    []string `json:"alternatives,omitempty" yaml:"alternatives,omitempty"`

	// Governance (MANDATORY)
	Governance      ClinicalGovernance `json:"governance" yaml:"governance"`
}

// StartEntry represents a START (Screening Tool to Alert to Right Treatment) criterion
// START criteria identify potential prescribing omissions in older adults
type StartEntry struct {
	// Criterion Identification
	ID            string   `json:"id" yaml:"id"`                     // e.g., "A1", "B2"
	Section       string   `json:"section" yaml:"section"`           // e.g., "A - Cardiovascular System"
	SectionName   string   `json:"sectionName" yaml:"sectionName"`   // e.g., "Cardiovascular Prescribing Omissions"

	// Condition Context (when START applies)
	Condition       string   `json:"condition" yaml:"condition"`           // Condition requiring treatment
	ConditionICD10  []string `json:"conditionICD10" yaml:"conditionICD10"` // ICD-10 codes for condition
	SNOMEDCodes     []string `json:"snomedCodes,omitempty" yaml:"snomedCodes,omitempty"`

	// Recommended Treatment
	RecommendedDrugs []string `json:"recommendedDrugs" yaml:"recommendedDrugs"` // Drugs that should be prescribed
	RxNormCodes     []string  `json:"rxnormCodes,omitempty" yaml:"rxnormCodes,omitempty"` // RxNorm codes for recommended drugs
	ATCCodes        []string  `json:"atcCodes,omitempty" yaml:"atcCodes,omitempty"`       // ATC codes for recommended drugs

	// Clinical Criteria
	Criteria        string   `json:"criteria" yaml:"criteria"`             // The actual START criterion text
	Rationale       string   `json:"rationale" yaml:"rationale"`           // Clinical rationale for starting
	EvidenceLevel   string   `json:"evidenceLevel" yaml:"evidenceLevel"`   // Evidence level (I, II, III)
	Exceptions      string   `json:"exceptions,omitempty" yaml:"exceptions,omitempty"` // When criterion doesn't apply

	// Governance (MANDATORY)
	Governance      ClinicalGovernance `json:"governance" yaml:"governance"`
}

// StoppViolation represents a detected STOPP criteria violation
type StoppViolation struct {
	Entry           *StoppEntry `json:"entry"`
	CurrentDrug     DrugInfo    `json:"currentDrug"`
	MatchedCondition string     `json:"matchedCondition,omitempty"`
	Message         string      `json:"message"`
	Severity        Severity    `json:"severity"`
}

// StartRecommendation represents a detected START criteria recommendation (prescribing omission)
type StartRecommendation struct {
	Entry           *StartEntry `json:"entry"`
	MatchedCondition string     `json:"matchedCondition"`
	RecommendedDrugs []string   `json:"recommendedDrugs"`
	Message         string      `json:"message"`
	Severity        Severity    `json:"severity"`
}

// =============================================================================
// INDIA-SPECIFIC TYPES (CDSCO, NLEM)
// =============================================================================
// Jurisdiction: IN
// Authorities: CDSCO (Central Drugs Standard Control Organisation),
//              Ministry of Health and Family Welfare (MoHFW)

// BannedCombinationComponent represents a single drug component in a banned FDC
type BannedCombinationComponent struct {
	Drug    string `json:"drug" yaml:"drug"`
	RxNorm  string `json:"rxnorm" yaml:"rxnorm"`
}

// BannedCombinationEntry represents a CDSCO banned fixed-dose combination
// India banned 344+ FDCs in 2016 under Section 26A of Drugs and Cosmetics Act
// These combinations were deemed irrational or lacking therapeutic justification
type BannedCombinationEntry struct {
	// Identification
	ID               string                       `json:"id" yaml:"id"`                           // e.g., "BAN-ANA-001"
	CombinationName  string                       `json:"combinationName" yaml:"combinationName"` // e.g., "Aceclofenac + Paracetamol + Tramadol"
	Components       []BannedCombinationComponent `json:"components" yaml:"components"`           // List of drug components
	Category         string                       `json:"category" yaml:"category"`               // Therapeutic category

	// Clinical Information
	BanRationale              string `json:"banRationale" yaml:"banRationale"`
	AlternativeRecommendation string `json:"alternativeRecommendation" yaml:"alternativeRecommendation"`

	// Governance (MANDATORY)
	Governance ClinicalGovernance `json:"governance" yaml:"governance"`
}

// NLEMMedication represents a medication in India's National List of Essential Medicines
// NLEM 2022 includes 384 medicines across 27 therapeutic categories
// Essential levels: P (Primary), S (Secondary), T (Tertiary) healthcare
type NLEMMedication struct {
	// Drug Identification
	RxNorm   string `json:"rxnorm" yaml:"rxnorm"`
	DrugName string `json:"drugName" yaml:"drugName"`
	Strength string `json:"strength" yaml:"strength"` // e.g., "50mg/mL injection"

	// NLEM Classification
	Category       string `json:"category" yaml:"category"`             // Therapeutic category
	EssentialLevel string `json:"essentialLevel" yaml:"essentialLevel"` // P=Primary, S=Secondary, T=Tertiary

	// Governance (MANDATORY)
	Governance ClinicalGovernance `json:"governance" yaml:"governance"`
}

// BannedCombinationViolation represents detection of a banned FDC prescription
type BannedCombinationViolation struct {
	Entry           *BannedCombinationEntry `json:"entry"`
	MatchedDrugs    []DrugInfo              `json:"matchedDrugs"`
	Message         string                  `json:"message"`
	Severity        Severity                `json:"severity"`
}
