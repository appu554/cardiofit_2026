package com.cardiofit.flink.safety;

import com.cardiofit.flink.models.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Contraindication Checker - Main safety coordinator for clinical actions
 *
 * Orchestrates comprehensive contraindication checking across multiple domains:
 * - Allergy checking with cross-reactivity rules
 * - Drug-drug interaction detection
 * - Renal function-based dosing adjustments
 * - Hepatic function-based dosing adjustments
 *
 * Safety-First Principle: When in doubt, flag as contraindication
 *
 * Architecture:
 * - Delegates to specialized checkers for domain-specific logic
 * - Modifies MedicationDetails in-place for dosing adjustments
 * - Returns comprehensive list of all contraindications found
 *
 * References:
 * - Lexi-Comp Drug Interactions Database
 * - Micromedex Drug Information
 * - FDA Drug Safety Communications
 * - UpToDate Clinical Decision Support
 *
 * @author CardioFit Platform - Module 3 Phase 4
 * @version 1.0
 * @since 2025-10-20
 */
public class ContraindicationChecker implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger logger = LoggerFactory.getLogger(ContraindicationChecker.class);

    // Specialized checkers
    private final AllergyChecker allergyChecker;
    private final DrugInteractionChecker drugInteractionChecker;
    private final RenalDosingAdjuster renalDosingAdjuster;
    private final HepaticDosingAdjuster hepaticDosingAdjuster;

    /**
     * Default constructor - initializes all specialized checkers
     */
    public ContraindicationChecker() {
        this.allergyChecker = new AllergyChecker();
        this.drugInteractionChecker = new DrugInteractionChecker();
        this.renalDosingAdjuster = new RenalDosingAdjuster();
        this.hepaticDosingAdjuster = new HepaticDosingAdjuster();

        logger.info("ContraindicationChecker initialized with all specialized checkers");
    }

    /**
     * Check all clinical actions for contraindications
     *
     * Performs comprehensive safety checking across all domains:
     * 1. Allergy checking (direct + cross-reactivity)
     * 2. Drug-drug interaction detection
     * 3. Renal function contraindications
     * 4. Hepatic function contraindications
     *
     * @param actions List of clinical actions to check
     * @param context Enriched patient context with full clinical state
     * @return List of all contraindications found (empty list if none)
     */
    public List<Contraindication> checkContraindications(
            List<ClinicalAction> actions,
            EnrichedPatientContext context) {

        if (actions == null || actions.isEmpty()) {
            logger.debug("No actions to check for contraindications");
            return new ArrayList<>();
        }

        if (context == null || context.getPatientState() == null) {
            logger.warn("Missing patient context - cannot perform contraindication checking");
            return new ArrayList<>();
        }

        List<Contraindication> allContraindications = new ArrayList<>();
        PatientContextState state = context.getPatientState();

        logger.info("Checking {} clinical actions for contraindications (patient: {})",
                actions.size(), context.getPatientId());

        // Check each action for contraindications
        for (ClinicalAction action : actions) {
            if (action == null) {
                continue;
            }

            // Only check therapeutic (medication) actions
            if (ClinicalAction.ActionType.THERAPEUTIC.equals(action.getActionType())) {
                MedicationDetails medication = action.getMedicationDetails();

                if (medication == null || medication.getName() == null) {
                    logger.warn("Therapeutic action missing medication details - skipping");
                    continue;
                }

                String medicationName = medication.getName();
                logger.debug("Checking contraindications for medication: {}", medicationName);

                // 1. Allergy checking
                List<Contraindication> allergyChecks = allergyChecker.checkAllergies(
                        action, state.getAllergies() != null ? state.getAllergies() : new ArrayList<>());
                allContraindications.addAll(allergyChecks);

                // 2. Drug-drug interaction checking
                List<Contraindication> interactionChecks = drugInteractionChecker.checkInteractions(
                        action, state.getActiveMedications());
                allContraindications.addAll(interactionChecks);

                // 3. Renal function checking
                List<Contraindication> renalChecks = renalDosingAdjuster.checkRenalContraindications(
                        action, state);
                allContraindications.addAll(renalChecks);

                // 4. Hepatic function checking
                List<Contraindication> hepaticChecks = hepaticDosingAdjuster.checkHepaticContraindications(
                        action, state);
                allContraindications.addAll(hepaticChecks);
            }
        }

        logger.info("Contraindication checking complete - found {} contraindications",
                allContraindications.size());

        return allContraindications;
    }

    /**
     * Apply dosing adjustments to clinical actions based on organ function
     *
     * Modifies MedicationDetails in-place:
     * - Adjusts doses for renal impairment (CrCl-based)
     * - Adjusts doses for hepatic impairment (Child-Pugh-based)
     * - Extends dosing intervals where appropriate
     * - Documents adjustment rationale in medication details
     *
     * Note: This method modifies the input actions directly
     *
     * @param actions List of clinical actions to adjust
     * @param context Enriched patient context with lab values
     */
    public void adjustDosing(
            List<ClinicalAction> actions,
            EnrichedPatientContext context) {

        if (actions == null || actions.isEmpty()) {
            logger.debug("No actions to adjust for dosing");
            return;
        }

        if (context == null || context.getPatientState() == null) {
            logger.warn("Missing patient context - cannot perform dosing adjustments");
            return;
        }

        PatientContextState state = context.getPatientState();
        logger.info("Applying dosing adjustments for {} clinical actions (patient: {})",
                actions.size(), context.getPatientId());

        int adjustmentCount = 0;

        // Apply adjustments to each therapeutic action
        for (ClinicalAction action : actions) {
            if (action == null || !ClinicalAction.ActionType.THERAPEUTIC.equals(action.getActionType())) {
                continue;
            }

            MedicationDetails medication = action.getMedicationDetails();
            if (medication == null || medication.getName() == null) {
                continue;
            }

            boolean adjusted = false;

            // 1. Renal dosing adjustments
            if (renalDosingAdjuster.adjustDose(medication, state)) {
                adjusted = true;
                logger.debug("Applied renal dose adjustment for {}", medication.getName());
            }

            // 2. Hepatic dosing adjustments
            if (hepaticDosingAdjuster.adjustDose(medication, state)) {
                adjusted = true;
                logger.debug("Applied hepatic dose adjustment for {}", medication.getName());
            }

            if (adjusted) {
                adjustmentCount++;
            }
        }

        logger.info("Dosing adjustment complete - adjusted {} medications", adjustmentCount);
    }

    /**
     * Comprehensive safety check - combines contraindication checking and dosing adjustment
     *
     * Convenience method that:
     * 1. Checks for contraindications
     * 2. Applies dosing adjustments
     * 3. Returns all contraindications found
     *
     * @param actions List of clinical actions to process
     * @param context Enriched patient context
     * @return List of contraindications found
     */
    public List<Contraindication> performSafetyCheck(
            List<ClinicalAction> actions,
            EnrichedPatientContext context) {

        logger.info("Performing comprehensive safety check for {} actions",
                actions != null ? actions.size() : 0);

        // First, check for contraindications
        List<Contraindication> contraindications = checkContraindications(actions, context);

        // Then, apply dosing adjustments (even if contraindications exist -
        // clinician may override with adjusted dose)
        adjustDosing(actions, context);

        return contraindications;
    }

    /**
     * Get count of absolute contraindications (must not use)
     *
     * @param contraindications List of contraindications to analyze
     * @return Count of absolute contraindications
     */
    public int getAbsoluteContraindicationCount(List<Contraindication> contraindications) {
        if (contraindications == null) {
            return 0;
        }
        return (int) contraindications.stream()
                .filter(c -> c != null && c.isAbsolute())
                .count();
    }

    /**
     * Check if any absolute contraindications exist
     *
     * @param contraindications List of contraindications to check
     * @return true if any absolute contraindications exist
     */
    public boolean hasAbsoluteContraindications(List<Contraindication> contraindications) {
        return getAbsoluteContraindicationCount(contraindications) > 0;
    }

    /**
     * Get highest risk score from contraindications
     *
     * @param contraindications List of contraindications to analyze
     * @return Highest risk score (0.0-1.0), or 0.0 if none
     */
    public double getMaxRiskScore(List<Contraindication> contraindications) {
        if (contraindications == null || contraindications.isEmpty()) {
            return 0.0;
        }
        return contraindications.stream()
                .filter(c -> c != null)
                .mapToDouble(Contraindication::getRiskScore)
                .max()
                .orElse(0.0);
    }

    @Override
    public String toString() {
        return "ContraindicationChecker{" +
                "allergyChecker=" + allergyChecker +
                ", drugInteractionChecker=" + drugInteractionChecker +
                ", renalDosingAdjuster=" + renalDosingAdjuster +
                ", hepaticDosingAdjuster=" + hepaticDosingAdjuster +
                '}';
    }
}
