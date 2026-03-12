package com.cardiofit.flink.cds.time;

import java.io.Serializable;
import java.time.Duration;
import java.time.Instant;

/**
 * Status of a single time constraint.
 *
 * <p>Contains all information about a time constraint evaluation including:
 * - Constraint identification (ID, bundle name)
 * - Timing information (deadline, time remaining)
 * - Alert status (level, message)
 * - Criticality flag
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class ConstraintStatus implements Serializable {
    private static final long serialVersionUID = 1L;

    private String constraintId;
    private String bundleName;
    private boolean critical;
    private Instant deadline;
    private Duration timeRemaining;
    private AlertLevel alertLevel;
    private String message;

    /**
     * Default constructor.
     */
    public ConstraintStatus() {
    }

    /**
     * Constructor with all fields.
     *
     * @param constraintId Unique constraint identifier
     * @param bundleName Human-readable bundle name
     * @param critical Whether this is a critical constraint
     * @param deadline When the constraint deadline expires
     * @param timeRemaining Duration until deadline
     * @param alertLevel Current alert level
     * @param message Human-readable message
     */
    public ConstraintStatus(String constraintId, String bundleName, boolean critical,
                           Instant deadline, Duration timeRemaining,
                           AlertLevel alertLevel, String message) {
        this.constraintId = constraintId;
        this.bundleName = bundleName;
        this.critical = critical;
        this.deadline = deadline;
        this.timeRemaining = timeRemaining;
        this.alertLevel = alertLevel;
        this.message = message;
    }

    /**
     * Checks if deadline has been exceeded.
     *
     * @return true if time remaining is negative
     */
    public boolean isDeadlineExceeded() {
        return timeRemaining != null && timeRemaining.isNegative();
    }

    /**
     * Gets time remaining in minutes.
     *
     * @return Minutes remaining (negative if exceeded), rounded to nearest minute
     */
    public long getMinutesRemaining() {
        if (timeRemaining == null) {
            return 0;
        }
        // Round to nearest minute instead of truncating
        // Example: 29 min 59 sec → 30 min, 29 min 29 sec → 29 min
        long seconds = timeRemaining.getSeconds();
        return (seconds + 30) / 60;  // Add 30 seconds before dividing for rounding
    }

    // Getters and setters

    public String getConstraintId() {
        return constraintId;
    }

    public void setConstraintId(String constraintId) {
        this.constraintId = constraintId;
    }

    public String getBundleName() {
        return bundleName;
    }

    public void setBundleName(String bundleName) {
        this.bundleName = bundleName;
    }

    public boolean isCritical() {
        return critical;
    }

    public void setCritical(boolean critical) {
        this.critical = critical;
    }

    public Instant getDeadline() {
        return deadline;
    }

    public void setDeadline(Instant deadline) {
        this.deadline = deadline;
    }

    public Duration getTimeRemaining() {
        return timeRemaining;
    }

    public void setTimeRemaining(Duration timeRemaining) {
        this.timeRemaining = timeRemaining;
    }

    public AlertLevel getAlertLevel() {
        return alertLevel;
    }

    public void setAlertLevel(AlertLevel alertLevel) {
        this.alertLevel = alertLevel;
    }

    public String getMessage() {
        return message;
    }

    public void setMessage(String message) {
        this.message = message;
    }

    @Override
    public String toString() {
        return "ConstraintStatus{" +
                "constraintId='" + constraintId + '\'' +
                ", bundleName='" + bundleName + '\'' +
                ", critical=" + critical +
                ", deadline=" + deadline +
                ", timeRemaining=" + timeRemaining +
                ", alertLevel=" + alertLevel +
                ", message='" + message + '\'' +
                '}';
    }
}
