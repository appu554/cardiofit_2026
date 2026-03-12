package com.cardiofit.flink.serialization;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.DeserializationFeature;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;

import java.io.IOException;

/**
 * Custom JSON deserializer for Kafka sources
 * Uses Jackson ObjectMapper for JSON deserialization
 *
 * @param <T> Type to deserialize into
 */
public class JsonDeserializer<T> implements DeserializationSchema<T> {
    private static final long serialVersionUID = 1L;

    private final Class<T> clazz;
    private transient ObjectMapper objectMapper;

    public JsonDeserializer(Class<T> clazz) {
        this.clazz = clazz;
    }

    @Override
    public T deserialize(byte[] message) throws IOException {
        if (objectMapper == null) {
            objectMapper = new ObjectMapper();
            objectMapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
            objectMapper.configure(DeserializationFeature.ACCEPT_EMPTY_STRING_AS_NULL_OBJECT, true);
        }

        if (message == null || message.length == 0) {
            return null;
        }

        return objectMapper.readValue(message, clazz);
    }

    @Override
    public boolean isEndOfStream(T nextElement) {
        return false;
    }

    @Override
    public TypeInformation<T> getProducedType() {
        return TypeInformation.of(clazz);
    }
}
