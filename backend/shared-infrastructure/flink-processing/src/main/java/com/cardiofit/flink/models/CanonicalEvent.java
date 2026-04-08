package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonAlias;
import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.time.Instant;
import java.util.Map;

/**
 * Canonical event model representing standardized healthcare events
 * after validation and transformation.
 *
 * ignoreUnknown = true ensures forward compatibility: when Module 1/1b adds
 * new fields, downstream modules (Module 2, 3, etc.) won't crash on
 * deserialization of events with unknown properties.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class CanonicalEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("eventId")
    private String id;

    @JsonProperty("patientId")
    @JsonAlias("patient_id")
    private String patientId;

    @JsonProperty("encounterId")
    private String encounterId;

    @JsonProperty("eventType")
    private EventType eventType;

    @JsonProperty("timestamp")
    private long eventTime;

    @JsonProperty("processingTime")
    private long processingTime;

    @JsonProperty("sourceSystem")
    private String sourceSystem;

    @JsonProperty("facilityId")
    private String facilityId;

    @JsonProperty("providerId")
    private String providerId;

    @JsonProperty("payload")
    private Map<String, Object> payload;

    @JsonProperty("clinicalContext")
    private ClinicalContext clinicalContext;

    @JsonProperty("ingestionMetadata")
    private IngestionMetadata ingestionMetadata;

    @JsonProperty("metadata")
    private EventMetadata metadata;

    @JsonProperty("correlationId")
    private String correlationId;

    // Default constructor
    public CanonicalEvent() {
        this.processingTime = System.currentTimeMillis();
    }

    // Builder pattern
    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private CanonicalEvent event = new CanonicalEvent();

        public Builder id(String id) {
            event.id = id;
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

        public Builder eventType(EventType eventType) {
            event.eventType = eventType;
            return this;
        }

        public Builder eventTime(long eventTime) {
            event.eventTime = eventTime;
            return this;
        }

        public Builder sourceSystem(String sourceSystem) {
            event.sourceSystem = sourceSystem;
            return this;
        }

        public Builder facilityId(String facilityId) {
            event.facilityId = facilityId;
            return this;
        }

        public Builder providerId(String providerId) {
            event.providerId = providerId;
            return this;
        }

        public Builder payload(Map<String, Object> payload) {
            event.payload = payload;
            return this;
        }

        public Builder clinicalContext(ClinicalContext clinicalContext) {
            event.clinicalContext = clinicalContext;
            return this;
        }

        public Builder metadata(EventMetadata metadata) {
            event.metadata = metadata;
            return this;
        }

        public Builder correlationId(String correlationId) {
            event.correlationId = correlationId;
            return this;
        }

        public CanonicalEvent build() {
            return event;
        }
    }

    // Getters and Setters
    public String getId() { return id; }
    public void setId(String id) { this.id = id; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getEncounterId() { return encounterId; }
    public void setEncounterId(String encounterId) { this.encounterId = encounterId; }

    public EventType getEventType() { return eventType; }
    public void setEventType(EventType eventType) { this.eventType = eventType; }

    public long getEventTime() { return eventTime; }
    public void setEventTime(long eventTime) { this.eventTime = eventTime; }

    public long getProcessingTime() { return processingTime; }
    public void setProcessingTime(long processingTime) { this.processingTime = processingTime; }

    public String getSourceSystem() { return sourceSystem; }
    public void setSourceSystem(String sourceSystem) { this.sourceSystem = sourceSystem; }

    public String getFacilityId() { return facilityId; }
    public void setFacilityId(String facilityId) { this.facilityId = facilityId; }

    public String getProviderId() { return providerId; }
    public void setProviderId(String providerId) { this.providerId = providerId; }

    public Map<String, Object> getPayload() { return payload; }
    public void setPayload(Map<String, Object> payload) { this.payload = payload; }

    public ClinicalContext getClinicalContext() { return clinicalContext; }
    public void setClinicalContext(ClinicalContext clinicalContext) { this.clinicalContext = clinicalContext; }

    public IngestionMetadata getIngestionMetadata() { return ingestionMetadata; }
    public void setIngestionMetadata(IngestionMetadata ingestionMetadata) { this.ingestionMetadata = ingestionMetadata; }

    public EventMetadata getMetadata() { return metadata; }
    public void setMetadata(EventMetadata metadata) { this.metadata = metadata; }

    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String correlationId) { this.correlationId = correlationId; }

    // Alias method for backwards compatibility
    @JsonIgnore
    public long getEventTimestamp() { return eventTime; }

    @Override
    public String toString() {
        return "CanonicalEvent{" +
            "id='" + id + '\'' +
            ", patientId='" + patientId + '\'' +
            ", eventType=" + eventType +
            ", eventTime=" + Instant.ofEpochMilli(eventTime) +
            '}';
    }

    // Helper class for clinical context
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class ClinicalContext implements Serializable {
        private String department;
        private String unit;
        private String bedNumber;
        private String careTeam;
        private String acuityLevel;

        // Constructor, getters, and setters
        public ClinicalContext() {}

        public String getDepartment() { return department; }
        public void setDepartment(String department) { this.department = department; }

        public String getUnit() { return unit; }
        public void setUnit(String unit) { this.unit = unit; }

        public String getBedNumber() { return bedNumber; }
        public void setBedNumber(String bedNumber) { this.bedNumber = bedNumber; }

        public String getCareTeam() { return careTeam; }
        public void setCareTeam(String careTeam) { this.careTeam = careTeam; }

        public String getAcuityLevel() { return acuityLevel; }
        public void setAcuityLevel(String acuityLevel) { this.acuityLevel = acuityLevel; }
    }

    // Helper class for ingestion metadata
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class IngestionMetadata implements Serializable {
        private int ingestionNode;
        private String version;
        private long latencyMs;
        private String validationStatus;

        public IngestionMetadata() {}

        public int getIngestionNode() { return ingestionNode; }
        public void setIngestionNode(int ingestionNode) { this.ingestionNode = ingestionNode; }

        public String getVersion() { return version; }
        public void setVersion(String version) { this.version = version; }

        public long getLatencyMs() { return latencyMs; }
        public void setLatencyMs(long latencyMs) { this.latencyMs = latencyMs; }

        public String getValidationStatus() { return validationStatus; }
        public void setValidationStatus(String validationStatus) { this.validationStatus = validationStatus; }
    }

    // Helper class for event metadata (device, location, source tracking)
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class EventMetadata implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("source")
        private String source;

        @JsonProperty("location")
        private String location;

        @JsonProperty("device_id")
        private String deviceId;

        // Default constructor
        public EventMetadata() {}

        // Constructor with all fields
        public EventMetadata(String source, String location, String deviceId) {
            this.source = source;
            this.location = location;
            this.deviceId = deviceId;
        }

        // Getters and Setters
        public String getSource() { return source; }
        public void setSource(String source) { this.source = source; }

        public String getLocation() { return location; }
        public void setLocation(String location) { this.location = location; }

        public String getDeviceId() { return deviceId; }
        public void setDeviceId(String deviceId) { this.deviceId = deviceId; }

        @Override
        public String toString() {
            return "EventMetadata{" +
                "source='" + source + '\'' +
                ", location='" + location + '\'' +
                ", deviceId='" + deviceId + '\'' +
                '}';
        }
    }
}