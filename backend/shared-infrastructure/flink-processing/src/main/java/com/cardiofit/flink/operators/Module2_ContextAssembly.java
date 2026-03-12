
package com.cardiofit.flink.operators;

import com.cardiofit.flink.clients.GoogleFHIRClient;
import com.cardiofit.flink.clients.Neo4jGraphClient;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.state.*;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.api.java.tuple.Tuple2;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.AsyncDataStream;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.streaming.api.functions.windowing.WindowFunction;
import org.apache.flink.streaming.api.windowing.assigners.TumblingEventTimeWindows;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

/**
 * Module 2: Context Assembly & Enrichment
 *
 * Responsibilities:
 * - Maintain per-patient context using keyed state
 * - Enrich events with historical context and trends
 * - Calculate derived metrics and clinical indicators
 * - Manage patient session windows and context snapshots
 * - Route enriched events for downstream processing
 */
public class Module2_ContextAssembly {
    private static final Logger LOG = LoggerFactory.getLogger(Module2_ContextAssembly.class);

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Module 2: Context Assembly & Enrichment");

        // Set up execution environment
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Configure for stateful processing
        env.setParallelism(2);
        env.enableCheckpointing(30000);
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);
        env.getCheckpointConfig().setCheckpointTimeout(600000);

        // Create context assembly pipeline
        createContextAssemblyPipeline(env);

        // Execute the job
        env.execute("Module 2: Context Assembly & Enrichment");
    }

    public static void createContextAssemblyPipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating context assembly pipeline for clinical events (ASYNC I/O MODE)");

        // Consume enriched events from Module 1
        DataStream<CanonicalEvent> canonicalEvents = createCanonicalEventSource(env);

        // ========== ASYNC I/O ENRICHMENT (NON-BLOCKING) ==========
        // Apply AsyncDataStream for non-blocking patient enrichment
        // AsyncPatientEnricher will create its own FHIR/Neo4j clients on TaskManager to avoid serialization issues
        // Capacity = (1000 events/sec × 0.1 first-time rate × 2s latency) × 1.5 safety = 300
        DataStream<AsyncPatientEnricher.EnrichedEventWithSnapshot> enrichedWithSnapshots =
            AsyncDataStream.unorderedWait(
                canonicalEvents,
                new AsyncPatientEnricher(),  // No-arg constructor - clients created in open() on TaskManager
                5000,                   // timeout in milliseconds (5s for FHIR API + Neo4j dual lookup)
                TimeUnit.MILLISECONDS,
                300                     // capacity (max concurrent async requests, increased for longer timeout)
            ).uid("Async Patient Enrichment");

        // Key by patient ID for stateful processing
        SingleOutputStreamOperator<EnrichedEvent> enrichedEvents = enrichedWithSnapshots
            .keyBy(AsyncPatientEnricher.EnrichedEventWithSnapshot::getPatientId)
            .process(new PatientContextProcessorAsync())
            .uid("Patient Context Assembly");

        // Create patient context snapshots using time windows
        DataStream<PatientContext> contextSnapshots = enrichedEvents
            .keyBy(EnrichedEvent::getPatientId)
            .window(TumblingEventTimeWindows.of(Duration.ofMinutes(15)))
            .apply(new PatientContextSnapshotFunction())
            .uid("Patient Context Snapshots");

        // Send enriched events to downstream processing
        enrichedEvents
            .sinkTo(createEnrichedEventsSink())
            .uid("Enriched Events Sink");

        // Send context snapshots for semantic mesh
        contextSnapshots
            .sinkTo(createContextSnapshotsSink())
            .uid("Context Snapshots Sink");

        LOG.info("Context assembly pipeline created successfully");
    }

    /**
     * Create Kafka source for canonical events from Module 1
     */
    private static DataStream<CanonicalEvent> createCanonicalEventSource(StreamExecutionEnvironment env) {
        KafkaSource<CanonicalEvent> source = KafkaSource.<CanonicalEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
            .setGroupId("context-assembly")
            // REMOVED: .setStartingOffsets() - causes ClassCastException, use auto.offset.reset from consumer config
            .setValueOnlyDeserializer(new CanonicalEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("context-assembly"))
            .build();

        return env.fromSource(source,
            WatermarkStrategy
                .<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                .withTimestampAssigner((event, timestamp) -> event.getEventTime()),
            "Canonical Events Source");
    }

    /**
     * Patient context processor that maintains state per patient.
     *
     * ENHANCED with Google FHIR API and Neo4j integration per Module 2 architecture specification.
     *
     * Key Features:
     * - PatientSnapshot state with 7-day TTL (readmission correlation)
     * - Async lookups to Google Cloud Healthcare FHIR API (500ms timeout)
     * - Async lookups to Neo4j graph database for care networks
     * - First-time patient detection (404 from FHIR = new patient)
     * - Progressive enrichment pattern (state evolves with each event)
     * - State flush to FHIR store on encounter closure
     */
    public static class PatientContextProcessor
            extends KeyedProcessFunction<String, CanonicalEvent, EnrichedEvent> {

        // PRIMARY STATE: PatientSnapshot with 7-day TTL (per architecture spec)
        private transient ValueState<PatientSnapshot> patientSnapshotState;

        // LEGACY STATE: Keep existing PatientContext for backward compatibility
        private transient ValueState<PatientContext> patientContextState;

        // Recent events buffer for trend calculation
        private transient ListState<CanonicalEvent> recentEventsState;

        // Medication state for drug interaction checking
        private transient MapState<String, MedicationState> activeMedicationsState;

        // Vital signs state for trend analysis
        private transient MapState<String, VitalSignTrend> vitalTrendsState;

        // EXTERNAL CLIENTS: Google FHIR API and Neo4j (per architecture spec)
        private transient com.cardiofit.flink.clients.GoogleFHIRClient fhirClient;
        private transient com.cardiofit.flink.clients.Neo4jGraphClient neo4jClient;

        // JSON mapper for serialization
        private transient ObjectMapper objectMapper;

        // @Override - Removed for Flink 2.x
        public void open(org.apache.flink.configuration.Configuration parameters) throws Exception {
            LOG.info("Opening PatientContextProcessor with Google FHIR API integration");

            // ========== PRIMARY STATE: PatientSnapshot with 7-day TTL ==========
            // Per architecture spec: 7-day TTL for readmission correlation
            StateTtlConfig ttlConfig = StateTtlConfig
                .newBuilder(java.time.Duration.ofDays(7))
                .setUpdateType(StateTtlConfig.UpdateType.OnCreateAndWrite)
                .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
                .build();

            ValueStateDescriptor<PatientSnapshot> snapshotDescriptor =
                new ValueStateDescriptor<>("patient-snapshot",
                    com.cardiofit.flink.models.PatientSnapshot.class);
            snapshotDescriptor.enableTimeToLive(ttlConfig);

            patientSnapshotState = getRuntimeContext().getState(snapshotDescriptor);
            LOG.info("Configured PatientSnapshot state with 7-day TTL");

            // ========== LEGACY STATE: Keep for backward compatibility ==========
            patientContextState = getRuntimeContext().getState(
                new ValueStateDescriptor<>("patient-context", PatientContext.class));

            recentEventsState = getRuntimeContext().getListState(
                new ListStateDescriptor<>("recent-events", CanonicalEvent.class));

            activeMedicationsState = getRuntimeContext().getMapState(
                new MapStateDescriptor<>("active-medications", String.class, MedicationState.class));

            vitalTrendsState = getRuntimeContext().getMapState(
                new MapStateDescriptor<>("vital-trends", String.class, VitalSignTrend.class));

            // ========== EXTERNAL CLIENTS: Google FHIR API + Neo4j ==========
            try {
                // Initialize Google FHIR client
                String credentialsPath = KafkaConfigLoader.getGoogleCloudCredentialsPath();
                LOG.info("Loading Google Cloud credentials from: {}", credentialsPath);

                fhirClient = new com.cardiofit.flink.clients.GoogleFHIRClient(
                    KafkaConfigLoader.getGoogleCloudProjectId(),
                    KafkaConfigLoader.getGoogleCloudLocation(),
                    KafkaConfigLoader.getGoogleCloudDatasetId(),
                    KafkaConfigLoader.getGoogleCloudFhirStoreId(),
                    credentialsPath
                );
                fhirClient.initialize();
                LOG.info("GoogleFHIRClient initialized successfully");

                // Initialize Neo4j client (optional - graceful degradation if not configured)
                try {
                    neo4jClient = new com.cardiofit.flink.clients.Neo4jGraphClient(
                        KafkaConfigLoader.getNeo4jUri(),
                        KafkaConfigLoader.getNeo4jUsername(),
                        KafkaConfigLoader.getNeo4jPassword()
                    );
                    neo4jClient.initialize();
                    LOG.info("Neo4jGraphClient initialized successfully");
                } catch (Exception e) {
                    LOG.warn("Neo4j client initialization failed - will continue without graph data: {}",
                        e.getMessage());
                    neo4jClient = null; // Graceful degradation
                }

            } catch (Exception e) {
                LOG.error("Failed to initialize external clients", e);
                throw e; // Fail fast - FHIR client is required
            }

            // Initialize JSON mapper
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());

            LOG.info("PatientContextProcessor initialization complete");
        }

        @Override
        public void processElement(CanonicalEvent event, Context ctx, Collector<EnrichedEvent> out)
                throws Exception {

            String patientId = event.getPatientId();

            try {
                // ========== FIRST-TIME PATIENT DETECTION (per architecture spec) ==========
                com.cardiofit.flink.models.PatientSnapshot snapshot = patientSnapshotState.value();

                if (snapshot == null) {
                    LOG.info("First-time patient detected: {} - performing async lookups", patientId);
                    snapshot = handleFirstTimePatient(patientId, event);
                    patientSnapshotState.update(snapshot);
                    LOG.info("Patient snapshot initialized for: {} (isNew={})",
                        patientId, snapshot.isNewPatient());
                }

                // ========== PROGRESSIVE ENRICHMENT (per architecture spec) ==========
                // Update snapshot with new event data
                snapshot.updateWithEvent(event);
                patientSnapshotState.update(snapshot);

                // ========== CREATE ENRICHED EVENT ==========
                // Convert PatientSnapshot to EnrichedEvent with patient context
                EnrichedEvent enrichedEvent = createEnrichedEventFromSnapshot(event, snapshot);

                // ========== LEGACY STATE UPDATE (backward compatibility) ==========
                PatientContext context = patientContextState.value();
                if (context == null) {
                    context = createNewPatientContext(event.getPatientId(), ctx.getCurrentKey());
                }
                context = updatePatientContext(context, event, ctx);
                patientContextState.update(context);

                // Update legacy state
                updateRecentEvents(event);
                updateMedicationState(event, ctx);
                updateVitalTrends(event, ctx);

                // ========== ENCOUNTER CLOSURE CHECK (per architecture spec) ==========
                if (isEncounterClosure(event)) {
                    LOG.info("Encounter closure detected for patient: {}", patientId);
                    flushStateToExternalSystems(snapshot);
                }

                // ========== EMIT ENRICHED EVENT ==========
                out.collect(enrichedEvent);

                LOG.debug("Successfully enriched event for patient {}: stateVersion={}, isNew={}",
                    patientId, snapshot.getStateVersion(), snapshot.isNewPatient());

            } catch (Exception e) {
                LOG.error("Failed to process event {} for patient {}: {}",
                    event.getId(), patientId, e.getMessage(), e);
                // TODO: Could emit to DLQ here if needed
            }
        }

        /**
         * Handle first-time patient with async lookups to FHIR and Neo4j.
         *
         * Per architecture specification (C01_10):
         * - Async lookups to Google FHIR API and Neo4j with 500ms timeout
         * - If 404 from FHIR → truly new patient, initialize empty state
         * - If data found → hydrate from history
         * - If timeout → initialize empty state, log for async hydration
         */
        private com.cardiofit.flink.models.PatientSnapshot handleFirstTimePatient(
                String patientId, CanonicalEvent event) {

            LOG.info("Performing async lookups for patient: {}", patientId);

            try {
                // ========== PARALLEL ASYNC LOOKUPS (500ms timeout) ==========
                CompletableFuture<com.cardiofit.flink.models.FHIRPatientData> fhirPatientFuture =
                    fhirClient.getPatientAsync(patientId);

                CompletableFuture<List<com.cardiofit.flink.models.Condition>> conditionsFuture =
                    fhirClient.getConditionsAsync(patientId);

                CompletableFuture<List<com.cardiofit.flink.models.Medication>> medicationsFuture =
                    fhirClient.getMedicationsAsync(patientId);

                CompletableFuture<com.cardiofit.flink.models.GraphData> neo4jFuture =
                    neo4jClient != null
                        ? neo4jClient.queryGraphAsync(patientId)
                        : CompletableFuture.completedFuture(new com.cardiofit.flink.models.GraphData());

                // Wait for all lookups with 500ms timeout
                CompletableFuture.allOf(fhirPatientFuture, conditionsFuture, medicationsFuture, neo4jFuture)
                    .get(500, java.util.concurrent.TimeUnit.MILLISECONDS);

                // ========== GET RESULTS ==========
                com.cardiofit.flink.models.FHIRPatientData fhirPatient = fhirPatientFuture.get();
                List<com.cardiofit.flink.models.Condition> conditions = conditionsFuture.get();
                List<com.cardiofit.flink.models.Medication> medications = medicationsFuture.get();
                com.cardiofit.flink.models.GraphData graphData = neo4jFuture.get();

                // ========== INITIALIZE OR HYDRATE STATE ==========
                if (fhirPatient == null) {
                    // 404 from FHIR API → truly new patient
                    LOG.info("Patient {} not found in FHIR store (404) - initializing empty state", patientId);
                    return com.cardiofit.flink.models.PatientSnapshot.createEmpty(patientId);
                } else {
                    // Existing patient → hydrate from history
                    LOG.info("Patient {} found in FHIR store - hydrating from history (conditions={}, meds={})",
                        patientId, conditions.size(), medications.size());
                    return com.cardiofit.flink.models.PatientSnapshot.hydrateFromHistory(
                        patientId, fhirPatient, conditions, medications, graphData);
                }

            } catch (java.util.concurrent.TimeoutException e) {
                // Timeout → initialize empty state, continue processing
                LOG.warn("Timeout (500ms) fetching patient {} from external systems - initializing empty state",
                    patientId);
                return com.cardiofit.flink.models.PatientSnapshot.createEmpty(patientId);

            } catch (java.util.concurrent.ExecutionException e) {
                // Check if it's a 404 error
                if (e.getCause() != null && e.getCause().getMessage() != null
                    && e.getCause().getMessage().contains("404")) {
                    LOG.info("Patient {} returned 404 from FHIR API - new patient", patientId);
                    return com.cardiofit.flink.models.PatientSnapshot.createEmpty(patientId);
                }

                LOG.error("Error fetching patient {} from external systems", patientId, e);
                return com.cardiofit.flink.models.PatientSnapshot.createEmpty(patientId);

            } catch (Exception e) {
                LOG.error("Unexpected error fetching patient {}", patientId, e);
                return com.cardiofit.flink.models.PatientSnapshot.createEmpty(patientId);
            }
        }

        /**
         * Create EnrichedEvent from PatientSnapshot and CanonicalEvent.
         */
        private EnrichedEvent createEnrichedEventFromSnapshot(
                CanonicalEvent event, com.cardiofit.flink.models.PatientSnapshot snapshot) {

            // Use PatientSnapshot's built-in method to create enriched event
            // This method populates PatientContext with all the enriched data
            EnrichedEvent enriched = EnrichedEvent.builder()
                .id(event.getId())
                .patientId(event.getPatientId())
                .encounterId(event.getEncounterId())
                .eventType(event.getEventType())
                .eventTime(event.getEventTime())
                .processingTime(System.currentTimeMillis())
                .sourceSystem(event.getSourceSystem())
                .payload(event.getPayload())
                .build();

            // Create PatientContext from snapshot
            PatientContext context = convertSnapshotToContext(snapshot);
            enriched.setPatientContext(context);

            // Add comprehensive enrichment metadata with FHIR and Neo4j data
            Map<String, Object> enrichmentData = new HashMap<>();

            // State metadata
            enrichmentData.put("state_version", snapshot.getStateVersion());
            enrichmentData.put("was_new_patient", snapshot.isNewPatient());

            // Risk scores
            if (snapshot.getSepsisScore() != null) {
                enrichmentData.put("sepsis_score", snapshot.getSepsisScore());
            }
            if (snapshot.getDeteriorationScore() != null) {
                enrichmentData.put("deterioration_score", snapshot.getDeteriorationScore());
            }
            if (snapshot.getReadmissionRisk() != null) {
                enrichmentData.put("readmission_risk", snapshot.getReadmissionRisk());
            }

            // FHIR demographics
            if (snapshot.getFirstName() != null || snapshot.getLastName() != null) {
                Map<String, Object> demographics = new HashMap<>();
                demographics.put("firstName", snapshot.getFirstName());
                demographics.put("lastName", snapshot.getLastName());
                demographics.put("dateOfBirth", snapshot.getDateOfBirth());
                demographics.put("age", snapshot.getAge());
                demographics.put("gender", snapshot.getGender());
                demographics.put("mrn", snapshot.getMrn());
                enrichmentData.put("fhir_demographics", demographics);
            }

            // FHIR clinical data - always include even if empty to show data was fetched
            enrichmentData.put("fhir_conditions", snapshot.getActiveConditions() != null ?
                snapshot.getActiveConditions() : new ArrayList<>());
            enrichmentData.put("fhir_medications", snapshot.getActiveMedications() != null ?
                snapshot.getActiveMedications() : new ArrayList<>());
            enrichmentData.put("fhir_allergies", snapshot.getAllergies() != null ?
                snapshot.getAllergies() : new ArrayList<>());

            // Neo4j graph data
            if (snapshot.getCareTeam() != null && !snapshot.getCareTeam().isEmpty()) {
                enrichmentData.put("neo4j_care_team", snapshot.getCareTeam());
            }
            if (snapshot.getRiskCohorts() != null && !snapshot.getRiskCohorts().isEmpty()) {
                enrichmentData.put("neo4j_risk_cohorts", snapshot.getRiskCohorts());
            }

            enriched.setEnrichmentData(enrichmentData);

            LOG.debug("Created enriched event for patient {} with {} enrichment fields",
                snapshot.getPatientId(), enrichmentData.size());

            return enriched;
        }

        /**
         * Convert PatientSnapshot to PatientContext for output.
         */
        private PatientContext convertSnapshotToContext(com.cardiofit.flink.models.PatientSnapshot snapshot) {
            PatientContext context = new PatientContext();
            context.setPatientId(snapshot.getPatientId());
            context.setFirstEventTime(snapshot.getFirstSeen());
            context.setLastEventTime(snapshot.getLastUpdated());

            // Convert demographics
            if (snapshot.getFirstName() != null || snapshot.getLastName() != null) {
                PatientContext.PatientDemographics demographics = new PatientContext.PatientDemographics();
                demographics.setAge(snapshot.getAge() != null ? snapshot.getAge() : 0);
                demographics.setGender(snapshot.getGender());
                context.setDemographics(demographics);
            }

            // Convert encounter context
            if (snapshot.getEncounterContext() != null) {
                context.setCurrentEncounterId(snapshot.getEncounterContext().getEncounterId());
                context.setAdmissionTime(snapshot.getEncounterContext().getAdmissionTime());

                // Convert to PatientLocation
                PatientContext.PatientLocation location = new PatientContext.PatientLocation();
                location.setUnit(snapshot.getEncounterContext().getDepartment());
                location.setRoom(snapshot.getEncounterContext().getRoom());
                location.setBed(snapshot.getEncounterContext().getBed());
                context.setLocation(location);
            }

            // Set risk scores
            context.setReadmissionRiskScore(snapshot.getReadmissionRisk());
            context.setAcuityScore(snapshot.getSepsisScore() != null ? snapshot.getSepsisScore() * 100 : 0.0);

            // Set allergies
            context.setAllergies(snapshot.getAllergies());

            // Set care team and risk cohorts from Neo4j graph data
            if (snapshot.getCareTeam() != null && !snapshot.getCareTeam().isEmpty()) {
                context.setCareTeam(snapshot.getCareTeam());
            }
            if (snapshot.getRiskCohorts() != null && !snapshot.getRiskCohorts().isEmpty()) {
                context.setRiskCohorts(snapshot.getRiskCohorts());
            }

            LOG.debug("Converting snapshot to context for {}: careTeam={}, riskCohorts={}",
                snapshot.getPatientId(),
                context.getCareTeam() != null ? context.getCareTeam().size() : 0,
                context.getRiskCohorts() != null ? context.getRiskCohorts().size() : 0);

            return context;
        }

        /**
         * Check if event indicates encounter closure (discharge).
         */
        private boolean isEncounterClosure(CanonicalEvent event) {
            if (event.getEventType() == EventType.PATIENT_DISCHARGE) {
                return true;
            }

            // Also check payload for discharge indicator
            Map<String, Object> payload = event.getPayload();
            if (payload != null && "discharge".equals(payload.get("encounter_type"))) {
                return true;
            }

            return false;
        }

        /**
         * Flush patient state to external systems on encounter closure.
         *
         * Per architecture spec:
         * - Persist complete snapshot to FHIR store for historical record
         * - Update Neo4j care network graph
         * - State remains in Flink for 7 days (TTL for readmission correlation)
         *
         * This method uses async submission to avoid blocking the Flink stream.
         */
        private void flushStateToExternalSystems(com.cardiofit.flink.models.PatientSnapshot snapshot) {
            LOG.info("Flushing state for patient {} to external systems", snapshot.getPatientId());

            try {
                // Flush to Google FHIR store (async - fire and forget)
                fhirClient.flushSnapshot(snapshot)
                    .thenAccept(result -> {
                        LOG.info("Successfully flushed snapshot to Google FHIR store for patient: {}",
                            snapshot.getPatientId());
                    })
                    .exceptionally(throwable -> {
                        LOG.error("Failed to flush snapshot to FHIR store for patient: {}",
                            snapshot.getPatientId(), throwable);
                        return null;
                    });

                // Update Neo4j care network (if available)
                if (neo4jClient != null) {
                    try {
                        neo4jClient.updateCareNetwork(snapshot);
                        LOG.info("Successfully updated Neo4j care network for patient: {}",
                            snapshot.getPatientId());
                    } catch (Exception neo4jError) {
                        LOG.error("Failed to update Neo4j for patient: {}",
                            snapshot.getPatientId(), neo4jError);
                    }
                }

                // Note: State remains in Flink with 7-day TTL for readmission correlation

            } catch (Exception e) {
                LOG.error("Error flushing state for patient {}", snapshot.getPatientId(), e);
                // Don't fail the stream - log error and continue
            }
        }

        @Override
        public void onTimer(long timestamp, OnTimerContext ctx, Collector<EnrichedEvent> out)
                throws Exception {
            // Clean up old context data
            PatientContext context = patientContextState.value();
            if (context != null && isContextExpired(context, timestamp)) {
                LOG.info("Cleaning up expired context for patient: {}", ctx.getCurrentKey());
                patientContextState.clear();
                recentEventsState.clear();
                // Keep medications and vitals as they may be longer-lived
            }
        }

        private PatientContext createNewPatientContext(String patientId, String key) {
            PatientContext context = new PatientContext();
            context.setPatientId(patientId);
            context.setFirstEventTime(System.currentTimeMillis());
            context.setLastEventTime(System.currentTimeMillis());
            context.setEventCount(0);
            context.setActiveMedications(new HashMap<>());
            context.setCurrentVitals(new HashMap<>());
            context.setRiskFactors(new ArrayList<>());
            context.setContextVersion("2.0");

            LOG.info("Created new patient context for: {}", patientId);
            return context;
        }

        private PatientContext updatePatientContext(PatientContext context, CanonicalEvent event, Context ctx)
                throws Exception {

            // Update basic metrics
            context.setLastEventTime(event.getEventTime());
            context.setEventCount(context.getEventCount() + 1);

            // Update encounter context
            if (event.getEncounterId() != null) {
                context.setCurrentEncounterId(event.getEncounterId());
            }

            // Update clinical context based on event type
            updateClinicalContext(context, event);

            // Calculate derived metrics
            calculateDerivedMetrics(context, event, ctx);

            return context;
        }

        private void updateClinicalContext(PatientContext context, CanonicalEvent event) {
            EventType eventType = event.getEventType();
            Map<String, Object> payload = event.getPayload();

            switch (eventType) {
                case VITAL_SIGN:
                    updateVitalSigns(context, payload);
                    break;
                case LAB_RESULT:
                    updateLabResults(context, payload);
                    break;
                case MEDICATION_ADMINISTERED:
                case MEDICATION_ORDERED:
                    updateMedications(context, payload);
                    break;
                case PATIENT_ADMISSION:
                    context.setAdmissionTime(event.getEventTime());
                    break;
                case PATIENT_DISCHARGE:
                    context.setDischargeTime(event.getEventTime());
                    break;
                default:
                    // Handle other event types
                    break;
            }
        }

        private void updateVitalSigns(PatientContext context, Map<String, Object> payload) {
            Map<String, Object> vitals = context.getCurrentVitals();

            // Extract vital sign values
            if (payload.containsKey("heart_rate")) {
                vitals.put("heart_rate", payload.get("heart_rate"));
            }
            if (payload.containsKey("blood_pressure")) {
                vitals.put("blood_pressure", payload.get("blood_pressure"));
            }
            if (payload.containsKey("temperature")) {
                vitals.put("temperature", payload.get("temperature"));
            }
            if (payload.containsKey("oxygen_saturation")) {
                vitals.put("oxygen_saturation", payload.get("oxygen_saturation"));
            }
        }

        private void updateLabResults(PatientContext context, Map<String, Object> payload) {
            // Update latest lab values
            if (payload.containsKey("test_name") && payload.containsKey("value")) {
                String testName = (String) payload.get("test_name");
                Object value = payload.get("value");

                Map<String, Object> labResults = (Map<String, Object>)
                    context.getCurrentVitals().computeIfAbsent("lab_results", k -> new HashMap<>());
                labResults.put(testName.toLowerCase(), value);
            }
        }

        private void updateMedications(PatientContext context, Map<String, Object> payload) {
            if (payload.containsKey("medication_name")) {
                String medName = (String) payload.get("medication_name");
                Map<String, Object> medInfo = new HashMap<>();
                medInfo.put("name", medName);
                medInfo.put("last_administered", System.currentTimeMillis());

                if (payload.containsKey("dosage")) {
                    medInfo.put("dosage", payload.get("dosage"));
                }

                context.getActiveMedications().put(medName, medInfo);
            }
        }

        private void calculateDerivedMetrics(PatientContext context, CanonicalEvent event, Context ctx)
                throws Exception {

            // Calculate event frequency
            Iterable<CanonicalEvent> recentEvents = recentEventsState.get();
            int recentEventCount = 0;
            for (CanonicalEvent recentEvent : recentEvents) {
                recentEventCount++;
            }
            context.setRecentEventCount(recentEventCount);

            // Calculate acuity score based on event types and vitals
            double acuityScore = calculateAcuityScore(context, event);
            context.setAcuityScore(acuityScore);

            // Update risk factors
            updateRiskFactors(context, event);
        }

        private double calculateAcuityScore(PatientContext context, CanonicalEvent event) {
            double score = 0.0;

            // Base score from event type
            if (event.getEventType().isCritical()) {
                score += 25.0;
            } else if (event.getEventType().isClinical()) {
                score += 10.0;
            } else {
                score += 5.0;
            }

            // Vital signs contribution
            Map<String, Object> vitals = context.getCurrentVitals();
            if (vitals.containsKey("heart_rate")) {
                Object hrObj = vitals.get("heart_rate");
                if (hrObj instanceof Number) {
                    double hr = ((Number) hrObj).doubleValue();
                    if (hr > 100 || hr < 60) score += 10.0;
                    if (hr > 120 || hr < 50) score += 20.0;
                }
            }

            // Event frequency contribution
            score += Math.min(context.getRecentEventCount() * 2.0, 30.0);

            return Math.min(score, 100.0); // Cap at 100
        }

        private void updateRiskFactors(PatientContext context, CanonicalEvent event) {
            List<String> riskFactors = context.getRiskFactors();

            // Check for drug interactions
            if (event.getEventType().isMedicationRelated()) {
                // This would typically query a drug interaction database
                // For now, just flag multiple concurrent medications
                if (context.getActiveMedications().size() > 3) {
                    if (!riskFactors.contains("polypharmacy")) {
                        riskFactors.add("polypharmacy");
                    }
                }
            }

            // Check for abnormal vital patterns
            if (context.getAcuityScore() > 70.0) {
                if (!riskFactors.contains("high_acuity")) {
                    riskFactors.add("high_acuity");
                }
            }
        }

        private EnrichedEvent enrichEventWithContext(CanonicalEvent event, PatientContext context, Context ctx) {
            EnrichedEvent enriched = new EnrichedEvent();

            // Copy base event data
            enriched.setId(event.getId());
            enriched.setPatientId(event.getPatientId());
            enriched.setEncounterId(event.getEncounterId());
            enriched.setEventType(event.getEventType());
            enriched.setEventTime(event.getEventTime());
            enriched.setSourceSystem(event.getSourceSystem());
            enriched.setPayload(new HashMap<>(event.getPayload()));

            // Add enrichment data
            enriched.setPatientContext(context);
            enriched.setProcessingTime(System.currentTimeMillis());
            enriched.setEnrichmentVersion("2.0");

            // Add trends and predictions
            Map<String, Object> enrichmentData = new HashMap<>();
            enrichmentData.put("acuity_score", context.getAcuityScore());
            enrichmentData.put("risk_factors", context.getRiskFactors());
            enrichmentData.put("event_sequence", context.getEventCount());
            enrichmentData.put("context_age_hours",
                (System.currentTimeMillis() - context.getFirstEventTime()) / (1000.0 * 3600.0));

            enriched.setEnrichmentData(enrichmentData);

            return enriched;
        }

        private void updateRecentEvents(CanonicalEvent event) throws Exception {
            // Add new event
            recentEventsState.add(event);

            // Keep only last 50 events
            List<CanonicalEvent> events = new ArrayList<>();
            for (CanonicalEvent e : recentEventsState.get()) {
                events.add(e);
            }

            if (events.size() > 50) {
                // Keep most recent 50
                events.sort((a, b) -> Long.compare(b.getEventTime(), a.getEventTime()));
                events = events.subList(0, 50);

                recentEventsState.clear();
                for (CanonicalEvent e : events) {
                    recentEventsState.add(e);
                }
            }
        }

        private void updateMedicationState(CanonicalEvent event, Context ctx) throws Exception {
            if (event.getEventType().isMedicationRelated()) {
                Map<String, Object> payload = event.getPayload();
                if (payload.containsKey("medication_name")) {
                    String medName = (String) payload.get("medication_name");

                    MedicationState medState = activeMedicationsState.get(medName);
                    if (medState == null) {
                        medState = new MedicationState();
                        medState.setMedicationName(medName);
                        medState.setFirstAdministered(event.getEventTime());
                    }

                    medState.setLastAdministered(event.getEventTime());
                    medState.setAdministrationCount(medState.getAdministrationCount() + 1);

                    activeMedicationsState.put(medName, medState);
                }
            }
        }

        private void updateVitalTrends(CanonicalEvent event, Context ctx) throws Exception {
            if (event.getEventType() == EventType.VITAL_SIGN) {
                Map<String, Object> payload = event.getPayload();

                for (Map.Entry<String, Object> entry : payload.entrySet()) {
                    String vitalName = entry.getKey();
                    Object value = entry.getValue();

                    if (value instanceof Number) {
                        VitalSignTrend trend = vitalTrendsState.get(vitalName);
                        if (trend == null) {
                            trend = new VitalSignTrend();
                            trend.setVitalName(vitalName);
                        }

                        trend.addValue(((Number) value).doubleValue(), event.getEventTime());
                        vitalTrendsState.put(vitalName, trend);
                    }
                }
            }
        }

        private boolean isContextExpired(PatientContext context, long currentTime) {
            // Context expires after 24 hours of inactivity
            return (currentTime - context.getLastEventTime()) > Duration.ofHours(24).toMillis();
        }

        /**
         * Clean up external resources when operator shuts down
         */
        @Override
        public void close() throws Exception {
            LOG.info("Closing PatientContextProcessor - cleaning up external clients");

            // Close GoogleFHIRClient and its HTTP client
            if (fhirClient != null) {
                try {
                    fhirClient.close();
                    LOG.info("GoogleFHIRClient closed successfully");
                } catch (Exception e) {
                    LOG.error("Error closing GoogleFHIRClient", e);
                }
            }

            // Close Neo4jGraphClient and its session
            if (neo4jClient != null) {
                try {
                    neo4jClient.close();
                    LOG.info("Neo4jGraphClient closed successfully");
                } catch (Exception e) {
                    LOG.error("Error closing Neo4jGraphClient", e);
                }
            }

            LOG.info("PatientContextProcessor shutdown complete");
        }
    }

    /**
     * Async-enabled patient context processor (AsyncDataStream pattern).
     *
     * This processor receives pre-enriched data from AsyncDataStream,
     * eliminating blocking .get() calls for 10x-50x throughput improvement.
     *
     * Key differences from PatientContextProcessor:
     * - No FHIR/Neo4j clients (async enrichment done upstream)
     * - Receives EnrichedEventWithSnapshot instead of CanonicalEvent
     * - No blocking I/O in processElement()
     * - State management identical to original processor
     */
    public static class PatientContextProcessorAsync
            extends KeyedProcessFunction<String, AsyncPatientEnricher.EnrichedEventWithSnapshot, EnrichedEvent> {

        // PRIMARY STATE: PatientSnapshot with 7-day TTL
        private transient ValueState<PatientSnapshot> patientSnapshotState;

        // LEGACY STATE: Keep for backward compatibility
        private transient ValueState<PatientContext> patientContextState;
        private transient ListState<CanonicalEvent> recentEventsState;
        private transient MapState<String, MedicationState> activeMedicationsState;
        private transient MapState<String, VitalSignTrend> vitalTrendsState;

        // FHIR client for encounter closure only (no async lookups)
        private transient GoogleFHIRClient fhirClient;

        // JSON mapper
        private transient ObjectMapper objectMapper;

        @Override
        public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
            super.open(openContext);
            LOG.info("Opening PatientContextProcessorAsync (AsyncDataStream mode)");

            // ========== PRIMARY STATE: PatientSnapshot with 7-day TTL ==========
            StateTtlConfig ttlConfig = StateTtlConfig
                .newBuilder(java.time.Duration.ofDays(7))
                .setUpdateType(StateTtlConfig.UpdateType.OnCreateAndWrite)
                .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
                .build();

            ValueStateDescriptor<PatientSnapshot> snapshotDescriptor =
                new ValueStateDescriptor<>("patient-snapshot", PatientSnapshot.class);
            snapshotDescriptor.enableTimeToLive(ttlConfig);

            patientSnapshotState = getRuntimeContext().getState(snapshotDescriptor);

            // ========== LEGACY STATE ==========
            patientContextState = getRuntimeContext().getState(
                new ValueStateDescriptor<>("patient-context", PatientContext.class));
            recentEventsState = getRuntimeContext().getListState(
                new ListStateDescriptor<>("recent-events", CanonicalEvent.class));
            activeMedicationsState = getRuntimeContext().getMapState(
                new MapStateDescriptor<>("active-medications", String.class, MedicationState.class));
            vitalTrendsState = getRuntimeContext().getMapState(
                new MapStateDescriptor<>("vital-trends", String.class, VitalSignTrend.class));

            // ========== FHIR CLIENT (for encounter closure only) ==========
            try {
                String credentialsPath = KafkaConfigLoader.getGoogleCloudCredentialsPath();
                fhirClient = new GoogleFHIRClient(
                    KafkaConfigLoader.getGoogleCloudProjectId(),
                    KafkaConfigLoader.getGoogleCloudLocation(),
                    KafkaConfigLoader.getGoogleCloudDatasetId(),
                    KafkaConfigLoader.getGoogleCloudFhirStoreId(),
                    credentialsPath
                );
                fhirClient.initialize();
                LOG.info("FHIR client initialized for encounter closure");
            } catch (Exception e) {
                LOG.error("Failed to initialize FHIR client", e);
                throw e;
            }

            // Initialize JSON mapper
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());

            LOG.info("PatientContextProcessorAsync initialization complete");
        }

        @Override
        public void processElement(
                AsyncPatientEnricher.EnrichedEventWithSnapshot enrichedInput,
                Context ctx,
                Collector<EnrichedEvent> out) throws Exception {

            CanonicalEvent event = enrichedInput.getEvent();
            PatientSnapshot asyncSnapshot = enrichedInput.getSnapshot();
            String patientId = event.getPatientId();

            try {
                // ========== STATE MANAGEMENT (NON-BLOCKING) ==========
                PatientSnapshot currentSnapshot = patientSnapshotState.value();

                if (currentSnapshot == null) {
                    // First-time patient: use async-enriched snapshot
                    LOG.info("First-time patient {}: using async-enriched snapshot", patientId);
                    patientSnapshotState.update(asyncSnapshot);
                    currentSnapshot = asyncSnapshot;
                } else {
                    // Existing patient: progressive enrichment
                    currentSnapshot.updateWithEvent(event);
                    patientSnapshotState.update(currentSnapshot);
                }

                // ========== CLINICAL SCORING (NEW) ==========
                Map<String, Double> clinicalScores = calculateClinicalScores(currentSnapshot);

                // ========== IMMEDIATE ALERT GENERATION (NEW) ==========
                List<SimpleAlert> immediateAlerts = generateImmediateAlerts(event, currentSnapshot, clinicalScores);

                // ========== RISK INDICATORS GENERATION (NEW) ==========
                RiskIndicators riskIndicators = buildRiskIndicators(currentSnapshot, clinicalScores);

                // ========== CHECK FOR ENCOUNTER CLOSURE ==========
                if (isEncounterClosureEvent(event)) {
                    LOG.info("Encounter closure detected for patient: {}", patientId);
                    flushSnapshotToFHIR(currentSnapshot);
                }

                // ========== CREATE ENRICHED EVENT WITH NEW FIELDS ==========
                EnrichedEvent enriched = createEnrichedEventFromSnapshot(event, currentSnapshot);

                // Set new enrichment fields
                enriched.setImmediateAlerts(immediateAlerts);
                enriched.setRiskIndicators(riskIndicators);
                enriched.setClinicalScores(clinicalScores);

                // ========== UPDATE LEGACY STATE ==========
                updateLegacyState(event);

                out.collect(enriched);

                // Log enrichment summary
                LOG.info("Enriched event for patient {}: alerts={}, riskFlags={}, scores={}",
                    patientId, immediateAlerts.size(), countSetFlags(riskIndicators), clinicalScores.keySet());

            } catch (Exception e) {
                LOG.error("Error processing event for patient {}: {}", patientId, e.getMessage(), e);
            }
        }

        /**
         * Check if event signals encounter closure (same logic as original processor).
         */
        private boolean isEncounterClosureEvent(CanonicalEvent event) {
            return event.getEventType() != null &&
                   (event.getEventType().toString().equalsIgnoreCase("discharge") ||
                    event.getEventType().toString().equalsIgnoreCase("encounter_end") ||
                    event.getEventType().toString().equalsIgnoreCase("admission_complete"));
        }

        /**
         * Flush patient snapshot to FHIR store (same logic as original processor).
         */
        private void flushSnapshotToFHIR(PatientSnapshot snapshot) {
            try {
                fhirClient.flushSnapshot(snapshot)
                    .thenAccept(voidResult -> {
                        LOG.info("Patient snapshot flushed to FHIR store: {}", snapshot.getPatientId());
                    })
                    .exceptionally(throwable -> {
                        LOG.error("Error flushing snapshot to FHIR: {}", throwable.getMessage());
                        return null;
                    });
            } catch (Exception e) {
                LOG.error("Error initiating FHIR snapshot flush: {}", e.getMessage());
            }
        }

        /**
         * Create enriched event from snapshot with FHIR and Neo4j enrichment data.
         */
        private EnrichedEvent createEnrichedEventFromSnapshot(CanonicalEvent event, PatientSnapshot snapshot) {
            EnrichedEvent enriched = EnrichedEvent.builder()
                .id(event.getId())
                .patientId(event.getPatientId())
                .encounterId(event.getEncounterId())
                .eventType(event.getEventType())
                .eventTime(event.getEventTime())
                .processingTime(System.currentTimeMillis())
                .sourceSystem(event.getSourceSystem())
                .payload(event.getPayload())
                .build();

            // Create PatientContext from snapshot
            PatientContext context = convertSnapshotToContext(snapshot);
            enriched.setPatientContext(context);

            // ===== POPULATE ENRICHMENT DATA WITH FHIR AND NEO4J DATA =====
            Map<String, Object> enrichmentData = new HashMap<>();

            // State metadata
            enrichmentData.put("state_version", snapshot.getStateVersion());
            enrichmentData.put("was_new_patient", snapshot.isNewPatient());

            // Risk scores
            if (snapshot.getSepsisScore() != null) {
                enrichmentData.put("sepsis_score", snapshot.getSepsisScore());
            }
            if (snapshot.getDeteriorationScore() != null) {
                enrichmentData.put("deterioration_score", snapshot.getDeteriorationScore());
            }
            if (snapshot.getReadmissionRisk() != null) {
                enrichmentData.put("readmission_risk", snapshot.getReadmissionRisk());
            }

            // FHIR demographics
            if (snapshot.getFirstName() != null || snapshot.getLastName() != null) {
                Map<String, Object> demographics = new HashMap<>();
                demographics.put("firstName", snapshot.getFirstName());
                demographics.put("lastName", snapshot.getLastName());
                demographics.put("dateOfBirth", snapshot.getDateOfBirth());
                demographics.put("age", snapshot.getAge());
                demographics.put("gender", snapshot.getGender());
                demographics.put("mrn", snapshot.getMrn());
                enrichmentData.put("fhir_demographics", demographics);
            }

            // FHIR clinical data - always include even if empty to show data was fetched
            enrichmentData.put("fhir_conditions", snapshot.getActiveConditions() != null ?
                snapshot.getActiveConditions() : new ArrayList<>());
            enrichmentData.put("fhir_medications", snapshot.getActiveMedications() != null ?
                snapshot.getActiveMedications() : new ArrayList<>());
            enrichmentData.put("fhir_allergies", snapshot.getAllergies() != null ?
                snapshot.getAllergies() : new ArrayList<>());

            // Neo4j graph data
            if (snapshot.getCareTeam() != null && !snapshot.getCareTeam().isEmpty()) {
                enrichmentData.put("neo4j_care_team", snapshot.getCareTeam());
            }
            if (snapshot.getRiskCohorts() != null && !snapshot.getRiskCohorts().isEmpty()) {
                enrichmentData.put("neo4j_risk_cohorts", snapshot.getRiskCohorts());
            }

            // Lab values if present
            if (snapshot.getLabHistory() != null && !snapshot.getLabHistory().isEmpty()) {
                LabValues latestLabs = snapshot.getLabHistory().getLatestAsLabValues();
                if (latestLabs != null) {
                    Map<String, Object> labData = new HashMap<>();
                    if (latestLabs.getCreatinine() != null) labData.put("creatinine", latestLabs.getCreatinine());
                    if (latestLabs.getLactate() != null) labData.put("lactate", latestLabs.getLactate());
                    if (latestLabs.getWbcCount() != null) labData.put("wbc_count", latestLabs.getWbcCount());
                    if (latestLabs.getPlatelets() != null) labData.put("platelets", latestLabs.getPlatelets());
                    if (latestLabs.getTroponin() != null) labData.put("troponin", latestLabs.getTroponin());
                    if (!labData.isEmpty()) {
                        enrichmentData.put("latest_labs", labData);
                    }
                }
            }

            // Vital signs if present
            if (snapshot.getVitalsHistory() != null && !snapshot.getVitalsHistory().isEmpty()) {
                VitalSign latestVitals = snapshot.getVitalsHistory().getLatest();
                if (latestVitals != null) {
                    Map<String, Object> vitalData = new HashMap<>();
                    if (latestVitals.getHeartRate() != null) vitalData.put("heart_rate", latestVitals.getHeartRate());
                    if (latestVitals.getBloodPressureSystolic() != null) {
                        vitalData.put("blood_pressure_systolic", latestVitals.getBloodPressureSystolic());
                    }
                    if (latestVitals.getBloodPressureDiastolic() != null) {
                        vitalData.put("blood_pressure_diastolic", latestVitals.getBloodPressureDiastolic());
                    }
                    if (latestVitals.getTemperature() != null) vitalData.put("temperature", latestVitals.getTemperature());
                    if (latestVitals.getOxygenSaturation() != null) {
                        vitalData.put("oxygen_saturation", latestVitals.getOxygenSaturation());
                    }
                    if (latestVitals.getRespiratoryRate() != null) {
                        vitalData.put("respiratory_rate", latestVitals.getRespiratoryRate());
                    }
                    if (!vitalData.isEmpty()) {
                        enrichmentData.put("latest_vitals", vitalData);
                    }
                }
            }

            enriched.setEnrichmentData(enrichmentData);

            LOG.debug("Created enriched event for patient {} with enrichment data containing {} fields",
                snapshot.getPatientId(), enrichmentData.size());
            // ===== END OF ENRICHMENT DATA POPULATION =====

            return enriched;
        }

        /**
         * Convert snapshot to patient context with full FHIR enrichment.
         */
        private PatientContext convertSnapshotToContext(PatientSnapshot snapshot) {
            PatientContext context = new PatientContext();
            context.setPatientId(snapshot.getPatientId());

            // Demographics - convert from snapshot's PatientDemographics to context's inner class
            if (snapshot.getDemographics() != null) {
                PatientContext.PatientDemographics contextDemographics = new PatientContext.PatientDemographics();
                com.cardiofit.flink.models.PatientDemographics snapshotDemographics = snapshot.getDemographics();
                contextDemographics.setAge(snapshotDemographics.getAge());
                contextDemographics.setGender(snapshotDemographics.getGender());
                // dateOfBirth not available in PatientContext.PatientDemographics (inner class is simplified)
                context.setDemographics(contextDemographics);
            }

            // Convert medications from List<Medication> to Map<String, Object>
            if (snapshot.getActiveMedications() != null && !snapshot.getActiveMedications().isEmpty()) {
                Map<String, Object> medicationsMap = new HashMap<>();
                for (Medication med : snapshot.getActiveMedications()) {
                    Map<String, Object> medDetails = new HashMap<>();
                    medDetails.put("name", med.getName());
                    medDetails.put("dosage", med.getDosage());
                    medDetails.put("frequency", med.getFrequency());
                    medDetails.put("route", med.getRoute());
                    medDetails.put("startDate", med.getStartDate());
                    medDetails.put("status", med.getStatus());
                    // Use code or name as key since there's no id field
                    String key = med.getCode() != null ? med.getCode() : med.getName();
                    medicationsMap.put(key, medDetails);
                }
                context.setActiveMedications(medicationsMap);
            }

            // Convert conditions from List<Condition> to List<String> (chronic condition names)
            if (snapshot.getActiveConditions() != null && !snapshot.getActiveConditions().isEmpty()) {
                List<String> conditionNames = new ArrayList<>();
                for (Condition condition : snapshot.getActiveConditions()) {
                    if (condition.getDisplay() != null) {
                        conditionNames.add(condition.getDisplay());
                    }
                }
                context.setChronicConditions(conditionNames);
            }

            // Allergies (already List<String>)
            context.setAllergies(snapshot.getAllergies());

            // Encounter context
            if (snapshot.getEncounterContext() != null) {
                context.setCurrentEncounterId(snapshot.getEncounterContext().getEncounterId());
                if (snapshot.getAdmissionTime() != null) {
                    context.setAdmissionTime(snapshot.getAdmissionTime());
                }
            }

            // Location (convert String to description if needed)
            if (snapshot.getLocation() != null) {
                PatientContext.PatientLocation loc = new PatientContext.PatientLocation();
                loc.setFacility(snapshot.getLocation());
                context.setLocation(loc);
            }

            // Risk scores
            context.setAcuityScore(snapshot.getSepsisScore() != null ? snapshot.getSepsisScore() : 0.0);
            context.setReadmissionRiskScore(snapshot.getReadmissionRisk());

            // Timestamps
            context.setLastEventTime(snapshot.getLastUpdated());
            context.setFirstEventTime(snapshot.getFirstSeen());

            // Context version
            context.setContextVersion("2.0");

            // Set care team and risk cohorts from Neo4j graph data
            if (snapshot.getCareTeam() != null && !snapshot.getCareTeam().isEmpty()) {
                context.setCareTeam(snapshot.getCareTeam());
            }
            if (snapshot.getRiskCohorts() != null && !snapshot.getRiskCohorts().isEmpty()) {
                context.setRiskCohorts(snapshot.getRiskCohorts());
            }

            LOG.debug("Converted snapshot to context for patient {}: demographics={}, meds={}, conditions={}, careTeam={}, cohorts={}",
                snapshot.getPatientId(),
                context.getDemographics() != null,
                context.getActiveMedications() != null ? context.getActiveMedications().size() : 0,
                context.getChronicConditions() != null ? context.getChronicConditions().size() : 0,
                context.getCareTeam() != null ? context.getCareTeam().size() : 0,
                context.getRiskCohorts() != null ? context.getRiskCohorts().size() : 0);

            return context;
        }

        /**
         * Update legacy state for backward compatibility.
         */
        private void updateLegacyState(CanonicalEvent event) throws Exception {
            // Add to recent events buffer
            recentEventsState.add(event);

            // Trim to last 100 events
            List<CanonicalEvent> recentEvents = new ArrayList<>();
            for (CanonicalEvent e : recentEventsState.get()) {
                recentEvents.add(e);
            }
            if (recentEvents.size() > 100) {
                recentEventsState.clear();
                recentEvents.subList(recentEvents.size() - 100, recentEvents.size())
                    .forEach(e -> {
                        try {
                            recentEventsState.add(e);
                        } catch (Exception ex) {
                            LOG.error("Error updating recent events", ex);
                        }
                    });
            }
        }

        /**
         * Generate immediate threshold-based alerts for vital sign and lab value breaches.
         *
         * Per architecture specification C05_10 Lines 207-272:
         * - Checks vital signs against clinical thresholds
         * - Checks lab values against critical ranges
         * - Creates SimpleAlert objects for each breach
         * - Returns empty list if no thresholds breached
         *
         * Clinical Thresholds (evidence-based):
         * - Severe tachycardia: HR > 140 bpm (cardiac emergency)
         * - Critical bradycardia: HR < 50 bpm (cardiac conduction)
         * - Severe hypotension: SBP < 85 mmHg (shock state)
         * - Hypertensive emergency: SBP > 200 mmHg
         * - High fever: Temp > 39.5°C (severe infection)
         * - Severe hypothermia: Temp < 35.0°C (severe sepsis)
         * - Critical hypoxia: SpO2 < 88% (respiratory failure)
         * - Critical lactate: > 4.0 mmol/L (septic shock)
         * - Critical creatinine: > 3.0 mg/dL (severe AKI)
         *
         * @param event The canonical event being processed
         * @param snapshot The patient's current state snapshot
         * @param scores Clinical scores (MEWS, qSOFA, etc.)
         * @return List of SimpleAlert objects (empty if no alerts)
         */
        private List<SimpleAlert> generateImmediateAlerts(
                CanonicalEvent event,
                PatientSnapshot snapshot,
                Map<String, Double> scores) {

            List<SimpleAlert> alerts = new ArrayList<>();

            // Extract vitals from snapshot's vitalsHistory
            VitalsHistory vitalsHistory = snapshot.getVitalsHistory();
            if (vitalsHistory == null || vitalsHistory.isEmpty()) {
                return alerts; // No vitals available
            }

            VitalSign latestVitals = vitalsHistory.getLatest();
            if (latestVitals == null) {
                return alerts;
            }

            // VITAL SIGN THRESHOLD CHECKS

            // Heart Rate
            if (latestVitals.getHeartRate() != null) {
                double hr = latestVitals.getHeartRate();

                if (hr > 140) {
                    alerts.add(SimpleAlert.builder()
                        .alertId(UUID.randomUUID().toString())
                        .patientId(snapshot.getPatientId())
                        .alertType(AlertType.VITAL_THRESHOLD_BREACH)
                        .severity(AlertSeverity.CRITICAL)
                        .message("Severe tachycardia detected: HR " + hr + " bpm (threshold: 140)")
                        .context(Map.of(
                            "vital_sign", "heart_rate",
                            "value", hr,
                            "threshold", 140,
                            "unit", "bpm"
                        ))
                        .timestamp(System.currentTimeMillis())
                        .sourceModule("MODULE_2_THRESHOLD")
                        .build());
                } else if (hr < 50) {
                    alerts.add(SimpleAlert.builder()
                        .alertId(UUID.randomUUID().toString())
                        .patientId(snapshot.getPatientId())
                        .alertType(AlertType.VITAL_THRESHOLD_BREACH)
                        .severity(AlertSeverity.CRITICAL)
                        .message("Critical bradycardia detected: HR " + hr + " bpm (threshold: 50)")
                        .context(Map.of(
                            "vital_sign", "heart_rate",
                            "value", hr,
                            "threshold", 50,
                            "unit", "bpm"
                        ))
                        .timestamp(System.currentTimeMillis())
                        .sourceModule("MODULE_2_THRESHOLD")
                        .build());
                }
            }

            // Blood Pressure
            if (latestVitals.getBloodPressureSystolic() != null) {
                double sbp = latestVitals.getBloodPressureSystolic();

                if (sbp < 85) {
                    alerts.add(SimpleAlert.builder()
                        .alertId(UUID.randomUUID().toString())
                        .patientId(snapshot.getPatientId())
                        .alertType(AlertType.VITAL_THRESHOLD_BREACH)
                        .severity(AlertSeverity.CRITICAL)
                        .message("Severe hypotension detected: SBP " + sbp + " mmHg (threshold: 85)")
                        .context(Map.of(
                            "vital_sign", "systolic_bp",
                            "value", sbp,
                            "threshold", 85,
                            "unit", "mmHg"
                        ))
                        .timestamp(System.currentTimeMillis())
                        .sourceModule("MODULE_2_THRESHOLD")
                        .build());
                } else if (sbp > 200) {
                    alerts.add(SimpleAlert.builder()
                        .alertId(UUID.randomUUID().toString())
                        .patientId(snapshot.getPatientId())
                        .alertType(AlertType.VITAL_THRESHOLD_BREACH)
                        .severity(AlertSeverity.HIGH)
                        .message("Hypertensive emergency: SBP " + sbp + " mmHg (threshold: 200)")
                        .context(Map.of(
                            "vital_sign", "systolic_bp",
                            "value", sbp,
                            "threshold", 200,
                            "unit", "mmHg"
                        ))
                        .timestamp(System.currentTimeMillis())
                        .sourceModule("MODULE_2_THRESHOLD")
                        .build());
                }
            }

            // Temperature
            if (latestVitals.getTemperature() != null) {
                double temp = latestVitals.getTemperature();

                if (temp > 39.5) {
                    alerts.add(SimpleAlert.builder()
                        .alertId(UUID.randomUUID().toString())
                        .patientId(snapshot.getPatientId())
                        .alertType(AlertType.VITAL_THRESHOLD_BREACH)
                        .severity(AlertSeverity.HIGH)
                        .message("High fever detected: " + temp + "°C (threshold: 39.5)")
                        .context(Map.of(
                            "vital_sign", "temperature",
                            "value", temp,
                            "threshold", 39.5,
                            "unit", "celsius"
                        ))
                        .timestamp(System.currentTimeMillis())
                        .sourceModule("MODULE_2_THRESHOLD")
                        .build());
                } else if (temp < 35.0) {
                    alerts.add(SimpleAlert.builder()
                        .alertId(UUID.randomUUID().toString())
                        .patientId(snapshot.getPatientId())
                        .alertType(AlertType.VITAL_THRESHOLD_BREACH)
                        .severity(AlertSeverity.CRITICAL)
                        .message("Severe hypothermia: " + temp + "°C (threshold: 35.0)")
                        .context(Map.of(
                            "vital_sign", "temperature",
                            "value", temp,
                            "threshold", 35.0,
                            "unit", "celsius"
                        ))
                        .timestamp(System.currentTimeMillis())
                        .sourceModule("MODULE_2_THRESHOLD")
                        .build());
                }
            }

            // Oxygen Saturation
            if (latestVitals.getOxygenSaturation() != null) {
                double spo2 = latestVitals.getOxygenSaturation();

                if (spo2 < 88) {
                    alerts.add(SimpleAlert.builder()
                        .alertId(UUID.randomUUID().toString())
                        .patientId(snapshot.getPatientId())
                        .alertType(AlertType.VITAL_THRESHOLD_BREACH)
                        .severity(AlertSeverity.CRITICAL)
                        .message("Critical hypoxia: SpO2 " + spo2 + "% (threshold: 88)")
                        .context(Map.of(
                            "vital_sign", "oxygen_saturation",
                            "value", spo2,
                            "threshold", 88,
                            "unit", "percent"
                        ))
                        .timestamp(System.currentTimeMillis())
                        .sourceModule("MODULE_2_THRESHOLD")
                        .build());
                }
            }

            // LAB VALUE THRESHOLD CHECKS
            LabHistory labHistory = snapshot.getLabHistory();
            if (labHistory != null && !labHistory.isEmpty()) {
                LabValues latestLabs = labHistory.getLatestAsLabValues();

                if (latestLabs != null) {
                    // Lactate
                    if (latestLabs.getLactate() != null && latestLabs.getLactate() > 4.0) {
                        alerts.add(SimpleAlert.builder()
                            .alertId(UUID.randomUUID().toString())
                            .patientId(snapshot.getPatientId())
                            .alertType(AlertType.LAB_CRITICAL_VALUE)
                            .severity(AlertSeverity.CRITICAL)
                            .message("Critical lactate elevation: " + latestLabs.getLactate() + " mmol/L (threshold: 4.0)")
                            .context(Map.of(
                                "lab_test", "lactate",
                                "value", latestLabs.getLactate(),
                                "threshold", 4.0,
                                "unit", "mmol/L",
                                "clinical_significance", "septic_shock_indicator"
                            ))
                            .timestamp(System.currentTimeMillis())
                            .sourceModule("MODULE_2_THRESHOLD")
                            .build());
                    }

                    // Creatinine
                    if (latestLabs.getCreatinine() != null && latestLabs.getCreatinine() > 3.0) {
                        alerts.add(SimpleAlert.builder()
                            .alertId(UUID.randomUUID().toString())
                            .patientId(snapshot.getPatientId())
                            .alertType(AlertType.LAB_CRITICAL_VALUE)
                            .severity(AlertSeverity.HIGH)
                            .message("Severe renal dysfunction: Creatinine " + latestLabs.getCreatinine() + " mg/dL (threshold: 3.0)")
                            .context(Map.of(
                                "lab_test", "creatinine",
                                "value", latestLabs.getCreatinine(),
                                "threshold", 3.0,
                                "unit", "mg/dL",
                                "clinical_significance", "severe_aki_or_ckd"
                            ))
                            .timestamp(System.currentTimeMillis())
                            .sourceModule("MODULE_2_THRESHOLD")
                            .build());
                    }
                }
            }

            // CLINICAL SCORE ALERTS
            if (scores != null) {
                Double qsofaScore = scores.get("qsofa_score");
                if (qsofaScore != null && qsofaScore >= 2.0) {
                    alerts.add(SimpleAlert.builder()
                        .alertId(UUID.randomUUID().toString())
                        .patientId(snapshot.getPatientId())
                        .alertType(AlertType.CLINICAL_SCORE_HIGH)
                        .severity(AlertSeverity.HIGH)
                        .message("High sepsis risk: qSOFA score " + qsofaScore + " (threshold: ≥2)")
                        .context(Map.of(
                            "score_type", "qsofa",
                            "value", qsofaScore,
                            "threshold", 2.0,
                            "clinical_significance", "high_mortality_risk_sepsis"
                        ))
                        .timestamp(System.currentTimeMillis())
                        .sourceModule("MODULE_2_THRESHOLD")
                        .build());
                }

                Double mewsScore = scores.get("mews_score");
                if (mewsScore != null && mewsScore >= 7.0) {
                    alerts.add(SimpleAlert.builder()
                        .alertId(UUID.randomUUID().toString())
                        .patientId(snapshot.getPatientId())
                        .alertType(AlertType.CLINICAL_SCORE_HIGH)
                        .severity(AlertSeverity.HIGH)
                        .message("High early warning score: MEWS " + mewsScore + " (threshold: ≥7)")
                        .context(Map.of(
                            "score_type", "mews",
                            "value", mewsScore,
                            "threshold", 7.0,
                            "clinical_significance", "high_deterioration_risk"
                        ))
                        .timestamp(System.currentTimeMillis())
                        .sourceModule("MODULE_2_THRESHOLD")
                        .build());
                }
            }

            LOG.debug("Generated {} immediate alerts for patient {}", alerts.size(), snapshot.getPatientId());
            return alerts;
        }

        /**
         * Build structured risk indicators from patient snapshot and clinical scores.
         *
         * Per architecture specification C05_10 Lines 1018-1064:
         * - Sets 30+ boolean flags based on vital signs, labs, medications, clinical context
         * - Calculates trend directions for key parameters
         * - Provides structured indicators for CEP pattern matching
         *
         * Replaces manual payload parsing in CEP patterns with clean boolean logic:
         * BEFORE: event -> (Double) event.getPayload().get("heartRate") > 100
         * AFTER: event -> event.getRiskIndicators().isTachycardia()
         *
         * @param snapshot Current patient state with vitals, labs, meds
         * @param scores Calculated clinical scores (MEWS, qSOFA)
         * @return RiskIndicators instance with all flags and trends set
         */
        private RiskIndicators buildRiskIndicators(
                PatientSnapshot snapshot,
                Map<String, Double> scores) {

            RiskIndicators.Builder builder = RiskIndicators.builder();

            // VITAL SIGN INDICATORS
            VitalsHistory vitalsHistory = snapshot.getVitalsHistory();
            if (vitalsHistory != null && !vitalsHistory.isEmpty()) {
                VitalSign latestVitals = vitalsHistory.getLatest();

                if (latestVitals != null) {
                    // Heart rate indicators
                    if (latestVitals.getHeartRate() != null) {
                        double hr = latestVitals.getHeartRate();
                        builder.tachycardia(hr > 100);
                        builder.bradycardia(hr < 60);
                    }

                    // Blood pressure indicators
                    if (latestVitals.getBloodPressureSystolic() != null) {
                        double sbp = latestVitals.getBloodPressureSystolic();
                        builder.hypotension(sbp < 90);
                        builder.hypertension(sbp > 180);
                    }

                    // Temperature indicators
                    if (latestVitals.getTemperature() != null) {
                        double temp = latestVitals.getTemperature();
                        builder.fever(temp > 38.3);
                        builder.hypothermia(temp < 36.0);
                    }

                    // Oxygen saturation indicator
                    if (latestVitals.getOxygenSaturation() != null) {
                        builder.hypoxia(latestVitals.getOxygenSaturation() < 92);
                    }

                    // Respiratory rate indicators
                    if (latestVitals.getRespiratoryRate() != null) {
                        double rr = latestVitals.getRespiratoryRate();
                        builder.tachypnea(rr > 22);
                        builder.bradypnea(rr < 12);
                    }
                }

                // TREND INDICATORS (from vitals history)
                builder.heartRateTrend(vitalsHistory.getHeartRateTrend());
                builder.bloodPressureTrend(vitalsHistory.getBloodPressureTrend());
                builder.oxygenSaturationTrend(vitalsHistory.getOxygenSaturationTrend());
                builder.temperatureTrend(vitalsHistory.getTemperatureTrend());
            }

            // LAB VALUE INDICATORS
            LabHistory labHistory = snapshot.getLabHistory();
            if (labHistory != null && !labHistory.isEmpty()) {
                LabValues latestLabs = labHistory.getLatestAsLabValues();

                if (latestLabs != null) {
                    // Lactate indicators
                    if (latestLabs.getLactate() != null) {
                        double lactate = latestLabs.getLactate();
                        builder.elevatedLactate(lactate > 2.0);
                        builder.severelyElevatedLactate(lactate > 4.0);
                    }

                    // Creatinine indicator (AKI)
                    if (latestLabs.getCreatinine() != null && snapshot.getBaselineCreatinine() != null) {
                        double currentCreat = latestLabs.getCreatinine();
                        double baselineCreat = snapshot.getBaselineCreatinine();
                        builder.elevatedCreatinine(currentCreat >= 1.5 * baselineCreat);
                    }

                    // WBC indicators
                    if (latestLabs.getWbcCount() != null) {
                        double wbc = latestLabs.getWbcCount();
                        builder.leukocytosis(wbc > 12.0); // K/µL units (12000 cells/µL = 12 K/µL)
                        builder.leukopenia(wbc < 4.0);   // K/µL units (4000 cells/µL = 4 K/µL)
                    }

                    // Platelet indicator
                    if (latestLabs.getPlatelets() != null) {
                        builder.thrombocytopenia(latestLabs.getPlatelets() < 100000);
                    }

                    // Troponin indicator (cardiac)
                    if (latestLabs.getTroponin() != null) {
                        builder.elevatedTroponin(latestLabs.getTroponin() > 0.04);
                    }
                }
            }

            // MEDICATION INDICATORS
            List<Medication> activeMeds = snapshot.getActiveMedications();
            if (activeMeds != null && !activeMeds.isEmpty()) {

                // Check for vasopressors (norepinephrine, epinephrine, vasopressin, dopamine)
                boolean onVasopressors = activeMeds.stream()
                    .anyMatch(med -> {
                        String medName = med.getMedicationName() != null ? med.getMedicationName().toLowerCase() : "";
                        return medName.contains("norepinephrine") ||
                               medName.contains("epinephrine") ||
                               medName.contains("vasopressin") ||
                               medName.contains("dopamine");
                    });
                builder.onVasopressors(onVasopressors);

                // Check for anticoagulation (warfarin, heparin, DOACs)
                boolean onAnticoagulation = activeMeds.stream()
                    .anyMatch(med -> {
                        String medName = med.getMedicationName() != null ? med.getMedicationName().toLowerCase() : "";
                        return medName.contains("warfarin") ||
                               medName.contains("heparin") ||
                               medName.contains("apixaban") ||
                               medName.contains("rivaroxaban");
                    });
                builder.onAnticoagulation(onAnticoagulation);

                // Check for nephrotoxic meds (vancomycin, aminoglycosides, NSAIDs)
                boolean onNephrotoxic = activeMeds.stream()
                    .anyMatch(med -> {
                        String medName = med.getMedicationName() != null ? med.getMedicationName().toLowerCase() : "";
                        return medName.contains("vancomycin") ||
                               medName.contains("gentamicin") ||
                               medName.contains("tobramycin") ||
                               medName.contains("ibuprofen") ||
                               medName.contains("naproxen");
                    });
                builder.onNephrotoxicMeds(onNephrotoxic);

                // Check for recent medication changes (within 24 hours)
                long twentyFourHoursAgo = System.currentTimeMillis() - (24 * 60 * 60 * 1000);
                boolean recentChange = activeMeds.stream()
                    .anyMatch(med -> med.getStartTime() != null && med.getStartTime() > twentyFourHoursAgo);
                builder.recentMedicationChange(recentChange);
            }

            // CLINICAL CONTEXT INDICATORS
            EncounterContext encounter = snapshot.getEncounterContext();
            if (encounter != null) {
                builder.inICU(encounter.getDepartment() != null &&
                             encounter.getDepartment().toUpperCase().contains("ICU"));
            }

            // Chronic conditions from snapshot
            List<Condition> conditions = snapshot.getActiveConditions();
            if (conditions != null) {
                builder.hasDiabetes(conditions.stream().anyMatch(c ->
                    c.getConditionName() != null && c.getConditionName().toLowerCase().contains("diabetes")));
                builder.hasChronicKidneyDisease(conditions.stream().anyMatch(c ->
                    c.getConditionName() != null && c.getConditionName().toLowerCase().contains("chronic kidney")));
                builder.hasHeartFailure(conditions.stream().anyMatch(c ->
                    c.getConditionName() != null && c.getConditionName().toLowerCase().contains("heart failure")));
            }

            // Post-operative indicator (within 48 hours of surgery)
            if (snapshot.getLastSurgeryTime() != null) {
                long fortyEightHoursAgo = System.currentTimeMillis() - (48 * 60 * 60 * 1000);
                builder.postOperative(snapshot.getLastSurgeryTime() > fortyEightHoursAgo);
            }

            // METADATA
            builder.lastUpdated(System.currentTimeMillis());

            // Confidence score based on data completeness
            int totalIndicators = 30; // Total possible indicators
            int populatedIndicators = 0;
            if (vitalsHistory != null && !vitalsHistory.isEmpty()) populatedIndicators += 5; // vitals category
            if (labHistory != null && !labHistory.isEmpty()) populatedIndicators += 5; // labs category
            if (activeMeds != null && !activeMeds.isEmpty()) populatedIndicators += 3; // meds category
            if (encounter != null) populatedIndicators += 1; // context category
            if (conditions != null && !conditions.isEmpty()) populatedIndicators += 3; // conditions category

            double confidenceScore = (double) populatedIndicators / totalIndicators;
            builder.confidenceScore(confidenceScore);

            // Build the indicators first to access calculated SIRS score
            RiskIndicators indicators = builder.build();

            // SEPSIS RISK CALCULATION (for Module 3 protocol matching)
            // Set sepsisRisk=true if patient meets SIRS criteria (≥2) or has severe sepsis indicators
            boolean sepsisRisk = indicators.calculateSIRS() >= 2 || indicators.hasSevereSepsisIndicators();
            indicators.setSepsisRisk(sepsisRisk);

            LOG.debug("Built risk indicators for patient {}: confidence={}, flags={}, sepsisRisk={}, SIRS={}",
                snapshot.getPatientId(), confidenceScore, countSetFlags(indicators), sepsisRisk, indicators.calculateSIRS());

            return indicators;
        }

        /**
         * Helper method to count how many risk flags are set to true.
         */
        private int countSetFlags(RiskIndicators indicators) {
            int count = 0;
            if (indicators.isTachycardia()) count++;
            if (indicators.isBradycardia()) count++;
            if (indicators.isHypotension()) count++;
            if (indicators.isHypertension()) count++;
            if (indicators.isFever()) count++;
            if (indicators.isHypothermia()) count++;
            if (indicators.isHypoxia()) count++;
            if (indicators.isTachypnea()) count++;
            if (indicators.isBradypnea()) count++;
            if (indicators.isElevatedLactate()) count++;
            if (indicators.isSeverelyElevatedLactate()) count++;
            if (indicators.isElevatedCreatinine()) count++;
            if (indicators.isLeukocytosis()) count++;
            if (indicators.isLeukopenia()) count++;
            if (indicators.isThrombocytopenia()) count++;
            if (indicators.isElevatedTroponin()) count++;
            if (indicators.isOnVasopressors()) count++;
            if (indicators.isOnAnticoagulation()) count++;
            if (indicators.isOnNephrotoxicMeds()) count++;
            if (indicators.isRecentMedicationChange()) count++;
            if (indicators.isInICU()) count++;
            if (indicators.isHasDiabetes()) count++;
            if (indicators.isHasChronicKidneyDisease()) count++;
            if (indicators.isHasHeartFailure()) count++;
            if (indicators.isPostOperative()) count++;
            return count;
        }

        /**
         * Calculate standard clinical scoring systems (MEWS, qSOFA, NEWS2, SOFA).
         *
         * Per architecture specification C05_10 Lines 273-391:
         * - MEWS (Modified Early Warning Score): 0-14 scale for deterioration
         * - qSOFA (Quick SOFA): 0-3 scale for sepsis mortality risk
         * - NEWS2 (National Early Warning Score): 0-20 scale for clinical deterioration
         * - SOFA (Sequential Organ Failure Assessment): placeholder for future
         *
         * These scores enable:
         * - Standardized deterioration detection
         * - Trend analysis over time
         * - Evidence-based escalation thresholds
         *
         * @param snapshot Current patient state with vitals and labs
         * @return Map of score names to values (e.g., "mews_score" -> 5.0)
         */
        private Map<String, Double> calculateClinicalScores(PatientSnapshot snapshot) {
            Map<String, Double> scores = new HashMap<>();

            VitalsHistory vitalsHistory = snapshot.getVitalsHistory();
            if (vitalsHistory == null || vitalsHistory.isEmpty()) {
                return scores; // Cannot calculate without vitals
            }

            VitalSign vitals = vitalsHistory.getLatest();
            if (vitals == null) {
                return scores;
            }

            // MEWS (Modified Early Warning Score)
            // Scale: 0-14, threshold: ≥5 = significant deterioration, ≥7 = critical
            double mewsScore = calculateMEWS(vitals);
            scores.put("mews_score", mewsScore);

            // qSOFA (Quick Sequential Organ Failure Assessment)
            // Scale: 0-3, threshold: ≥2 = high mortality risk in sepsis
            double qsofaScore = calculateQSOFA(vitals);
            scores.put("qsofa_score", qsofaScore);

            // NEWS2 (National Early Warning Score 2)
            // Scale: 0-20, threshold: ≥7 = urgent response
            double news2Score = calculateNEWS2(vitals);
            scores.put("news2_score", news2Score);

            // SOFA (Sequential Organ Failure Assessment) - placeholder
            // Requires lab values, currently not fully implemented
            scores.put("sofa_score", 0.0);

            LOG.debug("Calculated scores for patient {}: MEWS={}, qSOFA={}, NEWS2={}",
                snapshot.getPatientId(), mewsScore, qsofaScore, news2Score);

            return scores;
        }

        /**
         * Calculate MEWS (Modified Early Warning Score).
         * Range: 0-14 points
         * Parameters: HR, SBP, RR, Temp, AVPU (consciousness)
         */
        private double calculateMEWS(VitalSign vitals) {
            int score = 0;

            // Heart rate scoring
            if (vitals.getHeartRate() != null) {
                double hr = vitals.getHeartRate();
                if (hr < 40) score += 2;
                else if (hr >= 40 && hr < 50) score += 1;
                else if (hr >= 50 && hr < 100) score += 0;
                else if (hr >= 100 && hr < 110) score += 1;
                else if (hr >= 110 && hr < 130) score += 2;
                else if (hr >= 130) score += 3;
            }

            // Systolic BP scoring
            if (vitals.getBloodPressureSystolic() != null) {
                double sbp = vitals.getBloodPressureSystolic();
                if (sbp < 70) score += 3;
                else if (sbp >= 70 && sbp < 80) score += 2;
                else if (sbp >= 80 && sbp < 100) score += 1;
                else if (sbp >= 100 && sbp < 200) score += 0;
                else if (sbp >= 200) score += 2;
            }

            // Respiratory rate scoring
            if (vitals.getRespiratoryRate() != null) {
                double rr = vitals.getRespiratoryRate();
                if (rr < 9) score += 2;
                else if (rr >= 9 && rr < 15) score += 0;
                else if (rr >= 15 && rr < 20) score += 1;
                else if (rr >= 20 && rr < 30) score += 2;
                else if (rr >= 30) score += 3;
            }

            // Temperature scoring
            if (vitals.getTemperature() != null) {
                double temp = vitals.getTemperature();
                if (temp < 35.0) score += 2;
                else if (temp >= 35.0 && temp < 38.5) score += 0;
                else if (temp >= 38.5) score += 2;
            }

            // AVPU (Alert/Voice/Pain/Unresponsive) - not available, assume Alert (0 points)
            // Future enhancement: integrate consciousness level from patient assessment

            return (double) score;
        }

        /**
         * Calculate qSOFA (Quick Sequential Organ Failure Assessment).
         * Range: 0-3 points
         * Criteria: RR ≥22, SBP ≤100, altered mentation
         */
        private double calculateQSOFA(VitalSign vitals) {
            int score = 0;

            // Respiratory rate ≥ 22/min
            if (vitals.getRespiratoryRate() != null && vitals.getRespiratoryRate() >= 22) {
                score++;
            }

            // Systolic BP ≤ 100 mmHg
            if (vitals.getBloodPressureSystolic() != null && vitals.getBloodPressureSystolic() <= 100) {
                score++;
            }

            // Altered mentation (GCS < 15)
            // Not available in current data - future enhancement
            // For now, assume normal mentation unless evidence otherwise

            return (double) score;
        }

        /**
         * Calculate NEWS2 (National Early Warning Score 2).
         * Range: 0-20 points
         * Parameters: RR, SpO2, supplemental O2, SBP, HR, consciousness, temperature
         */
        private double calculateNEWS2(VitalSign vitals) {
            int score = 0;

            // Respiratory rate scoring
            if (vitals.getRespiratoryRate() != null) {
                double rr = vitals.getRespiratoryRate();
                if (rr <= 8) score += 3;
                else if (rr >= 9 && rr <= 11) score += 1;
                else if (rr >= 12 && rr <= 20) score += 0;
                else if (rr >= 21 && rr <= 24) score += 2;
                else if (rr >= 25) score += 3;
            }

            // SpO2 Scale 1 scoring (assuming no hypercapnic respiratory failure)
            if (vitals.getOxygenSaturation() != null) {
                double spo2 = vitals.getOxygenSaturation();
                if (spo2 <= 91) score += 3;
                else if (spo2 >= 92 && spo2 <= 93) score += 2;
                else if (spo2 >= 94 && spo2 <= 95) score += 1;
                else if (spo2 >= 96) score += 0;
            }

            // Systolic BP scoring
            if (vitals.getBloodPressureSystolic() != null) {
                double sbp = vitals.getBloodPressureSystolic();
                if (sbp <= 90) score += 3;
                else if (sbp >= 91 && sbp <= 100) score += 2;
                else if (sbp >= 101 && sbp <= 110) score += 1;
                else if (sbp >= 111 && sbp <= 219) score += 0;
                else if (sbp >= 220) score += 3;
            }

            // Heart rate scoring
            if (vitals.getHeartRate() != null) {
                double hr = vitals.getHeartRate();
                if (hr <= 40) score += 3;
                else if (hr >= 41 && hr <= 50) score += 1;
                else if (hr >= 51 && hr <= 90) score += 0;
                else if (hr >= 91 && hr <= 110) score += 1;
                else if (hr >= 111 && hr <= 130) score += 2;
                else if (hr >= 131) score += 3;
            }

            // Temperature scoring
            if (vitals.getTemperature() != null) {
                double temp = vitals.getTemperature();
                if (temp <= 35.0) score += 3;
                else if (temp >= 35.1 && temp <= 36.0) score += 1;
                else if (temp >= 36.1 && temp <= 38.0) score += 0;
                else if (temp >= 38.1 && temp <= 39.0) score += 1;
                else if (temp >= 39.1) score += 2;
            }

            // Consciousness (ACVPU) - not available, assume Alert (0 points)
            // Supplemental oxygen - not tracked, assume no (0 points)

            return (double) score;
        }

        @Override
        public void close() throws Exception {
            if (fhirClient != null) {
                fhirClient.close();
            }
            LOG.info("PatientContextProcessorAsync shutdown complete");
        }
    }

    /**
     * Window function to create patient context snapshots
     */
    public static class PatientContextSnapshotFunction
            implements WindowFunction<EnrichedEvent, PatientContext, String, TimeWindow> {

        @Override
        public void apply(String patientId, TimeWindow window,
                         Iterable<EnrichedEvent> input, Collector<PatientContext> out) {

            PatientContext snapshot = null;
            int eventCount = 0;

            for (EnrichedEvent event : input) {
                eventCount++;
                if (snapshot == null) {
                    // Use the context from the first event as base
                    snapshot = event.getPatientContext().clone();
                } else {
                    // Update with latest context data
                    snapshot = event.getPatientContext().clone();
                }
            }

            if (snapshot != null) {
                // Add window metadata
                snapshot.setSnapshotWindowStart(window.getStart());
                snapshot.setSnapshotWindowEnd(window.getEnd());
                snapshot.setWindowEventCount(eventCount);
                snapshot.setSnapshotTime(System.currentTimeMillis());

                out.collect(snapshot);

                LOG.debug("Created context snapshot for patient {} with {} events",
                    patientId, eventCount);
            }
        }
    }

    /**
     * Create sink for enriched events
     */
    private static KafkaSink<EnrichedEvent> createEnrichedEventsSink() {
        return KafkaSink.<EnrichedEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.CLINICAL_PATTERNS.getTopicName())
                .setKeySerializationSchema((SerializationSchema<EnrichedEvent>) event ->
                    event.getPatientId() != null ? event.getPatientId().getBytes() : new byte[0])
                .setValueSerializationSchema(new EnrichedEventSerializer())
                .build())
            .setKafkaProducerConfig(getProducerPropertiesWithoutSerializers())
            .setTransactionalIdPrefix("module2-clinical-patterns")
            .build();
    }

    /**
     * Create sink for context snapshots
     */
    private static KafkaSink<PatientContext> createContextSnapshotsSink() {
        return KafkaSink.<PatientContext>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.PATIENT_CONTEXT_SNAPSHOTS.getTopicName())
                .setKeySerializationSchema((SerializationSchema<PatientContext>) context -> context.getPatientId().getBytes())
                .setValueSerializationSchema(new PatientContextSerializer())
                .build())
            .setKafkaProducerConfig(getProducerPropertiesWithoutSerializers())
            .setTransactionalIdPrefix("module2-patient-context")
            .build();
    }

    private static String getBootstrapServers() {
        return KafkaConfigLoader.isRunningInDocker()
            ? "kafka:29092"
            : "localhost:9092";
    }

    /**
     * Create Google FHIR client for async enrichment.
     */
    private static GoogleFHIRClient createFHIRClient() {
        try {
            String credentialsPath = KafkaConfigLoader.getGoogleCloudCredentialsPath();
            LOG.info("Creating Google FHIR client with credentials: {}", credentialsPath);

            GoogleFHIRClient client = new GoogleFHIRClient(
                KafkaConfigLoader.getGoogleCloudProjectId(),
                KafkaConfigLoader.getGoogleCloudLocation(),
                KafkaConfigLoader.getGoogleCloudDatasetId(),
                KafkaConfigLoader.getGoogleCloudFhirStoreId(),
                credentialsPath
            );
            client.initialize();
            LOG.info("Google FHIR client initialized successfully");
            return client;
        } catch (Exception e) {
            LOG.error("Failed to create FHIR client", e);
            throw new RuntimeException("FHIR client initialization failed", e);
        }
    }

    /**
     * Create Neo4j graph client for async enrichment (optional).
     */
    private static Neo4jGraphClient createNeo4jClient() {
        try {
            Neo4jGraphClient client = new Neo4jGraphClient(
                KafkaConfigLoader.getNeo4jUri(),
                KafkaConfigLoader.getNeo4jUsername(),
                KafkaConfigLoader.getNeo4jPassword()
            );
            client.initialize();
            LOG.info("Neo4j graph client initialized successfully");
            return client;
        } catch (Exception e) {
            LOG.warn("Neo4j client initialization failed - will continue without graph data: {}", e.getMessage());
            return null; // Graceful degradation
        }
    }

    /**
     * Get Kafka producer properties WITHOUT key/value serializers
     * (those are handled by RecordSerializationSchema)
     */
    private static java.util.Properties getProducerPropertiesWithoutSerializers() {
        java.util.Properties props = new java.util.Properties();

        // Producer optimizations
        props.setProperty("compression.type", "snappy");
        props.setProperty("batch.size", "32768"); // 32KB
        props.setProperty("linger.ms", "100");
        props.setProperty("buffer.memory", "33554432"); // 32MB
        props.setProperty("acks", "all");
        props.setProperty("enable.idempotence", "true");
        props.setProperty("retries", "2147483647");
        props.setProperty("max.in.flight.requests.per.connection", "5");
        props.setProperty("delivery.timeout.ms", "120000");

        return props;
    }

    // Helper classes for state management
    public static class MedicationState {
        private String medicationName;
        private long firstAdministered;
        private long lastAdministered;
        private int administrationCount = 0;

        // Getters and setters
        public String getMedicationName() { return medicationName; }
        public void setMedicationName(String medicationName) { this.medicationName = medicationName; }

        public long getFirstAdministered() { return firstAdministered; }
        public void setFirstAdministered(long firstAdministered) { this.firstAdministered = firstAdministered; }

        public long getLastAdministered() { return lastAdministered; }
        public void setLastAdministered(long lastAdministered) { this.lastAdministered = lastAdministered; }

        public int getAdministrationCount() { return administrationCount; }
        public void setAdministrationCount(int administrationCount) { this.administrationCount = administrationCount; }
    }

    public static class VitalSignTrend {
        private String vitalName;
        private List<Double> values = new ArrayList<>();
        private List<Long> timestamps = new ArrayList<>();

        public void addValue(double value, long timestamp) {
            values.add(value);
            timestamps.add(timestamp);

            // Keep only last 20 values
            if (values.size() > 20) {
                values.remove(0);
                timestamps.remove(0);
            }
        }

        public double getTrend() {
            if (values.size() < 2) return 0.0;

            // Simple linear trend calculation
            double sum = 0.0;
            for (int i = 1; i < values.size(); i++) {
                sum += values.get(i) - values.get(i-1);
            }
            return sum / (values.size() - 1);
        }

        // Getters and setters
        public String getVitalName() { return vitalName; }
        public void setVitalName(String vitalName) { this.vitalName = vitalName; }

        public List<Double> getValues() { return values; }
        public List<Long> getTimestamps() { return timestamps; }
    }

    // Serialization classes
    private static class CanonicalEventDeserializer implements org.apache.flink.api.common.serialization.DeserializationSchema<CanonicalEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(org.apache.flink.api.common.serialization.DeserializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public CanonicalEvent deserialize(byte[] message) throws java.io.IOException {
            return objectMapper.readValue(message, CanonicalEvent.class);
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

    private static class EnrichedEventSerializer implements SerializationSchema<EnrichedEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(EnrichedEvent element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                throw new RuntimeException("Failed to serialize EnrichedEvent", e);
            }
        }
    }

    private static class PatientContextSerializer implements SerializationSchema<PatientContext> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(PatientContext element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                throw new RuntimeException("Failed to serialize PatientContext", e);
            }
        }
    }
}
