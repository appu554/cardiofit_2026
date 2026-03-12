package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.ClinicalEventBuilder;
import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.models.RawEvent;
import com.cardiofit.flink.utils.TestSink;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.streaming.api.operators.ProcessOperator;
import org.apache.flink.streaming.api.watermark.Watermark;
import org.apache.flink.streaming.util.OneInputStreamOperatorTestHarness;
import org.junit.jupiter.api.*;

import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

import static org.assertj.core.api.Assertions.*;

/**
 * Test harness for Module 1: Ingestion & Gateway.
 *
 * Tests the complete ingestion pipeline:
 * - Safe deserialization and validation
 * - Schema validation and DLQ routing
 * - Watermark generation for event time processing
 * - Canonicalization and metadata preservation
 *
 * Based on Flink test harness patterns from enterprise testing documentation.
 */
@TestInstance(TestInstance.Lifecycle.PER_CLASS)
@DisplayName("Module 1: Ingestion & Gateway Test Harness")
public class Module1IngestionRouterTest {

    private OneInputStreamOperatorTestHarness<RawEvent, CanonicalEvent> harness;

    @BeforeEach
    void setUp() throws Exception {
        // Create the processor under test
        Module1_Ingestion.EventValidationAndCanonicalization processor =
            new Module1_Ingestion.EventValidationAndCanonicalization();

        // Create the operator wrapper
        ProcessOperator<RawEvent, CanonicalEvent> operator =
            new ProcessOperator<>(processor);

        // Create test harness with explicit type parameters
        harness = new OneInputStreamOperatorTestHarness<>(operator);

        // Setup and open the harness
        harness.setup();
        harness.open();

        // Clear any previous test outputs
        TestSink.clear();
    }

    @AfterEach
    void tearDown() throws Exception {
        if (harness != null) {
            harness.close();
        }
    }

    // ====================================================================
    // Test Case 1: Valid Event Flow
    // ====================================================================

    @Test
    @DisplayName("Valid event flows to main output with correct timestamp")
    void testValidEventFlowsToMainOutput() throws Exception {
        // Arrange - create valid raw event
        long timestamp = 1640995200000L;
        RawEvent rawEvent = ClinicalEventBuilder.vitalsRaw("patient-001", timestamp, 78, 120);

        // Act - process the event
        harness.processElement(rawEvent, timestamp);

        // Assert - verify main output
        assertThat(harness.getOutput()).hasSize(1);

        CanonicalEvent output = (CanonicalEvent) harness.extractOutputValues().get(0);
        assertThat(output).isNotNull();
        assertThat(output.getPatientId()).isEqualTo("patient-001");
        assertThat(output.getEventType()).isEqualTo(EventType.VITAL_SIGNS);
        assertThat(output.getEventTime()).isEqualTo(timestamp);
        assertThat(output.getPayload()).containsKeys("heart_rate", "systolic_bp");

        // Verify metadata preservation
        assertThat(output.getMetadata()).isNotNull();
        assertThat(output.getMetadata().getSource()).isEqualTo("Test Harness");
        assertThat(output.getMetadata().getLocation()).isEqualTo("Test Ward");
        assertThat(output.getMetadata().getDeviceId()).isEqualTo("TEST-DEVICE-001");

        // Verify DLQ is empty
        assertThat(harness.getSideOutput(new org.apache.flink.util.OutputTag<RawEvent>("dlq-events"){}))
            .isEmpty();
    }

    // ====================================================================
    // Test Case 2: Malformed JSON Handling
    // ====================================================================

    @Test
    @DisplayName("Malformed JSON routes to DLQ with error metadata")
    void testMalformedJsonRoutesToDLQ() throws Exception {
        // Arrange - create raw event with invalid data (null payload triggers error)
        Map<String, Object> payload = null;  // This will trigger validation failure

        RawEvent malformedEvent = RawEvent.builder()
            .id("test-malformed-001")
            .patientId("patient-999")
            .type("vital_signs")
            .eventTime(System.currentTimeMillis())
            .payload(payload)  // null payload
            .build();

        // Act - process the malformed event
        harness.processElement(malformedEvent, System.currentTimeMillis());

        // Assert - main output should be empty
        assertThat(harness.getOutput()).isEmpty();

        // Assert - DLQ should have the failed event
        List<RawEvent> dlqRecords = harness.getSideOutput(
            new org.apache.flink.util.OutputTag<RawEvent>("dlq-events"){}
        ).stream()
            .map(record -> record.getValue())
            .collect(Collectors.toList());

        assertThat(dlqRecords).hasSize(1);
        assertThat(dlqRecords.get(0).getId()).isEqualTo("test-malformed-001");
        assertThat(dlqRecords.get(0).getPatientId()).isEqualTo("patient-999");
    }

    // ====================================================================
    // Test Case 3: Schema Validation - Missing Required Fields
    // ====================================================================

    @Test
    @DisplayName("Schema validation rejects events with missing required fields")
    void testSchemaValidationRejectsMissingFields() throws Exception {
        // Test Case 3a: Missing patient ID
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);

        RawEvent missingPatientId = RawEvent.builder()
            .id("test-003a")
            .patientId(null)  // Missing patient ID
            .type("vital_signs")
            .eventTime(System.currentTimeMillis())
            .payload(payload)
            .build();

        harness.processElement(missingPatientId, System.currentTimeMillis());

        // Should route to DLQ
        assertThat(harness.getOutput()).isEmpty();
        assertThat(harness.getSideOutput(new org.apache.flink.util.OutputTag<RawEvent>("dlq-events"){}))
            .hasSize(1);

        // Test Case 3b: Blank patient ID
        RawEvent blankPatientId = RawEvent.builder()
            .id("test-003b")
            .patientId("   ")  // Blank patient ID
            .type("vital_signs")
            .eventTime(System.currentTimeMillis())
            .payload(payload)
            .build();

        harness.processElement(blankPatientId, System.currentTimeMillis());

        // Should route to DLQ
        assertThat(harness.getSideOutput(new org.apache.flink.util.OutputTag<RawEvent>("dlq-events"){}))
            .hasSize(2);  // Now 2 events in DLQ

        // Test Case 3c: Invalid timestamp (zero)
        RawEvent zeroTimestamp = RawEvent.builder()
            .id("test-003c")
            .patientId("patient-003")
            .type("vital_signs")
            .eventTime(0)  // Invalid timestamp
            .payload(payload)
            .build();

        harness.processElement(zeroTimestamp, System.currentTimeMillis());

        // Should route to DLQ
        assertThat(harness.getSideOutput(new org.apache.flink.util.OutputTag<RawEvent>("dlq-events"){}))
            .hasSize(3);  // Now 3 events in DLQ

        // Test Case 3d: Timestamp too far in future (> 1 hour)
        long futureTime = System.currentTimeMillis() + (2 * 60 * 60 * 1000); // +2 hours

        RawEvent futureTimestamp = RawEvent.builder()
            .id("test-003d")
            .patientId("patient-004")
            .type("vital_signs")
            .eventTime(futureTime)  // Too far in future
            .payload(payload)
            .build();

        harness.processElement(futureTimestamp, futureTime);

        // Should route to DLQ
        assertThat(harness.getSideOutput(new org.apache.flink.util.OutputTag<RawEvent>("dlq-events"){}))
            .hasSize(4);  // Now 4 events in DLQ

        // Test Case 3e: Timestamp too old (> 30 days)
        long oldTime = System.currentTimeMillis() - (31L * 24 * 60 * 60 * 1000); // -31 days

        RawEvent oldTimestamp = RawEvent.builder()
            .id("test-003e")
            .patientId("patient-005")
            .type("vital_signs")
            .eventTime(oldTime)  // Too old
            .payload(payload)
            .build();

        harness.processElement(oldTimestamp, oldTime);

        // Should route to DLQ
        assertThat(harness.getSideOutput(new org.apache.flink.util.OutputTag<RawEvent>("dlq-events"){}))
            .hasSize(5);  // Now 5 events in DLQ
    }

    // ====================================================================
    // Test Case 4: Watermark Generation and Out-of-Order Events
    // ====================================================================

    @Test
    @DisplayName("Watermark generation handles out-of-order events")
    void testWatermarkHandlesOutOfOrderEvents() throws Exception {
        // Arrange - create events with out-of-order timestamps
        long baseTime = System.currentTimeMillis();

        RawEvent event1 = ClinicalEventBuilder.vitalsRaw("patient-006", baseTime + 3000, 80, 120); // T+3s
        RawEvent event2 = ClinicalEventBuilder.vitalsRaw("patient-006", baseTime + 1000, 75, 115); // T+1s (late)
        RawEvent event3 = ClinicalEventBuilder.vitalsRaw("patient-006", baseTime + 2000, 78, 118); // T+2s (late)

        // Act - process events out of order
        harness.processElement(event1, baseTime + 3000);  // First: T+3s
        harness.processElement(event2, baseTime + 1000);  // Second: T+1s (out of order)
        harness.processElement(event3, baseTime + 2000);  // Third: T+2s (out of order)

        // Process watermark
        harness.processWatermark(new Watermark(baseTime + 3000));

        // Assert - all events should be processed despite being out of order
        assertThat(harness.getOutput()).hasSizeGreaterThanOrEqualTo(3);

        // Extract canonical events (filtering out watermarks)
        List<CanonicalEvent> canonicalEvents = harness.extractOutputValues().stream()
            .filter(obj -> obj instanceof CanonicalEvent)
            .map(obj -> (CanonicalEvent) obj)
            .collect(Collectors.toList());

        assertThat(canonicalEvents).hasSize(3);

        // Verify all events were processed for the same patient
        canonicalEvents.forEach(event -> {
            assertThat(event.getPatientId()).isEqualTo("patient-006");
            assertThat(event.getEventType()).isEqualTo(EventType.VITAL_SIGNS);
        });

        // Verify watermark advanced
        assertThat(harness.getOutput().stream()
            .filter(o -> o instanceof Watermark)
            .map(o -> ((Watermark) o).getTimestamp())
            .findFirst())
            .isPresent()
            .hasValueSatisfying(wm -> assertThat(wm).isGreaterThanOrEqualTo(baseTime + 1000));
    }

    // ====================================================================
    // Additional Test: Missing Event Type Handling
    // ====================================================================

    @Test
    @DisplayName("Missing event type defaults to UNKNOWN without validation failure")
    void testMissingEventTypeHandling() throws Exception {
        // Arrange - create event without type
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);

        RawEvent missingType = RawEvent.builder()
            .id("test-004")
            .patientId("patient-007")
            .type(null)  // Missing type
            .eventTime(System.currentTimeMillis())
            .payload(payload)
            .metadata(new HashMap<>())
            .build();

        // Act
        harness.processElement(missingType, System.currentTimeMillis());

        // Assert - should process successfully with UNKNOWN type
        assertThat(harness.getOutput()).hasSize(1);

        CanonicalEvent output = (CanonicalEvent) harness.extractOutputValues().get(0);
        assertThat(output.getEventType()).isEqualTo(EventType.UNKNOWN);
        assertThat(output.getPatientId()).isEqualTo("patient-007");
    }

    // ====================================================================
    // Additional Test: Payload Normalization
    // ====================================================================

    @Test
    @DisplayName("Payload normalization handles numeric strings and case variations")
    void testPayloadNormalization() throws Exception {
        // Arrange - create event with mixed payload types
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart-rate", "78");  // String number with hyphen
        payload.put("Blood_Pressure", "120");  // Mixed case with underscore
        payload.put("Temperature", 98.6);  // Already numeric

        RawEvent mixedPayload = RawEvent.builder()
            .id("test-005")
            .patientId("patient-008")
            .type("vital_signs")
            .eventTime(System.currentTimeMillis())
            .payload(payload)
            .metadata(new HashMap<>())
            .build();

        // Act
        harness.processElement(mixedPayload, System.currentTimeMillis());

        // Assert - payload should be normalized
        assertThat(harness.getOutput()).hasSize(1);

        CanonicalEvent output = (CanonicalEvent) harness.extractOutputValues().get(0);

        // Verify key normalization (lowercase, underscores)
        assertThat(output.getPayload()).containsKeys("heart_rate", "blood_pressure", "temperature");

        // Verify numeric string parsing
        assertThat(output.getPayload().get("heart_rate")).isInstanceOf(Long.class);
        assertThat(output.getPayload().get("blood_pressure")).isInstanceOf(Long.class);
        assertThat(output.getPayload().get("temperature")).isInstanceOf(Double.class);
    }
}
