package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@JsonIgnoreProperties(ignoreUnknown = true)
public class ActivityCorrelationState implements Serializable {

    private static final long serialVersionUID = 1L;

    public static final long RECOVERY_WINDOW_MS = 2L * 3600_000L;
    public static final long GRACE_PERIOD_MS = 5L * 60_000L;
    public static final long MAX_ACTIVITY_DURATION_MS = 4L * 3600_000L;
    public static final long MAX_WINDOW_MS = MAX_ACTIVITY_DURATION_MS + RECOVERY_WINDOW_MS + GRACE_PERIOD_MS;
    public static final long PRE_EXERCISE_BP_LOOKBACK_MS = 30L * 60_000L;
    public static final long PRE_EXERCISE_HR_LOOKBACK_MS = 10L * 60_000L;
    public static final long CONCURRENT_ACTIVITY_THRESHOLD_MS = 30L * 60_000L;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("patientAge")
    private Integer patientAge;

    @JsonProperty("hrMax")
    private double hrMax;

    @JsonProperty("activeSessions")
    private Map<String, ActivitySession> activeSessions;

    @JsonProperty("lastRestingHR")
    private Double lastRestingHR;

    @JsonProperty("lastRestingHRTimestamp")
    private Long lastRestingHRTimestamp;

    @JsonProperty("lastBPSystolic")
    private Double lastBPSystolic;

    @JsonProperty("lastBPDiastolic")
    private Double lastBPDiastolic;

    @JsonProperty("lastBPTimestamp")
    private Long lastBPTimestamp;

    @JsonProperty("totalActivitiesProcessed")
    private long totalActivitiesProcessed;

    @JsonProperty("lastUpdated")
    private long lastUpdated;

    public ActivityCorrelationState() {
        this.activeSessions = new HashMap<>();
        this.hrMax = ActivityIntensityZone.DEFAULT_HR_MAX;
    }

    public ActivityCorrelationState(String patientId) {
        this();
        this.patientId = patientId;
    }

    public void setPatientAge(Integer age) {
        this.patientAge = age;
        if (age != null && age > 0) {
            this.hrMax = ActivityIntensityZone.estimateHRMax(age);
        }
    }

    public long openSession(String activityEventId, long activityStart,
                            long durationMs, Map<String, Object> payload) {
        ActivitySession session = new ActivitySession();
        session.activityEventId = activityEventId;
        session.activityStartTime = activityStart;
        session.activityPayload = payload != null ? new HashMap<>(payload) : new HashMap<>();

        long cappedDuration = Math.min(Math.max(durationMs, 30L * 60_000L), MAX_ACTIVITY_DURATION_MS);
        session.reportedDurationMs = durationMs;
        session.activityEndTime = activityStart + cappedDuration;

        session.hrWindow = new HRWindow();
        session.hrWindow.setActivityStartTime(activityStart);
        session.hrWindow.setActivityEndTime(session.activityEndTime);
        session.hrWindow.setHrMax(hrMax);
        if (lastRestingHR != null) {
            session.hrWindow.setRestingHR(lastRestingHR);
        }

        session.glucoseWindow = new GlucoseWindow();
        session.glucoseWindow.setWindowOpenTime(activityStart);
        session.glucoseWindow.setWindowCloseTime(session.activityEndTime + RECOVERY_WINDOW_MS);

        session.bpWindow = new BPWindow();

        if (lastBPTimestamp != null
                && (activityStart - lastBPTimestamp) <= PRE_EXERCISE_BP_LOOKBACK_MS
                && lastBPSystolic != null) {
            session.bpWindow.setPreMealSBP(lastBPSystolic);
            session.bpWindow.setPreMealDBP(lastBPDiastolic);
            session.bpWindow.setPreMealTimestamp(lastBPTimestamp);
        }

        Object typeObj = session.activityPayload.get("exercise_type");
        session.exerciseType = ExerciseType.fromString(typeObj != null ? typeObj.toString() : null);

        Object metsObj = session.activityPayload.get("mets");
        if (metsObj instanceof Number) {
            session.reportedMETs = ((Number) metsObj).doubleValue();
        } else {
            session.reportedMETs = session.exerciseType.getDefaultMETs();
        }

        for (ActivitySession existing : activeSessions.values()) {
            if (Math.abs(activityStart - existing.activityStartTime) < CONCURRENT_ACTIVITY_THRESHOLD_MS) {
                session.concurrent = true;
                existing.concurrent = true;
            }
        }

        activeSessions.put(activityEventId, session);
        totalActivitiesProcessed++;

        long timerFireTime = session.activityEndTime + RECOVERY_WINDOW_MS + GRACE_PERIOD_MS;
        session.timerFireTime = timerFireTime;
        return timerFireTime;
    }

    public void addHRReading(long timestamp, double heartRate, String source) {
        for (ActivitySession session : activeSessions.values()) {
            long windowEnd = session.activityEndTime + RECOVERY_WINDOW_MS;
            if (timestamp >= (session.activityStartTime - PRE_EXERCISE_HR_LOOKBACK_MS)
                    && timestamp <= windowEnd) {
                session.hrWindow.addReading(timestamp, heartRate, source);
            }
        }
    }

    public void addGlucoseReading(long timestamp, double value, String source) {
        for (ActivitySession session : activeSessions.values()) {
            long windowEnd = session.activityEndTime + RECOVERY_WINDOW_MS;
            long lookback = 30L * 60_000L;
            if (timestamp >= (session.activityStartTime - lookback) && timestamp <= windowEnd) {
                session.glucoseWindow.addReading(timestamp, value, source);
                if (session.glucoseWindow.getBaseline() == null
                        && timestamp < session.activityStartTime) {
                    session.glucoseWindow.setBaseline(value);
                }
            }
        }
    }

    public void addBPReading(long timestamp, double sbp, double dbp) {
        this.lastBPSystolic = sbp;
        this.lastBPDiastolic = dbp;
        this.lastBPTimestamp = timestamp;

        for (ActivitySession session : activeSessions.values()) {
            if (timestamp >= session.activityStartTime && timestamp <= session.activityEndTime) {
                if (session.peakExerciseSBP == null || sbp > session.peakExerciseSBP) {
                    session.peakExerciseSBP = sbp;
                    session.peakExerciseDBP = dbp;
                    session.peakExerciseBPTimestamp = timestamp;
                }
            }
            if (session.activityEndTime != null
                    && !session.bpWindow.hasPostMeal()
                    && timestamp > session.activityEndTime
                    && timestamp <= session.activityEndTime + RECOVERY_WINDOW_MS) {
                session.bpWindow.setPostMealSBP(sbp);
                session.bpWindow.setPostMealDBP(dbp);
                session.bpWindow.setPostMealTimestamp(timestamp);
            }
        }
    }

    public void updateRestingHR(double restingHR, long timestamp) {
        this.lastRestingHR = restingHR;
        this.lastRestingHRTimestamp = timestamp;
    }

    public ActivitySession closeSession(String activityEventId) {
        return activeSessions.remove(activityEventId);
    }

    public List<String> getSessionsForTimer(long timerTimestamp) {
        List<String> ids = new ArrayList<>();
        for (Map.Entry<String, ActivitySession> entry : activeSessions.entrySet()) {
            if (entry.getValue().timerFireTime == timerTimestamp) {
                ids.add(entry.getKey());
            }
        }
        return ids;
    }

    // --- Getters/Setters ---
    public String getPatientId() { return patientId; }
    public void setPatientId(String id) { this.patientId = id; }
    public Integer getPatientAge() { return patientAge; }
    public double getHrMax() { return hrMax; }
    public void setHrMax(double v) { this.hrMax = v; }
    public Map<String, ActivitySession> getActiveSessions() { return activeSessions; }
    public long getTotalActivitiesProcessed() { return totalActivitiesProcessed; }
    public void setTotalActivitiesProcessed(long c) { this.totalActivitiesProcessed = c; }
    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long ts) { this.lastUpdated = ts; }
    public Double getLastRestingHR() { return lastRestingHR; }
    public Double getLastBPSystolic() { return lastBPSystolic; }
    public Double getLastBPDiastolic() { return lastBPDiastolic; }
    public Long getLastBPTimestamp() { return lastBPTimestamp; }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class ActivitySession implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("activityEventId")
        public String activityEventId;

        @JsonProperty("activityStartTime")
        public long activityStartTime;

        @JsonProperty("activityEndTime")
        public Long activityEndTime;

        @JsonProperty("reportedDurationMs")
        public long reportedDurationMs;

        @JsonProperty("activityPayload")
        public Map<String, Object> activityPayload;

        @JsonProperty("exerciseType")
        public ExerciseType exerciseType;

        @JsonProperty("reportedMETs")
        public double reportedMETs;

        @JsonProperty("hrWindow")
        public HRWindow hrWindow;

        @JsonProperty("glucoseWindow")
        public GlucoseWindow glucoseWindow;

        @JsonProperty("bpWindow")
        public BPWindow bpWindow;

        @JsonProperty("peakExerciseSBP")
        public Double peakExerciseSBP;

        @JsonProperty("peakExerciseDBP")
        public Double peakExerciseDBP;

        @JsonProperty("peakExerciseBPTimestamp")
        public Long peakExerciseBPTimestamp;

        @JsonProperty("timerFireTime")
        public long timerFireTime;

        @JsonProperty("concurrent")
        public boolean concurrent;

        public ActivitySession() {
            this.activityPayload = new HashMap<>();
            this.exerciseType = ExerciseType.MIXED;
        }
    }
}
