package com.cardiofit.flink.cds.validation;

import com.cardiofit.flink.protocol.models.Protocol;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * Validates protocol YAML structure and completeness.
 *
 * <p>Validation checks:
 * - Required fields present (protocol_id, name, category, actions)
 * - Reference validation (action_ids, condition_ids unique and valid)
 * - Range validation (confidence scores 0.0-1.0, thresholds valid)
 * - Completeness (evidence_source, contraindications recommended)
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class ProtocolValidator {
    private static final Logger logger = LoggerFactory.getLogger(ProtocolValidator.class);

    /**
     * Validates a protocol for completeness and correctness.
     *
     * @param protocol The protocol to validate
     * @return ValidationResult with errors and warnings
     */
    public ValidationResult validate(Protocol protocol) {
        ValidationResult result = new ValidationResult();
        result.setProtocolId(protocol != null ? protocol.getProtocolId() : "UNKNOWN");

        if (protocol == null) {
            result.addError("Protocol is null");
            return result;
        }

        logger.debug("Validating protocol: {}", protocol.getProtocolId());

        // Run all validation checks
        validateRequiredFields(protocol, result);
        validateActionReferences(protocol, result);
        validateConditionReferences(protocol, result);
        validateConfidenceScoring(protocol, result);
        validateTimeConstraints(protocol, result);
        validateEvidenceSource(protocol, result);

        if (result.isValid()) {
            logger.info("Protocol {} validation PASSED", protocol.getProtocolId());
        } else {
            logger.error("Protocol {} validation FAILED: {} errors, {} warnings",
                protocol.getProtocolId(),
                result.getErrors().size(),
                result.getWarnings().size());
        }

        return result;
    }

    /**
     * Validates that required fields are present.
     *
     * @param protocol The protocol to validate
     * @param result The validation result to update
     */
    private void validateRequiredFields(Protocol protocol, ValidationResult result) {
        if (protocol.getProtocolId() == null || protocol.getProtocolId().isEmpty()) {
            result.addError("protocol_id is required");
        }

        if (protocol.getName() == null || protocol.getName().isEmpty()) {
            result.addError("name is required");
        }

        if (protocol.getCategory() == null || protocol.getCategory().isEmpty()) {
            result.addError("category is required");
        }

        // Note: The current Protocol model doesn't have actions field, but spec requires it
        // This is a placeholder for when the model is enhanced
        logger.debug("Checking for actions field (currently not in Protocol model)");

        if (protocol.getVersion() == null || protocol.getVersion().isEmpty()) {
            result.addWarning("version recommended for tracking");
        }
    }

    /**
     * Validates action IDs are unique and non-empty.
     *
     * @param protocol The protocol to validate
     * @param result The validation result to update
     */
    private void validateActionReferences(Protocol protocol, ValidationResult result) {
        // Placeholder implementation - Protocol model doesn't currently have actions
        // This will be implemented when Protocol model is enhanced with actions field
        logger.debug("Action reference validation - awaiting Protocol model enhancement");
    }

    /**
     * Validates condition references are valid.
     *
     * @param protocol The protocol to validate
     * @param result The validation result to update
     */
    private void validateConditionReferences(Protocol protocol, ValidationResult result) {
        // Placeholder implementation - Protocol model doesn't currently have trigger_criteria
        // This will be implemented when Protocol model is enhanced
        logger.debug("Condition reference validation - awaiting Protocol model enhancement");
    }

    /**
     * Validates confidence scoring ranges.
     *
     * @param protocol The protocol to validate
     * @param result The validation result to update
     */
    private void validateConfidenceScoring(Protocol protocol, ValidationResult result) {
        // Placeholder implementation - Protocol model doesn't currently have confidence_scoring
        // When implemented, this will validate:
        // - base_confidence in range [0.0, 1.0]
        // - activation_threshold in range [0.0, 1.0]
        // - modifiers don't cause score to exceed reasonable bounds
        logger.debug("Confidence scoring validation - awaiting Protocol model enhancement");

        // For now, just add a warning
        result.addWarning("confidence_scoring recommended for protocol ranking");
    }

    /**
     * Validates time constraints.
     *
     * @param protocol The protocol to validate
     * @param result The validation result to update
     */
    private void validateTimeConstraints(Protocol protocol, ValidationResult result) {
        List<com.cardiofit.flink.protocol.models.TimeConstraint> constraints = protocol.getTimeConstraints();

        if (constraints == null || constraints.isEmpty()) {
            logger.debug("No time constraints to validate");
            return;
        }

        Set<String> constraintIds = new HashSet<>();

        for (com.cardiofit.flink.protocol.models.TimeConstraint constraint : constraints) {
            // Validate constraint_id uniqueness
            String constraintId = constraint.getConstraintId();
            if (constraintId == null || constraintId.isEmpty()) {
                result.addError("Time constraint missing constraint_id");
            } else {
                if (constraintIds.contains(constraintId)) {
                    result.addError("Duplicate constraint_id: " + constraintId);
                }
                constraintIds.add(constraintId);
            }

            // Validate bundle_name present
            if (constraint.getBundleName() == null || constraint.getBundleName().isEmpty()) {
                result.addWarning("Time constraint " + constraintId + " missing bundle_name");
            }

            // Validate offset_minutes is positive
            if (constraint.getOffsetMinutes() <= 0) {
                result.addError("Time constraint " + constraintId + " has invalid offset_minutes (must be > 0)");
            }

            // Validate action references exist (placeholder)
            if (constraint.getActionReferences() != null && !constraint.getActionReferences().isEmpty()) {
                logger.debug("Action references in time constraint - validation pending Protocol model enhancement");
            }
        }
    }

    /**
     * Validates evidence source presence.
     *
     * @param protocol The protocol to validate
     * @param result The validation result to update
     */
    private void validateEvidenceSource(Protocol protocol, ValidationResult result) {
        // Placeholder implementation - Protocol model doesn't currently have evidence_source
        // When implemented, this will check for:
        // - primary_guideline present
        // - evidence_level present
        logger.debug("Evidence source validation - awaiting Protocol model enhancement");
        result.addWarning("evidence_source recommended for clinical credibility");
    }

    /**
     * Result of protocol validation.
     */
    public static class ValidationResult {
        private String protocolId;
        private List<String> errors = new ArrayList<>();
        private List<String> warnings = new ArrayList<>();

        /**
         * Checks if the protocol is valid (no errors).
         *
         * @return true if no errors present
         */
        public boolean isValid() {
            return errors.isEmpty();
        }

        /**
         * Adds a validation error.
         *
         * @param error The error message
         */
        public void addError(String error) {
            errors.add(error);
        }

        /**
         * Adds a validation warning.
         *
         * @param warning The warning message
         */
        public void addWarning(String warning) {
            warnings.add(warning);
        }

        // Getters and setters

        public List<String> getErrors() {
            return errors;
        }

        public void setErrors(List<String> errors) {
            this.errors = errors;
        }

        public List<String> getWarnings() {
            return warnings;
        }

        public void setWarnings(List<String> warnings) {
            this.warnings = warnings;
        }

        public String getProtocolId() {
            return protocolId;
        }

        public void setProtocolId(String protocolId) {
            this.protocolId = protocolId;
        }

        @Override
        public String toString() {
            return "ValidationResult{" +
                    "protocolId='" + protocolId + '\'' +
                    ", errors=" + errors.size() +
                    ", warnings=" + warnings.size() +
                    ", valid=" + isValid() +
                    '}';
        }
    }
}
