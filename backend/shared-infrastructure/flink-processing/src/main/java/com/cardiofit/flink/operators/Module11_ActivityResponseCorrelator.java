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
 * Module 11: Activity Response Correlator — main operator.
 *
 * Exercise-session-window-driven KeyedProcessFunction:
 * - ACTIVITY event (PATIENT_REPORTED + report_type=ACTIVITY_LOG) → OPENS session
 * - DEVICE_READING (HR) → FILLS the HR window
 * - DEVICE_READING (CGM) or LAB_RESULT (glucose) → FILLS glucose window
 * - VITAL_SIGN (BP) → Captures pre/peak/post BP + buffers for future sessions
 * - VITAL_SIGN (resting_hr) → Updates resting HR baseline
 * - Processing-time timer at activity_end + 2h + 5min → CLOSES and emits
 *
 * Session window duration: activity_duration + 2h recovery + 5min grace.
 * Capped at 6h05m total. Default activity duration: 30 min if unspecified.
 *
 * Keyed by patientId. Input: CanonicalEvent from enriched-patient-events-v1.
 * Output: ActivityResponseRecord to flink.activity-response (main output).
 *
 * State TTL: 7 days.
 */
public class Module11_ActivityResponseCorrelator
        extends KeyedProcessFunction<String, CanonicalEvent, ActivityResponseRecord> {

    private static final Logger LOG =
            LoggerFactory.getLogger(Module11_ActivityResponseCorrelator.class);

    public static final OutputTag<ActivityResponseRecord> FITNESS_PATTERN_FEED_TAG =
            new OutputTag<>("fitness-pattern-feed",
                    TypeInformation.of(ActivityResponseRecord.class));

    private transient ValueState<ActivityCorrelationState> correlationState;

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        ValueStateDescriptor<ActivityCorrelationState> stateDesc =
                new ValueStateDescriptor<>("activity-correlation-state", ActivityCorrelationState.class);

        StateTtlConfig ttl = StateTtlConfig
                .newBuilder(Duration.ofDays(7))
                .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
                .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
                .build();
        stateDesc.enableTimeToLive(ttl);
        correlationState = getRuntimeContext().getState(stateDesc);

        LOG.info("Module 11 Activity Response Correlator initialized (exercise-session-window, 3-phase HR)");
    }

    @Override
    public void processElement(CanonicalEvent event, Context ctx,
                               Collector<ActivityResponseRecord> out) throws Exception {
        ActivityCorrelationState state = correlationState.value();
        if (state == null) {
            state = new ActivityCorrelationState(event.getPatientId());
        }

        EventType eventType = event.getEventType();
        Map<String, Object> payload = event.getPayload();
        if (payload == null) {
            correlationState.update(state);
            return;
        }

        // Extract patient age if available (for HR_max calculation)
        if (state.getPatientAge() == null) {
            Object ageObj = payload.get("patient_age");
            if (ageObj instanceof Number) {
                state.setPatientAge(((Number) ageObj).intValue());
            }
        }

        // Route by event type
        if (isActivityEvent(eventType, payload)) {
            handleActivityEvent(event, state, ctx);
        } else if (isHRReading(eventType, payload)) {
            handleHRReading(event, state);
        } else if (isGlucoseReading(eventType, payload)) {
            handleGlucoseReading(event, state);
        } else if (eventType == EventType.VITAL_SIGN) {
            handleVitalSign(event, state);
        }

        state.setLastUpdated(ctx.timerService().currentProcessingTime());
        correlationState.update(state);
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx,
                        Collector<ActivityResponseRecord> out) throws Exception {
        ActivityCorrelationState state = correlationState.value();
        if (state == null) return;

        List<String> sessionIds = state.getSessionsForTimer(timestamp);
        for (String activityEventId : sessionIds) {
            ActivityCorrelationState.ActivitySession session = state.closeSession(activityEventId);
            if (session == null) continue;

            ActivityResponseRecord record = buildRecord(state, session);
            out.collect(record);

            ctx.output(FITNESS_PATTERN_FEED_TAG, record);

            LOG.debug("Activity session closed: patient={}, activity={}, type={}, peakHR={}, hrrClass={}",
                    state.getPatientId(), activityEventId, session.exerciseType,
                    record.getPeakHR(), record.getHrRecoveryClass());
        }

        correlationState.update(state);
    }

    // --- Event Classification ---

    private boolean isActivityEvent(EventType type, Map<String, Object> payload) {
        if (type != EventType.PATIENT_REPORTED) return false;
        Object reportType = payload.get("report_type");
        return "ACTIVITY_LOG".equalsIgnoreCase(reportType != null ? reportType.toString() : "");
    }

    private boolean isHRReading(EventType type, Map<String, Object> payload) {
        if (type != EventType.DEVICE_READING) return false;
        return payload.containsKey("heart_rate");
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

    private void handleActivityEvent(CanonicalEvent event, ActivityCorrelationState state, Context ctx) {
        String activityEventId = event.getId() != null ? event.getId() : UUID.randomUUID().toString();
        long activityStart = event.getEventTime();
        Map<String, Object> payload = event.getPayload();

        // Parse duration from payload
        long durationMs = 30L * 60_000L; // default 30 min
        Object durationObj = payload.get("duration_minutes");
        if (durationObj instanceof Number) {
            durationMs = ((Number) durationObj).longValue() * 60_000L;
        }

        long timerFireTime = state.openSession(activityEventId, activityStart, durationMs, payload);
        ctx.timerService().registerProcessingTimeTimer(timerFireTime);

        LOG.debug("Activity session opened: patient={}, activity={}, type={}, duration={}min, timerAt={}",
                state.getPatientId(), activityEventId,
                ExerciseType.fromString(payload.get("exercise_type") != null ? payload.get("exercise_type").toString() : null),
                durationMs / 60_000L, timerFireTime);
    }

    private void handleHRReading(CanonicalEvent event, ActivityCorrelationState state) {
        Map<String, Object> payload = event.getPayload();
        Object hrObj = payload.get("heart_rate");
        if (!(hrObj instanceof Number)) return;
        double heartRate = ((Number) hrObj).doubleValue();
        String source = payload.containsKey("source") ? payload.get("source").toString() : "WEARABLE";

        // Check if this is a resting HR reading
        Object activityFlag = payload.get("activity_state");
        if ("RESTING".equalsIgnoreCase(activityFlag != null ? activityFlag.toString() : "")) {
            state.updateRestingHR(heartRate, event.getEventTime());
        }

        state.addHRReading(event.getEventTime(), heartRate, source);
    }

    private void handleGlucoseReading(CanonicalEvent event, ActivityCorrelationState state) {
        Map<String, Object> payload = event.getPayload();
        double glucoseValue;
        String source;

        if (event.getEventType() == EventType.DEVICE_READING) {
            Object val = payload.get("glucose_value");
            if (!(val instanceof Number)) return;
            glucoseValue = ((Number) val).doubleValue();
            source = "CGM";
        } else {
            Object val = payload.get("value");
            if (!(val instanceof Number)) return;
            glucoseValue = ((Number) val).doubleValue();
            source = "SMBG";
        }

        state.addGlucoseReading(event.getEventTime(), glucoseValue, source);
    }

    private void handleVitalSign(CanonicalEvent event, ActivityCorrelationState state) {
        Map<String, Object> payload = event.getPayload();

        // BP reading
        Object sbpObj = payload.get("systolic_bp");
        if (sbpObj instanceof Number) {
            double sbp = ((Number) sbpObj).doubleValue();
            double dbp = (payload.get("diastolic_bp") instanceof Number)
                    ? ((Number) payload.get("diastolic_bp")).doubleValue() : 0.0;
            state.addBPReading(event.getEventTime(), sbp, dbp);
        }

        // Resting HR from vital sign
        Object restingHR = payload.get("resting_heart_rate");
        if (restingHR instanceof Number) {
            state.updateRestingHR(((Number) restingHR).doubleValue(), event.getEventTime());
        }
    }

    // --- Record Building ---

    private ActivityResponseRecord buildRecord(ActivityCorrelationState state,
                                               ActivityCorrelationState.ActivitySession session) {
        // Analyze HR
        Module11HRAnalyzer.Result hrResult = Module11HRAnalyzer.analyze(session.hrWindow);

        // Analyze glucose
        Module11GlucoseExerciseAnalyzer.Result glucoseResult =
                Module11GlucoseExerciseAnalyzer.analyze(
                        session.glucoseWindow,
                        session.activityStartTime,
                        session.activityEndTime != null ? session.activityEndTime : session.activityStartTime + 30 * 60_000L,
                        session.exerciseType);

        // Analyze BP
        Double preSBP = session.bpWindow.getPreMealSBP(); // reusing BPWindow (pre-meal = pre-exercise)
        Double peakSBP = session.peakExerciseSBP;
        Double postSBP = session.bpWindow.getPostMealSBP();
        Module11ExerciseBPAnalyzer.Result bpResult =
                Module11ExerciseBPAnalyzer.analyze(preSBP, peakSBP, postSBP);

        // Compute RPP
        Double peakRPP = null;
        if (hrResult != null && peakSBP != null) {
            peakRPP = Module11HRAnalyzer.computeRPP(hrResult.peakHR, peakSBP);
        }

        // MET-minutes
        double durationMin = session.reportedDurationMs / 60_000.0;
        double metMinutes = session.reportedMETs * durationMin;

        // Quality score
        double qualityScore = computeQualityScore(hrResult, glucoseResult, bpResult);

        // Window duration: use deterministic value from session boundaries
        long windowDurationMs = session.timerFireTime - session.activityStartTime;

        ActivityResponseRecord.Builder builder = ActivityResponseRecord.builder()
                .recordId("m11-" + UUID.randomUUID())
                .patientId(state.getPatientId())
                .activityEventId(session.activityEventId)
                .activityStartTime(session.activityStartTime)
                .activityDurationMin(durationMin)
                .exerciseType(session.exerciseType)
                .reportedMETs(session.reportedMETs)
                .metMinutes(metMinutes)
                .concurrent(session.concurrent)
                .windowDurationMs(windowDurationMs)
                .qualityScore(qualityScore);

        // HR features
        if (hrResult != null) {
            builder.peakHR(hrResult.peakHR)
                    .meanActiveHR(hrResult.meanActiveHR)
                    .hrr1(hrResult.hrr1)
                    .hrr2(hrResult.hrr2)
                    .hrRecoveryClass(hrResult.hrRecoveryClass)
                    .dominantZone(hrResult.dominantZone)
                    .hrReadingCount(hrResult.readingCount);
        }
        builder.restingHR(state.getLastRestingHR());

        // Glucose features
        if (glucoseResult != null) {
            builder.preExerciseGlucose(glucoseResult.preExerciseGlucose)
                    .exerciseGlucoseDelta(glucoseResult.exerciseGlucoseDelta)
                    .glucoseNadir(glucoseResult.glucoseNadir)
                    .hypoglycemiaFlag(glucoseResult.hypoglycemiaFlag)
                    .reboundHyperglycemiaFlag(glucoseResult.reboundHyperglycemiaFlag);
        }

        // BP features
        builder.preExerciseSBP(bpResult.preExerciseSBP)
                .peakExerciseSBP(bpResult.peakExerciseSBP)
                .postExerciseSBP(bpResult.postExerciseSBP)
                .exerciseBPResponse(bpResult.bpResponse)
                .peakRPP(peakRPP);

        return builder.build();
    }

    private double computeQualityScore(Module11HRAnalyzer.Result hr,
                                       Module11GlucoseExerciseAnalyzer.Result glucose,
                                       Module11ExerciseBPAnalyzer.Result bp) {
        double score = 0.0;
        if (hr != null) {
            score += 0.4 * hr.qualityScore;
            if (hr.hrr1 != null) score += 0.15; // HRR available adds quality
        }
        if (glucose != null && glucose.readingCount > 0) {
            score += 0.25;
        }
        if (bp != null && bp.bpResponse != ExerciseBPResponse.INCOMPLETE) {
            score += 0.2;
        }
        return Math.min(1.0, score);
    }
}
