package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.Map;

/**
 * Analytics Event model for Phase 3 supporting systems
 * Optimized for high-throughput analytics consumption (ClickHouse, etc.)
 */
public class AnalyticsEvent {

    @JsonProperty("event_id")
    private String eventId;

    @JsonProperty("patient_id")
    private String patientId;

    @JsonProperty("event_type")
    private String eventType;

    @JsonProperty("timestamp")
    private Long timestamp;

    @JsonProperty("clinical_significance")
    private Double clinicalSignificance;

    @JsonProperty("metrics")
    private Map<String, Object> metrics;

    @JsonProperty("dimensions")
    private Map<String, String> dimensions;

    @JsonProperty("event_hour")
    private String eventHour;  // For time-based partitioning

    @JsonProperty("organization_id")
    private String organizationId;

    @JsonProperty("department")
    private String department;

    @JsonProperty("session_id")
    private String sessionId;

    // Constructors
    public AnalyticsEvent() {}

    public AnalyticsEvent(String eventId, String patientId, String eventType, Long timestamp) {
        this.eventId = eventId;
        this.patientId = patientId;
        this.eventType = eventType;
        this.timestamp = timestamp;
        this.eventHour = generateEventHour(timestamp);
    }

    // Getters and Setters
    public String getEventId() { return eventId; }
    public void setEventId(String eventId) { this.eventId = eventId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getEventType() { return eventType; }
    public void setEventType(String eventType) { this.eventType = eventType; }

    public Long getTimestamp() { return timestamp; }
    public void setTimestamp(Long timestamp) {
        this.timestamp = timestamp;
        this.eventHour = generateEventHour(timestamp);
    }

    public Double getClinicalSignificance() { return clinicalSignificance; }
    public void setClinicalSignificance(Double clinicalSignificance) { this.clinicalSignificance = clinicalSignificance; }

    public Map<String, Object> getMetrics() { return metrics; }
    public void setMetrics(Map<String, Object> metrics) { this.metrics = metrics; }

    public Map<String, String> getDimensions() { return dimensions; }
    public void setDimensions(Map<String, String> dimensions) { this.dimensions = dimensions; }

    public String getEventHour() { return eventHour; }
    public void setEventHour(String eventHour) { this.eventHour = eventHour; }

    public String getOrganizationId() { return organizationId; }
    public void setOrganizationId(String organizationId) { this.organizationId = organizationId; }

    public String getDepartment() { return department; }
    public void setDepartment(String department) { this.department = department; }

    public String getSessionId() { return sessionId; }
    public void setSessionId(String sessionId) { this.sessionId = sessionId; }

    // Helper methods
    private String generateEventHour(Long timestamp) {
        if (timestamp == null) return null;
        return String.valueOf(timestamp / (1000 * 60 * 60)); // Hour-based partitioning
    }

    /**
     * Add a metric to the metrics map
     */
    public void addMetric(String key, Object value) {
        if (metrics == null) {
            metrics = new java.util.HashMap<>();
        }
        metrics.put(key, value);
    }

    /**
     * Add a dimension to the dimensions map
     */
    public void addDimension(String key, String value) {
        if (dimensions == null) {
            dimensions = new java.util.HashMap<>();
        }
        dimensions.put(key, value);
    }

    @Override
    public String toString() {
        return "AnalyticsEvent{" +
                "eventId='" + eventId + '\'' +
                ", patientId='" + patientId + '\'' +
                ", eventType='" + eventType + '\'' +
                ", timestamp=" + timestamp +
                ", clinicalSignificance=" + clinicalSignificance +
                ", eventHour='" + eventHour + '\'' +
                '}';
    }
}