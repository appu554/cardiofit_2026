package com.cardiofit.stream.models;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.Map;

public class ClinicalInsight implements Serializable {
    private static final long serialVersionUID = 1L;

    private String insightId;
    private String patientId;
    private String insightType;
    private String description;
    private String severity;
    private double confidence;
    private LocalDateTime timestamp;
    private Map<String, Object> metadata;

    public ClinicalInsight() {}

    public ClinicalInsight(String insightId, String patientId, String insightType) {
        this.insightId = insightId;
        this.patientId = patientId;
        this.insightType = insightType;
        this.timestamp = LocalDateTime.now();
    }

    // Constructor required by ClinicalPathwayAdherenceFunction
    public ClinicalInsight(String patientId, String eventId, String protocolId, String currentStep, java.util.List<String> recommendations, long timestamp) {
        this.insightId = java.util.UUID.randomUUID().toString();
        this.patientId = patientId;
        this.insightType = "PATHWAY_ADHERENCE";
        this.description = "Protocol: " + protocolId + ", Step: " + currentStep;
        this.confidence = 0.8; // Default confidence
        this.timestamp = java.time.LocalDateTime.ofInstant(java.time.Instant.ofEpochMilli(timestamp), java.time.ZoneId.systemDefault());

        // Store additional data in metadata
        if (this.metadata == null) {
            this.metadata = new java.util.HashMap<>();
        }
        this.metadata.put("eventId", eventId);
        this.metadata.put("protocolId", protocolId);
        this.metadata.put("currentStep", currentStep);
        this.metadata.put("recommendations", recommendations);
    }

    // Getters and Setters
    public String getInsightId() { return insightId; }
    public void setInsightId(String insightId) { this.insightId = insightId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getInsightType() { return insightType; }
    public void setInsightType(String insightType) { this.insightType = insightType; }

    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }

    public String getSeverity() { return severity; }
    public void setSeverity(String severity) { this.severity = severity; }

    public double getConfidence() { return confidence; }
    public void setConfidence(double confidence) { this.confidence = confidence; }

    public LocalDateTime getTimestamp() { return timestamp; }
    public void setTimestamp(LocalDateTime timestamp) { this.timestamp = timestamp; }

    public Map<String, Object> getMetadata() { return metadata; }
    public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }
}