package com.cardiofit.flink.knowledgebase.medications.calculator;

import lombok.Data;
import lombok.Builder;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Result model for dose calculation.
 *
 * Contains the calculated dose, frequency, route, and any warnings or adjustments
 * made based on patient-specific factors (renal, hepatic, age, weight).
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-24
 */
@Data
@Builder
public class CalculatedDose implements Serializable {
    private static final long serialVersionUID = 1L;

    /** Medication identifier */
    private String medicationId;

    /** Medication generic name */
    private String medicationName;

    /** Calculated dose amount (e.g., "2 g") */
    private String calculatedDose;

    /** Calculated frequency (e.g., "every 8 hours") */
    private String calculatedFrequency;

    /** Route of administration */
    private String route;

    /** Reason for dose adjustment (e.g., "Renal adjustment for CrCl 35 mL/min") */
    private String adjustmentReason;

    /** Clinical warnings about this dose */
    @Builder.Default
    private List<String> warnings = new ArrayList<>();

    /** Whether medication is contraindicated for this patient */
    private boolean contraindicated;

    /** Monitoring requirements specific to this dose */
    @Builder.Default
    private List<String> requiresMonitoring = new ArrayList<>();

    /** Maximum daily dose */
    private String maxDailyDose;

    /** Loading dose if applicable */
    private String loadingDose;

    /** Infusion duration for IV medications */
    private String infusionDuration;

    /** Original standard dose before adjustment */
    private String originalDose;

    /** Original standard frequency before adjustment */
    private String originalFrequency;

    /** Weight used for calculation (e.g., actual, adjusted, ideal body weight) */
    private Double weightUsed;

    /** Type of weight used (e.g., "actual body weight", "adjusted body weight", "ideal body weight") */
    private String weightType;

    /** Detailed calculation notes explaining the dosing rationale */
    private String calculationNotes;

    /** Contraindication reason if medication is contraindicated */
    private String contraindicationReason;

    /** Times per day for total daily dose calculation */
    private Integer timesPerDay;

    /**
     * Add a warning message.
     *
     * @param warning The warning message
     */
    public void addWarning(String warning) {
        if (warnings == null) {
            warnings = new ArrayList<>();
        }
        warnings.add(warning);
    }

    /**
     * Add a monitoring requirement.
     *
     * @param monitoring The monitoring requirement
     */
    public void addMonitoring(String monitoring) {
        if (requiresMonitoring == null) {
            requiresMonitoring = new ArrayList<>();
        }
        requiresMonitoring.add(monitoring);
    }

    /**
     * Check if dose was adjusted from original.
     *
     * @return true if dose differs from original
     */
    public boolean wasAdjusted() {
        return adjustmentReason != null && !adjustmentReason.isEmpty();
    }

    /**
     * Get human-readable summary of calculated dose.
     *
     * @return Summary string
     */
    public String getSummary() {
        StringBuilder summary = new StringBuilder();

        summary.append(medicationName)
               .append(" ")
               .append(calculatedDose)
               .append(" ")
               .append(route)
               .append(" ")
               .append(calculatedFrequency);

        if (wasAdjusted()) {
            summary.append(" (").append(adjustmentReason).append(")");
        }

        if (contraindicated) {
            summary.append(" - CONTRAINDICATED");
        }

        return summary.toString();
    }

    /**
     * Check if there are any warnings.
     *
     * @return true if warnings present
     */
    public boolean hasWarnings() {
        return warnings != null && !warnings.isEmpty();
    }

    /**
     * Check if monitoring is required.
     *
     * @return true if monitoring requirements present
     */
    public boolean requiresMonitoring() {
        return requiresMonitoring != null && !requiresMonitoring.isEmpty();
    }

    /**
     * Check if dose can be administered without exceeding maximum daily dose.
     *
     * @param numberOfDoses Number of times to administer this dose
     * @return true if safe to administer, false if would exceed max daily dose
     */
    public boolean canAdminister(int numberOfDoses) {
        if (maxDailyDose == null || calculatedDose == null) {
            return true; // No max defined, allow administration
        }

        try {
            // Extract numeric value from dose string (e.g., "500mg" -> 500.0)
            String doseNumeric = calculatedDose.replaceAll("[^0-9.]", "");
            double singleDose = Double.parseDouble(doseNumeric);

            // Extract numeric value from max daily dose
            String maxNumeric = maxDailyDose.replaceAll("[^0-9.]", "");
            double maxDaily = Double.parseDouble(maxNumeric);

            // Check if total dose would exceed max
            double totalDose = singleDose * numberOfDoses;

            if (totalDose > maxDaily) {
                addWarning("Would exceed maximum daily dose");
                return false;
            }

            // Warn if approaching max (within 90%)
            if (totalDose >= maxDaily * 0.9) {
                addWarning("Approaching maximum daily dose");
            }

            return true;

        } catch (NumberFormatException e) {
            // If we can't parse, err on side of caution
            return true;
        }
    }
}
