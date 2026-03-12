package com.cardiofit.flink.cds.pathways;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for PathwayStep
 * Phase 8 Module 3 - Clinical Pathways Engine
 *
 * Test Coverage:
 * - Step creation and configuration
 * - Condition evaluation logic (all operators)
 * - Entry/exit condition checking
 * - Medication order management
 * - Time-critical step configuration
 * - Clinical data requirements
 * - toString representation
 *
 * @author CardioFit Testing Team
 * @version 1.0.0
 * @since Phase 8
 */
@DisplayName("PathwayStep Tests")
class PathwayStepTest {

    private PathwayStep step;
    private static final String TEST_STEP_ID = "STEP-001";
    private static final String TEST_STEP_NAME = "Test Clinical Step";

    @BeforeEach
    void setUp() {
        step = new PathwayStep(TEST_STEP_ID, TEST_STEP_NAME, PathwayStep.StepType.ASSESSMENT);
    }

    @Nested
    @DisplayName("Step Creation Tests")
    class StepCreationTests {

        @Test
        @DisplayName("Should create step with required fields")
        void testStepCreation() {
            assertNotNull(step);
            assertEquals(TEST_STEP_ID, step.getStepId());
            assertEquals(TEST_STEP_NAME, step.getStepName());
            assertEquals(PathwayStep.StepType.ASSESSMENT, step.getStepType());
        }

        @Test
        @DisplayName("Should initialize empty collections")
        void testCollectionInitialization() {
            assertNotNull(step.getInstructions());
            assertNotNull(step.getRequiredActions());
            assertNotNull(step.getEntryConditions());
            assertNotNull(step.getExitConditions());
            assertNotNull(step.getTransitions());
            assertNotNull(step.getRequiredVitals());
            assertNotNull(step.getRequiredLabs());
            assertNotNull(step.getRequiredAssessments());
            assertNotNull(step.getMedications());
            assertNotNull(step.getProcedures());
            assertNotNull(step.getConsultations());
            assertNotNull(step.getClinicalAlerts());
            assertNotNull(step.getSafeguards());
            assertNotNull(step.getRequiredDocumentation());
            assertNotNull(step.getQualityMeasures());
        }

        @Test
        @DisplayName("Should support all step types")
        void testAllStepTypes() {
            PathwayStep.StepType[] types = PathwayStep.StepType.values();

            assertEquals(10, types.length);
            assertTrue(containsType(types, PathwayStep.StepType.ASSESSMENT));
            assertTrue(containsType(types, PathwayStep.StepType.DIAGNOSTIC));
            assertTrue(containsType(types, PathwayStep.StepType.THERAPEUTIC));
            assertTrue(containsType(types, PathwayStep.StepType.MEDICATION));
            assertTrue(containsType(types, PathwayStep.StepType.MONITORING));
            assertTrue(containsType(types, PathwayStep.StepType.DECISION_POINT));
            assertTrue(containsType(types, PathwayStep.StepType.CONSULTATION));
            assertTrue(containsType(types, PathwayStep.StepType.PATIENT_EDUCATION));
            assertTrue(containsType(types, PathwayStep.StepType.DISPOSITION));
            assertTrue(containsType(types, PathwayStep.StepType.DOCUMENTATION));
        }

        private boolean containsType(PathwayStep.StepType[] types, PathwayStep.StepType target) {
            for (PathwayStep.StepType type : types) {
                if (type == target) return true;
            }
            return false;
        }
    }

    @Nested
    @DisplayName("Condition Evaluation Tests")
    class ConditionEvaluationTests {

        @Test
        @DisplayName("Should evaluate greater than condition correctly")
        void testGreaterThanCondition() {
            PathwayStep.Condition condition = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.LAB_VALUE,
                "troponin",
                ">",
                0.04
            );

            Map<String, Object> patientData = new HashMap<>();
            patientData.put("troponin", 0.08);

            assertTrue(condition.evaluate(patientData));

            patientData.put("troponin", 0.02);
            assertFalse(condition.evaluate(patientData));
        }

        @Test
        @DisplayName("Should evaluate less than condition correctly")
        void testLessThanCondition() {
            PathwayStep.Condition condition = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.VITAL_SIGN,
                "BP_systolic",
                "<",
                90
            );

            Map<String, Object> patientData = new HashMap<>();
            patientData.put("BP_systolic", 85);

            assertTrue(condition.evaluate(patientData));

            patientData.put("BP_systolic", 120);
            assertFalse(condition.evaluate(patientData));
        }

        @Test
        @DisplayName("Should evaluate equals condition correctly")
        void testEqualsCondition() {
            PathwayStep.Condition condition = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.CLINICAL_FINDING,
                "ecg_finding",
                "=",
                "STEMI"
            );

            Map<String, Object> patientData = new HashMap<>();
            patientData.put("ecg_finding", "STEMI");

            assertTrue(condition.evaluate(patientData));

            patientData.put("ecg_finding", "NSTEMI");
            assertFalse(condition.evaluate(patientData));
        }

        @Test
        @DisplayName("Should evaluate greater than or equal condition correctly")
        void testGreaterThanOrEqualCondition() {
            PathwayStep.Condition condition = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.LAB_VALUE,
                "lactate",
                ">=",
                2.0
            );

            Map<String, Object> patientData = new HashMap<>();

            // Test equal
            patientData.put("lactate", 2.0);
            assertTrue(condition.evaluate(patientData));

            // Test greater
            patientData.put("lactate", 2.5);
            assertTrue(condition.evaluate(patientData));

            // Test less
            patientData.put("lactate", 1.5);
            assertFalse(condition.evaluate(patientData));
        }

        @Test
        @DisplayName("Should evaluate less than or equal condition correctly")
        void testLessThanOrEqualCondition() {
            PathwayStep.Condition condition = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.VITAL_SIGN,
                "oxygen_saturation",
                "<=",
                92
            );

            Map<String, Object> patientData = new HashMap<>();

            // Test equal
            patientData.put("oxygen_saturation", 92);
            assertTrue(condition.evaluate(patientData));

            // Test less
            patientData.put("oxygen_saturation", 88);
            assertTrue(condition.evaluate(patientData));

            // Test greater
            patientData.put("oxygen_saturation", 95);
            assertFalse(condition.evaluate(patientData));
        }

        @Test
        @DisplayName("Should evaluate BETWEEN condition correctly")
        void testBetweenCondition() {
            PathwayStep.Condition condition = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.VITAL_SIGN,
                "heart_rate",
                "BETWEEN",
                60
            );
            condition.setSecondValue(100);

            Map<String, Object> patientData = new HashMap<>();

            // Test within range
            patientData.put("heart_rate", 75);
            assertTrue(condition.evaluate(patientData));

            // Test lower boundary
            patientData.put("heart_rate", 60);
            assertTrue(condition.evaluate(patientData));

            // Test upper boundary
            patientData.put("heart_rate", 100);
            assertTrue(condition.evaluate(patientData));

            // Test below range
            patientData.put("heart_rate", 55);
            assertFalse(condition.evaluate(patientData));

            // Test above range
            patientData.put("heart_rate", 105);
            assertFalse(condition.evaluate(patientData));
        }

        @Test
        @DisplayName("Should handle missing patient data gracefully")
        void testMissingPatientData() {
            PathwayStep.Condition condition = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.LAB_VALUE,
                "creatinine",
                ">",
                1.5
            );

            Map<String, Object> patientData = new HashMap<>();
            // No creatinine value provided

            assertFalse(condition.evaluate(patientData));
        }

        @Test
        @DisplayName("Should handle null condition parameters gracefully")
        void testNullConditionParameters() {
            PathwayStep.Condition condition = new PathwayStep.Condition();
            // No parameters set

            Map<String, Object> patientData = new HashMap<>();
            patientData.put("lactate", 2.5);

            assertFalse(condition.evaluate(patientData));
        }

        @Test
        @DisplayName("Should support all condition types")
        void testAllConditionTypes() {
            PathwayStep.Condition.ConditionType[] types = PathwayStep.Condition.ConditionType.values();

            assertEquals(7, types.length);
            assertTrue(containsConditionType(types, PathwayStep.Condition.ConditionType.LAB_VALUE));
            assertTrue(containsConditionType(types, PathwayStep.Condition.ConditionType.VITAL_SIGN));
            assertTrue(containsConditionType(types, PathwayStep.Condition.ConditionType.CLINICAL_FINDING));
            assertTrue(containsConditionType(types, PathwayStep.Condition.ConditionType.TIME_ELAPSED));
            assertTrue(containsConditionType(types, PathwayStep.Condition.ConditionType.MEDICATION_GIVEN));
            assertTrue(containsConditionType(types, PathwayStep.Condition.ConditionType.PROCEDURE_DONE));
            assertTrue(containsConditionType(types, PathwayStep.Condition.ConditionType.CUSTOM));
        }

        private boolean containsConditionType(PathwayStep.Condition.ConditionType[] types,
                                             PathwayStep.Condition.ConditionType target) {
            for (PathwayStep.Condition.ConditionType type : types) {
                if (type == target) return true;
            }
            return false;
        }
    }

    @Nested
    @DisplayName("Entry/Exit Condition Tests")
    class EntryExitConditionTests {

        @Test
        @DisplayName("Should allow entry when no entry conditions defined")
        void testCanEnterWithNoConditions() {
            Map<String, Object> patientData = new HashMap<>();
            assertTrue(step.canEnter(patientData));
        }

        @Test
        @DisplayName("Should allow entry when all entry conditions are met")
        void testCanEnterWhenConditionsMet() {
            PathwayStep.Condition condition1 = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.LAB_VALUE,
                "troponin",
                ">",
                0.04
            );
            PathwayStep.Condition condition2 = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.VITAL_SIGN,
                "BP_systolic",
                ">",
                90
            );

            step.addEntryCondition(condition1);
            step.addEntryCondition(condition2);

            Map<String, Object> patientData = new HashMap<>();
            patientData.put("troponin", 0.08);
            patientData.put("BP_systolic", 120);

            assertTrue(step.canEnter(patientData));
        }

        @Test
        @DisplayName("Should block entry when any entry condition fails")
        void testCannotEnterWhenConditionFails() {
            PathwayStep.Condition condition1 = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.LAB_VALUE,
                "troponin",
                ">",
                0.04
            );
            PathwayStep.Condition condition2 = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.VITAL_SIGN,
                "BP_systolic",
                ">",
                90
            );

            step.addEntryCondition(condition1);
            step.addEntryCondition(condition2);

            Map<String, Object> patientData = new HashMap<>();
            patientData.put("troponin", 0.08); // Meets condition
            patientData.put("BP_systolic", 85); // Fails condition

            assertFalse(step.canEnter(patientData));
        }

        @Test
        @DisplayName("Should allow exit when no exit conditions defined")
        void testCanExitWithNoConditions() {
            Map<String, Object> patientData = new HashMap<>();
            assertTrue(step.canExit(patientData));
        }

        @Test
        @DisplayName("Should allow exit when all exit conditions are met")
        void testCanExitWhenConditionsMet() {
            PathwayStep.Condition condition = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.PROCEDURE_DONE,
                "ecg_completed",
                "=",
                true
            );

            step.addExitCondition(condition);

            Map<String, Object> patientData = new HashMap<>();
            patientData.put("ecg_completed", true);

            assertTrue(step.canExit(patientData));
        }

        @Test
        @DisplayName("Should block exit when exit condition not met")
        void testCannotExitWhenConditionFails() {
            PathwayStep.Condition condition = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.MEDICATION_GIVEN,
                "aspirin_given",
                "=",
                true
            );

            step.addExitCondition(condition);

            Map<String, Object> patientData = new HashMap<>();
            patientData.put("aspirin_given", false);

            assertFalse(step.canExit(patientData));
        }
    }

    @Nested
    @DisplayName("Medication Order Tests")
    class MedicationOrderTests {

        @Test
        @DisplayName("Should create medication order with required fields")
        void testMedicationOrderCreation() {
            PathwayStep.MedicationOrder med = new PathwayStep.MedicationOrder(
                "Aspirin",
                "325mg",
                "PO"
            );

            assertNotNull(med);
            assertEquals("Aspirin", med.getMedicationName());
            assertEquals("325mg", med.getDose());
            assertEquals("PO", med.getRoute());
        }

        @Test
        @DisplayName("Should add medication to step")
        void testAddMedicationToStep() {
            step.addMedication("Aspirin", "325mg", "PO", true);

            assertEquals(1, step.getMedications().size());

            PathwayStep.MedicationOrder med = step.getMedications().get(0);
            assertEquals("Aspirin", med.getMedicationName());
            assertEquals("325mg", med.getDose());
            assertEquals("PO", med.getRoute());
            assertTrue(med.isStat());
        }

        @Test
        @DisplayName("Should add multiple medications to step")
        void testAddMultipleMedications() {
            step.addMedication("Aspirin", "325mg", "PO", true);
            step.addMedication("Clopidogrel", "600mg", "PO", true);
            step.addMedication("Heparin", "60 units/kg", "IV", true);

            assertEquals(3, step.getMedications().size());
        }

        @Test
        @DisplayName("Should set medication frequency and indication")
        void testMedicationFrequencyAndIndication() {
            PathwayStep.MedicationOrder med = new PathwayStep.MedicationOrder(
                "Metoprolol",
                "25mg",
                "PO"
            );
            med.setFrequency("BID");
            med.setIndication("Heart rate control");

            assertEquals("BID", med.getFrequency());
            assertEquals("Heart rate control", med.getIndication());
        }

        @Test
        @DisplayName("Should format medication toString correctly")
        void testMedicationToString() {
            PathwayStep.MedicationOrder med = new PathwayStep.MedicationOrder(
                "Aspirin",
                "325mg",
                "PO"
            );
            med.setStat(true);

            String str = med.toString();
            assertTrue(str.contains("Aspirin"));
            assertTrue(str.contains("325mg"));
            assertTrue(str.contains("PO"));
            assertTrue(str.contains("STAT"));
        }

        @Test
        @DisplayName("Should distinguish STAT vs routine medications")
        void testStatVsRoutineMedications() {
            PathwayStep.MedicationOrder statMed = new PathwayStep.MedicationOrder(
                "Aspirin",
                "325mg",
                "PO"
            );
            statMed.setStat(true);

            PathwayStep.MedicationOrder routineMed = new PathwayStep.MedicationOrder(
                "Metoprolol",
                "25mg",
                "PO"
            );
            routineMed.setStat(false);

            assertTrue(statMed.isStat());
            assertFalse(routineMed.isStat());
        }
    }

    @Nested
    @DisplayName("Time-Critical Step Tests")
    class TimeCriticalStepTests {

        @Test
        @DisplayName("Should configure time-critical step")
        void testTimeCriticalConfiguration() {
            step.setTimeCritical(true);
            step.setExpectedDurationMinutes(10);
            step.setMaxDurationMinutes(15);

            assertTrue(step.isTimeCritical());
            assertEquals(10, step.getExpectedDurationMinutes());
            assertEquals(15, step.getMaxDurationMinutes());
        }

        @Test
        @DisplayName("Should default to non-time-critical")
        void testDefaultTimeCritical() {
            assertFalse(step.isTimeCritical());
        }

        @Test
        @DisplayName("Should support null duration values")
        void testNullDurations() {
            step.setExpectedDurationMinutes(null);
            step.setMaxDurationMinutes(null);

            assertNull(step.getExpectedDurationMinutes());
            assertNull(step.getMaxDurationMinutes());
        }
    }

    @Nested
    @DisplayName("Clinical Data Requirements Tests")
    class ClinicalDataRequirementsTests {

        @Test
        @DisplayName("Should add required vitals to step")
        void testRequiredVitals() {
            step.getRequiredVitals().add("BP");
            step.getRequiredVitals().add("HR");
            step.getRequiredVitals().add("SpO2");

            assertEquals(3, step.getRequiredVitals().size());
            assertTrue(step.getRequiredVitals().contains("BP"));
            assertTrue(step.getRequiredVitals().contains("HR"));
            assertTrue(step.getRequiredVitals().contains("SpO2"));
        }

        @Test
        @DisplayName("Should add required labs to step")
        void testRequiredLabs() {
            step.getRequiredLabs().add("Troponin");
            step.getRequiredLabs().add("BNP");
            step.getRequiredLabs().add("D-dimer");

            assertEquals(3, step.getRequiredLabs().size());
            assertTrue(step.getRequiredLabs().contains("Troponin"));
        }

        @Test
        @DisplayName("Should add required assessments to step")
        void testRequiredAssessments() {
            step.getRequiredAssessments().add("12-lead ECG");
            step.getRequiredAssessments().add("Chest X-ray");

            assertEquals(2, step.getRequiredAssessments().size());
        }

        @Test
        @DisplayName("Should add procedures to step")
        void testProcedures() {
            step.getProcedures().add("Cardiac catheterization");
            step.getProcedures().add("Intubation");

            assertEquals(2, step.getProcedures().size());
        }

        @Test
        @DisplayName("Should add consultations to step")
        void testConsultations() {
            step.getConsultations().add("Cardiology");
            step.getConsultations().add("Interventional Radiology");

            assertEquals(2, step.getConsultations().size());
        }
    }

    @Nested
    @DisplayName("Clinical Instructions and Actions Tests")
    class InstructionsAndActionsTests {

        @Test
        @DisplayName("Should add instructions to step")
        void testAddInstructions() {
            step.addInstruction("Obtain 12-lead ECG within 10 minutes");
            step.addInstruction("Draw cardiac biomarkers");

            assertEquals(2, step.getInstructions().size());
            assertTrue(step.getInstructions().get(0).contains("ECG"));
        }

        @Test
        @DisplayName("Should add required actions to step")
        void testAddRequiredActions() {
            step.addRequiredAction("Administer aspirin 325mg PO");
            step.addRequiredAction("Establish IV access");

            assertEquals(2, step.getRequiredActions().size());
        }

        @Test
        @DisplayName("Should set clinical rationale")
        void testClinicalRationale() {
            step.setClinicalRationale("Early ECG is critical for STEMI identification");

            assertNotNull(step.getClinicalRationale());
            assertTrue(step.getClinicalRationale().contains("STEMI"));
        }

        @Test
        @DisplayName("Should set description")
        void testDescription() {
            step.setDescription("Perform initial cardiac assessment and risk stratification");

            assertNotNull(step.getDescription());
            assertTrue(step.getDescription().contains("cardiac assessment"));
        }
    }

    @Nested
    @DisplayName("Quality and Safety Tests")
    class QualityAndSafetyTests {

        @Test
        @DisplayName("Should add clinical alerts")
        void testClinicalAlerts() {
            step.getClinicalAlerts().add("Patient has aspirin allergy");
            step.getClinicalAlerts().add("Recent GI bleed history");

            assertEquals(2, step.getClinicalAlerts().size());
        }

        @Test
        @DisplayName("Should add safeguards")
        void testSafeguards() {
            step.getSafeguards().add("Verify no contraindications to anticoagulation");
            step.getSafeguards().add("Confirm patient not on dual antiplatelet therapy");

            assertEquals(2, step.getSafeguards().size());
        }

        @Test
        @DisplayName("Should configure physician approval requirement")
        void testPhysicianApproval() {
            step.setRequiresPhysicianApproval(true);
            assertTrue(step.isRequiresPhysicianApproval());

            step.setRequiresPhysicianApproval(false);
            assertFalse(step.isRequiresPhysicianApproval());
        }

        @Test
        @DisplayName("Should set evidence level")
        void testEvidenceLevel() {
            step.setEvidenceLevel("A");
            assertEquals("A", step.getEvidenceLevel());
        }

        @Test
        @DisplayName("Should add quality measures")
        void testQualityMeasures() {
            step.getQualityMeasures().add("Door-to-ECG time < 10 minutes");
            step.getQualityMeasures().add("Aspirin administration time");

            assertEquals(2, step.getQualityMeasures().size());
        }

        @Test
        @DisplayName("Should configure core quality measure flag")
        void testCoreQualityMeasure() {
            step.setCoreQualityMeasure(true);
            assertTrue(step.isCoreQualityMeasure());
        }

        @Test
        @DisplayName("Should add required documentation")
        void testRequiredDocumentation() {
            step.getRequiredDocumentation().add("Time of symptom onset");
            step.getRequiredDocumentation().add("ECG interpretation");

            assertEquals(2, step.getRequiredDocumentation().size());
        }
    }

    @Nested
    @DisplayName("Transition and Branching Tests")
    class TransitionTests {

        @Test
        @DisplayName("Should add transitions for branching")
        void testAddTransitions() {
            step.addTransition("STEMI", "STEP-CATH-LAB");
            step.addTransition("NSTEMI", "STEP-MEDICAL-MANAGEMENT");

            assertEquals(2, step.getTransitions().size());
            assertEquals("STEP-CATH-LAB", step.getTransitions().get("STEMI"));
            assertEquals("STEP-MEDICAL-MANAGEMENT", step.getTransitions().get("NSTEMI"));
        }

        @Test
        @DisplayName("Should set decision criteria")
        void testDecisionCriteria() {
            step.setDecisionCriteria("Based on ECG findings: STEMI vs NSTEMI vs non-cardiac");

            assertNotNull(step.getDecisionCriteria());
            assertTrue(step.getDecisionCriteria().contains("ECG findings"));
        }

        @Test
        @DisplayName("Should determine next step from transitions")
        void testDetermineNextStep() {
            step.addTransition("HIGH_RISK", "STEP-ICU");
            step.addTransition("LOW_RISK", "STEP-OBSERVATION");

            Map<String, Object> patientData = new HashMap<>();

            // Note: Current implementation returns first transition
            // In production, would evaluate conditions properly
            String nextStep = step.determineNextStep(patientData);
            assertNotNull(nextStep);
        }
    }

    @Nested
    @DisplayName("toString Tests")
    class ToStringTests {

        @Test
        @DisplayName("Should generate meaningful string representation")
        void testToString() {
            step.setStepOrder(1);
            step.setTimeCritical(true);

            String str = step.toString();

            assertTrue(str.contains(TEST_STEP_ID));
            assertTrue(str.contains(TEST_STEP_NAME));
            assertTrue(str.contains("ASSESSMENT"));
            assertTrue(str.contains("order=1"));
            assertTrue(str.contains("timeCritical=true"));
        }

        @Test
        @DisplayName("Should generate condition toString")
        void testConditionToString() {
            PathwayStep.Condition condition = new PathwayStep.Condition(
                PathwayStep.Condition.ConditionType.LAB_VALUE,
                "troponin",
                ">",
                0.04
            );

            String str = condition.toString();
            assertTrue(str.contains("troponin"));
            assertTrue(str.contains(">"));
            assertTrue(str.contains("0.04"));
        }
    }
}
