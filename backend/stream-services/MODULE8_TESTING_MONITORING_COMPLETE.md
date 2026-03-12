# Module 8 Testing and Monitoring Suite - Complete

**Status**: ✅ Complete
**Created**: 2025-11-15
**Components**: Integration tests, benchmarks, monitoring, load tests, test data generation

---

## Overview

Comprehensive testing suite and monitoring setup for all 8 Module 8 storage projectors:
1. PostgreSQL Projector
2. MongoDB Projector
3. Elasticsearch Projector
4. ClickHouse Projector
5. InfluxDB Projector
6. UPS Read Model Projector
7. FHIR Store Projector
8. Neo4j Graph Projector

---

## 📦 Components Created

### 1. Integration Test Suite
**File**: `test-module8-integration.py`

Comprehensive end-to-end testing covering:

#### A. End-to-End Flow Tests
- **test_enriched_event_fanout**: Verify single event reaches all 6 storage systems
- **test_fhir_resource_projection**: Validate FHIR resource projection to Google FHIR Store
- **test_graph_mutation_execution**: Confirm graph mutations create nodes in Neo4j

#### B. Performance Tests
- **test_batch_processing_performance**: Measure throughput with 1000 events
  - PostgreSQL: >30 events/sec
  - MongoDB: >25 events/sec
  - Elasticsearch: >30 events/sec
  - ClickHouse: >30 events/sec
- **test_query_latency**: Validate query performance targets
  - PostgreSQL: <100ms
  - MongoDB: <200ms
  - Elasticsearch: <200ms
  - UPS: <50ms

#### C. Data Consistency Tests
- **test_data_consistency_across_stores**: Verify data matches across PostgreSQL and UPS
- **test_upsert_idempotency**: Confirm duplicate events handled correctly
  - PostgreSQL: 1 row (ON CONFLICT)
  - MongoDB: 1 document (upsert)
  - Elasticsearch: 1 document
  - ClickHouse: 5 rows (append-only)
  - UPS: 1 row

#### D. Error Handling Tests
- **test_dlq_routing**: Validate invalid events routed to DLQ
- **test_prometheus_metrics**: Verify all projectors expose required metrics

**Usage**:
```bash
# Install dependencies
pip install pytest kafka-python psycopg2-binary pymongo elasticsearch-py clickhouse-driver influxdb-client neo4j google-cloud-healthcare

# Run all tests
pytest test-module8-integration.py -v

# Run specific test category
pytest test-module8-integration.py -v -k "test_enriched"

# Run with detailed output
pytest test-module8-integration.py -v --tb=short --log-cli-level=INFO
```

**Expected Output**:
```
test_enriched_event_fanout PASSED              [12%]
test_fhir_resource_projection PASSED           [25%]
test_graph_mutation_execution PASSED           [37%]
test_batch_processing_performance PASSED       [50%]
test_query_latency PASSED                      [62%]
test_data_consistency_across_stores PASSED     [75%]
test_upsert_idempotency PASSED                 [87%]
test_dlq_routing PASSED                        [100%]

==================== 8 passed in 120.45s ====================
```

---

### 2. Performance Benchmark Suite
**File**: `benchmark-module8.py`

Automated performance benchmarking with multiple test scenarios:

#### Benchmark Types
1. **Throughput Tests**: Varying batch sizes (100, 500, 1K, 5K, 10K)
2. **Scaling Tests**: Multiple partition counts (1, 2, 4, 8)
3. **Sustained Load**: 30K events continuous processing
4. **Resource Monitoring**: CPU, memory, disk I/O tracking

#### Metrics Collected
- Events per second (throughput)
- Latency percentiles (p50, p95, p99)
- CPU usage per projector
- Memory consumption
- Disk I/O rates

#### Outputs Generated
1. **CSV Results**: `benchmark-results.csv`
   - Detailed results for every test run
   - Suitable for analysis in Excel/Python

2. **Markdown Report**: `BENCHMARK_REPORT.md`
   - Summary statistics
   - Throughput by projector
   - Scaling test results
   - Detailed result tables

3. **Grafana Dashboard**: `grafana-dashboard-benchmark.json`
   - Pre-configured dashboard
   - Throughput, latency, resource usage panels

**Usage**:
```bash
# Install dependencies
pip install psutil kafka-python

# Run full benchmark suite
python benchmark-module8.py

# Results files created:
# - benchmark-results.csv
# - BENCHMARK_REPORT.md
# - grafana-dashboard-benchmark.json
```

**Example Report Output**:
```markdown
## Summary Statistics

| Metric | Value |
|--------|-------|
| Total Events Processed | 50,000 |
| Average Throughput | 1,247 events/sec |
| Max Throughput | 3,456 events/sec |
| Average p95 Latency | 34.56 ms |

## Throughput by Projector

| Projector | Avg Throughput (events/sec) | p95 Latency (ms) |
|-----------|------------------------------|------------------|
| elasticsearch | 3,456 | 12.34 |
| clickhouse | 2,987 | 15.67 |
| postgresql | 1,234 | 34.56 |
| mongodb | 987 | 45.78 |
```

---

### 3. Monitoring Setup
**Directory**: `monitoring/`

#### A. Prometheus Configuration
**File**: `monitoring/prometheus.yml`

Scrapes metrics from all 8 projectors every 15 seconds:
- Module 8 projectors (ports 8050-8057)
- Kafka JMX metrics
- Database exporters (PostgreSQL, MongoDB, Elasticsearch, ClickHouse, Neo4j)
- Node exporter for system metrics

**Setup**:
```bash
# Copy configuration
cp monitoring/prometheus.yml /path/to/prometheus/

# Start Prometheus
prometheus --config.file=prometheus.yml
```

**Access**: http://localhost:9090

#### B. Alerting Rules
**File**: `monitoring/alerts-module8.yml`

Comprehensive alerting for:

**Consumer Lag Alerts**:
- HighConsumerLag: >1000 messages for 5 min (Warning)
- CriticalConsumerLag: >10000 messages for 5 min (Critical)

**Error Rate Alerts**:
- HighErrorRate: >5% for 5 min (Warning)
- CriticalErrorRate: >20% for 2 min (Critical)

**Service Health Alerts**:
- ServiceDown: Service unavailable for 2 min (Critical)
- HealthCheckFailing: Health endpoint failing (Critical)

**Performance Alerts**:
- SlowBatchProcessing: p95 >5s for 5 min (Warning)
- LowThroughput: <10 msg/sec for 10 min (Warning)

**Database Alerts**:
- DatabaseConnectionPoolExhaustion: >80% for 5 min (Warning)
- DatabaseConnectionErrors: >1/sec for 5 min (Warning)
- SlowDatabaseQueries: p95 >1s for 5 min (Warning)

**Resource Alerts**:
- HighMemoryUsage: >80% for 10 min (Warning)
- HighCPUUsage: >80% for 10 min (Warning)

**Data Quality Alerts**:
- DataInconsistency: PostgreSQL/UPS count differs by >100 (Warning)
- DLQBacklog: >1000 unprocessed for 30 min (Warning)

#### C. Grafana Dashboard
**File**: `monitoring/grafana-dashboard-module8.json`

Pre-built dashboard with 15 panels:

**Panel Overview**:
1. **Throughput**: Events processed per second by projector
2. **Consumer Lag**: Real-time lag monitoring
3. **Batch Processing Latency**: p95 and p99 percentiles
4. **Error Rate**: Failed messages per projector
5. **CPU Usage**: Resource consumption
6. **Memory Usage**: Memory per projector
7. **Database Query Latency**: p95 query duration
8. **Total Events Processed**: 1-hour totals
9. **Average Throughput**: System-wide average
10. **Total Errors**: 1-hour error counts
11. **Services Up**: Health status count
12. **Projector Status Summary**: Comprehensive table
13. **Connection Pool Usage**: Database pool metrics
14. **DLQ Messages Rate**: Dead letter queue flow
15. **Event Flow Diagram**: Sankey visualization

**Import**:
```bash
# Copy dashboard JSON
cp monitoring/grafana-dashboard-module8.json /path/to/grafana/

# Import via Grafana UI:
# Dashboards -> Import -> Upload JSON file
```

**Access**: http://localhost:3000

---

### 4. Smoke Test Script
**File**: `smoke-test-module8.sh`

Quick validation script for rapid testing:

#### Test Flow
1. **Health Checks**: Verify all 8 services respond to /health
2. **Publish Events**: Send 10 test events to Kafka
3. **Wait**: 30 seconds for processing
4. **Verify Counts**: Check events in each storage system
5. **Error Check**: Scan logs for errors

**Usage**:
```bash
# Make executable
chmod +x smoke-test-module8.sh

# Run smoke test
./smoke-test-module8.sh

# Expected output:
# ==========================================
# MODULE 8 SMOKE TEST
# ==========================================
#
# 1. HEALTH CHECKS
# ----------------------------------------
# Checking postgresql... ✓ UP
# Checking mongodb... ✓ UP
# Checking elasticsearch... ✓ UP
# Checking clickhouse... ✓ UP
# Checking influxdb... ✓ UP
# Checking ups... ✓ UP
# Checking fhir-store... ✓ UP
# Checking neo4j... ✓ UP
#
# 2. PUBLISHING TEST EVENTS
# ----------------------------------------
# Publishing 10 events for patient: smoke-test-1731686400
# ..........
# Published 10 events
#
# 3. WAITING FOR PROCESSING
# ----------------------------------------
# Waiting 30 seconds for projectors to process events...
# Wait complete
#
# 4. VERIFYING EVENT COUNTS
# ----------------------------------------
# PostgreSQL: ✓ 10 events
# MongoDB: ✓ 10 events
# Elasticsearch: ✓ 10 events
# ClickHouse: ✓ 10 events
# UPS Read Model: ✓ 1 patient record
#
# 5. CHECKING ERROR LOGS
# ----------------------------------------
# No errors found
#
# ==========================================
# SMOKE TEST SUMMARY
# ==========================================
# ✓ ALL TESTS PASSED
#
# Services healthy: 8/8
# Events published: 10
# Stores verified: 5/5
# Errors: 0
#
# Module 8 is functioning correctly
```

**Exit Codes**:
- 0: All tests passed
- 1: Tests failed (check logs)

---

### 5. Load Test Configuration
**File**: `locustfile-module8.py`

Locust-based load testing for realistic traffic simulation:

#### Features
- **Multiple Publishers**: Simulate 100 concurrent event sources
- **Realistic Traffic Mix**:
  - 70% enriched clinical events
  - 20% FHIR resources
  - 10% graph mutations
- **Event Types**: VITAL_SIGNS, LAB_RESULT, MEDICATION, etc.
- **Ramp-up Pattern**: Gradual increase to avoid shock
- **Sustained Load**: 30-minute continuous testing

#### Metrics Tracked
- Request rate (events/sec)
- Response time (p50, p95, p99)
- Failure rate
- Throughput degradation

**Usage**:
```bash
# Install Locust
pip install locust kafka-python

# Run load test (headless mode)
locust -f locustfile-module8.py --headless -u 100 -r 10 -t 30m

# Run with web UI
locust -f locustfile-module8.py

# Then open: http://localhost:8089
```

**Command Options**:
- `-u 100`: 100 concurrent users
- `-r 10`: Spawn 10 users per second
- `-t 30m`: Run for 30 minutes
- `--headless`: No web UI (automated)

**Expected Output**:
```
================================================================================
MODULE 8 LOAD TEST STARTING
================================================================================
Users: 100
Spawn rate: 10
Run time: 30m
================================================================================

...

================================================================================
MODULE 8 LOAD TEST COMPLETE
================================================================================
SUMMARY STATISTICS:
  Total requests: 180,000
  Failed requests: 45
  Failure rate: 0.03%
  Average response time: 23.45ms
  p95 response time: 67.89ms
  p99 response time: 123.45ms
  Requests per second: 100.23
================================================================================
```

---

### 6. Test Data Generator
**File**: `generate-test-data.py`

Realistic clinical event generator for testing and development:

#### Features
- **Event Types**: All 7 clinical event types
- **Realistic Values**: Age-based, condition-based adjustments
- **Time Distribution**: Spread events over configurable time range
- **Patient Profiles**: Consistent patient demographics and conditions
- **Output Modes**:
  - Kafka topics (real-time)
  - JSON files (replay scenarios)

#### Event Types Generated
1. **VITAL_SIGNS**: Heart rate, BP, temperature, SpO2, respiratory rate
2. **LAB_RESULT**: Glucose, creatinine, sodium, potassium, CBC
3. **MEDICATION_ADMINISTRATION**: Metformin, Lisinopril, Atorvastatin, etc.
4. **DIAGNOSTIC_PROCEDURE**: X-Ray, CT, MRI, Ultrasound, ECG
5. **CLINICAL_NOTE**: Progress notes, consultations, discharge summaries
6. **ALERT**: Critical, warning, info alerts
7. **DEVICE_READING**: Device-specific measurements

**Usage**:

```bash
# Generate to Kafka (real-time)
python generate-test-data.py --kafka --patients 100 --events-per-patient 50

# Generate to JSON files (for replay)
python generate-test-data.py --output ./test-data --patients 10 --events-per-patient 20

# Generate historical data
python generate-test-data.py --kafka --start-date 2024-01-01 --end-date 2024-12-31 --patients 50 --events-per-patient 100

# Reproducible data with seed
python generate-test-data.py --kafka --patients 10 --events-per-patient 20 --seed 42
```

**Command Options**:
- `--kafka`: Output to Kafka topics
- `--output <dir>`: Output to JSON files
- `--kafka-bootstrap <servers>`: Kafka bootstrap servers (default: localhost:9092)
- `--patients <n>`: Number of patients (default: 10)
- `--events-per-patient <n>`: Events per patient (default: 20)
- `--start-date <YYYY-MM-DD>`: Start date (default: 7 days ago)
- `--end-date <YYYY-MM-DD>`: End date (default: today)
- `--seed <n>`: Random seed for reproducibility

**Example Output**:
```
2025-11-15 10:23:45 - INFO - Generating data for 100 patients, 50 events each
2025-11-15 10:23:45 - INFO - Time range: 2024-01-01 to 2024-12-31
2025-11-15 10:23:46 - INFO - Generating events for patient 1/100: test-patient-000000
2025-11-15 10:23:50 - INFO - Generated 100 events...
2025-11-15 10:23:55 - INFO - Generated 200 events...
...
2025-11-15 10:35:12 - INFO - ✅ Generated 5000 total events
```

---

## 🚀 Quick Start Guide

### 1. Prerequisites
```bash
# Python dependencies
pip install pytest kafka-python psycopg2-binary pymongo elasticsearch-py \
    clickhouse-driver influxdb-client neo4j google-cloud-healthcare \
    psutil locust

# System tools (macOS)
brew install kafkacat postgresql@15 mongodb-community clickhouse

# Or use Docker for databases
cd monitoring
docker-compose up -d
```

### 2. Start Infrastructure
```bash
# Start all Module 8 infrastructure
./manage-module8-infrastructure.sh start

# Verify services
./manage-module8-infrastructure.sh status
```

### 3. Start Projectors
```bash
# Start all 8 projectors
docker-compose -f docker-compose.module8-services.yml up -d

# Check health
for port in 8050 8051 8052 8053 8054 8055 8056 8057; do
    curl http://localhost:$port/health
done
```

### 4. Run Smoke Test
```bash
# Quick validation
./smoke-test-module8.sh
```

### 5. Run Integration Tests
```bash
# Full test suite
pytest test-module8-integration.py -v
```

### 6. Start Monitoring
```bash
# Start Prometheus
prometheus --config.file=monitoring/prometheus.yml

# Start Grafana
grafana-server --config=/path/to/grafana.ini

# Import dashboard
# Open http://localhost:3000
# Dashboards -> Import -> Upload monitoring/grafana-dashboard-module8.json
```

### 7. Generate Test Data
```bash
# Generate 1000 events for testing
python generate-test-data.py --kafka --patients 10 --events-per-patient 100
```

### 8. Run Load Test
```bash
# 30-minute load test with 100 concurrent users
locust -f locustfile-module8.py --headless -u 100 -r 10 -t 30m
```

### 9. Run Benchmarks
```bash
# Full benchmark suite
python benchmark-module8.py

# Review results
cat BENCHMARK_REPORT.md
```

---

## 📊 Monitoring Best Practices

### Daily Operations

1. **Check Grafana Dashboard**
   - Review throughput trends
   - Monitor consumer lag
   - Check error rates

2. **Review Alerts**
   - Investigate any triggered alerts
   - Update thresholds as needed

3. **Run Smoke Test**
   - Quick daily validation
   - Ensure all services healthy

### Weekly Operations

1. **Run Integration Tests**
   - Verify data consistency
   - Check performance targets

2. **Review Metrics**
   - Analyze throughput trends
   - Identify optimization opportunities

3. **Check DLQ**
   - Review dead letter queue
   - Fix any recurring issues

### Monthly Operations

1. **Run Benchmarks**
   - Track performance over time
   - Identify degradation trends

2. **Load Testing**
   - Validate system capacity
   - Plan for scaling needs

3. **Review Alerts**
   - Tune alert thresholds
   - Add new alerts as needed

---

## 🐛 Troubleshooting

### Smoke Test Failures

**Service Down**:
```bash
# Check service logs
docker logs module8-postgresql-projector

# Restart service
docker-compose -f docker-compose.module8-services.yml restart postgresql-projector

# Check health
curl http://localhost:8050/health
```

**Event Count Mismatch**:
```bash
# Check consumer lag
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
    --describe --group module8-postgresql-projector

# Check projector metrics
curl http://localhost:8050/metrics | grep consumer_lag

# Check database directly
psql -h localhost -U postgres -d clinical_events \
    -c "SELECT COUNT(*) FROM enriched_events WHERE patient_id = 'test-patient-id';"
```

### Integration Test Failures

**Database Connection Issues**:
```bash
# Check database connectivity
psql -h localhost -U postgres -d clinical_events -c "SELECT 1;"
mongosh --eval "db.adminCommand('ping')"
curl http://localhost:9200/_cluster/health

# Verify credentials in test config
grep -A 5 "class TestConfig" test-module8-integration.py
```

**Kafka Connection Issues**:
```bash
# Check Kafka is running
kafka-broker-api-versions.sh --bootstrap-server localhost:9092

# Verify topics exist
kafka-topics.sh --bootstrap-server localhost:9092 --list

# Check consumer groups
kafka-consumer-groups.sh --bootstrap-server localhost:9092 --list
```

### Performance Issues

**High Consumer Lag**:
```bash
# Scale up partitions
kafka-topics.sh --bootstrap-server localhost:9092 \
    --alter --topic prod.ehr.events.enriched --partitions 8

# Increase projector replicas
docker-compose -f docker-compose.module8-services.yml \
    scale postgresql-projector=2

# Check resource usage
docker stats module8-postgresql-projector
```

**Slow Queries**:
```bash
# PostgreSQL slow query log
psql -h localhost -U postgres -d clinical_events \
    -c "SELECT query, mean_exec_time FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"

# Add indexes
psql -h localhost -U postgres -d clinical_events \
    -c "CREATE INDEX CONCURRENTLY idx_enriched_events_patient_time ON enriched_events(patient_id, event_time);"
```

---

## 📈 Performance Targets

### Throughput Targets
| Projector | Target (events/sec) | Notes |
|-----------|---------------------|-------|
| PostgreSQL | >30 | Transactional store |
| MongoDB | >25 | Document store |
| Elasticsearch | >30 | Search/analytics |
| ClickHouse | >100 | Analytics OLAP |
| InfluxDB | >100 | Time-series |
| UPS | >50 | Read model |
| FHIR Store | >20 | Google API limits |
| Neo4j | >30 | Graph queries |

### Latency Targets
| Operation | Target | Notes |
|-----------|--------|-------|
| Batch Processing | <5s p95 | End-to-end batch |
| Database Insert | <100ms p95 | Single insert |
| Database Query | <1s p95 | Single patient |
| Kafka Publish | <50ms p95 | Producer latency |

### Resource Targets
| Resource | Target | Notes |
|----------|--------|-------|
| CPU Usage | <80% avg | Per projector |
| Memory Usage | <80% max | Total system |
| Disk I/O | <80% max | Database servers |
| Network | <70% max | Kafka bandwidth |

---

## 📝 Test Coverage Summary

### Integration Tests
- ✅ End-to-end event flow (all 6 stores)
- ✅ FHIR resource projection
- ✅ Graph mutation execution
- ✅ Batch processing performance
- ✅ Query latency validation
- ✅ Data consistency across stores
- ✅ Upsert idempotency
- ✅ DLQ routing
- ✅ Prometheus metrics exposure

### Performance Tests
- ✅ Throughput benchmarks (varying batch sizes)
- ✅ Scaling tests (1, 2, 4, 8 partitions)
- ✅ Sustained load (30K events)
- ✅ Resource monitoring (CPU, memory, disk)
- ✅ Latency percentiles (p50, p95, p99)

### Load Tests
- ✅ Concurrent publishers (100 users)
- ✅ Mixed traffic (enriched, FHIR, graph)
- ✅ Realistic event distribution
- ✅ Ramp-up pattern
- ✅ Sustained load (30 minutes)

### Monitoring
- ✅ Prometheus metrics scraping
- ✅ Alerting rules (18 alerts)
- ✅ Grafana dashboard (15 panels)
- ✅ Health checks
- ✅ Error tracking

---

## ✅ Completion Checklist

- [x] Integration test suite (`test-module8-integration.py`)
- [x] Performance benchmark suite (`benchmark-module8.py`)
- [x] Prometheus configuration (`monitoring/prometheus.yml`)
- [x] Alerting rules (`monitoring/alerts-module8.yml`)
- [x] Grafana dashboard (`monitoring/grafana-dashboard-module8.json`)
- [x] Smoke test script (`smoke-test-module8.sh`)
- [x] Load test configuration (`locustfile-module8.py`)
- [x] Test data generator (`generate-test-data.py`)
- [x] Documentation (this file)

---

## 🎯 Next Steps

1. **Run Initial Tests**:
   ```bash
   ./smoke-test-module8.sh
   pytest test-module8-integration.py -v
   ```

2. **Set Up Monitoring**:
   - Import Grafana dashboard
   - Configure alert notifications
   - Set up on-call rotation

3. **Establish Baseline**:
   - Run benchmarks
   - Document baseline performance
   - Set SLAs based on results

4. **Continuous Testing**:
   - Schedule daily smoke tests
   - Weekly integration tests
   - Monthly benchmarks and load tests

5. **Iterate and Improve**:
   - Tune alert thresholds
   - Optimize slow queries
   - Scale based on metrics

---

## 📚 Additional Resources

- **Module 8 Documentation**: `MODULE8_QUICKSTART.md`
- **Infrastructure Setup**: `MODULE8_INFRASTRUCTURE_README.md`
- **Projector Documentation**: `MODULE8_*_PROJECTOR_COMPLETE.md`
- **Prometheus Docs**: https://prometheus.io/docs/
- **Grafana Docs**: https://grafana.com/docs/
- **Locust Docs**: https://docs.locust.io/

---

**Testing Suite Complete** ✅

All components created and ready for use. Follow the Quick Start Guide to begin testing and monitoring Module 8 storage projectors.
