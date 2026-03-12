package com.cardiofit.flink.models;

/**
 * AlertSeverity enum for clinical alert severity levels
 * Maps to clinical urgency and response time requirements:
 * - INFO: Informational, routine monitoring, review within 24 hours
 * - WARNING: Requires monitoring and potential intervention, review within 4-6 hours
 * - HIGH: Urgent attention required, review within 1 hour
 * - CRITICAL: Immediate response required, review within 15 minutes
 *
 * Note: LOW and MODERATE are deprecated aliases for backward compatibility
 */
public enum AlertSeverity {
    INFO,
    WARNING,
    HIGH,
    CRITICAL,

    // Deprecated - for backward compatibility
    @Deprecated
    LOW,
    @Deprecated
    MODERATE;

    /**
     * Get numeric severity score for comparison
     */
    public int getSeverityScore() {
        switch (this) {
            case INFO:
            case LOW:
                return 1;
            case WARNING:
            case MODERATE:
                return 2;
            case HIGH:
                return 3;
            case CRITICAL:
                return 4;
            default:
                return 0;
        }
    }

    /**
     * Check if this severity requires immediate action
     */
    public boolean requiresImmediateAction() {
        return this == CRITICAL;
    }

    /**
     * Check if this severity requires clinical review
     */
    public boolean requiresClinicalReview() {
        return this == HIGH || this == CRITICAL;
    }
}
