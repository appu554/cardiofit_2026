package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.models.OutboxEnvelope;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.functions.MapFunction;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
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
 * transforms them into CanonicalEvent format, and outputs to the
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
     * ingestion.* topics -> deserialize OutboxEnvelope -> map to CanonicalEvent -> ENRICHED_PATIENT_EVENTS
     */
    public static void createIngestionPipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating ingestion canonicalization pipeline for {} topics", 10);

        DataStream<OutboxEnvelope> unifiedStream = createUnifiedIngestionStream(env);

        DataStream<CanonicalEvent> canonicalEvents = unifiedStream
            .map(new OutboxToCanonicalMapper())
            .uid("Outbox-to-Canonical Mapper")
            .name("Outbox-to-Canonical Mapper");

        canonicalEvents
            .sinkTo(createCanonicalEventSink())
            .uid("Module1b Canonical Events Sink")
            .name("Module1b Canonical Events Sink");

        LOG.info("Ingestion canonicalization pipeline created successfully");
    }

    /**
     * Create a unified stream from all 10 ingestion.* topics.
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

        KafkaSource<OutboxEnvelope> source = KafkaSource.<OutboxEnvelope>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(ingestionTopics)
            .setGroupId(CONSUMER_GROUP)
            .setValueOnlyDeserializer(new OutboxEnvelopeDeserializer())
            .build();

        return env.fromSource(source,
            WatermarkStrategy
                .<OutboxEnvelope>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                .withTimestampAssigner((envelope, timestamp) -> extractTimestamp(envelope)),
            "Kafka Source: ingestion.* topics");
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
     * Maps OutboxEnvelope to CanonicalEvent.
     *
     * Sets sourceSystem = "ingestion-service" so downstream routing can identify
     * these events and skip FHIR persistence (already done at ingestion Stage 3).
     */
    static class OutboxToCanonicalMapper implements MapFunction<OutboxEnvelope, CanonicalEvent> {
        private static final long serialVersionUID = 1L;

        @Override
        public CanonicalEvent map(OutboxEnvelope envelope) throws Exception {
            OutboxEnvelope.IngestionEventData data = envelope.getEventData();

            if (data == null) {
                LOG.warn("OutboxEnvelope {} has null event_data, creating minimal canonical event",
                    envelope.getId());
                return CanonicalEvent.builder()
                    .id(envelope.getId() != null ? envelope.getId() : UUID.randomUUID().toString())
                    .patientId("UNKNOWN")
                    .eventType(EventType.UNKNOWN)
                    .eventTime(System.currentTimeMillis())
                    .sourceSystem(SOURCE_SYSTEM)
                    .correlationId(envelope.getCorrelationId())
                    .build();
            }

            // Map observation_type to EventType
            EventType eventType = mapObservationTypeToEventType(data.getObservationType());

            // Parse event timestamp
            long eventTime = System.currentTimeMillis();
            if (data.getTimestamp() != null) {
                try {
                    eventTime = Instant.parse(data.getTimestamp()).toEpochMilli();
                } catch (Exception e) {
                    LOG.debug("Could not parse event_data timestamp '{}', using current time",
                        data.getTimestamp());
                }
            }

            // Build payload map with clinical observation data
            Map<String, Object> payload = new HashMap<>();
            payload.put("loinc_code", data.getLoincCode());
            payload.put("value", data.getValue());
            payload.put("unit", data.getUnit());
            payload.put("observation_type", data.getObservationType());
            payload.put("quality_score", data.getQualityScore());
            payload.put("source_type", data.getSourceType());
            payload.put("source_id", data.getSourceId());
            payload.put("fhir_resource_id", data.getFhirResourceId());
            if (data.getFlags() != null) {
                payload.put("flags", data.getFlags());
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

            // V4: Add data tier and CGM activity flags for downstream MHRI computation
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

            LOG.debug("Canonicalized ingestion event: id={}, type={}, patient={}, data_tier={}",
                canonical.getId(), eventType, data.getPatientId(), payload.get("data_tier"));

            return canonical;
        }

        /**
         * Map ingestion service observation_type strings to canonical EventType values.
         *
         * Mapping strategy:
         * - LABS -> LAB_RESULT
         * - VITALS -> VITAL_SIGN
         * - DEVICE_DATA, WEARABLE, CGM -> DEVICE_READING
         * - MEDICATIONS -> MEDICATION_ORDERED
         * - PATIENT_REPORTED -> LAB_RESULT (observation-based)
         * - ABDM_RECORDS -> LAB_RESULT (default clinical observation)
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
                case "OBSERVATIONS":
                    return EventType.LAB_RESULT;

                case "ABDM_RECORDS":
                case "ABDM":
                    return EventType.LAB_RESULT;

                case "SAFETY_CRITICAL":
                    return EventType.ADVERSE_EVENT;

                default:
                    // Fall through to EventType's own mapping logic
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
            return objectMapper.readValue(message, OutboxEnvelope.class);
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
}
