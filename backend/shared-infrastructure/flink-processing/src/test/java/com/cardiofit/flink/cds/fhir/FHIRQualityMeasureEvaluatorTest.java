package com.cardiofit.flink.cds.fhir;

import com.cardiofit.flink.clients.GoogleFHIRClient;
import com.cardiofit.flink.cds.population.PatientCohort;
import com.cardiofit.flink.cds.population.QualityMeasure;
import com.cardiofit.flink.cds.fhir.FHIRObservationMapper.ClinicalObservation;
import com.cardiofit.flink.cds.fhir.FHIRObservationMapper.BloodPressureObservation;
import com.cardiofit.flink.cds.fhir.FHIRQualityMeasureEvaluator.MeasureEvaluationResult;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import org.mockito.Mock;
import org.mockito.MockitoAnnotations;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.*;
import java.util.concurrent.CompletableFuture;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.anyInt;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.*;

/**
 * Comprehensive test suite for FHIRQualityMeasureEvaluator.
 *
 * Tests cover:
 * - CDC-HbA1c Testing measure evaluation
 * - CDC-HbA1c Control measure evaluation (<8%)
 * - CDC-BP Control measure evaluation (<140/90)
 * - COL (Colorectal) Screening measure evaluation
 * - BCS (Breast Cancer) Screening measure evaluation
 * - Measure aggregation and compliance rate calculation
 * - Denominator/numerator/exclusion/exception logic
 *
 * Phase 8 Day 9-12: FHIR Integration Layer Tests
 */
class FHIRQualityMeasureEvaluatorTest {

    @Mock
    private GoogleFHIRClient mockFhirClient;

    @Mock
    private FHIRObservationMapper mockObservationMapper;

    @Mock
    private FHIRCohortBuilder mockCohortBuilder;

    private FHIRQualityMeasureEvaluator evaluator;

    @BeforeEach
    void setUp() {
        MockitoAnnotations.openMocks(this);
        evaluator = new FHIRQualityMeasureEvaluator(mockFhirClient, mockObservationMapper, mockCohortBuilder);
    }

    // ==================== Measurement Period Constants Tests ====================

    @Test
    @DisplayName("Should have correct measurement period constants")
    void testMeasurementPeriodConstants() {
        assertEquals(12, FHIRQualityMeasureEvaluator.ANNUAL_LOOKBACK_MONTHS);
        assertEquals(24, FHIRQualityMeasureEvaluator.BIENNIAL_LOOKBACK_MONTHS);
        assertEquals(10, FHIRQualityMeasureEvaluator.COLONOSCOPY_LOOKBACK_YEARS);
    }

    // ==================== CDC-HbA1c Testing Measure Tests ====================

    @Test
    @DisplayName("Should evaluate CDC-HbA1c Testing measure for compliant patients")
    void testEvaluateCDCHbA1cTesting_Compliant() throws Exception {
        // Given: 2 patients, both had HbA1c test in past 12 months
        QualityMeasure measure = createTestMeasure("CDC-HbA1c", "CDC-HbA1c Testing");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002"));

        when(mockObservationMapper.hasRecentHbA1c("P001", FHIRQualityMeasureEvaluator.ANNUAL_LOOKBACK_MONTHS))
            .thenReturn(CompletableFuture.completedFuture(true));
        when(mockObservationMapper.hasRecentHbA1c("P002", FHIRQualityMeasureEvaluator.ANNUAL_LOOKBACK_MONTHS))
            .thenReturn(CompletableFuture.completedFuture(true));

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then
        assertEquals(2, result.getDenominatorCount(), "Both patients in denominator");
        assertEquals(2, result.getNumeratorCount(), "Both patients had recent HbA1c");
        assertEquals(100.0, result.getComplianceRate(), 0.1, "100% compliance");
        assertNotNull(result.getLastCalculated());
    }

    @Test
    @DisplayName("Should evaluate CDC-HbA1c Testing measure for partially compliant cohort")
    void testEvaluateCDCHbA1cTesting_PartialCompliance() throws Exception {
        // Given: 3 patients, only 2 had HbA1c test in past 12 months
        QualityMeasure measure = createTestMeasure("CDC-HbA1c", "CDC-HbA1c Testing");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002", "P003"));

        when(mockObservationMapper.hasRecentHbA1c("P001", FHIRQualityMeasureEvaluator.ANNUAL_LOOKBACK_MONTHS))
            .thenReturn(CompletableFuture.completedFuture(true));
        when(mockObservationMapper.hasRecentHbA1c("P002", FHIRQualityMeasureEvaluator.ANNUAL_LOOKBACK_MONTHS))
            .thenReturn(CompletableFuture.completedFuture(true));
        when(mockObservationMapper.hasRecentHbA1c("P003", FHIRQualityMeasureEvaluator.ANNUAL_LOOKBACK_MONTHS))
            .thenReturn(CompletableFuture.completedFuture(false));

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then
        assertEquals(3, result.getDenominatorCount());
        assertEquals(2, result.getNumeratorCount());
        assertEquals(66.67, result.getComplianceRate(), 0.1, "66.67% compliance");
    }

    @Test
    @DisplayName("Should evaluate CDC-HbA1c Testing measure with no compliant patients")
    void testEvaluateCDCHbA1cTesting_NoCompliance() throws Exception {
        // Given: 2 patients, neither had HbA1c test in past 12 months
        QualityMeasure measure = createTestMeasure("CDC-HbA1c", "CDC-HbA1c Testing");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002"));

        when(mockObservationMapper.hasRecentHbA1c(anyString(), anyInt()))
            .thenReturn(CompletableFuture.completedFuture(false));

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then
        assertEquals(2, result.getDenominatorCount());
        assertEquals(0, result.getNumeratorCount());
        assertEquals(0.0, result.getComplianceRate(), 0.1, "0% compliance");
    }

    // ==================== CDC-HbA1c Control Measure Tests ====================

    @Test
    @DisplayName("Should evaluate CDC-HbA1c Control measure for controlled patients")
    void testEvaluateCDCHbA1cControl_Controlled() throws Exception {
        // Given: 2 patients with HbA1c < 8%
        QualityMeasure measure = createTestMeasure("CDC-HbA1c-Control", "CDC-HbA1c Control <8%");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002"));

        ClinicalObservation hba1c1 = createHbA1cObservation(7.2);
        ClinicalObservation hba1c2 = createHbA1cObservation(7.8);

        when(mockObservationMapper.getMostRecentHbA1c("P001"))
            .thenReturn(CompletableFuture.completedFuture(hba1c1));
        when(mockObservationMapper.getMostRecentHbA1c("P002"))
            .thenReturn(CompletableFuture.completedFuture(hba1c2));

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then
        assertEquals(2, result.getDenominatorCount());
        assertEquals(2, result.getNumeratorCount());
        assertEquals(100.0, result.getComplianceRate(), 0.1);
    }

    @Test
    @DisplayName("Should evaluate CDC-HbA1c Control measure for uncontrolled patients")
    void testEvaluateCDCHbA1cControl_Uncontrolled() throws Exception {
        // Given: 2 patients with HbA1c >= 8%
        QualityMeasure measure = createTestMeasure("CDC-HbA1c-Control", "CDC-HbA1c Control <8%");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002"));

        ClinicalObservation hba1c1 = createHbA1cObservation(8.5);
        ClinicalObservation hba1c2 = createHbA1cObservation(9.2);

        when(mockObservationMapper.getMostRecentHbA1c("P001"))
            .thenReturn(CompletableFuture.completedFuture(hba1c1));
        when(mockObservationMapper.getMostRecentHbA1c("P002"))
            .thenReturn(CompletableFuture.completedFuture(hba1c2));

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then
        assertEquals(2, result.getDenominatorCount());
        assertEquals(0, result.getNumeratorCount(), "No patients with HbA1c < 8%");
        assertEquals(0.0, result.getComplianceRate(), 0.1);
    }

    @Test
    @DisplayName("Should exclude patients with no HbA1c data from denominator")
    void testEvaluateCDCHbA1cControl_ExcludeNoData() throws Exception {
        // Given: 2 patients, one with no HbA1c data
        QualityMeasure measure = createTestMeasure("CDC-HbA1c-Control", "CDC-HbA1c Control <8%");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002"));

        ClinicalObservation hba1c1 = createHbA1cObservation(7.5);

        when(mockObservationMapper.getMostRecentHbA1c("P001"))
            .thenReturn(CompletableFuture.completedFuture(hba1c1));
        when(mockObservationMapper.getMostRecentHbA1c("P002"))
            .thenReturn(CompletableFuture.completedFuture(null)); // No data

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then: Only P001 in denominator
        assertEquals(1, result.getDenominatorCount(), "Only patients with HbA1c data in denominator");
        assertEquals(1, result.getNumeratorCount());
        assertEquals(100.0, result.getComplianceRate(), 0.1);
    }

    // ==================== CDC-BP Control Measure Tests ====================

    @Test
    @DisplayName("Should evaluate CDC-BP Control measure for controlled patients")
    void testEvaluateCDCBPControl_Controlled() throws Exception {
        // Given: 2 patients with BP < 140/90
        QualityMeasure measure = createTestMeasure("CDC-BP-Control", "CDC-BP Control <140/90");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002"));

        BloodPressureObservation bp1 = createBPObservation(130.0, 85.0, true);
        BloodPressureObservation bp2 = createBPObservation(125.0, 80.0, true);

        when(mockObservationMapper.getMostRecentBloodPressure("P001"))
            .thenReturn(CompletableFuture.completedFuture(bp1));
        when(mockObservationMapper.getMostRecentBloodPressure("P002"))
            .thenReturn(CompletableFuture.completedFuture(bp2));

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then
        assertEquals(2, result.getDenominatorCount());
        assertEquals(2, result.getNumeratorCount());
        assertEquals(100.0, result.getComplianceRate(), 0.1);
    }

    @Test
    @DisplayName("Should evaluate CDC-BP Control measure for uncontrolled patients")
    void testEvaluateCDCBPControl_Uncontrolled() throws Exception {
        // Given: 2 patients with BP >= 140/90
        QualityMeasure measure = createTestMeasure("CDC-BP-Control", "CDC-BP Control <140/90");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002"));

        BloodPressureObservation bp1 = createBPObservation(145.0, 92.0, false);
        BloodPressureObservation bp2 = createBPObservation(150.0, 95.0, false);

        when(mockObservationMapper.getMostRecentBloodPressure("P001"))
            .thenReturn(CompletableFuture.completedFuture(bp1));
        when(mockObservationMapper.getMostRecentBloodPressure("P002"))
            .thenReturn(CompletableFuture.completedFuture(bp2));

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then
        assertEquals(2, result.getDenominatorCount());
        assertEquals(0, result.getNumeratorCount());
        assertEquals(0.0, result.getComplianceRate(), 0.1);
    }

    @Test
    @DisplayName("Should exclude patients with no BP data from denominator")
    void testEvaluateCDCBPControl_ExcludeNoData() throws Exception {
        // Given: 1 patient with BP data, 1 without
        QualityMeasure measure = createTestMeasure("CDC-BP-Control", "CDC-BP Control <140/90");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002"));

        BloodPressureObservation bp1 = createBPObservation(130.0, 85.0, true);

        when(mockObservationMapper.getMostRecentBloodPressure("P001"))
            .thenReturn(CompletableFuture.completedFuture(bp1));
        when(mockObservationMapper.getMostRecentBloodPressure("P002"))
            .thenReturn(CompletableFuture.completedFuture(null)); // No data

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then: Only P001 in denominator
        assertEquals(1, result.getDenominatorCount());
        assertEquals(1, result.getNumeratorCount());
        assertEquals(100.0, result.getComplianceRate(), 0.1);
    }

    // ==================== COL Screening Measure Tests ====================

    @Test
    @DisplayName("Should evaluate COL Screening measure for compliant patients")
    void testEvaluateCOLScreening_Compliant() throws Exception {
        // Given: 2 patients with recent colorectal screening
        QualityMeasure measure = createTestMeasure("COL", "Colorectal Cancer Screening");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002"));

        when(mockObservationMapper.hasRecentColorectalScreening("P001", FHIRQualityMeasureEvaluator.ANNUAL_LOOKBACK_MONTHS))
            .thenReturn(CompletableFuture.completedFuture(true));
        when(mockObservationMapper.hasRecentColorectalScreening("P002", FHIRQualityMeasureEvaluator.ANNUAL_LOOKBACK_MONTHS))
            .thenReturn(CompletableFuture.completedFuture(true));

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then
        assertEquals(2, result.getDenominatorCount());
        assertEquals(2, result.getNumeratorCount());
        assertEquals(100.0, result.getComplianceRate(), 0.1);
    }

    @Test
    @DisplayName("Should evaluate COL Screening measure with partial compliance")
    void testEvaluateCOLScreening_PartialCompliance() throws Exception {
        // Given: 3 patients, only 1 with recent screening
        QualityMeasure measure = createTestMeasure("COL", "Colorectal Cancer Screening");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002", "P003"));

        when(mockObservationMapper.hasRecentColorectalScreening("P001", FHIRQualityMeasureEvaluator.ANNUAL_LOOKBACK_MONTHS))
            .thenReturn(CompletableFuture.completedFuture(true));
        when(mockObservationMapper.hasRecentColorectalScreening("P002", FHIRQualityMeasureEvaluator.ANNUAL_LOOKBACK_MONTHS))
            .thenReturn(CompletableFuture.completedFuture(false));
        when(mockObservationMapper.hasRecentColorectalScreening("P003", FHIRQualityMeasureEvaluator.ANNUAL_LOOKBACK_MONTHS))
            .thenReturn(CompletableFuture.completedFuture(false));

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then
        assertEquals(3, result.getDenominatorCount());
        assertEquals(1, result.getNumeratorCount());
        assertEquals(33.33, result.getComplianceRate(), 0.1);
    }

    // ==================== BCS Screening Measure Tests ====================

    @Test
    @DisplayName("Should evaluate BCS Screening measure")
    void testEvaluateBCSScreening() throws Exception {
        // Given: BCS measure (note: not fully implemented, returns placeholder results)
        QualityMeasure measure = createTestMeasure("BCS", "Breast Cancer Screening");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002"));

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then: Should handle gracefully (not fully implemented)
        assertNotNull(result);
        assertEquals(2, result.getDenominatorCount(), "All patients in denominator");
        // Note: Numerator will be 0 until mammography query is implemented
    }

    // ==================== Unknown Measure Tests ====================

    @Test
    @DisplayName("Should handle unknown quality measure code gracefully")
    void testEvaluateMeasure_UnknownCode() throws Exception {
        // Given: Unknown measure code
        QualityMeasure measure = createTestMeasure("UNKNOWN-CODE", "Unknown Measure");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001"));

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then: Should return original measure unchanged
        assertNotNull(result);
        assertEquals("UNKNOWN-CODE", result.getHedisCode());
    }

    // ==================== MeasureEvaluationResult DTO Tests ====================

    @Test
    @DisplayName("MeasureEvaluationResult should store all required fields")
    void testMeasureEvaluationResultDTO() {
        // Given
        MeasureEvaluationResult result = new MeasureEvaluationResult();

        // When
        result.setPatientId("P001");
        result.setInDenominator(true);
        result.setInNumerator(true);
        result.setExcluded(false);
        result.setException(false);
        result.setComplianceReason("HbA1c test performed in past 12 months");

        // Then
        assertEquals("P001", result.getPatientId());
        assertTrue(result.isInDenominator());
        assertTrue(result.isInNumerator());
        assertFalse(result.isExcluded());
        assertFalse(result.isException());
        assertEquals("HbA1c test performed in past 12 months", result.getComplianceReason());
    }

    @Test
    @DisplayName("MeasureEvaluationResult toString should format correctly")
    void testMeasureEvaluationResultToString() {
        // Given
        MeasureEvaluationResult result = new MeasureEvaluationResult();
        result.setPatientId("P001");
        result.setInDenominator(true);
        result.setInNumerator(false);
        result.setExcluded(false);

        // When
        String toString = result.toString();

        // Then
        assertTrue(toString.contains("P001"));
        assertTrue(toString.contains("denominator=true"));
        assertTrue(toString.contains("numerator=false"));
    }

    // ==================== Edge Case Tests ====================

    @Test
    @DisplayName("Should handle empty cohort gracefully")
    void testEvaluateMeasure_EmptyCohort() throws Exception {
        // Given: Empty cohort
        QualityMeasure measure = createTestMeasure("CDC-HbA1c", "CDC-HbA1c Testing");
        PatientCohort cohort = createTestCohort(new ArrayList<>());

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then
        assertEquals(0, result.getDenominatorCount());
        assertEquals(0, result.getNumeratorCount());
        assertEquals(0.0, result.getComplianceRate(), 0.1);
    }

    @Test
    @DisplayName("Should calculate compliance rate as 0% when denominator is 0")
    void testComplianceRateCalculation_ZeroDenominator() throws Exception {
        // Given: Cohort where all patients excluded
        QualityMeasure measure = createTestMeasure("CDC-HbA1c-Control", "CDC-HbA1c Control");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002"));

        // All patients have no HbA1c data
        when(mockObservationMapper.getMostRecentHbA1c(anyString()))
            .thenReturn(CompletableFuture.completedFuture(null));

        // When
        QualityMeasure result = evaluator.evaluateMeasure(measure, cohort).get();

        // Then
        assertEquals(0, result.getDenominatorCount(), "All patients excluded");
        assertEquals(0, result.getNumeratorCount());
        assertEquals(0.0, result.getComplianceRate(), "Compliance rate should be 0% when denominator is 0");
    }

    // ==================== Helper Methods ====================

    private QualityMeasure createTestMeasure(String code, String name) {
        QualityMeasure measure = new QualityMeasure();
        measure.setHedisCode(code);
        measure.setMeasureName(name);
        measure.setMeasureType(QualityMeasure.MeasureType.PROCESS);
        measure.setDescription("Test quality measure");
        return measure;
    }

    private PatientCohort createTestCohort(List<String> patientIds) {
        PatientCohort cohort = new PatientCohort();
        cohort.setCohortName("Test Cohort");
        cohort.setDescription("Test cohort");
        cohort.setCohortType(PatientCohort.CohortType.CUSTOM);
        cohort.getPatientIds().addAll(patientIds);
        cohort.setTotalPatients(patientIds.size());
        cohort.setLastUpdated(LocalDateTime.now());
        cohort.setActive(true);
        return cohort;
    }

    private ClinicalObservation createHbA1cObservation(double value) {
        ClinicalObservation obs = new ClinicalObservation(
            FHIRObservationMapper.LOINC_HBA1C,
            "HbA1c",
            value,
            "%",
            LocalDate.now()
        );
        return obs;
    }

    private BloodPressureObservation createBPObservation(double systolic, double diastolic, boolean controlled) {
        BloodPressureObservation bp = new BloodPressureObservation(systolic, diastolic, LocalDate.now());
        bp.setControlled(controlled);
        return bp;
    }
}
