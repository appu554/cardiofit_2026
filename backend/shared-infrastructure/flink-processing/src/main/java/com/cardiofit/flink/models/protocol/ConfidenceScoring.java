package com.cardiofit.flink.models.protocol;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Confidence scoring configuration for protocol matching.
 *
 * <p>Contains:
 * - Base confidence score (0.0-1.0)
 * - List of modifiers that adjust confidence based on patient conditions
 * - Activation threshold (minimum confidence required for protocol activation)
 *
 * <p>Score calculation:
 * confidence = base_confidence + sum(modifier.adjustment if condition_met)
 * clamped to [0.0, 1.0]
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class ConfidenceScoring implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Base confidence score for this protocol (0.0-1.0)
     */
    private double baseConfidence;

    /**
     * List of conditional modifiers that adjust confidence
     */
    private List<ConfidenceModifier> modifiers;

    /**
     * Minimum confidence required for protocol activation (0.0-1.0)
     */
    private double activationThreshold;

    public ConfidenceScoring() {
        this.modifiers = new ArrayList<>();
        this.baseConfidence = 0.85;
        this.activationThreshold = 0.70;
    }

    public ConfidenceScoring(double baseConfidence, double activationThreshold) {
        this();
        this.baseConfidence = baseConfidence;
        this.activationThreshold = activationThreshold;
    }

    // Getters and setters

    public double getBaseConfidence() {
        return baseConfidence;
    }

    public void setBaseConfidence(double baseConfidence) {
        this.baseConfidence = baseConfidence;
    }

    public List<ConfidenceModifier> getModifiers() {
        return modifiers;
    }

    public void setModifiers(List<ConfidenceModifier> modifiers) {
        this.modifiers = modifiers;
    }

    public void addModifier(ConfidenceModifier modifier) {
        if (this.modifiers == null) {
            this.modifiers = new ArrayList<>();
        }
        this.modifiers.add(modifier);
    }

    public double getActivationThreshold() {
        return activationThreshold;
    }

    public void setActivationThreshold(double activationThreshold) {
        this.activationThreshold = activationThreshold;
    }

    @Override
    public String toString() {
        return "ConfidenceScoring{" +
                "baseConfidence=" + baseConfidence +
                ", modifiers=" + (modifiers != null ? modifiers.size() : 0) +
                ", activationThreshold=" + activationThreshold +
                '}';
    }
}
