package search

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/elasticsearch"
	"kb-7-terminology/internal/metrics"

	"go.uber.org/zap"
)

// AutocompleteService provides real-time search suggestions for clinical terminology
type AutocompleteService struct {
	esIntegration *elasticsearch.Integration
	cache         cache.EnhancedCache
	logger        *zap.Logger
	metrics       *metrics.Collector
	config        *AutocompleteConfig
}

// AutocompleteConfig holds configuration for autocomplete functionality
type AutocompleteConfig struct {
	MinQueryLength       int                 `json:"min_query_length"`
	MaxSuggestions      int                 `json:"max_suggestions"`
	CacheTTL            time.Duration       `json:"cache_ttl"`
	EnablePersonalization bool              `json:"enable_personalization"`
	EnableTrending      bool                `json:"enable_trending"`
	EnableSpellCorrection bool              `json:"enable_spell_correction"`
	ResponseTimeout     time.Duration       `json:"response_timeout"`
	BoostFactors        AutocompleteBoostConfig `json:"boost_factors"`
	FilterDefaults      map[string]string   `json:"filter_defaults"`
}

// AutocompleteBoostConfig defines boost factors for different suggestion types
type AutocompleteBoostConfig struct {
	ExactPrefix        float64 `json:"exact_prefix"`
	PopularTerms       float64 `json:"popular_terms"`
	RecentlyUsed       float64 `json:"recently_used"`
	UserHistory        float64 `json:"user_history"`
	TrendingTerms      float64 `json:"trending_terms"`
	PreferredSystems   float64 `json:"preferred_systems"`
	SemanticMatch      float64 `json:"semantic_match"`
}

// AutocompleteRequest represents a request for autocomplete suggestions
type AutocompleteRequest struct {
	Query           string            `json:"query"`
	Context         *UserContext      `json:"context,omitempty"`
	Filters         *SuggestionFilters `json:"filters,omitempty"`
	Options         *SuggestionOptions `json:"options,omitempty"`
	MaxSuggestions  int               `json:"max_suggestions,omitempty"`
}

// UserContext provides context about the user making the request
type UserContext struct {
	UserID        string            `json:"user_id,omitempty"`
	SessionID     string            `json:"session_id,omitempty"`
	Role          string            `json:"role,omitempty"`
	Specialty     string            `json:"specialty,omitempty"`
	SearchHistory []string          `json:"search_history,omitempty"`
	Preferences   map[string]string `json:"preferences,omitempty"`
}

// SuggestionFilters defines filters for autocomplete suggestions
type SuggestionFilters struct {
	Systems      []string `json:"systems,omitempty"`
	Domains      []string `json:"domains,omitempty"`
	Languages    []string `json:"languages,omitempty"`
	OnlyActive   bool     `json:"only_active"`
	MinFrequency int64    `json:"min_frequency,omitempty"`
}

// SuggestionOptions defines options for suggestion behavior
type SuggestionOptions struct {
	IncludeDefinitions bool `json:"include_definitions"`
	IncludeContext     bool `json:"include_context"`
	GroupBySystems     bool `json:"group_by_systems"`
	HighlightMatch     bool `json:"highlight_match"`
	IncludeMetadata    bool `json:"include_metadata"`
}

// AutocompleteResponse contains autocomplete suggestions
type AutocompleteResponse struct {
	Query         string                  `json:"query"`
	Suggestions   []*Suggestion          `json:"suggestions"`
	Groups        map[string][]*Suggestion `json:"groups,omitempty"`
	TotalCount    int                     `json:"total_count"`
	ResponseTime  time.Duration           `json:"response_time"`
	CacheHit      bool                    `json:"cache_hit,omitempty"`
	Metadata      *ResponseMetadata       `json:"metadata,omitempty"`
}

// Suggestion represents a single autocomplete suggestion
type Suggestion struct {
	// Core suggestion data
	ID              string            `json:"id"`
	Text            string            `json:"text"`
	DisplayText     string            `json:"display_text,omitempty"`
	Type            SuggestionType    `json:"type"`
	Score           float64           `json:"score"`
	Rank            int              `json:"rank"`

	// Clinical terminology data
	ConceptID       string            `json:"concept_id,omitempty"`
	System          string            `json:"system,omitempty"`
	SystemLabel     string            `json:"system_label,omitempty"`
	Definition      string            `json:"definition,omitempty"`
	SemanticTags    []string          `json:"semantic_tags,omitempty"`
	ClinicalDomain  string            `json:"clinical_domain,omitempty"`

	// Suggestion metadata
	Frequency       int64             `json:"frequency,omitempty"`
	PopularityScore float64           `json:"popularity_score,omitempty"`
	TrendingScore   float64           `json:"trending_score,omitempty"`
	ContextualScore float64           `json:"contextual_score,omitempty"`
	MatchType       string            `json:"match_type,omitempty"`
	MatchHighlight  string            `json:"match_highlight,omitempty"`

	// Additional context
	ParentTerms     []string          `json:"parent_terms,omitempty"`
	ChildrenCount   int               `json:"children_count,omitempty"`
	Synonyms        []string          `json:"synonyms,omitempty"`
	AlternativeTerms []string         `json:"alternative_terms,omitempty"`
	UsageContext    string            `json:"usage_context,omitempty"`

	// Personalization
	UserRelevance   float64           `json:"user_relevance,omitempty"`
	RecentlyUsed    bool              `json:"recently_used,omitempty"`
	UserBookmarked  bool              `json:"user_bookmarked,omitempty"`
}

// SuggestionType defines the type of suggestion
type SuggestionType string

const (
	SuggestionTypeTerm        SuggestionType = "term"
	SuggestionTypeCode        SuggestionType = "code"
	SuggestionTypeSystem      SuggestionType = "system"
	SuggestionTypeDomain      SuggestionType = "domain"
	SuggestionTypeQuery       SuggestionType = "query"
	SuggestionTypeCorrection  SuggestionType = "correction"
	SuggestionTypeTrending    SuggestionType = "trending"
	SuggestionTypePersonal    SuggestionType = "personal"
)

// ResponseMetadata contains metadata about the autocomplete response
type ResponseMetadata struct {
	Algorithm         string            `json:"algorithm"`
	DataSources       []string          `json:"data_sources"`
	PersonalizationUsed bool            `json:"personalization_used"`
	TrendingDataUsed  bool              `json:"trending_data_used"`
	CacheStats        *CacheStats       `json:"cache_stats,omitempty"`
	PerformanceStats  *PerformanceStats `json:"performance_stats,omitempty"`
}

// CacheStats contains cache performance statistics
type CacheStats struct {
	CacheHit      bool          `json:"cache_hit"`
	CacheKey      string        `json:"cache_key,omitempty"`
	TTL           time.Duration `json:"ttl,omitempty"`
	HitRatio      float64       `json:"hit_ratio,omitempty"`
}

// PerformanceStats contains performance statistics
type PerformanceStats struct {
	QueryTime       time.Duration `json:"query_time"`
	ProcessingTime  time.Duration `json:"processing_time"`
	CacheTime       time.Duration `json:"cache_time"`
	RankingTime     time.Duration `json:"ranking_time"`
	TotalTime       time.Duration `json:"total_time"`
}

// NewAutocompleteService creates a new autocomplete service
func NewAutocompleteService(
	esIntegration *elasticsearch.Integration,
	cache cache.EnhancedCache,
	logger *zap.Logger,
	metrics *metrics.Collector,
	config *AutocompleteConfig,
) *AutocompleteService {
	if config == nil {
		config = DefaultAutocompleteConfig()
	}

	return &AutocompleteService{
		esIntegration: esIntegration,
		cache:         cache,
		logger:        logger,
		metrics:       metrics,
		config:        config,
	}
}

// DefaultAutocompleteConfig returns default autocomplete configuration
func DefaultAutocompleteConfig() *AutocompleteConfig {
	return &AutocompleteConfig{
		MinQueryLength:        2,
		MaxSuggestions:       10,
		CacheTTL:             5 * time.Minute,
		EnablePersonalization: true,
		EnableTrending:       true,
		EnableSpellCorrection: true,
		ResponseTimeout:      2 * time.Second,
		BoostFactors: AutocompleteBoostConfig{
			ExactPrefix:      3.0,
			PopularTerms:     2.0,
			RecentlyUsed:     2.5,
			UserHistory:      1.8,
			TrendingTerms:    1.5,
			PreferredSystems: 1.3,
			SemanticMatch:    1.2,
		},
		FilterDefaults: map[string]string{
			"status": "active",
		},
	}
}

// GetSuggestions returns autocomplete suggestions for a query
func (as *AutocompleteService) GetSuggestions(ctx context.Context, request *AutocompleteRequest) (*AutocompleteResponse, error) {
	startTime := time.Now()

	// Validate request
	if len(request.Query) < as.config.MinQueryLength {
		return &AutocompleteResponse{
			Query:        request.Query,
			Suggestions:  []*Suggestion{},
			TotalCount:   0,
			ResponseTime: time.Since(startTime),
		}, nil
	}

	// Create context with timeout
	searchCtx, cancel := context.WithTimeout(ctx, as.config.ResponseTimeout)
	defer cancel()

	// Check cache first
	cacheKey := as.generateCacheKey(request)
	if cached, exists := as.getCachedSuggestions(cacheKey); exists {
		cached.ResponseTime = time.Since(startTime)
		cached.CacheHit = true
		as.recordMetrics("cache_hit", request, cached)
		return cached, nil
	}

	as.logger.Debug("Generating autocomplete suggestions",
		zap.String("query", request.Query),
		zap.String("cache_key", cacheKey),
	)

	// Generate suggestions
	response, err := as.generateSuggestions(searchCtx, request)
	if err != nil {
		as.logger.Error("Failed to generate suggestions",
			zap.String("query", request.Query),
			zap.Error(err),
		)
		return nil, fmt.Errorf("suggestion generation failed: %w", err)
	}

	// Calculate response time
	response.ResponseTime = time.Since(startTime)
	response.CacheHit = false

	// Cache the response
	as.cacheSuggestions(cacheKey, response)

	// Record metrics
	as.recordMetrics("generated", request, response)

	return response, nil
}

// generateSuggestions generates autocomplete suggestions using multiple strategies
func (as *AutocompleteService) generateSuggestions(ctx context.Context, request *AutocompleteRequest) (*AutocompleteResponse, error) {
	response := &AutocompleteResponse{
		Query:       request.Query,
		Suggestions: make([]*Suggestion, 0),
	}

	// Strategy 1: Prefix matching suggestions
	prefixSuggestions, err := as.getPrefixSuggestions(ctx, request)
	if err != nil {
		as.logger.Warn("Prefix suggestions failed", zap.Error(err))
	} else {
		response.Suggestions = append(response.Suggestions, prefixSuggestions...)
	}

	// Strategy 2: Popular terms suggestions
	popularSuggestions, err := as.getPopularSuggestions(ctx, request)
	if err != nil {
		as.logger.Warn("Popular suggestions failed", zap.Error(err))
	} else {
		response.Suggestions = append(response.Suggestions, popularSuggestions...)
	}

	// Strategy 3: User history suggestions (if personalization enabled)
	if as.config.EnablePersonalization && request.Context != nil && request.Context.UserID != "" {
		historySuggestions, err := as.getUserHistorySuggestions(ctx, request)
		if err != nil {
			as.logger.Warn("History suggestions failed", zap.Error(err))
		} else {
			response.Suggestions = append(response.Suggestions, historySuggestions...)
		}
	}

	// Strategy 4: Trending terms suggestions (if trending enabled)
	if as.config.EnableTrending {
		trendingSuggestions, err := as.getTrendingSuggestions(ctx, request)
		if err != nil {
			as.logger.Warn("Trending suggestions failed", zap.Error(err))
		} else {
			response.Suggestions = append(response.Suggestions, trendingSuggestions...)
		}
	}

	// Strategy 5: Spell correction suggestions (if enabled)
	if as.config.EnableSpellCorrection {
		correctionSuggestions, err := as.getSpellCorrectionSuggestions(ctx, request)
		if err != nil {
			as.logger.Warn("Spell correction suggestions failed", zap.Error(err))
		} else {
			response.Suggestions = append(response.Suggestions, correctionSuggestions...)
		}
	}

	// Deduplicate, score, and rank suggestions
	response.Suggestions = as.processAndRankSuggestions(request, response.Suggestions)

	// Limit to max suggestions
	maxSuggestions := as.getMaxSuggestions(request)
	if len(response.Suggestions) > maxSuggestions {
		response.Suggestions = response.Suggestions[:maxSuggestions]
	}

	// Add metadata
	response.TotalCount = len(response.Suggestions)
	response.Metadata = as.generateResponseMetadata(request, response)

	// Group suggestions if requested
	if request.Options != nil && request.Options.GroupBySystems {
		response.Groups = as.groupSuggestionsBySystem(response.Suggestions)
	}

	return response, nil
}

// getPrefixSuggestions gets suggestions based on prefix matching
func (as *AutocompleteService) getPrefixSuggestions(ctx context.Context, request *AutocompleteRequest) ([]*Suggestion, error) {
	esRequest := &elasticsearch.SearchRequest{
		Query:           request.Query,
		SearchType:      elasticsearch.SearchTypeAutocomplete,
		Size:            as.config.MaxSuggestions * 2, // Get more for deduplication
		IncludeInactive: false,
		SortBy:          "usage_frequency",
		SortOrder:       "desc",
	}

	// Apply filters
	if request.Filters != nil {
		esRequest.Systems = request.Filters.Systems
		esRequest.Domains = request.Filters.Domains
	}

	results, err := as.esIntegration.SearchTerms(ctx, esRequest)
	if err != nil {
		return nil, err
	}

	suggestions := make([]*Suggestion, 0)
	for i, result := range results.Results {
		suggestion := &Suggestion{
			ID:              result.Term.TermID,
			Text:            result.Term.Term,
			DisplayText:     result.Term.PreferredTerm,
			Type:            SuggestionTypeTerm,
			Score:           result.Score * as.config.BoostFactors.ExactPrefix,
			Rank:            i + 1,
			ConceptID:       result.Term.ConceptID,
			System:          result.Term.TerminologySystem,
			SystemLabel:     as.getSystemLabel(result.Term.TerminologySystem),
			Definition:      result.Term.Definition,
			SemanticTags:    result.Term.SemanticTags,
			ClinicalDomain:  result.Term.ClinicalDomain,
			Frequency:       result.Term.UsageFrequency,
			PopularityScore: as.calculatePopularityScore(result.Term.UsageFrequency),
			MatchType:       "prefix",
			Synonyms:        result.Term.Synonyms,
		}

		// Add highlighting
		if request.Options != nil && request.Options.HighlightMatch {
			suggestion.MatchHighlight = as.highlightMatch(result.Term.Term, request.Query)
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// getPopularSuggestions gets popular terms as suggestions
func (as *AutocompleteService) getPopularSuggestions(ctx context.Context, request *AutocompleteRequest) ([]*Suggestion, error) {
	esRequest := &elasticsearch.SearchRequest{
		Query:           request.Query,
		SearchType:      elasticsearch.SearchTypeFuzzy,
		Size:            as.config.MaxSuggestions / 2,
		IncludeInactive: false,
		SortBy:          "usage_frequency",
		SortOrder:       "desc",
	}

	// Apply filters
	if request.Filters != nil {
		esRequest.Systems = request.Filters.Systems
		esRequest.Domains = request.Filters.Domains
	}

	results, err := as.esIntegration.SearchTerms(ctx, esRequest)
	if err != nil {
		return nil, err
	}

	suggestions := make([]*Suggestion, 0)
	for _, result := range results.Results {
		suggestion := &Suggestion{
			ID:              result.Term.TermID,
			Text:            result.Term.Term,
			DisplayText:     result.Term.PreferredTerm,
			Type:            SuggestionTypeTerm,
			Score:           result.Score * as.config.BoostFactors.PopularTerms,
			ConceptID:       result.Term.ConceptID,
			System:          result.Term.TerminologySystem,
			SystemLabel:     as.getSystemLabel(result.Term.TerminologySystem),
			Frequency:       result.Term.UsageFrequency,
			PopularityScore: as.calculatePopularityScore(result.Term.UsageFrequency),
			MatchType:       "popular",
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// getUserHistorySuggestions gets suggestions based on user search history
func (as *AutocompleteService) getUserHistorySuggestions(ctx context.Context, request *AutocompleteRequest) ([]*Suggestion, error) {
	if request.Context == nil || len(request.Context.SearchHistory) == 0 {
		return []*Suggestion{}, nil
	}

	suggestions := make([]*Suggestion, 0)
	query := strings.ToLower(request.Query)

	// Look for matches in user's search history
	for _, historyItem := range request.Context.SearchHistory {
		if strings.HasPrefix(strings.ToLower(historyItem), query) {
			suggestion := &Suggestion{
				ID:              fmt.Sprintf("history_%s", historyItem),
				Text:            historyItem,
				DisplayText:     historyItem,
				Type:            SuggestionTypePersonal,
				Score:           as.config.BoostFactors.UserHistory,
				MatchType:       "user_history",
				RecentlyUsed:    true,
				UserRelevance:   1.0,
			}

			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions, nil
}

// getTrendingSuggestions gets trending terms as suggestions
func (as *AutocompleteService) getTrendingSuggestions(ctx context.Context, request *AutocompleteRequest) ([]*Suggestion, error) {
	// This would connect to a trending analysis service
	// For now, return empty slice
	return []*Suggestion{}, nil
}

// getSpellCorrectionSuggestions gets spell correction suggestions
func (as *AutocompleteService) getSpellCorrectionSuggestions(ctx context.Context, request *AutocompleteRequest) ([]*Suggestion, error) {
	// This would implement spell correction logic
	// For now, return empty slice
	return []*Suggestion{}, nil
}

// processAndRankSuggestions deduplicates, scores, and ranks suggestions
func (as *AutocompleteService) processAndRankSuggestions(request *AutocompleteRequest, suggestions []*Suggestion) []*Suggestion {
	// Deduplicate by ID
	seen := make(map[string]*Suggestion)
	for _, suggestion := range suggestions {
		if existing, exists := seen[suggestion.ID]; exists {
			// Keep the one with higher score
			if suggestion.Score > existing.Score {
				seen[suggestion.ID] = suggestion
			}
		} else {
			seen[suggestion.ID] = suggestion
		}
	}

	// Convert back to slice
	deduplicated := make([]*Suggestion, 0, len(seen))
	for _, suggestion := range seen {
		deduplicated = append(deduplicated, suggestion)
	}

	// Apply personalization boost if enabled
	if as.config.EnablePersonalization && request.Context != nil {
		as.applyPersonalizationBoost(request.Context, deduplicated)
	}

	// Sort by score
	sort.Slice(deduplicated, func(i, j int) bool {
		return deduplicated[i].Score > deduplicated[j].Score
	})

	// Update ranks
	for i, suggestion := range deduplicated {
		suggestion.Rank = i + 1
	}

	return deduplicated
}

// applyPersonalizationBoost applies personalization boost to suggestions
func (as *AutocompleteService) applyPersonalizationBoost(context *UserContext, suggestions []*Suggestion) {
	for _, suggestion := range suggestions {
		// Boost based on user specialty
		if context.Specialty != "" && suggestion.ClinicalDomain == context.Specialty {
			suggestion.Score *= 1.2
			suggestion.UserRelevance += 0.2
		}

		// Boost based on user preferences
		if context.Preferences != nil {
			if preferredSystem, exists := context.Preferences["preferred_system"]; exists {
				if suggestion.System == preferredSystem {
					suggestion.Score *= as.config.BoostFactors.PreferredSystems
					suggestion.UserRelevance += 0.1
				}
			}
		}
	}
}

// Helper methods

func (as *AutocompleteService) generateCacheKey(request *AutocompleteRequest) string {
	// Generate a unique cache key based on request parameters
	key := fmt.Sprintf("autocomplete:%s", strings.ToLower(request.Query))

	if request.Filters != nil {
		if len(request.Filters.Systems) > 0 {
			key += fmt.Sprintf(":systems=%s", strings.Join(request.Filters.Systems, ","))
		}
		if len(request.Filters.Domains) > 0 {
			key += fmt.Sprintf(":domains=%s", strings.Join(request.Filters.Domains, ","))
		}
	}

	if request.Context != nil && request.Context.UserID != "" {
		key += fmt.Sprintf(":user=%s", request.Context.UserID)
	}

	return key
}

func (as *AutocompleteService) getCachedSuggestions(cacheKey string) (*AutocompleteResponse, bool) {
	if cached, err := as.cache.Get(cacheKey); err == nil {
		if response, ok := cached.(*AutocompleteResponse); ok {
			return response, true
		}
	}
	return nil, false
}

func (as *AutocompleteService) cacheSuggestions(cacheKey string, response *AutocompleteResponse) {
	as.cache.Set(cacheKey, response, as.config.CacheTTL)
}

func (as *AutocompleteService) getMaxSuggestions(request *AutocompleteRequest) int {
	if request.MaxSuggestions > 0 && request.MaxSuggestions < as.config.MaxSuggestions {
		return request.MaxSuggestions
	}
	return as.config.MaxSuggestions
}

func (as *AutocompleteService) calculatePopularityScore(frequency int64) float64 {
	if frequency <= 0 {
		return 0.0
	}
	// Logarithmic scaling
	return 1.0 - (1.0 / (1.0 + float64(frequency)/1000.0))
}

func (as *AutocompleteService) getSystemLabel(system string) string {
	labels := map[string]string{
		"SNOMED_CT": "SNOMED CT",
		"RXNORM":    "RxNorm",
		"ICD10CM":   "ICD-10-CM",
		"LOINC":     "LOINC",
		"CPT":       "CPT",
	}

	if label, exists := labels[system]; exists {
		return label
	}
	return system
}

func (as *AutocompleteService) highlightMatch(text, query string) string {
	// Simple highlighting - in production, use more sophisticated highlighting
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)

	if idx := strings.Index(lowerText, lowerQuery); idx >= 0 {
		return text[:idx] + "<mark>" + text[idx:idx+len(query)] + "</mark>" + text[idx+len(query):]
	}

	return text
}

func (as *AutocompleteService) generateResponseMetadata(request *AutocompleteRequest, response *AutocompleteResponse) *ResponseMetadata {
	return &ResponseMetadata{
		Algorithm:           "multi_strategy",
		DataSources:         []string{"elasticsearch", "cache"},
		PersonalizationUsed: as.config.EnablePersonalization && request.Context != nil && request.Context.UserID != "",
		TrendingDataUsed:    as.config.EnableTrending,
	}
}

func (as *AutocompleteService) groupSuggestionsBySystem(suggestions []*Suggestion) map[string][]*Suggestion {
	groups := make(map[string][]*Suggestion)

	for _, suggestion := range suggestions {
		system := suggestion.System
		if system == "" {
			system = "other"
		}

		groups[system] = append(groups[system], suggestion)
	}

	return groups
}

func (as *AutocompleteService) recordMetrics(eventType string, request *AutocompleteRequest, response *AutocompleteResponse) {
	as.metrics.RecordAutocompleteMetric("autocomplete_requests_total", "complete")
	as.metrics.RecordAutocompleteMetric("autocomplete_suggestions_count", fmt.Sprintf("%d", response.TotalCount))
	as.metrics.RecordAutocompleteMetric("autocomplete_response_time_seconds", fmt.Sprintf("%.3f", response.ResponseTime.Seconds()))

	labels := map[string]string{
		"event_type": eventType,
		"query_len":  fmt.Sprintf("%d", len(request.Query)),
	}

	if request.Context != nil && request.Context.UserID != "" {
		labels["personalized"] = "true"
	} else {
		labels["personalized"] = "false"
	}

	as.metrics.IncrementCounterWithLabels("autocomplete_events_total", labels)
}