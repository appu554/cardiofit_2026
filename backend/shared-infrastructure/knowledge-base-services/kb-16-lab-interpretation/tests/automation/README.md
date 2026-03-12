# KB-16 Lab Interpretation - Automation Test Suite

Clinical Validation automation artifacts for KB-16 Lab Interpretation & Trending Service.

## Overview

This directory contains automated testing tools for the KB-16 Clinical Validation Test Matrix:

| Artifact | Purpose |
|----------|---------|
| `clinical_validation_test_suite.json` | Complete test matrix (148 tests across 9 phases) |
| `KB16_Lab_Interpretation.postman_collection.json` | Postman collection for API testing |
| `kb16_local.postman_environment.json` | Local environment configuration |
| `run_newman_tests.sh` | CI/CD automation script |
| `package.json` | npm dependencies for Newman |

## Quick Start

### Prerequisites

```bash
# Install Newman and reporters
npm install -g newman newman-reporter-htmlextra newman-reporter-junitfull

# Or use local installation
cd tests/automation
npm install
```

### Running Tests

```bash
# Health check only
./run_newman_tests.sh --health-only

# Run all tests
./run_newman_tests.sh

# Run specific phase (1-9)
./run_newman_tests.sh --phase 3

# Generate HTML report
./run_newman_tests.sh --report html

# Generate JUnit report for CI/CD
./run_newman_tests.sh --report junit
```

### Using npm Scripts

```bash
npm test                  # Run all tests
npm run test:health       # Health checks only
npm run test:phase1       # KB-8 Integration tests
npm run test:phase2       # Core Interpretation tests
npm run test:phase3       # Panel Intelligence tests
npm run test:phase4       # Context-Aware tests
npm run test:phase5       # Severity Tiering tests
npm run test:phase6       # Care Gap Intelligence tests
npm run test:phase7       # Governance tests
npm run test:phase8       # Performance tests
npm run test:phase9       # Edge Case tests
npm run test:html         # Generate HTML report
npm run test:junit        # Generate JUnit report
npm run test:ci           # CI pipeline mode
```

## Test Phases

### Phase 1: KB-8 Dependency Validation (10 tests)
Tests integration with KB-8 clinical calculators (eGFR, anion gap) and graceful degradation.

### Phase 2: Core Lab Interpretation (20 tests)
Single lab value interpretation with flag classification (NORMAL, LOW, HIGH, CRITICAL).

### Phase 3: Panel-Level Intelligence (30 tests)
Multi-test panel interpretation with pattern detection (BMP, CBC, LFT, Lipid, Cardiac, Renal).

### Phase 4: Context-Aware Interpretation (20 tests)
Patient context modifies interpretation (pregnancy, pediatric, CKD, dialysis, oncology).

### Phase 5: Severity & Risk Tiering (16 tests)
Color-coded severity classification and risk scoring.

### Phase 6: Care Gap Intelligence (12 tests)
Integration with KB-9 for care gap detection (HbA1c, lipids, INR monitoring).

### Phase 7: Governance & Safety (15 tests)
Critical value governance, audit trails, SLA tracking, 4-eyes principle.

### Phase 8: Performance & Chaos (10 tests)
Performance benchmarks and chaos engineering (latency, concurrency, failure recovery).

### Phase 9: Clinical Edge Cases (15 tests)
Edge case handling (hemolyzed samples, delta checks, neonatal, unit conversion).

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KB16_BASE_URL` | `http://localhost:8095` | KB-16 service URL |
| `KB8_URL` | `http://localhost:8088` | KB-8 Clinical Calculators URL |
| `KB14_URL` | `http://localhost:8093` | KB-14 Care Navigator URL |
| `NEWMAN_TIMEOUT` | `30000` | Request timeout in ms |
| `CI_MODE` | `false` | Enable CI pipeline mode |

## CI/CD Integration

### GitHub Actions

```yaml
- name: Run KB-16 Clinical Validation Tests
  run: |
    cd backend/shared-infrastructure/knowledge-base-services/kb-16-lab-interpretation/tests/automation
    npm install
    CI_MODE=true ./run_newman_tests.sh --report junit

- name: Upload Test Results
  uses: actions/upload-artifact@v3
  with:
    name: kb16-test-results
    path: tests/automation/reports/
```

### GitLab CI

```yaml
kb16-tests:
  stage: test
  script:
    - cd backend/shared-infrastructure/knowledge-base-services/kb-16-lab-interpretation/tests/automation
    - npm install
    - CI_MODE=true ./run_newman_tests.sh --report junit
  artifacts:
    reports:
      junit: tests/automation/reports/*_junit.xml
```

### Jenkins Pipeline

```groovy
stage('KB-16 Clinical Validation') {
    steps {
        dir('backend/shared-infrastructure/knowledge-base-services/kb-16-lab-interpretation/tests/automation') {
            sh 'npm install'
            sh 'CI_MODE=true ./run_newman_tests.sh --report junit'
        }
    }
    post {
        always {
            junit 'tests/automation/reports/*_junit.xml'
        }
    }
}
```

## LOINC Code Reference

| Code | Lab Test | Critical Low | Critical High |
|------|----------|--------------|---------------|
| 2823-3 | Potassium | 2.5 mEq/L | 6.5 mEq/L |
| 2951-2 | Sodium | 120 mEq/L | 160 mEq/L |
| 2345-7 | Glucose | 40 mg/dL | 500 mg/dL |
| 718-7 | Hemoglobin | 5.0 g/dL | - |
| 777-3 | Platelets | 20,000 /uL | - |
| 34714-6 | INR | - | 8.0 |
| 2524-7 | Lactate | - | 7.0 mmol/L |

## Reports

Generated reports are saved to `tests/automation/reports/`:

- `*_junit.xml` - JUnit XML for CI/CD integration
- `*_report.html` - Interactive HTML report
- `*_results.json` - Raw JSON results

## Troubleshooting

### Newman not found
```bash
npm install -g newman
```

### Service not responding
```bash
# Check service health
curl http://localhost:8095/health

# Wait for service to be ready
./run_newman_tests.sh --health-only --verbose
```

### Timeout errors
```bash
# Increase timeout
NEWMAN_TIMEOUT=60000 ./run_newman_tests.sh
```

## Contributing

1. Add new tests to the Postman collection
2. Update `clinical_validation_test_suite.json` with test documentation
3. Run full test suite to verify
4. Update this README if adding new phases or features

---

*Generated by KB-16 Clinical Validation Framework | CardioFit Platform*
