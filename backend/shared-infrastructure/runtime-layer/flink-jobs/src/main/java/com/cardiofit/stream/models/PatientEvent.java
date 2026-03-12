package com.cardiofit.stream.models;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.time.LocalDateTime;
import java.util.Map;
import java.util.Objects;

/**
 * Patient Event Model for Flink Stream Processing
 * Represents clinical events from various sources (medications, vitals, lab results, etc.)
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class PatientEvent {

    @JsonProperty("event_id")
    private String eventId;

    @JsonProperty("patient_id")
    private String patientId;

    @JsonProperty("encounter_id")
    private String encounterId;

    @JsonProperty("event_type")
    private String eventType; // medication_order, lab_result, vital_signs, etc.

    @JsonProperty("event_category")
    private String eventCategory; // clinical, administrative, workflow

    @JsonProperty("timestamp")
    @JsonFormat(pattern = "yyyy-MM-dd'T'HH:mm:ss")
    private LocalDateTime timestamp;

    @JsonProperty("source_system")
    private String sourceSystem; // EHR, device, workflow_engine, etc.

    @JsonProperty("priority")
    private String priority; // low, medium, high, critical

    @JsonProperty("clinical_data")
    private Map<String, Object> clinicalData;

    @JsonProperty("metadata")
    private Map<String, Object> metadata;

    // Default constructor for Jackson
    public PatientEvent() {
    }

    // Constructor
    public PatientEvent(String eventId, String patientId, String eventType,
                       LocalDateTime timestamp, String sourceSystem) {
        this.eventId = eventId;
        this.patientId = patientId;
        this.eventType = eventType;
        this.timestamp = timestamp;
        this.sourceSystem = sourceSystem;
        this.priority = "medium"; // default priority
    }

    // Getters and setters
    public String getEventId() {
        return eventId;
    }

    public void setEventId(String eventId) {
        this.eventId = eventId;
    }

    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public String getEncounterId() {
        return encounterId;
    }

    public void setEncounterId(String encounterId) {
        this.encounterId = encounterId;
    }

    public String getEventType() {
        return eventType;
    }

    public void setEventType(String eventType) {
        this.eventType = eventType;
    }

    public String getEventCategory() {
        return eventCategory;
    }

    public void setEventCategory(String eventCategory) {
        this.eventCategory = eventCategory;
    }

    public LocalDateTime getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(LocalDateTime timestamp) {
        this.timestamp = timestamp;
    }

    public String getSourceSystem() {
        return sourceSystem;
    }

    public void setSourceSystem(String sourceSystem) {
        this.sourceSystem = sourceSystem;
    }

    public String getPriority() {
        return priority;
    }

    public void setPriority(String priority) {
        this.priority = priority;
    }

    public Map<String, Object> getClinicalData() {
        return clinicalData;
    }

    public void setClinicalData(Map<String, Object> clinicalData) {
        this.clinicalData = clinicalData;
    }

    public Map<String, Object> getMetadata() {
        return metadata;
    }

    public void setMetadata(Map<String, Object> metadata) {
        this.metadata = metadata;
    }

    // Helper methods
    public boolean isClinicalEvent() {
        return "clinical".equals(eventCategory);
    }

    public boolean isHighPriority() {
        return "high".equals(priority) || "critical".equals(priority);
    }

    public boolean isCritical() {
        return "critical".equals(priority);
    }

    /**
     * Check if this event requires immediate processing (< 500ms target)
     */
    public boolean requiresImmediateProcessing() {
        return isCritical() ||
               "medication_order".equals(eventType) ||
               "safety_alert".equals(eventType) ||
               "critical_lab_result".equals(eventType);
    }

    /**
     * Get processing timeout in milliseconds based on priority
     */
    public long getProcessingTimeoutMs() {
        switch (priority) {
            case "critical":
                return 200L; // 200ms for critical events
            case "high":
                return 500L; // 500ms for high priority
            case "medium":
                return 2000L; // 2s for medium priority
            case "low":
            default:
                return 5000L; // 5s for low priority
        }
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        PatientEvent that = (PatientEvent) o;
        return Objects.equals(eventId, that.eventId) &&
               Objects.equals(patientId, that.patientId) &&
               Objects.equals(timestamp, that.timestamp);
    }

    @Override
    public int hashCode() {
        return Objects.hash(eventId, patientId, timestamp);
    }

    @Override
    public String toString() {
        return "PatientEvent{" +
               "eventId='" + eventId + '\'' +
               ", patientId='" + patientId + '\'' +
               ", eventType='" + eventType + '\'' +
               ", priority='" + priority + '\'' +
               ", timestamp=" + timestamp +
               ", sourceSystem='" + sourceSystem + '\'' +
               '}';
    }
}