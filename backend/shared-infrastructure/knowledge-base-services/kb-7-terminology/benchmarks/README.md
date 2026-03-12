# KB7 Terminology Phase 3.5 Performance Benchmarking System
## Comprehensive Validation and Testing Suite

This benchmarking system provides comprehensive validation for all KB7 Terminology Phase 3.5 success criteria with realistic workloads, concurrent load testing, and detailed performance analysis.

## 🎯 Phase 3.5 Success Criteria Validated

- ✅ **PostgreSQL lookup queries <10ms (95th percentile)**
- ✅ **GraphDB reasoning queries <50ms (95th percentile)**
- ✅ **Query router uptime >99.9%**
- ✅ **Data migration completed with 100% integrity**
- ✅ **FHIR endpoints using hybrid architecture**
- ✅ **Cache hit ratio >90% for frequent operations**
- ✅ **Performance monitoring and alerting operational**

## 📁 Directory Structure

```
benchmarks/
├── phase35_validation.py      # Main Phase 3.5 success criteria validation
├── performance_tests.py       # Performance testing with realistic workloads
├── load_testing.py           # Concurrent load testing and capacity planning
├── google_fhir_benchmark.py  # Google FHIR integration performance
├── run_all_tests.py          # Test orchestrator with comprehensive reporting
├── test_config.yaml          # Comprehensive configuration file
├── requirements.txt          # Python dependencies
├── Makefile                  # Automation commands
└── README.md                 # This file
```

## 🚀 Quick Start

### 1. Environment Setup

```bash
# Install dependencies
pip install -r requirements.txt

# Ensure all services are running
docker-compose up -d  # From kb-7-terminology root
```

### 2. Run Complete Test Suite

```bash
# Run all Phase 3.5 validation tests
python run_all_tests.py

# Run specific test categories
python run_all_tests.py --skip-load --skip-google-fhir

# Run with custom configuration
python run_all_tests.py --config custom_config.yaml
```

### 3. Individual Test Execution

```bash
# Phase 3.5 validation only
python phase35_validation.py --output reports/phase35_results.json

# Performance testing
python performance_tests.py --workload realistic_mixed --duration 300

# Load testing
python load_testing.py --scenario stress_test

# Google FHIR testing
python google_fhir_benchmark.py --iterations 200
```

## 🔧 Configuration

The system uses `test_config.yaml` for comprehensive configuration:

```yaml
# Database connections
postgresql:
  host: localhost
  port: 5433
  database: terminology_db

# Performance targets (Phase 3.5 criteria)
performance_targets:
  postgresql_p95_ms: 10
  graphdb_p95_ms: 50
  router_uptime_pct: 99.9

# Test parameters
test_parameters:
  test_requests: 1000
  concurrent_users: 10
  test_duration_seconds: 300
```

## 📊 Test Types and Workloads

### 1. Phase 3.5 Validation (`phase35_validation.py`)

**Purpose**: Validates all success criteria with pass/fail results

**Tests**:
- PostgreSQL query performance validation
- Neo4j reasoning query validation
- Query router uptime testing
- Data migration integrity checks
- FHIR endpoint hybrid architecture validation
- Cache hit ratio validation
- Monitoring system validation

**Usage**:
```bash
python phase35_validation.py
```

### 2. Performance Testing (`performance_tests.py`)

**Purpose**: Realistic workload testing with multiple scenarios

**Workload Profiles**:
- `realistic_mixed`: 35% concept lookup, 25% search, 15% value sets, 15% semantic, 10% FHIR
- `heavy_reasoning`: 40% inference chains, 30% clustering, 20% hierarchy, 10% lookup
- `fhir_heavy`: 30% CodeSystem, 25% ValueSet, 20% ConceptMap, 15% search, 10% hybrid
- `cache_optimized`: 60% lookups, 20% value sets, 15% hierarchy, 5% FHIR

**Usage**:
```bash
python performance_tests.py --workload realistic_mixed --duration 300 --users 10
```

### 3. Load Testing (`load_testing.py`)

**Purpose**: Concurrent user simulation and capacity planning

**Test Scenarios**:
- `stress_test`: Gradually increase to 100 users over 5 minutes
- `soak_test`: Sustained 50 users for 1 hour
- `spike_test`: Sudden spike from 20 to 80 users
- `capacity_test`: Find breaking point up to 200 users

**Usage**:
```bash
python load_testing.py --scenario stress_test
```

### 4. Google FHIR Testing (`google_fhir_benchmark.py`)

**Purpose**: Google Healthcare API FHIR store performance validation

**Operations Tested**:
- CodeSystem $lookup operations
- CodeSystem $validate-code operations
- ValueSet $expand operations
- ValueSet $validate-code operations
- ConceptMap $translate operations
- Search operations

**Usage**:
```bash
python google_fhir_benchmark.py --iterations 100
```

## 📈 Performance Metrics

### Latency Metrics
- **Mean latency**: Average response time
- **P50 latency**: 50th percentile (median)
- **P95 latency**: 95th percentile (Phase 3.5 targets)
- **P99 latency**: 99th percentile
- **Max/Min latency**: Extreme values

### Throughput Metrics
- **Requests per second (RPS)**
- **Concurrent user capacity**
- **Breaking point analysis**

### System Metrics
- **CPU usage**
- **Memory usage**
- **Database connections**
- **Cache hit ratio**
- **Network I/O**

### Quality Metrics
- **Error rate percentage**
- **Success rate percentage**
- **Uptime percentage**
- **Data integrity score**

## 📋 Reporting

### Report Formats

1. **JSON Reports**: Machine-readable detailed results
2. **HTML Reports**: Human-readable with charts and tables
3. **JUnit XML**: CI/CD integration format
4. **Console Output**: Real-time progress and summaries

### Sample Report Structure
```json
{
  "test_run_id": "phase35_20231215_143022",
  "overall_status": "PASS",
  "success_criteria_results": [
    {
      "criterion": "PostgreSQL lookup queries <10ms (95th percentile)",
      "target": "<10ms",
      "actual": "8.2ms",
      "passed": true
    }
  ],
  "performance_summary": {
    "success_rate_pct": 98.5,
    "overall_p95_latency_ms": 45.2
  },
  "recommendations": [
    "All Phase 3.5 success criteria met! System ready for production."
  ]
}
```

## 🔍 Test Data

### Realistic Test Datasets

The system uses realistic medical terminology data:

**Code Systems**:
- SNOMED CT: Clinical terminology (10 sample codes)
- LOINC: Laboratory data (10 sample codes)
- RxNorm: Medications (7 sample codes)
- ICD-10: Diagnosis codes

**Value Sets**:
- Cardiovascular conditions
- Diabetes medications
- Respiratory conditions
- Laboratory tests
- Vital signs

**Concept Maps**:
- SNOMED CT ↔ LOINC mappings
- RxNorm ↔ SNOMED CT mappings
- ICD-10 ↔ SNOMED CT mappings

## 🐳 Docker Support

Run tests in isolated containers:

```bash
# Build test environment
docker build -t kb7-benchmarks .

# Run complete test suite
docker run --network kb7_test_network kb7-benchmarks

# Run with custom configuration
docker run -v $(pwd)/custom_config.yaml:/app/config.yaml kb7-benchmarks --config config.yaml
```

## 🔄 CI/CD Integration

### Jenkins Pipeline
```groovy
stage('KB7 Performance Tests') {
    steps {
        sh 'python benchmarks/run_all_tests.py --output-dir results/'
        publishTestResults testResultsPattern: 'results/junit_results.xml'
        publishHTML([
            allowMissing: false,
            alwaysLinkToLastBuild: true,
            keepAll: true,
            reportDir: 'results',
            reportFiles: 'test_suite_report.html',
            reportName: 'KB7 Performance Report'
        ])
    }
}
```

### GitHub Actions
```yaml
- name: Run KB7 Performance Tests
  run: |
    cd backend/services/medication-service/knowledge-bases/kb-7-terminology/benchmarks
    python run_all_tests.py --output-dir ${{ github.workspace }}/reports

- name: Upload Test Results
  uses: actions/upload-artifact@v3
  with:
    name: kb7-performance-reports
    path: reports/
```

## 🎛️ Advanced Usage

### Custom Workload Definition
```python
# Define custom workload in performance_tests.py
custom_workload = [
    WorkloadProfile('my_test', 'postgresql', 0.8, 'simple', True),
    WorkloadProfile('my_complex', 'neo4j', 0.2, 'complex', False),
]
```

### Performance Regression Detection
```bash
# Compare against baseline
python run_all_tests.py --baseline-file previous_results.json --fail-on-regression
```

### Monitoring Integration
```bash
# Export metrics to Prometheus
python run_all_tests.py --export-prometheus --prometheus-endpoint http://localhost:9090
```

## 🚨 Troubleshooting

### Common Issues

**Connection Failures**:
```bash
# Check service health
curl http://localhost:8090/health  # Query router
curl http://localhost:8014/health  # FHIR service
```

**Database Connection Issues**:
```bash
# Test PostgreSQL
psql -h localhost -p 5433 -U kb7_user -d terminology_db

# Test Redis
redis-cli -h localhost -p 6380 ping

# Test Neo4j
cypher-shell -a bolt://localhost:7687 -u neo4j -p kb7_neo4j
```

**Performance Issues**:
- Ensure adequate system resources (8GB+ RAM recommended)
- Check for competing processes during testing
- Verify network latency to services

### Debug Mode
```bash
# Enable verbose logging
python run_all_tests.py --log-level DEBUG

# Enable profiling
python performance_tests.py --enable-profiling
```

## 📚 Dependencies

### Core Requirements
- Python 3.8+
- PostgreSQL 13+
- Redis 6+
- Neo4j 5+

### Python Libraries
- `httpx`: HTTP client for async requests
- `psycopg2`: PostgreSQL adapter
- `redis`: Redis client
- `neo4j`: Neo4j driver
- `rich`: Terminal formatting
- `numpy`: Statistical calculations
- `pytest`: Testing framework

### Optional Dependencies
- `google-cloud-healthcare`: Google FHIR integration
- `prometheus-client`: Metrics export
- `docker`: Container integration

## 🤝 Contributing

### Adding New Tests
1. Create test function in appropriate module
2. Add configuration to `test_config.yaml`
3. Update `run_all_tests.py` orchestrator
4. Add documentation

### Custom Metrics
```python
# Add to configuration
custom_metrics:
  - name: my_metric
    query: "custom_calculation()"
```

## 📄 License

Part of KB7 Terminology service - CardioFit platform.

## 🔗 Related Documentation

- [KB7 Implementation Plan](../KB7_IMPLEMENTATION_PLAN.md)
- [Phase 3 Implementation](../PHASE3_IMPLEMENTATION_COMPLETE.md)
- [Performance Requirements](../docs/performance-requirements.md)

---

**Performance Engineer**: Claude Code
**Version**: 1.0.0
**Last Updated**: 2023-12-15