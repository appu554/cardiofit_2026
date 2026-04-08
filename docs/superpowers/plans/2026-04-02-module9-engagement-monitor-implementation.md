# Module 9: Engagement Monitor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Module 9, the Engagement Monitor, as a `KeyedProcessFunction<String, CanonicalEvent, EngagementSignal>` that produces daily per-patient engagement composite scores, detects engagement drops, and (Phase 2) computes early relapse prediction trajectory features for SPAN micro-engagement integration. Module 9 is fundamentally **timer-driven** — its most important output occurs on daily processing-time timers, not on incoming events.

**Architecture:** Module 9 is keyed by `patientId`, consuming events from `enriched-patient-events-v1`. Unlike Modules 7 and 8 (which produce output per-event), Module 9 accumulates signal bitmaps throughout the day and scores **once per day** via `onTimer()` at 23:59 UTC. This prevents CGM patients (288 readings/day) from producing 288 engagement scores. Eight boolean signal channels over a 14-day rolling window produce a weighted composite score. Engagement drop alerts require **3-day persistence** before emission (DD#8 Section 4.1), and thresholds are **channel-aware** (Corporate/Government/ACCHS).

**Tech Stack:** Java 17, Flink 2.1.0, Jackson 2.17, Kafka (Confluent Cloud), JUnit 5

---

## Pre-Conditions Verified (2026-04-02)

The following were confirmed against the actual codebase before finalizing this plan:

| Check | Status | Detail |
|---|---|---|
| Java version | **17** | `maven.compiler.source=17` in pom.xml |
| `CanonicalEvent.java` API | **Confirmed** | `getPatientId()`, `getEventType()` (returns `EventType` enum), `getPayload()` (Map), `getEventTime()` (long), `getCorrelationId()` |
| `EventType` enum | **Has needed types** | `VITAL_SIGN`, `LAB_RESULT`, `MEDICATION_ORDERED`, `MEDICATION_ADMINISTERED`, `MEDICATION_MISSED`, `PATIENT_REPORTED`, `DEVICE_READING`, `ENCOUNTER_START`, `TASK_COMPLETED` |
| `KafkaTopics` constants | **Two exist, one missing** | `FLINK_ENGAGEMENT_SIGNALS("flink.engagement-signals", 4, 30)` and `ALERTS_ENGAGEMENT_DROP("alerts.engagement-drop", 2, 90)` exist. **`ALERTS_RELAPSE_RISK` must be added** (Phase 2). |
| `MHRIScore.java` | **Has engagement slot** | `engagementComponent` field (Double), weighted 0.15 across all tiers. Module 9 output fills this field. |
| Orchestrator | **No Module 9 wiring** | `FlinkJobOrchestrator.java` has no `case "module9"` or `case "engagement"`. Must be added. |
| Consumer group naming | **Convention** | `vaidshala-v4-{module-name}-{function}` per architecture doc |
| State TTL | **14 days** | Architecture doc specifies 14-day TTL for Module 9 engagement state |
| Checkpointing | **30s / RocksDB / 3 retained** | Same as Modules 7-11 per architecture doc |
| KB-21 engagement service | **Exists** | Go service at `kb-21-behavioral-intelligence/internal/services/engagement_service.go` — 6 signal composite, phenotyping, loop trust. Module 9 mirrors this logic in Flink for real-time streaming. |
| Flink 2.x `open()` signature | **OpenContext, NOT Configuration** | Confirmed: Module 6 and Module 8 both use `open(org.apache.flink.api.common.functions.OpenContext openContext)`. The `Configuration` overload is deprecated in Flink 2.x. |
| `data_tier` in payload | **Present** | Module 1b canonicalizer sets `payload.put("data_tier", "TIER_1_CGM"/"TIER_2_HYBRID"/"TIER_3_SMBG")`. Available for Module 9 to extract. |
| `CanonicalEvent` channel field | **NOT present** | No `channel` field in CanonicalEvent. Channel must come from KB-20 patient profile via broadcast state (Phase 1b) or default to "CORPORATE" initially. |

### Why Module 9 is Timer-Driven, Not Event-Driven

**Critical design decision.** Modules 7 and 8 produce output per incoming event:
- Module 7: Each BP reading produces variability metrics
- Module 8: Each CanonicalEvent triggers CID rule evaluation

Module 9 **must not** follow this pattern because:

1. **CGM patients send 288 readings/day.** Event-driven scoring would emit 288 engagement scores per patient per day. Timer-driven: exactly 1.
2. **Engagement is about frequency across time, not individual events.** A single meal log tells you nothing about engagement. Whether 10 or 12 events arrived in a day both indicate "engaged."
3. **The most important signal is silence.** A patient who stops engaging generates NO events. Event-driven Module 9 would produce no output for them — the exact failure mode causing 49% loss-to-follow-up. The daily timer fires regardless of whether events arrived.

**Pattern:** `processElement()` classifies events into 8 signal channels and sets today's bitmap bit. It also extracts `data_tier` from the first event's payload. `onTimer()` (daily at 23:59 UTC processing time) computes the composite score and emits output.

### eventType Mapping Note (Same as Module 8)

`CanonicalEvent.getEventType()` returns the `EventType` **enum**, not a String. All signal classification must use:
```java
EventType type = event.getEventType();
if (type == EventType.MEDICATION_ADMINISTERED || type == EventType.MEDICATION_MISSED) { ... }
if (type == EventType.VITAL_SIGN) { ... }
if (type == EventType.PATIENT_REPORTED) { ... }
```

### Flink 2.x `open()` Signature (R5 Fix)

**Confirmed against Module 6 and Module 8 on disk:** Flink 2.1.0 uses `OpenContext`, NOT `Configuration`:
```java
@Override
public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
    super.open(openContext);
    // state initialization here
}
```
The `Configuration` overload compiles but is deprecated and may not initialize state correctly. All Module 9 code must use `OpenContext`.

---

## CRITICAL: Eight-Signal Engagement Model (DD#8 Reconciled)

### Signal Set Reconciliation (DD#8 vs KB-21)

Two different signal sets exist across design documents. **DD#8 is authoritative for clinical correctness.** KB-21's Go implementation is a convenience layer that may diverge. This reconciliation is documented here to prevent the same class of problem as the CID numbering divergence in Module 8.

| Signal | DD#8 Spec (Weight) | KB-21 Go Service | Reconciled (This Plan) | Rationale |
|--------|-------------------|-----------------|----------------------|-----------|
| Glucose monitoring | **0.25** | Not explicit | **S1: 0.20** | Highest clinical importance — glucose cessation is strongest disengagement predictor. Weight adjusted from 0.25 to 0.20 to accommodate 8 signals. |
| BP reading | **0.25** | 0.30 (interaction_freq) | **S2: 0.20** | Second-highest clinical importance. V-MCU titration requires both glucose + BP. |
| Medication reminder ack | **0.15** | 0.40 (adherence_7d) | **S3: 0.15** | DD#8 weight preserved exactly. |
| Meal logging | **0.20** | Implicit in quality | **S4: 0.15** | Slightly reduced from DD#8's 0.20 to fit 8-signal model. |
| App session | **0.10** | 0.10 (latency) | **S5: 0.05** | Lowest weight — may not have instrumentation yet. |
| Weight measurement | **0.05** | Not explicit | **S6: 0.05** | DD#8's weight preserved. Weekly engagement signal. |
| Goal completion | Not in DD#8 | Implicit | **S7: 0.10** | Added from V4 architecture spec. Behavioral engagement. |
| Appointment attendance | Not in DD#8 | Implicit | **S8: 0.10** | Added from V4 architecture spec. Healthcare system engagement. |

**Weights sum to 1.00.** DD#8 signals (S1-S6) contribute 0.80 total weight. V4-added signals (S7-S8) contribute 0.20.

**Key decision: Glucose monitoring (S1) is restored as the most important signal** (tied with BP at 0.20). DD#8 Section 1 identifies glucose monitoring cessation as the strongest predictor of full disengagement in the IHCI 49% loss-to-follow-up analysis. A patient who logs meals and takes medications but never checks glucose is clinically blind — the MHRI engagement component specifically lists "glucose data density" as a primary input.

### Signal Definitions (8 Signals)

| # | Signal | Weight | Event Source | Classification Logic |
|---|--------|--------|-------------|---------------------|
| S1 | Glucose Monitoring | 0.20 | `LAB_RESULT` with glucose/fbg/cgm lab_type, or `DEVICE_READING` with cgm data | Did the patient record at least one glucose reading today? |
| S2 | BP Measurement | 0.20 | `VITAL_SIGN` with `systolic_bp` in payload, or `DEVICE_READING` with BP | Did the patient record a BP reading today? |
| S3 | Medication Adherence | 0.15 | `MEDICATION_ADMINISTERED` | Did the patient take at least one scheduled medication today? |
| S4 | Meal Logging | 0.15 | `PATIENT_REPORTED` with `report_type`="MEAL_LOG" | Did the patient log at least one meal today? |
| S5 | App Session | 0.05 | `PATIENT_REPORTED` with `report_type`="APP_SESSION" | Did the patient interact with the app today? |
| S6 | Weight Measurement | 0.05 | `VITAL_SIGN` with `weight` in payload | Did the patient record a weight today? (Expected weekly, not daily) |
| S7 | Goal Completion | 0.10 | `PATIENT_REPORTED` with `report_type`="GOAL_COMPLETED" | Did the patient complete at least one goal today? |
| S8 | Appointment Attendance | 0.10 | `ENCOUNTER_START` or `TASK_COMPLETED` with appointment context | Did the patient attend an appointment today? |

### Signal → Bitmap Design

Each signal channel stores a `boolean[14]` (14-day rolling window). Index 0 = today, index 13 = 14 days ago. On each daily timer tick, all 8 arrays shift **right** (drop index 13, insert today at index 0).

**Density for signal S = count(trues in S.bitmap) / 14**

**Composite engagement score = SUM(density_i * weight_i) for i in S1..S8**

Result: 0.0 (completely disengaged) to 1.0 (perfect engagement across all 8 channels for 14 consecutive days).

**Note on bitmap shift direction (R4 Fix):** `System.arraycopy(bitmap, 0, bitmap, 1, 13)` shifts elements right (positions 0-12 → positions 1-13), dropping position 13 (oldest). Today's value is written to position 0. The comment must say "shift right" not "shift left."

### Engagement Level Classification — Channel-Aware (R2 Fix)

DD#8 specifies different threshold sets per deployment channel. A Government-tier patient at 0.35 is engaging within structural constraints and should NOT generate an alert. Universal thresholds would produce massive false-positive alerts for Government and ACCHS patients.

**Channel threshold matrix:**

| Level | Corporate | Government | ACCHS |
|-------|-----------|------------|-------|
| GREEN (HIGH) | >= 0.70 | >= 0.40 | >= 0.35 |
| YELLOW (MODERATE) | 0.40 - 0.69 | 0.25 - 0.39 | 0.20 - 0.34 |
| ORANGE (LOW) | 0.20 - 0.39 | 0.15 - 0.24 | 0.10 - 0.19 |
| RED (CRITICAL) | < 0.20 | < 0.15 | < 0.10 |

**SPAN action mapping** (same semantics, channel-adjusted thresholds):
- GREEN → Growth micro-challenges, 1-2/day
- YELLOW → Maintenance nudges, optimal timing
- ORANGE → Re-engagement micro-commitments, max 1/2 days (alert after 3-day persistence)
- RED → SPAN pauses after 3 days (alert immediately)

**Phase 1 implementation:** `EngagementLevel.fromScore(double score, String channel)` with hardcoded thresholds. Channel defaults to "CORPORATE" when unknown. Phase 1b: channel loaded from KB-20 patient profile via broadcast state.

### Engagement Drop Alert Triggers — With 3-Day Persistence (R3 Fix)

DD#8 Section 4.1 requires **3-day persistence** before level transition alerts fire. This prevents false alerts from weekends, travel, and temporary device issues.

An `EngagementDropAlert` is emitted when:
1. **Level transition downward (3-day persistence)**: Score crosses from GREEN→YELLOW, YELLOW→ORANGE, or ORANGE→RED **AND the new level persists for 3 consecutive days**. Exception: transitions directly to RED fire immediately (patient safety).
2. **Sustained low**: Score remains ORANGE for >= 5 consecutive days (emitted once, suppressed for 7 days)
3. **Cliff drop**: Score drops > 0.30 in a single day (engagement collapse — fires immediately, no persistence)

**3-day persistence mechanism:** `EngagementState` tracks `consecutiveDaysAtCurrentLevel` (separate from `consecutiveLowDays`). On each daily tick, if the level matches the previous day's level, increment. On level change, reset to 1. `LEVEL_TRANSITION` alerts only emit when `consecutiveDaysAtCurrentLevel >= 3` AND the level is alert-worthy (ORANGE or RED).

Suppression: 7-day window per patient per alert type (prevents daily re-alerting for chronically disengaged patients).

### Zombie State Prevention (R4 Fix)

A patient who sends one event then disappears creates a timer chain that fires daily forever (each `onTimer()` reads state → refreshes 14-day TTL → re-registers timer). For 10K patients with 5K permanent disengagements, this accumulates 5K zombie state entries.

**Fix:** `onTimer()` checks staleness before re-registering:
```java
long daysSinceLastEvent = (timestamp - state.getLastUpdated()) / DAY_MS;
if (daysSinceLastEvent > 21) {
    // Patient hasn't sent any event in 21 days (14d window + 7d grace)
    // Stop timer chain, let state expire via TTL
    LOG.info("Stopping timer chain for zombie patient={} ({}d since last event)",
             state.getPatientId(), daysSinceLastEvent);
    // Emit final signal with level=CRITICAL before stopping
    // Do NOT re-register timer
    engagementState.update(state); // Last TTL refresh
    return;
}
```

21 days = 14-day engagement window + 7-day grace period. The final signal emitted before stopping carries `level=CRITICAL` and `phenotype=DISENGAGED_TERMINATED`, giving downstream consumers (KB-20, BCE) a definitive "this patient has left" signal.

**Metric:** `module9.zombie_patients_terminated` counter tracks how many patients hit this condition.

---

## File Structure

All files under `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/`:

### New Models (`models/`)
```
EngagementState.java        — Per-patient 14-day engagement state (8 signal bitmaps + channel + dataTier)
EngagementSignal.java       — Daily output record (score, level, signal densities, phenotype, channel)
EngagementDropAlert.java    — Alert model for engagement threshold breaches (3-day persistence)
EngagementLevel.java        — Enum: GREEN, YELLOW, ORANGE, RED with channel-aware fromScore()
EngagementChannel.java      — Enum: CORPORATE, GOVERNMENT, ACCHS with per-channel thresholds
SignalType.java             — Enum: GLUCOSE_MONITORING, BP_MEASUREMENT, MEDICATION_ADHERENCE, MEAL_LOGGING, APP_SESSION, WEIGHT_MEASUREMENT, GOAL_COMPLETION, APPOINTMENT_ATTENDANCE
```

### New Operators (`operators/`)
```
Module9_EngagementMonitor.java     — Main KeyedProcessFunction (timer-driven)
Module9SignalClassifier.java       — Static: CanonicalEvent → SignalType (or null if irrelevant)
Module9ScoreComputer.java          — Static: EngagementState → score + level + phenotype
Module9DropDetector.java           — Static: current score + previous score + history → Optional<EngagementDropAlert>
```

### Phase 2 Files (Enhancement 6 — Relapse Prediction)
```
models/RelapseRiskScore.java       — Trajectory features + composite relapse score
operators/Module9TrajectoryAnalyzer.java — Static: 14-day composite history → 5 slope features → relapse risk
```

### New Tests (`src/test/java/com/cardiofit/flink/`)
```
operators/Module9SignalClassifierTest.java    — Event → signal classification
operators/Module9ScoreComputerTest.java       — Bitmap → composite score
operators/Module9DropDetectorTest.java        — Alert threshold logic
operators/Module9ProcessElementTest.java      — Full wiring test (processElement + onTimer)
operators/Module9DailyTimerTest.java          — Timer registration and firing
builders/Module9TestBuilder.java              — Test data factory
```

---

## Task 0: Pre-Implementation Setup

- [ ] **0.1** Verify `EventType` enum has `MEDICATION_MISSED` (needed for negative adherence signal)
  - If missing, add it to `EventType.java` with `fromString()` mapping for "MEDICATION_MISSED", "MED_MISSED"
- [ ] **0.2** Verify `KafkaTopics` has `FLINK_ENGAGEMENT_SIGNALS` and `ALERTS_ENGAGEMENT_DROP` (confirmed: both exist)
- [ ] **0.3** Create `Module9TestBuilder.java` in `src/test/java/com/cardiofit/flink/builders/`:

```java
package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import java.util.HashMap;
import java.util.Map;

public class Module9TestBuilder {

    private static final long DAY_MS = 86_400_000L;
    private static final long BASE_TIME = 1743552000000L; // 2025-04-02 00:00:00 UTC

    public static long daysAgo(int days) {
        return BASE_TIME - (days * DAY_MS);
    }

    public static long hoursAgo(int hours) {
        return BASE_TIME - (hours * 3600_000L);
    }

    public static CanonicalEvent medicationAdministered(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.MEDICATION_ADMINISTERED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("drug_name", "metformin");
        payload.put("drug_class", "BIGUANIDE");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent medicationMissed(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.MEDICATION_MISSED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("drug_name", "metformin");
        payload.put("reason", "FORGOT");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent bpReading(String patientId, long timestamp, double sbp, double dbp) {
        CanonicalEvent event = baseEvent(patientId, EventType.VITAL_SIGN, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("systolic_bp", sbp);
        payload.put("diastolic_bp", dbp);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent mealLog(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("report_type", "MEAL_LOG");
        payload.put("meal_type", "lunch");
        payload.put("carb_grams", 45.0);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent appSession(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("report_type", "APP_SESSION");
        payload.put("session_duration_sec", 120);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent goalCompleted(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("report_type", "GOAL_COMPLETED");
        payload.put("goal_type", "steps_10000");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent appointmentAttended(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.ENCOUNTER_START, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("encounter_type", "FOLLOW_UP");
        event.setPayload(payload);
        return event;
    }

    // R1: Glucose monitoring — the most clinically important engagement signal
    public static CanonicalEvent glucoseLabResult(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.LAB_RESULT, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("lab_type", "glucose");
        payload.put("value", 110.0);
        payload.put("unit", "mg/dL");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent cgmDeviceReading(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.DEVICE_READING, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("data_tier", "TIER_1_CGM");
        payload.put("glucose_value", 105.0);
        event.setPayload(payload);
        return event;
    }

    // R1: Weight measurement — weekly engagement signal
    public static CanonicalEvent weightReading(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.VITAL_SIGN, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("weight", 75.5);
        payload.put("unit", "kg");
        event.setPayload(payload);
        return event;
    }

    // R6: Event with data_tier in payload (set by Module 1b)
    public static CanonicalEvent withDataTier(CanonicalEvent event, String dataTier) {
        Map<String, Object> payload = event.getPayload();
        if (payload == null) payload = new HashMap<>();
        payload.put("data_tier", dataTier);
        event.setPayload(payload);
        return event;
    }

    // R2: State builder with channel override
    public static EngagementState stateWithChannel(String patientId, String channel) {
        EngagementState state = new EngagementState();
        state.setPatientId(patientId);
        state.setChannel(channel);
        return state;
    }

    public static EngagementState fullyEngagedState(String patientId) {
        EngagementState state = new EngagementState();
        state.setPatientId(patientId);
        for (SignalType signal : SignalType.values()) {
            boolean[] bitmap = new boolean[14];
            java.util.Arrays.fill(bitmap, true);
            state.setSignalBitmap(signal, bitmap);
        }
        return state;
    }

    public static EngagementState decliningState(String patientId) {
        EngagementState state = new EngagementState();
        state.setPatientId(patientId);
        // Days 0-6: sporadic, Days 7-13: all engaged (showing decline)
        for (SignalType signal : SignalType.values()) {
            boolean[] bitmap = new boolean[14];
            // Older half: engaged
            for (int i = 7; i < 14; i++) bitmap[i] = true;
            // Recent half: only 2 of 7 days
            bitmap[1] = true;
            bitmap[4] = true;
            state.setSignalBitmap(signal, bitmap);
        }
        return state;
    }

    public static EngagementState disengagedState(String patientId) {
        EngagementState state = new EngagementState();
        state.setPatientId(patientId);
        // All false — completely disengaged for 14 days
        for (SignalType signal : SignalType.values()) {
            state.setSignalBitmap(signal, new boolean[14]);
        }
        return state;
    }

    private static CanonicalEvent baseEvent(String patientId, EventType type, long timestamp) {
        CanonicalEvent event = new CanonicalEvent();
        event.setPatientId(patientId);
        event.setEventType(type);
        event.setEventTime(timestamp);
        event.setProcessingTime(System.currentTimeMillis());
        event.setCorrelationId("test-" + java.util.UUID.randomUUID());
        return event;
    }
}
```

---

## Task 1: Models — EngagementState, SignalType, EngagementLevel, EngagementChannel

### 1.1 SignalType Enum (8 Signals — DD#8 Reconciled)

- [ ] Create `models/SignalType.java`:

```java
package com.cardiofit.flink.models;

/**
 * Eight engagement signal channels (DD#8 authoritative + V4 additions).
 * S1-S6 from DD#8 (0.80 total weight), S7-S8 from V4 architecture (0.20).
 */
public enum SignalType {
    GLUCOSE_MONITORING(0.20, "Glucose reading recorded"),  // DD#8: 0.25, adjusted for 8 signals
    BP_MEASUREMENT(0.20, "BP reading recorded"),           // DD#8: 0.25, adjusted
    MEDICATION_ADHERENCE(0.15, "Medication taken"),         // DD#8: 0.15, preserved
    MEAL_LOGGING(0.15, "Meal logged"),                     // DD#8: 0.20, adjusted
    APP_SESSION(0.05, "App session"),                      // DD#8: 0.10, adjusted
    WEIGHT_MEASUREMENT(0.05, "Weight recorded"),           // DD#8: 0.05, preserved
    GOAL_COMPLETION(0.10, "Goal completed"),               // V4 addition
    APPOINTMENT_ATTENDANCE(0.10, "Appointment attended");   // V4 addition

    private final double weight;
    private final String displayName;

    SignalType(double weight, String displayName) {
        this.weight = weight;
        this.displayName = displayName;
    }

    public double getWeight() { return weight; }
    public String getDisplayName() { return displayName; }

    /** Validate weights sum to 1.0 at class load time */
    static {
        double sum = 0.0;
        for (SignalType s : values()) sum += s.weight;
        if (Math.abs(sum - 1.0) > 1e-9) {
            throw new ExceptionInInitializerError(
                "SignalType weights must sum to 1.0, got " + sum);
        }
    }
}
```

### 1.2 EngagementChannel Enum (R2 Fix — Channel-Aware Thresholds)

- [ ] Create `models/EngagementChannel.java`:

```java
package com.cardiofit.flink.models;

/**
 * Deployment channels with channel-specific engagement thresholds.
 * DD#8 Section 4.1: engagement expectations must be calibrated to patient context.
 * A Government patient at 0.35 is GREEN (engaging within structural constraints),
 * not LOW (which would fire a false-positive alert).
 */
public enum EngagementChannel {
    CORPORATE(0.70, 0.40, 0.20),   // Highest expectations: tech-literate, smartphone
    GOVERNMENT(0.40, 0.25, 0.15),  // Reduced: shared devices, limited data, rural
    ACCHS(0.35, 0.20, 0.10);       // Lowest: community health, cultural factors

    private final double greenThreshold;    // GREEN (HIGH) — above this
    private final double yellowThreshold;   // YELLOW (MODERATE) — above this
    private final double orangeThreshold;   // ORANGE (LOW) — above this; below = RED

    EngagementChannel(double green, double yellow, double orange) {
        this.greenThreshold = green;
        this.yellowThreshold = yellow;
        this.orangeThreshold = orange;
    }

    public double getGreenThreshold() { return greenThreshold; }
    public double getYellowThreshold() { return yellowThreshold; }
    public double getOrangeThreshold() { return orangeThreshold; }

    public static EngagementChannel fromString(String channel) {
        if (channel == null) return CORPORATE; // Default
        switch (channel.toUpperCase()) {
            case "GOVERNMENT": case "GOV": return GOVERNMENT;
            case "ACCHS": case "COMMUNITY": return ACCHS;
            default: return CORPORATE;
        }
    }
}
```

### 1.3 EngagementLevel Enum (Channel-Aware)

- [ ] Create `models/EngagementLevel.java`:

```java
package com.cardiofit.flink.models;

/**
 * Engagement levels using DD#8 color terminology (GREEN/YELLOW/ORANGE/RED).
 * Channel-aware: thresholds differ by deployment channel.
 */
public enum EngagementLevel {
    GREEN,    // HIGH — fully engaged
    YELLOW,   // MODERATE — adequate
    ORANGE,   // LOW — declining, alert after 3-day persistence
    RED;      // CRITICAL — disengaged, immediate alert

    /**
     * Classify score into engagement level using channel-specific thresholds.
     * Uses epsilon for IEEE 754 boundary safety (Rule 7).
     */
    public static EngagementLevel fromScore(double score, EngagementChannel channel) {
        if (score >= channel.getGreenThreshold() - 1e-9) return GREEN;
        if (score >= channel.getYellowThreshold() - 1e-9) return YELLOW;
        if (score >= channel.getOrangeThreshold() - 1e-9) return ORANGE;
        return RED;
    }

    /** Convenience overload defaulting to CORPORATE channel */
    public static EngagementLevel fromScore(double score) {
        return fromScore(score, EngagementChannel.CORPORATE);
    }

    public boolean isAlertWorthy() {
        return this == ORANGE || this == RED;
    }
}
```

### 1.4 EngagementState (Per-Patient Keyed State)

- [ ] Create `models/EngagementState.java`:

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.EnumMap;
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
 * R5: Flink 2.x OpenContext (affects operator, not state)
 * R7: History length tracking via validHistoryDays counter
 */
public class EngagementState implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final int WINDOW_DAYS = 14;

    @JsonProperty("patientId")
    private String patientId;

    // R2/R6: Channel and data tier (extracted from first event payload)
    @JsonProperty("channel")
    private String channel; // CORPORATE, GOVERNMENT, ACCHS (default: CORPORATE)

    @JsonProperty("dataTier")
    private String dataTier; // TIER_1_CGM, TIER_2_HYBRID, TIER_3_SMBG

    // 8 signal bitmaps: boolean[14] each
    @JsonProperty("signalBitmaps")
    private Map<SignalType, boolean[]> signalBitmaps;

    // Today's signal flags (reset on daily tick)
    @JsonProperty("todaySignals")
    private Map<SignalType, Boolean> todaySignals;

    // Previous composite score (for drop detection)
    @JsonProperty("previousScore")
    private Double previousScore;

    @JsonProperty("previousLevel")
    private EngagementLevel previousLevel;

    // R3: 3-day persistence counter for level transitions
    @JsonProperty("consecutiveDaysAtCurrentLevel")
    private int consecutiveDaysAtCurrentLevel;

    // Consecutive days at ORANGE or RED (for sustained-low alert)
    @JsonProperty("consecutiveLowDays")
    private int consecutiveLowDays;

    // Last alert emission timestamp per alert type (for 7-day suppression)
    @JsonProperty("alertSuppressionMap")
    private Map<String, Long> alertSuppressionMap;

    // Daily timer registered flag (prevent duplicate timer registration)
    @JsonProperty("dailyTimerRegistered")
    private boolean dailyTimerRegistered;

    // Timestamp of last daily tick (for day boundary detection)
    @JsonProperty("lastDailyTickTimestamp")
    private long lastDailyTickTimestamp;

    // Phase 2: 14-day composite score history for trajectory analysis
    @JsonProperty("compositeHistory14d")
    private double[] compositeHistory14d;

    // R7: Track valid history days (separate from > 0.0 proxy)
    @JsonProperty("validHistoryDays")
    private int validHistoryDays;

    // Counters for observability
    @JsonProperty("totalEventsProcessed")
    private long totalEventsProcessed;

    @JsonProperty("lastUpdated")
    private long lastUpdated;

    public EngagementState() {
        this.signalBitmaps = new EnumMap<>(SignalType.class);
        this.todaySignals = new EnumMap<>(SignalType.class);
        this.alertSuppressionMap = new java.util.HashMap<>();
        this.compositeHistory14d = new double[14];
        this.channel = "CORPORATE"; // Default until loaded from KB-20
        for (SignalType s : SignalType.values()) {
            signalBitmaps.put(s, new boolean[WINDOW_DAYS]);
            todaySignals.put(s, false);
        }
    }

    // --- Signal Operations ---

    /** Mark a signal as observed today */
    public void markSignalToday(SignalType signal) {
        todaySignals.put(signal, true);
    }

    /** Check if a signal was already observed today */
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
            // Shift RIGHT: positions 0-12 → positions 1-13 (drops oldest at 13)
            System.arraycopy(bitmap, 0, bitmap, 1, WINDOW_DAYS - 1);
            bitmap[0] = Boolean.TRUE.equals(todaySignals.get(signal));
            todaySignals.put(signal, false); // Reset for next day
        }

        // Shift composite history RIGHT for Phase 2 trajectory analysis
        System.arraycopy(compositeHistory14d, 0, compositeHistory14d, 1,
                         compositeHistory14d.length - 1);
        compositeHistory14d[0] = todayCompositeScore;

        // R7: Track valid history days
        if (validHistoryDays < WINDOW_DAYS) {
            validHistoryDays++;
        }
    }

    /** Get the 14-day density for a signal (count of true / 14) */
    public double getSignalDensity(SignalType signal) {
        boolean[] bitmap = signalBitmaps.get(signal);
        if (bitmap == null) return 0.0;
        int count = 0;
        for (boolean b : bitmap) {
            if (b) count++;
        }
        return (double) count / WINDOW_DAYS;
    }

    /** Get density map for all signals */
    public Map<SignalType, Double> getAllDensities() {
        Map<SignalType, Double> densities = new EnumMap<>(SignalType.class);
        for (SignalType s : SignalType.values()) {
            densities.put(s, getSignalDensity(s));
        }
        return densities;
    }

    /** R2: Get the engagement channel for threshold lookup */
    public EngagementChannel getEngagementChannel() {
        return EngagementChannel.fromString(channel);
    }

    // --- Alert Suppression ---

    public boolean isAlertSuppressed(String alertType, long currentTime) {
        Long lastEmission = alertSuppressionMap.get(alertType);
        if (lastEmission == null) return false;
        long suppressionWindowMs = 7L * 86_400_000L; // 7 days
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
}
```

**Design rationale for boolean[14] bitmaps vs rolling averages:**
Module 8's `ComorbidityState` uses `rollingBuffers` (Map<String, List<TimestampedValue>>) for continuous values like SBP and FBG. Module 9's signals are **binary** (engaged/not-engaged per day), so a boolean array is both simpler and cheaper:
- O(1) per-signal per day vs O(n) for searching timestamped lists
- No need for `getValueApproxDaysAgo()` interpolation
- No risk of the "average never computed" bug caught in Module 8 review (R2)

---

## Task 2: Models — EngagementSignal (Daily Output)

- [ ] Create `models/EngagementSignal.java`:

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.Map;
import java.util.UUID;

/**
 * Daily per-patient engagement signal emitted by Module 9.
 * Published to flink.engagement-signals.
 * Consumed by KB-20 (patient profile), KB-26 (MHRI), KB-21 (behavioral intelligence), SPAN.
 */
@JsonInclude(JsonInclude.Include.NON_NULL)
public class EngagementSignal implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("signalId")
    private String signalId;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("compositeScore")
    private double compositeScore; // 0.0 to 1.0

    @JsonProperty("engagementLevel")
    private EngagementLevel engagementLevel;

    @JsonProperty("signalDensities")
    private Map<SignalType, Double> signalDensities; // Per-signal 14-day density

    @JsonProperty("phenotype")
    private String phenotype; // STEADY, DECLINING, SPORADIC

    @JsonProperty("previousScore")
    private Double previousScore; // Yesterday's composite (for trend)

    @JsonProperty("scoreDelta")
    private Double scoreDelta; // today - yesterday

    @JsonProperty("consecutiveLowDays")
    private int consecutiveLowDays;

    @JsonProperty("computedAt")
    private long computedAt;

    @JsonProperty("correlationId")
    private String correlationId;

    // R2/R6: Channel and data tier for downstream consumers
    @JsonProperty("channel")
    private String channel; // CORPORATE, GOVERNMENT, ACCHS

    @JsonProperty("dataTier")
    private String dataTier; // TIER_1_CGM, TIER_2_HYBRID, TIER_3_SMBG

    // Phase 2: trajectory features
    @JsonProperty("relapseRiskScore")
    private Double relapseRiskScore;

    public EngagementSignal() {}

    public static EngagementSignal create(String patientId, double score,
                                           EngagementLevel level,
                                           Map<SignalType, Double> densities,
                                           String phenotype,
                                           Double previousScore,
                                           int consecutiveLowDays) {
        EngagementSignal signal = new EngagementSignal();
        signal.signalId = UUID.randomUUID().toString();
        signal.patientId = patientId;
        signal.compositeScore = score;
        signal.engagementLevel = level;
        signal.signalDensities = densities;
        signal.phenotype = phenotype;
        signal.previousScore = previousScore;
        signal.scoreDelta = (previousScore != null) ? score - previousScore : null;
        signal.consecutiveLowDays = consecutiveLowDays;
        signal.computedAt = System.currentTimeMillis();
        return signal;
    }

    // Getters
    public String getSignalId() { return signalId; }
    public String getPatientId() { return patientId; }
    public double getCompositeScore() { return compositeScore; }
    public EngagementLevel getEngagementLevel() { return engagementLevel; }
    public Map<SignalType, Double> getSignalDensities() { return signalDensities; }
    public String getPhenotype() { return phenotype; }
    public Double getPreviousScore() { return previousScore; }
    public Double getScoreDelta() { return scoreDelta; }
    public int getConsecutiveLowDays() { return consecutiveLowDays; }
    public long getComputedAt() { return computedAt; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String id) { this.correlationId = id; }
    public String getChannel() { return channel; }
    public void setChannel(String channel) { this.channel = channel; }
    public String getDataTier() { return dataTier; }
    public void setDataTier(String dataTier) { this.dataTier = dataTier; }
    public Double getRelapseRiskScore() { return relapseRiskScore; }
    public void setRelapseRiskScore(Double score) { this.relapseRiskScore = score; }
}
```

---

## Task 3: Models — EngagementDropAlert

- [ ] Create `models/EngagementDropAlert.java`:

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.UUID;

/**
 * Alert emitted when engagement drops below threshold or collapses.
 * Published to alerts.engagement-drop.
 * Consumed by notification-service, KB-21.
 */
public class EngagementDropAlert implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum DropType {
        LEVEL_TRANSITION,   // Crossed downward (YELLOW→ORANGE, ORANGE→RED, etc.) with 3-day persistence
        SUSTAINED_LOW,      // ORANGE or RED for >= 5 consecutive days
        CLIFF_DROP          // Score dropped > 0.30 in single day (fires immediately)
    }

    @JsonProperty("alertId")
    private String alertId;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("dropType")
    private DropType dropType;

    @JsonProperty("severity")
    private String severity; // WARNING or CRITICAL

    @JsonProperty("currentScore")
    private double currentScore;

    @JsonProperty("previousScore")
    private Double previousScore;

    @JsonProperty("currentLevel")
    private EngagementLevel currentLevel;

    @JsonProperty("previousLevel")
    private EngagementLevel previousLevel;

    @JsonProperty("consecutiveLowDays")
    private int consecutiveLowDays;

    @JsonProperty("triggerSummary")
    private String triggerSummary;

    @JsonProperty("recommendedAction")
    private String recommendedAction;

    @JsonProperty("suppressionKey")
    private String suppressionKey;

    @JsonProperty("createdAt")
    private long createdAt;

    public EngagementDropAlert() {}

    public static EngagementDropAlert create(String patientId, DropType dropType,
                                              double currentScore, Double previousScore,
                                              EngagementLevel currentLevel,
                                              EngagementLevel previousLevel,
                                              int consecutiveLowDays) {
        EngagementDropAlert alert = new EngagementDropAlert();
        alert.alertId = UUID.randomUUID().toString();
        alert.patientId = patientId;
        alert.dropType = dropType;
        alert.currentScore = currentScore;
        alert.previousScore = previousScore;
        alert.currentLevel = currentLevel;
        alert.previousLevel = previousLevel;
        alert.consecutiveLowDays = consecutiveLowDays;
        alert.createdAt = System.currentTimeMillis();

        // Severity: CRITICAL if score < 0.20 or cliff drop, WARNING otherwise
        alert.severity = (currentLevel == EngagementLevel.CRITICAL
                         || dropType == DropType.CLIFF_DROP)
                         ? "CRITICAL" : "WARNING";

        // Suppression key: dropType + patientId (7-day window)
        alert.suppressionKey = dropType.name() + ":" + patientId;

        // Trigger summary
        switch (dropType) {
            case LEVEL_TRANSITION:
                alert.triggerSummary = String.format(
                    "Engagement dropped from %s (%.2f) to %s (%.2f)",
                    previousLevel, previousScore, currentLevel, currentScore);
                alert.recommendedAction = currentLevel == EngagementLevel.CRITICAL
                    ? "Physician outreach recommended. Patient has been critically disengaged."
                    : "BCE to shift to re-engagement micro-commitments. Reduce SPAN frequency.";
                break;
            case SUSTAINED_LOW:
                alert.triggerSummary = String.format(
                    "Engagement has been LOW for %d consecutive days (score: %.2f)",
                    consecutiveLowDays, currentScore);
                alert.recommendedAction =
                    "Consider physician follow-up call. Review barriers to engagement.";
                break;
            case CLIFF_DROP:
                alert.triggerSummary = String.format(
                    "Engagement collapsed: %.2f → %.2f (delta: %.2f) in one day",
                    previousScore, currentScore, currentScore - previousScore);
                alert.recommendedAction =
                    "Urgent: Check for life event disruption. Empathetic outreach, not corrective.";
                break;
        }

        return alert;
    }

    // Getters
    public String getAlertId() { return alertId; }
    public String getPatientId() { return patientId; }
    public DropType getDropType() { return dropType; }
    public String getSeverity() { return severity; }
    public double getCurrentScore() { return currentScore; }
    public Double getPreviousScore() { return previousScore; }
    public EngagementLevel getCurrentLevel() { return currentLevel; }
    public EngagementLevel getPreviousLevel() { return previousLevel; }
    public int getConsecutiveLowDays() { return consecutiveLowDays; }
    public String getTriggerSummary() { return triggerSummary; }
    public String getRecommendedAction() { return recommendedAction; }
    public String getSuppressionKey() { return suppressionKey; }
    public long getCreatedAt() { return createdAt; }
}
```

---

## Task 4: Operators — Module9SignalClassifier (Static, No Flink Dependencies)

- [ ] Create `operators/Module9SignalClassifier.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.models.SignalType;
import java.util.Map;

/**
 * Classifies a CanonicalEvent into one of the 8 engagement signal channels.
 * Returns null if the event is irrelevant to engagement tracking.
 *
 * This is the HIGHEST-RISK piece of Module 9 — field names depend on the
 * actual CanonicalEvent payload schema from Module 1b.
 *
 * R1 Fix: Glucose monitoring (S1) is restored as the highest-weighted signal
 * (tied with BP at 0.20). Sourced from LAB_RESULT with glucose/fbg/cgm lab types
 * and from DEVICE_READING for CGM devices.
 */
public final class Module9SignalClassifier {

    private Module9SignalClassifier() {} // Static utility

    /**
     * Classify a CanonicalEvent into a SignalType.
     * @return SignalType if the event maps to an engagement signal, null otherwise
     */
    public static SignalType classify(CanonicalEvent event) {
        if (event == null || event.getEventType() == null) return null;

        EventType type = event.getEventType();
        Map<String, Object> payload = event.getPayload();

        switch (type) {
            // S3: Medication Adherence — patient took their medication
            case MEDICATION_ADMINISTERED:
                return SignalType.MEDICATION_ADHERENCE;

            // S1: Glucose Monitoring — lab result with glucose/fbg lab type
            case LAB_RESULT:
                return classifyLabResult(payload);

            // S2: BP Measurement, S6: Weight Measurement — vital sign sub-classification
            case VITAL_SIGN:
            case VITAL_SIGNS:
                return classifyVitalSign(payload);

            // S4, S5, S7: PATIENT_REPORTED — sub-classify by report_type
            case PATIENT_REPORTED:
                return classifyPatientReported(payload);

            // S8: Appointment Attendance
            case ENCOUNTER_START:
            case ENCOUNTER_UPDATE:
                return SignalType.APPOINTMENT_ATTENDANCE;

            // S8: Task completion (appointment/check-in completed)
            case TASK_COMPLETED:
                if (payload != null) {
                    String taskType = getStringField(payload, "task_type");
                    if ("APPOINTMENT".equalsIgnoreCase(taskType)
                        || "CHECKIN".equalsIgnoreCase(taskType)) {
                        return SignalType.APPOINTMENT_ATTENDANCE;
                    }
                }
                return null;

            // S1: CGM device reading (glucose), S2: Home BP device
            case DEVICE_READING:
                return classifyDeviceReading(payload);

            default:
                return null; // Event type not relevant to engagement
        }
    }

    /**
     * Classify LAB_RESULT events.
     * Glucose/FBG/CGM lab types → GLUCOSE_MONITORING (S1).
     * Other lab types (creatinine, potassium, etc.) are NOT patient-driven
     * engagement — they're ordered by physicians.
     */
    private static SignalType classifyLabResult(Map<String, Object> payload) {
        if (payload == null) return null;
        String labType = getStringField(payload, "lab_type");
        if (labType == null) labType = getStringField(payload, "labName");
        if (labType == null) return null;

        String upper = labType.toUpperCase();
        if (upper.contains("GLUCOSE") || upper.contains("FBG") || upper.contains("PPBG")
            || upper.contains("CGM") || upper.contains("HBA1C")
            || upper.contains("BLOOD_SUGAR") || upper.contains("SMBG")) {
            return SignalType.GLUCOSE_MONITORING;
        }
        return null; // Non-glucose labs are physician-ordered, not engagement
    }

    /**
     * Classify VITAL_SIGN events into BP (S2) or Weight (S6).
     * Vital signs with systolic_bp → BP_MEASUREMENT.
     * Vital signs with weight → WEIGHT_MEASUREMENT.
     * BP takes priority if both are present.
     */
    private static SignalType classifyVitalSign(Map<String, Object> payload) {
        if (payload == null) return null;
        if (payload.containsKey("systolic_bp")) {
            return SignalType.BP_MEASUREMENT;
        }
        if (payload.containsKey("weight")) {
            return SignalType.WEIGHT_MEASUREMENT;
        }
        return null; // Other vitals (HR, temp, SpO2) — not engagement signals
    }

    /**
     * Classify DEVICE_READING events.
     * CGM data → GLUCOSE_MONITORING (S1).
     * Home BP monitor → BP_MEASUREMENT (S2).
     */
    private static SignalType classifyDeviceReading(Map<String, Object> payload) {
        if (payload == null) return null;
        // CGM device readings
        String dataTier = getStringField(payload, "data_tier");
        if ("TIER_1_CGM".equals(dataTier)) {
            return SignalType.GLUCOSE_MONITORING;
        }
        Boolean cgmActive = payload.get("cgm_active") instanceof Boolean
            ? (Boolean) payload.get("cgm_active") : null;
        if (Boolean.TRUE.equals(cgmActive)) {
            return SignalType.GLUCOSE_MONITORING;
        }
        // Home BP device
        if (payload.containsKey("systolic_bp")) {
            return SignalType.BP_MEASUREMENT;
        }
        return null;
    }

    /**
     * Sub-classify PATIENT_REPORTED events by report_type field.
     * This handles the multiplexed nature of patient-reported data
     * (meals, goals, app sessions all arrive as PATIENT_REPORTED).
     */
    private static SignalType classifyPatientReported(Map<String, Object> payload) {
        if (payload == null) return null;

        String reportType = getStringField(payload, "report_type");
        if (reportType == null) {
            // Fallback: check for symptom_type (symptom reports are not engagement signals)
            String symptomType = getStringField(payload, "symptom_type");
            if (symptomType != null) return null; // Symptom reports → Module 8, not Module 9
            return null;
        }

        switch (reportType.toUpperCase()) {
            case "MEAL_LOG":
            case "FOOD_LOG":
            case "DIETARY_LOG":
                return SignalType.MEAL_LOGGING;

            case "APP_SESSION":
            case "APP_INTERACTION":
                return SignalType.APP_SESSION;

            case "GOAL_COMPLETED":
            case "GOAL_ACHIEVED":
            case "TARGET_MET":
                return SignalType.GOAL_COMPLETION;

            default:
                return null; // Other PATIENT_REPORTED types (exercise, mood) — future extension
        }
    }

    private static String getStringField(Map<String, Object> payload, String key) {
        Object val = payload.get(key);
        return (val instanceof String) ? (String) val : null;
    }
}
```

- [ ] Create `operators/Module9SignalClassifierTest.java` (TDD — write before implementation if preferred):

Test cases:
1. `LAB_RESULT` with `lab_type=glucose` → `GLUCOSE_MONITORING`
2. `LAB_RESULT` with `lab_type=fbg` → `GLUCOSE_MONITORING`
3. `LAB_RESULT` with `lab_type=creatinine` → `null` (physician-ordered, not engagement)
4. `DEVICE_READING` with `data_tier=TIER_1_CGM` → `GLUCOSE_MONITORING`
5. `MEDICATION_ADMINISTERED` → `MEDICATION_ADHERENCE`
6. `MEDICATION_MISSED` → `null` (missed dose is NOT engagement)
7. `VITAL_SIGN` with `systolic_bp` → `BP_MEASUREMENT`
8. `VITAL_SIGN` with `weight` only → `WEIGHT_MEASUREMENT`
9. `VITAL_SIGN` with both `systolic_bp` and `weight` → `BP_MEASUREMENT` (priority)
10. `VITAL_SIGN` without BP or weight (HR only) → `null`
11. `PATIENT_REPORTED` with `report_type=MEAL_LOG` → `MEAL_LOGGING`
12. `PATIENT_REPORTED` with `report_type=APP_SESSION` → `APP_SESSION`
13. `PATIENT_REPORTED` with `report_type=GOAL_COMPLETED` → `GOAL_COMPLETION`
14. `PATIENT_REPORTED` with `symptom_type=HYPOGLYCEMIA` (no report_type) → `null`
15. `ENCOUNTER_START` → `APPOINTMENT_ATTENDANCE`
16. `DEVICE_READING` with `systolic_bp` → `BP_MEASUREMENT`
17. `null` event → `null`
18. Event with null payload → `null`

---

## Task 5: Operators — Module9ScoreComputer (Static, Channel-Aware)

- [ ] Create `operators/Module9ScoreComputer.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.Map;

/**
 * Computes the weighted composite engagement score and phenotype
 * from the 14-day signal bitmaps. Pure function, no Flink dependencies.
 *
 * R2 Fix: Channel-aware level classification (CORPORATE/GOVERNMENT/ACCHS).
 * R7 Fix: Phenotype uses validHistoryDays counter, not > 0.0 proxy.
 */
public final class Module9ScoreComputer {

    private Module9ScoreComputer() {}

    public static Result compute(EngagementState state) {
        Map<SignalType, Double> densities = state.getAllDensities();

        double composite = 0.0;
        for (SignalType signal : SignalType.values()) {
            double density = densities.getOrDefault(signal, 0.0);
            composite += density * signal.getWeight();
        }
        // IEEE 754 safety: cap at 1.0
        composite = Math.min(1.0, composite);

        // R2: Channel-aware level classification
        EngagementChannel channel = state.getEngagementChannel();
        EngagementLevel level = EngagementLevel.fromScore(composite, channel);

        // R7: Phenotype with validHistoryDays tracking
        String phenotype = classifyPhenotype(state);

        return new Result(composite, level, densities, phenotype, channel);
    }

    /**
     * Classify engagement phenotype based on temporal pattern.
     *
     * STEADY:    Recent 7d average >= 90% of older 7d average
     * DECLINING: Recent 7d average < 70% of older 7d average
     * SPORADIC:  Everything else (inconsistent engagement)
     *
     * R7 Fix: Uses validHistoryDays counter instead of checking history[i] > 0.
     * A patient with a genuinely zero composite score (completely disengaged) was
     * previously excluded from the phenotype calculation, making ratio comparison
     * unreliable. Now we use the validHistoryDays counter to know exactly how many
     * days of actual data exist.
     */
    private static String classifyPhenotype(EngagementState state) {
        double[] history = state.getCompositeHistory14d();
        int validDays = state.getValidHistoryDays();

        // Need at least 10 days of history for meaningful phenotype
        if (validDays < 10) return "SPORADIC";

        // Recent half (days 0-6) vs older half (days 7-13)
        int recentDays = Math.min(validDays, 7);
        int olderDays = Math.min(validDays - 7, 7);
        if (olderDays < 3) return "SPORADIC"; // Not enough older data

        double recentSum = 0.0, olderSum = 0.0;
        for (int i = 0; i < recentDays; i++) {
            recentSum += history[i]; // Include zeros — they represent genuinely disengaged days
        }
        for (int i = 7; i < 7 + olderDays; i++) {
            olderSum += history[i];
        }

        double recentAvg = recentSum / recentDays;
        double olderAvg = olderSum / olderDays;

        if (olderAvg < 1e-9) {
            // Older period was fully disengaged
            return recentAvg > 1e-9 ? "STEADY" : "SPORADIC"; // Improvement or still zero
        }

        double ratio = recentAvg / olderAvg;

        if (ratio >= 0.90 - 1e-9) return "STEADY";
        if (ratio < 0.70 + 1e-9) return "DECLINING";
        return "SPORADIC";
    }

    public static class Result {
        public final double compositeScore;
        public final EngagementLevel level;
        public final Map<SignalType, Double> densities;
        public final String phenotype;
        public final EngagementChannel channel;

        public Result(double compositeScore, EngagementLevel level,
                     Map<SignalType, Double> densities, String phenotype,
                     EngagementChannel channel) {
            this.compositeScore = compositeScore;
            this.level = level;
            this.densities = densities;
            this.phenotype = phenotype;
            this.channel = channel;
        }
    }
}
```

- [ ] Write `Module9ScoreComputerTest.java`:

Test cases:
1. Fully engaged state → score 1.0, level GREEN, phenotype STEADY
2. Fully disengaged state → score 0.0, level RED (CORPORATE), level RED (GOVERNMENT)
3. Only glucose monitoring (weight 0.20), all 14 days → score 0.20, CORPORATE=ORANGE, GOVERNMENT=YELLOW
4. All signals active 7 of 14 days → score ~0.50, CORPORATE=MODERATE, GOVERNMENT=GREEN
5. Score 0.35: CORPORATE=ORANGE, GOVERNMENT=YELLOW, ACCHS=GREEN (channel divergence test)
6. Recent 7 days empty, older 7 days full, validHistoryDays=14 → phenotype DECLINING
7. Consistent 50% engagement, validHistoryDays=14 → phenotype STEADY (ratio ~1.0)
8. validHistoryDays=5 → phenotype always SPORADIC (insufficient data)
9. Edge: CORPORATE score at exactly 0.70 → level GREEN (epsilon boundary)
10. Patient with zero composite for 14 days (validHistoryDays=14) → phenotype correctly uses zeros, not excluded
11. Older period all zero, recent period improving → phenotype STEADY (not SPORADIC)

---

## Task 6: Operators — Module9DropDetector (Static, 3-Day Persistence)

- [ ] Create `operators/Module9DropDetector.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.Optional;

/**
 * Detects engagement drops that warrant alerts.
 * Three detection modes:
 * 1. Level transition downward — WITH 3-DAY PERSISTENCE (R3 Fix, DD#8 Section 4.1)
 * 2. Sustained low (ORANGE for >= 5 consecutive days)
 * 3. Cliff drop (score delta > 0.30 in one day — fires immediately)
 *
 * DD#8 Section 4.1: "IF composite transitions from YELLOW to ORANGE (or stays
 * ORANGE for 3 consecutive days): emit COACHING_NUDGE." This prevents false
 * alerts from weekends, travel, and temporary device issues.
 *
 * Exception: Transitions directly to RED fire immediately (patient safety).
 */
public final class Module9DropDetector {

    private static final int SUSTAINED_LOW_THRESHOLD_DAYS = 5;
    private static final double CLIFF_DROP_THRESHOLD = 0.30;
    private static final int LEVEL_TRANSITION_PERSISTENCE_DAYS = 3;

    private Module9DropDetector() {}

    /**
     * Check if an engagement drop alert should be emitted.
     * Returns empty if no alert needed, persistence not met, or suppressed.
     *
     * @param consecutiveDaysAtLevel How many days the patient has been at currentLevel
     */
    public static Optional<EngagementDropAlert> detect(
            String patientId,
            double currentScore,
            EngagementLevel currentLevel,
            Double previousScore,
            EngagementLevel previousLevel,
            int consecutiveLowDays,
            int consecutiveDaysAtLevel,
            EngagementState state,
            long currentTime) {

        // 1. Cliff drop detection (highest priority — fires immediately, no persistence)
        if (previousScore != null) {
            double delta = previousScore - currentScore;
            if (delta >= CLIFF_DROP_THRESHOLD - 1e-9) {
                String key = EngagementDropAlert.DropType.CLIFF_DROP.name() + ":" + patientId;
                if (!state.isAlertSuppressed(key, currentTime)) {
                    return Optional.of(EngagementDropAlert.create(
                        patientId, EngagementDropAlert.DropType.CLIFF_DROP,
                        currentScore, previousScore, currentLevel, previousLevel,
                        consecutiveLowDays));
                }
            }
        }

        // 2. Level transition downward — WITH 3-DAY PERSISTENCE (R3 Fix)
        // Exception: RED transitions fire immediately (patient safety)
        if (previousLevel != null && currentLevel.isAlertWorthy()) {
            boolean transitionedDown =
                (previousLevel == EngagementLevel.YELLOW && currentLevel == EngagementLevel.ORANGE) ||
                (previousLevel == EngagementLevel.ORANGE && currentLevel == EngagementLevel.RED) ||
                (previousLevel == EngagementLevel.YELLOW && currentLevel == EngagementLevel.RED) ||
                (previousLevel == EngagementLevel.GREEN && currentLevel == EngagementLevel.ORANGE) ||
                (previousLevel == EngagementLevel.GREEN && currentLevel == EngagementLevel.RED);

            if (transitionedDown) {
                // RED transitions fire immediately (patient safety override)
                boolean fireImmediately = (currentLevel == EngagementLevel.RED);

                // ORANGE transitions require 3-day persistence (DD#8 Section 4.1)
                boolean persistenceMet = fireImmediately
                    || consecutiveDaysAtLevel >= LEVEL_TRANSITION_PERSISTENCE_DAYS;

                if (persistenceMet) {
                    String key = EngagementDropAlert.DropType.LEVEL_TRANSITION.name()
                                 + ":" + patientId;
                    if (!state.isAlertSuppressed(key, currentTime)) {
                        return Optional.of(EngagementDropAlert.create(
                            patientId, EngagementDropAlert.DropType.LEVEL_TRANSITION,
                            currentScore, previousScore, currentLevel, previousLevel,
                            consecutiveLowDays));
                    }
                }
            }
        }

        // 3. Sustained low (ORANGE or RED for >= 5 consecutive days)
        if (currentLevel.isAlertWorthy()
            && consecutiveLowDays >= SUSTAINED_LOW_THRESHOLD_DAYS) {
            String key = EngagementDropAlert.DropType.SUSTAINED_LOW.name() + ":" + patientId;
            if (!state.isAlertSuppressed(key, currentTime)) {
                return Optional.of(EngagementDropAlert.create(
                    patientId, EngagementDropAlert.DropType.SUSTAINED_LOW,
                    currentScore, previousScore, currentLevel, previousLevel,
                    consecutiveLowDays));
            }
        }

        return Optional.empty();
    }
}
```

- [ ] Write `Module9DropDetectorTest.java`:

Test cases:
1. YELLOW → ORANGE, day 1 of ORANGE → NO alert (3-day persistence not met)
2. YELLOW → ORANGE, day 3 of ORANGE → alert emitted (persistence met, severity WARNING)
3. GREEN → RED transition → alert IMMEDIATELY (patient safety, no persistence)
4. ORANGE → RED transition → alert IMMEDIATELY
5. GREEN → GREEN (no change) → no alert
6. ORANGE for 4 days → no sustained alert
7. ORANGE for 5 days → sustained alert
8. Score drops 0.31 in one day → cliff drop alert (CRITICAL, fires immediately)
9. Score drops 0.29 → no cliff drop alert
10. Alert suppression: emit once, suppress for 7 days
11. Alert suppression: re-emit after 7 days expire
12. Multiple alert types in same tick: cliff drop wins (highest priority)
13. GOVERNMENT patient at 0.35 (GREEN in GOVERNMENT) → no alert (channel-aware threshold test — tested in ScoreComputer)

---

## Task 7: Main Operator — Module9_EngagementMonitor

- [ ] Create `operators/Module9_EngagementMonitor.java`:

```java
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
import java.util.Map;
import java.util.Optional;

/**
 * Module 9: Engagement Monitor.
 *
 * Timer-driven KeyedProcessFunction that tracks 8 engagement signals
 * (DD#8 reconciled) over a 14-day rolling window and emits daily
 * channel-aware composite scores.
 *
 * processElement(): Classifies events → marks today's signal bitmap.
 *                   Extracts data_tier (from payload) and channel (Phase 1b: KB-20).
 * onTimer(): (Daily at 23:59 UTC processing time) Computes score → emits
 *            signal → detects drops (3-day persistence) → advances day.
 *            Zombie check: stops timer chain after 21 days of no events.
 *
 * Input:  CanonicalEvent from enriched-patient-events-v1
 * Output: EngagementSignal to flink.engagement-signals (main)
 *         EngagementDropAlert to alerts.engagement-drop (side output)
 *
 * Review fixes incorporated: R1 (8 signals), R2 (channel-aware), R3 (3-day persistence),
 * R4 (zombie prevention), R5 (OpenContext), R6 (channel/dataTier extraction), R7 (validHistoryDays).
 */
public class Module9_EngagementMonitor
        extends KeyedProcessFunction<String, CanonicalEvent, EngagementSignal> {

    private static final Logger LOG = LoggerFactory.getLogger(Module9_EngagementMonitor.class);

    private static final long DAY_MS = 86_400_000L;
    /** 14d engagement window + 7d grace = 21d before zombie termination */
    private static final int ZOMBIE_THRESHOLD_DAYS = 21;

    /** Side output for engagement drop alerts */
    public static final OutputTag<EngagementDropAlert> ENGAGEMENT_DROP_TAG =
        new OutputTag<>("engagement-drop-alerts",
            TypeInformation.of(EngagementDropAlert.class));

    private transient ValueState<EngagementState> engagementState;

    /**
     * Flink 2.x open() signature — uses OpenContext, NOT Configuration.
     * Confirmed against Module 6 and Module 8 on disk (R5 Fix).
     */
    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        ValueStateDescriptor<EngagementState> stateDesc =
            new ValueStateDescriptor<>("engagement-state", EngagementState.class);

        // 14-day TTL per architecture spec (Section 6.2)
        org.apache.flink.api.common.state.StateTtlConfig ttl =
            org.apache.flink.api.common.state.StateTtlConfig
                .newBuilder(org.apache.flink.api.common.time.Time.days(14))
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
        // 1. Get or create state
        EngagementState state = engagementState.value();
        if (state == null) {
            state = new EngagementState();
            state.setPatientId(event.getPatientId());
            LOG.debug("Initializing engagement state for patient={}", event.getPatientId());
        }

        // 2. R6: Extract data_tier from first event's payload (set by Module 1b canonicalizer)
        // data_tier tells us CGM vs SMBG patient — affects glucose signal density expectations
        if (state.getDataTier() == null && event.getPayload() != null) {
            Object dataTier = event.getPayload().get("data_tier");
            if (dataTier instanceof String) {
                state.setDataTier((String) dataTier);
                LOG.debug("Set data_tier={} for patient={}", dataTier, event.getPatientId());
            }
        }

        // R6: Channel extraction — Phase 1: default CORPORATE.
        // Phase 1b: loaded from KB-20 patient profile via BroadcastProcessFunction.
        // Channel is set in EngagementState constructor as "CORPORATE".
        // When KB-20 broadcast state is available, it will update state.channel here.

        // 3. Classify event into signal channel
        SignalType signal = Module9SignalClassifier.classify(event);

        if (signal != null) {
            // Mark today's signal (idempotent — multiple BP readings still count as 1)
            state.markSignalToday(signal);
        }

        // 4. Register daily timer (processing time, once per patient)
        if (!state.isDailyTimerRegistered()) {
            long nextMidnight = computeNextDailyTick(ctx.timerService().currentProcessingTime());
            ctx.timerService().registerProcessingTimeTimer(nextMidnight);
            state.setDailyTimerRegistered(true);
            LOG.debug("Registered daily timer for patient={} at {}",
                      event.getPatientId(), Instant.ofEpochMilli(nextMidnight));
        }

        // 5. Update counters
        state.setTotalEventsProcessed(state.getTotalEventsProcessed() + 1);
        state.setLastUpdated(ctx.timerService().currentProcessingTime());

        engagementState.update(state);
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx,
                         Collector<EngagementSignal> out) throws Exception {
        EngagementState state = engagementState.value();
        if (state == null) return; // State expired, nothing to do

        // R4/R5: Zombie state prevention — stop timer chain after 21 days of silence
        long daysSinceLastEvent = (timestamp - state.getLastUpdated()) / DAY_MS;
        if (daysSinceLastEvent > ZOMBIE_THRESHOLD_DAYS) {
            LOG.info("Stopping timer chain for zombie patient={} ({}d since last event)",
                     state.getPatientId(), daysSinceLastEvent);

            // Emit final DISENGAGED_TERMINATED signal before stopping
            Module9ScoreComputer.Result finalResult = Module9ScoreComputer.compute(state);
            EngagementSignal finalSignal = EngagementSignal.create(
                state.getPatientId(),
                finalResult.compositeScore,
                EngagementLevel.RED, // Force RED for terminated patient
                finalResult.densities,
                "DISENGAGED_TERMINATED",
                state.getPreviousScore(),
                state.getConsecutiveLowDays()
            );
            finalSignal.setCorrelationId("zombie-termination-" + state.getPatientId());
            out.collect(finalSignal);

            // Last TTL refresh, then let state expire naturally — do NOT re-register timer
            engagementState.update(state);
            return;
        }

        // 1. Compute today's composite score (R2: channel-aware via ScoreComputer)
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

        // 3. Emit main output → flink.engagement-signals
        out.collect(signal);

        // R3: Track consecutive days at current level (for 3-day persistence)
        if (state.getPreviousLevel() != null && result.level == state.getPreviousLevel()) {
            state.setConsecutiveDaysAtCurrentLevel(
                state.getConsecutiveDaysAtCurrentLevel() + 1);
        } else {
            state.setConsecutiveDaysAtCurrentLevel(1); // New level, reset to day 1
        }

        // 4. Detect engagement drops → alerts.engagement-drop (side output)
        // R3: Pass consecutiveDaysAtLevel for 3-day persistence check
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
            state.setConsecutiveLowDays(0); // Reset on recovery
        }

        // 6. Advance the rolling window (shift bitmaps, store composite)
        state.advanceDay(result.compositeScore);

        // 7. Store previous score/level for next day's comparison
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
     * Compute next daily tick at 23:59 UTC.
     * If current time IS 23:59, schedule for next day.
     */
    static long computeNextDailyTick(long currentTimeMs) {
        ZonedDateTime now = Instant.ofEpochMilli(currentTimeMs).atZone(ZoneOffset.UTC);
        ZonedDateTime todayTick = now.toLocalDate().atTime(23, 59).atZone(ZoneOffset.UTC);

        if (now.isAfter(todayTick) || now.isEqual(todayTick)) {
            // Already past today's tick, schedule for tomorrow
            todayTick = todayTick.plusDays(1);
        }

        return todayTick.toInstant().toEpochMilli();
    }
}
```

**Key design notes captured during review:**

1. **processElement() NEVER computes scores.** It only classifies and sets bitmap bits. This is the fundamental difference from Module 8's processElement (which evaluates all 17 CID rules per event).

2. **Daily timer uses processing time, not event time.** Event time would not fire if the patient stops sending events — defeating the purpose. Processing time fires regardless.

3. **`markSignalToday()` is idempotent.** A CGM patient's 288th glucose reading still counts as 1 engaged day for S1. No overcounting.

4. **Timer re-registration happens in onTimer().** Each tick schedules the next one. The first timer is registered in processElement() on first event. If a patient never sends another event after Day 1, the timer chain continues for up to 21 days (14d window + 7d grace), then terminates to prevent zombie state accumulation (R4/R5).

5. **R5: open(OpenContext) not open(Configuration).** Flink 2.x deprecates the Configuration overload. Confirmed against Module 6 and Module 8 on disk.

6. **R6: data_tier extracted from first event's payload.** Module 1b canonicalizer sets `data_tier` during ingestion. Channel defaults to "CORPORATE" until KB-20 broadcast state is available (Phase 1b).

7. **R3: consecutiveDaysAtCurrentLevel tracked separately from consecutiveLowDays.** The former drives 3-day persistence for LEVEL_TRANSITION alerts. The latter drives SUSTAINED_LOW alerts (5-day threshold). Both reset on recovery, but on different triggers.

---

## Task 8: Orchestrator Wiring

- [ ] Add Module 9 cases to `FlinkJobOrchestrator.java`:

In the job type switch statement, add:
```java
case "engagement":
case "module9":
case "engagement-monitor":
    launchEngagementMonitor(env);
    break;
```

- [ ] Add `launchEngagementMonitor()` method:

```java
private static void launchEngagementMonitor(StreamExecutionEnvironment env) throws Exception {
    LOG.info("Launching Module 9: Engagement Monitor");

    String bootstrap = KafkaConfigLoader.getBootstrapServers();

    // Source: enriched-patient-events-v1 (same as Module 8)
    KafkaSource<CanonicalEvent> source = KafkaSource.<CanonicalEvent>builder()
        .setBootstrapServers(bootstrap)
        .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
        .setGroupId("flink-module9-engagement-monitor-v1")
        .setStartingOffsets(OffsetsInitializer.earliest())
        .setValueOnlyDeserializer(new CanonicalEventDeserializer())
        .build();

    // Pipeline: keyBy patientId → Module 9 operator
    SingleOutputStreamOperator<EngagementSignal> signals = env
        .fromSource(source,
            WatermarkStrategy.<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                .withTimestampAssigner((e, ts) -> e.getEventTime()),
            "Kafka Source: Enriched Patient Events (Module 9)")
        .keyBy(CanonicalEvent::getPatientId)
        .process(new Module9_EngagementMonitor())
        .uid("module9-engagement-monitor")
        .name("Module 9: Engagement Monitor");

    // Main output → flink.engagement-signals
    signals.sinkTo(
        KafkaSink.<EngagementSignal>builder()
            .setBootstrapServers(bootstrap)
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("m9-engagement-signals")
            .setRecordSerializer(
                KafkaRecordSerializationSchema.<EngagementSignal>builder()
                    .setTopic(KafkaTopics.FLINK_ENGAGEMENT_SIGNALS.getTopicName())
                    .setValueSerializationSchema(new JsonSerializer<EngagementSignal>())
                    .build())
            .build()
    ).name("Sink: Engagement Signals");

    // Side output → alerts.engagement-drop
    signals.getSideOutput(Module9_EngagementMonitor.ENGAGEMENT_DROP_TAG).sinkTo(
        KafkaSink.<EngagementDropAlert>builder()
            .setBootstrapServers(bootstrap)
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("m9-engagement-drop-alerts")
            .setRecordSerializer(
                KafkaRecordSerializationSchema.<EngagementDropAlert>builder()
                    .setTopic(KafkaTopics.ALERTS_ENGAGEMENT_DROP.getTopicName())
                    .setValueSerializationSchema(new JsonSerializer<EngagementDropAlert>())
                    .build())
            .build()
    ).name("Sink: Engagement Drop Alerts");

    env.execute("CardioFit Module 9: Engagement Monitor");
}
```

---

## Task 9: Integration Test — Module9ProcessElementTest

- [ ] Create `operators/Module9ProcessElementTest.java`:

Validates the full wiring: processElement() → bitmap update → onTimer() → EngagementSignal emission → side output EngagementDropAlert emission.

Test scenarios:
1. **Single engaged day (3 signals):** Send med_admin + bp_reading + meal_log → trigger timer → verify composite score = (1/14 * 0.15) + (1/14 * 0.20) + (1/14 * 0.15) = 0.0357, level=RED (CORPORATE)
2. **14 days fully engaged:** Set fullyEngagedState → trigger timer → score = 1.0, level = GREEN, phenotype = STEADY
3. **Engagement decline:** Set decliningState → trigger timer → verify DECLINING phenotype
4. **Drop alert with 3-day persistence (R3):** Set state at YELLOW → inject 3 consecutive days at ORANGE → verify LEVEL_TRANSITION alert fires on day 3 (not day 1)
5. **RED transition fires immediately (R3):** Set state at ORANGE → inject RED score → verify alert fires on day 1 (no persistence)
6. **Cliff drop:** Set previousScore = 0.70, current computes to 0.35 → verify CLIFF_DROP alert (fires immediately)
7. **Timer chaining:** Verify that after onTimer fires, next timer is registered for next day 23:59 UTC
8. **No double-count:** Send 5 BP readings in same day → verify S2 bitmap has single true (not 5)
9. **Glucose lab classification (R1):** Send LAB_RESULT with lab_type=glucose → verify GLUCOSE_MONITORING marked. Send LAB_RESULT with lab_type=creatinine → verify no signal marked.
10. **MEDICATION_MISSED does NOT count as adherence:** Send MEDICATION_MISSED → verify MEDICATION_ADHERENCE remains false for today
11. **Channel-aware scoring (R2):** Set channel="GOVERNMENT" with score 0.35 → verify level=YELLOW (not ORANGE as in CORPORATE)
12. **Zombie termination (R4):** Set lastUpdated to 22 days ago → trigger timer → verify DISENGAGED_TERMINATED signal emitted, no timer re-registered
13. **data_tier extraction (R6):** Send event with payload.data_tier="TIER_1_CGM" → verify state.dataTier set, signal.dataTier populated
14. **Weight measurement (R1):** Send VITAL_SIGN with weight only → verify WEIGHT_MEASUREMENT. Send with both systolic_bp and weight → verify BP_MEASUREMENT (priority)
15. **CGM device reading (R1):** Send DEVICE_READING with data_tier=TIER_1_CGM → verify GLUCOSE_MONITORING

---

## Task 10: Daily Timer Registration Test

- [ ] Create `operators/Module9DailyTimerTest.java`:

1. Verify `computeNextDailyTick(10:00 UTC)` returns today 23:59 UTC
2. Verify `computeNextDailyTick(23:59 UTC)` returns tomorrow 23:59 UTC
3. Verify `computeNextDailyTick(00:00 UTC)` returns today 23:59 UTC
4. Verify timer fires exactly once per day per patient (not per event)
5. Verify timer chain continues even when no events arrive (processing time timer doesn't depend on event flow)

---

## Phase 2: Enhancement 6 — Early Relapse Prediction (Deferred)

### P2.1 New Kafka Topic

- [ ] Add to `KafkaTopics.java`:
```java
ALERTS_RELAPSE_RISK("alerts.relapse-risk", 4, 90),
```

### P2.2 Trajectory Analyzer

- [ ] Create `operators/Module9TrajectoryAnalyzer.java`:

Uses `compositeHistory14d` (already in EngagementState) + 5 per-signal trajectory buffers to compute 7-day OLS regression slopes.

**Five trajectory features:**
1. Steps slope (from wearable aggregates → SignalType extension)
2. Meal quality slope (from meal log carb/protein ratio)
3. Response latency slope (from app session response time)
4. Check-in completeness slope (from number of fields filled per check-in)
5. Protein adherence slope (from meal log protein flag)

**Expert weights (initial):** steps 0.30, latency 0.25, meal 0.20, check-in 0.15, protein 0.10

**Relapse risk score:** Weighted sum of negative slopes → 0.0 (stable) to 1.0 (high relapse risk)

### P2.3 SPAN Integration

Module 9's `EngagementSignal` already has the `relapseRiskScore` field (nullable). Phase 2 populates it. SPAN consumes `flink.engagement-signals` and uses:
- `engagementLevel` → delivery frequency
- `phenotype` → messaging tone
- `relapseRiskScore` → intervention intensity

### P2.4 BCE Integration

`alerts.relapse-risk` consumed by BCE:
- MODERATE risk (0.40-0.69) → Recovery motivation phase, reduced targets
- HIGH risk (>=0.70) → KB-23 Decision Card + physician outreach call

---

## Risk Register

| Risk | Severity | Mitigation |
|------|----------|------------|
| `PATIENT_REPORTED.report_type` field name mismatch | HIGH | Task 4 classifier is first implementation target. Verify against actual ingestion-service payload schema before proceeding. |
| Channel defaults to CORPORATE until KB-20 integration | MEDIUM | Phase 1 uses hardcoded CORPORATE. Government/ACCHS patients may receive false-positive alerts until Phase 1b broadcast state integration. Monitor alert rates by patient cohort. |
| `app.session-events` topic may not exist | MEDIUM | APP_SESSION signal (weight 0.05) defaults to 0.0 density. Lowest-weighted signal by design. |
| CGM patients flooding processElement() | LOW | processElement() does O(1) work (classify + set bit). No scoring per event. |
| Timer drift under backpressure | LOW | Processing-time timers in Flink are guaranteed to fire once backpressure clears. Scoring will be delayed but not lost. |
| Zombie state accumulation (R4) | LOW (mitigated) | 21-day staleness check terminates timer chain. Emits DISENGAGED_TERMINATED signal before stopping. Tracked via `module9.zombie_patients_terminated` counter. |
| 3-day persistence delays legitimate RED alerts | LOW | RED transitions fire immediately (patient safety override). Only ORANGE-level transitions require 3-day persistence per DD#8 Section 4.1. |

---

## Observability

Module 9 emits (per architecture doc Section 6.5):
- `module9.events_in` — counter by event type
- `module9.events_classified` — counter by signal type
- `module9.events_unclassified` — counter (events with no signal mapping)
- `module9.daily_scores_emitted` — counter
- `module9.engagement_drops_emitted` — counter by drop type
- `module9.zombie_patients_terminated` — counter (R4: patients hitting 21-day staleness)
- `module9.composite_score` — gauge (latest per patient, sampled)
- `module9.processing_latency_ms` — histogram (processElement only)
- `module9.timer_latency_ms` — histogram (onTimer, should be < 10ms)

---

*Plan version: 2.0 | Created: 2 April 2026 | Updated: 2 April 2026 (R1-R7 review fixes) | Based on Vaidshala V4 Flink Architecture v2.0, DD#8 Engagement Monitor, KB-21 engagement service, Module 8 patterns*

**Review fixes applied (v2.0):**
- **R1:** 6→8 signals, glucose monitoring (S1) and weight measurement (S6) restored from DD#8
- **R2:** Channel-aware thresholds (CORPORATE/GOVERNMENT/ACCHS) — EngagementChannel enum
- **R3:** 3-day persistence for LEVEL_TRANSITION alerts (DD#8 Section 4.1), RED fires immediately
- **R4:** Zombie state prevention — 21-day staleness check terminates timer chain
- **R5:** Flink 2.x `open(OpenContext)` signature (not Configuration)
- **R6:** data_tier/channel extraction in processElement() from event payload
- **R7:** validHistoryDays counter replaces >0 proxy in phenotype classification
