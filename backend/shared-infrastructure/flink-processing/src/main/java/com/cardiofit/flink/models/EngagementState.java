package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Per-patient engagement state for Module 9.
 * Stores 8 boolean[14] signal bitmaps representing 14-day rolling engagement.
 * Index 0 = today, index 13 = 14 days ago.
 *
 * State TTL: 14 days (OnReadAndWrite + NeverReturnExpired).
 *
 * Review fixes incorporated:
 * R2: Channel-aware thresholds via channel + dataTier fields
 * R3: 3-day persistence via consecutiveDaysAtCurrentLevel
 * R4: Zombie prevention via lastUpdated staleness check
 * R7: History length tracking via validHistoryDays counter
 */
public class EngagementState implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final int WINDOW_DAYS = 14;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("channel")
    private String channel;

    @JsonProperty("dataTier")
    private String dataTier;

    @JsonProperty("signalBitmaps")
    private Map<SignalType, boolean[]> signalBitmaps;

    @JsonProperty("todaySignals")
    private Map<SignalType, Boolean> todaySignals;

    @JsonProperty("previousScore")
    private Double previousScore;

    @JsonProperty("previousLevel")
    private EngagementLevel previousLevel;

    @JsonProperty("consecutiveDaysAtCurrentLevel")
    private int consecutiveDaysAtCurrentLevel;

    @JsonProperty("consecutiveLowDays")
    private int consecutiveLowDays;

    @JsonProperty("alertSuppressionMap")
    private Map<String, Long> alertSuppressionMap;

    @JsonProperty("dailyTimerRegistered")
    private boolean dailyTimerRegistered;

    @JsonProperty("lastDailyTickTimestamp")
    private long lastDailyTickTimestamp;

    @JsonProperty("compositeHistory14d")
    private double[] compositeHistory14d;

    @JsonProperty("validHistoryDays")
    private int validHistoryDays;

    // Phase 2: 5 trajectory buffers for relapse prediction (7-day OLS windows)
    @JsonProperty("stepsBuffer7d")
    private double[] stepsBuffer7d;

    @JsonProperty("mealQualityBuffer7d")
    private double[] mealQualityBuffer7d;

    @JsonProperty("responseLatencyBuffer7d")
    private double[] responseLatencyBuffer7d;

    @JsonProperty("checkinCompletenessBuffer7d")
    private double[] checkinCompletenessBuffer7d;

    @JsonProperty("proteinAdherenceBuffer7d")
    private double[] proteinAdherenceBuffer7d;

    // Phase 2: Today's trajectory feature accumulators (reset daily)
    @JsonProperty("todaySteps")
    private double todaySteps;

    @JsonProperty("todayMealQuality")
    private double todayMealQuality;

    @JsonProperty("todayResponseLatency")
    private double todayResponseLatency;

    @JsonProperty("todayCheckinCompleteness")
    private double todayCheckinCompleteness;

    @JsonProperty("todayProteinAdherence")
    private double todayProteinAdherence;

    @JsonProperty("todayStepsSet")
    private boolean todayStepsSet;

    @JsonProperty("todayMealQualitySet")
    private boolean todayMealQualitySet;

    @JsonProperty("todayResponseLatencySet")
    private boolean todayResponseLatencySet;

    @JsonProperty("todayCheckinCompletenessSet")
    private boolean todayCheckinCompletenessSet;

    @JsonProperty("todayProteinAdherenceSet")
    private boolean todayProteinAdherenceSet;

    @JsonProperty("totalEventsProcessed")
    private long totalEventsProcessed;

    @JsonProperty("lastUpdated")
    private long lastUpdated;

    public EngagementState() {
        this.signalBitmaps = new LinkedHashMap<>();
        this.todaySignals = new LinkedHashMap<>();
        this.alertSuppressionMap = new java.util.HashMap<>();
        this.compositeHistory14d = new double[14];
        this.stepsBuffer7d = new double[7];
        this.mealQualityBuffer7d = new double[7];
        this.responseLatencyBuffer7d = new double[7];
        this.checkinCompletenessBuffer7d = new double[7];
        this.proteinAdherenceBuffer7d = new double[7];
        // Initialize trajectory buffers to -1.0 (sentinel: no data)
        java.util.Arrays.fill(stepsBuffer7d, -1.0);
        java.util.Arrays.fill(mealQualityBuffer7d, -1.0);
        java.util.Arrays.fill(responseLatencyBuffer7d, -1.0);
        java.util.Arrays.fill(checkinCompletenessBuffer7d, -1.0);
        java.util.Arrays.fill(proteinAdherenceBuffer7d, -1.0);
        this.channel = "CORPORATE";
        for (SignalType s : SignalType.values()) {
            signalBitmaps.put(s, new boolean[WINDOW_DAYS]);
            todaySignals.put(s, false);
        }
    }

    // --- Signal Operations ---

    public void markSignalToday(SignalType signal) {
        todaySignals.put(signal, true);
    }

    public boolean isSignalMarkedToday(SignalType signal) {
        Boolean marked = todaySignals.get(signal);
        return marked != null && marked;
    }

    /**
     * Advance the rolling window by one day.
     * Called by onTimer() at 23:59 UTC.
     *
     * 1. For each signal, shift bitmap RIGHT (drop index 13/oldest, insert today at index 0)
     * 2. Set index 0 to today's signal value
     * 3. Reset todaySignals for next day
     * 4. Shift compositeHistory14d right for trajectory
     * 5. Increment validHistoryDays (caps at 14)
     *
     * NOTE: "shift right" means elements move from lower to higher indices.
     * System.arraycopy(src, 0, dest, 1, 13) copies positions 0-12 to positions 1-13.
     * Position 13 (oldest) is overwritten. Position 0 is then set to today's value.
     */
    public void advanceDay(double todayCompositeScore) {
        for (SignalType signal : SignalType.values()) {
            boolean[] bitmap = signalBitmaps.get(signal);
            System.arraycopy(bitmap, 0, bitmap, 1, WINDOW_DAYS - 1);
            bitmap[0] = Boolean.TRUE.equals(todaySignals.get(signal));
            todaySignals.put(signal, false);
        }

        System.arraycopy(compositeHistory14d, 0, compositeHistory14d, 1,
                         compositeHistory14d.length - 1);
        compositeHistory14d[0] = todayCompositeScore;

        if (validHistoryDays < WINDOW_DAYS) {
            validHistoryDays++;
        }
    }

    public double getSignalDensity(SignalType signal) {
        boolean[] bitmap = signalBitmaps.get(signal);
        if (bitmap == null) return 0.0;
        int count = 0;
        for (boolean b : bitmap) {
            if (b) count++;
        }
        return (double) count / WINDOW_DAYS;
    }

    public Map<SignalType, Double> getAllDensities() {
        Map<SignalType, Double> densities = new LinkedHashMap<>();
        for (SignalType s : SignalType.values()) {
            densities.put(s, getSignalDensity(s));
        }
        return densities;
    }

    public EngagementChannel getEngagementChannel() {
        return EngagementChannel.fromString(channel);
    }

    // --- Alert Suppression ---

    public boolean isAlertSuppressed(String alertType, long currentTime) {
        Long lastEmission = alertSuppressionMap.get(alertType);
        if (lastEmission == null) return false;
        long suppressionWindowMs = 7L * 86_400_000L;
        return (currentTime - lastEmission) < suppressionWindowMs;
    }

    public void recordAlertEmission(String alertType, long currentTime) {
        alertSuppressionMap.put(alertType, currentTime);
    }

    // --- Bitmap Setter (for test builder) ---

    public void setSignalBitmap(SignalType signal, boolean[] bitmap) {
        if (bitmap.length != WINDOW_DAYS) {
            throw new IllegalArgumentException("Bitmap must be " + WINDOW_DAYS + " elements");
        }
        signalBitmaps.put(signal, bitmap);
    }

    public boolean[] getSignalBitmap(SignalType signal) {
        return signalBitmaps.get(signal);
    }

    // --- Standard Getters/Setters ---

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public String getChannel() { return channel; }
    public void setChannel(String channel) { this.channel = channel; }
    public String getDataTier() { return dataTier; }
    public void setDataTier(String dataTier) { this.dataTier = dataTier; }
    public Double getPreviousScore() { return previousScore; }
    public void setPreviousScore(Double previousScore) { this.previousScore = previousScore; }
    public EngagementLevel getPreviousLevel() { return previousLevel; }
    public void setPreviousLevel(EngagementLevel level) { this.previousLevel = level; }
    public int getConsecutiveDaysAtCurrentLevel() { return consecutiveDaysAtCurrentLevel; }
    public void setConsecutiveDaysAtCurrentLevel(int days) { this.consecutiveDaysAtCurrentLevel = days; }
    public int getConsecutiveLowDays() { return consecutiveLowDays; }
    public void setConsecutiveLowDays(int days) { this.consecutiveLowDays = days; }
    public boolean isDailyTimerRegistered() { return dailyTimerRegistered; }
    public void setDailyTimerRegistered(boolean registered) { this.dailyTimerRegistered = registered; }
    public long getLastDailyTickTimestamp() { return lastDailyTickTimestamp; }
    public void setLastDailyTickTimestamp(long ts) { this.lastDailyTickTimestamp = ts; }
    public double[] getCompositeHistory14d() { return compositeHistory14d; }
    public int getValidHistoryDays() { return validHistoryDays; }
    public void setValidHistoryDays(int days) { this.validHistoryDays = days; }
    public long getTotalEventsProcessed() { return totalEventsProcessed; }
    public void setTotalEventsProcessed(long count) { this.totalEventsProcessed = count; }
    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long ts) { this.lastUpdated = ts; }
    public Map<String, Long> getAlertSuppressionMap() { return alertSuppressionMap; }

    public static int getWindowDays() { return WINDOW_DAYS; }

    // --- Phase 2: Trajectory Buffer Operations ---

    /**
     * Advance all 5 trajectory buffers by one day (shift right, insert today's values at index 0).
     * Called by onTimer() after advanceDay().
     * Values of -1.0 = no data for that day (sentinel).
     */
    public void advanceTrajectoryBuffers(double steps, double mealQuality,
                                          double responseLatency,
                                          double checkinCompleteness,
                                          double proteinAdherence) {
        shiftAndInsert(stepsBuffer7d, steps);
        shiftAndInsert(mealQualityBuffer7d, mealQuality);
        shiftAndInsert(responseLatencyBuffer7d, responseLatency);
        shiftAndInsert(checkinCompletenessBuffer7d, checkinCompleteness);
        shiftAndInsert(proteinAdherenceBuffer7d, proteinAdherence);
    }

    private void shiftAndInsert(double[] buffer, double value) {
        System.arraycopy(buffer, 0, buffer, 1, buffer.length - 1);
        buffer[0] = value;
    }

    public double[] getStepsBuffer7d() { return stepsBuffer7d; }
    public double[] getMealQualityBuffer7d() { return mealQualityBuffer7d; }
    public double[] getResponseLatencyBuffer7d() { return responseLatencyBuffer7d; }
    public double[] getCheckinCompletenessBuffer7d() { return checkinCompletenessBuffer7d; }
    public double[] getProteinAdherenceBuffer7d() { return proteinAdherenceBuffer7d; }

    // --- Phase 2: Trajectory feature accumulators ---

    public void setTodaySteps(double steps) { this.todaySteps = steps; this.todayStepsSet = true; }
    public void setTodayMealQuality(double q) { this.todayMealQuality = q; this.todayMealQualitySet = true; }
    public void setTodayResponseLatency(double l) { this.todayResponseLatency = l; this.todayResponseLatencySet = true; }
    public void setTodayCheckinCompleteness(double c) { this.todayCheckinCompleteness = c; this.todayCheckinCompletenessSet = true; }
    public void setTodayProteinAdherence(double p) { this.todayProteinAdherence = p; this.todayProteinAdherenceSet = true; }

    public double getTodaySteps() { return todaySteps; }
    public double getTodayMealQuality() { return todayMealQuality; }
    public double getTodayResponseLatency() { return todayResponseLatency; }
    public double getTodayCheckinCompleteness() { return todayCheckinCompleteness; }
    public double getTodayProteinAdherence() { return todayProteinAdherence; }

    public boolean isTodayStepsSet() { return todayStepsSet; }
    public boolean isTodayMealQualitySet() { return todayMealQualitySet; }
    public boolean isTodayResponseLatencySet() { return todayResponseLatencySet; }
    public boolean isTodayCheckinCompletenessSet() { return todayCheckinCompletenessSet; }
    public boolean isTodayProteinAdherenceSet() { return todayProteinAdherenceSet; }

    /**
     * Flush today's trajectory features into buffers and reset accumulators.
     * Uses -1.0 sentinel for features not observed today.
     */
    public void flushTrajectoryAndReset() {
        advanceTrajectoryBuffers(
            todayStepsSet ? todaySteps : -1.0,
            todayMealQualitySet ? todayMealQuality : -1.0,
            todayResponseLatencySet ? todayResponseLatency : -1.0,
            todayCheckinCompletenessSet ? todayCheckinCompleteness : -1.0,
            todayProteinAdherenceSet ? todayProteinAdherence : -1.0
        );
        todaySteps = 0; todayStepsSet = false;
        todayMealQuality = 0; todayMealQualitySet = false;
        todayResponseLatency = 0; todayResponseLatencySet = false;
        todayCheckinCompleteness = 0; todayCheckinCompletenessSet = false;
        todayProteinAdherence = 0; todayProteinAdherenceSet = false;
    }
}
