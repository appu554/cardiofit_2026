# KB-7 Terminology Service API Documentation

**Version**: 1.0.0
**Base URL**: `http://localhost:8087`
**Protocol**: REST/JSON

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Architecture Overview](#architecture-overview)
3. [Authentication](#authentication)
4. [Core Endpoints](#core-endpoints)
   - [Health & System](#health--system)
   - [Region Management](#region-management)
   - [Terminology Systems](#terminology-systems)
   - [Concept Operations](#concept-operations)
   - [Value Sets](#value-sets)
   - [Rule Engine (THREE-CHECK PIPELINE)](#rule-engine-three-check-pipeline)
   - [Subsumption Testing](#subsumption-testing)
   - [HCC/RAF Calculation](#hccraf-calculation)
   - [Semantic Operations](#semantic-operations)
5. [Workflow Examples](#workflow-examples)
6. [Error Handling](#error-handling)

---

## Quick Start

### 1. Check Service Health
```bash
curl http://localhost:8087/health
```

### 2. Seed Value Sets (First Time Only)
```bash
curl -X POST http://localhost:8087/v1/rules/seed
```

### 3. Validate a Code Against a Value Set
```bash
curl -X POST http://localhost:8087/v1/rules/valuesets/SepsisDiagnosis/validate \
  -H "Content-Type: application/json" \
  -d '{"code": "448417001", "system": "http://snomed.info/sct"}'
```

### 4. Find All Value Sets for a Code (Reverse Lookup)
```bash
curl -X POST http://localhost:8087/v1/rules/classify \
  -H "Content-Type: application/json" \
  -d '{"code": "91302008", "system": "http://snomed.info/sct"}'
```

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                    KB-7 Terminology Service                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐            │
│  │   REST API   │   │   GraphQL    │   │   Metrics    │            │
│  │  (Gin/JSON)  │   │  (Optional)  │   │ (Prometheus) │            │
│  └──────┬───────┘   └──────────────┘   └──────────────┘            │
│         │                                                            │
│  ┌──────▼─────────────────────────────────────────────────────┐     │
│  │              THREE-CHECK VALIDATION PIPELINE                │     │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │     │
│  │  │ STEP 1      │  │ STEP 2      │  │ STEP 3              │ │     │
│  │  │ Expansion   │→ │ Exact Match │→ │ Subsumption (IS-A)  │ │     │
│  │  │ (PostgreSQL)│  │ (In-Memory) │  │ (Neo4j ELK/GraphDB) │ │     │
│  │  └─────────────┘  └─────────────┘  └─────────────────────┘ │     │
│  └────────────────────────────────────────────────────────────┘     │
│                                                                      │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐     │
│  │   PostgreSQL    │  │   Neo4j AU      │  │   Redis Cache   │     │
│  │ (Value Sets)    │  │ (SNOMED/AMT)    │  │ (Performance)   │     │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘     │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### THREE-CHECK PIPELINE Explained

| Step | Check | Purpose | Speed |
|------|-------|---------|-------|
| 1 | **Expansion** | Load all codes from value set definition | ~1ms (cached) |
| 2 | **Exact Match** | Direct membership lookup | ~0.01ms |
| 3 | **Subsumption** | IS-A hierarchy traversal via Neo4j ELK | ~2-5ms |

---

## Authentication

Currently no authentication required for development. For production, add:

```bash
curl -H "Authorization: Bearer <token>" \
     -H "X-Region: au" \
     http://localhost:8087/v1/...
```

### Headers

| Header | Description | Example |
|--------|-------------|---------|
| `X-Region` | Regional Neo4j database selection | `au`, `us`, `in` |
| `Content-Type` | Request body format | `application/json` |

---

## Core Endpoints

---

### Health & System

#### GET /health
Check service health status.

**Request:**
```bash
curl http://localhost:8087/health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "kb-7-terminology",
  "checks": {
    "database": { "status": "healthy" },
    "cache": { "status": "healthy" }
  },
  "graphdb": {
    "status": "healthy",
    "url": "http://localhost:7200",
    "repository": "kb7-terminology"
  }
}
```

---

#### GET /version
Get service version and capabilities.

**Request:**
```bash
curl http://localhost:8087/version
```

**Response:**
```json
{
  "service": "kb-7-terminology",
  "version": "1.0.0+sha.initial",
  "environment": "development",
  "capabilities": [
    "concept_lookup",
    "concept_search",
    "code_validation",
    "value_sets",
    "subsumption_testing",
    "owl_reasoning",
    "rule_engine_valuesets"
  ]
}
```

---

#### GET /metrics
Prometheus metrics endpoint.

**Request:**
```bash
curl http://localhost:8087/metrics
```

---

### Region Management

#### GET /v1/regions
List all supported terminology regions.

**Request:**
```bash
curl http://localhost:8087/v1/regions
```

**Response:**
```json
{
  "multi_region_enabled": true,
  "default_region": "us",
  "count": 3,
  "regions": [
    {
      "region": "au",
      "region_name": "Australia",
      "clinical_terminology": "SNOMED CT-AU",
      "drug_terminology": "AMT",
      "module_id": "900062011000036103"
    },
    {
      "region": "us",
      "region_name": "United States",
      "clinical_terminology": "SNOMED CT-US",
      "drug_terminology": "RxNorm"
    },
    {
      "region": "in",
      "region_name": "India",
      "clinical_terminology": "SNOMED CT",
      "drug_terminology": "CDCI"
    }
  ]
}
```

---

#### GET /v1/region
Get current region info (based on X-Region header).

**Request:**
```bash
curl -H "X-Region: au" http://localhost:8087/v1/region
```

---

### Terminology Systems

#### GET /v1/systems
List available terminology systems.

**Request:**
```bash
curl http://localhost:8087/v1/systems
```

---

#### GET /v1/systems/:identifier
Get details of a specific terminology system.

**Request:**
```bash
curl http://localhost:8087/v1/systems/http%3A%2F%2Fsnomed.info%2Fsct
```

---

### Concept Operations

#### GET /v1/concepts
Search for concepts.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `q` | string | Search query |
| `system` | string | Filter by code system |
| `limit` | int | Max results (default: 20) |
| `offset` | int | Pagination offset |

**Request:**
```bash
curl "http://localhost:8087/v1/concepts?q=sepsis&system=http://snomed.info/sct&limit=5"
```

---

#### GET /v1/concepts/:system/:code
Lookup a specific concept.

**Request:**
```bash
curl "http://localhost:8087/v1/concepts/http%3A%2F%2Fsnomed.info%2Fsct/91302008"
```

---

#### POST /v1/concepts/validate
Validate a code exists in a terminology system.

**Request:**
```bash
curl -X POST http://localhost:8087/v1/concepts/validate \
  -H "Content-Type: application/json" \
  -d '{
    "code": "91302008",
    "system": "http://snomed.info/sct"
  }'
```

---

#### POST /v1/concepts/batch-lookup
Batch lookup multiple concepts.

**Request:**
```bash
curl -X POST http://localhost:8087/v1/concepts/batch-lookup \
  -H "Content-Type: application/json" \
  -d '{
    "codes": [
      {"code": "91302008", "system": "http://snomed.info/sct"},
      {"code": "448417001", "system": "http://snomed.info/sct"}
    ]
  }'
```

---

### Value Sets

#### GET /v1/valuesets
List all FHIR value sets.

**Request:**
```bash
curl "http://localhost:8087/v1/valuesets?limit=10"
```

---

#### GET /v1/valuesets/:url
Get a specific value set.

**Request:**
```bash
curl "http://localhost:8087/v1/valuesets/http%3A%2F%2Fhl7.org%2Ffhir%2FValueSet%2Fadministrative-gender"
```

---

#### POST /v1/valuesets/:url/expand
Expand a value set to list all codes.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/valuesets/http%3A%2F%2Fhl7.org%2Ffhir%2FValueSet%2Fadministrative-gender/expand"
```

---

#### POST /v1/valuesets/:url/validate-code
Validate a code in a value set.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/valuesets/http%3A%2F%2Fhl7.org%2Ffhir%2FValueSet%2Fadministrative-gender/validate-code" \
  -H "Content-Type: application/json" \
  -d '{"code": "male"}'
```

---

### Rule Engine (THREE-CHECK PIPELINE)

The Rule Engine provides the most powerful validation with the **THREE-CHECK PIPELINE**:
1. **Expansion**: Get all codes in value set
2. **Exact Match**: Check if code is directly in the set
3. **Subsumption**: Check if code IS-A any code in the set (hierarchical)

---

#### GET /v1/rules/valuesets
List all rule-based value sets.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `limit` | int | Max results (default: 100) |
| `offset` | int | Pagination offset |
| `type` | string | Filter: `extensional`, `intensional` |

**Request:**
```bash
curl "http://localhost:8087/v1/rules/valuesets?limit=10"
```

**Response:**
```json
{
  "value_sets": [
    {
      "name": "SepsisDiagnosis",
      "description": "SNOMED CT codes for sepsis diagnosis",
      "type": "extensional",
      "code_count": 34
    },
    {
      "name": "AUSepsisConditions",
      "description": "Australian Sepsis Clinical Pathway conditions",
      "type": "extensional",
      "code_count": 28
    }
  ],
  "total": 55,
  "limit": 10,
  "offset": 0
}
```

---

#### GET /v1/rules/valuesets/:identifier
Get a specific value set definition.

**Request:**
```bash
curl "http://localhost:8087/v1/rules/valuesets/SepsisDiagnosis"
```

**Response:**
```json
{
  "identifier": "SepsisDiagnosis",
  "name": "SepsisDiagnosis",
  "description": "SNOMED CT codes for sepsis diagnosis",
  "type": "extensional",
  "system": "http://snomed.info/sct",
  "codes": [
    {"code": "91302008", "display": "Sepsis (disorder)"},
    {"code": "10001005", "display": "Bacterial sepsis"},
    {"code": "448417001", "display": "Streptococcal sepsis"}
  ]
}
```

---

#### POST /v1/rules/valuesets/:identifier/expand
Expand a value set to get all codes.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/rules/valuesets/SepsisDiagnosis/expand" \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Response:**
```json
{
  "value_set_id": "SepsisDiagnosis",
  "total": 34,
  "codes": [
    {"code": "91302008", "display": "Sepsis (disorder)", "system": "http://snomed.info/sct"},
    {"code": "10001005", "display": "Bacterial sepsis", "system": "http://snomed.info/sct"}
  ],
  "cached_result": true,
  "expanded_at": "2025-12-10T12:00:00Z"
}
```

---

#### POST /v1/rules/valuesets/:identifier/validate ⭐ KEY API

**This is the main THREE-CHECK PIPELINE endpoint.**

Validate a code against a value set using all three checks.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/rules/valuesets/SepsisDiagnosis/validate" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "448417001",
    "system": "http://snomed.info/sct"
  }'
```

**Response (Subsumption Match):**
```json
{
  "valid": true,
  "value_set_id": "SepsisDiagnosis",
  "code": "448417001",
  "system": "http://snomed.info/sct",
  "message": "Code '448417001' is valid via subsumption: IS-A '91302008' (Sepsis (disorder)) with path length 4",
  "match_type": "subsumption",
  "matched_code": "91302008",
  "pipeline": {
    "step1_expansion": {
      "status": "completed",
      "codes_count": 34,
      "cached": true,
      "duration": "1.124208ms"
    },
    "step2_exact_match": {
      "status": "no_match",
      "checked": true,
      "match_found": false,
      "checked_code": "448417001"
    },
    "step3_subsumption": {
      "status": "match",
      "checked": true,
      "match_found": true,
      "checked_code": "448417001",
      "matched_ancestor": "91302008",
      "ancestor_display": "Sepsis (disorder)",
      "path_length": 4,
      "source": "neo4j",
      "codes_checked": 1
    }
  }
}
```

**Response (Exact Match):**
```json
{
  "valid": true,
  "value_set_id": "SepsisDiagnosis",
  "code": "91302008",
  "system": "http://snomed.info/sct",
  "message": "Code found in value set via exact membership match",
  "match_type": "exact",
  "matched_code": "91302008",
  "pipeline": {
    "step1_expansion": {
      "status": "completed",
      "codes_count": 34,
      "cached": true,
      "duration": "0.5ms"
    },
    "step2_exact_match": {
      "status": "match",
      "checked": true,
      "match_found": true,
      "checked_code": "91302008"
    },
    "step3_subsumption": {
      "status": "skipped",
      "checked": false
    }
  }
}
```

**Response (No Match):**
```json
{
  "valid": false,
  "value_set_id": "SepsisDiagnosis",
  "code": "12345678",
  "system": "http://snomed.info/sct",
  "message": "Code '12345678' not found in value set 'SepsisDiagnosis' via membership or subsumption (checked 34 codes)",
  "match_type": "none",
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
      "checked_code": "12345678"
    },
    "step3_subsumption": {
      "status": "no_match",
      "checked": true,
      "match_found": false,
      "source": "neo4j",
      "codes_checked": 34
    }
  }
}
```

---

#### POST /v1/rules/classify ⭐ REVERSE LOOKUP

Find ALL value sets that contain a given code.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/rules/classify" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "91302008",
    "system": "http://snomed.info/sct"
  }'
```

**Response:**
```json
{
  "code": "91302008",
  "system": "http://snomed.info/sct",
  "match_count": 2,
  "total_value_sets_checked": 55,
  "matching_value_sets": [
    {
      "value_set_id": "SepsisDiagnosis",
      "match_type": "exact",
      "matched_code": "91302008"
    },
    {
      "value_set_id": "AUSepsisConditions",
      "match_type": "exact",
      "matched_code": "91302008"
    }
  ],
  "processing_time_ms": 408
}
```

---

#### POST /v1/rules/seed
Seed builtin value sets to database (run once on first setup).

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/rules/seed"
```

**Response:**
```json
{
  "message": "Builtin value sets seeded successfully",
  "description": "18 FHIR R4 standard value sets have been migrated to database",
  "timestamp": "2025-12-10T12:00:00Z"
}
```

---

#### POST /v1/rules/valuesets/:identifier/refresh
Refresh the cache for a value set.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/rules/valuesets/SepsisDiagnosis/refresh"
```

---

### Subsumption Testing

Test IS-A relationships in the SNOMED CT hierarchy using Neo4j's ELK-materialized ontology.

---

#### POST /v1/subsumption/test ⭐

Test if code_a IS-A code_b (subsumption relationship).

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/subsumption/test" \
  -H "Content-Type: application/json" \
  -d '{
    "code_a": "22973011000036107",
    "code_b": "414984009",
    "system": "http://snomed.info/sct"
  }'
```

**Response:**
```json
{
  "subsumes": true,
  "relationship": "subsumed_by",
  "code_a": "22973011000036107",
  "code_b": "414984009",
  "system": "http://snomed.info/sct",
  "path_length": 2,
  "reasoning_type": "neo4j",
  "execution_time_ms": 2.5,
  "cached_result": false,
  "tested_at": "2025-12-10T12:00:00Z"
}
```

**Interpretation:**
- `subsumes: true` = code_a IS-A code_b (code_a is a subtype of code_b)
- `path_length: 2` = There are 2 steps in the hierarchy between them
- `reasoning_type: neo4j` = Used Neo4j's pre-computed ELK hierarchy (fast)

---

#### POST /v1/subsumption/test/batch

Test multiple subsumption relationships at once.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/subsumption/test/batch" \
  -H "Content-Type: application/json" \
  -d '{
    "tests": [
      {"code_a": "22973011000036107", "code_b": "414984009", "system": "http://snomed.info/sct"},
      {"code_a": "448417001", "code_b": "91302008", "system": "http://snomed.info/sct"}
    ]
  }'
```

---

#### POST /v1/subsumption/ancestors

Get all ancestors (parents) of a concept up to max_depth.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/subsumption/ancestors" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "448417001",
    "system": "http://snomed.info/sct",
    "max_depth": 5
  }'
```

**Response:**
```json
{
  "code": "448417001",
  "system": "http://snomed.info/sct",
  "ancestors": [
    {"code": "91302008", "display": "Sepsis (disorder)", "depth": 1},
    {"code": "128139000", "display": "Inflammatory disorder", "depth": 2},
    {"code": "64572001", "display": "Disease", "depth": 3}
  ],
  "total_ancestors": 15,
  "max_depth_reached": false
}
```

---

#### POST /v1/subsumption/descendants

Get all descendants (children) of a concept.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/subsumption/descendants" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "91302008",
    "system": "http://snomed.info/sct",
    "max_depth": 2
  }'
```

---

#### POST /v1/subsumption/common-ancestors

Find common ancestors between two concepts.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/subsumption/common-ancestors" \
  -H "Content-Type: application/json" \
  -d '{
    "code_a": "448417001",
    "code_b": "10001005",
    "system": "http://snomed.info/sct"
  }'
```

---

#### GET /v1/subsumption/config

Get subsumption service configuration.

**Request:**
```bash
curl "http://localhost:8087/v1/subsumption/config"
```

---

### HCC/RAF Calculation

Map ICD-10 codes to HCC (Hierarchical Condition Categories) and calculate Risk Adjustment Factors.

---

#### GET /v1/hcc/map/:icd10_code

Map a single ICD-10 code to HCC.

**Request:**
```bash
curl "http://localhost:8087/v1/hcc/map/E1122"
```

---

#### POST /v1/hcc/map/batch

Batch map ICD-10 codes to HCC.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/hcc/map/batch" \
  -H "Content-Type: application/json" \
  -d '{
    "codes": ["E1122", "I10", "J449"]
  }'
```

---

#### POST /v1/hcc/raf/calculate

Calculate RAF score for a patient.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/hcc/raf/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "12345",
    "age": 67,
    "sex": "M",
    "icd10_codes": ["E1122", "I10", "J449"],
    "model_version": "V28"
  }'
```

---

#### GET /v1/hcc/hierarchies

Get HCC hierarchy information.

**Request:**
```bash
curl "http://localhost:8087/v1/hcc/hierarchies"
```

---

#### GET /v1/hcc/coefficients

Get RAF coefficients.

**Request:**
```bash
curl "http://localhost:8087/v1/hcc/coefficients"
```

---

### Semantic Operations

Direct GraphDB/SPARQL operations for advanced queries.

---

#### POST /v1/semantic/sparql

Execute a SPARQL query against GraphDB.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/semantic/sparql" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 10"
  }'
```

---

#### GET /v1/semantic/concepts/:uri

Get a concept by URI.

**Request:**
```bash
curl "http://localhost:8087/v1/semantic/concepts/http%3A%2F%2Fsnomed.info%2Fid%2F91302008"
```

---

#### GET /v1/semantic/drug-interactions/:medication_uri

Get drug interactions for a medication.

**Request:**
```bash
curl "http://localhost:8087/v1/semantic/drug-interactions/http%3A%2F%2Fsnomed.info%2Fid%2F414984009"
```

---

#### GET /v1/semantic/mappings/:source_code

Get semantic mappings for a code.

**Request:**
```bash
curl "http://localhost:8087/v1/semantic/mappings/91302008"
```

---

### Translation

Translate concepts between terminology systems.

---

#### POST /v1/translate

Translate a concept to another system.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/translate" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "91302008",
    "source_system": "http://snomed.info/sct",
    "target_system": "http://hl7.org/fhir/sid/icd-10"
  }'
```

---

#### POST /v1/translate/batch

Batch translate concepts.

**Request:**
```bash
curl -X POST "http://localhost:8087/v1/translate/batch" \
  -H "Content-Type: application/json" \
  -d '{
    "translations": [
      {"code": "91302008", "source_system": "http://snomed.info/sct", "target_system": "http://hl7.org/fhir/sid/icd-10"}
    ]
  }'
```

---

## Workflow Examples

### Workflow 1: Clinical Decision Support - Sepsis Protocol

```bash
# Step 1: Patient presents with diagnosis code 448417001 (Streptococcal sepsis)
# Step 2: Check if this triggers the Sepsis Protocol value set

curl -X POST "http://localhost:8087/v1/rules/valuesets/SepsisDiagnosis/validate" \
  -H "Content-Type: application/json" \
  -d '{"code": "448417001", "system": "http://snomed.info/sct"}'

# Response shows: valid=true via subsumption (IS-A Sepsis)
# → Trigger Sepsis Protocol workflow
```

### Workflow 2: Drug Validation - Australian Medications

```bash
# Step 1: Find what "Oxycodone" drug products exist (Australian region)
curl -H "X-Region: au" \
     "http://localhost:8087/v1/concepts?q=oxycodone&system=http://snomed.info/sct"

# Step 2: Check if it's a controlled substance
curl -X POST "http://localhost:8087/v1/subsumption/test" \
  -H "Content-Type: application/json" \
  -d '{
    "code_a": "22973011000036107",
    "code_b": "414984009",
    "system": "http://snomed.info/sct"
  }'

# Response: subsumes=true → This is an oxycodone product
```

### Workflow 3: Automatic Value Set Discovery

```bash
# Given a SNOMED code, find ALL clinical protocols it triggers

curl -X POST "http://localhost:8087/v1/rules/classify" \
  -H "Content-Type: application/json" \
  -d '{"code": "91302008", "system": "http://snomed.info/sct"}'

# Response lists all matching value sets:
# - SepsisDiagnosis (exact match)
# - AUSepsisConditions (exact match)
# → Activate both clinical protocols
```

### Workflow 4: RAF Score Calculation

```bash
# Calculate risk score for Medicare patient

curl -X POST "http://localhost:8087/v1/hcc/raf/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "P12345",
    "age": 72,
    "sex": "F",
    "icd10_codes": ["E1165", "I10", "N183"],
    "model_version": "V28"
  }'
```

---

## Error Handling

### Error Response Format

```json
{
  "error": "Error description",
  "details": "Additional context",
  "code": "ERROR_CODE"
}
```

### Common Error Codes

| HTTP Status | Error | Description |
|-------------|-------|-------------|
| 400 | `Invalid request format` | Malformed JSON or missing required fields |
| 404 | `Value set not found` | Requested value set doesn't exist |
| 500 | `Failed to expand value set` | Database or subsumption service error |
| 503 | `Service unavailable` | Neo4j or GraphDB not connected |

### Example Error Response

```json
{
  "error": "Invalid request format",
  "details": "Key: 'Request.Code' Error:Field validation for 'Code' failed on the 'required' tag",
  "usage": {
    "code": "Required. The SNOMED CT code to validate",
    "system": "Optional. Defaults to http://snomed.info/sct"
  }
}
```

---

## Service Ports Reference

| Service | Port | Description |
|---------|------|-------------|
| KB-7 API | 8087 | Main terminology service |
| Neo4j AU Browser | 7475 | Neo4j HTTP (Australian data) |
| Neo4j AU Bolt | 7688 | Neo4j Bolt protocol |
| GraphDB | 7200 | RDF/OWL reasoning |
| PostgreSQL | 5433 | Value set storage |
| Redis | 6380 | Cache layer |

---

## Quick Reference Card

| Task | Endpoint | Method |
|------|----------|--------|
| Check health | `/health` | GET |
| List value sets | `/v1/rules/valuesets` | GET |
| Validate code | `/v1/rules/valuesets/{id}/validate` | POST |
| Find value sets for code | `/v1/rules/classify` | POST |
| Test IS-A relationship | `/v1/subsumption/test` | POST |
| Get ancestors | `/v1/subsumption/ancestors` | POST |
| Expand value set | `/v1/rules/valuesets/{id}/expand` | POST |
| Seed builtin value sets | `/v1/rules/seed` | POST |

---

**Document Version**: 1.0.0
**Last Updated**: 2025-12-10
**Author**: KB-7 Terminology Service Team
