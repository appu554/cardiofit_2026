package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;

/**
 * Alert suppression and deduplication manager.
 *
 * Suppression rules:
 * - HALT alerts: Short dedup window (5 min) to prevent rapid-fire duplicates
 *   from successive events for the same patient. This preserves patient safety
 *   (HALTs re-fire if the clinical situation persists after 5 min) while
 *   eliminating noise from same-batch event processing.
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
     * HALT dedup window: 4 hours.
     * Prevents alert fatigue from repeated HALT alerts for the same persistent condition
     * while ensuring re-fire within a clinically meaningful interval. Combined with
     * the event-type guard in Module8_ComorbidityEngine (which skips evaluation for
     * non-clinical events like DEVICE_READING), this effectively limits HALT alerts
     * to once per 4 hours per unique rule+patient+medication combination.
     */
    static final long HALT_DEDUP_WINDOW_MS = 4 * 60 * 60 * 1000L;

    /**
     * Determine if an alert should be suppressed.
     * @param alert the candidate alert
     * @param state current patient state (contains suppression history)
     * @param currentTime current event time
     * @return true if the alert should be suppressed (not emitted)
     */
    public static boolean shouldSuppress(CIDAlert alert, ComorbidityState state, long currentTime) {
        if (alert == null || state == null) return false;

        // HALT alerts use a short dedup window (5 min) instead of the full 72h.
        // This prevents duplicate alerts from rapid successive events while
        // ensuring genuinely new clinical situations still fire.
        if ("HALT".equals(alert.getSeverity())) {
            return state.isSuppressedWithWindow(alert.getSuppressionKey(),
                currentTime, HALT_DEDUP_WINDOW_MS);
        }

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
