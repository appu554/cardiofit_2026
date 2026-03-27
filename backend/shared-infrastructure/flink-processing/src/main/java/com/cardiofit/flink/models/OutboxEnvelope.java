package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;
import java.util.List;
import java.util.Map;

/**
 * Deserialization model for ingestion service outbox events.
 *
 * Matches the JSON format produced by the outbox SDK in the ingestion service.
 * Top-level envelope wraps the clinical event_data with routing and tracing metadata.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class OutboxEnvelope implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("id")
    private String id;

    @JsonProperty("service_name")
    private String serviceName;

    @JsonProperty("event_type")
    private String eventType;

    @JsonProperty("topic")
    private String topic;

    @JsonProperty("correlation_id")
    private String correlationId;

    @JsonProperty("priority")
    private int priority;

    @JsonProperty("medical_context")
    private String medicalContext;

    @JsonProperty("metadata")
    private Map<String, String> metadata;

    @JsonProperty("event_data")
    private IngestionEventData eventData;

    @JsonProperty("created_at")
    private String createdAt;

    // Default constructor for Jackson
    public OutboxEnvelope() {}

    // Getters and setters
    public String getId() { return id; }
    public void setId(String id) { this.id = id; }

    public String getServiceName() { return serviceName; }
    public void setServiceName(String serviceName) { this.serviceName = serviceName; }

    public String getEventType() { return eventType; }
    public void setEventType(String eventType) { this.eventType = eventType; }

    public String getTopic() { return topic; }
    public void setTopic(String topic) { this.topic = topic; }

    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String correlationId) { this.correlationId = correlationId; }

    public int getPriority() { return priority; }
    public void setPriority(int priority) { this.priority = priority; }

    public String getMedicalContext() { return medicalContext; }
    public void setMedicalContext(String medicalContext) { this.medicalContext = medicalContext; }

    public Map<String, String> getMetadata() { return metadata; }
    public void setMetadata(Map<String, String> metadata) { this.metadata = metadata; }

    public IngestionEventData getEventData() { return eventData; }
    public void setEventData(IngestionEventData eventData) { this.eventData = eventData; }

    public String getCreatedAt() { return createdAt; }
    public void setCreatedAt(String createdAt) { this.createdAt = createdAt; }

    @Override
    public String toString() {
        return "OutboxEnvelope{" +
            "id='" + id + '\'' +
            ", serviceName='" + serviceName + '\'' +
            ", eventType='" + eventType + '\'' +
            ", topic='" + topic + '\'' +
            ", correlationId='" + correlationId + '\'' +
            '}';
    }

    /**
     * Inner class representing the clinical observation data within the outbox envelope.
     */
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class IngestionEventData implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("event_id")
        private String eventId;

        @JsonProperty("patient_id")
        private String patientId;

        @JsonProperty("tenant_id")
        private String tenantId;

        @JsonProperty("observation_type")
        private String observationType;

        @JsonProperty("loinc_code")
        private String loincCode;

        @JsonProperty("value")
        private Double value;

        @JsonProperty("unit")
        private String unit;

        @JsonProperty("timestamp")
        private String timestamp;

        @JsonProperty("source_type")
        private String sourceType;

        @JsonProperty("source_id")
        private String sourceId;

        @JsonProperty("quality_score")
        private Double qualityScore;

        @JsonProperty("flags")
        private List<String> flags;

        @JsonProperty("fhir_resource_id")
        private String fhirResourceId;

        // Default constructor for Jackson
        public IngestionEventData() {}

        // Getters and setters
        public String getEventId() { return eventId; }
        public void setEventId(String eventId) { this.eventId = eventId; }

        public String getPatientId() { return patientId; }
        public void setPatientId(String patientId) { this.patientId = patientId; }

        public String getTenantId() { return tenantId; }
        public void setTenantId(String tenantId) { this.tenantId = tenantId; }

        public String getObservationType() { return observationType; }
        public void setObservationType(String observationType) { this.observationType = observationType; }

        public String getLoincCode() { return loincCode; }
        public void setLoincCode(String loincCode) { this.loincCode = loincCode; }

        public Double getValue() { return value; }
        public void setValue(Double value) { this.value = value; }

        public String getUnit() { return unit; }
        public void setUnit(String unit) { this.unit = unit; }

        public String getTimestamp() { return timestamp; }
        public void setTimestamp(String timestamp) { this.timestamp = timestamp; }

        public String getSourceType() { return sourceType; }
        public void setSourceType(String sourceType) { this.sourceType = sourceType; }

        public String getSourceId() { return sourceId; }
        public void setSourceId(String sourceId) { this.sourceId = sourceId; }

        public Double getQualityScore() { return qualityScore; }
        public void setQualityScore(Double qualityScore) { this.qualityScore = qualityScore; }

        public List<String> getFlags() { return flags; }
        public void setFlags(List<String> flags) { this.flags = flags; }

        public String getFhirResourceId() { return fhirResourceId; }
        public void setFhirResourceId(String fhirResourceId) { this.fhirResourceId = fhirResourceId; }

        @Override
        public String toString() {
            return "IngestionEventData{" +
                "eventId='" + eventId + '\'' +
                ", patientId='" + patientId + '\'' +
                ", observationType='" + observationType + '\'' +
                ", loincCode='" + loincCode + '\'' +
                ", value=" + value +
                ", unit='" + unit + '\'' +
                '}';
        }
    }
}
