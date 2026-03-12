package elasticsearch

import (
	"time"
)

// ClinicalTerm represents a clinical terminology concept in Elasticsearch
type ClinicalTerm struct {
	TermID           string                 `json:"term_id"`
	ConceptID        string                 `json:"concept_id"`
	Term             string                 `json:"term"`
	PreferredTerm    string                 `json:"preferred_term"`
	Synonyms         []string               `json:"synonyms,omitempty"`
	Definition       string                 `json:"definition,omitempty"`
	TerminologySystem string                `json:"terminology_system"`
	TerminologyVersion string               `json:"terminology_version"`
	SemanticTags     []string               `json:"semantic_tags,omitempty"`
	HierarchyPath    []string               `json:"hierarchy_path,omitempty"`
	ParentConcepts   []string               `json:"parent_concepts,omitempty"`
	ChildConcepts    []string               `json:"child_concepts,omitempty"`
	RelatedConcepts  []string               `json:"related_concepts,omitempty"`
	Status           string                 `json:"status"`
	EffectiveDate    *time.Time             `json:"effective_date,omitempty"`
	ExpiryDate       *time.Time             `json:"expiry_date,omitempty"`
	ClinicalDomain   string                 `json:"clinical_domain,omitempty"`
	ComplexityScore  float32                `json:"complexity_score,omitempty"`
	UsageFrequency   int64                  `json:"usage_frequency,omitempty"`
	LastUpdated      time.Time              `json:"last_updated"`
	SearchMetadata   *SearchMetadata        `json:"search_metadata,omitempty"`
	FHIRMappings     []FHIRMapping          `json:"fhir_mappings,omitempty"`
}

// SearchMetadata contains search optimization data
type SearchMetadata struct {
	BoostFactor     float32 `json:"boost_factor,omitempty"`
	SearchWeight    float32 `json:"search_weight,omitempty"`
	PopularityScore float32 `json:"popularity_score,omitempty"`
}

// FHIRMapping represents FHIR code system mappings
type FHIRMapping struct {
	Code    string `json:"code"`
	System  string `json:"system"`
	Display string `json:"display"`
	Version string `json:"version,omitempty"`
}

// SearchRequest represents a clinical terminology search request
type SearchRequest struct {
	Query           string            `json:"query"`
	Systems         []string          `json:"systems,omitempty"`         // Filter by terminology systems
	Domains         []string          `json:"domains,omitempty"`         // Filter by clinical domains
	SemanticTags    []string          `json:"semantic_tags,omitempty"`   // Filter by semantic tags
	Status          string            `json:"status,omitempty"`          // Filter by status (active, inactive)
	SearchType      SearchType        `json:"search_type"`               // Type of search to perform
	Size            int               `json:"size,omitempty"`            // Number of results to return
	From            int               `json:"from,omitempty"`            // Offset for pagination
	Filters         map[string]string `json:"filters,omitempty"`         // Additional filters
	SortBy          string            `json:"sort_by,omitempty"`         // Sort field
	SortOrder       string            `json:"sort_order,omitempty"`      // Sort order (asc, desc)
	IncludeInactive bool              `json:"include_inactive"`          // Include inactive terms
	ExactMatch      bool              `json:"exact_match"`               // Require exact matching
}

// SearchType defines different types of search strategies
type SearchType string

const (
	SearchTypeStandard     SearchType = "standard"     // Standard text search with synonyms
	SearchTypeExact        SearchType = "exact"        // Exact term matching
	SearchTypeAutocomplete SearchType = "autocomplete" // Autocomplete/suggestion search
	SearchTypePhonetic     SearchType = "phonetic"     // Phonetic/sound-alike search
	SearchTypeFuzzy        SearchType = "fuzzy"        // Fuzzy matching with typo tolerance
	SearchTypeWildcard     SearchType = "wildcard"     // Wildcard pattern matching
)

// SearchResult represents a single search result
type SearchResult struct {
	Term        *ClinicalTerm `json:"term"`
	Score       float64       `json:"score"`
	Highlights  []string      `json:"highlights,omitempty"`
	MatchReason string        `json:"match_reason,omitempty"`
}

// SearchResults represents the complete search response
type SearchResults struct {
	Total       int             `json:"total"`
	Results     []*SearchResult `json:"results"`
	Took        int             `json:"took"`
	TimedOut    bool            `json:"timed_out"`
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
	Suggestions []Suggestion    `json:"suggestions,omitempty"`
}

// Suggestion represents a search suggestion
type Suggestion struct {
	Text    string  `json:"text"`
	Score   float64 `json:"score"`
	Freq    int     `json:"freq,omitempty"`
	Options []SuggestionOption `json:"options,omitempty"`
}

// SuggestionOption represents a suggestion option
type SuggestionOption struct {
	Text   string  `json:"text"`
	Score  float64 `json:"score"`
	Source *ClinicalTerm `json:"_source,omitempty"`
}

// IndexStats represents index statistics
type IndexStats struct {
	IndexName     string    `json:"index_name"`
	DocumentCount int64     `json:"document_count"`
	StoreSize     string    `json:"store_size"`
	LastUpdated   time.Time `json:"last_updated"`
	Health        string    `json:"health"`
	Shards        ShardInfo `json:"shards"`
}

// ShardInfo contains shard information
type ShardInfo struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Failed     int `json:"failed"`
}

// BulkIndexRequest represents a bulk indexing request
type BulkIndexRequest struct {
	IndexName string          `json:"index_name"`
	Terms     []*ClinicalTerm `json:"terms"`
	BatchSize int             `json:"batch_size,omitempty"`
	Refresh   bool            `json:"refresh,omitempty"`
}

// BulkIndexResponse represents a bulk indexing response
type BulkIndexResponse struct {
	Indexed  int           `json:"indexed"`
	Failed   int           `json:"failed"`
	Errors   []BulkError   `json:"errors,omitempty"`
	Took     time.Duration `json:"took"`
}

// BulkError represents an error during bulk indexing
type BulkError struct {
	DocumentID string `json:"document_id"`
	Error      string `json:"error"`
	Status     int    `json:"status"`
}

// TerminologySystem represents a clinical terminology system
type TerminologySystem struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Publisher   string    `json:"publisher,omitempty"`
	Description string    `json:"description,omitempty"`
	URL         string    `json:"url,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
	TermCount   int64     `json:"term_count"`
	Status      string    `json:"status"`
}

// QueryAnalysis represents the analysis of a search query
type QueryAnalysis struct {
	OriginalQuery    string            `json:"original_query"`
	NormalizedQuery  string            `json:"normalized_query"`
	DetectedSystems  []string          `json:"detected_systems,omitempty"`
	DetectedDomains  []string          `json:"detected_domains,omitempty"`
	QueryType        string            `json:"query_type"`
	Confidence       float64           `json:"confidence"`
	SuggestedFilters map[string]string `json:"suggested_filters,omitempty"`
	EstimatedResults int               `json:"estimated_results"`
}