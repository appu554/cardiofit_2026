package com.cardiofit.flink.safety;

import com.cardiofit.flink.models.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;

/**
 * Hepatic Dosing Adjuster - Child-Pugh score-based medication dosing
 *
 * Performs:
 * - Child-Pugh score calculation for hepatic function assessment
 * - Medication dose adjustments based on hepatic impairment
 * - Hepatotoxic medication flagging
 * - Contraindication detection for severe hepatic impairment
 *
 * Child-Pugh Scoring System:
 * Parameters (each scored 1-3 points):
 * - Total bilirubin: <2 mg/dL (1), 2-3 (2), >3 (3)
 * - Albumin: >3.5 g/dL (1), 2.8-3.5 (2), <2.8 (3)
 * - INR: <1.7 (1), 1.7-2.3 (2), >2.3 (3)
 * - Ascites: None (1), Mild (2), Moderate-Severe (3)
 * - Encephalopathy: None (1), Grade 1-2 (2), Grade 3-4 (3)
 *
 * Classification:
 * - Class A (5-6 points): Well-compensated disease
 * - Class B (7-9 points): Significant functional compromise
 * - Class C (10-15 points): Decompensated disease
 *
 * References:
 * - Pugh RN, et al. Transection of the oesophagus for bleeding oesophageal varices. Br J Surg. 1973
 * - AASLD Practice Guidelines on Hepatic Encephalopathy
 * - FDA Guidance for Industry: Pharmacokinetics in Patients with Impaired Hepatic Function
 * - Verbeeck RK. Pharmacokinetics and dosage adjustment in patients with hepatic dysfunction. Eur J Clin Pharmacol. 2008
 *
 * @author CardioFit Platform - Module 3 Phase 4
 * @version 1.0
 * @since 2025-10-20
 */
public class HepaticDosingAdjuster implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger logger = LoggerFactory.getLogger(HepaticDosingAdjuster.class);

    // Hepatic dosing guidelines database
    private static final Map<String, HepaticDosingGuideline> HEPATIC_DOSING_GUIDELINES = new HashMap<>();

    // Known hepatotoxic medications
    private static final Set<String> HEPATOTOXIC_MEDICATIONS = new HashSet<>();

    static {
        initializeHepaticDosingGuidelines();
        initializeHepatotoxicMedications();
    }

    /**
     * Initialize hepatic dosing guidelines for common medications
     */
    private static void initializeHepaticDosingGuidelines() {
        // Acetaminophen - hepatotoxic, especially in liver disease
        HEPATIC_DOSING_GUIDELINES.put("acetaminophen", new HepaticDosingGuideline(
                "acetaminophen",
                10,  // Contraindicated in Child-Pugh C (>9)
                Arrays.asList(
                        new HepaticDoseAdjustment("A", 1.0, "Standard dosing (max 3g/day)", "Use caution, monitor LFTs"),
                        new HepaticDoseAdjustment("B", 0.5, "Reduce to 2g/day max", "High risk of hepatotoxicity"),
                        new HepaticDoseAdjustment("C", 0.0, "Avoid use", "Contraindicated in decompensated cirrhosis")
                ),
                Contraindication.Severity.ABSOLUTE,
                "High risk of hepatotoxicity and acute liver failure",
                Arrays.asList("Opioid analgesics (morphine, oxycodone)", "NSAIDs with caution")
        ));

        // Opioids - reduced metabolism in hepatic impairment
        HEPATIC_DOSING_GUIDELINES.put("morphine", new HepaticDosingGuideline(
                "morphine",
                0,  // Can use with caution
                Arrays.asList(
                        new HepaticDoseAdjustment("A", 1.0, "Standard dosing", "Monitor for excessive sedation"),
                        new HepaticDoseAdjustment("B", 0.5, "Reduce dose by 50%", "Increased bioavailability, risk of encephalopathy"),
                        new HepaticDoseAdjustment("C", 0.33, "Reduce dose by 67%", "Start low, go slow; high encephalopathy risk")
                ),
                Contraindication.Severity.CAUTION,
                "Accumulation and prolonged half-life; precipitates hepatic encephalopathy",
                Arrays.asList("Fentanyl (better tolerated)", "Buprenorphine (partial agonist)")
        ));

        HEPATIC_DOSING_GUIDELINES.put("oxycodone", HEPATIC_DOSING_GUIDELINES.get("morphine"));

        // Benzodiazepines - contraindicated in severe hepatic impairment
        HEPATIC_DOSING_GUIDELINES.put("lorazepam", new HepaticDosingGuideline(
                "lorazepam",
                10,  // Avoid in Child-Pugh C
                Arrays.asList(
                        new HepaticDoseAdjustment("A", 0.75, "Reduce dose by 25%", "Monitor for sedation"),
                        new HepaticDoseAdjustment("B", 0.5, "Reduce dose by 50%", "High risk of encephalopathy"),
                        new HepaticDoseAdjustment("C", 0.0, "Avoid use", "May precipitate hepatic coma")
                ),
                Contraindication.Severity.ABSOLUTE,
                "Precipitates hepatic encephalopathy; long half-life in cirrhosis",
                Arrays.asList("Avoid sedatives", "Non-pharmacologic anxiety management")
        ));

        HEPATIC_DOSING_GUIDELINES.put("diazepam", HEPATIC_DOSING_GUIDELINES.get("lorazepam"));
        HEPATIC_DOSING_GUIDELINES.put("midazolam", HEPATIC_DOSING_GUIDELINES.get("lorazepam"));

        // Warfarin - increased sensitivity in hepatic impairment
        HEPATIC_DOSING_GUIDELINES.put("warfarin", new HepaticDosingGuideline(
                "warfarin",
                0,
                Arrays.asList(
                        new HepaticDoseAdjustment("A", 0.75, "Reduce by 25%", "Decreased clotting factor synthesis"),
                        new HepaticDoseAdjustment("B", 0.5, "Reduce by 50%", "Baseline coagulopathy present"),
                        new HepaticDoseAdjustment("C", 0.33, "Reduce by 67%", "Monitor INR every 1-2 days initially")
                ),
                Contraindication.Severity.CAUTION,
                "Decreased synthesis of vitamin K-dependent clotting factors",
                Arrays.asList("Direct oral anticoagulants with caution", "Consider risk vs benefit")
        ));

        // Statins - hepatotoxicity monitoring required
        HEPATIC_DOSING_GUIDELINES.put("atorvastatin", new HepaticDosingGuideline(
                "atorvastatin",
                10,  // Contraindicated in active liver disease
                Arrays.asList(
                        new HepaticDoseAdjustment("A", 1.0, "Standard dosing", "Monitor LFTs every 3 months"),
                        new HepaticDoseAdjustment("B", 0.5, "Reduce dose by 50%", "Monitor LFTs monthly"),
                        new HepaticDoseAdjustment("C", 0.0, "Contraindicated", "Active liver disease")
                ),
                Contraindication.Severity.ABSOLUTE,
                "Hepatotoxicity risk; contraindicated in active liver disease",
                Arrays.asList("Ezetimibe (no hepatic metabolism)", "PCSK9 inhibitors")
        ));

        HEPATIC_DOSING_GUIDELINES.put("simvastatin", HEPATIC_DOSING_GUIDELINES.get("atorvastatin"));

        // Metoprolol - reduced first-pass metabolism
        HEPATIC_DOSING_GUIDELINES.put("metoprolol", new HepaticDosingGuideline(
                "metoprolol",
                0,
                Arrays.asList(
                        new HepaticDoseAdjustment("A", 0.75, "Reduce by 25%", "Increased bioavailability"),
                        new HepaticDoseAdjustment("B", 0.5, "Reduce by 50%", "Monitor for bradycardia"),
                        new HepaticDoseAdjustment("C", 0.33, "Reduce by 67%", "Start low, titrate slowly")
                ),
                Contraindication.Severity.CAUTION,
                "Extensive hepatic metabolism; increased bioavailability in cirrhosis",
                Arrays.asList("Atenolol (renal elimination)", "Carvedilol with dose reduction")
        ));

        // Antibiotics - varied hepatic effects
        HEPATIC_DOSING_GUIDELINES.put("levofloxacin", new HepaticDosingGuideline(
                "levofloxacin",
                0,
                Arrays.asList(
                        new HepaticDoseAdjustment("A", 1.0, "No adjustment", "Primarily renal elimination"),
                        new HepaticDoseAdjustment("B", 1.0, "No adjustment", "Safe in hepatic impairment"),
                        new HepaticDoseAdjustment("C", 1.0, "No adjustment", "Renal dosing takes precedence")
                ),
                Contraindication.Severity.CAUTION,
                "Minimal hepatic metabolism - safe choice in liver disease",
                null
        ));

        // Rifampin - potent hepatotoxin
        HEPATIC_DOSING_GUIDELINES.put("rifampin", new HepaticDosingGuideline(
                "rifampin",
                10,  // Contraindicated in active liver disease
                Arrays.asList(
                        new HepaticDoseAdjustment("A", 0.75, "Reduce dose, monitor LFTs", "Known hepatotoxin"),
                        new HepaticDoseAdjustment("B", 0.5, "Reduce dose significantly", "High risk of decompensation"),
                        new HepaticDoseAdjustment("C", 0.0, "Contraindicated", "Severe hepatotoxicity risk")
                ),
                Contraindication.Severity.ABSOLUTE,
                "Potent hepatotoxin; can cause acute liver failure",
                Arrays.asList("Levofloxacin", "Moxifloxacin", "Avoid hepatotoxic antibiotics")
        ));
    }

    /**
     * Initialize known hepatotoxic medications
     */
    private static void initializeHepatotoxicMedications() {
        HEPATOTOXIC_MEDICATIONS.addAll(Arrays.asList(
                "acetaminophen", "paracetamol",
                "isoniazid", "rifampin",
                "valproic acid", "phenytoin", "carbamazepine",
                "methotrexate", "azathioprine",
                "amiodarone", "ketoconazole",
                "statins", "atorvastatin", "simvastatin",
                "tetracycline", "erythromycin",
                "nsaids", "diclofenac", "ibuprofen"
        ));
    }

    /**
     * Check for hepatic contraindications
     *
     * @param action Clinical action to check
     * @param state Patient context state with lab values
     * @return List of hepatic contraindications
     */
    public List<Contraindication> checkHepaticContraindications(
            ClinicalAction action,
            PatientContextState state) {

        List<Contraindication> contraindications = new ArrayList<>();

        if (action == null || action.getMedicationDetails() == null) {
            return contraindications;
        }

        MedicationDetails medication = action.getMedicationDetails();
        String medicationName = medication.getName();

        if (medicationName == null || medicationName.trim().isEmpty()) {
            return contraindications;
        }

        // Calculate Child-Pugh score if possible
        Integer childPughScore = calculateChildPughScore(state);

        if (childPughScore == null) {
            // If we can't calculate Child-Pugh, check for hepatotoxicity warning
            if (isHepatotoxic(medicationName)) {
                Contraindication contraindication = new Contraindication(
                        Contraindication.ContraindicationType.ORGAN_DYSFUNCTION,
                        "Hepatotoxic medication - monitor liver function"
                );
                contraindication.setSeverity(Contraindication.Severity.CAUTION);
                contraindication.setFound(true);
                contraindication.setEvidence("Medication has known hepatotoxicity risk");
                contraindication.setRiskScore(0.3);
                contraindication.setClinicalGuidance(
                        "Monitor AST, ALT, bilirubin at baseline and periodically. " +
                                "Discontinue if transaminases >3x ULN.");
                contraindications.add(contraindication);
            }
            return contraindications;
        }

        logger.debug("Calculated Child-Pugh score: {} for hepatic dosing check of {}",
                childPughScore, medicationName);

        // Check if medication requires hepatic adjustment
        HepaticDosingGuideline guideline = getHepaticDosingGuideline(medicationName.toLowerCase());

        if (guideline == null) {
            // Still warn about hepatotoxicity
            if (isHepatotoxic(medicationName)) {
                Contraindication contraindication = new Contraindication(
                        Contraindication.ContraindicationType.ORGAN_DYSFUNCTION,
                        String.format("Hepatotoxic medication with hepatic impairment (Child-Pugh %s)",
                                getChildPughClass(childPughScore))
                );
                contraindication.setSeverity(Contraindication.Severity.CAUTION);
                contraindication.setFound(true);
                contraindication.setEvidence(String.format("Child-Pugh score: %d (%s)",
                        childPughScore, getChildPughClass(childPughScore)));
                contraindication.setRiskScore(0.4);
                contraindication.setClinicalGuidance("Monitor LFTs closely; consider alternative agent");
                contraindications.add(contraindication);
            }
            return contraindications;
        }

        // Check if contraindicated in current hepatic function
        if (childPughScore > guideline.contraindicationThreshold) {
            String childPughClass = getChildPughClass(childPughScore);

            Contraindication contraindication = new Contraindication(
                    Contraindication.ContraindicationType.ORGAN_DYSFUNCTION,
                    String.format("Hepatic impairment contraindication (Child-Pugh %s, score %d)",
                            childPughClass, childPughScore)
            );
            contraindication.setSeverity(guideline.severity);
            contraindication.setFound(true);
            contraindication.setEvidence(String.format(
                    "Child-Pugh Class %s (score %d). %s",
                    childPughClass, childPughScore, guideline.rationale));
            contraindication.setRiskScore(calculateHepaticRiskScore(childPughScore));

            // Suggest alternatives if available
            if (guideline.alternatives != null && !guideline.alternatives.isEmpty()) {
                contraindication.setAlternativeAvailable(true);
                contraindication.setAlternativeMedication(String.join(" OR ", guideline.alternatives));
                contraindication.setAlternativeRationale("Alternative with better hepatic safety profile");
            }

            contraindications.add(contraindication);
            logger.warn("Hepatic contraindication: {} (Child-Pugh {} score {})",
                    medicationName, childPughClass, childPughScore);
        }

        return contraindications;
    }

    /**
     * Adjust medication dose based on hepatic function
     *
     * Modifies MedicationDetails in-place
     *
     * @param medication Medication to adjust
     * @param state Patient context state
     * @return true if dose was adjusted
     */
    public boolean adjustDose(MedicationDetails medication, PatientContextState state) {
        if (medication == null || medication.getName() == null) {
            return false;
        }

        String medicationName = medication.getName();
        Integer childPughScore = calculateChildPughScore(state);

        if (childPughScore == null) {
            logger.debug("Cannot adjust dose for {} - unable to calculate Child-Pugh score", medicationName);
            return false;
        }

        HepaticDosingGuideline guideline = getHepaticDosingGuideline(medicationName.toLowerCase());

        if (guideline == null || guideline.doseAdjustments == null) {
            return false;
        }

        // Find appropriate dose adjustment for current Child-Pugh class
        String childPughClass = getChildPughClass(childPughScore);
        HepaticDoseAdjustment adjustment = findDoseAdjustment(guideline.doseAdjustments, childPughClass);

        if (adjustment == null || adjustment.doseFactor == 1.0) {
            return false;  // No adjustment needed
        }

        // Check if medication is contraindicated (dose factor 0.0)
        if (adjustment.doseFactor == 0.0) {
            logger.warn("Medication {} contraindicated in Child-Pugh Class {}", medicationName, childPughClass);
            return false;  // Don't adjust - should be flagged as contraindication
        }

        // Apply dose adjustment
        double originalDose = medication.getCalculatedDose();
        double adjustedDose = originalDose * adjustment.doseFactor;

        medication.setCalculatedDose(adjustedDose);

        // Update dose calculation method
        String currentMethod = medication.getDoseCalculationMethod();
        if ("renal_adjusted".equals(currentMethod)) {
            medication.setDoseCalculationMethod("renal_hepatic_adjusted");
        } else {
            medication.setDoseCalculationMethod("hepatic_adjusted");
        }

        // Document adjustment
        String adjustmentNote = String.format(
                "Dose adjusted from %.1f to %.1f %s (%.0f%% of normal dose) for Child-Pugh Class %s (score %d). %s",
                originalDose, adjustedDose, medication.getDoseUnit(),
                adjustment.doseFactor * 100, childPughClass, childPughScore, adjustment.guidance);

        medication.setAdministrationInstructions(
                (medication.getAdministrationInstructions() != null ?
                        medication.getAdministrationInstructions() + "; " : "") +
                        adjustmentNote);

        logger.info("Hepatic dose adjustment applied for {}: {:.1f} → {:.1f} {} (Child-Pugh Class {})",
                medicationName, originalDose, adjustedDose, medication.getDoseUnit(), childPughClass);

        return true;
    }

    /**
     * Calculate Child-Pugh score from lab values
     *
     * Simplified calculation using available lab values only.
     * Full score requires clinical assessment of ascites and encephalopathy.
     *
     * @param state Patient context state
     * @return Child-Pugh score (5-15), or null if cannot calculate
     */
    public Integer calculateChildPughScore(PatientContextState state) {
        if (state == null || state.getRecentLabs() == null) {
            return null;
        }

        Map<String, LabResult> labs = state.getRecentLabs();

        // Get lab values
        Double bilirubin = getLabValue(labs, "bilirubin");
        Double albumin = getLabValue(labs, "albumin");
        Double inr = getLabValue(labs, "inr");

        // Need at least 2 of 3 lab values for estimation
        int availableValues = 0;
        if (bilirubin != null) availableValues++;
        if (albumin != null) availableValues++;
        if (inr != null) availableValues++;

        if (availableValues < 2) {
            logger.debug("Insufficient lab data for Child-Pugh calculation (need 2 of 3 values)");
            return null;
        }

        int score = 0;

        // Bilirubin scoring (mg/dL)
        if (bilirubin != null) {
            if (bilirubin < 2.0) {
                score += 1;
            } else if (bilirubin <= 3.0) {
                score += 2;
            } else {
                score += 3;
            }
        }

        // Albumin scoring (g/dL)
        if (albumin != null) {
            if (albumin > 3.5) {
                score += 1;
            } else if (albumin >= 2.8) {
                score += 2;
            } else {
                score += 3;
            }
        }

        // INR scoring
        if (inr != null) {
            if (inr < 1.7) {
                score += 1;
            } else if (inr <= 2.3) {
                score += 2;
            } else {
                score += 3;
            }
        }

        // Note: Ascites and encephalopathy require clinical assessment
        // For now, assume none (add 2 points for minimal scoring)
        score += 2;  // 1 point each for no ascites, no encephalopathy

        logger.debug("Child-Pugh score calculated: {} (Bili: {}, Alb: {}, INR: {})",
                score, bilirubin, albumin, inr);

        return score;
    }

    /**
     * Check if medication is hepatotoxic
     *
     * @param medicationName Medication name
     * @return true if medication has known hepatotoxicity
     */
    public boolean isHepatotoxic(String medicationName) {
        if (medicationName == null) {
            return false;
        }

        String medLower = medicationName.toLowerCase();
        return HEPATOTOXIC_MEDICATIONS.stream()
                .anyMatch(toxicMed -> medLower.contains(toxicMed));
    }

    /**
     * Get Child-Pugh class from score
     *
     * @param score Child-Pugh score (5-15)
     * @return Class A, B, or C
     */
    private String getChildPughClass(int score) {
        if (score >= 5 && score <= 6) {
            return "A";
        } else if (score >= 7 && score <= 9) {
            return "B";
        } else {
            return "C";
        }
    }

    /**
     * Get hepatic dosing guideline for medication
     *
     * @param medicationNameLower Medication name (lowercase)
     * @return HepaticDosingGuideline or null if not found
     */
    private HepaticDosingGuideline getHepaticDosingGuideline(String medicationNameLower) {
        for (Map.Entry<String, HepaticDosingGuideline> entry : HEPATIC_DOSING_GUIDELINES.entrySet()) {
            if (medicationNameLower.contains(entry.getKey())) {
                return entry.getValue();
            }
        }
        return null;
    }

    /**
     * Find appropriate dose adjustment for Child-Pugh class
     *
     * @param adjustments List of dose adjustments
     * @param childPughClass Child-Pugh class (A, B, or C)
     * @return HepaticDoseAdjustment or null if not found
     */
    private HepaticDoseAdjustment findDoseAdjustment(
            List<HepaticDoseAdjustment> adjustments,
            String childPughClass) {
        for (HepaticDoseAdjustment adjustment : adjustments) {
            if (adjustment.childPughClass.equals(childPughClass)) {
                return adjustment;
            }
        }
        return null;
    }

    /**
     * Calculate risk score based on Child-Pugh score
     *
     * @param childPughScore Score (5-15)
     * @return Risk score (0.0-1.0)
     */
    private double calculateHepaticRiskScore(int childPughScore) {
        if (childPughScore <= 6) {
            return 0.2;  // Class A - low risk
        } else if (childPughScore <= 9) {
            return 0.5;  // Class B - moderate risk
        } else {
            return 0.8;  // Class C - high risk
        }
    }

    /**
     * Get lab value from recent labs map
     *
     * @param labs Recent labs map
     * @param labName Lab name to search for
     * @return Lab value or null if not found
     */
    private Double getLabValue(Map<String, LabResult> labs, String labName) {
        for (Map.Entry<String, LabResult> entry : labs.entrySet()) {
            if (entry.getKey().toLowerCase().contains(labName.toLowerCase())) {
                return entry.getValue().getValue();
            }
        }
        return null;
    }

    /**
     * Hepatic Dosing Guideline definition
     */
    private static class HepaticDosingGuideline implements Serializable {
        private static final long serialVersionUID = 1L;

        String medicationName;
        int contraindicationThreshold;  // Child-Pugh score above which contraindicated
        List<HepaticDoseAdjustment> doseAdjustments;
        Contraindication.Severity severity;
        String rationale;
        List<String> alternatives;

        HepaticDosingGuideline(String medicationName, int contraindicationThreshold,
                             List<HepaticDoseAdjustment> doseAdjustments,
                             Contraindication.Severity severity, String rationale,
                             List<String> alternatives) {
            this.medicationName = medicationName;
            this.contraindicationThreshold = contraindicationThreshold;
            this.doseAdjustments = doseAdjustments;
            this.severity = severity;
            this.rationale = rationale;
            this.alternatives = alternatives;
        }
    }

    /**
     * Hepatic Dose Adjustment definition
     */
    private static class HepaticDoseAdjustment implements Serializable {
        private static final long serialVersionUID = 1L;

        String childPughClass;  // A, B, or C
        double doseFactor;      // Dose multiplier (0.5 = 50% of normal dose)
        String recommendation;  // Dosing recommendation
        String guidance;        // Clinical guidance

        HepaticDoseAdjustment(String childPughClass, double doseFactor,
                            String recommendation, String guidance) {
            this.childPughClass = childPughClass;
            this.doseFactor = doseFactor;
            this.recommendation = recommendation;
            this.guidance = guidance;
        }
    }

    @Override
    public String toString() {
        return "HepaticDosingAdjuster{guidelinesCount=" + HEPATIC_DOSING_GUIDELINES.size() +
                ", hepatotoxicMedsCount=" + HEPATOTOXIC_MEDICATIONS.size() + "}";
    }
}
