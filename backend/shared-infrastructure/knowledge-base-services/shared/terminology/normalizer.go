// Package terminology provides ontology-grounded terminology normalization services.
//
// Phase 3: Ontology-Grounded Terminology Normalization
//
// This package addresses three critical issues:
//   - Issue 1: FK Constraint Failures - Wrong RxCUI from SPL XML
//   - Issue 2: No FAERS Compatibility - Missing MedDRA PT codes
//   - Issue 3: Regex Ceiling - 47% noise reduction max with pattern matching
//
// Architecture:
//   - Layer 1: MedDRA SQLite dictionary (80,000+ official terms, sub-millisecond)
//   - Layer 2: RxNav-in-a-Box (localhost:4000, drug validation/correction)
//
// Both layers are deterministic, regulatory-grade, and free for non-commercial use.
package terminology

import (
	"context"
)

// =============================================================================
// DRUG NORMALIZATION (Issue 1 Fix: FK Constraint)
// =============================================================================

// NormalizedDrug contains validated drug information with canonical RxCUI.
// This fixes the FK constraint failures where SPL XML contains wrong/outdated RxCUIs.
//
// Example:
//
//	SPL says:    Lithium → RxCUI 5521 (WRONG - that's hydroxychloroquine!)
//	drug_master: Lithium → RxCUI 6448 (CORRECT)
//	After fix:   CanonicalRxCUI = "6448", WasCorrected = true
type NormalizedDrug struct {
	// CanonicalName is the clean drug name from RxNav
	// Example: "Lithium" (not "These highlights do not include all information...")
	CanonicalName string `json:"canonicalName"`

	// CanonicalRxCUI is the CORRECT RxCUI validated against drug_master
	// Example: "6448" for Lithium (not "5521" from SPL)
	CanonicalRxCUI string `json:"canonicalRxCUI"`

	// OriginalRxCUI is what the SPL XML claimed (may be wrong)
	// Example: "5521" (wrong RxCUI from SPL)
	OriginalRxCUI string `json:"originalRxCUI"`

	// WasCorrected indicates if we had to fix the RxCUI
	// true means SPL had wrong RxCUI, we corrected it via RxNav lookup
	WasCorrected bool `json:"wasCorrected"`

	// GenericName is the ingredient name (e.g., "lithium carbonate")
	GenericName string `json:"genericName,omitempty"`

	// BrandNames are associated brand names
	BrandNames []string `json:"brandNames,omitempty"`

	// TTY is the RxNorm term type (IN=Ingredient, SCD=Clinical Drug, etc.)
	TTY string `json:"tty,omitempty"`

	// Confidence is 1.0 for exact RxNav match, 0.95 for corrected
	Confidence float64 `json:"confidence"`

	// Source indicates where the normalization came from
	// "RXNAV_LOCAL" for RxNav-in-a-Box Docker instance
	Source string `json:"source"`
}

// DrugNormalizer validates and corrects drug RxCUIs using RxNav.
// This is the fix for Issue 1: FK Constraint Failures.
type DrugNormalizer interface {
	// ValidateAndNormalize checks RxCUI against RxNav, corrects if wrong.
	//
	// Parameters:
	//   - rxcui: The RxCUI from SPL XML (may be wrong)
	//   - drugName: The drug name to use for lookup if RxCUI is invalid
	//
	// Returns:
	//   - NormalizedDrug with CanonicalRxCUI that matches drug_master
	//   - error if drug cannot be found in RxNav
	//
	// Example:
	//   Input:  rxcui="5521", drugName="Lithium"
	//   Output: CanonicalRxCUI="6448", WasCorrected=true
	ValidateAndNormalize(ctx context.Context, rxcui, drugName string) (*NormalizedDrug, error)

	// GetCanonicalRxCUI looks up the correct RxCUI by drug name only.
	// Use this when you don't have an RxCUI to validate.
	GetCanonicalRxCUI(ctx context.Context, drugName string) (string, error)
}

// =============================================================================
// ADVERSE EVENT NORMALIZATION (Issue 2 & 3 Fix: FAERS + Noise)
// =============================================================================

// NormalizedAdverseEvent contains normalized adverse event with MedDRA codes.
// This fixes Issue 2 (FAERS compatibility) and Issue 3 (regex ceiling).
//
// MedDRA Hierarchy (for context):
//
//	SOC (System Organ Class)          e.g., "Gastrointestinal disorders"
//	  └─ HLGT (High Level Group Term) e.g., "Gastrointestinal signs and symptoms"
//	       └─ HLT (High Level Term)   e.g., "Nausea and vomiting symptoms"
//	            └─ PT (Preferred Term) e.g., "Nausea" (10028813) ← FAERS uses this
//	                 └─ LLT (Lowest Level Term) e.g., "feeling nauseous"
type NormalizedAdverseEvent struct {
	// CanonicalName is the normalized PT (Preferred Term) name
	// Example: "Nausea" (not "feeling nauseous")
	CanonicalName string `json:"canonicalName"`

	// OriginalText is the raw text from SPL table
	// Example: "feeling nauseous"
	OriginalText string `json:"originalText"`

	// MedDRAPT is the Preferred Term code (FAERS uses this!)
	// Example: "10028813" for Nausea
	MedDRAPT string `json:"meddraPT"`

	// MedDRAName is the official PT name
	// Example: "Nausea"
	MedDRAName string `json:"meddraName"`

	// MedDRALLT is the Lowest Level Term code (if matched at LLT level)
	// Example: "10016261" for "feeling nauseous"
	MedDRALLT string `json:"meddraLLT,omitempty"`

	// MedDRASOC is the System Organ Class code
	// Example: "10017947" for "Gastrointestinal disorders"
	MedDRASOC string `json:"meddraSOC,omitempty"`

	// MedDRASOCName is the SOC name
	// Example: "Gastrointestinal disorders"
	MedDRASOCName string `json:"meddraSOCName,omitempty"`

	// SNOMEDCode is the SNOMED CT concept ID (from official MedDRA-SNOMED map)
	// Example: "422587007" for "Nausea"
	// Having both MedDRA and SNOMED enables dual-coding for EHR integration
	SNOMEDCode string `json:"snomedCode,omitempty"`

	// IsValidTerm indicates if this is a real clinical term
	// true  = Found in MedDRA (80,000+ official terms)
	// false = Not found (noise like "Meatitis", "n=45", "DVT†")
	IsValidTerm bool `json:"isValidTerm"`

	// Confidence is 1.0 for exact dictionary match, lower for fuzzy match
	Confidence float64 `json:"confidence"`

	// Source indicates where the normalization came from
	// "MEDDRA_OFFICIAL" for dictionary lookup
	// "MEDDRA_FUZZY" for fuzzy matched term
	// "MEDDRA_NOT_FOUND" for noise (IsValidTerm=false)
	Source string `json:"source"`

	// Reason explains the classification (useful for debugging)
	// Example: "Term not in MedDRA dictionary (80,000+ official terms)"
	Reason string `json:"reason,omitempty"`
}

// AdverseEventNormalizer validates and normalizes adverse event terms using MedDRA.
// This is the fix for Issue 2 (FAERS) and Issue 3 (regex ceiling).
type AdverseEventNormalizer interface {
	// Normalize validates a term against MedDRA dictionary and returns MedDRA codes.
	//
	// This REPLACES regex-based noise filtering with dictionary lookup:
	//   - If term is in MedDRA (80,000+ terms): IsValidTerm=true, return PT code
	//   - If term is NOT in MedDRA: IsValidTerm=false (noise)
	//
	// Parameters:
	//   - text: Raw adverse event text from SPL table
	//
	// Returns:
	//   - NormalizedAdverseEvent with MedDRA PT code (FAERS compatible)
	//   - error only for system failures, not for "term not found"
	//
	// Examples:
	//   "Arthritis"         → IsValidTerm=true,  MedDRAPT="10003246"
	//   "Meatitis"          → IsValidTerm=false, Reason="Not in MedDRA"
	//   "n=45"              → IsValidTerm=false, Reason="Statistical notation"
	//   "feeling nauseous"  → IsValidTerm=true,  MedDRAPT="10028813" (→Nausea)
	Normalize(ctx context.Context, text string) (*NormalizedAdverseEvent, error)

	// BatchNormalize normalizes multiple terms efficiently.
	// Useful for processing entire adverse event tables.
	BatchNormalize(ctx context.Context, texts []string) ([]*NormalizedAdverseEvent, error)

	// IsLoaded returns true if MedDRA dictionary is loaded and ready.
	IsLoaded() bool

	// Stats returns dictionary statistics.
	Stats() *MedDRAStats
}

// MedDRAStats provides statistics about the loaded MedDRA dictionary.
type MedDRAStats struct {
	// LLTCount is the number of Lowest Level Terms
	// Expected: ~80,000+
	LLTCount int `json:"lltCount"`

	// PTCount is the number of Preferred Terms
	// Expected: ~24,000+
	PTCount int `json:"ptCount"`

	// SOCCount is the number of System Organ Classes
	// Expected: 27
	SOCCount int `json:"socCount"`

	// SNOMEDMappingCount is the number of MedDRA-SNOMED mappings
	// Expected: ~6,779 (as of 2024)
	SNOMEDMappingCount int `json:"snomedMappingCount"`

	// Version is the MedDRA version (e.g., "26.1")
	Version string `json:"version,omitempty"`

	// LoadedAt is when the dictionary was loaded
	LoadedAt string `json:"loadedAt,omitempty"`
}

// =============================================================================
// COMBINED SERVICE
// =============================================================================

// TerminologyService provides unified access to drug and adverse event normalization.
type TerminologyService interface {
	// Drug normalization (Issue 1 fix)
	DrugNormalizer() DrugNormalizer

	// Adverse event normalization (Issue 2 & 3 fix)
	AdverseEventNormalizer() AdverseEventNormalizer

	// HealthCheck verifies both services are operational
	HealthCheck(ctx context.Context) error

	// Close releases resources
	Close() error
}
