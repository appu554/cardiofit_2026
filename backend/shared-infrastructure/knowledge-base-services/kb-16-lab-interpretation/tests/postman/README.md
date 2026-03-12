# KB-16 Lab Interpretation - Postman + Newman Automation Pack

Comprehensive clinical validation test suite for KB-16 Lab Interpretation Service, designed for SaMD (Software as a Medical Device) compliance and CI/CD integration.

## Overview

This automation pack includes:
- **Postman Collection**: 60+ clinical validation tests across 9 phases
- **Environment Files**: Local, Docker, and Staging configurations
- **Newman Scripts**: CLI automation for CI/CD pipelines
- **HTML Reports**: Rich validation reports for CMO review

## Test Phases

| Phase | Name | Tests | Focus |
|-------|------|-------|-------|
| 0 | Health Checks | 3 | Service availability and readiness |
| 1 | KB-8 Dependency Validation | 7 | eGFR (CKD-EPI 2021), Anion Gap, Corrected Calcium |
| 2 | Core Lab Interpretation | 10 | Critical/panic values for K, Na, Hgb, Glucose, etc. |
| 3 | Panel-Level Intelligence | 6 | BMP, CBC, LFT, Renal, Thyroid panels with pattern detection |
| 4 | Context-Aware Interpretation | 4 | Pregnancy, pediatric, CKD, dialysis adjustments |
| 5 | Delta Check & Trending | 3 | Significant change detection, trajectory analysis |
| 6 | Care Gap Intelligence | 2 | Overdue monitoring detection (HbA1c, INR) |
| 7 | Governance & Safety | 2 | KB-14 task creation, audit trails |
| 8 | Performance | 3 | Response time SLAs (<200ms single, <500ms panel) |
| 9 | Clinical Edge Cases | 3 | Hemolyzed specimens, lipemia, implausible values |

## Prerequisites

1. **Newman CLI**:
   ```bash
   npm install -g newman newman-reporter-htmlextra newman-reporter-junit
   ```

2. **Running Services**:
   - KB-16 Lab Interpretation (port 8098)
   - KB-8 Calculator Service (port 8097)

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Start services
cd kb-16-lab-interpretation
docker-compose up -d

# Run all tests
cd tests/postman
./run-newman.sh docker
```

### Using Local Services

```bash
# Run with local environment
./run-newman.sh local
```

### Run Specific Phase

```bash
# Run only Phase 1 (KB-8 validation)
./run-phase.sh 1 docker

# Run only Phase 3 (Panel tests)
./run-phase.sh 3 docker

# Run all phases sequentially
./run-phase.sh all docker
```

## CI/CD Integration

### GitHub Actions

```yaml
jobs:
  clinical-validation:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'

      - name: Install Newman
        run: npm install -g newman newman-reporter-htmlextra newman-reporter-junit

      - name: Start Services
        run: docker-compose up -d
        working-directory: kb-16-lab-interpretation

      - name: Run Clinical Validation
        run: ./ci-pipeline.sh
        working-directory: kb-16-lab-interpretation/tests/postman
        env:
          KB16_TEST_ENV: docker
          KB16_TEST_RETRIES: 3

      - name: Upload Test Results
        uses: actions/upload-artifact@v3
        with:
          name: clinical-validation-report
          path: kb-16-lab-interpretation/tests/postman/reports/
```

### GitLab CI

```yaml
clinical-validation:
  stage: test
  image: node:18
  services:
    - docker:dind
  before_script:
    - npm install -g newman newman-reporter-htmlextra newman-reporter-junit
    - docker-compose up -d
  script:
    - cd tests/postman && ./ci-pipeline.sh
  artifacts:
    reports:
      junit: tests/postman/reports/ci-junit-latest.xml
    paths:
      - tests/postman/reports/
```

### Jenkins

```groovy
pipeline {
    agent any

    environment {
        KB16_TEST_ENV = 'docker'
        KB16_TEST_RETRIES = '3'
    }

    stages {
        stage('Clinical Validation') {
            steps {
                sh 'cd tests/postman && ./ci-pipeline.sh'
            }
            post {
                always {
                    junit 'tests/postman/reports/ci-junit-latest.xml'
                    archiveArtifacts artifacts: 'tests/postman/reports/*.html'
                }
            }
        }
    }
}
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KB16_TEST_ENV` | `docker` | Environment to use (local/docker/staging) |
| `KB16_TEST_RETRIES` | `3` | Number of retry attempts on failure |
| `KB16_HEALTH_TIMEOUT` | `60` | Health check timeout in seconds |
| `SKIP_HEALTH_CHECK` | `false` | Skip health checks before tests |
| `BAIL_ON_FAILURE` | `true` | Stop on first failure |

## Report Outputs

After running tests, reports are generated in `./reports/`:

| File | Format | Description |
|------|--------|-------------|
| `kb16-report-*.html` | HTML | Rich validation report with visualizations |
| `kb16-results-*.json` | JSON | Detailed test results for programmatic analysis |
| `kb16-junit-*.xml` | JUnit XML | CI/CD integration format |

### Sample HTML Report Sections

- **Executive Summary**: Pass/fail rates, critical failures
- **Phase Breakdown**: Per-phase results with timing
- **Request Details**: Full request/response for debugging
- **Assertion Results**: Individual assertion outcomes

## Critical Value Validation

The test suite validates panic value detection per CAP guidelines:

| Test | LOINC | Critical Low | Critical High |
|------|-------|--------------|---------------|
| Potassium | 2823-3 | ≤2.5 mEq/L | ≥6.5 mEq/L |
| Sodium | 2951-2 | ≤120 mEq/L | ≥160 mEq/L |
| Glucose | 2345-7 | ≤40 mg/dL | ≥500 mg/dL |
| Hemoglobin | 718-7 | ≤5.0 g/dL | - |
| INR | 34714-6 | - | ≥8.0 |
| Lactate | 2524-7 | - | ≥7.0 mmol/L |
| Platelets | 777-3 | ≤20,000/µL | - |

## SaMD Compliance

This test suite supports:
- **IEC 62304**: Software lifecycle requirements
- **FDA SaMD Guidance**: Clinical validation evidence
- **21 CFR Part 11**: Audit trail verification

For CMO sign-off, use the generated HTML report along with the Clinical Acceptance Sheet.

## Troubleshooting

### Services Not Responding

```bash
# Check service status
curl http://localhost:8098/health
curl http://localhost:8097/health

# View logs
docker-compose logs kb-16
docker-compose logs kb-8
```

### Newman Not Installed

```bash
npm install -g newman newman-reporter-htmlextra newman-reporter-junit
```

### Permission Denied

```bash
chmod +x run-newman.sh run-phase.sh ci-pipeline.sh
```

## Directory Structure

```
tests/postman/
├── KB16_Lab_Interpretation.postman_collection.json
├── newman-config.json
├── run-newman.sh           # Full test suite runner
├── run-phase.sh            # Single phase runner
├── ci-pipeline.sh          # CI/CD optimized runner
├── README.md
├── environments/
│   ├── local.postman_environment.json
│   ├── docker.postman_environment.json
│   └── staging.postman_environment.json
└── reports/                # Generated reports (gitignored)
    ├── kb16-report-latest.html
    ├── kb16-results-latest.json
    └── kb16-junit-latest.xml
```

## Maintenance

### Adding New Tests

1. Open Postman and import the collection
2. Add tests to appropriate phase folder
3. Export and replace collection JSON
4. Update phase counts in this README

### Updating Reference Ranges

Reference ranges are embedded in test assertions. Update `pm.expect()` values as clinical guidelines change.

---

**Version**: 1.0.0
**Last Updated**: 2024-01-15
**Maintainer**: Clinical Engineering Team
