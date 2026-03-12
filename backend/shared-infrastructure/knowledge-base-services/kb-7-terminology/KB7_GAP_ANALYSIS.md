# KB-7 Terminology Service: Gap Analysis Report

**Generated:** 2025-12-04
**Scope:** Specification vs Implementation Cross-Check
**Service:** KB-7 Terminology & Coding Service

---

## Executive Summary

The KB-7 Terminology Service implementation is a **hybrid Go/Python architecture** that provides clinical terminology operations. While the core infrastructure is sophisticated (semantic reasoning, multi-backend search, regional terminology support), several features specified in the service requirements are either missing or incomplete.

| Category | Coverage | Status |
|----------|----------|--------|
| Code Systems | 87.5% | 7/8 systems |
| Core Operations | 62.5% | 5/8 operations |
| API Endpoints | ~70% | Multiple stubs |
| Built-in Value Sets | 0% | 0/18 value sets |
| HCC/RAF Support | 0% | Not implemented |

---

## Architecture Overview

### Actual Implementation (Hybrid Architecture)

```
┌─────────────────────────────────────────────────────────────────┐
│                    KB-7 Terminology Service                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐    ┌──────────────────────────────────┐   │
│  │   Go REST API    │    │      Python FHIR Service         │   │
│  │   (Port 8087)    │    │      (FastAPI Router)            │   │
│  │                  │    │                                  │   │
│  │  /v1/concepts    │    │  /fhir/CodeSystem/$lookup        │   │
│  │  /v1/valuesets   │    │  /fhir/ValueSet/$expand          │   │
│  │  /v1/mappings    │    │  /fhir/ConceptMap/$translate     │   │
│  │  /v1/systems     │    │  /fhir/$validate-code            │   │
│  │  /graphql        │    │                                  │   │
│  └────────┬─────────┘    └─────────────┬────────────────────┘   │
│           │                            │                         │
│           └──────────┬─────────────────┘                         │
│                      │                                           │
│  ┌───────────────────▼───────────────────────────────────────┐  │
│  │                   Core Services Layer                      │  │
│  │                                                            │  │
│  │  TerminologyService    ConceptMapService    SNOMEDService  │  │
│  │  SearchEngine          ReasoningEngine      BulkLoader     │  │
│  └───────────────────────────────────────────────────────────┘  │
│                      │                                           │
│  ┌───────────────────▼───────────────────────────────────────┐  │
│  │                   Storage Layer                            │  │
│  │                                                            │  │
│  │  PostgreSQL │ Elasticsearch │ Redis │ GraphDB (RDF)        │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### GraphDB Infrastructure (Available but Not Wired)

```
┌─────────────────────────────────────────────────────────────────┐
│                    GraphDB - kb7-terminology                     │
│                    http://localhost:7200                         │
├─────────────────────────────────────────────────────────────────┤
│  Status: ✅ RUNNING (44 hours)                                   │
│  Repository: kb7-terminology                                     │
│  Endpoint: http://localhost:7200/repositories/kb7-terminology    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Implemented Components (NOT WIRED TO API):                      │
│  ├── internal/semantic/graphdb_client.go     ✅ Full SPARQL      │
│  ├── internal/semantic/reasoning_engine.go   ✅ OWL Reasoning    │
│  ├── semantic/sparql-proxy/main.go           ✅ SPARQL Proxy     │
│  └── scripts/test-graphdb-connection.go      ✅ Connection Test  │
│                                                                  │
│  Missing Wiring:                                                 │
│  ├── config.go: No GRAPHDB_URL env variable                      │
│  ├── main.go: GraphDB client not initialized                     │
│  ├── server.go: No semantic endpoints exposed                    │
│  └── TerminologyService: Not using GraphDB for lookups           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Specification Expectation (Single Service)

```
┌─────────────────────────────────────────┐
│   KB-7 Terminology Service (Port 8086)  │
├─────────────────────────────────────────┤
│  /api/v1/lookup                         │
│  /api/v1/validate                       │
│  /api/v1/search                         │
│  /api/v1/translate                      │
│  /api/v1/subsumes                       │
│  /api/v1/valuesets/*                    │
│  /api/v1/hcc/*                          │
│  /fhir/*                                │
└─────────────────────────────────────────┘
```

---

## Detailed Gap Analysis

### 1. Code Systems Support

| System | Spec URL | Implemented | Location |
|--------|----------|-------------|----------|
| SNOMED CT | `http://snomed.info/sct` | ✅ Yes | `internal/services/snomed_service.go` |
| ICD-10-CM | `http://hl7.org/fhir/sid/icd-10-cm` | ✅ Yes | `internal/bulkload/bulk_loader.go` |
| ICD-10-PCS | `http://hl7.org/fhir/sid/icd-10-pcs` | ⚠️ Partial | Not explicitly listed |
| RxNorm | `http://www.nlm.nih.gov/research/umls/rxnorm` | ✅ Yes | `internal/bulkload/bulk_loader.go` |
| NDC | `http://hl7.org/fhir/sid/ndc` | ✅ Yes | `internal/bulkload/bulk_loader.go` |
| LOINC | `http://loinc.org` | ✅ Yes | `internal/bulkload/bulk_loader.go` |
| CPT | `http://www.ama-assn.org/go/cpt` | ✅ Yes | `internal/bulkload/bulk_loader.go` |
| **HCC** | `http://cms.gov/hcc` | ❌ **NO** | Not implemented |

---

### 2. Core Operations

| Operation | Spec | Go API | Python FHIR | Notes |
|-----------|------|--------|-------------|-------|
| Code Lookup | ✅ | ✅ `GET /v1/concepts/:system/:code` | ✅ `/fhir/CodeSystem/$lookup` | Fully implemented |
| Code Validation | ✅ | ✅ `POST /v1/concepts/validate` | ✅ `/fhir/$validate-code` | Fully implemented |
| Code Search | ✅ | ✅ `GET /v1/concepts?q=` | ✅ `/terminology/search` | Advanced search engine |
| Code Translation | ✅ | ❌ Not exposed | ✅ `/fhir/ConceptMap/$translate` | Go has service but no endpoint |
| Value Set Expansion | ✅ | ⚠️ Stub | ✅ `/fhir/ValueSet/$expand` | Go returns "implementation pending" |
| **Subsumption Testing** | ✅ | ❌ **Missing** | ❌ **Missing** | Not implemented |
| **HCC Mapping** | ✅ | ❌ **Missing** | ❌ **Missing** | Not implemented |
| **RAF Calculation** | ✅ | ❌ **Missing** | ❌ **Missing** | Not implemented |

---

### 3. API Endpoint Gaps

#### 3.1 Missing Endpoints (Critical)

```yaml
# HCC/Risk Adjustment - NOT IMPLEMENTED
GET  /api/v1/hcc/lookup:
  description: "Map ICD-10 code to HCC category"
  params: [icd10, model]
  status: NOT_IMPLEMENTED
  priority: P0

GET  /api/v1/hcc/hierarchy:
  description: "Get HCC hierarchy rules"
  params: [hcc]
  status: NOT_IMPLEMENTED
  priority: P0

POST /api/v1/hcc/raf:
  description: "Calculate Risk Adjustment Factor"
  body: {icd10Codes: [], model: string, setting: string}
  status: NOT_IMPLEMENTED
  priority: P0

# Subsumption - NOT IMPLEMENTED
GET  /api/v1/subsumes:
  description: "Test is-a relationship between codes"
  params: [system, codeA, codeB]
  status: NOT_IMPLEMENTED
  priority: P1

# Translation API - Logic exists but not exposed
GET/POST /api/v1/translate:
  description: "Translate single code between systems"
  status: SERVICE_EXISTS_NO_ENDPOINT
  location: internal/services/concept_map_service.go
  priority: P0

POST /api/v1/translate/batch:
  description: "Batch translate multiple codes"
  status: SERVICE_EXISTS_NO_ENDPOINT
  location: internal/services/concept_map_service.go:184
  priority: P1
```

#### 3.2 Stub Implementations (Need Completion)

| Endpoint | File | Line | Current Response |
|----------|------|------|------------------|
| `GET /v1/systems` | `internal/api/server.go` | 131 | "implementation pending" |
| `GET /v1/valuesets` | `internal/api/server.go` | 237 | "implementation pending" |
| `POST /v1/valuesets/:url/expand` | `internal/api/server.go` | 266 | "implementation pending" |
| `GET /v1/mappings` | `internal/api/server.go` | 278 | "implementation pending" |
| `POST /v1/concepts/batch-lookup` | `internal/api/server.go` | 300 | "implementation pending" |
| `POST /v1/concepts/batch-validate` | `internal/api/server.go` | 325 | "implementation pending" |

---

### 4. Built-in Value Sets (0/18 Implemented)

#### Spec Requires These Pre-loaded Value Sets:

**Condition Value Sets:**
| ID | Name | System | Status |
|----|------|--------|--------|
| `vs-diabetes-conditions` | Diabetes Mellitus | ICD-10-CM | ❌ Missing |
| `vs-hypertension-conditions` | Hypertension | ICD-10-CM | ❌ Missing |
| `vs-heart-failure-conditions` | Heart Failure | ICD-10-CM | ❌ Missing |
| `vs-ckd-conditions` | Chronic Kidney Disease | ICD-10-CM | ❌ Missing |
| `vs-afib-conditions` | Atrial Fibrillation | ICD-10-CM | ❌ Missing |
| `vs-copd-conditions` | COPD | ICD-10-CM | ❌ Missing |

**Medication Value Sets:**
| ID | Name | System | Status |
|----|------|--------|--------|
| `vs-statin-medications` | Statins | RxNorm | ❌ Missing |
| `vs-acei-medications` | ACE Inhibitors | RxNorm | ❌ Missing |
| `vs-arb-medications` | ARBs | RxNorm | ❌ Missing |
| `vs-sglt2i-medications` | SGLT2 Inhibitors | RxNorm | ❌ Missing |
| `vs-glp1-medications` | GLP-1 Agonists | RxNorm | ❌ Missing |
| `vs-anticoagulant-medications` | Anticoagulants | RxNorm | ❌ Missing |

**Lab Value Sets:**
| ID | Name | System | Status |
|----|------|--------|--------|
| `vs-hba1c-labs` | HbA1c Tests | LOINC | ❌ Missing |
| `vs-kidney-function-labs` | Kidney Function | LOINC | ❌ Missing |
| `vs-lipid-panel-labs` | Lipid Panel | LOINC | ❌ Missing |
| `vs-vital-signs` | Vital Signs | LOINC | ❌ Missing |

---

### 5. HCC Mapping System (Entirely Missing)

#### Required HCC Categories (Not Implemented)

| HCC | Category | RAF (2024) | ICD-10 Examples |
|-----|----------|-----------|-----------------|
| HCC17 | Diabetes with Acute Complications | 0.302 | E10.10, E11.01 |
| HCC18 | Diabetes with Chronic Complications | 0.302 | E11.21, E11.65 |
| HCC19 | Diabetes without Complication | 0.104 | E11.9 |
| HCC85 | Congestive Heart Failure | 0.323 | I50.22, I50.32 |
| HCC96 | Specified Heart Arrhythmias | 0.273 | I48.0, I48.91 |
| HCC111 | COPD | 0.335 | J44.1, J44.9 |
| HCC136 | CKD Stage 5 | 0.237 | N18.5 |
| HCC137 | CKD Stage 4 | 0.237 | N18.4 |
| HCC138 | CKD Stage 3 | 0.069 | N18.3 |
| HCC59 | Major Depressive Disorders | 0.309 | F32.1, F33.1 |

#### Required Hierarchy Rules (Not Implemented)

```
Diabetes:     HCC17 > HCC18 > HCC19
CKD:          HCC136 > HCC137 > HCC138
CHF:          HCC85 > HCC86
Depression:   HCC59 > HCC60
```

---

### 6. GraphDB Integration Gap (Critical)

**Status:** GraphDB components exist but are NOT connected to the main service.

#### What Exists (Implemented)

| Component | File | Status |
|-----------|------|--------|
| GraphDB Client | `internal/semantic/graphdb_client.go` | ✅ Full SPARQL client |
| Reasoning Engine | `internal/semantic/reasoning_engine.go` | ✅ OWL 2 RL reasoning |
| SPARQL Proxy | `semantic/sparql-proxy/main.go` | ✅ REST-to-SPARQL gateway |
| Clinical Rules | `internal/semantic/reasoning_engine.go` | ✅ Drug interaction rules |
| Connection Test | `scripts/test-graphdb-connection.go` | ✅ Validates connectivity |

#### GraphDB Client Capabilities (Already Implemented)

```go
// internal/semantic/graphdb_client.go - READY TO USE

type GraphDBClient struct {
    baseURL      string  // http://localhost:7200
    repository   string  // kb7-terminology
    httpClient   *http.Client
}

// Available Methods:
- ExecuteSPARQL(ctx, query)        // Run SPARQL queries
- LoadTurtleFile(ctx, path, ctx)   // Load ontology files
- InsertTriples(ctx, triples)      // Add RDF triples
- ExecuteUpdate(ctx, updateQuery)  // SPARQL UPDATE
- GetConcept(ctx, conceptURI)      // Get concept by URI
- GetMappings(ctx, code, system)   // Get terminology mappings
- GetDrugInteractions(ctx, uri)    // Drug interaction queries
- HealthCheck(ctx)                 // Verify connectivity
```

#### What's Missing (Not Wired)

| Gap | Location | Impact |
|-----|----------|--------|
| No `GRAPHDB_URL` in config | `internal/config/config.go` | Can't configure GraphDB connection |
| No GraphDB init in main | `cmd/server/main.go` | Client never instantiated |
| No semantic endpoints | `internal/api/server.go` | Reasoning not exposed via REST |
| TerminologyService bypass | `internal/services/terminology_service.go` | Queries go to PostgreSQL, not GraphDB |

#### GraphDB Running Status

```
Container: graphdb-kb7
Repository: kb7-terminology
HTTP URL: http://localhost:7200
SPARQL Endpoint: http://localhost:7200/repositories/kb7-terminology
Status: ✅ Running (44+ hours)
```

---

### 7. Configuration Discrepancies

| Setting | Specification | Implementation | Location |
|---------|--------------|----------------|----------|
| Default Port | 8086 | 8087 | `internal/config/config.go:52` |
| API Prefix | `/api/v1/` | `/v1/` | `internal/api/server.go:64` |
| Database | Not specified | PostgreSQL + Redis + ES + GraphDB | Multiple |
| GraphDB URL | Not in config | Should be `http://localhost:7200` | Missing |
| GraphDB Repository | Not in config | Should be `kb7-terminology` | Missing |

---

## Remediation Roadmap

### Phase 0: GraphDB Integration (P0) - Day 1-2

**Priority:** HIGHEST - Infrastructure already running, just needs wiring.

#### 0.1 Add GraphDB Configuration
**Effort:** 30 minutes
**File:** `internal/config/config.go`

```go
// Add to Config struct
GraphDBURL        string `json:"graphdb_url"`
GraphDBRepository string `json:"graphdb_repository"`
GraphDBUsername   string `json:"graphdb_username"`
GraphDBPassword   string `json:"graphdb_password"`

// Add to Load() function
GraphDBURL:        getEnv("GRAPHDB_URL", "http://localhost:7200"),
GraphDBRepository: getEnv("GRAPHDB_REPOSITORY", "kb7-terminology"),
GraphDBUsername:   getEnv("GRAPHDB_USERNAME", ""),
GraphDBPassword:   getEnv("GRAPHDB_PASSWORD", ""),
```

#### 0.2 Initialize GraphDB Client in Main
**Effort:** 1 hour
**File:** `cmd/server/main.go`

```go
import "kb-7-terminology/internal/semantic"

// After config load
graphdbClient := semantic.NewGraphDBClient(
    config.GraphDBURL,
    config.GraphDBRepository,
    logger,
)

// Set auth if configured
if config.GraphDBUsername != "" {
    graphdbClient.SetAuthentication(config.GraphDBUsername, config.GraphDBPassword)
}

// Health check
if err := graphdbClient.HealthCheck(context.Background()); err != nil {
    logger.WithError(err).Warn("GraphDB not available - semantic features disabled")
} else {
    logger.Info("GraphDB connected successfully")
}

// Create reasoning engine
reasoningEngine := semantic.NewReasoningEngine(graphdbClient, logger)

// Pass to TerminologyService
terminologyService := services.NewTerminologyService(db, cache, logger, graphdbClient, reasoningEngine)
```

#### 0.3 Add Semantic Endpoints to Server
**Effort:** 2-3 hours
**File:** `internal/api/server.go`

```go
// Add to Server struct
graphdbClient    *semantic.GraphDBClient
reasoningEngine  *semantic.ReasoningEngine

// Add to SetupRoutes()
v1.GET("/concepts/:system/:code/semantic", s.semanticLookup)
v1.GET("/concepts/:system/:code/mappings/semantic", s.semanticMappings)
v1.POST("/reasoning/infer", s.performInference)
v1.GET("/sparql", s.sparqlQuery)  // Optional: Direct SPARQL access
```

#### 0.4 Wire TerminologyService to GraphDB
**Effort:** 2-3 hours
**File:** `internal/services/terminology_service.go`

Add GraphDB as a secondary lookup source for:
- Concept lookups (fallback to semantic search)
- Mapping queries (use SPARQL for complex mappings)
- Hierarchy traversal (SNOMED CT is-a relationships)

```go
// Hybrid lookup: PostgreSQL first, GraphDB for semantic enrichment
func (s *TerminologyService) LookupConcept(system, code string) (*models.Concept, error) {
    // 1. Fast lookup from PostgreSQL
    concept, err := s.db.GetConcept(system, code)
    if err != nil {
        return nil, err
    }

    // 2. Enrich with GraphDB semantic data
    if s.graphdb != nil {
        semanticData, _ := s.graphdb.GetConcept(ctx, concept.URI)
        concept.SemanticProperties = semanticData
    }

    return concept, nil
}
```

#### Verification Steps

```bash
# 1. Test GraphDB connection
curl http://localhost:7200/rest/repositories/kb7-terminology

# 2. Test SPARQL endpoint
curl -X POST http://localhost:7200/repositories/kb7-terminology \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "Accept: application/sparql-results+json" \
  -d "query=SELECT * WHERE { ?s ?p ?o } LIMIT 10"

# 3. Run connection test script
go run scripts/test-graphdb-connection.go
```

---

### Phase 1: Critical (P0) - Week 1-2

#### 1.1 Expose Translation API
**Effort:** Low (2-4 hours)
**Files to modify:** `internal/api/server.go`

```go
// Add to v1 route group
v1.POST("/translate", s.translateConcept)
v1.POST("/translate/batch", s.batchTranslateConcepts)
```

The service implementation already exists in `internal/services/concept_map_service.go`.

#### 1.2 Complete Stub Implementations
**Effort:** Medium (1-2 days)
**Files to modify:** `internal/api/server.go`

Connect existing service methods to stub endpoints:
- `listTerminologySystems` → Query `terminology_systems` table
- `listValueSets` → Query `value_sets` table
- `expandValueSet` → Use `TerminologyService.ExpandValueSet()`
- `batchLookupConcepts` → Use parallel `LookupConcept()` calls
- `batchValidateCodes` → Use parallel `ValidateCode()` calls

#### 1.3 HCC Mapping System
**Effort:** High (3-5 days)
**New files required:**
```
internal/services/hcc_service.go      # HCC mapping logic
internal/models/hcc_models.go         # HCC data structures
migrations/XXXX_hcc_tables.sql        # Database schema
data/hcc_mappings_v24.json            # HCC-ICD10 mappings
```

**Implementation approach:**
1. Create HCC database schema
2. Load CMS HCC model v24 mappings
3. Implement hierarchy rules engine
4. Add RAF calculation logic
5. Expose via REST endpoints

---

### Phase 2: Important (P1) - Week 3-4

#### 2.1 Subsumption Testing
**Effort:** Medium (2-3 days)
**Approach:** Leverage existing `ReasoningEngine` in `internal/semantic/reasoning_engine.go`

```go
// New endpoint
GET /v1/concepts/subsumes?system=&codeA=&codeB=

// Response
{
  "outcome": "equivalent|subsumes|subsumedBy|notSubsumed",
  "codeA": {...},
  "codeB": {...}
}
```

#### 2.2 Pre-load Clinical Value Sets
**Effort:** Medium (2-3 days)
**Files to create:**
```
data/valuesets/
├── conditions/
│   ├── vs-diabetes-conditions.json
│   ├── vs-hypertension-conditions.json
│   └── ...
├── medications/
│   ├── vs-statin-medications.json
│   └── ...
└── labs/
    ├── vs-hba1c-labs.json
    └── ...
```

#### 2.3 Port/Path Alignment
**Effort:** Low (1 hour)
**Decision required:** Update implementation to match spec OR update spec to match implementation

Option A - Update implementation:
```go
// config.go
Port: getEnvAsInt("PORT", 8086),  // Change from 8087

// server.go
v1 := router.Group("/api/v1")     // Add /api prefix
```

Option B - Update specification to reflect actual architecture.

---

### Phase 3: Nice-to-Have (P2) - Week 5+

#### 3.1 ICD-10-PCS Explicit Support
Add to bulk loader and search configuration.

#### 3.2 Enhanced HCC Features
- HCC gap identification
- Prospective vs retrospective RAF
- Model version switching (v24, v28)

#### 3.3 GraphQL Schema Completion
Current GraphQL handler returns stub response.

---

## Implementation Priority Matrix

| Feature | Business Impact | Technical Effort | Priority | Timeline |
|---------|----------------|------------------|----------|----------|
| **GraphDB Wiring** | High (Semantic) | Low (exists!) | **P0** | Day 1-2 |
| HCC Mapping | High (CDI/Billing) | High | **P0** | Week 1-2 |
| Translation API exposure | High (Interop) | Low | **P0** | Day 3 |
| Complete stubs | Medium (Functionality) | Medium | **P0** | Week 1 |
| Subsumption | Medium (Semantic) | Low (use GraphDB) | **P1** | Week 2 |
| Value Sets | Medium (Quality) | Medium | **P1** | Week 3 |
| Port alignment | Low (Config) | Low | **P2** | Anytime |

### Quick Wins (< 1 day effort)

| Task | Effort | Impact |
|------|--------|--------|
| Add GraphDB config vars | 30 min | Enables semantic features |
| Initialize GraphDB client in main.go | 1 hour | Connects existing client |
| Expose `/translate` endpoint | 2 hours | Service already exists |
| Add `/sparql` passthrough | 1 hour | Direct GraphDB access |

---

## Appendix A: File Reference

### Core Implementation Files
| File | Purpose | Lines |
|------|---------|-------|
| `cmd/server/main.go` | HTTP server entry point | ~150 |
| `internal/api/server.go` | REST API routes | ~450 |
| `internal/services/terminology_service.go` | Core terminology ops | ~600 |
| `internal/services/concept_map_service.go` | Translation service | ~640 |
| `internal/services/snomed_service.go` | SNOMED CT operations | ~500 |
| `internal/search/search_engine.go` | Advanced search | ~800 |
| `internal/semantic/reasoning_engine.go` | OWL reasoning | ~600 |
| `internal/bulkload/bulk_loader.go` | Data migration | ~700 |
| `fhir/endpoints.py` | Python FHIR API | ~945 |

### Configuration Files
| File | Purpose |
|------|---------|
| `internal/config/config.go` | Environment configuration |
| `go.mod` | Go dependencies |
| `fhir/google_config.py` | Google FHIR integration |

---

## Appendix B: Database Schema Requirements for HCC

```sql
-- HCC Categories
CREATE TABLE hcc_categories (
    hcc_code VARCHAR(10) PRIMARY KEY,
    category_name VARCHAR(255) NOT NULL,
    description TEXT,
    model_version VARCHAR(10) NOT NULL,
    raf_community DECIMAL(5,3),
    raf_institutional DECIMAL(5,3),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ICD-10 to HCC Mappings
CREATE TABLE icd10_hcc_mappings (
    id SERIAL PRIMARY KEY,
    icd10_code VARCHAR(10) NOT NULL,
    hcc_code VARCHAR(10) NOT NULL REFERENCES hcc_categories(hcc_code),
    model_version VARCHAR(10) NOT NULL,
    effective_date DATE,
    termination_date DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(icd10_code, hcc_code, model_version)
);

-- HCC Hierarchy Rules
CREATE TABLE hcc_hierarchy (
    id SERIAL PRIMARY KEY,
    higher_hcc VARCHAR(10) NOT NULL REFERENCES hcc_categories(hcc_code),
    lower_hcc VARCHAR(10) NOT NULL REFERENCES hcc_categories(hcc_code),
    model_version VARCHAR(10) NOT NULL,
    UNIQUE(higher_hcc, lower_hcc, model_version)
);

-- Indexes
CREATE INDEX idx_icd10_hcc_icd10 ON icd10_hcc_mappings(icd10_code);
CREATE INDEX idx_icd10_hcc_model ON icd10_hcc_mappings(model_version);
CREATE INDEX idx_hcc_hierarchy_higher ON hcc_hierarchy(higher_hcc);
```

---

## Appendix C: Sample HCC Service Interface

```go
// internal/services/hcc_service.go

type HCCService interface {
    // Lookup HCC for ICD-10 code
    LookupHCC(icd10Code string, model string) (*HCCMapping, error)

    // Get HCC hierarchy
    GetHCCHierarchy(hccCode string, model string) (*HCCHierarchy, error)

    // Calculate RAF score
    CalculateRAF(request RAFRequest) (*RAFResponse, error)

    // Batch HCC lookup
    BatchLookupHCC(icd10Codes []string, model string) ([]HCCMapping, error)

    // Apply hierarchy rules
    ApplyHierarchy(hccCodes []string, model string) ([]string, error)
}

type RAFRequest struct {
    ICD10Codes []string `json:"icd10_codes"`
    Model      string   `json:"model"`      // v24, v28
    Setting    string   `json:"setting"`    // community, institutional
    Age        int      `json:"age,omitempty"`
    Gender     string   `json:"gender,omitempty"`
}

type RAFResponse struct {
    TotalRAF       float64      `json:"total_raf"`
    HCCsIdentified []string     `json:"hccs_identified"`
    HCCsAfterHierarchy []string `json:"hccs_after_hierarchy"`
    Breakdown      []RAFDetail  `json:"breakdown"`
}
```

---

---

## Phase 4: Rule Engine Infrastructure

### Overview

Phase 4 transforms KB-7 from a system with **hardcoded value sets** to a **database-driven rule engine** that can be updated without code changes.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    PHASE 4: RULE ENGINE ARCHITECTURE                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   BEFORE (Current - Phases 0-3):                                        │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  builtin_valuesets.go (18 hardcoded value sets)                 │   │
│   │  → Edit Go code → Rebuild → Deploy (2-3 days per change)        │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│   AFTER (Phase 4):                                                      │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  PostgreSQL value_set_definitions table (5000+ rules)           │   │
│   │  → INSERT/UPDATE SQL → Instant (5 minutes per change)           │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 4.1 PostgreSQL `value_set_definitions` Table 🔴

**Priority:** CRITICAL
**Effort:** 4-6 hours
**File:** `migrations/XXX_value_set_definitions.sql`

#### What It Does

Stores the **definitions** (not hardcoded data) of clinical value sets in a database table.

#### Schema Definition

```sql
CREATE TABLE value_set_definitions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(255) NOT NULL UNIQUE,
    url                 VARCHAR(500) UNIQUE,
    title               VARCHAR(500),
    description         TEXT,
    publisher           VARCHAR(255),

    -- Definition Type: How to resolve this value set
    definition_type     VARCHAR(50) NOT NULL CHECK (
        definition_type IN ('explicit', 'extensional', 'intensional')
    ),

    -- For EXPLICIT definitions (fixed code list)
    explicit_codes      JSONB,  -- [{"system":"...", "code":"...", "display":"..."}]

    -- For INTENSIONAL definitions (graph-based expansion)
    root_concept_code   VARCHAR(100),
    root_concept_system VARCHAR(255),
    expansion_rule      VARCHAR(50) CHECK (
        expansion_rule IN ('descendants', 'ancestors', 'descendants_or_self', 'ancestors_or_self')
    ),

    -- For EXTENSIONAL definitions (composed from other value sets)
    composed_of         JSONB,  -- [{"valueset_id": "...", "operation": "include|exclude"}]

    -- Metadata
    version             VARCHAR(50) DEFAULT '1.0.0',
    status              VARCHAR(20) DEFAULT 'active' CHECK (
        status IN ('draft', 'active', 'retired', 'unknown')
    ),
    clinical_domain     VARCHAR(100),
    use_context         JSONB,

    -- Auditing
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE,
    created_by          VARCHAR(255),
    updated_by          VARCHAR(255)
);

-- Indexes for fast lookup
CREATE INDEX idx_vsdef_name ON value_set_definitions(name);
CREATE INDEX idx_vsdef_url ON value_set_definitions(url);
CREATE INDEX idx_vsdef_status ON value_set_definitions(status);
CREATE INDEX idx_vsdef_domain ON value_set_definitions(clinical_domain);
CREATE INDEX idx_vsdef_type ON value_set_definitions(definition_type);
```

#### Definition Types Explained

| Type | Description | Example |
|------|-------------|---------|
| `explicit` | Fixed list of codes stored directly in the table | Administrative gender: male, female, other, unknown |
| `extensional` | Composed from other value sets | "All cardiac conditions" = CHF + MI + AFib value sets |
| `intensional` | Expanded at runtime from graph hierarchy | "All diabetes codes" = descendants of SNOMED 73211009 |

#### Clinical Example

```sql
-- EXPLICIT: Fixed list (no graph query needed)
INSERT INTO value_set_definitions (name, url, definition_type, explicit_codes) VALUES (
    'administrative-gender',
    'http://hl7.org/fhir/ValueSet/administrative-gender',
    'explicit',
    '[
        {"system":"http://hl7.org/fhir/administrative-gender","code":"male","display":"Male"},
        {"system":"http://hl7.org/fhir/administrative-gender","code":"female","display":"Female"},
        {"system":"http://hl7.org/fhir/administrative-gender","code":"other","display":"Other"},
        {"system":"http://hl7.org/fhir/administrative-gender","code":"unknown","display":"Unknown"}
    ]'
);

-- INTENSIONAL: Expanded via GraphDB SPARQL query
INSERT INTO value_set_definitions (
    name, url, definition_type,
    root_concept_code, root_concept_system, expansion_rule,
    clinical_domain
) VALUES (
    'diabetes-all-types',
    'http://cardiofit.ai/ValueSet/diabetes-all-types',
    'intensional',
    '73211009',                        -- SNOMED CT root for Diabetes Mellitus
    'http://snomed.info/sct',
    'descendants_or_self',             -- Include root + all children
    'endocrinology'
);
-- At runtime: Queries GraphDB for all concepts where rdfs:subClassOf* :73211009
-- Returns 500+ diabetes codes (T1DM, T2DM, LADA, MODY, gestational, secondary, etc.)
```

#### Business Value

| Without (Current) | With Phase 4.1 |
|-------------------|----------------|
| 18 hardcoded value sets | 5,000+ value sets manageable |
| Developer required for updates | Clinical informaticist can update |
| 2-3 day deployment cycle | 5-minute configuration change |
| Code review required | SQL audit trail |

---

### 4.2 RuleManager Service (Go) 🔴

**Priority:** CRITICAL
**Effort:** 2-3 days
**File:** `internal/services/rule_manager.go`

#### What It Does

A Go service that:
1. Reads rule definitions from PostgreSQL
2. Expands `intensional` rules by querying GraphDB
3. Caches expanded results in Redis
4. Returns the full code list to callers

#### Architecture Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         RULE MANAGER FLOW                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   API Request: GET /api/v1/valuesets/diabetes-all-types/expand          │
│         │                                                               │
│         ▼                                                               │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ Step 1: Check Redis Cache                                       │   │
│   │         Key: "valueset:expanded:diabetes-all-types:v1.0.0"      │   │
│   │                                                                 │   │
│   │         CACHE HIT?  → Return cached list (< 1ms)                │   │
│   │         CACHE MISS? → Continue to Step 2                        │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│         │                                                               │
│         ▼                                                               │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ Step 2: Read Definition from PostgreSQL                         │   │
│   │                                                                 │   │
│   │   SELECT * FROM value_set_definitions                           │   │
│   │   WHERE name = 'diabetes-all-types';                            │   │
│   │                                                                 │   │
│   │   Result: {                                                     │   │
│   │     definition_type: "intensional",                             │   │
│   │     root_concept_code: "73211009",                              │   │
│   │     root_concept_system: "http://snomed.info/sct",              │   │
│   │     expansion_rule: "descendants_or_self"                       │   │
│   │   }                                                             │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│         │                                                               │
│         ▼                                                               │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ Step 3: Execute SPARQL on GraphDB (for intensional only)        │   │
│   │                                                                 │   │
│   │   SELECT ?code ?display WHERE {                                 │   │
│   │     ?concept rdfs:subClassOf* snomed:73211009 .                 │   │
│   │     ?concept skos:notation ?code .                              │   │
│   │     ?concept skos:prefLabel ?display .                          │   │
│   │   }                                                             │   │
│   │                                                                 │   │
│   │   Result: 500+ diabetes concept codes                           │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│         │                                                               │
│         ▼                                                               │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ Step 4: Cache Result in Redis                                   │   │
│   │                                                                 │   │
│   │   Key: "valueset:expanded:diabetes-all-types:v1.0.0"            │   │
│   │   TTL: 24 hours                                                 │   │
│   │   Value: [500+ codes as JSON array]                             │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│         │                                                               │
│         ▼                                                               │
│   Return expanded value set (500+ codes)                                │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### Interface Definition

```go
// internal/services/rule_manager.go

type RuleManager interface {
    // ExpandValueSet returns all codes for a value set (handles all definition types)
    ExpandValueSet(ctx context.Context, identifier string, version string) (*ExpandedValueSet, error)

    // ValidateCodeInValueSet checks if a code is in a value set
    ValidateCodeInValueSet(ctx context.Context, code, system, valueSetID string) (*ValidationResult, error)

    // GetValueSetDefinition returns the raw definition (not expanded)
    GetValueSetDefinition(ctx context.Context, identifier string) (*ValueSetDefinition, error)

    // RefreshCache forces re-expansion of a value set (invalidates cache)
    RefreshCache(ctx context.Context, identifier string) error

    // ListValueSets returns all value set definitions with optional filtering
    ListValueSets(ctx context.Context, filter ValueSetFilter) ([]ValueSetDefinition, error)
}

type ExpandedValueSet struct {
    Identifier    string              `json:"identifier"`
    Version       string              `json:"version"`
    Total         int                 `json:"total"`
    Codes         []ExpandedCode      `json:"codes"`
    ExpansionTime time.Time           `json:"expansion_time"`
    CachedResult  bool                `json:"cached_result"`
}

type ExpandedCode struct {
    System  string `json:"system"`
    Code    string `json:"code"`
    Display string `json:"display"`
}
```

#### Clinical Use Cases

| Use Case | Query | Result |
|----------|-------|--------|
| "Does patient have diabetes?" | Expand diabetes value set → Check if any patient code is in list | Match across 500+ diabetes codes |
| HEDIS quality measure | Expand "Comprehensive Diabetes Care" measure codes | All codes required for measure |
| CDS alert | "Is this medication an anticoagulant?" | Check against all 1000+ anticoagulant drugs |
| Population health | "All patients with heart failure" | Expand CHF value set → Query patient records |

---

### 4.3 Migrate Built-in Value Sets to Database 🟡

**Priority:** IMPORTANT
**Effort:** 1 day
**Files:** Migration script + seed data

#### What It Does

Moves the 18 FHIR R4 value sets from `builtin_valuesets.go` into the PostgreSQL `value_set_definitions` table.

#### Migration Script

```sql
-- migrations/XXX_migrate_builtin_valuesets.sql

-- 1. Administrative Gender (explicit)
INSERT INTO value_set_definitions (name, url, definition_type, explicit_codes, status, clinical_domain)
VALUES (
    'administrative-gender',
    'http://hl7.org/fhir/ValueSet/administrative-gender',
    'explicit',
    '[
        {"system":"http://hl7.org/fhir/administrative-gender","code":"male","display":"Male"},
        {"system":"http://hl7.org/fhir/administrative-gender","code":"female","display":"Female"},
        {"system":"http://hl7.org/fhir/administrative-gender","code":"other","display":"Other"},
        {"system":"http://hl7.org/fhir/administrative-gender","code":"unknown","display":"Unknown"}
    ]',
    'active',
    'demographics'
);

-- 2. Observation Status (explicit)
INSERT INTO value_set_definitions (name, url, definition_type, explicit_codes, status, clinical_domain)
VALUES (
    'observation-status',
    'http://hl7.org/fhir/ValueSet/observation-status',
    'explicit',
    '[
        {"system":"http://hl7.org/fhir/observation-status","code":"registered","display":"Registered"},
        {"system":"http://hl7.org/fhir/observation-status","code":"preliminary","display":"Preliminary"},
        {"system":"http://hl7.org/fhir/observation-status","code":"final","display":"Final"},
        {"system":"http://hl7.org/fhir/observation-status","code":"amended","display":"Amended"},
        {"system":"http://hl7.org/fhir/observation-status","code":"corrected","display":"Corrected"},
        {"system":"http://hl7.org/fhir/observation-status","code":"cancelled","display":"Cancelled"},
        {"system":"http://hl7.org/fhir/observation-status","code":"entered-in-error","display":"Entered in Error"},
        {"system":"http://hl7.org/fhir/observation-status","code":"unknown","display":"Unknown"}
    ]',
    'active',
    'observations'
);

-- ... (repeat for all 18 value sets)

-- 18. Vital Signs (explicit - LOINC codes)
INSERT INTO value_set_definitions (name, url, definition_type, explicit_codes, status, clinical_domain)
VALUES (
    'vital-signs',
    'http://hl7.org/fhir/ValueSet/observation-vitalsignresult',
    'explicit',
    '[
        {"system":"http://loinc.org","code":"85354-9","display":"Blood pressure panel"},
        {"system":"http://loinc.org","code":"8480-6","display":"Systolic blood pressure"},
        {"system":"http://loinc.org","code":"8462-4","display":"Diastolic blood pressure"},
        {"system":"http://loinc.org","code":"8867-4","display":"Heart rate"},
        {"system":"http://loinc.org","code":"9279-1","display":"Respiratory rate"},
        {"system":"http://loinc.org","code":"8310-5","display":"Body temperature"},
        {"system":"http://loinc.org","code":"29463-7","display":"Body weight"},
        {"system":"http://loinc.org","code":"8302-2","display":"Body height"},
        {"system":"http://loinc.org","code":"39156-5","display":"Body mass index"},
        {"system":"http://loinc.org","code":"2708-6","display":"Oxygen saturation"}
    ]',
    'active',
    'vital-signs'
);
```

#### Before/After Comparison

| Aspect | Before (builtin_valuesets.go) | After (PostgreSQL) |
|--------|-------------------------------|-------------------|
| Storage | Go source code | Database table |
| Updates | Require code change + deploy | SQL UPDATE statement |
| Who can update | Go developers only | Clinical informaticists |
| Version control | Git | Database versioning + audit |
| Deployment | 2-3 days | Instant |

---

### 4.4 Redis Caching for Expanded Value Sets 🟡

**Priority:** IMPORTANT
**Effort:** 4-6 hours
**File:** Modify `internal/services/rule_manager.go`

#### What It Does

Stores expanded value set code lists in Redis to avoid repeated GraphDB queries.

#### Cache Strategy

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         REDIS CACHING STRATEGY                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   Cache Key Format:                                                     │
│   "kb7:valueset:expanded:{name}:{version}:{hash}"                       │
│                                                                         │
│   Examples:                                                             │
│   - "kb7:valueset:expanded:diabetes-all-types:v1.0.0:abc123"            │
│   - "kb7:valueset:expanded:vital-signs:v2.0.0:def456"                   │
│                                                                         │
│   TTL (Time-to-Live):                                                   │
│   - Explicit value sets: 7 days (rarely change)                         │
│   - Intensional value sets: 24 hours (graph may update)                 │
│   - Extensional value sets: 24 hours (composed sets)                    │
│                                                                         │
│   Cache Invalidation Triggers:                                          │
│   - Value set definition updated in PostgreSQL                          │
│   - Manual refresh via API call                                         │
│   - GraphDB ontology update event (if CDC enabled)                      │
│   - TTL expiration                                                      │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### Performance Impact

```
WITHOUT CACHING:
┌─────────────────────────────────────────────────────────────────┐
│ Request 1: Expand "diabetes" → PostgreSQL + GraphDB → 200ms     │
│ Request 2: Expand "diabetes" → PostgreSQL + GraphDB → 200ms     │
│ Request 3: Expand "diabetes" → PostgreSQL + GraphDB → 200ms     │
│ Request 4: Expand "diabetes" → PostgreSQL + GraphDB → 200ms     │
│ Request 5: Expand "diabetes" → PostgreSQL + GraphDB → 200ms     │
│                                                                 │
│ Total: 1000ms for 5 requests                                    │
│ GraphDB load: 5 SPARQL queries                                  │
└─────────────────────────────────────────────────────────────────┘

WITH CACHING:
┌─────────────────────────────────────────────────────────────────┐
│ Request 1: Expand "diabetes" → CACHE MISS → GraphDB → 200ms     │
│ Request 2: Expand "diabetes" → CACHE HIT → Redis → 1ms          │
│ Request 3: Expand "diabetes" → CACHE HIT → Redis → 1ms          │
│ Request 4: Expand "diabetes" → CACHE HIT → Redis → 1ms          │
│ Request 5: Expand "diabetes" → CACHE HIT → Redis → 1ms          │
│                                                                 │
│ Total: 204ms for 5 requests (5x faster!)                        │
│ GraphDB load: 1 SPARQL query (80% reduction)                    │
└─────────────────────────────────────────────────────────────────┘
```

---

### 4.5 Rule Loader CLI Tool 🟢

**Priority:** NICE-TO-HAVE
**Effort:** 1-2 days
**File:** `cmd/kb7-loader/main.go`

#### What It Does

A command-line tool to bulk-load value sets from CSV, JSON, or external sources.

#### Usage Examples

```bash
# Load value sets from CSV file
./kb7-loader load --file hedis_2024_valuesets.csv --format csv

# Import from VSAC (Value Set Authority Center)
./kb7-loader import --source vsac --oid 2.16.840.1.113883.3.464.1003.103.12.1001

# Sync CMS HEDIS measure value sets
./kb7-loader sync --source cms-hedis --year 2024 --measures "diabetes,hypertension"

# Export value set definitions to JSON
./kb7-loader export --name "diabetes-*" --output diabetes_valuesets.json

# Validate value set definitions
./kb7-loader validate --file custom_valuesets.json

# Refresh cache for specific value set
./kb7-loader cache-refresh --name diabetes-all-types
```

#### Business Value

| Manual Process | With CLI Tool |
|----------------|---------------|
| Write 5,000 INSERT statements | `./kb7-loader load --file hedis.csv` |
| Copy codes from VSAC website | `./kb7-loader import --source vsac --oid X` |
| Annual HEDIS update takes days | `./kb7-loader sync --year 2024` (minutes) |

---

## Phase 5: Graph Data Population

### Overview

Phase 5 populates GraphDB with actual clinical terminology data so the OWL reasoning and subsumption queries return meaningful results.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    PHASE 5: GRAPH DATA POPULATION                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   CURRENT STATE (Phases 0-3):                                           │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  GraphDB is CONNECTED but EMPTY                                 │   │
│   │  Subsumption queries return: UNKNOWN                            │   │
│   │  Ancestor queries return: EMPTY LIST                            │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│   TARGET STATE (Phase 5):                                               │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  GraphDB contains:                                              │   │
│   │  • SNOMED-CT: ~350,000 concepts with hierarchy                  │   │
│   │  • ICD-10 → HCC: ~10,000 mappings                               │   │
│   │  • RxNorm: ~100,000 drug concepts                               │   │
│   │  • Total: ~14 million triples                                   │   │
│   │                                                                 │   │
│   │  Subsumption: "Is MI a Heart Disease?" → TRUE                   │   │
│   │  Ancestors: "Parents of T2DM" → [Diabetes, Endocrine Disease]   │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

### 5.1 Load SNOMED-CT Subset into GraphDB 🔴

**Priority:** CRITICAL
**Effort:** 2-3 days
**Data Size:** ~350,000 concepts, ~500,000 relationships

#### What Is SNOMED-CT?

SNOMED-CT (Systematized Nomenclature of Medicine - Clinical Terms) is the world's most comprehensive clinical terminology, organized in a poly-hierarchy:

```
SNOMED-CT Hierarchy (Simplified View):

Clinical Finding (404684003)
├── Disease (64572001)
│   ├── Cardiovascular Disease (49601007)
│   │   ├── Heart Disease (56265001)
│   │   │   ├── Ischemic Heart Disease (414545008)
│   │   │   │   ├── Myocardial Infarction (22298006)
│   │   │   │   │   ├── Acute MI (57054005)
│   │   │   │   │   ├── STEMI (401303003)
│   │   │   │   │   └── NSTEMI (401314000)
│   │   │   │   ├── Angina Pectoris (194828000)
│   │   │   │   └── Coronary Artery Disease (53741008)
│   │   │   ├── Heart Failure (84114007)
│   │   │   │   ├── Systolic Heart Failure (417996009)
│   │   │   │   ├── Diastolic Heart Failure (418304008)
│   │   │   │   └── CHF with Reduced EF (703272007)
│   │   │   └── Arrhythmia (698247007)
│   │   │       ├── Atrial Fibrillation (49436004)
│   │   │       └── Ventricular Tachycardia (25569003)
│   │   └── Hypertensive Disease (38341003)
│   │       ├── Essential Hypertension (59621000)
│   │       └── Secondary Hypertension (31992008)
│   │
│   ├── Endocrine Disease (362969004)
│   │   └── Diabetes Mellitus (73211009)              ← ROOT FOR "ALL DIABETES"
│   │       ├── Type 1 Diabetes (46635009)
│   │       │   ├── T1DM with Ketoacidosis (420422005)
│   │       │   └── T1DM with Nephropathy (420279001)
│   │       ├── Type 2 Diabetes (44054006)
│   │       │   ├── T2DM with Neuropathy (422088007)
│   │       │   ├── T2DM with Retinopathy (422034002)
│   │       │   └── T2DM with Nephropathy (422166005)
│   │       ├── Gestational Diabetes (11530004)
│   │       ├── Secondary Diabetes (8801005)
│   │       └── LADA (426875007)
│   │
│   └── Respiratory Disease (50043002)
│       └── COPD (13645005)
│           ├── Emphysema (87433001)
│           └── Chronic Bronchitis (63480004)
│
└── Procedure (71388002)
    └── Surgical Procedure (387713003)
        └── Cardiac Surgery (64915003)
```

#### Data Format (Turtle/TTL)

```turtle
@prefix snomed: <http://snomed.info/id/> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .

# Diabetes Mellitus (Root concept for all diabetes types)
snomed:73211009 a owl:Class ;
    skos:notation "73211009" ;
    skos:prefLabel "Diabetes mellitus"@en ;
    rdfs:subClassOf snomed:362969004 .  # Endocrine Disease

# Type 2 Diabetes IS-A Diabetes Mellitus
snomed:44054006 a owl:Class ;
    skos:notation "44054006" ;
    skos:prefLabel "Type 2 diabetes mellitus"@en ;
    rdfs:subClassOf snomed:73211009 .

# Type 2 Diabetes with Nephropathy IS-A Type 2 Diabetes
snomed:422166005 a owl:Class ;
    skos:notation "422166005" ;
    skos:prefLabel "Type 2 diabetes mellitus with diabetic nephropathy"@en ;
    rdfs:subClassOf snomed:44054006 .

# Myocardial Infarction IS-A Ischemic Heart Disease
snomed:22298006 a owl:Class ;
    skos:notation "22298006" ;
    skos:prefLabel "Myocardial infarction"@en ;
    rdfs:subClassOf snomed:414545008 .
```

#### What SNOMED Data Enables

| Query | Before (Empty GraphDB) | After (SNOMED Loaded) |
|-------|------------------------|----------------------|
| "Is MI a Heart Disease?" | Unknown | **TRUE** (via rdfs:subClassOf* chain) |
| "All types of Diabetes" | Empty list | **500+ codes** (T1DM, T2DM, LADA, MODY, etc.) |
| "Ancestors of T2DM with Nephropathy" | None | **[T2DM, Diabetes, Endocrine Disease, Disease, Clinical Finding]** |
| "Common ancestor of T1DM and T2DM" | None | **Diabetes Mellitus (73211009)** |

#### Loading Process

```bash
# 1. Download SNOMED-CT RF2 release (requires UMLS license)
# 2. Convert to RDF using snomed-owl-toolkit
./snomed-owl-toolkit -rf2-snapshot /path/to/snomed -output snomed.owl

# 3. Load into GraphDB
curl -X POST "http://localhost:7200/repositories/kb7-terminology/statements" \
  -H "Content-Type: application/x-turtle" \
  --data-binary @snomed.ttl

# Or use the GraphDB Workbench UI for large files
```

---

### 5.2 Load ICD-10 → HCC Mappings 🔴

**Priority:** CRITICAL
**Effort:** 1 day
**Data Size:** ~10,000 ICD-10 to HCC mappings

#### What Is HCC?

HCC (Hierarchical Condition Categories) is CMS's system for Medicare Advantage risk adjustment. Each HCC category has a **Risk Adjustment Factor (RAF)** coefficient that affects payment.

#### HCC Mapping Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    HCC RISK ADJUSTMENT CALCULATION                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   Patient Problem List:                                                 │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  • E11.9  - Type 2 diabetes without complications               │   │
│   │  • I50.9  - Heart failure, unspecified                          │   │
│   │  • J44.9  - COPD, unspecified                                   │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                          │                                              │
│                          ▼                                              │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  STEP 1: ICD-10 → HCC Mapping                                   │   │
│   │                                                                 │   │
│   │  E11.9 → HCC19 (Diabetes without Complication)     RAF: +0.105  │   │
│   │  I50.9 → HCC85 (Congestive Heart Failure)          RAF: +0.323  │   │
│   │  J44.9 → HCC111 (COPD)                             RAF: +0.335  │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                          │                                              │
│                          ▼                                              │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  STEP 2: Apply Hierarchy Rules                                  │   │
│   │                                                                 │   │
│   │  Diabetes:  HCC17 > HCC18 > HCC19 (HCC19 not trumped)          │   │
│   │  CHF:       HCC85 > HCC86        (HCC85 not trumped)           │   │
│   │  COPD:      HCC111 only          (no hierarchy)                │   │
│   │                                                                 │   │
│   │  Active HCCs: [HCC19, HCC85, HCC111]                           │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                          │                                              │
│                          ▼                                              │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  STEP 3: Calculate RAF Score                                    │   │
│   │                                                                 │   │
│   │  BASE RAF (Average beneficiary):     1.000                      │   │
│   │  + HCC19 (Diabetes):                +0.105                      │   │
│   │  + HCC85 (CHF):                     +0.323                      │   │
│   │  + HCC111 (COPD):                   +0.335                      │   │
│   │  ─────────────────────────────────────                          │   │
│   │  TOTAL RAF:                          1.763                      │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                          │                                              │
│                          ▼                                              │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  FINANCIAL IMPACT                                               │   │
│   │                                                                 │   │
│   │  Average Medicare payment:           $10,000/year               │   │
│   │  Adjusted payment (1.763 × $10,000): $17,630/year               │   │
│   │  Additional revenue per patient:     $7,630/year                │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### Data Format (Turtle)

```turtle
@prefix kb7: <http://cardiofit.ai/kb7/ontology#> .
@prefix icd10: <http://hl7.org/fhir/sid/icd-10-cm/> .
@prefix hcc: <http://cms.gov/hcc/> .

# HCC Category definition
hcc:HCC19 a kb7:HCCCategory ;
    kb7:hccCode "HCC19" ;
    kb7:categoryName "Diabetes without Complication" ;
    kb7:rafCommunity 0.105 ;
    kb7:rafInstitutional 0.105 ;
    kb7:modelVersion "v24" .

# ICD-10 to HCC mapping
icd10:E11.9 kb7:mapsToHCC hcc:HCC19 ;
    kb7:mappingConfidence 1.0 ;
    kb7:effectiveDate "2024-01-01" .

icd10:E11.21 kb7:mapsToHCC hcc:HCC18 ;  # Diabetes with nephropathy → higher HCC
    kb7:mappingConfidence 1.0 .

# HCC Hierarchy rule (HCC17 trumps HCC18 trumps HCC19)
hcc:HCC17 kb7:trumps hcc:HCC18 .
hcc:HCC18 kb7:trumps hcc:HCC19 .
```

---

### 5.3 Load RxNorm Drug Hierarchy 🟡

**Priority:** IMPORTANT
**Effort:** 1-2 days
**Data Size:** ~100,000 drug concepts

#### What Is RxNorm?

RxNorm is the standard U.S. terminology for medications, providing:
- Drug names (brand and generic)
- Ingredients
- Strengths and dose forms
- Drug class hierarchies
- Relationships between drugs

#### RxNorm Hierarchy (Simplified)

```
Pharmaceutical/Biologic Product
├── Cardiovascular Agent
│   ├── Antihypertensive Agent
│   │   ├── ACE Inhibitor
│   │   │   ├── Lisinopril (RxCUI: 29046)
│   │   │   ├── Enalapril (RxCUI: 3827)
│   │   │   ├── Captopril (RxCUI: 1998)
│   │   │   └── Ramipril (RxCUI: 35296)
│   │   ├── ARB (Angiotensin Receptor Blocker)
│   │   │   ├── Losartan (RxCUI: 52175)
│   │   │   └── Valsartan (RxCUI: 69749)
│   │   └── Beta Blocker
│   │       ├── Metoprolol (RxCUI: 6918)
│   │       └── Atenolol (RxCUI: 1202)
│   ├── Anticoagulant
│   │   ├── Warfarin (RxCUI: 11289)
│   │   ├── Heparin (RxCUI: 5224)
│   │   ├── Rivaroxaban (RxCUI: 1114195)
│   │   └── Apixaban (RxCUI: 1364430)
│   └── Statin
│       ├── Atorvastatin (RxCUI: 83367)
│       ├── Simvastatin (RxCUI: 36567)
│       └── Rosuvastatin (RxCUI: 301542)
│
├── Antidiabetic Agent
│   ├── Biguanide
│   │   └── Metformin (RxCUI: 6809)
│   ├── SGLT2 Inhibitor
│   │   ├── Empagliflozin (RxCUI: 1545653)
│   │   └── Dapagliflozin (RxCUI: 1488564)
│   ├── GLP-1 Agonist
│   │   ├── Semaglutide (RxCUI: 1991302)
│   │   └── Liraglutide (RxCUI: 475968)
│   └── Insulin
│       ├── Insulin Regular (RxCUI: 5856)
│       └── Insulin Glargine (RxCUI: 261551)
│
└── Analgesic
    └── NSAID
        ├── Aspirin (RxCUI: 1191)
        ├── Ibuprofen (RxCUI: 5640)
        └── Naproxen (RxCUI: 7258)
```

#### Clinical Use Cases

| Use Case | RxNorm Query | Result |
|----------|--------------|--------|
| Drug-Drug Interaction | "Is Warfarin + Aspirin risky?" | Yes (both affect coagulation pathway) |
| Therapeutic Alternative | "Alternatives to Lisinopril?" | Enalapril, Captopril, Ramipril (same class) |
| Class-based CDS | "Alert for any NSAID with anticoagulant" | Works across 1000s of drug combinations |
| Formulary Check | "Is Lipitor covered?" | Check Atorvastatin (generic) in formulary |
| Brand-Generic Mapping | "Generic for Lipitor?" | Atorvastatin |

---

### 5.4 Configure ELK Reasoner 🟡

**Priority:** IMPORTANT
**Effort:** 4-6 hours

#### What Is ELK?

ELK is an OWL reasoner that **pre-computes** the transitive closure of class hierarchies. Instead of traversing the hierarchy at query time, it creates a materialized table of all ancestor-descendant relationships.

#### Performance Comparison

```
WITHOUT ELK (Runtime Reasoning):
┌─────────────────────────────────────────────────────────────────┐
│ Query: "Is Myocardial Infarction a Cardiovascular Disease?"     │
│                                                                 │
│ GraphDB must traverse at RUNTIME:                               │
│   MI (22298006)                                                 │
│   └── rdfs:subClassOf → Ischemic Heart Disease (414545008)      │
│       └── rdfs:subClassOf → Heart Disease (56265001)            │
│           └── rdfs:subClassOf → Cardiovascular Disease (49601007)│
│                                                                 │
│ Time: 50-200ms per query                                        │
│ CPU: High (graph traversal)                                     │
└─────────────────────────────────────────────────────────────────┘

WITH ELK (Pre-computed Closure):
┌─────────────────────────────────────────────────────────────────┐
│ Query: "Is Myocardial Infarction a Cardiovascular Disease?"     │
│                                                                 │
│ ELK pre-computed this relationship:                             │
│                                                                 │
│ Materialized Table:                                             │
│ ┌──────────────────────┬──────────────────────────┬────────┐    │
│ │ Descendant           │ Ancestor                 │ Depth  │    │
│ ├──────────────────────┼──────────────────────────┼────────┤    │
│ │ MI (22298006)        │ Ischemic HD (414545008)  │ 1      │    │
│ │ MI (22298006)        │ Heart Disease (56265001) │ 2      │    │
│ │ MI (22298006)        │ CV Disease (49601007)    │ 3      │    │
│ │ MI (22298006)        │ Disease (64572001)       │ 4      │    │
│ │ MI (22298006)        │ Clinical Finding         │ 5      │    │
│ └──────────────────────┴──────────────────────────┴────────┘    │
│                                                                 │
│ Direct table lookup → Result: TRUE                              │
│ Time: 1-5ms (40x faster!)                                       │
│ CPU: Minimal (index lookup)                                     │
└─────────────────────────────────────────────────────────────────┘
```

#### GraphDB ELK Configuration

```
# Repository configuration for ELK reasoning
# File: repo-config.ttl

@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix rep: <http://www.openrdf.org/config/repository#> .
@prefix sr: <http://www.openrdf.org/config/repository/sail#> .
@prefix sail: <http://www.openrdf.org/config/sail#> .
@prefix owlim: <http://www.ontotext.com/trree/owlim#> .

[] a rep:Repository ;
   rep:repositoryID "kb7-terminology" ;
   rdfs:label "KB-7 Terminology Repository with ELK Reasoning" ;
   rep:repositoryImpl [
      rep:repositoryType "graphdb:SailRepository" ;
      sr:sailImpl [
         sail:sailType "graphdb:Sail" ;
         owlim:ruleset "owl2-rl-optimized" ;  # Enable OWL 2 RL reasoning
         owlim:check-for-inconsistencies "true" ;
         owlim:disable-sameAs "false" ;
      ]
   ] .
```

---

## Phase 6: Neo4j Integration (CRITICAL - Upgraded from Optional) 🔴

### Overview

**⚠️ PRIORITY UPGRADE: Phase 6 has been elevated from P3 (Optional) to P0 (Critical) based on the "Brain vs Face" architecture analysis.**

Phase 6 adds Neo4j as the **read replica** that enables the Go API service to query the 14M+ triples instead of relying on hardcoded in-memory maps.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    PHASE 6: DUAL GRAPH DATABASE ARCHITECTURE            │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   THE "BRAIN vs FACE" PROBLEM:                                          │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  Go API Service (The Face)    ❌    Graph Pipeline (The Brain)  │   │
│   │  • 45 SNOMED codes           NOT   • 350,000+ SNOMED concepts   │   │
│   │  • 85 RxNorm drugs          TALKING • 100,000+ RxNorm drugs     │   │
│   │  • Hardcoded in-memory maps        • ELK reasoner hierarchy     │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│   THE SOLUTION (Phase 6 + 7):                                           │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │                         KB-7 Service                            │   │
│   │                              │                                  │   │
│   │              ┌───────────────┼───────────────┐                  │   │
│   │              │               │               │                  │   │
│   │              ▼               ▼               ▼                  │   │
│   │   ┌─────────────────┐ ┌───────────┐ ┌─────────────────┐         │   │
│   │   │    GraphDB      │ │   Redis   │ │     Neo4j       │         │   │
│   │   │   (Master)      │ │ (Hot Cache)│ │ (Read Replica)  │         │   │
│   │   ├─────────────────┤ ├───────────┤ ├─────────────────┤         │   │
│   │   │ • OWL Reasoning │ │ • <1ms    │ │ • Fast Traversal│         │   │
│   │   │ • Write Ops     │ │ • Flink-  │ │ • Cypher Queries│         │   │
│   │   │ • FHIR Compliant│ │   fed     │ │ • 14M+ Triples  │         │   │
│   │   └─────────────────┘ └───────────┘ └─────────────────┘         │   │
│   │           │                               ▲                     │   │
│   │           └───────────── CDC ─────────────┘                     │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Why Phase 6 Was Upgraded to Critical

| Gap | Current State | Impact | Solution |
|-----|---------------|--------|----------|
| **Data Volume** | 45 SNOMED codes hardcoded | 99.99% of clinical concepts missing | Neo4j queries 350K+ concepts |
| **Reasoning** | Hardcoded `IsA` map | Cannot handle rare diseases or complex hierarchies | Cypher uses ELK materialized hierarchy |
| **Updates** | Recompile & redeploy for any change | 2-3 day deployment cycle | CDC provides <1s updates |
| **Scalability** | Memory-bound (in-memory maps) | Cannot scale horizontally | Neo4j read replicas scale independently |

### 6.1 Add Neo4j Driver 🔴

**Priority:** CRITICAL (Upgraded from Nice-to-have)
**Effort:** 1 day
**File:** `internal/semantic/neo4j_client.go`

#### What It Does

Adds the official Neo4j Go driver to KB-7, enabling direct Cypher queries to the read replica.

#### Implementation

```go
// internal/semantic/neo4j_client.go

package semantic

import (
    "context"
    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
    "github.com/sirupsen/logrus"
)

// Neo4jClient provides interface to Neo4j read replica
type Neo4jClient struct {
    driver neo4j.DriverWithContext
    logger *logrus.Logger
}

// NewNeo4jClient creates a new Neo4j client
func NewNeo4jClient(uri, username, password string, logger *logrus.Logger) (*Neo4jClient, error) {
    driver, err := neo4j.NewDriverWithContext(
        uri,
        neo4j.BasicAuth(username, password, ""),
    )
    if err != nil {
        return nil, fmt.Errorf("creating neo4j driver: %w", err)
    }

    return &Neo4jClient{
        driver: driver,
        logger: logger,
    }, nil
}

// GetConcept retrieves a concept by code using Cypher
func (n *Neo4jClient) GetConcept(ctx context.Context, code, system string) (*Concept, error) {
    session := n.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
    defer session.Close(ctx)

    result, err := session.Run(ctx, `
        MATCH (c:Class {code: $code})
        WHERE c.system = $system
        RETURN c.code as code, c.display as display, c.system as system
    `, map[string]interface{}{
        "code":   code,
        "system": system,
    })

    if err != nil {
        return nil, fmt.Errorf("querying concept: %w", err)
    }

    // ... map result to Concept struct
}

// IsSubsumedBy checks if childCode is subsumed by parentCode (using ELK materialized hierarchy)
func (n *Neo4jClient) IsSubsumedBy(ctx context.Context, childCode, parentCode, system string) (bool, error) {
    session := n.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
    defer session.Close(ctx)

    result, err := session.Run(ctx, `
        MATCH (child:Class {code: $childCode})-[:rdfs__subClassOf*]->(parent:Class {code: $parentCode})
        RETURN count(parent) > 0 as isSubsumed
    `, map[string]interface{}{
        "childCode":  childCode,
        "parentCode": parentCode,
    })

    // ... return boolean result
}

// GetDescendants retrieves all descendants of a concept
func (n *Neo4jClient) GetDescendants(ctx context.Context, code, system string, maxDepth int) ([]Concept, error) {
    session := n.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
    defer session.Close(ctx)

    result, err := session.Run(ctx, `
        MATCH (parent:Class {code: $code})<-[:rdfs__subClassOf*1..$maxDepth]-(child:Class)
        RETURN child.code as code, child.display as display, child.system as system
    `, map[string]interface{}{
        "code":     code,
        "maxDepth": maxDepth,
    })

    // ... return slice of Concepts
}
```

#### When to Use Neo4j vs GraphDB

| Use Case | Best Choice | Why |
|----------|-------------|-----|
| OWL reasoning / subsumption | GraphDB | Native OWL 2 support |
| FHIR compliance | GraphDB | RDF-based, standard compliant |
| **Fast graph traversals** | **Neo4j** | **Optimized for traversal queries** |
| **Concept lookups** | **Neo4j** | **<10ms vs 50-200ms** |
| Complex analytics | Neo4j | Better Cypher support |
| Developer-friendly queries | Neo4j | Cypher easier than SPARQL |

### 6.2 CDC Sync (GraphDB → Neo4j) 🔴

**Priority:** CRITICAL (Upgraded from Nice-to-have)
**Effort:** 2-3 days
**Files:** Kafka consumer + Neo4j writer

#### What It Does

Change Data Capture synchronizes GraphDB changes to Neo4j in real-time via Kafka, ensuring the read replica stays consistent with the master.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         CDC SYNCHRONIZATION FLOW                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   1. GraphDB Change Event                                               │
│      ┌─────────────────────────────────────────────────────────────┐    │
│      │  INSERT snomed:12345 rdfs:subClassOf snomed:67890           │    │
│      └─────────────────────────────────────────────────────────────┘    │
│                          │                                              │
│                          ▼                                              │
│   2. Kafka Topic: "kb7.graphdb.changes"                                 │
│      ┌─────────────────────────────────────────────────────────────┐    │
│      │  {                                                          │    │
│      │    "operation": "INSERT",                                   │    │
│      │    "subject": "snomed:12345",                               │    │
│      │    "predicate": "rdfs:subClassOf",                          │    │
│      │    "object": "snomed:67890",                                │    │
│      │    "timestamp": "2025-12-04T10:30:00Z"                      │    │
│      │  }                                                          │    │
│      └─────────────────────────────────────────────────────────────┘    │
│                          │                                              │
│                          ▼                                              │
│   3. Neo4j Consumer (Flink or Go service)                               │
│      ┌─────────────────────────────────────────────────────────────┐    │
│      │  MERGE (s:Class {uri: 'snomed:12345'})                      │    │
│      │  MERGE (o:Class {uri: 'snomed:67890'})                      │    │
│      │  MERGE (s)-[:rdfs__subClassOf]->(o)                         │    │
│      └─────────────────────────────────────────────────────────────┘    │
│                          │                                              │
│                          ▼                                              │
│   4. Neo4j Updated (Latency: < 1 second)                                │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### CDC Consumer Implementation

```go
// internal/cdc/neo4j_consumer.go

type CDCConsumer struct {
    kafkaReader *kafka.Reader
    neo4jClient *Neo4jClient
    logger      *logrus.Logger
}

func (c *CDCConsumer) ProcessChanges(ctx context.Context) error {
    for {
        msg, err := c.kafkaReader.ReadMessage(ctx)
        if err != nil {
            return err
        }

        var change GraphDBChange
        json.Unmarshal(msg.Value, &change)

        switch change.Operation {
        case "INSERT":
            c.applyInsert(ctx, change)
        case "DELETE":
            c.applyDelete(ctx, change)
        }
    }
}
```

---

## Implementation Priority Matrix (Updated - Post Brain/Face Integration)

> **⚠️ MAJOR REVISION**: This matrix reflects the upgraded priority of Neo4j integration
> based on the "Brain vs Face" gap analysis. Phase 6 items have been elevated from P3 to P0/P1.

| Task | Phase | Clinical Impact | Technical Effort | Priority | Timeline | Rationale |
|------|-------|-----------------|------------------|----------|----------|-----------|
| **Neo4j Driver Integration** | 6.1 | 🔴 **CRITICAL** | 🟡 Medium | **P0** | Day 1-2 | Connects API to 14M+ concepts |
| PostgreSQL value_set_definitions | 4.1 | 🔴 Essential | 🟢 Low | **P0** | Day 1 | Rule storage foundation |
| RuleManager Service | 4.2 | 🔴 Essential | 🟡 Medium | **P0** | Day 2-3 | Clinical rules engine |
| **Configure ELK Reasoner** | 5.4 | 🔴 **CRITICAL** | 🟡 Medium | **P0** | Day 3-4 | Prerequisite for subsumption |
| Load SNOMED-CT Subset | 5.1 | 🔴 Essential | 🟡 Medium | **P0** | Day 4-6 | Core terminology data |
| Load ICD-10 → HCC Mappings | 5.2 | 🔴 Essential | 🟢 Low | **P0** | Day 7 | Risk adjustment support |
| **CDC Sync (GraphDB→Neo4j)** | 6.2 | 🔴 **CRITICAL** | 🔴 High | **P1** | Week 2 | Real-time data sync |
| **Bridge: TerminologyService** | 7.1 | 🔴 **CRITICAL** | 🟡 Medium | **P1** | Week 2 | Connect Face to Brain |
| **Bridge: SubsumptionService** | 7.2 | 🔴 **CRITICAL** | 🟡 Medium | **P1** | Week 2 | Use ELK hierarchy |
| Migrate Built-in Value Sets | 4.3 | 🟡 Important | 🟢 Low | **P1** | Week 2 | Production value sets |
| Redis Caching | 4.4 | 🟡 Important | 🟢 Low | **P1** | Week 2 | Performance optimization |
| Load RxNorm Hierarchy | 5.3 | 🟡 Important | 🟡 Medium | **P1** | Week 2 | Drug terminology |
| Rule Loader CLI | 4.5 | 🟢 Nice-to-have | 🟢 Low | **P2** | Week 3+ | Admin tooling |

### Priority Legend

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  P0 (CRITICAL) - Must complete before API is production-ready               │
│  ├── Neo4j Driver: Without this, API returns ~45 codes instead of 14M+     │
│  ├── ELK Reasoner: Without this, subsumption returns false negatives       │
│  └── Core Data: SNOMED-CT, ICD-10 required for clinical operations         │
│                                                                             │
│  P1 (HIGH) - Required for full functionality                                │
│  ├── CDC Sync: Keeps Neo4j in sync with GraphDB master                     │
│  ├── Bridge Services: Refactor Go services to use new Neo4j backend        │
│  └── Supporting Data: RxNorm, value sets, caching                          │
│                                                                             │
│  P2 (MEDIUM) - Operational improvements                                     │
│  └── CLI Tools: Admin and maintenance utilities                            │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase 7: Bridge Implementation (Services Refactoring) 🔗

> **Purpose**: Connect the existing Go API services ("Face") to the Neo4j read replica ("Brain")

### 7.1 TerminologyService Bridge - Priority: **P1 (HIGH)**

**Current State**: TerminologyService queries PostgreSQL for concept lookups

**Target State**: TerminologyService queries Neo4j for concept lookups, falls back to PostgreSQL

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    TerminologyService Refactoring                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  BEFORE (Current):                                                          │
│  ┌──────────────────┐      ┌────────────────┐                              │
│  │ TerminologyService│ ───► │   PostgreSQL   │ (~45 hardcoded concepts)    │
│  └──────────────────┘      └────────────────┘                              │
│                                                                             │
│  AFTER (Target):                                                            │
│  ┌──────────────────┐      ┌────────────────┐                              │
│  │ TerminologyService│ ───► │     Neo4j      │ (14M+ concepts)             │
│  └──────────────────┘      └────────────────┘                              │
│           │                        │                                        │
│           │ (fallback)             │ (cache)                               │
│           ▼                        ▼                                        │
│  ┌────────────────┐         ┌────────────────┐                             │
│  │   PostgreSQL   │         │     Redis      │                             │
│  └────────────────┘         └────────────────┘                             │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Refactored TerminologyService

```go
// internal/services/terminology_service_v2.go

type TerminologyServiceV2 struct {
    neo4jClient *semantic.Neo4jClient   // PRIMARY: 14M+ concepts
    db          *sql.DB                  // FALLBACK: PostgreSQL
    cache       *cache.RedisClient       // CACHE: Hot concepts
    logger      *logrus.Logger
    metrics     *metrics.Collector
}

func NewTerminologyServiceV2(
    neo4j *semantic.Neo4jClient,
    db *sql.DB,
    cache *cache.RedisClient,
    logger *logrus.Logger,
    metrics *metrics.Collector,
) *TerminologyServiceV2 {
    return &TerminologyServiceV2{
        neo4jClient: neo4j,
        db:          db,
        cache:       cache,
        logger:      logger,
        metrics:     metrics,
    }
}

// LookupConcept - Primary path through Neo4j with fallback
func (s *TerminologyServiceV2) LookupConcept(systemIdentifier, code string) (*models.LookupResult, error) {
    start := time.Now()

    // 1. Check Redis cache first
    cacheKey := cache.ConceptCacheKey(systemIdentifier, code)
    var result models.LookupResult
    if err := s.cache.Get(cacheKey, &result); err == nil {
        s.metrics.RecordCacheHit("concept_lookup", "concept")
        return &result, nil
    }
    s.metrics.RecordCacheMiss("concept_lookup", "concept")

    // 2. PRIMARY: Query Neo4j (14M+ concepts)
    if s.neo4jClient != nil {
        concept, err := s.neo4jClient.GetConcept(context.Background(), code, systemIdentifier)
        if err == nil && concept != nil {
            result = s.convertNeo4jConcept(concept)
            s.cache.Set(cacheKey, result, 1*time.Hour)
            s.metrics.RecordConceptLookup(systemIdentifier, "neo4j_hit", time.Since(start))
            return &result, nil
        }
        // Log Neo4j miss but don't fail - try PostgreSQL fallback
        s.logger.WithFields(logrus.Fields{
            "code":   code,
            "system": systemIdentifier,
        }).Debug("Neo4j lookup miss, trying PostgreSQL fallback")
    }

    // 3. FALLBACK: Query PostgreSQL (legacy data)
    result, err := s.lookupFromPostgreSQL(systemIdentifier, code)
    if err != nil {
        s.metrics.RecordConceptLookup(systemIdentifier, "miss", time.Since(start))
        return nil, err
    }

    s.cache.Set(cacheKey, result, 1*time.Hour)
    s.metrics.RecordConceptLookup(systemIdentifier, "postgres_hit", time.Since(start))
    return result, nil
}

func (s *TerminologyServiceV2) convertNeo4jConcept(neo4jConcept *semantic.Concept) models.LookupResult {
    return models.LookupResult{
        Concept: models.TerminologyConcept{
            Code:           neo4jConcept.Code,
            Display:        neo4jConcept.Display,
            Definition:     neo4jConcept.Definition,
            Status:         neo4jConcept.Status,
            ParentCodes:    neo4jConcept.ParentCodes,
            ChildCodes:     neo4jConcept.ChildCodes,
            ClinicalDomain: neo4jConcept.ClinicalDomain,
        },
    }
}
```

### 7.2 SubsumptionService Bridge - Priority: **P1 (HIGH)**

**Current State**: SubsumptionService uses GraphDB SPARQL for real-time reasoning

**Target State**: SubsumptionService uses Neo4j with pre-computed ELK hierarchy (materialized closure)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    SubsumptionService Refactoring                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  BEFORE (Current):                                                          │
│  ┌──────────────────┐      ┌────────────────┐                              │
│  │SubsumptionService│ ───► │    GraphDB     │ (Real-time OWL reasoning)    │
│  └──────────────────┘      │    SPARQL      │ (~100-500ms per query)       │
│                            └────────────────┘                              │
│                                                                             │
│  AFTER (Target):                                                            │
│  ┌──────────────────┐      ┌────────────────┐                              │
│  │SubsumptionService│ ───► │     Neo4j      │ (Pre-computed ELK closure)   │
│  └──────────────────┘      │    Cypher      │ (~1-10ms per query)          │
│                            └────────────────┘                              │
│           │                                                                 │
│           │ (complex reasoning fallback)                                    │
│           ▼                                                                 │
│  ┌────────────────┐                                                         │
│  │    GraphDB     │ (Only for queries requiring real-time OWL reasoning)   │
│  └────────────────┘                                                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Refactored SubsumptionService

```go
// internal/services/subsumption_service_v2.go

type SubsumptionServiceV2 struct {
    neo4jClient   *semantic.Neo4jClient   // PRIMARY: Pre-computed ELK hierarchy
    graphDBClient *semantic.GraphDBClient // FALLBACK: Complex OWL reasoning
    cache         *cache.RedisClient
    logger        *logrus.Logger
}

func NewSubsumptionServiceV2(
    neo4j *semantic.Neo4jClient,
    graphDB *semantic.GraphDBClient,
    cache *cache.RedisClient,
    logger *logrus.Logger,
) *SubsumptionServiceV2 {
    return &SubsumptionServiceV2{
        neo4jClient:   neo4j,
        graphDBClient: graphDB,
        cache:         cache,
        logger:        logger,
    }
}

// IsSubsumedBy - Uses Neo4j pre-computed closure for O(1) lookup
func (s *SubsumptionServiceV2) IsSubsumedBy(ctx context.Context, childCode, parentCode, system string) (bool, error) {
    // 1. Check cache first
    cacheKey := fmt.Sprintf("subsumption:%s:%s:%s", system, childCode, parentCode)
    var result bool
    if err := s.cache.Get(cacheKey, &result); err == nil {
        return result, nil
    }

    // 2. PRIMARY: Query Neo4j materialized ELK closure
    //    The ELK reasoner pre-computes all transitive subClassOf relationships
    //    Neo4j stores these as direct edges for O(1) path lookup
    if s.neo4jClient != nil {
        isSubsumed, err := s.neo4jClient.IsSubsumedBy(ctx, childCode, parentCode, system)
        if err == nil {
            s.cache.Set(cacheKey, isSubsumed, 24*time.Hour) // Long TTL - hierarchy is stable
            return isSubsumed, nil
        }
        s.logger.WithError(err).Warn("Neo4j subsumption check failed, falling back to GraphDB")
    }

    // 3. FALLBACK: GraphDB SPARQL for complex reasoning
    //    Only used when Neo4j is unavailable or for queries requiring
    //    real-time OWL reasoning (e.g., property restrictions)
    return s.checkSubsumptionViaGraphDB(ctx, childCode, parentCode, system)
}

// GetAncestors - Returns all ancestors using Neo4j transitive closure
func (s *SubsumptionServiceV2) GetAncestors(ctx context.Context, code, system string) ([]string, error) {
    if s.neo4jClient == nil {
        return nil, fmt.Errorf("neo4j client not available")
    }

    // Neo4j Cypher query against ELK-materialized closure
    // This is a simple path traversal, not real-time reasoning
    ancestors, err := s.neo4jClient.GetAncestors(ctx, code, system)
    if err != nil {
        return nil, err
    }

    var codes []string
    for _, a := range ancestors {
        codes = append(codes, a.Code)
    }
    return codes, nil
}

func (s *SubsumptionServiceV2) checkSubsumptionViaGraphDB(ctx context.Context, childCode, parentCode, system string) (bool, error) {
    // Fallback SPARQL query for complex OWL reasoning
    query := &semantic.SPARQLQuery{
        Query: fmt.Sprintf(`
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
            PREFIX snomed: <http://snomed.info/id/>

            ASK {
                snomed:%s rdfs:subClassOf+ snomed:%s .
            }
        `, childCode, parentCode),
    }

    results, err := s.graphDBClient.ExecuteSPARQL(ctx, query)
    if err != nil {
        return false, err
    }

    return results.GetBooleanResult(), nil
}
```

### 7.3 ValueSetService Bridge - Priority: **P1 (HIGH)**

**Current State**: ValueSetService uses PostgreSQL with 18 built-in value sets

**Target State**: ValueSetService uses Redis hot cache populated by Flink from GraphDB

```go
// internal/services/value_set_service_v2.go

type ValueSetServiceV2 struct {
    redis      *cache.RedisClient       // PRIMARY: Hot cache from Flink
    db         *sql.DB                  // FALLBACK: PostgreSQL
    neo4j      *semantic.Neo4jClient   // EXPANSION: For intensional value sets
    logger     *logrus.Logger
}

// ExpandValueSet - Expands intensional value sets using Neo4j hierarchy
func (s *ValueSetServiceV2) ExpandValueSet(ctx context.Context, url string) (*models.ValueSetExpansion, error) {
    // 1. Check Redis hot cache (populated by Flink from GraphDB)
    cacheKey := fmt.Sprintf("valueset:expansion:%s", url)
    var expansion models.ValueSetExpansion
    if err := s.redis.Get(cacheKey, &expansion); err == nil {
        return &expansion, nil
    }

    // 2. Get value set definition
    valueSet, err := s.GetValueSet(url, "")
    if err != nil {
        return nil, err
    }

    // 3. Expand intensional definitions using Neo4j
    //    For example: "all descendants of SNOMED 73211009 (Diabetes mellitus)"
    if s.neo4j != nil && valueSet.Compose != nil {
        expansion, err = s.expandIntensional(ctx, valueSet)
        if err == nil {
            s.redis.Set(cacheKey, expansion, 1*time.Hour)
            return &expansion, nil
        }
    }

    // 4. Fallback to extensional expansion from PostgreSQL
    return s.expandExtensional(ctx, valueSet)
}

func (s *ValueSetServiceV2) expandIntensional(ctx context.Context, vs *models.ValueSet) (models.ValueSetExpansion, error) {
    var expansion models.ValueSetExpansion

    // Parse compose rules (e.g., include all descendants of X)
    // Use Neo4j GetDescendants for efficient hierarchy traversal
    for _, include := range vs.Compose.Include {
        for _, filter := range include.Filter {
            if filter.Property == "concept" && filter.Op == "is-a" {
                // Get all descendants from Neo4j ELK-materialized hierarchy
                descendants, err := s.neo4j.GetDescendants(ctx, filter.Value, include.System, 10)
                if err != nil {
                    return expansion, err
                }

                for _, d := range descendants {
                    expansion.Contains = append(expansion.Contains, models.ValueSetContains{
                        System:  include.System,
                        Code:    d.Code,
                        Display: d.Display,
                    })
                }
            }
        }
    }

    expansion.Total = len(expansion.Contains)
    expansion.Timestamp = time.Now()
    return expansion, nil
}
```

### 7.4 Server Initialization Update - Priority: **P1 (HIGH)**

```go
// cmd/server/main.go - Updated initialization

func main() {
    // ... existing config loading ...

    // Initialize Neo4j client (NEW - PRIMARY DATA SOURCE)
    var neo4jClient *semantic.Neo4jClient
    if cfg.Neo4jEnabled {
        var err error
        neo4jClient, err = semantic.NewNeo4jClient(
            cfg.Neo4jURI,
            cfg.Neo4jUsername,
            cfg.Neo4jPassword,
            logger,
        )
        if err != nil {
            logger.WithError(err).Fatal("Failed to connect to Neo4j - this is now CRITICAL")
        }
        defer neo4jClient.Close()

        logger.WithFields(logrus.Fields{
            "uri": cfg.Neo4jURI,
        }).Info("Connected to Neo4j (Brain) - 14M+ concepts available")
    }

    // Initialize services with Neo4j as PRIMARY
    terminologyService := services.NewTerminologyServiceV2(
        neo4jClient,  // PRIMARY: Neo4j with 14M+ concepts
        db,           // FALLBACK: PostgreSQL
        redisClient,
        logger,
        metricsCollector,
    )

    subsumptionService := services.NewSubsumptionServiceV2(
        neo4jClient,   // PRIMARY: ELK-materialized hierarchy
        graphDBClient, // FALLBACK: Complex OWL reasoning
        redisClient,
        logger,
    )

    valueSetService := services.NewValueSetServiceV2(
        redisClient,  // PRIMARY: Hot cache from Flink
        db,           // FALLBACK: PostgreSQL
        neo4jClient,  // EXPANSION: Intensional value sets
        logger,
    )

    // Start CDC consumer for real-time sync (Background)
    if cfg.CDCEnabled {
        cdcConsumer := cdc.NewCDCConsumer(cfg.KafkaBrokers, neo4jClient, logger)
        go cdcConsumer.ProcessChanges(context.Background())
        logger.Info("CDC Consumer started - GraphDB changes will sync to Neo4j in <1s")
    }

    // ... rest of server initialization ...
}
```

---

## Execution Roadmap (Revised)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     KB-7 Implementation Timeline (Revised)                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Week 1: Foundation + Neo4j Integration (P0 Critical)                       │
│  ├── Day 1-2: Neo4j Driver (6.1) + PostgreSQL schema (4.1)                 │
│  ├── Day 3-4: ELK Reasoner configuration (5.4)                             │
│  ├── Day 5-6: Load SNOMED-CT subset (5.1)                                  │
│  └── Day 7: ICD-10 → HCC mappings (5.2) + RuleManager (4.2)                │
│                                                                             │
│  Week 2: Bridge Implementation (P1 High)                                    │
│  ├── Day 8-9: TerminologyService V2 with Neo4j primary (7.1)               │
│  ├── Day 10-11: SubsumptionService V2 with ELK closure (7.2)               │
│  ├── Day 12-13: CDC Sync pipeline (6.2)                                    │
│  └── Day 14: ValueSetService V2 + Redis caching (7.3, 4.4)                 │
│                                                                             │
│  Week 3: Production Hardening (P1-P2)                                       │
│  ├── Load RxNorm hierarchy (5.3)                                           │
│  ├── Migrate built-in value sets (4.3)                                     │
│  ├── Rule Loader CLI (4.5)                                                 │
│  └── Performance testing & optimization                                    │
│                                                                             │
│  Milestone: API returns 14M+ concepts instead of ~45                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Document Control

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-12-04 | Claude Code | Initial gap analysis |
| 2.0 | 2025-12-04 | Claude Code | Added detailed Phase 4-6 implementation roadmap |
| 3.0 | 2025-12-04 | Claude Code | **MAJOR**: Integrated "Brain vs Face" analysis, upgraded Phase 6 to CRITICAL, added Phase 7 Bridge Implementation |

---

## Appendix: Brain vs Face Summary

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    THE PROBLEM WE SOLVED                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  BEFORE (Disconnected):                                                     │
│                                                                             │
│  ┌─────────────┐          ┌──────────────┐                                 │
│  │  Go API     │          │ Graph        │                                 │
│  │  (Face)     │    ✗     │ Pipeline     │                                 │
│  │  ~45 codes  │          │ (Brain)      │                                 │
│  │  hardcoded  │          │ 14M+ triples │                                 │
│  └─────────────┘          └──────────────┘                                 │
│                                                                             │
│  AFTER (Connected via Phase 6 + 7):                                         │
│                                                                             │
│  ┌─────────────┐          ┌──────────────┐          ┌──────────────┐       │
│  │  Go API     │ ◄─────── │    Neo4j     │ ◄─CDC──  │   GraphDB    │       │
│  │  (Face)     │  Cypher  │    (Fast)    │  <1sec   │   (Master)   │       │
│  │  14M+ codes │          │  Read Replica│          │ OWL Reasoning│       │
│  └─────────────┘          └──────────────┘          └──────────────┘       │
│        │                         │                                          │
│        ▼                         ▼                                          │
│  ┌─────────────┐          ┌──────────────┐                                 │
│  │   Redis     │          │ ELK Reasoner │                                 │
│  │  Hot Cache  │          │  Hierarchy   │                                 │
│  └─────────────┘          └──────────────┘                                 │
│                                                                             │
│  Result: API can now access full clinical terminology corpus                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

