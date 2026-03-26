package com.cardiofit.flink.models;

import java.time.Instant;
import java.util.List;
import java.util.Map;

public class ComorbidityAlert {
    private String alertId;
    private String patientId;
    private String ruleId;          // CID-01 through CID-17
    private String ruleName;        // e.g., "Triple Whammy AKI"
    private AlertSeverity severity;  // HALT, PAUSE, SOFT_FLAG

    public enum AlertSeverity { HALT, PAUSE, SOFT_FLAG }
    private String alertContent;    // Human-readable alert text
    private String recommendedAction;
    private Map<String, Object> triggerValues;  // Lab/vital values that triggered
    private List<String> involvedMedications;   // Drug classes in combination
    private Instant detectedAt;
    private String sourceModule;

    public ComorbidityAlert() {}

    public ComorbidityAlert(String patientId, String ruleId, String ruleName,
                            AlertSeverity severity, String alertContent,
                            String recommendedAction) {
        this.alertId = java.util.UUID.randomUUID().toString();
        this.patientId = patientId;
        this.ruleId = ruleId;
        this.ruleName = ruleName;
        this.severity = severity;
        this.alertContent = alertContent;
        this.recommendedAction = recommendedAction;
        this.detectedAt = Instant.now();
        this.sourceModule = "module-8-comorbidity-interaction";
    }

    // All getters and setters
    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public String getRuleId() { return ruleId; }
    public void setRuleId(String ruleId) { this.ruleId = ruleId; }
    public String getRuleName() { return ruleName; }
    public void setRuleName(String ruleName) { this.ruleName = ruleName; }
    public AlertSeverity getSeverity() { return severity; }
    public void setSeverity(AlertSeverity severity) { this.severity = severity; }
    public String getAlertContent() { return alertContent; }
    public void setAlertContent(String alertContent) { this.alertContent = alertContent; }
    public String getRecommendedAction() { return recommendedAction; }
    public void setRecommendedAction(String recommendedAction) { this.recommendedAction = recommendedAction; }
    public Map<String, Object> getTriggerValues() { return triggerValues; }
    public void setTriggerValues(Map<String, Object> triggerValues) { this.triggerValues = triggerValues; }
    public List<String> getInvolvedMedications() { return involvedMedications; }
    public void setInvolvedMedications(List<String> involvedMedications) { this.involvedMedications = involvedMedications; }
    public Instant getDetectedAt() { return detectedAt; }
    public void setDetectedAt(Instant detectedAt) { this.detectedAt = detectedAt; }
    public String getSourceModule() { return sourceModule; }
    public void setSourceModule(String sourceModule) { this.sourceModule = sourceModule; }
}
