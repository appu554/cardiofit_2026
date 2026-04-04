package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module11GlucoseExerciseAnalyzerTest {

    private static final long BASE = 1743552000000L;
    private static final long MIN_5 = 5 * 60_000L;
    private static final long MIN_30 = 30 * 60_000L;

    @Test
    void aerobicExercise_glucoseDropsFromBaseline() {
        // Simulate 30 min moderate jog: glucose drops from 120 to 90
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(120.0);
        long actStart = BASE;
        long actEnd = BASE + MIN_30;
        window.addReading(BASE - MIN_5, 120.0, "CGM");   // pre-exercise
        window.addReading(BASE + 10 * 60_000L, 110.0, "CGM");
        window.addReading(BASE + 20 * 60_000L, 100.0, "CGM");
        window.addReading(BASE + 30 * 60_000L, 90.0, "CGM");  // end of exercise
        window.addReading(BASE + 45 * 60_000L, 95.0, "CGM");  // recovery
        window.addReading(BASE + 60 * 60_000L, 105.0, "CGM");
        window.sortByTime();

        Module11GlucoseExerciseAnalyzer.Result result =
                Module11GlucoseExerciseAnalyzer.analyze(window, actStart, actEnd, ExerciseType.AEROBIC);

        assertNotNull(result);
        assertEquals(120.0, result.preExerciseGlucose);
        assertTrue(result.exerciseGlucoseDelta < 0); // glucose dropped
        assertEquals(90.0, result.glucoseNadir);
        assertFalse(result.hypoglycemiaFlag); // 90 > 70
    }

    @Test
    void hypoglycemiaFlagged_below70() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(100.0);
        long actStart = BASE;
        long actEnd = BASE + MIN_30;
        window.addReading(BASE - MIN_5, 100.0, "CGM");
        window.addReading(BASE + 15 * 60_000L, 80.0, "CGM");
        window.addReading(BASE + 25 * 60_000L, 65.0, "CGM"); // hypo!
        window.addReading(BASE + 40 * 60_000L, 75.0, "CGM");
        window.sortByTime();

        Module11GlucoseExerciseAnalyzer.Result result =
                Module11GlucoseExerciseAnalyzer.analyze(window, actStart, actEnd, ExerciseType.AEROBIC);

        assertTrue(result.hypoglycemiaFlag);
        assertEquals(65.0, result.glucoseNadir);
    }

    @Test
    void hiitExercise_reboundHyperglycemiaDetected() {
        // HIIT: catecholamine spike, glucose rises above baseline + 40
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(110.0);
        long actStart = BASE;
        long actEnd = BASE + 20 * 60_000L;
        window.addReading(BASE - MIN_5, 110.0, "CGM");
        window.addReading(BASE + 10 * 60_000L, 130.0, "CGM");
        window.addReading(BASE + 20 * 60_000L, 145.0, "CGM");
        window.addReading(BASE + 35 * 60_000L, 160.0, "CGM"); // rebound: 160 > 110+40=150
        window.addReading(BASE + 50 * 60_000L, 125.0, "CGM");
        window.sortByTime();

        Module11GlucoseExerciseAnalyzer.Result result =
                Module11GlucoseExerciseAnalyzer.analyze(window, actStart, actEnd, ExerciseType.HIIT);

        assertTrue(result.reboundHyperglycemiaFlag);
        assertTrue(result.exerciseGlucoseDelta > 0); // glucose went up during HIIT
    }

    @Test
    void noGlucoseReadings_returnsNull() {
        GlucoseWindow window = new GlucoseWindow();
        Module11GlucoseExerciseAnalyzer.Result result =
                Module11GlucoseExerciseAnalyzer.analyze(window, BASE, BASE + MIN_30, ExerciseType.AEROBIC);
        assertNull(result);
    }

    @Test
    void preExerciseGlucose_fromReadingBeforeActivityStart() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(115.0);
        long actStart = BASE;
        long actEnd = BASE + MIN_30;
        // Two pre-exercise readings — should pick the one closest to start
        window.addReading(BASE - 20 * 60_000L, 108.0, "CGM");
        window.addReading(BASE - 5 * 60_000L, 115.0, "CGM"); // closest
        window.addReading(BASE + 15 * 60_000L, 100.0, "CGM");
        window.sortByTime();

        Module11GlucoseExerciseAnalyzer.Result result =
                Module11GlucoseExerciseAnalyzer.analyze(window, actStart, actEnd, ExerciseType.AEROBIC);

        assertEquals(115.0, result.preExerciseGlucose, 0.1);
    }
}
