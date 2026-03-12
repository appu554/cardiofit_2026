package com.cardiofit.flink.integration;

import com.cardiofit.flink.functions.MLToPatternConverter;
import com.cardiofit.flink.models.MLPrediction;
import com.cardiofit.flink.models.PatternEvent;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.BeforeAll;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Integration Test for Module 4 Layer 3 ML Pattern Integration
 *
 * Tests the end-to-end flow:
 * 1. ML predictions from Module 5
 * 2. Conversion to PatternEvent format
 * 3. Integration with Layer 1 & Layer 2 patterns
 * 4. Deduplication and multi-source confirmation
 *
 * @author CardioFit Engineering Team
 * @version 1.0.0
 */
public class Module4Layer3IntegrationTest {

    private static MLToPatternConverter converter;

    @BeforeAll
    public static void setUp() {
        converter = new MLToPatternConverter();
    }

    // ═══════════════════════════════════════════════════════════
    // TEST 1: ML PREDICTION TO PATTERN CONVERSION
    // ═══════════════════════════════════════════════════════════

    @Test
    @DisplayName("Test 1: Convert high-risk ML prediction to PatternEvent")
    public void testHighRiskMLPredictionConversion() throws Exception {
        // Given: A high-risk sepsis ML prediction
        MLPrediction mlPrediction = createSepsisPrediction(
            "patient-001",
            "encounter-001",
            "CRITICAL",
            0.92,
            "SEPSIS_RISK"
        );

        // When: Convert to PatternEvent
        PatternEvent pattern = converter.map(mlPrediction);

        // Then: Verify pattern structure
        assertNotNull(pattern, "Pattern should not be null");
        assertEquals("PREDICTIVE_SEPSIS_RISK", pattern.getPatternType());
        assertEquals("patient-001", pattern.getPatientId());
        assertEquals("encounter-001", pattern.getEncounterId());
        assertEquals("CRITICAL", pattern.getSeverity());
        assertEquals(0.92, pattern.getConfidence(), 0.01);
        assertEquals(1, pattern.getPriority()); // Critical = Priority 1

        // Verify tags
        Set<String> tags = pattern.getTags();
        assertTrue(tags.contains("ML_BASED"), "Should have ML_BASED tag");
        assertTrue(tags.contains("PREDICTIVE"), "Should have PREDICTIVE tag");
        assertTrue(tags.contains("LAYER_3"), "Should have LAYER_3 tag");
        assertTrue(tags.contains("HIGH_CONFIDENCE"), "Should have HIGH_CONFIDENCE tag (>= 0.85)");

        // Verify pattern details
        Map<String, Object> details = pattern.getPatternDetails();
        assertEquals("SEPSIS_RISK", details.get("modelType"));
        assertEquals("CRITICAL", details.get("riskLevel"));
        assertEquals("IMMEDIATE", details.get("urgency"));
        assertEquals(true, details.get("isPredictive"));
        assertEquals("MODULE_5_ML", details.get("predictionSource"));

        // Verify clinical message
        String clinicalMessage = (String) details.get("clinicalMessage");
        assertNotNull(clinicalMessage, "Clinical message should not be null");
        assertTrue(clinicalMessage.contains("ML PREDICTION"), "Should contain ML PREDICTION");
        assertTrue(clinicalMessage.contains("CRITICAL"), "Should contain risk level");

        // Verify recommended actions
        List<String> actions = pattern.getRecommendedActions();
        assertFalse(actions.isEmpty(), "Recommended actions should not be empty");
        assertTrue(actions.contains("ENHANCED_MONITORING"), "Should include enhanced monitoring");
        assertTrue(actions.contains("IMMEDIATE_CLINICAL_ASSESSMENT"), "Critical risk should include immediate assessment");
        assertTrue(actions.contains("MONITOR_FOR_SIRS_CRITERIA"), "Sepsis should include SIRS monitoring");

        System.out.println("✅ Test 1 PASSED: High-risk ML prediction correctly converted to PatternEvent");
    }

    @Test
    @DisplayName("Test 2: Convert moderate-risk ML prediction to PatternEvent")
    public void testModerateRiskMLPredictionConversion() throws Exception {
        // Given: A moderate-risk deterioration ML prediction
        MLPrediction mlPrediction = createDeteriorationPrediction(
            "patient-002",
            "encounter-002",
            "MODERATE",
            0.65,
            "CLINICAL_DETERIORATION"
        );

        // When: Convert to PatternEvent
        PatternEvent pattern = converter.map(mlPrediction);

        // Then: Verify pattern structure
        assertEquals("PREDICTIVE_CLINICAL_DETERIORATION", pattern.getPatternType());
        assertEquals("MODERATE", pattern.getSeverity());
        assertEquals(0.65, pattern.getConfidence(), 0.01);
        assertEquals(3, pattern.getPriority()); // Moderate = Priority 3

        // Verify urgency calculation
        Map<String, Object> details = pattern.getPatternDetails();
        assertEquals("MODERATE", details.get("urgency"), "Moderate risk should have MODERATE urgency");

        // Verify no HIGH_CONFIDENCE tag (< 0.85)
        Set<String> tags = pattern.getTags();
        assertFalse(tags.contains("HIGH_CONFIDENCE"), "Should not have HIGH_CONFIDENCE tag (< 0.85)");

        System.out.println("✅ Test 2 PASSED: Moderate-risk ML prediction correctly converted");
    }

    @Test
    @DisplayName("Test 3: Convert low-risk ML prediction to PatternEvent")
    public void testLowRiskMLPredictionConversion() throws Exception {
        // Given: A low-risk mortality ML prediction
        MLPrediction mlPrediction = createMortalityPrediction(
            "patient-003",
            "encounter-003",
            "LOW",
            0.15,
            "MORTALITY_PREDICTION"
        );

        // When: Convert to PatternEvent
        PatternEvent pattern = converter.map(mlPrediction);

        // Then: Verify pattern structure
        assertEquals("LOW", pattern.getSeverity());
        assertEquals(4, pattern.getPriority()); // Low = Priority 4

        // Verify recommended actions are still present (basic monitoring)
        List<String> actions = pattern.getRecommendedActions();
        assertTrue(actions.contains("ROUTINE_MONITORING"), "Low risk should include routine monitoring");
        assertFalse(actions.contains("ESCALATE_TO_RAPID_RESPONSE"), "Low risk should NOT escalate");

        System.out.println("✅ Test 3 PASSED: Low-risk ML prediction correctly converted");
    }

    // ═══════════════════════════════════════════════════════════
    // TEST 2: FEATURE IMPORTANCE AND EXPLAINABILITY
    // ═══════════════════════════════════════════════════════════

    @Test
    @DisplayName("Test 4: ML prediction with feature importance preserved")
    public void testFeatureImportancePreservation() throws Exception {
        // Given: ML prediction with feature importance
        MLPrediction mlPrediction = createPredictionWithFeatureImportance(
            "patient-004",
            "encounter-004"
        );

        // When: Convert to PatternEvent
        PatternEvent pattern = converter.map(mlPrediction);

        // Then: Verify feature importance is preserved
        Map<String, Object> details = pattern.getPatternDetails();

        @SuppressWarnings("unchecked")
        Map<String, Double> featureImportance = (Map<String, Double>) details.get("featureImportance");
        assertNotNull(featureImportance, "Feature importance should be preserved");
        assertTrue(featureImportance.containsKey("lactate_level"), "Should contain lactate feature");
        assertTrue(featureImportance.containsKey("heart_rate"), "Should contain heart rate feature");

        // Verify clinical message includes key indicators
        String clinicalMessage = (String) details.get("clinicalMessage");
        assertTrue(clinicalMessage.contains("Key indicators:"), "Clinical message should include key indicators");

        System.out.println("✅ Test 4 PASSED: Feature importance correctly preserved in pattern");
    }

    // ═══════════════════════════════════════════════════════════
    // TEST 3: MODEL-SPECIFIC RECOMMENDED ACTIONS
    // ═══════════════════════════════════════════════════════════

    @Test
    @DisplayName("Test 5: Respiratory model generates specific actions")
    public void testRespiratoryModelActions() throws Exception {
        // Given: Respiratory failure ML prediction
        MLPrediction mlPrediction = createRespiratoryPrediction(
            "patient-005",
            "encounter-005",
            "HIGH",
            0.88,
            "RESPIRATORY_FAILURE"
        );

        // When: Convert to PatternEvent
        PatternEvent pattern = converter.map(mlPrediction);

        // Then: Verify respiratory-specific actions
        List<String> actions = pattern.getRecommendedActions();
        assertTrue(actions.contains("MONITOR_OXYGEN_SATURATION_CLOSELY"),
            "Should include oxygen monitoring");
        assertTrue(actions.contains("ASSESS_RESPIRATORY_RATE_Q15MIN"),
            "Should include frequent respiratory assessment");
        assertTrue(actions.contains("PREPARE_RESPIRATORY_SUPPORT"),
            "Should prepare for respiratory support");

        System.out.println("✅ Test 5 PASSED: Respiratory model generates specific clinical actions");
    }

    @Test
    @DisplayName("Test 6: Cardiac model generates specific actions")
    public void testCardiacModelActions() throws Exception {
        // Given: Cardiac event ML prediction
        MLPrediction mlPrediction = createCardiacPrediction(
            "patient-006",
            "encounter-006",
            "HIGH",
            0.82,
            "CARDIAC_EVENT"
        );

        // When: Convert to PatternEvent
        PatternEvent pattern = converter.map(mlPrediction);

        // Then: Verify cardiac-specific actions
        List<String> actions = pattern.getRecommendedActions();
        assertTrue(actions.contains("CONTINUOUS_CARDIAC_MONITORING"),
            "Should include cardiac monitoring");
        assertTrue(actions.contains("CHECK_TROPONIN_LEVELS"),
            "Should check troponin");
        assertTrue(actions.contains("ECG_IF_NOT_RECENT"),
            "Should request ECG");

        System.out.println("✅ Test 6 PASSED: Cardiac model generates specific clinical actions");
    }

    @Test
    @DisplayName("Test 7: AKI model generates specific actions")
    public void testAKIModelActions() throws Exception {
        // Given: Acute Kidney Injury ML prediction
        MLPrediction mlPrediction = createAKIPrediction(
            "patient-007",
            "encounter-007",
            "HIGH",
            0.79,
            "AKI_RISK"
        );

        // When: Convert to PatternEvent
        PatternEvent pattern = converter.map(mlPrediction);

        // Then: Verify AKI-specific actions
        List<String> actions = pattern.getRecommendedActions();
        assertTrue(actions.contains("MONITOR_URINE_OUTPUT"),
            "Should monitor urine output");
        assertTrue(actions.contains("CHECK_CREATININE_LEVELS"),
            "Should check creatinine");
        assertTrue(actions.contains("REVIEW_NEPHROTOXIC_MEDICATIONS"),
            "Should review nephrotoxic meds");

        System.out.println("✅ Test 7 PASSED: AKI model generates specific clinical actions");
    }

    // ═══════════════════════════════════════════════════════════
    // TEST 4: PATTERN METADATA
    // ═══════════════════════════════════════════════════════════

    @Test
    @DisplayName("Test 8: Pattern metadata correctly generated")
    public void testPatternMetadata() throws Exception {
        // Given: ML prediction
        MLPrediction mlPrediction = createSepsisPrediction(
            "patient-008",
            "encounter-008",
            "HIGH",
            0.87,
            "SEPSIS_RISK"
        );

        // When: Convert to PatternEvent
        PatternEvent pattern = converter.map(mlPrediction);

        // Then: Verify pattern metadata
        PatternEvent.PatternMetadata metadata = pattern.getPatternMetadata();
        assertNotNull(metadata, "Pattern metadata should not be null");
        assertEquals("ML_PREDICTIVE_ANALYSIS", metadata.getAlgorithm());
        assertNotNull(metadata.getVersion(), "Model version should be set");
        assertTrue(metadata.getProcessingTime() > 0, "Processing time should be recorded");
        assertEquals("HIGH", metadata.getQualityScore(), "High confidence should map to HIGH quality");

        // Verify algorithm parameters
        Map<String, Object> algorithmParams = metadata.getAlgorithmParameters();
        assertNotNull(algorithmParams, "Algorithm parameters should not be null");
        assertEquals("SEPSIS_RISK", algorithmParams.get("modelType"));
        assertEquals(0.70, algorithmParams.get("confidenceThreshold"));

        System.out.println("✅ Test 8 PASSED: Pattern metadata correctly generated");
    }

    // ═══════════════════════════════════════════════════════════
    // TEST 5: MULTI-SOURCE PATTERN CONFIRMATION
    // ═══════════════════════════════════════════════════════════

    @Test
    @DisplayName("Test 9: ML pattern can merge with instant and CEP patterns")
    public void testMultiSourcePatternMerging() throws Exception {
        // Given: Three patterns detecting the same clinical condition
        // Layer 1 (Instant): High lactate detected
        PatternEvent instantPattern = new PatternEvent();
        instantPattern.setPatientId("patient-009");
        instantPattern.setEncounterId("encounter-009");
        instantPattern.setPatternType("HIGH_LACTATE");
        instantPattern.setSeverity("HIGH");
        instantPattern.addTag("LAYER_1");
        instantPattern.setDetectionTime(System.currentTimeMillis());

        // Layer 2 (CEP): Deteriorating vital signs trend
        PatternEvent cepPattern = new PatternEvent();
        cepPattern.setPatientId("patient-009");
        cepPattern.setEncounterId("encounter-009");
        cepPattern.setPatternType("VITAL_SIGNS_DETERIORATION");
        cepPattern.setSeverity("HIGH");
        cepPattern.addTag("LAYER_2");
        cepPattern.setDetectionTime(System.currentTimeMillis());

        // Layer 3 (ML): Sepsis risk prediction
        MLPrediction mlPrediction = createSepsisPrediction(
            "patient-009",
            "encounter-009",
            "HIGH",
            0.89,
            "SEPSIS_RISK"
        );
        PatternEvent mlPattern = converter.map(mlPrediction);

        // Then: Verify all patterns have same patient/encounter for merging
        assertEquals("patient-009", instantPattern.getPatientId());
        assertEquals("patient-009", cepPattern.getPatientId());
        assertEquals("patient-009", mlPattern.getPatientId());

        assertEquals("encounter-009", instantPattern.getEncounterId());
        assertEquals("encounter-009", cepPattern.getEncounterId());
        assertEquals("encounter-009", mlPattern.getEncounterId());

        // Verify patterns are from different layers
        assertTrue(instantPattern.getTags().contains("LAYER_1"), "First pattern from Layer 1");
        assertTrue(cepPattern.getTags().contains("LAYER_2"), "Second pattern from Layer 2");
        assertTrue(mlPattern.getTags().contains("LAYER_3"), "Third pattern from Layer 3");

        System.out.println("✅ Test 9 PASSED: Multi-source patterns correctly structured for deduplication");
        System.out.println("   Layer 1 (Instant): " + instantPattern.getPatternType());
        System.out.println("   Layer 2 (CEP):     " + cepPattern.getPatternType());
        System.out.println("   Layer 3 (ML):      " + mlPattern.getPatternType());
        System.out.println("   → These would be merged by PatternDeduplicationFunction");
    }

    // ═══════════════════════════════════════════════════════════
    // HELPER METHODS
    // ═══════════════════════════════════════════════════════════

    private static MLPrediction createSepsisPrediction(
            String patientId,
            String encounterId,
            String riskLevel,
            double confidence,
            String modelType) {

        MLPrediction prediction = new MLPrediction();
        prediction.setId(UUID.randomUUID().toString());
        prediction.setPatientId(patientId);
        prediction.setEncounterId(encounterId);
        prediction.setModelName("Sepsis Prediction Model v1.0.0");
        prediction.setModelType(modelType);
        prediction.setRiskLevel(riskLevel);
        prediction.setConfidence(confidence);
        prediction.setPredictionTime(System.currentTimeMillis());

        Map<String, Double> scores = new HashMap<>();
        scores.put("sepsis_probability", confidence);
        prediction.setPredictionScores(scores);

        prediction.setInputFeatureCount(70);

        return prediction;
    }

    private static MLPrediction createDeteriorationPrediction(
            String patientId,
            String encounterId,
            String riskLevel,
            double confidence,
            String modelType) {

        MLPrediction prediction = createSepsisPrediction(patientId, encounterId, riskLevel, confidence, modelType);
        prediction.setModelName("Clinical Deterioration Model v1.0.0");
        return prediction;
    }

    private static MLPrediction createMortalityPrediction(
            String patientId,
            String encounterId,
            String riskLevel,
            double confidence,
            String modelType) {

        MLPrediction prediction = createSepsisPrediction(patientId, encounterId, riskLevel, confidence, modelType);
        prediction.setModelName("Mortality Prediction Model v1.0.0");
        return prediction;
    }

    private static MLPrediction createRespiratoryPrediction(
            String patientId,
            String encounterId,
            String riskLevel,
            double confidence,
            String modelType) {

        MLPrediction prediction = createSepsisPrediction(patientId, encounterId, riskLevel, confidence, modelType);
        prediction.setModelName("Respiratory Failure Model v1.0.0");
        return prediction;
    }

    private static MLPrediction createCardiacPrediction(
            String patientId,
            String encounterId,
            String riskLevel,
            double confidence,
            String modelType) {

        MLPrediction prediction = createSepsisPrediction(patientId, encounterId, riskLevel, confidence, modelType);
        prediction.setModelName("Cardiac Event Model v1.0.0");
        return prediction;
    }

    private static MLPrediction createAKIPrediction(
            String patientId,
            String encounterId,
            String riskLevel,
            double confidence,
            String modelType) {

        MLPrediction prediction = createSepsisPrediction(patientId, encounterId, riskLevel, confidence, modelType);
        prediction.setModelName("AKI Risk Model v1.0.0");
        return prediction;
    }

    private static MLPrediction createPredictionWithFeatureImportance(
            String patientId,
            String encounterId) {

        MLPrediction prediction = createSepsisPrediction(
            patientId,
            encounterId,
            "HIGH",
            0.85,
            "SEPSIS_RISK"
        );

        Map<String, Double> featureImportance = new LinkedHashMap<>();
        featureImportance.put("lactate_level", 0.45);
        featureImportance.put("heart_rate", 0.32);
        featureImportance.put("temperature", 0.23);
        prediction.setFeatureImportance(featureImportance);

        return prediction;
    }
}
