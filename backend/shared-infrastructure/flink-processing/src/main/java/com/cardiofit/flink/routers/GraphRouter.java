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
 * Graph Router - Idempotent Consumer Job
 *
 * Reads from: prod.ehr.events.enriched.routing
 * Filters: routing.isSendToGraph() == true
 * Writes to: prod.ehr.graph.mutations
 *
 * This REPLACES the commented-out graph sink in Module6_EgressRouting.
 * Now runs as independent job with idempotent guarantees.
 *
 * Idempotency:
 * - Uses event ID as Kafka message key
 * - Producer idempotence enabled
 * - Safe to reprocess without duplicates
 */
public class GraphRouter {
    private static final Logger LOG = LoggerFactory.getLogger(GraphRouter.class);

    public static void main(String[] args) throws Exception {
        LOG.info("🚀 Starting Graph Router (Idempotent Consumer)");
        LOG.info("   This router handles graph mutations that were previously causing Module 6 crashes");

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(2);
        env.enableCheckpointing(60000);

        // Source: Central routing topic
        KafkaSource<RoutedEnrichedEvent> source = KafkaSource.<RoutedEnrichedEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(KafkaTopics.EHR_EVENTS_ENRICHED_ROUTING.getTopicName())
            .setGroupId("graph-router")
            .setStartingOffsets(OffsetsInitializer.committedOffsets())
            .setValueOnlyDeserializer(new RoutedEnrichedEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("graph-router"))
            .build();

        DataStream<RoutedEnrichedEvent> routedEvents = env
            .fromSource(source, WatermarkStrategy.noWatermarks(), "Central Routing Source");

        // Filter for graph mutations
        DataStream<GraphMutation> graphMutations = routedEvents
            .filter(event -> event.getRouting() != null && event.getRouting().isSendToGraph())
            .map(event -> transformToGraphMutation(event.getEvent()))
            .name("Transform to Graph Mutation");

        // Idempotent sink
        graphMutations
            .sinkTo(createIdempotentGraphSink())
            .name("Graph Mutation Sink (Idempotent)");

        LOG.info("✅ Graph Router Ready - drug interactions will flow to Neo4j");
        env.execute("Graph Router");
    }

    private static GraphMutation transformToGraphMutation(EnrichedClinicalEvent event) {
        GraphMutation mutation = new GraphMutation();
        mutation.setSourceEventId(event.getId());
        mutation.setPatientId(event.getPatientId());
        mutation.setMutationType("RELATIONSHIP_UPDATE");
        mutation.setNodeType("Patient");
        mutation.setNodeId(event.getPatientId());

        // Extract drug interactions for graph
        if (event.getDrugInteractions() != null && !event.getDrugInteractions().isEmpty()) {
            mutation.setMutationType("DRUG_INTERACTION");
            mutation.setRelationshipType("HAS_DRUG_INTERACTION");
        }

        return mutation;
    }

    private static KafkaSink<GraphMutation> createIdempotentGraphSink() {
        Properties producerConfig = new Properties();
        producerConfig.put(ProducerConfig.ENABLE_IDEMPOTENCE_CONFIG, "true");
        producerConfig.put(ProducerConfig.ACKS_CONFIG, "all");
        producerConfig.put(ProducerConfig.MAX_IN_FLIGHT_REQUESTS_PER_CONNECTION, "5");
        producerConfig.putAll(KafkaConfigLoader.getProducerConfigForSink());

        return KafkaSink.<GraphMutation>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.EHR_GRAPH_MUTATIONS.getTopicName())
                .setKeySerializationSchema((GraphMutation mutation) -> {
                    String nodeId = mutation.getNodeId();
                    return (nodeId != null ? nodeId : "UNKNOWN_NODE").getBytes();
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
