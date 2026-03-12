package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Individual vital sign reading for use in serialization and state storage.
 *
 * This class represents a single vital signs measurement captured at a specific point in time,
 * used within VitalsHistory circular buffer and state backend serialization.
 */
public class VitalReading implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("timestamp")
    private long timestamp;

    @JsonProperty("heartRate")
    private Integer heartRate; // beats per minute

    @JsonProperty("bloodPressureSystolic")
    private Integer bloodPressureSystolic; // mmHg

    @JsonProperty("bloodPressureDiastolic")
    private Integer bloodPressureDiastolic; // mmHg

    @JsonProperty("temperature")
    private Double temperature; // Fahrenheit

    @JsonProperty("respiratoryRate")
    private Integer respiratoryRate; // breaths per minute

    @JsonProperty("oxygenSaturation")
    private Integer oxygenSaturation; // SpO2 percentage

    @JsonProperty("painScore")
    private Integer painScore; // 0-10 scale

    @JsonProperty("consciousnessLevel")
    private String consciousnessLevel; // alert, verbal, pain, unresponsive (AVPU scale)

    // Default constructor
    public VitalReading() {
        this.timestamp = System.currentTimeMillis();
    }

    /**
     * Constructor with timestamp.
     */
    public VitalReading(long timestamp) {
        this.timestamp = timestamp;
    }

    // Getters and Setters
    public long getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(long timestamp) {
        this.timestamp = timestamp;
    }

    public Integer getHeartRate() {
        return heartRate;
    }

    public void setHeartRate(Integer heartRate) {
        this.heartRate = heartRate;
    }

    public Integer getBloodPressureSystolic() {
        return bloodPressureSystolic;
    }

    public void setBloodPressureSystolic(Integer bloodPressureSystolic) {
        this.bloodPressureSystolic = bloodPressureSystolic;
    }

    public Integer getBloodPressureDiastolic() {
        return bloodPressureDiastolic;
    }

    public void setBloodPressureDiastolic(Integer bloodPressureDiastolic) {
        this.bloodPressureDiastolic = bloodPressureDiastolic;
    }

    public Double getTemperature() {
        return temperature;
    }

    public void setTemperature(Double temperature) {
        this.temperature = temperature;
    }

    public Integer getRespiratoryRate() {
        return respiratoryRate;
    }

    public void setRespiratoryRate(Integer respiratoryRate) {
        this.respiratoryRate = respiratoryRate;
    }

    public Integer getOxygenSaturation() {
        return oxygenSaturation;
    }

    public void setOxygenSaturation(Integer oxygenSaturation) {
        this.oxygenSaturation = oxygenSaturation;
    }

    public Integer getPainScore() {
        return painScore;
    }

    public void setPainScore(Integer painScore) {
        this.painScore = painScore;
    }

    public String getConsciousnessLevel() {
        return consciousnessLevel;
    }

    public void setConsciousnessLevel(String consciousnessLevel) {
        this.consciousnessLevel = consciousnessLevel;
    }

    @Override
    public String toString() {
        return "VitalReading{" +
                "timestamp=" + timestamp +
                ", heartRate=" + heartRate +
                ", bloodPressure=" + bloodPressureSystolic + "/" + bloodPressureDiastolic +
                ", temperature=" + temperature +
                ", respiratoryRate=" + respiratoryRate +
                ", spO2=" + oxygenSaturation +
                '}';
    }
}
