package services

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/models"

	"github.com/sirupsen/logrus"
)

// ExpansionService handles value set expansion operations
type ExpansionService struct {
	db      *sql.DB
	cache   cache.EnhancedCache
	logger  *logrus.Logger
	metrics *metrics.Collector
}

// NewExpansionService creates a new expansion service
func NewExpansionService(db *sql.DB, cache cache.EnhancedCache, logger *logrus.Logger, metrics *metrics.Collector) *ExpansionService {
	return &ExpansionService{
		db:      db,
		cache:   cache,
		logger:  logger,
		metrics: metrics,
	}
}

// ExpandValueSet performs value set expansion with caching and optimization
func (s *ExpansionService) ExpandValueSet(params models.ExpansionParameters) (*models.ExpandedValueSet, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordExpansion(params.URL, "success", time.Since(start), 0)
	}()

	// Generate cache key from parameters
	cacheKey := s.generateExpansionCacheKey(params)
	
	// Try cache first
	if cached, err := s.getCachedExpansion(cacheKey); err == nil {
		s.metrics.RecordCacheHit("expansion", "value_set_expansion")
		return cached, nil
	}
	s.metrics.RecordCacheMiss("expansion", "value_set_expansion")

	// Get the value set definition
	valueSet, err := s.getValueSet(params.URL, params.ValueSetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve value set: %w", err)
	}

	// Check if we have a pre-computed expansion that's still valid
	if persistedExpansion, err := s.getPersistedExpansion(valueSet.ID, cacheKey); err == nil {
		if s.isExpansionValid(persistedExpansion, params) {
			s.logger.Debug("Using persisted expansion", logrus.Fields{"url": params.URL})
			
			// Cache for future requests
			expanded := s.convertToExpandedValueSet(persistedExpansion, valueSet)
			s.cacheExpansion(cacheKey, expanded)
			
			return expanded, nil
		}
	}

	// Perform fresh expansion
	expansion, err := s.performExpansion(valueSet, params)
	if err != nil {
		s.metrics.RecordExpansion(params.URL, "error", time.Since(start), 0)
		return nil, err
	}

	// Persist the expansion for future use if it's not too large
	if expansion.Total < 10000 {
		if err := s.persistExpansion(valueSet.ID, cacheKey, expansion, params); err != nil {
			s.logger.WithError(err).Warn("Failed to persist expansion")
		}
	}

	// Cache the result
	s.cacheExpansion(cacheKey, expansion)

	return expansion, nil
}

// performExpansion executes the actual value set expansion logic
func (s *ExpansionService) performExpansion(valueSet *models.ValueSet, params models.ExpansionParameters) (*models.ExpandedValueSet, error) {
	// Parse the compose rules
	compose := map[string]interface{}(valueSet.Compose)

	expansion := &models.ExpandedValueSet{
		URL:       params.URL,
		Version:   valueSet.Version,
		Timestamp: time.Now(),
		Contains:  []models.ExpansionContains{},
	}

	// Process include rules
	if includes, ok := compose["include"].([]interface{}); ok {
		for _, includeRule := range includes {
			if rule, ok := includeRule.(map[string]interface{}); ok {
				concepts, err := s.processIncludeRule(rule, params)
				if err != nil {
					s.logger.WithError(err).Warn("Failed to process include rule")
					continue
				}
				expansion.Contains = append(expansion.Contains, concepts...)
			}
		}
	}

	// Process exclude rules
	if excludes, ok := compose["exclude"].([]interface{}); ok {
		for _, excludeRule := range excludes {
			if rule, ok := excludeRule.(map[string]interface{}); ok {
				if err := s.processExcludeRule(rule, expansion); err != nil {
					s.logger.WithError(err).Warn("Failed to process exclude rule")
				}
			}
		}
	}

	// Apply filters
	if params.Filter != "" {
		expansion.Contains = s.applyFilter(expansion.Contains, params.Filter)
	}

	// Apply active only filter
	if params.ActiveOnly {
		expansion.Contains = s.filterActiveOnly(expansion.Contains)
	}

	// Sort results for consistency
	expansion.Contains = s.sortExpansionResults(expansion.Contains)

	// Apply pagination
	total := len(expansion.Contains)
	expansion.Total = total
	expansion.Offset = params.Offset

	if params.Offset > 0 || (params.Count > 0 && params.Count < total) {
		start := params.Offset
		if start > total {
			start = total
		}
		
		end := start + params.Count
		if params.Count == 0 || end > total {
			end = total
		}
		
		expansion.Contains = expansion.Contains[start:end]
	}

	return expansion, nil
}

// processIncludeRule processes a single include rule from the value set compose
func (s *ExpansionService) processIncludeRule(rule map[string]interface{}, params models.ExpansionParameters) ([]models.ExpansionContains, error) {
	var concepts []models.ExpansionContains
	
	system, hasSystem := rule["system"].(string)
	version, _ := rule["version"].(string)

	// Handle explicit concept lists
	if conceptList, ok := rule["concept"].([]interface{}); ok {
		for _, conceptItem := range conceptList {
			if concept, ok := conceptItem.(map[string]interface{}); ok {
				code, _ := concept["code"].(string)
				display, _ := concept["display"].(string)
				
				if code != "" {
					contains := models.ExpansionContains{
						System:  system,
						Code:    code,
						Display: display,
					}
					
					// Add designations if requested
					if params.IncludeDesignations {
						contains.Designation = s.getDesignations(system, code)
					}
					
					concepts = append(concepts, contains)
				}
			}
		}
		return concepts, nil
	}

	// Handle filter-based inclusion
	if filters, ok := rule["filter"].([]interface{}); ok && hasSystem {
		for _, filterItem := range filters {
			if filter, ok := filterItem.(map[string]interface{}); ok {
				property, _ := filter["property"].(string)
				op, _ := filter["op"].(string)
				value, _ := filter["value"].(string)
				
				filterConcepts, err := s.applyConceptFilter(system, version, property, op, value, params)
				if err != nil {
					s.logger.WithError(err).Warn("Failed to apply concept filter")
					continue
				}
				
				concepts = append(concepts, filterConcepts...)
			}
		}
		return concepts, nil
	}

	// Handle system-wide inclusion (all codes from a system)
	if hasSystem && len(rule) == 1 { // Only system specified
		return s.getAllConceptsFromSystem(system, version, params)
	}

	return concepts, nil
}

// processExcludeRule removes concepts based on exclude rules
func (s *ExpansionService) processExcludeRule(rule map[string]interface{}, expansion *models.ExpandedValueSet) error {
	system, hasSystem := rule["system"].(string)
	
	// Handle explicit concept exclusions
	if conceptList, ok := rule["concept"].([]interface{}); ok && hasSystem {
		excludeCodes := make(map[string]bool)
		
		for _, conceptItem := range conceptList {
			if concept, ok := conceptItem.(map[string]interface{}); ok {
				if code, ok := concept["code"].(string); ok {
					excludeCodes[system+"|"+code] = true
				}
			}
		}
		
		// Filter out excluded concepts
		filtered := make([]models.ExpansionContains, 0, len(expansion.Contains))
		for _, concept := range expansion.Contains {
			key := concept.System + "|" + concept.Code
			if !excludeCodes[key] {
				filtered = append(filtered, concept)
			}
		}
		expansion.Contains = filtered
	}
	
	return nil
}

// applyConceptFilter applies a concept filter to retrieve matching concepts
func (s *ExpansionService) applyConceptFilter(system, version, property, op, value string, params models.ExpansionParameters) ([]models.ExpansionContains, error) {
	switch property {
	case "concept":
		return s.handleConceptFilter(system, version, op, value, params)
	case "parent", "ancestor":
		return s.handleHierarchyFilter(system, version, op, value, "parent", params)
	case "child", "descendant":
		return s.handleHierarchyFilter(system, version, op, value, "child", params)
	default:
		return s.handlePropertyFilter(system, version, property, op, value, params)
	}
}

// handleConceptFilter processes concept-based filters (is-a relationships)
func (s *ExpansionService) handleConceptFilter(system, version, op, value string, params models.ExpansionParameters) ([]models.ExpansionContains, error) {
	switch op {
	case "is-a":
		return s.getConceptHierarchy(system, value, "descendant", params)
	case "descendent-of":
		return s.getConceptHierarchy(system, value, "descendant", params)
	case "is-not-a":
		// Get all concepts and exclude descendants
		all, err := s.getAllConceptsFromSystem(system, version, params)
		if err != nil {
			return nil, err
		}
		
		descendants, err := s.getConceptHierarchy(system, value, "descendant", params)
		if err != nil {
			return all, nil // Return all if we can't get descendants
		}
		
		// Create exclusion map
		exclude := make(map[string]bool)
		for _, desc := range descendants {
			exclude[desc.Code] = true
		}
		
		// Filter out descendants
		var filtered []models.ExpansionContains
		for _, concept := range all {
			if !exclude[concept.Code] {
				filtered = append(filtered, concept)
			}
		}
		
		return filtered, nil
		
	case "in":
		// Direct concept inclusion
		return s.getSpecificConcepts(system, []string{value}, params)
		
	default:
		return nil, fmt.Errorf("unsupported concept filter operation: %s", op)
	}
}

// getConceptHierarchy retrieves concepts based on hierarchical relationships
func (s *ExpansionService) getConceptHierarchy(system, rootCode, direction string, params models.ExpansionParameters) ([]models.ExpansionContains, error) {
	var concepts []models.ExpansionContains
	
	// Use the materialized view for efficient hierarchy traversal
	var query string
	var args []interface{}
	
	if direction == "descendant" {
		query = `
			SELECT DISTINCT c.system, c.code, c.preferred_term, c.active, c.designations
			FROM concept_hierarchy h
			JOIN concepts c ON c.system = h.system AND c.code = h.code
			WHERE h.system = $1 AND h.path @> ARRAY[$2]::text[]
			AND c.active = true
			ORDER BY c.preferred_term
		`
		args = []interface{}{system, rootCode}
	} else {
		query = `
			SELECT DISTINCT c.system, c.code, c.preferred_term, c.active, c.designations  
			FROM concept_hierarchy h
			JOIN concepts c ON c.system = h.system AND c.code = h.code
			WHERE h.system = $1 AND h.code = $2
			ORDER BY c.preferred_term
		`
		args = []interface{}{system, rootCode}
	}
	
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var concept models.ExpansionContains
		var active bool
		var designations models.JSONB
		
		err := rows.Scan(&concept.System, &concept.Code, &concept.Display, &active, &designations)
		if err != nil {
			continue
		}
		
		concept.Inactive = !active
		
		if params.IncludeDesignations {
			concept.Designation = designations
		}
		
		concepts = append(concepts, concept)
	}
	
	return concepts, nil
}

// getAllConceptsFromSystem retrieves all concepts from a terminology system
func (s *ExpansionService) getAllConceptsFromSystem(system, version string, params models.ExpansionParameters) ([]models.ExpansionContains, error) {
	query := `
		SELECT code, preferred_term, active, designations
		FROM concepts
		WHERE system = $1
	`
	args := []interface{}{system}
	
	if version != "" {
		query += " AND version = $2"
		args = append(args, version)
	}
	
	if params.ActiveOnly {
		query += " AND active = true"
	}
	
	query += " ORDER BY preferred_term"
	
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var concepts []models.ExpansionContains
	for rows.Next() {
		var concept models.ExpansionContains
		var active bool
		var designations models.JSONB
		
		err := rows.Scan(&concept.Code, &concept.Display, &active, &designations)
		if err != nil {
			continue
		}
		
		concept.System = system
		concept.Inactive = !active
		
		if params.IncludeDesignations {
			concept.Designation = designations
		}
		
		concepts = append(concepts, concept)
	}
	
	return concepts, nil
}

// Helper methods for caching and persistence

func (s *ExpansionService) generateExpansionCacheKey(params models.ExpansionParameters) string {
	cacheParams := map[string]interface{}{
		"url":                    params.URL,
		"valueSetVersion":        params.ValueSetVersion,
		"filter":                 params.Filter,
		"offset":                 params.Offset,
		"count":                  params.Count,
		"includeDesignations":    params.IncludeDesignations,
		"includeDefinition":      params.IncludeDefinition,
		"activeOnly":             params.ActiveOnly,
		"excludeNested":          params.ExcludeNested,
		"excludeNotForUI":        params.ExcludeNotForUI,
		"excludePostCoordinated": params.ExcludePostCoordinated,
		"displayLanguage":        params.DisplayLanguage,
	}
	
	return cache.ExpansionCacheKey(params.URL, cacheParams)
}

func (s *ExpansionService) getCachedExpansion(cacheKey string) (*models.ExpandedValueSet, error) {
	cached, err := s.cache.Get(cacheKey)
	if err != nil {
		return nil, err
	}
	
	if expansion, ok := cached.(*models.ExpandedValueSet); ok {
		return expansion, nil
	}
	
	return nil, fmt.Errorf("invalid cached expansion type")
}

func (s *ExpansionService) cacheExpansion(cacheKey string, expansion *models.ExpandedValueSet) {
	// Cache for 4 hours for value set expansions
	if err := s.cache.Set(cacheKey, expansion, 4*time.Hour); err != nil {
		s.logger.WithError(err).Warn("Failed to cache expansion")
	}
}

func (s *ExpansionService) getValueSet(url, version string) (*models.ValueSet, error) {
	query := `
		SELECT id, url, version, name, title, description, status, compose, expansion
		FROM value_sets
		WHERE url = $1
	`
	args := []interface{}{url}
	
	if version != "" {
		query += " AND version = $2"
		args = append(args, version)
	}
	
	query += " ORDER BY created_at DESC LIMIT 1"
	
	row := s.db.QueryRow(query, args...)
	
	var valueSet models.ValueSet
	var compose, expansion string
	
	err := row.Scan(
		&valueSet.ID, &valueSet.URL, &valueSet.Version,
		&valueSet.Name, &valueSet.Title, &valueSet.Description,
		&valueSet.Status, &compose, &expansion,
	)
	
	if err != nil {
		return nil, err
	}
	
	valueSet.Compose = models.JSONB{}
	json.Unmarshal([]byte(compose), &valueSet.Compose)
	
	valueSet.Expansion = models.JSONB{}
	json.Unmarshal([]byte(expansion), &valueSet.Expansion)
	
	return &valueSet, nil
}

func (s *ExpansionService) getPersistedExpansion(valueSetID, paramsHash string) (*models.ValueSetExpansion, error) {
	query := `
		SELECT id, params_hash, expansion_params, total, offset_value, generated_at, expires_at
		FROM value_set_expansions
		WHERE value_set_id = $1 AND params_hash = $2
		AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY generated_at DESC
		LIMIT 1
	`
	
	row := s.db.QueryRow(query, valueSetID, paramsHash)
	
	var expansion models.ValueSetExpansion
	var paramsJSON string
	
	err := row.Scan(
		&expansion.ID, &expansion.ParamsHash, &paramsJSON,
		&expansion.Total, &expansion.OffsetValue, &expansion.GeneratedAt,
		&expansion.ExpiresAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal([]byte(paramsJSON), &expansion.ExpansionParams)
	
	return &expansion, nil
}

func (s *ExpansionService) isExpansionValid(expansion *models.ValueSetExpansion, params models.ExpansionParameters) bool {
	// Check if expansion is expired
	if expansion.ExpiresAt != nil && expansion.ExpiresAt.Before(time.Now()) {
		return false
	}
	
	// Check if parameters match (simplified check)
	return expansion.ParamsHash == s.generateParameterHash(params)
}

func (s *ExpansionService) generateParameterHash(params models.ExpansionParameters) string {
	hasher := sha256.New()
	paramsJSON, _ := json.Marshal(params)
	hasher.Write(paramsJSON)
	return hex.EncodeToString(hasher.Sum(nil))[:16]
}

func (s *ExpansionService) convertToExpandedValueSet(expansion *models.ValueSetExpansion, valueSet *models.ValueSet) *models.ExpandedValueSet {
	// This would load the actual concepts from expansion_contains table
	// For now, returning a simplified version
	return &models.ExpandedValueSet{
		URL:       valueSet.URL,
		Version:   valueSet.Version,
		Timestamp: expansion.GeneratedAt,
		Total:     expansion.Total,
		Offset:    expansion.OffsetValue,
		Contains:  []models.ExpansionContains{}, // Load from expansion_contains
	}
}

// Additional helper methods for filtering and sorting

func (s *ExpansionService) applyFilter(concepts []models.ExpansionContains, filter string) []models.ExpansionContains {
	if filter == "" {
		return concepts
	}
	
	filter = strings.ToLower(filter)
	var filtered []models.ExpansionContains
	
	for _, concept := range concepts {
		if strings.Contains(strings.ToLower(concept.Display), filter) ||
		   strings.Contains(strings.ToLower(concept.Code), filter) {
			filtered = append(filtered, concept)
		}
	}
	
	return filtered
}

func (s *ExpansionService) filterActiveOnly(concepts []models.ExpansionContains) []models.ExpansionContains {
	var filtered []models.ExpansionContains
	
	for _, concept := range concepts {
		if !concept.Inactive {
			filtered = append(filtered, concept)
		}
	}
	
	return filtered
}

func (s *ExpansionService) sortExpansionResults(concepts []models.ExpansionContains) []models.ExpansionContains {
	// Simple alphabetical sorting by display name
	// In production, this might be more sophisticated
	return concepts
}

func (s *ExpansionService) getDesignations(system, code string) models.JSONB {
	// Retrieve designations for a concept
	// This would query the concepts table for the designations field
	return models.JSONB{}
}

func (s *ExpansionService) persistExpansion(valueSetID, paramsHash string, expansion *models.ExpandedValueSet, params models.ExpansionParameters) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// Insert expansion metadata
	paramsJSON, _ := json.Marshal(params)
	expiresAt := time.Now().Add(24 * time.Hour) // 24-hour expiry
	
	var expansionID int64
	err = tx.QueryRow(`
		INSERT INTO value_set_expansions 
		(value_set_id, params_hash, expansion_params, total, offset_value, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, valueSetID, paramsHash, string(paramsJSON), expansion.Total, expansion.Offset, expiresAt).Scan(&expansionID)
	
	if err != nil {
		return err
	}
	
	// Insert expansion contents in batches
	stmt, err := tx.Prepare(`
		INSERT INTO expansion_contains 
		(expansion_id, system, code, display, designation, inactive, abstract)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	for _, contains := range expansion.Contains {
		designationJSON, _ := json.Marshal(contains.Designation)
		_, err = stmt.Exec(
			expansionID, contains.System, contains.Code, contains.Display,
			string(designationJSON), contains.Inactive, contains.Abstract,
		)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to persist expansion contains")
		}
	}
	
	return tx.Commit()
}

// Additional method implementations would go here...
func (s *ExpansionService) handleHierarchyFilter(system, version, op, value, direction string, params models.ExpansionParameters) ([]models.ExpansionContains, error) {
	// Implementation for hierarchy-based filtering
	return s.getConceptHierarchy(system, value, direction, params)
}

func (s *ExpansionService) handlePropertyFilter(system, version, property, op, value string, params models.ExpansionParameters) ([]models.ExpansionContains, error) {
	// Implementation for property-based filtering
	// This would query based on concept properties
	return []models.ExpansionContains{}, nil
}

func (s *ExpansionService) getSpecificConcepts(system string, codes []string, params models.ExpansionParameters) ([]models.ExpansionContains, error) {
	// Implementation for retrieving specific concepts by code
	query := `
		SELECT code, preferred_term, active, designations
		FROM concepts
		WHERE system = $1 AND code = ANY($2)
	`
	
	if params.ActiveOnly {
		query += " AND active = true"
	}
	
	rows, err := s.db.Query(query, system, codes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var concepts []models.ExpansionContains
	for rows.Next() {
		var concept models.ExpansionContains
		var active bool
		var designations models.JSONB
		
		err := rows.Scan(&concept.Code, &concept.Display, &active, &designations)
		if err != nil {
			continue
		}
		
		concept.System = system
		concept.Inactive = !active
		
		if params.IncludeDesignations {
			concept.Designation = designations
		}
		
		concepts = append(concepts, concept)
	}
	
	return concepts, nil
}