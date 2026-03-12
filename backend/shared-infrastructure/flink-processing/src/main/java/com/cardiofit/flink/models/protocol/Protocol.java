package com.cardiofit.flink.models.protocol;

import java.io.Serializable;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;

/**
 * Clinical Protocol representation for CDS engine.
 *
 * <p>Represents a complete clinical protocol with:
 * - Metadata (ID, name, version, category, specialty)
 * - Trigger criteria (conditions that activate the protocol)
 * - Confidence scoring (how well patient matches protocol)
 * - Actions (clinical interventions to perform)
 * - Time constraints (bundle deadlines)
 * - Escalation rules (ICU transfer criteria)
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class Protocol implements Serializable {
    private static final long serialVersionUID = 1L;

    // Metadata
    private String protocolId;
    private String name;
    private String version;
    private String category; // INFECTIOUS, CARDIOVASCULAR, METABOLIC, etc.
    private String specialty; // CRITICAL_CARE, CARDIOLOGY, etc.
    private String description;

    // Trigger criteria
    private TriggerCriteria triggerCriteria;

    // Confidence scoring
    private ConfidenceScoring confidenceScoring;

    // Actions (not fully implemented in this phase, placeholder)
    private List<Object> actions;

    // Time constraints (not fully implemented in this phase, placeholder)
    private List<Object> timeConstraints;

    // Escalation rules
    private List<EscalationRule> escalationRules;

    // Evidence and validation
    private String evidenceSource;
    private String evidenceLevel;
    private List<String> contraindications;

    // Evidence Repository Integration (Phase 7)
    // List of citation PMIDs supporting this protocol
    private List<String> citationIds;

    // Date when evidence was last reviewed/updated
    private LocalDate evidenceLastUpdated;

    // Overall evidence strength calculated from linked citations
    // Values: "STRONG", "MODERATE", "WEAK", "INSUFFICIENT"
    private String overallEvidenceStrength;

    // Map of protocol step IDs to their supporting citation PMIDs
    // Allows granular evidence tracking at the step level
    // Example: {"step_1": ["26903338", "12345678"], "step_2": ["87654321"]}
    private Map<String, List<String>> stepCitations;

    public Protocol() {
        this.actions = new ArrayList<>();
        this.timeConstraints = new ArrayList<>();
        this.escalationRules = new ArrayList<>();
        this.contraindications = new ArrayList<>();
        this.citationIds = new ArrayList<>();
        this.stepCitations = new HashMap<>();
    }

    public Protocol(String protocolId, String name) {
        this();
        this.protocolId = protocolId;
        this.name = name;
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

    public TriggerCriteria getTriggerCriteria() {
        return triggerCriteria;
    }

    public void setTriggerCriteria(TriggerCriteria triggerCriteria) {
        this.triggerCriteria = triggerCriteria;
    }

    public ConfidenceScoring getConfidenceScoring() {
        return confidenceScoring;
    }

    public void setConfidenceScoring(ConfidenceScoring confidenceScoring) {
        this.confidenceScoring = confidenceScoring;
    }

    public List<Object> getActions() {
        return actions;
    }

    public void setActions(List<Object> actions) {
        this.actions = actions;
    }

    public List<Object> getTimeConstraints() {
        return timeConstraints;
    }

    public void setTimeConstraints(List<Object> timeConstraints) {
        this.timeConstraints = timeConstraints;
    }

    public List<EscalationRule> getEscalationRules() {
        return escalationRules;
    }

    public void setEscalationRules(List<EscalationRule> escalationRules) {
        this.escalationRules = escalationRules;
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

    // Evidence Repository Integration getters and setters (Phase 7)

    public List<String> getCitationIds() {
        return citationIds;
    }

    public void setCitationIds(List<String> citationIds) {
        this.citationIds = citationIds;
    }

    public LocalDate getEvidenceLastUpdated() {
        return evidenceLastUpdated;
    }

    public void setEvidenceLastUpdated(LocalDate evidenceLastUpdated) {
        this.evidenceLastUpdated = evidenceLastUpdated;
    }

    public String getOverallEvidenceStrength() {
        return overallEvidenceStrength;
    }

    public void setOverallEvidenceStrength(String overallEvidenceStrength) {
        this.overallEvidenceStrength = overallEvidenceStrength;
    }

    public Map<String, List<String>> getStepCitations() {
        return stepCitations;
    }

    public void setStepCitations(Map<String, List<String>> stepCitations) {
        this.stepCitations = stepCitations;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        Protocol protocol = (Protocol) o;
        return Objects.equals(protocolId, protocol.protocolId);
    }

    @Override
    public int hashCode() {
        return Objects.hash(protocolId);
    }

    @Override
    public String toString() {
        return "Protocol{" +
                "protocolId='" + protocolId + '\'' +
                ", name='" + name + '\'' +
                ", version='" + version + '\'' +
                ", category='" + category + '\'' +
                ", specialty='" + specialty + '\'' +
                '}';
    }
}
