package com.clinicalsynthesis.validator.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.databind.JsonNode;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Positive;

import java.time.Instant;
import java.util.Objects;

/**
 * Device Reading Model
 * 
 * Represents raw device data consumed from the Global Outbox Service.
 * This model matches the exact structure of device data from the
 * existing PySpark pipeline to ensure compatibility.
 */
public class DeviceReading {
    
    @NotNull
    @JsonProperty("device_id")
    private String deviceId;
    
    @NotNull
    @Positive
    @JsonProperty("timestamp")
    private Long timestamp;
    
    @NotNull
    @JsonProperty("reading_type")
    private String readingType;
    
    @NotNull
    @JsonProperty("value")
    private Double value;
    
    @NotNull
    @JsonProperty("unit")
    private String unit;
    
    @JsonProperty("patient_id")
    private String patientId;
    
    @JsonProperty("metadata")
    private JsonNode metadata;
    
    @JsonProperty("vendor_info")
    private JsonNode vendorInfo;

    // Default constructor for Jackson
    public DeviceReading() {}

    // Constructor
    public DeviceReading(String deviceId, Long timestamp, String readingType, 
                        Double value, String unit, String patientId, 
                        JsonNode metadata, JsonNode vendorInfo) {
        this.deviceId = deviceId;
        this.timestamp = timestamp;
        this.readingType = readingType;
        this.value = value;
        this.unit = unit;
        this.patientId = patientId;
        this.metadata = metadata;
        this.vendorInfo = vendorInfo;
    }

    // Getters and Setters
    public String getDeviceId() {
        return deviceId;
    }

    public void setDeviceId(String deviceId) {
        this.deviceId = deviceId;
    }

    public Long getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(Long timestamp) {
        this.timestamp = timestamp;
    }

    public String getReadingType() {
        return readingType;
    }

    public void setReadingType(String readingType) {
        this.readingType = readingType;
    }

    public Double getValue() {
        return value;
    }

    public void setValue(Double value) {
        this.value = value;
    }

    public String getUnit() {
        return unit;
    }

    public void setUnit(String unit) {
        this.unit = unit;
    }

    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public JsonNode getMetadata() {
        return metadata;
    }

    public void setMetadata(JsonNode metadata) {
        this.metadata = metadata;
    }

    public JsonNode getVendorInfo() {
        return vendorInfo;
    }

    public void setVendorInfo(JsonNode vendorInfo) {
        this.vendorInfo = vendorInfo;
    }

    // Utility methods
    @JsonIgnore
    public Instant getTimestampAsInstant() {
        return timestamp != null ? Instant.ofEpochSecond(timestamp) : null;
    }

    public boolean isValidReading() {
        return deviceId != null && !deviceId.trim().isEmpty() &&
               timestamp != null && timestamp > 0 &&
               readingType != null && !readingType.trim().isEmpty() &&
               value != null && !value.isNaN() && !value.isInfinite() &&
               unit != null && !unit.trim().isEmpty();
    }

    public boolean isCriticalMedicalData() {
        if (readingType == null) return false;
        
        // Define critical medical data types (same logic as PySpark)
        return readingType.toLowerCase().contains("emergency") ||
               readingType.toLowerCase().contains("critical") ||
               readingType.toLowerCase().contains("alert") ||
               (readingType.equals("heart_rate") && value != null && (value < 40 || value > 150)) ||
               (readingType.equals("blood_pressure_systolic") && value != null && (value < 70 || value > 200)) ||
               (readingType.equals("oxygen_saturation") && value != null && value < 90);
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        DeviceReading that = (DeviceReading) o;
        return Objects.equals(deviceId, that.deviceId) &&
               Objects.equals(timestamp, that.timestamp) &&
               Objects.equals(readingType, that.readingType) &&
               Objects.equals(value, that.value) &&
               Objects.equals(unit, that.unit) &&
               Objects.equals(patientId, that.patientId);
    }

    @Override
    public int hashCode() {
        return Objects.hash(deviceId, timestamp, readingType, value, unit, patientId);
    }

    @Override
    public String toString() {
        return "DeviceReading{" +
               "deviceId='" + deviceId + '\'' +
               ", timestamp=" + timestamp +
               ", readingType='" + readingType + '\'' +
               ", value=" + value +
               ", unit='" + unit + '\'' +
               ", patientId='" + patientId + '\'' +
               '}';
    }
}
