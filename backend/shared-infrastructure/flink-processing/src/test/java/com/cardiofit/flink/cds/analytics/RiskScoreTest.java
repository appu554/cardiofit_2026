package com.cardiofit.flink.cds.analytics;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;

import java.time.LocalDateTime;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for RiskScore model
 * Phase 8 Module 3 - Predictive Risk Scoring
 *
 * Test Coverage:
 * - Core model creation and validation
 * - Risk categorization logic
 * - Input parameter and feature weight tracking
 * - Confidence interval validation
 * - Clinical action recommendations
 *
 * @author CardioFit Testing Team
 * @version 1.0.0
 * @since Phase 8
 */
@DisplayName("RiskScore Model Tests")
class RiskScoreTest {

    private RiskScore riskScore;
    private static final String TEST_PATIENT_ID = "PATIENT-12345";
    private static final double DELTA = 0.001; // Precision for double comparisons

    @BeforeEach
    void setUp() {
        riskScore = new RiskScore();
    }

    @Nested
    @DisplayName("Core Model Tests")
    class CoreModelTests {

        @Test
        @DisplayName("Should create risk score with default values")
        void testDefaultConstruction() {
            assertNotNull(riskScore);
            assertNotNull(riskScore.getInputParameters());
            assertNotNull(riskScore.getFeatureWeights());
            assertNotNull(riskScore.getCalculationTime());
            assertFalse(riskScore.isValidated());
        }

        @Test
        @DisplayName("Should create risk score with patient ID and type")
        void testParameterizedConstruction() {
            RiskScore score = new RiskScore(TEST_PATIENT_ID, RiskScore.RiskType.MORTALITY, 0.35);

            assertEquals(TEST_PATIENT_ID, score.getPatientId());
            assertEquals(RiskScore.RiskType.MORTALITY, score.getRiskType());
            assertEquals(0.35, score.getScore(), DELTA);
            assertNotNull(score.getScoreId());
            assertTrue(score.getScoreId().contains("RISK_"));
            assertTrue(score.getScoreId().contains("MORTALITY"));
        }

        @Test
        @DisplayName("Should generate unique score IDs")
        void testUniqueScoreIds() throws InterruptedException {
            RiskScore score1 = new RiskScore(TEST_PATIENT_ID, RiskScore.RiskType.MORTALITY, 0.25);
            Thread.sleep(2); // Ensure different timestamps
            RiskScore score2 = new RiskScore(TEST_PATIENT_ID, RiskScore.RiskType.MORTALITY, 0.30);

            assertNotEquals(score1.getScoreId(), score2.getScoreId());
        }
    }

    @Nested
    @DisplayName("Risk Categorization Tests")
    class RiskCategorizationTests {

        @Test
        @DisplayName("Should categorize low risk (score < 0.2)")
        void testLowRiskCategorization() {
            riskScore.setScore(0.10);
            RiskScore.RiskCategory category = riskScore.categorizeRisk();

            assertEquals(RiskScore.RiskCategory.LOW, category);
            assertEquals(0, category.getSeverity());
            assertEquals("Routine monitoring sufficient", category.getClinicalGuidance());
        }

        @Test
        @DisplayName("Should categorize moderate risk (0.2 <= score < 0.5)")
        void testModerateRiskCategorization() {
            riskScore.setScore(0.35);
            RiskScore.RiskCategory category = riskScore.categorizeRisk();

            assertEquals(RiskScore.RiskCategory.MODERATE, category);
            assertEquals(1, category.getSeverity());
            assertEquals("Enhanced monitoring recommended", category.getClinicalGuidance());
        }

        @Test
        @DisplayName("Should categorize high risk (0.5 <= score < 0.8)")
        void testHighRiskCategorization() {
            riskScore.setScore(0.65);
            RiskScore.RiskCategory category = riskScore.categorizeRisk();

            assertEquals(RiskScore.RiskCategory.HIGH, category);
            assertEquals(2, category.getSeverity());
            assertEquals("Immediate clinical assessment required", category.getClinicalGuidance());
        }

        @Test
        @DisplayName("Should categorize critical risk (score >= 0.8)")
        void testCriticalRiskCategorization() {
            riskScore.setScore(0.90);
            RiskScore.RiskCategory category = riskScore.categorizeRisk();

            assertEquals(RiskScore.RiskCategory.CRITICAL, category);
            assertEquals(3, category.getSeverity());
            assertEquals("Urgent intervention required", category.getClinicalGuidance());
        }

        @Test
        @DisplayName("Should handle boundary values correctly")
        void testBoundaryValues() {
            // Test exact boundary at 0.2 (should be MODERATE)
            riskScore.setScore(0.2);
            assertEquals(RiskScore.RiskCategory.MODERATE, riskScore.categorizeRisk());

            // Test exact boundary at 0.5 (should be HIGH)
            riskScore.setScore(0.5);
            assertEquals(RiskScore.RiskCategory.HIGH, riskScore.categorizeRisk());

            // Test exact boundary at 0.8 (should be CRITICAL)
            riskScore.setScore(0.8);
            assertEquals(RiskScore.RiskCategory.CRITICAL, riskScore.categorizeRisk());

            // Test minimum value (should be LOW)
            riskScore.setScore(0.0);
            assertEquals(RiskScore.RiskCategory.LOW, riskScore.categorizeRisk());

            // Test maximum value (should be CRITICAL)
            riskScore.setScore(1.0);
            assertEquals(RiskScore.RiskCategory.CRITICAL, riskScore.categorizeRisk());
        }
    }

    @Nested
    @DisplayName("Immediate Action Tests")
    class ImmediateActionTests {

        @Test
        @DisplayName("Should require immediate action for HIGH risk")
        void testHighRiskRequiresAction() {
            riskScore.setRiskCategory(RiskScore.RiskCategory.HIGH);
            assertTrue(riskScore.requiresImmediateAction());
        }

        @Test
        @DisplayName("Should require immediate action for CRITICAL risk")
        void testCriticalRiskRequiresAction() {
            riskScore.setRiskCategory(RiskScore.RiskCategory.CRITICAL);
            assertTrue(riskScore.requiresImmediateAction());
        }

        @Test
        @DisplayName("Should NOT require immediate action for MODERATE risk")
        void testModerateRiskNoAction() {
            riskScore.setRiskCategory(RiskScore.RiskCategory.MODERATE);
            assertFalse(riskScore.requiresImmediateAction());
        }

        @Test
        @DisplayName("Should NOT require immediate action for LOW risk")
        void testLowRiskNoAction() {
            riskScore.setRiskCategory(RiskScore.RiskCategory.LOW);
            assertFalse(riskScore.requiresImmediateAction());
        }
    }

    @Nested
    @DisplayName("Input Parameters Tests")
    class InputParametersTests {

        @Test
        @DisplayName("Should add and retrieve input parameters")
        void testAddInputParameter() {
            riskScore.addInputParameter("heart_rate", 110.0);
            riskScore.addInputParameter("systolic_bp", 85.0);
            riskScore.addInputParameter("is_icu_patient", true);

            assertEquals(3, riskScore.getInputParameters().size());
            assertEquals(110.0, riskScore.getInputParameters().get("heart_rate"));
            assertEquals(85.0, riskScore.getInputParameters().get("systolic_bp"));
            assertEquals(true, riskScore.getInputParameters().get("is_icu_patient"));
        }

        @Test
        @DisplayName("Should handle various input parameter types")
        void testVariousParameterTypes() {
            riskScore.addInputParameter("age", 65);
            riskScore.addInputParameter("temperature", 38.5);
            riskScore.addInputParameter("diagnosis", "Sepsis");
            riskScore.addInputParameter("ventilated", true);

            assertEquals(4, riskScore.getInputParameters().size());
            assertTrue(riskScore.getInputParameters().get("age") instanceof Integer);
            assertTrue(riskScore.getInputParameters().get("temperature") instanceof Double);
            assertTrue(riskScore.getInputParameters().get("diagnosis") instanceof String);
            assertTrue(riskScore.getInputParameters().get("ventilated") instanceof Boolean);
        }
    }

    @Nested
    @DisplayName("Feature Weights Tests")
    class FeatureWeightsTests {

        @Test
        @DisplayName("Should add and retrieve feature weights")
        void testAddFeatureWeight() {
            riskScore.addFeatureWeight("age", 15.0);
            riskScore.addFeatureWeight("chronic_conditions", 8.0);
            riskScore.addFeatureWeight("acute_physiology", 42.0);

            assertEquals(3, riskScore.getFeatureWeights().size());
            assertEquals(15.0, riskScore.getFeatureWeights().get("age"), DELTA);
            assertEquals(8.0, riskScore.getFeatureWeights().get("chronic_conditions"), DELTA);
            assertEquals(42.0, riskScore.getFeatureWeights().get("acute_physiology"), DELTA);
        }

        @Test
        @DisplayName("Should retrieve top N contributing factors")
        void testGetTopContributors() {
            riskScore.addFeatureWeight("age", 15.0);
            riskScore.addFeatureWeight("heart_rate", 7.0);
            riskScore.addFeatureWeight("blood_pressure", 23.0);
            riskScore.addFeatureWeight("creatinine", 10.0);
            riskScore.addFeatureWeight("wbc", 4.0);

            var top3 = riskScore.getTopContributors(3);

            assertEquals(3, top3.size());
            assertTrue(top3.containsKey("blood_pressure"));
            assertTrue(top3.containsKey("age"));
            assertTrue(top3.containsKey("creatinine"));
            assertFalse(top3.containsKey("wbc")); // Should not be in top 3
        }

        @Test
        @DisplayName("Should return all factors if N exceeds total")
        void testGetTopContributorsExceedingTotal() {
            riskScore.addFeatureWeight("factor1", 10.0);
            riskScore.addFeatureWeight("factor2", 5.0);

            var top5 = riskScore.getTopContributors(5);

            assertEquals(2, top5.size()); // Only 2 factors available
        }
    }

    @Nested
    @DisplayName("Validation Tests")
    class ValidationTests {

        @Test
        @DisplayName("Should pass validation with complete valid data")
        void testValidationSuccess() {
            riskScore.setPatientId(TEST_PATIENT_ID);
            riskScore.setRiskType(RiskScore.RiskType.MORTALITY);
            riskScore.setScore(0.35);
            riskScore.setConfidenceLower(0.25);
            riskScore.setConfidenceUpper(0.45);
            riskScore.setCalculationMethod("APACHE_III_v1991");

            boolean isValid = riskScore.validate();

            assertTrue(isValid);
            assertTrue(riskScore.isValidated());
            assertEquals("Validation passed", riskScore.getValidationNotes());
        }

        @Test
        @DisplayName("Should fail validation if score out of range (too low)")
        void testValidationFailsScoreTooLow() {
            riskScore.setPatientId(TEST_PATIENT_ID);
            riskScore.setRiskType(RiskScore.RiskType.MORTALITY);
            riskScore.setScore(-0.1);
            riskScore.setCalculationMethod("APACHE_III_v1991");

            boolean isValid = riskScore.validate();

            assertFalse(isValid);
            assertFalse(riskScore.isValidated());
            assertTrue(riskScore.getValidationNotes().contains("out of range"));
        }

        @Test
        @DisplayName("Should fail validation if score out of range (too high)")
        void testValidationFailsScoreTooHigh() {
            riskScore.setPatientId(TEST_PATIENT_ID);
            riskScore.setRiskType(RiskScore.RiskType.MORTALITY);
            riskScore.setScore(1.5);
            riskScore.setCalculationMethod("APACHE_III_v1991");

            boolean isValid = riskScore.validate();

            assertFalse(isValid);
            assertTrue(riskScore.getValidationNotes().contains("out of range"));
        }

        @Test
        @DisplayName("Should fail validation if confidence interval invalid")
        void testValidationFailsInvalidConfidenceInterval() {
            riskScore.setPatientId(TEST_PATIENT_ID);
            riskScore.setRiskType(RiskScore.RiskType.MORTALITY);
            riskScore.setScore(0.35);
            riskScore.setConfidenceLower(0.50); // Lower > Upper (invalid)
            riskScore.setConfidenceUpper(0.30);
            riskScore.setCalculationMethod("APACHE_III_v1991");

            boolean isValid = riskScore.validate();

            assertFalse(isValid);
            assertTrue(riskScore.getValidationNotes().contains("confidence interval"));
        }

        @Test
        @DisplayName("Should fail validation if patient ID missing")
        void testValidationFailsMissingPatientId() {
            riskScore.setPatientId(null);
            riskScore.setRiskType(RiskScore.RiskType.MORTALITY);
            riskScore.setScore(0.35);
            riskScore.setCalculationMethod("APACHE_III_v1991");

            boolean isValid = riskScore.validate();

            assertFalse(isValid);
            assertTrue(riskScore.getValidationNotes().contains("patient ID"));
        }

        @Test
        @DisplayName("Should fail validation if patient ID empty")
        void testValidationFailsEmptyPatientId() {
            riskScore.setPatientId("");
            riskScore.setRiskType(RiskScore.RiskType.MORTALITY);
            riskScore.setScore(0.35);
            riskScore.setCalculationMethod("APACHE_III_v1991");

            boolean isValid = riskScore.validate();

            assertFalse(isValid);
            assertTrue(riskScore.getValidationNotes().contains("patient ID"));
        }

        @Test
        @DisplayName("Should fail validation if risk type missing")
        void testValidationFailsMissingRiskType() {
            riskScore.setPatientId(TEST_PATIENT_ID);
            riskScore.setRiskType(null);
            riskScore.setScore(0.35);
            riskScore.setCalculationMethod("APACHE_III_v1991");

            boolean isValid = riskScore.validate();

            assertFalse(isValid);
            assertTrue(riskScore.getValidationNotes().contains("risk type"));
        }

        @Test
        @DisplayName("Should fail validation if calculation method missing")
        void testValidationFailsMissingCalculationMethod() {
            riskScore.setPatientId(TEST_PATIENT_ID);
            riskScore.setRiskType(RiskScore.RiskType.MORTALITY);
            riskScore.setScore(0.35);
            riskScore.setCalculationMethod(null);

            boolean isValid = riskScore.validate();

            assertFalse(isValid);
            assertTrue(riskScore.getValidationNotes().contains("calculation method"));
        }

        @Test
        @DisplayName("Should fail validation if calculation method empty")
        void testValidationFailsEmptyCalculationMethod() {
            riskScore.setPatientId(TEST_PATIENT_ID);
            riskScore.setRiskType(RiskScore.RiskType.MORTALITY);
            riskScore.setScore(0.35);
            riskScore.setCalculationMethod("");

            boolean isValid = riskScore.validate();

            assertFalse(isValid);
            assertTrue(riskScore.getValidationNotes().contains("calculation method"));
        }
    }

    @Nested
    @DisplayName("Risk Type Enum Tests")
    class RiskTypeEnumTests {

        @Test
        @DisplayName("Should have all required risk types")
        void testRiskTypeValues() {
            RiskScore.RiskType[] types = RiskScore.RiskType.values();

            assertEquals(8, types.length);
            assertTrue(containsType(types, RiskScore.RiskType.MORTALITY));
            assertTrue(containsType(types, RiskScore.RiskType.READMISSION));
            assertTrue(containsType(types, RiskScore.RiskType.SEPSIS));
            assertTrue(containsType(types, RiskScore.RiskType.DETERIORATION));
            assertTrue(containsType(types, RiskScore.RiskType.CARDIAC_EVENT));
            assertTrue(containsType(types, RiskScore.RiskType.RESPIRATORY_FAILURE));
            assertTrue(containsType(types, RiskScore.RiskType.RENAL_FAILURE));
            assertTrue(containsType(types, RiskScore.RiskType.CUSTOM));
        }

        private boolean containsType(RiskScore.RiskType[] types, RiskScore.RiskType target) {
            for (RiskScore.RiskType type : types) {
                if (type == target) return true;
            }
            return false;
        }
    }

    @Nested
    @DisplayName("Risk Category Enum Tests")
    class RiskCategoryEnumTests {

        @Test
        @DisplayName("Should have all required risk categories")
        void testRiskCategoryValues() {
            RiskScore.RiskCategory[] categories = RiskScore.RiskCategory.values();

            assertEquals(4, categories.length);
        }

        @Test
        @DisplayName("Should have correct severity ordering")
        void testSeverityOrdering() {
            assertTrue(RiskScore.RiskCategory.LOW.getSeverity() <
                      RiskScore.RiskCategory.MODERATE.getSeverity());
            assertTrue(RiskScore.RiskCategory.MODERATE.getSeverity() <
                      RiskScore.RiskCategory.HIGH.getSeverity());
            assertTrue(RiskScore.RiskCategory.HIGH.getSeverity() <
                      RiskScore.RiskCategory.CRITICAL.getSeverity());
        }

        @Test
        @DisplayName("Should have clinical guidance for all categories")
        void testClinicalGuidance() {
            assertNotNull(RiskScore.RiskCategory.LOW.getClinicalGuidance());
            assertNotNull(RiskScore.RiskCategory.MODERATE.getClinicalGuidance());
            assertNotNull(RiskScore.RiskCategory.HIGH.getClinicalGuidance());
            assertNotNull(RiskScore.RiskCategory.CRITICAL.getClinicalGuidance());

            assertFalse(RiskScore.RiskCategory.LOW.getClinicalGuidance().isEmpty());
            assertFalse(RiskScore.RiskCategory.MODERATE.getClinicalGuidance().isEmpty());
            assertFalse(RiskScore.RiskCategory.HIGH.getClinicalGuidance().isEmpty());
            assertFalse(RiskScore.RiskCategory.CRITICAL.getClinicalGuidance().isEmpty());
        }
    }

    @Nested
    @DisplayName("toString Tests")
    class ToStringTests {

        @Test
        @DisplayName("Should generate meaningful string representation")
        void testToString() {
            RiskScore score = new RiskScore(TEST_PATIENT_ID, RiskScore.RiskType.MORTALITY, 0.45);
            score.setCalculationMethod("APACHE_III_v1991");
            score.setRiskCategory(RiskScore.RiskCategory.MODERATE);
            score.validate();

            String str = score.toString();

            assertTrue(str.contains(TEST_PATIENT_ID));
            assertTrue(str.contains("MORTALITY"));
            assertTrue(str.contains("0.450"));
            assertTrue(str.contains("MODERATE"));
            assertTrue(str.contains("APACHE_III_v1991"));
            assertTrue(str.contains("validated=true"));
        }
    }

    @Nested
    @DisplayName("Metadata Tests")
    class MetadataTests {

        @Test
        @DisplayName("Should store calculation metadata")
        void testCalculationMetadata() {
            LocalDateTime calcTime = LocalDateTime.now();
            riskScore.setCalculationTime(calcTime);
            riskScore.setCalculatedBy("PredictiveEngine-1.0.0");
            riskScore.setModelVersion("APACHE_III_v1991");

            assertEquals(calcTime, riskScore.getCalculationTime());
            assertEquals("PredictiveEngine-1.0.0", riskScore.getCalculatedBy());
            assertEquals("APACHE_III_v1991", riskScore.getModelVersion());
        }

        @Test
        @DisplayName("Should store clinical context")
        void testClinicalContext() {
            riskScore.setPrimaryDiagnosis("J96.00"); // Acute respiratory failure
            riskScore.setRequiresIntervention(true);
            riskScore.setRecommendedAction("Initiate mechanical ventilation");

            assertEquals("J96.00", riskScore.getPrimaryDiagnosis());
            assertTrue(riskScore.isRequiresIntervention());
            assertEquals("Initiate mechanical ventilation", riskScore.getRecommendedAction());
        }
    }
}
