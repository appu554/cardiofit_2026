
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.state.ListState;
import org.apache.flink.api.common.state.ListStateDescriptor;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.connector.base.DeliveryGuarantee;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.runtime.state.FunctionInitializationContext;
import org.apache.flink.runtime.state.FunctionSnapshotContext;
import org.apache.flink.streaming.api.checkpoint.CheckpointedFunction;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * Transactional Multi-Sink Router for Hybrid Kafka Topic Architecture
 *
 * This is the heart of the EHR Intelligence Engine's data distribution strategy.
 * It implements the recommended hybrid architecture by:
 *
 * 1. Writing ALL enriched events to the central system of record topic
 * 2. Intelligently routing events to purpose-built action topics
 * 3. Ensuring EXACTLY_ONCE semantics with transactional guarantees
 * 4. Transforming events to match target topic schemas
 *
 * Architecture Benefits:
 * - Central source of truth for audit, replay, and governance
 * - Operational isolation for critical systems (alerts, FHIR)
 * - Atomic writes prevent partial state scenarios
 * - Schema transformation at the routing boundary
 */
public class TransactionalMultiSinkRouter extends ProcessFunction<EnrichedClinicalEvent, Void>
    implements CheckpointedFunction {

    private static final Logger LOG = LoggerFactory.getLogger(TransactionalMultiSinkRouter.class);

    // Side output tags for routing to different topics
    public static final OutputTag<EnrichedClinicalEvent> CENTRAL_OUTPUT =
        new OutputTag<EnrichedClinicalEvent>("central-topic") {};
    public static final OutputTag<CriticalAlert> ALERTS_OUTPUT =
        new OutputTag<CriticalAlert>("alerts-topic") {};
    public static final OutputTag<FHIRResource> FHIR_OUTPUT =
        new OutputTag<FHIRResource>("fhir-topic") {};
    public static final OutputTag<AnalyticsEvent> ANALYTICS_OUTPUT =
        new OutputTag<AnalyticsEvent>("analytics-topic") {};
    public static final OutputTag<GraphMutation> GRAPH_OUTPUT =
        new OutputTag<GraphMutation>("graph-topic") {};
    public static final OutputTag<AuditLogEntry> AUDIT_OUTPUT =
        new OutputTag<AuditLogEntry>("audit-topic") {};

    // State for checkpoint coordination
    private transient ListState<String> pendingTransactions;

    // Kafka Configuration (sinks are wired externally in Module6_EgressRouting)
    private String kafkaBootstrapServers;

    // Configuration (transient for Flink serialization)
    private transient ObjectMapper objectMapper;

    public TransactionalMultiSinkRouter() {
        // ObjectMapper initialization moved to open() for serialization compatibility
    }

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        // Initialize ObjectMapper for serialization (transient field)
        this.objectMapper = new ObjectMapper();
        this.objectMapper.registerModule(new JavaTimeModule());

        // Initialize Kafka bootstrap servers based on environment
        this.kafkaBootstrapServers = KafkaConfigLoader.isRunningInDocker()
            ? "kafka1:29092,kafka2:29093,kafka3:29094"
            : "localhost:9092,localhost:9093,localhost:9094";

        // NOTE: Kafka sinks are NOT initialized here - they are created in Module6_EgressRouting
        // and wired to the side outputs. This ProcessFunction only writes to side outputs.

        LOG.info("🔌 TransactionalMultiSinkRouter initialized - side outputs only (sinks wired externally)");
    }

    @Override
    public void processElement(
        EnrichedClinicalEvent enrichedEvent,
        Context ctx,
        Collector<Void> out
    ) throws Exception {

        long startTime = System.currentTimeMillis();

        // Phase 1: ALWAYS write to central system of record
        writeToCentralTopic(enrichedEvent, ctx);

        // Phase 2: Intelligent routing to action topics
        RouteDecision decision = determineRouting(enrichedEvent);

        if (decision.shouldAlert()) {
            writeToAlertsTopic(enrichedEvent, ctx);
        }

        if (decision.shouldPersistFHIR()) {
            writeToFHIRTopic(enrichedEvent, ctx);
        }

        // Phase 3: Supporting systems (analytics, graph)
        if (decision.shouldAnalyze()) {
            writeToAnalyticsTopic(enrichedEvent, ctx);
        }

        if (decision.shouldUpdateGraph()) {
            writeToGraphTopic(enrichedEvent, ctx);
        }

        // Always audit for compliance
        writeToAuditTopic(enrichedEvent, ctx);

        long processingTime = System.currentTimeMillis() - startTime;
        LOG.debug("🔄 Routed event {} in {}ms to {} destinations",
            enrichedEvent.getId(), processingTime, decision.getDestinationCount());
    }

    /**
     * Core routing intelligence - determines which action topics should receive the event
     */
    private RouteDecision determineRouting(EnrichedClinicalEvent event) {
        RouteDecision decision = new RouteDecision();

        // Critical alerts routing
        if (isCriticalEvent(event)) {
            decision.setAlert(true);
            LOG.debug("🚨 Critical event detected: {}", event.getId());
        }

        // FHIR persistence routing
        if (shouldPersistToFHIR(event)) {
            decision.setPersistFHIR(true);
        }

        // Analytics routing
        if (hasAnalyticalValue(event)) {
            decision.setAnalyze(true);
        }

        // Graph database routing
        if (hasGraphImplications(event)) {
            decision.setUpdateGraph(true);
        }

        return decision;
    }

    /**
     * Critical event detection logic
     */
    private boolean isCriticalEvent(EnrichedClinicalEvent event) {
        // High clinical significance - using primitive double comparison
        if (event.getClinicalSignificance() > 0.8) {
            return true;
        }

        // Drug interactions - check if drug interactions list exists and is not empty
        if (event.getDrugInteractions() != null && !event.getDrugInteractions().isEmpty()) {
            return true;
        }

        // High-risk ML predictions
        if (event.getMlPredictions() != null) {
            return event.getMlPredictions().stream()
                .anyMatch(pred -> pred.getRiskLevel() != null &&
                         pred.getRiskLevel().equals("HIGH"));
        }

        // Pattern-based alerts - check for critical patterns in string list
        if (event.getDetectedPatterns() != null) {
            return event.getDetectedPatterns().stream()
                .anyMatch(pattern -> pattern != null &&
                         (pattern.contains("CRITICAL") || pattern.contains("URGENT") || pattern.contains("EMERGENCY")));
        }

        return false;
    }

    /**
     * FHIR persistence decision logic
     *
     * CRITICAL FIX: Updated to handle BOTH primary clinical events AND derived analytical events
     *
     * Primary Clinical Events (from Module 1-3):
     * - EventType enum values: PATIENT_ADMISSION, VITAL_SIGN, LAB_RESULT, MEDICATION_ORDERED, etc.
     * - These match contains("PATIENT"), contains("MEDICATION"), contains("OBSERVATION")
     *
     * Derived Analytical Events (from Module 4-5):
     * - PATTERN_EVENT: Clinical patterns detected by CEP engine (Module 4)
     * - ML_PREDICTION: Risk predictions from ML models (Module 5)
     * - These are important for clinical decision support and should also persist to FHIR
     */
    private boolean shouldPersistToFHIR(EnrichedClinicalEvent event) {
        if (event.getSourceEventType() == null) {
            return false;
        }

        String eventType = event.getSourceEventType();

        // Primary clinical events (from EventType enum)
        boolean isPrimaryClinical = eventType.contains("CLINICAL") ||
                                     eventType.contains("PATIENT") ||
                                     eventType.contains("MEDICATION") ||
                                     eventType.contains("OBSERVATION") ||
                                     eventType.contains("VITAL") ||
                                     eventType.contains("LAB") ||
                                     eventType.contains("DIAGNOSTIC") ||
                                     eventType.contains("PROCEDURE") ||
                                     eventType.contains("ENCOUNTER") ||
                                     eventType.contains("ADVERSE") ||
                                     eventType.contains("ALLERGY") ||
                                     eventType.contains("DRUG_INTERACTION") ||
                                     eventType.contains("CONTRAINDICATION");

        // Derived analytical events that should persist for clinical decision support
        boolean isDerivedClinical = eventType.equals("PATTERN_EVENT") ||
                                    eventType.equals("ML_PREDICTION");

        return isPrimaryClinical || isDerivedClinical;
    }

    /**
     * Analytics value assessment
     */
    private boolean hasAnalyticalValue(EnrichedClinicalEvent event) {
        // Events with ML predictions
        if (event.getMlPredictions() != null && !event.getMlPredictions().isEmpty()) {
            return true;
        }

        // Pattern detection results
        if (event.getDetectedPatterns() != null && !event.getDetectedPatterns().isEmpty()) {
            return true;
        }

        // Clinical significance above threshold - using primitive double comparison
        return event.getClinicalSignificance() > 0.3;
    }

    /**
     * Graph database implications
     */
    private boolean hasGraphImplications(EnrichedClinicalEvent event) {
        // Patient relationship changes
        if (event.hasPatientRelationshipChanges()) {
            return true;
        }

        // Clinical concept relationships
        if (event.hasClinicalConceptRelationships()) {
            return true;
        }

        // Drug interaction networks
        return event.hasDrugInteractions();
    }

    // ===== Sink Writing Methods =====

    private void writeToCentralTopic(EnrichedClinicalEvent event, Context ctx) throws Exception {
        // The enriched event goes directly to central topic as-is
        // This maintains the complete, canonical representation
        // Key: patientId for proper partitioning
        ctx.output(CENTRAL_OUTPUT, event);
    }

    private void writeToAlertsTopic(EnrichedClinicalEvent event, Context ctx) throws Exception {
        CriticalAlert alert = transformToCriticalAlert(event);
        ctx.output(ALERTS_OUTPUT, alert);
    }

    private void writeToFHIRTopic(EnrichedClinicalEvent event, Context ctx) throws Exception {
        FHIRResource fhirResource = transformToFHIRResource(event);
        ctx.output(FHIR_OUTPUT, fhirResource);
    }

    private void writeToAnalyticsTopic(EnrichedClinicalEvent event, Context ctx) throws Exception {
        AnalyticsEvent analyticsEvent = transformToAnalyticsEvent(event);
        ctx.output(ANALYTICS_OUTPUT, analyticsEvent);
    }

    private void writeToGraphTopic(EnrichedClinicalEvent event, Context ctx) throws Exception {
        List<GraphMutation> mutations = transformToGraphMutations(event);
        for (GraphMutation mutation : mutations) {
            ctx.output(GRAPH_OUTPUT, mutation);
        }
    }

    private void writeToAuditTopic(EnrichedClinicalEvent event, Context ctx) throws Exception {
        AuditLogEntry auditEntry = transformToAuditEntry(event);
        ctx.output(AUDIT_OUTPUT, auditEntry);
    }

    // ===== Transformation Methods =====

    private CriticalAlert transformToCriticalAlert(EnrichedClinicalEvent event) {
        CriticalAlert alert = new CriticalAlert();
        alert.setId(UUID.randomUUID().toString());
        alert.setPatientId(event.getPatientId());
        alert.setAlertType(determineCriticalAlertType(event));
        alert.setSeverity(determineSeverity(event));
        alert.setMessage(generateAlertMessage(event));
        alert.setTimestamp(System.currentTimeMillis());
        alert.setSourceEventId(event.getId());
        alert.setRequiresImmedateAction(true);

        // Include relevant clinical context
        if (event.getDrugInteractions() != null) {
            // Convert SemanticEvent.DrugInteraction to DrugInteraction
            List<DrugInteraction> drugInteractions = new ArrayList<>();
            for (SemanticEvent.DrugInteraction semanticDrugInteraction : event.getDrugInteractions()) {
                DrugInteraction drugInteraction = new DrugInteraction();
                // Map drug1 and drug2 to medicationIds list
                List<String> medicationIds = new ArrayList<>();
                if (semanticDrugInteraction.getDrug1() != null) medicationIds.add(semanticDrugInteraction.getDrug1());
                if (semanticDrugInteraction.getDrug2() != null) medicationIds.add(semanticDrugInteraction.getDrug2());
                drugInteraction.setMedicationIds(medicationIds);
                drugInteraction.setSeverity(semanticDrugInteraction.getSeverity());
                drugInteraction.setDescription(semanticDrugInteraction.getRecommendation()); // Use recommendation as description
                drugInteractions.add(drugInteraction);
            }
            alert.setDrugInteractions(drugInteractions);
        }

        return alert;
    }

    private FHIRResource transformToFHIRResource(EnrichedClinicalEvent event) {
        FHIRResource resource = new FHIRResource();
        resource.setResourceType(determineFHIRResourceType(event));
        resource.setResourceId(generateFHIRResourceId(event));
        resource.setPatientId(event.getPatientId());
        resource.setLastUpdated(System.currentTimeMillis());
        resource.setVersion("1");

        // Transform clinical data to FHIR format
        Map<String, Object> fhirData = new HashMap<>();
        fhirData.put("resourceType", resource.getResourceType());
        fhirData.put("id", resource.getResourceId());
        fhirData.put("subject", Map.of("reference", "Patient/" + event.getPatientId()));
        // Convert LocalDateTime to milliseconds for Date constructor
        long timestampMillis = event.getTimestamp().atZone(java.time.ZoneId.systemDefault()).toInstant().toEpochMilli();
        fhirData.put("effectiveDateTime", new Date(timestampMillis));

        // Add clinical content based on event type
        addClinicalContentToFHIR(event, fhirData);

        resource.setFhirData(fhirData);
        return resource;
    }

    private AnalyticsEvent transformToAnalyticsEvent(EnrichedClinicalEvent event) {
        AnalyticsEvent analyticsEvent = new AnalyticsEvent();
        analyticsEvent.setEventId(event.getId());
        analyticsEvent.setPatientId(event.getPatientId());
        analyticsEvent.setEventType(event.getSourceEventType());
        // Convert LocalDateTime to epoch milliseconds for Long timestamp
        long timestampMillis = event.getTimestamp().atZone(java.time.ZoneId.systemDefault()).toInstant().toEpochMilli();
        analyticsEvent.setTimestamp(timestampMillis);
        analyticsEvent.setClinicalSignificance(event.getClinicalSignificance());

        // Flatten for analytics consumption
        Map<String, Object> metrics = new HashMap<>();

        if (event.getMlPredictions() != null) {
            metrics.put("ml_prediction_count", event.getMlPredictions().size());
            event.getMlPredictions().forEach(pred -> {
                metrics.put("ml_" + pred.getModelType() + "_confidence", pred.getConfidence());
                metrics.put("ml_" + pred.getModelType() + "_risk", pred.getRiskLevel());
            });
        }

        if (event.getDetectedPatterns() != null) {
            metrics.put("pattern_count", event.getDetectedPatterns().size());
            // getDetectedPatterns() returns List<String>, so we treat them as pattern names
            for (int i = 0; i < event.getDetectedPatterns().size(); i++) {
                metrics.put("pattern_" + i + "_name", event.getDetectedPatterns().get(i));
            }
        }

        analyticsEvent.setMetrics(metrics);
        return analyticsEvent;
    }

    private List<GraphMutation> transformToGraphMutations(EnrichedClinicalEvent event) {
        List<GraphMutation> mutations = new ArrayList<>();

        // Patient node update
        GraphMutation patientUpdate = new GraphMutation();
        patientUpdate.setMutationType("MERGE");
        patientUpdate.setNodeType("Patient");
        patientUpdate.setNodeId(event.getPatientId());
        patientUpdate.setProperties(Map.of(
            "lastUpdated", System.currentTimeMillis(),
            "eventCount", "+1"  // Increment counter
        ));
        mutations.add(patientUpdate);

        // Event node creation
        GraphMutation eventNode = new GraphMutation();
        eventNode.setMutationType("CREATE");
        eventNode.setNodeType("ClinicalEvent");
        eventNode.setNodeId(event.getId());
        eventNode.setProperties(Map.of(
            "eventType", event.getSourceEventType(),
            "timestamp", event.getTimestamp(),
            "significance", event.getClinicalSignificance()
        ));
        mutations.add(eventNode);

        // Relationship between patient and event
        GraphMutation relationship = new GraphMutation();
        relationship.setMutationType("CREATE");
        relationship.setRelationshipType("HAS_EVENT");
        relationship.setFromNodeId(event.getPatientId());
        relationship.setToNodeId(event.getId());
        relationship.setProperties(Map.of("createdAt", System.currentTimeMillis()));
        mutations.add(relationship);

        return mutations;
    }

    private AuditLogEntry transformToAuditEntry(EnrichedClinicalEvent event) {
        AuditLogEntry audit = new AuditLogEntry();
        audit.setId(UUID.randomUUID().toString());
        audit.setEventId(event.getId());
        audit.setPatientId(event.getPatientId());
        audit.setTimestamp(System.currentTimeMillis());
        audit.setEventType("CLINICAL_EVENT_PROCESSED");
        audit.setSource("TransactionalMultiSinkRouter");
        audit.setDetails(Map.of(
            "sourceEventType", event.getSourceEventType(),
            "clinicalSignificance", event.getClinicalSignificance(),
            "processingTimestamp", System.currentTimeMillis()
        ));
        return audit;
    }

    // ===== Helper Methods =====

    private String determineCriticalAlertType(EnrichedClinicalEvent event) {
        if (event.hasDrugInteractions()) {
            return "DRUG_INTERACTION";
        }
        if (event.getClinicalSignificance() > 0.9) {
            return "HIGH_CLINICAL_SIGNIFICANCE";
        }
        return "CRITICAL_PATTERN_DETECTED";
    }

    private String determineSeverity(EnrichedClinicalEvent event) {
        if (event.getClinicalSignificance() > 0.9) {
            return "CRITICAL";
        } else if (event.getClinicalSignificance() > 0.7) {
            return "HIGH";
        }
        return "MEDIUM";
    }

    private String generateAlertMessage(EnrichedClinicalEvent event) {
        StringBuilder message = new StringBuilder();
        message.append("Critical clinical event detected for patient ").append(event.getPatientId());

        if (event.hasDrugInteractions()) {
            message.append(" - Drug interaction alert");
        }

        if (event.getClinicalSignificance() > 0.8) {
            message.append(" - High clinical significance (")
                   .append(String.format("%.2f", event.getClinicalSignificance()))
                   .append(")");
        }

        return message.toString();
    }

    private String determineFHIRResourceType(EnrichedClinicalEvent event) {
        String eventType = event.getSourceEventType();
        if (eventType.contains("MEDICATION")) {
            return "MedicationStatement";
        } else if (eventType.contains("OBSERVATION")) {
            return "Observation";
        } else if (eventType.contains("PATTERN")) {
            return "DiagnosticReport";
        } else if (eventType.contains("PREDICTION")) {
            return "RiskAssessment";
        }
        return "Basic";
    }

    private String generateFHIRResourceId(EnrichedClinicalEvent event) {
        return event.getSourceEventType() + "-" + event.getId();
    }

    private void addClinicalContentToFHIR(EnrichedClinicalEvent event, Map<String, Object> fhirData) {
        // Add specific FHIR content based on event type
        // This is simplified - in practice would need full FHIR R4 compliance
        fhirData.put("status", "final");
        fhirData.put("category", List.of(Map.of(
            "coding", List.of(Map.of(
                "system", "http://cardiofit.com/clinical-events",
                "code", event.getSourceEventType(),
                "display", event.getSourceEventType()
            ))
        )));

        // Always add clinical significance since it's a primitive double
        fhirData.put("valueQuantity", Map.of(
            "value", event.getClinicalSignificance(),
            "unit", "significance-score",
            "system", "http://cardiofit.com/units"
        ));
    }

    // ===== Sink Initialization =====

    /**
     * DEPRECATED: These sink initialization methods are NO LONGER USED.
     *
     * TransactionalMultiSinkRouter now uses side outputs only.
     * Actual Kafka sinks are created and wired to side outputs in Module6_EgressRouting.java.
     *
     * Keeping these methods commented out for reference, but they should not be called.
     * Previous architecture had duplicate transactional producers which caused Kafka timeouts.
     */

    /*
    private void initializeCentralSink() {
        // UNUSED - Sinks now created in Module6_EgressRouting
    }

    private void initializeCriticalActionSinks() {
        // UNUSED - Sinks now created in Module6_EgressRouting
    }

    private void initializeSupportingSinks() {
        // UNUSED - Sinks now created in Module6_EgressRouting
    }
    */

    // ===== Checkpointing =====

    @Override
    public void snapshotState(FunctionSnapshotContext context) throws Exception {
        // Checkpoint any pending transaction state
        LOG.debug("📸 Snapshotting transactional state for checkpoint {}", context.getCheckpointId());
    }

    @Override
    public void initializeState(FunctionInitializationContext context) throws Exception {
        pendingTransactions = context.getOperatorStateStore()
            .getListState(new ListStateDescriptor<>("pending-transactions", String.class));
        LOG.debug("🔄 Initialized transactional state from checkpoint");
    }

    // ===== Inner Classes =====

    /**
     * Routing decision container
     */
    private static class RouteDecision {
        private boolean alert = false;
        private boolean persistFHIR = false;
        private boolean analyze = false;
        private boolean updateGraph = false;

        public boolean shouldAlert() { return alert; }
        public boolean shouldPersistFHIR() { return persistFHIR; }
        public boolean shouldAnalyze() { return analyze; }
        public boolean shouldUpdateGraph() { return updateGraph; }

        public void setAlert(boolean alert) { this.alert = alert; }
        public void setPersistFHIR(boolean persistFHIR) { this.persistFHIR = persistFHIR; }
        public void setAnalyze(boolean analyze) { this.analyze = analyze; }
        public void setUpdateGraph(boolean updateGraph) { this.updateGraph = updateGraph; }

        public int getDestinationCount() {
            int count = 1; // Always central topic
            if (alert) count++;
            if (persistFHIR) count++;
            if (analyze) count++;
            if (updateGraph) count++;
            return count + 1; // +1 for audit
        }
    }

    // ===== Serializers =====

    private class EnrichedEventSerializer implements SerializationSchema<EnrichedClinicalEvent> {
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

    private class CriticalAlertSerializer implements SerializationSchema<CriticalAlert> {
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

    private class FHIRResourceSerializer implements SerializationSchema<FHIRResource> {
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

    private class AnalyticsEventSerializer implements SerializationSchema<AnalyticsEvent> {
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

    private class GraphMutationSerializer implements SerializationSchema<GraphMutation> {
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

    private class AuditLogSerializer implements SerializationSchema<AuditLogEntry> {
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
