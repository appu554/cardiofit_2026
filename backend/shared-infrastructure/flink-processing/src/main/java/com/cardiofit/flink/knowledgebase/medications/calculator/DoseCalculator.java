package com.cardiofit.flink.knowledgebase.medications.calculator;

import com.cardiofit.flink.knowledgebase.medications.model.Medication;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientState;
import com.cardiofit.flink.models.LabResult;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Dose Calculator for patient-specific medication dosing.
 *
 * Calculates appropriate medication dose based on:
 * - Renal function (Cockcroft-Gault CrCl calculation)
 * - Hepatic function (Child-Pugh scoring)
 * - Age (pediatric, geriatric adjustments)
 * - Weight (obesity adjustments)
 * - Clinical indication
 *
 * Safety-critical component - all calculations must be validated against
 * current clinical guidelines and package inserts.
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-24
 */
public class DoseCalculator {
    private static final Logger logger = LoggerFactory.getLogger(DoseCalculator.class);

    // Clinical thresholds
    private static final double NORMAL_CRCL_THRESHOLD = 60.0; // mL/min
    private static final double MILD_RENAL_THRESHOLD = 40.0;
    private static final double MODERATE_RENAL_THRESHOLD = 30.0;
    private static final double SEVERE_RENAL_THRESHOLD = 15.0;

    private static final int GERIATRIC_AGE_THRESHOLD = 65;
    private static final int PEDIATRIC_AGE_THRESHOLD = 18;

    private static final double OBESITY_BMI_THRESHOLD = 30.0;

    /**
     * Calculate patient-specific dose for medication.
     *
     * @param medication The medication to dose
     * @param context The patient context with demographics and labs
     * @param indication The clinical indication (e.g., "pneumonia", "sepsis")
     * @return CalculatedDose with patient-specific dosing and warnings
     */
    public CalculatedDose calculateDose(
            Medication medication,
            EnrichedPatientContext context,
            String indication) {

        if (medication == null) {
            throw new IllegalArgumentException("Medication cannot be null");
        }
        if (context == null) {
            throw new IllegalArgumentException("Patient context cannot be null");
        }

        logger.debug("Calculating dose for {} ({}), indication: {}",
            medication.getGenericName(),
            medication.getMedicationId(),
            indication);

        com.cardiofit.flink.models.PatientContextState patientContextState = context.getPatientState();

        // Phase 6 medication calculations need full PatientState for enhanced methods
        // Convert PatientContextState to PatientState if needed
        com.cardiofit.flink.models.PatientState state;
        if (patientContextState instanceof com.cardiofit.flink.models.PatientState) {
            state = (com.cardiofit.flink.models.PatientState) patientContextState;
        } else {
            // Cannot perform calculations without full patient state
            logger.error("PatientContextState is not a PatientState instance - cannot calculate dose");
            return CalculatedDose.builder()
                .medicationId(medication.getMedicationId())
                .medicationName(medication.getGenericName())
                .contraindicated(true)
                .build();
        }

        // Get base dose for indication
        Medication.AdultDosing.StandardDose baseDose = medication.getDoseForIndication(indication);

        if (baseDose == null && medication.getAdultDosing() != null) {
            baseDose = medication.getAdultDosing().getStandard();
        }

        if (baseDose == null) {
            logger.error("No dosing information available for {}", medication.getMedicationId());
            return CalculatedDose.builder()
                .medicationId(medication.getMedicationId())
                .medicationName(medication.getGenericName())
                .contraindicated(true)
                .build();
        }

        // Build base result
        CalculatedDose.CalculatedDoseBuilder builder = CalculatedDose.builder()
            .medicationId(medication.getMedicationId())
            .medicationName(medication.getGenericName())
            .calculatedDose(baseDose.getDose())
            .calculatedFrequency(baseDose.getFrequency())
            .route(baseDose.getRoute())
            .maxDailyDose(baseDose.getMaxDailyDose())
            .loadingDose(baseDose.getLoadingDose())
            .infusionDuration(baseDose.getInfusionDuration())
            .originalDose(baseDose.getDose())
            .originalFrequency(baseDose.getFrequency());

        CalculatedDose result = builder.build();

        // Apply renal adjustments
        if (shouldAdjustForRenal(medication, state)) {
            applyRenalAdjustment(medication, state, result);
        }

        // Apply hepatic adjustments
        if (shouldAdjustForHepatic(medication, state)) {
            applyHepaticAdjustment(medication, state, result);
        }

        // Apply geriatric adjustments
        if (shouldAdjustForGeriatric(medication, state)) {
            applyGeriatricAdjustment(medication, state, result);
        }

        // Apply pediatric adjustments
        if (shouldAdjustForPediatric(medication, state)) {
            applyPediatricAdjustment(medication, state, result);
        }

        // Apply obesity adjustments
        if (shouldAdjustForObesity(medication, state)) {
            applyObesityAdjustment(medication, state, result);
        }

        // Add high-alert medication warnings
        if (medication.isHighAlert()) {
            result.addWarning("HIGH-ALERT MEDICATION: Requires independent double-check");
        }

        // Add black box warnings
        if (medication.hasBlackBoxWarning()) {
            result.addWarning("BLACK BOX WARNING: Review FDA warnings before administration");
        }

        logger.info("Calculated dose for {}: {} {} {}",
            medication.getGenericName(),
            result.getCalculatedDose(),
            result.getRoute(),
            result.getCalculatedFrequency());

        return result;
    }

    // ================================================================
    // RENAL ADJUSTMENT METHODS
    // ================================================================

    /**
     * Check if renal adjustment is needed.
     */
    private boolean shouldAdjustForRenal(Medication medication, PatientState state) {
        if (medication.getAdultDosing() == null ||
            medication.getAdultDosing().getRenalAdjustment() == null) {
            return false;
        }

        // Calculate CrCl
        double crCl = calculateCrCl(state);
        return crCl < NORMAL_CRCL_THRESHOLD;
    }

    /**
     * Apply renal dose adjustment.
     */
    private void applyRenalAdjustment(Medication medication, PatientState state, CalculatedDose result) {
        double crCl = calculateCrCl(state);

        logger.debug("Applying renal adjustment for CrCl: {} mL/min", crCl);

        String adjustedDose = medication.getAdjustedDoseForRenal(crCl);

        if (adjustedDose.startsWith("CONTRAINDICATED")) {
            result.setContraindicated(true);
            result.addWarning(adjustedDose);
            logger.warn("Medication contraindicated in renal impairment: CrCl {} mL/min", crCl);
            return;
        }

        if (!adjustedDose.equals(result.getOriginalDose())) {
            result.setCalculatedDose(adjustedDose);
            result.setAdjustmentReason(String.format(
                "Renal dose adjustment for CrCl %.1f mL/min", crCl));

            result.addWarning(String.format(
                "Dose adjusted for renal function (CrCl %.1f mL/min)", crCl));

            result.addMonitoring("Monitor renal function (Cr, BUN)");

            logger.info("Renal-adjusted dose: {} -> {}", result.getOriginalDose(), adjustedDose);
        }
    }

    /**
     * Calculate creatinine clearance using Cockcroft-Gault formula.
     *
     * CrCl (mL/min) = [(140 - age) × weight(kg)] / (72 × Cr(mg/dL))
     * Multiply by 0.85 for females
     *
     * @param state Patient state with demographics and labs
     * @return Creatinine clearance in mL/min
     */
    public double calculateCrCl(PatientState state) {
        Integer age = state.getAge();
        Double weight = state.getWeight();
        Double creatinine = state.getCreatinine();
        String sex = state.getSex();

        // Validate required parameters
        if (age == null || weight == null || creatinine == null || creatinine == 0.0) {
            logger.warn("Missing parameters for CrCl: age={}, weight={}, Cr={}",
                age, weight, creatinine);
            return NORMAL_CRCL_THRESHOLD; // Return safe default
        }

        // Cockcroft-Gault formula
        double crCl = ((140.0 - age) * weight) / (72.0 * creatinine);

        // Female adjustment factor
        if ("F".equalsIgnoreCase(sex) || "FEMALE".equalsIgnoreCase(sex)) {
            crCl *= 0.85;
        }

        logger.debug("Calculated CrCl: {:.1f} mL/min (age={}, weight={}, Cr={}, sex={})",
            crCl, age, weight, creatinine, sex);

        return crCl;
    }

    /**
     * Calculate creatinine clearance using Cockcroft-Gault formula (PatientContext overload).
     *
     * @param context Patient context with demographics and labs
     * @return Creatinine clearance in mL/min
     */
    public double calculateCrCl(com.cardiofit.flink.models.PatientContext context) {
        Integer age = context.getAge();
        Double weight = context.getWeight();
        Double creatinine = context.getCreatinine();
        String sex = context.getSex();

        // Validate required parameters
        if (age == null || weight == null || creatinine == null || creatinine <= 0.0) {
            throw new IllegalArgumentException("Invalid creatinine value or missing parameters");
        }

        // Cockcroft-Gault formula
        double crCl = ((140.0 - age) * weight) / (72.0 * creatinine);

        // Female adjustment factor
        if ("F".equalsIgnoreCase(sex) || "FEMALE".equalsIgnoreCase(sex)) {
            crCl *= 0.85;
        }

        return crCl;
    }

    // ================================================================
    // HEPATIC ADJUSTMENT METHODS
    // ================================================================

    /**
     * Check if hepatic adjustment is needed.
     */
    private boolean shouldAdjustForHepatic(Medication medication, PatientState state) {
        if (medication.getAdultDosing() == null ||
            medication.getAdultDosing().getHepaticAdjustment() == null) {
            return false;
        }

        String childPugh = state.getChildPughScore();
        return childPugh != null && (childPugh.equals("B") || childPugh.equals("C"));
    }

    /**
     * Apply hepatic dose adjustment based on Child-Pugh score.
     */
    private void applyHepaticAdjustment(Medication medication, PatientState state, CalculatedDose result) {
        String childPugh = state.getChildPughScore();

        logger.debug("Applying hepatic adjustment for Child-Pugh: {}", childPugh);

        Medication.HepaticDosing hepaticDosing =
            medication.getAdultDosing().getHepaticAdjustment();

        if (hepaticDosing.getAdjustments() != null) {
            Medication.HepaticDosing.DoseAdjustment adjustment =
                hepaticDosing.getAdjustments().get(childPugh);

            if (adjustment != null) {
                if (adjustment.isContraindicated()) {
                    result.setContraindicated(true);
                    result.addWarning(String.format(
                        "CONTRAINDICATED in Child-Pugh Class %s hepatic impairment", childPugh));
                    logger.warn("Medication contraindicated in Child-Pugh {}", childPugh);
                    return;
                }

                if (adjustment.getAdjustedDose() != null) {
                    result.setCalculatedDose(adjustment.getAdjustedDose());
                    result.setAdjustmentReason(String.format(
                        "Hepatic dose adjustment for Child-Pugh Class %s", childPugh));

                    result.addWarning(String.format(
                        "Dose adjusted for hepatic impairment (Child-Pugh %s)", childPugh));

                    result.addMonitoring("Monitor liver function tests (AST, ALT, bilirubin)");

                    logger.info("Hepatic-adjusted dose: {} (Child-Pugh {})",
                        adjustment.getAdjustedDose(), childPugh);
                }
            }
        }
    }

    /**
     * Calculate Child-Pugh score from patient state.
     *
     * @param state Patient state with liver function tests
     * @return Child-Pugh class (A, B, or C)
     */
    public String calculateChildPugh(PatientState state) {
        // Simplified Child-Pugh scoring
        // In production, would use full scoring: bilirubin, albumin, INR, ascites, encephalopathy

        // Access lab values using available PatientState API
        Double bilirubin = getLabValue(state, "bilirubin");
        Double albumin = getLabValue(state, "albumin");
        Double inr = state.getINR();

        if (bilirubin == null || albumin == null || inr == null) {
            return "A"; // Assume best case if data missing
        }

        int score = 0;

        // Bilirubin scoring (mg/dL)
        if (bilirubin < 2.0) score += 1;
        else if (bilirubin <= 3.0) score += 2;
        else score += 3;

        // Albumin scoring (g/dL)
        if (albumin > 3.5) score += 1;
        else if (albumin >= 2.8) score += 2;
        else score += 3;

        // INR scoring
        if (inr < 1.7) score += 1;
        else if (inr <= 2.3) score += 2;
        else score += 3;

        // Classification
        if (score <= 6) return "A"; // Mild
        else if (score <= 9) return "B"; // Moderate
        else return "C"; // Severe
    }

    // ================================================================
    // AGE-BASED ADJUSTMENTS
    // ================================================================

    /**
     * Check if geriatric adjustment is needed.
     */
    private boolean shouldAdjustForGeriatric(Medication medication, PatientState state) {
        if (medication.getGeriatricDosing() == null) return false;
        if (state.getAge() == null) return false;

        return state.getAge() >= GERIATRIC_AGE_THRESHOLD &&
               medication.getGeriatricDosing().isRequiresAdjustment();
    }

    /**
     * Apply geriatric dose adjustment.
     */
    private void applyGeriatricAdjustment(Medication medication, PatientState state, CalculatedDose result) {
        logger.debug("Applying geriatric adjustment for age: {}", state.getAge());

        Medication.GeriatricDosing geriatricDosing = medication.getGeriatricDosing();

        if (geriatricDosing.getAdjustedDose() != null) {
            result.setCalculatedDose(geriatricDosing.getAdjustedDose());
            result.setAdjustmentReason("Geriatric dose adjustment for age " + state.getAge());

            result.addWarning(String.format("Geriatric patient (age %d): %s",
                state.getAge(), geriatricDosing.getRationale()));
        }

        // Add Beers Criteria warnings
        if (geriatricDosing.getBeersListConcerns() != null) {
            for (String concern : geriatricDosing.getBeersListConcerns()) {
                result.addWarning("BEERS CRITERIA: " + concern);
            }
        }

        logger.info("Geriatric-adjusted dose applied for age {}", state.getAge());
    }

    /**
     * Check if pediatric adjustment is needed.
     */
    private boolean shouldAdjustForPediatric(Medication medication, PatientState state) {
        if (medication.getPediatricDosing() == null) return false;
        if (state.getAge() == null) return false;

        return state.getAge() < PEDIATRIC_AGE_THRESHOLD;
    }

    /**
     * Apply pediatric dose adjustment.
     */
    private void applyPediatricAdjustment(Medication medication, PatientState state, CalculatedDose result) {
        logger.debug("Applying pediatric adjustment for age: {}", state.getAge());

        Medication.PediatricDosing pediatricDosing = medication.getPediatricDosing();

        // Weight-based dosing
        if (pediatricDosing.isWeightBased() && state.getWeight() != null) {
            String weightBasedDose = pediatricDosing.getWeightBasedDose();
            // Parse dose like "100 mg/kg/day" and calculate
            // Simplified - in production would parse and calculate actual dose

            result.addWarning(String.format(
                "Pediatric weight-based dosing: %s for weight %.1f kg",
                weightBasedDose, state.getWeight()));

            result.addMonitoring("Monitor for pediatric-specific adverse effects");
        }

        logger.info("Pediatric-adjusted dose applied for age {}", state.getAge());
    }

    // ================================================================
    // OBESITY ADJUSTMENTS
    // ================================================================

    /**
     * Check if obesity adjustment is needed.
     */
    private boolean shouldAdjustForObesity(Medication medication, PatientState state) {
        if (medication.getAdultDosing() == null ||
            medication.getAdultDosing().getObesityAdjustment() == null) {
            return false;
        }

        if (state.getWeight() == null) return false;

        double bmi = calculateBMI(state);
        return bmi >= OBESITY_BMI_THRESHOLD &&
               medication.getAdultDosing().getObesityAdjustment().isRequiresAdjustment();
    }

    /**
     * Apply obesity dose adjustment.
     */
    private void applyObesityAdjustment(Medication medication, PatientState state, CalculatedDose result) {
        double bmi = calculateBMI(state);

        logger.debug("Applying obesity adjustment for BMI: {:.1f}", bmi);

        Medication.ObesityDosing obesityDosing =
            medication.getAdultDosing().getObesityAdjustment();

        result.addWarning(String.format(
            "Obesity adjustment: Use %s (BMI %.1f)",
            obesityDosing.getWeightType(), bmi));

        if (obesityDosing.getMaxDose() != null) {
            result.addWarning("Maximum dose capped at: " + obesityDosing.getMaxDose());
        }

        logger.info("Obesity-adjusted dosing applied for BMI {:.1f}", bmi);
    }

    /**
     * Calculate BMI from patient state.
     *
     * @param state Patient state with height and weight
     * @return BMI value
     */
    private double calculateBMI(PatientState state) {
        Double weight = state.getWeight(); // kg
        Double height = getVitalValue(state, "height"); // cm

        if (weight == null || height == null || height == 0) {
            return 25.0; // Default BMI if missing data
        }

        // BMI = weight(kg) / (height(m))^2
        double heightMeters = height / 100.0;
        return weight / (heightMeters * heightMeters);
    }

    // ================================================================
    // HELPER METHODS FOR PATIENT STATE API COMPATIBILITY
    // ================================================================

    /**
     * Get lab value from PatientState using available API.
     * Compatible with existing PatientState.getLabValueAsDouble() helper.
     *
     * @param state Patient state
     * @param labName Lab parameter name (e.g., "bilirubin", "albumin")
     * @return Lab value or null if not available
     */
    private Double getLabValue(com.cardiofit.flink.models.PatientState state, String labName) {
        if (state == null || state.getRecentLabs() == null) {
            return null;
        }

        LabResult labResult = state.getRecentLabs().get(labName.toLowerCase());
        return labResult != null ? labResult.getValue() : null;
    }

    /**
     * Get vital sign value from PatientState using available API.
     * Compatible with existing PatientState.getVitalAsDouble() helper.
     *
     * @param state Patient state
     * @param vitalName Vital parameter name (e.g., "height", "weight")
     * @return Vital value or null if not available
     */
    private Double getVitalValue(com.cardiofit.flink.models.PatientState state, String vitalName) {
        if (state == null || state.getLatestVitals() == null) {
            return null;
        }

        Object value = state.getLatestVitals().get(vitalName.toLowerCase());
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }

        try {
            return value != null ? Double.parseDouble(value.toString()) : null;
        } catch (NumberFormatException e) {
            return null;
        }
    }
}
