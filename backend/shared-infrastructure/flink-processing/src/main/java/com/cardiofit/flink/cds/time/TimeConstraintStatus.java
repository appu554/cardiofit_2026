package com.cardiofit.flink.cds.time;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Status of all time constraints for a protocol.
 *
 * <p>Container class that holds the evaluation results for all time constraints
 * associated with a clinical protocol. Provides convenience methods to query
 * critical and warning alerts.
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class TimeConstraintStatus implements Serializable {
    private static final long serialVersionUID = 1L;

    private String protocolId;
    private List<ConstraintStatus> constraintStatuses = new ArrayList<>();

    /**
     * Default constructor.
     */
    public TimeConstraintStatus() {
    }

    /**
     * Constructor with protocol ID.
     *
     * @param protocolId The protocol identifier
     */
    public TimeConstraintStatus(String protocolId) {
        this.protocolId = protocolId;
    }

    /**
     * Adds a constraint status to the collection.
     *
     * @param constraintStatus The constraint status to add
     */
    public void addConstraintStatus(ConstraintStatus constraintStatus) {
        this.constraintStatuses.add(constraintStatus);
    }

    /**
     * Checks if any critical alerts are present.
     *
     * @return true if at least one constraint has CRITICAL alert level
     */
    public boolean hasCriticalAlerts() {
        return constraintStatuses.stream()
            .anyMatch(cs -> cs.getAlertLevel() == AlertLevel.CRITICAL);
    }

    /**
     * Checks if any warning alerts are present.
     *
     * @return true if at least one constraint has WARNING alert level
     */
    public boolean hasWarningAlerts() {
        return constraintStatuses.stream()
            .anyMatch(cs -> cs.getAlertLevel() == AlertLevel.WARNING);
    }

    /**
     * Gets all critical alerts.
     *
     * @return List of constraint statuses with CRITICAL alert level
     */
    public List<ConstraintStatus> getCriticalAlerts() {
        return constraintStatuses.stream()
            .filter(cs -> cs.getAlertLevel() == AlertLevel.CRITICAL)
            .collect(Collectors.toList());
    }

    /**
     * Gets all warning alerts.
     *
     * @return List of constraint statuses with WARNING alert level
     */
    public List<ConstraintStatus> getWarningAlerts() {
        return constraintStatuses.stream()
            .filter(cs -> cs.getAlertLevel() == AlertLevel.WARNING)
            .collect(Collectors.toList());
    }

    /**
     * Gets all info-level statuses.
     *
     * @return List of constraint statuses with INFO alert level
     */
    public List<ConstraintStatus> getInfoStatuses() {
        return constraintStatuses.stream()
            .filter(cs -> cs.getAlertLevel() == AlertLevel.INFO)
            .collect(Collectors.toList());
    }

    // Getters and setters

    public String getProtocolId() {
        return protocolId;
    }

    public void setProtocolId(String protocolId) {
        this.protocolId = protocolId;
    }

    public List<ConstraintStatus> getConstraintStatuses() {
        return constraintStatuses;
    }

    public void setConstraintStatuses(List<ConstraintStatus> constraintStatuses) {
        this.constraintStatuses = constraintStatuses;
    }

    @Override
    public String toString() {
        return "TimeConstraintStatus{" +
                "protocolId='" + protocolId + '\'' +
                ", totalConstraints=" + constraintStatuses.size() +
                ", criticalAlerts=" + getCriticalAlerts().size() +
                ", warningAlerts=" + getWarningAlerts().size() +
                '}';
    }
}
