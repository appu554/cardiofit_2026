package com.cardiofit.flink.models.protocol;

import java.io.Serializable;

/**
 * Confidence modifier that adjusts protocol confidence based on patient conditions.
 *
 * <p>Modifiers allow fine-tuning of confidence scores based on specific patient factors:
 * - Positive modifiers (+0.05, +0.10) increase confidence
 * - Negative modifiers (-0.05, -0.10) decrease confidence
 *
 * <p>Example:
 * If patient age >= 65, add +0.10 to confidence (elderly patients match sepsis protocol better)
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class ConfidenceModifier implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Unique identifier for this modifier
     */
    private String modifierId;

    /**
     * Condition that must be met for this modifier to apply
     */
    private ProtocolCondition condition;

    /**
     * Adjustment value to add to confidence (-1.0 to +1.0)
     * Positive values increase confidence, negative values decrease it
     */
    private double adjustment;

    /**
     * Human-readable description of what this modifier does
     */
    private String description;

    public ConfidenceModifier() {
    }

    public ConfidenceModifier(String modifierId, ProtocolCondition condition, double adjustment) {
        this.modifierId = modifierId;
        this.condition = condition;
        this.adjustment = adjustment;
    }

    // Getters and setters

    public String getModifierId() {
        return modifierId;
    }

    public void setModifierId(String modifierId) {
        this.modifierId = modifierId;
    }

    public ProtocolCondition getCondition() {
        return condition;
    }

    public void setCondition(ProtocolCondition condition) {
        this.condition = condition;
    }

    public double getAdjustment() {
        return adjustment;
    }

    public void setAdjustment(double adjustment) {
        this.adjustment = adjustment;
    }

    public String getDescription() {
        return description;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    @Override
    public String toString() {
        return "ConfidenceModifier{" +
                "modifierId='" + modifierId + '\'' +
                ", adjustment=" + adjustment +
                ", description='" + description + '\'' +
                '}';
    }
}
