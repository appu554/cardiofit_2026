package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CGMAnalyticsEvent;
import com.cardiofit.flink.models.CGMReadingBuffer;
import com.cardiofit.flink.models.CGMReadingBuffer.ConsecutiveRun;

import java.util.List;

/**
 * Module 3 — CGM Analytics: International Consensus 2023 metrics.
 *
 * Computes TIR / TBR(L1+L2) / TAR(L1+L2), CV, GMI, GRI with zone
 * classification, and sustained hypo/hyper detection from a CGMReadingBuffer.
 *
 * All thresholds from shared/cgm_targets.yaml (International Consensus 2023).
 */
public class Module3_CGMAnalytics {

    // --- Range thresholds (mg/dL) ---
    public static final double VERY_LOW = 54.0;
    public static final double LOW = 70.0;
    public static final double HIGH = 180.0;
    public static final double VERY_HIGH = 250.0;

    // --- GMI formula constants ---
    public static final double GMI_INTERCEPT = 3.31;
    public static final double GMI_SLOPE = 0.02392;

    // --- GRI weights ---
    public static final double GRI_WEIGHT_VERY_LOW = 3.0;
    public static final double GRI_WEIGHT_LOW = 2.4;
    public static final double GRI_WEIGHT_VERY_HIGH = 1.6;
    public static final double GRI_WEIGHT_HIGH = 0.8;
    public static final double GRI_MAX = 100.0;

    // --- CV threshold ---
    public static final double CV_STABLE = 36.0;

    // --- Data sufficiency ---
    public static final int READINGS_PER_DAY = 96;
    public static final double MIN_COVERAGE = 70.0;

    private Module3_CGMAnalytics() {} // utility class

    /**
     * Compute all CGM metrics from a list of glucose readings.
     *
     * @param readings   list of glucose values in mg/dL
     * @param windowDays number of days the readings span
     * @return CGMAnalyticsEvent with all computed metrics
     */
    public static CGMAnalyticsEvent computeMetrics(List<Double> readings, int windowDays) {
        int n = readings.size();

        // --- Coverage ---
        double expected = (double) windowDays * READINGS_PER_DAY;
        double coveragePct = expected > 0 ? (n / expected) * 100.0 : 0.0;
        boolean sufficient = coveragePct >= MIN_COVERAGE;

        // --- Bucket readings into 5 ranges ---
        int countVeryLow = 0, countLow = 0, countInRange = 0;
        int countHigh = 0, countVeryHigh = 0;
        double sum = 0.0;

        for (double g : readings) {
            sum += g;
            if (g < VERY_LOW) {
                countVeryLow++;
            } else if (g < LOW) {
                countLow++;
            } else if (g <= HIGH) {
                countInRange++;
            } else if (g <= VERY_HIGH) {
                countHigh++;
            } else {
                countVeryHigh++;
            }
        }

        // --- Percentages ---
        double tirPct = n > 0 ? (countInRange / (double) n) * 100.0 : 0.0;
        double tbrL1Pct = n > 0 ? (countLow / (double) n) * 100.0 : 0.0;
        double tbrL2Pct = n > 0 ? (countVeryLow / (double) n) * 100.0 : 0.0;
        double tarL1Pct = n > 0 ? (countHigh / (double) n) * 100.0 : 0.0;
        double tarL2Pct = n > 0 ? (countVeryHigh / (double) n) * 100.0 : 0.0;

        // --- Mean, SD, CV ---
        double mean = n > 0 ? sum / n : 0.0;
        double varianceSum = 0.0;
        for (double g : readings) {
            varianceSum += (g - mean) * (g - mean);
        }
        double sd = n > 1 ? Math.sqrt(varianceSum / (n - 1)) : 0.0;
        double cv = mean > 0 ? (sd / mean) * 100.0 : 0.0;

        // --- GMI ---
        double gmi = GMI_INTERCEPT + GMI_SLOPE * mean;

        // --- GRI ---
        double gri = GRI_WEIGHT_VERY_LOW * tbrL2Pct
                + GRI_WEIGHT_LOW * tbrL1Pct
                + GRI_WEIGHT_VERY_HIGH * tarL2Pct
                + GRI_WEIGHT_HIGH * tarL1Pct;
        gri = Math.min(gri, GRI_MAX);

        // --- GRI Zone ---
        String griZone;
        if (gri <= 20.0) griZone = "A";
        else if (gri <= 40.0) griZone = "B";
        else if (gri <= 60.0) griZone = "C";
        else if (gri <= 80.0) griZone = "D";
        else griZone = "E";

        // --- Confidence ---
        String confidence;
        if (coveragePct >= 70.0) confidence = "HIGH";
        else if (coveragePct >= 50.0) confidence = "MODERATE";
        else confidence = "LOW";

        return CGMAnalyticsEvent.builder()
                .coveragePct(coveragePct)
                .totalReadings(n)
                .windowDays(windowDays)
                .sufficientData(sufficient)
                .confidenceLevel(confidence)
                .meanGlucose(mean)
                .sdGlucose(sd)
                .cvPct(cv)
                .glucoseStable(cv <= CV_STABLE)
                .tirPct(tirPct)
                .tbrL1Pct(tbrL1Pct)
                .tbrL2Pct(tbrL2Pct)
                .tarL1Pct(tarL1Pct)
                .tarL2Pct(tarL2Pct)
                .gmi(gmi)
                .gri(gri)
                .griZone(griZone)
                .computedAt(System.currentTimeMillis())
                .build();
    }

    /**
     * Detect sustained hypoglycaemia: any consecutive run below threshold
     * lasting at least minMinutes.
     */
    public static boolean detectSustainedHypo(
            CGMReadingBuffer buffer, long windowStart, long windowEnd,
            double threshold, double minMinutes) {
        List<ConsecutiveRun> runs =
                buffer.findConsecutiveRunsBelowThreshold(threshold, windowStart, windowEnd);
        for (ConsecutiveRun run : runs) {
            if (run.durationMinutes() >= minMinutes) {
                return true;
            }
        }
        return false;
    }

    /**
     * Detect nocturnal hypoglycaemia within a specific overnight window.
     * Delegates to sustained hypo detection within the nocturnal time range.
     */
    public static boolean detectNocturnalHypo(
            CGMReadingBuffer buffer, long nocturnalStart, long nocturnalEnd,
            double threshold, double minMinutes) {
        return detectSustainedHypo(buffer, nocturnalStart, nocturnalEnd, threshold, minMinutes);
    }
}
