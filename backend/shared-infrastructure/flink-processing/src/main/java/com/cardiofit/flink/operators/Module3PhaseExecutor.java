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

    /**
     * Phase 2: Clinical Scoring + MHRI Computation.
     */
    public static CDSPhaseResult executePhase2(EnrichedPatientContext context) {
        long start = System.nanoTime();
        CDSPhaseResult result = new CDSPhaseResult("PHASE_2_CLINICAL_SCORING");

        PatientContextState state = context.getPatientState();
        if (state == null) {
            result.setActive(false);
            result.setDurationMs((System.nanoTime() - start) / 1_000_000);
            return result;
        }

        result.setActive(true);

        // Extract Module 2 scores
        if (state.getNews2Score() != null) result.addDetail("news2Score", state.getNews2Score());
        if (state.getQsofaScore() != null) result.addDetail("qsofaScore", state.getQsofaScore());
        if (state.getCombinedAcuityScore() != null) result.addDetail("combinedAcuityScore", state.getCombinedAcuityScore());

        // CKD-EPI eGFR estimation
        Double egfr = estimateCKDEPI(state);
        if (egfr != null) result.addDetail("estimatedGFR", egfr);

        // Compute MHRI
        MHRIScore mhri = computeMHRI(context, state, egfr);
        result.addDetail("mhriScore", mhri);

        result.setDurationMs((System.nanoTime() - start) / 1_000_000);
        return result;
    }

    /**
     * CKD-EPI 2021 eGFR estimation (race-free).
     */
    private static Double estimateCKDEPI(PatientContextState state) {
        LabResult creatinineResult = state.getRecentLabs() != null ? state.getRecentLabs().get("2160-0") : null;
        if (creatinineResult == null) return null;

        PatientDemographics demo = state.getDemographics();
        if (demo == null || demo.getAge() == null || demo.getAge() <= 0) return null;

        double scr = creatinineResult.getValue();
        int age = demo.getAge();
        boolean isFemale = "female".equalsIgnoreCase(demo.getGender());

        double kappa = isFemale ? 0.7 : 0.9;
        double alpha = isFemale ? -0.241 : -0.302;
        double multiplier = isFemale ? 1.012 : 1.0;

        double scrOverKappa = scr / kappa;
        double minTerm = Math.pow(Math.min(scrOverKappa, 1.0), alpha);
        double maxTerm = Math.pow(Math.max(scrOverKappa, 1.0), -1.200);

        return 142.0 * minTerm * maxTerm * Math.pow(0.9938, age) * multiplier;
    }

    /**
     * Compute MHRI composite from patient data with piecewise linear normalization.
     */
    private static MHRIScore computeMHRI(EnrichedPatientContext context, PatientContextState state, Double egfr) {
        MHRIScore mhri = new MHRIScore();
        mhri.setDataTier(context.getDataTier() != null ? context.getDataTier() : "TIER_3_SMBG");

        mhri.setGlycemicComponent(normalizeGlycemic(state));
        mhri.setHemodynamicComponent(normalizeHemodynamic(state));
        mhri.setRenalComponent(normalizeRenal(egfr));
        mhri.setMetabolicComponent(normalizeMetabolic(state));
        mhri.setEngagementComponent(normalizeEngagement(state));

        mhri.computeComposite();
        return mhri;
    }

    /**
     * Normalize HbA1c to 0-100 risk score.
     */
    private static double normalizeGlycemic(PatientContextState state) {
        if (state.getRecentLabs() == null) return 30.0;
        LabResult hba1c = state.getRecentLabs().get("4548-4");
        if (hba1c == null) return 30.0;

        double val = hba1c.getValue();
        if (val < 5.7) return 0.0;
        if (val <= 6.4) return piecewiseLinear(val, 5.7, 6.4, 10.0, 30.0);
        if (val <= 8.0) return piecewiseLinear(val, 6.4, 8.0, 30.0, 60.0);
        if (val <= 10.0) return piecewiseLinear(val, 8.0, 10.0, 60.0, 85.0);
        return Math.min(100.0, piecewiseLinear(val, 10.0, 14.0, 85.0, 100.0));
    }

    /**
     * Normalize BP to 0-100 hemodynamic risk score.
     */
    private static double normalizeHemodynamic(PatientContextState state) {
        Object sbpObj = state.getLatestVitals().get("systolicbloodpressure");
        if (sbpObj == null) return 30.0;
        double sbp = ((Number) sbpObj).doubleValue();

        if (sbp < 120) return 0.0;
        if (sbp <= 139) return piecewiseLinear(sbp, 120, 139, 10.0, 30.0);
        if (sbp <= 159) return piecewiseLinear(sbp, 139, 159, 30.0, 60.0);
        if (sbp <= 179) return piecewiseLinear(sbp, 159, 179, 60.0, 85.0);
        return Math.min(100.0, piecewiseLinear(sbp, 179, 200, 85.0, 100.0));
    }

    /**
     * Normalize eGFR to 0-100 renal risk score.
     */
    private static double normalizeRenal(Double egfr) {
        if (egfr == null) return 20.0;
        if (egfr >= 90) return 0.0;
        if (egfr >= 60) return piecewiseLinear(egfr, 90, 60, 0.0, 30.0);
        if (egfr >= 30) return piecewiseLinear(egfr, 60, 30, 30.0, 65.0);
        if (egfr >= 15) return piecewiseLinear(egfr, 30, 15, 65.0, 85.0);
        return Math.min(100.0, piecewiseLinear(egfr, 15, 0, 85.0, 100.0));
    }

    private static double normalizeMetabolic(PatientContextState state) {
        int medCount = state.getActiveMedications() != null ? state.getActiveMedications().size() : 0;
        return Math.min(100.0, medCount * 15.0);
    }

    private static double normalizeEngagement(PatientContextState state) {
        long events = state.getEventCount();
        if (events <= 0) return 50.0;
        if (events <= 5) return 40.0;
        if (events <= 20) return 30.0;
        return 20.0;
    }

    private static double piecewiseLinear(double x, double x0, double x1, double y0, double y1) {
        if (x1 == x0) return y0;
        double t = (x - x0) / (x1 - x0);
        return y0 + t * (y1 - y0);
    }
}
