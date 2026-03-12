# Phase 1 Testing & Validation Strategy
**KB-7 GraphDB Foundation Implementation**

**Document Version**: 1.0
**Created**: November 22, 2025
**Status**: Implementation Ready
**Phase**: Phase 1 - GraphDB Foundation (Weeks 1-2)

---

## Executive Summary

This document defines comprehensive testing and validation strategies for Phase 1 of the KB-7 GraphDB transformation. Phase 1 establishes GraphDB as parallel storage alongside PostgreSQL and migrates 520K SNOMED CT, RxNorm, and LOINC concepts to RDF triple format.

**Critical Success Factors**:
- Zero data loss during migration (520K concepts)
- Data integrity verification at triple level
- Query performance validation (<100ms)
- Rollback capability tested and documented
- PostgreSQL/Elasticsearch continue functioning

**Risk Level**: 🔴 **CRITICAL** - Clinical terminology data where errors impact patient safety

---

## Table of Contents

1. [Testing Philosophy](#testing-philosophy)
2. [Test Environment Setup](#test-environment-setup)
3. [Test Categories](#test-categories)
4. [Component-Specific Test Specifications](#component-specific-test-specifications)
5. [Automated Validation Scripts](#automated-validation-scripts)
6. [Performance Benchmarks](#performance-benchmarks)
7. [Data Integrity Validation](#data-integrity-validation)
8. [Regression Testing Strategy](#regression-testing-strategy)
9. [Success Criteria](#success-criteria)
10. [Test Execution Plan](#test-execution-plan)
11. [Rollback Testing](#rollback-testing)

---

## Testing Philosophy

### Quality Gates Approach

**Gate 1: Infrastructure** → **Gate 2: Data Migration** → **Gate 3: Query Functionality** → **Gate 4: Performance** → **Gate 5: Integration**

Each gate must pass 100% before proceeding to the next phase.

### Testing Principles

1. **Data Integrity First**: Every concept must be verified at triple level
2. **Performance is a Feature**: Query latency is a pass/fail criterion, not just a metric
3. **Clinical Safety**: Test with real clinical scenarios (drug hierarchies, interactions)
4. **Regression Protection**: PostgreSQL and Elasticsearch must continue functioning
5. **Reproducibility**: All tests must be automated and repeatable

### Test Pyramid for Phase 1

```
                    ┌────────────────┐
                    │  E2E Clinical  │  10% - 5 scenarios
                    │   Workflows    │
                    └────────────────┘
                ┌──────────────────────┐
                │  Integration Tests    │  30% - 20 test cases
                │  GraphDB ↔ PostgreSQL │
                └──────────────────────┘
            ┌──────────────────────────────┐
            │    Component Tests            │  40% - 50 test cases
            │  ETL, SPARQL, Triple Load     │
            └──────────────────────────────┘
        ┌──────────────────────────────────────┐
        │         Unit Tests                    │  20% - 100+ test cases
        │  Data Validators, Converters, Parsers │
        └──────────────────────────────────────┘
```

---

## Test Environment Setup

### Infrastructure Requirements

```yaml
Test Environment Configuration:
  graphdb:
    version: "10.0+"
    heap: "4GB"
    repository: "kb7-terminology-test"
    ruleset: "owl2-rl-optimized"
    endpoint: "http://localhost:7200"

  postgresql:
    version: "15+"
    database: "kb7_terminology_test"
    port: 5433
    test_data: "production_snapshot_anonymized"

  redis:
    version: "7+"
    port: 6380
    datasets: 15  # Separate test dataset

  resources:
    disk_space: "50GB minimum"
    ram: "16GB recommended"
    cpu_cores: "4+ for parallel testing"
```

### Test Data Preparation

```bash
# scripts/test/setup-test-environment.sh
#!/bin/bash
set -e

echo "=== Setting Up Phase 1 Test Environment ==="

# 1. Create test database (isolated from production)
psql -U postgres -c "CREATE DATABASE kb7_terminology_test;"

# 2. Load production schema
psql -U postgres -d kb7_terminology_test < migrations/001_initial_schema.sql

# 3. Copy subset of production data (10K concepts for quick tests)
psql -U postgres << SQL
INSERT INTO kb7_terminology_test.terminology_concepts
SELECT * FROM kb7_terminology.terminology_concepts
ORDER BY RANDOM() LIMIT 10000;
SQL

# 4. Create GraphDB test repository
./scripts/create-graphdb-repository.sh kb7-terminology-test

# 5. Initialize test Redis dataset
redis-cli -n 15 FLUSHDB

echo "✅ Test environment ready"
echo "📊 Test database: 10K concepts loaded"
echo "🗄️  GraphDB repository: kb7-terminology-test created"
```

### Test Data Sets

| Dataset | Size | Purpose | Location |
|---------|------|---------|----------|
| **Unit Test Data** | 100 concepts | Fast unit testing | `tests/fixtures/sample_concepts.json` |
| **Integration Test Data** | 10K concepts | Integration testing | `tests/fixtures/integration_10k.sql` |
| **Performance Test Data** | 520K concepts | Full production load | Production snapshot |
| **Edge Case Data** | 500 concepts | Boundary condition testing | `tests/fixtures/edge_cases.json` |

---

## Test Categories

### 1. Infrastructure Tests (Quality Gate 1)

**Objective**: Verify GraphDB is operational and accessible

#### Test Cases

| Test ID | Test Name | Description | Pass Criteria |
|---------|-----------|-------------|---------------|
| `INF-001` | GraphDB Server Health | Verify GraphDB is running | HTTP 200 from /health |
| `INF-002` | Repository Creation | Create kb7-terminology-test | Repository appears in /repositories |
| `INF-003` | Repository Configuration | Verify OWL2-RL ruleset | Config shows owl2-rl-optimized |
| `INF-004` | SPARQL Endpoint Access | Query endpoint responds | SELECT query returns results |
| `INF-005` | Triple Insertion | Insert 10 test triples | All 10 triples queryable |
| `INF-006` | Connection Pool | Test 50 concurrent connections | No connection errors |

#### Implementation

```go
// tests/infrastructure/graphdb_test.go
package infrastructure

import (
    "context"
    "testing"
    "time"
    "kb-7-terminology/internal/semantic"
)

func TestINF001_GraphDBServerHealth(t *testing.T) {
    client := semantic.NewGraphDBClient(
        "http://localhost:7200",
        "kb7-terminology-test",
        testLogger,
    )

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err := client.HealthCheck(ctx)
    if err != nil {
        t.Fatalf("GraphDB health check failed: %v", err)
    }
}

func TestINF002_RepositoryCreation(t *testing.T) {
    // Repository should be created by setup script
    client := semantic.NewGraphDBClient(
        "http://localhost:7200",
        "kb7-terminology-test",
        testLogger,
    )

    ctx := context.Background()
    info, err := client.GetRepositoryInfo(ctx)

    if err != nil {
        t.Fatalf("Repository not accessible: %v", err)
    }

    if info.ID != "kb7-terminology-test" {
        t.Errorf("Wrong repository ID: got %s, want kb7-terminology-test", info.ID)
    }
}

func TestINF003_RepositoryConfiguration(t *testing.T) {
    // Verify OWL2-RL ruleset is configured
    client := semantic.NewGraphDBClient(
        "http://localhost:7200",
        "kb7-terminology-test",
        testLogger,
    )

    ctx := context.Background()
    config, err := client.GetRepositoryConfig(ctx)

    if err != nil {
        t.Fatalf("Failed to get repository config: %v", err)
    }

    if config.Ruleset != "owl2-rl-optimized" {
        t.Errorf("Wrong ruleset: got %s, want owl2-rl-optimized", config.Ruleset)
    }
}

func TestINF005_TripleInsertion(t *testing.T) {
    client := semantic.NewGraphDBClient(
        "http://localhost:7200",
        "kb7-terminology-test",
        testLogger,
    )

    ctx := context.Background()

    // Insert 10 test triples
    testTriples := []semantic.TripleData{
        {
            Subject:   "http://test.cardiofit.ai/concept/001",
            Predicate: "http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
            Object:    "http://cardiofit.ai/kb7/ontology#ClinicalConcept",
        },
        // ... 9 more test triples
    }

    err := client.InsertTriples(ctx, testTriples)
    if err != nil {
        t.Fatalf("Triple insertion failed: %v", err)
    }

    // Verify triples are queryable
    query := &semantic.SPARQLQuery{
        Query: `SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }`,
    }

    results, err := client.ExecuteSPARQL(ctx, query)
    if err != nil {
        t.Fatalf("Query after insertion failed: %v", err)
    }

    count := results.Results.Bindings[0]["count"].Value
    if count != "10" {
        t.Errorf("Expected 10 triples, got %s", count)
    }
}
```

---

### 2. ETL Pipeline Tests (Quality Gate 2)

**Objective**: Verify ETL correctly transforms PostgreSQL → RDF → GraphDB

#### Test Cases

| Test ID | Test Name | Description | Pass Criteria |
|---------|-----------|-------------|---------------|
| `ETL-001` | PostgreSQL Data Read | Read 10K concepts from PostgreSQL | All concepts retrieved |
| `ETL-002` | RDF Conversion SNOMED | Convert SNOMED concept to RDF | Valid Turtle output |
| `ETL-003` | RDF Conversion RxNorm | Convert RxNorm concept to RDF | Valid Turtle output |
| `ETL-004` | RDF Conversion LOINC | Convert LOINC concept to RDF | Valid Turtle output |
| `ETL-005` | Batch Processing | Process 1K concepts in batch | No memory errors |
| `ETL-006` | Error Handling | Handle malformed concepts | Errors logged, processing continues |
| `ETL-007` | Progress Tracking | Monitor ETL progress | Progress updates every 1K |
| `ETL-008` | Transaction Consistency | ETL with partial failure | Rollback on error |
| `ETL-009` | Relationship Preservation | Parent-child relationships | Hierarchies maintained |
| `ETL-010` | Full 520K Migration | Complete production migration | 520K concepts loaded |

#### SNOMED CT Conversion Test

```go
// tests/etl/snomed_conversion_test.go
package etl

import (
    "testing"
    "kb-7-terminology/internal/etl"
    "kb-7-terminology/internal/models"
)

func TestETL002_RDFConversionSNOMED(t *testing.T) {
    // Test data: SNOMED CT 387517004 (Paracetamol)
    concept := &models.TerminologyConcept{
        Code:    "387517004",
        System:  "SNOMED-CT",
        Display: "Paracetamol",
        ParentCode: sql.NullString{String: "7947003", Valid: true}, // Drug parent
    }

    converter := etl.NewRDFConverter(testLogger)
    triples, err := converter.ConvertConceptToRDF(concept)

    if err != nil {
        t.Fatalf("Conversion failed: %v", err)
    }

    // Verify triple structure
    expectedTriples := map[string]bool{
        "rdf:type":          false,
        "rdfs:label":        false,
        "kb7:code":          false,
        "kb7:system":        false,
        "rdfs:subClassOf":   false, // Parent relationship
    }

    for _, triple := range triples {
        predicate := extractPredicate(triple.Predicate)
        if _, exists := expectedTriples[predicate]; exists {
            expectedTriples[predicate] = true
        }
    }

    // All required predicates must be present
    for pred, found := range expectedTriples {
        if !found {
            t.Errorf("Missing required predicate: %s", pred)
        }
    }

    // Verify parent relationship specifically
    hasParent := false
    for _, triple := range triples {
        if strings.Contains(triple.Predicate, "subClassOf") {
            hasParent = true
            if !strings.Contains(triple.Object, "7947003") {
                t.Errorf("Wrong parent code in triple: %s", triple.Object)
            }
        }
    }

    if !hasParent {
        t.Error("Parent relationship (rdfs:subClassOf) not created")
    }
}

func TestETL009_RelationshipPreservation(t *testing.T) {
    // Test hierarchical relationship preservation
    // Load a parent-child-grandchild chain from PostgreSQL
    testData := []models.TerminologyConcept{
        {Code: "1", Display: "Grandparent", ParentCode: sql.NullString{Valid: false}},
        {Code: "2", Display: "Parent", ParentCode: sql.NullString{String: "1", Valid: true}},
        {Code: "3", Display: "Child", ParentCode: sql.NullString{String: "2", Valid: true}},
    }

    converter := etl.NewRDFConverter(testLogger)

    var allTriples []semantic.TripleData
    for _, concept := range testData {
        triples, err := converter.ConceptToRDF(&concept)
        if err != nil {
            t.Fatalf("Conversion failed for %s: %v", concept.Code, err)
        }
        allTriples = append(allTriples, triples...)
    }

    // Insert into GraphDB
    client := getTestGraphDBClient()
    err := client.InsertTriples(context.Background(), allTriples)
    if err != nil {
        t.Fatalf("Triple insertion failed: %v", err)
    }

    // Verify hierarchy via SPARQL
    query := &semantic.SPARQLQuery{
        Query: `
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
            SELECT ?child ?parent WHERE {
                ?child rdfs:subClassOf+ ?parent .
                FILTER(?child = <http://cardiofit.ai/kb7/concept/3>)
                FILTER(?parent = <http://cardiofit.ai/kb7/concept/1>)
            }
        `,
    }

    results, err := client.ExecuteSPARQL(context.Background(), query)
    if err != nil {
        t.Fatalf("Hierarchy query failed: %v", err)
    }

    if len(results.Results.Bindings) == 0 {
        t.Error("Grandparent relationship not preserved (transitive rdfs:subClassOf)")
    }
}
```

#### Full 520K Migration Test

```go
// tests/etl/full_migration_test.go
package etl

import (
    "testing"
    "context"
    "time"
)

func TestETL010_Full520KMigration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping full migration test in short mode")
    }

    t.Log("Starting full 520K concept migration test...")
    startTime := time.Now()

    // Initialize ETL coordinator
    coordinator := etl.NewTripleStoreCoordinator(
        getTestPostgresDB(),
        getTestGraphDBClient(),
        testLogger,
    )

    ctx := context.Background()

    // Progress tracking
    progressChan := make(chan int, 10)
    go func() {
        for count := range progressChan {
            t.Logf("Progress: %d concepts migrated", count)
        }
    }()

    // Execute full migration
    stats, err := coordinator.MigrateAllConcepts(ctx, progressChan)
    close(progressChan)

    if err != nil {
        t.Fatalf("Migration failed: %v", err)
    }

    duration := time.Since(startTime)

    // Validation checks
    t.Logf("Migration completed in %v", duration)
    t.Logf("Statistics: %+v", stats)

    // Check 1: Concept count matches
    if stats.ConceptsProcessed != 520000 {
        t.Errorf("Expected 520K concepts, processed %d", stats.ConceptsProcessed)
    }

    // Check 2: No errors
    if stats.Errors > 0 {
        t.Errorf("Migration had %d errors", stats.Errors)
    }

    // Check 3: Verify triple count in GraphDB
    query := &semantic.SPARQLQuery{
        Query: `SELECT (COUNT(*) as ?count) WHERE { ?s a <http://cardiofit.ai/kb7/ontology#ClinicalConcept> }`,
    }

    results, err := getTestGraphDBClient().ExecuteSPARQL(ctx, query)
    if err != nil {
        t.Fatalf("Triple count query failed: %v", err)
    }

    tripleCount := results.Results.Bindings[0]["count"].Value
    expectedCount := "520000"

    if tripleCount != expectedCount {
        t.Errorf("GraphDB has %s concepts, expected %s", tripleCount, expectedCount)
    }

    // Check 4: Performance target (should complete within 30 minutes for 520K)
    maxDuration := 30 * time.Minute
    if duration > maxDuration {
        t.Errorf("Migration took %v, exceeded target of %v", duration, maxDuration)
    }

    t.Logf("✅ Full migration test passed")
    t.Logf("   - Concepts: %d", stats.ConceptsProcessed)
    t.Logf("   - Duration: %v", duration)
    t.Logf("   - Rate: %.0f concepts/sec", float64(stats.ConceptsProcessed)/duration.Seconds())
}
```

---

### 3. Data Integrity Tests (Quality Gate 2)

**Objective**: Verify 100% data accuracy during migration

#### Test Cases

| Test ID | Test Name | Description | Pass Criteria |
|---------|-----------|-------------|---------------|
| `INT-001` | Concept Count Match | PostgreSQL count == GraphDB count | Exact match ±0 |
| `INT-002` | Code Integrity | All codes preserved | 100% code match |
| `INT-003` | Display Name Integrity | All display names preserved | 100% display match |
| `INT-004` | System Integrity | All systems preserved | 100% system match |
| `INT-005` | Parent Relationship Integrity | All parent links preserved | 100% hierarchy match |
| `INT-006` | No Duplicate Triples | No duplicate RDF triples | Zero duplicates |
| `INT-007` | No Orphaned Concepts | All concepts have system | Zero orphans |
| `INT-008` | UTF-8 Character Preservation | Special characters preserved | 100% character match |
| `INT-009` | Null Handling | NULL values handled correctly | No data corruption |
| `INT-010` | Checksum Validation | SHA256 validation | Checksums match |

#### Implementation

```bash
# scripts/validation/data-integrity-check.sh
#!/bin/bash
set -e

echo "=== Phase 1 Data Integrity Validation ==="

POSTGRES_DB="kb7_terminology"
GRAPHDB_REPO="kb7-terminology"
GRAPHDB_URL="http://localhost:7200"

# INT-001: Concept Count Match
echo "Test INT-001: Concept Count Match"
PG_COUNT=$(psql -U kb7_user -d $POSTGRES_DB -t -c \
    "SELECT COUNT(*) FROM terminology_concepts;")

GDB_COUNT=$(curl -s -X POST "$GRAPHDB_URL/repositories/$GRAPHDB_REPO" \
    --data-urlencode "query=SELECT (COUNT(*) as ?count) WHERE { ?s a <http://cardiofit.ai/kb7/ontology#ClinicalConcept> }" \
    -H "Accept: application/sparql-results+json" | \
    jq -r '.results.bindings[0].count.value')

echo "PostgreSQL: $PG_COUNT concepts"
echo "GraphDB: $GDB_COUNT concepts"

if [ "$PG_COUNT" -ne "$GDB_COUNT" ]; then
    echo "❌ FAIL: Concept count mismatch!"
    echo "   Difference: $((PG_COUNT - GDB_COUNT))"
    exit 1
fi
echo "✅ PASS: Concept counts match exactly"

# INT-002: Code Integrity (Sample Check)
echo ""
echo "Test INT-002: Code Integrity (Sample Check)"
# Get 100 random codes from PostgreSQL
SAMPLE_CODES=$(psql -U kb7_user -d $POSTGRES_DB -t -c \
    "SELECT code FROM terminology_concepts ORDER BY RANDOM() LIMIT 100;")

MISMATCH_COUNT=0
for code in $SAMPLE_CODES; do
    # Query GraphDB for this code
    GDB_RESULT=$(curl -s -X POST "$GRAPHDB_URL/repositories/$GRAPHDB_REPO" \
        --data-urlencode "query=SELECT ?concept WHERE { ?concept <http://cardiofit.ai/kb7/ontology#code> \"$code\" }" \
        -H "Accept: application/sparql-results+json" | \
        jq -r '.results.bindings | length')

    if [ "$GDB_RESULT" -eq 0 ]; then
        echo "⚠️  Code missing in GraphDB: $code"
        ((MISMATCH_COUNT++))
    fi
done

if [ $MISMATCH_COUNT -gt 0 ]; then
    echo "❌ FAIL: $MISMATCH_COUNT codes missing in GraphDB"
    exit 1
fi
echo "✅ PASS: All 100 sample codes found in GraphDB"

# INT-005: Parent Relationship Integrity
echo ""
echo "Test INT-005: Parent Relationship Integrity"
PG_PARENT_COUNT=$(psql -U kb7_user -d $POSTGRES_DB -t -c \
    "SELECT COUNT(*) FROM terminology_concepts WHERE parent_code IS NOT NULL;")

GDB_PARENT_COUNT=$(curl -s -X POST "$GRAPHDB_URL/repositories/$GRAPHDB_REPO" \
    --data-urlencode "query=SELECT (COUNT(*) as ?count) WHERE { ?child <http://www.w3.org/2000/01/rdf-schema#subClassOf> ?parent }" \
    -H "Accept: application/sparql-results+json" | \
    jq -r '.results.bindings[0].count.value')

echo "PostgreSQL parent relationships: $PG_PARENT_COUNT"
echo "GraphDB parent relationships: $GDB_PARENT_COUNT"

if [ "$PG_PARENT_COUNT" -ne "$GDB_PARENT_COUNT" ]; then
    echo "❌ FAIL: Parent relationship count mismatch!"
    exit 1
fi
echo "✅ PASS: All parent relationships preserved"

# INT-006: No Duplicate Triples
echo ""
echo "Test INT-006: No Duplicate Triples"
DUPLICATE_COUNT=$(curl -s -X POST "$GRAPHDB_URL/repositories/$GRAPHDB_REPO" \
    --data-urlencode "query=SELECT ?s ?p ?o (COUNT(*) as ?count) WHERE { ?s ?p ?o } GROUP BY ?s ?p ?o HAVING (COUNT(*) > 1)" \
    -H "Accept: application/sparql-results+json" | \
    jq -r '.results.bindings | length')

if [ "$DUPLICATE_COUNT" -gt 0 ]; then
    echo "❌ FAIL: Found $DUPLICATE_COUNT duplicate triples"
    exit 1
fi
echo "✅ PASS: No duplicate triples found"

# INT-007: No Orphaned Concepts
echo ""
echo "Test INT-007: No Orphaned Concepts"
ORPHAN_COUNT=$(curl -s -X POST "$GRAPHDB_URL/repositories/$GRAPHDB_REPO" \
    --data-urlencode "query=SELECT (COUNT(*) as ?count) WHERE { ?concept a <http://cardiofit.ai/kb7/ontology#ClinicalConcept> . FILTER NOT EXISTS { ?concept <http://cardiofit.ai/kb7/ontology#system> ?system } }" \
    -H "Accept: application/sparql-results+json" | \
    jq -r '.results.bindings[0].count.value')

if [ "$ORPHAN_COUNT" -gt 0 ]; then
    echo "❌ FAIL: Found $ORPHAN_COUNT orphaned concepts (missing system)"
    exit 1
fi
echo "✅ PASS: No orphaned concepts"

echo ""
echo "============================================"
echo "✅ All Data Integrity Tests PASSED"
echo "============================================"
```

---

### 4. SPARQL Query Tests (Quality Gate 3)

**Objective**: Verify SPARQL queries return correct results

#### Test Cases

| Test ID | Test Name | Description | Pass Criteria |
|---------|-----------|-------------|---------------|
| `SPQ-001` | Basic Concept Lookup | Query concept by code | Correct concept returned |
| `SPQ-002` | Subsumption Query | Get all child concepts | All children returned |
| `SPQ-003` | Transitive Closure | Get all descendants | Transitive hierarchy works |
| `SPQ-004` | Label Search | Find concepts by label | Fuzzy match works |
| `SPQ-005` | System Filter | Filter by terminology system | Only SNOMED returned |
| `SPQ-006` | Complex Join | Multi-predicate query | Correct results |
| `SPQ-007` | Aggregate Query | COUNT/GROUP BY queries | Aggregation works |
| `SPQ-008` | Property Path | Use SPARQL 1.1 property paths | Path queries work |
| `SPQ-009` | FILTER Clause | Complex FILTER expressions | Filters work correctly |
| `SPQ-010` | OPTIONAL Clause | Left join semantics | Optional data returned |

#### Implementation

```go
// tests/sparql/query_test.go
package sparql

import (
    "testing"
    "context"
    "kb-7-terminology/internal/semantic"
)

func TestSPQ001_BasicConceptLookup(t *testing.T) {
    client := getTestGraphDBClient()
    ctx := context.Background()

    // Query for Paracetamol (SNOMED 387517004)
    query := &semantic.SPARQLQuery{
        Query: `
            PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

            SELECT ?concept ?label ?system WHERE {
                ?concept kb7:code "387517004" ;
                        rdfs:label ?label ;
                        kb7:system ?system .
            }
        `,
    }

    results, err := client.ExecuteSPARQL(ctx, query)
    if err != nil {
        t.Fatalf("Query failed: %v", err)
    }

    if len(results.Results.Bindings) != 1 {
        t.Fatalf("Expected 1 result, got %d", len(results.Results.Bindings))
    }

    binding := results.Results.Bindings[0]

    if binding["label"].Value != "Paracetamol" {
        t.Errorf("Wrong label: got %s, want Paracetamol", binding["label"].Value)
    }

    if binding["system"].Value != "SNOMED-CT" {
        t.Errorf("Wrong system: got %s, want SNOMED-CT", binding["system"].Value)
    }
}

func TestSPQ002_SubsumptionQuery(t *testing.T) {
    client := getTestGraphDBClient()
    ctx := context.Background()

    // Get all direct children of a parent concept
    query := &semantic.SPARQLQuery{
        Query: `
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
            PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>

            SELECT ?child ?label WHERE {
                ?child rdfs:subClassOf <http://snomed.info/id/7947003> ;
                       rdfs:label ?label ;
                       kb7:system "SNOMED-CT" .
            }
            ORDER BY ?label
        `,
    }

    results, err := client.ExecuteSPARQL(ctx, query)
    if err != nil {
        t.Fatalf("Subsumption query failed: %v", err)
    }

    if len(results.Results.Bindings) == 0 {
        t.Error("No child concepts found (hierarchy not loaded correctly)")
    }

    t.Logf("Found %d child concepts", len(results.Results.Bindings))
}

func TestSPQ003_TransitiveClosure(t *testing.T) {
    client := getTestGraphDBClient()
    ctx := context.Background()

    // Use SPARQL 1.1 property path for transitive closure
    // Get ALL descendants (children, grandchildren, etc.)
    query := &semantic.SPARQLQuery{
        Query: `
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
            PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>

            SELECT ?descendant ?label WHERE {
                ?descendant rdfs:subClassOf+ <http://snomed.info/id/7947003> ;
                           rdfs:label ?label ;
                           kb7:system "SNOMED-CT" .
            }
        `,
    }

    results, err := client.ExecuteSPARQL(ctx, query)
    if err != nil {
        t.Fatalf("Transitive query failed: %v", err)
    }

    // Should have more descendants than direct children
    // This validates OWL2-RL reasoning is working
    if len(results.Results.Bindings) == 0 {
        t.Error("No descendants found - transitive closure not working")
    }

    t.Logf("Found %d total descendants (transitive)", len(results.Results.Bindings))
}

func TestSPQ004_LabelSearch(t *testing.T) {
    client := getTestGraphDBClient()
    ctx := context.Background()

    // Fuzzy label search with FILTER and REGEX
    query := &semantic.SPARQLQuery{
        Query: `
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
            PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>

            SELECT ?concept ?label ?code WHERE {
                ?concept rdfs:label ?label ;
                        kb7:code ?code ;
                        kb7:system "SNOMED-CT" .
                FILTER (REGEX(?label, "paracetamol", "i"))
            }
            LIMIT 10
        `,
    }

    results, err := client.ExecuteSPARQL(ctx, query)
    if err != nil {
        t.Fatalf("Label search failed: %v", err)
    }

    if len(results.Results.Bindings) == 0 {
        t.Error("Label search returned no results")
    }

    // Verify "Paracetamol" is in results
    found := false
    for _, binding := range results.Results.Bindings {
        if binding["code"].Value == "387517004" {
            found = true
            break
        }
    }

    if !found {
        t.Error("Paracetamol (387517004) not found in label search results")
    }
}
```

---

### 5. Performance Tests (Quality Gate 4)

**Objective**: Verify query performance meets <100ms target

#### Performance Benchmark Specification

```yaml
Performance Targets (P95):
  simple_lookup:
    query: "Lookup concept by code"
    target: "< 50ms"
    method: "SPARQL SELECT with exact match"

  subsumption:
    query: "Get direct children of concept"
    target: "< 100ms"
    method: "rdfs:subClassOf query"

  transitive_hierarchy:
    query: "Get all descendants (transitive)"
    target: "< 200ms"
    method: "rdfs:subClassOf+ property path"

  label_search:
    query: "Fuzzy search by label"
    target: "< 150ms"
    method: "FILTER with REGEX"

  aggregate:
    query: "COUNT concepts by system"
    target: "< 300ms"
    method: "GROUP BY with COUNT"
```

#### Performance Test Implementation

```go
// tests/performance/query_performance_test.go
package performance

import (
    "testing"
    "context"
    "time"
    "kb-7-terminology/internal/semantic"
)

func BenchmarkSimpleLookup(b *testing.B) {
    client := getTestGraphDBClient()
    ctx := context.Background()

    query := &semantic.SPARQLQuery{
        Query: `
            PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
            SELECT ?concept WHERE {
                ?concept kb7:code "387517004" .
            }
        `,
    }

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _, err := client.ExecuteSPARQL(ctx, query)
        if err != nil {
            b.Fatalf("Query failed: %v", err)
        }
    }
}

func TestPerformanceTargets(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping performance tests in short mode")
    }

    client := getTestGraphDBClient()
    ctx := context.Background()

    tests := []struct {
        name       string
        query      string
        targetP95  time.Duration
    }{
        {
            name: "Simple Lookup",
            query: `PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
                    SELECT ?concept WHERE { ?concept kb7:code "387517004" }`,
            targetP95: 50 * time.Millisecond,
        },
        {
            name: "Subsumption",
            query: `PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
                    SELECT ?child WHERE {
                        ?child rdfs:subClassOf <http://snomed.info/id/7947003>
                    }`,
            targetP95: 100 * time.Millisecond,
        },
        {
            name: "Transitive Hierarchy",
            query: `PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
                    SELECT ?descendant WHERE {
                        ?descendant rdfs:subClassOf+ <http://snomed.info/id/7947003>
                    }`,
            targetP95: 200 * time.Millisecond,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Run query 100 times to get P95
            var durations []time.Duration

            for i := 0; i < 100; i++ {
                start := time.Now()
                _, err := client.ExecuteSPARQL(ctx, &semantic.SPARQLQuery{Query: tt.query})
                duration := time.Since(start)

                if err != nil {
                    t.Fatalf("Query failed: %v", err)
                }

                durations = append(durations, duration)
            }

            // Calculate P95
            p95 := calculateP95(durations)

            t.Logf("Query: %s", tt.name)
            t.Logf("  P95: %v", p95)
            t.Logf("  Target: %v", tt.targetP95)

            if p95 > tt.targetP95 {
                t.Errorf("Performance target missed: P95 %v > target %v", p95, tt.targetP95)
            } else {
                t.Logf("  ✅ PASS: Performance target met")
            }
        })
    }
}

func calculateP95(durations []time.Duration) time.Duration {
    sort.Slice(durations, func(i, j int) bool {
        return durations[i] < durations[j]
    })

    index := int(float64(len(durations)) * 0.95)
    if index >= len(durations) {
        index = len(durations) - 1
    }

    return durations[index]
}
```

---

## Automated Validation Scripts

### Master Validation Script

```bash
# scripts/validation/phase1-master-validation.sh
#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/../../test-results/phase1"
mkdir -p "$RESULTS_DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$RESULTS_DIR/validation-report-$TIMESTAMP.txt"

echo "=== Phase 1 Master Validation ===" | tee "$REPORT_FILE"
echo "Started: $(date)" | tee -a "$REPORT_FILE"
echo "" | tee -a "$REPORT_FILE"

# Track overall pass/fail
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

run_test_suite() {
    local suite_name=$1
    local script_path=$2

    echo "Running test suite: $suite_name" | tee -a "$REPORT_FILE"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if bash "$script_path" >> "$REPORT_FILE" 2>&1; then
        echo "✅ PASS: $suite_name" | tee -a "$REPORT_FILE"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "❌ FAIL: $suite_name" | tee -a "$REPORT_FILE"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi

    echo "" | tee -a "$REPORT_FILE"
}

# Gate 1: Infrastructure
run_test_suite "Infrastructure Tests" "$SCRIPT_DIR/infrastructure-check.sh"

# Gate 2: Data Migration & Integrity
run_test_suite "ETL Pipeline Tests" "$SCRIPT_DIR/etl-validation.sh"
run_test_suite "Data Integrity Tests" "$SCRIPT_DIR/data-integrity-check.sh"

# Gate 3: Query Functionality
run_test_suite "SPARQL Query Tests" "$SCRIPT_DIR/sparql-query-validation.sh"

# Gate 4: Performance
run_test_suite "Performance Benchmarks" "$SCRIPT_DIR/performance-validation.sh"

# Gate 5: Regression
run_test_suite "Regression Tests" "$SCRIPT_DIR/regression-validation.sh"

# Final Report
echo "=======================================" | tee -a "$REPORT_FILE"
echo "Phase 1 Validation Summary" | tee -a "$REPORT_FILE"
echo "=======================================" | tee -a "$REPORT_FILE"
echo "Total Test Suites: $TOTAL_TESTS" | tee -a "$REPORT_FILE"
echo "Passed: $PASSED_TESTS" | tee -a "$REPORT_FILE"
echo "Failed: $FAILED_TESTS" | tee -a "$REPORT_FILE"
echo "" | tee -a "$REPORT_FILE"

if [ $FAILED_TESTS -eq 0 ]; then
    echo "🎉 ALL TESTS PASSED - Phase 1 Complete!" | tee -a "$REPORT_FILE"
    echo "Report saved: $REPORT_FILE" | tee -a "$REPORT_FILE"
    exit 0
else
    echo "❌ VALIDATION FAILED - $FAILED_TESTS test suite(s) failed" | tee -a "$REPORT_FILE"
    echo "Report saved: $REPORT_FILE" | tee -a "$REPORT_FILE"
    exit 1
fi
```

---

## Success Criteria

### Phase 1 Acceptance Checklist

Phase 1 is considered **COMPLETE** when ALL of the following criteria are met:

#### Data Migration Criteria

- [ ] **INT-001**: PostgreSQL concept count exactly matches GraphDB concept count
- [ ] **INT-002**: 100% of codes preserved and queryable in GraphDB
- [ ] **INT-003**: 100% of display names preserved
- [ ] **INT-005**: 100% of parent-child relationships preserved
- [ ] **INT-006**: Zero duplicate triples in GraphDB
- [ ] **INT-007**: Zero orphaned concepts (all have system)
- [ ] **INT-010**: SHA256 checksums validate for all migrated data

#### Query Functionality Criteria

- [ ] **SPQ-001**: Basic concept lookup by code works
- [ ] **SPQ-002**: Subsumption queries return correct children
- [ ] **SPQ-003**: Transitive closure queries work (OWL2-RL reasoning)
- [ ] **SPQ-004**: Label fuzzy search works correctly
- [ ] **SPQ-005**: System filtering works (SNOMED, RxNorm, LOINC)

#### Performance Criteria

- [ ] **PERF-001**: Simple lookup P95 < 50ms
- [ ] **PERF-002**: Subsumption query P95 < 100ms
- [ ] **PERF-003**: Transitive hierarchy P95 < 200ms
- [ ] **PERF-004**: Label search P95 < 150ms
- [ ] **PERF-005**: Full 520K migration completes < 30 minutes

#### Regression Criteria

- [ ] **REG-001**: PostgreSQL API endpoints continue working
- [ ] **REG-002**: Elasticsearch dual-write continues functioning
- [ ] **REG-003**: Existing Redis cache still operational
- [ ] **REG-004**: Service health checks pass
- [ ] **REG-005**: Existing clients (Flow2, Medication Service) not broken

#### Infrastructure Criteria

- [ ] **INF-001**: GraphDB server healthy and accessible
- [ ] **INF-002**: Repository created with correct configuration
- [ ] **INF-003**: OWL2-RL ruleset active
- [ ] **INF-006**: Connection pool handles 50 concurrent connections

#### Documentation Criteria

- [ ] **DOC-001**: Rollback procedure documented and tested
- [ ] **DOC-002**: Migration runbook created
- [ ] **DOC-003**: Performance benchmarks documented
- [ ] **DOC-004**: Known issues list compiled

### Quantitative Success Metrics

```yaml
Required Metrics:
  data_integrity:
    concept_accuracy: "100%"  # All 520K concepts migrated
    relationship_accuracy: "100%"  # All hierarchies preserved
    zero_data_loss: "mandatory"

  performance:
    simple_lookup_p95: "< 50ms"
    subsumption_p95: "< 100ms"
    transitive_p95: "< 200ms"
    migration_time: "< 30 minutes"

  availability:
    graphdb_uptime: "99.9%"
    query_success_rate: "> 99%"
    no_downtime: "mandatory"

  regression:
    postgresql_performance: "no degradation"
    existing_apis: "100% functional"
    client_compatibility: "100% maintained"
```

---

## Regression Testing Strategy

### Objective

Ensure PostgreSQL and Elasticsearch continue functioning during GraphDB integration

### Test Scenarios

```yaml
Regression Test Scenarios:

1. PostgreSQL Continues Working:
   - REST API /v1/concepts endpoints
   - Existing terminology lookups
   - Validation service
   - Value set expansion

2. Elasticsearch Continues Working:
   - Dual-write still operational
   - Full-text search functional
   - Index synchronization

3. Service Integration:
   - Flow2 orchestrator still works
   - Medication service still works
   - Clinical reasoning service still works

4. Performance Not Degraded:
   - PostgreSQL query times unchanged
   - API response times unchanged
   - Cache hit rates maintained
```

### Implementation

```bash
# scripts/validation/regression-validation.sh
#!/bin/bash
set -e

echo "=== Regression Testing: PostgreSQL & Elasticsearch ==="

# Test 1: PostgreSQL REST API
echo "Test REG-001: PostgreSQL API Endpoints"
RESPONSE=$(curl -s -w "%{http_code}" http://localhost:8092/v1/concepts/SNOMED/387517004)
HTTP_CODE="${RESPONSE: -3}"

if [ "$HTTP_CODE" != "200" ]; then
    echo "❌ FAIL: PostgreSQL API returned HTTP $HTTP_CODE"
    exit 1
fi
echo "✅ PASS: PostgreSQL API operational"

# Test 2: Elasticsearch Search
echo ""
echo "Test REG-002: Elasticsearch Full-Text Search"
ES_RESULT=$(curl -s -X POST http://localhost:9200/terminology/_search \
    -H "Content-Type: application/json" \
    -d '{"query": {"match": {"display": "paracetamol"}}}' | \
    jq -r '.hits.total.value')

if [ "$ES_RESULT" -eq 0 ]; then
    echo "❌ FAIL: Elasticsearch returned no results"
    exit 1
fi
echo "✅ PASS: Elasticsearch search working (found $ES_RESULT results)"

# Test 3: Service Health Checks
echo ""
echo "Test REG-004: Service Health Checks"
HEALTH_STATUS=$(curl -s http://localhost:8092/health | jq -r '.status')

if [ "$HEALTH_STATUS" != "healthy" ]; then
    echo "❌ FAIL: Service health check failed: $HEALTH_STATUS"
    exit 1
fi
echo "✅ PASS: Service health check passed"

# Test 4: Performance Baseline
echo ""
echo "Test REG-005: PostgreSQL Query Performance"
START_TIME=$(date +%s%N)
curl -s http://localhost:8092/v1/concepts/SNOMED/387517004 > /dev/null
END_TIME=$(date +%s%N)
LATENCY=$(( (END_TIME - START_TIME) / 1000000 ))  # Convert to milliseconds

BASELINE_LATENCY=50  # Pre-GraphDB baseline
if [ $LATENCY -gt $((BASELINE_LATENCY * 2)) ]; then
    echo "⚠️  WARNING: Query latency ${LATENCY}ms exceeded baseline ${BASELINE_LATENCY}ms"
else
    echo "✅ PASS: Query latency ${LATENCY}ms within acceptable range"
fi

echo ""
echo "======================================="
echo "✅ All Regression Tests PASSED"
echo "======================================="
```

---

## Rollback Testing

### Rollback Scenarios

```yaml
Rollback Test Scenarios:

1. Partial Migration Failure:
   - Trigger: ETL fails at 50% completion
   - Action: Rollback GraphDB, keep PostgreSQL
   - Validation: PostgreSQL data intact, service operational

2. Data Corruption Detection:
   - Trigger: Integrity check fails
   - Action: Full rollback, restore from backup
   - Validation: Original state restored

3. Performance Degradation:
   - Trigger: Queries exceed SLA
   - Action: Disable GraphDB queries, fallback to PostgreSQL
   - Validation: Service continues with PostgreSQL only
```

### Rollback Validation Script

```bash
# scripts/validation/test-rollback-procedure.sh
#!/bin/bash
set -e

echo "=== Testing Rollback Procedure ==="

# Step 1: Backup current state
echo "1. Creating backup of current state..."
BACKUP_FILE="./backups/pre-rollback-test-$(date +%Y%m%d_%H%M%S).sql"
pg_dump -U kb7_user -d kb7_terminology > "$BACKUP_FILE"
echo "✅ Backup created: $BACKUP_FILE"

# Step 2: Simulate migration failure
echo ""
echo "2. Simulating migration failure..."
# Clear GraphDB repository
curl -X DELETE http://localhost:7200/repositories/kb7-terminology/statements

# Step 3: Execute rollback
echo ""
echo "3. Executing rollback procedure..."
./scripts/rollback/restore-postgresql-concepts.sh "$BACKUP_FILE"

# Step 4: Validate rollback
echo ""
echo "4. Validating post-rollback state..."

# Check PostgreSQL count
PG_COUNT=$(psql -U kb7_user -d kb7_terminology -t -c "SELECT COUNT(*) FROM terminology_concepts WHERE archived_at IS NULL;")
echo "PostgreSQL concepts: $PG_COUNT"

if [ "$PG_COUNT" -lt 520000 ]; then
    echo "❌ FAIL: Rollback did not restore all concepts"
    exit 1
fi

# Check API still works
API_STATUS=$(curl -s -w "%{http_code}" http://localhost:8092/v1/concepts/SNOMED/387517004)
HTTP_CODE="${API_STATUS: -3}"

if [ "$HTTP_CODE" != "200" ]; then
    echo "❌ FAIL: API not functional after rollback"
    exit 1
fi

echo ""
echo "======================================="
echo "✅ Rollback Test PASSED"
echo "======================================="
echo "Rollback procedure validated successfully"
echo "System restored to operational state"
```

---

## Test Execution Plan

### Timeline

```
Week 1 - Days 1-2: Infrastructure Setup & Testing
├─ Set up test environment
├─ Run infrastructure tests (Gate 1)
├─ Create test repositories
└─ Validate basic connectivity

Week 1 - Days 3-5: ETL Development & Testing
├─ Develop ETL pipeline
├─ Unit test RDF conversion
├─ Test small batches (1K concepts)
└─ Validate data integrity on samples

Week 2 - Days 1-2: Full Migration Testing
├─ Run full 520K migration
├─ Execute data integrity tests (Gate 2)
├─ Performance benchmarking
└─ Identify and fix issues

Week 2 - Days 3-4: Query & Integration Testing
├─ SPARQL query tests (Gate 3)
├─ Performance validation (Gate 4)
├─ Regression testing (Gate 5)
└─ Client integration validation

Week 2 - Day 5: Rollback Testing & Sign-off
├─ Test rollback procedures
├─ Final validation run
├─ Documentation review
└─ Phase 1 sign-off
```

### Daily Checklist

```bash
# Daily validation checklist
daily-validation.sh:

1. Morning Health Check:
   ✓ GraphDB server running
   ✓ PostgreSQL accessible
   ✓ Redis cache operational
   ✓ All services healthy

2. Run Quick Validation:
   ✓ Concept count check
   ✓ Sample query test
   ✓ Performance spot check

3. Review Metrics:
   ✓ Query latency trends
   ✓ Error rates
   ✓ Resource utilization

4. Evening Backup:
   ✓ Backup test database
   ✓ Export test results
   ✓ Update progress log
```

---

## Deliverables

### Test Artifacts

1. **Test Results Report** (`test-results/phase1/validation-report-YYYYMMDD.txt`)
2. **Performance Benchmark Data** (`test-results/phase1/performance-metrics.csv`)
3. **Data Integrity Report** (`test-results/phase1/integrity-validation.json`)
4. **Regression Test Results** (`test-results/phase1/regression-report.txt`)
5. **Rollback Test Evidence** (`test-results/phase1/rollback-validation.log`)

### Documentation

1. **Phase 1 Test Plan** (this document)
2. **Migration Runbook** (`docs/phase1-migration-runbook.md`)
3. **Rollback Procedure** (`docs/phase1-rollback-procedure.md`)
4. **Known Issues Log** (`docs/phase1-known-issues.md`)
5. **Performance Baselines** (`docs/phase1-performance-baselines.md`)

### Acceptance Sign-off

```yaml
Phase 1 Acceptance Sign-off:

Technical Lead: __________________  Date: ________
  ✓ All quality gates passed
  ✓ Performance targets met
  ✓ Rollback tested

Clinical Informaticist: __________________  Date: ________
  ✓ Data integrity validated
  ✓ Clinical scenarios tested
  ✓ Safety verification complete

DevOps Engineer: __________________  Date: ________
  ✓ Infrastructure stable
  ✓ Monitoring configured
  ✓ Rollback procedures documented

Project Manager: __________________  Date: ________
  ✓ All deliverables complete
  ✓ Documentation reviewed
  ✓ Phase 2 readiness confirmed
```

---

## Risk Mitigation

### Identified Risks

| Risk | Probability | Impact | Mitigation Strategy |
|------|-------------|--------|---------------------|
| Data loss during migration | Medium | Critical | Incremental migration with validation at each step |
| Performance degradation | Medium | High | Benchmark early, optimize before full load |
| GraphDB instability | Low | High | Test with stress tests, have rollback ready |
| Client compatibility issues | Medium | Medium | Regression testing before deployment |
| Insufficient test coverage | Low | Medium | Comprehensive test suite, code review |

### Contingency Plans

**If migration fails at 50%:**
- Stop ETL immediately
- Preserve PostgreSQL state
- Analyze failure logs
- Fix issues and restart from checkpoint

**If performance targets not met:**
- Analyze slow queries
- Add indexes to GraphDB
- Optimize SPARQL queries
- Consider hardware upgrades

**If data integrity issues found:**
- Halt migration
- Identify corruption source
- Restore from backup
- Fix conversion logic
- Restart with validation

---

## Conclusion

This comprehensive testing strategy ensures Phase 1 implementation meets all technical, performance, and clinical safety requirements. The multi-layered validation approach provides confidence that 520K clinical concepts are migrated accurately with zero data loss.

**Key Success Factors:**
- Automated validation scripts catch issues early
- Performance benchmarks prevent deployment of slow queries
- Regression testing protects existing functionality
- Rollback procedures provide safety net
- Clinical data integrity is validated at triple level

**Next Steps:**
1. Review and approve this testing strategy
2. Set up test environment (Week 1, Day 1)
3. Begin infrastructure testing (Week 1, Day 1-2)
4. Execute test plan according to timeline
5. Document results and obtain sign-off

---

**Document Maintenance:**
- Version: 1.0
- Last Updated: November 22, 2025
- Next Review: Start of Week 2 (after infrastructure testing)
- Owner: Quality Engineering Team / KB-7 Technical Lead
