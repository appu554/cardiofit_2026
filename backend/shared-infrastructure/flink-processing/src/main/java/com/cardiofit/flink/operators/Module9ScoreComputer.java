package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.Map;

/**
 * Computes the weighted composite engagement score and phenotype
 * from the 14-day signal bitmaps. Pure function, no Flink dependencies.
 *
 * R2 Fix: Channel-aware level classification (CORPORATE/GOVERNMENT/ACCHS).
 * R7 Fix: Phenotype uses validHistoryDays counter, not > 0.0 proxy.
 */
public final class Module9ScoreComputer {

    private Module9ScoreComputer() {}

    public static Result compute(EngagementState state) {
        Map<SignalType, Double> densities = state.getAllDensities();

        double composite = 0.0;
        for (SignalType signal : SignalType.values()) {
            double density = densities.getOrDefault(signal, 0.0);
            composite += density * signal.getWeight();
        }
        composite = Math.min(1.0, composite);

        EngagementChannel channel = state.getEngagementChannel();
        EngagementLevel level = EngagementLevel.fromScore(composite, channel);

        String phenotype = classifyPhenotype(state);

        return new Result(composite, level, densities, phenotype, channel);
    }

    /**
     * Classify engagement phenotype based on temporal pattern.
     *
     * STEADY:    Recent 7d average >= 90% of older 7d average
     * DECLINING: Recent 7d average < 70% of older 7d average
     * SPORADIC:  Everything else (inconsistent engagement)
     *
     * R7 Fix: Uses validHistoryDays counter instead of checking history[i] > 0.
     */
    private static String classifyPhenotype(EngagementState state) {
        double[] history = state.getCompositeHistory14d();
        int validDays = state.getValidHistoryDays();

        if (validDays < 10) return "SPORADIC";

        int recentDays = Math.min(validDays, 7);
        int olderDays = Math.min(validDays - 7, 7);
        if (olderDays < 3) return "SPORADIC";

        double recentSum = 0.0, olderSum = 0.0;
        for (int i = 0; i < recentDays; i++) {
            recentSum += history[i];
        }
        for (int i = 7; i < 7 + olderDays; i++) {
            olderSum += history[i];
        }

        double recentAvg = recentSum / recentDays;
        double olderAvg = olderSum / olderDays;

        if (olderAvg < 1e-9) {
            return recentAvg > 1e-9 ? "STEADY" : "SPORADIC";
        }

        double ratio = recentAvg / olderAvg;

        if (ratio >= 0.90 - 1e-9) return "STEADY";
        if (ratio < 0.70 + 1e-9) return "DECLINING";
        return "SPORADIC";
    }

    public static class Result {
        public final double compositeScore;
        public final EngagementLevel level;
        public final Map<SignalType, Double> densities;
        public final String phenotype;
        public final EngagementChannel channel;

        public Result(double compositeScore, EngagementLevel level,
                     Map<SignalType, Double> densities, String phenotype,
                     EngagementChannel channel) {
            this.compositeScore = compositeScore;
            this.level = level;
            this.densities = densities;
            this.phenotype = phenotype;
            this.channel = channel;
        }
    }
}
