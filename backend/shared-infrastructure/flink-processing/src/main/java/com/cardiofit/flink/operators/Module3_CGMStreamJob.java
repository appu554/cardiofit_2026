package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CGMAnalyticsEvent;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;

import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.functions.MapFunction;
import org.apache.flink.api.common.serialization.SimpleStringSchema;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.KeyedStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.windowing.ProcessWindowFunction;
import org.apache.flink.streaming.api.windowing.assigners.SlidingEventTimeWindows;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.Properties;

/**
 * Module 3 — CGM Analytics Streaming Job.
 *
 * <p>Phase 7 P7-E: first Flink pipeline to actually consume
 * {@code ingestion.cgm-raw}. Windows raw CGM glucose readings per patient
 * over a 14-day sliding window (1-day slide), runs the existing pure
 * {@link Module3_CGMAnalytics#computeMetrics(List, int)} compute, and
 * publishes one {@link CGMAnalyticsEvent} per patient per day to
 * {@code clinical.cgm-analytics.v1} for KB-26 persistence.
 *
 * <p>Architecture:
 * <pre>
 *   Kafka (ingestion.cgm-raw)
 *     → Parse JSON → (patientId, timestampMs, glucoseMgDl)
 *     → Watermark: bounded out-of-orderness 5 min (device batching)
 *     → keyBy(patientId)
 *     → SlidingEventTimeWindow(14 days, 1 day)
 *       → ProcessWindowFunction: collect readings → Module3_CGMAnalytics.computeMetrics
 *       → emit CGMAnalyticsEvent
 *     → Kafka (clinical.cgm-analytics.v1)
 * </pre>
 *
 * <p>Consumed topics:
 * <ul>
 *   <li>{@link KafkaTopics#INGESTION_CGM_RAW} — raw CGM readings from the ingestion layer
 * </ul>
 *
 * <p>Produced topics:
 * <ul>
 *   <li>{@link KafkaTopics#CLINICAL_CGM_ANALYTICS} — windowed CGM analytics events
 * </ul>
 *
 * <p>Env vars:
 * <ul>
 *   <li>{@code KAFKA_BOOTSTRAP_SERVERS} — Kafka bootstrap (fallback: localhost:9092)
 *   <li>{@code CGM_SOURCE_TOPIC} — override source topic (default: ingestion.cgm-raw)
 *   <li>{@code CGM_OUTPUT_TOPIC} — override sink topic (default: clinical.cgm-analytics.v1)
 *   <li>{@code CGM_CONSUMER_GROUP} — override consumer group (default: module3-cgm-analytics)
 * </ul>
 *
 * <p>Canonical shape copied from {@link Module3_ComprehensiveCDS} — same
 * getBootstrapServers + getTopicName + KafkaConfigLoader helpers. This
 * keeps the job consistent with every other Flink job in the codebase.
 */
public class Module3_CGMStreamJob {
    private static final Logger LOG = LoggerFactory.getLogger(Module3_CGMStreamJob.class);

    private static final ObjectMapper MAPPER = new ObjectMapper();

    /** CGM sliding window: 14 days (AGP reporting convention). */
    public static final int WINDOW_DAYS = 14;

    /** Window slide: 1 day (fresh report per patient per day). */
    public static final int SLIDE_DAYS = 1;

    /** Default consumer group ID. */
    private static final String DEFAULT_CONSUMER_GROUP = "module3-cgm-analytics";

    /** Default transactional ID prefix for the sink producer. */
    private static final String SINK_TX_PREFIX = "module3-cgm-analytics-tx";

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Module 3: CGM Analytics Streaming Job (Phase 7 P7-E)");

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Parallelism + checkpointing match Module3_ComprehensiveCDS so
        // the CGM job schedules consistently with the existing Module 3
        // driver on the shared cluster.
        env.setParallelism(2);
        env.enableCheckpointing(30000);
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);

        createCGMAnalyticsPipeline(env);

        env.execute("Module 3: CGM Analytics (14-day sliding window)");
    }

    /**
     * Build the CGM analytics pipeline on the given execution environment.
     * Exposed as a static builder so a parent orchestrator can compose
     * this pipeline alongside other Module 3 jobs (same pattern as
     * {@link Module3_ComprehensiveCDS#createComprehensiveCDSPipeline}).
     */
    public static void createCGMAnalyticsPipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating Module 3 CGM analytics pipeline: source={}, sink={}, window={}d/{}d",
                KafkaTopics.INGESTION_CGM_RAW.getTopicName(),
                KafkaTopics.CLINICAL_CGM_ANALYTICS.getTopicName(),
                WINDOW_DAYS, SLIDE_DAYS);

        // 1. Source: raw CGM readings from ingestion.cgm-raw
        KafkaSource<String> rawSource = KafkaSource.<String>builder()
                .setBootstrapServers(getBootstrapServers())
                .setTopics(getTopicName("CGM_SOURCE_TOPIC", KafkaTopics.INGESTION_CGM_RAW.getTopicName()))
                .setGroupId(getTopicName("CGM_CONSUMER_GROUP", DEFAULT_CONSUMER_GROUP))
                .setValueOnlyDeserializer(new SimpleStringSchema())
                .setProperties(KafkaConfigLoader.getAutoConsumerConfig(
                        getTopicName("CGM_CONSUMER_GROUP", DEFAULT_CONSUMER_GROUP)))
                .build();

        DataStream<String> rawStream = env.fromSource(
                rawSource,
                // CGM devices batch-upload readings; 5 min out-of-orderness
                // covers the vast majority of device behaviours without
                // pushing watermarks so far back that window firing stalls.
                WatermarkStrategy.<String>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                        .withTimestampAssigner((event, recordTs) -> extractTimestamp(event)),
                "cgm-raw-source");

        // 2. Parse: JSON string → typed CGMReading record
        DataStream<CGMReading> parsed = rawStream
                .map(new CGMReadingParser())
                .filter(r -> r != null && r.patientId != null && !r.patientId.isEmpty())
                .name("parse-cgm-readings");

        // 3. Key by patient_id so all of a patient's readings land in
        //    the same keyed state, giving Flink the ability to maintain
        //    one window per patient independently.
        KeyedStream<CGMReading, String> keyed = parsed.keyBy(r -> r.patientId);

        // 4. Window: 14-day sliding window advancing every 1 day.
        // Flink 2.1 uses java.time.Duration for window sizing
        // (the legacy Time class was removed).
        DataStream<String> analyticsJson = keyed
                .window(SlidingEventTimeWindows.of(
                        Duration.ofDays(WINDOW_DAYS),
                        Duration.ofDays(SLIDE_DAYS)))
                .process(new CGMWindowProcessor())
                .name("cgm-14d-sliding-window");

        // 5. Sink: serialized JSON events to clinical.cgm-analytics.v1
        Properties producerConfig = new Properties();
        producerConfig.setProperty("bootstrap.servers", getBootstrapServers());
        producerConfig.setProperty("compression.type", "snappy");
        producerConfig.setProperty("batch.size", "32768");
        producerConfig.setProperty("linger.ms", "100");
        producerConfig.setProperty("acks", "all");
        producerConfig.setProperty("enable.idempotence", "true");
        producerConfig.setProperty("retries", "2147483647");
        producerConfig.setProperty("max.in.flight.requests.per.connection", "5");
        producerConfig.setProperty("delivery.timeout.ms", "120000");

        KafkaSink<String> analyticsSink = KafkaSink.<String>builder()
                .setBootstrapServers(getBootstrapServers())
                .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                        .setTopic(getTopicName("CGM_OUTPUT_TOPIC",
                                KafkaTopics.CLINICAL_CGM_ANALYTICS.getTopicName()))
                        .setKeySerializationSchema((String json) -> extractPatientIdBytes(json))
                        .setValueSerializationSchema(new SimpleStringSchema())
                        .build())
                .setTransactionalIdPrefix(SINK_TX_PREFIX)
                .setKafkaProducerConfig(producerConfig)
                .build();

        analyticsJson.sinkTo(analyticsSink).name("cgm-analytics-sink");

        LOG.info("Module 3 CGM analytics pipeline configured successfully");
    }

    // ─────────────── Reading model ───────────────

    /**
     * Parsed CGM reading from the ingestion.cgm-raw topic. Flink keyed
     * state uses this as the windowed element type; only patient_id,
     * timestamp, and glucose value are needed for the compute function.
     */
    public static class CGMReading implements Serializable {
        private static final long serialVersionUID = 1L;
        public String patientId;
        public long timestampMs;
        public double glucoseMgDl;

        public CGMReading() {}

        public CGMReading(String patientId, long timestampMs, double glucoseMgDl) {
            this.patientId = patientId;
            this.timestampMs = timestampMs;
            this.glucoseMgDl = glucoseMgDl;
        }
    }

    // ─────────────── Operators ───────────────

    /**
     * Parse raw JSON strings into CGMReading records. Tolerant of
     * missing timestamps (falls back to current processing time) and
     * accepts multiple field name variants for the glucose value.
     */
    public static class CGMReadingParser implements MapFunction<String, CGMReading> {
        private static final long serialVersionUID = 1L;

        @Override
        public CGMReading map(String json) {
            return parseReading(json);
        }
    }

    /**
     * Per-window processor that collects glucose readings from the
     * window iterable, delegates to {@link Module3_CGMAnalytics#computeMetrics},
     * and serializes the output CGMAnalyticsEvent to JSON for the sink.
     *
     * <p>Using ProcessWindowFunction (not AggregateFunction) because
     * the downstream compute needs all readings at window close, not a
     * running aggregate. Trade-off: Flink holds the full window buffer
     * in state. Bounded by 14 days × 288 readings/day = ~4K readings
     * per patient per window — acceptable keyed-state footprint.
     */
    public static class CGMWindowProcessor
            extends ProcessWindowFunction<CGMReading, String, String, TimeWindow> {
        private static final long serialVersionUID = 1L;

        @Override
        public void process(
                String patientId,
                Context ctx,
                Iterable<CGMReading> readings,
                Collector<String> out) throws Exception {
            List<Double> glucoseValues = new ArrayList<>();
            for (CGMReading r : readings) {
                glucoseValues.add(r.glucoseMgDl);
            }
            if (glucoseValues.isEmpty()) {
                return;
            }

            CGMAnalyticsEvent event = Module3_CGMAnalytics.computeMetrics(glucoseValues, WINDOW_DAYS);

            // Stamp identity fields that computeMetrics leaves unset —
            // it's a pure function and doesn't know the patient or the
            // window boundaries.
            String json = serializeAnalyticsEvent(event, patientId, ctx.window().getEnd());
            out.collect(json);
        }
    }

    // ─────────────── Pure helpers (exported for unit tests) ───────────────

    /**
     * Parse a single CGM reading JSON envelope. Tolerant parser — unknown
     * fields are ignored, missing timestamps fall back to now, and both
     * {@code glucose_mg_dl} and {@code glucose} field names are accepted.
     * Returns {@code null} on parse failure so the upstream filter can
     * drop the record without aborting the stream.
     */
    public static CGMReading parseReading(String json) {
        if (json == null || json.isEmpty()) {
            return null;
        }
        try {
            JsonNode node = MAPPER.readTree(json);
            CGMReading r = new CGMReading();
            r.patientId = node.path("patient_id").asText("");
            if (r.patientId.isEmpty()) {
                // Second chance: some upstream producers use camelCase
                r.patientId = node.path("patientId").asText("");
            }
            if (r.patientId.isEmpty()) {
                return null;
            }

            long ts = node.path("timestamp_ms").asLong(0L);
            if (ts == 0L) {
                String tsStr = node.path("timestamp").asText("");
                if (!tsStr.isEmpty()) {
                    try {
                        ts = Instant.parse(tsStr).toEpochMilli();
                    } catch (Exception ignored) {
                        ts = 0L;
                    }
                }
            }
            if (ts == 0L) {
                ts = System.currentTimeMillis();
            }
            r.timestampMs = ts;

            if (node.has("glucose_mg_dl")) {
                r.glucoseMgDl = node.path("glucose_mg_dl").asDouble();
            } else if (node.has("glucose")) {
                r.glucoseMgDl = node.path("glucose").asDouble();
            } else {
                return null; // No glucose value — unusable record
            }
            return r;
        } catch (Exception e) {
            LOG.warn("Module3 CGM: failed to parse reading: {}", e.getMessage());
            return null;
        }
    }

    /**
     * Extract event-time from a raw JSON envelope. Used by the watermark
     * strategy; falls back to current time on any parse failure so bad
     * records don't stall the pipeline.
     */
    public static long extractTimestamp(String json) {
        if (json == null || json.isEmpty()) {
            return System.currentTimeMillis();
        }
        try {
            JsonNode node = MAPPER.readTree(json);
            long ts = node.path("timestamp_ms").asLong(0L);
            if (ts > 0L) {
                return ts;
            }
            String tsStr = node.path("timestamp").asText("");
            if (!tsStr.isEmpty()) {
                return Instant.parse(tsStr).toEpochMilli();
            }
        } catch (Exception ignored) {
            // Fall through
        }
        return System.currentTimeMillis();
    }

    /**
     * Serialize a CGMAnalyticsEvent to the JSON wire format expected by
     * downstream consumers (KB-26's cgm_analytics_consumer). Keeps the
     * projection narrow — only the fields KB-26 persists — so wire size
     * stays small. AGP percentiles are deliberately omitted because the
     * persistence table doesn't carry them; a future Phase 8 enrichment
     * can add them if clinicians need per-hour percentile overlays.
     */
    public static String serializeAnalyticsEvent(
            CGMAnalyticsEvent event,
            String patientId,
            long windowEndMs) {
        try {
            com.fasterxml.jackson.databind.node.ObjectNode node = MAPPER.createObjectNode();
            node.put("event_type", "CGMAnalyticsEvent");
            node.put("event_version", "v1");
            node.put("patient_id", patientId);
            node.put("computed_at_ms", System.currentTimeMillis());
            node.put("window_end_ms", windowEndMs);
            node.put("window_days", WINDOW_DAYS);
            node.put("total_readings", event.getTotalReadings());
            node.put("coverage_pct", event.getCoveragePct());
            node.put("sufficient_data", event.isSufficientData());
            node.put("confidence_level", event.getConfidenceLevel());
            node.put("mean_glucose", event.getMeanGlucose());
            node.put("sd_glucose", event.getSdGlucose());
            node.put("cv_pct", event.getCvPct());
            node.put("glucose_stable", event.isGlucoseStable());
            node.put("tir_pct", event.getTirPct());
            node.put("tbr_l1_pct", event.getTbrL1Pct());
            node.put("tbr_l2_pct", event.getTbrL2Pct());
            node.put("tar_l1_pct", event.getTarL1Pct());
            node.put("tar_l2_pct", event.getTarL2Pct());
            node.put("gmi", event.getGmi());
            node.put("gri", event.getGri());
            node.put("gri_zone", event.getGriZone());
            node.put("sustained_hypo_detected", event.isSustainedHypoDetected());
            node.put("sustained_severe_hypo_detected", event.isSustainedSevereHypoDetected());
            node.put("sustained_hyper_detected", event.isSustainedHyperDetected());
            node.put("nocturnal_hypo_detected", event.isNocturnalHypoDetected());
            node.put("rapid_rise_detected", event.isRapidRiseDetected());
            node.put("rapid_fall_detected", event.isRapidFallDetected());
            return node.toString();
        } catch (Exception e) {
            LOG.error("Module3 CGM: failed to serialize analytics event for {}: {}",
                    patientId, e.getMessage());
            return "{\"error\":\"serialization_failure\",\"patient_id\":\"" + patientId + "\"}";
        }
    }

    /**
     * Extract the patient_id field from a serialized analytics JSON as
     * bytes, used as the Kafka record key for partition affinity.
     * Returns empty bytes on failure so the record still ships without
     * a key (Kafka falls back to round-robin partitioning).
     */
    public static byte[] extractPatientIdBytes(String json) {
        try {
            JsonNode node = MAPPER.readTree(json);
            String pid = node.path("patient_id").asText("");
            return pid.getBytes();
        } catch (Exception ignored) {
            return new byte[0];
        }
    }

    // ─────────────── Env var helpers ───────────────

    /**
     * Read the Kafka bootstrap servers from the canonical Flink env var
     * with a localhost fallback for local dev. Matches
     * {@link Module3_ComprehensiveCDS#getBootstrapServers} exactly so
     * ops scripts that set KAFKA_BOOTSTRAP_SERVERS work for both jobs.
     */
    private static String getBootstrapServers() {
        String kafkaServers = System.getenv("KAFKA_BOOTSTRAP_SERVERS");
        return (kafkaServers != null && !kafkaServers.isEmpty())
                ? kafkaServers
                : "localhost:9092";
    }

    /**
     * Env-var override for a topic name with a default fallback.
     * Matches the pattern used by other Module 3 operators so ops can
     * point a running job at a different topic without recompiling.
     */
    private static String getTopicName(String envVar, String defaultTopic) {
        String topic = System.getenv(envVar);
        return (topic != null && !topic.isEmpty()) ? topic : defaultTopic;
    }
}
