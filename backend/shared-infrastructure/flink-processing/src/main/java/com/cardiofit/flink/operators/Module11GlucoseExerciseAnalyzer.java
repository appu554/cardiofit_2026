package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.ExerciseType;
import com.cardiofit.flink.models.GlucoseWindow;
import java.util.List;

/**
 * Exercise-specific glucose analysis for Module 11.
 *
 * Unlike Module 10's meal glucose analysis (which looks for iAUC above baseline),
 * exercise glucose analysis tracks:
 * 1. Pre-exercise glucose (baseline from last reading before activity start)
 * 2. Exercise glucose delta (mean during exercise minus pre-exercise)
 * 3. Glucose nadir (minimum during activity + 1h post-exercise recovery)
 * 4. Hypoglycemia flag (any reading < 70 mg/dL during exercise or recovery)
 * 5. Rebound hyperglycemia flag (any reading > baseline + 40 mg/dL post-exercise)
 *
 * Expected patterns:
 * - Aerobic: glucose drops 20–60 mg/dL (GLUT4 translocation)
 * - HIIT/Resistance: glucose spikes 20–40 mg/dL (catecholamine surge)
 * - Post-exercise: continued drop for 2h (glycogen resynthesis)
 *
 * Stateless utility class.
 */
public class Module11GlucoseExerciseAnalyzer {

    private static final double HYPOGLYCEMIA_THRESHOLD = 70.0;       // mg/dL
    private static final double REBOUND_HYPERGLYCEMIA_MARGIN = 40.0; // mg/dL above baseline
    private static final long ONE_HOUR_MS = 3_600_000L;

    private Module11GlucoseExerciseAnalyzer() {}

    /**
     * Analyze glucose during an exercise session.
     *
     * @param window         glucose readings for the session
     * @param activityStart  activity start timestamp
     * @param activityEnd    activity end timestamp
     * @param exerciseType   type of exercise (affects expected pattern)
     * @return analysis result, or null if no readings
     */
    public static Result analyze(GlucoseWindow window, long activityStart, long activityEnd,
                                 ExerciseType exerciseType) {
        if (window == null || window.isEmpty()) return null;

        window.sortByTime();
        List<GlucoseWindow.Reading> readings = window.getReadings();

        // 1. Pre-exercise glucose: last reading before activity start
        Double preExerciseGlucose = null;
        for (int i = readings.size() - 1; i >= 0; i--) {
            if (readings.get(i).timestamp < activityStart) {
                preExerciseGlucose = readings.get(i).value;
                break;
            }
        }
        // Fallback to explicit baseline if no pre-exercise reading
        if (preExerciseGlucose == null) {
            preExerciseGlucose = window.getBaseline();
        }
        // Final fallback to first reading
        if (preExerciseGlucose == null) {
            preExerciseGlucose = readings.get(0).value;
        }

        // 2. Exercise glucose delta: mean during exercise minus pre-exercise
        double sumDuring = 0;
        int countDuring = 0;
        for (GlucoseWindow.Reading r : readings) {
            if (r.timestamp >= activityStart && r.timestamp <= activityEnd) {
                sumDuring += r.value;
                countDuring++;
            }
        }
        Double exerciseGlucoseDelta = null;
        if (countDuring > 0) {
            double meanDuring = sumDuring / countDuring;
            exerciseGlucoseDelta = meanDuring - preExerciseGlucose;
        }

        // 3. Glucose nadir: minimum during activity + 1h post
        double glucoseNadir = Double.MAX_VALUE;
        long nadirWindow = activityEnd + ONE_HOUR_MS;
        for (GlucoseWindow.Reading r : readings) {
            if (r.timestamp >= activityStart && r.timestamp <= nadirWindow) {
                if (r.value < glucoseNadir) {
                    glucoseNadir = r.value;
                }
            }
        }
        if (glucoseNadir == Double.MAX_VALUE) glucoseNadir = preExerciseGlucose;

        // 4. Hypoglycemia flag: any reading < 70 during exercise or recovery
        boolean hypoglycemiaFlag = false;
        for (GlucoseWindow.Reading r : readings) {
            if (r.timestamp >= activityStart && r.value < HYPOGLYCEMIA_THRESHOLD) {
                hypoglycemiaFlag = true;
                break;
            }
        }

        // 5. Rebound hyperglycemia: any reading > baseline + 40 post-exercise
        boolean reboundFlag = false;
        double reboundThreshold = preExerciseGlucose + REBOUND_HYPERGLYCEMIA_MARGIN;
        for (GlucoseWindow.Reading r : readings) {
            if (r.timestamp > activityEnd && r.value > reboundThreshold) {
                reboundFlag = true;
                break;
            }
        }

        Result result = new Result();
        result.preExerciseGlucose = preExerciseGlucose;
        result.exerciseGlucoseDelta = exerciseGlucoseDelta;
        result.glucoseNadir = glucoseNadir;
        result.hypoglycemiaFlag = hypoglycemiaFlag;
        result.reboundHyperglycemiaFlag = reboundFlag;
        result.readingCount = readings.size();
        return result;
    }

    public static class Result {
        public Double preExerciseGlucose;
        public Double exerciseGlucoseDelta;
        public double glucoseNadir;
        public boolean hypoglycemiaFlag;
        public boolean reboundHyperglycemiaFlag;
        public int readingCount;
    }
}
