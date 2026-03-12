package com.cardiofit.flink.routers;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.serialization.*;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.connector.base.DeliveryGuarantee;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.kafka.clients.producer.ProducerConfig;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Properties;

/**
 * FHIR Router - Idempotent Consumer Job
 *
 * Reads from: prod.ehr.events.enriched.routing
 * Filters: routing.isSendToFHIR() == true
 * Writes to: prod.ehr.fhir.upsert
 *
 * Idempotency:
 * - Uses event ID as Kafka message key
 * - Producer idempotence enabled
 * - Safe to reprocess without duplicates
 */
public class FHIRRouter {
    private static final Logger LOG = LoggerFactory.getLogger(FHIRRouter.class);

    public static void main(String[] args) throws Exception {
        LOG.info("🚀 Starting FHIR Router (Idempotent Consumer)");

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(2);
        env.enableCheckpointing(60000);

        // Source: Central routing topic
        KafkaSource<RoutedEnrichedEvent> source = KafkaSource.<RoutedEnrichedEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(KafkaTopics.EHR_EVENTS_ENRICHED_ROUTING.getTopicName())
            .setGroupId("fhir-router")
            .setStartingOffsets(OffsetsInitializer.committedOffsets())
            .setValueOnlyDeserializer(new RoutedEnrichedEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("fhir-router"))
            .build();

        DataStream<RoutedEnrichedEvent> routedEvents = env
            .fromSource(source, WatermarkStrategy.noWatermarks(), "Central Routing Source");

        // Filter for FHIR persistence
        DataStream<FHIRResource> fhirResources = routedEvents
            .filter(event -> event.getRouting() != null && event.getRouting().isSendToFHIR())
            .map(event -> transformToFHIRResource(event.getEvent()))
            .name("Transform to FHIR Resource");

        // Idempotent sink
        fhirResources
            .sinkTo(createIdempotentFHIRSink())
            .name("FHIR Sink (Idempotent)");

        LOG.info("✅ FHIR Router Ready");
        env.execute("FHIR Router");
    }

    private static FHIRResource transformToFHIRResource(EnrichedClinicalEvent event) {
        FHIRResource resource = new FHIRResource();
        resource.setResourceId(event.getId());
        resource.setResourceType(event.getFhirResourceType() != null ? event.getFhirResourceType() : "Observation");
        resource.setPatientId(event.getPatientId());
        // FHIRResource stores full data in fhirData field
        if (event.getFhirData() != null) {
            resource.setFhirData(event.getFhirData());  // Use setFhirData() not setProperties()
        }
        return resource;
    }

    private static KafkaSink<FHIRResource> createIdempotentFHIRSink() {
        Properties producerConfig = new Properties();
        producerConfig.put(ProducerConfig.ENABLE_IDEMPOTENCE_CONFIG, "true");
        producerConfig.put(ProducerConfig.ACKS_CONFIG, "all");
        producerConfig.put(ProducerConfig.MAX_IN_FLIGHT_REQUESTS_PER_CONNECTION, "5");
        producerConfig.putAll(KafkaConfigLoader.getProducerConfigForSink());

        return KafkaSink.<FHIRResource>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.EHR_FHIR_UPSERT.getTopicName())
                .setKeySerializationSchema((FHIRResource resource) -> {
                    return resource.getResourceId().getBytes();  // Resource ID as key
                })
                .setValueSerializationSchema(new com.cardiofit.flink.serialization.JsonSerializer<>())
                .build())
            .setKafkaProducerConfig(producerConfig)
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    private static String getBootstrapServers() {
        return KafkaConfigLoader.isRunningInDocker()
            ? "kafka1:29092,kafka2:29093,kafka3:29094"
            : "localhost:9092";
    }
}
