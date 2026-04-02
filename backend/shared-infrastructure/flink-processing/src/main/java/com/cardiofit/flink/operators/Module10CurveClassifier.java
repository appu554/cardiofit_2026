package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CurveShape;
import com.cardiofit.flink.models.GlucoseWindow;
import java.util.List;

/**
 * Curve shape classification for Module 10.
 *
 * Algorithm:
 * 1. Apply 3-point moving average smoothing
 * 2. Compute first derivative (rate of change per 5 min)
 * 3. Classify based on:
 *    - FLAT: max excursion < 20 mg/dL
 *    - RAPID_SPIKE: peak within 30 min of first reading
 *    - DOUBLE_PEAK: two distinct peaks separated by ≥20 min
 *    - PLATEAU: post-peak fall rate < 0.5 mg/dL/min for ≥15 min
 *    - SLOW_RISE: default if peak after 30 min
 *
 * Requires Tier 1 (CGM) with ≥6 readings.
 */
public class Module10CurveClassifier {

    private static final double FLAT_THRESHOLD = 20.0;       // mg/dL
    private static final double RAPID_SPIKE_PEAK_MIN = 30.0; // minutes
    private static final double PLATEAU_FALL_RATE = 0.5;     // mg/dL per minute
    private static final double PLATEAU_DURATION_MIN = 15.0;
    private static final double DOUBLE_PEAK_GAP_MIN = 20.0;  // minutes between peaks
    private static final double DOUBLE_PEAK_DIP_FRACTION = 0.3; // dip must be ≥30% of excursion

    private Module10CurveClassifier() {}

    public static CurveShape classify(GlucoseWindow window) {
        if (window == null || window.size() < CurveShape.MIN_READINGS_FOR_CLASSIFICATION) {
            return CurveShape.UNKNOWN;
        }

        window.sortByTime();
        List<GlucoseWindow.Reading> readings = window.getReadings();
        double baseline = window.getBaseline() != null ? window.getBaseline() : readings.get(0).value;

        // Step 1: 3-point moving average smoothing
        double[] smoothed = smooth(readings);

        // Step 2: Find peak
        int peakIdx = 0;
        double peakVal = smoothed[0];
        for (int i = 1; i < smoothed.length; i++) {
            if (smoothed[i] > peakVal) {
                peakVal = smoothed[i];
                peakIdx = i;
            }
        }

        double excursion = peakVal - baseline;

        // Rule 1: FLAT — excursion below threshold
        if (excursion < FLAT_THRESHOLD) {
            return CurveShape.FLAT;
        }

        // Rule 2: DOUBLE_PEAK — check for two distinct peaks
        if (hasDoublePeak(smoothed, baseline, excursion, readings)) {
            return CurveShape.DOUBLE_PEAK;
        }

        // Rule 3: PLATEAU — sustained elevation after peak
        if (isPlateau(smoothed, peakIdx, readings)) {
            return CurveShape.PLATEAU;
        }

        // Rule 4: RAPID_SPIKE vs SLOW_RISE — based on time to peak
        double timeToPeakMin = (readings.get(peakIdx).timestamp - readings.get(0).timestamp) / 60_000.0;
        if (timeToPeakMin <= RAPID_SPIKE_PEAK_MIN) {
            return CurveShape.RAPID_SPIKE;
        }

        return CurveShape.SLOW_RISE;
    }

    private static double[] smooth(List<GlucoseWindow.Reading> readings) {
        double[] smoothed = new double[readings.size()];
        smoothed[0] = readings.get(0).value;
        smoothed[readings.size() - 1] = readings.get(readings.size() - 1).value;
        for (int i = 1; i < readings.size() - 1; i++) {
            smoothed[i] = (readings.get(i - 1).value + readings.get(i).value + readings.get(i + 1).value) / 3.0;
        }
        return smoothed;
    }

    private static boolean hasDoublePeak(double[] smoothed, double baseline, double excursion,
                                          List<GlucoseWindow.Reading> readings) {
        // Find first peak
        int peak1 = -1;
        for (int i = 1; i < smoothed.length - 1; i++) {
            if (smoothed[i] > smoothed[i - 1] && smoothed[i] >= smoothed[i + 1]
                    && (smoothed[i] - baseline) > excursion * 0.5) {
                peak1 = i;
                break;
            }
        }
        if (peak1 < 0) return false;

        // Find valley after first peak
        int valley = -1;
        double valleyVal = smoothed[peak1];
        for (int i = peak1 + 1; i < smoothed.length; i++) {
            if (smoothed[i] < valleyVal) {
                valleyVal = smoothed[i];
                valley = i;
            }
            if (smoothed[i] > valleyVal + excursion * DOUBLE_PEAK_DIP_FRACTION) break;
        }
        if (valley < 0) return false;

        double dipDepth = smoothed[peak1] - valleyVal;
        if (dipDepth < excursion * DOUBLE_PEAK_DIP_FRACTION) return false;

        // Find second peak after valley
        for (int i = valley + 1; i < smoothed.length - 1; i++) {
            if (smoothed[i] > smoothed[i - 1] && smoothed[i] >= smoothed[i + 1]
                    && (smoothed[i] - baseline) > excursion * 0.4) {
                double gapMin = (readings.get(i).timestamp - readings.get(peak1).timestamp) / 60_000.0;
                if (gapMin >= DOUBLE_PEAK_GAP_MIN) {
                    return true;
                }
            }
        }
        return false;
    }

    private static boolean isPlateau(double[] smoothed, int peakIdx,
                                      List<GlucoseWindow.Reading> readings) {
        if (peakIdx >= smoothed.length - 2) return false;

        double sustainedStart = readings.get(peakIdx).timestamp;
        for (int i = peakIdx + 1; i < smoothed.length; i++) {
            double dtMin = (readings.get(i).timestamp - readings.get(i - 1).timestamp) / 60_000.0;
            if (dtMin <= 0) continue;
            double fallRate = (smoothed[i - 1] - smoothed[i]) / dtMin;
            if (fallRate > PLATEAU_FALL_RATE) {
                // Check if we sustained long enough before the drop
                double sustainedMin = (readings.get(i - 1).timestamp - sustainedStart) / 60_000.0;
                return sustainedMin >= PLATEAU_DURATION_MIN;
            }
        }

        // If we never dropped fast, entire post-peak is a plateau
        double totalSustainedMin = (readings.get(smoothed.length - 1).timestamp - sustainedStart) / 60_000.0;
        return totalSustainedMin >= PLATEAU_DURATION_MIN;
    }
}
