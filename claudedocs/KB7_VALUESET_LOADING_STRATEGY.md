# KB-7 ValueSet Loading Strategy

## CTO/CMO DIRECTIVE (NON-NEGOTIABLE)

> **"CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."**

| Forbidden | Required |
|-----------|----------|
| Runtime Neo4j traversal during `$expand` | Precomputed expansions stored in PostgreSQL |
| Graph queries at request time | Pure DB reads at runtime (<50ms) |

---

## Why 5% Materialization Rate is CORRECT

The 864 ValueSets (5%) materialized via Neo4j is **expected and correct behavior**.

### The Four Buckets of ValueSets

| Bucket | % | Description | Loading Method |
|--------|---|-------------|----------------|
| **A. Explicit** | ~60-70% | Direct code enumeration in `compose.include.concept[]` | Load from expansion files |
| **B. Refsets** | ~20-25% | Membership lists (^ operator) | Load from expansion files |
| **C. Complex ECL** | ~5-10% | OR/AND/MINUS expressions that cannot reduce to single root | Load from expansion files |
| **D. Hierarchical** | ~4-6% | Pure is-a hierarchy with single root code | Neo4j materialization (the 864) |

### Key Insight

**Ontoserver has already pre-expanded 22,003 ValueSets!**

These expansion files contain the authoritative codes computed by Ontoserver (the official Australian FHIR terminology server). We should **trust and load** these expansions rather than try to re-compute them.

---

## Loading Strategy

### Step 1: Load ALL Expansion Files (95% of ValueSets)

```bash
# Creates load_all_expansions.py that:
# - Iterates through ALL 22,003 expansion files
# - Extracts codes from expansion.contains[]
# - Bulk inserts into precomputed_valueset_codes
# - NO SKIP LOGIC - fresh load

python3 scripts/ontoserver/load_all_expansions.py
```

**Source**: `data/ontoserver-valuesets/expansions/*_expanded.json`

### Step 2: Run Neo4j Materialization for Gaps (5% of ValueSets)

```bash
# For ValueSets with root_code that don't have expansion files
# ~864 ValueSets requiring SNOMED hierarchy traversal

python3 scripts/ontoserver/materialize_expansions.py
```

**Source**: Neo4j graph database with SNOMED ontology

---

## Implementation Pattern Mapping

### Mapping Suggestions to Our Infrastructure

| Suggestion Pattern | Our Infrastructure Implementation |
|-------------------|----------------------------------|
| `value_set_expansions` table | `precomputed_valueset_codes` table |
| `valueset_id` (string) | `valueset_url` (VARCHAR 500) |
| `version` | `snomed_version` |
| `DEFINITIONS_DIR` | `data/ontoserver-valuesets/definitions/` |
| `EXPANSIONS_DIR` | `data/ontoserver-valuesets/expansions/` |
| `materialized_at` | `created_at` |

### Key Design Patterns

#### 1. Expansion-First Iteration (CRITICAL)

```python
# ❌ WRONG: Definition-first (old approach)
for def_file in DEFINITIONS_DIR.glob("*.json"):
    # Try to find expansion... many won't have one

# ✅ CORRECT: Expansion-first (new approach)
for exp_file in EXPANSIONS_DIR.glob("*_expanded.json"):
    # Every file IS an expansion - just load it
    valueset_id = exp_file.stem.replace("_expanded", "")
    codes = extract_codes_from_expansion(exp_file)
    bulk_insert_codes(valueset_url, codes)
```

**Why**: Ontoserver has ALREADY computed 22,003 expansions. We iterate over what we KNOW exists.

#### 2. Idempotency Guard

```python
# Check before inserting (prevents duplicates on re-run)
cursor.execute("""
    SELECT COUNT(*) FROM precomputed_valueset_codes
    WHERE valueset_url = %s AND snomed_version = %s
""", (valueset_url, SNOMED_RELEASE))

if existing_count > 0:
    stats["skipped"] += 1
    continue  # Already loaded
```

**Why**: Safe to re-run loader without duplicates or errors.

#### 3. OID Extraction from Definition

```python
def extract_oid(definition: dict) -> Optional[str]:
    """Extract OID from ValueSet identifier array."""
    for ident in definition.get("identifier", []):
        if "oid" in ident.get("system", "").lower():
            return ident.get("value", "").replace("urn:oid:", "")
    return None
```

**Why**: CQL references by OID, so we need to store OIDs for lookup.

#### 4. No Inference, No Neo4j

```python
# ✅ Pure JSON → PostgreSQL (no reasoning)
codes = expansion.get("expansion", {}).get("contains", [])
for concept in codes:
    insert_code(concept["system"], concept["code"], concept["display"])

# ❌ NO Neo4j traversal in this loader
# ❌ NO ECL evaluation
# ❌ NO hierarchy inference
```

**Why**: Ontoserver did the hard work. We just load their results.

---

## OID Resolution (CRITICAL for CQL)

### Why OID Resolution Matters

CQL files reference ValueSets by OID:
```cql
valueset "Diabetes": 'urn:oid:2.16.840.1.113883.3.464.1003.103.12.1001'
```

At runtime, KB-7 must resolve this OID to return precomputed codes.

### Current Implementation

The `fhir_handlers.go` already supports OID lookup:
```sql
WHERE url = $1 OR name = $2 OR oid = $3
```

### Required Fix

Strip `urn:oid:` prefix before lookup:
```go
// CQL sends: urn:oid:2.16.840.1.113883.3.464...
// DB stores: 2.16.840.1.113883.3.464...

oidLookup := identifier
if strings.HasPrefix(identifier, "urn:oid:") {
    oidLookup = strings.TrimPrefix(identifier, "urn:oid:")
}
```

---

## Database Schema

### Table: `precomputed_valueset_codes`

```sql
CREATE TABLE precomputed_valueset_codes (
    id SERIAL PRIMARY KEY,
    valueset_url VARCHAR(500) NOT NULL,
    valueset_id UUID,
    snomed_version VARCHAR(20) NOT NULL,
    code_system VARCHAR(200) NOT NULL,
    code VARCHAR(50) NOT NULL,
    display VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(valueset_url, snomed_version, code_system, code)
);

CREATE INDEX idx_pvc_url_version ON precomputed_valueset_codes(valueset_url, snomed_version);
CREATE INDEX idx_pvc_code ON precomputed_valueset_codes(code_system, code);
```

---

## Success Criteria

| Criterion | Target | Verification |
|-----------|--------|--------------|
| Total Codes | 500,000+ | `SELECT COUNT(*) FROM precomputed_valueset_codes` |
| ValueSets Loaded | 22,000+ | `SELECT COUNT(DISTINCT valueset_url) FROM precomputed_valueset_codes` |
| OID Resolution | Working | `curl /fhir/ValueSet/urn:oid:XXX/$expand` returns data |
| Response Time | <50ms | `curl -w "%{time_total}"` on $expand endpoint |
| No Runtime Neo4j | 0 calls | Verify logs show only PostgreSQL queries |

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│  BUILD TIME (Deploy/Sync - Neo4j runs HERE, ONCE)               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Source 1: Ontoserver Expansion Files (95%)                     │
│  ────────────────────────────────────────                       │
│  22,003 *_expanded.json files                                   │
│       ↓                                                          │
│  load_all_expansions.py → PostgreSQL                            │
│                                                                  │
│  Source 2: Neo4j Materialization (5%)                           │
│  ────────────────────────────────────                           │
│  864 hierarchical ValueSets with root_code                      │
│       ↓                                                          │
│  materialize_expansions.py → Neo4j traversal → PostgreSQL       │
│                                                                  │
│  Result: precomputed_valueset_codes table                       │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ valueset_url | snomed_version | code_system | code | display│ │
│  │ http://...   | 20241130       | snomed.../sct| 73211009 | ...│ │
│  │ ... 500,000+ precomputed codes ...                         │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│  RUNTIME ($expand call - NO Neo4j!)                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  CQL Executor                                                   │
│       ↓                                                          │
│  GET /fhir/ValueSet/{id}/$expand                                │
│       ↓                                                          │
│  KB-7 fhir_handlers.go                                          │
│       ↓                                                          │
│  PostgreSQL: SELECT * FROM precomputed_valueset_codes           │
│              WHERE valueset_url = ? AND snomed_version = ?      │
│       ↓                                                          │
│  FHIR R4 Response (precomputed codes)                           │
│       ↓                                                          │
│  CQL: Is patient code IN expanded list? → Clinical Decision     │
│                                                                  │
│  Performance: O(1) indexed read, <50ms, deterministic           │
└─────────────────────────────────────────────────────────────────┘
```

---

## File Locations

| File | Purpose |
|------|---------|
| `scripts/ontoserver/load_all_expansions.py` | Load 22,003 expansion files (TO CREATE) |
| `scripts/ontoserver/materialize_expansions.py` | Neo4j materialization for hierarchical (EXISTS) |
| `scripts/ontoserver/load_valuesets_with_roots.py` | Load ValueSet metadata (EXISTS) |
| `scripts/ontoserver/load_explicit_valuesets.py` | Load explicit ValueSets (EXISTS) |
| `internal/api/fhir_handlers.go` | FHIR $expand handler (EXISTS) |
| `migrations/007_valueset_ontoserver.sql` | Schema migration (EXISTS) |

---

## Current State (as of session)

| Component | Status |
|-----------|--------|
| FHIR Routes | ✅ DONE - `/fhir/ValueSet/:id/$expand` working |
| fhir_handlers.go | ✅ DONE - Pure DB read, OID/URL/ID resolution + `urn:oid:` prefix stripping |
| Migration 007 | ✅ DONE - `precomputed_valueset_codes` table exists |
| ValueSet Metadata | ✅ DONE - 18,659 loaded in `value_sets` table |
| Neo4j Materialization | ✅ DONE - 864 intensional ValueSets (CORRECT!) |
| load_all_expansions.py | ✅ CREATED - Expansion-first loader for 22,003 files |
| OID Prefix Stripping | ✅ DONE - `urn:oid:` → raw OID for CQL compatibility |
| Expansion File Loading | 🔄 RUNNING - Loading 22,003 expansion files |

---

## Next Steps

1. **Create `load_all_expansions.py`** - Comprehensive loader for ALL expansion files
2. **Run expansion loader** - Populate precomputed_valueset_codes (~30-60 min)
3. **Run Neo4j materialization** - Fill gaps for pure hierarchical ValueSets
4. **Add OID prefix stripping** - Minor fix in fhir_handlers.go
5. **Verify CQL integration** - Test with OID-based lookup

---

## References

- **Ontoserver**: Official Australian FHIR terminology server (CSIRO)
- **SNOMED Release**: 20241130
- **FHIR R4**: ValueSet/$expand operation specification
- **CQL**: Clinical Quality Language specification
