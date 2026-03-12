package models

import (
	"time"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Enhanced models supporting the new schema with partitioning and additional features

// Concept represents a clinical concept from any terminology system
type Concept struct {
	ID               string    `json:"id" db:"id"`
	ConceptUUID      string    `json:"concept_uuid" db:"concept_uuid"`
	SystemID         string    `json:"system_id" db:"system_id"`
	System           string    `json:"system" db:"system"`
	Version          string    `json:"version" db:"version"`
	Code             string    `json:"code" db:"code"`
	PreferredTerm    string    `json:"preferred_term" db:"preferred_term"`
	Definition       string    `json:"definition" db:"definition"`
	Status           string    `json:"status" db:"status"`
	Active           bool      `json:"active" db:"active"`
	Properties       JSONB     `json:"properties" db:"properties"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// JSONB is a custom type for PostgreSQL JSONB columns
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONB", value)
	}
	
	return json.Unmarshal(bytes, j)
}

// UnmarshalJSON implements json.Unmarshaler interface
func (j *JSONB) UnmarshalJSON(data []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*j = JSONB(m)
	return nil
}

// DrugConcept represents specialized drug/medication concepts with clinical attributes
type DrugConcept struct {
	ID         int64  `json:"id" db:"id"`
	RxNormCui  string `json:"rxnorm_cui" db:"rxnorm_cui"`
	
	// Drug identification
	Ingredient string   `json:"ingredient" db:"ingredient"`
	Strength   string   `json:"strength" db:"strength"`
	DoseForm   string   `json:"dose_form" db:"dose_form"`
	BrandNames []string `json:"brand_names" db:"brand_names"`
	
	// Classification
	ATCCodes   []string `json:"atc_codes" db:"atc_codes"`
	DrugClass  string   `json:"drug_class" db:"drug_class"`
	Schedule   string   `json:"schedule" db:"schedule"`
	
	// Clinical attributes
	IsGeneric     *bool `json:"is_generic" db:"is_generic"`
	IsVaccine     bool  `json:"is_vaccine" db:"is_vaccine"`
	IsInsulin     bool  `json:"is_insulin" db:"is_insulin"`
	IsControlled  bool  `json:"is_controlled" db:"is_controlled"`
	
	// Relationships
	HasTradename []string `json:"has_tradename" db:"has_tradename"`
	ConsistsOf   JSONB    `json:"consists_of" db:"consists_of"` // Multi-ingredient drugs
	
	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// LabReference represents laboratory reference ranges for LOINC codes
type LabReference struct {
	ID        int64  `json:"id" db:"id"`
	LoincCode string `json:"loinc_code" db:"loinc_code"`
	TestName  string `json:"test_name" db:"test_name"`
	
	// Reference ranges
	Unit        string   `json:"unit" db:"unit"`
	NormalLow   *float64 `json:"normal_low" db:"normal_low"`
	NormalHigh  *float64 `json:"normal_high" db:"normal_high"`
	CriticalLow *float64 `json:"critical_low" db:"critical_low"`
	CriticalHigh *float64 `json:"critical_high" db:"critical_high"`
	
	// Population specifics
	AgeLow     *int   `json:"age_low" db:"age_low"`
	AgeHigh    *int   `json:"age_high" db:"age_high"`
	Sex        string `json:"sex" db:"sex"` // M, F, U
	Conditions JSONB  `json:"conditions" db:"conditions"`
	
	// Source information
	Source        string    `json:"source" db:"source"`
	EffectiveDate time.Time `json:"effective_date" db:"effective_date"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// ValueSetExpansion represents cached value set expansions
type ValueSetExpansion struct {
	ID               int64     `json:"id" db:"id"`
	ValueSetID       string    `json:"value_set_id" db:"value_set_id"`
	ParamsHash       string    `json:"params_hash" db:"params_hash"`
	ExpansionParams  JSONB     `json:"expansion_params" db:"expansion_params"`
	Total            int       `json:"total" db:"total"`
	OffsetValue      int       `json:"offset_value" db:"offset_value"`
	GeneratedAt      time.Time `json:"generated_at" db:"generated_at"`
	ExpiresAt        *time.Time `json:"expires_at" db:"expires_at"`
}

// ExpansionContains represents individual codes within a value set expansion
type ExpansionContains struct {
	ID          int64  `json:"id" db:"id"`
	ExpansionID int64  `json:"expansion_id" db:"expansion_id"`
	System      string `json:"system" db:"system"`
	Code        string `json:"code" db:"code"`
	Display     string `json:"display" db:"display"`
	Designation JSONB  `json:"designation" db:"designation"`
	Inactive    bool   `json:"inactive" db:"inactive"`
	Abstract    bool   `json:"abstract" db:"abstract"`
}

// SNOMEDExpression represents SNOMED CT compositional expressions
type SNOMEDExpression struct {
	ID               int64    `json:"id" db:"id"`
	ExpressionHash   string   `json:"expression_hash" db:"expression_hash"`
	Expression       string   `json:"expression" db:"expression"`
	NormalForm       string   `json:"normal_form" db:"normal_form"`
	FocusConcepts    []string `json:"focus_concepts" db:"focus_concepts"`
	Refinements      JSONB    `json:"refinements" db:"refinements"`
	ValidationStatus string   `json:"validation_status" db:"validation_status"` // valid, invalid, pending
	ValidationErrors JSONB    `json:"validation_errors" db:"validation_errors"`
	
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TenantOverlay represents multi-tenant customizations
type TenantOverlay struct {
	ID                  int64  `json:"id" db:"id"`
	TenantID           string `json:"tenant_id" db:"tenant_id"`
	Priority           int    `json:"priority" db:"priority"`
	OverlayType        string `json:"overlay_type" db:"overlay_type"` // concept, valueset, map
	TargetSystem       string `json:"target_system" db:"target_system"`
	TargetCode         string `json:"target_code" db:"target_code"`
	OverlayData        JSONB  `json:"overlay_data" db:"overlay_data"`
	ConflictResolution string `json:"conflict_resolution" db:"conflict_resolution"` // override, merge, skip
	Active             bool   `json:"active" db:"active"`
	
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TerminologyAudit represents audit trail for terminology operations
type TerminologyAudit struct {
	ID                   int64     `json:"id" db:"id"`
	Timestamp            time.Time `json:"timestamp" db:"timestamp"`
	UserID               string    `json:"user_id" db:"user_id"`
	TenantID             string    `json:"tenant_id" db:"tenant_id"`
	Operation            string    `json:"operation" db:"operation"`
	ResourceType         string    `json:"resource_type" db:"resource_type"`
	ResourceID           string    `json:"resource_id" db:"resource_id"`
	Parameters           JSONB     `json:"parameters" db:"parameters"`
	ResultCount          *int      `json:"result_count" db:"result_count"`
	DurationMs           *int      `json:"duration_ms" db:"duration_ms"`
	CacheHit             *bool     `json:"cache_hit" db:"cache_hit"`
	LicenseCheckPassed   bool      `json:"license_check_passed" db:"license_check_passed"`
}

// Enhanced search and query models

// EnhancedSearchQuery extends SearchQuery with additional filters and options
type EnhancedSearchQuery struct {
	SearchQuery
	
	// Advanced search options
	FuzzySearch         bool     `json:"fuzzy_search,omitempty"`
	PhoneticSearch      bool     `json:"phonetic_search,omitempty"`
	IncludeInactive     bool     `json:"include_inactive,omitempty"`
	IncludeHierarchy    bool     `json:"include_hierarchy,omitempty"`
	MaxDepth            int      `json:"max_depth,omitempty"`
	PreferredLanguage   string   `json:"preferred_language,omitempty"`
	TenantID            string   `json:"tenant_id,omitempty"`
	
	// Domain-specific filters
	ClinicalDomains     []string `json:"clinical_domains,omitempty"`
	ConceptTypes        []string `json:"concept_types,omitempty"`
	DateRange           *DateRange `json:"date_range,omitempty"`
}

// DateRange represents a date range filter
type DateRange struct {
	From *time.Time `json:"from,omitempty"`
	To   *time.Time `json:"to,omitempty"`
}

// EnhancedSearchResult extends SearchResult with additional metadata
type EnhancedSearchResult struct {
	SearchResult
	
	// Search metadata
	SearchTime          float64                `json:"search_time_ms"`
	CacheHit            bool                   `json:"cache_hit"`
	DidYouMean          string                 `json:"did_you_mean,omitempty"`
	SearchStrategy      string                 `json:"search_strategy"` // exact, fulltext, fuzzy, phonetic
	
	// Faceted results
	Facets              map[string][]Facet     `json:"facets,omitempty"`
	Suggestions         []SearchSuggestion     `json:"suggestions,omitempty"`
}

// Facet represents faceted search results
type Facet struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// SearchSuggestion represents search suggestions
type SearchSuggestion struct {
	Term        string  `json:"term"`
	Score       float64 `json:"score"`
	Type        string  `json:"type"` // spelling, synonym, related
}

// BatchLookupRequest represents a batch lookup request
type BatchLookupRequest struct {
	Requests []LookupRequest `json:"requests"`
	Options  BatchOptions    `json:"options,omitempty"`
}

// LookupRequest represents a single lookup request
type LookupRequest struct {
	System  string `json:"system" binding:"required"`
	Code    string `json:"code" binding:"required"`
	Version string `json:"version,omitempty"`
}

// BatchValidationRequest represents a batch validation request
type BatchValidationRequest struct {
	Requests []ValidationRequest `json:"requests"`
	Options  BatchOptions        `json:"options,omitempty"`
}

// ValidationRequest represents a single validation request
type ValidationRequest struct {
	Code    string `json:"code" binding:"required"`
	System  string `json:"system" binding:"required"`
	Version string `json:"version,omitempty"`
	Display string `json:"display,omitempty"`
}

// BatchOptions provides options for batch operations
type BatchOptions struct {
	IncludeHierarchy    bool   `json:"include_hierarchy,omitempty"`
	IncludeDesignations bool   `json:"include_designations,omitempty"`
	TenantID            string `json:"tenant_id,omitempty"`
	ParallelProcessing  bool   `json:"parallel_processing,omitempty"`
	MaxConcurrency      int    `json:"max_concurrency,omitempty"`
}

// BatchLookupResponse represents a batch lookup response
type BatchLookupResponse struct {
	Results   []LookupResult `json:"results"`
	Metadata  BatchMetadata  `json:"metadata"`
}

// BatchValidationResponse represents a batch validation response
type BatchValidationResponse struct {
	Results  []ValidationResult `json:"results"`
	Metadata BatchMetadata      `json:"metadata"`
}

// BatchMetadata contains metadata about batch operations
type BatchMetadata struct {
	TotalRequests    int     `json:"total_requests"`
	SuccessfulCount  int     `json:"successful_count"`
	FailedCount      int     `json:"failed_count"`
	ProcessingTimeMs float64 `json:"processing_time_ms"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
}

// ConceptHierarchy represents hierarchical concept relationships
type ConceptHierarchy struct {
	System        string `json:"system" db:"system"`
	Code          string `json:"code" db:"code"`
	PreferredTerm string `json:"preferred_term" db:"preferred_term"`
	ParentCodes   []string `json:"parent_codes" db:"parent_codes"`
	Level         int    `json:"level" db:"level"`
	Path          []string `json:"path" db:"path"`
}

// ExpansionParameters represents parameters for value set expansion
type ExpansionParameters struct {
	URL                 string    `json:"url" binding:"required"`
	ValueSetVersion     string    `json:"valueSetVersion,omitempty"`
	Context             string    `json:"context,omitempty"`
	ContextDirection    string    `json:"contextDirection,omitempty"` // incoming, outgoing
	Filter              string    `json:"filter,omitempty"`
	Date                *time.Time `json:"date,omitempty"`
	Offset              int       `json:"offset,omitempty"`
	Count               int       `json:"count,omitempty"`
	IncludeDesignations bool      `json:"includeDesignations,omitempty"`
	IncludeDefinition   bool      `json:"includeDefinition,omitempty"`
	ActiveOnly          bool      `json:"activeOnly,omitempty"`
	ExcludeNested       bool      `json:"excludeNested,omitempty"`
	ExcludeNotForUI     bool      `json:"excludeNotForUI,omitempty"`
	ExcludePostCoordinated bool   `json:"excludePostCoordinated,omitempty"`
	DisplayLanguage     string    `json:"displayLanguage,omitempty"`
}

// ExpandedValueSet represents an expanded value set
type ExpandedValueSet struct {
	URL         string                `json:"url"`
	Version     string                `json:"version,omitempty"`
	Identifier  string                `json:"identifier,omitempty"`
	Timestamp   time.Time             `json:"timestamp"`
	Total       int                   `json:"total,omitempty"`
	Offset      int                   `json:"offset,omitempty"`
	Parameter   []ExpansionParameter  `json:"parameter,omitempty"`
	Contains    []ExpansionContains   `json:"contains,omitempty"`
}

// ExpansionParameter represents expansion parameters used
type ExpansionParameter struct {
	Name        string      `json:"name"`
	ValueString *string     `json:"valueString,omitempty"`
	ValueBoolean *bool      `json:"valueBoolean,omitempty"`
	ValueInteger *int       `json:"valueInteger,omitempty"`
	ValueDecimal *float64   `json:"valueDecimal,omitempty"`
	ValueUri     *string    `json:"valueUri,omitempty"`
	ValueCode    *string    `json:"valueCode,omitempty"`
}

// SNOMEDExpressionValidationResult represents SNOMED expression validation results
type SNOMEDExpressionValidationResult struct {
	Valid            bool                    `json:"valid"`
	Expression       string                  `json:"expression"`
	NormalForm       string                  `json:"normal_form,omitempty"`
	NormalizedHash   string                  `json:"normalized_hash,omitempty"`
	FocusConcepts    []string                `json:"focus_concepts,omitempty"`
	Refinements      []SNOMEDRefinement      `json:"refinements,omitempty"`
	ValidationErrors []ValidationIssue       `json:"validation_errors,omitempty"`
	SemanticCheck    *SNOMEDSemanticCheck    `json:"semantic_check,omitempty"`
}

// SNOMEDRefinement represents a SNOMED expression refinement
type SNOMEDRefinement struct {
	Concept      string `json:"concept"`
	Relationship string `json:"relationship"`
	Value        string `json:"value"`
	ValueType    string `json:"value_type"` // concept, literal
}

// SNOMEDSemanticCheck represents semantic validation results
type SNOMEDSemanticCheck struct {
	Valid               bool              `json:"valid"`
	SemanticConsistency bool              `json:"semantic_consistency"`
	ClinicallyMeaningful bool             `json:"clinically_meaningful"`
	Issues              []string          `json:"issues,omitempty"`
	Suggestions         []string          `json:"suggestions,omitempty"`
}

// HealthCheckResult represents the health status of the service
type HealthCheckResult struct {
	Service   string                    `json:"service"`
	Status    string                    `json:"status"` // healthy, degraded, unhealthy
	Timestamp time.Time                 `json:"timestamp"`
	Version   string                    `json:"version,omitempty"`
	Checks    map[string]ComponentCheck `json:"checks"`
}

// ComponentCheck represents the health status of a service component
type ComponentCheck struct {
	Status      string                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Error       string                 `json:"error,omitempty"`
	LastUpdated time.Time              `json:"last_updated"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
}

// ServiceMetrics represents key performance metrics for the terminology service
type ServiceMetrics struct {
	RequestsPerSecond   float64           `json:"requests_per_second"`
	AverageLatencyMs    float64           `json:"average_latency_ms"`
	P95LatencyMs        float64           `json:"p95_latency_ms"`
	P99LatencyMs        float64           `json:"p99_latency_ms"`
	CacheHitRate        float64           `json:"cache_hit_rate"`
	ErrorRate           float64           `json:"error_rate"`
	ActiveConnections   int               `json:"active_connections"`
	
	// Operation-specific metrics
	LookupMetrics       OperationMetrics  `json:"lookup_metrics"`
	SearchMetrics       OperationMetrics  `json:"search_metrics"`
	ValidationMetrics   OperationMetrics  `json:"validation_metrics"`
	ExpansionMetrics    OperationMetrics  `json:"expansion_metrics"`
}

// OperationMetrics represents metrics for specific operations
type OperationMetrics struct {
	Count            int64   `json:"count"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	P95LatencyMs     float64 `json:"p95_latency_ms"`
	ErrorRate        float64 `json:"error_rate"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
}

// BatchValidationOptions provides options for batch validation operations
type BatchValidationOptions struct {
	StrictValidation    bool   `json:"strict_validation,omitempty"`
	IncludeInactive     bool   `json:"include_inactive,omitempty"`
	TenantID            string `json:"tenant_id,omitempty"`
	ParallelProcessing  bool   `json:"parallel_processing,omitempty"`
	MaxConcurrency      int    `json:"max_concurrency,omitempty"`
}