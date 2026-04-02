package com.cardiofit.flink.models;

/**
 * Glucose response curve shape classification.
 * Determined by Module10CurveClassifier using first-derivative analysis
 * on 3-point moving-average smoothed CGM readings.
 */
public enum CurveShape {
    RAPID_SPIKE,    // Sharp rise, sharp fall — peak within first 30 min
    SLOW_RISE,      // Gradual ascent over 60-90 min
    PLATEAU,        // Rise then sustained elevation (fall rate < 0.5 mg/dL/min)
    DOUBLE_PEAK,    // Two distinct peaks separated by ≥20 min
    FLAT,           // Max excursion < 20 mg/dL above baseline
    UNKNOWN;        // Insufficient data points for classification

    /** Minimum CGM readings required for classification (Tier 1 only) */
    public static final int MIN_READINGS_FOR_CLASSIFICATION = 6;
}
