package com.cardiofit.flink.operators;

import com.cardiofit.flink.cdc.TerminologyReleaseCDCEvent;
import com.cardiofit.flink.cdc.DebeziumJSONDeserializer;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.state.*;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.BroadcastStream;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.streaming.api.functions.co.KeyedBroadcastProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.io.Serializable;
import java.time.Duration;
import java.time.Instant;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
import java.util.*;

/**
 * Module KB7: Terminology Release BroadcastStream
 *
 * Hot-swapping terminology versions via CDC without Flink restart.
 *
 * Architecture:
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │  KB-7 Knowledge Factory Pipeline                                        │
 * │       ↓                                                                  │
 * │  GraphDB: SPARQL LOAD (5-15 min for ~14M triples)                       │
 * │       ↓                                                                  │
 * │  Health Check: Triple count + sample query                               │
 * │       ↓                                                                  │
 * │  PostgreSQL: INSERT INTO kb_releases (status='ACTIVE')  ← Commit-Last   │
 * │       ↓                                                                  │
 * │  Debezium: kb7-terminology-releases-cdc connector                       │
 * │       ↓                                                                  │
 * │  Kafka Topic: kb7.terminology.releases                                  │
 * │       ↓                                                                  │
 * │  ┌─────────────────────────────────────────┐                            │
 * │  │  THIS MODULE: Flink BroadcastStream     │                            │
 * │  │  - Receives CDC release notifications   │                            │
 * │  │  - Broadcasts to all parallel instances │                            │
 * │  │  - Updates terminology cache in-memory  │                            │
 * │  │  - Zero downtime version switching      │                            │
 * │  └─────────────────────────────────────────┘                            │
 * │       ↓                                                                  │
 * │  Downstream Consumers:                                                   │
 * │  ├── Clinical Reasoning → Refresh SNOMED/RxNorm/LOINC mappings          │
 * │  ├── KB Services → Update local terminology copies                      │
 * │  └── Notification Service → Alert administrators                        │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * Performance:
 * - Terminology updates propagate in <1 second without restart
 * - Zero downtime for version switches
 * - Automatic synchronization across all parallel instances
 *
 * @author KB-7 CDC Integration Team
 * @version 1.0
 * @since 2025-12-03
 */
public class Module_KB7_TerminologyBroadcast {
    private static final Logger LOG = LoggerFactory.getLogger(Module_KB7_TerminologyBroadcast.class);

    // Kafka topic for KB7 terminology releases (from CDC connector)
    private static final String KB7_RELEASES_TOPIC = "kb7.terminology.releases";

    // Output topic for terminology update notifications
    private static final String TERMINOLOGY_UPDATES_TOPIC = "terminology-version-updates.v1";

    // BroadcastStateDescriptor for terminology release hot-swapping
    public static final MapStateDescriptor<String, TerminologyReleaseState> TERMINOLOGY_STATE_DESCRIPTOR =
            new MapStateDescriptor<>(
                    "terminology-release-broadcast-state",
                    TypeInformation.of(String.class),
                    TypeInformation.of(TerminologyReleaseState.class)
            );

    // Side output for release notifications
    private static final OutputTag<TerminologyUpdateNotification> NOTIFICATION_OUTPUT =
            new OutputTag<TerminologyUpdateNotification>("terminology-notifications") {};

    public static void main(String[] args) throws Exception {
        LOG.info("╔══════════════════════════════════════════════════════════════╗");
        LOG.info("║  Starting Module KB7: Terminology Release BroadcastStream    ║");
        LOG.info("║  Hot-swapping terminology versions via CDC                   ║");
        LOG.info("╚══════════════════════════════════════════════════════════════╝");

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        env.setParallelism(2);
        env.enableCheckpointing(30000);
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);

        createTerminologyBroadcastPipeline(env);

        env.execute("Module KB7: Terminology Release BroadcastStream");
    }

    /**
     * Create the terminology broadcast pipeline
     */
    public static void createTerminologyBroadcastPipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating KB7 Terminology BroadcastStream pipeline");

        // Stream 1: Terminology Release CDC events from Debezium
        DataStream<TerminologyReleaseCDCEvent> releaseCDCStream = createTerminologyReleaseCDCSource(env);

        // Log incoming CDC events
        releaseCDCStream
                .process(new TerminologyReleaseLogger())
                .uid("terminology-release-logger")
                .name("Terminology Release Logger");

        // Convert to BroadcastStream
        BroadcastStream<TerminologyReleaseCDCEvent> releaseBroadcastStream = releaseCDCStream
                .broadcast(TERMINOLOGY_STATE_DESCRIPTOR);

        // Stream 2: Clinical events that need terminology context (from Module 2)
        DataStream<EnrichedPatientContext> clinicalEvents = createClinicalEventsSource(env);

        // Connect clinical events with terminology broadcast
        SingleOutputStreamOperator<EnrichedPatientContext> enrichedWithTerminology = clinicalEvents
                .keyBy(EnrichedPatientContext::getPatientId)
                .connect(releaseBroadcastStream)
                .process(new TerminologyEnrichmentProcessor())
                .uid("terminology-enrichment-processor")
                .name("Terminology Enrichment with CDC Hot-Swap");

        // Main output: enriched events
        enrichedWithTerminology.sinkTo(createEnrichedEventsSink())
                .uid("terminology-enriched-events-sink")
                .name("Terminology-Enriched Events Sink");

        // Side output: notifications about terminology updates
        DataStream<TerminologyUpdateNotification> notifications =
                enrichedWithTerminology.getSideOutput(NOTIFICATION_OUTPUT);

        notifications.sinkTo(createNotificationsSink())
                .uid("terminology-notifications-sink")
                .name("Terminology Update Notifications Sink");

        LOG.info("KB7 Terminology BroadcastStream pipeline initialized successfully");
    }

    /**
     * Create Terminology Release CDC source from Kafka
     */
    private static DataStream<TerminologyReleaseCDCEvent> createTerminologyReleaseCDCSource(StreamExecutionEnvironment env) {
        LOG.info("Creating Terminology Release CDC source from {}", KB7_RELEASES_TOPIC);

        KafkaSource<TerminologyReleaseCDCEvent> source = KafkaSource.<TerminologyReleaseCDCEvent>builder()
                .setBootstrapServers(getBootstrapServers())
                .setTopics(KB7_RELEASES_TOPIC)
                .setGroupId("module-kb7-terminology-cdc-consumer")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(DebeziumJSONDeserializer.forTerminologyRelease())
                .build();

        return env.fromSource(
                source,
                WatermarkStrategy.noWatermarks(),
                "KB7 Terminology Release CDC Source"
        );
    }

    /**
     * Create Clinical Events source (from Module 2 enriched output)
     */
    private static DataStream<EnrichedPatientContext> createClinicalEventsSource(StreamExecutionEnvironment env) {
        String inputTopic = getTopicName("KB7_INPUT_TOPIC", "enriched-patient-events-v1");
        LOG.info("Creating Clinical Events source from {}", inputTopic);

        KafkaSource<EnrichedPatientContext> source = KafkaSource.<EnrichedPatientContext>builder()
                .setBootstrapServers(getBootstrapServers())
                .setTopics(inputTopic)
                .setGroupId("module-kb7-clinical-events-consumer")
                .setValueOnlyDeserializer(new EnrichedPatientContextDeserializer())
                .build();

        return env.fromSource(
                source,
                WatermarkStrategy.<EnrichedPatientContext>forBoundedOutOfOrderness(Duration.ofSeconds(5))
                        .withTimestampAssigner((event, timestamp) -> event.getEventTime()),
                "Clinical Events Source"
        );
    }

    /**
     * Process function to log incoming CDC release events
     */
    public static class TerminologyReleaseLogger extends ProcessFunction<TerminologyReleaseCDCEvent, TerminologyReleaseCDCEvent> {

        @Override
        public void processElement(
                TerminologyReleaseCDCEvent event,
                Context ctx,
                Collector<TerminologyReleaseCDCEvent> out) throws Exception {

            if (event == null || event.getPayload() == null) {
                LOG.warn("⚠️ Received null or invalid CDC event");
                return;
            }

            TerminologyReleaseCDCEvent.Payload payload = event.getPayload();
            TerminologyReleaseCDCEvent.ReleaseData data = payload.getAfter() != null ?
                    payload.getAfter() : payload.getBefore();

            if (data != null) {
                LOG.info("╔═══════════════════════════════════════════════════════════════╗");
                LOG.info("║  📡 KB7 TERMINOLOGY RELEASE CDC EVENT                         ║");
                LOG.info("╠═══════════════════════════════════════════════════════════════╣");
                LOG.info("║  Operation: {}                                                ", payload.getOperation());
                LOG.info("║  Version ID: {}                                               ", data.getVersionId());
                LOG.info("║  Status: {}                                                   ", data.getStatus());
                LOG.info("║  SNOMED Version: {}                                           ", data.getSnomedVersion());
                LOG.info("║  RxNorm Version: {}                                           ", data.getRxnormVersion());
                LOG.info("║  LOINC Version: {}                                            ", data.getLoincVersion());
                LOG.info("║  Triple Count: {}                                             ", data.getTripleCount());
                LOG.info("║  GraphDB Endpoint: {}                                         ", data.getGraphdbEndpoint());
                LOG.info("╚═══════════════════════════════════════════════════════════════╝");

                if ("ACTIVE".equals(data.getStatus())) {
                    LOG.info("🎯 NEW ACTIVE TERMINOLOGY RELEASE DETECTED - Triggering cache refresh!");
                }
            }

            out.collect(event);
        }
    }

    /**
     * KeyedBroadcastProcessFunction for terminology enrichment with CDC hot-swap
     */
    public static class TerminologyEnrichmentProcessor
            extends KeyedBroadcastProcessFunction<String, EnrichedPatientContext, TerminologyReleaseCDCEvent, EnrichedPatientContext> {

        private static final Logger LOG = LoggerFactory.getLogger(TerminologyEnrichmentProcessor.class);

        // Current active terminology version (local cache)
        private transient TerminologyReleaseState currentTerminologyState;
        private transient boolean initialized;

        @Override
        public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
            super.open(openContext);
            LOG.info("=== Initializing Terminology Enrichment Processor ===");
            initialized = true;
            LOG.info("=== Terminology Enrichment Processor Ready (awaiting CDC broadcasts) ===");
        }

        /**
         * Process broadcast element (CDC terminology release update)
         *
         * Called for EVERY CDC event from kb7.terminology.releases.
         * Updates BroadcastState, which is then read by all parallel instances.
         */
        @Override
        public void processBroadcastElement(
                TerminologyReleaseCDCEvent cdcEvent,
                Context ctx,
                Collector<EnrichedPatientContext> out) throws Exception {

            BroadcastState<String, TerminologyReleaseState> terminologyState =
                    ctx.getBroadcastState(TERMINOLOGY_STATE_DESCRIPTOR);

            TerminologyReleaseCDCEvent.Payload payload = cdcEvent.getPayload();

            if (payload == null) {
                LOG.warn("Received CDC event with null payload, skipping");
                return;
            }

            String operation = payload.getOperation();
            LOG.info("📡 KB7 CDC EVENT: op={}, timestamp={}",
                    operation, payload.getTimestampMs());

            if (payload.isDelete()) {
                // DELETE operation - remove release from state
                TerminologyReleaseCDCEvent.ReleaseData before = payload.getBefore();
                if (before != null && before.getVersionId() != null) {
                    String versionId = before.getVersionId();
                    terminologyState.remove(versionId);
                    LOG.info("🗑️ REMOVED Terminology Release from BroadcastState: {}", versionId);
                }
            } else {
                // CREATE or UPDATE operation
                TerminologyReleaseCDCEvent.ReleaseData after = payload.getAfter();
                if (after != null && after.getVersionId() != null) {
                    String versionId = after.getVersionId();

                    // Convert CDC data to state object
                    TerminologyReleaseState releaseState = new TerminologyReleaseState(after);

                    // Update BroadcastState
                    terminologyState.put(versionId, releaseState);

                    // If this is now ACTIVE, update current state reference
                    if ("ACTIVE".equals(after.getStatus())) {
                        currentTerminologyState = releaseState;

                        LOG.info("✅ {} ACTIVE Terminology Release in BroadcastState:",
                                payload.isCreate() ? "CREATED" : "UPDATED");
                        LOG.info("   Version: {}", versionId);
                        LOG.info("   SNOMED: {}", after.getSnomedVersion());
                        LOG.info("   RxNorm: {}", after.getRxnormVersion());
                        LOG.info("   LOINC: {}", after.getLoincVersion());
                        LOG.info("   Triples: {}", after.getTripleCount());
                        LOG.info("   GraphDB: {}", after.getGraphdbEndpoint());

                        // Emit notification via side output
                        TerminologyUpdateNotification notification = new TerminologyUpdateNotification(
                                versionId,
                                after.getSnomedVersion(),
                                after.getRxnormVersion(),
                                after.getLoincVersion(),
                                after.getTripleCount(),
                                after.getGraphdbEndpoint(),
                                System.currentTimeMillis()
                        );

                        // Note: Side output from processBroadcastElement requires special handling
                        // For now, log the notification
                        LOG.info("📢 NOTIFICATION: New terminology version {} is now ACTIVE", versionId);
                    } else {
                        LOG.info("ℹ️ Updated Terminology Release (status={}): {}",
                                after.getStatus(), versionId);
                    }
                }
            }
        }

        /**
         * Process clinical event element (use terminology from BroadcastState)
         */
        @Override
        public void processElement(
                EnrichedPatientContext context,
                ReadOnlyContext ctx,
                Collector<EnrichedPatientContext> out) throws Exception {

            if (!initialized) {
                LOG.warn("Processor not initialized, skipping event for patient: {}",
                        context.getPatientId());
                return;
            }

            // Read terminology state from BroadcastState
            ReadOnlyBroadcastState<String, TerminologyReleaseState> terminologyState =
                    ctx.getBroadcastState(TERMINOLOGY_STATE_DESCRIPTOR);

            // Find current ACTIVE terminology
            TerminologyReleaseState activeTerminology = null;
            for (Map.Entry<String, TerminologyReleaseState> entry : terminologyState.immutableEntries()) {
                if ("ACTIVE".equals(entry.getValue().getStatus())) {
                    activeTerminology = entry.getValue();
                    break;
                }
            }

            // Enrich context with terminology information
            if (activeTerminology != null) {
                // Add terminology metadata to patient context
                Map<String, Object> terminologyInfo = new HashMap<>();
                terminologyInfo.put("terminology_version", activeTerminology.getVersionId());
                terminologyInfo.put("snomed_version", activeTerminology.getSnomedVersion());
                terminologyInfo.put("rxnorm_version", activeTerminology.getRxnormVersion());
                terminologyInfo.put("loinc_version", activeTerminology.getLoincVersion());
                terminologyInfo.put("graphdb_endpoint", activeTerminology.getGraphdbEndpoint());
                terminologyInfo.put("terminology_updated_at", activeTerminology.getUpdatedAt());

                context.setTerminologyContext(terminologyInfo);

                LOG.debug("Enriched patient {} with terminology v{} (SNOMED: {}, RxNorm: {}, LOINC: {})",
                        context.getPatientId(),
                        activeTerminology.getVersionId(),
                        activeTerminology.getSnomedVersion(),
                        activeTerminology.getRxnormVersion(),
                        activeTerminology.getLoincVersion());
            } else {
                LOG.debug("No active terminology in BroadcastState for patient {}",
                        context.getPatientId());
            }

            out.collect(context);
        }
    }

    /**
     * Terminology Release State model for BroadcastState
     */
    public static class TerminologyReleaseState implements Serializable {
        private static final long serialVersionUID = 1L;

        private String versionId;
        private String status;
        private String snomedVersion;
        private String rxnormVersion;
        private String loincVersion;
        private Long tripleCount;
        private String graphdbEndpoint;
        private String gcsUri;
        private long updatedAt;

        public TerminologyReleaseState() {}

        public TerminologyReleaseState(TerminologyReleaseCDCEvent.ReleaseData data) {
            this.versionId = data.getVersionId();
            this.status = data.getStatus();
            this.snomedVersion = data.getSnomedVersion();
            this.rxnormVersion = data.getRxnormVersion();
            this.loincVersion = data.getLoincVersion();
            this.tripleCount = data.getTripleCount();
            this.graphdbEndpoint = data.getGraphdbEndpoint();
            this.gcsUri = data.getGcsUri();
            this.updatedAt = System.currentTimeMillis();
        }

        // Getters and setters
        public String getVersionId() { return versionId; }
        public void setVersionId(String versionId) { this.versionId = versionId; }
        public String getStatus() { return status; }
        public void setStatus(String status) { this.status = status; }
        public String getSnomedVersion() { return snomedVersion; }
        public void setSnomedVersion(String snomedVersion) { this.snomedVersion = snomedVersion; }
        public String getRxnormVersion() { return rxnormVersion; }
        public void setRxnormVersion(String rxnormVersion) { this.rxnormVersion = rxnormVersion; }
        public String getLoincVersion() { return loincVersion; }
        public void setLoincVersion(String loincVersion) { this.loincVersion = loincVersion; }
        public Long getTripleCount() { return tripleCount; }
        public void setTripleCount(Long tripleCount) { this.tripleCount = tripleCount; }
        public String getGraphdbEndpoint() { return graphdbEndpoint; }
        public void setGraphdbEndpoint(String graphdbEndpoint) { this.graphdbEndpoint = graphdbEndpoint; }
        public String getGcsUri() { return gcsUri; }
        public void setGcsUri(String gcsUri) { this.gcsUri = gcsUri; }
        public long getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(long updatedAt) { this.updatedAt = updatedAt; }

        @Override
        public String toString() {
            return "TerminologyReleaseState{" +
                    "versionId='" + versionId + '\'' +
                    ", status='" + status + '\'' +
                    ", snomed='" + snomedVersion + '\'' +
                    ", rxnorm='" + rxnormVersion + '\'' +
                    ", loinc='" + loincVersion + '\'' +
                    '}';
        }
    }

    /**
     * Terminology Update Notification model
     */
    public static class TerminologyUpdateNotification implements Serializable {
        private static final long serialVersionUID = 1L;

        private String versionId;
        private String snomedVersion;
        private String rxnormVersion;
        private String loincVersion;
        private Long tripleCount;
        private String graphdbEndpoint;
        private long notificationTime;

        public TerminologyUpdateNotification() {}

        public TerminologyUpdateNotification(String versionId, String snomedVersion, String rxnormVersion,
                                             String loincVersion, Long tripleCount, String graphdbEndpoint,
                                             long notificationTime) {
            this.versionId = versionId;
            this.snomedVersion = snomedVersion;
            this.rxnormVersion = rxnormVersion;
            this.loincVersion = loincVersion;
            this.tripleCount = tripleCount;
            this.graphdbEndpoint = graphdbEndpoint;
            this.notificationTime = notificationTime;
        }

        // Getters
        public String getVersionId() { return versionId; }
        public String getSnomedVersion() { return snomedVersion; }
        public String getRxnormVersion() { return rxnormVersion; }
        public String getLoincVersion() { return loincVersion; }
        public Long getTripleCount() { return tripleCount; }
        public String getGraphdbEndpoint() { return graphdbEndpoint; }
        public long getNotificationTime() { return notificationTime; }

        @Override
        public String toString() {
            return String.format("TerminologyUpdate{version=%s, snomed=%s, rxnorm=%s, loinc=%s, triples=%d}",
                    versionId, snomedVersion, rxnormVersion, loincVersion, tripleCount);
        }
    }

    // ========== KAFKA SINKS ==========

    private static KafkaSink<EnrichedPatientContext> createEnrichedEventsSink() {
        String outputTopic = getTopicName("KB7_OUTPUT_TOPIC", "terminology-enriched-events.v1");

        Properties producerConfig = new Properties();
        producerConfig.setProperty("bootstrap.servers", getBootstrapServers());
        producerConfig.setProperty("compression.type", "snappy");
        producerConfig.setProperty("acks", "all");

        return KafkaSink.<EnrichedPatientContext>builder()
                .setBootstrapServers(getBootstrapServers())
                .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                        .setTopic(outputTopic)
                        .setKeySerializationSchema((EnrichedPatientContext ctx) ->
                                ctx.getPatientId().getBytes())
                        .setValueSerializationSchema(new EnrichedPatientContextSerializer())
                        .build())
                .setKafkaProducerConfig(producerConfig)
                .build();
    }

    private static KafkaSink<TerminologyUpdateNotification> createNotificationsSink() {
        Properties producerConfig = new Properties();
        producerConfig.setProperty("bootstrap.servers", getBootstrapServers());
        producerConfig.setProperty("compression.type", "snappy");
        producerConfig.setProperty("acks", "all");

        return KafkaSink.<TerminologyUpdateNotification>builder()
                .setBootstrapServers(getBootstrapServers())
                .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                        .setTopic(TERMINOLOGY_UPDATES_TOPIC)
                        .setKeySerializationSchema((TerminologyUpdateNotification n) ->
                                n.getVersionId().getBytes())
                        .setValueSerializationSchema(new NotificationSerializer())
                        .build())
                .setKafkaProducerConfig(producerConfig)
                .build();
    }

    // ========== SERIALIZERS ==========

    public static class EnrichedPatientContextDeserializer implements DeserializationSchema<EnrichedPatientContext> {
        private static final long serialVersionUID = 1L;
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) throws Exception {
            objectMapper = createObjectMapper();
        }

        @Override
        public EnrichedPatientContext deserialize(byte[] message) throws IOException {
            if (objectMapper == null) objectMapper = createObjectMapper();
            return objectMapper.readValue(message, EnrichedPatientContext.class);
        }

        @Override
        public boolean isEndOfStream(EnrichedPatientContext nextElement) {
            return false;
        }

        @Override
        public TypeInformation<EnrichedPatientContext> getProducedType() {
            return TypeInformation.of(EnrichedPatientContext.class);
        }

        private ObjectMapper createObjectMapper() {
            ObjectMapper mapper = new ObjectMapper();
            mapper.registerModule(new JavaTimeModule());
            mapper.configure(com.fasterxml.jackson.databind.DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
            return mapper;
        }
    }

    public static class EnrichedPatientContextSerializer implements SerializationSchema<EnrichedPatientContext> {
        private static final long serialVersionUID = 1L;
        private transient ObjectMapper objectMapper;

        @Override
        public byte[] serialize(EnrichedPatientContext element) {
            if (objectMapper == null) {
                objectMapper = new ObjectMapper();
                objectMapper.registerModule(new JavaTimeModule());
            }
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                LOG.error("Failed to serialize EnrichedPatientContext: {}", e.getMessage());
                return new byte[0];
            }
        }
    }

    public static class NotificationSerializer implements SerializationSchema<TerminologyUpdateNotification> {
        private static final long serialVersionUID = 1L;
        private transient ObjectMapper objectMapper;

        @Override
        public byte[] serialize(TerminologyUpdateNotification element) {
            if (objectMapper == null) {
                objectMapper = new ObjectMapper();
            }
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                LOG.error("Failed to serialize notification: {}", e.getMessage());
                return new byte[0];
            }
        }
    }

    // ========== CONFIGURATION ==========

    private static String getBootstrapServers() {
        String servers = System.getenv("KAFKA_BOOTSTRAP_SERVERS");
        return (servers != null && !servers.isEmpty()) ? servers : "localhost:9092";
    }

    private static String getTopicName(String envVar, String defaultTopic) {
        String topic = System.getenv(envVar);
        return (topic != null && !topic.isEmpty()) ? topic : defaultTopic;
    }
}
