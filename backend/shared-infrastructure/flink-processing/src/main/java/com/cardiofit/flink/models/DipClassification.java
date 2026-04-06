package com.cardiofit.flink.models;

/**
 * Nocturnal dipping pattern classification.
 *
 * Dip ratio = 1 - (nocturnal_mean_SBP / daytime_mean_SBP)
 *
 * DIPPER:           10-20% nocturnal reduction (normal, cardioprotective)
 * NON_DIPPER:       0-10% reduction (elevated CV risk)
 * EXTREME_DIPPER:   > 20% reduction (cerebral hypoperfusion risk)
 * REVERSE_DIPPER:   < 0% (nocturnal > daytime - highest CV risk)
 * INSUFFICIENT_DATA: fewer than 3 morning + 3 evening readings in 7 days
 *
 * Non-dippers have 2-3x higher cardiovascular event rates per MAPEC study.
 * Detection requires morning AND evening (or nocturnal) readings on the same days.
 */
public enum DipClassification {
    DIPPER,
    NON_DIPPER,
    EXTREME_DIPPER,
    REVERSE_DIPPER,
    INSUFFICIENT_DATA;

    /**
     * Classify from dip ratio.
     * @param dipRatio 1 - (nightMean / dayMean)
     */
    public static DipClassification fromDipRatio(double dipRatio) {
        if (dipRatio < 0.0) return REVERSE_DIPPER;
        if (dipRatio < 0.10 - 1e-9) return NON_DIPPER;
        if (dipRatio < 0.20 - 1e-9) return DIPPER;
        return EXTREME_DIPPER;
    }
}
