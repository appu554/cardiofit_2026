package com.cardiofit.flink.protocol.models;

import com.cardiofit.flink.models.protocol.ConfidenceScoring;
import com.cardiofit.flink.models.protocol.TriggerCriteria;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Enhanced clinical protocol model for Module 3 CDS.
 *
 * <p>Represents a complete clinical protocol with:
 * - Identification and metadata
 * - Trigger criteria for automatic protocol activation (Phase 1)
 * - Confidence scoring for protocol ranking (Phase 2)
 * - Time constraints for bundle tracking
 * - Clinical actions and recommendations
 * - Evidence-based guidelines
 *
 * @author Module 3 CDS Team
 * @version 1.2 - Phase 2 Integration
 * @since 2025-01-15
 */
public class Protocol implements Serializable {
    private static final long serialVersionUID = 1L;

    private String protocolId;
    private String name;
    private String category;
    private String specialty;
    private String version;
    private TriggerCriteria triggerCriteria; // Phase 1 integration
    private ConfidenceScoring confidenceScoring; // Phase 2 integration
    private List<TimeConstraint> timeConstraints;

    /**
     * Default constructor.
     */
    public Protocol() {
        this.timeConstraints = new ArrayList<>();
    }

    /**
     * Constructor with basic fields.
     *
     * @param protocolId Unique protocol identifier
     * @param name Protocol name
     * @param category Protocol category
     */
    public Protocol(String protocolId, String name, String category) {
        this();
        this.protocolId = protocolId;
        this.name = name;
        this.category = category;
    }

    /**
     * Adds a time constraint to the protocol.
     *
     * @param constraint The time constraint to add
     */
    public void addTimeConstraint(TimeConstraint constraint) {
        if (this.timeConstraints == null) {
            this.timeConstraints = new ArrayList<>();
        }
        this.timeConstraints.add(constraint);
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

    public String getVersion() {
        return version;
    }

    public void setVersion(String version) {
        this.version = version;
    }

    public List<TimeConstraint> getTimeConstraints() {
        return timeConstraints;
    }

    public void setTimeConstraints(List<TimeConstraint> timeConstraints) {
        this.timeConstraints = timeConstraints;
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

    @Override
    public String toString() {
        return "Protocol{" +
                "protocolId='" + protocolId + '\'' +
                ", name='" + name + '\'' +
                ", category='" + category + '\'' +
                ", triggerCriteria=" + (triggerCriteria != null ? "present" : "null") +
                ", confidenceScoring=" + (confidenceScoring != null ? "present" : "null") +
                ", timeConstraints=" + (timeConstraints != null ? timeConstraints.size() : 0) +
                '}';
    }
}
