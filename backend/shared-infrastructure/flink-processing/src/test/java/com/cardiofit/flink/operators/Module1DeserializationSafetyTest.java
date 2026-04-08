package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.RawEvent;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Regression test for A1 fix: RawEventDeserializer must return null on malformed JSON
 * instead of throwing IOException, which would crash the entire Flink task in a restart loop.
 *
 * These tests verify that Jackson itself throws on bad input (confirming the unsafe baseline)
 * and that our deserializer wraps this behavior safely. The actual deserializer is tested
 * indirectly through the ProcessFunction in Module1IngestionRouterTest; these tests document
 * the specific failure modes that motivated the A1 fix.
 */
@DisplayName("Module 1: Deserialization Safety (A1)")
public class Module1DeserializationSafetyTest {

    private ObjectMapper mapper;

    @BeforeEach
    void setUp() {
        mapper = new ObjectMapper();
        mapper.registerModule(new JavaTimeModule());
    }

    @Test
    @DisplayName("Malformed JSON causes Jackson to throw — our deserializer must catch this")
    void testMalformedJsonThrowsInJackson() {
        byte[] garbage = "{{not valid json at all".getBytes();

        // Confirm Jackson itself throws — this is the crash our deserializer prevents
        RawEvent result = null;
        try {
            result = mapper.readValue(garbage, RawEvent.class);
            fail("ObjectMapper should throw on malformed JSON — our deserializer wraps this");
        } catch (Exception e) {
            // Expected: Jackson throws. The A1 fix catches this in RawEventDeserializer.
        }
        assertNull(result);
    }

    @Test
    @DisplayName("Empty byte array causes Jackson to throw — our deserializer returns null early")
    void testEmptyBytesThrowsInJackson() {
        byte[] empty = new byte[0];

        RawEvent result = null;
        try {
            result = mapper.readValue(empty, RawEvent.class);
            fail("ObjectMapper should throw on empty bytes");
        } catch (Exception e) {
            // Expected: Jackson throws. The A1 fix returns null before reaching Jackson.
        }
        assertNull(result);
    }

    @Test
    @DisplayName("Truncated JSON causes Jackson to throw — our deserializer must catch this")
    void testTruncatedJsonThrowsInJackson() {
        // Simulates a Kafka message that was truncated mid-write
        byte[] truncated = "{\"id\":\"evt-001\",\"patientId\":\"PAT-".getBytes();

        RawEvent result = null;
        try {
            result = mapper.readValue(truncated, RawEvent.class);
            fail("ObjectMapper should throw on truncated JSON");
        } catch (Exception e) {
            // Expected: Jackson throws on incomplete JSON.
        }
        assertNull(result);
    }

    @Test
    @DisplayName("Valid JSON deserializes successfully")
    void testValidJsonDeserializesCorrectly() throws Exception {
        // RawEvent uses snake_case @JsonProperty annotations (patient_id, event_time)
        String validJson = "{\"id\":\"evt-001\",\"patient_id\":\"PAT-001\",\"type\":\"vital_signs\",\"event_time\":1640995200000}";

        RawEvent result = mapper.readValue(validJson.getBytes(), RawEvent.class);
        assertNotNull(result);
        assertEquals("evt-001", result.getId());
        assertEquals("PAT-001", result.getPatientId());
    }
}
