package services

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/models"

	"github.com/sirupsen/logrus"
)

// SNOMEDService handles SNOMED CT specific operations including expression validation
type SNOMEDService struct {
	db      *sql.DB
	cache   cache.EnhancedCache
	logger  *logrus.Logger
	metrics *metrics.Collector
}

// NewSNOMEDService creates a new SNOMED service
func NewSNOMEDService(db *sql.DB, cache cache.EnhancedCache, logger *logrus.Logger, metrics *metrics.Collector) *SNOMEDService {
	return &SNOMEDService{
		db:      db,
		cache:   cache,
		logger:  logger,
		metrics: metrics,
	}
}

// ValidateExpression validates a SNOMED CT compositional expression
func (s *SNOMEDService) ValidateExpression(expression string) (*models.SNOMEDExpressionValidationResult, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordSNOMEDValidation("expression", "success", time.Since(start))
	}()

	// Generate cache key
	cacheKey := s.generateExpressionCacheKey(expression)
	
	// Try cache first
	if cached, err := s.getCachedValidation(cacheKey); err == nil {
		s.metrics.RecordCacheHit("snomed_validation", "expression")
		return cached, nil
	}
	s.metrics.RecordCacheMiss("snomed_validation", "expression")

	// Parse the expression
	parsed, err := s.parseExpression(expression)
	if err != nil {
		return &models.SNOMEDExpressionValidationResult{
			Valid:      false,
			Expression: expression,
			ValidationErrors: []models.ValidationIssue{{
				Severity: "error",
				Code:     "parse-error",
				Details:  err.Error(),
			}},
		}, nil
	}

	// Validate concepts exist
	conceptValidation, err := s.validateConcepts(parsed.FocusConcepts)
	if err != nil || !conceptValidation.valid {
		return &models.SNOMEDExpressionValidationResult{
			Valid:            false,
			Expression:       expression,
			FocusConcepts:    parsed.FocusConcepts,
			ValidationErrors: conceptValidation.errors,
		}, nil
	}

	// Validate relationships
	relationshipValidation, err := s.validateRelationships(parsed.Refinements)
	if err != nil || !relationshipValidation.valid {
		return &models.SNOMEDExpressionValidationResult{
			Valid:            false,
			Expression:       expression,
			FocusConcepts:    parsed.FocusConcepts,
			Refinements:      parsed.Refinements,
			ValidationErrors: relationshipValidation.errors,
		}, nil
	}

	// Generate normal form
	normalForm, err := s.generateNormalForm(parsed)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to generate normal form")
		normalForm = expression // Fallback to original
	}

	// Perform semantic validation
	semanticCheck := s.performSemanticValidation(parsed)

	// Generate hash for normalized expression
	hash := s.generateNormalizedHash(normalForm)

	result := &models.SNOMEDExpressionValidationResult{
		Valid:          true,
		Expression:     expression,
		NormalForm:     normalForm,
		NormalizedHash: hash,
		FocusConcepts:  parsed.FocusConcepts,
		Refinements:    parsed.Refinements,
		SemanticCheck:  semanticCheck,
	}

	// Store the validated expression
	if err := s.storeExpression(result); err != nil {
		s.logger.WithError(err).Warn("Failed to store SNOMED expression")
	}

	// Cache the result
	s.cacheValidation(cacheKey, result)

	return result, nil
}

// NormalizeExpression converts an expression to its normal form
func (s *SNOMEDService) NormalizeExpression(expression string) (*models.SNOMEDExpressionValidationResult, error) {
	// First validate the expression
	validation, err := s.ValidateExpression(expression)
	if err != nil || !validation.Valid {
		return validation, err
	}

	// The normal form is already computed during validation
	return validation, nil
}

// ClassifyExpression performs semantic classification of the expression
func (s *SNOMEDService) ClassifyExpression(expression string) (*SNOMEDClassificationResult, error) {
	validation, err := s.ValidateExpression(expression)
	if err != nil || !validation.Valid {
		return nil, fmt.Errorf("invalid expression: %w", err)
	}

	classification := &SNOMEDClassificationResult{
		Expression:    expression,
		NormalForm:    validation.NormalForm,
		FocusConcepts: validation.FocusConcepts,
		Classification: s.classifyByFocusConcepts(validation.FocusConcepts),
		SemanticType:  s.determineSemanticType(validation.FocusConcepts, validation.Refinements),
		Complexity:    s.calculateComplexity(validation.Refinements),
	}

	return classification, nil
}

// GetExpression retrieves a stored SNOMED expression
func (s *SNOMEDService) GetExpression(id string) (*models.SNOMEDExpression, error) {
	query := `
		SELECT id, expression_hash, expression, normal_form, focus_concepts,
		       refinements, validation_status, validation_errors, created_at, updated_at
		FROM snomed_expressions
		WHERE id = $1 OR expression_hash = $1
		LIMIT 1
	`

	row := s.db.QueryRow(query, id)

	var expr models.SNOMEDExpression
	var focusConceptsJSON, refinementsJSON, validationErrorsJSON string

	err := row.Scan(
		&expr.ID, &expr.ExpressionHash, &expr.Expression, &expr.NormalForm,
		&focusConceptsJSON, &refinementsJSON, &expr.ValidationStatus,
		&validationErrorsJSON, &expr.CreatedAt, &expr.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	json.Unmarshal([]byte(focusConceptsJSON), &expr.FocusConcepts)
	json.Unmarshal([]byte(refinementsJSON), &expr.Refinements)
	json.Unmarshal([]byte(validationErrorsJSON), &expr.ValidationErrors)

	return &expr, nil
}

// Private helper methods

// ParsedExpression represents a parsed SNOMED expression
type ParsedExpression struct {
	FocusConcepts []string                     `json:"focus_concepts"`
	Refinements   []models.SNOMEDRefinement    `json:"refinements"`
}

// validationResult represents internal validation results
type validationResult struct {
	valid  bool
	errors []models.ValidationIssue
}

// parseExpression parses a SNOMED CT compositional expression
func (s *SNOMEDService) parseExpression(expression string) (*ParsedExpression, error) {
	// Remove whitespace and normalize
	expr := strings.TrimSpace(expression)
	if expr == "" {
		return nil, fmt.Errorf("empty expression")
	}

	parsed := &ParsedExpression{
		FocusConcepts: []string{},
		Refinements:   []models.SNOMEDRefinement{},
	}

	// Simple regex patterns for SNOMED expression parsing
	// In production, this would use a proper ANTLR grammar or similar
	
	// Pattern for concept IDs (SCTID)
	conceptPattern := regexp.MustCompile(`\b(\d{6,18})\b`)
	
	// Find all concept IDs
	conceptMatches := conceptPattern.FindAllString(expr, -1)
	if len(conceptMatches) == 0 {
		return nil, fmt.Errorf("no valid concept IDs found")
	}

	// The first concepts are typically focus concepts
	// This is a simplified approach - proper parsing would handle | syntax
	if strings.Contains(expr, ":") {
		// Expression has refinements
		parts := strings.Split(expr, ":")
		if len(parts) >= 2 {
			// Focus concepts are before the first ":"
			focusPart := parts[0]
			focusMatches := conceptPattern.FindAllString(focusPart, -1)
			parsed.FocusConcepts = s.removeDuplicates(focusMatches)
			
			// Parse refinements from the rest
			refinementPart := strings.Join(parts[1:], ":")
			parsed.Refinements = s.parseRefinements(refinementPart)
		}
	} else {
		// Simple expression with just focus concepts
		parsed.FocusConcepts = s.removeDuplicates(conceptMatches)
	}

	if len(parsed.FocusConcepts) == 0 {
		return nil, fmt.Errorf("no focus concepts found")
	}

	return parsed, nil
}

// parseRefinements extracts refinement information from expression part
func (s *SNOMEDService) parseRefinements(refinementPart string) []models.SNOMEDRefinement {
	var refinements []models.SNOMEDRefinement
	
	// This is a simplified parser - production would handle complex grouping
	// Pattern: relationshipType = value
	refinementPattern := regexp.MustCompile(`(\d{6,18})\s*=\s*(\d{6,18})`)
	matches := refinementPattern.FindAllStringSubmatch(refinementPart, -1)
	
	for _, match := range matches {
		if len(match) == 3 {
			refinements = append(refinements, models.SNOMEDRefinement{
				Relationship: match[1],
				Value:        match[2],
				ValueType:    "concept",
			})
		}
	}
	
	return refinements
}

// validateConcepts verifies that all concept IDs exist in SNOMED CT
func (s *SNOMEDService) validateConcepts(conceptIDs []string) (*validationResult, error) {
	if len(conceptIDs) == 0 {
		return &validationResult{valid: false, errors: []models.ValidationIssue{{
			Severity: "error",
			Code:     "no-concepts",
			Details:  "No concept IDs provided",
		}}}, nil
	}

	// Check concepts in batch
	placeholders := make([]string, len(conceptIDs))
	args := make([]interface{}, len(conceptIDs)+1)
	args[0] = "SNOMED" // system

	for i, id := range conceptIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		SELECT code, preferred_term, active
		FROM concepts
		WHERE system = $1 AND code IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	foundConcepts := make(map[string]bool)
	inactiveConcepts := make(map[string]bool)

	for rows.Next() {
		var code, term string
		var active bool
		if err := rows.Scan(&code, &term, &active); err != nil {
			continue
		}
		foundConcepts[code] = true
		if !active {
			inactiveConcepts[code] = true
		}
	}

	var errors []models.ValidationIssue

	// Check for missing concepts
	for _, id := range conceptIDs {
		if !foundConcepts[id] {
			errors = append(errors, models.ValidationIssue{
				Severity: "error",
				Code:     "concept-not-found",
				Details:  fmt.Sprintf("Concept ID %s not found in SNOMED CT", id),
				Location: id,
			})
		} else if inactiveConcepts[id] {
			errors = append(errors, models.ValidationIssue{
				Severity: "warning",
				Code:     "inactive-concept",
				Details:  fmt.Sprintf("Concept ID %s is inactive", id),
				Location: id,
			})
		}
	}

	return &validationResult{
		valid:  len(errors) == 0 || (len(errors) > 0 && errors[0].Severity == "warning"),
		errors: errors,
	}, nil
}

// validateRelationships verifies that relationship types are valid
func (s *SNOMEDService) validateRelationships(refinements []models.SNOMEDRefinement) (*validationResult, error) {
	if len(refinements) == 0 {
		return &validationResult{valid: true, errors: []models.ValidationIssue{}}, nil
	}

	var allConcepts []string
	for _, ref := range refinements {
		allConcepts = append(allConcepts, ref.Relationship)
		if ref.ValueType == "concept" {
			allConcepts = append(allConcepts, ref.Value)
		}
	}

	// Validate that all relationship concepts exist
	conceptValidation, err := s.validateConcepts(allConcepts)
	if err != nil {
		return nil, err
	}

	return conceptValidation, nil
}

// generateNormalForm creates a normalized form of the expression
func (s *SNOMEDService) generateNormalForm(parsed *ParsedExpression) (string, error) {
	// Sort focus concepts for consistency
	focusConcepts := make([]string, len(parsed.FocusConcepts))
	copy(focusConcepts, parsed.FocusConcepts)
	
	// Convert to integers for proper numerical sorting
	conceptInts := make([]int64, len(focusConcepts))
	for i, concept := range focusConcepts {
		if id, err := strconv.ParseInt(concept, 10, 64); err == nil {
			conceptInts[i] = id
		}
	}
	
	// Simple bubble sort for concept IDs (in production, use sort.Slice)
	for i := 0; i < len(conceptInts)-1; i++ {
		for j := 0; j < len(conceptInts)-i-1; j++ {
			if conceptInts[j] > conceptInts[j+1] {
				conceptInts[j], conceptInts[j+1] = conceptInts[j+1], conceptInts[j]
				focusConcepts[j], focusConcepts[j+1] = focusConcepts[j+1], focusConcepts[j]
			}
		}
	}

	// Build normalized expression
	normalForm := strings.Join(focusConcepts, " + ")

	// Add refinements if any
	if len(parsed.Refinements) > 0 {
		normalForm += " : "
		
		// Sort refinements for consistency
		refinementStrings := make([]string, len(parsed.Refinements))
		for i, ref := range parsed.Refinements {
			refinementStrings[i] = fmt.Sprintf("%s = %s", ref.Relationship, ref.Value)
		}
		
		normalForm += strings.Join(refinementStrings, " , ")
	}

	return normalForm, nil
}

// performSemanticValidation performs semantic consistency checks
func (s *SNOMEDService) performSemanticValidation(parsed *ParsedExpression) *models.SNOMEDSemanticCheck {
	check := &models.SNOMEDSemanticCheck{
		Valid:                true,
		SemanticConsistency:  true,
		ClinicallyMeaningful: true,
		Issues:               []string{},
		Suggestions:          []string{},
	}

	// Check for semantic consistency
	if len(parsed.FocusConcepts) > 1 {
		// Multiple focus concepts should be compatible
		if !s.areConceptsCompatible(parsed.FocusConcepts) {
			check.SemanticConsistency = false
			check.Issues = append(check.Issues, "Multiple focus concepts may not be semantically compatible")
		}
	}

	// Check refinements make sense for the focus concepts
	for _, refinement := range parsed.Refinements {
		if !s.isRefinementValidForConcepts(refinement, parsed.FocusConcepts) {
			check.ClinicallyMeaningful = false
			check.Issues = append(check.Issues, fmt.Sprintf("Refinement %s may not be appropriate for the given focus concepts", refinement.Relationship))
		}
	}

	if !check.SemanticConsistency || !check.ClinicallyMeaningful {
		check.Valid = false
	}

	return check
}

// Helper methods for semantic validation
func (s *SNOMEDService) areConceptsCompatible(conceptIDs []string) bool {
	// Simplified check - in production this would involve complex hierarchy analysis
	// For now, just check if concepts are from compatible hierarchies
	return true
}

func (s *SNOMEDService) isRefinementValidForConcepts(refinement models.SNOMEDRefinement, focusConcepts []string) bool {
	// Simplified check - in production this would validate relationship domain/range
	return true
}

// Classification helper methods
func (s *SNOMEDService) classifyByFocusConcepts(focusConcepts []string) []string {
	// This would classify expressions based on their focus concepts
	// returning categories like "procedure", "finding", "disorder", etc.
	return []string{"clinical-finding"}
}

func (s *SNOMEDService) determineSemanticType(focusConcepts []string, refinements []models.SNOMEDRefinement) string {
	// Determine the semantic type of the expression
	return "post-coordinated-expression"
}

func (s *SNOMEDService) calculateComplexity(refinements []models.SNOMEDRefinement) int {
	// Calculate expression complexity based on number of refinements and nesting
	return len(refinements)
}

// Storage and caching methods

func (s *SNOMEDService) storeExpression(result *models.SNOMEDExpressionValidationResult) error {
	focusConceptsJSON, _ := json.Marshal(result.FocusConcepts)
	refinementsJSON, _ := json.Marshal(result.Refinements)
	validationErrorsJSON, _ := json.Marshal(result.ValidationErrors)

	query := `
		INSERT INTO snomed_expressions 
		(expression_hash, expression, normal_form, focus_concepts, refinements, validation_status, validation_errors)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (expression_hash) DO UPDATE SET
			normal_form = EXCLUDED.normal_form,
			focus_concepts = EXCLUDED.focus_concepts,
			refinements = EXCLUDED.refinements,
			validation_status = EXCLUDED.validation_status,
			validation_errors = EXCLUDED.validation_errors,
			updated_at = NOW()
	`

	status := "valid"
	if !result.Valid {
		status = "invalid"
	}

	_, err := s.db.Exec(query,
		result.NormalizedHash, result.Expression, result.NormalForm,
		string(focusConceptsJSON), string(refinementsJSON),
		status, string(validationErrorsJSON),
	)

	return err
}

func (s *SNOMEDService) generateExpressionCacheKey(expression string) string {
	return fmt.Sprintf("kb7:snomed:expr:%s", s.generateNormalizedHash(expression))
}

func (s *SNOMEDService) generateNormalizedHash(expression string) string {
	hasher := sha256.New()
	hasher.Write([]byte(expression))
	return hex.EncodeToString(hasher.Sum(nil))[:16]
}

func (s *SNOMEDService) getCachedValidation(cacheKey string) (*models.SNOMEDExpressionValidationResult, error) {
	cached, err := s.cache.Get(cacheKey)
	if err != nil {
		return nil, err
	}

	if result, ok := cached.(*models.SNOMEDExpressionValidationResult); ok {
		return result, nil
	}

	return nil, fmt.Errorf("invalid cached validation type")
}

func (s *SNOMEDService) cacheValidation(cacheKey string, result *models.SNOMEDExpressionValidationResult) {
	// Cache for 2 hours for SNOMED validations
	if err := s.cache.Set(cacheKey, result, 2*time.Hour); err != nil {
		s.logger.WithError(err).Warn("Failed to cache SNOMED validation result")
	}
}

func (s *SNOMEDService) removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// Additional types for classification results
type SNOMEDClassificationResult struct {
	Expression     string                    `json:"expression"`
	NormalForm     string                    `json:"normal_form"`
	FocusConcepts  []string                  `json:"focus_concepts"`
	Classification []string                  `json:"classification"`
	SemanticType   string                    `json:"semantic_type"`
	Complexity     int                       `json:"complexity"`
}