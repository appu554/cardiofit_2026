package com.cardiofit.flink.analytics;

/** DD#1 Section 4.3 — Nocturnal dip ratio */
public class DippingPatternClassifier {
    public static String classify(double daytimeSBPAvg, double nighttimeSBPAvg) {
        if (daytimeSBPAvg <= 0) return "NON_DIPPER"; // guard: invalid reading
        double dipRatio = 1.0 - (nighttimeSBPAvg / daytimeSBPAvg);
        double dipPercent = dipRatio * 100.0;
        if (dipPercent < 0)    return "REVERSE";
        if (dipPercent < 10.0) return "NON_DIPPER";
        if (dipPercent <= 20.0) return "DIPPER";
        return "EXTREME";
    }

    public static String confidence(boolean hasCufflessData) {
        return hasCufflessData ? "HIGH" : "LOW";
    }
}
