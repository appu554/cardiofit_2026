package com.cardiofit.stream.state;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.Map;
import java.util.List;

public class PatientContext implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private LocalDateTime lastUpdate;
    private Map<String, Object> clinicalState;
    private List<String> activeProtocols;
    private Map<String, Double> riskScores;
    private String currentPhase;

    public PatientContext() {}

    public PatientContext(String patientId) {
        this.patientId = patientId;
        this.lastUpdate = LocalDateTime.now();
    }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public LocalDateTime getLastUpdate() { return lastUpdate; }
    public void setLastUpdate(LocalDateTime lastUpdate) { this.lastUpdate = lastUpdate; }

    public Map<String, Object> getClinicalState() { return clinicalState; }
    public void setClinicalState(Map<String, Object> clinicalState) { this.clinicalState = clinicalState; }

    public List<String> getActiveProtocols() { return activeProtocols; }
    public void setActiveProtocols(List<String> activeProtocols) { this.activeProtocols = activeProtocols; }

    public Map<String, Double> getRiskScores() { return riskScores; }
    public void setRiskScores(Map<String, Double> riskScores) { this.riskScores = riskScores; }

    public String getCurrentPhase() { return currentPhase; }
    public void setCurrentPhase(String currentPhase) { this.currentPhase = currentPhase; }

    // Methods required by ClinicalPathwayAdherenceFunction
    public void addActiveMedication(String medication) {
        if (clinicalState == null) {
            clinicalState = new java.util.HashMap<>();
        }

        @SuppressWarnings("unchecked")
        java.util.List<String> activeMeds = (java.util.List<String>) clinicalState.get("active_medications");
        if (activeMeds == null) {
            activeMeds = new java.util.ArrayList<>();
            clinicalState.put("active_medications", activeMeds);
        }
        if (!activeMeds.contains(medication)) {
            activeMeds.add(medication);
        }
    }

    public void removeActiveMedication(String medication) {
        if (clinicalState != null) {
            @SuppressWarnings("unchecked")
            java.util.List<String> activeMeds = (java.util.List<String>) clinicalState.get("active_medications");
            if (activeMeds != null) {
                activeMeds.remove(medication);
            }
        }
    }

    public void updateLabValue(String labType, Object labValue) {
        if (clinicalState == null) {
            clinicalState = new java.util.HashMap<>();
        }

        @SuppressWarnings("unchecked")
        Map<String, Object> labResults = (Map<String, Object>) clinicalState.get("lab_results");
        if (labResults == null) {
            labResults = new java.util.HashMap<>();
            clinicalState.put("lab_results", labResults);
        }
        labResults.put(labType, labValue);
    }

    public void addCondition(String diagnosis) {
        if (clinicalState == null) {
            clinicalState = new java.util.HashMap<>();
        }

        @SuppressWarnings("unchecked")
        java.util.List<String> conditions = (java.util.List<String>) clinicalState.get("conditions");
        if (conditions == null) {
            conditions = new java.util.ArrayList<>();
            clinicalState.put("conditions", conditions);
        }
        if (!conditions.contains(diagnosis)) {
            conditions.add(diagnosis);
        }
    }

    public void updateVitals(Map<String, Object> vitals) {
        if (clinicalState == null) {
            clinicalState = new java.util.HashMap<>();
        }
        clinicalState.put("vitals", vitals);
    }

    public void setCurrentEncounterId(String encounterId) {
        if (clinicalState == null) {
            clinicalState = new java.util.HashMap<>();
        }
        clinicalState.put("current_encounter_id", encounterId);
    }

    public void setLastUpdated(long timestamp) {
        this.lastUpdate = java.time.LocalDateTime.ofInstant(
            java.time.Instant.ofEpochMilli(timestamp),
            java.time.ZoneId.systemDefault()
        );
    }

    // Additional methods required by ClinicalPathwayAdherenceFunction
    @SuppressWarnings("unchecked")
    public java.util.List<String> getActiveMedications() {
        if (clinicalState == null) return new java.util.ArrayList<>();
        java.util.List<String> activeMeds = (java.util.List<String>) clinicalState.get("active_medications");
        return activeMeds != null ? activeMeds : new java.util.ArrayList<>();
    }

    @SuppressWarnings("unchecked")
    public java.util.List<String> getConditions() {
        if (clinicalState == null) return new java.util.ArrayList<>();
        java.util.List<String> conditions = (java.util.List<String>) clinicalState.get("conditions");
        return conditions != null ? conditions : new java.util.ArrayList<>();
    }

    public boolean hasHighRiskFactors() {
        if (clinicalState == null) return false;
        Object riskFactors = clinicalState.get("high_risk_factors");
        return riskFactors != null && Boolean.parseBoolean(riskFactors.toString());
    }

    public boolean requiresEnhancedMonitoring() {
        if (clinicalState == null) return false;
        Object monitoring = clinicalState.get("enhanced_monitoring");
        return monitoring != null && Boolean.parseBoolean(monitoring.toString());
    }

    public boolean isDataComplete() {
        if (clinicalState == null) return false;
        Object complete = clinicalState.get("data_complete");
        return complete != null && Boolean.parseBoolean(complete.toString());
    }

    public double calculateRiskScore() {
        if (clinicalState == null) return 0.0;
        Object riskScore = clinicalState.get("risk_score");
        if (riskScore instanceof Number) {
            return ((Number) riskScore).doubleValue();
        }
        return 0.0;
    }
}