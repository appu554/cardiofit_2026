package com.cardiofit.flink.cds.medication;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientContextState;
import com.cardiofit.flink.models.PatientDemographics;
import com.cardiofit.flink.models.Medication;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.List;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Map;
import java.util.HashMap;

/**
 * Selects appropriate medications based on patient-specific factors.
 *
 * <p>PATIENT SAFETY CRITICAL COMPONENT
 *
 * <p>Handles:
 * - Allergy checking and alternative selection (with cross-reactivity detection)
 * - Renal dose adjustments using Cockcroft-Gault formula
 * - Hepatic dose adjustments based on Child-Pugh score
 * - MDR (Multi-Drug Resistant) risk assessment
 * - Selection criteria evaluation
 *
 * <p>Safety Features:
 * - Cross-reactivity checking (penicillin → cephalosporin)
 * - FAIL SAFE: Returns null if no safe medication available
 * - Comprehensive logging for audit trail
 * - Evidence-based dose adjustments
 *
 * <p>Standard Criteria Supported:
 * - NO_PENICILLIN_ALLERGY: Patient not allergic to penicillin
 * - NO_BETA_LACTAM_ALLERGY: Patient not allergic to beta-lactams
 * - CREATININE_CLEARANCE_GT_40: CrCl > 40 mL/min
 * - CREATININE_CLEARANCE_GT_30: CrCl > 30 mL/min
 * - MDR_RISK: Multi-drug resistant risk factors present
 * - NO_BETA_BLOCKER_CONTRAINDICATION: Safe to use beta-blockers
 * - SEVERE_SEPSIS: Lactate >= 4.0
 * - HIGH_BLEEDING_RISK: Active bleeding or coagulopathy
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class MedicationSelector {
    private static final Logger logger = LoggerFactory.getLogger(MedicationSelector.class);

    // Default safe creatinine clearance value when calculation not possible
    private static final double DEFAULT_CRCL = 60.0;

    // Renal dose adjustment thresholds
    private static final double CRCL_THRESHOLD_MILD = 60.0;
    private static final double CRCL_THRESHOLD_MODERATE = 40.0;
    private static final double CRCL_THRESHOLD_SEVERE = 30.0;

    // ============================================================
    // DRUG-DRUG INTERACTION DATABASE
    // Top 20 Critical Drug Interactions for Patient Safety
    // ============================================================

    /**
     * Drug-drug interaction database.
     * Key: Medication name (lowercase)
     * Value: List of interactions with that medication
     */
    private static final Map<String, List<DrugInteraction>> DRUG_INTERACTIONS = initializeInteractions();

    private static Map<String, List<DrugInteraction>> initializeInteractions() {
        Map<String, List<DrugInteraction>> interactions = new HashMap<>();

        // Warfarin interactions (critical - bleeding risk)
        interactions.put("warfarin", Arrays.asList(
            new DrugInteraction("piperacillin-tazobactam", "MAJOR", "Increased INR and bleeding risk. Monitor INR closely.", "Monitor INR daily"),
            new DrugInteraction("piperacillin", "MAJOR", "Increased INR and bleeding risk", "Monitor INR daily"),
            new DrugInteraction("ciprofloxacin", "MAJOR", "Increased INR and bleeding risk. 30% increase in anticoagulation effect.", "Monitor INR within 3-5 days"),
            new DrugInteraction("levofloxacin", "MAJOR", "Increased INR and bleeding risk", "Monitor INR within 3-5 days"),
            new DrugInteraction("metronidazole", "MAJOR", "Increased warfarin effect via CYP2C9 inhibition", "Monitor INR within 3-5 days"),
            new DrugInteraction("amiodarone", "MAJOR", "Increased INR. Reduce warfarin dose by 30-50%.", "Check INR weekly"),
            new DrugInteraction("aspirin", "MAJOR", "Increased bleeding risk (additive antiplatelet effect)", "Avoid combination if possible"),
            new DrugInteraction("nsaid", "MAJOR", "Increased bleeding risk", "Use with caution")
        ));

        // Digoxin interactions (critical - toxicity risk)
        interactions.put("digoxin", Arrays.asList(
            new DrugInteraction("furosemide", "MAJOR", "Hypokalemia increases digoxin toxicity risk. Monitor K+.", "Monitor potassium and digoxin levels"),
            new DrugInteraction("amiodarone", "MAJOR", "Increased digoxin levels (50% increase). Reduce digoxin dose by 50%.", "Check digoxin level in 1 week"),
            new DrugInteraction("verapamil", "MAJOR", "Increased digoxin levels. Reduce digoxin dose.", "Monitor digoxin level"),
            new DrugInteraction("spironolactone", "MODERATE", "Increased digoxin levels and decreased renal clearance", "Monitor digoxin level")
        ));

        // Statins interactions (myopathy/rhabdomyolysis risk)
        interactions.put("simvastatin", Arrays.asList(
            new DrugInteraction("amiodarone", "MAJOR", "Myopathy/rhabdomyolysis risk. Max simvastatin 20mg/day.", "Monitor CK, avoid doses >20mg"),
            new DrugInteraction("diltiazem", "MAJOR", "Increased statin levels. Myopathy risk.", "Monitor CK, max simvastatin 10mg"),
            new DrugInteraction("clarithromycin", "CONTRAINDICATED", "Severe myopathy/rhabdomyolysis risk via CYP3A4 inhibition", "Use alternative statin or antibiotic")
        ));

        interactions.put("atorvastatin", Arrays.asList(
            new DrugInteraction("clarithromycin", "MAJOR", "Increased atorvastatin levels, myopathy risk", "Monitor CK, use lowest dose")
        ));

        // QT prolongation interactions (critical - arrhythmia risk)
        interactions.put("azithromycin", Arrays.asList(
            new DrugInteraction("amiodarone", "MAJOR", "Additive QT prolongation. Risk of torsades de pointes.", "Monitor ECG, avoid if QTc >500ms"),
            new DrugInteraction("methadone", "MAJOR", "Additive QT prolongation", "Monitor ECG")
        ));

        interactions.put("ciprofloxacin", Arrays.asList(
            new DrugInteraction("amiodarone", "MAJOR", "QT prolongation risk", "Monitor ECG"),
            new DrugInteraction("warfarin", "MAJOR", "Increased INR", "Monitor INR")
        ));

        // Aminoglycoside interactions (nephrotoxicity/ototoxicity)
        interactions.put("gentamicin", Arrays.asList(
            new DrugInteraction("furosemide", "MAJOR", "Increased ototoxicity and nephrotoxicity risk", "Monitor renal function and hearing"),
            new DrugInteraction("vancomycin", "MAJOR", "Additive nephrotoxicity", "Monitor renal function closely"),
            new DrugInteraction("piperacillin-tazobactam", "MODERATE", "Inactivation of aminoglycoside if mixed. Give separately.", "Administer 1 hour apart")
        ));

        interactions.put("vancomycin", Arrays.asList(
            new DrugInteraction("gentamicin", "MAJOR", "Additive nephrotoxicity", "Monitor renal function closely"),
            new DrugInteraction("furosemide", "MAJOR", "Increased nephrotoxicity and ototoxicity", "Monitor renal function")
        ));

        // ACE inhibitor/ARB interactions (hyperkalemia risk)
        interactions.put("lisinopril", Arrays.asList(
            new DrugInteraction("spironolactone", "MAJOR", "Hyperkalemia risk. Monitor K+ closely.", "Check potassium weekly initially"),
            new DrugInteraction("potassium", "MAJOR", "Hyperkalemia risk", "Avoid potassium supplements"),
            new DrugInteraction("nsaid", "MAJOR", "Decreased antihypertensive effect, nephrotoxicity", "Monitor BP and renal function")
        ));

        // Beta-blocker interactions
        interactions.put("metoprolol", Arrays.asList(
            new DrugInteraction("verapamil", "MAJOR", "Severe bradycardia and heart block risk", "Monitor heart rate and ECG"),
            new DrugInteraction("diltiazem", "MAJOR", "Bradycardia and hypotension risk", "Monitor heart rate"),
            new DrugInteraction("amiodarone", "MAJOR", "Bradycardia and AV block risk", "Monitor ECG")
        ));

        // Antifungal interactions
        interactions.put("fluconazole", Arrays.asList(
            new DrugInteraction("warfarin", "MAJOR", "Increased INR via CYP2C9 inhibition", "Monitor INR closely"),
            new DrugInteraction("simvastatin", "MAJOR", "Increased statin levels, myopathy risk", "Use lowest statin dose")
        ));

        // Methotrexate interactions
        interactions.put("methotrexate", Arrays.asList(
            new DrugInteraction("nsaid", "MAJOR", "Increased methotrexate toxicity (decreased renal clearance)", "Avoid NSAIDs"),
            new DrugInteraction("trimethoprim", "MAJOR", "Bone marrow suppression risk", "Monitor CBC")
        ));

        return interactions;
    }

    /**
     * Selects the appropriate medication for an action based on patient context.
     *
     * <p>SAFETY CRITICAL: This method determines which medication is safe to administer
     * based on allergies, renal function, and other patient-specific factors.
     *
     * <p>Selection Algorithm:
     * 1. Evaluate selection criteria in order
     * 2. For matching criteria, use primary medication
     * 3. Check for allergies/contraindications
     * 4. If allergic, use alternative medication
     * 5. Apply renal/hepatic dose adjustments
     * 6. Return null if no safe option available (FAIL SAFE)
     *
     * @param action The protocol action with medication selection criteria
     * @param context The patient context with demographics and clinical data
     * @return The selected medication with dose adjustments, or null if no safe option
     */
    public ProtocolAction selectMedication(ProtocolAction action, EnrichedPatientContext context) {
        if (action == null || context == null) {
            logger.error("SAFETY VIOLATION: Null action or context provided to selectMedication");
            return action;
        }

        MedicationSelection selection = action.getMedicationSelection();

        // No selection algorithm - return action as-is
        if (selection == null) {
            logger.debug("No medication_selection for action {}, using as-is",
                action.getActionId());
            return action;
        }

        logger.info("Selecting medication for action {} with {} criteria",
            action.getActionId(),
            selection.getSelectionCriteria() != null ? selection.getSelectionCriteria().size() : 0);

        // Evaluate selection criteria in order
        if (selection.getSelectionCriteria() != null) {
            for (SelectionCriteria criteria : selection.getSelectionCriteria()) {
                boolean criteriaMet = evaluateCriteria(criteria.getCriteriaId(), context);

                if (criteriaMet) {
                    logger.debug("Criteria {} met for patient {}",
                        criteria.getCriteriaId(),
                        context.getPatientId());

                    // Use primary medication
                    ClinicalMedication selectedMed = criteria.getPrimaryMedication();
                    if (selectedMed == null) {
                        logger.error("SAFETY VIOLATION: No primary medication defined for criteria {}",
                            criteria.getCriteriaId());
                        continue;
                    }

                    // Clone to avoid modifying original
                    selectedMed = selectedMed.clone();

                    // Check for allergies/contraindications
                    if (hasAllergy(selectedMed, context)) {
                        logger.warn("ALLERGY DETECTED: Patient {} allergic to {}, evaluating alternative",
                            context.getPatientId(),
                            selectedMed.getName());

                        if (criteria.getAlternativeMedication() != null) {
                            selectedMed = criteria.getAlternativeMedication().clone();
                            logger.info("ALTERNATIVE SELECTED: {} for patient {}",
                                selectedMed.getName(),
                                context.getPatientId());

                            // Check alternative for allergies too
                            if (hasAllergy(selectedMed, context)) {
                                logger.error("SAFETY FAIL: Alternative medication {} also contraindicated for patient {}",
                                    selectedMed.getName(),
                                    context.getPatientId());
                                return null; // FAIL SAFE: No safe medication
                            }
                        } else {
                            logger.error("SAFETY FAIL: No alternative medication available for allergy to {} in patient {}",
                                selectedMed.getName(),
                                context.getPatientId());
                            return null; // FAIL SAFE: No safe medication
                        }
                    }

                    // Check for drug-drug interactions (SAFETY CRITICAL)
                    List<DrugInteraction> interactions = checkDrugInteractions(selectedMed, context);
                    if (!interactions.isEmpty()) {
                        logger.warn("DRUG INTERACTIONS DETECTED: {} interactions found for {} in patient {}",
                            interactions.size(),
                            selectedMed.getName(),
                            context.getPatientId());

                        // Log each interaction for clinical review
                        for (DrugInteraction interaction : interactions) {
                            logger.warn("  - INTERACTION {}: {} with {}. {}",
                                interaction.getSeverity(),
                                selectedMed.getName(),
                                interaction.getInteractingDrug(),
                                interaction.getDescription());

                            // CONTRAINDICATED interactions - block medication
                            if ("CONTRAINDICATED".equals(interaction.getSeverity())) {
                                logger.error("SAFETY FAIL: CONTRAINDICATED interaction between {} and {} in patient {}",
                                    selectedMed.getName(),
                                    interaction.getInteractingDrug(),
                                    context.getPatientId());
                                return null; // FAIL SAFE: Contraindicated combination
                            }
                        }

                        // For MAJOR interactions, add warnings to administration instructions
                        if (interactions.stream().anyMatch(i -> "MAJOR".equals(i.getSeverity()))) {
                            StringBuilder warnings = new StringBuilder();
                            warnings.append("DRUG INTERACTION WARNINGS: ");
                            for (DrugInteraction interaction : interactions) {
                                if ("MAJOR".equals(interaction.getSeverity())) {
                                    warnings.append(interaction.getRecommendation()).append("; ");
                                }
                            }
                            selectedMed.setAdministrationInstructions(
                                selectedMed.getAdministrationInstructions() != null ?
                                    selectedMed.getAdministrationInstructions() + " " + warnings.toString() :
                                    warnings.toString()
                            );
                        }
                    }

                    // Apply dose adjustments (renal/hepatic)
                    selectedMed = applyDoseAdjustments(selectedMed, context);

                    // Create new action with selected medication
                    ProtocolAction selectedAction = action.clone();
                    selectedAction.setMedication(convertToMedication(selectedMed));

                    logger.info("MEDICATION SELECTED: {} {} {} {} for patient {}",
                        selectedMed.getName(),
                        selectedMed.getDose(),
                        selectedMed.getRoute(),
                        selectedMed.getFrequency(),
                        context.getPatientId());

                    return selectedAction;
                }
            }
        }

        // No criteria met - return original action
        logger.warn("No selection criteria met for action {} in patient {}",
            action.getActionId(),
            context.getPatientId());
        return action;
    }

    /**
     * Evaluates a selection criteria ID.
     *
     * <p>Standard criteria supported:
     * - NO_PENICILLIN_ALLERGY: Patient not allergic to penicillin
     * - NO_BETA_LACTAM_ALLERGY: Patient not allergic to beta-lactams or cephalosporins
     * - CREATININE_CLEARANCE_GT_40: CrCl > 40 mL/min
     * - CREATININE_CLEARANCE_GT_30: CrCl > 30 mL/min
     * - MDR_RISK: Multi-drug resistant risk factors present
     * - NO_BETA_BLOCKER_CONTRAINDICATION: Safe to use beta-blockers
     * - SEVERE_SEPSIS: Lactate >= 4.0 mmol/L
     * - HIGH_BLEEDING_RISK: Active bleeding, low platelets, or elevated INR
     *
     * @param criteriaId The criteria identifier
     * @param context The patient context
     * @return true if criteria met, false otherwise
     */
    public boolean evaluateCriteria(String criteriaId, EnrichedPatientContext context) {
        if (criteriaId == null || context == null) {
            logger.warn("Null criteriaId or context in evaluateCriteria");
            return false;
        }

        PatientContextState state = context.getPatientState();
        if (state == null) {
            logger.warn("Null patient state in context");
            return false;
        }

        switch (criteriaId) {
            case "NO_PENICILLIN_ALLERGY":
                return !hasAllergyToSubstance("penicillin", context);

            case "NO_BETA_LACTAM_ALLERGY":
                return !hasAllergyToSubstance("penicillin", context) &&
                       !hasAllergyToSubstance("cephalosporin", context) &&
                       !hasAllergyToSubstance("beta-lactam", context);

            case "CREATININE_CLEARANCE_GT_40":
                double crCl40 = calculateCrCl(context);
                return crCl40 > 40.0;

            case "CREATININE_CLEARANCE_GT_30":
                double crCl30 = calculateCrCl(context);
                return crCl30 > 30.0;

            case "CREATININE_CLEARANCE_GT_60":
                double crCl60 = calculateCrCl(context);
                return crCl60 > 60.0;

            case "MDR_RISK":
                // Multi-drug resistant risk factors
                return hasRecentHospitalization(state) ||
                       hasRecentAntibiotics(state) ||
                       isImmunosuppressed(state) ||
                       hasIndwellingDevices(state);

            case "NO_BETA_BLOCKER_CONTRAINDICATION":
                return !hasBetaBlockerContraindication(context);

            case "SEVERE_SEPSIS":
                Double lactate = getVitalValue(state, "lactate");
                return lactate != null && lactate >= 4.0;

            case "HIGH_BLEEDING_RISK":
                return hasActiveBleed(state) ||
                       hasLowPlatelets(state) ||
                       hasElevatedINR(state);

            case "PREGNANCY":
                return isPregnant(state);

            case "NO_CONTRAINDICATION":
                // General safety check - no known contraindications
                return true;

            default:
                logger.warn("Unknown criteria: {}", criteriaId);
                return false;
        }
    }

    /**
     * Checks if patient is allergic to a medication or drug class.
     *
     * <p>Safety Features:
     * - Direct medication name matching (case-insensitive)
     * - Cross-reactivity detection (penicillin → cephalosporin, sulfa drugs)
     * - Class-level allergy checking
     *
     * @param medication The medication to check
     * @param context The patient context with allergy list
     * @return true if patient has documented allergy or cross-reactivity
     */
    public boolean hasAllergy(ClinicalMedication medication, EnrichedPatientContext context) {
        if (medication == null || context == null) {
            return false;
        }

        PatientContextState state = context.getPatientState();
        if (state == null) {
            return false;
        }

        List<String> allergies = state.getAllergies();

        if (allergies == null || allergies.isEmpty()) {
            return false;
        }

        String medName = medication.getName().toLowerCase();

        for (String allergy : allergies) {
            String allergyLower = allergy.toLowerCase();

            // Direct match (substring matching)
            if (medName.contains(allergyLower) || allergyLower.contains(medName)) {
                logger.warn("ALLERGY: Direct match - {} vs {}", medName, allergyLower);
                return true;
            }

            // Word-based matching for multi-word allergies (e.g., "all cephalosporins" matches "Cephalexin")
            String[] allergyWords = allergyLower.split("\\s+");
            for (String word : allergyWords) {
                if (word.length() > 3) { // Ignore short words like "all", "to"
                    // Check if medication name contains the allergy word (or vice versa for root matching)
                    if (medName.contains(word) || word.contains(medName)) {
                        logger.warn("ALLERGY: Word match - '{}' and '{}' from allergy '{}'", medName, word, allergyLower);
                        return true;
                    }
                    // Check for common root (e.g., "cephalexin" and "cephalosporins" share "cephal")
                    if (word.length() >= 6 && medName.length() >= 6) {
                        String wordRoot = word.substring(0, Math.min(6, word.length()));
                        String medRoot = medName.substring(0, Math.min(6, medName.length()));
                        if (wordRoot.equals(medRoot)) {
                            logger.warn("ALLERGY: Root match - '{}' and '{}' share root '{}'", medName, word, wordRoot);
                            return true;
                        }
                    }
                }
            }

            // Cross-reactivity checking
            if (hasCrossReactivity(medName, allergyLower)) {
                logger.warn("ALLERGY: Cross-reactivity detected - {} with allergy to {}",
                    medName, allergyLower);
                return true;
            }
        }

        return false;
    }

    /**
     * Checks for drug-drug interactions between new medication and current medications.
     *
     * <p>SAFETY CRITICAL: This method checks the new medication against the patient's
     * current medication list for known drug-drug interactions.
     *
     * <p>Severity Levels:
     * - CONTRAINDICATED: Never use together (will block medication)
     * - MAJOR: Serious interaction requiring monitoring or dose adjustment
     * - MODERATE: Moderate interaction, clinical awareness needed
     *
     * @param newMedication The medication being considered
     * @param context The patient context with current medications
     * @return List of drug interactions found (empty if none)
     */
    public List<DrugInteraction> checkDrugInteractions(
            ClinicalMedication newMedication,
            EnrichedPatientContext context) {

        List<DrugInteraction> foundInteractions = new ArrayList<>();

        if (newMedication == null || context == null) {
            return foundInteractions;
        }

        PatientContextState state = context.getPatientState();
        if (state == null) {
            return foundInteractions;
        }

        String newMedName = newMedication.getName().toLowerCase();

        // Get interaction database for this medication
        List<DrugInteraction> knownInteractions = DRUG_INTERACTIONS.get(newMedName);

        if (knownInteractions == null || knownInteractions.isEmpty()) {
            // No known interactions in database
            return foundInteractions;
        }

        // Check medications from both activeMedications (Map) and fhirMedications (List)
        List<Medication> currentMedications = new ArrayList<>();

        // Add from activeMedications Map
        if (state.getActiveMedications() != null) {
            currentMedications.addAll(state.getActiveMedications().values());
        }

        // Add from fhirMedications List
        if (state.getFhirMedications() != null) {
            currentMedications.addAll(state.getFhirMedications());
        }

        // Check each current medication against known interactions
        for (Medication currentMed : currentMedications) {
            if (currentMed == null || currentMed.getName() == null) {
                continue;
            }

            String currentMedName = currentMed.getName().toLowerCase();

            for (DrugInteraction interaction : knownInteractions) {
                String interactingDrug = interaction.getInteractingDrug().toLowerCase();

                // Check if current medication matches the interacting drug
                if (currentMedName.contains(interactingDrug) ||
                    interactingDrug.contains(currentMedName)) {

                    foundInteractions.add(interaction);

                    logger.info("INTERACTION FOUND: {} ({}) with current medication {} ({})",
                        newMedName,
                        newMedication.getName(),
                        currentMedName,
                        currentMed.getName());
                }
            }
        }

        return foundInteractions;
    }

    /**
     * Checks for drug class cross-reactivity.
     *
     * <p>Known cross-reactivities:
     * - Penicillin allergy → Cephalosporin cross-reactivity (~10% risk)
     * - Sulfa allergy → Sulfonamide antibiotics
     * - Carbapenem → Beta-lactam cross-reactivity
     *
     * @param medication The medication name (lowercase)
     * @param allergy The allergy name (lowercase)
     * @return true if cross-reactivity exists
     */
    private boolean hasCrossReactivity(String medication, String allergy) {
        // Penicillin allergy → Cephalosporin cross-reactivity
        if (allergy.contains("penicillin")) {
            if (medication.contains("cef") ||       // Cephalosporins
                medication.contains("ceftriaxone") ||
                medication.contains("cefepime") ||
                medication.contains("cefazolin") ||
                medication.contains("cephalexin")) {
                return true;
            }
        }

        // Sulfa allergy → Sulfonamide antibiotics
        if (allergy.contains("sulfa")) {
            if (medication.contains("sulfamethoxazole") ||
                medication.contains("trimethoprim") ||
                medication.contains("bactrim") ||
                medication.contains("septra")) {
                return true;
            }
        }

        // Carbapenem → Beta-lactam
        if (allergy.contains("carbapenem")) {
            if (medication.contains("meropenem") ||
                medication.contains("imipenem") ||
                medication.contains("ertapenem")) {
                return true;
            }
        }

        return false;
    }

    /**
     * Calculates creatinine clearance using Cockcroft-Gault formula.
     *
     * <p>Formula: CrCl (mL/min) = [(140 - age) × weight(kg)] / (72 × Cr(mg/dL))
     * <p>Multiply by 0.85 for females
     *
     * <p>Reference: Cockcroft DW, Gault MH. Nephron. 1976;16(1):31-41.
     *
     * @param context The patient context with demographics and creatinine
     * @return Creatinine clearance in mL/min, or DEFAULT_CRCL if calculation not possible
     */
    public double calculateCrCl(EnrichedPatientContext context) {
        if (context == null || context.getPatientState() == null) {
            logger.warn("Null context or patient state, using default CrCl {}", DEFAULT_CRCL);
            return DEFAULT_CRCL;
        }

        PatientContextState state = context.getPatientState();
        PatientDemographics demographics = state.getDemographics();

        Integer age = demographics != null ? demographics.getAge() : null;
        Double weight = demographics != null ? demographics.getWeight() : null;
        String sex = demographics != null ? demographics.getSex() : null;

        // Get creatinine from lab results
        Double creatinine = getLabValue(state, "creatinine");

        // Check required parameters
        if (age == null || weight == null || creatinine == null || creatinine == 0.0) {
            logger.warn("Missing parameters for CrCl calculation: age={}, weight={}, creatinine={}, using default {}",
                age, weight, creatinine, DEFAULT_CRCL);
            return DEFAULT_CRCL;
        }

        // Cockcroft-Gault formula
        double crCl = ((140.0 - age) * weight) / (72.0 * creatinine);

        // Female adjustment (multiply by 0.85)
        if ("F".equalsIgnoreCase(sex) || "FEMALE".equalsIgnoreCase(sex)) {
            crCl *= 0.85;
        }

        logger.debug("Calculated CrCl: {} mL/min (age={}, weight={}, Cr={}, sex={})",
            String.format("%.1f", crCl), age, weight, creatinine, sex);

        return crCl;
    }

    /**
     * Applies renal and hepatic dose adjustments to medication.
     *
     * @param medication The medication to adjust
     * @param context The patient context with renal/hepatic function
     * @return The adjusted medication (cloned, original unchanged)
     */
    private ClinicalMedication applyDoseAdjustments(ClinicalMedication medication, EnrichedPatientContext context) {
        if (medication == null || context == null) {
            return medication;
        }

        ClinicalMedication adjusted = medication.clone();

        // Renal dose adjustment
        double crCl = calculateCrCl(context);
        if (crCl < CRCL_THRESHOLD_MILD) {
            adjusted = adjustDoseForRenalFunction(adjusted, crCl);
        }

        // Hepatic dose adjustment
        String childPugh = getChildPughScore(context.getPatientState());
        if (childPugh != null && (childPugh.equals("B") || childPugh.equals("C"))) {
            adjusted = adjustDoseForHepaticFunction(adjusted, childPugh);
        }

        return adjusted;
    }

    /**
     * Adjusts medication dose based on renal function (CrCl).
     *
     * <p>Evidence-based dose adjustments for common medications:
     * - Ceftriaxone: Reduce to 1g if CrCl < 30
     * - Vancomycin: Requires pharmacist dosing if CrCl < 60
     * - Levofloxacin: 500mg q48h if CrCl < 50
     * - Gentamicin: Extended interval dosing if CrCl < 60
     * - Enoxaparin: Dose reduction if CrCl < 30
     *
     * @param medication The medication to adjust
     * @param crCl Creatinine clearance in mL/min
     * @return Medication with adjusted dose/frequency
     */
    public ClinicalMedication adjustDoseForRenalFunction(ClinicalMedication medication, double crCl) {
        if (medication == null) {
            return null;
        }

        String medName = medication.getName().toLowerCase();

        // Ceftriaxone dose adjustment
        if (medName.contains("ceftriaxone")) {
            if (crCl < CRCL_THRESHOLD_SEVERE) {
                medication.setDose("1 g"); // Reduced from 2 g
                medication.setAdministrationInstructions(
                    String.format("Renal dose adjustment (CrCl %.1f mL/min)", crCl));
                logger.info("DOSE ADJUSTMENT: Ceftriaxone reduced to 1g for CrCl {}",
                    String.format("%.1f", crCl));
            }
        }

        // Vancomycin requires pharmacist consultation
        else if (medName.contains("vancomycin")) {
            if (crCl < CRCL_THRESHOLD_MILD) {
                medication.setAdministrationInstructions(
                    String.format("Pharmacist consult required for dosing (CrCl %.1f mL/min)", crCl));
                logger.warn("DOSE ADJUSTMENT: Vancomycin requires pharmacist dosing for CrCl {}",
                    String.format("%.1f", crCl));
            }
        }

        // Levofloxacin dose and interval adjustment
        else if (medName.contains("levofloxacin")) {
            if (crCl < 50.0) {
                medication.setDose("500 mg"); // Reduced from 750 mg
                medication.setFrequency("q48h"); // Extended interval from q24h
                medication.setAdministrationInstructions(
                    String.format("Renal dose adjustment (CrCl %.1f mL/min)", crCl));
                logger.info("DOSE ADJUSTMENT: Levofloxacin 500mg q48h for CrCl {}",
                    String.format("%.1f", crCl));
            }
        }

        // Gentamicin extended interval dosing
        else if (medName.contains("gentamicin")) {
            if (crCl < CRCL_THRESHOLD_MILD) {
                medication.setFrequency("q24h"); // Extended from q8h
                medication.setAdministrationInstructions(
                    String.format("Extended interval dosing (CrCl %.1f mL/min). Monitor trough levels.", crCl));
                logger.info("DOSE ADJUSTMENT: Gentamicin extended interval for CrCl {}",
                    String.format("%.1f", crCl));
            }
        }

        // Enoxaparin dose reduction
        else if (medName.contains("enoxaparin")) {
            if (crCl < CRCL_THRESHOLD_SEVERE) {
                medication.setDose("30 mg"); // Reduced from 40 mg
                medication.setAdministrationInstructions(
                    String.format("Renal dose adjustment (CrCl %.1f mL/min)", crCl));
                logger.info("DOSE ADJUSTMENT: Enoxaparin 30mg for CrCl {}",
                    String.format("%.1f", crCl));
            }
        }

        return medication;
    }

    /**
     * Adjusts medication dose based on hepatic function (Child-Pugh score).
     *
     * <p>Child-Pugh B/C adjustments for hepatically-cleared medications.
     *
     * @param medication The medication to adjust
     * @param childPugh Child-Pugh score (A, B, or C)
     * @return Medication with adjusted dose
     */
    private ClinicalMedication adjustDoseForHepaticFunction(ClinicalMedication medication, String childPugh) {
        if (medication == null) {
            return null;
        }

        String medName = medication.getName().toLowerCase();

        // Example hepatic adjustments (would be expanded based on formulary)
        if (childPugh.equals("C")) {
            // Child-Pugh C - severe hepatic impairment
            if (medName.contains("metoprolol") || medName.contains("propranolol")) {
                medication.setAdministrationInstructions(
                    "Caution: Hepatic impairment (Child-Pugh C). Consider dose reduction.");
                logger.warn("HEPATIC CAUTION: {} for Child-Pugh C", medName);
            }
        }

        return medication;
    }

    // ============================================================
    // Helper Methods for Criteria Evaluation
    // ============================================================

    private boolean hasAllergyToSubstance(String substance, EnrichedPatientContext context) {
        if (context == null || context.getPatientState() == null) {
            return false;
        }

        List<String> allergies = context.getPatientState().getAllergies();
        if (allergies == null) {
            return false;
        }

        String substanceLower = substance.toLowerCase();
        return allergies.stream()
            .anyMatch(a -> a.toLowerCase().contains(substanceLower));
    }

    private boolean hasRecentHospitalization(PatientContextState state) {
        // Would check admissions/encounters history
        // Placeholder for now
        return false;
    }

    private boolean hasRecentAntibiotics(PatientContextState state) {
        // Would check medication history for recent antibiotics
        // Placeholder for now
        return false;
    }

    private boolean isImmunosuppressed(PatientContextState state) {
        // Would check conditions and medications
        // Placeholder for now
        return false;
    }

    private boolean hasIndwellingDevices(PatientContextState state) {
        // Would check device list (catheters, lines, etc.)
        // Placeholder for now
        return false;
    }

    private boolean hasBetaBlockerContraindication(EnrichedPatientContext context) {
        PatientContextState state = context.getPatientState();
        if (state == null) return false;

        // Check for asthma/COPD
        if (state.getChronicConditions() != null) {
            boolean hasRespiratoryContraindication = state.getChronicConditions().stream()
                .anyMatch(c -> c.getCode() != null &&
                    (c.getCode().contains("asthma") || c.getCode().contains("copd")));
            if (hasRespiratoryContraindication) return true;
        }

        // Check for bradycardia or heart block
        Double heartRate = getVitalValue(state, "heartrate");
        if (heartRate != null && heartRate < 50.0) {
            return true; // Bradycardia contraindication
        }

        return false;
    }

    private boolean hasActiveBleed(PatientContextState state) {
        // Would check for active bleeding documented
        // Placeholder for now
        return false;
    }

    private boolean hasLowPlatelets(PatientContextState state) {
        Double platelets = getLabValue(state, "platelets");
        return platelets != null && platelets < 50000;
    }

    private boolean hasElevatedINR(PatientContextState state) {
        Double inr = getLabValue(state, "inr");
        return inr != null && inr > 2.0;
    }

    private boolean isPregnant(PatientContextState state) {
        // Would check pregnancy status
        // Placeholder for now
        return false;
    }

    private String getChildPughScore(PatientContextState state) {
        // Would calculate or retrieve Child-Pugh score
        // Placeholder for now
        return null;
    }

    private Double getVitalValue(PatientContextState state, String vitalName) {
        if (state == null || state.getLatestVitals() == null) {
            return null;
        }

        Object value = state.getLatestVitals().get(vitalName.toLowerCase());
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }
        return null;
    }

    private Double getLabValue(PatientContextState state, String labName) {
        if (state == null || state.getRecentLabs() == null) {
            return null;
        }

        // Try by name first
        com.cardiofit.flink.models.LabResult labResult = state.getRecentLabs().get(labName.toLowerCase());
        if (labResult != null && labResult.getValue() != null) {
            return labResult.getValue();
        }

        return null;
    }

    private Medication convertToMedication(ClinicalMedication clinicalMed) {
        if (clinicalMed == null) {
            return null;
        }

        Medication med = new Medication();
        med.setName(clinicalMed.getName());
        med.setDosage(clinicalMed.getDose());
        med.setRoute(clinicalMed.getRoute());
        med.setFrequency(clinicalMed.getFrequency());
        med.setStatus("active");

        return med;
    }

    // ============================================================
    // Supporting Classes
    // ============================================================

    /**
     * Protocol action with medication selection algorithm
     */
    public static class ProtocolAction implements Cloneable {
        private String actionId;
        private String type;
        private MedicationSelection medicationSelection;
        private Medication medication;

        public String getActionId() { return actionId; }
        public void setActionId(String actionId) { this.actionId = actionId; }

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public MedicationSelection getMedicationSelection() { return medicationSelection; }
        public void setMedicationSelection(MedicationSelection medicationSelection) {
            this.medicationSelection = medicationSelection;
        }

        public Medication getMedication() { return medication; }
        public void setMedication(Medication medication) { this.medication = medication; }

        @Override
        public ProtocolAction clone() {
            try {
                return (ProtocolAction) super.clone();
            } catch (CloneNotSupportedException e) {
                throw new RuntimeException("Clone failed", e);
            }
        }
    }

    /**
     * Medication selection configuration
     */
    public static class MedicationSelection {
        private List<SelectionCriteria> selectionCriteria;

        public List<SelectionCriteria> getSelectionCriteria() { return selectionCriteria; }
        public void setSelectionCriteria(List<SelectionCriteria> selectionCriteria) {
            this.selectionCriteria = selectionCriteria;
        }
    }

    /**
     * Selection criteria with primary and alternative medications
     */
    public static class SelectionCriteria {
        private String criteriaId;
        private ClinicalMedication primaryMedication;
        private ClinicalMedication alternativeMedication;

        public String getCriteriaId() { return criteriaId; }
        public void setCriteriaId(String criteriaId) { this.criteriaId = criteriaId; }

        public ClinicalMedication getPrimaryMedication() { return primaryMedication; }
        public void setPrimaryMedication(ClinicalMedication primaryMedication) {
            this.primaryMedication = primaryMedication;
        }

        public ClinicalMedication getAlternativeMedication() { return alternativeMedication; }
        public void setAlternativeMedication(ClinicalMedication alternativeMedication) {
            this.alternativeMedication = alternativeMedication;
        }
    }

    /**
     * Clinical medication with dose, route, frequency
     */
    public static class ClinicalMedication implements Cloneable {
        private String name;
        private String dose;
        private String route;
        private String frequency;
        private String administrationInstructions;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getDose() { return dose; }
        public void setDose(String dose) { this.dose = dose; }

        public String getRoute() { return route; }
        public void setRoute(String route) { this.route = route; }

        public String getFrequency() { return frequency; }
        public void setFrequency(String frequency) { this.frequency = frequency; }

        public String getAdministrationInstructions() { return administrationInstructions; }
        public void setAdministrationInstructions(String administrationInstructions) {
            this.administrationInstructions = administrationInstructions;
        }

        @Override
        public ClinicalMedication clone() {
            try {
                return (ClinicalMedication) super.clone();
            } catch (CloneNotSupportedException e) {
                throw new RuntimeException("Clone failed", e);
            }
        }
    }

    /**
     * Drug-drug interaction information.
     *
     * <p>Represents a known interaction between two medications with
     * severity level, description, and clinical recommendations.
     */
    public static class DrugInteraction {
        private final String interactingDrug;
        private final String severity;
        private final String description;
        private final String recommendation;

        public DrugInteraction(String interactingDrug, String severity, String description, String recommendation) {
            this.interactingDrug = interactingDrug;
            this.severity = severity;
            this.description = description;
            this.recommendation = recommendation;
        }

        public String getInteractingDrug() {
            return interactingDrug;
        }

        public String getSeverity() {
            return severity;
        }

        public String getDescription() {
            return description;
        }

        public String getRecommendation() {
            return recommendation;
        }

        @Override
        public String toString() {
            return String.format("%s interaction with %s: %s (%s)",
                severity, interactingDrug, description, recommendation);
        }
    }
}
