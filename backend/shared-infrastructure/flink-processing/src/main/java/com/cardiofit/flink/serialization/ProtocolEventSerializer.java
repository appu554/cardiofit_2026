package com.cardiofit.flink.serialization;

import com.cardiofit.flink.models.ProtocolEvent;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Kafka serializer for ProtocolEvent objects
 * Phase 4 Enhancement: Serializes protocol trigger events to JSON for Kafka
 */
public class ProtocolEventSerializer implements SerializationSchema<ProtocolEvent> {
    private static final Logger LOG = LoggerFactory.getLogger(ProtocolEventSerializer.class);
    private static final ObjectMapper objectMapper = new ObjectMapper()
        .configure(SerializationFeature.FAIL_ON_EMPTY_BEANS, false);

    @Override
    public byte[] serialize(ProtocolEvent event) {
        try {
            return objectMapper.writeValueAsBytes(event);
        } catch (Exception e) {
            LOG.error("Failed to serialize ProtocolEvent: {}", event, e);
            return new byte[0];
        }
    }
}
