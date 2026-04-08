package com.cardiofit.flink.operators;

import com.cardiofit.flink.alerts.SmartAlertGenerator;
import com.cardiofit.flink.clients.GoogleFHIRClient;
import com.cardiofit.flink.clients.Neo4jGraphClient;
import com.cardiofit.flink.models.ProtocolEvent;
import com.cardiofit.flink.serialization.ProtocolEventSerializer;
import com.cardiofit.flink.enrichment.AdvancedNeo4jEnricher;
import com.cardiofit.flink.enrichment.AdvancedNeo4jEnricher.*;
import com.cardiofit.flink.indicators.EnhancedRiskIndicators;
import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.neo4j.AdvancedNeo4jQueries;
import com.cardiofit.flink.neo4j.CohortInsights;
import com.cardiofit.flink.neo4j.SimilarPatient;
import com.cardiofit.flink.protocols.ProtocolMatcher;
import com.cardiofit.flink.protocols.ProtocolMatcher.Protocol;
import com.cardiofit.flink.recommendations.RecommendationEngine;
import com.cardiofit.flink.recommendations.Recommendations;
import com.cardiofit.flink.models.ClinicalIntelligence;
import com.cardiofit.flink.models.Condition;
import com.cardiofit.flink.models.EnrichedEvent;
import com.cardiofit.flink.models.FHIRPatientData;
import com.cardiofit.flink.models.GraphData;
import com.cardiofit.flink.models.Medication;
import com.cardiofit.flink.models.PatientContext;
import com.cardiofit.flink.models.PatientSnapshot;
import com.cardiofit.flink.models.RiskIndicators;
import com.cardiofit.flink.models.SimpleAlert;
import com.cardiofit.flink.models.AlertType;
import com.cardiofit.flink.models.AlertSeverity;
import com.cardiofit.flink.models.TrendDirection;
import com.cardiofit.flink.models.LabResult;

// Phase 6: Unified Pipeline imports
import com.cardiofit.flink.models.GenericEvent;
import com.cardiofit.flink.models.VitalsPayload;
import com.cardiofit.flink.models.LabPayload;
import com.cardiofit.flink.models.MedicationPayload;
import com.cardiofit.flink.models.PatientContextState;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.scoring.ClinicalScoreCalculators;
import com.cardiofit.flink.scoring.CombinedAcuityCalculator;
import com.cardiofit.flink.scoring.ConfidenceScoreCalculator;
import com.cardiofit.flink.scoring.MetabolicAcuityCalculator;
import com.cardiofit.flink.scoring.NEWS2Calculator;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.state.*;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.AsyncDataStream;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.api.common.functions.FlatMapFunction;
import org.apache.flink.streaming.api.functions.async.ResultFuture;
import org.apache.flink.streaming.api.functions.async.RichAsyncFunction;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.time.Duration;
import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;
import java.util.stream.Collectors;

/**
 * Enhanced Module 2: Advanced Context Assembly, Protocol Matching & Recommendations
 *
 * This comprehensive module combines:
 * - Patient context enrichment from FHIR and Neo4j
 * - Clinical protocol matching based on conditions
 * - Similar patient analysis and cohort statistics
 * - Predictive analytics and trajectory prediction
 * - Comprehensive recommendation generation
 *
 * All operations are performed asynchronously for optimal performance.
 */
public class Module2_Enhanced {
    private static final Logger LOG = LoggerFactory.getLogger(Module2_Enhanced.class);

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Enhanced Module 2: Unified Clinical Reasoning Pipeline");

        // Set up execution environment
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Configure for high-throughput stateful processing
        env.setParallelism(2);  // Match Module1 parallelism for resource efficiency
        env.enableCheckpointing(30000);
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);
        env.getCheckpointConfig().setCheckpointTimeout(600000);

        // Create unified pipeline (Phases 1-6: Unified State Management)
        createUnifiedPipeline(env);

        // Execute the job
        env.execute("Enhanced Module 2: Unified Clinical Reasoning Pipeline");
    }

    public static void createEnhancedPipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating enhanced context assembly pipeline with recommendations");

        // Consume canonical events from Module 1
        DataStream<CanonicalEvent> canonicalEvents = createCanonicalEventSource(env);

        // INTEGRATED Phase 1 + Phase 2 Enrichment
        // This includes FHIR lookup, Neo4j graph data, Phase 1 clinical intelligence,
        // and Phase 2 protocols/recommendations - all in ONE comprehensive pass
        DataStream<EnrichedEvent> enrichedEvents = AsyncDataStream.unorderedWait(
            canonicalEvents,
            new ComprehensiveEnrichmentFunction(),
            10000,  // 10 second timeout for comprehensive enrichment
            TimeUnit.MILLISECONDS,
            500     // Higher capacity for parallel processing
        ).uid("comprehensive-enrichment");

        // Output sinks for different data products

        // 1. Enriched events with full Phase 1 + Phase 2 context
        enrichedEvents
            .sinkTo(createEnrichedEventsSink())
            .uid("enriched-events-sink");

        // PHASE 4 Enhancement: Extract and emit protocol trigger events to separate topic
        DataStream<ProtocolEvent> protocolEventStream = enrichedEvents
            .flatMap(new FlatMapFunction<EnrichedEvent, ProtocolEvent>() {
                @Override
                public void flatMap(EnrichedEvent enrichedEvent, Collector<ProtocolEvent> out) {
                    // Extract protocol events from enriched event's enrichment data
                    if (enrichedEvent.getEnrichmentData() != null &&
                        enrichedEvent.getEnrichmentData().containsKey("protocol_events")) {

                        Object protocolEventsObj = enrichedEvent.getEnrichmentData().get("protocol_events");
                        if (protocolEventsObj instanceof List) {
                            List<?> eventsList = (List<?>) protocolEventsObj;
                            for (Object eventObj : eventsList) {
                                if (eventObj instanceof ProtocolEvent) {
                                    out.collect((ProtocolEvent) eventObj);
                                }
                            }
                        }
                    }
                }
            })
            .uid("extract-protocol-events");

        // 2. Protocol trigger events for audit trail
        protocolEventStream
            .sinkTo(createProtocolEventsSink())
            .uid("protocol-events-sink");

        LOG.info("Enhanced pipeline created successfully with all components integrated including protocol audit trail");
    }

    /**
     * Comprehensive Enrichment Function
     * Performs FHIR lookup, Neo4j enrichment, and initial context building
     */
    public static class ComprehensiveEnrichmentFunction
            extends RichAsyncFunction<CanonicalEvent, EnrichedEvent> {

        private transient GoogleFHIRClient fhirClient;
        private transient Neo4jGraphClient neo4jClient;
        private transient ObjectMapper objectMapper;

        @Override
        public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
            LOG.info("Initializing Comprehensive Enrichment Function");

            // Initialize FHIR client using same method as old Module2
            String credentialsPath = com.cardiofit.flink.utils.KafkaConfigLoader.getGoogleCloudCredentialsPath();
            LOG.info("Loading Google Cloud credentials from: {}", credentialsPath);

            fhirClient = new GoogleFHIRClient(
                com.cardiofit.flink.utils.KafkaConfigLoader.getGoogleCloudProjectId(),
                com.cardiofit.flink.utils.KafkaConfigLoader.getGoogleCloudLocation(),
                com.cardiofit.flink.utils.KafkaConfigLoader.getGoogleCloudDatasetId(),
                com.cardiofit.flink.utils.KafkaConfigLoader.getGoogleCloudFhirStoreId(),
                credentialsPath
            );
            fhirClient.initialize();
            LOG.info("GoogleFHIRClient initialized successfully");

            // Initialize Neo4j client — graceful degradation if credentials missing
            String neo4jPassword = System.getenv("NEO4J_PASSWORD");
            if (neo4jPassword == null || neo4jPassword.isEmpty()) {
                LOG.warn("NEO4J_PASSWORD not set — Neo4j enrichment disabled (circuit breaker mode). "
                    + "Graph data (careTeam, riskFactors, carePathways) will be empty.");
                neo4jClient = null;
            } else {
                try {
                    String neo4jUri = com.cardiofit.flink.utils.KafkaConfigLoader.getNeo4jUri();
                    String neo4jUser = System.getenv().getOrDefault("NEO4J_USER", "neo4j");
                    LOG.info("Connecting to Neo4j at: {}", neo4jUri);
                    neo4jClient = new Neo4jGraphClient(neo4jUri, neo4jUser, neo4jPassword);
                    neo4jClient.initialize();
                    LOG.info("Neo4jGraphClient initialized successfully");
                } catch (Exception e) {
                    LOG.warn("Neo4j initialization failed — enrichment disabled: {}", e.getMessage());
                    neo4jClient = null;
                }
            }

            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
            objectMapper.configure(com.fasterxml.jackson.databind.DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
        }

        @Override
        public void asyncInvoke(CanonicalEvent event, ResultFuture<EnrichedEvent> resultFuture) {
            String patientId = event.getPatientId();

            // Parallel enrichment from multiple sources
            CompletableFuture<Map<String, Object>> fhirDataFuture =
                fetchFHIRData(patientId);

            CompletableFuture<Map<String, Object>> graphDataFuture =
                fetchGraphData(patientId);

            // Combine all futures (demographics extracted from FHIR data after fetch)
            CompletableFuture.allOf(fhirDataFuture, graphDataFuture)
                .thenAccept(v -> {
                    try {
                        EnrichedEvent enrichedEvent = new EnrichedEvent();
                        enrichedEvent.setId(event.getId());
                        enrichedEvent.setPatientId(patientId);
                        enrichedEvent.setEventType(event.getEventType());
                        enrichedEvent.setEventTime(event.getEventTime());
                        enrichedEvent.setPayload(event.getPayload());

                        // Combine all enrichment data
                        Map<String, Object> enrichmentData = new HashMap<>();
                        enrichmentData.putAll(fhirDataFuture.join());
                        enrichmentData.putAll(graphDataFuture.join());

                        // Extract demographics from FHIR patient data (single source of truth)
                        enrichmentData.put("age", extractAgeFromEnrichmentData(enrichmentData));
                        enrichmentData.put("gender", extractGenderFromEnrichmentData(enrichmentData));

                        // Add clinical context
                        enrichmentData.put("clinicalContext", buildClinicalContext(event, enrichmentData));

                        // Build and set PatientContext
                        PatientContext patientContext = buildPatientContextFromEnrichment(patientId, enrichmentData, event);
                        enrichedEvent.setPatientContext(patientContext);

                        // CRITICAL FIX: Extract top-level fields from clinicalIntelligence
                        // These fields were being calculated but not surfaced to EnrichedEvent
                        extractAndPopulateTopLevelFields(enrichedEvent, enrichmentData);

                        enrichedEvent.setEnrichmentData(enrichmentData);
                        enrichedEvent.setProcessingTime(System.currentTimeMillis());

                        resultFuture.complete(Collections.singleton(enrichedEvent));
                    } catch (Exception e) {
                        LOG.error("Error in comprehensive enrichment", e);
                        // Return original event with minimal enrichment on error
                        EnrichedEvent fallback = createFallbackEnrichedEvent(event);
                        resultFuture.complete(Collections.singleton(fallback));
                    }
                })
                .exceptionally(throwable -> {
                    LOG.error("Comprehensive enrichment failed", throwable);
                    EnrichedEvent fallback = createFallbackEnrichedEvent(event);
                    resultFuture.complete(Collections.singleton(fallback));
                    return null;
                });
        }

        private CompletableFuture<Map<String, Object>> fetchFHIRData(String patientId) {
            return CompletableFuture.supplyAsync(() -> {
                Map<String, Object> fhirData = new HashMap<>();
                try {
                    // Fetch patient resource (async call with .get() to block)
                    FHIRPatientData patient = fhirClient.getPatientAsync(patientId).get(500, java.util.concurrent.TimeUnit.MILLISECONDS);
                    if (patient != null) {
                        fhirData.put("patient", patient);
                        fhirData.put("isFirstTime", false);
                    } else {
                        fhirData.put("isFirstTime", true);
                    }

                    // Fetch recent conditions (async)
                    List<Condition> conditions = fhirClient.getConditionsAsync(patientId).get(500, java.util.concurrent.TimeUnit.MILLISECONDS);
                    fhirData.put("conditions", conditions);
                    fhirData.put("diagnoses", extractDiagnosisCodesFromConditions(conditions));

                    // Fetch medications (async)
                    List<Medication> medications = fhirClient.getMedicationsAsync(patientId).get(500, java.util.concurrent.TimeUnit.MILLISECONDS);
                    fhirData.put("medications", extractMedicationNamesFromMedications(medications));

                    // Note: Observations would need separate vitals/labs endpoints or parsing
                    // For now, create empty lists to avoid null issues
                    fhirData.put("vitalSigns", new java.util.ArrayList<>());
                    fhirData.put("labResults", new java.util.ArrayList<>());

                } catch (Exception e) {
                    LOG.error("FHIR data fetch failed for patient {}", patientId, e);
                }
                return fhirData;
            });
        }

        private CompletableFuture<Map<String, Object>> fetchGraphData(String patientId) {
            // Use existing queryGraphAsync method from Neo4jGraphClient
            if (neo4jClient == null) {
                return CompletableFuture.completedFuture(new HashMap<>());
            }

            return neo4jClient.queryGraphAsync(patientId)
                .thenApply(graphData -> {
                    Map<String, Object> result = new HashMap<>();
                    if (graphData != null) {
                        result.put("careTeam", graphData.getCareTeam());
                        result.put("riskFactors", graphData.getRiskCohorts()); // Risk cohorts from GraphData
                        result.put("carePathways", graphData.getCarePathways());

                        // Extract risk factors from risk cohorts
                        if (graphData.getRiskCohorts() != null) {
                            List<String> riskFactors = new ArrayList<>(graphData.getRiskCohorts());
                            result.put("riskFactors", riskFactors);
                        }
                    }
                    return result;
                })
                .exceptionally(throwable -> {
                    LOG.warn("Graph data fetch failed for patient {}: {}", patientId, throwable.getMessage());
                    return new HashMap<>();
                });
        }

        private CompletableFuture<Map<String, Object>> fetchDemographics(String patientId, Map<String, Object> enrichmentData) {
            return CompletableFuture.supplyAsync(() -> {
                Map<String, Object> demographics = new HashMap<>();
                try {
                    // Extract from FHIR patient resource or use defaults
                    demographics.put("age", extractAgeFromEnrichmentData(enrichmentData));
                    demographics.put("gender", extractGenderFromEnrichmentData(enrichmentData));
                    demographics.put("location", getLocation(patientId));
                } catch (Exception e) {
                    LOG.warn("Demographics fetch failed for patient {}: {}", patientId, e.getMessage());
                }
                return demographics;
            });
        }

        private Map<String, Object> buildClinicalContext(CanonicalEvent event,
                                                         Map<String, Object> enrichmentData) {
            Map<String, Object> context = new HashMap<>();

            // Build PatientSnapshot from enrichment data
            PatientSnapshot snapshot = buildPatientSnapshot(event.getPatientId(), enrichmentData);

            // Extract vitals and labs for Phase 1 components
            Map<String, Object> vitals = extractVitalsMap(event, enrichmentData);
            Map<String, Object> labs = extractLabsMap(enrichmentData);

            // Phase 1: Critical Clinical Intelligence
            ClinicalIntelligence clinicalIntelligence = calculateClinicalIntelligence(
                snapshot, vitals, labs, event.getPatientId());

            // Add clinical intelligence to context
            context.put("clinicalIntelligence", clinicalIntelligence);
            context.put("urgency", clinicalIntelligence.getOverallUrgency());
            context.put("requiresImmediateAttention", clinicalIntelligence.requiresImmediateAttention());
            context.put("summaryFindings", clinicalIntelligence.getSummaryFindings());

            // PHASE 4 Enhancement: Generate protocol trigger events for audit trail
            if (clinicalIntelligence.getApplicableProtocols() != null &&
                !clinicalIntelligence.getApplicableProtocols().isEmpty()) {
                List<ProtocolEvent> protocolEvents = generateProtocolEvents(
                    event.getPatientId(),
                    event.getEncounterId(),
                    event.getId(),
                    clinicalIntelligence.getApplicableProtocols(),
                    clinicalIntelligence.getRiskAssessment(),
                    clinicalIntelligence.getNews2Score(),
                    clinicalIntelligence.getQsofaScore()
                );
                // Store in enrichment data for extraction and emission to protocol-triggers topic
                enrichmentData.put("protocol_events", protocolEvents);
            }

            // Determine clinical urgency (legacy)
            context.put("urgencyLegacy", determineUrgency(event, enrichmentData));

            // Identify active problems
            context.put("activeProblems", identifyActiveProblems(enrichmentData));

            // Calculate risk scores (legacy - now replaced by Phase 1 components)
            context.put("riskScores", calculateRiskScores(enrichmentData));

            return context;
        }

        /**
         * Build PatientContext from enrichment data
         */
        private PatientContext buildPatientContextFromEnrichment(
                String patientId,
                Map<String, Object> enrichmentData,
                CanonicalEvent event) {

            PatientContext context = new PatientContext();
            context.setPatientId(patientId);
            context.setLastEventTime(event.getEventTime());
            context.setFirstEventTime(event.getEventTime()); // Will be overwritten if historical data exists

            // Demographics from FHIR patient data (use single source of truth from enrichmentData)
            if (enrichmentData.containsKey("patient")) {
                Object patientObj = enrichmentData.get("patient");
                if (patientObj instanceof FHIRPatientData) {
                    FHIRPatientData patientData = (FHIRPatientData) patientObj;
                    PatientContext.PatientDemographics demographics = new PatientContext.PatientDemographics();
                    // Use age from enrichmentData (already extracted consistently)
                    demographics.setAge((Integer) enrichmentData.get("age"));
                    // Use gender from enrichmentData (already extracted consistently)
                    demographics.setGender((String) enrichmentData.getOrDefault("gender", "unknown"));
                    context.setDemographics(demographics);
                }
            }

            // Location from enrichment data (if available)
            if (enrichmentData.containsKey("location")) {
                String locationStr = extractString(enrichmentData, "location", "unknown");
                PatientContext.PatientLocation location = new PatientContext.PatientLocation();
                location.setFacility(locationStr);
                context.setLocation(location);
            }

            // Active medications
            if (enrichmentData.containsKey("medications")) {
                Map<String, Object> medications = new HashMap<>();
                List<String> medList = (List<String>) enrichmentData.get("medications");
                if (medList != null) {
                    for (int i = 0; i < medList.size(); i++) {
                        medications.put("med_" + i, medList.get(i));
                    }
                }
                context.setActiveMedications(medications);
            }

            // Current vitals (from payload if vital_signs event)
            if (event.getPayload() != null) {
                context.setCurrentVitals(event.getPayload());
            }

            // Risk factors from graph data
            if (enrichmentData.containsKey("riskFactors")) {
                context.setRiskFactors((List<String>) enrichmentData.get("riskFactors"));
            }

            // Chronic/Active conditions from both diagnoses and conditions lists (ensure empty list instead of null)
            List<String> conditionCodes = new ArrayList<>();

            // Add from diagnoses (FHIR diagnostic codes)
            if (enrichmentData.containsKey("diagnoses")) {
                List<String> diagnoses = (List<String>) enrichmentData.get("diagnoses");
                if (diagnoses != null) {
                    conditionCodes.addAll(diagnoses);
                }
            }

            // Add from conditions list (Condition objects with codes)
            if (enrichmentData.containsKey("conditions")) {
                Object conditionsObj = enrichmentData.get("conditions");
                if (conditionsObj instanceof List) {
                    List<Condition> conditions = (List<Condition>) conditionsObj;
                    for (Condition condition : conditions) {
                        if (condition.getCode() != null && !conditionCodes.contains(condition.getCode())) {
                            conditionCodes.add(condition.getCode());
                        }
                    }
                }
            }

            // Set chronic conditions (this populates both chronic_conditions and active_conditions in output)
            context.setChronicConditions(conditionCodes);

            // Allergies (ensure empty list instead of null)
            if (enrichmentData.containsKey("allergies")) {
                List<String> allergies = (List<String>) enrichmentData.get("allergies");
                context.setAllergies(allergies != null ? allergies : new ArrayList<>());
            } else {
                context.setAllergies(new ArrayList<>());
            }

            // Care team from graph
            if (enrichmentData.containsKey("careTeam")) {
                context.setCareTeam((List<String>) enrichmentData.get("careTeam"));
            }

            // Risk cohorts
            if (enrichmentData.containsKey("riskCohorts")) {
                context.setRiskCohorts((List<String>) enrichmentData.get("riskCohorts"));
            }

            // Acuity score from clinical intelligence
            if (enrichmentData.containsKey("clinicalContext")) {
                Object clinicalContextObj = enrichmentData.get("clinicalContext");
                if (clinicalContextObj instanceof Map) {
                    Map<String, Object> clinicalContext = (Map<String, Object>) clinicalContextObj;
                    if (clinicalContext.containsKey("clinicalIntelligence")) {
                        Object intelligenceObj = clinicalContext.get("clinicalIntelligence");
                        if (intelligenceObj instanceof ClinicalIntelligence) {
                            ClinicalIntelligence intelligence = (ClinicalIntelligence) intelligenceObj;
                            if (intelligence.getCombinedAcuityScore() != null) {
                                context.setAcuityScore(intelligence.getCombinedAcuityScore().getCombinedAcuityScore());
                            }
                        }
                    }
                }
            }

            return context;
        }

        /**
         * Calculate comprehensive clinical intelligence using all Phase 1 components
         */
        private ClinicalIntelligence calculateClinicalIntelligence(
                PatientSnapshot snapshot,
                Map<String, Object> vitals,
                Map<String, Object> labs,
                String patientId) {

            try {
                // 1. Enhanced Risk Assessment
                EnhancedRiskIndicators.RiskAssessment riskAssessment =
                    EnhancedRiskIndicators.assessRisk(snapshot, vitals);

                // 2. NEWS2 Scoring
                boolean isOnOxygen = extractBoolean(vitals, "supplementalOxygen", false);
                NEWS2Calculator.NEWS2Score news2Score =
                    NEWS2Calculator.calculate(vitals, isOnOxygen);

                // 3. Metabolic Acuity Scoring (NEW - Gap 1)
                MetabolicAcuityCalculator.MetabolicAcuityScore metabolicAcuityScore =
                    MetabolicAcuityCalculator.calculate(snapshot, vitals, labs);

                // 4. Combined Acuity Score (NEW - Gap 2)
                // Weighted combination: (0.7 * NEWS2) + (0.3 * Metabolic)
                CombinedAcuityCalculator.CombinedAcuityScore combinedAcuityScore =
                    CombinedAcuityCalculator.calculate(news2Score, metabolicAcuityScore);

                // 5. Smart Alert Generation
                List<SmartAlertGenerator.ClinicalAlert> alerts =
                    SmartAlertGenerator.generateAlerts(patientId, riskAssessment, news2Score, vitals);

                // 6. Clinical Scores
                ClinicalScoreCalculators.FraminghamScore framinghamScore = null;
                ClinicalScoreCalculators.CHADS2VAScScore chadsVascScore = null;
                ClinicalScoreCalculators.qSOFAScore qsofaScore = null;
                ClinicalScoreCalculators.MetabolicSyndromeScore metabolicSyndromeScore = null;

                // Only calculate Framingham if sufficient data
                if (hasFraminghamData(snapshot, labs)) {
                    framinghamScore = ClinicalScoreCalculators.calculateFraminghamScore(snapshot, labs);
                }

                // Only calculate CHADS-VASc if patient has relevant conditions
                if (hasAtrialFibrillation(snapshot)) {
                    chadsVascScore = ClinicalScoreCalculators.calculateCHADS2VAScScore(snapshot);
                }

                // qSOFA can be calculated with vitals alone
                qsofaScore = ClinicalScoreCalculators.calculateQSOFAScore(vitals);

                // Metabolic Syndrome Risk Score (NEW - Gap 3)
                metabolicSyndromeScore = ClinicalScoreCalculators.calculateMetabolicSyndromeScore(
                    snapshot, vitals, labs);

                // 7. Confidence Scoring
                String assessmentType = determineAssessmentType(snapshot, vitals);
                ConfidenceScoreCalculator.ConfidenceScore confidenceScore =
                    ConfidenceScoreCalculator.calculateConfidence(snapshot, vitals, labs, assessmentType);

                // ═══════════════════════════════════════════════════════════
                // PHASE 2: Advanced Context & Recommendations
                // ═══════════════════════════════════════════════════════════

                // 8. Clinical Protocol Matching
                List<Protocol> applicableProtocols = ProtocolMatcher.matchProtocols(
                    snapshot,
                    riskAssessment,
                    news2Score,
                    metabolicSyndromeScore,
                    qsofaScore
                );

                // PHASE 4 Enhancement: Protocol events will be generated after we have encounter_id
                // Note: Protocol events are stored later in the enrichmentData map

                // 9. Similar Patient Analysis (requires Neo4j - may be null)
                List<SimilarPatient> similarPatients = null;
                CohortInsights cohortInsights = null;
                Map<String, Integer> interventionSuccessMap = null;

                try {
                    if (neo4jClient != null && neo4jClient.getDriver() != null) {
                        AdvancedNeo4jQueries advancedQueries = new AdvancedNeo4jQueries(neo4jClient.getDriver());

                        // Find similar patients (top 3)
                        similarPatients = advancedQueries.findSimilarPatients(patientId, snapshot, 3);

                        // Get cohort analytics
                        cohortInsights = advancedQueries.getCohortAnalytics(patientId);

                        // Extract successful interventions from similar patients
                        interventionSuccessMap = advancedQueries.findSuccessfulInterventions(patientId, snapshot);
                    }
                } catch (Exception e) {
                    LOG.warn("Could not fetch advanced Neo4j data for patient {}: {}", patientId, e.getMessage());
                }

                // 10. Intelligent Recommendations
                Recommendations recommendations = RecommendationEngine.generateRecommendations(
                    snapshot,
                    riskAssessment,
                    combinedAcuityScore,
                    alerts,
                    applicableProtocols,
                    similarPatients,
                    interventionSuccessMap
                );

                // ═══════════════════════════════════════════════════════════
                // Bundle all Phase 1 + Phase 2 outputs
                // ═══════════════════════════════════════════════════════════
                ClinicalIntelligence intelligence = new ClinicalIntelligence();

                // Phase 1 Components
                intelligence.setRiskAssessment(riskAssessment);
                intelligence.setNews2Score(news2Score);
                intelligence.setMetabolicAcuityScore(metabolicAcuityScore);
                intelligence.setCombinedAcuityScore(combinedAcuityScore);
                intelligence.setAlerts(alerts);
                intelligence.setFraminghamScore(framinghamScore);
                intelligence.setChadsVascScore(chadsVascScore);
                intelligence.setQsofaScore(qsofaScore);
                intelligence.setMetabolicSyndromeScore(metabolicSyndromeScore);
                intelligence.setConfidenceScore(confidenceScore);

                // Phase 2 Components
                intelligence.setApplicableProtocols(applicableProtocols);
                intelligence.setSimilarPatients(similarPatients);
                intelligence.setCohortInsights(cohortInsights);
                intelligence.setRecommendations(recommendations);

                // Metadata
                intelligence.setPatientId(patientId);
                intelligence.setCalculationTimestamp(System.currentTimeMillis());

                LOG.info("Complete clinical intelligence (Phase 1 + Phase 2) calculated for patient {}: " +
                        "urgency={}, combinedAcuity={}, NEWS2={}, metabolic={}, alerts={}, protocols={}, " +
                        "similarPatients={}, recommendations={}, confidence={}",
                    patientId,
                    intelligence.getOverallUrgency(),
                    combinedAcuityScore.getCombinedAcuityScore(),
                    news2Score.getTotalScore(),
                    metabolicAcuityScore.getScore(),
                    alerts.size(),
                    applicableProtocols != null ? applicableProtocols.size() : 0,
                    similarPatients != null ? similarPatients.size() : 0,
                    recommendations != null ? recommendations.getImmediateActions().size() + " actions" : "none",
                    confidenceScore.getConfidenceLevel());

                return intelligence;

            } catch (Exception e) {
                LOG.error("Error calculating clinical intelligence for patient {}: {}", patientId, e.getMessage());
                // Return minimal intelligence on error
                return new ClinicalIntelligence();
            }
        }

        /**
         * Build PatientSnapshot from enrichment data
         */
        private PatientSnapshot buildPatientSnapshot(String patientId, Map<String, Object> enrichmentData) {
            PatientSnapshot snapshot = new PatientSnapshot(patientId);

            // Extract demographics from FHIRPatientData object
            if (enrichmentData.containsKey("patient")) {
                Object patientObj = enrichmentData.get("patient");
                if (patientObj instanceof FHIRPatientData) {
                    FHIRPatientData patientData = (FHIRPatientData) patientObj;
                    snapshot.setAge(patientData.getAge());
                    snapshot.setGender(patientData.getGender());
                    snapshot.setDateOfBirth(patientData.getDateOfBirth());
                }
            }

            // Extract conditions from List<Condition>
            if (enrichmentData.containsKey("conditions")) {
                Object conditionsObj = enrichmentData.get("conditions");
                if (conditionsObj instanceof List) {
                    snapshot.setActiveConditions((List<Condition>) conditionsObj);
                }
            }

            // Extract medications from List<String> (medication names already extracted)
            if (enrichmentData.containsKey("medications")) {
                List<String> medNames = (List<String>) enrichmentData.get("medications");
                List<Medication> medications = new ArrayList<>();
                for (String medName : medNames) {
                    Medication med = new Medication();
                    med.setName(medName);
                    medications.add(med);
                }
                snapshot.setActiveMedications(medications);
            }

            // Extract allergies
            if (enrichmentData.containsKey("allergies")) {
                snapshot.setAllergies((List<String>) enrichmentData.get("allergies"));
            }

            // Extract risk cohorts from graph data
            if (enrichmentData.containsKey("riskFactors")) {
                snapshot.setRiskCohorts((List<String>) enrichmentData.get("riskFactors"));
            }

            return snapshot;
        }

        /**
         * Extract vitals as a unified map for Phase 1 components
         * Normalizes lowercase field names from event payload to camelCase for scoring algorithms
         */
        private Map<String, Object> extractVitalsMap(CanonicalEvent event, Map<String, Object> enrichmentData) {
            Map<String, Object> vitals = new HashMap<>();

            // First, extract from enrichmentData.vitalSigns if present
            if (enrichmentData.containsKey("vitalSigns")) {
                Object vitalSignsObj = enrichmentData.get("vitalSigns");
                // Handle both Map and List representations
                if (vitalSignsObj instanceof Map) {
                    Map<String, Object> vitalSigns = (Map<String, Object>) vitalSignsObj;
                    vitals.putAll(vitalSigns);
                }
                // Skip if it's a List - we'll get vitals from event payload instead
            }

            // Then extract from event payload and normalize field names
            // This handles lowercase field names from incoming events
            if (event != null && event.getPayload() != null) {
                Map<String, Object> payload = event.getPayload();

                // Map lowercase to camelCase expected by scoring algorithms
                if (payload.containsKey("respiratoryrate")) vitals.put("respiratoryRate", payload.get("respiratoryrate"));
                if (payload.containsKey("heartrate")) vitals.put("heartRate", payload.get("heartrate"));
                if (payload.containsKey("oxygensaturation")) vitals.put("oxygenSaturation", payload.get("oxygensaturation"));
                if (payload.containsKey("systolicbp")) vitals.put("systolicBP", payload.get("systolicbp"));
                if (payload.containsKey("diastolicbp")) vitals.put("diastolicBP", payload.get("diastolicbp"));
                if (payload.containsKey("temperature")) vitals.put("temperature", payload.get("temperature"));
                if (payload.containsKey("consciousness")) vitals.put("consciousness", payload.get("consciousness"));
                if (payload.containsKey("supplementaloxygen")) vitals.put("supplementalOxygen", payload.get("supplementaloxygen"));

                // Also support camelCase field names if already normalized
                if (payload.containsKey("respiratoryRate")) vitals.put("respiratoryRate", payload.get("respiratoryRate"));
                if (payload.containsKey("heartRate")) vitals.put("heartRate", payload.get("heartRate"));
                if (payload.containsKey("oxygenSaturation")) vitals.put("oxygenSaturation", payload.get("oxygenSaturation"));
                if (payload.containsKey("systolicBP")) vitals.put("systolicBP", payload.get("systolicBP"));
                if (payload.containsKey("diastolicBP")) vitals.put("diastolicBP", payload.get("diastolicBP"));
                if (payload.containsKey("supplementalOxygen")) vitals.put("supplementalOxygen", payload.get("supplementalOxygen"));
            }

            // Add timestamp if not present
            if (!vitals.containsKey("timestamp")) {
                vitals.put("timestamp", System.currentTimeMillis());
            }

            return vitals;
        }

        /**
         * Extract labs as a unified map for Phase 1 components
         */
        private Map<String, Object> extractLabsMap(Map<String, Object> enrichmentData) {
            Map<String, Object> labs = new HashMap<>();

            if (enrichmentData.containsKey("labResults")) {
                Object labResultsObj = enrichmentData.get("labResults");

                // Handle both Map and List formats
                if (labResultsObj instanceof Map) {
                    Map<String, Object> labResults = (Map<String, Object>) labResultsObj;
                    labs.putAll(labResults);
                } else if (labResultsObj instanceof java.util.List) {
                    // labResults is a List - convert to Map or handle appropriately
                    // For now, return empty map to avoid crash
                    LOG.debug("labResults is a List, not a Map. Returning empty labs map.");
                }
            }

            return labs;
        }

        /**
         * Check if patient has sufficient data for Framingham score
         */
        private boolean hasFraminghamData(PatientSnapshot snapshot, Map<String, Object> labs) {
            return snapshot.getAge() != null &&
                   labs.containsKey("totalCholesterol") &&
                   labs.containsKey("hdlCholesterol");
        }

        /**
         * Check if patient has atrial fibrillation (for CHADS-VASc)
         */
        private boolean hasAtrialFibrillation(PatientSnapshot snapshot) {
            if (snapshot.getActiveConditions() == null) {
                return false;
            }
            return snapshot.getActiveConditions().stream()
                .anyMatch(c -> c.getCode() != null && c.getCode().contains("I48"));
        }

        /**
         * Determine assessment type for confidence scoring
         */
        private String determineAssessmentType(PatientSnapshot snapshot, Map<String, Object> vitals) {
            // Determine primary assessment type based on data availability
            if (vitals.containsKey("respiratoryRate") &&
                vitals.containsKey("oxygenSaturation") &&
                vitals.containsKey("heartRate")) {
                return "NEWS2";
            } else if (vitals.containsKey("heartRate") || vitals.containsKey("systolicBP")) {
                return "CARDIAC";
            } else {
                return "COMPREHENSIVE";
            }
        }

        // Helper methods for data extraction

        private Integer extractInteger(Map<String, Object> map, String key) {
            Object value = map.get(key);
            if (value == null) return null;
            if (value instanceof Integer) return (Integer) value;
            if (value instanceof Number) return ((Number) value).intValue();
            try {
                return Integer.parseInt(value.toString());
            } catch (NumberFormatException e) {
                return null;
            }
        }

        private String extractString(Map<String, Object> map, String key) {
            Object value = map.get(key);
            return value != null ? value.toString() : null;
        }

        private String extractString(Map<String, Object> map, String key, String defaultValue) {
            String value = extractString(map, key);
            return value != null ? value : defaultValue;
        }

        private Integer extractInteger(Map<String, Object> map, String key, int defaultValue) {
            Integer value = extractInteger(map, key);
            return value != null ? value : defaultValue;
        }

        private boolean extractBoolean(Map<String, Object> map, String key, boolean defaultValue) {
            Object value = map.get(key);
            if (value == null) return defaultValue;
            if (value instanceof Boolean) return (Boolean) value;
            return Boolean.parseBoolean(value.toString());
        }

        private List<String> extractDiagnosisCodes(List<Map<String, Object>> conditions) {
            return conditions.stream()
                .map(c -> (Map<String, Object>) c.get("code"))
                .filter(Objects::nonNull)
                .map(code -> (List<Map<String, Object>>) code.get("coding"))
                .filter(Objects::nonNull)
                .flatMap(List::stream)
                .map(coding -> (String) coding.get("code"))
                .filter(Objects::nonNull)
                .distinct()
                .collect(Collectors.toList());
        }

        private List<String> extractMedicationNames(List<Map<String, Object>> medications) {
            return medications.stream()
                .map(m -> (Map<String, Object>) m.get("medicationCodeableConcept"))
                .filter(Objects::nonNull)
                .map(med -> (String) med.get("text"))
                .filter(Objects::nonNull)
                .collect(Collectors.toList());
        }

        private Map<String, Object> extractVitalSigns(List<Map<String, Object>> observations) {
            Map<String, Object> vitals = new HashMap<>();
            // Extract latest vital signs from observations
            // Implementation simplified for brevity
            return vitals;
        }

        private Map<String, Object> extractLabResults(List<Map<String, Object>> observations) {
            Map<String, Object> labs = new HashMap<>();
            // Extract latest lab results from observations
            // Implementation simplified for brevity
            return labs;
        }

        /**
         * Extract diagnosis codes from typed Condition objects
         */
        private List<String> extractDiagnosisCodesFromConditions(List<Condition> conditions) {
            if (conditions == null) return new ArrayList<>();
            return conditions.stream()
                .map(c -> c.getCode() != null ? c.getCode() : c.getDisplay())
                .filter(code -> code != null)
                .collect(java.util.stream.Collectors.toList());
        }

        /**
         * Extract medication names from typed Medication objects
         */
        private List<String> extractMedicationNamesFromMedications(List<Medication> medications) {
            if (medications == null) return new ArrayList<>();
            return medications.stream()
                .map(m -> m.getName())
                .filter(name -> name != null)
                .collect(java.util.stream.Collectors.toList());
        }

        private String determineUrgency(CanonicalEvent event, Map<String, Object> enrichmentData) {
            // Logic to determine clinical urgency
            if (event.getEventType() != null && "EMERGENCY".equals(event.getEventType().name())) {
                return "CRITICAL";
            }
            return "ROUTINE";
        }

        private List<String> identifyActiveProblems(Map<String, Object> enrichmentData) {
            List<String> problems = new ArrayList<>();
            // Logic to identify active clinical problems
            return problems;
        }

        private Map<String, Double> calculateRiskScores(Map<String, Object> enrichmentData) {
            Map<String, Double> scores = new HashMap<>();
            // Calculate various risk scores
            scores.put("readmissionRisk", 0.0);
            scores.put("deteriorationRisk", 0.0);
            return scores;
        }

        /**
         * Extract age from FHIR patient data (single source of truth)
         */
        private Integer extractAgeFromEnrichmentData(Map<String, Object> enrichmentData) {
            if (enrichmentData.containsKey("patient")) {
                Object patientObj = enrichmentData.get("patient");
                if (patientObj instanceof FHIRPatientData) {
                    FHIRPatientData patientData = (FHIRPatientData) patientObj;
                    return patientData.getAge();
                }
            }
            return null; // Return null instead of placeholder if not available
        }

        /**
         * Extract gender from FHIR patient data (single source of truth)
         */
        private String extractGenderFromEnrichmentData(Map<String, Object> enrichmentData) {
            if (enrichmentData.containsKey("patient")) {
                Object patientObj = enrichmentData.get("patient");
                if (patientObj instanceof FHIRPatientData) {
                    FHIRPatientData patientData = (FHIRPatientData) patientObj;
                    return patientData.getGender();
                }
            }
            return "unknown";
        }

        private String getLocation(String patientId) {
            return "unknown";
        }

        private EnrichedEvent createFallbackEnrichedEvent(CanonicalEvent event) {
            EnrichedEvent enrichedEvent = new EnrichedEvent();
            enrichedEvent.setId(event.getId());  // Use setId() not setEventId()
            enrichedEvent.setPatientId(event.getPatientId());
            enrichedEvent.setEventType(event.getEventType());  // Pass EventType enum directly
            enrichedEvent.setEventTime(event.getEventTime());
            enrichedEvent.setPayload(event.getPayload());  // Use setPayload() not setEventData()
            enrichedEvent.setEnrichmentData(new HashMap<>());
            enrichedEvent.setProcessingTime(System.currentTimeMillis());
            return enrichedEvent;
        }

        public void close() throws Exception {
            if (fhirClient != null) {
                fhirClient.close();
            }
            if (neo4jClient != null) {
                neo4jClient.close();
            }
        }
    }

    // Helper methods
    // (Removed old helper methods that referenced deleted classes)

    // Source and sink creation methods (simplified for brevity)

    private static DataStream<CanonicalEvent> createCanonicalEventSource(StreamExecutionEnvironment env) {
        // Kafka source configuration for canonical events from Module1 output
        // Use Docker network address when running in Flink cluster
        String kafkaBootstrapServers = System.getenv().getOrDefault("KAFKA_BOOTSTRAP_SERVERS", "kafka:29092");

        // Custom deserializer for CanonicalEvent — crash-safe (null-on-failure)
        // NOTE: ObjectMapper must be transient because JavaTimeModule contains
        // DateTimeFormatter instances which are NOT java.io.Serializable.
        // Flink serializes DeserializationSchema for cluster distribution.
        org.apache.flink.api.common.serialization.DeserializationSchema<CanonicalEvent> deserializer =
            new org.apache.flink.api.common.serialization.DeserializationSchema<CanonicalEvent>() {
                private transient ObjectMapper mapper;

                private ObjectMapper getMapper() {
                    if (mapper == null) {
                        mapper = new ObjectMapper();
                        mapper.registerModule(new JavaTimeModule());
                    }
                    return mapper;
                }

                @Override
                public CanonicalEvent deserialize(byte[] message) throws java.io.IOException {
                    try {
                        return getMapper().readValue(message, CanonicalEvent.class);
                    } catch (Exception e) {
                        LOG.error("Failed to deserialize CanonicalEvent ({} bytes), skipping: {}",
                            message != null ? message.length : 0, e.getMessage());
                        return null;  // Null = skip this record, don't crash-loop
                    }
                }

                @Override
                public boolean isEndOfStream(CanonicalEvent nextElement) {
                    return false;
                }

                @Override
                public org.apache.flink.api.common.typeinfo.TypeInformation<CanonicalEvent> getProducedType() {
                    return org.apache.flink.api.common.typeinfo.TypeInformation.of(CanonicalEvent.class);
                }
            };

        // Read from Module1's output topic: enriched-patient-events-v1
        KafkaSource<CanonicalEvent> kafkaSource = KafkaSource.<CanonicalEvent>builder()
            .setBootstrapServers(kafkaBootstrapServers)
            .setTopics("enriched-patient-events-v1")
            .setGroupId("module2-enhanced-consumer")
            .setStartingOffsets(org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer.latest())
            .setValueOnlyDeserializer(deserializer)
            .build();

        return env.fromSource(
            kafkaSource,
            org.apache.flink.api.common.eventtime.WatermarkStrategy
                .<CanonicalEvent>forBoundedOutOfOrderness(java.time.Duration.ofMinutes(5))
                .withTimestampAssigner((event, recordTimestamp) ->
                    event.getEventTime() > 0 ? event.getEventTime() : recordTimestamp)
                .withIdleness(java.time.Duration.ofMinutes(5)),
            "Kafka-CanonicalEvents-Source"
        );
    }

    private static KafkaSink<EnrichedEvent> createEnrichedEventsSink() {
        // Kafka sink configuration for enriched events with clinical patterns
        // Module2 output goes to clinical-patterns.v1 (NOT back to enriched-patient-events-v1)
        // Use Docker network address when running in Flink cluster
        String kafkaBootstrapServers = System.getenv().getOrDefault("KAFKA_BOOTSTRAP_SERVERS", "kafka:29092");

        return org.apache.flink.connector.kafka.sink.KafkaSink.<EnrichedEvent>builder()
            .setBootstrapServers(kafkaBootstrapServers)
            .setTransactionalIdPrefix("module2-enhanced-enriched-events")
            .setRecordSerializer(org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema.builder()
                .setTopic("clinical-patterns.v1")
                .setValueSerializationSchema(new org.apache.flink.api.common.serialization.SerializationSchema<EnrichedEvent>() {
                    private static final ObjectMapper MAPPER = new ObjectMapper();
                    static {
                        MAPPER.registerModule(new JavaTimeModule());
                    }

                    @Override
                    public byte[] serialize(EnrichedEvent event) {
                        try {
                            return MAPPER.writeValueAsBytes(event);
                        } catch (Exception e) {
                            LOG.error("Failed to serialize enriched event for patient {}: {}",
                                event != null ? event.getPatientId() : "null", e.getMessage());
                            throw new RuntimeException("EnrichedEvent serialization failed", e);
                        }
                    }
                })
                .build()
            )
            .setDeliveryGuarantee(org.apache.flink.connector.base.DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    /**
     * Create Kafka sink for Protocol Trigger events
     * Phase 4 Enhancement: Emit protocol triggers for audit trail
     */
    private static KafkaSink<ProtocolEvent> createProtocolEventsSink() {
        String kafkaBootstrapServers = System.getenv().getOrDefault("KAFKA_BOOTSTRAP_SERVERS", "kafka:29092");

        return org.apache.flink.connector.kafka.sink.KafkaSink.<ProtocolEvent>builder()
            .setBootstrapServers(kafkaBootstrapServers)
            .setTransactionalIdPrefix("module2-enhanced-protocol-events")
            .setRecordSerializer(org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.PROTOCOL_TRIGGERS.getTopicName())
                .setValueSerializationSchema(new ProtocolEventSerializer())
                .build()
            )
            .setDeliveryGuarantee(org.apache.flink.connector.base.DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    /**
     * Generate Protocol Trigger Events from matched protocols
     * Phase 4 Enhancement: Creates audit trail for clinical protocol triggers
     */
    private static List<ProtocolEvent> generateProtocolEvents(
            String patientId,
            String encounterId,
            String sourceEventId,
            List<com.cardiofit.flink.protocols.ProtocolMatcher.Protocol> applicableProtocols,
            EnhancedRiskIndicators.RiskAssessment riskAssessment,
            NEWS2Calculator.NEWS2Score news2Score,
            ClinicalScoreCalculators.qSOFAScore qsofaScore) {

        List<ProtocolEvent> protocolEvents = new ArrayList<>();

        if (applicableProtocols == null || applicableProtocols.isEmpty()) {
            return protocolEvents;
        }

        for (Protocol protocol : applicableProtocols) {
            // Build clinical indicators map
            Map<String, Object> clinicalIndicators = new HashMap<>();

            if (riskAssessment != null) {
                clinicalIndicators.put("heart_rate", riskAssessment.getCurrentHeartRate());
                clinicalIndicators.put("blood_pressure", riskAssessment.getCurrentBloodPressure());
                clinicalIndicators.put("tachycardia_severity",
                    riskAssessment.getTachycardiaSeverity() != null ?
                    riskAssessment.getTachycardiaSeverity().toString() : "NONE");
                clinicalIndicators.put("bradycardia_severity",
                    riskAssessment.getBradycardiaSeverity() != null ?
                    riskAssessment.getBradycardiaSeverity().toString() : "NONE");
                clinicalIndicators.put("hypertension_stage",
                    riskAssessment.getHypertensionStage() != null ?
                    riskAssessment.getHypertensionStage().toString() : "NONE");
            }

            if (news2Score != null) {
                clinicalIndicators.put("news2_score", news2Score.getTotalScore());
                clinicalIndicators.put("news2_risk_level", news2Score.getRiskLevel());
            }

            if (qsofaScore != null) {
                clinicalIndicators.put("qsofa_score", qsofaScore.getTotalScore());
                clinicalIndicators.put("qsofa_risk_level", qsofaScore.getRiskLevel());
            }

            // Convert action items to semicolon-separated string
            String actions = "No specific actions defined";
            if (protocol.getActionItems() != null && !protocol.getActionItems().isEmpty()) {
                List<String> actionStrings = new ArrayList<>();
                for (Object actionItem : protocol.getActionItems()) {
                    if (actionItem instanceof String) {
                        actionStrings.add((String) actionItem);
                    } else if (actionItem instanceof ProtocolMatcher.ActionItem) {
                        ProtocolMatcher.ActionItem item = (ProtocolMatcher.ActionItem) actionItem;
                        actionStrings.add(item.getAction());
                    }
                }
                actions = actionStrings.isEmpty() ? "No specific actions defined" : String.join("; ", actionStrings);
            }

            ProtocolEvent protocolEvent = ProtocolEvent.builder()
                .eventId(java.util.UUID.randomUUID().toString())
                .patientId(patientId)
                .encounterId(encounterId)
                .sourceEventId(sourceEventId)
                .protocolName(protocol.getName())
                .protocolCategory(protocol.getCategory())
                .triggerReason(protocol.getTriggerReason())
                .severity(protocol.getPriority())
                .recommendedActions(actions)
                .clinicalIndicators(clinicalIndicators)
                .triggeredAt(System.currentTimeMillis())
                .build();

            protocolEvents.add(protocolEvent);
            LOG.info("Protocol trigger event created: {} for patient {}", protocol.getName(), patientId);
        }

        return protocolEvents;
    }

    /**
     * CRITICAL FIX: Extract and populate top-level EnrichedEvent fields from clinicalIntelligence
     *
     * Problem: We were calculating alerts, risk indicators, and clinical scores deep in
     * enrichmentData.clinicalContext.clinicalIntelligence, but never surfacing them to the
     * top-level fields that downstream systems expect.
     *
     * This method extracts:
     * 1. immediate_alerts from clinicalIntelligence.alerts
     * 2. risk_indicators from clinicalIntelligence.riskAssessment
     * 3. clinical_scores from clinicalIntelligence scores (NEWS2, qSOFA, etc)
     */
    private static void extractAndPopulateTopLevelFields(
            EnrichedEvent enrichedEvent,
            Map<String, Object> enrichmentData) {

        try {
            // Navigate to clinicalIntelligence
            Object clinicalContextObj = enrichmentData.get("clinicalContext");
            if (clinicalContextObj == null || !(clinicalContextObj instanceof Map)) {
                LOG.warn("clinicalContext not found in enrichmentData");
                return;
            }

            Map<String, Object> clinicalContext = (Map<String, Object>) clinicalContextObj;
            Object intelligenceObj = clinicalContext.get("clinicalIntelligence");

            if (intelligenceObj == null || !(intelligenceObj instanceof ClinicalIntelligence)) {
                LOG.warn("clinicalIntelligence not found in clinicalContext");
                return;
            }

            ClinicalIntelligence intelligence = (ClinicalIntelligence) intelligenceObj;

            // 1. Extract immediate_alerts from intelligence.alerts
            if (intelligence.getAlerts() != null) {
                LOG.info("Intelligence alerts: {} total", intelligence.getAlerts().size());
                if (!intelligence.getAlerts().isEmpty()) {
                    List<SimpleAlert> simpleAlerts = new ArrayList<>();
                    for (SmartAlertGenerator.ClinicalAlert alert : intelligence.getAlerts()) {
                        LOG.debug("Processing alert: {} - {}", alert.getAlertId(), alert.getMessage());
                        SimpleAlert simpleAlert = SimpleAlert.builder()
                            .patientId(intelligence.getPatientId())
                            .alertType(mapToAlertType(alert.getCategory()))
                            .severity(mapToAlertSeverity(alert.getPriority()))
                            .message(alert.getMessage())
                            .sourceModule("MODULE_2_CLINICAL_INTELLIGENCE")
                            .build();
                        simpleAlerts.add(simpleAlert);
                    }
                    enrichedEvent.setImmediateAlerts(simpleAlerts);
                    LOG.info("Populated {} immediate alerts", simpleAlerts.size());
                } else {
                    LOG.warn("Intelligence alerts list is empty - no alerts generated or all suppressed");
                }
            } else {
                LOG.warn("Intelligence alerts is null");
            }

            // 1b. Determine primary clinical finding - headline alert for dashboard
            String primaryFinding = determinePrimaryClinicalFinding(intelligence);
            if (primaryFinding != null) {
                enrichedEvent.setPrimaryClinicalFinding(primaryFinding);
                LOG.info("Set primary clinical finding: {}", primaryFinding);
            }

            // 1c. Check medication effectiveness for hypertensive crisis on antihypertensives
            checkMedicationEffectiveness(intelligence, enrichedEvent);

            // 2. Extract risk_indicators from riskAssessment
            if (intelligence.getRiskAssessment() != null) {
                // Pass vitals map for respiratory indicators (hypoxia, tachypnea)
                Map<String, Object> vitals = enrichedEvent.getPatientContext() != null ?
                    enrichedEvent.getPatientContext().getCurrentVitals() : null;

                RiskIndicators indicators = buildRiskIndicatorsFromAssessment(
                    intelligence.getRiskAssessment(),
                    enrichedEvent.getPatientContext(),
                    vitals
                );
                enrichedEvent.setRiskIndicators(indicators);
                LOG.info("Populated risk indicators: tachycardia={}, hypertension={}, diabetes={}, hypoxia={}, tachypnea={}",
                    indicators.isTachycardia(), indicators.isHypertension(), indicators.isHasDiabetes(),
                    indicators.isHypoxia(), indicators.isTachypnea());
            }

            // 3. Extract clinical_scores
            Map<String, Double> clinicalScores = new HashMap<>();

            if (intelligence.getNews2Score() != null) {
                clinicalScores.put("news2_score", (double) intelligence.getNews2Score().getTotalScore());
            }

            if (intelligence.getQsofaScore() != null) {
                clinicalScores.put("qsofa_score", (double) intelligence.getQsofaScore().getTotalScore());
            }

            if (intelligence.getMetabolicSyndromeScore() != null) {
                clinicalScores.put("metabolic_syndrome_risk", intelligence.getMetabolicSyndromeScore().getRiskScore());
            }

            if (intelligence.getCombinedAcuityScore() != null) {
                // Round to 1 decimal place to avoid floating point artifacts
                double combinedScore = Math.round(intelligence.getCombinedAcuityScore().getCombinedAcuityScore() * 10.0) / 10.0;
                clinicalScores.put("combined_acuity_score", combinedScore);
            }

            if (intelligence.getConfidenceScore() != null) {
                clinicalScores.put("confidence_score", intelligence.getConfidenceScore().getOverallConfidence());
            }

            enrichedEvent.setClinicalScores(clinicalScores);
            LOG.info("Populated {} clinical scores", clinicalScores.size());

            // 4. Extract applicable_protocols from intelligence.applicableProtocols
            // Convert Protocol objects to protocol names for top-level field
            if (intelligence.getApplicableProtocols() != null && !intelligence.getApplicableProtocols().isEmpty()) {
                List<String> protocolNames = new ArrayList<>();
                for (Protocol protocol : intelligence.getApplicableProtocols()) {
                    protocolNames.add(protocol.getName());
                }
                enrichedEvent.setApplicableProtocols(protocolNames);
                LOG.info("Populated {} applicable protocols", protocolNames.size());
            }

        } catch (Exception e) {
            LOG.error("Error extracting top-level fields from clinicalIntelligence: {}", e.getMessage(), e);
        }
    }

    /**
     * Determine primary clinical finding - headline alert for dashboard display
     * Priority order:
     * 1. CRITICAL severity alerts (highest priority)
     * 2. Hypertensive crisis
     * 3. Severe sepsis risk
     * 4. Acute hypoxia
     * 5. Other HIGH severity alerts
     * 6. First protocol triggered
     */
    private static String determinePrimaryClinicalFinding(ClinicalIntelligence intelligence) {
        if (intelligence == null) return null;

        // Priority 1: CRITICAL alerts from alert generator
        if (intelligence.getAlerts() != null && !intelligence.getAlerts().isEmpty()) {
            for (SmartAlertGenerator.ClinicalAlert alert : intelligence.getAlerts()) {
                if (alert.getPriority() == SmartAlertGenerator.AlertPriority.CRITICAL) {
                    return alert.getMessage().toUpperCase(); // Uppercase for dashboard prominence
                }
            }
        }

        // Priority 2: Check risk assessment for specific critical conditions
        if (intelligence.getRiskAssessment() != null) {
            EnhancedRiskIndicators.RiskAssessment risk = intelligence.getRiskAssessment();

            // Hypertensive crisis
            if (risk.isHypertensionCrisis()) {
                return "HYPERTENSIVE CRISIS - BP: " + risk.getCurrentBloodPressure();
            }

            // Sepsis risk (QSOFA >= 2)
            if (intelligence.getQsofaScore() != null && intelligence.getQsofaScore().getTotalScore() >= 2) {
                return "SEVERE SEPSIS RISK - qSOFA: " + intelligence.getQsofaScore().getTotalScore();
            }

            // Severe hypoxia (check if hypoxia finding is present)
            if (risk.isHypoxia()) {
                return "ACUTE HYPOXIA DETECTED";
            }
        }

        // Priority 3: HIGH severity alerts
        if (intelligence.getAlerts() != null && !intelligence.getAlerts().isEmpty()) {
            for (SmartAlertGenerator.ClinicalAlert alert : intelligence.getAlerts()) {
                if (alert.getPriority() == SmartAlertGenerator.AlertPriority.HIGH) {
                    return alert.getMessage().toUpperCase();
                }
            }
        }

        // Priority 4: First applicable protocol
        if (intelligence.getApplicableProtocols() != null && !intelligence.getApplicableProtocols().isEmpty()) {
            Protocol firstProtocol = intelligence.getApplicableProtocols().get(0);
            return firstProtocol.getTriggerReason();
        }

        // Priority 5: First alert of any severity
        if (intelligence.getAlerts() != null && !intelligence.getAlerts().isEmpty()) {
            return intelligence.getAlerts().get(0).getMessage().toUpperCase();
        }

        return null; // No critical findings
    }

    /**
     * Build RiskIndicators from RiskAssessment, PatientContext, and current vitals
     */
    private static RiskIndicators buildRiskIndicatorsFromAssessment(
            EnhancedRiskIndicators.RiskAssessment assessment,
            PatientContext patientContext,
            Map<String, Object> vitals) {

        RiskIndicators indicators = new RiskIndicators();

        // Cardiac indicators
        indicators.setTachycardia(
            assessment.getTachycardiaSeverity() != null &&
            assessment.getTachycardiaSeverity() != EnhancedRiskIndicators.Severity.NONE
        );

        indicators.setBradycardia(
            assessment.getBradycardiaSeverity() != null &&
            assessment.getBradycardiaSeverity() != EnhancedRiskIndicators.Severity.NONE
        );

        // Blood pressure indicators
        indicators.setHypertension(
            assessment.getHypertensionStage() != null &&
            assessment.getHypertensionStage() != EnhancedRiskIndicators.HypertensionStage.NORMAL
        );

        indicators.setHypotension(
            assessment.getCurrentBloodPressure() != null &&
            assessment.getCurrentBloodPressure().contains("/") &&
            extractSystolic(assessment.getCurrentBloodPressure()) < 90
        );

        // FIXED: Respiratory indicators from vitals (hypoxia and tachypnea)
        // Clinical thresholds: Hypoxia = SpO2 < 92%, Tachypnea = RR > 20
        if (vitals != null) {
            // Hypoxia: Check from RiskAssessment first, then fallback to SpO2 threshold
            boolean hypoxiaDetected = assessment.isHypoxia(); // From findings
            if (!hypoxiaDetected) {
                // Check SpO2 threshold if not already detected
                // Note: vitals uses lowercase keys (oxygensaturation, not oxygenSaturation)
                Integer spo2 = extractInteger(vitals, "oxygensaturation");
                if (spo2 == null) {
                    spo2 = extractInteger(vitals, "oxygenSaturation"); // fallback to camelCase
                }
                if (spo2 != null && spo2 < 92) {
                    hypoxiaDetected = true;
                }
            }
            indicators.setHypoxia(hypoxiaDetected);

            // Tachypnea: Respiratory rate > 20
            // Note: vitals uses lowercase keys (respiratoryrate, not respiratoryRate)
            Integer rr = extractInteger(vitals, "respiratoryrate");
            if (rr == null) {
                rr = extractInteger(vitals, "respiratoryRate"); // fallback to camelCase
            }
            indicators.setTachypnea(rr != null && rr > 20);

            // Bradypnea: Respiratory rate < 12
            indicators.setBradypnea(rr != null && rr < 12);
        }

        // FIXED: Calculate trends based on current vital thresholds (ELEVATED/LOW/NORMAL)
        // Previous: Used generic TrendDirection which only worked with historical data
        // Now: Assess current vital signs against clinical thresholds

        // Heart Rate Trend (Normal: 60-100 bpm)
        if (assessment.getCurrentHeartRate() != null) {
            int hr = assessment.getCurrentHeartRate();
            if (hr > 100) {
                indicators.setHeartRateTrend(TrendDirection.ELEVATED);
            } else if (hr < 60) {
                indicators.setHeartRateTrend(TrendDirection.LOW);
            } else {
                indicators.setHeartRateTrend(TrendDirection.NORMAL);
            }
        }

        // Blood Pressure Trend (Normal: <120/80, Elevated: ≥130/80)
        if (assessment.getCurrentBloodPressure() != null) {
            int systolic = extractSystolic(assessment.getCurrentBloodPressure());
            if (systolic >= 130) {
                indicators.setBloodPressureTrend(TrendDirection.ELEVATED);
            } else if (systolic < 90) {
                indicators.setBloodPressureTrend(TrendDirection.LOW);
            } else {
                indicators.setBloodPressureTrend(TrendDirection.NORMAL);
            }
        }

        // Oxygen Saturation Trend (Normal: ≥95%)
        // Extract SpO2 from vitals map using same pattern as respiratory indicators
        // Clinical thresholds: <85% CRITICALLY_LOW, 85-91% LOW, 92-94% BORDERLINE, ≥95% NORMAL
        if (vitals != null) {
            Integer spo2 = extractInteger(vitals, "oxygensaturation");
            if (spo2 == null) {
                spo2 = extractInteger(vitals, "oxygenSaturation"); // fallback to camelCase
            }

            if (spo2 != null) {
                if (spo2 < 85) {
                    indicators.setOxygenSaturationTrend(TrendDirection.CRITICALLY_LOW);
                } else if (spo2 < 92) {
                    indicators.setOxygenSaturationTrend(TrendDirection.LOW);
                } else if (spo2 < 95) {
                    indicators.setOxygenSaturationTrend(TrendDirection.BORDERLINE);
                } else {
                    indicators.setOxygenSaturationTrend(TrendDirection.NORMAL);
                }
            } else {
                indicators.setOxygenSaturationTrend(TrendDirection.UNKNOWN);
            }
        } else {
            indicators.setOxygenSaturationTrend(TrendDirection.UNKNOWN);
        }

        // Temperature Trend (Normal: 36.1-37.2°C)
        // Extract temperature from vitals map and set both trend and hypothermia/fever flags
        // Clinical thresholds: <35°C HYPOTHERMIA, 35-36°C LOW, 36.1-37.2°C NORMAL, 37.5-38°C ELEVATED, >38°C FEVER
        if (vitals != null) {
            // Try multiple possible keys for temperature (Celsius)
            Double temp = extractDouble(vitals, "temperature");
            if (temp == null) temp = extractDouble(vitals, "bodyTemperature");
            if (temp == null) temp = extractDouble(vitals, "bodytemperature");
            if (temp == null) temp = extractDouble(vitals, "temp");

            if (temp != null) {
                if (temp < 35.0) {
                    indicators.setTemperatureTrend(TrendDirection.HYPOTHERMIA);
                    indicators.setHypothermia(true);
                } else if (temp < 36.0) {
                    indicators.setTemperatureTrend(TrendDirection.LOW);
                    indicators.setHypothermia(false);
                } else if (temp <= 37.2) {
                    indicators.setTemperatureTrend(TrendDirection.NORMAL);
                    indicators.setHypothermia(false);
                    indicators.setFever(false);
                } else if (temp <= 38.0) {
                    indicators.setTemperatureTrend(TrendDirection.ELEVATED);
                    indicators.setFever(false);
                } else {
                    indicators.setTemperatureTrend(TrendDirection.FEVER);
                    indicators.setFever(true);
                }
            } else {
                indicators.setTemperatureTrend(TrendDirection.UNKNOWN);
            }
        } else {
            indicators.setTemperatureTrend(TrendDirection.UNKNOWN);
        }

        // Chronic conditions from patient context (condition names as strings)
        if (patientContext != null && patientContext.getActiveConditions() != null) {
            List<String> conditions = patientContext.getActiveConditions();

            // FIXED: Include Prediabetes as diabetes risk indicator
            // PatientContext.getActiveConditions() returns condition names as strings
            // Check if any condition name contains "diabetes" (includes "Diabetes", "Prediabetes", etc.)
            boolean hasDiabetes = conditions.stream().anyMatch(conditionName ->
                conditionName != null && conditionName.toLowerCase().contains("diabetes")
            );

            indicators.setHasDiabetes(hasDiabetes);

            // Chronic kidney disease
            indicators.setHasChronicKidneyDisease(conditions.stream().anyMatch(conditionName ->
                conditionName != null &&
                (conditionName.toLowerCase().contains("chronic kidney") ||
                 conditionName.toLowerCase().contains("ckd"))
            ));

            // Heart failure
            indicators.setHasHeartFailure(conditions.stream().anyMatch(conditionName ->
                conditionName != null &&
                conditionName.toLowerCase().contains("heart failure")
            ));
        }

        // Confidence
        indicators.setConfidenceScore(0.85); // Default confidence from assessment

        return indicators;
    }

    /**
     * Extract systolic BP from "120/80" format
     */
    private static int extractSystolic(String bp) {
        try {
            String[] parts = bp.split("/");
            return Integer.parseInt(parts[0].trim());
        } catch (Exception e) {
            return 120; // Default
        }
    }

    /**
     * Extract integer value from vitals map
     */
    private static Integer extractInteger(Map<String, Object> vitals, String key) {
        if (vitals == null || !vitals.containsKey(key)) return null;

        Object value = vitals.get(key);
        if (value instanceof Integer) {
            return (Integer) value;
        } else if (value instanceof Number) {
            return ((Number) value).intValue();
        } else if (value instanceof String) {
            try {
                return Integer.parseInt((String) value);
            } catch (NumberFormatException e) {
                return null;
            }
        }
        return null;
    }

    /**
     * Extract double value from vitals map (for temperature, etc.)
     */
    private static Double extractDouble(Map<String, Object> vitals, String key) {
        if (vitals == null || !vitals.containsKey(key)) return null;

        Object value = vitals.get(key);
        if (value instanceof Double) {
            return (Double) value;
        } else if (value instanceof Number) {
            return ((Number) value).doubleValue();
        } else if (value instanceof String) {
            try {
                return Double.parseDouble((String) value);
            } catch (NumberFormatException e) {
                return null;
            }
        }
        return null;
    }

    /**
     * Check medication effectiveness for hypertensive crisis patients on antihypertensives
     * Clinical logic: If patient is on BP medication but presenting with hypertensive crisis,
     * flag potential compliance issue or inadequate treatment
     */
    private static void checkMedicationEffectiveness(
            ClinicalIntelligence intelligence,
            EnrichedEvent enrichedEvent) {

        // Only check if patient has hypertensive crisis
        if (intelligence.getRiskAssessment() == null ||
            !intelligence.getRiskAssessment().isHypertensionCrisis()) {
            return;
        }

        PatientContext context = enrichedEvent.getPatientContext();
        if (context == null || context.getActiveMedications() == null ||
            context.getActiveMedications().isEmpty()) {
            return;
        }

        // Check if patient is on antihypertensive medications
        // activeMedications is Map<String, Object> where values are medication name strings
        // Keys are like "med_0", "med_1", values are like "Telmisartan 40 mg Tablet"
        boolean onAntihypertensive = false;
        String medicationName = null;

        for (Object medValue : context.getActiveMedications().values()) {
            if (medValue == null) continue;

            String medName = medValue.toString();
            String medNameLower = medName.toLowerCase();

            // Check for common antihypertensive classes
            boolean isARB = medNameLower.contains("sartan"); // ARBs end in -sartan (e.g., telmisartan)
            boolean isACEI = medNameLower.contains("pril");  // ACE inhibitors end in -pril
            boolean isBetaBlocker = medNameLower.contains("olol"); // Beta blockers end in -olol
            boolean isCCB = medNameLower.contains("dipine"); // CCBs end in -dipine
            boolean isDiuretic = medNameLower.contains("thiazide") ||
                                medNameLower.contains("furosemide") ||
                                medNameLower.contains("chlorthalidone");

            if (isARB || isACEI || isBetaBlocker || isCCB || isDiuretic) {
                onAntihypertensive = true;
                medicationName = medName;
                break;
            }
        }

        // If patient is on antihypertensive but has hypertensive crisis, add alert
        if (onAntihypertensive) {
            String alertMessage = String.format(
                "Patient on %s presenting with hypertensive crisis (BP %s). " +
                "Consider: medication compliance, dosing adequacy, acute stressor.",
                medicationName,
                intelligence.getRiskAssessment().getCurrentBloodPressure()
            );

            SimpleAlert effectivenessAlert = new SimpleAlert();
            effectivenessAlert.setAlertType(AlertType.DRUG_INTERACTION); // Use existing enum value
            effectivenessAlert.setSeverity(AlertSeverity.HIGH);
            effectivenessAlert.setMessage(alertMessage);
            effectivenessAlert.setTimestamp(System.currentTimeMillis());
            effectivenessAlert.setSourceModule("MODULE_2_MED_EFFECTIVENESS");

            // Add to immediate alerts
            if (enrichedEvent.getImmediateAlerts() == null) {
                enrichedEvent.setImmediateAlerts(new ArrayList<>());
            }
            enrichedEvent.getImmediateAlerts().add(effectivenessAlert);

            LOG.warn("Medication effectiveness concern: {} with hypertensive crisis on {}",
                enrichedEvent.getPatientId(), medicationName);
        }
    }

    /**
     * Map SmartAlertGenerator.AlertCategory to SimpleAlert.AlertType
     */
    private static AlertType mapToAlertType(SmartAlertGenerator.AlertCategory category) {
        if (category == null) return AlertType.VITAL_THRESHOLD_BREACH;

        switch (category) {
            case CARDIAC:
                return AlertType.CARDIAC_EVENT;
            case BLOOD_PRESSURE:
                return AlertType.VITAL_THRESHOLD_BREACH;
            case RESPIRATORY:
                return AlertType.RESPIRATORY_DISTRESS;
            case TEMPERATURE:
                return AlertType.VITAL_THRESHOLD_BREACH;
            case ACUITY:
                return AlertType.CLINICAL_SCORE_HIGH;
            case TRENDING:
                return AlertType.DETERIORATION_PATTERN;
            case DATA_QUALITY:
                return AlertType.VITAL_THRESHOLD_BREACH; // Closest match
            default:
                return AlertType.VITAL_THRESHOLD_BREACH;
        }
    }

    /**
     * Map SmartAlertGenerator.AlertPriority to SimpleAlert.AlertSeverity
     */
    private static AlertSeverity mapToAlertSeverity(SmartAlertGenerator.AlertPriority priority) {
        if (priority == null) return AlertSeverity.WARNING;

        switch (priority) {
            case CRITICAL:
                return AlertSeverity.CRITICAL;
            case HIGH:
                return AlertSeverity.HIGH;
            case MEDIUM:
                return AlertSeverity.WARNING;
            case LOW:
                return AlertSeverity.INFO;
            default:
                return AlertSeverity.WARNING;
        }
    }

    /**
     * Map EnhancedRiskIndicators.TrendDirection to RiskIndicators.TrendDirection
     */
    private static TrendDirection mapToTrendDirection(EnhancedRiskIndicators.TrendDirection enhancedTrend) {
        if (enhancedTrend == null) return TrendDirection.STABLE;

        switch (enhancedTrend) {
            case IMPROVING:
                return TrendDirection.DECREASING; // Improving = decreasing risk
            case STABLE:
                return TrendDirection.STABLE;
            case DETERIORATING:
                return TrendDirection.INCREASING; // Deteriorating = increasing risk
            default:
                return TrendDirection.STABLE;
        }
    }

    // ========================================================================================
    // PHASE 6: UNIFIED CLINICAL REASONING PIPELINE
    // ========================================================================================

    /**
     * Create Unified Clinical Reasoning Pipeline (Phases 1-5)
     *
     * This is an alternative to createEnhancedPipeline() that uses the new unified
     * state management operators to eliminate race conditions.
     *
     * Architecture:
     * 1. Read CanonicalEvent from enriched-patient-events-v1 (Module 1 output)
     * 2. Convert CanonicalEvent → GenericEvent (unified wrapper)
     * 3. KeyBy patientId for state partitioning
     * 4. PatientContextAggregator - unified state management (labs, meds, vitals)
     * 5. ClinicalIntelligenceEvaluator - advanced pattern detection (sepsis, ACS, MODS)
     * 6. ClinicalEventFinalizer - pass-through with logging
     * 7. Sink to clinical-patterns.v1 (same output topic as original pipeline)
     *
     * Benefits vs Original Pipeline:
     * - Eliminates race conditions from separate lab/med operators
     * - Unified patient state in RocksDB (single source of truth)
     * - Exactly-once semantics for all clinical logic
     * - Simplified debugging and monitoring
     *
     * To use this pipeline instead of the original, modify main():
     * Replace: createEnhancedPipeline(env);
     * With:    createUnifiedPipeline(env);
     */
    public static void createUnifiedPipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating UNIFIED Clinical Reasoning Pipeline (Phases 1-5)");

        // Step 1: Read CanonicalEvent from Module 1 output
        DataStream<CanonicalEvent> canonicalEvents = createCanonicalEventSource(env);

        // Filter out nulls from crash-safe deserializer (malformed messages)
        DataStream<CanonicalEvent> validEvents = canonicalEvents
                .filter(event -> event != null)
                .uid("module2-null-event-filter");

        // Step 2: Convert CanonicalEvent to GenericEvent for unified processing
        DataStream<GenericEvent> genericEvents = validEvents
                .flatMap(new CanonicalEventToGenericEventConverter())
                .uid("canonical-to-generic-converter");

        // Step 3 & 4: Key by patientId and apply PatientContextAggregator
        // Use SingleOutputStreamOperator to access DLQ side-output
        org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator<EnrichedPatientContext> aggregatedContext = genericEvents
                .keyBy(GenericEvent::getPatientId)
                .process(new PatientContextAggregator())
                .uid("unified-patient-context-aggregator");

        // DLQ: Capture failed events from aggregator
        aggregatedContext.getSideOutput(PatientContextAggregator.DLQ_TAG)
                .sinkTo(createDlqSink("module2-dlq.v1"))
                .uid("module2-dlq-sink");

        // Step 4.5: FHIR & Neo4j Enrichment (Lazy Enrichment Pattern)
        // This step enriches patient context with FHIR data (demographics, conditions, medications)
        // and Neo4j graph data (cohorts, care team, pathways) AFTER aggregation.
        // Lazy enrichment: Only enriches on first event per patient (99% API call reduction)
        DataStream<EnrichedPatientContext> enrichedContext = AsyncDataStream
                .unorderedWait(
                        aggregatedContext,
                        new PatientContextEnricher(),
                        10000,  // 10 second timeout for FHIR/Neo4j calls
                        TimeUnit.MILLISECONDS,
                        500     // Max 500 concurrent async requests
                )
                .uid("patient-context-enricher");

        // Step 5: ClinicalIntelligenceEvaluator - advanced pattern detection
        DataStream<EnrichedPatientContext> intelligentContext = enrichedContext
                .process(new ClinicalIntelligenceEvaluator())
                .uid("clinical-intelligence-evaluator");

        // Step 6: ClinicalEventFinalizer - pass-through with logging
        DataStream<EnrichedPatientContext> finalizedContext = intelligentContext
                .process(new ClinicalEventFinalizer())
                .uid("clinical-event-finalizer");

        // Step 7: Sink to Kafka clinical-patterns.v1
        finalizedContext
                .sinkTo(createEnrichedPatientContextSink())
                .uid("unified-pipeline-sink");

        LOG.info("Unified Clinical Reasoning Pipeline created successfully");
        LOG.info("Pipeline operators: CanonicalEvent → GenericEvent → Aggregator → Intelligence → Finalizer → Sink");
    }

    /**
     * Converter: CanonicalEvent → GenericEvent
     *
     * Extracts vitals, labs, and medications from CanonicalEvent payload and creates
     * appropriate GenericEvent wrappers with eventType discrimination.
     *
     * Since CanonicalEvent may contain ALL data types in a single payload, we emit
     * multiple GenericEvents per CanonicalEvent:
     * - 1 VITAL_SIGN event if vitals present
     * - N LAB_RESULT events (one per lab)
     * - M MEDICATION_UPDATE events (one per medication)
     */
    private static class CanonicalEventToGenericEventConverter implements FlatMapFunction<CanonicalEvent, GenericEvent> {
        private static final Logger LOG = LoggerFactory.getLogger(CanonicalEventToGenericEventConverter.class);

        @Override
        public void flatMap(CanonicalEvent canonical, Collector<GenericEvent> out) throws Exception {
            String patientId = canonical.getPatientId();
            long eventTime = canonical.getEventTime();
            EventType eventType = canonical.getEventType();
            String sourceSystem = canonical.getSourceSystem();

            if (patientId == null) {
                LOG.warn("Received CanonicalEvent with null patientId, skipping");
                return;
            }

            // Extract payload map
            Map<String, Object> payload = canonical.getPayload();
            if (payload == null || payload.isEmpty()) {
                LOG.debug("CanonicalEvent {} has empty payload", canonical.getId());
                return;
            }

            // DYNAMIC PAYLOAD HANDLING: Support both nested and flat structures
            // Module 1 outputs FLAT payload with eventType indicating the data type
            // Future modules might output NESTED payload with vitals/labs/medications keys

            // Check if payload is nested (has vitals/labs/medications keys) or flat (direct fields)
            boolean hasNestedVitals = payload.containsKey("vitals");
            boolean hasNestedLabs = payload.containsKey("labs");
            boolean hasNestedMeds = payload.containsKey("medications");

            // STRATEGY 1: Nested structure - multiple event types in one payload
            if (hasNestedVitals || hasNestedLabs || hasNestedMeds) {
                LOG.debug("Processing NESTED payload structure for CanonicalEvent {}", canonical.getId());

                // Emit VITAL_SIGN event if vitals present
                if (hasNestedVitals) {
                    Object vitalsObj = payload.get("vitals");
                    if (vitalsObj instanceof Map) {
                        Map<String, Object> vitals = (Map<String, Object>) vitalsObj;
                        if (!vitals.isEmpty()) {
                            VitalsPayload vitalsPayload = convertToVitalsPayload(vitals);
                            GenericEvent vitalEvent = new GenericEvent();
                            vitalEvent.setEventType("VITAL_SIGN");
                            vitalEvent.setPatientId(patientId);
                            vitalEvent.setEventTime(eventTime);
                            vitalEvent.setPayload(vitalsPayload);
                            vitalEvent.setEncounterId(canonical.getEncounterId());
                            vitalEvent.setSource(sourceSystem);
                            out.collect(vitalEvent);
                            LOG.debug("Emitted VITAL_SIGN GenericEvent from nested structure");
                        }
                    }
                }

                // Emit LAB_RESULT events if labs present
                if (hasNestedLabs) {
                    Object labsObj = payload.get("labs");
                    if (labsObj instanceof Map) {
                        Map<String, Object> labs = (Map<String, Object>) labsObj;
                        for (Map.Entry<String, Object> entry : labs.entrySet()) {
                            LabPayload labPayload = convertToLabPayload(entry.getKey(), entry.getValue());
                            if (labPayload != null) {
                                GenericEvent labEvent = new GenericEvent();
                                labEvent.setEventType("LAB_RESULT");
                                labEvent.setPatientId(patientId);
                                labEvent.setEventTime(eventTime);
                                labEvent.setPayload(labPayload);
                                labEvent.setEncounterId(canonical.getEncounterId());
                                labEvent.setSource(sourceSystem);
                                out.collect(labEvent);
                                LOG.debug("Emitted LAB_RESULT GenericEvent from nested structure");
                            }
                        }
                    }
                }

                // Emit medication events if medications present (preserve original event type or default to MEDICATION_ORDERED)
                if (hasNestedMeds) {
                    Object medsObj = payload.get("medications");
                    if (medsObj instanceof List) {
                        List<?> medications = (List<?>) medsObj;
                        for (Object medObj : medications) {
                            MedicationPayload medPayload = convertToMedicationPayload(medObj);
                            if (medPayload != null) {
                                GenericEvent medEvent = new GenericEvent();
                                // Use eventType if medication-related, otherwise default to MEDICATION_ORDERED for nested structure
                                String medEventType = (eventType != null && eventType.isMedicationRelated())
                                    ? eventType.name()
                                    : "MEDICATION_ORDERED";
                                medEvent.setEventType(medEventType);
                                medEvent.setPatientId(patientId);
                                medEvent.setEventTime(eventTime);
                                medEvent.setPayload(medPayload);
                                medEvent.setEncounterId(canonical.getEncounterId());
                                medEvent.setSource(sourceSystem);
                                out.collect(medEvent);
                                LOG.debug("Emitted {} GenericEvent from nested structure", medEventType);
                            }
                        }
                    }
                }
            }
            // STRATEGY 2: Flat structure - use eventType to determine how to interpret payload
            else {
                LOG.debug("Processing FLAT payload structure for CanonicalEvent {} with eventType={}",
                    canonical.getId(), eventType);

                if (eventType == null) {
                    LOG.warn("CanonicalEvent {} has null eventType with flat payload, cannot determine data type",
                        canonical.getId());
                    return;
                }

                switch (eventType) {
                    case VITAL_SIGN:
                    case VITAL_SIGNS:
                        // Entire payload is vital signs data
                        VitalsPayload vitalsPayload = convertToVitalsPayload(payload);
                        GenericEvent vitalEvent = new GenericEvent();
                        vitalEvent.setEventType("VITAL_SIGN");
                        vitalEvent.setPatientId(patientId);
                        vitalEvent.setEventTime(eventTime);
                        vitalEvent.setPayload(vitalsPayload);
                        vitalEvent.setEncounterId(canonical.getEncounterId());
                        vitalEvent.setSource(sourceSystem);
                        out.collect(vitalEvent);
                        LOG.info("Emitted VITAL_SIGN GenericEvent from flat payload for patient {}", patientId);
                        break;

                    case LAB_RESULT:
                    case DIAGNOSTIC_RESULT:
                        // Entire payload is lab result data
                        // Assume payload has lab fields directly
                        LabPayload labPayload = convertFlatPayloadToLabPayload(payload);
                        if (labPayload != null) {
                            GenericEvent labEvent = new GenericEvent();
                            labEvent.setEventType("LAB_RESULT");
                            labEvent.setPatientId(patientId);
                            labEvent.setEventTime(eventTime);
                            labEvent.setPayload(labPayload);
                            labEvent.setEncounterId(canonical.getEncounterId());
                            labEvent.setSource(sourceSystem);
                            out.collect(labEvent);
                            LOG.info("Emitted LAB_RESULT GenericEvent from flat payload for patient {}", patientId);
                        }
                        break;

                    case MEDICATION_ORDERED:
                    case MEDICATION_PRESCRIBED:
                    case MEDICATION_ADMINISTERED:
                    case MEDICATION_DISCONTINUED:
                    case MEDICATION_MISSED:
                        // Entire payload is medication data
                        MedicationPayload medPayload = convertFlatPayloadToMedicationPayload(payload);
                        if (medPayload != null) {
                            GenericEvent medEvent = new GenericEvent();
                            // CRITICAL FIX: Preserve original medication event type instead of hardcoding "MEDICATION_UPDATE"
                            // This ensures PatientContextAggregator routing works correctly
                            medEvent.setEventType(eventType.name());
                            medEvent.setPatientId(patientId);
                            medEvent.setEventTime(eventTime);
                            medEvent.setPayload(medPayload);
                            medEvent.setEncounterId(canonical.getEncounterId());
                            medEvent.setSource(sourceSystem);
                            out.collect(medEvent);
                            LOG.info("Emitted {} GenericEvent from flat payload for patient {}", eventType.name(), patientId);
                        }
                        break;

                    case PATIENT_REPORTED:
                        // V4: Patient-reported outcomes (PROs) — symptoms, questionnaires, lifestyle data
                        GenericEvent proEvent = new GenericEvent();
                        proEvent.setEventType("PATIENT_REPORTED");
                        proEvent.setPatientId(patientId);
                        proEvent.setEventTime(eventTime);
                        proEvent.setPayload(payload);  // Pass raw payload — PRO structure varies
                        proEvent.setEncounterId(canonical.getEncounterId());
                        proEvent.setSource(sourceSystem);
                        out.collect(proEvent);
                        LOG.debug("Emitted PATIENT_REPORTED GenericEvent from flat payload for patient {}", patientId);
                        break;

                    case CLINICAL_DOCUMENT:
                        // V4: Clinical documents (discharge summaries, notes, imaging reports)
                        GenericEvent docEvent = new GenericEvent();
                        docEvent.setEventType("CLINICAL_DOCUMENT");
                        docEvent.setPatientId(patientId);
                        docEvent.setEventTime(eventTime);
                        docEvent.setPayload(payload);  // Pass raw payload — document structure varies
                        docEvent.setEncounterId(canonical.getEncounterId());
                        docEvent.setSource(sourceSystem);
                        out.collect(docEvent);
                        LOG.debug("Emitted CLINICAL_DOCUMENT GenericEvent from flat payload for patient {}", patientId);
                        break;

                    default:
                        LOG.warn("Unsupported eventType {} for flat payload conversion", eventType);
                        break;
                }
            }
        }

        // Keys already extracted into VitalsPayload typed fields — exclude from additionalVitals
        // to prevent duplication in PatientContextState.latestVitals (Caveat 2 from TIER_1_CGM trace).
        private static final Set<String> EXTRACTED_VITAL_KEYS = Set.of(
            "heartrate", "heartRate",
            "systolicbloodpressure", "systolicBP",
            "diastolicbloodpressure", "diastolicBP",
            "oxygensaturation", "oxygenSaturation", "spo2",
            "respiratoryrate", "respiratoryRate", "rr",
            "temperature", "bodyTemperature", "temp"
        );

        private VitalsPayload convertToVitalsPayload(Map<String, Object> vitals) {
            VitalsPayload payload = new VitalsPayload();

            // Extract vital signs with fallback key handling
            payload.setHeartRate(extractInteger(vitals, "heartrate", "heartRate"));
            payload.setSystolicBP(extractInteger(vitals, "systolicbloodpressure", "systolicBP"));
            payload.setDiastolicBP(extractInteger(vitals, "diastolicbloodpressure", "diastolicBP"));
            payload.setOxygenSaturation(extractInteger(vitals, "oxygensaturation", "oxygenSaturation", "spo2"));
            payload.setRespiratoryRate(extractInteger(vitals, "respiratoryrate", "respiratoryRate", "rr"));
            payload.setTemperature(extractDouble(vitals, "temperature", "bodyTemperature", "temp"));

            // Store only non-vital metadata fields (data_tier, device_type, source_system, etc.)
            // in additionalVitals. Already-extracted vital sign keys are excluded to prevent
            // duplication when toVitalsMap() merges additionalVitals back into the vitals map.
            Map<String, Object> metadata = new HashMap<>();
            for (Map.Entry<String, Object> entry : vitals.entrySet()) {
                if (!EXTRACTED_VITAL_KEYS.contains(entry.getKey())) {
                    metadata.put(entry.getKey(), entry.getValue());
                }
            }
            payload.setAdditionalVitals(metadata);
            return payload;
        }

        private LabPayload convertToLabPayload(String key, Object value) {
            LabPayload payload = new LabPayload();
            payload.setLoincCode(key);  // Assume key is LOINC code or lab name

            if (value instanceof LabResult) {
                LabResult lab = (LabResult) value;
                payload.setLabName(lab.getLabType());
                payload.setValue(lab.getValue());
                payload.setUnit(lab.getUnit());
                payload.setAbnormal(lab.isAbnormal());
                // Note: LabPayload doesn't have timestamp field - it's in GenericEvent
            } else if (value instanceof Number) {
                payload.setValue(((Number) value).doubleValue());
            } else if (value instanceof Map) {
                Map<String, Object> labMap = (Map<String, Object>) value;
                payload.setValue(extractDouble(labMap, "value"));
                payload.setUnit((String) labMap.get("unit"));
            }

            return payload;
        }

        private MedicationPayload convertToMedicationPayload(Object medObj) {
            if (medObj instanceof Medication) {
                Medication med = (Medication) medObj;
                MedicationPayload payload = new MedicationPayload();
                payload.setRxNormCode(med.getCode());
                payload.setMedicationName(med.getName());
                payload.setGenericName(med.getName());  // Assume name is generic
                payload.setRoute(med.getRoute());
                payload.setFrequency(med.getFrequency());
                payload.setAdministrationStatus(med.getStatus());

                if (med.getStartDate() != null) {
                    payload.setStartTime(med.getStartDate());
                }

                return payload;
            }
            return null;
        }

        private Integer extractInteger(Map<String, Object> map, String... keys) {
            for (String key : keys) {
                Object value = map.get(key);
                if (value instanceof Integer) return (Integer) value;
                if (value instanceof Number) return ((Number) value).intValue();
                if (value instanceof String) {
                    try {
                        return Integer.parseInt((String) value);
                    } catch (NumberFormatException e) {
                        continue;
                    }
                }
            }
            return null;
        }

        private Double extractDouble(Map<String, Object> map, String... keys) {
            for (String key : keys) {
                Object value = map.get(key);
                if (value instanceof Double) return (Double) value;
                if (value instanceof Number) return ((Number) value).doubleValue();
                if (value instanceof String) {
                    try {
                        return Double.parseDouble((String) value);
                    } catch (NumberFormatException e) {
                        continue;
                    }
                }
            }
            return null;
        }

        /**
         * Extract string with multiple fallback keys (for Module 1 lowercase key compatibility)
         */
        private String extractString(Map<String, Object> map, String... keys) {
            for (String key : keys) {
                Object value = map.get(key);
                if (value != null) {
                    return value.toString();
                }
            }
            return null;
        }

        /**
         * Convert flat payload to LabPayload when eventType=LAB_RESULT
         * Assumes payload contains lab fields directly (labName, value, unit, etc.)
         */
        private LabPayload convertFlatPayloadToLabPayload(Map<String, Object> payload) {
            LabPayload labPayload = new LabPayload();

            // Extract common lab fields from flat structure
            // Must handle 3 naming conventions:
            //   camelCase (labName)     — Module 2 internal / LabPayload serialization
            //   lowercase (labname)     — Module 1 lowercased keys
            //   snake_case (lab_name)   — RawEvent / ingestion service payload keys
            String labName = (String) payload.getOrDefault("labName",
                    payload.getOrDefault("labname", payload.get("lab_name")));
            String loincCode = (String) payload.getOrDefault("loincCode",
                    payload.getOrDefault("loinccode", payload.get("loinc_code")));
            Object valueObj = payload.get("value");
            String unit = (String) payload.get("unit");
            Boolean abnormal = (Boolean) payload.get("abnormal");
            String abnormalFlag = (String) payload.getOrDefault("abnormalFlag",
                    payload.getOrDefault("abnormalflag", payload.get("abnormal_flag")));

            // Reference ranges — check camelCase, lowercase, and snake_case
            Object refLowObj = payload.getOrDefault("referenceRangeLow",
                    payload.getOrDefault("referencerangelow", payload.get("reference_range_low")));
            Object refHighObj = payload.getOrDefault("referenceRangeHigh",
                    payload.getOrDefault("referencerangehigh", payload.get("reference_range_high")));

            labPayload.setLabName(labName);
            labPayload.setLoincCode(loincCode != null ? loincCode : labName);
            labPayload.setAbnormalFlag(abnormalFlag);

            if (valueObj instanceof Number) {
                labPayload.setValue(((Number) valueObj).doubleValue());
            }

            labPayload.setUnit(unit);
            if (abnormal != null) {
                labPayload.setAbnormal(abnormal);
            }

            // Set reference ranges
            if (refLowObj instanceof Number) {
                labPayload.setReferenceRangeLow(((Number) refLowObj).doubleValue());
            }
            if (refHighObj instanceof Number) {
                labPayload.setReferenceRangeHigh(((Number) refHighObj).doubleValue());
            }

            // Calculate abnormal status if not set
            labPayload.calculateAbnormalStatus();

            return labPayload;
        }

        /**
         * Convert flat payload to MedicationPayload when eventType=MEDICATION_UPDATE
         * Assumes payload contains medication fields directly (medicationName, dose, route, etc.)
         */
        private MedicationPayload convertFlatPayloadToMedicationPayload(Map<String, Object> payload) {
            MedicationPayload medPayload = new MedicationPayload();

            // Extract common medication fields from flat structure with fallback keys
            // Must handle 3 naming conventions: camelCase, lowercase, and snake_case
            String medName = extractString(payload, "medicationName", "medicationname", "medication_name");
            String genericName = extractString(payload, "genericName", "genericname", "generic_name");
            Object rxNormCodeObj = payload.getOrDefault("rxNormCode",
                    payload.getOrDefault("rxnormcode", payload.get("rx_norm_code")));
            String rxNormCode = (rxNormCodeObj != null) ? rxNormCodeObj.toString() : null;
            // Also check medication_code as alternate key for medication identifier
            if (rxNormCode == null) {
                Object medCodeObj = payload.getOrDefault("medicationCode",
                        payload.getOrDefault("medicationcode", payload.get("medication_code")));
                rxNormCode = (medCodeObj != null) ? medCodeObj.toString() : null;
            }
            String route = extractString(payload, "route");
            String frequency = extractString(payload, "frequency");
            String status = extractString(payload, "status", "administrationStatus", "administrationstatus", "administration_status");
            Object doseObj = payload.getOrDefault("dose",
                    payload.getOrDefault("doseAmount", payload.get("dose_amount")));
            String doseUnit = extractString(payload, "doseUnit", "doseunit", "dose_unit");
            Object startTimeObj = payload.getOrDefault("startTime",
                    payload.getOrDefault("starttime", payload.get("start_time")));
            String brandName = extractString(payload, "brandName", "brandname", "brand_name");
            String therapeuticClass = extractString(payload, "therapeuticClass", "therapeuticclass", "therapeutic_class");
            // Also pick up dosage text (from ingestion service payload format)
            String dosageText = extractString(payload, "dosage", "dosageText", "dosage_text");

            medPayload.setMedicationName(medName);
            medPayload.setGenericName(genericName != null ? genericName : medName);
            medPayload.setRxNormCode(rxNormCode);
            medPayload.setRoute(route);
            medPayload.setFrequency(frequency);
            medPayload.setAdministrationStatus(status);
            medPayload.setBrandName(brandName);
            medPayload.setTherapeuticClass(therapeuticClass);
            medPayload.setDoseUnit(doseUnit);

            if (doseObj instanceof Number) {
                medPayload.setDose(((Number) doseObj).doubleValue());
            }

            if (startTimeObj instanceof Number) {
                medPayload.setStartTime(((Number) startTimeObj).longValue());
            }

            return medPayload;
        }
    }

    /**
     * Create Kafka sink for EnrichedPatientContext
     * Outputs to clinical-patterns.v1 topic (same as original pipeline)
     */
    private static KafkaSink<GenericEvent> createDlqSink(String topic) {
        String kafkaBootstrapServers = System.getenv().getOrDefault("KAFKA_BOOTSTRAP_SERVERS", "kafka:29092");

        return org.apache.flink.connector.kafka.sink.KafkaSink.<GenericEvent>builder()
                .setBootstrapServers(kafkaBootstrapServers)
                .setRecordSerializer(
                        org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema.builder()
                                .setTopic(topic)
                                .setValueSerializationSchema(new SafeGenericEventSerializer())
                                .build()
                )
                .setDeliveryGuarantee(org.apache.flink.connector.base.DeliveryGuarantee.AT_LEAST_ONCE)
                .build();
    }

    private static class SafeGenericEventSerializer implements org.apache.flink.api.common.serialization.SerializationSchema<GenericEvent> {
        private static final ObjectMapper MAPPER = new ObjectMapper();
        static {
            MAPPER.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(GenericEvent event) {
            try {
                return MAPPER.writeValueAsBytes(event);
            } catch (Exception e) {
                // For DLQ, return a best-effort JSON with error info rather than throwing
                String fallback = String.format("{\"error\":\"serialization_failed\",\"patientId\":\"%s\",\"eventType\":\"%s\"}",
                        event != null ? event.getPatientId() : "null",
                        event != null ? event.getEventType() : "null");
                return fallback.getBytes(java.nio.charset.StandardCharsets.UTF_8);
            }
        }
    }

    private static KafkaSink<EnrichedPatientContext> createEnrichedPatientContextSink() {
        String kafkaBootstrapServers = System.getenv().getOrDefault("KAFKA_BOOTSTRAP_SERVERS", "kafka:29092");

        return org.apache.flink.connector.kafka.sink.KafkaSink.<EnrichedPatientContext>builder()
                .setBootstrapServers(kafkaBootstrapServers)
                .setRecordSerializer(
                        org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema.builder()
                                .setTopic("clinical-patterns.v1")
                                .setValueSerializationSchema(new org.apache.flink.api.common.serialization.SerializationSchema<EnrichedPatientContext>() {
                                    private static final ObjectMapper MAPPER = new ObjectMapper();
                                    static {
                                        MAPPER.registerModule(new JavaTimeModule());
                                    }

                                    @Override
                                    public byte[] serialize(EnrichedPatientContext element) {
                                        try {
                                            return MAPPER.writeValueAsBytes(element);
                                        } catch (Exception e) {
                                            LOG.error("Failed to serialize EnrichedPatientContext for patient {}: {}",
                                                element != null ? element.getPatientId() : "null", e.getMessage());
                                            throw new RuntimeException(
                                                "EnrichedPatientContext serialization failed", e);
                                        }
                                    }
                                })
                                .build()
                )
                .setTransactionalIdPrefix("module2-unified-enriched-context")
                .setDeliveryGuarantee(org.apache.flink.connector.base.DeliveryGuarantee.AT_LEAST_ONCE)
                .build();
    }

}