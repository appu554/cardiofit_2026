package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.state.StateTtlConfig;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.util.Map;
import java.util.UUID;

/**
 * Module 12b: Intervention Delta Computer.
 *
 * Consumes WINDOW_CLOSED signals from clinical.intervention-window-signals
 * AND enriched-patient-events-v1 (for current vital baselines).
 * Computes streaming metric deltas for the physician dashboard.
 *
 * Separate Flink job from Module 12 for failure isolation
 * (same pattern as Module 10/10b and 11/11b).
 *
 * State TTL: 90 days to retain baseline snapshots across long observation windows.
 */
public class Module12b_InterventionDeltaComputer
        extends KeyedProcessFunction<String, CanonicalEvent, InterventionDeltaRecord> {

    private static final Logger LOG =
            LoggerFactory.getLogger(Module12b_InterventionDeltaComputer.class);

    /** Stores per-patient baseline snapshots captured at WINDOW_OPENED. */
    private transient ValueState<InterventionBaselineState> baselineState;

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        ValueStateDescriptor<InterventionBaselineState> stateDesc =
                new ValueStateDescriptor<>("intervention-baseline-state",
                        InterventionBaselineState.class);

        StateTtlConfig ttl = StateTtlConfig
                .newBuilder(Duration.ofDays(90))
                .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
                .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
                .build();
        stateDesc.enableTimeToLive(ttl);
        baselineState = getRuntimeContext().getState(stateDesc);

        LOG.info("Module 12b Intervention Delta Computer initialized (90-day TTL)");
    }

    @Override
    public void processElement(CanonicalEvent event, Context ctx,
                               Collector<InterventionDeltaRecord> out) throws Exception {
        InterventionBaselineState state = baselineState.value();
        if (state == null) {
            state = new InterventionBaselineState();
            state.patientId = event.getPatientId();
        }

        Map<String, Object> payload = event.getPayload();
        if (payload == null) {
            baselineState.update(state);
            return;
        }

        String signalType = getStr(payload, "signal_type");

        if ("WINDOW_OPENED".equals(signalType)) {
            // Capture baseline snapshot
            String interventionId = getStr(payload, "intervention_id");
            InterventionBaselineState.Baseline baseline = new InterventionBaselineState.Baseline();
            baseline.fbg = state.currentFBG;
            baseline.sbp = state.currentSBP;
            baseline.dbp = state.currentDBP;
            baseline.weight = state.currentWeight;
            baseline.hba1c = state.currentHbA1c;
            baseline.egfr = state.currentEGFR;
            baseline.tir = state.currentTIR;
            baseline.trajectoryAtOpen = getStr(payload, "trajectory_at_signal");
            baseline.interventionType = getStr(payload, "intervention_type");
            baseline.concurrentCount = getIntFromPayload(payload, "concurrent_intervention_count", 0);
            state.baselines.put(interventionId, baseline);

        } else if ("WINDOW_CLOSED".equals(signalType)) {
            // Compute deltas
            String interventionId = getStr(payload, "intervention_id");
            InterventionBaselineState.Baseline baseline = state.baselines.get(interventionId);
            if (baseline == null) {
                LOG.warn("WINDOW_CLOSED without baseline for intervention: {}", interventionId);
                baselineState.update(state);
                return;
            }

            TrajectoryClassification before = safeTrajectory(baseline.trajectoryAtOpen);
            String trajectoryDuringStr = getStr(payload, "trajectory_at_signal");
            TrajectoryClassification during = safeTrajectory(trajectoryDuringStr);
            TrajectoryAttribution attribution = TrajectoryAttribution.fromTrajectories(before, during);

            InterventionType type = safeInterventionType(baseline.interventionType);

            // Extract adherence score from the WINDOW_CLOSED signal's midpoint data
            Double adherenceScore = extractAdherenceScore(payload);

            InterventionDeltaRecord delta = InterventionDeltaRecord.builder()
                    .deltaId(UUID.randomUUID().toString())
                    .interventionId(interventionId)
                    .patientId(state.patientId)
                    .interventionType(type)
                    .fbgDelta(computeDelta(state.currentFBG, baseline.fbg))
                    .sbpDelta(computeDelta(state.currentSBP, baseline.sbp))
                    .dbpDelta(computeDelta(state.currentDBP, baseline.dbp))
                    .weightDeltaKg(computeDelta(state.currentWeight, baseline.weight))
                    .hba1cDelta(computeDelta(state.currentHbA1c, baseline.hba1c))
                    .egfrDelta(computeDelta(state.currentEGFR, baseline.egfr))
                    .tirDelta(computeDelta(state.currentTIR, baseline.tir))
                    .trajectoryAttribution(attribution)
                    .adherenceScore(adherenceScore)
                    .concurrentCount(baseline.concurrentCount)
                    .dataCompletenessScore(computeCompleteness(state))
                    .processingTimestamp(System.currentTimeMillis())
                    .build();

            out.collect(delta);

            // Clean up baseline
            state.baselines.remove(interventionId);

            LOG.info("Delta computed: intervention={}, fbg={}, sbp={}, attribution={}",
                    interventionId, delta.getFbgDelta(), delta.getSbpDelta(), attribution);

        } else {
            // Patient data event — update current baselines
            updateCurrentValues(event, state);
        }

        baselineState.update(state);
    }

    private void updateCurrentValues(CanonicalEvent event, InterventionBaselineState state) {
        Map<String, Object> payload = event.getPayload();
        if (payload == null) return;

        EventType eventType = event.getEventType();
        if (eventType == EventType.LAB_RESULT) {
            String labType = getStr(payload, "lab_type");
            Double value = getDouble(payload, "value");
            if ("FBG".equals(labType) && value != null) state.currentFBG = value;
            if ("EGFR".equals(labType) && value != null) state.currentEGFR = value;
            if ("HBA1C".equals(labType) && value != null) state.currentHbA1c = value;
        } else if (eventType == EventType.VITAL_SIGN) {
            Double sbp = getDouble(payload, "systolic_bp");
            Double dbp = getDouble(payload, "diastolic_bp");
            Double weight = getDouble(payload, "weight_kg");
            if (sbp != null) state.currentSBP = sbp;
            if (dbp != null) state.currentDBP = dbp;
            if (weight != null) state.currentWeight = weight;
        }
    }

    @SuppressWarnings("unchecked")
    private static Double extractAdherenceScore(Map<String, Object> payload) {
        Object adherenceObj = payload.get("adherence_signals_at_midpoint");
        if (adherenceObj instanceof Map) {
            Object score = ((Map<String, Object>) adherenceObj).get("score");
            if (score instanceof Number) return ((Number) score).doubleValue();
        }
        return null;
    }

    private static Double computeDelta(Double current, Double baseline) {
        if (current == null || baseline == null) return null;
        return current - baseline;
    }

    private static double computeCompleteness(InterventionBaselineState state) {
        int available = 0;
        int total = 5; // FBG, SBP, weight, HbA1c, eGFR
        if (state.currentFBG != null) available++;
        if (state.currentSBP != null) available++;
        if (state.currentWeight != null) available++;
        if (state.currentHbA1c != null) available++;
        if (state.currentEGFR != null) available++;
        return (double) available / total;
    }

    private static TrajectoryClassification safeTrajectory(String s) {
        if (s == null) return TrajectoryClassification.UNKNOWN;
        try { return TrajectoryClassification.valueOf(s); }
        catch (IllegalArgumentException e) { return TrajectoryClassification.UNKNOWN; }
    }

    private static InterventionType safeInterventionType(String s) {
        if (s == null) return null;
        try { return InterventionType.valueOf(s); }
        catch (IllegalArgumentException e) { return null; }
    }

    private static String getStr(Map<String, Object> m, String key) {
        Object v = m.get(key);
        return v != null ? v.toString() : null;
    }

    private static Double getDouble(Map<String, Object> m, String key) {
        Object v = m.get(key);
        if (v instanceof Number) return ((Number) v).doubleValue();
        return null;
    }

    private static int getIntFromPayload(Map<String, Object> m, String key, int def) {
        Object v = m.get(key);
        if (v instanceof Number) return ((Number) v).intValue();
        return def;
    }

    /**
     * Per-patient baseline state for Module 12b.
     * Stores current vital values and per-intervention baseline snapshots.
     */
    public static class InterventionBaselineState implements java.io.Serializable {
        private static final long serialVersionUID = 1L;
        public String patientId;
        public Double currentFBG;
        public Double currentSBP;
        public Double currentDBP;
        public Double currentWeight;
        public Double currentHbA1c;
        public Double currentEGFR;
        public Double currentTIR;
        public java.util.Map<String, Baseline> baselines = new java.util.HashMap<>();

        public static class Baseline implements java.io.Serializable {
            private static final long serialVersionUID = 1L;
            public Double fbg, sbp, dbp, weight, hba1c, egfr, tir;
            public String trajectoryAtOpen;
            public String interventionType;
            public int concurrentCount;
        }
    }
}
