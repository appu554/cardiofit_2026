package com.cardiofit.flink.analytics;

/** DD#1 Section 6 — SBP>180 or DBP>120 bypass */
public class HypertensiveCrisisDetector {
    public static boolean isCrisis(double sbp, double dbp) {
        return sbp > 180.0 || dbp > 120.0;
    }

    public static boolean requiresCuffConfirmation(boolean isCuffless) {
        return isCuffless; // cuffless critical readings → prompt cuff, not direct alert
    }
}
