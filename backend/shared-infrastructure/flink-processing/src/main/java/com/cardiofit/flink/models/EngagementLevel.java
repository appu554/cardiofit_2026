package com.cardiofit.flink.models;

/**
 * Engagement levels using DD#8 color terminology (GREEN/YELLOW/ORANGE/RED).
 * Channel-aware: thresholds differ by deployment channel.
 */
public enum EngagementLevel {
    GREEN,
    YELLOW,
    ORANGE,
    RED;

    /**
     * Classify score into engagement level using channel-specific thresholds.
     * Uses epsilon for IEEE 754 boundary safety.
     */
    public static EngagementLevel fromScore(double score, EngagementChannel channel) {
        if (score >= channel.getGreenThreshold() - 1e-9) return GREEN;
        if (score >= channel.getYellowThreshold() - 1e-9) return YELLOW;
        if (score >= channel.getOrangeThreshold() - 1e-9) return ORANGE;
        return RED;
    }

    /** Convenience overload defaulting to CORPORATE channel */
    public static EngagementLevel fromScore(double score) {
        return fromScore(score, EngagementChannel.CORPORATE);
    }

    public boolean isAlertWorthy() {
        return this == ORANGE || this == RED;
    }
}
