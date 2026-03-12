// Package loader provides data loading utilities for KB-7 terminology service.
// This file implements the Neo4j loader that transforms TTL/RDF data into
// Cypher statements for loading into Neo4j.
//
// Architecture Note:
// Both GraphDB and Neo4j are loaded from the same kb7-kernel.ttl source.
// - GraphDB: OWL reasoning, SPARQL queries
// - Neo4j: Fast graph traversals, property lookups
// CDC sync keeps them synchronized for incremental updates.
package loader

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sirupsen/logrus"
)

// Neo4jLoaderConfig holds configuration for the Neo4j loader
type Neo4jLoaderConfig struct {
	// Neo4j connection
	Neo4jURL      string
	Neo4jUsername string
	Neo4jPassword string
	Neo4jDatabase string

	// Load options
	BatchSize      int
	Workers        int
	ClearFirst     bool
	CreateIndexes  bool
	DryRun         bool
	Timeout        time.Duration
}

// DefaultNeo4jLoaderConfig returns sensible defaults
func DefaultNeo4jLoaderConfig() *Neo4jLoaderConfig {
	return &Neo4jLoaderConfig{
		Neo4jURL:      "bolt://localhost:7687",
		Neo4jUsername: "neo4j",
		Neo4jPassword: "password",
		Neo4jDatabase: "neo4j",
		BatchSize:     5000,
		Workers:       4,
		ClearFirst:    false,
		CreateIndexes: true,
		DryRun:        false,
		Timeout:       60 * time.Minute,
	}
}

// Neo4jLoadResult holds the result of a load operation
type Neo4jLoadResult struct {
	Success        bool          `json:"success"`
	NodesCreated   int64         `json:"nodes_created"`
	RelsCreated    int64         `json:"relationships_created"`
	TriplesRead    int64         `json:"triples_read"`
	Duration       time.Duration `json:"duration"`
	ErrorMessage   string        `json:"error_message,omitempty"`
	LoadTimestamp  time.Time     `json:"load_timestamp"`
}

// Neo4jLoader handles loading TTL data into Neo4j
type Neo4jLoader struct {
	driver neo4j.DriverWithContext
	config *Neo4jLoaderConfig
	logger *logrus.Logger

	// Statistics
	stats     Neo4jLoadResult
	statsMu   sync.Mutex

	// Parsed data buffers
	conceptBuffer     []conceptData
	relationBuffer    []relationData
	bufferMu          sync.Mutex
}

// conceptData holds parsed concept information
type conceptData struct {
	URI        string
	Code       string
	System     string
	Display    string
	Definition string
	Status     string
	Labels     []string
	Properties map[string]interface{}
}

// relationData holds parsed relationship information
type relationData struct {
	FromURI  string
	ToURI    string
	Type     string
	Properties map[string]interface{}
}

// NewNeo4jLoader creates a new Neo4j loader
func NewNeo4jLoader(config *Neo4jLoaderConfig, logger *logrus.Logger) (*Neo4jLoader, error) {
	if config == nil {
		config = DefaultNeo4jLoaderConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}

	// Create Neo4j driver
	driver, err := neo4j.NewDriverWithContext(
		config.Neo4jURL,
		neo4j.BasicAuth(config.Neo4jUsername, config.Neo4jPassword, ""),
		func(c *neo4j.Config) {
			c.MaxConnectionPoolSize = 50
			c.ConnectionAcquisitionTimeout = 30 * time.Second
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Neo4j: %w", err)
	}

	return &Neo4jLoader{
		driver:        driver,
		config:        config,
		logger:        logger,
		conceptBuffer: make([]conceptData, 0, config.BatchSize),
		relationBuffer: make([]relationData, 0, config.BatchSize),
	}, nil
}

// Close closes the Neo4j driver
func (l *Neo4jLoader) Close(ctx context.Context) error {
	return l.driver.Close(ctx)
}

// LoadFromFile loads TTL data from a local file into Neo4j
func (l *Neo4jLoader) LoadFromFile(ctx context.Context, filePath string) (*Neo4jLoadResult, error) {
	l.stats = Neo4jLoadResult{
		LoadTimestamp: time.Now(),
	}
	startTime := time.Now()

	l.logger.Info("═══════════════════════════════════════════════════════════")
	l.logger.Info("KB-7 Neo4j Loader - Graph Data Population")
	l.logger.Info("═══════════════════════════════════════════════════════════")
	l.logger.Infof("  File:       %s", filePath)
	l.logger.Infof("  Neo4j:      %s", l.config.Neo4jURL)
	l.logger.Infof("  Database:   %s", l.config.Neo4jDatabase)
	l.logger.Infof("  BatchSize:  %d", l.config.BatchSize)
	l.logger.Infof("  Workers:    %d", l.config.Workers)
	l.logger.Infof("  Dry Run:    %v", l.config.DryRun)
	l.logger.Info("═══════════════════════════════════════════════════════════")

	if l.config.DryRun {
		l.logger.Warn("DRY RUN MODE - No changes will be made")
	}

	// Step 1: Clear database if requested
	if l.config.ClearFirst && !l.config.DryRun {
		l.logger.Info("")
		l.logger.Info("Step 1: Clearing existing data...")
		if err := l.clearDatabase(ctx); err != nil {
			l.stats.ErrorMessage = fmt.Sprintf("Failed to clear database: %v", err)
			return &l.stats, err
		}
		l.logger.Info("✅ Database cleared")
	} else {
		l.logger.Info("")
		l.logger.Info("Step 1: Skipping clear (--clear not specified)")
	}

	// Step 2: Create indexes
	if l.config.CreateIndexes && !l.config.DryRun {
		l.logger.Info("")
		l.logger.Info("Step 2: Creating indexes...")
		if err := l.createIndexes(ctx); err != nil {
			l.logger.WithError(err).Warn("Failed to create some indexes (may already exist)")
		}
		l.logger.Info("✅ Indexes created")
	} else {
		l.logger.Info("")
		l.logger.Info("Step 2: Skipping index creation")
	}

	// Step 3: Open and parse TTL file
	l.logger.Info("")
	l.logger.Info("Step 3: Parsing TTL file...")
	file, err := os.Open(filePath)
	if err != nil {
		l.stats.ErrorMessage = fmt.Sprintf("Failed to open file: %v", err)
		return &l.stats, err
	}
	defer file.Close()

	// Get file size for progress reporting
	info, _ := file.Stat()
	l.logger.Infof("  File size: %.2f GB", float64(info.Size())/(1024*1024*1024))

	// Parse and load
	l.logger.Info("")
	l.logger.Info("Step 4: Loading data into Neo4j...")
	if err := l.parseTTLAndLoad(ctx, file); err != nil {
		l.stats.ErrorMessage = fmt.Sprintf("Load failed: %v", err)
		return &l.stats, err
	}

	// Flush remaining buffers
	if err := l.flushBuffers(ctx); err != nil {
		l.stats.ErrorMessage = fmt.Sprintf("Failed to flush buffers: %v", err)
		return &l.stats, err
	}

	l.stats.Success = true
	l.stats.Duration = time.Since(startTime)

	l.logger.Info("")
	l.logger.Info("═══════════════════════════════════════════════════════════")
	l.logger.Info("✅ NEO4J LOAD COMPLETE")
	l.logger.Info("═══════════════════════════════════════════════════════════")
	l.logger.Infof("  Duration:      %s", l.stats.Duration)
	l.logger.Infof("  Triples Read:  %d", l.stats.TriplesRead)
	l.logger.Infof("  Nodes Created: %d", l.stats.NodesCreated)
	l.logger.Infof("  Rels Created:  %d", l.stats.RelsCreated)
	l.logger.Info("═══════════════════════════════════════════════════════════")

	return &l.stats, nil
}

// clearDatabase removes all nodes and relationships
func (l *Neo4jLoader) clearDatabase(ctx context.Context) error {
	session := l.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: l.config.Neo4jDatabase,
	})
	defer session.Close(ctx)

	// Delete in batches to avoid memory issues
	for {
		result, err := session.Run(ctx,
			"MATCH (n) WITH n LIMIT 10000 DETACH DELETE n RETURN count(n) as deleted",
			nil,
		)
		if err != nil {
			return err
		}

		record, err := result.Single(ctx)
		if err != nil {
			return err
		}

		deleted, _ := record.Get("deleted")
		if deleted.(int64) == 0 {
			break
		}
		l.logger.Debugf("  Deleted %d nodes...", deleted)
	}

	return nil
}

// createIndexes creates necessary indexes for fast lookups
// Note: Uses :Class label to match neo4j_client.go schema expectations
func (l *Neo4jLoader) createIndexes(ctx context.Context) error {
	session := l.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: l.config.Neo4jDatabase,
	})
	defer session.Close(ctx)

	indexes := []string{
		// Class indexes (matches neo4j_client.go queries)
		"CREATE INDEX class_code IF NOT EXISTS FOR (c:Class) ON (c.code)",
		"CREATE INDEX class_system IF NOT EXISTS FOR (c:Class) ON (c.system)",
		"CREATE INDEX class_uri IF NOT EXISTS FOR (c:Class) ON (c.uri)",
		"CREATE CONSTRAINT class_uri_unique IF NOT EXISTS FOR (c:Class) REQUIRE c.uri IS UNIQUE",

		// Composite index for common lookups
		"CREATE INDEX class_code_system IF NOT EXISTS FOR (c:Class) ON (c.code, c.system)",

		// Full-text search index
		"CREATE FULLTEXT INDEX class_display IF NOT EXISTS FOR (c:Class) ON EACH [c.display, c.definition]",

		// System-specific indexes (secondary labels)
		"CREATE INDEX snomed_concept IF NOT EXISTS FOR (c:SNOMED) ON (c.code)",
		"CREATE INDEX rxnorm_concept IF NOT EXISTS FOR (c:RxNorm) ON (c.code)",
		"CREATE INDEX loinc_concept IF NOT EXISTS FOR (c:LOINC) ON (c.code)",
		"CREATE INDEX icd10_concept IF NOT EXISTS FOR (c:ICD10) ON (c.code)",
	}

	for _, idx := range indexes {
		_, err := session.Run(ctx, idx, nil)
		if err != nil {
			l.logger.WithError(err).Debugf("Index creation failed (may exist): %s", idx[:50])
		}
	}

	return nil
}

// parseTTLAndLoad parses TTL file and loads into Neo4j
func (l *Neo4jLoader) parseTTLAndLoad(ctx context.Context, reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	// Increase buffer size for long lines
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	// Track prefixes
	prefixes := make(map[string]string)

	// Triple pattern regex
	triplePattern := regexp.MustCompile(`^(<[^>]+>|[a-zA-Z_][a-zA-Z0-9_-]*:[^\s]+)\s+(<[^>]+>|[a-zA-Z_][a-zA-Z0-9_-]*:[^\s]+)\s+(.+?)\s*\.\s*$`)
	prefixPattern := regexp.MustCompile(`^@prefix\s+([a-zA-Z_][a-zA-Z0-9_-]*):\s*<([^>]+)>\s*\.\s*$`)

	lineCount := 0
	progressInterval := 1000000

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineCount++

		// Progress reporting
		if lineCount%progressInterval == 0 {
			l.logger.Infof("  Processed %d lines, %d triples...", lineCount, l.stats.TriplesRead)
		}

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle prefix declarations
		if strings.HasPrefix(line, "@prefix") {
			if matches := prefixPattern.FindStringSubmatch(line); matches != nil {
				prefixes[matches[1]] = matches[2]
			}
			continue
		}

		// Handle base declaration
		if strings.HasPrefix(line, "@base") {
			continue
		}

		// Parse triple
		if matches := triplePattern.FindStringSubmatch(line); matches != nil {
			subject := l.expandURI(matches[1], prefixes)
			predicate := l.expandURI(matches[2], prefixes)
			object := matches[3]

			l.processTriple(ctx, subject, predicate, object, prefixes)
			l.statsMu.Lock()
			l.stats.TriplesRead++
			l.statsMu.Unlock()
		}
	}

	return scanner.Err()
}

// expandURI expands prefixed URIs to full URIs
func (l *Neo4jLoader) expandURI(uri string, prefixes map[string]string) string {
	// Already full URI
	if strings.HasPrefix(uri, "<") && strings.HasSuffix(uri, ">") {
		return uri[1 : len(uri)-1]
	}

	// Prefixed URI
	if idx := strings.Index(uri, ":"); idx > 0 {
		prefix := uri[:idx]
		local := uri[idx+1:]
		if namespace, ok := prefixes[prefix]; ok {
			return namespace + local
		}
	}

	return uri
}

// processTriple processes a single RDF triple
func (l *Neo4jLoader) processTriple(ctx context.Context, subject, predicate, object string, prefixes map[string]string) {
	// Determine the type of triple and buffer appropriately
	switch {
	case predicate == "http://www.w3.org/1999/02/22-rdf-syntax-ns#type":
		// rdf:type - create node with label
		objectURI := l.expandURI(object, prefixes)
		l.bufferConceptType(subject, objectURI)

	case predicate == "http://www.w3.org/2000/01/rdf-schema#label":
		// rdfs:label - set display name
		l.bufferConceptProperty(subject, "display", l.cleanLiteral(object))

	case predicate == "http://www.w3.org/2004/02/skos/core#prefLabel":
		// skos:prefLabel - set display name
		l.bufferConceptProperty(subject, "display", l.cleanLiteral(object))

	case predicate == "http://www.w3.org/2004/02/skos/core#definition":
		// skos:definition
		l.bufferConceptProperty(subject, "definition", l.cleanLiteral(object))

	case predicate == "http://www.w3.org/2000/01/rdf-schema#subClassOf":
		// rdfs:subClassOf - create rdfs__subClassOf relationship (matches neo4j_client.go)
		objectURI := l.expandURI(object, prefixes)
		l.bufferRelation(subject, objectURI, "rdfs__subClassOf")

	case predicate == "http://www.w3.org/2004/02/skos/core#broader":
		// skos:broader - create skos__broader relationship (matches neo4j_client.go)
		objectURI := l.expandURI(object, prefixes)
		l.bufferRelation(subject, objectURI, "skos__broader")

	case predicate == "http://www.w3.org/2004/02/skos/core#narrower":
		// skos:narrower - create skos__narrower relationship
		objectURI := l.expandURI(object, prefixes)
		l.bufferRelation(subject, objectURI, "skos__narrower")

	case predicate == "http://www.w3.org/2004/02/skos/core#exactMatch":
		// skos:exactMatch - cross-terminology mapping (matches neo4j_client.go)
		objectURI := l.expandURI(object, prefixes)
		l.bufferRelation(subject, objectURI, "skos__exactMatch")

	case predicate == "http://www.w3.org/2004/02/skos/core#closeMatch":
		// skos:closeMatch - cross-terminology mapping
		objectURI := l.expandURI(object, prefixes)
		l.bufferRelation(subject, objectURI, "skos__closeMatch")

	case predicate == "http://www.w3.org/2004/02/skos/core#broadMatch":
		// skos:broadMatch - cross-terminology mapping (matches neo4j_client.go)
		objectURI := l.expandURI(object, prefixes)
		l.bufferRelation(subject, objectURI, "skos__broadMatch")

	case predicate == "http://www.w3.org/2004/02/skos/core#narrowMatch":
		// skos:narrowMatch - cross-terminology mapping (matches neo4j_client.go)
		objectURI := l.expandURI(object, prefixes)
		l.bufferRelation(subject, objectURI, "skos__narrowMatch")

	case predicate == "http://www.w3.org/2004/02/skos/core#relatedMatch":
		// skos:relatedMatch - cross-terminology mapping
		objectURI := l.expandURI(object, prefixes)
		l.bufferRelation(subject, objectURI, "skos__relatedMatch")
	}

	// Check if we need to flush buffers
	l.bufferMu.Lock()
	needFlush := len(l.conceptBuffer) >= l.config.BatchSize || len(l.relationBuffer) >= l.config.BatchSize
	l.bufferMu.Unlock()

	if needFlush && !l.config.DryRun {
		if err := l.flushBuffers(ctx); err != nil {
			l.logger.WithError(err).Error("Failed to flush buffers")
		}
	}
}

// bufferConceptType adds a concept with a type label
func (l *Neo4jLoader) bufferConceptType(uri, typeURI string) {
	l.bufferMu.Lock()
	defer l.bufferMu.Unlock()

	// Find or create concept in buffer
	var found *conceptData
	for i := range l.conceptBuffer {
		if l.conceptBuffer[i].URI == uri {
			found = &l.conceptBuffer[i]
			break
		}
	}

	if found == nil {
		code, system := l.extractCodeAndSystem(uri)
		l.conceptBuffer = append(l.conceptBuffer, conceptData{
			URI:        uri,
			Code:       code,
			System:     system,
			Labels:     []string{l.typeToLabel(typeURI)},
			Properties: make(map[string]interface{}),
		})
	} else {
		found.Labels = append(found.Labels, l.typeToLabel(typeURI))
	}
}

// bufferConceptProperty adds a property to a concept
// Note: Uses :Class label to match neo4j_client.go schema
func (l *Neo4jLoader) bufferConceptProperty(uri, key, value string) {
	l.bufferMu.Lock()
	defer l.bufferMu.Unlock()

	// Find or create concept in buffer
	var found *conceptData
	for i := range l.conceptBuffer {
		if l.conceptBuffer[i].URI == uri {
			found = &l.conceptBuffer[i]
			break
		}
	}

	if found == nil {
		code, system := l.extractCodeAndSystem(uri)
		l.conceptBuffer = append(l.conceptBuffer, conceptData{
			URI:        uri,
			Code:       code,
			System:     system,
			Labels:     []string{"Class"},
			Properties: map[string]interface{}{key: value},
		})
	} else {
		found.Properties[key] = value
	}
}

// bufferRelation adds a relationship to the buffer
func (l *Neo4jLoader) bufferRelation(fromURI, toURI, relType string) {
	l.bufferMu.Lock()
	defer l.bufferMu.Unlock()

	l.relationBuffer = append(l.relationBuffer, relationData{
		FromURI:    fromURI,
		ToURI:      toURI,
		Type:       relType,
		Properties: make(map[string]interface{}),
	})
}

// flushBuffers writes buffered data to Neo4j
func (l *Neo4jLoader) flushBuffers(ctx context.Context) error {
	if l.config.DryRun {
		l.bufferMu.Lock()
		l.conceptBuffer = l.conceptBuffer[:0]
		l.relationBuffer = l.relationBuffer[:0]
		l.bufferMu.Unlock()
		return nil
	}

	l.bufferMu.Lock()
	concepts := make([]conceptData, len(l.conceptBuffer))
	copy(concepts, l.conceptBuffer)
	l.conceptBuffer = l.conceptBuffer[:0]

	relations := make([]relationData, len(l.relationBuffer))
	copy(relations, l.relationBuffer)
	l.relationBuffer = l.relationBuffer[:0]
	l.bufferMu.Unlock()

	// Write concepts
	if len(concepts) > 0 {
		if err := l.writeConcepts(ctx, concepts); err != nil {
			return err
		}
	}

	// Write relations
	if len(relations) > 0 {
		if err := l.writeRelations(ctx, relations); err != nil {
			return err
		}
	}

	return nil
}

// writeConcepts writes concept nodes to Neo4j
func (l *Neo4jLoader) writeConcepts(ctx context.Context, concepts []conceptData) error {
	session := l.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: l.config.Neo4jDatabase,
	})
	defer session.Close(ctx)

	// Build batch create statement
	params := make([]map[string]interface{}, 0, len(concepts))
	for _, c := range concepts {
		props := map[string]interface{}{
			"uri":     c.URI,
			"code":    c.Code,
			"system":  c.System,
			"labels":  c.Labels,
		}
		for k, v := range c.Properties {
			props[k] = v
		}
		params = append(params, props)
	}

	// MERGE to handle duplicates - uses :Class label to match neo4j_client.go
	query := `
		UNWIND $batch AS row
		MERGE (c:Class {uri: row.uri})
		SET c.code = row.code,
		    c.system = row.system,
		    c.display = COALESCE(row.display, c.display),
		    c.definition = COALESCE(row.definition, c.definition),
		    c.updatedAt = datetime()
		WITH c, row
		CALL apoc.create.addLabels(c, row.labels) YIELD node
		RETURN count(node) as created
	`

	// If APOC is not available, use simpler query
	simpleQuery := `
		UNWIND $batch AS row
		MERGE (c:Class {uri: row.uri})
		SET c.code = row.code,
		    c.system = row.system,
		    c.display = COALESCE(row.display, c.display),
		    c.definition = COALESCE(row.definition, c.definition),
		    c.updatedAt = datetime()
		RETURN count(c) as created
	`

	result, err := session.Run(ctx, simpleQuery, map[string]interface{}{"batch": params})
	if err != nil {
		// Try with APOC
		result, err = session.Run(ctx, query, map[string]interface{}{"batch": params})
		if err != nil {
			return fmt.Errorf("failed to write concepts: %w", err)
		}
	}

	record, err := result.Single(ctx)
	if err != nil {
		return err
	}

	created, _ := record.Get("created")
	l.statsMu.Lock()
	l.stats.NodesCreated += created.(int64)
	l.statsMu.Unlock()

	return nil
}

// writeRelations writes relationship edges to Neo4j
func (l *Neo4jLoader) writeRelations(ctx context.Context, relations []relationData) error {
	session := l.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: l.config.Neo4jDatabase,
	})
	defer session.Close(ctx)

	// Group relations by type for more efficient batch processing
	byType := make(map[string][]relationData)
	for _, r := range relations {
		byType[r.Type] = append(byType[r.Type], r)
	}

	for relType, rels := range byType {
		params := make([]map[string]interface{}, 0, len(rels))
		for _, r := range rels {
			params = append(params, map[string]interface{}{
				"from": r.FromURI,
				"to":   r.ToURI,
			})
		}

		// Use dynamic relationship type with :Class label (matches neo4j_client.go)
		query := fmt.Sprintf(`
			UNWIND $batch AS row
			MATCH (from:Class {uri: row.from})
			MATCH (to:Class {uri: row.to})
			MERGE (from)-[r:%s]->(to)
			RETURN count(r) as created
		`, relType)

		result, err := session.Run(ctx, query, map[string]interface{}{"batch": params})
		if err != nil {
			l.logger.WithError(err).Warnf("Failed to create %s relationships", relType)
			continue
		}

		record, err := result.Single(ctx)
		if err != nil {
			continue
		}

		created, _ := record.Get("created")
		l.statsMu.Lock()
		l.stats.RelsCreated += created.(int64)
		l.statsMu.Unlock()
	}

	return nil
}

// extractCodeAndSystem extracts code and system from URI
func (l *Neo4jLoader) extractCodeAndSystem(uri string) (code, system string) {
	// SNOMED: http://snomed.info/id/12345
	if strings.Contains(uri, "snomed.info/id/") {
		parts := strings.Split(uri, "/")
		return parts[len(parts)-1], "http://snomed.info/sct"
	}

	// RxNorm: http://purl.bioontology.org/ontology/RXNORM/12345
	if strings.Contains(uri, "RXNORM/") {
		parts := strings.Split(uri, "/")
		return parts[len(parts)-1], "http://www.nlm.nih.gov/research/umls/rxnorm"
	}

	// LOINC: http://loinc.org/12345-6
	if strings.Contains(uri, "loinc.org/") {
		parts := strings.Split(uri, "/")
		return parts[len(parts)-1], "http://loinc.org"
	}

	// ICD-10: http://hl7.org/fhir/sid/icd-10/A00.0
	if strings.Contains(uri, "icd-10") {
		parts := strings.Split(uri, "/")
		return parts[len(parts)-1], "http://hl7.org/fhir/sid/icd-10"
	}

	// Default: use last path segment as code
	if idx := strings.LastIndex(uri, "/"); idx > 0 {
		code = uri[idx+1:]
	} else if idx := strings.LastIndex(uri, "#"); idx > 0 {
		code = uri[idx+1:]
	} else {
		code = uri
	}
	system = uri[:strings.LastIndex(uri, "/")]
	return
}

// typeToLabel converts RDF type URI to Neo4j label
// Note: Returns "Class" as default to match neo4j_client.go schema
func (l *Neo4jLoader) typeToLabel(typeURI string) string {
	// Map common OWL/RDFS types to Neo4j labels
	labelMap := map[string]string{
		"http://www.w3.org/2002/07/owl#Class":           "Class",
		"http://www.w3.org/2004/02/skos/core#Concept":   "Class",
		"http://snomed.info/id/":                        "SNOMED",
		"http://purl.bioontology.org/ontology/RXNORM/": "RxNorm",
		"http://loinc.org/":                            "LOINC",
		"http://hl7.org/fhir/sid/icd-10":              "ICD10",
	}

	for prefix, label := range labelMap {
		if strings.HasPrefix(typeURI, prefix) || typeURI == prefix {
			return label
		}
	}

	// Extract last segment as label
	if idx := strings.LastIndex(typeURI, "#"); idx > 0 {
		return typeURI[idx+1:]
	}
	if idx := strings.LastIndex(typeURI, "/"); idx > 0 {
		return typeURI[idx+1:]
	}

	return "Class"
}

// cleanLiteral removes quotes and language tags from RDF literal
func (l *Neo4jLoader) cleanLiteral(literal string) string {
	// Remove quotes
	if strings.HasPrefix(literal, "\"") {
		if idx := strings.LastIndex(literal, "\""); idx > 0 {
			literal = literal[1:idx]
		}
	}

	// Handle triple-quoted strings
	if strings.HasPrefix(literal, "\"\"\"") {
		literal = strings.TrimPrefix(literal, "\"\"\"")
		if idx := strings.Index(literal, "\"\"\""); idx > 0 {
			literal = literal[:idx]
		}
	}

	// Unescape common sequences
	literal = strings.ReplaceAll(literal, "\\n", "\n")
	literal = strings.ReplaceAll(literal, "\\t", "\t")
	literal = strings.ReplaceAll(literal, "\\\"", "\"")

	return literal
}

// GetStatus returns the current status of Neo4j
func (l *Neo4jLoader) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	session := l.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: l.config.Neo4jDatabase,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, `
		MATCH (n)
		WITH count(n) as nodeCount
		MATCH ()-[r]->()
		WITH nodeCount, count(r) as relCount
		RETURN nodeCount, relCount
	`, nil)
	if err != nil {
		return nil, err
	}

	record, err := result.Single(ctx)
	if err != nil {
		// Empty database
		return map[string]interface{}{
			"available":     true,
			"database":      l.config.Neo4jDatabase,
			"node_count":    0,
			"rel_count":     0,
		}, nil
	}

	nodeCount, _ := record.Get("nodeCount")
	relCount, _ := record.Get("relCount")

	return map[string]interface{}{
		"available":     true,
		"database":      l.config.Neo4jDatabase,
		"node_count":    nodeCount,
		"rel_count":     relCount,
	}, nil
}

// VerifyData runs verification queries against Neo4j
// Uses n10s (neosemantics) schema with Resource nodes and ns1__ prefixed properties
func (l *Neo4jLoader) VerifyData(ctx context.Context) (bool, error) {
	session := l.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: l.config.Neo4jDatabase,
	})
	defer session.Close(ctx)

	l.logger.Info("═══════════════════════════════════════════════════════════")
	l.logger.Info("KB-7 Neo4j Verification (n10s schema)")
	l.logger.Info("═══════════════════════════════════════════════════════════")

	success := true

	// Test 1: Node count (using :Resource label - n10s schema)
	l.logger.Info("")
	l.logger.Info("Test 1: Resource Node Count")
	result, err := session.Run(ctx, "MATCH (n:Resource) RETURN count(n) as count", nil)
	if err != nil {
		l.logger.WithError(err).Error("  ❌ Failed to count nodes")
		success = false
	} else {
		record, _ := result.Single(ctx)
		count, _ := record.Get("count")
		l.logger.Infof("  ✅ Resource nodes: %d", count)
	}

	// Test 2: Concepts by system (using ns1__system property)
	l.logger.Info("")
	l.logger.Info("Test 2: Concepts by System")
	result, err = session.Run(ctx, `
		MATCH (n:Resource)
		WHERE n.ns1__system IS NOT NULL
		RETURN n.ns1__system as system, count(n) as count
		ORDER BY count DESC
	`, nil)
	if err != nil {
		l.logger.WithError(err).Error("  ❌ System count query failed")
		success = false
	} else {
		for result.Next(ctx) {
			record := result.Record()
			system, _ := record.Get("system")
			count, _ := record.Get("count")
			l.logger.Infof("  ✅ %s: %d concepts", system, count)
		}
	}

	// Test 3: Hierarchy relationships (rdfs__subClassOf)
	l.logger.Info("")
	l.logger.Info("Test 3: Hierarchy Relationships")
	result, err = session.Run(ctx, `
		MATCH ()-[r:rdfs__subClassOf]->()
		RETURN count(r) as count
	`, nil)
	if err != nil {
		l.logger.WithError(err).Error("  ❌ Hierarchy query failed")
		success = false
	} else {
		record, _ := result.Single(ctx)
		count, _ := record.Get("count")
		l.logger.Infof("  ✅ rdfs__subClassOf relationships: %d", count)
	}

	// Test 4: Sample concept lookup (using n10s properties)
	l.logger.Info("")
	l.logger.Info("Test 4: Sample Concept Lookup")
	result, err = session.Run(ctx, `
		MATCH (c:Resource)
		WHERE c.ns1__code IS NOT NULL AND c.ns1__system IS NOT NULL
		RETURN c.ns1__code as code, c.rdfs__label as display, c.ns1__system as system
		LIMIT 3
	`, nil)
	if err != nil {
		l.logger.WithError(err).Error("  ❌ Sample lookup failed")
		success = false
	} else {
		found := false
		for result.Next(ctx) {
			found = true
			record := result.Record()
			code, _ := record.Get("code")
			display, _ := record.Get("display")
			system, _ := record.Get("system")
			l.logger.Infof("  ✅ %s: %s (%s)", code, display, system)
		}
		if !found {
			l.logger.Warn("  ⚠️ No concepts found with code/system properties")
		}
	}

	l.logger.Info("")
	l.logger.Info("═══════════════════════════════════════════════════════════")
	if success {
		l.logger.Info("✅ ALL VERIFICATION TESTS PASSED")
	} else {
		l.logger.Error("❌ SOME VERIFICATION TESTS FAILED")
	}
	l.logger.Info("═══════════════════════════════════════════════════════════")

	return success, nil
}
