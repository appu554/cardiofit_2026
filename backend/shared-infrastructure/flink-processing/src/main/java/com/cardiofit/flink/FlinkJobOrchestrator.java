package com.cardiofit.flink;

import com.cardiofit.flink.models.AlertAcknowledgment;
import com.cardiofit.flink.models.AuditRecord;
import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.CIDAlert;
import com.cardiofit.flink.models.ClinicalAction;
import com.cardiofit.flink.models.ClinicalEvent;
import com.cardiofit.flink.models.EngagementDropAlert;
import com.cardiofit.flink.models.EngagementSignal;
import com.cardiofit.flink.models.RelapseRiskScore;
import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.models.FhirWriteRequest;
import com.cardiofit.flink.models.MealResponseRecord;
import com.cardiofit.flink.models.MealPatternSummary;
import com.cardiofit.flink.models.ActivityResponseRecord;
import com.cardiofit.flink.models.ClinicalStateChangeEvent;
import com.cardiofit.flink.models.FitnessPatternSummary;
import com.cardiofit.flink.models.KB20StateUpdate;
import com.cardiofit.flink.models.NotificationRequest;
import com.cardiofit.flink.operators.*;
import com.cardiofit.flink.serialization.PatientIdKeySerializer;
import com.cardiofit.flink.serialization.SourceTaggingDeserializer;
import com.cardiofit.flink.sinks.KB20AsyncSinkFunction;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.connector.base.DeliveryGuarantee;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.CheckpointingMode;
import org.apache.flink.contrib.streaming.state.EmbeddedRocksDBStateBackend;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.time.Duration;

/**
 * Main entry point for CardioFit Flink EHR Intelligence Engine
 * Complete 6-module pipeline orchestrator for hybrid topic architecture
 */
public class FlinkJobOrchestrator {
    private static final Logger LOG = LoggerFactory.getLogger(FlinkJobOrchestrator.class);

    public static void main(String[] args) throws Exception {
        LOG.info("Starting CardioFit EHR Intelligence Engine - Complete Pipeline");

        // Parse command line arguments
        // Default to comprehensive-cds (Module 3 with all 8 phases integrated)
        String jobType = args.length > 0 ? args[0] : "comprehensive-cds";
        String environmentMode = args.length > 1 ? args[1] : "production";

        // Initialize Flink execution environment
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Configure environment for healthcare workloads
        configureEnvironment(env, environmentMode);

        // Launch complete pipeline based on job type
        switch (jobType.toLowerCase()) {
            case "full-pipeline":
                launchFullPipeline(env);
                break;
            case "ingestion-only":
                Module1_Ingestion.createIngestionPipeline(env);
                break;
            case "context-assembly":
                // Use unified pipeline: outputs EnrichedPatientContext (camelCase)
                // which Module 3 expects, instead of EnrichedEvent (snake_case)
                Module2_Enhanced.createUnifiedPipeline(env);
                break;
            case "comprehensive-cds":
                // Module 3: Comprehensive CDS with all 8 phases integrated
                Module3_ComprehensiveCDS.createComprehensiveCDSPipeline(env);
                break;
            case "semantic-mesh":
                // Module 3: Basic semantic mesh (legacy)
                Module3_SemanticMesh.createSemanticMeshPipeline(env);
                break;
            case "pattern-detection":
                Module4_PatternDetection.createPatternDetectionPipeline(env);
                break;
            case "ml-inference":
                Module5_MLInference.createMLInferencePipeline(env);
                break;
            case "egress-routing":
                Module6_EgressRouting.createEgressRoutingPipeline(env);
                break;
            case "module1b-canonicalizer":
            case "ingestion-canonicalizer":
                // Module 1b: Canonicalizes ingestion service outbox events
                // Consumes all 9 ingestion.* topics → enriched-patient-events-v1
                Module1b_IngestionCanonicalizer.createIngestionPipeline(env);
                break;
            case "bp-variability":
            case "module7":
            case "bp-variability-engine":
                launchBPVariabilityEngine(env);
                break;
            case "comorbidity":
            case "module8":
            case "comorbidity-interaction":
                launchComorbidityEngine(env);
                break;
            case "clinical-action-engine":
            case "module6-cae":
                launchClinicalActionEngine(env);
                break;
            case "engagement":
            case "module9":
            case "engagement-monitor":
                launchEngagementMonitor(env);
                break;
            case "meal-response":
            case "module10":
            case "meal-response-correlator":
                launchMealResponseCorrelator(env);
                break;
            case "meal-patterns":
            case "module10b":
            case "meal-pattern-aggregator":
                launchMealPatternAggregator(env);
                break;
            case "activity-response":
            case "module11":
            case "activity-response-correlator":
                launchActivityResponseCorrelator(env);
                break;
            case "fitness-patterns":
            case "module11b":
            case "fitness-pattern-aggregator":
                launchFitnessPatternAggregator(env);
                break;
            case "clinical-state-sync":
            case "module13":
            case "clinical-state-synchroniser":
                launchClinicalStateSynchroniser(env);
                break;
            default:
                LOG.warn("Unknown job type: {}. Defaulting to comprehensive CDS.", jobType);
                Module3_ComprehensiveCDS.createComprehensiveCDSPipeline(env);
        }

        // Execute the complete pipeline
        String jobName = String.format("CardioFit EHR Intelligence - %s (%s)",
                                      jobType, environmentMode);
        LOG.info("Executing job: {}", jobName);
        env.execute(jobName);
    }

    /**
     * Configure Flink environment for healthcare data processing
     */
    private static void configureEnvironment(StreamExecutionEnvironment env, String environmentMode) {
        LOG.info("Configuring Flink environment for mode: {}", environmentMode);

        // Set parallelism based on environment
        // Reduced from 8 to 2 for initial deployment to avoid RPC coordination overhead
        int parallelism = "production".equals(environmentMode) ? 2 : 2;
        env.setParallelism(parallelism);

        // Configure checkpointing for exactly-once processing
        env.enableCheckpointing(30000); // 30 second checkpoints
        env.getCheckpointConfig().setCheckpointingMode(CheckpointingMode.EXACTLY_ONCE);
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);
        env.getCheckpointConfig().setCheckpointTimeout(600000); // 10 minutes
        env.getCheckpointConfig().setTolerableCheckpointFailureNumber(3);

        // Configure state backend for large state (Flink 2.x compatible)
        try {
            // For Flink 2.x, state backend configuration is different
            // Using default state backend for compatibility
            LOG.info("Using default state backend (compatible with Flink 2.x)");
        } catch (Exception e) {
            LOG.warn("Failed to configure state backend, using default: {}", e.getMessage());
        }

        // Configure restart strategy
        env.getConfig().setAutoWatermarkInterval(1000);

        // Configure for healthcare compliance
        env.getConfig().setGlobalJobParameters(KafkaConfigLoader.getGlobalParameters());

        LOG.info("Environment configured: parallelism={}, checkpointing=30s", parallelism);
    }

    /**
     * Launch the Module 6 Clinical Action Engine pipeline.
     *
     * Consumes ClinicalEvent records from CLINICAL_REASONING_EVENTS, keys by patientId,
     * processes through Module6_ClinicalActionEngine, and routes outputs to:
     *   - Main output (ClinicalAction)      → CLINICAL_ACTIONS
     *   - NOTIFICATION_TAG side output      → CLINICAL_NOTIFICATIONS
     *   - AUDIT_TAG side output             → CLINICAL_AUDIT
     *   - FHIR_TAG side output              → FHIR_WRITEBACK
     */
    private static void launchClinicalActionEngine(StreamExecutionEnvironment env) {
        LOG.info("Launching Module 6: Clinical Action Engine pipeline (dual-input)");

        String bootstrapServers = KafkaConfigLoader.getBootstrapServers();

        // Input 1: ClinicalEvent from Modules 3/4/5
        KafkaSource<ClinicalEvent> eventSource = KafkaSource.<ClinicalEvent>builder()
            .setBootstrapServers(bootstrapServers)
            .setTopics(KafkaTopics.CLINICAL_REASONING_EVENTS.getTopicName())
            .setGroupId("flink-module6-clinical-action-engine")
            .setStartingOffsets(OffsetsInitializer.latest())
            .setValueOnlyDeserializer(new ClinicalEventDeserializer())
            .build();

        // Input 2: AlertAcknowledgment from physicians
        KafkaSource<AlertAcknowledgment> ackSource = KafkaSource.<AlertAcknowledgment>builder()
            .setBootstrapServers(bootstrapServers)
            .setTopics(KafkaTopics.ALERT_ACKNOWLEDGMENTS.getTopicName())
            .setGroupId("flink-module6-acknowledgments")
            .setStartingOffsets(OffsetsInitializer.latest())
            .setValueOnlyDeserializer(new AlertAcknowledgmentDeserializer())
            .build();

        var eventStream = env.fromSource(
            eventSource,
            WatermarkStrategy.<ClinicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                .withTimestampAssigner((event, ts) -> event.getEventTime()),
            "Kafka Source: Clinical Reasoning Events"
        ).keyBy(ClinicalEvent::getPatientId);

        var ackStream = env.fromSource(
            ackSource,
            WatermarkStrategy.<AlertAcknowledgment>forBoundedOutOfOrderness(Duration.ofSeconds(30))
                .withTimestampAssigner((ack, ts) -> ack.getTimestamp()),
            "Kafka Source: Alert Acknowledgments"
        ).keyBy(AlertAcknowledgment::getPatientId);

        SingleOutputStreamOperator<ClinicalAction> actions = eventStream
            .connect(ackStream)
            .process(new Module6_ClinicalActionEngine())
            .uid("Module6 Clinical Action Engine")
            .name("Module6 Clinical Action Engine");

        // Main output → clinical-actions.v1
        actions.sinkTo(
            KafkaSink.<ClinicalAction>builder()
                .setBootstrapServers(bootstrapServers)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.CLINICAL_ACTIONS.getTopicName())
                        .setValueSerializationSchema(new ClinicalActionSerializer())
                        .build())
                .build()
        );

        // Side output: notifications → clinical-notifications.v1
        actions.getSideOutput(Module6_ClinicalActionEngine.NOTIFICATION_TAG).sinkTo(
            KafkaSink.<NotificationRequest>builder()
                .setBootstrapServers(bootstrapServers)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.<NotificationRequest>builder()
                        .setTopic(KafkaTopics.CLINICAL_NOTIFICATIONS.getTopicName())
                        .setValueSerializationSchema(new JsonSerializer<NotificationRequest>())
                        .build())
                .build()
        );

        // Side output: audit records → clinical-audit.v1
        actions.getSideOutput(Module6_ClinicalActionEngine.AUDIT_TAG).sinkTo(
            KafkaSink.<AuditRecord>builder()
                .setBootstrapServers(bootstrapServers)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.<AuditRecord>builder()
                        .setTopic(KafkaTopics.CLINICAL_AUDIT.getTopicName())
                        .setValueSerializationSchema(new JsonSerializer<AuditRecord>())
                        .build())
                .build()
        );

        // Side output: FHIR writeback → fhir-writeback.v1
        actions.getSideOutput(Module6_ClinicalActionEngine.FHIR_TAG).sinkTo(
            KafkaSink.<FhirWriteRequest>builder()
                .setBootstrapServers(bootstrapServers)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.<FhirWriteRequest>builder()
                        .setTopic(KafkaTopics.FHIR_WRITEBACK.getTopicName())
                        .setValueSerializationSchema(new JsonSerializer<FhirWriteRequest>())
                        .build())
                .build()
        );

        LOG.info("Module 6 Clinical Action Engine pipeline configured: "
            + "sources=[{}, {}], sinks=[{}, {}, {}, {}]",
            KafkaTopics.CLINICAL_REASONING_EVENTS.getTopicName(),
            KafkaTopics.ALERT_ACKNOWLEDGMENTS.getTopicName(),
            KafkaTopics.CLINICAL_ACTIONS.getTopicName(),
            KafkaTopics.CLINICAL_NOTIFICATIONS.getTopicName(),
            KafkaTopics.CLINICAL_AUDIT.getTopicName(),
            KafkaTopics.FHIR_WRITEBACK.getTopicName());
    }

    /** Deserializes JSON bytes into a ClinicalEvent using Jackson. */
    private static class ClinicalEventDeserializer implements DeserializationSchema<ClinicalEvent> {
        private transient ObjectMapper mapper;

        @Override
        public void open(InitializationContext context) {
            mapper = new ObjectMapper();
            mapper.registerModule(new JavaTimeModule());
        }

        @Override
        public ClinicalEvent deserialize(byte[] message) throws IOException {
            if (message == null || message.length == 0) return null;
            return mapper.readValue(message, ClinicalEvent.class);
        }

        @Override
        public boolean isEndOfStream(ClinicalEvent nextElement) {
            return false;
        }

        @Override
        public TypeInformation<ClinicalEvent> getProducedType() {
            return TypeInformation.of(ClinicalEvent.class);
        }
    }

    /** Deserializes JSON bytes into an AlertAcknowledgment using Jackson. */
    private static class AlertAcknowledgmentDeserializer implements DeserializationSchema<AlertAcknowledgment> {
        private transient ObjectMapper mapper;

        @Override
        public void open(InitializationContext context) {
            mapper = new ObjectMapper();
            mapper.registerModule(new JavaTimeModule());
        }

        @Override
        public AlertAcknowledgment deserialize(byte[] message) throws IOException {
            if (message == null || message.length == 0) return null;
            return mapper.readValue(message, AlertAcknowledgment.class);
        }

        @Override
        public boolean isEndOfStream(AlertAcknowledgment nextElement) {
            return false;
        }

        @Override
        public TypeInformation<AlertAcknowledgment> getProducedType() {
            return TypeInformation.of(AlertAcknowledgment.class);
        }
    }

    /** Serializes a ClinicalAction to JSON bytes. */
    private static class ClinicalActionSerializer implements SerializationSchema<ClinicalAction> {
        private transient ObjectMapper mapper;

        @Override
        public void open(InitializationContext context) {
            mapper = new ObjectMapper();
            mapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(ClinicalAction element) {
            try {
                return mapper.writeValueAsBytes(element);
            } catch (Exception e) {
                throw new RuntimeException("Failed to serialize ClinicalAction", e);
            }
        }
    }

    /** Generic JSON serializer for side-output model types. */
    private static class JsonSerializer<T> implements SerializationSchema<T> {
        private transient ObjectMapper mapper;

        @Override
        public void open(InitializationContext context) {
            mapper = new ObjectMapper();
            mapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(T element) {
            try {
                return mapper.writeValueAsBytes(element);
            } catch (Exception e) {
                throw new RuntimeException("Failed to serialize " + element.getClass().getSimpleName(), e);
            }
        }
    }

    /**
     * Module 7: BP Variability Engine.
     * Consumes ingestion.vitals, keys by patientId, produces bp-variability-metrics
     * and safety-critical side output.
     */
    private static void launchBPVariabilityEngine(StreamExecutionEnvironment env) {
        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        // Kafka source: BPReading from ingestion.vitals
        KafkaSource<com.cardiofit.flink.models.BPReading> source = KafkaSource
            .<com.cardiofit.flink.models.BPReading>builder()
            .setBootstrapServers(bootstrap)
            .setTopics(KafkaTopics.INGESTION_VITALS.getTopicName())
            .setGroupId("flink-module7-bp-variability-v2")
            .setValueOnlyDeserializer(new BPReadingDeserializer())
            .build();

        SingleOutputStreamOperator<com.cardiofit.flink.models.BPVariabilityMetrics> metrics = env
            .fromSource(source,
                WatermarkStrategy.<com.cardiofit.flink.models.BPReading>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                    .withTimestampAssigner((r, ts) -> r.getTimestamp()),
                "Kafka Source: BP Readings")
            .filter(r -> r != null && r.getPatientId() != null)
            .keyBy(com.cardiofit.flink.models.BPReading::getPatientId)
            .process(new Module7_BPVariabilityEngine())
            .uid("module7-bp-variability-engine")
            .name("Module 7: BP Variability Engine");

        // Main output → flink.bp-variability-metrics
        metrics.sinkTo(
            KafkaSink.<com.cardiofit.flink.models.BPVariabilityMetrics>builder()
                .setBootstrapServers(bootstrap)
                .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                .setTransactionalIdPrefix("m7-metrics")
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.FLINK_BP_VARIABILITY_METRICS.getTopicName())
                        .setValueSerializationSchema(new BPMetricsSerializer())
                        .build())
                .build());

        // Crisis side output → ingestion.safety-critical
        metrics.getSideOutput(Module7_BPVariabilityEngine.CRISIS_TAG).sinkTo(
            KafkaSink.<com.cardiofit.flink.models.BPReading>builder()
                .setBootstrapServers(bootstrap)
                .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                .setTransactionalIdPrefix("m7-crisis")
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.INGESTION_SAFETY_CRITICAL.getTopicName())
                        .setValueSerializationSchema(new BPReadingSerializer())
                        .build())
                .build());

        // Acute surge side output → ingestion.safety-critical (same topic, separate tag
        // for future routing flexibility — downstream distinguishes by SBP delta vs threshold)
        metrics.getSideOutput(Module7_BPVariabilityEngine.ACUTE_SURGE_TAG).sinkTo(
            KafkaSink.<com.cardiofit.flink.models.BPReading>builder()
                .setBootstrapServers(bootstrap)
                .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                .setTransactionalIdPrefix("m7-surge")
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.INGESTION_SAFETY_CRITICAL.getTopicName())
                        .setValueSerializationSchema(new BPReadingSerializer())
                        .build())
                .build());
    }

    // --- Module 7 serializers ---

    static class BPReadingDeserializer implements DeserializationSchema<com.cardiofit.flink.models.BPReading> {
        private transient ObjectMapper mapper;
        @Override public void open(InitializationContext ctx) {
            mapper = new ObjectMapper();
        }
        @Override public com.cardiofit.flink.models.BPReading deserialize(byte[] bytes) throws IOException {
            if (bytes == null || bytes.length == 0) return null;
            try {
                return mapper.readValue(bytes, com.cardiofit.flink.models.BPReading.class);
            } catch (Exception e) {
                LOG.warn("Failed to deserialize BPReading, skipping: {}", e.getMessage());
                return null;
            }
        }
        @Override public boolean isEndOfStream(com.cardiofit.flink.models.BPReading r) { return false; }
        @Override public TypeInformation<com.cardiofit.flink.models.BPReading> getProducedType() {
            return TypeInformation.of(com.cardiofit.flink.models.BPReading.class);
        }
    }

    static class BPMetricsSerializer implements SerializationSchema<com.cardiofit.flink.models.BPVariabilityMetrics> {
        private transient ObjectMapper mapper;
        @Override public void open(InitializationContext ctx) {
            mapper = new ObjectMapper();
        }
        @Override public byte[] serialize(com.cardiofit.flink.models.BPVariabilityMetrics m) {
            try { return mapper.writeValueAsBytes(m); }
            catch (Exception e) { throw new RuntimeException("Serialize BPMetrics failed", e); }
        }
    }

    static class BPReadingSerializer implements SerializationSchema<com.cardiofit.flink.models.BPReading> {
        private transient ObjectMapper mapper;
        @Override public void open(InitializationContext ctx) {
            mapper = new ObjectMapper();
        }
        @Override public byte[] serialize(com.cardiofit.flink.models.BPReading r) {
            try { return mapper.writeValueAsBytes(r); }
            catch (Exception e) { throw new RuntimeException("Serialize BPReading failed", e); }
        }
    }

    /**
     * R10: Explicit launcher with dual-sink wiring.
     * Main output → alerts.comorbidity-interactions
     * HALT side-output → ingestion.safety-critical (patient safety fast-path)
     */
    private static void launchComorbidityEngine(StreamExecutionEnvironment env) {
        LOG.info("Launching Module 8: Comorbidity Interaction Engine pipeline");

        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        // Source: CanonicalEvent from enriched-patient-events-v1
        KafkaSource<CanonicalEvent> source = KafkaSource.<CanonicalEvent>builder()
            .setBootstrapServers(bootstrap)
            .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
            .setGroupId("flink-module8-comorbidity-engine-v2")
            .setStartingOffsets(OffsetsInitializer.earliest())
            .setValueOnlyDeserializer(new CanonicalEventDeserializer())
            .build();

        SingleOutputStreamOperator<CIDAlert> alerts = env
            .fromSource(source,
                WatermarkStrategy.<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                    .withTimestampAssigner((e, ts) -> e.getEventTime()),
                "Kafka Source: Enriched Patient Events (Module 8)")
            .keyBy(CanonicalEvent::getPatientId)
            .process(new Module8_ComorbidityEngine())
            .uid("module8-comorbidity-engine")
            .name("Module 8: Comorbidity Interaction Engine");

        // Main output → alerts.comorbidity-interactions (all severities)
        alerts.sinkTo(
            KafkaSink.<CIDAlert>builder()
                .setBootstrapServers(bootstrap)
                .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                .setTransactionalIdPrefix("m8-comorbidity-alerts")
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.<CIDAlert>builder()
                        .setTopic(KafkaTopics.ALERTS_COMORBIDITY_INTERACTIONS.getTopicName())
                        .setValueSerializationSchema(new JsonSerializer<CIDAlert>())
                        .build())
                .build()
        ).name("Sink: Comorbidity Alerts");

        // HALT side-output → ingestion.safety-critical (fast-path, never suppressed)
        alerts.getSideOutput(Module8_ComorbidityEngine.HALT_SAFETY_TAG).sinkTo(
            KafkaSink.<CIDAlert>builder()
                .setBootstrapServers(bootstrap)
                .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                .setTransactionalIdPrefix("m8-halt-safety")
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.<CIDAlert>builder()
                        .setTopic(KafkaTopics.INGESTION_SAFETY_CRITICAL.getTopicName())
                        .setValueSerializationSchema(new JsonSerializer<CIDAlert>())
                        .build())
                .build()
        ).name("Sink: HALT Safety-Critical Alerts");

        LOG.info("Module 8 Comorbidity Engine pipeline configured: "
            + "source=[{}], sinks=[{}, {}]",
            KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName(),
            KafkaTopics.ALERTS_COMORBIDITY_INTERACTIONS.getTopicName(),
            KafkaTopics.INGESTION_SAFETY_CRITICAL.getTopicName());
    }

    /**
     * Module 9: Engagement Monitor.
     * Timer-driven daily engagement scoring with 8 DD#8-reconciled signal channels.
     * Dual sink: engagement signals (main) + drop alerts (side output).
     */
    private static void launchEngagementMonitor(StreamExecutionEnvironment env) {
        LOG.info("Launching Module 9: Engagement Monitor pipeline");

        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        // Source: CanonicalEvent from enriched-patient-events-v1 (same as Module 8)
        KafkaSource<CanonicalEvent> source = KafkaSource.<CanonicalEvent>builder()
            .setBootstrapServers(bootstrap)
            .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
            .setGroupId("flink-module9-engagement-monitor-v1")
            .setStartingOffsets(OffsetsInitializer.earliest())
            .setValueOnlyDeserializer(new CanonicalEventDeserializer())
            .build();

        // Pipeline: keyBy patientId -> Module 9 operator
        SingleOutputStreamOperator<EngagementSignal> signals = env
            .fromSource(source,
                WatermarkStrategy.<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                    .withTimestampAssigner((e, ts) -> e.getEventTime()),
                "Kafka Source: Enriched Patient Events (Module 9)")
            .keyBy(CanonicalEvent::getPatientId)
            .process(new Module9_EngagementMonitor())
            .uid("module9-engagement-monitor")
            .name("Module 9: Engagement Monitor");

        // Main output -> flink.engagement-signals
        signals.sinkTo(
            KafkaSink.<EngagementSignal>builder()
                .setBootstrapServers(bootstrap)
                .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                .setTransactionalIdPrefix("m9-engagement-signals")
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.<EngagementSignal>builder()
                        .setTopic(KafkaTopics.FLINK_ENGAGEMENT_SIGNALS.getTopicName())
                        .setValueSerializationSchema(new JsonSerializer<EngagementSignal>())
                        .build())
                .build()
        ).name("Sink: Engagement Signals");

        // Side output -> alerts.engagement-drop
        signals.getSideOutput(Module9_EngagementMonitor.ENGAGEMENT_DROP_TAG).sinkTo(
            KafkaSink.<EngagementDropAlert>builder()
                .setBootstrapServers(bootstrap)
                .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                .setTransactionalIdPrefix("m9-engagement-drop-alerts")
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.<EngagementDropAlert>builder()
                        .setTopic(KafkaTopics.ALERTS_ENGAGEMENT_DROP.getTopicName())
                        .setValueSerializationSchema(new JsonSerializer<EngagementDropAlert>())
                        .build())
                .build()
        ).name("Sink: Engagement Drop Alerts");

        // Phase 2 side output -> alerts.relapse-risk
        signals.getSideOutput(Module9_EngagementMonitor.RELAPSE_RISK_TAG).sinkTo(
            KafkaSink.<RelapseRiskScore>builder()
                .setBootstrapServers(bootstrap)
                .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                .setTransactionalIdPrefix("m9-relapse-risk-alerts")
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.<RelapseRiskScore>builder()
                        .setTopic(KafkaTopics.ALERTS_RELAPSE_RISK.getTopicName())
                        .setValueSerializationSchema(new JsonSerializer<RelapseRiskScore>())
                        .build())
                .build()
        ).name("Sink: Relapse Risk Alerts");

        LOG.info("Module 9 Engagement Monitor pipeline configured: "
            + "source=[{}], sinks=[{}, {}, {}]",
            KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName(),
            KafkaTopics.FLINK_ENGAGEMENT_SIGNALS.getTopicName(),
            KafkaTopics.ALERTS_ENGAGEMENT_DROP.getTopicName(),
            KafkaTopics.ALERTS_RELAPSE_RISK.getTopicName());
    }

    /**
     * Module 10: Meal Response Correlator.
     * Session-window-driven per-meal glucose/BP correlation.
     * Single sink: MealResponseRecord → flink.meal-response.
     */
    private static void launchMealResponseCorrelator(StreamExecutionEnvironment env) {
        LOG.info("Launching Module 10: Meal Response Correlator pipeline");

        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        KafkaSource<CanonicalEvent> source = KafkaSource.<CanonicalEvent>builder()
            .setBootstrapServers(bootstrap)
            .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
            .setGroupId("flink-module10-meal-response-v1")
            .setStartingOffsets(OffsetsInitializer.earliest())
            .setValueOnlyDeserializer(new CanonicalEventDeserializer())
            .build();

        SingleOutputStreamOperator<MealResponseRecord> records = env
            .fromSource(source,
                WatermarkStrategy.<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                    .withTimestampAssigner((e, ts) -> e.getEventTime()),
                "Kafka Source: Enriched Patient Events (Module 10)")
            .keyBy(CanonicalEvent::getPatientId)
            .process(new Module10_MealResponseCorrelator())
            .uid("module10-meal-response-correlator")
            .name("Module 10: Meal Response Correlator");

        // Main output → flink.meal-response
        records.sinkTo(
            KafkaSink.<MealResponseRecord>builder()
                .setBootstrapServers(bootstrap)
                .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                .setTransactionalIdPrefix("m10-meal-response")
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.<MealResponseRecord>builder()
                        .setTopic(KafkaTopics.FLINK_MEAL_RESPONSE.getTopicName())
                        .setValueSerializationSchema(new JsonSerializer<MealResponseRecord>())
                        .build())
                .build()
        ).name("Sink: Meal Response Records");

        LOG.info("Module 10 pipeline configured: source=[{}], sink=[{}]",
            KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName(),
            KafkaTopics.FLINK_MEAL_RESPONSE.getTopicName());
    }

    /**
     * Module 10b: Meal Pattern Aggregator.
     * Weekly aggregation of meal response records with OLS salt sensitivity.
     * Separate job from Module 10 for failure isolation.
     * Input: MealResponseRecord from flink.meal-response
     * Output: MealPatternSummary to flink.meal-patterns
     */
    private static void launchMealPatternAggregator(StreamExecutionEnvironment env) {
        LOG.info("Launching Module 10b: Meal Pattern Aggregator pipeline");

        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        // Source: MealResponseRecord from flink.meal-response (output of Module 10)
        KafkaSource<MealResponseRecord> source = KafkaSource.<MealResponseRecord>builder()
            .setBootstrapServers(bootstrap)
            .setTopics(KafkaTopics.FLINK_MEAL_RESPONSE.getTopicName())
            .setGroupId("flink-module10b-meal-patterns-v1")
            .setStartingOffsets(OffsetsInitializer.earliest())
            .setValueOnlyDeserializer(new MealResponseRecordDeserializer())
            .build();

        SingleOutputStreamOperator<MealPatternSummary> summaries = env
            .fromSource(source,
                WatermarkStrategy.<MealResponseRecord>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                    .withTimestampAssigner((r, ts) -> r.getMealTimestamp()),
                "Kafka Source: Meal Response Records (Module 10b)")
            .keyBy(MealResponseRecord::getPatientId)
            .process(new Module10b_MealPatternAggregator())
            .uid("module10b-meal-pattern-aggregator")
            .name("Module 10b: Meal Pattern Aggregator");

        // Output → flink.meal-patterns
        summaries.sinkTo(
            KafkaSink.<MealPatternSummary>builder()
                .setBootstrapServers(bootstrap)
                .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                .setTransactionalIdPrefix("m10b-meal-patterns")
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.<MealPatternSummary>builder()
                        .setTopic(KafkaTopics.FLINK_MEAL_PATTERNS.getTopicName())
                        .setValueSerializationSchema(new JsonSerializer<MealPatternSummary>())
                        .build())
                .build()
        ).name("Sink: Meal Pattern Summaries");

        LOG.info("Module 10b pipeline configured: source=[{}], sink=[{}]",
            KafkaTopics.FLINK_MEAL_RESPONSE.getTopicName(),
            KafkaTopics.FLINK_MEAL_PATTERNS.getTopicName());
    }

    /** Deserializes JSON bytes into a MealResponseRecord using Jackson. */
    static class MealResponseRecordDeserializer implements DeserializationSchema<MealResponseRecord> {
        private transient ObjectMapper mapper;

        @Override
        public void open(InitializationContext context) {
            mapper = new ObjectMapper();
            mapper.registerModule(new JavaTimeModule());
        }

        private ObjectMapper mapper() {
            if (mapper == null) {
                mapper = new ObjectMapper();
                mapper.registerModule(new JavaTimeModule());
            }
            return mapper;
        }

        @Override
        public MealResponseRecord deserialize(byte[] message) throws IOException {
            if (message == null || message.length == 0) return null;
            return mapper().readValue(message, MealResponseRecord.class);
        }

        @Override
        public boolean isEndOfStream(MealResponseRecord nextElement) {
            return false;
        }

        @Override
        public TypeInformation<MealResponseRecord> getProducedType() {
            return TypeInformation.of(MealResponseRecord.class);
        }
    }

    /**
     * Module 11: Activity Response Correlator.
     * Exercise-session-window-driven per-activity HR/glucose/BP correlation.
     * Single sink: ActivityResponseRecord → flink.activity-response.
     */
    private static void launchActivityResponseCorrelator(StreamExecutionEnvironment env) {
        LOG.info("Launching Module 11: Activity Response Correlator pipeline");

        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        KafkaSource<CanonicalEvent> source = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
                .setGroupId("flink-module11-activity-response-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new CanonicalEventDeserializer())
                .build();

        SingleOutputStreamOperator<ActivityResponseRecord> records = env
                .fromSource(source,
                        WatermarkStrategy.<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                                .withTimestampAssigner((e, ts) -> e.getEventTime()),
                        "Kafka Source: Enriched Patient Events (Module 11)")
                .keyBy(CanonicalEvent::getPatientId)
                .process(new Module11_ActivityResponseCorrelator())
                .uid("module11-activity-response-correlator")
                .name("Module 11: Activity Response Correlator");

        records.sinkTo(
                KafkaSink.<ActivityResponseRecord>builder()
                        .setBootstrapServers(bootstrap)
                        .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                        .setTransactionalIdPrefix("m11-activity-response")
                        .setRecordSerializer(
                                KafkaRecordSerializationSchema.<ActivityResponseRecord>builder()
                                        .setTopic(KafkaTopics.FLINK_ACTIVITY_RESPONSE.getTopicName())
                                        .setValueSerializationSchema(new JsonSerializer<ActivityResponseRecord>())
                                        .build())
                        .build()
        ).name("Sink: Activity Response Records");

        LOG.info("Module 11 pipeline configured: source=[{}], sink=[{}]",
                KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName(),
                KafkaTopics.FLINK_ACTIVITY_RESPONSE.getTopicName());
    }

    /**
     * Module 11b: Fitness Pattern Aggregator.
     * Weekly aggregation of activity response records with VO2max estimation.
     * Separate job from Module 11 for failure isolation.
     */
    private static void launchFitnessPatternAggregator(StreamExecutionEnvironment env) {
        LOG.info("Launching Module 11b: Fitness Pattern Aggregator pipeline");

        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        KafkaSource<ActivityResponseRecord> source = KafkaSource.<ActivityResponseRecord>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.FLINK_ACTIVITY_RESPONSE.getTopicName())
                .setGroupId("flink-module11b-fitness-patterns-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new ActivityResponseRecordDeserializer())
                .build();

        SingleOutputStreamOperator<FitnessPatternSummary> summaries = env
                .fromSource(source,
                        WatermarkStrategy.<ActivityResponseRecord>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                                .withTimestampAssigner((r, ts) -> r.getActivityStartTime()),
                        "Kafka Source: Activity Response Records (Module 11b)")
                .keyBy(ActivityResponseRecord::getPatientId)
                .process(new Module11b_FitnessPatternAggregator())
                .uid("module11b-fitness-pattern-aggregator")
                .name("Module 11b: Fitness Pattern Aggregator");

        summaries.sinkTo(
                KafkaSink.<FitnessPatternSummary>builder()
                        .setBootstrapServers(bootstrap)
                        .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                        .setTransactionalIdPrefix("m11b-fitness-patterns")
                        .setRecordSerializer(
                                KafkaRecordSerializationSchema.<FitnessPatternSummary>builder()
                                        .setTopic(KafkaTopics.FLINK_FITNESS_PATTERNS.getTopicName())
                                        .setValueSerializationSchema(new JsonSerializer<FitnessPatternSummary>())
                                        .build())
                        .build()
        ).name("Sink: Fitness Pattern Summaries");

        LOG.info("Module 11b pipeline configured: source=[{}], sink=[{}]",
                KafkaTopics.FLINK_ACTIVITY_RESPONSE.getTopicName(),
                KafkaTopics.FLINK_FITNESS_PATTERNS.getTopicName());
    }

    /** Deserializes JSON bytes into an ActivityResponseRecord using Jackson. */
    static class ActivityResponseRecordDeserializer implements DeserializationSchema<ActivityResponseRecord> {
        private transient ObjectMapper mapper;

        @Override
        public void open(InitializationContext context) {
            mapper = new ObjectMapper();
            mapper.registerModule(new JavaTimeModule());
        }

        private ObjectMapper mapper() {
            if (mapper == null) {
                mapper = new ObjectMapper();
                mapper.registerModule(new JavaTimeModule());
            }
            return mapper;
        }

        @Override
        public ActivityResponseRecord deserialize(byte[] message) throws IOException {
            if (message == null || message.length == 0) return null;
            return mapper().readValue(message, ActivityResponseRecord.class);
        }

        @Override
        public boolean isEndOfStream(ActivityResponseRecord nextElement) {
            return false;
        }

        @Override
        public TypeInformation<ActivityResponseRecord> getProducedType() {
            return TypeInformation.of(ActivityResponseRecord.class);
        }
    }

    /** Deserializes JSON bytes into a CanonicalEvent using Jackson. */
    static class CanonicalEventDeserializer implements DeserializationSchema<CanonicalEvent> {
        private transient ObjectMapper mapper;

        @Override
        public void open(InitializationContext context) {
            mapper = new ObjectMapper();
            mapper.registerModule(new JavaTimeModule());
        }

        @Override
        public CanonicalEvent deserialize(byte[] message) throws IOException {
            if (message == null || message.length == 0) return null;
            return mapper.readValue(message, CanonicalEvent.class);
        }

        @Override
        public boolean isEndOfStream(CanonicalEvent nextElement) {
            return false;
        }

        @Override
        public TypeInformation<CanonicalEvent> getProducedType() {
            return TypeInformation.of(CanonicalEvent.class);
        }
    }

    /**
     * Launch the complete 6-module EHR Intelligence pipeline
     */
    private static void launchFullPipeline(StreamExecutionEnvironment env) {
        LOG.info("Launching complete EHR Intelligence pipeline with all 13 modules (1, 1b, 2-11b)");

        try {
            // Module 1: Ingestion & Gateway (traditional EHR sources)
            LOG.info("Initializing Module 1: Ingestion & Gateway");
            Module1_Ingestion.createIngestionPipeline(env);

            // Module 1b: Ingestion Canonicalizer (outbox events from ingestion service)
            LOG.info("Initializing Module 1b: Ingestion Canonicalizer");
            Module1b_IngestionCanonicalizer.createIngestionPipeline(env);

            // Module 2: Enhanced Context Assembly
            LOG.info("Initializing Module 2: Enhanced Context Assembly");
            Module2_Enhanced.createEnhancedPipeline(env);

            // Module 3: Semantic Mesh
            LOG.info("Initializing Module 3: Semantic Mesh");
            Module3_SemanticMesh.createSemanticMeshPipeline(env);

            // Module 4: Pattern Detection
            LOG.info("Initializing Module 4: Pattern Detection");
            Module4_PatternDetection.createPatternDetectionPipeline(env);

            // Module 5: ML Inference
            LOG.info("Initializing Module 5: ML Inference");
            Module5_MLInference.createMLInferencePipeline(env);

            // Module 6: Egress Routing
            LOG.info("Initializing Module 6: Egress Routing");
            Module6_EgressRouting.createEgressRoutingPipeline(env);

            // Module 7: BP Variability Engine
            LOG.info("Initializing Module 7: BP Variability Engine");
            launchBPVariabilityEngine(env);

            // Module 8: Comorbidity Interaction Detector
            LOG.info("Initializing Module 8: Comorbidity Interaction Detector");
            launchComorbidityEngine(env);

            // Module 9: Engagement Monitor
            LOG.info("Initializing Module 9: Engagement Monitor");
            launchEngagementMonitor(env);

            // Module 10: Meal Response Correlator
            LOG.info("Initializing Module 10: Meal Response Correlator");
            launchMealResponseCorrelator(env);

            // Module 10b: Meal Pattern Aggregator
            LOG.info("Initializing Module 10b: Meal Pattern Aggregator");
            launchMealPatternAggregator(env);

            // Module 11: Activity Response Correlator
            LOG.info("Initializing Module 11: Activity Response Correlator");
            launchActivityResponseCorrelator(env);

            // Module 11b: Fitness Pattern Aggregator
            LOG.info("Initializing Module 11b: Fitness Pattern Aggregator");
            launchFitnessPatternAggregator(env);

            // Module 13: Clinical State Synchroniser
            LOG.info("Initializing Module 13: Clinical State Synchroniser");
            launchClinicalStateSynchroniser(env);

            LOG.info("All 14 modules initialized successfully - Complete EHR Intelligence Pipeline Ready");

        } catch (Exception e) {
            LOG.error("Failed to initialize complete pipeline", e);
            throw new RuntimeException("Pipeline initialization failed", e);
        }
    }

    /**
     * Launch the Module 13 Clinical State Synchroniser pipeline.
     * Consumes from 7 topics via multi-source union, outputs state change events
     * and KB-20 state projections.
     */
    private static void launchClinicalStateSynchroniser(StreamExecutionEnvironment env) {
        LOG.info("Launching Module 13: Clinical State Synchroniser pipeline (7-source fan-in)");

        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        // All sources use SourceTaggingDeserializer to inject source_module tag.
        // Upstream modules DON'T emit source_module, and Sources 5-6 emit
        // InterventionWindowSignal/InterventionDeltaRecord, not CanonicalEvent.
        // SourceTaggingDeserializer wraps raw JSON → CanonicalEvent with injected tag.

        // Source 1: BP Variability Metrics (Module 7)
        KafkaSource<CanonicalEvent> bpVarSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.FLINK_BP_VARIABILITY_METRICS.getTopicName())
                .setGroupId("flink-module13-bp-variability-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setDeserializer(new SourceTaggingDeserializer("module7", EventType.VITAL_SIGN))
                .build();

        // Source 2: Engagement Signals (Module 9)
        KafkaSource<CanonicalEvent> engagementSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.FLINK_ENGAGEMENT_SIGNALS.getTopicName())
                .setGroupId("flink-module13-engagement-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setDeserializer(new SourceTaggingDeserializer("module9", EventType.PATIENT_REPORTED))
                .build();

        // Source 3: Meal Patterns (Module 10b)
        KafkaSource<CanonicalEvent> mealSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.FLINK_MEAL_PATTERNS.getTopicName())
                .setGroupId("flink-module13-meal-patterns-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setDeserializer(new SourceTaggingDeserializer("module10b", EventType.PATIENT_REPORTED))
                .build();

        // Source 4: Fitness Patterns (Module 11b)
        KafkaSource<CanonicalEvent> fitnessSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.FLINK_FITNESS_PATTERNS.getTopicName())
                .setGroupId("flink-module13-fitness-patterns-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setDeserializer(new SourceTaggingDeserializer("module11b", EventType.DEVICE_READING))
                .build();

        // Source 5: Intervention Window Signals (Module 12) — NOT CanonicalEvent upstream!
        KafkaSource<CanonicalEvent> windowSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.CLINICAL_INTERVENTION_WINDOW_SIGNALS.getTopicName())
                .setGroupId("flink-module13-intervention-windows-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setDeserializer(new SourceTaggingDeserializer("module12", EventType.MEDICATION_ORDERED))
                .build();

        // Source 6: Intervention Deltas (Module 12b) — NOT CanonicalEvent upstream!
        KafkaSource<CanonicalEvent> deltaSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.FLINK_INTERVENTION_DELTAS.getTopicName())
                .setGroupId("flink-module13-intervention-deltas-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setDeserializer(new SourceTaggingDeserializer("module12b", EventType.LAB_RESULT))
                .build();

        // Source 7: Enriched Patient Events (labs, vitals)
        KafkaSource<CanonicalEvent> enrichedSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
                .setGroupId("flink-module13-enriched-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setDeserializer(new SourceTaggingDeserializer("enriched", EventType.LAB_RESULT))
                .build();

        // PIPE-7: Source 8: Comorbidity Alerts (Module 8) — CID state for renal velocity
        KafkaSource<CanonicalEvent> cidSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.ALERTS_COMORBIDITY_INTERACTIONS.getTopicName())
                .setGroupId("flink-module13-comorbidity-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setDeserializer(new SourceTaggingDeserializer("module8", EventType.ADVERSE_EVENT))
                .build();

        // Create streams with watermark strategy
        WatermarkStrategy<CanonicalEvent> watermark = WatermarkStrategy
                .<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                .withTimestampAssigner((e, ts) -> e.getEventTime());

        DataStream<CanonicalEvent> bpVarStream = env.fromSource(bpVarSource, watermark,
                "Kafka Source: BP Variability (Module 13)");
        DataStream<CanonicalEvent> engagementStream = env.fromSource(engagementSource, watermark,
                "Kafka Source: Engagement Signals (Module 13)");
        DataStream<CanonicalEvent> mealStream = env.fromSource(mealSource, watermark,
                "Kafka Source: Meal Patterns (Module 13)");
        DataStream<CanonicalEvent> fitnessStream = env.fromSource(fitnessSource, watermark,
                "Kafka Source: Fitness Patterns (Module 13)");
        DataStream<CanonicalEvent> windowStream = env.fromSource(windowSource, watermark,
                "Kafka Source: Intervention Windows (Module 13)");
        DataStream<CanonicalEvent> deltaStream = env.fromSource(deltaSource, watermark,
                "Kafka Source: Intervention Deltas (Module 13)");
        DataStream<CanonicalEvent> enrichedStream = env.fromSource(enrichedSource, watermark,
                "Kafka Source: Enriched Patient Events (Module 13)");
        DataStream<CanonicalEvent> cidStream = env.fromSource(cidSource, watermark,
                "Kafka Source: Comorbidity Alerts (Module 13)");

        // Union all 8 sources and process
        Module13_ClinicalStateSynchroniser processor = new Module13_ClinicalStateSynchroniser();

        SingleOutputStreamOperator<ClinicalStateChangeEvent> stateChanges = bpVarStream
                .union(engagementStream, mealStream, fitnessStream, windowStream, deltaStream, enrichedStream, cidStream)
                .keyBy(CanonicalEvent::getPatientId)
                .process(processor)
                .uid("module13-clinical-state-synchroniser")
                .name("Module 13: Clinical State Synchroniser");

        // Main sink: State change events → Kafka
        // PIPE-8: Key by patientId so downstream consumers of state-change-events.v1
        // can co-partition with other patient-keyed topics (e.g., KB-20 projections)
        stateChanges.sinkTo(
                KafkaSink.<ClinicalStateChangeEvent>builder()
                        .setBootstrapServers(bootstrap)
                        .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                        .setTransactionalIdPrefix("m13-state-changes")
                        .setRecordSerializer(
                                KafkaRecordSerializationSchema.<ClinicalStateChangeEvent>builder()
                                        .setTopic(KafkaTopics.CLINICAL_STATE_CHANGE_EVENTS.getTopicName())
                                        .setKeySerializationSchema(new PatientIdKeySerializer<>())
                                        .setValueSerializationSchema(new JsonSerializer<ClinicalStateChangeEvent>())
                                        .build())
                        .build()
        ).name("Sink: Clinical State Change Events");

        // Side output sink: KB-20 state updates → async PostgreSQL + Redis
        DataStream<KB20StateUpdate> kb20Updates = stateChanges
                .getSideOutput(Module13_ClinicalStateSynchroniser.KB20_SIDE_OUTPUT);

        String pgUrl = System.getenv().getOrDefault("KB20_POSTGRES_URL", "jdbc:postgresql://localhost:5433/kb20");
        String redisUrl = System.getenv().getOrDefault("KB20_REDIS_URL", "redis://localhost:6380");

        kb20Updates.sinkTo(new KB20AsyncSinkFunction(pgUrl, redisUrl))
                .name("Sink: KB-20 State Projections (PostgreSQL + Redis)");

        LOG.info("Module 13 pipeline configured: 7 sources, 2 sinks (Kafka + KB-20)");
    }
}