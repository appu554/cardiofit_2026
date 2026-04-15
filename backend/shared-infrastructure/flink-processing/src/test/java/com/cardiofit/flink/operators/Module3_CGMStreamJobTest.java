package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CGMAnalyticsEvent;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.Test;

import java.util.Collections;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for the pure helpers inside {@link Module3_CGMStreamJob}.
 *
 * <p>Phase 7 P7-E: a full Flink mini-cluster integration test is
 * deferred to Phase 8 per the P7-E plan (requires flink-test-utils
 * dependency addition + MiniClusterWithClientResource setup). These
 * unit tests cover the JSON parsing, timestamp extraction, and
 * analytics-event serialization paths that comprise the bulk of the
 * pipeline's risk surface — the Flink windowing primitives themselves
 * are covered by Flink's own test suite.
 */
class Module3_CGMStreamJobTest {

    private static final ObjectMapper MAPPER = new ObjectMapper();

    // ─────────────── parseReading tests ───────────────

    @Test
    void parseReading_acceptsCanonicalSnakeCaseEnvelope() {
        String json = "{\"patient_id\":\"p1\",\"timestamp_ms\":1700000000000,\"glucose_mg_dl\":140.5}";
        Module3_CGMStreamJob.CGMReading r = Module3_CGMStreamJob.parseReading(json);
        assertNotNull(r);
        assertEquals("p1", r.patientId);
        assertEquals(1700000000000L, r.timestampMs);
        assertEquals(140.5, r.glucoseMgDl, 0.001);
    }

    @Test
    void parseReading_acceptsCamelCasePatientIdAsFallback() {
        // Some upstream CGM producers use camelCase; the parser must
        // accept both to avoid silently dropping valid records.
        String json = "{\"patientId\":\"p2\",\"timestamp_ms\":1700000000000,\"glucose_mg_dl\":95}";
        Module3_CGMStreamJob.CGMReading r = Module3_CGMStreamJob.parseReading(json);
        assertNotNull(r);
        assertEquals("p2", r.patientId);
        assertEquals(95.0, r.glucoseMgDl, 0.001);
    }

    @Test
    void parseReading_acceptsIso8601TimestampFallback() {
        String json = "{\"patient_id\":\"p3\",\"timestamp\":\"2026-04-15T10:30:00Z\",\"glucose_mg_dl\":110}";
        Module3_CGMStreamJob.CGMReading r = Module3_CGMStreamJob.parseReading(json);
        assertNotNull(r);
        assertEquals("p3", r.patientId);
        assertTrue(r.timestampMs > 0);
        // 2026-04-15T10:30:00Z expressed in epoch millis
        assertEquals(1776249000000L, r.timestampMs);
    }

    @Test
    void parseReading_acceptsShortGlucoseFieldName() {
        String json = "{\"patient_id\":\"p4\",\"timestamp_ms\":1700000000000,\"glucose\":180}";
        Module3_CGMStreamJob.CGMReading r = Module3_CGMStreamJob.parseReading(json);
        assertNotNull(r);
        assertEquals(180.0, r.glucoseMgDl, 0.001);
    }

    @Test
    void parseReading_fallsBackToNowOnMissingTimestamp() {
        String json = "{\"patient_id\":\"p5\",\"glucose_mg_dl\":120}";
        long before = System.currentTimeMillis();
        Module3_CGMStreamJob.CGMReading r = Module3_CGMStreamJob.parseReading(json);
        long after = System.currentTimeMillis();
        assertNotNull(r);
        assertTrue(r.timestampMs >= before && r.timestampMs <= after,
                "expected timestampMs within [before, after]");
    }

    @Test
    void parseReading_returnsNullWhenPatientIdMissing() {
        String json = "{\"timestamp_ms\":1700000000000,\"glucose_mg_dl\":120}";
        assertNull(Module3_CGMStreamJob.parseReading(json));
    }

    @Test
    void parseReading_returnsNullWhenGlucoseMissing() {
        String json = "{\"patient_id\":\"p6\",\"timestamp_ms\":1700000000000}";
        assertNull(Module3_CGMStreamJob.parseReading(json));
    }

    @Test
    void parseReading_returnsNullOnMalformedJson() {
        assertNull(Module3_CGMStreamJob.parseReading("{not json"));
        assertNull(Module3_CGMStreamJob.parseReading(""));
        assertNull(Module3_CGMStreamJob.parseReading(null));
    }

    // ─────────────── extractTimestamp tests ───────────────

    @Test
    void extractTimestamp_readsTimestampMsField() {
        String json = "{\"patient_id\":\"p1\",\"timestamp_ms\":1700000000000,\"glucose_mg_dl\":120}";
        assertEquals(1700000000000L, Module3_CGMStreamJob.extractTimestamp(json));
    }

    @Test
    void extractTimestamp_fallsBackToIso8601() {
        String json = "{\"patient_id\":\"p1\",\"timestamp\":\"2026-04-15T10:30:00Z\",\"glucose_mg_dl\":120}";
        assertEquals(1776249000000L, Module3_CGMStreamJob.extractTimestamp(json));
    }

    @Test
    void extractTimestamp_fallsBackToNowOnBadJson() {
        long before = System.currentTimeMillis();
        long ts = Module3_CGMStreamJob.extractTimestamp("{not json");
        long after = System.currentTimeMillis();
        assertTrue(ts >= before && ts <= after);
    }

    @Test
    void extractTimestamp_fallsBackToNowOnEmptyOrNull() {
        long before = System.currentTimeMillis();
        long ts1 = Module3_CGMStreamJob.extractTimestamp("");
        long ts2 = Module3_CGMStreamJob.extractTimestamp(null);
        long after = System.currentTimeMillis();
        assertTrue(ts1 >= before && ts1 <= after);
        assertTrue(ts2 >= before && ts2 <= after);
    }

    // ─────────────── serializeAnalyticsEvent tests ───────────────

    @Test
    void serializeAnalyticsEvent_containsCoreMetricsForAllInRangePatient() throws Exception {
        // Seed the compute with all-in-range readings (1344 @ 140 mg/dL
        // — 14 days × 96 readings/day at 15-min intervals).
        List<Double> readings = Collections.nCopies(1344, 140.0);
        CGMAnalyticsEvent event = Module3_CGMAnalytics.computeMetrics(
                readings, Module3_CGMStreamJob.WINDOW_DAYS);

        String json = Module3_CGMStreamJob.serializeAnalyticsEvent(event, "p1", 1776249000000L);
        JsonNode node = MAPPER.readTree(json);

        assertEquals("CGMAnalyticsEvent", node.path("event_type").asText());
        assertEquals("v1", node.path("event_version").asText());
        assertEquals("p1", node.path("patient_id").asText());
        assertEquals(1776249000000L, node.path("window_end_ms").asLong());
        assertEquals(14, node.path("window_days").asInt());
        assertEquals(1344, node.path("total_readings").asInt());
        assertEquals(100.0, node.path("tir_pct").asDouble(), 0.01);
        assertEquals(140.0, node.path("mean_glucose").asDouble(), 0.01);
    }

    @Test
    void serializeAnalyticsEvent_survivesNullFieldsGracefully() {
        // An event with mostly default fields should still serialize
        // to valid JSON so a pathological compute output doesn't take
        // the sink down.
        CGMAnalyticsEvent event = CGMAnalyticsEvent.builder()
                .patientId("ignored-in-serialize")
                .totalReadings(0)
                .build();
        String json = Module3_CGMStreamJob.serializeAnalyticsEvent(event, "p-empty", 0L);
        assertNotNull(json);
        assertTrue(json.contains("\"patient_id\":\"p-empty\""));
    }

    // ─────────────── extractPatientIdBytes tests ───────────────

    @Test
    void extractPatientIdBytes_readsPatientIdField() {
        String json = "{\"event_type\":\"CGMAnalyticsEvent\",\"patient_id\":\"p-abc\"}";
        byte[] bytes = Module3_CGMStreamJob.extractPatientIdBytes(json);
        assertEquals("p-abc", new String(bytes));
    }

    @Test
    void extractPatientIdBytes_returnsEmptyOnBadJson() {
        byte[] bytes = Module3_CGMStreamJob.extractPatientIdBytes("{not json");
        assertEquals(0, bytes.length);
    }

    // ─────────────── Pipeline wiring smoke test ───────────────

    @Test
    void createCGMAnalyticsPipeline_wiresWithoutThrowing() {
        // Smoke test: the pipeline builder constructs source + window +
        // sink without throwing when handed a live local execution
        // environment. Guards against wiring typos (missing operator,
        // wrong type parameter) that would only surface at deploy.
        //
        // We build the pipeline but do NOT call env.execute() — that
        // would attempt to connect to Kafka which is not available in
        // a unit test context.
        org.apache.flink.streaming.api.environment.StreamExecutionEnvironment env =
                org.apache.flink.streaming.api.environment.StreamExecutionEnvironment
                        .createLocalEnvironment(1);
        assertDoesNotThrow(() -> Module3_CGMStreamJob.createCGMAnalyticsPipeline(env));
    }
}
