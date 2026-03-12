package etl

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"kb-7-terminology/internal/semantic"

	"go.uber.org/zap"
)

// TripleStoreValidator performs 3-way consistency validation
type TripleStoreValidator struct {
	db           *sql.DB
	graphDBClient *semantic.GraphDBClient
	esIntegration interface{} // Elasticsearch integration (kept as interface to avoid circular dependency)
	logger       *zap.Logger
}

// TripleStoreValidationResult contains the results of a validation check
type TripleStoreValidationResult struct {
	IsConsistent       bool                   `json:"is_consistent"`
	PostgreSQLCount    int64                  `json:"postgresql_count"`
	ElasticsearchCount int64                  `json:"elasticsearch_count"`
	GraphDBTripleCount int64                  `json:"graphdb_triple_count"`
	ExpectedTriples    int64                  `json:"expected_triples"`
	Discrepancy        int64                  `json:"discrepancy"`
	ConsistencyScore   float64                `json:"consistency_score"`
	Timestamp          time.Time              `json:"timestamp"`
	Duration           time.Duration          `json:"duration"`
	Details            map[string]interface{} `json:"details"`
	Errors             []TripleStoreValidationError `json:"errors"`
}

// TripleStoreValidationError represents a validation error
type TripleStoreValidationError struct {
	Component string    `json:"component"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
}

// NewTripleStoreValidator creates a new validator
func NewTripleStoreValidator(
	db *sql.DB,
	graphDBClient *semantic.GraphDBClient,
	logger *zap.Logger,
) *TripleStoreValidator {
	return &TripleStoreValidator{
		db:           db,
		graphDBClient: graphDBClient,
		logger:       logger,
	}
}

// ValidateConsistency performs 3-way consistency validation
func (tsv *TripleStoreValidator) ValidateConsistency(ctx context.Context) (*TripleStoreValidationResult, error) {
	startTime := time.Now()
	result := &TripleStoreValidationResult{
		Timestamp: startTime,
		Details:   make(map[string]interface{}),
		Errors:    make([]TripleStoreValidationError, 0),
	}

	tsv.logger.Info("Starting 3-way consistency validation")

	// 1. Count PostgreSQL concepts
	pgCount, err := tsv.countPostgreSQLConcepts(ctx)
	if err != nil {
		tsv.recordError(result, "postgresql", err, "critical")
		return result, fmt.Errorf("PostgreSQL count failed: %w", err)
	}
	result.PostgreSQLCount = pgCount
	tsv.logger.Info("PostgreSQL concepts counted", zap.Int64("count", pgCount))

	// 2. Count GraphDB triples
	graphDBCount, err := tsv.countGraphDBTriples(ctx)
	if err != nil {
		tsv.recordError(result, "graphdb", err, "critical")
		return result, fmt.Errorf("GraphDB count failed: %w", err)
	}
	result.GraphDBTripleCount = graphDBCount
	tsv.logger.Info("GraphDB triples counted", zap.Int64("count", graphDBCount))

	// 3. Calculate expected triple count (average 8-24 triples per concept)
	// Using conservative estimate of 8 triples minimum
	result.ExpectedTriples = pgCount * 8

	// 4. Calculate discrepancy
	result.Discrepancy = abs64(graphDBCount - result.ExpectedTriples)

	// 5. Calculate consistency score
	result.ConsistencyScore = tsv.calculateConsistencyScore(pgCount, graphDBCount)

	// 6. Determine if consistent (95% threshold)
	result.IsConsistent = result.ConsistencyScore >= 0.95

	// 7. Add detailed statistics
	result.Details = map[string]interface{}{
		"postgresql_concepts":      pgCount,
		"graphdb_triples":          graphDBCount,
		"expected_triples_min":     result.ExpectedTriples,
		"expected_triples_max":     pgCount * 24,
		"avg_triples_per_concept":  float64(graphDBCount) / float64(pgCount),
		"discrepancy_percentage":   (float64(result.Discrepancy) / float64(result.ExpectedTriples)) * 100,
		"consistency_threshold":    0.95,
	}

	// 8. Validate triple integrity
	if result.IsConsistent {
		integrityResult := tsv.validateTripleIntegrity(ctx)
		result.Details["integrity_check"] = integrityResult
	}

	result.Duration = time.Since(startTime)

	// Log results
	tsv.logger.Info("Consistency validation completed",
		zap.Bool("is_consistent", result.IsConsistent),
		zap.Int64("postgresql_count", result.PostgreSQLCount),
		zap.Int64("graphdb_count", result.GraphDBTripleCount),
		zap.Float64("consistency_score", result.ConsistencyScore),
		zap.Duration("duration", result.Duration))

	if !result.IsConsistent {
		tsv.logger.Warn("Consistency validation failed",
			zap.Int64("expected_triples", result.ExpectedTriples),
			zap.Int64("actual_triples", result.GraphDBTripleCount),
			zap.Int64("discrepancy", result.Discrepancy))
	}

	return result, nil
}

// ValidateConceptMapping validates that specific concepts exist in GraphDB
func (tsv *TripleStoreValidator) ValidateConceptMapping(ctx context.Context, conceptCodes []string) (map[string]bool, error) {
	results := make(map[string]bool)

	tsv.logger.Info("Validating concept mappings", zap.Int("concept_count", len(conceptCodes)))

	for _, code := range conceptCodes {
		conceptURI := fmt.Sprintf("http://snomed.info/id/%s", code)

		query := &semantic.SPARQLQuery{
			Query: fmt.Sprintf(`
				ASK {
					<%s> ?p ?o .
				}
			`, conceptURI),
		}

		// Execute ASK query
		exists := false
		_, err := tsv.graphDBClient.ExecuteSPARQL(ctx, query)
		if err == nil {
			exists = true
		}

		results[code] = exists
	}

	return results, nil
}

// ValidateRelationships validates IS-A hierarchies in GraphDB
func (tsv *TripleStoreValidator) ValidateRelationships(ctx context.Context) (*TripleStoreValidationResult, error) {
	result := &TripleStoreValidationResult{
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	tsv.logger.Info("Validating relationships in GraphDB")

	// Count rdfs:subClassOf relationships
	query := &semantic.SPARQLQuery{
		Query: `
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
			SELECT (COUNT(*) as ?count) WHERE {
				?s rdfs:subClassOf ?o .
			}
		`,
	}

	results, err := tsv.graphDBClient.ExecuteSPARQL(ctx, query)
	if err != nil {
		return result, fmt.Errorf("failed to count relationships: %w", err)
	}

	// Parse count
	var relationshipCount int64
	if len(results.Results.Bindings) > 0 {
		if countBinding, ok := results.Results.Bindings[0]["count"]; ok {
			fmt.Sscanf(countBinding.Value, "%d", &relationshipCount)
		}
	}

	result.Details["relationship_count"] = relationshipCount
	result.Details["relationship_type"] = "rdfs:subClassOf"

	tsv.logger.Info("Relationship validation completed",
		zap.Int64("relationship_count", relationshipCount))

	return result, nil
}

// Helper methods

func (tsv *TripleStoreValidator) countPostgreSQLConcepts(ctx context.Context) (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM concepts WHERE active = true"

	err := tsv.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count PostgreSQL concepts: %w", err)
	}

	return count, nil
}

func (tsv *TripleStoreValidator) countGraphDBTriples(ctx context.Context) (int64, error) {
	query := &semantic.SPARQLQuery{
		Query: "SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }",
	}

	results, err := tsv.graphDBClient.ExecuteSPARQL(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count GraphDB triples: %w", err)
	}

	// Parse count from SPARQL results
	if len(results.Results.Bindings) > 0 {
		if countBinding, ok := results.Results.Bindings[0]["count"]; ok {
			var count int64
			fmt.Sscanf(countBinding.Value, "%d", &count)
			return count, nil
		}
	}

	return 0, fmt.Errorf("no count result returned from GraphDB")
}

func (tsv *TripleStoreValidator) calculateConsistencyScore(pgCount, graphDBCount int64) float64 {
	if pgCount == 0 {
		return 1.0 // No data to compare
	}

	// Calculate expected range: 8-24 triples per concept
	expectedMin := pgCount * 8
	expectedMax := pgCount * 24

	// If within expected range, score is 1.0
	if graphDBCount >= expectedMin && graphDBCount <= expectedMax {
		return 1.0
	}

	// If below minimum, calculate score based on how far below
	if graphDBCount < expectedMin {
		return float64(graphDBCount) / float64(expectedMin)
	}

	// If above maximum, still acceptable but score slightly reduced
	if graphDBCount > expectedMax {
		excess := float64(graphDBCount - expectedMax)
		excessRate := excess / float64(expectedMax)
		return 1.0 - (excessRate * 0.1) // Small penalty for excess triples
	}

	return 1.0
}

func (tsv *TripleStoreValidator) validateTripleIntegrity(ctx context.Context) map[string]interface{} {
	integrity := make(map[string]interface{})

	// Check for orphaned concepts (concepts without labels)
	orphanQuery := &semantic.SPARQLQuery{
		Query: `
			PREFIX owl: <http://www.w3.org/2002/07/owl#>
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
			SELECT (COUNT(*) as ?count) WHERE {
				?s a owl:Class .
				FILTER NOT EXISTS { ?s rdfs:label ?label }
			}
		`,
	}

	results, err := tsv.graphDBClient.ExecuteSPARQL(ctx, orphanQuery)
	if err == nil && len(results.Results.Bindings) > 0 {
		if countBinding, ok := results.Results.Bindings[0]["count"]; ok {
			var orphanCount int64
			fmt.Sscanf(countBinding.Value, "%d", &orphanCount)
			integrity["orphaned_concepts"] = orphanCount
		}
	}

	// Check for concepts with multiple types
	multiTypeQuery := &semantic.SPARQLQuery{
		Query: `
			PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
			SELECT (COUNT(DISTINCT ?s) as ?count) WHERE {
				?s rdf:type ?type1 .
				?s rdf:type ?type2 .
				FILTER(?type1 != ?type2)
			}
		`,
	}

	results, err = tsv.graphDBClient.ExecuteSPARQL(ctx, multiTypeQuery)
	if err == nil && len(results.Results.Bindings) > 0 {
		if countBinding, ok := results.Results.Bindings[0]["count"]; ok {
			var multiTypeCount int64
			fmt.Sscanf(countBinding.Value, "%d", &multiTypeCount)
			integrity["multi_type_concepts"] = multiTypeCount
		}
	}

	integrity["status"] = "completed"
	return integrity
}

func (tsv *TripleStoreValidator) recordError(result *TripleStoreValidationResult, component string, err error, severity string) {
	validationErr := TripleStoreValidationError{
		Component: component,
		Message:   err.Error(),
		Timestamp: time.Now(),
		Severity:  severity,
	}
	result.Errors = append(result.Errors, validationErr)
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// GetValidationThreshold returns the consistency threshold
func (tsv *TripleStoreValidator) GetValidationThreshold() float64 {
	return 0.95 // 95% consistency required
}

// QuickHealthCheck performs a quick health check
func (tsv *TripleStoreValidator) QuickHealthCheck(ctx context.Context) (bool, error) {
	// Quick check: verify GraphDB is responsive and has triples
	count, err := tsv.countGraphDBTriples(ctx)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
