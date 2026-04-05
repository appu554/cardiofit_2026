package com.cardiofit.flink.serialization;

import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.EventType;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
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
public class SourceTaggingDeserializer implements DeserializationSchema<CanonicalEvent> {

    private static final Logger LOG = LoggerFactory.getLogger(SourceTaggingDeserializer.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();
    private static final TypeReference<Map<String, Object>> MAP_TYPE = new TypeReference<>() {};

    private final String sourceModuleTag;
    private final EventType defaultEventType;

    public SourceTaggingDeserializer(String sourceModuleTag, EventType defaultEventType) {
        this.sourceModuleTag = sourceModuleTag;
        this.defaultEventType = defaultEventType;
    }

    @Override
    public CanonicalEvent deserialize(byte[] message) throws IOException {
        if (message == null || message.length == 0) return null;

        try {
            Map<String, Object> raw = MAPPER.readValue(message, MAP_TYPE);

            Map<String, Object> payload = new HashMap<>(raw);
            payload.put("source_module", sourceModuleTag);

            String patientId = extractString(raw, "patient_id", "patientId");

            long eventTime = extractLong(raw, "event_time", "eventTime", "timestamp",
                    "processing_timestamp", "observation_start_ms");

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

            return CanonicalEvent.builder()
                    .id(id)
                    .patientId(patientId)
                    .eventType(eventType)
                    .eventTime(eventTime)
                    .sourceSystem("flink-module13-" + sourceModuleTag)
                    .payload(payload)
                    .build();

        } catch (Exception e) {
            LOG.warn("Failed to deserialize {} event: {}", sourceModuleTag, e.getMessage());
            return null;
        }
    }

    @Override
    public boolean isEndOfStream(CanonicalEvent nextElement) {
        return false;
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

    private static long extractLong(Map<String, Object> map, String... keys) {
        for (String key : keys) {
            Object v = map.get(key);
            if (v instanceof Number) return ((Number) v).longValue();
            if (v != null) {
                try { return Long.parseLong(v.toString()); }
                catch (NumberFormatException ignored) {}
            }
        }
        return System.currentTimeMillis();
    }
}
