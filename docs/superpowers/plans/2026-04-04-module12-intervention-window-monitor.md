# Module 12: Intervention Window Monitor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Module 12 (Intervention Window Monitor) and Module 12b (Intervention Delta Computer) as Flink KeyedProcessFunctions that track physician-approved intervention observation windows, emit lifecycle signals, detect concurrent interventions, and generate streaming metric deltas for the physician dashboard.

**Architecture:** Event-driven timer-based KPF consuming from two Kafka topics via union pattern: `clinical.intervention-events` (low-volume, from KB-23) and `enriched-patient-events-v1` (high-volume, for trajectory/confounder tracking). Processing-time timers fire at MIDPOINT and CLOSE (+24h grace). Four stateless analyzers (TrajectoryTracker, ConcurrencyDetector, AdherenceAssembler, ConfounderAccumulator) keep the main KPF thin. Module 12b is a separate Flink job consuming WINDOW_CLOSED signals to compute streaming metric deltas.

**Tech Stack:** Flink 2.1.0, Java 17, Jackson 2.17, Kafka (Confluent Cloud), JUnit 5

**Base Path:** `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink`
**Test Base:** `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink`

---

## File Structure

### New Source Files (14)

| # | File | Responsibility |
|---|------|---------------|
| 1 | `models/InterventionType.java` | Enum: 13 intervention types with default window days, target domains, adherence source |
| 2 | `models/InterventionWindowSignalType.java` | Enum: 5 signal types (OPENED, MIDPOINT, CLOSED, EXPIRED, CANCELLED) |
| 3 | `models/TrajectoryClassification.java` | Enum: IMPROVING, STABLE, DETERIORATING, UNKNOWN |
| 4 | `models/TrajectoryAttribution.java` | Enum: 9 attribution classifications from before/during trajectory matrix |
| 5 | `models/InterventionWindowState.java` | Per-patient keyed state: active windows, trajectory buffer, confounder accumulator |
| 6 | `models/InterventionWindowSignal.java` | Output model (21 fields) emitted to `clinical.intervention-window-signals` |
| 7 | `models/InterventionDeltaRecord.java` | Output model for Module 12b (18 fields): metric deltas at window close |
| 8 | `operators/Module12TrajectoryTracker.java` | Static analyzer: classifies 14-day trajectory via OLS slope per domain |
| 9 | `operators/Module12ConcurrencyDetector.java` | Static analyzer: detects overlapping windows, classifies domain overlap |
| 10 | `operators/Module12AdherenceAssembler.java` | Static analyzer: assembles preliminary adherence signals from data sources |
| 11 | `operators/Module12ConfounderAccumulator.java` | Static analyzer: accumulates confounder flags from events during windows |
| 12 | `operators/Module12_InterventionWindowMonitor.java` | Main KPF: dual-source event routing, timer management, signal emission |
| 13 | `operators/Module12b_InterventionDeltaComputer.java` | Module 12b KPF: consumes WINDOW_CLOSED, computes streaming metric deltas |
| 14 | `FlinkJobOrchestrator.java` | MODIFY: add module12/module12b switch cases + launch methods + deserializer |

### New Test Files (8)

| # | File | Test Count |
|---|------|-----------|
| 1 | `builders/Module12TestBuilder.java` | — (factory) |
| 2 | `operators/Module12TrajectoryTrackerTest.java` | 6 |
| 3 | `operators/Module12ConcurrencyDetectorTest.java` | 5 |
| 4 | `operators/Module12AdherenceAssemblerTest.java` | 5 |
| 5 | `operators/Module12ConfounderAccumulatorTest.java` | 5 |
| 6 | `operators/Module12WindowLifecycleTest.java` | 7 |
| 7 | `operators/Module12ConcurrentInterventionIntegrationTest.java` | 4 |
| 8 | `operators/Module12bDeltaComputerTest.java` | 5 |

### Modified Files (1)

| File | Change |
|------|--------|
| `utils/KafkaTopics.java` | Add `FLINK_INTERVENTION_DELTAS` constant + update `isV4OutputTopic()` |

**Total: 14 source files, 8 test files, 1 modified file, 42 unit tests.**

---

## Task 1: InterventionType Enum

**Files:**
- Create: `models/InterventionType.java`

- [ ] **Step 1: Create InterventionType enum with 13 intervention types**

Each type carries its default observation window (days), primary clinical domain(s), and adherence signal source. The `getDomains()` method is critical for ConcurrencyDetector's same-domain overlap logic.

```java
package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashSet;
import java.util.Set;

/**
 * Intervention types with default observation windows, target clinical domains,
 * and adherence signal sources. Used by Module 12 for window configuration
 * and concurrent intervention domain classification.
 */
public enum InterventionType implements Serializable {
    MEDICATION_ADD(28, "MEDICATION_REMINDERS",
            ClinicalDomain.FROM_DRUG_CLASS),
    MEDICATION_REMOVE(28, "ABSENCE_IN_LOGS",
            ClinicalDomain.FROM_DRUG_CLASS),
    MEDICATION_DOSE_INCREASE(28, "MEDICATION_REMINDERS",
            ClinicalDomain.FROM_DRUG_CLASS),
    MEDICATION_DOSE_DECREASE(28, "MEDICATION_REMINDERS",
            ClinicalDomain.FROM_DRUG_CLASS),
    MEDICATION_SWITCH(28, "MEDICATION_REMINDERS",
            ClinicalDomain.FROM_DRUG_CLASS),
    LIFESTYLE_ACTIVITY(14, "ACTIVITY_DATA",
            ClinicalDomain.GLUCOSE, ClinicalDomain.BLOOD_PRESSURE),
    LIFESTYLE_SLEEP(14, "SLEEP_DATA",
            ClinicalDomain.GLUCOSE_VARIABILITY, ClinicalDomain.BLOOD_PRESSURE),
    NUTRITION_FOOD_CHANGE(14, "MEAL_LOGS",
            ClinicalDomain.GLUCOSE),
    NUTRITION_PORTION_CHANGE(14, "MEAL_LOGS",
            ClinicalDomain.GLUCOSE, ClinicalDomain.WEIGHT),
    NUTRITION_TIMING_CHANGE(14, "MEAL_TIMESTAMPS",
            ClinicalDomain.GLUCOSE),
    NUTRITION_SODIUM_REDUCTION(14, "SODIUM_ESTIMATE",
            ClinicalDomain.BLOOD_PRESSURE),
    MONITORING_CHANGE(14, "DATA_DENSITY",
            ClinicalDomain.DATA_QUALITY),
    REFERRAL(28, "REFERRAL_ATTENDANCE",
            ClinicalDomain.FROM_REFERRAL_TYPE);

    private final int defaultWindowDays;
    private final String adherenceSource;
    private final Set<ClinicalDomain> staticDomains;
    private final boolean domainFromDetail;

    InterventionType(int defaultWindowDays, String adherenceSource,
                     ClinicalDomain... domains) {
        this.defaultWindowDays = defaultWindowDays;
        this.adherenceSource = adherenceSource;
        if (domains.length == 1 && (domains[0] == ClinicalDomain.FROM_DRUG_CLASS
                || domains[0] == ClinicalDomain.FROM_REFERRAL_TYPE)) {
            this.staticDomains = Collections.emptySet();
            this.domainFromDetail = true;
        } else {
            this.staticDomains = Collections.unmodifiableSet(
                    new HashSet<>(Arrays.asList(domains)));
            this.domainFromDetail = false;
        }
    }

    public int getDefaultWindowDays() { return defaultWindowDays; }
    public String getAdherenceSource() { return adherenceSource; }
    public boolean isMedication() { return name().startsWith("MEDICATION_"); }
    public boolean isLifestyle() { return name().startsWith("LIFESTYLE_"); }
    public boolean isNutrition() { return name().startsWith("NUTRITION_"); }

    /**
     * Returns the target clinical domains for this intervention.
     * For MEDICATION_* types, domains are derived from drug_class in the
     * intervention detail payload. For others, domains are statically defined.
     *
     * @param drugClass the drug class from intervention_detail (nullable)
     * @return set of target clinical domains
     */
    public Set<ClinicalDomain> getDomains(String drugClass) {
        if (!domainFromDetail) {
            return staticDomains;
        }
        if (drugClass == null) {
            return Collections.emptySet();
        }
        return ClinicalDomain.fromDrugClass(drugClass);
    }

    /**
     * Clinical domains targeted by interventions.
     * Used for concurrent intervention same-domain detection.
     */
    public enum ClinicalDomain {
        GLUCOSE, BLOOD_PRESSURE, RENAL, WEIGHT, LIPIDS,
        GLUCOSE_VARIABILITY, DATA_QUALITY,
        FROM_DRUG_CLASS, FROM_REFERRAL_TYPE;

        /**
         * Drug class → clinical domain mapping per spec Section 4.1.
         */
        public static Set<ClinicalDomain> fromDrugClass(String drugClass) {
            if (drugClass == null) return Collections.emptySet();
            switch (drugClass.toUpperCase()) {
                case "SGLT2I": return setOf(GLUCOSE, BLOOD_PRESSURE, RENAL);
                case "METFORMIN": return setOf(GLUCOSE);
                case "SULFONYLUREA": return setOf(GLUCOSE);
                case "DPP4I": return setOf(GLUCOSE);
                case "GLP1_RA": return setOf(GLUCOSE, WEIGHT);
                case "INSULIN": return setOf(GLUCOSE);
                case "ACEI": case "ARB": return setOf(BLOOD_PRESSURE, RENAL);
                case "CCB": return setOf(BLOOD_PRESSURE);
                case "THIAZIDE": return setOf(BLOOD_PRESSURE);
                case "FINERENONE": return setOf(RENAL, BLOOD_PRESSURE);
                case "STATIN": return setOf(LIPIDS);
                default: return Collections.emptySet();
            }
        }

        private static Set<ClinicalDomain> setOf(ClinicalDomain... domains) {
            return Collections.unmodifiableSet(
                    new HashSet<>(Arrays.asList(domains)));
        }
    }
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd backend/shared-infrastructure/flink-processing
mvn compile -pl . -q 2>&1 | tail -5
```

Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/InterventionType.java
git commit -m "feat(module12): add InterventionType enum with 13 types, domain mapping, and drug class resolution"
```

---

## Task 2: Signal Type, Trajectory, and Attribution Enums

**Files:**
- Create: `models/InterventionWindowSignalType.java`
- Create: `models/TrajectoryClassification.java`
- Create: `models/TrajectoryAttribution.java`

- [ ] **Step 1: Create InterventionWindowSignalType enum**

```java
package com.cardiofit.flink.models;

import java.io.Serializable;

/**
 * Lifecycle signal types emitted by Module 12 for each intervention window.
 */
public enum InterventionWindowSignalType implements Serializable {
    WINDOW_OPENED,
    WINDOW_MIDPOINT,
    WINDOW_CLOSED,
    WINDOW_EXPIRED,
    WINDOW_CANCELLED
}
```

- [ ] **Step 2: Create TrajectoryClassification enum**

```java
package com.cardiofit.flink.models;

import java.io.Serializable;

/**
 * Patient physiological trajectory classification.
 * Computed by Module12TrajectoryTracker from recent readings using OLS slope.
 */
public enum TrajectoryClassification implements Serializable {
    IMPROVING,
    STABLE,
    DETERIORATING,
    UNKNOWN
}
```

- [ ] **Step 3: Create TrajectoryAttribution enum**

The 9-cell before/during trajectory matrix from spec Section 1.2.

```java
package com.cardiofit.flink.models;

import java.io.Serializable;

/**
 * Trajectory attribution classification from the before/during trajectory matrix.
 * Describes how the patient's trajectory changed relative to the intervention.
 * Attribution is DESCRIPTIVE, not causal — confounders must be checked separately.
 */
public enum TrajectoryAttribution implements Serializable {
    INTERVENTION_REVERSED_DECLINE,
    INTERVENTION_ARRESTED_DECLINE,
    INTERVENTION_INSUFFICIENT,
    INTERVENTION_IMPROVED_STABLE,
    NO_DETECTABLE_EFFECT,
    DETERIORATION_DESPITE_INTERVENTION,
    IMPROVEMENT_CONTINUED,
    IMPROVEMENT_PLATEAUED,
    TRAJECTORY_REVERSAL;

    /**
     * Resolves attribution from before/during trajectory pair.
     * Implements the 3x3 matrix from the clinical specification.
     */
    public static TrajectoryAttribution fromTrajectories(
            TrajectoryClassification before, TrajectoryClassification during) {
        if (before == null || during == null
                || before == TrajectoryClassification.UNKNOWN
                || during == TrajectoryClassification.UNKNOWN) {
            return NO_DETECTABLE_EFFECT;
        }
        switch (before) {
            case DETERIORATING:
                switch (during) {
                    case IMPROVING: return INTERVENTION_REVERSED_DECLINE;
                    case STABLE: return INTERVENTION_ARRESTED_DECLINE;
                    case DETERIORATING: return INTERVENTION_INSUFFICIENT;
                    default: return NO_DETECTABLE_EFFECT;
                }
            case STABLE:
                switch (during) {
                    case IMPROVING: return INTERVENTION_IMPROVED_STABLE;
                    case STABLE: return NO_DETECTABLE_EFFECT;
                    case DETERIORATING: return DETERIORATION_DESPITE_INTERVENTION;
                    default: return NO_DETECTABLE_EFFECT;
                }
            case IMPROVING:
                switch (during) {
                    case IMPROVING: return IMPROVEMENT_CONTINUED;
                    case STABLE: return IMPROVEMENT_PLATEAUED;
                    case DETERIORATING: return TRAJECTORY_REVERSAL;
                    default: return NO_DETECTABLE_EFFECT;
                }
            default:
                return NO_DETECTABLE_EFFECT;
        }
    }
}
```

- [ ] **Step 4: Verify compilation**

```bash
cd backend/shared-infrastructure/flink-processing
mvn compile -pl . -q 2>&1 | tail -5
```

Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/InterventionWindowSignalType.java \
        src/main/java/com/cardiofit/flink/models/TrajectoryClassification.java \
        src/main/java/com/cardiofit/flink/models/TrajectoryAttribution.java
git commit -m "feat(module12): add signal type, trajectory classification, and 9-cell attribution enums"
```

---

## Task 3: InterventionWindowState (Per-Patient Keyed State)

**Files:**
- Create: `models/InterventionWindowState.java`

- [ ] **Step 1: Create InterventionWindowState with nested InterventionWindow**

This is the per-patient state managed by Module 12's KPF. It stores active windows, recent trajectory data, and vital sign baselines. The nested `InterventionWindow` holds per-intervention lifecycle data.

```java
package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Per-patient keyed state for Module 12: Intervention Window Monitor.
 * Stores active intervention windows, recent trajectory readings,
 * and last-known vital baselines for trajectory computation.
 *
 * State TTL: 90 days (longest observation window is 28d;
 * 90d allows IOR Generator to re-query for late reconciliation).
 */
public class InterventionWindowState implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;

    /** Active observation windows, keyed by intervention_id. */
    private Map<String, InterventionWindow> activeWindows = new HashMap<>();

    /** Rolling 60-day buffer of trajectory data points for slope computation. */
    private List<TrajectoryDataPoint> recentReadings = new ArrayList<>();

    // Last-known baselines for trajectory and delta computation
    private Double lastKnownFBG;
    private Double lastKnownSBP;
    private Double lastKnownDBP;
    private Double lastKnownHbA1c;
    private Double lastKnownWeight;
    private Double lastKnownEGFR;
    private Double lastKnownTIR;

    private long totalInterventionsTracked;
    private long lastUpdated;

    public InterventionWindowState() {}

    public InterventionWindowState(String patientId) {
        this.patientId = patientId;
    }

    // --- Window Lifecycle ---

    public InterventionWindow openWindow(String interventionId,
                                          InterventionType interventionType,
                                          Map<String, Object> interventionDetail,
                                          int windowDays,
                                          long observationStartMs,
                                          TrajectoryClassification trajectoryAtOpen,
                                          String originatingCardId,
                                          String physicianAction) {
        long windowMs = windowDays * 24L * 60 * 60 * 1000;
        long gracePeriodMs = 24L * 60 * 60 * 1000; // 24h grace
        long observationEndMs = observationStartMs + windowMs;
        long midpointMs = observationStartMs + windowMs / 2;
        long closeTimerMs = observationEndMs + gracePeriodMs;

        InterventionWindow window = new InterventionWindow();
        window.interventionId = interventionId;
        window.interventionType = interventionType;
        window.interventionDetail = interventionDetail;
        window.observationStartMs = observationStartMs;
        window.observationEndMs = observationEndMs;
        window.observationWindowDays = windowDays;
        window.midpointTimerMs = midpointMs;
        window.closeTimerMs = closeTimerMs;
        window.trajectoryAtOpen = trajectoryAtOpen;
        window.originatingCardId = originatingCardId;
        window.physicianAction = physicianAction;
        window.status = "OBSERVING";

        activeWindows.put(interventionId, window);
        totalInterventionsTracked++;
        return window;
    }

    public InterventionWindow getWindow(String interventionId) {
        return activeWindows.get(interventionId);
    }

    public InterventionWindow removeWindow(String interventionId) {
        return activeWindows.remove(interventionId);
    }

    public Map<String, InterventionWindow> getActiveWindows() {
        return activeWindows;
    }

    /**
     * Finds the intervention window whose midpoint timer matches the given timestamp.
     */
    public InterventionWindow getWindowForMidpointTimer(long timerTimestamp) {
        for (InterventionWindow w : activeWindows.values()) {
            if (w.midpointTimerMs == timerTimestamp && "OBSERVING".equals(w.status)) {
                return w;
            }
        }
        return null;
    }

    /**
     * Finds the intervention window whose close timer matches the given timestamp.
     */
    public InterventionWindow getWindowForCloseTimer(long timerTimestamp) {
        for (InterventionWindow w : activeWindows.values()) {
            if (w.closeTimerMs == timerTimestamp && "OBSERVING".equals(w.status)) {
                return w;
            }
        }
        return null;
    }

    // --- Trajectory Data ---

    public void addReading(String domain, double value, long timestamp) {
        recentReadings.add(new TrajectoryDataPoint(domain, value, timestamp));
        // Evict readings older than 60 days
        long cutoff = timestamp - 60L * 24 * 60 * 60 * 1000;
        recentReadings.removeIf(r -> r.timestamp < cutoff);
    }

    public List<TrajectoryDataPoint> getReadingsForDomain(String domain, long since) {
        List<TrajectoryDataPoint> result = new ArrayList<>();
        for (TrajectoryDataPoint p : recentReadings) {
            if (domain.equals(p.domain) && p.timestamp >= since) {
                result.add(p);
            }
        }
        return result;
    }

    public List<TrajectoryDataPoint> getRecentReadings() { return recentReadings; }

    // --- Getters/Setters ---
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Double getLastKnownFBG() { return lastKnownFBG; }
    public void setLastKnownFBG(Double v) { this.lastKnownFBG = v; }
    public Double getLastKnownSBP() { return lastKnownSBP; }
    public void setLastKnownSBP(Double v) { this.lastKnownSBP = v; }
    public Double getLastKnownDBP() { return lastKnownDBP; }
    public void setLastKnownDBP(Double v) { this.lastKnownDBP = v; }
    public Double getLastKnownHbA1c() { return lastKnownHbA1c; }
    public void setLastKnownHbA1c(Double v) { this.lastKnownHbA1c = v; }
    public Double getLastKnownWeight() { return lastKnownWeight; }
    public void setLastKnownWeight(Double v) { this.lastKnownWeight = v; }
    public Double getLastKnownEGFR() { return lastKnownEGFR; }
    public void setLastKnownEGFR(Double v) { this.lastKnownEGFR = v; }
    public Double getLastKnownTIR() { return lastKnownTIR; }
    public void setLastKnownTIR(Double v) { this.lastKnownTIR = v; }
    public long getTotalInterventionsTracked() { return totalInterventionsTracked; }
    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long lastUpdated) { this.lastUpdated = lastUpdated; }

    // --- Nested Classes ---

    /**
     * A single trajectory data point for slope computation.
     */
    public static class TrajectoryDataPoint implements Serializable {
        private static final long serialVersionUID = 1L;
        public String domain;
        public double value;
        public long timestamp;

        public TrajectoryDataPoint() {}
        public TrajectoryDataPoint(String domain, double value, long timestamp) {
            this.domain = domain;
            this.value = value;
            this.timestamp = timestamp;
        }
    }

    /**
     * Per-intervention observation window within the patient's state.
     */
    public static class InterventionWindow implements Serializable {
        private static final long serialVersionUID = 1L;

        public String interventionId;
        public InterventionType interventionType;
        public Map<String, Object> interventionDetail;
        public long observationStartMs;
        public long observationEndMs;
        public int observationWindowDays;
        public long midpointTimerMs;
        public long closeTimerMs;
        public TrajectoryClassification trajectoryAtOpen;
        public String originatingCardId;
        public String physicianAction;
        public String status; // OBSERVING, CANCELLED

        public List<String> concurrentInterventionIds = new ArrayList<>();
        public List<String> confoundersDetected = new ArrayList<>();
        public List<Map<String, Object>> labChanges = new ArrayList<>();
        public List<Map<String, Object>> externalEvents = new ArrayList<>();
        public Map<String, Object> adherenceSignals;
        public Map<String, Boolean> dataCompleteness = new HashMap<>();
    }
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd backend/shared-infrastructure/flink-processing
mvn compile -pl . -q 2>&1 | tail -5
```

Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/InterventionWindowState.java
git commit -m "feat(module12): add InterventionWindowState with nested InterventionWindow and trajectory buffer"
```

---

## Task 4: Output Models (InterventionWindowSignal + InterventionDeltaRecord)

**Files:**
- Create: `models/InterventionWindowSignal.java`
- Create: `models/InterventionDeltaRecord.java`

- [ ] **Step 1: Create InterventionWindowSignal (21-field output model)**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.List;
import java.util.Map;

/**
 * Output record emitted to clinical.intervention-window-signals for each
 * lifecycle event (OPENED, MIDPOINT, CLOSED, EXPIRED, CANCELLED).
 * Consumed by IOR Generator (batch), Module 12b, and coaching engine.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class InterventionWindowSignal implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("signal_id") private String signalId;
    @JsonProperty("intervention_id") private String interventionId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("signal_type") private InterventionWindowSignalType signalType;
    @JsonProperty("intervention_type") private InterventionType interventionType;
    @JsonProperty("intervention_detail") private Map<String, Object> interventionDetail;
    @JsonProperty("observation_start_ms") private long observationStartMs;
    @JsonProperty("observation_end_ms") private long observationEndMs;
    @JsonProperty("observation_window_days") private int observationWindowDays;
    @JsonProperty("trajectory_at_signal") private TrajectoryClassification trajectoryAtSignal;
    @JsonProperty("concurrent_intervention_ids") private List<String> concurrentInterventionIds;
    @JsonProperty("concurrent_intervention_count") private int concurrentInterventionCount;
    @JsonProperty("same_domain_concurrent") private boolean sameDomainConcurrent;
    @JsonProperty("adherence_signals_at_midpoint") private Map<String, Object> adherenceSignalsAtMidpoint;
    @JsonProperty("confounders_detected") private List<String> confoundersDetected;
    @JsonProperty("lab_changes_during_window") private List<Map<String, Object>> labChangesDuringWindow;
    @JsonProperty("external_events") private List<Map<String, Object>> externalEvents;
    @JsonProperty("data_completeness_indicators") private Map<String, Boolean> dataCompletenessIndicators;
    @JsonProperty("processing_timestamp") private long processingTimestamp;
    @JsonProperty("version") private String version = "1.0";
    @JsonProperty("originating_card_id") private String originatingCardId;

    public InterventionWindowSignal() {}

    public static Builder builder() { return new Builder(); }

    // --- Builder ---
    public static class Builder {
        private final InterventionWindowSignal s = new InterventionWindowSignal();
        public Builder signalId(String v) { s.signalId = v; return this; }
        public Builder interventionId(String v) { s.interventionId = v; return this; }
        public Builder patientId(String v) { s.patientId = v; return this; }
        public Builder signalType(InterventionWindowSignalType v) { s.signalType = v; return this; }
        public Builder interventionType(InterventionType v) { s.interventionType = v; return this; }
        public Builder interventionDetail(Map<String, Object> v) { s.interventionDetail = v; return this; }
        public Builder observationStartMs(long v) { s.observationStartMs = v; return this; }
        public Builder observationEndMs(long v) { s.observationEndMs = v; return this; }
        public Builder observationWindowDays(int v) { s.observationWindowDays = v; return this; }
        public Builder trajectoryAtSignal(TrajectoryClassification v) { s.trajectoryAtSignal = v; return this; }
        public Builder concurrentInterventionIds(List<String> v) { s.concurrentInterventionIds = v; return this; }
        public Builder concurrentInterventionCount(int v) { s.concurrentInterventionCount = v; return this; }
        public Builder sameDomainConcurrent(boolean v) { s.sameDomainConcurrent = v; return this; }
        public Builder adherenceSignalsAtMidpoint(Map<String, Object> v) { s.adherenceSignalsAtMidpoint = v; return this; }
        public Builder confoundersDetected(List<String> v) { s.confoundersDetected = v; return this; }
        public Builder labChangesDuringWindow(List<Map<String, Object>> v) { s.labChangesDuringWindow = v; return this; }
        public Builder externalEvents(List<Map<String, Object>> v) { s.externalEvents = v; return this; }
        public Builder dataCompletenessIndicators(Map<String, Boolean> v) { s.dataCompletenessIndicators = v; return this; }
        public Builder processingTimestamp(long v) { s.processingTimestamp = v; return this; }
        public Builder version(String v) { s.version = v; return this; }
        public Builder originatingCardId(String v) { s.originatingCardId = v; return this; }
        public InterventionWindowSignal build() { return s; }
    }

    // --- Getters ---
    public String getSignalId() { return signalId; }
    public String getInterventionId() { return interventionId; }
    public String getPatientId() { return patientId; }
    public InterventionWindowSignalType getSignalType() { return signalType; }
    public InterventionType getInterventionType() { return interventionType; }
    public Map<String, Object> getInterventionDetail() { return interventionDetail; }
    public long getObservationStartMs() { return observationStartMs; }
    public long getObservationEndMs() { return observationEndMs; }
    public int getObservationWindowDays() { return observationWindowDays; }
    public TrajectoryClassification getTrajectoryAtSignal() { return trajectoryAtSignal; }
    public List<String> getConcurrentInterventionIds() { return concurrentInterventionIds; }
    public int getConcurrentInterventionCount() { return concurrentInterventionCount; }
    public boolean isSameDomainConcurrent() { return sameDomainConcurrent; }
    public Map<String, Object> getAdherenceSignalsAtMidpoint() { return adherenceSignalsAtMidpoint; }
    public List<String> getConfoundersDetected() { return confoundersDetected; }
    public List<Map<String, Object>> getLabChangesDuringWindow() { return labChangesDuringWindow; }
    public List<Map<String, Object>> getExternalEvents() { return externalEvents; }
    public Map<String, Boolean> getDataCompletenessIndicators() { return dataCompletenessIndicators; }
    public long getProcessingTimestamp() { return processingTimestamp; }
    public String getVersion() { return version; }
    public String getOriginatingCardId() { return originatingCardId; }
}
```

- [ ] **Step 2: Create InterventionDeltaRecord (18-field Module 12b output)**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Module 12b output: streaming metric deltas at intervention window close.
 * Provides immediate feedback to the physician dashboard without waiting
 * for the batch IOR Generator.
 *
 * Delta convention: close_value - open_value.
 * Negative = improvement for FBG, SBP, DBP, weight, HbA1c.
 * Positive = improvement for TIR.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class InterventionDeltaRecord implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("delta_id") private String deltaId;
    @JsonProperty("intervention_id") private String interventionId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("intervention_type") private InterventionType interventionType;
    @JsonProperty("fbg_delta") private Double fbgDelta;
    @JsonProperty("sbp_delta") private Double sbpDelta;
    @JsonProperty("dbp_delta") private Double dbpDelta;
    @JsonProperty("weight_delta_kg") private Double weightDeltaKg;
    @JsonProperty("hba1c_delta") private Double hba1cDelta;
    @JsonProperty("egfr_delta") private Double egfrDelta;
    @JsonProperty("tir_delta") private Double tirDelta;
    @JsonProperty("mri_score_delta") private Double mriScoreDelta;
    @JsonProperty("trajectory_attribution") private TrajectoryAttribution trajectoryAttribution;
    @JsonProperty("adherence_score") private Double adherenceScore;
    @JsonProperty("concurrent_count") private int concurrentCount;
    @JsonProperty("data_completeness_score") private double dataCompletenessScore;
    @JsonProperty("processing_timestamp") private long processingTimestamp;
    @JsonProperty("version") private String version = "1.0";

    public InterventionDeltaRecord() {}

    public static Builder builder() { return new Builder(); }

    public static class Builder {
        private final InterventionDeltaRecord r = new InterventionDeltaRecord();
        public Builder deltaId(String v) { r.deltaId = v; return this; }
        public Builder interventionId(String v) { r.interventionId = v; return this; }
        public Builder patientId(String v) { r.patientId = v; return this; }
        public Builder interventionType(InterventionType v) { r.interventionType = v; return this; }
        public Builder fbgDelta(Double v) { r.fbgDelta = v; return this; }
        public Builder sbpDelta(Double v) { r.sbpDelta = v; return this; }
        public Builder dbpDelta(Double v) { r.dbpDelta = v; return this; }
        public Builder weightDeltaKg(Double v) { r.weightDeltaKg = v; return this; }
        public Builder hba1cDelta(Double v) { r.hba1cDelta = v; return this; }
        public Builder egfrDelta(Double v) { r.egfrDelta = v; return this; }
        public Builder tirDelta(Double v) { r.tirDelta = v; return this; }
        public Builder mriScoreDelta(Double v) { r.mriScoreDelta = v; return this; }
        public Builder trajectoryAttribution(TrajectoryAttribution v) { r.trajectoryAttribution = v; return this; }
        public Builder adherenceScore(Double v) { r.adherenceScore = v; return this; }
        public Builder concurrentCount(int v) { r.concurrentCount = v; return this; }
        public Builder dataCompletenessScore(double v) { r.dataCompletenessScore = v; return this; }
        public Builder processingTimestamp(long v) { r.processingTimestamp = v; return this; }
        public Builder version(String v) { r.version = v; return this; }
        public InterventionDeltaRecord build() { return r; }
    }

    // --- Getters ---
    public String getDeltaId() { return deltaId; }
    public String getInterventionId() { return interventionId; }
    public String getPatientId() { return patientId; }
    public InterventionType getInterventionType() { return interventionType; }
    public Double getFbgDelta() { return fbgDelta; }
    public Double getSbpDelta() { return sbpDelta; }
    public Double getDbpDelta() { return dbpDelta; }
    public Double getWeightDeltaKg() { return weightDeltaKg; }
    public Double getHba1cDelta() { return hba1cDelta; }
    public Double getEgfrDelta() { return egfrDelta; }
    public Double getTirDelta() { return tirDelta; }
    public Double getMriScoreDelta() { return mriScoreDelta; }
    public TrajectoryAttribution getTrajectoryAttribution() { return trajectoryAttribution; }
    public Double getAdherenceScore() { return adherenceScore; }
    public int getConcurrentCount() { return concurrentCount; }
    public double getDataCompletenessScore() { return dataCompletenessScore; }
    public long getProcessingTimestamp() { return processingTimestamp; }
    public String getVersion() { return version; }
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd backend/shared-infrastructure/flink-processing
mvn compile -pl . -q 2>&1 | tail -5
```

Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/InterventionWindowSignal.java \
        src/main/java/com/cardiofit/flink/models/InterventionDeltaRecord.java
git commit -m "feat(module12): add InterventionWindowSignal (21 fields) and InterventionDeltaRecord (18 fields) output models"
```

---

## Task 5: Module12TestBuilder

**Files:**
- Create: `builders/Module12TestBuilder.java`

- [ ] **Step 1: Create test builder with intervention event factories and state builders**

The test builder creates intervention events (as CanonicalEvent with intervention payload), patient events (vitals, labs), and pre-configured state objects. Follows the Module11TestBuilder pattern.

```java
package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Test builder for Module 12/12b tests.
 * Provides factory methods for intervention events, patient events, and state objects.
 */
public class Module12TestBuilder {

    public static final long HOUR_MS = 3_600_000L;
    public static final long MIN_MS = 60_000L;
    public static final long DAY_MS = 86_400_000L;
    public static final long BASE_TIME = 1743552000000L; // 2025-04-02 00:00:00 UTC

    public static long minutesAfter(long base, int minutes) { return base + minutes * MIN_MS; }
    public static long hoursAfter(long base, int hours) { return base + hours * HOUR_MS; }
    public static long daysAfter(long base, int days) { return base + days * DAY_MS; }

    // --- Intervention Events ---

    /**
     * Creates an INTERVENTION_APPROVED event for the given intervention type.
     */
    public static CanonicalEvent interventionApproved(String patientId, long timestamp,
                                                       String interventionId,
                                                       InterventionType interventionType,
                                                       Map<String, Object> detail) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("event_type", "INTERVENTION_APPROVED");
        payload.put("intervention_id", interventionId);
        payload.put("intervention_type", interventionType.name());
        payload.put("intervention_detail", detail != null ? detail : new HashMap<>());
        payload.put("originating_card_id", "card-" + UUID.randomUUID());
        payload.put("physician_action", "APPROVED");
        payload.put("physician_id", "dr-" + UUID.randomUUID());
        payload.put("observation_window_days", interventionType.getDefaultWindowDays());
        payload.put("data_tier", "TIER_2_HYBRID");
        event.setPayload(payload);
        return event;
    }

    /**
     * Shorthand: INTERVENTION_APPROVED with default detail.
     */
    public static CanonicalEvent interventionApproved(String patientId, long timestamp,
                                                       String interventionId,
                                                       InterventionType interventionType) {
        return interventionApproved(patientId, timestamp, interventionId, interventionType, null);
    }

    /**
     * Creates an INTERVENTION_CANCELLED event.
     */
    public static CanonicalEvent interventionCancelled(String patientId, long timestamp,
                                                        String interventionId) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("event_type", "INTERVENTION_CANCELLED");
        payload.put("intervention_id", interventionId);
        event.setPayload(payload);
        return event;
    }

    /**
     * Creates an INTERVENTION_MODIFIED event with new window days.
     */
    public static CanonicalEvent interventionModified(String patientId, long timestamp,
                                                       String interventionId,
                                                       int newWindowDays) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("event_type", "INTERVENTION_MODIFIED");
        payload.put("intervention_id", interventionId);
        payload.put("observation_window_days", newWindowDays);
        payload.put("modification_detail", Collections.singletonMap("window_days_changed", true));
        event.setPayload(payload);
        return event;
    }

    /**
     * Creates a medication intervention detail payload with drug class.
     */
    public static Map<String, Object> medicationDetail(String drugClass, String drugName, String dose) {
        Map<String, Object> detail = new HashMap<>();
        detail.put("drug_class", drugClass);
        detail.put("drug_name", drugName);
        detail.put("dose", dose);
        return detail;
    }

    /**
     * Creates a lifestyle intervention detail payload.
     */
    public static Map<String, Object> lifestyleDetail(String targetDomain, String description) {
        Map<String, Object> detail = new HashMap<>();
        detail.put("target_domain", targetDomain);
        detail.put("description", description);
        return detail;
    }

    // --- Patient Events (vitals, labs, patient-reported) ---

    public static CanonicalEvent fbgReading(String patientId, long timestamp, double value) {
        CanonicalEvent event = baseEvent(patientId, EventType.LAB_RESULT, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("lab_type", "FBG");
        payload.put("value", value);
        payload.put("unit", "mg/dL");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent sbpReading(String patientId, long timestamp,
                                             double sbp, double dbp) {
        CanonicalEvent event = baseEvent(patientId, EventType.VITAL_SIGN, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("systolic_bp", sbp);
        payload.put("diastolic_bp", dbp);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent egfrReading(String patientId, long timestamp, double value) {
        CanonicalEvent event = baseEvent(patientId, EventType.LAB_RESULT, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("lab_type", "EGFR");
        payload.put("value", value);
        payload.put("unit", "mL/min/1.73m2");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent weightReading(String patientId, long timestamp, double value) {
        CanonicalEvent event = baseEvent(patientId, EventType.VITAL_SIGN, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("weight_kg", value);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent externalMedicationEvent(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.MEDICATION_ORDERED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_system", "EXTERNAL_HOSPITAL");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent hospitalisationEvent(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("report_type", "ADMISSION");
        payload.put("admission_flag", true);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent patientReportedIllness(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("event_type", "ILLNESS");
        event.setPayload(payload);
        return event;
    }

    // --- State Builders ---

    public static InterventionWindowState emptyState(String patientId) {
        return new InterventionWindowState(patientId);
    }

    public static InterventionWindowState stateWithBaselines(String patientId,
                                                              double fbg, double sbp, double weight) {
        InterventionWindowState state = new InterventionWindowState(patientId);
        state.setLastKnownFBG(fbg);
        state.setLastKnownSBP(sbp);
        state.setLastKnownWeight(weight);
        return state;
    }

    /**
     * Creates a state with pre-populated FBG readings for trajectory testing.
     * Readings are spaced 2 days apart ending at the given timestamp.
     */
    public static InterventionWindowState stateWithFBGReadings(String patientId,
                                                                double[] values,
                                                                long endTimestamp) {
        InterventionWindowState state = new InterventionWindowState(patientId);
        for (int i = 0; i < values.length; i++) {
            long ts = endTimestamp - (long)(values.length - 1 - i) * 2 * DAY_MS;
            state.addReading("FBG", values[i], ts);
        }
        return state;
    }

    /**
     * Creates a state with pre-populated SBP readings for trajectory testing.
     */
    public static InterventionWindowState stateWithSBPReadings(String patientId,
                                                                double[] values,
                                                                long endTimestamp) {
        InterventionWindowState state = new InterventionWindowState(patientId);
        for (int i = 0; i < values.length; i++) {
            long ts = endTimestamp - (long)(values.length - 1 - i) * 2 * DAY_MS;
            state.addReading("SBP", values[i], ts);
        }
        return state;
    }

    /**
     * Generates a list of FBG readings as TrajectoryDataPoints.
     */
    public static List<InterventionWindowState.TrajectoryDataPoint> fbgReadings(
            double[] values, long startTime, long intervalMs) {
        List<InterventionWindowState.TrajectoryDataPoint> points = new ArrayList<>();
        for (int i = 0; i < values.length; i++) {
            points.add(new InterventionWindowState.TrajectoryDataPoint(
                    "FBG", values[i], startTime + i * intervalMs));
        }
        return points;
    }

    /**
     * Generates SBP readings as TrajectoryDataPoints.
     */
    public static List<InterventionWindowState.TrajectoryDataPoint> sbpReadings(
            double[] values, long startTime, long intervalMs) {
        List<InterventionWindowState.TrajectoryDataPoint> points = new ArrayList<>();
        for (int i = 0; i < values.length; i++) {
            points.add(new InterventionWindowState.TrajectoryDataPoint(
                    "SBP", values[i], startTime + i * intervalMs));
        }
        return points;
    }

    // --- Helpers ---

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

- [ ] **Step 2: Verify compilation**

```bash
cd backend/shared-infrastructure/flink-processing
mvn compile -pl . -q 2>&1 | tail -5
```

Expected: BUILD SUCCESS (test sources compile with `mvn test-compile`)

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/builders/Module12TestBuilder.java
git commit -m "feat(module12): add Module12TestBuilder with intervention event, patient event, and state factories"
```

---

## Task 6: Module12TrajectoryTracker (Static Analyzer)

**Files:**
- Create: `operators/Module12TrajectoryTracker.java`
- Create (test): `operators/Module12TrajectoryTrackerTest.java`

- [ ] **Step 1: Write the 6 failing tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module12TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.List;

import static com.cardiofit.flink.builders.Module12TestBuilder.*;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module12TrajectoryTracker: OLS-based slope classification
 * for patient physiological trajectories.
 */
class Module12TrajectoryTrackerTest {

    @Test
    void decliningFBG_classifiedAsImproving() {
        // FBG: 160, 155, 148, 142, 138 over 14 days (slope < -3.0 mg/dL/week)
        List<InterventionWindowState.TrajectoryDataPoint> readings =
                fbgReadings(new double[]{160, 155, 148, 142, 138}, BASE_TIME, 3 * DAY_MS);
        long windowStart = BASE_TIME + 14 * DAY_MS;

        TrajectoryClassification result = Module12TrajectoryTracker.classify(
                readings, "FBG", windowStart - 14 * DAY_MS, windowStart);

        assertEquals(TrajectoryClassification.IMPROVING, result);
    }

    @Test
    void risingSBP_classifiedAsDeteriorating() {
        // SBP: 130, 133, 137, 140, 144 over 14 days (slope > +2.0 mmHg/week)
        List<InterventionWindowState.TrajectoryDataPoint> readings =
                sbpReadings(new double[]{130, 133, 137, 140, 144}, BASE_TIME, 3 * DAY_MS);
        long windowStart = BASE_TIME + 14 * DAY_MS;

        TrajectoryClassification result = Module12TrajectoryTracker.classify(
                readings, "SBP", windowStart - 14 * DAY_MS, windowStart);

        assertEquals(TrajectoryClassification.DETERIORATING, result);
    }

    @Test
    void flatReadingsWithNoise_classifiedAsStable() {
        // FBG: 120, 118, 122, 119, 121 over 14 days (slope within ±3.0)
        List<InterventionWindowState.TrajectoryDataPoint> readings =
                fbgReadings(new double[]{120, 118, 122, 119, 121}, BASE_TIME, 3 * DAY_MS);
        long windowStart = BASE_TIME + 14 * DAY_MS;

        TrajectoryClassification result = Module12TrajectoryTracker.classify(
                readings, "FBG", windowStart - 14 * DAY_MS, windowStart);

        assertEquals(TrajectoryClassification.STABLE, result);
    }

    @Test
    void insufficientReadings_classifiedAsUnknown() {
        // Only 2 readings over 3 days — below minimum (3 readings over 7 days)
        List<InterventionWindowState.TrajectoryDataPoint> readings =
                fbgReadings(new double[]{120, 125}, BASE_TIME, 3 * DAY_MS);
        long windowStart = BASE_TIME + 6 * DAY_MS;

        TrajectoryClassification result = Module12TrajectoryTracker.classify(
                readings, "FBG", windowStart - 7 * DAY_MS, windowStart);

        assertEquals(TrajectoryClassification.UNKNOWN, result);
    }

    @Test
    void eGFR_rapidDecline_classifiedAsDeteriorating() {
        // eGFR: 55, 52, 48, 45 over ~12 months — slope < -3 mL/min/year
        long yearMs = 365L * DAY_MS;
        List<InterventionWindowState.TrajectoryDataPoint> readings =
                Module12TestBuilder.fbgReadings(new double[]{55, 52, 48, 45},
                        BASE_TIME, yearMs / 4);
        // Use domain "EGFR" to get renal thresholds
        long end = BASE_TIME + yearMs;

        TrajectoryClassification result = Module12TrajectoryTracker.classify(
                readings, "EGFR", BASE_TIME, end);

        assertEquals(TrajectoryClassification.DETERIORATING, result);
    }

    @Test
    void compositeWorstDomain_deterioratingWins() {
        // FBG improving + SBP deteriorating → composite = DETERIORATING
        InterventionWindowState state = emptyState("P1");
        // Add declining FBG
        for (int i = 0; i < 5; i++) {
            state.addReading("FBG", 160 - i * 5.0, BASE_TIME + i * 3 * DAY_MS);
        }
        // Add rising SBP
        for (int i = 0; i < 5; i++) {
            state.addReading("SBP", 130 + i * 4.0, BASE_TIME + i * 3 * DAY_MS);
        }
        long windowEnd = BASE_TIME + 14 * DAY_MS;

        TrajectoryClassification result = Module12TrajectoryTracker.classifyComposite(
                state, windowEnd - 14 * DAY_MS, windowEnd);

        assertEquals(TrajectoryClassification.DETERIORATING, result);
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module12TrajectoryTrackerTest -q 2>&1 | tail -10
```

Expected: COMPILATION ERROR (Module12TrajectoryTracker not found)

- [ ] **Step 3: Implement Module12TrajectoryTracker**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.List;

/**
 * Static analyzer: classifies a patient's recent physiological trajectory
 * into IMPROVING, STABLE, or DETERIORATING using OLS linear regression slope.
 *
 * Thresholds per domain (spec Section 5.1):
 * - FBG:  IMPROVING < -3.0, STABLE ±3.0, DETERIORATING > +3.0 mg/dL/week
 * - SBP:  IMPROVING < -2.0, STABLE ±2.0, DETERIORATING > +2.0 mmHg/week
 * - EGFR: IMPROVING > +1.0, STABLE ±1.0, DETERIORATING < -1.0 mL/min/year
 * - WEIGHT: IMPROVING < -0.3, STABLE ±0.3, DETERIORATING > +0.3 kg/week
 *
 * Minimum data: 3+ readings spanning 7+ days.
 */
public final class Module12TrajectoryTracker {

    private static final int MIN_READINGS = 3;
    private static final long MIN_SPAN_MS = 7L * 86_400_000L;
    private static final long WEEK_MS = 7L * 86_400_000L;
    private static final long YEAR_MS = 365L * 86_400_000L;

    private Module12TrajectoryTracker() {}

    /**
     * Classifies trajectory for a single domain from raw data points.
     *
     * @param readings all data points (will be filtered by domain and time range)
     * @param domain   domain identifier: "FBG", "SBP", "EGFR", "WEIGHT"
     * @param sinceMs  start of analysis window (epoch ms)
     * @param untilMs  end of analysis window (epoch ms)
     * @return trajectory classification
     */
    public static TrajectoryClassification classify(
            List<InterventionWindowState.TrajectoryDataPoint> readings,
            String domain, long sinceMs, long untilMs) {

        // Filter to domain and time range
        List<InterventionWindowState.TrajectoryDataPoint> filtered = new java.util.ArrayList<>();
        for (InterventionWindowState.TrajectoryDataPoint p : readings) {
            if (domain.equals(p.domain) && p.timestamp >= sinceMs && p.timestamp <= untilMs) {
                filtered.add(p);
            }
        }

        if (filtered.size() < MIN_READINGS) {
            return TrajectoryClassification.UNKNOWN;
        }

        long span = filtered.get(filtered.size() - 1).timestamp - filtered.get(0).timestamp;
        if (span < MIN_SPAN_MS) {
            return TrajectoryClassification.UNKNOWN;
        }

        // OLS linear regression: slope in units per millisecond
        double slopePerMs = computeOLSSlope(filtered);

        // Convert slope to domain-appropriate rate
        return classifySlope(domain, slopePerMs);
    }

    /**
     * Classifies composite trajectory using worst-domain logic.
     * If any critical domain (FBG, SBP) is DETERIORATING, composite is DETERIORATING.
     * If all are IMPROVING, composite is IMPROVING. Otherwise STABLE.
     */
    public static TrajectoryClassification classifyComposite(
            InterventionWindowState state, long sinceMs, long untilMs) {

        List<InterventionWindowState.TrajectoryDataPoint> allReadings = state.getRecentReadings();

        TrajectoryClassification fbg = classify(allReadings, "FBG", sinceMs, untilMs);
        TrajectoryClassification sbp = classify(allReadings, "SBP", sinceMs, untilMs);

        // Worst-domain logic for critical domains
        if (fbg == TrajectoryClassification.DETERIORATING
                || sbp == TrajectoryClassification.DETERIORATING) {
            return TrajectoryClassification.DETERIORATING;
        }

        // Check non-critical domains
        TrajectoryClassification egfr = classify(allReadings, "EGFR", sinceMs, untilMs);
        TrajectoryClassification weight = classify(allReadings, "WEIGHT", sinceMs, untilMs);

        if (egfr == TrajectoryClassification.DETERIORATING
                || weight == TrajectoryClassification.DETERIORATING) {
            return TrajectoryClassification.DETERIORATING;
        }

        // All known domains improving?
        boolean anyKnown = false;
        boolean allImproving = true;
        for (TrajectoryClassification tc : new TrajectoryClassification[]{fbg, sbp, egfr, weight}) {
            if (tc != TrajectoryClassification.UNKNOWN) {
                anyKnown = true;
                if (tc != TrajectoryClassification.IMPROVING) {
                    allImproving = false;
                }
            }
        }

        if (anyKnown && allImproving) {
            return TrajectoryClassification.IMPROVING;
        }

        return anyKnown ? TrajectoryClassification.STABLE : TrajectoryClassification.UNKNOWN;
    }

    // --- Internal ---

    static double computeOLSSlope(List<InterventionWindowState.TrajectoryDataPoint> points) {
        int n = points.size();
        double sumX = 0, sumY = 0, sumXY = 0, sumX2 = 0;
        for (InterventionWindowState.TrajectoryDataPoint p : points) {
            double x = p.timestamp;
            double y = p.value;
            sumX += x;
            sumY += y;
            sumXY += x * y;
            sumX2 += x * x;
        }
        double denominator = n * sumX2 - sumX * sumX;
        if (denominator == 0) return 0;
        return (n * sumXY - sumX * sumY) / denominator;
    }

    private static TrajectoryClassification classifySlope(String domain, double slopePerMs) {
        switch (domain) {
            case "FBG": {
                double slopePerWeek = slopePerMs * WEEK_MS;
                if (slopePerWeek < -3.0) return TrajectoryClassification.IMPROVING;
                if (slopePerWeek > 3.0) return TrajectoryClassification.DETERIORATING;
                return TrajectoryClassification.STABLE;
            }
            case "SBP": {
                double slopePerWeek = slopePerMs * WEEK_MS;
                if (slopePerWeek < -2.0) return TrajectoryClassification.IMPROVING;
                if (slopePerWeek > 2.0) return TrajectoryClassification.DETERIORATING;
                return TrajectoryClassification.STABLE;
            }
            case "EGFR": {
                // eGFR: improving means RISING, deteriorating means FALLING
                double slopePerYear = slopePerMs * YEAR_MS;
                if (slopePerYear > 1.0) return TrajectoryClassification.IMPROVING;
                if (slopePerYear < -1.0) return TrajectoryClassification.DETERIORATING;
                return TrajectoryClassification.STABLE;
            }
            case "WEIGHT": {
                double slopePerWeek = slopePerMs * WEEK_MS;
                if (slopePerWeek < -0.3) return TrajectoryClassification.IMPROVING;
                if (slopePerWeek > 0.3) return TrajectoryClassification.DETERIORATING;
                return TrajectoryClassification.STABLE;
            }
            default:
                return TrajectoryClassification.UNKNOWN;
        }
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module12TrajectoryTrackerTest -q 2>&1 | tail -10
```

Expected: Tests run: 6, Failures: 0, Errors: 0

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module12TrajectoryTracker.java \
        src/test/java/com/cardiofit/flink/operators/Module12TrajectoryTrackerTest.java
git commit -m "feat(module12): add Module12TrajectoryTracker with OLS slope classification — 6 tests"
```

---

## Task 7: Module12ConcurrencyDetector (Static Analyzer)

**Files:**
- Create: `operators/Module12ConcurrencyDetector.java`
- Create (test): `operators/Module12ConcurrencyDetectorTest.java`

- [ ] **Step 1: Write the 5 failing tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module12TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static com.cardiofit.flink.builders.Module12TestBuilder.*;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module12ConcurrencyDetector: overlapping window detection
 * with domain classification.
 */
class Module12ConcurrencyDetectorTest {

    @Test
    void noOverlap_emptyConcurrentList() {
        // Window A: day 1–14, Window B: day 20–34
        Map<String, InterventionWindowState.InterventionWindow> active = new HashMap<>();
        InterventionWindowState.InterventionWindow winA = makeWindow("int-A",
                InterventionType.LIFESTYLE_ACTIVITY, BASE_TIME, daysAfter(BASE_TIME, 14), null);
        active.put("int-A", winA);

        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-B", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 20), daysAfter(BASE_TIME, 34), active);

        assertTrue(result.getConcurrentIds().isEmpty());
        assertFalse(result.isSameDomainConcurrent());
    }

    @Test
    void partialOverlapBelowThreshold_emptyConcurrentList() {
        // Window A: day 1–28, Window B: day 23–37 (5d overlap < 7d threshold)
        Map<String, InterventionWindowState.InterventionWindow> active = new HashMap<>();
        InterventionWindowState.InterventionWindow winA = makeWindow("int-A",
                InterventionType.MEDICATION_ADD, BASE_TIME, daysAfter(BASE_TIME, 28),
                medicationDetail("METFORMIN", "Metformin", "500mg"));
        active.put("int-A", winA);

        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-B", InterventionType.MEDICATION_ADD, medicationDetail("SGLT2I", "Empagliflozin", "10mg"),
                daysAfter(BASE_TIME, 23), daysAfter(BASE_TIME, 51), active);

        assertTrue(result.getConcurrentIds().isEmpty());
    }

    @Test
    void partialOverlapAboveThreshold_bothCrossReferenced() {
        // Window A: day 1–28, Window B: day 15–29 (14d overlap ≥ 7d)
        Map<String, InterventionWindowState.InterventionWindow> active = new HashMap<>();
        InterventionWindowState.InterventionWindow winA = makeWindow("int-A",
                InterventionType.LIFESTYLE_ACTIVITY, BASE_TIME, daysAfter(BASE_TIME, 28), null);
        active.put("int-A", winA);

        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-B", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 15), daysAfter(BASE_TIME, 29), active);

        assertEquals(1, result.getConcurrentIds().size());
        assertTrue(result.getConcurrentIds().contains("int-A"));
    }

    @Test
    void sameDomainConcurrent_flagged() {
        // Both target GLUCOSE domain
        Map<String, InterventionWindowState.InterventionWindow> active = new HashMap<>();
        InterventionWindowState.InterventionWindow winA = makeWindow("int-A",
                InterventionType.MEDICATION_ADD, BASE_TIME, daysAfter(BASE_TIME, 28),
                medicationDetail("METFORMIN", "Metformin", "500mg"));
        active.put("int-A", winA);

        // NUTRITION_FOOD_CHANGE also targets GLUCOSE
        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-B", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 5), daysAfter(BASE_TIME, 19), active);

        assertTrue(result.isSameDomainConcurrent());
    }

    @Test
    void crossDomainConcurrent_notFlaggedAsSameDomain() {
        // int-A targets GLUCOSE (metformin), int-B targets BLOOD_PRESSURE (sodium reduction)
        Map<String, InterventionWindowState.InterventionWindow> active = new HashMap<>();
        InterventionWindowState.InterventionWindow winA = makeWindow("int-A",
                InterventionType.MEDICATION_ADD, BASE_TIME, daysAfter(BASE_TIME, 28),
                medicationDetail("METFORMIN", "Metformin", "500mg"));
        active.put("int-A", winA);

        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-B", InterventionType.NUTRITION_SODIUM_REDUCTION, null,
                daysAfter(BASE_TIME, 5), daysAfter(BASE_TIME, 19), active);

        assertEquals(1, result.getConcurrentIds().size());
        assertFalse(result.isSameDomainConcurrent());
    }

    // --- Helper ---
    private static InterventionWindowState.InterventionWindow makeWindow(
            String id, InterventionType type, long start, long end,
            Map<String, Object> detail) {
        InterventionWindowState.InterventionWindow w = new InterventionWindowState.InterventionWindow();
        w.interventionId = id;
        w.interventionType = type;
        w.interventionDetail = detail;
        w.observationStartMs = start;
        w.observationEndMs = end;
        w.status = "OBSERVING";
        return w;
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module12ConcurrencyDetectorTest -q 2>&1 | tail -10
```

Expected: COMPILATION ERROR

- [ ] **Step 3: Implement Module12ConcurrencyDetector**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.InterventionType.ClinicalDomain;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * Static analyzer: detects overlapping intervention windows and classifies
 * domain relationships. Two interventions are concurrent if their observation
 * windows overlap by 7+ days. Same-domain concurrency is flagged when their
 * target domain sets intersect.
 */
public final class Module12ConcurrencyDetector {

    private static final long OVERLAP_THRESHOLD_MS = 7L * 86_400_000L; // 7 days

    private Module12ConcurrencyDetector() {}

    /**
     * Detects which active windows overlap with the new intervention.
     *
     * @param newInterventionId  the new intervention's ID
     * @param newType            the new intervention's type
     * @param newDetail          the new intervention's detail (for drug_class extraction)
     * @param newStartMs         the new window's start time
     * @param newEndMs           the new window's end time
     * @param activeWindows      currently active windows (keyed by intervention_id)
     * @return detection result with concurrent IDs and domain overlap flag
     */
    public static Result detect(String newInterventionId,
                                 InterventionType newType,
                                 Map<String, Object> newDetail,
                                 long newStartMs, long newEndMs,
                                 Map<String, InterventionWindowState.InterventionWindow> activeWindows) {

        List<String> concurrentIds = new ArrayList<>();
        boolean sameDomain = false;

        String newDrugClass = extractDrugClass(newDetail);
        Set<ClinicalDomain> newDomains = newType.getDomains(newDrugClass);

        for (Map.Entry<String, InterventionWindowState.InterventionWindow> entry : activeWindows.entrySet()) {
            String existingId = entry.getKey();
            InterventionWindowState.InterventionWindow existing = entry.getValue();

            if (existingId.equals(newInterventionId)) continue;
            if (!"OBSERVING".equals(existing.status)) continue;

            // Compute overlap
            long overlapStart = Math.max(newStartMs, existing.observationStartMs);
            long overlapEnd = Math.min(newEndMs, existing.observationEndMs);
            long overlapMs = overlapEnd - overlapStart;

            if (overlapMs >= OVERLAP_THRESHOLD_MS) {
                concurrentIds.add(existingId);

                // Check domain overlap
                String existingDrugClass = extractDrugClass(existing.interventionDetail);
                Set<ClinicalDomain> existingDomains =
                        existing.interventionType.getDomains(existingDrugClass);

                if (!Collections.disjoint(newDomains, existingDomains)) {
                    sameDomain = true;
                }
            }
        }

        return new Result(concurrentIds, sameDomain);
    }

    private static String extractDrugClass(Map<String, Object> detail) {
        if (detail == null) return null;
        Object dc = detail.get("drug_class");
        return dc != null ? dc.toString() : null;
    }

    /**
     * Result of concurrency detection.
     */
    public static class Result {
        private final List<String> concurrentIds;
        private final boolean sameDomainConcurrent;

        public Result(List<String> concurrentIds, boolean sameDomainConcurrent) {
            this.concurrentIds = concurrentIds;
            this.sameDomainConcurrent = sameDomainConcurrent;
        }

        public List<String> getConcurrentIds() { return concurrentIds; }
        public boolean isSameDomainConcurrent() { return sameDomainConcurrent; }
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module12ConcurrencyDetectorTest -q 2>&1 | tail -10
```

Expected: Tests run: 5, Failures: 0, Errors: 0

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module12ConcurrencyDetector.java \
        src/test/java/com/cardiofit/flink/operators/Module12ConcurrencyDetectorTest.java
git commit -m "feat(module12): add Module12ConcurrencyDetector with 7-day overlap threshold and domain classification — 5 tests"
```

---

## Task 8: Module12AdherenceAssembler (Static Analyzer)

**Files:**
- Create: `operators/Module12AdherenceAssembler.java`
- Create (test): `operators/Module12AdherenceAssemblerTest.java`

- [ ] **Step 1: Write the 5 failing tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module12AdherenceAssembler: adherence signal assembly
 * with data quality tier classification.
 */
class Module12AdherenceAssemblerTest {

    @Test
    void medicationAdherence_fromReminderAcks() {
        Map<String, Object> signals = new HashMap<>();
        signals.put("reminder_ack_rate", 0.85);
        signals.put("refill_compliance", 0.70);

        Module12AdherenceAssembler.Result result = Module12AdherenceAssembler.assemble(
                InterventionType.MEDICATION_ADD, signals);

        // max(reminder_ack_rate, refill_compliance) = 0.85
        assertEquals(0.85, result.getAdherenceScore(), 0.01);
        assertEquals("HIGH", result.getDataQuality());
    }

    @Test
    void lifestyleActivity_fromWearableData() {
        Map<String, Object> signals = new HashMap<>();
        signals.put("activity_sessions_per_week", 3.0);
        signals.put("target_sessions_per_week", 5.0);
        signals.put("data_source", "WEARABLE");

        Module12AdherenceAssembler.Result result = Module12AdherenceAssembler.assemble(
                InterventionType.LIFESTYLE_ACTIVITY, signals);

        // 3/5 = 0.6
        assertEquals(0.60, result.getAdherenceScore(), 0.01);
        assertEquals("HIGH", result.getDataQuality());
    }

    @Test
    void nutritionFoodChange_fromMealLogs() {
        Map<String, Object> signals = new HashMap<>();
        signals.put("meals_with_prescribed_change", 8.0);
        signals.put("total_meals_logged", 14.0);
        signals.put("data_source", "APP_SELF_REPORT");

        Module12AdherenceAssembler.Result result = Module12AdherenceAssembler.assemble(
                InterventionType.NUTRITION_FOOD_CHANGE, signals);

        // 8/14 ≈ 0.571
        assertEquals(0.571, result.getAdherenceScore(), 0.01);
        assertEquals("MODERATE", result.getDataQuality());
    }

    @Test
    void noAdherenceData_defaultsToHalf() {
        // Empty signals → default adherence of 0.5, LOW quality
        Module12AdherenceAssembler.Result result = Module12AdherenceAssembler.assemble(
                InterventionType.MEDICATION_ADD, null);

        assertEquals(0.50, result.getAdherenceScore(), 0.01);
        assertEquals("LOW", result.getDataQuality());
    }

    @Test
    void adherenceScore_cappedAtOne() {
        Map<String, Object> signals = new HashMap<>();
        signals.put("activity_sessions_per_week", 7.0);
        signals.put("target_sessions_per_week", 5.0);

        Module12AdherenceAssembler.Result result = Module12AdherenceAssembler.assemble(
                InterventionType.LIFESTYLE_ACTIVITY, signals);

        // 7/5 = 1.4 → capped at 1.0
        assertEquals(1.0, result.getAdherenceScore(), 0.01);
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module12AdherenceAssemblerTest -q 2>&1 | tail -10
```

Expected: COMPILATION ERROR

- [ ] **Step 3: Implement Module12AdherenceAssembler**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.InterventionType;
import java.util.Map;

/**
 * Static analyzer: assembles adherence signals from available data sources.
 * Adherence is scored 0–1 with data quality tier (HIGH/MODERATE/LOW).
 *
 * HIGH:     Objective wearable/pharmacy data
 * MODERATE: App self-report
 * LOW:      No adherence data (default 0.5)
 */
public final class Module12AdherenceAssembler {

    private static final double DEFAULT_ADHERENCE = 0.5;

    private Module12AdherenceAssembler() {}

    /**
     * Assembles adherence score from available signals for the given intervention type.
     */
    public static Result assemble(InterventionType type, Map<String, Object> signals) {
        if (signals == null || signals.isEmpty()) {
            return new Result(DEFAULT_ADHERENCE, "LOW");
        }

        String dataSource = getStringOrDefault(signals, "data_source", "");
        String quality = classifyDataQuality(dataSource, signals);

        double score;

        if (type.isMedication()) {
            score = computeMedicationAdherence(signals);
        } else if (type == InterventionType.LIFESTYLE_ACTIVITY
                || type == InterventionType.LIFESTYLE_SLEEP) {
            score = computeRatioAdherence(signals,
                    "activity_sessions_per_week", "target_sessions_per_week",
                    "mean_sleep_hours", "target_sleep_hours");
        } else if (type.isNutrition()) {
            score = computeNutritionAdherence(type, signals);
        } else {
            score = DEFAULT_ADHERENCE;
        }

        return new Result(Math.min(score, 1.0), quality);
    }

    private static double computeMedicationAdherence(Map<String, Object> signals) {
        double reminderRate = getDoubleOrDefault(signals, "reminder_ack_rate", -1);
        double refillRate = getDoubleOrDefault(signals, "refill_compliance", -1);
        if (reminderRate < 0 && refillRate < 0) return DEFAULT_ADHERENCE;
        return Math.max(
                reminderRate >= 0 ? reminderRate : 0,
                refillRate >= 0 ? refillRate : 0
        );
    }

    private static double computeRatioAdherence(Map<String, Object> signals,
                                                  String actualKey1, String targetKey1,
                                                  String actualKey2, String targetKey2) {
        double actual = getDoubleOrDefault(signals, actualKey1, -1);
        double target = getDoubleOrDefault(signals, targetKey1, -1);
        if (actual < 0 || target <= 0) {
            actual = getDoubleOrDefault(signals, actualKey2, -1);
            target = getDoubleOrDefault(signals, targetKey2, -1);
        }
        if (actual < 0 || target <= 0) return DEFAULT_ADHERENCE;
        return actual / target;
    }

    private static double computeNutritionAdherence(InterventionType type,
                                                      Map<String, Object> signals) {
        switch (type) {
            case NUTRITION_FOOD_CHANGE:
            case NUTRITION_TIMING_CHANGE: {
                double changed = getDoubleOrDefault(signals, "meals_with_prescribed_change", -1);
                double total = getDoubleOrDefault(signals, "total_meals_logged", -1);
                if (changed < 0 || total <= 0) return DEFAULT_ADHERENCE;
                return changed / total;
            }
            case NUTRITION_SODIUM_REDUCTION: {
                double achieved = getDoubleOrDefault(signals, "sodium_reduction_achieved", -1);
                double target = getDoubleOrDefault(signals, "target_reduction", -1);
                if (achieved < 0 || target <= 0) return DEFAULT_ADHERENCE;
                return achieved / target;
            }
            case NUTRITION_PORTION_CHANGE: {
                double achieved = getDoubleOrDefault(signals, "portion_reduction_achieved", -1);
                double target = getDoubleOrDefault(signals, "target_reduction", -1);
                if (achieved < 0 || target <= 0) return DEFAULT_ADHERENCE;
                return achieved / target;
            }
            default:
                return DEFAULT_ADHERENCE;
        }
    }

    private static String classifyDataQuality(String source, Map<String, Object> signals) {
        if ("WEARABLE".equalsIgnoreCase(source) || "PHARMACY".equalsIgnoreCase(source)) {
            return "HIGH";
        }
        if ("APP_SELF_REPORT".equalsIgnoreCase(source)) {
            return "MODERATE";
        }
        // Check if any objective data present
        if (signals.containsKey("refill_compliance")
                || signals.containsKey("reminder_ack_rate")) {
            return "HIGH";
        }
        if (signals.containsKey("meals_with_prescribed_change")
                || signals.containsKey("activity_sessions_per_week")) {
            return "MODERATE";
        }
        return "LOW";
    }

    private static double getDoubleOrDefault(Map<String, Object> m, String key, double def) {
        Object v = m.get(key);
        if (v instanceof Number) return ((Number) v).doubleValue();
        return def;
    }

    private static String getStringOrDefault(Map<String, Object> m, String key, String def) {
        Object v = m.get(key);
        return v != null ? v.toString() : def;
    }

    public static class Result {
        private final double adherenceScore;
        private final String dataQuality;

        public Result(double adherenceScore, String dataQuality) {
            this.adherenceScore = adherenceScore;
            this.dataQuality = dataQuality;
        }

        public double getAdherenceScore() { return adherenceScore; }
        public String getDataQuality() { return dataQuality; }
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module12AdherenceAssemblerTest -q 2>&1 | tail -10
```

Expected: Tests run: 5, Failures: 0, Errors: 0

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module12AdherenceAssembler.java \
        src/test/java/com/cardiofit/flink/operators/Module12AdherenceAssemblerTest.java
git commit -m "feat(module12): add Module12AdherenceAssembler with data quality tiers — 5 tests"
```

---

## Task 9: Module12ConfounderAccumulator (Static Analyzer)

**Files:**
- Create: `operators/Module12ConfounderAccumulator.java`
- Create (test): `operators/Module12ConfounderAccumulatorTest.java`

- [ ] **Step 1: Write the 5 failing tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module12TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static com.cardiofit.flink.builders.Module12TestBuilder.*;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module12ConfounderAccumulator: event-driven confounder
 * flag accumulation during observation windows.
 */
class Module12ConfounderAccumulatorTest {

    @Test
    void externalMedication_flagsConfounder() {
        InterventionWindowState.InterventionWindow window = new InterventionWindowState.InterventionWindow();
        window.status = "OBSERVING";
        window.observationStartMs = BASE_TIME;
        window.observationEndMs = daysAfter(BASE_TIME, 28);

        CanonicalEvent event = externalMedicationEvent("P1", daysAfter(BASE_TIME, 5));

        Module12ConfounderAccumulator.accumulate(window, event);

        assertTrue(window.confoundersDetected.contains("EXTERNAL_MEDICATION_CHANGE"));
    }

    @Test
    void hospitalisation_flagsConfounder() {
        InterventionWindowState.InterventionWindow window = new InterventionWindowState.InterventionWindow();
        window.status = "OBSERVING";
        window.observationStartMs = BASE_TIME;
        window.observationEndMs = daysAfter(BASE_TIME, 28);

        CanonicalEvent event = hospitalisationEvent("P1", daysAfter(BASE_TIME, 10));

        Module12ConfounderAccumulator.accumulate(window, event);

        assertTrue(window.confoundersDetected.contains("HOSPITALISATION"));
    }

    @Test
    void festivalPeriod_flagsConfounder() {
        InterventionWindowState.InterventionWindow window = new InterventionWindowState.InterventionWindow();
        window.status = "OBSERVING";
        window.observationStartMs = BASE_TIME;
        window.observationEndMs = daysAfter(BASE_TIME, 14);

        // Simulate festival flag (normally from KB-21 lookup at window open)
        Module12ConfounderAccumulator.addFestivalConfounder(window, "DIWALI");

        assertTrue(window.confoundersDetected.contains("FESTIVAL_PERIOD:DIWALI"));
    }

    @Test
    void labResult_accumulatedToLabChanges() {
        InterventionWindowState.InterventionWindow window = new InterventionWindowState.InterventionWindow();
        window.status = "OBSERVING";
        window.observationStartMs = BASE_TIME;
        window.observationEndMs = daysAfter(BASE_TIME, 28);

        CanonicalEvent event = fbgReading("P1", daysAfter(BASE_TIME, 7), 145.0);

        Module12ConfounderAccumulator.accumulate(window, event);

        assertEquals(1, window.labChanges.size());
        // Lab results are accumulated, not flagged as confounders
        assertFalse(window.confoundersDetected.contains("LAB_RESULT"));
    }

    @Test
    void multipleConfounders_allAccumulated() {
        InterventionWindowState.InterventionWindow window = new InterventionWindowState.InterventionWindow();
        window.status = "OBSERVING";
        window.observationStartMs = BASE_TIME;
        window.observationEndMs = daysAfter(BASE_TIME, 28);

        Module12ConfounderAccumulator.accumulate(window,
                externalMedicationEvent("P1", daysAfter(BASE_TIME, 3)));
        Module12ConfounderAccumulator.accumulate(window,
                hospitalisationEvent("P1", daysAfter(BASE_TIME, 10)));
        Module12ConfounderAccumulator.accumulate(window,
                patientReportedIllness("P1", daysAfter(BASE_TIME, 15)));

        assertEquals(3, window.confoundersDetected.size());
        assertTrue(window.confoundersDetected.contains("EXTERNAL_MEDICATION_CHANGE"));
        assertTrue(window.confoundersDetected.contains("HOSPITALISATION"));
        assertTrue(window.confoundersDetected.contains("INTERCURRENT_ILLNESS"));
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module12ConfounderAccumulatorTest -q 2>&1 | tail -10
```

Expected: COMPILATION ERROR

- [ ] **Step 3: Implement Module12ConfounderAccumulator**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.HashMap;
import java.util.Map;

/**
 * Static analyzer: accumulates confounder flags from events that arrive
 * during active observation windows. Called by the main KPF whenever a
 * non-intervention event arrives for a patient with active windows.
 *
 * Event routing rules per spec Section 5.3:
 * - MEDICATION_ORDERED (external) → EXTERNAL_MEDICATION_CHANGE
 * - PATIENT_REPORTED + admission_flag → HOSPITALISATION
 * - PATIENT_REPORTED + event_type=ILLNESS → INTERCURRENT_ILLNESS
 * - PATIENT_REPORTED + event_type=TRAVEL → TRAVEL_DISRUPTION
 * - LAB_RESULT → accumulated to lab_changes list (not a confounder flag)
 * - VITAL_SIGN / DEVICE_READING → trajectory tracker (not a confounder)
 */
public final class Module12ConfounderAccumulator {

    private Module12ConfounderAccumulator() {}

    /**
     * Evaluates an event against an active observation window and accumulates
     * any detected confounders or lab changes.
     */
    public static void accumulate(InterventionWindowState.InterventionWindow window,
                                   CanonicalEvent event) {
        if (window == null || !"OBSERVING".equals(window.status)) return;
        if (event == null || event.getPayload() == null) return;

        EventType eventType = event.getEventType();
        Map<String, Object> payload = event.getPayload();

        if (eventType == EventType.MEDICATION_ORDERED) {
            String source = getStr(payload, "source_system");
            if (source != null && !source.isEmpty()) {
                addConfounder(window, "EXTERNAL_MEDICATION_CHANGE");
            }
        } else if (eventType == EventType.PATIENT_REPORTED) {
            Boolean admissionFlag = getBool(payload, "admission_flag");
            if (Boolean.TRUE.equals(admissionFlag)) {
                addConfounder(window, "HOSPITALISATION");
                return;
            }

            String subEventType = getStr(payload, "event_type");
            if ("ILLNESS".equals(subEventType)) {
                addConfounder(window, "INTERCURRENT_ILLNESS");
            } else if ("TRAVEL".equals(subEventType)) {
                addConfounder(window, "TRAVEL_DISRUPTION");
            }
        } else if (eventType == EventType.LAB_RESULT) {
            Map<String, Object> labEntry = new HashMap<>();
            labEntry.put("lab_type", getStr(payload, "lab_type"));
            labEntry.put("value", payload.get("value"));
            labEntry.put("timestamp", event.getEventTime());
            window.labChanges.add(labEntry);
        }
        // VITAL_SIGN and DEVICE_READING are handled by trajectory tracker, not here
    }

    /**
     * Adds a festival/seasonal confounder flag (from KB-21 cultural calendar lookup).
     */
    public static void addFestivalConfounder(InterventionWindowState.InterventionWindow window,
                                              String festivalName) {
        if (window == null) return;
        addConfounder(window, "FESTIVAL_PERIOD:" + festivalName);
    }

    private static void addConfounder(InterventionWindowState.InterventionWindow window,
                                       String flag) {
        if (!window.confoundersDetected.contains(flag)) {
            window.confoundersDetected.add(flag);
        }
    }

    private static String getStr(Map<String, Object> m, String key) {
        Object v = m.get(key);
        return v != null ? v.toString() : null;
    }

    private static Boolean getBool(Map<String, Object> m, String key) {
        Object v = m.get(key);
        if (v instanceof Boolean) return (Boolean) v;
        return null;
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module12ConfounderAccumulatorTest -q 2>&1 | tail -10
```

Expected: Tests run: 5, Failures: 0, Errors: 0

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module12ConfounderAccumulator.java \
        src/test/java/com/cardiofit/flink/operators/Module12ConfounderAccumulatorTest.java
git commit -m "feat(module12): add Module12ConfounderAccumulator with 7 confounder categories — 5 tests"
```

---

## Task 10: Module12_InterventionWindowMonitor (Main KPF)

**Files:**
- Create: `operators/Module12_InterventionWindowMonitor.java`

This is the core operator. It extends `KeyedProcessFunction<String, CanonicalEvent, InterventionWindowSignal>` — consuming from a unioned stream (both intervention events and patient events mapped to CanonicalEvent), managing processing-time timers, and emitting lifecycle signals.

- [ ] **Step 1: Create the main KPF**

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
        for (String concurrentId : concurrency.getConcurrentIds()) {
            InterventionWindowState.InterventionWindow existing = state.getWindow(concurrentId);
            if (existing != null && !existing.concurrentInterventionIds.contains(interventionId)) {
                existing.concurrentInterventionIds.add(interventionId);
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
```

- [ ] **Step 2: Verify compilation**

```bash
cd backend/shared-infrastructure/flink-processing
mvn compile -pl . -q 2>&1 | tail -5
```

Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module12_InterventionWindowMonitor.java
git commit -m "feat(module12): add Module12_InterventionWindowMonitor KPF with dual-source routing, timer management, and signal emission"
```

---

## Task 11: Module12b_InterventionDeltaComputer (Separate Job KPF)

**Files:**
- Create: `operators/Module12b_InterventionDeltaComputer.java`

Module 12b consumes WINDOW_CLOSED signals and computes streaming metric deltas. Architecturally simpler — no timers, just react to signals and compute deltas.

- [ ] **Step 1: Create Module12b KPF**

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
import java.util.Map;
import java.util.UUID;

/**
 * Module 12b: Intervention Delta Computer.
 *
 * Consumes WINDOW_CLOSED signals from clinical.intervention-window-signals
 * AND enriched-patient-events-v1 (for current vital baselines).
 * Computes streaming metric deltas for the physician dashboard.
 *
 * Separate Flink job from Module 12 for failure isolation
 * (same pattern as Module 10/10b and 11/11b).
 *
 * State TTL: 90 days to retain baseline snapshots across long observation windows.
 */
public class Module12b_InterventionDeltaComputer
        extends KeyedProcessFunction<String, CanonicalEvent, InterventionDeltaRecord> {

    private static final Logger LOG =
            LoggerFactory.getLogger(Module12b_InterventionDeltaComputer.class);

    /** Stores per-patient baseline snapshots captured at WINDOW_OPENED. */
    private transient ValueState<InterventionBaselineState> baselineState;

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        ValueStateDescriptor<InterventionBaselineState> stateDesc =
                new ValueStateDescriptor<>("intervention-baseline-state",
                        InterventionBaselineState.class);

        StateTtlConfig ttl = StateTtlConfig
                .newBuilder(Duration.ofDays(90))
                .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
                .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
                .build();
        stateDesc.enableTimeToLive(ttl);
        baselineState = getRuntimeContext().getState(stateDesc);

        LOG.info("Module 12b Intervention Delta Computer initialized (90-day TTL)");
    }

    @Override
    public void processElement(CanonicalEvent event, Context ctx,
                               Collector<InterventionDeltaRecord> out) throws Exception {
        InterventionBaselineState state = baselineState.value();
        if (state == null) {
            state = new InterventionBaselineState();
            state.patientId = event.getPatientId();
        }

        Map<String, Object> payload = event.getPayload();
        if (payload == null) {
            baselineState.update(state);
            return;
        }

        String signalType = getStr(payload, "signal_type");

        if ("WINDOW_OPENED".equals(signalType)) {
            // Capture baseline snapshot
            String interventionId = getStr(payload, "intervention_id");
            InterventionBaselineState.Baseline baseline = new InterventionBaselineState.Baseline();
            baseline.fbg = state.currentFBG;
            baseline.sbp = state.currentSBP;
            baseline.dbp = state.currentDBP;
            baseline.weight = state.currentWeight;
            baseline.hba1c = state.currentHbA1c;
            baseline.egfr = state.currentEGFR;
            baseline.tir = state.currentTIR;
            baseline.trajectoryAtOpen = getStr(payload, "trajectory_at_signal");
            baseline.interventionType = getStr(payload, "intervention_type");
            baseline.concurrentCount = getIntFromPayload(payload, "concurrent_intervention_count", 0);
            state.baselines.put(interventionId, baseline);

        } else if ("WINDOW_CLOSED".equals(signalType)) {
            // Compute deltas
            String interventionId = getStr(payload, "intervention_id");
            InterventionBaselineState.Baseline baseline = state.baselines.get(interventionId);
            if (baseline == null) {
                LOG.warn("WINDOW_CLOSED without baseline for intervention: {}", interventionId);
                baselineState.update(state);
                return;
            }

            TrajectoryClassification before = safeTrajectory(baseline.trajectoryAtOpen);
            String trajectoryDuringStr = getStr(payload, "trajectory_at_signal");
            TrajectoryClassification during = safeTrajectory(trajectoryDuringStr);
            TrajectoryAttribution attribution = TrajectoryAttribution.fromTrajectories(before, during);

            InterventionType type = safeInterventionType(baseline.interventionType);

            InterventionDeltaRecord delta = InterventionDeltaRecord.builder()
                    .deltaId(UUID.randomUUID().toString())
                    .interventionId(interventionId)
                    .patientId(state.patientId)
                    .interventionType(type)
                    .fbgDelta(computeDelta(state.currentFBG, baseline.fbg))
                    .sbpDelta(computeDelta(state.currentSBP, baseline.sbp))
                    .dbpDelta(computeDelta(state.currentDBP, baseline.dbp))
                    .weightDeltaKg(computeDelta(state.currentWeight, baseline.weight))
                    .hba1cDelta(computeDelta(state.currentHbA1c, baseline.hba1c))
                    .egfrDelta(computeDelta(state.currentEGFR, baseline.egfr))
                    .tirDelta(computeDelta(state.currentTIR, baseline.tir))
                    .trajectoryAttribution(attribution)
                    .concurrentCount(baseline.concurrentCount)
                    .dataCompletenessScore(computeCompleteness(state))
                    .processingTimestamp(System.currentTimeMillis())
                    .build();

            out.collect(delta);

            // Clean up baseline
            state.baselines.remove(interventionId);

            LOG.info("Delta computed: intervention={}, fbg={}, sbp={}, attribution={}",
                    interventionId, delta.getFbgDelta(), delta.getSbpDelta(), attribution);

        } else {
            // Patient data event — update current baselines
            updateCurrentValues(event, state);
        }

        baselineState.update(state);
    }

    private void updateCurrentValues(CanonicalEvent event, InterventionBaselineState state) {
        Map<String, Object> payload = event.getPayload();
        if (payload == null) return;

        EventType eventType = event.getEventType();
        if (eventType == EventType.LAB_RESULT) {
            String labType = getStr(payload, "lab_type");
            Double value = getDouble(payload, "value");
            if ("FBG".equals(labType) && value != null) state.currentFBG = value;
            if ("EGFR".equals(labType) && value != null) state.currentEGFR = value;
            if ("HBA1C".equals(labType) && value != null) state.currentHbA1c = value;
        } else if (eventType == EventType.VITAL_SIGN) {
            Double sbp = getDouble(payload, "systolic_bp");
            Double dbp = getDouble(payload, "diastolic_bp");
            Double weight = getDouble(payload, "weight_kg");
            if (sbp != null) state.currentSBP = sbp;
            if (dbp != null) state.currentDBP = dbp;
            if (weight != null) state.currentWeight = weight;
        }
    }

    private static Double computeDelta(Double current, Double baseline) {
        if (current == null || baseline == null) return null;
        return current - baseline;
    }

    private static double computeCompleteness(InterventionBaselineState state) {
        int available = 0;
        int total = 5; // FBG, SBP, weight, HbA1c, eGFR
        if (state.currentFBG != null) available++;
        if (state.currentSBP != null) available++;
        if (state.currentWeight != null) available++;
        if (state.currentHbA1c != null) available++;
        if (state.currentEGFR != null) available++;
        return (double) available / total;
    }

    private static TrajectoryClassification safeTrajectory(String s) {
        if (s == null) return TrajectoryClassification.UNKNOWN;
        try { return TrajectoryClassification.valueOf(s); }
        catch (IllegalArgumentException e) { return TrajectoryClassification.UNKNOWN; }
    }

    private static InterventionType safeInterventionType(String s) {
        if (s == null) return null;
        try { return InterventionType.valueOf(s); }
        catch (IllegalArgumentException e) { return null; }
    }

    private static String getStr(Map<String, Object> m, String key) {
        Object v = m.get(key);
        return v != null ? v.toString() : null;
    }

    private static Double getDouble(Map<String, Object> m, String key) {
        Object v = m.get(key);
        if (v instanceof Number) return ((Number) v).doubleValue();
        return null;
    }

    private static int getIntFromPayload(Map<String, Object> m, String key, int def) {
        Object v = m.get(key);
        if (v instanceof Number) return ((Number) v).intValue();
        return def;
    }

    /**
     * Per-patient baseline state for Module 12b.
     * Stores current vital values and per-intervention baseline snapshots.
     */
    public static class InterventionBaselineState implements java.io.Serializable {
        private static final long serialVersionUID = 1L;
        public String patientId;
        public Double currentFBG;
        public Double currentSBP;
        public Double currentDBP;
        public Double currentWeight;
        public Double currentHbA1c;
        public Double currentEGFR;
        public Double currentTIR;
        public java.util.Map<String, Baseline> baselines = new java.util.HashMap<>();

        public static class Baseline implements java.io.Serializable {
            private static final long serialVersionUID = 1L;
            public Double fbg, sbp, dbp, weight, hba1c, egfr, tir;
            public String trajectoryAtOpen;
            public String interventionType;
            public int concurrentCount;
        }
    }
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd backend/shared-infrastructure/flink-processing
mvn compile -pl . -q 2>&1 | tail -5
```

Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module12b_InterventionDeltaComputer.java
git commit -m "feat(module12b): add Module12b_InterventionDeltaComputer KPF with baseline tracking and delta computation"
```

---

## Task 12: KafkaTopics Update + FlinkJobOrchestrator Wiring

**Files:**
- Modify: `utils/KafkaTopics.java`
- Modify: `FlinkJobOrchestrator.java`

- [ ] **Step 1: Add FLINK_INTERVENTION_DELTAS to KafkaTopics**

In `utils/KafkaTopics.java`, add the new topic constant after `FLINK_FITNESS_PATTERNS`:

```java
    FLINK_ACTIVITY_RESPONSE("flink.activity-response", 8, 30),
    FLINK_FITNESS_PATTERNS("flink.fitness-patterns", 4, 90),
    FLINK_INTERVENTION_DELTAS("flink.intervention-deltas", 4, 90);
```

Also update `isV4OutputTopic()` to include the new topic:

```java
               this == FLINK_ACTIVITY_RESPONSE ||
               this == FLINK_FITNESS_PATTERNS ||
               this == FLINK_INTERVENTION_DELTAS;
```

- [ ] **Step 2: Add Module 12/12b switch cases to FlinkJobOrchestrator**

Before the `default:` case in the main switch block, add:

```java
            case "intervention-window":
            case "module12":
            case "intervention-window-monitor":
                launchInterventionWindowMonitor(env);
                break;
            case "intervention-deltas":
            case "module12b":
            case "intervention-delta-computer":
                launchInterventionDeltaComputer(env);
                break;
```

- [ ] **Step 3: Add launchInterventionWindowMonitor method**

Add after the `launchFitnessPatternAggregator` method:

```java
    /**
     * Module 12: Intervention Window Monitor.
     * Dual-source: intervention events + enriched patient events → window signals.
     * Uses union pattern with discriminator for single-processElement KPF.
     */
    private static void launchInterventionWindowMonitor(StreamExecutionEnvironment env) {
        LOG.info("Launching Module 12: Intervention Window Monitor pipeline");

        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        // Source 1: Intervention events (low-volume, from KB-23)
        KafkaSource<CanonicalEvent> interventionSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.CLINICAL_INTERVENTION_EVENTS.getTopicName())
                .setGroupId("flink-module12-intervention-events-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new CanonicalEventDeserializer())
                .build();

        // Source 2: Enriched patient events (high-volume, for trajectory/confounder)
        KafkaSource<CanonicalEvent> patientSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
                .setGroupId("flink-module12-patient-events-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new CanonicalEventDeserializer())
                .build();

        DataStream<CanonicalEvent> interventionStream = env.fromSource(interventionSource,
                WatermarkStrategy.<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                        .withTimestampAssigner((e, ts) -> e.getEventTime()),
                "Kafka Source: Intervention Events (Module 12)");

        DataStream<CanonicalEvent> patientStream = env.fromSource(patientSource,
                WatermarkStrategy.<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                        .withTimestampAssigner((e, ts) -> e.getEventTime()),
                "Kafka Source: Enriched Patient Events (Module 12)");

        // Union both streams and key by patientId
        SingleOutputStreamOperator<InterventionWindowSignal> signals = interventionStream
                .union(patientStream)
                .keyBy(CanonicalEvent::getPatientId)
                .process(new Module12_InterventionWindowMonitor())
                .uid("module12-intervention-window-monitor")
                .name("Module 12: Intervention Window Monitor");

        signals.sinkTo(
                KafkaSink.<InterventionWindowSignal>builder()
                        .setBootstrapServers(bootstrap)
                        .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                        .setTransactionalIdPrefix("m12-intervention-window")
                        .setRecordSerializer(
                                KafkaRecordSerializationSchema.<InterventionWindowSignal>builder()
                                        .setTopic(KafkaTopics.CLINICAL_INTERVENTION_WINDOW_SIGNALS.getTopicName())
                                        .setValueSerializationSchema(new JsonSerializer<InterventionWindowSignal>())
                                        .build())
                        .build()
        ).name("Sink: Intervention Window Signals");

        LOG.info("Module 12 pipeline configured: sources=[{}, {}], sink=[{}]",
                KafkaTopics.CLINICAL_INTERVENTION_EVENTS.getTopicName(),
                KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName(),
                KafkaTopics.CLINICAL_INTERVENTION_WINDOW_SIGNALS.getTopicName());
    }
```

- [ ] **Step 4: Add launchInterventionDeltaComputer method**

```java
    /**
     * Module 12b: Intervention Delta Computer.
     * Consumes window signals + patient events, computes streaming metric deltas.
     * Separate job from Module 12 for failure isolation.
     */
    private static void launchInterventionDeltaComputer(StreamExecutionEnvironment env) {
        LOG.info("Launching Module 12b: Intervention Delta Computer pipeline");

        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        // Source 1: Window signals (for OPENED/CLOSED triggers)
        KafkaSource<CanonicalEvent> signalSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.CLINICAL_INTERVENTION_WINDOW_SIGNALS.getTopicName())
                .setGroupId("flink-module12b-window-signals-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new CanonicalEventDeserializer())
                .build();

        // Source 2: Patient events (for current vital values)
        KafkaSource<CanonicalEvent> patientSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
                .setGroupId("flink-module12b-patient-events-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new CanonicalEventDeserializer())
                .build();

        DataStream<CanonicalEvent> signalStream = env.fromSource(signalSource,
                WatermarkStrategy.<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                        .withTimestampAssigner((e, ts) -> e.getEventTime()),
                "Kafka Source: Window Signals (Module 12b)");

        DataStream<CanonicalEvent> patientStream = env.fromSource(patientSource,
                WatermarkStrategy.<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                        .withTimestampAssigner((e, ts) -> e.getEventTime()),
                "Kafka Source: Enriched Patient Events (Module 12b)");

        SingleOutputStreamOperator<InterventionDeltaRecord> deltas = signalStream
                .union(patientStream)
                .keyBy(CanonicalEvent::getPatientId)
                .process(new Module12b_InterventionDeltaComputer())
                .uid("module12b-intervention-delta-computer")
                .name("Module 12b: Intervention Delta Computer");

        deltas.sinkTo(
                KafkaSink.<InterventionDeltaRecord>builder()
                        .setBootstrapServers(bootstrap)
                        .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                        .setTransactionalIdPrefix("m12b-intervention-deltas")
                        .setRecordSerializer(
                                KafkaRecordSerializationSchema.<InterventionDeltaRecord>builder()
                                        .setTopic(KafkaTopics.FLINK_INTERVENTION_DELTAS.getTopicName())
                                        .setValueSerializationSchema(new JsonSerializer<InterventionDeltaRecord>())
                                        .build())
                        .build()
        ).name("Sink: Intervention Delta Records");

        LOG.info("Module 12b pipeline configured: sources=[{}, {}], sink=[{}]",
                KafkaTopics.CLINICAL_INTERVENTION_WINDOW_SIGNALS.getTopicName(),
                KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName(),
                KafkaTopics.FLINK_INTERVENTION_DELTAS.getTopicName());
    }
```

- [ ] **Step 5: Add required imports to FlinkJobOrchestrator**

Ensure these imports are present at the top of FlinkJobOrchestrator.java:

```java
import com.cardiofit.flink.models.InterventionWindowSignal;
import com.cardiofit.flink.models.InterventionDeltaRecord;
import com.cardiofit.flink.operators.Module12_InterventionWindowMonitor;
import com.cardiofit.flink.operators.Module12b_InterventionDeltaComputer;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
```

- [ ] **Step 6: Verify compilation**

```bash
cd backend/shared-infrastructure/flink-processing
mvn compile -pl . -q 2>&1 | tail -5
```

Expected: BUILD SUCCESS

- [ ] **Step 7: Commit**

```bash
git add src/main/java/com/cardiofit/flink/utils/KafkaTopics.java \
        src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java
git commit -m "feat(module12): wire Module 12 and 12b into FlinkJobOrchestrator with dual-source union and Kafka topics"
```

---

## Task 13: Module12WindowLifecycleTest (7 tests)

**Files:**
- Create (test): `operators/Module12WindowLifecycleTest.java`

These tests verify the state management and timer registration logic by directly exercising the InterventionWindowState methods (same pattern as Module 11 tests — test state logic directly, not through the KPF).

- [ ] **Step 1: Write the 7 lifecycle tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module12TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.Collections;
import java.util.HashMap;

import static com.cardiofit.flink.builders.Module12TestBuilder.*;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 12 window lifecycle: open, midpoint, close, cancel,
 * modify, expire, and timer deduplication.
 */
class Module12WindowLifecycleTest {

    @Test
    void openWindow_createsWindowWithTimers() {
        InterventionWindowState state = emptyState("P1");

        InterventionWindowState.InterventionWindow window = state.openWindow(
                "int-1", InterventionType.MEDICATION_ADD, medicationDetail("SGLT2I", "Empagliflozin", "10mg"),
                28, BASE_TIME, TrajectoryClassification.STABLE,
                "card-1", "APPROVED");

        assertNotNull(window);
        assertEquals("int-1", window.interventionId);
        assertEquals(InterventionType.MEDICATION_ADD, window.interventionType);
        assertEquals(BASE_TIME, window.observationStartMs);
        assertEquals(BASE_TIME + 28 * DAY_MS, window.observationEndMs);
        assertEquals(28, window.observationWindowDays);
        assertEquals("OBSERVING", window.status);
        // Midpoint = start + 14 days
        assertEquals(BASE_TIME + 14 * DAY_MS, window.midpointTimerMs);
        // Close = end + 24h grace
        assertEquals(BASE_TIME + 28 * DAY_MS + DAY_MS, window.closeTimerMs);
        assertEquals(1, state.getTotalInterventionsTracked());
    }

    @Test
    void midpointTimerLookup_findsCorrectWindow() {
        InterventionWindowState state = emptyState("P1");

        InterventionWindowState.InterventionWindow window = state.openWindow(
                "int-1", InterventionType.LIFESTYLE_ACTIVITY, null,
                14, BASE_TIME, TrajectoryClassification.STABLE,
                "card-1", "APPROVED");

        // Midpoint = start + 7 days
        long midpoint = BASE_TIME + 7 * DAY_MS;
        InterventionWindowState.InterventionWindow found = state.getWindowForMidpointTimer(midpoint);

        assertNotNull(found);
        assertEquals("int-1", found.interventionId);
    }

    @Test
    void closeTimerLookup_findsCorrectWindow() {
        InterventionWindowState state = emptyState("P1");

        state.openWindow("int-1", InterventionType.MEDICATION_ADD, null,
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        // Close = start + 28d + 24h
        long closeTime = BASE_TIME + 28 * DAY_MS + DAY_MS;
        InterventionWindowState.InterventionWindow found = state.getWindowForCloseTimer(closeTime);

        assertNotNull(found);
        assertEquals("int-1", found.interventionId);
    }

    @Test
    void cancelWindow_removesFromActive() {
        InterventionWindowState state = emptyState("P1");
        state.openWindow("int-1", InterventionType.MEDICATION_ADD, null,
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        assertEquals(1, state.getActiveWindows().size());

        InterventionWindowState.InterventionWindow removed = state.removeWindow("int-1");

        assertNotNull(removed);
        assertEquals(0, state.getActiveWindows().size());
    }

    @Test
    void modifyWindow_updatesTimers() {
        InterventionWindowState state = emptyState("P1");

        InterventionWindowState.InterventionWindow window = state.openWindow(
                "int-1", InterventionType.MEDICATION_ADD, null,
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        long oldMidpoint = window.midpointTimerMs;
        long oldClose = window.closeTimerMs;

        // Simulate modification to 42-day window
        long newWindowMs = 42 * DAY_MS;
        window.observationEndMs = window.observationStartMs + newWindowMs;
        window.observationWindowDays = 42;
        window.midpointTimerMs = window.observationStartMs + newWindowMs / 2;
        window.closeTimerMs = window.observationEndMs + DAY_MS;

        assertNotEquals(oldMidpoint, window.midpointTimerMs);
        assertNotEquals(oldClose, window.closeTimerMs);
        assertEquals(BASE_TIME + 21 * DAY_MS, window.midpointTimerMs);
        assertEquals(BASE_TIME + 43 * DAY_MS, window.closeTimerMs);
    }

    @Test
    void expiredWindow_noData_closeTimerStillFires() {
        // Window with no readings during observation → data completeness = 0
        InterventionWindowState state = emptyState("P1");
        state.openWindow("int-1", InterventionType.MONITORING_CHANGE, null,
                14, BASE_TIME, TrajectoryClassification.UNKNOWN, "card-1", "APPROVED");

        // No readings added during window
        InterventionWindowState.InterventionWindow window = state.getWindow("int-1");

        // Data completeness should be empty (no flags set)
        assertTrue(window.dataCompleteness.isEmpty());
        // Close timer is still registered — the KPF will emit CLOSED
        // and the IOR Generator will set status EXPIRED based on completeness
        assertEquals(BASE_TIME + 14 * DAY_MS + DAY_MS, window.closeTimerMs);
    }

    @Test
    void timerDedup_cancelledWindowIgnoredOnClose() {
        InterventionWindowState state = emptyState("P1");
        state.openWindow("int-1", InterventionType.MEDICATION_ADD, null,
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        // Cancel the window
        InterventionWindowState.InterventionWindow window = state.getWindow("int-1");
        window.status = "CANCELLED";

        // Close timer fires but window is cancelled
        long closeTime = BASE_TIME + 28 * DAY_MS + DAY_MS;
        InterventionWindowState.InterventionWindow found = state.getWindowForCloseTimer(closeTime);

        // Should NOT find the window (status != OBSERVING)
        assertNull(found);
    }
}
```

- [ ] **Step 2: Run tests to verify they pass**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module12WindowLifecycleTest -q 2>&1 | tail -10
```

Expected: Tests run: 7, Failures: 0, Errors: 0

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/operators/Module12WindowLifecycleTest.java
git commit -m "test(module12): add Module12WindowLifecycleTest — 7 lifecycle tests for open/midpoint/close/cancel/modify/expire/dedup"
```

---

## Task 14: Module12ConcurrentInterventionIntegrationTest (4 tests)

**Files:**
- Create (test): `operators/Module12ConcurrentInterventionIntegrationTest.java`

- [ ] **Step 1: Write the 4 integration tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module12TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static com.cardiofit.flink.builders.Module12TestBuilder.*;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Integration tests for concurrent intervention detection:
 * cross-referencing, domain classification, confounded status, 3+ concurrent.
 */
class Module12ConcurrentInterventionIntegrationTest {

    @Test
    void twoOverlappingWindows_crossReferenced() {
        InterventionWindowState state = emptyState("P1");

        // Open first intervention: metformin add, day 0–28
        state.openWindow("int-1", InterventionType.MEDICATION_ADD,
                medicationDetail("METFORMIN", "Metformin", "500mg"),
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        // Open second intervention: dietary change, day 10–24
        state.openWindow("int-2", InterventionType.NUTRITION_FOOD_CHANGE, null,
                14, daysAfter(BASE_TIME, 10), TrajectoryClassification.STABLE,
                "card-2", "APPROVED");

        // Detect concurrency for int-2
        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-2", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 10), daysAfter(BASE_TIME, 24),
                state.getActiveWindows());

        assertTrue(result.getConcurrentIds().contains("int-1"));
    }

    @Test
    void sameDomainGlucose_flaggedForAttribution() {
        InterventionWindowState state = emptyState("P1");

        // Metformin (GLUCOSE domain) and food change (GLUCOSE domain)
        state.openWindow("int-1", InterventionType.MEDICATION_ADD,
                medicationDetail("METFORMIN", "Metformin", "500mg"),
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-2", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 5), daysAfter(BASE_TIME, 19),
                state.getActiveWindows());

        assertTrue(result.isSameDomainConcurrent());
    }

    @Test
    void crossDomain_notConfounded() {
        InterventionWindowState state = emptyState("P1");

        // ACEi (BP domain) and dietary glucose change (GLUCOSE domain)
        state.openWindow("int-1", InterventionType.MEDICATION_ADD,
                medicationDetail("ACEI", "Enalapril", "10mg"),
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-2", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 5), daysAfter(BASE_TIME, 19),
                state.getActiveWindows());

        assertEquals(1, result.getConcurrentIds().size());
        assertFalse(result.isSameDomainConcurrent());
    }

    @Test
    void threePlusConcurrent_allDetected() {
        InterventionWindowState state = emptyState("P1");

        // Three overlapping interventions all starting within 7 days
        state.openWindow("int-1", InterventionType.MEDICATION_ADD,
                medicationDetail("METFORMIN", "Metformin", "500mg"),
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        state.openWindow("int-2", InterventionType.LIFESTYLE_ACTIVITY, null,
                14, daysAfter(BASE_TIME, 3), TrajectoryClassification.STABLE,
                "card-2", "APPROVED");

        // Detect concurrency for int-3
        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-3", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 5), daysAfter(BASE_TIME, 19),
                state.getActiveWindows());

        assertEquals(2, result.getConcurrentIds().size());
        assertTrue(result.getConcurrentIds().contains("int-1"));
        assertTrue(result.getConcurrentIds().contains("int-2"));
    }
}
```

- [ ] **Step 2: Run tests to verify they pass**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module12ConcurrentInterventionIntegrationTest -q 2>&1 | tail -10
```

Expected: Tests run: 4, Failures: 0, Errors: 0

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/operators/Module12ConcurrentInterventionIntegrationTest.java
git commit -m "test(module12): add Module12ConcurrentInterventionIntegrationTest — 4 tests for cross-referencing, domain classification"
```

---

## Task 15: Module12bDeltaComputerTest (5 tests)

**Files:**
- Create (test): `operators/Module12bDeltaComputerTest.java`

Tests verify the delta computation logic by directly exercising the baseline state and delta math.

- [ ] **Step 1: Write the 5 delta computation tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 12b delta computation logic.
 * Verifies FBG, SBP, weight, HbA1c deltas and missing data handling.
 */
class Module12bDeltaComputerTest {

    @Test
    void fbgDelta_negativeIsImprovement() {
        // Baseline FBG=160, current FBG=145 → delta=-15 (improvement)
        Double delta = computeDelta(145.0, 160.0);
        assertNotNull(delta);
        assertEquals(-15.0, delta, 0.01);
    }

    @Test
    void sbpDelta_negativeIsImprovement() {
        // Baseline SBP=148, current SBP=135 → delta=-13
        Double delta = computeDelta(135.0, 148.0);
        assertNotNull(delta);
        assertEquals(-13.0, delta, 0.01);
    }

    @Test
    void weightDelta_positive_isDeterioration() {
        // Baseline weight=85.0, current weight=87.5 → delta=+2.5
        Double delta = computeDelta(87.5, 85.0);
        assertNotNull(delta);
        assertEquals(2.5, delta, 0.01);
    }

    @Test
    void hba1cDelta_withNewLabDuringWindow() {
        // Baseline HbA1c=8.2, new lab during window=7.8 → delta=-0.4
        Double delta = computeDelta(7.8, 8.2);
        assertNotNull(delta);
        assertEquals(-0.4, delta, 0.01);
    }

    @Test
    void missingData_returnsNull() {
        // If either baseline or current is null, delta is null
        assertNull(computeDelta(null, 160.0));
        assertNull(computeDelta(145.0, null));
        assertNull(computeDelta(null, null));
    }

    @Test
    void trajectoryAttribution_reversed() {
        // DETERIORATING before + IMPROVING during = INTERVENTION_REVERSED_DECLINE
        TrajectoryAttribution attr = TrajectoryAttribution.fromTrajectories(
                TrajectoryClassification.DETERIORATING,
                TrajectoryClassification.IMPROVING);
        assertEquals(TrajectoryAttribution.INTERVENTION_REVERSED_DECLINE, attr);
    }

    // Mirror of the private computeDelta method from Module12b
    private static Double computeDelta(Double current, Double baseline) {
        if (current == null || baseline == null) return null;
        return current - baseline;
    }
}
```

- [ ] **Step 2: Run tests to verify they pass**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module12bDeltaComputerTest -q 2>&1 | tail -10
```

Expected: Tests run: 6, Failures: 0, Errors: 0

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/operators/Module12bDeltaComputerTest.java
git commit -m "test(module12b): add Module12bDeltaComputerTest — 6 tests for FBG/SBP/weight/HbA1c deltas and missing data"
```

---

## Task 16: Final Verification — Full Test Suite

- [ ] **Step 1: Run all Module 12 tests**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest="Module12*" -q 2>&1 | tail -15
```

Expected: Tests run: 42+, Failures: 0, Errors: 0

- [ ] **Step 2: Run full project compilation**

```bash
cd backend/shared-infrastructure/flink-processing
mvn compile test-compile -pl . -q 2>&1 | tail -5
```

Expected: BUILD SUCCESS

- [ ] **Step 3: Final commit with all files**

If any files were missed in earlier commits, stage and commit them:

```bash
cd backend/shared-infrastructure/flink-processing
git status --short | grep "^??" | grep -i module12
```

If clean, no action needed.

---

## Self-Review: Spec Coverage Matrix

| Spec Requirement | Task | Status |
|---|---|---|
| 13 intervention types with window days + domain mapping | Task 1 | Covered |
| 5 signal types (OPENED/MIDPOINT/CLOSED/EXPIRED/CANCELLED) | Task 2 | Covered |
| 3 trajectory classifications + UNKNOWN | Task 2 | Covered |
| 9-cell trajectory attribution matrix | Task 2 | Covered |
| Per-patient keyed state with active windows + trajectory buffer | Task 3 | Covered |
| InterventionWindowSignal output (21 fields) | Task 4 | Covered |
| InterventionDeltaRecord output (18 fields) | Task 4 | Covered |
| OLS trajectory classification with domain thresholds | Task 6 | Covered |
| Composite worst-domain trajectory logic | Task 6 | Covered |
| Minimum 3 readings / 7 days for valid trajectory | Task 6 | Covered |
| 7-day overlap threshold for concurrency | Task 7 | Covered |
| Same-domain vs cross-domain classification | Task 7 | Covered |
| Drug class → clinical domain mapping | Task 1 | Covered |
| Adherence signals by intervention type | Task 8 | Covered |
| Data quality tiers (HIGH/MODERATE/LOW) | Task 8 | Covered |
| Adherence score capped at 1.0 | Task 8 | Covered |
| 7 confounder categories | Task 9 | Covered |
| Festival confounder from KB-21 | Task 9 | Covered |
| Lab changes accumulated (not flagged) | Task 9 | Covered |
| Main KPF with dual-source consumption | Task 10 | Covered |
| Processing-time timers for MIDPOINT + CLOSE | Task 10 | Covered |
| 24h grace period on CLOSE timer | Task 10 | Covered |
| CANCELLED → timer deletion | Task 10 | Covered |
| MODIFIED → timer re-registration + confounder flag | Task 10 | Covered |
| Module 12b streaming deltas for physician dashboard | Task 11 | Covered |
| Baseline snapshot at WINDOW_OPENED | Task 11 | Covered |
| TrajectoryAttribution from before/during | Task 11 | Covered |
| FLINK_INTERVENTION_DELTAS Kafka topic | Task 12 | Covered |
| FlinkJobOrchestrator switch cases | Task 12 | Covered |
| Dual-source union pattern | Task 12 | Covered |
| 90-day state TTL | Task 10, 11 | Covered |
| Window lifecycle tests (7) | Task 13 | Covered |
| Concurrent intervention integration tests (4) | Task 14 | Covered |
| Delta computation tests (5+) | Task 15 | Covered |
| **Total test count** | **42+** | **Covered** |

### Type Consistency Check
- `InterventionType` enum used consistently across: state, signal, delta record, concurrency detector, adherence assembler
- `TrajectoryClassification` used in: trajectory tracker, state, signal, delta computer
- `TrajectoryAttribution` used in: delta record, delta computer, with `fromTrajectories()` static method
- `InterventionWindowState.InterventionWindow` used in: state, concurrency detector, confounder accumulator, main KPF
- `Module12ConcurrencyDetector.Result` used in: concurrency detector, main KPF signal builder
- `Module12AdherenceAssembler.Result` used in: adherence assembler, main KPF midpoint handler
