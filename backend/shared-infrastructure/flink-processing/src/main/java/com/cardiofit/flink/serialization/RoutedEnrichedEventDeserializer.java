package com.cardiofit.flink.serialization;

import com.cardiofit.flink.models.RoutedEnrichedEvent;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;

/**
 * JSON deserializer for RoutedEnrichedEvent.
 *
 * Used by idempotent router jobs to read events from prod.ehr.events.enriched.routing.
 * Each router filters based on RoutingDecision flags.
 */
public class RoutedEnrichedEventDeserializer implements DeserializationSchema<RoutedEnrichedEvent> {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(RoutedEnrichedEventDeserializer.class);

    private transient ObjectMapper objectMapper;

    @Override
    public RoutedEnrichedEvent deserialize(byte[] message) throws IOException {
        if (objectMapper == null) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        if (message == null || message.length == 0) {
            LOG.warn("Received empty message for RoutedEnrichedEvent deserialization");
            return null;
        }

        try {
            return objectMapper.readValue(message, RoutedEnrichedEvent.class);
        } catch (Exception e) {
            LOG.error("Failed to deserialize RoutedEnrichedEvent: message length={}", message.length, e);
            throw new IOException("Failed to deserialize RoutedEnrichedEvent", e);
        }
    }

    @Override
    public boolean isEndOfStream(RoutedEnrichedEvent nextElement) {
        return false;
    }

    @Override
    public TypeInformation<RoutedEnrichedEvent> getProducedType() {
        return TypeInformation.of(RoutedEnrichedEvent.class);
    }
}
