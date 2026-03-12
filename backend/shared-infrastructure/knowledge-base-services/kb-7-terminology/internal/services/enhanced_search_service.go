package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/models"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// SearchOptions configures search behavior
type SearchOptions struct {
	TargetSystem    string  `json:"target_system,omitempty"`
	ExpandSynonyms  bool    `json:"expand_synonyms"`
	UsePhonetic     bool    `json:"use_phonetic"`
	MaxResults      int     `json:"max_results"`
	IncludeInactive bool    `json:"include_inactive"`
	MinRank         float64 `json:"min_rank"`
}

// SearchResult represents a search result with ranking information
type SearchResult struct {
	models.Concept
	Rank      float64 `json:"rank"`
	MatchType string  `json:"match_type"`
	Highlight string  `json:"highlight,omitempty"`
}

// SearchResponse contains search results and metadata
type SearchResponse struct {
	Results          []SearchResult `json:"results"`
	TotalCount       int            `json:"total_count"`
	SearchTerm       string         `json:"search_term"`
	ExpandedTerms    []string       `json:"expanded_terms,omitempty"`
	SearchDurationMs float64        `json:"search_duration_ms"`
	UsedCache        bool           `json:"used_cache"`
	SearchOptions    SearchOptions  `json:"search_options"`
}

// EnhancedSearchService provides advanced medical terminology search capabilities
type EnhancedSearchService struct {
	db      *sql.DB
	cache   cache.EnhancedCache
	logger  *logrus.Logger
	metrics *metrics.Collector
}

// NewEnhancedSearchService creates a new enhanced search service
func NewEnhancedSearchService(db *sql.DB, cache cache.EnhancedCache, logger *logrus.Logger, metrics *metrics.Collector) *EnhancedSearchService {
	return &EnhancedSearchService{
		db:      db,
		cache:   cache,
		logger:  logger,
		metrics: metrics,
	}
}

// Search performs comprehensive medical terminology search
func (s *EnhancedSearchService) Search(searchTerm string, options SearchOptions) (*SearchResponse, error) {
	start := time.Now()
	
	// Validate and set defaults
	if options.MaxResults <= 0 {
		options.MaxResults = 50
	}
	if options.MaxResults > 1000 {
		options.MaxResults = 1000
	}
	
	// Check cache first
	cacheKey := s.generateCacheKey(searchTerm, options)
	if cached, err := s.getCachedSearch(cacheKey); err == nil {
		s.metrics.RecordCacheHit("enhanced_search", "search_result")
		cached.SearchDurationMs = float64(time.Since(start).Nanoseconds()) / 1e6
		cached.UsedCache = true
		return cached, nil
	}
	s.metrics.RecordCacheMiss("enhanced_search", "search_result")
	
	// Perform search
	results, expandedTerms, err := s.performSearch(searchTerm, options)
	if err != nil {
		s.recordSearchStatistics(searchTerm, options, 0, time.Since(start), false)
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	// Create response
	response := &SearchResponse{
		Results:          results,
		TotalCount:       len(results),
		SearchTerm:       searchTerm,
		ExpandedTerms:    expandedTerms,
		SearchDurationMs: float64(time.Since(start).Nanoseconds()) / 1e6,
		UsedCache:        false,
		SearchOptions:    options,
	}
	
	// Cache the result
	s.cacheSearch(cacheKey, response)
	
	// Record statistics
	s.recordSearchStatistics(searchTerm, options, len(results), time.Since(start), false)
	
	return response, nil
}

// performSearch executes the actual database search
func (s *EnhancedSearchService) performSearch(searchTerm string, options SearchOptions) ([]SearchResult, []string, error) {
	// Use the comprehensive medical search function
	query := `
		SELECT 
			c.concept_uuid, c.system, c.code, c.preferred_term, c.active, 
			c.version, c.properties, c.created_at, c.updated_at,
			search.rank, search.match_type
		FROM comprehensive_medical_search($1, $2, $3, $4, $5) search
		JOIN concepts c ON c.concept_uuid = search.concept_uuid
		WHERE ($6 OR c.active = true)
		  AND (search.rank >= $7 OR $7 = 0)
		ORDER BY search.rank DESC, c.preferred_term ASC
	`
	
	var targetSystem *string
	if options.TargetSystem != "" {
		targetSystem = &options.TargetSystem
	}
	
	rows, err := s.db.Query(query, 
		searchTerm, 
		targetSystem, 
		options.ExpandSynonyms, 
		options.UsePhonetic, 
		options.MaxResults,
		options.IncludeInactive,
		options.MinRank,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	
	var results []SearchResult
	var expandedTerms []string
	
	for rows.Next() {
		var result SearchResult
		var createdAt, updatedAt sql.NullTime
		var properties sql.NullString
		
		err := rows.Scan(
			&result.ConceptUUID,
			&result.System,
			&result.Code,
			&result.PreferredTerm,
			&result.Active,
			&result.Version,
			&properties,
			&createdAt,
			&updatedAt,
			&result.Rank,
			&result.MatchType,
		)
		if err != nil {
			s.logger.Error("Failed to scan search result", logrus.WithError(err))
			continue
		}
		
		// Parse properties
		if properties.Valid {
			if err := result.Properties.UnmarshalJSON([]byte(properties.String)); err != nil {
				s.logger.Warn("Failed to parse concept properties", 
					logrus.WithError(err),
					logrus.WithField("concept_uuid", result.ConceptUUID))
			}
		}
		
		// Set timestamps
		if createdAt.Valid {
			result.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			result.UpdatedAt = updatedAt.Time
		}
		
		// Generate highlight
		result.Highlight = s.generateHighlight(result.PreferredTerm, searchTerm)
		
		results = append(results, result)
	}
	
	// Get expanded terms if synonym expansion was used
	if options.ExpandSynonyms {
		expandedTerms, err = s.getExpandedTerms(searchTerm)
		if err != nil {
			s.logger.Warn("Failed to get expanded terms", logrus.WithError(err))
		}
	}
	
	return results, expandedTerms, nil
}

// SuggestConcepts provides concept suggestions based on partial input
func (s *EnhancedSearchService) SuggestConcepts(partialTerm string, options SearchOptions) (*SearchResponse, error) {
	start := time.Now()
	
	if len(partialTerm) < 2 {
		return &SearchResponse{
			Results:          []SearchResult{},
			TotalCount:       0,
			SearchTerm:       partialTerm,
			SearchDurationMs: float64(time.Since(start).Nanoseconds()) / 1e6,
			SearchOptions:    options,
		}, nil
	}
	
	// Use prefix matching with text search
	query := `
		SELECT 
			c.concept_uuid, c.system, c.code, c.preferred_term, c.active,
			c.version, c.properties, c.created_at, c.updated_at,
			ts_rank(c.search_vector, plainto_tsquery('medical_english', $1)) as rank
		FROM concepts c
		WHERE (
			c.preferred_term ILIKE $2 OR 
			c.code ILIKE $2 OR
			c.search_vector @@ plainto_tsquery('medical_english', $1)
		)
		AND ($3::text IS NULL OR c.system = $3)
		AND ($4 OR c.active = true)
		ORDER BY 
			CASE WHEN c.preferred_term ILIKE $2 THEN 1 ELSE 2 END,
			rank DESC,
			length(c.preferred_term) ASC
		LIMIT $5
	`
	
	var targetSystem *string
	if options.TargetSystem != "" {
		targetSystem = &options.TargetSystem
	}
	
	rows, err := s.db.Query(query, 
		partialTerm,
		partialTerm+"%",
		targetSystem,
		options.IncludeInactive,
		options.MaxResults,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []SearchResult
	
	for rows.Next() {
		var result SearchResult
		var createdAt, updatedAt sql.NullTime
		var properties sql.NullString
		
		err := rows.Scan(
			&result.ConceptUUID,
			&result.System,
			&result.Code,
			&result.PreferredTerm,
			&result.Active,
			&result.Version,
			&properties,
			&createdAt,
			&updatedAt,
			&result.Rank,
		)
		if err != nil {
			s.logger.Error("Failed to scan suggestion result", logrus.WithError(err))
			continue
		}
		
		result.MatchType = "suggestion"
		result.Highlight = s.generateHighlight(result.PreferredTerm, partialTerm)
		
		// Parse properties
		if properties.Valid {
			if err := result.Properties.UnmarshalJSON([]byte(properties.String)); err != nil {
				s.logger.Warn("Failed to parse concept properties", 
					logrus.WithError(err),
					logrus.WithField("concept_uuid", result.ConceptUUID))
			}
		}
		
		// Set timestamps
		if createdAt.Valid {
			result.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			result.UpdatedAt = updatedAt.Time
		}
		
		results = append(results, result)
	}
	
	return &SearchResponse{
		Results:          results,
		TotalCount:       len(results),
		SearchTerm:       partialTerm,
		SearchDurationMs: float64(time.Since(start).Nanoseconds()) / 1e6,
		SearchOptions:    options,
	}, nil
}

// FindSimilarConcepts finds concepts similar to a given concept
func (s *EnhancedSearchService) FindSimilarConcepts(conceptUUID string, limit int) (*SearchResponse, error) {
	start := time.Now()
	
	// Get the source concept
	var sourceConcept models.Concept
	query := `SELECT concept_uuid, system, code, preferred_term, active, version, properties, created_at, updated_at FROM concepts WHERE concept_uuid = $1`
	
	row := s.db.QueryRow(query, conceptUUID)
	var createdAt, updatedAt sql.NullTime
	var properties sql.NullString
	
	err := row.Scan(
		&sourceConcept.ConceptUUID,
		&sourceConcept.System,
		&sourceConcept.Code,
		&sourceConcept.PreferredTerm,
		&sourceConcept.Active,
		&sourceConcept.Version,
		&properties,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("source concept not found: %w", err)
	}
	
	// Find similar concepts using text similarity
	similarQuery := `
		SELECT 
			c.concept_uuid, c.system, c.code, c.preferred_term, c.active,
			c.version, c.properties, c.created_at, c.updated_at,
			similarity(c.preferred_term, $1) as similarity_score
		FROM concepts c
		WHERE c.concept_uuid != $2
		  AND c.system = $3
		  AND c.active = true
		  AND similarity(c.preferred_term, $1) > 0.3
		ORDER BY similarity_score DESC
		LIMIT $4
	`
	
	rows, err := s.db.Query(similarQuery, sourceConcept.PreferredTerm, conceptUUID, sourceConcept.System, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []SearchResult
	
	for rows.Next() {
		var result SearchResult
		var createdAt, updatedAt sql.NullTime
		var properties sql.NullString
		
		err := rows.Scan(
			&result.ConceptUUID,
			&result.System,
			&result.Code,
			&result.PreferredTerm,
			&result.Active,
			&result.Version,
			&properties,
			&createdAt,
			&updatedAt,
			&result.Rank,
		)
		if err != nil {
			s.logger.Error("Failed to scan similarity result", logrus.WithError(err))
			continue
		}
		
		result.MatchType = "similarity"
		
		// Parse properties and set timestamps
		if properties.Valid {
			if err := result.Properties.UnmarshalJSON([]byte(properties.String)); err != nil {
				s.logger.Warn("Failed to parse concept properties", 
					logrus.WithError(err),
					logrus.WithField("concept_uuid", result.ConceptUUID))
			}
		}
		
		if createdAt.Valid {
			result.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			result.UpdatedAt = updatedAt.Time
		}
		
		results = append(results, result)
	}
	
	return &SearchResponse{
		Results:          results,
		TotalCount:       len(results),
		SearchTerm:       sourceConcept.PreferredTerm,
		SearchDurationMs: float64(time.Since(start).Nanoseconds()) / 1e6,
	}, nil
}

// Helper methods

func (s *EnhancedSearchService) generateCacheKey(searchTerm string, options SearchOptions) string {
	return fmt.Sprintf("search:%s:%s:%t:%t:%d:%t:%.2f", 
		searchTerm, 
		options.TargetSystem, 
		options.ExpandSynonyms, 
		options.UsePhonetic, 
		options.MaxResults,
		options.IncludeInactive,
		options.MinRank,
	)
}

func (s *EnhancedSearchService) getCachedSearch(cacheKey string) (*SearchResponse, error) {
	cached, err := s.cache.Get(cacheKey)
	if err != nil {
		return nil, err
	}
	
	if response, ok := cached.(*SearchResponse); ok {
		return response, nil
	}
	
	return nil, fmt.Errorf("invalid cached search type")
}

func (s *EnhancedSearchService) cacheSearch(cacheKey string, response *SearchResponse) {
	// Cache search results for 15 minutes
	if err := s.cache.Set(cacheKey, response, 15*time.Minute); err != nil {
		s.logger.WithError(err).Warn("Failed to cache search result")
	}
}

func (s *EnhancedSearchService) getExpandedTerms(searchTerm string) ([]string, error) {
	query := `SELECT expand_medical_synonyms($1)`
	
	row := s.db.QueryRow(query, searchTerm)
	
	var expandedArray []string
	err := row.Scan((*pq.StringArray)(&expandedArray))
	if err != nil {
		return nil, err
	}
	
	return expandedArray, nil
}

func (s *EnhancedSearchService) generateHighlight(text, searchTerm string) string {
	// Simple highlighting - wrap matched terms in <mark> tags
	lowerText := strings.ToLower(text)
	lowerSearch := strings.ToLower(searchTerm)
	
	if idx := strings.Index(lowerText, lowerSearch); idx != -1 {
		return text[:idx] + "<mark>" + text[idx:idx+len(searchTerm)] + "</mark>" + text[idx+len(searchTerm):]
	}
	
	return text
}

func (s *EnhancedSearchService) recordSearchStatistics(searchTerm string, options SearchOptions, resultCount int, duration time.Duration, usedCache bool) {
	query := `
		INSERT INTO search_statistics (
			search_term, target_system, result_count, search_duration_ms,
			used_synonyms, used_phonetic
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	durationMs := float64(duration.Nanoseconds()) / 1e6
	
	_, err := s.db.Exec(query, 
		searchTerm, 
		options.TargetSystem, 
		resultCount, 
		durationMs,
		options.ExpandSynonyms,
		options.UsePhonetic,
	)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to record search statistics")
	}
}

// SearchStatistics returns search performance statistics
func (s *EnhancedSearchService) SearchStatistics(days int) (map[string]interface{}, error) {
	if days <= 0 {
		days = 7
	}
	
	query := `
		SELECT 
			COUNT(*) as total_searches,
			COUNT(DISTINCT search_term) as unique_terms,
			AVG(search_duration_ms) as avg_duration_ms,
			AVG(result_count) as avg_result_count,
			COUNT(*) FILTER (WHERE used_synonyms = true) as synonym_searches,
			COUNT(*) FILTER (WHERE used_phonetic = true) as phonetic_searches,
			COUNT(*) FILTER (WHERE target_system IS NOT NULL) as system_specific_searches
		FROM search_statistics
		WHERE created_at >= NOW() - INTERVAL '%d days'
	`
	
	row := s.db.QueryRow(fmt.Sprintf(query, days))
	
	var stats struct {
		TotalSearches           int     `json:"total_searches"`
		UniqueTerms            int     `json:"unique_terms"`
		AvgDurationMs          float64 `json:"avg_duration_ms"`
		AvgResultCount         float64 `json:"avg_result_count"`
		SynonymSearches        int     `json:"synonym_searches"`
		PhoneticSearches       int     `json:"phonetic_searches"`
		SystemSpecificSearches int     `json:"system_specific_searches"`
	}
	
	err := row.Scan(
		&stats.TotalSearches,
		&stats.UniqueTerms,
		&stats.AvgDurationMs,
		&stats.AvgResultCount,
		&stats.SynonymSearches,
		&stats.PhoneticSearches,
		&stats.SystemSpecificSearches,
	)
	if err != nil {
		return nil, err
	}
	
	result := map[string]interface{}{
		"period_days":              days,
		"total_searches":           stats.TotalSearches,
		"unique_terms":            stats.UniqueTerms,
		"avg_duration_ms":         stats.AvgDurationMs,
		"avg_result_count":        stats.AvgResultCount,
		"synonym_usage_rate":      float64(stats.SynonymSearches) / float64(stats.TotalSearches),
		"phonetic_usage_rate":     float64(stats.PhoneticSearches) / float64(stats.TotalSearches),
		"system_specific_rate":    float64(stats.SystemSpecificSearches) / float64(stats.TotalSearches),
		"generated_at":           time.Now(),
	}
	
	return result, nil
}