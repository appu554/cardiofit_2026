package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * Protocol Event model for audit trail of clinical protocol triggers
 * Phase 4 Enhancement: Tracks when and why clinical protocols are recommended
 *
 * Purpose: Provides complete audit trail for clinical decision support by recording:
 * - What protocol was triggered
 * - Why it was triggered (clinical indicators)
 * - When it was triggered
 * - What patient data led to the trigger
 *
 * Use cases:
 * - Clinical audit and quality improvement
 * - Regulatory compliance and documentation
 * - Performance analytics for CDS system
 * - Research and protocol effectiveness evaluation
 */
public class ProtocolEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("event_id")
    private String eventId;

    @JsonProperty("patient_id")
    private String patientId;

    @JsonProperty("encounter_id")
    private String encounterId;

    @JsonProperty("protocol_name")
    private String protocolName;

    @JsonProperty("protocol_category")
    private String protocolCategory;

    @JsonProperty("trigger_reason")
    private String triggerReason;

    @JsonProperty("severity")
    private String severity; // CRITICAL, HIGH, MEDIUM, LOW

    @JsonProperty("triggered_at")
    private long triggeredAt;

    @JsonProperty("clinical_indicators")
    private Map<String, Object> clinicalIndicators;

    @JsonProperty("recommended_actions")
    private String recommendedActions;

    @JsonProperty("source_event_id")
    private String sourceEventId;

    // Default constructor
    public ProtocolEvent() {
        this.triggeredAt = System.currentTimeMillis();
        this.clinicalIndicators = new HashMap<>();
    }

    // Builder pattern
    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private ProtocolEvent event = new ProtocolEvent();

        public Builder eventId(String eventId) {
            event.eventId = eventId;
            return this;
        }

        public Builder patientId(String patientId) {
            event.patientId = patientId;
            return this;
        }

        public Builder encounterId(String encounterId) {
            event.encounterId = encounterId;
            return this;
        }

        public Builder protocolName(String protocolName) {
            event.protocolName = protocolName;
            return this;
        }

        public Builder protocolCategory(String protocolCategory) {
            event.protocolCategory = protocolCategory;
            return this;
        }

        public Builder triggerReason(String triggerReason) {
            event.triggerReason = triggerReason;
            return this;
        }

        public Builder severity(String severity) {
            event.severity = severity;
            return this;
        }

        public Builder triggeredAt(long triggeredAt) {
            event.triggeredAt = triggeredAt;
            return this;
        }

        public Builder clinicalIndicators(Map<String, Object> clinicalIndicators) {
            event.clinicalIndicators = clinicalIndicators;
            return this;
        }

        public Builder addClinicalIndicator(String key, Object value) {
            if (event.clinicalIndicators == null) {
                event.clinicalIndicators = new HashMap<>();
            }
            event.clinicalIndicators.put(key, value);
            return this;
        }

        public Builder recommendedActions(String recommendedActions) {
            event.recommendedActions = recommendedActions;
            return this;
        }

        public Builder sourceEventId(String sourceEventId) {
            event.sourceEventId = sourceEventId;
            return this;
        }

        public ProtocolEvent build() {
            return event;
        }
    }

    // Getters and Setters
    public String getEventId() { return eventId; }
    public void setEventId(String eventId) { this.eventId = eventId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getEncounterId() { return encounterId; }
    public void setEncounterId(String encounterId) { this.encounterId = encounterId; }

    public String getProtocolName() { return protocolName; }
    public void setProtocolName(String protocolName) { this.protocolName = protocolName; }

    public String getProtocolCategory() { return protocolCategory; }
    public void setProtocolCategory(String protocolCategory) { this.protocolCategory = protocolCategory; }

    public String getTriggerReason() { return triggerReason; }
    public void setTriggerReason(String triggerReason) { this.triggerReason = triggerReason; }

    public String getSeverity() { return severity; }
    public void setSeverity(String severity) { this.severity = severity; }

    public long getTriggeredAt() { return triggeredAt; }
    public void setTriggeredAt(long triggeredAt) { this.triggeredAt = triggeredAt; }

    public Map<String, Object> getClinicalIndicators() { return clinicalIndicators; }
    public void setClinicalIndicators(Map<String, Object> clinicalIndicators) {
        this.clinicalIndicators = clinicalIndicators;
    }

    public String getRecommendedActions() { return recommendedActions; }
    public void setRecommendedActions(String recommendedActions) {
        this.recommendedActions = recommendedActions;
    }

    public String getSourceEventId() { return sourceEventId; }
    public void setSourceEventId(String sourceEventId) { this.sourceEventId = sourceEventId; }

    @Override
    public String toString() {
        return "ProtocolEvent{" +
            "eventId='" + eventId + '\'' +
            ", patientId='" + patientId + '\'' +
            ", protocolName='" + protocolName + '\'' +
            ", severity='" + severity + '\'' +
            ", triggeredAt=" + triggeredAt +
            '}';
    }
}
