package com.cardiofit.flink.models;

/**
 * ARV-based variability classification.
 * Thresholds from J-HOP study data (upper quartile ~12-15 mmHg).
 *
 * LOW        (ARV < 8):     Normal - no action
 * MODERATE   (8 <= ARV < 12): Monitor - track trend over 30 days
 * ELEVATED   (12 <= ARV < 16): Flag in physician dashboard, consider medication timing
 * HIGH       (ARV >= 16):    Decision Card: medication review
 */
public enum VariabilityClassification {
    LOW,
    MODERATE,
    ELEVATED,
    HIGH,
    INSUFFICIENT_DATA;

    public static VariabilityClassification fromARV(Double arv) {
        if (arv == null) return INSUFFICIENT_DATA;
        if (arv < 8.0 - 1e-9) return LOW;
        if (arv < 12.0 - 1e-9) return MODERATE;
        if (arv < 16.0 - 1e-9) return ELEVATED;
        return HIGH;
    }
}
