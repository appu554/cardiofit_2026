// Package factstore provides the Canonical Fact Store models.
// These models represent the immutable clinical knowledge spine.
//
// DESIGN PRINCIPLE: "Freeze meaning. Fluidly replace intelligence."
// Facts are immutable meaning that outlive LLM models, API changes, and vendor pivots.
package factstore

import (
	"encoding/json"
	"time"
)

// =============================================================================
// FACT TYPES (The Six Canonical Categories)
// =============================================================================

// FactType represents the category of clinical fact
type FactType string

const (
	// FactTypeOrganImpairment covers renal, hepatic, and other organ-based dosing adjustments
	FactTypeOrganImpairment FactType = "ORGAN_IMPAIRMENT"

	// FactTypeSafetySignal covers black box warnings, contraindications, alerts
	FactTypeSafetySignal FactType = "SAFETY_SIGNAL"

	// FactTypeReproductiveSafety covers pregnancy categories, lactation, teratogenicity
	FactTypeReproductiveSafety FactType = "REPRODUCTIVE_SAFETY"

	// FactTypeInteraction covers drug-drug, drug-food, drug-lab interactions
	FactTypeInteraction FactType = "INTERACTION"

	// FactTypeFormulary covers coverage, tiers, prior auth requirements
	FactTypeFormulary FactType = "FORMULARY"

	// FactTypeLabReference covers reference ranges, critical values, monitoring requirements
	FactTypeLabReference FactType = "LAB_REFERENCE"
)

// =============================================================================
// FACT STABILITY CONTRACT (Gap 1 - Clinical Knowledge OS)
// =============================================================================
// Every fact must declare its volatility to enable:
// - Automatic staleness detection
// - Refresh scheduling
// - Compliance with temporal audit requirements

// VolatilityClass classifies how often a fact is expected to change
type VolatilityClass string

const (
	// VolatilityStable - Facts that rarely change (≤1 change/year)
	// Examples: contraindications, boxed warnings, mechanism of action
	VolatilityStable VolatilityClass = "STABLE"

	// VolatilityEvolving - Facts that update periodically (monthly)
	// Examples: formulary coverage, clinical guidelines, drug pricing
	VolatilityEvolving VolatilityClass = "EVOLVING"

	// VolatilityVolatile - Facts that change frequently (weekly or more)
	// Examples: drug availability, shortages, real-time pricing
	VolatilityVolatile VolatilityClass = "VOLATILE"
)

// ExpectedHalfLife indicates how long before a fact should be revalidated
type ExpectedHalfLife string

const (
	HalfLifeYears  ExpectedHalfLife = "YEARS"  // Revalidate annually
	HalfLifeMonths ExpectedHalfLife = "MONTHS" // Revalidate monthly
	HalfLifeWeeks  ExpectedHalfLife = "WEEKS"  // Revalidate weekly
	HalfLifeDays   ExpectedHalfLife = "DAYS"   // Revalidate daily
)

// FactStability tracks refresh expectations and staleness for a fact
// This enables proactive revalidation and prevents "zombie facts"
type FactStability struct {
	// Volatility classification
	Volatility VolatilityClass `json:"volatility" db:"volatility"`

	// Expected refresh interval
	ExpectedHalfLife ExpectedHalfLife `json:"expectedHalfLife" db:"expected_half_life"`

	// Auto-revalidation interval in days (0 = manual only)
	AutoRevalidationDays int `json:"autoRevalidationDays" db:"auto_revalidation_days"`

	// What triggers a refresh (e.g., "FDA_LABEL_UPDATE", "POST_MARKET_SURVEILLANCE")
	ChangeTriggers []string `json:"changeTriggers" db:"-"`

	// When this fact was last refreshed from source
	LastRefreshedAt time.Time `json:"lastRefreshedAt" db:"last_refreshed_at"`

	// Which refresh mechanism was used
	RefreshSource string `json:"refreshSource" db:"refresh_source"`

	// When this fact becomes stale (null = never auto-stale)
	StaleAfter *time.Time `json:"staleAfter,omitempty" db:"stale_after"`

	// Whether the fact is currently considered stale
	IsStale bool `json:"isStale" db:"-"`
}

// DefaultStability returns default stability settings based on fact type
func DefaultStability(factType FactType) FactStability {
	now := time.Now()

	switch factType {
	case FactTypeOrganImpairment, FactTypeSafetySignal:
		// Renal/hepatic dosing and safety signals are relatively stable
		staleAfter := now.AddDate(1, 0, 0) // 1 year
		return FactStability{
			Volatility:           VolatilityStable,
			ExpectedHalfLife:     HalfLifeYears,
			AutoRevalidationDays: 365,
			ChangeTriggers:       []string{"FDA_LABEL_UPDATE", "POST_MARKET_SURVEILLANCE"},
			LastRefreshedAt:      now,
			RefreshSource:        "INITIAL_EXTRACTION",
			StaleAfter:           &staleAfter,
		}

	case FactTypeInteraction:
		// DDI data is fairly stable but monitored
		staleAfter := now.AddDate(0, 6, 0) // 6 months
		return FactStability{
			Volatility:           VolatilityStable,
			ExpectedHalfLife:     HalfLifeMonths,
			AutoRevalidationDays: 180,
			ChangeTriggers:       []string{"NEW_INTERACTION_REPORT", "CLINICAL_STUDY_PUBLICATION"},
			LastRefreshedAt:      now,
			RefreshSource:        "INITIAL_EXTRACTION",
			StaleAfter:           &staleAfter,
		}

	case FactTypeFormulary:
		// Formulary changes monthly/quarterly
		staleAfter := now.AddDate(0, 1, 0) // 1 month
		return FactStability{
			Volatility:           VolatilityEvolving,
			ExpectedHalfLife:     HalfLifeMonths,
			AutoRevalidationDays: 30,
			ChangeTriggers:       []string{"CMS_PUF_UPDATE", "FORMULARY_COMMITTEE_DECISION"},
			LastRefreshedAt:      now,
			RefreshSource:        "INITIAL_EXTRACTION",
			StaleAfter:           &staleAfter,
		}

	case FactTypeLabReference:
		// Lab ranges are very stable
		staleAfter := now.AddDate(2, 0, 0) // 2 years
		return FactStability{
			Volatility:           VolatilityStable,
			ExpectedHalfLife:     HalfLifeYears,
			AutoRevalidationDays: 730,
			ChangeTriggers:       []string{"NHANES_UPDATE", "LOINC_REVISION"},
			LastRefreshedAt:      now,
			RefreshSource:        "INITIAL_EXTRACTION",
			StaleAfter:           &staleAfter,
		}

	case FactTypeReproductiveSafety:
		// Pregnancy categories are relatively stable
		staleAfter := now.AddDate(1, 0, 0) // 1 year
		return FactStability{
			Volatility:           VolatilityStable,
			ExpectedHalfLife:     HalfLifeYears,
			AutoRevalidationDays: 365,
			ChangeTriggers:       []string{"FDA_PLLR_UPDATE", "TERATOGENICITY_STUDY"},
			LastRefreshedAt:      now,
			RefreshSource:        "INITIAL_EXTRACTION",
			StaleAfter:           &staleAfter,
		}

	default:
		// Default to stable with annual review
		staleAfter := now.AddDate(1, 0, 0)
		return FactStability{
			Volatility:           VolatilityStable,
			ExpectedHalfLife:     HalfLifeYears,
			AutoRevalidationDays: 365,
			ChangeTriggers:       []string{},
			LastRefreshedAt:      now,
			RefreshSource:        "INITIAL_EXTRACTION",
			StaleAfter:           &staleAfter,
		}
	}
}

// CheckStaleness determines if a fact is stale based on its stability settings
func (s *FactStability) CheckStaleness() bool {
	if s.StaleAfter == nil {
		return false
	}
	return time.Now().After(*s.StaleAfter)
}

// RefreshDue returns true if the fact is due for revalidation
func (s *FactStability) RefreshDue() bool {
	if s.AutoRevalidationDays == 0 {
		return false // Manual refresh only
	}
	daysElapsed := int(time.Since(s.LastRefreshedAt).Hours() / 24)
	return daysElapsed >= s.AutoRevalidationDays
}

// =============================================================================
// FACT LIFECYCLE STATES
// =============================================================================

// FactStatus represents the lifecycle state of a fact
type FactStatus string

const (
	// StatusDraft - newly extracted, awaiting governance
	StatusDraft FactStatus = "DRAFT"

	// StatusApproved - passed governance, awaiting activation
	StatusApproved FactStatus = "APPROVED"

	// StatusActive - live in production, used by KB-19 runtime
	StatusActive FactStatus = "ACTIVE"

	// StatusSuperseded - replaced by newer version, kept for audit
	StatusSuperseded FactStatus = "SUPERSEDED"

	// StatusDeprecated - no longer valid, kept for audit
	StatusDeprecated FactStatus = "DEPRECATED"

	// StatusArchived - rejected during governance, kept for audit
	StatusArchived FactStatus = "ARCHIVED"
)

// =============================================================================
// FACT SCOPE (Class vs Drug)
// =============================================================================

// FactScope defines whether the fact applies to a drug class or specific drug
type FactScope string

const (
	// ScopeDrug - fact applies to a specific drug (RxCUI)
	ScopeDrug FactScope = "DRUG"

	// ScopeClass - fact applies to a drug class (RxClass)
	ScopeClass FactScope = "CLASS"
)

// =============================================================================
// CONFIDENCE MODEL
// =============================================================================

// ConfidenceBand represents the confidence tier for governance routing
type ConfidenceBand string

const (
	// ConfidenceHigh - score >= 0.85, auto-activate
	ConfidenceHigh ConfidenceBand = "HIGH"

	// ConfidenceMedium - score 0.65-0.84, requires pharmacist review
	ConfidenceMedium ConfidenceBand = "MEDIUM"

	// ConfidenceLow - score < 0.65, auto-reject to archive
	ConfidenceLow ConfidenceBand = "LOW"
)

// FactConfidence captures the confidence score and signals
type FactConfidence struct {
	// ─────────────────────────────────────────────────────────────────────────
	// CONFIDENCE SCORES (Multi-dimensional)
	// ─────────────────────────────────────────────────────────────────────────

	// Overall is the combined confidence value (0.0 to 1.0)
	// This is the primary score used for governance decisions
	Overall float64 `json:"overall" db:"confidence_overall"`

	// SourceQuality measures reliability of the data source (0.0 to 1.0)
	// Higher for authoritative sources (FDA SPL, peer-reviewed), lower for secondary sources
	SourceQuality float64 `json:"sourceQuality" db:"confidence_source_quality"`

	// ExtractionCertainty measures how confident the extractor is (0.0 to 1.0)
	// Higher when extraction is unambiguous, lower for ambiguous/uncertain extractions
	ExtractionCertainty float64 `json:"extractionCertainty" db:"confidence_extraction_certainty"`

	// Score is an alias for Overall (for backward compatibility)
	// Deprecated: Use Overall instead
	Score float64 `json:"score" db:"confidence_score"`

	// ─────────────────────────────────────────────────────────────────────────
	// CONFIDENCE BAND & SIGNALS
	// ─────────────────────────────────────────────────────────────────────────

	// Band is the tier for governance routing (HIGH, MEDIUM, LOW)
	Band ConfidenceBand `json:"band" db:"confidence_band"`

	// Signals are the individual factors contributing to confidence
	Signals []ConfidenceSignal `json:"signals" db:"-"`

	// SourceDiversity indicates how many independent sources agree
	SourceDiversity int `json:"sourceDiversity" db:"source_diversity"`

	// ─────────────────────────────────────────────────────────────────────────
	// HUMAN VERIFICATION
	// ─────────────────────────────────────────────────────────────────────────

	// HumanVerified indicates if a human has reviewed this fact
	HumanVerified bool `json:"humanVerified" db:"human_verified"`

	// VerifiedBy is the ID of the verifier (if human verified)
	VerifiedBy string `json:"verifiedBy,omitempty" db:"verified_by"`

	// VerifiedAt is when human verification occurred
	VerifiedAt *time.Time `json:"verifiedAt,omitempty" db:"verified_at"`

	// CalibrationID identifies which confidence calibration was used
	CalibrationID string `json:"calibrationId" db:"calibration_id"`
}

// ConfidenceSignal represents a single factor in the confidence calculation
type ConfidenceSignal struct {
	// Name is the signal identifier (e.g., "numeric_threshold", "explicit_verb")
	Name string `json:"name"`

	// Weight is how much this signal contributed to the score
	Weight float64 `json:"weight"`

	// Present indicates whether this signal was detected
	Present bool `json:"present"`

	// RawValue is the actual detected value (for audit)
	RawValue string `json:"rawValue,omitempty"`
}

// =============================================================================
// CORE FACT STRUCTURE
// =============================================================================

// Fact represents a single atomic clinical fact in the Canonical Fact Store
type Fact struct {
	// ─────────────────────────────────────────────────────────────────────────
	// IDENTITY
	// ─────────────────────────────────────────────────────────────────────────

	// FactID is the unique identifier (UUID)
	FactID string `json:"factId" db:"fact_id"`

	// FactType categorizes the fact (ORGAN_IMPAIRMENT, SAFETY_SIGNAL, etc.)
	FactType FactType `json:"factType" db:"fact_type"`

	// ─────────────────────────────────────────────────────────────────────────
	// DRUG REFERENCE (Linked to Drug Master Table)
	// ─────────────────────────────────────────────────────────────────────────

	// RxCUI is the RxNorm concept ID (anchors fact to drug universe)
	RxCUI string `json:"rxcui" db:"rxcui"`

	// DrugName is the human-readable name (denormalized for convenience)
	DrugName string `json:"drugName" db:"drug_name"`

	// Scope indicates if this applies to a specific drug or class
	Scope FactScope `json:"scope" db:"scope"`

	// ClassID is the RxClass ID if Scope is CLASS
	ClassID string `json:"classId,omitempty" db:"class_id"`

	// ─────────────────────────────────────────────────────────────────────────
	// CONTENT (JSONB - type-specific payload)
	// ─────────────────────────────────────────────────────────────────────────

	// Content is the type-specific fact data (stored as JSONB)
	Content json.RawMessage `json:"content" db:"content"`

	// ─────────────────────────────────────────────────────────────────────────
	// CONFIDENCE & GOVERNANCE
	// ─────────────────────────────────────────────────────────────────────────

	// Confidence captures the extraction confidence and signals
	Confidence FactConfidence `json:"confidence" db:"-"`

	// Status is the lifecycle state (DRAFT, APPROVED, ACTIVE, etc.)
	Status FactStatus `json:"status" db:"status"`

	// ─────────────────────────────────────────────────────────────────────────
	// TEMPORAL VERSIONING
	// ─────────────────────────────────────────────────────────────────────────

	// EffectiveFrom is when this fact became/becomes active
	EffectiveFrom time.Time `json:"effectiveFrom" db:"effective_from"`

	// EffectiveTo is when this fact expires (null = no expiry)
	EffectiveTo *time.Time `json:"effectiveTo,omitempty" db:"effective_to"`

	// SupersededBy is the FactID of the newer version (if superseded)
	SupersededBy *string `json:"supersededBy,omitempty" db:"superseded_by"`

	// Version is the version number within the fact's lineage
	Version int `json:"version" db:"version"`

	// ─────────────────────────────────────────────────────────────────────────
	// PROVENANCE (Extraction Audit Trail)
	// ─────────────────────────────────────────────────────────────────────────

	// ExtractorID identifies which extractor produced this fact
	ExtractorID string `json:"extractorId" db:"extractor_id"`

	// ExtractorVersion is the version of the extractor
	ExtractorVersion string `json:"extractorVersion" db:"extractor_version"`

	// SourceURL is the original data source URL
	SourceURL string `json:"sourceUrl" db:"source_url"`

	// SourceVersion is the version/date of the source document
	SourceVersion string `json:"sourceVersion" db:"source_version"`

	// EvidenceID links to the original evidence unit
	EvidenceID string `json:"evidenceId" db:"evidence_id"`

	// ─────────────────────────────────────────────────────────────────────────
	// REGULATORY JURISDICTION
	// ─────────────────────────────────────────────────────────────────────────

	// Jurisdictions is the list of regulatory jurisdictions (e.g., ["US", "AU", "IN"])
	Jurisdictions []string `json:"jurisdictions" db:"-"`

	// RegulatoryBody is the primary regulatory authority (e.g., "FDA", "TGA", "CDSCO")
	RegulatoryBody string `json:"regulatoryBody" db:"regulatory_body"`

	// ─────────────────────────────────────────────────────────────────────────
	// FACT STABILITY (Gap 1 - Clinical Knowledge OS)
	// ─────────────────────────────────────────────────────────────────────────

	// Stability tracks volatility and refresh expectations for this fact
	Stability FactStability `json:"stability" db:"-"`

	// ─────────────────────────────────────────────────────────────────────────
	// AUDIT TIMESTAMPS
	// ─────────────────────────────────────────────────────────────────────────

	// CreatedAt is when the fact was created
	CreatedAt time.Time `json:"createdAt" db:"created_at"`

	// UpdatedAt is when the fact was last updated
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`

	// ApprovedAt is when the fact was approved (if applicable)
	ApprovedAt *time.Time `json:"approvedAt,omitempty" db:"approved_at"`

	// ApprovedBy is who approved the fact (if applicable)
	ApprovedBy string `json:"approvedBy,omitempty" db:"approved_by"`
}

// =============================================================================
// FACT CONTENT TYPES (Type-specific payloads for Content field)
// =============================================================================

// OrganImpairmentContent is the content for ORGAN_IMPAIRMENT facts
type OrganImpairmentContent struct {
	// OrganSystem is the affected organ (RENAL, HEPATIC, CARDIAC, etc.)
	OrganSystem string `json:"organSystem"`

	// ImpairmentLevel describes severity (MILD, MODERATE, SEVERE, ESRD, etc.)
	ImpairmentLevel string `json:"impairmentLevel"`

	// ThresholdType is the measurement type (EGFR, CLCR, CHILD_PUGH, etc.)
	ThresholdType string `json:"thresholdType"`

	// ThresholdOperator is the comparison (<, <=, >, >=, ==, BETWEEN)
	ThresholdOperator string `json:"thresholdOperator"`

	// ThresholdValue is the numeric threshold
	ThresholdValue float64 `json:"thresholdValue"`

	// ThresholdValueHigh is the upper bound for BETWEEN operator
	ThresholdValueHigh float64 `json:"thresholdValueHigh,omitempty"`

	// ThresholdUnit is the unit of measurement (mL/min, mL/min/1.73m2, etc.)
	ThresholdUnit string `json:"thresholdUnit"`

	// Recommendation is the action to take (REDUCE_DOSE, AVOID, MONITOR, etc.)
	Recommendation string `json:"recommendation"`

	// AdjustmentFactor is the dose multiplier (e.g., 0.5 for 50% reduction)
	AdjustmentFactor float64 `json:"adjustmentFactor,omitempty"`

	// MaxDose is the maximum allowed dose (if applicable)
	MaxDose string `json:"maxDose,omitempty"`

	// Frequency is the adjusted frequency (if applicable)
	Frequency string `json:"frequency,omitempty"`

	// RawText is the original extracted text (for audit)
	RawText string `json:"rawText"`

	// SPLSection is the source section in SPL document
	SPLSection string `json:"splSection,omitempty"`
}

// SafetySignalContent is the content for SAFETY_SIGNAL facts
type SafetySignalContent struct {
	// SignalType is the category (BOXED_WARNING, CONTRAINDICATION, WARNING, PRECAUTION)
	SignalType string `json:"signalType"`

	// Severity is the clinical severity (CRITICAL, HIGH, MEDIUM, LOW)
	Severity string `json:"severity"`

	// Condition is the clinical condition triggering the signal
	Condition string `json:"condition"`

	// Action is the recommended action (AVOID, CONTRAINDICATED, MONITOR, CAUTION)
	Action string `json:"action"`

	// Population is the affected population (if specific)
	Population string `json:"population,omitempty"`

	// Mechanism is the clinical mechanism (if known)
	Mechanism string `json:"mechanism,omitempty"`

	// RawText is the original warning text
	RawText string `json:"rawText"`

	// SPLSection is the source section in SPL document
	SPLSection string `json:"splSection,omitempty"`
}

// ReproductiveSafetyContent is the content for REPRODUCTIVE_SAFETY facts
type ReproductiveSafetyContent struct {
	// Category is the pregnancy category (A, B, C, D, X, or newer PLLR)
	Category string `json:"category"`

	// Trimester is the affected trimester (1, 2, 3, ALL)
	Trimester string `json:"trimester,omitempty"`

	// LactationRisk describes breastfeeding safety
	LactationRisk string `json:"lactationRisk,omitempty"`

	// FertilityImpact describes fertility effects
	FertilityImpact string `json:"fertilityImpact,omitempty"`

	// TeratogenicRisk indicates known teratogenic effects
	TeratogenicRisk bool `json:"teratogenicRisk"`

	// Recommendation is the clinical recommendation
	Recommendation string `json:"recommendation"`

	// RawText is the original text
	RawText string `json:"rawText"`
}

// InteractionContent is the content for INTERACTION facts
type InteractionContent struct {
	// InteractionType is the category (DRUG_DRUG, DRUG_FOOD, DRUG_LAB, DRUG_DISEASE)
	InteractionType string `json:"interactionType"`

	// InteractantRxCUI is the RxCUI of the interacting drug
	InteractantRxCUI string `json:"interactantRxcui"`

	// InteractantName is the name of the interacting substance
	InteractantName string `json:"interactantName"`

	// Severity is the interaction severity (CRITICAL, MAJOR, MODERATE, MINOR)
	Severity string `json:"severity"`

	// Mechanism is the pharmacological mechanism
	Mechanism string `json:"mechanism,omitempty"`

	// ClinicalEffect describes the clinical outcome
	ClinicalEffect string `json:"clinicalEffect"`

	// Management is the recommended management strategy
	Management string `json:"management"`

	// EvidenceLevel is the quality of evidence (HIGH, MODERATE, LOW, THEORETICAL)
	EvidenceLevel string `json:"evidenceLevel"`

	// Source is the interaction database source (DrugBank, MED-RT, etc.)
	Source string `json:"source"`

	// ─────────────────────────────────────────────────────────────────────────
	// DIRECTIONALITY (Perpetrator vs Victim - Review Refinement)
	// ─────────────────────────────────────────────────────────────────────────
	// Critical for clinical decision support: some interactions are bidirectional
	// (both drugs contribute), while others are unidirectional (only one drug
	// needs adjustment, e.g., CYP3A4 inhibitor + substrate).

	// AffectedDrugRxCUI identifies which drug requires adjustment (if unidirectional)
	// If NULL/empty, the interaction is bidirectional (both drugs contribute equally)
	// If populated, only this drug needs dose adjustment or monitoring
	AffectedDrugRxCUI *string `json:"affectedDrugRxcui,omitempty"`

	// InteractionMechanism provides structured mechanism classification for directionality
	// Examples: "CYP3A4_INHIBITION", "CYP2D6_INDUCTION", "QT_ADDITIVE", "BLEEDING_ADDITIVE"
	InteractionMechanism string `json:"interactionMechanism,omitempty"`

	// IsBidirectional indicates if both drugs equally contribute to the interaction
	// When false and AffectedDrugRxCUI is set, only that drug needs clinical action
	IsBidirectional bool `json:"isBidirectional"`

	// PrecipitantRxCUI is the drug that causes the interaction (perpetrator)
	// The precipitant affects the object drug's pharmacokinetics or pharmacodynamics
	PrecipitantRxCUI *string `json:"precipitantRxcui,omitempty"`

	// ObjectRxCUI is the drug that is affected by the interaction (victim)
	// This is the drug whose levels/effects are altered
	ObjectRxCUI *string `json:"objectRxcui,omitempty"`
}

// FormularyContent is the content for FORMULARY facts
type FormularyContent struct {
	// FormularyID identifies the formulary
	FormularyID string `json:"formularyId"`

	// FormularyName is the name of the formulary
	FormularyName string `json:"formularyName"`

	// Tier is the formulary tier (1, 2, 3, etc.)
	Tier int `json:"tier"`

	// CoverageStatus indicates if covered (COVERED, NOT_COVERED, CONDITIONAL)
	CoverageStatus string `json:"coverageStatus"`

	// PriorAuthRequired indicates if prior authorization is needed
	PriorAuthRequired bool `json:"priorAuthRequired"`

	// StepTherapyRequired indicates if step therapy is required
	StepTherapyRequired bool `json:"stepTherapyRequired"`

	// QuantityLimits describes any quantity limits
	QuantityLimits string `json:"quantityLimits,omitempty"`

	// Copay is the copay amount (if known)
	Copay float64 `json:"copay,omitempty"`

	// EffectiveDate is when this formulary entry is effective
	EffectiveDate time.Time `json:"effectiveDate"`

	// ExpirationDate is when this entry expires
	ExpirationDate *time.Time `json:"expirationDate,omitempty"`
}

// LabReferenceContent is the content for LAB_REFERENCE facts
type LabReferenceContent struct {
	// LOINCCode is the LOINC code for the lab test
	LOINCCode string `json:"loincCode"`

	// LabName is the human-readable lab test name
	LabName string `json:"labName"`

	// ReferenceRangeLow is the lower bound of normal
	ReferenceRangeLow float64 `json:"referenceRangeLow"`

	// ReferenceRangeHigh is the upper bound of normal
	ReferenceRangeHigh float64 `json:"referenceRangeHigh"`

	// Unit is the measurement unit
	Unit string `json:"unit"`

	// CriticalLow is the critical low value (requires immediate action)
	CriticalLow float64 `json:"criticalLow,omitempty"`

	// CriticalHigh is the critical high value (requires immediate action)
	CriticalHigh float64 `json:"criticalHigh,omitempty"`

	// Population describes the reference population (adult, pediatric, etc.)
	Population string `json:"population,omitempty"`

	// MonitoringFrequency describes how often to monitor when on this drug
	MonitoringFrequency string `json:"monitoringFrequency,omitempty"`

	// BaselineRequired indicates if baseline measurement is required
	BaselineRequired bool `json:"baselineRequired"`

	// Source is the reference range source (NHANES, lab-specific, etc.)
	Source string `json:"source"`
}

// =============================================================================
// DRUG MASTER TABLE (Layer 0 - Drug Universe)
// =============================================================================

// Drug represents an entry in the Drug Master Table
type Drug struct {
	// RxCUI is the RxNorm concept unique identifier
	RxCUI string `json:"rxcui" db:"rxcui"`

	// Name is the drug name
	Name string `json:"name" db:"name"`

	// TTY is the RxNorm term type (SCD, SBD, GPCK, etc.)
	TTY string `json:"tty" db:"tty"`

	// IsGeneric indicates if this is a generic drug
	IsGeneric bool `json:"isGeneric" db:"is_generic"`

	// BrandNames contains associated brand names
	BrandNames []string `json:"brandNames" db:"-"`

	// ActiveIngredients lists the active ingredients
	ActiveIngredients []string `json:"activeIngredients" db:"-"`

	// DrugClasses contains RxClass classifications
	DrugClasses []DrugClass `json:"drugClasses" db:"-"`

	// NDCs contains National Drug Codes
	NDCs []string `json:"ndcs" db:"-"`

	// SPLSetID is the DailyMed SPL Set ID for label retrieval
	SPLSetID string `json:"splSetId,omitempty" db:"spl_set_id"`

	// Status indicates if the drug is active in the formulary
	Status string `json:"status" db:"status"`

	// LastSyncedAt is when this drug was last synced from RxNav
	LastSyncedAt time.Time `json:"lastSyncedAt" db:"last_synced_at"`

	// CreatedAt is when this drug was added
	CreatedAt time.Time `json:"createdAt" db:"created_at"`

	// UpdatedAt is when this drug was last updated
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// DrugClass represents a drug classification from RxClass
type DrugClass struct {
	// ClassID is the class identifier
	ClassID string `json:"classId"`

	// ClassName is the human-readable class name
	ClassName string `json:"className"`

	// ClassType is the classification system (ATC, VA, EPC, etc.)
	ClassType string `json:"classType"`

	// Relation describes how the drug relates to the class
	Relation string `json:"relation"`
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// CalculateConfidenceBand determines the band from a score
func CalculateConfidenceBand(score float64) ConfidenceBand {
	switch {
	case score >= 0.85:
		return ConfidenceHigh
	case score >= 0.65:
		return ConfidenceMedium
	default:
		return ConfidenceLow
	}
}

// NewFact creates a new Fact with defaults
func NewFact(factType FactType, rxcui string, drugName string) *Fact {
	now := time.Now()
	return &Fact{
		FactType:      factType,
		RxCUI:         rxcui,
		DrugName:      drugName,
		Scope:         ScopeDrug,
		Status:        StatusDraft,
		Version:       1,
		EffectiveFrom: now,
		CreatedAt:     now,
		UpdatedAt:     now,
		Jurisdictions: []string{"US"},
		Stability:     DefaultStability(factType), // Gap 1: Auto-assign stability based on fact type
	}
}

// NewFactWithStability creates a new Fact with custom stability settings
func NewFactWithStability(factType FactType, rxcui string, drugName string, stability FactStability) *Fact {
	fact := NewFact(factType, rxcui, drugName)
	fact.Stability = stability
	return fact
}

// SetContent marshals and sets the content field
func (f *Fact) SetContent(content interface{}) error {
	data, err := json.Marshal(content)
	if err != nil {
		return err
	}
	f.Content = data
	return nil
}

// GetOrganImpairmentContent unmarshals content as OrganImpairmentContent
func (f *Fact) GetOrganImpairmentContent() (*OrganImpairmentContent, error) {
	if f.FactType != FactTypeOrganImpairment {
		return nil, nil
	}
	var content OrganImpairmentContent
	if err := json.Unmarshal(f.Content, &content); err != nil {
		return nil, err
	}
	return &content, nil
}

// GetSafetySignalContent unmarshals content as SafetySignalContent
func (f *Fact) GetSafetySignalContent() (*SafetySignalContent, error) {
	if f.FactType != FactTypeSafetySignal {
		return nil, nil
	}
	var content SafetySignalContent
	if err := json.Unmarshal(f.Content, &content); err != nil {
		return nil, err
	}
	return &content, nil
}

// GetInteractionContent unmarshals content as InteractionContent
func (f *Fact) GetInteractionContent() (*InteractionContent, error) {
	if f.FactType != FactTypeInteraction {
		return nil, nil
	}
	var content InteractionContent
	if err := json.Unmarshal(f.Content, &content); err != nil {
		return nil, err
	}
	return &content, nil
}
