package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module4TestBuilder;
import com.cardiofit.flink.models.PatternEvent;
import com.cardiofit.flink.models.SemanticEvent;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 4 CEP select functions.
 * Each function receives a CEP match map and produces a PatternEvent.
 * These are pure functions — no Flink runtime needed.
 */
public class Module4CEPSelectFunctionTest {

    // ── DeteriorationPatternSelectFunction ─────────────────────

    @Test
    void deterioration_producesHighSeverityPattern() throws Exception {
        var fn = new Module4_PatternDetection.DeteriorationPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.deteriorationMatchMap("PAT-D001");

        PatternEvent result = fn.select(matchMap);

        assertEquals("CLINICAL_DETERIORATION", result.getPatternType());
        assertEquals("PAT-D001", result.getPatientId());
        assertEquals("HIGH", result.getSeverity());
        assertEquals(0.85, result.getConfidence(), 0.01);
        assertEquals(3, result.getInvolvedEvents().size(), "Should reference baseline + warning + critical events");

        // Verify pattern details contain deterioration metrics
        assertNotNull(result.getPatternDetails().get("deterioration_rate"));
        assertNotNull(result.getPatternDetails().get("timespan_hours"));
        double rate = (double) result.getPatternDetails().get("deterioration_rate");
        assertTrue(rate > 0, "Deterioration rate should be positive (worsening): baseline=0.19, critical=0.87");

        // Verify recommended actions
        assertTrue(result.getRecommendedActions().contains("IMMEDIATE_ASSESSMENT_REQUIRED"));
        assertTrue(result.getRecommendedActions().contains("ESCALATE_TO_PHYSICIAN"));
    }

    // ── CrossDomainDeteriorationSelectFunction ─────────────────

    @Test
    void crossDomain_detectsGlycaemicAndHemodynamicDecline() throws Exception {
        var fn = new Module4_PatternDetection.CrossDomainDeteriorationSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.crossDomainMatchMap("PAT-CD001");

        PatternEvent result = fn.select(matchMap);

        assertEquals("CROSS_DOMAIN_DECLINE", result.getPatternType());
        assertEquals("PAT-CD001", result.getPatientId());
        assertEquals("HIGH", result.getSeverity());
        assertEquals(2, result.getInvolvedEvents().size());

        // Verify cross-domain details
        assertEquals("DECLINING", result.getPatternDetails().get("glycaemic_trajectory"));
        assertEquals("DECLINING", result.getPatternDetails().get("hemodynamic_trajectory"));
        assertEquals("GLYCAEMIC", result.getPatternDetails().get("glycaemic_domain"));
        assertEquals("HEMODYNAMIC", result.getPatternDetails().get("hemodynamic_domain"));

        // V4-specific: MHRI recomputation action
        assertTrue(result.getRecommendedActions().contains("MHRI_RECOMPUTATION_NEEDED"));
    }

    // ── MedicationPatternSelectFunction ────────────────────────

    @Test
    void medication_missedDose_producesModerateAlert() throws Exception {
        var fn = new Module4_PatternDetection.MedicationPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.medicationMatchMap("PAT-M001", true);

        PatternEvent result = fn.select(matchMap);

        assertEquals("MEDICATION_ADHERENCE", result.getPatternType());
        assertEquals("PAT-M001", result.getPatientId());
        assertEquals("MODERATE", result.getSeverity());
        assertEquals("MISSED", result.getPatternDetails().get("medication_status"));
        assertTrue(result.getRecommendedActions().contains("VERIFY_PATIENT_STATUS"));
    }

    @Test
    void medication_administered_producesLowAlert() throws Exception {
        var fn = new Module4_PatternDetection.MedicationPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.medicationMatchMap("PAT-M002", false);

        PatternEvent result = fn.select(matchMap);

        assertEquals("MEDICATION_ADHERENCE", result.getPatternType());
        assertEquals("LOW", result.getSeverity());
        assertEquals("ADMINISTERED", result.getPatternDetails().get("medication_status"));
        // No recommended actions are set when medication is administered (not missed)
        assertTrue(result.getRecommendedActions().isEmpty(),
            "No recommended actions expected for administered medication");
    }

    // ── VitalTrendPatternSelectFunction ────────────────────────

    @Test
    void vitalTrend_deteriorating_producesHighSeverity() throws Exception {
        var fn = new Module4_PatternDetection.VitalTrendPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.vitalTrendMatchMap("PAT-VT001", true);

        PatternEvent result = fn.select(matchMap);

        assertEquals("VITAL_SIGNS_TREND", result.getPatternType());
        assertEquals("HIGH", result.getSeverity());
        assertEquals("DETERIORATING", result.getPatternDetails().get("trend_direction"));
        // reading_count is stored as Integer(3) — use intValue to avoid ambiguous assertEquals overload
        assertEquals(3, ((Number) result.getPatternDetails().get("reading_count")).intValue());
        assertTrue(result.getRecommendedActions().contains("INCREASE_VITAL_MONITORING"));
    }

    @Test
    void vitalTrend_improving_producesLowSeverity() throws Exception {
        var fn = new Module4_PatternDetection.VitalTrendPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.vitalTrendMatchMap("PAT-VT002", false);

        PatternEvent result = fn.select(matchMap);

        assertEquals("VITAL_SIGNS_TREND", result.getPatternType());
        assertEquals("LOW", result.getSeverity());
        assertEquals("IMPROVING", result.getPatternDetails().get("trend_direction"));
        // No recommended actions are set when trend is improving
        assertTrue(result.getRecommendedActions().isEmpty(),
            "No recommended actions expected for improving vital trend");
    }

    // ── PathwayCompliancePatternSelectFunction ─────────────────

    @Test
    void pathway_completed_producesLowSeverity() throws Exception {
        var fn = new Module4_PatternDetection.PathwayCompliancePatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.pathwayMatchMap("PAT-P001");

        PatternEvent result = fn.select(matchMap);

        assertEquals("PATHWAY_COMPLIANCE", result.getPatternType());
        assertEquals("PAT-P001", result.getPatientId());
        assertEquals("LOW", result.getSeverity());
        assertTrue((boolean) result.getPatternDetails().get("pathway_completed"));
        assertNotNull(result.getPatternDetails().get("time_to_assessment"));
        assertNotNull(result.getPatternDetails().get("time_to_intervention"));
    }
}
