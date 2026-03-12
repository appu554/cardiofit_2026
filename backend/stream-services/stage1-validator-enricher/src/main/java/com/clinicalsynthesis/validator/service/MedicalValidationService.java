package com.clinicalsynthesis.validator.service;

import com.clinicalsynthesis.validator.model.DeviceReading;
import com.clinicalsynthesis.validator.model.ValidationResult;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

import java.util.HashMap;
import java.util.Map;

/**
 * Medical Validation Service
 * 
 * Ports the exact same medical validation logic from the PySpark ETL pipeline.
 * This service validates device readings using the same physiological ranges
 * and medical rules that were implemented in business_logic/transformations.py
 */
@Service
public class MedicalValidationService {
    
    private static final Logger logger = LoggerFactory.getLogger(MedicalValidationService.class);
    
    // Normal physiological ranges (same as PySpark implementation)
    private static final Map<String, double[]> NORMAL_RANGES = new HashMap<>();
    private static final Map<String, double[]> CRITICAL_RANGES = new HashMap<>();
    
    static {
        // Normal ranges (from PySpark transformations.py)
        NORMAL_RANGES.put("heart_rate", new double[]{60, 100});
        NORMAL_RANGES.put("blood_pressure_systolic", new double[]{90, 140});
        NORMAL_RANGES.put("blood_pressure_diastolic", new double[]{60, 90});
        NORMAL_RANGES.put("blood_glucose", new double[]{70, 140});
        NORMAL_RANGES.put("temperature", new double[]{97.0, 99.5}); // Fahrenheit
        NORMAL_RANGES.put("oxygen_saturation", new double[]{95, 100});
        NORMAL_RANGES.put("respiratory_rate", new double[]{12, 20});
        NORMAL_RANGES.put("weight", new double[]{50, 300}); // kg
        
        // Critical ranges (from PySpark transformations.py)
        CRITICAL_RANGES.put("heart_rate", new double[]{40, 150});
        CRITICAL_RANGES.put("blood_pressure_systolic", new double[]{70, 180});
        CRITICAL_RANGES.put("blood_pressure_diastolic", new double[]{40, 110});
        CRITICAL_RANGES.put("blood_glucose", new double[]{50, 200});
        CRITICAL_RANGES.put("temperature", new double[]{95.0, 104.0}); // Fahrenheit
        CRITICAL_RANGES.put("oxygen_saturation", new double[]{85, 100});
        CRITICAL_RANGES.put("respiratory_rate", new double[]{8, 30});
        CRITICAL_RANGES.put("weight", new double[]{30, 500}); // kg
    }
    
    /**
     * Validates a device reading using the same logic as PySpark pipeline
     * 
     * @param deviceReading The device reading to validate
     * @return ValidationResult containing validation status and details
     */
    public ValidationResult validateDeviceReading(DeviceReading deviceReading) {
        logger.debug("Validating device reading: {}", deviceReading);
        
        ValidationResult result = new ValidationResult();
        result.setDeviceId(deviceReading.getDeviceId());
        result.setReadingType(deviceReading.getReadingType());
        result.setValue(deviceReading.getValue());
        result.setValidationTimestamp(System.currentTimeMillis());
        
        try {
            // Step 1: Basic field validation (same as PySpark)
            if (!isValidBasicFields(deviceReading)) {
                result.setValid(false);
                result.setAlertLevel("invalid");
                result.setValidationMessage("Missing required fields");
                result.setRequiresAttention(true);
                return result;
            }
            
            // Step 2: Physiological range validation (same as PySpark)
            String alertLevel = validatePhysiologicalRange(
                deviceReading.getReadingType(), 
                deviceReading.getValue()
            );
            
            result.setAlertLevel(alertLevel);
            result.setValid(!alertLevel.equals("invalid"));
            result.setRequiresAttention(alertLevel.equals("critical") || alertLevel.equals("emergency"));
            
            // Step 3: Critical medical data detection (same as PySpark)
            boolean isCritical = deviceReading.isCriticalMedicalData();
            result.setCriticalData(isCritical);
            
            // Step 4: Set validation message
            result.setValidationMessage(generateValidationMessage(alertLevel, isCritical));
            
            logger.debug("Validation result: {} - {}", alertLevel, result.getValidationMessage());
            
        } catch (Exception e) {
            logger.error("Error validating device reading: {}", e.getMessage(), e);
            result.setValid(false);
            result.setAlertLevel("error");
            result.setValidationMessage("Validation error: " + e.getMessage());
            result.setRequiresAttention(true);
        }
        
        return result;
    }
    
    /**
     * Basic field validation (same logic as PySpark isValidReading)
     */
    private boolean isValidBasicFields(DeviceReading reading) {
        return reading.getDeviceId() != null && !reading.getDeviceId().trim().isEmpty() &&
               reading.getTimestamp() != null && reading.getTimestamp() > 0 &&
               reading.getReadingType() != null && !reading.getReadingType().trim().isEmpty() &&
               reading.getValue() != null && !reading.getValue().isNaN() && !reading.getValue().isInfinite() &&
               reading.getUnit() != null && !reading.getUnit().trim().isEmpty();
    }
    
    /**
     * Physiological range validation (same logic as PySpark validate_physiological_range)
     */
    private String validatePhysiologicalRange(String readingType, Double value) {
        if (readingType == null || value == null) {
            return "invalid";
        }
        
        // Get ranges for this reading type
        double[] normalRange = NORMAL_RANGES.get(readingType.toLowerCase());
        double[] criticalRange = CRITICAL_RANGES.get(readingType.toLowerCase());
        
        if (normalRange == null) {
            logger.warn("No validation ranges found for reading type: {}", readingType);
            return "normal"; // Unknown reading type, assume normal
        }
        
        // Check critical ranges first (same as PySpark)
        if (criticalRange != null) {
            if (value < criticalRange[0] || value > criticalRange[1]) {
                return "critical";
            }
        }
        
        // Check normal ranges
        if (value < normalRange[0]) {
            return "low";
        } else if (value > normalRange[1]) {
            return "high";
        } else {
            return "normal";
        }
    }
    
    /**
     * Generate validation message based on alert level
     */
    private String generateValidationMessage(String alertLevel, boolean isCritical) {
        switch (alertLevel) {
            case "normal":
                return isCritical ? "Normal reading (flagged as critical data type)" : "Normal reading";
            case "low":
                return "Reading below normal range";
            case "high":
                return "Reading above normal range";
            case "critical":
                return "CRITICAL: Reading outside safe physiological range";
            case "emergency":
                return "EMERGENCY: Immediate medical attention required";
            case "invalid":
                return "Invalid reading data";
            case "error":
                return "Validation error occurred";
            default:
                return "Unknown validation status";
        }
    }
    
    /**
     * Check if reading type requires special handling (same as PySpark)
     */
    public boolean isEmergencyReading(DeviceReading reading) {
        if (reading.getReadingType() == null || reading.getValue() == null) {
            return false;
        }
        
        String type = reading.getReadingType().toLowerCase();
        Double value = reading.getValue();
        
        // Emergency thresholds (same as PySpark isCriticalMedicalData)
        switch (type) {
            case "heart_rate":
                return value < 40 || value > 150;
            case "blood_pressure_systolic":
                return value < 70 || value > 200;
            case "oxygen_saturation":
                return value < 90;
            case "temperature":
                return value < 95.0 || value > 104.0; // Fahrenheit
            case "blood_glucose":
                return value < 50 || value > 300;
            default:
                return type.contains("emergency") || type.contains("critical") || type.contains("alert");
        }
    }
    
    /**
     * Get age group classification (same as PySpark get_age_group)
     */
    public String getAgeGroup(Double ageYears) {
        if (ageYears == null) {
            return "adult"; // Default to adult if age unknown
        }
        
        if (ageYears < 1.0/12.0) { // Less than 1 month
            return "newborn";
        } else if (ageYears < 1) {
            return "infant";
        } else if (ageYears < 3) {
            return "toddler";
        } else if (ageYears < 6) {
            return "preschooler";
        } else if (ageYears < 12) {
            return "school_age";
        } else if (ageYears < 18) {
            return "adolescent";
        } else if (ageYears < 65) {
            return "adult";
        } else {
            return "elderly";
        }
    }
    
    /**
     * Check if validation rules are loaded and current
     */
    public boolean isValidationSystemHealthy() {
        return NORMAL_RANGES.size() > 0 && CRITICAL_RANGES.size() > 0;
    }
}
