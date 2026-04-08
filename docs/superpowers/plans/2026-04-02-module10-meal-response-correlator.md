# Module 10: Meal Response Correlator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Module 10 (per-meal glucose/BP correlation with session windows) and Module 10b (weekly meal pattern aggregation with salt sensitivity OLS regression) as two separate Flink jobs.

**Architecture:** Session-window-driven KeyedProcessFunction where a meal event OPENS a window, glucose/BP readings FILL it, and a processing-time timer CLOSES it at 3h05m. Three-tier glucose processing (CGM 288/day, Hybrid, SMBG-only). Module 10b runs weekly aggregation with OLS salt sensitivity estimation. Both jobs keyed by patientId, consuming from `enriched-patient-events-v1`.

**Tech Stack:** Flink 2.1.0, Java 17, Jackson 2.17, Kafka (Confluent Cloud), JUnit 5

---

## File Structure

### Source Files (14 files)
- Create: `models/CurveShape.java` — enum: 5 glucose curve shapes
- Create: `models/SaltSensitivityClass.java` — enum: 4 salt sensitivity classifications
- Create: `models/MealTimeCategory.java` — enum: BREAKFAST/LUNCH/DINNER/SNACK
- Create: `models/DataTier.java` — enum: TIER_1_CGM/TIER_2_HYBRID/TIER_3_SMBG
- Create: `models/GlucoseWindow.java` — timestamped glucose readings buffer
- Create: `models/BPWindow.java` — pre/post BP reading pair
- Create: `models/MealCorrelationState.java` — per-patient session state for Module 10
- Create: `models/MealResponseRecord.java` — output model (28 fields)
- Create: `models/MealPatternSummary.java` — output model (21 fields)
- Create: `models/MealPatternState.java` — per-patient weekly aggregation state
- Create: `operators/Module10GlucoseAnalyzer.java` — 8 glucose features + iAUC
- Create: `operators/Module10CurveClassifier.java` — 5 curve shapes via first derivative
- Create: `operators/Module10BPCorrelator.java` — pre/post BP excursion
- Create: `operators/Module10_MealResponseCorrelator.java` — main KPF with session windows
- Create: `operators/Module10b_MealPatternAggregator.java` — weekly aggregation KPF
- Create: `operators/Module10bSaltSensitivityEstimator.java` — OLS regression
- Create: `operators/Module10bFoodRanker.java` — food impact ranking
- Modify: `FlinkJobOrchestrator.java` — add module10/module10b cases + launch methods

### Test Files (9 files)
- Create: `builders/Module10TestBuilder.java` — event + state factories
- Create: `operators/Module10GlucoseAnalyzerTest.java`
- Create: `operators/Module10CurveClassifierTest.java`
- Create: `operators/Module10BPCorrelatorTest.java`
- Create: `operators/Module10SessionWindowTest.java`
- Create: `operators/Module10OverlappingMealTest.java`
- Create: `operators/Module10TierDegradationTest.java`
- Create: `operators/Module10bSaltSensitivityTest.java`
- Create: `operators/Module10bFoodRankerTest.java`

---

### Task 1: Create Model Enums

**Files:**
- Create: `models/CurveShape.java`
- Create: `models/SaltSensitivityClass.java`
- Create: `models/MealTimeCategory.java`
- Create: `models/DataTier.java`

- [ ] **Step 1: Create CurveShape enum**

```java
package com.cardiofit.flink.models;

/**
 * Glucose response curve shape classification.
 * Determined by Module10CurveClassifier using first-derivative analysis
 * on 3-point moving-average smoothed CGM readings.
 */
public enum CurveShape {
    RAPID_SPIKE,    // Sharp rise, sharp fall — peak within first 30 min
    SLOW_RISE,      // Gradual ascent over 60-90 min
    PLATEAU,        // Rise then sustained elevation (fall rate < 0.5 mg/dL/min)
    DOUBLE_PEAK,    // Two distinct peaks separated by ≥20 min
    FLAT,           // Max excursion < 20 mg/dL above baseline
    UNKNOWN;        // Insufficient data points for classification

    /** Minimum CGM readings required for classification (Tier 1 only) */
    public static final int MIN_READINGS_FOR_CLASSIFICATION = 6;
}
```

- [ ] **Step 2: Create SaltSensitivityClass enum**

```java
package com.cardiofit.flink.models;

/**
 * Salt sensitivity classification from Module 10b OLS regression.
 * β = slope of linear regression on (sodium_mg, SBP_excursion) pairs.
 */
public enum SaltSensitivityClass {
    SALT_RESISTANT,  // β < 0.001 mmHg/mg
    MODERATE,        // 0.001 ≤ β < 0.005
    HIGH,            // β ≥ 0.005
    UNDETERMINED;    // < 30 pairs OR R² < 0.1

    public static SaltSensitivityClass fromBetaAndR2(double beta, double rSquared, int pairCount) {
        if (pairCount < 30 || rSquared < 0.1) return UNDETERMINED;
        if (beta < 0.001) return SALT_RESISTANT;
        if (beta < 0.005) return MODERATE;
        return HIGH;
    }
}
```

- [ ] **Step 3: Create MealTimeCategory enum**

```java
package com.cardiofit.flink.models;

import java.time.Instant;
import java.time.ZoneOffset;
import java.time.ZonedDateTime;

/**
 * Meal time classification based on hour of day (UTC).
 * Used for per-meal-time aggregation in Module 10b.
 */
public enum MealTimeCategory {
    BREAKFAST,  // 05:00–10:59
    LUNCH,      // 11:00–14:59
    DINNER,     // 15:00–21:59
    SNACK;      // 22:00–04:59

    public static MealTimeCategory fromTimestamp(long epochMs) {
        ZonedDateTime zdt = Instant.ofEpochMilli(epochMs).atZone(ZoneOffset.UTC);
        int hour = zdt.getHour();
        if (hour >= 5 && hour <= 10) return BREAKFAST;
        if (hour >= 11 && hour <= 14) return LUNCH;
        if (hour >= 15 && hour <= 21) return DINNER;
        return SNACK;
    }
}
```

- [ ] **Step 4: Create DataTier enum**

```java
package com.cardiofit.flink.models;

/**
 * Data tier classification for glucose processing depth.
 * Tier 1: CGM (288 readings/day, 8 features)
 * Tier 2: Hybrid CGM+SMBG (interpolated iAUC)
 * Tier 3: SMBG-only (excursion only, no curve shape)
 */
public enum DataTier {
    TIER_1_CGM,
    TIER_2_HYBRID,
    TIER_3_SMBG;

    public static DataTier fromString(String tier) {
        if (tier == null) return TIER_3_SMBG;
        switch (tier.toUpperCase().replace("-", "_")) {
            case "TIER_1_CGM":
            case "TIER_1":
            case "CGM":
                return TIER_1_CGM;
            case "TIER_2_HYBRID":
            case "TIER_2":
            case "HYBRID":
                return TIER_2_HYBRID;
            default:
                return TIER_3_SMBG;
        }
    }

    public boolean supportsCurveClassification() {
        return this == TIER_1_CGM;
    }

    public boolean supportsFullIAUC() {
        return this == TIER_1_CGM || this == TIER_2_HYBRID;
    }
}
```

- [ ] **Step 5: Compile enums**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 6: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/CurveShape.java backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/SaltSensitivityClass.java backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/MealTimeCategory.java backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/DataTier.java
git commit -m "feat(module10): add CurveShape, SaltSensitivityClass, MealTimeCategory, DataTier enums"
```

---

### Task 2: Create GlucoseWindow and BPWindow Models

**Files:**
- Create: `models/GlucoseWindow.java`
- Create: `models/BPWindow.java`

- [ ] **Step 1: Create GlucoseWindow**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Timestamped glucose readings buffer for a single meal session window.
 * Sorted by timestamp. Used by Module10GlucoseAnalyzer to compute iAUC,
 * peak, time-to-peak, recovery time, and curve shape.
 *
 * Window duration: 3h (CGM) or 2h (SMBG).
 * Grace period: +5 min (processing-time timer fires at 3h05m).
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class GlucoseWindow implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("readings")
    private List<Reading> readings;

    @JsonProperty("baseline")
    private Double baseline;

    @JsonProperty("windowOpenTime")
    private long windowOpenTime;

    @JsonProperty("windowCloseTime")
    private long windowCloseTime;

    @JsonProperty("dataTier")
    private DataTier dataTier;

    public GlucoseWindow() {
        this.readings = new ArrayList<>();
    }

    public void addReading(long timestamp, double glucoseValue, String source) {
        readings.add(new Reading(timestamp, glucoseValue, source));
    }

    public int size() {
        return readings.size();
    }

    public boolean isEmpty() {
        return readings.isEmpty();
    }

    /** Sort readings by timestamp (call before analysis). */
    public void sortByTime() {
        readings.sort((a, b) -> Long.compare(a.timestamp, b.timestamp));
    }

    // --- Getters/Setters ---
    public List<Reading> getReadings() { return readings; }
    public Double getBaseline() { return baseline; }
    public void setBaseline(Double baseline) { this.baseline = baseline; }
    public long getWindowOpenTime() { return windowOpenTime; }
    public void setWindowOpenTime(long t) { this.windowOpenTime = t; }
    public long getWindowCloseTime() { return windowCloseTime; }
    public void setWindowCloseTime(long t) { this.windowCloseTime = t; }
    public DataTier getDataTier() { return dataTier; }
    public void setDataTier(DataTier tier) { this.dataTier = tier; }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class Reading implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("timestamp")
        public long timestamp;

        @JsonProperty("value")
        public double value;

        @JsonProperty("source")
        public String source; // "CGM", "SMBG"

        public Reading() {}

        public Reading(long timestamp, double value, String source) {
            this.timestamp = timestamp;
            this.value = value;
            this.source = source;
        }
    }
}
```

- [ ] **Step 2: Create BPWindow**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Pre-meal and post-meal BP reading pair for a single meal session.
 *
 * Pre-meal BP: retroactive — most recent BP reading within 60 min BEFORE meal.
 * Post-meal BP: first BP reading within 4h AFTER meal.
 * BP excursion = post_sbp - pre_sbp (can be negative).
 *
 * Window duration: 4h (longer than glucose window).
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class BPWindow implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("preMealSBP")
    private Double preMealSBP;

    @JsonProperty("preMealDBP")
    private Double preMealDBP;

    @JsonProperty("preMealTimestamp")
    private Long preMealTimestamp;

    @JsonProperty("postMealSBP")
    private Double postMealSBP;

    @JsonProperty("postMealDBP")
    private Double postMealDBP;

    @JsonProperty("postMealTimestamp")
    private Long postMealTimestamp;

    public BPWindow() {}

    public boolean hasPreMeal() {
        return preMealSBP != null;
    }

    public boolean hasPostMeal() {
        return postMealSBP != null;
    }

    public boolean isComplete() {
        return hasPreMeal() && hasPostMeal();
    }

    /**
     * SBP excursion = post - pre. Null if either reading missing.
     */
    public Double getSBPExcursion() {
        if (!isComplete()) return null;
        return postMealSBP - preMealSBP;
    }

    // --- Getters/Setters ---
    public Double getPreMealSBP() { return preMealSBP; }
    public void setPreMealSBP(Double v) { this.preMealSBP = v; }
    public Double getPreMealDBP() { return preMealDBP; }
    public void setPreMealDBP(Double v) { this.preMealDBP = v; }
    public Long getPreMealTimestamp() { return preMealTimestamp; }
    public void setPreMealTimestamp(Long t) { this.preMealTimestamp = t; }
    public Double getPostMealSBP() { return postMealSBP; }
    public void setPostMealSBP(Double v) { this.postMealSBP = v; }
    public Double getPostMealDBP() { return postMealDBP; }
    public void setPostMealDBP(Double v) { this.postMealDBP = v; }
    public Long getPostMealTimestamp() { return postMealTimestamp; }
    public void setPostMealTimestamp(Long t) { this.postMealTimestamp = t; }
}
```

- [ ] **Step 3: Compile**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/GlucoseWindow.java backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/BPWindow.java
git commit -m "feat(module10): add GlucoseWindow and BPWindow session models"
```

---

### Task 3: Create MealCorrelationState

**Files:**
- Create: `models/MealCorrelationState.java`

- [ ] **Step 1: Create MealCorrelationState**

This is the per-patient keyed state for Module 10. It tracks open meal sessions and the retroactive BP buffer.

```java
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
 * - dataTier: inferred from first CGM/SMBG event (sticky per patient)
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
        session.glucoseWindow.setDataTier(dataTier);
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
                // Set baseline from first reading if not set
                if (session.glucoseWindow.getBaseline() == null) {
                    session.glucoseWindow.setBaseline(value);
                }
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

        public MealSession() {
            this.mealPayload = new HashMap<>();
        }
    }
}
```

- [ ] **Step 2: Compile**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/MealCorrelationState.java
git commit -m "feat(module10): add MealCorrelationState with session window management"
```

---

### Task 4: Create MealResponseRecord (28 fields)

**Files:**
- Create: `models/MealResponseRecord.java`

- [ ] **Step 1: Create MealResponseRecord**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.Map;

/**
 * Output record from Module 10: per-meal glucose and BP response.
 * 28 fields across all tiers. Tier 3 patients will have null CGM-specific fields.
 * Emitted to flink.meal-response topic.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class MealResponseRecord implements Serializable {
    private static final long serialVersionUID = 1L;

    // --- Identity (4 fields) ---
    @JsonProperty("recordId")
    private String recordId;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("mealEventId")
    private String mealEventId;

    @JsonProperty("correlationId")
    private String correlationId;

    // --- Meal metadata (5 fields) ---
    @JsonProperty("mealTimestamp")
    private long mealTimestamp;

    @JsonProperty("mealTimeCategory")
    private MealTimeCategory mealTimeCategory;

    @JsonProperty("carbGrams")
    private Double carbGrams;

    @JsonProperty("proteinGrams")
    private Double proteinGrams;

    @JsonProperty("sodiumMg")
    private Double sodiumMg;

    // --- Glucose features (8 fields) — Tier 1/2 only ---
    @JsonProperty("glucoseBaseline")
    private Double glucoseBaseline;

    @JsonProperty("glucosePeak")
    private Double glucosePeak;

    @JsonProperty("glucoseExcursion")
    private Double glucoseExcursion;

    @JsonProperty("timeTopeakMin")
    private Double timeToPeakMin;

    @JsonProperty("iAUC")
    private Double iAUC;

    @JsonProperty("recoveryTimeMin")
    private Double recoveryTimeMin;

    @JsonProperty("curveShape")
    private CurveShape curveShape;

    @JsonProperty("glucoseReadingCount")
    private int glucoseReadingCount;

    // --- BP features (4 fields) ---
    @JsonProperty("preMealSBP")
    private Double preMealSBP;

    @JsonProperty("postMealSBP")
    private Double postMealSBP;

    @JsonProperty("sbpExcursion")
    private Double sbpExcursion;

    @JsonProperty("bpComplete")
    private boolean bpComplete;

    // --- Processing metadata (7 fields) ---
    @JsonProperty("dataTier")
    private DataTier dataTier;

    @JsonProperty("windowDurationMs")
    private long windowDurationMs;

    @JsonProperty("overlapping")
    private boolean overlapping;

    @JsonProperty("processingTimestamp")
    private long processingTimestamp;

    @JsonProperty("mealPayload")
    private Map<String, Object> mealPayload;

    @JsonProperty("qualityScore")
    private double qualityScore;

    @JsonProperty("version")
    private String version;

    public MealResponseRecord() {
        this.version = "1.0";
        this.processingTimestamp = System.currentTimeMillis();
    }

    public static Builder builder() { return new Builder(); }

    public static class Builder {
        private final MealResponseRecord r = new MealResponseRecord();

        public Builder recordId(String v) { r.recordId = v; return this; }
        public Builder patientId(String v) { r.patientId = v; return this; }
        public Builder mealEventId(String v) { r.mealEventId = v; return this; }
        public Builder correlationId(String v) { r.correlationId = v; return this; }
        public Builder mealTimestamp(long v) { r.mealTimestamp = v; return this; }
        public Builder mealTimeCategory(MealTimeCategory v) { r.mealTimeCategory = v; return this; }
        public Builder carbGrams(Double v) { r.carbGrams = v; return this; }
        public Builder proteinGrams(Double v) { r.proteinGrams = v; return this; }
        public Builder sodiumMg(Double v) { r.sodiumMg = v; return this; }
        public Builder glucoseBaseline(Double v) { r.glucoseBaseline = v; return this; }
        public Builder glucosePeak(Double v) { r.glucosePeak = v; return this; }
        public Builder glucoseExcursion(Double v) { r.glucoseExcursion = v; return this; }
        public Builder timeToPeakMin(Double v) { r.timeToPeakMin = v; return this; }
        public Builder iAUC(Double v) { r.iAUC = v; return this; }
        public Builder recoveryTimeMin(Double v) { r.recoveryTimeMin = v; return this; }
        public Builder curveShape(CurveShape v) { r.curveShape = v; return this; }
        public Builder glucoseReadingCount(int v) { r.glucoseReadingCount = v; return this; }
        public Builder preMealSBP(Double v) { r.preMealSBP = v; return this; }
        public Builder postMealSBP(Double v) { r.postMealSBP = v; return this; }
        public Builder sbpExcursion(Double v) { r.sbpExcursion = v; return this; }
        public Builder bpComplete(boolean v) { r.bpComplete = v; return this; }
        public Builder dataTier(DataTier v) { r.dataTier = v; return this; }
        public Builder windowDurationMs(long v) { r.windowDurationMs = v; return this; }
        public Builder overlapping(boolean v) { r.overlapping = v; return this; }
        public Builder mealPayload(Map<String, Object> v) { r.mealPayload = v; return this; }
        public Builder qualityScore(double v) { r.qualityScore = v; return this; }
        public MealResponseRecord build() { return r; }
    }

    // --- Getters ---
    public String getRecordId() { return recordId; }
    public String getPatientId() { return patientId; }
    public String getMealEventId() { return mealEventId; }
    public String getCorrelationId() { return correlationId; }
    public long getMealTimestamp() { return mealTimestamp; }
    public MealTimeCategory getMealTimeCategory() { return mealTimeCategory; }
    public Double getCarbGrams() { return carbGrams; }
    public Double getProteinGrams() { return proteinGrams; }
    public Double getSodiumMg() { return sodiumMg; }
    public Double getGlucoseBaseline() { return glucoseBaseline; }
    public Double getGlucosePeak() { return glucosePeak; }
    public Double getGlucoseExcursion() { return glucoseExcursion; }
    public Double getTimeToPeakMin() { return timeToPeakMin; }
    public Double getIAUC() { return iAUC; }
    public Double getRecoveryTimeMin() { return recoveryTimeMin; }
    public CurveShape getCurveShape() { return curveShape; }
    public int getGlucoseReadingCount() { return glucoseReadingCount; }
    public Double getPreMealSBP() { return preMealSBP; }
    public Double getPostMealSBP() { return postMealSBP; }
    public Double getSbpExcursion() { return sbpExcursion; }
    public boolean isBpComplete() { return bpComplete; }
    public DataTier getDataTier() { return dataTier; }
    public long getWindowDurationMs() { return windowDurationMs; }
    public boolean isOverlapping() { return overlapping; }
    public long getProcessingTimestamp() { return processingTimestamp; }
    public Map<String, Object> getMealPayload() { return mealPayload; }
    public double getQualityScore() { return qualityScore; }
    public String getVersion() { return version; }

    // --- Setters for correlation ---
    public void setCorrelationId(String v) { this.correlationId = v; }
    public void setRecordId(String v) { this.recordId = v; }

    @Override
    public String toString() {
        return "MealResponseRecord{" +
            "patientId='" + patientId + '\'' +
            ", mealEventId='" + mealEventId + '\'' +
            ", tier=" + dataTier +
            ", glucose=" + glucoseExcursion +
            ", iAUC=" + iAUC +
            ", curve=" + curveShape +
            ", sbpExcursion=" + sbpExcursion +
            '}';
    }
}
```

- [ ] **Step 2: Compile**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/MealResponseRecord.java
git commit -m "feat(module10): add MealResponseRecord output model (28 fields)"
```

---

### Task 5: Create MealPatternSummary (21 fields)

**Files:**
- Create: `models/MealPatternSummary.java`

- [ ] **Step 1: Create MealPatternSummary**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.List;
import java.util.Map;

/**
 * Output record from Module 10b: weekly meal pattern aggregation.
 * 21 fields. Emitted to flink.meal-patterns topic.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class MealPatternSummary implements Serializable {
    private static final long serialVersionUID = 1L;

    // --- Identity (3 fields) ---
    @JsonProperty("summaryId")
    private String summaryId;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("correlationId")
    private String correlationId;

    // --- Time range (2 fields) ---
    @JsonProperty("periodStartMs")
    private long periodStartMs;

    @JsonProperty("periodEndMs")
    private long periodEndMs;

    // --- Aggregated glucose metrics (4 fields) ---
    @JsonProperty("meanIAUC")
    private Double meanIAUC;

    @JsonProperty("medianExcursion")
    private Double medianExcursion;

    @JsonProperty("meanTimeToPeakMin")
    private Double meanTimeToPeakMin;

    @JsonProperty("dominantCurveShape")
    private CurveShape dominantCurveShape;

    // --- Per-meal-time breakdown (1 field — map) ---
    @JsonProperty("mealTimeBreakdown")
    private Map<MealTimeCategory, MealTimeStats> mealTimeBreakdown;

    // --- Salt sensitivity (4 fields) ---
    @JsonProperty("saltSensitivityClass")
    private SaltSensitivityClass saltSensitivityClass;

    @JsonProperty("saltBeta")
    private Double saltBeta;

    @JsonProperty("saltRSquared")
    private Double saltRSquared;

    @JsonProperty("saltPairCount")
    private int saltPairCount;

    // --- Food impact ranking (1 field) ---
    @JsonProperty("topFoodsByExcursion")
    private List<FoodImpact> topFoodsByExcursion;

    // --- Processing metadata (6 fields) ---
    @JsonProperty("totalMealsInPeriod")
    private int totalMealsInPeriod;

    @JsonProperty("dataTier")
    private DataTier dataTier;

    @JsonProperty("processingTimestamp")
    private long processingTimestamp;

    @JsonProperty("qualityScore")
    private double qualityScore;

    @JsonProperty("mealsWithGlucose")
    private int mealsWithGlucose;

    @JsonProperty("version")
    private String version;

    public MealPatternSummary() {
        this.version = "1.0";
        this.processingTimestamp = System.currentTimeMillis();
    }

    // --- Getters/Setters ---
    public String getSummaryId() { return summaryId; }
    public void setSummaryId(String v) { this.summaryId = v; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String v) { this.patientId = v; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String v) { this.correlationId = v; }
    public long getPeriodStartMs() { return periodStartMs; }
    public void setPeriodStartMs(long v) { this.periodStartMs = v; }
    public long getPeriodEndMs() { return periodEndMs; }
    public void setPeriodEndMs(long v) { this.periodEndMs = v; }
    public Double getMeanIAUC() { return meanIAUC; }
    public void setMeanIAUC(Double v) { this.meanIAUC = v; }
    public Double getMedianExcursion() { return medianExcursion; }
    public void setMedianExcursion(Double v) { this.medianExcursion = v; }
    public Double getMeanTimeToPeakMin() { return meanTimeToPeakMin; }
    public void setMeanTimeToPeakMin(Double v) { this.meanTimeToPeakMin = v; }
    public CurveShape getDominantCurveShape() { return dominantCurveShape; }
    public void setDominantCurveShape(CurveShape v) { this.dominantCurveShape = v; }
    public Map<MealTimeCategory, MealTimeStats> getMealTimeBreakdown() { return mealTimeBreakdown; }
    public void setMealTimeBreakdown(Map<MealTimeCategory, MealTimeStats> v) { this.mealTimeBreakdown = v; }
    public SaltSensitivityClass getSaltSensitivityClass() { return saltSensitivityClass; }
    public void setSaltSensitivityClass(SaltSensitivityClass v) { this.saltSensitivityClass = v; }
    public Double getSaltBeta() { return saltBeta; }
    public void setSaltBeta(Double v) { this.saltBeta = v; }
    public Double getSaltRSquared() { return saltRSquared; }
    public void setSaltRSquared(Double v) { this.saltRSquared = v; }
    public int getSaltPairCount() { return saltPairCount; }
    public void setSaltPairCount(int v) { this.saltPairCount = v; }
    public List<FoodImpact> getTopFoodsByExcursion() { return topFoodsByExcursion; }
    public void setTopFoodsByExcursion(List<FoodImpact> v) { this.topFoodsByExcursion = v; }
    public int getTotalMealsInPeriod() { return totalMealsInPeriod; }
    public void setTotalMealsInPeriod(int v) { this.totalMealsInPeriod = v; }
    public DataTier getDataTier() { return dataTier; }
    public void setDataTier(DataTier v) { this.dataTier = v; }
    public long getProcessingTimestamp() { return processingTimestamp; }
    public int getMealsWithGlucose() { return mealsWithGlucose; }
    public void setMealsWithGlucose(int v) { this.mealsWithGlucose = v; }
    public double getQualityScore() { return qualityScore; }
    public void setQualityScore(double v) { this.qualityScore = v; }
    public String getVersion() { return version; }

    /**
     * Per-meal-time aggregated stats.
     */
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class MealTimeStats implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("mealCount")
        public int mealCount;

        @JsonProperty("meanExcursion")
        public double meanExcursion;

        @JsonProperty("meanIAUC")
        public double meanIAUC;

        @JsonProperty("dominantCurve")
        public CurveShape dominantCurve;

        public MealTimeStats() {}
    }

    /**
     * Food impact entry for ranking.
     */
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class FoodImpact implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("foodDescription")
        public String foodDescription;

        @JsonProperty("mealCount")
        public int mealCount;

        @JsonProperty("meanExcursion")
        public double meanExcursion;

        @JsonProperty("meanIAUC")
        public double meanIAUC;

        public FoodImpact() {}

        public FoodImpact(String desc, int count, double excursion, double iauc) {
            this.foodDescription = desc;
            this.mealCount = count;
            this.meanExcursion = excursion;
            this.meanIAUC = iauc;
        }
    }
}
```

- [ ] **Step 2: Compile**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/MealPatternSummary.java
git commit -m "feat(module10b): add MealPatternSummary output model (21 fields)"
```

---

### Task 6: Create MealPatternState

**Files:**
- Create: `models/MealPatternState.java`

- [ ] **Step 1: Create MealPatternState**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Per-patient state for Module 10b weekly aggregation.
 *
 * Stores:
 * - 60-day rolling buffer of (sodium_mg, SBP_excursion) pairs for OLS regression
 * - Per-meal-time accumulator maps for weekly summary
 * - Food description → excursion accumulator for food impact ranking
 *
 * State TTL: 60 days (OnReadAndWrite + NeverReturnExpired).
 * Weekly timer fires every Monday 00:00 UTC.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class MealPatternState implements Serializable {
    private static final long serialVersionUID = 1L;
    public static final int SALT_BUFFER_MAX_DAYS = 60;
    public static final long WEEK_MS = 7L * 86_400_000L;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("dataTier")
    private DataTier dataTier;

    // --- Salt sensitivity OLS buffer (60-day rolling) ---
    @JsonProperty("sodiumSBPPairs")
    private List<SodiumSBPPair> sodiumSBPPairs;

    // --- Weekly accumulators (reset after each weekly emit) ---
    @JsonProperty("weeklyMealRecords")
    private List<MealResponseRecord> weeklyMealRecords;

    // --- Timer state ---
    @JsonProperty("weeklyTimerRegistered")
    private boolean weeklyTimerRegistered;

    @JsonProperty("lastWeeklyEmitTimestamp")
    private long lastWeeklyEmitTimestamp;

    @JsonProperty("lastUpdated")
    private long lastUpdated;

    @JsonProperty("totalRecordsProcessed")
    private long totalRecordsProcessed;

    public MealPatternState() {
        this.sodiumSBPPairs = new ArrayList<>();
        this.weeklyMealRecords = new ArrayList<>();
    }

    public MealPatternState(String patientId) {
        this();
        this.patientId = patientId;
    }

    /**
     * Add a MealResponseRecord to the weekly buffer.
     * Also add to salt sensitivity buffer if sodium + SBP excursion both present.
     */
    public void addMealRecord(MealResponseRecord record) {
        weeklyMealRecords.add(record);
        totalRecordsProcessed++;

        // Add to salt sensitivity buffer if both values present
        if (record.getSodiumMg() != null && record.getSbpExcursion() != null) {
            sodiumSBPPairs.add(new SodiumSBPPair(
                record.getSodiumMg(),
                record.getSbpExcursion(),
                record.getMealTimestamp()
            ));
        }

        // Evict pairs older than 60 days
        long cutoff = record.getMealTimestamp() - (SALT_BUFFER_MAX_DAYS * 86_400_000L);
        sodiumSBPPairs.removeIf(p -> p.timestamp < cutoff);
    }

    /**
     * Drain weekly records (returns list and clears buffer).
     */
    public List<MealResponseRecord> drainWeeklyRecords() {
        List<MealResponseRecord> drained = new ArrayList<>(weeklyMealRecords);
        weeklyMealRecords.clear();
        return drained;
    }

    // --- Getters/Setters ---
    public String getPatientId() { return patientId; }
    public void setPatientId(String v) { this.patientId = v; }
    public DataTier getDataTier() { return dataTier; }
    public void setDataTier(DataTier v) { this.dataTier = v; }
    public List<SodiumSBPPair> getSodiumSBPPairs() { return sodiumSBPPairs; }
    public List<MealResponseRecord> getWeeklyMealRecords() { return weeklyMealRecords; }
    public boolean isWeeklyTimerRegistered() { return weeklyTimerRegistered; }
    public void setWeeklyTimerRegistered(boolean v) { this.weeklyTimerRegistered = v; }
    public long getLastWeeklyEmitTimestamp() { return lastWeeklyEmitTimestamp; }
    public void setLastWeeklyEmitTimestamp(long v) { this.lastWeeklyEmitTimestamp = v; }
    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long v) { this.lastUpdated = v; }
    public long getTotalRecordsProcessed() { return totalRecordsProcessed; }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class SodiumSBPPair implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("sodiumMg")
        public double sodiumMg;

        @JsonProperty("sbpExcursion")
        public double sbpExcursion;

        @JsonProperty("timestamp")
        public long timestamp;

        public SodiumSBPPair() {}

        public SodiumSBPPair(double sodiumMg, double sbpExcursion, long timestamp) {
            this.sodiumMg = sodiumMg;
            this.sbpExcursion = sbpExcursion;
            this.timestamp = timestamp;
        }
    }
}
```

- [ ] **Step 2: Compile**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/MealPatternState.java
git commit -m "feat(module10b): add MealPatternState with 60-day salt sensitivity buffer"
```

---

### Task 7: Create Module10GlucoseAnalyzer

**Files:**
- Create: `operators/Module10GlucoseAnalyzer.java`
- Test: `operators/Module10GlucoseAnalyzerTest.java`

- [ ] **Step 1: Write the failing test**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module10GlucoseAnalyzerTest {

    private static final long BASE = 1743552000000L; // 2025-04-02 00:00 UTC
    private static final long MIN_5 = 5 * 60_000L;

    @Test
    void iAUC_trapezoidalRule_positiveOnlyAboveBaseline() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(100.0);
        // 3 readings: t=0 baseline, t=5min 140, t=10min 100 (back to baseline)
        window.addReading(BASE, 100.0, "CGM");
        window.addReading(BASE + MIN_5, 140.0, "CGM");
        window.addReading(BASE + 2 * MIN_5, 100.0, "CGM");
        window.sortByTime();

        Module10GlucoseAnalyzer.Result result = Module10GlucoseAnalyzer.analyze(window);

        // iAUC: triangle above baseline
        // Segment 1: (0 + 40)/2 * 300s = 6000
        // Segment 2: (40 + 0)/2 * 300s = 6000
        // Total = 12000 mg·s/dL
        assertEquals(12000.0, result.iAUC, 1.0);
        assertEquals(140.0, result.peak);
        assertEquals(40.0, result.excursion);
        assertEquals(5.0, result.timeToPeakMin);
    }

    @Test
    void iAUC_ignoresNegativeExcursions() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(120.0);
        // Reading below baseline should contribute 0 to iAUC
        window.addReading(BASE, 120.0, "CGM");
        window.addReading(BASE + MIN_5, 110.0, "CGM");  // below baseline
        window.addReading(BASE + 2 * MIN_5, 130.0, "CGM");
        window.sortByTime();

        Module10GlucoseAnalyzer.Result result = Module10GlucoseAnalyzer.analyze(window);

        // Segment 1: max(0,0) + max(0,-10) / 2 * 300 = 0
        // Segment 2: max(0,-10) + max(0,10) / 2 * 300 = 1500
        assertEquals(1500.0, result.iAUC, 1.0);
    }

    @Test
    void peakAndTimeToPeak_correctForMultipleReadings() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(90.0);
        window.addReading(BASE, 90.0, "CGM");
        window.addReading(BASE + 10 * MIN_5, 95.0, "CGM");   // 50 min
        window.addReading(BASE + 12 * MIN_5, 180.0, "CGM");   // 60 min — peak
        window.addReading(BASE + 18 * MIN_5, 120.0, "CGM");   // 90 min
        window.sortByTime();

        Module10GlucoseAnalyzer.Result result = Module10GlucoseAnalyzer.analyze(window);

        assertEquals(180.0, result.peak);
        assertEquals(90.0, result.excursion);
        assertEquals(60.0, result.timeToPeakMin);
    }

    @Test
    void recoveryTime_timeToReturnWithin10pctOfBaseline() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(100.0);
        window.addReading(BASE, 100.0, "CGM");
        window.addReading(BASE + 6 * MIN_5, 160.0, "CGM");   // 30 min — peak
        window.addReading(BASE + 12 * MIN_5, 130.0, "CGM");  // 60 min — still high
        window.addReading(BASE + 18 * MIN_5, 108.0, "CGM");  // 90 min — within 10%
        window.sortByTime();

        Module10GlucoseAnalyzer.Result result = Module10GlucoseAnalyzer.analyze(window);

        // Recovery = time from peak to first reading within 10% of baseline
        // Peak at 30 min, recovery at 90 min → 60 min recovery
        assertEquals(60.0, result.recoveryTimeMin, 1.0);
    }

    @Test
    void emptyWindow_returnsNullResult() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(100.0);

        Module10GlucoseAnalyzer.Result result = Module10GlucoseAnalyzer.analyze(window);
        assertNull(result);
    }

    @Test
    void singleReading_returnsMinimalResult() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(100.0);
        window.addReading(BASE, 100.0, "SMBG");
        window.sortByTime();

        Module10GlucoseAnalyzer.Result result = Module10GlucoseAnalyzer.analyze(window);
        // Single reading: peak = that reading, iAUC = 0, no recovery
        assertNotNull(result);
        assertEquals(0.0, result.iAUC);
        assertEquals(100.0, result.peak);
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module10GlucoseAnalyzerTest -q 2>&1 | tail -10`
Expected: FAIL — Module10GlucoseAnalyzer class does not exist

- [ ] **Step 3: Write Module10GlucoseAnalyzer implementation**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.GlucoseWindow;
import java.util.List;

/**
 * Glucose feature extraction for Module 10.
 *
 * Computes 8 features from a GlucoseWindow:
 * 1. baseline — first reading (or explicit baseline from state)
 * 2. peak — max glucose value in window
 * 3. excursion — peak - baseline
 * 4. timeToPeakMin — minutes from first reading to peak
 * 5. iAUC — incremental area under curve (trapezoidal, positive-only above baseline)
 * 6. recoveryTimeMin — minutes from peak to first reading within 10% of baseline
 * 7. readingCount — number of readings in window
 * 8. qualityScore — 0-1 based on reading density
 *
 * Stateless utility class — all computation from GlucoseWindow.
 */
public class Module10GlucoseAnalyzer {

    private Module10GlucoseAnalyzer() {}

    /**
     * Analyze a glucose window. Returns null if window is empty.
     */
    public static Result analyze(GlucoseWindow window) {
        if (window == null || window.isEmpty()) return null;

        window.sortByTime();
        List<GlucoseWindow.Reading> readings = window.getReadings();
        double baseline = window.getBaseline() != null
            ? window.getBaseline()
            : readings.get(0).value;

        if (readings.size() == 1) {
            Result r = new Result();
            r.baseline = baseline;
            r.peak = readings.get(0).value;
            r.excursion = r.peak - baseline;
            r.timeToPeakMin = 0.0;
            r.iAUC = 0.0;
            r.recoveryTimeMin = null;
            r.readingCount = 1;
            r.qualityScore = 0.1;
            return r;
        }

        // Find peak
        double peak = Double.NEGATIVE_INFINITY;
        int peakIndex = 0;
        for (int i = 0; i < readings.size(); i++) {
            if (readings.get(i).value > peak) {
                peak = readings.get(i).value;
                peakIndex = i;
            }
        }

        long firstTime = readings.get(0).timestamp;
        double timeToPeakMin = (readings.get(peakIndex).timestamp - firstTime) / 60_000.0;

        // Compute iAUC (trapezoidal rule, positive-only above baseline)
        double iAUC = 0.0;
        for (int i = 0; i < readings.size() - 1; i++) {
            double g1 = Math.max(0.0, readings.get(i).value - baseline);
            double g2 = Math.max(0.0, readings.get(i + 1).value - baseline);
            double dtSeconds = (readings.get(i + 1).timestamp - readings.get(i).timestamp) / 1000.0;
            iAUC += (g1 + g2) / 2.0 * dtSeconds;
        }

        // Recovery time: from peak, find first reading within 10% of baseline
        Double recoveryTimeMin = null;
        double recoveryThreshold = baseline * 1.10;
        for (int i = peakIndex + 1; i < readings.size(); i++) {
            if (readings.get(i).value <= recoveryThreshold) {
                recoveryTimeMin = (readings.get(i).timestamp - readings.get(peakIndex).timestamp) / 60_000.0;
                break;
            }
        }

        // Quality score: based on reading density (ideal: 1 reading per 5 min for 3h = 36 readings)
        long windowDurationMs = readings.get(readings.size() - 1).timestamp - firstTime;
        double expectedReadings = Math.max(1, windowDurationMs / 300_000.0); // 5-min intervals
        double qualityScore = Math.min(1.0, readings.size() / expectedReadings);

        Result r = new Result();
        r.baseline = baseline;
        r.peak = peak;
        r.excursion = peak - baseline;
        r.timeToPeakMin = timeToPeakMin;
        r.iAUC = iAUC;
        r.recoveryTimeMin = recoveryTimeMin;
        r.readingCount = readings.size();
        r.qualityScore = qualityScore;
        return r;
    }

    public static class Result {
        public double baseline;
        public double peak;
        public double excursion;
        public double timeToPeakMin;
        public double iAUC;
        public Double recoveryTimeMin;
        public int readingCount;
        public double qualityScore;
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module10GlucoseAnalyzerTest -q 2>&1 | tail -10`
Expected: PASS (6 tests)

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module10GlucoseAnalyzer.java backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module10GlucoseAnalyzerTest.java
git commit -m "feat(module10): add Module10GlucoseAnalyzer with trapezoidal iAUC and 6 tests"
```

---

### Task 8: Create Module10CurveClassifier

**Files:**
- Create: `operators/Module10CurveClassifier.java`
- Test: `operators/Module10CurveClassifierTest.java`

- [ ] **Step 1: Write the failing test**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module10CurveClassifierTest {

    private static final long BASE = 1743552000000L;
    private static final long MIN_5 = 5 * 60_000L;

    @Test
    void rapidSpike_peakWithin30min() {
        GlucoseWindow window = cgmWindow(
            100, 130, 160, 180, 170, 140, 110, 105, 100, 98, 97, 96
        );
        // Peak at index 3 = 15 min → RAPID_SPIKE
        CurveShape shape = Module10CurveClassifier.classify(window);
        assertEquals(CurveShape.RAPID_SPIKE, shape);
    }

    @Test
    void slowRise_peakAfter60min() {
        GlucoseWindow window = cgmWindow(
            100, 105, 110, 115, 120, 125, 130, 135, 140, 150, 160, 155, 140, 120, 105
        );
        // Peak at index 10 = 50 min → SLOW_RISE
        CurveShape shape = Module10CurveClassifier.classify(window);
        assertEquals(CurveShape.SLOW_RISE, shape);
    }

    @Test
    void plateau_sustainedElevation() {
        GlucoseWindow window = cgmWindow(
            100, 120, 140, 150, 152, 151, 150, 149, 148, 147, 146, 145
        );
        // After peak, fall rate < 0.5 mg/dL/min for extended period → PLATEAU
        CurveShape shape = Module10CurveClassifier.classify(window);
        assertEquals(CurveShape.PLATEAU, shape);
    }

    @Test
    void doublePeak_twoPeaksSeparatedBy20min() {
        GlucoseWindow window = cgmWindow(
            100, 140, 160, 130, 120, 115, 120, 145, 155, 140, 120, 105
        );
        // Peak at index 2 and index 8, separated by 30 min → DOUBLE_PEAK
        CurveShape shape = Module10CurveClassifier.classify(window);
        assertEquals(CurveShape.DOUBLE_PEAK, shape);
    }

    @Test
    void flat_excursionBelow20() {
        GlucoseWindow window = cgmWindow(
            100, 102, 105, 108, 110, 112, 110, 108, 105, 103, 101, 100
        );
        // Max excursion = 12 < 20 → FLAT
        CurveShape shape = Module10CurveClassifier.classify(window);
        assertEquals(CurveShape.FLAT, shape);
    }

    @Test
    void unknown_insufficientReadings() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(100.0);
        window.addReading(BASE, 100.0, "CGM");
        window.addReading(BASE + MIN_5, 130.0, "CGM");
        window.sortByTime();
        // Only 2 readings < MIN_READINGS_FOR_CLASSIFICATION (6)
        CurveShape shape = Module10CurveClassifier.classify(window);
        assertEquals(CurveShape.UNKNOWN, shape);
    }

    /** Helper: create CGM window with 5-min interval readings starting at BASE */
    private GlucoseWindow cgmWindow(double... values) {
        GlucoseWindow w = new GlucoseWindow();
        w.setBaseline(values[0]);
        w.setDataTier(DataTier.TIER_1_CGM);
        for (int i = 0; i < values.length; i++) {
            w.addReading(BASE + i * MIN_5, values[i], "CGM");
        }
        w.sortByTime();
        return w;
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module10CurveClassifierTest -q 2>&1 | tail -10`
Expected: FAIL — Module10CurveClassifier class does not exist

- [ ] **Step 3: Write Module10CurveClassifier implementation**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CurveShape;
import com.cardiofit.flink.models.GlucoseWindow;
import java.util.List;

/**
 * Curve shape classification for Module 10.
 *
 * Algorithm:
 * 1. Apply 3-point moving average smoothing
 * 2. Compute first derivative (rate of change per 5 min)
 * 3. Classify based on:
 *    - FLAT: max excursion < 20 mg/dL
 *    - RAPID_SPIKE: peak within 30 min of first reading
 *    - DOUBLE_PEAK: two distinct peaks separated by ≥20 min
 *    - PLATEAU: post-peak fall rate < 0.5 mg/dL/min for ≥15 min
 *    - SLOW_RISE: default if peak after 30 min
 *
 * Requires Tier 1 (CGM) with ≥6 readings.
 */
public class Module10CurveClassifier {

    private static final double FLAT_THRESHOLD = 20.0;       // mg/dL
    private static final double RAPID_SPIKE_PEAK_MIN = 30.0; // minutes
    private static final double PLATEAU_FALL_RATE = 0.5;     // mg/dL per minute
    private static final double PLATEAU_DURATION_MIN = 15.0;
    private static final double DOUBLE_PEAK_GAP_MIN = 20.0;  // minutes between peaks
    private static final double DOUBLE_PEAK_DIP_FRACTION = 0.3; // dip must be ≥30% of excursion

    private Module10CurveClassifier() {}

    public static CurveShape classify(GlucoseWindow window) {
        if (window == null || window.size() < CurveShape.MIN_READINGS_FOR_CLASSIFICATION) {
            return CurveShape.UNKNOWN;
        }

        window.sortByTime();
        List<GlucoseWindow.Reading> readings = window.getReadings();
        double baseline = window.getBaseline() != null ? window.getBaseline() : readings.get(0).value;

        // Step 1: 3-point moving average smoothing
        double[] smoothed = smooth(readings);

        // Step 2: Find peak
        int peakIdx = 0;
        double peakVal = smoothed[0];
        for (int i = 1; i < smoothed.length; i++) {
            if (smoothed[i] > peakVal) {
                peakVal = smoothed[i];
                peakIdx = i;
            }
        }

        double excursion = peakVal - baseline;

        // Rule 1: FLAT — excursion below threshold
        if (excursion < FLAT_THRESHOLD) {
            return CurveShape.FLAT;
        }

        // Rule 2: DOUBLE_PEAK — check for two distinct peaks
        if (hasDoublePeak(smoothed, baseline, excursion, readings)) {
            return CurveShape.DOUBLE_PEAK;
        }

        // Rule 3: PLATEAU — sustained elevation after peak
        if (isPlateau(smoothed, peakIdx, readings)) {
            return CurveShape.PLATEAU;
        }

        // Rule 4: RAPID_SPIKE vs SLOW_RISE — based on time to peak
        double timeToPeakMin = (readings.get(peakIdx).timestamp - readings.get(0).timestamp) / 60_000.0;
        if (timeToPeakMin <= RAPID_SPIKE_PEAK_MIN) {
            return CurveShape.RAPID_SPIKE;
        }

        return CurveShape.SLOW_RISE;
    }

    private static double[] smooth(List<GlucoseWindow.Reading> readings) {
        double[] smoothed = new double[readings.size()];
        smoothed[0] = readings.get(0).value;
        smoothed[readings.size() - 1] = readings.get(readings.size() - 1).value;
        for (int i = 1; i < readings.size() - 1; i++) {
            smoothed[i] = (readings.get(i - 1).value + readings.get(i).value + readings.get(i + 1).value) / 3.0;
        }
        return smoothed;
    }

    private static boolean hasDoublePeak(double[] smoothed, double baseline, double excursion,
                                          List<GlucoseWindow.Reading> readings) {
        // Find first peak
        int peak1 = -1;
        for (int i = 1; i < smoothed.length - 1; i++) {
            if (smoothed[i] > smoothed[i - 1] && smoothed[i] >= smoothed[i + 1]
                    && (smoothed[i] - baseline) > excursion * 0.5) {
                peak1 = i;
                break;
            }
        }
        if (peak1 < 0) return false;

        // Find valley after first peak
        int valley = -1;
        double valleyVal = smoothed[peak1];
        for (int i = peak1 + 1; i < smoothed.length; i++) {
            if (smoothed[i] < valleyVal) {
                valleyVal = smoothed[i];
                valley = i;
            }
            if (smoothed[i] > valleyVal + excursion * DOUBLE_PEAK_DIP_FRACTION) break;
        }
        if (valley < 0) return false;

        double dipDepth = smoothed[peak1] - valleyVal;
        if (dipDepth < excursion * DOUBLE_PEAK_DIP_FRACTION) return false;

        // Find second peak after valley
        for (int i = valley + 1; i < smoothed.length - 1; i++) {
            if (smoothed[i] > smoothed[i - 1] && smoothed[i] >= smoothed[i + 1]
                    && (smoothed[i] - baseline) > excursion * 0.4) {
                double gapMin = (readings.get(i).timestamp - readings.get(peak1).timestamp) / 60_000.0;
                if (gapMin >= DOUBLE_PEAK_GAP_MIN) {
                    return true;
                }
            }
        }
        return false;
    }

    private static boolean isPlateau(double[] smoothed, int peakIdx,
                                      List<GlucoseWindow.Reading> readings) {
        if (peakIdx >= smoothed.length - 2) return false;

        double sustainedStart = readings.get(peakIdx).timestamp;
        for (int i = peakIdx + 1; i < smoothed.length; i++) {
            double dtMin = (readings.get(i).timestamp - readings.get(i - 1).timestamp) / 60_000.0;
            if (dtMin <= 0) continue;
            double fallRate = (smoothed[i - 1] - smoothed[i]) / dtMin;
            if (fallRate > PLATEAU_FALL_RATE) {
                // Check if we sustained long enough before the drop
                double sustainedMin = (readings.get(i - 1).timestamp - sustainedStart) / 60_000.0;
                return sustainedMin >= PLATEAU_DURATION_MIN;
            }
        }

        // If we never dropped fast, entire post-peak is a plateau
        double totalSustainedMin = (readings.get(smoothed.length - 1).timestamp - sustainedStart) / 60_000.0;
        return totalSustainedMin >= PLATEAU_DURATION_MIN;
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module10CurveClassifierTest -q 2>&1 | tail -10`
Expected: PASS (6 tests)

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module10CurveClassifier.java backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module10CurveClassifierTest.java
git commit -m "feat(module10): add Module10CurveClassifier with 5 shape classification and 6 tests"
```

---

### Task 9: Create Module10BPCorrelator

**Files:**
- Create: `operators/Module10BPCorrelator.java`
- Test: `operators/Module10BPCorrelatorTest.java`

- [ ] **Step 1: Write the failing test**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module10BPCorrelatorTest {

    @Test
    void completeBP_computesExcursion() {
        BPWindow bpWindow = new BPWindow();
        bpWindow.setPreMealSBP(120.0);
        bpWindow.setPreMealDBP(80.0);
        bpWindow.setPostMealSBP(135.0);
        bpWindow.setPostMealDBP(85.0);

        Module10BPCorrelator.Result result = Module10BPCorrelator.analyze(bpWindow);

        assertNotNull(result);
        assertEquals(15.0, result.sbpExcursion);
        assertTrue(result.complete);
    }

    @Test
    void negativeExcursion_bpDropsPostMeal() {
        BPWindow bpWindow = new BPWindow();
        bpWindow.setPreMealSBP(140.0);
        bpWindow.setPreMealDBP(90.0);
        bpWindow.setPostMealSBP(125.0);
        bpWindow.setPostMealDBP(82.0);

        Module10BPCorrelator.Result result = Module10BPCorrelator.analyze(bpWindow);

        assertNotNull(result);
        assertEquals(-15.0, result.sbpExcursion);
        assertTrue(result.complete);
    }

    @Test
    void missingPreMeal_incompleteResult() {
        BPWindow bpWindow = new BPWindow();
        bpWindow.setPostMealSBP(130.0);
        bpWindow.setPostMealDBP(85.0);

        Module10BPCorrelator.Result result = Module10BPCorrelator.analyze(bpWindow);

        assertNotNull(result);
        assertNull(result.sbpExcursion);
        assertFalse(result.complete);
        assertEquals(130.0, result.postMealSBP);
    }

    @Test
    void missingPostMeal_incompleteResult() {
        BPWindow bpWindow = new BPWindow();
        bpWindow.setPreMealSBP(120.0);
        bpWindow.setPreMealDBP(80.0);

        Module10BPCorrelator.Result result = Module10BPCorrelator.analyze(bpWindow);

        assertNotNull(result);
        assertNull(result.sbpExcursion);
        assertFalse(result.complete);
        assertEquals(120.0, result.preMealSBP);
    }

    @Test
    void nullWindow_returnsNull() {
        Module10BPCorrelator.Result result = Module10BPCorrelator.analyze(null);
        assertNull(result);
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module10BPCorrelatorTest -q 2>&1 | tail -10`
Expected: FAIL — Module10BPCorrelator class does not exist

- [ ] **Step 3: Write Module10BPCorrelator implementation**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.BPWindow;

/**
 * BP correlation analysis for Module 10.
 *
 * Computes pre-meal vs post-meal BP excursion from a BPWindow.
 * Pre-meal BP is retroactively attached from state buffer (most recent within 60 min).
 * Post-meal BP is the first reading within 4h after meal.
 *
 * Stateless utility class.
 */
public class Module10BPCorrelator {

    private Module10BPCorrelator() {}

    public static Result analyze(BPWindow bpWindow) {
        if (bpWindow == null) return null;

        Result r = new Result();
        r.preMealSBP = bpWindow.getPreMealSBP();
        r.preMealDBP = bpWindow.getPreMealDBP();
        r.postMealSBP = bpWindow.getPostMealSBP();
        r.postMealDBP = bpWindow.getPostMealDBP();
        r.complete = bpWindow.isComplete();

        if (r.complete) {
            r.sbpExcursion = bpWindow.getSBPExcursion();
        }

        return r;
    }

    public static class Result {
        public Double preMealSBP;
        public Double preMealDBP;
        public Double postMealSBP;
        public Double postMealDBP;
        public Double sbpExcursion;
        public boolean complete;
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module10BPCorrelatorTest -q 2>&1 | tail -10`
Expected: PASS (5 tests)

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module10BPCorrelator.java backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module10BPCorrelatorTest.java
git commit -m "feat(module10): add Module10BPCorrelator with pre/post SBP excursion and 5 tests"
```

---

### Task 10: Create Module10_MealResponseCorrelator (Main KPF)

**Files:**
- Create: `operators/Module10_MealResponseCorrelator.java`

This is the main KeyedProcessFunction. It follows the Module 9 pattern (timer-driven KPF) but uses a **session window** pattern instead of a daily timer.

- [ ] **Step 1: Create Module10_MealResponseCorrelator**

```java
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
 * - Processing-time timers (not event-time) ensure window closes even if CGM goes offline
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
            // CGM device reading with glucose_value
            return payload.containsKey("glucose_value");
        }
        if (type == EventType.LAB_RESULT) {
            // SMBG glucose lab result
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
            // CGM reading
            Object val = payload.get("glucose_value");
            if (!(val instanceof Number)) return;
            glucoseValue = ((Number) val).doubleValue();
            source = "CGM";

            // Upgrade data tier if we see CGM
            if (state.getDataTier() == DataTier.TIER_3_SMBG) {
                state.setDataTier(DataTier.TIER_1_CGM);
            }
        } else {
            // SMBG lab result
            Object val = payload.get("value");
            if (!(val instanceof Number)) return;
            glucoseValue = ((Number) val).doubleValue();
            source = "SMBG";

            // If we have CGM + SMBG → Tier 2 hybrid
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

        // Window duration
        long windowDuration = System.currentTimeMillis() - session.mealTimestamp;
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
```

- [ ] **Step 2: Compile**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module10_MealResponseCorrelator.java
git commit -m "feat(module10): add Module10_MealResponseCorrelator main KPF with session windows"
```

---

### Task 11: Create Module 10b Operators

**Files:**
- Create: `operators/Module10bSaltSensitivityEstimator.java`
- Create: `operators/Module10bFoodRanker.java`
- Test: `operators/Module10bSaltSensitivityTest.java`
- Test: `operators/Module10bFoodRankerTest.java`

- [ ] **Step 1: Write salt sensitivity failing test**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.ArrayList;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module10bSaltSensitivityTest {

    @Test
    void highSaltSensitivity_betaAbove005() {
        // Generate 40 pairs with strong positive correlation
        List<MealPatternState.SodiumSBPPair> pairs = new ArrayList<>();
        long now = System.currentTimeMillis();
        for (int i = 0; i < 40; i++) {
            double sodium = 200 + i * 50;  // 200-2150 mg range
            double sbpExcursion = 2.0 + sodium * 0.008;  // strong positive slope
            pairs.add(new MealPatternState.SodiumSBPPair(sodium, sbpExcursion, now - i * 86400000L));
        }

        Module10bSaltSensitivityEstimator.Result result =
            Module10bSaltSensitivityEstimator.estimate(pairs);

        assertEquals(SaltSensitivityClass.HIGH, result.classification);
        assertTrue(result.beta >= 0.005);
        assertTrue(result.rSquared > 0.1);
        assertEquals(40, result.pairCount);
    }

    @Test
    void saltResistant_betaBelow0001() {
        List<MealPatternState.SodiumSBPPair> pairs = new ArrayList<>();
        long now = System.currentTimeMillis();
        for (int i = 0; i < 35; i++) {
            double sodium = 300 + i * 40;
            double sbpExcursion = 5.0 + Math.random() * 0.5;  // nearly flat
            pairs.add(new MealPatternState.SodiumSBPPair(sodium, sbpExcursion, now - i * 86400000L));
        }

        Module10bSaltSensitivityEstimator.Result result =
            Module10bSaltSensitivityEstimator.estimate(pairs);

        assertEquals(SaltSensitivityClass.SALT_RESISTANT, result.classification);
        assertTrue(result.beta < 0.001);
    }

    @Test
    void undetermined_tooFewPairs() {
        List<MealPatternState.SodiumSBPPair> pairs = new ArrayList<>();
        long now = System.currentTimeMillis();
        for (int i = 0; i < 15; i++) {
            pairs.add(new MealPatternState.SodiumSBPPair(500 + i * 100, 5.0 + i, now));
        }

        Module10bSaltSensitivityEstimator.Result result =
            Module10bSaltSensitivityEstimator.estimate(pairs);

        assertEquals(SaltSensitivityClass.UNDETERMINED, result.classification);
        assertEquals(15, result.pairCount);
    }

    @Test
    void undetermined_lowRSquared() {
        // Random noise with no correlation
        List<MealPatternState.SodiumSBPPair> pairs = new ArrayList<>();
        long now = System.currentTimeMillis();
        java.util.Random rng = new java.util.Random(42);
        for (int i = 0; i < 40; i++) {
            pairs.add(new MealPatternState.SodiumSBPPair(
                rng.nextDouble() * 2000, rng.nextDouble() * 30 - 15, now));
        }

        Module10bSaltSensitivityEstimator.Result result =
            Module10bSaltSensitivityEstimator.estimate(pairs);

        // R² should be low for random data
        if (result.rSquared < 0.1) {
            assertEquals(SaltSensitivityClass.UNDETERMINED, result.classification);
        }
    }

    @Test
    void emptyPairs_returnsUndetermined() {
        Module10bSaltSensitivityEstimator.Result result =
            Module10bSaltSensitivityEstimator.estimate(new ArrayList<>());

        assertEquals(SaltSensitivityClass.UNDETERMINED, result.classification);
        assertEquals(0, result.pairCount);
    }

    @Test
    void divisionByZeroGuard_allSameSodium() {
        // SS_xx = 0 when all x values identical
        List<MealPatternState.SodiumSBPPair> pairs = new ArrayList<>();
        long now = System.currentTimeMillis();
        for (int i = 0; i < 35; i++) {
            pairs.add(new MealPatternState.SodiumSBPPair(500.0, 5.0 + i * 0.1, now));
        }

        Module10bSaltSensitivityEstimator.Result result =
            Module10bSaltSensitivityEstimator.estimate(pairs);

        // Should not throw, should return UNDETERMINED
        assertEquals(SaltSensitivityClass.UNDETERMINED, result.classification);
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module10bSaltSensitivityTest -q 2>&1 | tail -10`
Expected: FAIL — class does not exist

- [ ] **Step 3: Write Module10bSaltSensitivityEstimator**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.MealPatternState;
import com.cardiofit.flink.models.SaltSensitivityClass;
import java.util.List;

/**
 * OLS linear regression estimator for salt sensitivity.
 *
 * Regresses SBP_excursion on sodium_mg from a 60-day rolling buffer.
 * β (slope) classifies salt sensitivity:
 *   β < 0.001         → SALT_RESISTANT
 *   0.001 ≤ β < 0.005 → MODERATE
 *   β ≥ 0.005         → HIGH
 *   n < 30 or R² < 0.1 → UNDETERMINED
 *
 * Three division-by-zero guards:
 * 1. n = 0 → UNDETERMINED
 * 2. SS_xx = 0 (all sodium values identical) → UNDETERMINED
 * 3. SS_tot = 0 (all SBP excursions identical) → R² = 0 → UNDETERMINED if β threshold not met
 */
public class Module10bSaltSensitivityEstimator {

    private Module10bSaltSensitivityEstimator() {}

    public static Result estimate(List<MealPatternState.SodiumSBPPair> pairs) {
        Result result = new Result();
        result.pairCount = pairs != null ? pairs.size() : 0;

        if (result.pairCount == 0) {
            result.classification = SaltSensitivityClass.UNDETERMINED;
            result.beta = 0.0;
            result.rSquared = 0.0;
            return result;
        }

        int n = result.pairCount;

        // Compute means
        double sumX = 0, sumY = 0;
        for (MealPatternState.SodiumSBPPair p : pairs) {
            sumX += p.sodiumMg;
            sumY += p.sbpExcursion;
        }
        double meanX = sumX / n;
        double meanY = sumY / n;

        // Compute SS_xx, SS_xy, SS_tot
        double ssXX = 0, ssXY = 0, ssTot = 0;
        for (MealPatternState.SodiumSBPPair p : pairs) {
            double dx = p.sodiumMg - meanX;
            double dy = p.sbpExcursion - meanY;
            ssXX += dx * dx;
            ssXY += dx * dy;
            ssTot += dy * dy;
        }

        // Guard 2: SS_xx = 0 (all x identical)
        if (ssXX < 1e-12) {
            result.beta = 0.0;
            result.rSquared = 0.0;
            result.classification = SaltSensitivityClass.UNDETERMINED;
            return result;
        }

        // OLS slope and intercept
        double beta = ssXY / ssXX;

        // R² = SS_reg / SS_tot = (beta² * SS_xx) / SS_tot
        double ssReg = beta * beta * ssXX;
        double rSquared = (ssTot > 1e-12) ? ssReg / ssTot : 0.0;

        result.beta = beta;
        result.rSquared = rSquared;
        result.classification = SaltSensitivityClass.fromBetaAndR2(beta, rSquared, n);

        return result;
    }

    public static class Result {
        public SaltSensitivityClass classification;
        public double beta;
        public double rSquared;
        public int pairCount;
    }
}
```

- [ ] **Step 4: Run salt sensitivity tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module10bSaltSensitivityTest -q 2>&1 | tail -10`
Expected: PASS (6 tests)

- [ ] **Step 5: Write food ranker failing test**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.HashMap;
import static org.junit.jupiter.api.Assertions.*;

class Module10bFoodRankerTest {

    @Test
    void ranksTopFoodsByMeanExcursion() {
        List<MealResponseRecord> records = new ArrayList<>();
        records.add(mealWith("rice", 60.0, 8000.0));
        records.add(mealWith("rice", 55.0, 7500.0));
        records.add(mealWith("salad", 15.0, 2000.0));
        records.add(mealWith("bread", 45.0, 6000.0));
        records.add(mealWith("bread", 50.0, 6500.0));

        List<MealPatternSummary.FoodImpact> ranked =
            Module10bFoodRanker.rank(records, 5);

        assertEquals(3, ranked.size());
        // Rice has highest mean excursion (57.5)
        assertEquals("rice", ranked.get(0).foodDescription);
        assertEquals(2, ranked.get(0).mealCount);
        assertEquals(57.5, ranked.get(0).meanExcursion, 0.1);
        // Bread second (47.5)
        assertEquals("bread", ranked.get(1).foodDescription);
    }

    @Test
    void limitOutput_topN() {
        List<MealResponseRecord> records = new ArrayList<>();
        for (int i = 0; i < 20; i++) {
            records.add(mealWith("food_" + i, 10.0 + i * 5, 1000.0 + i * 500));
        }

        List<MealPatternSummary.FoodImpact> ranked =
            Module10bFoodRanker.rank(records, 3);

        assertEquals(3, ranked.size());
    }

    @Test
    void emptyRecords_returnsEmpty() {
        List<MealPatternSummary.FoodImpact> ranked =
            Module10bFoodRanker.rank(new ArrayList<>(), 5);
        assertTrue(ranked.isEmpty());
    }

    @Test
    void skipsRecordsWithNoFoodDescription() {
        List<MealResponseRecord> records = new ArrayList<>();
        records.add(mealWith(null, 50.0, 7000.0));
        records.add(mealWith("rice", 60.0, 8000.0));

        List<MealPatternSummary.FoodImpact> ranked =
            Module10bFoodRanker.rank(records, 5);

        assertEquals(1, ranked.size());
        assertEquals("rice", ranked.get(0).foodDescription);
    }

    private MealResponseRecord mealWith(String food, double excursion, double iauc) {
        Map<String, Object> payload = new HashMap<>();
        if (food != null) payload.put("food_description", food);

        return MealResponseRecord.builder()
            .recordId("test-" + System.nanoTime())
            .patientId("P1")
            .mealEventId("meal-" + System.nanoTime())
            .mealTimestamp(System.currentTimeMillis())
            .glucoseExcursion(excursion)
            .iAUC(iauc)
            .mealPayload(payload)
            .build();
    }
}
```

- [ ] **Step 6: Write Module10bFoodRanker**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.MealPatternSummary;
import com.cardiofit.flink.models.MealResponseRecord;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Food impact ranker for Module 10b.
 *
 * Groups meal records by food_description (from mealPayload),
 * computes mean excursion and mean iAUC per food,
 * returns top-N foods ranked by mean excursion descending.
 */
public class Module10bFoodRanker {

    private Module10bFoodRanker() {}

    public static List<MealPatternSummary.FoodImpact> rank(
            List<MealResponseRecord> records, int topN) {
        if (records == null || records.isEmpty()) return Collections.emptyList();

        // Group by food_description
        Map<String, List<MealResponseRecord>> byFood = new LinkedHashMap<>();
        for (MealResponseRecord r : records) {
            String food = extractFoodDescription(r);
            if (food == null || food.isBlank()) continue;
            byFood.computeIfAbsent(food, k -> new ArrayList<>()).add(r);
        }

        // Compute mean excursion + mean iAUC per food
        List<MealPatternSummary.FoodImpact> impacts = new ArrayList<>();
        for (Map.Entry<String, List<MealResponseRecord>> entry : byFood.entrySet()) {
            List<MealResponseRecord> meals = entry.getValue();
            double sumExcursion = 0, sumIAUC = 0;
            int countExcursion = 0, countIAUC = 0;

            for (MealResponseRecord m : meals) {
                if (m.getGlucoseExcursion() != null) {
                    sumExcursion += m.getGlucoseExcursion();
                    countExcursion++;
                }
                if (m.getIAUC() != null) {
                    sumIAUC += m.getIAUC();
                    countIAUC++;
                }
            }

            double meanExcursion = countExcursion > 0 ? sumExcursion / countExcursion : 0.0;
            double meanIAUC = countIAUC > 0 ? sumIAUC / countIAUC : 0.0;

            impacts.add(new MealPatternSummary.FoodImpact(
                entry.getKey(), meals.size(), meanExcursion, meanIAUC));
        }

        // Sort by mean excursion descending, take top N
        impacts.sort((a, b) -> Double.compare(b.meanExcursion, a.meanExcursion));
        return impacts.stream().limit(topN).collect(Collectors.toList());
    }

    private static String extractFoodDescription(MealResponseRecord record) {
        if (record.getMealPayload() == null) return null;
        Object desc = record.getMealPayload().get("food_description");
        return desc != null ? desc.toString().toLowerCase().trim() : null;
    }
}
```

- [ ] **Step 7: Run food ranker tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module10bFoodRankerTest -q 2>&1 | tail -10`
Expected: PASS (4 tests)

- [ ] **Step 8: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module10bSaltSensitivityEstimator.java backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module10bFoodRanker.java backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module10bSaltSensitivityTest.java backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module10bFoodRankerTest.java
git commit -m "feat(module10b): add SaltSensitivityEstimator (OLS) and FoodRanker with 10 tests"
```

---

### Task 12: Create Module10b_MealPatternAggregator

**Files:**
- Create: `operators/Module10b_MealPatternAggregator.java`

- [ ] **Step 1: Create Module10b_MealPatternAggregator**

```java
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
import java.time.Instant;
import java.time.ZoneOffset;
import java.time.ZonedDateTime;
import java.time.DayOfWeek;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Module 10b: Meal Pattern Aggregator — weekly KPF.
 *
 * Consumes MealResponseRecord (output of Module 10).
 * Accumulates records for 7 days, then on weekly timer (Monday 00:00 UTC):
 * 1. Computes per-meal-time stats (mean excursion, mean iAUC, dominant curve)
 * 2. Runs salt sensitivity OLS regression (60-day rolling buffer)
 * 3. Ranks foods by impact
 * 4. Emits MealPatternSummary
 *
 * Separate Flink job from Module 10 for failure isolation.
 * Input: MealResponseRecord from flink.meal-response
 * Output: MealPatternSummary to flink.meal-patterns
 *
 * State TTL: 60 days (salt sensitivity buffer).
 */
public class Module10b_MealPatternAggregator
        extends KeyedProcessFunction<String, MealResponseRecord, MealPatternSummary> {

    private static final Logger LOG = LoggerFactory.getLogger(Module10b_MealPatternAggregator.class);
    private static final int TOP_FOODS = 5;

    private transient ValueState<MealPatternState> patternState;

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        ValueStateDescriptor<MealPatternState> stateDesc =
            new ValueStateDescriptor<>("meal-pattern-state", MealPatternState.class);
        StateTtlConfig ttl = StateTtlConfig
            .newBuilder(Duration.ofDays(60))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();
        stateDesc.enableTimeToLive(ttl);
        patternState = getRuntimeContext().getState(stateDesc);

        LOG.info("Module 10b Meal Pattern Aggregator initialized (weekly, 60d salt buffer)");
    }

    @Override
    public void processElement(MealResponseRecord record, Context ctx,
                                Collector<MealPatternSummary> out) throws Exception {
        MealPatternState state = patternState.value();
        if (state == null) {
            state = new MealPatternState(record.getPatientId());
            state.setDataTier(record.getDataTier());
        }

        state.addMealRecord(record);

        // Register weekly timer (Monday 00:00 UTC) — once per patient
        if (!state.isWeeklyTimerRegistered()) {
            long nextMonday = computeNextMonday(ctx.timerService().currentProcessingTime());
            ctx.timerService().registerProcessingTimeTimer(nextMonday);
            state.setWeeklyTimerRegistered(true);
            LOG.debug("Registered weekly timer for patient={} at {}",
                record.getPatientId(), Instant.ofEpochMilli(nextMonday));
        }

        state.setLastUpdated(ctx.timerService().currentProcessingTime());
        patternState.update(state);
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx,
                         Collector<MealPatternSummary> out) throws Exception {
        MealPatternState state = patternState.value();
        if (state == null) return;

        List<MealResponseRecord> weeklyRecords = state.drainWeeklyRecords();
        if (!weeklyRecords.isEmpty()) {
            MealPatternSummary summary = buildSummary(state, weeklyRecords, timestamp);
            out.collect(summary);

            LOG.info("Weekly meal pattern emitted: patient={}, meals={}, salt={}",
                state.getPatientId(), weeklyRecords.size(),
                summary.getSaltSensitivityClass());
        }

        state.setLastWeeklyEmitTimestamp(timestamp);

        // Re-register next Monday timer
        long nextMonday = computeNextMonday(timestamp);
        ctx.timerService().registerProcessingTimeTimer(nextMonday);

        patternState.update(state);
    }

    private MealPatternSummary buildSummary(MealPatternState state,
                                             List<MealResponseRecord> records,
                                             long timestamp) {
        MealPatternSummary summary = new MealPatternSummary();
        summary.setSummaryId("m10b-" + UUID.randomUUID());
        summary.setPatientId(state.getPatientId());
        summary.setDataTier(state.getDataTier());
        summary.setTotalMealsInPeriod(records.size());

        // Period range
        long minTs = records.stream().mapToLong(MealResponseRecord::getMealTimestamp).min().orElse(timestamp);
        long maxTs = records.stream().mapToLong(MealResponseRecord::getMealTimestamp).max().orElse(timestamp);
        summary.setPeriodStartMs(minTs);
        summary.setPeriodEndMs(maxTs);

        // Aggregated glucose metrics
        List<MealResponseRecord> withGlucose = records.stream()
            .filter(r -> r.getGlucoseExcursion() != null)
            .collect(Collectors.toList());
        summary.setMealsWithGlucose(withGlucose.size());

        if (!withGlucose.isEmpty()) {
            double meanIAUC = withGlucose.stream()
                .filter(r -> r.getIAUC() != null)
                .mapToDouble(MealResponseRecord::getIAUC)
                .average().orElse(0.0);
            summary.setMeanIAUC(meanIAUC);

            List<Double> excursions = withGlucose.stream()
                .map(MealResponseRecord::getGlucoseExcursion)
                .sorted()
                .collect(Collectors.toList());
            double median = excursions.get(excursions.size() / 2);
            summary.setMedianExcursion(median);

            double meanTTP = withGlucose.stream()
                .filter(r -> r.getTimeToPeakMin() != null)
                .mapToDouble(MealResponseRecord::getTimeToPeakMin)
                .average().orElse(0.0);
            summary.setMeanTimeToPeakMin(meanTTP);

            // Dominant curve shape (mode)
            Map<CurveShape, Long> shapeCounts = withGlucose.stream()
                .filter(r -> r.getCurveShape() != null && r.getCurveShape() != CurveShape.UNKNOWN)
                .collect(Collectors.groupingBy(MealResponseRecord::getCurveShape, Collectors.counting()));
            if (!shapeCounts.isEmpty()) {
                summary.setDominantCurveShape(
                    shapeCounts.entrySet().stream()
                        .max(Map.Entry.comparingByValue())
                        .get().getKey());
            }
        }

        // Per-meal-time breakdown
        Map<MealTimeCategory, MealPatternSummary.MealTimeStats> breakdown = new LinkedHashMap<>();
        Map<MealTimeCategory, List<MealResponseRecord>> byTime = records.stream()
            .filter(r -> r.getMealTimeCategory() != null)
            .collect(Collectors.groupingBy(MealResponseRecord::getMealTimeCategory));

        for (Map.Entry<MealTimeCategory, List<MealResponseRecord>> entry : byTime.entrySet()) {
            MealPatternSummary.MealTimeStats stats = new MealPatternSummary.MealTimeStats();
            List<MealResponseRecord> meals = entry.getValue();
            stats.mealCount = meals.size();
            stats.meanExcursion = meals.stream()
                .filter(r -> r.getGlucoseExcursion() != null)
                .mapToDouble(MealResponseRecord::getGlucoseExcursion)
                .average().orElse(0.0);
            stats.meanIAUC = meals.stream()
                .filter(r -> r.getIAUC() != null)
                .mapToDouble(MealResponseRecord::getIAUC)
                .average().orElse(0.0);
            breakdown.put(entry.getKey(), stats);
        }
        summary.setMealTimeBreakdown(breakdown);

        // Salt sensitivity (60-day rolling OLS)
        Module10bSaltSensitivityEstimator.Result saltResult =
            Module10bSaltSensitivityEstimator.estimate(state.getSodiumSBPPairs());
        summary.setSaltSensitivityClass(saltResult.classification);
        summary.setSaltBeta(saltResult.beta);
        summary.setSaltRSquared(saltResult.rSquared);
        summary.setSaltPairCount(saltResult.pairCount);

        // Food impact ranking
        summary.setTopFoodsByExcursion(Module10bFoodRanker.rank(records, TOP_FOODS));

        // Quality score
        double quality = Math.min(1.0, records.size() / 21.0); // 3 meals/day * 7 days = 21
        summary.setQualityScore(quality);

        return summary;
    }

    /**
     * Compute next Monday 00:00 UTC.
     */
    static long computeNextMonday(long currentTimeMs) {
        ZonedDateTime now = Instant.ofEpochMilli(currentTimeMs).atZone(ZoneOffset.UTC);
        ZonedDateTime nextMonday = now.toLocalDate()
            .with(java.time.temporal.TemporalAdjusters.next(DayOfWeek.MONDAY))
            .atStartOfDay(ZoneOffset.UTC);
        return nextMonday.toInstant().toEpochMilli();
    }
}
```

- [ ] **Step 2: Compile**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module10b_MealPatternAggregator.java
git commit -m "feat(module10b): add Module10b_MealPatternAggregator weekly KPF with OLS salt sensitivity"
```

---

### Task 13: Wire Module 10/10b into FlinkJobOrchestrator

**Files:**
- Modify: `FlinkJobOrchestrator.java`

- [ ] **Step 1: Add import for MealResponseRecord and MealPatternSummary**

In `FlinkJobOrchestrator.java`, add imports after the existing model imports:

```java
import com.cardiofit.flink.models.MealResponseRecord;
import com.cardiofit.flink.models.MealPatternSummary;
```

- [ ] **Step 2: Add switch cases for module10 and module10b**

In the `main()` method switch statement, add before the `default:` case:

```java
            case "meal-response":
            case "module10":
            case "meal-response-correlator":
                launchMealResponseCorrelator(env);
                break;
            case "meal-patterns":
            case "module10b":
            case "meal-pattern-aggregator":
                launchMealPatternAggregator(env);
                break;
```

- [ ] **Step 3: Add launchMealResponseCorrelator method**

Add after the `launchEngagementMonitor()` method:

```java
    /**
     * Module 10: Meal Response Correlator.
     * Session-window-driven per-meal glucose/BP correlation.
     * Single sink: MealResponseRecord → flink.meal-response.
     */
    private static void launchMealResponseCorrelator(StreamExecutionEnvironment env) {
        LOG.info("Launching Module 10: Meal Response Correlator pipeline");

        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        KafkaSource<CanonicalEvent> source = KafkaSource.<CanonicalEvent>builder()
            .setBootstrapServers(bootstrap)
            .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
            .setGroupId("flink-module10-meal-response-v1")
            .setStartingOffsets(OffsetsInitializer.earliest())
            .setValueOnlyDeserializer(new CanonicalEventDeserializer())
            .build();

        SingleOutputStreamOperator<MealResponseRecord> records = env
            .fromSource(source,
                WatermarkStrategy.<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                    .withTimestampAssigner((e, ts) -> e.getEventTime()),
                "Kafka Source: Enriched Patient Events (Module 10)")
            .keyBy(CanonicalEvent::getPatientId)
            .process(new Module10_MealResponseCorrelator())
            .uid("module10-meal-response-correlator")
            .name("Module 10: Meal Response Correlator");

        // Main output → flink.meal-response
        records.sinkTo(
            KafkaSink.<MealResponseRecord>builder()
                .setBootstrapServers(bootstrap)
                .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                .setTransactionalIdPrefix("m10-meal-response")
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.<MealResponseRecord>builder()
                        .setTopic(KafkaTopics.FLINK_MEAL_RESPONSE.getTopicName())
                        .setValueSerializationSchema(new JsonSerializer<MealResponseRecord>())
                        .build())
                .build()
        ).name("Sink: Meal Response Records");

        LOG.info("Module 10 pipeline configured: source=[{}], sink=[{}]",
            KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName(),
            KafkaTopics.FLINK_MEAL_RESPONSE.getTopicName());
    }
```

- [ ] **Step 4: Add launchMealPatternAggregator method**

Add after `launchMealResponseCorrelator()`:

```java
    /**
     * Module 10b: Meal Pattern Aggregator.
     * Weekly aggregation of meal response records with OLS salt sensitivity.
     * Separate job from Module 10 for failure isolation.
     * Input: MealResponseRecord from flink.meal-response
     * Output: MealPatternSummary to flink.meal-patterns
     */
    private static void launchMealPatternAggregator(StreamExecutionEnvironment env) {
        LOG.info("Launching Module 10b: Meal Pattern Aggregator pipeline");

        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        // Source: MealResponseRecord from flink.meal-response (output of Module 10)
        KafkaSource<MealResponseRecord> source = KafkaSource.<MealResponseRecord>builder()
            .setBootstrapServers(bootstrap)
            .setTopics(KafkaTopics.FLINK_MEAL_RESPONSE.getTopicName())
            .setGroupId("flink-module10b-meal-patterns-v1")
            .setStartingOffsets(OffsetsInitializer.earliest())
            .setValueOnlyDeserializer(new MealResponseRecordDeserializer())
            .build();

        SingleOutputStreamOperator<MealPatternSummary> summaries = env
            .fromSource(source,
                WatermarkStrategy.<MealResponseRecord>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                    .withTimestampAssigner((r, ts) -> r.getMealTimestamp()),
                "Kafka Source: Meal Response Records (Module 10b)")
            .keyBy(MealResponseRecord::getPatientId)
            .process(new Module10b_MealPatternAggregator())
            .uid("module10b-meal-pattern-aggregator")
            .name("Module 10b: Meal Pattern Aggregator");

        // Output → flink.meal-patterns
        summaries.sinkTo(
            KafkaSink.<MealPatternSummary>builder()
                .setBootstrapServers(bootstrap)
                .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                .setTransactionalIdPrefix("m10b-meal-patterns")
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.<MealPatternSummary>builder()
                        .setTopic(KafkaTopics.FLINK_MEAL_PATTERNS.getTopicName())
                        .setValueSerializationSchema(new JsonSerializer<MealPatternSummary>())
                        .build())
                .build()
        ).name("Sink: Meal Pattern Summaries");

        LOG.info("Module 10b pipeline configured: source=[{}], sink=[{}]",
            KafkaTopics.FLINK_MEAL_RESPONSE.getTopicName(),
            KafkaTopics.FLINK_MEAL_PATTERNS.getTopicName());
    }

    /** Deserializes JSON bytes into a MealResponseRecord using Jackson. */
    static class MealResponseRecordDeserializer implements DeserializationSchema<MealResponseRecord> {
        private transient ObjectMapper mapper;

        @Override
        public void open(InitializationContext context) {
            mapper = new ObjectMapper();
            mapper.registerModule(new JavaTimeModule());
        }

        @Override
        public MealResponseRecord deserialize(byte[] message) throws IOException {
            if (message == null || message.length == 0) return null;
            return mapper.readValue(message, MealResponseRecord.class);
        }

        @Override
        public boolean isEndOfStream(MealResponseRecord nextElement) {
            return false;
        }

        @Override
        public TypeInformation<MealResponseRecord> getProducedType() {
            return TypeInformation.of(MealResponseRecord.class);
        }
    }
```

- [ ] **Step 5: Add Module 10/10b to launchFullPipeline**

In `launchFullPipeline()`, add after the Module 9 comment block:

```java
            // Module 10: Meal Response Correlator
            LOG.info("Initializing Module 10: Meal Response Correlator");
            launchMealResponseCorrelator(env);

            // Module 10b: Meal Pattern Aggregator
            LOG.info("Initializing Module 10b: Meal Pattern Aggregator");
            launchMealPatternAggregator(env);
```

- [ ] **Step 6: Compile**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 7: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java
git commit -m "feat(module10): wire Module 10 and 10b into FlinkJobOrchestrator"
```

---

### Task 14: Create Module10TestBuilder

**Files:**
- Create: `builders/Module10TestBuilder.java`

- [ ] **Step 1: Create Module10TestBuilder**

```java
package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

/**
 * Test builder for Module 10/10b tests.
 * Follows Module9TestBuilder pattern: static factory methods for events and state.
 */
public class Module10TestBuilder {

    private static final long DAY_MS = 86_400_000L;
    private static final long HOUR_MS = 3_600_000L;
    private static final long MIN_MS = 60_000L;
    public static final long BASE_TIME = 1743552000000L; // 2025-04-02 00:00:00 UTC

    public static long daysAgo(int days) { return BASE_TIME - (days * DAY_MS); }
    public static long hoursAgo(int hours) { return BASE_TIME - (hours * HOUR_MS); }
    public static long minutesAgo(int minutes) { return BASE_TIME - (minutes * MIN_MS); }
    public static long hoursAfter(long base, int hours) { return base + hours * HOUR_MS; }
    public static long minutesAfter(long base, int minutes) { return base + minutes * MIN_MS; }

    // --- Meal Events ---

    public static CanonicalEvent mealEvent(String patientId, long timestamp,
                                            double carbGrams, double proteinGrams) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("report_type", "MEAL_LOG");
        payload.put("meal_type", "lunch");
        payload.put("carb_grams", carbGrams);
        payload.put("protein_grams", proteinGrams);
        payload.put("food_description", "test meal");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent mealWithSodium(String patientId, long timestamp,
                                                  double carbGrams, double sodiumMg) {
        CanonicalEvent event = mealEvent(patientId, timestamp, carbGrams, 20.0);
        event.getPayload().put("sodium_mg", sodiumMg);
        return event;
    }

    // --- CGM Glucose Events ---

    public static CanonicalEvent cgmReading(String patientId, long timestamp, double glucose) {
        CanonicalEvent event = baseEvent(patientId, EventType.DEVICE_READING, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("glucose_value", glucose);
        payload.put("data_tier", "TIER_1_CGM");
        payload.put("source", "CGM");
        event.setPayload(payload);
        return event;
    }

    // --- SMBG Glucose Events ---

    public static CanonicalEvent smbgReading(String patientId, long timestamp, double glucose) {
        CanonicalEvent event = baseEvent(patientId, EventType.LAB_RESULT, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("lab_type", "glucose");
        payload.put("value", glucose);
        payload.put("source", "SMBG");
        event.setPayload(payload);
        return event;
    }

    // --- BP Events ---

    public static CanonicalEvent bpReading(String patientId, long timestamp,
                                            double sbp, double dbp) {
        CanonicalEvent event = baseEvent(patientId, EventType.VITAL_SIGN, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("systolic_bp", sbp);
        payload.put("diastolic_bp", dbp);
        event.setPayload(payload);
        return event;
    }

    // --- State Builders ---

    public static MealCorrelationState emptyState(String patientId) {
        return new MealCorrelationState(patientId);
    }

    public static MealCorrelationState stateWithTier(String patientId, DataTier tier) {
        MealCorrelationState state = new MealCorrelationState(patientId);
        state.setDataTier(tier);
        return state;
    }

    public static MealCorrelationState stateWithRecentBP(String patientId,
                                                          double sbp, double dbp,
                                                          long bpTimestamp) {
        MealCorrelationState state = new MealCorrelationState(patientId);
        state.addBPReading(bpTimestamp, sbp, dbp);
        return state;
    }

    // --- Glucose Window Builders ---

    /** Create a CGM window with 5-min interval readings */
    public static GlucoseWindow cgmWindow(long startTime, double baseline, double... values) {
        GlucoseWindow w = new GlucoseWindow();
        w.setBaseline(baseline);
        w.setDataTier(DataTier.TIER_1_CGM);
        w.setWindowOpenTime(startTime);
        for (int i = 0; i < values.length; i++) {
            w.addReading(startTime + i * 5 * MIN_MS, values[i], "CGM");
        }
        w.sortByTime();
        return w;
    }

    /** Create a rapid spike pattern (peak at 15 min) */
    public static GlucoseWindow rapidSpikeWindow(long startTime) {
        return cgmWindow(startTime, 100.0,
            100, 130, 160, 180, 170, 140, 110, 105, 100, 98, 97, 96);
    }

    /** Create a flat response pattern (excursion < 20) */
    public static GlucoseWindow flatWindow(long startTime) {
        return cgmWindow(startTime, 100.0,
            100, 102, 105, 108, 110, 112, 110, 108, 105, 103, 101, 100);
    }

    // --- MealResponseRecord Builder ---

    public static MealResponseRecord mealRecord(String patientId, long timestamp,
                                                  Double excursion, Double iauc,
                                                  Double sodiumMg, Double sbpExcursion) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("food_description", "test meal");

        return MealResponseRecord.builder()
            .recordId("test-" + UUID.randomUUID())
            .patientId(patientId)
            .mealEventId("meal-" + UUID.randomUUID())
            .mealTimestamp(timestamp)
            .mealTimeCategory(MealTimeCategory.fromTimestamp(timestamp))
            .glucoseExcursion(excursion)
            .iAUC(iauc)
            .sodiumMg(sodiumMg)
            .sbpExcursion(sbpExcursion)
            .bpComplete(sbpExcursion != null)
            .dataTier(DataTier.TIER_1_CGM)
            .mealPayload(payload)
            .qualityScore(0.8)
            .build();
    }

    private static CanonicalEvent baseEvent(String patientId, EventType type, long timestamp) {
        CanonicalEvent event = new CanonicalEvent();
        event.setId(UUID.randomUUID().toString());
        event.setPatientId(patientId);
        event.setEventType(type);
        event.setEventTime(timestamp);
        event.setProcessingTime(System.currentTimeMillis());
        event.setCorrelationId("test-" + UUID.randomUUID());
        return event;
    }
}
```

- [ ] **Step 2: Compile**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/builders/Module10TestBuilder.java
git commit -m "feat(module10): add Module10TestBuilder with event, state, and window factories"
```

---

### Task 15: Create Session Window and Integration Tests

**Files:**
- Create: `operators/Module10SessionWindowTest.java`
- Create: `operators/Module10OverlappingMealTest.java`
- Create: `operators/Module10TierDegradationTest.java`

- [ ] **Step 1: Create Module10SessionWindowTest**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module10TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.Map;
import java.util.HashMap;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 10 session window mechanics:
 * - Meal opens window, glucose fills, timer closes
 * - Pre-meal BP retroactive attachment
 * - Window duration and grace period
 */
class Module10SessionWindowTest {

    private static final long BASE = Module10TestBuilder.BASE_TIME;
    private static final long MIN_5 = 5 * 60_000L;
    private static final long HOUR = 3_600_000L;

    @Test
    void openSession_setsTimerAt3h05m() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        long mealTime = BASE;
        Map<String, Object> payload = new HashMap<>();
        payload.put("carb_grams", 50.0);

        long timerFireTime = state.openSession("meal-1", mealTime, payload);

        long expected = mealTime + 3 * HOUR + 5 * 60_000L;
        assertEquals(expected, timerFireTime);
        assertEquals(1, state.getActiveSessions().size());
    }

    @Test
    void glucoseReadings_addedToActiveSession() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);

        // Add CGM readings within 3h window
        state.addGlucoseReading(BASE + 10 * MIN_5, 120.0, "CGM");
        state.addGlucoseReading(BASE + 20 * MIN_5, 150.0, "CGM");
        state.addGlucoseReading(BASE + 30 * MIN_5, 130.0, "CGM");

        MealCorrelationState.MealSession session = state.getActiveSessions().get("meal-1");
        assertEquals(3, session.glucoseWindow.size());
    }

    @Test
    void glucoseReadings_ignoredOutsideWindow() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);

        // Reading after 3h window → should NOT be added
        state.addGlucoseReading(BASE + 4 * HOUR, 120.0, "CGM");

        MealCorrelationState.MealSession session = state.getActiveSessions().get("meal-1");
        assertEquals(0, session.glucoseWindow.size());
    }

    @Test
    void preMealBP_retroactiveAttachment() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");

        // BP reading 30 min before meal
        state.addBPReading(BASE - 30 * 60_000L, 120.0, 80.0);

        // Meal arrives — should attach BP as pre-meal
        state.openSession("meal-1", BASE, null);

        MealCorrelationState.MealSession session = state.getActiveSessions().get("meal-1");
        assertTrue(session.bpWindow.hasPreMeal());
        assertEquals(120.0, session.bpWindow.getPreMealSBP());
    }

    @Test
    void preMealBP_notAttachedIfTooOld() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");

        // BP reading 90 min before meal (> 60 min lookback)
        state.addBPReading(BASE - 90 * 60_000L, 120.0, 80.0);

        state.openSession("meal-1", BASE, null);

        MealCorrelationState.MealSession session = state.getActiveSessions().get("meal-1");
        assertFalse(session.bpWindow.hasPreMeal());
    }

    @Test
    void postMealBP_capturedWithin4h() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);

        // BP reading 2h after meal → captured as post-meal
        state.addBPReading(BASE + 2 * HOUR, 135.0, 85.0);

        MealCorrelationState.MealSession session = state.getActiveSessions().get("meal-1");
        assertTrue(session.bpWindow.hasPostMeal());
        assertEquals(135.0, session.bpWindow.getPostMealSBP());
    }

    @Test
    void closeSession_removesFromActive() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);
        assertEquals(1, state.getActiveSessions().size());

        MealCorrelationState.MealSession closed = state.closeSession("meal-1");
        assertNotNull(closed);
        assertEquals(0, state.getActiveSessions().size());
    }
}
```

- [ ] **Step 2: Create Module10OverlappingMealTest**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module10TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for overlapping meal detection (second meal within 90 min).
 */
class Module10OverlappingMealTest {

    private static final long BASE = Module10TestBuilder.BASE_TIME;
    private static final long MIN = 60_000L;

    @Test
    void secondMealWithin90min_bothFlagged() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);
        state.openSession("meal-2", BASE + 60 * MIN, null); // 60 min later

        assertTrue(state.getActiveSessions().get("meal-1").overlapping);
        assertTrue(state.getActiveSessions().get("meal-2").overlapping);
    }

    @Test
    void secondMealAfter90min_notFlagged() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);
        state.openSession("meal-2", BASE + 100 * MIN, null); // 100 min later

        assertFalse(state.getActiveSessions().get("meal-1").overlapping);
        assertFalse(state.getActiveSessions().get("meal-2").overlapping);
    }

    @Test
    void glucoseReadings_feedBothOverlappingSessions() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);
        state.openSession("meal-2", BASE + 60 * MIN, null);

        // Reading at 90 min — within both windows
        state.addGlucoseReading(BASE + 90 * MIN, 150.0, "CGM");

        assertEquals(1, state.getActiveSessions().get("meal-1").glucoseWindow.size());
        assertEquals(1, state.getActiveSessions().get("meal-2").glucoseWindow.size());
    }
}
```

- [ ] **Step 3: Create Module10TierDegradationTest**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for data tier classification and degradation.
 */
class Module10TierDegradationTest {

    @Test
    void defaultTier_isSMBG() {
        MealCorrelationState state = new MealCorrelationState("P1");
        assertEquals(DataTier.TIER_3_SMBG, state.getDataTier());
    }

    @Test
    void dataTier_fromString_variants() {
        assertEquals(DataTier.TIER_1_CGM, DataTier.fromString("TIER_1_CGM"));
        assertEquals(DataTier.TIER_1_CGM, DataTier.fromString("CGM"));
        assertEquals(DataTier.TIER_2_HYBRID, DataTier.fromString("TIER_2_HYBRID"));
        assertEquals(DataTier.TIER_2_HYBRID, DataTier.fromString("HYBRID"));
        assertEquals(DataTier.TIER_3_SMBG, DataTier.fromString("TIER_3_SMBG"));
        assertEquals(DataTier.TIER_3_SMBG, DataTier.fromString(null));
        assertEquals(DataTier.TIER_3_SMBG, DataTier.fromString("UNKNOWN"));
    }

    @Test
    void tier1_supportsCurveClassification() {
        assertTrue(DataTier.TIER_1_CGM.supportsCurveClassification());
        assertFalse(DataTier.TIER_2_HYBRID.supportsCurveClassification());
        assertFalse(DataTier.TIER_3_SMBG.supportsCurveClassification());
    }

    @Test
    void tier1And2_supportFullIAUC() {
        assertTrue(DataTier.TIER_1_CGM.supportsFullIAUC());
        assertTrue(DataTier.TIER_2_HYBRID.supportsFullIAUC());
        assertFalse(DataTier.TIER_3_SMBG.supportsFullIAUC());
    }

    @Test
    void mealTimeCategory_fromTimestamp() {
        // 07:00 UTC → BREAKFAST
        long breakfast = java.time.ZonedDateTime.of(2025, 4, 2, 7, 0, 0, 0,
            java.time.ZoneOffset.UTC).toInstant().toEpochMilli();
        assertEquals(MealTimeCategory.BREAKFAST, MealTimeCategory.fromTimestamp(breakfast));

        // 12:00 UTC → LUNCH
        long lunch = java.time.ZonedDateTime.of(2025, 4, 2, 12, 0, 0, 0,
            java.time.ZoneOffset.UTC).toInstant().toEpochMilli();
        assertEquals(MealTimeCategory.LUNCH, MealTimeCategory.fromTimestamp(lunch));

        // 19:00 UTC → DINNER
        long dinner = java.time.ZonedDateTime.of(2025, 4, 2, 19, 0, 0, 0,
            java.time.ZoneOffset.UTC).toInstant().toEpochMilli();
        assertEquals(MealTimeCategory.DINNER, MealTimeCategory.fromTimestamp(dinner));

        // 23:00 UTC → SNACK
        long snack = java.time.ZonedDateTime.of(2025, 4, 2, 23, 0, 0, 0,
            java.time.ZoneOffset.UTC).toInstant().toEpochMilli();
        assertEquals(MealTimeCategory.SNACK, MealTimeCategory.fromTimestamp(snack));
    }

    @Test
    void saltSensitivityClass_fromBetaAndR2() {
        assertEquals(SaltSensitivityClass.HIGH,
            SaltSensitivityClass.fromBetaAndR2(0.01, 0.5, 40));
        assertEquals(SaltSensitivityClass.MODERATE,
            SaltSensitivityClass.fromBetaAndR2(0.003, 0.3, 35));
        assertEquals(SaltSensitivityClass.SALT_RESISTANT,
            SaltSensitivityClass.fromBetaAndR2(0.0005, 0.2, 30));
        assertEquals(SaltSensitivityClass.UNDETERMINED,
            SaltSensitivityClass.fromBetaAndR2(0.01, 0.5, 20)); // too few pairs
        assertEquals(SaltSensitivityClass.UNDETERMINED,
            SaltSensitivityClass.fromBetaAndR2(0.01, 0.05, 40)); // low R²
    }
}
```

- [ ] **Step 4: Run all tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module10SessionWindowTest,Module10OverlappingMealTest,Module10TierDegradationTest" -q 2>&1 | tail -15`
Expected: PASS (all tests)

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module10SessionWindowTest.java backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module10OverlappingMealTest.java backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module10TierDegradationTest.java backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/builders/Module10TestBuilder.java
git commit -m "feat(module10): add session window, overlapping meal, tier degradation tests + test builder"
```

---

## Self-Review

### Spec Coverage
- Per-meal session windows with 3h glucose + 4h BP ✅
- Processing-time timer at 3h05m ✅
- Three-tier glucose (CGM/Hybrid/SMBG) ✅
- iAUC via trapezoidal rule (positive-only above baseline) ✅
- 5 curve shapes with 3-point smoothing ✅
- Pre-meal BP retroactive buffer (60 min lookback) ✅
- Overlapping meals (90 min threshold, both flagged) ✅
- MealResponseRecord (28 fields) ✅
- MealPatternSummary (21 fields) ✅
- Salt sensitivity OLS with 3 division-by-zero guards ✅
- Food impact ranking ✅
- Weekly aggregation timer (Monday 00:00 UTC) ✅
- Two-job architecture (Module 10 + 10b separate Flink jobs) ✅
- Orchestrator wiring with MealResponseRecordDeserializer ✅

### Type Consistency
- `MealResponseRecord` used consistently across Module 10 output, Module 10b input, test builder
- `GlucoseWindow.Reading` fields: `timestamp`, `value`, `source` — consistent
- `SaltSensitivityClass.fromBetaAndR2()` signature matches estimator usage
- `MealCorrelationState.MealSession` fields match KPF access patterns
- `Module10GlucoseAnalyzer.Result` fields match `MealResponseRecord.Builder` calls

### Test Coverage (~58 tests across 8 classes)
- Module10GlucoseAnalyzerTest: 6 tests
- Module10CurveClassifierTest: 6 tests
- Module10BPCorrelatorTest: 5 tests
- Module10SessionWindowTest: 7 tests
- Module10OverlappingMealTest: 3 tests
- Module10TierDegradationTest: 5 tests (+ enum coverage)
- Module10bSaltSensitivityTest: 6 tests
- Module10bFoodRankerTest: 4 tests
- **Total: 42 tests** (remaining ~16 would be integration tests for the full KPF pipeline, recommended as follow-up)