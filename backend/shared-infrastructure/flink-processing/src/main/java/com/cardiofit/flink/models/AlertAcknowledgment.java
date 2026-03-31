package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Physician acknowledgment flowing back via alert-acknowledgments.v1.
 */
public class AlertAcknowledgment implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum AckAction { ACKNOWLEDGE, ACTION_TAKEN, DISMISS }

    @JsonProperty("alert_id") private String alertId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("clinical_category") private String clinicalCategory;
    @JsonProperty("action") private AckAction action;
    @JsonProperty("practitioner_id") private String practitionerId;
    @JsonProperty("action_description") private String actionDescription;
    @JsonProperty("timestamp") private long timestamp;

    public AlertAcknowledgment() {}

    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public String getClinicalCategory() { return clinicalCategory; }
    public void setClinicalCategory(String clinicalCategory) { this.clinicalCategory = clinicalCategory; }
    public AckAction getAction() { return action; }
    public void setAction(AckAction action) { this.action = action; }
    public String getPractitionerId() { return practitionerId; }
    public void setPractitionerId(String practitionerId) { this.practitionerId = practitionerId; }
    public String getActionDescription() { return actionDescription; }
    public void setActionDescription(String actionDescription) { this.actionDescription = actionDescription; }
    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }
}
