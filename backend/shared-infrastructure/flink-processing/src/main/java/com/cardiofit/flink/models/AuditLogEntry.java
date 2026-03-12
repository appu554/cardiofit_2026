package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.Map;

/**
 * Audit Log Entry model for compliance and monitoring
 * Captures all processing events for regulatory compliance
 */
public class AuditLogEntry {

    @JsonProperty("id")
    private String id;

    @JsonProperty("event_id")
    private String eventId;

    @JsonProperty("patient_id")
    private String patientId;

    @JsonProperty("timestamp")
    private Long timestamp;

    @JsonProperty("event_type")
    private String eventType;

    @JsonProperty("source")
    private String source;

    @JsonProperty("user_id")
    private String userId;

    @JsonProperty("session_id")
    private String sessionId;

    @JsonProperty("action")
    private String action;

    @JsonProperty("resource_type")
    private String resourceType;

    @JsonProperty("resource_id")
    private String resourceId;

    @JsonProperty("outcome")
    private String outcome; // SUCCESS, FAILURE, WARNING

    @JsonProperty("details")
    private Map<String, Object> details;

    @JsonProperty("ip_address")
    private String ipAddress;

    @JsonProperty("user_agent")
    private String userAgent;

    @JsonProperty("organization_id")
    private String organizationId;

    @JsonProperty("department")
    private String department;

    @JsonProperty("compliance_flags")
    private Map<String, Boolean> complianceFlags;

    // Constructors
    public AuditLogEntry() {
        this.id = java.util.UUID.randomUUID().toString();
        this.timestamp = System.currentTimeMillis();
    }

    public AuditLogEntry(String eventId, String patientId, String eventType, String source) {
        this();
        this.eventId = eventId;
        this.patientId = patientId;
        this.eventType = eventType;
        this.source = source;
    }

    // Getters and Setters
    public String getId() { return id; }
    public void setId(String id) { this.id = id; }

    public String getEventId() { return eventId; }
    public void setEventId(String eventId) { this.eventId = eventId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public Long getTimestamp() { return timestamp; }
    public void setTimestamp(Long timestamp) { this.timestamp = timestamp; }

    public String getEventType() { return eventType; }
    public void setEventType(String eventType) { this.eventType = eventType; }

    public String getSource() { return source; }
    public void setSource(String source) { this.source = source; }

    public String getUserId() { return userId; }
    public void setUserId(String userId) { this.userId = userId; }

    public String getSessionId() { return sessionId; }
    public void setSessionId(String sessionId) { this.sessionId = sessionId; }

    public String getAction() { return action; }
    public void setAction(String action) { this.action = action; }

    public String getResourceType() { return resourceType; }
    public void setResourceType(String resourceType) { this.resourceType = resourceType; }

    public String getResourceId() { return resourceId; }
    public void setResourceId(String resourceId) { this.resourceId = resourceId; }

    public String getOutcome() { return outcome; }
    public void setOutcome(String outcome) { this.outcome = outcome; }

    public Map<String, Object> getDetails() { return details; }
    public void setDetails(Map<String, Object> details) { this.details = details; }

    public String getIpAddress() { return ipAddress; }
    public void setIpAddress(String ipAddress) { this.ipAddress = ipAddress; }

    public String getUserAgent() { return userAgent; }
    public void setUserAgent(String userAgent) { this.userAgent = userAgent; }

    public String getOrganizationId() { return organizationId; }
    public void setOrganizationId(String organizationId) { this.organizationId = organizationId; }

    public String getDepartment() { return department; }
    public void setDepartment(String department) { this.department = department; }

    public Map<String, Boolean> getComplianceFlags() { return complianceFlags; }
    public void setComplianceFlags(Map<String, Boolean> complianceFlags) { this.complianceFlags = complianceFlags; }

    /**
     * Add a detail entry to the details map
     */
    public void addDetail(String key, Object value) {
        if (details == null) {
            details = new java.util.HashMap<>();
        }
        details.put(key, value);
    }

    /**
     * Add a compliance flag
     */
    public void addComplianceFlag(String flag, Boolean value) {
        if (complianceFlags == null) {
            complianceFlags = new java.util.HashMap<>();
        }
        complianceFlags.put(flag, value);
    }

    /**
     * Mark as HIPAA compliant event
     */
    public void markHIPAACompliant() {
        addComplianceFlag("HIPAA_COMPLIANT", true);
        addComplianceFlag("AUDIT_REQUIRED", true);
    }

    /**
     * Mark as high-risk event requiring enhanced auditing
     */
    public void markHighRisk() {
        addComplianceFlag("HIGH_RISK", true);
        addComplianceFlag("ENHANCED_AUDIT", true);
    }

    @Override
    public String toString() {
        return "AuditLogEntry{" +
                "id='" + id + '\'' +
                ", eventId='" + eventId + '\'' +
                ", patientId='" + patientId + '\'' +
                ", eventType='" + eventType + '\'' +
                ", source='" + source + '\'' +
                ", outcome='" + outcome + '\'' +
                ", timestamp=" + timestamp +
                '}';
    }
}