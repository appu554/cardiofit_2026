package search

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"kb-7-terminology/internal/elasticsearch"
	"kb-7-terminology/internal/metrics"

	"go.uber.org/zap"
)

// SearchEngine provides advanced clinical terminology search capabilities
type SearchEngine struct {
	esIntegration *elasticsearch.Integration
	searchService *elasticsearch.SearchService
	logger        *zap.Logger
	metrics       *metrics.Collector
	config        *SearchConfig
}

// SearchConfig holds configuration for the search engine
type SearchConfig struct {
	DefaultPageSize        int                    `json:"default_page_size"`
	MaxPageSize           int                    `json:"max_page_size"`
	EnableFacetedSearch   bool                   `json:"enable_faceted_search"`
	EnableAutoComplete    bool                   `json:"enable_autocomplete"`
	EnableSpellCorrection bool                   `json:"enable_spell_correction"`
	SearchTimeout         time.Duration          `json:"search_timeout"`
	CacheTTL              time.Duration          `json:"cache_ttl"`
	BoostFactors          map[string]float64     `json:"boost_factors"`
	DefaultFilters        map[string]string      `json:"default_filters"`
	EnableMetrics         bool                   `json:"enable_metrics"`
	QueryAnalysisEnabled  bool                   `json:"query_analysis_enabled"`
}

// ClinicalSearchRequest represents a comprehensive search request
type ClinicalSearchRequest struct {
	// Query parameters
	Query           string                    `json:"query"`
	SearchMode      ClinicalSearchMode        `json:"search_mode"`
	QueryType       QueryType                 `json:"query_type"`

	// Filters
	Systems         []string                  `json:"systems,omitempty"`
	Domains         []string                  `json:"domains,omitempty"`
	SemanticTags    []string                  `json:"semantic_tags,omitempty"`
	Status          string                    `json:"status,omitempty"`
	Languages       []string                  `json:"languages,omitempty"`
	DateRange       *DateRangeFilter          `json:"date_range,omitempty"`
	CustomFilters   map[string]interface{}    `json:"custom_filters,omitempty"`

	// Search options
	IncludeInactive bool                      `json:"include_inactive"`
	ExactMatch      bool                      `json:"exact_match"`
	FuzzyThreshold  float64                   `json:"fuzzy_threshold"`
	BoostRecent     bool                      `json:"boost_recent"`

	// Pagination and sorting
	Page            int                       `json:"page"`
	PageSize        int                       `json:"page_size"`
	SortBy          string                    `json:"sort_by,omitempty"`
	SortOrder       string                    `json:"sort_order,omitempty"`

	// Response options
	IncludeHighlights    bool                 `json:"include_highlights"`
	IncludeFacets        bool                 `json:"include_facets"`
	IncludeSpellCheck    bool                 `json:"include_spell_check"`
	IncludeRelated       bool                 `json:"include_related"`
	IncludeDefinitions   bool                 `json:"include_definitions"`
	FieldsToReturn       []string             `json:"fields_to_return,omitempty"`

	// Context and preferences
	UserContext     *UserSearchContext        `json:"user_context,omitempty"`
	SearchIntent    SearchIntent              `json:"search_intent,omitempty"`
	PreferredSources []string                 `json:"preferred_sources,omitempty"`
}

// ClinicalSearchMode defines different search strategies
type ClinicalSearchMode string

const (
	SearchModeStandard    ClinicalSearchMode = "standard"     // Standard clinical search
	SearchModeExact       ClinicalSearchMode = "exact"        // Exact term matching
	SearchModeAutocomplete ClinicalSearchMode = "autocomplete" // Autocomplete suggestions
	SearchModePhonetic    ClinicalSearchMode = "phonetic"     // Sound-alike matching
	SearchModeFuzzy       ClinicalSearchMode = "fuzzy"        // Fuzzy matching with typos
	SearchModeWildcard    ClinicalSearchMode = "wildcard"     // Pattern matching
	SearchModeSemantic    ClinicalSearchMode = "semantic"     // Semantic similarity
	SearchModeHybrid      ClinicalSearchMode = "hybrid"       // Multiple strategies combined
)

// QueryType defines the nature of the query
type QueryType string

const (
	QueryTypeGeneral      QueryType = "general"       // General terminology search
	QueryTypeDiagnostic   QueryType = "diagnostic"    // Diagnostic codes/terms
	QueryTypeProcedural   QueryType = "procedural"    // Procedure codes/terms
	QueryTypeMedication   QueryType = "medication"    // Drug/medication terms
	QueryTypeLaboratory   QueryType = "laboratory"    // Lab tests/results
	QueryTypeAnatomy      QueryType = "anatomy"       // Anatomical terms
	QueryTypeSymptom      QueryType = "symptom"       // Symptoms and findings
)

// SearchIntent represents the user's search intent
type SearchIntent string

const (
	IntentLookup     SearchIntent = "lookup"      // Looking up specific term
	IntentExplore    SearchIntent = "explore"     // Exploring related concepts
	IntentValidate   SearchIntent = "validate"    // Validating term existence
	IntentTranslate  SearchIntent = "translate"   // Cross-terminology mapping
	IntentBrowse     SearchIntent = "browse"      // Browsing hierarchy
)

// DateRangeFilter represents a date range filter
type DateRangeFilter struct {
	From   *time.Time `json:"from,omitempty"`
	To     *time.Time `json:"to,omitempty"`
	Field  string     `json:"field"` // effective_date, last_updated, etc.
}

// UserSearchContext provides context about the user and session
type UserSearchContext struct {
	UserID          string            `json:"user_id,omitempty"`
	SessionID       string            `json:"session_id,omitempty"`
	Role            string            `json:"role,omitempty"`
	Specialty       string            `json:"specialty,omitempty"`
	PreferredLang   string            `json:"preferred_lang,omitempty"`
	SearchHistory   []string          `json:"search_history,omitempty"`
	Preferences     map[string]string `json:"preferences,omitempty"`
}

// ClinicalSearchResponse represents the search response
type ClinicalSearchResponse struct {
	// Response metadata
	SearchID        string            `json:"search_id"`
	Query           string            `json:"query"`
	ProcessedQuery  string            `json:"processed_query,omitempty"`
	SearchMode      ClinicalSearchMode `json:"search_mode"`
	Timestamp       time.Time         `json:"timestamp"`

	// Results
	Results         []*ClinicalSearchResult `json:"results"`
	TotalCount      int64                  `json:"total_count"`
	ReturnedCount   int                    `json:"returned_count"`
	Page            int                    `json:"page"`
	PageSize        int                    `json:"page_size"`
	HasNextPage     bool                   `json:"has_next_page"`

	// Search enhancements
	Highlights      map[string][]string    `json:"highlights,omitempty"`
	Facets          *SearchFacets          `json:"facets,omitempty"`
	SpellCheck      *SpellCheckSuggestion  `json:"spell_check,omitempty"`
	RelatedTerms    []*RelatedTerm         `json:"related_terms,omitempty"`
	QueryAnalysis   *QueryAnalysisResult   `json:"query_analysis,omitempty"`

	// Performance metrics
	SearchTime      time.Duration          `json:"search_time"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	Explanations    []string               `json:"explanations,omitempty"`

	// Recommendations
	Suggestions     []*SearchSuggestion    `json:"suggestions,omitempty"`
	DidYouMean      string                 `json:"did_you_mean,omitempty"`
	AlternativeQueries []string            `json:"alternative_queries,omitempty"`
}

// ClinicalSearchResult represents a single search result
type ClinicalSearchResult struct {
	// Core term data
	TermID           string                 `json:"term_id"`
	ConceptID        string                 `json:"concept_id"`
	Term             string                 `json:"term"`
	PreferredTerm    string                 `json:"preferred_term"`
	Definition       string                 `json:"definition,omitempty"`
	System           string                 `json:"system"`
	Version          string                 `json:"version,omitempty"`
	Status           string                 `json:"status"`

	// Search metadata
	Score            float64                `json:"score"`
	Rank             int                    `json:"rank"`
	MatchType        string                 `json:"match_type"`
	MatchedFields    []string               `json:"matched_fields"`
	Confidence       float64                `json:"confidence"`

	// Additional data
	Synonyms         []string               `json:"synonyms,omitempty"`
	SemanticTags     []string               `json:"semantic_tags,omitempty"`
	ClinicalDomain   string                 `json:"clinical_domain,omitempty"`
	Hierarchy        *TermHierarchy         `json:"hierarchy,omitempty"`
	CrossMappings    []*CrossMapping        `json:"cross_mappings,omitempty"`
	UsageStats       *TermUsageStats        `json:"usage_stats,omitempty"`

	// Highlighting and explanations
	Highlights       map[string][]string    `json:"highlights,omitempty"`
	ScoreExplanation string                 `json:"score_explanation,omitempty"`
	ContextualInfo   string                 `json:"contextual_info,omitempty"`
}

// SearchFacets contains faceted search results
type SearchFacets struct {
	Systems        []*FacetValue `json:"systems,omitempty"`
	Domains        []*FacetValue `json:"domains,omitempty"`
	SemanticTags   []*FacetValue `json:"semantic_tags,omitempty"`
	Status         []*FacetValue `json:"status,omitempty"`
	Languages      []*FacetValue `json:"languages,omitempty"`
	CustomFacets   map[string][]*FacetValue `json:"custom_facets,omitempty"`
}

// FacetValue represents a facet value with count
type FacetValue struct {
	Value string `json:"value"`
	Label string `json:"label,omitempty"`
	Count int64  `json:"count"`
}

// SpellCheckSuggestion contains spell check results
type SpellCheckSuggestion struct {
	OriginalQuery string                    `json:"original_query"`
	Corrections   []*SpellCorrection        `json:"corrections,omitempty"`
	HasSuggestions bool                     `json:"has_suggestions"`
	Confidence    float64                   `json:"confidence"`
}

// SpellCorrection represents a spelling correction
type SpellCorrection struct {
	Original    string  `json:"original"`
	Suggestion  string  `json:"suggestion"`
	Confidence  float64 `json:"confidence"`
	Frequency   int64   `json:"frequency,omitempty"`
}

// RelatedTerm represents a related clinical term
type RelatedTerm struct {
	TermID       string  `json:"term_id"`
	Term         string  `json:"term"`
	Relationship string  `json:"relationship"` // parent, child, synonym, related
	Score        float64 `json:"score"`
	System       string  `json:"system"`
}

// TermHierarchy represents hierarchical information
type TermHierarchy struct {
	Path     []HierarchyNode `json:"path,omitempty"`
	Parents  []HierarchyNode `json:"parents,omitempty"`
	Children []HierarchyNode `json:"children,omitempty"`
	Level    int            `json:"level"`
}

// HierarchyNode represents a node in the hierarchy
type HierarchyNode struct {
	ConceptID string `json:"concept_id"`
	Term      string `json:"term"`
	Level     int    `json:"level"`
}

// CrossMapping represents cross-terminology mappings
type CrossMapping struct {
	TargetSystem    string  `json:"target_system"`
	TargetConceptID string  `json:"target_concept_id"`
	TargetTerm      string  `json:"target_term"`
	MappingType     string  `json:"mapping_type"` // equivalent, broader, narrower, related
	Confidence      float64 `json:"confidence"`
}

// TermUsageStats contains usage statistics
type TermUsageStats struct {
	UsageFrequency   int64   `json:"usage_frequency"`
	PopularityScore  float64 `json:"popularity_score"`
	TrendingScore    float64 `json:"trending_score,omitempty"`
	LastUsed         *time.Time `json:"last_used,omitempty"`
}

// SearchSuggestion represents a search suggestion
type SearchSuggestion struct {
	Query       string  `json:"query"`
	Type        string  `json:"type"`
	Confidence  float64 `json:"confidence"`
	Description string  `json:"description,omitempty"`
}

// QueryAnalysisResult contains query analysis information
type QueryAnalysisResult struct {
	DetectedIntent      SearchIntent         `json:"detected_intent"`
	DetectedQueryType   QueryType            `json:"detected_query_type"`
	ExtractedEntities   []*QueryEntity       `json:"extracted_entities,omitempty"`
	SuggestedFilters    map[string]string    `json:"suggested_filters,omitempty"`
	QueryComplexity     string               `json:"query_complexity"`
	ProcessingSteps     []string             `json:"processing_steps,omitempty"`
	Confidence          float64              `json:"confidence"`
}

// QueryEntity represents an entity extracted from the query
type QueryEntity struct {
	Text       string  `json:"text"`
	Type       string  `json:"type"`       // code, term, system, domain
	Confidence float64 `json:"confidence"`
	Span       [2]int  `json:"span"`       // start and end positions
}

// NewSearchEngine creates a new clinical search engine
func NewSearchEngine(
	esIntegration *elasticsearch.Integration,
	logger *zap.Logger,
	metrics *metrics.Collector,
	config *SearchConfig,
) *SearchEngine {
	if config == nil {
		config = DefaultSearchConfig()
	}

	return &SearchEngine{
		esIntegration: esIntegration,
		logger:        logger,
		metrics:       metrics,
		config:        config,
	}
}

// DefaultSearchConfig returns default search configuration
func DefaultSearchConfig() *SearchConfig {
	return &SearchConfig{
		DefaultPageSize:       20,
		MaxPageSize:          100,
		EnableFacetedSearch:  true,
		EnableAutoComplete:   true,
		EnableSpellCorrection: true,
		SearchTimeout:        30 * time.Second,
		CacheTTL:            5 * time.Minute,
		BoostFactors: map[string]float64{
			"exact_match":       3.0,
			"preferred_term":    2.5,
			"synonyms":         2.0,
			"recent_usage":     1.5,
			"high_frequency":   1.3,
		},
		DefaultFilters: map[string]string{
			"status": "active",
		},
		EnableMetrics:         true,
		QueryAnalysisEnabled: true,
	}
}

// Search performs a comprehensive clinical terminology search
func (se *SearchEngine) Search(ctx context.Context, request *ClinicalSearchRequest) (*ClinicalSearchResponse, error) {
	searchID := fmt.Sprintf("search_%d", time.Now().UnixNano())
	startTime := time.Now()

	se.logger.Info("Starting clinical search",
		zap.String("search_id", searchID),
		zap.String("query", request.Query),
		zap.String("search_mode", string(request.SearchMode)),
	)

	// Create search context with timeout
	searchCtx, cancel := context.WithTimeout(ctx, se.config.SearchTimeout)
	defer cancel()

	// Initialize response
	response := &ClinicalSearchResponse{
		SearchID:    searchID,
		Query:       request.Query,
		SearchMode:  request.SearchMode,
		Timestamp:   startTime,
		Page:        request.Page,
		PageSize:    se.getEffectivePageSize(request.PageSize),
		Results:     make([]*ClinicalSearchResult, 0),
	}

	// Query analysis (if enabled)
	if se.config.QueryAnalysisEnabled && request.Query != "" {
		if analysis, err := se.analyzeQuery(request); err == nil {
			response.QueryAnalysis = analysis
			response.ProcessedQuery = se.processQuery(request, analysis)
		}
	}

	// Execute search based on mode
	var searchResults *elasticsearch.SearchResults
	var err error

	switch request.SearchMode {
	case SearchModeStandard:
		searchResults, err = se.executeStandardSearch(searchCtx, request)
	case SearchModeExact:
		searchResults, err = se.executeExactSearch(searchCtx, request)
	case SearchModeAutocomplete:
		searchResults, err = se.executeAutocompleteSearch(searchCtx, request)
	case SearchModePhonetic:
		searchResults, err = se.executePhoneticSearch(searchCtx, request)
	case SearchModeFuzzy:
		searchResults, err = se.executeFuzzySearch(searchCtx, request)
	case SearchModeWildcard:
		searchResults, err = se.executeWildcardSearch(searchCtx, request)
	case SearchModeSemantic:
		searchResults, err = se.executeSemanticSearch(searchCtx, request)
	case SearchModeHybrid:
		searchResults, err = se.executeHybridSearch(searchCtx, request)
	default:
		searchResults, err = se.executeStandardSearch(searchCtx, request)
	}

	if err != nil {
		se.logger.Error("Search execution failed",
			zap.String("search_id", searchID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("search execution failed: %w", err)
	}

	// Convert results
	response.Results = se.convertSearchResults(searchResults.Results)
	response.TotalCount = int64(searchResults.Total)
	response.ReturnedCount = len(response.Results)
	response.HasNextPage = se.calculateHasNextPage(response)

	// Add search enhancements
	if request.IncludeHighlights {
		se.addHighlights(response, searchResults)
	}

	if request.IncludeFacets && se.config.EnableFacetedSearch {
		response.Facets = se.generateFacets(searchResults)
	}

	if request.IncludeSpellCheck && se.config.EnableSpellCorrection {
		response.SpellCheck = se.generateSpellCheck(request.Query)
	}

	if request.IncludeRelated {
		response.RelatedTerms = se.findRelatedTerms(searchCtx, request)
	}

	// Generate suggestions
	response.Suggestions = se.generateSuggestions(searchCtx, request, response)

	// Calculate performance metrics
	response.SearchTime = time.Since(startTime)
	response.ProcessingTime = time.Duration(searchResults.Took) * time.Millisecond

	// Record metrics
	if se.config.EnableMetrics {
		se.recordSearchMetrics(request, response)
	}

	se.logger.Info("Clinical search completed",
		zap.String("search_id", searchID),
		zap.Int64("total_count", response.TotalCount),
		zap.Int("returned_count", response.ReturnedCount),
		zap.Duration("search_time", response.SearchTime),
	)

	return response, nil
}

// executeStandardSearch performs standard clinical terminology search
func (se *SearchEngine) executeStandardSearch(ctx context.Context, request *ClinicalSearchRequest) (*elasticsearch.SearchResults, error) {
	esRequest := &elasticsearch.SearchRequest{
		Query:           request.Query,
		Systems:         request.Systems,
		Domains:         request.Domains,
		SemanticTags:    request.SemanticTags,
		Status:          request.Status,
		SearchType:      elasticsearch.SearchTypeStandard,
		Size:            request.PageSize,
		From:            request.Page * request.PageSize,
		IncludeInactive: request.IncludeInactive,
		ExactMatch:      request.ExactMatch,
		SortBy:          request.SortBy,
		SortOrder:       request.SortOrder,
	}

	// Add custom filters
	if request.CustomFilters != nil {
		esRequest.Filters = make(map[string]string)
		for k, v := range request.CustomFilters {
			if strVal, ok := v.(string); ok {
				esRequest.Filters[k] = strVal
			}
		}
	}

	return se.esIntegration.SearchTerms(ctx, esRequest)
}

// executeExactSearch performs exact matching search
func (se *SearchEngine) executeExactSearch(ctx context.Context, request *ClinicalSearchRequest) (*elasticsearch.SearchResults, error) {
	esRequest := &elasticsearch.SearchRequest{
		Query:           request.Query,
		Systems:         request.Systems,
		Domains:         request.Domains,
		SemanticTags:    request.SemanticTags,
		Status:          request.Status,
		SearchType:      elasticsearch.SearchTypeExact,
		Size:            request.PageSize,
		From:            request.Page * request.PageSize,
		IncludeInactive: request.IncludeInactive,
		ExactMatch:      true,
		SortBy:          request.SortBy,
		SortOrder:       request.SortOrder,
	}

	return se.esIntegration.SearchTerms(ctx, esRequest)
}

// executeAutocompleteSearch performs autocomplete search
func (se *SearchEngine) executeAutocompleteSearch(ctx context.Context, request *ClinicalSearchRequest) (*elasticsearch.SearchResults, error) {
	esRequest := &elasticsearch.SearchRequest{
		Query:           request.Query,
		Systems:         request.Systems,
		Domains:         request.Domains,
		SearchType:      elasticsearch.SearchTypeAutocomplete,
		Size:            request.PageSize,
		From:            0, // Autocomplete typically doesn't paginate
		IncludeInactive: request.IncludeInactive,
		SortBy:          "usage_frequency",
		SortOrder:       "desc",
	}

	return se.esIntegration.SearchTerms(ctx, esRequest)
}

// executePhoneticSearch performs phonetic matching search
func (se *SearchEngine) executePhoneticSearch(ctx context.Context, request *ClinicalSearchRequest) (*elasticsearch.SearchResults, error) {
	esRequest := &elasticsearch.SearchRequest{
		Query:           request.Query,
		Systems:         request.Systems,
		Domains:         request.Domains,
		SearchType:      elasticsearch.SearchTypePhonetic,
		Size:            request.PageSize,
		From:            request.Page * request.PageSize,
		IncludeInactive: request.IncludeInactive,
	}

	return se.esIntegration.SearchTerms(ctx, esRequest)
}

// executeFuzzySearch performs fuzzy matching search
func (se *SearchEngine) executeFuzzySearch(ctx context.Context, request *ClinicalSearchRequest) (*elasticsearch.SearchResults, error) {
	esRequest := &elasticsearch.SearchRequest{
		Query:           request.Query,
		Systems:         request.Systems,
		Domains:         request.Domains,
		SearchType:      elasticsearch.SearchTypeFuzzy,
		Size:            request.PageSize,
		From:            request.Page * request.PageSize,
		IncludeInactive: request.IncludeInactive,
	}

	return se.esIntegration.SearchTerms(ctx, esRequest)
}

// executeWildcardSearch performs wildcard pattern matching
func (se *SearchEngine) executeWildcardSearch(ctx context.Context, request *ClinicalSearchRequest) (*elasticsearch.SearchResults, error) {
	esRequest := &elasticsearch.SearchRequest{
		Query:           request.Query,
		Systems:         request.Systems,
		Domains:         request.Domains,
		SearchType:      elasticsearch.SearchTypeWildcard,
		Size:            request.PageSize,
		From:            request.Page * request.PageSize,
		IncludeInactive: request.IncludeInactive,
	}

	return se.esIntegration.SearchTerms(ctx, esRequest)
}

// executeSemanticSearch performs semantic similarity search
func (se *SearchEngine) executeSemanticSearch(ctx context.Context, request *ClinicalSearchRequest) (*elasticsearch.SearchResults, error) {
	// For now, use standard search with enhanced synonym matching
	// In future, this could integrate with vector embeddings
	return se.executeStandardSearch(ctx, request)
}

// executeHybridSearch combines multiple search strategies
func (se *SearchEngine) executeHybridSearch(ctx context.Context, request *ClinicalSearchRequest) (*elasticsearch.SearchResults, error) {
	// Execute multiple search strategies and combine results
	strategies := []ClinicalSearchMode{
		SearchModeExact,
		SearchModeStandard,
		SearchModeFuzzy,
	}

	allResults := make([]*elasticsearch.SearchResult, 0)
	seen := make(map[string]bool)

	for _, strategy := range strategies {
		strategyRequest := *request
		strategyRequest.SearchMode = strategy
		strategyRequest.PageSize = request.PageSize / len(strategies) // Distribute page size

		results, err := se.executeSearchByMode(ctx, &strategyRequest)
		if err != nil {
			se.logger.Warn("Hybrid search strategy failed",
				zap.String("strategy", string(strategy)),
				zap.Error(err),
			)
			continue
		}

		// Deduplicate and add results
		for _, result := range results.Results {
			if !seen[result.Term.TermID] {
				seen[result.Term.TermID] = true
				allResults = append(allResults, result)
			}
		}
	}

	// Sort combined results by score
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	// Limit to requested page size
	if len(allResults) > request.PageSize {
		allResults = allResults[:request.PageSize]
	}

	return &elasticsearch.SearchResults{
		Total:   len(allResults),
		Results: allResults,
	}, nil
}

// Helper methods

func (se *SearchEngine) executeSearchByMode(ctx context.Context, request *ClinicalSearchRequest) (*elasticsearch.SearchResults, error) {
	switch request.SearchMode {
	case SearchModeExact:
		return se.executeExactSearch(ctx, request)
	case SearchModePhonetic:
		return se.executePhoneticSearch(ctx, request)
	case SearchModeFuzzy:
		return se.executeFuzzySearch(ctx, request)
	default:
		return se.executeStandardSearch(ctx, request)
	}
}

func (se *SearchEngine) getEffectivePageSize(requested int) int {
	if requested <= 0 {
		return se.config.DefaultPageSize
	}
	if requested > se.config.MaxPageSize {
		return se.config.MaxPageSize
	}
	return requested
}

func (se *SearchEngine) convertSearchResults(esResults []*elasticsearch.SearchResult) []*ClinicalSearchResult {
	results := make([]*ClinicalSearchResult, len(esResults))

	for i, esResult := range esResults {
		results[i] = &ClinicalSearchResult{
			TermID:           esResult.Term.TermID,
			ConceptID:        esResult.Term.ConceptID,
			Term:             esResult.Term.Term,
			PreferredTerm:    esResult.Term.PreferredTerm,
			Definition:       esResult.Term.Definition,
			System:           esResult.Term.TerminologySystem,
			Version:          esResult.Term.TerminologyVersion,
			Status:           esResult.Term.Status,
			Score:            esResult.Score,
			Rank:             i + 1,
			MatchType:        esResult.MatchReason,
			Synonyms:         esResult.Term.Synonyms,
			SemanticTags:     esResult.Term.SemanticTags,
			ClinicalDomain:   esResult.Term.ClinicalDomain,
			Confidence:       se.calculateConfidence(esResult.Score),
		}

		// Add matched fields
		results[i].MatchedFields = se.determineMatchedFields(esResult)

		// Add cross mappings from FHIR mappings
		if len(esResult.Term.FHIRMappings) > 0 {
			results[i].CrossMappings = make([]*CrossMapping, len(esResult.Term.FHIRMappings))
			for j, fhirMapping := range esResult.Term.FHIRMappings {
				results[i].CrossMappings[j] = &CrossMapping{
					TargetSystem:    fhirMapping.System,
					TargetConceptID: fhirMapping.Code,
					TargetTerm:      fhirMapping.Display,
					MappingType:     "equivalent",
					Confidence:      1.0,
				}
			}
		}

		// Add usage stats
		if esResult.Term.UsageFrequency > 0 {
			results[i].UsageStats = &TermUsageStats{
				UsageFrequency:  esResult.Term.UsageFrequency,
				PopularityScore: se.calculatePopularityScore(esResult.Term.UsageFrequency),
			}
		}
	}

	return results
}

func (se *SearchEngine) calculateHasNextPage(response *ClinicalSearchResponse) bool {
	totalPages := (response.TotalCount + int64(response.PageSize) - 1) / int64(response.PageSize)
	return int64(response.Page+1) < totalPages
}

func (se *SearchEngine) calculateConfidence(score float64) float64 {
	// Normalize score to confidence between 0 and 1
	// This is a simplified calculation - could be more sophisticated
	return score / (score + 1.0)
}

func (se *SearchEngine) calculatePopularityScore(frequency int64) float64 {
	// Simple logarithmic scaling
	if frequency <= 0 {
		return 0.0
	}
	return 1.0 - (1.0 / (1.0 + float64(frequency)/1000.0))
}

func (se *SearchEngine) determineMatchedFields(result *elasticsearch.SearchResult) []string {
	// This would analyze which fields contributed to the match
	// For now, return common fields
	return []string{"term", "preferred_term"}
}

// Placeholder implementations for additional features

func (se *SearchEngine) analyzeQuery(request *ClinicalSearchRequest) (*QueryAnalysisResult, error) {
	// Simple query analysis implementation
	analysis := &QueryAnalysisResult{
		DetectedIntent:    IntentLookup,
		DetectedQueryType: QueryTypeGeneral,
		QueryComplexity:   "simple",
		Confidence:        0.8,
		ProcessingSteps:   []string{"tokenization", "entity_extraction", "intent_detection"},
	}

	// Detect query type based on keywords
	query := strings.ToLower(request.Query)
	if strings.Contains(query, "drug") || strings.Contains(query, "medication") {
		analysis.DetectedQueryType = QueryTypeMedication
	} else if strings.Contains(query, "diagnosis") || strings.Contains(query, "disease") {
		analysis.DetectedQueryType = QueryTypeDiagnostic
	} else if strings.Contains(query, "procedure") || strings.Contains(query, "surgery") {
		analysis.DetectedQueryType = QueryTypeProcedural
	}

	return analysis, nil
}

func (se *SearchEngine) processQuery(request *ClinicalSearchRequest, analysis *QueryAnalysisResult) string {
	// Process query based on analysis
	return strings.TrimSpace(request.Query)
}

func (se *SearchEngine) addHighlights(response *ClinicalSearchResponse, searchResults *elasticsearch.SearchResults) {
	// Add highlighting information to results
	response.Highlights = make(map[string][]string)
	// Implementation would extract highlights from Elasticsearch response
}

func (se *SearchEngine) generateFacets(searchResults *elasticsearch.SearchResults) *SearchFacets {
	// Generate faceted search results
	return &SearchFacets{
		Systems: []*FacetValue{
			{Value: "SNOMED_CT", Label: "SNOMED CT", Count: 1000},
			{Value: "RXNORM", Label: "RxNorm", Count: 500},
			{Value: "ICD10CM", Label: "ICD-10-CM", Count: 300},
			{Value: "LOINC", Label: "LOINC", Count: 200},
		},
	}
}

func (se *SearchEngine) generateSpellCheck(query string) *SpellCheckSuggestion {
	// Generate spell check suggestions
	return &SpellCheckSuggestion{
		OriginalQuery:  query,
		HasSuggestions: false,
		Confidence:     1.0,
	}
}

func (se *SearchEngine) findRelatedTerms(ctx context.Context, request *ClinicalSearchRequest) []*RelatedTerm {
	// Find related terms based on the search
	return []*RelatedTerm{}
}

func (se *SearchEngine) generateSuggestions(ctx context.Context, request *ClinicalSearchRequest, response *ClinicalSearchResponse) []*SearchSuggestion {
	// Generate search suggestions
	return []*SearchSuggestion{}
}

func (se *SearchEngine) recordSearchMetrics(request *ClinicalSearchRequest, response *ClinicalSearchResponse) {
	// Record search metrics
	se.metrics.RecordSearchMetric("search_requests_total", "complete")
	se.metrics.RecordSearchMetric("search_results_count", fmt.Sprintf("%d", response.ReturnedCount))
	se.metrics.RecordSearchMetric("search_time_seconds", fmt.Sprintf("%.3f", response.SearchTime.Seconds()))

	labels := map[string]string{
		"search_mode": string(request.SearchMode),
		"query_type":  string(request.QueryType),
	}
	se.metrics.IncrementCounterWithLabels("clinical_searches_total", labels)
}