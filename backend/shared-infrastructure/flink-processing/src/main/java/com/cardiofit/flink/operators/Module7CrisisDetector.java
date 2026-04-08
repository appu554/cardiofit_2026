package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.BPReading;

/**
 * Hypertensive crisis detection. Evaluated FIRST on every reading,
 * before any windowed computation.
 *
 * Crisis thresholds (per ACC/AHA):
 *   SBP >= 180 mmHg OR DBP >= 120 mmHg
 *
 * Acute surge detection:
 *   SBP increase > 30 mmHg within < 1 hour
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module7CrisisDetector {

    private Module7CrisisDetector() {}

    /** Inclusive threshold per ACC/AHA: SBP >= 180 OR DBP >= 120. */
    public static boolean isCrisis(BPReading reading) {
        if (reading == null || reading.getSystolic() == null || reading.getDiastolic() == null) {
            return false;
        }
        return reading.getSystolic() >= 180.0 || reading.getDiastolic() >= 120.0;
    }

    /**
     * Acute surge: SBP increase > 30 mmHg within < 1 hour.
     * Requires a previous reading for comparison.
     */
    public static boolean isAcuteSurge(BPReading previous, BPReading current) {
        if (previous == null || current == null) return false;
        if (previous.getSystolic() == null || current.getSystolic() == null) return false;

        long timeDeltaMs = current.getTimestamp() - previous.getTimestamp();
        if (timeDeltaMs <= 0 || timeDeltaMs > 60 * 60 * 1000L) return false;

        double sbpDelta = current.getSystolic() - previous.getSystolic();
        return sbpDelta > 30.0;
    }
}
