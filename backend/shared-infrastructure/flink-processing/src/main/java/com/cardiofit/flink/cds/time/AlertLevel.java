package com.cardiofit.flink.cds.time;

/**
 * Alert level for time constraints.
 *
 * <p>Classification:
 * - INFO: On track, more than 30 minutes remaining
 * - WARNING: Less than 30 minutes remaining (requires attention)
 * - CRITICAL: Deadline exceeded (immediate action required)
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public enum AlertLevel {
    /**
     * On track - more than 30 minutes remaining until deadline.
     * No urgent action required.
     */
    INFO,

    /**
     * Warning - less than or equal to 30 minutes remaining.
     * Attention required to meet deadline.
     */
    WARNING,

    /**
     * Critical - deadline exceeded.
     * Immediate action required, bundle compliance failed.
     */
    CRITICAL
}
