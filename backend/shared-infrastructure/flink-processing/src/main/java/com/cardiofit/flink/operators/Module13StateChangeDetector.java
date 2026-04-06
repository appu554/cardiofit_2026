package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;

import java.util.*;

/**
 * Module 13 — State Change Detector
 *
 * Static analyser (no Flink dependencies) that detects clinically significant
 * transitions from a {@link ClinicalStateSummary}, a freshly-computed
 * {@link CKMRiskVelocity}, and the current processing timestamp.
 *
 * Nine change types are evaluated:
 *  1. CKM_RISK_ESCALATION       — non-DETERIORATING → DETERIORATING
 *  2. CKM_DOMAIN_DIVERGENCE     — 2+ domains worsening simultaneously
 *  3. RENAL_RAPID_DECLINE       — eGFR < 45
 *  4. ENGAGEMENT_COLLAPSE       — engagement score drop >= 0.35
 *  5. INTERVENTION_FUTILITY     — 2+ consecutive INSUFFICIENT attributions
 *  6. TRAJECTORY_REVERSAL       — IMPROVING → DETERIORATING
 *  7. METABOLIC_MILESTONE       — FBG < 110
 *  8. BP_MILESTONE              — mean SBP < 130
 *  9. CROSS_MODULE_INCONSISTENCY — high adherence + deteriorating trajectory
 *
 * All events are subject to a 24-hour dedup window stored in the state's
 * {@code lastEmittedChangeTimestamps} EnumMap.
 */
public final class Module13StateChangeDetector {

    private static final long DEDUP_WINDOW_MS = 24 * 3_600_000L;
    private static final double FBG_TARGET = 110.0;
    private static final double SBP_TARGET = 130.0;
    private static final double ENGAGEMENT_COLLAPSE_DELTA = 0.35;
    private static final int FUTILITY_CONSECUTIVE_COUNT = 2;

    private Module13StateChangeDetector() {}

    /**
     * Evaluate all nine change-type detectors against the current state and
     * newly computed velocity. Returns a (possibly empty) list of deduplicated
     * {@link ClinicalStateChangeEvent} instances.
     */
    public static List<ClinicalStateChangeEvent> detect(
            ClinicalStateSummary state,
            CKMRiskVelocity newVelocity,
            long currentTimestamp) {

        List<ClinicalStateChangeEvent> events = new ArrayList<>();

        checkCKMRiskEscalation(state, newVelocity, currentTimestamp, events);
        checkCKMDomainDivergence(newVelocity, state, currentTimestamp, events);
        checkRenalRapidDecline(state, currentTimestamp, events);
        checkEngagementCollapse(state, currentTimestamp, events);
        checkInterventionFutility(state, currentTimestamp, events);
        checkTrajectoryReversal(state, newVelocity, currentTimestamp, events);
        checkMetabolicMilestone(state, currentTimestamp, events);
        checkBPMilestone(state, currentTimestamp, events);
        checkCrossModuleInconsistency(state, newVelocity, currentTimestamp, events);

        return events;
    }

    // ---- Individual detectors ----

    private static void checkCKMRiskEscalation(ClinicalStateSummary state,
            CKMRiskVelocity newVelocity, long ts, List<ClinicalStateChangeEvent> events) {
        if (state.getLastComputedVelocity() == null) return;
        CKMRiskVelocity.CompositeClassification prev = state.getLastComputedVelocity().getCompositeClassification();
        CKMRiskVelocity.CompositeClassification curr = newVelocity.getCompositeClassification();
        if (prev != CKMRiskVelocity.CompositeClassification.DETERIORATING
                && curr == CKMRiskVelocity.CompositeClassification.DETERIORATING) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.CKM_RISK_ESCALATION,
                    prev.name(), curr.name(), null, "module13", newVelocity);
        }
    }

    private static void checkCKMDomainDivergence(CKMRiskVelocity velocity,
            ClinicalStateSummary state, long ts, List<ClinicalStateChangeEvent> events) {
        if (velocity.isCrossDomainAmplification()) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.CKM_DOMAIN_DIVERGENCE,
                    String.valueOf(velocity.getDomainsDeteriorating()), "2+ domains worsening",
                    null, "module13", velocity);
        }
    }

    private static void checkRenalRapidDecline(ClinicalStateSummary state,
            long ts, List<ClinicalStateChangeEvent> events) {
        Double egfr = state.current().egfr;
        if (egfr != null && egfr < 45.0) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.RENAL_RAPID_DECLINE,
                    "eGFR_baseline", String.valueOf(egfr),
                    CKMRiskDomain.RENAL, "module7", state.getLastComputedVelocity());
        }
    }

    private static void checkEngagementCollapse(ClinicalStateSummary state,
            long ts, List<ClinicalStateChangeEvent> events) {
        Double currentEngagement = state.current().engagementScore;
        if (state.getPreviousEngagementScore() != null && currentEngagement != null) {
            double drop = state.getPreviousEngagementScore() - currentEngagement;
            if (drop >= ENGAGEMENT_COLLAPSE_DELTA) {
                emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.ENGAGEMENT_COLLAPSE,
                        String.valueOf(state.getPreviousEngagementScore()),
                        String.valueOf(currentEngagement),
                        null, "module9", state.getLastComputedVelocity());
            }
        }
    }

    private static void checkInterventionFutility(ClinicalStateSummary state,
            long ts, List<ClinicalStateChangeEvent> events) {
        List<ClinicalStateSummary.InterventionDeltaSummary> deltas = state.getRecentInterventionDeltas();
        if (deltas.size() >= FUTILITY_CONSECUTIVE_COUNT) {
            int consecutiveInsufficient = 0;
            for (int i = deltas.size() - 1; i >= 0; i--) {
                if (deltas.get(i).getAttribution() == TrajectoryAttribution.INTERVENTION_INSUFFICIENT) {
                    consecutiveInsufficient++;
                } else {
                    break;
                }
            }
            if (consecutiveInsufficient >= FUTILITY_CONSECUTIVE_COUNT) {
                emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.INTERVENTION_FUTILITY,
                        consecutiveInsufficient + " consecutive INSUFFICIENT",
                        "Phenotype review needed", null, "module12b",
                        state.getLastComputedVelocity());
            }
        }
    }

    private static void checkTrajectoryReversal(ClinicalStateSummary state,
            CKMRiskVelocity newVelocity, long ts, List<ClinicalStateChangeEvent> events) {
        if (state.getLastComputedVelocity() == null) return;
        CKMRiskVelocity.CompositeClassification prev = state.getLastComputedVelocity().getCompositeClassification();
        CKMRiskVelocity.CompositeClassification curr = newVelocity.getCompositeClassification();
        if (prev == CKMRiskVelocity.CompositeClassification.IMPROVING
                && curr == CKMRiskVelocity.CompositeClassification.DETERIORATING) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.TRAJECTORY_REVERSAL,
                    prev.name(), curr.name(), null, "module13", newVelocity);
        }
    }

    private static void checkMetabolicMilestone(ClinicalStateSummary state,
            long ts, List<ClinicalStateChangeEvent> events) {
        Double fbg = state.current().fbg;
        if (fbg != null && fbg < FBG_TARGET) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.METABOLIC_MILESTONE,
                    "FBG above target", String.valueOf(fbg),
                    CKMRiskDomain.METABOLIC, "enriched", state.getLastComputedVelocity());
        }
    }

    private static void checkBPMilestone(ClinicalStateSummary state,
            long ts, List<ClinicalStateChangeEvent> events) {
        Double meanSBP = state.current().meanSBP;
        if (meanSBP != null && meanSBP < SBP_TARGET) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.BP_MILESTONE,
                    "SBP above target", String.valueOf(meanSBP),
                    CKMRiskDomain.CARDIOVASCULAR, "module7", state.getLastComputedVelocity());
        }
    }

    private static void checkCrossModuleInconsistency(ClinicalStateSummary state,
            CKMRiskVelocity velocity, long ts, List<ClinicalStateChangeEvent> events) {
        boolean highAdherence = state.current().engagementScore != null
                && state.current().engagementScore > 0.7;
        boolean hasHighAdherenceDeltas = state.getRecentInterventionDeltas().stream()
                .anyMatch(d -> d.getAdherenceScore() != null && d.getAdherenceScore() > 0.7);
        boolean deteriorating = velocity.getCompositeClassification()
                == CKMRiskVelocity.CompositeClassification.DETERIORATING;

        if ((highAdherence || hasHighAdherenceDeltas) && deteriorating) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.CROSS_MODULE_INCONSISTENCY,
                    "HIGH adherence", "DETERIORATING trajectory", null, "module13", velocity);
        }
    }

    // ---- Dedup + emit helper ----

    private static void emitIfNotDeduped(ClinicalStateSummary state, long currentTs,
            List<ClinicalStateChangeEvent> events, ClinicalStateChangeType type,
            String previousValue, String currentValue, CKMRiskDomain domain,
            String triggerModule, CKMRiskVelocity velocity) {

        Long lastEmitted = state.getLastEmittedChangeTimestamps().get(type);
        if (lastEmitted != null && (currentTs - lastEmitted) < DEDUP_WINDOW_MS) {
            return;
        }

        events.add(ClinicalStateChangeEvent.builder()
                .changeId(UUID.randomUUID().toString())
                .patientId(state.getPatientId())
                .changeType(type)
                .previousValue(previousValue)
                .currentValue(currentValue)
                .domain(domain)
                .triggerModule(triggerModule)
                .ckmVelocityAtChange(velocity)
                .dataCompletenessAtChange(state.getDataCompletenessScore())
                .confidenceScore(state.getDataCompletenessScore())
                .processingTimestamp(currentTs)
                .build());
    }
}
