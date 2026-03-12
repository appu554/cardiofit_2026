
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
        DataStream<RawEvent> unifiedEventStream = createUnifiedEventStream(env);

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
     * Create a unified stream from multiple Kafka topics
     */
    private static DataStream<RawEvent> createUnifiedEventStream(StreamExecutionEnvironment env) {

        // Patient events stream
        DataStream<RawEvent> patientEvents = createKafkaSource(env,
            KafkaTopics.PATIENT_EVENTS, "patient-ingestion-v2");

        // Medication events stream
        DataStream<RawEvent> medicationEvents = createKafkaSource(env,
            KafkaTopics.MEDICATION_EVENTS, "medication-ingestion");

        // Observation events stream
        DataStream<RawEvent> observationEvents = createKafkaSource(env,
            KafkaTopics.OBSERVATION_EVENTS, "observation-ingestion");

        // Vital signs events stream
        DataStream<RawEvent> vitalEvents = createKafkaSource(env,
            KafkaTopics.VITAL_SIGNS_EVENTS, "vital-ingestion");

        // Lab result events stream
        DataStream<RawEvent> labEvents = createKafkaSource(env,
            KafkaTopics.LAB_RESULT_EVENTS, "lab-ingestion");

        // Device data stream
        DataStream<RawEvent> deviceEvents = createKafkaSource(env,
            KafkaTopics.VALIDATED_DEVICE_DATA, "device-ingestion");

        // Union all streams
        return patientEvents
            .union(medicationEvents)
            .union(observationEvents)
            .union(vitalEvents)
            .union(labEvents)
            .union(deviceEvents);
    }

    /**
     * Create Kafka source for a specific topic
     */
    private static DataStream<RawEvent> createKafkaSource(
            StreamExecutionEnvironment env,
            KafkaTopics topic,
            String consumerGroup) {

        KafkaSource<RawEvent> source = KafkaSource.<RawEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(topic.getTopicName())
            .setGroupId(consumerGroup)
            .setValueOnlyDeserializer(new RawEventDeserializer())
            // REMOVED: .setStartingOffsets() - causes ClassCastException, use auto.offset.reset from consumer config
            .build();

        return env.fromSource(source,
            WatermarkStrategy
                .<RawEvent>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                .withTimestampAssigner((event, timestamp) -> event.getEventTime()),
            "Kafka Source: " + topic.getTopicName());
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
            // 1. Patient ID validation - check both null and blank
            if (event.getPatientId() == null || event.getPatientId().trim().isEmpty()) {
                return ValidationResult.invalid("Missing or blank patient ID");
            }

            // 2. Event type validation - allow missing with warning (will default to UNKNOWN)
            if (event.getType() == null || event.getType().trim().isEmpty()) {
                LOG.warn("Missing event type for event {}, will default to UNKNOWN", event.getId());
                // Don't fail validation - parseEventType() will handle the default
            }

            // 3. Timestamp validation - explicit null/zero check
            if (event.getEventTime() <= 0) {
                return ValidationResult.invalid("Invalid or zero event timestamp");
            }

            // 4. Timestamp sanity checks
            long now = System.currentTimeMillis();

            // Event time should not be too far in the future (1 hour tolerance)
            if (event.getEventTime() > now + Duration.ofHours(1).toMillis()) {
                return ValidationResult.invalid("Event time too far in future (max 1 hour tolerance)");
            }

            // Event time should not be too old (30 days retention window)
            if (event.getEventTime() < now - Duration.ofDays(30).toMillis()) {
                return ValidationResult.invalid("Event time too old (>30 days, outside retention window)");
            }

            // 5. Payload validation
            if (event.getPayload() == null || event.getPayload().isEmpty()) {
                return ValidationResult.invalid("Missing or empty payload");
            }

            return ValidationResult.valid();
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
                .payload(normalizePayload(raw.getPayload()))
                .metadata(eventMetadata)  // Preserve clinical context metadata
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
        private final boolean valid;
        private final String errorMessage;

        private ValidationResult(boolean valid, String errorMessage) {
            this.valid = valid;
            this.errorMessage = errorMessage;
        }

        public static ValidationResult valid() {
            return new ValidationResult(true, null);
        }

        public static ValidationResult invalid(String errorMessage) {
            return new ValidationResult(false, errorMessage);
        }

        public boolean isValid() {
            return valid;
        }

        public String getErrorMessage() {
            return errorMessage;
        }
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
                .setKeySerializationSchema(event -> ((CanonicalEvent)event).getPatientId().getBytes())
                .setValueSerializationSchema(new CanonicalEventSerializer())
                .build())
            .setTransactionalIdPrefix("module1-enriched-events")
            .build();
    }

    /**
     * Create sink for Dead Letter Queue
     */
    private static KafkaSink<RawEvent> createDLQSink() {
        // Don't use setKafkaProducerConfig() when using custom serialization schemas
        // The RecordSerializer handles serialization, not the Kafka producer config
        return KafkaSink.<RawEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.DLQ_PROCESSING_ERRORS.getTopicName())
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
            return objectMapper.readValue(message, RawEvent.class);
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
