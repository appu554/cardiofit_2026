package com.cardiofit.flink.analytics;

/**
 * Average Real Variability — DD#1 Section 4.1
 * ARV = (1/(N-1)) × Σ|SBP_{i+1} - SBP_i| for consecutive daily averages
 */
public class ARVCalculator {

    public static Double compute(double[] dailySBPAverages) {
        if (dailySBPAverages == null || dailySBPAverages.length < 3) return null;
        double sum = 0.0;
        for (int i = 1; i < dailySBPAverages.length; i++) {
            sum += Math.abs(dailySBPAverages[i] - dailySBPAverages[i - 1]);
        }
        return sum / (dailySBPAverages.length - 1);
    }

    public static String classify(double arv) {
        if (arv < 8.0)  return "LOW";
        if (arv < 12.0) return "MODERATE";
        if (arv < 15.0) return "HIGH";
        return "VERY_HIGH";
    }
}
