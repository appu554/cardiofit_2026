package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;

import java.util.*;

public class Module13TestBuilder {

    public static final long HOUR_MS = 3_600_000L;
    public static final long DAY_MS = 86_400_000L;
    public static final long WEEK_MS = 7 * DAY_MS;
    public static final long BASE_TIME = 1743552000000L; // 2025-04-02 00:00:00 UTC

    public static long hoursAfter(long base, int hours) { return base + hours * HOUR_MS; }
    public static long daysAfter(long base, int days) { return base + days * DAY_MS; }
    public static long weeksAfter(long base, int weeks) { return base + weeks * WEEK_MS; }

    // ---- Event builders keyed by source module ----

    /** Module 7: BP variability metric event */
    public static CanonicalEvent bpVariabilityEvent(String patientId, long timestamp,
            double arv, String variabilityClass, double meanSBP, double meanDBP) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "module7");
        payload.put("arv", arv);
        payload.put("variability_classification", variabilityClass);
        payload.put("mean_sbp", meanSBP);
        payload.put("mean_dbp", meanDBP);
        return baseEvent(patientId, EventType.VITAL_SIGN, timestamp, payload);
    }

    /** Module 9: Engagement signal event */
    public static CanonicalEvent engagementEvent(String patientId, long timestamp,
            double compositeScore, String engagementLevel, String phenotype, String dataTier) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "module9");
        payload.put("composite_score", compositeScore);
        payload.put("engagement_level", engagementLevel);
        payload.put("phenotype", phenotype);
        payload.put("data_tier", dataTier);
        return baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp, payload);
    }

    /** Module 10b: Meal pattern summary event */
    public static CanonicalEvent mealPatternEvent(String patientId, long timestamp,
            double meanIAUC, double medianExcursion, String saltSensitivity, Double saltBeta) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "module10b");
        payload.put("mean_iauc", meanIAUC);
        payload.put("median_excursion", medianExcursion);
        payload.put("salt_sensitivity_class", saltSensitivity);
        payload.put("salt_beta", saltBeta);
        return baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp, payload);
    }

    /** Module 11b: Fitness pattern summary event */
    public static CanonicalEvent fitnessPatternEvent(String patientId, long timestamp,
            double estimatedVO2max, double vo2maxTrend, double totalMetMinutes, double meanGlucoseDelta) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "module11b");
        payload.put("estimated_vo2max", estimatedVO2max);
        payload.put("vo2max_trend", vo2maxTrend);
        payload.put("total_met_minutes", totalMetMinutes);
        payload.put("mean_exercise_glucose_delta", meanGlucoseDelta);
        return baseEvent(patientId, EventType.DEVICE_READING, timestamp, payload);
    }

    /** Module 12: Intervention window signal event */
    public static CanonicalEvent interventionWindowEvent(String patientId, long timestamp,
            String interventionId, String signalType, String interventionType,
            long observationStartMs, long observationEndMs) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "module12");
        payload.put("intervention_id", interventionId);
        payload.put("signal_type", signalType);
        payload.put("intervention_type", interventionType);
        payload.put("observation_start_ms", observationStartMs);
        payload.put("observation_end_ms", observationEndMs);
        return baseEvent(patientId, EventType.MEDICATION_ORDERED, timestamp, payload);
    }

    /** Module 12b: Intervention delta event */
    public static CanonicalEvent interventionDeltaEvent(String patientId, long timestamp,
            String interventionId, String attribution, Double adherenceScore,
            Double fbgDelta, Double sbpDelta, Double egfrDelta) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "module12b");
        payload.put("intervention_id", interventionId);
        payload.put("trajectory_attribution", attribution);
        payload.put("adherence_score", adherenceScore);
        payload.put("fbg_delta", fbgDelta);
        payload.put("sbp_delta", sbpDelta);
        payload.put("egfr_delta", egfrDelta);
        return baseEvent(patientId, EventType.LAB_RESULT, timestamp, payload);
    }

    /** Lab result from enriched-patient-events-v1 */
    public static CanonicalEvent labEvent(String patientId, long timestamp,
            String labType, double value) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "enriched");
        payload.put("lab_type", labType); // FBG, HBA1C, EGFR, UACR, LDL, TOTAL_CHOLESTEROL
        payload.put("value", value);
        return baseEvent(patientId, EventType.LAB_RESULT, timestamp, payload);
    }

    // ---- State builders ----

    /** Empty state for new patient */
    public static ClinicalStateSummary emptyState(String patientId) {
        return new ClinicalStateSummary(patientId);
    }

    /** State with baselines — simulates a patient with 2 weeks of history (current snapshot populated) */
    public static ClinicalStateSummary stateWithBaselines(String patientId) {
        ClinicalStateSummary s = new ClinicalStateSummary(patientId);
        s.current().fbg = 130.0;
        s.current().hba1c = 7.2;
        s.current().egfr = 65.0;
        s.current().meanSBP = 142.0;
        s.current().meanDBP = 88.0;
        s.current().arv = 10.5;
        s.current().variabilityClass = VariabilityClassification.MODERATE;
        s.current().engagementScore = 0.72;
        s.current().engagementLevel = EngagementLevel.GREEN;
        s.current().estimatedVO2max = 35.0;
        s.setDataTier("TIER_2_SMBG");
        s.setChannel("CORPORATE");
        s.recordModuleSeen("module7", BASE_TIME);
        s.recordModuleSeen("module8", BASE_TIME);
        s.recordModuleSeen("module9", BASE_TIME);
        s.recordModuleSeen("module10b", BASE_TIME);
        s.recordModuleSeen("module11b", BASE_TIME);
        s.recordModuleSeen("enriched", BASE_TIME);
        s.recordModuleSeen("module12", BASE_TIME);
        s.recordModuleSeen("module12b", BASE_TIME);
        s.setDataCompletenessScore(0.875); // 7/8 modules seen — above HIGH-priority gate threshold
        s.setLastUpdated(BASE_TIME);
        return s;
    }

    /**
     * State with snapshot pair — simulates a patient after first 7-day rotation.
     * Both previousSnapshot and currentSnapshot are populated with typical South Asian defaults.
     * This is required for CKMRiskComputer velocity tests (Issue 1 fix).
     */
    public static ClinicalStateSummary stateWithSnapshotPair(String patientId) {
        ClinicalStateSummary s = stateWithBaselines(patientId);
        // Rotate: current → previous, then reset current to same defaults
        s.rotateSnapshots(BASE_TIME);
        // Re-populate current with same baselines (tests will override specific fields)
        s.current().fbg = 130.0;
        s.current().hba1c = 7.2;
        s.current().egfr = 65.0;
        s.current().meanSBP = 142.0;
        s.current().meanDBP = 88.0;
        s.current().arv = 10.5;
        s.current().variabilityClass = VariabilityClassification.MODERATE;
        s.current().engagementScore = 0.72;
        s.current().engagementLevel = EngagementLevel.GREEN;
        s.current().estimatedVO2max = 35.0;
        return s;
    }

    /** State with an active intervention window */
    public static ClinicalStateSummary stateWithActiveIntervention(String patientId,
            String interventionId, InterventionType type) {
        ClinicalStateSummary s = stateWithBaselines(patientId);
        ClinicalStateSummary.InterventionWindowSummary iw = new ClinicalStateSummary.InterventionWindowSummary();
        iw.setInterventionId(interventionId);
        iw.setInterventionType(type);
        iw.setStatus("OPENED");
        iw.setObservationStartMs(BASE_TIME);
        iw.setObservationEndMs(BASE_TIME + 28 * DAY_MS);
        iw.setTrajectoryAtOpen(TrajectoryClassification.STABLE);
        s.getActiveInterventions().put(interventionId, iw);
        return s;
    }

    /** State with historical CKM velocity (for testing transitions) */
    public static ClinicalStateSummary stateWithVelocity(String patientId,
            CKMRiskVelocity.CompositeClassification classification) {
        ClinicalStateSummary s = stateWithBaselines(patientId);
        CKMRiskVelocity velocity = CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.METABOLIC, classification == CKMRiskVelocity.CompositeClassification.IMPROVING ? -0.4 : 0.1)
                .domainVelocity(CKMRiskDomain.RENAL, 0.0)
                .domainVelocity(CKMRiskDomain.CARDIOVASCULAR, 0.0)
                .compositeClassification(classification)
                .compositeScore(classification == CKMRiskVelocity.CompositeClassification.IMPROVING ? -0.3 : 0.1)
                .dataCompleteness(0.85)
                .computationTimestamp(BASE_TIME)
                .build();
        s.setLastComputedVelocity(velocity);
        return s;
    }

    /** Module 8: Comorbidity interaction alert event */
    public static CanonicalEvent comorbidityAlertEvent(String patientId, long timestamp,
            String ruleId, String severity) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "module8");
        payload.put("ruleId", ruleId);
        payload.put("severity", severity);
        return baseEvent(patientId, EventType.CLINICAL_DOCUMENT, timestamp, payload);
    }

    /** Lab result with KB-20 personalised targets embedded in payload */
    public static CanonicalEvent labEventWithPersonalizedTargets(String patientId, long timestamp,
            String labType, double value, Map<String, Object> kb20Targets) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "enriched");
        payload.put("lab_type", labType);
        payload.put("value", value);
        if (kb20Targets != null) {
            payload.put("kb20_personalized_targets", kb20Targets);
        }
        return baseEvent(patientId, EventType.LAB_RESULT, timestamp, payload);
    }

    // ---- Private helpers ----

    private static CanonicalEvent baseEvent(String patientId, EventType type,
            long timestamp, Map<String, Object> payload) {
        return CanonicalEvent.builder()
                .id(UUID.randomUUID().toString())
                .patientId(patientId)
                .eventType(type)
                .eventTime(timestamp)
                .sourceSystem("flink-module13-test")
                .payload(payload)
                .build();
    }
}
