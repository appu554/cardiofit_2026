package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.models.RawEvent;
import org.apache.flink.streaming.api.operators.ProcessOperator;
import org.apache.flink.streaming.util.OneInputStreamOperatorTestHarness;
import org.apache.flink.util.OutputTag;
import org.junit.jupiter.api.*;

import java.util.HashMap;
import java.util.Map;

import static org.assertj.core.api.Assertions.*;

/**
 * Regression tests for C1, Q1, Q2 fixes:
 * - Q2: Missing event type should pass validation with UNKNOWN type (not route to DLQ)
 * - C1: Future/old timestamps should be clamped to boundaries (not rejected to DLQ)
 *
 * These tests exercise the behavioral contract change from "reject" to "clamp and sanitize"
 * in Module1_Ingestion.validateEvent().
 */
@DisplayName("Module 1: Validation Fix Tests (C1, Q1, Q2)")
public class Module1ValidationFixTest {

    private OneInputStreamOperatorTestHarness<RawEvent, CanonicalEvent> harness;

    @BeforeEach
    void setUp() throws Exception {
        Module1_Ingestion.EventValidationAndCanonicalization processor =
            new Module1_Ingestion.EventValidationAndCanonicalization();
        ProcessOperator<RawEvent, CanonicalEvent> operator = new ProcessOperator<>(processor);
        harness = new OneInputStreamOperatorTestHarness<>(operator);
        harness.setup();
        harness.open();
    }

    @AfterEach
    void tearDown() throws Exception {
        if (harness != null) harness.close();
    }

    @Test
    @DisplayName("Q2: Missing event type should pass validation with UNKNOWN type")
    void testMissingEventTypePassesValidation() throws Exception {
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);

        RawEvent event = RawEvent.builder()
            .id("test-q2")
            .patientId("PAT-Q2")
            .type(null)
            .eventTime(System.currentTimeMillis())
            .payload(payload)
            .metadata(new HashMap<>())
            .build();

        harness.processElement(event, System.currentTimeMillis());

        // Should pass to main output, NOT DLQ
        assertThat(harness.extractOutputValues()).hasSize(1);
        CanonicalEvent output = (CanonicalEvent) harness.extractOutputValues().get(0);
        assertThat(output.getEventType()).isEqualTo(EventType.UNKNOWN);
        assertThat(output.getPatientId()).isEqualTo("PAT-Q2");

        // DLQ should be empty
        assertThat(harness.getSideOutput(new OutputTag<RawEvent>("dlq-events"){}))
            .isNullOrEmpty();
    }

    @Test
    @DisplayName("Q2: Empty string event type should also pass validation")
    void testEmptyEventTypePassesValidation() throws Exception {
        Map<String, Object> payload = new HashMap<>();
        payload.put("glucose", 5.5);

        RawEvent event = RawEvent.builder()
            .id("test-q2b")
            .patientId("PAT-Q2B")
            .type("")
            .eventTime(System.currentTimeMillis())
            .payload(payload)
            .metadata(new HashMap<>())
            .build();

        harness.processElement(event, System.currentTimeMillis());

        assertThat(harness.extractOutputValues()).hasSize(1);
        CanonicalEvent output = (CanonicalEvent) harness.extractOutputValues().get(0);
        assertThat(output.getEventType()).isEqualTo(EventType.UNKNOWN);
    }

    @Test
    @DisplayName("C1: Future timestamp should be clamped to now+1h, not rejected")
    void testFutureTimestampIsClamped() throws Exception {
        long futureTime = System.currentTimeMillis() + (2 * 60 * 60 * 1000); // +2h
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);

        RawEvent event = RawEvent.builder()
            .id("test-c1-future")
            .patientId("PAT-C1")
            .type("vital_signs")
            .eventTime(futureTime)
            .payload(payload)
            .metadata(new HashMap<>())
            .build();

        harness.processElement(event, futureTime);

        // Should pass to main output (clamped), NOT DLQ
        assertThat(harness.extractOutputValues()).hasSize(1);
        CanonicalEvent output = (CanonicalEvent) harness.extractOutputValues().get(0);
        // Clamped timestamp should be <= now + 1h (with 5s tolerance for test execution)
        long maxAllowed = System.currentTimeMillis() + (60 * 60 * 1000) + 5000;
        assertThat(output.getEventTime()).isLessThanOrEqualTo(maxAllowed);

        // DLQ should be empty
        assertThat(harness.getSideOutput(new OutputTag<RawEvent>("dlq-events"){}))
            .isNullOrEmpty();
    }

    @Test
    @DisplayName("C1: Old timestamp (>30d) should be clamped to 30d boundary, not rejected")
    void testOldTimestampIsClamped() throws Exception {
        long oldTime = System.currentTimeMillis() - (45L * 24 * 60 * 60 * 1000); // -45d
        Map<String, Object> payload = new HashMap<>();
        payload.put("hba1c", 7.2);

        RawEvent event = RawEvent.builder()
            .id("test-c1-old")
            .patientId("PAT-C1-OLD")
            .type("lab_result")
            .eventTime(oldTime)
            .payload(payload)
            .metadata(new HashMap<>())
            .build();

        harness.processElement(event, oldTime);

        // Should pass to main output (clamped), NOT DLQ
        assertThat(harness.extractOutputValues()).hasSize(1);
        CanonicalEvent output = (CanonicalEvent) harness.extractOutputValues().get(0);
        long minAllowed = System.currentTimeMillis() - (30L * 24 * 60 * 60 * 1000) - 5000;
        assertThat(output.getEventTime()).isGreaterThanOrEqualTo(minAllowed);

        // DLQ should be empty
        assertThat(harness.getSideOutput(new OutputTag<RawEvent>("dlq-events"){}))
            .isNullOrEmpty();
    }

    @Test
    @DisplayName("Null patient ID still routes to DLQ (not affected by clamping changes)")
    void testNullPatientIdStillRoutesDLQ() throws Exception {
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);

        RawEvent event = RawEvent.builder()
            .id("test-still-dlq")
            .patientId(null)
            .type("vital_signs")
            .eventTime(System.currentTimeMillis())
            .payload(payload)
            .metadata(new HashMap<>())
            .build();

        harness.processElement(event, System.currentTimeMillis());

        // Should route to DLQ
        assertThat(harness.extractOutputValues()).isEmpty();
        assertThat(harness.getSideOutput(new OutputTag<RawEvent>("dlq-events"){}))
            .hasSize(1);
    }
}
