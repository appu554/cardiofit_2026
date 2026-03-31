package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.ActionTier;

import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * Cross-module alert deduplication.
 *
 * Different from Module 4's within-pattern dedup. Module 6 deduplicates ACROSS modules —
 * three modules detecting the same clinical situation produce ONE alert.
 *
 * Dedup windows are tier-specific:
 * - HALT:      5 minutes  (critical alerts need fast re-evaluation)
 * - PAUSE:     30 minutes (physician review window)
 * - SOFT_FLAG: 60 minutes (advisory, lower noise)
 * - ROUTINE:   no dedup   (always pass through)
 */
public class Module6CrossModuleDedup implements Serializable {
    private static final long serialVersionUID = 1L;

    private static final long HALT_DEDUP_WINDOW_MS = 5 * 60 * 1000L;
    private static final long PAUSE_DEDUP_WINDOW_MS = 30 * 60 * 1000L;
    private static final long SOFT_FLAG_DEDUP_WINDOW_MS = 60 * 60 * 1000L;

    private final Map<String, Long> recentAlertState = new HashMap<>();

    public boolean shouldEmit(String patientId, ActionTier tier,
                              String clinicalCategory, long eventTime) {

        if (tier == ActionTier.ROUTINE) return true;

        String dedupKey = patientId + ":" + tier + ":" + clinicalCategory;
        Long lastEmitted = recentAlertState.get(dedupKey);

        long window = switch (tier) {
            case HALT -> HALT_DEDUP_WINDOW_MS;
            case PAUSE -> PAUSE_DEDUP_WINDOW_MS;
            case SOFT_FLAG -> SOFT_FLAG_DEDUP_WINDOW_MS;
            default -> Long.MAX_VALUE;
        };

        if (lastEmitted != null && (eventTime - lastEmitted) < window) {
            return false; // within dedup window — suppress
        }

        recentAlertState.put(dedupKey, eventTime);
        return true;
    }

    /** Remove entries older than the maximum window to bound memory. */
    public void pruneExpired(long currentTime) {
        recentAlertState.entrySet().removeIf(entry ->
            (currentTime - entry.getValue()) > SOFT_FLAG_DEDUP_WINDOW_MS);
    }
}
