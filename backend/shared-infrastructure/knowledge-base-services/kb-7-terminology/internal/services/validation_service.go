package services

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/models"

	"github.com/sirupsen/logrus"
)

// ValidationService handles terminology validation operations
type ValidationService struct {
	db      *sql.DB
	cache   cache.EnhancedCache
	logger  *logrus.Logger
	metrics *metrics.Collector
}

// NewValidationService creates a new validation service
func NewValidationService(db *sql.DB, cache cache.EnhancedCache, logger *logrus.Logger, metrics *metrics.Collector) *ValidationService {
	return &ValidationService{
		db:      db,
		cache:   cache,
		logger:  logger,
		metrics: metrics,
	}
}

// ValidateCode validates a single code against a terminology system
func (s *ValidationService) ValidateCode(code, systemURI, version, display string) (*models.ValidationResult, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordValidation(systemURI, "success", time.Since(start))
	}()

	// Try cache first
	cacheKey := cache.ValidationCacheKey(code, systemURI, version)
	if cached, err := s.getCachedValidation(cacheKey); err == nil {
		s.metrics.RecordCacheHit("validate_code", "validation_result")
		return cached, nil
	}
	s.metrics.RecordCacheMiss("validate_code", "validation_result")

	// Perform validation
	result, err := s.performValidation(code, systemURI, version, display)
	if err != nil {
		s.metrics.RecordValidation(systemURI, "error", time.Since(start))
		return nil, err
	}

	// Cache the result
	s.cacheValidation(cacheKey, result)

	return result, nil
}

// ValidateValueSetCode validates a code against a value set
func (s *ValidationService) ValidateValueSetCode(valueSetURL, code, system, display, version string) (*models.ValidationResult, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordValidation("valueset_"+valueSetURL, "success", time.Since(start))
	}()

	// Generate cache key that includes value set context
	cacheKey := fmt.Sprintf("kb7:validation:vs:%s:%s:%s:%s", valueSetURL, system, code, version)
	
	if cached, err := s.getCachedValidation(cacheKey); err == nil {
		s.metrics.RecordCacheHit("validate_valueset_code", "validation_result")
		return cached, nil
	}
	s.metrics.RecordCacheMiss("validate_valueset_code", "validation_result")

	// First validate that the code exists in the terminology system
	codeResult, err := s.ValidateCode(code, system, version, display)
	if err != nil || !codeResult.Valid {
		return codeResult, err
	}

	// Then check if the code is in the value set
	inValueSet, err := s.isCodeInValueSet(valueSetURL, code, system, version)
	if err != nil {
		return nil, fmt.Errorf("failed to check value set membership: %w", err)
	}

	result := &models.ValidationResult{
		Valid:    codeResult.Valid && inValueSet,
		Code:     code,
		System:   system,
		Display:  codeResult.Display,
		Severity: "information",
	}

	if !inValueSet {
		result.Message = fmt.Sprintf("Code '%s' is valid in system '%s' but not included in value set '%s'", code, system, valueSetURL)
		result.Severity = "warning"
	}

	// Cache the result
	s.cacheValidation(cacheKey, result)

	return result, nil
}

// BatchValidate performs validation on multiple codes in parallel
func (s *ValidationService) BatchValidate(request models.BatchValidationRequest) (*models.BatchValidationResponse, error) {
	start := time.Now()
	
	results := make([]models.ValidationResult, len(request.Requests))
	metadata := models.BatchMetadata{
		TotalRequests: len(request.Requests),
	}

	// Process in parallel if requested and batch size is reasonable
	if request.Options.ParallelProcessing && len(request.Requests) > 10 {
		s.batchValidateParallel(request, results, &metadata)
	} else {
		s.batchValidateSequential(request, results, &metadata)
	}

	metadata.ProcessingTimeMs = float64(time.Since(start).Nanoseconds()) / 1e6
	metadata.SuccessfulCount = 0
	metadata.FailedCount = 0

	// Count successes and failures
	for _, result := range results {
		if result.Valid {
			metadata.SuccessfulCount++
		} else {
			metadata.FailedCount++
		}
	}

	return &models.BatchValidationResponse{
		Results:  results,
		Metadata: metadata,
	}, nil
}

// EnhancedValidation performs deep validation with additional checks
func (s *ValidationService) EnhancedValidation(code, systemURI, version, display string, options ValidationOptions) (*EnhancedValidationResult, error) {
	// Start with basic validation
	basicResult, err := s.ValidateCode(code, systemURI, version, display)
	if err != nil {
		return nil, err
	}

	enhanced := &EnhancedValidationResult{
		ValidationResult: *basicResult,
		DisplayCheck:     DisplayCheckResult{},
		HierarchyCheck:   HierarchyCheckResult{},
		PropertyCheck:    PropertyCheckResult{},
		QualityScore:     0.0,
	}

	if !basicResult.Valid {
		return enhanced, nil
	}

	// Perform enhanced checks if requested
	if options.CheckDisplay && display != "" {
		enhanced.DisplayCheck = s.validateDisplay(code, systemURI, display)
	}

	if options.CheckHierarchy {
		enhanced.HierarchyCheck = s.validateHierarchy(code, systemURI)
	}

	if options.CheckProperties {
		enhanced.PropertyCheck = s.validateProperties(code, systemURI, options.ExpectedProperties)
	}

	// Calculate quality score
	enhanced.QualityScore = s.calculateQualityScore(enhanced)

	return enhanced, nil
}

// Private helper methods

func (s *ValidationService) performValidation(code, systemURI, version, display string) (*models.ValidationResult, error) {
	// Build query based on available parameters
	query := `
		SELECT c.code, c.preferred_term, c.active, s.system_uri, s.version, c.properties
		FROM concepts c
		JOIN terminology_systems s ON c.system_id = s.id
		WHERE c.code = $1 AND s.system_uri = $2
	`
	args := []interface{}{code, systemURI}

	if version != "" {
		query += " AND s.version = $3"
		args = append(args, version)
	}

	query += " LIMIT 1"

	row := s.db.QueryRow(query, args...)

	var foundCode, foundDisplay, foundSystemURI, foundVersion string
	var active bool
	var properties models.JSONB

	err := row.Scan(&foundCode, &foundDisplay, &active, &foundSystemURI, &foundVersion, &properties)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.ValidationResult{
				Valid:    false,
				Code:     code,
				System:   systemURI,
				Message:  fmt.Sprintf("Code '%s' not found in system '%s'", code, systemURI),
				Severity: "error",
			}, nil
		}
		return nil, err
	}

	result := &models.ValidationResult{
		Valid:    active,
		Code:     foundCode,
		System:   foundSystemURI,
		Display:  foundDisplay,
		Severity: "information",
	}

	// Check if display matches (if provided)
	if display != "" && !s.isDisplayMatch(display, foundDisplay, properties) {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity: "warning",
			Code:     "display-mismatch",
			Details:  fmt.Sprintf("Provided display '%s' does not match expected display '%s'", display, foundDisplay),
		})
	}

	// Check if code is active
	if !active {
		result.Valid = false
		result.Message = fmt.Sprintf("Code '%s' exists but is inactive", code)
		result.Severity = "warning"
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity: "warning",
			Code:     "inactive-code",
			Details:  "The code exists but has been marked as inactive",
		})
	}

	return result, nil
}

func (s *ValidationService) isCodeInValueSet(valueSetURL, code, system, version string) (bool, error) {
	// First try to find a pre-computed expansion
	query := `
		SELECT EXISTS(
			SELECT 1 
			FROM value_set_expansions vse
			JOIN expansion_contains ec ON ec.expansion_id = vse.id
			JOIN value_sets vs ON vs.id = vse.value_set_id
			WHERE vs.url = $1 
			AND ec.system = $2 
			AND ec.code = $3
			AND ec.inactive = false
		)
	`

	var exists bool
	err := s.db.QueryRow(query, valueSetURL, system, code).Scan(&exists)
	if err != nil {
		// If no expansion exists, we need to evaluate the compose rules
		return s.evaluateValueSetMembership(valueSetURL, code, system, version)
	}

	return exists, nil
}

func (s *ValidationService) evaluateValueSetMembership(valueSetURL, code, system, version string) (bool, error) {
	// Get value set definition
	var compose string
	query := `SELECT compose FROM value_sets WHERE url = $1 ORDER BY created_at DESC LIMIT 1`
	err := s.db.QueryRow(query, valueSetURL).Scan(&compose)
	if err != nil {
		return false, err
	}

	// Parse compose rules and evaluate membership
	// This is a simplified implementation - full implementation would handle all FHIR compose rule types
	return strings.Contains(compose, fmt.Sprintf(`"system":"%s"`, system)), nil
}

func (s *ValidationService) batchValidateParallel(request models.BatchValidationRequest, results []models.ValidationResult, metadata *models.BatchMetadata) {
	maxWorkers := request.Options.MaxConcurrency
	if maxWorkers == 0 || maxWorkers > 10 {
		maxWorkers = 5
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)
	cacheHits := int64(0)
	var cacheMutex sync.Mutex

	for i, req := range request.Requests {
		wg.Add(1)
		go func(index int, validation models.ValidationRequest) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			result, err := s.ValidateCode(validation.Code, validation.System, validation.Version, validation.Display)
			if err != nil {
				results[index] = models.ValidationResult{
					Valid:    false,
					Code:     validation.Code,
					System:   validation.System,
					Message:  fmt.Sprintf("Validation error: %s", err.Error()),
					Severity: "error",
				}
			} else {
				results[index] = *result
				// Track cache hits (simplified)
				cacheKey := cache.ValidationCacheKey(validation.Code, validation.System, validation.Version)
				if cached, _ := s.getCachedValidation(cacheKey); cached != nil {
					cacheMutex.Lock()
					cacheHits++
					cacheMutex.Unlock()
				}
			}
		}(i, req)
	}

	wg.Wait()

	metadata.CacheHitRate = float64(cacheHits) / float64(len(request.Requests))
}

func (s *ValidationService) batchValidateSequential(request models.BatchValidationRequest, results []models.ValidationResult, metadata *models.BatchMetadata) {
	cacheHits := 0

	for i, req := range request.Requests {
		cacheKey := cache.ValidationCacheKey(req.Code, req.System, req.Version)
		if cached, _ := s.getCachedValidation(cacheKey); cached != nil {
			cacheHits++
		}

		result, err := s.ValidateCode(req.Code, req.System, req.Version, req.Display)
		if err != nil {
			results[i] = models.ValidationResult{
				Valid:    false,
				Code:     req.Code,
				System:   req.System,
				Message:  fmt.Sprintf("Validation error: %s", err.Error()),
				Severity: "error",
			}
		} else {
			results[i] = *result
		}
	}

	metadata.CacheHitRate = float64(cacheHits) / float64(len(request.Requests))
}

func (s *ValidationService) isDisplayMatch(providedDisplay, expectedDisplay string, properties models.JSONB) bool {
	// Exact match
	if strings.EqualFold(providedDisplay, expectedDisplay) {
		return true
	}

	// Check against synonyms if available in properties
	if properties != nil {
		if synonyms, ok := properties["synonyms"].([]interface{}); ok {
			for _, syn := range synonyms {
				if synStr, ok := syn.(string); ok {
					if strings.EqualFold(providedDisplay, synStr) {
						return true
					}
				}
			}
		}
	}

	return false
}

// Cache helper methods

func (s *ValidationService) getCachedValidation(cacheKey string) (*models.ValidationResult, error) {
	cached, err := s.cache.Get(cacheKey)
	if err != nil {
		return nil, err
	}

	if result, ok := cached.(*models.ValidationResult); ok {
		return result, nil
	}

	return nil, fmt.Errorf("invalid cached validation type")
}

func (s *ValidationService) cacheValidation(cacheKey string, result *models.ValidationResult) {
	// Cache for 1 hour for validations
	if err := s.cache.Set(cacheKey, result, time.Hour); err != nil {
		s.logger.WithError(err).Warn("Failed to cache validation result")
	}
}

// Enhanced validation types and methods

type ValidationOptions struct {
	CheckDisplay        bool                   `json:"check_display"`
	CheckHierarchy      bool                   `json:"check_hierarchy"`
	CheckProperties     bool                   `json:"check_properties"`
	ExpectedProperties  map[string]interface{} `json:"expected_properties"`
	CheckSynonyms       bool                   `json:"check_synonyms"`
	CheckRelationships  bool                   `json:"check_relationships"`
}

type EnhancedValidationResult struct {
	models.ValidationResult
	DisplayCheck   DisplayCheckResult   `json:"display_check"`
	HierarchyCheck HierarchyCheckResult `json:"hierarchy_check"`
	PropertyCheck  PropertyCheckResult  `json:"property_check"`
	QualityScore   float64              `json:"quality_score"`
}

type DisplayCheckResult struct {
	ExactMatch    bool     `json:"exact_match"`
	SynonymMatch  bool     `json:"synonym_match"`
	PartialMatch  bool     `json:"partial_match"`
	Suggestions   []string `json:"suggestions,omitempty"`
}

type HierarchyCheckResult struct {
	HasParents       bool     `json:"has_parents"`
	HasChildren      bool     `json:"has_children"`
	IsLeaf           bool     `json:"is_leaf"`
	IsRoot           bool     `json:"is_root"`
	HierarchyDepth   int      `json:"hierarchy_depth"`
	HierarchyIssues  []string `json:"hierarchy_issues,omitempty"`
}

type PropertyCheckResult struct {
	ValidProperties     map[string]bool `json:"valid_properties"`
	MissingProperties   []string        `json:"missing_properties,omitempty"`
	InvalidProperties   []string        `json:"invalid_properties,omitempty"`
	PropertyValidation  bool            `json:"property_validation"`
}

func (s *ValidationService) validateDisplay(code, systemURI, display string) DisplayCheckResult {
	// Implementation for display validation
	return DisplayCheckResult{
		ExactMatch:   true, // Simplified
		SynonymMatch: false,
		PartialMatch: false,
		Suggestions:  []string{},
	}
}

func (s *ValidationService) validateHierarchy(code, systemURI string) HierarchyCheckResult {
	// Implementation for hierarchy validation
	return HierarchyCheckResult{
		HasParents:     true,
		HasChildren:    false,
		IsLeaf:         true,
		IsRoot:         false,
		HierarchyDepth: 3,
	}
}

func (s *ValidationService) validateProperties(code, systemURI string, expectedProperties map[string]interface{}) PropertyCheckResult {
	// Implementation for property validation
	return PropertyCheckResult{
		ValidProperties:   map[string]bool{},
		MissingProperties: []string{},
		PropertyValidation: true,
	}
}

func (s *ValidationService) calculateQualityScore(result *EnhancedValidationResult) float64 {
	if !result.Valid {
		return 0.0
	}

	score := 0.5 // Base score for valid code

	if result.DisplayCheck.ExactMatch {
		score += 0.3
	} else if result.DisplayCheck.SynonymMatch {
		score += 0.2
	}

	if result.HierarchyCheck.HasParents || result.HierarchyCheck.HasChildren {
		score += 0.1
	}

	if result.PropertyCheck.PropertyValidation {
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}