package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@JsonIgnoreProperties(ignoreUnknown = true)
@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class CDSEvent implements Serializable {
    private static final long serialVersionUID = 2L;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("patientState")
    private PatientContextState patientState;

    @JsonProperty("eventType")
    private String eventType;

    @JsonProperty("eventTime")
    private long eventTime;

    @JsonProperty("processingTime")
    private long processingTime;

    @JsonProperty("latencyMs")
    private Long latencyMs;

    @JsonProperty("dataTier")
    private String dataTier;

    @JsonProperty("phaseResults")
    private List<CDSPhaseResult> phaseResults;

    @JsonProperty("recommendations")
    private List<Map<String, Object>> recommendations;

    @JsonProperty("safetyAlerts")
    private List<Map<String, Object>> safetyAlerts;

    @JsonProperty("mhriScore")
    private MHRIScore mhriScore;

    @JsonProperty("protocolsMatched")
    private int protocolsMatched;

    @JsonProperty("broadcastStateReady")
    private boolean broadcastStateReady;

    public CDSEvent() {
        this.processingTime = System.currentTimeMillis();
        this.phaseResults = new ArrayList<>();
        this.recommendations = new ArrayList<>();
        this.safetyAlerts = new ArrayList<>();
    }

    public CDSEvent(EnrichedPatientContext context) {
        this();
        this.patientId = context.getPatientId();
        this.patientState = context.getPatientState();
        this.eventType = context.getEventType();
        this.eventTime = context.getEventTime();
        this.processingTime = System.currentTimeMillis();
        this.latencyMs = context.getLatencyMs();
        this.dataTier = context.getDataTier();
    }

    public void addPhaseResult(CDSPhaseResult result) {
        this.phaseResults.add(result);
    }

    public void addRecommendation(Map<String, Object> rec) {
        this.recommendations.add(rec);
    }

    public void addSafetyAlert(Map<String, Object> alert) {
        this.safetyAlerts.add(alert);
    }

    // Getters and setters
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public PatientContextState getPatientState() { return patientState; }
    public void setPatientState(PatientContextState patientState) { this.patientState = patientState; }
    public String getEventType() { return eventType; }
    public void setEventType(String eventType) { this.eventType = eventType; }
    public long getEventTime() { return eventTime; }
    public void setEventTime(long eventTime) { this.eventTime = eventTime; }
    public long getProcessingTime() { return processingTime; }
    public void setProcessingTime(long processingTime) { this.processingTime = processingTime; }
    public Long getLatencyMs() { return latencyMs; }
    public void setLatencyMs(Long latencyMs) { this.latencyMs = latencyMs; }
    public String getDataTier() { return dataTier; }
    public void setDataTier(String dataTier) { this.dataTier = dataTier; }
    public List<CDSPhaseResult> getPhaseResults() { return phaseResults; }
    public void setPhaseResults(List<CDSPhaseResult> phaseResults) { this.phaseResults = phaseResults; }
    public List<Map<String, Object>> getRecommendations() { return recommendations; }
    public void setRecommendations(List<Map<String, Object>> recs) { this.recommendations = recs; }
    public List<Map<String, Object>> getSafetyAlerts() { return safetyAlerts; }
    public void setSafetyAlerts(List<Map<String, Object>> alerts) { this.safetyAlerts = alerts; }
    public MHRIScore getMhriScore() { return mhriScore; }
    public void setMhriScore(MHRIScore mhriScore) { this.mhriScore = mhriScore; }
    public int getProtocolsMatched() { return protocolsMatched; }
    public void setProtocolsMatched(int protocolsMatched) { this.protocolsMatched = protocolsMatched; }
    public boolean isBroadcastStateReady() { return broadcastStateReady; }
    public void setBroadcastStateReady(boolean ready) { this.broadcastStateReady = ready; }

    @Override
    public String toString() {
        return String.format("CDSEvent{patient='%s', type='%s', phases=%d, recs=%d, safety=%d, mhri=%s}",
                patientId, eventType,
                phaseResults != null ? phaseResults.size() : 0,
                recommendations != null ? recommendations.size() : 0,
                safetyAlerts != null ? safetyAlerts.size() : 0,
                mhriScore != null ? mhriScore.getComposite() : "null");
    }
}
