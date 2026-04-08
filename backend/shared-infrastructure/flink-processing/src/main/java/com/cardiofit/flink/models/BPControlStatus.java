package com.cardiofit.flink.models;

/**
 * BP control status based on 7-day average SBP and DBP.
 * AHA/ACC 2017 guidelines + JSH 2025 home BP targets.
 *
 * Home BP targets (lower than clinic thresholds):
 *   CONTROLLED:         SBP < 130 AND DBP < 80 (per JSH 2025 Asakatsu BP)
 *   ELEVATED:           130 <= SBP < 135 OR 80 <= DBP < 85
 *   STAGE_1:            135 <= SBP < 145 OR 85 <= DBP < 90
 *   STAGE_2:            SBP >= 145 OR DBP >= 90
 *   CRISIS:             SBP > 180 OR DBP > 120 (any single reading)
 *
 * Note: These are HOME BP thresholds, which are 5 mmHg lower than clinic thresholds.
 * The V-MCU correction loop uses bp_control_status to trigger Channel C
 * HTN-1 protocol for titration when STAGE_1 or worse persists >= 14 days.
 */
public enum BPControlStatus {
    CONTROLLED,
    ELEVATED,
    STAGE_1_UNCONTROLLED,
    STAGE_2_UNCONTROLLED,
    CRISIS;

    public static BPControlStatus fromAverages(double avgSBP, double avgDBP) {
        if (avgSBP >= 145.0 - 1e-9 || avgDBP >= 90.0 - 1e-9) return STAGE_2_UNCONTROLLED;
        if (avgSBP >= 135.0 - 1e-9 || avgDBP >= 85.0 - 1e-9) return STAGE_1_UNCONTROLLED;
        if (avgSBP >= 130.0 - 1e-9 || avgDBP >= 80.0 - 1e-9) return ELEVATED;
        return CONTROLLED;
    }
}
