package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module5TestBuilder;
import com.cardiofit.flink.models.PatientMLState;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Validates that real production CDS events produce valid feature vectors.
 * Catches schema mismatches before they reach ONNX inference.
 *
 * This is the HIGHEST PRIORITY test — run first, fix first.
 */
class Module5RealDataMappingTest {

    @Test
    @DisplayName("Production vital keys resolve to non-zero features")
    void productionVitalKeys_resolveCorrectly() {
        PatientMLState state = Module5TestBuilder.stablePatientState("P001");

        float[] features = Module5FeatureExtractor.extractFeatures(state);

        // Vitals should be non-zero (indices 0-5)
        assertNotEquals(0.0f, features[0], "heartrate should resolve");
        assertNotEquals(0.0f, features[1], "systolicbloodpressure should resolve");
        assertNotEquals(0.0f, features[2], "diastolicbloodpressure should resolve");
        assertNotEquals(0.0f, features[3], "respiratoryrate should resolve");
        assertNotEquals(0.0f, features[4], "oxygensaturation should resolve");
        assertNotEquals(0.0f, features[5], "temperature should resolve");
    }

    @Test
    @DisplayName("Snake_case vital keys normalize to production format")
    void snakeCaseVitalKeys_normalizeToProductionFormat() {
        PatientMLState state = new PatientMLState();
        state.setPatientId("P002");
        // Simulate Module4TestBuilder format (snake_case)
        state.setLatestVitals(Map.of(
            "heart_rate", 90.0,
            "systolic_bp", 130.0,
            "diastolic_bp", 85.0,
            "respiratory_rate", 18.0,
            "oxygen_saturation", 95.0,
            "temperature", 37.5
        ));
        state.setLatestLabs(Collections.emptyMap());
        state.setRiskIndicators(Collections.emptyMap());
        state.setActiveAlerts(Collections.emptyMap());

        float[] features = Module5FeatureExtractor.extractFeatures(state);

        // All vitals should resolve despite snake_case input
        assertNotEquals(0.0f, features[0], "heart_rate → heartrate should resolve");
        assertNotEquals(0.0f, features[1], "systolic_bp → systolicbloodpressure should resolve");
    }

    @Test
    @DisplayName("Demographic fields excluded from vital extraction")
    void demographicFields_excludedFromVitals() {
        Map<String, Double> vitalsWithDemographics = new HashMap<>(Map.of(
            "heartrate", 80.0,
            "systolicbloodpressure", 120.0,
            "age", 65.0,
            "weight", 78.0
        ));

        Map<String, Double> normalized = Module5FeatureExtractor.normalizeVitalKeys(vitalsWithDemographics);

        assertTrue(normalized.containsKey("heartrate"), "heartrate should be kept");
        assertFalse(normalized.containsKey("age"), "age should be excluded");
        assertFalse(normalized.containsKey("weight"), "weight should be excluded");
    }

    @Test
    @DisplayName("Feature vector has no NaN or Infinity values")
    void featureVector_noNaNOrInfinity() {
        // Test all patient scenarios
        String[] scenarios = {"stable", "sepsis", "aki", "druglab", "sparse"};
        PatientMLState[] states = {
            Module5TestBuilder.stablePatientState("P1"),
            Module5TestBuilder.sepsisPatientState("P2"),
            Module5TestBuilder.akiPatientState("P3"),
            Module5TestBuilder.drugLabPatientState("P4"),
            Module5TestBuilder.sparsePatientState("P5")
        };

        for (int s = 0; s < states.length; s++) {
            float[] features = Module5FeatureExtractor.extractFeatures(states[s]);
            assertEquals(Module5FeatureExtractor.FEATURE_COUNT, features.length,
                scenarios[s] + ": wrong feature count");

            for (int i = 0; i < features.length; i++) {
                assertFalse(Float.isNaN(features[i]),
                    scenarios[s] + ": NaN at index " + i);
                assertFalse(Float.isInfinite(features[i]),
                    scenarios[s] + ": Infinity at index " + i);
            }
        }
    }

    @Test
    @DisplayName("Missing labs produce -1.0 sentinel (not 0.0 or NaN)")
    void missingLabs_produceSentinelValue() {
        PatientMLState state = Module5TestBuilder.sparsePatientState("P006");

        float[] features = Module5FeatureExtractor.extractFeatures(state);

        // Lab features [45-49] should be -1.0 when labs are empty
        assertEquals(-1.0f, features[45], "missing lactate should be -1.0");
        assertEquals(-1.0f, features[46], "missing creatinine should be -1.0");
        assertEquals(-1.0f, features[47], "missing potassium should be -1.0");
        assertEquals(-1.0f, features[48], "missing wbc should be -1.0");
        assertEquals(-1.0f, features[49], "missing platelets should be -1.0");
    }

    @Test
    @DisplayName("AKI patient alerts produce non-zero alert features")
    void akiPatient_alertFeaturesPopulated() {
        PatientMLState state = Module5TestBuilder.akiPatientState("P007");

        float[] features = Module5FeatureExtractor.extractFeatures(state);

        // AKI_RISK alert present (index 51)
        assertEquals(1.0f, features[51], "AKI_RISK alert should be present");
        // Alert max severity should be non-zero (CRITICAL hyperkalemia)
        assertTrue(features[54] > 0.0f, "alert max severity should be non-zero");
    }

    @Test
    @DisplayName("Null PatientMLState produces zero vector (no crash)")
    void nullState_producesZeroVector() {
        float[] features = Module5FeatureExtractor.extractFeatures(null);

        assertEquals(Module5FeatureExtractor.FEATURE_COUNT, features.length);
        for (float f : features) {
            assertEquals(0.0f, f, "null state should produce all zeros");
        }
    }
}
