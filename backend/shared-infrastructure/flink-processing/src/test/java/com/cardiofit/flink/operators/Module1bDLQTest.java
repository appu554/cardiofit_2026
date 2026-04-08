package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.OutboxEnvelope;
import org.apache.flink.streaming.api.operators.ProcessOperator;
import org.apache.flink.streaming.util.OneInputStreamOperatorTestHarness;
import org.apache.flink.util.OutputTag;
import org.junit.jupiter.api.*;

import static org.assertj.core.api.Assertions.*;

/**
 * Regression tests for A5, A6, Q7 fixes:
 * - A5: Null eventData routes to DLQ (not silently passed through)
 * - A6: ProcessFunction enables DLQ side output (was MapFunction before)
 * - Q7: Null patientId routes to DLQ (not defaulted to "UNKNOWN")
 *
 * These tests verify that Module 1b's ProcessFunction correctly routes
 * invalid OutboxEnvelope events to the DLQ side output rather than
 * letting corrupt data enter the clinical pipeline.
 */
@DisplayName("Module 1b: DLQ Routing Tests (A5, A6, Q7)")
public class Module1bDLQTest {

    private static final OutputTag<OutboxEnvelope> DLQ_TAG =
        new OutputTag<OutboxEnvelope>("dlq-ingestion-events"){};

    private OneInputStreamOperatorTestHarness<OutboxEnvelope, CanonicalEvent> harness;

    @BeforeEach
    void setUp() throws Exception {
        Module1b_IngestionCanonicalizer.OutboxValidationAndCanonicalization processor =
            new Module1b_IngestionCanonicalizer.OutboxValidationAndCanonicalization();
        ProcessOperator<OutboxEnvelope, CanonicalEvent> operator = new ProcessOperator<>(processor);
        harness = new OneInputStreamOperatorTestHarness<>(operator);
        harness.setup();
        harness.open();
    }

    @AfterEach
    void tearDown() throws Exception {
        if (harness != null) harness.close();
    }

    @Test
    @DisplayName("A5: Null eventData routes to DLQ, not main output")
    void testNullEventDataRoutesToDLQ() throws Exception {
        OutboxEnvelope envelope = new OutboxEnvelope();
        envelope.setId("test-null-data");
        envelope.setCorrelationId("corr-001");
        envelope.setEventData(null);

        harness.processElement(envelope, System.currentTimeMillis());

        assertThat(harness.extractOutputValues()).isEmpty();
        assertThat(harness.getSideOutput(DLQ_TAG)).hasSize(1);
    }

    @Test
    @DisplayName("Q7: Null patientId routes to DLQ, not main output")
    void testNullPatientIdRoutesToDLQ() throws Exception {
        OutboxEnvelope envelope = new OutboxEnvelope();
        envelope.setId("test-null-patient");
        OutboxEnvelope.IngestionEventData data = new OutboxEnvelope.IngestionEventData();
        data.setPatientId(null);
        data.setObservationType("VITALS");
        data.setTimestamp("2026-03-27T10:00:00Z");
        envelope.setEventData(data);

        harness.processElement(envelope, System.currentTimeMillis());

        assertThat(harness.extractOutputValues()).isEmpty();
        assertThat(harness.getSideOutput(DLQ_TAG)).hasSize(1);
    }

    @Test
    @DisplayName("Q7: Blank patientId routes to DLQ, not main output")
    void testBlankPatientIdRoutesToDLQ() throws Exception {
        OutboxEnvelope envelope = new OutboxEnvelope();
        envelope.setId("test-blank-patient");
        OutboxEnvelope.IngestionEventData data = new OutboxEnvelope.IngestionEventData();
        data.setPatientId("   ");
        data.setObservationType("VITALS");
        data.setTimestamp("2026-03-27T10:00:00Z");
        envelope.setEventData(data);

        harness.processElement(envelope, System.currentTimeMillis());

        assertThat(harness.extractOutputValues()).isEmpty();
        assertThat(harness.getSideOutput(DLQ_TAG)).hasSize(1);
    }

    @Test
    @DisplayName("Null timestamp routes to DLQ")
    void testNullTimestampRoutesToDLQ() throws Exception {
        OutboxEnvelope envelope = new OutboxEnvelope();
        envelope.setId("test-null-ts");
        OutboxEnvelope.IngestionEventData data = new OutboxEnvelope.IngestionEventData();
        data.setPatientId("PAT-TS");
        data.setObservationType("VITALS");
        data.setTimestamp(null);
        envelope.setEventData(data);

        harness.processElement(envelope, System.currentTimeMillis());

        assertThat(harness.extractOutputValues()).isEmpty();
        assertThat(harness.getSideOutput(DLQ_TAG)).hasSize(1);
    }

    @Test
    @DisplayName("Unparseable timestamp routes to DLQ")
    void testUnparseableTimestampRoutesToDLQ() throws Exception {
        OutboxEnvelope envelope = new OutboxEnvelope();
        envelope.setId("test-bad-ts");
        OutboxEnvelope.IngestionEventData data = new OutboxEnvelope.IngestionEventData();
        data.setPatientId("PAT-BADTS");
        data.setObservationType("LABS");
        data.setTimestamp("not-a-timestamp");
        envelope.setEventData(data);

        harness.processElement(envelope, System.currentTimeMillis());

        assertThat(harness.extractOutputValues()).isEmpty();
        assertThat(harness.getSideOutput(DLQ_TAG)).hasSize(1);
    }

    @Test
    @DisplayName("Valid envelope passes to main output with correct fields")
    void testValidEnvelopePassesToMainOutput() throws Exception {
        OutboxEnvelope envelope = new OutboxEnvelope();
        envelope.setId("test-valid");
        envelope.setCorrelationId("corr-002");
        OutboxEnvelope.IngestionEventData data = new OutboxEnvelope.IngestionEventData();
        data.setEventId("evt-001");
        data.setPatientId("PAT-VALID");
        data.setObservationType("VITALS");
        data.setValue(120.0);
        data.setUnit("mmHg");
        data.setTimestamp("2026-03-27T10:00:00Z");
        envelope.setEventData(data);

        harness.processElement(envelope, System.currentTimeMillis());

        assertThat(harness.extractOutputValues()).hasSize(1);
        CanonicalEvent output = (CanonicalEvent) harness.extractOutputValues().get(0);
        assertThat(output.getPatientId()).isEqualTo("PAT-VALID");
        assertThat(output.getSourceSystem()).isEqualTo("ingestion-service");
        assertThat(output.getCorrelationId()).isEqualTo("corr-002");
        assertThat(harness.getSideOutput(DLQ_TAG)).isNullOrEmpty();
    }
}
