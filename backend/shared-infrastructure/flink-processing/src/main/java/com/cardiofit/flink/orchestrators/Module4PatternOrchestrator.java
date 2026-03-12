package com.cardiofit.flink.orchestrators;

import com.cardiofit.flink.models.PatternEvent;
import com.cardiofit.flink.models.SemanticEvent;
import com.cardiofit.flink.models.MLPrediction;
import com.cardiofit.flink.functions.ClinicalConditionDetector;
import com.cardiofit.flink.functions.ClinicalMessageBuilder;
import com.cardiofit.flink.functions.PatternDeduplicationFunction;
import com.cardiofit.flink.functions.MLToPatternConverter;
import com.cardiofit.flink.patterns.ClinicalPatterns;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import org.apache.flink.cep.PatternStream;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.KeyedStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Module 4 Pattern Detection Orchestrator
 *
 * Implements clean separation of concerns for multi-layer pattern detection:
 * - Layer 1: Instant State Assessment (State-Based Reasoning - "Triage Nurse")
 * - Layer 2: Complex Event Processing (CEP Temporal Patterns)
 * - Layer 3: ML Predictive Analysis (Future Integration)
 *
 * This orchestrator separates immediate clinical assessment from temporal pattern detection,
 * providing both instant triage decisions and longitudinal event sequence analysis.
 */
public class Module4PatternOrchestrator {

    private static final Logger LOG = LoggerFactory.getLogger(Module4PatternOrchestrator.class);

    /**
     * Main orchestration method that coordinates all pattern detection layers.
     *
     * @param semanticEvents Input stream of semantic events from Module 3
     * @param env Flink execution environment
     * @return Unified, deduplicated stream of pattern events for Module 5
     */
    public static DataStream<PatternEvent> orchestrate(
            DataStream<SemanticEvent> semanticEvents,
            StreamExecutionEnvironment env) {

        LOG.info("🎯 MODULE 4 PATTERN ORCHESTRATOR - Starting multi-layer pattern detection");

        // Key semantic events by patient ID for stateful operations
        KeyedStream<SemanticEvent, String> keyedSemanticEvents = semanticEvents
            .keyBy(SemanticEvent::getPatientId);

        // ===== LAYER 1: INSTANT STATE ASSESSMENT =====
        // Provides immediate clinical assessment without waiting for event sequences
        // Acts as "Triage Nurse" - instant evaluation of current patient state
        DataStream<PatternEvent> instantPatterns = instantStateAssessment(semanticEvents);

        // ===== LAYER 2: COMPLEX EVENT PROCESSING (CEP) =====
        // Detects temporal patterns across event sequences
        // Identifies deterioration trajectories, medication patterns, trends
        DataStream<PatternEvent> cepPatterns = cepPatternDetection(keyedSemanticEvents);

        // ===== LAYER 3: ML PREDICTIVE ANALYSIS =====
        // Consumes ML predictions from Module 5 and integrates into pattern stream
        // Provides forward-looking clinical intelligence
        DataStream<PatternEvent> mlPatterns = mlPredictiveAnalysis(env);

        // ===== PATTERN UNIFICATION =====
        // Combine all pattern detection layers (Layer 1 + Layer 2 + Layer 3)
        DataStream<PatternEvent> allPatterns = instantPatterns
            .union(cepPatterns)
            .union(mlPatterns);

        // ===== DEDUPLICATION =====
        // Remove duplicate patterns using intelligent deduplication logic
        DataStream<PatternEvent> dedupedPatterns = allPatterns
            .keyBy(PatternEvent::getPatientId)
            .process(new PatternDeduplicationFunction());

        LOG.info("✅ MODULE 4 PATTERN ORCHESTRATOR - Multi-layer pattern detection configured");

        return dedupedPatterns;
    }

    /**
     * Layer 1: Instant State Assessment
     *
     * Implements STATE-BASED REASONING - immediate clinical assessment without temporal dependencies.
     * Every semantic event is instantly converted to a comprehensive pattern event with:
     * - Condition detection (respiratory failure, shock, sepsis)
     * - Automatic clinical actions based on independent condition assessment
     * - Human-readable clinical messages
     * - Complete metadata and quality scoring
     *
     * This layer acts as the "Triage Nurse" - providing instant clinical judgment
     * based solely on current patient state, not historical sequences.
     *
     * @param semanticEvents Input stream of semantic events
     * @return Stream of instant pattern events
     */
    private static DataStream<PatternEvent> instantStateAssessment(
            DataStream<SemanticEvent> semanticEvents) {

        LOG.info("🏥 Layer 1: Instant State Assessment - Activating state-based clinical reasoning");

        return semanticEvents
            .map(semanticEvent -> {
                long startTime = System.nanoTime();

                PatternEvent pe = new PatternEvent();

                // ===== BASIC IDENTIFICATION =====
                pe.setId(java.util.UUID.randomUUID().toString());

                // Use condition-specific detection instead of generic pass-through
                String conditionType = ClinicalConditionDetector.determineConditionType(semanticEvent);
                pe.setPatternType(conditionType);

                pe.setPatientId(semanticEvent.getPatientId());
                pe.setEncounterId(semanticEvent.getEncounterId());
                pe.setDetectionTime(System.currentTimeMillis());
                pe.setCorrelationId(semanticEvent.getCorrelationId());

                // ===== TEMPORAL CONTEXT =====
                pe.setPatternStartTime(semanticEvent.getEventTime());
                pe.setPatternEndTime(semanticEvent.getEventTime()); // Single event

                // ===== SEVERITY & CONFIDENCE =====
                String riskLevel = semanticEvent.getRiskLevel();
                pe.setSeverity(riskLevel != null ? riskLevel.toUpperCase() : "UNKNOWN");
                pe.setConfidence(semanticEvent.getClinicalSignificance());

                // ===== INVOLVED EVENTS =====
                pe.addInvolvedEvent(semanticEvent.getId());

                // ===== RECOMMENDED ACTIONS FROM SEMANTIC EVENT =====
                if (semanticEvent.hasClinicalAlerts()) {
                    for (SemanticEvent.ClinicalAlert alert : semanticEvent.getClinicalAlerts()) {
                        if (alert.getRecommendedAction() != null && !alert.getRecommendedAction().isEmpty()) {
                            pe.addRecommendedAction(alert.getRecommendedAction());
                        }
                    }
                }

                if (semanticEvent.hasGuidelineRecommendations()) {
                    for (SemanticEvent.GuidelineRecommendation rec : semanticEvent.getGuidelineRecommendations()) {
                        if (rec.getRecommendation() != null && !rec.getRecommendation().isEmpty()) {
                            pe.addRecommendedAction(rec.getRecommendation());
                        }
                    }
                }

                // Add condition-specific automatic actions based on independent clinical detection
                if (ClinicalConditionDetector.hasRespiratoryFailure(semanticEvent)) {
                    pe.addRecommendedAction("CRITICAL: Assess airway, breathing, circulation");
                    pe.addRecommendedAction("Consider supplemental oxygen or escalation to high-flow");
                    pe.addRecommendedAction("Prepare for possible intubation if deteriorating");
                    pe.addRecommendedAction("Notify respiratory therapy STAT");
                    pe.addRecommendedAction("Arterial blood gas if not recent");
                    pe.addRecommendedAction("ESCALATE_TO_RAPID_RESPONSE");
                    pe.addRecommendedAction("NOTIFY_CARE_TEAM");
                }
                else if (ClinicalConditionDetector.isInShock(semanticEvent)) {
                    pe.addRecommendedAction("CRITICAL: Immediate fluid resuscitation");
                    pe.addRecommendedAction("Establish large-bore IV access x2");
                    pe.addRecommendedAction("Administer 500ml bolus crystalloid stat");
                    pe.addRecommendedAction("Consider vasopressor support if MAP < 65");
                    pe.addRecommendedAction("Urgent ICU consultation");
                    pe.addRecommendedAction("ESCALATE_TO_RAPID_RESPONSE");
                    pe.addRecommendedAction("NOTIFY_CARE_TEAM");
                }
                else if (ClinicalConditionDetector.meetsSepsisCriteria(semanticEvent)) {
                    pe.addRecommendedAction("Initiate sepsis bundle immediately");
                    pe.addRecommendedAction("Blood cultures x2 before antibiotics");
                    pe.addRecommendedAction("Administer broad-spectrum antibiotics within 1 hour");
                    pe.addRecommendedAction("Measure serum lactate");
                    pe.addRecommendedAction("Begin fluid resuscitation 30ml/kg crystalloid");
                    pe.addRecommendedAction("ESCALATE_TO_RAPID_RESPONSE");
                    pe.addRecommendedAction("NOTIFY_CARE_TEAM");
                }
                else if (ClinicalConditionDetector.isCriticalState(semanticEvent)) {
                    pe.addRecommendedAction("IMMEDIATE_ASSESSMENT_REQUIRED");
                    pe.addRecommendedAction("INCREASE_MONITORING_FREQUENCY");
                    pe.addRecommendedAction("ESCALATE_TO_RAPID_RESPONSE");
                    pe.addRecommendedAction("NOTIFY_CARE_TEAM");
                    pe.addRecommendedAction("Continuous vital signs monitoring");
                    pe.addRecommendedAction("Consider ICU transfer");
                }
                else if (ClinicalConditionDetector.isHighRiskState(semanticEvent)) {
                    pe.addRecommendedAction("IMMEDIATE_ASSESSMENT_REQUIRED");
                    pe.addRecommendedAction("INCREASE_MONITORING_FREQUENCY");
                    pe.addRecommendedAction("Vital signs every 15-30 minutes");
                    pe.addRecommendedAction("Notify attending physician");
                }
                else if ("MODERATE".equalsIgnoreCase(riskLevel)) {
                    pe.addRecommendedAction("REASSESS_IN_30_MINUTES");
                    pe.addRecommendedAction("VITAL_SIGNS_Q30MIN");
                }

                // ===== CLINICAL CONTEXT =====
                PatternEvent.ClinicalContext clinicalContext = new PatternEvent.ClinicalContext();

                // Extract from patient context if available
                if (semanticEvent.getPatientContext() != null) {
                    // Department/unit would come from patient context
                    clinicalContext.setAcuityLevel(riskLevel);
                }

                // Set active problems from clinical alerts
                if (semanticEvent.hasClinicalAlerts()) {
                    List<String> activeProblems = new ArrayList<>();
                    for (SemanticEvent.ClinicalAlert alert : semanticEvent.getClinicalAlerts()) {
                        if (alert.getMessage() != null && !alert.getMessage().isEmpty()) {
                            activeProblems.add(alert.getMessage());
                        }
                    }
                    clinicalContext.setActiveProblems(activeProblems);
                }

                pe.setClinicalContext(clinicalContext);

                // ===== PATTERN DETAILS (Extended Information) =====
                Map<String, Object> patternDetails = new HashMap<>();

                // Build human-readable clinical message (Gap 3)
                String clinicalMessage = ClinicalMessageBuilder.buildMessage(semanticEvent, conditionType);
                patternDetails.put("clinicalMessage", clinicalMessage);

                // Add structured clinical scores for downstream parsing
                if (semanticEvent.getClinicalData() != null) {
                    Map<String, Object> clinicalData = semanticEvent.getClinicalData();

                    // Extract NEWS2 score
                    Integer news2 = extractIntegerFromMap(clinicalData, "news2", "NEWS2", "news2Score");
                    if (news2 == null) {
                        // Try nested clinicalScores
                        Object clinicalScores = clinicalData.get("clinicalScores");
                        if (clinicalScores instanceof Map) {
                            news2 = extractIntegerFromMap((Map<String, Object>) clinicalScores, "news2", "NEWS2");
                        }
                    }
                    if (news2 != null) {
                        patternDetails.put("news2Score", news2);
                    }

                    // Extract qSOFA score
                    Integer qsofa = extractIntegerFromMap(clinicalData, "qsofa", "qSOFA", "qsofaScore");
                    if (qsofa == null) {
                        // Try nested clinicalScores
                        Object clinicalScores = clinicalData.get("clinicalScores");
                        if (clinicalScores instanceof Map) {
                            qsofa = extractIntegerFromMap((Map<String, Object>) clinicalScores, "qsofa", "qSOFA");
                        }
                    }
                    if (qsofa != null) {
                        patternDetails.put("qsofaScore", qsofa);
                    }

                    // Extract combined acuity (same as confidence)
                    Double acuity = semanticEvent.getClinicalSignificance();
                    if (acuity != null) {
                        patternDetails.put("combinedAcuity", acuity);
                    }
                }

                // Extract current vitals from originalPayload
                if (semanticEvent.getOriginalPayload() != null) {
                    Object vitals = semanticEvent.getOriginalPayload().get("vitals");
                    if (vitals instanceof Map) {
                        Map<String, Object> vitalsMap = (Map<String, Object>) vitals;
                        Map<String, Object> structuredVitals = new HashMap<>();

                        // Extract key vital signs
                        structuredVitals.put("heartRate", getDoubleValue(vitalsMap, "heartRate"));
                        structuredVitals.put("systolicBP", getDoubleValue(vitalsMap, "systolicBP"));
                        structuredVitals.put("diastolicBP", getDoubleValue(vitalsMap, "diastolicBP"));
                        structuredVitals.put("respiratoryRate", getDoubleValue(vitalsMap, "respiratoryRate"));
                        structuredVitals.put("oxygenSaturation", getDoubleValue(vitalsMap, "oxygenSaturation", "spO2"));
                        structuredVitals.put("temperature", getDoubleValue(vitalsMap, "temperature"));

                        // Calculate shock index if available
                        Double hr = (Double) structuredVitals.get("heartRate");
                        Double sbp = (Double) structuredVitals.get("systolicBP");
                        if (hr != null && sbp != null && sbp > 0) {
                            structuredVitals.put("shockIndex", hr / sbp);
                        }

                        patternDetails.put("currentVitals", structuredVitals);
                    }
                }

                // Add event type info
                patternDetails.put("eventType", semanticEvent.getEventType() != null ?
                    semanticEvent.getEventType().toString() : "UNKNOWN");

                // Add temporal context
                String temporalContext = semanticEvent.getTemporalContext();
                if (temporalContext != null) {
                    patternDetails.put("temporalContext", temporalContext);
                    patternDetails.put("isAcute", semanticEvent.isAcute());
                }

                // Add clinical alerts summary
                if (semanticEvent.hasClinicalAlerts()) {
                    int criticalAlerts = 0, highAlerts = 0, moderateAlerts = 0;
                    for (SemanticEvent.ClinicalAlert alert : semanticEvent.getClinicalAlerts()) {
                        String severity = alert.getSeverity();
                        if ("CRITICAL".equalsIgnoreCase(severity)) criticalAlerts++;
                        else if ("HIGH".equalsIgnoreCase(severity)) highAlerts++;
                        else if ("MODERATE".equalsIgnoreCase(severity)) moderateAlerts++;
                    }
                    patternDetails.put("criticalAlerts", criticalAlerts);
                    patternDetails.put("highAlerts", highAlerts);
                    patternDetails.put("moderateAlerts", moderateAlerts);
                    patternDetails.put("totalAlerts", semanticEvent.getClinicalAlerts().size());
                }

                // Add drug interactions
                if (semanticEvent.hasDrugInteractions()) {
                    patternDetails.put("hasDrugInteractions", true);
                    patternDetails.put("drugInteractionCount", semanticEvent.getDrugInteractions().size());
                }

                // Add source system
                if (semanticEvent.getSourceSystem() != null) {
                    patternDetails.put("sourceSystem", semanticEvent.getSourceSystem());
                }

                // Add semantic quality metrics
                if (semanticEvent.getSemanticQuality() != null) {
                    Map<String, Object> quality = new HashMap<>();
                    quality.put("completeness", semanticEvent.getSemanticQuality().getCompleteness());
                    quality.put("accuracy", semanticEvent.getSemanticQuality().getAccuracy());
                    quality.put("overallScore", semanticEvent.getSemanticQuality().getOverallScore());
                    patternDetails.put("semanticQuality", quality);
                }

                pe.setPatternDetails(patternDetails);

                // ===== PATTERN METADATA =====
                PatternEvent.PatternMetadata metadata = new PatternEvent.PatternMetadata();
                metadata.setAlgorithm("STATE_BASED_IMMEDIATE_ASSESSMENT");
                metadata.setVersion("1.0.0");

                Map<String, Object> algorithmParams = new HashMap<>();
                algorithmParams.put("minConfidence", 0.0);
                algorithmParams.put("assessmentMode", "IMMEDIATE");
                algorithmParams.put("reasoningType", "STATE_BASED");
                metadata.setAlgorithmParameters(algorithmParams);

                long endTime = System.nanoTime();
                double processingTimeMs = (endTime - startTime) / 1_000_000.0;
                metadata.setProcessingTime(processingTimeMs);

                // Quality scoring based on semantic event quality
                if (semanticEvent.getSemanticQuality() != null &&
                    semanticEvent.getSemanticQuality().isHighQuality()) {
                    metadata.setQualityScore("HIGH");
                } else {
                    metadata.setQualityScore("MODERATE");
                }

                pe.setPatternMetadata(metadata);

                // ===== TAGS =====
                pe.addTag("STATE_BASED");
                pe.addTag("IMMEDIATE_ASSESSMENT");
                if (semanticEvent.isAcute()) {
                    pe.addTag("ACUTE");
                }
                if (pe.isHighSeverity()) {
                    pe.addTag("HIGH_SEVERITY");
                }
                if (pe.isHighConfidence()) {
                    pe.addTag("HIGH_CONFIDENCE");
                }
                if (semanticEvent.hasClinicalAlerts()) {
                    pe.addTag("HAS_ALERTS");
                }
                if (semanticEvent.hasDrugInteractions()) {
                    pe.addTag("DRUG_INTERACTIONS");
                }

                LOG.debug("✅ INSTANT PATTERN for patient {}: type={}, severity={}, confidence={:.3f}, actions={}, processingTime={:.2f}ms",
                    semanticEvent.getPatientId(),
                    pe.getPatternType(),
                    pe.getSeverity(),
                    pe.getConfidence(),
                    pe.getRecommendedActions().size(),
                    processingTimeMs);

                return pe;
            });
    }

    /**
     * Layer 2: Complex Event Processing (CEP) Pattern Detection
     *
     * Implements TEMPORAL PATTERN RECOGNITION across event sequences.
     * Detects patterns that emerge over time:
     * - Sepsis development patterns (qSOFA, SIRS progression)
     * - Rapid clinical deterioration
     * - Drug-lab monitoring compliance
     * - Sepsis pathway compliance
     *
     * This layer complements instant assessment by identifying patterns that
     * require temporal context and event sequence analysis.
     *
     * @param keyedSemanticEvents Keyed stream of semantic events (by patient ID)
     * @return Stream of CEP-detected pattern events
     */
    private static DataStream<PatternEvent> cepPatternDetection(
            KeyedStream<SemanticEvent, String> keyedSemanticEvents) {

        LOG.info("⏱️ Layer 2: CEP Pattern Detection - Activating temporal pattern recognition");

        // Sepsis pattern detection (qSOFA progression, SIRS criteria)
        PatternStream<SemanticEvent> sepsisPatterns =
            ClinicalPatterns.detectSepsisPattern(keyedSemanticEvents);

        DataStream<PatternEvent> sepsisEvents = sepsisPatterns
            .select(new ClinicalPatterns.SepsisPatternSelectFunction());

        // Rapid deterioration patterns
        PatternStream<SemanticEvent> rapidDeteriorationPatterns =
            ClinicalPatterns.detectRapidDeteriorationPattern(keyedSemanticEvents);

        DataStream<PatternEvent> rapidDeteriorationEvents = rapidDeteriorationPatterns
            .select(new ClinicalPatterns.RapidDeteriorationPatternSelectFunction());

        // Drug-lab monitoring patterns
        PatternStream<SemanticEvent> drugLabMonitoringPatterns =
            ClinicalPatterns.detectDrugLabMonitoringPattern(keyedSemanticEvents);

        DataStream<PatternEvent> drugLabMonitoringEvents = drugLabMonitoringPatterns
            .select(new ClinicalPatterns.DrugLabMonitoringPatternSelectFunction());

        // Sepsis pathway compliance
        PatternStream<SemanticEvent> sepsisPathwayPatterns =
            ClinicalPatterns.detectSepsisPathwayCompliancePattern(keyedSemanticEvents);

        DataStream<PatternEvent> sepsisPathwayEvents = sepsisPathwayPatterns
            .select(new ClinicalPatterns.SepsisPathwayCompliancePatternSelectFunction());

        // Union all CEP pattern streams
        DataStream<PatternEvent> allCepPatterns = sepsisEvents
            .union(rapidDeteriorationEvents)
            .union(drugLabMonitoringEvents)
            .union(sepsisPathwayEvents);

        LOG.info("✅ Layer 2: CEP Pattern Detection - Temporal pattern recognition configured");

        return allCepPatterns;
    }

    /**
     * Layer 3: ML Predictive Analysis
     *
     * Consumes ML predictions from Module 5 and converts them to PatternEvent format.
     * This allows ML predictions to participate in multi-source confirmation and
     * deduplication alongside Layer 1 (instant state) and Layer 2 (CEP patterns).
     *
     * ML Models supported:
     * - Sepsis onset prediction (6-12 hour horizon)
     * - Mortality risk assessment (30-day horizon)
     * - Respiratory failure prediction (2-4 hour horizon)
     * - AKI progression risk (24-48 hour horizon)
     *
     * @param env Flink execution environment
     * @return Stream of ML-based pattern events
     */
    private static DataStream<PatternEvent> mlPredictiveAnalysis(
            StreamExecutionEnvironment env) {

        LOG.info("🤖 Layer 3: ML Predictive Analysis - Activating predictive intelligence");

        // ═══════════════════════════════════════════════════════════
        // INPUT: ML PREDICTIONS FROM MODULE 5 (KAFKA)
        // ═══════════════════════════════════════════════════════════

        // Create Kafka source for ML predictions
        String topicName = System.getenv().getOrDefault(
            "MODULE4_ML_INPUT_TOPIC",
            "ml-predictions.v1"
        );

        String bootstrapServers = KafkaConfigLoader.isRunningInDocker() ?
            "kafka:29092" : "localhost:9092";

        org.apache.flink.connector.kafka.source.KafkaSource<MLPrediction> mlSource =
            org.apache.flink.connector.kafka.source.KafkaSource.<MLPrediction>builder()
                .setBootstrapServers(bootstrapServers)
                .setTopics(topicName)
                .setGroupId("pattern-detection-ml")
                .setStartingOffsets(org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer.latest())
                .setValueOnlyDeserializer(new MLPredictionDeserializer())
                .setProperties(KafkaConfigLoader.getAutoConsumerConfig("pattern-detection-ml"))
                .build();

        DataStream<MLPrediction> mlPredictions = env
            .fromSource(
                mlSource,
                org.apache.flink.api.common.eventtime.WatermarkStrategy
                    .<MLPrediction>forBoundedOutOfOrderness(java.time.Duration.ofMinutes(2))
                    .withTimestampAssigner((pred, ts) -> pred.getPredictionTime()),
                "ML Predictions from Module 5"
            );

        LOG.info("📥 ML predictions source configured - consuming from {}", topicName);

        // ═══════════════════════════════════════════════════════════
        // CONVERT: MLPrediction → PatternEvent
        // ═══════════════════════════════════════════════════════════

        DataStream<PatternEvent> mlPatterns = mlPredictions
            .map(new MLToPatternConverter());

        LOG.info("🔄 ML to PatternEvent converter configured");
        LOG.info("✅ Layer 3: ML Predictive Analysis - Ready to process predictions");

        return mlPatterns;
    }

    /**
     * Deserializer for ML Prediction events from Kafka
     */
    private static class MLPredictionDeserializer
        implements org.apache.flink.api.common.serialization.DeserializationSchema<MLPrediction> {

        private static final com.fasterxml.jackson.databind.ObjectMapper objectMapper =
            new com.fasterxml.jackson.databind.ObjectMapper();

        @Override
        public MLPrediction deserialize(byte[] message) throws IOException {
            return objectMapper.readValue(message, MLPrediction.class);
        }

        @Override
        public boolean isEndOfStream(MLPrediction nextElement) {
            return false;
        }

        @Override
        public org.apache.flink.api.common.typeinfo.TypeInformation<MLPrediction> getProducedType() {
            return org.apache.flink.api.common.typeinfo.TypeInformation.of(MLPrediction.class);
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // HELPER METHODS FOR CLINICAL DATA EXTRACTION
    // ═══════════════════════════════════════════════════════════════════════

    /**
     * Extract integer value from map trying multiple key variations
     *
     * @param map Map to extract from
     * @param keys Possible key names to try
     * @return Integer value or null if not found
     */
    private static Integer extractIntegerFromMap(Map<String, Object> map, String... keys) {
        if (map == null || keys == null) return null;

        for (String key : keys) {
            Object value = map.get(key);
            if (value instanceof Number) {
                return ((Number) value).intValue();
            }
        }
        return null;
    }

    /**
     * Safely extract double value from vitals map
     * Handles multiple key variations (camelCase, lowercase, snake_case)
     *
     * @param map Map to extract from
     * @param keys Possible key names to try
     * @return Double value or null if not found
     */
    private static Double getDoubleValue(Map<String, Object> map, String... keys) {
        if (map == null || keys == null) return null;

        for (String key : keys) {
            // Try exact key
            Object value = map.get(key);
            if (value instanceof Number) {
                return ((Number) value).doubleValue();
            }

            // Try lowercase
            value = map.get(key.toLowerCase());
            if (value instanceof Number) {
                return ((Number) value).doubleValue();
            }

            // Try snake_case for camelCase keys
            String snakeCase = camelToSnake(key);
            value = map.get(snakeCase);
            if (value instanceof Number) {
                return ((Number) value).doubleValue();
            }
        }

        return null;
    }

    /**
     * Convert camelCase to snake_case
     *
     * @param camel CamelCase string
     * @return snake_case string
     */
    private static String camelToSnake(String camel) {
        return camel.replaceAll("([a-z])([A-Z])", "$1_$2").toLowerCase();
    }
}
