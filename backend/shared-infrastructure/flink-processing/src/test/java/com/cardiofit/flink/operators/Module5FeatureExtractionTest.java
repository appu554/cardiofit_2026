package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module5TestBuilder;
import com.cardiofit.flink.models.PatientMLState;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

class Module5FeatureExtractionTest {

    @Test
    @DisplayName("normalize: value at min returns 0.0")
    void normalize_atMin_returnsZero() {
        assertEquals(0.0f, Module5FeatureExtractor.normalize(30, 30, 200));
    }

    @Test
    @DisplayName("normalize: value at max returns 1.0")
    void normalize_atMax_returnsOne() {
        assertEquals(1.0f, Module5FeatureExtractor.normalize(200, 30, 200));
    }

    @Test
    @DisplayName("normalize: value below min clamps to 0.0")
    void normalize_belowMin_clampsToZero() {
        assertEquals(0.0f, Module5FeatureExtractor.normalize(10, 30, 200));
    }

    @Test
    @DisplayName("normalize: value above max clamps to 1.0")
    void normalize_aboveMax_clampsToOne() {
        assertEquals(1.0f, Module5FeatureExtractor.normalize(300, 30, 200));
    }

    @Test
    @DisplayName("normalize: equal min/max returns 0.0 (no division by zero)")
    void normalize_equalMinMax_returnsZero() {
        assertEquals(0.0f, Module5FeatureExtractor.normalize(5, 5, 5));
    }

    @Test
    @DisplayName("safeRiskFlag: null map returns 0.0")
    void safeRiskFlag_nullMap_returnsZero() {
        assertEquals(0.0f, Module5FeatureExtractor.safeRiskFlag(null, "tachycardia"));
    }

    @Test
    @DisplayName("safeRiskFlag: missing key returns 0.0")
    void safeRiskFlag_missingKey_returnsZero() {
        assertEquals(0.0f, Module5FeatureExtractor.safeRiskFlag(
            Map.of("fever", true), "tachycardia"));
    }

    @Test
    @DisplayName("safeRiskFlag: non-Boolean value returns 0.0 (Gap 2 resilience)")
    void safeRiskFlag_nonBooleanValue_returnsZero() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("onAnticoagulation", "maybe");
        assertEquals(0.0f, Module5FeatureExtractor.safeRiskFlag(risk, "onAnticoagulation"));
    }

    @Test
    @DisplayName("safeLabFeature: null labs returns -1.0 sentinel")
    void safeLabFeature_nullLabs_returnsSentinel() {
        assertEquals(-1.0f, Module5FeatureExtractor.safeLabFeature(null, "lactate", 0, 20));
    }

    @Test
    @DisplayName("safeLabFeature: null value returns -1.0 sentinel (Gap 1)")
    void safeLabFeature_nullValue_returnsSentinel() {
        Map<String, Double> labs = new HashMap<>();
        labs.put("lactate", null);
        assertEquals(-1.0f, Module5FeatureExtractor.safeLabFeature(labs, "lactate", 0, 20));
    }

    @Test
    @DisplayName("Sepsis patient: elevated features produce high feature values")
    void sepsisPatient_producesHighFeatureValues() {
        PatientMLState state = Module5TestBuilder.sepsisPatientState("P-sepsis");

        float[] features = Module5FeatureExtractor.extractFeatures(state);

        // NEWS2=9 → normalize(9, 0, 20) = 0.45
        assertTrue(features[6] > 0.4f, "NEWS2 should be elevated");
        // qSOFA=2 → normalize(2, 0, 3) = 0.667
        assertTrue(features[7] > 0.6f, "qSOFA should be elevated");
        // Tachycardia flag
        assertEquals(1.0f, features[35], "tachycardia should be set");
        // Elevated lactate flag
        assertEquals(1.0f, features[39], "elevatedLactate should be set");
        // Sepsis alert
        assertEquals(1.0f, features[50], "SEPSIS_PATTERN alert should be present");
    }

    @Test
    @DisplayName("NEWS2 slope: increasing history returns positive slope")
    void news2Slope_increasingHistory_positiveSlope() {
        PatientMLState state = new PatientMLState();
        state.setPatientId("P-slope");
        state.pushNews2(2);
        state.pushNews2(4);
        state.pushNews2(6);
        state.pushNews2(8);

        assertTrue(state.news2Slope() > 0, "Increasing NEWS2 should produce positive slope");
    }

    @Test
    @DisplayName("Feature count is exactly 55")
    void featureCount_isExactly55() {
        PatientMLState state = Module5TestBuilder.stablePatientState("P-count");
        float[] features = Module5FeatureExtractor.extractFeatures(state);
        assertEquals(55, features.length);
        assertEquals(55, Module5FeatureExtractor.FEATURE_COUNT);
    }
}
