package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Blood Pressure Structured Type (V2 Schema - Type Change Example).
 *
 * <p><b>Migration Context:</b> Previously blood pressure was stored as a String "120/80".
 * Now it's a structured type with separate systolic and diastolic values for proper
 * clinical calculations and alerting.
 *
 * <p><b>Clinical Benefits of Structured Type:</b>
 * <ul>
 *   <li>Enable hypotension alerts (SBP < 90 mmHg)</li>
 *   <li>Enable hypertension alerts (SBP > 180 mmHg)</li>
 *   <li>Calculate MAP (Mean Arterial Pressure) = diastolic + (1/3)(systolic - diastolic)</li>
 *   <li>Detect pulse pressure abnormalities (systolic - diastolic)</li>
 *   <li>Support clinical scoring (MEWS, NEWS2) requiring numeric BP values</li>
 * </ul>
 *
 * @see VitalReadingSerializer Handles String → BloodPressure type conversion
 */
public class BloodPressure implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Systolic blood pressure in mmHg.
     * Normal range: 90-120 mmHg
     */
    @JsonProperty("systolic")
    private int systolic;

    /**
     * Diastolic blood pressure in mmHg.
     * Normal range: 60-80 mmHg
     */
    @JsonProperty("diastolic")
    private int diastolic;

    /**
     * Blood pressure measurement method.
     * AUSCULTATORY = manual cuff, OSCILLOMETRIC = automatic cuff, INVASIVE = arterial line
     */
    @JsonProperty("method")
    private MeasurementMethod method;

    /**
     * Measurement timestamp (epoch milliseconds).
     */
    @JsonProperty("timestamp")
    private long timestamp;

    // Constructors
    public BloodPressure() {
        this.timestamp = System.currentTimeMillis();
        this.method = MeasurementMethod.OSCILLOMETRIC; // Default to automatic
    }

    public BloodPressure(int systolic, int diastolic) {
        this();
        this.systolic = systolic;
        this.diastolic = diastolic;
    }

    public BloodPressure(int systolic, int diastolic, MeasurementMethod method) {
        this(systolic, diastolic);
        this.method = method;
    }

    // ========================================================================================
    // CLINICAL CALCULATIONS
    // ========================================================================================

    /**
     * Calculate Mean Arterial Pressure (MAP).
     * Formula: MAP = diastolic + (1/3)(systolic - diastolic)
     *
     * Clinical significance: MAP < 65 mmHg indicates inadequate tissue perfusion.
     */
    public double calculateMAP() {
        return diastolic + ((systolic - diastolic) / 3.0);
    }

    /**
     * Calculate Pulse Pressure.
     * Formula: PP = systolic - diastolic
     *
     * Clinical significance:
     * - PP < 25 mmHg: Low (shock, heart failure)
     * - PP > 60 mmHg: High (arterial stiffness, aortic regurgitation)
     */
    public int calculatePulsePressure() {
        return systolic - diastolic;
    }

    /**
     * Check if blood pressure is in normal range.
     */
    public boolean isNormal() {
        return systolic >= 90 && systolic <= 120 &&
               diastolic >= 60 && diastolic <= 80;
    }

    /**
     * Check if patient is hypotensive (low blood pressure).
     */
    public boolean isHypotensive() {
        return systolic < 90;
    }

    /**
     * Check if patient is hypertensive (high blood pressure).
     */
    public boolean isHypertensive() {
        return systolic > 140 || diastolic > 90;
    }

    /**
     * Check if hypertensive crisis (emergency).
     */
    public boolean isHypertensiveCrisis() {
        return systolic > 180 || diastolic > 120;
    }

    // ========================================================================================
    // MEASUREMENT METHOD ENUM
    // ========================================================================================

    public enum MeasurementMethod {
        /** Manual cuff measurement by clinician (most accurate) */
        AUSCULTATORY,

        /** Automatic oscillometric device (most common) */
        OSCILLOMETRIC,

        /** Invasive arterial line (gold standard for critical patients) */
        INVASIVE
    }

    // ========================================================================================
    // GETTERS AND SETTERS
    // ========================================================================================

    public int getSystolic() { return systolic; }
    public void setSystolic(int systolic) { this.systolic = systolic; }

    public int getDiastolic() { return diastolic; }
    public void setDiastolic(int diastolic) { this.diastolic = diastolic; }

    public MeasurementMethod getMethod() { return method; }
    public void setMethod(MeasurementMethod method) { this.method = method; }

    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

    @Override
    public String toString() {
        return systolic + "/" + diastolic + " mmHg (MAP: " +
               String.format("%.1f", calculateMAP()) + ")";
    }

    /**
     * Format as traditional string "120/80".
     */
    public String toTraditionalString() {
        return systolic + "/" + diastolic;
    }

    /**
     * Parse blood pressure from traditional string format "120/80".
     * Used for migration from V1 string-based storage.
     *
     * @param bpString Blood pressure in "systolic/diastolic" format
     * @return Parsed BloodPressure object
     */
    public static BloodPressure parse(String bpString) {
        if (bpString == null || bpString.trim().isEmpty()) {
            return null;
        }

        try {
            String[] parts = bpString.split("/");
            if (parts.length != 2) {
                throw new IllegalArgumentException(
                    "Invalid BP format: " + bpString + " (expected 'systolic/diastolic')"
                );
            }

            int systolic = Integer.parseInt(parts[0].trim());
            int diastolic = Integer.parseInt(parts[1].trim());

            return new BloodPressure(systolic, diastolic);
        } catch (NumberFormatException e) {
            throw new IllegalArgumentException(
                "Invalid BP format: " + bpString + " (non-numeric values)", e
            );
        }
    }
}
