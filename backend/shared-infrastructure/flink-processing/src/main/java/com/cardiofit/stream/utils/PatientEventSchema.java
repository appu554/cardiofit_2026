package com.cardiofit.stream.utils;

import com.cardiofit.stream.models.PatientEvent;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;

import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.connector.kafka.source.reader.deserializer.KafkaRecordDeserializationSchema;
import org.apache.flink.util.Collector;
import org.apache.kafka.clients.consumer.ConsumerRecord;

import java.io.IOException;

/**
 * Kafka Schema for Patient Event Serialization/Deserialization
 *
 * Implements both KafkaRecordDeserializationSchema for Kafka source consumption
 * and SerializationSchema for producing events to Kafka.
 */
public class PatientEventSchema implements
    KafkaRecordDeserializationSchema<PatientEvent>,
    SerializationSchema<PatientEvent> {

    private static final long serialVersionUID = 1L;
    private transient ObjectMapper objectMapper;

    /**
     * Initialize the ObjectMapper for JSON processing
     */
    private void initializeObjectMapper() {
        if (objectMapper == null) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }
    }

    @Override
    public void open(DeserializationSchema.InitializationContext context) throws Exception {
        initializeObjectMapper();
    }

    /**
     * Deserialize Kafka ConsumerRecord to PatientEvent
     * Required by KafkaRecordDeserializationSchema
     */
    @Override
    public void deserialize(ConsumerRecord<byte[], byte[]> record, Collector<PatientEvent> out) throws IOException {
        initializeObjectMapper();

        if (record.value() != null) {
            PatientEvent event = objectMapper.readValue(record.value(), PatientEvent.class);
            out.collect(event);
        }
    }

    /**
     * Get the TypeInformation for PatientEvent
     * Required by KafkaRecordDeserializationSchema
     */
    @Override
    public TypeInformation<PatientEvent> getProducedType() {
        return TypeInformation.of(PatientEvent.class);
    }

    /**
     * Serialize PatientEvent to byte array for producing to Kafka
     * Required by SerializationSchema
     */
    @Override
    public byte[] serialize(PatientEvent element) {
        try {
            initializeObjectMapper();
            return objectMapper.writeValueAsBytes(element);
        } catch (Exception e) {
            throw new RuntimeException("Failed to serialize PatientEvent", e);
        }
    }
}