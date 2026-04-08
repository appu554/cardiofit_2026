package com.cardiofit.flink.serialization;

import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.EventType;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.connector.kafka.source.reader.deserializer.KafkaRecordDeserializationSchema;
import org.apache.flink.util.Collector;
import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

/**
 * Generic deserializer that wraps raw JSON into a CanonicalEvent and injects
 * a source_module tag. This solves two problems:
 * 1. Upstream modules don't emit source_module in their payload.
 * 2. Sources 5-6 emit InterventionWindowSignal/InterventionDeltaRecord, not CanonicalEvent.
 *
 * Parameterized per topic at construction time.
 */
public class SourceTaggingDeserializer implements KafkaRecordDeserializationSchema<CanonicalEvent>, Serializable {

    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(SourceTaggingDeserializer.class);
    private static final TypeReference<Map<String, Object>> MAP_TYPE = new TypeReference<>() {};

    private transient ObjectMapper mapper;

    private final String sourceModuleTag;
    private final EventType defaultEventType;

    public SourceTaggingDeserializer(String sourceModuleTag, EventType defaultEventType) {
        this.sourceModuleTag = sourceModuleTag;
        this.defaultEventType = defaultEventType;
    }

    @Override
    public void deserialize(ConsumerRecord<byte[], byte[]> record, Collector<CanonicalEvent> out) {
        byte[] message = record.value();
        if (message == null || message.length == 0) return;

        if (mapper == null) {
            mapper = new ObjectMapper();
        }

        try {
            Map<String, Object> raw = mapper.readValue(message, MAP_TYPE);

            Map<String, Object> payload = new HashMap<>(raw);
            payload.put("source_module", sourceModuleTag);

            String patientId = extractString(raw, "patient_id", "patientId");

            long eventTime = extractLong(raw, "event_time", "eventTime", "timestamp",
                    "processing_timestamp", "observation_start_ms", "computed_at");

            EventType eventType = defaultEventType;
            String typeStr = extractString(raw, "event_type", "eventType");
            if (typeStr != null) {
                try { eventType = EventType.valueOf(typeStr); }
                catch (IllegalArgumentException ignored) {}
            }

            String id = extractString(raw, "id", "event_id");
            if (id == null) {
                id = UUID.randomUUID().toString();
            }

            CanonicalEvent event = CanonicalEvent.builder()
                    .id(id)
                    .patientId(patientId)
                    .eventType(eventType)
                    .eventTime(eventTime)
                    .sourceSystem("flink-module13-" + sourceModuleTag)
                    .payload(payload)
                    .build();

            out.collect(event);

        } catch (Exception e) {
            LOG.warn("Failed to deserialize {} event: {}", sourceModuleTag, e.getMessage());
        }
    }

    @Override
    public TypeInformation<CanonicalEvent> getProducedType() {
        return TypeInformation.of(CanonicalEvent.class);
    }

    private static String extractString(Map<String, Object> map, String... keys) {
        for (String key : keys) {
            Object v = map.get(key);
            if (v != null) return v.toString();
        }
        return null;
    }

    private long extractLong(Map<String, Object> map, String... keys) {
        for (String key : keys) {
            Object v = map.get(key);
            if (v instanceof Number) return ((Number) v).longValue();
            if (v != null) {
                try { return Long.parseLong(v.toString()); }
                catch (NumberFormatException ignored) {}
            }
        }
        LOG.debug("No timestamp field found for {} event, falling back to wall-clock time", sourceModuleTag);
        return System.currentTimeMillis();
    }
}
