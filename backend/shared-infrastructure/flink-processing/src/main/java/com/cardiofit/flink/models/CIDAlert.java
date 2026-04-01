package com.cardiofit.flink.models;

import java.util.List;
import java.util.UUID;

/**
 * CID Alert event — output model for Module 8.
 * 14 fields per DD#7 Section 4 alert event schema.
 *
 * All enum fields are String-typed for Kafka JSON serialization
 * (same pattern as BPVariabilityMetrics).
 */
public class CIDAlert implements java.io.Serializable {
    private static final long serialVersionUID = 1L;

    // Identity
    private String alertId;          // UUID v7
    private String patientId;

    // Rule
    private String ruleId;           // CID_01 through CID_17
    private String severity;         // HALT / PAUSE / SOFT_FLAG
    private String triggerSummary;   // Human-readable trigger description

    // Clinical context
    private List<String> medicationsInvolved;    // Drug names in the interaction
    private String labValuesInvolved;            // JSON: relevant labs at trigger time
    private String vitalsInvolved;               // JSON: relevant vitals
    private String recommendedAction;            // Deterministic recommendation from rule

    // Deduplication
    private String suppressionKey;   // ruleId + patientId + hash(medications)

    // Lifecycle
    private long createdAt;
    private Long resolvedAt;         // nullable — set when acknowledged/resolved
    private String resolution;       // nullable — PHYSICIAN_ACKNOWLEDGED / PHYSICIAN_ACTIONED / AUTO_RESOLVED / EXPIRED

    // Provenance
    private String correlationId;    // From triggering CanonicalEvent

    // --- Constructors ---
    public CIDAlert() {}

    public static CIDAlert create(CIDRuleId rule, String patientId,
                                   String triggerSummary, List<String> medications,
                                   String recommendedAction, String correlationId) {
        CIDAlert alert = new CIDAlert();
        alert.alertId = UUID.randomUUID().toString();
        alert.patientId = patientId;
        alert.ruleId = rule.name();
        alert.severity = rule.getSeverity().name();
        alert.triggerSummary = triggerSummary;
        alert.medicationsInvolved = medications;
        alert.recommendedAction = recommendedAction;
        alert.correlationId = correlationId;
        alert.createdAt = System.currentTimeMillis();

        // R8 fix: Suppression key uses pipe-delimited meds for deterministic hashing.
        // Without delimiter, "ACEI"+"SGLT2I" and "ACEISGLT"+"2I" hash identically.
        String medsHash = medications == null || medications.isEmpty() ? "none"
            : String.valueOf(medications.stream().sorted()
                .collect(java.util.stream.Collectors.joining("|")).hashCode());
        alert.suppressionKey = rule.name() + ":" + patientId + ":" + medsHash;

        return alert;
    }

    // --- All getters and setters ---
    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public String getRuleId() { return ruleId; }
    public void setRuleId(String ruleId) { this.ruleId = ruleId; }
    public String getSeverity() { return severity; }
    public void setSeverity(String severity) { this.severity = severity; }
    public String getTriggerSummary() { return triggerSummary; }
    public void setTriggerSummary(String triggerSummary) { this.triggerSummary = triggerSummary; }
    public List<String> getMedicationsInvolved() { return medicationsInvolved; }
    public void setMedicationsInvolved(List<String> medicationsInvolved) { this.medicationsInvolved = medicationsInvolved; }
    public String getLabValuesInvolved() { return labValuesInvolved; }
    public void setLabValuesInvolved(String labValuesInvolved) { this.labValuesInvolved = labValuesInvolved; }
    public String getVitalsInvolved() { return vitalsInvolved; }
    public void setVitalsInvolved(String vitalsInvolved) { this.vitalsInvolved = vitalsInvolved; }
    public String getRecommendedAction() { return recommendedAction; }
    public void setRecommendedAction(String recommendedAction) { this.recommendedAction = recommendedAction; }
    public String getSuppressionKey() { return suppressionKey; }
    public void setSuppressionKey(String suppressionKey) { this.suppressionKey = suppressionKey; }
    public long getCreatedAt() { return createdAt; }
    public void setCreatedAt(long createdAt) { this.createdAt = createdAt; }
    public Long getResolvedAt() { return resolvedAt; }
    public void setResolvedAt(Long resolvedAt) { this.resolvedAt = resolvedAt; }
    public String getResolution() { return resolution; }
    public void setResolution(String resolution) { this.resolution = resolution; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String correlationId) { this.correlationId = correlationId; }
}
