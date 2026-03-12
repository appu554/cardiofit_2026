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
 * Critical Alert Router - Idempotent Consumer Job
 *
 * Reads from: prod.ehr.events.enriched.routing
 * Filters: routing.isSendToCriticalAlerts() == true
 * Writes to: prod.ehr.alerts.critical
 *
 * Idempotency:
 * - Uses event ID as Kafka message key
 * - Producer idempotence enabled
 * - Safe to reprocess without duplicates
 */
public class CriticalAlertRouter {
    private static final Logger LOG = LoggerFactory.getLogger(CriticalAlertRouter.class);

    public static void main(String[] args) throws Exception {
        LOG.info("🚀 Starting Critical Alert Router (Idempotent Consumer)");

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(2);
        env.enableCheckpointing(60000);

        // Source: Central routing topic
        KafkaSource<RoutedEnrichedEvent> source = KafkaSource.<RoutedEnrichedEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(KafkaTopics.EHR_EVENTS_ENRICHED_ROUTING.getTopicName())
            .setGroupId("critical-alert-router")
            .setStartingOffsets(OffsetsInitializer.committedOffsets())
            .setValueOnlyDeserializer(new RoutedEnrichedEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("critical-alert-router"))
            .build();

        DataStream<RoutedEnrichedEvent> routedEvents = env
            .fromSource(source, WatermarkStrategy.noWatermarks(), "Central Routing Source");

        // Filter for critical alerts
        DataStream<CriticalAlert> criticalAlerts = routedEvents
            .filter(event -> event.getRouting() != null && event.getRouting().isSendToCriticalAlerts())
            .map(event -> transformToCriticalAlert(event.getEvent()))
            .name("Transform to Critical Alert");

        // Idempotent sink
        criticalAlerts
            .sinkTo(createIdempotentCriticalAlertSink())
            .name("Critical Alert Sink (Idempotent)");

        LOG.info("✅ Critical Alert Router Ready");
        env.execute("Critical Alert Router");
    }

    private static CriticalAlert transformToCriticalAlert(EnrichedClinicalEvent event) {
        CriticalAlert alert = new CriticalAlert();
        alert.setId(event.getId());  // Use setId() not setEventId()
        alert.setPatientId(event.getPatientId());
        alert.setAlertType("CRITICAL");
        alert.setSeverity(event.getClinicalSignificance() > 0.9 ? "CRITICAL" : "HIGH");
        // Convert LocalDateTime to Long milliseconds
        alert.setTimestamp(event.getTimestamp().atZone(java.time.ZoneId.systemDefault()).toInstant().toEpochMilli());
        return alert;
    }

    private static KafkaSink<CriticalAlert> createIdempotentCriticalAlertSink() {
        Properties producerConfig = new Properties();
        producerConfig.put(ProducerConfig.ENABLE_IDEMPOTENCE_CONFIG, "true");
        producerConfig.put(ProducerConfig.ACKS_CONFIG, "all");
        producerConfig.put(ProducerConfig.MAX_IN_FLIGHT_REQUESTS_PER_CONNECTION, "5");
        producerConfig.putAll(KafkaConfigLoader.getProducerConfigForSink());

        return KafkaSink.<CriticalAlert>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.EHR_ALERTS_CRITICAL_ACTION.getTopicName())
                .setKeySerializationSchema((CriticalAlert alert) -> {
                    return alert.getId().getBytes();  // Use getId() not getEventId()
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
