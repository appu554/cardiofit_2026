package com.cardiofit.flink.models;

/**
 * CID alert severity levels.
 *
 * HALT: Life-threatening. Immediate physician notification (<5 min SLA).
 *       Side-output to ingestion.safety-critical. All Decision Cards paused.
 *
 * PAUSE: Correction loop paused. Physician review within 48 hours.
 *        Main output to alerts.comorbidity-interactions.
 *
 * SOFT_FLAG: Warning attached to Decision Cards. No pause.
 *            Main output to alerts.comorbidity-interactions.
 */
public enum CIDSeverity {
    HALT,
    PAUSE,
    SOFT_FLAG;

    /**
     * Whether this severity level requires immediate safety-critical routing.
     */
    public boolean isSafetyCritical() {
        return this == HALT;
    }

    /**
     * Whether this severity level pauses the correction loop.
     */
    public boolean pausesCorrectionLoop() {
        return this == HALT || this == PAUSE;
    }
}
