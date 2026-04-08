package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.UUID;

/**
 * Alert emitted when engagement drops below threshold or collapses.
 * Published to alerts.engagement-drop.
 * Consumed by notification-service, KB-21.
 */
public class EngagementDropAlert implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum DropType {
        LEVEL_TRANSITION,
        SUSTAINED_LOW,
        CLIFF_DROP
    }

    @JsonProperty("alertId")
    private String alertId;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("dropType")
    private DropType dropType;

    @JsonProperty("severity")
    private String severity;

    @JsonProperty("currentScore")
    private double currentScore;

    @JsonProperty("previousScore")
    private Double previousScore;

    @JsonProperty("currentLevel")
    private EngagementLevel currentLevel;

    @JsonProperty("previousLevel")
    private EngagementLevel previousLevel;

    @JsonProperty("consecutiveLowDays")
    private int consecutiveLowDays;

    @JsonProperty("triggerSummary")
    private String triggerSummary;

    @JsonProperty("recommendedAction")
    private String recommendedAction;

    @JsonProperty("suppressionKey")
    private String suppressionKey;

    @JsonProperty("createdAt")
    private long createdAt;

    public EngagementDropAlert() {}

    public static EngagementDropAlert create(String patientId, DropType dropType,
                                              double currentScore, Double previousScore,
                                              EngagementLevel currentLevel,
                                              EngagementLevel previousLevel,
                                              int consecutiveLowDays) {
        EngagementDropAlert alert = new EngagementDropAlert();
        alert.alertId = UUID.randomUUID().toString();
        alert.patientId = patientId;
        alert.dropType = dropType;
        alert.currentScore = currentScore;
        alert.previousScore = previousScore;
        alert.currentLevel = currentLevel;
        alert.previousLevel = previousLevel;
        alert.consecutiveLowDays = consecutiveLowDays;
        alert.createdAt = System.currentTimeMillis();

        alert.severity = (currentLevel == EngagementLevel.RED
                         || dropType == DropType.CLIFF_DROP)
                         ? "CRITICAL" : "WARNING";

        alert.suppressionKey = dropType.name() + ":" + patientId;

        switch (dropType) {
            case LEVEL_TRANSITION:
                alert.triggerSummary = String.format(
                    "Engagement dropped from %s (%.2f) to %s (%.2f)",
                    previousLevel, previousScore, currentLevel, currentScore);
                alert.recommendedAction = currentLevel == EngagementLevel.RED
                    ? "Physician outreach recommended. Patient has been critically disengaged."
                    : "BCE to shift to re-engagement micro-commitments. Reduce SPAN frequency.";
                break;
            case SUSTAINED_LOW:
                alert.triggerSummary = String.format(
                    "Engagement has been LOW for %d consecutive days (score: %.2f)",
                    consecutiveLowDays, currentScore);
                alert.recommendedAction =
                    "Consider physician follow-up call. Review barriers to engagement.";
                break;
            case CLIFF_DROP:
                alert.triggerSummary = String.format(
                    "Engagement collapsed: %.2f -> %.2f (delta: %.2f) in one day",
                    previousScore, currentScore, currentScore - previousScore);
                alert.recommendedAction =
                    "Urgent: Check for life event disruption. Empathetic outreach, not corrective.";
                break;
        }

        return alert;
    }

    // Getters
    public String getAlertId() { return alertId; }
    public String getPatientId() { return patientId; }
    public DropType getDropType() { return dropType; }
    public String getSeverity() { return severity; }
    public double getCurrentScore() { return currentScore; }
    public Double getPreviousScore() { return previousScore; }
    public EngagementLevel getCurrentLevel() { return currentLevel; }
    public EngagementLevel getPreviousLevel() { return previousLevel; }
    public int getConsecutiveLowDays() { return consecutiveLowDays; }
    public String getTriggerSummary() { return triggerSummary; }
    public String getRecommendedAction() { return recommendedAction; }
    public String getSuppressionKey() { return suppressionKey; }
    public long getCreatedAt() { return createdAt; }
}
