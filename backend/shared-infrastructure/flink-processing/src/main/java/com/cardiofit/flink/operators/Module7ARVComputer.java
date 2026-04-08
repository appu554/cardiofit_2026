package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.DailyBPSummary;
import java.util.List;

/**
 * Average Real Variability (ARV) and related BP statistics.
 *
 * <p><b>Primary ARV (Mena et al. 2005, ESH 2023)</b>:
 * Computed across ALL sequential readings sorted by timestamp, regardless of
 * time of day. The morning-evening oscillation IS the variability signal —
 * averaging readings into daily values before computing ARV destroys this
 * critical clinical information.</p>
 *
 * <p>Formula: ARV = (1/(N-1)) × Σ|SBP[i+1] - SBP[i]|</p>
 *
 * <p><b>Secondary (day-to-day ARV)</b>:
 * Uses daily-averaged SBP for visit-to-visit variability tracking.
 * Useful for long-term trend but NOT the primary clinical metric.</p>
 */
public final class Module7ARVComputer {

    private Module7ARVComputer() {}

    /**
     * PRIMARY: Compute reading-level ARV per Mena et al. (2005) / ESH 2023.
     * ARV = mean of |SBP[i+1] - SBP[i]| across ALL sequential readings.
     * Requires at least 3 readings.
     *
     * @param readingSBPs chronologically ordered SBP values (all time contexts mixed)
     */
    public static Double computeReadingLevelARV(List<Double> readingSBPs) {
        if (readingSBPs == null || readingSBPs.size() < 3) return null;
        double sumAbsDiff = 0;
        for (int i = 1; i < readingSBPs.size(); i++) {
            sumAbsDiff += Math.abs(readingSBPs.get(i) - readingSBPs.get(i - 1));
        }
        return sumAbsDiff / (readingSBPs.size() - 1);
    }

    /**
     * SECONDARY: Compute day-to-day ARV using daily-averaged SBP.
     * mean of |dayAvg[i] - dayAvg[i-1]| — smoothed metric for visit-to-visit variability.
     * Requires at least 3 days.
     */
    public static Double computeDayToDayARV(List<DailyBPSummary> summaries) {
        if (summaries == null || summaries.size() < 3) return null;
        double sumAbsDiff = 0;
        int pairs = 0;
        double prevAvg = summaries.get(0).getAvgSBP();
        for (int i = 1; i < summaries.size(); i++) {
            double currentAvg = summaries.get(i).getAvgSBP();
            sumAbsDiff += Math.abs(currentAvg - prevAvg);
            pairs++;
            prevAvg = currentAvg;
        }
        return pairs > 0 ? sumAbsDiff / pairs : null;
    }

    /**
     * @deprecated Use {@link #computeReadingLevelARV} for primary ARV
     *             or {@link #computeDayToDayARV} for secondary day-to-day metric.
     */
    @Deprecated
    public static Double computeARV(List<DailyBPSummary> summaries) {
        return computeDayToDayARV(summaries);
    }

    public static double computeMeanSBP(List<DailyBPSummary> summaries) {
        return summaries.stream().mapToDouble(DailyBPSummary::getAvgSBP).average().orElse(0.0);
    }

    public static double computeMeanDBP(List<DailyBPSummary> summaries) {
        return summaries.stream().mapToDouble(DailyBPSummary::getAvgDBP).average().orElse(0.0);
    }

    public static double computeSD(List<DailyBPSummary> summaries) {
        if (summaries.size() < 2) return 0.0;
        double mean = computeMeanSBP(summaries);
        double sumSqDiff = summaries.stream()
            .mapToDouble(s -> { double diff = s.getAvgSBP() - mean; return diff * diff; })
            .sum();
        return Math.sqrt(sumSqDiff / (summaries.size() - 1));
    }

    public static double computeCV(List<DailyBPSummary> summaries) {
        double mean = computeMeanSBP(summaries);
        if (mean < 1e-9) return 0.0;
        return computeSD(summaries) / mean;
    }
}
