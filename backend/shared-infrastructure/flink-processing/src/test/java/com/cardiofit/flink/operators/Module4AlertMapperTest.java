package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module4TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 4 alert-to-PatternEvent mapper functions.
 * These convert analytics engine output into the unified PatternEvent stream.
 *
 * Pattern types produced by the mappers (verified from source):
 *   MEWSAlertToPatternEventMapper        → "MEWS_ALERT"
 *   LabTrendAlertToPatternEventMapper    → "LAB_TREND_ALERT"
 *   VitalVariabilityAlertToPatternEventMapper → "VITAL_VARIABILITY_ALERT"
 */
public class Module4AlertMapperTest {

    @Test
    void mewsMapper_highScore_producesMewsAlertPattern() throws Exception {
        MEWSAlert alert = Module4TestBuilder.mewsAlert("MEWS-001", 7, "HIGH");

        var mapper = new Module4_PatternDetection.MEWSAlertToPatternEventMapper();
        PatternEvent result = mapper.map(alert);

        assertNotNull(result);
        assertEquals("MEWS-001", result.getPatientId());
        assertEquals("MEWS_ALERT", result.getPatternType());
        assertNotNull(result.getPatternDetails());
        assertEquals(7, result.getPatternDetails().get("mews_score"));
    }

    @Test
    void mewsMapper_criticalScore_producesCriticalSeverity() throws Exception {
        // Score >= 5 → CRITICAL, score >= 3 → HIGH, else MODERATE
        MEWSAlert alert = Module4TestBuilder.mewsAlert("MEWS-002", 6, "IMMEDIATE");

        var mapper = new Module4_PatternDetection.MEWSAlertToPatternEventMapper();
        PatternEvent result = mapper.map(alert);

        assertEquals("CRITICAL", result.getSeverity());
        assertEquals(0.95, result.getConfidence(), 0.001);
    }

    @Test
    void labTrendMapper_risingCreatinine_producesLabTrendAlertPattern() throws Exception {
        LabTrendAlert alert = Module4TestBuilder.labTrendAlert("LAB-001", "Creatinine", "RISING");

        var mapper = new Module4_PatternDetection.LabTrendAlertToPatternEventMapper();
        PatternEvent result = mapper.map(alert);

        assertNotNull(result);
        assertEquals("LAB-001", result.getPatientId());
        assertEquals("LAB_TREND_ALERT", result.getPatternType());
        assertNotNull(result.getPatternDetails());
        assertEquals("Creatinine", result.getPatternDetails().get("lab_name"));
        assertEquals("RISING", result.getPatternDetails().get("trend_direction"));
    }

    @Test
    void vitalVariabilityMapper_highBPCV_producesVitalVariabilityAlertPattern() throws Exception {
        VitalVariabilityAlert alert = Module4TestBuilder.vitalVariabilityAlert("VV-001", "systolicBP", 22.5);

        var mapper = new Module4_PatternDetection.VitalVariabilityAlertToPatternEventMapper();
        PatternEvent result = mapper.map(alert);

        assertNotNull(result);
        assertEquals("VV-001", result.getPatientId());
        assertEquals("VITAL_VARIABILITY_ALERT", result.getPatternType());
        assertNotNull(result.getPatternDetails());
        assertEquals("systolicBP", result.getPatternDetails().get("vital_sign_name"));
        assertEquals(22.5, (Double) result.getPatternDetails().get("coefficient_of_variation"), 0.001);
        assertEquals(0.85, result.getConfidence(), 0.001);
        assertEquals("HIGH", result.getSeverity());
    }
}
