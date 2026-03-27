package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.protocol.SimplifiedProtocol;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * Stateless phase executor for Module 3 CDS.
 * Each phase is a static method that takes patient context + knowledge base data
 * and returns a CDSPhaseResult. Extracted from the operator for testability.
 */
public class Module3PhaseExecutor {
    private static final Logger LOG = LoggerFactory.getLogger(Module3PhaseExecutor.class);

    /**
     * Phase 1: Protocol Matching.
     * Matches patient vitals/scores against SimplifiedProtocol triggerThresholds.
     * Returns matched protocol IDs ranked by confidence.
     */
    public static CDSPhaseResult executePhase1(
            EnrichedPatientContext context,
            Map<String, SimplifiedProtocol> protocols) {

        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_1_PROTOCOL_MATCH");

        if (protocols == null || protocols.isEmpty()) {
            result.setActive(false);
            result.addDetail("matchedCount", 0);
            result.addDetail("protocolSource", "BROADCAST_STATE");
            result.setDurationMs((System.nanoTime() - start) / 1_000_000);
            return result;
        }

        PatientContextState state = context.getPatientState();
        Map<String, Object> vitals = (state != null) ? state.getLatestVitals() : Collections.emptyMap();

        List<String> matchedIds = new ArrayList<>();
        List<Map<String, Object>> matchDetails = new ArrayList<>();

        for (SimplifiedProtocol protocol : protocols.values()) {
            double matchScore = evaluateProtocolMatch(protocol, state, vitals);
            if (matchScore >= protocol.getActivationThreshold()) {
                matchedIds.add(protocol.getProtocolId());
                Map<String, Object> detail = new HashMap<>();
                detail.put("protocolId", protocol.getProtocolId());
                detail.put("name", protocol.getName());
                detail.put("confidence", matchScore);
                detail.put("category", protocol.getCategory());
                matchDetails.add(detail);
            }
        }

        // Sort by confidence descending
        matchDetails.sort((a, b) -> Double.compare(
                (double) b.get("confidence"), (double) a.get("confidence")));

        result.setActive(!matchedIds.isEmpty());
        result.addDetail("matchedCount", matchedIds.size());
        result.addDetail("matchedProtocolIds", matchedIds);
        result.addDetail("matchDetails", matchDetails);
        result.addDetail("protocolSource", "BROADCAST_STATE");
        result.addDetail("totalProtocolsEvaluated", protocols.size());
        result.setDurationMs((System.nanoTime() - start) / 1_000_000);

        LOG.debug("Phase 1: patient={} matched {}/{} protocols",
                context.getPatientId(), matchedIds.size(), protocols.size());

        return result;
    }

    /**
     * Evaluate how well a patient matches a protocol's trigger thresholds.
     * Returns confidence score [0.0, 1.0].
     */
    private static double evaluateProtocolMatch(
            SimplifiedProtocol protocol,
            PatientContextState state,
            Map<String, Object> vitals) {

        Map<String, Double> thresholds = protocol.getTriggerThresholds();
        if (thresholds == null || thresholds.isEmpty()) {
            return protocol.getBaseConfidence();
        }

        int totalCriteria = thresholds.size();
        int metCriteria = 0;

        for (Map.Entry<String, Double> entry : thresholds.entrySet()) {
            String param = entry.getKey();
            double threshold = entry.getValue();

            Double patientValue = extractNumericValue(param, state, vitals);
            if (patientValue != null && patientValue >= threshold) {
                metCriteria++;
            }
        }

        double matchRatio = (double) metCriteria / totalCriteria;
        return protocol.getBaseConfidence() * matchRatio;
    }

    /**
     * Extract a numeric value from patient state, checking vitals map and scores.
     * Resolution order:
     *   1. Clinical scores (qSOFA, NEWS2, combinedAcuityScore)
     *   2. Vitals map (case-insensitive lowercase key)
     *   3. Vitals map (exact case key)
     *   4. Labs map by LOINC code (exact key)
     *   5. Labs map by labType name (case-insensitive fallback)
     */
    private static Double extractNumericValue(
            String paramName, PatientContextState state, Map<String, Object> vitals) {

        // 1. Check clinical scores first
        if (state != null) {
            switch (paramName.toLowerCase()) {
                case "qsofascore":
                    return state.getQsofaScore() != null ? state.getQsofaScore().doubleValue() : null;
                case "news2score":
                    return state.getNews2Score() != null ? state.getNews2Score().doubleValue() : null;
                case "combinedacuityscore":
                    return state.getCombinedAcuityScore();
            }
        }

        // 2. Check vitals map (lowercase key — how vitals are stored)
        Object value = vitals.get(paramName.toLowerCase());
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }

        // 3. Also try exact case key in vitals
        value = vitals.get(paramName);
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }

        // 4. Check labs by LOINC code (exact key match)
        if (state != null && state.getRecentLabs() != null) {
            LabResult lab = state.getRecentLabs().get(paramName);
            if (lab != null) {
                return lab.getValue();
            }
        }

        // 5. Fallback: search labs by labType name (case-insensitive)
        //    Required because sepsis protocol threshold key is "lactate" but
        //    the patient's lab is keyed by LOINC "32693-4" with labType "Lactate".
        if (state != null && state.getRecentLabs() != null) {
            for (LabResult lab : state.getRecentLabs().values()) {
                if (lab.getLabType() != null && lab.getLabType().equalsIgnoreCase(paramName)) {
                    return lab.getValue();
                }
            }
        }

        return null;
    }
}
