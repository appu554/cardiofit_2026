package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Instant;
import java.time.ZoneOffset;
import java.time.ZonedDateTime;
import java.util.Optional;

/**
 * Module 9: Engagement Monitor.
 *
 * Timer-driven KeyedProcessFunction that tracks 8 engagement signals
 * (DD#8 reconciled) over a 14-day rolling window and emits daily
 * channel-aware composite scores.
 *
 * processElement(): Classifies events -> marks today's signal bitmap.
 *                   Extracts data_tier (from payload) and channel (Phase 1b: KB-20).
 * onTimer(): (Daily at 23:59 UTC processing time) Computes score -> emits
 *            signal -> detects drops (3-day persistence) -> advances day.
 *            Zombie check: stops timer chain after 21 days of no events.
 *
 * Input:  CanonicalEvent from enriched-patient-events-v1
 * Output: EngagementSignal to flink.engagement-signals (main)
 *         EngagementDropAlert to alerts.engagement-drop (side output)
 *
 * Review fixes: R1 (8 signals), R2 (channel-aware), R3 (3-day persistence),
 * R4 (zombie prevention), R5 (OpenContext), R6 (channel/dataTier), R7 (validHistoryDays).
 */
public class Module9_EngagementMonitor
        extends KeyedProcessFunction<String, CanonicalEvent, EngagementSignal> {

    private static final Logger LOG = LoggerFactory.getLogger(Module9_EngagementMonitor.class);

    private static final long DAY_MS = 86_400_000L;
    private static final int ZOMBIE_THRESHOLD_DAYS = 21;

    public static final OutputTag<EngagementDropAlert> ENGAGEMENT_DROP_TAG =
        new OutputTag<>("engagement-drop-alerts",
            TypeInformation.of(EngagementDropAlert.class));

    public static final OutputTag<RelapseRiskScore> RELAPSE_RISK_TAG =
        new OutputTag<>("relapse-risk-alerts",
            TypeInformation.of(RelapseRiskScore.class));

    private transient ValueState<EngagementState> engagementState;

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        ValueStateDescriptor<EngagementState> stateDesc =
            new ValueStateDescriptor<>("engagement-state", EngagementState.class);

        org.apache.flink.api.common.state.StateTtlConfig ttl =
            org.apache.flink.api.common.state.StateTtlConfig
                .newBuilder(java.time.Duration.ofDays(14))
                .setUpdateType(
                    org.apache.flink.api.common.state.StateTtlConfig.UpdateType.OnReadAndWrite)
                .setStateVisibility(
                    org.apache.flink.api.common.state.StateTtlConfig.StateVisibility.NeverReturnExpired)
                .build();
        stateDesc.enableTimeToLive(ttl);

        engagementState = getRuntimeContext().getState(stateDesc);
        LOG.info("Module 9 Engagement Monitor initialized (8 signals, channel-aware, 3-day persistence)");
    }

    @Override
    public void processElement(CanonicalEvent event, Context ctx,
                                Collector<EngagementSignal> out) throws Exception {
        EngagementState state = engagementState.value();
        if (state == null) {
            state = new EngagementState();
            state.setPatientId(event.getPatientId());
            LOG.debug("Initializing engagement state for patient={}", event.getPatientId());
        }

        // R6: Extract data_tier from first event's payload (set by Module 1b canonicalizer)
        if (state.getDataTier() == null && event.getPayload() != null) {
            Object dataTier = event.getPayload().get("data_tier");
            if (dataTier instanceof String) {
                state.setDataTier((String) dataTier);
                LOG.debug("Set data_tier={} for patient={}", dataTier, event.getPatientId());
            }
        }

        // Classify event into signal channel
        SignalType signal = Module9SignalClassifier.classify(event);

        if (signal != null) {
            state.markSignalToday(signal);
        }

        // Phase 2: Extract trajectory features from event payload
        extractTrajectoryFeatures(event, state);

        // Register daily timer (processing time, once per patient)
        if (!state.isDailyTimerRegistered()) {
            long nextMidnight = computeNextDailyTick(ctx.timerService().currentProcessingTime());
            ctx.timerService().registerProcessingTimeTimer(nextMidnight);
            state.setDailyTimerRegistered(true);
            LOG.debug("Registered daily timer for patient={} at {}",
                      event.getPatientId(), Instant.ofEpochMilli(nextMidnight));
        }

        state.setTotalEventsProcessed(state.getTotalEventsProcessed() + 1);
        state.setLastUpdated(ctx.timerService().currentProcessingTime());

        engagementState.update(state);
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx,
                         Collector<EngagementSignal> out) throws Exception {
        EngagementState state = engagementState.value();
        if (state == null) return;

        // R4: Zombie state prevention — stop timer chain after 21 days of silence
        long daysSinceLastEvent = (timestamp - state.getLastUpdated()) / DAY_MS;
        if (daysSinceLastEvent > ZOMBIE_THRESHOLD_DAYS) {
            LOG.info("Stopping timer chain for zombie patient={} ({}d since last event)",
                     state.getPatientId(), daysSinceLastEvent);

            Module9ScoreComputer.Result finalResult = Module9ScoreComputer.compute(state);
            EngagementSignal finalSignal = EngagementSignal.create(
                state.getPatientId(),
                finalResult.compositeScore,
                EngagementLevel.RED,
                finalResult.densities,
                "DISENGAGED_TERMINATED",
                state.getPreviousScore(),
                state.getConsecutiveLowDays()
            );
            finalSignal.setCorrelationId("zombie-termination-" + state.getPatientId());
            finalSignal.setChannel(state.getChannel());
            finalSignal.setDataTier(state.getDataTier());
            out.collect(finalSignal);

            engagementState.update(state);
            return; // Do NOT re-register timer
        }

        // 1. Compute today's composite score (R2: channel-aware)
        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);

        // 2. Create engagement signal
        EngagementSignal signal = EngagementSignal.create(
            state.getPatientId(),
            result.compositeScore,
            result.level,
            result.densities,
            result.phenotype,
            state.getPreviousScore(),
            state.getConsecutiveLowDays()
        );
        signal.setChannel(state.getChannel());
        signal.setDataTier(state.getDataTier());

        // 2b. Phase 2: Trajectory analysis — populate relapse risk on signal BEFORE emit
        Optional<RelapseRiskScore> relapseRisk = Module9TrajectoryAnalyzer.analyze(state, result);
        if (relapseRisk.isPresent()) {
            RelapseRiskScore risk = relapseRisk.get();
            signal.setRelapseRiskScore(risk.getRelapseRiskScore());
        }

        // 3. Emit main output (signal fully populated including relapseRiskScore)
        out.collect(signal);

        // 3b. Phase 2: Emit relapse risk side output (after main signal)
        if (relapseRisk.isPresent() && Module9TrajectoryAnalyzer.isAlertWorthy(relapseRisk.get())) {
            ctx.output(RELAPSE_RISK_TAG, relapseRisk.get());
            LOG.info("Relapse risk alert for patient={}: tier={}, score={}",
                     state.getPatientId(), relapseRisk.get().getRiskTier(),
                     relapseRisk.get().getRelapseRiskScore());
        }

        // R3: Track consecutive days at current level (for 3-day persistence)
        if (state.getPreviousLevel() != null && result.level == state.getPreviousLevel()) {
            state.setConsecutiveDaysAtCurrentLevel(
                state.getConsecutiveDaysAtCurrentLevel() + 1);
        } else {
            state.setConsecutiveDaysAtCurrentLevel(1);
        }

        // 4. Detect engagement drops (R3: pass consecutiveDaysAtLevel)
        Optional<EngagementDropAlert> alert = Module9DropDetector.detect(
            state.getPatientId(),
            result.compositeScore,
            result.level,
            state.getPreviousScore(),
            state.getPreviousLevel(),
            state.getConsecutiveLowDays(),
            state.getConsecutiveDaysAtCurrentLevel(),
            state,
            timestamp
        );

        if (alert.isPresent()) {
            EngagementDropAlert dropAlert = alert.get();
            ctx.output(ENGAGEMENT_DROP_TAG, dropAlert);
            state.recordAlertEmission(dropAlert.getSuppressionKey(), timestamp);
            LOG.info("Engagement drop alert for patient={}: {} (score={}, channel={})",
                     state.getPatientId(), dropAlert.getDropType(),
                     result.compositeScore, state.getChannel());
        }

        // 5. Update consecutive low day counter
        if (result.level.isAlertWorthy()) {
            state.setConsecutiveLowDays(state.getConsecutiveLowDays() + 1);
        } else {
            state.setConsecutiveLowDays(0);
        }

        // 6. Advance the rolling window
        state.advanceDay(result.compositeScore);

        // 6b. Phase 2: Flush trajectory features into 7-day buffers and reset
        state.flushTrajectoryAndReset();

        // 7. Store previous score/level
        state.setPreviousScore(result.compositeScore);
        state.setPreviousLevel(result.level);

        // 8. Re-register next daily timer
        long nextTick = computeNextDailyTick(timestamp);
        ctx.timerService().registerProcessingTimeTimer(nextTick);
        state.setLastDailyTickTimestamp(timestamp);

        engagementState.update(state);

        LOG.debug("Daily tick for patient={}: score={}, level={}, phenotype={}, channel={}",
                  state.getPatientId(), result.compositeScore, result.level,
                  result.phenotype, state.getChannel());
    }

    /**
     * Phase 2: Extract trajectory features from event payload.
     *
     * Feature extraction mapping:
     *   steps         ← DEVICE_READING (step_count), normalized 0-1 (10000 steps = 1.0)
     *   mealQuality   ← PATIENT_REPORTED/MEAL_LOG (carb_grams / protein_grams ratio, inverted)
     *   latency       ← PATIENT_REPORTED/APP_SESSION (session_duration_sec, normalized)
     *   checkin       ← PATIENT_REPORTED/GOAL_COMPLETED (fields_completed/total_fields)
     *   protein       ← PATIENT_REPORTED/MEAL_LOG (protein_flag: 1.0 if present, 0.0 if not)
     *
     * Each feature uses the LAST value seen today (overwrite on duplicate).
     */
    private void extractTrajectoryFeatures(CanonicalEvent event, EngagementState state) {
        if (event.getPayload() == null) return;
        java.util.Map<String, Object> payload = event.getPayload();
        EventType type = event.getEventType();

        if (type == EventType.DEVICE_READING) {
            // Steps from wearable
            Object stepCount = payload.get("step_count");
            if (stepCount instanceof Number) {
                double normalized = Math.min(1.0, ((Number) stepCount).doubleValue() / 10000.0);
                state.setTodaySteps(normalized);
            }
        } else if (type == EventType.PATIENT_REPORTED) {
            String reportType = payload.get("report_type") instanceof String
                ? ((String) payload.get("report_type")).toUpperCase() : "";

            if ("MEAL_LOG".equals(reportType)) {
                // Meal quality: lower carb/protein ratio = better quality
                Object carbGrams = payload.get("carb_grams");
                Object proteinGrams = payload.get("protein_grams");
                if (carbGrams instanceof Number && proteinGrams instanceof Number) {
                    double carbs = ((Number) carbGrams).doubleValue();
                    double protein = ((Number) proteinGrams).doubleValue();
                    if (protein > 0) {
                        // Ratio 1:1 or better = 1.0, ratio 5:1 = 0.0
                        double ratio = carbs / protein;
                        double quality = Math.max(0.0, Math.min(1.0, 1.0 - (ratio - 1.0) / 4.0));
                        state.setTodayMealQuality(quality);
                    }
                }

                // Protein adherence: any protein in meal = 1.0
                Object proteinFlag = payload.get("protein_flag");
                if (proteinFlag instanceof Boolean) {
                    state.setTodayProteinAdherence(((Boolean) proteinFlag) ? 1.0 : 0.0);
                } else if (proteinGrams instanceof Number && ((Number) proteinGrams).doubleValue() > 0) {
                    state.setTodayProteinAdherence(1.0);
                }

            } else if ("APP_SESSION".equals(reportType)) {
                // Response latency: session duration (normalized, 300s = 1.0 max)
                Object duration = payload.get("session_duration_sec");
                if (duration instanceof Number) {
                    double normalized = Math.min(1.0, ((Number) duration).doubleValue() / 300.0);
                    state.setTodayResponseLatency(normalized);
                }

            } else if ("GOAL_COMPLETED".equals(reportType)) {
                // Check-in completeness: fields_completed / total_fields
                Object completed = payload.get("fields_completed");
                Object total = payload.get("total_fields");
                if (completed instanceof Number && total instanceof Number) {
                    double t = ((Number) total).doubleValue();
                    if (t > 0) {
                        double completeness = Math.min(1.0,
                            ((Number) completed).doubleValue() / t);
                        state.setTodayCheckinCompleteness(completeness);
                    }
                } else {
                    // Goal completed without fields info → full credit
                    state.setTodayCheckinCompleteness(1.0);
                }
            }
        }
    }

    /**
     * Compute next daily tick at 23:59 UTC.
     * If current time IS 23:59, schedule for next day.
     */
    static long computeNextDailyTick(long currentTimeMs) {
        ZonedDateTime now = Instant.ofEpochMilli(currentTimeMs).atZone(ZoneOffset.UTC);
        ZonedDateTime todayTick = now.toLocalDate().atTime(23, 59).atZone(ZoneOffset.UTC);

        if (now.isAfter(todayTick) || now.isEqual(todayTick)) {
            todayTick = todayTick.plusDays(1);
        }

        return todayTick.toInstant().toEpochMilli();
    }
}
