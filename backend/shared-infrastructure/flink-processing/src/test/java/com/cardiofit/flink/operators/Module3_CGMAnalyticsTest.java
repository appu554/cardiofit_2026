package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CGMAnalyticsEvent;
import com.cardiofit.flink.models.CGMReadingBuffer;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class Module3_CGMAnalyticsTest {

    // ---- 1. TIR — all in range ----
    @Test
    void testTIR_AllInRange() {
        List<Double> readings = Collections.nCopies(100, 120.0);
        CGMAnalyticsEvent event = Module3_CGMAnalytics.computeMetrics(readings, 1);

        assertEquals(100.0, event.getTirPct(), 0.01);
        assertEquals(0.0, event.getTbrL1Pct(), 0.01);
        assertEquals(0.0, event.getTbrL2Pct(), 0.01);
        assertEquals(0.0, event.getTarL1Pct(), 0.01);
        assertEquals(0.0, event.getTarL2Pct(), 0.01);
    }

    // ---- 2. TIR — mixed ranges ----
    @Test
    void testTIR_MixedRanges() {
        // 100 readings: 70 in-range, 10 L1-hypo, 5 L2-hypo, 10 L1-hyper, 5 L2-hyper
        List<Double> readings = new ArrayList<>();
        for (int i = 0; i < 70; i++) readings.add(120.0);   // in range (70-180)
        for (int i = 0; i < 10; i++) readings.add(60.0);    // L1 hypo (54-70)
        for (int i = 0; i < 5; i++)  readings.add(40.0);    // L2 hypo (<54)
        for (int i = 0; i < 10; i++) readings.add(200.0);   // L1 hyper (180-250)
        for (int i = 0; i < 5; i++)  readings.add(300.0);   // L2 hyper (>250)

        CGMAnalyticsEvent event = Module3_CGMAnalytics.computeMetrics(readings, 1);

        assertEquals(70.0, event.getTirPct(), 0.01);
        assertEquals(10.0, event.getTbrL1Pct(), 0.01);
        assertEquals(5.0, event.getTbrL2Pct(), 0.01);
        assertEquals(10.0, event.getTarL1Pct(), 0.01);
        assertEquals(5.0, event.getTarL2Pct(), 0.01);
    }

    // ---- 3. CV — stable glucose ----
    @Test
    void testCV_StableGlucose() {
        // Readings clustered tightly around 120 → CV should be < 36%
        List<Double> readings = new ArrayList<>();
        for (int i = 0; i < 50; i++) readings.add(118.0);
        for (int i = 0; i < 50; i++) readings.add(122.0);

        CGMAnalyticsEvent event = Module3_CGMAnalytics.computeMetrics(readings, 1);

        assertTrue(event.getCvPct() < 36.0, "CV should be < 36% for stable glucose");
        assertTrue(event.isGlucoseStable(), "glucoseStable should be true");
    }

    // ---- 4. GMI — standard formula ----
    @Test
    void testGMI_StandardFormula() {
        // Mean = 154 → GMI = 3.31 + 0.02392 * 154 = 6.99368
        List<Double> readings = Collections.nCopies(100, 154.0);
        CGMAnalyticsEvent event = Module3_CGMAnalytics.computeMetrics(readings, 1);

        double expectedGmi = 3.31 + 0.02392 * 154.0;
        assertEquals(expectedGmi, event.getGmi(), 0.01);
    }

    // ---- 5. GRI — zone A (all in range) ----
    @Test
    void testGRI_ZoneA() {
        List<Double> readings = Collections.nCopies(100, 120.0);
        CGMAnalyticsEvent event = Module3_CGMAnalytics.computeMetrics(readings, 1);

        assertEquals(0.0, event.getGri(), 0.01);
        assertEquals("A", event.getGriZone());
    }

    // ---- 6. Coverage — sufficient data ----
    @Test
    void testCoverage_SufficientData() {
        // 1000 readings in 14 days → coverage = 1000 / (14*96) * 100 ≈ 74.4%
        List<Double> readings = Collections.nCopies(1000, 120.0);
        CGMAnalyticsEvent event = Module3_CGMAnalytics.computeMetrics(readings, 14);

        assertTrue(event.isSufficientData(), "1000 readings in 14d should be sufficient (>70%)");
        assertEquals("HIGH", event.getConfidenceLevel());
        double expectedCoverage = (1000.0 / (14.0 * 96.0)) * 100.0;
        assertEquals(expectedCoverage, event.getCoveragePct(), 0.1);
    }

    // ---- 7. Coverage — insufficient data ----
    @Test
    void testCoverage_InsufficientData() {
        // 50 readings in 14 days → coverage = 50 / (14*96) * 100 ≈ 3.72%
        List<Double> readings = Collections.nCopies(50, 120.0);
        CGMAnalyticsEvent event = Module3_CGMAnalytics.computeMetrics(readings, 14);

        assertFalse(event.isSufficientData(), "50 readings in 14d should be insufficient");
        assertEquals("LOW", event.getConfidenceLevel());
    }

    // ---- 8. Buffer — sensor warmup excluded ----
    @Test
    void testBuffer_SensorWarmupExcluded() {
        CGMReadingBuffer buffer = new CGMReadingBuffer();
        long sensorStart = 1_000_000L;
        buffer.setSensorStartTime(sensorStart);

        // 30 min after sensor start — within warmup → rejected
        boolean added30 = buffer.addReading(sensorStart + 30 * 60_000L, 120.0);
        assertFalse(added30, "Reading at 30min should be rejected (within warmup)");

        // 90 min after sensor start — past warmup → accepted
        boolean added90 = buffer.addReading(sensorStart + 90 * 60_000L, 120.0);
        assertTrue(added90, "Reading at 90min should be accepted (past warmup)");

        assertEquals(1, buffer.size());
    }

    // ---- 9. Buffer — physiologically impossible excluded ----
    @Test
    void testBuffer_PhysiologicallyImpossibleExcluded() {
        CGMReadingBuffer buffer = new CGMReadingBuffer();

        boolean addedLow = buffer.addReading(1_000_000L, 10.0);   // < 20 mg/dL
        boolean addedHigh = buffer.addReading(2_000_000L, 550.0);  // > 500 mg/dL
        boolean addedValid = buffer.addReading(3_000_000L, 120.0); // valid

        assertFalse(addedLow, "10 mg/dL should be rejected as physiologically impossible");
        assertFalse(addedHigh, "550 mg/dL should be rejected as physiologically impossible");
        assertTrue(addedValid, "120 mg/dL should be accepted");

        assertEquals(1, buffer.size());
    }
}
