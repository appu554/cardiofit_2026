package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.List;
import java.util.Map;

@JsonIgnoreProperties(ignoreUnknown = true)
public class InterventionWindowSignal implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("signal_id") private String signalId;
    @JsonProperty("intervention_id") private String interventionId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("signal_type") private InterventionWindowSignalType signalType;
    @JsonProperty("intervention_type") private InterventionType interventionType;
    @JsonProperty("intervention_detail") private Map<String, Object> interventionDetail;
    @JsonProperty("observation_start_ms") private long observationStartMs;
    @JsonProperty("observation_end_ms") private long observationEndMs;
    @JsonProperty("observation_window_days") private int observationWindowDays;
    @JsonProperty("trajectory_at_signal") private TrajectoryClassification trajectoryAtSignal;
    @JsonProperty("concurrent_intervention_ids") private List<String> concurrentInterventionIds;
    @JsonProperty("concurrent_intervention_count") private int concurrentInterventionCount;
    @JsonProperty("same_domain_concurrent") private boolean sameDomainConcurrent;
    @JsonProperty("adherence_signals_at_midpoint") private Map<String, Object> adherenceSignalsAtMidpoint;
    @JsonProperty("confounders_detected") private List<String> confoundersDetected;
    @JsonProperty("lab_changes_during_window") private List<Map<String, Object>> labChangesDuringWindow;
    @JsonProperty("external_events") private List<Map<String, Object>> externalEvents;
    @JsonProperty("data_completeness_indicators") private Map<String, Boolean> dataCompletenessIndicators;
    @JsonProperty("processing_timestamp") private long processingTimestamp;
    @JsonProperty("version") private String version = "1.0";
    @JsonProperty("originating_card_id") private String originatingCardId;

    public InterventionWindowSignal() {}

    public static Builder builder() { return new Builder(); }

    public static class Builder {
        private final InterventionWindowSignal s = new InterventionWindowSignal();
        public Builder signalId(String v) { s.signalId = v; return this; }
        public Builder interventionId(String v) { s.interventionId = v; return this; }
        public Builder patientId(String v) { s.patientId = v; return this; }
        public Builder signalType(InterventionWindowSignalType v) { s.signalType = v; return this; }
        public Builder interventionType(InterventionType v) { s.interventionType = v; return this; }
        public Builder interventionDetail(Map<String, Object> v) { s.interventionDetail = v; return this; }
        public Builder observationStartMs(long v) { s.observationStartMs = v; return this; }
        public Builder observationEndMs(long v) { s.observationEndMs = v; return this; }
        public Builder observationWindowDays(int v) { s.observationWindowDays = v; return this; }
        public Builder trajectoryAtSignal(TrajectoryClassification v) { s.trajectoryAtSignal = v; return this; }
        public Builder concurrentInterventionIds(List<String> v) { s.concurrentInterventionIds = v; return this; }
        public Builder concurrentInterventionCount(int v) { s.concurrentInterventionCount = v; return this; }
        public Builder sameDomainConcurrent(boolean v) { s.sameDomainConcurrent = v; return this; }
        public Builder adherenceSignalsAtMidpoint(Map<String, Object> v) { s.adherenceSignalsAtMidpoint = v; return this; }
        public Builder confoundersDetected(List<String> v) { s.confoundersDetected = v; return this; }
        public Builder labChangesDuringWindow(List<Map<String, Object>> v) { s.labChangesDuringWindow = v; return this; }
        public Builder externalEvents(List<Map<String, Object>> v) { s.externalEvents = v; return this; }
        public Builder dataCompletenessIndicators(Map<String, Boolean> v) { s.dataCompletenessIndicators = v; return this; }
        public Builder processingTimestamp(long v) { s.processingTimestamp = v; return this; }
        public Builder version(String v) { s.version = v; return this; }
        public Builder originatingCardId(String v) { s.originatingCardId = v; return this; }
        public InterventionWindowSignal build() { return s; }
    }

    public String getSignalId() { return signalId; }
    public String getInterventionId() { return interventionId; }
    public String getPatientId() { return patientId; }
    public InterventionWindowSignalType getSignalType() { return signalType; }
    public InterventionType getInterventionType() { return interventionType; }
    public Map<String, Object> getInterventionDetail() { return interventionDetail; }
    public long getObservationStartMs() { return observationStartMs; }
    public long getObservationEndMs() { return observationEndMs; }
    public int getObservationWindowDays() { return observationWindowDays; }
    public TrajectoryClassification getTrajectoryAtSignal() { return trajectoryAtSignal; }
    public List<String> getConcurrentInterventionIds() { return concurrentInterventionIds; }
    public int getConcurrentInterventionCount() { return concurrentInterventionCount; }
    public boolean isSameDomainConcurrent() { return sameDomainConcurrent; }
    public Map<String, Object> getAdherenceSignalsAtMidpoint() { return adherenceSignalsAtMidpoint; }
    public List<String> getConfoundersDetected() { return confoundersDetected; }
    public List<Map<String, Object>> getLabChangesDuringWindow() { return labChangesDuringWindow; }
    public List<Map<String, Object>> getExternalEvents() { return externalEvents; }
    public Map<String, Boolean> getDataCompletenessIndicators() { return dataCompletenessIndicators; }
    public long getProcessingTimestamp() { return processingTimestamp; }
    public String getVersion() { return version; }
    public String getOriginatingCardId() { return originatingCardId; }
}
