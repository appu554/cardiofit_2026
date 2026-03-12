// Package semantic provides Neo4j client for KB-7 read replica operations.
// Phase 6.1: Neo4j Integration
//
// This client connects to Neo4j as a read replica for fast graph traversals,
// complementing GraphDB which serves as the OWL reasoning master.
//
// Architecture:
//   GraphDB (Master)     → OWL reasoning, FHIR compliance, write operations
//   Neo4j (Read Replica) → Fast traversals, Cypher queries, <10ms lookups
//
// The Neo4j instance is synchronized via CDC (Change Data Capture) from GraphDB.
package semantic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sirupsen/logrus"
)

// TerminologySystem represents supported terminology systems
type TerminologySystem string

const (
	// Core terminology systems
	SystemSNOMED  TerminologySystem = "http://snomed.info/sct"
	SystemRxNorm  TerminologySystem = "http://www.nlm.nih.gov/research/umls/rxnorm"
	SystemLOINC   TerminologySystem = "http://loinc.org"
	SystemICD10   TerminologySystem = "http://hl7.org/fhir/sid/icd-10-cm"
	SystemICD10AM TerminologySystem = "http://hl7.org.au/fhir/sid/icd-10-am"

	// Regional drug terminology systems (all use RF2 format like SNOMED)
	SystemAMT  TerminologySystem = "http://snomed.info/sct/900062011000036103" // Australian Medicines Terminology
	SystemCDCI TerminologySystem = "http://snomed.info/sct/cdci"               // Central Drug Standard Control Index (India)

	// Regional SNOMED extensions
	SystemSNOMEDAU TerminologySystem = "http://snomed.info/sct/32506021000036107" // SNOMED CT-AU
	SystemSNOMEDUS TerminologySystem = "http://snomed.info/sct/731000124108"      // SNOMED CT-US
)

// Region represents a deployment region with its own Neo4j database
type Region string

const (
	RegionAU Region = "au" // Australia
	RegionIN Region = "in" // India
	RegionUS Region = "us" // United States
)

// Neo4jConfig holds configuration for Neo4j connection
//
// Performance Tuning Guide:
//
//   MaxConnections (Connection Pool Size):
//     Formula: (RequestsPerSec × AvgQueryTime) + 50% buffer
//     Example: 500 req/s × 0.02s = 10 connections, add buffer → 15-20 connections
//     Standard API: 50-100 connections
//     High-traffic API or Flink batch: 100-200 connections
//
//   ConnTimeout (Connection Acquisition):
//     How long a goroutine waits for a connection from the pool.
//     Fail fast (10s) is better than hanging forever.
//     Recommended: 10-30 seconds
//
//   MaxConnLife (Connection Recycling):
//     Recycle connections to handle load balancer shifts (AWS/GCP NLB).
//     Prevents stale connections after LB target group changes.
//     Recommended: 30 minutes for cloud deployments
type Neo4jConfig struct {
	URL            string        // Neo4j bolt URL (e.g., bolt://localhost:7687)
	Username       string        // Neo4j username
	Password       string        // Neo4j password
	Database       string        // Database name (e.g., neo4j, kb7-au)
	MaxConnections int           // Pool size for concurrent goroutines (default: 100)
	ConnTimeout    time.Duration // Connection acquisition timeout (default: 10s)
	ReadTimeout    time.Duration // Query execution timeout (default: 60s)
	MaxConnLife    time.Duration // Connection recycling for AWS/GCP LB (default: 30min)
}

// RegionalNeo4jConfig holds configuration for multi-region Neo4j deployment
type RegionalNeo4jConfig struct {
	DefaultRegion Region
	Regions       map[Region]*Neo4jConfig
}

// RegionalNeo4jManager manages multiple Neo4j clients for different regions
type RegionalNeo4jManager struct {
	clients       map[Region]*Neo4jClient
	defaultRegion Region
	logger        *logrus.Logger
}

// NewRegionalNeo4jManager creates a manager for region-specific Neo4j clients
func NewRegionalNeo4jManager(config *RegionalNeo4jConfig, logger *logrus.Logger) (*RegionalNeo4jManager, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if logger == nil {
		logger = logrus.New()
	}

	manager := &RegionalNeo4jManager{
		clients:       make(map[Region]*Neo4jClient),
		defaultRegion: config.DefaultRegion,
		logger:        logger,
	}

	// Initialize clients for each region
	for region, regionConfig := range config.Regions {
		client, err := NewNeo4jClient(regionConfig, logger)
		if err != nil {
			// Close already created clients on failure
			for _, c := range manager.clients {
				c.Close(context.Background())
			}
			return nil, fmt.Errorf("creating client for region %s: %w", region, err)
		}
		manager.clients[region] = client
		logger.WithField("region", region).Info("Connected to regional Neo4j database")
	}

	return manager, nil
}

// GetClient returns the Neo4j client for a specific region
func (m *RegionalNeo4jManager) GetClient(region Region) *Neo4jClient {
	if client, ok := m.clients[region]; ok {
		return client
	}
	// Fallback to default region
	return m.clients[m.defaultRegion]
}

// GetClientForRequest determines the region from context or header and returns appropriate client
func (m *RegionalNeo4jManager) GetClientForRequest(ctx context.Context, regionHint string) *Neo4jClient {
	if regionHint != "" {
		region := Region(strings.ToLower(regionHint))
		if client, ok := m.clients[region]; ok {
			return client
		}
	}
	return m.clients[m.defaultRegion]
}

// Close closes all regional Neo4j clients
func (m *RegionalNeo4jManager) Close(ctx context.Context) error {
	var lastErr error
	for region, client := range m.clients {
		if err := client.Close(ctx); err != nil {
			m.logger.WithError(err).WithField("region", region).Error("Failed to close regional client")
			lastErr = err
		}
	}
	return lastErr
}

// HealthAll checks health of all regional databases
func (m *RegionalNeo4jManager) HealthAll(ctx context.Context) map[Region]error {
	results := make(map[Region]error)
	for region, client := range m.clients {
		results[region] = client.Health(ctx)
	}
	return results
}

// Neo4jClient provides interface to Neo4j read replica for fast traversals
type Neo4jClient struct {
	driver   neo4j.DriverWithContext
	database string
	logger   *logrus.Logger
	config   *Neo4jConfig
}

// Concept represents a clinical concept from the knowledge graph
type Concept struct {
	Code        string            `json:"code"`
	Display     string            `json:"display"`
	System      string            `json:"system"`
	URI         string            `json:"uri,omitempty"`
	Definition  string            `json:"definition,omitempty"`
	Status      string            `json:"status,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
	ParentCount int               `json:"parent_count,omitempty"`
	ChildCount  int               `json:"child_count,omitempty"`
}

// Relationship represents a relationship between concepts
type Relationship struct {
	SourceCode   string `json:"source_code"`
	SourceSystem string `json:"source_system"`
	TargetCode   string `json:"target_code"`
	TargetSystem string `json:"target_system"`
	Type         string `json:"type"`
	URI          string `json:"uri,omitempty"`
}

// HierarchyResult contains hierarchy traversal results
type HierarchyResult struct {
	Concept       *Concept   `json:"concept"`
	Ancestors     []*Concept `json:"ancestors,omitempty"`
	Descendants   []*Concept `json:"descendants,omitempty"`
	Siblings      []*Concept `json:"siblings,omitempty"`
	TotalAncestors   int     `json:"total_ancestors"`
	TotalDescendants int     `json:"total_descendants"`
	MaxDepth         int     `json:"max_depth"`
}

// SubsumptionResult contains the result of a subsumption check
type SubsumptionResult struct {
	IsSubsumed    bool          `json:"is_subsumed"`
	ChildCode     string        `json:"child_code"`
	ParentCode    string        `json:"parent_code"`
	System        string        `json:"system"`
	PathLength    int           `json:"path_length,omitempty"`
	IntermediateNodes []string  `json:"intermediate_nodes,omitempty"`
	ComputationTime   time.Duration `json:"computation_time"`
}

// NewNeo4jClient creates a new Neo4j client with the provided configuration
func NewNeo4jClient(config *Neo4jConfig, logger *logrus.Logger) (*Neo4jClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if logger == nil {
		logger = logrus.New()
	}

	// Set defaults with production-ready values
	if config.MaxConnections == 0 {
		config.MaxConnections = 100 // Default pool size for standard API traffic
	}
	if config.ConnTimeout == 0 {
		config.ConnTimeout = 10 * time.Second // Fail fast - don't hang forever
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 60 * time.Second
	}
	if config.MaxConnLife == 0 {
		config.MaxConnLife = 30 * time.Minute // Recycle for load balancer compatibility
	}

	// Create driver with connection pool configuration
	// See Neo4jConfig comments for tuning guidance
	driver, err := neo4j.NewDriverWithContext(
		config.URL,
		neo4j.BasicAuth(config.Username, config.Password, ""),
		func(conf *neo4j.Config) {
			// POOL SIZE: Matches your maximum expected goroutines accessing Neo4j simultaneously
			conf.MaxConnectionPoolSize = config.MaxConnections

			// ACQUISITION TIMEOUT: How long a goroutine waits for a connection before failing
			// Fail fast (10s) is better than hanging forever
			conf.ConnectionAcquisitionTimeout = config.ConnTimeout

			// RETRY TIME: Maximum time for transaction retries
			conf.MaxTransactionRetryTime = config.ReadTimeout

			// LIFETIME: Recycle connections to handle load balancer shifts (AWS/GCP NLB)
			// This prevents stale connections when LB target groups change
			conf.MaxConnectionLifetime = config.MaxConnLife

			// LOGGING: Enable for debugging slow queries (optional)
			// conf.Log = neo4j.ConsoleLogger(neo4j.WarningLevel)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("creating neo4j driver: %w", err)
	}

	client := &Neo4jClient{
		driver:   driver,
		database: config.Database,
		logger:   logger,
		config:   config,
	}

	// Verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("verifying neo4j connectivity: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"url":      config.URL,
		"database": config.Database,
	}).Info("Connected to Neo4j read replica")

	return client, nil
}

// Close closes the Neo4j driver connection
func (n *Neo4jClient) Close(ctx context.Context) error {
	return n.driver.Close(ctx)
}

// ExecuteRead executes a Cypher read query and returns results as a slice of maps
// This is a generic method for arbitrary read queries
func (n *Neo4jClient) ExecuteRead(ctx context.Context, cypher string, params map[string]interface{}) ([]map[string]interface{}, error) {
	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: n.database,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("executing cypher query: %w", err)
	}

	var results []map[string]interface{}
	for result.Next(ctx) {
		record := result.Record()
		row := make(map[string]interface{})
		for _, key := range record.Keys {
			val, _ := record.Get(key)
			row[key] = val
		}
		results = append(results, row)
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("reading query results: %w", err)
	}

	return results, nil
}

// Health checks if Neo4j is responsive
func (n *Neo4jClient) Health(ctx context.Context) error {
	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: n.database,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, "RETURN 1 as health", nil)
	if err != nil {
		return fmt.Errorf("health check query failed: %w", err)
	}

	if result.Next(ctx) {
		return nil
	}

	return fmt.Errorf("health check returned no results")
}

// GetConcept retrieves a concept by code and system using Cypher
// Note: Uses n10s (neosemantics) schema with Resource nodes
// - LOINC/RxNorm: ns1__code + ns1__system properties
// - SNOMED: uri = http://snomed.info/id/{code}, no ns1__system
func (n *Neo4jClient) GetConcept(ctx context.Context, code, system string) (*Concept, error) {
	start := time.Now()
	defer func() {
		n.logger.WithFields(logrus.Fields{
			"code":     code,
			"system":   system,
			"duration": time.Since(start),
		}).Debug("GetConcept completed")
	}()

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: n.database,
	})
	defer session.Close(ctx)

	normalizedSystem := normalizeSystemForN10s(system)
	var query string
	var params map[string]interface{}

	if normalizedSystem == "SNOMED" {
		// SNOMED uses URI pattern: http://snomed.info/id/{code}
		query = `
			MATCH (c:Resource)
			WHERE c.uri = $uri
			OPTIONAL MATCH (c)-[:subClassOf]->(parent:Resource)
			OPTIONAL MATCH (child:Resource)-[:subClassOf]->(c)
			RETURN $code as code,
			       c.rdfs__label as display,
			       'SNOMED' as system,
			       c.uri as uri,
			       c.skos__prefLabel as definition,
			       '' as status,
			       count(DISTINCT parent) as parentCount,
			       count(DISTINCT child) as childCount
		`
		params = map[string]interface{}{
			"code": code,
			"uri":  fmt.Sprintf("http://snomed.info/id/%s", code),
		}
	} else {
		// LOINC/RxNorm use ns1__code + ns1__system properties
		query = `
			MATCH (c:Resource)
			WHERE c.ns1__code = $code AND c.ns1__system = $system
			OPTIONAL MATCH (c)-[:subClassOf]->(parent:Resource)
			OPTIONAL MATCH (child:Resource)-[:subClassOf]->(c)
			RETURN c.ns1__code as code,
			       c.rdfs__label as display,
			       c.ns1__system as system,
			       c.uri as uri,
			       c.ns1__definition as definition,
			       c.ns1__status as status,
			       count(DISTINCT parent) as parentCount,
			       count(DISTINCT child) as childCount
		`
		params = map[string]interface{}{
			"code":   code,
			"system": normalizedSystem,
		}
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("querying concept: %w", err)
	}

	if result.Next(ctx) {
		record := result.Record()
		concept := &Concept{
			Code:        getStringValue(record, "code"),
			Display:     getStringValue(record, "display"),
			System:      getStringValue(record, "system"),
			URI:         getStringValue(record, "uri"),
			Definition:  getStringValue(record, "definition"),
			Status:      getStringValue(record, "status"),
			ParentCount: int(getInt64Value(record, "parentCount")),
			ChildCount:  int(getInt64Value(record, "childCount")),
		}
		return concept, nil
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("reading result: %w", err)
	}

	return nil, nil // Not found
}

// GetConceptBatch retrieves multiple concepts by codes (batch lookup)
// Uses n10s schema with Resource nodes and ns1__ prefixed properties
func (n *Neo4jClient) GetConceptBatch(ctx context.Context, codes []string, system string) ([]*Concept, error) {
	if len(codes) == 0 {
		return nil, nil
	}

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: n.database,
	})
	defer session.Close(ctx)

	query := `
		UNWIND $codes as code
		MATCH (c:Resource)
		WHERE c.ns1__code = code AND c.ns1__system = $system
		RETURN c.ns1__code as code,
		       c.rdfs__label as display,
		       c.ns1__system as system,
		       c.uri as uri,
		       c.ns1__definition as definition
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"codes":  codes,
		"system": normalizeSystemForN10s(system),
	})
	if err != nil {
		return nil, fmt.Errorf("batch querying concepts: %w", err)
	}

	var concepts []*Concept
	for result.Next(ctx) {
		record := result.Record()
		concepts = append(concepts, &Concept{
			Code:       getStringValue(record, "code"),
			Display:    getStringValue(record, "display"),
			System:     getStringValue(record, "system"),
			URI:        getStringValue(record, "uri"),
			Definition: getStringValue(record, "definition"),
		})
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("reading batch results: %w", err)
	}

	return concepts, nil
}

// IsSubsumedBy checks if childCode is subsumed by parentCode using the ELK materialized hierarchy
// This is the core method for clinical reasoning - "Is diabetes a type of endocrine disorder?"
//
// All terminologies use subClassOf for hierarchy traversal:
// - SNOMED/AMT/CDCI: URI-based lookup (http://snomed.info/id/{code})
// - LOINC/RxNorm/Others: ns1__code + ns1__system properties
//
// Note: RxNorm hierarchy is also materialized using subClassOf after ELK reasoning.
// The ns1__rxnorm_RB/RN relationships are NOT used for subsumption - only subClassOf.
func (n *Neo4jClient) IsSubsumedBy(ctx context.Context, childCode, parentCode, system string) (*SubsumptionResult, error) {
	start := time.Now()

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: n.database,
	})
	defer session.Close(ctx)

	normalizedSystem := normalizeSystemForN10s(system)
	var query string
	var params map[string]interface{}

	// SNOMED-based terminologies (SNOMED, AMT, CDCI) use URI pattern
	// Using shortestPath() for optimized BFS traversal - more efficient than arbitrary path matching
	if isSNOMEDBasedSystem(normalizedSystem) {
		query = `
			MATCH (child:Resource), (parent:Resource)
			WHERE child.uri = $childUri AND parent.uri = $parentUri
			MATCH path = shortestPath((child)-[:subClassOf*1..15]->(parent))
			RETURN length(path) as pathLength,
			       [n in nodes(path) |
			        CASE WHEN n.uri STARTS WITH 'http://snomed.info/id/'
			             THEN substring(n.uri, 24)
			             ELSE n.uri END
			       ] as pathCodes
		`
		params = map[string]interface{}{
			"childUri":  fmt.Sprintf("http://snomed.info/id/%s", childCode),
			"parentUri": fmt.Sprintf("http://snomed.info/id/%s", parentCode),
		}
	} else {
		// LOINC, RxNorm, and other terminologies use ns1__code + ns1__system
		// All use subClassOf for hierarchy (materialized by ELK reasoning)
		// Using shortestPath() for optimized BFS traversal
		query = `
			MATCH (child:Resource), (parent:Resource)
			WHERE child.ns1__code = $childCode AND child.ns1__system = $system
			  AND parent.ns1__code = $parentCode AND parent.ns1__system = $system
			MATCH path = shortestPath((child)-[:subClassOf*1..15]->(parent))
			RETURN length(path) as pathLength,
			       [n in nodes(path) | n.ns1__code] as pathCodes
		`
		params = map[string]interface{}{
			"childCode":  childCode,
			"parentCode": parentCode,
			"system":     normalizedSystem,
		}
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("querying subsumption: %w", err)
	}

	subsumptionResult := &SubsumptionResult{
		ChildCode:       childCode,
		ParentCode:      parentCode,
		System:          system,
		ComputationTime: time.Since(start),
	}

	if result.Next(ctx) {
		record := result.Record()
		subsumptionResult.IsSubsumed = true
		subsumptionResult.PathLength = int(getInt64Value(record, "pathLength"))

		// Extract intermediate nodes (excluding first and last)
		if pathCodes, ok := record.Get("pathCodes"); ok {
			if codes, ok := pathCodes.([]interface{}); ok && len(codes) > 2 {
				for i := 1; i < len(codes)-1; i++ {
					if code, ok := codes[i].(string); ok {
						subsumptionResult.IntermediateNodes = append(subsumptionResult.IntermediateNodes, code)
					}
				}
			}
		}
	} else {
		subsumptionResult.IsSubsumed = false
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("reading subsumption result: %w", err)
	}

	n.logger.WithFields(logrus.Fields{
		"childCode":   childCode,
		"parentCode":  parentCode,
		"isSubsumed":  subsumptionResult.IsSubsumed,
		"pathLength":  subsumptionResult.PathLength,
		"duration_ms": subsumptionResult.ComputationTime.Milliseconds(),
	}).Debug("Subsumption check completed")

	return subsumptionResult, nil
}

// GetAncestors retrieves all ancestors of a concept up to maxDepth
// All terminologies use subClassOf for hierarchy traversal (materialized by ELK reasoning):
// - SNOMED/AMT/CDCI: URI-based lookup (http://snomed.info/id/{code})
// - LOINC/RxNorm/Others: ns1__code + ns1__system properties
func (n *Neo4jClient) GetAncestors(ctx context.Context, code, system string, maxDepth int) ([]*Concept, error) {
	if maxDepth <= 0 {
		maxDepth = 20 // Default max depth
	}

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: n.database,
	})
	defer session.Close(ctx)

	normalizedSystem := normalizeSystemForN10s(system)
	var query string
	var params map[string]interface{}

	// SNOMED-based terminologies (SNOMED, AMT, CDCI) use URI pattern
	if isSNOMEDBasedSystem(normalizedSystem) {
		query = fmt.Sprintf(`
			MATCH (child:Resource)-[:subClassOf*1..%d]->(ancestor:Resource)
			WHERE child.uri = $uri
			RETURN DISTINCT
			       CASE WHEN ancestor.uri STARTS WITH 'http://snomed.info/id/'
			            THEN substring(ancestor.uri, 24)
			            ELSE ancestor.uri END as code,
			       ancestor.rdfs__label as display,
			       'SNOMED' as system,
			       ancestor.uri as uri
			ORDER BY ancestor.rdfs__label
			LIMIT 1000
		`, maxDepth)
		params = map[string]interface{}{
			"uri": fmt.Sprintf("http://snomed.info/id/%s", code),
		}
	} else {
		// LOINC, RxNorm, and other terminologies use ns1__code + ns1__system
		// All use subClassOf for hierarchy (materialized by ELK reasoning)
		query = fmt.Sprintf(`
			MATCH (child:Resource)-[:subClassOf*1..%d]->(ancestor:Resource)
			WHERE child.ns1__code = $code AND child.ns1__system = $system
			RETURN DISTINCT ancestor.ns1__code as code,
			       ancestor.rdfs__label as display,
			       ancestor.ns1__system as system,
			       ancestor.uri as uri
			ORDER BY ancestor.rdfs__label
			LIMIT 1000
		`, maxDepth)
		params = map[string]interface{}{
			"code":   code,
			"system": normalizedSystem,
		}
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("querying ancestors: %w", err)
	}

	var ancestors []*Concept
	for result.Next(ctx) {
		record := result.Record()
		ancestors = append(ancestors, &Concept{
			Code:    getStringValue(record, "code"),
			Display: getStringValue(record, "display"),
			System:  getStringValue(record, "system"),
			URI:     getStringValue(record, "uri"),
		})
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("reading ancestors: %w", err)
	}

	return ancestors, nil
}

// GetDescendants retrieves all descendants of a concept up to maxDepth
// All terminologies use subClassOf (reverse) for hierarchy traversal:
// - SNOMED/AMT/CDCI: URI-based lookup (http://snomed.info/id/{code})
// - LOINC/RxNorm/Others: ns1__code + ns1__system properties
func (n *Neo4jClient) GetDescendants(ctx context.Context, code, system string, maxDepth int) ([]*Concept, error) {
	if maxDepth <= 0 {
		maxDepth = 5 // Default max depth (descendants can be many)
	}

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: n.database,
	})
	defer session.Close(ctx)

	normalizedSystem := normalizeSystemForN10s(system)
	var query string
	var params map[string]interface{}

	// SNOMED-based terminologies (SNOMED, AMT, CDCI) use URI pattern
	if isSNOMEDBasedSystem(normalizedSystem) {
		query = fmt.Sprintf(`
			MATCH (parent:Resource)<-[:subClassOf*1..%d]-(descendant:Resource)
			WHERE parent.uri = $uri
			RETURN DISTINCT
			       CASE WHEN descendant.uri STARTS WITH 'http://snomed.info/id/'
			            THEN substring(descendant.uri, 24)
			            ELSE descendant.uri END as code,
			       descendant.rdfs__label as display,
			       'SNOMED' as system,
			       descendant.uri as uri
			ORDER BY descendant.rdfs__label
			LIMIT 1000
		`, maxDepth)
		params = map[string]interface{}{
			"uri": fmt.Sprintf("http://snomed.info/id/%s", code),
		}
	} else {
		// LOINC, RxNorm, and other terminologies use ns1__code + ns1__system
		// All use subClassOf for hierarchy (materialized by ELK reasoning)
		query = fmt.Sprintf(`
			MATCH (parent:Resource)<-[:subClassOf*1..%d]-(descendant:Resource)
			WHERE parent.ns1__code = $code AND parent.ns1__system = $system
			RETURN DISTINCT descendant.ns1__code as code,
			       descendant.rdfs__label as display,
			       descendant.ns1__system as system,
			       descendant.uri as uri
			ORDER BY descendant.rdfs__label
			LIMIT 1000
		`, maxDepth)
		params = map[string]interface{}{
			"code":   code,
			"system": normalizedSystem,
		}
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("querying descendants: %w", err)
	}

	var descendants []*Concept
	for result.Next(ctx) {
		record := result.Record()
		descendants = append(descendants, &Concept{
			Code:    getStringValue(record, "code"),
			Display: getStringValue(record, "display"),
			System:  getStringValue(record, "system"),
			URI:     getStringValue(record, "uri"),
		})
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("reading descendants: %w", err)
	}

	return descendants, nil
}

// GetHierarchy retrieves the full hierarchy context for a concept
func (n *Neo4jClient) GetHierarchy(ctx context.Context, code, system string, ancestorDepth, descendantDepth int) (*HierarchyResult, error) {
	concept, err := n.GetConcept(ctx, code, system)
	if err != nil {
		return nil, fmt.Errorf("getting concept: %w", err)
	}
	if concept == nil {
		return nil, fmt.Errorf("concept not found: %s", code)
	}

	ancestors, err := n.GetAncestors(ctx, code, system, ancestorDepth)
	if err != nil {
		return nil, fmt.Errorf("getting ancestors: %w", err)
	}

	descendants, err := n.GetDescendants(ctx, code, system, descendantDepth)
	if err != nil {
		return nil, fmt.Errorf("getting descendants: %w", err)
	}

	return &HierarchyResult{
		Concept:          concept,
		Ancestors:        ancestors,
		Descendants:      descendants,
		TotalAncestors:   len(ancestors),
		TotalDescendants: len(descendants),
	}, nil
}

// SearchConcepts searches for concepts matching a text query
// Uses n10s schema with Resource nodes:
// - LOINC/RxNorm: ns1__system + ns1__code properties
// - SNOMED: uri pattern + rdfs__label
func (n *Neo4jClient) SearchConcepts(ctx context.Context, query, system string, limit int) ([]*Concept, error) {
	if limit <= 0 {
		limit = 100
	}

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: n.database,
	})
	defer session.Close(ctx)

	normalizedSystem := normalizeSystemForN10s(system)
	var cypherQuery string
	var params map[string]interface{}

	if normalizedSystem == "SNOMED" {
		// SNOMED uses URI pattern, search by label
		cypherQuery = `
			MATCH (c:Resource)
			WHERE c.uri STARTS WITH 'http://snomed.info/id/'
			  AND toLower(c.rdfs__label) CONTAINS toLower($query)
			RETURN
			       substring(c.uri, 24) as code,
			       c.rdfs__label as display,
			       'SNOMED' as system,
			       c.uri as uri
			ORDER BY
			  CASE WHEN toLower(c.rdfs__label) STARTS WITH toLower($query) THEN 0 ELSE 1 END,
			  c.rdfs__label
			LIMIT $limit
		`
		params = map[string]interface{}{
			"query": query,
			"limit": limit,
		}
	} else {
		// LOINC/RxNorm use ns1__system + ns1__code
		cypherQuery = `
			MATCH (c:Resource)
			WHERE c.ns1__system = $system
			  AND (toLower(c.rdfs__label) CONTAINS toLower($query)
			       OR toLower(c.ns1__code) CONTAINS toLower($query))
			RETURN c.ns1__code as code,
			       c.rdfs__label as display,
			       c.ns1__system as system,
			       c.uri as uri
			ORDER BY
			  CASE WHEN toLower(c.rdfs__label) STARTS WITH toLower($query) THEN 0 ELSE 1 END,
			  c.rdfs__label
			LIMIT $limit
		`
		params = map[string]interface{}{
			"query":  query,
			"system": normalizedSystem,
			"limit":  limit,
		}
	}

	result, err := session.Run(ctx, cypherQuery, params)
	if err != nil {
		return nil, fmt.Errorf("searching concepts: %w", err)
	}

	var concepts []*Concept
	for result.Next(ctx) {
		record := result.Record()
		concepts = append(concepts, &Concept{
			Code:    getStringValue(record, "code"),
			Display: getStringValue(record, "display"),
			System:  getStringValue(record, "system"),
			URI:     getStringValue(record, "uri"),
		})
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("reading search results: %w", err)
	}

	return concepts, nil
}

// GetMappings retrieves cross-system mappings for a concept (e.g., SNOMED → ICD-10)
// Uses n10s schema with Resource nodes and ns1__ prefixed properties
func (n *Neo4jClient) GetMappings(ctx context.Context, code, sourceSystem, targetSystem string) ([]*Relationship, error) {
	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: n.database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (source:Resource)-[r]->(target:Resource)
		WHERE source.ns1__code = $code AND source.ns1__system = $sourceSystem
		  AND target.ns1__system = $targetSystem
		  AND type(r) IN ['skos__exactMatch', 'skos__broadMatch', 'skos__narrowMatch', 'skos__relatedMatch']
		RETURN source.ns1__code as sourceCode,
		       source.ns1__system as sourceSystem,
		       target.ns1__code as targetCode,
		       target.ns1__system as targetSystem,
		       type(r) as relType
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"code":         code,
		"sourceSystem": normalizeSystemForN10s(sourceSystem),
		"targetSystem": normalizeSystemForN10s(targetSystem),
	})
	if err != nil {
		return nil, fmt.Errorf("querying mappings: %w", err)
	}

	var mappings []*Relationship
	for result.Next(ctx) {
		record := result.Record()
		mappings = append(mappings, &Relationship{
			SourceCode:   getStringValue(record, "sourceCode"),
			SourceSystem: getStringValue(record, "sourceSystem"),
			TargetCode:   getStringValue(record, "targetCode"),
			TargetSystem: getStringValue(record, "targetSystem"),
			Type:         getStringValue(record, "relType"),
		})
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("reading mappings: %w", err)
	}

	return mappings, nil
}

// GetRxNormRelationships retrieves RxNorm-specific relationships (ingredients, trade names, etc.)
// Uses n10s schema with Resource nodes and ns1__ prefixed properties
func (n *Neo4jClient) GetRxNormRelationships(ctx context.Context, code string, relationType string) ([]*Relationship, error) {
	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: n.database,
	})
	defer session.Close(ctx)

	// RxNorm-specific relationships in n10s schema: ns1__rxnorm_RB, ns1__rxnorm_RN, ns1__rxnorm_RO, etc.
	query := `
		MATCH (drug:Resource)-[r]->(related:Resource)
		WHERE drug.ns1__code = $code AND drug.ns1__system = $rxnormSystem
		  AND (type(r) = $relationType OR $relationType = '')
		RETURN drug.ns1__code as sourceCode,
		       drug.ns1__system as sourceSystem,
		       related.ns1__code as targetCode,
		       related.ns1__system as targetSystem,
		       type(r) as relType
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"code":         code,
		"rxnormSystem": "RXNORM",
		"relationType": relationType,
	})
	if err != nil {
		return nil, fmt.Errorf("querying rxnorm relationships: %w", err)
	}

	var relationships []*Relationship
	for result.Next(ctx) {
		record := result.Record()
		relationships = append(relationships, &Relationship{
			SourceCode:   getStringValue(record, "sourceCode"),
			SourceSystem: getStringValue(record, "sourceSystem"),
			TargetCode:   getStringValue(record, "targetCode"),
			TargetSystem: getStringValue(record, "targetSystem"),
			Type:         getStringValue(record, "relType"),
		})
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("reading rxnorm relationships: %w", err)
	}

	return relationships, nil
}

// GetStatistics returns statistics about the knowledge graph
// Uses n10s schema with Resource nodes:
// - LOINC/RxNorm: ns1__system property
// - SNOMED: uri starts with http://snomed.info/id/
func (n *Neo4jClient) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: n.database,
	})
	defer session.Close(ctx)

	stats := map[string]interface{}{
		"systems": make(map[string]int64),
	}
	systems := stats["systems"].(map[string]int64)
	var total int64

	// Query 1: Get counts for LOINC, RxNorm, etc. (nodes with ns1__system)
	query1 := `
		MATCH (c:Resource)
		WHERE c.ns1__system IS NOT NULL
		WITH c.ns1__system as system, count(c) as conceptCount
		RETURN system, conceptCount
		ORDER BY conceptCount DESC
	`

	result, err := session.Run(ctx, query1, nil)
	if err != nil {
		return nil, fmt.Errorf("querying statistics: %w", err)
	}

	for result.Next(ctx) {
		record := result.Record()
		system := getStringValue(record, "system")
		count := getInt64Value(record, "conceptCount")
		systems[system] = count
		total += count
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("reading statistics: %w", err)
	}

	// Query 2: Count SNOMED concepts (uri starts with http://snomed.info/id/)
	query2 := `
		MATCH (c:Resource)
		WHERE c.uri STARTS WITH 'http://snomed.info/id/'
		RETURN count(c) as snomedCount
	`

	result2, err := session.Run(ctx, query2, nil)
	if err != nil {
		return nil, fmt.Errorf("querying SNOMED count: %w", err)
	}

	if result2.Next(ctx) {
		record := result2.Record()
		snomedCount := getInt64Value(record, "snomedCount")
		if snomedCount > 0 {
			systems["SNOMED"] = snomedCount
			total += snomedCount
		}
	}

	if err := result2.Err(); err != nil {
		return nil, fmt.Errorf("reading SNOMED count: %w", err)
	}

	// Query 3: Count hierarchy relationships
	query3 := `
		MATCH ()-[r:subClassOf]->()
		RETURN count(r) as hierarchyCount
	`

	result3, err := session.Run(ctx, query3, nil)
	if err != nil {
		return nil, fmt.Errorf("querying hierarchy count: %w", err)
	}

	if result3.Next(ctx) {
		record := result3.Record()
		stats["hierarchy_relationships"] = getInt64Value(record, "hierarchyCount")
	}

	stats["total_concepts"] = total

	return stats, nil
}

// Helper functions

// getStringValue extracts a string value from a Neo4j record
// Handles both direct strings and arrays (extracts first element)
// This is needed because n10s imports may store rdfs:label as an array
func getStringValue(record *neo4j.Record, key string) string {
	if val, ok := record.Get(key); ok && val != nil {
		// Direct string value
		if str, ok := val.(string); ok {
			return str
		}
		// Array value (e.g., rdfs__label: ['Diabetes mellitus type 2 (disorder)'])
		// Extract first element as the display name
		if arr, ok := val.([]interface{}); ok && len(arr) > 0 {
			if str, ok := arr[0].(string); ok {
				return str
			}
		}
	}
	return ""
}

func getInt64Value(record *neo4j.Record, key string) int64 {
	if val, ok := record.Get(key); ok && val != nil {
		if num, ok := val.(int64); ok {
			return num
		}
	}
	return 0
}

// normalizeSystem converts common system aliases to their canonical URIs
func normalizeSystem(system string) string {
	// Handle common aliases
	lowerSystem := strings.ToLower(system)
	switch {
	case lowerSystem == "snomed" || lowerSystem == "snomedct" || lowerSystem == "snomed-ct":
		return string(SystemSNOMED)
	case lowerSystem == "rxnorm":
		return string(SystemRxNorm)
	case lowerSystem == "loinc":
		return string(SystemLOINC)
	case lowerSystem == "icd10" || lowerSystem == "icd-10" || lowerSystem == "icd10cm" || lowerSystem == "icd-10-cm":
		return string(SystemICD10)
	case lowerSystem == "icd10am" || lowerSystem == "icd-10-am":
		return string(SystemICD10AM)
	case lowerSystem == "amt":
		return string(SystemAMT)
	default:
		return system
	}
}

// normalizeSystemForN10s converts system identifiers to the short format used in n10s data
// The n10s imported data uses short names like "LOINC", "RXNORM", "SNOMED" instead of full URIs
func normalizeSystemForN10s(system string) string {
	lowerSystem := strings.ToLower(system)

	// Map full URIs to short names
	switch {
	case strings.Contains(lowerSystem, "snomed"):
		return "SNOMED"
	case strings.Contains(lowerSystem, "amt") || strings.Contains(lowerSystem, "900062011000036103"):
		return "SNOMED" // AMT uses SNOMED URI pattern
	case strings.Contains(lowerSystem, "cdci"):
		return "SNOMED" // CDCI uses SNOMED URI pattern
	case strings.Contains(lowerSystem, "rxnorm"):
		return "RXNORM"
	case strings.Contains(lowerSystem, "loinc"):
		return "LOINC"
	case strings.Contains(lowerSystem, "icd-10"):
		return "ICD10"
	case strings.Contains(lowerSystem, "icd10"):
		return "ICD10"
	default:
		// Return uppercase version for short names
		return strings.ToUpper(system)
	}
}

// isSNOMEDBasedSystem checks if the terminology uses SNOMED's URI pattern (http://snomed.info/id/{code})
// This includes SNOMED CT, AMT (Australian Medicines Terminology), and CDCI (India)
// These all use RF2 format and have subClassOf relationships after ELK reasoning
func isSNOMEDBasedSystem(normalizedSystem string) bool {
	switch normalizedSystem {
	case "SNOMED", "AMT", "CDCI":
		return true
	default:
		return false
	}
}
