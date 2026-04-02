package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.state.StateTtlConfig;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Module 10: Meal Response Correlator — main operator.
 *
 * Session-window-driven KeyedProcessFunction:
 * - MEAL event (PATIENT_REPORTED + report_type=MEAL_LOG) → OPENS a session window
 * - DEVICE_READING (CGM) or LAB_RESULT (SMBG glucose) → FILLS the glucose window
 * - VITAL_SIGN (BP) → FILLS the BP window + buffers as pre-meal for future meals
 * - Processing-time timer at meal+3h05m → CLOSES the window and emits MealResponseRecord
 *
 * Keyed by patientId. Input: CanonicalEvent from enriched-patient-events-v1.
 * Output: MealResponseRecord to flink.meal-response (main output).
 *
 * Design decisions:
 * - Processing-time timers (not event-time) ensure window closes even if CGM goes offline.
 *   CAVEAT: during Kafka offset resets or batch replay, windows close immediately (wall-clock,
 *   not 3h of event time), producing near-empty glucose windows. For backfill, consider
 *   adding an event-time mode with event-time timers.
 * - 5-minute grace period: timer at 3h05m not 3h00m for late-arriving readings
 * - Overlapping meals: if second meal within 90 min, both flagged, first window truncated
 * - Pre-meal BP: retroactive buffer — most recent BP within 60 min before meal
 * - Data tier: inferred from first glucose source (CGM→Tier1, SMBG→Tier3)
 *
 * State TTL: 7 days (covers meal windows + some buffer).
 */
public class Module10_MealResponseCorrelator
        extends KeyedProcessFunction<String, CanonicalEvent, MealResponseRecord> {

    private static final Logger LOG = LoggerFactory.getLogger(Module10_MealResponseCorrelator.class);

    // Side output for meal records that also feed Module 10b
    public static final OutputTag<MealResponseRecord> MEAL_PATTERN_FEED_TAG =
        new OutputTag<>("meal-pattern-feed",
            TypeInformation.of(MealResponseRecord.class));

    private transient ValueState<MealCorrelationState> correlationState;

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        ValueStateDescriptor<MealCorrelationState> stateDesc =
            new ValueStateDescriptor<>("meal-correlation-state", MealCorrelationState.class);
        StateTtlConfig ttl = StateTtlConfig
            .newBuilder(Duration.ofDays(7))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();
        stateDesc.enableTimeToLive(ttl);
        correlationState = getRuntimeContext().getState(stateDesc);

        LOG.info("Module 10 Meal Response Correlator initialized (session-window, 3-tier glucose)");
    }

    @Override
    public void processElement(CanonicalEvent event, Context ctx,
                                Collector<MealResponseRecord> out) throws Exception {
        MealCorrelationState state = correlationState.value();
        if (state == null) {
            state = new MealCorrelationState(event.getPatientId());
        }

        EventType eventType = event.getEventType();
        Map<String, Object> payload = event.getPayload();
        if (payload == null) {
            correlationState.update(state);
            return;
        }

        // Route by event type
        if (isMealEvent(eventType, payload)) {
            handleMealEvent(event, state, ctx);
        } else if (isGlucoseReading(eventType, payload)) {
            handleGlucoseReading(event, state);
        } else if (eventType == EventType.VITAL_SIGN) {
            handleBPReading(event, state);
        }

        state.setLastUpdated(ctx.timerService().currentProcessingTime());
        correlationState.update(state);
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx,
                         Collector<MealResponseRecord> out) throws Exception {
        MealCorrelationState state = correlationState.value();
        if (state == null) return;

        // Find sessions whose timer fires at this timestamp
        List<String> sessionIds = state.getSessionsForTimer(timestamp);

        for (String mealEventId : sessionIds) {
            MealCorrelationState.MealSession session = state.closeSession(mealEventId);
            if (session == null) continue;

            MealResponseRecord record = buildRecord(state, session);
            out.collect(record);

            // Also emit to side output for Module 10b consumption
            ctx.output(MEAL_PATTERN_FEED_TAG, record);

            LOG.debug("Meal session closed: patient={}, meal={}, tier={}, glucose={}, bp={}",
                state.getPatientId(), mealEventId, state.getDataTier(),
                record.getGlucoseReadingCount(), record.isBpComplete());
        }

        correlationState.update(state);
    }

    // --- Event Classification ---

    private boolean isMealEvent(EventType type, Map<String, Object> payload) {
        if (type != EventType.PATIENT_REPORTED) return false;
        Object reportType = payload.get("report_type");
        return "MEAL_LOG".equalsIgnoreCase(reportType != null ? reportType.toString() : "");
    }

    private boolean isGlucoseReading(EventType type, Map<String, Object> payload) {
        if (type == EventType.DEVICE_READING) {
            return payload.containsKey("glucose_value");
        }
        if (type == EventType.LAB_RESULT) {
            Object labType = payload.get("lab_type");
            return "glucose".equalsIgnoreCase(labType != null ? labType.toString() : "");
        }
        return false;
    }

    // --- Event Handlers ---

    private void handleMealEvent(CanonicalEvent event, MealCorrelationState state, Context ctx) {
        String mealEventId = event.getId() != null ? event.getId() : UUID.randomUUID().toString();
        long mealTimestamp = event.getEventTime();

        long timerFireTime = state.openSession(mealEventId, mealTimestamp, event.getPayload());

        // Register processing-time timer for window close
        ctx.timerService().registerProcessingTimeTimer(timerFireTime);

        LOG.debug("Meal session opened: patient={}, meal={}, timerAt={}",
            state.getPatientId(), mealEventId, timerFireTime);
    }

    private void handleGlucoseReading(CanonicalEvent event, MealCorrelationState state) {
        Map<String, Object> payload = event.getPayload();
        long timestamp = event.getEventTime();

        double glucoseValue;
        String source;

        if (event.getEventType() == EventType.DEVICE_READING) {
            Object val = payload.get("glucose_value");
            if (!(val instanceof Number)) return;
            glucoseValue = ((Number) val).doubleValue();
            source = "CGM";

            if (state.getDataTier() == DataTier.TIER_3_SMBG) {
                state.setDataTier(DataTier.TIER_1_CGM);
            }
        } else {
            Object val = payload.get("value");
            if (!(val instanceof Number)) return;
            glucoseValue = ((Number) val).doubleValue();
            source = "SMBG";

            if (state.getDataTier() == DataTier.TIER_1_CGM) {
                state.setDataTier(DataTier.TIER_2_HYBRID);
            }
        }

        state.addGlucoseReading(timestamp, glucoseValue, source);
    }

    private void handleBPReading(CanonicalEvent event, MealCorrelationState state) {
        Map<String, Object> payload = event.getPayload();
        Object sbpObj = payload.get("systolic_bp");
        Object dbpObj = payload.get("diastolic_bp");

        if (!(sbpObj instanceof Number)) return;
        double sbp = ((Number) sbpObj).doubleValue();
        double dbp = (dbpObj instanceof Number) ? ((Number) dbpObj).doubleValue() : 0.0;

        state.addBPReading(event.getEventTime(), sbp, dbp);
    }

    // --- Record Building ---

    private MealResponseRecord buildRecord(MealCorrelationState state,
                                            MealCorrelationState.MealSession session) {
        // Analyze glucose
        Module10GlucoseAnalyzer.Result glucoseResult =
            Module10GlucoseAnalyzer.analyze(session.glucoseWindow);

        // Classify curve shape (Tier 1 only)
        CurveShape curveShape = CurveShape.UNKNOWN;
        if (state.getDataTier().supportsCurveClassification() && glucoseResult != null) {
            curveShape = Module10CurveClassifier.classify(session.glucoseWindow);
        }

        // Analyze BP
        Module10BPCorrelator.Result bpResult =
            Module10BPCorrelator.analyze(session.bpWindow);

        // Extract meal metadata from payload
        Map<String, Object> mealPayload = session.mealPayload;
        Double carbGrams = getDoubleField(mealPayload, "carb_grams");
        Double proteinGrams = getDoubleField(mealPayload, "protein_grams");
        Double sodiumMg = getDoubleField(mealPayload, "sodium_mg");

        // Compute quality score
        double qualityScore = computeQualityScore(glucoseResult, bpResult, state.getDataTier());

        MealResponseRecord.Builder builder = MealResponseRecord.builder()
            .recordId("m10-" + UUID.randomUUID())
            .patientId(state.getPatientId())
            .mealEventId(session.mealEventId)
            .mealTimestamp(session.mealTimestamp)
            .mealTimeCategory(MealTimeCategory.fromTimestamp(session.mealTimestamp))
            .carbGrams(carbGrams)
            .proteinGrams(proteinGrams)
            .sodiumMg(sodiumMg)
            .dataTier(state.getDataTier())
            .overlapping(session.overlapping)
            .mealPayload(mealPayload)
            .qualityScore(qualityScore);

        // Glucose features
        if (glucoseResult != null) {
            builder.glucoseBaseline(glucoseResult.baseline)
                   .glucosePeak(glucoseResult.peak)
                   .glucoseExcursion(glucoseResult.excursion)
                   .timeToPeakMin(glucoseResult.timeToPeakMin)
                   .iAUC(glucoseResult.iAUC)
                   .recoveryTimeMin(glucoseResult.recoveryTimeMin)
                   .glucoseReadingCount(glucoseResult.readingCount);
        }
        builder.curveShape(curveShape);

        // BP features
        if (bpResult != null) {
            builder.preMealSBP(bpResult.preMealSBP)
                   .postMealSBP(bpResult.postMealSBP)
                   .sbpExcursion(bpResult.sbpExcursion)
                   .bpComplete(bpResult.complete);
        }

        // Window duration: use deterministic session boundaries, not wall-clock.
        // timerFireTime = mealTimestamp + GLUCOSE_WINDOW_MS + GLUCOSE_GRACE_MS (3h05m)
        long windowDuration = session.timerFireTime - session.mealTimestamp;
        builder.windowDurationMs(windowDuration);

        return builder.build();
    }

    private double computeQualityScore(Module10GlucoseAnalyzer.Result glucose,
                                        Module10BPCorrelator.Result bp, DataTier tier) {
        double score = 0.0;
        if (glucose != null) {
            score += 0.5 * glucose.qualityScore;
        }
        if (bp != null && bp.complete) {
            score += 0.3;
        } else if (bp != null && (bp.preMealSBP != null || bp.postMealSBP != null)) {
            score += 0.15;
        }
        if (tier == DataTier.TIER_1_CGM) {
            score += 0.2;
        } else if (tier == DataTier.TIER_2_HYBRID) {
            score += 0.1;
        }
        return Math.min(1.0, score);
    }

    private static Double getDoubleField(Map<String, Object> payload, String key) {
        if (payload == null) return null;
        Object val = payload.get(key);
        if (val instanceof Number) return ((Number) val).doubleValue();
        if (val instanceof String) {
            try { return Double.parseDouble((String) val); }
            catch (NumberFormatException e) { return null; }
        }
        return null;
    }
}
