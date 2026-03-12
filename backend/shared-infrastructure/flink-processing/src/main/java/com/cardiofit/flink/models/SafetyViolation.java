package com.cardiofit.flink.models;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.Map;

/**
 * SafetyViolation represents a clinical safety violation event
 */
public class SafetyViolation implements Serializable {
    private static final long serialVersionUID = 1L;

    private String violationId;
    private String patientId;
    private String violationType;
    private String ruleId;
    private String description;
    private String severity;
    private double riskScore;
    private LocalDateTime detectedAt;
    private Map<String, Object> violationDetails;
    private String source;
    private boolean requiresIntervention;
    private String recommendedAction;

    public SafetyViolation() {}

    public SafetyViolation(String violationId, String patientId, String violationType, String severity) {
        this.violationId = violationId;
        this.patientId = patientId;
        this.violationType = violationType;
        this.severity = severity;
        this.detectedAt = LocalDateTime.now();
    }

    // Getters and Setters
    public String getViolationId() { return violationId; }
    public void setViolationId(String violationId) { this.violationId = violationId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getViolationType() { return violationType; }
    public void setViolationType(String violationType) { this.violationType = violationType; }

    public String getRuleId() { return ruleId; }
    public void setRuleId(String ruleId) { this.ruleId = ruleId; }

    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }

    public String getSeverity() { return severity; }
    public void setSeverity(String severity) { this.severity = severity; }

    public double getRiskScore() { return riskScore; }
    public void setRiskScore(double riskScore) { this.riskScore = riskScore; }

    public LocalDateTime getDetectedAt() { return detectedAt; }
    public void setDetectedAt(LocalDateTime detectedAt) { this.detectedAt = detectedAt; }

    public Map<String, Object> getViolationDetails() { return violationDetails; }
    public void setViolationDetails(Map<String, Object> violationDetails) {
        this.violationDetails = violationDetails;
    }

    public String getSource() { return source; }
    public void setSource(String source) { this.source = source; }

    public boolean isRequiresIntervention() { return requiresIntervention; }
    public void setRequiresIntervention(boolean requiresIntervention) {
        this.requiresIntervention = requiresIntervention;
    }

    public String getRecommendedAction() { return recommendedAction; }
    public void setRecommendedAction(String recommendedAction) { this.recommendedAction = recommendedAction; }

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private String violationId;
        private String patientId;
        private String violationType;
        private String ruleId;
        private String description;
        private String severity;
        private double riskScore;
        private LocalDateTime detectedAt;
        private Map<String, Object> violationDetails;
        private String source;
        private boolean requiresIntervention;
        private String recommendedAction;

        public Builder violationId(String violationId) { this.violationId = violationId; return this; }
        public Builder patientId(String patientId) { this.patientId = patientId; return this; }
        public Builder violationType(String violationType) { this.violationType = violationType; return this; }
        public Builder ruleId(String ruleId) { this.ruleId = ruleId; return this; }
        public Builder description(String description) { this.description = description; return this; }
        public Builder severity(String severity) { this.severity = severity; return this; }
        public Builder riskScore(double riskScore) { this.riskScore = riskScore; return this; }
        public Builder detectedAt(LocalDateTime detectedAt) { this.detectedAt = detectedAt; return this; }
        public Builder violationDetails(Map<String, Object> violationDetails) { this.violationDetails = violationDetails; return this; }
        public Builder source(String source) { this.source = source; return this; }
        public Builder requiresIntervention(boolean requiresIntervention) { this.requiresIntervention = requiresIntervention; return this; }
        public Builder recommendedAction(String recommendedAction) { this.recommendedAction = recommendedAction; return this; }

        public SafetyViolation build() {
            SafetyViolation violation = new SafetyViolation();
            violation.violationId = this.violationId;
            violation.patientId = this.patientId;
            violation.violationType = this.violationType;
            violation.ruleId = this.ruleId;
            violation.description = this.description;
            violation.severity = this.severity;
            violation.riskScore = this.riskScore;
            violation.detectedAt = this.detectedAt;
            violation.violationDetails = this.violationDetails;
            violation.source = this.source;
            violation.requiresIntervention = this.requiresIntervention;
            violation.recommendedAction = this.recommendedAction;
            return violation;
        }
    }
}