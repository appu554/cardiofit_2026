package com.cardiofit.flink.models;

/**
 * AlertType enum for clinical alert categorization
 * Used to classify different types of clinical alerts for routing and handling
 */
public enum AlertType {
    VITAL_THRESHOLD_BREACH,
    LAB_CRITICAL_VALUE,
    MEDICATION_MISSED,
    CLINICAL_SCORE_HIGH,
    SEPSIS,                // Sepsis detection (simplified alias for SEPSIS_PATTERN)
    SEPSIS_PATTERN,
    DETERIORATION_PATTERN,
    DRUG_INTERACTION,
    ALLERGY_ALERT,
    CARDIAC_EVENT,
    RESPIRATORY_DISTRESS,

    // Added for Unified Clinical Reasoning Pipeline (Phase 2-4)
    CLINICAL,              // General clinical pattern alerts (sepsis, MODS, ACS, etc.)
    MEDICATION,            // Medication-related alerts (interactions, nephrotoxic risk, etc.)
    LAB_ABNORMALITY        // Lab abnormality alerts (cardiac markers, electrolytes, etc.)
}
