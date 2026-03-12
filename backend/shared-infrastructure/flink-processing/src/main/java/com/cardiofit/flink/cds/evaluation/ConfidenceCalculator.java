package com.cardiofit.flink.cds.evaluation;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.protocol.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;

/**
 * Calculates confidence scores for protocol matches using base confidence + modifiers.
 *
 * <p>Confidence scoring allows ranking of multiple matching protocols by how well
 * the patient matches the protocol's intended population.
 *
 * <p>Score calculation:
 * <ol>
 *   <li>Start with base_confidence from protocol</li>
 *   <li>For each modifier, evaluate condition and add adjustment if true</li>
 *   <li>Clamp final score to [0.0, 1.0]</li>
 *   <li>Check if score >= activation_threshold</li>
 * </ol>
 *
 * <p>Example:
 * <pre>
 * ConditionEvaluator evaluator = new ConditionEvaluator();
 * ConfidenceCalculator calculator = new ConfidenceCalculator(evaluator);
 * double confidence = calculator.calculateConfidence(protocol, context);
 * if (calculator.meetsActivationThreshold(protocol, confidence)) {
 *     // Protocol activates
 * }
 * </pre>
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class ConfidenceCalculator implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger logger = LoggerFactory.getLogger(ConfidenceCalculator.class);

    public static final double DEFAULT_BASE_CONFIDENCE = 0.85;
    public static final double DEFAULT_ACTIVATION_THRESHOLD = 0.70;

    private final ConditionEvaluator conditionEvaluator;

    /**
     * Creates a new ConfidenceCalculator with the specified condition evaluator.
     *
     * @param conditionEvaluator The evaluator to use for modifier conditions
     * @throws IllegalArgumentException if conditionEvaluator is null
     */
    public ConfidenceCalculator(ConditionEvaluator conditionEvaluator) {
        if (conditionEvaluator == null) {
            throw new IllegalArgumentException("ConditionEvaluator cannot be null");
        }
        this.conditionEvaluator = conditionEvaluator;
    }

    /**
     * Calculates the confidence score for a protocol match.
     *
     * <p>Algorithm:
     * <ol>
     *   <li>Start with base_confidence from protocol (or default if not set)</li>
     *   <li>For each confidence modifier:
     *     <ul>
     *       <li>Evaluate the modifier's condition using ConditionEvaluator</li>
     *       <li>If condition is true, add modifier's adjustment to confidence</li>
     *     </ul>
     *   </li>
     *   <li>Clamp final confidence to [0.0, 1.0] range</li>
     * </ol>
     *
     * @param protocol The protocol to score
     * @param context The patient context
     * @return Confidence score (0.0-1.0)
     */
    public double calculateConfidence(Protocol protocol, EnrichedPatientContext context) {
        if (protocol == null || context == null) {
            logger.warn("Null protocol or context provided");
            return 0.0;
        }

        ConfidenceScoring scoring = protocol.getConfidenceScoring();

        // Use default if no scoring defined
        if (scoring == null) {
            logger.debug("No confidence_scoring defined for protocol {}, using default {}",
                    protocol.getProtocolId(), DEFAULT_BASE_CONFIDENCE);
            return DEFAULT_BASE_CONFIDENCE;
        }

        double confidence = scoring.getBaseConfidence();
        logger.debug("Protocol {} base confidence: {}", protocol.getProtocolId(), confidence);

        // Apply modifiers
        if (scoring.getModifiers() != null && !scoring.getModifiers().isEmpty()) {
            for (ConfidenceModifier modifier : scoring.getModifiers()) {
                if (modifier == null || modifier.getCondition() == null) {
                    logger.warn("Invalid modifier encountered, skipping");
                    continue;
                }

                try {
                    // Evaluate modifier condition
                    boolean conditionMet = conditionEvaluator.evaluateCondition(
                            modifier.getCondition(),
                            context,
                            0); // Start at depth 0

                    if (conditionMet) {
                        double adjustment = modifier.getAdjustment();
                        confidence += adjustment;

                        logger.debug("Applied modifier {}: {} (new confidence: {})",
                                modifier.getModifierId(),
                                adjustment,
                                confidence);
                    } else {
                        logger.debug("Modifier {} condition not met, skipping",
                                modifier.getModifierId());
                    }
                } catch (Exception e) {
                    logger.error("Error evaluating modifier {}: {}",
                            modifier.getModifierId(), e.getMessage());
                    // Continue with other modifiers
                }
            }
        }

        // Clamp to [0.0, 1.0]
        confidence = clamp(confidence, 0.0, 1.0);

        logger.info("Final confidence for protocol {}: {}",
                protocol.getProtocolId(), confidence);

        return confidence;
    }

    /**
     * Checks if confidence score meets the activation threshold.
     *
     * <p>The threshold is taken from the protocol's confidence_scoring section.
     * If not defined, uses DEFAULT_ACTIVATION_THRESHOLD (0.70).
     *
     * @param protocol The protocol
     * @param confidence The calculated confidence score
     * @return true if confidence >= activation_threshold
     */
    public boolean meetsActivationThreshold(Protocol protocol, double confidence) {
        if (protocol == null) {
            logger.warn("Null protocol provided to threshold check");
            return false;
        }

        double threshold = DEFAULT_ACTIVATION_THRESHOLD;

        if (protocol.getConfidenceScoring() != null) {
            threshold = protocol.getConfidenceScoring().getActivationThreshold();
        }

        boolean meets = confidence >= threshold;

        logger.debug("Protocol {} confidence {} {} threshold {}",
                protocol.getProtocolId(),
                confidence,
                meets ? ">=" : "<",
                threshold);

        return meets;
    }

    /**
     * Clamps value to [min, max] range.
     *
     * <p>If value < min, returns min.
     * If value > max, returns max.
     * Otherwise returns value unchanged.
     *
     * @param value The value to clamp
     * @param min Minimum allowed value
     * @param max Maximum allowed value
     * @return Clamped value
     */
    public double clamp(double value, double min, double max) {
        if (value < min) {
            logger.debug("Confidence {} clamped to min {}", value, min);
            return min;
        }
        if (value > max) {
            logger.debug("Confidence {} clamped to max {}", value, max);
            return max;
        }
        return value;
    }
}
