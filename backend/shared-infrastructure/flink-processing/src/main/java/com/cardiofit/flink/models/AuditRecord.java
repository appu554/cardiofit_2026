package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * Audit trail record. Every clinical decision, alert, notification, and action
 * must be auditable. Healthcare regulations require 7-year retention.
 * Emitted to prod.ehr.audit.logs (retention: 2555 days / 7 years).
 */
public class AuditRecord implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("audit_id") private String auditId;
    @JsonProperty("timestamp") private long timestamp = System.currentTimeMillis();
    @JsonProperty("event_type") private String eventType;
    @JsonProperty("event_description") private String eventDescription;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("source_module") private String sourceModule;
    @JsonProperty("tier") private ActionTier tier;
    @JsonProperty("clinical_category") private String clinicalCategory;
    @JsonProperty("clinical_data") private Map<String, Object> clinicalData = new HashMap<>();
    @JsonProperty("correlation_id") private String correlationId;
    @JsonProperty("alert_id") private String alertId;
    @JsonProperty("model_version") private String modelVersion;
    @JsonProperty("input_snapshot") private Map<String, Object> inputSnapshot = new HashMap<>();

    public AuditRecord() {}

    public static AuditRecord alertCreated(ClinicalAlert alert, ClinicalEvent sourceEvent) {
        AuditRecord r = new AuditRecord();
        r.auditId = java.util.UUID.randomUUID().toString();
        r.eventType = "ALERT_CREATED";
        r.eventDescription = "Clinical alert created: " + alert.getTier() + " " + alert.getClinicalCategory();
        r.patientId = alert.getPatientId();
        r.sourceModule = alert.getSourceModule();
        r.tier = alert.getTier();
        r.clinicalCategory = alert.getClinicalCategory();
        r.alertId = alert.getAlertId();
        r.correlationId = alert.getCorrelationId();
        return r;
    }

    public String getAuditId() { return auditId; }
    public void setAuditId(String auditId) { this.auditId = auditId; }
    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }
    public String getEventType() { return eventType; }
    public void setEventType(String eventType) { this.eventType = eventType; }
    public String getEventDescription() { return eventDescription; }
    public void setEventDescription(String eventDescription) { this.eventDescription = eventDescription; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public String getSourceModule() { return sourceModule; }
    public void setSourceModule(String sourceModule) { this.sourceModule = sourceModule; }
    public ActionTier getTier() { return tier; }
    public void setTier(ActionTier tier) { this.tier = tier; }
    public String getClinicalCategory() { return clinicalCategory; }
    public void setClinicalCategory(String clinicalCategory) { this.clinicalCategory = clinicalCategory; }
    public Map<String, Object> getClinicalData() { return clinicalData; }
    public void setClinicalData(Map<String, Object> clinicalData) { this.clinicalData = clinicalData; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String correlationId) { this.correlationId = correlationId; }
    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }
    public String getModelVersion() { return modelVersion; }
    public void setModelVersion(String modelVersion) { this.modelVersion = modelVersion; }
    public Map<String, Object> getInputSnapshot() { return inputSnapshot; }
    public void setInputSnapshot(Map<String, Object> inputSnapshot) { this.inputSnapshot = inputSnapshot; }
}
