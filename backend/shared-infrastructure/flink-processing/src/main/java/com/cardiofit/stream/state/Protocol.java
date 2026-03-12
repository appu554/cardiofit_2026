package com.cardiofit.stream.state;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.Map;
import java.util.List;

public class Protocol implements Serializable {
    private static final long serialVersionUID = 1L;

    private String protocolId;
    private String protocolName;
    private String version;
    private String description;
    private List<String> phases;
    private Map<String, Object> parameters;
    private LocalDateTime createdAt;
    private boolean active;

    public Protocol() {}

    public Protocol(String protocolId, String protocolName, String version) {
        this.protocolId = protocolId;
        this.protocolName = protocolName;
        this.version = version;
        this.createdAt = LocalDateTime.now();
        this.active = true;
    }

    public String getProtocolId() { return protocolId; }
    public void setProtocolId(String protocolId) { this.protocolId = protocolId; }

    public String getProtocolName() { return protocolName; }
    public void setProtocolName(String protocolName) { this.protocolName = protocolName; }

    public String getVersion() { return version; }
    public void setVersion(String version) { this.version = version; }

    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }

    public List<String> getPhases() { return phases; }
    public void setPhases(List<String> phases) { this.phases = phases; }

    public Map<String, Object> getParameters() { return parameters; }
    public void setParameters(Map<String, Object> parameters) { this.parameters = parameters; }

    public LocalDateTime getCreatedAt() { return createdAt; }
    public void setCreatedAt(LocalDateTime createdAt) { this.createdAt = createdAt; }

    public boolean isActive() { return active; }
    public void setActive(boolean active) { this.active = active; }

    // Methods required by ClinicalPathwayAdherenceFunction
    public String getId() {
        return this.protocolId;
    }

    public String getInitialStep() {
        // Return first phase as initial step, or default if no phases defined
        if (phases != null && !phases.isEmpty()) {
            return phases.get(0);
        }
        return "initial_assessment";
    }

    public List<String> getExpectedActions(String step) {
        // Return expected actions for a given step/phase
        // In production, this would be based on clinical protocols
        List<String> expectedActions = new java.util.ArrayList<>();

        if (step != null) {
            switch (step.toLowerCase()) {
                case "initial_assessment":
                case "assessment":
                    expectedActions.add("vital_signs_check");
                    expectedActions.add("patient_history_review");
                    break;
                case "treatment":
                    expectedActions.add("medication_administration");
                    expectedActions.add("intervention_planning");
                    break;
                case "monitoring":
                    expectedActions.add("continuous_monitoring");
                    expectedActions.add("progress_evaluation");
                    break;
                default:
                    expectedActions.add("standard_care_protocol");
            }
        }

        return expectedActions;
    }

    // Additional methods required by ClinicalPathwayAdherenceFunction
    public boolean isCriticalPathway() {
        // Determine if this is a critical pathway based on protocol ID or name
        return protocolId != null &&
               (protocolId.toLowerCase().contains("critical") ||
                protocolId.toLowerCase().contains("emergency") ||
                protocolId.toLowerCase().contains("urgent"));
    }

    public java.util.List<String> getRecommendations(String currentStep) {
        java.util.List<String> recommendations = new java.util.ArrayList<>();

        if (currentStep != null) {
            switch (currentStep.toLowerCase()) {
                case "initial_assessment":
                case "assessment":
                    recommendations.add("Complete comprehensive assessment within 30 minutes");
                    recommendations.add("Document all vital signs and symptoms");
                    break;
                case "treatment":
                    recommendations.add("Follow evidence-based treatment protocols");
                    recommendations.add("Monitor patient response to interventions");
                    break;
                case "monitoring":
                    recommendations.add("Continue monitoring per protocol guidelines");
                    recommendations.add("Alert physician for any concerning changes");
                    break;
                default:
                    recommendations.add("Follow standard protocol guidelines");
            }
        }

        return recommendations;
    }

    public String getName() {
        return this.protocolName != null ? this.protocolName : this.protocolId;
    }

    public String getCurrentStep() {
        // Return the current step/phase - in a real implementation this would track
        // the current position in the protocol workflow
        if (phases != null && !phases.isEmpty()) {
            return phases.get(0); // Return first phase as default current step
        }
        return "initial_assessment";
    }

    public String getNextStep(String currentStep, String eventType) {
        if (phases == null || phases.isEmpty()) {
            return "monitoring"; // Default next step
        }

        // Find current step index and return next step
        for (int i = 0; i < phases.size(); i++) {
            if (phases.get(i).equals(currentStep)) {
                if (i + 1 < phases.size()) {
                    return phases.get(i + 1);
                } else {
                    return "completed"; // End of protocol
                }
            }
        }

        // If current step not found, return first phase
        return phases.get(0);
    }
}