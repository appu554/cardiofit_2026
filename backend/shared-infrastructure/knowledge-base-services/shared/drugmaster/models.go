// Package drugmaster provides the Drug Master Table - the canonical drug registry
// that serves as the foundation for all Knowledge Base services.
//
// DESIGN PRINCIPLE: "Layer 0 - Single Source of Truth for Drug Identity"
// Every KB references drugs through this canonical registry, ensuring semantic
// consistency across the entire platform.
package drugmaster

import (
	"time"
)

// =============================================================================
// DRUG MASTER TABLE - CANONICAL DRUG REGISTRY
// =============================================================================

// DrugMaster represents a single canonical drug entry
// This is the foundation that all KBs reference
type DrugMaster struct {
	// ─────────────────────────────────────────────────────────────────────────
	// PRIMARY IDENTIFIER (RxNorm CUI)
	// ─────────────────────────────────────────────────────────────────────────

	// RxCUI is the RxNorm Concept Unique Identifier - the primary key
	RxCUI string `json:"rxcui" db:"rxcui"`

	// ─────────────────────────────────────────────────────────────────────────
	// NAMES
	// ─────────────────────────────────────────────────────────────────────────

	// DrugName is the canonical drug name
	DrugName string `json:"drugName" db:"drug_name"`

	// GenericName is the non-proprietary name
	GenericName string `json:"genericName,omitempty" db:"generic_name"`

	// BrandNames lists proprietary/trade names
	BrandNames []string `json:"brandNames,omitempty" db:"brand_names"`

	// Synonyms lists alternative names
	Synonyms []string `json:"synonyms,omitempty" db:"synonyms"`

	// ─────────────────────────────────────────────────────────────────────────
	// RXNORM CLASSIFICATION
	// ─────────────────────────────────────────────────────────────────────────

	// TTY is the RxNorm Term Type
	TTY RxNormTTY `json:"tty" db:"tty"`

	// IngredientRxCUIs are the active ingredient RxCUIs (for products)
	IngredientRxCUIs []string `json:"ingredientRxcuis,omitempty" db:"ingredient_rxcuis"`

	// ParentRxCUI is the parent concept (e.g., SCD parent of SBD)
	ParentRxCUI string `json:"parentRxcui,omitempty" db:"parent_rxcui"`

	// ─────────────────────────────────────────────────────────────────────────
	// THERAPEUTIC CLASSIFICATION
	// ─────────────────────────────────────────────────────────────────────────

	// ATCCodes are WHO ATC classification codes
	ATCCodes []string `json:"atcCodes,omitempty" db:"atc_codes"`

	// TherapeuticClass is the primary therapeutic class
	TherapeuticClass string `json:"therapeuticClass,omitempty" db:"therapeutic_class"`

	// PharmacologicClass is the FDA Established Pharmacologic Class
	PharmacologicClass string `json:"pharmacologicClass,omitempty" db:"pharmacologic_class"`

	// MechanismOfAction describes how the drug works
	MechanismOfAction string `json:"mechanismOfAction,omitempty" db:"mechanism_of_action"`

	// ─────────────────────────────────────────────────────────────────────────
	// FORMULATION
	// ─────────────────────────────────────────────────────────────────────────

	// DoseForm is the pharmaceutical form (tablet, capsule, injection, etc.)
	DoseForm string `json:"doseForm,omitempty" db:"dose_form"`

	// Strength is the drug strength (e.g., "10 mg", "500 mg/5 mL")
	Strength string `json:"strength,omitempty" db:"strength"`

	// RouteOfAdministration is the administration route
	RouteOfAdministration string `json:"routeOfAdministration,omitempty" db:"route_of_administration"`

	// ─────────────────────────────────────────────────────────────────────────
	// CROSS-REFERENCES
	// ─────────────────────────────────────────────────────────────────────────

	// NDCs are the National Drug Codes
	NDCs []string `json:"ndcs,omitempty" db:"ndcs"`

	// SPLSetIDs are the DailyMed SPL Set IDs
	SPLSetIDs []string `json:"splSetIds,omitempty" db:"spl_set_ids"`

	// DrugBankID is the DrugBank identifier
	DrugBankID string `json:"drugbankId,omitempty" db:"drugbank_id"`

	// UNII is the FDA Unique Ingredient Identifier
	UNII string `json:"unii,omitempty" db:"unii"`

	// SNOMEDCodes are SNOMED CT codes
	SNOMEDCodes []string `json:"snomedCodes,omitempty" db:"snomed_codes"`

	// ─────────────────────────────────────────────────────────────────────────
	// CLINICAL FLAGS (KB-SPECIFIC TAGGING)
	// ─────────────────────────────────────────────────────────────────────────

	// RenalRelevance indicates if the drug requires renal consideration
	RenalRelevance RenalRelevance `json:"renalRelevance" db:"renal_relevance"`

	// HepaticRelevance indicates if the drug requires hepatic consideration
	HepaticRelevance HepaticRelevance `json:"hepaticRelevance" db:"hepatic_relevance"`

	// HasBoxedWarning indicates if the drug has an FDA boxed warning
	HasBoxedWarning bool `json:"hasBoxedWarning" db:"has_boxed_warning"`

	// HasInteractions indicates if the drug has known interactions
	HasInteractions bool `json:"hasInteractions" db:"has_interactions"`

	// PregnancyCategory is the FDA pregnancy category (legacy + new system)
	PregnancyCategory string `json:"pregnancyCategory,omitempty" db:"pregnancy_category"`

	// ControlledSubstance indicates DEA schedule
	ControlledSubstance string `json:"controlledSubstance,omitempty" db:"controlled_substance"`

	// ─────────────────────────────────────────────────────────────────────────
	// STATUS & METADATA
	// ─────────────────────────────────────────────────────────────────────────

	// Status indicates the drug's market status
	Status DrugStatus `json:"status" db:"status"`

	// Source identifies where this record came from
	Source string `json:"source" db:"source"`

	// LastUpdated is when this record was last updated
	LastUpdated time.Time `json:"lastUpdated" db:"last_updated"`

	// CreatedAt is when this record was created
	CreatedAt time.Time `json:"createdAt" db:"created_at"`

	// Version for optimistic locking
	Version int `json:"version" db:"version"`
}

// =============================================================================
// ENUMS AND TYPES
// =============================================================================

// RxNormTTY represents RxNorm Term Types
type RxNormTTY string

const (
	// Ingredient Term Types
	TTYIngredient    RxNormTTY = "IN"  // Ingredient
	TTYMinIngredient RxNormTTY = "MIN" // Multiple Ingredients

	// Clinical Drug Term Types
	TTYSCDC RxNormTTY = "SCDC" // Semantic Clinical Drug Component
	TTYSCD  RxNormTTY = "SCD"  // Semantic Clinical Drug
	TTYSBDC RxNormTTY = "SBDC" // Semantic Branded Drug Component
	TTYSBD  RxNormTTY = "SBD"  // Semantic Branded Drug

	// Pack Term Types
	TTYBPCK RxNormTTY = "BPCK" // Branded Pack
	TTYGPCK RxNormTTY = "GPCK" // Generic Pack

	// Dose Form Term Types
	TTYDF RxNormTTY = "DF" // Dose Form
	TTYBN RxNormTTY = "BN" // Brand Name
)

// RenalRelevance indicates the renal clinical relevance of a drug
type RenalRelevance string

const (
	RenalRelevanceUnknown RenalRelevance = "UNKNOWN" // Not yet evaluated
	RenalRelevanceNone    RenalRelevance = "NONE"    // No renal concerns
	RenalRelevanceMonitor RenalRelevance = "MONITOR" // Monitor renal function
	RenalRelevanceAdjust  RenalRelevance = "ADJUST"  // Dose adjustment may be needed
	RenalRelevanceAvoid   RenalRelevance = "AVOID"   // Avoid in renal impairment
)

// HepaticRelevance indicates the hepatic clinical relevance of a drug
type HepaticRelevance string

const (
	HepaticRelevanceUnknown HepaticRelevance = "UNKNOWN"
	HepaticRelevanceNone    HepaticRelevance = "NONE"
	HepaticRelevanceMonitor HepaticRelevance = "MONITOR"
	HepaticRelevanceAdjust  HepaticRelevance = "ADJUST"
	HepaticRelevanceAvoid   HepaticRelevance = "AVOID"
)

// DrugStatus indicates the market status of a drug
type DrugStatus string

const (
	DrugStatusActive       DrugStatus = "ACTIVE"       // Currently marketed
	DrugStatusDiscontinued DrugStatus = "DISCONTINUED" // No longer marketed
	DrugStatusPending      DrugStatus = "PENDING"      // Awaiting approval
	DrugStatusWithdrawn    DrugStatus = "WITHDRAWN"    // Withdrawn from market
)

// =============================================================================
// RENAL INTENT (for KB-1)
// =============================================================================

// RenalIntent represents the clinical intent for renal dosing
type RenalIntent string

const (
	// RenalIntentAbsolute means absolute contraindication in renal impairment
	RenalIntentAbsolute RenalIntent = "ABSOLUTE"

	// RenalIntentAdjust means dose adjustment required based on renal function
	RenalIntentAdjust RenalIntent = "ADJUST"

	// RenalIntentMonitor means monitor renal function but no dose change
	RenalIntentMonitor RenalIntent = "MONITOR"

	// RenalIntentNone means no renal concerns identified
	RenalIntentNone RenalIntent = "NONE"
)

// =============================================================================
// DRUG MASTER REPOSITORY INTERFACE
// =============================================================================

// Repository defines the interface for Drug Master Table operations
type Repository interface {
	// ─────────────────────────────────────────────────────────────────────────
	// CRUD OPERATIONS
	// ─────────────────────────────────────────────────────────────────────────

	// GetByRxCUI retrieves a drug by RxCUI
	GetByRxCUI(rxcui string) (*DrugMaster, error)

	// GetByNDC retrieves a drug by NDC
	GetByNDC(ndc string) (*DrugMaster, error)

	// GetByName searches for drugs by name (fuzzy match)
	GetByName(name string, limit int) ([]DrugMaster, error)

	// Create adds a new drug to the registry
	Create(drug *DrugMaster) error

	// Update modifies an existing drug
	Update(drug *DrugMaster) error

	// Delete removes a drug (soft delete)
	Delete(rxcui string) error

	// ─────────────────────────────────────────────────────────────────────────
	// BULK OPERATIONS
	// ─────────────────────────────────────────────────────────────────────────

	// BatchGet retrieves multiple drugs by RxCUI
	BatchGet(rxcuis []string) (map[string]*DrugMaster, error)

	// BatchCreate adds multiple drugs
	BatchCreate(drugs []*DrugMaster) error

	// BatchUpdate updates multiple drugs
	BatchUpdate(drugs []*DrugMaster) error

	// ─────────────────────────────────────────────────────────────────────────
	// CLINICAL QUERIES
	// ─────────────────────────────────────────────────────────────────────────

	// GetRenalRelevantDrugs returns all drugs requiring renal consideration
	GetRenalRelevantDrugs() ([]DrugMaster, error)

	// GetHepaticRelevantDrugs returns all drugs requiring hepatic consideration
	GetHepaticRelevantDrugs() ([]DrugMaster, error)

	// GetDrugsWithBoxedWarnings returns all drugs with boxed warnings
	GetDrugsWithBoxedWarnings() ([]DrugMaster, error)

	// GetDrugsByTherapeuticClass returns drugs in a therapeutic class
	GetDrugsByTherapeuticClass(class string) ([]DrugMaster, error)

	// GetDrugsByATCCode returns drugs with a specific ATC code prefix
	GetDrugsByATCCode(atcPrefix string) ([]DrugMaster, error)

	// ─────────────────────────────────────────────────────────────────────────
	// STATISTICS
	// ─────────────────────────────────────────────────────────────────────────

	// Count returns the total number of drugs
	Count() (int64, error)

	// CountByStatus returns counts by drug status
	CountByStatus() (map[DrugStatus]int64, error)

	// CountByRenalRelevance returns counts by renal relevance
	CountByRenalRelevance() (map[RenalRelevance]int64, error)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// NewDrugMaster creates a new drug entry with defaults
func NewDrugMaster(rxcui string, drugName string) *DrugMaster {
	now := time.Now()
	return &DrugMaster{
		RxCUI:            rxcui,
		DrugName:         drugName,
		Status:           DrugStatusActive,
		RenalRelevance:   RenalRelevanceUnknown,
		HepaticRelevance: HepaticRelevanceUnknown,
		CreatedAt:        now,
		LastUpdated:      now,
		Version:          1,
		BrandNames:       []string{},
		Synonyms:         []string{},
		IngredientRxCUIs: []string{},
		ATCCodes:         []string{},
		NDCs:             []string{},
		SPLSetIDs:        []string{},
		SNOMEDCodes:      []string{},
	}
}

// IsRenalRelevant returns true if the drug requires renal consideration
func (d *DrugMaster) IsRenalRelevant() bool {
	return d.RenalRelevance != RenalRelevanceNone && d.RenalRelevance != RenalRelevanceUnknown
}

// IsHepaticRelevant returns true if the drug requires hepatic consideration
func (d *DrugMaster) IsHepaticRelevant() bool {
	return d.HepaticRelevance != HepaticRelevanceNone && d.HepaticRelevance != HepaticRelevanceUnknown
}

// NeedsRenalDoseAdjustment returns true if dose adjustment is needed for renal impairment
func (d *DrugMaster) NeedsRenalDoseAdjustment() bool {
	return d.RenalRelevance == RenalRelevanceAdjust || d.RenalRelevance == RenalRelevanceAvoid
}

// NeedsHepaticDoseAdjustment returns true if dose adjustment is needed for hepatic impairment
func (d *DrugMaster) NeedsHepaticDoseAdjustment() bool {
	return d.HepaticRelevance == HepaticRelevanceAdjust || d.HepaticRelevance == HepaticRelevanceAvoid
}

// GetRenalIntent returns the renal intent for KB-1
func (d *DrugMaster) GetRenalIntent() RenalIntent {
	switch d.RenalRelevance {
	case RenalRelevanceAvoid:
		return RenalIntentAbsolute
	case RenalRelevanceAdjust:
		return RenalIntentAdjust
	case RenalRelevanceMonitor:
		return RenalIntentMonitor
	default:
		return RenalIntentNone
	}
}

// IsIngredient returns true if this is an ingredient-level concept
func (d *DrugMaster) IsIngredient() bool {
	return d.TTY == TTYIngredient || d.TTY == TTYMinIngredient
}

// IsClinicalDrug returns true if this is a clinical drug concept
func (d *DrugMaster) IsClinicalDrug() bool {
	return d.TTY == TTYSCD || d.TTY == TTYSCDC
}

// IsBrandedDrug returns true if this is a branded drug concept
func (d *DrugMaster) IsBrandedDrug() bool {
	return d.TTY == TTYSBD || d.TTY == TTYSBDC
}

// HasSPL returns true if the drug has associated SPL documents
func (d *DrugMaster) HasSPL() bool {
	return len(d.SPLSetIDs) > 0
}

// =============================================================================
// DRUG MASTER SYNC STATUS
// =============================================================================

// SyncStatus tracks the synchronization status of the drug registry
type SyncStatus struct {
	// LastFullSync is when the last complete sync occurred
	LastFullSync time.Time `json:"lastFullSync"`

	// LastIncrementalSync is when the last incremental sync occurred
	LastIncrementalSync time.Time `json:"lastIncrementalSync"`

	// TotalDrugs is the count after last sync
	TotalDrugs int64 `json:"totalDrugs"`

	// DrugsByStatus counts drugs by status
	DrugsByStatus map[DrugStatus]int64 `json:"drugsByStatus"`

	// SyncErrors records any errors from last sync
	SyncErrors []string `json:"syncErrors,omitempty"`

	// Source identifies the sync source
	Source string `json:"source"`
}
