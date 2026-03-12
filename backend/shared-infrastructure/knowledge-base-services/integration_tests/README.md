# Cross-Service Integration Tests

Comprehensive integration test suite for all Knowledge Base (KB) services, validating end-to-end workflows and service interactions.

## Overview

This test suite validates interactions between all 7 KB services:

1. **KB-1 Drug Rules** (port 8081) - PostgreSQL
2. **KB-2 Clinical Context** (port 8082) - MongoDB  
3. **KB-3 Guideline Evidence** (port 8084) - Neo4j
4. **KB-4 Patient Safety** (port 8085) - PostgreSQL
5. **KB-5 Drug Interactions** (port 8086) - PostgreSQL
6. **KB-6 Formulary** (port 8087) - Elasticsearch
7. **KB-7 Terminology** (port 8088) - PostgreSQL

## Test Scenarios

### 1. Medication Decision Support Workflow
Complete end-to-end medication decision support involving all services:
- Terminology validation
- Clinical phenotype matching
- Guideline recommendations
- Drug interaction checks
- Safety contraindication screening
- Formulary alternatives
- Dosing calculations

### 2. Terminology Consistency Check
Validates terminology consistency across drug-related services.

### 3. Clinical Workflow Integration
Tests integration between clinical context, guidelines, and safety services for population-level recommendations.

### 4. Performance Stress Test
Validates system performance under concurrent load with batch operations.

## Prerequisites

1. **All KB services running** on their respective ports
2. **Required databases** (PostgreSQL, MongoDB, Neo4j, Elasticsearch, Redis)
3. **Python 3.9+** with required packages

## Installation

```bash
# Install dependencies
pip install -r requirements.txt

# Make scripts executable (Linux/Mac)
chmod +x run_tests.py cross_service_tests.py
```

## Usage

### Quick Health Check
```bash
python run_tests.py --quick
```

### List Available Scenarios
```bash
python run_tests.py --list-scenarios
```

### Run Specific Scenario
```bash
python run_tests.py --scenario medication_decision_support_workflow
```

### Run All Integration Tests
```bash
python run_tests.py
```

### Run with Custom Report File
```bash
python run_tests.py --report my_test_report.md
```

### Verbose Output
```bash
python run_tests.py --verbose
```

## Makefile Commands

```bash
# Install dependencies
make install

# Quick health check of all services
make health

# Run all integration tests
make test

# Run specific scenario
make test-scenario SCENARIO=terminology_consistency_check

# Clean up test artifacts
make clean
```

## Test Configuration

Tests are configured via `test_config.yaml`:

- **Service endpoints** and health check paths
- **Test data sets** for consistent testing
- **Performance thresholds** for response times
- **Environment-specific settings**

## Output

Tests generate:

1. **Console output** with real-time progress
2. **Detailed logs** saved to `integration_tests_YYYYMMDD_HHMMSS.log`
3. **Markdown report** with comprehensive results
4. **JSON results** for programmatic access

## Sample Report Structure

```markdown
# Cross-Service Integration Test Report

## Summary
- **Total Scenarios**: 4
- **Successful**: 3
- **Failed**: 1
- **Success Rate**: 75.0%
- **Total Execution Time**: 45.32s

## Detailed Results

### medication_decision_support_workflow ✅ PASSED
**Execution Time**: 15.42s
**Expected Outcome**: Complete medication recommendation with dosing, safety checks, and alternatives
**Steps Completed**: 7

### Service Health Status
- ✅ **KB-1 Drug Rules Service**: healthy
- ✅ **KB-2 Clinical Context Service**: healthy
- ❌ **KB-3 Guideline Evidence Service**: unhealthy
```

## Troubleshooting

### Common Issues

1. **Service Unavailable**
   ```
   Error: Unhealthy services: ['KB-X Service Name']
   ```
   - Ensure all services are running on correct ports
   - Check service logs for startup errors
   - Verify database connections

2. **Timeout Errors**
   ```
   Error: Scenario timeout after 30 seconds
   ```
   - Increase timeout in scenario definition
   - Check for performance bottlenecks
   - Verify database query optimization

3. **Authentication Errors**
   ```
   Error: 401 Unauthorized
   ```
   - Some services may require authentication headers
   - Update test configuration with auth tokens

### Performance Considerations

- Tests are designed for **concurrent execution**
- **Batch operations** minimize individual request overhead
- **Circuit breakers** prevent cascade failures during testing
- **Connection pooling** improves test execution speed

## CI/CD Integration

The test suite is designed for integration with CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Run Integration Tests
  run: |
    python run_tests.py --report integration_report_${{ github.run_id }}.md
    
- name: Upload Test Report
  uses: actions/upload-artifact@v3
  with:
    name: integration-test-report
    path: integration_report_*.md
```

## Extending Tests

### Adding New Scenarios

1. **Define scenario** in `CrossServiceTestRunner._define_test_scenarios()`
2. **Add test data** to `test_config.yaml` if needed
3. **Update documentation**

### Adding New Services

1. **Add service endpoint** to `services` dictionary
2. **Update health checks**
3. **Create relevant test scenarios**

## Support

For issues or questions:
- Check service logs for detailed error information
- Verify all prerequisites are met
- Review test configuration settings
- Ensure all services are healthy before running tests