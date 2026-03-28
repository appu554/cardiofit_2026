
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.EnrichedEvent;
import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.models.PatternEvent;
import com.cardiofit.flink.models.PatientContext;
import com.cardiofit.flink.models.PatientContextState;
import com.cardiofit.flink.models.SemanticEvent;
import com.cardiofit.flink.models.MEWSAlert;
import com.cardiofit.flink.models.LabTrendAlert;
import com.cardiofit.flink.models.VitalVariabilityAlert;
import com.cardiofit.flink.models.DailyRiskScore;
import com.cardiofit.flink.models.SimpleAlert;
import com.cardiofit.flink.patterns.ClinicalPatterns;
import com.cardiofit.flink.analytics.MEWSCalculator;
import com.cardiofit.flink.analytics.LabTrendAnalyzer;
import com.cardiofit.flink.analytics.VitalVariabilityAnalyzer;
import com.cardiofit.flink.analytics.RiskScoreCalculator;
import com.cardiofit.flink.functions.ClinicalConditionDetector;
import com.cardiofit.flink.functions.ClinicalMessageBuilder;
import com.cardiofit.flink.functions.PatternDeduplicationFunction;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.functions.AggregateFunction;
import org.apache.flink.api.common.functions.MapFunction;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.api.java.tuple.Tuple2;
import org.apache.flink.cep.CEP;
import org.apache.flink.cep.PatternSelectFunction;
import org.apache.flink.cep.PatternStream;
import org.apache.flink.cep.pattern.Pattern;
import org.apache.flink.cep.pattern.conditions.SimpleCondition;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.base.DeliveryGuarantee;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.KeyedStream;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.windowing.ProcessWindowFunction;
import org.apache.flink.streaming.api.functions.windowing.WindowFunction;
import org.apache.flink.streaming.api.windowing.assigners.SlidingEventTimeWindows;
import org.apache.flink.streaming.api.windowing.assigners.TumblingEventTimeWindows;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;
import java.time.Duration;import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.time.Duration;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Module 4: Pattern Detection (CEP & Windowed Analytics)
 *
 * Responsibilities:
 * - Detect complex clinical patterns using Complex Event Processing (CEP)
 * - Perform windowed analytics for trend detection
 * - Identify deterioration patterns and early warning signs
 * - Generate pathway adherence events
 * - Detect anomalies and outliers in clinical data
 * - Monitor clinical protocols and care pathways
 */
public class Module4_PatternDetection {
    private static final Logger LOG = LoggerFactory.getLogger(Module4_PatternDetection.class);

    // Output tags for different pattern types
    private static final OutputTag<PatternEvent> DETERIORATION_PATTERN_TAG =
        new OutputTag<PatternEvent>("deterioration-patterns"){};

    private static final OutputTag<PatternEvent> PATHWAY_ADHERENCE_TAG =
        new OutputTag<PatternEvent>("pathway-adherence"){};

    private static final OutputTag<PatternEvent> ANOMALY_DETECTION_TAG =
        new OutputTag<PatternEvent>("anomaly-detection"){};

    private static final OutputTag<PatternEvent> TREND_ANALYSIS_TAG =
        new OutputTag<PatternEvent>("trend-analysis"){};

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Module 4: Pattern Detection (CEP & Windowed Analytics)");

        // Set up execution environment
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Configure for pattern processing
        env.setParallelism(2);
        env.enableCheckpointing(30000);
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);

        // Create pattern detection pipeline
        createPatternDetectionPipeline(env);

        // Execute the job
        env.execute("Module 4: Pattern Detection");
    }

    public static void createPatternDetectionPipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating pattern detection pipeline");

        // PRODUCTION: CDS Event source from Module 3 Comprehensive CDS
        DataStream<Module3_ComprehensiveCDS.CDSEvent> cdsEvents = createCDSEventSource(env);

        // Convert CDS events to Semantic events for pattern detection
        DataStream<SemanticEvent> semanticEvents = cdsEvents
            .map(cdsEvent -> convertCDSEventToSemanticEvent(cdsEvent))
            .name("Convert CDS to Semantic Events");

        // Output semantic events to Kafka for Module 5 consumption
        semanticEvents
            .sinkTo(createSemanticEventsSink())
            .uid("Module4 Semantic Events Sink")
            .name("Module4 Semantic Events to Kafka");

        // DEPRECATED: Direct semantic events source (bypasses Module 3)
        // DataStream<SemanticEvent> semanticEvents = createSemanticEventSource(env);

        // DEPRECATED: Direct enriched events from Module 2
        // Use CDS events which contain the full patient state including enriched context
        // DataStream<EnrichedEvent> enrichedEvents = createEnrichedEventSource(env);

        // DEBUG: Log events before keying for CEP input
        DataStream<SemanticEvent> loggedSemanticEvents = semanticEvents
            .process(new org.apache.flink.streaming.api.functions.ProcessFunction<SemanticEvent, SemanticEvent>() {
                @Override
                public void processElement(SemanticEvent event, Context ctx, org.apache.flink.util.Collector<SemanticEvent> out) {
                    LOG.debug("🎯 CEP INPUT - Patient {}: sig={}, risk={}, eventType={}",
                        event.getPatientId(), event.getClinicalSignificance(),
                        event.getRiskLevel(), event.getEventType());
                    out.collect(event);
                }
            })
            .name("Log CEP Input Events");

        // Key by patient ID for pattern detection
        KeyedStream<SemanticEvent, String> keyedSemanticEvents = loggedSemanticEvents
            .keyBy(SemanticEvent::getPatientId);

        // IMMEDIATE: Convert every semantic event to comprehensive pattern event for Module 5
        // This implements STATE-BASED REASONING (The "Triage Nurse")
        // Provides immediate clinical assessment without waiting for event sequences
        DataStream<PatternEvent> immediatePatternEvents = loggedSemanticEvents
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

                // ===== CLINICAL CONTEXT (Gap 7: Complete Context) =====
                PatternEvent.ClinicalContext clinicalContext = new PatternEvent.ClinicalContext();

                // Extract from patient context if available
                if (semanticEvent.getPatientContext() != null) {
                    PatientContext patientCtx = semanticEvent.getPatientContext();

                    // Department and unit extraction from location
                    if (patientCtx.getLocation() != null) {
                        PatientContext.PatientLocation location = patientCtx.getLocation();
                        if (location.getUnit() != null) {
                            clinicalContext.setDepartment(location.getFacility());  // Facility serves as department
                            clinicalContext.setUnit(location.getUnit());
                        }
                    }

                    // Care team
                    if (patientCtx.getCareTeam() != null && !patientCtx.getCareTeam().isEmpty()) {
                        clinicalContext.setCareTeam(String.join(", ", patientCtx.getCareTeam()));
                    }

                    // Primary diagnosis
                    if (patientCtx.getPrimaryDiagnosis() != null) {
                        clinicalContext.setPrimaryDiagnosis(patientCtx.getPrimaryDiagnosis());
                    }

                    // Acuity level
                    clinicalContext.setAcuityLevel(riskLevel);
                }

                // Set active problems from clinical alerts
                if (semanticEvent.hasClinicalAlerts()) {
                    List<String> activeProblems = new java.util.ArrayList<>();
                    List<String> recentAlerts = new java.util.ArrayList<>();

                    for (SemanticEvent.ClinicalAlert alert : semanticEvent.getClinicalAlerts()) {
                        if (alert.getMessage() != null && !alert.getMessage().isEmpty()) {
                            activeProblems.add(alert.getMessage());
                            // Also add to recent alerts with timestamp
                            recentAlerts.add(alert.getSeverity() + ": " + alert.getMessage());
                        }
                    }
                    clinicalContext.setActiveProblems(activeProblems);

                    // Merge with existing recent alerts if any
                    if (clinicalContext.getRecentAlerts() == null) {
                        clinicalContext.setRecentAlerts(recentAlerts);
                    } else {
                        clinicalContext.getRecentAlerts().addAll(recentAlerts);
                    }
                }

                pe.setClinicalContext(clinicalContext);

                // ===== PATTERN DETAILS (Extended Information) =====
                Map<String, Object> patternDetails = new java.util.HashMap<>();

                // Build human-readable clinical message (Gap 3)
                String clinicalMessage = ClinicalMessageBuilder.buildMessage(semanticEvent, conditionType);
                patternDetails.put("clinicalMessage", clinicalMessage);

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
                    Map<String, Object> quality = new java.util.HashMap<>();
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

                Map<String, Object> algorithmParams = new java.util.HashMap<>();
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

                LOG.debug("✅ COMPREHENSIVE IMMEDIATE PATTERN for patient {}: type={}, severity={}, confidence={:.3f}, actions={}, alerts={}, processingTime={:.2f}ms",
                    semanticEvent.getPatientId(),
                    pe.getPatternType(),
                    pe.getSeverity(),
                    pe.getConfidence(),
                    pe.getRecommendedActions().size(),
                    semanticEvent.hasClinicalAlerts() ? semanticEvent.getClinicalAlerts().size() : 0,
                    processingTimeMs);

                return pe;
            })
            .name("Comprehensive Immediate Pattern Events");

        // ===== Complex Event Processing (CEP) Patterns =====

        // Clinical deterioration patterns
        PatternStream<SemanticEvent> deteriorationPatterns = detectDeteriorationPatterns(keyedSemanticEvents);

        // V4: Cross-domain deterioration (glycaemic + hemodynamic concurrent decline)
        PatternStream<SemanticEvent> crossDomainPatterns = detectCrossDomainDeteriorationPatterns(keyedSemanticEvents);

        // Acute Kidney Injury detection pattern (KDIGO criteria) - uses EnrichedEvent with RiskIndicators
        // TODO: Extract risk indicators from CDS event for AKI pattern detection
        // PatternStream<EnrichedEvent> akiPatterns = ClinicalPatterns.detectAKIPattern(enrichedEvents);

        // Medication adherence patterns
        PatternStream<SemanticEvent> medicationPatterns = detectMedicationPatterns(keyedSemanticEvents);

        // Vital signs trend patterns
        PatternStream<SemanticEvent> vitalTrendPatterns = detectVitalTrendPatterns(keyedSemanticEvents);

        // Clinical pathway compliance patterns
        PatternStream<SemanticEvent> pathwayPatterns = detectPathwayCompliancePatterns(keyedSemanticEvents);

        // NEW: Advanced CEP Patterns from Phase 2
        PatternStream<SemanticEvent> sepsisPatterns = ClinicalPatterns.detectSepsisPattern(keyedSemanticEvents);
        PatternStream<SemanticEvent> rapidDeteriorationPatterns = ClinicalPatterns.detectRapidDeteriorationPattern(keyedSemanticEvents);
        PatternStream<SemanticEvent> drugLabMonitoringPatterns = ClinicalPatterns.detectDrugLabMonitoringPattern(keyedSemanticEvents);
        PatternStream<SemanticEvent> sepsisPathwayPatterns = ClinicalPatterns.detectSepsisPathwayCompliancePattern(keyedSemanticEvents);

        // ===== Windowed Analytics =====

        // Trend analysis using sliding windows
        DataStream<PatternEvent> trendAnalysis = performTrendAnalysis(keyedSemanticEvents);

        // Anomaly detection using tumbling windows
        DataStream<PatternEvent> anomalyDetection = performAnomalyDetection(keyedSemanticEvents);

        // Clinical protocol monitoring
        DataStream<PatternEvent> protocolMonitoring = monitorClinicalProtocols(keyedSemanticEvents);

        // NEW: Advanced Windowed Analytics from Phase 3
        DataStream<MEWSAlert> mewsAlerts = MEWSCalculator.calculateMEWS(keyedSemanticEvents);
        DataStream<LabTrendAlert> creatinineAlerts = LabTrendAnalyzer.analyzeCreatinineTrends(keyedSemanticEvents);
        DataStream<LabTrendAlert> glucoseAlerts = LabTrendAnalyzer.analyzeGlucoseTrends(keyedSemanticEvents);
        DataStream<VitalVariabilityAlert> vitalVariabilityAlerts = VitalVariabilityAnalyzer.analyzeAllVitalVariability(keyedSemanticEvents);

        // NEW: Daily Aggregate Risk Scoring (24-hour tumbling window)
        DataStream<DailyRiskScore> dailyRiskScores = keyedSemanticEvents
            .window(TumblingEventTimeWindows.of(Duration.ofHours(24)))
            .apply(new RiskScoreCalculator.DailyRiskScoringWindowFunction())
            .name("Daily Risk Scoring")
            .uid("Daily-Risk-Scoring");

        // ===== Pattern Event Generation =====

        // Convert CEP patterns to pattern events
        DataStream<PatternEvent> deteriorationEvents = deteriorationPatterns
            .select(new DeteriorationPatternSelectFunction())
            .uid("Deterioration Pattern Events");

        // V4: Cross-domain deterioration events
        DataStream<PatternEvent> crossDomainEvents = crossDomainPatterns
            .select(new CrossDomainDeteriorationSelectFunction())
            .uid("Cross-Domain Deterioration Events");

        DataStream<PatternEvent> medicationEvents = medicationPatterns
            .select(new MedicationPatternSelectFunction())
            .uid("Medication Pattern Events");

        DataStream<PatternEvent> vitalTrendEvents = vitalTrendPatterns
            .select(new VitalTrendPatternSelectFunction())
            .uid("Vital Trend Pattern Events");

        DataStream<PatternEvent> pathwayEvents = pathwayPatterns
            .select(new PathwayCompliancePatternSelectFunction())
            .uid("Pathway Compliance Events");

        // TODO: Re-enable AKI pattern detection once risk indicators are extracted from CDS events
        // DataStream<PatternEvent> akiEvents = akiPatterns
        //     .select(new ClinicalPatterns.AKIPatternSelectFunction())
        //     .uid("AKI Pattern Events");

        // NEW: Convert advanced CEP patterns to pattern events
        DataStream<PatternEvent> sepsisEvents = sepsisPatterns
            .select(new ClinicalPatterns.SepsisPatternSelectFunction())
            .uid("Sepsis Pattern Events");

        DataStream<PatternEvent> rapidDeteriorationEvents = rapidDeteriorationPatterns
            .select(new ClinicalPatterns.RapidDeteriorationPatternSelectFunction())
            .uid("Rapid Deterioration Pattern Events");

        DataStream<PatternEvent> drugLabMonitoringEvents = drugLabMonitoringPatterns
            .select(new ClinicalPatterns.DrugLabMonitoringPatternSelectFunction())
            .uid("Drug Lab Monitoring Pattern Events");

        DataStream<PatternEvent> sepsisPathwayEvents = sepsisPathwayPatterns
            .select(new ClinicalPatterns.SepsisPathwayCompliancePatternSelectFunction())
            .uid("Sepsis Pathway Compliance Pattern Events");

        // NEW: Convert advanced analytics to pattern events
        DataStream<PatternEvent> mewsEvents = mewsAlerts
            .map(new MEWSAlertToPatternEventMapper())
            .uid("MEWS Alert Events");

        DataStream<PatternEvent> labTrendEvents = creatinineAlerts
            .union(glucoseAlerts)
            .map(new LabTrendAlertToPatternEventMapper())
            .uid("Lab Trend Alert Events");

        DataStream<PatternEvent> vitalVariabilityEvents = vitalVariabilityAlerts
            .map(new VitalVariabilityAlertToPatternEventMapper())
            .uid("Vital Variability Alert Events");

        // ===== Unified Pattern Stream =====

        // Unified pattern stream (union operation does not support uid/name)
        DataStream<PatternEvent> allPatternEvents = immediatePatternEvents  // START with immediate events
            .union(deteriorationEvents)
            .union(crossDomainEvents)
            .union(medicationEvents)
            .union(vitalTrendEvents)
            .union(pathwayEvents)
            // .union(akiEvents)  // TODO: Re-enable when AKI pattern is fixed
            .union(trendAnalysis)
            .union(anomalyDetection)
            .union(protocolMonitoring)
            .union(sepsisEvents)
            .union(rapidDeteriorationEvents)
            .union(drugLabMonitoringEvents)
            .union(sepsisPathwayEvents)
            .union(mewsEvents)
            .union(labTrendEvents)
            .union(vitalVariabilityEvents);

        // ===== DEDUPLICATION & MULTI-SOURCE CONFIRMATION (Gap 1) =====
        // Apply 5-minute deduplication window to prevent alert storms
        // Merges patterns from Layer 1 (instant state) + Layer 2 (CEP) when they fire together
        DataStream<PatternEvent> dedupedPatterns = allPatternEvents
            .keyBy(PatternEvent::getPatientId)
            .process(new PatternDeduplicationFunction())
            .uid("Pattern Deduplication")
            .name("Deduplicated Multi-Source Patterns");

        // ===== Pattern Classification and Routing =====

        SingleOutputStreamOperator<PatternEvent> classifiedPatterns = dedupedPatterns
            .map(new PatternClassificationFunction())
            .uid("Pattern Classification");

        // Route patterns to appropriate sinks
        classifiedPatterns
            .sinkTo(createPatternEventsSink())
            .uid("Pattern Events Sink");

        // Route specific pattern types to specialized topics
        classifiedPatterns.getSideOutput(DETERIORATION_PATTERN_TAG)
            .sinkTo(createDeteriorationPatternSink())
            .uid("Deterioration Patterns Sink");

        classifiedPatterns.getSideOutput(PATHWAY_ADHERENCE_TAG)
            .sinkTo(createPathwayAdherenceSink())
            .uid("Pathway Adherence Sink");

        classifiedPatterns.getSideOutput(ANOMALY_DETECTION_TAG)
            .sinkTo(createAnomalyDetectionSink())
            .uid("Anomaly Detection Sink");

        classifiedPatterns.getSideOutput(TREND_ANALYSIS_TAG)
            .sinkTo(createTrendAnalysisSink())
            .uid("Trend Analysis Sink");

        // Route daily risk scores to dedicated topic
        dailyRiskScores
            .sinkTo(createDailyRiskScoreSink())
            .uid("Daily Risk Score Sink");

        LOG.info("Pattern detection pipeline created successfully (includes daily risk scoring)");
    }

    /**
     * Create semantic event source from Module 3
     */
    private static DataStream<SemanticEvent> createSemanticEventSource(StreamExecutionEnvironment env) {
        KafkaSource<SemanticEvent> source = KafkaSource.<SemanticEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(getTopicName("MODULE4_SEMANTIC_INPUT_TOPIC", "semantic-mesh-updates.v1"))
            .setGroupId("pattern-detection")
            .setStartingOffsets(OffsetsInitializer.timestamp(System.currentTimeMillis()))
            .setValueOnlyDeserializer(new SemanticEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("pattern-detection"))
            .build();

        return env.fromSource(source,
            WatermarkStrategy
                .<SemanticEvent>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                .withTimestampAssigner((event, timestamp) -> event.getEventTime()),
            "Semantic Events Source");
    }

    /**
     * Create enriched event source from Module 2 for RiskIndicators-based patterns
     * DEPRECATED: Use createCDSEventSource() instead to consume from Module 3
     */
    private static DataStream<EnrichedEvent> createEnrichedEventSource(StreamExecutionEnvironment env) {
        KafkaSource<EnrichedEvent> source = KafkaSource.<EnrichedEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(getTopicName("MODULE4_ENRICHED_INPUT_TOPIC", "clinical-patterns.v1"))
            .setGroupId("pattern-detection-enriched")
            .setStartingOffsets(OffsetsInitializer.timestamp(System.currentTimeMillis()))
            .setValueOnlyDeserializer(new EnrichedEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("pattern-detection-enriched"))
            .build();

        return env.fromSource(source,
            WatermarkStrategy
                .<EnrichedEvent>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                .withTimestampAssigner((event, timestamp) -> event.getEventTime()),
            "Enriched Events Source");
    }

    /**
     * Create CDS event source from Module 3 comprehensive CDS output
     * This is the PRIMARY input for Module 4 - contains full clinical intelligence
     */
    private static DataStream<Module3_ComprehensiveCDS.CDSEvent> createCDSEventSource(StreamExecutionEnvironment env) {
        KafkaSource<Module3_ComprehensiveCDS.CDSEvent> source = KafkaSource.<Module3_ComprehensiveCDS.CDSEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(getTopicName("MODULE4_CDS_INPUT_TOPIC", "comprehensive-cds-events.v1"))
            .setGroupId("pattern-detection-cds")
            .setStartingOffsets(OffsetsInitializer.latest())  // Process only new events
            .setValueOnlyDeserializer(new CDSEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("pattern-detection-cds"))
            .build();

        return env.fromSource(source,
            WatermarkStrategy
                .<Module3_ComprehensiveCDS.CDSEvent>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                .withTimestampAssigner((event, timestamp) -> event.getEventTime()),
            "CDS Events Source");
    }

    /**
     * Convert CDSEvent from Module 3 to SemanticEvent for pattern detection
     * Extracts semantic enrichment and patient context from CDS analysis
     */
    private static SemanticEvent convertCDSEventToSemanticEvent(Module3_ComprehensiveCDS.CDSEvent cdsEvent) {
        SemanticEvent semanticEvent = new SemanticEvent();

        // Basic event info
        semanticEvent.setPatientId(cdsEvent.getPatientId());
        semanticEvent.setEventTime(cdsEvent.getEventTime());
        semanticEvent.setProcessingTime(cdsEvent.getProcessingTime());

        // Convert String eventType to EventType enum
        try {
            semanticEvent.setEventType(EventType.valueOf(cdsEvent.getEventType()));
        } catch (Exception e) {
            semanticEvent.setEventType(EventType.UNKNOWN);  // Fallback
        }

        // Note: PatientContextState doesn't have getPatientContext() method
        // The patient context data is embedded in PatientContextState itself
        // For now, we'll use the data from semanticAnnotations
        // TODO: Create proper PatientContext from PatientContextState if needed

        // Store semantic enrichment in semanticAnnotations
        if (cdsEvent.getSemanticEnrichment() != null) {
            Map<String, Object> annotations = new HashMap<>();
            com.cardiofit.flink.models.SemanticEnrichment enrichment = cdsEvent.getSemanticEnrichment();

            // Add semantic enrichment data as annotations
            annotations.put("matchedProtocols", enrichment.getMatchedProtocols());
            annotations.put("clinicalThresholds", enrichment.getClinicalThresholds());
            annotations.put("cepPatternFlags", enrichment.getCepPatternFlags());
            annotations.put("semanticTags", enrichment.getSemanticTags());
            annotations.put("knowledgeBaseSources", enrichment.getKnowledgeBaseSources());

            // CRITICAL FIX: Calculate clinical_significance and risk_level for CEP pattern matching
            PatientContextState patientState = cdsEvent.getPatientState();
            LOG.debug("🔍 DEBUG - Processing patient {}: patientState is {}",
                cdsEvent.getPatientId(), (patientState == null ? "NULL" : "NOT NULL"));

            if (patientState != null) {
                // Calculate clinical significance from NEWS2/qSOFA scores and combined acuity
                double clinicalSignificance = calculateClinicalSignificance(
                    patientState.getNews2Score(),
                    patientState.getQsofaScore(),
                    patientState.getCombinedAcuityScore()
                );
                annotations.put("clinical_significance", clinicalSignificance);

                // Determine risk level based on scores and alerts
                String riskLevel = determineRiskLevel(
                    patientState.getNews2Score(),
                    patientState.getQsofaScore(),
                    patientState.getAllAlerts()
                );
                annotations.put("risk_level", riskLevel);

                LOG.debug("✅ CEP DATA - Patient {}: clinical_significance={}, risk_level={} (NEWS2={}, qSOFA={}, acuity={})",
                    cdsEvent.getPatientId(), clinicalSignificance, riskLevel,
                    patientState.getNews2Score(), patientState.getQsofaScore(),
                    patientState.getCombinedAcuityScore());
            } else {
                LOG.warn("⚠️ MISSING DATA - Patient {}: patientState is NULL, cannot calculate clinical_significance/risk_level",
                    cdsEvent.getPatientId());
            }

            semanticEvent.setSemanticAnnotations(annotations);
        }

        // Add CDS recommendations as enrichment data
        if (cdsEvent.getCdsRecommendations() != null && !cdsEvent.getCdsRecommendations().isEmpty()) {
            semanticEvent.setEnrichmentData(cdsEvent.getCdsRecommendations());
        }

        // CRITICAL FIX: Extract clinical data from PatientContextState for CEP pattern matching
        PatientContextState patientState = cdsEvent.getPatientState();
        if (patientState != null) {
            Map<String, Object> clinicalData = new HashMap<>();

            // Extract vital signs (lowercase keys as expected by sepsis pattern)
            Map<String, Object> latestVitals = patientState.getLatestVitals();
            if (latestVitals != null && !latestVitals.isEmpty()) {
                Map<String, Object> vitalSigns = new HashMap<>();

                // Convert lowercase keys to snake_case for pattern matching
                // Module 2 stores as: heartrate, systolicbp, diastolicbp, respiratoryrate, oxygensaturation, temperature
                if (latestVitals.get("heartrate") != null) {
                    vitalSigns.put("heart_rate", latestVitals.get("heartrate"));
                }
                if (latestVitals.get("systolicbp") != null) {
                    vitalSigns.put("systolic_bp", latestVitals.get("systolicbp"));
                }
                if (latestVitals.get("diastolicbp") != null) {
                    vitalSigns.put("diastolic_bp", latestVitals.get("diastolicbp"));
                }
                if (latestVitals.get("respiratoryrate") != null) {
                    vitalSigns.put("respiratory_rate", latestVitals.get("respiratoryrate"));
                }
                if (latestVitals.get("temperature") != null) {
                    vitalSigns.put("temperature", latestVitals.get("temperature"));
                }
                if (latestVitals.get("oxygensaturation") != null) {
                    vitalSigns.put("oxygen_saturation", latestVitals.get("oxygensaturation"));
                }

                clinicalData.put("vitalSigns", vitalSigns);
            }

            // Extract lab values (keyed by LOINC code and standardized names)
            Map<String, com.cardiofit.flink.models.LabResult> recentLabs = patientState.getRecentLabs();
            if (recentLabs != null && !recentLabs.isEmpty()) {
                Map<String, Object> labValues = new HashMap<>();

                for (Map.Entry<String, com.cardiofit.flink.models.LabResult> entry : recentLabs.entrySet()) {
                    com.cardiofit.flink.models.LabResult lab = entry.getValue();
                    if (lab != null && lab.getValue() != null) {
                        String loincCode = entry.getKey();
                        Double value = lab.getValue();

                        // Store by LOINC code
                        labValues.put(loincCode, value);

                        // Map LOINC codes to standardized names expected by CEP patterns
                        // These match the keys used in ClinicalPatterns.java
                        switch (loincCode) {
                            case "2524-7":  // Lactate
                                labValues.put("lactate", value);
                                break;
                            case "6690-2":  // WBC
                                labValues.put("wbc_count", value.intValue());
                                break;
                            case "33959-8": // Procalcitonin
                                labValues.put("procalcitonin", value);
                                break;
                            case "2160-0":  // Creatinine
                                labValues.put("creatinine", value);
                                break;
                            case "777-3":   // Platelets
                                labValues.put("platelet_count", value.intValue());
                                break;
                        }

                        // Also store by labType if available
                        String labType = lab.getLabType();
                        if (labType != null) {
                            labValues.put(labType.toLowerCase(), value);
                        }
                    }
                }

                clinicalData.put("labValues", labValues);
            }

            // Extract risk indicators for additional context
            com.cardiofit.flink.models.RiskIndicators riskIndicators = patientState.getRiskIndicators();
            if (riskIndicators != null) {
                Map<String, Object> riskData = new HashMap<>();
                riskData.put("tachycardia", riskIndicators.isTachycardia());
                riskData.put("hypotension", riskIndicators.isHypotension());
                riskData.put("fever", riskIndicators.isFever());
                riskData.put("hypoxia", riskIndicators.isHypoxia());
                riskData.put("tachypnea", riskIndicators.isTachypnea());
                riskData.put("elevatedLactate", riskIndicators.isElevatedLactate());
                riskData.put("severelyElevatedLactate", riskIndicators.isSeverelyElevatedLactate());
                riskData.put("leukocytosis", riskIndicators.isLeukocytosis());
                riskData.put("sepsisRisk", riskIndicators.getSepsisRisk());

                clinicalData.put("riskIndicators", riskData);
            }

            // Set clinical data on semantic event
            semanticEvent.setClinicalData(clinicalData);

            // DEBUG: Log clinical data extraction with details
            if (clinicalData.containsKey("vitalSigns")) {
                Map<String, Object> vitalSigns = (Map<String, Object>) clinicalData.get("vitalSigns");
                LOG.debug("DEBUG - Patient {}: vitalSigns keys={}, labValues={} entries",
                    cdsEvent.getPatientId(),
                    vitalSigns.keySet(),
                    clinicalData.containsKey("labValues") ? ((Map)clinicalData.get("labValues")).size() : 0);
            } else {
                LOG.debug("DEBUG - Patient {}: NO vitalSigns extracted! latestVitals from state was null/empty",
                    cdsEvent.getPatientId());
            }
        }

        return semanticEvent;
    }

    /**
     * Calculate clinical significance score (0.0-1.0) from NEWS2, qSOFA, and acuity scores
     * Maps clinical scores to CEP pattern thresholds:
     * - 0.0-0.3: Low significance (baseline)
     * - 0.3-0.6: Moderate significance (warning)
     * - 0.6-0.8: High significance (early deterioration)
     * - 0.8-1.0: Critical significance (severe deterioration)
     */
    private static double calculateClinicalSignificance(int news2Score, int qsofaScore, double acuityScore) {
        return Module4ClinicalScoring.calculateClinicalSignificance(news2Score, qsofaScore, acuityScore);
    }

    /**
     * Determine risk level category based on clinical scores and alerts
     * Returns: "low", "moderate", "high", or "unknown"
     *
     * CEP Pattern Mapping:
     * - "low": NEWS2 0-4, qSOFA 0, minimal alerts → baseline candidate
     * - "moderate": NEWS2 5-9, qSOFA 1, some HIGH alerts → baseline/warning candidate
     * - "high": NEWS2 ≥10, qSOFA ≥2, multiple CRITICAL alerts → critical event
     */
    private static String determineRiskLevel(int news2Score, int qsofaScore, Set<SimpleAlert> alerts) {
        return Module4ClinicalScoring.determineRiskLevel(news2Score, qsofaScore, alerts);
    }

    // ===== CEP Pattern Definitions =====

    /**
     * Detect clinical deterioration patterns
     */
    private static PatternStream<SemanticEvent> detectDeteriorationPatterns(DataStream<SemanticEvent> input) {
        Pattern<SemanticEvent, ?> deteriorationPattern = Pattern
            .<SemanticEvent>begin("baseline")
            .where(new SimpleCondition<SemanticEvent>() {
                @Override
                public boolean filter(SemanticEvent event) {
                    boolean matches = event.getClinicalSignificance() > 0.3 &&
                                     !event.getRiskLevel().equals("high");
                    LOG.debug("🔍 CEP BASELINE CHECK - Patient {}: sig={}, risk={}, matches={}",
                        event.getPatientId(), event.getClinicalSignificance(),
                        event.getRiskLevel(), matches);
                    return matches;
                }
            })
            .followedBy("warning")
            .where(new SimpleCondition<SemanticEvent>() {
                @Override
                public boolean filter(SemanticEvent event) {
                    boolean matches = event.getClinicalSignificance() > 0.6 ||
                                     event.getRiskLevel().equals("moderate");
                    LOG.debug("🔍 CEP WARNING CHECK - Patient {}: sig={}, risk={}, matches={}",
                        event.getPatientId(), event.getClinicalSignificance(),
                        event.getRiskLevel(), matches);
                    return matches;
                }
            })
            .followedBy("critical")
            .where(new SimpleCondition<SemanticEvent>() {
                @Override
                public boolean filter(SemanticEvent event) {
                    boolean matches = event.getClinicalSignificance() > 0.8 ||
                                     event.getRiskLevel().equals("high") ||
                                     event.hasClinicalAlerts();
                    LOG.debug("🔍 CEP CRITICAL CHECK - Patient {}: sig={}, risk={}, alerts={}, matches={}",
                        event.getPatientId(), event.getClinicalSignificance(),
                        event.getRiskLevel(), event.hasClinicalAlerts(), matches);
                    return matches;
                }
            })
            .within(Duration.ofHours(6)); // Pattern must occur within 6 hours

        return CEP.pattern(input, deteriorationPattern);
    }

    /**
     * V4: Detect cross-domain deterioration — glycaemic AND hemodynamic concurrent decline.
     * Fires when both domains show declining trajectory within 72h window.
     */
    private static PatternStream<SemanticEvent> detectCrossDomainDeteriorationPatterns(
            DataStream<SemanticEvent> input) {
        Pattern<SemanticEvent, ?> crossDomainDecline = Pattern
            .<SemanticEvent>begin("glycaemic_decline")
            .where(new SimpleCondition<SemanticEvent>() {
                @Override
                public boolean filter(SemanticEvent event) {
                    return "GLYCAEMIC".equals(event.getClinicalDomain())
                        && ("DECLINING".equals(event.getTrajectoryClass())
                            || "RAPID_RISING".equals(event.getTrajectoryClass()));
                }
            })
            .followedBy("hemodynamic_decline")
            .where(new SimpleCondition<SemanticEvent>() {
                @Override
                public boolean filter(SemanticEvent event) {
                    return "HEMODYNAMIC".equals(event.getClinicalDomain())
                        && ("DECLINING".equals(event.getTrajectoryClass())
                            || "RAPID_RISING".equals(event.getTrajectoryClass()));
                }
            })
            .within(Duration.ofHours(72));

        return CEP.pattern(input, crossDomainDecline);
    }

    /**
     * Detect medication adherence patterns
     */
    private static PatternStream<SemanticEvent> detectMedicationPatterns(DataStream<SemanticEvent> input) {
        Pattern<SemanticEvent, ?> medicationPattern = Pattern
            .<SemanticEvent>begin("medication_ordered")
            .where(new SimpleCondition<SemanticEvent>() {
                @Override
                public boolean filter(SemanticEvent event) {
                    return event.getEventType() == EventType.MEDICATION_ORDERED;
                }
            })
            .followedBy("administration_due")
            .where(new SimpleCondition<SemanticEvent>() {
                @Override
                public boolean filter(SemanticEvent event) {
                    return event.getEventType() == EventType.MEDICATION_ADMINISTERED ||
                           event.getEventType() == EventType.MEDICATION_MISSED;
                }
            })
            .within(Duration.ofHours(2)); // Medication should be administered within 2 hours

        return CEP.pattern(input, medicationPattern);
    }

    /**
     * Detect vital signs trend patterns
     */
    private static PatternStream<SemanticEvent> detectVitalTrendPatterns(DataStream<SemanticEvent> input) {
        Pattern<SemanticEvent, ?> vitalTrendPattern = Pattern
            .<SemanticEvent>begin("vital1")
            .where(new SimpleCondition<SemanticEvent>() {
                @Override
                public boolean filter(SemanticEvent event) {
                    return event.getEventType() == EventType.VITAL_SIGN;
                }
            })
            .next("vital2")
            .where(new SimpleCondition<SemanticEvent>() {
                @Override
                public boolean filter(SemanticEvent event) {
                    return event.getEventType() == EventType.VITAL_SIGN;
                }
            })
            .next("vital3")
            .where(new SimpleCondition<SemanticEvent>() {
                @Override
                public boolean filter(SemanticEvent event) {
                    return event.getEventType() == EventType.VITAL_SIGN;
                }
            })
            .within(Duration.ofHours(1)); // Three consecutive vital sign readings within 1 hour

        return CEP.pattern(input, vitalTrendPattern);
    }

    /**
     * Detect clinical pathway compliance patterns
     */
    private static PatternStream<SemanticEvent> detectPathwayCompliancePatterns(DataStream<SemanticEvent> input) {
        Pattern<SemanticEvent, ?> pathwayPattern = Pattern
            .<SemanticEvent>begin("admission")
            .where(new SimpleCondition<SemanticEvent>() {
                @Override
                public boolean filter(SemanticEvent event) {
                    return event.getEventType() == EventType.PATIENT_ADMISSION;
                }
            })
            .followedBy("assessment")
            .where(new SimpleCondition<SemanticEvent>() {
                @Override
                public boolean filter(SemanticEvent event) {
                    return event.getEventType() == EventType.VITAL_SIGN ||
                           event.getEventType() == EventType.LAB_RESULT;
                }
            })
            .followedBy("intervention")
            .where(new SimpleCondition<SemanticEvent>() {
                @Override
                public boolean filter(SemanticEvent event) {
                    return event.getEventType() == EventType.MEDICATION_ORDERED ||
                           event.getEventType() == EventType.PROCEDURE_SCHEDULED;
                }
            })
            .within(Duration.ofHours(24)); // Pathway should be completed within 24 hours

        return CEP.pattern(input, pathwayPattern);
    }

    // ===== Windowed Analytics Functions =====

    /**
     * Perform trend analysis using sliding windows
     */
    private static DataStream<PatternEvent> performTrendAnalysis(DataStream<SemanticEvent> input) {
        return input
            .filter(event -> event.getEventType() == EventType.VITAL_SIGN ||
                           event.getEventType() == EventType.LAB_RESULT)
            .keyBy(SemanticEvent::getPatientId)
            .window(SlidingEventTimeWindows.of(Duration.ofHours(4), Duration.ofHours(1)))
            .apply(new TrendAnalysisWindowFunction())
            .uid("Trend Analysis");
    }

    /**
     * Perform anomaly detection using tumbling windows
     */
    private static DataStream<PatternEvent> performAnomalyDetection(DataStream<SemanticEvent> input) {
        return input
            .keyBy(SemanticEvent::getPatientId)
            .window(TumblingEventTimeWindows.of(Duration.ofMinutes(30)))
            .process(new AnomalyDetectionWindowFunction())
            .uid("Anomaly Detection");
    }

    /**
     * Monitor clinical protocols
     */
    private static DataStream<PatternEvent> monitorClinicalProtocols(DataStream<SemanticEvent> input) {
        return input
            .filter(event -> event.hasGuidelineRecommendations())
            .keyBy(SemanticEvent::getPatientId)
            .window(TumblingEventTimeWindows.of(Duration.ofHours(2)))
            .apply(new ProtocolMonitoringWindowFunction())
            .uid("Protocol Monitoring");
    }

    // ===== Pattern Select Functions =====

    public static class DeteriorationPatternSelectFunction implements PatternSelectFunction<SemanticEvent, PatternEvent> {
        @Override
        public PatternEvent select(Map<String, List<SemanticEvent>> pattern) throws Exception {
            List<SemanticEvent> baseline = pattern.get("baseline");
            List<SemanticEvent> warning = pattern.get("warning");
            List<SemanticEvent> critical = pattern.get("critical");

            PatternEvent patternEvent = new PatternEvent();
            patternEvent.setId(UUID.randomUUID().toString());
            patternEvent.setPatternType("CLINICAL_DETERIORATION");
            patternEvent.setPatientId(baseline.get(0).getPatientId());
            patternEvent.setDetectionTime(System.currentTimeMillis());
            patternEvent.setSeverity("HIGH");
            patternEvent.setConfidence(0.85);

            // Calculate pattern details
            Map<String, Object> patternDetails = new HashMap<>();
            patternDetails.put("baseline_significance", baseline.get(0).getClinicalSignificance());
            patternDetails.put("warning_significance", warning.get(0).getClinicalSignificance());
            patternDetails.put("critical_significance", critical.get(0).getClinicalSignificance());
            patternDetails.put("deterioration_rate", calculateDeteriorationRate(baseline, warning, critical));
            patternDetails.put("timespan_hours", calculateTimespan(baseline.get(0), critical.get(0)));

            patternEvent.setPatternDetails(patternDetails);

            List<String> involvedEvents = new ArrayList<>();
            involvedEvents.add(baseline.get(0).getId());
            involvedEvents.add(warning.get(0).getId());
            involvedEvents.add(critical.get(0).getId());
            patternEvent.setInvolvedEvents(involvedEvents);

            patternEvent.setRecommendedActions(Arrays.asList(
                "IMMEDIATE_ASSESSMENT_REQUIRED",
                "ESCALATE_TO_PHYSICIAN",
                "INCREASE_MONITORING_FREQUENCY"
            ));

            LOG.info("Detected clinical deterioration pattern for patient: {}", patternEvent.getPatientId());
            return patternEvent;
        }

        private double calculateDeteriorationRate(List<SemanticEvent> baseline,
                                                  List<SemanticEvent> warning,
                                                  List<SemanticEvent> critical) {
            double baselineScore = baseline.get(0).getClinicalSignificance();
            double criticalScore = critical.get(0).getClinicalSignificance();
            return criticalScore - baselineScore;
        }

        private double calculateTimespan(SemanticEvent first, SemanticEvent last) {
            return (last.getEventTime() - first.getEventTime()) / (1000.0 * 3600.0);
        }
    }

    /**
     * V4: Select function for cross-domain deterioration patterns.
     * Fires when glycaemic and hemodynamic domains both show declining/rapid-rising trajectory.
     */
    public static class CrossDomainDeteriorationSelectFunction implements PatternSelectFunction<SemanticEvent, PatternEvent> {
        @Override
        public PatternEvent select(Map<String, List<SemanticEvent>> pattern) throws Exception {
            List<SemanticEvent> glycaemic = pattern.get("glycaemic_decline");
            List<SemanticEvent> hemodynamic = pattern.get("hemodynamic_decline");

            PatternEvent patternEvent = new PatternEvent();
            patternEvent.setId(UUID.randomUUID().toString());
            patternEvent.setPatternType("CROSS_DOMAIN_DECLINE");
            patternEvent.setPatientId(glycaemic.get(0).getPatientId());
            patternEvent.setDetectionTime(System.currentTimeMillis());
            patternEvent.setSeverity("HIGH");
            patternEvent.setConfidence(0.80);

            Map<String, Object> patternDetails = new HashMap<>();
            patternDetails.put("glycaemic_trajectory", glycaemic.get(0).getTrajectoryClass());
            patternDetails.put("hemodynamic_trajectory", hemodynamic.get(0).getTrajectoryClass());
            patternDetails.put("glycaemic_domain", glycaemic.get(0).getClinicalDomain());
            patternDetails.put("hemodynamic_domain", hemodynamic.get(0).getClinicalDomain());
            patternDetails.put("timespan_hours",
                (hemodynamic.get(0).getEventTime() - glycaemic.get(0).getEventTime()) / (1000.0 * 3600.0));
            patternEvent.setPatternDetails(patternDetails);

            List<String> involvedEvents = new ArrayList<>();
            involvedEvents.add(glycaemic.get(0).getId());
            involvedEvents.add(hemodynamic.get(0).getId());
            patternEvent.setInvolvedEvents(involvedEvents);

            patternEvent.setRecommendedActions(Arrays.asList(
                "CROSS_DOMAIN_ASSESSMENT_REQUIRED",
                "ESCALATE_TO_PHYSICIAN",
                "MHRI_RECOMPUTATION_NEEDED"
            ));

            LOG.info("V4: Detected cross-domain deterioration for patient {}: glycaemic={}, hemodynamic={}",
                patternEvent.getPatientId(),
                glycaemic.get(0).getTrajectoryClass(),
                hemodynamic.get(0).getTrajectoryClass());
            return patternEvent;
        }
    }

    public static class MedicationPatternSelectFunction implements PatternSelectFunction<SemanticEvent, PatternEvent> {
        @Override
        public PatternEvent select(Map<String, List<SemanticEvent>> pattern) throws Exception {
            List<SemanticEvent> ordered = pattern.get("medication_ordered");
            List<SemanticEvent> administered = pattern.get("administration_due");

            PatternEvent patternEvent = new PatternEvent();
            patternEvent.setId(UUID.randomUUID().toString());
            patternEvent.setPatternType("MEDICATION_ADHERENCE");
            patternEvent.setPatientId(ordered.get(0).getPatientId());
            patternEvent.setDetectionTime(System.currentTimeMillis());

            boolean isMissed = administered.get(0).getEventType() == EventType.MEDICATION_MISSED;
            patternEvent.setSeverity(isMissed ? "MODERATE" : "LOW");
            patternEvent.setConfidence(0.9);

            Map<String, Object> patternDetails = new HashMap<>();
            patternDetails.put("medication_status", isMissed ? "MISSED" : "ADMINISTERED");
            patternDetails.put("time_to_administration",
                (administered.get(0).getEventTime() - ordered.get(0).getEventTime()) / (1000.0 * 60.0));

            patternEvent.setPatternDetails(patternDetails);

            if (isMissed) {
                patternEvent.setRecommendedActions(Arrays.asList(
                    "VERIFY_PATIENT_STATUS",
                    "RESCHEDULE_MEDICATION",
                    "ASSESS_ADHERENCE_BARRIERS"
                ));
            }

            return patternEvent;
        }
    }

    public static class VitalTrendPatternSelectFunction implements PatternSelectFunction<SemanticEvent, PatternEvent> {
        @Override
        public PatternEvent select(Map<String, List<SemanticEvent>> pattern) throws Exception {
            List<SemanticEvent> vital1 = pattern.get("vital1");
            List<SemanticEvent> vital2 = pattern.get("vital2");
            List<SemanticEvent> vital3 = pattern.get("vital3");

            PatternEvent patternEvent = new PatternEvent();
            patternEvent.setId(UUID.randomUUID().toString());
            patternEvent.setPatternType("VITAL_SIGNS_TREND");
            patternEvent.setPatientId(vital1.get(0).getPatientId());
            patternEvent.setDetectionTime(System.currentTimeMillis());

            // Analyze trend direction
            String trendDirection = analyzeTrendDirection(vital1.get(0), vital2.get(0), vital3.get(0));
            patternEvent.setSeverity(trendDirection.equals("DETERIORATING") ? "HIGH" : "LOW");
            patternEvent.setConfidence(0.75);

            Map<String, Object> patternDetails = new HashMap<>();
            patternDetails.put("trend_direction", trendDirection);
            patternDetails.put("reading_count", 3);
            patternDetails.put("timespan_minutes",
                (vital3.get(0).getEventTime() - vital1.get(0).getEventTime()) / (1000.0 * 60.0));

            patternEvent.setPatternDetails(patternDetails);

            if (trendDirection.equals("DETERIORATING")) {
                patternEvent.setRecommendedActions(Arrays.asList(
                    "INCREASE_VITAL_MONITORING",
                    "CLINICAL_ASSESSMENT",
                    "CONSIDER_INTERVENTION"
                ));
            }

            return patternEvent;
        }

        private String analyzeTrendDirection(SemanticEvent v1, SemanticEvent v2, SemanticEvent v3) {
            double s1 = v1.getClinicalSignificance();
            double s2 = v2.getClinicalSignificance();
            double s3 = v3.getClinicalSignificance();

            if (s3 > s2 && s2 > s1 && (s3 - s1) > 0.2) {
                return "DETERIORATING";
            } else if (s1 > s2 && s2 > s3 && (s1 - s3) > 0.2) {
                return "IMPROVING";
            } else {
                return "STABLE";
            }
        }
    }

    public static class PathwayCompliancePatternSelectFunction implements PatternSelectFunction<SemanticEvent, PatternEvent> {
        @Override
        public PatternEvent select(Map<String, List<SemanticEvent>> pattern) throws Exception {
            List<SemanticEvent> admission = pattern.get("admission");
            List<SemanticEvent> assessment = pattern.get("assessment");
            List<SemanticEvent> intervention = pattern.get("intervention");

            PatternEvent patternEvent = new PatternEvent();
            patternEvent.setId(UUID.randomUUID().toString());
            patternEvent.setPatternType("PATHWAY_COMPLIANCE");
            patternEvent.setPatientId(admission.get(0).getPatientId());
            patternEvent.setDetectionTime(System.currentTimeMillis());
            patternEvent.setSeverity("LOW");
            patternEvent.setConfidence(0.8);

            Map<String, Object> patternDetails = new HashMap<>();
            patternDetails.put("pathway_completed", true);
            patternDetails.put("time_to_assessment",
                (assessment.get(0).getEventTime() - admission.get(0).getEventTime()) / (1000.0 * 60.0));
            patternDetails.put("time_to_intervention",
                (intervention.get(0).getEventTime() - admission.get(0).getEventTime()) / (1000.0 * 60.0));

            patternEvent.setPatternDetails(patternDetails);

            return patternEvent;
        }
    }

    // ===== Window Functions =====

    public static class TrendAnalysisWindowFunction
            implements WindowFunction<SemanticEvent, PatternEvent, String, TimeWindow> {

        @Override
        public void apply(String patientId, TimeWindow window,
                         Iterable<SemanticEvent> input, Collector<PatternEvent> out) {

            List<SemanticEvent> events = new ArrayList<>();
            for (SemanticEvent event : input) {
                events.add(event);
            }

            if (events.size() < 3) return; // Need at least 3 points for trend analysis

            // Sort by event time
            events.sort((a, b) -> Long.compare(a.getEventTime(), b.getEventTime()));

            PatternEvent trendEvent = new PatternEvent();
            trendEvent.setId(UUID.randomUUID().toString());
            trendEvent.setPatternType("TREND_ANALYSIS");
            trendEvent.setPatientId(patientId);
            trendEvent.setDetectionTime(System.currentTimeMillis());

            // Calculate trend
            TrendAnalysis trend = calculateTrend(events);
            trendEvent.setSeverity(trend.getSeverity());
            trendEvent.setConfidence(trend.getConfidence());

            Map<String, Object> patternDetails = new HashMap<>();
            patternDetails.put("trend_slope", trend.getSlope());
            patternDetails.put("trend_direction", trend.getDirection());
            patternDetails.put("data_points", events.size());
            patternDetails.put("window_start", window.getStart());
            patternDetails.put("window_end", window.getEnd());

            trendEvent.setPatternDetails(patternDetails);

            if (!trend.getDirection().equals("STABLE")) {
                out.collect(trendEvent);
            }
        }

        private TrendAnalysis calculateTrend(List<SemanticEvent> events) {
            // Simple linear regression for trend analysis
            double n = events.size();
            double sumX = 0, sumY = 0, sumXY = 0, sumX2 = 0;

            for (int i = 0; i < events.size(); i++) {
                double x = i; // Time index
                double y = events.get(i).getClinicalSignificance();

                sumX += x;
                sumY += y;
                sumXY += x * y;
                sumX2 += x * x;
            }

            double slope = (n * sumXY - sumX * sumY) / (n * sumX2 - sumX * sumX);

            TrendAnalysis trend = new TrendAnalysis();
            trend.setSlope(slope);

            if (slope > 0.1) {
                trend.setDirection("INCREASING");
                trend.setSeverity("MODERATE");
            } else if (slope < -0.1) {
                trend.setDirection("DECREASING");
                trend.setSeverity("LOW");
            } else {
                trend.setDirection("STABLE");
                trend.setSeverity("LOW");
            }

            trend.setConfidence(Math.min(0.9, 0.5 + Math.abs(slope)));

            return trend;
        }
    }

    public static class AnomalyDetectionWindowFunction
            extends ProcessWindowFunction<SemanticEvent, PatternEvent, String, TimeWindow> {

        @Override
        public void process(String patientId, Context context,
                           Iterable<SemanticEvent> elements, Collector<PatternEvent> out) {

            List<SemanticEvent> events = new ArrayList<>();
            for (SemanticEvent event : elements) {
                events.add(event);
            }

            if (events.size() < 5) return; // Need sufficient data for anomaly detection

            // Calculate statistical baseline
            double mean = events.stream()
                .mapToDouble(SemanticEvent::getClinicalSignificance)
                .average()
                .orElse(0.0);

            double variance = events.stream()
                .mapToDouble(e -> Math.pow(e.getClinicalSignificance() - mean, 2))
                .average()
                .orElse(0.0);

            double stdDev = Math.sqrt(variance);
            double threshold = mean + 2 * stdDev; // 2 standard deviations

            // Find anomalies
            List<SemanticEvent> anomalies = events.stream()
                .filter(e -> e.getClinicalSignificance() > threshold)
                .collect(Collectors.toList());

            if (!anomalies.isEmpty()) {
                PatternEvent anomalyEvent = new PatternEvent();
                anomalyEvent.setId(UUID.randomUUID().toString());
                anomalyEvent.setPatternType("ANOMALY_DETECTION");
                anomalyEvent.setPatientId(patientId);
                anomalyEvent.setDetectionTime(System.currentTimeMillis());
                anomalyEvent.setSeverity("MODERATE");
                anomalyEvent.setConfidence(0.8);

                Map<String, Object> patternDetails = new HashMap<>();
                patternDetails.put("anomaly_count", anomalies.size());
                patternDetails.put("statistical_threshold", threshold);
                patternDetails.put("baseline_mean", mean);
                patternDetails.put("baseline_stddev", stdDev);

                anomalyEvent.setPatternDetails(patternDetails);
                anomalyEvent.setInvolvedEvents(
                    anomalies.stream().map(SemanticEvent::getId).collect(Collectors.toList())
                );

                out.collect(anomalyEvent);
            }
        }
    }

    public static class ProtocolMonitoringWindowFunction
            implements WindowFunction<SemanticEvent, PatternEvent, String, TimeWindow> {

        @Override
        public void apply(String patientId, TimeWindow window,
                         Iterable<SemanticEvent> input, Collector<PatternEvent> out) {

            List<SemanticEvent> events = new ArrayList<>();
            for (SemanticEvent event : input) {
                events.add(event);
            }

            if (events.isEmpty()) return;

            // Analyze protocol adherence
            long recommendationsCount = events.stream()
                .mapToLong(e -> e.getGuidelineRecommendations() != null ?
                               e.getGuidelineRecommendations().size() : 0)
                .sum();

            if (recommendationsCount > 0) {
                PatternEvent protocolEvent = new PatternEvent();
                protocolEvent.setId(UUID.randomUUID().toString());
                protocolEvent.setPatternType("PROTOCOL_MONITORING");
                protocolEvent.setPatientId(patientId);
                protocolEvent.setDetectionTime(System.currentTimeMillis());
                protocolEvent.setSeverity("LOW");
                protocolEvent.setConfidence(0.7);

                Map<String, Object> patternDetails = new HashMap<>();
                patternDetails.put("recommendations_count", recommendationsCount);
                patternDetails.put("events_with_recommendations", events.size());
                patternDetails.put("window_start", window.getStart());
                patternDetails.put("window_end", window.getEnd());

                protocolEvent.setPatternDetails(patternDetails);

                out.collect(protocolEvent);
            }
        }
    }

    // ===== Pattern Classification Function =====

    public static class PatternClassificationFunction implements MapFunction<PatternEvent, PatternEvent> {
        @Override
        public PatternEvent map(PatternEvent event) throws Exception {
            // Classify patterns and route to side outputs if needed
            switch (event.getPatternType()) {
                case "CLINICAL_DETERIORATION":
                    // Already handled in main stream
                    break;
                case "PATHWAY_COMPLIANCE":
                    // Mark for pathway adherence tracking
                    event.addTag("pathway_adherence");
                    break;
                case "ANOMALY_DETECTION":
                    // Mark for anomaly processing
                    event.addTag("anomaly");
                    break;
                case "TREND_ANALYSIS":
                    // Mark for trend tracking
                    event.addTag("trend");
                    break;
            }
            return event;
        }
    }

    // ===== Helper Classes =====

    public static class TrendAnalysis {
        private double slope;
        private String direction;
        private String severity;
        private double confidence;

        // Getters and setters
        public double getSlope() { return slope; }
        public void setSlope(double slope) { this.slope = slope; }

        public String getDirection() { return direction; }
        public void setDirection(String direction) { this.direction = direction; }

        public String getSeverity() { return severity; }
        public void setSeverity(String severity) { this.severity = severity; }

        public double getConfidence() { return confidence; }
        public void setConfidence(double confidence) { this.confidence = confidence; }
    }

    // ===== Sink Creation Methods =====

    private static KafkaSink<PatternEvent> createPatternEventsSink() {
        return KafkaSink.<PatternEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(getTopicName("MODULE4_PATTERN_EVENTS_TOPIC", "pattern-events.v1"))
                .setKeySerializationSchema((SerializationSchema<PatternEvent>) event -> event.getPatientId().getBytes())
                .setValueSerializationSchema(new PatternEventSerializer())
                .build())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module4-pattern-events")
            // REMOVED: .setKafkaProducerConfig() - conflicts with custom key serialization schema
            .build();
    }

    private static KafkaSink<SemanticEvent> createSemanticEventsSink() {
        return KafkaSink.<SemanticEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.SEMANTIC_MESH_UPDATES.getTopicName())
                .setKeySerializationSchema((SerializationSchema<SemanticEvent>) event -> event.getPatientId().getBytes())
                .setValueSerializationSchema(new SemanticEventSerializer())
                .build())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module4-semantic-events")
            .build();
    }

    private static KafkaSink<PatternEvent> createDeteriorationPatternSink() {
        return KafkaSink.<PatternEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(getTopicName("MODULE4_DETERIORATION_TOPIC", "alert-management.v1"))
                .setKeySerializationSchema((SerializationSchema<PatternEvent>) event -> event.getPatientId().getBytes())
                .setValueSerializationSchema(new PatternEventSerializer())
                .build())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module4-deterioration")
            // REMOVED: .setKafkaProducerConfig() - conflicts with custom key serialization schema
            .build();
    }

    private static KafkaSink<PatternEvent> createPathwayAdherenceSink() {
        return KafkaSink.<PatternEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(getTopicName("MODULE4_PATHWAY_ADHERENCE_TOPIC", "pathway-adherence-events.v1"))
                .setKeySerializationSchema((SerializationSchema<PatternEvent>) event -> event.getPatientId().getBytes())
                .setValueSerializationSchema(new PatternEventSerializer())
                .build())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module4-pathway-adherence")
            // REMOVED: .setKafkaProducerConfig() - conflicts with custom key serialization schema
            .build();
    }

    private static KafkaSink<PatternEvent> createAnomalyDetectionSink() {
        return KafkaSink.<PatternEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(getTopicName("MODULE4_ANOMALY_DETECTION_TOPIC", "safety-events.v1"))
                .setKeySerializationSchema((SerializationSchema<PatternEvent>) event -> event.getPatientId().getBytes())
                .setValueSerializationSchema(new PatternEventSerializer())
                .build())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module4-anomaly-detection")
            // REMOVED: .setKafkaProducerConfig() - conflicts with custom key serialization schema
            .build();
    }

    private static KafkaSink<PatternEvent> createTrendAnalysisSink() {
        return KafkaSink.<PatternEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(getTopicName("MODULE4_TREND_ANALYSIS_TOPIC", "clinical-reasoning-events.v1"))
                .setKeySerializationSchema((SerializationSchema<PatternEvent>) event -> event.getId().getBytes())
                .setValueSerializationSchema(new PatternEventSerializer())
                .build())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module4-trend-analysis")
            // REMOVED: .setKafkaProducerConfig() - conflicts with custom key serialization schema
            .build();
    }

    private static KafkaSink<DailyRiskScore> createDailyRiskScoreSink() {
        return KafkaSink.<DailyRiskScore>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(getTopicName("MODULE4_DAILY_RISK_SCORE_TOPIC", "daily-risk-scores.v1"))
                .setKeySerializationSchema((SerializationSchema<DailyRiskScore>) score -> score.getPatientId().getBytes())
                .setValueSerializationSchema(new DailyRiskScoreSerializer())
                .build())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module4-daily-risk-score")
            // REMOVED: .setKafkaProducerConfig() - conflicts with custom key serialization schema
            .build();
    }

    private static String getBootstrapServers() {
        String servers = System.getenv("KAFKA_BOOTSTRAP_SERVERS");
        return (servers != null && !servers.isEmpty()) ? servers : "localhost:9092";
    }

    private static String getTopicName(String envVar, String defaultTopic) {
        String topic = System.getenv(envVar);
        return (topic != null && !topic.isEmpty()) ? topic : defaultTopic;
    }

    // ===== Serialization Classes =====

    private static class SemanticEventDeserializer implements DeserializationSchema<SemanticEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public SemanticEvent deserialize(byte[] message) throws IOException {
            return objectMapper.readValue(message, SemanticEvent.class);
        }

        @Override
        public boolean isEndOfStream(SemanticEvent nextElement) { return false; }

        @Override
        public TypeInformation<SemanticEvent> getProducedType() {
            return TypeInformation.of(SemanticEvent.class);
        }
    }

    private static class PatternEventSerializer implements SerializationSchema<PatternEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(PatternEvent element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                throw new RuntimeException("Failed to serialize PatternEvent", e);
            }
        }
    }

    private static class SemanticEventSerializer implements SerializationSchema<SemanticEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
            // Use snake_case for JSON property names to match SemanticEvent expectations
            objectMapper.setPropertyNamingStrategy(com.fasterxml.jackson.databind.PropertyNamingStrategies.SNAKE_CASE);
            // CRITICAL: Only serialize fields with @JsonProperty annotations, ignore computed getters
            objectMapper.setVisibility(com.fasterxml.jackson.annotation.PropertyAccessor.ALL,
                                      com.fasterxml.jackson.annotation.JsonAutoDetect.Visibility.NONE);
            objectMapper.setVisibility(com.fasterxml.jackson.annotation.PropertyAccessor.FIELD,
                                      com.fasterxml.jackson.annotation.JsonAutoDetect.Visibility.ANY);
        }

        @Override
        public byte[] serialize(SemanticEvent element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                throw new RuntimeException("Failed to serialize SemanticEvent", e);
            }
        }
    }

    private static class DailyRiskScoreSerializer implements SerializationSchema<DailyRiskScore> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(DailyRiskScore element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                throw new RuntimeException("Failed to serialize DailyRiskScore", e);
            }
        }
    }

    private static class EnrichedEventDeserializer implements DeserializationSchema<EnrichedEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public EnrichedEvent deserialize(byte[] message) throws IOException {
            return objectMapper.readValue(message, EnrichedEvent.class);
        }

        @Override
        public boolean isEndOfStream(EnrichedEvent nextElement) { return false; }

        @Override
        public TypeInformation<EnrichedEvent> getProducedType() {
            return TypeInformation.of(EnrichedEvent.class);
        }
    }

    /**
     * Deserializer for CDSEvent from Module 3 comprehensive CDS output
     */
    private static class CDSEventDeserializer implements DeserializationSchema<Module3_ComprehensiveCDS.CDSEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
            // Configure to handle missing/empty fields gracefully
            objectMapper.configure(com.fasterxml.jackson.databind.DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
            objectMapper.configure(com.fasterxml.jackson.databind.DeserializationFeature.ACCEPT_EMPTY_STRING_AS_NULL_OBJECT, true);
            objectMapper.configure(com.fasterxml.jackson.databind.DeserializationFeature.ACCEPT_EMPTY_ARRAY_AS_NULL_OBJECT, true);
        }

        @Override
        public Module3_ComprehensiveCDS.CDSEvent deserialize(byte[] message) throws IOException {
            try {
                return objectMapper.readValue(message, Module3_ComprehensiveCDS.CDSEvent.class);
            } catch (Exception e) {
                LOG.error("Failed to deserialize CDS event: {}", e.getMessage(), e);
                LOG.error("Problematic message: {}", new String(message));
                throw e;
            }
        }

        @Override
        public boolean isEndOfStream(Module3_ComprehensiveCDS.CDSEvent nextElement) { return false; }

        @Override
        public TypeInformation<Module3_ComprehensiveCDS.CDSEvent> getProducedType() {
            return TypeInformation.of(Module3_ComprehensiveCDS.CDSEvent.class);
        }
    }

    // ===== Alert to PatternEvent Mapper Classes =====

    /**
     * Convert MEWSAlert to PatternEvent
     */
    static class MEWSAlertToPatternEventMapper implements MapFunction<MEWSAlert, PatternEvent> {
        @Override
        public PatternEvent map(MEWSAlert alert) {
            PatternEvent event = new PatternEvent();
            event.setId("mews-" + alert.getPatientId() + "-" + alert.getTimestamp());
            event.setPatientId(alert.getPatientId());
            event.setPatternType("MEWS_ALERT");
            event.setDetectionTime(alert.getTimestamp());
            event.setPatternStartTime(alert.getWindowStart());
            event.setPatternEndTime(alert.getWindowEnd());

            event.setSeverity(determineMEWSSeverity(alert.getMewsScore()));
            event.setConfidence(0.95);  // MEWS is validated scoring system

            Map<String, Object> patternDetails = new HashMap<>();
            patternDetails.put("mews_score", alert.getMewsScore());
            patternDetails.put("score_breakdown", alert.getScoreBreakdown());
            patternDetails.put("concerning_vitals", alert.getConcerningVitals());
            patternDetails.put("urgency", alert.getUrgency());
            patternDetails.put("recommendations", alert.getRecommendations());
            patternDetails.put("description", String.format("MEWS Alert: Score %d - %s",
                alert.getMewsScore(), alert.getUrgency()));
            event.setPatternDetails(patternDetails);

            return event;
        }

        private String determineMEWSSeverity(int mewsScore) {
            if (mewsScore >= 5) return "CRITICAL";
            if (mewsScore >= 3) return "HIGH";
            return "MODERATE";
        }
    }

    /**
     * Convert LabTrendAlert to PatternEvent
     */
    static class LabTrendAlertToPatternEventMapper implements MapFunction<LabTrendAlert, PatternEvent> {
        @Override
        public PatternEvent map(LabTrendAlert alert) {
            PatternEvent event = new PatternEvent();
            event.setId("lab-trend-" + alert.getPatientId() + "-" + alert.getTimestamp());
            event.setPatientId(alert.getPatientId());
            event.setPatternType("LAB_TREND_ALERT");
            event.setDetectionTime(alert.getTimestamp());
            event.setPatternStartTime(alert.getWindowStart());
            event.setPatternEndTime(alert.getWindowEnd());

            event.setSeverity(determineLabTrendSeverity(alert));
            event.setConfidence(calculateLabTrendConfidence(alert));

            Map<String, Object> patternDetails = new HashMap<>();
            patternDetails.put("lab_name", alert.getLabName());
            patternDetails.put("first_value", alert.getFirstValue());
            patternDetails.put("last_value", alert.getLastValue());
            patternDetails.put("absolute_change", alert.getAbsoluteChange());
            patternDetails.put("percent_change", alert.getPercentChange());
            patternDetails.put("trend_slope", alert.getTrendSlope());
            patternDetails.put("trend_direction", alert.getTrendDirection());

            if (alert.getAkiStage() != null) {
                patternDetails.put("aki_stage", alert.getAkiStage());
            }
            if (alert.getCoefficientOfVariation() != null) {
                patternDetails.put("coefficient_of_variation", alert.getCoefficientOfVariation());
            }
            patternDetails.put("interpretation", alert.getInterpretation());
            patternDetails.put("description", String.format("%s Trend Alert: %s",
                alert.getLabName(), alert.getInterpretation().split("\n")[0]));
            event.setPatternDetails(patternDetails);

            return event;
        }

        private String determineLabTrendSeverity(LabTrendAlert alert) {
            if (alert.getAkiStage() != null) {
                if (alert.getAkiStage().equals("AKI_STAGE_3")) return "CRITICAL";
                if (alert.getAkiStage().equals("AKI_STAGE_2")) return "HIGH";
                if (alert.getAkiStage().equals("AKI_STAGE_1")) return "MODERATE";
            }
            if (alert.getLabName().contains("Glucose")) {
                if (alert.getLastValue() < 70 || alert.getLastValue() > 300) return "CRITICAL";
                if (alert.getCoefficientOfVariation() != null && alert.getCoefficientOfVariation() > 36) {
                    return "HIGH";
                }
            }
            if (Math.abs(alert.getPercentChange()) > 50) return "HIGH";
            return "MODERATE";
        }

        private double calculateLabTrendConfidence(LabTrendAlert alert) {
            // Higher confidence for more data points and stronger trends
            double baseConfidence = 0.80;
            if (alert.getTrendSlope() != null && Math.abs(alert.getTrendSlope()) > 0.1) {
                baseConfidence += 0.10;
            }
            if (alert.getAkiStage() != null && !alert.getAkiStage().equals("NO_AKI")) {
                baseConfidence += 0.10;  // KDIGO criteria are well-validated
            }
            return Math.min(baseConfidence, 0.95);
        }
    }

    /**
     * Convert VitalVariabilityAlert to PatternEvent
     */
    static class VitalVariabilityAlertToPatternEventMapper implements MapFunction<VitalVariabilityAlert, PatternEvent> {
        @Override
        public PatternEvent map(VitalVariabilityAlert alert) {
            PatternEvent event = new PatternEvent();
            event.setId("vital-variability-" + alert.getPatientId() + "-" + alert.getTimestamp());
            event.setPatientId(alert.getPatientId());
            event.setPatternType("VITAL_VARIABILITY_ALERT");
            event.setDetectionTime(alert.getTimestamp());
            event.setPatternStartTime(alert.getWindowStart());
            event.setPatternEndTime(alert.getWindowEnd());

            event.setSeverity(alert.getVariabilityLevel());  // Already set as LOW/MODERATE/HIGH/CRITICAL
            event.setConfidence(0.85);  // CV-based analysis is statistically sound

            Map<String, Object> patternDetails = new HashMap<>();
            patternDetails.put("vital_sign_name", alert.getVitalSignName());
            patternDetails.put("mean_value", alert.getMeanValue());
            patternDetails.put("standard_deviation", alert.getStandardDeviation());
            patternDetails.put("coefficient_of_variation", alert.getCoefficientOfVariation());
            patternDetails.put("variability_level", alert.getVariabilityLevel());
            patternDetails.put("clinical_significance", alert.getClinicalSignificance());
            patternDetails.put("description", String.format("%s Variability Alert: %s (CV: %.1f%%)",
                alert.getVitalSignName(),
                alert.getVariabilityLevel(),
                alert.getCoefficientOfVariation()));
            event.setPatternDetails(patternDetails);

            return event;
        }
    }
}
