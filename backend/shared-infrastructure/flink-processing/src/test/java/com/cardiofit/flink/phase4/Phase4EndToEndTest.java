package com.cardiofit.flink.phase4;

import com.cardiofit.flink.intelligence.TestRecommender;
import com.cardiofit.flink.loader.DiagnosticTestLoader;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.diagnostics.TestRecommendation;
import com.cardiofit.flink.processors.ActionBuilder;
import com.cardiofit.flink.protocol.models.Protocol;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.time.Instant;
import java.util.*;

import static org.assertj.core.api.Assertions.*;

/**
 * End-to-End Integration Tests for Module 3 Phase 4 Complete Pipeline.
 *
 * Tests complete diagnostic test recommendation flow:
 * PatientEvent → Protocol Match → TestRecommender → ActionBuilder → ClinicalActions
 *
 * Scenarios:
 * 1. Sepsis: Patient with fever, hypotension → Sepsis protocol → Lactate, blood cultures, procalcitonin
 * 2. STEMI: Chest pain → STEMI protocol → Troponin I serial, ECG, CK-MB
 * 3. Verify all ClinicalAction objects have ActionType.DIAGNOSTIC
 * 4. Assert expected tests are recommended
 * 5. Verify urgency and priority mapping
 * 6. Test complete integration without mocks
 *
 * @author Module 3 Phase 4 - Quality Engineering Team
 * @version 1.0
 * @since 2025-10-23
 */
@DisplayName("Phase 4 End-to-End Integration Tests")
class Phase4EndToEndTest {

    private DiagnosticTestLoader testLoader;
    private TestRecommender testRecommender;
    private ActionBuilder actionBuilder;

    @BeforeEach
    void setUp() {
        // Initialize complete Phase 4 pipeline components
        testLoader = DiagnosticTestLoader.getInstance();
        testRecommender = new TestRecommender(testLoader);
        actionBuilder = new ActionBuilder(); // Uses TestRecommender internally
    }

    // ============================================================
    // SEPSIS SCENARIO - END-TO-END
    // ============================================================

    @Test
    @DisplayName("E2E Sepsis: Patient with fever and hypotension → Complete sepsis diagnostic bundle")
    void testEndToEnd_SepsisScenario_CompletePipeline() {
        // ===== STEP 1: Patient Event Simulation =====
        // Given: Patient with fever (38.9°C), hypotension (85/55), tachycardia (115)
        EnrichedPatientContext sepsisPatient = createSepsisPatient(
                "PATIENT-E2E-SEPSIS-001",
                65, // age
                "M", // sex
                115.0, // heart rate
                85.0, // systolic BP
                38.9, // temperature
                26.0, // respiratory rate
                90.0  // SpO2
        );

        // ===== STEP 2: Protocol Match =====
        // When: Sepsis protocol is matched
        Protocol sepsisProtocol = createProtocol(
                "SEPSIS-SSC-2021",
                "Sepsis Management Bundle",
                "INFECTIOUS",
                "Emergency Medicine"
        );

        // ===== STEP 3: TestRecommender Intelligence =====
        // When: TestRecommender generates diagnostic recommendations
        List<TestRecommendation> testRecommendations = testRecommender.recommendTests(
                sepsisPatient,
                sepsisProtocol
        );

        // Then: Should generate sepsis-specific test recommendations
        assertThat(testRecommendations).isNotEmpty()
                .withFailMessage("TestRecommender should generate sepsis diagnostic recommendations");

        assertThat(testRecommendations.size()).isGreaterThanOrEqualTo(3)
                .withFailMessage("Sepsis bundle should include at least 3 essential tests");

        // ===== STEP 4: ActionBuilder Conversion =====
        // When: ActionBuilder converts recommendations to clinical actions
        List<ClinicalAction> clinicalActions = actionBuilder.buildDiagnosticActions(
                sepsisProtocol,
                sepsisPatient
        );

        // Then: Should have clinical actions
        assertThat(clinicalActions).isNotEmpty()
                .withFailMessage("ActionBuilder should generate clinical actions");

        // ===== STEP 5: Verify Complete Pipeline Output =====

        // 5.1: All actions are DIAGNOSTIC type
        assertThat(clinicalActions).allMatch(action ->
                action.getActionType() == ClinicalAction.ActionType.DIAGNOSTIC)
                .withFailMessage("All sepsis actions should be DIAGNOSTIC type");

        // 5.2: Essential sepsis tests are present
        boolean hasLactate = clinicalActions.stream()
                .anyMatch(action -> action.getDiagnosticDetails() != null &&
                        action.getDiagnosticDetails().getTestName() != null &&
                        action.getDiagnosticDetails().getTestName().toLowerCase().contains("lactate"));

        assertThat(hasLactate).isTrue()
                .withFailMessage("Sepsis bundle should include serum lactate test");

        boolean hasBloodCultures = clinicalActions.stream()
                .anyMatch(action -> action.getDiagnosticDetails() != null &&
                        action.getDiagnosticDetails().getTestName() != null &&
                        action.getDiagnosticDetails().getTestName().toLowerCase().contains("blood culture"));

        assertThat(hasBloodCultures).isTrue()
                .withFailMessage("Sepsis bundle should include blood cultures");

        // 5.3: Verify urgency levels for critical sepsis tests
        long statOrUrgentActions = clinicalActions.stream()
                .filter(action -> "STAT".equals(action.getUrgency()) || "URGENT".equals(action.getUrgency()))
                .count();

        assertThat(statOrUrgentActions).isGreaterThan(0)
                .withFailMessage("Sepsis bundle should have STAT or URGENT tests");

        // 5.4: Verify all actions have complete diagnostic details
        for (ClinicalAction action : clinicalActions) {
            assertThat(action.getDiagnosticDetails()).isNotNull()
                    .withFailMessage("Action should have diagnostic details");

            assertThat(action.getDiagnosticDetails().getTestName()).isNotNull()
                    .withFailMessage("Diagnostic details should have test name");

            assertThat(action.getDiagnosticDetails().getClinicalIndication()).isNotNull()
                    .withFailMessage("Diagnostic details should have clinical indication");

            assertThat(action.getClinicalRationale()).isNotNull()
                    .withFailMessage("Action should have clinical rationale");

            assertThat(action.getUrgency()).isNotNull()
                    .withFailMessage("Action should have urgency level");
        }
    }

    @Test
    @DisplayName("E2E Sepsis: Verify lactate has P0_CRITICAL priority and STAT urgency")
    void testEndToEnd_SepsisScenario_LactateMetadata() {
        // Given: Sepsis patient and protocol
        EnrichedPatientContext sepsisPatient = createSepsisPatient(
                "PATIENT-E2E-SEPSIS-002", 68, "F", 120.0, 82.0, 39.2, 28.0, 88.0
        );
        Protocol sepsisProtocol = createSepsisProtocol();

        // When: Generate clinical actions
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisPatient);

        // Find lactate action
        ClinicalAction lactateAction = actions.stream()
                .filter(action -> action.getDiagnosticDetails() != null &&
                        action.getDiagnosticDetails().getTestName() != null &&
                        action.getDiagnosticDetails().getTestName().toLowerCase().contains("lactate"))
                .findFirst()
                .orElse(null);

        // Then: Lactate should have critical urgency
        assertThat(lactateAction).isNotNull()
                .withFailMessage("Sepsis bundle should include lactate");

        assertThat(lactateAction.getUrgency()).isEqualTo("STAT")
                .withFailMessage("Lactate should have STAT urgency");

        assertThat(lactateAction.getEvidenceStrength()).isIn("A", "B")
                .withFailMessage("Lactate should have strong evidence level");
    }

    @Test
    @DisplayName("E2E Sepsis: Respiratory symptoms trigger chest X-ray")
    void testEndToEnd_SepsisScenario_RespiratorySymptomsCXR() {
        // Given: Sepsis patient WITH respiratory symptoms
        EnrichedPatientContext respiratorySepsisPatient = createSepsisPatient(
                "PATIENT-E2E-SEPSIS-003", 70, "M", 118.0, 88.0, 38.5, 30.0, 86.0 // High RR, low SpO2
        );
        Protocol sepsisProtocol = createSepsisProtocol();

        // When: Generate clinical actions
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, respiratorySepsisPatient);

        // Then: Should include chest X-ray
        boolean hasCXR = actions.stream()
                .anyMatch(action -> action.getDiagnosticDetails() != null &&
                        action.getDiagnosticDetails().getTestName() != null &&
                        action.getDiagnosticDetails().getTestName().toLowerCase().contains("chest"));

        assertThat(hasCXR).isTrue()
                .withFailMessage("Sepsis with respiratory symptoms should include chest X-ray");
    }

    // ============================================================
    // STEMI SCENARIO - END-TO-END
    // ============================================================

    @Test
    @DisplayName("E2E STEMI: Patient with chest pain → Complete STEMI diagnostic bundle")
    void testEndToEnd_STEMIScenario_CompletePipeline() {
        // ===== STEP 1: Patient Event Simulation =====
        // Given: Patient with chest pain, ST elevation
        EnrichedPatientContext stemiPatient = createSTEMIPatient(
                "PATIENT-E2E-STEMI-001",
                58, // age
                "M", // sex
                95.0, // heart rate
                140.0, // systolic BP
                37.0  // temperature
        );

        // ===== STEP 2: Protocol Match =====
        Protocol stemiProtocol = createProtocol(
                "STEMI-AHA-ACC-2023",
                "STEMI Management Protocol",
                "CARDIAC",
                "Cardiology"
        );

        // ===== STEP 3: TestRecommender Intelligence =====
        List<TestRecommendation> testRecommendations = testRecommender.recommendTests(
                stemiPatient,
                stemiProtocol
        );

        // Then: Should generate STEMI-specific test recommendations
        assertThat(testRecommendations).isNotEmpty()
                .withFailMessage("TestRecommender should generate STEMI diagnostic recommendations");

        // ===== STEP 4: ActionBuilder Conversion =====
        List<ClinicalAction> clinicalActions = actionBuilder.buildDiagnosticActions(
                stemiProtocol,
                stemiPatient
        );

        // ===== STEP 5: Verify Complete Pipeline Output =====

        // 5.1: All actions are DIAGNOSTIC type
        assertThat(clinicalActions).allMatch(action ->
                action.getActionType() == ClinicalAction.ActionType.DIAGNOSTIC)
                .withFailMessage("All STEMI actions should be DIAGNOSTIC type");

        // 5.2: Essential STEMI tests are present
        boolean hasTroponin = clinicalActions.stream()
                .anyMatch(action -> action.getDiagnosticDetails() != null &&
                        action.getDiagnosticDetails().getTestName() != null &&
                        action.getDiagnosticDetails().getTestName().toLowerCase().contains("troponin"));

        assertThat(hasTroponin).isTrue()
                .withFailMessage("STEMI bundle should include troponin test");

        // 5.3: Troponin has STAT urgency
        ClinicalAction troponinAction = clinicalActions.stream()
                .filter(action -> action.getDiagnosticDetails() != null &&
                        action.getDiagnosticDetails().getTestName() != null &&
                        action.getDiagnosticDetails().getTestName().toLowerCase().contains("troponin"))
                .findFirst()
                .orElse(null);

        if (troponinAction != null) {
            assertThat(troponinAction.getUrgency()).isEqualTo("STAT")
                    .withFailMessage("Troponin should have STAT urgency in STEMI");
        }

        // 5.4: Verify comprehensive metabolic panel for renal function
        boolean hasCMP = clinicalActions.stream()
                .anyMatch(action -> action.getDiagnosticDetails() != null &&
                        action.getDiagnosticDetails().getTestName() != null &&
                        action.getDiagnosticDetails().getTestName().toLowerCase().contains("metabolic"));

        assertThat(hasCMP).isTrue()
                .withFailMessage("STEMI bundle should include CMP for renal function assessment");
    }

    @Test
    @DisplayName("E2E STEMI: Troponin has serial testing follow-up guidance")
    void testEndToEnd_STEMIScenario_TroponinSerialTesting() {
        // Given: STEMI patient
        EnrichedPatientContext stemiPatient = createSTEMIPatient(
                "PATIENT-E2E-STEMI-002", 62, "M", 100.0, 135.0, 37.2
        );
        Protocol stemiProtocol = createSTEMIProtocol();

        // When: Generate test recommendations (not actions, to check follow-up guidance)
        List<TestRecommendation> recommendations = testRecommender.recommendTests(stemiPatient, stemiProtocol);

        // Find troponin recommendation
        TestRecommendation troponinRec = recommendations.stream()
                .filter(rec -> rec.getTestName().toLowerCase().contains("troponin"))
                .findFirst()
                .orElse(null);

        // Then: Should have follow-up guidance for serial testing
        assertThat(troponinRec).isNotNull();
        assertThat(troponinRec.getFollowUpGuidance()).isNotNull()
                .withFailMessage("Troponin should have follow-up guidance");

        assertThat(troponinRec.getFollowUpGuidance().getRepeatIntervalHours()).isEqualTo(3)
                .withFailMessage("Troponin should recommend repeat in 3 hours");

        assertThat(troponinRec.getFollowUpGuidance().getReflexTests()).isNotNull()
                .isNotEmpty()
                .withFailMessage("Troponin should have reflex tests");
    }

    // ============================================================
    // CROSS-SCENARIO TESTS
    // ============================================================

    @Test
    @DisplayName("E2E Cross-Scenario: Different protocols generate different test bundles")
    void testEndToEnd_CrossScenario_DifferentBundles() {
        // Given: Same patient, different protocols
        EnrichedPatientContext patient = createSepsisPatient(
                "PATIENT-E2E-CROSS-001", 60, "M", 110.0, 90.0, 38.0, 24.0, 92.0
        );

        Protocol sepsisProtocol = createSepsisProtocol();
        Protocol stemiProtocol = createSTEMIProtocol();

        // When: Generate actions for both protocols
        List<ClinicalAction> sepsisActions = actionBuilder.buildDiagnosticActions(sepsisProtocol, patient);
        List<ClinicalAction> stemiActions = actionBuilder.buildDiagnosticActions(stemiProtocol, patient);

        // Then: Should generate different test bundles
        assertThat(sepsisActions).isNotEmpty();
        assertThat(stemiActions).isNotEmpty();

        // Sepsis should have lactate
        boolean sepsisHasLactate = sepsisActions.stream()
                .anyMatch(action -> action.getDiagnosticDetails().getTestName().toLowerCase().contains("lactate"));

        // STEMI should have troponin
        boolean stemiHasTroponin = stemiActions.stream()
                .anyMatch(action -> action.getDiagnosticDetails().getTestName().toLowerCase().contains("troponin"));

        assertThat(sepsisHasLactate).isTrue()
                .withFailMessage("Sepsis protocol should include lactate");

        assertThat(stemiHasTroponin).isTrue()
                .withFailMessage("STEMI protocol should include troponin");
    }

    @Test
    @DisplayName("E2E Cross-Scenario: All actions across scenarios have valid structure")
    void testEndToEnd_CrossScenario_AllActionsValidStructure() {
        // Given: Multiple patients and protocols
        EnrichedPatientContext sepsisPatient = createSepsisPatient(
                "PATIENT-E2E-VALID-001", 65, "F", 115.0, 85.0, 39.0, 26.0, 90.0
        );
        EnrichedPatientContext stemiPatient = createSTEMIPatient(
                "PATIENT-E2E-VALID-002", 58, "M", 95.0, 140.0, 37.0
        );

        Protocol sepsisProtocol = createSepsisProtocol();
        Protocol stemiProtocol = createSTEMIProtocol();

        // When: Generate all actions
        List<ClinicalAction> allActions = new ArrayList<>();
        allActions.addAll(actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisPatient));
        allActions.addAll(actionBuilder.buildDiagnosticActions(stemiProtocol, stemiPatient));

        // Then: All actions should have valid structure
        for (ClinicalAction action : allActions) {
            // Required fields
            assertThat(action.getActionId()).isNotNull();
            assertThat(action.getActionType()).isEqualTo(ClinicalAction.ActionType.DIAGNOSTIC);
            assertThat(action.getUrgency()).isNotNull().isIn("STAT", "URGENT", "TODAY", "ROUTINE", "SCHEDULED");
            assertThat(action.getDiagnosticDetails()).isNotNull();

            // Diagnostic details
            DiagnosticDetails details = action.getDiagnosticDetails();
            assertThat(details.getTestName()).isNotNull();
            assertThat(details.getClinicalIndication()).isNotNull();

            // Clinical rationale
            assertThat(action.getClinicalRationale()).isNotNull();

            // Description
            assertThat(action.getDescription()).isNotNull();
        }
    }

    // ============================================================
    // PERFORMANCE TESTS
    // ============================================================

    @Test
    @DisplayName("E2E Performance: Complete pipeline executes in <500ms")
    void testEndToEnd_Performance_FastExecution() {
        // Given: Patient and protocol
        EnrichedPatientContext patient = createSepsisPatient(
                "PATIENT-E2E-PERF-001", 65, "M", 115.0, 85.0, 38.9, 26.0, 90.0
        );
        Protocol protocol = createSepsisProtocol();

        // When: Measure complete pipeline execution time
        long startTime = System.currentTimeMillis();
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(protocol, patient);
        long elapsedTime = System.currentTimeMillis() - startTime;

        // Then: Should execute quickly
        assertThat(elapsedTime).isLessThan(500)
                .withFailMessage("Complete Phase 4 pipeline should execute in <500ms, took %dms", elapsedTime);

        assertThat(actions).isNotEmpty()
                .withFailMessage("Should generate actions within time limit");
    }

    // ============================================================
    // HELPER METHODS - TEST DATA CREATION
    // ============================================================

    private EnrichedPatientContext createSepsisPatient(
            String patientId,
            int age,
            String sex,
            double heartRate,
            double systolicBP,
            double temperature,
            double respiratoryRate,
            double spo2
    ) {
        EnrichedPatientContext context = new EnrichedPatientContext();
        context.setPatientId(patientId);
        context.setEncounterId("ENC-" + patientId);
        context.setTriggerTime(Instant.now());

        PatientContextState state = new PatientContextState();

        PatientDemographics demographics = new PatientDemographics();
        demographics.setAge(age);
        demographics.setWeight(75.0);
        demographics.setSex(sex);
        state.setDemographics(demographics);

        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartrate", heartRate);
        vitals.put("systolicbp", systolicBP);
        vitals.put("temperature", temperature);
        vitals.put("respiratoryrate", respiratoryRate);
        vitals.put("spo2", spo2);
        state.setLatestVitals(vitals);

        setCreatinineValue(state, 1.2);

        context.setPatientState(state);
        return context;
    }

    private EnrichedPatientContext createSTEMIPatient(
            String patientId,
            int age,
            String sex,
            double heartRate,
            double systolicBP,
            double temperature
    ) {
        EnrichedPatientContext context = new EnrichedPatientContext();
        context.setPatientId(patientId);
        context.setEncounterId("ENC-" + patientId);
        context.setTriggerTime(Instant.now());

        PatientContextState state = new PatientContextState();

        PatientDemographics demographics = new PatientDemographics();
        demographics.setAge(age);
        demographics.setWeight(85.0);
        demographics.setSex(sex);
        state.setDemographics(demographics);

        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartrate", heartRate);
        vitals.put("systolicbp", systolicBP);
        vitals.put("temperature", temperature);
        vitals.put("respiratoryrate", 18.0);
        state.setLatestVitals(vitals);

        setCreatinineValue(state, 1.0);

        context.setPatientState(state);
        return context;
    }

    private Protocol createSepsisProtocol() {
        return createProtocol(
                "SEPSIS-SSC-2021",
                "Sepsis Management Bundle",
                "INFECTIOUS",
                "Emergency Medicine"
        );
    }

    private Protocol createSTEMIProtocol() {
        return createProtocol(
                "STEMI-AHA-ACC-2023",
                "STEMI Management Protocol",
                "CARDIAC",
                "Cardiology"
        );
    }

    private Protocol createProtocol(String id, String name, String category, String specialty) {
        Protocol protocol = new Protocol();
        protocol.setProtocolId(id);
        protocol.setName(name);
        protocol.setCategory(category);
        protocol.setSpecialty(specialty);
        return protocol;
    }

    /**
     * Helper method to set creatinine value using Map-based API
     */
    private void setCreatinineValue(PatientContextState state, double creatinineValue) {
        Map<String, LabResult> recentLabs = state.getRecentLabs();
        if (recentLabs == null) {
            recentLabs = new HashMap<>();
            state.setRecentLabs(recentLabs);
        }

        LabResult creatinineResult = new LabResult();
        creatinineResult.setLabCode("2160-0"); // LOINC code for creatinine
        creatinineResult.setLabType("Creatinine");
        creatinineResult.setValue(creatinineValue);
        creatinineResult.setUnit("mg/dL");
        creatinineResult.setTimestamp(System.currentTimeMillis());

        recentLabs.put("2160-0", creatinineResult);
    }
}
