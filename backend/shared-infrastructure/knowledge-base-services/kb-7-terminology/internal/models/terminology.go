package models

import (
	"time"
)


// TerminologySystem represents a clinical terminology system (SNOMED CT, ICD-10, RxNorm, etc.)
type TerminologySystem struct {
	ID          string    `json:"id" db:"id"`
	SystemURI   string    `json:"system_uri" db:"system_uri"`
	SystemName  string    `json:"system_name" db:"system_name"`
	Version     string    `json:"version" db:"version"`
	Description string    `json:"description" db:"description"`
	Publisher   string    `json:"publisher" db:"publisher"`
	Status      string    `json:"status" db:"status"` // active, draft, retired
	
	// Metadata
	Metadata JSONB `json:"metadata" db:"metadata"`
	
	// Regional support
	SupportedRegions []string `json:"supported_regions" db:"supported_regions"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TerminologyConcept represents a single concept within a terminology system
type TerminologyConcept struct {
	ID        string `json:"id" db:"id"`
	SystemID  string `json:"system_id" db:"system_id"`
	Code      string `json:"code" db:"code"`
	Display   string `json:"display" db:"display"`
	Definition string `json:"definition" db:"definition"`
	Status    string `json:"status" db:"status"` // active, inactive, entered-in-error
	
	// Concept hierarchy
	ParentCodes []string `json:"parent_codes" db:"parent_codes"`
	ChildCodes  []string `json:"child_codes" db:"child_codes"`
	
	// Additional properties
	Properties JSONB `json:"properties" db:"properties"`
	Designations JSONB `json:"designations" db:"designations"` // Alternative terms, translations
	
	// Clinical relevance
	ClinicalDomain string `json:"clinical_domain" db:"clinical_domain"`
	Specialty      string `json:"specialty" db:"specialty"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ConceptMapping represents mappings between different terminology systems
type ConceptMapping struct {
	ID           string `json:"id" db:"id"`
	
	// Source concept
	SourceSystemID string `json:"source_system_id" db:"source_system_id"`
	SourceCode     string `json:"source_code" db:"source_code"`
	
	// Target concept
	TargetSystemID string `json:"target_system_id" db:"target_system_id"`
	TargetCode     string `json:"target_code" db:"target_code"`
	
	// Mapping metadata
	Equivalence    string `json:"equivalence" db:"equivalence"` // equivalent, equal, wider, subsumes, narrower, specializes, inexact, unmatched, disjoint
	MappingType    string `json:"mapping_type" db:"mapping_type"`
	Confidence     float64 `json:"confidence" db:"confidence"`
	
	// Additional mapping info
	Comment       string `json:"comment" db:"comment"`
	MappedBy      string `json:"mapped_by" db:"mapped_by"`
	Evidence      JSONB  `json:"evidence" db:"evidence"`
	
	// Quality assurance
	Verified      bool      `json:"verified" db:"verified"`
	VerifiedBy    string    `json:"verified_by" db:"verified_by"`
	VerifiedAt    *time.Time `json:"verified_at" db:"verified_at"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ValueSet represents a set of concepts for a specific use case
type ValueSet struct {
	ID          string `json:"id" db:"id"`
	URL         string `json:"url" db:"url"`
	Version     string `json:"version" db:"version"`
	Name        string `json:"name" db:"name"`
	Title       string `json:"title" db:"title"`
	Description string `json:"description" db:"description"`
	Status      string `json:"status" db:"status"` // draft, active, retired, unknown
	
	// Publisher information
	Publisher string `json:"publisher" db:"publisher"`
	Contact   JSONB  `json:"contact" db:"contact"`
	
	// Use case context
	UseContext    JSONB    `json:"use_context" db:"use_context"`
	Purpose       string   `json:"purpose" db:"purpose"`
	ClinicalDomain string  `json:"clinical_domain" db:"clinical_domain"`
	
	// Value set composition
	Compose JSONB `json:"compose" db:"compose"` // Include/exclude rules
	Expansion JSONB `json:"expansion" db:"expansion"` // Computed expansion
	
	// Regional variations
	SupportedRegions []string `json:"supported_regions" db:"supported_regions"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	ExpiredAt *time.Time `json:"expired_at" db:"expired_at"`
}

// ConceptDesignation represents alternative terms or translations for a concept
type ConceptDesignation struct {
	Language string `json:"language"`
	Use      JSONB  `json:"use"`
	Value    string `json:"value"`
}

// SearchQuery represents a terminology search query
type SearchQuery struct {
	Query        string            `json:"query"`
	SystemURI    string            `json:"system_uri,omitempty"`
	Count        int               `json:"count,omitempty"`
	Offset       int               `json:"offset,omitempty"`
	Filter       map[string]string `json:"filter,omitempty"`
	IncludeDesignations bool       `json:"include_designations,omitempty"`
}

// SearchResult represents the result of a terminology search
type SearchResult struct {
	Total    int64                 `json:"total"`
	Concepts []TerminologyConcept  `json:"concepts"`
}

// ValidationResult represents the result of terminology validation
type ValidationResult struct {
	Valid        bool                    `json:"valid"`
	Code         string                  `json:"code"`
	System       string                  `json:"system"`
	Display      string                  `json:"display,omitempty"`
	Message      string                  `json:"message,omitempty"`
	Severity     string                  `json:"severity"` // error, warning, information
	Issues       []ValidationIssue       `json:"issues,omitempty"`
}

// ValidationIssue represents a specific validation issue
type ValidationIssue struct {
	Severity    string `json:"severity"`
	Code        string `json:"code"`
	Details     string `json:"details"`
	Location    string `json:"location,omitempty"`
}

// LookupResult represents the result of a concept lookup
type LookupResult struct {
	Concept     TerminologyConcept `json:"concept"`
	Properties  JSONB              `json:"properties,omitempty"`
	Designations []ConceptDesignation `json:"designations,omitempty"`
	Parents     []TerminologyConcept `json:"parents,omitempty"`
	Children    []TerminologyConcept `json:"children,omitempty"`
}