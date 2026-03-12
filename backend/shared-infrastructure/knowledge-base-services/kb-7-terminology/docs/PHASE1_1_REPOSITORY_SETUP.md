# Phase 1.1: GraphDB Repository & Infrastructure Setup

**Status**: ✅ COMPLETE
**Date**: November 22, 2025
**Implementation**: KB-7 Terminology Service

---

## Overview

This document details the implementation of Phase 1.1 from the KB7 Architecture Transformation Plan, which establishes GraphDB as the primary semantic storage for the KB-7 Terminology Service.

## Objectives

1. Create GraphDB repository with OWL2-RL reasoning capabilities
2. Configure repository for 2.5M triple capacity
3. Enable clinical ontology indexes (context, literal, predicate)
4. Establish health monitoring and validation scripts
5. Document repository configuration and access patterns

## Repository Specification

### Core Configuration

| Parameter | Value | Purpose |
|-----------|-------|---------|
| Repository ID | `kb7-terminology` | Unique identifier |
| Repository Type | `file-repository` | Persistent storage |
| Ruleset | `owl2-rl-optimized` | OWL2-RL inference for clinical ontologies |
| Base URL | `http://cardiofit.ai/ontology/` | Namespace base for KB-7 ontology |
| Storage Folder | `storage` | Physical storage location |

### Capacity & Performance

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Entity Index Size | 10,000,000 | Supports 2.5M triples with headroom |
| Entity ID Size | 32 bits | Balance between memory and capacity |
| Query Timeout | Unlimited (0) | Clinical queries may be complex |
| Query Result Limit | Unlimited (0) | Full result sets for safety validation |

### Index Configuration

| Index | Status | Purpose |
|-------|--------|---------|
| Context Index | ✅ ENABLED | Named graph versioning and provenance |
| Predicate List | ✅ ENABLED | Fast predicate enumeration for schema discovery |
| Literal Index | ✅ ENABLED | Efficient text search in labels and descriptions |
| In-Memory Literals | ✅ ENABLED | Cache language tags for performance |

### Clinical Safety Settings

| Setting | Value | Purpose |
|---------|-------|---------|
| Consistency Checks | ✅ ENABLED | Detect OWL inconsistencies in clinical data |
| owl:sameAs | ✅ ENABLED | Support concept equivalence reasoning |
| Read-Only | ❌ DISABLED | Allow data loading and updates |
| SHACL Validation | ✅ ENABLED | Shape-based data validation |

## Implementation

### Scripts Created

```
scripts/graphdb/
├── create-repository.sh     # Repository creation with validation
└── health-check.sh           # Operational health verification
```

### Repository Creation

**Script**: `scripts/graphdb/create-repository.sh`

**Features**:
- Pre-flight connectivity checks
- Existing repository detection with confirmation prompt
- Automated repository configuration via REST API
- Post-creation verification with detailed output
- Clean error handling and rollback

**Usage**:
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology
chmod +x scripts/graphdb/create-repository.sh
./scripts/graphdb/create-repository.sh
```

**Expected Output**:
```
=========================================
KB-7 GraphDB Repository Creation
=========================================
GraphDB URL: http://localhost:7200
Repository ID: kb7-terminology
Ruleset: owl2-rl-optimized

✅ GraphDB is accessible
📝 Creating repository configuration...
✅ Configuration file created
🚀 Creating repository via GraphDB API...
   HTTP Status: 201
✅ Repository created successfully!
✅ Repository verification successful
=========================================
✅ GraphDB Repository Setup Complete!
=========================================
```

### Health Check

**Script**: `scripts/graphdb/health-check.sh`

**Validation Tests**:
1. GraphDB service availability
2. Repository existence
3. Repository state (RUNNING/STARTING)
4. Read/write permissions
5. SPARQL endpoint functionality
6. Triple count verification
7. Configuration validation (ruleset, indexes, base URL)
8. Go client connectivity (optional)

**Usage**:
```bash
chmod +x scripts/graphdb/health-check.sh
./scripts/graphdb/health-check.sh
```

**Expected Output**:
```
=========================================
KB-7 GraphDB Health Check
=========================================

1. GraphDB Service... ✓ RUNNING
2. Repository Exists... ✓ FOUND
3. Repository State... ✓ RUNNING
4. Read Permission... ✓ ENABLED
5. Write Permission... ✓ ENABLED
6. SPARQL Endpoint... ✓ OPERATIONAL
   Triples in repository: 0
7. Configuration Validation:
   - Ruleset... ✓ owl2-rl-optimized
   - Base URL... ✓ http://cardiofit.ai/ontology/
   - Context Index... ✓ ENABLED
   - Predicate List... ✓ ENABLED
   - Literal Index... ✓ ENABLED

✅ All systems operational - ready for Phase 1.2
```

## Repository Access

### GraphDB Workbench UI

**URL**: http://localhost:7200

**Features**:
- Visual SPARQL query editor
- Repository statistics and monitoring
- Data import/export tools
- Namespace and prefix management
- Repository configuration viewer

### SPARQL Endpoint

**URL**: http://localhost:7200/repositories/kb7-terminology

**Protocols**:
- SPARQL 1.1 Query (GET/POST)
- SPARQL 1.1 Update (POST)
- SPARQL 1.1 Graph Protocol (GET/PUT/DELETE/POST)

**Example Query**:
```bash
curl -X POST \
  -H "Accept: application/sparql-results+json" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "query=SELECT * WHERE { ?s ?p ?o } LIMIT 10" \
  http://localhost:7200/repositories/kb7-terminology
```

### Go Client Integration

**Client**: `internal/semantic/graphdb_client.go`

**Example Usage**:
```go
package main

import (
    "context"
    "kb-7-terminology/internal/semantic"
    "github.com/sirupsen/logrus"
)

func main() {
    logger := logrus.New()
    client := semantic.NewGraphDBClient(
        "http://localhost:7200",
        "kb7-terminology",
        logger,
    )

    // Test connectivity
    err := client.HealthCheck(context.Background())
    if err != nil {
        logger.Fatal("GraphDB health check failed:", err)
    }

    // Execute SPARQL query
    query := &semantic.SPARQLQuery{
        Query: "SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }",
    }
    results, err := client.ExecuteSPARQL(context.Background(), query)
    if err != nil {
        logger.Fatal("SPARQL query failed:", err)
    }

    logger.Info("Triple count:", results.Results.Bindings[0]["count"].Value)
}
```

## Repository Configuration Details

### Turtle Configuration File

The repository is created using a Turtle (RDF) configuration file with the following structure:

```turtle
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#>.
@prefix rep: <http://www.openrdf.org/config/repository#>.
@prefix sr: <http://www.openrdf.org/config/repository/sail#>.
@prefix sail: <http://www.openrdf.org/config/sail#>.
@prefix owlim: <http://www.ontotext.com/trree/owlim#>.

[] a rep:Repository ;
    rep:repositoryID "kb7-terminology" ;
    rdfs:label "KB-7 Clinical Terminology Repository" ;
    rep:repositoryImpl [
        rep:repositoryType "graphdb:SailRepository" ;
        sr:sailImpl [
            sail:sailType "graphdb:Sail" ;
            owlim:ruleset "owl2-rl-optimized" ;
            owlim:base-URL "http://cardiofit.ai/ontology/" ;
            # ... additional configuration ...
        ]
    ].
```

### REST API Endpoint

**Endpoint**: `POST http://localhost:7200/rest/repositories`

**Content-Type**: `multipart/form-data`

**Form Field**: `config` (Turtle configuration file)

**Response**: HTTP 201 Created with repository details

## Verification & Testing

### Manual Verification Steps

1. **GraphDB Workbench**:
   - Navigate to http://localhost:7200
   - Verify `kb7-terminology` appears in repository list
   - Check repository state is "RUNNING"

2. **SPARQL Test Query**:
```sparql
SELECT * WHERE { ?s ?p ?o } LIMIT 1
```
Expected: Empty result set (0 bindings) - repository is initialized but empty

3. **Configuration Review**:
   - Open repository settings in Workbench
   - Confirm ruleset is "owl2-rl-optimized"
   - Verify indexes are enabled

### Automated Testing

**Go Test**:
```bash
go run test-graphdb-connection.go
```

**Health Check**:
```bash
./scripts/graphdb/health-check.sh
```

## Troubleshooting

### Repository Creation Fails

**Issue**: HTTP 500 or 400 error during creation

**Diagnosis**:
```bash
# Check GraphDB logs
docker logs kb7-graphdb

# Verify GraphDB is running
docker ps | grep graphdb

# Test connectivity
curl http://localhost:7200/rest/repositories
```

**Solutions**:
- Ensure GraphDB container is fully started (may take 30-60 seconds)
- Verify port 7200 is not blocked by firewall
- Check configuration file syntax (Turtle format)

### Repository State "STARTING"

**Issue**: Repository stuck in STARTING state

**Diagnosis**:
```bash
# Check repository state
curl -s http://localhost:7200/rest/repositories | jq '.[] | select(.id == "kb7-terminology") | .state'
```

**Solutions**:
- Wait 30-60 seconds for initialization
- Restart GraphDB container if state persists
- Check GraphDB logs for initialization errors

### SPARQL Queries Fail

**Issue**: 404 or connection refused on SPARQL endpoint

**Diagnosis**:
```bash
# Verify repository exists
curl http://localhost:7200/rest/repositories/kb7-terminology

# Test simple query
curl -X POST \
  --data-urlencode "query=ASK { ?s ?p ?o }" \
  http://localhost:7200/repositories/kb7-terminology
```

**Solutions**:
- Ensure repository state is "RUNNING"
- Verify SPARQL endpoint URL format
- Check repository permissions (readable/writable)

## Next Steps

With Phase 1.1 complete, proceed to:

### Phase 1.2: Extend ETL Pipeline
- Integrate GraphDB client into existing ETL coordinator
- Create RDF conversion layer for PostgreSQL data
- Implement triple store coordinator for dual-write

### Phase 1.3: Data Migration
- Migrate existing 520K concepts from PostgreSQL to GraphDB
- Validate data consistency between stores
- Benchmark query performance

### Phase 1.4: Testing & Validation
- Execute comprehensive test suite
- Validate OWL2-RL inference behavior
- Performance benchmarking with clinical queries

## References

- **Architecture Plan**: `KB7_ARCHITECTURE_TRANSFORMATION_PLAN.md`
- **GraphDB Client**: `internal/semantic/graphdb_client.go`
- **Repository Config**: `semantic/config/kb7-repository-config.ttl`
- **GraphDB Documentation**: https://graphdb.ontotext.com/documentation/10.7/

---

**Implementation Status**: ✅ Complete
**Validated**: November 22, 2025
**Ready for Phase 1.2**: ✅ Yes
