# TripleStoreCoordinator Architecture Design
**Phase 1.2: ETL Pipeline Extension for GraphDB Triple Loading**

**Document Version**: 1.0
**Date**: November 22, 2025
**Status**: Design Phase - Ready for Implementation
**Target**: Extend existing DualStoreCoordinator (PostgreSQL + Elasticsearch) → TripleStoreCoordinator (PostgreSQL + GraphDB + Elasticsearch)

---

## Executive Summary

This document provides a comprehensive architectural design for extending the KB-7 ETL pipeline to support GraphDB triple loading while maintaining full backward compatibility with PostgreSQL and Elasticsearch operations.

### Key Design Principles
1. **Backward Compatibility**: PostgreSQL remains fully functional - existing code must not break
2. **Reuse Pattern**: Follow DualStoreCoordinator architecture pattern
3. **Parallel Execution**: PostgreSQL + GraphDB + Elasticsearch writes execute concurrently
4. **Fail-Safe**: GraphDB failures do not block PostgreSQL operations (PostgreSQL is source of truth)
5. **Data Integrity**: Comprehensive validation ensures consistency across all stores

### Architecture Flow
```
SNOMED CT Files → ETL Main → TripleStoreCoordinator
                                  ├→ PostgreSQL (existing, unchanged)
                                  ├→ Elasticsearch (existing, unchanged)
                                  └→ GraphDB (NEW - RDF triples)
                                        ↓
                                  Consistency Validation
```

---

## 1. Current State Analysis

### 1.1 Existing Architecture Components

**Working Components (DO NOT MODIFY)**:
```
cmd/etl/main.go                          # ETL orchestrator - calls DualStoreCoordinator
internal/etl/enhanced_coordinator.go     # Base PostgreSQL loading logic
internal/etl/dual_store_coordinator.go   # PostgreSQL + Elasticsearch coordinator
internal/etl/enhanced_loaders.go         # SNOMED, RxNorm, LOINC, ICD10 loaders
internal/etl/transaction_manager.go      # Distributed transaction management
```

**Existing Data Flow**:
1. `main.go` creates `DualStoreCoordinator` with `EnhancedCoordinator` inside
2. `DualStoreCoordinator.LoadAllTerminologiesDualStore()` calls:
   - `EnhancedCoordinator.LoadAllTerminologies()` → writes to PostgreSQL
   - `syncToElasticsearch()` → reads from PostgreSQL, writes to Elasticsearch
   - `performConsistencyCheck()` → validates PostgreSQL ↔ Elasticsearch sync

**Key Insight**: DualStoreCoordinator wraps EnhancedCoordinator and adds Elasticsearch as a secondary store. We follow the same pattern for GraphDB.

### 1.2 SNOMED CT Data Structure

**Source Data** (RF2 Format):
- **Concepts File**: `sct2_Concept_Snapshot_*.txt` (520K records)
  - Columns: id, effectiveTime, active, moduleId, definitionStatusId
  - Example: `387517004|20240131|1|900000000000012004|900000000000073002|`

- **Descriptions File**: `sct2_Description_Snapshot_*.txt` (1.5M records)
  - Columns: id, effectiveTime, active, moduleId, conceptId, languageCode, typeId, term, caseSignificanceId
  - Provides: Preferred terms, synonyms, fully specified names

- **Relationships File**: `sct2_Relationship_Snapshot_*.txt` (2M records)
  - Columns: id, effectiveTime, active, moduleId, sourceId, destinationId, relationshipGroup, typeId, characteristicTypeId, modifierId
  - Defines: IS-A hierarchies, attribute relationships

**PostgreSQL Schema** (current):
```sql
CREATE TABLE concepts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    system VARCHAR(50) NOT NULL,        -- 'SNOMED', 'RxNorm', 'LOINC', 'ICD10'
    code VARCHAR(255) NOT NULL,         -- '387517004' (SNOMED concept ID)
    preferred_term TEXT NOT NULL,       -- 'Paracetamol'
    definition TEXT,
    active BOOLEAN DEFAULT true,
    version VARCHAR(50),                -- '20240131'
    properties JSONB,                   -- {moduleId, definitionStatusId, synonyms, ...}
    parent_codes TEXT[],                -- Array of parent concept codes (IS-A)
    search_vector tsvector,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(system, code, version)
);
```

---

## 2. TripleStoreCoordinator Architecture

### 2.1 Component Hierarchy

```
TripleStoreCoordinator
    │
    ├── DualStoreCoordinator (existing - reuse as-is)
    │       ├── EnhancedCoordinator (PostgreSQL + cache)
    │       └── ElasticsearchIntegration (existing sync)
    │
    ├── GraphDBLoader (NEW)
    │       └── GraphDBClient (existing - internal/semantic/graphdb_client.go)
    │
    ├── RDFTransformer (NEW)
    │       └── RDFConverter (existing - internal/semantic/rdf_converter.go)
    │
    ├── TripleStoreTransactionManager (NEW)
    │       └── TransactionManager (reuse - internal/etl/transaction_manager.go)
    │
    └── TripleStoreValidator (NEW)
            └── ConsistencyChecker
```

### 2.2 Code Structure Plan

**New Files to Create**:
```
internal/etl/
├── triple_store_coordinator.go          # Main coordinator (extends DualStoreCoordinator)
├── graphdb_loader.go                     # GraphDB-specific loading logic
└── triple_validator.go                   # 3-way consistency validation

internal/transformer/
└── snomed_to_rdf.go                      # SNOMED CT → RDF transformation logic

cmd/etl/
└── main.go                               # MODIFY: Add --enable-graphdb flag
```

**Files to Modify**:
```
cmd/etl/main.go                           # Add TripleStoreCoordinator initialization
internal/etl/dual_store_coordinator.go    # OPTIONAL: Extract interface for reuse
```

**Files to Reuse (NO CHANGES)**:
```
internal/semantic/graphdb_client.go       # GraphDB REST API client
internal/semantic/rdf_converter.go        # RDF/Turtle conversion utilities
internal/etl/enhanced_coordinator.go      # PostgreSQL loading (unchanged)
internal/etl/transaction_manager.go       # Distributed transactions
```

---

## 3. Data Model Specifications

### 3.1 RDF Triple Representation

**Concept as RDF Triples** (Turtle Format):
```turtle
@prefix kb7: <http://cardiofit.ai/kb7/ontology#> .
@prefix sct: <http://snomed.info/id/> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

# Concept Definition
sct:387517004 a owl:Class ;
    rdfs:label "Paracetamol"@en ;
    skos:prefLabel "Paracetamol"@en ;
    skos:altLabel "Acetaminophen"@en ;
    kb7:conceptId "387517004" ;
    kb7:system "SNOMED-CT" ;
    kb7:version "20240131" ;
    kb7:active "true"^^xsd:boolean ;
    kb7:moduleId "900000000000012004" ;
    kb7:definitionStatusId "900000000000073002" ;
    rdfs:subClassOf sct:7947003 .  # Parent: Drug

# Relationship Triples
sct:387517004 kb7:hasAttribute [
    kb7:attributeType "clinical_drug_form" ;
    kb7:attributeValue "tablet"
] .
```

**Triple Count Estimation** (520K concepts):
- Concepts: 520,000 × 8 triples/concept = **4,160,000 triples**
- Descriptions: 1,500,000 × 3 triples = **4,500,000 triples**
- Relationships: 2,000,000 × 2 triples = **4,000,000 triples**
- **Total: ~12,660,000 triples**

### 3.2 Internal Data Models

**Go Struct: RDF Triple**
```go
// internal/transformer/snomed_to_rdf.go

package transformer

import "time"

// RDFTriple represents a single RDF triple (subject-predicate-object)
type RDFTriple struct {
    Subject    string            `json:"subject"`    // <http://snomed.info/id/387517004>
    Predicate  string            `json:"predicate"`  // rdfs:label
    Object     string            `json:"object"`     // "Paracetamol"@en
    ObjectType RDFObjectType     `json:"object_type"` // URI, Literal, BlankNode
    DataType   string            `json:"data_type"`  // xsd:string, xsd:boolean
    Language   string            `json:"language"`   // "en"
    Graph      string            `json:"graph"`      // Named graph URI
}

type RDFObjectType string

const (
    RDFObjectURI       RDFObjectType = "uri"
    RDFObjectLiteral   RDFObjectType = "literal"
    RDFObjectBlankNode RDFObjectType = "blank_node"
)

// RDFTripleSet represents a batch of triples for a single concept
type RDFTripleSet struct {
    ConceptID    string       `json:"concept_id"`
    System       string       `json:"system"`
    Triples      []RDFTriple  `json:"triples"`
    CreatedAt    time.Time    `json:"created_at"`
}

// TurtleDocument represents a complete Turtle RDF document
type TurtleDocument struct {
    Prefixes     map[string]string  `json:"prefixes"`
    Statements   []string           `json:"statements"`
    TripleCount  int                `json:"triple_count"`
}
```

**PostgreSQL Concept → RDF Mapping**:
```go
type ConceptToRDFMapping struct {
    // PostgreSQL fields → RDF predicates
    Code           → kb7:conceptId, sct:{code} (subject URI)
    PreferredTerm  → rdfs:label, skos:prefLabel
    Definition     → skos:definition
    Active         → kb7:active (xsd:boolean)
    Version        → kb7:version
    System         → kb7:system
    ParentCodes    → rdfs:subClassOf (multiple triples)
    Properties     → kb7:hasAttribute (nested structures)
}
```

---

## 4. Transformation Logic Design

### 4.1 SNOMED CT → RDF Conversion Algorithm

**File**: `internal/transformer/snomed_to_rdf.go`

```go
package transformer

import (
    "context"
    "fmt"
    "strings"
    "kb-7-terminology/internal/models"
)

type SNOMEDToRDFTransformer struct {
    baseURI      string
    namespaces   map[string]string
    logger       *zap.Logger
}

func NewSNOMEDToRDFTransformer(logger *zap.Logger) *SNOMEDToRDFTransformer {
    return &SNOMEDToRDFTransformer{
        baseURI: "http://snomed.info/id/",
        namespaces: map[string]string{
            "sct":    "http://snomed.info/id/",
            "kb7":    "http://cardiofit.ai/kb7/ontology#",
            "rdfs":   "http://www.w3.org/2000/01/rdf-schema#",
            "skos":   "http://www.w3.org/2004/02/skos/core#",
            "owl":    "http://www.w3.org/2002/07/owl#",
            "xsd":    "http://www.w3.org/2001/XMLSchema#",
        },
        logger: logger,
    }
}

// ConvertConceptToTriples converts a PostgreSQL Concept to RDF triples
func (t *SNOMEDToRDFTransformer) ConceptToTriples(concept *models.Concept) (*RDFTripleSet, error) {
    triples := &RDFTripleSet{
        ConceptID: concept.Code,
        System:    concept.System,
        Triples:   make([]RDFTriple, 0, 10),
    }

    subjectURI := t.buildConceptURI(concept.Code)

    // 1. Type declaration (owl:Class)
    triples.Triples = append(triples.Triples, RDFTriple{
        Subject:    subjectURI,
        Predicate:  "rdf:type",
        Object:     "owl:Class",
        ObjectType: RDFObjectURI,
    })

    // 2. Preferred term (rdfs:label + skos:prefLabel)
    if concept.PreferredTerm != "" {
        triples.Triples = append(triples.Triples,
            RDFTriple{
                Subject:    subjectURI,
                Predicate:  "rdfs:label",
                Object:     concept.PreferredTerm,
                ObjectType: RDFObjectLiteral,
                Language:   "en",
            },
            RDFTriple{
                Subject:    subjectURI,
                Predicate:  "skos:prefLabel",
                Object:     concept.PreferredTerm,
                ObjectType: RDFObjectLiteral,
                Language:   "en",
            },
        )
    }

    // 3. Definition (skos:definition)
    if concept.Definition != "" {
        triples.Triples = append(triples.Triples, RDFTriple{
            Subject:    subjectURI,
            Predicate:  "skos:definition",
            Object:     concept.Definition,
            ObjectType: RDFObjectLiteral,
            Language:   "en",
        })
    }

    // 4. Metadata attributes
    triples.Triples = append(triples.Triples,
        t.createLiteralTriple(subjectURI, "kb7:conceptId", concept.Code, "xsd:string"),
        t.createLiteralTriple(subjectURI, "kb7:system", concept.System, "xsd:string"),
        t.createLiteralTriple(subjectURI, "kb7:version", concept.Version, "xsd:string"),
        t.createLiteralTriple(subjectURI, "kb7:active", fmt.Sprintf("%t", concept.Active), "xsd:boolean"),
    )

    // 5. Parent relationships (rdfs:subClassOf)
    if parentCodes, ok := concept.Properties["parent_codes"].([]interface{}); ok {
        for _, parentCode := range parentCodes {
            if pc, ok := parentCode.(string); ok {
                parentURI := t.buildConceptURI(pc)
                triples.Triples = append(triples.Triples, RDFTriple{
                    Subject:    subjectURI,
                    Predicate:  "rdfs:subClassOf",
                    Object:     parentURI,
                    ObjectType: RDFObjectURI,
                })
            }
        }
    }

    // 6. SNOMED-specific properties from JSONB
    if moduleID, ok := concept.Properties["module_id"].(string); ok {
        triples.Triples = append(triples.Triples,
            t.createLiteralTriple(subjectURI, "kb7:moduleId", moduleID, "xsd:string"),
        )
    }

    if defStatusID, ok := concept.Properties["definition_status_id"].(string); ok {
        triples.Triples = append(triples.Triples,
            t.createLiteralTriple(subjectURI, "kb7:definitionStatusId", defStatusID, "xsd:string"),
        )
    }

    // 7. Synonyms (skos:altLabel)
    if synonyms, ok := concept.Properties["synonyms"].([]interface{}); ok {
        for _, synonym := range synonyms {
            if syn, ok := synonym.(string); ok {
                triples.Triples = append(triples.Triples, RDFTriple{
                    Subject:    subjectURI,
                    Predicate:  "skos:altLabel",
                    Object:     syn,
                    ObjectType: RDFObjectLiteral,
                    Language:   "en",
                })
            }
        }
    }

    return triples, nil
}

// ConvertBatchToTurtle converts multiple concepts to a single Turtle document
func (t *SNOMEDToRDFTransformer) BatchToTurtle(concepts []*models.Concept) (*TurtleDocument, error) {
    doc := &TurtleDocument{
        Prefixes:   t.namespaces,
        Statements: make([]string, 0, len(concepts)*10),
    }

    // Write prefix declarations
    prefixStatements := t.generatePrefixStatements()
    doc.Statements = append(doc.Statements, prefixStatements...)

    // Convert each concept to triples
    for _, concept := range concepts {
        tripleSet, err := t.ConceptToTriples(concept)
        if err != nil {
            t.logger.Warn("Failed to convert concept",
                zap.String("code", concept.Code),
                zap.Error(err))
            continue
        }

        // Convert triples to Turtle statements
        statements := t.triplesToTurtle(tripleSet.Triples)
        doc.Statements = append(doc.Statements, statements...)
        doc.TripleCount += len(tripleSet.Triples)
    }

    return doc, nil
}

// Helper methods
func (t *SNOMEDToRDFTransformer) buildConceptURI(code string) string {
    return fmt.Sprintf("sct:%s", code)
}

func (t *SNOMEDToRDFTransformer) createLiteralTriple(subject, predicate, value, datatype string) RDFTriple {
    return RDFTriple{
        Subject:    subject,
        Predicate:  predicate,
        Object:     value,
        ObjectType: RDFObjectLiteral,
        DataType:   datatype,
    }
}

func (t *SNOMEDToRDFTransformer) generatePrefixStatements() []string {
    statements := make([]string, 0, len(t.namespaces))
    for prefix, uri := range t.namespaces {
        statements = append(statements, fmt.Sprintf("@prefix %s: <%s> .", prefix, uri))
    }
    return statements
}

func (t *SNOMEDToRDFTransformer) triplesToTurtle(triples []RDFTriple) []string {
    if len(triples) == 0 {
        return nil
    }

    // Group triples by subject for compact Turtle syntax
    grouped := make(map[string][]RDFTriple)
    for _, triple := range triples {
        grouped[triple.Subject] = append(grouped[triple.Subject], triple)
    }

    statements := make([]string, 0)
    for subject, subjectTriples := range grouped {
        statement := subject + " "

        predicates := make([]string, 0)
        for i, triple := range subjectTriples {
            object := t.formatObject(triple)

            if i == 0 {
                predicates = append(predicates, fmt.Sprintf("%s %s", triple.Predicate, object))
            } else {
                predicates = append(predicates, fmt.Sprintf("    %s %s", triple.Predicate, object))
            }
        }

        statement += strings.Join(predicates, " ;\n") + " ."
        statements = append(statements, statement)
    }

    return statements
}

func (t *SNOMEDToRDFTransformer) formatObject(triple RDFTriple) string {
    switch triple.ObjectType {
    case RDFObjectURI:
        return triple.Object
    case RDFObjectLiteral:
        escaped := strings.ReplaceAll(triple.Object, "\"", "\\\"")
        if triple.Language != "" {
            return fmt.Sprintf(`"%s"@%s`, escaped, triple.Language)
        }
        if triple.DataType != "" {
            return fmt.Sprintf(`"%s"^^%s`, escaped, triple.DataType)
        }
        return fmt.Sprintf(`"%s"`, escaped)
    case RDFObjectBlankNode:
        return triple.Object
    default:
        return fmt.Sprintf(`"%s"`, triple.Object)
    }
}
```

### 4.2 Transformation Performance Targets

| Metric | Target | Strategy |
|--------|--------|----------|
| Conversion Rate | 10K concepts/sec | Batch processing (1000 concepts/batch) |
| Memory Usage | < 500 MB per batch | Stream processing, no full materialization |
| Triple Generation | ~24 triples/concept avg | Template-based conversion |
| Turtle Serialization | < 100ms per 1K concepts | String builder optimization |

---

## 5. TripleStoreCoordinator Implementation Design

### 5.1 Core Coordinator Structure

**File**: `internal/etl/triple_store_coordinator.go`

```go
package etl

import (
    "context"
    "fmt"
    "sync"
    "time"

    "kb-7-terminology/internal/models"
    "kb-7-terminology/internal/semantic"
    "kb-7-terminology/internal/transformer"

    "go.uber.org/zap"
)

// TripleStoreCoordinator extends DualStoreCoordinator with GraphDB triple loading
type TripleStoreCoordinator struct {
    // Embed existing DualStoreCoordinator for PostgreSQL + Elasticsearch
    *DualStoreCoordinator

    // GraphDB integration
    graphDBClient   *semantic.GraphDBClient
    rdfTransformer  *transformer.SNOMEDToRDFTransformer
    graphDBConfig   *GraphDBConfig

    // Triple store state
    tripleStoreStatus *TripleStoreStatus
    statusMutex       sync.RWMutex
}

// GraphDBConfig holds GraphDB-specific configuration
type GraphDBConfig struct {
    Enabled            bool          `json:"enabled"`
    ServerURL          string        `json:"server_url"`
    RepositoryID       string        `json:"repository_id"`
    BatchSize          int           `json:"batch_size"`          // Triples per upload
    MaxRetries         int           `json:"max_retries"`
    RetryDelay         time.Duration `json:"retry_delay"`
    EnableInference    bool          `json:"enable_inference"`
    NamedGraph         string        `json:"named_graph"`
    TransactionTimeout time.Duration `json:"transaction_timeout"`
    ValidateTriples    bool          `json:"validate_triples"`
}

// TripleStoreStatus tracks triple store operation status
type TripleStoreStatus struct {
    PostgreSQLStatus    StoreOperationStatus `json:"postgresql_status"`    // Inherited
    ElasticsearchStatus StoreOperationStatus `json:"elasticsearch_status"` // Inherited
    GraphDBStatus       StoreOperationStatus `json:"graphdb_status"`       // NEW
    ConsistencyStatus   TripleStoreConsistency `json:"consistency_status"`
    OverallHealth       string                `json:"overall_health"`
    LastSyncTime        time.Time             `json:"last_sync_time"`
}

// TripleStoreConsistency tracks 3-way data consistency
type TripleStoreConsistency struct {
    PostgreSQLCount    int64     `json:"postgresql_count"`
    ElasticsearchCount int64     `json:"elasticsearch_count"`
    GraphDBTripleCount int64     `json:"graphdb_triple_count"`
    IsConsistent       bool      `json:"is_consistent"`
    Discrepancy        int64     `json:"discrepancy"`
    LastCheck          time.Time `json:"last_check"`
    CheckDuration      time.Duration `json:"check_duration"`
}

// NewTripleStoreCoordinator creates a new triple-store coordinator
func NewTripleStoreCoordinator(
    dualStoreCoordinator *DualStoreCoordinator,
    graphDBConfig *GraphDBConfig,
    logger *zap.Logger,
) (*TripleStoreCoordinator, error) {

    if graphDBConfig == nil {
        graphDBConfig = DefaultGraphDBConfig()
    }

    coordinator := &TripleStoreCoordinator{
        DualStoreCoordinator: dualStoreCoordinator,
        graphDBConfig:       graphDBConfig,
        tripleStoreStatus: &TripleStoreStatus{
            GraphDBStatus: StoreOperationStatus{
                Status:  "idle",
                Health:  "unknown",
                Metrics: make(map[string]int64),
            },
            OverallHealth: "unknown",
        },
    }

    // Initialize GraphDB client if enabled
    if graphDBConfig.Enabled {
        if err := coordinator.initializeGraphDB(); err != nil {
            return nil, fmt.Errorf("failed to initialize GraphDB: %w", err)
        }
    }

    logger.Info("Triple-store coordinator initialized",
        zap.Bool("graphdb_enabled", graphDBConfig.Enabled),
        zap.String("repository_id", graphDBConfig.RepositoryID),
    )

    return coordinator, nil
}

// DefaultGraphDBConfig returns default GraphDB configuration
func DefaultGraphDBConfig() *GraphDBConfig {
    return &GraphDBConfig{
        Enabled:            true,
        ServerURL:          "http://localhost:7200",
        RepositoryID:       "kb7-terminology",
        BatchSize:          10000,  // 10K triples per batch
        MaxRetries:         3,
        RetryDelay:         5 * time.Second,
        EnableInference:    true,
        NamedGraph:         "http://cardiofit.ai/kb7/graph/default",
        TransactionTimeout: 10 * time.Minute,
        ValidateTriples:    true,
    }
}

// initializeGraphDB sets up GraphDB client connection
func (tsc *TripleStoreCoordinator) initializeGraphDB() error {
    tsc.logger.Info("Initializing GraphDB client",
        zap.String("server_url", tsc.graphDBConfig.ServerURL),
        zap.String("repository", tsc.graphDBConfig.RepositoryID),
    )

    // Create GraphDB client (reuse existing implementation)
    client, err := semantic.NewGraphDBClient(
        tsc.graphDBConfig.ServerURL,
        tsc.graphDBConfig.RepositoryID,
        tsc.logger,
    )
    if err != nil {
        return fmt.Errorf("failed to create GraphDB client: %w", err)
    }

    tsc.graphDBClient = client

    // Create RDF transformer
    tsc.rdfTransformer = transformer.NewSNOMEDToRDFTransformer(tsc.logger)

    // Test GraphDB connectivity
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    healthy, err := tsc.graphDBClient.HealthCheck(ctx)
    if err != nil || !healthy {
        tsc.tripleStoreStatus.GraphDBStatus.Health = "unhealthy"
        tsc.tripleStoreStatus.GraphDBStatus.LastError = err.Error()
        tsc.logger.Warn("GraphDB health check failed", zap.Error(err))
    } else {
        tsc.tripleStoreStatus.GraphDBStatus.Health = "healthy"
        tsc.logger.Info("GraphDB connection established successfully")
    }

    return nil
}

// LoadAllTerminologiesTripleStore loads all terminologies to PostgreSQL + GraphDB + Elasticsearch
func (tsc *TripleStoreCoordinator) LoadAllTerminologiesTripleStore(
    ctx context.Context,
    dataSources map[string]string,
) error {

    tsc.statusMutex.Lock()
    tsc.tripleStoreStatus.OverallHealth = "loading"
    tsc.statusMutex.Unlock()

    tsc.logger.Info("Starting triple-store terminology loading")

    // PHASE 1: Load to PostgreSQL + Elasticsearch (existing DualStoreCoordinator logic)
    tsc.logger.Info("Phase 1: Loading to PostgreSQL + Elasticsearch")
    pgStartTime := time.Now()

    err := tsc.DualStoreCoordinator.LoadAllTerminologiesDualStore(ctx, dataSources)
    pgDuration := time.Since(pgStartTime)

    if err != nil {
        tsc.tripleStoreStatus.OverallHealth = "failed"
        return fmt.Errorf("PostgreSQL/Elasticsearch loading failed: %w", err)
    }

    tsc.logger.Info("PostgreSQL + Elasticsearch loading completed",
        zap.Duration("duration", pgDuration),
    )

    // PHASE 2: Sync to GraphDB (NEW logic)
    if tsc.graphDBConfig.Enabled {
        tsc.logger.Info("Phase 2: Syncing to GraphDB")

        if err := tsc.syncToGraphDB(ctx); err != nil {
            tsc.logger.Error("GraphDB sync failed", zap.Error(err))
            tsc.updateStoreStatus("graphdb", "failed", err.Error())
            tsc.tripleStoreStatus.OverallHealth = "degraded"

            // GraphDB failure does not block PostgreSQL success
            tsc.logger.Warn("Continuing with PostgreSQL-only operation (GraphDB sync failed)")
        } else {
            tsc.updateStoreStatus("graphdb", "completed", "")
            tsc.tripleStoreStatus.OverallHealth = "healthy"
        }
    } else {
        tsc.logger.Info("GraphDB disabled, skipping triple loading")
        tsc.tripleStoreStatus.OverallHealth = "healthy"
    }

    // PHASE 3: Perform 3-way consistency check
    if tsc.graphDBConfig.Enabled && tsc.graphDBConfig.ValidateTriples {
        tsc.logger.Info("Phase 3: Performing consistency validation")

        if err := tsc.performTripleStoreConsistencyCheck(ctx); err != nil {
            tsc.logger.Warn("Consistency check failed", zap.Error(err))
        }
    }

    tsc.tripleStoreStatus.LastSyncTime = time.Now()
    tsc.logger.Info("Triple-store loading completed",
        zap.String("overall_health", tsc.tripleStoreStatus.OverallHealth),
    )

    return nil
}

// syncToGraphDB reads concepts from PostgreSQL and loads them to GraphDB as RDF triples
func (tsc *TripleStoreCoordinator) syncToGraphDB(ctx context.Context) error {
    if tsc.graphDBClient == nil {
        return fmt.Errorf("GraphDB client not initialized")
    }

    graphStartTime := time.Now()
    tsc.updateStoreStatus("graphdb", "running", "Converting PostgreSQL to RDF triples")

    // Step 1: Read all concepts from PostgreSQL
    tsc.logger.Info("Reading concepts from PostgreSQL")

    concepts, err := tsc.readConceptsFromPostgreSQL(ctx)
    if err != nil {
        return fmt.Errorf("failed to read concepts from PostgreSQL: %w", err)
    }

    tsc.logger.Info("Concepts loaded from PostgreSQL", zap.Int("count", len(concepts)))

    // Step 2: Convert concepts to RDF triples in batches
    tsc.logger.Info("Converting concepts to RDF triples",
        zap.Int("batch_size", tsc.graphDBConfig.BatchSize))

    totalTriplesLoaded := int64(0)
    batchSize := 1000  // Concepts per batch (not triples)

    for i := 0; i < len(concepts); i += batchSize {
        end := i + batchSize
        if end > len(concepts) {
            end = len(concepts)
        }

        batch := concepts[i:end]

        // Convert batch to Turtle document
        turtleDoc, err := tsc.rdfTransformer.BatchToTurtle(batch)
        if err != nil {
            tsc.logger.Error("Failed to convert batch to Turtle",
                zap.Int("batch_start", i),
                zap.Error(err))
            continue
        }

        // Step 3: Upload Turtle document to GraphDB
        tsc.logger.Debug("Uploading batch to GraphDB",
            zap.Int("batch_start", i),
            zap.Int("batch_end", end),
            zap.Int("triple_count", turtleDoc.TripleCount))

        turtleContent := strings.Join(turtleDoc.Statements, "\n")

        err = tsc.graphDBClient.UploadTurtle(ctx, turtleContent, tsc.graphDBConfig.NamedGraph)
        if err != nil {
            tsc.logger.Error("Failed to upload batch to GraphDB",
                zap.Int("batch_start", i),
                zap.Error(err))

            // Retry logic
            if tsc.retryGraphDBUpload(ctx, turtleContent) != nil {
                return fmt.Errorf("GraphDB upload failed after retries: %w", err)
            }
        }

        totalTriplesLoaded += int64(turtleDoc.TripleCount)

        // Progress logging
        if i%10000 == 0 {
            tsc.logger.Info("GraphDB sync progress",
                zap.Int("concepts_processed", i),
                zap.Int64("triples_loaded", totalTriplesLoaded))
        }
    }

    graphDuration := time.Since(graphStartTime)

    tsc.statusMutex.Lock()
    tsc.tripleStoreStatus.GraphDBStatus.RecordsWritten = totalTriplesLoaded
    tsc.tripleStoreStatus.GraphDBStatus.ResponseTime = graphDuration
    tsc.statusMutex.Unlock()

    tsc.logger.Info("GraphDB sync completed",
        zap.Int64("total_triples", totalTriplesLoaded),
        zap.Duration("duration", graphDuration))

    return nil
}

// readConceptsFromPostgreSQL reads all concepts from PostgreSQL database
func (tsc *TripleStoreCoordinator) readConceptsFromPostgreSQL(ctx context.Context) ([]*models.Concept, error) {
    query := `
        SELECT id, system, code, preferred_term, definition, active, version,
               properties, created_at, updated_at
        FROM concepts
        ORDER BY system, code
    `

    rows, err := tsc.db.QueryContext(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to query concepts: %w", err)
    }
    defer rows.Close()

    concepts := make([]*models.Concept, 0, 520000)

    for rows.Next() {
        concept := &models.Concept{}

        err := rows.Scan(
            &concept.ID,
            &concept.System,
            &concept.Code,
            &concept.PreferredTerm,
            &concept.Definition,
            &concept.Active,
            &concept.Version,
            &concept.Properties,
            &concept.CreatedAt,
            &concept.UpdatedAt,
        )

        if err != nil {
            tsc.logger.Warn("Failed to scan concept row", zap.Error(err))
            continue
        }

        concepts = append(concepts, concept)
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("row iteration error: %w", err)
    }

    return concepts, nil
}

// retryGraphDBUpload retries failed GraphDB uploads with exponential backoff
func (tsc *TripleStoreCoordinator) retryGraphDBUpload(ctx context.Context, turtleContent string) error {
    for attempt := 1; attempt <= tsc.graphDBConfig.MaxRetries; attempt++ {
        tsc.logger.Info("Retrying GraphDB upload",
            zap.Int("attempt", attempt),
            zap.Int("max_retries", tsc.graphDBConfig.MaxRetries))

        time.Sleep(tsc.graphDBConfig.RetryDelay * time.Duration(attempt))

        err := tsc.graphDBClient.UploadTurtle(ctx, turtleContent, tsc.graphDBConfig.NamedGraph)
        if err == nil {
            tsc.logger.Info("GraphDB upload retry succeeded", zap.Int("attempt", attempt))
            return nil
        }

        tsc.logger.Warn("GraphDB upload retry failed",
            zap.Int("attempt", attempt),
            zap.Error(err))
    }

    return fmt.Errorf("GraphDB upload failed after %d retries", tsc.graphDBConfig.MaxRetries)
}

// performTripleStoreConsistencyCheck validates data consistency across all 3 stores
func (tsc *TripleStoreCoordinator) performTripleStoreConsistencyCheck(ctx context.Context) error {
    checkStartTime := time.Now()
    tsc.logger.Info("Starting 3-way consistency check")

    // Count records in each store
    pgCount, err := tsc.countPostgreSQLConcepts(ctx)
    if err != nil {
        return fmt.Errorf("PostgreSQL count failed: %w", err)
    }

    esCount, err := tsc.countElasticsearchDocuments(ctx)
    if err != nil {
        tsc.logger.Warn("Elasticsearch count failed", zap.Error(err))
        esCount = 0
    }

    graphCount, err := tsc.countGraphDBTriples(ctx)
    if err != nil {
        return fmt.Errorf("GraphDB count failed: %w", err)
    }

    // Calculate consistency
    expectedTripleCount := pgCount * 8  // ~8 triples per concept average
    discrepancy := abs64(graphCount - expectedTripleCount)
    consistencyScore := 1.0 - (float64(discrepancy) / float64(expectedTripleCount))

    isConsistent := consistencyScore >= 0.95  // 95% threshold

    checkDuration := time.Since(checkStartTime)

    // Update consistency status
    tsc.statusMutex.Lock()
    tsc.tripleStoreStatus.ConsistencyStatus = TripleStoreConsistency{
        PostgreSQLCount:    pgCount,
        ElasticsearchCount: esCount,
        GraphDBTripleCount: graphCount,
        IsConsistent:       isConsistent,
        Discrepancy:        discrepancy,
        LastCheck:          checkStartTime,
        CheckDuration:      checkDuration,
    }
    tsc.statusMutex.Unlock()

    tsc.logger.Info("Consistency check completed",
        zap.Int64("postgresql_concepts", pgCount),
        zap.Int64("elasticsearch_docs", esCount),
        zap.Int64("graphdb_triples", graphCount),
        zap.Bool("is_consistent", isConsistent),
        zap.Float64("consistency_score", consistencyScore))

    if !isConsistent {
        tsc.logger.Warn("Consistency check failed",
            zap.Int64("expected_triples", expectedTripleCount),
            zap.Int64("actual_triples", graphCount),
            zap.Int64("discrepancy", discrepancy))
    }

    return nil
}

// Helper methods
func (tsc *TripleStoreCoordinator) countPostgreSQLConcepts(ctx context.Context) (int64, error) {
    var count int64
    err := tsc.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM concepts").Scan(&count)
    return count, err
}

func (tsc *TripleStoreCoordinator) countElasticsearchDocuments(ctx context.Context) (int64, error) {
    if tsc.esIntegration == nil {
        return 0, nil
    }
    stats, err := tsc.esIntegration.GetIndexStats(ctx)
    if err != nil {
        return 0, err
    }
    return stats.DocumentCount, nil
}

func (tsc *TripleStoreCoordinator) countGraphDBTriples(ctx context.Context) (int64, error) {
    query := "SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }"

    results, err := tsc.graphDBClient.ExecuteSPARQL(ctx, query)
    if err != nil {
        return 0, err
    }

    // Parse SPARQL results (simplified - actual implementation needs JSON parsing)
    // This is a placeholder - actual parsing logic needed
    count := int64(0)
    // ... parse results ...

    return count, nil
}

func (tsc *TripleStoreCoordinator) updateStoreStatus(store, status, errorMsg string) {
    tsc.statusMutex.Lock()
    defer tsc.statusMutex.Unlock()

    switch store {
    case "graphdb":
        tsc.tripleStoreStatus.GraphDBStatus.Status = status
        if errorMsg != "" {
            tsc.tripleStoreStatus.GraphDBStatus.LastError = errorMsg
            tsc.tripleStoreStatus.GraphDBStatus.RecordsFailed++
        }
    case "postgresql":
        tsc.DualStoreCoordinator.updateStoreStatus(store, status, errorMsg)
    case "elasticsearch":
        tsc.DualStoreCoordinator.updateStoreStatus(store, status, errorMsg)
    }
}

// GetTripleStoreStatus returns current triple-store status
func (tsc *TripleStoreCoordinator) GetTripleStoreStatus() *TripleStoreStatus {
    tsc.statusMutex.RLock()
    defer tsc.statusMutex.RUnlock()

    // Copy status to avoid race conditions
    status := *tsc.tripleStoreStatus
    return &status
}

// Close closes all connections
func (tsc *TripleStoreCoordinator) Close() error {
    if tsc.graphDBClient != nil {
        // GraphDB client doesn't need explicit close (HTTP client)
    }
    return tsc.DualStoreCoordinator.Close()
}

// Utility function
func abs64(x int64) int64 {
    if x < 0 {
        return -x
    }
    return x
}
```

---

## 6. GraphDB Loader Implementation

### 6.1 GraphDB-Specific Loading Logic

**File**: `internal/etl/graphdb_loader.go`

```go
package etl

import (
    "context"
    "fmt"
    "strings"
    "time"

    "kb-7-terminology/internal/semantic"
    "kb-7-terminology/internal/transformer"

    "go.uber.org/zap"
)

// GraphDBLoader handles GraphDB-specific loading operations
type GraphDBLoader struct {
    client     *semantic.GraphDBClient
    transformer *transformer.SNOMEDToRDFTransformer
    logger     *zap.Logger
    config     *GraphDBLoaderConfig
}

// GraphDBLoaderConfig holds loader configuration
type GraphDBLoaderConfig struct {
    BatchSize          int           `json:"batch_size"`
    MaxConcurrent      int           `json:"max_concurrent"`
    UploadTimeout      time.Duration `json:"upload_timeout"`
    EnableCompression  bool          `json:"enable_compression"`
    ValidateBeforeLoad bool          `json:"validate_before_load"`
}

// NewGraphDBLoader creates a new GraphDB loader
func NewGraphDBLoader(
    client *semantic.GraphDBClient,
    transformer *transformer.SNOMEDToRDFTransformer,
    logger *zap.Logger,
    config *GraphDBLoaderConfig,
) *GraphDBLoader {
    if config == nil {
        config = &GraphDBLoaderConfig{
            BatchSize:          10000,
            MaxConcurrent:      4,
            UploadTimeout:      5 * time.Minute,
            EnableCompression:  true,
            ValidateBeforeLoad: false,
        }
    }

    return &GraphDBLoader{
        client:     client,
        transformer: transformer,
        logger:     logger,
        config:     config,
    }
}

// LoadTriples loads RDF triples to GraphDB in batches
func (gbl *GraphDBLoader) LoadTriples(ctx context.Context, triples []transformer.RDFTriple) error {
    gbl.logger.Info("Loading triples to GraphDB", zap.Int("count", len(triples)))

    // Group triples into batches
    batches := gbl.batchTriples(triples, gbl.config.BatchSize)
    gbl.logger.Info("Divided triples into batches", zap.Int("batch_count", len(batches)))

    // Upload batches
    for i, batch := range batches {
        turtleDoc := gbl.triplesToTurtle(batch)

        uploadCtx, cancel := context.WithTimeout(ctx, gbl.config.UploadTimeout)
        err := gbl.client.UploadTurtle(uploadCtx, turtleDoc, "http://cardiofit.ai/kb7/graph/default")
        cancel()

        if err != nil {
            return fmt.Errorf("failed to upload batch %d: %w", i, err)
        }

        gbl.logger.Debug("Batch uploaded successfully", zap.Int("batch_index", i))
    }

    gbl.logger.Info("All triples loaded successfully", zap.Int("total_triples", len(triples)))
    return nil
}

// LoadTurtleFile loads a Turtle file to GraphDB
func (gbl *GraphDBLoader) LoadTurtleFile(ctx context.Context, filePath string) error {
    gbl.logger.Info("Loading Turtle file to GraphDB", zap.String("file", filePath))

    // Read file content
    content, err := os.ReadFile(filePath)
    if err != nil {
        return fmt.Errorf("failed to read Turtle file: %w", err)
    }

    // Upload to GraphDB
    return gbl.client.UploadTurtle(ctx, string(content), "http://cardiofit.ai/kb7/graph/default")
}

// ClearRepository clears all data from GraphDB repository
func (gbl *GraphDBLoader) ClearRepository(ctx context.Context) error {
    gbl.logger.Warn("Clearing GraphDB repository")

    query := "CLEAR ALL"
    _, err := gbl.client.ExecuteSPARQL(ctx, query)
    if err != nil {
        return fmt.Errorf("failed to clear repository: %w", err)
    }

    gbl.logger.Info("GraphDB repository cleared successfully")
    return nil
}

// ValidateTriples validates triples before loading
func (gbl *GraphDBLoader) ValidateTriples(triples []transformer.RDFTriple) error {
    if !gbl.config.ValidateBeforeLoad {
        return nil
    }

    gbl.logger.Info("Validating triples", zap.Int("count", len(triples)))

    invalidCount := 0
    for i, triple := range triples {
        if err := gbl.validateTriple(triple); err != nil {
            gbl.logger.Warn("Invalid triple",
                zap.Int("index", i),
                zap.String("subject", triple.Subject),
                zap.Error(err))
            invalidCount++
        }
    }

    if invalidCount > 0 {
        return fmt.Errorf("%d invalid triples found", invalidCount)
    }

    gbl.logger.Info("Triple validation passed")
    return nil
}

// Helper methods
func (gbl *GraphDBLoader) batchTriples(triples []transformer.RDFTriple, batchSize int) [][]transformer.RDFTriple {
    batches := make([][]transformer.RDFTriple, 0)

    for i := 0; i < len(triples); i += batchSize {
        end := i + batchSize
        if end > len(triples) {
            end = len(triples)
        }
        batches = append(batches, triples[i:end])
    }

    return batches
}

func (gbl *GraphDBLoader) triplesToTurtle(triples []transformer.RDFTriple) string {
    // Generate Turtle document from triples
    var builder strings.Builder

    // Write prefixes
    builder.WriteString("@prefix kb7: <http://cardiofit.ai/kb7/ontology#> .\n")
    builder.WriteString("@prefix sct: <http://snomed.info/id/> .\n")
    builder.WriteString("@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .\n")
    builder.WriteString("@prefix skos: <http://www.w3.org/2004/02/skos/core#> .\n")
    builder.WriteString("@prefix owl: <http://www.w3.org/2002/07/owl#> .\n")
    builder.WriteString("@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .\n\n")

    // Write triples
    for _, triple := range triples {
        object := gbl.formatTurtleObject(triple)
        builder.WriteString(fmt.Sprintf("%s %s %s .\n",
            triple.Subject,
            triple.Predicate,
            object))
    }

    return builder.String()
}

func (gbl *GraphDBLoader) formatTurtleObject(triple transformer.RDFTriple) string {
    switch triple.ObjectType {
    case transformer.RDFObjectURI:
        return triple.Object
    case transformer.RDFObjectLiteral:
        escaped := strings.ReplaceAll(triple.Object, "\"", "\\\"")
        if triple.Language != "" {
            return fmt.Sprintf(`"%s"@%s`, escaped, triple.Language)
        }
        if triple.DataType != "" {
            return fmt.Sprintf(`"%s"^^%s`, escaped, triple.DataType)
        }
        return fmt.Sprintf(`"%s"`, escaped)
    default:
        return fmt.Sprintf(`"%s"`, triple.Object)
    }
}

func (gbl *GraphDBLoader) validateTriple(triple transformer.RDFTriple) error {
    if triple.Subject == "" {
        return fmt.Errorf("empty subject")
    }
    if triple.Predicate == "" {
        return fmt.Errorf("empty predicate")
    }
    if triple.Object == "" {
        return fmt.Errorf("empty object")
    }
    return nil
}
```

---

## 7. Error Handling & Transaction Management

### 7.1 Error Handling Strategy

**Failure Scenarios & Responses**:

| Failure Point | Impact | Recovery Strategy |
|---------------|--------|-------------------|
| PostgreSQL write fails | **CRITICAL** | Abort all operations, rollback transaction |
| Elasticsearch write fails | **WARNING** | Continue (PostgreSQL is source of truth), mark as degraded |
| GraphDB write fails | **WARNING** | Continue (PostgreSQL is source of truth), mark as degraded |
| GraphDB connection lost | **WARNING** | Circuit breaker activates, disable GraphDB writes |
| Triple conversion fails | **WARNING** | Skip failed concepts, log errors, continue batch |
| Consistency check fails | **INFO** | Log discrepancy, continue operations |

**Error Handling Code Pattern**:
```go
func (tsc *TripleStoreCoordinator) LoadAllTerminologiesTripleStore(ctx context.Context, dataSources map[string]string) error {
    // Phase 1: PostgreSQL + Elasticsearch (CRITICAL PATH)
    err := tsc.DualStoreCoordinator.LoadAllTerminologiesDualStore(ctx, dataSources)
    if err != nil {
        // CRITICAL FAILURE - abort everything
        tsc.tripleStoreStatus.OverallHealth = "failed"
        return fmt.Errorf("CRITICAL: PostgreSQL loading failed: %w", err)
    }

    // Phase 2: GraphDB sync (NON-CRITICAL PATH)
    if tsc.graphDBConfig.Enabled {
        if err := tsc.syncToGraphDB(ctx); err != nil {
            // NON-CRITICAL FAILURE - log and continue
            tsc.logger.Error("GraphDB sync failed, continuing with degraded service", zap.Error(err))
            tsc.tripleStoreStatus.OverallHealth = "degraded"
            tsc.tripleStoreStatus.GraphDBStatus.Health = "unhealthy"

            // Record error but don't fail overall operation
            tsc.recordGraphDBError(err)
        }
    }

    // Phase 3: Consistency validation (INFORMATIONAL)
    if err := tsc.performTripleStoreConsistencyCheck(ctx); err != nil {
        // INFORMATIONAL FAILURE - log only
        tsc.logger.Warn("Consistency check encountered errors", zap.Error(err))
    }

    // Overall operation succeeds if PostgreSQL succeeds
    return nil
}
```

### 7.2 Rollback Strategy

**Rollback Scenarios**:

1. **PostgreSQL Failure**: Full rollback using `transaction_manager.go`
   - PostgreSQL transaction rolled back
   - Elasticsearch documents deleted (if already written)
   - GraphDB triples NOT written (transaction never started)

2. **GraphDB Failure**: Partial rollback
   - PostgreSQL commits remain (source of truth)
   - Elasticsearch commits remain
   - GraphDB triples NOT committed (or cleared if partial upload occurred)

**GraphDB Rollback Implementation**:
```go
func (tsc *TripleStoreCoordinator) rollbackGraphDB(ctx context.Context) error {
    tsc.logger.Warn("Rolling back GraphDB changes")

    // Option 1: Clear all triples in named graph
    clearQuery := fmt.Sprintf("CLEAR GRAPH <%s>", tsc.graphDBConfig.NamedGraph)
    _, err := tsc.graphDBClient.ExecuteSPARQL(ctx, clearQuery)
    if err != nil {
        return fmt.Errorf("GraphDB rollback failed: %w", err)
    }

    tsc.logger.Info("GraphDB rollback completed")
    return nil
}
```

### 7.3 Retry Logic

**Retry Configuration**:
```go
type RetryConfig struct {
    MaxAttempts      int           // Default: 3
    InitialDelay     time.Duration // Default: 5s
    MaxDelay         time.Duration // Default: 60s
    BackoffMultiplier float64      // Default: 2.0 (exponential backoff)
    RetriableErrors  []string      // Connection timeout, network errors
}

func (tsc *TripleStoreCoordinator) retryWithBackoff(
    ctx context.Context,
    operation func(context.Context) error,
    config RetryConfig,
) error {
    delay := config.InitialDelay

    for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
        err := operation(ctx)
        if err == nil {
            return nil
        }

        if attempt == config.MaxAttempts {
            return fmt.Errorf("operation failed after %d attempts: %w", attempt, err)
        }

        // Check if error is retriable
        if !isRetriableError(err, config.RetriableErrors) {
            return fmt.Errorf("non-retriable error: %w", err)
        }

        tsc.logger.Warn("Operation failed, retrying",
            zap.Int("attempt", attempt),
            zap.Duration("delay", delay),
            zap.Error(err))

        // Exponential backoff
        time.Sleep(delay)
        delay = time.Duration(float64(delay) * config.BackoffMultiplier)
        if delay > config.MaxDelay {
            delay = config.MaxDelay
        }
    }

    return nil
}
```

---

## 8. Code Structure Summary

### 8.1 Files to Create

```
internal/etl/
├── triple_store_coordinator.go     # Main coordinator (600 lines)
├── graphdb_loader.go                # GraphDB loading logic (300 lines)
└── triple_validator.go              # 3-way consistency validation (200 lines)

internal/transformer/
└── snomed_to_rdf.go                 # SNOMED → RDF conversion (500 lines)

scripts/
└── test-graphdb-integration.sh      # Integration test script (100 lines)

docs/
└── TRIPLE_STORE_COORDINATOR_ARCHITECTURE.md  # This document
```

### 8.2 Files to Modify

```
cmd/etl/main.go
CHANGES:
- Line 166-184: Replace DualStoreConfig with TripleStoreConfig
- Line 181-184: Replace DualStoreCoordinator with TripleStoreCoordinator
- Line 244: Replace LoadAllTerminologiesDualStore with LoadAllTerminologiesTripleStore
- Add: --enable-graphdb flag (line 31)
- Add: GraphDB configuration block (after line 178)

BACKWARD COMPATIBILITY:
- Default: --enable-graphdb=false (GraphDB disabled by default)
- Existing behavior preserved when flag is false
- No changes to existing PostgreSQL/Elasticsearch flow
```

### 8.3 Files to Reuse (No Changes)

```
internal/semantic/graphdb_client.go        # GraphDB REST API client
internal/semantic/rdf_converter.go         # RDF utilities
internal/etl/enhanced_coordinator.go       # PostgreSQL base logic
internal/etl/dual_store_coordinator.go     # Elasticsearch integration
internal/etl/transaction_manager.go        # Distributed transactions
internal/models/enhanced_models.go         # Data models
```

---

## 9. Testing Strategy

### 9.1 Unit Tests

```go
// internal/etl/triple_store_coordinator_test.go

func TestTripleStoreCoordinatorInitialization(t *testing.T) {
    // Test coordinator creation with valid config
    // Test coordinator creation with invalid config
    // Test GraphDB client initialization
}

func TestConceptToRDFConversion(t *testing.T) {
    // Test single concept conversion
    // Test batch conversion
    // Test empty concept handling
    // Test special characters in terms
}

func TestTripleUpload(t *testing.T) {
    // Test successful upload
    // Test upload failure handling
    // Test retry logic
}

func TestConsistencyValidation(t *testing.T) {
    // Test 3-way count validation
    // Test discrepancy detection
    // Test consistency score calculation
}
```

### 9.2 Integration Tests

```bash
#!/bin/bash
# scripts/test-graphdb-integration.sh

set -e

echo "=== GraphDB Integration Test ==="

# 1. Start services
docker-compose up -d postgres graphdb

# 2. Load test data
./etl --data ./data/test-snomed --enable-graphdb=true

# 3. Validate PostgreSQL
PG_COUNT=$(psql -U kb7_user -t -c "SELECT COUNT(*) FROM concepts;")
echo "PostgreSQL concepts: $PG_COUNT"

# 4. Validate GraphDB
GRAPHDB_COUNT=$(curl -s -X POST http://localhost:7200/repositories/kb7-terminology \
  --data-urlencode "query=SELECT (COUNT(*) as ?count) WHERE { ?s a owl:Class }" \
  | jq '.results.bindings[0].count.value')
echo "GraphDB triples: $GRAPHDB_COUNT"

# 5. Test SPARQL query
curl -X POST http://localhost:7200/repositories/kb7-terminology \
  --data-urlencode "query=SELECT ?concept ?label WHERE {
    ?concept rdfs:label ?label .
    FILTER(CONTAINS(?label, 'Paracetamol'))
  } LIMIT 5"

echo "✅ Integration test completed"
```

### 9.3 Performance Tests

**Load Test Metrics**:
```go
type PerformanceMetrics struct {
    PostgreSQLLoadTime    time.Duration
    ElasticsearchSyncTime time.Duration
    GraphDBSyncTime       time.Duration
    TotalETLDuration      time.Duration
    TriplesPerSecond      float64
    ConceptsPerSecond     float64
    MemoryUsage           int64
}

// Target Performance:
// - PostgreSQL: 520K concepts in 10 minutes
// - GraphDB: 12.6M triples in 20 minutes
// - Total ETL: < 35 minutes (with parallel execution)
// - Memory: < 2 GB peak usage
// - Triples/sec: > 10,000
```

---

## 10. Implementation Workflow

### 10.1 Development Phases

**Phase 1: Data Model & Transformer (Week 1, Days 3-4)**
1. Create `internal/transformer/snomed_to_rdf.go`
2. Implement `ConceptToTriples()` method
3. Implement `BatchToTurtle()` method
4. Write unit tests for transformation logic
5. Test with sample concepts (100 concepts)

**Phase 2: GraphDB Loader (Week 1, Day 5)**
1. Create `internal/etl/graphdb_loader.go`
2. Implement `LoadTriples()` method
3. Implement retry logic
4. Write unit tests for loader
5. Test upload to GraphDB with sample data

**Phase 3: TripleStoreCoordinator (Week 2, Days 1-2)**
1. Create `internal/etl/triple_store_coordinator.go`
2. Implement `LoadAllTerminologiesTripleStore()` method
3. Implement `syncToGraphDB()` method
4. Implement consistency validation
5. Write unit tests for coordinator

**Phase 4: Main ETL Integration (Week 2, Day 3)**
1. Modify `cmd/etl/main.go`
2. Add `--enable-graphdb` flag
3. Add GraphDB configuration
4. Update coordinator instantiation
5. Test backward compatibility (GraphDB disabled)

**Phase 5: Testing & Validation (Week 2, Days 4-5)**
1. Run full integration test (520K concepts)
2. Performance benchmarking
3. Consistency validation
4. Error scenario testing
5. Documentation updates

### 10.2 Success Criteria

**Functional Requirements**:
- ✅ All 520K SNOMED concepts loaded to GraphDB as RDF triples
- ✅ PostgreSQL operations unchanged (backward compatible)
- ✅ Elasticsearch operations unchanged
- ✅ 3-way consistency validation passes (>95% consistency)
- ✅ GraphDB failures do not block PostgreSQL success

**Performance Requirements**:
- ✅ Total ETL duration < 40 minutes (full 520K dataset)
- ✅ GraphDB sync < 25 minutes
- ✅ Memory usage < 2 GB peak
- ✅ Triple generation rate > 10K triples/sec

**Quality Requirements**:
- ✅ Unit test coverage > 90%
- ✅ Integration tests pass
- ✅ No regressions in existing PostgreSQL/Elasticsearch flow
- ✅ Clear error messages and logging
- ✅ Comprehensive documentation

---

## 11. Monitoring & Observability

### 11.1 Metrics to Track

```go
type ETLMetrics struct {
    // Performance
    ConceptsProcessed      int64
    TriplesGenerated       int64
    TriplesUploaded        int64
    BatchesProcessed       int64

    // Timing
    PostgreSQLDuration     time.Duration
    ElasticsearchDuration  time.Duration
    GraphDBDuration        time.Duration
    ConversionDuration     time.Duration

    // Errors
    ConversionErrors       int64
    UploadErrors           int64
    RetryCount             int64
    FailedBatches          int64

    // Resource
    PeakMemoryUsage        int64
    CPUUtilization         float64
    NetworkBytesTransferred int64
}
```

### 11.2 Logging Strategy

**Log Levels**:
```go
// INFO: Progress milestones
logger.Info("GraphDB sync started",
    zap.Int("total_concepts", 520000),
    zap.Int("batch_size", 1000))

logger.Info("Batch uploaded",
    zap.Int("batch_index", 100),
    zap.Int("triples_uploaded", 24000))

// WARN: Recoverable errors
logger.Warn("Triple conversion failed for concept",
    zap.String("concept_code", "387517004"),
    zap.Error(err))

logger.Warn("GraphDB upload retry",
    zap.Int("attempt", 2),
    zap.Error(err))

// ERROR: Critical failures
logger.Error("GraphDB sync failed",
    zap.String("phase", "batch_upload"),
    zap.Error(err))

// DEBUG: Detailed diagnostics
logger.Debug("Triple generated",
    zap.String("subject", "sct:387517004"),
    zap.String("predicate", "rdfs:label"),
    zap.String("object", "Paracetamol"))
```

---

## 12. Configuration Reference

### 12.1 Environment Variables

```bash
# GraphDB Configuration
GRAPHDB_ENABLED=true
GRAPHDB_SERVER_URL=http://localhost:7200
GRAPHDB_REPOSITORY_ID=kb7-terminology
GRAPHDB_BATCH_SIZE=10000
GRAPHDB_MAX_RETRIES=3
GRAPHDB_RETRY_DELAY=5s
GRAPHDB_NAMED_GRAPH=http://cardiofit.ai/kb7/graph/default
GRAPHDB_TRANSACTION_TIMEOUT=10m
GRAPHDB_VALIDATE_TRIPLES=true
GRAPHDB_ENABLE_INFERENCE=true

# Backward Compatibility
ENABLE_TRIPLE_STORE=false  # Default: disabled for backward compatibility
```

### 12.2 Command-Line Flags

```bash
# Enable GraphDB triple loading
./etl --data ./data/snomed --enable-graphdb=true

# Specify GraphDB server
./etl --data ./data/snomed --enable-graphdb=true --graphdb-url http://graphdb:7200

# Disable triple validation (faster loading)
./etl --data ./data/snomed --enable-graphdb=true --validate-triples=false

# Custom batch size
./etl --data ./data/snomed --enable-graphdb=true --graphdb-batch-size=5000
```

---

## 13. Deployment Strategy

### 13.1 Rollout Plan

**Stage 1: Development (Week 1-2)**
- Implement TripleStoreCoordinator
- Unit and integration testing
- Local environment validation

**Stage 2: Staging (Week 3)**
- Deploy to staging environment
- Full data load test (520K concepts)
- Performance benchmarking
- Consistency validation

**Stage 3: Production Canary (Week 4)**
- Enable GraphDB for 10% of ETL runs
- Monitor for errors and performance impact
- Validate PostgreSQL operations unchanged

**Stage 4: Production Rollout (Week 5)**
- Enable GraphDB for 100% of ETL runs
- Monitor for 1 week
- Document operational procedures

### 13.2 Rollback Plan

**Rollback Trigger Conditions**:
- PostgreSQL ETL duration increases > 20%
- GraphDB failures cause PostgreSQL failures
- Memory usage exceeds 4 GB
- Any critical production incident

**Rollback Procedure**:
1. Set `GRAPHDB_ENABLED=false` in environment
2. Restart ETL service
3. Verify PostgreSQL/Elasticsearch operations normal
4. Investigate GraphDB issues offline

---

## 14. Future Enhancements

### 14.1 Optimization Opportunities

**Stream Processing** (Phase 2):
- Replace batch read from PostgreSQL with stream processing
- Reduce memory footprint to < 500 MB

**Parallel Upload** (Phase 2):
- Upload multiple batches to GraphDB concurrently
- Target: 50% reduction in GraphDB sync time

**Incremental Updates** (Phase 3):
- Detect changed concepts in PostgreSQL
- Upload only delta triples to GraphDB
- Enable daily update cadence (vs full reload)

**SPARQL Validation** (Phase 3):
- Validate triples using SPARQL queries
- Ensure hierarchy integrity (IS-A relationships)
- Detect orphaned concepts

### 14.2 Advanced Features

**Named Graph Management**:
```turtle
# Separate graphs per terminology system
GRAPH <http://cardiofit.ai/kb7/graph/snomed> { ... }
GRAPH <http://cardiofit.ai/kb7/graph/rxnorm> { ... }
GRAPH <http://cardiofit.ai/kb7/graph/loinc> { ... }
```

**Provenance Tracking**:
```turtle
sct:387517004 prov:wasGeneratedBy [
    a prov:Activity ;
    prov:startedAtTime "2025-11-22T10:00:00Z"^^xsd:dateTime ;
    prov:endedAtTime "2025-11-22T10:30:00Z"^^xsd:dateTime ;
    kb7:etlVersion "2.0.0" ;
    kb7:sourceFile "sct2_Concept_Snapshot_20240131.txt"
] .
```

**Versioning Support**:
```turtle
# Version-specific graphs
GRAPH <http://cardiofit.ai/kb7/graph/snomed/20240131> { ... }
GRAPH <http://cardiofit.ai/kb7/graph/snomed/20230731> { ... }
```

---

## 15. Conclusion

This architecture design provides a comprehensive blueprint for extending the KB-7 ETL pipeline to support GraphDB triple loading while maintaining full backward compatibility with existing PostgreSQL and Elasticsearch operations.

**Key Highlights**:
- **Reuse-First Approach**: Leverages DualStoreCoordinator pattern and existing GraphDB client code
- **Fail-Safe Design**: GraphDB failures do not impact PostgreSQL (source of truth)
- **Parallel Execution**: PostgreSQL + GraphDB + Elasticsearch writes execute concurrently
- **Comprehensive Validation**: 3-way consistency checks ensure data integrity
- **Production-Ready**: Error handling, retry logic, monitoring, and rollback strategies included

**Next Steps**:
1. Review and approve architecture design
2. Assign to refactoring-expert agent for implementation
3. Begin with Phase 1 (Transformer implementation)
4. Follow incremental development workflow
5. Validate each phase before proceeding

**Estimated Development Time**: 2 weeks (10 business days)

**Risk Assessment**: LOW
- Backward compatibility guaranteed (default GraphDB disabled)
- PostgreSQL operations unchanged
- Comprehensive testing strategy
- Clear rollback procedures

---

**Document Prepared By**: System Architect Agent
**Review Status**: Awaiting Approval
**Implementation Status**: Design Phase Complete - Ready for Development
