package com.cardiofit.flink.models;

/**
 * Data tier classification for glucose processing depth.
 * Tier 1: CGM (288 readings/day, 8 features)
 * Tier 2: Hybrid CGM+SMBG (interpolated iAUC)
 * Tier 3: SMBG-only (excursion only, no curve shape)
 */
public enum DataTier {
    TIER_1_CGM,
    TIER_2_HYBRID,
    TIER_3_SMBG;

    public static DataTier fromString(String tier) {
        if (tier == null) return TIER_3_SMBG;
        switch (tier.toUpperCase().replace("-", "_")) {
            case "TIER_1_CGM":
            case "TIER_1":
            case "CGM":
                return TIER_1_CGM;
            case "TIER_2_HYBRID":
            case "TIER_2":
            case "HYBRID":
                return TIER_2_HYBRID;
            default:
                return TIER_3_SMBG;
        }
    }

    public boolean supportsCurveClassification() {
        return this == TIER_1_CGM;
    }

    public boolean supportsFullIAUC() {
        return this == TIER_1_CGM || this == TIER_2_HYBRID;
    }
}
