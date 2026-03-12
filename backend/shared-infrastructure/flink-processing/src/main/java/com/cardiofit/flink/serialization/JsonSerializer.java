package com.cardiofit.flink.serialization;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import org.apache.flink.api.common.serialization.SerializationSchema;

/**
 * Custom JSON serializer for Kafka sinks
 * Uses Jackson ObjectMapper for JSON serialization
 *
 * @param <T> Type to serialize
 */
public class JsonSerializer<T> implements SerializationSchema<T> {
    private static final long serialVersionUID = 1L;

    private transient ObjectMapper objectMapper;

    @Override
    public byte[] serialize(T element) {
        if (objectMapper == null) {
            objectMapper = new ObjectMapper();
            objectMapper.configure(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS, true);
            objectMapper.configure(SerializationFeature.FAIL_ON_EMPTY_BEANS, false);
        }

        if (element == null) {
            return null;
        }

        try {
            return objectMapper.writeValueAsBytes(element);
        } catch (Exception e) {
            throw new RuntimeException("Failed to serialize object to JSON", e);
        }
    }
}
