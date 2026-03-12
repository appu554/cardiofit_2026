package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Clinical Action - Individual actionable recommendation
 *
 * Detailed clinical action with medication specifics, dosing calculations,
 * diagnostic details, and evidence-based rationale. Part of a clinical
 * recommendation bundle.
 *
 * @author CardioFit Platform - Module 3
 * @version 1.0
 * @since 2025-10-20
 */
public class ClinicalAction implements Serializable {
    private static final long serialVersionUID = 1L;

    // Action Classification
    @JsonProperty("action_id")
    private String actionId;

    @JsonProperty("action_type")
    private ActionType actionType;

    @JsonProperty("description")
    private String description;

    @JsonProperty("sequence_order")
    private int sequenceOrder;

    // Timing
    @JsonProperty("urgency")
    private String urgency;  // STAT, URGENT, ROUTINE

    @JsonProperty("timeframe")
    private String timeframe;  // "within 1 hour", "within 4 hours", "within 24 hours"

    @JsonProperty("timeframe_rationale")
    private String timeframeRationale;

    // Action-Specific Details
    @JsonProperty("medication_details")
    private MedicationDetails medicationDetails;

    @JsonProperty("diagnostic_details")
    private DiagnosticDetails diagnosticDetails;

    // Rationale & Evidence
    @JsonProperty("clinical_rationale")
    private String clinicalRationale;

    @JsonProperty("evidence_references")
    private List<EvidenceReference> evidenceReferences;

    @JsonProperty("evidence_strength")
    private String evidenceStrength;  // STRONG, MODERATE, WEAK, EXPERT_CONSENSUS

    // Prerequisites
    @JsonProperty("prerequisite_checks")
    private List<String> prerequisiteChecks;

    @JsonProperty("required_lab_values")
    private List<String> requiredLabValues;

    // Monitoring
    @JsonProperty("expected_outcome")
    private String expectedOutcome;

    @JsonProperty("monitoring_parameters")
    private String monitoringParameters;

    // Default constructor
    public ClinicalAction() {
        this.actionId = java.util.UUID.randomUUID().toString();
        this.evidenceReferences = new ArrayList<>();
        this.prerequisiteChecks = new ArrayList<>();
        this.requiredLabValues = new ArrayList<>();
    }

    // Constructor with type and description
    public ClinicalAction(ActionType actionType, String description) {
        this();
        this.actionType = actionType;
        this.description = description;
    }

    // Getters and Setters
    public String getActionId() { return actionId; }
    public void setActionId(String actionId) { this.actionId = actionId; }

    public ActionType getActionType() { return actionType; }
    public void setActionType(ActionType actionType) { this.actionType = actionType; }

    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }

    public int getSequenceOrder() { return sequenceOrder; }
    public void setSequenceOrder(int sequenceOrder) { this.sequenceOrder = sequenceOrder; }

    public String getUrgency() { return urgency; }
    public void setUrgency(String urgency) { this.urgency = urgency; }

    public String getTimeframe() { return timeframe; }
    public void setTimeframe(String timeframe) { this.timeframe = timeframe; }

    public String getTimeframeRationale() { return timeframeRationale; }
    public void setTimeframeRationale(String timeframeRationale) { this.timeframeRationale = timeframeRationale; }

    public MedicationDetails getMedicationDetails() { return medicationDetails; }
    public void setMedicationDetails(MedicationDetails medicationDetails) { this.medicationDetails = medicationDetails; }

    public DiagnosticDetails getDiagnosticDetails() { return diagnosticDetails; }
    public void setDiagnosticDetails(DiagnosticDetails diagnosticDetails) { this.diagnosticDetails = diagnosticDetails; }

    public String getClinicalRationale() { return clinicalRationale; }
    public void setClinicalRationale(String clinicalRationale) { this.clinicalRationale = clinicalRationale; }

    public List<EvidenceReference> getEvidenceReferences() { return evidenceReferences; }
    public void setEvidenceReferences(List<EvidenceReference> evidenceReferences) {
        this.evidenceReferences = evidenceReferences;
    }

    public String getEvidenceStrength() { return evidenceStrength; }
    public void setEvidenceStrength(String evidenceStrength) { this.evidenceStrength = evidenceStrength; }

    public List<String> getPrerequisiteChecks() { return prerequisiteChecks; }
    public void setPrerequisiteChecks(List<String> prerequisiteChecks) { this.prerequisiteChecks = prerequisiteChecks; }

    public List<String> getRequiredLabValues() { return requiredLabValues; }
    public void setRequiredLabValues(List<String> requiredLabValues) { this.requiredLabValues = requiredLabValues; }

    public String getExpectedOutcome() { return expectedOutcome; }
    public void setExpectedOutcome(String expectedOutcome) { this.expectedOutcome = expectedOutcome; }

    public String getMonitoringParameters() { return monitoringParameters; }
    public void setMonitoringParameters(String monitoringParameters) { this.monitoringParameters = monitoringParameters; }

    // Utility methods

    /**
     * Check if action is STAT urgency
     */
    public boolean isStatUrgency() {
        return "STAT".equals(urgency);
    }

    /**
     * Check if action is therapeutic (medication)
     */
    public boolean isTherapeutic() {
        return ActionType.THERAPEUTIC.equals(actionType);
    }

    /**
     * Check if action is diagnostic
     */
    public boolean isDiagnostic() {
        return ActionType.DIAGNOSTIC.equals(actionType);
    }

    /**
     * Check if action has strong evidence
     */
    public boolean hasStrongEvidence() {
        return "STRONG".equals(evidenceStrength);
    }

    @Override
    public String toString() {
        return "ClinicalAction{" +
            "actionId='" + actionId + '\'' +
            ", actionType=" + actionType +
            ", description='" + description + '\'' +
            ", urgency='" + urgency + '\'' +
            ", timeframe='" + timeframe + '\'' +
            ", sequenceOrder=" + sequenceOrder +
            '}';
    }

    /**
     * Action Type Enumeration
     */
    public enum ActionType {
        DIAGNOSTIC,           // Order tests (labs, imaging, cultures)
        THERAPEUTIC,          // Medications, procedures, interventions
        MONITORING,           // Vital sign monitoring, lab monitoring
        ESCALATION,           // ICU transfer, specialist consult
        MEDICATION_REVIEW     // Review/adjust existing medications
    }
}
