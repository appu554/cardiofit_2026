package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Contraindication - Safety warning for clinical actions
 *
 * Represents a contraindication check result with severity assessment,
 * evidence of the contraindication, and alternative recommendations.
 * Used to ensure patient safety in clinical decision support.
 *
 * @author CardioFit Platform - Module 3
 * @version 1.0
 * @since 2025-10-20
 */
public class Contraindication implements Serializable {
    private static final long serialVersionUID = 1L;

    // Contraindication Identity
    @JsonProperty("contraindication_id")
    private String contraindicationId;

    @JsonProperty("contraindication_type")
    private ContraindicationType contraindicationType;

    @JsonProperty("contraindication_description")
    private String contraindicationDescription;

    @JsonProperty("severity")
    private Severity severity;

    // Check Result
    @JsonProperty("found")
    private boolean found;

    @JsonProperty("evidence")
    private String evidence;

    @JsonProperty("risk_score")
    private double riskScore;  // 0.0-1.0

    // Alternative Plan
    @JsonProperty("alternative_available")
    private boolean alternativeAvailable;

    @JsonProperty("alternative_medication")
    private String alternativeMedication;

    @JsonProperty("alternative_rationale")
    private String alternativeRationale;

    // Clinical Decision Support
    @JsonProperty("clinical_guidance")
    private String clinicalGuidance;

    @JsonProperty("override_justification")
    private String overrideJustification;

    // Default constructor
    public Contraindication() {
        this.contraindicationId = java.util.UUID.randomUUID().toString();
        this.found = false;
        this.riskScore = 0.0;
    }

    // Constructor with type and description
    public Contraindication(ContraindicationType type, String description) {
        this();
        this.contraindicationType = type;
        this.contraindicationDescription = description;
    }

    // Getters and Setters
    public String getContraindicationId() { return contraindicationId; }
    public void setContraindicationId(String contraindicationId) { this.contraindicationId = contraindicationId; }

    public ContraindicationType getContraindicationType() { return contraindicationType; }
    public void setContraindicationType(ContraindicationType contraindicationType) {
        this.contraindicationType = contraindicationType;
    }

    public String getContraindicationDescription() { return contraindicationDescription; }
    public void setContraindicationDescription(String contraindicationDescription) {
        this.contraindicationDescription = contraindicationDescription;
    }

    public Severity getSeverity() { return severity; }
    public void setSeverity(Severity severity) { this.severity = severity; }

    public boolean isFound() { return found; }
    public void setFound(boolean found) { this.found = found; }

    public String getEvidence() { return evidence; }
    public void setEvidence(String evidence) { this.evidence = evidence; }

    public double getRiskScore() { return riskScore; }
    public void setRiskScore(double riskScore) { this.riskScore = riskScore; }

    public boolean isAlternativeAvailable() { return alternativeAvailable; }
    public void setAlternativeAvailable(boolean alternativeAvailable) {
        this.alternativeAvailable = alternativeAvailable;
    }

    public String getAlternativeMedication() { return alternativeMedication; }
    public void setAlternativeMedication(String alternativeMedication) {
        this.alternativeMedication = alternativeMedication;
    }

    public String getAlternativeRationale() { return alternativeRationale; }
    public void setAlternativeRationale(String alternativeRationale) {
        this.alternativeRationale = alternativeRationale;
    }

    public String getClinicalGuidance() { return clinicalGuidance; }
    public void setClinicalGuidance(String clinicalGuidance) { this.clinicalGuidance = clinicalGuidance; }

    public String getOverrideJustification() { return overrideJustification; }
    public void setOverrideJustification(String overrideJustification) {
        this.overrideJustification = overrideJustification;
    }

    // Utility methods

    /**
     * Check if contraindication is absolute (must not use)
     */
    public boolean isAbsolute() {
        return Severity.ABSOLUTE.equals(severity);
    }

    /**
     * Check if contraindication requires caution
     */
    public boolean requiresCaution() {
        return Severity.CAUTION.equals(severity);
    }

    /**
     * Check if high risk (risk score > 0.7)
     */
    public boolean isHighRisk() {
        return riskScore > 0.7;
    }

    @Override
    public String toString() {
        return "Contraindication{" +
            "contraindicationType=" + contraindicationType +
            ", description='" + contraindicationDescription + '\'' +
            ", severity=" + severity +
            ", found=" + found +
            ", riskScore=" + riskScore +
            '}';
    }

    /**
     * Contraindication Type Enumeration
     */
    public enum ContraindicationType {
        ALLERGY,              // Drug allergy
        DRUG_INTERACTION,     // Drug-drug interaction
        ORGAN_DYSFUNCTION,    // Renal/hepatic/cardiac dysfunction
        PREGNANCY,            // Pregnancy/breastfeeding
        AGE_RESTRICTION,      // Age-based contraindication
        DISEASE_STATE         // Specific disease contraindication
    }

    /**
     * Severity Enumeration
     */
    public enum Severity {
        ABSOLUTE,    // Absolutely contraindicated, do not use
        RELATIVE,    // Relative contraindication, use with caution
        CAUTION      // Use with monitoring and caution
    }
}
