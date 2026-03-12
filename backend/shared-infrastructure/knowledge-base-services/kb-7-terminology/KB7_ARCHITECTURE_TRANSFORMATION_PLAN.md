# KB-7 Architecture Transformation Plan
**Objective**: Transform KB-7 from PostgreSQL-centric to GraphDB-centric semantic reasoning platform

**Status**: Implementation Roadmap
**Timeline**: 16 weeks (6 phases)
**Last Updated**: November 24, 2025

---

## Executive Summary

**Current State**: KB-7 is a fully functional terminology lookup service with PostgreSQL as primary storage, supporting 520K concepts (SNOMED-CT, RxNorm, LOINC) via REST API.

**Proposed State**: Transform into a semantic reasoning platform with GraphDB as primary storage, enabling SPARQL queries, OWL inference, and real-time CDC streaming.

**Key Changes**:
- 🔄 **Flip Architecture**: GraphDB becomes primary, PostgreSQL becomes metadata registry
- ➕ **Add Capabilities**: Semantic reasoning, automated updates, CDC streaming
- ♻️ **Reuse Existing**: ETL pipeline, REST API, caching layer, data files
- 🗑️ **Remove Unused**: Duplicate clients, unexposed Elasticsearch code

---

## Current State Analysis

### What Works Today ✅

| Component | Status | Location | Notes |
|-----------|--------|----------|-------|
| **PostgreSQL Storage** | 100% Operational | Port 5433 | 520K concepts loaded |
| **ETL Pipeline** | 100% Operational | `cmd/etl/main.go` | Batch loading working |
| **REST API** | 80% Operational | `cmd/server/main.go` | Missing semantic endpoints |
| **Redis Cache** | 100% Operational | Port 6380 | 3-tier caching |
| **Elasticsearch** | 50% Operational | Port 9200 | Dual-write works, API doesn't use it |
| **GraphDB Container** | 10% Operational | Port 7200 | Running but empty |
| **GraphDB Client** | Code Complete | `internal/semantic/` | Tested, but unused |

### Architecture Gap

**Current Flow**:
```
External Files → ETL → PostgreSQL → REST API
                          ↓
                    Elasticsearch (dual-write)
```

**Target Flow** (from document):
```
External APIs → Knowledge Factory → GraphDB → PostgreSQL (metadata) → CDC → Kafka
```

### What Needs to Change

| Current | Target | Action Required |
|---------|--------|-----------------|
| PostgreSQL = concepts | GraphDB = concepts | Migrate data, flip roles |
| Static files | External API downloads | Add Knowledge Factory |
| Manual updates | Automated pipeline | ROBOT + SNOMED-OWL-Toolkit |
| No semantic queries | SPARQL endpoints | Activate GraphDB integration |
| No streaming | CDC with Kafka | Add Debezium connector |

---

## Technology Stack

### Core Infrastructure
| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| **Semantic Storage** | OntoText GraphDB | 10.7.0 | Primary RDF triplestore with OWL2-RL reasoning |
| **Metadata Registry** | PostgreSQL | 15 | Kernel snapshots, deployment metadata |
| **Cache Layer** | Redis | 7.0 | 3-tier caching (L1: exact match, L2: semantic, L3: search) |
| **Search Engine** | Elasticsearch | 8.0 | Full-text search and analytics |

### Serverless Knowledge Factory
| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| **Source Downloads** | AWS Lambda | Python 3.11 | SNOMED/RxNorm/LOINC API downloads |
| **Orchestration** | AWS Step Functions | - | Parallel download coordination |
| **Storage** | AWS S3 | - | Source files and transformed kernels |
| **Secrets** | AWS Secrets Manager | - | API credentials with rotation |
| **Scheduling** | CloudWatch Events | - | Monthly cron triggers |
| **CI/CD** | GitHub Actions | - | 7-stage transformation pipeline |

### Terminology Transformation Tools
| Tool | Version | Purpose |
|------|---------|---------|
| **SNOMED-OWL-Toolkit** | v4.0.6 | Convert SNOMED RF2 snapshot → OWL ontology |
| **ROBOT** | v1.9.5 | Ontology merge, ELK reasoning, SPARQL validation |
| **RxNorm Converter** | Custom (Python) | Convert RxNorm RRF → RDF/Turtle |
| **LOINC Converter** | Custom (Python) | Convert LOINC CSV → RDF via ROBOT templates |
| **ELK Reasoner** | v0.5.0 | OWL inference engine (via ROBOT) |

### API & Service Layer
| Component | Technology | Purpose |
|-----------|-----------|---------|
| **REST API** | Go (Gin framework) | Terminology lookup and search |
| **Query Router** | Go | Route queries to PostgreSQL/GraphDB/Elasticsearch |
| **CDC Streaming** | Debezium + Kafka | Real-time snapshot change events |
| **GraphQL** | Apollo Federation | Federated graph queries (optional) |

### Development & Operations
| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Containerization** | Docker + Docker Compose | Local dev environment |
| **Monitoring** | Prometheus + Grafana | Metrics and dashboards |
| **Logging** | ELK Stack | Centralized log aggregation |
| **IaC** | CloudFormation | AWS infrastructure provisioning |

---

## Cost Analysis

### Implementation Costs

#### Phase 1: GraphDB Foundation & Serverless Knowledge Factory (4 weeks)
| Item | Cost Type | Amount | Notes |
|------|-----------|--------|-------|
| Senior Backend Engineer (4 weeks) | Labor | $24,000 | $150/hr × 40 hrs/wk × 4 wks |
| DevOps Engineer (2 weeks) | Labor | $12,000 | AWS infrastructure setup |
| **Phase 1 Total** | - | **$36,000** | One-time implementation |

#### Phases 2-6 (12 weeks)
| Item | Cost Type | Amount | Notes |
|------|-----------|--------|-------|
| Backend Engineering (12 weeks) | Labor | $72,000 | Query layer, CDC, cleanup |
| **Phases 2-6 Total** | - | **$72,000** | Remaining implementation |

#### **Total Implementation Cost**: **$108,000** (16 weeks)

---

### Operational Costs (Monthly)

#### AWS Serverless Infrastructure
| Component | Monthly Cost | Annual Cost | Details |
|-----------|--------------|-------------|---------|
| **S3 Storage** | $5.00 | $60 | 200GB @ $0.023/GB (sources + kernels) |
| **Lambda Invocations** | $1.50 | $18 | 4 functions × 1 run/month × $0.20 each |
| **Step Functions** | $0.50 | $6 | 1 execution/month (parallel downloads) |
| **Secrets Manager** | $0.40 | $5 | 4 secrets @ $0.40/secret/month |
| **CloudWatch Logs** | $0.50 | $6 | Log retention and metrics |
| **GitHub Actions** | $12.00 | $144 | Larger runners (16GB RAM) for ROBOT reasoning @ $0.16/min |
| **Data Transfer** | $1.00 | $12 | S3 → Lambda → GitHub Actions |
| **Total AWS Monthly** | **$20.90** | **$251/year** | Serverless automation + GitHub runners |

#### Existing Infrastructure (No Change)
| Component | Monthly Cost | Annual Cost | Details |
|-----------|--------------|-------------|---------|
| GraphDB (self-hosted) | $0 | $0 | Docker container on existing server |
| PostgreSQL (self-hosted) | $0 | $0 | Docker container on existing server |
| Redis (self-hosted) | $0 | $0 | Docker container on existing server |
| Elasticsearch (self-hosted) | $0 | $0 | Docker container on existing server |
| **Total Infrastructure** | **$0** | **$0** | Self-hosted on existing hardware |

#### **Total Operational Cost**: **$20.90/month** or **$251/year**

**Note**: GitHub Larger Runners cost is conservative estimate. Standard runners (free tier) may work for most months; larger runners only needed if OOM occurs during reasoning stage.

---

### Labor Savings (Monthly)

#### Manual Terminology Updates (Current State)
| Task | Frequency | Time per Task | Monthly Hours | Monthly Cost |
|------|-----------|---------------|---------------|--------------|
| Download SNOMED/RxNorm/LOINC | Monthly | 2 hours | 2 hours | $300 |
| Manual validation & QA | Monthly | 4 hours | 4 hours | $600 |
| PostgreSQL data import | Monthly | 3 hours | 3 hours | $450 |
| Troubleshooting failures | As needed | 1.5 hours avg | 1.5 hours | $225 |
| **Total Manual Labor** | - | **10.5 hrs/month** | - | **$1,575/month** |

#### Automated Updates (Future State with Knowledge Factory)
| Task | Frequency | Time per Task | Monthly Hours | Monthly Cost |
|------|-----------|---------------|---------------|--------------|
| Monitor pipeline success | Monthly | 0.5 hours | 0.5 hours | $75 |
| Handle failures (5% failure rate) | Occasional | 0 hours (auto-retry) | 0 hours | $0 |
| Review kernel before deployment | Monthly | 0.5 hours | 0.5 hours | $75 |
| **Total Automated Labor** | - | **1 hr/month** | - | **$150/month** |

#### **Monthly Labor Savings**: **$1,575 - $150 = $1,425/month**

---

### Return on Investment (ROI)

#### Cost-Benefit Summary
| Category | Amount | Calculation |
|----------|--------|-------------|
| **One-Time Implementation** | $108,000 | 16 weeks × $6,750/week |
| **Monthly Operational Cost** | $20.90 | AWS + GitHub Larger Runners |
| **Monthly Labor Savings** | $1,425 | Eliminated manual work |
| **Net Monthly Savings** | $1,404 | $1,425 - $20.90 |
| **Annual Savings** | $16,848 | $1,404 × 12 months |

#### **ROI Break-Even: 7.7 months**
- Implementation cost: $108,000
- Monthly savings: $1,404
- Break-even: $108,000 ÷ $1,404 = **7.7 months**

#### **3-Year Total Cost of Ownership (TCO)**
| Item | Cost |
|------|------|
| Implementation (Year 0) | $108,000 |
| Operational (3 years) | $753 ($251/year × 3) |
| **Total Cost** | **$108,753** |
| Labor Savings (3 years) | -$51,300 ($1,425/month × 36) |
| **Net 3-Year Cost** | **$57,453** |
| **3-Year Savings** | **$50,547** |

---

## Implementation Roadmap

## Phase 1: GraphDB Foundation & Serverless Knowledge Factory (Weeks 1-4)
**Goal**: Activate GraphDB as primary semantic storage with automated terminology updates via serverless pipeline

### 1.1 Create GraphDB Repository

**Location**: GraphDB Workbench (http://localhost:7200)

**Configuration**:
```yaml
Repository ID: kb7-terminology
Repository Type: GraphDB Free
Ruleset: owl2-rl-optimized
Base URL: http://cardiofit.ai/ontology/
Storage Folder: storage
Entity Index Size: 10000000
Query Timeout: 0 (unlimited)
```

**Implementation**:
- Option A: Manual creation via web UI
- Option B: Automated script using GraphDB REST API
- Option C: Repository config file (`.ttl` format)

**Files to Create**:
```
scripts/graphdb/
├── create-repository.sh          # Automated repository setup
├── repository-config.ttl          # Repository configuration
└── validate-repository.sh         # Health check script
```

### 1.2 Extend ETL Pipeline (REUSE Existing)

**Current Code** (reuse):
- ✅ `cmd/etl/main.go` - Main ETL orchestrator
- ✅ `internal/etl/enhanced_coordinator.go` - Batch processing
- ✅ `internal/etl/snomed_loader.go` - SNOMED-CT loader
- ✅ `internal/etl/rxnorm_loader.go` - RxNorm loader
- ✅ `internal/etl/loinc_loader.go` - LOINC loader

**New Code** (add):
```go
// internal/etl/triple_store_coordinator.go
type TripleStoreCoordinator struct {
    postgresCoordinator *DualStoreCoordinator  // Reuse existing
    graphDBClient       *semantic.GraphDBClient
    rdfConverter        *RDFConverter
    logger              *zap.Logger
}

func (t *TripleStoreCoordinator) LoadAllTerminologies(ctx context.Context) error {
    // 1. Load to PostgreSQL + Elasticsearch (existing flow)
    err := t.postgresCoordinator.LoadAllTerminologiesDualStore(ctx, dataSources)

    // 2. Convert PostgreSQL records to RDF triples
    triples, err := t.rdfConverter.ConvertFromPostgreSQL(ctx)

    // 3. Load triples to GraphDB
    err = t.graphDBClient.LoadTriples(ctx, triples)

    // 4. Validate consistency
    return t.validateConsistency(ctx)
}
```

**Files to Create**:
```
internal/etl/
├── triple_store_coordinator.go    # NEW: GraphDB integration
├── rdf_converter.go                # NEW: PostgreSQL → RDF conversion
└── triple_validator.go             # NEW: Consistency checks
```

### 1.3 Serverless Knowledge Factory

**Purpose**: Automate monthly terminology updates from external authoritative sources (NCTS, UMLS, LOINC) using serverless pipeline

**Architecture**: AWS Lambda + Step Functions + GitHub Actions + ROBOT + SNOMED-OWL-Toolkit

**Key Innovation**: Replace manual PostgreSQL data files with automated downloads from:
- **SNOMED CT**: UK NHS TRUD API (monthly releases)
- **RxNorm**: NIH/NLM UMLS API (monthly updates)
- **LOINC**: Regenstrief Institute API (biannual releases)

---

#### 1.3.1 Bootstrap GraphDB (Week 1)

**Purpose**: One-time migration of existing 520K concepts to unblock Phase 2 development

**Script**: `scripts/bootstrap/postgres-to-graphdb.go`

```go
package main

import (
    "context"
    "kb-7-terminology/internal/semantic"
    "kb-7-terminology/internal/database"
)

func main() {
    // 1. Connect to databases
    pgDB := database.Connect(os.Getenv("PG_URL"))
    graphDB := semantic.NewGraphDBClient("http://localhost:7200", "kb7-terminology", logger)

    // 2. Batch migrate concepts
    batch := 1000
    for offset := 0; offset < 520000; offset += batch {
        concepts := fetchConceptBatch(pgDB, offset, batch)
        triples := convertToRDF(concepts) // Reuse existing RDF converter
        graphDB.LoadTurtleData(ctx, []byte(triples), "http://cardiofit.ai/bootstrap")
        logger.Info("Migrated %d concepts", offset+len(concepts))
    }

    // 3. Quick validation
    validateConceptCounts(pgDB, graphDB)
}
```

**Execution Time**: 2-4 hours for 520K concepts

**Note**: This bootstrap data will be replaced by Knowledge Factory kernel in Week 4

---

#### 1.3.2 AWS Infrastructure Setup (Week 2)

**Components**:
1. **S3 Buckets**:
   - `cardiofit-kb-sources`: Raw API downloads (SNOMED RF2, RxNorm RRF, LOINC CSV)
   - `cardiofit-kb-artifacts`: Transformed kernels (Turtle files)

2. **Lambda Functions** (Python 3.11):
   - `snomed-downloader`: Fetch from NHS TRUD API (1.2GB RF2 snapshot)
     - **Config**: 15-min timeout, 10GB memory, streaming S3 upload
     - **Risk Mitigation**: Uses `boto3.s3.upload_fileobj()` for chunked transfer (avoids OOM)
     - **Fallback**: If timeouts persist, migrate to ECS Fargate (no timeout limit)
   - `rxnorm-downloader`: Fetch from UMLS API (450MB RRF files)
     - **Config**: 10-min timeout, 3GB memory
   - `loinc-downloader`: Fetch from Regenstrief API (180MB CSV)
     - **Config**: 5-min timeout, 2GB memory
   - `github-dispatcher`: Trigger GitHub Actions workflow via `repository_dispatch`
     - **Security**: Does NOT pass secrets in payload (GitHub uses its own Secrets store)

3. **Step Functions Workflow**:
   ```
   Start → [SNOMED Download || RxNorm Download || LOINC Download] → GitHub Dispatch → End
   ```
   (Parallel execution: ~15 minutes total)

4. **Secrets Manager** (AWS-side only):
   - NHS TRUD API key (90-day rotation) - Used by Lambda
   - UMLS API key (annual renewal) - Used by Lambda
   - LOINC credentials (biannual rotation) - Used by Lambda
   - GitHub Personal Access Token (repo scope) - Used by github-dispatcher Lambda
   - **Note**: GitHub Actions uses GitHub Secrets (separate store) for ROBOT/transformation tasks
   - **Separation of Concerns**: Lambda downloads to S3 → GitHub Actions reads from S3 (no credential sharing)

5. **CloudWatch Events**:
   - Cron: `cron(0 2 1 * ? *)` (1st of month, 2 AM UTC)
   - Target: Step Functions state machine

**CloudFormation Templates**:
```
aws/cloudformation/
├── s3-buckets.yaml           # Storage infrastructure
├── lambda-functions.yaml      # Download functions
├── step-functions.yaml        # Orchestration workflow
└── secrets-manager.yaml       # Credential management
```

**Deployment**:
```bash
cd aws/scripts
./setup-infrastructure.sh  # Deploys all CloudFormation stacks
./test-lambda-functions.sh # Validates download functions
```

**Cost**: ~$5/month (S3 storage) + $1.50/month (Lambda invocations) + $0.40/month (Secrets Manager)

---

#### 1.3.3 Knowledge Factory Pipeline (Week 3)

**GitHub Repository**: `cardiofit-knowledge-factory` (separate repo from KB-7 service)

**GitHub Actions Workflow**: `.github/workflows/kb-factory.yml`

**7-Stage Pipeline**:

1. **Download Stage** (triggered by Lambda):
   - Pull raw sources from S3
   - Verify checksums and file integrity
   - Extract archives (ZIP/RRF/CSV)

2. **Transform Stage**:
   - **SNOMED**: SNOMED-OWL-Toolkit v4.0.6
     ```bash
     java -jar snomed-owl-toolkit.jar \
       -rf2-to-owl \
       -rf2-snapshot-archives SnomedCT_InternationalRF2_PRODUCTION_*.zip \
       -output snomed-ontology.owl
     ```
   - **RxNorm**: Custom Python converter (RRF → RDF/Turtle)
   - **LOINC**: ROBOT template-based conversion

3. **Merge Stage** (ROBOT v1.9.5):
   ```bash
   robot merge \
     --input snomed-ontology.owl \
     --input rxnorm-ontology.ttl \
     --input loinc-ontology.ttl \
     --collapse-import-closure false \
     --output kb7-merged.owl
   ```

4. **Reasoning Stage** (ELK via ROBOT):
   ```bash
   robot reason \
     --reasoner ELK \
     --input kb7-merged.owl \
     --output kb7-inferred.owl
   ```
   - **Risk Mitigation (Memory)**: Standard GitHub runners (~7GB RAM) may OOM with 8M+ triples
   - **Primary Solution**: Use GitHub Larger Runners (16GB or 32GB RAM)
     - Cost: ~$0.16/min (~$10-15/month for monthly runs)
     - Configuration: `runs-on: ubuntu-latest-16-core` in workflow YAML
   - **Alternative Options**:
     - Option B: Migrate reasoning stage to AWS CodeBuild (custom instance sizing)
     - Option C: Split reasoning into 3 parallel jobs (SNOMED, RxNorm, LOINC), then merge

5. **Validation Stage** (5 SPARQL quality gates - see 1.3.4)

6. **Package Stage**:
   - Convert OWL → Turtle for GraphDB
   - Generate metadata manifest (JSON)
   - Create version snapshot (YYYYMMDD)

7. **Upload Stage**:
   - Push kernel to S3 `cardiofit-kb-artifacts/YYYYMMDD/kb7-kernel.ttl`
   - Update metadata registry (PostgreSQL `kb7_snapshots` table)
   - Notify Slack/email on success/failure

**Docker Containers** (GitHub Actions runners):
```
docker/
├── Dockerfile.snomed-toolkit   # Java 17 + SNOMED-OWL-Toolkit
├── Dockerfile.robot             # Java 11 + ROBOT v1.9.5
└── Dockerfile.converters        # Python 3.11 + RxNorm/LOINC converters
```

**Execution Time**: 45-60 minutes end-to-end

**Artifacts**:
- `kb7-kernel.ttl`: ~2.5GB Turtle file with 8M+ triples
- `kb7-manifest.json`: Metadata (concept counts, provenance, checksums)
- `kb7-YYYYMMDD.tar.gz`: Versioned snapshot for rollback

---

#### 1.3.4 Quality Gates & Validation (Week 3)

**5 SPARQL Validation Queries** (zero errors required):

1. **Concept Count** (`validation/concept-count.sparql`):
   ```sparql
   SELECT (COUNT(DISTINCT ?concept) AS ?count) WHERE {
     ?concept a owl:Class .
     FILTER(STRSTARTS(STR(?concept), "http://snomed.info/id/"))
   }
   # Expected: >500,000 SNOMED concepts
   ```

2. **Orphaned Concepts** (`validation/orphaned-concepts.sparql`):
   ```sparql
   SELECT ?concept WHERE {
     ?concept a owl:Class .
     FILTER NOT EXISTS { ?concept rdfs:subClassOf ?parent }
     FILTER(?concept != owl:Thing)
   }
   # Expected: <10 orphaned concepts
   ```

3. **SNOMED Hierarchy Roots** (`validation/snomed-roots.sparql`):
   ```sparql
   SELECT (COUNT(?root) AS ?count) WHERE {
     ?root rdfs:subClassOf <http://snomed.info/id/138875005> .
   }
   # Expected: Exactly 1 root (SNOMED CT Concept)
   ```

4. **RxNorm Drug Count** (`validation/rxnorm-drugs.sparql`):
   ```sparql
   SELECT (COUNT(?drug) AS ?count) WHERE {
     ?drug a owl:Class .
     FILTER(STRSTARTS(STR(?drug), "http://purl.bioontology.org/ontology/RXNORM/"))
   }
   # Expected: >100,000 RxNorm concepts
   ```

5. **LOINC Code Count** (`validation/loinc-codes.sparql`):
   ```sparql
   SELECT (COUNT(?code) AS ?count) WHERE {
     ?code a owl:Class .
     FILTER(STRSTARTS(STR(?code), "http://loinc.org/rdf/"))
   }
   # Expected: >90,000 LOINC codes
   ```

**Validation Execution**:
```bash
cd knowledge-factory/validation
robot verify --input kb7-inferred.owl --queries *.sparql
```

**Failure Handling**:
- Any validation failure aborts pipeline
- Slack/email alert to KB-7 team
- Previous month's kernel remains active
- Automated retry after 6 hours (3 attempts max)

---

#### 1.3.5 Automation & Monitoring (Week 4)

**Monthly Automation**:
- CloudWatch Events trigger: 1st of month, 2 AM UTC
- Step Functions orchestrate parallel downloads
- Lambda dispatches GitHub Actions workflow
- Total runtime: ~60-75 minutes (download + transform + upload)

**Monitoring & Alerting**:

1. **CloudWatch Metrics & Alarms**:
   - Lambda invocation count, duration, errors
   - **Critical Alarm**: Lambda duration > 10 minutes → SNS alert (catches timeout risk early)
   - Step Functions execution status
   - S3 bucket size and object count
   - GitHub Actions workflow trigger success/failure

2. **GitHub Actions Monitoring**:
   - Workflow success/failure rate
   - Stage execution times
   - Artifact upload success

3. **Slack Notifications**:
   - ✅ Success: "KB-7 Knowledge Factory: January 2025 kernel published (523,451 concepts)"
   - ❌ Failure: "KB-7 Knowledge Factory: Validation failed - orphaned concepts > 10"
   - ⚠️ Warning: "KB-7 Knowledge Factory: Download retry attempt 2/3"

4. **Email Alerts** (SNS topics):
   - Critical failures (all 3 retries exhausted)
   - API credential expiration warnings (30 days before)
   - Storage quota warnings (S3 bucket >80% capacity)

**GraphDB Kernel Deployment**:
- Manual review of `kb7-manifest.json` after pipeline success
- Execute: `scripts/deploy-kernel.sh YYYYMMDD`
- Script:
  1. Download kernel from S3
  2. Load to GraphDB test repository
  3. Run validation queries
  4. If passed: Swap test→production repository
  5. Update PostgreSQL metadata registry
  6. Clear Redis cache
- Rollback: `scripts/rollback-kernel.sh YYYYMMDD-prev`

**Success Criteria**:
- ✅ Monthly pipeline executes automatically
- ✅ 95%+ success rate over 6 months
- ✅ Zero manual intervention for standard runs
- ✅ Kernel deployment: <5 minutes downtime
- ✅ Concept count increases by 1-2% monthly (industry growth rate)

---

## Phase 2: Hybrid Query Layer (Weeks 5-6)
**Goal**: Route queries to appropriate database based on query type

### 2.1 Activate Query Router (REUSE Existing)

**Current Code** (reuse):
- ✅ `query-router/cmd/main.go` - Router service
- ✅ `query-router/internal/router/router.go` - Routing logic
- ✅ `query-router/internal/postgres/client.go` - PostgreSQL client
- ✅ `query-router/internal/graphdb/client.go` - GraphDB client

**Enhancement** (update):
```go
// query-router/internal/router/router.go

func (r *HybridQueryRouter) RouteQuery(req *QueryRequest) (*QueryResponse, error) {
    switch req.QueryType {
    case "simple_lookup":
        // Fast path: PostgreSQL (2-3ms)
        return r.postgresClient.LookupConcept(req.Code, req.System)

    case "subsumption":
        // Semantic: GraphDB SPARQL (12-15ms)
        return r.graphDBClient.GetSubconcepts(req.Code)

    case "drug_interaction":
        // Semantic: GraphDB reasoning (20-30ms)
        return r.graphDBClient.GetDrugInteractions(req.Code)

    case "full_text_search":
        // Search: Elasticsearch (5-8ms)
        return r.elasticsearchClient.Search(req.Query)

    case "complex_relationship":
        // Hybrid: PostgreSQL + GraphDB
        pgData := r.postgresClient.GetConcept(req.Code)
        relationships := r.graphDBClient.GetRelationships(req.Code)
        return r.merge(pgData, relationships)

    default:
        return nil, ErrUnsupportedQueryType
    }
}
```

**Circuit Breaker** (add resilience):
```go
// query-router/internal/router/circuit_breaker.go

type CircuitBreaker struct {
    failureThreshold int
    resetTimeout     time.Duration
    state            string // "closed", "open", "half-open"
}

func (r *HybridQueryRouter) QueryWithCircuitBreaker(req *QueryRequest) (*QueryResponse, error) {
    if r.graphDBCircuit.IsOpen() {
        // Fallback: Use PostgreSQL only
        logger.Warn("GraphDB circuit open, using PostgreSQL fallback")
        return r.postgresClient.Lookup(req.Code)
    }

    resp, err := r.RouteQuery(req)
    if err != nil {
        r.graphDBCircuit.RecordFailure()
        return r.postgresClient.Lookup(req.Code) // Fallback
    }

    r.graphDBCircuit.RecordSuccess()
    return resp, nil
}
```

### 2.2 Add SPARQL Endpoints to REST API

**File**: `internal/api/semantic_handlers.go` (NEW)

```go
package api

import (
    "github.com/gin-gonic/gin"
    "kb-7-terminology/internal/semantic"
)

// GET /v1/concepts/:system/:code/subconcepts
// Example: GET /v1/concepts/SNOMED/387517004/subconcepts
func (s *Server) HandleSubconceptQuery(c *gin.Context) {
    system := c.Param("system")
    code := c.Param("code")

    // SPARQL query for subsumption
    query := fmt.Sprintf(`
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        SELECT ?subconcept ?label WHERE {
            ?subconcept rdfs:subClassOf <%s> ;
                        rdfs:label ?label .
        } LIMIT 100
    `, conceptURI(system, code))

    results, err := s.graphDB.ExecuteSPARQL(ctx, &semantic.SPARQLQuery{Query: query})
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, results)
}

// POST /v1/interactions
// Body: {"medications": ["387517004", "372764001"]}
func (s *Server) HandleDrugInteractions(c *gin.Context) {
    var req struct {
        Medications []string `json:"medications"`
    }
    c.BindJSON(&req)

    // SPARQL query for drug interactions
    query := buildInteractionQuery(req.Medications)
    results, err := s.graphDB.ExecuteSPARQL(ctx, &semantic.SPARQLQuery{Query: query})

    c.JSON(200, gin.H{
        "interactions": parseInteractions(results),
        "query_time_ms": results.Meta["executionTime"],
    })
}
```

**Update Main Server** (`cmd/server/main.go`):
```go
// Add GraphDB client
graphDBClient := semantic.NewGraphDBClient(
    cfg.GraphDBEndpoint,
    "kb7-terminology",
    logger,
)

// Inject into API server
apiServer := api.NewServer(cfg, terminologyService, graphDBClient, logger, metricsCollector)

// Add new routes
router.GET("/v1/concepts/:system/:code/subconcepts", apiServer.HandleSubconceptQuery)
router.POST("/v1/interactions", apiServer.HandleDrugInteractions)
router.GET("/v1/concepts/:system/:code/relationships", apiServer.HandleHybridQuery)
```

### 2.3 Performance Benchmarking

**Benchmark Script**: `scripts/benchmark/query-performance.sh`

```bash
#!/bin/bash

echo "=== Query Performance Comparison ==="

# 1. Simple Lookup (PostgreSQL)
echo "PostgreSQL Lookup:"
time curl "http://localhost:8092/v1/concepts/SNOMED/387517004"

# 2. Subsumption (GraphDB SPARQL)
echo "GraphDB Subsumption:"
time curl "http://localhost:8092/v1/concepts/SNOMED/387517004/subconcepts"

# 3. Drug Interactions (GraphDB Reasoning)
echo "GraphDB Interactions:"
time curl -X POST "http://localhost:8092/v1/interactions" \
  -d '{"medications": ["387517004", "372764001"]}'

# 4. Hybrid Query
echo "Hybrid Query:"
time curl "http://localhost:8092/v1/concepts/SNOMED/387517004/relationships"

# 5. Load Test (1000 concurrent)
echo "Load Test:"
ab -n 1000 -c 100 "http://localhost:8092/v1/concepts/SNOMED/387517004"
```

**Target Metrics**:
| Query Type | Target Latency | Database |
|------------|----------------|----------|
| Simple lookup | <5ms | PostgreSQL |
| Subsumption | <50ms | GraphDB SPARQL |
| Drug interactions | <100ms | GraphDB reasoning |
| Full-text search | <10ms | Elasticsearch |
| Hybrid query | <80ms | PostgreSQL + GraphDB |

**Success Criteria**:
- ✅ All new endpoints respond successfully
- ✅ SPARQL queries return correct semantic relationships
- ✅ Query latency meets targets (p95)
- ✅ Circuit breaker activates on GraphDB failures
- ✅ Backward compatibility maintained (existing endpoints unchanged)

---

## Phase 3: PostgreSQL Schema Refactoring (Weeks 7-9)
**Goal**: Transform PostgreSQL from concept storage to metadata registry and integrate kernel deployment workflows

### 3.1 Create Metadata Schema

**Migration**: `migrations/006_create_snapshot_registry.up.sql`

```sql
-- KB-7 Kernel Snapshot Registry
CREATE TABLE kb7_snapshots (
    snapshot_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    version VARCHAR(50) NOT NULL UNIQUE,  -- e.g., "v2.3.0"
    kernel_uri VARCHAR(500) NOT NULL,     -- http://cardiofit.ai/kernels/v2.3.0
    graphdb_graph_uri VARCHAR(500),       -- Named graph URI in GraphDB

    -- Content statistics
    concept_count INTEGER NOT NULL,
    triple_count BIGINT NOT NULL,
    snomed_version VARCHAR(50),
    rxnorm_version VARCHAR(50),
    loinc_version VARCHAR(50),

    -- Provenance
    source_checksum VARCHAR(64),          -- SHA-256 of source files
    built_by VARCHAR(100),                -- GitHub user or system
    build_timestamp TIMESTAMP NOT NULL,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'building',  -- building, active, deprecated
    activated_at TIMESTAMP,
    deprecated_at TIMESTAMP,

    -- Metadata
    release_notes TEXT,
    validation_status JSONB,              -- Quality gate results

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_kb7_snapshots_version ON kb7_snapshots(version);
CREATE INDEX idx_kb7_snapshots_status ON kb7_snapshots(status);
CREATE INDEX idx_kb7_snapshots_activated ON kb7_snapshots(activated_at DESC);

-- Snapshot change events (for CDC)
CREATE TABLE kb7_snapshot_events (
    event_id BIGSERIAL PRIMARY KEY,
    snapshot_id UUID REFERENCES kb7_snapshots(snapshot_id),
    event_type VARCHAR(50) NOT NULL,  -- created, activated, deprecated
    event_data JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_kb7_snapshot_events_created ON kb7_snapshot_events(created_at DESC);
```

### 3.2 Archive Existing Concept Data

**DO NOT DELETE** - Mark as archived for historical queries

**Migration**: `migrations/007_archive_concept_tables.up.sql`

```sql
-- Add archived flag to existing tables
ALTER TABLE terminology_concepts ADD COLUMN archived_at TIMESTAMP;
ALTER TABLE terminology_systems ADD COLUMN archived_at TIMESTAMP;

-- Create read-only view for backward compatibility
CREATE VIEW active_concepts AS
SELECT * FROM terminology_concepts
WHERE archived_at IS NULL;

-- Comment indicating archival strategy
COMMENT ON TABLE terminology_concepts IS
'ARCHIVED: Concepts now stored in GraphDB. This table maintained for historical queries and rollback.';
```

**Export Script**: `scripts/archive/export-postgresql-backup.sh`

```bash
#!/bin/bash
set -e

BACKUP_DIR="./backups/postgresql-archive-$(date +%Y%m%d)"
mkdir -p "$BACKUP_DIR"

echo "Exporting PostgreSQL concept data..."

# Export schema
pg_dump -U kb7_user -d kb7_terminology --schema-only > "$BACKUP_DIR/schema.sql"

# Export concept data
pg_dump -U kb7_user -d kb7_terminology \
  --table=terminology_concepts \
  --table=terminology_systems \
  --table=concept_mappings \
  --data-only > "$BACKUP_DIR/concepts.sql"

# Export as CSV for easy inspection
psql -U kb7_user -d kb7_terminology -c "\COPY terminology_concepts TO '$BACKUP_DIR/concepts.csv' CSV HEADER"

# Compress
tar -czf "$BACKUP_DIR.tar.gz" "$BACKUP_DIR"

echo "✅ Backup created: $BACKUP_DIR.tar.gz"
```

### 3.3 Update ETL Pipeline to Insert Metadata

**File**: `internal/etl/snapshot_manager.go` (NEW)

```go
package etl

import (
    "context"
    "crypto/sha256"
    "database/sql"
)

type SnapshotManager struct {
    db     *sql.DB
    logger *zap.Logger
}

func (s *SnapshotManager) CreateSnapshot(ctx context.Context, buildInfo *BuildInfo) (string, error) {
    // 1. Query GraphDB for statistics
    tripleCount := s.queryGraphDBTripleCount(ctx)
    conceptCount := s.queryGraphDBConceptCount(ctx)

    // 2. Calculate source checksum
    checksum := s.calculateSourceChecksum(buildInfo.SourceFiles)

    // 3. Insert snapshot record
    query := `
        INSERT INTO kb7_snapshots (
            version, kernel_uri, graphdb_graph_uri,
            concept_count, triple_count,
            snomed_version, rxnorm_version, loinc_version,
            source_checksum, built_by, build_timestamp, status
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        RETURNING snapshot_id
    `

    var snapshotID string
    err := s.db.QueryRowContext(ctx, query,
        buildInfo.Version,
        fmt.Sprintf("http://cardiofit.ai/kernels/%s", buildInfo.Version),
        fmt.Sprintf("http://cardiofit.ai/kernels/%s", buildInfo.Version),
        conceptCount,
        tripleCount,
        buildInfo.SNOMEDVersion,
        buildInfo.RxNormVersion,
        buildInfo.LOINCVersion,
        checksum,
        buildInfo.BuiltBy,
        time.Now(),
        "building",
    ).Scan(&snapshotID)

    return snapshotID, err
}

func (s *SnapshotManager) ActivateSnapshot(ctx context.Context, snapshotID string) error {
    // 1. Deprecate current active snapshot
    _, err := s.db.ExecContext(ctx, `
        UPDATE kb7_snapshots
        SET status = 'deprecated', deprecated_at = NOW()
        WHERE status = 'active'
    `)

    // 2. Activate new snapshot
    _, err = s.db.ExecContext(ctx, `
        UPDATE kb7_snapshots
        SET status = 'active', activated_at = NOW()
        WHERE snapshot_id = $1
    `, snapshotID)

    // 3. Create activation event (triggers CDC)
    _, err = s.db.ExecContext(ctx, `
        INSERT INTO kb7_snapshot_events (snapshot_id, event_type, event_data)
        VALUES ($1, 'activated', $2)
    `, snapshotID, json.Marshal(map[string]interface{}{
        "activated_at": time.Now(),
        "activated_by": "etl-pipeline",
    }))

    return err
}
```

**Update ETL Main** (`cmd/etl/main.go`):
```go
// After successful GraphDB load
snapshotMgr := etl.NewSnapshotManager(db, logger)

// Create snapshot record
snapshotID, err := snapshotMgr.CreateSnapshot(ctx, &etl.BuildInfo{
    Version:       "v1.0.0",
    SNOMEDVersion: "20240331",
    RxNormVersion: "20240304",
    LOINCVersion:  "2.77",
    SourceFiles:   []string{"./data/snomed.zip", "./data/rxnorm.zip", "./data/loinc.zip"},
    BuiltBy:       "manual-etl",
})

// Activate snapshot (triggers CDC event)
err = snapshotMgr.ActivateSnapshot(ctx, snapshotID)
```

### 3.4 Rollback Procedure

**Script**: `scripts/rollback/restore-postgresql-concepts.sh`

```bash
#!/bin/bash
set -e

BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup-file.tar.gz>"
    exit 1
fi

echo "=== Rollback: Restoring PostgreSQL Concepts ==="

# 1. Extract backup
tar -xzf "$BACKUP_FILE"
BACKUP_DIR="${BACKUP_FILE%.tar.gz}"

# 2. Restore concept data
psql -U kb7_user -d kb7_terminology < "$BACKUP_DIR/concepts.sql"

# 3. Mark as active (remove archived flag)
psql -U kb7_user -d kb7_terminology -c "UPDATE terminology_concepts SET archived_at = NULL;"

# 4. Verify restoration
COUNT=$(psql -U kb7_user -d kb7_terminology -t -c "SELECT COUNT(*) FROM terminology_concepts WHERE archived_at IS NULL;")
echo "✅ Restored $COUNT concepts to PostgreSQL"

# 5. Update API to use PostgreSQL (disable GraphDB routes)
echo "⚠️  Manual step: Update API configuration to disable GraphDB routes"
```

**Success Criteria**:
- ✅ `kb7_snapshots` table created with metadata schema
- ✅ Existing concept data exported and archived (not deleted)
- ✅ ETL pipeline creates snapshot records after GraphDB load
- ✅ Sample snapshot activation triggers event
- ✅ Rollback procedure tested and documented

---

## Phase 5: CDC Streaming (Weeks 10-13)
**Goal**: Enable real-time downstream updates via Kafka with comprehensive testing

### 5.1 Debezium PostgreSQL Connector

**File**: `docker-compose.cdc.yml`

```yaml
version: '3.8'

services:
  # Zookeeper for Kafka
  zookeeper:
    image: confluentinc/cp-zookeeper:7.5.0
    container_name: kb7-zookeeper
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    ports:
      - "2181:2181"

  # Kafka Broker
  kafka:
    image: confluentinc/cp-kafka:7.5.0
    container_name: kb7-kafka
    depends_on:
      - zookeeper
    ports:
      - "9092:9092"
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1

  # Kafka Connect with Debezium
  kafka-connect:
    image: debezium/connect:2.5
    container_name: kb7-kafka-connect
    depends_on:
      - kafka
      - postgres-terminology
    ports:
      - "8083:8083"
    environment:
      BOOTSTRAP_SERVERS: kafka:9092
      GROUP_ID: kb7-connect-cluster
      CONFIG_STORAGE_TOPIC: kb7_connect_configs
      OFFSET_STORAGE_TOPIC: kb7_connect_offsets
      STATUS_STORAGE_TOPIC: kb7_connect_statuses

  # Schema Registry
  schema-registry:
    image: confluentinc/cp-schema-registry:7.5.0
    container_name: kb7-schema-registry
    depends_on:
      - kafka
    ports:
      - "8081:8081"
    environment:
      SCHEMA_REGISTRY_HOST_NAME: schema-registry
      SCHEMA_REGISTRY_KAFKASTORE_BOOTSTRAP_SERVERS: kafka:9092
```

### 5.2 Debezium Connector Configuration

**File**: `scripts/cdc/register-connector.sh`

```bash
#!/bin/bash
set -e

echo "=== Registering Debezium PostgreSQL Connector ==="

curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "kb7-snapshot-connector",
    "config": {
      "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
      "database.hostname": "postgres-terminology",
      "database.port": "5432",
      "database.user": "kb7_user",
      "database.password": "kb7_secure_password",
      "database.dbname": "kb7_terminology",
      "database.server.name": "kb7",
      "table.include.list": "public.kb7_snapshots,public.kb7_snapshot_events",
      "plugin.name": "pgoutput",
      "publication.autocreate.mode": "filtered",
      "topic.prefix": "kb7",
      "transforms": "route",
      "transforms.route.type": "org.apache.kafka.connect.transforms.RegexRouter",
      "transforms.route.regex": "([^.]+)\\.([^.]+)\\.([^.]+)",
      "transforms.route.replacement": "kb7.$3"
    }
  }'

echo "✅ Connector registered successfully"

# Verify connector status
curl http://localhost:8083/connectors/kb7-snapshot-connector/status | jq
```

### 5.3 Event Consumer Example

**File**: `examples/cdc-consumer/consumer.go`

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/segmentio/kafka-go"
)

type SnapshotEvent struct {
    Op      string                 `json:"op"`      // c=create, u=update
    After   map[string]interface{} `json:"after"`   // New snapshot data
    Before  map[string]interface{} `json:"before"`  // Old snapshot data (for updates)
    Source  map[string]interface{} `json:"source"`  // Metadata
}

func main() {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers: []string{"localhost:9092"},
        Topic:   "kb7.kb7_snapshot_events",
        GroupID: "clinical-decision-support",
    })
    defer reader.Close()

    fmt.Println("Listening for KB-7 snapshot events...")

    for {
        msg, err := reader.ReadMessage(context.Background())
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }

        var event SnapshotEvent
        json.Unmarshal(msg.Value, &event)

        if event.After["event_type"] == "activated" {
            version := event.After["version"].(string)
            fmt.Printf("🚀 New kernel activated: %s\n", version)

            // Trigger downstream action
            updateClinicalDecisionSupport(version)
        }
    }
}

func updateClinicalDecisionSupport(version string) {
    // Implementation: Invalidate cache, reload rules, etc.
    fmt.Printf("Updating clinical decision support to kernel %s...\n", version)
}
```

### 5.4 Monitoring & Alerting

**File**: `monitoring/cdc-dashboard.json` (Grafana)

```json
{
  "dashboard": {
    "title": "KB-7 CDC Monitoring",
    "panels": [
      {
        "title": "Kafka Lag",
        "targets": [{
          "expr": "kafka_consumer_lag{topic=\"kb7.kb7_snapshot_events\"}",
          "legendFormat": "Consumer Lag"
        }]
      },
      {
        "title": "Event Processing Latency",
        "targets": [{
          "expr": "histogram_quantile(0.95, rate(cdc_event_latency_seconds_bucket[5m]))",
          "legendFormat": "p95 Latency"
        }]
      },
      {
        "title": "Connector Status",
        "targets": [{
          "expr": "debezium_connector_status{connector=\"kb7-snapshot-connector\"}",
          "legendFormat": "Status"
        }]
      }
    ]
  }
}
```

**Prometheus Alerts** (`monitoring/alerts/cdc.yml`):

```yaml
groups:
  - name: kb7_cdc
    interval: 30s
    rules:
      - alert: HighKafkaLag
        expr: kafka_consumer_lag{topic="kb7.kb7_snapshot_events"} > 100
        for: 5m
        annotations:
          summary: "High Kafka lag for KB-7 events"
          description: "Consumer lag is {{ $value }} messages"

      - alert: DebeziumConnectorDown
        expr: debezium_connector_status{connector="kb7-snapshot-connector"} != 1
        for: 2m
        annotations:
          summary: "Debezium connector is down"
          description: "KB-7 CDC connector is not running"

      - alert: SlowEventProcessing
        expr: histogram_quantile(0.95, rate(cdc_event_latency_seconds_bucket[5m])) > 0.8
        for: 10m
        annotations:
          summary: "Slow CDC event processing"
          description: "p95 latency is {{ $value }}s (target: <800ms)"
```

**Success Criteria**:
- ✅ Debezium connector successfully registered
- ✅ PostgreSQL WAL configured for logical replication
- ✅ Kafka topics created (`kb7.kb7_snapshots`, `kb7.kb7_snapshot_events`)
- ✅ Sample event consumed by downstream service
- ✅ End-to-end latency <800ms (snapshot insert → Kafka delivery)
- ✅ Monitoring dashboard shows connector health
- ✅ Alerts fire on lag/failures

---

## Phase 6: Cleanup & Optimization (Weeks 14-16)
**Goal**: Remove unused code, consolidate, optimize, and conduct end-to-end validation

### 6.1 Code Removal Plan

**Remove** (unused/duplicate code):

```bash
# 1. Old PostgreSQL concept writers (replaced by GraphDB)
rm internal/etl/postgres_concept_writer.go

# 2. Duplicate GraphDB clients
# Keep: internal/semantic/graphdb_client.go (primary)
# Remove: query-router/internal/graphdb/client.go (use primary instead)
rm query-router/internal/graphdb/client.go

# 3. Unused Elasticsearch API code (if not exposing full-text search)
# If decision is to NOT expose Elasticsearch directly:
rm internal/api/elasticsearch_handlers.go

# 4. Old migration scripts (pre-GraphDB)
mv migrations/001-005_*.sql migrations/archive/

# 5. Test files for removed code
find . -name "*_test.go" -exec grep -l "postgres_concept_writer\|old_etl" {} \; | xargs rm
```

**Consolidate**:

```go
// query-router/internal/graphdb/client.go → DELETE
// All GraphDB operations should use: internal/semantic/graphdb_client.go

// Update query-router to use semantic package:
import "kb-7-terminology/internal/semantic"

func NewHybridQueryRouter(...) *HybridQueryRouter {
    return &HybridQueryRouter{
        graphDBClient: semantic.NewGraphDBClient(graphDBURL, "kb7-terminology", logger),
        // ...
    }
}
```

### 6.2 Performance Optimization

**GraphDB Query Caching** (`internal/api/graphdb_cache.go`):

```go
package api

import (
    "context"
    "crypto/sha256"
    "fmt"
    "kb-7-terminology/internal/cache"
    "kb-7-terminology/internal/semantic"
    "time"
)

type CachedGraphDBClient struct {
    client *semantic.GraphDBClient
    cache  *cache.RedisClient
    ttl    time.Duration
}

func (c *CachedGraphDBClient) ExecuteSPARQL(ctx context.Context, query *semantic.SPARQLQuery) (*semantic.SPARQLResults, error) {
    // Generate cache key from query
    key := fmt.Sprintf("sparql:%x", sha256.Sum256([]byte(query.Query)))

    // Try cache first
    var cached semantic.SPARQLResults
    err := c.cache.Get(ctx, key, &cached)
    if err == nil {
        return &cached, nil  // Cache hit
    }

    // Cache miss: Execute query
    results, err := c.client.ExecuteSPARQL(ctx, query)
    if err != nil {
        return nil, err
    }

    // Store in cache
    c.cache.Set(ctx, key, results, c.ttl)

    return results, nil
}
```

**Connection Pooling** (`internal/semantic/graphdb_client.go`):

```go
// Update GraphDBClient to use connection pool
type GraphDBClient struct {
    baseURL    string
    httpClient *http.Client
    pool       *ConnectionPool  // Add connection pooling
}

type ConnectionPool struct {
    maxConnections int
    activeConns    chan *http.Client
}

func NewGraphDBClient(baseURL string, repo string, logger *logrus.Logger) *GraphDBClient {
    pool := &ConnectionPool{
        maxConnections: 50,
        activeConns:    make(chan *http.Client, 50),
    }

    // Pre-warm connection pool
    for i := 0; i < 50; i++ {
        pool.activeConns <- &http.Client{
            Timeout: 60 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 50,
                IdleConnTimeout:     90 * time.Second,
            },
        }
    }

    return &GraphDBClient{
        baseURL: baseURL,
        pool:    pool,
        logger:  logger,
    }
}
```

**Index Optimization** (PostgreSQL):

```sql
-- migrations/008_optimize_snapshot_indexes.up.sql

-- Partial index for active snapshots only
CREATE INDEX idx_kb7_snapshots_active
ON kb7_snapshots(version, activated_at)
WHERE status = 'active';

-- Index for CDC event lookups
CREATE INDEX idx_kb7_snapshot_events_snapshot_type
ON kb7_snapshot_events(snapshot_id, event_type, created_at DESC);

-- Vacuum and analyze
VACUUM ANALYZE kb7_snapshots;
VACUUM ANALYZE kb7_snapshot_events;
```

### 6.3 Documentation Update

**Files to Update**:

1. **README.md** - Update architecture diagram and quick start
2. **DATABASE.md** - Document new schema (snapshots, not concepts)
3. **API.md** (NEW) - Document SPARQL endpoints
4. **DEPLOYMENT.md** - Update with CDC setup steps
5. **CONTRIBUTING.md** - Knowledge Factory contribution guide

**New Architecture Diagram** (for README.md):

```
┌─────────────────────────────────────────────────────────┐
│              KB-7 Terminology Service                    │
│           GraphDB-Centric Architecture                   │
└─────────────────────────────────────────────────────────┘

External Sources (NCTS, NIH, LOINC.org)
         ↓
  Knowledge Factory Pipeline
   │
   ├─ SNOMED-OWL-Toolkit (RF2 → OWL)
   ├─ ROBOT (Merge + Reason + Validate)
   └─ Python Converters (RxNorm/LOINC → RDF)
         ↓
   ┌──────────────┐
   │   GraphDB    │  ← PRIMARY: 2.5M triples
   │  Port 7200   │     Semantic reasoning
   └──────────────┘     SPARQL queries
         ↓
   Extract Metadata
         ↓
   ┌──────────────┐
   │ PostgreSQL   │  ← METADATA: Version tracking
   │  Port 5433   │     kb7_snapshots table
   └──────────────┘     CDC triggers
         ↓
   ┌──────────────┐
   │ Debezium CDC │  → Kafka Topics
   └──────────────┘
         ↓
   Downstream Services (Clinical Decision Support)

Query Layer:
┌─────────────────────────────────────────────┐
│          Hybrid Query Router                │
│  - Simple lookup    → PostgreSQL (cache)    │
│  - Semantic query   → GraphDB (SPARQL)      │
│  - Full-text search → Elasticsearch         │
│  - Complex query    → Hybrid (both)         │
└─────────────────────────────────────────────┘
```

### 6.4 Final Validation

**Script**: `scripts/validation/final-architecture-test.sh`

```bash
#!/bin/bash
set -e

echo "=== Final Architecture Validation ==="

# 1. GraphDB is primary storage
echo "Test 1: GraphDB Primary Storage"
GRAPHDB_TRIPLES=$(curl -s -X POST http://localhost:7200/repositories/kb7-terminology \
  --data-urlencode "query=SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }" | \
  jq -r '.results.bindings[0].count.value')

if [ "$GRAPHDB_TRIPLES" -lt 2000000 ]; then
    echo "❌ GraphDB should have >2M triples, found: $GRAPHDB_TRIPLES"
    exit 1
fi
echo "✅ GraphDB has $GRAPHDB_TRIPLES triples"

# 2. PostgreSQL is metadata only
echo "Test 2: PostgreSQL Metadata Registry"
PG_SNAPSHOTS=$(psql -U kb7_user -d kb7_terminology -t -c "SELECT COUNT(*) FROM kb7_snapshots;")

if [ "$PG_SNAPSHOTS" -lt 1 ]; then
    echo "❌ PostgreSQL should have snapshot records"
    exit 1
fi
echo "✅ PostgreSQL has $PG_SNAPSHOTS snapshots"

# 3. SPARQL endpoints work
echo "Test 3: SPARQL Endpoints"
SPARQL_RESPONSE=$(curl -s http://localhost:8092/v1/concepts/SNOMED/387517004/subconcepts)

if [ -z "$SPARQL_RESPONSE" ]; then
    echo "❌ SPARQL endpoint not responding"
    exit 1
fi
echo "✅ SPARQL endpoints operational"

# 4. CDC streaming active
echo "Test 4: CDC Streaming"
CONNECTOR_STATUS=$(curl -s http://localhost:8083/connectors/kb7-snapshot-connector/status | jq -r '.connector.state')

if [ "$CONNECTOR_STATUS" != "RUNNING" ]; then
    echo "❌ CDC connector not running: $CONNECTOR_STATUS"
    exit 1
fi
echo "✅ CDC connector running"

# 5. Query performance
echo "Test 5: Query Performance"
START_TIME=$(date +%s%N)
curl -s http://localhost:8092/v1/concepts/SNOMED/387517004 > /dev/null
END_TIME=$(date +%s%N)
LATENCY=$(( (END_TIME - START_TIME) / 1000000 ))

if [ "$LATENCY" -gt 50 ]; then
    echo "⚠️  Query latency high: ${LATENCY}ms (target: <50ms)"
else
    echo "✅ Query latency: ${LATENCY}ms"
fi

# 6. Knowledge Factory executable
echo "Test 6: Knowledge Factory"
if [ ! -f "scripts/knowledge-factory/build-kernel.sh" ]; then
    echo "❌ Knowledge Factory script missing"
    exit 1
fi
echo "✅ Knowledge Factory scripts present"

echo ""
echo "✅ All architecture validations passed!"
echo "KB-7 transformation complete."
```

**Success Criteria**:
- ✅ All unused code removed (confirmed via grep/file checks)
- ✅ GraphDB client consolidated to single package
- ✅ Query performance meets targets (p95 <50ms)
- ✅ Connection pooling reduces latency by 20%+
- ✅ Redis caching reduces GraphDB queries by 60%+
- ✅ Documentation updated and accurate
- ✅ Final validation script passes 100%

---

## Rollback Strategy

### Per-Phase Rollback

| Phase | Trigger | Rollback Procedure | Time | Data Loss Risk |
|-------|---------|-------------------|------|----------------|
| **Phase 1** | GraphDB/Knowledge Factory fails | Use PostgreSQL bootstrap + manual updates | 5 min | None |
| **Phase 2** | API errors spike | Disable GraphDB routes, PostgreSQL only | 2 min | None |
| **Phase 3** | Metadata issues | Restore from PostgreSQL backup | 30 min | None (archived) |
| **Phase 5** | CDC lag >1s | Disable Debezium, manual updates | 5 min | None (async) |
| **Phase 6** | Performance regression | Revert optimization commits | 15 min | None |

### Emergency Rollback (Complete)

**Scenario**: Critical failure requiring full rollback to PostgreSQL-only

**Script**: `scripts/rollback/emergency-rollback.sh`

```bash
#!/bin/bash
set -e

echo "=== EMERGENCY ROLLBACK: Reverting to PostgreSQL-Only Architecture ==="

# 1. Disable GraphDB routes
kubectl set env deployment/kb7-api GRAPHDB_ENABLED=false

# 2. Disable CDC
curl -X DELETE http://localhost:8083/connectors/kb7-snapshot-connector

# 3. Restore PostgreSQL concepts from archive
./scripts/rollback/restore-postgresql-concepts.sh ./backups/postgresql-archive-latest.tar.gz

# 4. Update API configuration
kubectl set env deployment/kb7-api PRIMARY_DB=postgresql

# 5. Verify service health
kubectl rollout status deployment/kb7-api

echo "✅ Rollback complete. System running on PostgreSQL only."
```

---

## Success Metrics

### Overall Project Success

| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| **Primary Storage** | PostgreSQL | GraphDB | `SELECT source FROM system_info` |
| **Semantic Queries** | 0% | 100% | SPARQL endpoint requests/day |
| **Query Latency (p95)** | 15ms | <50ms | Prometheus histogram |
| **Update Automation** | 0% (manual) | 100% (automated) | GitHub Actions success rate |
| **CDC Latency** | N/A | <800ms | Kafka lag monitoring |
| **Code Reduction** | Baseline | -20% | `cloc` comparison |
| **Test Coverage** | 65% | 80% | `go test -cover` |

### Per-Phase Success Criteria Summary

| Phase | Success Criteria |
|-------|-----------------|
| **1. GraphDB Foundation** | ✅ 520K concepts in GraphDB<br>✅ SPARQL queries functional<br>✅ PostgreSQL == GraphDB count |
| **2. Hybrid Query Layer** | ✅ All endpoints respond<br>✅ Semantic queries work<br>✅ Latency targets met |
| **3. PostgreSQL Refactor** | ✅ Metadata schema created<br>✅ Concepts archived (not deleted)<br>✅ Rollback tested |
| **4. Knowledge Factory** | ✅ Automated build succeeds<br>✅ All quality gates pass<br>✅ GitHub Actions runs |
| **5. CDC Streaming** | ✅ Debezium connector running<br>✅ Events consumed<br>✅ Latency <800ms |
| **6. Cleanup & Optimize** | ✅ Unused code removed<br>✅ Performance improved<br>✅ Docs updated |

---

## Resource Requirements

### Infrastructure

| Component | Current | Required | Upgrade |
|-----------|---------|----------|---------|
| **GraphDB Heap** | 4GB | 8GB | 2x increase for reasoning workload |
| **PostgreSQL** | 5GB | 10GB | 2x increase for snapshot metadata |
| **Kafka Cluster** | None | 3 brokers (or Confluent Cloud) | New for CDC streaming |
| **Redis** | 1GB | 2GB | 2x increase for semantic cache |
| **GitHub Actions Runner** | None | Self-hosted or GitHub Cloud | New for Knowledge Factory |

### Cloud Infrastructure (AWS)

| Component | Purpose | Estimated Cost |
|-----------|---------|----------------|
| **S3 Buckets (2)** | Source files + transformed kernels | $5/month |
| **Lambda Functions (4)** | SNOMED/RxNorm/LOINC downloads + dispatcher | $1.50/month |
| **Step Functions** | Orchestrate parallel downloads | $0.50/month |
| **Secrets Manager** | API credentials with rotation | $0.40/month |
| **CloudWatch** | Scheduling, logging, metrics | $0.50/month |

**Total AWS Cost**: ~$8.90/month (see Cost Analysis section for details)

### Team & Timeline

| Resource | Allocation | Duration |
|----------|-----------|----------|
| **Senior Backend Engineer (Go)** | 1 FTE | 16 weeks |
| **DevOps Engineer** | 0.75 FTE | 16 weeks (AWS setup in Phase 1) |
| **Clinical Informaticist** | 0.25 FTE | Knowledge Factory validation |
| **QA Engineer** | 0.5 FTE | Weeks 8-16 (testing) |

**Total Effort**: ~2.5 FTE × 16 weeks = 40 person-weeks

### External Dependencies

| Dependency | Required | Source | Cost |
|------------|----------|--------|------|
| **SNOMED-OWL-Toolkit** | v4.0.6+ | GitHub (open source) | Free |
| **ROBOT Tool** | v1.9.5+ | GitHub (open source) | Free |
| **NCTS API Access** | Yes | Australian Digital Health Agency | Free (registration) |
| **NIH RxNorm Access** | Yes | NLM UTS Account | Free (registration) |
| **LOINC Access** | Yes | LOINC.org Account | Free (terms acceptance) |
| **Debezium** | v2.5+ | Docker Hub | Free |
| **Kafka** | 3 brokers | Self-hosted or Confluent Cloud | Variable |

---

## Quick Wins (Start Immediately)

These can be completed independently while planning phases 1-6:

### 1. Create GraphDB Repository (30 minutes)
```bash
# Visit http://localhost:7200
# Setup → Repositories → Create new repository
# Repository ID: kb7-terminology
# Ruleset: OWL2-RL (Optimized)
```

### 2. Test GraphDB Connection (5 minutes)
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology
go run test-graphdb-connection.go
```

### 3. Load Sample Concepts to GraphDB (1 hour)
```bash
# Create sample RDF file
cat > sample-concepts.ttl << 'EOF'
@prefix kb7: <http://cardiofit.ai/kb7/ontology#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .

<http://snomed.info/id/387517004> a kb7:ClinicalConcept ;
    rdfs:label "Paracetamol" ;
    kb7:code "387517004" ;
    kb7:system "SNOMED-CT" .
EOF

# Load to GraphDB
curl -X POST http://localhost:7200/repositories/kb7-terminology/statements \
  -H "Content-Type: text/turtle" \
  --data-binary @sample-concepts.ttl
```

### 4. Execute First SPARQL Query (30 minutes)
```bash
# Query GraphDB
curl -X POST http://localhost:7200/repositories/kb7-terminology \
  --data-urlencode "query=SELECT ?s ?label WHERE { ?s rdfs:label ?label } LIMIT 10"
```

### 5. Update Docker Compose (15 minutes)
```bash
# Start all services
docker-compose -f docker-compose.hybrid.yml up -d

# Verify
docker ps | grep kb7
```

---

## Next Steps

1. **Review & Approve**: Stakeholder review of this transformation plan
2. **Allocate Resources**: Assign team members and infrastructure
3. **Phase 1 Kickoff**: Begin GraphDB foundation implementation
4. **Weekly Checkpoints**: Status updates every Friday
5. **Phase Gates**: Sign-off required before proceeding to next phase

**Questions?** Contact: kb7-architecture@cardiofit.ai

---

## Appendix

### A. Current vs Proposed Architecture Comparison

**Current (PostgreSQL-Centric)**:
```
Pros:
✅ Simple architecture
✅ Fast simple lookups (3ms)
✅ Well-understood PostgreSQL

Cons:
❌ No semantic reasoning
❌ Slow graph queries (>800ms)
❌ Manual updates only
❌ Cannot do subsumption
❌ Cannot detect drug interactions
```

**Proposed (GraphDB-Centric)**:
```
Pros:
✅ Semantic reasoning (SPARQL)
✅ Fast graph queries (<50ms)
✅ Automated updates (Knowledge Factory)
✅ Real-time CDC streaming
✅ OWL inference support

Cons:
⚠️ More complex architecture
⚠️ Higher infrastructure cost
⚠️ Team learning curve
⚠️ 14-week transformation timeline
```

### B. Decision Matrix

| Criterion | Weight | PostgreSQL-Only | GraphDB-Centric | Winner |
|-----------|--------|-----------------|-----------------|--------|
| **Semantic Reasoning** | 30% | 0/10 | 10/10 | GraphDB |
| **Query Performance** | 25% | 8/10 | 9/10 | GraphDB |
| **Operational Simplicity** | 20% | 9/10 | 5/10 | PostgreSQL |
| **Automation** | 15% | 3/10 | 10/10 | GraphDB |
| **Cost** | 10% | 8/10 | 5/10 | PostgreSQL |
| **Total** | 100% | **5.5/10** | **8.2/10** | **GraphDB** |

**Recommendation**: Proceed with GraphDB-centric transformation

---

**Document Version**: 1.0
**Created**: November 22, 2025
**Status**: Ready for Implementation
**Approval**: Pending
