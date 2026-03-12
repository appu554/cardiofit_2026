package models

import (
	"time"
)

// ============================================================================
// NCTS Refset Models
// ============================================================================
// Data models for SNOMED CT-AU reference set management.
// Supports membership queries, version tracking, and import operations.
// ============================================================================

// Refset represents an NCTS reference set definition
type Refset struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	ModuleID    string    `json:"module_id" db:"module_id"`
	Version     string    `json:"version,omitempty" db:"version"`
	Description string    `json:"description,omitempty" db:"description"`
	Active      bool      `json:"active" db:"active"`
	MemberCount int       `json:"member_count,omitempty" db:"member_count"`
	CreatedAt   time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// RefsetMember represents a concept's membership in a reference set
type RefsetMember struct {
	MemberID              string    `json:"member_id" db:"member_id"`
	RefsetID              string    `json:"refset_id" db:"refset_id"`
	ReferencedComponentID string    `json:"referenced_component_id" db:"referenced_component_id"`
	ModuleID              string    `json:"module_id" db:"module_id"`
	EffectiveTime         time.Time `json:"effective_time" db:"effective_time"`
	Active                bool      `json:"active" db:"active"`
	// Additional fields for enriched responses
	ConceptCode  string `json:"concept_code,omitempty"`
	ConceptLabel string `json:"concept_label,omitempty"`
}

// RefsetLookupResult represents the result of a refset member lookup
type RefsetLookupResult struct {
	Success     bool           `json:"success"`
	RefsetID    string         `json:"refset_id"`
	RefsetName  string         `json:"refset_name"`
	Members     []RefsetMember `json:"members,omitempty"`
	MemberCount int            `json:"member_count"`
	TotalCount  int            `json:"total_count,omitempty"` // For pagination
	Offset      int            `json:"offset,omitempty"`
	Limit       int            `json:"limit,omitempty"`
	QueryTimeMs float64        `json:"query_time_ms"`
}

// ConceptRefsets represents the refsets a concept belongs to
type ConceptRefsets struct {
	Success     bool      `json:"success"`
	ConceptID   string    `json:"concept_id"`
	ConceptCode string    `json:"concept_code,omitempty"`
	Refsets     []Refset  `json:"refsets"`
	QueryTimeMs float64   `json:"query_time_ms"`
}

// RefsetMembershipCheck represents a membership check result
type RefsetMembershipCheck struct {
	Success     bool    `json:"success"`
	ConceptCode string  `json:"concept_code"`
	RefsetID    string  `json:"refset_id"`
	IsMember    bool    `json:"is_member"`
	MemberID    string  `json:"member_id,omitempty"`
	QueryTimeMs float64 `json:"query_time_ms"`
}

// ============================================================================
// Import Metadata Models
// ============================================================================

// ImportMetadata tracks refset import versions and history
type ImportMetadata struct {
	Type              string    `json:"type" db:"type"`
	Version           string    `json:"version" db:"version"`
	ImportedAt        time.Time `json:"imported_at" db:"imported_at"`
	ImportedBy        string    `json:"imported_by,omitempty" db:"imported_by"`
	FileCount         int       `json:"file_count,omitempty" db:"file_count"`
	RelationshipCount int       `json:"relationship_count,omitempty" db:"relationship_count"`
	Neo4jURI          string    `json:"neo4j_uri,omitempty" db:"neo4j_uri"`
}

// ImportStatusResponse represents the API response for import status
type ImportStatusResponse struct {
	Success           bool               `json:"success"`
	CurrentVersion    string             `json:"current_version"`
	ImportedAt        time.Time          `json:"imported_at"`
	FileCount         int                `json:"file_count"`
	RelationshipCount int                `json:"relationship_count"`
	RefsetTypes       map[string]int     `json:"refset_types,omitempty"`
	History           []ImportMetadata   `json:"history,omitempty"`
	QueryTimeMs       float64            `json:"query_time_ms"`
}

// ============================================================================
// Query Options
// ============================================================================

// RefsetQueryOptions configures refset member queries
type RefsetQueryOptions struct {
	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Filters
	ActiveOnly   bool     `json:"active_only,omitempty"`
	ModuleIDs    []string `json:"module_ids,omitempty"`

	// Sorting
	SortBy    string `json:"sort_by,omitempty"`    // "code", "label", "effective_time"
	SortOrder string `json:"sort_order,omitempty"` // "asc", "desc"

	// Include options
	IncludeInactive bool `json:"include_inactive,omitempty"`
	IncludeCounts   bool `json:"include_counts,omitempty"`
}

// DefaultRefsetQueryOptions returns sensible default query options
func DefaultRefsetQueryOptions() *RefsetQueryOptions {
	return &RefsetQueryOptions{
		Limit:        100,
		Offset:       0,
		ActiveOnly:   true,
		SortBy:       "label",
		SortOrder:    "asc",
		IncludeCounts: true,
	}
}

// ============================================================================
// RF2 Parsing Models
// ============================================================================

// RF2RefsetRow represents a row from an RF2 Simple Refset file
// Format: id | effectiveTime | active | moduleId | refsetId | referencedComponentId
type RF2RefsetRow struct {
	ID                    string `json:"id"`
	EffectiveTime         string `json:"effective_time"`
	Active                string `json:"active"`
	ModuleID              string `json:"module_id"`
	RefsetID              string `json:"refset_id"`
	ReferencedComponentID string `json:"referenced_component_id"`
}

// IsActive returns true if the RF2 row represents an active membership
func (r *RF2RefsetRow) IsActive() bool {
	return r.Active == "1"
}

// ParseEffectiveTime parses the YYYYMMDD format into a time.Time
func (r *RF2RefsetRow) ParseEffectiveTime() (time.Time, error) {
	return time.Parse("20060102", r.EffectiveTime)
}

// RF2AssociationRow represents a row from an RF2 Association Refset file
// Format: id | effectiveTime | active | moduleId | refsetId | referencedComponentId | targetComponentId
type RF2AssociationRow struct {
	RF2RefsetRow
	TargetComponentID string `json:"target_component_id"`
}

// RF2LanguageRow represents a row from an RF2 Language Refset file
// Format: id | effectiveTime | active | moduleId | refsetId | referencedComponentId | acceptabilityId
type RF2LanguageRow struct {
	RF2RefsetRow
	AcceptabilityID string `json:"acceptability_id"`
}

// ============================================================================
// Loader Statistics
// ============================================================================

// RefsetLoaderStats tracks import statistics
type RefsetLoaderStats struct {
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	Duration          string    `json:"duration"`
	FilesProcessed    int       `json:"files_processed"`
	RowsRead          int       `json:"rows_read"`
	RowsImported      int       `json:"rows_imported"`
	RowsSkipped       int       `json:"rows_skipped"`
	RefsetNodesCreated int      `json:"refset_nodes_created"`
	RelationshipsCreated int    `json:"relationships_created"`
	Errors            []string  `json:"errors,omitempty"`
}

// ============================================================================
// Module ID Constants
// ============================================================================

const (
	// ModuleSnomedAU is the SNOMED CT-AU module ID
	ModuleSnomedAU = "32506021000036107"

	// ModuleAMT is the Australian Medicines Terminology module ID
	ModuleAMT = "900062011000036103"

	// ModuleSnomedINT is the SNOMED International module ID
	ModuleSnomedINT = "900000000000207008"

	// RefsetTypeSimple indicates a simple reference set
	RefsetTypeSimple = "simple"

	// RefsetTypeAssociation indicates an association reference set
	RefsetTypeAssociation = "association"

	// RefsetTypeLanguage indicates a language reference set
	RefsetTypeLanguage = "language"

	// RefsetTypeMap indicates a map reference set
	RefsetTypeMap = "map"
)

// GetModuleName returns a human-readable name for a module ID
func GetModuleName(moduleID string) string {
	switch moduleID {
	case ModuleSnomedAU:
		return "SNOMED CT-AU"
	case ModuleAMT:
		return "Australian Medicines Terminology"
	case ModuleSnomedINT:
		return "SNOMED International"
	default:
		return "Unknown Module"
	}
}

// ============================================================================
// List Response Models
// ============================================================================

// RefsetListResponse represents the API response for listing refsets
type RefsetListResponse struct {
	Success     bool      `json:"success"`
	Refsets     []Refset  `json:"refsets"`
	TotalCount  int       `json:"total_count"`
	QueryTimeMs float64   `json:"query_time_ms"`
}

// RefsetDetailResponse represents the API response for a single refset
type RefsetDetailResponse struct {
	Success     bool    `json:"success"`
	Refset      *Refset `json:"refset"`
	MemberCount int     `json:"member_count"`
	QueryTimeMs float64 `json:"query_time_ms"`
}
