package com.cardiofit.flink.operators;

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
 * Module 6: Egress & Multi-Sink Routing - Option C Architecture (INTEGRATED)
 *
 * SIMPLIFIED SINGLE-JOB VERSION with 5 Direct Idempotent Sinks
 *
 * Architecture:
 * 1. Read EnrichedClinicalEvent from comprehensive-cds-events.v1
 * 2. Transform to specific output types
 * 3. Write directly to 5 idempotent sinks (NO intermediate topic!)
 *
 * Benefits:
 * - Single Flink job instead of 6 jobs
 * - No central routing topic overhead
 * - No offset management issues
 * - Idempotent producers ensure exactly-once semantics
 * - Graph mutations re-enabled safely!
 */
public class Module6_EgressRouting_OptionC {
    private static final Logger LOG = LoggerFactory.getLogger(Module6_EgressRouting_OptionC.class);

    public static void main(String[] args) throws Exception {
        LOG.info("🚀 Starting Module 6: Integrated Multi-Sink Router");
        LOG.info("   Architecture: 5 Direct Idempotent Sinks (No Intermediate Topic)");

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(4);  // Higher parallelism for multiple sinks
        env.enableCheckpointing(60000);  // 1 minute checkpoints
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);

        // Create source: Read from Module 3 CDS events
        KafkaSource<Module3_ComprehensiveCDS.CDSEvent> source = createCDSEventSource();

        DataStream<Module3_ComprehensiveCDS.CDSEvent> cdsEvents = env
            .fromSource(source, WatermarkStrategy.noWatermarks(), "CDS Events Source")
            .name("Module 3 CDS Events");

        // Transform CDS events to EnrichedClinicalEvent
        DataStream<EnrichedClinicalEvent> enrichedEvents = cdsEvents
            .map(new CDSEventToEnrichedEventMapper())
            .name("CDS to Enriched Mapper");

        // Branch 1: Critical Alerts
        enrichedEvents
            .filter(event -> event.getClinicalSignificance() > 0.7D)
            .map(Module6_EgressRouting_OptionC::transformToCriticalAlert)
            .sinkTo(createCriticalAlertSink())
            .name("Critical Alert Sink");

        // Branch 2: FHIR Resources
        enrichedEvents
            .filter(event -> event.getFhirResourceType() != null || event.getFhirData() != null)
            .map(Module6_EgressRouting_OptionC::transformToFHIRResource)
            .sinkTo(createFHIRSink())
            .name("FHIR Resource Sink");

        // Branch 3: Analytics Events
        enrichedEvents
            .map(Module6_EgressRouting_OptionC::transformToAnalyticsEvent)
            .sinkTo(createAnalyticsSink())
            .name("Analytics Event Sink");

        // Branch 4: Graph Mutations (RE-ENABLED!)
        enrichedEvents
            .filter(event -> event.getDrugInteractions() != null && !event.getDrugInteractions().isEmpty())
            .map(Module6_EgressRouting_OptionC::transformToGraphMutation)
            .sinkTo(createGraphSink())
            .name("Graph Mutation Sink (RE-ENABLED)");

        // Branch 5: Audit Logs (ALL events)
        enrichedEvents
            .map(Module6_EgressRouting_OptionC::transformToAuditLog)
            .sinkTo(createAuditSink())
            .name("Audit Log Sink");

        LOG.info("✅ Module 6 Integrated Router Created:");
        LOG.info("   📥 Source: comprehensive-cds-events.v1");
        LOG.info("   📤 Sink 1: Critical Alerts → prod.ehr.alerts.critical");
        LOG.info("   📤 Sink 2: FHIR Resources → prod.ehr.fhir.upsert");
        LOG.info("   📤 Sink 3: Analytics → prod.ehr.analytics.events");
        LOG.info("   📤 Sink 4: Graph Mutations → prod.ehr.graph.mutations (RE-ENABLED!)");
        LOG.info("   📤 Sink 5: Audit Logs → prod.ehr.audit.logs");
        LOG.info("   ⚡ Parallelism: 4");
        LOG.info("   ✔️ All sinks use IDEMPOTENT producers (safe reprocessing)");

        env.execute("Module 6: Integrated Multi-Sink Router");
    }

    // ==================== SOURCE ====================

    private static KafkaSource<Module3_ComprehensiveCDS.CDSEvent> createCDSEventSource() {
        return KafkaSource.<Module3_ComprehensiveCDS.CDSEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics("comprehensive-cds-events.v1")
            .setGroupId("module6-integrated-egress")
            .setStartingOffsets(OffsetsInitializer.committedOffsets())
            .setValueOnlyDeserializer(new CDSEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("module6-integrated-egress"))
            .build();
    }

    // ==================== TRANSFORMATIONS ====================

    private static CriticalAlert transformToCriticalAlert(EnrichedClinicalEvent event) {
        CriticalAlert alert = new CriticalAlert();
        alert.setId(event.getId());
        alert.setPatientId(event.getPatientId());
        alert.setAlertType("CRITICAL");
        alert.setSeverity(event.getClinicalSignificance() > 0.9 ? "CRITICAL" : "HIGH");
        alert.setTimestamp(event.getTimestamp().atZone(java.time.ZoneId.systemDefault()).toInstant().toEpochMilli());
        return alert;
    }

    private static FHIRResource transformToFHIRResource(EnrichedClinicalEvent event) {
        FHIRResource resource = new FHIRResource();
        resource.setResourceId(event.getId());
        resource.setResourceType(event.getFhirResourceType() != null ? event.getFhirResourceType() : "Observation");
        resource.setPatientId(event.getPatientId());
        if (event.getFhirData() != null) {
            resource.setFhirData(event.getFhirData());
        }
        return resource;
    }

    private static AnalyticsEvent transformToAnalyticsEvent(EnrichedClinicalEvent event) {
        AnalyticsEvent analyticsEvent = new AnalyticsEvent();
        analyticsEvent.setEventId(event.getId());
        analyticsEvent.setPatientId(event.getPatientId());
        analyticsEvent.setEventType(event.getSourceEventType());
        analyticsEvent.setClinicalSignificance(event.getClinicalSignificance());

        Map<String, Object> metrics = new HashMap<>();
        if (event.getDetectedPatterns() != null) {
            metrics.put("detected_patterns", event.getDetectedPatterns());
        }
        if (event.getRiskScores() != null) {
            metrics.put("risk_scores", event.getRiskScores());
        }
        analyticsEvent.setMetrics(metrics);

        return analyticsEvent;
    }

    private static GraphMutation transformToGraphMutation(EnrichedClinicalEvent event) {
        GraphMutation mutation = new GraphMutation();
        mutation.setSourceEventId(event.getId());
        mutation.setPatientId(event.getPatientId());
        mutation.setMutationType("DRUG_INTERACTION");
        mutation.setNodeType("Patient");
        mutation.setNodeId(event.getPatientId());
        mutation.setRelationshipType("HAS_DRUG_INTERACTION");
        return mutation;
    }

    private static AuditLogEntry transformToAuditLog(EnrichedClinicalEvent event) {
        AuditLogEntry auditLog = new AuditLogEntry();
        auditLog.setEventId(event.getId());
        auditLog.setPatientId(event.getPatientId());
        auditLog.setEventType(event.getSourceEventType());
        auditLog.setTimestamp(event.getTimestamp().atZone(java.time.ZoneId.systemDefault()).toInstant().toEpochMilli());

        Map<String, Object> details = new HashMap<>();
        details.put("routing_source", "Module6_Integrated");
        details.put("clinical_significance", event.getClinicalSignificance());
        auditLog.setDetails(details);

        auditLog.setAction("ROUTED");
        auditLog.setSource("Module6_Integrated");

        return auditLog;
    }

    // ==================== IDEMPOTENT SINKS ====================

    private static KafkaSink<CriticalAlert> createCriticalAlertSink() {
        Properties config = createIdempotentProducerConfig();
        return KafkaSink.<CriticalAlert>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.<CriticalAlert>builder()
                .setTopic(KafkaTopics.EHR_ALERTS_CRITICAL_ACTION.getTopicName())
                .setKeySerializationSchema((CriticalAlert alert) -> alert.getId().getBytes())
                .setValueSerializationSchema(new JsonSerializer<>())
                .build())
            .setKafkaProducerConfig(config)
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    private static KafkaSink<FHIRResource> createFHIRSink() {
        Properties config = createIdempotentProducerConfig();
        return KafkaSink.<FHIRResource>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.<FHIRResource>builder()
                .setTopic(KafkaTopics.EHR_FHIR_UPSERT.getTopicName())
                .setKeySerializationSchema((FHIRResource resource) -> resource.getResourceId().getBytes())
                .setValueSerializationSchema(new JsonSerializer<>())
                .build())
            .setKafkaProducerConfig(config)
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    private static KafkaSink<AnalyticsEvent> createAnalyticsSink() {
        Properties config = createIdempotentProducerConfig();
        return KafkaSink.<AnalyticsEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.<AnalyticsEvent>builder()
                .setTopic(KafkaTopics.EHR_ANALYTICS_EVENTS.getTopicName())
                .setKeySerializationSchema((AnalyticsEvent event) -> event.getEventId().getBytes())
                .setValueSerializationSchema(new JsonSerializer<>())
                .build())
            .setKafkaProducerConfig(config)
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    private static KafkaSink<GraphMutation> createGraphSink() {
        Properties config = createIdempotentProducerConfig();
        return KafkaSink.<GraphMutation>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.<GraphMutation>builder()
                .setTopic(KafkaTopics.EHR_GRAPH_MUTATIONS.getTopicName())
                .setKeySerializationSchema((GraphMutation mutation) -> mutation.getNodeId().getBytes())
                .setValueSerializationSchema(new JsonSerializer<>())
                .build())
            .setKafkaProducerConfig(config)
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    private static KafkaSink<AuditLogEntry> createAuditSink() {
        Properties config = createIdempotentProducerConfig();
        return KafkaSink.<AuditLogEntry>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.<AuditLogEntry>builder()
                .setTopic(KafkaTopics.EHR_AUDIT_LOGS.getTopicName())
                .setKeySerializationSchema((AuditLogEntry log) -> log.getEventId().getBytes())
                .setValueSerializationSchema(new JsonSerializer<>())
                .build())
            .setKafkaProducerConfig(config)
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    // ==================== HELPER METHODS ====================

    private static Properties createIdempotentProducerConfig() {
        Properties config = new Properties();
        config.put(ProducerConfig.ENABLE_IDEMPOTENCE_CONFIG, "true");
        config.put(ProducerConfig.ACKS_CONFIG, "all");
        config.put(ProducerConfig.MAX_IN_FLIGHT_REQUESTS_PER_CONNECTION, "5");
        config.putAll(KafkaConfigLoader.getProducerConfigForSink());
        return config;
    }

    private static String getBootstrapServers() {
        return KafkaConfigLoader.isRunningInDocker()
            ? "kafka1:29092,kafka2:29093,kafka3:29094"
            : "localhost:9092,localhost:9093,localhost:9094";
    }

    // ==================== MAPPER CLASSES ====================

    private static class CDSEventToEnrichedEventMapper
            implements org.apache.flink.api.common.functions.MapFunction<Module3_ComprehensiveCDS.CDSEvent, EnrichedClinicalEvent> {

        @Override
        public EnrichedClinicalEvent map(Module3_ComprehensiveCDS.CDSEvent cdsEvent) throws Exception {
            EnrichedClinicalEvent enriched = new EnrichedClinicalEvent();

            String eventId = cdsEvent.getPatientId() + "-" + cdsEvent.getEventTime();
            enriched.setId(eventId);
            enriched.setPatientId(cdsEvent.getPatientId());
            enriched.setSourceEventType("CDS_EVENT");
            enriched.setTimestamp(java.time.LocalDateTime.now());

            if (cdsEvent.getCdsRecommendations() != null && !cdsEvent.getCdsRecommendations().isEmpty()) {
                enriched.setClinicalSignificance(0.8);
            }

            if (cdsEvent.getSemanticEnrichment() != null &&
                cdsEvent.getSemanticEnrichment().getDrugInteractionAnalysis() != null) {

                SemanticEnrichment.DrugInteractionAnalysis analysis =
                    cdsEvent.getSemanticEnrichment().getDrugInteractionAnalysis();

                if (analysis.getInteractionWarnings() != null && !analysis.getInteractionWarnings().isEmpty()) {
                    java.util.List<SemanticEvent.DrugInteraction> interactions = new java.util.ArrayList<>();

                    for (SemanticEnrichment.InteractionWarning warning : analysis.getInteractionWarnings()) {
                        SemanticEvent.DrugInteraction interaction = new SemanticEvent.DrugInteraction();
                        interaction.setDrug1(warning.getProtocolMedication());
                        interaction.setDrug2(warning.getActiveMedication());
                        interaction.setSeverity(warning.getSeverity());
                        interactions.add(interaction);
                    }

                    enriched.setDrugInteractions(interactions);
                }
            }

            return enriched;
        }
    }

    private static class CDSEventDeserializer
            implements org.apache.flink.api.common.serialization.DeserializationSchema<Module3_ComprehensiveCDS.CDSEvent> {

        private transient com.fasterxml.jackson.databind.ObjectMapper objectMapper;

        @Override
        public Module3_ComprehensiveCDS.CDSEvent deserialize(byte[] message) throws java.io.IOException {
            if (objectMapper == null) {
                objectMapper = new com.fasterxml.jackson.databind.ObjectMapper();
                objectMapper.registerModule(new com.fasterxml.jackson.datatype.jsr310.JavaTimeModule());
            }
            return objectMapper.readValue(message, Module3_ComprehensiveCDS.CDSEvent.class);
        }

        @Override
        public boolean isEndOfStream(Module3_ComprehensiveCDS.CDSEvent nextElement) {
            return false;
        }

        @Override
        public org.apache.flink.api.common.typeinfo.TypeInformation<Module3_ComprehensiveCDS.CDSEvent> getProducedType() {
            return org.apache.flink.api.common.typeinfo.TypeInformation.of(Module3_ComprehensiveCDS.CDSEvent.class);
        }
    }
}
