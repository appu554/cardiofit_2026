package com.cardiofit.flink.models;

import java.io.Serializable;

/**
 * Alert Priority Levels (P0-P4) with response time expectations
 *
 * Priority levels are assigned based on multi-dimensional scoring (0-30 points):
 * - Clinical Severity (0-10 pts × 2.0 weight)
 * - Time Sensitivity (0-5 pts × 1.5 weight)
 * - Patient Vulnerability (0-5 pts × 1.0 weight)
 * - Trending Pattern (0-3 pts × 1.5 weight)
 * - Confidence Score (0-2 pts × 0.5 weight)
 *
 * Reference: ALERT_PRIORITIZATION_DESIGN.md
 */
public enum AlertPriority implements Serializable {

    /**
     * CRITICAL - Alias for P0_CRITICAL (for simplified test usage)
     */
    CRITICAL(25, 30, "CRITICAL", "Immediate (<5 min)", new String[]{"PUSH", "SMS", "PAGE", "ALARM"}),

    /**
     * P0 - CRITICAL: Immediate response required (<5 minutes)
     * Examples: Cardiac arrest, severe septic shock, respiratory failure
     * Notification: Push + SMS + Page + Alarm
     */
    P0_CRITICAL(25, 30, "CRITICAL", "Immediate (<5 min)", new String[]{"PUSH", "SMS", "PAGE", "ALARM"}),

    /**
     * P1 - URGENT: Urgent intervention needed (<15 minutes)
     * Examples: Sepsis-3 criteria met, acute MI, severe hypotension, NEWS2 ≥7
     * Notification: Push + SMS + Desktop alert
     */
    P1_URGENT(15, 24, "URGENT", "<15 minutes", new String[]{"PUSH", "SMS", "DESKTOP"}),

    /**
     * P2 - HIGH: Prompt clinical review required (<1 hour)
     * Examples: SIRS criteria, moderate hypotension, persistent fever
     * Notification: Push notification + Badge
     */
    P2_HIGH(10, 14, "HIGH", "<1 hour", new String[]{"PUSH", "BADGE"}),

    /**
     * P3 - MEDIUM: Routine clinical review (1-4 hours)
     * Examples: Mild abnormalities, trending concerns, early warnings
     * Notification: Badge count update
     */
    P3_MEDIUM(5, 9, "MEDIUM", "1-4 hours", new String[]{"BADGE"}),

    /**
     * P4 - LOW: Scheduled or administrative (>4 hours)
     * Examples: Routine monitoring, medication due, documentation
     * Notification: Silent/inbox only
     */
    P4_LOW(0, 4, "LOW", ">4 hours", new String[]{"SILENT"});

    private final int minScore;
    private final int maxScore;
    private final String label;
    private final String responseTime;
    private final String[] notificationChannels;

    AlertPriority(int minScore, int maxScore, String label, String responseTime, String[] channels) {
        this.minScore = minScore;
        this.maxScore = maxScore;
        this.label = label;
        this.responseTime = responseTime;
        this.notificationChannels = channels;
    }

    /**
     * Determine priority level from priority score
     *
     * @param score Priority score (0-30)
     * @return AlertPriority level
     */
    public static AlertPriority fromScore(double score) {
        int roundedScore = (int) Math.round(score);

        // Ensure score is within valid range
        if (roundedScore > 30) roundedScore = 30;
        if (roundedScore < 0) roundedScore = 0;

        // Find matching priority level
        for (AlertPriority priority : values()) {
            if (roundedScore >= priority.minScore && roundedScore <= priority.maxScore) {
                return priority;
            }
        }

        // Fallback to P4_LOW (should never reach here with valid input)
        return P4_LOW;
    }

    // Getters

    public int getMinScore() {
        return minScore;
    }

    public int getMaxScore() {
        return maxScore;
    }

    public String getLabel() {
        return label;
    }

    public String getResponseTime() {
        return responseTime;
    }

    public String[] getNotificationChannels() {
        return notificationChannels;
    }

    @Override
    public String toString() {
        return this.name() + " (" + label + ", " + responseTime + ")";
    }
}
