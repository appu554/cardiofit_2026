package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;
import java.util.Map;

/**
 * Critical Alert model for Phase 2 action topic
 * Represents urgent clinical alerts requiring immediate attention
 */
public class CriticalAlert {

    @JsonProperty("id")
    private String id;

    @JsonProperty("patient_id")
    private String patientId;

    @JsonProperty("alert_type")
    private String alertType;

    @JsonProperty("severity")
    private String severity;

    @JsonProperty("message")
    private String message;

    @JsonProperty("timestamp")
    private Long timestamp;

    @JsonProperty("source_event_id")
    private String sourceEventId;

    @JsonProperty("requires_immediate_action")
    private Boolean requiresImmedateAction;

    @JsonProperty("drug_interactions")
    private List<DrugInteraction> drugInteractions;

    @JsonProperty("clinical_context")
    private Map<String, Object> clinicalContext;

    @JsonProperty("recommended_actions")
    private List<String> recommendedActions;

    @JsonProperty("priority_score")
    private Double priorityScore;

    // Constructors
    public CriticalAlert() {}

    public CriticalAlert(String id, String patientId, String alertType, String severity, String message) {
        this.id = id;
        this.patientId = patientId;
        this.alertType = alertType;
        this.severity = severity;
        this.message = message;
        this.timestamp = System.currentTimeMillis();
    }

    // Getters and Setters
    public String getId() { return id; }
    public void setId(String id) { this.id = id; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getAlertType() { return alertType; }
    public void setAlertType(String alertType) { this.alertType = alertType; }

    public String getSeverity() { return severity; }
    public void setSeverity(String severity) { this.severity = severity; }

    public String getMessage() { return message; }
    public void setMessage(String message) { this.message = message; }

    public Long getTimestamp() { return timestamp; }
    public void setTimestamp(Long timestamp) { this.timestamp = timestamp; }

    public String getSourceEventId() { return sourceEventId; }
    public void setSourceEventId(String sourceEventId) { this.sourceEventId = sourceEventId; }

    public Boolean getRequiresImmedateAction() { return requiresImmedateAction; }
    public void setRequiresImmedateAction(Boolean requiresImmedateAction) { this.requiresImmedateAction = requiresImmedateAction; }

    public List<DrugInteraction> getDrugInteractions() { return drugInteractions; }
    public void setDrugInteractions(List<DrugInteraction> drugInteractions) { this.drugInteractions = drugInteractions; }

    public Map<String, Object> getClinicalContext() { return clinicalContext; }
    public void setClinicalContext(Map<String, Object> clinicalContext) { this.clinicalContext = clinicalContext; }

    public List<String> getRecommendedActions() { return recommendedActions; }
    public void setRecommendedActions(List<String> recommendedActions) { this.recommendedActions = recommendedActions; }

    public Double getPriorityScore() { return priorityScore; }
    public void setPriorityScore(Double priorityScore) { this.priorityScore = priorityScore; }

    @Override
    public String toString() {
        return "CriticalAlert{" +
                "id='" + id + '\'' +
                ", patientId='" + patientId + '\'' +
                ", alertType='" + alertType + '\'' +
                ", severity='" + severity + '\'' +
                ", message='" + message + '\'' +
                ", timestamp=" + timestamp +
                '}';
    }
}