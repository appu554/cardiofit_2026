package com.cardiofit.stream.utils;

import com.cardiofit.stream.models.PatientEvent;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;

import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;

import java.io.IOException;

/**
 * Kafka Schema for Patient Event Serialization/Deserialization
 */
public class PatientEventSchema implements DeserializationSchema<PatientEvent>, SerializationSchema<PatientEvent> {

    private static final long serialVersionUID = 1L;
    private transient ObjectMapper objectMapper;

    @Override
    public void open(InitializationContext context) throws Exception {
        objectMapper = new ObjectMapper();
        objectMapper.registerModule(new JavaTimeModule());
    }

    @Override
    public PatientEvent deserialize(byte[] message) throws IOException {
        if (objectMapper == null) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }
        return objectMapper.readValue(message, PatientEvent.class);
    }

    @Override
    public boolean isEndOfStream(PatientEvent nextElement) {
        return false;
    }

    @Override
    public TypeInformation<PatientEvent> getProducedType() {
        return TypeInformation.of(PatientEvent.class);
    }

    @Override
    public byte[] serialize(PatientEvent element) {
        try {
            if (objectMapper == null) {
                objectMapper = new ObjectMapper();
                objectMapper.registerModule(new JavaTimeModule());
            }
            return objectMapper.writeValueAsBytes(element);
        } catch (Exception e) {
            throw new RuntimeException("Failed to serialize PatientEvent", e);
        }
    }
}