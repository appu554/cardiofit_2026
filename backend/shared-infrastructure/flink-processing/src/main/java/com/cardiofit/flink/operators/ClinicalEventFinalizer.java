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
 * Pass-Through Operator with Logging
 * -----------------------------------
 * This stateless operator emits EnrichedPatientContext directly to downstream operators.
 * Phase 5 implementation is simplified to avoid CanonicalEvent conversion complexity.
 *
 * The EnrichedPatientContext already contains all necessary clinical intelligence:
 * - Complete patient state (vitals, labs, meds)
 * - All generated alerts (from aggregator + intelligence evaluator)
 * - Clinical scores (NEWS2, qSOFA, combined acuity)
 * - Risk indicators
 *
 * Downstream operators (Neo4j sink, Elasticsearch sink, etc.) can consume
 * EnrichedPatientContext directly or implement their own converters.
 *
 * Input: EnrichedPatientContext (from ClinicalIntelligenceEvaluator)
 * Output: EnrichedPatientContext (pass-through)
 *
 * @author CardioFit Platform - Module 2 Enhancement
 * @version 2.0
 * @since 2025-01-15
 */
public class ClinicalEventFinalizer extends ProcessFunction<EnrichedPatientContext, EnrichedPatientContext> {

    private static final Logger logger = LoggerFactory.getLogger(ClinicalEventFinalizer.class);
    private static final long serialVersionUID = 1L;

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

        // Pass through enriched context as-is
        // All clinical intelligence has been computed by upstream operators:
        // - PatientContextAggregator: state management, basic clinical rules
        // - ClinicalIntelligenceEvaluator: advanced pattern detection (sepsis, ACS, MODS)

        out.collect(enrichedContext);

        // Log summary for monitoring
        int alertCount = state.getActiveAlerts() != null ? state.getActiveAlerts().size() : 0;
        Double acuityScore = state.getCombinedAcuityScore();

        logger.debug("Finalized enriched context for patient {}: eventType={}, alerts={}, acuity={}",
                patientId, enrichedContext.getEventType(), alertCount, acuityScore);
    }

}
