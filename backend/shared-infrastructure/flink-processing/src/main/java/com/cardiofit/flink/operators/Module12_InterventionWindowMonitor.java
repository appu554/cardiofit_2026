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
import java.util.*;

/**
 * Module 12: Intervention Window Monitor — main operator.
 *
 * Event-driven timer-based KeyedProcessFunction keyed by patientId.
 * Consumes from two sources (unioned via discriminator):
 * 1. clinical.intervention-events → OPENS/MODIFIES/CANCELS observation windows
 * 2. enriched-patient-events-v1 → trajectory tracking + confounder detection
 *
 * Processing-time timers fire at MIDPOINT and CLOSE (+24h grace).
 * Emits InterventionWindowSignal to clinical.intervention-window-signals.
 *
 * State TTL: 90 days.
 */
public class Module12_InterventionWindowMonitor
        extends KeyedProcessFunction<String, CanonicalEvent, InterventionWindowSignal> {

    private static final Logger LOG =
            LoggerFactory.getLogger(Module12_InterventionWindowMonitor.class);

    private transient ValueState<InterventionWindowState> windowState;

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        ValueStateDescriptor<InterventionWindowState> stateDesc =
                new ValueStateDescriptor<>("intervention-window-state",
                        InterventionWindowState.class);

        StateTtlConfig ttl = StateTtlConfig
                .newBuilder(Duration.ofDays(90))
                .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
                .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
                .build();
        stateDesc.enableTimeToLive(ttl);
        windowState = getRuntimeContext().getState(stateDesc);

        LOG.info("Module 12 Intervention Window Monitor initialized (90-day TTL, dual-source)");
    }

    @Override
    public void processElement(CanonicalEvent event, Context ctx,
                               Collector<InterventionWindowSignal> out) throws Exception {
        InterventionWindowState state = windowState.value();
        if (state == null) {
            state = new InterventionWindowState(event.getPatientId());
        }

        Map<String, Object> payload = event.getPayload();
        if (payload == null) {
            windowState.update(state);
            return;
        }

        // Route: is this an intervention lifecycle event or a patient data event?
        String interventionEventType = getStr(payload, "event_type");

        if ("INTERVENTION_APPROVED".equals(interventionEventType)) {
            handleInterventionApproved(event, state, ctx, out);
        } else if ("INTERVENTION_MODIFIED".equals(interventionEventType)) {
            handleInterventionModified(event, state, ctx);
        } else if ("INTERVENTION_CANCELLED".equals(interventionEventType)) {
            handleInterventionCancelled(event, state, ctx, out);
        } else {
            // Patient data event → trajectory tracking + confounder detection
            handlePatientDataEvent(event, state);
        }

        state.setLastUpdated(ctx.timerService().currentProcessingTime());
        windowState.update(state);
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx,
                        Collector<InterventionWindowSignal> out) throws Exception {
        InterventionWindowState state = windowState.value();
        if (state == null) return;

        // Check if this is a midpoint timer
        InterventionWindowState.InterventionWindow midpointWindow =
                state.getWindowForMidpointTimer(timestamp);
        if (midpointWindow != null) {
            handleMidpointTimer(midpointWindow, state, ctx, out);
            windowState.update(state);
            return;
        }

        // Check if this is a close timer
        InterventionWindowState.InterventionWindow closeWindow =
                state.getWindowForCloseTimer(timestamp);
        if (closeWindow != null) {
            handleCloseTimer(closeWindow, state, ctx, out);
            windowState.update(state);
            return;
        }

        // Timer for a cancelled/removed window — no-op
        LOG.debug("Timer fired for unknown/cancelled window at {}", timestamp);
    }

    // --- Intervention Lifecycle Handlers ---

    private void handleInterventionApproved(CanonicalEvent event,
                                             InterventionWindowState state,
                                             Context ctx,
                                             Collector<InterventionWindowSignal> out) {
        Map<String, Object> payload = event.getPayload();
        String interventionId = getStr(payload, "intervention_id");
        String typeStr = getStr(payload, "intervention_type");
        InterventionType interventionType = InterventionType.valueOf(typeStr);

        @SuppressWarnings("unchecked")
        Map<String, Object> detail = (Map<String, Object>) payload.get("intervention_detail");

        int windowDays = getInt(payload, "observation_window_days",
                interventionType.getDefaultWindowDays());

        long now = ctx.timerService().currentProcessingTime();

        // Compute trajectory at open
        TrajectoryClassification trajectoryAtOpen = Module12TrajectoryTracker.classifyComposite(
                state, now - 14L * 86_400_000L, now);

        // Open window
        InterventionWindowState.InterventionWindow window = state.openWindow(
                interventionId, interventionType, detail, windowDays, now,
                trajectoryAtOpen,
                getStr(payload, "originating_card_id"),
                getStr(payload, "physician_action"));

        // Detect concurrent interventions
        Module12ConcurrencyDetector.Result concurrency = Module12ConcurrencyDetector.detect(
                interventionId, interventionType, detail,
                window.observationStartMs, window.observationEndMs,
                state.getActiveWindows());

        window.concurrentInterventionIds.addAll(concurrency.getConcurrentIds());

        // Cross-reference: add this intervention to existing concurrent windows
        // and emit CONCURRENCY_UPDATED so downstream consumers see the updated count
        for (String concurrentId : concurrency.getConcurrentIds()) {
            InterventionWindowState.InterventionWindow existing = state.getWindow(concurrentId);
            if (existing != null && !existing.concurrentInterventionIds.contains(interventionId)) {
                existing.concurrentInterventionIds.add(interventionId);
                // Retroactive signal: notify downstream that this window now has a new concurrent
                out.collect(buildSignal(existing, state,
                        InterventionWindowSignalType.CONCURRENCY_UPDATED,
                        existing.trajectoryAtOpen, null));
                LOG.info("Concurrency updated: existing intervention={} now concurrent with {}",
                        concurrentId, interventionId);
            }
        }

        // Register processing-time timers
        ctx.timerService().registerProcessingTimeTimer(window.midpointTimerMs);
        ctx.timerService().registerProcessingTimeTimer(window.closeTimerMs);

        // Emit WINDOW_OPENED signal
        out.collect(buildSignal(window, state,
                InterventionWindowSignalType.WINDOW_OPENED,
                trajectoryAtOpen, concurrency));

        LOG.info("Window opened: patient={}, intervention={}, type={}, window={}d, concurrent={}",
                state.getPatientId(), interventionId, interventionType,
                windowDays, concurrency.getConcurrentIds().size());
    }

    private void handleInterventionModified(CanonicalEvent event,
                                             InterventionWindowState state,
                                             Context ctx) {
        Map<String, Object> payload = event.getPayload();
        String interventionId = getStr(payload, "intervention_id");

        InterventionWindowState.InterventionWindow window = state.getWindow(interventionId);
        if (window == null) {
            LOG.warn("MODIFIED event for unknown intervention: {}", interventionId);
            return;
        }

        // Log modification as a confounder
        window.confoundersDetected.add("INTERVENTION_MODIFIED");

        // If window days changed, re-register timers
        int newWindowDays = getInt(payload, "observation_window_days", window.observationWindowDays);
        if (newWindowDays != window.observationWindowDays) {
            // Delete old timers
            ctx.timerService().deleteProcessingTimeTimer(window.midpointTimerMs);
            ctx.timerService().deleteProcessingTimeTimer(window.closeTimerMs);

            // Recompute
            long windowMs = newWindowDays * 24L * 60 * 60 * 1000;
            long gracePeriodMs = 24L * 60 * 60 * 1000;
            window.observationEndMs = window.observationStartMs + windowMs;
            window.observationWindowDays = newWindowDays;
            window.midpointTimerMs = window.observationStartMs + windowMs / 2;
            window.closeTimerMs = window.observationEndMs + gracePeriodMs;

            // Register new timers
            ctx.timerService().registerProcessingTimeTimer(window.midpointTimerMs);
            ctx.timerService().registerProcessingTimeTimer(window.closeTimerMs);

            LOG.info("Window modified: intervention={}, new window={}d", interventionId, newWindowDays);
        }
    }

    private void handleInterventionCancelled(CanonicalEvent event,
                                              InterventionWindowState state,
                                              Context ctx,
                                              Collector<InterventionWindowSignal> out) {
        Map<String, Object> payload = event.getPayload();
        String interventionId = getStr(payload, "intervention_id");

        InterventionWindowState.InterventionWindow window = state.getWindow(interventionId);
        if (window == null) {
            LOG.warn("CANCELLED event for unknown intervention: {}", interventionId);
            return;
        }

        // Delete timers
        ctx.timerService().deleteProcessingTimeTimer(window.midpointTimerMs);
        ctx.timerService().deleteProcessingTimeTimer(window.closeTimerMs);

        // Mark cancelled and emit signal
        window.status = "CANCELLED";
        out.collect(buildSignal(window, state,
                InterventionWindowSignalType.WINDOW_CANCELLED,
                TrajectoryClassification.UNKNOWN, null));

        // Remove from active windows
        state.removeWindow(interventionId);

        LOG.info("Window cancelled: patient={}, intervention={}",
                state.getPatientId(), interventionId);
    }

    // --- Patient Data Handler ---

    private void handlePatientDataEvent(CanonicalEvent event,
                                         InterventionWindowState state) {
        EventType eventType = event.getEventType();
        Map<String, Object> payload = event.getPayload();

        // Update trajectory readings
        if (eventType == EventType.LAB_RESULT) {
            String labType = getStr(payload, "lab_type");
            Double value = getDouble(payload, "value");
            if (labType != null && value != null) {
                state.addReading(labType, value, event.getEventTime());
                if ("FBG".equals(labType)) state.setLastKnownFBG(value);
                if ("EGFR".equals(labType)) state.setLastKnownEGFR(value);
                if ("HBA1C".equals(labType)) state.setLastKnownHbA1c(value);
            }
        } else if (eventType == EventType.VITAL_SIGN) {
            Double sbp = getDouble(payload, "systolic_bp");
            Double dbp = getDouble(payload, "diastolic_bp");
            if (sbp != null) {
                state.addReading("SBP", sbp, event.getEventTime());
                state.setLastKnownSBP(sbp);
            }
            if (dbp != null) state.setLastKnownDBP(dbp);
            Double weight = getDouble(payload, "weight_kg");
            if (weight != null) {
                state.addReading("WEIGHT", weight, event.getEventTime());
                state.setLastKnownWeight(weight);
            }
        } else if (eventType == EventType.DEVICE_READING) {
            Double glucose = getDouble(payload, "glucose_value");
            if (glucose != null) {
                state.addReading("FBG", glucose, event.getEventTime());
            }
        }

        // Accumulate confounders for all active windows
        for (InterventionWindowState.InterventionWindow window : state.getActiveWindows().values()) {
            if ("OBSERVING".equals(window.status)) {
                Module12ConfounderAccumulator.accumulate(window, event);
            }
        }
    }

    // --- Timer Handlers ---

    private void handleMidpointTimer(InterventionWindowState.InterventionWindow window,
                                      InterventionWindowState state,
                                      OnTimerContext ctx,
                                      Collector<InterventionWindowSignal> out) {
        long now = ctx.timerService().currentProcessingTime();

        // Compute trajectory during window so far
        TrajectoryClassification trajectoryDuring = Module12TrajectoryTracker.classifyComposite(
                state, window.observationStartMs, now);

        // Assemble preliminary adherence
        Module12AdherenceAssembler.Result adherence = Module12AdherenceAssembler.assemble(
                window.interventionType, window.adherenceSignals);
        Map<String, Object> adherenceMap = new HashMap<>();
        adherenceMap.put("score", adherence.getAdherenceScore());
        adherenceMap.put("data_quality", adherence.getDataQuality());
        window.adherenceSignals = adherenceMap;

        out.collect(buildSignal(window, state,
                InterventionWindowSignalType.WINDOW_MIDPOINT,
                trajectoryDuring, null));

        LOG.debug("Midpoint signal: intervention={}, trajectory={}, adherence={}",
                window.interventionId, trajectoryDuring, adherence.getAdherenceScore());
    }

    private void handleCloseTimer(InterventionWindowState.InterventionWindow window,
                                   InterventionWindowState state,
                                   OnTimerContext ctx,
                                   Collector<InterventionWindowSignal> out) {
        long now = ctx.timerService().currentProcessingTime();

        // Compute final trajectory during window
        TrajectoryClassification trajectoryDuring = Module12TrajectoryTracker.classifyComposite(
                state, window.observationStartMs, now);

        // Data completeness indicators
        window.dataCompleteness.put("has_bp", state.getLastKnownSBP() != null);
        window.dataCompleteness.put("has_fbg", state.getLastKnownFBG() != null);
        window.dataCompleteness.put("has_weight", state.getLastKnownWeight() != null);
        window.dataCompleteness.put("has_hba1c", state.getLastKnownHbA1c() != null);

        out.collect(buildSignal(window, state,
                InterventionWindowSignalType.WINDOW_CLOSED,
                trajectoryDuring, null));

        // Remove window from active state
        state.removeWindow(window.interventionId);

        LOG.info("Window closed: intervention={}, trajectory={}, confounders={}",
                window.interventionId, trajectoryDuring, window.confoundersDetected.size());
    }

    // --- Signal Builder ---

    private InterventionWindowSignal buildSignal(InterventionWindowState.InterventionWindow window,
                                                  InterventionWindowState state,
                                                  InterventionWindowSignalType signalType,
                                                  TrajectoryClassification trajectory,
                                                  Module12ConcurrencyDetector.Result concurrency) {
        return InterventionWindowSignal.builder()
                .signalId(UUID.randomUUID().toString())
                .interventionId(window.interventionId)
                .patientId(state.getPatientId())
                .signalType(signalType)
                .interventionType(window.interventionType)
                .interventionDetail(window.interventionDetail)
                .observationStartMs(window.observationStartMs)
                .observationEndMs(window.observationEndMs)
                .observationWindowDays(window.observationWindowDays)
                .trajectoryAtSignal(trajectory)
                .concurrentInterventionIds(window.concurrentInterventionIds)
                .concurrentInterventionCount(window.concurrentInterventionIds.size())
                .sameDomainConcurrent(concurrency != null && concurrency.isSameDomainConcurrent())
                .adherenceSignalsAtMidpoint(window.adherenceSignals)
                .confoundersDetected(window.confoundersDetected)
                .labChangesDuringWindow(window.labChanges)
                .externalEvents(window.externalEvents)
                .dataCompletenessIndicators(window.dataCompleteness)
                .processingTimestamp(System.currentTimeMillis())
                .originatingCardId(window.originatingCardId)
                .build();
    }

    // --- Helpers ---

    private static String getStr(Map<String, Object> m, String key) {
        Object v = m.get(key);
        return v != null ? v.toString() : null;
    }

    private static Double getDouble(Map<String, Object> m, String key) {
        Object v = m.get(key);
        if (v instanceof Number) return ((Number) v).doubleValue();
        return null;
    }

    private static int getInt(Map<String, Object> m, String key, int defaultVal) {
        Object v = m.get(key);
        if (v instanceof Number) return ((Number) v).intValue();
        return defaultVal;
    }
}
