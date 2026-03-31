package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Structured FHIR write-back request emitted to fhir-writeback topic.
 * A separate FHIR Writer Service handles delivery — no HTTP calls inside Flink.
 */
public class FhirWriteRequest implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum ResourceType { OBSERVATION, RISK_ASSESSMENT, DETECTED_ISSUE, CLINICAL_IMPRESSION, FLAG, COMMUNICATION_REQUEST }
    public enum WritePriority { CRITICAL, NORMAL, LOW }

    @JsonProperty("request_id") private String requestId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("resource_type") private ResourceType resourceType;
    @JsonProperty("fhir_resource_json") private String fhirResourceJson;
    @JsonProperty("priority") private WritePriority priority;
    @JsonProperty("created_at") private long createdAt = System.currentTimeMillis();
    @JsonProperty("max_retries") private int maxRetries = 3;

    public FhirWriteRequest() {}

    public String getRequestId() { return requestId; }
    public void setRequestId(String requestId) { this.requestId = requestId; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public ResourceType getResourceType() { return resourceType; }
    public void setResourceType(ResourceType resourceType) { this.resourceType = resourceType; }
    public String getFhirResourceJson() { return fhirResourceJson; }
    public void setFhirResourceJson(String fhirResourceJson) { this.fhirResourceJson = fhirResourceJson; }
    public WritePriority getPriority() { return priority; }
    public void setPriority(WritePriority priority) { this.priority = priority; }
    public long getCreatedAt() { return createdAt; }
    public void setCreatedAt(long createdAt) { this.createdAt = createdAt; }
    public int getMaxRetries() { return maxRetries; }
    public void setMaxRetries(int maxRetries) { this.maxRetries = maxRetries; }
}
