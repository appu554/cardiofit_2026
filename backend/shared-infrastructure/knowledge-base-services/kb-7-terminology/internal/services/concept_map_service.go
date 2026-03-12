package services

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/models"

	"github.com/sirupsen/logrus"
)

// MappingConfidence represents the confidence level of a concept mapping
type MappingConfidence string

const (
	ConfidenceEquivalent    MappingConfidence = "equivalent"     // 1.0 - Exact match
	ConfidenceRelatedTo     MappingConfidence = "relatedto"      // 0.8 - Closely related
	ConfidenceInexact       MappingConfidence = "inexact"        // 0.6 - Similar but not exact
	ConfidenceUnmatched     MappingConfidence = "unmatched"      // 0.2 - No good match
	ConfidenceDisjoint      MappingConfidence = "disjoint"       // 0.0 - Cannot be mapped
)

// ConceptMapping represents a mapping between concepts in different code systems
type ConceptMapping struct {
	ID               string            `json:"id"`
	SourceSystem     string            `json:"source_system"`
	SourceCode       string            `json:"source_code"`
	SourceDisplay    string            `json:"source_display"`
	TargetSystem     string            `json:"target_system"`
	TargetCode       string            `json:"target_code"`
	TargetDisplay    string            `json:"target_display"`
	Equivalence      MappingConfidence `json:"equivalence"`
	ConfidenceScore  float64           `json:"confidence_score"`
	Comment          string            `json:"comment,omitempty"`
	DependsOn        []string          `json:"depends_on,omitempty"`
	Product          []ConceptMapping  `json:"product,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        *time.Time        `json:"updated_at,omitempty"`
}

// TranslationRequest represents a request to translate concepts between systems
type TranslationRequest struct {
	SourceSystem string `json:"source_system"`
	TargetSystem string `json:"target_system"`
	Code         string `json:"code"`
	Display      string `json:"display,omitempty"`
}

// BatchTranslationRequest represents a batch translation request
type BatchTranslationRequest struct {
	SourceSystem string                `json:"source_system"`
	TargetSystem string                `json:"target_system"`
	Concepts     []TranslationConcept  `json:"concepts"`
	Options      TranslationOptions    `json:"options"`
}

// TranslationConcept represents a concept to be translated
type TranslationConcept struct {
	Code    string `json:"code"`
	Display string `json:"display,omitempty"`
}

// TranslationOptions configures translation behavior
type TranslationOptions struct {
	MinConfidence    float64 `json:"min_confidence"`
	IncludeInexact   bool    `json:"include_inexact"`
	MaxResults       int     `json:"max_results"`
	IncludeHierarchy bool    `json:"include_hierarchy"`
}

// TranslationResponse represents the response from a translation operation
type TranslationResponse struct {
	SourceSystem     string           `json:"source_system"`
	TargetSystem     string           `json:"target_system"`
	SourceCode       string           `json:"source_code"`
	SourceDisplay    string           `json:"source_display"`
	Match            bool             `json:"match"`
	Mappings         []ConceptMapping `json:"mappings"`
	Message          string           `json:"message,omitempty"`
}

// BatchTranslationResponse represents a batch translation response
type BatchTranslationResponse struct {
	SourceSystem      string                `json:"source_system"`
	TargetSystem      string                `json:"target_system"`
	Results           []TranslationResponse `json:"results"`
	Summary           TranslationSummary    `json:"summary"`
	ProcessingTimeMs  float64               `json:"processing_time_ms"`
}

// TranslationSummary provides statistics about batch translation
type TranslationSummary struct {
	TotalRequests    int     `json:"total_requests"`
	SuccessfulMaps   int     `json:"successful_maps"`
	PartialMaps      int     `json:"partial_maps"`
	NoMaps           int     `json:"no_maps"`
	AverageConfidence float64 `json:"average_confidence"`
}

// ConceptMapService handles concept mapping and translation operations
type ConceptMapService struct {
	db      *sql.DB
	cache   cache.EnhancedCache
	logger  *logrus.Logger
	metrics *metrics.Collector
}

// NewConceptMapService creates a new concept mapping service
func NewConceptMapService(db *sql.DB, cache cache.EnhancedCache, logger *logrus.Logger, metrics *metrics.Collector) *ConceptMapService {
	return &ConceptMapService{
		db:      db,
		cache:   cache,
		logger:  logger,
		metrics: metrics,
	}
}

// TranslateConcept translates a single concept between terminology systems
func (s *ConceptMapService) TranslateConcept(request TranslationRequest) (*TranslationResponse, error) {
	start := time.Now()
	var status string = "success"
	defer func() {
		s.metrics.RecordTranslation(request.SourceSystem, request.TargetSystem, status, time.Since(start))
	}()

	// Check cache first
	cacheKey := fmt.Sprintf("translation:%s:%s:%s:%s", 
		request.SourceSystem, request.TargetSystem, request.Code, request.Display)
	
	if cached, err := s.getCachedTranslation(cacheKey); err == nil {
		s.metrics.RecordCacheHit("concept_map", "translation")
		return cached, nil
	}
	s.metrics.RecordCacheMiss("concept_map", "translation")

	// Get source concept details
	sourceConcept, err := s.getConceptDetails(request.SourceSystem, request.Code)
	if err != nil {
		status = "error"
		return nil, fmt.Errorf("source concept not found: %w", err)
	}

	// Find mappings
	mappings, err := s.findMappings(request.SourceSystem, request.Code, request.TargetSystem)
	if err != nil {
		status = "error"
		return nil, fmt.Errorf("failed to find mappings: %w", err)
	}

	// If no direct mappings, try semantic matching
	if len(mappings) == 0 {
		semanticMappings, err := s.findSemanticMappings(sourceConcept, request.TargetSystem)
		if err != nil {
			s.logger.WithError(err).Warn("Semantic mapping failed")
		} else {
			mappings = append(mappings, semanticMappings...)
		}
	}

	response := &TranslationResponse{
		SourceSystem:  request.SourceSystem,
		TargetSystem:  request.TargetSystem,
		SourceCode:    request.Code,
		SourceDisplay: sourceConcept.PreferredTerm,
		Match:         len(mappings) > 0,
		Mappings:      mappings,
	}

	if len(mappings) == 0 {
		response.Message = "No mappings found"
	}

	// Cache the result
	s.cacheTranslation(cacheKey, response)

	return response, nil
}

// BatchTranslateConcepts translates multiple concepts in a single operation
func (s *ConceptMapService) BatchTranslateConcepts(request BatchTranslationRequest) (*BatchTranslationResponse, error) {
	start := time.Now()
	
	if len(request.Concepts) == 0 {
		return nil, fmt.Errorf("no concepts provided for translation")
	}

	// Set default options
	if request.Options.MaxResults == 0 {
		request.Options.MaxResults = 10
	}
	if request.Options.MinConfidence == 0 {
		request.Options.MinConfidence = 0.5
	}

	results := make([]TranslationResponse, len(request.Concepts))
	
	// Process in parallel with worker pool
	maxWorkers := 10
	if len(request.Concepts) < maxWorkers {
		maxWorkers = len(request.Concepts)
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)
	
	for i, concept := range request.Concepts {
		wg.Add(1)
		go func(index int, c TranslationConcept) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			translationReq := TranslationRequest{
				SourceSystem: request.SourceSystem,
				TargetSystem: request.TargetSystem,
				Code:         c.Code,
				Display:      c.Display,
			}

			result, err := s.TranslateConcept(translationReq)
			if err != nil {
				results[index] = TranslationResponse{
					SourceSystem:  request.SourceSystem,
					TargetSystem:  request.TargetSystem,
					SourceCode:    c.Code,
					SourceDisplay: c.Display,
					Match:         false,
					Mappings:      []ConceptMapping{},
					Message:       fmt.Sprintf("Translation error: %s", err.Error()),
				}
			} else {
				// Filter by confidence if specified
				if request.Options.MinConfidence > 0 {
					filteredMappings := make([]ConceptMapping, 0)
					for _, mapping := range result.Mappings {
						if mapping.ConfidenceScore >= request.Options.MinConfidence {
							filteredMappings = append(filteredMappings, mapping)
						}
					}
					result.Mappings = filteredMappings
					result.Match = len(filteredMappings) > 0
				}

				// Limit results if specified
				if request.Options.MaxResults > 0 && len(result.Mappings) > request.Options.MaxResults {
					result.Mappings = result.Mappings[:request.Options.MaxResults]
				}

				results[index] = *result
			}
		}(i, concept)
	}

	wg.Wait()

	// Calculate summary statistics
	summary := s.calculateTranslationSummary(results)

	return &BatchTranslationResponse{
		SourceSystem:     request.SourceSystem,
		TargetSystem:     request.TargetSystem,
		Results:          results,
		Summary:          summary,
		ProcessingTimeMs: float64(time.Since(start).Nanoseconds()) / 1e6,
	}, nil
}

// CreateConceptMapping creates a new concept mapping
func (s *ConceptMapService) CreateConceptMapping(mapping ConceptMapping) error {
	// Validate the mapping
	if err := s.validateMapping(mapping); err != nil {
		return fmt.Errorf("invalid mapping: %w", err)
	}

	// Calculate confidence score if not provided
	if mapping.ConfidenceScore == 0 {
		mapping.ConfidenceScore = s.calculateConfidenceScore(mapping)
	}

	// Insert into database
	query := `
		INSERT INTO concept_mappings (
			id, source_system, source_code, source_display, 
			target_system, target_code, target_display,
			equivalence, confidence_score, comment, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (source_system, source_code, target_system, target_code)
		DO UPDATE SET 
			equivalence = $8,
			confidence_score = $9,
			comment = $10,
			updated_at = NOW()
	`

	_, err := s.db.Exec(query,
		mapping.ID,
		mapping.SourceSystem,
		mapping.SourceCode,
		mapping.SourceDisplay,
		mapping.TargetSystem,
		mapping.TargetCode,
		mapping.TargetDisplay,
		string(mapping.Equivalence),
		mapping.ConfidenceScore,
		mapping.Comment,
	)

	if err != nil {
		return fmt.Errorf("failed to create mapping: %w", err)
	}

	// Invalidate cache
	s.invalidateMappingCache(mapping.SourceSystem, mapping.TargetSystem)

	return nil
}

// GetMappingStatistics returns statistics about concept mappings
func (s *ConceptMapService) GetMappingStatistics() (map[string]interface{}, error) {
	query := `
		SELECT 
			source_system,
			target_system,
			COUNT(*) as mapping_count,
			AVG(confidence_score) as avg_confidence,
			COUNT(*) FILTER (WHERE equivalence = 'equivalent') as equivalent_count,
			COUNT(*) FILTER (WHERE equivalence = 'relatedto') as related_count,
			COUNT(*) FILTER (WHERE equivalence = 'inexact') as inexact_count
		FROM concept_mappings
		GROUP BY source_system, target_system
		ORDER BY mapping_count DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	systems := make([]map[string]interface{}, 0)
	totalMappings := 0

	for rows.Next() {
		var sourceSystem, targetSystem string
		var mappingCount, equivalentCount, relatedCount, inexactCount int
		var avgConfidence float64

		err := rows.Scan(
			&sourceSystem, &targetSystem, &mappingCount, &avgConfidence,
			&equivalentCount, &relatedCount, &inexactCount,
		)
		if err != nil {
			continue
		}

		systems = append(systems, map[string]interface{}{
			"source_system":     sourceSystem,
			"target_system":     targetSystem,
			"mapping_count":     mappingCount,
			"avg_confidence":    avgConfidence,
			"equivalent_count":  equivalentCount,
			"related_count":     relatedCount,
			"inexact_count":     inexactCount,
		})

		totalMappings += mappingCount
	}

	return map[string]interface{}{
		"total_mappings": totalMappings,
		"system_pairs":   len(systems),
		"systems":        systems,
		"generated_at":   time.Now(),
	}, nil
}

// Private helper methods

func (s *ConceptMapService) getConceptDetails(system, code string) (*models.Concept, error) {
	query := `
		SELECT concept_uuid, system, code, preferred_term, active, version, properties, created_at, updated_at
		FROM concepts
		WHERE system = $1 AND code = $2 AND active = true
		LIMIT 1
	`

	var concept models.Concept
	var createdAt, updatedAt sql.NullTime
	var properties sql.NullString

	err := s.db.QueryRow(query, system, code).Scan(
		&concept.ConceptUUID,
		&concept.System,
		&concept.Code,
		&concept.PreferredTerm,
		&concept.Active,
		&concept.Version,
		&properties,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Parse properties and timestamps
	if properties.Valid {
		if err := concept.Properties.UnmarshalJSON([]byte(properties.String)); err != nil {
			s.logger.Warn("Failed to parse concept properties", logrus.WithError(err))
		}
	}

	if createdAt.Valid {
		concept.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		concept.UpdatedAt = updatedAt.Time
	}

	return &concept, nil
}

func (s *ConceptMapService) findMappings(sourceSystem, sourceCode, targetSystem string) ([]ConceptMapping, error) {
	query := `
		SELECT 
			id, source_system, source_code, source_display,
			target_system, target_code, target_display,
			equivalence, confidence_score, comment, created_at, updated_at
		FROM concept_mappings
		WHERE source_system = $1 AND source_code = $2 AND target_system = $3
		ORDER BY confidence_score DESC
	`

	rows, err := s.db.Query(query, sourceSystem, sourceCode, targetSystem)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []ConceptMapping

	for rows.Next() {
		var mapping ConceptMapping
		var equivalence string
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&mapping.ID,
			&mapping.SourceSystem,
			&mapping.SourceCode,
			&mapping.SourceDisplay,
			&mapping.TargetSystem,
			&mapping.TargetCode,
			&mapping.TargetDisplay,
			&equivalence,
			&mapping.ConfidenceScore,
			&mapping.Comment,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			continue
		}

		mapping.Equivalence = MappingConfidence(equivalence)

		if createdAt.Valid {
			mapping.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			mapping.UpdatedAt = &updatedAt.Time
		}

		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

func (s *ConceptMapService) findSemanticMappings(sourceConcept *models.Concept, targetSystem string) ([]ConceptMapping, error) {
	// Use text similarity to find potential mappings
	query := `
		SELECT 
			c.concept_uuid, c.code, c.preferred_term,
			similarity(c.preferred_term, $1) as similarity_score
		FROM concepts c
		WHERE c.system = $2 
		  AND c.active = true
		  AND similarity(c.preferred_term, $1) > 0.6
		ORDER BY similarity_score DESC
		LIMIT 5
	`

	rows, err := s.db.Query(query, sourceConcept.PreferredTerm, targetSystem)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []ConceptMapping

	for rows.Next() {
		var conceptUUID, code, display string
		var similarity float64

		err := rows.Scan(&conceptUUID, &code, &display, &similarity)
		if err != nil {
			continue
		}

		// Determine equivalence based on similarity
		var equivalence MappingConfidence
		if similarity >= 0.95 {
			equivalence = ConfidenceEquivalent
		} else if similarity >= 0.8 {
			equivalence = ConfidenceRelatedTo
		} else {
			equivalence = ConfidenceInexact
		}

		mapping := ConceptMapping{
			ID:              fmt.Sprintf("semantic_%s_%s_%s_%s", sourceConcept.System, sourceConcept.Code, targetSystem, code),
			SourceSystem:    sourceConcept.System,
			SourceCode:      sourceConcept.Code,
			SourceDisplay:   sourceConcept.PreferredTerm,
			TargetSystem:    targetSystem,
			TargetCode:      code,
			TargetDisplay:   display,
			Equivalence:     equivalence,
			ConfidenceScore: similarity,
			Comment:         "Generated by semantic matching",
			CreatedAt:       time.Now(),
		}

		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

func (s *ConceptMapService) calculateConfidenceScore(mapping ConceptMapping) float64 {
	switch mapping.Equivalence {
	case ConfidenceEquivalent:
		return 1.0
	case ConfidenceRelatedTo:
		return 0.8
	case ConfidenceInexact:
		return 0.6
	case ConfidenceUnmatched:
		return 0.2
	case ConfidenceDisjoint:
		return 0.0
	default:
		return 0.5
	}
}

func (s *ConceptMapService) validateMapping(mapping ConceptMapping) error {
	if mapping.SourceSystem == "" || mapping.SourceCode == "" {
		return fmt.Errorf("source system and code are required")
	}
	if mapping.TargetSystem == "" || mapping.TargetCode == "" {
		return fmt.Errorf("target system and code are required")
	}
	if mapping.ConfidenceScore < 0 || mapping.ConfidenceScore > 1 {
		return fmt.Errorf("confidence score must be between 0 and 1")
	}
	return nil
}

func (s *ConceptMapService) calculateTranslationSummary(results []TranslationResponse) TranslationSummary {
	summary := TranslationSummary{
		TotalRequests: len(results),
	}

	var totalConfidence float64
	confidenceCount := 0

	for _, result := range results {
		if result.Match && len(result.Mappings) > 0 {
			if len(result.Mappings) == 1 || result.Mappings[0].ConfidenceScore >= 0.8 {
				summary.SuccessfulMaps++
			} else {
				summary.PartialMaps++
			}

			// Use highest confidence mapping for average calculation
			if result.Mappings[0].ConfidenceScore > 0 {
				totalConfidence += result.Mappings[0].ConfidenceScore
				confidenceCount++
			}
		} else {
			summary.NoMaps++
		}
	}

	if confidenceCount > 0 {
		summary.AverageConfidence = totalConfidence / float64(confidenceCount)
	}

	return summary
}

// Cache helper methods

func (s *ConceptMapService) getCachedTranslation(cacheKey string) (*TranslationResponse, error) {
	cached, err := s.cache.Get(cacheKey)
	if err != nil {
		return nil, err
	}

	if response, ok := cached.(*TranslationResponse); ok {
		return response, nil
	}

	return nil, fmt.Errorf("invalid cached translation type")
}

func (s *ConceptMapService) cacheTranslation(cacheKey string, response *TranslationResponse) {
	// Cache translation results for 1 hour
	if err := s.cache.Set(cacheKey, response, time.Hour); err != nil {
		s.logger.WithError(err).Warn("Failed to cache translation result")
	}
}

func (s *ConceptMapService) invalidateMappingCache(sourceSystem, targetSystem string) {
	// Invalidate relevant cache entries
	pattern := fmt.Sprintf("translation:%s:%s:*", sourceSystem, targetSystem)
	// Note: This would require a cache implementation that supports pattern-based invalidation
	s.logger.WithField("pattern", pattern).Debug("Cache invalidation needed")
}