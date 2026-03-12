// Package evidence provides the Evidence Router for unified data ingestion.
// The Evidence Router is the single entry point for all external data sources,
// routing evidence units to appropriate processing streams.
//
// DESIGN PRINCIPLE: "One Factory, Many Assembly Lines"
// Instead of N pipelines for N KBs, we have one router with pluggable streams.
package evidence

import (
	"time"
)

// =============================================================================
// SOURCE TYPES
// =============================================================================

// SourceType categorizes the origin of clinical evidence
type SourceType string

const (
	// SourceTypeSPL is FDA Structured Product Labeling (narrative text requiring LLM)
	SourceTypeSPL SourceType = "SPL"

	// SourceTypeAPI is structured API data (DrugBank, RxNav, MED-RT)
	SourceTypeAPI SourceType = "API"

	// SourceTypeCSV is government datasets (CMS PUF, NHANES)
	SourceTypeCSV SourceType = "CSV"

	// SourceTypeGuideline is clinical guidelines (future expansion)
	SourceTypeGuideline SourceType = "GUIDELINE"

	// SourceTypePDF is regulatory PDFs (CDSCO, TGA - future expansion)
	SourceTypePDF SourceType = "PDF"

	// SourceTypeFHIR is FHIR resources
	SourceTypeFHIR SourceType = "FHIR"
)

// =============================================================================
// CLINICAL DOMAINS
// =============================================================================

// ClinicalDomain identifies the clinical area of the evidence
type ClinicalDomain string

const (
	DomainRenal       ClinicalDomain = "renal"
	DomainHepatic     ClinicalDomain = "hepatic"
	DomainCardiac     ClinicalDomain = "cardiac"
	DomainSafety      ClinicalDomain = "safety"
	DomainInteraction ClinicalDomain = "interaction"
	DomainFormulary   ClinicalDomain = "formulary"
	DomainLab         ClinicalDomain = "lab"
	DomainReproductive ClinicalDomain = "reproductive"
	DomainGeriatric   ClinicalDomain = "geriatric"
	DomainPediatric   ClinicalDomain = "pediatric"
)

// =============================================================================
// EVIDENCE UNIT
// =============================================================================

// EvidenceUnit represents a single unit of clinical evidence before extraction.
// Each unit is tagged with metadata for routing to the appropriate processing stream.
type EvidenceUnit struct {
	// ─────────────────────────────────────────────────────────────────────────
	// IDENTITY
	// ─────────────────────────────────────────────────────────────────────────

	// EvidenceID is the unique identifier for this evidence unit
	EvidenceID string `json:"evidenceId"`

	// SourceType categorizes the evidence origin (SPL, API, CSV, etc.)
	SourceType SourceType `json:"sourceType"`

	// SourceVersion is the version identifier of the source (e.g., SPL version date)
	SourceVersion string `json:"sourceVersion"`

	// ─────────────────────────────────────────────────────────────────────────
	// ROUTING METADATA
	// ─────────────────────────────────────────────────────────────────────────

	// ClinicalDomains identifies which clinical domains this evidence applies to
	ClinicalDomains []ClinicalDomain `json:"clinicalDomains"`

	// KBTargets specifies which KBs should process this evidence
	KBTargets []string `json:"kbTargets"`

	// Priority indicates processing priority (1=highest, 10=lowest)
	Priority int `json:"priority"`

	// ─────────────────────────────────────────────────────────────────────────
	// DRUG REFERENCE
	// ─────────────────────────────────────────────────────────────────────────

	// RxCUI is the RxNorm concept ID (if known)
	RxCUI string `json:"rxcui,omitempty"`

	// DrugName is the drug name (for display/logging)
	DrugName string `json:"drugName,omitempty"`

	// NDC is the National Drug Code (if applicable)
	NDC string `json:"ndc,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// CONTENT
	// ─────────────────────────────────────────────────────────────────────────

	// RawContent is the original bytes from the source
	RawContent []byte `json:"rawContent"`

	// ContentType is the MIME type (application/xml, application/json, text/csv, etc.)
	ContentType string `json:"contentType"`

	// ParsedContent is the structured representation (if pre-parsed)
	ParsedContent interface{} `json:"parsedContent,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// QUALITY SIGNALS
	// ─────────────────────────────────────────────────────────────────────────

	// ConfidenceFloor is the minimum acceptable confidence for extracted facts
	ConfidenceFloor float64 `json:"confidenceFloor"`

	// QualityScore is a pre-extraction quality assessment (0.0-1.0)
	QualityScore float64 `json:"qualityScore,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// PROVENANCE
	// ─────────────────────────────────────────────────────────────────────────

	// FetchedAt is when the evidence was retrieved
	FetchedAt time.Time `json:"fetchedAt"`

	// SourceURL is the original URL of the evidence
	SourceURL string `json:"sourceUrl"`

	// Checksum is a hash for deduplication
	Checksum string `json:"checksum"`

	// SourceMetadata contains source-specific metadata
	SourceMetadata map[string]string `json:"sourceMetadata,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// REGULATORY
	// ─────────────────────────────────────────────────────────────────────────

	// Jurisdiction is the regulatory jurisdiction (US, AU, IN, etc.)
	Jurisdiction string `json:"jurisdiction"`

	// RegulatoryBody is the regulatory authority (FDA, TGA, CDSCO, etc.)
	RegulatoryBody string `json:"regulatoryBody"`
}

// =============================================================================
// SPL-SPECIFIC STRUCTURES
// =============================================================================

// SPLSection represents a section of an FDA SPL document
type SPLSection struct {
	// SectionCode is the LOINC code for the section
	SectionCode string `json:"sectionCode"`

	// SectionName is the human-readable name
	SectionName string `json:"sectionName"`

	// Content is the section text
	Content string `json:"content"`

	// Subsections contains nested sections
	Subsections []SPLSection `json:"subsections,omitempty"`
}

// SPLDocument represents a parsed FDA SPL document
type SPLDocument struct {
	// SetID is the SPL Set ID
	SetID string `json:"setId"`

	// VersionNumber is the SPL version
	VersionNumber string `json:"versionNumber"`

	// EffectiveTime is when this version became effective
	EffectiveTime time.Time `json:"effectiveTime"`

	// Sections contains the document sections
	Sections []SPLSection `json:"sections"`

	// DrugName is the proprietary name
	DrugName string `json:"drugName"`

	// GenericName is the non-proprietary name
	GenericName string `json:"genericName"`

	// NDCs contains the National Drug Codes
	NDCs []string `json:"ndcs"`
}

// =============================================================================
// PROCESSING RESULT
// =============================================================================

// ProcessingStatus indicates the outcome of evidence processing
type ProcessingStatus string

const (
	StatusProcessed   ProcessingStatus = "PROCESSED"
	StatusSkipped     ProcessingStatus = "SKIPPED"
	StatusFailed      ProcessingStatus = "FAILED"
	StatusNoExtraction ProcessingStatus = "NO_EXTRACTION"
)

// ProcessingResult captures the outcome of processing an evidence unit
type ProcessingResult struct {
	// EvidenceID is the ID of the processed evidence
	EvidenceID string `json:"evidenceId"`

	// Status is the processing outcome
	Status ProcessingStatus `json:"status"`

	// FactsExtracted is the number of facts extracted
	FactsExtracted int `json:"factsExtracted"`

	// FactIDs lists the IDs of extracted facts
	FactIDs []string `json:"factIds,omitempty"`

	// ProcessingTimeMs is the processing duration
	ProcessingTimeMs int64 `json:"processingTimeMs"`

	// StreamUsed identifies which processing stream handled this
	StreamUsed string `json:"streamUsed"`

	// Error contains any error message
	Error string `json:"error,omitempty"`

	// Warnings contains non-fatal warnings
	Warnings []string `json:"warnings,omitempty"`
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// NewEvidenceUnit creates a new evidence unit with defaults
func NewEvidenceUnit(sourceType SourceType, sourceURL string) *EvidenceUnit {
	return &EvidenceUnit{
		SourceType:      sourceType,
		SourceURL:       sourceURL,
		FetchedAt:       time.Now(),
		ConfidenceFloor: 0.65, // Default to medium confidence threshold
		Priority:        5,    // Default to medium priority
		ClinicalDomains: []ClinicalDomain{},
		KBTargets:       []string{},
		SourceMetadata:  make(map[string]string),
	}
}

// AddClinicalDomain adds a domain to the evidence unit
func (e *EvidenceUnit) AddClinicalDomain(domain ClinicalDomain) {
	for _, d := range e.ClinicalDomains {
		if d == domain {
			return // Already exists
		}
	}
	e.ClinicalDomains = append(e.ClinicalDomains, domain)
}

// AddKBTarget adds a KB target to the evidence unit
func (e *EvidenceUnit) AddKBTarget(kb string) {
	for _, k := range e.KBTargets {
		if k == kb {
			return // Already exists
		}
	}
	e.KBTargets = append(e.KBTargets, kb)
}

// HasDomain checks if the evidence applies to a clinical domain
func (e *EvidenceUnit) HasDomain(domain ClinicalDomain) bool {
	for _, d := range e.ClinicalDomains {
		if d == domain {
			return true
		}
	}
	return false
}

// HasKBTarget checks if a KB is targeted by this evidence
func (e *EvidenceUnit) HasKBTarget(kb string) bool {
	for _, k := range e.KBTargets {
		if k == kb {
			return true
		}
	}
	return false
}
