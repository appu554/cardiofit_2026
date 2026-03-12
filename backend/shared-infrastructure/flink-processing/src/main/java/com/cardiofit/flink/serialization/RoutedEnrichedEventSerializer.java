package com.cardiofit.flink.serialization;

import com.cardiofit.flink.models.RoutedEnrichedEvent;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * JSON serializer for RoutedEnrichedEvent.
 *
 * Used by the central routing sink to write events to prod.ehr.events.enriched.routing.
 * Includes proper LocalDateTime handling via JavaTimeModule.
 */
public class RoutedEnrichedEventSerializer implements SerializationSchema<RoutedEnrichedEvent> {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(RoutedEnrichedEventSerializer.class);

    private transient ObjectMapper objectMapper;

    @Override
    public byte[] serialize(RoutedEnrichedEvent element) {
        if (objectMapper == null) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
            objectMapper.configure(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS, false);
            objectMapper.configure(SerializationFeature.FAIL_ON_EMPTY_BEANS, false);
        }

        if (element == null) {
            LOG.warn("Attempting to serialize null RoutedEnrichedEvent");
            return null;
        }

        try {
            return objectMapper.writeValueAsBytes(element);
        } catch (Exception e) {
            LOG.error("Failed to serialize RoutedEnrichedEvent: eventId={}, patientId={}",
                element.getEventId(), element.getPatientId(), e);
            throw new RuntimeException("Failed to serialize RoutedEnrichedEvent", e);
        }
    }
}
