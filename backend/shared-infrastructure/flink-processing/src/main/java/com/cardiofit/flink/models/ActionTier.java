package com.cardiofit.flink.models;

/**
 * Three-tier clinical action severity model + routine.
 *
 * HALT  = Critical safety — SMS + FCM + phone fallback, 30 min SLA
 * PAUSE = Needs physician review — FCM push + email, 24 hr SLA
 * SOFT_FLAG = Advisory — attached to next Decision Card, no SLA
 * ROUTINE = No action required
 */
public enum ActionTier {
    HALT(1, 30 * 60 * 1000L),
    PAUSE(2, 24 * 60 * 60 * 1000L),
    SOFT_FLAG(3, -1L),
    ROUTINE(4, -1L);

    private final int priority;
    private final long slaMs;

    ActionTier(int priority, long slaMs) {
        this.priority = priority;
        this.slaMs = slaMs;
    }

    public int getPriority() { return priority; }
    public long getSlaMs() { return slaMs; }
    public boolean requiresNotification() { return this == HALT || this == PAUSE; }
    public boolean requiresEscalation() { return slaMs > 0; }
}
