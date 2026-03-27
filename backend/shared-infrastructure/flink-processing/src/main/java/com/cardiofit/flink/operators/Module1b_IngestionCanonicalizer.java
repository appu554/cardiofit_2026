package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.models.OutboxEnvelope;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Module 1b: Ingestion Service Canonicalizer
 *
 * Separate Flink job that reads outbox events from ingestion.* topics,
 * validates and transforms them into CanonicalEvent format, and outputs to the
 * ENRICHED_PATIENT_EVENTS topic for downstream Module 2 processing.
 *
 * This module bridges the ingestion service's outbox SDK output format
 * into the same canonical pipeline used by Module 1 (EHR ingestion).
 * Key difference: ingestion events already have FHIR resources persisted
 * at Stage 3, so the downstream router skips FHIR persistence for these.
 *
 * Consumer group: flink-module1b-ingestion
 * Parallelism: 4
 * Checkpoint interval: 60s
 */
public class Module1b_IngestionCanonicalizer {
    private static final Logger LOG = LoggerFactory.getLogger(Module1b_IngestionCanonicalizer.class);

    private static final String CONSUMER_GROUP = "flink-module1b-ingestion";
    private static final String SOURCE_SYSTEM = "ingestion-service";
    private static final int PARALLELISM = 4;
    private static final long CHECKPOINT_INTERVAL_MS = 60_000L;

    // DLQ output tag — package-visible for test access
    static final OutputTag<OutboxEnvelope> DLQ_OUTPUT_TAG =
        new OutputTag<OutboxEnvelope>("dlq-ingestion-events"){};

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Module 1b: Ingestion Service Canonicalizer");

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        env.setParallelism(PARALLELISM);
        env.enableCheckpointing(CHECKPOINT_INTERVAL_MS);
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(10_000L);
        env.getCheckpointConfig().setCheckpointTimeout(600_000L);

        createIngestionPipeline(env);

        env.execute("Module 1b: Ingestion Service Canonicalizer");
    }

    /**
     * Build the ingestion canonicalization pipeline:
     * ingestion.* topics -> deserialize OutboxEnvelope -> validate + canonicalize -> ENRICHED_PATIENT_EVENTS
     * Invalid events route to DLQ side output.
     */
    public static void createIngestionPipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating ingestion canonicalization pipeline for {} topics", 9);

        DataStream<OutboxEnvelope> unifiedStream = createUnifiedIngestionStream(env);

        SingleOutputStreamOperator<CanonicalEvent> canonicalEvents = unifiedStream
            .process(new OutboxValidationAndCanonicalization())
            .uid("Outbox Validation & Canonicalization")
            .name("Outbox Validation & Canonicalization");

        canonicalEvents
            .sinkTo(createCanonicalEventSink())
            .uid("Module1b Canonical Events Sink")
            .name("Module1b Canonical Events Sink");

        // DLQ sink for failed events
        canonicalEvents.getSideOutput(DLQ_OUTPUT_TAG)
            .sinkTo(createDLQSink())
            .uid("Module1b DLQ Sink")
            .name("Module1b DLQ Sink");

        LOG.info("Ingestion canonicalization pipeline created successfully");
    }

    /**
     * Create a unified stream from all 9 ingestion.* topics.
     * Uses a single KafkaSource with topic list for efficient consumer group management.
     */
    private static DataStream<OutboxEnvelope> createUnifiedIngestionStream(StreamExecutionEnvironment env) {
        List<String> ingestionTopics = new ArrayList<>();
        ingestionTopics.add(KafkaTopics.INGESTION_LABS.getTopicName());
        ingestionTopics.add(KafkaTopics.INGESTION_VITALS.getTopicName());
        ingestionTopics.add(KafkaTopics.INGESTION_DEVICE_DATA.getTopicName());
        ingestionTopics.add(KafkaTopics.INGESTION_PATIENT_REPORTED.getTopicName());
        ingestionTopics.add(KafkaTopics.INGESTION_WEARABLE_AGGREGATES.getTopicName());
        ingestionTopics.add(KafkaTopics.INGESTION_CGM_RAW.getTopicName());
        ingestionTopics.add(KafkaTopics.INGESTION_ABDM_RECORDS.getTopicName());
        ingestionTopics.add(KafkaTopics.INGESTION_MEDICATIONS.getTopicName());
        ingestionTopics.add(KafkaTopics.INGESTION_OBSERVATIONS.getTopicName());
        // NOTE: ingestion.safety-critical is NOT consumed here — critical events
        // are already on their source topics (e.g., ingestion.labs via dual-publish).
        // Subscribing would cause duplicate processing in the Flink pipeline.
        // KB-22 consumes ingestion.safety-critical directly for fast deterioration detection.
        LOG.info("Excluding ingestion.safety-critical topic — consumed directly by KB-22 for fast deterioration detection");

        KafkaSource<OutboxEnvelope> source = KafkaSource.<OutboxEnvelope>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(ingestionTopics)
            .setGroupId(CONSUMER_GROUP)
            .setValueOnlyDeserializer(new OutboxEnvelopeDeserializer())
            .build();

        return env.fromSource(source,
            WatermarkStrategy
                .<OutboxEnvelope>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                .withTimestampAssigner((envelope, timestamp) -> extractTimestamp(envelope))
                .withIdleness(Duration.ofMinutes(2)),
            "Kafka Source: ingestion.* topics (9)");
    }

    /**
     * Extract event timestamp from OutboxEnvelope for watermark assignment.
     * Falls back to current time if event_data timestamp is missing or unparseable.
     */
    private static long extractTimestamp(OutboxEnvelope envelope) {
        if (envelope.getEventData() != null && envelope.getEventData().getTimestamp() != null) {
            try {
                return Instant.parse(envelope.getEventData().getTimestamp()).toEpochMilli();
            } catch (Exception e) {
                LOG.debug("Could not parse timestamp '{}', using current time",
                    envelope.getEventData().getTimestamp());
            }
        }
        return System.currentTimeMillis();
    }

    /**
     * Validates and canonicalizes OutboxEnvelope events.
     * Routes invalid events (null eventData, null patientId, unparseable timestamp) to DLQ side output.
     * Public visibility for test harness access.
     */
    public static class OutboxValidationAndCanonicalization
            extends ProcessFunction<OutboxEnvelope, CanonicalEvent> {

        private static final long serialVersionUID = 1L;

        @Override
        public void processElement(OutboxEnvelope envelope, Context ctx, Collector<CanonicalEvent> out) {
            try {
                OutboxEnvelope.IngestionEventData data = envelope.getEventData();

                // Validate: null eventData -> DLQ
                if (data == null) {
                    LOG.warn("OutboxEnvelope {} has null event_data, routing to DLQ", envelope.getId());
                    ctx.output(DLQ_OUTPUT_TAG, envelope);
                    return;
                }

                // Validate: null/blank patientId -> DLQ (Q7 fix)
                if (data.getPatientId() == null || data.getPatientId().trim().isEmpty()) {
                    LOG.warn("OutboxEnvelope {} has null/blank patient_id, routing to DLQ", envelope.getId());
                    ctx.output(DLQ_OUTPUT_TAG, envelope);
                    return;
                }

                // Parse event timestamp — null or unparseable -> DLQ (A7 fix: don't silently use currentTimeMillis)
                long eventTime;
                if (data.getTimestamp() == null) {
                    LOG.warn("OutboxEnvelope {} has null timestamp, routing to DLQ", envelope.getId());
                    ctx.output(DLQ_OUTPUT_TAG, envelope);
                    return;
                }
                try {
                    eventTime = Instant.parse(data.getTimestamp()).toEpochMilli();
                } catch (Exception e) {
                    LOG.warn("OutboxEnvelope {} has unparseable timestamp '{}', routing to DLQ",
                        envelope.getId(), data.getTimestamp());
                    ctx.output(DLQ_OUTPUT_TAG, envelope);
                    return;
                }

                // Map observation_type to EventType
                EventType eventType = mapObservationTypeToEventType(data.getObservationType());

                // Build payload map with clinical observation data
                // V4: Data tier and CGM flags set BEFORE building CanonicalEvent (Q6 fix)
                Map<String, Object> payload = new HashMap<>();
                payload.put("loinc_code", data.getLoincCode());
                if (data.getValue() != null) {
                    payload.put("value", data.getValue());
                }
                payload.put("unit", data.getUnit());
                payload.put("observation_type", data.getObservationType());
                if (data.getQualityScore() != null) {
                    payload.put("quality_score", data.getQualityScore());
                }
                payload.put("source_type", data.getSourceType());
                payload.put("source_id", data.getSourceId());
                payload.put("fhir_resource_id", data.getFhirResourceId());
                if (data.getFlags() != null) {
                    payload.put("flags", data.getFlags());
                }

                // V4: Data tier classification — done before builder so payload is complete
                String sourceType = data.getSourceType() != null ? data.getSourceType().toUpperCase() : "";
                String obsType = data.getObservationType() != null ? data.getObservationType().toUpperCase() : "";
                if (obsType.contains("CGM")) {
                    payload.put("data_tier", "TIER_1_CGM");
                    payload.put("cgm_active", true);
                } else if (sourceType.contains("WEARABLE") || obsType.contains("DEVICE")) {
                    payload.put("data_tier", "TIER_2_HYBRID");
                    payload.put("cgm_active", false);
                } else {
                    payload.put("data_tier", "TIER_3_SMBG");
                    payload.put("cgm_active", false);
                }

                // Build metadata from envelope
                CanonicalEvent.EventMetadata metadata = new CanonicalEvent.EventMetadata(
                    data.getSourceType() != null ? data.getSourceType() : SOURCE_SYSTEM,
                    data.getTenantId() != null ? data.getTenantId() : "UNKNOWN",
                    data.getSourceId() != null ? data.getSourceId() : "UNKNOWN"
                );

                CanonicalEvent canonical = CanonicalEvent.builder()
                    .id(data.getEventId() != null ? data.getEventId() : UUID.randomUUID().toString())
                    .patientId(data.getPatientId())
                    .eventType(eventType)
                    .eventTime(eventTime)
                    .sourceSystem(SOURCE_SYSTEM)
                    .payload(payload)
                    .metadata(metadata)
                    .correlationId(envelope.getCorrelationId())
                    .build();

                LOG.debug("Canonicalized ingestion event: id={}, type={}, patient={}, data_tier={}",
                    canonical.getId(), eventType, data.getPatientId(), payload.get("data_tier"));

                out.collect(canonical);

            } catch (Exception e) {
                LOG.error("Unexpected error processing OutboxEnvelope {}: {}",
                    envelope.getId(), e.getMessage(), e);
                ctx.output(DLQ_OUTPUT_TAG, envelope);
            }
        }

        /**
         * Map ingestion service observation_type strings to canonical EventType values.
         *
         * Mapping strategy:
         * - LABS -> LAB_RESULT
         * - VITALS -> VITAL_SIGN
         * - DEVICE_DATA, WEARABLE, CGM -> DEVICE_READING
         * - MEDICATIONS -> MEDICATION_ORDERED
         * - PATIENT_REPORTED -> PATIENT_REPORTED (V4 — self-reported symptoms, meals, exercise)
         * - OBSERVATIONS -> LAB_RESULT (clinician-recorded observations)
         * - ABDM_RECORDS -> CLINICAL_DOCUMENT (V4 — discharge summaries, prescriptions)
         * - SAFETY_CRITICAL -> ADVERSE_EVENT
         * - Fallback: EventType.fromString() for unmapped types
         */
        private EventType mapObservationTypeToEventType(String observationType) {
            if (observationType == null || observationType.trim().isEmpty()) {
                return EventType.UNKNOWN;
            }

            String normalized = observationType.toUpperCase().replace("-", "_").replace(" ", "_");

            switch (normalized) {
                case "LABS":
                case "LAB":
                case "LABORATORY":
                    return EventType.LAB_RESULT;

                case "VITALS":
                case "VITAL":
                case "VITAL_SIGNS":
                    return EventType.VITAL_SIGN;

                case "DEVICE_DATA":
                case "WEARABLE":
                case "WEARABLE_AGGREGATES":
                case "CGM":
                case "CGM_RAW":
                    return EventType.DEVICE_READING;

                case "MEDICATIONS":
                case "MEDICATION":
                    return EventType.MEDICATION_ORDERED;

                case "PATIENT_REPORTED":
                    return EventType.PATIENT_REPORTED;

                case "OBSERVATIONS":
                    return EventType.LAB_RESULT;

                case "ABDM_RECORDS":
                case "ABDM":
                    return EventType.CLINICAL_DOCUMENT;

                case "SAFETY_CRITICAL":
                    return EventType.ADVERSE_EVENT;

                default:
                    return EventType.fromString(normalized);
            }
        }
    }

    /**
     * Create Kafka sink for CanonicalEvent output to ENRICHED_PATIENT_EVENTS.
     * This feeds directly into Module 2 (Context Assembly) for enrichment.
     */
    private static KafkaSink<CanonicalEvent> createCanonicalEventSink() {
        return KafkaSink.<CanonicalEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
                .setKeySerializationSchema(event ->
                    ((CanonicalEvent) event).getPatientId() != null
                        ? ((CanonicalEvent) event).getPatientId().getBytes()
                        : "UNKNOWN".getBytes())
                .setValueSerializationSchema(new CanonicalEventSerializer())
                .build())
            .setTransactionalIdPrefix("module1b-ingestion-canonical")
            .setDeliveryGuarantee(org.apache.flink.connector.base.DeliveryGuarantee.EXACTLY_ONCE)
            .build();
    }

    /**
     * Create Kafka sink for DLQ events.
     * Uses crash-safe serializer — DLQ must never throw (reviewer fix: DLQ poisoning anti-pattern).
     */
    private static KafkaSink<OutboxEnvelope> createDLQSink() {
        return KafkaSink.<OutboxEnvelope>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.DLQ_PROCESSING_ERRORS.getTopicName())
                .setKeySerializationSchema(event -> {
                    OutboxEnvelope env = (OutboxEnvelope) event;
                    if (env.getEventData() != null && env.getEventData().getPatientId() != null) {
                        return env.getEventData().getPatientId().getBytes();
                    }
                    return "UNKNOWN".getBytes();
                })
                .setValueSerializationSchema(new SafeOutboxEnvelopeSerializer())
                .build())
            .setTransactionalIdPrefix("module1b-dlq-errors")
            .build();
    }

    private static String getBootstrapServers() {
        String envServers = System.getenv("KAFKA_BOOTSTRAP_SERVERS");
        if (envServers != null && !envServers.isEmpty()) {
            return envServers;
        }
        return KafkaConfigLoader.isRunningInDocker()
            ? "kafka:29092"
            : "localhost:9092";
    }

    // --- Serialization / Deserialization ---

    private static class OutboxEnvelopeDeserializer implements DeserializationSchema<OutboxEnvelope> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public OutboxEnvelope deserialize(byte[] message) throws IOException {
            if (message == null || message.length == 0) {
                LOG.warn("Received null or empty message in Module1b, skipping");
                return null;
            }
            try {
                return objectMapper.readValue(message, OutboxEnvelope.class);
            } catch (Exception e) {
                LOG.error("Failed to deserialize OutboxEnvelope ({} bytes): {}. Message dropped.",
                    message.length, e.getMessage());
                return null;
            }
        }

        @Override
        public boolean isEndOfStream(OutboxEnvelope nextElement) {
            return false;
        }

        @Override
        public TypeInformation<OutboxEnvelope> getProducedType() {
            return TypeInformation.of(OutboxEnvelope.class);
        }
    }

    private static class CanonicalEventSerializer implements SerializationSchema<CanonicalEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(CanonicalEvent element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                throw new RuntimeException("Failed to serialize CanonicalEvent", e);
            }
        }
    }

    /**
     * Crash-safe serializer for DLQ events. Must NEVER throw — a DLQ that
     * crashes the job is worse than no DLQ at all (reviewer fix: DLQ poisoning).
     */
    private static class SafeOutboxEnvelopeSerializer implements SerializationSchema<OutboxEnvelope> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(OutboxEnvelope element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                // Fallback: minimal JSON with envelope ID so the event is not lost entirely
                LOG.error("Failed to serialize OutboxEnvelope for DLQ, using fallback: {}", e.getMessage());
                String fallback = "{\"dlq_serialization_error\":true,\"envelope_id\":\""
                    + (element.getId() != null ? element.getId() : "UNKNOWN")
                    + "\",\"error\":\"" + e.getMessage().replace("\"", "'") + "\"}";
                return fallback.getBytes();
            }
        }
    }
}
