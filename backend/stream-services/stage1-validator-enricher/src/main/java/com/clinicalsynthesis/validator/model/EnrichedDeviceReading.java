package com.clinicalsynthesis.validator.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.databind.JsonNode;

import java.time.Instant;
import java.util.Objects;

/**
 * Enriched Device Reading Model
 * 
 * Represents device data that has been validated and enriched with patient context.
 * This is the output of Stage 1 and input to Stage 2 (Storage Fan-Out).
 */
public class EnrichedDeviceReading {
    
    // Original device reading data
    @JsonProperty("device_id")
    private String deviceId;
    
    @JsonProperty("timestamp")
    private Long timestamp;
    
    @JsonProperty("reading_type")
    private String readingType;
    
    @JsonProperty("value")
    private Double value;
    
    @JsonProperty("unit")
    private String unit;
    
    @JsonProperty("patient_id")
    private String patientId;
    
    @JsonProperty("metadata")
    private JsonNode metadata;
    
    @JsonProperty("vendor_info")
    private JsonNode vendorInfo;
    
    // Enriched patient context data
    @JsonProperty("patient_context")
    private PatientContext patientContext;
    
    // Validation metadata
    @JsonProperty("validation_timestamp")
    private Long validationTimestamp;
    
    @JsonProperty("validation_stage")
    private String validationStage = "stage1-validator-enricher";
    
    @JsonProperty("is_critical_data")
    private Boolean isCriticalData;

    // Default constructor
    public EnrichedDeviceReading() {
        this.validationTimestamp = Instant.now().getEpochSecond();
    }

    // Constructor from DeviceReading
    public EnrichedDeviceReading(DeviceReading deviceReading) {
        this.deviceId = deviceReading.getDeviceId();
        this.timestamp = deviceReading.getTimestamp();
        this.readingType = deviceReading.getReadingType();
        this.value = deviceReading.getValue();
        this.unit = deviceReading.getUnit();
        this.patientId = deviceReading.getPatientId();
        this.metadata = deviceReading.getMetadata();
        this.vendorInfo = deviceReading.getVendorInfo();
        this.validationTimestamp = Instant.now().getEpochSecond();
        this.isCriticalData = deviceReading.isCriticalMedicalData();
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

    public PatientContext getPatientContext() {
        return patientContext;
    }

    public void setPatientContext(PatientContext patientContext) {
        this.patientContext = patientContext;
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

    public Boolean getIsCriticalData() {
        return isCriticalData;
    }

    public void setIsCriticalData(Boolean isCriticalData) {
        this.isCriticalData = isCriticalData;
    }

    // Utility methods
    public boolean hasPatientContext() {
        return patientContext != null && patientContext.getPatientId() != null;
    }

    @JsonIgnore
    public Instant getTimestampAsInstant() {
        return timestamp != null ? Instant.ofEpochSecond(timestamp) : null;
    }

    @JsonIgnore
    public Instant getValidationTimestampAsInstant() {
        return validationTimestamp != null ? Instant.ofEpochSecond(validationTimestamp) : null;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        EnrichedDeviceReading that = (EnrichedDeviceReading) o;
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
        return "EnrichedDeviceReading{" +
               "deviceId='" + deviceId + '\'' +
               ", timestamp=" + timestamp +
               ", readingType='" + readingType + '\'' +
               ", value=" + value +
               ", unit='" + unit + '\'' +
               ", patientId='" + patientId + '\'' +
               ", hasPatientContext=" + hasPatientContext() +
               ", isCriticalData=" + isCriticalData +
               '}';
    }

    /**
     * Patient Context nested class
     */
    public static class PatientContext {
        @JsonProperty("patient_id")
        private String patientId;
        
        @JsonProperty("patient_name")
        private String patientName;
        
        @JsonProperty("age")
        private Integer age;
        
        @JsonProperty("gender")
        private String gender;
        
        @JsonProperty("medical_conditions")
        private JsonNode medicalConditions;
        
        @JsonProperty("cache_timestamp")
        private Long cacheTimestamp;

        // Default constructor
        public PatientContext() {}

        // Getters and Setters
        public String getPatientId() {
            return patientId;
        }

        public void setPatientId(String patientId) {
            this.patientId = patientId;
        }

        public String getPatientName() {
            return patientName;
        }

        public void setPatientName(String patientName) {
            this.patientName = patientName;
        }

        public Integer getAge() {
            return age;
        }

        public void setAge(Integer age) {
            this.age = age;
        }

        public String getGender() {
            return gender;
        }

        public void setGender(String gender) {
            this.gender = gender;
        }

        public JsonNode getMedicalConditions() {
            return medicalConditions;
        }

        public void setMedicalConditions(JsonNode medicalConditions) {
            this.medicalConditions = medicalConditions;
        }

        public Long getCacheTimestamp() {
            return cacheTimestamp;
        }

        public void setCacheTimestamp(Long cacheTimestamp) {
            this.cacheTimestamp = cacheTimestamp;
        }

        @Override
        public String toString() {
            return "PatientContext{" +
                   "patientId='" + patientId + '\'' +
                   ", patientName='" + patientName + '\'' +
                   ", age=" + age +
                   ", gender='" + gender + '\'' +
                   '}';
        }
    }
}
