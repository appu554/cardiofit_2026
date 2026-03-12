# GraphDB Repository Management Scripts

Scripts for managing the kb7-terminology GraphDB repository.

## Prerequisites

- GraphDB container running at localhost:7200
- curl and jq installed
- Bash shell

## Quick Start

```bash
# Create repository
./create-repository.sh

# Verify health
./health-check.sh

# Validate functionality
./validate-repository.sh
```

## Scripts

### create-repository.sh

Creates the kb7-terminology GraphDB repository with optimized configuration.

**Usage**:
```bash
./create-repository.sh
```

**Features**:
- Pre-flight connectivity checks
- Existing repository detection
- Automated configuration via REST API
- Post-creation verification

**Output**: HTTP 201 Created

### health-check.sh

Comprehensive health check for repository operational status.

**Usage**:
```bash
./health-check.sh
```

**Validates**:
- GraphDB service availability
- Repository existence and state
- Read/write permissions
- SPARQL endpoint functionality
- Configuration parameters

**Exit Codes**:
- 0: All checks passed
- 1: One or more checks failed

### validate-repository.sh

Functional validation suite testing SPARQL capabilities.

**Usage**:
```bash
./validate-repository.sh
```

**Tests**:
1. Data insertion (INSERT DATA)
2. Data retrieval (SELECT)
3. Aggregation (COUNT)
4. Named graphs (GRAPH)
5. FILTER operations
6. Data deletion (DELETE WHERE)
7. Deletion verification

**Exit Codes**:
- 0: All tests passed
- 1: One or more tests failed

## Environment Variables

```bash
GRAPHDB_URL=http://localhost:7200   # GraphDB base URL
```

## Troubleshooting

**Repository creation fails**:
```bash
# Check GraphDB is running
docker ps | grep graphdb

# Check connectivity
curl http://localhost:7200/rest/repositories

# View logs
docker logs kb7-graphdb
```

**Health check fails**:
```bash
# Verify repository exists
curl http://localhost:7200/rest/repositories | jq '.'

# Check repository state
./health-check.sh
```

## See Also

- [Phase 1.1 Setup Documentation](../../docs/PHASE1_1_REPOSITORY_SETUP.md)
- [Implementation Report](../../PHASE1_1_IMPLEMENTATION_REPORT.md)
- [Architecture Plan](../../KB7_ARCHITECTURE_TRANSFORMATION_PLAN.md)
