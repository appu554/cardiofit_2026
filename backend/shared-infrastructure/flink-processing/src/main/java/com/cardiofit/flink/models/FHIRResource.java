package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.Map;

/**
 * FHIR Resource model for compacted topic (Phase 2)
 * Represents FHIR R4 compliant resources for clinical persistence
 */
public class FHIRResource {

    @JsonProperty("resource_type")
    private String resourceType;

    @JsonProperty("resource_id")
    private String resourceId;

    @JsonProperty("patient_id")
    private String patientId;

    @JsonProperty("version")
    private String version;

    @JsonProperty("last_updated")
    private Long lastUpdated;

    @JsonProperty("fhir_data")
    private Map<String, Object> fhirData;

    @JsonProperty("meta")
    private Map<String, Object> meta;

    @JsonProperty("source_event_id")
    private String sourceEventId;

    @JsonProperty("organization_id")
    private String organizationId;

    // Constructors
    public FHIRResource() {}

    public FHIRResource(String resourceType, String resourceId, String patientId) {
        this.resourceType = resourceType;
        this.resourceId = resourceId;
        this.patientId = patientId;
        this.version = "1";
        this.lastUpdated = System.currentTimeMillis();
    }

    // Getters and Setters
    public String getResourceType() { return resourceType; }
    public void setResourceType(String resourceType) { this.resourceType = resourceType; }

    public String getResourceId() { return resourceId; }
    public void setResourceId(String resourceId) { this.resourceId = resourceId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getVersion() { return version; }
    public void setVersion(String version) { this.version = version; }

    public Long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(Long lastUpdated) { this.lastUpdated = lastUpdated; }

    public Map<String, Object> getFhirData() { return fhirData; }
    public void setFhirData(Map<String, Object> fhirData) { this.fhirData = fhirData; }

    public Map<String, Object> getMeta() { return meta; }
    public void setMeta(Map<String, Object> meta) { this.meta = meta; }

    public String getSourceEventId() { return sourceEventId; }
    public void setSourceEventId(String sourceEventId) { this.sourceEventId = sourceEventId; }

    public String getOrganizationId() { return organizationId; }
    public void setOrganizationId(String organizationId) { this.organizationId = organizationId; }

    /**
     * Get composite key for compacted topic
     */
    public String getCompositeKey() {
        return resourceType + "|" + resourceId;
    }

    @Override
    public String toString() {
        return "FHIRResource{" +
                "resourceType='" + resourceType + '\'' +
                ", resourceId='" + resourceId + '\'' +
                ", patientId='" + patientId + '\'' +
                ", version='" + version + '\'' +
                ", lastUpdated=" + lastUpdated +
                '}';
    }
}