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

type TerminologyService struct {
	db      *sql.DB
	cache   *cache.RedisClient
	logger  *logrus.Logger
	metrics *metrics.Collector
}

func NewTerminologyService(db *sql.DB, cache *cache.RedisClient, logger *logrus.Logger, metrics *metrics.Collector) *TerminologyService {
	return &TerminologyService{
		db:      db,
		cache:   cache,
		logger:  logger,
		metrics: metrics,
	}
}

// GetTerminologySystem retrieves a terminology system by ID or URI
func (s *TerminologyService) GetTerminologySystem(identifier string) (*models.TerminologySystem, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordDBQuery("get_terminology_system", "success", time.Since(start))
	}()

	query := `
		SELECT id, system_uri, system_name, version, description, publisher, status,
		       metadata, supported_regions, created_at, updated_at
		FROM terminology_systems 
		WHERE id = $1 OR system_uri = $1`

	row := s.db.QueryRow(query, identifier)

	var system models.TerminologySystem
	err := row.Scan(
		&system.ID, &system.SystemURI, &system.SystemName, &system.Version,
		&system.Description, &system.Publisher, &system.Status, &system.Metadata,
		&system.SupportedRegions, &system.CreatedAt, &system.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("terminology system not found: %s", identifier)
		}
		s.logger.WithError(err).Error("Failed to get terminology system")
		return nil, err
	}

	return &system, nil
}

// LookupConcept retrieves a concept by system and code
// Supports both old terminology_concepts schema and new Phase 3.5 concepts schema
func (s *TerminologyService) LookupConcept(systemIdentifier, code string) (*models.LookupResult, error) {
	start := time.Now()

	// Try cache first
	cacheKey := cache.ConceptCacheKey(systemIdentifier, code)
	var result models.LookupResult
	if err := s.cache.Get(cacheKey, &result); err == nil {
		s.metrics.RecordCacheHit("concept_lookup", "concept")
		s.metrics.RecordConceptLookup(systemIdentifier, "success", time.Since(start))
		return &result, nil
	}
	s.metrics.RecordCacheMiss("concept_lookup", "concept")

	// Get from database
	defer func() {
		s.metrics.RecordConceptLookup(systemIdentifier, "success", time.Since(start))
	}()

	// Try Phase 3.5 concepts table first (where RxNorm/SNOMED data is loaded)
	// This table uses system as a VARCHAR column directly
	// Schema: id, concept_uuid, system, code, version, preferred_term, fully_specified_name,
	//         synonyms, parent_codes, is_leaf, active, properties, designations, etc.
	phase35Query := `
		SELECT id, concept_uuid, code,
		       COALESCE(preferred_term, '') as display,
		       COALESCE(fully_specified_name, '') as definition,
		       CASE WHEN active THEN 'active' ELSE 'inactive' END as status,
		       COALESCE(parent_codes, '{}') as parent_codes,
		       '{}' as child_codes,
		       COALESCE(properties, '{}') as properties,
		       COALESCE(designations, '[]') as designations,
		       '' as clinical_domain,
		       '' as specialty,
		       created_at, updated_at
		FROM concepts
		WHERE (UPPER(system) = UPPER($1) OR system = $1) AND code = $2`

	row := s.db.QueryRow(phase35Query, systemIdentifier, code)

	var conceptUUID sql.NullString
	err := row.Scan(
		&result.Concept.ID, &conceptUUID, &result.Concept.Code,
		&result.Concept.Display, &result.Concept.Definition, &result.Concept.Status,
		pq.Array(&result.Concept.ParentCodes), pq.Array(&result.Concept.ChildCodes),
		&result.Concept.Properties, &result.Concept.Designations,
		&result.Concept.ClinicalDomain, &result.Concept.Specialty,
		&result.Concept.CreatedAt, &result.Concept.UpdatedAt,
	)

	if err == nil {
		// Found in Phase 3.5 concepts table
		s.logger.WithFields(logrus.Fields{
			"system": systemIdentifier,
			"code":   code,
			"table":  "concepts",
		}).Debug("Concept found in Phase 3.5 concepts table")

		// Cache the result
		if err := s.cache.Set(cacheKey, result, 1*time.Hour); err != nil {
			s.logger.WithError(err).Warn("Failed to cache concept lookup result")
		}
		return &result, nil
	}

	// Fallback to old terminology_concepts schema
	if err == sql.ErrNoRows {
		oldQuery := `
			SELECT c.id, c.system_id, c.code, c.display, c.definition, c.status,
			       c.parent_codes, c.child_codes, c.properties, c.designations,
			       c.clinical_domain, c.specialty, c.created_at, c.updated_at
			FROM terminology_concepts c
			JOIN terminology_systems s ON c.system_id = s.id
			WHERE (s.id::text = $1 OR s.system_uri = $1 OR LOWER(s.system_name) = LOWER($1)) AND c.code = $2`

		row = s.db.QueryRow(oldQuery, systemIdentifier, code)

		err = row.Scan(
			&result.Concept.ID, &result.Concept.SystemID, &result.Concept.Code,
			&result.Concept.Display, &result.Concept.Definition, &result.Concept.Status,
			pq.Array(&result.Concept.ParentCodes), pq.Array(&result.Concept.ChildCodes),
			&result.Concept.Properties, &result.Concept.Designations,
			&result.Concept.ClinicalDomain, &result.Concept.Specialty,
			&result.Concept.CreatedAt, &result.Concept.UpdatedAt,
		)

		if err == nil {
			s.logger.WithFields(logrus.Fields{
				"system": systemIdentifier,
				"code":   code,
				"table":  "terminology_concepts",
			}).Debug("Concept found in terminology_concepts table")

			// Cache the result
			if err := s.cache.Set(cacheKey, result, 1*time.Hour); err != nil {
				s.logger.WithError(err).Warn("Failed to cache concept lookup result")
			}
			return &result, nil
		}
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("concept not found: %s in system %s", code, systemIdentifier)
		}
		s.logger.WithError(err).Error("Failed to lookup concept")
		s.metrics.RecordConceptLookup(systemIdentifier, "error", time.Since(start))
		return nil, err
	}

	return &result, nil
}

// SearchConcepts searches for concepts based on query parameters
func (s *TerminologyService) SearchConcepts(searchQuery models.SearchQuery) (*models.SearchResult, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordSearch(searchQuery.SystemURI, "success", time.Since(start))
	}()

	// Try cache first
	cacheKey := cache.SearchCacheKey(searchQuery.Query, searchQuery.SystemURI, searchQuery.Count, searchQuery.Offset)
	var result models.SearchResult
	if err := s.cache.Get(cacheKey, &result); err == nil {
		s.metrics.RecordCacheHit("search", "search_result")
		return &result, nil
	}
	s.metrics.RecordCacheMiss("search", "search_result")

	// Build the search query
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Base query
	baseQuery := `
		FROM terminology_concepts c
		JOIN terminology_systems s ON c.system_id = s.id
		WHERE c.status = 'active'`

	conditions = append(conditions, baseQuery)

	// Add system filter if specified
	if searchQuery.SystemURI != "" {
		conditions = append(conditions, fmt.Sprintf(" AND s.system_uri = $%d", argIndex))
		args = append(args, searchQuery.SystemURI)
		argIndex++
	}

	// Add text search
	if searchQuery.Query != "" {
		searchCondition := fmt.Sprintf(`
			AND (
				c.display ILIKE $%d 
				OR c.definition ILIKE $%d
				OR c.code ILIKE $%d
			)`, argIndex, argIndex, argIndex)
		conditions = append(conditions, searchCondition)
		args = append(args, "%"+searchQuery.Query+"%")
		argIndex++
	}

	whereClause := strings.Join(conditions, " ")

	// Count query
	countQuery := "SELECT COUNT(*) " + whereClause
	var total int64
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		s.logger.WithError(err).Error("Failed to count search results")
		s.metrics.RecordSearch(searchQuery.SystemURI, "error", time.Since(start))
		return nil, err
	}

	// Main search query
	selectQuery := `
		SELECT c.id, c.system_id, c.code, c.display, c.definition, c.status,
		       c.parent_codes, c.child_codes, c.properties, c.designations,
		       c.clinical_domain, c.specialty, c.created_at, c.updated_at ` +
		whereClause + `
		ORDER BY 
			CASE WHEN c.display ILIKE $` + fmt.Sprint(len(args)) + ` THEN 1 ELSE 2 END,
			c.display
		LIMIT $` + fmt.Sprint(len(args)+1) + ` OFFSET $` + fmt.Sprint(len(args)+2)

	// Add search term for ordering (if exists)
	if searchQuery.Query != "" {
		args = append(args, searchQuery.Query+"%")
	} else {
		args = append(args, "")
	}

	// Add limit and offset
	limit := searchQuery.Count
	if limit <= 0 || limit > 100 {
		limit = 20 // Default limit
	}
	offset := searchQuery.Offset
	if offset < 0 {
		offset = 0
	}

	args = append(args, limit, offset)

	rows, err := s.db.Query(selectQuery, args...)
	if err != nil {
		s.logger.WithError(err).Error("Failed to execute search query")
		s.metrics.RecordSearch(searchQuery.SystemURI, "error", time.Since(start))
		return nil, err
	}
	defer rows.Close()

	var concepts []models.TerminologyConcept
	for rows.Next() {
		var concept models.TerminologyConcept
		err := rows.Scan(
			&concept.ID, &concept.SystemID, &concept.Code, &concept.Display,
			&concept.Definition, &concept.Status, &concept.ParentCodes,
			&concept.ChildCodes, &concept.Properties, &concept.Designations,
			&concept.ClinicalDomain, &concept.Specialty,
			&concept.CreatedAt, &concept.UpdatedAt,
		)
		if err != nil {
			s.logger.WithError(err).Error("Failed to scan search result")
			continue
		}
		concepts = append(concepts, concept)
	}

	result = models.SearchResult{
		Total:    total,
		Concepts: concepts,
	}

	// Cache the result for 30 minutes
	if err := s.cache.Set(cacheKey, result, 30*time.Minute); err != nil {
		s.logger.WithError(err).Warn("Failed to cache search result")
	}

	return &result, nil
}

// ValidateCode validates if a code exists in a specific terminology system
func (s *TerminologyService) ValidateCode(code, systemURI, version string) (*models.ValidationResult, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordValidation(systemURI, "success", time.Since(start))
	}()

	// Try cache first
	cacheKey := cache.ValidationCacheKey(code, systemURI, version)
	var result models.ValidationResult
	if err := s.cache.Get(cacheKey, &result); err == nil {
		s.metrics.RecordCacheHit("validate_code", "validation_result")
		return &result, nil
	}
	s.metrics.RecordCacheMiss("validate_code", "validation_result")

	// Query to check if code exists
	query := `
		SELECT c.code, c.display, c.status, s.system_uri, s.version
		FROM terminology_concepts c
		JOIN terminology_systems s ON c.system_id = s.id
		WHERE c.code = $1 AND s.system_uri = $2`

	var args []interface{}
	args = append(args, code, systemURI)

	if version != "" {
		query += " AND s.version = $3"
		args = append(args, version)
	}

	row := s.db.QueryRow(query, args...)

	var foundCode, display, status, foundSystemURI, foundVersion string
	err := row.Scan(&foundCode, &display, &status, &foundSystemURI, &foundVersion)

	if err != nil {
		if err == sql.ErrNoRows {
			result = models.ValidationResult{
				Valid:    false,
				Code:     code,
				System:   systemURI,
				Message:  fmt.Sprintf("Code '%s' not found in system '%s'", code, systemURI),
				Severity: "error",
			}
		} else {
			s.logger.WithError(err).Error("Failed to validate code")
			s.metrics.RecordValidation(systemURI, "error", time.Since(start))
			return nil, err
		}
	} else {
		valid := status == "active"
		result = models.ValidationResult{
			Valid:    valid,
			Code:     foundCode,
			System:   foundSystemURI,
			Display:  display,
			Severity: "information",
		}

		if !valid {
			result.Message = fmt.Sprintf("Code '%s' found but is %s", code, status)
			result.Severity = "warning"
		}
	}

	// Cache the result for 1 hour
	if err := s.cache.Set(cacheKey, result, 1*time.Hour); err != nil {
		s.logger.WithError(err).Warn("Failed to cache validation result")
	}

	return &result, nil
}

// GetValueSet retrieves a value set by URL and version
func (s *TerminologyService) GetValueSet(url, version string) (*models.ValueSet, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordDBQuery("get_value_set", "success", time.Since(start))
	}()

	// Try cache first
	cacheKey := cache.ValueSetCacheKey(url, version)
	var valueSet models.ValueSet
	if err := s.cache.Get(cacheKey, &valueSet); err == nil {
		s.metrics.RecordCacheHit("get_value_set", "value_set")
		return &valueSet, nil
	}
	s.metrics.RecordCacheMiss("get_value_set", "value_set")

	query := `
		SELECT id, url, version, name, title, description, status, publisher,
		       contact, use_context, purpose, clinical_domain, compose, expansion,
		       supported_regions, created_at, updated_at, expired_at
		FROM value_sets
		WHERE url = $1`

	var args []interface{}
	args = append(args, url)

	if version != "" {
		query += " AND version = $2"
		args = append(args, version)
	}

	query += " ORDER BY created_at DESC LIMIT 1"

	row := s.db.QueryRow(query, args...)

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
			return nil, fmt.Errorf("value set not found: %s", url)
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

// HealthCheck performs a health check of the terminology service
func (s *TerminologyService) HealthCheck() map[string]interface{} {
	health := map[string]interface{}{
		"service": "kb-7-terminology",
		"status":  "healthy",
		"checks":  make(map[string]interface{}),
	}

	// Check database connection
	err := s.db.Ping()
	if err != nil {
		health["status"] = "unhealthy"
		health["checks"].(map[string]interface{})["database"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		health["checks"].(map[string]interface{})["database"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Check cache connection
	_, err = s.cache.Exists("health_check")
	if err != nil {
		health["status"] = "unhealthy"
		health["checks"].(map[string]interface{})["cache"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		health["checks"].(map[string]interface{})["cache"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	return health
}

// RxNormSearchResult represents results from RxNorm drug name search
type RxNormSearchResult struct {
	Results []RxNormConcept `json:"results"`
	Count   int             `json:"count"`
	Query   string          `json:"query"`
}

// RxNormConcept represents an RxNorm concept from the concepts table
type RxNormConcept struct {
	RxNormCode   string   `json:"rxnorm_code"`
	Name         string   `json:"name"`
	TTY          string   `json:"tty"`           // Term Type (IN, BN, SCD, etc.)
	GenericName  string   `json:"generic_name"`
	BrandNames   []string `json:"brand_names"`
	DrugClass    string   `json:"drug_class"`
	ATCCodes     []string `json:"atc_codes"`
	NDCs         []string `json:"ndcs"`
	Ingredients  []string `json:"ingredients"`
	DoseForms    []string `json:"dose_forms"`
	Strengths    []string `json:"strengths"`
}

// ConceptRelationship represents a relationship from concept_relationships table
type ConceptRelationship struct {
	SourceCode       string  `json:"source_code"`
	TargetCode       string  `json:"target_code"`
	RelationshipType string  `json:"relationship_type"`
	RelationshipAttr *string `json:"relationship_attr,omitempty"`
}

// RelationshipsResult contains relationships for a concept
type RelationshipsResult struct {
	Code          string                `json:"code"`
	System        string                `json:"system"`
	Relationships []ConceptRelationship `json:"relationships"`
	Count         int                   `json:"count"`
}

// GetRelationships retrieves relationships for a concept from concept_relationships table
// Supports: RxNorm (1.6M), SNOMED IS-A (617K), LOINC (228K) relationships
func (s *TerminologyService) GetRelationships(system, code, relType string, limit int) (*RelationshipsResult, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordDBQuery("get_relationships", "success", time.Since(start))
	}()

	// Map system name to source_vocab value in concept_relationships table
	vocabMap := map[string]string{
		"rxnorm":    "RxNorm",
		"snomed":    "SNOMED",
		"snomed-ct": "SNOMED",
		"loinc":     "LOINC",
		"icd10":     "ICD10",
		"icd-10-cm": "ICD10",
	}

	sourceVocab, ok := vocabMap[strings.ToLower(system)]
	if !ok {
		sourceVocab = system // Use as-is if not in map
	}

	// Set default limit
	if limit <= 0 || limit > 1000 {
		limit = 500
	}

	// Query concept_relationships table
	query := `
		SELECT source_code, target_code, relationship_type, relationship_attr
		FROM concept_relationships
		WHERE source_code = $1 AND source_vocab = $2
	`
	args := []interface{}{code, sourceVocab}

	if relType != "" {
		query += " AND relationship_type = $3"
		args = append(args, relType)
	}

	query += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"system": system,
			"code":   code,
		}).Error("Failed to query concept relationships")
		return nil, fmt.Errorf("failed to query relationships: %w", err)
	}
	defer rows.Close()

	relationships := make([]ConceptRelationship, 0)
	for rows.Next() {
		var rel ConceptRelationship
		if err := rows.Scan(&rel.SourceCode, &rel.TargetCode, &rel.RelationshipType, &rel.RelationshipAttr); err != nil {
			s.logger.WithError(err).Warn("Failed to scan relationship row")
			continue
		}
		relationships = append(relationships, rel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating relationships: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"system":      system,
		"code":        code,
		"count":       len(relationships),
		"duration_ms": time.Since(start).Milliseconds(),
	}).Debug("Relationships query completed")

	return &RelationshipsResult{
		Code:          code,
		System:        system,
		Relationships: relationships,
		Count:         len(relationships),
	}, nil
}

// SearchRxNorm searches for RxNorm concepts by drug name in the Phase 3.5 concepts table
// This method is specifically designed for FDA ingestion to resolve drug names to RxNorm codes
func (s *TerminologyService) SearchRxNorm(drugName string, limit int) (*RxNormSearchResult, error) {
	if drugName == "" {
		return nil, fmt.Errorf("drug name cannot be empty")
	}

	start := time.Now()
	defer func() {
		s.metrics.RecordSearch("RxNorm", "success", time.Since(start))
	}()

	// Set default limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// Search in Phase 3.5 concepts table where RxNorm data is loaded
	// Use ILIKE for case-insensitive search and prioritize exact matches
	// Note: The concepts table has: code, preferred_term, properties (no tty column)
	query := `
		SELECT code, preferred_term
		FROM concepts
		WHERE UPPER(system) = 'RXNORM'
		  AND active = true
		  AND (
		      preferred_term ILIKE $1
		      OR preferred_term ILIKE $2
		      OR code = $3
		  )
		ORDER BY
		    CASE
		        WHEN LOWER(preferred_term) = LOWER($4) THEN 1
		        WHEN preferred_term ILIKE $1 THEN 2
		        ELSE 3
		    END,
		    LENGTH(preferred_term)
		LIMIT $5`

	// Arguments: exact prefix match, contains match, exact code match, exact name match, limit
	rows, err := s.db.Query(query, drugName+"%", "%"+drugName+"%", drugName, drugName, limit)
	if err != nil {
		s.logger.WithError(err).WithField("drug_name", drugName).Error("Failed to search RxNorm concepts")
		return nil, fmt.Errorf("failed to search RxNorm: %w", err)
	}
	defer rows.Close()

	var results []RxNormConcept
	for rows.Next() {
		var concept RxNormConcept
		err := rows.Scan(&concept.RxNormCode, &concept.Name)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan RxNorm result")
			continue
		}
		concept.GenericName = concept.Name // For simplicity, use preferred_term as generic name
		results = append(results, concept)
	}

	if err := rows.Err(); err != nil {
		s.logger.WithError(err).Error("Error iterating RxNorm search results")
		return nil, fmt.Errorf("error reading search results: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"drug_name":    drugName,
		"result_count": len(results),
		"duration_ms":  time.Since(start).Milliseconds(),
	}).Debug("RxNorm search completed")

	return &RxNormSearchResult{
		Results: results,
		Count:   len(results),
		Query:   drugName,
	}, nil
}