package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * AlertHistory tracks alert firing history for deduplication in Module 6
 *
 * Used to implement 30-minute suppression window to prevent alert fatigue:
 * - Tracks when an alert type last fired for a patient
 * - Counts how many duplicate alerts were suppressed
 * - Enables time-based suppression logic
 *
 * Stored in Flink MapState keyed by alert type or alert type + patient ID
 *
 * Example usage in Flink ProcessFunction:
 * <pre>
 * MapStateDescriptor<String, AlertHistory> descriptor =
 *     new MapStateDescriptor<>("alert-history", String.class, AlertHistory.class);
 * MapState<String, AlertHistory> state = getRuntimeContext().getMapState(descriptor);
 *
 * String alertKey = alert.getAlertType() + ":" + alert.getPatientId();
 * AlertHistory history = state.get(alertKey);
 *
 * if (history != null && history.shouldSuppress(System.currentTimeMillis(), 30 * 60 * 1000L)) {
 *     history.incrementSuppressedCount();
 *     state.put(alertKey, history);
 *     return; // Suppress duplicate
 * }
 *
 * // Fire new alert
 * state.put(alertKey, new AlertHistory(
 *     alert.getAlertType(),
 *     alert.getPatientId(),
 *     System.currentTimeMillis(),
 *     alert.getSeverity()
 * ));
 * </pre>
 */
public class AlertHistory implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("alert_type")
    private String alertType;

    @JsonProperty("patient_id")
    private String patientId;

    @JsonProperty("last_fired_time")
    private long lastFiredTime;

    @JsonProperty("suppressed_count")
    private int suppressedCount;

    @JsonProperty("last_severity")
    private AlertSeverity lastSeverity;

    public AlertHistory() {
        this.suppressedCount = 0;
    }

    public AlertHistory(String alertType, String patientId, long lastFiredTime, AlertSeverity lastSeverity) {
        this.alertType = alertType;
        this.patientId = patientId;
        this.lastFiredTime = lastFiredTime;
        this.suppressedCount = 0;
        this.lastSeverity = lastSeverity;
    }

    /**
     * Determine if the alert should be suppressed based on time window
     *
     * @param currentTime Current timestamp in milliseconds
     * @param suppressionWindowMs Suppression window duration in milliseconds (typically 30 minutes = 1,800,000 ms)
     * @return true if alert should be suppressed, false if it should fire
     */
    public boolean shouldSuppress(long currentTime, long suppressionWindowMs) {
        if (lastFiredTime == 0) {
            return false; // Never fired before, don't suppress
        }

        long timeSinceLastFired = currentTime - lastFiredTime;
        return timeSinceLastFired < suppressionWindowMs;
    }

    /**
     * Increment the suppressed count and update last fired time
     */
    public void incrementSuppressedCount() {
        this.suppressedCount++;
        this.lastFiredTime = System.currentTimeMillis();
    }

    /**
     * Update the alert history with a new firing
     *
     * @param currentTime Current timestamp
     * @param severity Alert severity
     */
    public void updateFiring(long currentTime, AlertSeverity severity) {
        this.lastFiredTime = currentTime;
        this.lastSeverity = severity;
        this.suppressedCount = 0; // Reset suppression count on new firing
    }

    /**
     * Check if severity has escalated since last firing
     *
     * @param currentSeverity Current alert severity
     * @return true if severity has increased (e.g., WARNING -> CRITICAL)
     */
    public boolean hasEscalatedSeverity(AlertSeverity currentSeverity) {
        if (lastSeverity == null) {
            return false;
        }
        return currentSeverity.getSeverityScore() > lastSeverity.getSeverityScore();
    }

    // Getters and Setters
    public String getAlertType() { return alertType; }
    public void setAlertType(String alertType) { this.alertType = alertType; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public long getLastFiredTime() { return lastFiredTime; }
    public void setLastFiredTime(long lastFiredTime) { this.lastFiredTime = lastFiredTime; }

    public int getSuppressedCount() { return suppressedCount; }
    public void setSuppressedCount(int suppressedCount) { this.suppressedCount = suppressedCount; }

    public AlertSeverity getLastSeverity() { return lastSeverity; }
    public void setLastSeverity(AlertSeverity lastSeverity) { this.lastSeverity = lastSeverity; }

    @Override
    public String toString() {
        return "AlertHistory{" +
                "alertType='" + alertType + '\'' +
                ", patientId='" + patientId + '\'' +
                ", lastFiredTime=" + lastFiredTime +
                ", suppressedCount=" + suppressedCount +
                ", lastSeverity=" + lastSeverity +
                '}';
    }
}
