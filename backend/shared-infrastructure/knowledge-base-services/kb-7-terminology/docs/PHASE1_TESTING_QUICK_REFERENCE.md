# Phase 1 Testing Quick Reference
**KB-7 GraphDB Foundation - Developer Checklist**

---

## Daily Testing Checklist

### Morning Setup
```bash
# 1. Start test environment
docker-compose -f docker-compose.hybrid.yml up -d

# 2. Verify services running
curl http://localhost:7200/rest/repositories      # GraphDB
psql -U kb7_user -h localhost -p 5433 -l         # PostgreSQL
redis-cli -p 6380 ping                           # Redis

# 3. Quick health check
./scripts/validation/infrastructure-check.sh
```

### After Code Changes
```bash
# 1. Run unit tests
cd internal/etl
go test ./...

# 2. Run integration tests (if ETL changes)
go test -tags=integration ./tests/integration/...

# 3. Quick data integrity check (sample)
./scripts/validation/quick-integrity-check.sh
```

### Before Commit
```bash
# 1. Run full test suite
make test

# 2. Check test coverage
go test -cover ./...

# 3. Lint code
golangci-lint run

# 4. Commit with test status
git add .
git commit -m "feat: ETL pipeline - All tests passing"
```

---

## Critical Test Commands

### Infrastructure Tests
```bash
# Full infrastructure validation
./scripts/validation/infrastructure-check.sh

# GraphDB health only
curl http://localhost:7200/rest/repositories/kb7-terminology

# Repository configuration
curl http://localhost:7200/rest/repositories/kb7-terminology/config
```

### ETL Pipeline Tests
```bash
# Test SNOMED conversion (unit)
go test -v ./internal/etl -run TestSNOMEDConversion

# Test RDF conversion (unit)
go test -v ./internal/etl -run TestRDFConversion

# Test batch processing (integration)
go test -v -tags=integration ./tests/integration -run TestBatchProcessing

# Test full 520K migration (long-running)
go test -v -timeout 60m ./tests/etl -run TestFull520KMigration
```

### Data Integrity Tests
```bash
# Quick integrity check (100 samples)
./scripts/validation/quick-integrity-check.sh

# Full integrity validation (520K concepts)
./scripts/validation/data-integrity-check.sh

# Concept count comparison
psql -U kb7_user -d kb7_terminology -c "SELECT COUNT(*) FROM terminology_concepts;"
curl -X POST http://localhost:7200/repositories/kb7-terminology \
  --data-urlencode "query=SELECT (COUNT(*) as ?count) WHERE { ?s a <http://cardiofit.ai/kb7/ontology#ClinicalConcept> }"
```

### SPARQL Query Tests
```bash
# Run all SPARQL tests
go test -v ./tests/sparql/...

# Test specific query type
go test -v ./tests/sparql -run TestBasicConceptLookup
go test -v ./tests/sparql -run TestSubsumptionQuery
go test -v ./tests/sparql -run TestTransitiveClosure
```

### Performance Tests
```bash
# Quick performance spot check
go test -bench=BenchmarkSimpleLookup ./tests/performance/...

# Full performance validation
./scripts/validation/performance-validation.sh

# Detailed benchmarks with profiling
go test -bench=. -benchmem -cpuprofile cpu.prof ./tests/performance/...
```

### Regression Tests
```bash
# Full regression suite
./scripts/validation/regression-validation.sh

# PostgreSQL API check only
curl http://localhost:8092/v1/concepts/SNOMED/387517004

# Elasticsearch check only
curl -X POST http://localhost:9200/terminology/_search \
  -H "Content-Type: application/json" \
  -d '{"query": {"match": {"display": "paracetamol"}}}'
```

---

## Test Pass/Fail Criteria

### Infrastructure (Gate 1)

| Test | Command | Pass Criteria |
|------|---------|---------------|
| GraphDB Health | `curl http://localhost:7200/rest/repositories` | HTTP 200 |
| Repository Exists | `curl .../kb7-terminology` | Repository JSON returned |
| SPARQL Endpoint | Basic SELECT query | Results returned |

### ETL Pipeline (Gate 2)

| Test | Pass Criteria |
|------|---------------|
| SNOMED Conversion | Valid Turtle RDF output |
| RxNorm Conversion | Valid Turtle RDF output |
| LOINC Conversion | Valid Turtle RDF output |
| Batch Processing | No memory errors, progress tracking |
| Full 520K Migration | Completes in < 30 minutes, zero errors |

### Data Integrity (Gate 2)

| Test | Pass Criteria |
|------|---------------|
| Concept Count | PostgreSQL == GraphDB (exact match) |
| Code Integrity | 100% codes found in GraphDB |
| Parent Relationships | 100% preserved |
| No Duplicates | Zero duplicate triples |
| No Orphans | Zero concepts without system |

### SPARQL Queries (Gate 3)

| Test | Pass Criteria |
|------|---------------|
| Basic Lookup | Correct concept returned |
| Subsumption | All children found |
| Transitive Closure | OWL2-RL reasoning working |
| Label Search | Fuzzy match working |
| System Filter | Only requested system returned |

### Performance (Gate 4)

| Query Type | P95 Target | Measurement |
|------------|-----------|-------------|
| Simple Lookup | < 50ms | 100 iterations |
| Subsumption | < 100ms | 100 iterations |
| Transitive | < 200ms | 100 iterations |
| Label Search | < 150ms | 100 iterations |

### Regression (Gate 5)

| Test | Pass Criteria |
|------|---------------|
| PostgreSQL API | HTTP 200, correct data |
| Elasticsearch | Search returns results |
| Redis Cache | Cache hits working |
| Service Health | Status: "healthy" |
| Performance | No degradation > 2x baseline |

---

## Common Issues & Quick Fixes

### Issue: GraphDB Connection Refused
```bash
# Check if GraphDB is running
docker ps | grep graphdb

# Restart GraphDB
docker-compose restart graphdb-reasoning

# Wait for startup (30 seconds)
sleep 30

# Verify
curl http://localhost:7200/rest/repositories
```

### Issue: Repository Not Found
```bash
# Create repository
./scripts/create-graphdb-repository.sh

# Verify creation
curl http://localhost:7200/rest/repositories/kb7-terminology
```

### Issue: PostgreSQL Connection Failed
```bash
# Check PostgreSQL is running
psql -U kb7_user -h localhost -p 5433 -c "SELECT 1;"

# Restart if needed
docker-compose restart postgres-terminology

# Check logs
docker logs kb7-postgres-terminology
```

### Issue: Test Timeouts
```bash
# Increase timeout for long tests
go test -timeout 60m ./tests/etl -run TestFull520KMigration

# Run with verbose output
go test -v -timeout 30m ./tests/integration/...
```

### Issue: Memory Errors During Migration
```bash
# Check batch size configuration
grep -r "batch_size" internal/etl/

# Reduce batch size if needed (edit config)
# Default: 1000, try 500 for large concepts

# Monitor memory during test
watch -n 1 'ps aux | grep etl'
```

### Issue: SPARQL Query Timeout
```bash
# Check query execution plan
# Add EXPLAIN to query in GraphDB Workbench

# Verify indexes exist
curl http://localhost:7200/rest/repositories/kb7-terminology/config

# Increase GraphDB heap if needed (docker-compose.yml)
# graphdb:
#   environment:
#     - GDB_HEAP_SIZE=8g
```

---

## Pre-Deployment Checklist

### Day Before Deployment

- [ ] Run full validation suite: `./scripts/validation/phase1-master-validation.sh`
- [ ] Review all test results: `cat test-results/phase1/validation-report-*.txt`
- [ ] Verify performance benchmarks met
- [ ] Test rollback procedure: `./scripts/validation/test-rollback-procedure.sh`
- [ ] Create production backup: `pg_dump kb7_terminology > backup.sql`
- [ ] Document any known issues

### Deployment Day

- [ ] Run quick validation: `./scripts/validation/quick-validation.sh`
- [ ] Verify all services healthy
- [ ] Check GraphDB heap allocation (production: 8GB+)
- [ ] Monitor migration progress
- [ ] Run post-migration integrity check
- [ ] Validate sample queries
- [ ] Confirm rollback plan ready

### Post-Deployment

- [ ] Monitor query performance (first hour)
- [ ] Check error rates in logs
- [ ] Verify client integrations (Flow2, Medication Service)
- [ ] Run regression tests on production
- [ ] Document actual vs expected metrics

---

## Emergency Rollback

### Quick Rollback Procedure
```bash
# 1. Stop GraphDB queries immediately
# Update config or feature flag to disable GraphDB routes

# 2. Verify PostgreSQL still operational
curl http://localhost:8092/v1/concepts/SNOMED/387517004

# 3. Execute rollback script
./scripts/rollback/emergency-rollback.sh

# 4. Verify service operational
curl http://localhost:8092/health

# 5. Check logs for errors
tail -f logs/kb7-terminology.log

# 6. Notify stakeholders
# Send notification to team
```

---

## Test Result Interpretation

### All Tests Passing
```
Total Test Suites: 6
Passed: 6
Failed: 0
🎉 ALL TESTS PASSED - Phase 1 Complete!
```
**Action**: Proceed to Phase 2 planning

### Partial Failures
```
Total Test Suites: 6
Passed: 4
Failed: 2
❌ VALIDATION FAILED - 2 test suite(s) failed
```
**Action**:
1. Review failed test logs
2. Fix issues
3. Re-run failed suites
4. Do NOT proceed until 100% pass

### Performance Degradation
```
⚠️ WARNING: Query latency 150ms exceeded baseline 50ms
```
**Action**:
1. Analyze slow queries
2. Check GraphDB indexes
3. Review resource utilization
4. Consider scaling resources

---

## Useful SPARQL Queries for Testing

### Count All Concepts
```sparql
SELECT (COUNT(*) as ?count) WHERE {
    ?s a <http://cardiofit.ai/kb7/ontology#ClinicalConcept>
}
```

### Count by System
```sparql
PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
SELECT ?system (COUNT(*) as ?count) WHERE {
    ?s kb7:system ?system
}
GROUP BY ?system
ORDER BY DESC(?count)
```

### Find Concepts Without Parent
```sparql
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>

SELECT ?concept ?label WHERE {
    ?concept a kb7:ClinicalConcept ;
             rdfs:label ?label .
    FILTER NOT EXISTS {
        ?concept rdfs:subClassOf ?parent
    }
}
LIMIT 100
```

### Find Duplicate Triples
```sparql
SELECT ?s ?p ?o (COUNT(*) as ?count) WHERE {
    ?s ?p ?o
}
GROUP BY ?s ?p ?o
HAVING (COUNT(*) > 1)
```

### Check for Orphaned Concepts
```sparql
PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>

SELECT (COUNT(*) as ?count) WHERE {
    ?concept a kb7:ClinicalConcept .
    FILTER NOT EXISTS {
        ?concept kb7:system ?system
    }
}
```

---

## Performance Monitoring

### Real-Time Metrics
```bash
# GraphDB query performance
watch -n 2 'curl -s http://localhost:7200/rest/monitor/infrastructure/query-time'

# PostgreSQL active connections
watch -n 2 'psql -U kb7_user -c "SELECT count(*) FROM pg_stat_activity;"'

# Redis memory usage
watch -n 2 'redis-cli -p 6380 INFO memory | grep used_memory_human'
```

### Log Monitoring
```bash
# GraphDB logs
tail -f /path/to/graphdb/logs/main.log

# Application logs
tail -f logs/kb7-terminology.log | grep ERROR

# ETL progress
tail -f logs/etl-migration.log | grep "Progress:"
```

---

## Contact & Escalation

### Test Failures
- **Technical Lead**: Review test logs, identify root cause
- **Clinical Informaticist**: Validate clinical data integrity
- **DevOps**: Infrastructure and performance issues

### Emergency Rollback
- **Immediate**: Execute rollback procedure
- **Notify**: Technical Lead, Project Manager
- **Document**: Failure reason, rollback timestamp, restoration status

### Performance Issues
- **Monitor**: Query latency trends
- **Analyze**: Slow query logs
- **Escalate**: If P95 > 2x target for > 1 hour

---

**Last Updated**: November 22, 2025
**Version**: 1.0
**Owner**: Quality Engineering Team
