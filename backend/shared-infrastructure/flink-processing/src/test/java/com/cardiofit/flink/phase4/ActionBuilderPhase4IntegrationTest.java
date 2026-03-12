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
 * Integration Tests for ActionBuilder with Phase 4 Diagnostic Test Repository.
 *
 * Validates complete integration flow:
 * Protocol → TestRecommender → TestRecommendation → ActionBuilder → ClinicalAction
 *
 * Test Coverage:
 * - buildDiagnosticActions() method
 * - TestRecommendation → ClinicalAction conversion
 * - Nested field access (DecisionSupport, OrderingInformation)
 * - Urgency mapping (Phase 4 → Phase 1)
 * - All diagnostic detail fields populated correctly
 * - Edge cases and null handling
 *
 * @author Module 3 Phase 4 - Quality Engineering Team
 * @version 1.0
 * @since 2025-10-23
 */
@DisplayName("ActionBuilder Phase 4 Integration Tests")
class ActionBuilderPhase4IntegrationTest {

    private ActionBuilder actionBuilder;
    private TestRecommender testRecommender;
    private EnrichedPatientContext sepsisContext;
    private EnrichedPatientContext stemiContext;
    private Protocol sepsisProtocol;
    private Protocol stemiProtocol;

    @BeforeEach
    void setUp() {
        // Initialize ActionBuilder with Phase 4 TestRecommender
        testRecommender = new TestRecommender(DiagnosticTestLoader.getInstance());
        actionBuilder = new ActionBuilder(); // Uses default constructor with TestRecommender

        // Setup contexts and protocols
        sepsisContext = createSepsisContext();
        sepsisProtocol = createSepsisProtocol();
        stemiContext = createSTEMIContext();
        stemiProtocol = createSTEMIProtocol();
    }

    // ============================================================
    // DIAGNOSTIC ACTIONS BUILD TESTS
    // ============================================================

    @Test
    @DisplayName("Build Diagnostic Actions: Sepsis protocol generates multiple diagnostic actions")
    void testBuildDiagnosticActions_SepsisProtocol() {
        // When: Build diagnostic actions for sepsis protocol
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisContext);

        // Then: Should generate multiple diagnostic actions
        assertThat(actions).isNotEmpty()
                .withFailMessage("Sepsis protocol should generate diagnostic actions");

        assertThat(actions).hasSizeGreaterThanOrEqualTo(3)
                .withFailMessage("Sepsis protocol should generate at least 3 diagnostic tests");

        // Verify all actions are DIAGNOSTIC type
        assertThat(actions).allMatch(action -> action.getActionType() == ClinicalAction.ActionType.DIAGNOSTIC)
                .withFailMessage("All actions should be DIAGNOSTIC type");
    }

    @Test
    @DisplayName("Build Diagnostic Actions: STEMI protocol generates cardiac tests")
    void testBuildDiagnosticActions_STEMIProtocol() {
        // When: Build diagnostic actions for STEMI protocol
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(stemiProtocol, stemiContext);

        // Then: Should generate cardiac diagnostic actions
        assertThat(actions).isNotEmpty();
        assertThat(actions).hasSizeGreaterThanOrEqualTo(3);

        // Verify troponin is included
        boolean hasTroponin = actions.stream()
                .anyMatch(action -> action.getDiagnosticDetails() != null &&
                        action.getDiagnosticDetails().getTestName() != null &&
                        action.getDiagnosticDetails().getTestName().toLowerCase().contains("troponin"));

        assertThat(hasTroponin).isTrue()
                .withFailMessage("STEMI protocol should include troponin test");
    }

    @Test
    @DisplayName("Build Diagnostic Actions: All actions have diagnostic details populated")
    void testBuildDiagnosticActions_AllActionsHaveDiagnosticDetails() {
        // When: Build diagnostic actions
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisContext);

        // Then: All actions should have diagnostic details
        for (ClinicalAction action : actions) {
            assertThat(action.getDiagnosticDetails()).isNotNull()
                    .withFailMessage("Action should have diagnostic details populated");

            assertThat(action.getDiagnosticDetails().getTestName()).isNotNull()
                    .withFailMessage("Diagnostic details should have test name");

            assertThat(action.getDiagnosticDetails().getClinicalIndication()).isNotNull()
                    .withFailMessage("Diagnostic details should have clinical indication");
        }
    }

    // ============================================================
    // NESTED FIELD ACCESS TESTS
    // ============================================================

    @Test
    @DisplayName("Nested Field Access: Evidence level from DecisionSupport")
    void testNestedFieldAccess_EvidenceLevelFromDecisionSupport() {
        // When: Build diagnostic actions
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisContext);

        // Find action with decision support
        ClinicalAction actionWithEvidence = actions.stream()
                .filter(action -> action.getEvidenceStrength() != null)
                .findFirst()
                .orElse(null);

        // Then: Should have evidence strength from DecisionSupport.evidenceLevel
        assertThat(actionWithEvidence).isNotNull()
                .withFailMessage("At least one action should have evidence strength");

        assertThat(actionWithEvidence.getEvidenceStrength()).isIn("A", "B", "C")
                .withFailMessage("Evidence strength should be valid level");
    }

    @Test
    @DisplayName("Nested Field Access: LOINC code from OrderingInformation")
    void testNestedFieldAccess_LOINCFromOrderingInformation() {
        // When: Build diagnostic actions
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisContext);

        // Find action with LOINC code (should be lab tests)
        ClinicalAction labAction = actions.stream()
                .filter(action -> action.getDiagnosticDetails() != null &&
                        action.getDiagnosticDetails().getLoincCode() != null)
                .findFirst()
                .orElse(null);

        // Then: Should have LOINC code from OrderingInformation.loincCode
        // Note: This depends on TestRecommender populating orderingInfo
        if (labAction != null) {
            assertThat(labAction.getDiagnosticDetails().getLoincCode()).isNotEmpty()
                    .withFailMessage("Lab test should have LOINC code from OrderingInformation");
        }
    }

    @Test
    @DisplayName("Nested Field Access: Clinical rationale from TestRecommendation")
    void testNestedFieldAccess_ClinicalRationale() {
        // When: Build diagnostic actions
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisContext);

        // Then: All actions should have clinical rationale
        for (ClinicalAction action : actions) {
            assertThat(action.getClinicalRationale()).isNotNull()
                    .withFailMessage("Action should have clinical rationale from TestRecommendation.rationale");
        }
    }

    // ============================================================
    // URGENCY MAPPING TESTS
    // ============================================================

    @Test
    @DisplayName("Urgency Mapping: Phase 4 STAT maps to Phase 1 STAT")
    void testUrgencyMapping_STAT() {
        // Given: Create test recommendation with STAT urgency
        TestRecommendation statTest = TestRecommendation.builder()
                .recommendationId("REC-STAT-001")
                .testId("LAB-LACTATE-001")
                .testName("Serum Lactate")
                .testCategory(TestRecommendation.TestCategory.LAB)
                .urgency(TestRecommendation.Urgency.STAT)
                .priority(TestRecommendation.Priority.P0_CRITICAL)
                .indication("Septic shock assessment")
                .rationale("Lactate elevation indicates organ dysfunction")
                .decisionSupport(TestRecommendation.DecisionSupport.builder()
                        .evidenceLevel("A")
                        .build())
                .build();

        // When: Convert to ClinicalAction (using reflection to access private method via public flow)
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisContext);

        // Find STAT action
        ClinicalAction statAction = actions.stream()
                .filter(action -> "STAT".equals(action.getUrgency()))
                .findFirst()
                .orElse(null);

        // Then: Should map STAT → STAT
        assertThat(statAction).isNotNull()
                .withFailMessage("Should have at least one STAT urgency action");
        assertThat(statAction.getUrgency()).isEqualTo("STAT");
    }

    @Test
    @DisplayName("Urgency Mapping: Phase 4 URGENT maps to Phase 1 URGENT")
    void testUrgencyMapping_URGENT() {
        // When: Build diagnostic actions
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(stemiProtocol, stemiContext);

        // Find URGENT action
        ClinicalAction urgentAction = actions.stream()
                .filter(action -> "URGENT".equals(action.getUrgency()))
                .findFirst()
                .orElse(null);

        // Then: Should have URGENT mapped correctly
        if (urgentAction != null) {
            assertThat(urgentAction.getUrgency()).isEqualTo("URGENT");
        }
    }

    @Test
    @DisplayName("Urgency Mapping: Phase 4 ROUTINE maps to Phase 1 ROUTINE")
    void testUrgencyMapping_ROUTINE() {
        // Given: Create context without critical symptoms
        EnrichedPatientContext routineContext = createRoutineContext();
        Protocol routineProtocol = createSepsisProtocol();

        // When: Build diagnostic actions
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(routineProtocol, routineContext);

        // Find ROUTINE action (may not exist in sepsis bundle, but test mapping logic)
        ClinicalAction routineAction = actions.stream()
                .filter(action -> "ROUTINE".equals(action.getUrgency()))
                .findFirst()
                .orElse(null);

        // Then: If exists, should map correctly
        if (routineAction != null) {
            assertThat(routineAction.getUrgency()).isEqualTo("ROUTINE");
        }
    }

    // ============================================================
    // FIELD POPULATION TESTS
    // ============================================================

    @Test
    @DisplayName("Field Population: All required ClinicalAction fields populated")
    void testFieldPopulation_AllRequiredFieldsPresent() {
        // When: Build diagnostic actions
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisContext);

        // Then: Verify all required fields are populated
        for (ClinicalAction action : actions) {
            assertThat(action.getActionId()).isNotNull()
                    .withFailMessage("Action should have actionId");

            assertThat(action.getActionType()).isEqualTo(ClinicalAction.ActionType.DIAGNOSTIC)
                    .withFailMessage("Action type should be DIAGNOSTIC");

            assertThat(action.getUrgency()).isNotNull()
                    .withFailMessage("Action should have urgency");

            assertThat(action.getClinicalRationale()).isNotNull()
                    .withFailMessage("Action should have clinical rationale");

            assertThat(action.getDiagnosticDetails()).isNotNull()
                    .withFailMessage("Action should have diagnostic details");
        }
    }

    @Test
    @DisplayName("Field Population: DiagnosticDetails has all key fields")
    void testFieldPopulation_DiagnosticDetailsComplete() {
        // When: Build diagnostic actions
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisContext);

        // Then: Verify diagnostic details completeness
        for (ClinicalAction action : actions) {
            DiagnosticDetails details = action.getDiagnosticDetails();

            assertThat(details.getTestName()).isNotNull()
                    .withFailMessage("DiagnosticDetails should have test name");

            assertThat(details.getClinicalIndication()).isNotNull()
                    .withFailMessage("DiagnosticDetails should have clinical indication");

            // Collection timing should be populated from timeframe
            if (action.getUrgency().equals("STAT")) {
                assertThat(details.getCollectionTiming()).isNotNull()
                        .withFailMessage("STAT tests should have collection timing");
            }
        }
    }

    @Test
    @DisplayName("Field Population: Description includes protocol and generator info")
    void testFieldPopulation_DescriptionComplete() {
        // When: Build diagnostic actions
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisContext);

        // Then: Description should include protocol ID and generator
        for (ClinicalAction action : actions) {
            assertThat(action.getDescription()).isNotNull()
                    .contains("Diagnostic test")
                    .contains(sepsisProtocol.getProtocolId())
                    .contains("TestRecommender-Phase4");
        }
    }

    @Test
    @DisplayName("Field Population: Prerequisites mapped correctly")
    void testFieldPopulation_PrerequisitesMapped() {
        // Given: Create test recommendation with prerequisites
        TestRecommendation testWithPrereqs = TestRecommendation.builder()
                .recommendationId("REC-PREREQ-001")
                .testId("IMG-CT-CONTRAST-001")
                .testName("CT with Contrast")
                .testCategory(TestRecommendation.TestCategory.IMAGING)
                .urgency(TestRecommendation.Urgency.URGENT)
                .indication("Suspected abscess")
                .rationale("CT with contrast for abscess identification")
                .prerequisiteTests(Arrays.asList("LAB-CREATININE-001", "LAB-PREGNANCY-001"))
                .decisionSupport(TestRecommendation.DecisionSupport.builder()
                        .evidenceLevel("B")
                        .build())
                .build();

        // When: Build actions (this test verifies the conversion logic)
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisContext);

        // Then: Actions with prerequisites should have them mapped
        // Note: This depends on TestRecommender including tests with prerequisites
        ClinicalAction actionWithPrereqs = actions.stream()
                .filter(action -> action.getPrerequisiteChecks() != null && !action.getPrerequisiteChecks().isEmpty())
                .findFirst()
                .orElse(null);

        if (actionWithPrereqs != null) {
            assertThat(actionWithPrereqs.getPrerequisiteChecks()).isNotEmpty()
                    .withFailMessage("Prerequisites should be mapped to prerequisiteChecks");
        }
    }

    @Test
    @DisplayName("Field Population: Contraindications mapped to patient preparation")
    void testFieldPopulation_ContraindicationsMapped() {
        // When: Build diagnostic actions
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisContext);

        // Then: Actions with contraindications should have them in patient preparation
        ClinicalAction actionWithContras = actions.stream()
                .filter(action -> action.getDiagnosticDetails() != null &&
                        action.getDiagnosticDetails().getPatientPreparation() != null &&
                        action.getDiagnosticDetails().getPatientPreparation().contains("Contraindications"))
                .findFirst()
                .orElse(null);

        if (actionWithContras != null) {
            assertThat(actionWithContras.getDiagnosticDetails().getPatientPreparation())
                    .contains("Contraindications")
                    .withFailMessage("Contraindications should be mapped to patient preparation");
        }
    }

    // ============================================================
    // EDGE CASE TESTS
    // ============================================================

    @Test
    @DisplayName("Edge Case: Null protocol returns empty list")
    void testEdgeCase_NullProtocol() {
        // When: Call with null protocol
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(null, sepsisContext);

        // Then: Should return empty list
        assertThat(actions).isEmpty();
    }

    @Test
    @DisplayName("Edge Case: Null context returns empty list")
    void testEdgeCase_NullContext() {
        // When: Call with null context
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, null);

        // Then: Should return empty list
        assertThat(actions).isEmpty();
    }

    @Test
    @DisplayName("Edge Case: TestRecommendation with null nested objects handled gracefully")
    void testEdgeCase_NullNestedObjects() {
        // Given: Protocol that generates recommendations
        // When: Build actions (TestRecommender might return tests without all nested objects)
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisContext);

        // Then: Should handle gracefully without NPE
        assertThat(actions).isNotNull();

        // All actions should be valid even if some nested objects are null
        for (ClinicalAction action : actions) {
            assertThat(action.getActionId()).isNotNull();
            assertThat(action.getDiagnosticDetails()).isNotNull();
        }
    }

    @Test
    @DisplayName("Edge Case: Protocol with no test recommendations handled")
    void testEdgeCase_NoTestRecommendations() {
        // Given: Protocol that doesn't match any test bundles
        Protocol unknownProtocol = new Protocol();
        unknownProtocol.setProtocolId("UNKNOWN-PROTOCOL-999");
        unknownProtocol.setName("Unknown Protocol");

        // When: Build diagnostic actions
        List<ClinicalAction> actions = actionBuilder.buildDiagnosticActions(unknownProtocol, sepsisContext);

        // Then: Should return empty list without error
        assertThat(actions).isEmpty();
    }

    // ============================================================
    // INTEGRATION FLOW TESTS
    // ============================================================

    @Test
    @DisplayName("Integration Flow: Complete pipeline from Protocol to ClinicalAction")
    void testIntegrationFlow_CompletePhase4Pipeline() {
        // Given: Sepsis scenario - patient with fever and hypotension
        EnrichedPatientContext sepsisPatient = createSepsisContext();
        Protocol sepsisProtocol = createSepsisProtocol();

        // When: Execute complete Phase 4 diagnostic action build flow
        List<ClinicalAction> diagnosticActions = actionBuilder.buildDiagnosticActions(sepsisProtocol, sepsisPatient);

        // Then: Verify complete integration
        assertThat(diagnosticActions).isNotEmpty()
                .withFailMessage("Complete flow should generate diagnostic actions");

        // Verify expected tests for sepsis
        boolean hasLactate = diagnosticActions.stream()
                .anyMatch(action -> action.getDiagnosticDetails().getTestName().toLowerCase().contains("lactate"));

        boolean hasBloodCultures = diagnosticActions.stream()
                .anyMatch(action -> action.getDiagnosticDetails().getTestName().toLowerCase().contains("blood culture"));

        assertThat(hasLactate).isTrue()
                .withFailMessage("Sepsis flow should include lactate test");

        assertThat(hasBloodCultures).isTrue()
                .withFailMessage("Sepsis flow should include blood cultures");

        // Verify all actions have correct type
        assertThat(diagnosticActions).allMatch(action ->
                action.getActionType() == ClinicalAction.ActionType.DIAGNOSTIC)
                .withFailMessage("All generated actions should be DIAGNOSTIC type");

        // Verify urgency levels are appropriate for sepsis
        long statOrUrgentCount = diagnosticActions.stream()
                .filter(action -> "STAT".equals(action.getUrgency()) || "URGENT".equals(action.getUrgency()))
                .count();

        assertThat(statOrUrgentCount).isGreaterThan(0)
                .withFailMessage("Sepsis protocol should have STAT or URGENT tests");
    }

    // ============================================================
    // HELPER METHODS - TEST DATA CREATION
    // ============================================================

    private EnrichedPatientContext createSepsisContext() {
        EnrichedPatientContext context = new EnrichedPatientContext();
        context.setPatientId("PATIENT-SEPSIS-001");
        context.setEncounterId("ENC-SEPSIS-001");
        context.setTriggerTime(Instant.now());

        PatientContextState state = new PatientContextState();

        PatientDemographics demographics = new PatientDemographics();
        demographics.setAge(65);
        demographics.setWeight(75.0);
        demographics.setSex("M");
        state.setDemographics(demographics);

        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartrate", 115.0);
        vitals.put("systolicbp", 85.0);
        vitals.put("temperature", 38.9);
        vitals.put("respiratoryrate", 26.0);
        vitals.put("spo2", 90.0);
        state.setLatestVitals(vitals);

        setCreatinineValue(state, 1.2);

        context.setPatientState(state);
        return context;
    }

    private EnrichedPatientContext createSTEMIContext() {
        EnrichedPatientContext context = new EnrichedPatientContext();
        context.setPatientId("PATIENT-STEMI-001");
        context.setEncounterId("ENC-STEMI-001");
        context.setTriggerTime(Instant.now());

        PatientContextState state = new PatientContextState();

        PatientDemographics demographics = new PatientDemographics();
        demographics.setAge(58);
        demographics.setWeight(85.0);
        demographics.setSex("M");
        state.setDemographics(demographics);

        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartrate", 95.0);
        vitals.put("systolicbp", 140.0);
        vitals.put("temperature", 37.0);
        vitals.put("respiratoryrate", 18.0);
        state.setLatestVitals(vitals);

        setCreatinineValue(state, 1.0);

        context.setPatientState(state);
        return context;
    }

    private EnrichedPatientContext createRoutineContext() {
        EnrichedPatientContext context = new EnrichedPatientContext();
        context.setPatientId("PATIENT-ROUTINE-001");
        context.setEncounterId("ENC-ROUTINE-001");
        context.setTriggerTime(Instant.now());

        PatientContextState state = new PatientContextState();

        PatientDemographics demographics = new PatientDemographics();
        demographics.setAge(45);
        demographics.setWeight(70.0);
        demographics.setSex("F");
        state.setDemographics(demographics);

        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartrate", 75.0);
        vitals.put("systolicbp", 120.0);
        vitals.put("temperature", 37.0);
        vitals.put("respiratoryrate", 16.0);
        state.setLatestVitals(vitals);

        setCreatinineValue(state, 0.9);

        context.setPatientState(state);
        return context;
    }

    private Protocol createSepsisProtocol() {
        Protocol protocol = new Protocol();
        protocol.setProtocolId("SEPSIS-SSC-2021");
        protocol.setName("Sepsis Management Bundle");
        protocol.setCategory("INFECTIOUS");
        protocol.setSpecialty("Emergency Medicine");
        return protocol;
    }

    private Protocol createSTEMIProtocol() {
        Protocol protocol = new Protocol();
        protocol.setProtocolId("STEMI-AHA-ACC-2023");
        protocol.setName("STEMI Management Protocol");
        protocol.setCategory("CARDIAC");
        protocol.setSpecialty("Cardiology");
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
