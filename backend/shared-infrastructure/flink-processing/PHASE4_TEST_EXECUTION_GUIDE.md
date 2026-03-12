# Phase 4 Test Execution Quick Reference Guide

## Test Files Overview

| Test File | Purpose | Test Count |
|-----------|---------|------------|
| `TestRecommenderTest.java` | Test Recommender intelligence engine unit tests | 19 |
| `ActionBuilderPhase4IntegrationTest.java` | ActionBuilder Phase 4 integration tests | 19 |
| `DiagnosticTestLoaderTest.java` | YAML loader and caching tests | 26 |
| `Phase4EndToEndTest.java` | Complete pipeline E2E tests | 8 |

**Total: 72 tests**

---

## Quick Start

### Run All Phase 4 Tests
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

# Run all Phase 4 tests
mvn test -Dtest="com.cardiofit.flink.phase4.*"
```

### Run Individual Test Classes
```bash
# Test Recommender tests
mvn test -Dtest="TestRecommenderTest"

# ActionBuilder Phase 4 integration tests
mvn test -Dtest="ActionBuilderPhase4IntegrationTest"

# DiagnosticTestLoader tests
mvn test -Dtest="DiagnosticTestLoaderTest"

# End-to-end tests
mvn test -Dtest="Phase4EndToEndTest"
```

### Run Specific Test Methods
```bash
# Run single test method
mvn test -Dtest="TestRecommenderTest#testSepsisDiagnosticBundle_GeneratesEssentialTests"

# Run multiple test methods
mvn test -Dtest="TestRecommenderTest#testSepsisDiagnosticBundle_*"
```

---

## Test Execution with Coverage

### Generate JaCoCo Coverage Report
```bash
# Run tests with coverage
mvn clean test jacoco:report -Dtest="com.cardiofit.flink.phase4.*"

# View coverage report
open target/site/jacoco/index.html
```

### Coverage Targets
```yaml
Package Coverage Targets:
  com.cardiofit.flink.intelligence.TestRecommender: > 85%
  com.cardiofit.flink.processors.ActionBuilder: > 80%
  com.cardiofit.flink.loader.DiagnosticTestLoader: > 90%
```

---

## Test Execution Modes

### 1. Quick Validation (E2E only)
```bash
# Run only end-to-end tests for quick validation
mvn test -Dtest="Phase4EndToEndTest"

# Expected time: ~5 seconds
```

### 2. Component Testing
```bash
# Test individual components
mvn test -Dtest="TestRecommenderTest,DiagnosticTestLoaderTest"

# Expected time: ~10 seconds
```

### 3. Full Test Suite
```bash
# Run all Phase 4 tests
mvn test -Dtest="com.cardiofit.flink.phase4.*"

# Expected time: ~15 seconds
```

### 4. Integration with All Tests
```bash
# Run all Module 3 tests (Phase 1-4)
mvn test

# Expected time: ~30 seconds
```

---

## Debugging Failed Tests

### Enable Detailed Logging
```bash
# Run with debug logging
mvn test -Dtest="TestRecommenderTest" -X

# Run with custom log level
mvn test -Dtest="TestRecommenderTest" -Dorg.slf4j.simpleLogger.defaultLogLevel=DEBUG
```

### Run Single Failing Test
```bash
# Isolate failing test
mvn test -Dtest="TestRecommenderTest#testSepsisDiagnosticBundle_GeneratesEssentialTests"

# Enable assertions
mvn test -Dtest="TestRecommenderTest#testSepsisDiagnosticBundle_GeneratesEssentialTests" -ea
```

### View Test Output
```bash
# Show test output in console
mvn test -Dtest="TestRecommenderTest" -Dsurefire.printSummary=true

# View detailed report
cat target/surefire-reports/com.cardiofit.flink.phase4.TestRecommenderTest.txt
```

---

## CI/CD Integration

### GitHub Actions Workflow
```yaml
name: Phase 4 Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v3

    - name: Set up JDK 17
      uses: actions/setup-java@v3
      with:
        java-version: '17'
        distribution: 'temurin'

    - name: Cache Maven packages
      uses: actions/cache@v3
      with:
        path: ~/.m2
        key: ${{ runner.os }}-m2-${{ hashFiles('**/pom.xml') }}
        restore-keys: ${{ runner.os }}-m2

    - name: Run Phase 4 Tests
      run: mvn test -Dtest="com.cardiofit.flink.phase4.*"

    - name: Generate Coverage Report
      run: mvn jacoco:report

    - name: Upload Coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        files: ./target/site/jacoco/jacoco.xml
        flags: phase4
        name: Phase4-Coverage
```

### Jenkins Pipeline
```groovy
pipeline {
    agent any

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Test Phase 4') {
            steps {
                dir('backend/shared-infrastructure/flink-processing') {
                    sh 'mvn clean test -Dtest="com.cardiofit.flink.phase4.*"'
                }
            }
        }

        stage('Coverage Report') {
            steps {
                dir('backend/shared-infrastructure/flink-processing') {
                    sh 'mvn jacoco:report'
                    publishHTML([
                        reportDir: 'target/site/jacoco',
                        reportFiles: 'index.html',
                        reportName: 'Phase 4 Coverage Report'
                    ])
                }
            }
        }
    }

    post {
        always {
            junit 'backend/shared-infrastructure/flink-processing/target/surefire-reports/*.xml'
        }
    }
}
```

---

## Test Data Dependencies

### Required Classes
```
✅ EnrichedPatientContext
✅ PatientContextState
✅ PatientDemographics
✅ Protocol
✅ TestRecommendation
✅ ClinicalAction
✅ DiagnosticDetails
✅ LabTest
✅ ImagingStudy
✅ TestResult
```

### YAML Knowledge Base Files
```
Expected location: src/main/resources/knowledge-base/diagnostic-tests/

Lab Tests (chemistry/):
  - lactate.yaml
  - glucose.yaml
  - creatinine.yaml
  - sodium.yaml
  - potassium.yaml
  - bun.yaml

Lab Tests (hematology/):
  - hemoglobin.yaml
  - platelets.yaml
  - wbc.yaml
  - pt-inr.yaml

Imaging Studies (radiology/):
  - chest-xray.yaml
  - ct-chest.yaml
  - ct-head.yaml

Imaging Studies (cardiac/):
  - echocardiogram.yaml

Imaging Studies (ultrasound/):
  - abdominal-ultrasound.yaml
```

---

## Expected Test Results

### Successful Test Run Output
```
[INFO] -------------------------------------------------------
[INFO]  T E S T S
[INFO] -------------------------------------------------------
[INFO] Running com.cardiofit.flink.phase4.TestRecommenderTest
[INFO] Tests run: 19, Failures: 0, Errors: 0, Skipped: 0
[INFO]
[INFO] Running com.cardiofit.flink.phase4.ActionBuilderPhase4IntegrationTest
[INFO] Tests run: 19, Failures: 0, Errors: 0, Skipped: 0
[INFO]
[INFO] Running com.cardiofit.flink.phase4.DiagnosticTestLoaderTest
[INFO] Tests run: 26, Failures: 0, Errors: 0, Skipped: 0
[INFO]
[INFO] Running com.cardiofit.flink.phase4.Phase4EndToEndTest
[INFO] Tests run: 8, Failures: 0, Errors: 0, Skipped: 0
[INFO]
[INFO] Results:
[INFO]
[INFO] Tests run: 72, Failures: 0, Errors: 0, Skipped: 0
[INFO]
[INFO] ------------------------------------------------------------------------
[INFO] BUILD SUCCESS
[INFO] ------------------------------------------------------------------------
[INFO] Total time: 12.345 s
[INFO] Finished at: 2025-10-23T20:59:00Z
[INFO] ------------------------------------------------------------------------
```

### Test Summary
```
✅ TestRecommenderTest ..................... 19/19 passed
✅ ActionBuilderPhase4IntegrationTest ...... 19/19 passed
✅ DiagnosticTestLoaderTest ................ 26/26 passed
✅ Phase4EndToEndTest ...................... 8/8 passed

Total: 72/72 tests passed (100%)
```

---

## Troubleshooting

### Common Issues

#### 1. Test Compilation Errors
```bash
# Issue: Missing dependencies
# Solution: Update pom.xml and rebuild
mvn clean install -DskipTests
mvn test -Dtest="com.cardiofit.flink.phase4.*"
```

#### 2. DiagnosticTestLoader Initialization Failure
```bash
# Issue: YAML files not found
# Check: Verify knowledge base directory exists
ls -la src/main/resources/knowledge-base/diagnostic-tests/

# Solution: Create YAML files or update loader paths
```

#### 3. AssertJ Dependency Not Found
```bash
# Issue: AssertJ not in classpath
# Solution: Add to pom.xml
<dependency>
    <groupId>org.assertj</groupId>
    <artifactId>assertj-core</artifactId>
    <version>3.24.2</version>
    <scope>test</scope>
</dependency>
```

#### 4. Test Timeout
```bash
# Issue: Tests taking too long
# Solution: Increase timeout or check for performance issues
mvn test -Dtest="Phase4EndToEndTest" -Dsurefire.timeout=300
```

---

## Performance Benchmarks

### Expected Execution Times
```
TestRecommenderTest:                    ~3 seconds
ActionBuilderPhase4IntegrationTest:     ~4 seconds
DiagnosticTestLoaderTest:               ~5 seconds
Phase4EndToEndTest:                     ~3 seconds

Total Phase 4 Suite:                    ~15 seconds
```

### Performance Optimization
```bash
# Run tests in parallel
mvn test -Dtest="com.cardiofit.flink.phase4.*" -DforkCount=2C -DreuseForks=true

# Skip slow tests
mvn test -Dtest="com.cardiofit.flink.phase4.*" -DexcludedGroups=slow
```

---

## Test Maintenance

### Adding New Tests
```bash
# 1. Create new test method
# 2. Run specific test to verify
mvn test -Dtest="TestRecommenderTest#testNewFeature"

# 3. Run full suite to ensure no regressions
mvn test -Dtest="com.cardiofit.flink.phase4.*"
```

### Updating Test Data
```bash
# 1. Update helper methods in test classes
# 2. Verify all tests still pass
mvn test -Dtest="com.cardiofit.flink.phase4.*"

# 3. Update documentation if needed
```

### Refactoring Tests
```bash
# 1. Make changes to test code
# 2. Run affected tests
mvn test -Dtest="TestRecommenderTest"

# 3. Run full suite
mvn test -Dtest="com.cardiofit.flink.phase4.*"

# 4. Check coverage hasn't decreased
mvn jacoco:report
```

---

## Contact & Support

### Test Ownership
- **Team**: Quality Engineering - Module 3 Phase 4
- **Created**: 2025-10-23
- **Location**: `/src/test/java/com/cardiofit/flink/phase4/`

### Documentation
- **Coverage Report**: `PHASE4_TEST_COVERAGE_REPORT.md`
- **Execution Guide**: `PHASE4_TEST_EXECUTION_GUIDE.md`

---

**Last Updated**: 2025-10-23
**Version**: 1.0
**Status**: Production Ready ✅
