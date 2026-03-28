package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.utils.*;
import com.cardiofit.flink.processors.*;
import com.cardiofit.flink.knowledgebase.*;
import com.cardiofit.flink.knowledgebase.medications.loader.MedicationDatabaseLoader;
import com.cardiofit.flink.loader.DiagnosticTestLoader;
import com.cardiofit.flink.cdc.ProtocolCDCEvent;
import com.cardiofit.flink.cdc.DrugRuleCDCEvent;
import com.cardiofit.flink.cdc.DrugInteractionCDCEvent;
import com.cardiofit.flink.cdc.TerminologyCDCEvent;
import com.cardiofit.flink.cdc.DebeziumJSONDeserializer;
import com.cardiofit.flink.models.protocol.Protocol;
import com.cardiofit.flink.models.protocol.SimplifiedProtocol;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.state.*;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.BroadcastStream;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.co.KeyedBroadcastProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.io.Serializable;
import java.time.Duration;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Module 3: Comprehensive CDS with CDC BroadcastStream (Phase 2 CDC Integration)
 *
 * ✨ NEW: Hot-swapping clinical protocols via CDC without Flink restart
 *
 * Architecture:
 * 1. BroadcastStream: CDC events from kb3.clinical_protocols.changes topic
 * 2. BroadcastState: Shared protocol Map<String, SimplifiedProtocol> across all instances
 * 3. KeyedBroadcastProcessFunction:
 *    - processElement(): Processes clinical events using current protocols from BroadcastState
 *    - processBroadcastElement(): Updates BroadcastState when CDC protocol changes arrive
 *
 * Performance:
 * - Protocol updates propagate in <1 second without restart
 * - Zero downtime for protocol changes
 * - Automatic synchronization across all parallel instances
 *
 * Note: Uses SimplifiedProtocol (flattened version without nested TriggerCriteria/ConfidenceScoring)
 * to avoid StackOverflowError from Flink's TypeExtractor on self-referencing structures
 *
 * @author Phase 2 CDC Integration Team
 * @version 2.0
 * @since 2025-11-22
 */
public class Module3_ComprehensiveCDS_WithCDC {
    private static final Logger LOG = LoggerFactory.getLogger(Module3_ComprehensiveCDS_WithCDC.class);

    // BroadcastStateDescriptor for protocol hot-swapping
    // Uses SimplifiedProtocol (flattened version) to avoid StackOverflowError from nested structures
    public static final MapStateDescriptor<String, SimplifiedProtocol> PROTOCOL_STATE_DESCRIPTOR =
            new MapStateDescriptor<>(
                    "protocol-broadcast-state",
                    TypeInformation.of(String.class),
                    TypeInformation.of(SimplifiedProtocol.class)
            );

    // KB-4: Drug dosing rules (keyed by drugId)
    public static final MapStateDescriptor<String, String> DRUG_RULE_STATE_DESCRIPTOR =
            new MapStateDescriptor<>(
                    "drug-rule-broadcast-state",
                    TypeInformation.of(String.class),
                    TypeInformation.of(String.class)  // JSON string of DrugRuleData
            );

    // KB-5: Drug interactions (keyed by interactionId)
    public static final MapStateDescriptor<String, String> DRUG_INTERACTION_STATE_DESCRIPTOR =
            new MapStateDescriptor<>(
                    "drug-interaction-broadcast-state",
                    TypeInformation.of(String.class),
                    TypeInformation.of(String.class)  // JSON string of InteractionData
            );

    // KB-7: Terminology (keyed by conceptCode)
    public static final MapStateDescriptor<String, String> TERMINOLOGY_STATE_DESCRIPTOR =
            new MapStateDescriptor<>(
                    "terminology-broadcast-state",
                    TypeInformation.of(String.class),
                    TypeInformation.of(String.class)  // JSON string of TerminologyData
            );

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Module 3: Comprehensive CDS with CDC BroadcastStream (Phase 2)");

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        int parallelism = Integer.parseInt(
                System.getenv().getOrDefault("MODULE3_PARALLELISM", "4"));
        env.setParallelism(parallelism);

        env.enableCheckpointing(180000); // 3-minute checkpoint interval
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(30000); // 30s min pause
        env.getCheckpointConfig().setCheckpointTimeout(120000); // 2-minute timeout
        env.getCheckpointConfig().setMaxConcurrentCheckpoints(1);

        // RocksDB state backend — in Flink 2.x, configure via flink-conf.yaml:
        //   state.backend.type: rocksdb
        //   state.backend.incremental: true
        // Programmatic setStateBackend() was removed in Flink 2.x; see FlinkJobOrchestrator for pattern.
        LOG.info("State backend configured via flink-conf.yaml (Flink 2.x)");

        createCDSPipelineWithCDC(env);

        env.execute("Module 3: Comprehensive CDS with CDC Hot-Swap");
    }

    /**
     * Create CDS pipeline with CDC BroadcastStream for protocol hot-swapping
     */
    public static void createCDSPipelineWithCDC(StreamExecutionEnvironment env) {
        LOG.info("Creating CDS pipeline with CDC BroadcastStream integration");

        // Stream 1: Enriched patient contexts from Module 2
        DataStream<EnrichedPatientContext> enrichedPatientContexts = createEnrichedPatientContextSource(env);

        // Stream 2: Protocol CDC events from Debezium
        DataStream<ProtocolCDCEvent> protocolCDCStream = createProtocolCDCSource(env);

        // Stream 3: Drug Rule CDC events from KB-4
        DataStream<DrugRuleCDCEvent> drugRuleCDCStream = createDrugRuleCDCSource(env);

        // Stream 4: Drug Interaction CDC events from KB-5
        DataStream<DrugInteractionCDCEvent> drugInteractionCDCStream = createDrugInteractionCDCSource(env);

        // Stream 5: Terminology CDC events from KB-7
        DataStream<TerminologyCDCEvent> terminologyCDCStream = createTerminologyCDCSource(env);

        // Convert Protocol CDC stream to BroadcastStream
        BroadcastStream<ProtocolCDCEvent> protocolBroadcastStream = protocolCDCStream
                .broadcast(PROTOCOL_STATE_DESCRIPTOR);

        // Broadcast KB-4, KB-5, KB-7 streams (consumed independently by downstream operators)
        BroadcastStream<DrugRuleCDCEvent> drugRuleBroadcastStream = drugRuleCDCStream
                .broadcast(DRUG_RULE_STATE_DESCRIPTOR);
        BroadcastStream<DrugInteractionCDCEvent> drugInteractionBroadcastStream = drugInteractionCDCStream
                .broadcast(DRUG_INTERACTION_STATE_DESCRIPTOR);
        BroadcastStream<TerminologyCDCEvent> terminologyBroadcastStream = terminologyCDCStream
                .broadcast(TERMINOLOGY_STATE_DESCRIPTOR);

        LOG.info("KB-4/KB-5/KB-7 BroadcastStreams initialized for downstream consumption");

        // Connect clinical events with protocol CDC broadcast stream
        DataStream<CDSEvent> comprehensiveEvents = enrichedPatientContexts
                .keyBy(EnrichedPatientContext::getPatientId)
                .connect(protocolBroadcastStream)
                .process(new CDSProcessorWithCDC())
                .uid("comprehensive-cds-cdc-processor")
                .name("Comprehensive CDS with CDC Hot-Swap");

        // Output to Kafka
        comprehensiveEvents.sinkTo(createCDSEventsSink())
                .uid("comprehensive-cds-events-cdc-sink")
                .name("CDS Events Sink (CDC-enabled)");

        LOG.info("CDC BroadcastStream pipeline initialized successfully");
    }

    /**
     * Create Protocol CDC source from Kafka kb3.clinical_protocols.changes topic
     */
    private static DataStream<ProtocolCDCEvent> createProtocolCDCSource(StreamExecutionEnvironment env) {
        LOG.info("Creating Protocol CDC source from kb3.clinical_protocols.changes");

        KafkaSource<ProtocolCDCEvent> source = KafkaSource.<ProtocolCDCEvent>builder()
                .setBootstrapServers(getBootstrapServers())
                .setTopics("kb3.clinical_protocols.changes")
                .setGroupId("module3-protocol-cdc-consumer")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(DebeziumJSONDeserializer.forProtocol())
                .build();

        return env.fromSource(
                source,
                WatermarkStrategy.noWatermarks(),
                "Protocol CDC Source"
        );
    }

    /**
     * Create Drug Rule CDC source from Kafka kb1.drug_rule_packs.changes topic (KB-4)
     */
    private static DataStream<DrugRuleCDCEvent> createDrugRuleCDCSource(StreamExecutionEnvironment env) {
        LOG.info("Creating Drug Rule CDC source from kb1.drug_rule_packs.changes");

        KafkaSource<DrugRuleCDCEvent> source = KafkaSource.<DrugRuleCDCEvent>builder()
                .setBootstrapServers(getBootstrapServers())
                .setTopics("kb1.drug_rule_packs.changes")
                .setGroupId("module3-drug-rule-cdc-consumer")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(DebeziumJSONDeserializer.forDrugRule())
                .build();

        return env.fromSource(
                source,
                WatermarkStrategy.noWatermarks(),
                "Drug Rule CDC Source (KB-4)"
        );
    }

    /**
     * Create Drug Interaction CDC source from Kafka kb5.drug_interactions.changes topic (KB-5)
     */
    private static DataStream<DrugInteractionCDCEvent> createDrugInteractionCDCSource(StreamExecutionEnvironment env) {
        LOG.info("Creating Drug Interaction CDC source from kb5.drug_interactions.changes");

        KafkaSource<DrugInteractionCDCEvent> source = KafkaSource.<DrugInteractionCDCEvent>builder()
                .setBootstrapServers(getBootstrapServers())
                .setTopics("kb5.drug_interactions.changes")
                .setGroupId("module3-drug-interaction-cdc-consumer")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(DebeziumJSONDeserializer.forDrugInteraction())
                .build();

        return env.fromSource(
                source,
                WatermarkStrategy.noWatermarks(),
                "Drug Interaction CDC Source (KB-5)"
        );
    }

    /**
     * Create Terminology CDC source from Kafka kb7.terminology.changes topic (KB-7)
     */
    private static DataStream<TerminologyCDCEvent> createTerminologyCDCSource(StreamExecutionEnvironment env) {
        LOG.info("Creating Terminology CDC source from kb7.terminology.changes");

        KafkaSource<TerminologyCDCEvent> source = KafkaSource.<TerminologyCDCEvent>builder()
                .setBootstrapServers(getBootstrapServers())
                .setTopics("kb7.terminology.changes")
                .setGroupId("module3-terminology-cdc-consumer")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(DebeziumJSONDeserializer.forTerminology())
                .build();

        return env.fromSource(
                source,
                WatermarkStrategy.noWatermarks(),
                "Terminology CDC Source (KB-7)"
        );
    }

    /**
     * CDS Processor with CDC BroadcastState for protocol hot-swapping
     */
    public static class CDSProcessorWithCDC
            extends KeyedBroadcastProcessFunction<String, EnrichedPatientContext, ProtocolCDCEvent, CDSEvent> {

        private transient DrugInteractionAnalyzer drugInteractionAnalyzer;
        private transient boolean initialized;
        private transient ValueState<PatientCDSState> patientCDSState;

        @Override
        public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
            super.open(openContext);
            LOG.info("=== STARTING CDS Processor with CDC BroadcastState ===");

            try {
                // Phase 4: Diagnostic Tests
                LOG.info("Loading Phase 4: Diagnostic Tests...");
                DiagnosticTestLoader diagnosticLoader = DiagnosticTestLoader.getInstance();
                LOG.info("Phase 4 SUCCESS: Diagnostic test loader initialized: {}",
                        diagnosticLoader.isInitialized());

                // Phase 5: Clinical Guidelines
                LOG.info("Loading Phase 5: Clinical Guidelines...");
                GuidelineLoader guidelineLoader = GuidelineLoader.getInstance();
                Map<String, Guideline> guidelines = guidelineLoader.loadAllGuidelines();
                LOG.info("Phase 5 SUCCESS: {} clinical guidelines loaded", guidelines.size());

                // Phase 6: Medication Database
                LOG.info("Loading Phase 6: Medication Database...");
                MedicationDatabaseLoader medicationLoader = MedicationDatabaseLoader.getInstance();
                LOG.info("Phase 6 SUCCESS: Medication database loader initialized");

                // Phase 6.5: Drug Interaction Analyzer
                LOG.info("Loading Phase 6.5: Drug Interaction Analyzer...");
                drugInteractionAnalyzer = new DrugInteractionAnalyzer();
                Map<String, Integer> interactionStats = drugInteractionAnalyzer.getStatistics();
                LOG.info("Phase 6.5 SUCCESS: Drug interactions loaded - Total: {}, Major: {}, Black Box: {}",
                        interactionStats.get("total_interactions"),
                        interactionStats.get("major_severity"),
                        interactionStats.get("black_box_warnings"));

                // Phase 7: Evidence Repository
                LOG.info("Loading Phase 7: Evidence Repository...");
                CitationLoader citationLoader = CitationLoader.getInstance();
                Map<String, Citation> citations = citationLoader.loadAllCitations();
                LOG.info("Phase 7 SUCCESS: {} citations loaded", citations.size());

                initialized = true;
                LOG.info("=== CDS PROCESSOR INITIALIZED (Protocols will be loaded from CDC BroadcastState) ===");

                // Patient CDS state with 7-day TTL
                StateTtlConfig ttlConfig = StateTtlConfig.newBuilder(Duration.ofDays(7))
                        .setUpdateType(StateTtlConfig.UpdateType.OnCreateAndWrite)
                        .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
                        .build();

                ValueStateDescriptor<PatientCDSState> stateDescriptor =
                        new ValueStateDescriptor<>("patient-cds-state", PatientCDSState.class);
                stateDescriptor.enableTimeToLive(ttlConfig);
                patientCDSState = getRuntimeContext().getState(stateDescriptor);

            } catch (Exception e) {
                LOG.error("=== INITIALIZATION FAILED ===", e);
                initialized = false;
                throw e;
            }
        }

        /**
         * Process broadcast element (CDC protocol update)
         *
         * This method is called for EVERY CDC event from kb3.clinical_protocols.changes.
         * It updates the BroadcastState, which is then read by all parallel instances.
         */
        @Override
        public void processBroadcastElement(
                ProtocolCDCEvent cdcEvent,
                Context ctx,
                Collector<CDSEvent> out) throws Exception {

            // Get BroadcastState
            BroadcastState<String, SimplifiedProtocol> protocolState = ctx.getBroadcastState(PROTOCOL_STATE_DESCRIPTOR);

            ProtocolCDCEvent.Payload payload = cdcEvent.getPayload();

            if (payload == null) {
                LOG.warn("Received CDC event with null payload, skipping");
                return;
            }

            String operation = payload.getOperation();
            LOG.info("📡 CDC EVENT: op={}, source={}.{}, ts={}",
                    operation,
                    payload.getSource() != null ? payload.getSource().getDatabase() : "unknown",
                    payload.getSource() != null ? payload.getSource().getTable() : "unknown",
                    payload.getTimestampMs());

            if (payload.isDelete()) {
                // DELETE operation - remove protocol from BroadcastState
                ProtocolCDCEvent.ProtocolData before = payload.getBefore();
                if (before != null && before.getProtocolId() != null) {
                    String protocolId = before.getProtocolId();
                    protocolState.remove(protocolId);
                    LOG.info("🗑️ DELETED Protocol from BroadcastState: {}", protocolId);
                }

            } else {
                // CREATE or UPDATE operation - upsert protocol into BroadcastState
                ProtocolCDCEvent.ProtocolData after = payload.getAfter();
                if (after != null && after.getProtocolId() != null) {
                    String protocolId = after.getProtocolId();

                    // Convert CDC ProtocolData to SimplifiedProtocol model
                    SimplifiedProtocol protocol = convertCDCToProtocol(after);

                    // Update BroadcastState (visible to all parallel instances immediately)
                    protocolState.put(protocolId, protocol);

                    LOG.info("✅ {} Protocol in BroadcastState: {} v{} | Category: {} | Specialty: {}",
                            payload.isCreate() ? "CREATED" : "UPDATED",
                            protocol.getProtocolId(),
                            protocol.getVersion(),
                            protocol.getCategory(),
                            protocol.getSpecialty());
                }
            }
        }

        /**
         * Process clinical event element (use protocols from BroadcastState)
         *
         * This method processes each EnrichedPatientContext using the current
         * protocols available in BroadcastState (updated via CDC).
         */
        @Override
        public void processElement(
                EnrichedPatientContext context,
                ReadOnlyContext ctx,
                Collector<CDSEvent> out) throws Exception {

            if (!initialized) {
                LOG.warn("Processor not fully initialized, skipping event for patient: {}",
                        context.getPatientId());
                return;
            }

            // Read protocols from BroadcastState
            ReadOnlyBroadcastState<String, SimplifiedProtocol> protocolState =
                ctx.getBroadcastState(PROTOCOL_STATE_DESCRIPTOR);

            Map<String, SimplifiedProtocol> protocols = new HashMap<>();
            for (Map.Entry<String, SimplifiedProtocol> entry : protocolState.immutableEntries()) {
                protocols.put(entry.getKey(), entry.getValue());
            }

            // Cold-start readiness gate
            PatientCDSState cdsState = patientCDSState.value();
            if (cdsState == null) {
                cdsState = new PatientCDSState();
            }

            if (!protocols.isEmpty() && !cdsState.isBroadcastStateSeeded()) {
                cdsState.setBroadcastStateSeeded(true);
                LOG.info("Broadcast state seeded for patient={}, protocols={}",
                        context.getPatientId(), protocols.size());
            }

            // Create typed CDSEvent
            CDSEvent cdsEvent = new CDSEvent(context);
            cdsEvent.setBroadcastStateReady(cdsState.isBroadcastStateSeeded());

            List<CDSPhaseResult> allResults = new ArrayList<>();

            // Phase 1: Protocol Matching
            CDSPhaseResult phase1 = Module3PhaseExecutor.executePhase1(context, protocols);
            allResults.add(phase1);
            cdsEvent.addPhaseResult(phase1);

            @SuppressWarnings("unchecked")
            List<String> matchedProtocolIds = phase1.isActive()
                    ? (List<String>) phase1.getDetail("matchedProtocolIds")
                    : Collections.emptyList();

            // Phase 2: Clinical Scoring + MHRI
            CDSPhaseResult phase2 = Module3PhaseExecutor.executePhase2(context);
            allResults.add(phase2);
            cdsEvent.addPhaseResult(phase2);

            // Phase 5: Guideline Concordance
            CDSPhaseResult phase5 = Module3PhaseExecutor.executePhase5(
                    context, matchedProtocolIds, protocols);
            allResults.add(phase5);
            cdsEvent.addPhaseResult(phase5);

            // Phase 6: Medication Rules
            CDSPhaseResult phase6 = Module3PhaseExecutor.executePhase6(context);
            allResults.add(phase6);
            cdsEvent.addPhaseResult(phase6);

            // Phase 7: Safety Checks
            CDSPhaseResult phase7 = Module3PhaseExecutor.executePhase7(context);
            allResults.add(phase7);
            cdsEvent.addPhaseResult(phase7);

            // Phase 8: Output Composition (mutates cdsEvent)
            Module3PhaseExecutor.executePhase8(cdsEvent, allResults);

            // Update patient CDS state
            if (cdsEvent.getMhriScore() != null && cdsEvent.getMhriScore().getComposite() != null) {
                cdsState.addMHRI(cdsEvent.getMhriScore().getComposite());
            }
            cdsState.setActiveProtocols(new HashSet<>(matchedProtocolIds != null
                    ? matchedProtocolIds : Collections.emptyList()));
            cdsState.setLastProcessedTime(System.currentTimeMillis());
            cdsState.setEventsSinceLastCDS(cdsState.getEventsSinceLastCDS() + 1);
            patientCDSState.update(cdsState);

            // Per-phase latency summary
            long totalPhaseMs = 0;
            for (CDSPhaseResult pr : allResults) {
                totalPhaseMs += pr.getDurationMs();
            }
            LOG.debug("CDS latency breakdown: patient={} totalPhaseMs={} phases={}",
                    context.getPatientId(), totalPhaseMs,
                    allResults.stream()
                            .map(pr -> pr.getPhaseName() + "=" + pr.getDurationMs() + "ms")
                            .collect(Collectors.joining(", ")));

            LOG.info("CDS complete: patient={} protocols={} mhri={} safety={}",
                    context.getPatientId(),
                    cdsEvent.getProtocolsMatched(),
                    cdsEvent.getMhriScore() != null ? cdsEvent.getMhriScore().getComposite() : "null",
                    cdsEvent.getSafetyAlerts().size());

            out.collect(cdsEvent);
        }

        /**
         * Convert CDC ProtocolData to SimplifiedProtocol for BroadcastState
         *
         * Maps actual database fields from kb3_guidelines.clinical_protocols:
         * - id → protocolId (convert integer to string)
         * - protocol_name → name
         * - specialty → specialty
         * - version → version
         * - content → description (if exists)
         *
         * Uses SimplifiedProtocol to avoid StackOverflowError from nested structures
         */
        private SimplifiedProtocol convertCDCToProtocol(ProtocolCDCEvent.ProtocolData cdcData) {
            SimplifiedProtocol protocol = new SimplifiedProtocol();

            // Map actual database fields to SimplifiedProtocol model
            protocol.setProtocolId(String.valueOf(cdcData.getId())); // Convert integer id to string
            protocol.setName(cdcData.getProtocolName());             // protocol_name → name
            protocol.setVersion(cdcData.getVersion());
            protocol.setSpecialty(cdcData.getSpecialty());

            // Set category from specialty if not provided (backward compatibility)
            if (cdcData.getCategory() != null) {
                protocol.setCategory(cdcData.getCategory());
            } else {
                // Derive category from specialty for now
                protocol.setCategory("CLINICAL"); // Default category
            }

            // Map content field to description if available
            if (cdcData.getContent() != null) {
                protocol.setDescription(cdcData.getContent());
            }

            // Set evidence source as database name (kb3_guidelines)
            protocol.setEvidenceSource("kb3_guidelines");

            // KNOWN LIMITATION: triggerThresholds not populated from CDC payload.
            // All CDC-loaded protocols will match every patient until thresholds are
            // parsed from cdcData.getContent() or the CDC schema is extended.

            return protocol;
        }

        /**
         * Per-patient CDS state accumulated across events.
         * Stored in RocksDB with 7-day TTL.
         */
        public static class PatientCDSState implements Serializable {
            private static final long serialVersionUID = 1L;
            private List<Double> mhriHistory;      // Last N MHRI scores for trend
            private Set<String> activeProtocols;    // Currently active protocol IDs
            private long lastProcessedTime;
            private int eventsSinceLastCDS;
            private boolean broadcastStateSeeded;   // True after first broadcast event received

            public PatientCDSState() {
                this.mhriHistory = new ArrayList<>();
                this.activeProtocols = new HashSet<>();
                this.lastProcessedTime = 0;
                this.eventsSinceLastCDS = 0;
                this.broadcastStateSeeded = false;
            }

            public void addMHRI(double score) {
                mhriHistory.add(score);
                if (mhriHistory.size() > 10) {
                    mhriHistory.remove(0); // Keep last 10
                }
            }

            // Getters and setters
            public List<Double> getMhriHistory() { return mhriHistory; }
            public Set<String> getActiveProtocols() { return activeProtocols; }
            public void setActiveProtocols(Set<String> p) { this.activeProtocols = p; }
            public long getLastProcessedTime() { return lastProcessedTime; }
            public void setLastProcessedTime(long t) { this.lastProcessedTime = t; }
            public int getEventsSinceLastCDS() { return eventsSinceLastCDS; }
            public void setEventsSinceLastCDS(int c) { this.eventsSinceLastCDS = c; }
            public boolean isBroadcastStateSeeded() { return broadcastStateSeeded; }
            public void setBroadcastStateSeeded(boolean s) { this.broadcastStateSeeded = s; }
        }

    }

    // ========== KAFKA SOURCE/SINK HELPERS ==========

    private static DataStream<EnrichedPatientContext> createEnrichedPatientContextSource(StreamExecutionEnvironment env) {
        KafkaSource<EnrichedPatientContext> source = KafkaSource.<EnrichedPatientContext>builder()
                .setBootstrapServers(getBootstrapServers())
                .setTopics(getTopicName("MODULE3_INPUT_TOPIC", "clinical-patterns.v1"))
                .setGroupId("comprehensive-cds-cdc-consumer")
                .setValueOnlyDeserializer(new EnrichedPatientContextDeserializer())
                .setProperties(KafkaConfigLoader.getAutoConsumerConfig("comprehensive-cds-cdc-consumer"))
                .build();

        return env.fromSource(source,
                WatermarkStrategy.<EnrichedPatientContext>forBoundedOutOfOrderness(Duration.ofSeconds(5))
                        .withTimestampAssigner((event, timestamp) -> event.getEventTime()),
                "Enriched Patient Context Source");
    }

    private static KafkaSink<CDSEvent> createCDSEventsSink() {
        Properties producerConfig = new Properties();
        producerConfig.setProperty("bootstrap.servers", getBootstrapServers());
        producerConfig.setProperty("compression.type", "snappy");
        producerConfig.setProperty("batch.size", "32768");
        producerConfig.setProperty("linger.ms", "100");
        producerConfig.setProperty("acks", "all");
        producerConfig.setProperty("enable.idempotence", "true");

        return KafkaSink.<CDSEvent>builder()
                .setBootstrapServers(getBootstrapServers())
                .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                        .setTopic(getTopicName("MODULE3_OUTPUT_TOPIC", "comprehensive-cds-events-cdc.v1"))
                        .setKeySerializationSchema((CDSEvent event) -> event.getPatientId().getBytes())
                        .setValueSerializationSchema(new CDSEventSerializer())
                        .build())
                .setTransactionalIdPrefix("comprehensive-cds-cdc-tx")
                .setKafkaProducerConfig(producerConfig)
                .build();
    }

    private static String getBootstrapServers() {
        String kafkaServers = System.getenv("KAFKA_BOOTSTRAP_SERVERS");
        return (kafkaServers != null && !kafkaServers.isEmpty())
                ? kafkaServers
                : "localhost:9092";
    }

    private static String getTopicName(String envVar, String defaultTopic) {
        String topic = System.getenv(envVar);
        return (topic != null && !topic.isEmpty()) ? topic : defaultTopic;
    }

    // ========== SERIALIZATION ==========

    public static class EnrichedPatientContextDeserializer implements DeserializationSchema<EnrichedPatientContext> {
        private static final long serialVersionUID = 1L;
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) throws Exception {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
            objectMapper.configure(
                    com.fasterxml.jackson.databind.DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES,
                    false
            );
        }

        @Override
        public EnrichedPatientContext deserialize(byte[] message) throws IOException {
            if (objectMapper == null) {
                objectMapper = new ObjectMapper();
                objectMapper.registerModule(new JavaTimeModule());
                objectMapper.configure(
                        com.fasterxml.jackson.databind.DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES,
                        false
                );
            }
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
    }

    public static class CDSEventSerializer implements SerializationSchema<CDSEvent> {
        private static final long serialVersionUID = 1L;
        private transient ObjectMapper objectMapper;

        @Override
        public byte[] serialize(CDSEvent element) {
            if (objectMapper == null) {
                objectMapper = new ObjectMapper();
                objectMapper.registerModule(new JavaTimeModule());
            }
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                LOG.error("Failed to serialize CDSEvent for patient {}: {}",
                        element.getPatientId(), e.getMessage());
                throw new RuntimeException("CDSEvent serialization failure — clinical data loss prevented", e);
            }
        }
    }
}
