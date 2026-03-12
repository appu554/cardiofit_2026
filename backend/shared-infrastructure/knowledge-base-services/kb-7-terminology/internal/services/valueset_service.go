package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/models"
	"kb-7-terminology/internal/valuesets"

	"github.com/sirupsen/logrus"
)

// ============================================================================
// ValueSetService - FHIR Value Set Operations
// Manages both built-in and database-backed value sets with expansion support
// ============================================================================

// ValueSetService handles value set operations
type ValueSetService struct {
	db             *sql.DB
	cache          *cache.RedisClient
	logger         *logrus.Logger
	metrics        *metrics.Collector
	builtinSets    map[string]*valuesets.BuiltinValueSet // keyed by URL
	builtinByID    map[string]*valuesets.BuiltinValueSet // keyed by ID
	mu             sync.RWMutex
}

// ValueSetExpansion represents an expanded value set
type ValueSetExpansion struct {
	URL         string                  `json:"url"`
	Version     string                  `json:"version"`
	Name        string                  `json:"name"`
	Title       string                  `json:"title"`
	Status      string                  `json:"status"`
	Timestamp   time.Time               `json:"timestamp"`
	Total       int                     `json:"total"`
	Offset      int                     `json:"offset,omitempty"`
	Contains    []ValueSetExpansionItem `json:"contains"`
}

// ValueSetExpansionItem represents a single concept in an expansion
type ValueSetExpansionItem struct {
	System    string `json:"system"`
	Code      string `json:"code"`
	Display   string `json:"display"`
	Version   string `json:"version,omitempty"`
	Abstract  bool   `json:"abstract,omitempty"`
	Inactive  bool   `json:"inactive,omitempty"`
}

// ValueSetValidationResult represents validation against a value set
type ValueSetValidationResult struct {
	Valid        bool   `json:"valid"`
	Code         string `json:"code"`
	System       string `json:"system"`
	Display      string `json:"display,omitempty"`
	ValueSetURL  string `json:"valueset_url"`
	Message      string `json:"message,omitempty"`
}

// NewValueSetService creates a new ValueSetService (database-driven only, no hardcoded sets)
func NewValueSetService(db *sql.DB, cache *cache.RedisClient, logger *logrus.Logger, metrics *metrics.Collector) *ValueSetService {
	svc := &ValueSetService{
		db:          db,
		cache:       cache,
		logger:      logger,
		metrics:     metrics,
		builtinSets: make(map[string]*valuesets.BuiltinValueSet), // Kept for API compatibility, will be empty
		builtinByID: make(map[string]*valuesets.BuiltinValueSet), // Kept for API compatibility, will be empty
	}

	// NOTE: No longer loading hardcoded value sets
	// All value sets are now stored in PostgreSQL via RuleManager
	// Use POST /v1/rules/seed to populate database with FHIR R4 value sets
	logger.Info("ValueSetService initialized (database-driven mode - no hardcoded value sets)")

	return svc
}

// loadBuiltinValueSets pre-loads all 18 FHIR standard value sets
func (s *ValueSetService) loadBuiltinValueSets() {
	start := time.Now()

	builtins := valuesets.GetBuiltinValueSets()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, vs := range builtins {
		s.builtinSets[vs.Definition.URL] = vs
		s.builtinByID[vs.Definition.ID] = vs
	}

	s.logger.WithFields(logrus.Fields{
		"count":    len(builtins),
		"duration": time.Since(start),
	}).Info("Loaded built-in value sets")
}

// ListValueSets returns all available value sets (built-in + database)
func (s *ValueSetService) ListValueSets(ctx context.Context, includeBuiltin bool, status string, offset, limit int) (*ValueSetListResult, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordDBQuery("list_value_sets", "success", time.Since(start))
	}()

	result := &ValueSetListResult{
		ValueSets: make([]ValueSetSummary, 0),
	}

	// Add built-in value sets first
	if includeBuiltin {
		s.mu.RLock()
		for _, vs := range s.builtinSets {
			if status == "" || vs.Definition.Status == status {
				result.ValueSets = append(result.ValueSets, ValueSetSummary{
					ID:          vs.Definition.ID,
					URL:         vs.Definition.URL,
					Version:     vs.Definition.Version,
					Name:        vs.Definition.Name,
					Title:       vs.Definition.Title,
					Status:      vs.Definition.Status,
					Publisher:   vs.Definition.Publisher,
					ConceptCount: len(vs.Concepts),
					IsBuiltin:   true,
				})
			}
		}
		s.mu.RUnlock()
	}

	// Query database for custom value sets
	if s.db != nil {
		query := `
			SELECT id, url, version, name, title, status, publisher,
			       (SELECT COUNT(*) FROM value_set_concepts WHERE value_set_id = vs.id) as concept_count
			FROM value_sets vs
			WHERE ($1 = '' OR status = $1)
			ORDER BY name
			LIMIT $2 OFFSET $3`

		rows, err := s.db.QueryContext(ctx, query, status, limit, offset)
		if err != nil {
			s.logger.WithError(err).Error("Failed to list value sets from database")
			// Continue with built-in sets even if DB query fails
		} else {
			defer rows.Close()
			for rows.Next() {
				var vs ValueSetSummary
				if err := rows.Scan(&vs.ID, &vs.URL, &vs.Version, &vs.Name, &vs.Title, &vs.Status, &vs.Publisher, &vs.ConceptCount); err != nil {
					s.logger.WithError(err).Warn("Failed to scan value set row")
					continue
				}
				vs.IsBuiltin = false
				result.ValueSets = append(result.ValueSets, vs)
			}
		}
	}

	result.Total = len(result.ValueSets)
	return result, nil
}

// ValueSetListResult represents the result of listing value sets
type ValueSetListResult struct {
	Total     int               `json:"total"`
	ValueSets []ValueSetSummary `json:"value_sets"`
}

// ValueSetSummary provides a summary of a value set
type ValueSetSummary struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	Version      string `json:"version"`
	Name         string `json:"name"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	Publisher    string `json:"publisher"`
	ConceptCount int    `json:"concept_count"`
	IsBuiltin    bool   `json:"is_builtin"`
}

// GetValueSet retrieves a value set by URL or ID
func (s *ValueSetService) GetValueSet(ctx context.Context, identifier, version string) (*models.ValueSet, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordDBQuery("get_value_set", "success", time.Since(start))
	}()

	// Check cache first
	cacheKey := cache.ValueSetCacheKey(identifier, version)
	var cached models.ValueSet
	if err := s.cache.Get(cacheKey, &cached); err == nil {
		s.metrics.RecordCacheHit("get_value_set", "value_set")
		return &cached, nil
	}
	s.metrics.RecordCacheMiss("get_value_set", "value_set")

	// Check built-in value sets (for API compatibility - will be empty in production)
	s.mu.RLock()
	if vs, ok := s.builtinSets[identifier]; ok {
		s.mu.RUnlock()
		// Convert BuiltinValueSet to models.ValueSet for API compatibility
		result := &models.ValueSet{
			ID:          vs.Definition.ID,
			URL:         vs.Definition.URL,
			Version:     vs.Definition.Version,
			Name:        vs.Definition.Name,
			Title:       vs.Definition.Title,
			Description: vs.Definition.Description,
			Status:      vs.Definition.Status,
			Publisher:   vs.Definition.Publisher,
			CreatedAt:   vs.Definition.CreatedAt,
			UpdatedAt:   vs.Definition.UpdatedAt,
		}
		// Cache the result
		if err := s.cache.Set(cacheKey, result, 24*time.Hour); err != nil {
			s.logger.WithError(err).Warn("Failed to cache value set")
		}
		return result, nil
	}
	if vs, ok := s.builtinByID[identifier]; ok {
		s.mu.RUnlock()
		// Convert BuiltinValueSet to models.ValueSet for API compatibility
		result := &models.ValueSet{
			ID:          vs.Definition.ID,
			URL:         vs.Definition.URL,
			Version:     vs.Definition.Version,
			Name:        vs.Definition.Name,
			Title:       vs.Definition.Title,
			Description: vs.Definition.Description,
			Status:      vs.Definition.Status,
			Publisher:   vs.Definition.Publisher,
			CreatedAt:   vs.Definition.CreatedAt,
			UpdatedAt:   vs.Definition.UpdatedAt,
		}
		// Cache the result
		if err := s.cache.Set(cacheKey, result, 24*time.Hour); err != nil {
			s.logger.WithError(err).Warn("Failed to cache value set")
		}
		return result, nil
	}
	s.mu.RUnlock()

	// Query database
	if s.db == nil {
		return nil, fmt.Errorf("value set not found: %s", identifier)
	}

	query := `
		SELECT id, url, version, name, title, description, status, publisher,
		       contact, use_context, purpose, clinical_domain, compose, expansion,
		       supported_regions, created_at, updated_at, expired_at
		FROM value_sets
		WHERE (url = $1 OR id = $1)`

	args := []interface{}{identifier}
	if version != "" {
		query += " AND version = $2"
		args = append(args, version)
	}
	query += " ORDER BY created_at DESC LIMIT 1"

	row := s.db.QueryRowContext(ctx, query, args...)

	var valueSet models.ValueSet
	err := row.Scan(
		&valueSet.ID, &valueSet.URL, &valueSet.Version, &valueSet.Name,
		&valueSet.Title, &valueSet.Description, &valueSet.Status,
		&valueSet.Publisher, &valueSet.Contact, &valueSet.UseContext,
		&valueSet.Purpose, &valueSet.ClinicalDomain, &valueSet.Compose,
		&valueSet.Expansion, &valueSet.SupportedRegions,
		&valueSet.CreatedAt, &valueSet.UpdatedAt, &valueSet.ExpiredAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("value set not found: %s", identifier)
		}
		s.logger.WithError(err).Error("Failed to get value set")
		return nil, err
	}

	// Cache the result
	if err := s.cache.Set(cacheKey, valueSet, 2*time.Hour); err != nil {
		s.logger.WithError(err).Warn("Failed to cache value set")
	}

	return &valueSet, nil
}

// ExpandValueSet expands a value set, returning all contained concepts
func (s *ValueSetService) ExpandValueSet(ctx context.Context, identifier, version string, filter string, offset, count int) (*ValueSetExpansion, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordDBQuery("expand_value_set", "success", time.Since(start))
	}()

	// Check cache first
	cacheKey := fmt.Sprintf("vs:expand:%s:%s:%s:%d:%d", identifier, version, filter, offset, count)
	var cached ValueSetExpansion
	if err := s.cache.Get(cacheKey, &cached); err == nil {
		s.metrics.RecordCacheHit("expand_value_set", "expansion")
		return &cached, nil
	}
	s.metrics.RecordCacheMiss("expand_value_set", "expansion")

	// Check built-in value sets first
	s.mu.RLock()
	var builtinVS *valuesets.BuiltinValueSet
	if vs, ok := s.builtinSets[identifier]; ok {
		builtinVS = vs
	} else if vs, ok := s.builtinByID[identifier]; ok {
		builtinVS = vs
	}
	s.mu.RUnlock()

	if builtinVS != nil {
		expansion := s.expandBuiltinValueSet(builtinVS, filter, offset, count)
		// Cache the expansion
		if err := s.cache.Set(cacheKey, expansion, 1*time.Hour); err != nil {
			s.logger.WithError(err).Warn("Failed to cache value set expansion")
		}
		return expansion, nil
	}

	// Expand from database
	expansion, err := s.expandDatabaseValueSet(ctx, identifier, version, filter, offset, count)
	if err != nil {
		return nil, err
	}

	// Cache the expansion
	if err := s.cache.Set(cacheKey, expansion, 30*time.Minute); err != nil {
		s.logger.WithError(err).Warn("Failed to cache value set expansion")
	}

	return expansion, nil
}

// expandBuiltinValueSet expands a built-in value set
func (s *ValueSetService) expandBuiltinValueSet(vs *valuesets.BuiltinValueSet, filter string, offset, count int) *ValueSetExpansion {
	expansion := &ValueSetExpansion{
		URL:       vs.Definition.URL,
		Version:   vs.Definition.Version,
		Name:      vs.Definition.Name,
		Title:     vs.Definition.Title,
		Status:    vs.Definition.Status,
		Timestamp: time.Now(),
		Contains:  make([]ValueSetExpansionItem, 0),
		Offset:    offset,
	}

	// Filter and paginate concepts
	filterLower := strings.ToLower(filter)
	filteredConcepts := make([]valuesets.ValueSetConcept, 0)

	for _, concept := range vs.Concepts {
		if filter == "" ||
			strings.Contains(strings.ToLower(concept.Code), filterLower) ||
			strings.Contains(strings.ToLower(concept.Display), filterLower) {
			filteredConcepts = append(filteredConcepts, concept)
		}
	}

	expansion.Total = len(filteredConcepts)

	// Apply pagination
	if count <= 0 {
		count = 100 // default
	}
	endIdx := offset + count
	if endIdx > len(filteredConcepts) {
		endIdx = len(filteredConcepts)
	}
	if offset < len(filteredConcepts) {
		for _, concept := range filteredConcepts[offset:endIdx] {
			expansion.Contains = append(expansion.Contains, ValueSetExpansionItem{
				System:  concept.System,
				Code:    concept.Code,
				Display: concept.Display,
				Version: concept.Version,
			})
		}
	}

	return expansion
}

// expandDatabaseValueSet expands a value set from the database
func (s *ValueSetService) expandDatabaseValueSet(ctx context.Context, identifier, version, filter string, offset, count int) (*ValueSetExpansion, error) {
	if s.db == nil {
		return nil, fmt.Errorf("value set not found: %s", identifier)
	}

	// Get value set metadata
	vs, err := s.GetValueSet(ctx, identifier, version)
	if err != nil {
		return nil, err
	}

	expansion := &ValueSetExpansion{
		URL:       vs.URL,
		Version:   vs.Version,
		Name:      vs.Name,
		Title:     vs.Title,
		Status:    vs.Status,
		Timestamp: time.Now(),
		Contains:  make([]ValueSetExpansionItem, 0),
		Offset:    offset,
	}

	// Query concepts from database
	query := `
		SELECT system, code, display, version
		FROM value_set_concepts
		WHERE value_set_id = $1`

	args := []interface{}{vs.ID}
	argIdx := 2

	if filter != "" {
		query += fmt.Sprintf(" AND (code ILIKE $%d OR display ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+filter+"%")
		argIdx++
	}

	query += " ORDER BY code"

	if count <= 0 {
		count = 100
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, count, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		s.logger.WithError(err).Error("Failed to expand value set from database")
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item ValueSetExpansionItem
		var versionPtr *string
		if err := rows.Scan(&item.System, &item.Code, &item.Display, &versionPtr); err != nil {
			s.logger.WithError(err).Warn("Failed to scan expansion item")
			continue
		}
		if versionPtr != nil {
			item.Version = *versionPtr
		}
		expansion.Contains = append(expansion.Contains, item)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM value_set_concepts WHERE value_set_id = $1`
	countArgs := []interface{}{vs.ID}
	if filter != "" {
		countQuery += " AND (code ILIKE $2 OR display ILIKE $2)"
		countArgs = append(countArgs, "%"+filter+"%")
	}

	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&expansion.Total); err != nil {
		s.logger.WithError(err).Warn("Failed to get expansion total count")
	}

	return expansion, nil
}

// ValidateCodeInValueSet validates if a code is a member of a value set
func (s *ValueSetService) ValidateCodeInValueSet(ctx context.Context, code, system, valueSetURL string) (*ValueSetValidationResult, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordValidation(valueSetURL, "success", time.Since(start))
	}()

	result := &ValueSetValidationResult{
		Code:        code,
		System:      system,
		ValueSetURL: valueSetURL,
	}

	// Check cache first
	cacheKey := fmt.Sprintf("vs:validate:%s:%s:%s", code, system, valueSetURL)
	var cached ValueSetValidationResult
	if err := s.cache.Get(cacheKey, &cached); err == nil {
		s.metrics.RecordCacheHit("validate_in_valueset", "validation")
		return &cached, nil
	}
	s.metrics.RecordCacheMiss("validate_in_valueset", "validation")

	// Check built-in value sets first
	s.mu.RLock()
	if vs, ok := s.builtinSets[valueSetURL]; ok {
		for _, concept := range vs.Concepts {
			if concept.Code == code && (system == "" || concept.System == system) {
				result.Valid = true
				result.Display = concept.Display
				result.System = concept.System
				s.mu.RUnlock()
				// Cache the result
				if err := s.cache.Set(cacheKey, result, 1*time.Hour); err != nil {
					s.logger.WithError(err).Warn("Failed to cache validation result")
				}
				return result, nil
			}
		}
	}
	s.mu.RUnlock()

	// Check database - try value_set_concepts first, then precomputed_valueset_codes
	if s.db != nil {
		// Resolve short ValueSet name to full URL if needed
		// e.g., "ace-inhibitors" -> "http://kb7.health/ValueSet/ace-inhibitors"
		resolvedURL := valueSetURL
		if !strings.HasPrefix(valueSetURL, "http") {
			resolvedURL = "http://kb7.health/ValueSet/" + valueSetURL
		}

		// Try 1: Check value_set_concepts table (for standard FHIR ValueSets)
		query := `
			SELECT vsc.code, vsc.system, vsc.display
			FROM value_set_concepts vsc
			JOIN value_sets vs ON vsc.value_set_id = vs.id
			WHERE vs.url = $1 AND vsc.code = $2`

		args := []interface{}{resolvedURL, code}
		if system != "" {
			query += " AND vsc.system = $3"
			args = append(args, system)
		}
		query += " LIMIT 1"

		row := s.db.QueryRowContext(ctx, query, args...)
		var foundCode, foundSystem, display string
		if err := row.Scan(&foundCode, &foundSystem, &display); err == nil {
			result.Valid = true
			result.Display = display
			result.System = foundSystem
		}

		// Try 2: Check precomputed_valueset_codes table (for RxNorm drug classes, expanded SNOMED)
		if !result.Valid {
			precomputedQuery := `
				SELECT code, code_system, display
				FROM precomputed_valueset_codes
				WHERE valueset_url = $1 AND code = $2`

			precomputedArgs := []interface{}{resolvedURL, code}
			if system != "" {
				precomputedQuery += " AND code_system = $3"
				precomputedArgs = append(precomputedArgs, system)
			}
			precomputedQuery += " LIMIT 1"

			precomputedRow := s.db.QueryRowContext(ctx, precomputedQuery, precomputedArgs...)
			if err := precomputedRow.Scan(&foundCode, &foundSystem, &display); err == nil {
				result.Valid = true
				result.Display = display
				result.System = foundSystem
				s.logger.WithFields(logrus.Fields{
					"code":        code,
					"valueset":    resolvedURL,
					"source":      "precomputed_valueset_codes",
				}).Debug("Found code in precomputed valueset")
			}
		}
	}

	if !result.Valid {
		result.Message = fmt.Sprintf("Code '%s' not found in value set '%s'", code, valueSetURL)
	}

	// Cache the result
	if err := s.cache.Set(cacheKey, result, 1*time.Hour); err != nil {
		s.logger.WithError(err).Warn("Failed to cache validation result")
	}

	return result, nil
}

// GetBuiltinValueSetConcepts returns concepts for a built-in value set
func (s *ValueSetService) GetBuiltinValueSetConcepts(identifier string) ([]valuesets.ValueSetConcept, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if vs, ok := s.builtinSets[identifier]; ok {
		return vs.Concepts, true
	}
	if vs, ok := s.builtinByID[identifier]; ok {
		return vs.Concepts, true
	}
	return nil, false
}

// GetBuiltinValueSetCount returns the number of loaded built-in value sets
func (s *ValueSetService) GetBuiltinValueSetCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.builtinSets)
}

// IsBuiltinValueSet checks if a value set URL is a built-in
func (s *ValueSetService) IsBuiltinValueSet(url string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.builtinSets[url]
	return ok
}

// ExportValueSetAsJSON exports a value set as JSON (useful for FHIR compliance)
func (s *ValueSetService) ExportValueSetAsJSON(ctx context.Context, identifier, version string) ([]byte, error) {
	vs, err := s.GetValueSet(ctx, identifier, version)
	if err != nil {
		return nil, err
	}

	// Get expansion
	expansion, err := s.ExpandValueSet(ctx, identifier, version, "", 0, 1000)
	if err != nil {
		return nil, err
	}

	// Create FHIR-compliant export structure
	export := map[string]interface{}{
		"resourceType": "ValueSet",
		"id":           vs.ID,
		"url":          vs.URL,
		"version":      vs.Version,
		"name":         vs.Name,
		"title":        vs.Title,
		"status":       vs.Status,
		"publisher":    vs.Publisher,
		"description":  vs.Description,
		"expansion": map[string]interface{}{
			"identifier": fmt.Sprintf("%s|%s", vs.URL, vs.Version),
			"timestamp":  expansion.Timestamp.Format(time.RFC3339),
			"total":      expansion.Total,
			"contains":   expansion.Contains,
		},
	}

	return json.MarshalIndent(export, "", "  ")
}

// LookupCodeInBuiltinValueSets searches for a code across all built-in value sets
func (s *ValueSetService) LookupCodeInBuiltinValueSets(code, system string) []ValueSetLookupResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]ValueSetLookupResult, 0)

	for url, vs := range s.builtinSets {
		for _, concept := range vs.Concepts {
			if concept.Code == code && (system == "" || concept.System == system) {
				results = append(results, ValueSetLookupResult{
					ValueSetURL:   url,
					ValueSetTitle: vs.Definition.Title,
					Code:          concept.Code,
					System:        concept.System,
					Display:       concept.Display,
				})
			}
		}
	}

	return results
}

// ValueSetLookupResult represents a lookup result across value sets
type ValueSetLookupResult struct {
	ValueSetURL   string `json:"valueset_url"`
	ValueSetTitle string `json:"valueset_title"`
	Code          string `json:"code"`
	System        string `json:"system"`
	Display       string `json:"display"`
}
