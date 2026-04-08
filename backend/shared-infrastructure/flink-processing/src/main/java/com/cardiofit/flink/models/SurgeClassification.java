package com.cardiofit.flink.models;

/**
 * Morning BP surge classification.
 * Surge = morning_SBP_avg - evening_SBP_avg (sleep-trough method).
 * Per Kario et al., morning surge > 25 mmHg is an independent stroke predictor.
 *
 * NORMAL:    surge < 20 mmHg
 * ELEVATED:  20 <= surge < 35 mmHg (monitor, consider medication timing)
 * HIGH:      surge >= 35 mmHg (Decision Card: assess medication timing, sleep apnea)
 */
public enum SurgeClassification {
    NORMAL,
    ELEVATED,
    HIGH,
    INSUFFICIENT_DATA;

    public static SurgeClassification fromSurge(Double surgeMmHg) {
        if (surgeMmHg == null) return INSUFFICIENT_DATA;
        if (surgeMmHg < 20.0 - 1e-9) return NORMAL;
        if (surgeMmHg < 35.0 - 1e-9) return ELEVATED;
        return HIGH;
    }
}
