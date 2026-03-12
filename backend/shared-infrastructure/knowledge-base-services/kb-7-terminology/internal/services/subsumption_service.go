package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/models"
	"kb-7-terminology/internal/semantic"

	"github.com/sirupsen/logrus"
)

// ============================================================================
// SubsumptionService - OWL Reasoning and Concept Hierarchy Testing
// Uses GraphDB SPARQL queries with RDFS/OWL inference for subsumption testing
// ============================================================================

// SubsumptionService handles subsumption testing operations
type SubsumptionService struct {
	graphDB *semantic.GraphDBClient
	cache   *cache.RedisClient
	logger  *logrus.Logger
	config  models.OWLReasoningConfig
}

// NewSubsumptionService creates a new SubsumptionService
func NewSubsumptionService(graphDB *semantic.GraphDBClient, cache *cache.RedisClient, logger *logrus.Logger) *SubsumptionService {
	return &SubsumptionService{
		graphDB: graphDB,
		cache:   cache,
		logger:  logger,
		config:  models.DefaultOWLReasoningConfig(),
	}
}

// SetReasoningConfig updates the OWL reasoning configuration
func (s *SubsumptionService) SetReasoningConfig(config models.OWLReasoningConfig) {
	s.config = config
}

// TestSubsumption tests if CodeA is subsumed by CodeB (A is-a B)
func (s *SubsumptionService) TestSubsumption(ctx context.Context, req *models.SubsumptionRequest) (*models.SubsumptionResult, error) {
	start := time.Now()

	result := &models.SubsumptionResult{
		CodeA:         req.CodeA,
		CodeB:         req.CodeB,
		System:        req.System,
		Relationship:  models.RelationshipUnknown,
		ReasoningType: "owl",
		TestedAt:      time.Now(),
	}

	// Check cache first
	cacheKey := fmt.Sprintf("subsumption:%s:%s:%s", req.System, req.CodeA, req.CodeB)
	var cached models.SubsumptionResult
	if s.cache != nil {
		if err := s.cache.Get(cacheKey, &cached); err == nil {
			cached.CachedResult = true
			cached.ExecutionTime = float64(time.Since(start).Microseconds()) / 1000.0
			return &cached, nil
		}
	}

	// Check if GraphDB is available
	if s.graphDB == nil {
		return nil, fmt.Errorf("GraphDB client not available for subsumption testing")
	}

	// Build system URI namespace
	systemNS := s.getSystemNamespace(req.System)

	// First check for equivalence
	if req.CodeA == req.CodeB {
		result.Subsumes = true
		result.Relationship = models.RelationshipEquivalent
		result.PathLength = 0
		result.ExecutionTime = float64(time.Since(start).Microseconds()) / 1000.0
		s.cacheResult(cacheKey, result)
		return result, nil
	}

	// Test if A is subsumed by B (A rdfs:subClassOf B)
	subsumedBy, pathLength, err := s.testSubsumptionQuery(ctx, req.CodeA, req.CodeB, systemNS)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"code_a": req.CodeA,
			"code_b": req.CodeB,
			"system": req.System,
		}).Error("Failed to test subsumption")
		return nil, err
	}

	if subsumedBy {
		result.Subsumes = true
		result.Relationship = models.RelationshipSubsumedBy
		result.PathLength = pathLength
	} else {
		// Test reverse: if B is subsumed by A (B rdfs:subClassOf A)
		subsumes, reversePath, err := s.testSubsumptionQuery(ctx, req.CodeB, req.CodeA, systemNS)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to test reverse subsumption")
		} else if subsumes {
			result.Subsumes = false
			result.Relationship = models.RelationshipSubsumes
			result.PathLength = reversePath
		} else {
			result.Subsumes = false
			result.Relationship = models.RelationshipNotSubsumed
		}
	}

	// Get display names
	displayA, displayB := s.getDisplayNames(ctx, req.CodeA, req.CodeB, systemNS)
	result.DisplayA = displayA
	result.DisplayB = displayB

	result.ExecutionTime = float64(time.Since(start).Microseconds()) / 1000.0
	s.cacheResult(cacheKey, result)

	return result, nil
}

// testSubsumptionQuery executes a SPARQL query to test subsumption
func (s *SubsumptionService) testSubsumptionQuery(ctx context.Context, codeA, codeB, systemNS string) (bool, int, error) {
	// SPARQL query using rdfs:subClassOf* for transitive closure
	// GraphDB with RDFS inference will handle transitive reasoning
	query := fmt.Sprintf(`
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
		PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
		PREFIX sys: <%s>

		SELECT (COUNT(?mid) AS ?pathLength) WHERE {
			{
				# Direct subclass relationship
				sys:%s rdfs:subClassOf* sys:%s .
				OPTIONAL {
					sys:%s rdfs:subClassOf+ ?mid .
					?mid rdfs:subClassOf* sys:%s .
				}
			}
			UNION
			{
				# SKOS broader relationship (alternative hierarchy)
				sys:%s skos:broader* sys:%s .
				OPTIONAL {
					sys:%s skos:broader+ ?mid .
					?mid skos:broader* sys:%s .
				}
			}
		}
	`, systemNS, codeA, codeB, codeA, codeB, codeA, codeB, codeA, codeB)

	// Try a simpler ASK query first for performance
	askQuery := fmt.Sprintf(`
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
		PREFIX sys: <%s>

		ASK {
			{
				sys:%s rdfs:subClassOf+ sys:%s .
			}
			UNION
			{
				sys:%s skos:broader+ sys:%s .
			}
		}
	`, systemNS, codeA, codeB, codeA, codeB)

	sparqlQuery := &semantic.SPARQLQuery{
		Query:  askQuery,
		Format: "json",
	}

	results, err := s.graphDB.ExecuteSPARQL(ctx, sparqlQuery)
	if err != nil {
		// Fall back to alternative query format
		sparqlQuery.Query = query
		results, err = s.graphDB.ExecuteSPARQL(ctx, sparqlQuery)
		if err != nil {
			return false, 0, err
		}
	}

	// Parse ASK result
	if results.Boolean != nil && *results.Boolean {
		// Get path length with separate query
		pathLength := s.getPathLength(ctx, codeA, codeB, systemNS)
		return true, pathLength, nil
	}

	// Check SELECT results
	if len(results.Results.Bindings) > 0 {
		if pathLen, ok := results.Results.Bindings[0]["pathLength"]; ok {
			if pathLen.Value != "0" {
				return true, 1, nil // Simplified path length
			}
		}
	}

	return false, 0, nil
}

// getPathLength calculates the hierarchy path length between two concepts
func (s *SubsumptionService) getPathLength(ctx context.Context, codeA, codeB, systemNS string) int {
	query := fmt.Sprintf(`
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX sys: <%s>

		SELECT (COUNT(?mid) + 1 AS ?depth) WHERE {
			sys:%s rdfs:subClassOf+ ?mid .
			?mid rdfs:subClassOf* sys:%s .
		}
	`, systemNS, codeA, codeB)

	sparqlQuery := &semantic.SPARQLQuery{
		Query:  query,
		Format: "json",
	}

	results, err := s.graphDB.ExecuteSPARQL(ctx, sparqlQuery)
	if err != nil {
		return 1 // Default to 1 if we can't determine
	}

	if len(results.Results.Bindings) > 0 {
		if depth, ok := results.Results.Bindings[0]["depth"]; ok {
			// Parse depth value
			var d int
			fmt.Sscanf(depth.Value, "%d", &d)
			if d > 0 {
				return d
			}
		}
	}

	return 1
}

// getDisplayNames retrieves display names for the codes
func (s *SubsumptionService) getDisplayNames(ctx context.Context, codeA, codeB, systemNS string) (string, string) {
	query := fmt.Sprintf(`
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
		PREFIX sys: <%s>

		SELECT ?codeA_label ?codeB_label WHERE {
			OPTIONAL { sys:%s rdfs:label ?codeA_label }
			OPTIONAL { sys:%s skos:prefLabel ?codeA_pref }
			OPTIONAL { sys:%s rdfs:label ?codeB_label }
			OPTIONAL { sys:%s skos:prefLabel ?codeB_pref }
			BIND(COALESCE(?codeA_label, ?codeA_pref, "%s") AS ?codeA_label)
			BIND(COALESCE(?codeB_label, ?codeB_pref, "%s") AS ?codeB_label)
		} LIMIT 1
	`, systemNS, codeA, codeA, codeB, codeB, codeA, codeB)

	sparqlQuery := &semantic.SPARQLQuery{
		Query:  query,
		Format: "json",
	}

	results, err := s.graphDB.ExecuteSPARQL(ctx, sparqlQuery)
	if err != nil {
		return "", ""
	}

	var displayA, displayB string
	if len(results.Results.Bindings) > 0 {
		if label, ok := results.Results.Bindings[0]["codeA_label"]; ok {
			displayA = label.Value
		}
		if label, ok := results.Results.Bindings[0]["codeB_label"]; ok {
			displayB = label.Value
		}
	}

	return displayA, displayB
}

// GetAncestors retrieves all ancestors of a concept
func (s *SubsumptionService) GetAncestors(ctx context.Context, req *models.AncestorsRequest) (*models.AncestorsResult, error) {
	if s.graphDB == nil {
		return nil, fmt.Errorf("GraphDB client not available")
	}

	systemNS := s.getSystemNamespace(req.System)

	result := &models.AncestorsResult{
		Code:      req.Code,
		System:    req.System,
		Ancestors: make([]models.ConceptAncestor, 0),
	}

	// Check cache
	cacheKey := fmt.Sprintf("ancestors:%s:%s:%d", req.System, req.Code, req.MaxDepth)
	if s.cache != nil {
		var cached models.AncestorsResult
		if err := s.cache.Get(cacheKey, &cached); err == nil {
			return &cached, nil
		}
	}

	// Build depth limit clause
	depthClause := ""
	if req.MaxDepth > 0 {
		depthClause = fmt.Sprintf("FILTER(?depth <= %d)", req.MaxDepth)
	}

	// SPARQL query for ancestors with depth calculation
	query := fmt.Sprintf(`
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
		PREFIX sys: <%s>

		SELECT DISTINCT ?ancestor ?label ?depth ?direct WHERE {
			{
				sys:%s rdfs:subClassOf+ ?ancestor .
				BIND(EXISTS { sys:%s rdfs:subClassOf ?ancestor } AS ?direct)
			}
			UNION
			{
				sys:%s skos:broader+ ?ancestor .
				BIND(EXISTS { sys:%s skos:broader ?ancestor } AS ?direct)
			}
			OPTIONAL { ?ancestor rdfs:label ?label }
			OPTIONAL { ?ancestor skos:prefLabel ?prefLabel }

			# Calculate depth (simplified - actual depth requires recursive counting)
			BIND(IF(?direct, 1, 2) AS ?depth)
			%s
		}
		ORDER BY ?depth
		LIMIT 500
	`, systemNS, req.Code, req.Code, req.Code, req.Code, depthClause)

	sparqlQuery := &semantic.SPARQLQuery{
		Query:  query,
		Format: "json",
	}

	results, err := s.graphDB.ExecuteSPARQL(ctx, sparqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query ancestors: %w", err)
	}

	maxDepth := 0
	for _, binding := range results.Results.Bindings {
		ancestor := models.ConceptAncestor{}

		if ancestorURI, ok := binding["ancestor"]; ok {
			// Extract code from URI
			ancestor.Code = s.extractCodeFromURI(ancestorURI.Value)
		}

		if label, ok := binding["label"]; ok {
			ancestor.Display = label.Value
		} else if prefLabel, ok := binding["prefLabel"]; ok {
			ancestor.Display = prefLabel.Value
		}

		if depth, ok := binding["depth"]; ok {
			fmt.Sscanf(depth.Value, "%d", &ancestor.Depth)
			if ancestor.Depth > maxDepth {
				maxDepth = ancestor.Depth
			}
		}

		if direct, ok := binding["direct"]; ok {
			ancestor.Direct = direct.Value == "true"
		}

		result.Ancestors = append(result.Ancestors, ancestor)
	}

	result.Total = len(result.Ancestors)
	result.MaxDepth = maxDepth

	// Get display name for the query concept
	displayA, _ := s.getDisplayNames(ctx, req.Code, req.Code, systemNS)
	result.Display = displayA

	// Cache result
	if s.cache != nil {
		if err := s.cache.Set(cacheKey, result, 30*time.Minute); err != nil {
			s.logger.WithError(err).Warn("Failed to cache ancestors result")
		}
	}

	return result, nil
}

// GetDescendants retrieves all descendants of a concept
func (s *SubsumptionService) GetDescendants(ctx context.Context, req *models.DescendantsRequest) (*models.DescendantsResult, error) {
	if s.graphDB == nil {
		return nil, fmt.Errorf("GraphDB client not available")
	}

	systemNS := s.getSystemNamespace(req.System)

	result := &models.DescendantsResult{
		Code:        req.Code,
		System:      req.System,
		Descendants: make([]models.ConceptDescendant, 0),
	}

	// Check cache
	cacheKey := fmt.Sprintf("descendants:%s:%s:%d:%d", req.System, req.Code, req.MaxDepth, req.Limit)
	if s.cache != nil {
		var cached models.DescendantsResult
		if err := s.cache.Get(cacheKey, &cached); err == nil {
			return &cached, nil
		}
	}

	// Build limit clause
	limit := req.Limit
	if limit <= 0 || limit > 1000 {
		limit = 500
	}

	// SPARQL query for descendants
	query := fmt.Sprintf(`
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
		PREFIX sys: <%s>

		SELECT DISTINCT ?descendant ?label ?depth ?direct WHERE {
			{
				?descendant rdfs:subClassOf+ sys:%s .
				BIND(EXISTS { ?descendant rdfs:subClassOf sys:%s } AS ?direct)
			}
			UNION
			{
				?descendant skos:broader+ sys:%s .
				BIND(EXISTS { ?descendant skos:broader sys:%s } AS ?direct)
			}
			OPTIONAL { ?descendant rdfs:label ?label }
			OPTIONAL { ?descendant skos:prefLabel ?prefLabel }

			BIND(IF(?direct, 1, 2) AS ?depth)
		}
		ORDER BY ?depth ?descendant
		LIMIT %d
	`, systemNS, req.Code, req.Code, req.Code, req.Code, limit+1)

	sparqlQuery := &semantic.SPARQLQuery{
		Query:  query,
		Format: "json",
	}

	results, err := s.graphDB.ExecuteSPARQL(ctx, sparqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query descendants: %w", err)
	}

	maxDepth := 0
	for i, binding := range results.Results.Bindings {
		if i >= limit {
			result.Truncated = true
			break
		}

		descendant := models.ConceptDescendant{}

		if descURI, ok := binding["descendant"]; ok {
			descendant.Code = s.extractCodeFromURI(descURI.Value)
		}

		if label, ok := binding["label"]; ok {
			descendant.Display = label.Value
		} else if prefLabel, ok := binding["prefLabel"]; ok {
			descendant.Display = prefLabel.Value
		}

		if depth, ok := binding["depth"]; ok {
			fmt.Sscanf(depth.Value, "%d", &descendant.Depth)
			if descendant.Depth > maxDepth {
				maxDepth = descendant.Depth
			}
		}

		if direct, ok := binding["direct"]; ok {
			descendant.Direct = direct.Value == "true"
		}

		result.Descendants = append(result.Descendants, descendant)
	}

	result.Total = len(result.Descendants)
	result.MaxDepth = maxDepth

	// Get display name
	displayA, _ := s.getDisplayNames(ctx, req.Code, req.Code, systemNS)
	result.Display = displayA

	// Cache result
	if s.cache != nil {
		if err := s.cache.Set(cacheKey, result, 30*time.Minute); err != nil {
			s.logger.WithError(err).Warn("Failed to cache descendants result")
		}
	}

	return result, nil
}

// FindCommonAncestors finds common ancestors of multiple concepts
func (s *SubsumptionService) FindCommonAncestors(ctx context.Context, req *models.CommonAncestorRequest) (*models.CommonAncestorResult, error) {
	if s.graphDB == nil {
		return nil, fmt.Errorf("GraphDB client not available")
	}

	if len(req.Codes) < 2 {
		return nil, fmt.Errorf("at least 2 codes required for common ancestor search")
	}

	systemNS := s.getSystemNamespace(req.System)

	result := &models.CommonAncestorResult{
		Codes:           req.Codes,
		System:          req.System,
		CommonAncestors: make([]models.ConceptAncestor, 0),
	}

	// Build SPARQL query for common ancestors
	// Each code must have the ancestor as a superclass
	codeConditions := make([]string, len(req.Codes))
	for i, code := range req.Codes {
		codeConditions[i] = fmt.Sprintf("sys:%s rdfs:subClassOf* ?ancestor", code)
	}

	query := fmt.Sprintf(`
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
		PREFIX sys: <%s>

		SELECT DISTINCT ?ancestor ?label ?maxDepth WHERE {
			%s
			OPTIONAL { ?ancestor rdfs:label ?label }
			OPTIONAL { ?ancestor skos:prefLabel ?prefLabel }

			# Filter to ensure it's a meaningful common ancestor (not the root)
			FILTER(EXISTS { ?ancestor rdfs:subClassOf ?something } || EXISTS { ?ancestor skos:broader ?something })
		}
		ORDER BY DESC(?maxDepth)
		LIMIT 100
	`, systemNS, strings.Join(codeConditions, " .\n\t\t\t"))

	sparqlQuery := &semantic.SPARQLQuery{
		Query:  query,
		Format: "json",
	}

	results, err := s.graphDB.ExecuteSPARQL(ctx, sparqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to find common ancestors: %w", err)
	}

	for _, binding := range results.Results.Bindings {
		ancestor := models.ConceptAncestor{}

		if ancestorURI, ok := binding["ancestor"]; ok {
			ancestor.Code = s.extractCodeFromURI(ancestorURI.Value)
		}

		if label, ok := binding["label"]; ok {
			ancestor.Display = label.Value
		} else if prefLabel, ok := binding["prefLabel"]; ok {
			ancestor.Display = prefLabel.Value
		}

		result.CommonAncestors = append(result.CommonAncestors, ancestor)
	}

	result.Total = len(result.CommonAncestors)

	// The first result (if any) is the lowest common ancestor
	if len(result.CommonAncestors) > 0 {
		lca := result.CommonAncestors[0]
		result.LowestCommonAncestor = &lca
	}

	return result, nil
}

// BatchTestSubsumption tests multiple subsumption relationships
func (s *SubsumptionService) BatchTestSubsumption(ctx context.Context, req *models.BatchSubsumptionRequest) (*models.BatchSubsumptionResult, error) {
	start := time.Now()

	result := &models.BatchSubsumptionResult{
		Results: make([]models.SubsumptionResult, 0, len(req.Tests)),
		Errors:  make([]models.SubsumptionError, 0),
	}

	for _, test := range req.Tests {
		testResult, err := s.TestSubsumption(ctx, &test)
		if err != nil {
			result.Errors = append(result.Errors, models.SubsumptionError{
				CodeA:  test.CodeA,
				CodeB:  test.CodeB,
				System: test.System,
				Error:  err.Error(),
			})
			result.ErrorCount++
		} else {
			result.Results = append(result.Results, *testResult)
			result.SuccessCount++
		}
	}

	result.TotalCount = len(req.Tests)
	result.ExecutionTime = float64(time.Since(start).Milliseconds())

	return result, nil
}

// getSystemNamespace returns the namespace URI for a terminology system
func (s *SubsumptionService) getSystemNamespace(system string) string {
	// Map common terminology systems to their namespaces
	namespaces := map[string]string{
		"http://snomed.info/sct":                       "http://snomed.info/id/",
		"http://hl7.org/fhir/sid/icd-10":               "http://hl7.org/fhir/sid/icd-10/",
		"http://hl7.org/fhir/sid/icd-10-cm":            "http://hl7.org/fhir/sid/icd-10-cm/",
		"http://www.nlm.nih.gov/research/umls/rxnorm":  "http://www.nlm.nih.gov/research/umls/rxnorm/",
		"http://loinc.org":                             "http://loinc.org/",
		"SNOMED-CT":                                    "http://snomed.info/id/",
		"ICD-10":                                       "http://hl7.org/fhir/sid/icd-10/",
		"ICD-10-CM":                                    "http://hl7.org/fhir/sid/icd-10-cm/",
		"RxNorm":                                       "http://www.nlm.nih.gov/research/umls/rxnorm/",
		"LOINC":                                        "http://loinc.org/",
	}

	if ns, ok := namespaces[system]; ok {
		return ns
	}

	// Default: use system URI as namespace
	if !strings.HasSuffix(system, "/") && !strings.HasSuffix(system, "#") {
		return system + "/"
	}
	return system
}

// extractCodeFromURI extracts the concept code from a full URI
func (s *SubsumptionService) extractCodeFromURI(uri string) string {
	// Handle different URI formats
	if idx := strings.LastIndex(uri, "/"); idx != -1 {
		return uri[idx+1:]
	}
	if idx := strings.LastIndex(uri, "#"); idx != -1 {
		return uri[idx+1:]
	}
	return uri
}

// cacheResult caches a subsumption result
func (s *SubsumptionService) cacheResult(key string, result *models.SubsumptionResult) {
	if s.cache != nil {
		if err := s.cache.Set(key, result, 1*time.Hour); err != nil {
			s.logger.WithError(err).Warn("Failed to cache subsumption result")
		}
	}
}

// IsAvailable returns true if subsumption testing is available
func (s *SubsumptionService) IsAvailable() bool {
	return s.graphDB != nil
}

// GetReasoningConfig returns the current OWL reasoning configuration
func (s *SubsumptionService) GetReasoningConfig() models.OWLReasoningConfig {
	return s.config
}
