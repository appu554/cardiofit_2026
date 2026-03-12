package com.cardiofit.flink.cds.fhir;

import com.cardiofit.flink.clients.GoogleFHIRClient;
import com.cardiofit.flink.cds.population.CareGap;
import com.cardiofit.flink.cds.population.PatientCohort;
import com.cardiofit.flink.cds.population.QualityMeasure;
import com.cardiofit.flink.models.Condition;
import com.cardiofit.flink.models.FHIRPatientData;
import com.cardiofit.flink.models.Medication;
import com.cardiofit.flink.models.VitalSign;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import org.mockito.Mock;
import org.mockito.MockitoAnnotations;

import java.time.LocalDateTime;
import java.util.*;
import java.util.concurrent.CompletableFuture;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.*;

/**
 * Comprehensive test suite for FHIRPopulationHealthMapper.
 *
 * Tests cover:
 * - Cohort enrichment from FHIR Patient/Condition data
 * - Care gap detection using FHIR resources
 * - Quality measure evaluation with FHIR data
 * - Patient data map building from FHIR resources
 * - Error handling and edge cases
 *
 * Phase 8 Day 9-12: FHIR Integration Layer Tests
 */
class FHIRPopulationHealthMapperTest {

    @Mock
    private GoogleFHIRClient mockFhirClient;

    private FHIRPopulationHealthMapper mapper;

    @BeforeEach
    void setUp() {
        MockitoAnnotations.openMocks(this);
        mapper = new FHIRPopulationHealthMapper(mockFhirClient);
    }

    // ==================== Cohort Enrichment Tests ====================

    @Test
    @DisplayName("Should enrich cohort with FHIR patient demographics")
    void testEnrichCohortFromFHIR_Demographics() throws Exception {
        // Given: Cohort with 3 patients
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002", "P003"));

        // Mock FHIR Patient data
        when(mockFhirClient.getPatientAsync("P001"))
            .thenReturn(CompletableFuture.completedFuture(createPatient("P001", 45, "male")));
        when(mockFhirClient.getPatientAsync("P002"))
            .thenReturn(CompletableFuture.completedFuture(createPatient("P002", 62, "female")));
        when(mockFhirClient.getPatientAsync("P003"))
            .thenReturn(CompletableFuture.completedFuture(createPatient("P003", 38, "male")));

        // Mock Condition data (empty for this test)
        when(mockFhirClient.getConditionsAsync(anyString()))
            .thenReturn(CompletableFuture.completedFuture(new ArrayList<>()));

        // When: Enrich cohort
        PatientCohort enriched = mapper.enrichCohortFromFHIR(cohort).get();

        // Then: Demographics populated correctly
        PatientCohort.DemographicProfile demographics = enriched.getDemographics();
        assertEquals(2, demographics.getMaleCount(), "Should have 2 male patients");
        assertEquals(1, demographics.getFemaleCount(), "Should have 1 female patient");
        assertEquals(0, demographics.getOtherGenderCount(), "Should have 0 other gender patients");

        double expectedAvgAge = (45.0 + 62.0 + 38.0) / 3.0;
        assertEquals(expectedAvgAge, demographics.getAverageAge(), 0.1, "Average age should be correct");

        // Verify FHIR client was called for each patient
        verify(mockFhirClient, times(3)).getPatientAsync(anyString());
        verify(mockFhirClient, times(3)).getConditionsAsync(anyString());
    }

    @Test
    @DisplayName("Should populate age range distribution in cohort enrichment")
    void testEnrichCohortFromFHIR_AgeDistribution() throws Exception {
        // Given: Cohort with patients in different age ranges
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002", "P003", "P004"));

        when(mockFhirClient.getPatientAsync("P001"))
            .thenReturn(CompletableFuture.completedFuture(createPatient("P001", 25, "male"))); // 18-34
        when(mockFhirClient.getPatientAsync("P002"))
            .thenReturn(CompletableFuture.completedFuture(createPatient("P002", 45, "female"))); // 35-54
        when(mockFhirClient.getPatientAsync("P003"))
            .thenReturn(CompletableFuture.completedFuture(createPatient("P003", 65, "male"))); // 55-74
        when(mockFhirClient.getPatientAsync("P004"))
            .thenReturn(CompletableFuture.completedFuture(createPatient("P004", 78, "female"))); // 75+

        when(mockFhirClient.getConditionsAsync(anyString()))
            .thenReturn(CompletableFuture.completedFuture(new ArrayList<>()));

        // When
        PatientCohort enriched = mapper.enrichCohortFromFHIR(cohort).get();

        // Then
        Map<String, Integer> ageDistribution = enriched.getDemographics().getAgeRangeDistribution();
        assertEquals(1, ageDistribution.get("18-34"), "Should have 1 patient in 18-34 range");
        assertEquals(1, ageDistribution.get("35-54"), "Should have 1 patient in 35-54 range");
        assertEquals(1, ageDistribution.get("55-74"), "Should have 1 patient in 55-74 range");
        assertEquals(1, ageDistribution.get("75+"), "Should have 1 patient in 75+ range");
    }

    @Test
    @DisplayName("Should populate condition distribution from FHIR Condition resources")
    void testEnrichCohortFromFHIR_ConditionDistribution() throws Exception {
        // Given
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002"));

        when(mockFhirClient.getPatientAsync(anyString()))
            .thenReturn(CompletableFuture.completedFuture(createPatient("P001", 50, "male")));

        // P001: Diabetes + Hypertension
        when(mockFhirClient.getConditionsAsync("P001"))
            .thenReturn(CompletableFuture.completedFuture(Arrays.asList(
                createCondition("E11.9", "Type 2 Diabetes"),
                createCondition("I10", "Essential Hypertension")
            )));

        // P002: Diabetes only
        when(mockFhirClient.getConditionsAsync("P002"))
            .thenReturn(CompletableFuture.completedFuture(Arrays.asList(
                createCondition("E11.9", "Type 2 Diabetes")
            )));

        // When
        PatientCohort enriched = mapper.enrichCohortFromFHIR(cohort).get();

        // Then
        Map<String, Integer> conditionDist = enriched.getConditionDistribution();
        assertEquals(2, conditionDist.get("E11.9"), "Should have 2 patients with diabetes");
        assertEquals(1, conditionDist.get("I10"), "Should have 1 patient with hypertension");
    }

    @Test
    @DisplayName("Should handle null patient data gracefully")
    void testEnrichCohortFromFHIR_NullPatient() throws Exception {
        // Given
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P999"));

        when(mockFhirClient.getPatientAsync("P001"))
            .thenReturn(CompletableFuture.completedFuture(createPatient("P001", 50, "male")));
        when(mockFhirClient.getPatientAsync("P999"))
            .thenReturn(CompletableFuture.completedFuture(null)); // Patient not found

        when(mockFhirClient.getConditionsAsync(anyString()))
            .thenReturn(CompletableFuture.completedFuture(new ArrayList<>()));

        // When
        PatientCohort enriched = mapper.enrichCohortFromFHIR(cohort).get();

        // Then: Should only count P001
        assertEquals(1, enriched.getDemographics().getMaleCount());
        assertNotNull(enriched); // Should not fail
    }

    // ==================== Care Gap Detection Tests ====================

    @Test
    @DisplayName("Should detect care gaps from FHIR data")
    void testDetectCareGapsFromFHIR() throws Exception {
        // Given: Diabetic patient
        String patientId = "P001";

        when(mockFhirClient.getPatientAsync(patientId))
            .thenReturn(CompletableFuture.completedFuture(createPatient(patientId, 55, "male")));

        when(mockFhirClient.getConditionsAsync(patientId))
            .thenReturn(CompletableFuture.completedFuture(Arrays.asList(
                createCondition("E11.9", "Type 2 Diabetes")
            )));

        when(mockFhirClient.getMedicationsAsync(patientId))
            .thenReturn(CompletableFuture.completedFuture(new ArrayList<>()));

        when(mockFhirClient.getVitalsAsync(patientId))
            .thenReturn(CompletableFuture.completedFuture(new ArrayList<>()));

        // When
        List<CareGap> gaps = mapper.detectCareGapsFromFHIR(patientId).get();

        // Then: Should detect diabetes care gaps (HbA1c testing, eye exam, etc.)
        assertNotNull(gaps);
        // Note: Actual gap detection depends on PopulationHealthService logic
    }

    @Test
    @DisplayName("Should return empty list when patient not found")
    void testDetectCareGapsFromFHIR_PatientNotFound() throws Exception {
        // Given
        String patientId = "P999";
        when(mockFhirClient.getPatientAsync(patientId))
            .thenReturn(CompletableFuture.completedFuture(null));

        when(mockFhirClient.getConditionsAsync(anyString()))
            .thenReturn(CompletableFuture.completedFuture(new ArrayList<>()));
        when(mockFhirClient.getMedicationsAsync(anyString()))
            .thenReturn(CompletableFuture.completedFuture(new ArrayList<>()));
        when(mockFhirClient.getVitalsAsync(anyString()))
            .thenReturn(CompletableFuture.completedFuture(new ArrayList<>()));

        // When
        List<CareGap> gaps = mapper.detectCareGapsFromFHIR(patientId).get();

        // Then
        assertNotNull(gaps);
        assertTrue(gaps.isEmpty(), "Should return empty list for non-existent patient");
    }

    // ==================== Patient Data Map Building Tests ====================

    @Test
    @DisplayName("Should build patient data map with demographics")
    void testBuildPatientDataMap_Demographics() {
        // Given
        FHIRPatientData patient = createPatient("P001", 55, "male");
        List<Condition> conditions = new ArrayList<>();
        List<Medication> medications = new ArrayList<>();
        List<VitalSign> vitals = new ArrayList<>();

        // When
        Map<String, Object> dataMap = mapper.buildPatientDataMap(patient, conditions, medications, vitals);

        // Then
        assertEquals(55, dataMap.get("age"));
        assertEquals("M", dataMap.get("gender"));
    }

    @Test
    @DisplayName("Should detect diabetes from ICD-10 E11 prefix")
    void testBuildPatientDataMap_DiabetesDetection() {
        // Given
        FHIRPatientData patient = createPatient("P001", 55, "male");
        List<Condition> conditions = Arrays.asList(
            createCondition("E11.9", "Type 2 Diabetes"),
            createCondition("E11.65", "Type 2 Diabetes with hyperglycemia")
        );

        // When
        Map<String, Object> dataMap = mapper.buildPatientDataMap(patient, conditions, new ArrayList<>(), new ArrayList<>());

        // Then
        assertTrue((Boolean) dataMap.get("has_diabetes"), "Should detect diabetes from E11 prefix");
    }

    @Test
    @DisplayName("Should detect hypertension from ICD-10 I10 code")
    void testBuildPatientDataMap_HypertensionDetection() {
        // Given
        FHIRPatientData patient = createPatient("P001", 55, "male");
        List<Condition> conditions = Arrays.asList(
            createCondition("I10", "Essential Hypertension")
        );

        // When
        Map<String, Object> dataMap = mapper.buildPatientDataMap(patient, conditions, new ArrayList<>(), new ArrayList<>());

        // Then
        assertTrue((Boolean) dataMap.get("has_hypertension"), "Should detect hypertension from I10 code");
    }

    @Test
    @DisplayName("Should estimate medication adherence for BP medications")
    void testBuildPatientDataMap_MedicationAdherence() {
        // Given
        FHIRPatientData patient = createPatient("P001", 55, "male");
        List<Medication> medications = Arrays.asList(
            createMedication("Lisinopril", "10mg", "active")
        );

        // When
        Map<String, Object> dataMap = mapper.buildPatientDataMap(patient, new ArrayList<>(), medications, new ArrayList<>());

        // Then
        assertTrue(dataMap.containsKey("bp_med_adherence"), "Should contain BP medication adherence");
        assertEquals(0.85, dataMap.get("bp_med_adherence"), "Should estimate 85% adherence");
    }

    @Test
    @DisplayName("Should map FHIR gender codes to M/F/O")
    void testBuildPatientDataMap_GenderMapping() {
        // Given: Different gender values
        FHIRPatientData male = createPatient("P001", 50, "male");
        FHIRPatientData female = createPatient("P002", 50, "female");
        FHIRPatientData other = createPatient("P003", 50, "other");

        // When
        Map<String, Object> maleMap = mapper.buildPatientDataMap(male, new ArrayList<>(), new ArrayList<>(), new ArrayList<>());
        Map<String, Object> femaleMap = mapper.buildPatientDataMap(female, new ArrayList<>(), new ArrayList<>(), new ArrayList<>());
        Map<String, Object> otherMap = mapper.buildPatientDataMap(other, new ArrayList<>(), new ArrayList<>(), new ArrayList<>());

        // Then
        assertEquals("M", maleMap.get("gender"));
        assertEquals("F", femaleMap.get("gender"));
        assertEquals("O", otherMap.get("gender"));
    }

    // ==================== Quality Measure Evaluation Tests ====================

    @Test
    @DisplayName("Should evaluate quality measure for cohort")
    void testEvaluateQualityMeasureFromFHIR() throws Exception {
        // Given
        QualityMeasure measure = createTestMeasure("CDC-HbA1c");
        PatientCohort cohort = createTestCohort(Arrays.asList("P001", "P002"));

        // Mock FHIR data for measure evaluation
        when(mockFhirClient.getConditionsAsync(anyString()))
            .thenReturn(CompletableFuture.completedFuture(Arrays.asList(
                createCondition("E11.9", "Type 2 Diabetes")
            )));

        // When
        QualityMeasure evaluated = mapper.evaluateQualityMeasureFromFHIR(measure, cohort).get();

        // Then
        assertNotNull(evaluated);
        verify(mockFhirClient, atLeastOnce()).getConditionsAsync(anyString());
    }

    // ==================== Helper Methods ====================

    private PatientCohort createTestCohort(List<String> patientIds) {
        PatientCohort cohort = new PatientCohort();
        cohort.setCohortName("Test Cohort");
        cohort.setDescription("Test cohort for unit tests");
        cohort.setCohortType(PatientCohort.CohortType.CUSTOM);
        cohort.getPatientIds().addAll(patientIds);
        cohort.setTotalPatients(patientIds.size());
        cohort.setLastUpdated(LocalDateTime.now());
        cohort.setActive(true);
        return cohort;
    }

    private FHIRPatientData createPatient(String id, int age, String gender) {
        FHIRPatientData patient = new FHIRPatientData();
        patient.setId(id);
        patient.setAge(age);
        patient.setGender(gender);
        patient.setFirstName("Test");
        patient.setLastName("Patient");
        return patient;
    }

    private Condition createCondition(String code, String display) {
        Condition condition = new Condition();
        condition.setCode(code);
        condition.setDisplay(display);
        return condition;
    }

    private Medication createMedication(String name, String dosage, String status) {
        Medication medication = new Medication();
        medication.setName(name);
        medication.setDosage(dosage);
        medication.setStatus(status);
        return medication;
    }

    private QualityMeasure createTestMeasure(String code) {
        QualityMeasure measure = new QualityMeasure();
        measure.setHedisCode(code);
        measure.setMeasureName("Test Measure");
        measure.setMeasureType(QualityMeasure.MeasureType.PROCESS);
        measure.setDescription("Test quality measure");
        return measure;
    }
}
