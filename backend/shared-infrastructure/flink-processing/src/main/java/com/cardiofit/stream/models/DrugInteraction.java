package com.cardiofit.stream.models;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.List;

/**
 * DrugInteraction represents a detected drug interaction event
 */
public class DrugInteraction implements Serializable {
    private static final long serialVersionUID = 1L;

    private String interactionId;
    private String patientId;
    private List<String> medicationIds;
    private String interactionType;
    private String severity;
    private String description;
    private double riskScore;
    private LocalDateTime detectedAt;
    private String source;
    private boolean requiresIntervention;

    public DrugInteraction() {}

    public DrugInteraction(String interactionId, String patientId, List<String> medicationIds,
                          String interactionType, String severity) {
        this.interactionId = interactionId;
        this.patientId = patientId;
        this.medicationIds = medicationIds;
        this.interactionType = interactionType;
        this.severity = severity;
        this.detectedAt = LocalDateTime.now();
    }

    // Getters and Setters
    public String getInteractionId() { return interactionId; }
    public void setInteractionId(String interactionId) { this.interactionId = interactionId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public List<String> getMedicationIds() { return medicationIds; }
    public void setMedicationIds(List<String> medicationIds) { this.medicationIds = medicationIds; }

    public String getInteractionType() { return interactionType; }
    public void setInteractionType(String interactionType) { this.interactionType = interactionType; }

    public String getSeverity() { return severity; }
    public void setSeverity(String severity) { this.severity = severity; }

    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }

    public double getRiskScore() { return riskScore; }
    public void setRiskScore(double riskScore) { this.riskScore = riskScore; }

    public LocalDateTime getDetectedAt() { return detectedAt; }
    public void setDetectedAt(LocalDateTime detectedAt) { this.detectedAt = detectedAt; }

    public String getSource() { return source; }
    public void setSource(String source) { this.source = source; }

    public boolean isRequiresIntervention() { return requiresIntervention; }
    public void setRequiresIntervention(boolean requiresIntervention) {
        this.requiresIntervention = requiresIntervention;
    }
}