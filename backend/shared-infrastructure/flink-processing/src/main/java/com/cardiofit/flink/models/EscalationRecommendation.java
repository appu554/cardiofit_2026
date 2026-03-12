package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;

/**
 * Escalation Recommendation with clinical evidence.
 *
 * Represents a triggered escalation rule with patient-specific evidence for
 * clinical decision support. Generated when escalation_rules are triggered.
 *
 * Key Features:
 * - Contains escalation level and rationale from protocol
 * - Includes patient-specific clinical evidence supporting escalation
 * - Provides urgency level for prioritization
 * - Tracks patient and encounter identifiers for FHIR compliance
 *
 * Example Usage:
 * When a sepsis protocol escalation rule triggers for lactate >= 4.0:
 * - ruleId: "SEPSIS-ESC-001"
 * - escalationLevel: "ICU_TRANSFER"
 * - rationale: "Septic shock requiring vasopressor support"
 * - evidence: ["lactate: 4.5 mmol/L (>= threshold)", "Systolic BP: 85 mmHg (hypotensive)"]
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-10-21
 */
public class EscalationRecommendation implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Rule and protocol identifiers
     */
    @JsonProperty("ruleId")
    private String ruleId;

    @JsonProperty("protocolId")
    private String protocolId;

    @JsonProperty("protocolName")
    private String protocolName;

    /**
     * Escalation details
     */
    @JsonProperty("escalationLevel")
    private String escalationLevel; // ICU_TRANSFER, SPECIALIST_CONSULT, RAPID_RESPONSE, etc.

    @JsonProperty("specialty")
    private String specialty; // Critical Care, Cardiology, Infectious Disease, etc.

    @JsonProperty("rationale")
    private String rationale; // Clinical rationale for escalation

    @JsonProperty("urgency")
    private String urgency; // IMMEDIATE, URGENT, ROUTINE

    /**
     * Timestamp when escalation was triggered
     */
    @JsonProperty("timestamp")
    private Instant timestamp;

    /**
     * Clinical evidence supporting escalation decision
     * List of specific observations and values that triggered escalation
     */
    @JsonProperty("evidence")
    private List<String> evidence;

    /**
     * Patient identifiers for tracking
     */
    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("encounterId")
    private String encounterId;

    public EscalationRecommendation() {
        this.evidence = new ArrayList<>();
        this.timestamp = Instant.now();
    }

    public EscalationRecommendation(String ruleId, String escalationLevel) {
        this();
        this.ruleId = ruleId;
        this.escalationLevel = escalationLevel;
    }

    // Getters and setters

    public String getRuleId() {
        return ruleId;
    }

    public void setRuleId(String ruleId) {
        this.ruleId = ruleId;
    }

    public String getProtocolId() {
        return protocolId;
    }

    public void setProtocolId(String protocolId) {
        this.protocolId = protocolId;
    }

    public String getProtocolName() {
        return protocolName;
    }

    public void setProtocolName(String protocolName) {
        this.protocolName = protocolName;
    }

    public String getEscalationLevel() {
        return escalationLevel;
    }

    public void setEscalationLevel(String escalationLevel) {
        this.escalationLevel = escalationLevel;
    }

    public String getSpecialty() {
        return specialty;
    }

    public void setSpecialty(String specialty) {
        this.specialty = specialty;
    }

    public String getRationale() {
        return rationale;
    }

    public void setRationale(String rationale) {
        this.rationale = rationale;
    }

    public String getUrgency() {
        return urgency;
    }

    public void setUrgency(String urgency) {
        this.urgency = urgency;
    }

    public Instant getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(Instant timestamp) {
        this.timestamp = timestamp;
    }

    public List<String> getEvidence() {
        return evidence;
    }

    public void setEvidence(List<String> evidence) {
        this.evidence = evidence;
    }

    public void addEvidence(String evidenceItem) {
        if (this.evidence == null) {
            this.evidence = new ArrayList<>();
        }
        this.evidence.add(evidenceItem);
    }

    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public String getEncounterId() {
        return encounterId;
    }

    public void setEncounterId(String encounterId) {
        this.encounterId = encounterId;
    }

    /**
     * Check if escalation is immediate priority
     */
    @com.fasterxml.jackson.annotation.JsonIgnore
    public boolean isImmediate() {
        return "IMMEDIATE".equalsIgnoreCase(urgency);
    }

    /**
     * Get evidence count for metrics
     */
    @com.fasterxml.jackson.annotation.JsonIgnore
    public int getEvidenceCount() {
        return evidence != null ? evidence.size() : 0;
    }

    @Override
    public String toString() {
        return "EscalationRecommendation{" +
                "ruleId='" + ruleId + '\'' +
                ", protocolId='" + protocolId + '\'' +
                ", escalationLevel='" + escalationLevel + '\'' +
                ", specialty='" + specialty + '\'' +
                ", urgency='" + urgency + '\'' +
                ", patientId='" + patientId + '\'' +
                ", evidenceCount=" + getEvidenceCount() +
                ", timestamp=" + timestamp +
                '}';
    }
}
