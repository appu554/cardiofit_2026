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
 * Analytics Router - Idempotent Consumer Job
 *
 * Reads from: prod.ehr.events.enriched.routing
 * Filters: routing.isSendToAnalytics() == true
 * Writes to: prod.ehr.analytics.events
 *
 * Idempotency:
 * - Uses event ID as Kafka message key
 * - Producer idempotence enabled
 * - Safe to reprocess without duplicates
 */
public class AnalyticsRouter {
    private static final Logger LOG = LoggerFactory.getLogger(AnalyticsRouter.class);

    public static void main(String[] args) throws Exception {
        LOG.info("🚀 Starting Analytics Router (Idempotent Consumer)");

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(4);  // Higher parallelism for analytics workload
        env.enableCheckpointing(60000);

        // Source: Central routing topic
        KafkaSource<RoutedEnrichedEvent> source = KafkaSource.<RoutedEnrichedEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(KafkaTopics.EHR_EVENTS_ENRICHED_ROUTING.getTopicName())
            .setGroupId("analytics-router")
            .setStartingOffsets(OffsetsInitializer.committedOffsets())
            .setValueOnlyDeserializer(new RoutedEnrichedEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("analytics-router"))
            .build();

        DataStream<RoutedEnrichedEvent> routedEvents = env
            .fromSource(source, WatermarkStrategy.noWatermarks(), "Central Routing Source");

        // Filter for analytics
        DataStream<AnalyticsEvent> analyticsEvents = routedEvents
            .filter(event -> event.getRouting() != null && event.getRouting().isSendToAnalytics())
            .map(event -> transformToAnalyticsEvent(event.getEvent()))
            .name("Transform to Analytics Event");

        // Idempotent sink
        analyticsEvents
            .sinkTo(createIdempotentAnalyticsSink())
            .name("Analytics Sink (Idempotent)");

        LOG.info("✅ Analytics Router Ready");
        env.execute("Analytics Router");
    }

    private static AnalyticsEvent transformToAnalyticsEvent(EnrichedClinicalEvent event) {
        AnalyticsEvent analyticsEvent = new AnalyticsEvent();
        analyticsEvent.setEventId(event.getId());
        analyticsEvent.setPatientId(event.getPatientId());
        analyticsEvent.setEventType(event.getSourceEventType());
        analyticsEvent.setClinicalSignificance(event.getClinicalSignificance());

        // Store patterns and risk scores in metrics map
        java.util.Map<String, Object> metrics = new java.util.HashMap<>();
        if (event.getDetectedPatterns() != null) {
            metrics.put("detected_patterns", event.getDetectedPatterns());
        }
        if (event.getRiskScores() != null) {
            metrics.put("risk_scores", event.getRiskScores());
        }
        analyticsEvent.setMetrics(metrics);

        return analyticsEvent;
    }

    private static KafkaSink<AnalyticsEvent> createIdempotentAnalyticsSink() {
        Properties producerConfig = new Properties();
        producerConfig.put(ProducerConfig.ENABLE_IDEMPOTENCE_CONFIG, "true");
        producerConfig.put(ProducerConfig.ACKS_CONFIG, "all");
        producerConfig.put(ProducerConfig.MAX_IN_FLIGHT_REQUESTS_PER_CONNECTION, "5");
        producerConfig.putAll(KafkaConfigLoader.getProducerConfigForSink());

        return KafkaSink.<AnalyticsEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.EHR_ANALYTICS_EVENTS.getTopicName())
                .setKeySerializationSchema((AnalyticsEvent analyticsEvent) -> {
                    return analyticsEvent.getEventId().getBytes();  // Event ID as key
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
