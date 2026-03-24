package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.List;

/**
 * Option C: Single Transactional Sink Router
 *
 * ARCHITECTURAL CHANGE:
 * - Old: ProcessFunction<EnrichedClinicalEvent, Void> with 6 side outputs
 * - New: ProcessFunction<EnrichedClinicalEvent, RoutedEnrichedEvent> with single output
 *
 * This eliminates the 6 competing transactional Kafka sinks problem by:
 * 1. Making routing decisions (which destinations)
 * 2. Wrapping event with routing metadata
 * 3. Outputting to SINGLE transactional sink → prod.ehr.events.enriched.routing
 * 4. Idempotent router jobs downstream filter and route to final destinations
 *
 * Benefits:
 * - Single transactional producer (no coordinator contention)
 * - Independent router job scaling
 * - Fault-tolerant reprocessing (idempotent consumers)
 * - Maintains EXACTLY_ONCE end-to-end
 */
public class TransactionalMultiSinkRouterV2_OptionC extends ProcessFunction<EnrichedClinicalEvent, RoutedEnrichedEvent> {

    private static final Logger LOG = LoggerFactory.getLogger(TransactionalMultiSinkRouterV2_OptionC.class);

    private long eventCount = 0;

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);
        LOG.info("🚀 Option C Router initialized - single output with routing metadata");
    }

    @Override
    public void processElement(
        EnrichedClinicalEvent enrichedEvent,
        Context ctx,
        Collector<RoutedEnrichedEvent> out
    ) throws Exception {

        long startTime = System.currentTimeMillis();
        eventCount++;

        // Make routing decisions using same logic as V1
        RoutingDecision routing = determineRouting(enrichedEvent);

        // Create wrapped event with routing metadata
        RoutedEnrichedEvent routedEvent = new RoutedEnrichedEvent(enrichedEvent, routing);
        routedEvent.setRoutingSource("TransactionalMultiSinkRouterV2_OptionC");

        // Single output to central routing topic
        out.collect(routedEvent);

        long processingTime = System.currentTimeMillis() - startTime;

        if (eventCount % 1000 == 0) {
            LOG.info("📊 Routed {} events | Latest: eventId={}, destinations={}, latency={}ms",
                eventCount, enrichedEvent.getId(), routing.getDestinationCount(), processingTime);
        }

        LOG.debug("🔄 Routed event {} in {}ms → {} destinations: {}",
            enrichedEvent.getId(), processingTime, routing.getDestinationCount(), routing);
    }

    /**
     * Core routing intelligence - determines which destinations should receive the event.
     * SAME LOGIC as V1, but returns RoutingDecision instead of writing to side outputs.
     */
    private RoutingDecision determineRouting(EnrichedClinicalEvent event) {
        RoutingDecision decision = new RoutingDecision();

        // Critical alerts routing
        if (isCriticalEvent(event)) {
            decision.setSendToCriticalAlerts(true);
            decision.addMetadata("alert_reason", "high_significance_or_drug_interaction");
            LOG.debug("🚨 Critical event detected: {}", event.getId());
        }

        // FHIR persistence routing
        if (shouldPersistToFHIR(event)) {
            decision.setSendToFHIR(true);
            decision.addMetadata("fhir_resource_type", event.getFhirResourceType());
        }

        // Analytics routing
        if (hasAnalyticalValue(event)) {
            decision.setSendToAnalytics(true);
        }

        // Graph database routing
        if (hasGraphImplications(event)) {
            decision.setSendToGraph(true);
            decision.addMetadata("graph_operation", "relationship_update");
        }

        // Always audit for compliance
        decision.setSendToAudit(true);
        decision.addMetadata("audit_category", event.getEventType());

        return decision;
    }

    /**
     * Critical event detection logic (same as V1, extended for ingestion events)
     */
    private boolean isCriticalEvent(EnrichedClinicalEvent event) {
        // Ingestion service events: check for CRITICAL_VALUE flag in payload
        if ("ingestion-service".equals(event.getSourceSystem())) {
            Object flags = event.getPayload() != null ? event.getPayload().get("flags") : null;
            if (flags instanceof List && ((List<?>) flags).contains("CRITICAL_VALUE")) {
                LOG.debug("Critical ingestion event detected via CRITICAL_VALUE flag: {}", event.getId());
                return true;
            }
            // Ingestion events without CRITICAL_VALUE flag are not critical
            // (standard clinical significance checks below may still apply)
        }

        // High clinical significance
        if (event.getClinicalSignificance() > 0.8) {
            return true;
        }

        // Drug interactions
        if (event.getDrugInteractions() != null && !event.getDrugInteractions().isEmpty()) {
            return true;
        }

        // High-risk ML predictions
        if (event.getMlPredictions() != null) {
            return event.getMlPredictions().stream()
                .anyMatch(pred -> pred.getRiskLevel() != null &&
                         pred.getRiskLevel().equals("HIGH"));
        }

        // Pattern-based alerts
        if (event.getDetectedPatterns() != null) {
            return event.getDetectedPatterns().stream()
                .anyMatch(pattern -> pattern != null &&
                         (pattern.contains("CRITICAL") || pattern.contains("URGENT") || pattern.contains("EMERGENCY")));
        }

        return false;
    }

    /**
     * FHIR persistence decision logic (same as V1, with ingestion-service skip)
     *
     * Ingestion service events already have raw Observations persisted to FHIR
     * at ingestion Stage 3, so we skip duplicate FHIR writes here.
     */
    private boolean shouldPersistToFHIR(EnrichedClinicalEvent event) {
        // Ingestion service events: FHIR resource already persisted at Stage 3
        if ("ingestion-service".equals(event.getSourceSystem())) {
            LOG.debug("Skipping FHIR persistence for ingestion-service event: {}", event.getId());
            return false;
        }

        if (event.getSourceEventType() == null) {
            return false;
        }

        String eventType = event.getSourceEventType();

        // Primary clinical events
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

        // Derived analytical events
        boolean isDerivedClinical = eventType.equals("PATTERN_EVENT") ||
                                    eventType.equals("ML_PREDICTION");

        return isPrimaryClinical || isDerivedClinical;
    }

    /**
     * Analytics value assessment (same as V1)
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

        // Clinical significance above threshold
        return event.getClinicalSignificance() > 0.3;
    }

    /**
     * Graph database implications (same as V1)
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
}
