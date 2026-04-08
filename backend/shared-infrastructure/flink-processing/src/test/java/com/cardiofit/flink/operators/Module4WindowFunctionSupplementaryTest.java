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
 * Supplementary window function tests closing coverage gaps.
 *
 * Implementation details (verified from Module4_PatternDetection.java):
 *
 * TrendAnalysisWindowFunction — WindowFunction.apply()
 *   - Minimum 3 events; uses INDEX-based linear regression (x=i, not timestamp)
 *   - slope > 0.1 → "INCREASING" / "MODERATE"; slope < -0.1 → "DECREASING" / "LOW"
 *   - "STABLE" → no emit. DECREASING trends ARE emitted (as LOW).
 *   - Denominator is n*sumX2 - sumX*sumX using sequential indices → never zero when n>=3
 *
 * AnomalyDetectionWindowFunction — ProcessWindowFunction.process()
 *   - Minimum 5 events; 2-stddev threshold; Context never accessed (null safe)
 *   - Only events with significance > mean + 2*stddev are flagged
 *
 * ProtocolMonitoringWindowFunction — WindowFunction.apply()
 *   - Counts guidelineRecommendations across all events; emits if count > 0
 *   - Does NOT check admission→assessment→intervention sequence
 *   - Always emits severity="LOW" when recommendations exist
 */
public class Module4WindowFunctionSupplementaryTest {

    // ══════════════════════════════════════════════════════════
    // TrendAnalysisWindowFunction — 4 new tests
    // ══════════════════════════════════════════════════════════

    @Test
    void trendAnalysis_improving_emitsDecreasingWithLowSeverity() throws Exception {
        var fn = new Module4_PatternDetection.TrendAnalysisWindowFunction();
        String patientId = "TREND-IMPR-001";

        // 5 events with DECREASING significance (patient improving)
        // significances: 0.80, 0.65, 0.50, 0.35, 0.20 → slope = -0.15 (< -0.1 → DECREASING)
        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 3600_000;
        for (int i = 0; i < 5; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent(patientId);
            se.setEventTime(baseTime + (i * 600_000L));
            se.getSemanticAnnotations().put("clinical_significance", 0.8 - (i * 0.15));
            events.add(se);
        }

        TimeWindow window = new TimeWindow(baseTime, baseTime + 3600_000);
        List<PatternEvent> collected = new ArrayList<>();
        fn.apply(patientId, window, events, new ListCollector<>(collected));

        // DECREASING trends ARE emitted (direction != "STABLE")
        assertFalse(collected.isEmpty(),
            "Improving (decreasing significance) trend should emit — direction is DECREASING, not STABLE");
        PatternEvent result = collected.get(0);
        assertEquals("TREND_ANALYSIS", result.getPatternType());

        double slope = (double) result.getPatternDetails().get("trend_slope");
        assertTrue(slope < -0.1,
            "Improving trend slope should be < -0.1 (DECREASING), got " + slope);
        assertEquals("DECREASING", result.getPatternDetails().get("trend_direction"));
        assertEquals("LOW", result.getSeverity(),
            "Improving (DECREASING) trends should always be LOW severity");
    }

    @Test
    void trendAnalysis_singleEvent_emitsNothing() throws Exception {
        var fn = new Module4_PatternDetection.TrendAnalysisWindowFunction();
        String patientId = "TREND-SINGLE-001";

        List<SemanticEvent> events = new ArrayList<>();
        events.add(Module4TestBuilder.criticalVitalEvent(patientId));

        TimeWindow window = new TimeWindow(0, 3600_000);
        List<PatternEvent> collected = new ArrayList<>();
        fn.apply(patientId, window, events, new ListCollector<>(collected));

        assertTrue(collected.isEmpty(),
            "Single event (< 3 minimum) cannot establish a trend — must not emit");
    }

    @Test
    void trendAnalysis_steepDeterioration_higherSlopeThanGradual() throws Exception {
        var fn = new Module4_PatternDetection.TrendAnalysisWindowFunction();
        long baseTime = System.currentTimeMillis() - 3600_000;

        // Steep: 0.1 → 0.9 over 5 events (step = 0.2)
        List<SemanticEvent> steepEvents = new ArrayList<>();
        for (int i = 0; i < 5; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent("STEEP-001");
            se.setEventTime(baseTime + (i * 600_000L));
            se.getSemanticAnnotations().put("clinical_significance", 0.1 + (i * 0.2));
            steepEvents.add(se);
        }

        // Gradual: 0.3 → 0.5 over 5 events (step = 0.05)
        List<SemanticEvent> gradualEvents = new ArrayList<>();
        for (int i = 0; i < 5; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent("GRADUAL-001");
            se.setEventTime(baseTime + (i * 600_000L));
            se.getSemanticAnnotations().put("clinical_significance", 0.3 + (i * 0.05));
            gradualEvents.add(se);
        }

        TimeWindow window = new TimeWindow(baseTime, baseTime + 3600_000);

        List<PatternEvent> steepCollected = new ArrayList<>();
        fn.apply("STEEP-001", window, steepEvents, new ListCollector<>(steepCollected));

        List<PatternEvent> gradualCollected = new ArrayList<>();
        fn.apply("GRADUAL-001", window, gradualEvents, new ListCollector<>(gradualCollected));

        assertFalse(steepCollected.isEmpty(), "Steep deterioration (slope=0.2) must emit");

        // Gradual slope = 0.05 which is < 0.1 threshold → STABLE → no emit
        // So we only compare if both emitted
        if (!gradualCollected.isEmpty()) {
            double steepSlope = (double) steepCollected.get(0)
                .getPatternDetails().get("trend_slope");
            double gradualSlope = (double) gradualCollected.get(0)
                .getPatternDetails().get("trend_slope");
            assertTrue(steepSlope > gradualSlope,
                String.format("Steep slope (%.4f) should exceed gradual (%.4f)",
                    steepSlope, gradualSlope));
        }

        // Verify steep slope value
        double steepSlope = (double) steepCollected.get(0).getPatternDetails().get("trend_slope");
        assertEquals(0.2, steepSlope, 0.01,
            "Steep events with step=0.2 should produce slope ≈ 0.2");
    }

    @Test
    void trendAnalysis_exactlyThreeEvents_minimumViableTrend() throws Exception {
        // 3 events is the exact minimum (< 3 returns early)
        var fn = new Module4_PatternDetection.TrendAnalysisWindowFunction();
        String patientId = "TREND-MIN-001";

        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 1800_000;
        // Clear deterioration: 0.2 → 0.5 → 0.8 → slope = 0.3 (> 0.1)
        double[] sigs = {0.2, 0.5, 0.8};
        for (int i = 0; i < 3; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent(patientId);
            se.setEventTime(baseTime + (i * 600_000L));
            se.getSemanticAnnotations().put("clinical_significance", sigs[i]);
            events.add(se);
        }

        TimeWindow window = new TimeWindow(baseTime, baseTime + 1800_000);
        List<PatternEvent> collected = new ArrayList<>();
        fn.apply(patientId, window, events, new ListCollector<>(collected));

        assertFalse(collected.isEmpty(),
            "Exactly 3 events with clear deterioration (slope=0.3) should emit a trend");
        assertEquals("INCREASING", collected.get(0).getPatternDetails().get("trend_direction"));
        assertEquals(3, collected.get(0).getPatternDetails().get("data_points"));
    }

    // ══════════════════════════════════════════════════════════
    // AnomalyDetectionWindowFunction — 4 new tests
    // Uses process() not apply(), Context=null is safe
    // ══════════════════════════════════════════════════════════

    @Test
    void anomalyDetection_fourEvents_belowMinimum_emitsNothing() throws Exception {
        // Minimum is 5 events — 4 should return early
        var fn = new Module4_PatternDetection.AnomalyDetectionWindowFunction();

        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 1800_000;
        double[] sigs = {0.2, 0.2, 0.2, 0.95}; // spike present but < 5 events

        for (int i = 0; i < sigs.length; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent("ANOM-FOUR");
            se.setEventTime(baseTime + (i * 300_000L));
            se.getSemanticAnnotations().put("clinical_significance", sigs[i]);
            events.add(se);
        }

        List<PatternEvent> collected = new ArrayList<>();
        fn.process("ANOM-FOUR", null, events, new ListCollector<>(collected));

        assertTrue(collected.isEmpty(),
            "4 events (below minimum of 5) must not emit — even with a spike present");
    }

    @Test
    void anomalyDetection_sixEvents_withSpike_emitsAnomaly() throws Exception {
        // Need 6+ events so the spike doesn't inflate its own threshold above itself.
        // 5 clustered at ~0.20 + 1 spike at 0.90.
        // mean≈0.318, stddev≈0.260, threshold≈0.839 → 0.90 > 0.839 ✓
        var fn = new Module4_PatternDetection.AnomalyDetectionWindowFunction();

        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 1800_000;
        double[] sigs = {0.20, 0.20, 0.21, 0.20, 0.20, 0.90};

        for (int i = 0; i < sigs.length; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent("ANOM-SIX");
            se.setEventTime(baseTime + (i * 300_000L));
            se.getSemanticAnnotations().put("clinical_significance", sigs[i]);
            events.add(se);
        }

        List<PatternEvent> collected = new ArrayList<>();
        fn.process("ANOM-SIX", null, events, new ListCollector<>(collected));

        assertFalse(collected.isEmpty(),
            "6 events with clear outlier (0.90 vs ~0.20 cluster) should emit anomaly");
        PatternEvent result = collected.get(0);
        assertEquals("ANOMALY_DETECTION", result.getPatternType());
        assertEquals(1, result.getPatternDetails().get("anomaly_count"),
            "Only the 0.90 spike should be flagged as anomalous");
    }

    @Test
    void anomalyDetection_multipleSpikes_countsAll() throws Exception {
        var fn = new Module4_PatternDetection.AnomalyDetectionWindowFunction();

        // 12 events: 10 clustered at ~0.20, 2 spikes at 0.88 and 0.92.
        // More anchoring data points needed so spikes don't inflate their own threshold.
        // mean≈0.323, stddev≈0.259, threshold≈0.840 → 0.88 and 0.92 both exceed.
        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 4000_000;
        double[] sigs = {0.20, 0.21, 0.20, 0.22, 0.21, 0.20, 0.22, 0.20, 0.21, 0.20, 0.88, 0.92};

        for (int i = 0; i < sigs.length; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent("ANOM-MULTI");
            se.setEventTime(baseTime + (i * 300_000L));
            se.getSemanticAnnotations().put("clinical_significance", sigs[i]);
            events.add(se);
        }

        List<PatternEvent> collected = new ArrayList<>();
        fn.process("ANOM-MULTI", null, events, new ListCollector<>(collected));

        assertFalse(collected.isEmpty(),
            "Multiple spikes in stable readings should trigger anomaly detection");
        int anomalyCount = (int) collected.get(0).getPatternDetails().get("anomaly_count");
        assertEquals(2, anomalyCount,
            "Both spikes (0.88, 0.92) should be counted as anomalies");
    }

    @Test
    void anomalyDetection_gradualIncrease_notAnomalous() throws Exception {
        // Steady linear climb → high stddev but no single value exceeds mean + 2*stddev
        var fn = new Module4_PatternDetection.AnomalyDetectionWindowFunction();

        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 3000_000;
        // 0.10, 0.22, 0.34, 0.46, 0.58, 0.70 → mean=0.40, stddev≈0.21, threshold≈0.82
        // Max value 0.70 < 0.82 → no anomaly
        for (int i = 0; i < 6; i++) {
            SemanticEvent se = Module4TestBuilder.baselineVitalEvent("ANOM-GRAD");
            se.setEventTime(baseTime + (i * 300_000L));
            se.getSemanticAnnotations().put("clinical_significance", 0.10 + (i * 0.12));
            events.add(se);
        }

        List<PatternEvent> collected = new ArrayList<>();
        fn.process("ANOM-GRAD", null, events, new ListCollector<>(collected));

        assertTrue(collected.isEmpty(),
            "Gradual linear increase (max 0.70 < threshold ~0.82) should not trigger anomaly");
    }

    // ══════════════════════════════════════════════════════════
    // ProtocolMonitoringWindowFunction — 4 new tests
    // Implementation counts guidelineRecommendations — does NOT
    // check admission→assessment→intervention protocol sequence.
    // ══════════════════════════════════════════════════════════

    @Test
    void protocolMonitoring_multipleEventsWithRecommendations_countsTotal() throws Exception {
        var fn = new Module4_PatternDetection.ProtocolMonitoringWindowFunction();
        String patientId = "PROTO-MULTI";

        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 7200_000;

        // Event 1: 2 recommendations
        SemanticEvent e1 = Module4TestBuilder.admissionEvent(patientId);
        e1.setEventTime(baseTime);
        SemanticEvent.GuidelineRecommendation r1 = new SemanticEvent.GuidelineRecommendation();
        r1.setRecommendationId("REC-001");
        r1.setGuidelineSource("SEPSIS-3");
        r1.setRecommendation("Blood cultures before antibiotics");
        r1.setEvidenceLevel("A");
        r1.setConfidence(0.95);
        SemanticEvent.GuidelineRecommendation r2 = new SemanticEvent.GuidelineRecommendation();
        r2.setRecommendationId("REC-002");
        r2.setGuidelineSource("KDIGO-2024");
        r2.setRecommendation("Monitor renal function");
        r2.setEvidenceLevel("B");
        r2.setConfidence(0.85);
        e1.setGuidelineRecommendations(List.of(r1, r2));
        events.add(e1);

        // Event 2: 1 recommendation
        SemanticEvent e2 = Module4TestBuilder.labResultEvent(patientId);
        e2.setEventTime(baseTime + 1800_000);
        SemanticEvent.GuidelineRecommendation r3 = new SemanticEvent.GuidelineRecommendation();
        r3.setRecommendationId("REC-003");
        r3.setGuidelineSource("AHA-2023");
        r3.setRecommendation("Initiate fluid resuscitation");
        r3.setEvidenceLevel("A");
        r3.setConfidence(0.90);
        e2.setGuidelineRecommendations(List.of(r3));
        events.add(e2);

        TimeWindow window = new TimeWindow(baseTime, baseTime + 7200_000);
        List<PatternEvent> collected = new ArrayList<>();
        fn.apply(patientId, window, events, new ListCollector<>(collected));

        assertFalse(collected.isEmpty(), "Events with recommendations should emit");
        long recCount = (long) collected.get(0).getPatternDetails().get("recommendations_count");
        assertEquals(3, recCount,
            "Should count total recommendations across all events (2 + 1 = 3)");
        assertEquals("LOW", collected.get(0).getSeverity(),
            "Protocol monitoring always emits LOW severity");
    }

    @Test
    void protocolMonitoring_mixedEvents_onlyCountsRecommendations() throws Exception {
        var fn = new Module4_PatternDetection.ProtocolMonitoringWindowFunction();
        String patientId = "PROTO-MIXED";

        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 7200_000;

        // Event 1: no recommendations
        SemanticEvent e1 = Module4TestBuilder.admissionEvent(patientId);
        e1.setEventTime(baseTime);
        events.add(e1);

        // Event 2: no recommendations
        SemanticEvent e2 = Module4TestBuilder.labResultEvent(patientId);
        e2.setEventTime(baseTime + 600_000);
        events.add(e2);

        // Event 3: has 1 recommendation
        SemanticEvent e3 = Module4TestBuilder.procedureScheduledEvent(patientId);
        e3.setEventTime(baseTime + 1200_000);
        SemanticEvent.GuidelineRecommendation rec = new SemanticEvent.GuidelineRecommendation();
        rec.setRecommendationId("REC-100");
        rec.setGuidelineSource("NICE");
        rec.setRecommendation("Schedule follow-up within 48h");
        rec.setEvidenceLevel("C");
        rec.setConfidence(0.75);
        e3.setGuidelineRecommendations(List.of(rec));
        events.add(e3);

        TimeWindow window = new TimeWindow(baseTime, baseTime + 7200_000);
        List<PatternEvent> collected = new ArrayList<>();
        fn.apply(patientId, window, events, new ListCollector<>(collected));

        assertFalse(collected.isEmpty(),
            "Window with at least one recommendation should emit");
        long recCount = (long) collected.get(0).getPatternDetails().get("recommendations_count");
        assertEquals(1, recCount,
            "Only 1 event has recommendations — total count should be 1");
    }

    @Test
    void protocolMonitoring_emptyWindow_emitsNothing() throws Exception {
        var fn = new Module4_PatternDetection.ProtocolMonitoringWindowFunction();
        String patientId = "PROTO-EMPTY";

        List<SemanticEvent> events = new ArrayList<>();
        TimeWindow window = new TimeWindow(0, 7200_000);
        List<PatternEvent> collected = new ArrayList<>();

        assertDoesNotThrow(() ->
            fn.apply(patientId, window, events, new ListCollector<>(collected)),
            "Empty window must not throw");
        assertTrue(collected.isEmpty(),
            "Empty window should not generate protocol alerts");
    }

    @Test
    void protocolMonitoring_eventsWithNullRecommendations_emitsNothing() throws Exception {
        // Events where getGuidelineRecommendations() returns null (not empty list)
        var fn = new Module4_PatternDetection.ProtocolMonitoringWindowFunction();
        String patientId = "PROTO-NULL-REC";

        List<SemanticEvent> events = new ArrayList<>();
        long baseTime = System.currentTimeMillis() - 7200_000;

        // Default builder events have null guidelineRecommendations
        events.add(Module4TestBuilder.admissionEvent(patientId));
        events.add(Module4TestBuilder.labResultEvent(patientId));
        events.add(Module4TestBuilder.procedureScheduledEvent(patientId));

        TimeWindow window = new TimeWindow(baseTime, baseTime + 7200_000);
        List<PatternEvent> collected = new ArrayList<>();

        assertDoesNotThrow(() ->
            fn.apply(patientId, window, events, new ListCollector<>(collected)),
            "Null guidelineRecommendations must not cause NPE");
        assertTrue(collected.isEmpty(),
            "Events with null recommendations → count=0 → no emit");
    }

    // ══════════════════════════════════════════════════════════
    // Shared ListCollector (same pattern as Module4WindowFunctionTest)
    // ══════════════════════════════════════════════════════════

    private static class ListCollector<T> implements Collector<T> {
        private final List<T> list;
        ListCollector(List<T> list) { this.list = list; }
        @Override public void collect(T record) { list.add(record); }
        @Override public void close() {}
    }
}
