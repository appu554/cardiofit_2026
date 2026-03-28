package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module4TestBuilder;
import com.cardiofit.flink.models.PatternEvent;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 4 PatternClassificationFunction.
 * Verifies pattern type classification and tag enrichment.
 *
 * Key behaviours (from implementation):
 *   CLINICAL_DETERIORATION  → no tag added (handled in main stream)
 *   PATHWAY_COMPLIANCE      → adds "pathway_adherence" tag
 *   ANOMALY_DETECTION       → adds "anomaly" tag
 *   TREND_ANALYSIS          → adds "trend" tag
 */
public class Module4ClassificationTest {

    // ── CLINICAL_DETERIORATION ────────────────────────────────────────────────

    @Test
    void classification_deterioration_enrichedCorrectly() throws Exception {
        var fn = new Module4_PatternDetection.PatternClassificationFunction();
        PatternEvent input = Module4TestBuilder.deteriorationPattern("CLASS-001", "HIGH", 0.85);

        PatternEvent result = fn.map(input);

        assertNotNull(result);
        assertEquals("CLINICAL_DETERIORATION", result.getPatternType());
        assertEquals("CLASS-001", result.getPatientId());
    }

    @Test
    void classification_deterioration_noTagAdded() throws Exception {
        // CLINICAL_DETERIORATION is "handled in main stream" — no tag enrichment
        var fn = new Module4_PatternDetection.PatternClassificationFunction();
        PatternEvent input = Module4TestBuilder.deteriorationPattern("CLASS-003", "HIGH", 0.85);

        PatternEvent result = fn.map(input);

        assertFalse(result.getTags().contains("pathway_adherence"));
        assertFalse(result.getTags().contains("anomaly"));
        assertFalse(result.getTags().contains("trend"));
    }

    @Test
    void classification_preservesAllFields() throws Exception {
        var fn = new Module4_PatternDetection.PatternClassificationFunction();
        PatternEvent input = Module4TestBuilder.deteriorationPattern("CLASS-002", "CRITICAL", 0.95);
        input.addTag("CEP_DETECTED");

        PatternEvent result = fn.map(input);

        assertEquals("CRITICAL", result.getSeverity());
        assertEquals(0.95, result.getConfidence(), 0.01);
        assertTrue(result.getTags().contains("CEP_DETECTED"));
    }

    // ── PATHWAY_COMPLIANCE ────────────────────────────────────────────────────

    @Test
    void classification_pathwayCompliance_addsPathwayAdherenceTag() throws Exception {
        var fn = new Module4_PatternDetection.PatternClassificationFunction();
        PatternEvent input = Module4TestBuilder.deteriorationPattern("CLASS-010", "LOW", 0.70);
        input.setPatternType("PATHWAY_COMPLIANCE");

        PatternEvent result = fn.map(input);

        assertNotNull(result);
        assertEquals("PATHWAY_COMPLIANCE", result.getPatternType());
        assertTrue(result.getTags().contains("pathway_adherence"),
            "PATHWAY_COMPLIANCE events must carry 'pathway_adherence' tag");
    }

    // ── ANOMALY_DETECTION ─────────────────────────────────────────────────────

    @Test
    void classification_anomalyDetection_addsAnomalyTag() throws Exception {
        var fn = new Module4_PatternDetection.PatternClassificationFunction();
        PatternEvent input = Module4TestBuilder.deteriorationPattern("CLASS-020", "MEDIUM", 0.75);
        input.setPatternType("ANOMALY_DETECTION");

        PatternEvent result = fn.map(input);

        assertNotNull(result);
        assertEquals("ANOMALY_DETECTION", result.getPatternType());
        assertTrue(result.getTags().contains("anomaly"),
            "ANOMALY_DETECTION events must carry 'anomaly' tag");
    }

    // ── TREND_ANALYSIS ────────────────────────────────────────────────────────

    @Test
    void classification_trendAnalysis_addsTrendTag() throws Exception {
        var fn = new Module4_PatternDetection.PatternClassificationFunction();
        PatternEvent input = Module4TestBuilder.deteriorationPattern("CLASS-030", "LOW", 0.65);
        input.setPatternType("TREND_ANALYSIS");

        PatternEvent result = fn.map(input);

        assertNotNull(result);
        assertEquals("TREND_ANALYSIS", result.getPatternType());
        assertTrue(result.getTags().contains("trend"),
            "TREND_ANALYSIS events must carry 'trend' tag");
    }
}
