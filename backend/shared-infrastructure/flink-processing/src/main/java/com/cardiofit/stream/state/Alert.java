package com.cardiofit.stream.state;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.Map;

public class Alert implements Serializable {
    private static final long serialVersionUID = 1L;

    private String alertId;
    private String patientId;
    private String alertType;
    private String severity;
    private String message;
    private String source;
    private LocalDateTime createdAt;
    private LocalDateTime expiresAt;
    private boolean acknowledged;
    private Map<String, Object> metadata;

    public Alert() {}

    public Alert(String alertId, String patientId, String alertType, String severity, String message) {
        this.alertId = alertId;
        this.patientId = patientId;
        this.alertType = alertType;
        this.severity = severity;
        this.message = message;
        this.createdAt = LocalDateTime.now();
        this.acknowledged = false;
    }

    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getAlertType() { return alertType; }
    public void setAlertType(String alertType) { this.alertType = alertType; }

    public String getSeverity() { return severity; }
    public void setSeverity(String severity) { this.severity = severity; }

    public String getMessage() { return message; }
    public void setMessage(String message) { this.message = message; }

    public String getSource() { return source; }
    public void setSource(String source) { this.source = source; }

    public LocalDateTime getCreatedAt() { return createdAt; }
    public void setCreatedAt(LocalDateTime createdAt) { this.createdAt = createdAt; }

    public LocalDateTime getExpiresAt() { return expiresAt; }
    public void setExpiresAt(LocalDateTime expiresAt) { this.expiresAt = expiresAt; }

    public boolean isAcknowledged() { return acknowledged; }
    public void setAcknowledged(boolean acknowledged) { this.acknowledged = acknowledged; }

    public Map<String, Object> getMetadata() { return metadata; }
    public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

    // Additional methods required by ClinicalPathwayAdherenceFunction
    public void setEventId(String eventId) {
        if (metadata == null) {
            metadata = new java.util.HashMap<>();
        }
        metadata.put("event_id", eventId);
    }

    public void setProtocolId(String protocolId) {
        if (metadata == null) {
            metadata = new java.util.HashMap<>();
        }
        metadata.put("protocol_id", protocolId);
    }

    public void setRecommendations(java.util.List<String> recommendations) {
        if (metadata == null) {
            metadata = new java.util.HashMap<>();
        }
        metadata.put("recommendations", recommendations);
    }

    public void setTimestamp(long timestamp) {
        this.createdAt = java.time.LocalDateTime.ofInstant(
            java.time.Instant.ofEpochMilli(timestamp),
            java.time.ZoneId.systemDefault()
        );
    }

    public void setRiskScore(double riskScore) {
        if (metadata == null) {
            metadata = new java.util.HashMap<>();
        }
        metadata.put("risk_score", riskScore);
    }
}