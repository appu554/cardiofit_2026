# Phase 7 Testing Guide

**Status**: Ready for Testing
**Test Suite**: Integration + Clinical Scenarios
**Total Test Cases**: 11 comprehensive tests

---

## Test Suite Overview

### Created Test Files

1. **[Phase7IntegrationTest.java](../backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/phase7/Phase7IntegrationTest.java)**
   - 7 integration tests
   - Focus: Component integration and API validation
   - Duration: ~30 seconds

2. **[ClinicalScenarioTest.java](../backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/phase7/ClinicalScenarioTest.java)**
   - 4 clinical scenario tests
   - Focus: Real-world clinical workflows
   - Duration: ~20 seconds

---

## Test Coverage

### Integration Tests (Phase7IntegrationTest.java)

| Test # | Name | Purpose | Validates |
|--------|------|---------|-----------|
| 1 | Protocol Library Loading | Load all YAML protocols | 10+ protocols load successfully |
| 2 | Sepsis Protocol Matching | Match alerts to protocols | Correct protocol selection |
| 3 | Allergy Detection | Safety validation | Penicillin allergy detected |
| 4 | Safe Medication Validation | Safety validation | No contraindications = safe |
| 5 | Dose Calculation Integration | Phase 6 integration | Dose calculated correctly |
| 6 | Medication Action Builder | Action generation | Complete action structure |
| 7 | Complete Clinical Workflow | End-to-end | Full sepsis management workflow |

### Clinical Scenario Tests (ClinicalScenarioTest.java)

| Test # | Scenario | Clinical Relevance | Key Validation |
|--------|----------|-------------------|----------------|
| 1 | Septic Shock | Most common ICU emergency | Broad-spectrum antibiotics with timing |
| 2 | STEMI | Time-critical cardiac care | Door-to-balloon < 90 min protocol |
| 3 | Penicillin Allergy | Common drug allergy | Cross-reactivity detection |
| 4 | Renal Dysfunction | Dose adjustment required | Renal dose calculation |

---

## Running the Tests

### Quick Start - Run All Tests

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

# Run all Phase 7 tests
mvn test -Dtest="com.cardiofit.flink.phase7.*"
```

### Run Specific Test Classes

```bash
# Integration tests only
mvn test -Dtest=Phase7IntegrationTest

# Clinical scenarios only
mvn test -Dtest=ClinicalScenarioTest

# Specific test method
mvn test -Dtest=Phase7IntegrationTest#testProtocolLibraryLoading
```

### Run with Detailed Output

```bash
# Full test output with logs
mvn test -Dtest="com.cardiofit.flink.phase7.*" -X

# Quieter output
mvn test -Dtest="com.cardiofit.flink.phase7.*" -q
```

---

## Expected Test Results

### Success Criteria

✅ **All 11 tests should PASS**

**Integration Tests** (7/7):
- [x] Protocol library loads 10+ protocols
- [x] Sepsis protocol matches correctly
- [x] Allergies are detected
- [x] Safe medications validate successfully
- [x] Doses calculate correctly
- [x] Actions build with complete details
- [x] End-to-end workflow completes

**Clinical Scenarios** (4/4):
- [x] Septic shock generates appropriate antibiotics
- [x] STEMI protocol has time-critical actions
- [x] Penicillin allergy prevents beta-lactam use
- [x] Renal dysfunction triggers dose adjustment

### Sample Success Output

```
[INFO] -------------------------------------------------------
[INFO]  T E S T S
[INFO] -------------------------------------------------------
[INFO] Running com.cardiofit.flink.phase7.Phase7IntegrationTest
[INFO] === TEST 1: Protocol Library Loading ===
[INFO] ✅ Loaded 10 protocols
[INFO] ✅ TEST 1 PASSED - All protocols loaded and validated
[INFO]
[INFO] === TEST 2: Sepsis Protocol Matching ===
[INFO] ✅ Matched protocol: Sepsis Management Bundle (ID: SEPSIS-BUNDLE-001)
[INFO] ✅ TEST 2 PASSED - Sepsis protocol matched successfully
...
[INFO] Tests run: 7, Failures: 0, Errors: 0, Skipped: 0

[INFO] Running com.cardiofit.flink.phase7.ClinicalScenarioTest
[INFO] SCENARIO 1: Septic Shock - Broad-Spectrum Antibiotics
[INFO] ✅ SCENARIO 1 COMPLETE - 3 medication actions recommended
...
[INFO] Tests run: 4, Failures: 0, Errors: 0, Skipped: 0

[INFO] Results:
[INFO] Tests run: 11, Failures: 0, Errors: 0, Skipped: 0
[INFO] BUILD SUCCESS
```

---

## Troubleshooting

### Common Issues

#### Issue 1: Protocols Not Found
**Error**: `FileNotFoundException: protocols/SEPSIS-BUNDLE-001.yaml`

**Solution**:
```bash
# Verify protocol files exist
ls backend/shared-infrastructure/flink-processing/src/main/resources/protocols/

# Should see: SEPSIS-BUNDLE-001.yaml, STEMI-001.yaml, etc.
```

#### Issue 2: Medication Database Empty
**Error**: `Medication not found: MED-PIPT-001`

**Solution**:
- Ensure Phase 6 medication database is populated
- Check MedicationRepository initialization

#### Issue 3: Jackson YAML Parsing Errors
**Error**: `UnrecognizedPropertyException`

**Solution**:
```bash
# Verify Jackson dependencies
mvn dependency:tree | grep jackson

# Should include:
# - jackson-databind:2.17.0
# - jackson-dataformat-yaml:2.17.0
# - jackson-datatype-jsr310:2.17.0
```

#### Issue 4: Test Compilation Errors
**Error**: Import errors for Phase 7 classes

**Solution**:
```bash
# Rebuild project
mvn clean compile

# Verify all Phase 7 classes compiled
ls target/classes/com/cardiofit/flink/clinical/
ls target/classes/com/cardiofit/flink/protocols/
```

---

## Test Data Requirements

### Required Protocol Files (10)

Located in: `src/main/resources/protocols/`

1. SEPSIS-BUNDLE-001.yaml - Sepsis management
2. STEMI-001.yaml - ST-elevation MI
3. HF-ACUTE-001.yaml - Acute heart failure
4. DKA-001.yaml - Diabetic ketoacidosis
5. ARDS-001.yaml - Acute respiratory distress
6. STROKE-001.yaml - Acute ischemic stroke
7. ANAPHYLAXIS-001.yaml - Anaphylactic shock
8. HYPERKALEMIA-001.yaml - Severe hyperkalemia
9. ACS-NSTEMI-001.yaml - Non-STEMI ACS
10. HYPERTENSIVE-CRISIS-001.yaml - Hypertensive emergency

### Required Medication Database

**Phase 6 Medications** (from MedicationRepository):
- MED-PIPT-001: Piperacillin-Tazobactam
- Additional antibiotics for sepsis protocols
- Cardiac medications for STEMI protocols

**Database Location**: Phase 6 medication database (JSON files or in-memory)

---

## Performance Benchmarks

### Expected Test Execution Times

| Test Suite | Expected Duration | Actual Duration |
|------------|------------------|-----------------|
| Phase7IntegrationTest | < 30 seconds | TBD |
| ClinicalScenarioTest | < 20 seconds | TBD |
| **Total** | **< 50 seconds** | **TBD** |

### Performance Targets

- Protocol loading: < 2 seconds
- Safety validation: < 100ms per medication
- Dose calculation: < 50ms per calculation
- Complete workflow: < 500ms per patient

---

## Next Steps After Testing

### If All Tests Pass ✅

1. **Create Flink Pipeline E2E Test**
   - Test complete Kafka → Flink → Kafka flow
   - Validate serialization/deserialization
   - Test state management with RocksDB

2. **Performance Testing**
   - Throughput test: >100 events/second
   - Latency test: <100ms processing time
   - Load test: 1000+ patients

3. **Completion Report**
   - Document all Phase 7 deliverables
   - Create deployment guide
   - Provide operations manual

### If Tests Fail ❌

1. **Analyze Failure Logs**
   - Review stack traces
   - Check component initialization
   - Validate test data

2. **Fix Issues**
   - Update component implementations
   - Adjust test expectations if needed
   - Re-run tests

3. **Document Findings**
   - Update PHASE7_ISSUES.md
   - Track bug fixes
   - Validate fixes with tests

---

## Test Maintenance

### Adding New Tests

```java
@Test
@Order(8)
@DisplayName("Test 8: New Feature")
void testNewFeature() throws Exception {
    LOG.info("\n>>> TEST 8: New Feature <<<");

    // Arrange
    // Act
    // Assert

    LOG.info("✅ TEST 8 PASSED");
}
```

### Updating Clinical Scenarios

```java
private PatientContextState createNewScenarioPatient() {
    PatientContextState patient = new PatientContextState();
    // Configure patient for scenario
    return patient;
}
```

### Test Utilities

Located in test helper methods:
- `createTestPatient()` - Basic patient template
- `createSepticShockPatient()` - Sepsis scenario
- `createSTEMIPatient()` - Cardiac scenario
- `convertToEnrichedContext()` - Context conversion

---

## Continuous Integration

### GitHub Actions / CI Pipeline

```yaml
# .github/workflows/phase7-tests.yml
name: Phase 7 Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Set up JDK 17
        uses: actions/setup-java@v2
        with:
          java-version: '17'
      - name: Run Phase 7 Tests
        run: mvn test -Dtest="com.cardiofit.flink.phase7.*"
```

---

**Testing Status**: 🎯 Ready for Execution
**Next Action**: Run `mvn test -Dtest="com.cardiofit.flink.phase7.*"`
**Expected Outcome**: 11/11 tests PASS
