package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.Map;

/**
 * Raw event model representing incoming healthcare events from various sources
 */
public class RawEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("id")
    private String id;

    @JsonProperty("source")
    private String source;

    @JsonProperty("type")
    private String type;

    @JsonProperty("patient_id")
    private String patientId;

    @JsonProperty("encounter_id")
    private String encounterId;

    @JsonProperty("event_time")
    private long eventTime;

    @JsonProperty("received_time")
    private long receivedTime;

    @JsonProperty("payload")
    private Map<String, Object> payload;

    @JsonProperty("metadata")
    private Map<String, String> metadata;

    @JsonProperty("correlation_id")
    private String correlationId;

    @JsonProperty("version")
    private String version;

    // Default constructor
    public RawEvent() {
        this.receivedTime = System.currentTimeMillis();
        this.version = "1.0";
    }

    // Builder pattern for easy construction
    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private RawEvent event = new RawEvent();

        public Builder id(String id) {
            event.id = id;
            return this;
        }

        public Builder source(String source) {
            event.source = source;
            return this;
        }

        public Builder type(String type) {
            event.type = type;
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

        public Builder eventTime(long eventTime) {
            event.eventTime = eventTime;
            return this;
        }

        public Builder payload(Map<String, Object> payload) {
            event.payload = payload;
            return this;
        }

        public Builder metadata(Map<String, String> metadata) {
            event.metadata = metadata;
            return this;
        }

        public Builder correlationId(String correlationId) {
            event.correlationId = correlationId;
            return this;
        }

        public RawEvent build() {
            return event;
        }
    }

    // Getters and Setters
    public String getId() { return id; }
    public void setId(String id) { this.id = id; }

    public String getSource() { return source; }
    public void setSource(String source) { this.source = source; }

    public String getType() { return type; }
    public void setType(String type) { this.type = type; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getEncounterId() { return encounterId; }
    public void setEncounterId(String encounterId) { this.encounterId = encounterId; }

    public long getEventTime() { return eventTime; }
    public void setEventTime(long eventTime) { this.eventTime = eventTime; }

    public long getReceivedTime() { return receivedTime; }
    public void setReceivedTime(long receivedTime) { this.receivedTime = receivedTime; }

    public Map<String, Object> getPayload() { return payload; }
    public void setPayload(Map<String, Object> payload) { this.payload = payload; }

    public Map<String, String> getMetadata() { return metadata; }
    public void setMetadata(Map<String, String> metadata) { this.metadata = metadata; }

    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String correlationId) { this.correlationId = correlationId; }

    public String getVersion() { return version; }
    public void setVersion(String version) { this.version = version; }

    @Override
    public String toString() {
        return "RawEvent{" +
            "id='" + id + '\'' +
            ", type='" + type + '\'' +
            ", patientId='" + patientId + '\'' +
            ", eventTime=" + eventTime +
            '}';
    }
}