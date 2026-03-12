package com.cardiofit.flink.models;

import java.io.Serializable;

/**
 * Vital sign variability alert for detecting unstable physiological states.
 *
 * High variability in vital signs can indicate:
 * - Autonomic dysfunction
 * - Sepsis or systemic inflammatory response
 * - Poor glycemic control (for glucose)
 * - Hemodynamic instability
 * - Increased mortality risk
 *
 * Coefficient of Variation (CV) = (Standard Deviation / Mean) × 100%
 *
 * Clinical thresholds vary by vital sign:
 * - Heart Rate CV: >15% may indicate cardiac instability
 * - Blood Pressure CV: >15% may indicate hemodynamic instability
 * - Glucose CV: >36% indicates high glycemic variability
 * - Temperature CV: >5% may indicate infection or inflammatory process
 */
public class VitalVariabilityAlert implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private String vitalSignName;
    private Double meanValue;
    private Double standardDeviation;
    private Double coefficientOfVariation;
    private String variabilityLevel;  // LOW, MODERATE, HIGH, CRITICAL
    private String clinicalSignificance;
    private Long timestamp;
    private Long windowStart;
    private Long windowEnd;

    public VitalVariabilityAlert() {}

    // Getters and Setters
    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public String getVitalSignName() {
        return vitalSignName;
    }

    public void setVitalSignName(String vitalSignName) {
        this.vitalSignName = vitalSignName;
    }

    public Double getMeanValue() {
        return meanValue;
    }

    public void setMeanValue(Double meanValue) {
        this.meanValue = meanValue;
    }

    public Double getStandardDeviation() {
        return standardDeviation;
    }

    public void setStandardDeviation(Double standardDeviation) {
        this.standardDeviation = standardDeviation;
    }

    public Double getCoefficientOfVariation() {
        return coefficientOfVariation;
    }

    public void setCoefficientOfVariation(Double coefficientOfVariation) {
        this.coefficientOfVariation = coefficientOfVariation;
    }

    public String getVariabilityLevel() {
        return variabilityLevel;
    }

    public void setVariabilityLevel(String variabilityLevel) {
        this.variabilityLevel = variabilityLevel;
    }

    public String getClinicalSignificance() {
        return clinicalSignificance;
    }

    public void setClinicalSignificance(String clinicalSignificance) {
        this.clinicalSignificance = clinicalSignificance;
    }

    public Long getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(Long timestamp) {
        this.timestamp = timestamp;
    }

    public Long getWindowStart() {
        return windowStart;
    }

    public void setWindowStart(Long windowStart) {
        this.windowStart = windowStart;
    }

    public Long getWindowEnd() {
        return windowEnd;
    }

    public void setWindowEnd(Long windowEnd) {
        this.windowEnd = windowEnd;
    }

    @Override
    public String toString() {
        return "VitalVariabilityAlert{" +
                "patientId='" + patientId + '\'' +
                ", vitalSignName='" + vitalSignName + '\'' +
                ", meanValue=" + meanValue +
                ", standardDeviation=" + standardDeviation +
                ", coefficientOfVariation=" + coefficientOfVariation +
                ", variabilityLevel='" + variabilityLevel + '\'' +
                ", clinicalSignificance='" + clinicalSignificance + '\'' +
                ", timestamp=" + timestamp +
                ", windowStart=" + windowStart +
                ", windowEnd=" + windowEnd +
                '}';
    }
}
