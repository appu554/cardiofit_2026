package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Instant;
import java.util.*;
import java.util.stream.Collectors;

/**
 * ClinicalEventFinalizer - Phase 5 of Unified Clinical Reasoning Pipeline
 *
 * Enrichment Metadata Stamping Operator
 * -------------------------------------
 * Stamps enrichment metadata onto each EnrichedPatientContext before it reaches
 * the Kafka sink. Downstream consumers (Module 3, Module 4 CEP) can inspect the
 * enrichmentData map to determine which data sources contributed to the event:
 *   - has_fhir_data: whether FHIR enrichment was applied
 *   - has_neo4j_data: whether Neo4j graph enrichment was applied
 *   - enrichment_complete: whether the full enrichment pipeline completed
 *   - enrichment_status: FULL / PARTIAL / NONE (convenience field)
 *   - enrichment_timestamp: when finalization occurred (epoch millis)
 *   - pipeline_version: version tag for the producing pipeline
 *
 * This is especially important when circuit breakers open and enrichment is partial.
 *
 * Input: EnrichedPatientContext (from ClinicalIntelligenceEvaluator)
 * Output: EnrichedPatientContext (with enrichment metadata stamped)
 *
 * @author CardioFit Platform - Module 2 Enhancement
 * @version 2.1
 * @since 2025-01-15
 */
public class ClinicalEventFinalizer extends ProcessFunction<EnrichedPatientContext, EnrichedPatientContext> {

    private static final Logger logger = LoggerFactory.getLogger(ClinicalEventFinalizer.class);
    private static final long serialVersionUID = 1L;
    private static final String PIPELINE_VERSION = "module2-unified-v1";

    @Override
    public void processElement(
            EnrichedPatientContext enrichedContext,
            Context ctx,
            Collector<EnrichedPatientContext> out) throws Exception {

        PatientContextState state = enrichedContext.getPatientState();
        if (state == null) {
            logger.warn("Received EnrichedPatientContext with null state for patient {}",
                    enrichedContext.getPatientId());
            return;
        }

        String patientId = enrichedContext.getPatientId();
        logger.debug("Finalizing enriched context for patient {}", patientId);

        // --- Enrichment metadata stamping ---
        // Stamp source-attribution metadata so downstream consumers (Module 3,
        // Module 4 CEP) know which enrichment sources contributed to this event.
        Map<String, Object> enrichmentMeta = enrichedContext.getEnrichmentData();
        if (enrichmentMeta == null) {
            enrichmentMeta = new HashMap<>();
        }

        enrichmentMeta.put("enrichment_timestamp", System.currentTimeMillis());
        enrichmentMeta.put("pipeline_version", PIPELINE_VERSION);
        enrichmentMeta.put("has_fhir_data", state.isHasFhirData());
        enrichmentMeta.put("has_neo4j_data", state.isHasNeo4jData());
        enrichmentMeta.put("enrichment_complete", state.isEnrichmentComplete());

        // Derive a convenience status for quick downstream filtering
        String enrichmentStatus;
        if (state.isEnrichmentComplete()) {
            enrichmentStatus = "FULL";
        } else if (state.isHasFhirData() || state.isHasNeo4jData()) {
            enrichmentStatus = "PARTIAL";
        } else {
            enrichmentStatus = "NONE";
        }
        enrichmentMeta.put("enrichment_status", enrichmentStatus);

        enrichedContext.setEnrichmentData(enrichmentMeta);

        out.collect(enrichedContext);

        // Log summary for monitoring
        int alertCount = state.getActiveAlerts() != null ? state.getActiveAlerts().size() : 0;
        Double acuityScore = state.getCombinedAcuityScore();

        logger.debug("Finalized enriched context for patient {}: eventType={}, alerts={}, acuity={}, enrichment={}",
                patientId, enrichedContext.getEventType(), alertCount, acuityScore, enrichmentStatus);
    }

}
