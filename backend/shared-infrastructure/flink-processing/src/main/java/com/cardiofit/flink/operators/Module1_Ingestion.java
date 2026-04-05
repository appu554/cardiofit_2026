
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.RawEvent;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.functions.MapFunction;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.connector.base.DeliveryGuarantee;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.apache.kafka.clients.consumer.OffsetResetStrategy;
import org.apache.kafka.clients.producer.ProducerRecord;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.annotation.Nullable;
import java.io.IOException;
import java.time.Duration;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Module 1: Ingestion & Gateway
 *
 * Responsibilities:
 * - Consume events from multiple Kafka topics
 * - Validate and canonicalize events
 * - Route invalid events to Dead Letter Queue
 * - Output clean, validated events for downstream processing
 */
public class Module1_Ingestion {
    private static final Logger LOG = LoggerFactory.getLogger(Module1_Ingestion.class);

    // Dead Letter Queue output tag
    private static final OutputTag<RawEvent> DLQ_OUTPUT_TAG =
        new OutputTag<RawEvent>("dlq-events"){};

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Module 1: EHR Event Ingestion & Gateway");

        // Set up execution environment
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Configure environment for healthcare workloads
        env.setParallelism(2);
        env.enableCheckpointing(30000); // 30 second checkpoints
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);
        env.getCheckpointConfig().setCheckpointTimeout(600000); // 10 minutes

        // Create ingestion pipeline
        createIngestionPipeline(env);

        // Execute the job
        env.execute("Module 1: EHR Event Ingestion");
    }

    public static void createIngestionPipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating ingestion pipeline for clinical events");

        // Create unified stream from multiple clinical event topics
        DataStream<RawEvent> unifiedEventStream = createUnifiedEventStream(env)
            .filter(event -> event != null)
            .uid("Null Event Filter");

        // Process and validate events
        SingleOutputStreamOperator<CanonicalEvent> processedEvents = unifiedEventStream
            .process(new EventValidationAndCanonicalization())
            .uid("Event Validation & Canonicalization");

        // Send validated events to enriched events topic
        processedEvents
            .sinkTo(createCleanEventsSink())
            .uid("Clean Events Sink");

        // Send failed events to Dead Letter Queue
        processedEvents.getSideOutput(DLQ_OUTPUT_TAG)
            .sinkTo(createDLQSink())
            .uid("Dead Letter Queue Sink");

        LOG.info("Ingestion pipeline created successfully");
    }

    /**
     * Create a unified stream from all 6 legacy EHR topics using a single KafkaSource.
     * Single consumer group (A2 fix) + watermark idleness (A3 fix).
     */
    private static DataStream<RawEvent> createUnifiedEventStream(StreamExecutionEnvironment env) {
        List<String> ehrTopics = java.util.Arrays.asList(
            KafkaTopics.PATIENT_EVENTS.getTopicName(),
            KafkaTopics.MEDICATION_EVENTS.getTopicName(),
            KafkaTopics.OBSERVATION_EVENTS.getTopicName(),
            KafkaTopics.VITAL_SIGNS_EVENTS.getTopicName(),
            KafkaTopics.LAB_RESULT_EVENTS.getTopicName(),
            KafkaTopics.VALIDATED_DEVICE_DATA.getTopicName()
        );

        KafkaSource<RawEvent> source = KafkaSource.<RawEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(ehrTopics)
            .setGroupId("flink-module1-ingestion")
            .setValueOnlyDeserializer(new RawEventDeserializer())
            .build();

        return env.fromSource(source,
            WatermarkStrategy
                .<RawEvent>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                .withTimestampAssigner((event, timestamp) -> event.getEventTime())
                .withIdleness(Duration.ofMinutes(2)),
            "Kafka Source: EHR topics (6)");
    }

    /**
     * Event validation and canonicalization processor
     */
    public static class EventValidationAndCanonicalization
            extends ProcessFunction<RawEvent, CanonicalEvent> {

        private transient ObjectMapper objectMapper;

        // @Override - Removed for Flink 2.x
        public void open(org.apache.flink.configuration.Configuration parameters) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public void processElement(RawEvent rawEvent, Context ctx, Collector<CanonicalEvent> out) {
            try {
                // Validate the raw event
                ValidationResult validation = validateEvent(rawEvent);

                if (!validation.isValid()) {
                    LOG.warn("Event validation failed for event {}: {}",
                        rawEvent.getId(), validation.getErrorMessage());

                    // Route to Dead Letter Queue
                    ctx.output(DLQ_OUTPUT_TAG, rawEvent);
                    return;
                }

                // Canonicalize the event
                CanonicalEvent canonical = canonicalizeEvent(rawEvent, ctx);

                // Wire validation notes into the event (reviewer fix: get-or-create, don't overwrite)
                if (validation.isSanitized()) {
                    CanonicalEvent.IngestionMetadata ingMeta = canonical.getIngestionMetadata();
                    if (ingMeta == null) {
                        ingMeta = new CanonicalEvent.IngestionMetadata();
                    }
                    ingMeta.setValidationStatus("SANITIZED");
                    canonical.setIngestionMetadata(ingMeta);
                }

                // Emit the canonical event
                out.collect(canonical);

                LOG.debug("Successfully processed event: {}", canonical.getId());

            } catch (Exception e) {
                LOG.error("Failed to process event: " + rawEvent.getId(), e);

                // Route to DLQ on any processing error
                ctx.output(DLQ_OUTPUT_TAG, rawEvent);
            }
        }

        private ValidationResult validateEvent(RawEvent event) {
            java.util.List<String> notes = new java.util.ArrayList<>();

            // 1. Patient ID validation - check both null and blank
            if (event.getPatientId() == null || event.getPatientId().trim().isEmpty()) {
                return ValidationResult.invalid("Missing or blank patient ID");
            }

            // 2. Event type validation - allow missing with warning (will default to UNKNOWN)
            if (event.getType() == null || event.getType().trim().isEmpty()) {
                LOG.info("Missing event type for event {}, will default to UNKNOWN", event.getId());
                notes.add("event_type defaulted to UNKNOWN");
            }

            // 3. Timestamp validation - explicit null/zero check
            if (event.getEventTime() <= 0) {
                return ValidationResult.invalid("Invalid or zero event timestamp");
            }

            // 4. Timestamp sanity checks — CLAMP instead of reject (C1 fix)
            long now = System.currentTimeMillis();
            long maxFuture = now + Duration.ofHours(1).toMillis();
            long maxPast = now - Duration.ofDays(30).toMillis();

            if (event.getEventTime() > maxFuture) {
                long originalTime = event.getEventTime();
                event.setEventTime(maxFuture);
                LOG.info("Event {} timestamp {} clamped from future to now+1h", event.getId(), originalTime);
                notes.add("timestamp clamped from future (" + originalTime + ") to now+1h");
            }

            if (event.getEventTime() < maxPast) {
                long originalTime = event.getEventTime();
                event.setEventTime(maxPast);
                LOG.info("Event {} timestamp {} clamped from >30d past to 30d boundary", event.getId(), originalTime);
                notes.add("timestamp clamped from >30d past (" + originalTime + ") to 30d boundary");
            }

            // 5. Payload validation
            if (event.getPayload() == null || event.getPayload().isEmpty()) {
                return ValidationResult.invalid("Missing or empty payload");
            }

            return notes.isEmpty() ? ValidationResult.valid() : ValidationResult.sanitized(notes);
        }

        private CanonicalEvent canonicalizeEvent(RawEvent raw, Context ctx) {
            // Extract and preserve metadata from raw event
            CanonicalEvent.EventMetadata eventMetadata = extractMetadata(raw);

            return CanonicalEvent.builder()
                .id(raw.getId() != null ? raw.getId() : java.util.UUID.randomUUID().toString())
                .patientId(raw.getPatientId())
                .encounterId(null)  // Explicitly null for Module 1, hydrated in Module 2
                .eventType(parseEventType(raw.getType()))
                .eventTime(raw.getEventTime())
                .sourceSystem("legacy-ehr")
                .payload(normalizePayload(raw.getPayload()))
                .metadata(eventMetadata)  // Preserve clinical context metadata
                .correlationId(raw.getCorrelationId() != null ? raw.getCorrelationId() : java.util.UUID.randomUUID().toString())
                .build();
        }

        /**
         * Extract metadata from RawEvent, providing defaults for missing values
         */
        private CanonicalEvent.EventMetadata extractMetadata(RawEvent raw) {
            Map<String, String> rawMeta = raw.getMetadata();

            if (rawMeta == null || rawMeta.isEmpty()) {
                // Return defaults when metadata is missing
                return new CanonicalEvent.EventMetadata("UNKNOWN", "UNKNOWN", "UNKNOWN");
            }

            return new CanonicalEvent.EventMetadata(
                rawMeta.getOrDefault("source", "UNKNOWN"),
                rawMeta.getOrDefault("location", "UNKNOWN"),
                rawMeta.getOrDefault("device_id", "UNKNOWN")
            );
        }

        /**
         * Parse event type with fallback to UNKNOWN for missing/invalid types
         */
        private EventType parseEventType(String type) {
            if (type == null || type.trim().isEmpty()) {
                LOG.warn("Missing event type, defaulting to UNKNOWN");
                return EventType.UNKNOWN;
            }

            // Use EventType.fromString() which has intelligent mapping logic
            EventType parsed = EventType.fromString(type);

            if (parsed == EventType.UNKNOWN) {
                LOG.warn("Could not map event type '{}' to known EventType, defaulting to UNKNOWN", type);
            }

            return parsed;
        }

        @SuppressWarnings("unchecked")
        private Map<String, Object> normalizePayload(Map<String, Object> payload) {
            Map<String, Object> normalized = new HashMap<>();

            for (Map.Entry<String, Object> entry : payload.entrySet()) {
                String key = entry.getKey().toLowerCase().replace("-", "_");
                Object value = entry.getValue();

                // Normalize common clinical data types
                if (value instanceof String) {
                    String strValue = (String) value;

                    // Try to parse numeric values
                    if (isNumeric(strValue)) {
                        try {
                            if (strValue.contains(".")) {
                                normalized.put(key, Double.parseDouble(strValue));
                            } else {
                                normalized.put(key, Long.parseLong(strValue));
                            }
                        } catch (NumberFormatException e) {
                            normalized.put(key, strValue);
                        }
                    } else {
                        normalized.put(key, strValue.trim());
                    }
                } else {
                    normalized.put(key, value);
                }
            }

            return normalized;
        }

        private boolean isNumeric(String str) {
            if (str == null || str.trim().isEmpty()) {
                return false;
            }
            try {
                Double.parseDouble(str);
                return true;
            } catch (NumberFormatException e) {
                return false;
            }
        }
    }

    /**
     * Validation result helper class
     */
    private static class ValidationResult {
        enum Status { VALID, SANITIZED, INVALID }

        private final Status status;
        private final String errorMessage;
        private final java.util.List<String> notes;

        private ValidationResult(Status status, String errorMessage, java.util.List<String> notes) {
            this.status = status;
            this.errorMessage = errorMessage;
            this.notes = notes;
        }

        public static ValidationResult valid() {
            return new ValidationResult(Status.VALID, null, java.util.Collections.emptyList());
        }

        public static ValidationResult sanitized(java.util.List<String> notes) {
            return new ValidationResult(Status.SANITIZED, null, notes);
        }

        public static ValidationResult invalid(String errorMessage) {
            return new ValidationResult(Status.INVALID, errorMessage, java.util.Collections.emptyList());
        }

        public boolean isValid() { return status != Status.INVALID; }
        public boolean isSanitized() { return status == Status.SANITIZED; }
        public String getErrorMessage() { return errorMessage; }
        public java.util.List<String> getNotes() { return notes; }
    }

    /**
     * Create sink for clean, validated events
     */
    private static KafkaSink<CanonicalEvent> createCleanEventsSink() {
        // Don't use setKafkaProducerConfig() when using custom serialization schemas
        // The RecordSerializer handles serialization, not the Kafka producer config
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
            .setTransactionalIdPrefix("module1-enriched-events")
            .setDeliveryGuarantee(DeliveryGuarantee.EXACTLY_ONCE)
            .build();
    }

    /**
     * Create sink for Dead Letter Queue
     */
    private static KafkaSink<RawEvent> createDLQSink() {
        return KafkaSink.<RawEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.DLQ_PROCESSING_ERRORS.getTopicName())
                .setKeySerializationSchema(event ->
                    ((RawEvent) event).getPatientId() != null
                        ? ((RawEvent) event).getPatientId().getBytes()
                        : "UNKNOWN".getBytes())
                .setValueSerializationSchema(new RawEventSerializer())
                .build())
            .setTransactionalIdPrefix("module1-dlq-errors")
            .build();
    }

    private static String getBootstrapServers() {
        return KafkaConfigLoader.isRunningInDocker()
            ? "kafka:29092"
            : "localhost:9092";
    }

    // Serialization schemas
    private static class RawEventDeserializer implements DeserializationSchema<RawEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public RawEvent deserialize(byte[] message) throws IOException {
            if (message == null || message.length == 0) {
                LOG.warn("Received null or empty message, skipping");
                return null;
            }
            try {
                return objectMapper.readValue(message, RawEvent.class);
            } catch (Exception e) {
                LOG.error("Failed to deserialize raw event ({} bytes): {}. Message dropped.",
                    message.length, e.getMessage());
                return null;
            }
        }

        @Override
        public boolean isEndOfStream(RawEvent nextElement) {
            return false;
        }

        @Override
        public TypeInformation<RawEvent> getProducedType() {
            return TypeInformation.of(RawEvent.class);
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

    private static class RawEventSerializer implements SerializationSchema<RawEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(RawEvent element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                throw new RuntimeException("Failed to serialize RawEvent", e);
            }
        }
    }
}
