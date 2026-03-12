package com.cardiofit.flink.knowledgebase.medications.integration;

import com.cardiofit.flink.knowledgebase.medications.model.Medication;
import com.cardiofit.flink.knowledgebase.medications.loader.MedicationDatabaseLoader;
import com.cardiofit.flink.models.ProtocolAction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Medication Integration Service.
 *
 * Bridge between Phase 6 enhanced medication database and existing Phase 1-5 components.
 *
 * Responsibilities:
 * - Convert between legacy Medication model and enhanced Medication model
 * - Integrate with Phase 1 ProtocolAction (actions reference medicationId)
 * - Integrate with Phase 5 Guidelines (medications linked to guideline recommendations)
 * - Provide facade pattern for clean integration
 *
 * This service ensures backward compatibility while enabling Phase 6 enhancements.
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-24
 */
public class MedicationIntegrationService {
    private static final Logger logger = LoggerFactory.getLogger(MedicationIntegrationService.class);

    private final MedicationDatabaseLoader medicationLoader;

    public MedicationIntegrationService() {
        this.medicationLoader = MedicationDatabaseLoader.getInstance();
    }

    /**
     * Get enhanced medication for a protocol action.
     *
     * Retrieves full Phase 6 medication details for medication referenced in protocol action.
     *
     * @param protocolAction The protocol action with medication reference
     * @return Enhanced Medication object or null if not found
     */
    public Medication getMedicationForProtocolAction(ProtocolAction protocolAction) {
        if (protocolAction == null) {
            logger.warn("Protocol action is null");
            return null;
        }

        // Get medication ID from protocol action
        // Protocol actions may have medication embedded or reference medicationId
        Object medicationObj = protocolAction.getMedication();

        if (medicationObj == null) {
            logger.warn("No medication in protocol action {}", protocolAction.getActionId());
            return null;
        }

        // Handle both legacy Medication and Phase 6 Medication
        String medicationName = null;
        String code = null;

        if (medicationObj instanceof com.cardiofit.flink.models.Medication) {
            com.cardiofit.flink.models.Medication legacyMed =
                (com.cardiofit.flink.models.Medication) medicationObj;
            medicationName = legacyMed.getName();
            code = legacyMed.getCode();
        } else if (medicationObj instanceof Medication) {
            Medication phase6Med = (Medication) medicationObj;
            medicationName = phase6Med.getGenericName();
            code = phase6Med.getRxNormCode();
        }

        // Try to find enhanced medication by name
        if (medicationName != null) {
            Medication enhanced = medicationLoader.getMedicationByName(medicationName);
            if (enhanced != null) {
                logger.debug("Found enhanced medication for protocol action: {}",
                    enhanced.getGenericName());
                return enhanced;
            }
        }

        // Try by code (RxNorm)
        if (code != null) {
            // Search by RxNorm code
            for (Medication med : medicationLoader.getAllMedications()) {
                if (code.equals(med.getRxNormCode())) {
                    logger.debug("Found enhanced medication by RxNorm: {}", med.getGenericName());
                    return med;
                }
            }
        }

        logger.warn("Enhanced medication not found for: {}", medicationName);
        return null;
    }

    /**
     * Get evidence sources for medication from Phase 5 guidelines.
     *
     * @param medicationId The medication ID
     * @return List of guideline IDs that reference this medication
     */
    public java.util.List<String> getEvidenceForMedication(String medicationId) {
        Medication medication = medicationLoader.getMedication(medicationId);

        if (medication == null) {
            logger.warn("Medication not found: {}", medicationId);
            return java.util.Collections.emptyList();
        }

        if (medication.getGuidelineReferences() == null) {
            return java.util.Collections.emptyList();
        }

        logger.debug("Found {} guideline references for {}",
            medication.getGuidelineReferences().size(),
            medication.getGenericName());

        return new java.util.ArrayList<>(medication.getGuidelineReferences());
    }

    /**
     * Convert Phase 6 enhanced Medication to legacy Medication model.
     *
     * For backward compatibility with existing Flink operators that expect legacy model.
     *
     * @param enhanced The enhanced Phase 6 medication
     * @return Legacy Medication object
     */
    public com.cardiofit.flink.models.Medication convertToLegacyModel(Medication enhanced) {
        if (enhanced == null) {
            return null;
        }

        com.cardiofit.flink.models.Medication legacy = new com.cardiofit.flink.models.Medication();

        legacy.setName(enhanced.getGenericName());
        legacy.setCode(enhanced.getRxNormCode());

        // Get standard adult dose
        if (enhanced.getAdultDosing() != null &&
            enhanced.getAdultDosing().getStandard() != null) {

            Medication.AdultDosing.StandardDose standard = enhanced.getAdultDosing().getStandard();

            legacy.setDosage(standard.getDose());
            legacy.setRoute(standard.getRoute());
            legacy.setFrequency(standard.getFrequency());
        }

        legacy.setStatus("active");

        // Set display name
        if (enhanced.getBrandNames() != null && !enhanced.getBrandNames().isEmpty()) {
            legacy.setDisplay(enhanced.getGenericName() + " (" + enhanced.getBrandNames().get(0) + ")");
        } else {
            legacy.setDisplay(enhanced.getGenericName());
        }

        logger.debug("Converted enhanced medication to legacy: {}", enhanced.getGenericName());

        return legacy;
    }

    /**
     * Convert legacy Medication to Phase 6 enhanced model (best effort).
     *
     * Attempts to find matching enhanced medication by name or code.
     *
     * @param legacy The legacy medication
     * @return Enhanced Medication if found, null otherwise
     */
    public Medication convertFromLegacyModel(com.cardiofit.flink.models.Medication legacy) {
        if (legacy == null) {
            return null;
        }

        // Try to find by name
        if (legacy.getName() != null) {
            Medication enhanced = medicationLoader.getMedicationByName(legacy.getName());
            if (enhanced != null) {
                logger.debug("Converted legacy medication to enhanced: {}", enhanced.getGenericName());
                return enhanced;
            }
        }

        // Try to find by code
        if (legacy.getCode() != null) {
            for (Medication med : medicationLoader.getAllMedications()) {
                if (legacy.getCode().equals(med.getRxNormCode())) {
                    logger.debug("Converted legacy medication to enhanced by code: {}",
                        med.getGenericName());
                    return med;
                }
            }
        }

        logger.warn("Could not convert legacy medication to enhanced: {}", legacy.getName());
        return null;
    }

    /**
     * Enrich protocol action with Phase 6 medication details.
     *
     * Updates protocol action with enhanced medication information without changing structure.
     *
     * @param protocolAction The protocol action to enrich
     * @return true if enrichment successful
     */
    public boolean enrichProtocolAction(ProtocolAction protocolAction) {
        if (protocolAction == null || protocolAction.getMedication() == null) {
            return false;
        }

        Medication enhanced = getMedicationForProtocolAction(protocolAction);
        if (enhanced == null) {
            return false;
        }

        // Update legacy medication with enhanced details
        Object medObj = protocolAction.getMedication();
        if (!(medObj instanceof com.cardiofit.flink.models.Medication)) {
            return false; // Can only enrich legacy medications
        }
        com.cardiofit.flink.models.Medication legacyMed =
            (com.cardiofit.flink.models.Medication) medObj;

        // Update code if missing
        if (legacyMed.getCode() == null && enhanced.getRxNormCode() != null) {
            legacyMed.setCode(enhanced.getRxNormCode());
        }

        // Update display with brand names
        if (enhanced.getBrandNames() != null && !enhanced.getBrandNames().isEmpty()) {
            legacyMed.setDisplay(
                enhanced.getGenericName() + " (" + enhanced.getBrandNames().get(0) + ")");
        }

        logger.debug("Enriched protocol action {} with enhanced medication details",
            protocolAction.getActionId());

        return true;
    }

    /**
     * Get medication name for display purposes.
     *
     * Returns generic name with brand name if available.
     *
     * @param medicationId The medication ID
     * @return Display name or null if not found
     */
    public String getMedicationDisplayName(String medicationId) {
        Medication medication = medicationLoader.getMedication(medicationId);

        if (medication == null) {
            return null;
        }

        StringBuilder displayName = new StringBuilder(medication.getGenericName());

        if (medication.getBrandNames() != null && !medication.getBrandNames().isEmpty()) {
            displayName.append(" (")
                      .append(String.join(", ", medication.getBrandNames()))
                      .append(")");
        }

        return displayName.toString();
    }

    /**
     * Check if medication exists in Phase 6 database.
     *
     * @param medicationName The medication name
     * @return true if medication exists in enhanced database
     */
    public boolean medicationExists(String medicationName) {
        return medicationLoader.getMedicationByName(medicationName) != null;
    }

    /**
     * Get medication count in Phase 6 database.
     *
     * @return Total number of medications loaded
     */
    public int getMedicationCount() {
        return medicationLoader.getMedicationCount();
    }

    /**
     * Validate medication reference in protocol action.
     *
     * Checks if medication referenced in protocol exists in Phase 6 database.
     *
     * @param protocolAction The protocol action to validate
     * @return ValidationResult with status and messages
     */
    public ValidationResult validateMedicationReference(ProtocolAction protocolAction) {
        ValidationResult result = new ValidationResult();

        if (protocolAction == null) {
            result.setValid(false);
            result.addError("Protocol action is null");
            return result;
        }

        Object medObj = protocolAction.getMedication();
        if (medObj == null) {
            result.setValid(true); // Not all actions have medications
            return result;
        }

        String medicationName = null;
        if (medObj instanceof com.cardiofit.flink.models.Medication) {
            com.cardiofit.flink.models.Medication legacyMed =
                (com.cardiofit.flink.models.Medication) medObj;
            medicationName = legacyMed.getName();
        } else if (medObj instanceof Medication) {
            Medication phase6Med = (Medication) medObj;
            medicationName = phase6Med.getGenericName();
        }

        if (medicationName == null || medicationName.trim().isEmpty()) {
            result.setValid(false);
            result.addError("Medication name is null or empty");
            return result;
        }

        Medication enhanced = medicationLoader.getMedicationByName(medicationName);

        if (enhanced == null) {
            result.setValid(false);
            result.addWarning("Medication not found in Phase 6 database: " + medicationName);
            result.addWarning("Consider adding to knowledge-base/medications/");
        } else {
            result.setValid(true);
            result.addInfo("Medication found: " + enhanced.getGenericName() +
                          " (" + enhanced.getMedicationId() + ")");
        }

        return result;
    }

    // ================================================================
    // VALIDATION RESULT CLASS
    // ================================================================

    public static class ValidationResult {
        private boolean valid;
        private final java.util.List<String> errors = new java.util.ArrayList<>();
        private final java.util.List<String> warnings = new java.util.ArrayList<>();
        private final java.util.List<String> info = new java.util.ArrayList<>();

        public boolean isValid() { return valid; }
        public void setValid(boolean valid) { this.valid = valid; }

        public java.util.List<String> getErrors() { return errors; }
        public java.util.List<String> getWarnings() { return warnings; }
        public java.util.List<String> getInfo() { return info; }

        public void addError(String error) { errors.add(error); }
        public void addWarning(String warning) { warnings.add(warning); }
        public void addInfo(String info) { this.info.add(info); }

        public boolean hasErrors() { return !errors.isEmpty(); }
        public boolean hasWarnings() { return !warnings.isEmpty(); }
    }
}
