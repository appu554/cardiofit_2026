package com.clinicalsynthesis.validator.model;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.time.Instant;
import java.util.Objects;

/**
 * Validation Result Model
 * 
 * Represents the result of medical validation for a device reading.
 * This matches the validation result structure from the PySpark pipeline.
 */
public class ValidationResult {
    
    @JsonProperty("device_id")
    private String deviceId;
    
    @JsonProperty("reading_type")
    private String readingType;
    
    @JsonProperty("value")
    private Double value;
    
    @JsonProperty("is_valid")
    private Boolean valid;
    
    @JsonProperty("alert_level")
    private String alertLevel; // normal, low, high, critical, emergency, invalid, error
    
    @JsonProperty("requires_attention")
    private Boolean requiresAttention;
    
    @JsonProperty("is_critical_data")
    private Boolean criticalData;
    
    @JsonProperty("validation_message")
    private String validationMessage;
    
    @JsonProperty("validation_timestamp")
    private Long validationTimestamp;
    
    @JsonProperty("validation_stage")
    private String validationStage = "stage1-validator-enricher";
    
    @JsonProperty("age_group")
    private String ageGroup;
    
    @JsonProperty("validation_rules_version")
    private String validationRulesVersion = "1.0.0";

    // Default constructor
    public ValidationResult() {
        this.validationTimestamp = Instant.now().getEpochSecond();
    }

    // Getters and Setters
    public String getDeviceId() {
        return deviceId;
    }

    public void setDeviceId(String deviceId) {
        this.deviceId = deviceId;
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

    public Boolean getValid() {
        return valid;
    }

    public void setValid(Boolean valid) {
        this.valid = valid;
    }

    public String getAlertLevel() {
        return alertLevel;
    }

    public void setAlertLevel(String alertLevel) {
        this.alertLevel = alertLevel;
    }

    public Boolean getRequiresAttention() {
        return requiresAttention;
    }

    public void setRequiresAttention(Boolean requiresAttention) {
        this.requiresAttention = requiresAttention;
    }

    public Boolean getCriticalData() {
        return criticalData;
    }

    public void setCriticalData(Boolean criticalData) {
        this.criticalData = criticalData;
    }

    public String getValidationMessage() {
        return validationMessage;
    }

    public void setValidationMessage(String validationMessage) {
        this.validationMessage = validationMessage;
    }

    public Long getValidationTimestamp() {
        return validationTimestamp;
    }

    public void setValidationTimestamp(Long validationTimestamp) {
        this.validationTimestamp = validationTimestamp;
    }

    public String getValidationStage() {
        return validationStage;
    }

    public void setValidationStage(String validationStage) {
        this.validationStage = validationStage;
    }

    public String getAgeGroup() {
        return ageGroup;
    }

    public void setAgeGroup(String ageGroup) {
        this.ageGroup = ageGroup;
    }

    public String getValidationRulesVersion() {
        return validationRulesVersion;
    }

    public void setValidationRulesVersion(String validationRulesVersion) {
        this.validationRulesVersion = validationRulesVersion;
    }

    // Utility methods
    public Instant getValidationTimestampAsInstant() {
        return validationTimestamp != null ? Instant.ofEpochSecond(validationTimestamp) : null;
    }

    public boolean isEmergency() {
        return "emergency".equals(alertLevel) || "critical".equals(alertLevel);
    }

    public boolean isNormal() {
        return "normal".equals(alertLevel);
    }

    public boolean isAbnormal() {
        return "low".equals(alertLevel) || "high".equals(alertLevel);
    }

    public boolean isInvalid() {
        return "invalid".equals(alertLevel) || "error".equals(alertLevel);
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        ValidationResult that = (ValidationResult) o;
        return Objects.equals(deviceId, that.deviceId) &&
               Objects.equals(readingType, that.readingType) &&
               Objects.equals(value, that.value) &&
               Objects.equals(validationTimestamp, that.validationTimestamp);
    }

    @Override
    public int hashCode() {
        return Objects.hash(deviceId, readingType, value, validationTimestamp);
    }

    @Override
    public String toString() {
        return "ValidationResult{" +
               "deviceId='" + deviceId + '\'' +
               ", readingType='" + readingType + '\'' +
               ", value=" + value +
               ", valid=" + valid +
               ", alertLevel='" + alertLevel + '\'' +
               ", requiresAttention=" + requiresAttention +
               ", criticalData=" + criticalData +
               ", validationMessage='" + validationMessage + '\'' +
               '}';
    }
}
