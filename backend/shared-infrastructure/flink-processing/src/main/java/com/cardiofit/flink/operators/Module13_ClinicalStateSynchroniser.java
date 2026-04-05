package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.api.common.state.StateTtlConfig;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.util.*;

public class Module13_ClinicalStateSynchroniser
        extends KeyedProcessFunction<String, CanonicalEvent, ClinicalStateChangeEvent> {

    private static final Logger LOG = LoggerFactory.getLogger(Module13_ClinicalStateSynchroniser.class);

    public static final OutputTag<KB20StateUpdate> KB20_SIDE_OUTPUT =
            new OutputTag<KB20StateUpdate>("kb20-state-updates") {};

    private static final long COALESCING_WINDOW_MS = 5_000L;
    private static final long DAILY_TIMER_INTERVAL_MS = 24 * 3_600_000L;
    private static final long SNAPSHOT_ROTATION_INTERVAL_MS = 7 * 86_400_000L;
    private static final int IDLE_QUIESCENCE_THRESHOLD = 30;

    private transient ValueState<ClinicalStateSummary> summaryState;

    @Override
    public void open(OpenContext openContext) throws Exception {
        ValueStateDescriptor<ClinicalStateSummary> stateDesc =
                new ValueStateDescriptor<>("clinical-state-summary", ClinicalStateSummary.class);
        StateTtlConfig ttl = StateTtlConfig
                .newBuilder(Duration.ofDays(90))
                .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
                .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
                .build();
        stateDesc.enableTimeToLive(ttl);
        summaryState = getRuntimeContext().getState(stateDesc);
        LOG.info("Module 13 Clinical State Synchroniser initialized (90-day TTL, 7-source fan-in)");
    }

    @Override
    public void processElement(CanonicalEvent event, Context ctx,
                               Collector<ClinicalStateChangeEvent> out) throws Exception {
        ClinicalStateSummary state = summaryState.value();
        if (state == null) {
            state = new ClinicalStateSummary(event.getPatientId());
            LOG.info("New patient state created: {}", event.getPatientId());
        }

        // 1. Route event and update state fields
        String sourceModule = routeAndUpdateState(event, state);
        if (sourceModule == null) {
            LOG.debug("Unroutable event for patient {}: type={}", event.getPatientId(), event.getEventType());
            summaryState.update(state);
            return;
        }
        state.recordModuleSeen(sourceModule, event.getEventTime());

        // 2. Compute data completeness
        Module13DataCompletenessMonitor.Result completeness =
                Module13DataCompletenessMonitor.evaluate(state, ctx.timerService().currentProcessingTime());
        state.setDataCompletenessScore(completeness.getCompositeScore());

        // 3. Compute CKM risk velocity
        CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);

        // 4. Detect state changes
        List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                state, velocity, ctx.timerService().currentProcessingTime());

        // 5. Emit state change events + update dedup timestamps
        for (ClinicalStateChangeEvent change : changes) {
            out.collect(change);
            state.getLastEmittedChangeTimestamps().put(
                    change.getChangeType(), change.getProcessingTimestamp());
            LOG.info("State change emitted: patient={}, type={}, priority={}",
                    event.getPatientId(), change.getChangeType(), change.getPriority());
        }

        // 6. Check data absence events from completeness monitor
        if (completeness.isDataAbsenceCritical()) {
            emitDataAbsenceIfNeeded(state, ctx, out, ClinicalStateChangeType.DATA_ABSENCE_CRITICAL, velocity);
        } else if (!completeness.getDataGapFlags().isEmpty()) {
            emitDataAbsenceIfNeeded(state, ctx, out, ClinicalStateChangeType.DATA_ABSENCE_WARNING, velocity);
        }

        // 7. Update velocity in state
        state.setLastComputedVelocity(velocity);
        state.setLastUpdated(ctx.timerService().currentProcessingTime());

        // 8. Project KB-20 updates and buffer for coalescing (ADAPTED for List<KB20StateUpdate>)
        List<KB20StateUpdate> kb20Updates = Module13KB20StateProjector.project(event);
        if (!kb20Updates.isEmpty()) {
            state.getCoalescingBuffer().addAll(kb20Updates);

            // Add computed fields individually (ADAPTED for single-field model)
            long now = ctx.timerService().currentProcessingTime();
            state.getCoalescingBuffer().add(KB20StateUpdate.builder()
                    .patientId(event.getPatientId())
                    .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                    .sourceModule("module13")
                    .fieldPath("ckm_risk_velocity")
                    .value(velocity)
                    .updateTimestamp(now)
                    .build());
            state.getCoalescingBuffer().add(KB20StateUpdate.builder()
                    .patientId(event.getPatientId())
                    .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                    .sourceModule("module13")
                    .fieldPath("data_completeness")
                    .value(completeness.getCompositeScore())
                    .updateTimestamp(now)
                    .build());
            state.getCoalescingBuffer().add(KB20StateUpdate.builder()
                    .patientId(event.getPatientId())
                    .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                    .sourceModule("module13")
                    .fieldPath("last_streaming_update")
                    .value(now)
                    .updateTimestamp(now)
                    .build());

            // Register coalescing timer if not already set
            if (state.getCoalescingTimerMs() < 0) {
                long timerTs = now + COALESCING_WINDOW_MS;
                ctx.timerService().registerProcessingTimeTimer(timerTs);
                state.setCoalescingTimerMs(timerTs);
            }
        }

        // 9. Register daily timer if not set (skip if idle)
        if (state.getDailyTimerMs() < 0
                && state.getConsecutiveZeroCompletenessDays() < IDLE_QUIESCENCE_THRESHOLD) {
            long dailyTs = ctx.timerService().currentProcessingTime() + DAILY_TIMER_INTERVAL_MS;
            ctx.timerService().registerProcessingTimeTimer(dailyTs);
            state.setDailyTimerMs(dailyTs);
        }

        // 10. Register snapshot rotation timer if not set (7-day interval for velocity computation)
        if (state.getSnapshotRotationTimerMs() < 0) {
            long rotationTs = ctx.timerService().currentProcessingTime() + SNAPSHOT_ROTATION_INTERVAL_MS;
            ctx.timerService().registerProcessingTimeTimer(rotationTs);
            state.setSnapshotRotationTimerMs(rotationTs);
        }

        // 11. Reset idle counter since we received real data
        state.setConsecutiveZeroCompletenessDays(0);

        summaryState.update(state);
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx,
                        Collector<ClinicalStateChangeEvent> out) throws Exception {
        ClinicalStateSummary state = summaryState.value();
        if (state == null) return;

        if (timestamp == state.getCoalescingTimerMs()) {
            // Flush coalescing buffer via side output
            for (KB20StateUpdate update : state.getCoalescingBuffer()) {
                ctx.output(KB20_SIDE_OUTPUT, update);
            }
            LOG.debug("Flushed {} KB-20 updates for patient {}",
                    state.getCoalescingBuffer().size(), state.getPatientId());
            state.getCoalescingBuffer().clear();
            state.setCoalescingTimerMs(-1L);

        } else if (timestamp == state.getDailyTimerMs()) {
            // Daily data absence check
            Module13DataCompletenessMonitor.Result completeness =
                    Module13DataCompletenessMonitor.evaluate(state, timestamp);
            state.setDataCompletenessScore(completeness.getCompositeScore());

            CKMRiskVelocity velocity = state.getLastComputedVelocity();
            if (velocity == null) {
                velocity = Module13CKMRiskComputer.compute(state);
                state.setLastComputedVelocity(velocity);
            }

            if (completeness.isDataAbsenceCritical()) {
                emitDataAbsenceIfNeeded(state, ctx, out,
                        ClinicalStateChangeType.DATA_ABSENCE_CRITICAL, velocity);
            } else if (!completeness.getDataGapFlags().isEmpty()) {
                emitDataAbsenceIfNeeded(state, ctx, out,
                        ClinicalStateChangeType.DATA_ABSENCE_WARNING, velocity);
            }

            // Idle-patient timer quiescence
            if (completeness.getCompositeScore() < 0.01) {
                int idle = state.getConsecutiveZeroCompletenessDays() + 1;
                state.setConsecutiveZeroCompletenessDays(idle);
                if (idle >= IDLE_QUIESCENCE_THRESHOLD) {
                    LOG.info("Patient {} idle for {} days, stopping daily timer",
                            state.getPatientId(), idle);
                    state.setDailyTimerMs(-1L);
                } else {
                    long nextDaily = timestamp + DAILY_TIMER_INTERVAL_MS;
                    ctx.timerService().registerProcessingTimeTimer(nextDaily);
                    state.setDailyTimerMs(nextDaily);
                }
            } else {
                state.setConsecutiveZeroCompletenessDays(0);
                long nextDaily = timestamp + DAILY_TIMER_INTERVAL_MS;
                ctx.timerService().registerProcessingTimeTimer(nextDaily);
                state.setDailyTimerMs(nextDaily);
            }

        } else if (timestamp == state.getSnapshotRotationTimerMs()) {
            // 7-day snapshot rotation for velocity computation
            state.rotateSnapshots(timestamp);
            LOG.info("Snapshot rotated for patient {}: previous snapshot captured at {}",
                    state.getPatientId(), timestamp);

            // Re-register rotation timer
            long nextRotation = timestamp + SNAPSHOT_ROTATION_INTERVAL_MS;
            ctx.timerService().registerProcessingTimeTimer(nextRotation);
            state.setSnapshotRotationTimerMs(nextRotation);
        }

        summaryState.update(state);
    }

    private String routeAndUpdateState(CanonicalEvent event, ClinicalStateSummary state) {
        Map<String, Object> payload = event.getPayload();
        if (payload == null) return null;

        String sourceModule = payload.get("source_module") != null
                ? payload.get("source_module").toString() : "";

        switch (sourceModule) {
            case "module7":
                updateFromBPVariability(payload, state);
                return "module7";
            case "module9":
                updateFromEngagement(payload, state);
                return "module9";
            case "module10b":
                updateFromMealPatterns(payload, state);
                return "module10b";
            case "module11b":
                updateFromFitnessPatterns(payload, state);
                return "module11b";
            case "module12":
                updateFromInterventionWindow(payload, state);
                return "module12";
            case "module12b":
                updateFromInterventionDelta(payload, state);
                return "module12b";
            case "enriched":
                updateFromLabResult(payload, state);
                return "enriched";
            default:
                return null;
        }
    }

    private void updateFromBPVariability(Map<String, Object> payload, ClinicalStateSummary state) {
        state.current().arv = toDouble(payload.get("arv"));
        String vc = payload.get("variability_classification") != null
                ? payload.get("variability_classification").toString() : null;
        if (vc != null) {
            try { state.current().variabilityClass = VariabilityClassification.valueOf(vc); }
            catch (IllegalArgumentException ignored) {}
        }
        state.current().meanSBP = toDouble(payload.get("mean_sbp"));
        state.current().meanDBP = toDouble(payload.get("mean_dbp"));
        if (payload.get("morning_surge_magnitude") != null) {
            state.current().morningSurgeMagnitude = toDouble(payload.get("morning_surge_magnitude"));
        }
        if (payload.get("dip_classification") != null) {
            try { state.current().dipClass = DipClassification.valueOf(payload.get("dip_classification").toString()); }
            catch (IllegalArgumentException ignored) {}
        }
    }

    private void updateFromEngagement(Map<String, Object> payload, ClinicalStateSummary state) {
        state.setPreviousEngagementScore(state.current().engagementScore);
        state.current().engagementScore = toDouble(payload.get("composite_score"));
        String level = payload.get("engagement_level") != null
                ? payload.get("engagement_level").toString() : null;
        if (level != null) {
            try { state.current().engagementLevel = EngagementLevel.valueOf(level); }
            catch (IllegalArgumentException ignored) {}
        }
        state.setLatestPhenotype(payload.get("phenotype") != null ? payload.get("phenotype").toString() : null);
        state.setDataTier(payload.get("data_tier") != null ? payload.get("data_tier").toString() : null);
    }

    private void updateFromMealPatterns(Map<String, Object> payload, ClinicalStateSummary state) {
        state.current().meanIAUC = toDouble(payload.get("mean_iauc"));
        state.current().medianExcursion = toDouble(payload.get("median_excursion"));
        String sc = payload.get("salt_sensitivity_class") != null
                ? payload.get("salt_sensitivity_class").toString() : null;
        if (sc != null) {
            try { state.current().saltSensitivity = SaltSensitivityClass.valueOf(sc); }
            catch (IllegalArgumentException ignored) {}
        }
        state.current().saltBeta = toDouble(payload.get("salt_beta"));
    }

    private void updateFromFitnessPatterns(Map<String, Object> payload, ClinicalStateSummary state) {
        state.current().estimatedVO2max = toDouble(payload.get("estimated_vo2max"));
        state.current().vo2maxTrend = toDouble(payload.get("vo2max_trend"));
        state.current().totalMetMinutes = toDouble(payload.get("total_met_minutes"));
        state.current().meanExerciseGlucoseDelta = toDouble(payload.get("mean_exercise_glucose_delta"));
    }

    private void updateFromInterventionWindow(Map<String, Object> payload, ClinicalStateSummary state) {
        String interventionId = payload.get("intervention_id") != null
                ? payload.get("intervention_id").toString() : null;
        String signalType = payload.get("signal_type") != null
                ? payload.get("signal_type").toString() : "";
        if (interventionId == null) return;

        if ("WINDOW_OPENED".equals(signalType)) {
            ClinicalStateSummary.InterventionWindowSummary iw = new ClinicalStateSummary.InterventionWindowSummary();
            iw.setInterventionId(interventionId);
            iw.setStatus("OPENED");
            String itStr = payload.get("intervention_type") != null
                    ? payload.get("intervention_type").toString() : null;
            if (itStr != null) {
                try { iw.setInterventionType(InterventionType.valueOf(itStr)); }
                catch (IllegalArgumentException ignored) {}
            }
            if (payload.get("observation_start_ms") != null)
                iw.setObservationStartMs(toLong(payload.get("observation_start_ms")));
            if (payload.get("observation_end_ms") != null)
                iw.setObservationEndMs(toLong(payload.get("observation_end_ms")));
            state.getActiveInterventions().put(interventionId, iw);
        } else if ("WINDOW_MIDPOINT".equals(signalType)) {
            ClinicalStateSummary.InterventionWindowSummary iw = state.getActiveInterventions().get(interventionId);
            if (iw != null) iw.setStatus("MIDPOINT");
        } else if ("WINDOW_CLOSED".equals(signalType) || "WINDOW_EXPIRED".equals(signalType)
                || "WINDOW_CANCELLED".equals(signalType)) {
            state.getActiveInterventions().remove(interventionId);
        }
    }

    private void updateFromInterventionDelta(Map<String, Object> payload, ClinicalStateSummary state) {
        ClinicalStateSummary.InterventionDeltaSummary delta = new ClinicalStateSummary.InterventionDeltaSummary();
        delta.setInterventionId(payload.get("intervention_id") != null
                ? payload.get("intervention_id").toString() : null);
        String attr = payload.get("trajectory_attribution") != null
                ? payload.get("trajectory_attribution").toString() : null;
        if (attr != null) {
            try { delta.setAttribution(TrajectoryAttribution.valueOf(attr)); }
            catch (IllegalArgumentException ignored) {}
        }
        delta.setAdherenceScore(toDouble(payload.get("adherence_score")));
        delta.setClosedAtMs(System.currentTimeMillis());
        state.getRecentInterventionDeltas().add(delta);

        // Keep only last 10 deltas
        while (state.getRecentInterventionDeltas().size() > 10) {
            state.getRecentInterventionDeltas().remove(0);
        }
    }

    private void updateFromLabResult(Map<String, Object> payload, ClinicalStateSummary state) {
        String labType = payload.get("lab_type") != null ? payload.get("lab_type").toString() : "";
        Double value = toDouble(payload.get("value"));
        if (value == null) return;

        switch (labType) {
            case "FBG": state.current().fbg = value; break;
            case "HBA1C": state.current().hba1c = value; break;
            case "EGFR": state.current().egfr = value; break;
            case "UACR": state.current().uacr = value; break;
            case "LDL": state.current().ldl = value; break;
            case "TOTAL_CHOLESTEROL": state.current().totalCholesterol = value; break;
            case "WEIGHT": state.current().weight = value; break;
            default: break;
        }
    }

    private void emitDataAbsenceIfNeeded(ClinicalStateSummary state, Context ctx,
            Collector<ClinicalStateChangeEvent> out, ClinicalStateChangeType type,
            CKMRiskVelocity velocity) {
        Long lastEmitted = state.getLastEmittedChangeTimestamps().get(type);
        long now = ctx.timerService().currentProcessingTime();
        if (lastEmitted != null && (now - lastEmitted) < 24 * 3_600_000L) return;

        ClinicalStateChangeEvent event = ClinicalStateChangeEvent.builder()
                .changeId(UUID.randomUUID().toString())
                .patientId(state.getPatientId())
                .changeType(type)
                .previousValue("expected data")
                .currentValue("no data received")
                .triggerModule("module13")
                .ckmVelocityAtChange(velocity)
                .dataCompletenessAtChange(state.getDataCompletenessScore())
                .confidenceScore(state.getDataCompletenessScore())
                .processingTimestamp(now)
                .build();
        out.collect(event);
        state.getLastEmittedChangeTimestamps().put(type, now);
        LOG.warn("Data absence detected: patient={}, type={}", state.getPatientId(), type);
    }

    private static Double toDouble(Object v) {
        if (v == null) return null;
        if (v instanceof Number) return ((Number) v).doubleValue();
        try { return Double.parseDouble(v.toString()); }
        catch (NumberFormatException e) { return null; }
    }

    private static long toLong(Object v) {
        if (v instanceof Number) return ((Number) v).longValue();
        try { return Long.parseLong(v.toString()); }
        catch (NumberFormatException e) { return 0L; }
    }
}
