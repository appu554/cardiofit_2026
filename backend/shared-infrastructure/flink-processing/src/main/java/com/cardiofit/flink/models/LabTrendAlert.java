package com.cardiofit.flink.models;

import java.io.Serializable;

/**
 * Lab value trend analysis alert for detecting significant clinical changes.
 *
 * Supports KDIGO AKI criteria for creatinine and glucose variability analysis.
 *
 * KDIGO AKI Staging Criteria:
 * - Stage 1: SCr increase ≥0.3 mg/dL within 48h OR increase ≥1.5x baseline within 7 days
 * - Stage 2: SCr increase ≥2.0x baseline
 * - Stage 3: SCr increase ≥3.0x baseline OR SCr ≥4.0 mg/dL
 *
 * Glucose Variability:
 * - CV (Coefficient of Variation) >36% indicates high glycemic variability
 * - Associated with increased mortality in critically ill patients
 */
public class LabTrendAlert implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private String labName;
    private Double firstValue;
    private Double lastValue;
    private Double absoluteChange;
    private Double percentChange;
    private Double trendSlope;
    private String trendDirection;
    private String akiStage;  // For creatinine: AKI_STAGE_1, AKI_STAGE_2, AKI_STAGE_3, NO_AKI
    private Double meanValue;
    private Double standardDeviation;
    private Double coefficientOfVariation;  // For glucose: CV >36% indicates high variability
    private String interpretation;
    private Long timestamp;
    private Long windowStart;
    private Long windowEnd;

    public LabTrendAlert() {}

    // Getters and Setters
    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public String getLabName() {
        return labName;
    }

    public void setLabName(String labName) {
        this.labName = labName;
    }

    public Double getFirstValue() {
        return firstValue;
    }

    public void setFirstValue(Double firstValue) {
        this.firstValue = firstValue;
    }

    public Double getLastValue() {
        return lastValue;
    }

    public void setLastValue(Double lastValue) {
        this.lastValue = lastValue;
    }

    public Double getAbsoluteChange() {
        return absoluteChange;
    }

    public void setAbsoluteChange(Double absoluteChange) {
        this.absoluteChange = absoluteChange;
    }

    public Double getPercentChange() {
        return percentChange;
    }

    public void setPercentChange(Double percentChange) {
        this.percentChange = percentChange;
    }

    public Double getTrendSlope() {
        return trendSlope;
    }

    public void setTrendSlope(Double trendSlope) {
        this.trendSlope = trendSlope;
    }

    public String getTrendDirection() {
        return trendDirection;
    }

    public void setTrendDirection(String trendDirection) {
        this.trendDirection = trendDirection;
    }

    public String getAkiStage() {
        return akiStage;
    }

    public void setAkiStage(String akiStage) {
        this.akiStage = akiStage;
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

    public String getInterpretation() {
        return interpretation;
    }

    public void setInterpretation(String interpretation) {
        this.interpretation = interpretation;
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
        return "LabTrendAlert{" +
                "patientId='" + patientId + '\'' +
                ", labName='" + labName + '\'' +
                ", firstValue=" + firstValue +
                ", lastValue=" + lastValue +
                ", absoluteChange=" + absoluteChange +
                ", percentChange=" + percentChange +
                ", trendSlope=" + trendSlope +
                ", trendDirection='" + trendDirection + '\'' +
                ", akiStage='" + akiStage + '\'' +
                ", meanValue=" + meanValue +
                ", standardDeviation=" + standardDeviation +
                ", coefficientOfVariation=" + coefficientOfVariation +
                ", interpretation='" + interpretation + '\'' +
                ", timestamp=" + timestamp +
                ", windowStart=" + windowStart +
                ", windowEnd=" + windowEnd +
                '}';
    }
}
