package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Per-patient state for Module 10 Meal Response Correlator.
 *
 * Key design decisions:
 * - activeSessions: Map<mealEventId, MealSession> — supports overlapping meals
 * - lastBPReading: retroactive buffer for pre-meal BP (most recent within 60 min)
 * - dataTier: per-session, computed at close time from glucose sources in that session
 * - overlappingMealIds: tracks meal IDs flagged for overlap (second meal within 90 min)
 *
 * State TTL: 7 days (OnReadAndWrite + NeverReturnExpired).
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class MealCorrelationState implements Serializable {
    private static final long serialVersionUID = 1L;

    public static final long GLUCOSE_WINDOW_MS = 3L * 3600_000L;       // 3 hours
    public static final long GLUCOSE_GRACE_MS = 5L * 60_000L;          // 5 min grace
    public static final long BP_WINDOW_MS = 4L * 3600_000L;            // 4 hours
    public static final long PRE_MEAL_BP_LOOKBACK_MS = 60L * 60_000L;  // 60 min
    public static final long OVERLAP_THRESHOLD_MS = 90L * 60_000L;     // 90 min

    @JsonProperty("patientId")
    private String patientId;

    /**
     * Patient-level data tier — legacy field retained for serialization compatibility.
     * Since the per-session tier fix, the authoritative tier for each meal record is
     * computed by {@code MealSession.computeSessionTier()}, not this field.
     * Module 10b weekly summaries use the worst (highest ordinal) tier across the
     * week's records. This field is still initialized to TIER_3_SMBG as a safe default.
     */
    @JsonProperty("dataTier")
    private DataTier dataTier;

    @JsonProperty("activeSessions")
    private Map<String, MealSession> activeSessions;

    @JsonProperty("lastBPSystolic")
    private Double lastBPSystolic;

    @JsonProperty("lastBPDiastolic")
    private Double lastBPDiastolic;

    @JsonProperty("lastBPTimestamp")
    private Long lastBPTimestamp;

    @JsonProperty("totalMealsProcessed")
    private long totalMealsProcessed;

    @JsonProperty("lastUpdated")
    private long lastUpdated;

    public MealCorrelationState() {
        this.activeSessions = new HashMap<>();
        this.dataTier = DataTier.TIER_3_SMBG;
    }

    public MealCorrelationState(String patientId) {
        this();
        this.patientId = patientId;
    }

    /**
     * Open a new meal session. Returns the timer fire time (meal + 3h05m).
     */
    public long openSession(String mealEventId, long mealTimestamp,
                            Map<String, Object> mealPayload) {
        MealSession session = new MealSession();
        session.mealEventId = mealEventId;
        session.mealTimestamp = mealTimestamp;
        session.mealPayload = mealPayload != null ? new HashMap<>(mealPayload) : new HashMap<>();
        session.glucoseWindow = new GlucoseWindow();
        session.glucoseWindow.setWindowOpenTime(mealTimestamp);
        session.bpWindow = new BPWindow();

        // Retroactive pre-meal BP: attach if within 60 min
        if (lastBPTimestamp != null
                && (mealTimestamp - lastBPTimestamp) <= PRE_MEAL_BP_LOOKBACK_MS
                && lastBPSystolic != null) {
            session.bpWindow.setPreMealSBP(lastBPSystolic);
            session.bpWindow.setPreMealDBP(lastBPDiastolic);
            session.bpWindow.setPreMealTimestamp(lastBPTimestamp);
        }

        // Check overlapping meals
        for (MealSession existing : activeSessions.values()) {
            if ((mealTimestamp - existing.mealTimestamp) < OVERLAP_THRESHOLD_MS) {
                session.overlapping = true;
                existing.overlapping = true;
            }
        }

        activeSessions.put(mealEventId, session);
        totalMealsProcessed++;

        long timerFireTime = mealTimestamp + GLUCOSE_WINDOW_MS + GLUCOSE_GRACE_MS;
        session.timerFireTime = timerFireTime;
        return timerFireTime;
    }

    /**
     * Add a glucose reading to ALL active sessions whose window hasn't closed.
     */
    public void addGlucoseReading(long timestamp, double value, String source) {
        for (MealSession session : activeSessions.values()) {
            long windowEnd = session.mealTimestamp + GLUCOSE_WINDOW_MS;
            if (timestamp >= session.mealTimestamp && timestamp <= windowEnd) {
                session.glucoseWindow.addReading(timestamp, value, source);
                // Track glucose source per session for per-session tier computation
                if ("CGM".equalsIgnoreCase(source)) {
                    session.hasCGM = true;
                } else if ("SMBG".equalsIgnoreCase(source)) {
                    session.hasSMBG = true;
                }
                // Baseline is NOT set here — deferred to Module10GlucoseAnalyzer.analyze()
                // which uses the chronologically earliest reading (after sortByTime).
                // Setting it here would be incorrect if Kafka delivers out-of-order.
            }
        }
    }

    /**
     * Add a BP reading: buffer as lastBP AND feed to active sessions as post-meal.
     */
    public void addBPReading(long timestamp, double sbp, double dbp) {
        this.lastBPSystolic = sbp;
        this.lastBPDiastolic = dbp;
        this.lastBPTimestamp = timestamp;

        for (MealSession session : activeSessions.values()) {
            if (!session.bpWindow.hasPostMeal()
                    && timestamp > session.mealTimestamp
                    && timestamp <= session.mealTimestamp + BP_WINDOW_MS) {
                session.bpWindow.setPostMealSBP(sbp);
                session.bpWindow.setPostMealDBP(dbp);
                session.bpWindow.setPostMealTimestamp(timestamp);
            }
        }
    }

    /**
     * Close a session and remove from active map. Returns the session (or null).
     */
    public MealSession closeSession(String mealEventId) {
        return activeSessions.remove(mealEventId);
    }

    /**
     * Find session whose timer should fire at given timestamp.
     * NOTE: Multiple sessions can share the same timerFireTime if two meals arrive with
     * identical timestamps (same mealTimestamp → same timerFireTime). This is intentional:
     * onTimer will close and emit all of them in the same callback.
     */
    public List<String> getSessionsForTimer(long timerTimestamp) {
        List<String> ids = new ArrayList<>();
        for (Map.Entry<String, MealSession> entry : activeSessions.entrySet()) {
            if (entry.getValue().timerFireTime == timerTimestamp) {
                ids.add(entry.getKey());
            }
        }
        return ids;
    }

    // --- Getters/Setters ---
    public String getPatientId() { return patientId; }
    public void setPatientId(String id) { this.patientId = id; }
    public DataTier getDataTier() { return dataTier; }
    public void setDataTier(DataTier tier) { this.dataTier = tier; }
    public Map<String, MealSession> getActiveSessions() { return activeSessions; }
    public long getTotalMealsProcessed() { return totalMealsProcessed; }
    public void setTotalMealsProcessed(long c) { this.totalMealsProcessed = c; }
    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long ts) { this.lastUpdated = ts; }
    public Double getLastBPSystolic() { return lastBPSystolic; }
    public Double getLastBPDiastolic() { return lastBPDiastolic; }
    public Long getLastBPTimestamp() { return lastBPTimestamp; }

    /**
     * Per-meal session: tracks glucose window, BP window, and meal metadata.
     */
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class MealSession implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("mealEventId")
        public String mealEventId;

        @JsonProperty("mealTimestamp")
        public long mealTimestamp;

        @JsonProperty("mealPayload")
        public Map<String, Object> mealPayload;

        @JsonProperty("glucoseWindow")
        public GlucoseWindow glucoseWindow;

        @JsonProperty("bpWindow")
        public BPWindow bpWindow;

        @JsonProperty("timerFireTime")
        public long timerFireTime;

        @JsonProperty("overlapping")
        public boolean overlapping;

        @JsonProperty("hasCGM")
        public boolean hasCGM;

        @JsonProperty("hasSMBG")
        public boolean hasSMBG;

        public MealSession() {
            this.mealPayload = new HashMap<>();
        }

        /**
         * Compute data tier from glucose sources seen in THIS session.
         * CGM-only → Tier 1, both CGM+SMBG → Tier 2, SMBG-only or none → Tier 3.
         */
        public DataTier computeSessionTier() {
            if (hasCGM && hasSMBG) return DataTier.TIER_2_HYBRID;
            if (hasCGM) return DataTier.TIER_1_CGM;
            return DataTier.TIER_3_SMBG;
        }
    }
}
