package com.cardiofit.flink.safety;

import com.cardiofit.flink.models.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;

/**
 * Drug Interaction Checker - Drug-drug interaction detection and assessment
 *
 * Detects and assesses clinical significance of drug-drug interactions:
 * - Major interactions (potentially life-threatening)
 * - Moderate interactions (require monitoring)
 * - Minor interactions (clinically insignificant)
 *
 * Major Interactions Implemented:
 * - Warfarin + NSAIDs → Increased bleeding risk
 * - Warfarin + Antibiotics (ciprofloxacin, metronidazole) → Elevated INR
 * - ACE inhibitors + Potassium-sparing diuretics → Hyperkalemia
 * - Statins + Macrolides → Rhabdomyolysis risk
 * - Beta-blockers + Calcium channel blockers → Bradycardia/hypotension
 * - Digoxin + Diuretics → Hypokalemia-induced toxicity
 *
 * References:
 * - Micromedex Drug Interactions Database
 * - Lexi-Comp Drug Interactions
 * - FDA Drug Safety Communications
 * - Hansten and Horn's Drug Interactions Analysis and Management
 *
 * @author CardioFit Platform - Module 3 Phase 4
 * @version 1.0
 * @since 2025-10-20
 */
public class DrugInteractionChecker implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger logger = LoggerFactory.getLogger(DrugInteractionChecker.class);

    // Drug interaction database: drug1 → drug2 → interaction details
    private static final Map<String, Map<String, DrugInteraction>> INTERACTION_DATABASE = new HashMap<>();

    static {
        initializeInteractionDatabase();
    }

    /**
     * Initialize drug-drug interaction database
     */
    private static void initializeInteractionDatabase() {
        // Warfarin interactions
        Map<String, DrugInteraction> warfarinInteractions = new HashMap<>();

        warfarinInteractions.put("ciprofloxacin", new DrugInteraction(
                "Warfarin + Ciprofloxacin",
                InteractionSeverity.MAJOR,
                "Increased INR and bleeding risk",
                "Fluoroquinolones inhibit warfarin metabolism via CYP450. " +
                        "Monitor INR closely. Consider reducing warfarin dose by 10-20%.",
                Arrays.asList("Monitor INR 2-3 days after starting ciprofloxacin",
                        "Watch for signs of bleeding", "Consider alternative antibiotic if possible"),
                0.8
        ));

        warfarinInteractions.put("metronidazole", new DrugInteraction(
                "Warfarin + Metronidazole",
                InteractionSeverity.MAJOR,
                "Increased INR and bleeding risk",
                "Metronidazole inhibits warfarin metabolism. " +
                        "Monitor INR closely. Consider reducing warfarin dose by 20-30%.",
                Arrays.asList("Check INR within 3-5 days", "Watch for bleeding signs",
                        "Reduce warfarin dose empirically if INR >3.0"),
                0.8
        ));

        warfarinInteractions.put("nsaid", new DrugInteraction(
                "Warfarin + NSAID",
                InteractionSeverity.MAJOR,
                "Increased bleeding risk (antiplatelet effect + anticoagulation)",
                "NSAIDs inhibit platelet function and may cause GI bleeding. " +
                        "Combined with warfarin significantly increases hemorrhage risk.",
                Arrays.asList("Avoid combination if possible", "Use acetaminophen instead",
                        "If necessary: monitor INR, use PPI for GI protection"),
                0.75
        ));

        warfarinInteractions.put("ibuprofen", warfarinInteractions.get("nsaid"));
        warfarinInteractions.put("naproxen", warfarinInteractions.get("nsaid"));

        INTERACTION_DATABASE.put("warfarin", warfarinInteractions);

        // ACE Inhibitor interactions
        Map<String, DrugInteraction> aceInhibitorInteractions = new HashMap<>();

        aceInhibitorInteractions.put("spironolactone", new DrugInteraction(
                "ACE Inhibitor + Spironolactone",
                InteractionSeverity.MAJOR,
                "Hyperkalemia risk",
                "Both medications increase potassium retention. " +
                        "Combined use significantly increases hyperkalemia risk (K+ >5.5 mEq/L).",
                Arrays.asList("Check potassium within 1 week of starting combination",
                        "Monitor renal function", "Avoid potassium supplements",
                        "Consider stopping one agent if K+ >5.0 mEq/L"),
                0.7
        ));

        aceInhibitorInteractions.put("amiloride", aceInhibitorInteractions.get("spironolactone"));
        aceInhibitorInteractions.put("triamterene", aceInhibitorInteractions.get("spironolactone"));

        INTERACTION_DATABASE.put("lisinopril", aceInhibitorInteractions);
        INTERACTION_DATABASE.put("enalapril", aceInhibitorInteractions);
        INTERACTION_DATABASE.put("ramipril", aceInhibitorInteractions);

        // Statin interactions
        Map<String, DrugInteraction> statinInteractions = new HashMap<>();

        statinInteractions.put("clarithromycin", new DrugInteraction(
                "Statin + Macrolide Antibiotic",
                InteractionSeverity.MAJOR,
                "Rhabdomyolysis risk",
                "Macrolides inhibit CYP3A4, increasing statin levels. " +
                        "Risk of myopathy and rhabdomyolysis (muscle breakdown).",
                Arrays.asList("Consider holding statin during macrolide course",
                        "Counsel patient on muscle pain/weakness",
                        "Check CK if muscle symptoms develop"),
                0.65
        ));

        statinInteractions.put("azithromycin", new DrugInteraction(
                "Statin + Azithromycin",
                InteractionSeverity.MODERATE,
                "Mild increase in rhabdomyolysis risk",
                "Azithromycin has minimal CYP3A4 interaction but some risk remains.",
                Arrays.asList("Monitor for muscle symptoms", "Patient counseling on myopathy signs"),
                0.3
        ));

        INTERACTION_DATABASE.put("atorvastatin", statinInteractions);
        INTERACTION_DATABASE.put("simvastatin", statinInteractions);
        INTERACTION_DATABASE.put("lovastatin", statinInteractions);

        // Beta-blocker + Calcium channel blocker interactions
        Map<String, DrugInteraction> betaBlockerInteractions = new HashMap<>();

        betaBlockerInteractions.put("verapamil", new DrugInteraction(
                "Beta-blocker + Verapamil",
                InteractionSeverity.MAJOR,
                "Bradycardia and heart block risk",
                "Both medications slow AV conduction. " +
                        "Combined use increases risk of severe bradycardia and AV block.",
                Arrays.asList("Monitor heart rate and EKG", "Watch for dizziness/syncope",
                        "Consider alternative agents", "Reduce doses if combination necessary"),
                0.7
        ));

        betaBlockerInteractions.put("diltiazem", new DrugInteraction(
                "Beta-blocker + Diltiazem",
                InteractionSeverity.MAJOR,
                "Bradycardia and heart block risk",
                "Negative chronotropic effects are additive.",
                Arrays.asList("Monitor heart rate closely", "Avoid in patients with AV block",
                        "Use with extreme caution"),
                0.7
        ));

        INTERACTION_DATABASE.put("metoprolol", betaBlockerInteractions);
        INTERACTION_DATABASE.put("carvedilol", betaBlockerInteractions);
        INTERACTION_DATABASE.put("atenolol", betaBlockerInteractions);

        // Digoxin interactions
        Map<String, DrugInteraction> digoxinInteractions = new HashMap<>();

        digoxinInteractions.put("furosemide", new DrugInteraction(
                "Digoxin + Loop Diuretic",
                InteractionSeverity.MODERATE,
                "Hypokalemia-induced digoxin toxicity",
                "Diuretics cause potassium loss. Hypokalemia increases digoxin toxicity risk.",
                Arrays.asList("Monitor potassium levels", "Monitor digoxin level",
                        "Consider potassium supplementation", "Watch for arrhythmias"),
                0.5
        ));

        digoxinInteractions.put("hydrochlorothiazide", digoxinInteractions.get("furosemide"));

        INTERACTION_DATABASE.put("digoxin", digoxinInteractions);
    }

    /**
     * Check for drug-drug interactions
     *
     * @param newMedication Clinical action for new medication
     * @param activeMedications Map of active medications (RxNorm code → Medication)
     * @return List of interaction contraindications
     */
    public List<Contraindication> checkInteractions(
            ClinicalAction newMedication,
            Map<String, Medication> activeMedications) {

        List<Contraindication> contraindications = new ArrayList<>();

        if (newMedication == null || newMedication.getMedicationDetails() == null) {
            return contraindications;
        }

        MedicationDetails newMed = newMedication.getMedicationDetails();
        String newMedName = newMed.getName();

        if (newMedName == null || newMedName.trim().isEmpty()) {
            logger.warn("New medication name is null or empty - cannot check interactions");
            return contraindications;
        }

        if (activeMedications == null || activeMedications.isEmpty()) {
            logger.debug("No active medications - no drug-drug interactions");
            return contraindications;
        }

        logger.debug("Checking drug interactions for {} against {} active medications",
                newMedName, activeMedications.size());

        String newMedLower = newMedName.toLowerCase();

        // Check against each active medication
        for (Map.Entry<String, Medication> entry : activeMedications.entrySet()) {
            Medication activeMed = entry.getValue();
            if (activeMed == null || activeMed.getName() == null) {
                continue;
            }

            String activeMedName = activeMed.getName();
            DrugInteraction interaction = findInteraction(newMedLower, activeMedName.toLowerCase());

            if (interaction != null) {
                Contraindication contraindication = createInteractionContraindication(
                        interaction, newMedName, activeMedName);
                contraindications.add(contraindication);

                logger.warn("Drug interaction detected: {} + {} (severity: {})",
                        newMedName, activeMedName, interaction.severity);
            }
        }

        return contraindications;
    }

    /**
     * Find interaction between two medications
     *
     * Bidirectional search in interaction database
     *
     * @param medication1 First medication name (lowercase)
     * @param medication2 Second medication name (lowercase)
     * @return DrugInteraction object if found, null otherwise
     */
    public DrugInteraction findInteraction(String medication1, String medication2) {
        if (medication1 == null || medication2 == null) {
            return null;
        }

        // Check medication1 → medication2
        DrugInteraction interaction = searchInteractionDatabase(medication1, medication2);
        if (interaction != null) {
            return interaction;
        }

        // Check medication2 → medication1 (reverse direction)
        return searchInteractionDatabase(medication2, medication1);
    }

    /**
     * Search interaction database for specific drug pair
     *
     * @param drug1 First drug (lowercase)
     * @param drug2 Second drug (lowercase)
     * @return DrugInteraction if found, null otherwise
     */
    private DrugInteraction searchInteractionDatabase(String drug1, String drug2) {
        for (Map.Entry<String, Map<String, DrugInteraction>> entry : INTERACTION_DATABASE.entrySet()) {
            String primaryDrug = entry.getKey();

            // Check if drug1 matches primary drug
            if (drug1.contains(primaryDrug)) {
                Map<String, DrugInteraction> interactions = entry.getValue();

                // Check if drug2 matches any interacting drug
                for (Map.Entry<String, DrugInteraction> interactionEntry : interactions.entrySet()) {
                    String interactingDrug = interactionEntry.getKey();
                    if (drug2.contains(interactingDrug)) {
                        return interactionEntry.getValue();
                    }
                }
            }
        }

        return null;
    }

    /**
     * Create contraindication object from drug interaction
     *
     * @param interaction DrugInteraction details
     * @param newMedName New medication name
     * @param activeMedName Active medication name
     * @return Contraindication object
     */
    private Contraindication createInteractionContraindication(
            DrugInteraction interaction,
            String newMedName,
            String activeMedName) {

        Contraindication contraindication = new Contraindication(
                Contraindication.ContraindicationType.DRUG_INTERACTION,
                String.format("%s + %s: %s", newMedName, activeMedName, interaction.effect)
        );

        // Map interaction severity to contraindication severity
        contraindication.setSeverity(mapSeverity(interaction.severity));
        contraindication.setFound(true);
        contraindication.setEvidence(String.format(
                "Active medication: %s. %s",
                activeMedName, interaction.mechanism));
        contraindication.setRiskScore(interaction.riskScore);
        contraindication.setClinicalGuidance(formatClinicalGuidance(interaction));

        return contraindication;
    }

    /**
     * Map interaction severity to contraindication severity
     *
     * @param interactionSeverity Interaction severity level
     * @return Contraindication severity
     */
    private Contraindication.Severity mapSeverity(InteractionSeverity interactionSeverity) {
        switch (interactionSeverity) {
            case MAJOR:
                return Contraindication.Severity.ABSOLUTE;
            case MODERATE:
                return Contraindication.Severity.RELATIVE;
            case MINOR:
                return Contraindication.Severity.CAUTION;
            default:
                return Contraindication.Severity.CAUTION;
        }
    }

    /**
     * Format clinical guidance from interaction
     *
     * @param interaction DrugInteraction object
     * @return Formatted clinical guidance string
     */
    private String formatClinicalGuidance(DrugInteraction interaction) {
        StringBuilder guidance = new StringBuilder();
        guidance.append(interaction.mechanism);

        if (interaction.monitoringRecommendations != null && !interaction.monitoringRecommendations.isEmpty()) {
            guidance.append(" Monitoring: ");
            guidance.append(String.join("; ", interaction.monitoringRecommendations));
        }

        return guidance.toString();
    }

    /**
     * Assess clinical significance of interaction
     *
     * @param interaction DrugInteraction object
     * @return Clinical significance assessment string
     */
    public String assessClinicalSignificance(DrugInteraction interaction) {
        if (interaction == null) {
            return "No interaction";
        }

        switch (interaction.severity) {
            case MAJOR:
                return "MAJOR - Potentially life-threatening or capable of causing permanent damage. " +
                        "Avoid combination or use only with intensive monitoring.";
            case MODERATE:
                return "MODERATE - May cause deterioration in patient status. " +
                        "Monitor closely and consider dose adjustments.";
            case MINOR:
                return "MINOR - Limited clinical significance. " +
                        "Monitor but intervention usually not required.";
            default:
                return "Unknown clinical significance";
        }
    }

    /**
     * Drug Interaction definition
     */
    public static class DrugInteraction implements Serializable {
        private static final long serialVersionUID = 1L;

        String interactionName;
        InteractionSeverity severity;
        String effect;
        String mechanism;
        List<String> monitoringRecommendations;
        double riskScore; // 0.0-1.0

        public DrugInteraction(String interactionName, InteractionSeverity severity,
                             String effect, String mechanism,
                             List<String> monitoringRecommendations, double riskScore) {
            this.interactionName = interactionName;
            this.severity = severity;
            this.effect = effect;
            this.mechanism = mechanism;
            this.monitoringRecommendations = monitoringRecommendations;
            this.riskScore = riskScore;
        }

        public InteractionSeverity getSeverity() { return severity; }
        public String getEffect() { return effect; }
        public double getRiskScore() { return riskScore; }
    }

    /**
     * Interaction Severity Enumeration
     */
    public enum InteractionSeverity {
        MAJOR,      // Life-threatening or permanent damage
        MODERATE,   // Deterioration in patient status
        MINOR       // Limited clinical significance
    }

    @Override
    public String toString() {
        return "DrugInteractionChecker{interactionDatabaseSize=" + INTERACTION_DATABASE.size() + "}";
    }
}
