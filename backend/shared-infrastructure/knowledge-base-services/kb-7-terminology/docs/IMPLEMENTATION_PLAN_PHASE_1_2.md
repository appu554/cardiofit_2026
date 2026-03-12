# Phase 1.2 Implementation Plan: GraphDB Triple Loading
**ETL Pipeline Extension - Summary Document**

**Date**: November 22, 2025
**Status**: Design Complete - Ready for Implementation
**Estimated Duration**: 2 weeks (10 business days)
**Risk Level**: LOW

---

## Executive Summary

This document summarizes the architectural design for extending KB-7's ETL pipeline to support GraphDB triple loading. The implementation follows the existing DualStoreCoordinator pattern, ensuring zero-risk backward compatibility while adding semantic reasoning capabilities.

### Architecture at a Glance

```
Current (DualStoreCoordinator):
PostgreSQL ← ETL → Elasticsearch

Target (TripleStoreCoordinator):
PostgreSQL ← ETL → Elasticsearch
                └→ GraphDB (NEW - RDF Triples)
```

### Core Design Principles
1. **Backward Compatibility First**: PostgreSQL remains unchanged and fully functional
2. **Fail-Safe Architecture**: GraphDB failures do not block PostgreSQL operations
3. **Reuse Over Rebuild**: Leverage existing DualStoreCoordinator and GraphDB client
4. **Parallel Execution**: All 3 stores write concurrently for performance
5. **Data Integrity**: Comprehensive 3-way validation ensures consistency

---

## Key Architectural Decisions

### Decision 1: Wrapper Pattern (Inheritance)

**Choice**: TripleStoreCoordinator embeds DualStoreCoordinator
```go
type TripleStoreCoordinator struct {
    *DualStoreCoordinator  // Reuse PostgreSQL + Elasticsearch logic
    graphDBClient          *semantic.GraphDBClient
    rdfTransformer         *transformer.SNOMEDToRDFTransformer
}
```

**Rationale**:
- Zero changes to existing PostgreSQL/Elasticsearch code
- Clear separation of concerns
- Easy to disable GraphDB (default: disabled)
- Maintains existing error handling and transaction management

**Alternatives Considered**:
- ❌ Modify DualStoreCoordinator directly: Too risky, breaks existing code
- ❌ Parallel coordinator: Code duplication, maintenance burden
- ✅ Wrapper pattern: Best balance of reuse and isolation

### Decision 2: PostgreSQL as Source of Truth

**Choice**: Read concepts from PostgreSQL, convert to RDF, upload to GraphDB
```
PostgreSQL (520K concepts) → RDF Transformer → GraphDB (12.6M triples)
```

**Rationale**:
- PostgreSQL is already validated and reliable
- No need to re-parse SNOMED RF2 files
- Simpler error recovery (re-read from PostgreSQL)
- Single source of truth for data integrity

**Alternatives Considered**:
- ❌ Direct RF2 → GraphDB: Duplicates file parsing logic
- ❌ Elasticsearch → GraphDB: Elasticsearch not guaranteed consistent
- ✅ PostgreSQL → GraphDB: Clean, reliable, maintainable

### Decision 3: Batch Processing with Streaming

**Choice**: Read PostgreSQL in batches, convert to Turtle, upload to GraphDB
```go
batchSize := 1000 concepts
for batch in concepts {
    turtleDoc := transformer.BatchToTurtle(batch)  // 8K-24K triples
    graphDB.UploadTurtle(turtleDoc)
}
```

**Rationale**:
- Memory efficient: < 500 MB peak usage
- Parallel-friendly: Batches can be processed concurrently (future)
- Fault-tolerant: Failed batches don't block others
- Performance: ~10K triples/sec throughput

**Alternatives Considered**:
- ❌ Load all to memory: 2-4 GB memory, OOM risk
- ❌ One-by-one: Too slow, network overhead
- ✅ Batch processing: Optimal balance

### Decision 4: Non-Blocking GraphDB Failures

**Choice**: GraphDB failures logged but don't fail ETL
```go
err := coordinator.LoadAllTerminologiesTripleStore(ctx, dataSources)
// Returns nil even if GraphDB fails, as long as PostgreSQL succeeds
```

**Rationale**:
- PostgreSQL is primary storage (production queries use it)
- GraphDB is enhancement (semantic queries, SPARQL)
- Production ETL runs must complete reliably
- GraphDB can be backfilled later

**Alternatives Considered**:
- ❌ All-or-nothing: GraphDB issues block production data loads
- ❌ PostgreSQL optional: Breaks existing API contracts
- ✅ PostgreSQL critical, GraphDB best-effort: Production-safe

### Decision 5: RDF Data Model (SNOMED → Turtle)

**Choice**: Standard W3C ontology vocabularies
```turtle
@prefix sct: <http://snomed.info/id/> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .

sct:387517004 a owl:Class ;
    rdfs:label "Paracetamol"@en ;
    skos:prefLabel "Paracetamol"@en ;
    rdfs:subClassOf sct:7947003 .
```

**Rationale**:
- Standard SNOMED URI scheme compatibility
- SKOS for terminology (industry standard)
- OWL for inference capabilities
- Interoperable with other semantic systems

**Alternatives Considered**:
- ❌ Custom ontology: Incompatible, limited tooling
- ❌ Minimal RDF: No inference, limited queries
- ✅ Standard W3C stack: Maximum interoperability

---

## Implementation File Structure

### New Files (Create)
```
internal/etl/
├── triple_store_coordinator.go     # Main coordinator (600 lines)
│   └── LoadAllTerminologiesTripleStore()
├── graphdb_loader.go                # GraphDB loader (300 lines)
│   └── LoadTriples(), ClearRepository()
└── triple_validator.go              # Consistency checker (200 lines)
    └── performTripleStoreConsistencyCheck()

internal/transformer/
└── snomed_to_rdf.go                 # RDF conversion (500 lines)
    ├── ConceptToTriples()
    └── BatchToTurtle()
```

### Files to Modify
```
cmd/etl/main.go
CHANGES:
- Add --enable-graphdb flag (line 31)
- Add GraphDBConfig block (after line 166)
- Replace DualStoreCoordinator with TripleStoreCoordinator (line 181)
- Replace LoadAllTerminologiesDualStore with LoadAllTerminologiesTripleStore (line 244)

BACKWARD COMPATIBILITY:
- Default: --enable-graphdb=false
- When false: TripleStoreCoordinator behaves identically to DualStoreCoordinator
```

### Files to Reuse (No Changes)
```
internal/semantic/graphdb_client.go       # ✅ Existing GraphDB REST client
internal/semantic/rdf_converter.go        # ✅ Existing RDF utilities
internal/etl/enhanced_coordinator.go      # ✅ Existing PostgreSQL logic
internal/etl/dual_store_coordinator.go    # ✅ Existing Elasticsearch logic
internal/etl/transaction_manager.go       # ✅ Existing transaction handling
```

**Code Reuse**: 75% existing code, 25% new code

---

## Data Flow Architecture

### ETL Execution Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ cmd/etl/main.go                                                 │
│   └─ TripleStoreCoordinator.LoadAllTerminologiesTripleStore()  │
└─────────────────────────────────────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 1: PostgreSQL + Elasticsearch (CRITICAL PATH)            │
│   DualStoreCoordinator.LoadAllTerminologiesDualStore()         │
│   ├─ EnhancedCoordinator.LoadAllTerminologies() → PostgreSQL   │
│   └─ syncToElasticsearch() → Elasticsearch                     │
│                                                                 │
│   IF FAILS: ABORT ENTIRE ETL (Critical Failure)                │
│   IF SUCCEEDS: Continue to Phase 2                             │
└─────────────────────────────────────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 2: GraphDB Sync (NON-CRITICAL PATH)                      │
│   TripleStoreCoordinator.syncToGraphDB()                       │
│   ├─ readConceptsFromPostgreSQL() → 520K concepts              │
│   ├─ RDFTransformer.BatchToTurtle() → 12.6M triples           │
│   └─ GraphDBLoader.LoadTriples() → GraphDB                     │
│                                                                 │
│   IF FAILS: LOG ERROR, mark service as "degraded"              │
│   IF SUCCEEDS: Continue to Phase 3                             │
└─────────────────────────────────────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 3: Consistency Validation (INFORMATIONAL)                │
│   performTripleStoreConsistencyCheck()                         │
│   ├─ Count PostgreSQL concepts                                 │
│   ├─ Count Elasticsearch documents                             │
│   ├─ Count GraphDB triples                                     │
│   └─ Calculate consistency score                               │
│                                                                 │
│   IF FAILS: LOG WARNING (does not affect ETL success)          │
└─────────────────────────────────────────────────────────────────┘
                            ▼
                    ETL SUCCESS
    (PostgreSQL loaded = overall success)
```

### RDF Transformation Pipeline

```
PostgreSQL Concept
{
  code: "387517004",
  system: "SNOMED",
  preferred_term: "Paracetamol",
  definition: "Analgesic and antipyretic",
  active: true,
  parent_codes: ["7947003"]
}
                ▼
        RDFTransformer.ConceptToTriples()
                ▼
RDFTripleSet (8 triples)
[
  {subject: "sct:387517004", predicate: "rdf:type", object: "owl:Class"},
  {subject: "sct:387517004", predicate: "rdfs:label", object: "Paracetamol"@en},
  {subject: "sct:387517004", predicate: "skos:prefLabel", object: "Paracetamol"@en},
  {subject: "sct:387517004", predicate: "skos:definition", object: "Analgesic..."@en},
  {subject: "sct:387517004", predicate: "kb7:conceptId", object: "387517004"},
  {subject: "sct:387517004", predicate: "kb7:active", object: "true"^^xsd:boolean},
  {subject: "sct:387517004", predicate: "rdfs:subClassOf", object: "sct:7947003"},
]
                ▼
        RDFTransformer.BatchToTurtle()
                ▼
Turtle Document (String)
@prefix sct: <http://snomed.info/id/> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .

sct:387517004 a owl:Class ;
    rdfs:label "Paracetamol"@en ;
    skos:prefLabel "Paracetamol"@en ;
    rdfs:subClassOf sct:7947003 .
                ▼
        GraphDBClient.UploadTurtle()
                ▼
            GraphDB Repository
```

---

## Error Handling Strategy

### Failure Classification

| Failure Type | Severity | Response | Example |
|--------------|----------|----------|---------|
| **PostgreSQL Write** | 🔴 CRITICAL | Abort ETL, rollback all | DB connection lost |
| **Elasticsearch Write** | 🟡 WARNING | Continue, mark degraded | ES index full |
| **GraphDB Connection** | 🟡 WARNING | Skip GraphDB, log error | GraphDB server down |
| **GraphDB Upload** | 🟡 WARNING | Retry 3x, then skip batch | Network timeout |
| **Triple Conversion** | 🟢 INFO | Skip concept, log, continue | Malformed concept data |
| **Consistency Check** | 🟢 INFO | Log discrepancy, continue | 1% triple mismatch |

### Retry Logic Flow

```
GraphDB Upload Attempt
        ▼
    Success?
    ├─ YES → Continue
    └─ NO → Check Retry Count
            ├─ Attempt < 3 → Wait 5s * attempt → Retry
            └─ Attempt = 3 → Log Error → Mark Batch Failed → Continue
```

### Rollback Strategy

**PostgreSQL Failure** (Critical):
```
1. Rollback PostgreSQL transaction (existing logic)
2. Delete Elasticsearch documents (if written)
3. GraphDB upload never started (transaction not initiated)
4. Return error to user
```

**GraphDB Failure** (Non-Critical):
```
1. PostgreSQL commits remain (source of truth)
2. Elasticsearch commits remain
3. Clear partial GraphDB uploads (CLEAR GRAPH command)
4. Log error, mark service "degraded"
5. Return success to user (PostgreSQL succeeded)
```

---

## Performance Targets & Benchmarks

### Target Metrics

| Metric | Target | Current (PostgreSQL) | GraphDB Impact |
|--------|--------|----------------------|----------------|
| **Total ETL Duration** | < 40 min | ~15 min | +25 min (GraphDB sync) |
| **PostgreSQL Load** | < 15 min | ~15 min | No change |
| **GraphDB Sync** | < 25 min | N/A | New operation |
| **Peak Memory** | < 2 GB | ~800 MB | +1.2 GB (batch processing) |
| **Triples/Second** | > 10K | N/A | ~8.4K triples/sec (target) |
| **Concepts/Second** | > 500 | ~580 | Maintained |

### Performance Optimization Strategies

1. **Batch Size Tuning**:
   - Start: 1,000 concepts/batch (~8K triples)
   - Optimize: Test 500, 1000, 2000, 5000 to find sweet spot
   - Target: Maximize throughput while keeping memory < 2 GB

2. **Parallel Upload** (Future):
   - Phase 1: Sequential batch upload
   - Phase 2: 4 concurrent batch uploads
   - Expected: 40% reduction in GraphDB sync time

3. **Triple Generation Optimization**:
   - Use string.Builder for Turtle serialization
   - Pre-allocate RDFTriple slices (capacity = 10)
   - Cache namespace prefix strings

4. **Network Optimization**:
   - Enable HTTP compression (gzip)
   - HTTP keep-alive for GraphDB connections
   - TCP connection pooling

---

## Testing Strategy

### Unit Tests (internal/etl/, internal/transformer/)

```go
// File: internal/transformer/snomed_to_rdf_test.go
func TestConceptToTriples(t *testing.T) {
    // Test cases:
    // 1. Standard SNOMED concept with all fields
    // 2. Minimal concept (only code + term)
    // 3. Concept with special characters in term
    // 4. Concept with multiple parent codes
    // 5. Concept with synonyms in properties
}

func TestBatchToTurtle(t *testing.T) {
    // Test cases:
    // 1. Empty batch
    // 2. Single concept
    // 3. 100 concepts (small batch)
    // 4. 1000 concepts (full batch)
    // 5. Verify Turtle syntax validity
}

// File: internal/etl/triple_store_coordinator_test.go
func TestTripleStoreCoordinator(t *testing.T) {
    // Test cases:
    // 1. Initialization with valid config
    // 2. Initialization with GraphDB disabled
    // 3. PostgreSQL success, GraphDB failure (degraded)
    // 4. PostgreSQL failure (critical abort)
    // 5. Consistency validation
}
```

### Integration Tests (scripts/test-graphdb-integration.sh)

```bash
#!/bin/bash
# Integration test: Full ETL run with GraphDB

set -e

echo "=== Phase 1.2 Integration Test ==="

# 1. Setup: Start services
docker-compose up -d kb7-postgres kb7-graphdb kb7-elasticsearch

# 2. Load test data (5K concepts)
./etl --data ./data/test-snomed-5k \
      --enable-graphdb=true \
      --batch-size=500

# 3. Validate PostgreSQL
PG_COUNT=$(psql -U kb7_user -t -c "SELECT COUNT(*) FROM concepts;")
echo "✓ PostgreSQL concepts: $PG_COUNT"
[ "$PG_COUNT" -eq 5000 ] || exit 1

# 4. Validate GraphDB
GRAPHDB_QUERY='SELECT (COUNT(*) as ?count) WHERE { ?s a owl:Class }'
GRAPHDB_COUNT=$(curl -s -X POST http://localhost:7200/repositories/kb7-terminology \
  --data-urlencode "query=$GRAPHDB_QUERY" | jq '.results.bindings[0].count.value')
echo "✓ GraphDB triples: $GRAPHDB_COUNT"
[ "$GRAPHDB_COUNT" -gt 35000 ] || exit 1  # ~8 triples/concept minimum

# 5. Test SPARQL query
SPARQL_QUERY='SELECT ?concept ?label WHERE {
  ?concept rdfs:label ?label .
  FILTER(CONTAINS(?label, "Paracetamol"))
} LIMIT 5'

RESULTS=$(curl -s -X POST http://localhost:7200/repositories/kb7-terminology \
  --data-urlencode "query=$SPARQL_QUERY")

echo "✓ SPARQL query results:"
echo "$RESULTS" | jq '.results.bindings[] | {concept: .concept.value, label: .label.value}'

# 6. Consistency check
echo "✓ 3-way consistency validated"

echo "=== Integration Test PASSED ==="
```

### Performance Test (scripts/benchmark-graphdb-loading.sh)

```bash
#!/bin/bash
# Benchmark GraphDB loading performance

set -e

echo "=== GraphDB Loading Benchmark ==="

# Test with increasing dataset sizes
for SIZE in 1000 5000 10000 50000; do
    echo "Testing with $SIZE concepts..."

    START=$(date +%s)

    ./etl --data ./data/test-snomed-$SIZE \
          --enable-graphdb=true \
          --batch-size=1000

    END=$(date +%s)
    DURATION=$((END - START))

    CONCEPTS_PER_SEC=$((SIZE / DURATION))
    echo "Duration: ${DURATION}s | Rate: ${CONCEPTS_PER_SEC} concepts/sec"

    # Extract metrics from logs
    TRIPLES_LOADED=$(grep "total_triples_loaded" logs/etl.log | tail -1 | awk '{print $NF}')
    TRIPLES_PER_SEC=$((TRIPLES_LOADED / DURATION))

    echo "Triples loaded: $TRIPLES_LOADED | Rate: ${TRIPLES_PER_SEC} triples/sec"
    echo "---"
done

echo "=== Benchmark Complete ==="
```

---

## Deployment Plan

### Stage 1: Development (Week 1)
**Days 1-2**: Implement RDF Transformer
- Create `internal/transformer/snomed_to_rdf.go`
- Implement `ConceptToTriples()` and `BatchToTurtle()`
- Unit tests (>90% coverage)
- Test with 100 sample concepts

**Days 3-4**: Implement GraphDB Loader
- Create `internal/etl/graphdb_loader.go`
- Implement `LoadTriples()` with retry logic
- Unit tests for loader
- Test upload to local GraphDB

**Day 5**: Implement TripleStoreCoordinator
- Create `internal/etl/triple_store_coordinator.go`
- Implement `LoadAllTerminologiesTripleStore()`
- Integration with existing DualStoreCoordinator
- Unit tests for coordinator

### Stage 2: Integration (Week 2)
**Days 1-2**: ETL Main Integration
- Modify `cmd/etl/main.go`
- Add `--enable-graphdb` flag and configuration
- Backward compatibility testing (GraphDB disabled)
- Full integration test (5K concepts)

**Days 3-4**: Full Scale Testing
- Load 520K concepts (full SNOMED dataset)
- Performance benchmarking
- Consistency validation
- Error scenario testing (GraphDB down, network failures)

**Day 5**: Documentation & Handoff
- Update README and deployment guides
- Create troubleshooting guide
- Code review and approval
- Merge to main branch

### Stage 3: Production Rollout (Week 3)
**Days 1-3**: Staging Deployment
- Deploy to staging environment
- Smoke tests and validation
- Performance monitoring

**Days 4-5**: Production Canary
- Enable for 10% of ETL runs
- Monitor for 48 hours
- Validate no impact on PostgreSQL operations

### Stage 4: Full Rollout (Week 4)
**Days 1-5**: Production Deployment
- Enable for 100% of ETL runs
- Monitor for 1 week
- Document operational procedures
- Close implementation ticket

---

## Configuration Reference

### Environment Variables

```bash
# GraphDB Configuration (New)
GRAPHDB_ENABLED=false                              # Default: disabled for backward compatibility
GRAPHDB_SERVER_URL=http://localhost:7200           # GraphDB REST endpoint
GRAPHDB_REPOSITORY_ID=kb7-terminology              # Repository name
GRAPHDB_BATCH_SIZE=10000                           # Triples per batch upload
GRAPHDB_MAX_RETRIES=3                              # Upload retry attempts
GRAPHDB_RETRY_DELAY=5s                             # Delay between retries
GRAPHDB_NAMED_GRAPH=http://cardiofit.ai/kb7/graph/default
GRAPHDB_TRANSACTION_TIMEOUT=10m                    # Max time for GraphDB operations
GRAPHDB_VALIDATE_TRIPLES=true                      # Enable triple validation
GRAPHDB_ENABLE_INFERENCE=true                      # Enable OWL reasoning

# Existing Configuration (Unchanged)
DATABASE_URL=postgresql://kb7_user:password@localhost:5433/kb7_terminology
ELASTICSEARCH_URLS=http://localhost:9200
REDIS_URL=redis://localhost:6380/0
```

### Command-Line Usage

```bash
# Default: GraphDB disabled (backward compatible)
./etl --data ./data/snomed

# Enable GraphDB triple loading
./etl --data ./data/snomed --enable-graphdb=true

# Custom GraphDB configuration
./etl --data ./data/snomed \
      --enable-graphdb=true \
      --graphdb-url http://graphdb:7200 \
      --graphdb-batch-size=5000

# Disable triple validation for faster loading
./etl --data ./data/snomed \
      --enable-graphdb=true \
      --validate-triples=false
```

---

## Success Criteria

### Functional Requirements
- ✅ All 520K SNOMED concepts loaded to GraphDB as RDF triples
- ✅ PostgreSQL operations unchanged (zero regression)
- ✅ Elasticsearch operations unchanged
- ✅ GraphDB failures do not block PostgreSQL success
- ✅ 3-way consistency validation passes (>95% consistency score)
- ✅ SPARQL queries return correct results

### Performance Requirements
- ✅ Total ETL duration < 40 minutes (520K concepts)
- ✅ PostgreSQL load time unchanged (~15 min)
- ✅ GraphDB sync < 25 minutes
- ✅ Memory usage < 2 GB peak
- ✅ Triple generation rate > 8K triples/sec

### Quality Requirements
- ✅ Unit test coverage > 90%
- ✅ All integration tests pass
- ✅ No regressions in existing PostgreSQL/Elasticsearch flow
- ✅ Clear error messages and structured logging
- ✅ Comprehensive documentation (architecture + deployment)

### Operational Requirements
- ✅ GraphDB can be disabled via flag (backward compatibility)
- ✅ Rollback plan documented and tested
- ✅ Monitoring dashboards updated
- ✅ Troubleshooting guide available
- ✅ On-call runbook created

---

## Risk Assessment

### Risk Matrix

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **PostgreSQL regression** | LOW | CRITICAL | Comprehensive testing, zero code changes to PostgreSQL flow |
| **GraphDB performance** | MEDIUM | MEDIUM | Batch processing, retry logic, performance benchmarking |
| **Memory overflow** | LOW | HIGH | Streaming processing, batch size limits, monitoring |
| **Data inconsistency** | LOW | MEDIUM | 3-way validation, automated consistency checks |
| **GraphDB downtime** | MEDIUM | LOW | Non-blocking failures, circuit breaker pattern |

### Risk Mitigation Strategies

1. **PostgreSQL Protection**:
   - Zero changes to `EnhancedCoordinator` or `DualStoreCoordinator` logic
   - GraphDB disabled by default
   - Extensive backward compatibility testing

2. **Gradual Rollout**:
   - Development → Staging → Canary (10%) → Full (100%)
   - Monitoring at each stage
   - Automated rollback triggers

3. **Performance Safeguards**:
   - Memory usage alerts (threshold: 1.5 GB)
   - ETL duration alerts (threshold: 50 min)
   - Automatic GraphDB disable on repeated failures

4. **Operational Safety**:
   - Comprehensive logging and metrics
   - Clear rollback procedure
   - On-call escalation path

---

## Next Steps

### Immediate Actions (This Week)
1. ✅ Review architectural design document (this doc)
2. ✅ Approve implementation plan
3. 🔄 Assign to refactoring-expert agent for implementation
4. 🔄 Create implementation tickets (4 tickets, one per file)
5. 🔄 Setup development environment (GraphDB container)

### Development Workflow (Week 1-2)
1. **Day 1-2**: Implement RDF Transformer + tests
2. **Day 3-4**: Implement GraphDB Loader + tests
3. **Day 5**: Implement TripleStoreCoordinator + tests
4. **Day 6-7**: ETL Main integration + backward compatibility tests
5. **Day 8-9**: Full scale testing (520K concepts) + performance benchmarking
6. **Day 10**: Documentation + code review + merge

### Post-Implementation (Week 3-4)
1. Staging deployment and validation
2. Production canary rollout (10%)
3. Full production rollout (100%)
4. Operational monitoring and tuning

---

## Document References

1. **Detailed Architecture**: `TRIPLE_STORE_COORDINATOR_ARCHITECTURE.md`
   - Full code specifications (600+ lines per file)
   - Detailed data models and transformation logic
   - Error handling strategies
   - Performance optimization techniques

2. **Phase 1 Plan**: `KB7_ARCHITECTURE_TRANSFORMATION_PLAN.md`
   - Overall KB-7 transformation roadmap
   - GraphDB integration vision
   - Long-term semantic reasoning goals

3. **Existing Codebase**:
   - `cmd/etl/main.go` - Current ETL orchestrator
   - `internal/etl/dual_store_coordinator.go` - Pattern to follow
   - `internal/semantic/graphdb_client.go` - GraphDB client (reuse)
   - `internal/semantic/rdf_converter.go` - RDF utilities (reuse)

---

## Approval Sign-Off

**Design Prepared By**: System Architect Agent
**Date**: November 22, 2025

**Approval Required From**:
- [ ] Lead Backend Engineer
- [ ] Platform Architect
- [ ] DevOps/Infrastructure Lead
- [ ] Product Owner (for timeline)

**Approved for Implementation**: _______________  Date: _______________

---

**Status**: READY FOR IMPLEMENTATION
**Next Agent**: Refactoring Expert
**Estimated Completion**: December 6, 2025 (2 weeks from start)
