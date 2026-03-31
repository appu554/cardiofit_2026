package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Output action record emitted by Module 6.
 * Main collector output of the ClinicalActionEngine.
 *
 * Also serves as the detailed clinical action record used by Module 3 (ActionBuilder).
 * The Module 6 fields (action_type as String, alert) coexist with Module 3 fields.
 */
public class ClinicalAction implements Serializable {
    private static final long serialVersionUID = 1L;

    // ── Module 6 fields ──────────────────────────────────────────────────────

    @JsonProperty("action_id") private String actionId;
    /** Module 6 action type: NEW_ALERT, ESCALATION, AUTO_RESOLVED, ACKNOWLEDGMENT */
    @JsonProperty("action_type_str") private String actionTypeStr;
    @JsonProperty("alert") private ClinicalAlert alert;
    @JsonProperty("timestamp") private long timestamp = System.currentTimeMillis();

    // ── Module 3 / ActionBuilder fields ──────────────────────────────────────

    @JsonProperty("action_type")
    private ActionType actionType;

    @JsonProperty("description")
    private String description;

    @JsonProperty("sequence_order")
    private int sequenceOrder;

    @JsonProperty("urgency")
    private String urgency;

    @JsonProperty("timeframe")
    private String timeframe;

    @JsonProperty("timeframe_rationale")
    private String timeframeRationale;

    @JsonProperty("medication_details")
    private MedicationDetails medicationDetails;

    @JsonProperty("diagnostic_details")
    private DiagnosticDetails diagnosticDetails;

    @JsonProperty("clinical_rationale")
    private String clinicalRationale;

    @JsonProperty("evidence_references")
    private List<EvidenceReference> evidenceReferences;

    @JsonProperty("evidence_strength")
    private String evidenceStrength;

    @JsonProperty("prerequisite_checks")
    private List<String> prerequisiteChecks;

    @JsonProperty("required_lab_values")
    private List<String> requiredLabValues;

    @JsonProperty("expected_outcome")
    private String expectedOutcome;

    @JsonProperty("monitoring_parameters")
    private String monitoringParameters;

    public ClinicalAction() {
        this.actionId = java.util.UUID.randomUUID().toString();
        this.evidenceReferences = new ArrayList<>();
        this.prerequisiteChecks = new ArrayList<>();
        this.requiredLabValues = new ArrayList<>();
    }

    // ── Module 6 static factories ─────────────────────────────────────────────

    public static ClinicalAction newAlert(ClinicalAlert alert) {
        ClinicalAction a = new ClinicalAction();
        a.actionTypeStr = "NEW_ALERT";
        a.alert = alert;
        return a;
    }

    public static ClinicalAction escalation(ClinicalAlert alert, String escalateTo) {
        ClinicalAction a = new ClinicalAction();
        a.actionTypeStr = "ESCALATION";
        a.alert = alert;
        return a;
    }

    public static ClinicalAction autoResolved(ClinicalAlert alert) {
        ClinicalAction a = new ClinicalAction();
        a.actionTypeStr = "AUTO_RESOLVED";
        a.alert = alert;
        return a;
    }

    // ── Module 6 accessors ────────────────────────────────────────────────────

    public String getActionId() { return actionId; }
    public void setActionId(String actionId) { this.actionId = actionId; }

    /** Module 6 string action type. Use {@link #getActionType()} for Module 3 enum type. */
    public String getActionTypeStr() { return actionTypeStr; }
    public void setActionTypeStr(String actionTypeStr) { this.actionTypeStr = actionTypeStr; }

    public ClinicalAlert getAlert() { return alert; }
    public void setAlert(ClinicalAlert alert) { this.alert = alert; }

    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

    // ── Module 3 / ActionBuilder accessors ───────────────────────────────────

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
    public void setEvidenceReferences(List<EvidenceReference> evidenceReferences) { this.evidenceReferences = evidenceReferences; }

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

    public boolean isStatUrgency() { return "STAT".equals(urgency); }
    public boolean isTherapeutic() { return ActionType.THERAPEUTIC.equals(actionType); }
    public boolean isDiagnostic() { return ActionType.DIAGNOSTIC.equals(actionType); }
    public boolean hasStrongEvidence() { return "STRONG".equals(evidenceStrength); }

    @Override
    public String toString() {
        return "ClinicalAction{actionId='" + actionId + "', actionType=" + actionType +
            ", actionTypeStr='" + actionTypeStr + "', description='" + description + "'}";
    }

    /**
     * Action Type Enumeration (Module 3 / ActionBuilder).
     */
    public enum ActionType {
        DIAGNOSTIC,
        THERAPEUTIC,
        MONITORING,
        ESCALATION,
        MEDICATION_REVIEW
    }
}
