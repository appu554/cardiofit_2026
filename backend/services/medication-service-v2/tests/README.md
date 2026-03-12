# Medication Service V2 - Comprehensive Test Suite

This directory contains a comprehensive test suite for the Clinical Synthesis Hub Medication Service V2, designed to validate all implemented functionality with healthcare-grade quality assurance and regulatory compliance.

## 📋 Test Suite Overview

### Test Categories

| Category | Directory | Purpose | Coverage Target |
|----------|-----------|---------|-----------------|
| **Unit Tests** | `internal/application/services/tests/` | Component-level testing | >90% code coverage |
| **Integration Tests** | `tests/integration/` | End-to-end workflow testing | All critical workflows |
| **Performance Tests** | `tests/performance/` | Performance target validation | All target metrics |
| **Clinical Safety Tests** | `tests/clinical/` | Clinical logic & FHIR compliance | Safety & accuracy validation |
| **Security Tests** | `tests/security/` | Authentication & HIPAA compliance | Security & data protection |

### Key Performance Targets

- **End-to-End Response Time**: <250ms (95th percentile)
- **Recipe Resolution**: <10ms average
- **Sustained Throughput**: 1000+ RPS
- **Memory Usage**: <512MB per service instance
- **Cache Effectiveness**: 30%+ improvement

## 🚀 Quick Start

### Prerequisites

```bash
# Required services
docker-compose -f deployments/docker-compose.test.yml up -d

# Required tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/golang/mock/mockgen@latest
```

### Run All Tests

```bash
# Comprehensive test suite with reporting
make test-comprehensive

# Quick unit tests only
make test-quick

# Performance validation
make validate-performance

# Clinical safety validation
make test-clinical-safety

# Security compliance testing
make test-security-compliance
```

## 🧪 Detailed Test Categories

### Unit Tests (`internal/application/services/tests/`)

Tests individual components in isolation with mocked dependencies.

```bash
# Run unit tests with coverage
make test-unit-comprehensive

# Test recipe resolver performance (<10ms target)
make test-recipe-resolver

# Generate coverage report
make coverage-comprehensive
```

**Key Test Files:**
- `medication_service_test.go` - Core medication service functionality
- `recipe_resolver_service_test.go` - Recipe resolution with performance validation

**Coverage Requirements:**
- Overall: >90% code coverage
- Critical paths: 100% coverage
- Performance: All operations under target times

### Integration Tests (`tests/integration/`)

End-to-end workflow testing across all services with real database interactions.

```bash
# Run integration tests
make test-integration-comprehensive

# Test clinical workflows
make test-clinical-workflows
```

**Key Test Files:**
- `medication_workflow_integration_test.go` - Complete 4-phase workflow testing

**Test Scenarios:**
- Complete medication proposal workflow (<250ms)
- Pediatric patient handling
- Renal impaired patient adjustments
- Concurrent workflow execution
- Error recovery and rollback
- Cache effectiveness validation

### Performance Tests (`tests/performance/`)

Validates all performance targets with realistic load scenarios.

```bash
# Validate all performance targets
make validate-performance

# Load testing (1000+ RPS)
make test-load

# Memory usage validation (<512MB)
make test-memory

# Cache performance testing
make test-cache-performance
```

**Key Test Files:**
- `performance_test.go` - Comprehensive performance validation

**Performance Scenarios:**
- End-to-end response time validation
- Recipe resolver performance (<10ms)
- Clinical engine performance (<50ms)
- Sustained throughput testing (1000+ RPS)
- Memory usage under load
- Cache effectiveness measurement

### Clinical Safety Tests (`tests/clinical/`)

Validates clinical logic, drug interactions, dosing accuracy, and FHIR compliance.

```bash
# Run clinical safety tests
make test-clinical-safety

# Test drug interactions
make test-drug-interactions

# Test dosage accuracy
make test-dosage-accuracy

# Test FHIR compliance
make test-fhir-compliance

# Test patient safety rules
make test-patient-safety
```

**Key Test Files:**
- `clinical_safety_test.go` - Comprehensive clinical validation

**Clinical Scenarios:**
- Dosage calculation accuracy (BSA-based, weight-based, age-adjusted)
- Drug interaction detection (minor, moderate, major, contraindicated)
- Allergy contraindication checks
- Age-based safety validations (pediatric, geriatric)
- Organ function adjustments (renal, hepatic)
- FHIR R4 MedicationRequest compliance
- Clinical decision support rules
- Monitoring requirement generation

### Security Tests (`tests/security/`)

Validates authentication, authorization, and HIPAA compliance.

```bash
# Run security compliance tests
make test-security-compliance

# Test authentication
make test-auth

# Test HIPAA compliance
make test-hipaa-compliance

# Test input validation
make test-input-validation

# Test data protection
make test-data-protection
```

**Key Test Files:**
- `security_test.go` - Comprehensive security validation

**Security Scenarios:**
- JWT authentication validation
- Role-based authorization (admin, clinician, read-only)
- Scope-based permissions
- HIPAA audit trail compliance
- Input sanitization (SQL injection, XSS, path traversal)
- Data encryption in transit
- Rate limiting and throttling
- Security headers validation

## 🏥 Healthcare-Specific Testing

### Clinical Logic Validation

```bash
# Adult BSA-based dosing
TestDosageCalculationAccuracy/Adult_BSA-based_vincristine

# Pediatric weight-based dosing  
TestDosageCalculationAccuracy/Pediatric_weight-based_dosing

# Renal impaired dosing
TestDosageCalculationAccuracy/Renal_impaired_adult
```

### Drug Interaction Testing

```bash
# No interactions
TestDrugInteractionDetection/No_drug_interactions

# Moderate interaction with phenytoin
TestDrugInteractionDetection/Moderate_interaction_with_phenytoin

# Major interaction with azole antifungal
TestDrugInteractionDetection/Major_interaction_with_azole_antifungal
```

### FHIR R4 Compliance

```bash
# Resource structure validation
TestFHIRResourceCompliance

# Required fields validation
# Status/intent value validation  
# Extension compliance
# Clinical context preservation
```

## 🔒 Security & Compliance Testing

### Authentication Testing

```bash
# Valid tokens (admin, clinician, read-only)
TestJWTAuthenticationValidation/Valid_*_token

# Invalid/expired tokens
TestJWTAuthenticationValidation/Expired_token
TestJWTAuthenticationValidation/Invalid_token_format
```

### Authorization Testing

```bash
# Role-based permissions
TestRoleBasedAuthorization/Admin_can_*
TestRoleBasedAuthorization/Clinician_can_*
TestRoleBasedAuthorization/Read-only_cannot_*

# Scope-based permissions
TestScopeBasedAuthorization
```

### HIPAA Compliance

```bash
# Audit trail completeness
TestHIPAAAuditCompliance

# Required audit fields
# No sensitive data in logs
# Proper action classification
```

## ⚡ Performance Testing

### Response Time Validation

```bash
# <250ms end-to-end target
TestMedicationProposalEndToEndPerformance

# <10ms recipe resolution target
TestRecipeResolverPerformance

# <50ms clinical engine target
TestClinicalEnginePerformance
```

### Throughput Testing

```bash
# 1000+ RPS sustained target
TestThroughputUnderLoad

# Concurrent execution
TestConcurrentWorkflowExecution

# Load testing scenarios
TestWorkflowPerformanceUnderLoad
```

### Resource Usage

```bash
# <512MB memory target
TestMemoryUsageUnderLoad

# Cache effectiveness (30%+ improvement)
TestCachePerformanceImprovement
```

## 📊 Test Configuration

### Configuration File

`tests/test_config.yaml` contains comprehensive test configuration:

```yaml
performance:
  end_to_end_target_ms: 250
  recipe_resolution_target_ms: 10
  target_rps: 1000
  memory_limit_mb: 512

coverage:
  unit_tests_target: 90
  integration_tests_target: 85
  overall_target: 85

clinical_test_data:
  patients:
    adult_standard: { age: 45, weight_kg: 70.0, ... }
    pediatric: { age: 8, weight_kg: 25.0, ... }
    renal_impaired: { age: 65, egfr: 45.0, ... }

security:
  test_users:
    admin: { roles: ["admin", "clinician"], scopes: [...] }
    clinician: { roles: ["clinician"], scopes: [...] }
```

### Test Data & Fixtures

`tests/helpers/fixtures/fixtures.go` provides:
- Valid clinical contexts for different patient types
- Medication proposals with all required fields
- Recipe definitions with comprehensive rules
- FHIR-compliant test data
- Performance test scenarios

### Mock Services

`tests/helpers/mocks/mocks.go` provides mocks for:
- External service dependencies
- Database repositories
- Redis caching layer
- Rust clinical engine
- Apollo Federation client
- Context Gateway service

## 🔧 Test Infrastructure

### Database Setup

```bash
# Test database configuration
host: "localhost"
port: "5434"  # Isolated test port
name: "medication_service_test"

# Automatic migrations and cleanup
make test-setup    # Setup test environment
make test-cleanup  # Clean up test environment
make test-db-reset # Reset test database
```

### Service Dependencies

```bash
# Mock services (default for unit/integration tests)
USE_MOCK_SERVICES=true

# Real services (for system testing)
USE_REAL_RUST_ENGINE=true
USE_REAL_APOLLO=true
USE_REAL_CONTEXT_GATEWAY=true
```

## 📈 Continuous Integration

### CI Pipeline Integration

```bash
# Full CI test suite
make ci-test

# Quick CI tests (unit + basic integration)  
make ci-test-quick

# Pre-commit validation
make pre-commit
```

### Coverage Requirements

```bash
# Check coverage meets requirements
make coverage-check

# Generate comprehensive coverage report
make coverage-comprehensive
```

## 🎯 Test Execution Examples

### Development Workflow

```bash
# 1. Quick development feedback
make test-unit

# 2. Integration validation
make test-integration-comprehensive

# 3. Performance check
make validate-performance

# 4. Full validation before PR
make test-comprehensive
```

### Specialized Testing

```bash
# Clinical validation for medication changes
make test-clinical-safety

# Security review
make test-security-compliance

# Performance regression testing
make benchmark-all

# Memory profiling
make profile-memory
```

### Production Readiness

```bash
# Complete validation suite
make test-complete

# Performance benchmarking
make validate-performance

# Security compliance
make test-security-compliance

# Coverage verification
make coverage-check
```

## 📋 Test Reports

### Generated Reports

- `coverage-comprehensive.html` - Visual coverage report
- `test-results/test-report.json` - Structured test results
- `ci-results.json` - CI/CD integration results
- Performance metrics in test output

### Key Metrics Tracked

- Response time percentiles (50th, 95th, 99th)
- Throughput (requests per second)
- Error rates and failure modes
- Memory usage patterns
- Cache hit rates
- Clinical accuracy scores
- Security compliance scores

## 🆘 Troubleshooting

### Common Issues

1. **Test Database Connection**
   ```bash
   make test-setup  # Ensure test infrastructure is running
   ```

2. **Performance Test Failures**
   ```bash
   # Check system resources
   make profile-memory
   make profile-cpu
   ```

3. **Coverage Below Target**
   ```bash
   make coverage-comprehensive  # Identify uncovered code
   ```

4. **Integration Test Timeouts**
   ```bash
   # Increase timeout or check service dependencies
   export TEST_TIMEOUT_MULTIPLIER=2.0
   ```

### Debug Mode

```bash
# Verbose test output
go test -v ./tests/...

# Race condition detection
go test -race ./tests/...

# Memory debugging
go test -memprofile=mem.prof ./tests/...
```

## 🎉 Success Criteria

### Test Passing Requirements

✅ **Unit Tests**: >90% code coverage, all tests passing  
✅ **Integration Tests**: All critical workflows validated  
✅ **Performance Tests**: All targets met (<250ms, 1000+ RPS, <512MB)  
✅ **Clinical Tests**: 100% safety validations passing  
✅ **Security Tests**: Complete compliance validation  
✅ **FHIR Tests**: Full R4 compliance verified  

### Ready for Production

- All test suites passing
- Performance targets validated
- Security compliance verified
- Clinical safety validated
- HIPAA audit trails complete
- Coverage requirements met

---

**Note**: This test suite provides comprehensive validation for a healthcare-grade medication service with clinical safety, regulatory compliance, and production performance requirements.