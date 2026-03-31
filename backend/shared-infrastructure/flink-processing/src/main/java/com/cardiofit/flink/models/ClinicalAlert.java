package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.*;

/**
 * Full alert entity with lifecycle tracking.
 * Replaces ComposedAlert for Module 6 upgrade output.
 */
public class ClinicalAlert implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("alert_id") private String alertId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("encounter_id") private String encounterId;

    // Classification
    @JsonProperty("tier") private ActionTier tier;
    @JsonProperty("clinical_category") private String clinicalCategory;

    // Clinical content
    @JsonProperty("title") private String title;
    @JsonProperty("body") private String body;
    @JsonProperty("recommended_actions") private List<String> recommendedActions = new ArrayList<>();
    @JsonProperty("clinical_context") private Map<String, Object> clinicalContext = new HashMap<>();
    @JsonProperty("ml_predictions") private Map<String, Double> mlPredictions = new HashMap<>();

    // Source provenance
    @JsonProperty("source_module") private String sourceModule;
    @JsonProperty("trigger_event_id") private String triggerEventId;
    @JsonProperty("correlation_id") private String correlationId;
    @JsonProperty("contributing_sources") private List<String> contributingSources = new ArrayList<>();

    // Lifecycle
    @JsonProperty("state") private AlertState state = AlertState.ACTIVE;
    @JsonProperty("created_at") private long createdAt = System.currentTimeMillis();
    @JsonProperty("acknowledged_at") private Long acknowledgedAt;
    @JsonProperty("actioned_at") private Long actionedAt;
    @JsonProperty("resolved_at") private Long resolvedAt;
    @JsonProperty("escalated_at") private Long escalatedAt;
    @JsonProperty("acknowledged_by") private String acknowledgedBy;
    @JsonProperty("action_description") private String actionDescription;

    // SLA
    @JsonProperty("sla_deadline_ms") private long slaDeadlineMs;
    @JsonProperty("escalation_level") private int escalationLevel;
    @JsonProperty("assigned_to") private String assignedTo;

    public ClinicalAlert() {}

    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public String getEncounterId() { return encounterId; }
    public void setEncounterId(String encounterId) { this.encounterId = encounterId; }
    public ActionTier getTier() { return tier; }
    public void setTier(ActionTier tier) { this.tier = tier; }
    public String getClinicalCategory() { return clinicalCategory; }
    public void setClinicalCategory(String clinicalCategory) { this.clinicalCategory = clinicalCategory; }
    public String getTitle() { return title; }
    public void setTitle(String title) { this.title = title; }
    public String getBody() { return body; }
    public void setBody(String body) { this.body = body; }
    public List<String> getRecommendedActions() { return recommendedActions; }
    public void setRecommendedActions(List<String> recommendedActions) { this.recommendedActions = recommendedActions; }
    public Map<String, Object> getClinicalContext() { return clinicalContext; }
    public void setClinicalContext(Map<String, Object> clinicalContext) { this.clinicalContext = clinicalContext; }
    public Map<String, Double> getMlPredictions() { return mlPredictions; }
    public void setMlPredictions(Map<String, Double> mlPredictions) { this.mlPredictions = mlPredictions; }
    public String getSourceModule() { return sourceModule; }
    public void setSourceModule(String sourceModule) { this.sourceModule = sourceModule; }
    public String getTriggerEventId() { return triggerEventId; }
    public void setTriggerEventId(String triggerEventId) { this.triggerEventId = triggerEventId; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String correlationId) { this.correlationId = correlationId; }
    public List<String> getContributingSources() { return contributingSources; }
    public void setContributingSources(List<String> contributingSources) { this.contributingSources = contributingSources; }
    public AlertState getState() { return state; }
    public void setState(AlertState state) { this.state = state; }
    public long getCreatedAt() { return createdAt; }
    public void setCreatedAt(long createdAt) { this.createdAt = createdAt; }
    public Long getAcknowledgedAt() { return acknowledgedAt; }
    public void setAcknowledgedAt(Long acknowledgedAt) { this.acknowledgedAt = acknowledgedAt; }
    public Long getActionedAt() { return actionedAt; }
    public void setActionedAt(Long actionedAt) { this.actionedAt = actionedAt; }
    public Long getResolvedAt() { return resolvedAt; }
    public void setResolvedAt(Long resolvedAt) { this.resolvedAt = resolvedAt; }
    public Long getEscalatedAt() { return escalatedAt; }
    public void setEscalatedAt(Long escalatedAt) { this.escalatedAt = escalatedAt; }
    public String getAcknowledgedBy() { return acknowledgedBy; }
    public void setAcknowledgedBy(String acknowledgedBy) { this.acknowledgedBy = acknowledgedBy; }
    public String getActionDescription() { return actionDescription; }
    public void setActionDescription(String actionDescription) { this.actionDescription = actionDescription; }
    public long getSlaDeadlineMs() { return slaDeadlineMs; }
    public void setSlaDeadlineMs(long slaDeadlineMs) { this.slaDeadlineMs = slaDeadlineMs; }
    public int getEscalationLevel() { return escalationLevel; }
    public void setEscalationLevel(int escalationLevel) { this.escalationLevel = escalationLevel; }
    public String getAssignedTo() { return assignedTo; }
    public void setAssignedTo(String assignedTo) { this.assignedTo = assignedTo; }
}
