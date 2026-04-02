package com.cardiofit.flink.models;

/**
 * Salt sensitivity classification from Module 10b OLS regression.
 * β = slope of linear regression on (sodium_mg, SBP_excursion) pairs.
 */
public enum SaltSensitivityClass {
    SALT_RESISTANT,  // β < 0.001 mmHg/mg
    MODERATE,        // 0.001 ≤ β < 0.005
    HIGH,            // β ≥ 0.005
    UNDETERMINED;    // < 30 pairs OR R² < 0.1

    public static SaltSensitivityClass fromBetaAndR2(double beta, double rSquared, int pairCount) {
        if (pairCount < 30) return UNDETERMINED;
        // When the regression fit is poor (R² < 0.1), only classify as SALT_RESISTANT
        // if beta is essentially zero (< 1e-4), confirming total absence of sensitivity signal.
        // Otherwise return UNDETERMINED — we cannot distinguish true resistance from noise.
        if (rSquared < 0.1) {
            if (beta < 1e-4) return SALT_RESISTANT;
            return UNDETERMINED;
        }
        if (beta < 0.001) return SALT_RESISTANT;
        if (beta < 0.005) return MODERATE;
        return HIGH;
    }
}
