package com.cardiofit.flink.models.diagnostics;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Comprehensive unit tests for LabTest model (Phase 4).
 *
 * Test Coverage:
 * - Result interpretation tests (normal, high, low, critical): 6 tests
 * - Ordering rules and contraindications: 4 tests
 * - Timing and turnaround calculations: 3 tests
 * - Reference range selection: 3 tests
 * - Helper methods: 4 tests
 *
 * Total: 20 unit tests for LabTest
 *
 * @author Module 3 Phase 4 Team
 * @version 1.0
 * @since 2025-10-23
 */
@DisplayName("LabTest Model Tests")
class LabTestTest {

    private LabTest lactateTest;
    private LabTest creatinineTest;

    @BeforeEach
    void setUp() {
        lactateTest = createLactateTest();
        creatinineTest = createCreatinineTest();
    }

    // ============================================================
    // RESULT INTERPRETATION TESTS (6 tests)
    // ============================================================

    @Test
    @DisplayName("Interpret Result: Normal lactate (1.5 mmol/L)")
    void testInterpretResult_NormalLactate() {
        // Given: Normal lactate value
        Double value = 1.5;

        // When: Interpret result
        String interpretation = lactateTest.interpretResult(value, "adult");

        // Then: Should be NORMAL
        assertEquals("NORMAL", interpretation);
        assertFalse(lactateTest.isCritical(value, "adult"));
    }

    @Test
    @DisplayName("Interpret Result: Elevated lactate (3.5 mmol/L)")
    void testInterpretResult_ElevatedLactate() {
        // Given: Elevated lactate (above normal but below critical)
        Double value = 3.5;

        // When: Interpret result
        String interpretation = lactateTest.interpretResult(value, "adult");

        // Then: Should be HIGH
        assertEquals("HIGH", interpretation);
        assertFalse(lactateTest.isCritical(value, "adult"));
    }

    @Test
    @DisplayName("Interpret Result: Critical high lactate (4.5 mmol/L)")
    void testInterpretResult_CriticalHighLactate() {
        // Given: Critical high lactate (≥4.0 mmol/L)
        Double value = 4.5;

        // When: Interpret result
        String interpretation = lactateTest.interpretResult(value, "adult");

        // Then: Should be CRITICAL_HIGH
        assertEquals("CRITICAL_HIGH", interpretation);
        assertTrue(lactateTest.isCritical(value, "adult"));
    }

    @Test
    @DisplayName("Interpret Result: Lactate at critical threshold (4.0 mmol/L)")
    void testInterpretResult_LactateAtThreshold() {
        // Given: Lactate exactly at critical threshold
        Double value = 4.0;

        // When: Interpret result
        String interpretation = lactateTest.interpretResult(value, "adult");

        // Then: Should NOT be critical (> threshold, not >=)
        assertEquals("HIGH", interpretation);
        assertFalse(lactateTest.isCritical(value, "adult"));
    }

    @Test
    @DisplayName("Interpret Result: Low creatinine")
    void testInterpretResult_LowCreatinine() {
        // Given: Low creatinine value
        Double value = 0.3;

        // When: Interpret result
        String interpretation = creatinineTest.interpretResult(value, "adult");

        // Then: Should be LOW
        assertEquals("LOW", interpretation);
    }

    @Test
    @DisplayName("Interpret Result: Null value returns UNKNOWN")
    void testInterpretResult_NullValue() {
        // When: Interpret null value
        String interpretation = lactateTest.interpretResult(null, "adult");

        // Then: Should be UNKNOWN
        assertEquals("UNKNOWN", interpretation);
    }

    // ============================================================
    // ORDERING RULES TESTS (4 tests)
    // ============================================================

    @Test
    @DisplayName("Can Order: No contraindications allows ordering")
    void testCanOrder_NoContraindications() {
        // Given: Patient with no contraindications
        LabTest.PatientContext safePatient = condition -> false;

        // When: Check if can order
        boolean canOrder = lactateTest.canOrder(safePatient);

        // Then: Should allow ordering
        assertTrue(canOrder);
    }

    @Test
    @DisplayName("Can Order: Contraindication prevents ordering")
    void testCanOrder_WithContraindication() {
        // Given: Lactate test with contraindication setup
        LabTest.OrderingRules rules = LabTest.OrderingRules.builder()
            .contraindications(Arrays.asList("active seizure"))
            .build();
        lactateTest.setOrderingRules(rules);

        // Patient has contraindication
        LabTest.PatientContext patientWithSeizure = condition ->
            condition.toLowerCase().contains("seizure");

        // When: Check if can order
        boolean canOrder = lactateTest.canOrder(patientWithSeizure);

        // Then: Should prevent ordering
        assertFalse(canOrder);
    }

    @Test
    @DisplayName("Can Reorder: Minimum interval not met")
    void testCanReorder_MinimumIntervalNotMet() {
        // Given: Lactate ordered 1 hour ago (minimum interval is 2 hours)
        long oneHourAgo = System.currentTimeMillis() - (1000 * 60 * 60);

        // When: Check if can reorder
        boolean canReorder = lactateTest.canReorder(oneHourAgo);

        // Then: Should NOT allow reorder yet
        assertFalse(canReorder);
    }

    @Test
    @DisplayName("Can Reorder: Minimum interval met")
    void testCanReorder_MinimumIntervalMet() {
        // Given: Lactate ordered 3 hours ago (minimum interval is 2 hours)
        long threeHoursAgo = System.currentTimeMillis() - (1000 * 60 * 60 * 3);

        // When: Check if can reorder
        boolean canReorder = lactateTest.canReorder(threeHoursAgo);

        // Then: Should allow reorder
        assertTrue(canReorder);
    }

    // ============================================================
    // TIMING TESTS (3 tests)
    // ============================================================

    @Test
    @DisplayName("Get Turnaround Time: STAT urgency")
    void testGetTurnaroundTime_STAT() {
        // When: Get turnaround time for STAT order
        int turnaround = lactateTest.getTurnaroundTime(true);

        // Then: Should be urgent turnaround (15 minutes)
        assertEquals(15, turnaround);
    }

    @Test
    @DisplayName("Get Turnaround Time: Routine urgency")
    void testGetTurnaroundTime_Routine() {
        // When: Get turnaround time for routine order
        int turnaround = lactateTest.getTurnaroundTime(false);

        // Then: Should be standard turnaround (30 minutes)
        assertEquals(30, turnaround);
    }

    @Test
    @DisplayName("Get Turnaround Time: Missing timing defaults to 120 minutes")
    void testGetTurnaroundTime_MissingTiming() {
        // Given: Lab test with no timing information
        LabTest testWithoutTiming = LabTest.builder()
            .testId("TEST-001")
            .testName("Test Without Timing")
            .build();

        // When: Get turnaround time
        int turnaround = testWithoutTiming.getTurnaroundTime(false);

        // Then: Should default to 120 minutes
        assertEquals(120, turnaround);
    }

    // ============================================================
    // REFERENCE RANGE SELECTION TESTS (3 tests)
    // ============================================================

    @Test
    @DisplayName("Get Reference Range: Adult patient")
    void testGetReferenceRangeForPatient_Adult() {
        // Given: Adult patient (45 years old, male)
        Integer age = 45;
        String sex = "M";

        // When: Get reference range
        LabTest.ReferenceRange range = lactateTest.getReferenceRangeForPatient(age, sex);

        // Then: Should return adult range
        assertNotNull(range);
        assertEquals("adult", range.getPopulation());
        assertEquals(0.5, range.getNormal().getMin());
        assertEquals(2.0, range.getNormal().getMax());
    }

    @Test
    @DisplayName("Get Reference Range: Pediatric patient")
    void testGetReferenceRangeForPatient_Pediatric() {
        // Given: Pediatric patient (10 years old)
        Integer age = 10;
        String sex = "M";

        // When: Get reference range
        LabTest.ReferenceRange range = lactateTest.getReferenceRangeForPatient(age, sex);

        // Then: Should return pediatric range
        assertNotNull(range);
        assertEquals("pediatric", range.getPopulation());
    }

    @Test
    @DisplayName("Get Reference Range: No matching range defaults to adult")
    void testGetReferenceRangeForPatient_DefaultsToAdult() {
        // Given: Patient with age outside all specific ranges
        Integer age = 150;
        String sex = "M";

        // When: Get reference range
        LabTest.ReferenceRange range = lactateTest.getReferenceRangeForPatient(age, sex);

        // Then: Should default to adult range
        assertNotNull(range);
        assertEquals("adult", range.getPopulation());
    }

    // ============================================================
    // HELPER METHOD TESTS (4 tests)
    // ============================================================

    @Test
    @DisplayName("Requires Preparation: Fasting required")
    void testRequiresPreparation_FastingRequired() {
        // Given: Test that requires fasting
        LabTest fastingTest = LabTest.builder()
            .testId("TEST-GLUCOSE")
            .testName("Fasting Glucose")
            .specimen(LabTest.SpecimenRequirements.builder()
                .fastingRequired(true)
                .build())
            .build();

        // When: Check if requires preparation
        boolean requiresPrep = fastingTest.requiresPreparation();

        // Then: Should require preparation
        assertTrue(requiresPrep);
    }

    @Test
    @DisplayName("Requires Preparation: No fasting required")
    void testRequiresPreparation_NoFastingRequired() {
        // When: Check if lactate requires preparation
        boolean requiresPrep = lactateTest.requiresPreparation();

        // Then: Should NOT require preparation
        assertFalse(requiresPrep);
    }

    @Test
    @DisplayName("Test Builder: Complete test construction")
    void testBuilder_CompleteConstruction() {
        // When: Build complete test using builder
        LabTest test = LabTest.builder()
            .testId("TEST-001")
            .testName("Complete Test")
            .loincCode("1234-5")
            .category("CHEMISTRY")
            .build();

        // Then: All fields should be set correctly
        assertNotNull(test);
        assertEquals("TEST-001", test.getTestId());
        assertEquals("Complete Test", test.getTestName());
        assertEquals("1234-5", test.getLoincCode());
        assertEquals("CHEMISTRY", test.getCategory());
    }

    @Test
    @DisplayName("Test YAML Integration: All fields populated")
    void testYAMLIntegration_AllFieldsPopulated() {
        // Then: Verify lactate test has all critical fields from YAML
        assertNotNull(lactateTest.getTestId());
        assertNotNull(lactateTest.getTestName());
        assertNotNull(lactateTest.getLoincCode());
        assertNotNull(lactateTest.getSpecimen());
        assertNotNull(lactateTest.getTiming());
        assertNotNull(lactateTest.getReferenceRanges());
        assertNotNull(lactateTest.getOrderingRules());

        // Verify reference ranges
        assertTrue(lactateTest.getReferenceRanges().containsKey("adult"));
        assertTrue(lactateTest.getReferenceRanges().containsKey("pediatric"));

        // Verify critical values
        LabTest.ReferenceRange adultRange = lactateTest.getReferenceRanges().get("adult");
        assertNotNull(adultRange.getCritical());
        assertEquals(4.0, adultRange.getCritical().getCriticalHigh());
    }

    // ============================================================
    // HELPER METHODS - TEST DATA CREATION
    // ============================================================

    private LabTest createLactateTest() {
        // Create lactate test matching lactate.yaml structure
        Map<String, LabTest.ReferenceRange> ranges = new HashMap<>();

        // Adult range
        ranges.put("adult", LabTest.ReferenceRange.builder()
            .population("adult")
            .sex("ALL")
            .ageMin(18)
            .ageMax(120)
            .unit("mmol/L")
            .normal(LabTest.ReferenceRange.NormalRange.builder()
                .min(0.5)
                .max(2.0)
                .interpretation("Normal tissue perfusion")
                .build())
            .critical(LabTest.ReferenceRange.CriticalRange.builder()
                .criticalHigh(4.0)
                .criticalInterpretation("Severe lactic acidosis")
                .build())
            .build());

        // Pediatric range
        ranges.put("pediatric", LabTest.ReferenceRange.builder()
            .population("pediatric")
            .sex("ALL")
            .ageMin(1)
            .ageMax(17)
            .unit("mmol/L")
            .normal(LabTest.ReferenceRange.NormalRange.builder()
                .min(0.5)
                .max(2.0)
                .interpretation("Normal")
                .build())
            .critical(LabTest.ReferenceRange.CriticalRange.builder()
                .criticalHigh(4.0)
                .criticalInterpretation("Critical")
                .build())
            .build());

        return LabTest.builder()
            .testId("LAB-LACTATE-001")
            .testName("Serum Lactate")
            .loincCode("2524-7")
            .category("CHEMISTRY")
            .specimen(LabTest.SpecimenRequirements.builder()
                .specimenType("BLOOD")
                .collection("Venipuncture")
                .container("Gray top tube")
                .volumeRequired(1.0)
                .fastingRequired(false)
                .build())
            .timing(LabTest.TestTiming.builder()
                .turnaroundTimeMinutes(30)
                .urgentTurnaroundMinutes(15)
                .criticalResultMinutes(15)
                .availability("24/7")
                .pointOfCareAvailable(true)
                .build())
            .referenceRanges(ranges)
            .orderingRules(LabTest.OrderingRules.builder()
                .indications(Arrays.asList("Suspected sepsis", "Shock", "Metabolic acidosis"))
                .minimumIntervalHours(2)
                .requiresConsent(false)
                .appropriatenessLevel("USUALLY_APPROPRIATE")
                .build())
            .evidenceLevel("A")
            .version("1.0")
            .build();
    }

    private LabTest createCreatinineTest() {
        Map<String, LabTest.ReferenceRange> ranges = new HashMap<>();

        ranges.put("adult", LabTest.ReferenceRange.builder()
            .population("adult")
            .sex("ALL")
            .ageMin(18)
            .ageMax(120)
            .unit("mg/dL")
            .normal(LabTest.ReferenceRange.NormalRange.builder()
                .min(0.6)
                .max(1.3)
                .interpretation("Normal renal function")
                .build())
            .critical(LabTest.ReferenceRange.CriticalRange.builder()
                .criticalHigh(5.0)
                .criticalInterpretation("Severe renal failure")
                .build())
            .build());

        return LabTest.builder()
            .testId("LAB-CREATININE-001")
            .testName("Serum Creatinine")
            .loincCode("2160-0")
            .category("CHEMISTRY")
            .referenceRanges(ranges)
            .orderingRules(LabTest.OrderingRules.builder()
                .minimumIntervalHours(24)
                .build())
            .build();
    }
}
