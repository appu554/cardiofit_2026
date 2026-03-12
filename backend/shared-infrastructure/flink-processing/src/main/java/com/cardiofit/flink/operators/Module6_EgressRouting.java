
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import com.cardiofit.flink.config.GoogleHealthcareConfig;
import com.cardiofit.flink.sinks.GoogleFHIRStoreSink;
import com.cardiofit.flink.sinks.ElasticsearchSink;
import com.cardiofit.flink.sinks.Neo4jGraphSink;
import com.cardiofit.flink.sinks.ClickHouseSink;
import com.cardiofit.flink.sinks.RedisCacheSink;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.functions.FilterFunction;
import org.apache.flink.api.common.functions.MapFunction;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.base.DeliveryGuarantee;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.time.Duration;
import java.util.*;

/**
 * Module 6: Egress & Multi-Sink Routing
 *
 * Responsibilities:
 * - Route processed events to appropriate downstream systems
 * - Handle multi-sink delivery with different formats
 * - Manage priority-based routing for critical events
 * - Implement fail-safe routing and dead letter queues
 * - Provide real-time and batch export capabilities
 * - Transform events for different consumer systems
 * - Monitor routing health and performance
 */
public class Module6_EgressRouting {
    private static final Logger LOG = LoggerFactory.getLogger(Module6_EgressRouting.class);

    // Output tags for different routing destinations
    private static final OutputTag<RoutedEvent> CRITICAL_ALERTS_TAG =
        new OutputTag<RoutedEvent>("critical-alerts"){};

    private static final OutputTag<RoutedEvent> CLINICAL_WORKFLOW_TAG =
        new OutputTag<RoutedEvent>("clinical-workflow"){};

    private static final OutputTag<RoutedEvent> ANALYTICS_TAG =
        new OutputTag<RoutedEvent>("analytics"){};

    private static final OutputTag<RoutedEvent> EXTERNAL_SYSTEMS_TAG =
        new OutputTag<RoutedEvent>("external-systems"){};

    private static final OutputTag<RoutedEvent> AUDIT_TAG =
        new OutputTag<RoutedEvent>("audit"){};

    private static final OutputTag<RoutedEvent> FAILED_ROUTING_TAG =
        new OutputTag<RoutedEvent>("failed-routing"){};

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Module 6: Egress & Multi-Sink Routing");

        // Set up execution environment
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Configure for egress processing
        env.setParallelism(10);
        env.enableCheckpointing(30000);
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);

        // Create egress routing pipeline
        createEgressRoutingPipeline(env);

        // Execute the job
        env.execute("Module 6: Egress & Multi-Sink Routing");
    }

    public static void createEgressRoutingPipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating egress routing pipeline");

        // Input streams from all previous modules
        DataStream<SemanticEvent> semanticEvents = createSemanticEventSource(env);
        DataStream<PatternEvent> patternEvents = createPatternEventSource(env);
        DataStream<MLPrediction> mlPredictions = createMLPredictionSource(env);
        DataStream<Module3_ComprehensiveCDS.CDSEvent> cdsEvents = createCDSEventSource(env);

        // ===== Event Prioritization and Routing =====

        // Priority routing for critical events
        SingleOutputStreamOperator<RoutedEvent> routedSemanticEvents = semanticEvents
            .process(new SemanticEventRouter());

        SingleOutputStreamOperator<RoutedEvent> routedPatternEvents = patternEvents
            .process(new PatternEventRouter());

        SingleOutputStreamOperator<RoutedEvent> routedMLPredictions = mlPredictions
            .process(new MLPredictionRouter());

        SingleOutputStreamOperator<RoutedEvent> routedCDSEvents = cdsEvents
            .process(new CDSEventRouter());

        // ===== Unified Routing Stream =====

        DataStream<RoutedEvent> allRoutedEvents = routedSemanticEvents
            .union(routedPatternEvents)
            .union(routedMLPredictions)
            .union(routedCDSEvents);

        // ===== Multi-Format Transformation =====

        // Transform events for different consumer formats
        SingleOutputStreamOperator<RoutedEvent> transformedEvents = allRoutedEvents
            .map(new EventTransformationFunction());

        // ===== HYBRID KAFKA TOPIC ARCHITECTURE =====
        // Transform to EnrichedClinicalEvent format for new architecture

        DataStream<EnrichedClinicalEvent> enrichedEvents = transformedEvents
            .map(new RoutedEventToEnrichedEventMapper());

        // **CORE: Transactional Multi-Sink Router**
        // This implements the recommended hybrid architecture with EXACTLY_ONCE semantics
        // Capture the output stream to connect side outputs to Kafka sinks
        SingleOutputStreamOperator<Void> routedStream = enrichedEvents
            .process(new TransactionalMultiSinkRouter());

        // Wire all 6 side outputs to their respective Kafka sinks
        // This ensures events actually get written to hybrid topics
        routedStream.getSideOutput(TransactionalMultiSinkRouter.CENTRAL_OUTPUT)
            .sinkTo(createHybridCentralSink())
            .name("Hybrid Central Sink");

        routedStream.getSideOutput(TransactionalMultiSinkRouter.ALERTS_OUTPUT)
            .sinkTo(createHybridAlertsSink())
            .name("Hybrid Alerts Sink");

        routedStream.getSideOutput(TransactionalMultiSinkRouter.FHIR_OUTPUT)
            .sinkTo(createHybridFHIRSink())
            .name("Hybrid FHIR Sink");

        routedStream.getSideOutput(TransactionalMultiSinkRouter.ANALYTICS_OUTPUT)
            .sinkTo(createHybridAnalyticsSink())
            .name("Hybrid Analytics Sink");

        // TEMPORARILY DISABLED: Graph mutations sink causing crashes
        // TODO: Re-enable after architectural fix (Option C: single transactional sink + idempotent consumers)
        // routedStream.getSideOutput(TransactionalMultiSinkRouter.GRAPH_OUTPUT)
        //     .sinkTo(createHybridGraphSink())
        //     .name("Hybrid Graph Sink");

        routedStream.getSideOutput(TransactionalMultiSinkRouter.AUDIT_OUTPUT)
            .sinkTo(createHybridAuditSink())
            .name("Hybrid Audit Sink");

        LOG.info("🏗️ Hybrid Architecture Activated:");
        LOG.info("  📊 Central Topic: {}", KafkaTopics.EHR_EVENTS_ENRICHED.getTopicName());
        LOG.info("  🚨 Critical Alerts: {}", KafkaTopics.EHR_ALERTS_CRITICAL_ACTION.getTopicName());
        LOG.info("  🏥 FHIR Upsert: {}", KafkaTopics.EHR_FHIR_UPSERT.getTopicName());
        LOG.info("  📈 Analytics: {}", KafkaTopics.EHR_ANALYTICS_EVENTS.getTopicName());
        // LOG.info("  🕸️ Graph: {}", KafkaTopics.EHR_GRAPH_MUTATIONS.getTopicName()); // DISABLED: Causing crashes
        LOG.info("  📋 Audit: {}", KafkaTopics.EHR_AUDIT_LOGS.getTopicName());

        // ===== LEGACY ROUTING (for migration period) =====

        // Route to primary clinical systems (legacy)
        transformedEvents
            .filter(event -> event.hasDestination("clinical_workflow"))
            .sinkTo(createClinicalWorkflowSink());

        // Route critical alerts with high priority (legacy)
        transformedEvents.getSideOutput(CRITICAL_ALERTS_TAG)
            .sinkTo(createCriticalAlertsSink());

        // Route to analytics systems (legacy)
        transformedEvents.getSideOutput(ANALYTICS_TAG)
            .sinkTo(createAnalyticsSink());

        // Route to external systems (legacy)
        transformedEvents.getSideOutput(EXTERNAL_SYSTEMS_TAG)
            .sinkTo(createExternalSystemsSink());

        // Route to audit and compliance systems
        transformedEvents.getSideOutput(AUDIT_TAG)
            .sinkTo(createAuditSink());

        // Handle routing failures
        transformedEvents.getSideOutput(FAILED_ROUTING_TAG)
            .sinkTo(createFailedRoutingSink());

        // ===== Real-time Export Endpoints =====

        // High-priority events for immediate notification
        DataStream<RoutedEvent> highPriorityEvents = transformedEvents
            .filter(event -> event.getPriority() == RoutedEvent.Priority.CRITICAL ||
                           event.getPriority() == RoutedEvent.Priority.HIGH);

        highPriorityEvents
            .sinkTo(createRealtimeNotificationSink());

        // ===== Batch Export for Analytics =====

        // Batch events for analytics and reporting
        DataStream<RoutedEvent> analyticsEvents = transformedEvents
            .filter(event -> event.hasDestination("analytics") || event.hasDestination("reporting"));

        analyticsEvents
            .sinkTo(createBatchAnalyticsSink());

        LOG.info("Egress routing pipeline created successfully");
    }

    // ===== Input Sources =====

    private static DataStream<SemanticEvent> createSemanticEventSource(StreamExecutionEnvironment env) {
        KafkaSource<SemanticEvent> source = KafkaSource.<SemanticEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(KafkaTopics.SEMANTIC_MESH_UPDATES.getTopicName())
            .setGroupId("egress-semantic")
            .setStartingOffsets(OffsetsInitializer.earliest())  // Read all historical events
            .setValueOnlyDeserializer(new SemanticEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("egress-semantic"))
            .build();

        return env.fromSource(source, WatermarkStrategy.noWatermarks(), "Egress Semantic Events");
    }

    private static DataStream<PatternEvent> createPatternEventSource(StreamExecutionEnvironment env) {
        KafkaSource<PatternEvent> source = KafkaSource.<PatternEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(KafkaTopics.CLINICAL_PATTERNS.getTopicName())
            .setGroupId("egress-patterns")
            .setStartingOffsets(OffsetsInitializer.earliest())  // Read all historical events
            .setValueOnlyDeserializer(new PatternEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("egress-patterns"))
            .build();

        return env.fromSource(source, WatermarkStrategy.noWatermarks(), "Egress Pattern Events");
    }

    private static DataStream<MLPrediction> createMLPredictionSource(StreamExecutionEnvironment env) {
        KafkaSource<MLPrediction> source = KafkaSource.<MLPrediction>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(KafkaTopics.INFERENCE_RESULTS.getTopicName())
            .setGroupId("egress-ml")
            .setStartingOffsets(OffsetsInitializer.earliest())  // Read all historical events
            .setValueOnlyDeserializer(new MLPredictionDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("egress-ml"))
            .build();

        return env.fromSource(source, WatermarkStrategy.noWatermarks(), "Egress ML Predictions");
    }

    private static DataStream<Module3_ComprehensiveCDS.CDSEvent> createCDSEventSource(StreamExecutionEnvironment env) {
        KafkaSource<Module3_ComprehensiveCDS.CDSEvent> source = KafkaSource.<Module3_ComprehensiveCDS.CDSEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics("comprehensive-cds-events.v1")
            .setGroupId("egress-cds-events")
            .setStartingOffsets(OffsetsInitializer.earliest())  // CRITICAL: Read historical CDS events with drug interactions
            .setValueOnlyDeserializer(new CDSEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("egress-cds-events"))
            .build();

        return env.fromSource(source, WatermarkStrategy.noWatermarks(), "Egress CDS Events");
    }

    // ===== Routing Functions =====

    /**
     * Router for semantic events
     */
    public static class SemanticEventRouter extends ProcessFunction<SemanticEvent, RoutedEvent> {
        @Override
        public void processElement(SemanticEvent value, Context ctx, Collector<RoutedEvent> out) throws Exception {
            RoutedEvent routed = new RoutedEvent();
            routed.setId(UUID.randomUUID().toString());
            routed.setSourceEventId(value.getId());
            // CRITICAL FIX: Preserve original clinical EventType from upstream
            // SemanticEvent contains EventType enum (PATIENT_ADMISSION, VITAL_SIGN, MEDICATION_ORDERED, etc.)
            // This enables FHIR routing based on clinical event type, not technical taxonomy
            String originalEventType = (value.getEventType() != null)
                ? value.getEventType().name()  // e.g., "PATIENT_ADMISSION", "VITAL_SIGN", "LAB_RESULT"
                : "SEMANTIC_EVENT";            // Fallback for events without type
            routed.setSourceEventType(originalEventType);
            routed.setPatientId(value.getPatientId());
            routed.setRoutingTime(System.currentTimeMillis());
            routed.setOriginalPayload(value);

            // Determine routing destinations based on event characteristics
            Set<String> destinations = new HashSet<>();

            // All semantic events go to analytics
            destinations.add("analytics");
            destinations.add("audit");

            // Critical events get priority routing
            if (value.hasHighClinicalSignificance()) {
                destinations.add("clinical_workflow");
                destinations.add("fhir_store"); // Store significant clinical events in FHIR
                routed.setPriority(RoutedEvent.Priority.HIGH);

                if (value.hasClinicalAlerts()) {
                    destinations.add("critical_alerts");
                    routed.setPriority(RoutedEvent.Priority.CRITICAL);
                    ctx.output(CRITICAL_ALERTS_TAG, routed);
                }
            } else {
                destinations.add("fhir_store"); // Store all semantic events in FHIR
                routed.setPriority(RoutedEvent.Priority.NORMAL);
            }

            // Drug interactions require external system notification
            if (value.hasDrugInteractions()) {
                destinations.add("external_systems");
                destinations.add("pharmacy_systems");
            }

            // Guideline recommendations go to clinical workflow
            if (value.hasGuidelineRecommendations()) {
                destinations.add("clinical_workflow");
                destinations.add("quality_systems");
            }

            routed.setDestinations(destinations);
            routed.setTransformationRules(getTransformationRules(value, destinations));

            // Route to appropriate side outputs
            routeToSideOutputs(routed, ctx);

            out.collect(routed);
        }

        private Set<String> getTransformationRules(SemanticEvent event, Set<String> destinations) {
            Set<String> rules = new HashSet<>();

            for (String destination : destinations) {
                switch (destination) {
                    case "clinical_workflow":
                        rules.add("FHIR_R4_TRANSFORM");
                        rules.add("CLINICAL_SUMMARY");
                        break;
                    case "fhir_store":
                    case "google_healthcare":
                        rules.add("FHIR_R4_TRANSFORM");
                        rules.add("CLINICAL_SUMMARY");
                        break;
                    case "external_systems":
                        rules.add("HL7_V2_TRANSFORM");
                        rules.add("INTEROP_FORMAT");
                        break;
                    case "analytics":
                        rules.add("ANALYTICS_FLATTEN");
                        rules.add("TIMESTAMP_NORMALIZE");
                        break;
                    case "audit":
                        rules.add("AUDIT_FORMAT");
                        rules.add("PRIVACY_FILTER");
                        break;
                }
            }

            return rules;
        }
    }

    /**
     * Router for pattern events
     */
    public static class PatternEventRouter extends ProcessFunction<PatternEvent, RoutedEvent> {
        @Override
        public void processElement(PatternEvent value, Context ctx, Collector<RoutedEvent> out) throws Exception {
            RoutedEvent routed = new RoutedEvent();
            routed.setId(UUID.randomUUID().toString());
            routed.setSourceEventId(value.getId());
            routed.setSourceEventType("PATTERN_EVENT");
            routed.setPatientId(value.getPatientId());
            routed.setRoutingTime(System.currentTimeMillis());
            routed.setOriginalPayload(value);

            Set<String> destinations = new HashSet<>();
            destinations.add("analytics");

            // Route based on pattern type and severity
            if (value.isDeteriorationPattern() && value.isHighSeverity()) {
                destinations.add("critical_alerts");
                destinations.add("clinical_workflow");
                destinations.add("fhir_store"); // Store critical patterns in FHIR
                routed.setPriority(RoutedEvent.Priority.CRITICAL);
                ctx.output(CRITICAL_ALERTS_TAG, routed);
            } else if (value.isHighSeverity()) {
                destinations.add("clinical_workflow");
                destinations.add("fhir_store"); // Store high-severity patterns in FHIR
                routed.setPriority(RoutedEvent.Priority.HIGH);
            } else {
                destinations.add("fhir_store"); // Store all patterns in FHIR for analysis
                routed.setPriority(RoutedEvent.Priority.NORMAL);
            }

            // Pathway compliance patterns go to quality systems
            if (value.isPathwayCompliancePattern()) {
                destinations.add("quality_systems");
                destinations.add("workflow_systems");
            }

            // Medication adherence patterns go to pharmacy systems
            if (value.isMedicationAdherencePattern()) {
                destinations.add("pharmacy_systems");
                destinations.add("care_coordination");
            }

            routed.setDestinations(destinations);
            routed.setTransformationRules(getPatternTransformationRules(value, destinations));

            routeToSideOutputs(routed, ctx);
            out.collect(routed);
        }

        private Set<String> getPatternTransformationRules(PatternEvent event, Set<String> destinations) {
            Set<String> rules = new HashSet<>();

            rules.add("PATTERN_SUMMARY");
            rules.add("TIMESTAMP_NORMALIZE");

            if (destinations.contains("clinical_workflow")) {
                rules.add("CLINICAL_PATTERN_FORMAT");
                rules.add("RECOMMENDATION_EXTRACT");
            }

            if (destinations.contains("analytics")) {
                rules.add("PATTERN_ANALYTICS_FORMAT");
                rules.add("STATISTICAL_SUMMARY");
            }

            return rules;
        }
    }

    /**
     * Router for ML predictions
     */
    public static class MLPredictionRouter extends ProcessFunction<MLPrediction, RoutedEvent> {
        @Override
        public void processElement(MLPrediction value, Context ctx, Collector<RoutedEvent> out) throws Exception {
            RoutedEvent routed = new RoutedEvent();
            routed.setId(UUID.randomUUID().toString());
            routed.setSourceEventId(value.getId());
            routed.setSourceEventType("ML_PREDICTION");
            routed.setPatientId(value.getPatientId());
            routed.setRoutingTime(System.currentTimeMillis());
            routed.setOriginalPayload(value);

            Set<String> destinations = new HashSet<>();
            destinations.add("analytics");

            // Route based on prediction type and risk level
            if (value.requiresImmediateAttention()) {
                destinations.add("critical_alerts");
                destinations.add("clinical_workflow");
                destinations.add("fhir_store"); // Store critical predictions in FHIR
                routed.setPriority(RoutedEvent.Priority.CRITICAL);
                ctx.output(CRITICAL_ALERTS_TAG, routed);
            } else if (value.isHighRisk()) {
                destinations.add("clinical_workflow");
                destinations.add("fhir_store"); // Store high-risk predictions in FHIR
                routed.setPriority(RoutedEvent.Priority.HIGH);
            } else {
                destinations.add("fhir_store"); // Store all predictions in FHIR for tracking
                routed.setPriority(RoutedEvent.Priority.NORMAL);
            }

            // Specific prediction types require specialized routing
            switch (value.getModelType()) {
                case "SEPSIS_PREDICTION":
                    if (value.isHighRisk()) {
                        destinations.add("infection_control");
                        destinations.add("laboratory_systems");
                    }
                    break;
                case "READMISSION_RISK":
                    destinations.add("case_management");
                    destinations.add("discharge_planning");
                    break;
                case "FALL_RISK":
                    destinations.add("safety_systems");
                    destinations.add("nursing_workflow");
                    break;
                case "MORTALITY_RISK":
                    if (value.isHighRisk()) {
                        destinations.add("palliative_care");
                        destinations.add("ethics_committee");
                    }
                    break;
            }

            routed.setDestinations(destinations);
            routed.setTransformationRules(getMLTransformationRules(value, destinations));

            routeToSideOutputs(routed, ctx);
            out.collect(routed);
        }

        private Set<String> getMLTransformationRules(MLPrediction prediction, Set<String> destinations) {
            Set<String> rules = new HashSet<>();

            rules.add("ML_PREDICTION_FORMAT");
            rules.add("CONFIDENCE_SCORE_INCLUDE");

            if (destinations.contains("clinical_workflow")) {
                rules.add("CLINICAL_INTERPRETATION");
                rules.add("RECOMMENDATION_FORMAT");
            }

            if (prediction.isEnsemblePrediction()) {
                rules.add("ENSEMBLE_METADATA");
            }

            return rules;
        }
    }

    /**
     * Router for context snapshots
     */
    public static class CDSEventRouter extends ProcessFunction<Module3_ComprehensiveCDS.CDSEvent, RoutedEvent> {
        @Override
        public void processElement(Module3_ComprehensiveCDS.CDSEvent value, Context ctx, Collector<RoutedEvent> out) throws Exception {
            RoutedEvent routed = new RoutedEvent();
            routed.setId(UUID.randomUUID().toString());
            routed.setSourceEventId(value.getPatientId() + "_cds_" + value.getEventTime());
            // CRITICAL FIX: Preserve original clinical eventType from CDSEvent
            // CDSEvent.eventType contains the original event type string from PatientContextState
            // (e.g., "PATIENT_ADMISSION", "VITAL_SIGN", "LAB_RESULT", "MEDICATION_ORDERED")
            String originalEventType = (value.getEventType() != null && !value.getEventType().isEmpty())
                ? value.getEventType()  // e.g., "PATIENT_ADMISSION", "VITAL_SIGN"
                : "CDS_EVENT";          // Fallback for events without type
            routed.setSourceEventType(originalEventType);
            routed.setPatientId(value.getPatientId());
            routed.setRoutingTime(System.currentTimeMillis());
            routed.setOriginalPayload(value);

            Set<String> destinations = new HashSet<>();
            destinations.add("analytics");
            destinations.add("data_warehouse");

            // Use patient state for routing logic
            PatientContextState state = value.getPatientState();
            if (state != null) {
                // High acuity patients get clinical workflow and care coordination routing
                Double acuityScore = state.getCombinedAcuityScore();
                if (acuityScore != null && acuityScore >= 7.0) {  // CRITICAL/HIGH acuity
                    destinations.add("clinical_workflow");
                    destinations.add("care_coordination");  // High acuity implies active care coordination
                    routed.setPriority(RoutedEvent.Priority.HIGH);
                } else {
                    routed.setPriority(RoutedEvent.Priority.NORMAL);
                }
            } else {
                routed.setPriority(RoutedEvent.Priority.NORMAL);
            }

            routed.setDestinations(destinations);
            routed.setTransformationRules(getCDSTransformationRules(value, destinations));

            routeToSideOutputs(routed, ctx);
            out.collect(routed);
        }

        private Set<String> getCDSTransformationRules(Module3_ComprehensiveCDS.CDSEvent cdsEvent, Set<String> destinations) {
            Set<String> rules = new HashSet<>();

            rules.add("CDS_SUMMARY");
            rules.add("PHASE_DATA_METADATA");

            if (destinations.contains("clinical_workflow")) {
                rules.add("CLINICAL_CDS_FORMAT");
            }

            if (destinations.contains("analytics")) {
                rules.add("ANALYTICS_CDS_FORMAT");
                rules.add("CDS_TREND_CALCULATION");
            }

            return rules;
        }
    }

    /**
     * Route events to appropriate side outputs
     */
    private static void routeToSideOutputs(RoutedEvent routed, ProcessFunction.Context ctx) {
        Set<String> destinations = routed.getDestinations();

        if (destinations.contains("clinical_workflow")) {
            ctx.output(CLINICAL_WORKFLOW_TAG, routed);
        }

        if (destinations.contains("analytics") || destinations.contains("data_warehouse")) {
            ctx.output(ANALYTICS_TAG, routed);
        }

        if (destinations.contains("external_systems") || destinations.contains("interop")) {
            ctx.output(EXTERNAL_SYSTEMS_TAG, routed);
        }

        if (destinations.contains("audit") || destinations.contains("compliance")) {
            ctx.output(AUDIT_TAG, routed);
        }
    }

    // ===== Event Transformation =====

    /**
     * Transform events based on destination requirements
     */
    public static class EventTransformationFunction implements MapFunction<RoutedEvent, RoutedEvent> {
        @Override
        public RoutedEvent map(RoutedEvent event) throws Exception {
            // Apply transformation rules based on destinations
            for (String rule : event.getTransformationRules()) {
                applyTransformationRule(event, rule);
            }

            // Set transformation metadata
            Map<String, Object> transformationMetadata = new HashMap<>();
            transformationMetadata.put("applied_rules", event.getTransformationRules());
            transformationMetadata.put("transformation_time", System.currentTimeMillis());
            transformationMetadata.put("destinations", event.getDestinations());

            event.setTransformationMetadata(transformationMetadata);

            return event;
        }

        private void applyTransformationRule(RoutedEvent event, String rule) {
            Map<String, Object> transformedPayload = new HashMap<>();

            switch (rule) {
                case "FHIR_R4_TRANSFORM":
                    transformedPayload = transformToFHIR(event);
                    break;
                case "HL7_V2_TRANSFORM":
                    transformedPayload = transformToHL7(event);
                    break;
                case "ANALYTICS_FLATTEN":
                    transformedPayload = flattenForAnalytics(event);
                    break;
                case "CLINICAL_SUMMARY":
                    transformedPayload = createClinicalSummary(event);
                    break;
                case "AUDIT_FORMAT":
                    transformedPayload = formatForAudit(event);
                    break;
                case "PRIVACY_FILTER":
                    transformedPayload = applyPrivacyFilter(event);
                    break;
                default:
                    transformedPayload.put("original", event.getOriginalPayload());
                    break;
            }

            event.getTransformedPayloads().put(rule, transformedPayload);
        }

        private Map<String, Object> transformToFHIR(RoutedEvent event) {
            Map<String, Object> fhirPayload = new HashMap<>();
            fhirPayload.put("resourceType", determineFHIRResourceType(event));
            fhirPayload.put("id", event.getSourceEventId());
            fhirPayload.put("meta", createFHIRMeta());
            fhirPayload.put("subject", Map.of("reference", "Patient/" + event.getPatientId()));
            fhirPayload.put("effectiveDateTime", new Date(event.getRoutingTime()));
            fhirPayload.put("valueCodeableConcept", extractClinicalConcepts(event));
            return fhirPayload;
        }

        private Map<String, Object> transformToHL7(RoutedEvent event) {
            Map<String, Object> hl7Payload = new HashMap<>();
            hl7Payload.put("MSH", createHL7Header(event));
            hl7Payload.put("PID", createHL7PatientInfo(event));
            hl7Payload.put("OBX", createHL7Observation(event));
            return hl7Payload;
        }

        private Map<String, Object> flattenForAnalytics(RoutedEvent event) {
            Map<String, Object> flatPayload = new HashMap<>();
            flatPayload.put("event_id", event.getSourceEventId());
            flatPayload.put("patient_id", event.getPatientId());
            flatPayload.put("event_type", event.getSourceEventType());
            flatPayload.put("routing_time", event.getRoutingTime());
            flatPayload.put("priority", event.getPriority().name());
            flatPayload.put("destination_count", event.getDestinations().size());

            // Flatten nested data structures
            if (event.getOriginalPayload() instanceof Map) {
                flattenMap((Map<String, Object>) event.getOriginalPayload(), flatPayload, "");
            }

            return flatPayload;
        }

        private Map<String, Object> createClinicalSummary(RoutedEvent event) {
            Map<String, Object> summary = new HashMap<>();
            summary.put("patient_id", event.getPatientId());
            summary.put("event_type", event.getSourceEventType());
            summary.put("priority", event.getPriority().name());
            summary.put("timestamp", new Date(event.getRoutingTime()));
            summary.put("clinical_significance", extractClinicalSignificance(event));
            summary.put("recommended_actions", extractRecommendedActions(event));
            return summary;
        }

        private Map<String, Object> formatForAudit(RoutedEvent event) {
            Map<String, Object> auditPayload = new HashMap<>();
            auditPayload.put("audit_id", UUID.randomUUID().toString());
            auditPayload.put("event_id", event.getSourceEventId());
            auditPayload.put("patient_id", event.getPatientId());
            auditPayload.put("event_type", event.getSourceEventType());
            auditPayload.put("timestamp", event.getRoutingTime());
            auditPayload.put("destinations", event.getDestinations());
            auditPayload.put("transformation_rules", event.getTransformationRules());
            auditPayload.put("data_classification", "CLINICAL");
            return auditPayload;
        }

        private Map<String, Object> applyPrivacyFilter(RoutedEvent event) {
            Map<String, Object> filteredPayload = new HashMap<>();

            // Remove or mask sensitive data
            filteredPayload.put("event_id", event.getSourceEventId());
            filteredPayload.put("patient_id_hash", hashPatientId(event.getPatientId()));
            filteredPayload.put("event_type", event.getSourceEventType());
            filteredPayload.put("timestamp", event.getRoutingTime());
            filteredPayload.put("clinical_indicators", extractNonSensitiveClinicalData(event));

            return filteredPayload;
        }

        // Helper methods for transformations
        private String determineFHIRResourceType(RoutedEvent event) {
            switch (event.getSourceEventType()) {
                case "SEMANTIC_EVENT":
                    return "Observation";
                case "PATTERN_EVENT":
                    return "DiagnosticReport";
                case "ML_PREDICTION":
                    return "RiskAssessment";
                case "CONTEXT_SNAPSHOT":
                    return "Encounter";
                default:
                    return "Basic";
            }
        }

        private Map<String, Object> createFHIRMeta() {
            Map<String, Object> meta = new HashMap<>();
            meta.put("versionId", "1");
            meta.put("lastUpdated", new Date());
            meta.put("source", "CardioFit-EHR-Intelligence");
            return meta;
        }

        private Map<String, Object> createHL7Header(RoutedEvent event) {
            Map<String, Object> msh = new HashMap<>();
            msh.put("sending_application", "CardioFit-Flink");
            msh.put("sending_facility", "CardioFit-Hospital");
            msh.put("message_type", "ORU^R01^ORU_R01");
            msh.put("timestamp", new Date(event.getRoutingTime()));
            return msh;
        }

        private Map<String, Object> createHL7PatientInfo(RoutedEvent event) {
            Map<String, Object> pid = new HashMap<>();
            pid.put("patient_id", event.getPatientId());
            return pid;
        }

        private Map<String, Object> createHL7Observation(RoutedEvent event) {
            Map<String, Object> obx = new HashMap<>();
            obx.put("observation_identifier", event.getSourceEventType());
            obx.put("observation_value", extractObservationValue(event));
            obx.put("observation_datetime", new Date(event.getRoutingTime()));
            return obx;
        }

        private void flattenMap(Map<String, Object> source, Map<String, Object> target, String prefix) {
            for (Map.Entry<String, Object> entry : source.entrySet()) {
                String key = prefix.isEmpty() ? entry.getKey() : prefix + "_" + entry.getKey();
                Object value = entry.getValue();

                if (value instanceof Map) {
                    flattenMap((Map<String, Object>) value, target, key);
                } else {
                    target.put(key, value);
                }
            }
        }

        private Object extractClinicalConcepts(RoutedEvent event) {
            // Extract clinical concepts based on event type
            return Map.of("text", "Clinical event processed by EHR Intelligence Engine");
        }

        private Object extractClinicalSignificance(RoutedEvent event) {
            // Extract clinical significance from original payload
            return "Moderate"; // Simplified
        }

        private Object extractRecommendedActions(RoutedEvent event) {
            // Extract recommended actions from original payload
            return Arrays.asList("Monitor patient", "Review clinical data");
        }

        private String hashPatientId(String patientId) {
            // Simple hash for privacy (in practice, use proper cryptographic hash)
            return "HASH_" + patientId.hashCode();
        }

        private Object extractNonSensitiveClinicalData(RoutedEvent event) {
            // Extract only non-sensitive clinical indicators
            Map<String, Object> indicators = new HashMap<>();
            indicators.put("event_category", event.getSourceEventType());
            indicators.put("priority_level", event.getPriority().name());
            return indicators;
        }

        private Object extractObservationValue(RoutedEvent event) {
            // Extract observation value from event
            return event.getSourceEventType() + "_OBSERVATION";
        }
    }

    // ===== Sink Creation Methods =====

    private static KafkaSink<RoutedEvent> createClinicalWorkflowSink() {
        return createRoutedEventSink(KafkaTopics.WORKFLOW_EVENTS, "module6-clinical-workflow");
    }

    private static KafkaSink<RoutedEvent> createCriticalAlertsSink() {
        return createRoutedEventSink(KafkaTopics.ALERT_MANAGEMENT, "module6-critical-alerts");
    }

    private static KafkaSink<RoutedEvent> createAnalyticsSink() {
        return createRoutedEventSink(KafkaTopics.PERFORMANCE_METRICS, "module6-analytics");
    }

    private static KafkaSink<RoutedEvent> createExternalSystemsSink() {
        return createRoutedEventSink(KafkaTopics.HL7_OUTBOUND, "module6-external-systems");
    }

    private static KafkaSink<RoutedEvent> createAuditSink() {
        return createRoutedEventSink(KafkaTopics.AUDIT_EVENTS, "module6-audit");
    }

    private static KafkaSink<RoutedEvent> createFailedRoutingSink() {
        return createRoutedEventSink(KafkaTopics.DLQ_PROCESSING_ERRORS, "module6-failed-routing");
    }

    private static KafkaSink<RoutedEvent> createRealtimeNotificationSink() {
        return createRoutedEventSink(KafkaTopics.NOTIFICATION_EVENTS, "module6-realtime-notification");
    }

    private static KafkaSink<RoutedEvent> createBatchAnalyticsSink() {
        return createRoutedEventSink(KafkaTopics.PRECOMPUTED_VIEWS, "module6-batch-analytics");
    }


    private static GoogleFHIRStoreSink createGoogleFHIRStoreSink() {
        // Use configuration compatible with patient-service settings
        GoogleHealthcareConfig config = GoogleHealthcareConfig.createDevelopmentConfig();

        LOG.info("Creating Google FHIR Store sink with configuration: {}", config.toString());

        return new GoogleFHIRStoreSink(config);
    }

    private static KafkaSink<RoutedEvent> createRoutedEventSink(KafkaTopics topic, String transactionalIdPrefix) {
        // Create producer config WITHOUT key/value serializers (Flink handles serialization)
        java.util.Properties producerConfig = KafkaConfigLoader.getAutoProducerConfig();
        producerConfig.remove("key.serializer");
        producerConfig.remove("value.serializer");

        return KafkaSink.<RoutedEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(topic.getTopicName())
                .setKeySerializationSchema((RoutedEvent event) -> {
                    String patientId = event.getPatientId();
                    return (patientId != null ? patientId : "UNKNOWN").getBytes();
                })
                .setValueSerializationSchema(new RoutedEventSerializer())
                .build())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix(transactionalIdPrefix)
            .setKafkaProducerConfig(producerConfig)
            .build();
    }

    private static String getBootstrapServers() {
        return KafkaConfigLoader.isRunningInDocker()
            ? "kafka1:29092,kafka2:29093,kafka3:29094"
            : "localhost:9092,localhost:9093,localhost:9094";
    }

    // ===== Serialization Classes =====

    private static class SemanticEventDeserializer implements DeserializationSchema<SemanticEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
            objectMapper.configure(com.fasterxml.jackson.databind.DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
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

    private static class PatternEventDeserializer implements DeserializationSchema<PatternEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
            objectMapper.configure(com.fasterxml.jackson.databind.DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
        }

        @Override
        public PatternEvent deserialize(byte[] message) throws IOException {
            return objectMapper.readValue(message, PatternEvent.class);
        }

        @Override
        public boolean isEndOfStream(PatternEvent nextElement) { return false; }

        @Override
        public TypeInformation<PatternEvent> getProducedType() {
            return TypeInformation.of(PatternEvent.class);
        }
    }

    private static class MLPredictionDeserializer implements DeserializationSchema<MLPrediction> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
            objectMapper.configure(com.fasterxml.jackson.databind.DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
        }

        @Override
        public MLPrediction deserialize(byte[] message) throws IOException {
            return objectMapper.readValue(message, MLPrediction.class);
        }

        @Override
        public boolean isEndOfStream(MLPrediction nextElement) { return false; }

        @Override
        public TypeInformation<MLPrediction> getProducedType() {
            return TypeInformation.of(MLPrediction.class);
        }
    }

    private static class CDSEventDeserializer implements DeserializationSchema<Module3_ComprehensiveCDS.CDSEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
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

    private static class RoutedEventSerializer implements SerializationSchema<RoutedEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(RoutedEvent element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                throw new RuntimeException("Failed to serialize RoutedEvent", e);
            }
        }
    }

    /**
     * Mapper to convert RoutedEvent to EnrichedClinicalEvent for hybrid architecture
     */
    private static class RoutedEventToEnrichedEventMapper implements MapFunction<RoutedEvent, EnrichedClinicalEvent> {
    public static class DetectedPattern {
        private String patternType;
        private double confidence;
        public DetectedPattern(String type, double conf) { this.patternType = type; this.confidence = conf; }
        public String getPatternType() { return patternType; }
        public double getConfidence() { return confidence; }
    }

        @Override
        public EnrichedClinicalEvent map(RoutedEvent routedEvent) throws Exception {
            EnrichedClinicalEvent enriched = new EnrichedClinicalEvent();

            // Basic event information
            enriched.setId(routedEvent.getSourceEventId());
            enriched.setPatientId(routedEvent.getPatientId());
            enriched.setTimestamp(routedEvent.getRoutingTime());
            enriched.setSourceEventType(routedEvent.getSourceEventType());

            // DEBUG: Log payload type
            String payloadType = routedEvent.getOriginalPayload() != null ?
                routedEvent.getOriginalPayload().getClass().getSimpleName() : "NULL";
            System.out.println("[MODULE6B-MAPPER] Processing " + routedEvent.getSourceEventType() +
                " with payload type: " + payloadType + " for patient: " + routedEvent.getPatientId());

            // Extract clinical context from original payload
            if (routedEvent.getOriginalPayload() instanceof SemanticEvent) {
                SemanticEvent semantic = (SemanticEvent) routedEvent.getOriginalPayload();
                enriched.setClinicalSignificance(semantic.getClinicalSignificance());
                enriched.setDrugInteractions(semantic.getDrugInteractions());
                enriched.setClinicalConcepts(semantic.getClinicalConcepts());
                // Note: setPatientRelationshipChanges and setClinicalConceptRelationships methods not available
                // in EnrichedClinicalEvent - skipping these enrichments
            } else if (routedEvent.getOriginalPayload() instanceof PatternEvent) {
                PatternEvent pattern = (PatternEvent) routedEvent.getOriginalPayload();
                enriched.setClinicalSignificance(pattern.getConfidence());

                // Convert pattern to detected patterns list
                List<String> patterns = new ArrayList<>();
                patterns.add(pattern.getPatternType() + ":" + pattern.getConfidence());
                enriched.setDetectedPatterns(patterns);
            } else if (routedEvent.getOriginalPayload() instanceof MLPrediction) {
                MLPrediction prediction = (MLPrediction) routedEvent.getOriginalPayload();

                // Convert ML prediction to predictions list
                List<MLPrediction> predictions = new ArrayList<>();
                predictions.add(prediction);
                enriched.setMlPredictions(predictions);
                enriched.setClinicalSignificance(prediction.getConfidence());
            } else if (routedEvent.getOriginalPayload() instanceof Module3_ComprehensiveCDS.CDSEvent) {
                // CRITICAL FIX: Extract drug interactions and clinical data from CDSEvent
                Module3_ComprehensiveCDS.CDSEvent cdsEvent = (Module3_ComprehensiveCDS.CDSEvent) routedEvent.getOriginalPayload();
                enriched.setClinicalSignificance(0.8); // CDS events have high clinical significance

                LOG.info("🔍 [MODULE6B-DEBUG] Processing CDSEvent for patient: {}", cdsEvent.getPatientId());

                // Extract drug interactions from semantic enrichment
                if (cdsEvent.getSemanticEnrichment() == null) {
                    LOG.warn("⚠️ [MODULE6B-DEBUG] SemanticEnrichment is NULL for patient: {}", cdsEvent.getPatientId());
                } else if (cdsEvent.getSemanticEnrichment().getDrugInteractionAnalysis() == null) {
                    LOG.warn("⚠️ [MODULE6B-DEBUG] DrugInteractionAnalysis is NULL for patient: {}", cdsEvent.getPatientId());
                } else {
                    SemanticEnrichment.DrugInteractionAnalysis analysis =
                        cdsEvent.getSemanticEnrichment().getDrugInteractionAnalysis();

                    LOG.info("🔍 [MODULE6B-DEBUG] DrugInteractionAnalysis found with {} interactions detected",
                        analysis.getInteractionsDetected());

                    // Convert InteractionWarnings to SemanticEvent.DrugInteraction format
                    if (analysis.getInteractionWarnings() == null) {
                        LOG.warn("⚠️ [MODULE6B-DEBUG] InteractionWarnings list is NULL");
                    } else if (analysis.getInteractionWarnings().isEmpty()) {
                        LOG.warn("⚠️ [MODULE6B-DEBUG] InteractionWarnings list is EMPTY");
                    } else {
                        List<SemanticEvent.DrugInteraction> drugInteractions = new ArrayList<>();

                        for (SemanticEnrichment.InteractionWarning warning : analysis.getInteractionWarnings()) {
                            SemanticEvent.DrugInteraction interaction = new SemanticEvent.DrugInteraction();
                            interaction.setInteractionId(java.util.UUID.randomUUID().toString());
                            interaction.setDrug1(warning.getProtocolMedication());
                            interaction.setDrug2(warning.getActiveMedication());
                            interaction.setSeverity(warning.getSeverity());
                            interaction.setInteractionType("DRUG_INTERACTION");
                            interaction.setClinicalEffect(warning.getClinicalEffect());
                            interaction.setRecommendation(warning.getManagement());
                            interaction.setConfidence(0.85); // High confidence from KB
                            drugInteractions.add(interaction);
                        }

                        enriched.setDrugInteractions(drugInteractions);
                        LOG.info("✅ [MODULE6B-DEBUG] Set {} drug interactions on EnrichedClinicalEvent",
                            drugInteractions.size());
                    }
                }

                // Extract patient context from CDSEvent
                if (cdsEvent.getPatientState() != null) {
                    com.cardiofit.stream.state.PatientContext streamContext =
                        new com.cardiofit.stream.state.PatientContext(cdsEvent.getPatientId());
                    enriched.setPatientContext(streamContext);
                }
            } else if (routedEvent.getOriginalPayload() instanceof PatientContext) {
                // Note: PatientContext from flink.models package, conversion needed for EnrichedClinicalEvent
                PatientContext fhirContext = (PatientContext) routedEvent.getOriginalPayload();
                enriched.setClinicalSignificance(0.5); // Default significance for context events

                // Convert flink.models.PatientContext to stream.state.PatientContext
                com.cardiofit.stream.state.PatientContext streamContext =
                    new com.cardiofit.stream.state.PatientContext(fhirContext.getPatientId());
                // Copy basic fields from fhir context to stream context if needed
                enriched.setPatientContext(streamContext);
            }

            // Map routing destinations to clinical event fields
            if (routedEvent.getDestinations() != null) {
                enriched.setDestinations(new HashSet<>(routedEvent.getDestinations()));
            }

            // Set priority based on routing priority
            if (routedEvent.getPriority() != null) {
                switch (routedEvent.getPriority()) {
                    case CRITICAL:
                        enriched.setCriticalEvent(true);
                        break;
                    case HIGH:
                        enriched.setHighClinicalSignificance(true);
                        break;
                    default:
                        enriched.setCriticalEvent(false);
                        enriched.setHighClinicalSignificance(false);
                }
            }

            return enriched;
        }
    }

    // ===== Hybrid Kafka Sink Creation Methods =====

    private static KafkaSink<EnrichedClinicalEvent> createHybridCentralSink() {
        return KafkaSink.<EnrichedClinicalEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.EHR_EVENTS_ENRICHED.getTopicName())
                .setKeySerializationSchema((EnrichedClinicalEvent event) -> {
                    String patientId = event.getPatientId();
                    return (patientId != null ? patientId : "UNKNOWN_PATIENT").getBytes();
                })
                .setValueSerializationSchema(new HybridEnrichedEventSerializer())
                .build())
            .setKafkaProducerConfig(KafkaConfigLoader.getProducerConfigForSink())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module6-hybrid-central")
            .build();
    }

    private static KafkaSink<CriticalAlert> createHybridAlertsSink() {
        return KafkaSink.<CriticalAlert>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.EHR_ALERTS_CRITICAL_ACTION.getTopicName())
                .setKeySerializationSchema((CriticalAlert alert) -> {
                    String patientId = alert.getPatientId();
                    return (patientId != null ? patientId : "UNKNOWN_PATIENT").getBytes();
                })
                .setValueSerializationSchema(new HybridCriticalAlertSerializer())
                .build())
            .setKafkaProducerConfig(KafkaConfigLoader.getProducerConfigForSink())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module6-hybrid-alerts")
            .build();
    }

    private static KafkaSink<FHIRResource> createHybridFHIRSink() {
        return KafkaSink.<FHIRResource>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.EHR_FHIR_UPSERT.getTopicName())
                .setKeySerializationSchema((FHIRResource resource) -> {
                    String resourceType = resource.getResourceType() != null ? resource.getResourceType() : "UNKNOWN";
                    String resourceId = resource.getResourceId() != null ? resource.getResourceId() : "UNKNOWN";
                    return (resourceType + "|" + resourceId).getBytes();
                })
                .setValueSerializationSchema(new HybridFHIRResourceSerializer())
                .build())
            .setKafkaProducerConfig(KafkaConfigLoader.getProducerConfigForSink())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module6-hybrid-fhir")
            .build();
    }

    private static KafkaSink<AnalyticsEvent> createHybridAnalyticsSink() {
        return KafkaSink.<AnalyticsEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.EHR_ANALYTICS_EVENTS.getTopicName())
                .setKeySerializationSchema((AnalyticsEvent event) -> {
                    String patientId = event.getPatientId();
                    return (patientId != null ? patientId : "UNKNOWN_PATIENT").getBytes();
                })
                .setValueSerializationSchema(new HybridAnalyticsEventSerializer())
                .build())
            .setKafkaProducerConfig(KafkaConfigLoader.getProducerConfigForSink())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module6-hybrid-analytics")
            .build();
    }

    // TEMPORARILY DISABLED: Graph mutations sink causing crashes
    // TODO: Re-enable after architectural fix (Option C: single transactional sink + idempotent consumers)
    /*
    private static KafkaSink<GraphMutation> createHybridGraphSink() {
        return KafkaSink.<GraphMutation>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.EHR_GRAPH_MUTATIONS.getTopicName())
                .setKeySerializationSchema((GraphMutation mutation) -> {
                    String nodeId = mutation.getNodeId();
                    return (nodeId != null ? nodeId : "UNKNOWN_NODE").getBytes();
                })
                .setValueSerializationSchema(new HybridGraphMutationSerializer())
                .build())
            .setKafkaProducerConfig(KafkaConfigLoader.getProducerConfigForSink())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module6-hybrid-graph")
            .build();
    }
    */

    private static KafkaSink<AuditLogEntry> createHybridAuditSink() {
        return KafkaSink.<AuditLogEntry>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.EHR_AUDIT_LOGS.getTopicName())
                .setKeySerializationSchema((AuditLogEntry audit) -> {
                    String eventId = audit.getEventId();
                    return (eventId != null ? eventId : "UNKNOWN_EVENT").getBytes();
                })
                .setValueSerializationSchema(new HybridAuditLogSerializer())
                .build())
            .setKafkaProducerConfig(KafkaConfigLoader.getProducerConfigForSink())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module6-hybrid-audit")
            .build();
    }

    // ===== Hybrid Serializer Classes =====

    private static class HybridEnrichedEventSerializer implements SerializationSchema<EnrichedClinicalEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(EnrichedClinicalEvent element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                LOG.error("Failed to serialize EnrichedClinicalEvent", e);
                throw new RuntimeException(e);
            }
        }
    }

    private static class HybridCriticalAlertSerializer implements SerializationSchema<CriticalAlert> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(CriticalAlert element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                LOG.error("Failed to serialize CriticalAlert", e);
                throw new RuntimeException(e);
            }
        }
    }

    private static class HybridFHIRResourceSerializer implements SerializationSchema<FHIRResource> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(FHIRResource element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                LOG.error("Failed to serialize FHIRResource", e);
                throw new RuntimeException(e);
            }
        }
    }

    private static class HybridAnalyticsEventSerializer implements SerializationSchema<AnalyticsEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(AnalyticsEvent element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                LOG.error("Failed to serialize AnalyticsEvent", e);
                throw new RuntimeException(e);
            }
        }
    }

    private static class HybridGraphMutationSerializer implements SerializationSchema<GraphMutation> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(GraphMutation element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                LOG.error("Failed to serialize GraphMutation", e);
                throw new RuntimeException(e);
            }
        }
    }

    private static class HybridAuditLogSerializer implements SerializationSchema<AuditLogEntry> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(AuditLogEntry element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                LOG.error("Failed to serialize AuditLogEntry", e);
                throw new RuntimeException(e);
            }
        }
    }
}
