package com.cardiofit.flink.models;

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
    private String recommendedAction;
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

    public String getRecommendedAction() { return recommendedAction; }
    public void setRecommendedAction(String recommendedAction) { this.recommendedAction = recommendedAction; }

    public boolean isRequiresIntervention() { return requiresIntervention; }
    public void setRequiresIntervention(boolean requiresIntervention) {
        this.requiresIntervention = requiresIntervention;
    }

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private String interactionId;
        private String patientId;
        private List<String> medicationIds;
        private String interactionType;
        private String severity;
        private String description;
        private double riskScore;
        private LocalDateTime detectedAt;
        private String recommendedAction;
        private boolean requiresIntervention;

        public Builder interactionId(String interactionId) { this.interactionId = interactionId; return this; }
        public Builder patientId(String patientId) { this.patientId = patientId; return this; }
        public Builder medicationIds(List<String> medicationIds) { this.medicationIds = medicationIds; return this; }
        public Builder interactionType(String interactionType) { this.interactionType = interactionType; return this; }
        public Builder severity(String severity) { this.severity = severity; return this; }
        public Builder description(String description) { this.description = description; return this; }
        public Builder riskScore(double riskScore) { this.riskScore = riskScore; return this; }
        public Builder detectedAt(LocalDateTime detectedAt) { this.detectedAt = detectedAt; return this; }
        public Builder recommendedAction(String recommendedAction) { this.recommendedAction = recommendedAction; return this; }
        public Builder requiresIntervention(boolean requiresIntervention) { this.requiresIntervention = requiresIntervention; return this; }

        public DrugInteraction build() {
            DrugInteraction interaction = new DrugInteraction();
            interaction.interactionId = this.interactionId;
            interaction.patientId = this.patientId;
            interaction.medicationIds = this.medicationIds;
            interaction.interactionType = this.interactionType;
            interaction.severity = this.severity;
            interaction.description = this.description;
            interaction.riskScore = this.riskScore;
            interaction.detectedAt = this.detectedAt;
            interaction.recommendedAction = this.recommendedAction;
            interaction.requiresIntervention = this.requiresIntervention;
            return interaction;
        }
    }
}