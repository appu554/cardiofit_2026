# KB-7 PostgreSQL Data Loading Guide

**Last Updated:** December 2025
**Purpose:** Load Australian SNOMED CT ValueSets from Ontoserver into PostgreSQL for runtime $expand operations

---

## Overview

KB-7 uses a **build-time materialization** strategy where:
1. ValueSet definitions are loaded into PostgreSQL at build/deploy time
2. Neo4j is queried for hierarchical expansions at build time
3. Runtime `$expand` operations use **pure PostgreSQL reads** (no Neo4j at runtime)

> **CTO/CMO Directive:** *"CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."*

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     KB-7 DATA LOADING ARCHITECTURE                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐  │
│  │   Ontoserver    │────▶│  JSON Files      │────▶│   PostgreSQL    │  │
│  │   (CSIRO)       │     │  (23,706 files)  │     │   value_sets    │  │
│  └─────────────────┘     └──────────────────┘     └─────────────────┘  │
│                                                            │            │
│  ┌─────────────────┐                                       │            │
│  │   Neo4j AU      │◀──────── Root Codes ──────────────────┘            │
│  │   (SNOMED CT)   │                                                    │
│  └────────┬────────┘                                                    │
│           │                                                             │
│           │ subClassOf* traversal                                       │
│           ▼                                                             │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                 precomputed_valueset_codes                       │   │
│  │                 (Runtime $expand source)                         │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## The Two Tables

| Table | Purpose | Source |
|-------|---------|--------|
| `value_sets` | ValueSet metadata, root codes, URLs | Ontoserver JSON files |
| `precomputed_valueset_codes` | Expanded SNOMED codes for $expand | Neo4j AU traversal |

### Table: `value_sets`

```sql
CREATE TABLE value_sets (
    id UUID PRIMARY KEY,
    url VARCHAR NOT NULL,
    version VARCHAR,
    name VARCHAR,
    title VARCHAR,
    description TEXT,
    status VARCHAR,
    publisher VARCHAR,
    compose JSONB,
    root_code VARCHAR,          -- SNOMED root for hierarchy traversal
    root_system VARCHAR,        -- Usually http://snomed.info/sct
    definition_type VARCHAR,    -- 'explicit', 'intensional', 'refset', 'ecl'
    oid VARCHAR,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    UNIQUE(url, version)
);
```

### Table: `precomputed_valueset_codes`

```sql
CREATE TABLE precomputed_valueset_codes (
    id BIGSERIAL PRIMARY KEY,
    valueset_url VARCHAR NOT NULL,
    valueset_id UUID,
    snomed_version VARCHAR NOT NULL,    -- e.g., '20241130'
    code_system VARCHAR NOT NULL,       -- http://snomed.info/sct
    code VARCHAR NOT NULL,
    display VARCHAR,
    materialized_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(valueset_url, snomed_version, code_system, code)
);
```

---

## Prerequisites

### 1. Docker Containers Running

```bash
# Verify containers are running
docker ps | grep -E "kb7-postgres|kb7-neo4j-au"

# Expected output:
# kb7-postgres    Up X hours (healthy)   0.0.0.0:5437->5432/tcp
# kb7-neo4j-au    Up X hours             0.0.0.0:7688->7687/tcp
```

### 2. Neo4j AU Has SNOMED Data

```bash
# Verify Neo4j has SNOMED data (6M+ nodes expected)
docker exec kb7-neo4j-au cypher-shell -u neo4j -p password \
  "MATCH (n:Resource) RETURN count(n) as node_count"
```

### 3. Python Dependencies

```bash
cd scripts/ontoserver
pip install psycopg2-binary neo4j tqdm
```

### 4. Data Source Files Exist

```bash
# Check Ontoserver ValueSet files exist
ls data/ontoserver-valuesets/definitions/ | wc -l
# Expected: 23706
```

---

## Data Loading Steps

### Step 1: Download ValueSets from Ontoserver (if not already done)

```bash
cd scripts/ontoserver

# Download all ValueSets from Ontoserver
python download_valuesets.py

# Output: data/ontoserver-valuesets/definitions/*.json (23,706 files)
```

### Step 2: Load ValueSets into PostgreSQL

This loads ValueSet metadata and extracts root codes for hierarchy traversal.

```bash
cd scripts/ontoserver

# Set environment variables
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5437
export POSTGRES_DB=kb_terminology
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=password

# Load ValueSets (upsert - safe to re-run)
python load_valuesets_with_roots.py

# Options:
#   --dry-run       Preview without changes
#   --filter NAME   Load only ValueSets matching NAME
```

**Expected Output:**
```
IMPORT COMPLETE!
  Files Processed:      23,706
  ValueSets Imported:   12,156
  ValueSets Updated:    0
  Root Code Extraction:
    - With Root Codes:  5,944 (need materialization)
    - Explicit Codes:   10,325 (no Neo4j needed)
```

### Step 3: Materialize Expansions from Neo4j

This queries Neo4j for all descendants of each root code and stores in PostgreSQL.

```bash
cd scripts/ontoserver

# Set environment variables
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5437
export POSTGRES_DB=kb_terminology
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=password
export NEO4J_URI=bolt://localhost:7688
export NEO4J_USER=neo4j
export NEO4J_PASSWORD=password
export SNOMED_RELEASE=20241130

# Materialize all intensional ValueSets
python materialize_expansions.py

# Options:
#   --dry-run           Preview without changes
#   --limit N           Process only N ValueSets (for testing)
#   --valueset NAME     Process specific ValueSet by name
#   --snomed-version    SNOMED release version (default: 20241130)
```

**Expected Output:**
```
MATERIALIZATION COMPLETE!
  SNOMED Release: 20241130
  ValueSets Processed:    5,944
  ValueSets Materialized: 5,500+
  Total Codes Inserted:   60,000+
```

---

## Verification

### Check Table Row Counts

```bash
docker exec kb7-postgres psql -U postgres -d kb_terminology -c "
SELECT 'value_sets' as table_name, COUNT(*) as rows FROM value_sets
UNION ALL
SELECT 'precomputed_valueset_codes', COUNT(*) FROM precomputed_valueset_codes;
"
```

**Expected:**
```
         table_name         | rows
----------------------------+-------
 value_sets                 | 12156
 precomputed_valueset_codes | 60781
```

### Test $expand Endpoint

```bash
# Test ValueSet expansion (should use PostgreSQL only)
curl -s "http://localhost:8092/v1/valuesets/prescribedquantityunit/expand" | jq '.total'
```

### Check Specific ValueSet

```bash
docker exec kb7-postgres psql -U postgres -d kb_terminology -c "
SELECT name, definition_type, root_code
FROM value_sets
WHERE name ILIKE '%sepsis%'
LIMIT 5;
"
```

---

## Scripts Reference

| Script | Purpose | Target Table |
|--------|---------|--------------|
| `download_valuesets.py` | Download from Ontoserver | JSON files |
| `load_valuesets_with_roots.py` | Load metadata + extract roots | `value_sets` |
| `materialize_expansions.py` | Neo4j → PostgreSQL expansion | `precomputed_valueset_codes` |
| `load_concepts_from_neo4j.py` | Load SNOMED/LOINC concepts | `concepts` |
| `import_to_postgres.py` | Alternative loader (legacy) | `value_sets`, `value_set_concepts` |
| `load_explicit_valuesets.py` | Load explicit-only ValueSets | `value_sets` |
| `load_all_expansions.py` | Batch expansion loader | `precomputed_valueset_codes` |

---

## Step 4: Load Concepts from Neo4j (Optional but Recommended)

This loads SNOMED and LOINC concepts from Neo4j AU into the `concepts` table for direct terminology lookups.

```bash
cd scripts/ontoserver

# Set environment variables (same as Step 3)
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5437
export NEO4J_URI=bolt://localhost:7688
# ... other variables as above

# Load all concepts (SNOMED + LOINC)
python load_concepts_from_neo4j.py

# Options:
#   --system snomed    Load only SNOMED concepts
#   --system loinc     Load only LOINC concepts
#   --limit N          Load only N concepts (for testing)
#   --dry-run          Preview without changes
```

**Expected Output:**
```
LOADING COMPLETE!
  Concepts Processed:  ~834,000
  Concepts Inserted:   ~834,000
  - SNOMED:            544,783
  - LOINC:             289,458
```

---

## Troubleshooting

### Error: "definition_type_check" constraint violation

**Cause:** ValueSets with `definition_type = 'unknown'` are rejected.

**Solution:** This is expected. Only ValueSets with valid types (`explicit`, `intensional`, `refset`, `ecl`) can be loaded. Unknown types don't have usable hierarchy definitions.

### Error: Neo4j connection failed

**Check:**
```bash
# Verify Neo4j is running
docker logs kb7-neo4j-au --tail 20

# Test Bolt connection
docker exec kb7-neo4j-au cypher-shell -u neo4j -p password "RETURN 1"
```

### Error: PostgreSQL connection refused

**Check:**
```bash
# Verify correct port (5437 for kb7-postgres)
docker port kb7-postgres

# Test connection
docker exec kb7-postgres psql -U postgres -d kb_terminology -c "SELECT 1"
```

### Empty precomputed_valueset_codes after materialization

**Cause:** Neo4j doesn't have SNOMED data loaded.

**Solution:** Load SNOMED CT AU into Neo4j first using the NCTS RF2 import scripts.

---

## When to Re-run

| Scenario | Action |
|----------|--------|
| New SNOMED CT release | Re-run Steps 2 and 3 with new `SNOMED_RELEASE` |
| New Ontoserver ValueSets | Re-run Steps 1, 2, and 3 |
| Database reset | Re-run Steps 2 and 3 |
| Cache refresh | Re-run Step 3 only |

---

## Environment Variables Summary

```bash
# PostgreSQL (KB7)
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5437
export POSTGRES_DB=kb_terminology
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=password

# Neo4j AU
export NEO4J_URI=bolt://localhost:7688
export NEO4J_USER=neo4j
export NEO4J_PASSWORD=password

# SNOMED Version
export SNOMED_RELEASE=20241130
```

---

## Related Documentation

- [KB7_VERIFICATION_REPORT.md](./KB7_VERIFICATION_REPORT.md) - Service verification status
- [KB7_IMPLEMENTATION_STATUS.md](./KB7_IMPLEMENTATION_STATUS.md) - Implementation details
- [RUNTIME_LAYER_IMPLEMENTATION.md](./RUNTIME_LAYER_IMPLEMENTATION.md) - Runtime architecture

---

*Generated: December 2025*
