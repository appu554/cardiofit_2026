package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;

/**
 * Alert suppression and deduplication manager.
 *
 * Suppression rules:
 * - HALT alerts: NEVER suppressed (patient safety overrides fatigue prevention)
 * - PAUSE/SOFT_FLAG: Suppressed for 72 hours per suppression key
 * - Suppression key = ruleId + patientId + hash(sorted medications)
 * - Different medication combinations = different suppression keys
 * - Severity escalation (e.g., SOFT_FLAG → PAUSE for same combination)
 *   bypasses suppression
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module8SuppressionManager {
    private Module8SuppressionManager() {}

    /**
     * Determine if an alert should be suppressed.
     * @param alert the candidate alert
     * @param state current patient state (contains suppression history)
     * @param currentTime current event time
     * @return true if the alert should be suppressed (not emitted)
     */
    public static boolean shouldSuppress(CIDAlert alert, ComorbidityState state, long currentTime) {
        if (alert == null || state == null) return false;

        // HALT alerts are NEVER suppressed — patient safety override
        if ("HALT".equals(alert.getSeverity())) return false;

        return state.isSuppressed(alert.getSuppressionKey(), currentTime);
    }

    /**
     * Record that an alert was emitted (for future suppression checks).
     */
    public static void recordEmission(CIDAlert alert, ComorbidityState state, long currentTime) {
        if (alert == null || state == null) return;
        state.recordSuppression(alert.getSuppressionKey(), currentTime);
    }
}
