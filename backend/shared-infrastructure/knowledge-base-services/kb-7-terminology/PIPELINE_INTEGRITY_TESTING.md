# KB-7 Pipeline Integrity Testing Guide

> Validates the "nervous system" architecture: GraphDB (Brain) → Kafka (CDC) → Neo4j (Read Replica) → Go API (Face)

## Overview

The Pipeline Integrity Test Suite ensures that when the "Brain" (GraphDB) learns something new, the "Face" (Go API) can speak about it within seconds.

```
┌──────────────────────────────────────────────────────────────────────────┐
│                     CDC PIPELINE ARCHITECTURE                            │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   GraphDB (Brain)     Kafka (CDC)      Neo4j (Replica)    Go API (Face)  │
│   ┌─────────────┐    ┌───────────┐    ┌─────────────┐    ┌───────────┐  │
│   │  OWL Ontology│───▶│  CDC Topic │───▶│  ELK Hier.  │───▶│  REST API │  │
│   │  (Source)   │    │ (kb7.cdc) │    │ (Fast Read) │    │   :8087   │  │
│   └─────────────┘    └───────────┘    └─────────────┘    └───────────┘  │
│         │                                    │                  │       │
│         │◀─────────── "Canary Test" ────────────────────────────│       │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

## Test Types

### 1. Smoke Test (Bash Script)

Quick validation of the complete CDC pipeline. Injects a "canary" concept into GraphDB and waits for it to appear in the Go API.

```bash
# Run the smoke test
./scripts/verify-pipeline-integrity.sh

# Options
./scripts/verify-pipeline-integrity.sh --verbose        # Show detailed output
./scripts/verify-pipeline-integrity.sh --skip-cleanup   # Leave canary for debugging
./scripts/verify-pipeline-integrity.sh --cleanup-only   # Just cleanup old canary
```

**Expected Output:**
```
═══════════════════════════════════════════════════════════════════
    🧪 KB-7 PIPELINE INTEGRITY TEST
═══════════════════════════════════════════════════════════════════

0️⃣  Pre-flight checks...
   ✅ Go API is healthy
   ✅ GraphDB is responding (repo: kb7-terminology)

1️⃣  Injecting Canary Concept into GraphDB...
   ✅ Canary inserted into GraphDB

2️⃣  Waiting for CDC Sync (GraphDB → Kafka → Neo4j → Go API)...
..
   ✅ Canary found in Go API (Neo4j backend)
   ⚡ CDC Latency: 2s

3️⃣  Results:
   ✅ Canary found in Go API (Neo4j backend)
   🚀 Excellent! Sub-2-second sync

═══════════════════════════════════════════════════════════════════
    🎉 TEST PASSED: Brain and Face are Connected!
═══════════════════════════════════════════════════════════════════
```

### 2. Go Integration Tests

Comprehensive tests that can be run in CI/CD pipelines.

```bash
# Run all integration tests
go test -v -tags=integration ./tests/integration/...

# Run specific test
go test -v -run TestEndToEndCDCSync ./tests/integration/...

# Skip in short mode (unit tests only)
go test -short ./...
```

**Test Cases:**

| Test | Description | Target Latency |
|------|-------------|----------------|
| `TestEndToEndCDCSync` | Full CDC pipeline validation | <10s |
| `TestNeo4jBridgeConceptLookup` | Direct concept lookup | <50ms |
| `TestSubsumptionViaNeoBridge` | IS-A hierarchy test | <100ms |

### 3. Operations Verification

Verify all 6 spec-compliant operations are working:

```bash
./scripts/verify_kb7_au.sh
```

## Configuration

### Environment Variables

```bash
# Test endpoints
export GRAPHDB_URL="http://localhost:7200"
export GRAPHDB_REPO="kb7-terminology"
export GO_API_URL="http://localhost:8087"
export NEO4J_URL="bolt://localhost:7688"

# Test parameters
export MAX_WAIT=10  # Seconds to wait for CDC sync
```

### Prerequisites

Ensure all services are running:

```bash
# 1. GraphDB
docker ps | grep graphdb

# 2. Kafka (CDC pipeline)
docker ps | grep kafka

# 3. Neo4j
docker ps | grep neo4j

# 4. KB-7 Go API
curl -s http://localhost:8087/health | jq '.status'
```

## Performance Tuning

### Neo4j Connection Pool

The connection pool size should match expected concurrency:

```
Formula: PoolSize = (RequestsPerSec × AvgQueryTime) + 50% buffer

Examples:
  - Standard API (500 req/s × 20ms = 10) → Pool: 15-20
  - High-traffic (2000 req/s × 20ms = 40) → Pool: 60-100
  - Flink batch (50 parallel threads) → Pool: 50-100
```

Configuration in `neo4j_client.go`:

```go
type Neo4jConfig struct {
    MaxConnections int           // Pool size (default: 100)
    ConnTimeout    time.Duration // Acquisition timeout (default: 10s)
    MaxConnLife    time.Duration // Connection recycling (default: 30min)
}
```

### CDC Latency Targets

| Latency | Rating | Action |
|---------|--------|--------|
| <2s | 🚀 Excellent | Optimal performance |
| 2-5s | 👍 Good | Acceptable |
| 5-10s | ⚠️ Slow | Tune Kafka consumer |
| >10s | ❌ Too slow | Check pipeline health |

## Troubleshooting

### Canary Not Found

If the canary doesn't appear after timeout:

1. **Check Kafka topic:**
   ```bash
   kafka-console-consumer --bootstrap-server localhost:9092 \
     --topic kb7.graphdb.changes --from-beginning --max-messages 5
   ```

2. **Check Neo4j CDC consumer logs:**
   ```bash
   docker logs neo4j-consumer --tail 50
   ```

3. **Verify Neo4j connectivity:**
   ```bash
   curl -s http://localhost:7688/db/neo4j/tx | head -1
   ```

4. **Check Go API logs:**
   ```bash
   grep -i "neo4j" /var/log/kb7/service.log | tail -20
   ```

### GraphDB Insert Failed

```bash
# Check GraphDB health
curl -s http://localhost:7200/rest/repositories/kb7-terminology/size

# Check repository exists
curl -s http://localhost:7200/rest/repositories
```

### Neo4j Connection Issues

```bash
# Verify Neo4j is responding
cypher-shell -a bolt://localhost:7688 -u neo4j -p kb7aupassword "RETURN 1"

# Check connection pool stats
curl -s http://localhost:8087/v1/subsumption/config | jq '.neo4j_stats'
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests
on: [push, pull_request]

jobs:
  pipeline-test:
    runs-on: ubuntu-latest
    services:
      graphdb:
        image: ontotext/graphdb:10.5.0
        ports: ["7200:7200"]
      neo4j:
        image: neo4j:5.15.0
        ports: ["7687:7687"]
      kafka:
        image: confluentinc/cp-kafka:7.5.0
        ports: ["9092:9092"]

    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Start KB-7 Service
        run: |
          make build
          ./server &
          sleep 10

      - name: Run Integration Tests
        run: go test -v -tags=integration ./tests/integration/...

      - name: Run Smoke Test
        run: ./scripts/verify-pipeline-integrity.sh
```

### Makefile Integration

```makefile
.PHONY: test-integration test-pipeline

test-integration:
	go test -v -tags=integration ./tests/integration/...

test-pipeline:
	./scripts/verify-pipeline-integrity.sh

test-all: test test-integration test-pipeline
```

## Files Reference

| File | Purpose |
|------|---------|
| `scripts/verify-pipeline-integrity.sh` | CDC pipeline smoke test |
| `scripts/verify_kb7_au.sh` | 6 operations verification |
| `tests/integration/cdc_sync_test.go` | Go integration tests |
| `internal/semantic/neo4j_client.go` | Neo4j client with pool tuning |

---

## Summary

The Pipeline Integrity Test Suite provides confidence that:

1. ✅ GraphDB changes propagate through Kafka CDC
2. ✅ Neo4j read replica receives updates within seconds
3. ✅ Go API serves fresh data from Neo4j
4. ✅ Connection pools are properly tuned for production load

Run these tests before every deployment to ensure the "Brain" and "Face" remain connected.
