package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.Map;

/**
 * Vital signs measurement.
 */
public class VitalSign implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("timestamp")
    private long timestamp;

    @JsonProperty("heartRate")
    private Double heartRate; // beats per minute

    @JsonProperty("bloodPressureSystolic")
    private Double bloodPressureSystolic; // mmHg

    @JsonProperty("bloodPressureDiastolic")
    private Double bloodPressureDiastolic; // mmHg

    @JsonProperty("temperature")
    private Double temperature; // Fahrenheit

    @JsonProperty("respiratoryRate")
    private Double respiratoryRate; // breaths per minute

    @JsonProperty("oxygenSaturation")
    private Double oxygenSaturation; // percentage

    public VitalSign() {
        this.timestamp = System.currentTimeMillis();
    }

    public static VitalSign fromPayload(Map<String, Object> payload) {
        VitalSign vital = new VitalSign();

        vital.heartRate = getDoubleValue(payload, "heart_rate");
        vital.temperature = getDoubleValue(payload, "temperature");
        vital.respiratoryRate = getDoubleValue(payload, "respiratory_rate");
        vital.oxygenSaturation = getDoubleValue(payload, "oxygen_saturation");

        // Parse blood pressure (can be string like "120/80" or separate values)
        Object bp = payload.get("blood_pressure");
        if (bp instanceof String) {
            String[] parts = ((String) bp).split("/");
            if (parts.length == 2) {
                try {
                    vital.bloodPressureSystolic = Double.parseDouble(parts[0].trim());
                    vital.bloodPressureDiastolic = Double.parseDouble(parts[1].trim());
                } catch (NumberFormatException e) {
                    // Invalid format, leave null
                }
            }
        } else {
            vital.bloodPressureSystolic = getDoubleValue(payload, "blood_pressure_systolic");
            vital.bloodPressureDiastolic = getDoubleValue(payload, "blood_pressure_diastolic");
        }

        return vital;
    }

    private static Double getDoubleValue(Map<String, Object> map, String key) {
        Object value = map.get(key);
        if (value == null) return null;
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }
        try {
            return Double.parseDouble(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    // Getters and setters
    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

    public Double getHeartRate() { return heartRate; }
    public void setHeartRate(Double heartRate) { this.heartRate = heartRate; }

    public Double getBloodPressureSystolic() { return bloodPressureSystolic; }
    public void setBloodPressureSystolic(Double bloodPressureSystolic) {
        this.bloodPressureSystolic = bloodPressureSystolic;
    }

    public Double getBloodPressureDiastolic() { return bloodPressureDiastolic; }
    public void setBloodPressureDiastolic(Double bloodPressureDiastolic) {
        this.bloodPressureDiastolic = bloodPressureDiastolic;
    }

    public Double getTemperature() { return temperature; }
    public void setTemperature(Double temperature) { this.temperature = temperature; }

    public Double getRespiratoryRate() { return respiratoryRate; }
    public void setRespiratoryRate(Double respiratoryRate) {
        this.respiratoryRate = respiratoryRate;
    }

    public Double getOxygenSaturation() { return oxygenSaturation; }
    public void setOxygenSaturation(Double oxygenSaturation) {
        this.oxygenSaturation = oxygenSaturation;
    }

    @Override
    public String toString() {
        return "VitalSign{" +
                "HR=" + heartRate +
                ", BP=" + bloodPressureSystolic + "/" + bloodPressureDiastolic +
                ", Temp=" + temperature +
                ", RR=" + respiratoryRate +
                ", SpO2=" + oxygenSaturation +
                '}';
    }
}
