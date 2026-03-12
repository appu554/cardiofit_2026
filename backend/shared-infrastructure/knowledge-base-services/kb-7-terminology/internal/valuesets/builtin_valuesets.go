// Package valuesets provides FHIR R4 value set type definitions.
// Note: Value sets are now stored in PostgreSQL. This package provides
// type definitions for API compatibility only.
package valuesets

import "time"

// ValueSetDefinition contains the metadata for a value set
type ValueSetDefinition struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Version     string    `json:"version"`
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Status      string    `json:"status"`
	Publisher   string    `json:"publisher"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BuiltinValueSet represents a pre-defined FHIR R4 value set
// Note: This struct is kept for API compatibility. Value sets are now in PostgreSQL.
type BuiltinValueSet struct {
	Definition ValueSetDefinition `json:"definition"`
	Concepts   []ValueSetConcept  `json:"concepts"`
}

// ValueSetConcept represents a concept within a value set
type ValueSetConcept struct {
	System      string `json:"system"`
	Version     string `json:"version,omitempty"`
	Code        string `json:"code"`
	Display     string `json:"display"`
	Definition  string `json:"definition,omitempty"`
	Designation string `json:"designation,omitempty"`
}

// ToModelValueSet converts ValueSetDefinition to a format compatible with models.ValueSet
// Returns a map that can be used for API responses when builtin value sets are accessed
type ModelValueSet struct {
	ID               string     `json:"id"`
	URL              string     `json:"url"`
	Version          string     `json:"version"`
	Name             string     `json:"name"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	Status           string     `json:"status"`
	Publisher        string     `json:"publisher"`
	Contact          interface{} `json:"contact,omitempty"`
	UseContext       interface{} `json:"use_context,omitempty"`
	Purpose          string     `json:"purpose,omitempty"`
	ClinicalDomain   string     `json:"clinical_domain,omitempty"`
	Compose          interface{} `json:"compose,omitempty"`
	Expansion        interface{} `json:"expansion,omitempty"`
	SupportedRegions []string   `json:"supported_regions,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ExpiredAt        *time.Time `json:"expired_at,omitempty"`
}

// ToModelValueSet converts a BuiltinValueSet to ModelValueSet for API compatibility
func (b *BuiltinValueSet) ToModelValueSet() *ModelValueSet {
	return &ModelValueSet{
		ID:          b.Definition.ID,
		URL:         b.Definition.URL,
		Version:     b.Definition.Version,
		Name:        b.Definition.Name,
		Title:       b.Definition.Title,
		Description: b.Definition.Description,
		Status:      b.Definition.Status,
		Publisher:   b.Definition.Publisher,
		CreatedAt:   b.Definition.CreatedAt,
		UpdatedAt:   b.Definition.UpdatedAt,
	}
}

// GetBuiltinValueSets returns an empty slice.
// Value sets are now managed through PostgreSQL database.
// This function is kept for API compatibility only.
func GetBuiltinValueSets() []*BuiltinValueSet {
	// All value sets are now in PostgreSQL - no hardcoded builtins
	return []*BuiltinValueSet{}
}
