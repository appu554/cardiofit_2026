package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.models.RawEvent;
import com.cardiofit.flink.models.CanonicalEvent;

import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

/**
 * Builder utility for creating test clinical events.
 *
 * Provides factory methods for common clinical event types with reasonable defaults.
 * Reduces test boilerplate and ensures consistent test data structure.
 */
public class ClinicalEventBuilder {

    /**
     * Create a vital signs RawEvent for testing Module 1.
     *
     * @param patientId Patient identifier
     * @param timestamp Event timestamp in milliseconds
     * @param heartRate Heart rate in bpm
     * @param systolicBP Systolic blood pressure in mmHg
     * @return Configured RawEvent
     */
    public static RawEvent vitalsRaw(String patientId, long timestamp, int heartRate, int systolicBP) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", heartRate);
        payload.put("systolic_bp", systolicBP);
        payload.put("diastolic_bp", systolicBP - 40); // Reasonable default
        payload.put("temperature", 37.0);
        payload.put("respiratory_rate", 16);
        payload.put("oxygen_saturation", 98);

        Map<String, String> metadata = new HashMap<>();
        metadata.put("source", "Test Harness");
        metadata.put("location", "Test Ward");
        metadata.put("device_id", "TEST-DEVICE-001");

        return RawEvent.builder()
            .id(UUID.randomUUID().toString())
            .patientId(patientId)
            .type("vital_signs")
            .eventTime(timestamp)
            .payload(payload)
            .metadata(metadata)
            .build();
    }

    /**
     * Create a vital signs CanonicalEvent for testing Module 2.
     *
     * @param patientId Patient identifier
     * @param timestamp Event timestamp in milliseconds
     * @param heartRate Heart rate in bpm
     * @param systolicBP Systolic blood pressure in mmHg
     * @return Configured CanonicalEvent
     */
    public static CanonicalEvent vitalsCanonical(String patientId, long timestamp, int heartRate, int systolicBP) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", heartRate);
        payload.put("systolic_bp", systolicBP);
        payload.put("diastolic_bp", systolicBP - 40);
        payload.put("temperature", 37.0);

        CanonicalEvent.EventMetadata metadata =
            new CanonicalEvent.EventMetadata("Test Harness", "Test Ward", "TEST-DEVICE-001");

        return CanonicalEvent.builder()
            .id(UUID.randomUUID().toString())
            .patientId(patientId)
            .encounterId(null)
            .eventType(EventType.VITAL_SIGNS)
            .eventTime(timestamp)
            .payload(payload)
            .metadata(metadata)
            .build();
    }

    /**
     * Create a medication order RawEvent for testing.
     *
     * @param patientId Patient identifier
     * @param timestamp Event timestamp in milliseconds
     * @param drugCode Drug code identifier
     * @param drugName Human-readable drug name
     * @return Configured RawEvent
     */
    public static RawEvent medicationOrderRaw(String patientId, long timestamp, String drugCode, String drugName) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("drug_code", drugCode);
        payload.put("drug_name", drugName);
        payload.put("dose", "10mg");
        payload.put("route", "PO");
        payload.put("frequency", "BID");

        Map<String, String> metadata = new HashMap<>();
        metadata.put("source", "Test Harness");
        metadata.put("location", "Test Pharmacy");
        metadata.put("device_id", "TEST-PHARMACY-001");

        return RawEvent.builder()
            .id(UUID.randomUUID().toString())
            .patientId(patientId)
            .type("medication_order")
            .eventTime(timestamp)
            .payload(payload)
            .metadata(metadata)
            .build();
    }

    /**
     * Create a medication order CanonicalEvent for testing.
     *
     * @param patientId Patient identifier
     * @param timestamp Event timestamp in milliseconds
     * @param drugCode Drug code identifier
     * @param drugName Human-readable drug name
     * @return Configured CanonicalEvent
     */
    public static CanonicalEvent medicationOrderCanonical(String patientId, long timestamp,
                                                          String drugCode, String drugName) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("drug_code", drugCode);
        payload.put("drug_name", drugName);
        payload.put("dose", "10mg");
        payload.put("route", "PO");

        CanonicalEvent.EventMetadata metadata =
            new CanonicalEvent.EventMetadata("Test Harness", "Test Pharmacy", "TEST-PHARMACY-001");

        return CanonicalEvent.builder()
            .id(UUID.randomUUID().toString())
            .patientId(patientId)
            .encounterId(null)
            .eventType(EventType.MEDICATION)
            .eventTime(timestamp)
            .payload(payload)
            .metadata(metadata)
            .build();
    }

    /**
     * Create a lab result RawEvent for testing.
     *
     * @param patientId Patient identifier
     * @param timestamp Event timestamp in milliseconds
     * @param labType Type of lab test (e.g., "lactate", "creatinine")
     * @param value Numeric result value
     * @return Configured RawEvent
     */
    public static RawEvent labResultRaw(String patientId, long timestamp, String labType, double value) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("lab_type", labType);
        payload.put("value", value);
        payload.put("unit", getUnitForLab(labType));
        payload.put("reference_range", getReferenceRangeForLab(labType));

        Map<String, String> metadata = new HashMap<>();
        metadata.put("source", "Test Harness");
        metadata.put("location", "Test Lab");
        metadata.put("device_id", "TEST-LAB-001");

        return RawEvent.builder()
            .id(UUID.randomUUID().toString())
            .patientId(patientId)
            .type("lab_result")
            .eventTime(timestamp)
            .payload(payload)
            .metadata(metadata)
            .build();
    }

    /**
     * Create a lab result CanonicalEvent for testing.
     *
     * @param patientId Patient identifier
     * @param timestamp Event timestamp in milliseconds
     * @param labType Type of lab test
     * @param value Numeric result value
     * @return Configured CanonicalEvent
     */
    public static CanonicalEvent labResultCanonical(String patientId, long timestamp,
                                                     String labType, double value) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("lab_type", labType);
        payload.put("value", value);
        payload.put("unit", getUnitForLab(labType));

        CanonicalEvent.EventMetadata metadata =
            new CanonicalEvent.EventMetadata("Test Harness", "Test Lab", "TEST-LAB-001");

        return CanonicalEvent.builder()
            .id(UUID.randomUUID().toString())
            .patientId(patientId)
            .encounterId(null)
            .eventType(EventType.LAB_RESULT)
            .eventTime(timestamp)
            .payload(payload)
            .metadata(metadata)
            .build();
    }

    /**
     * Create an admission RawEvent for testing.
     *
     * @param patientId Patient identifier
     * @param timestamp Event timestamp in milliseconds
     * @return Configured RawEvent
     */
    public static RawEvent admissionRaw(String patientId, long timestamp) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("location", "Ward-3B");
        payload.put("admission_type", "emergency");
        payload.put("department", "Cardiology");

        Map<String, String> metadata = new HashMap<>();
        metadata.put("source", "Test Harness");
        metadata.put("location", "Test Admissions");
        metadata.put("device_id", "TEST-ADMISSION-001");

        return RawEvent.builder()
            .id(UUID.randomUUID().toString())
            .patientId(patientId)
            .type("admission")
            .eventTime(timestamp)
            .payload(payload)
            .metadata(metadata)
            .build();
    }

    /**
     * Create an admission CanonicalEvent for testing.
     *
     * @param patientId Patient identifier
     * @param timestamp Event timestamp in milliseconds
     * @return Configured CanonicalEvent
     */
    public static CanonicalEvent admissionCanonical(String patientId, long timestamp) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("location", "Ward-3B");
        payload.put("admission_type", "emergency");

        CanonicalEvent.EventMetadata metadata =
            new CanonicalEvent.EventMetadata("Test Harness", "Test Admissions", "TEST-ADMISSION-001");

        return CanonicalEvent.builder()
            .id(UUID.randomUUID().toString())
            .patientId(patientId)
            .encounterId(null)
            .eventType(EventType.ADMISSION)
            .eventTime(timestamp)
            .payload(payload)
            .metadata(metadata)
            .build();
    }

    /**
     * Get standard unit for lab type.
     */
    private static String getUnitForLab(String labType) {
        Map<String, String> units = new HashMap<>();
        units.put("lactate", "mmol/L");
        units.put("creatinine", "mg/dL");
        units.put("glucose", "mg/dL");
        units.put("sodium", "mEq/L");
        units.put("potassium", "mEq/L");
        return units.getOrDefault(labType, "units");
    }

    /**
     * Get standard reference range for lab type.
     */
    private static String getReferenceRangeForLab(String labType) {
        Map<String, String> ranges = new HashMap<>();
        ranges.put("lactate", "0.5-2.0");
        ranges.put("creatinine", "0.6-1.2");
        ranges.put("glucose", "70-100");
        ranges.put("sodium", "135-145");
        ranges.put("potassium", "3.5-5.0");
        return ranges.getOrDefault(labType, "N/A");
    }
}
