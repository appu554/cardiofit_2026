package com.cardiofit.flink.models.diagnostics;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Comprehensive unit tests for TestResult model (Phase 4).
 *
 * Test Coverage:
 * - Result interpretation: 5 tests
 * - Trending and delta checks: 4 tests
 * - Critical value detection: 3 tests
 * - Quality and specimen checks: 3 tests
 * - Helper methods: 5 tests
 *
 * Total: 20 unit tests for TestResult
 *
 * @author Module 3 Phase 4 Team
 * @version 1.0
 * @since 2025-10-23
 */
@DisplayName("TestResult Model Tests")
class TestResultTest {

    private TestResult normalLactateResult;
    private TestResult criticalLactateResult;
    private TestResult creatinineResultWithTrend;

    @BeforeEach
    void setUp() {
        normalLactateResult = createNormalLactateResult();
        criticalLactateResult = createCriticalLactateResult();
        creatinineResultWithTrend = createCreatinineResultWithTrend();
    }

    // ============================================================
    // RESULT INTERPRETATION TESTS (5 tests)
    // ============================================================

    @Test
    @DisplayName("Interpret Value: Normal result")
    void testInterpretValue_Normal() {
        // When: Interpret normal lactate (1.5 mmol/L)
        TestResult.ResultInterpretation interpretation = normalLactateResult.interpretValue();

        // Then: Should be NORMAL
        assertEquals(TestResult.ResultInterpretation.NORMAL, interpretation);
        assertTrue(normalLactateResult.isNormal());
        assertFalse(normalLactateResult.isAbnormal());
    }

    @Test
    @DisplayName("Interpret Value: Critical high result")
    void testInterpretValue_CriticalHigh() {
        // When: Interpret critical lactate (4.5 mmol/L)
        TestResult.ResultInterpretation interpretation = criticalLactateResult.interpretValue();

        // Then: Should be CRITICAL_HIGH
        assertEquals(TestResult.ResultInterpretation.CRITICAL_HIGH, interpretation);
        assertTrue(criticalLactateResult.isCritical());
        assertTrue(criticalLactateResult.requiresImmediateAction());
    }

    @Test
    @DisplayName("Interpret Value: High but not critical")
    void testInterpretValue_HighButNotCritical() {
        // Given: Lactate 3.0 mmol/L (high but < 4.0 critical)
        TestResult result = TestResult.builder()
            .testName("Lactate")
            .numericValue(3.0)
            .referenceRange(TestResult.ReferenceRange.builder()
                .normalLow(0.5)
                .normalHigh(2.0)
                .criticalHigh(4.0)
                .unit("mmol/L")
                .build())
            .build();

        // When: Interpret value
        TestResult.ResultInterpretation interpretation = result.interpretValue();

        // Then: Should be HIGH, not CRITICAL_HIGH
        assertEquals(TestResult.ResultInterpretation.HIGH, interpretation);
        assertFalse(result.requiresImmediateAction());
    }

    @Test
    @DisplayName("Interpret Value: Low result")
    void testInterpretValue_Low() {
        // Given: Creatinine 0.3 mg/dL (low)
        TestResult result = TestResult.builder()
            .testName("Creatinine")
            .numericValue(0.3)
            .referenceRange(TestResult.ReferenceRange.builder()
                .normalLow(0.6)
                .normalHigh(1.3)
                .unit("mg/dL")
                .build())
            .build();

        // When: Interpret value
        TestResult.ResultInterpretation interpretation = result.interpretValue();

        // Then: Should be LOW
        assertEquals(TestResult.ResultInterpretation.LOW, interpretation);
    }

    @Test
    @DisplayName("Interpret Value: Null value returns INDETERMINATE")
    void testInterpretValue_NullValue() {
        // Given: Result with null numeric value
        TestResult result = TestResult.builder()
            .testName("Test")
            .numericValue(null)
            .referenceRange(TestResult.ReferenceRange.builder()
                .normalLow(0.5)
                .normalHigh(2.0)
                .build())
            .build();

        // When: Interpret value
        TestResult.ResultInterpretation interpretation = result.interpretValue();

        // Then: Should be INDETERMINATE
        assertEquals(TestResult.ResultInterpretation.INDETERMINATE, interpretation);
    }

    // ============================================================
    // TRENDING AND DELTA CHECK TESTS (4 tests)
    // ============================================================

    @Test
    @DisplayName("Calculate Percentage Change: Increasing trend")
    void testCalculatePercentageChange_Increasing() {
        // When: Calculate percentage change (1.5 → 2.5 = +66.7%)
        Double percentChange = creatinineResultWithTrend.calculatePercentageChange();

        // Then: Should be approximately +66.7%
        assertNotNull(percentChange);
        assertEquals(66.67, percentChange, 0.1);
    }

    @Test
    @DisplayName("Get Trend: INCREASING when value rises")
    void testGetTrend_Increasing() {
        // When: Get trend direction
        String trend = creatinineResultWithTrend.getTrend();

        // Then: Should be INCREASING
        assertEquals("INCREASING", trend);
    }

    @Test
    @DisplayName("Get Trend: STABLE when change < 5%")
    void testGetTrend_Stable() {
        // Given: Result with minimal change (1.5 → 1.52 = +1.3%)
        TestResult result = TestResult.builder()
            .numericValue(1.52)
            .previousResults(TestResult.PreviousResults.builder()
                .previousValue(1.5)
                .build())
            .build();

        // When: Get trend
        String trend = result.getTrend();

        // Then: Should be STABLE
        assertEquals("STABLE", trend);
    }

    @Test
    @DisplayName("Get Trend: DECREASING when value drops")
    void testGetTrend_Decreasing() {
        // Given: Result with decreasing value (2.0 → 1.2 = -40%)
        TestResult result = TestResult.builder()
            .numericValue(1.2)
            .previousResults(TestResult.PreviousResults.builder()
                .previousValue(2.0)
                .build())
            .build();

        // When: Get trend
        String trend = result.getTrend();

        // Then: Should be DECREASING
        assertEquals("DECREASING", trend);
    }

    // ============================================================
    // CRITICAL VALUE DETECTION TESTS (3 tests)
    // ============================================================

    @Test
    @DisplayName("Requires Immediate Action: Critical value")
    void testRequiresImmediateAction_CriticalValue() {
        // Then: Critical lactate requires immediate action
        assertTrue(criticalLactateResult.requiresImmediateAction());
        assertEquals("CRITICAL", criticalLactateResult.getSeverity());
    }

    @Test
    @DisplayName("Needs Physician Review: Critical result")
    void testNeedsPhysicianReview_CriticalResult() {
        // Then: Critical result needs physician review
        assertTrue(criticalLactateResult.needsPhysicianReview());
        assertTrue(criticalLactateResult.isRequiresPhysicianReview());
    }

    @Test
    @DisplayName("Needs Physician Review: Significant delta change")
    void testNeedsPhysicianReview_SignificantChange() {
        // Given: Result with significant change from previous
        creatinineResultWithTrend.getPreviousResults().setSignificantChange(true);

        // Then: Should need physician review
        assertTrue(creatinineResultWithTrend.needsPhysicianReview());
    }

    // ============================================================
    // QUALITY AND SPECIMEN CHECKS (3 tests)
    // ============================================================

    @Test
    @DisplayName("Is Specimen Acceptable: Good quality specimen")
    void testIsSpecimenAcceptable_GoodQuality() {
        // When: Check specimen quality for normal result
        boolean acceptable = normalLactateResult.isSpecimenAcceptable();

        // Then: Should be acceptable
        assertTrue(acceptable);
    }

    @Test
    @DisplayName("Is Specimen Acceptable: Hemolyzed specimen")
    void testIsSpecimenAcceptable_Hemolyzed() {
        // Given: Result with hemolyzed specimen
        TestResult result = TestResult.builder()
            .testName("Potassium")
            .numericValue(6.5)
            .qualityIndicators(TestResult.QualityIndicators.builder()
                .specimenAcceptable(false)
                .specimenIssues(Arrays.asList("Hemolyzed", "Elevated K may be artifact"))
                .build())
            .build();

        // When: Check specimen quality
        boolean acceptable = result.isSpecimenAcceptable();

        // Then: Should NOT be acceptable
        assertFalse(acceptable);
    }

    @Test
    @DisplayName("Quality Indicators: Interference detected")
    void testQualityIndicators_InterferenceDetected() {
        // Given: Result with quality issues
        TestResult result = TestResult.builder()
            .testName("Test")
            .qualityIndicators(TestResult.QualityIndicators.builder()
                .specimenAcceptable(true)
                .interferenceDetected(true)
                .interferingFactors(Arrays.asList("Lipemia", "Possible falsely elevated"))
                .build())
            .build();

        // Then: Should detect interference
        assertTrue(result.getQualityIndicators().isInterferenceDetected());
        assertFalse(result.getQualityIndicators().getInterferingFactors().isEmpty());
    }

    // ============================================================
    // HELPER METHOD TESTS (5 tests)
    // ============================================================

    @Test
    @DisplayName("Get Formatted Result: Numeric value with unit")
    void testGetFormattedResult_NumericWithUnit() {
        // When: Get formatted result
        String formatted = normalLactateResult.getFormattedResult();

        // Then: Should include value and unit
        assertEquals("1.50 mmol/L", formatted);
    }

    @Test
    @DisplayName("Get Age In Hours: Recent result")
    void testGetAgeInHours_RecentResult() {
        // Given: Result from 2 hours ago
        long twoHoursAgo = System.currentTimeMillis() - (2 * 60 * 60 * 1000);
        TestResult result = TestResult.builder()
            .resultTimestamp(twoHoursAgo)
            .build();

        // When: Get age in hours
        long age = result.getAgeInHours();

        // Then: Should be approximately 2 hours
        assertEquals(2, age);
        assertTrue(result.isRecent()); // < 24 hours
    }

    @Test
    @DisplayName("Is Recent: Result < 24 hours old")
    void testIsRecent_LessThan24Hours() {
        // Given: Result from 12 hours ago
        long twelveHoursAgo = System.currentTimeMillis() - (12 * 60 * 60 * 1000);
        TestResult result = TestResult.builder()
            .resultTimestamp(twelveHoursAgo)
            .build();

        // Then: Should be recent
        assertTrue(result.isRecent());
    }

    @Test
    @DisplayName("Is Recent: Result > 24 hours old")
    void testIsRecent_GreaterThan24Hours() {
        // Given: Result from 36 hours ago
        long thirtySixHoursAgo = System.currentTimeMillis() - (36 * 60 * 60 * 1000);
        TestResult result = TestResult.builder()
            .resultTimestamp(thirtySixHoursAgo)
            .build();

        // Then: Should NOT be recent
        assertFalse(result.isRecent());
    }

    @Test
    @DisplayName("Get Severity: Severity levels based on flags")
    void testGetSeverity_SeverityLevels() {
        // Critical result
        assertEquals("CRITICAL", criticalLactateResult.getSeverity());

        // Abnormal result
        TestResult abnormalResult = TestResult.builder()
            .isAbnormal(true)
            .isCritical(false)
            .build();
        assertEquals("HIGH", abnormalResult.getSeverity());

        // Requires follow-up
        TestResult followUpResult = TestResult.builder()
            .isAbnormal(false)
            .requiresFollowUp(true)
            .build();
        assertEquals("MODERATE", followUpResult.getSeverity());

        // Normal result
        assertEquals("LOW", normalLactateResult.getSeverity());
    }

    // ============================================================
    // HELPER METHODS - TEST DATA CREATION
    // ============================================================

    private TestResult createNormalLactateResult() {
        return TestResult.builder()
            .resultId("RESULT-LACTATE-001")
            .testId("LAB-LACTATE-001")
            .testName("Serum Lactate")
            .patientId("ROHAN-001")
            .encounterId("ENC-001")
            .orderId("ORDER-001")
            .testType(TestResult.TestType.LAB)
            .resultValue("1.5")
            .numericValue(1.5)
            .resultUnit("mmol/L")
            .status(TestResult.ResultStatus.FINAL)
            .collectionTimestamp(System.currentTimeMillis() - (2 * 60 * 60 * 1000)) // 2 hours ago
            .resultTimestamp(System.currentTimeMillis() - (90 * 60 * 1000)) // 90 min ago
            .verificationTimestamp(System.currentTimeMillis() - (60 * 60 * 1000)) // 1 hour ago
            .turnaroundTimeMinutes(30)
            .resultInterpretation(TestResult.ResultInterpretation.NORMAL)
            .referenceRange(TestResult.ReferenceRange.builder()
                .normalLow(0.5)
                .normalHigh(2.0)
                .criticalHigh(4.0)
                .unit("mmol/L")
                .population("adult")
                .sex("ALL")
                .interpretationText("Normal tissue perfusion")
                .build())
            .isAbnormal(false)
            .isCritical(false)
            .requiresFollowUp(false)
            .requiresPhysicianReview(false)
            .flags(new ArrayList<>())
            .clinicalSignificance("Normal lactate indicates adequate tissue perfusion")
            .interpretation("NORMAL - No evidence of lactic acidosis or tissue hypoperfusion")
            .qualityIndicators(TestResult.QualityIndicators.builder()
                .specimenAcceptable(true)
                .specimenIssues(new ArrayList<>())
                .interferenceDetected(false)
                .specimenType("Blood - Venous")
                .collectionMethod("Venipuncture")
                .deltaCheckPassed(true)
                .build())
            .performingLab("CardioFit Clinical Laboratory")
            .performingLocation("Main Hospital")
            .methodology("Enzymatic assay")
            .build();
    }

    private TestResult createCriticalLactateResult() {
        return TestResult.builder()
            .resultId("RESULT-LACTATE-CRITICAL-001")
            .testId("LAB-LACTATE-001")
            .testName("Serum Lactate")
            .patientId("SEPSIS-PATIENT-001")
            .encounterId("ENC-SEPSIS-001")
            .orderId("ORDER-STAT-001")
            .testType(TestResult.TestType.LAB)
            .resultValue("4.5")
            .numericValue(4.5)
            .resultUnit("mmol/L")
            .status(TestResult.ResultStatus.FINAL)
            .collectionTimestamp(System.currentTimeMillis() - (30 * 60 * 1000)) // 30 min ago
            .resultTimestamp(System.currentTimeMillis() - (15 * 60 * 1000)) // 15 min ago
            .verificationTimestamp(System.currentTimeMillis() - (10 * 60 * 1000)) // 10 min ago
            .turnaroundTimeMinutes(15)
            .resultInterpretation(TestResult.ResultInterpretation.CRITICAL_HIGH)
            .referenceRange(TestResult.ReferenceRange.builder()
                .normalLow(0.5)
                .normalHigh(2.0)
                .criticalHigh(4.0)
                .unit("mmol/L")
                .population("adult")
                .sex("ALL")
                .interpretationText("Severe lactic acidosis - septic shock")
                .build())
            .isAbnormal(true)
            .isCritical(true)
            .requiresFollowUp(true)
            .requiresPhysicianReview(true)
            .flags(Arrays.asList("HH", "CRITICAL"))
            .alertMessage("CRITICAL: Lactate ≥4.0 mmol/L - SEPTIC SHOCK - Immediate physician notification required")
            .clinicalSignificance("Lactate ≥4.0 mmol/L indicates septic shock with severe tissue hypoperfusion and high mortality risk")
            .interpretation("CRITICAL HIGH - Septic shock. Immediate aggressive resuscitation required.")
            .possibleCauses(Arrays.asList(
                "Septic shock",
                "Cardiogenic shock",
                "Severe hypovolemia",
                "Mesenteric ischemia"
            ))
            .actionRequired("IMMEDIATE: 1) Aggressive fluid resuscitation 2) Vasopressors 3) Broad-spectrum antibiotics 4) ICU transfer 5) Repeat lactate in 2 hours")
            .differentialDiagnosis(Arrays.asList(
                "Septic shock",
                "Cardiogenic shock",
                "Hypovolemic shock"
            ))
            .qualityIndicators(TestResult.QualityIndicators.builder()
                .specimenAcceptable(true)
                .specimenIssues(new ArrayList<>())
                .interferenceDetected(false)
                .specimenType("Blood - Arterial")
                .collectionMethod("Arterial line")
                .deltaCheckPassed(false) // Significant change from previous
                .build())
            .reflexActions(TestResult.ReflexActions.builder()
                .reflexTriggered(true)
                .reflexTests(Arrays.asList("LAB-LACTATE-001")) // Repeat lactate
                .reflexReason("Lactate ≥4.0 mmol/L triggers automatic repeat in 2 hours")
                .reflexProtocol("SEPSIS-LACTATE-MONITORING")
                .build())
            .performingLab("CardioFit Clinical Laboratory")
            .performingLocation("Emergency Department")
            .methodology("Point-of-care testing")
            .build();
    }

    private TestResult createCreatinineResultWithTrend() {
        return TestResult.builder()
            .resultId("RESULT-CREATININE-002")
            .testId("LAB-CREATININE-001")
            .testName("Serum Creatinine")
            .patientId("PATIENT-AKI-001")
            .encounterId("ENC-002")
            .testType(TestResult.TestType.LAB)
            .numericValue(2.5)
            .resultUnit("mg/dL")
            .status(TestResult.ResultStatus.FINAL)
            .resultTimestamp(System.currentTimeMillis())
            .resultInterpretation(TestResult.ResultInterpretation.HIGH)
            .referenceRange(TestResult.ReferenceRange.builder()
                .normalLow(0.6)
                .normalHigh(1.3)
                .criticalHigh(5.0)
                .unit("mg/dL")
                .population("adult")
                .build())
            .isAbnormal(true)
            .isCritical(false)
            .requiresFollowUp(true)
            .requiresPhysicianReview(false)
            .previousResults(TestResult.PreviousResults.builder()
                .previousValue(1.5)
                .previousTimestamp(System.currentTimeMillis() - (24 * 60 * 60 * 1000))
                .percentageChange(66.67)
                .trend("INCREASING")
                .significantChange(true)
                .history(Arrays.asList(
                    TestResult.PreviousResults.HistoricalValue.builder()
                        .value(1.2)
                        .timestamp(System.currentTimeMillis() - (48 * 60 * 60 * 1000))
                        .interpretation("NORMAL")
                        .build(),
                    TestResult.PreviousResults.HistoricalValue.builder()
                        .value(1.5)
                        .timestamp(System.currentTimeMillis() - (24 * 60 * 60 * 1000))
                        .interpretation("HIGH")
                        .build()
                ))
                .build())
            .clinicalSignificance("Rising creatinine suggests acute kidney injury")
            .interpretation("ABNORMAL - Creatinine increasing from 1.5 to 2.5 mg/dL over 24 hours suggests AKI")
            .possibleCauses(Arrays.asList(
                "Acute kidney injury - prerenal",
                "Acute tubular necrosis",
                "Interstitial nephritis",
                "Obstructive uropathy"
            ))
            .actionRequired("Evaluate for AKI, check urine output, assess volume status, review nephrotoxic medications")
            .build();
    }
}
