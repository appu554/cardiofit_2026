package com.cardiofit.stream.state;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.Map;

public class ProtocolState implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private String protocolId;
    private String currentPhase;
    private String status;
    private LocalDateTime phaseStartTime;
    private LocalDateTime lastUpdate;
    private Map<String, Object> stateData;
    private double completionPercentage;
    private String nextAction;

    public ProtocolState() {}

    public ProtocolState(String patientId, String protocolId, String currentPhase) {
        this.patientId = patientId;
        this.protocolId = protocolId;
        this.currentPhase = currentPhase;
        this.phaseStartTime = LocalDateTime.now();
        this.lastUpdate = LocalDateTime.now();
        this.status = "ACTIVE";
        this.completionPercentage = 0.0;
    }

    // Constructor required by ClinicalPathwayAdherenceFunction
    public ProtocolState(String protocolId, String initialStep) {
        this.protocolId = protocolId;
        this.currentPhase = initialStep;
        this.phaseStartTime = LocalDateTime.now();
        this.lastUpdate = LocalDateTime.now();
        this.status = "ACTIVE";
        this.completionPercentage = 0.0;
    }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getProtocolId() { return protocolId; }
    public void setProtocolId(String protocolId) { this.protocolId = protocolId; }

    public String getCurrentPhase() { return currentPhase; }
    public void setCurrentPhase(String currentPhase) { this.currentPhase = currentPhase; }

    public String getStatus() { return status; }
    public void setStatus(String status) { this.status = status; }

    public LocalDateTime getPhaseStartTime() { return phaseStartTime; }
    public void setPhaseStartTime(LocalDateTime phaseStartTime) { this.phaseStartTime = phaseStartTime; }

    public LocalDateTime getLastUpdate() { return lastUpdate; }
    public void setLastUpdate(LocalDateTime lastUpdate) { this.lastUpdate = lastUpdate; }

    public Map<String, Object> getStateData() { return stateData; }
    public void setStateData(Map<String, Object> stateData) { this.stateData = stateData; }

    public double getCompletionPercentage() { return completionPercentage; }
    public void setCompletionPercentage(double completionPercentage) { this.completionPercentage = completionPercentage; }

    public String getNextAction() { return nextAction; }
    public void setNextAction(String nextAction) { this.nextAction = nextAction; }

    // Method required by ClinicalPathwayAdherenceFunction
    public String getCurrentStep() {
        return this.currentPhase;
    }

    // Additional methods required by ClinicalPathwayAdherenceFunction
    public void setCurrentStep(String currentStep) {
        this.currentPhase = currentStep;
        this.lastUpdate = LocalDateTime.now();
    }

    public void setLastTransition(long timestamp) {
        this.lastUpdate = LocalDateTime.ofInstant(
            java.time.Instant.ofEpochMilli(timestamp),
            java.time.ZoneId.systemDefault()
        );
    }

    public void incrementStepCount() {
        // Increment completion percentage (assuming max 10 steps)
        this.completionPercentage = Math.min(100.0, this.completionPercentage + 10.0);

        // Store step count in state data
        if (stateData == null) {
            stateData = new java.util.HashMap<>();
        }
        Integer currentCount = (Integer) stateData.get("step_count");
        stateData.put("step_count", currentCount != null ? currentCount + 1 : 1);
    }
}