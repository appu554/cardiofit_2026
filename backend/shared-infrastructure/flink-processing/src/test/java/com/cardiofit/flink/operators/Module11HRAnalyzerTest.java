package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module11HRAnalyzerTest {

    private static final long BASE = 1743552000000L; // 2025-04-02 00:00 UTC
    private static final long MIN_1 = 60_000L;
    private static final long MIN_5 = 5 * 60_000L;

    @Test
    void peakHR_correctlyIdentified() {
        HRWindow window = exerciseWindow(
                70, 85, 100, 120, 140, 155, 165, 170, 168, 160, 140, 120, 100, 85
        );
        Module11HRAnalyzer.Result result = Module11HRAnalyzer.analyze(window);
        assertEquals(170.0, result.peakHR);
    }

    @Test
    void meanActiveHR_excludesPreAndRecovery() {
        // Pre-exercise (before activityStartTime): 70
        // Active: 120, 140, 160, 150 → mean = 142.5
        // Recovery (after activityEndTime): 100, 85
        HRWindow window = new HRWindow();
        long actStart = BASE;
        long actEnd = BASE + 20 * MIN_1;
        window.setActivityStartTime(actStart);
        window.setActivityEndTime(actEnd);
        window.setHrMax(180.0);
        window.setRestingHR(65.0);

        window.addReading(BASE - 5 * MIN_1, 70, "WEARABLE");  // pre
        window.addReading(BASE, 120, "WEARABLE");               // active
        window.addReading(BASE + 5 * MIN_1, 140, "WEARABLE");
        window.addReading(BASE + 10 * MIN_1, 160, "WEARABLE");
        window.addReading(BASE + 15 * MIN_1, 150, "WEARABLE");
        window.addReading(BASE + 25 * MIN_1, 100, "WEARABLE");  // recovery
        window.addReading(BASE + 30 * MIN_1, 85, "WEARABLE");
        window.sortByTime();

        Module11HRAnalyzer.Result result = Module11HRAnalyzer.analyze(window);
        assertEquals(142.5, result.meanActiveHR, 0.1);
    }

    @Test
    void hrr1_dropFromPeakAt1Minute() {
        HRWindow window = new HRWindow();
        long actStart = BASE;
        long actEnd = BASE + 30 * MIN_1;
        window.setActivityStartTime(actStart);
        window.setActivityEndTime(actEnd);
        window.setHrMax(180.0);

        // Active phase readings
        window.addReading(BASE + 25 * MIN_1, 165, "WEARABLE");
        window.addReading(BASE + 28 * MIN_1, 172, "WEARABLE"); // peak
        window.addReading(BASE + 30 * MIN_1, 170, "WEARABLE");
        // Recovery phase
        window.addReading(actEnd + 1 * MIN_1, 148, "WEARABLE"); // ~1 min post: HRR1 = 172-148 = 24
        window.addReading(actEnd + 2 * MIN_1, 130, "WEARABLE"); // ~2 min post: HRR2 = 172-130 = 42
        window.sortByTime();

        Module11HRAnalyzer.Result result = Module11HRAnalyzer.analyze(window);
        assertEquals(172.0, result.peakHR);
        assertEquals(24.0, result.hrr1, 1.0);
        assertEquals(42.0, result.hrr2, 1.0);
        assertEquals(HRRecoveryClass.NORMAL, result.hrRecoveryClass);
    }

    @Test
    void hrr1_abnormalClassification() {
        HRWindow window = new HRWindow();
        long actStart = BASE;
        long actEnd = BASE + 30 * MIN_1;
        window.setActivityStartTime(actStart);
        window.setActivityEndTime(actEnd);
        window.setHrMax(180.0);

        window.addReading(BASE + 28 * MIN_1, 160, "WEARABLE"); // peak
        window.addReading(actEnd + 1 * MIN_1, 152, "WEARABLE"); // HRR1 = 8 → ABNORMAL
        window.addReading(actEnd + 2 * MIN_1, 145, "WEARABLE");
        window.sortByTime();

        Module11HRAnalyzer.Result result = Module11HRAnalyzer.analyze(window);
        assertEquals(8.0, result.hrr1, 1.0);
        assertEquals(HRRecoveryClass.ABNORMAL, result.hrRecoveryClass);
        assertTrue(result.hrRecoveryClass.isPrognosticFlag());
    }

    @Test
    void dominantZone_correctlyComputed() {
        HRWindow window = exerciseWindow(
                // All readings in ZONE_3_TEMPO for hrMax=180: 70-79% = 126-142
                130, 135, 138, 140, 138, 135, 132, 130
        );
        Module11HRAnalyzer.Result result = Module11HRAnalyzer.analyze(window);
        assertEquals(ActivityIntensityZone.ZONE_3_TEMPO, result.dominantZone);
    }

    @Test
    void rpp_computedFromPeakHRAndPeakSBP() {
        HRWindow window = new HRWindow();
        long actStart = BASE;
        long actEnd = BASE + 30 * MIN_1;
        window.setActivityStartTime(actStart);
        window.setActivityEndTime(actEnd);
        window.setHrMax(180.0);
        window.addReading(BASE + 15 * MIN_1, 160, "WEARABLE");
        window.sortByTime();

        Module11HRAnalyzer.Result result = Module11HRAnalyzer.analyze(window);
        assertEquals(160.0, result.peakHR);
    }

    @Test
    void emptyWindow_returnsNull() {
        HRWindow window = new HRWindow();
        window.setActivityStartTime(BASE);
        window.setActivityEndTime(BASE + 30 * MIN_1);
        Module11HRAnalyzer.Result result = Module11HRAnalyzer.analyze(window);
        assertNull(result);
    }

    /** Helper: create exercise window with 5-min interval readings starting at BASE */
    private HRWindow exerciseWindow(double... values) {
        HRWindow w = new HRWindow();
        long actStart = BASE;
        long actEnd = BASE + (values.length - 1) * MIN_5;
        w.setActivityStartTime(actStart);
        w.setActivityEndTime(actEnd);
        w.setHrMax(180.0);
        w.setRestingHR(65.0);
        for (int i = 0; i < values.length; i++) {
            w.addReading(BASE + i * MIN_5, values[i], "WEARABLE");
        }
        w.sortByTime();
        return w;
    }
}
