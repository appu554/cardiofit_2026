package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module10GlucoseAnalyzerTest {

    private static final long BASE = 1743552000000L; // 2025-04-02 00:00 UTC
    private static final long MIN_5 = 5 * 60_000L;

    @Test
    void iAUC_trapezoidalRule_positiveOnlyAboveBaseline() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(100.0);
        window.addReading(BASE, 100.0, "CGM");
        window.addReading(BASE + MIN_5, 140.0, "CGM");
        window.addReading(BASE + 2 * MIN_5, 100.0, "CGM");
        window.sortByTime();

        Module10GlucoseAnalyzer.Result result = Module10GlucoseAnalyzer.analyze(window);

        // iAUC: triangle above baseline
        // Segment 1: (0 + 40)/2 * 300s = 6000
        // Segment 2: (40 + 0)/2 * 300s = 6000
        // Total = 12000 mg·s/dL
        assertEquals(12000.0, result.iAUC, 1.0);
        assertEquals(140.0, result.peak);
        assertEquals(40.0, result.excursion);
        assertEquals(5.0, result.timeToPeakMin);
    }

    @Test
    void iAUC_ignoresNegativeExcursions() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(120.0);
        window.addReading(BASE, 120.0, "CGM");
        window.addReading(BASE + MIN_5, 110.0, "CGM");  // below baseline
        window.addReading(BASE + 2 * MIN_5, 130.0, "CGM");
        window.sortByTime();

        Module10GlucoseAnalyzer.Result result = Module10GlucoseAnalyzer.analyze(window);

        // Segment 1: max(0,0) + max(0,-10) / 2 * 300 = 0
        // Segment 2: max(0,-10) + max(0,10) / 2 * 300 = 1500
        assertEquals(1500.0, result.iAUC, 1.0);
    }

    @Test
    void peakAndTimeToPeak_correctForMultipleReadings() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(90.0);
        window.addReading(BASE, 90.0, "CGM");
        window.addReading(BASE + 10 * MIN_5, 95.0, "CGM");   // 50 min
        window.addReading(BASE + 12 * MIN_5, 180.0, "CGM");   // 60 min — peak
        window.addReading(BASE + 18 * MIN_5, 120.0, "CGM");   // 90 min
        window.sortByTime();

        Module10GlucoseAnalyzer.Result result = Module10GlucoseAnalyzer.analyze(window);

        assertEquals(180.0, result.peak);
        assertEquals(90.0, result.excursion);
        assertEquals(60.0, result.timeToPeakMin);
    }

    @Test
    void recoveryTime_timeToReturnWithin10pctOfBaseline() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(100.0);
        window.addReading(BASE, 100.0, "CGM");
        window.addReading(BASE + 6 * MIN_5, 160.0, "CGM");   // 30 min — peak
        window.addReading(BASE + 12 * MIN_5, 130.0, "CGM");  // 60 min — still high
        window.addReading(BASE + 18 * MIN_5, 108.0, "CGM");  // 90 min — within 10%
        window.sortByTime();

        Module10GlucoseAnalyzer.Result result = Module10GlucoseAnalyzer.analyze(window);

        // Recovery = time from peak to first reading within 10% of baseline
        // Peak at 30 min, recovery at 90 min → 60 min recovery
        assertEquals(60.0, result.recoveryTimeMin, 1.0);
    }

    @Test
    void emptyWindow_returnsNullResult() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(100.0);

        Module10GlucoseAnalyzer.Result result = Module10GlucoseAnalyzer.analyze(window);
        assertNull(result);
    }

    @Test
    void singleReading_returnsMinimalResult() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(100.0);
        window.addReading(BASE, 100.0, "SMBG");
        window.sortByTime();

        Module10GlucoseAnalyzer.Result result = Module10GlucoseAnalyzer.analyze(window);
        assertNotNull(result);
        assertEquals(0.0, result.iAUC);
        assertEquals(100.0, result.peak);
    }
}
