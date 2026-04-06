package com.cardiofit.flink.operators;

import com.cardiofit.flink.config.Module13PilotConfig;
import com.cardiofit.flink.metrics.Module13Metrics;
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
    private static final int MAX_COALESCING_BUFFER_SIZE = 100;
    private static final long DAILY_TIMER_INTERVAL_MS = 24 * 3_600_000L;
    private static final long SNAPSHOT_ROTATION_INTERVAL_MS = 7 * 86_400_000L;
    private static final int IDLE_QUIESCENCE_THRESHOLD = 30;

    private transient ValueState<ClinicalStateSummary> summaryState;
    private transient Module13Metrics metrics;
    private transient Module13PilotConfig pilotConfig;

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
        metrics = new Module13Metrics(getRuntimeContext().getMetricGroup());
        pilotConfig = Module13PilotConfig.fromEnvironment();
        LOG.info("Module 13 Clinical State Synchroniser initialized (90-day TTL, 7-source fan-in, enabled={})",
                pilotConfig.isEnabled());
    }

    @Override
    public void processElement(CanonicalEvent event, Context ctx,
                               Collector<ClinicalStateChangeEvent> out) throws Exception {
        // Feature flag: master kill-switch
        if (!pilotConfig.isEnabled()) return;

        // Feature flag: pilot cohort filter
        if (!pilotConfig.isPatientInPilot(event.getPatientId())) return;

        ClinicalStateSummary state = summaryState.value();
        if (state == null) {
            state = new ClinicalStateSummary(event.getPatientId());
            LOG.info("New patient state created: {}", event.getPatientId());
        }

        // 1. Route event and update state fields
        long processStart = System.nanoTime();
        String sourceModule = routeAndUpdateState(event, state, ctx.timerService().currentProcessingTime());
        if (sourceModule == null) {
            LOG.debug("Unroutable event for patient {}: type={}", event.getPatientId(), event.getEventType());
            metrics.recordUnroutableEvent();
            summaryState.update(state);
            return;
        }
        state.recordModuleSeen(sourceModule, event.getEventTime());

        // 2. Compute data completeness
        Module13DataCompletenessMonitor.Result completeness =
                Module13DataCompletenessMonitor.evaluate(state, ctx.timerService().currentProcessingTime());
        state.setDataCompletenessScore(completeness.getCompositeScore());

        // 3. Compute CKM risk velocity (feature-flagged)
        CKMRiskVelocity velocity;
        if (pilotConfig.isCkmVelocityEnabled()) {
            velocity = Module13CKMRiskComputer.compute(state);
        } else {
            velocity = CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.UNKNOWN)
                    .compositeScore(0.0).dataCompleteness(0.0)
                    .computationTimestamp(ctx.timerService().currentProcessingTime()).build();
        }

        // 4. Detect state changes (feature-flagged)
        List<ClinicalStateChangeEvent> changes;
        if (pilotConfig.isStateChangesEnabled()) {
            changes = Module13StateChangeDetector.detect(
                    state, velocity, ctx.timerService().currentProcessingTime());
            // Safety cap
            if (changes.size() > pilotConfig.getMaxStateChangesPerEvent()) {
                LOG.warn("State changes capped: {} → {} for patient {}",
                        changes.size(), pilotConfig.getMaxStateChangesPerEvent(), event.getPatientId());
                changes = changes.subList(0, pilotConfig.getMaxStateChangesPerEvent());
            }
        } else {
            changes = Collections.emptyList();
        }

        // 5. Record metrics for velocity and completeness
        metrics.recordVelocityClassification(
                velocity.getCompositeClassification().name(),
                velocity.getCompositeScore(),
                velocity.isCrossDomainAmplification());
        metrics.recordDataCompleteness(completeness.getCompositeScore());

        // 6. Emit state change events + update dedup timestamps (dry-run guard)
        for (ClinicalStateChangeEvent change : changes) {
            if (!pilotConfig.isDryRun()) {
                out.collect(change);
            }
            state.getLastEmittedChangeTimestamps().put(
                    change.getChangeType(), change.getProcessingTimestamp());
            metrics.recordStateChangeEmitted(change.getPriority());
            LOG.info("State change {}: patient={}, type={}, priority={}",
                    pilotConfig.isDryRun() ? "computed (dry-run)" : "emitted",
                    event.getPatientId(), change.getChangeType(), change.getPriority());
        }

        // 7. Check data absence events from completeness monitor
        if (completeness.isDataAbsenceCritical()) {
            emitDataAbsenceIfNeeded(state, ctx, out, ClinicalStateChangeType.DATA_ABSENCE_CRITICAL, velocity);
        } else if (!completeness.getDataGapFlags().isEmpty()) {
            emitDataAbsenceIfNeeded(state, ctx, out, ClinicalStateChangeType.DATA_ABSENCE_WARNING, velocity);
        }

        // 7. Update velocity in state
        state.setLastComputedVelocity(velocity);
        state.setLastUpdated(ctx.timerService().currentProcessingTime());

        // 8. Project KB-20 updates and buffer for coalescing (feature-flagged, dry-run guarded)
        List<KB20StateUpdate> kb20Updates = pilotConfig.isKb20WritebackEnabled() && !pilotConfig.isDryRun()
                ? Module13KB20StateProjector.project(event) : Collections.emptyList();
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

            // Safety cap: evict oldest if buffer exceeds limit
            while (state.getCoalescingBuffer().size() > MAX_COALESCING_BUFFER_SIZE) {
                state.getCoalescingBuffer().remove(0);
                metrics.recordCoalescingEviction();
            }

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

        // 12. Record processing latency and A1 enrichment status
        long latencyMs = (System.nanoTime() - processStart) / 1_000_000;
        metrics.recordEventProcessed(sourceModule, latencyMs);
        if (state.getPersonalizedFBGTarget() == null) {
            metrics.recordPersonalizedTargetsFallback();
        }
        metrics.setCoalescingBufferDepth(state.getCoalescingBuffer().size());

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
            int flushedCount = state.getCoalescingBuffer().size();
            LOG.debug("Flushed {} KB-20 updates for patient {}",
                    flushedCount, state.getPatientId());
            state.getCoalescingBuffer().clear();
            state.setCoalescingTimerMs(-1L);
            metrics.recordCoalescingFlush(flushedCount);
            metrics.setCoalescingBufferDepth(0);

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
            metrics.recordSnapshotRotation();
            LOG.info("Snapshot rotated for patient {}: previous snapshot captured at {}",
                    state.getPatientId(), timestamp);

            // Re-register rotation timer
            long nextRotation = timestamp + SNAPSHOT_ROTATION_INTERVAL_MS;
            ctx.timerService().registerProcessingTimeTimer(nextRotation);
            state.setSnapshotRotationTimerMs(nextRotation);
        }

        summaryState.update(state);
    }

    private String routeAndUpdateState(CanonicalEvent event, ClinicalStateSummary state, long processingTime) {
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
                updateFromInterventionDelta(payload, state, processingTime);
                return "module12b";
            case "module8":
                updateFromComorbidityAlert(payload, state);
                return "module8";
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

    private void updateFromInterventionDelta(Map<String, Object> payload, ClinicalStateSummary state, long processingTime) {
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
        delta.setClosedAtMs(processingTime);
        state.getRecentInterventionDeltas().add(delta);

        // Keep only last 10 deltas
        while (state.getRecentInterventionDeltas().size() > 10) {
            state.getRecentInterventionDeltas().remove(0);
        }
    }

    /** PIPE-7: Update CID alert state from Module 8 comorbidity alerts */
    private void updateFromComorbidityAlert(Map<String, Object> payload, ClinicalStateSummary state) {
        String ruleId = payload.get("ruleId") != null ? payload.get("ruleId").toString() : "";
        String severity = payload.get("severity") != null ? payload.get("severity").toString() : "";

        if (!ruleId.isEmpty()) {
            state.getActiveCIDRuleIds().add(ruleId);
        }
        if ("HALT".equals(severity)) {
            state.setActiveCIDHaltCount(state.getActiveCIDHaltCount() + 1);
        } else if ("PAUSE".equals(severity)) {
            state.setActiveCIDPauseCount(state.getActiveCIDPauseCount() + 1);
        }
        state.setLastCIDAlertTimestamp(System.currentTimeMillis());
    }

    private void updateFromLabResult(Map<String, Object> payload, ClinicalStateSummary state) {
        // A1: Extract personalised targets from KB-20 enrichment (when available)
        extractPersonalizedTargets(payload, state);

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

    /**
     * A1: Extract personalised targets from KB-20 enrichment data.
     *
     * When KB-20 computes per-patient targets (based on age, CKD stage, diabetes
     * duration, comorbidity burden), they arrive in the enriched event payload under
     * the "kb20_personalized_targets" map. This method extracts them into the state
     * so that Module13StateChangeDetector and Module13CKMRiskComputer can use them
     * instead of hardcoded population defaults.
     *
     * Until KB-20 delivers these fields, the state values remain null and the
     * detectors fall back to hardcoded thresholds (de-risked deployment).
     */
    @SuppressWarnings("unchecked")
    private void extractPersonalizedTargets(Map<String, Object> payload, ClinicalStateSummary state) {
        if (!pilotConfig.isPersonalizedTargetsEnabled()) return;
        Object targetsObj = payload.get("kb20_personalized_targets");
        if (targetsObj == null) return;

        Map<String, Object> targets;
        if (targetsObj instanceof Map) {
            targets = (Map<String, Object>) targetsObj;
        } else {
            return;
        }

        Double fbgTarget = toDouble(targets.get("fbg_target"));
        if (fbgTarget != null) state.setPersonalizedFBGTarget(fbgTarget);

        Double hba1cTarget = toDouble(targets.get("hba1c_target"));
        if (hba1cTarget != null) state.setPersonalizedHbA1cTarget(hba1cTarget);

        Double sbpTarget = toDouble(targets.get("sbp_target"));
        if (sbpTarget != null) state.setPersonalizedSBPTarget(sbpTarget);

        Double egfrThreshold = toDouble(targets.get("egfr_threshold"));
        if (egfrThreshold != null) state.setPersonalizedEGFRThreshold(egfrThreshold);

        Double sbpKidneyThreshold = toDouble(targets.get("sbp_kidney_threshold"));
        if (sbpKidneyThreshold != null) state.setPersonalizedSBPKidneyThreshold(sbpKidneyThreshold);

        metrics.recordPersonalizedTargetsEnriched();
        LOG.debug("Personalised targets updated for patient {}: FBG={}, HbA1c={}, SBP={}, eGFR={}, SBP-kidney={}",
                state.getPatientId(), fbgTarget, hba1cTarget, sbpTarget, egfrThreshold, sbpKidneyThreshold);
    }

    private void emitDataAbsenceIfNeeded(ClinicalStateSummary state, Context ctx,
            Collector<ClinicalStateChangeEvent> out, ClinicalStateChangeType type,
            CKMRiskVelocity velocity) {
        Long lastEmitted = state.getLastEmittedChangeTimestamps().get(type);
        long now = ctx.timerService().currentProcessingTime();
        if (lastEmitted != null && (now - lastEmitted) < 24 * 3_600_000L) return;

        ClinicalStateChangeEvent absenceEvent = ClinicalStateChangeEvent.builder()
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
        if (!pilotConfig.isDryRun()) {
            out.collect(absenceEvent);
        }
        state.getLastEmittedChangeTimestamps().put(type, now);
        if (type == ClinicalStateChangeType.DATA_ABSENCE_CRITICAL) {
            metrics.recordDataAbsenceCritical();
        } else {
            metrics.recordDataAbsenceWarning();
        }
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
