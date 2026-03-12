package com.cardiofit.flink.safety;

import com.cardiofit.flink.models.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;

/**
 * Renal Dosing Adjuster - Creatinine clearance-based medication dosing
 *
 * Performs:
 * - Creatinine clearance calculation using Cockcroft-Gault formula
 * - Medication dose adjustments based on CrCl
 * - Contraindication flagging for severe renal impairment
 * - Dosing interval adjustments
 *
 * Cockcroft-Gault Formula:
 * CrCl (mL/min) = [(140 - age) × weight (kg)] / (72 × serum creatinine mg/dL)
 * Multiply by 0.85 for females
 *
 * CKD Staging:
 * - Stage 1: CrCl ≥90 mL/min (normal)
 * - Stage 2: CrCl 60-89 mL/min (mild)
 * - Stage 3a: CrCl 45-59 mL/min (mild-moderate)
 * - Stage 3b: CrCl 30-44 mL/min (moderate-severe)
 * - Stage 4: CrCl 15-29 mL/min (severe)
 * - Stage 5: CrCl <15 mL/min (kidney failure)
 *
 * References:
 * - Cockcroft DW, Gault MH. Prediction of creatinine clearance from serum creatinine. Nephron. 1976;16(1):31-41
 * - KDIGO Clinical Practice Guideline for Chronic Kidney Disease
 * - Lexicomp Renal Dosing Database
 * - FDA Drug Prescribing Information
 *
 * @author CardioFit Platform - Module 3 Phase 4
 * @version 1.0
 * @since 2025-10-20
 */
public class RenalDosingAdjuster implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger logger = LoggerFactory.getLogger(RenalDosingAdjuster.class);

    // Renal dosing guidelines database
    private static final Map<String, RenalDosingGuideline> RENAL_DOSING_GUIDELINES = new HashMap<>();

    static {
        initializeRenalDosingGuidelines();
    }

    /**
     * Initialize renal dosing guidelines for common medications
     */
    private static void initializeRenalDosingGuidelines() {
        // Metformin - contraindicated in severe renal impairment
        RENAL_DOSING_GUIDELINES.put("metformin", new RenalDosingGuideline(
                "metformin",
                30.0,  // Contraindicated if CrCl <30
                null,  // No dose adjustment needed above threshold
                Contraindication.Severity.ABSOLUTE,
                "Risk of lactic acidosis in renal impairment",
                Arrays.asList("DPP-4 inhibitor (sitagliptin)", "GLP-1 agonist (liraglutide)")
        ));

        // Enoxaparin - dose adjustment for CrCl <30
        RENAL_DOSING_GUIDELINES.put("enoxaparin", new RenalDosingGuideline(
                "enoxaparin",
                30.0,
                Arrays.asList(
                        new DoseAdjustment(30, 60, 0.75, "q12h → q24h", "Reduce to 75% of normal dose OR extend interval"),
                        new DoseAdjustment(0, 30, 0.5, "Consider UFH", "Consider unfractionated heparin instead")
                ),
                Contraindication.Severity.RELATIVE,
                "Increased bleeding risk due to accumulation",
                Arrays.asList("Unfractionated heparin (UFH)", "Fondaparinux if CrCl >30")
        ));

        // Gabapentin - significant renal dosing adjustment
        RENAL_DOSING_GUIDELINES.put("gabapentin", new RenalDosingGuideline(
                "gabapentin",
                15.0,  // Contraindicated if CrCl <15 (ESRD without dialysis)
                Arrays.asList(
                        new DoseAdjustment(60, 200, 1.0, "No adjustment", "Normal dosing"),
                        new DoseAdjustment(30, 60, 0.5, "Reduce to 50%", "Reduce dose by 50% or extend interval"),
                        new DoseAdjustment(15, 30, 0.33, "Reduce to 33%", "Reduce dose to 300-600 mg daily"),
                        new DoseAdjustment(0, 15, 0.25, "Reduce to 25%", "Hemodialysis patients: dose after dialysis")
                ),
                Contraindication.Severity.RELATIVE,
                "CNS side effects and sedation increase with accumulation",
                Arrays.asList("Pregabalin (also requires renal adjustment)", "Duloxetine (no renal adjustment)")
        ));

        // Vancomycin - therapeutic drug monitoring required
        RENAL_DOSING_GUIDELINES.put("vancomycin", new RenalDosingGuideline(
                "vancomycin",
                10.0,  // Can use with dialysis
                Arrays.asList(
                        new DoseAdjustment(60, 200, 1.0, "Standard dosing", "15-20 mg/kg q8-12h"),
                        new DoseAdjustment(30, 60, 0.75, "Extend interval", "15-20 mg/kg q24h"),
                        new DoseAdjustment(10, 30, 0.5, "Extend interval further", "15-20 mg/kg q48-72h")
                ),
                Contraindication.Severity.CAUTION,
                "Nephrotoxicity risk; requires therapeutic drug monitoring",
                Arrays.asList("Linezolid (no renal adjustment)", "Daptomycin (dose adjustment needed)")
        ));

        // Dabigatran - direct oral anticoagulant with renal elimination
        RENAL_DOSING_GUIDELINES.put("dabigatran", new RenalDosingGuideline(
                "dabigatran",
                15.0,  // Contraindicated if CrCl <15
                Arrays.asList(
                        new DoseAdjustment(50, 200, 1.0, "Standard 150 mg BID", "No adjustment"),
                        new DoseAdjustment(30, 50, 0.5, "Reduce to 110 mg BID", "Use reduced dose"),
                        new DoseAdjustment(15, 30, 0.5, "Reduce to 75 mg BID", "Use reduced dose with caution")
                ),
                Contraindication.Severity.ABSOLUTE,
                "80% renal elimination; high bleeding risk in renal impairment",
                Arrays.asList("Apixaban (lower renal elimination)", "Warfarin (no renal adjustment)")
        ));

        // Digoxin - narrow therapeutic index with renal elimination
        RENAL_DOSING_GUIDELINES.put("digoxin", new RenalDosingGuideline(
                "digoxin",
                10.0,
                Arrays.asList(
                        new DoseAdjustment(60, 200, 1.0, "Standard 0.125-0.25 mg daily", "No adjustment"),
                        new DoseAdjustment(30, 60, 0.75, "Reduce by 25%", "0.125 mg daily or every other day"),
                        new DoseAdjustment(10, 30, 0.5, "Reduce by 50%", "0.0625-0.125 mg every other day")
                ),
                Contraindication.Severity.CAUTION,
                "Narrow therapeutic index; toxicity risk with accumulation",
                Arrays.asList("Monitor digoxin level", "Consider beta-blocker or calcium channel blocker")
        ));

        // Piperacillin-Tazobactam - common broad-spectrum antibiotic
        RENAL_DOSING_GUIDELINES.put("piperacillin", new RenalDosingGuideline(
                "piperacillin",
                10.0,
                Arrays.asList(
                        new DoseAdjustment(40, 200, 1.0, "4.5g q6h", "No adjustment"),
                        new DoseAdjustment(20, 40, 0.75, "3.375g q6h", "Reduce dose by 25%"),
                        new DoseAdjustment(10, 20, 0.5, "2.25g q6h", "Reduce dose by 50%")
                ),
                Contraindication.Severity.CAUTION,
                "Seizure risk with accumulation",
                Arrays.asList("Cefepime (also requires adjustment)", "Meropenem (requires adjustment)")
        ));

        // Atorvastatin - minimal renal adjustment needed
        RENAL_DOSING_GUIDELINES.put("atorvastatin", new RenalDosingGuideline(
                "atorvastatin",
                0.0,  // No contraindication
                Arrays.asList(new DoseAdjustment(0, 200, 1.0, "No adjustment needed", "Hepatic metabolism, safe in CKD")),
                Contraindication.Severity.CAUTION,
                "No dose adjustment required for renal impairment",
                null
        ));
    }

    /**
     * Check for renal contraindications
     *
     * @param action Clinical action to check
     * @param state Patient context state with lab values
     * @return List of renal contraindications
     */
    public List<Contraindication> checkRenalContraindications(
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

        // Calculate creatinine clearance
        Double crCl = calculateCrCl(state);

        if (crCl == null) {
            logger.warn("Cannot calculate CrCl for {} - missing required data", medicationName);
            return contraindications;
        }

        logger.debug("Calculated CrCl: {:.1f} mL/min for renal dosing check of {}",
                crCl, medicationName);

        // Check if medication requires renal adjustment
        RenalDosingGuideline guideline = getRenalDosingGuideline(medicationName.toLowerCase());

        if (guideline == null) {
            logger.debug("No renal dosing guideline for {}", medicationName);
            return contraindications;
        }

        // Check if contraindicated in current renal function
        if (isContraindicatedInRenalImpairment(medicationName, crCl)) {
            Contraindication contraindication = new Contraindication(
                    Contraindication.ContraindicationType.ORGAN_DYSFUNCTION,
                    String.format("Severe renal impairment contraindication (CrCl %.1f mL/min)", crCl)
            );
            contraindication.setSeverity(guideline.severity);
            contraindication.setFound(true);
            contraindication.setEvidence(String.format(
                    "CrCl %.1f mL/min (threshold: %.1f mL/min). %s",
                    crCl, guideline.contraindicationThreshold, guideline.rationale));
            contraindication.setRiskScore(calculateRenalRiskScore(crCl, guideline.contraindicationThreshold));

            // Suggest alternatives if available
            if (guideline.alternatives != null && !guideline.alternatives.isEmpty()) {
                contraindication.setAlternativeAvailable(true);
                contraindication.setAlternativeMedication(String.join(" OR ", guideline.alternatives));
                contraindication.setAlternativeRationale("Alternative with better renal safety profile");
            }

            contraindications.add(contraindication);
            logger.warn("Renal contraindication: {} (CrCl {:.1f} < threshold {:.1f})",
                    medicationName, crCl, guideline.contraindicationThreshold);
        }

        return contraindications;
    }

    /**
     * Adjust medication dose based on renal function
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
        Double crCl = calculateCrCl(state);

        if (crCl == null) {
            logger.warn("Cannot adjust dose for {} - missing CrCl calculation data", medicationName);
            return false;
        }

        RenalDosingGuideline guideline = getRenalDosingGuideline(medicationName.toLowerCase());

        if (guideline == null || guideline.doseAdjustments == null) {
            return false;
        }

        // Find appropriate dose adjustment for current CrCl
        DoseAdjustment adjustment = findDoseAdjustment(guideline.doseAdjustments, crCl);

        if (adjustment == null || adjustment.doseFactor == 1.0) {
            return false;  // No adjustment needed
        }

        // Apply dose adjustment
        double originalDose = medication.getCalculatedDose();
        double adjustedDose = originalDose * adjustment.doseFactor;

        medication.setCalculatedDose(adjustedDose);
        medication.setDoseCalculationMethod("renal_adjusted");
        medication.setPatientEgfr(crCl);  // Store CrCl as eGFR approximation
        medication.setRenalAdjustmentApplied(String.format(
                "Dose adjusted from %.1f to %.1f %s (%.0f%% of normal dose) for CrCl %.1f mL/min. %s",
                originalDose, adjustedDose, medication.getDoseUnit(),
                adjustment.doseFactor * 100, crCl, adjustment.guidance));

        // Update administration instructions if interval changed
        if (adjustment.frequencyChange != null && !adjustment.frequencyChange.contains("No adjustment")) {
            medication.setAdministrationInstructions(
                    (medication.getAdministrationInstructions() != null ?
                            medication.getAdministrationInstructions() + "; " : "") +
                            "Frequency adjusted: " + adjustment.frequencyChange);
        }

        logger.info("Renal dose adjustment applied for {}: {:.1f} → {:.1f} {} (CrCl {:.1f})",
                medicationName, originalDose, adjustedDose, medication.getDoseUnit(), crCl);

        return true;
    }

    /**
     * Calculate creatinine clearance using Cockcroft-Gault formula
     *
     * Formula: CrCl = [(140 - age) × weight (kg)] / (72 × serum creatinine)
     * Multiply by 0.85 for females
     *
     * @param state Patient context state
     * @return Creatinine clearance in mL/min, or null if calculation not possible
     */
    public Double calculateCrCl(PatientContextState state) {
        if (state == null) {
            return null;
        }

        // Get required values
        Double creatinine = getLatestCreatinine(state);
        Integer age = getAge(state);
        Double weight = getWeight(state);
        String sex = getSex(state);

        // Validate all required values are present
        if (creatinine == null || age == null || weight == null || sex == null) {
            logger.debug("Missing data for CrCl calculation - Cr: {}, Age: {}, Weight: {}, Sex: {}",
                    creatinine, age, weight, sex);
            return null;
        }

        // Validate ranges
        if (creatinine <= 0 || age <= 0 || weight <= 0) {
            logger.warn("Invalid values for CrCl calculation - Cr: {}, Age: {}, Weight: {}",
                    creatinine, age, weight);
            return null;
        }

        // Cockcroft-Gault formula
        double crCl = ((140.0 - age) * weight) / (72.0 * creatinine);

        // Apply female correction factor
        if ("female".equalsIgnoreCase(sex) || "f".equalsIgnoreCase(sex)) {
            crCl *= 0.85;
        }

        logger.debug("Calculated CrCl: {:.1f} mL/min (Age: {}, Weight: {} kg, Cr: {:.2f} mg/dL, Sex: {})",
                crCl, age, weight, creatinine, sex);

        return crCl;
    }

    /**
     * Check if medication is contraindicated in renal impairment
     *
     * @param medicationName Medication name
     * @param crCl Creatinine clearance in mL/min
     * @return true if contraindicated
     */
    public boolean isContraindicatedInRenalImpairment(String medicationName, double crCl) {
        if (medicationName == null) {
            return false;
        }

        RenalDosingGuideline guideline = getRenalDosingGuideline(medicationName.toLowerCase());

        if (guideline == null) {
            return false;
        }

        return crCl < guideline.contraindicationThreshold;
    }

    /**
     * Get renal dosing guideline for medication
     *
     * @param medicationNameLower Medication name (lowercase)
     * @return RenalDosingGuideline or null if not found
     */
    private RenalDosingGuideline getRenalDosingGuideline(String medicationNameLower) {
        for (Map.Entry<String, RenalDosingGuideline> entry : RENAL_DOSING_GUIDELINES.entrySet()) {
            if (medicationNameLower.contains(entry.getKey())) {
                return entry.getValue();
            }
        }
        return null;
    }

    /**
     * Find appropriate dose adjustment for current CrCl
     *
     * @param adjustments List of dose adjustments
     * @param crCl Current creatinine clearance
     * @return DoseAdjustment or null if not found
     */
    private DoseAdjustment findDoseAdjustment(List<DoseAdjustment> adjustments, double crCl) {
        for (DoseAdjustment adjustment : adjustments) {
            if (crCl >= adjustment.crClMin && crCl < adjustment.crClMax) {
                return adjustment;
            }
        }
        return null;
    }

    /**
     * Calculate risk score based on CrCl and threshold
     *
     * @param crCl Current creatinine clearance
     * @param threshold Contraindication threshold
     * @return Risk score (0.0-1.0)
     */
    private double calculateRenalRiskScore(double crCl, double threshold) {
        if (crCl >= threshold) {
            return 0.0;
        }
        // Higher risk as CrCl decreases below threshold
        double percentBelow = (threshold - crCl) / threshold;
        return Math.min(0.3 + (percentBelow * 0.5), 0.9);
    }

    // Helper methods to extract values from patient state

    private Double getLatestCreatinine(PatientContextState state) {
        if (state.getRecentLabs() == null) {
            return null;
        }
        // Look for creatinine in recent labs (LOINC code 2160-0 or name)
        for (Map.Entry<String, LabResult> entry : state.getRecentLabs().entrySet()) {
            String key = entry.getKey().toLowerCase();
            if (key.contains("creatinine") || key.equals("2160-0")) {
                return entry.getValue().getValue();
            }
        }
        return null;
    }

    private Integer getAge(PatientContextState state) {
        if (state.getDemographics() != null) {
            return state.getDemographics().getAge();
        }
        return null;
    }

    private Double getWeight(PatientContextState state) {
        if (state.getLatestVitals() != null) {
            Object weight = state.getLatestVitals().get("weight");
            if (weight instanceof Number) {
                return ((Number) weight).doubleValue();
            }
        }
        return null;
    }

    private String getSex(PatientContextState state) {
        if (state.getDemographics() != null) {
            return state.getDemographics().getGender();
        }
        return null;
    }

    /**
     * Renal Dosing Guideline definition
     */
    private static class RenalDosingGuideline implements Serializable {
        private static final long serialVersionUID = 1L;

        String medicationName;
        double contraindicationThreshold;  // CrCl threshold below which contraindicated
        List<DoseAdjustment> doseAdjustments;
        Contraindication.Severity severity;
        String rationale;
        List<String> alternatives;

        RenalDosingGuideline(String medicationName, double contraindicationThreshold,
                           List<DoseAdjustment> doseAdjustments,
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
     * Dose Adjustment definition
     */
    private static class DoseAdjustment implements Serializable {
        private static final long serialVersionUID = 1L;

        double crClMin;       // Minimum CrCl for this adjustment
        double crClMax;       // Maximum CrCl for this adjustment
        double doseFactor;    // Dose multiplier (0.5 = 50% of normal dose)
        String frequencyChange; // Frequency adjustment description
        String guidance;      // Clinical guidance

        DoseAdjustment(double crClMin, double crClMax, double doseFactor,
                      String frequencyChange, String guidance) {
            this.crClMin = crClMin;
            this.crClMax = crClMax;
            this.doseFactor = doseFactor;
            this.frequencyChange = frequencyChange;
            this.guidance = guidance;
        }
    }

    @Override
    public String toString() {
        return "RenalDosingAdjuster{guidelinesCount=" + RENAL_DOSING_GUIDELINES.size() + "}";
    }
}
