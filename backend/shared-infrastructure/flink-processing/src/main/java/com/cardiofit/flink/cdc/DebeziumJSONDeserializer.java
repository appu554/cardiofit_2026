package com.cardiofit.flink.cdc;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.DeserializationFeature;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;

/**
 * Base Debezium JSON Deserializer for CDC Events
 *
 * Provides common deserialization logic for all Debezium CDC events
 * using Jackson ObjectMapper with proper configuration for PostgreSQL CDC.
 *
 * @param <T> CDC Event type (ProtocolCDCEvent, DrugRuleCDCEvent, etc.)
 * @author Phase 2 CDC Integration Team
 * @version 1.0
 * @since 2025-11-22
 */
public class DebeziumJSONDeserializer<T> implements DeserializationSchema<T> {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(DebeziumJSONDeserializer.class);

    private final Class<T> clazz;
    private final ObjectMapper objectMapper;

    /**
     * Constructor with CDC event class
     *
     * @param clazz The CDC event class to deserialize to
     */
    public DebeziumJSONDeserializer(Class<T> clazz) {
        this.clazz = clazz;
        this.objectMapper = new ObjectMapper();

        // Configure ObjectMapper for Debezium CDC events
        objectMapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
        objectMapper.configure(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES, false);
        objectMapper.configure(DeserializationFeature.ACCEPT_EMPTY_STRING_AS_NULL_OBJECT, true);
    }

    @Override
    public T deserialize(byte[] message) throws IOException {
        if (message == null || message.length == 0) {
            LOG.warn("Received null or empty message");
            return null;
        }

        try {
            T event = objectMapper.readValue(message, clazz);

            // Validate Debezium envelope structure
            if (event instanceof ProtocolCDCEvent) {
                ProtocolCDCEvent cdcEvent = (ProtocolCDCEvent) event;
                if (cdcEvent.getPayload() == null) {
                    LOG.warn("CDC event missing payload: {}", new String(message));
                    return null;
                }
            }

            return event;
        } catch (IOException e) {
            LOG.error("Failed to deserialize CDC event: {}", e.getMessage());
            LOG.debug("Problematic message: {}", new String(message));
            throw e;
        }
    }

    @Override
    public boolean isEndOfStream(T nextElement) {
        return false; // CDC streams are infinite
    }

    @Override
    public TypeInformation<T> getProducedType() {
        return TypeInformation.of(clazz);
    }

    /**
     * Factory method for Protocol CDC events
     */
    public static DebeziumJSONDeserializer<ProtocolCDCEvent> forProtocol() {
        return new DebeziumJSONDeserializer<>(ProtocolCDCEvent.class);
    }

    /**
     * Factory method for Clinical Phenotype CDC events
     */
    public static DebeziumJSONDeserializer<ClinicalPhenotypeCDCEvent> forPhenotype() {
        return new DebeziumJSONDeserializer<>(ClinicalPhenotypeCDCEvent.class);
    }

    /**
     * Factory method for Drug Rule CDC events
     */
    public static DebeziumJSONDeserializer<DrugRuleCDCEvent> forDrugRule() {
        return new DebeziumJSONDeserializer<>(DrugRuleCDCEvent.class);
    }

    /**
     * Factory method for Drug Interaction CDC events
     */
    public static DebeziumJSONDeserializer<DrugInteractionCDCEvent> forDrugInteraction() {
        return new DebeziumJSONDeserializer<>(DrugInteractionCDCEvent.class);
    }

    /**
     * Factory method for Formulary Drug CDC events
     */
    public static DebeziumJSONDeserializer<FormularyDrugCDCEvent> forFormulary() {
        return new DebeziumJSONDeserializer<>(FormularyDrugCDCEvent.class);
    }

    /**
     * Factory method for Terminology CDC events
     */
    public static DebeziumJSONDeserializer<TerminologyCDCEvent> forTerminology() {
        return new DebeziumJSONDeserializer<>(TerminologyCDCEvent.class);
    }

    /**
     * Factory method for Terminology Release CDC events (kb_releases outbox table)
     *
     * Used for KB-7 terminology version notifications from the Knowledge Factory pipeline.
     * Consumes from: kb7.terminology.releases topic
     */
    public static DebeziumJSONDeserializer<TerminologyReleaseCDCEvent> forTerminologyRelease() {
        return new DebeziumJSONDeserializer<>(TerminologyReleaseCDCEvent.class);
    }
}
