# Module 3: Implementation Phases
## Detailed Development Roadmap for CDS Alignment

**Document Version**: 1.0
**Created**: 2025-10-21
**Purpose**: Phase-by-phase implementation guide with tasks, acceptance criteria, and validation

---

## Overview

This document breaks down the CDS alignment implementation into **3 distinct phases** over **12-18 hours** of development. Each phase has:

- **Clear objectives** and scope
- **Detailed task breakdown** with effort estimates
- **Dependencies** between tasks
- **Acceptance criteria** for phase completion
- **Validation steps** to verify functionality

---

## Table of Contents

1. [Phase 1: Critical Runtime Logic](#phase-1-critical-runtime-logic-5-8-hours)
2. [Phase 2: Quality & Performance](#phase-2-quality--performance-4-6-hours)
3. [Phase 3: Advanced Features](#phase-3-advanced-features-3-4-hours)
4. [Testing Strategy](#testing-strategy)
5. [Integration Plan](#integration-plan)
6. [Deployment Checklist](#deployment-checklist)

---

## Phase 1: Critical Runtime Logic (5-8 hours)

**Goal**: Enable automatic protocol activation and safe medication selection

**Criticality**: **BLOCKING** - System unusable without these components

**Developers**: 2 backend Java developers (parallel work possible)

---

### Phase 1 Tasks

#### Task 1.1: ConditionEvaluator.java (3-4 hours)
**Developer**: Backend Dev 1
**Priority**: **CRITICAL**
**Dependencies**: None

**Subtasks**:
1. **Create class structure** (30 min)
   - Package: `com.cardiofit.flink.cds.evaluation`
   - Enum: `ComparisonOperator` (>=, <=, ==, !=, CONTAINS, NOT_CONTAINS)
   - Enum: `MatchLogic` (ALL_OF, ANY_OF)

2. **Implement evaluate() method** (1 hour)
   - Handle ALL_OF (AND) logic with short-circuit evaluation
   - Handle ANY_OF (OR) logic with short-circuit evaluation
   - Logging for debugging

3. **Implement evaluateCondition() method** (1 hour)
   - Base case: leaf condition evaluation
   - Recursive case: nested conditions
   - Parameter extraction and comparison

4. **Implement compareValues() method** (30 min)
   - Numeric comparison (>=, <=, >, <)
   - Equality (==, !=) for numbers, booleans, strings
   - String operations (CONTAINS, NOT_CONTAINS)
   - Type conversion helpers

5. **Implement extractParameterValue() method** (1 hour)
   - Map common parameter names (lactate, systolic_bp, age, etc.)
   - Handle vital signs, lab values, demographics
   - Use reflection for extensibility
   - Null handling

6. **Unit tests** (1 hour)
   - Simple condition tests (10 tests)
   - ALL_OF logic tests (3 tests)
   - ANY_OF logic tests (3 tests)
   - Nested condition tests (4 tests)
   - Operator tests (6 tests)
   - Parameter extraction tests (5 tests)
   - **Total**: 31 unit tests

**Acceptance Criteria**:
- ✅ ALL_OF logic works correctly with short-circuit evaluation
- ✅ ANY_OF logic works correctly with short-circuit evaluation
- ✅ Nested conditions (3 levels deep) evaluate correctly
- ✅ All 6 operators (>=, <=, ==, !=, CONTAINS, NOT_CONTAINS) work
- ✅ Parameter extraction succeeds for 15+ common parameters
- ✅ All unit tests passing (≥30 tests)
- ✅ Code coverage ≥85%

**Validation**:
```bash
mvn test -Dtest=ConditionEvaluatorTest
# Should pass all tests
```

---

#### Task 1.2: MedicationSelector.java (4-5 hours)
**Developer**: Backend Dev 2
**Priority**: **CRITICAL** (Patient Safety)
**Dependencies**: None (can work in parallel with Task 1.1)

**Subtasks**:
1. **Create class structure** (30 min)
   - Package: `com.cardiofit.flink.cds.medication`
   - Dependencies: EnrichedPatientContext, Protocol models

2. **Implement selectMedication() method** (1.5 hours)
   - Evaluate selection_criteria in order
   - Select primary medication if criteria met
   - Check allergies → select alternative if needed
   - Apply dose adjustments (renal/hepatic)
   - Create new ProtocolAction with selected medication
   - **FAIL SAFE**: Return null if no safe medication available

3. **Implement evaluateCriteria() method** (1 hour)
   - Standard criteria: NO_PENICILLIN_ALLERGY, NO_BETA_LACTAM_ALLERGY
   - Renal criteria: CREATININE_CLEARANCE_GT_40, CREATININE_CLEARANCE_GT_30
   - Risk criteria: MDR_RISK, HIGH_BLEEDING_RISK
   - Contraindication checks: NO_BETA_BLOCKER_CONTRAINDICATION
   - Handle unknown criteria gracefully

4. **Implement hasAllergy() method** (45 min)
   - Direct allergy match (medication name in allergy list)
   - Cross-reactivity checking:
     - Penicillin allergy → cephalosporin cross-reactivity
     - Sulfa allergy → sulfonamide antibiotics
   - Case-insensitive matching

5. **Implement calculateCrCl() method** (30 min)
   - Cockcroft-Gault formula: `((140 - age) × weight) / (72 × Cr)`
   - Female adjustment: multiply by 0.85
   - Parameter validation (age, weight, creatinine)
   - Default to 60.0 mL/min if parameters missing

6. **Implement applyDoseAdjustments() method** (1 hour)
   - Renal dose adjustment (CrCl < 60)
   - Hepatic dose adjustment (Child-Pugh B/C)
   - Medication-specific adjustment rules:
     - Ceftriaxone: 1 g if CrCl < 30
     - Vancomycin: Pharmacist consult if CrCl < 60
     - Levofloxacin: 500 mg q48h if CrCl < 50

7. **Unit tests** (1.5 hours)
   - Selection tests (no allergy → primary, allergy → alternative): 5 tests
   - Criteria evaluation tests (all standard criteria): 8 tests
   - Allergy detection tests (direct match, cross-reactivity): 6 tests
   - CrCl calculation tests (male, female, edge cases): 5 tests
   - Dose adjustment tests (renal, hepatic): 6 tests
   - **Total**: 30 unit tests

**Acceptance Criteria**:
- ✅ Medication selection respects documented allergies
- ✅ Alternative medications selected when primary contraindicated
- ✅ CrCl calculation accurate within 1 mL/min of expected
- ✅ Renal dose adjustments applied for CrCl < 60
- ✅ Hepatic dose adjustments applied for Child-Pugh B/C
- ✅ FAIL SAFE: Returns null if no safe medication (no silent failures)
- ✅ All unit tests passing (≥30 tests)
- ✅ Code coverage ≥85%

**Validation**:
```bash
mvn test -Dtest=MedicationSelectorTest
# Should pass all tests

# Safety validation
# Create test case: penicillin allergy + cephalosporin primary → should use alternative
# Create test case: allergic to both primary and alternative → should return null
```

---

#### Task 1.3: TimeConstraintTracker.java (3-4 hours)
**Developer**: Backend Dev 1 (after Task 1.1 complete)
**Priority**: **CRITICAL**
**Dependencies**: Task 1.1 (ConditionEvaluator)

**Subtasks**:
1. **Create class structure** (30 min)
   - Package: `com.cardiofit.flink.cds.time`
   - Classes: TimeConstraintStatus, ConstraintStatus
   - Enum: AlertLevel (INFO, WARNING, CRITICAL)

2. **Implement evaluateConstraints() method** (1.5 hours)
   - Extract trigger time from context (use current time if missing)
   - Iterate through protocol time constraints
   - Evaluate each constraint (call evaluateConstraint())
   - Collect ConstraintStatus objects
   - Log WARNING/CRITICAL alerts

3. **Implement evaluateConstraint() method** (1 hour)
   - Calculate deadline: trigger_time + offset_minutes
   - Calculate time remaining: deadline - current_time
   - Determine alert level (INFO, WARNING, CRITICAL)
   - Generate human-readable message
   - Create ConstraintStatus object

4. **Implement determineAlertLevel() method** (30 min)
   - CRITICAL: time_remaining < 0 (deadline exceeded)
   - WARNING: 0 ≤ time_remaining ≤ 30 minutes
   - INFO: time_remaining > 30 minutes

5. **Helper methods** (30 min)
   - generateMessage(): Human-readable status message
   - formatDuration(): Format duration as "Xh Ym" or "Xm"

6. **Unit tests** (1 hour)
   - On-track tests (>30 min remaining): 2 tests
   - Warning tests (10-30 min remaining): 3 tests
   - Critical tests (deadline exceeded): 3 tests
   - Bundle compliance tests: 2 tests
   - **Total**: 10 unit tests

**Acceptance Criteria**:
- ✅ Deadline calculation accurate (trigger_time + offset_minutes)
- ✅ WARNING alert generated when <30 min remaining
- ✅ CRITICAL alert generated when deadline exceeded
- ✅ Human-readable messages (e.g., "Hour-1 Bundle deadline in 15 minutes")
- ✅ All unit tests passing (≥10 tests)
- ✅ Code coverage ≥85%

**Validation**:
```bash
mvn test -Dtest=TimeConstraintTrackerTest

# Integration test:
# - Sepsis protocol with Hour-1 bundle
# - Trigger time: now - 45 minutes
# - Deadline: now + 15 minutes
# - Expected: WARNING alert "Hour-1 Bundle deadline in 15 minutes"
```

---

#### Task 1.4: Integration - Update ProtocolMatcher.java (1-2 hours)
**Developer**: Backend Dev 2 (after Task 1.1 complete)
**Priority**: **HIGH**
**Dependencies**: Task 1.1 (ConditionEvaluator)

**Subtasks**:
1. **Add ConditionEvaluator dependency** (15 min)
   - Constructor injection
   - Instantiate in ClinicalRecommendationEngine

2. **Update matchProtocols() method** (1 hour)
   - Iterate through all protocols
   - Evaluate trigger_criteria using ConditionEvaluator
   - Collect triggered protocols
   - Log trigger evaluation results

3. **Unit tests** (45 min)
   - Test protocol matching with simple triggers: 2 tests
   - Test protocol matching with complex triggers (AND/OR): 3 tests
   - Test no protocols match: 1 test
   - **Total**: 6 unit tests

**Acceptance Criteria**:
- ✅ Protocols with trigger_criteria automatically activate when conditions met
- ✅ Protocols without trigger_criteria ignored
- ✅ Logging shows which protocols triggered and why
- ✅ All unit tests passing

**Validation**:
```bash
mvn test -Dtest=ProtocolMatcherTest

# Integration test:
# - Patient: lactate 3.5, SBP 85, infection suspected
# - Expected: Sepsis protocol triggers
```

---

#### Task 1.5: Integration - Update ActionBuilder.java (1-2 hours)
**Developer**: Backend Dev 1 (after Task 1.2 complete)
**Priority**: **HIGH**
**Dependencies**: Task 1.2 (MedicationSelector), Task 1.3 (TimeConstraintTracker)

**Subtasks**:
1. **Add dependencies** (15 min)
   - MedicationSelector (constructor injection)
   - TimeConstraintTracker (constructor injection)

2. **Update buildActions() method** (1 hour)
   - Call MedicationSelector.selectMedication() for each action
   - Handle null returns (no safe medication) with warning
   - Apply time constraints using TimeConstraintTracker
   - Return selected actions

3. **Unit tests** (45 min)
   - Test medication selection integration: 3 tests
   - Test time constraint application: 2 tests
   - Test null medication handling: 1 test
   - **Total**: 6 unit tests

**Acceptance Criteria**:
- ✅ Medication selection applied to all actions with medication_selection
- ✅ Time constraints tracked for all actions
- ✅ Null medication returns logged as errors
- ✅ All unit tests passing

---

### Phase 1 Validation

**End-to-End Test**: ROHAN-001 Sepsis Patient

```java
@Test
void testPhase1_SepsisPatient_ROHAN001() {
    // Given: Sepsis patient (ROHAN-001 test case)
    EnrichedPatientContext context = new EnrichedPatientContext();
    PatientState state = new PatientState();
    state.setPatientId("ROHAN-001");
    state.setLactate(3.8);
    state.setSystolicBP(88);
    state.setInfectionSuspected(true);
    state.setAllergies(Arrays.asList("penicillin"));
    state.setAge(72);
    state.setWeight(70.0);
    state.setCreatinine(1.5);
    context.setPatientState(state);
    context.setTriggerTime(Instant.now().minus(30, ChronoUnit.MINUTES));

    // When: Generate recommendations
    ClinicalRecommendationEngine engine = new ClinicalRecommendationEngine();
    List<ClinicalRecommendation> recommendations = engine.generateRecommendations(context);

    // Then: Sepsis protocol triggers
    assertFalse(recommendations.isEmpty(), "Should have recommendations");

    ClinicalRecommendation sepsisRec = recommendations.stream()
        .filter(r -> r.getProtocolId().equals("SEPSIS-BUNDLE-001"))
        .findFirst()
        .orElse(null);

    assertNotNull(sepsisRec, "Sepsis protocol should trigger");

    // Medication selection respects penicillin allergy
    List<ProtocolAction> actions = sepsisRec.getActions();
    ProtocolAction antibioticAction = actions.stream()
        .filter(a -> a.getType() == ActionType.MEDICATION)
        .findFirst()
        .orElse(null);

    assertNotNull(antibioticAction);
    assertEquals("Levofloxacin", antibioticAction.getMedication().getName(),
        "Should use alternative due to penicillin allergy");

    // Time constraint tracking
    TimeConstraintStatus timeStatus = sepsisRec.getTimeConstraintStatus();
    assertNotNull(timeStatus);
    assertTrue(timeStatus.getConstraintStatuses().size() > 0);

    ConstraintStatus hour1Bundle = timeStatus.getConstraintStatuses().stream()
        .filter(cs -> cs.getBundleName().contains("Hour-1"))
        .findFirst()
        .orElse(null);

    assertNotNull(hour1Bundle);
    assertEquals(AlertLevel.INFO, hour1Bundle.getAlertLevel(),
        "30 minutes into hour-1 bundle, should be INFO");
}
```

**Acceptance Criteria for Phase 1 Complete**:
- ✅ ConditionEvaluator: All 31 unit tests passing
- ✅ MedicationSelector: All 30 unit tests passing
- ✅ TimeConstraintTracker: All 10 unit tests passing
- ✅ ProtocolMatcher integration: All 6 unit tests passing
- ✅ ActionBuilder integration: All 6 unit tests passing
- ✅ End-to-end test (ROHAN-001): PASSING
- ✅ No compilation errors
- ✅ Code coverage ≥85% for all new classes

**Phase 1 Deliverables**:
1. ConditionEvaluator.java (400 lines, 31 tests)
2. MedicationSelector.java (650 lines, 30 tests)
3. TimeConstraintTracker.java (500 lines, 10 tests)
4. Updated ProtocolMatcher.java (+50 lines, 6 tests)
5. Updated ActionBuilder.java (+100 lines, 6 tests)
6. End-to-end test suite (ROHAN-001 test case)

**Estimated Effort**: 5-8 hours (parallel development possible)

---

## Phase 2: Quality & Performance (4-6 hours)

**Goal**: Add confidence ranking, validation, and optimized protocol lookup

**Criticality**: **HIGH** - Required for production readiness

**Developers**: 2 backend Java developers (parallel work possible)

---

### Phase 2 Tasks

#### Task 2.1: ConfidenceCalculator.java (2-3 hours)
**Developer**: Backend Dev 1
**Priority**: **HIGH**
**Dependencies**: Phase 1 (ConditionEvaluator)

**Subtasks**:
1. **Create class structure** (30 min)
   - Package: `com.cardiofit.flink.cds.evaluation`
   - Dependencies: ConditionEvaluator (for modifier evaluation)

2. **Implement calculateConfidence() method** (1 hour)
   - Start with base_confidence from protocol
   - Iterate through modifiers
   - Evaluate modifier condition using ConditionEvaluator
   - Add adjustment if condition met
   - Clamp final score to [0.0, 1.0]

3. **Implement meetsActivationThreshold() method** (30 min)
   - Compare confidence to activation_threshold
   - Use default 0.70 if not specified

4. **Helper methods** (15 min)
   - clamp(): Clamp value to [min, max]

5. **Unit tests** (1 hour)
   - No modifiers (base confidence): 1 test
   - Positive modifiers: 3 tests
   - Negative modifiers: 2 tests
   - Clamping above 1.0: 1 test
   - Clamping below 0.0: 1 test
   - Activation threshold: 3 tests
   - **Total**: 11 unit tests

**Acceptance Criteria**:
- ✅ Confidence calculation accurate (base + sum of modifiers)
- ✅ Clamping to [0.0, 1.0] works correctly
- ✅ Activation threshold filtering works
- ✅ All unit tests passing (≥11 tests)
- ✅ Code coverage ≥85%

---

#### Task 2.2: ProtocolValidator.java (2 hours)
**Developer**: Backend Dev 2
**Priority**: **HIGH**
**Dependencies**: None (can work in parallel with Task 2.1)

**Subtasks**:
1. **Create class structure** (30 min)
   - Package: `com.cardiofit.flink.cds.validation`
   - Class: ValidationResult (errors/warnings lists)

2. **Implement validate() method** (30 min)
   - Call all validation methods
   - Collect errors and warnings in ValidationResult
   - Log validation outcome

3. **Validation methods** (1 hour)
   - validateRequiredFields(): protocol_id, name, category, actions
   - validateActionReferences(): unique action_ids, no duplicates
   - validateConditionReferences(): valid condition structures
   - validateConfidenceScoring(): base_confidence [0.0, 1.0], threshold [0.0, 1.0]
   - validateTimeConstraints(): valid offset_minutes, valid action references
   - validateEvidenceSource(): primary_guideline, evidence_level present

4. **Unit tests** (30 min)
   - Valid protocol passes: 1 test
   - Missing required fields: 4 tests
   - Duplicate action_ids: 1 test
   - Invalid confidence ranges: 2 tests
   - **Total**: 8 unit tests

**Acceptance Criteria**:
- ✅ Required field validation works
- ✅ Duplicate action_id detection works
- ✅ Confidence score range validation works
- ✅ Validation errors reported with clear messages
- ✅ All unit tests passing (≥8 tests)
- ✅ Code coverage ≥85%

---

#### Task 2.3: KnowledgeBaseManager.java (4-5 hours)
**Developer**: Backend Dev 1 (after Task 2.1 complete)
**Priority**: **HIGH**
**Dependencies**: Task 2.2 (ProtocolValidator)

**Subtasks**:
1. **Create class structure** (1 hour)
   - Package: `com.cardiofit.flink.cds.knowledge`
   - Singleton pattern with double-checked locking
   - ConcurrentHashMap for thread-safe storage
   - CopyOnWriteArrayList for indexes

2. **Implement singleton getInstance()** (30 min)
   - Double-checked locking
   - Thread-safe initialization

3. **Implement loadAllProtocols()** (1 hour)
   - Call ProtocolLoader.loadProtocols()
   - Validate each protocol using ProtocolValidator
   - Reject invalid protocols with error logging
   - Clear existing storage
   - Add to ConcurrentHashMap
   - Build indexes

4. **Implement buildIndexes()** (30 min)
   - Category index: Map<ProtocolCategory, List<Protocol>>
   - Specialty index: Map<String, List<Protocol>>

5. **Query methods** (30 min)
   - getProtocol(id): Direct HashMap lookup
   - getAllProtocols(): Return all protocols
   - getByCategory(): Use category index
   - getBySpecialty(): Use specialty index
   - search(query): Filter by name/id/category

6. **Hot reload** (1 hour)
   - initializeWatchService(): FileWatcher for YAML directory
   - startWatchService(): Background thread monitoring file changes
   - reloadProtocols(): Thread-safe reload with lock

7. **Unit tests** (1 hour)
   - Singleton test: 1 test
   - Protocol lookup tests: 3 tests
   - Category index tests: 2 tests
   - Specialty index tests: 2 tests
   - Search tests: 2 tests
   - Hot reload tests: 2 tests
   - **Total**: 12 unit tests

**Acceptance Criteria**:
- ✅ Singleton pattern works (same instance returned)
- ✅ All protocols loaded and validated
- ✅ Category index lookup <5ms
- ✅ Specialty index lookup <5ms
- ✅ Search works for name/id/category
- ✅ Hot reload triggers on file change
- ✅ Thread-safe under concurrent access
- ✅ All unit tests passing (≥12 tests)
- ✅ Code coverage ≥80%

---

#### Task 2.4: Integration - Update ProtocolMatcher with Confidence (1 hour)
**Developer**: Backend Dev 2
**Priority**: **HIGH**
**Dependencies**: Task 2.1 (ConfidenceCalculator)

**Subtasks**:
1. **Add ConfidenceCalculator dependency** (15 min)

2. **Update matchProtocols() method** (45 min)
   - After trigger evaluation, calculate confidence for each matched protocol
   - Filter by activation_threshold
   - Sort matched protocols by confidence (descending)
   - Log confidence scores

3. **Unit tests** (30 min)
   - Multiple protocols match, ranked by confidence: 2 tests
   - Protocol below threshold filtered out: 1 test
   - **Total**: 3 tests

**Acceptance Criteria**:
- ✅ Confidence scores calculated for all matched protocols
- ✅ Protocols below activation_threshold filtered out
- ✅ Recommendations sorted by confidence (highest first)
- ✅ All unit tests passing

---

#### Task 2.5: Integration - Update ProtocolLoader with Validation (1 hour)
**Developer**: Backend Dev 1
**Priority**: **HIGH**
**Dependencies**: Task 2.2 (ProtocolValidator)

**Subtasks**:
1. **Add ProtocolValidator dependency** (15 min)

2. **Update loadProtocols() method** (45 min)
   - Validate each protocol after loading
   - Log validation errors/warnings
   - Reject protocols with errors
   - Accept protocols with only warnings

3. **Unit tests** (30 min)
   - Valid protocol loads successfully: 1 test
   - Invalid protocol rejected: 1 test
   - **Total**: 2 tests

**Acceptance Criteria**:
- ✅ All loaded protocols pass validation
- ✅ Invalid protocols rejected with error messages
- ✅ All unit tests passing

---

### Phase 2 Validation

**Performance Test**:
```java
@Test
void testPhase2_Performance() {
    KnowledgeBaseManager kb = KnowledgeBaseManager.getInstance();

    // Category lookup performance (<5ms)
    long start = System.currentTimeMillis();
    List<Protocol> infectious = kb.getByCategory(ProtocolCategory.INFECTIOUS);
    long duration = System.currentTimeMillis() - start;

    assertTrue(duration < 5, "Category lookup should be <5ms, was: " + duration);

    // Confidence ranking
    EnrichedPatientContext context = createTestContext();
    ProtocolMatcher matcher = new ProtocolMatcher(
        new ConditionEvaluator(),
        new ConfidenceCalculator(new ConditionEvaluator())
    );

    List<Protocol> matched = matcher.matchProtocols(context, kb.getAllProtocols());

    // Should be sorted by confidence (descending)
    for (int i = 0; i < matched.size() - 1; i++) {
        double conf1 = matched.get(i).getConfidence();
        double conf2 = matched.get(i + 1).getConfidence();
        assertTrue(conf1 >= conf2, "Protocols should be sorted by confidence");
    }
}
```

**Acceptance Criteria for Phase 2 Complete**:
- ✅ ConfidenceCalculator: All 11 unit tests passing
- ✅ ProtocolValidator: All 8 unit tests passing
- ✅ KnowledgeBaseManager: All 12 unit tests passing
- ✅ ProtocolMatcher confidence integration: All 3 unit tests passing
- ✅ ProtocolLoader validation integration: All 2 unit tests passing
- ✅ Performance test: Category lookup <5ms
- ✅ No compilation errors
- ✅ Code coverage ≥85% for all new classes

**Phase 2 Deliverables**:
1. ConfidenceCalculator.java (300 lines, 11 tests)
2. ProtocolValidator.java (250 lines, 8 tests)
3. KnowledgeBaseManager.java (750 lines, 12 tests)
4. Updated ProtocolMatcher.java (+50 lines, 3 tests)
5. Updated ProtocolLoader.java (+50 lines, 2 tests)

**Estimated Effort**: 4-6 hours (parallel development possible)

---

## Phase 3: Advanced Features (3-4 hours)

**Goal**: Add special populations and auto-escalation

**Criticality**: **MEDIUM** - Nice-to-have for comprehensive CDS

**Developers**: 1-2 backend Java developers

---

### Phase 3 Tasks

#### Task 3.1: EscalationRuleEvaluator.java (2-3 hours)
**Developer**: Backend Dev 1
**Priority**: **MEDIUM**
**Dependencies**: Phase 1 (ConditionEvaluator)

**Subtasks**:
1. **Create class structure** (30 min)
   - Package: `com.cardiofit.flink.cds.escalation`
   - Classes: EscalationRecommendation, SpecialistConsultation
   - Enum: EscalationLevel (CONSULT, TRANSFER, IMMEDIATE_TRANSFER)

2. **Implement evaluateEscalation() method** (1 hour)
   - Iterate through protocol escalation_rules
   - Evaluate escalation_trigger using ConditionEvaluator
   - Create EscalationRecommendation if triggered
   - Gather clinical evidence
   - Sort by escalation level (IMMEDIATE first)

3. **Implement gatherClinicalEvidence() method** (30 min)
   - Extract parameters from trigger conditions
   - Get parameter values from patient context
   - Build evidence map

4. **Unit tests** (1 hour)
   - Septic shock triggers ICU: 2 tests
   - No escalation triggers: 1 test
   - Multiple escalations, sorted: 1 test
   - Clinical evidence captured: 2 tests
   - **Total**: 6 unit tests

**Acceptance Criteria**:
- ✅ Escalation rules trigger correctly
- ✅ ICU transfer recommendations generated
- ✅ Clinical evidence included in recommendation
- ✅ Escalations sorted by level (IMMEDIATE first)
- ✅ All unit tests passing (≥6 tests)
- ✅ Code coverage ≥85%

---

#### Task 3.2: Enhanced YAML Migration (3-4 hours)
**Developer**: Backend Dev 2
**Priority**: **MEDIUM**
**Dependencies**: None (can work in parallel with Task 3.1)

**Subtasks**:
1. **Add special_populations to sepsis protocol** (30 min)
   - Elderly: Reduced doses for CrCl <30
   - Pregnancy: Medication contraindications
   - Immunocompromised: Broader antimicrobial coverage

2. **Add escalation_rules to sepsis protocol** (30 min)
   - Septic shock (lactate ≥4.0): ICU_TRANSFER
   - Persistent hypotension: CONSULT
   - Multi-organ dysfunction (SOFA ≥8): IMMEDIATE_TRANSFER

3. **Migrate 5 additional high-priority protocols** (2 hours)
   - STEMI: Add special_populations, escalation_rules
   - Stroke: Add special_populations, escalation_rules
   - DKA: Add special_populations, escalation_rules
   - Heart Failure: Add special_populations, escalation_rules
   - Pneumonia: Add special_populations, escalation_rules

4. **Validation** (1 hour)
   - Run ProtocolValidator on all migrated protocols
   - Fix validation errors
   - Verify YAML syntax

**Acceptance Criteria**:
- ✅ 6 protocols enhanced (Sepsis, STEMI, Stroke, DKA, HF, Pneumonia)
- ✅ All enhanced protocols pass ProtocolValidator
- ✅ No YAML syntax errors
- ✅ special_populations correctly structured
- ✅ escalation_rules correctly structured

**Validation**:
```bash
# Validate all enhanced protocols
java -jar protocol-validator.jar sepsis-management.yaml
java -jar protocol-validator.jar stemi-management.yaml
# ... (all 6)

# All should return: VALIDATION PASSED
```

---

#### Task 3.3: Integration - Update ClinicalRecommendationEngine (1 hour)
**Developer**: Backend Dev 1 (after Task 3.1 complete)
**Priority**: **MEDIUM**
**Dependencies**: Task 3.1 (EscalationRuleEvaluator)

**Subtasks**:
1. **Add EscalationRuleEvaluator dependency** (15 min)

2. **Update generateRecommendations() method** (45 min)
   - After action building, evaluate escalation rules
   - Add escalation recommendations to ClinicalRecommendation
   - Log escalation recommendations

3. **Unit tests** (30 min)
   - Escalation recommendation generated: 2 tests
   - No escalation: 1 test
   - **Total**: 3 tests

**Acceptance Criteria**:
- ✅ Escalation recommendations included in output
- ✅ Logging shows escalation triggers
- ✅ All unit tests passing

---

### Phase 3 Validation

**End-to-End Test**: Septic Shock Patient with ICU Escalation

```java
@Test
void testPhase3_SepticShock_ICUEscalation() {
    // Given: Septic shock patient (lactate ≥4.0)
    EnrichedPatientContext context = new EnrichedPatientContext();
    PatientState state = new PatientState();
    state.setLactate(4.5);
    state.setSystolicBP(85);
    state.setMeanArterialPressure(58);
    state.setInfectionSuspected(true);
    context.setPatientState(state);

    // When: Generate recommendations
    ClinicalRecommendationEngine engine = new ClinicalRecommendationEngine();
    List<ClinicalRecommendation> recommendations = engine.generateRecommendations(context);

    // Then: Sepsis protocol triggers with ICU escalation
    ClinicalRecommendation sepsisRec = recommendations.stream()
        .filter(r -> r.getProtocolId().equals("SEPSIS-BUNDLE-001"))
        .findFirst()
        .orElse(null);

    assertNotNull(sepsisRec);

    // Escalation recommendation generated
    List<EscalationRecommendation> escalations = sepsisRec.getEscalationRecommendations();
    assertFalse(escalations.isEmpty(), "Should have escalation recommendation");

    EscalationRecommendation icuEscalation = escalations.stream()
        .filter(e -> e.getEscalationLevel() == EscalationLevel.ICU_TRANSFER)
        .findFirst()
        .orElse(null);

    assertNotNull(icuEscalation, "Should recommend ICU transfer");
    assertTrue(icuEscalation.getRationale().contains("septic shock"),
        "Rationale should mention septic shock");
    assertEquals(4.5, icuEscalation.getClinicalEvidence().get("lactate"),
        "Clinical evidence should include lactate value");
}
```

**Acceptance Criteria for Phase 3 Complete**:
- ✅ EscalationRuleEvaluator: All 6 unit tests passing
- ✅ 6 protocols enhanced with special_populations and escalation_rules
- ✅ All enhanced protocols pass ProtocolValidator
- ✅ ClinicalRecommendationEngine integration: All 3 unit tests passing
- ✅ End-to-end test (septic shock ICU escalation): PASSING
- ✅ No compilation errors
- ✅ Code coverage ≥85% for EscalationRuleEvaluator

**Phase 3 Deliverables**:
1. EscalationRuleEvaluator.java (350 lines, 6 tests)
2. 6 enhanced protocol YAML files (Sepsis, STEMI, Stroke, DKA, HF, Pneumonia)
3. Updated ClinicalRecommendationEngine.java (+50 lines, 3 tests)

**Estimated Effort**: 3-4 hours

---

## Testing Strategy

### Unit Testing (Per Phase)

**Phase 1 Unit Tests**: 83 tests
- ConditionEvaluator: 31 tests
- MedicationSelector: 30 tests
- TimeConstraintTracker: 10 tests
- ProtocolMatcher: 6 tests
- ActionBuilder: 6 tests

**Phase 2 Unit Tests**: 36 tests
- ConfidenceCalculator: 11 tests
- ProtocolValidator: 8 tests
- KnowledgeBaseManager: 12 tests
- ProtocolMatcher: 3 tests
- ProtocolLoader: 2 tests

**Phase 3 Unit Tests**: 9 tests
- EscalationRuleEvaluator: 6 tests
- ClinicalRecommendationEngine: 3 tests

**Total Unit Tests**: 128 tests

### Integration Testing

**Integration Test 1**: ROHAN-001 Sepsis Patient (Phase 1)
- Trigger evaluation: lactate ≥2.0, SBP <90, infection suspected
- Medication selection: Levofloxacin (penicillin allergy)
- Time tracking: Hour-1 bundle (30 min into bundle)

**Integration Test 2**: Multiple Protocols Ranking (Phase 2)
- Patient matches both Sepsis and Pneumonia
- Confidence scores: Sepsis 0.92, Pneumonia 0.78
- Recommendations sorted by confidence

**Integration Test 3**: Septic Shock ICU Escalation (Phase 3)
- Lactate 4.5 triggers septic shock escalation
- ICU_TRANSFER recommendation generated
- Clinical evidence includes lactate value

### Performance Testing

**Performance Benchmarks**:
- Protocol lookup (by ID): <2ms
- Category index lookup: <5ms
- Trigger evaluation (16 protocols): <20ms
- Confidence calculation (16 protocols): <10ms
- Medication selection: <30ms
- Time tracking: <10ms
- Escalation evaluation: <10ms
- **Total recommendation generation**: <100ms

**Load Testing**:
- 100 concurrent requests: <150ms p95 latency
- 1000 concurrent requests: <250ms p95 latency

---

## Integration Plan

### Component Integration Order

```
Phase 1:
  ConditionEvaluator → ProtocolMatcher (trigger evaluation)
  MedicationSelector → ActionBuilder (medication selection)
  TimeConstraintTracker → ActionBuilder (time tracking)

Phase 2:
  ConditionEvaluator → ConfidenceCalculator (modifier evaluation)
  ConfidenceCalculator → ProtocolMatcher (confidence ranking)
  ProtocolValidator → ProtocolLoader (validation at load time)
  ProtocolLoader → KnowledgeBaseManager (validated protocol storage)

Phase 3:
  ConditionEvaluator → EscalationRuleEvaluator (trigger evaluation)
  EscalationRuleEvaluator → ClinicalRecommendationEngine (escalation recommendations)
```

### Dependency Injection

**ClinicalRecommendationEngine Constructor**:
```java
public ClinicalRecommendationEngine() {
    this.conditionEvaluator = new ConditionEvaluator();
    this.confidenceCalculator = new ConfidenceCalculator(conditionEvaluator);
    this.medicationSelector = new MedicationSelector();
    this.timeTracker = new TimeConstraintTracker();
    this.escalationEvaluator = new EscalationRuleEvaluator(conditionEvaluator);
    this.knowledgeBase = KnowledgeBaseManager.getInstance();

    this.protocolMatcher = new ProtocolMatcher(
        conditionEvaluator,
        confidenceCalculator
    );

    this.actionBuilder = new ActionBuilder(
        medicationSelector,
        timeTracker
    );
}
```

---

## Deployment Checklist

### Pre-Deployment

- [ ] All unit tests passing (128 tests)
- [ ] All integration tests passing (3 tests)
- [ ] Performance tests passing (<100ms recommendation generation)
- [ ] Code coverage ≥85% for all new classes
- [ ] No compilation errors or warnings
- [ ] SonarQube analysis: No critical issues
- [ ] All enhanced protocols validated (6 protocols)

### Deployment Steps

1. **Build**:
   ```bash
   mvn clean package -DskipTests=false
   ```

2. **Run full test suite**:
   ```bash
   mvn test
   # Should show: Tests run: 128, Failures: 0, Errors: 0
   ```

3. **Deploy to staging**:
   ```bash
   # Copy JAR to staging environment
   # Run integration tests in staging
   ```

4. **Smoke test in staging**:
   - Load all 16 protocols successfully
   - Generate recommendations for ROHAN-001 test case
   - Verify protocol matching, medication selection, time tracking

5. **Deploy to production**:
   - Blue-green deployment
   - Monitor recommendation generation latency (<100ms)
   - Monitor error rates (should be 0%)

### Post-Deployment Monitoring

**Metrics to Track**:
- Recommendation generation latency (p50, p95, p99)
- Protocol match rate (% of requests that match a protocol)
- Confidence score distribution
- Time constraint alert rate (WARNING, CRITICAL)
- Escalation recommendation rate
- Medication selection alternative rate (% using alternatives due to allergies)

**Alerts**:
- Latency >200ms: WARNING
- Latency >500ms: CRITICAL
- Protocol load failure: CRITICAL
- Validation errors: WARNING
- Medication selection returns null: CRITICAL (patient safety)

---

## Summary

### Total Implementation Effort

| Phase | Effort | Components | Tests |
|-------|--------|------------|-------|
| Phase 1 | 5-8 hours | 3 new classes, 2 integrations | 83 tests |
| Phase 2 | 4-6 hours | 3 new classes, 2 integrations | 36 tests |
| Phase 3 | 3-4 hours | 1 new class, 1 integration, 6 YAML migrations | 9 tests |
| **Total** | **12-18 hours** | **7 new classes, 5 integrations, 6 YAMLs** | **128 tests** |

### Completion Criteria

**Phase 1 Complete** (CRITICAL):
- ✅ Protocols activate automatically based on patient state
- ✅ Medication selection respects allergies and selects safe alternatives
- ✅ Time-critical interventions tracked with deadline alerts
- ✅ ROHAN-001 test case generates correct sepsis recommendations

**Phase 2 Complete** (HIGH):
- ✅ Multiple protocols ranked by confidence
- ✅ All protocols validated at load time
- ✅ Fast protocol lookup using indexes (<5ms)
- ✅ Performance benchmarks met (<100ms total)

**Phase 3 Complete** (MEDIUM):
- ✅ Special population modifications applied
- ✅ Escalation recommendations generated for ICU criteria
- ✅ 6 protocols enhanced with advanced features
- ✅ End-to-end septic shock test passing

### Next Steps

1. **Review this implementation plan** with development team
2. **Assign developers** to Phase 1 tasks (parallel work)
3. **Begin Phase 1 implementation** (5-8 hours)
4. **Validate Phase 1** with ROHAN-001 test case
5. **Proceed to Phase 2** if Phase 1 validation passes
6. **Optional Phase 3** based on timeline and priorities

---

**Document Status**: COMPLETE
**Ready for**: Development Team Implementation
