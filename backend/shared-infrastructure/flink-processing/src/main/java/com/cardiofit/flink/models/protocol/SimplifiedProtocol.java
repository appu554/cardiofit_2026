package com.cardiofit.flink.models.protocol;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Simplified Protocol for BroadcastState (Phase 2 CDC Integration)
 *
 * This is a flattened version of Protocol that avoids complex nested structures
 * (TriggerCriteria, ConfidenceScoring, EscalationRule) which cause Flink's TypeExtractor
 * to fail with StackOverflowError due to self-referencing ProtocolCondition trees.
 *
 * Used in BroadcastState for CDC hot-swapping of protocols. The full Protocol class
 * with all nested structures is still used internally for protocol matching logic.
 *
 * @author Phase 2 CDC Integration Team
 * @version 1.0
 * @since 2025-11-22
 */
public class SimplifiedProtocol implements Serializable {
    private static final long serialVersionUID = 1L;

    // Basic metadata
    private String protocolId;
    private String name;
    private String version;
    private String category;
    private String specialty;
    private String description;

    // Evidence metadata
    private String evidenceSource;
    private String evidenceLevel;
    private List<String> contraindications;

    // Simplified trigger representation (just parameter names, no nested conditions)
    private List<String> triggerParameters;

    // Simplified confidence values (no modifiers, just thresholds)
    private double baseConfidence;
    private double activationThreshold;

    public SimplifiedProtocol() {
        this.contraindications = new ArrayList<>();
        this.triggerParameters = new ArrayList<>();
        this.baseConfidence = 0.85;
        this.activationThreshold = 0.70;
    }

    /**
     * Convert full Protocol to SimplifiedProtocol for BroadcastState storage
     */
    public static SimplifiedProtocol fromProtocol(Protocol protocol) {
        SimplifiedProtocol simplified = new SimplifiedProtocol();

        simplified.setProtocolId(protocol.getProtocolId());
        simplified.setName(protocol.getName());
        simplified.setVersion(protocol.getVersion());
        simplified.setCategory(protocol.getCategory());
        simplified.setSpecialty(protocol.getSpecialty());
        simplified.setDescription(protocol.getDescription());
        simplified.setEvidenceSource(protocol.getEvidenceSource());
        simplified.setEvidenceLevel(protocol.getEvidenceLevel());
        simplified.setContraindications(protocol.getContraindications());

        // Extract trigger parameters from TriggerCriteria (if exists)
        if (protocol.getTriggerCriteria() != null && protocol.getTriggerCriteria().getConditions() != null) {
            List<String> params = new ArrayList<>();
            extractParametersFromConditions(protocol.getTriggerCriteria().getConditions(), params);
            simplified.setTriggerParameters(params);
        }

        // Extract confidence values from ConfidenceScoring (if exists)
        if (protocol.getConfidenceScoring() != null) {
            simplified.setBaseConfidence(protocol.getConfidenceScoring().getBaseConfidence());
            simplified.setActivationThreshold(protocol.getConfidenceScoring().getActivationThreshold());
        }

        return simplified;
    }

    /**
     * Recursively extract parameter names from ProtocolCondition tree
     */
    private static void extractParametersFromConditions(List<ProtocolCondition> conditions, List<String> params) {
        if (conditions == null) return;

        for (ProtocolCondition condition : conditions) {
            if (condition.getParameter() != null) {
                params.add(condition.getParameter());
            }
            // Recurse into nested conditions
            if (condition.getConditions() != null && !condition.getConditions().isEmpty()) {
                extractParametersFromConditions(condition.getConditions(), params);
            }
        }
    }

    // Getters and setters

    public String getProtocolId() {
        return protocolId;
    }

    public void setProtocolId(String protocolId) {
        this.protocolId = protocolId;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getVersion() {
        return version;
    }

    public void setVersion(String version) {
        this.version = version;
    }

    public String getCategory() {
        return category;
    }

    public void setCategory(String category) {
        this.category = category;
    }

    public String getSpecialty() {
        return specialty;
    }

    public void setSpecialty(String specialty) {
        this.specialty = specialty;
    }

    public String getDescription() {
        return description;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public String getEvidenceSource() {
        return evidenceSource;
    }

    public void setEvidenceSource(String evidenceSource) {
        this.evidenceSource = evidenceSource;
    }

    public String getEvidenceLevel() {
        return evidenceLevel;
    }

    public void setEvidenceLevel(String evidenceLevel) {
        this.evidenceLevel = evidenceLevel;
    }

    public List<String> getContraindications() {
        return contraindications;
    }

    public void setContraindications(List<String> contraindications) {
        this.contraindications = contraindications;
    }

    public List<String> getTriggerParameters() {
        return triggerParameters;
    }

    public void setTriggerParameters(List<String> triggerParameters) {
        this.triggerParameters = triggerParameters;
    }

    public double getBaseConfidence() {
        return baseConfidence;
    }

    public void setBaseConfidence(double baseConfidence) {
        this.baseConfidence = baseConfidence;
    }

    public double getActivationThreshold() {
        return activationThreshold;
    }

    public void setActivationThreshold(double activationThreshold) {
        this.activationThreshold = activationThreshold;
    }

    @Override
    public String toString() {
        return "SimplifiedProtocol{" +
                "protocolId='" + protocolId + '\'' +
                ", name='" + name + '\'' +
                ", version='" + version + '\'' +
                ", category='" + category + '\'' +
                ", specialty='" + specialty + '\'' +
                ", triggerParameters=" + (triggerParameters != null ? triggerParameters.size() : 0) +
                '}';
    }
}
