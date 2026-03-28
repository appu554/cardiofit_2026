package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module4TestBuilder;
import com.cardiofit.flink.models.PatternEvent;
import com.cardiofit.flink.models.SemanticEvent;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 4 window functions.
 * Uses direct invocation with mock collectors — no Flink runtime needed.
 *
 * Key implementation details verified before writing:
 *
 * TrendAnalysisWindowFunction — implements WindowFunction → uses apply()
 *   - Needs >= 3 events; emits only when direction != "STABLE"
 *   - slope > 0.1 → INCREASING; slope < -0.1 → DECREASING; otherwise STABLE (no emit)
 *
 * AnomalyDetectionWindowFunction — extends ProcessWindowFunction → uses process()
 *   - Needs >= 5 events; 2-stddev threshold; context parameter is never accessed (null is safe)
 *
 * ProtocolMonitoringWindowFunction — implements WindowFunction → uses apply()
 *   - Emits when sum(guidelineRecommendations.size()) > 0 across all events
 *   - Events with NO guideline recommendations → no emit
 */
public class Module4WindowFunctionTest {

    // ── TrendAnalysisWindowFunction ───────────────────────────

    @Test
    void trendAnalysis_deteriorating_emitsPattern() throws Exception {
        var fn = new Module4_PatternDetection.TrendAnalysisWindowFunction();
        String patientId = "TREND-001";

        // 5 events with strictly increasing clinical significance
        // significances: 0.20, 0.35, 0.50, 0.65, 0.80 → slope = 0.15 (> 0.1 threshold → INCREASING)
        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 3600_000;
        for (int i = 0; i < 5; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent(patientId);
            se.setEventTime(baseTime + (i * 600_000L)); // 10 min apart
            se.getSemanticAnnotations().put("clinical_significance", 0.2 + (i * 0.15));
            events.add(se);
        }

        TimeWindow window = new TimeWindow(baseTime, baseTime + 3600_000);
        List<PatternEvent> collected = new ArrayList<>();
        fn.apply(patientId, window, events, new ListCollector<>(collected));

        assertFalse(collected.isEmpty(), "Deteriorating trend should emit a pattern event");
        PatternEvent result = collected.get(0);
        assertEquals("TREND_ANALYSIS", result.getPatternType());
        assertEquals(patientId, result.getPatientId());

        double slope = (double) result.getPatternDetails().get("trend_slope");
        assertTrue(slope > 0.1, "Slope should exceed 0.1 (INCREASING threshold), got: " + slope);
    }

    @Test
    void trendAnalysis_stable_emitsNothing() throws Exception {
        var fn = new Module4_PatternDetection.TrendAnalysisWindowFunction();
        String patientId = "TREND-002";

        // 4 events with nearly flat significance (slope ≈ 0.01, well within STABLE range)
        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 3600_000;
        for (int i = 0; i < 4; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent(patientId);
            se.setEventTime(baseTime + (i * 600_000L));
            se.getSemanticAnnotations().put("clinical_significance", 0.35 + (i * 0.01));
            events.add(se);
        }

        TimeWindow window = new TimeWindow(baseTime, baseTime + 3600_000);
        List<PatternEvent> collected = new ArrayList<>();
        fn.apply(patientId, window, events, new ListCollector<>(collected));

        assertTrue(collected.isEmpty(), "Stable trend (slope ~0.01) should NOT emit a pattern event");
    }

    @Test
    void trendAnalysis_tooFewEvents_emitsNothing() throws Exception {
        var fn = new Module4_PatternDetection.TrendAnalysisWindowFunction();
        String patientId = "TREND-003";

        // Only 2 events — below the minimum of 3
        List<SemanticEvent> events = new ArrayList<>();
        events.add(Module4TestBuilder.baselineVitalEvent(patientId));
        events.add(Module4TestBuilder.warningVitalEvent(patientId));

        TimeWindow window = new TimeWindow(0, 3600_000);
        List<PatternEvent> collected = new ArrayList<>();
        fn.apply(patientId, window, events, new ListCollector<>(collected));

        assertTrue(collected.isEmpty(), "< 3 events should not produce trend analysis");
    }

    // ── AnomalyDetectionWindowFunction (ProcessWindowFunction — uses process(), not apply()) ──

    @Test
    void anomalyDetection_highStdDev_emitsAnomaly() throws Exception {
        var fn = new Module4_PatternDetection.AnomalyDetectionWindowFunction();
        String patientId = "ANOM-001";

        // 8 events: 7 tightly clustered near 0.20-0.22 + 1 spike at 0.90
        // With 7 low values anchoring mean/stddev, spike of 0.9 clearly exceeds mean + 2*stddev
        // Verified: mean≈0.295, stddev≈0.229, threshold≈0.753 → 0.9 > 0.753
        double[] significances = {0.20, 0.21, 0.20, 0.22, 0.21, 0.20, 0.22, 0.90};
        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 1800_000;
        for (int i = 0; i < significances.length; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent(patientId);
            se.setEventTime(baseTime + (i * 300_000L));
            se.getSemanticAnnotations().put("clinical_significance", significances[i]);
            events.add(se);
        }

        // AnomalyDetectionWindowFunction extends ProcessWindowFunction.
        // process() never accesses the Context parameter — null is safe to pass.
        List<PatternEvent> collected = new ArrayList<>();
        fn.process(patientId, null, events, new ListCollector<>(collected));

        assertFalse(collected.isEmpty(), "Anomalous spike should emit an anomaly pattern");
        PatternEvent result = collected.get(0);
        assertEquals("ANOMALY_DETECTION", result.getPatternType());
        assertEquals(patientId, result.getPatientId());
    }

    @Test
    void anomalyDetection_uniform_emitsNothing() throws Exception {
        var fn = new Module4_PatternDetection.AnomalyDetectionWindowFunction();
        String patientId = "ANOM-002";

        // 5 events all at exactly the same significance → stddev=0, threshold=mean, nothing exceeds it
        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 1800_000;
        for (int i = 0; i < 5; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent(patientId);
            se.setEventTime(baseTime + (i * 300_000L));
            se.getSemanticAnnotations().put("clinical_significance", 0.3);
            events.add(se);
        }

        List<PatternEvent> collected = new ArrayList<>();
        fn.process(patientId, null, events, new ListCollector<>(collected));

        assertTrue(collected.isEmpty(), "Uniform readings should NOT trigger anomaly detection");
    }

    // ── ProtocolMonitoringWindowFunction ─────────────────────

    /**
     * ProtocolMonitoringWindowFunction emits when events carry guidelineRecommendations.
     * This test verifies that an admission event WITH a guideline recommendation triggers emission.
     *
     * Note: The plan described this as "noAssessment_emitsComplianceAlert" but the actual
     * implementation does not check for missing assessments — it counts guideline recommendations
     * across the window and emits if recommendations_count > 0.
     */
    @Test
    void protocolMonitoring_withRecommendations_emitsProtocolPattern() throws Exception {
        var fn = new Module4_PatternDetection.ProtocolMonitoringWindowFunction();
        String patientId = "PROTO-001";

        SemanticEvent admission = Module4TestBuilder.admissionEvent(patientId);

        // Attach a guideline recommendation so the window function has something to emit
        SemanticEvent.GuidelineRecommendation rec = new SemanticEvent.GuidelineRecommendation();
        rec.setRecommendationId("REC-001");
        rec.setGuidelineSource("SEPSIS-3");
        rec.setRecommendation("Obtain blood cultures before antibiotics");
        rec.setEvidenceLevel("A");
        rec.setConfidence(0.95);
        admission.setGuidelineRecommendations(List.of(rec));

        List<SemanticEvent> events = new ArrayList<>();
        events.add(admission);

        long baseTime = System.currentTimeMillis() - 7200_000;
        TimeWindow window = new TimeWindow(baseTime, baseTime + 7200_000);
        List<PatternEvent> collected = new ArrayList<>();
        fn.apply(patientId, window, events, new ListCollector<>(collected));

        assertFalse(collected.isEmpty(), "Event with guideline recommendation should emit a PROTOCOL_MONITORING pattern");
        PatternEvent result = collected.get(0);
        assertEquals("PROTOCOL_MONITORING", result.getPatternType());
        assertEquals(patientId, result.getPatientId());
        long recCount = (long) result.getPatternDetails().get("recommendations_count");
        assertTrue(recCount > 0, "recommendations_count should be > 0, got: " + recCount);
    }

    @Test
    void protocolMonitoring_noRecommendations_emitsNothing() throws Exception {
        var fn = new Module4_PatternDetection.ProtocolMonitoringWindowFunction();
        String patientId = "PROTO-002";

        // Admission event with NO guideline recommendations → recommendationsCount = 0 → no emit
        List<SemanticEvent> events = new ArrayList<>();
        events.add(Module4TestBuilder.admissionEvent(patientId));

        long baseTime = System.currentTimeMillis() - 7200_000;
        TimeWindow window = new TimeWindow(baseTime, baseTime + 7200_000);
        List<PatternEvent> collected = new ArrayList<>();
        fn.apply(patientId, window, events, new ListCollector<>(collected));

        assertTrue(collected.isEmpty(), "Events without guideline recommendations should NOT emit a protocol pattern");
    }

    // ── Helper: ListCollector ─────────────────────────────────

    /**
     * Simple Collector implementation that stores results in a list.
     * No Flink runtime required.
     */
    private static class ListCollector<T> implements Collector<T> {
        private final List<T> list;

        ListCollector(List<T> list) {
            this.list = list;
        }

        @Override
        public void collect(T record) {
            list.add(record);
        }

        @Override
        public void close() {}
    }
}
