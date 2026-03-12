# KB-7 AU Terminology Service - Operations Guide

> Complete guide for starting and using the KB-7 Terminology Service with all 6 spec-compliant operations for Australia (AU) region.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [The 6 Spec-Compliant Operations](#the-6-spec-compliant-operations)
- [API Reference](#api-reference)
- [Testing & Verification](#testing--verification)
- [Troubleshooting](#troubleshooting)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     KB-7 AU Terminology Service                              │
│                          Port: 8087                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐         │
│  │  Concept Lookup │    │   Subsumption   │    │   HCC Mapping   │         │
│  │     <50ms       │    │     <100ms      │    │    <200ms       │         │
│  │  Neo4j Index    │    │  ELK Hierarchy  │    │  Graph Traversal│         │
│  └────────┬────────┘    └────────┬────────┘    └────────┬────────┘         │
│           │                      │                      │                   │
│           └──────────────────────┼──────────────────────┘                   │
│                                  ▼                                          │
│                    ┌─────────────────────────┐                              │
│                    │   Neo4j AU Database     │                              │
│                    │   bolt://localhost:7688 │                              │
│                    │   ELK Materialized      │                              │
│                    │   rdfs__subClassOf      │                              │
│                    └─────────────────────────┘                              │
│                                                                              │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐         │
│  │  Ancestor Query │    │ Value Set Expand│    │  HCC Batch Map  │         │
│  │     <300ms      │    │     <500ms      │    │    <400ms       │         │
│  │  IS-A Traversal │    │ PostgreSQL+Redis│    │  Parallel Query │         │
│  └────────┬────────┘    └────────┬────────┘    └────────┬────────┘         │
│           │                      │                      │                   │
│           ▼                      ▼                      ▼                   │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐         │
│  │     Neo4j AU    │    │   PostgreSQL    │    │     Neo4j AU    │         │
│  │  Hierarchy Walk │    │  Port: 5432     │    │  Batch Mapping  │         │
│  └─────────────────┘    └─────────────────┘    └─────────────────┘         │
│                                  │                                          │
│                                  ▼                                          │
│                    ┌─────────────────────────┐                              │
│                    │   Redis Cache           │                              │
│                    │   Port: 6379            │                              │
│                    │   30min TTL             │                              │
│                    └─────────────────────────┘                              │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Data Sources

| Component | Purpose | Connection |
|-----------|---------|------------|
| **Neo4j AU** | SNOMED AU concepts, ELK hierarchy | `bolt://localhost:7688` |
| **Neo4j US** | SNOMED US concepts (fallback) | `bolt://localhost:7687` |
| **PostgreSQL** | Value sets, rules, metadata | `localhost:5432` |
| **Redis** | Response caching, session | `localhost:6379` |
| **GraphDB** | SPARQL fallback (optional) | `localhost:7200` |

---

## Prerequisites

### Required Services

```bash
# 1. Neo4j AU Database (with SNOMED AU + ELK hierarchy)
docker run -d \
  --name neo4j-au \
  -p 7688:7687 \
  -e NEO4J_AUTH=neo4j/kb7aupassword \
  -v neo4j-au-data:/data \
  neo4j:5.15.0

# 2. PostgreSQL (for value sets and rules)
docker run -d \
  --name kb7-postgres \
  -p 5432:5432 \
  -e POSTGRES_USER=kb_user \
  -e POSTGRES_PASSWORD=kb_password \
  -e POSTGRES_DB=clinical_governance \
  postgres:15

# 3. Redis (for caching)
docker run -d \
  --name kb7-redis \
  -p 6379:6379 \
  redis:7-alpine
```

### Verify Services

```bash
# Check Neo4j AU
curl -s http://localhost:7688/db/neo4j/tx | head -1

# Check PostgreSQL
psql -h localhost -U kb_user -d clinical_governance -c "SELECT 1"

# Check Redis
redis-cli ping
```

---

## Quick Start

### Option 1: Using Make (Recommended)

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-7-terminology

# Start the service
make run

# Or with Docker
make docker-run
```

### Option 2: Direct Go Run

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-7-terminology

# Load environment and run
source .env && go run ./cmd/server
```

### Option 3: Build and Run Binary

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-7-terminology

# Build
go build -o server ./cmd/server

# Run
source .env && ./server
```

### Verify Service Started

```bash
# Health check
curl -s http://localhost:8087/health | jq

# Expected output:
# {
#   "status": "healthy",
#   "database": { "status": "connected" },
#   "redis": { "status": "connected" },
#   "graphdb": { "status": "connected" }
# }

# Check subsumption backend
curl -s http://localhost:8087/v1/subsumption/config | jq '.preferred_backend'
# Expected: "neo4j"
```

---

## Configuration

### Environment Variables (.env)

```bash
# Server
PORT=8087
ENVIRONMENT=development
LOG_LEVEL=4

# PostgreSQL
DATABASE_URL=postgresql://kb_user:kb_password@localhost:5432/clinical_governance

# Redis
REDIS_URL=redis://localhost:6379/7

# GraphDB (optional fallback)
GRAPHDB_ENABLED=true
GRAPHDB_URL=http://localhost:7200
GRAPHDB_REPOSITORY=kb7-terminology

# Multi-Region Neo4j (Phase 7)
NEO4J_MULTI_REGION_ENABLED=true
NEO4J_DEFAULT_REGION=us

# Australia Region (Primary for AU)
NEO4J_AU_URL=bolt://localhost:7688
NEO4J_AU_USERNAME=neo4j
NEO4J_AU_PASSWORD=kb7aupassword
NEO4J_AU_DATABASE=neo4j
NEO4J_AU_ENABLED=true

# US Region (Fallback)
NEO4J_US_URL=bolt://localhost:7687
NEO4J_US_USERNAME=neo4j
NEO4J_US_PASSWORD=password
NEO4J_US_DATABASE=neo4j
NEO4J_US_ENABLED=true

# Value Set Seeding (first run only)
SEED_BUILTIN_VALUE_SETS=false
```

### Region Priority

The service automatically selects the AU region when available:

```
Priority: AU > Default Region > Any Enabled Region
```

---

## The 6 Spec-Compliant Operations

### Overview

| # | Operation | Latency Target | Purpose | Backend |
|---|-----------|----------------|---------|---------|
| 1 | **Concept Lookup** | <50ms | Silent Translator - code→display | Neo4j O(1) Index |
| 2 | **Subsumption Check** | <100ms | Is-A Logic - parent/child test | Neo4j ELK Hierarchy |
| 3 | **HCC Single Mapping** | <200ms | ICD→HCC risk category | Neo4j Graph |
| 4 | **HCC Batch Mapping** | <400ms | Problem list→RAF score | Neo4j Parallel |
| 5 | **Ancestor Query** | <300ms | Population health grouper | Neo4j IS-A Walk |
| 6 | **Value Set Expansion** | <500ms | HEDIS reporting codes | PostgreSQL+Redis |

---

### 1️⃣ Concept Lookup (Silent Translator)

**Purpose**: Translate SNOMED codes to human-readable display names in <50ms.

**Use Case**: Display "Diabetes mellitus type 2 (disorder)" instead of code "44054006" in clinical UI.

```bash
# Request
curl -s http://localhost:8087/v1/concepts/SNOMED/44054006 | jq

# Response
{
  "code": "44054006",
  "system": "SNOMED",
  "display": "Diabetes mellitus type 2 (disorder)",
  "active": true,
  "backend": "neo4j"
}
```

**Technical Details**:
- Uses Neo4j index lookup on `code` property
- Handles n10s array format for `rdfs__label`
- Falls back to PostgreSQL if Neo4j unavailable

---

### 2️⃣ Subsumption Check (Is-A Logic)

**Purpose**: Test if concept A is a subtype of concept B using OWL reasoning in <100ms.

**Use Case**: "Is Type 2 Diabetes a kind of Diabetes Mellitus?" → Yes

```bash
# Request
curl -s -X POST http://localhost:8087/v1/subsumption/test \
  -H "Content-Type: application/json" \
  -d '{
    "code_a": "44054006",
    "code_b": "73211009",
    "system": "SNOMED"
  }' | jq

# Response
{
  "subsumes": true,
  "code_a": "44054006",
  "code_b": "73211009",
  "relationship": "is_subsumed_by",
  "backend": "neo4j",
  "reasoning_method": "elk_materialized"
}
```

**Technical Details**:
- Uses pre-computed ELK hierarchy in Neo4j (`rdfs__subClassOf` relationships)
- No runtime OWL reasoning required
- Falls back to GraphDB SPARQL if Neo4j unavailable

---

### 3️⃣ HCC Single Mapping

**Purpose**: Map a single ICD-10 code to HCC risk category in <200ms.

**Use Case**: "What HCC category is E11.9 (Type 2 DM without complications)?" → HCC 19

```bash
# Request
curl -s -X POST http://localhost:8087/v1/hcc/map \
  -H "Content-Type: application/json" \
  -d '{
    "icd_code": "E11.9",
    "model_year": "2024"
  }' | jq

# Response
{
  "icd_code": "E11.9",
  "hcc_category": "HCC 19",
  "hcc_description": "Diabetes without Complication",
  "coefficient": 0.105,
  "model_year": "2024"
}
```

---

### 4️⃣ HCC Batch Mapping

**Purpose**: Map multiple ICD codes to HCC categories with RAF score in <400ms.

**Use Case**: Calculate risk adjustment factor for patient's entire problem list.

```bash
# Request
curl -s -X POST http://localhost:8087/v1/hcc/batch \
  -H "Content-Type: application/json" \
  -d '{
    "icd_codes": ["E11.9", "I10", "J44.9", "F32.9"],
    "model_year": "2024",
    "demographics": {
      "age": 67,
      "sex": "M",
      "medicaid": false,
      "disabled": false
    }
  }' | jq

# Response
{
  "mappings": [
    {"icd_code": "E11.9", "hcc": "HCC 19", "coefficient": 0.105},
    {"icd_code": "I10", "hcc": null, "coefficient": 0},
    {"icd_code": "J44.9", "hcc": "HCC 111", "coefficient": 0.335},
    {"icd_code": "F32.9", "hcc": "HCC 59", "coefficient": 0.309}
  ],
  "raf_score": 1.249,
  "hierarchies_applied": ["HCC 19 trumps HCC 18"],
  "model_year": "2024"
}
```

---

### 5️⃣ Ancestor Query (Grouper)

**Purpose**: Get all ancestors of a concept for population health grouping in <300ms.

**Use Case**: Find all parent categories of "Type 2 Diabetes" for cohort analysis.

```bash
# Request
curl -s -X POST http://localhost:8087/v1/subsumption/ancestors \
  -H "Content-Type: application/json" \
  -d '{
    "code": "44054006",
    "system": "SNOMED",
    "max_depth": 20
  }' | jq

# Response
{
  "code": "44054006",
  "system": "SNOMED",
  "ancestors": [
    {"code": "73211009", "display": "Diabetes mellitus", "depth": 1},
    {"code": "126877002", "display": "Disorder of glucose metabolism", "depth": 2},
    {"code": "75934005", "display": "Metabolic disease", "depth": 3},
    {"code": "64572001", "display": "Disease", "depth": 4}
  ],
  "total_ancestors": 11,
  "backend": "neo4j"
}
```

**Technical Details**:
- Walks `rdfs__subClassOf` relationships in Neo4j
- Limited by `max_depth` parameter (default: 20)
- Includes depth level for each ancestor

---

### 6️⃣ Value Set Expansion

**Purpose**: Expand value set to member codes for HEDIS/quality reporting in <500ms.

**Use Case**: Get all codes in "Diabetes Value Set" for measure calculation.

```bash
# List all value sets
curl -s http://localhost:8087/v1/rules/valuesets | jq

# Response
{
  "value_sets": [
    {"id": "diabetes-mellitus", "name": "Diabetes Mellitus", "count": 156},
    {"id": "hypertension", "name": "Hypertension", "count": 89},
    {"id": "heart-failure", "name": "Heart Failure", "count": 234}
  ],
  "total": 18
}

# Expand specific value set
curl -s http://localhost:8087/v1/rules/valuesets/diabetes-mellitus | jq

# Response
{
  "id": "diabetes-mellitus",
  "name": "Diabetes Mellitus",
  "version": "2024.1",
  "codes": [
    {"code": "44054006", "system": "SNOMED", "display": "Type 2 diabetes mellitus"},
    {"code": "46635009", "system": "SNOMED", "display": "Type 1 diabetes mellitus"},
    {"code": "E11.9", "system": "ICD10", "display": "Type 2 DM without complications"}
  ],
  "total_codes": 156,
  "cached": true
}
```

**Technical Details**:
- 18 FHIR R4 value sets stored in PostgreSQL
- Redis caching with 30-minute TTL
- Database-driven (no hardcoded value sets)

---

## API Reference

### Base URL

```
http://localhost:8087
```

### Endpoints Summary

| Method | Endpoint | Operation |
|--------|----------|-----------|
| GET | `/health` | Health check |
| GET | `/v1/concepts/{system}/{code}` | Concept Lookup |
| POST | `/v1/subsumption/test` | Subsumption Check |
| POST | `/v1/hcc/map` | HCC Single Mapping |
| POST | `/v1/hcc/batch` | HCC Batch Mapping |
| POST | `/v1/subsumption/ancestors` | Ancestor Query |
| GET | `/v1/rules/valuesets` | List Value Sets |
| GET | `/v1/rules/valuesets/{id}` | Expand Value Set |
| GET | `/v1/subsumption/config` | Backend Configuration |

### Health Check

```bash
curl -s http://localhost:8087/health | jq
```

### Subsumption Configuration

```bash
curl -s http://localhost:8087/v1/subsumption/config | jq
```

---

## Testing & Verification

### Complete Verification Script

```bash
#!/bin/bash
# verify_kb7_au.sh - Verify all 6 KB-7 AU operations

echo "═══════════════════════════════════════════════════════════════════"
echo "    KB-7 AU TERMINOLOGY SERVICE - VERIFICATION"
echo "═══════════════════════════════════════════════════════════════════"

BASE_URL="http://localhost:8087"

# 1. Concept Lookup
echo -e "\n1️⃣ CONCEPT LOOKUP (<50ms)"
START=$(date +%s%3N)
RESULT=$(curl -s "$BASE_URL/v1/concepts/SNOMED/44054006")
END=$(date +%s%3N)
LATENCY=$((END - START))
DISPLAY=$(echo $RESULT | jq -r '.display // "?"')
if [ "$LATENCY" -lt 50 ] && [ "$DISPLAY" != "?" ]; then
  echo "   ✅ ${LATENCY}ms PASS - Display: $DISPLAY"
else
  echo "   ❌ ${LATENCY}ms FAIL - Display: $DISPLAY"
fi

# 2. Subsumption Check
echo -e "\n2️⃣ SUBSUMPTION CHECK (<100ms)"
START=$(date +%s%3N)
RESULT=$(curl -s -X POST "$BASE_URL/v1/subsumption/test" \
  -H "Content-Type: application/json" \
  -d '{"code_a": "44054006", "code_b": "73211009", "system": "SNOMED"}')
END=$(date +%s%3N)
LATENCY=$((END - START))
SUBSUMES=$(echo $RESULT | jq -r '.subsumes')
BACKEND=$(echo $RESULT | jq -r '.backend // "unknown"')
if [ "$LATENCY" -lt 100 ] && [ "$SUBSUMES" = "true" ]; then
  echo "   ✅ ${LATENCY}ms PASS - subsumes=$SUBSUMES, backend=$BACKEND"
else
  echo "   ❌ ${LATENCY}ms FAIL - subsumes=$SUBSUMES"
fi

# 3. HCC Single Mapping
echo -e "\n3️⃣ HCC SINGLE MAPPING (<200ms)"
START=$(date +%s%3N)
RESULT=$(curl -s -X POST "$BASE_URL/v1/hcc/map" \
  -H "Content-Type: application/json" \
  -d '{"icd_code": "E11.9", "model_year": "2024"}')
END=$(date +%s%3N)
LATENCY=$((END - START))
if [ "$LATENCY" -lt 200 ]; then
  echo "   ✅ ${LATENCY}ms PASS"
else
  echo "   ❌ ${LATENCY}ms FAIL"
fi

# 4. HCC Batch Mapping
echo -e "\n4️⃣ HCC BATCH MAPPING (<400ms)"
START=$(date +%s%3N)
RESULT=$(curl -s -X POST "$BASE_URL/v1/hcc/batch" \
  -H "Content-Type: application/json" \
  -d '{"icd_codes": ["E11.9", "I10", "J44.9"], "model_year": "2024"}')
END=$(date +%s%3N)
LATENCY=$((END - START))
if [ "$LATENCY" -lt 400 ]; then
  echo "   ✅ ${LATENCY}ms PASS"
else
  echo "   ❌ ${LATENCY}ms FAIL"
fi

# 5. Ancestor Query
echo -e "\n5️⃣ ANCESTOR QUERY (<300ms)"
START=$(date +%s%3N)
RESULT=$(curl -s -X POST "$BASE_URL/v1/subsumption/ancestors" \
  -H "Content-Type: application/json" \
  -d '{"code": "44054006", "system": "SNOMED", "max_depth": 20}')
END=$(date +%s%3N)
LATENCY=$((END - START))
ANCESTORS=$(echo $RESULT | jq -r '.total_ancestors // .ancestors | length')
if [ "$LATENCY" -lt 300 ]; then
  echo "   ✅ ${LATENCY}ms PASS - Ancestors: $ANCESTORS"
else
  echo "   ❌ ${LATENCY}ms FAIL"
fi

# 6. Value Set Expansion
echo -e "\n6️⃣ VALUE SET EXPANSION (<500ms)"
START=$(date +%s%3N)
RESULT=$(curl -s "$BASE_URL/v1/rules/valuesets")
END=$(date +%s%3N)
LATENCY=$((END - START))
VS_COUNT=$(echo $RESULT | jq -r '.value_sets | length')
if [ "$LATENCY" -lt 500 ] && [ "$VS_COUNT" -gt 0 ]; then
  echo "   ✅ ${LATENCY}ms PASS - Value Sets: $VS_COUNT (FHIR R4)"
else
  echo "   ❌ ${LATENCY}ms FAIL - Value Sets: $VS_COUNT"
fi

echo -e "\n═══════════════════════════════════════════════════════════════════"
echo "    VERIFICATION COMPLETE"
echo "═══════════════════════════════════════════════════════════════════"
```

### Run Verification

```bash
chmod +x verify_kb7_au.sh
./verify_kb7_au.sh
```

### Expected Output

```
═══════════════════════════════════════════════════════════════════
    KB-7 AU TERMINOLOGY SERVICE - VERIFICATION
═══════════════════════════════════════════════════════════════════

1️⃣ CONCEPT LOOKUP (<50ms)
   ✅ 37ms PASS - Display: Diabetes mellitus type 2 (disorder)

2️⃣ SUBSUMPTION CHECK (<100ms)
   ✅ 28ms PASS - subsumes=true, backend=neo4j

3️⃣ HCC SINGLE MAPPING (<200ms)
   ✅ 29ms PASS

4️⃣ HCC BATCH MAPPING (<400ms)
   ✅ 27ms PASS

5️⃣ ANCESTOR QUERY (<300ms)
   ✅ 30ms PASS - Ancestors: 11

6️⃣ VALUE SET EXPANSION (<500ms)
   ✅ 42ms PASS - Value Sets: 18 (FHIR R4)

═══════════════════════════════════════════════════════════════════
    ✅ ALL 6 OPERATIONS VERIFIED - KB-7 AU VERSION COMPLETE!
═══════════════════════════════════════════════════════════════════
```

---

## Troubleshooting

### Common Issues

#### 1. Display Name Shows "?"

**Symptom**: Concept lookup returns `"display": "?"` instead of actual name.

**Cause**: Neo4j stores `rdfs__label` as an array (n10s import format).

**Fix**: Ensure `getStringValue()` in `neo4j_client.go` handles arrays:

```go
// Fixed: Handles both string and []interface{} types
if arr, ok := val.([]interface{}); ok && len(arr) > 0 {
    if str, ok := arr[0].(string); ok {
        return str
    }
}
```

#### 2. Subsumption Returns "graphdb" Backend

**Symptom**: `"backend": "graphdb"` instead of `"neo4j"`.

**Cause**: Neo4j AU not configured or not available.

**Fix**: Check `.env` configuration:

```bash
NEO4J_MULTI_REGION_ENABLED=true
NEO4J_AU_URL=bolt://localhost:7688
NEO4J_AU_ENABLED=true
```

#### 3. Value Sets Return 0

**Symptom**: Value set list returns empty.

**Cause**: Value sets not seeded to PostgreSQL.

**Fix**: Run with seeding enabled (first time only):

```bash
SEED_BUILTIN_VALUE_SETS=true go run ./cmd/server
```

#### 4. Service Won't Start

**Symptom**: Port 8087 already in use.

**Fix**: Kill existing process:

```bash
lsof -ti:8087 | xargs kill -9
```

#### 5. Neo4j Connection Failed

**Symptom**: Log shows "Neo4j connection failed".

**Fix**: Verify Neo4j AU is running:

```bash
# Check container
docker ps | grep neo4j-au

# Check connection
curl -s http://localhost:7688/db/neo4j/tx
```

### Logs Location

```bash
# Service logs (stdout)
go run ./cmd/server 2>&1 | tee kb7.log

# Filter for errors
grep -i error kb7.log

# Filter for Neo4j
grep -i neo4j kb7.log
```

---

## Files Reference

| File | Purpose |
|------|---------|
| `cmd/server/main.go` | Service entry point, region selection |
| `internal/api/server.go` | HTTP handlers, Neo4jBridge integration |
| `internal/semantic/neo4j_client.go` | Neo4j client, `getStringValue()` fix |
| `internal/services/neo4j_bridge.go` | Bridge service for fast operations |
| `internal/config/config.go` | Configuration loading, multi-region |
| `.env` | Environment configuration |

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2024-12 | Initial AU region support |
| 1.1.0 | 2024-12 | Fixed display name array handling |
| 1.2.0 | 2024-12 | Added AU region preference logic |

---

## Contact

For issues or questions about KB-7 Terminology Service:
- Check existing documentation in `/docs` folder
- Review troubleshooting section above
- Check service logs for detailed error messages
