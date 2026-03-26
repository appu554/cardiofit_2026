package com.cardiofit.flink.analytics;

/** DD#1 Section 4.2 — Sleep-trough surge method */
public class MorningSurgeDetector {
    public static double computeSurge(double morningSBP, double eveningSBP) {
        return morningSBP - eveningSBP;
    }

    public static String classify(double surge) {
        if (surge < 20.0)  return "NORMAL";
        if (surge <= 35.0) return "ELEVATED";
        return "EXAGGERATED";
    }
}
