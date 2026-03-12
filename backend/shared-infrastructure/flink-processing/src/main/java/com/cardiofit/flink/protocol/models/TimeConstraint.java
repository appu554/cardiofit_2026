package com.cardiofit.flink.protocol.models;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Time constraint for clinical bundles.
 *
 * <p>Defines a time-based requirement for protocol compliance.
 * Examples:
 * - Sepsis Hour-0 Bundle: Complete within 60 minutes of trigger
 * - Sepsis Hour-1 Bundle: Complete within 60 minutes of trigger
 * - STEMI Door-to-Balloon: Complete within 90 minutes
 *
 * <p>Calculation: deadline = trigger_time + offset_minutes
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class TimeConstraint implements Serializable {
    private static final long serialVersionUID = 1L;

    private String constraintId;
    private String bundleName;
    private int offsetMinutes;
    private boolean critical;
    private List<String> actionReferences;

    /**
     * Default constructor.
     */
    public TimeConstraint() {
        this.actionReferences = new ArrayList<>();
    }

    /**
     * Constructor with all fields.
     *
     * @param constraintId Unique constraint identifier
     * @param bundleName Human-readable bundle name
     * @param offsetMinutes Minutes from trigger to deadline
     * @param critical Whether this is a critical constraint
     */
    public TimeConstraint(String constraintId, String bundleName, int offsetMinutes, boolean critical) {
        this.constraintId = constraintId;
        this.bundleName = bundleName;
        this.offsetMinutes = offsetMinutes;
        this.critical = critical;
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

    public int getOffsetMinutes() {
        return offsetMinutes;
    }

    public void setOffsetMinutes(int offsetMinutes) {
        this.offsetMinutes = offsetMinutes;
    }

    public boolean isCritical() {
        return critical;
    }

    public void setCritical(boolean critical) {
        this.critical = critical;
    }

    public List<String> getActionReferences() {
        return actionReferences;
    }

    public void setActionReferences(List<String> actionReferences) {
        this.actionReferences = actionReferences;
    }

    @Override
    public String toString() {
        return "TimeConstraint{" +
                "constraintId='" + constraintId + '\'' +
                ", bundleName='" + bundleName + '\'' +
                ", offsetMinutes=" + offsetMinutes +
                ", critical=" + critical +
                ", actionReferences=" + (actionReferences != null ? actionReferences.size() : 0) +
                '}';
    }
}
