package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * Vital signs payload for GenericEvent wrapper.
 *
 * Contains all physiological measurements from patient monitoring devices.
 * Supports both structured fields and flexible key-value map for vendor-specific vitals.
 *
 * Clinical Parameters:
 * - Heart Rate (bpm): Normal 60-100, Tachycardia >100, Bradycardia <60
 * - Blood Pressure (mmHg): Normal <120/80, Hypertensive ≥130/80
 * - SpO2 (%): Normal ≥95%, Hypoxemia <92%, Critical <85%
 * - Respiratory Rate (bpm): Normal 12-20, Tachypnea >20, Bradypnea <12
 * - Temperature (°C): Normal 36.1-37.2, Fever >38, Hypothermia <35
 */
public class VitalsPayload implements Serializable {
    private static final long serialVersionUID = 1L;

    // Core vital signs - structured fields for common parameters

    @JsonProperty("heartRate")
    private Integer heartRate; // bpm

    @JsonProperty("systolicBP")
    private Integer systolicBP; // mmHg

    @JsonProperty("diastolicBP")
    private Integer diastolicBP; // mmHg

    @JsonProperty("oxygenSaturation")
    private Integer oxygenSaturation; // SpO2 %

    @JsonProperty("respiratoryRate")
    private Integer respiratoryRate; // breaths per minute

    @JsonProperty("temperature")
    private Double temperature; // Celsius

    @JsonProperty("temperatureUnit")
    private String temperatureUnit; // "C" or "F"

    // Extended vitals - flexible map for additional measurements
    // Examples: "meanArterialPressure", "pulseWidth", "perfusionIndex", "etCO2"
    @JsonProperty("additionalVitals")
    private Map<String, Object> additionalVitals;

    /**
     * Device metadata
     */
    @JsonProperty("deviceId")
    private String deviceId;

    @JsonProperty("deviceType")
    private String deviceType; // "philips_intellivue", "ge_carescape", "welch_allyn"

    @JsonProperty("measurementTimestamp")
    private long measurementTimestamp;

    /**
     * Quality indicators
     */
    @JsonProperty("signalQuality")
    private String signalQuality; // "good", "fair", "poor", "artifact"

    @JsonProperty("alarmState")
    private String alarmState; // "none", "technical", "physiological"

    public VitalsPayload() {
        this.additionalVitals = new HashMap<>();
        this.measurementTimestamp = System.currentTimeMillis();
        this.temperatureUnit = "C"; // Default to Celsius
    }

    // Getters and setters

    public Integer getHeartRate() {
        return heartRate;
    }

    public void setHeartRate(Integer heartRate) {
        this.heartRate = heartRate;
    }

    public Integer getSystolicBP() {
        return systolicBP;
    }

    public void setSystolicBP(Integer systolicBP) {
        this.systolicBP = systolicBP;
    }

    public Integer getDiastolicBP() {
        return diastolicBP;
    }

    public void setDiastolicBP(Integer diastolicBP) {
        this.diastolicBP = diastolicBP;
    }

    public Integer getOxygenSaturation() {
        return oxygenSaturation;
    }

    public void setOxygenSaturation(Integer oxygenSaturation) {
        this.oxygenSaturation = oxygenSaturation;
    }

    public Integer getRespiratoryRate() {
        return respiratoryRate;
    }

    public void setRespiratoryRate(Integer respiratoryRate) {
        this.respiratoryRate = respiratoryRate;
    }

    public Double getTemperature() {
        return temperature;
    }

    public void setTemperature(Double temperature) {
        this.temperature = temperature;
    }

    public String getTemperatureUnit() {
        return temperatureUnit;
    }

    public void setTemperatureUnit(String temperatureUnit) {
        this.temperatureUnit = temperatureUnit;
    }

    public Map<String, Object> getAdditionalVitals() {
        return additionalVitals;
    }

    public void setAdditionalVitals(Map<String, Object> additionalVitals) {
        this.additionalVitals = additionalVitals;
    }

    public String getDeviceId() {
        return deviceId;
    }

    public void setDeviceId(String deviceId) {
        this.deviceId = deviceId;
    }

    public String getDeviceType() {
        return deviceType;
    }

    public void setDeviceType(String deviceType) {
        this.deviceType = deviceType;
    }

    public long getMeasurementTimestamp() {
        return measurementTimestamp;
    }

    public void setMeasurementTimestamp(long measurementTimestamp) {
        this.measurementTimestamp = measurementTimestamp;
    }

    public String getSignalQuality() {
        return signalQuality;
    }

    public void setSignalQuality(String signalQuality) {
        this.signalQuality = signalQuality;
    }

    public String getAlarmState() {
        return alarmState;
    }

    public void setAlarmState(String alarmState) {
        this.alarmState = alarmState;
    }

    /**
     * Get blood pressure in "systolic/diastolic" format
     */
    public String getBloodPressure() {
        if (systolicBP != null && diastolicBP != null) {
            return systolicBP + "/" + diastolicBP;
        }
        return null;
    }

    /**
     * Convert to legacy vitals map format for backward compatibility
     */
    public Map<String, Object> toVitalsMap() {
        Map<String, Object> vitals = new HashMap<>();

        if (heartRate != null) vitals.put("heartrate", heartRate);
        if (systolicBP != null) vitals.put("systolicbloodpressure", systolicBP);
        if (diastolicBP != null) vitals.put("diastolicbloodpressure", diastolicBP);
        if (oxygenSaturation != null) vitals.put("oxygensaturation", oxygenSaturation);
        if (respiratoryRate != null) vitals.put("respiratoryrate", respiratoryRate);
        if (temperature != null) vitals.put("temperature", temperature);

        // Add blood pressure string
        if (systolicBP != null && diastolicBP != null) {
            vitals.put("bloodpressure", systolicBP + "/" + diastolicBP);
        }

        // Merge additional vitals
        if (additionalVitals != null) {
            vitals.putAll(additionalVitals);
        }

        return vitals;
    }

    @Override
    public String toString() {
        return "VitalsPayload{" +
                "HR=" + heartRate +
                ", BP=" + getBloodPressure() +
                ", SpO2=" + oxygenSaturation +
                ", RR=" + respiratoryRate +
                ", Temp=" + temperature + temperatureUnit +
                ", device=" + deviceType +
                ", quality=" + signalQuality +
                '}';
    }
}
