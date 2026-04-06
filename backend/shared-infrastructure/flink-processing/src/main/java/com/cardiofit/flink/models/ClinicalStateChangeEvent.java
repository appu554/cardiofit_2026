package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;
import java.util.*;

@JsonIgnoreProperties(ignoreUnknown = true)
public class ClinicalStateChangeEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("change_id") private String changeId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("change_type") private ClinicalStateChangeType changeType;
    @JsonProperty("priority") private String priority;
    @JsonProperty("previous_value") private String previousValue;
    @JsonProperty("current_value") private String currentValue;
    @JsonProperty("domain") private CKMRiskDomain domain;
    @JsonProperty("trigger_module") private String triggerModule;
    @JsonProperty("ckm_velocity_at_change") private CKMRiskVelocity ckmVelocityAtChange;
    @JsonProperty("recommended_action") private String recommendedAction;
    @JsonProperty("originating_signals") private List<String> originatingSignals;
    @JsonProperty("data_completeness_at_change") private double dataCompletenessAtChange;
    @JsonProperty("confidence_score") private double confidenceScore;
    @JsonProperty("metadata") private Map<String, Object> metadata;
    @JsonProperty("processing_timestamp") private long processingTimestamp;
    @JsonProperty("version") private String version = "1.0";

    public ClinicalStateChangeEvent() {
        this.originatingSignals = new ArrayList<>();
        this.metadata = new HashMap<>();
    }

    public static Builder builder() { return new Builder(); }

    // Getters
    public String getChangeId() { return changeId; }
    public String getPatientId() { return patientId; }
    public ClinicalStateChangeType getChangeType() { return changeType; }
    public String getPriority() { return priority; }
    public String getPreviousValue() { return previousValue; }
    public String getCurrentValue() { return currentValue; }
    public CKMRiskDomain getDomain() { return domain; }
    public String getTriggerModule() { return triggerModule; }
    public CKMRiskVelocity getCkmVelocityAtChange() { return ckmVelocityAtChange; }
    public String getRecommendedAction() { return recommendedAction; }
    public List<String> getOriginatingSignals() { return originatingSignals; }
    public double getDataCompletenessAtChange() { return dataCompletenessAtChange; }
    public double getConfidenceScore() { return confidenceScore; }
    public Map<String, Object> getMetadata() { return metadata; }
    public long getProcessingTimestamp() { return processingTimestamp; }
    public String getVersion() { return version; }

    public static class Builder {
        private final ClinicalStateChangeEvent e = new ClinicalStateChangeEvent();

        public Builder changeId(String id) { e.changeId = id; return this; }
        public Builder patientId(String id) { e.patientId = id; return this; }
        public Builder changeType(ClinicalStateChangeType t) {
            e.changeType = t;
            e.priority = t.getPriority();
            e.recommendedAction = t.getRecommendedAction();
            return this;
        }
        public Builder previousValue(String v) { e.previousValue = v; return this; }
        public Builder currentValue(String v) { e.currentValue = v; return this; }
        public Builder domain(CKMRiskDomain d) { e.domain = d; return this; }
        public Builder triggerModule(String m) { e.triggerModule = m; return this; }
        public Builder ckmVelocityAtChange(CKMRiskVelocity v) { e.ckmVelocityAtChange = v; return this; }
        public Builder originatingSignals(List<String> s) { e.originatingSignals = s; return this; }
        public Builder dataCompletenessAtChange(double d) { e.dataCompletenessAtChange = d; return this; }
        public Builder confidenceScore(double c) { e.confidenceScore = c; return this; }
        public Builder metadata(Map<String, Object> m) { e.metadata = m; return this; }
        public Builder processingTimestamp(long t) { e.processingTimestamp = t; return this; }
        public ClinicalStateChangeEvent build() { return e; }
    }
}
