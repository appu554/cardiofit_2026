# KB-7 Bootstrap Migration: PostgreSQL to GraphDB

This directory contains the bootstrap migration script for Phase 1.2 of the KB-7 GraphDB transformation. The script performs a one-time migration of existing PostgreSQL concepts to GraphDB to enable Phase 2 development.

## Overview

**Purpose**: Migrate 520K concepts from PostgreSQL to GraphDB as RDF/Turtle triples

**Architecture**:
```
PostgreSQL (port 5433)
    ↓
  Batch Reader (1000 concepts/batch)
    ↓
  RDF Converter (Turtle format)
    ↓
  GraphDB Loader (HTTP API)
    ↓
GraphDB Repository (kb7-terminology)
```

## Prerequisites

1. **PostgreSQL Database**: Running on port 5433 with kb7_terminology database
2. **GraphDB Server**: Running on port 7200 with kb7-terminology repository created
3. **Go Environment**: Go 1.21+ installed
4. **Environment Variables** (optional):
   ```bash
   export DATABASE_URL="postgresql://kb7_user:kb7_password@localhost:5433/kb7_terminology"
   export GRAPHDB_URL="http://localhost:7200"
   export GRAPHDB_REPOSITORY="kb7-terminology"
   ```

## Quick Start

### 1. Test with Small Batch (10 concepts)

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology

# Test migration with just 10 concepts
go run scripts/bootstrap/postgres-to-graphdb.go \
  --max 10 \
  --batch 10 \
  --log-interval 5
```

**Expected Output**:
```
INFO[2025-11-24T11:00:00Z] Starting PostgreSQL to GraphDB migration
INFO[2025-11-24T11:00:01Z] Connecting to PostgreSQL...
INFO[2025-11-24T11:00:01Z] Connecting to GraphDB...
INFO[2025-11-24T11:00:01Z] GraphDB connection successful
INFO[2025-11-24T11:00:02Z] Migration plan prepared
INFO[2025-11-24T11:00:02Z] Migration progress                           migrated=10 progress=100.00% triples=70
INFO[2025-11-24T11:00:03Z] Validating migration...
INFO[2025-11-24T11:00:03Z] Validation results                           postgresql_count=10 graphdb_count=10 match=true
INFO[2025-11-24T11:00:03Z] === Migration Complete ===                   migrated=10 total_triples=70 duration=3s
INFO[2025-11-24T11:00:03Z] Migration completed successfully
```

### 2. Dry Run (Full Migration Plan)

```bash
go run scripts/bootstrap/postgres-to-graphdb.go --dry-run
```

This shows migration plan without executing:
- Total concept count
- Estimated execution time
- Database connection validation

### 3. Full Migration (520K concepts)

**WARNING**: This will take 2-4 hours. Run in tmux/screen session.

```bash
# Start tmux session
tmux new -s kb7-migration

# Run full migration
go run scripts/bootstrap/postgres-to-graphdb.go

# Detach: Ctrl+B, then D
# Reattach: tmux attach -t kb7-migration
```

## Command-Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `--postgres` | `DATABASE_URL` env or default | PostgreSQL connection string |
| `--graphdb` | `GRAPHDB_URL` env or `http://localhost:7200` | GraphDB base URL |
| `--repo` | `GRAPHDB_REPOSITORY` env or `kb7-terminology` | GraphDB repository name |
| `--batch` | `1000` | Number of concepts per batch |
| `--log-interval` | `10000` | Log progress every N concepts |
| `--dry-run` | `false` | Show plan without executing |
| `--start` | `0` | Start offset (for resuming failed migration) |
| `--max` | `0` | Max concepts to migrate (0 = all) |

## Advanced Usage

### Resume Failed Migration

If migration fails at offset 250000:

```bash
go run scripts/bootstrap/postgres-to-graphdb.go --start 250000
```

### Partial Migration (Testing)

Migrate only concepts 1000-2000:

```bash
go run scripts/bootstrap/postgres-to-graphdb.go --start 1000 --max 2000
```

### Custom Batch Size (Performance Tuning)

```bash
# Smaller batches (more reliable, slower)
go run scripts/bootstrap/postgres-to-graphdb.go --batch 500

# Larger batches (faster, more memory)
go run scripts/bootstrap/postgres-to-graphdb.go --batch 2000
```

## Validation

The script automatically validates migration by:

1. **Concept Count**: PostgreSQL count == GraphDB count
2. **Triple Integrity**: All concepts have required properties
3. **Context Verification**: All triples loaded to bootstrap context

### Manual Validation

Query GraphDB to verify concepts:

```bash
curl -X POST http://localhost:7200/repositories/kb7-terminology \
  --data-urlencode "query=PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
SELECT (COUNT(?concept) AS ?count) WHERE {
  ?concept a kb7:ClinicalConcept .
}"
```

Sample concept lookup:

```bash
curl -X POST http://localhost:7200/repositories/kb7-terminology \
  --data-urlencode "query=PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
SELECT ?code ?label WHERE {
  ?concept a kb7:ClinicalConcept ;
    kb7:code ?code ;
    rdfs:label ?label .
} LIMIT 10"
```

## RDF/Turtle Format

Each concept is converted to this structure:

```turtle
@prefix kb7: <http://cardiofit.ai/kb7/ontology#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .

<http://cardiofit.ai/kb7/concepts/123e4567-e89b-12d3-a456-426614174000> a kb7:ClinicalConcept ;
    kb7:code "387517004" ;
    kb7:system "SNOMED-CT" ;
    rdfs:label "Paracetamol" ;
    skos:definition "Analgesic and antipyretic medication" ;
    kb7:clinicalDomain "pharmacology" ;
    kb7:specialty "pain-management" ;
    kb7:status "active" .
```

**Triple Count per Concept**: ~7 triples (type + 6 properties)

## Performance Expectations

### Hardware Requirements

- **RAM**: 4GB minimum, 8GB recommended
- **Disk**: 10GB free space (for GraphDB indices)
- **Network**: Local (PostgreSQL and GraphDB on same machine)

### Timing Estimates

| Concepts | Batch Size | Estimated Time |
|----------|------------|----------------|
| 10 | 10 | 3-5 seconds |
| 1,000 | 1000 | 30-60 seconds |
| 10,000 | 1000 | 5-10 minutes |
| 100,000 | 1000 | 50-90 minutes |
| 520,000 | 1000 | 2-4 hours |

**Factors**:
- GraphDB performance (OWL reasoning disabled for bootstrap)
- PostgreSQL query speed
- Network latency (minimal if local)

### Monitoring Progress

Watch real-time stats:

```bash
# In separate terminal
watch -n 5 'curl -s -X POST http://localhost:7200/repositories/kb7-terminology \
  --data-urlencode "query=SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }" | \
  jq -r ".results.bindings[0].count.value"'
```

## Troubleshooting

### Issue: "GraphDB health check failed"

**Solution**: Verify GraphDB is running and repository exists
```bash
curl http://localhost:7200/rest/repositories
```

### Issue: "Failed to connect to PostgreSQL"

**Solution**: Check connection string and database status
```bash
psql postgresql://kb7_user:kb7_password@localhost:5433/kb7_terminology -c "SELECT COUNT(*) FROM terminology_concepts;"
```

### Issue: "Failed to load batch to GraphDB"

**Causes**:
- Invalid Turtle syntax (check logs for specific concept)
- GraphDB memory exhaustion (restart GraphDB)
- Network timeout (reduce batch size)

**Recovery**:
```bash
# Resume from last successful batch
go run scripts/bootstrap/postgres-to-graphdb.go --start <OFFSET>
```

### Issue: "Validation failed - count mismatch"

**Investigation**:
```bash
# Check for failed batches in logs
grep "Failed to load batch" migration.log

# Query GraphDB for gaps
curl -X POST http://localhost:7200/repositories/kb7-terminology \
  --data-urlencode "query=SELECT ?code WHERE { ?s kb7:code ?code } ORDER BY ?code LIMIT 100"
```

## Post-Migration

### 1. Verify Migration Success

```bash
# Run test script
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology
go run scripts/bootstrap/test-migration.go
```

### 2. Backup GraphDB Repository

```bash
# Export repository (for rollback)
curl -X GET http://localhost:7200/repositories/kb7-terminology/statements \
  -H "Accept: application/x-turtlestar" \
  > kb7-bootstrap-backup-$(date +%Y%m%d).ttl
```

### 3. Enable Phase 2 Development

Migration complete! Now you can:
- Develop SPARQL endpoints (Phase 2.2)
- Test semantic queries
- Begin Knowledge Factory integration (Phase 1.3)

## Important Notes

### Data Lifecycle

This bootstrap data will be **replaced** by Knowledge Factory kernel in Phase 1.3.4:
- Bootstrap context: `http://cardiofit.ai/bootstrap` (temporary)
- Production context: `http://cardiofit.ai/kernels/v1.0.0` (from Knowledge Factory)

### Cleanup

After Knowledge Factory activation, remove bootstrap context:

```bash
curl -X DELETE "http://localhost:7200/repositories/kb7-terminology/statements?context=http://cardiofit.ai/bootstrap"
```

### Do Not Re-Run

This is a **one-time migration**. After successful completion:
- GraphDB becomes source of truth for concepts
- PostgreSQL transitions to metadata registry role (Phase 3)
- Future updates via Knowledge Factory pipeline (Phase 1.3)

## Support

**Questions?** Contact: kb7-architecture@cardiofit.ai

**Issues?** Check logs and validation output before reporting.

---

**Migration Status**: Ready for execution
**Last Updated**: November 24, 2025
**Phase**: 1.2 Bootstrap GraphDB
