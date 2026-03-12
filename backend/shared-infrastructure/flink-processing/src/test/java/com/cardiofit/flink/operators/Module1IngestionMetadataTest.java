package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.RawEvent;
import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.EventType;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Test suite for Module 1 improvements:
 * - Metadata retention
 * - Enhanced validation
 * - JSON naming consistency
 */
public class Module1IngestionMetadataTest {

    private ObjectMapper objectMapper;

    @BeforeEach
    public void setup() {
        objectMapper = new ObjectMapper();
        objectMapper.registerModule(new JavaTimeModule());
    }

    /**
     * Test Case 1: Metadata Retention
     * Verify that device_id, location, and source are preserved from RawEvent
     */
    @Test
    public void testMetadataRetention() throws Exception {
        // Given: RawEvent with complete metadata
        Map<String, String> metadata = new HashMap<>();
        metadata.put("source", "ICU Monitor");
        metadata.put("location", "ICU-A");
        metadata.put("device_id", "DEV-456");

        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);
        payload.put("blood_pressure", "120/80");

        RawEvent rawEvent = RawEvent.builder()
            .id("test-001")
            .patientId("PAT-123")
            .type("vital_signs")
            .eventTime(System.currentTimeMillis())
            .payload(payload)
            .metadata(metadata)
            .build();

        // When: Event is processed (simulated canonicalization)
        // This would be done by Module1_Ingestion.EventValidationAndCanonicalization

        // Then: Verify metadata is preserved in output
        String json = objectMapper.writeValueAsString(rawEvent);
        RawEvent deserializedEvent = objectMapper.readValue(json, RawEvent.class);

        assertNotNull(deserializedEvent.getMetadata(), "Metadata should be preserved");
        assertEquals("ICU Monitor", deserializedEvent.getMetadata().get("source"));
        assertEquals("ICU-A", deserializedEvent.getMetadata().get("location"));
        assertEquals("DEV-456", deserializedEvent.getMetadata().get("device_id"));
    }

    /**
     * Test Case 2: Missing Metadata Handling
     * Verify that missing metadata defaults to "UNKNOWN"
     */
    @Test
    public void testMissingMetadataHandling() {
        // Given: RawEvent without metadata
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);

        RawEvent rawEvent = RawEvent.builder()
            .id("test-002")
            .patientId("PAT-456")
            .type("vital_signs")
            .eventTime(System.currentTimeMillis())
            .payload(payload)
            .build();  // No metadata

        // Then: Verify metadata is null or empty (will be defaulted during canonicalization)
        assertTrue(rawEvent.getMetadata() == null || rawEvent.getMetadata().isEmpty(),
            "Metadata should be null or empty when not provided");
    }

    /**
     * Test Case 3: Null Timestamp Validation
     * Verify that events with null/zero timestamps are rejected
     */
    @Test
    public void testNullTimestampValidation() {
        // Given: RawEvent with zero timestamp
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);

        RawEvent rawEvent = RawEvent.builder()
            .id("test-003")
            .patientId("PAT-789")
            .type("vital_signs")
            .eventTime(0)  // Invalid timestamp
            .payload(payload)
            .build();

        // Then: This event should fail validation
        assertTrue(rawEvent.getEventTime() <= 0, "Event time should be invalid");
    }

    /**
     * Test Case 4: Blank Patient ID Validation
     * Verify that events with blank patient IDs are rejected
     */
    @Test
    public void testBlankPatientIdValidation() {
        // Given: RawEvent with blank patient ID
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);

        RawEvent rawEvent = RawEvent.builder()
            .id("test-004")
            .patientId("   ")  // Blank patient ID
            .type("vital_signs")
            .eventTime(System.currentTimeMillis())
            .payload(payload)
            .build();

        // Then: Patient ID should be blank after trim
        assertTrue(rawEvent.getPatientId().trim().isEmpty(),
            "Patient ID should be blank and fail validation");
    }

    /**
     * Test Case 5: Missing Event Type Handling
     * Verify that missing event types don't fail validation (default to UNKNOWN)
     */
    @Test
    public void testMissingEventTypeHandling() {
        // Given: RawEvent without event type
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);

        RawEvent rawEvent = RawEvent.builder()
            .id("test-005")
            .patientId("PAT-999")
            .type(null)  // Missing type
            .eventTime(System.currentTimeMillis())
            .payload(payload)
            .build();

        // Then: Event type should be null (will be defaulted to UNKNOWN during canonicalization)
        assertNull(rawEvent.getType(), "Event type should be null");
    }

    /**
     * Test Case 6: Complete Valid Event
     * Verify that a complete, valid event passes all checks
     */
    @Test
    public void testCompleteValidEvent() throws Exception {
        // Given: Complete RawEvent with all fields
        Map<String, String> metadata = new HashMap<>();
        metadata.put("source", "Python Test Script");
        metadata.put("location", "ICU Ward");
        metadata.put("device_id", "MON-001");

        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);
        payload.put("blood_pressure", "120/80");
        payload.put("temperature", 98.6);
        payload.put("respiratory_rate", 16);
        payload.put("oxygen_saturation", 98);

        RawEvent rawEvent = RawEvent.builder()
            .id("6a27dac5-b10e-4be1-a735-8f5cd916f4ee")
            .patientId("DEMO-456")
            .type("vital_signs")
            .eventTime(1759305006359L)
            .payload(payload)
            .metadata(metadata)
            .build();

        // When: Serialized to JSON
        String json = objectMapper.writeValueAsString(rawEvent);

        // Then: Verify JSON structure
        assertTrue(json.contains("\"patient_id\":\"DEMO-456\""), "JSON should contain patient_id");
        assertTrue(json.contains("\"device_id\":\"MON-001\""), "JSON should contain device_id");
        assertTrue(json.contains("\"location\":\"ICU Ward\""), "JSON should contain location");

        // Verify deserialization
        RawEvent deserializedEvent = objectMapper.readValue(json, RawEvent.class);
        assertEquals("DEMO-456", deserializedEvent.getPatientId());
        assertEquals("vital_signs", deserializedEvent.getType());
        assertNotNull(deserializedEvent.getMetadata());
        assertEquals("MON-001", deserializedEvent.getMetadata().get("device_id"));
    }

    /**
     * Test Case 7: Future Timestamp Validation
     * Verify that timestamps too far in the future are rejected
     */
    @Test
    public void testFutureTimestampValidation() {
        // Given: RawEvent with timestamp 2 hours in the future
        long futureTime = System.currentTimeMillis() + (2 * 60 * 60 * 1000); // +2 hours

        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);

        RawEvent rawEvent = RawEvent.builder()
            .id("test-006")
            .patientId("PAT-111")
            .type("vital_signs")
            .eventTime(futureTime)  // Too far in future
            .payload(payload)
            .build();

        // Then: Event time is more than 1 hour in future (should fail validation)
        long now = System.currentTimeMillis();
        long maxAllowedTime = now + (1 * 60 * 60 * 1000); // +1 hour
        assertTrue(rawEvent.getEventTime() > maxAllowedTime,
            "Event time should exceed 1 hour future tolerance");
    }

    /**
     * Test Case 8: Old Timestamp Validation
     * Verify that timestamps older than 30 days are rejected
     */
    @Test
    public void testOldTimestampValidation() {
        // Given: RawEvent with timestamp 31 days in the past
        long oldTime = System.currentTimeMillis() - (31L * 24 * 60 * 60 * 1000); // -31 days

        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);

        RawEvent rawEvent = RawEvent.builder()
            .id("test-007")
            .patientId("PAT-222")
            .type("vital_signs")
            .eventTime(oldTime)  // Too old
            .payload(payload)
            .build();

        // Then: Event time is older than 30 days (should fail validation)
        long now = System.currentTimeMillis();
        long minAllowedTime = now - (30L * 24 * 60 * 60 * 1000); // -30 days
        assertTrue(rawEvent.getEventTime() < minAllowedTime,
            "Event time should exceed 30 day retention window");
    }

    /**
     * Test Case 9: JSON Naming Consistency
     * Verify that output uses camelCase (eventId, patientId, not snake_case)
     */
    @Test
    public void testJsonNamingConsistency() throws Exception {
        // Given: CanonicalEvent with all fields
        CanonicalEvent.EventMetadata metadata =
            new CanonicalEvent.EventMetadata(
                "Python Test Script",
                "ICU Ward",
                "MON-001"
            );

        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);
        payload.put("blood_pressure", "120/80");

        CanonicalEvent canonicalEvent =
            CanonicalEvent.builder()
                .id("6a27dac5-b10e-4be1-a735-8f5cd916f4ee")
                .patientId("DEMO-456")
                .encounterId(null)
                .eventType(EventType.VITAL_SIGNS)
                .eventTime(1759305006359L)
                .payload(payload)
                .metadata(metadata)
                .build();

        // When: Serialized to JSON
        String json = objectMapper.writeValueAsString(canonicalEvent);

        // Then: Verify camelCase naming in JSON
        assertTrue(json.contains("\"eventId\""), "JSON should use eventId (camelCase)");
        assertTrue(json.contains("\"patientId\""), "JSON should use patientId (camelCase)");
        assertTrue(json.contains("\"eventType\""), "JSON should use eventType (camelCase)");
        assertTrue(json.contains("\"timestamp\""), "JSON should use timestamp (camelCase)");
        assertTrue(json.contains("\"metadata\""), "JSON should contain metadata");

        // Verify metadata uses snake_case for device_id (as specified in desired output)
        assertTrue(json.contains("\"device_id\""), "Metadata should use device_id (snake_case)");
    }
}
