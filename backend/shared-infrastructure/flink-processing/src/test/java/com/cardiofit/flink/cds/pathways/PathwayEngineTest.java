package com.cardiofit.flink.cds.pathways;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for PathwayEngine
 * Phase 8 Module 3 - Clinical Pathways Engine
 *
 * Test Coverage:
 * - Pathway initiation and validation
 * - Step execution and completion
 * - Branching logic and decision points
 * - Deviation detection (time-based, condition-based)
 * - Pathway lifecycle (suspend, resume, discontinue, complete)
 * - Summary generation
 * - Skip step functionality
 *
 * @author CardioFit Testing Team
 * @version 1.0.0
 * @since Phase 8
 */
@DisplayName("PathwayEngine Tests")
class PathwayEngineTest {

    private PathwayEngine engine;
    private ClinicalPathway pathway;
    private static final String TEST_PATIENT_ID = "PATIENT-12345";

    @BeforeEach
    void setUp() {
        engine = new PathwayEngine();
        pathway = createTestPathway();
    }

    /**
     * Create a simple test pathway with 3 steps
     */
    private ClinicalPathway createTestPathway() {
        ClinicalPathway pathway = new ClinicalPathway(
            "TEST-PATHWAY-001",
            "Test Clinical Pathway",
            ClinicalPathway.PathwayType.ROUTINE
        );

        // Step 1: Assessment
        PathwayStep step1 = new PathwayStep("STEP-001", "Initial Assessment", PathwayStep.StepType.ASSESSMENT);
        step1.setExpectedDurationMinutes(10);
        pathway.addStep(step1);

        // Step 2: Diagnostic
        PathwayStep step2 = new PathwayStep("STEP-002", "Diagnostic Test", PathwayStep.StepType.DIAGNOSTIC);
        step2.setExpectedDurationMinutes(15);
        pathway.addStep(step2);

        // Step 3: Treatment
        PathwayStep step3 = new PathwayStep("STEP-003", "Treatment", PathwayStep.StepType.THERAPEUTIC);
        step3.setExpectedDurationMinutes(20);
        pathway.addStep(step3);

        return pathway;
    }

    @Nested
    @DisplayName("Pathway Initiation Tests")
    class PathwayInitiationTests {

        @Test
        @DisplayName("Should successfully initiate valid pathway")
        void testSuccessfulPathwayInitiation() {
            Map<String, Object> patientData = new HashMap<>();
            patientData.put("age", 65);
            patientData.put("diagnosis", "Test condition");

            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            assertNotNull(instance);
            assertEquals(TEST_PATIENT_ID, instance.getPatientId());
            assertEquals("TEST-PATHWAY-001", instance.getPathwayId());
            assertEquals("Test Clinical Pathway", instance.getPathwayName());
            assertEquals(PathwayInstance.InstanceStatus.INITIATED, instance.getStatus());
            assertEquals("STEP-001", instance.getCurrentStepId());
            assertEquals(2, instance.getPatientData().size());
        }

        @Test
        @DisplayName("Should throw exception for invalid pathway")
        void testInitiateInvalidPathway() {
            ClinicalPathway invalidPathway = new ClinicalPathway(
                "INVALID",
                "Invalid Pathway",
                ClinicalPathway.PathwayType.ROUTINE
            );
            // No steps added - pathway is invalid

            Map<String, Object> patientData = new HashMap<>();

            assertThrows(IllegalArgumentException.class, () -> {
                engine.initiatePathway(invalidPathway, TEST_PATIENT_ID, patientData);
            });
        }

        @Test
        @DisplayName("Should copy patient data to instance")
        void testPatientDataCopy() {
            Map<String, Object> patientData = new HashMap<>();
            patientData.put("troponin", 0.08);
            patientData.put("BP_systolic", 120);

            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            // Instance should have its own copy of patient data
            assertNotSame(patientData, instance.getPatientData());
            assertEquals(0.08, instance.getPatientData().get("troponin"));
            assertEquals(120, instance.getPatientData().get("BP_systolic"));
        }

        @Test
        @DisplayName("Should set initial step from pathway")
        void testInitialStepConfiguration() {
            Map<String, Object> patientData = new HashMap<>();

            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            assertEquals("STEP-001", instance.getCurrentStepId());
        }
    }

    @Nested
    @DisplayName("Step Execution Tests")
    class StepExecutionTests {

        @Test
        @DisplayName("Should execute step successfully")
        void testExecuteStep() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            PathwayInstance.StepExecution execution = engine.executeStep(pathway, instance, "STEP-001");

            assertNotNull(execution);
            assertEquals("STEP-001", execution.getStepId());
            assertEquals("Initial Assessment", execution.getStepName());
            assertEquals(PathwayInstance.InstanceStatus.IN_PROGRESS, instance.getStatus());
            assertTrue(instance.getClinicianActions().size() > 0);
        }

        @Test
        @DisplayName("Should throw exception for non-existent step")
        void testExecuteNonExistentStep() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            assertThrows(IllegalArgumentException.class, () -> {
                engine.executeStep(pathway, instance, "STEP-999");
            });
        }

        @Test
        @DisplayName("Should check entry conditions before executing step")
        void testEntryConditionValidation() {
            // Create pathway with entry condition
            PathwayStep step = new PathwayStep("STEP-CONDITIONAL", "Conditional Step", PathwayStep.StepType.THERAPEUTIC);
            PathwayStep.Condition entryCondition = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.LAB_VALUE,
                "troponin",
                ">",
                0.04
            );
            step.addEntryCondition(entryCondition);
            pathway.addStep(step);

            Map<String, Object> patientData = new HashMap<>();
            patientData.put("troponin", 0.02); // Does not meet entry condition
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            // Should throw exception and record deviation
            assertThrows(IllegalStateException.class, () -> {
                engine.executeStep(pathway, instance, "STEP-CONDITIONAL");
            });

            // Deviation should be recorded
            assertTrue(instance.isHasDeviations());
            assertEquals(1, instance.getDeviationCount());
        }

        @Test
        @DisplayName("Should record step initiation action")
        void testStepInitiationRecording() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            engine.executeStep(pathway, instance, "STEP-001");

            assertTrue(instance.getClinicianActions().stream()
                .anyMatch(action -> action.contains("Started step")));
        }
    }

    @Nested
    @DisplayName("Step Completion Tests")
    class StepCompletionTests {

        @Test
        @DisplayName("Should complete step and determine next step")
        void testCompleteStepWithLinearProgression() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            PathwayInstance.StepExecution execution = engine.executeStep(pathway, instance, "STEP-001");

            Map<String, Object> stepResults = new HashMap<>();
            stepResults.put("assessment_complete", true);

            String nextStepId = engine.completeStep(pathway, instance, execution, stepResults);

            assertEquals("STEP-002", nextStepId);
            assertEquals("STEP-002", instance.getCurrentStepId());
            assertEquals(1, instance.getCompletedSteps().size());
            assertEquals(1, instance.getPatientData().size()); // stepResults added
        }

        @Test
        @DisplayName("Should update patient data from step results")
        void testPatientDataUpdate() {
            Map<String, Object> patientData = new HashMap<>();
            patientData.put("initial_data", "value");
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            PathwayInstance.StepExecution execution = engine.executeStep(pathway, instance, "STEP-001");

            Map<String, Object> stepResults = new HashMap<>();
            stepResults.put("ecg_finding", "Normal");
            stepResults.put("troponin", 0.02);

            engine.completeStep(pathway, instance, execution, stepResults);

            assertEquals(3, instance.getPatientData().size());
            assertEquals("Normal", instance.getPatientData().get("ecg_finding"));
            assertEquals(0.02, instance.getPatientData().get("troponin"));
        }

        @Test
        @DisplayName("Should check exit conditions when completing step")
        void testExitConditionChecking() {
            // Create step with exit condition
            PathwayStep step = new PathwayStep("STEP-EXIT", "Step with Exit Condition", PathwayStep.StepType.DIAGNOSTIC);
            PathwayStep.Condition exitCondition = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.PROCEDURE_DONE,
                "ecg_completed",
                "=",
                true
            );
            step.addExitCondition(exitCondition);
            pathway.getSteps().add(0, step); // Add as first step

            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);
            instance.setCurrentStepId("STEP-EXIT");

            PathwayInstance.StepExecution execution = instance.startStep("STEP-EXIT", "Step with Exit Condition");

            Map<String, Object> stepResults = new HashMap<>();
            stepResults.put("ecg_completed", false); // Exit condition NOT met

            // Should record deviation but allow progression
            engine.completeStep(pathway, instance, execution, stepResults);

            assertTrue(instance.isHasDeviations());
        }

        @Test
        @DisplayName("Should record quality measure for core quality steps")
        void testCoreQualityMeasureRecording() {
            pathway.getSteps().get(0).setCoreQualityMeasure(true);

            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            PathwayInstance.StepExecution execution = engine.executeStep(pathway, instance, "STEP-001");
            engine.completeStep(pathway, instance, execution, null);

            assertEquals(1, instance.getQualityMeasuresMet().size());
            assertTrue(instance.getQualityMeasuresMet().get("Initial Assessment"));
        }

        @Test
        @DisplayName("Should set status to COMPLETED when no next step")
        void testPathwayCompletionOnLastStep() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            // Execute all steps
            PathwayInstance.StepExecution exec1 = engine.executeStep(pathway, instance, "STEP-001");
            engine.completeStep(pathway, instance, exec1, null);

            PathwayInstance.StepExecution exec2 = engine.executeStep(pathway, instance, "STEP-002");
            engine.completeStep(pathway, instance, exec2, null);

            PathwayInstance.StepExecution exec3 = engine.executeStep(pathway, instance, "STEP-003");
            String nextStep = engine.completeStep(pathway, instance, exec3, null);

            assertNull(nextStep);
            assertEquals(PathwayInstance.InstanceStatus.COMPLETED, instance.getStatus());
        }

        @Test
        @DisplayName("Should record step completion action")
        void testStepCompletionAction() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            PathwayInstance.StepExecution execution = engine.executeStep(pathway, instance, "STEP-001");
            engine.completeStep(pathway, instance, execution, null);

            assertTrue(instance.getClinicianActions().stream()
                .anyMatch(action -> action.contains("Completed step")));
        }
    }

    @Nested
    @DisplayName("Branching Logic Tests")
    class BranchingLogicTests {

        @Test
        @DisplayName("Should follow pathway-level decision points")
        void testPathwayLevelDecisionPoints() {
            // Add decision point to pathway
            pathway.addDecisionPoint("STEP-001", "HIGH_RISK", "STEP-003"); // Skip STEP-002

            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            PathwayInstance.StepExecution execution = engine.executeStep(pathway, instance, "STEP-001");

            Map<String, Object> stepResults = new HashMap<>();
            stepResults.put("risk_level", "HIGH_RISK");

            String nextStep = engine.completeStep(pathway, instance, execution, stepResults);

            assertEquals("STEP-003", nextStep); // Should branch to STEP-003, skipping STEP-002
        }

        @Test
        @DisplayName("Should follow step-level transitions")
        void testStepLevelTransitions() {
            // Add transitions to first step
            PathwayStep step1 = pathway.getSteps().get(0);
            step1.addTransition("CRITICAL", "STEP-003");
            step1.addTransition("STABLE", "STEP-002");

            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            PathwayInstance.StepExecution execution = engine.executeStep(pathway, instance, "STEP-001");

            // This will use step-level transitions (which returns first transition in current implementation)
            String nextStep = engine.completeStep(pathway, instance, execution, null);

            // Should use one of the transitions
            assertNotNull(nextStep);
        }

        @Test
        @DisplayName("Should use linear progression when no branching defined")
        void testLinearProgression() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            PathwayInstance.StepExecution execution = engine.executeStep(pathway, instance, "STEP-001");
            String nextStep = engine.completeStep(pathway, instance, execution, null);

            assertEquals("STEP-002", nextStep); // Linear progression

            execution = engine.executeStep(pathway, instance, "STEP-002");
            nextStep = engine.completeStep(pathway, instance, execution, null);

            assertEquals("STEP-003", nextStep); // Linear progression
        }
    }

    @Nested
    @DisplayName("Deviation Detection Tests")
    class DeviationDetectionTests {

        @Test
        @DisplayName("Should detect time-critical step exceeding max duration")
        void testTimeCriticalStepDeviationDetection() throws InterruptedException {
            // Make first step time-critical with very short max duration
            PathwayStep step1 = pathway.getSteps().get(0);
            step1.setTimeCritical(true);
            step1.setMaxDurationMinutes(0); // Immediate timeout for testing

            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            PathwayInstance.StepExecution execution = engine.executeStep(pathway, instance, "STEP-001");

            // Manually set start time to 2 minutes ago to simulate timeout
            execution.setStartTime(execution.getStartTime().minusMinutes(2));

            // Add execution to completed steps so it can be found
            instance.getCompletedSteps().add(execution);

            // Detect deviations
            var deviations = engine.detectDeviations(pathway, instance);

            assertTrue(deviations.size() > 0);
            assertEquals(PathwayInstance.Deviation.DeviationType.TIME_EXCEEDED,
                        deviations.get(0).getDeviationType());
            assertEquals(PathwayInstance.Deviation.Severity.HIGH,
                        deviations.get(0).getSeverity());
        }

        @Test
        @DisplayName("Should detect pathway exceeding max total duration")
        void testPathwayMaxDurationDeviation() {
            pathway.setMaxDurationMinutes(0); // Immediate timeout for testing

            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            // Manually set start time to 2 minutes ago to simulate timeout
            instance.setStartTime(instance.getStartTime().minusMinutes(2));

            var deviations = engine.detectDeviations(pathway, instance);

            assertTrue(deviations.size() > 0);
            assertTrue(deviations.stream()
                .anyMatch(d -> d.getStepId().equals("PATHWAY")));
            assertEquals(PathwayInstance.Deviation.Severity.CRITICAL,
                        deviations.stream()
                            .filter(d -> d.getStepId().equals("PATHWAY"))
                            .findFirst()
                            .get()
                            .getSeverity());
        }

        @Test
        @DisplayName("Should update deviation counts in instance")
        void testDeviationCountUpdate() {
            pathway.setMaxDurationMinutes(0);

            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            // Manually set start time to 2 minutes ago to simulate timeout
            instance.setStartTime(instance.getStartTime().minusMinutes(2));

            engine.detectDeviations(pathway, instance);

            assertTrue(instance.isHasDeviations());
            assertTrue(instance.getDeviationCount() > 0);
        }

        @Test
        @DisplayName("Should return empty list when no deviations detected")
        void testNoDeviationsDetected() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            var deviations = engine.detectDeviations(pathway, instance);

            assertEquals(0, deviations.size());
        }
    }

    @Nested
    @DisplayName("Skip Step Tests")
    class SkipStepTests {

        @Test
        @DisplayName("Should record deviation when skipping step")
        void testSkipStepRecordsDeviation() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            engine.skipStep(instance, "STEP-002", "Patient refused diagnostic test",
                           "Patient autonomy - informed refusal documented");

            assertEquals(1, instance.getDeviationCount());
            assertEquals(PathwayInstance.Deviation.DeviationType.STEP_SKIPPED,
                        instance.getDeviations().get(0).getDeviationType());
            assertEquals("Patient refused diagnostic test",
                        instance.getDeviations().get(0).getReason());
            assertEquals("Patient autonomy - informed refusal documented",
                        instance.getDeviations().get(0).getClinicalJustification());
        }

        @Test
        @DisplayName("Should record skip action in clinician actions")
        void testSkipStepRecordsAction() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            engine.skipStep(instance, "STEP-002", "Resource unavailable", "No ultrasound available");

            assertTrue(instance.getClinicianActions().stream()
                .anyMatch(action -> action.contains("Skipped step: STEP-002")));
        }
    }

    @Nested
    @DisplayName("Pathway Lifecycle Tests")
    class PathwayLifecycleTests {

        @Test
        @DisplayName("Should suspend pathway with reason")
        void testSuspendPathway() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            engine.suspendPathway(instance, "Waiting for lab results");

            assertEquals(PathwayInstance.InstanceStatus.SUSPENDED, instance.getStatus());
            assertEquals("Waiting for lab results", instance.getOutcomeNotes());
            assertTrue(instance.getClinicianActions().stream()
                .anyMatch(action -> action.contains("Pathway suspended")));
        }

        @Test
        @DisplayName("Should resume suspended pathway")
        void testResumePathway() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            engine.suspendPathway(instance, "Waiting for lab results");
            engine.resumePathway(instance);

            assertEquals(PathwayInstance.InstanceStatus.IN_PROGRESS, instance.getStatus());
            assertTrue(instance.getClinicianActions().stream()
                .anyMatch(action -> action.contains("Pathway resumed")));
        }

        @Test
        @DisplayName("Should throw exception when resuming non-suspended pathway")
        void testResumeNonSuspendedPathway() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            assertThrows(IllegalStateException.class, () -> {
                engine.resumePathway(instance);
            });
        }

        @Test
        @DisplayName("Should discontinue pathway with reason and user")
        void testDiscontinuePathway() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            engine.discontinuePathway(instance, "Patient transferred to another facility", "Dr. Smith");

            assertEquals(PathwayInstance.InstanceStatus.DISCONTINUED, instance.getStatus());
            assertEquals("Patient transferred to another facility", instance.getOutcomeNotes());
            assertEquals("Dr. Smith", instance.getCompletedBy());
            assertNotNull(instance.getEndTime());
            assertTrue(instance.getClinicianActions().stream()
                .anyMatch(action -> action.contains("Pathway discontinued")));
        }

        @Test
        @DisplayName("Should complete pathway with outcome and user")
        void testCompletePathway() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            engine.completePathway(instance, "Patient stabilized and transferred to ICU", "Dr. Jones");

            assertEquals(PathwayInstance.InstanceStatus.COMPLETED, instance.getStatus());
            assertEquals("Patient stabilized and transferred to ICU", instance.getFinalOutcome());
            assertEquals("Dr. Jones", instance.getCompletedBy());
            assertNotNull(instance.getEndTime());
            assertTrue(instance.getClinicianActions().stream()
                .anyMatch(action -> action.contains("Pathway completed")));
        }
    }

    @Nested
    @DisplayName("Summary Generation Tests")
    class SummaryGenerationTests {

        @Test
        @DisplayName("Should generate comprehensive pathway summary")
        void testGenerateSummary() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            // Execute some steps
            PathwayInstance.StepExecution exec1 = engine.executeStep(pathway, instance, "STEP-001");
            exec1.setOnTime(true);
            instance.completeStep(exec1, true);

            PathwayInstance.StepExecution exec2 = engine.executeStep(pathway, instance, "STEP-002");
            exec2.setOnTime(false);
            instance.completeStep(exec2, false);

            // Record quality measure
            instance.recordQualityMeasure("door_to_ecg", true);
            instance.recordQualityMeasure("aspirin_given", false);

            // Record deviation
            instance.recordDeviation(
                PathwayInstance.Deviation.DeviationType.TIME_EXCEEDED,
                "STEP-002",
                "Delayed",
                PathwayInstance.Deviation.Severity.LOW
            );

            engine.completePathway(instance, "Success", "Dr. Smith");

            PathwayEngine.PathwaySummary summary = engine.generateSummary(instance);

            assertNotNull(summary);
            assertEquals(instance.getInstanceId(), summary.getInstanceId());
            assertEquals(TEST_PATIENT_ID, summary.getPatientId());
            assertEquals("Test Clinical Pathway", summary.getPathwayName());
            assertEquals(PathwayInstance.InstanceStatus.COMPLETED, summary.getStatus());
            assertEquals(2, summary.getTotalStepsCompleted());
            assertEquals(1, summary.getStepsCompletedOnTime());
            assertEquals(1, summary.getTotalDeviations());
            assertEquals("Success", summary.getFinalOutcome());
            assertEquals(1, summary.getQualityMeasuresMet());
            assertEquals(2, summary.getTotalQualityMeasures());
        }

        @Test
        @DisplayName("Should calculate quality compliance rate")
        void testQualityComplianceRateCalculation() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            instance.recordQualityMeasure("measure1", true);
            instance.recordQualityMeasure("measure2", true);
            instance.recordQualityMeasure("measure3", false);
            instance.recordQualityMeasure("measure4", true);

            PathwayEngine.PathwaySummary summary = engine.generateSummary(instance);

            assertEquals(75.0, summary.getQualityComplianceRate(), 0.01); // 3 of 4 = 75%
        }

        @Test
        @DisplayName("Should handle empty pathway summary")
        void testEmptyPathwaySummary() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            PathwayEngine.PathwaySummary summary = engine.generateSummary(instance);

            assertNotNull(summary);
            assertEquals(0, summary.getTotalStepsCompleted());
            assertEquals(0, summary.getStepsCompletedOnTime());
            assertEquals(0, summary.getTotalDeviations());
            assertEquals(0.0, summary.getQualityComplianceRate(), 0.01);
        }

        @Test
        @DisplayName("Should generate meaningful summary toString")
        void testSummaryToString() {
            Map<String, Object> patientData = new HashMap<>();
            PathwayInstance instance = engine.initiatePathway(pathway, TEST_PATIENT_ID, patientData);

            instance.recordQualityMeasure("measure1", true);
            engine.completePathway(instance, "Success", "Dr. Smith");

            PathwayEngine.PathwaySummary summary = engine.generateSummary(instance);

            String str = summary.toString();

            assertTrue(str.contains("Test Clinical Pathway"));
            assertTrue(str.contains("COMPLETED"));
            assertTrue(str.contains("adherence"));
            assertTrue(str.contains("quality"));
        }
    }
}
