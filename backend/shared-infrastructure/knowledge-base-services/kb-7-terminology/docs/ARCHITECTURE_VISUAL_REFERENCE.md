# TripleStoreCoordinator Architecture - Visual Reference
**Quick Reference Guide for Developers**

---

## System Architecture Diagram

```
┌───────────────────────────────────────────────────────────────────────────┐
│                          KB-7 ETL PIPELINE                                 │
│                     (Triple Store Coordinator)                             │
└───────────────────────────────────────────────────────────────────────────┘

                              ┌──────────────┐
                              │  SNOMED CT   │
                              │  RF2 Files   │
                              │  (520K       │
                              │  concepts)   │
                              └──────┬───────┘
                                     │
                                     ▼
                    ┌────────────────────────────────┐
                    │   cmd/etl/main.go             │
                    │   - Parse command flags        │
                    │   - Load configuration         │
                    │   - Initialize coordinator     │
                    └────────────┬───────────────────┘
                                 │
                                 ▼
        ┌────────────────────────────────────────────────────────────┐
        │          TripleStoreCoordinator                            │
        │   (internal/etl/triple_store_coordinator.go)              │
        │                                                            │
        │   LoadAllTerminologiesTripleStore()                       │
        └────┬──────────────────────────────────────────────────┬───┘
             │                                                   │
    ┌────────▼────────┐                              ┌──────────▼─────────┐
    │  PHASE 1        │                              │  PHASE 2           │
    │  PostgreSQL +   │                              │  GraphDB Sync      │
    │  Elasticsearch  │                              │  (NEW)             │
    └────────┬────────┘                              └──────────┬─────────┘
             │                                                   │
             ▼                                                   ▼
┌────────────────────────────┐                    ┌────────────────────────┐
│  DualStoreCoordinator      │                    │  syncToGraphDB()       │
│  (EXISTING - REUSE)        │                    │                        │
│                            │                    │  1. Read PostgreSQL    │
│  1. PostgreSQL Load        │                    │  2. Convert to RDF     │
│  2. Elasticsearch Sync     │                    │  3. Upload to GraphDB  │
│  3. Consistency Check      │                    │                        │
└────────┬───────────────────┘                    └────────┬───────────────┘
         │                                                  │
         ▼                                                  ▼
┌─────────────────┐   ┌─────────────────┐    ┌─────────────────────────────┐
│   PostgreSQL    │   │ Elasticsearch   │    │      GraphDB                │
│   (Primary)     │   │ (Search Index)  │    │   (Semantic Triple Store)   │
│                 │   │                 │    │                             │
│   520K          │   │   520K          │    │   12.6M Triples            │
│   Concepts      │   │   Documents     │    │   (8-24 per concept)       │
│                 │   │                 │    │                             │
│   Port: 5433    │   │   Port: 9200    │    │   Port: 7200               │
└─────────────────┘   └─────────────────┘    └─────────────────────────────┘
         │                       │                           │
         └───────────────────────┴───────────────────────────┘
                                 │
                                 ▼
                    ┌────────────────────────────┐
                    │   PHASE 3                  │
                    │   3-Way Consistency Check  │
                    │                            │
                    │   - Count PostgreSQL       │
                    │   - Count Elasticsearch    │
                    │   - Count GraphDB          │
                    │   - Validate consistency   │
                    └────────────────────────────┘
```

---

## Component Architecture (Zoom In)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    TripleStoreCoordinator                               │
│                                                                         │
│  ┌───────────────────────────────────────────────────────────────┐    │
│  │  Embedded: *DualStoreCoordinator                              │    │
│  │    └─ *EnhancedCoordinator (PostgreSQL)                       │    │
│  │    └─ *ElasticsearchIntegration                               │    │
│  └───────────────────────────────────────────────────────────────┘    │
│                                                                         │
│  ┌───────────────────────────────────────────────────────────────┐    │
│  │  New Components:                                              │    │
│  │                                                               │    │
│  │  graphDBClient:      *semantic.GraphDBClient                  │    │
│  │  rdfTransformer:     *transformer.SNOMEDToRDFTransformer     │    │
│  │  graphDBLoader:      *etl.GraphDBLoader                      │    │
│  │  tripleStoreStatus:  *TripleStoreStatus                      │    │
│  └───────────────────────────────────────────────────────────────┘    │
│                                                                         │
│  ┌───────────────────────────────────────────────────────────────┐    │
│  │  Key Methods:                                                 │    │
│  │                                                               │    │
│  │  LoadAllTerminologiesTripleStore(ctx, dataSources)           │    │
│  │  syncToGraphDB(ctx)                                          │    │
│  │  readConceptsFromPostgreSQL(ctx)                             │    │
│  │  performTripleStoreConsistencyCheck(ctx)                     │    │
│  └───────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Data Transformation Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    PostgreSQL Concept → RDF Triples                     │
└─────────────────────────────────────────────────────────────────────────┘

STEP 1: Read from PostgreSQL
┌──────────────────────────────────────────────────┐
│  Concept (PostgreSQL Row)                        │
│  ────────────────────────────                    │
│  id:              "uuid-123"                     │
│  system:          "SNOMED"                       │
│  code:            "387517004"                    │
│  preferred_term:  "Paracetamol"                  │
│  definition:      "Analgesic and antipyretic"    │
│  active:          true                           │
│  version:         "20240131"                     │
│  parent_codes:    ["7947003"]                    │
│  properties:      {                              │
│    module_id: "900000000000012004",              │
│    synonyms: ["Acetaminophen", "APAP"]           │
│  }                                               │
└──────────────────────────────────────────────────┘
                     │
                     ▼
STEP 2: Transform to RDF Triples
┌──────────────────────────────────────────────────┐
│  RDFTransformer.ConceptToTriples()               │
│  ────────────────────────────────────            │
│  Input:  Concept struct                          │
│  Output: RDFTripleSet (8-24 triples)             │
│                                                  │
│  Triples Generated:                              │
│  ────────────────────                            │
│  1. Type:         sct:387517004 a owl:Class      │
│  2. Label:        rdfs:label "Paracetamol"@en    │
│  3. PrefLabel:    skos:prefLabel "Paracetamol"@en│
│  4. Definition:   skos:definition "Analgesic.."@en│
│  5. ConceptID:    kb7:conceptId "387517004"      │
│  6. System:       kb7:system "SNOMED-CT"         │
│  7. Active:       kb7:active "true"^^xsd:boolean │
│  8. SubClassOf:   rdfs:subClassOf sct:7947003    │
│  9. Synonym:      skos:altLabel "Acetaminophen"@en│
│  10. Synonym:     skos:altLabel "APAP"@en        │
└──────────────────────────────────────────────────┘
                     │
                     ▼
STEP 3: Serialize to Turtle
┌──────────────────────────────────────────────────┐
│  RDFTransformer.BatchToTurtle()                  │
│  ────────────────────────────────                │
│  Input:  []RDFTriple (batch of 1000 concepts)    │
│  Output: Turtle Document (string)                │
│                                                  │
│  Generated Turtle:                               │
│  ────────────────                                │
│  @prefix sct: <http://snomed.info/id/> .        │
│  @prefix rdfs: <...> .                           │
│  @prefix skos: <...> .                           │
│  @prefix kb7: <...> .                            │
│                                                  │
│  sct:387517004 a owl:Class ;                     │
│      rdfs:label "Paracetamol"@en ;               │
│      skos:prefLabel "Paracetamol"@en ;           │
│      skos:definition "Analgesic..."@en ;         │
│      kb7:conceptId "387517004" ;                 │
│      kb7:active "true"^^xsd:boolean ;            │
│      rdfs:subClassOf sct:7947003 .               │
└──────────────────────────────────────────────────┘
                     │
                     ▼
STEP 4: Upload to GraphDB
┌──────────────────────────────────────────────────┐
│  GraphDBClient.UploadTurtle()                    │
│  ────────────────────────────────                │
│  HTTP POST /repositories/kb7-terminology/        │
│           statements                             │
│                                                  │
│  Content-Type: text/turtle                       │
│  Body: [Turtle document from Step 3]            │
│                                                  │
│  Response: 204 No Content (Success)              │
└──────────────────────────────────────────────────┘
                     │
                     ▼
            GraphDB Repository
      (Triples stored and indexed)
```

---

## Error Handling Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Error Handling Strategy                         │
└─────────────────────────────────────────────────────────────────────────┘

LoadAllTerminologiesTripleStore()
        │
        ├─ PHASE 1: PostgreSQL + Elasticsearch
        │           │
        │           ├─ Success? ───────────────┐
        │           │                          │
        │           └─ Failure? ──────────┐    │
        │                                 │    │
        │                             🔴 ABORT │
        │                             CRITICAL │
        │                             Rollback │
        │                             Return Error
        │                                      │
        │                                      ▼
        ├─ PHASE 2: GraphDB Sync         Continue
        │           │
        │           ├─ Success? ───────────────┐
        │           │                          │
        │           └─ Failure? ──────────┐    │
        │                                 │    │
        │                             🟡 WARN  │
        │                             NON-CRITICAL
        │                             Log Error│
        │                             Mark Degraded
        │                             Continue │
        │                                      │
        │                                      ▼
        └─ PHASE 3: Consistency Check    Continue
                    │
                    ├─ Success? ───────────────┐
                    │                          │
                    └─ Failure? ──────────┐    │
                                          │    │
                                      🟢 INFO  │
                                      Log Warning
                                      Continue │
                                               │
                                               ▼
                                      ETL SUCCESS
                               (PostgreSQL loaded)

╔═══════════════════════════════════════════════════════════════════════╗
║  KEY PRINCIPLE: PostgreSQL success = Overall ETL success            ║
║  GraphDB and Elasticsearch failures are non-blocking                ║
╚═══════════════════════════════════════════════════════════════════════╝
```

---

## Batch Processing Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Batch Processing Architecture                        │
└─────────────────────────────────────────────────────────────────────────┘

PostgreSQL (520,000 concepts)
        │
        │ Read in batches of 1,000
        ▼
┌────────────────────────────────────────────────┐
│  Batch 1 (Concepts 1-1,000)                    │
│  ────────────────────────────                  │
│  1. Read from PostgreSQL                       │
│  2. Convert to ~8,000 triples                  │
│  3. Serialize to Turtle                        │
│  4. Upload to GraphDB                          │
│  5. Log progress                               │
└────────────────────────────────────────────────┘
        │
        ▼
┌────────────────────────────────────────────────┐
│  Batch 2 (Concepts 1,001-2,000)                │
│  ────────────────────────────────────          │
│  [Same process]                                │
└────────────────────────────────────────────────┘
        │
        ▼
        ...
        │
        ▼
┌────────────────────────────────────────────────┐
│  Batch 520 (Concepts 519,001-520,000)          │
│  ──────────────────────────────────────        │
│  [Same process]                                │
└────────────────────────────────────────────────┘

╔═══════════════════════════════════════════════════════════════════════╗
║  Memory Profile:                                                     ║
║  - Per Batch: ~50 MB (1,000 concepts + triples + Turtle string)     ║
║  - Peak Total: ~1.2 GB (includes PostgreSQL connection pool)        ║
║  - Target: < 2 GB to avoid OOM                                      ║
╚═══════════════════════════════════════════════════════════════════════╝

Performance Metrics per Batch:
┌────────────────────────────────────────────────┐
│  PostgreSQL Read:      200-500ms               │
│  RDF Conversion:       500-800ms               │
│  Turtle Serialization: 100-200ms               │
│  GraphDB Upload:       1000-2000ms             │
│  ─────────────────────────────────             │
│  Total per Batch:      ~3 seconds              │
│  Total for 520K:       ~26 minutes             │
└────────────────────────────────────────────────┘
```

---

## File Structure Map

```
kb-7-terminology/
├── cmd/
│   └── etl/
│       └── main.go                          # MODIFY: Add --enable-graphdb flag
│                                            #         Add GraphDB config
│                                            #         Use TripleStoreCoordinator
│
├── internal/
│   ├── etl/
│   │   ├── enhanced_coordinator.go          # REUSE: No changes
│   │   ├── dual_store_coordinator.go        # REUSE: No changes
│   │   ├── transaction_manager.go           # REUSE: No changes
│   │   │
│   │   ├── triple_store_coordinator.go      # CREATE: Main coordinator
│   │   ├── graphdb_loader.go                # CREATE: GraphDB operations
│   │   └── triple_validator.go              # CREATE: Consistency checks
│   │
│   ├── transformer/
│   │   └── snomed_to_rdf.go                 # CREATE: RDF conversion
│   │
│   └── semantic/
│       ├── graphdb_client.go                # REUSE: Existing GraphDB client
│       └── rdf_converter.go                 # REUSE: RDF utilities
│
├── docs/
│   ├── TRIPLE_STORE_COORDINATOR_ARCHITECTURE.md    # Detailed design
│   ├── IMPLEMENTATION_PLAN_PHASE_1_2.md            # This summary
│   └── ARCHITECTURE_VISUAL_REFERENCE.md            # Visual diagrams
│
└── scripts/
    ├── test-graphdb-integration.sh          # Integration tests
    └── benchmark-graphdb-loading.sh         # Performance tests
```

---

## Code Reuse Matrix

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Code Reuse Strategy                            │
└─────────────────────────────────────────────────────────────────────────┘

Component                              Status      LOC    Reuse %
──────────────────────────────────────────────────────────────────────
✅ EnhancedCoordinator                 REUSE       650    100%
✅ DualStoreCoordinator                REUSE       590    100%
✅ TransactionManager                  REUSE       640    100%
✅ GraphDBClient                       REUSE       405    100%
✅ RDFConverter (utilities)            REUSE       380    100%
📝 main.go                            MODIFY       355     95% (add 20 lines)
──────────────────────────────────────────────────────────────────────
🆕 TripleStoreCoordinator             CREATE       600      0%
🆕 GraphDBLoader                       CREATE       300      0%
🆕 TripleValidator                     CREATE       200      0%
🆕 SNOMEDToRDFTransformer              CREATE       500      0%
──────────────────────────────────────────────────────────────────────

Total Existing Code:                           2,665 LOC (62%)
Total New Code:                                1,600 LOC (38%)
──────────────────────────────────────────────────────────────────────
Overall Code Reuse:                                      62%

╔═══════════════════════════════════════════════════════════════════════╗
║  IMPACT ASSESSMENT:                                                  ║
║  - Zero risk to existing PostgreSQL/Elasticsearch code              ║
║  - GraphDB disabled by default (backward compatible)                ║
║  - New code isolated to 4 new files                                 ║
║  - Minimal changes to main.go (20 lines)                            ║
╚═══════════════════════════════════════════════════════════════════════╝
```

---

## Configuration Quick Reference

### Enabling GraphDB

**Option 1: Command-Line Flag**
```bash
./etl --data ./data/snomed --enable-graphdb=true
```

**Option 2: Environment Variable**
```bash
export GRAPHDB_ENABLED=true
./etl --data ./data/snomed
```

**Option 3: Config File** (config.yaml)
```yaml
graphdb:
  enabled: true
  server_url: http://localhost:7200
  repository_id: kb7-terminology
  batch_size: 10000
```

### Disabling GraphDB (Default)

```bash
# No flag = GraphDB disabled (backward compatible)
./etl --data ./data/snomed

# Explicit disable
./etl --data ./data/snomed --enable-graphdb=false
```

### Performance Tuning

```bash
# Increase batch size for faster loading (more memory)
./etl --data ./data/snomed \
      --enable-graphdb=true \
      --graphdb-batch-size=20000

# Decrease batch size for lower memory usage
./etl --data ./data/snomed \
      --enable-graphdb=true \
      --graphdb-batch-size=5000
```

---

## Monitoring Dashboard

### Key Metrics to Track

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      ETL Pipeline Metrics                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  PostgreSQL Concepts:     520,000  ✅                                   │
│  Elasticsearch Docs:      520,000  ✅                                   │
│  GraphDB Triples:      12,660,000  ✅                                   │
│                                                                         │
│  Consistency Score:          98.5% ✅                                   │
│  Discrepancy:                 7,800 triples (0.06%)                    │
│                                                                         │
│  PostgreSQL Load Time:     14m 32s ✅                                   │
│  GraphDB Sync Time:        23m 18s ✅                                   │
│  Total ETL Duration:       37m 50s ✅                                   │
│                                                                         │
│  Peak Memory Usage:         1.8 GB ✅                                   │
│  Triples/Second:            9,043  ✅                                   │
│  Concepts/Second:             570  ✅                                   │
│                                                                         │
│  PostgreSQL Status:        Healthy ✅                                   │
│  Elasticsearch Status:     Healthy ✅                                   │
│  GraphDB Status:           Healthy ✅                                   │
│                                                                         │
│  Overall Health:           Healthy ✅                                   │
└─────────────────────────────────────────────────────────────────────────┘
```

### Alert Thresholds

```yaml
alerts:
  critical:
    - postgresql_duration > 20m
    - total_duration > 50m
    - memory_usage > 3.5 GB
    - postgresql_failure: true

  warning:
    - graphdb_duration > 30m
    - consistency_score < 95%
    - triples_per_second < 5000
    - graphdb_failure: true

  info:
    - elasticsearch_failure: true
    - consistency_check_failure: true
```

---

## Troubleshooting Quick Guide

### Issue: GraphDB Upload Fails

**Symptoms**:
```
ERROR: GraphDB upload failed: connection timeout
GraphDB Status: unhealthy
Overall Health: degraded
```

**Diagnosis**:
```bash
# Check GraphDB is running
curl http://localhost:7200/rest/repositories/kb7-terminology

# Check repository exists
curl http://localhost:7200/rest/repositories | jq '.'

# Check network connectivity
ping localhost -c 3
```

**Resolution**:
```bash
# Restart GraphDB
docker-compose restart kb7-graphdb

# Verify repository
./scripts/create-graphdb-repository.sh

# Re-run ETL
./etl --data ./data/snomed --enable-graphdb=true
```

### Issue: Memory Overflow

**Symptoms**:
```
ERROR: Out of memory
Process killed by OOM killer
Exit code: 137
```

**Diagnosis**:
```bash
# Check current batch size
grep "batch_size" logs/etl.log

# Check memory usage pattern
grep "memory_usage" logs/etl.log
```

**Resolution**:
```bash
# Reduce batch size
./etl --data ./data/snomed \
      --enable-graphdb=true \
      --graphdb-batch-size=500

# Increase container memory limit
docker-compose up -d --build --force-recreate
```

### Issue: Consistency Check Fails

**Symptoms**:
```
WARN: Consistency check failed
Discrepancy: 250,000 triples
Consistency Score: 82%
```

**Diagnosis**:
```bash
# Count PostgreSQL concepts
psql -U kb7_user -c "SELECT COUNT(*) FROM concepts;"

# Count GraphDB triples
curl -X POST http://localhost:7200/repositories/kb7-terminology \
  --data-urlencode "query=SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }"
```

**Resolution**:
```bash
# Clear GraphDB and reload
./etl --clear-graphdb --enable-graphdb=true
./etl --data ./data/snomed --enable-graphdb=true
```

---

**Document Status**: COMPLETE - Ready for Implementation
**Last Updated**: November 22, 2025
**Next Review**: Upon implementation completion
