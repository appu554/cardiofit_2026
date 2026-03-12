# KB-7 Terminology Service: Core Components Documentation

## Overview

This document provides comprehensive API documentation for the four core components of the KB-7 Terminology Service CDSS (Clinical Decision Support System) architecture:

1. **Value Set Loader** - Load and manage Value Set definitions in PostgreSQL
2. **CDSS Evaluation Endpoint** - Process patient data against clinical rules
3. **Rule Engine** - Evaluate clinical protocols using the THREE-CHECK PIPELINE
4. **Fact Builder** - Convert patient clinical data to structured facts

---

## Component 1: Value Set Loader

### Purpose

The Value Set Loader is responsible for importing Value Set definitions into PostgreSQL, making them available for the THREE-CHECK PIPELINE. It bridges static JSON/Go definitions with the runtime query system.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         VALUE SET LOADER                                     │
│                                                                              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐   │
│  │   JSON/Go   │    │   Parser    │    │  Validator  │    │   Storage   │   │
│  │   Files     │───▶│   Engine    │───▶│   Engine    │───▶│   Writer    │   │
│  │             │    │             │    │             │    │             │   │
│  └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘   │
│        │                  │                  │                  │           │
│        ▼                  ▼                  ▼                  ▼           │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐   │
│  │  File       │    │  FHIR R4    │    │  Business   │    │ PostgreSQL  │   │
│  │  System     │    │  Schema     │    │  Rules      │    │ + Redis     │   │
│  └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### API Endpoints

#### 1. Seed Builtin Value Sets

Seeds the 49 predefined clinical value sets from Go code into PostgreSQL.

```http
POST /v1/rules/seed
Content-Type: application/json
```

**Response (200 OK):**
```json
{
  "message": "Builtin value sets seeded successfully",
  "seeded_count": 49,
  "categories": {
    "FHIR_R4": 18,
    "AU_Clinical": 6,
    "KB7_Clinical": 25
  },
  "duration_ms": 1234
}
```

**CLI Alternative:**
```bash
# Enable seeding on server startup
SEED_BUILTIN_VALUE_SETS=true go run ./cmd/server
```

#### 2. List Value Sets

Returns all loaded value set definitions with optional filtering.

```http
GET /v1/rules/valuesets
GET /v1/rules/valuesets?category=sepsis&status=active&limit=50
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `category` | string | Filter by clinical domain (e.g., "sepsis", "renal", "cardiac") |
| `status` | string | Filter by status ("active", "draft", "retired") |
| `definition_type` | string | Filter by type ("explicit", "intensional", "extensional") |
| `limit` | int | Maximum results to return (default: 100) |
| `offset` | int | Pagination offset |

**Response (200 OK):**
```json
{
  "value_sets": [
    {
      "id": "SepsisDiagnosis",
      "name": "SepsisDiagnosis",
      "url": "http://cardiofit.ai/fhir/ValueSet/sepsis-diagnosis",
      "title": "Sepsis Diagnosis Codes",
      "definition_type": "explicit",
      "version": "1.0.0",
      "status": "active",
      "clinical_domain": "infectious-disease",
      "code_count": 34
    }
  ],
  "total": 49,
  "limit": 100,
  "offset": 0
}
```

#### 3. Get Value Set Definition

Returns the raw definition for a specific value set (not expanded).

```http
GET /v1/rules/valuesets/:identifier
GET /v1/rules/valuesets/SepsisDiagnosis
```

**Response (200 OK):**
```json
{
  "id": "SepsisDiagnosis",
  "name": "SepsisDiagnosis",
  "url": "http://cardiofit.ai/fhir/ValueSet/sepsis-diagnosis",
  "title": "Sepsis Diagnosis Codes",
  "description": "SNOMED CT codes representing sepsis conditions",
  "publisher": "CardioFit Clinical Intelligence",
  "definition_type": "explicit",
  "version": "1.0.0",
  "status": "active",
  "clinical_domain": "infectious-disease",
  "explicit_codes": [
    {
      "system": "http://snomed.info/sct",
      "code": "91302008",
      "display": "Sepsis"
    },
    {
      "system": "http://snomed.info/sct",
      "code": "10001005",
      "display": "Bacterial sepsis"
    }
  ],
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z"
}
```

### Definition Types

| Type | Description | Storage Strategy |
|------|-------------|------------------|
| **EXPLICIT** | Pre-computed list of codes | Codes stored in `value_set_codes` table |
| **INTENSIONAL** | Rule-based expansion (e.g., "all descendants of Diabetes") | Root concept + expansion rule stored; expanded at query time via Neo4j |
| **EXTENSIONAL** | Composed from other value sets | References stored; recursively expanded at query time |

### Value Set Categories

The loader supports 49 predefined value sets across 3 categories:

**FHIR R4 Standard (18 value sets):**
- AdministrativeGender, Observation Status, Condition Clinical Status
- Medication Status, AllergyIntolerance Clinical Status
- DiagnosticReport Status, and more...

**Australian Clinical (6 value sets):**
- AUAKIConditions, AUSepsisConditions
- AUCardiacConditions, AURespiratoryConditions
- AUPBSMedicationCategories, AULoincCommonTests

**KB7 Clinical (25 value sets):**
- SepsisDiagnosis, AcuteRenalFailure, DiabetesMellitus
- Hypertension, HeartFailure, RespiratoryFailure
- SepsisLabIndicators, RenalLabIndicators
- ACEInhibitors, ARBs, Diuretics, and more...

---

## Component 2: CDSS Evaluation Endpoint

### Purpose

The CDSS Evaluate endpoint is the primary integration point between clinical systems and KB7. It receives patient data, converts it to facts, evaluates against clinical rules, and returns actionable alerts.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      CDSS EVALUATE ENDPOINT                                  │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     Request Processing                               │    │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐         │    │
│  │  │  Parse    │  │ Validate  │  │ Normalize │  │ Enrich    │         │    │
│  │  │  Request  │─▶│  Input    │─▶│  Codes    │─▶│  Context  │         │    │
│  │  └───────────┘  └───────────┘  └───────────┘  └───────────┘         │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│                                    ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     THREE-CHECK PIPELINE                             │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  │    │
│  │  │  STEP 1:    │  │  STEP 2:    │  │  STEP 3:    │                  │    │
│  │  │  Expansion  │─▶│ Exact Match │─▶│ Subsumption │                  │    │
│  │  │             │  │  (O(1))     │  │  (Neo4j)    │                  │    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│                                    ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     Response Building                                │    │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐         │    │
│  │  │  Match    │  │    HCC    │  │Terminology│  │  Metrics  │         │    │
│  │  │  Result   │  │  Mapper   │  │  Enricher │  │  Recorder │         │    │
│  │  └───────────┘  └───────────┘  └───────────┘  └───────────┘         │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

### API Endpoints

#### 1. Validate Code in Value Set

Validates if a code belongs to a specific value set using the THREE-CHECK PIPELINE.

```http
POST /v1/rules/valuesets/:identifier/validate
Content-Type: application/json
```

**Request Body:**
```json
{
  "code": "91302008",
  "system": "http://snomed.info/sct"
}
```

**Response (200 OK) - Exact Match:**
```json
{
  "valid": true,
  "value_set_id": "SepsisDiagnosis",
  "code": "91302008",
  "system": "http://snomed.info/sct",
  "display": "Sepsis",
  "match_type": "exact",
  "matched_code": "91302008",
  "message": "Code found in value set via exact membership match (O(1) hash lookup)",
  "pipeline": {
    "step1_expansion": {
      "status": "completed",
      "codes_count": 34,
      "cached": true,
      "duration": "1.2ms"
    },
    "step2_exact_match": {
      "status": "match",
      "checked": true,
      "match_found": true,
      "checked_code": "91302008"
    },
    "step3_subsumption": {
      "status": "skipped",
      "checked": false,
      "match_found": false
    }
  }
}
```

**Response (200 OK) - Subsumption Match:**
```json
{
  "valid": true,
  "value_set_id": "SepsisDiagnosis",
  "code": "127081009",
  "system": "http://snomed.info/sct",
  "display": "Gram-negative sepsis",
  "match_type": "subsumption",
  "matched_code": "91302008",
  "message": "Code matched via hierarchical subsumption (IS-A relationship)",
  "pipeline": {
    "step1_expansion": {
      "status": "completed",
      "codes_count": 34,
      "cached": true,
      "duration": "0.8ms"
    },
    "step2_exact_match": {
      "status": "no_match",
      "checked": true,
      "match_found": false,
      "checked_code": "127081009"
    },
    "step3_subsumption": {
      "status": "match",
      "checked": true,
      "match_found": true,
      "checked_code": "127081009",
      "matched_ancestor": "91302008",
      "ancestor_display": "Sepsis",
      "path_length": 1,
      "source": "neo4j",
      "codes_checked": 34
    }
  }
}
```

#### 2. Classify Code (FindValueSetsForCode)

Finds ALL value sets that contain a given code - reverse lookup feature.

```http
POST /v1/rules/classify
Content-Type: application/json
```

**Request Body:**
```json
{
  "code": "91302008",
  "system": "http://snomed.info/sct"
}
```

**Response (200 OK):**
```json
{
  "code": "91302008",
  "system": "http://snomed.info/sct",
  "display": "Sepsis",
  "matching_valuesets": [
    {
      "valueset_id": "SepsisDiagnosis",
      "valueset_title": "Sepsis Diagnosis Codes",
      "match_type": "exact",
      "clinical_domain": "infectious-disease"
    },
    {
      "valueset_id": "AUSepsisConditions",
      "valueset_title": "Australian Sepsis Conditions",
      "match_type": "exact",
      "clinical_domain": "infectious-disease"
    }
  ],
  "non_matching_valuesets": 47,
  "total_valuesets_checked": 49,
  "evaluation_time_ms": 45.3
}
```

#### 3. Expand Value Set

Returns all codes in a value set (fully expanded).

```http
POST /v1/rules/valuesets/:identifier/expand
Content-Type: application/json
```

**Request Body (optional):**
```json
{
  "version": "1.0.0",
  "include_inactive": false,
  "limit": 1000
}
```

**Response (200 OK):**
```json
{
  "identifier": "SepsisDiagnosis",
  "url": "http://cardiofit.ai/fhir/ValueSet/sepsis-diagnosis",
  "version": "1.0.0",
  "total": 34,
  "codes": [
    {
      "system": "http://snomed.info/sct",
      "code": "91302008",
      "display": "Sepsis",
      "version": ""
    },
    {
      "system": "http://snomed.info/sct",
      "code": "10001005",
      "display": "Bacterial sepsis",
      "version": ""
    }
  ],
  "expansion_time": "2025-12-10T10:30:00Z",
  "cached_result": false
}
```

---

## Component 3: Rule Engine (THREE-CHECK PIPELINE)

### Purpose

The Rule Engine evaluates clinical protocols and guidelines against patient facts using the THREE-CHECK PIPELINE: **Expansion → Exact Match → Subsumption**.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    THREE-CHECK PIPELINE                                      │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  STEP 1: EXPANSION                                                   │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  │    │
│  │  │   Redis     │  │  PostgreSQL │  │   Neo4j     │                  │    │
│  │  │   Cache     │─▶│   Database  │─▶│   (Graph)   │                  │    │
│  │  │   ~1ms      │  │   ~5ms      │  │   ~10ms     │                  │    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│                                    ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  STEP 2: EXACT MATCH (O(1) Hash Lookup)                             │    │
│  │                                                                      │    │
│  │  expanded.Contains(system, code)  →  O(1) hash lookup               │    │
│  │  CodeIndex[system][code] = *ExpandedCode                            │    │
│  │                                                                      │    │
│  │  If MATCH: Return immediately (skip Step 3)                          │    │
│  │  If NO MATCH: Proceed to Step 3                                      │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│                                    ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  STEP 3: SUBSUMPTION (Neo4j shortestPath)                           │    │
│  │                                                                      │    │
│  │  For each code in expanded value set:                                │    │
│  │    MATCH (child), (parent)                                           │    │
│  │    MATCH path = shortestPath((child)-[:subClassOf*1..15]->(parent)) │    │
│  │    RETURN length(path) as pathLength                                 │    │
│  │                                                                      │    │
│  │  If pathLength > 0: IS-A relationship exists → MATCH                │    │
│  │  If no path: NO MATCH                                                │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Pipeline Execution

```go
// Step 1: Expansion
expanded, err := ruleManager.ExpandValueSet(ctx, valueSetID, version)
// Returns: ExpandedValueSet with codes + O(1) hash index

// Step 2: Exact Match (O(1))
if ec, found := expanded.Contains(system, code); found {
    return &RuleValidationResult{
        Valid:     true,
        MatchType: MatchTypeExact,
        Message:   "Code found via O(1) hash lookup",
    }
}

// Step 3: Subsumption (Neo4j BFS)
for _, ec := range expanded.Codes {
    result, err := neo4jBridge.TestSubsumption(ctx, inputCode, ec.Code, system)
    if result.IsSubsumed {
        return &RuleValidationResult{
            Valid:       true,
            MatchType:   MatchTypeSubsumption,
            MatchedCode: ec.Code,
            PathLength:  result.PathLength,
        }
    }
}
```

### API Endpoints

#### 1. Test Subsumption

Tests if one concept is a subtype (IS-A) of another.

```http
POST /v1/subsumption/test
Content-Type: application/json
```

**Request Body:**
```json
{
  "code_a": "127081009",
  "code_b": "91302008",
  "system": "http://snomed.info/sct"
}
```

**Response (200 OK):**
```json
{
  "subsumes": true,
  "relationship": "subsumed",
  "code_a": "127081009",
  "code_b": "91302008",
  "system": "http://snomed.info/sct",
  "path_length": 1,
  "intermediate_nodes": [],
  "reasoning_type": "neo4j",
  "execution_time_ms": 12.5,
  "cached_result": false,
  "tested_at": "2025-12-10T10:30:00Z"
}
```

#### 2. Get Ancestors

Retrieves all ancestors of a concept up to a maximum depth.

```http
POST /v1/subsumption/ancestors
Content-Type: application/json
```

**Request Body:**
```json
{
  "code": "127081009",
  "system": "http://snomed.info/sct",
  "max_depth": 10
}
```

**Response (200 OK):**
```json
{
  "code": "127081009",
  "system": "http://snomed.info/sct",
  "display": "Gram-negative sepsis",
  "ancestors": [
    {
      "code": "91302008",
      "display": "Sepsis",
      "system": "http://snomed.info/sct",
      "depth": 1
    },
    {
      "code": "40733004",
      "display": "Infectious disease",
      "system": "http://snomed.info/sct",
      "depth": 2
    }
  ],
  "total_ancestors": 15,
  "max_depth_reached": false
}
```

#### 3. Get Descendants

Retrieves all descendants of a concept.

```http
POST /v1/subsumption/descendants
Content-Type: application/json
```

**Request Body:**
```json
{
  "code": "91302008",
  "system": "http://snomed.info/sct",
  "max_depth": 5
}
```

**Response (200 OK):**
```json
{
  "code": "91302008",
  "system": "http://snomed.info/sct",
  "display": "Sepsis",
  "descendants": [
    {
      "code": "10001005",
      "display": "Bacterial sepsis",
      "system": "http://snomed.info/sct",
      "depth": 1
    },
    {
      "code": "127081009",
      "display": "Gram-negative sepsis",
      "system": "http://snomed.info/sct",
      "depth": 2
    }
  ],
  "total_descendants": 150,
  "max_depth_reached": true
}
```

### Match Types

| Match Type | Description | Performance |
|------------|-------------|-------------|
| **exact** | Code directly listed in value set | O(1) hash lookup ~0.05ms |
| **subsumption** | Code is a descendant (IS-A) of a value set code | Neo4j BFS ~10ms |
| **none** | Code not found via any method | Full pipeline cost |

---

## Component 4: Fact Builder (O(1) Hash Map Optimization)

### Purpose

The Fact Builder converts raw patient clinical data into structured facts for rule evaluation. The O(1) hash map optimization provides constant-time exact match lookup regardless of value set size.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    O(1) HASH MAP STRUCTURE                                   │
│                                                                              │
│  ExpandedValueSet {                                                          │
│    Codes: []ExpandedCode         // Original list for iteration             │
│    CodeIndex: map[string]        // O(1) lookup index                       │
│               map[string]                                                    │
│               *ExpandedCode                                                  │
│  }                                                                           │
│                                                                              │
│  Two-Level Map:                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Level 1: System URI                                                 │    │
│  │  "http://snomed.info/sct" ─┬─▶ Level 2: Code Map                    │    │
│  │  "http://loinc.org"       ─┤   "91302008" ─▶ *ExpandedCode          │    │
│  │  "http://hl7.org/fhir"    ─┘   "10001005" ─▶ *ExpandedCode          │    │
│  │                                 "127081009" ─▶ *ExpandedCode         │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  Lookup: O(1) via expanded.Contains(system, code)                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Data Structures

```go
// ExpandedValueSet with O(1) hash index
type ExpandedValueSet struct {
    Identifier    string         `json:"identifier"`
    URL           string         `json:"url"`
    Version       string         `json:"version"`
    Total         int            `json:"total"`
    Codes         []ExpandedCode `json:"codes"`
    ExpansionTime time.Time      `json:"expansion_time"`
    CachedResult  bool           `json:"cached_result"`

    // O(1) lookup index (not serialized to JSON)
    CodeIndex map[string]map[string]*ExpandedCode `json:"-"`
}

// BuildIndex constructs the O(1) hash index
func (e *ExpandedValueSet) BuildIndex() {
    e.CodeIndex = make(map[string]map[string]*ExpandedCode)
    for i := range e.Codes {
        code := &e.Codes[i]
        system := code.System
        if system == "" {
            system = "_default_"
        }
        if e.CodeIndex[system] == nil {
            e.CodeIndex[system] = make(map[string]*ExpandedCode)
        }
        e.CodeIndex[system][code.Code] = code
    }
}

// Contains performs O(1) lookup
func (e *ExpandedValueSet) Contains(system, code string) (*ExpandedCode, bool) {
    if e.CodeIndex == nil {
        e.BuildIndex()
    }

    if system != "" {
        if systemMap, ok := e.CodeIndex[system]; ok {
            if ec, found := systemMap[code]; found {
                return ec, true
            }
        }
        return nil, false
    }

    // Search across all systems if system not specified
    for _, systemMap := range e.CodeIndex {
        if ec, found := systemMap[code]; found {
            return ec, true
        }
    }
    return nil, false
}
```

### Performance Comparison

| Operation | Before (O(n) Loop) | After (O(1) Hash) | Improvement |
|-----------|-------------------|-------------------|-------------|
| Single lookup (100 codes) | ~500ns | ~50ns | **10x** |
| Single lookup (1000 codes) | ~5µs | ~50ns | **100x** |
| Batch lookup (10 codes, 500 code VS) | ~25µs | ~500ns | **50x** |
| Memory per Value Set | baseline | +24KB | - |

### Cache Metrics API

Monitor the multi-layer cache performance:

```http
GET /v1/cache/metrics
```

**Response (200 OK):**
```json
{
  "cache_architecture": {
    "description": "Multi-layer caching for clinical terminology validation",
    "layers": [
      {"layer": "L0", "name": "Bloom Filter", "latency": "~0.001ms"},
      {"layer": "L1", "name": "Hot Sets", "latency": "~0.01ms"},
      {"layer": "L2", "name": "Local Cache", "latency": "~0.1ms"},
      {"layer": "L2.5", "name": "Redis", "latency": "~1ms"},
      {"layer": "L3", "name": "Neo4j", "latency": "~5-10ms"}
    ]
  },
  "terminology_bridge": {
    "bloom_filters_loaded": 5,
    "hot_sets_loaded": 5,
    "hot_codes_total": 130,
    "local_cache_size": 0,
    "local_cache_max": 100000,
    "redis_enabled": true,
    "subsumption_enabled": true
  },
  "neo4j_bridge": {
    "neo4j_available": true,
    "graphdb_available": true,
    "neo4j_queries": 287,
    "neo4j_success_rate": 100,
    "cache_hit_rate": 53.02,
    "avg_latency_ms": 0.51
  }
}
```

### Cache Management

#### Refresh Cache

Force re-expansion of a value set:

```http
POST /v1/rules/valuesets/:identifier/refresh
```

**Response (200 OK):**
```json
{
  "message": "Cache refreshed successfully",
  "value_set_id": "SepsisDiagnosis",
  "new_code_count": 34,
  "refresh_duration_ms": 45
}
```

#### Invalidate Specific Cache

```http
POST /v1/cache/invalidate/:valueSetID
```

#### Refresh All Caches

```http
POST /v1/cache/refresh
```

---

## System Flow: How Components Connect

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        SYSTEM FLOW                                           │
│                                                                              │
│  1. VALUE SET LOADER (Startup/Update)                                        │
│     ├── Reads JSON/Go definitions                                            │
│     ├── Validates against schema                                             │
│     ├── Stores in PostgreSQL                                                 │
│     ├── Builds hash map indexes ◄─────────────────┐                          │
│     └── Warms Redis cache                         │                          │
│                                                    │                          │
│  2. CDSS EVALUATE ENDPOINT (Request Time)         │                          │
│     ├── Receives patient data                     │                          │
│     ├── Builds facts from codes                   │                          │
│     │   └── Uses THREE-CHECK PIPELINE             │                          │
│     │       ├── Step 1: Get expanded Value Set    │                          │
│     │       ├── Step 2: O(1) HASH MAP LOOKUP ─────┘                          │
│     │       └── Step 3: Neo4j subsumption (if needed)                        │
│     ├── Populates Working Memory                                             │
│     └── Returns validation result                                            │
│                                                                              │
│  3. RULE ENGINE (THREE-CHECK PIPELINE)                                       │
│     ├── Expansion → Exact Match → Subsumption                                │
│     ├── Uses O(1) hash index for Step 2                                      │
│     ├── Uses Neo4j shortestPath for Step 3                                   │
│     └── Returns match result with evidence                                   │
│                                                                              │
│  4. O(1) HASH MAP (Optimization Layer)                                       │
│     └── Provides instant exact match for Step 2 ◄────────────────────────────┘
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Quick Reference: Common Operations

### Validate a SNOMED Code

```bash
# Check if code is in SepsisDiagnosis value set
curl -X POST http://localhost:8087/v1/rules/valuesets/SepsisDiagnosis/validate \
  -H "Content-Type: application/json" \
  -d '{"code":"91302008","system":"http://snomed.info/sct"}'
```

### Find All Value Sets for a Code

```bash
# Classify code across all value sets
curl -X POST http://localhost:8087/v1/rules/classify \
  -H "Content-Type: application/json" \
  -d '{"code":"91302008","system":"http://snomed.info/sct"}'
```

### Expand a Value Set

```bash
# Get all codes in a value set
curl -X POST http://localhost:8087/v1/rules/valuesets/SepsisDiagnosis/expand \
  -H "Content-Type: application/json" \
  -d '{}'
```

### Test Subsumption

```bash
# Check if code_a IS-A code_b
curl -X POST http://localhost:8087/v1/subsumption/test \
  -H "Content-Type: application/json" \
  -d '{"code_a":"127081009","code_b":"91302008","system":"http://snomed.info/sct"}'
```

### Seed Value Sets (First Run)

```bash
# Seed all 49 builtin value sets to PostgreSQL
curl -X POST http://localhost:8087/v1/rules/seed
```

---

## Implementation Status

| Component | Status | Notes |
|-----------|--------|-------|
| Value Set Loader | ✅ Complete | 49 value sets seeded, idempotent loading |
| CDSS Evaluate Endpoint | ✅ Complete | THREE-CHECK PIPELINE fully operational |
| Rule Engine | ✅ Complete | O(1) hash + shortestPath + compound conditions |
| Fact Builder | ✅ Complete | FHIR R4 parsing, fact extraction, normalization |
| Alert Generator | ✅ Complete | Domain grouping, severity prioritization |
| Full CDSS Pipeline | ✅ Complete | Facts → Evaluation → Rules → Alerts |

---

## Component 5: Full CDSS Patient Evaluation Pipeline

### Purpose

The Full CDSS Pipeline provides end-to-end clinical decision support by evaluating patient data (FHIR resources) against clinical rules and generating actionable alerts. This is the primary integration point for clinical systems.

### Complete Pipeline Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        POST /v1/cdss/evaluate                                │
│                                                                              │
│  Input: FHIR Bundle OR Individual Resources (Conditions, Observations, etc.)│
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 1: FactBuilder                                                         │
│  ─────────────────────                                                       │
│  • Parses FHIR resources (Condition, Observation, MedicationRequest, etc.)  │
│  • Extracts clinical codes (SNOMED-CT, LOINC, ICD-10, RxNorm)               │
│  • Creates normalized ClinicalFact objects                                   │
│  • Builds PatientFactSet with categorized facts                             │
│                                                                              │
│  Output: PatientFactSet { Conditions[], Observations[], Medications[], ... } │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 2: THREE-CHECK PIPELINE (via RuleManager.ClassifyCode)                │
│  ─────────────────────────────────────────────────────────────              │
│  For each fact, checks membership in clinical value sets:                   │
│                                                                              │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐         │
│  │  CHECK 1:       │ →  │  CHECK 2:       │ →  │  CHECK 3:       │         │
│  │  Expansion      │    │  Exact Match    │    │  Subsumption    │         │
│  │  (Pre-expanded) │    │  O(1) Hash      │    │  (Neo4j IS-A)   │         │
│  └─────────────────┘    └─────────────────┘    └─────────────────┘         │
│                                                                              │
│  Output: EvaluationResult[] with matched value sets per fact                │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 2.5: RuleEngine (Compound Conditions)                                  │
│  ──────────────────────────────────────────                                  │
│  Evaluates clinical rules that combine:                                      │
│  • Value Set matches (from Step 2)                                          │
│  • Lab thresholds (Lactate > 2.0, Creatinine > 2.0, etc.)                  │
│  • Temporal conditions (50% creatinine rise in 48h)                         │
│  • Compound logic (AND, OR, NOT)                                            │
│                                                                              │
│  Example Rule: "Sepsis AND Lactate > 2.0 mmol/L" → CRITICAL alert           │
│                                                                              │
│  Output: FiredRule[] with matching facts and evidence                       │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 3: AlertGenerator                                                      │
│  ──────────────────────                                                      │
│  • Converts EvaluationResults + FiredRules → CDSSAlerts                     │
│  • Groups alerts by clinical domain (prevents alert fatigue)                │
│  • Merges similar alerts                                                     │
│  • Prioritizes by severity (Critical > High > Moderate > Low)               │
│  • Adds clinical recommendations from ClinicalIndicatorRegistry             │
│                                                                              │
│  Output: CDSSAlert[] sorted by severity with recommendations                │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        CDSSEvaluationResponse                                │
│  {                                                                           │
│    "success": true,                                                          │
│    "evaluation_id": "uuid",                                                  │
│    "facts_extracted": 12,                                                    │
│    "facts_evaluated": 12,                                                    │
│    "rules_fired": 2,                                                         │
│    "matches_found": 5,                                                       │
│    "alerts_generated": 3,                                                    │
│    "alerts": [ ... ],                                                        │
│    "pipeline_used": "THREE-CHECK",                                           │
│    "execution_time_ms": 45.2                                                 │
│  }                                                                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### CDSS API Endpoints

All CDSS endpoints are under the `/v1/cdss` base path.

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/evaluate` | **Main endpoint** - Full CDSS evaluation pipeline |
| `POST` | `/facts/build` | Extract facts from FHIR resources |
| `POST` | `/evaluate/facts` | Evaluate pre-built facts |
| `POST` | `/alerts/generate` | Generate alerts from evaluation results |
| `POST` | `/validate` | Quick single-code validation |
| `GET` | `/health` | CDSS health check |
| `GET` | `/domains` | List clinical domains |
| `GET` | `/indicators` | List clinical indicators |
| `GET` | `/severity-mapping` | Value set to severity mappings |

---

### API Endpoint Details

#### 1. Full CDSS Evaluation (Main Endpoint)

The primary integration point for clinical systems. Accepts FHIR resources, evaluates against all clinical rules, and returns alerts.

```http
POST /v1/cdss/evaluate
Content-Type: application/json
```

**Request Body:**
```json
{
  "patient_id": "patient-123",
  "encounter_id": "encounter-456",
  "conditions": [
    {
      "resourceType": "Condition",
      "code": {
        "coding": [{
          "system": "http://snomed.info/sct",
          "code": "91302008",
          "display": "Sepsis"
        }]
      },
      "clinicalStatus": {
        "coding": [{ "code": "active" }]
      }
    }
  ],
  "observations": [
    {
      "resourceType": "Observation",
      "code": {
        "coding": [{
          "system": "http://loinc.org",
          "code": "2524-7",
          "display": "Lactate"
        }]
      },
      "valueQuantity": {
        "value": 3.5,
        "unit": "mmol/L"
      },
      "status": "final"
    }
  ],
  "medications": [],
  "procedures": [],
  "allergies": [],
  "options": {
    "enable_subsumption": true,
    "generate_alerts": true,
    "evaluate_rules": true,
    "include_details": true,
    "group_alerts_by_domain": true
  }
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "evaluation_id": "eval-a1b2c3d4-e5f6-7890",
  "patient_id": "patient-123",
  "encounter_id": "encounter-456",
  "facts_extracted": 2,
  "facts_evaluated": 2,
  "rules_evaluated": 10,
  "rules_fired": 1,
  "matches_found": 2,
  "alerts_generated": 2,
  "alerts": [
    {
      "alert_id": "rule-sepsis-lactate-elevated-1702300000000000000",
      "severity": "critical",
      "clinical_domain": "sepsis",
      "title": "CRITICAL: Sepsis with Elevated Lactate",
      "description": "Patient has sepsis diagnosis with lactate > 2.0 mmol/L indicating tissue hypoperfusion",
      "evidence": [
        {
          "fact_id": "fact-abc123",
          "code": "91302008",
          "display": "Sepsis",
          "value_set_id": "SepsisDiagnosis"
        },
        {
          "fact_id": "fact-def456",
          "code": "2524-7",
          "display": "Lactate",
          "numeric_value": 3.5,
          "unit": "mmol/L"
        }
      ],
      "recommendations": [
        "Initiate Sepsis Hour-1 Bundle immediately",
        "Obtain blood cultures before antibiotics",
        "Administer broad-spectrum antibiotics within 1 hour",
        "Begin 30 mL/kg crystalloid resuscitation if hypotensive",
        "Repeat lactate in 2-4 hours"
      ],
      "guideline_links": ["Surviving Sepsis Campaign 2021"],
      "generated_at": "2025-12-11T10:30:00Z",
      "status": "active",
      "metadata": {
        "rule_id": "sepsis-lactate-elevated",
        "rule_name": "Sepsis with Elevated Lactate",
        "rule_version": "1.0",
        "rule_category": "diagnosis",
        "rule_priority": 1
      }
    }
  ],
  "matched_domains": ["sepsis"],
  "pipeline_used": "THREE-CHECK",
  "execution_time_ms": 45.2,
  "warnings": []
}
```

#### 2. Build Facts from FHIR Resources

Extracts clinical facts from FHIR resources without full evaluation.

```http
POST /v1/cdss/facts/build
Content-Type: application/json
```

**Request Body:**
```json
{
  "patient_id": "patient-123",
  "conditions": [...],
  "observations": [...],
  "medications": [...],
  "options": {
    "extract_primary_codes_only": true,
    "include_inactive_facts": false
  }
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "patient_id": "patient-123",
  "fact_set": {
    "patient_id": "patient-123",
    "conditions": [
      {
        "id": "fact-abc123",
        "fact_type": "condition",
        "code": "91302008",
        "system": "http://snomed.info/sct",
        "display": "Sepsis",
        "status": "active"
      }
    ],
    "observations": [
      {
        "id": "fact-def456",
        "fact_type": "lab",
        "code": "2524-7",
        "system": "http://loinc.org",
        "display": "Lactate",
        "numeric_value": 3.5,
        "unit": "mmol/L",
        "status": "active"
      }
    ],
    "medications": [],
    "procedures": [],
    "allergies": [],
    "total_facts": 2
  },
  "total_facts_extracted": 2,
  "extraction_time_ms": 5.2
}
```

#### 3. Evaluate Pre-Built Facts

Evaluates a pre-built fact set against clinical rules (skips fact building step).

```http
POST /v1/cdss/evaluate/facts
Content-Type: application/json
```

**Request Body:**
```json
{
  "patient_id": "patient-123",
  "fact_set": { ... },
  "options": {
    "enable_subsumption": true,
    "generate_alerts": true,
    "evaluate_rules": true
  }
}
```

#### 4. Generate Alerts from Evaluation Results

Generates clinical alerts from pre-computed evaluation results.

```http
POST /v1/cdss/alerts/generate
Content-Type: application/json
```

**Request Body:**
```json
{
  "patient_id": "patient-123",
  "evaluation_results": [
    {
      "fact_id": "fact-abc123",
      "matched": true,
      "matched_value_sets": [
        { "value_set_id": "SepsisDiagnosis", "match_type": "exact" }
      ]
    }
  ],
  "fact_set": { ... },
  "options": {
    "minimum_severity": "low",
    "group_by_domain": true,
    "include_recommendations": true,
    "merge_similar_alerts": true
  }
}
```

#### 5. Quick Single-Code Validation

Validates a single clinical code against value sets.

```http
POST /v1/cdss/validate
Content-Type: application/json
```

**Request Body:**
```json
{
  "code": "91302008",
  "system": "http://snomed.info/sct",
  "value_set_ids": ["SepsisDiagnosis", "AUSepsisConditions"]
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "code": "91302008",
  "system": "http://snomed.info/sct",
  "matched": true,
  "matched_value_sets": [
    {
      "value_set_id": "SepsisDiagnosis",
      "value_set_name": "Sepsis Diagnosis Codes",
      "match_type": "exact",
      "domain": "sepsis"
    }
  ],
  "evaluation_time_ms": 2.3
}
```

#### 6. CDSS Health Check

Returns health status of all CDSS components.

```http
GET /v1/cdss/health
```

**Response (200 OK):**
```json
{
  "status": "healthy",
  "timestamp": "2025-12-11T10:30:00Z",
  "components": {
    "fact_builder": "available",
    "evaluator": "available",
    "alert_generator": "available"
  },
  "version": "1.0.0",
  "pipeline_enabled": "THREE-CHECK"
}
```

#### 7. List Clinical Domains

Returns supported clinical domains and their indicators.

```http
GET /v1/cdss/domains
```

**Response (200 OK):**
```json
{
  "success": true,
  "domains": [
    {
      "domain": "sepsis",
      "indicators": ["Sepsis Diagnosis", "Septic Shock"],
      "indicator_count": 2
    },
    {
      "domain": "renal",
      "indicators": ["AKI", "CKD", "Dialysis Required"],
      "indicator_count": 3
    }
  ],
  "total": 8
}
```

#### 8. List Clinical Indicators

Returns clinical indicators with optional domain filtering.

```http
GET /v1/cdss/indicators
GET /v1/cdss/indicators?domain=sepsis
```

**Response (200 OK):**
```json
{
  "success": true,
  "indicators": [
    {
      "id": "sepsis-indicator",
      "name": "Sepsis Diagnosis",
      "description": "Patient has an active sepsis diagnosis",
      "domain": "sepsis",
      "severity": "critical",
      "value_sets": ["SepsisDiagnosis"],
      "recommendations": [
        "Consider Sepsis-3 criteria evaluation",
        "Obtain lactate level if not recent"
      ]
    }
  ],
  "total": 15
}
```

---

## Component 6: RuleEngine Deep Dive

### Purpose

The RuleEngine evaluates compound clinical rules that combine value set membership with lab thresholds, temporal conditions, and boolean logic.

### Condition Types

| Type | Description | Example |
|------|-------------|---------|
| `VALUE_SET` | Code matches a value set | `SepsisDiagnosis` contains code |
| `THRESHOLD` | Numeric comparison | `Lactate > 2.0 mmol/L` |
| `COMPOUND` | AND/OR/NOT logic | `Sepsis AND Lactate > 2.0` |
| `TEMPORAL` | Change over time | `Creatinine ↑50% in 48h` |
| `PRESENT` | Fact type exists | Has any observation |
| `ABSENT` | Fact type missing | No allergies documented |

### Threshold Operators

| Operator | Description |
|----------|-------------|
| `>` | Greater than |
| `>=` | Greater than or equal |
| `<` | Less than |
| `<=` | Less than or equal |
| `==` | Equal to |
| `!=` | Not equal to |
| `between` | Value in range [low, high] |
| `outside` | Value outside range |

### Default Clinical Rules

| Rule ID | Description | Conditions | Severity |
|---------|-------------|------------|----------|
| `sepsis-lactate-elevated` | Sepsis + Lactate > 2.0 | Compound (AND) | Critical |
| `sepsis-diagnosis` | Any sepsis diagnosis | Value Set only | Critical |
| `aki-creatinine-elevated` | AKI + Creatinine > 2.0 | Compound (AND) | High |
| `aki-creatinine-rise` | 50% creatinine ↑ in 48h | Temporal | High |
| `hypoglycemia-critical` | Diabetes + Glucose < 70 | Compound (AND) | Critical |
| `hf-elevated-bnp` | Heart Failure + BNP > 400 | Compound (AND) | High |
| `resp-failure-hypoxia` | Resp Failure + SpO2 < 90% | Compound (AND) | Critical |

### LOINC Codes for Lab Thresholds

| Code | Display | Unit |
|------|---------|------|
| `2524-7` | Lactate | mmol/L |
| `2160-0` | Creatinine | mg/dL |
| `2339-0` | Glucose | mg/dL |
| `30934-4` | BNP | pg/mL |
| `2708-6` | SpO2 | % |
| `718-7` | Hemoglobin | g/dL |
| `2823-3` | Potassium | mEq/L |
| `2951-2` | Sodium | mEq/L |
| `6690-2` | WBC | K/uL |
| `777-3` | Platelets | K/uL |

### Rule Definition Example

```go
{
  ID:          "sepsis-lactate-elevated",
  Name:        "Sepsis with Elevated Lactate",
  Description: "Alert for sepsis diagnosis with lactate > 2.0 mmol/L",
  Domain:      DomainSepsis,
  Severity:    SeverityCritical,
  Category:    "diagnosis",
  Conditions: []RuleCondition{
    {
      Type:             ConditionTypeCompound,
      CompoundOperator: OpAnd,
      SubConditions: []RuleCondition{
        {Type: ConditionTypeValueSet, ValueSetID: "SepsisDiagnosis"},
        {Type: ConditionTypeThreshold, LoincCode: "2524-7",
         Operator: OpGreaterThan, Value: 2.0, Unit: "mmol/L"},
      },
    },
  },
  AlertTitle:       "CRITICAL: Sepsis with Elevated Lactate",
  AlertDescription: "Patient has sepsis diagnosis with lactate > 2.0 mmol/L",
  Recommendations: []string{
    "Initiate Sepsis Hour-1 Bundle immediately",
    "Obtain blood cultures before antibiotics",
  },
  GuidelineReferences: []string{"Surviving Sepsis Campaign 2021"},
  Enabled:  true,
  Priority: 1,
}
```

---

## Component 7: FactBuilder Deep Dive

### Purpose

Converts FHIR R4 resources into normalized `ClinicalFact` objects for rule evaluation.

### Supported FHIR Resources

| Resource Type | Fact Type | Extracted Data |
|---------------|-----------|----------------|
| `Condition` | `condition` | Code, status, onset, severity |
| `Observation` | `observation/lab/vital_sign` | Code, value, unit, interpretation |
| `MedicationRequest` | `medication` | Code, dosage, status, intent |
| `Procedure` | `procedure` | Code, status, performed date |
| `AllergyIntolerance` | `allergy` | Code, criticality, type |

### ClinicalFact Structure

```go
type ClinicalFact struct {
  ID                string     // Deterministic SHA256 hash
  FactType          FactType   // condition, observation, medication, etc.
  Code              string     // SNOMED, LOINC, ICD-10, RxNorm code
  System            string     // Code system URI
  Display           string     // Human-readable display
  Status            FactStatus // active, inactive, resolved
  NumericValue      *float64   // For observations
  Unit              string     // Unit of measure
  Interpretation    string     // H, HH, L, LL, N, A, etc.
  EffectiveDateTime *time.Time // When fact was recorded
  Severity          string     // Clinical severity
}
```

### Deterministic ID Generation

Fact IDs are generated using SHA256 hashing to ensure:
- Same clinical data always produces same fact ID
- Enables deduplication across sessions
- Supports caching and reference tracking

```go
// ID = SHA256(patient_id + code + system + effective_date)
hash := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s",
  patientID, code, system, effectiveDate)))
factID := fmt.Sprintf("fact-%x", hash[:8])
```

---

## Component 8: AlertGenerator Deep Dive

### Purpose

Converts evaluation results and fired rules into actionable clinical alerts with recommendations.

### Alert Severity Levels

| Severity | Priority | Description |
|----------|----------|-------------|
| `critical` | 1 | Immediate action required |
| `high` | 2 | Urgent attention needed |
| `moderate` | 3 | Action recommended |
| `low` | 4 | Informational |

### Alert Grouping & Deduplication

- **Domain Grouping**: Alerts grouped by clinical domain to reduce fatigue
- **Similar Alert Merging**: Alerts with same domain/severity merged
- **Evidence Consolidation**: Multiple facts combined into single alert

### CDSSAlert Structure

```json
{
  "alert_id": "rule-sepsis-lactate-elevated-1702300000",
  "severity": "critical",
  "clinical_domain": "sepsis",
  "title": "CRITICAL: Sepsis with Elevated Lactate",
  "description": "Patient has sepsis with tissue hypoperfusion",
  "evidence": [
    {
      "fact_id": "fact-abc123",
      "fact_type": "condition",
      "code": "91302008",
      "display": "Sepsis"
    }
  ],
  "recommendations": [
    "Initiate Sepsis Hour-1 Bundle",
    "Obtain blood cultures"
  ],
  "guideline_links": ["Surviving Sepsis Campaign 2021"],
  "generated_at": "2025-12-11T10:30:00Z",
  "status": "active",
  "metadata": {
    "rule_id": "sepsis-lactate-elevated",
    "patient_id": "patient-123"
  }
}
```

---

## Performance Characteristics

| Component | Typical Latency | Notes |
|-----------|----------------|-------|
| FactBuilder | ~1ms | For 10 FHIR resources |
| THREE-CHECK Pipeline | ~5ms/fact | O(1) hash + optional Neo4j |
| RuleEngine | ~2ms | For 10 rules (uses lookup maps) |
| AlertGenerator | ~1ms | In-memory grouping |
| **Total E2E** | **15-50ms** | Typical patient evaluation |

---

## Quick Reference: Common CDSS Operations

### Full Patient Evaluation

```bash
curl -X POST http://localhost:8092/v1/cdss/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "patient-123",
    "conditions": [
      {"code": {"coding": [{"system": "http://snomed.info/sct", "code": "91302008"}]},
       "clinicalStatus": {"coding": [{"code": "active"}]}}
    ],
    "observations": [
      {"code": {"coding": [{"system": "http://loinc.org", "code": "2524-7"}]},
       "valueQuantity": {"value": 3.5, "unit": "mmol/L"}}
    ],
    "options": {"evaluate_rules": true, "generate_alerts": true}
  }'
```

### Quick Code Validation

```bash
curl -X POST http://localhost:8092/v1/cdss/validate \
  -H "Content-Type: application/json" \
  -d '{"code": "91302008", "system": "http://snomed.info/sct"}'
```

### CDSS Health Check

```bash
curl http://localhost:8092/v1/cdss/health
```

### List Clinical Domains

```bash
curl http://localhost:8092/v1/cdss/domains
```

---

*Documentation updated: 2025-12-11*
*KB-7 Terminology Service v1.0.0*
*CDSS Pipeline: 100% Complete*
