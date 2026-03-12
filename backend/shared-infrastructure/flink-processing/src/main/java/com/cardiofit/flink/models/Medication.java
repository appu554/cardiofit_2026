package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.Map;
import java.util.Objects;

/**
 * Medication information.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class Medication implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("name")
    private String name; // Medication name

    @JsonProperty("code")
    private String code; // RxNorm or other coding system

    @JsonProperty("dosage")
    private String dosage; // "10mg", "500mg", etc.

    @JsonProperty("route")
    private String route; // "oral", "IV", "subcutaneous"

    @JsonProperty("frequency")
    private String frequency; // "BID", "TID", "QD", etc.

    @JsonProperty("status")
    private String status; // "active", "stopped", "completed"

    @JsonProperty("startDate")
    private Long startDate;

    @JsonProperty("display")
    private String display; // Human-readable display name (e.g., "Telmisartan 40 mg Tablet")

    public Medication() {
    }

    public Medication(String name, String dosage, String frequency) {
        this.name = name;
        this.dosage = dosage;
        this.frequency = frequency;
        this.status = "active";
    }

    public static Medication fromPayload(Map<String, Object> payload) {
        Medication med = new Medication();

        med.name = (String) payload.get("medication_name");
        med.code = (String) payload.get("medication_code");
        med.dosage = (String) payload.get("dosage");
        med.route = (String) payload.get("route");
        med.frequency = (String) payload.get("frequency");
        med.status = (String) payload.getOrDefault("status", "active");

        Object startObj = payload.get("start_date");
        if (startObj instanceof Number) {
            med.startDate = ((Number) startObj).longValue();
        }

        return med;
    }

    // Getters and setters
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }

    public String getCode() { return code; }
    public void setCode(String code) { this.code = code; }

    public String getDosage() { return dosage; }
    public void setDosage(String dosage) { this.dosage = dosage; }

    public String getRoute() { return route; }
    public void setRoute(String route) { this.route = route; }

    public String getFrequency() { return frequency; }
    public void setFrequency(String frequency) { this.frequency = frequency; }

    public String getStatus() { return status; }
    public void setStatus(String status) { this.status = status; }

    public Long getStartDate() { return startDate; }
    public void setStartDate(Long startDate) { this.startDate = startDate; }

    public String getDisplay() { return display; }
    public void setDisplay(String display) { this.display = display; }

    // ============================================================
    // ALIAS METHODS (for Module 2 compatibility)
    // ============================================================

    /**
     * Get medication name (alias for getName()).
     * Used by Module 2 for medication interaction detection.
     */
    public String getMedicationName() {
        return name;
    }

    /**
     * Set medication name (alias for setName()).
     * Allows Jackson to deserialize "medicationName" field from Module 2 output.
     */
    @JsonProperty("medicationName")
    public void setMedicationName(String medicationName) {
        this.name = medicationName;
    }

    /**
     * Get medication start time (alias for getStartDate()).
     * Used by Module 2 for recent medication analysis.
     */
    public Long getStartTime() {
        return startDate;
    }

    /**
     * Set medication start time (alias for setStartDate()).
     * Allows Jackson to deserialize "startTime" field from Module 2 output.
     */
    @JsonProperty("startTime")
    public void setStartTime(Long startTime) {
        this.startDate = startTime;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        Medication that = (Medication) o;
        return Objects.equals(name, that.name) &&
                Objects.equals(dosage, that.dosage);
    }

    @Override
    public int hashCode() {
        return Objects.hash(name, dosage);
    }

    @Override
    public String toString() {
        return "Medication{" +
                "name='" + name + '\'' +
                ", dosage='" + dosage + '\'' +
                ", frequency='" + frequency + '\'' +
                '}';
    }
}
