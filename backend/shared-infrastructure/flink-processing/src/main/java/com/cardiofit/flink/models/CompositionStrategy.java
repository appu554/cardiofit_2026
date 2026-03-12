package com.cardiofit.flink.models;

/**
 * CompositionStrategy enum for composed alert generation strategy
 * Indicates how the alert was composed from different detection modules
 */
public enum CompositionStrategy {
    THRESHOLD_ONLY,      // Alert from threshold-based detection only (Module 2)
    CEP_ONLY,            // Alert from CEP pattern matching only (Module 4)
    COMBINED,            // Alert from both threshold and CEP analysis
    ML_ENRICHED          // Alert enhanced with ML predictions (Module 5)
}
