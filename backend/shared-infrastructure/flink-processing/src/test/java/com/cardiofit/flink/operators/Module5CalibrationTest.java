package com.cardiofit.flink.operators;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import static org.junit.jupiter.api.Assertions.*;

class Module5CalibrationTest {

    @Test
    @DisplayName("Platt calibration returns value in [0, 1]")
    void plattCalibration_inRange() {
        for (String category : new String[]{"sepsis", "deterioration", "readmission", "fall", "mortality"}) {
            for (double raw = 0.0; raw <= 1.0; raw += 0.1) {
                double calibrated = Module5ClinicalScoring.calibrate(raw, category);
                assertTrue(calibrated >= 0.0 && calibrated <= 1.0,
                    category + " calibrated=" + calibrated + " out of range for raw=" + raw);
            }
        }
    }

    @Test
    @DisplayName("Unknown category returns raw score (uncalibrated fallback)")
    void unknownCategory_returnsRawScore() {
        assertEquals(0.75, Module5ClinicalScoring.calibrate(0.75, "unknown_model"), 1e-9);
    }

    @Test
    @DisplayName("Sepsis thresholds: LOW < MODERATE < HIGH < CRITICAL")
    void sepsisThresholds_correctOrder() {
        assertEquals("LOW", Module5ClinicalScoring.classifyRiskLevel(0.10, "sepsis"));
        assertEquals("MODERATE", Module5ClinicalScoring.classifyRiskLevel(0.20, "sepsis"));
        assertEquals("HIGH", Module5ClinicalScoring.classifyRiskLevel(0.40, "sepsis"));
        assertEquals("CRITICAL", Module5ClinicalScoring.classifyRiskLevel(0.65, "sepsis"));
    }

    @Test
    @DisplayName("Sepsis HIGH threshold is lower than deterioration HIGH (clinical asymmetry)")
    void sepsisThreshold_lowerThanDeterioration() {
        // 0.40 should be HIGH for sepsis but only MODERATE for deterioration
        assertEquals("HIGH", Module5ClinicalScoring.classifyRiskLevel(0.40, "sepsis"));
        assertEquals("MODERATE", Module5ClinicalScoring.classifyRiskLevel(0.40, "deterioration"));
    }

    @Test
    @DisplayName("Floating-point boundary: 0.35 exactly is HIGH for sepsis (epsilon tolerance)")
    void floatingPointBoundary_epsilonTolerance() {
        // Lesson 6: 0.35 should classify as HIGH, not MODERATE
        assertEquals("HIGH", Module5ClinicalScoring.classifyRiskLevel(0.35, "sepsis"));
    }

    @Test
    @DisplayName("Readmission thresholds: higher than other categories")
    void readmissionThresholds_higherThanOthers() {
        // 0.50 is HIGH for sepsis, but only MODERATE for readmission
        assertEquals("HIGH", Module5ClinicalScoring.classifyRiskLevel(0.50, "sepsis"));
        assertEquals("MODERATE", Module5ClinicalScoring.classifyRiskLevel(0.50, "readmission"));
    }

    @Test
    @DisplayName("Score 0.0 always classifies as LOW")
    void scoreZero_alwaysLow() {
        for (String cat : new String[]{"sepsis", "deterioration", "readmission", "fall", "mortality"}) {
            assertEquals("LOW", Module5ClinicalScoring.classifyRiskLevel(0.0, cat));
        }
    }

    @Test
    @DisplayName("Score 1.0 always classifies as CRITICAL")
    void scoreOne_alwaysCritical() {
        for (String cat : new String[]{"sepsis", "deterioration", "readmission", "fall", "mortality"}) {
            assertEquals("CRITICAL", Module5ClinicalScoring.classifyRiskLevel(1.0, cat));
        }
    }
}
