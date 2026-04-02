package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.GlucoseWindow;
import java.util.List;

/**
 * Glucose feature extraction for Module 10.
 *
 * Computes 8 features from a GlucoseWindow:
 * 1. baseline — first reading (or explicit baseline from state)
 * 2. peak — max glucose value in window
 * 3. excursion — peak - baseline
 * 4. timeToPeakMin — minutes from first reading to peak
 * 5. iAUC — incremental area under curve (trapezoidal, positive-only above baseline)
 * 6. recoveryTimeMin — minutes from peak to first reading within 10% of baseline
 * 7. readingCount — number of readings in window
 * 8. qualityScore — 0-1 based on reading density
 *
 * Stateless utility class — all computation from GlucoseWindow.
 */
public class Module10GlucoseAnalyzer {

    private Module10GlucoseAnalyzer() {}

    /**
     * Analyze a glucose window. Returns null if window is empty.
     */
    public static Result analyze(GlucoseWindow window) {
        if (window == null || window.isEmpty()) return null;

        window.sortByTime();
        List<GlucoseWindow.Reading> readings = window.getReadings();
        double baseline = window.getBaseline() != null
            ? window.getBaseline()
            : readings.get(0).value;

        if (readings.size() == 1) {
            Result r = new Result();
            r.baseline = baseline;
            r.peak = readings.get(0).value;
            r.excursion = r.peak - baseline;
            r.timeToPeakMin = 0.0;
            r.iAUC = 0.0;
            r.recoveryTimeMin = null;
            r.readingCount = 1;
            r.qualityScore = 0.1;
            return r;
        }

        // Find peak
        double peak = Double.NEGATIVE_INFINITY;
        int peakIndex = 0;
        for (int i = 0; i < readings.size(); i++) {
            if (readings.get(i).value > peak) {
                peak = readings.get(i).value;
                peakIndex = i;
            }
        }

        long firstTime = readings.get(0).timestamp;
        double timeToPeakMin = (readings.get(peakIndex).timestamp - firstTime) / 60_000.0;

        // Compute iAUC (trapezoidal rule, positive-only above baseline)
        double iAUC = 0.0;
        for (int i = 0; i < readings.size() - 1; i++) {
            double g1 = Math.max(0.0, readings.get(i).value - baseline);
            double g2 = Math.max(0.0, readings.get(i + 1).value - baseline);
            double dtSeconds = (readings.get(i + 1).timestamp - readings.get(i).timestamp) / 1000.0;
            iAUC += (g1 + g2) / 2.0 * dtSeconds;
        }

        // Recovery time: from peak, find first reading within 10% of baseline
        Double recoveryTimeMin = null;
        double recoveryThreshold = baseline * 1.10;
        for (int i = peakIndex + 1; i < readings.size(); i++) {
            if (readings.get(i).value <= recoveryThreshold) {
                recoveryTimeMin = (readings.get(i).timestamp - readings.get(peakIndex).timestamp) / 60_000.0;
                break;
            }
        }

        // Quality score: based on reading density (ideal: 1 reading per 5 min for 3h = 36 readings)
        long windowDurationMs = readings.get(readings.size() - 1).timestamp - firstTime;
        double expectedReadings = Math.max(1, windowDurationMs / 300_000.0); // 5-min intervals
        double qualityScore = Math.min(1.0, readings.size() / expectedReadings);

        Result r = new Result();
        r.baseline = baseline;
        r.peak = peak;
        r.excursion = peak - baseline;
        r.timeToPeakMin = timeToPeakMin;
        r.iAUC = iAUC;
        r.recoveryTimeMin = recoveryTimeMin;
        r.readingCount = readings.size();
        r.qualityScore = qualityScore;
        return r;
    }

    public static class Result {
        public double baseline;
        public double peak;
        public double excursion;
        public double timeToPeakMin;
        public double iAUC;
        public Double recoveryTimeMin;
        public int readingCount;
        public double qualityScore;
    }
}
