package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.DailyBPSummary;
import java.util.List;

public final class Module7ARVComputer {

    private Module7ARVComputer() {}

    public static Double computeARV(List<DailyBPSummary> summaries) {
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
