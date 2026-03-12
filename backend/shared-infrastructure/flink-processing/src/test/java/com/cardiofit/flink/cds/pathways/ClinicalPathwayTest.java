package com.cardiofit.flink.cds.pathways;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for ClinicalPathway
 * Phase 8 Module 3 - Clinical Pathways Engine
 *
 * Test Coverage:
 * - Pathway creation and configuration
 * - Step management and ordering
 * - Decision point branching logic
 * - Validation rules
 * - Inclusion/exclusion criteria
 *
 * @author CardioFit Testing Team
 * @version 1.0.0
 * @since Phase 8
 */
@DisplayName("ClinicalPathway Tests")
class ClinicalPathwayTest {

    private ClinicalPathway pathway;
    private static final String TEST_PATHWAY_ID = "TEST-PATHWAY-001";
    private static final String TEST_PATHWAY_NAME = "Test Clinical Pathway";

    @BeforeEach
    void setUp() {
        pathway = new ClinicalPathway(
            TEST_PATHWAY_ID,
            TEST_PATHWAY_NAME,
            ClinicalPathway.PathwayType.EMERGENCY
        );
    }

    @Nested
    @DisplayName("Core Pathway Creation Tests")
    class CoreCreationTests {

        @Test
        @DisplayName("Should create pathway with required fields")
        void testPathwayCreation() {
            assertNotNull(pathway);
            assertEquals(TEST_PATHWAY_ID, pathway.getPathwayId());
            assertEquals(TEST_PATHWAY_NAME, pathway.getPathwayName());
            assertEquals(ClinicalPathway.PathwayType.EMERGENCY, pathway.getPathwayType());
            assertTrue(pathway.isActive());
            assertNotNull(pathway.getCreatedDate());
        }

        @Test
        @DisplayName("Should initialize empty collections")
        void testCollectionInitialization() {
            assertNotNull(pathway.getSteps());
            assertNotNull(pathway.getDecisionPoints());
            assertNotNull(pathway.getApplicableDiagnoses());
            assertNotNull(pathway.getInclusionCriteria());
            assertNotNull(pathway.getExclusionCriteria());
            assertNotNull(pathway.getQualityMetrics());
            assertNotNull(pathway.getExpectedOutcomes());

            assertTrue(pathway.getSteps().isEmpty());
            assertTrue(pathway.getDecisionPoints().isEmpty());
        }

        @Test
        @DisplayName("Should support all pathway types")
        void testAllPathwayTypes() {
            ClinicalPathway.PathwayType[] types = ClinicalPathway.PathwayType.values();

            assertEquals(8, types.length);
            assertTrue(containsType(types, ClinicalPathway.PathwayType.EMERGENCY));
            assertTrue(containsType(types, ClinicalPathway.PathwayType.URGENT));
            assertTrue(containsType(types, ClinicalPathway.PathwayType.ROUTINE));
            assertTrue(containsType(types, ClinicalPathway.PathwayType.CHRONIC_MANAGEMENT));
        }

        private boolean containsType(ClinicalPathway.PathwayType[] types, ClinicalPathway.PathwayType target) {
            for (ClinicalPathway.PathwayType type : types) {
                if (type == target) return true;
            }
            return false;
        }
    }

    @Nested
    @DisplayName("Step Management Tests")
    class StepManagementTests {

        @Test
        @DisplayName("Should add steps to pathway")
        void testAddStep() {
            PathwayStep step1 = new PathwayStep("STEP-1", "First Step", PathwayStep.StepType.ASSESSMENT);
            PathwayStep step2 = new PathwayStep("STEP-2", "Second Step", PathwayStep.StepType.DIAGNOSTIC);

            pathway.addStep(step1);
            pathway.addStep(step2);

            assertEquals(2, pathway.getSteps().size());
            assertEquals("STEP-1", pathway.getSteps().get(0).getStepId());
            assertEquals("STEP-2", pathway.getSteps().get(1).getStepId());
        }

        @Test
        @DisplayName("Should set initial step automatically for first added step")
        void testAutomaticInitialStep() {
            PathwayStep step1 = new PathwayStep("STEP-1", "First Step", PathwayStep.StepType.ASSESSMENT);
            pathway.addStep(step1);

            assertEquals("STEP-1", pathway.getInitialStepId());
        }

        @Test
        @DisplayName("Should find step by ID")
        void testFindStep() {
            PathwayStep step1 = new PathwayStep("STEP-1", "First Step", PathwayStep.StepType.ASSESSMENT);
            PathwayStep step2 = new PathwayStep("STEP-2", "Second Step", PathwayStep.StepType.DIAGNOSTIC);

            pathway.addStep(step1);
            pathway.addStep(step2);

            PathwayStep found = pathway.findStep("STEP-2");
            assertNotNull(found);
            assertEquals("Second Step", found.getStepName());
        }

        @Test
        @DisplayName("Should return null for non-existent step")
        void testFindNonExistentStep() {
            PathwayStep found = pathway.findStep("NON-EXISTENT");
            assertNull(found);
        }

        @Test
        @DisplayName("Should get steps by type")
        void testGetStepsByType() {
            PathwayStep step1 = new PathwayStep("STEP-1", "Assessment", PathwayStep.StepType.ASSESSMENT);
            PathwayStep step2 = new PathwayStep("STEP-2", "Diagnostic", PathwayStep.StepType.DIAGNOSTIC);
            PathwayStep step3 = new PathwayStep("STEP-3", "Another Assessment", PathwayStep.StepType.ASSESSMENT);

            pathway.addStep(step1);
            pathway.addStep(step2);
            pathway.addStep(step3);

            var assessmentSteps = pathway.getStepsByType(PathwayStep.StepType.ASSESSMENT);
            assertEquals(2, assessmentSteps.size());

            var diagnosticSteps = pathway.getStepsByType(PathwayStep.StepType.DIAGNOSTIC);
            assertEquals(1, diagnosticSteps.size());
        }

        @Test
        @DisplayName("Should get critical time-sensitive steps")
        void testGetCriticalSteps() {
            PathwayStep step1 = new PathwayStep("STEP-1", "Critical", PathwayStep.StepType.THERAPEUTIC);
            step1.setTimeCritical(true);

            PathwayStep step2 = new PathwayStep("STEP-2", "Non-Critical", PathwayStep.StepType.ASSESSMENT);
            step2.setTimeCritical(false);

            PathwayStep step3 = new PathwayStep("STEP-3", "Also Critical", PathwayStep.StepType.MEDICATION);
            step3.setTimeCritical(true);

            pathway.addStep(step1);
            pathway.addStep(step2);
            pathway.addStep(step3);

            var criticalSteps = pathway.getCriticalSteps();
            assertEquals(2, criticalSteps.size());
            assertTrue(criticalSteps.stream().allMatch(PathwayStep::isTimeCritical));
        }
    }

    @Nested
    @DisplayName("Decision Point Tests")
    class DecisionPointTests {

        @Test
        @DisplayName("Should add decision points for branching")
        void testAddDecisionPoint() {
            pathway.addDecisionPoint("STEP-1", "HIGH_RISK", "STEP-2A");
            pathway.addDecisionPoint("STEP-1", "LOW_RISK", "STEP-2B");

            assertEquals(2, pathway.getDecisionPoints().size());
        }

        @Test
        @DisplayName("Should get next step based on condition")
        void testGetNextStep() {
            pathway.addDecisionPoint("STEP-1", "HIGH_RISK", "STEP-2A");
            pathway.addDecisionPoint("STEP-1", "LOW_RISK", "STEP-2B");

            String nextStepHighRisk = pathway.getNextStep("STEP-1", "HIGH_RISK");
            assertEquals("STEP-2A", nextStepHighRisk);

            String nextStepLowRisk = pathway.getNextStep("STEP-1", "LOW_RISK");
            assertEquals("STEP-2B", nextStepLowRisk);
        }

        @Test
        @DisplayName("Should return null for non-existent decision path")
        void testGetNextStepNonExistent() {
            pathway.addDecisionPoint("STEP-1", "HIGH_RISK", "STEP-2A");

            String nextStep = pathway.getNextStep("STEP-1", "MEDIUM_RISK");
            assertNull(nextStep);
        }
    }

    @Nested
    @DisplayName("Duration Calculation Tests")
    class DurationTests {

        @Test
        @DisplayName("Should calculate total expected duration from all steps")
        void testCalculateTotalExpectedDuration() {
            PathwayStep step1 = new PathwayStep("STEP-1", "Step 1", PathwayStep.StepType.ASSESSMENT);
            step1.setExpectedDurationMinutes(10);

            PathwayStep step2 = new PathwayStep("STEP-2", "Step 2", PathwayStep.StepType.DIAGNOSTIC);
            step2.setExpectedDurationMinutes(15);

            PathwayStep step3 = new PathwayStep("STEP-3", "Step 3", PathwayStep.StepType.THERAPEUTIC);
            step3.setExpectedDurationMinutes(30);

            pathway.addStep(step1);
            pathway.addStep(step2);
            pathway.addStep(step3);

            int totalDuration = pathway.calculateTotalExpectedDuration();
            assertEquals(55, totalDuration);
        }

        @Test
        @DisplayName("Should handle steps with null duration")
        void testCalculateDurationWithNulls() {
            PathwayStep step1 = new PathwayStep("STEP-1", "Step 1", PathwayStep.StepType.ASSESSMENT);
            step1.setExpectedDurationMinutes(10);

            PathwayStep step2 = new PathwayStep("STEP-2", "Step 2", PathwayStep.StepType.DIAGNOSTIC);
            step2.setExpectedDurationMinutes(null); // No duration set

            pathway.addStep(step1);
            pathway.addStep(step2);

            int totalDuration = pathway.calculateTotalExpectedDuration();
            assertEquals(10, totalDuration);
        }

        @Test
        @DisplayName("Should return zero for empty pathway")
        void testCalculateDurationEmpty() {
            int totalDuration = pathway.calculateTotalExpectedDuration();
            assertEquals(0, totalDuration);
        }
    }

    @Nested
    @DisplayName("Validation Tests")
    class ValidationTests {

        @Test
        @DisplayName("Should validate pathway with valid configuration")
        void testValidPathway() {
            PathwayStep step1 = new PathwayStep("STEP-1", "Step 1", PathwayStep.StepType.ASSESSMENT);
            pathway.addStep(step1);
            pathway.setInitialStepId("STEP-1");

            assertTrue(pathway.validate());
        }

        @Test
        @DisplayName("Should fail validation if no steps")
        void testValidationFailsNoSteps() {
            assertFalse(pathway.validate());
        }

        @Test
        @DisplayName("Should fail validation if initial step not set")
        void testValidationFailsNoInitialStep() {
            PathwayStep step1 = new PathwayStep("STEP-1", "Step 1", PathwayStep.StepType.ASSESSMENT);
            pathway.addStep(step1);
            pathway.setInitialStepId(null);

            assertFalse(pathway.validate());
        }

        @Test
        @DisplayName("Should fail validation if initial step doesn't exist")
        void testValidationFailsInvalidInitialStep() {
            PathwayStep step1 = new PathwayStep("STEP-1", "Step 1", PathwayStep.StepType.ASSESSMENT);
            pathway.addStep(step1);
            pathway.setInitialStepId("NON-EXISTENT");

            assertFalse(pathway.validate());
        }

        @Test
        @DisplayName("Should fail validation if step has no ID")
        void testValidationFailsStepNoId() {
            PathwayStep step1 = new PathwayStep(null, "Step 1", PathwayStep.StepType.ASSESSMENT);
            pathway.addStep(step1);

            assertFalse(pathway.validate());
        }

        @Test
        @DisplayName("Should fail validation if decision point references non-existent step")
        void testValidationFailsInvalidDecisionPoint() {
            PathwayStep step1 = new PathwayStep("STEP-1", "Step 1", PathwayStep.StepType.ASSESSMENT);
            pathway.addStep(step1);
            pathway.setInitialStepId("STEP-1");

            pathway.addDecisionPoint("STEP-1", "CONDITION", "NON-EXISTENT-STEP");

            assertFalse(pathway.validate());
        }
    }

    @Nested
    @DisplayName("Clinical Criteria Tests")
    class ClinicalCriteriaTests {

        @Test
        @DisplayName("Should add applicable diagnoses")
        void testAddApplicableDiagnoses() {
            pathway.addApplicableDiagnosis("I21.0"); // STEMI
            pathway.addApplicableDiagnosis("I21.4"); // NSTEMI

            assertEquals(2, pathway.getApplicableDiagnoses().size());
            assertTrue(pathway.getApplicableDiagnoses().contains("I21.0"));
        }

        @Test
        @DisplayName("Should add inclusion criteria")
        void testAddInclusionCriteria() {
            pathway.addInclusionCriterion("Chest pain within 24 hours");
            pathway.addInclusionCriterion("Age >= 18 years");

            assertEquals(2, pathway.getInclusionCriteria().size());
        }

        @Test
        @DisplayName("Should add exclusion criteria")
        void testAddExclusionCriteria() {
            pathway.addExclusionCriterion("Comfort care only");
            pathway.addExclusionCriterion("Active DNR status");

            assertEquals(2, pathway.getExclusionCriteria().size());
        }

        @Test
        @DisplayName("Should meet inclusion criteria when none defined")
        void testMeetsInclusionNoCriteria() {
            Map<String, Object> patientData = new HashMap<>();
            assertTrue(pathway.meetsInclusionCriteria(patientData));
        }

        @Test
        @DisplayName("Should not meet exclusion criteria when none defined")
        void testMeetsExclusionNoCriteria() {
            Map<String, Object> patientData = new HashMap<>();
            assertFalse(pathway.meetsExclusionCriteria(patientData));
        }
    }

    @Nested
    @DisplayName("Quality and Outcome Tests")
    class QualityOutcomeTests {

        @Test
        @DisplayName("Should add quality metrics")
        void testAddQualityMetric() {
            pathway.addQualityMetric("door_to_ecg_10min", true);
            pathway.addQualityMetric("aspirin_given", false);

            assertEquals(2, pathway.getQualityMetrics().size());
            assertTrue((Boolean) pathway.getQualityMetrics().get("door_to_ecg_10min"));
        }

        @Test
        @DisplayName("Should add expected outcomes")
        void testAddExpectedOutcomes() {
            pathway.addExpectedOutcome("Symptom resolution");
            pathway.addExpectedOutcome("Hemodynamic stability");

            assertEquals(2, pathway.getExpectedOutcomes().size());
        }

        @Test
        @DisplayName("Should add historical success rates")
        void testAddHistoricalSuccessRates() {
            pathway.addHistoricalSuccessRate("Survival", 0.92);
            pathway.addHistoricalSuccessRate("Readmission_avoided", 0.85);

            assertEquals(2, pathway.getHistoricalSuccessRates().size());
            assertEquals(0.92, pathway.getHistoricalSuccessRates().get("Survival"), 0.001);
        }

        @Test
        @DisplayName("Should add critical time points")
        void testAddCriticalTimePoints() {
            pathway.addCriticalTimePoint("STEP-ECG");
            pathway.addCriticalTimePoint("STEP-ANTIBIOTICS");

            assertEquals(2, pathway.getCriticalTimePoints().size());
            assertTrue(pathway.getCriticalTimePoints().contains("STEP-ECG"));
        }
    }

    @Nested
    @DisplayName("toString Tests")
    class ToStringTests {

        @Test
        @DisplayName("Should generate meaningful string representation")
        void testToString() {
            PathwayStep step1 = new PathwayStep("STEP-1", "Step 1", PathwayStep.StepType.ASSESSMENT);
            pathway.addStep(step1);
            pathway.setClinicalGuideline("AHA 2021");

            String str = pathway.toString();

            assertTrue(str.contains(TEST_PATHWAY_ID));
            assertTrue(str.contains(TEST_PATHWAY_NAME));
            assertTrue(str.contains("EMERGENCY"));
            assertTrue(str.contains("steps=1"));
            assertTrue(str.contains("AHA 2021"));
        }
    }
}
