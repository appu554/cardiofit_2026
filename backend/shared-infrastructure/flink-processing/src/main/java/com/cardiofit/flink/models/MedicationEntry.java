package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Medication Entry - Active medication in patient's medication list
 *
 * Represents a currently active medication with dosing, frequency,
 * route, and administration details. Used for drug interaction checking
 * and contraindication validation.
 *
 * @author CardioFit Platform - Module 3
 * @version 1.0
 * @since 2025-10-20
 */
public class MedicationEntry implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("medication_id")
    private String medicationId;

    @JsonProperty("medication_name")
    private String medicationName;

    @JsonProperty("generic_name")
    private String genericName;

    @JsonProperty("brand_name")
    private String brandName;

    @JsonProperty("drug_class")
    private String drugClass;

    @JsonProperty("dose")
    private String dose;

    @JsonProperty("dose_unit")
    private String doseUnit;

    @JsonProperty("route")
    private String route;  // IV, PO, IM, SC, etc.

    @JsonProperty("frequency")
    private String frequency;  // q6h, q12h, daily, BID, TID, PRN

    @JsonProperty("start_date")
    private long startDate;

    @JsonProperty("end_date")
    private Long endDate;  // null if ongoing

    @JsonProperty("prescribing_provider")
    private String prescribingProvider;

    @JsonProperty("indication")
    private String indication;

    @JsonProperty("status")
    private String status;  // ACTIVE, HELD, DISCONTINUED

    // Default constructor
    public MedicationEntry() {}

    // Constructor with essential fields
    public MedicationEntry(String medicationName, String dose, String route, String frequency) {
        this.medicationName = medicationName;
        this.dose = dose;
        this.route = route;
        this.frequency = frequency;
        this.status = "ACTIVE";
        this.startDate = System.currentTimeMillis();
    }

    // Getters and Setters
    public String getMedicationId() { return medicationId; }
    public void setMedicationId(String medicationId) { this.medicationId = medicationId; }

    public String getMedicationName() { return medicationName; }
    public void setMedicationName(String medicationName) { this.medicationName = medicationName; }

    public String getGenericName() { return genericName; }
    public void setGenericName(String genericName) { this.genericName = genericName; }

    public String getBrandName() { return brandName; }
    public void setBrandName(String brandName) { this.brandName = brandName; }

    public String getDrugClass() { return drugClass; }
    public void setDrugClass(String drugClass) { this.drugClass = drugClass; }

    public String getDose() { return dose; }
    public void setDose(String dose) { this.dose = dose; }

    public String getDoseUnit() { return doseUnit; }
    public void setDoseUnit(String doseUnit) { this.doseUnit = doseUnit; }

    public String getRoute() { return route; }
    public void setRoute(String route) { this.route = route; }

    public String getFrequency() { return frequency; }
    public void setFrequency(String frequency) { this.frequency = frequency; }

    public long getStartDate() { return startDate; }
    public void setStartDate(long startDate) { this.startDate = startDate; }

    public Long getEndDate() { return endDate; }
    public void setEndDate(Long endDate) { this.endDate = endDate; }

    public String getPrescribingProvider() { return prescribingProvider; }
    public void setPrescribingProvider(String prescribingProvider) { this.prescribingProvider = prescribingProvider; }

    public String getIndication() { return indication; }
    public void setIndication(String indication) { this.indication = indication; }

    public String getStatus() { return status; }
    public void setStatus(String status) { this.status = status; }

    // Utility methods

    /**
     * Check if medication is currently active
     */
    public boolean isActive() {
        return "ACTIVE".equals(status);
    }

    /**
     * Check if medication matches given name (case-insensitive)
     */
    public boolean matchesMedication(String searchName) {
        if (searchName == null) return false;

        String search = searchName.toLowerCase();
        return (medicationName != null && medicationName.toLowerCase().contains(search)) ||
               (genericName != null && genericName.toLowerCase().contains(search)) ||
               (brandName != null && brandName.toLowerCase().contains(search));
    }

    /**
     * Check if medication is in specific drug class
     */
    public boolean isInDrugClass(String className) {
        return drugClass != null && drugClass.toLowerCase().contains(className.toLowerCase());
    }

    @Override
    public String toString() {
        return "MedicationEntry{" +
            "medicationName='" + medicationName + '\'' +
            ", dose='" + dose + '\'' +
            ", route='" + route + '\'' +
            ", frequency='" + frequency + '\'' +
            ", status='" + status + '\'' +
            '}';
    }
}
