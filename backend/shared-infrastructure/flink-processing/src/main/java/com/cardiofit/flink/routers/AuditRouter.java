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

import java.util.HashMap;
import java.util.Map;
import java.util.Properties;

/**
 * Audit Router - Idempotent Consumer Job
 *
 * Reads from: prod.ehr.events.enriched.routing
 * Filters: routing.isSendToAudit() == true (ALL events for compliance)
 * Writes to: prod.ehr.audit.logs
 *
 * Idempotency:
 * - Uses event ID as Kafka message key
 * - Producer idempotence enabled
 * - Safe to reprocess without duplicates
 */
public class AuditRouter {
    private static final Logger LOG = LoggerFactory.getLogger(AuditRouter.class);

    public static void main(String[] args) throws Exception {
        LOG.info("🚀 Starting Audit Router (Idempotent Consumer)");

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(2);
        env.enableCheckpointing(60000);

        // Source: Central routing topic
        KafkaSource<RoutedEnrichedEvent> source = KafkaSource.<RoutedEnrichedEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(KafkaTopics.EHR_EVENTS_ENRICHED_ROUTING.getTopicName())
            .setGroupId("audit-router")
            .setStartingOffsets(OffsetsInitializer.committedOffsets())
            .setValueOnlyDeserializer(new RoutedEnrichedEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("audit-router"))
            .build();

        DataStream<RoutedEnrichedEvent> routedEvents = env
            .fromSource(source, WatermarkStrategy.noWatermarks(), "Central Routing Source");

        // Filter for audit (typically ALL events)
        DataStream<AuditLogEntry> auditLogs = routedEvents
            .filter(event -> event.getRouting() != null && event.getRouting().isSendToAudit())
            .map(event -> transformToAuditLog(event))
            .name("Transform to Audit Log");

        // Idempotent sink
        auditLogs
            .sinkTo(createIdempotentAuditSink())
            .name("Audit Log Sink (Idempotent)");

        LOG.info("✅ Audit Router Ready");
        env.execute("Audit Router");
    }

    private static AuditLogEntry transformToAuditLog(RoutedEnrichedEvent routedEvent) {
        EnrichedClinicalEvent event = routedEvent.getEvent();
        AuditLogEntry auditLog = new AuditLogEntry();

        auditLog.setEventId(event.getId());
        auditLog.setPatientId(event.getPatientId());
        auditLog.setEventType(event.getSourceEventType());
        // Convert LocalDateTime to Long milliseconds
        auditLog.setTimestamp(event.getTimestamp().atZone(java.time.ZoneId.systemDefault()).toInstant().toEpochMilli());

        // Store routing metadata in details map
        Map<String, Object> details = new HashMap<>();
        details.put("routing_id", routedEvent.getRoutingId());
        details.put("destination_count", routedEvent.getRouting().getDestinationCount());
        details.put("routing_source", "Module6_OptionC");
        details.put("routing_timestamp", routedEvent.getRoutingTimestamp());
        auditLog.setDetails(details);

        // Audit metadata
        auditLog.setAction("ROUTED");
        auditLog.setSource("Module6_OptionC");

        return auditLog;
    }

    private static KafkaSink<AuditLogEntry> createIdempotentAuditSink() {
        Properties producerConfig = new Properties();
        producerConfig.put(ProducerConfig.ENABLE_IDEMPOTENCE_CONFIG, "true");
        producerConfig.put(ProducerConfig.ACKS_CONFIG, "all");
        producerConfig.put(ProducerConfig.MAX_IN_FLIGHT_REQUESTS_PER_CONNECTION, "5");
        producerConfig.putAll(KafkaConfigLoader.getProducerConfigForSink());

        return KafkaSink.<AuditLogEntry>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.EHR_AUDIT_LOGS.getTopicName())
                .setKeySerializationSchema((AuditLogEntry auditLog) -> {
                    return auditLog.getEventId().getBytes();  // Event ID as key
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
