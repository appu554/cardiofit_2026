package com.cardiofit.flink.cds.analytics;

import com.cardiofit.flink.cds.analytics.models.LabResults;
import com.cardiofit.flink.cds.analytics.models.PatientContext;
import com.cardiofit.flink.models.VitalSigns;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.Arrays;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for PredictiveEngine
 * Phase 8 Module 3 - Predictive Risk Scoring
 *
 * Test Coverage:
 * - APACHE III Mortality Risk Calculator
 * - HOSPITAL Score Readmission Risk
 * - qSOFA Sepsis Screening
 * - MEWS Deterioration Detection
 * - Edge cases and boundary conditions
 *
 * Test data based on published clinical validation studies.
 *
 * @author CardioFit Testing Team
 * @version 1.0.0
 * @since Phase 8
 */
@DisplayName("PredictiveEngine Tests")
class PredictiveEngineTest {

    private PredictiveEngine engine;
    private PatientContext patientContext;
    private VitalSigns vitals;
    private LabResults labs;

    private static final String TEST_PATIENT_ID = "PATIENT-RISK-001";
    private static final double DELTA = 0.001;

    @BeforeEach
    void setUp() {
        engine = new PredictiveEngine();

        // Setup default patient context
        patientContext = new PatientContext(TEST_PATIENT_ID);
        patientContext.setDateOfBirth(LocalDate.of(1955, 6, 15)); // 68 years old
        patientContext.setGender("M");
        patientContext.setICUPatient(true);

        // Setup default vital signs
        vitals = new VitalSigns();
        vitals.setPatientId(TEST_PATIENT_ID);
        vitals.setTimestamp(LocalDateTime.now());

        // Setup default labs
        labs = new LabResults(TEST_PATIENT_ID);
    }

    @Nested
    @DisplayName("APACHE III Mortality Risk Tests")
    class ApacheIIIMortalityTests {

        @Test
        @DisplayName("Should calculate low mortality risk for stable ICU patient")
        void testLowMortalityRisk() {
            // Stable 68-year-old patient with normal vital signs and labs
            vitals.setHeartRate(75.0);
            vitals.setSystolicBP(125.0);
            vitals.setDiastolicBP(75.0);
            vitals.setTemperature(37.0);
            vitals.setRespiratoryRate(14.0);
            vitals.setOxygenSaturation(98.0);

            labs.setSodium(140.0);
            labs.setPotassium(4.0);
            labs.setCreatinine(1.0);
            labs.setHematocrit(40.0);
            labs.setWBC(8.0);
            labs.setBUN(18.0);

            RiskScore riskScore = engine.calculateMortalityRisk(patientContext, vitals, labs);

            assertNotNull(riskScore);
            assertEquals(TEST_PATIENT_ID, riskScore.getPatientId());
            assertEquals(RiskScore.RiskType.MORTALITY, riskScore.getRiskType());
            assertTrue(riskScore.getScore() < 0.20, "Expected low mortality risk (<20%)");
            assertTrue(riskScore.isValidated());
            assertFalse(riskScore.isRequiresIntervention());
        }

        @Test
        @DisplayName("Should calculate high mortality risk for critically ill elderly patient")
        void testHighMortalityRisk() {
            // 85-year-old with severe physiologic derangement
            patientContext.setDateOfBirth(LocalDate.of(1938, 1, 1)); // 85 years old
            patientContext.addCondition("Cirrhosis");
            patientContext.addCondition("Chronic Heart Failure");

            // Severely abnormal vitals
            vitals.setHeartRate(145.0);      // Tachycardia (4 points)
            vitals.setSystolicBP(75.0);      // Hypotension (MAP ~60, 7 points)
            vitals.setDiastolicBP(50.0);
            vitals.setTemperature(39.5);     // Fever (3 points)
            vitals.setRespiratoryRate(32.0); // Tachypnea (6 points)
            vitals.setOxygenSaturation(86.0); // Hypoxemia (2 points)

            // Severely abnormal labs
            labs.setSodium(125.0);          // Hyponatremia (3 points)
            labs.setPotassium(6.2);         // Hyperkalemia (4 points)
            labs.setCreatinine(3.8);        // Renal failure (10 points)
            labs.setHematocrit(25.0);       // Anemia (3 points)
            labs.setWBC(25.0);              // Leukocytosis (4 points)
            labs.setBUN(65.0);              // Elevated (5 points)

            RiskScore riskScore = engine.calculateMortalityRisk(patientContext, vitals, labs);

            assertNotNull(riskScore);
            assertTrue(riskScore.getScore() > 0.50, "Expected high mortality risk (>50%)");
            assertEquals(RiskScore.RiskCategory.CRITICAL, riskScore.getRiskCategory());
            assertTrue(riskScore.isRequiresIntervention());
            assertTrue(riskScore.getRecommendedAction().contains("CRITICAL"));
            assertTrue(riskScore.getRecommendedAction().toLowerCase().contains("intensivist"));
        }

        @Test
        @DisplayName("Should calculate APACHE III score components correctly")
        void testApacheIIIComponents() {
            // Moderate severity patient
            patientContext.setDateOfBirth(LocalDate.of(1963, 1, 1)); // 60 years old
            patientContext.addCondition("Chronic Kidney Disease");

            vitals.setHeartRate(115.0);      // Tachycardia
            vitals.setSystolicBP(95.0);      // Borderline hypotension
            vitals.setDiastolicBP(60.0);
            vitals.setTemperature(38.2);     // Low-grade fever
            vitals.setRespiratoryRate(22.0); // Tachypnea
            vitals.setOxygenSaturation(93.0);

            labs.setSodium(133.0);
            labs.setPotassium(5.2);
            labs.setCreatinine(2.1);
            labs.setHematocrit(32.0);
            labs.setWBC(14.0);

            RiskScore riskScore = engine.calculateMortalityRisk(patientContext, vitals, labs);

            // Verify input parameters were captured
            assertTrue(riskScore.getInputParameters().containsKey("heart_rate"));
            assertTrue(riskScore.getInputParameters().containsKey("mean_arterial_pressure"));
            assertTrue(riskScore.getInputParameters().containsKey("temperature_celsius"));
            assertTrue(riskScore.getInputParameters().containsKey("respiratory_rate"));
            assertTrue(riskScore.getInputParameters().containsKey("age_years"));
            assertTrue(riskScore.getInputParameters().containsKey("apache_iii_total_score"));

            // Verify feature weights were assigned
            assertTrue(riskScore.getFeatureWeights().size() > 0);
            assertTrue(riskScore.getFeatureWeights().containsKey("heart_rate"));
            assertTrue(riskScore.getFeatureWeights().containsKey("age"));
        }

        @Test
        @DisplayName("Should handle missing vital signs gracefully")
        void testMissingVitalSigns() {
            // Only provide some vitals
            vitals.setHeartRate(80.0);
            vitals.setTemperature(37.2);
            // Missing: BP, RR, SpO2

            labs.setSodium(140.0);
            labs.setPotassium(4.0);

            RiskScore riskScore = engine.calculateMortalityRisk(patientContext, vitals, labs);

            assertNotNull(riskScore);
            assertTrue(riskScore.isValidated());
            // Should still calculate score with available data
            assertNotNull(riskScore.getScore());
        }

        @Test
        @DisplayName("Should apply age points correctly")
        void testAgeScoringBrackets() {
            // Test different age groups
            labs.setSodium(140.0);
            labs.setPotassium(4.0);
            vitals.setHeartRate(75.0);
            vitals.setSystolicBP(120.0);
            vitals.setDiastolicBP(75.0);

            // 45-year-old (3 points)
            patientContext.setDateOfBirth(LocalDate.of(1978, 1, 1));
            RiskScore score45 = engine.calculateMortalityRisk(patientContext, vitals, labs);

            // 70-year-old (13 points)
            patientContext.setDateOfBirth(LocalDate.of(1953, 1, 1));
            RiskScore score70 = engine.calculateMortalityRisk(patientContext, vitals, labs);

            // 85-year-old (24 points)
            patientContext.setDateOfBirth(LocalDate.of(1938, 1, 1));
            RiskScore score85 = engine.calculateMortalityRisk(patientContext, vitals, labs);

            // Older patients should have higher risk with same vitals/labs
            assertTrue(score45.getScore() < score70.getScore());
            assertTrue(score70.getScore() < score85.getScore());
        }

        @Test
        @DisplayName("Should apply chronic health points for comorbidities")
        void testChronicHealthScoring() {
            vitals.setHeartRate(75.0);
            vitals.setSystolicBP(120.0);
            vitals.setDiastolicBP(75.0);
            labs.setSodium(140.0);

            // No chronic conditions
            patientContext.setActiveConditions(Arrays.asList());
            RiskScore scoreNoComorbidities = engine.calculateMortalityRisk(patientContext, vitals, labs);

            // With cirrhosis (16 points)
            patientContext.setActiveConditions(Arrays.asList("Cirrhosis"));
            RiskScore scoreCirrhosis = engine.calculateMortalityRisk(patientContext, vitals, labs);

            // With AIDS (23 points)
            patientContext.setActiveConditions(Arrays.asList("AIDS"));
            RiskScore scoreAIDS = engine.calculateMortalityRisk(patientContext, vitals, labs);

            // Chronic conditions should increase mortality risk
            assertTrue(scoreNoComorbidities.getScore() < scoreCirrhosis.getScore());
            assertTrue(scoreCirrhosis.getScore() < scoreAIDS.getScore());
        }
    }

    @Nested
    @DisplayName("HOSPITAL Score Readmission Risk Tests")
    class HospitalScoreReadmissionTests {

        @Test
        @DisplayName("Should calculate low readmission risk (HOSPITAL score ≤4)")
        void testLowReadmissionRisk() {
            labs.setHemoglobin(13.5);   // Normal (0 points)
            labs.setSodium(140.0);       // Normal (0 points)

            RiskScore riskScore = engine.calculateReadmissionRisk(
                patientContext,
                labs,
                3,              // LOS <5 days (0 points)
                1,              // 1 prior admission (0 points)
                false,          // No procedure (0 points)
                false           // Elective admission (0 points)
            );

            assertNotNull(riskScore);
            assertEquals(RiskScore.RiskType.READMISSION, riskScore.getRiskType());
            assertEquals(0.052, riskScore.getScore(), DELTA); // 5.2% readmission rate
            assertEquals(RiskScore.RiskCategory.LOW, riskScore.getRiskCategory());
            assertFalse(riskScore.isRequiresIntervention());
        }

        @Test
        @DisplayName("Should calculate intermediate readmission risk (HOSPITAL score 5-6)")
        void testIntermediateReadmissionRisk() {
            labs.setHemoglobin(11.0);    // <12 g/dL (1 point)
            labs.setSodium(132.0);        // <135 mEq/L (1 point)

            RiskScore riskScore = engine.calculateReadmissionRisk(
                patientContext,
                labs,
                6,              // LOS ≥5 days (2 points)
                3,              // 2-5 prior admissions (2 points)
                false,          // No procedure (0 points)
                false           // Elective (0 points)
            );
            // Total: 1 + 1 + 2 + 2 = 6 points

            assertEquals(0.087, riskScore.getScore(), DELTA); // 8.7% readmission rate
            assertEquals(RiskScore.RiskCategory.MODERATE, riskScore.getRiskCategory());
            assertTrue(riskScore.isRequiresIntervention());
            assertTrue(riskScore.getRecommendedAction().contains("MODERATE"));
        }

        @Test
        @DisplayName("Should calculate high readmission risk (HOSPITAL score ≥7)")
        void testHighReadmissionRisk() {
            // Cancer patient (oncology service)
            patientContext.addCondition("Metastatic Lung Cancer");

            labs.setHemoglobin(9.5);     // <12 g/dL (1 point)
            labs.setSodium(129.0);        // <135 mEq/L (1 point)

            RiskScore riskScore = engine.calculateReadmissionRisk(
                patientContext,
                labs,
                7,              // LOS ≥5 days (2 points)
                6,              // >5 prior admissions (5 points)
                true,           // Had procedure (1 point)
                true            // Urgent admission (1 point)
            );
            // Total: 2 (oncology) + 1 + 1 + 2 + 5 + 1 + 1 = 13 points (max ~13)

            assertEquals(0.168, riskScore.getScore(), DELTA); // 16.8% readmission rate
            assertEquals(RiskScore.RiskCategory.HIGH, riskScore.getRiskCategory());
            assertTrue(riskScore.isRequiresIntervention());
            assertTrue(riskScore.getRecommendedAction().contains("HIGH RISK"));
            assertTrue(riskScore.getRecommendedAction().toLowerCase().contains("discharge planning"));
        }

        @Test
        @DisplayName("Should handle HOSPITAL score boundary values")
        void testHospitalScoreBoundaries() {
            labs.setHemoglobin(13.0);
            labs.setSodium(138.0);

            // Score = 4 (LOW risk threshold)
            RiskScore score4 = engine.calculateReadmissionRisk(
                patientContext, labs, 3, 3, false, false  // 2 points from admissions
            );
            assertEquals(RiskScore.RiskCategory.LOW, score4.getRiskCategory());

            // Score = 5 (MODERATE risk threshold)
            RiskScore score5 = engine.calculateReadmissionRisk(
                patientContext, labs, 5, 1, true, false  // 2 (LOS) + 1 (procedure) = 3, need 2 more
            );
            assertEquals(RiskScore.RiskCategory.MODERATE, score5.getRiskCategory());

            // Score = 7 (HIGH risk threshold)
            RiskScore score7 = engine.calculateReadmissionRisk(
                patientContext, labs, 5, 6, true, false  // 2 + 5 + 1 = 8, but capped logic
            );
            assertEquals(RiskScore.RiskCategory.HIGH, score7.getRiskCategory());
        }
    }

    @Nested
    @DisplayName("qSOFA Sepsis Screening Tests")
    class QSofaSepsisTests {

        @Test
        @DisplayName("Should identify low sepsis risk (qSOFA = 0)")
        void testLowSepsisRisk() {
            vitals.setRespiratoryRate(16.0);    // <22 (normal)
            vitals.setSystolicBP(125.0);        // >100 (normal)
            int gcs = 15;                       // Alert (normal)

            RiskScore riskScore = engine.calculateSepsisRisk(vitals, gcs);

            assertNotNull(riskScore);
            assertEquals(RiskScore.RiskType.SEPSIS, riskScore.getRiskType());
            assertEquals(0, riskScore.getInputParameters().get("qsofa_total_score"));
            assertEquals(0.05, riskScore.getScore(), DELTA);
            assertEquals(RiskScore.RiskCategory.LOW, riskScore.getRiskCategory());
            assertFalse(riskScore.isRequiresIntervention());
        }

        @Test
        @DisplayName("Should identify moderate sepsis concern (qSOFA = 1)")
        void testModerateSepsisConcern() {
            vitals.setRespiratoryRate(24.0);    // ≥22 (1 point)
            vitals.setSystolicBP(115.0);        // >100 (0 points)
            int gcs = 15;                       // Alert (0 points)

            RiskScore riskScore = engine.calculateSepsisRisk(vitals, gcs);

            assertEquals(1, riskScore.getInputParameters().get("qsofa_total_score"));
            assertEquals(0.15, riskScore.getScore(), DELTA);
            assertEquals(RiskScore.RiskCategory.MODERATE, riskScore.getRiskCategory());
            assertTrue(riskScore.isRequiresIntervention());
            assertTrue(riskScore.getRecommendedAction().contains("MONITOR"));
        }

        @Test
        @DisplayName("Should trigger sepsis alert (qSOFA ≥2)")
        void testSepsisAlert() {
            vitals.setRespiratoryRate(26.0);    // ≥22 (1 point)
            vitals.setSystolicBP(92.0);         // ≤100 (1 point)
            int gcs = 13;                       // <15 (1 point)

            RiskScore riskScore = engine.calculateSepsisRisk(vitals, gcs);

            assertEquals(3, riskScore.getInputParameters().get("qsofa_total_score"));
            assertEquals(0.35, riskScore.getScore(), DELTA);
            assertEquals(RiskScore.RiskCategory.HIGH, riskScore.getRiskCategory());
            assertTrue(riskScore.isRequiresIntervention());
            assertTrue(riskScore.getRecommendedAction().contains("SEPSIS ALERT"));
            assertTrue(riskScore.getRecommendedAction().toLowerCase().contains("antibiotics"));
            assertTrue(riskScore.getRecommendedAction().toLowerCase().contains("lactate"));
        }

        @Test
        @DisplayName("Should evaluate each qSOFA criterion independently")
        void testQSofaCriteriaIndependence() {
            // Test respiratory rate criterion
            vitals.setRespiratoryRate(22.0);    // Exactly 22 (should be 1 point)
            vitals.setSystolicBP(120.0);
            RiskScore rrScore = engine.calculateSepsisRisk(vitals, 15);
            assertTrue((Boolean) rrScore.getInputParameters().get("rr_criteria_met"));

            // Test systolic BP criterion
            vitals.setRespiratoryRate(18.0);
            vitals.setSystolicBP(100.0);        // Exactly 100 (should be 1 point)
            RiskScore bpScore = engine.calculateSepsisRisk(vitals, 15);
            assertTrue((Boolean) bpScore.getInputParameters().get("sbp_criteria_met"));

            // Test GCS criterion
            vitals.setRespiratoryRate(18.0);
            vitals.setSystolicBP(120.0);
            RiskScore gcsScore = engine.calculateSepsisRisk(vitals, 14); // <15 (1 point)
            assertTrue((Boolean) gcsScore.getInputParameters().get("gcs_criteria_met"));
        }

        @Test
        @DisplayName("Should handle maximum qSOFA score (all 3 criteria met)")
        void testMaximumQSofaScore() {
            vitals.setRespiratoryRate(30.0);    // ≥22 (1 point)
            vitals.setSystolicBP(85.0);         // ≤100 (1 point)
            int gcs = 10;                       // <15 (1 point)

            RiskScore riskScore = engine.calculateSepsisRisk(vitals, gcs);

            assertEquals(3, riskScore.getInputParameters().get("qsofa_total_score"));
            assertTrue((Boolean) riskScore.getInputParameters().get("rr_criteria_met"));
            assertTrue((Boolean) riskScore.getInputParameters().get("sbp_criteria_met"));
            assertTrue((Boolean) riskScore.getInputParameters().get("gcs_criteria_met"));
            assertEquals(RiskScore.RiskCategory.HIGH, riskScore.getRiskCategory());
        }
    }

    @Nested
    @DisplayName("MEWS Deterioration Detection Tests")
    class MewsDeteriorationTests {

        @Test
        @DisplayName("Should identify low deterioration risk (MEWS 0-2)")
        void testLowDeteriorationRisk() {
            vitals.setRespiratoryRate(14.0);    // 0 points
            vitals.setHeartRate(75.0);          // 0 points
            vitals.setSystolicBP(125.0);        // 0 points
            vitals.setTemperature(37.0);        // 0 points

            RiskScore riskScore = engine.calculateDeteriorationRisk(vitals, "ALERT");

            assertNotNull(riskScore);
            assertEquals(RiskScore.RiskType.DETERIORATION, riskScore.getRiskType());
            assertEquals(0, riskScore.getInputParameters().get("mews_total_score"));
            assertEquals(0.05, riskScore.getScore(), DELTA);
            assertEquals(RiskScore.RiskCategory.LOW, riskScore.getRiskCategory());
            assertFalse(riskScore.isRequiresIntervention());
        }

        @Test
        @DisplayName("Should identify moderate deterioration risk (MEWS 3-4)")
        void testModerateDeteriorationRisk() {
            vitals.setRespiratoryRate(22.0);    // 2 points
            vitals.setHeartRate(105.0);         // 1 point
            vitals.setSystolicBP(105.0);        // 0 points
            vitals.setTemperature(37.5);        // 0 points

            RiskScore riskScore = engine.calculateDeteriorationRisk(vitals, "ALERT");

            assertEquals(3, riskScore.getInputParameters().get("mews_total_score"));
            assertEquals(0.20, riskScore.getScore(), DELTA);
            assertEquals(RiskScore.RiskCategory.MODERATE, riskScore.getRiskCategory());
            assertTrue(riskScore.isRequiresIntervention());
            assertTrue(riskScore.getRecommendedAction().contains("MODERATE RISK"));
        }

        @Test
        @DisplayName("Should trigger urgent review (MEWS ≥5)")
        void testHighDeteriorationRisk() {
            vitals.setRespiratoryRate(32.0);    // 3 points
            vitals.setHeartRate(125.0);         // 2 points
            vitals.setSystolicBP(95.0);         // 1 point
            vitals.setTemperature(38.6);        // 2 points

            RiskScore riskScore = engine.calculateDeteriorationRisk(vitals, "VOICE"); // 1 point

            int mewsScore = (int) riskScore.getInputParameters().get("mews_total_score");
            assertTrue(mewsScore >= 5, "MEWS score should be ≥5");
            assertEquals(0.50, riskScore.getScore(), DELTA);
            assertEquals(RiskScore.RiskCategory.HIGH, riskScore.getRiskCategory());
            assertTrue(riskScore.isRequiresIntervention());
            assertTrue(riskScore.getRecommendedAction().contains("URGENT"));
            assertTrue(riskScore.getRecommendedAction().toLowerCase().contains("rapid response"));
        }

        @Test
        @DisplayName("Should score AVPU consciousness levels correctly")
        void testAvpuScoring() {
            vitals.setRespiratoryRate(14.0);
            vitals.setHeartRate(75.0);
            vitals.setSystolicBP(120.0);
            vitals.setTemperature(37.0);

            // Alert (0 points)
            RiskScore alertScore = engine.calculateDeteriorationRisk(vitals, "ALERT");
            assertEquals(0.0, alertScore.getFeatureWeights().get("consciousness_level"), DELTA);

            // Voice (1 point)
            RiskScore voiceScore = engine.calculateDeteriorationRisk(vitals, "VOICE");
            assertEquals(1.0, voiceScore.getFeatureWeights().get("consciousness_level"), DELTA);

            // Pain (2 points)
            RiskScore painScore = engine.calculateDeteriorationRisk(vitals, "PAIN");
            assertEquals(2.0, painScore.getFeatureWeights().get("consciousness_level"), DELTA);

            // Unresponsive (3 points)
            RiskScore unresponsiveScore = engine.calculateDeteriorationRisk(vitals, "UNRESPONSIVE");
            assertEquals(3.0, unresponsiveScore.getFeatureWeights().get("consciousness_level"), DELTA);
        }

        @Test
        @DisplayName("Should handle extreme vital sign values")
        void testExtremeMEWSValues() {
            // Critically abnormal vitals
            vitals.setRespiratoryRate(6.0);     // <9 (2 points)
            vitals.setHeartRate(135.0);         // ≥130 (3 points)
            vitals.setSystolicBP(65.0);         // <70 (3 points)
            vitals.setTemperature(34.0);        // <35 (2 points)

            RiskScore riskScore = engine.calculateDeteriorationRisk(vitals, "PAIN"); // 2 points

            int mewsScore = (int) riskScore.getInputParameters().get("mews_total_score");
            assertTrue(mewsScore >= 10, "MEWS score should be very high for critical values");
            assertEquals(RiskScore.RiskCategory.HIGH, riskScore.getRiskCategory());
        }
    }

    @Nested
    @DisplayName("Cross-Calculator Consistency Tests")
    class CrossCalculatorConsistencyTests {

        @Test
        @DisplayName("All calculators should produce validated scores")
        void testAllCalculatorsProduceValidatedScores() {
            // Setup standard patient data
            vitals.setHeartRate(85.0);
            vitals.setSystolicBP(115.0);
            vitals.setDiastolicBP(70.0);
            vitals.setRespiratoryRate(16.0);
            vitals.setTemperature(37.2);
            vitals.setOxygenSaturation(96.0);

            labs.setSodium(140.0);
            labs.setPotassium(4.0);
            labs.setHemoglobin(13.0);

            // Test APACHE III
            RiskScore mortalityScore = engine.calculateMortalityRisk(patientContext, vitals, labs);
            assertTrue(mortalityScore.isValidated());

            // Test HOSPITAL
            RiskScore readmissionScore = engine.calculateReadmissionRisk(patientContext, labs, 4, 1, false, false);
            assertTrue(readmissionScore.isValidated());

            // Test qSOFA
            RiskScore sepsisScore = engine.calculateSepsisRisk(vitals, 15);
            assertTrue(sepsisScore.isValidated());

            // Test MEWS
            RiskScore deteriorationScore = engine.calculateDeteriorationRisk(vitals, "ALERT");
            assertTrue(deteriorationScore.isValidated());
        }

        @Test
        @DisplayName("All calculators should set calculation method and version")
        void testCalculationMetadata() {
            vitals.setHeartRate(85.0);
            vitals.setSystolicBP(115.0);
            vitals.setDiastolicBP(70.0);
            labs.setSodium(140.0);

            RiskScore mortalityScore = engine.calculateMortalityRisk(patientContext, vitals, labs);
            assertNotNull(mortalityScore.getCalculationMethod());
            assertTrue(mortalityScore.getCalculationMethod().contains("APACHE_III"));
            assertNotNull(mortalityScore.getModelVersion());

            RiskScore readmissionScore = engine.calculateReadmissionRisk(patientContext, labs, 3, 0, false, false);
            assertNotNull(readmissionScore.getCalculationMethod());
            assertTrue(readmissionScore.getCalculationMethod().contains("HOSPITAL"));

            RiskScore sepsisScore = engine.calculateSepsisRisk(vitals, 15);
            assertNotNull(sepsisScore.getCalculationMethod());
            assertTrue(sepsisScore.getCalculationMethod().contains("qSOFA"));

            RiskScore deteriorationScore = engine.calculateDeteriorationRisk(vitals, "ALERT");
            assertNotNull(deteriorationScore.getCalculationMethod());
            assertTrue(deteriorationScore.getCalculationMethod().contains("MEWS"));
        }
    }
}
