package com.cardiofit.flink.models.diagnostics;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Comprehensive unit tests for TestRecommendation model (Phase 4).
 *
 * Test Coverage:
 * - Priority and urgency calculations: 6 tests
 * - Contraindication checks: 3 tests
 * - Validity and timing checks: 4 tests
 * - Prerequisite checks: 2 tests
 * - Helper methods: 5 tests
 *
 * Total: 20 unit tests for TestRecommendation
 *
 * @author Module 3 Phase 4 Team
 * @version 1.0
 * @since 2025-10-23
 */
@DisplayName("TestRecommendation Model Tests")
class TestRecommendationTest {

    private TestRecommendation statLactateRecommendation;
    private TestRecommendation routineChestXRayRecommendation;

    @BeforeEach
    void setUp() {
        statLactateRecommendation = createSTATLactateRecommendation();
        routineChestXRayRecommendation = createRoutineChestXRayRecommendation();
    }

    // ============================================================
    // PRIORITY AND URGENCY TESTS (6 tests)
    // ============================================================

    @Test
    @DisplayName("Is High Priority: P0 CRITICAL is high priority")
    void testIsHighPriority_P0Critical() {
        // Then: STAT lactate with P0 priority should be high priority
        assertTrue(statLactateRecommendation.isHighPriority());
        assertEquals(TestRecommendation.Priority.P0_CRITICAL,
                    statLactateRecommendation.getPriority());
    }

    @Test
    @DisplayName("Is High Priority: P1 URGENT is high priority")
    void testIsHighPriority_P1Urgent() {
        // Given: Recommendation with P1 priority
        TestRecommendation recommendation = TestRecommendation.builder()
            .recommendationId("REC-001")
            .priority(TestRecommendation.Priority.P1_URGENT)
            .build();

        // Then: Should be high priority
        assertTrue(recommendation.isHighPriority());
    }

    @Test
    @DisplayName("Is High Priority: P2 IMPORTANT is not high priority")
    void testIsHighPriority_P2NotHigh() {
        // Given: Recommendation with P2 priority
        TestRecommendation recommendation = TestRecommendation.builder()
            .recommendationId("REC-001")
            .priority(TestRecommendation.Priority.P2_IMPORTANT)
            .build();

        // Then: Should NOT be high priority
        assertFalse(recommendation.isHighPriority());
    }

    @Test
    @DisplayName("Requires Immediate Action: STAT urgency")
    void testRequiresImmediateAction_STAT() {
        // Then: STAT urgency requires immediate action
        assertTrue(statLactateRecommendation.requiresImmediateAction());
        assertEquals(TestRecommendation.Urgency.STAT,
                    statLactateRecommendation.getUrgency());
    }

    @Test
    @DisplayName("Requires Immediate Action: URGENT urgency")
    void testRequiresImmediateAction_Urgent() {
        // Given: Recommendation with URGENT urgency
        TestRecommendation recommendation = TestRecommendation.builder()
            .recommendationId("REC-001")
            .urgency(TestRecommendation.Urgency.URGENT)
            .build();

        // Then: Should require immediate action
        assertTrue(recommendation.requiresImmediateAction());
    }

    @Test
    @DisplayName("Requires Immediate Action: ROUTINE does not require immediate action")
    void testRequiresImmediateAction_RoutineDoesNot() {
        // Then: ROUTINE urgency does NOT require immediate action
        assertFalse(routineChestXRayRecommendation.requiresImmediateAction());
        assertEquals(TestRecommendation.Urgency.ROUTINE,
                    routineChestXRayRecommendation.getUrgency());
    }

    // ============================================================
    // CONTRAINDICATION TESTS (3 tests)
    // ============================================================

    @Test
    @DisplayName("Has Contraindication: Patient condition matches contraindication")
    void testHasContraindication_MatchFound() {
        // Given: Recommendation with contrast contraindication
        TestRecommendation ctWithContrast = TestRecommendation.builder()
            .recommendationId("REC-CT-001")
            .testName("CT with Contrast")
            .contraindications(Arrays.asList("severe renal failure", "contrast allergy"))
            .build();

        // Patient has renal failure
        List<String> patientConditions = Arrays.asList(
            "Severe renal failure (GFR 15)",
            "Hypertension"
        );

        // When: Check for contraindication
        boolean hasContra = ctWithContrast.hasContraindication(patientConditions);

        // Then: Should detect contraindication
        assertTrue(hasContra);
    }

    @Test
    @DisplayName("Has Contraindication: No match found")
    void testHasContraindication_NoMatchFound() {
        // Given: Recommendation with specific contraindications
        TestRecommendation recommendation = TestRecommendation.builder()
            .recommendationId("REC-001")
            .contraindications(Arrays.asList("pregnancy", "severe allergy"))
            .build();

        // Patient has no matching conditions
        List<String> patientConditions = Arrays.asList("Hypertension", "Diabetes");

        // When: Check for contraindication
        boolean hasContra = recommendation.hasContraindication(patientConditions);

        // Then: Should NOT detect contraindication
        assertFalse(hasContra);
    }

    @Test
    @DisplayName("Has Contraindication: Null or empty lists return false")
    void testHasContraindication_NullLists() {
        // Given: Recommendation with contraindications
        TestRecommendation recommendation = TestRecommendation.builder()
            .recommendationId("REC-001")
            .contraindications(Arrays.asList("pregnancy"))
            .build();

        // When: Check with null patient conditions
        boolean hasContra = recommendation.hasContraindication(null);

        // Then: Should return false (no match possible)
        assertFalse(hasContra);

        // When: Check with empty list
        hasContra = recommendation.hasContraindication(new ArrayList<>());

        // Then: Should return false
        assertFalse(hasContra);
    }

    // ============================================================
    // VALIDITY AND TIMING TESTS (4 tests)
    // ============================================================

    @Test
    @DisplayName("Is Still Valid: Recent recommendation within timeframe")
    void testIsStillValid_RecentRecommendation() {
        // Given: Recommendation created 10 minutes ago with 60-minute timeframe
        long tenMinutesAgo = System.currentTimeMillis() - (10 * 60 * 1000);
        TestRecommendation recommendation = TestRecommendation.builder()
            .recommendationId("REC-001")
            .timestamp(tenMinutesAgo)
            .timeframeMinutes(60)
            .build();

        // When: Check if still valid
        boolean valid = recommendation.isStillValid();

        // Then: Should be valid
        assertTrue(valid);
    }

    @Test
    @DisplayName("Is Still Valid: Expired recommendation outside timeframe")
    void testIsStillValid_ExpiredRecommendation() {
        // Given: Recommendation created 90 minutes ago with 60-minute timeframe
        long ninetyMinutesAgo = System.currentTimeMillis() - (90 * 60 * 1000);
        TestRecommendation recommendation = TestRecommendation.builder()
            .recommendationId("REC-001")
            .timestamp(ninetyMinutesAgo)
            .timeframeMinutes(60)
            .build();

        // When: Check if still valid
        boolean valid = recommendation.isStillValid();

        // Then: Should NOT be valid
        assertFalse(valid);
    }

    @Test
    @DisplayName("Get Urgency Deadline: STAT = 1 hour")
    void testGetUrgencyDeadline_STAT() {
        // When: Get urgency deadline for STAT
        Long deadline = statLactateRecommendation.getUrgencyDeadline();

        // Then: Should be timestamp + 1 hour
        assertNotNull(deadline);
        long expectedDeadline = statLactateRecommendation.getTimestamp() + (60 * 60 * 1000);
        assertEquals(expectedDeadline, deadline);
    }

    @Test
    @DisplayName("Get Urgency Deadline: ROUTINE = 48 hours")
    void testGetUrgencyDeadline_Routine() {
        // When: Get urgency deadline for ROUTINE
        Long deadline = routineChestXRayRecommendation.getUrgencyDeadline();

        // Then: Should be timestamp + 48 hours
        assertNotNull(deadline);
        long expectedDeadline = routineChestXRayRecommendation.getTimestamp() + (48 * 60 * 60 * 1000);
        assertEquals(expectedDeadline, deadline);
    }

    // ============================================================
    // PREREQUISITE TESTS (2 tests)
    // ============================================================

    @Test
    @DisplayName("Prerequisites Met: All prerequisites completed")
    void testArePrerequisitesMet_AllCompleted() {
        // Given: Recommendation requiring creatinine
        TestRecommendation ctWithContrast = TestRecommendation.builder()
            .recommendationId("REC-CT-001")
            .prerequisiteTests(Arrays.asList("LAB-CREATININE-001", "LAB-PREGNANCY-001"))
            .build();

        // All prerequisites completed
        List<String> completedTests = Arrays.asList(
            "LAB-CREATININE-001",
            "LAB-PREGNANCY-001",
            "LAB-CBC-001"
        );

        // When: Check prerequisites
        boolean met = ctWithContrast.arePrerequisitesMet(completedTests);

        // Then: Should be met
        assertTrue(met);
    }

    @Test
    @DisplayName("Prerequisites Met: Missing prerequisite")
    void testArePrerequisitesMet_MissingPrerequisite() {
        // Given: Recommendation requiring multiple tests
        TestRecommendation recommendation = TestRecommendation.builder()
            .recommendationId("REC-001")
            .prerequisiteTests(Arrays.asList("LAB-CREATININE-001", "LAB-PREGNANCY-001"))
            .build();

        // Only one prerequisite completed
        List<String> completedTests = Arrays.asList("LAB-CREATININE-001");

        // When: Check prerequisites
        boolean met = recommendation.arePrerequisitesMet(completedTests);

        // Then: Should NOT be met
        assertFalse(met);
    }

    // ============================================================
    // HELPER METHOD TESTS (5 tests)
    // ============================================================

    @Test
    @DisplayName("Get Confidence Score: Returns decision support confidence")
    void testGetConfidenceScore() {
        // When: Get confidence score
        Double confidence = statLactateRecommendation.getConfidenceScore();

        // Then: Should return 0.95
        assertNotNull(confidence);
        assertEquals(0.95, confidence);
    }

    @Test
    @DisplayName("Is Lab Test: LAB category returns true")
    void testIsLabTest() {
        // Then: Lactate recommendation should be lab test
        assertTrue(statLactateRecommendation.isLabTest());
        assertFalse(statLactateRecommendation.isImagingStudy());
    }

    @Test
    @DisplayName("Is Imaging Study: IMAGING category returns true")
    void testIsImagingStudy() {
        // Then: Chest X-ray recommendation should be imaging
        assertTrue(routineChestXRayRecommendation.isImagingStudy());
        assertFalse(routineChestXRayRecommendation.isLabTest());
    }

    @Test
    @DisplayName("Get Priority Description: Human-readable text")
    void testGetPriorityDescription() {
        // When: Get priority description
        String description = statLactateRecommendation.getPriorityDescription();

        // Then: Should return readable description
        assertEquals("Critical - Life-threatening", description);

        // Test other priorities
        TestRecommendation p3 = TestRecommendation.builder()
            .priority(TestRecommendation.Priority.P3_ROUTINE)
            .build();
        assertEquals("Routine - Screening or monitoring", p3.getPriorityDescription());
    }

    @Test
    @DisplayName("Get Urgency Description: Human-readable text")
    void testGetUrgencyDescription() {
        // When: Get urgency description
        String description = statLactateRecommendation.getUrgencyDescription();

        // Then: Should return readable description
        assertEquals("STAT - Within 1 hour", description);

        // Test routine urgency
        assertEquals("Routine - Within 48 hours",
                    routineChestXRayRecommendation.getUrgencyDescription());
    }

    // ============================================================
    // HELPER METHODS - TEST DATA CREATION
    // ============================================================

    private TestRecommendation createSTATLactateRecommendation() {
        return TestRecommendation.builder()
            .recommendationId("REC-LACTATE-STAT-001")
            .testId("LAB-LACTATE-001")
            .testName("Serum Lactate")
            .testCategory(TestRecommendation.TestCategory.LAB)
            .timestamp(System.currentTimeMillis())
            .priority(TestRecommendation.Priority.P0_CRITICAL)
            .urgency(TestRecommendation.Urgency.STAT)
            .indication("Suspected septic shock")
            .rationale("Lactate >2 mmol/L indicates organ dysfunction in sepsis (SSC 2021)")
            .expectedFindings("Elevated lactate (>2.0 mmol/L) indicates tissue hypoperfusion")
            .interpretationGuidance("Lactate ≥4.0 mmol/L = septic shock, ≥2.0 mmol/L = organ dysfunction")
            .differentialDiagnosis(Arrays.asList(
                "Septic shock",
                "Cardiogenic shock",
                "Hypovolemic shock",
                "Lactic acidosis type B"
            ))
            .timeframeMinutes(60)
            .collectionTiming("STAT - within 15 minutes")
            .optimalTiming("Before antibiotic administration if possible")
            .repeatTest(false)
            .contraindications(new ArrayList<>())
            .warnings(Arrays.asList("Use gray-top tube", "Transport on ice", "Analyze within 15 minutes"))
            .prerequisiteTests(new ArrayList<>())
            .requiresConsent(false)
            .decisionSupport(TestRecommendation.DecisionSupport.builder()
                .guidelineReference("Surviving Sepsis Campaign 2021")
                .evidenceLevel("A")
                .recommendationStrength("Strong")
                .confidenceScore(0.95)
                .supportingEvidence(Arrays.asList("PMID: 34605781", "PMID: 21378355"))
                .clinicalReasoning("Elevated lactate is a key criterion for sepsis-induced organ dysfunction and septic shock definition")
                .build())
            .orderingInfo(TestRecommendation.OrderingInformation.builder()
                .orderCode("LAB-LAC")
                .loincCode("2524-7")
                .specimenType("Blood (venous or arterial)")
                .collectionInstructions("Gray-top tube (sodium fluoride), avoid prolonged tourniquet")
                .transportInstructions("Transport on ice, analyze within 15 minutes")
                .fastingRequired(false)
                .build())
            .alternatives(new ArrayList<>())
            .followUpGuidance(TestRecommendation.FollowUpGuidance.builder()
                .actionIfNormal("Continue sepsis evaluation if clinical suspicion remains")
                .actionIfAbnormal("Initiate sepsis resuscitation, repeat in 2-4 hours")
                .actionIfCritical("Septic shock - aggressive fluid resuscitation, vasopressors, ICU transfer")
                .reflexTests(Arrays.asList("Repeat lactate in 2 hours"))
                .repeatIntervalHours(2)
                .monitoringPlan("Serial lactate every 2-4 hours until clearing (target >10% clearance)")
                .build())
            .estimatedCost(45.0)
            .highUtilization(true)
            .patientId("ROHAN-001")
            .encounterId("ENC-001")
            .generatedBy("SEPSIS-PROTOCOL-001")
            .protocolId("SEPSIS-BUNDLE-001")
            .build();
    }

    private TestRecommendation createRoutineChestXRayRecommendation() {
        return TestRecommendation.builder()
            .recommendationId("REC-CXR-ROUTINE-001")
            .testId("IMG-CXR-001")
            .testName("Chest X-Ray (2-View)")
            .testCategory(TestRecommendation.TestCategory.IMAGING)
            .timestamp(System.currentTimeMillis())
            .priority(TestRecommendation.Priority.P2_IMPORTANT)
            .urgency(TestRecommendation.Urgency.ROUTINE)
            .indication("Suspected community-acquired pneumonia")
            .rationale("CXR is first-line imaging for pneumonia diagnosis (IDSA/ATS 2019)")
            .expectedFindings("Infiltrate or consolidation if bacterial pneumonia present")
            .interpretationGuidance("Lobar consolidation suggests bacterial pneumonia, diffuse interstitial pattern suggests viral/atypical")
            .differentialDiagnosis(Arrays.asList(
                "Community-acquired pneumonia",
                "Aspiration pneumonia",
                "Pulmonary edema",
                "Atelectasis"
            ))
            .timeframeMinutes(1440) // 24 hours
            .collectionTiming("Within 24 hours")
            .repeatTest(false)
            .contraindications(Arrays.asList("Pregnancy (relative contraindication)"))
            .warnings(Arrays.asList("Shield abdomen if pregnant", "Minimal radiation exposure"))
            .prerequisiteTests(new ArrayList<>())
            .requiresConsent(false)
            .decisionSupport(TestRecommendation.DecisionSupport.builder()
                .guidelineReference("IDSA/ATS Community-Acquired Pneumonia Guidelines 2019")
                .evidenceLevel("A")
                .recommendationStrength("Strong")
                .confidenceScore(0.90)
                .supportingEvidence(Arrays.asList("PMID: 31573350"))
                .clinicalReasoning("CXR is recommended for all patients with suspected pneumonia to confirm diagnosis and assess severity")
                .build())
            .orderingInfo(TestRecommendation.OrderingInformation.builder()
                .orderCode("IMG-CXR-2V")
                .cptCode("71046")
                .collectionInstructions("PA and lateral views, standing if able")
                .patientPreparation("Remove metal objects, jewelry from chest/neck")
                .fastingRequired(false)
                .build())
            .alternatives(Arrays.asList(
                TestRecommendation.TestAlternative.builder()
                    .testId("IMG-CT-CHEST-001")
                    .testName("CT Chest")
                    .reason("More sensitive for subtle infiltrates, but higher cost and radiation")
                    .betterAppropriate(false)
                    .lowerCost(false)
                    .fasterResult(false)
                    .lessSensitive(false)
                    .build()
            ))
            .followUpGuidance(TestRecommendation.FollowUpGuidance.builder()
                .actionIfNormal("Consider alternative diagnoses if clinical suspicion remains")
                .actionIfAbnormal("Initiate antibiotic therapy per CAP guidelines")
                .actionIfCritical("Consult pulmonology or critical care if severe findings")
                .reflexTests(new ArrayList<>())
                .repeatIntervalHours(null)
                .monitoringPlan("Repeat CXR in 6 weeks to confirm resolution (smokers, age >50)")
                .build())
            .estimatedCost(250.0)
            .highUtilization(true)
            .patientId("PATIENT-002")
            .encounterId("ENC-002")
            .generatedBy("PNEUMONIA-PROTOCOL-001")
            .protocolId("CAP-PROTOCOL-001")
            .build();
    }
}
