package com.cardiofit.flink.cds.fhir;

import com.cardiofit.flink.clients.GoogleFHIRClient;
import com.cardiofit.flink.cds.population.PatientCohort;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import org.mockito.Mock;
import org.mockito.MockitoAnnotations;

import java.time.LocalDate;
import java.util.*;
import java.util.concurrent.CompletableFuture;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

/**
 * Comprehensive test suite for FHIRCohortBuilder.
 *
 * Tests cover:
 * - Condition-based cohort building (diabetes, hypertension, CKD)
 * - Age-based cohort building (geriatric, pediatric, age ranges)
 * - Medication-based cohort building
 * - Geographic cohort building
 * - Composite cohort building (intersection logic)
 * - Risk-stratified cohort building
 * - HEDIS measure denominator cohorts
 *
 * Phase 8 Day 9-12: FHIR Integration Layer Tests
 */
class FHIRCohortBuilderTest {

    @Mock
    private GoogleFHIRClient mockFhirClient;

    @Mock
    private FHIRPopulationHealthMapper mockPopulationMapper;

    private FHIRCohortBuilder builder;

    @BeforeEach
    void setUp() {
        MockitoAnnotations.openMocks(this);
        builder = new FHIRCohortBuilder(mockFhirClient, mockPopulationMapper);
    }

    // ==================== ICD-10 Code Constants Tests ====================

    @Test
    @DisplayName("Should have correct ICD-10 condition code constants")
    void testICD10CodeConstants() {
        assertEquals("E11", FHIRCohortBuilder.ICD10_DIABETES_PREFIX);
        assertEquals("I10", FHIRCohortBuilder.ICD10_HYPERTENSION_PREFIX);
        assertEquals("I50", FHIRCohortBuilder.ICD10_HEART_FAILURE_PREFIX);
        assertEquals("N18", FHIRCohortBuilder.ICD10_CKD_PREFIX);
        assertEquals("J44", FHIRCohortBuilder.ICD10_COPD_PREFIX);
        assertEquals("J45", FHIRCohortBuilder.ICD10_ASTHMA_PREFIX);
    }

    @Test
    @DisplayName("Should have correct age threshold constants")
    void testAgeThresholdConstants() {
        assertEquals(18, FHIRCohortBuilder.AGE_PEDIATRIC);
        assertEquals(65, FHIRCohortBuilder.AGE_ADULT);
        assertEquals(75, FHIRCohortBuilder.AGE_GERIATRIC);
    }

    // ==================== Condition Cohort Building Tests ====================

    @Test
    @DisplayName("Should build diabetic cohort with correct criteria")
    void testBuildDiabeticCohort() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildDiabeticCohort();
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("Diabetic Patients", cohort.getCohortName());
        assertEquals(PatientCohort.CohortType.DISEASE_BASED, cohort.getCohortType());

        // Check inclusion criteria (now List<CriteriaRule>)
        List<PatientCohort.CriteriaRule> rules = cohort.getInclusionCriteria();
        assertEquals(1, rules.size());
        assertEquals(PatientCohort.CriteriaRule.CriteriaType.DIAGNOSIS, rules.get(0).getCriteriaType());
        assertEquals(FHIRCohortBuilder.ICD10_DIABETES_PREFIX, rules.get(0).getValue());

        assertTrue(cohort.getDescription().contains("Type 2 Diabetes"));
    }

    @Test
    @DisplayName("Should build hypertensive cohort with correct criteria")
    void testBuildHypertensiveCohort() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildHypertensiveCohort();
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("Hypertensive Patients", cohort.getCohortName());
        assertEquals(PatientCohort.CohortType.DISEASE_BASED, cohort.getCohortType());

        // Check inclusion criteria
        List<PatientCohort.CriteriaRule> rules = cohort.getInclusionCriteria();
        assertEquals(1, rules.size());
        assertEquals(FHIRCohortBuilder.ICD10_HYPERTENSION_PREFIX, rules.get(0).getValue());
    }

    @Test
    @DisplayName("Should build CKD cohort with correct criteria")
    void testBuildCKDCohort() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildCKDCohort();
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("CKD Patients", cohort.getCohortName());
        assertEquals(PatientCohort.CohortType.DISEASE_BASED, cohort.getCohortType());

        // Check inclusion criteria
        List<PatientCohort.CriteriaRule> rules = cohort.getInclusionCriteria();
        assertEquals(1, rules.size());
        assertEquals(FHIRCohortBuilder.ICD10_CKD_PREFIX, rules.get(0).getValue());
    }

    @Test
    @DisplayName("Should build custom condition cohort")
    void testBuildConditionCohort_Custom() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildConditionCohort(
            "J44",
            "COPD Patients",
            "Patients with COPD"
        );
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("COPD Patients", cohort.getCohortName());

        // Check inclusion criteria
        List<PatientCohort.CriteriaRule> rules = cohort.getInclusionCriteria();
        assertEquals(1, rules.size());
        assertEquals("J44", rules.get(0).getValue());
    }

    // ==================== Age Cohort Building Tests ====================

    @Test
    @DisplayName("Should build geriatric cohort (age >= 65)")
    void testBuildGeriatricCohort() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildGeriatricCohort();
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("Geriatric Patients", cohort.getCohortName());
        assertEquals(PatientCohort.CohortType.DEMOGRAPHIC, cohort.getCohortType());

        // Check inclusion criteria - should have single age rule with min age
        List<PatientCohort.CriteriaRule> rules = cohort.getInclusionCriteria();
        assertEquals(1, rules.size());
        assertEquals(PatientCohort.CriteriaRule.CriteriaType.AGE, rules.get(0).getCriteriaType());
        assertEquals("65", rules.get(0).getValue());
        assertEquals(">=", rules.get(0).getOperator());
        assertNull(rules.get(0).getSecondValue(), "Geriatric cohort should have no upper age limit");
    }

    @Test
    @DisplayName("Should build age cohort with min and max bounds")
    void testBuildAgeCohort_WithBounds() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildAgeCohort(
            50, 75,
            "Middle-Aged Adults",
            "Adults aged 50-75"
        );
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("Middle-Aged Adults", cohort.getCohortName());

        // Check inclusion criteria - should have age rule with BETWEEN operator
        List<PatientCohort.CriteriaRule> rules = cohort.getInclusionCriteria();
        assertEquals(1, rules.size());
        assertEquals(PatientCohort.CriteriaRule.CriteriaType.AGE, rules.get(0).getCriteriaType());
        assertEquals("50", rules.get(0).getValue());
        assertEquals("75", rules.get(0).getSecondValue());
        assertEquals("BETWEEN", rules.get(0).getOperator());
    }

    @Test
    @DisplayName("Should build age cohort with only min bound")
    void testBuildAgeCohort_MinOnly() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildAgeCohort(
            18, null,
            "Adults",
            "Patients aged 18 and older"
        );
        PatientCohort cohort = future.get();

        // Then - should have age rule with >= operator
        List<PatientCohort.CriteriaRule> rules = cohort.getInclusionCriteria();
        assertEquals(1, rules.size());
        assertEquals("18", rules.get(0).getValue());
        assertEquals(">=", rules.get(0).getOperator());
        assertNull(rules.get(0).getSecondValue());
    }

    // ==================== Medication Cohort Building Tests ====================

    @Test
    @DisplayName("Should build medication cohort with correct criteria")
    void testBuildMedicationCohort() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildMedicationCohort(
            "statin",
            "Statin Users",
            "Patients on statin medications"
        );
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("Statin Users", cohort.getCohortName());
        assertEquals(PatientCohort.CohortType.CUSTOM, cohort.getCohortType());

        // Check inclusion criteria
        List<PatientCohort.CriteriaRule> rules = cohort.getInclusionCriteria();
        assertEquals(1, rules.size());
        assertEquals(PatientCohort.CriteriaRule.CriteriaType.MEDICATION, rules.get(0).getCriteriaType());
        assertEquals("statin", rules.get(0).getValue());
    }

    // ==================== Geographic Cohort Building Tests ====================

    @Test
    @DisplayName("Should build geographic cohort with zip code criteria")
    void testBuildGeographicCohort() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildGeographicCohort(
            "94103",
            "San Francisco 94103",
            "Patients in San Francisco zip code 94103"
        );
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("San Francisco 94103", cohort.getCohortName());
        assertEquals(PatientCohort.CohortType.GEOGRAPHIC, cohort.getCohortType());

        // Check inclusion criteria
        List<PatientCohort.CriteriaRule> rules = cohort.getInclusionCriteria();
        assertEquals(1, rules.size());
        assertEquals(PatientCohort.CriteriaRule.CriteriaType.GEOGRAPHIC, rules.get(0).getCriteriaType());
        assertEquals("94103", rules.get(0).getValue());
    }

    // ==================== Composite Cohort Building Tests ====================

    @Test
    @DisplayName("Should build composite cohort with intersection logic")
    void testBuildCompositeCohort_Intersection() throws Exception {
        // Given: Two cohorts with overlapping patients
        CompletableFuture<PatientCohort> cohort1Future = CompletableFuture.completedFuture(
            createTestCohort("Cohort 1", Arrays.asList("P001", "P002", "P003"))
        );
        CompletableFuture<PatientCohort> cohort2Future = CompletableFuture.completedFuture(
            createTestCohort("Cohort 2", Arrays.asList("P002", "P003", "P004"))
        );

        // When
        CompletableFuture<PatientCohort> compositeFuture = builder.buildCompositeCohort(
            "Intersection Cohort",
            "Patients in both cohorts",
            Arrays.asList(cohort1Future, cohort2Future)
        );
        PatientCohort composite = compositeFuture.get();

        // Then: Should only have P002 and P003 (intersection)
        assertEquals(2, composite.getTotalPatients());
        assertTrue(composite.getPatientIds().contains("P002"));
        assertTrue(composite.getPatientIds().contains("P003"));
        assertFalse(composite.getPatientIds().contains("P001"));
        assertFalse(composite.getPatientIds().contains("P004"));
        assertEquals(PatientCohort.CohortType.CUSTOM, composite.getCohortType());
    }

    @Test
    @DisplayName("Should handle empty composite cohort when no intersection")
    void testBuildCompositeCohort_NoIntersection() throws Exception {
        // Given: Two cohorts with no overlap
        CompletableFuture<PatientCohort> cohort1Future = CompletableFuture.completedFuture(
            createTestCohort("Cohort 1", Arrays.asList("P001", "P002"))
        );
        CompletableFuture<PatientCohort> cohort2Future = CompletableFuture.completedFuture(
            createTestCohort("Cohort 2", Arrays.asList("P003", "P004"))
        );

        // When
        CompletableFuture<PatientCohort> compositeFuture = builder.buildCompositeCohort(
            "Empty Intersection",
            "Should be empty",
            Arrays.asList(cohort1Future, cohort2Future)
        );
        PatientCohort composite = compositeFuture.get();

        // Then
        assertEquals(0, composite.getTotalPatients());
        assertTrue(composite.getPatientIds().isEmpty());
    }

    @Test
    @DisplayName("Should merge inclusion criteria from all cohorts")
    void testBuildCompositeCohort_MergeCriteria() throws Exception {
        // Given: Cohorts with different criteria
        PatientCohort cohort1 = createTestCohort("Age Cohort", Arrays.asList("P001", "P002"));
        PatientCohort.CriteriaRule ageRule = new PatientCohort.CriteriaRule(
            PatientCohort.CriteriaRule.CriteriaType.AGE, "age_years", ">=", "50"
        );
        cohort1.addInclusionCriteria(ageRule);

        PatientCohort cohort2 = createTestCohort("Condition Cohort", Arrays.asList("P001", "P002"));
        PatientCohort.CriteriaRule conditionRule = new PatientCohort.CriteriaRule(
            PatientCohort.CriteriaRule.CriteriaType.DIAGNOSIS, "ICD-10", "STARTS_WITH", "E11"
        );
        cohort2.addInclusionCriteria(conditionRule);

        // When
        CompletableFuture<PatientCohort> compositeFuture = builder.buildCompositeCohort(
            "Merged Criteria",
            "Should merge criteria",
            Arrays.asList(
                CompletableFuture.completedFuture(cohort1),
                CompletableFuture.completedFuture(cohort2)
            )
        );
        PatientCohort composite = compositeFuture.get();

        // Then - should have both criteria rules merged
        List<PatientCohort.CriteriaRule> rules = composite.getInclusionCriteria();
        assertEquals(2, rules.size());
        assertTrue(rules.stream().anyMatch(r -> r.getCriteriaType() == PatientCohort.CriteriaRule.CriteriaType.AGE));
        assertTrue(rules.stream().anyMatch(r -> r.getCriteriaType() == PatientCohort.CriteriaRule.CriteriaType.DIAGNOSIS));
    }

    // ==================== Risk-Stratified Cohort Tests ====================

    @Test
    @DisplayName("Should build high-risk cardiovascular cohort structure")
    void testBuildHighRiskCardiovascularCohort() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildHighRiskCardiovascularCohort();
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("High-Risk Cardiovascular", cohort.getCohortName());
        assertEquals(PatientCohort.CohortType.RISK_BASED, cohort.getCohortType());

        // Check inclusion criteria - should have multiple rules for age and conditions
        List<PatientCohort.CriteriaRule> rules = cohort.getInclusionCriteria();
        assertTrue(rules.size() >= 2, "Should have at least age and condition criteria");
        assertTrue(rules.stream().anyMatch(r -> r.getCriteriaType() == PatientCohort.CriteriaRule.CriteriaType.AGE),
            "Should have age criteria");
        assertTrue(rules.stream().anyMatch(r -> r.getCriteriaType() == PatientCohort.CriteriaRule.CriteriaType.DIAGNOSIS),
            "Should have diagnosis criteria for diabetes or hypertension");
    }

    // ==================== Custom Cohort Tests ====================

    @Test
    @DisplayName("Should build custom cohort from patient ID list")
    void testBuildCustomCohort() throws Exception {
        // Given
        List<String> patientIds = Arrays.asList("P001", "P002", "P003");

        // Mock enrichment
        when(mockPopulationMapper.enrichCohortFromFHIR(any(PatientCohort.class)))
            .thenAnswer(invocation -> {
                PatientCohort cohort = invocation.getArgument(0);
                return CompletableFuture.completedFuture(cohort);
            });

        // When
        CompletableFuture<PatientCohort> future = builder.buildCustomCohort(
            patientIds,
            "Custom Test Cohort",
            "Test cohort from custom patient list"
        );
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("Custom Test Cohort", cohort.getCohortName());
        assertEquals(PatientCohort.CohortType.CUSTOM, cohort.getCohortType());
        assertEquals(3, cohort.getTotalPatients());
        assertTrue(cohort.getPatientIds().containsAll(patientIds));

        // Verify enrichment was called
        verify(mockPopulationMapper, times(1)).enrichCohortFromFHIR(any(PatientCohort.class));
    }

    // ==================== HEDIS Measure Denominator Tests ====================

    @Test
    @DisplayName("Should build CDC-HbA1c measure denominator cohort")
    void testBuildHEDISMeasureDenominator_CDCHbA1c() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildHEDISMeasureDenominator("CDC-HbA1c");
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("CDC-HbA1c Denominator", cohort.getCohortName());
        assertTrue(cohort.getDescription().contains("diabetic patients aged 18-75"));
    }

    @Test
    @DisplayName("Should build COL measure denominator cohort")
    void testBuildHEDISMeasureDenominator_COL() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildHEDISMeasureDenominator("COL");
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("COL Denominator", cohort.getCohortName());
        assertTrue(cohort.getDescription().contains("aged 50-75"));
    }

    @Test
    @DisplayName("Should build BCS measure denominator cohort with gender filter")
    void testBuildHEDISMeasureDenominator_BCS() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildHEDISMeasureDenominator("BCS");
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("BCS Denominator", cohort.getCohortName());

        // Check inclusion criteria - should have gender rule
        List<PatientCohort.CriteriaRule> rules = cohort.getInclusionCriteria();
        assertTrue(rules.stream().anyMatch(r ->
            r.getCriteriaType() == PatientCohort.CriteriaRule.CriteriaType.GENDER &&
            "female".equals(r.getValue())),
            "Should have female gender criteria");

        assertTrue(cohort.getDescription().contains("women aged 50-74"));
    }

    @Test
    @DisplayName("Should build SAA measure denominator cohort")
    void testBuildHEDISMeasureDenominator_SAA() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildHEDISMeasureDenominator("SAA");
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("SAA Denominator", cohort.getCohortName());
        assertTrue(cohort.getDescription().contains("antipsychotic"));
    }

    @Test
    @DisplayName("Should handle unknown HEDIS measure code")
    void testBuildHEDISMeasureDenominator_Unknown() throws Exception {
        // When
        CompletableFuture<PatientCohort> future = builder.buildHEDISMeasureDenominator("UNKNOWN-CODE");
        PatientCohort cohort = future.get();

        // Then
        assertNotNull(cohort);
        assertEquals("Unknown Measure", cohort.getCohortName());
        assertEquals(0, cohort.getTotalPatients());
    }

    // ==================== Utility Method Tests ====================

    @Test
    @DisplayName("Should generate cohort summary with all details")
    void testGetCohortSummary() {
        // Given
        PatientCohort cohort = createTestCohort("Test Cohort", Arrays.asList("P001", "P002"));
        cohort.setDescription("Test cohort description");

        // Add inclusion criteria as CriteriaRule objects
        PatientCohort.CriteriaRule ageRule = new PatientCohort.CriteriaRule(
            PatientCohort.CriteriaRule.CriteriaType.AGE, "age_years", ">=", "50"
        );
        cohort.addInclusionCriteria(ageRule);

        PatientCohort.CriteriaRule conditionRule = new PatientCohort.CriteriaRule(
            PatientCohort.CriteriaRule.CriteriaType.DIAGNOSIS, "ICD-10", "STARTS_WITH", "E11"
        );
        cohort.addInclusionCriteria(conditionRule);

        // When
        String summary = builder.getCohortSummary(cohort);

        // Then
        assertTrue(summary.contains("Cohort: Test Cohort"));
        assertTrue(summary.contains("Type: CUSTOM"));
        assertTrue(summary.contains("Patients: 2"));
        assertTrue(summary.contains("AGE") || summary.contains("age"));
        assertTrue(summary.contains("DIAGNOSIS") || summary.contains("E11"));
        assertTrue(summary.contains("Test cohort description"));
    }

    // ==================== Helper Methods ====================

    private PatientCohort createTestCohort(String name, List<String> patientIds) {
        PatientCohort cohort = new PatientCohort();
        cohort.setCohortName(name);
        cohort.setDescription("Test cohort");
        cohort.setCohortType(PatientCohort.CohortType.CUSTOM);
        cohort.getPatientIds().addAll(patientIds);
        cohort.setTotalPatients(patientIds.size());
        cohort.setLastUpdated(java.time.LocalDateTime.now());
        cohort.setActive(true);
        return cohort;
    }
}
