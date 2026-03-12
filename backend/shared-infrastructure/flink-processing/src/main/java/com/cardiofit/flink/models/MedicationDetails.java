package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Medication Details - Comprehensive medication information with dosing
 *
 * Detailed medication information including dosing calculations,
 * administration instructions, and safety parameters. Used in
 * therapeutic clinical actions.
 *
 * @author CardioFit Platform - Module 3
 * @version 1.0
 * @since 2025-10-20
 */
public class MedicationDetails implements Serializable {
    private static final long serialVersionUID = 1L;

    // Medication Identity
    @JsonProperty("name")
    private String name;

    @JsonProperty("brand_name")
    private String brandName;

    @JsonProperty("drug_class")
    private String drugClass;

    // Dosing
    @JsonProperty("dose_calculation_method")
    private String doseCalculationMethod;  // fixed, weight_based, renal_adjusted, bsa_based

    @JsonProperty("calculated_dose")
    private double calculatedDose;

    @JsonProperty("dose_unit")
    private String doseUnit;  // mg, g, units, mcg

    @JsonProperty("dose_range")
    private String doseRange;

    // Dosing Parameters
    @JsonProperty("patient_weight")
    private Double patientWeight;  // kg

    @JsonProperty("patient_egfr")
    private Double patientEgfr;  // mL/min/1.73m²

    @JsonProperty("renal_adjustment_applied")
    private String renalAdjustmentApplied;

    // Administration
    @JsonProperty("route")
    private String route;  // IV, PO, IM, SC, inhaled, topical

    @JsonProperty("administration_instructions")
    private String administrationInstructions;

    @JsonProperty("frequency")
    private String frequency;  // q6h, q12h, daily, BID, TID, PRN

    @JsonProperty("duration")
    private String duration;  // 7 days, Until cultures negative, Indefinite

    // Safety Parameters
    @JsonProperty("max_single_dose")
    private String maxSingleDose;

    @JsonProperty("max_daily_dose")
    private String maxDailyDose;

    @JsonProperty("black_box_warnings")
    private List<String> blackBoxWarnings;

    @JsonProperty("adverse_effects")
    private List<String> adverseEffects;

    // Monitoring
    @JsonProperty("lab_monitoring")
    private List<String> labMonitoring;

    @JsonProperty("therapeutic_range")
    private String therapeuticRange;

    // Default constructor
    public MedicationDetails() {
        this.blackBoxWarnings = new ArrayList<>();
        this.adverseEffects = new ArrayList<>();
        this.labMonitoring = new ArrayList<>();
    }

    // Constructor with essential fields
    public MedicationDetails(String name, double calculatedDose, String doseUnit, String route, String frequency) {
        this();
        this.name = name;
        this.calculatedDose = calculatedDose;
        this.doseUnit = doseUnit;
        this.route = route;
        this.frequency = frequency;
    }

    // Getters and Setters
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }

    public String getBrandName() { return brandName; }
    public void setBrandName(String brandName) { this.brandName = brandName; }

    public String getDrugClass() { return drugClass; }
    public void setDrugClass(String drugClass) { this.drugClass = drugClass; }

    public String getDoseCalculationMethod() { return doseCalculationMethod; }
    public void setDoseCalculationMethod(String doseCalculationMethod) {
        this.doseCalculationMethod = doseCalculationMethod;
    }

    public double getCalculatedDose() { return calculatedDose; }
    public void setCalculatedDose(double calculatedDose) { this.calculatedDose = calculatedDose; }

    public String getDoseUnit() { return doseUnit; }
    public void setDoseUnit(String doseUnit) { this.doseUnit = doseUnit; }

    public String getDoseRange() { return doseRange; }
    public void setDoseRange(String doseRange) { this.doseRange = doseRange; }

    public Double getPatientWeight() { return patientWeight; }
    public void setPatientWeight(Double patientWeight) { this.patientWeight = patientWeight; }

    public Double getPatientEgfr() { return patientEgfr; }
    public void setPatientEgfr(Double patientEgfr) { this.patientEgfr = patientEgfr; }

    public String getRenalAdjustmentApplied() { return renalAdjustmentApplied; }
    public void setRenalAdjustmentApplied(String renalAdjustmentApplied) {
        this.renalAdjustmentApplied = renalAdjustmentApplied;
    }

    public String getRoute() { return route; }
    public void setRoute(String route) { this.route = route; }

    public String getAdministrationInstructions() { return administrationInstructions; }
    public void setAdministrationInstructions(String administrationInstructions) {
        this.administrationInstructions = administrationInstructions;
    }

    public String getFrequency() { return frequency; }
    public void setFrequency(String frequency) { this.frequency = frequency; }

    public String getDuration() { return duration; }
    public void setDuration(String duration) { this.duration = duration; }

    public String getMaxSingleDose() { return maxSingleDose; }
    public void setMaxSingleDose(String maxSingleDose) { this.maxSingleDose = maxSingleDose; }

    public String getMaxDailyDose() { return maxDailyDose; }
    public void setMaxDailyDose(String maxDailyDose) { this.maxDailyDose = maxDailyDose; }

    public List<String> getBlackBoxWarnings() { return blackBoxWarnings; }
    public void setBlackBoxWarnings(List<String> blackBoxWarnings) { this.blackBoxWarnings = blackBoxWarnings; }

    public List<String> getAdverseEffects() { return adverseEffects; }
    public void setAdverseEffects(List<String> adverseEffects) { this.adverseEffects = adverseEffects; }

    public List<String> getLabMonitoring() { return labMonitoring; }
    public void setLabMonitoring(List<String> labMonitoring) { this.labMonitoring = labMonitoring; }

    public String getTherapeuticRange() { return therapeuticRange; }
    public void setTherapeuticRange(String therapeuticRange) { this.therapeuticRange = therapeuticRange; }

    // Utility methods

    /**
     * Check if medication requires renal adjustment
     */
    public boolean requiresRenalAdjustment() {
        return "renal_adjusted".equals(doseCalculationMethod);
    }

    /**
     * Check if medication is weight-based dosing
     */
    public boolean isWeightBased() {
        return "weight_based".equals(doseCalculationMethod);
    }

    /**
     * Check if medication has black box warnings
     */
    public boolean hasBlackBoxWarnings() {
        return blackBoxWarnings != null && !blackBoxWarnings.isEmpty();
    }

    /**
     * Check if medication requires lab monitoring
     */
    public boolean requiresLabMonitoring() {
        return labMonitoring != null && !labMonitoring.isEmpty();
    }

    /**
     * Get formatted dose string
     */
    public String getFormattedDose() {
        return calculatedDose + " " + doseUnit;
    }

    @Override
    public String toString() {
        return "MedicationDetails{" +
            "name='" + name + '\'' +
            ", calculatedDose=" + calculatedDose +
            ", doseUnit='" + doseUnit + '\'' +
            ", route='" + route + '\'' +
            ", frequency='" + frequency + '\'' +
            ", duration='" + duration + '\'' +
            '}';
    }
}
