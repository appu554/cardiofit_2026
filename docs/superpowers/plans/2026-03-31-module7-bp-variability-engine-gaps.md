# Module 7: BP Variability Engine — Remaining Gaps Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the Module 7 BP Variability Engine by implementing the 5 static analyzers (ARV, surge, dipping, crisis, BP control), main operator, orchestrator wiring, and all tests. The data foundation (10 model/enum files) is already implemented.

**Architecture:** Module 7 is a `KeyedProcessFunction<String, BPReading, BPVariabilityMetrics>` keyed by `patientId`, consuming raw BP readings from `ingestion.vitals` and `ingestion.clinic-bp`. Five static analyzer classes (zero Flink dependencies) compute clinically validated metrics. The main operator wires them together with Flink keyed state, crisis side-output, and Kafka sinks. All threshold comparisons use `>= threshold - 1e-9` for IEEE 754 safety.

**Tech Stack:** Java 17, Flink 2.1.0, Jackson 2.17, Kafka (Confluent Cloud), JUnit 5

---

## Already Implemented (Tasks 0–4)

These 10 files exist on disk and are verified correct. **Do not recreate them.**

| File | Purpose |
|------|---------|
| `src/main/java/com/cardiofit/flink/models/TimeContext.java` | Enum: MORNING/AFTERNOON/EVENING/NIGHT/UNKNOWN + `fromHour()`, `isDaytime()`, `isNocturnal()` |
| `src/main/java/com/cardiofit/flink/models/BPSource.java` | Enum: HOME_CUFF/CLINIC/CUFFLESS/UNKNOWN + `isClinicalGrade()` |
| `src/main/java/com/cardiofit/flink/models/VariabilityClassification.java` | Enum: LOW/MODERATE/ELEVATED/HIGH/INSUFFICIENT_DATA + `fromARV(Double)` |
| `src/main/java/com/cardiofit/flink/models/DipClassification.java` | Enum: DIPPER/NON_DIPPER/EXTREME_DIPPER/REVERSE_DIPPER/INSUFFICIENT_DATA + `fromDipRatio(double)` |
| `src/main/java/com/cardiofit/flink/models/SurgeClassification.java` | Enum: NORMAL/ELEVATED/HIGH/INSUFFICIENT_DATA + `fromSurge(Double)` |
| `src/main/java/com/cardiofit/flink/models/BPControlStatus.java` | Enum: CONTROLLED/ELEVATED/STAGE_1_UNCONTROLLED/STAGE_2_UNCONTROLLED/CRISIS + `fromAverages(double, double)` |
| `src/main/java/com/cardiofit/flink/models/BPReading.java` | Canonical input model with `isValid()`, `resolveTimeContext()`, `resolveSource()` |
| `src/main/java/com/cardiofit/flink/models/DailyBPSummary.java` | Per-day aggregation with morning/evening/nocturnal/clinic/home partitions |
| `src/main/java/com/cardiofit/flink/models/PatientBPState.java` | 30-day rolling `LinkedHashMap`, auto-eviction, `getContextDepth()` |
| `src/main/java/com/cardiofit/flink/models/BPVariabilityMetrics.java` | Full output model (25+ fields, all String-typed enums for Kafka serialization) |

### Critical Note: BPVariabilityMetrics Field Types

The on-disk `BPVariabilityMetrics.java` uses **String-typed fields** for enum values (e.g., `String variabilityClassification7d`, `String bpControlStatus`) — this is intentional for Kafka serialization. The original plan document shows typed enums (e.g., `VariabilityClassification variabilityClassification`). **Follow the on-disk version.** The main operator must call `.name()` on enum results before setting fields on BPVariabilityMetrics.

Key field name mappings (plan → on-disk):
- `meanSBP7d` → `sbp7dAvg` (renamed to match downstream consumer contracts)
- `meanDBP7d` → `dbp7dAvg`
- `VariabilityClassification variabilityClassification` → `String variabilityClassification7d`
- `BPSource triggerSource` → `String triggerSource`
- `TimeContext triggerTimeContext` → `String triggerTimeContext`
- `DipClassification dipClassification` → `String dipClassification`
- `SurgeClassification surgeClassification` → `String surgeClassification`
- `BPControlStatus bpControlStatus` → `String bpControlStatus`
- `crisisDetected` → `crisisFlag`
- `whiteCoatSuspect` → `whiteCoatSuspected`
- `maskedHtnSuspect` → `maskedHTNSuspected`
- `clinicHomeDelta` → `clinicHomeGapSBP`
- `timestamp` → `computedAt`

### Existing Infrastructure (Do NOT modify)

| File | Status | Notes |
|------|--------|-------|
| `src/main/java/com/cardiofit/flink/utils/KafkaTopics.java` | **KEEP** | Already has `FLINK_BP_VARIABILITY_METRICS` (line 169), `INGESTION_SAFETY_CRITICAL` (line 158), `INGESTION_VITALS` (line 150) |
| `src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java` | **MODIFY** | Lines 89-91 already have `case "bp-variability": case "module7": Module7_BPVariability.createBPVariabilityPipeline(env);` — re-point to new class |
| `src/main/java/com/cardiofit/flink/operators/Module7_BPVariability.java` | **REPLACE** | V3 monolith — CSV-encoded state, `Map<String,Object>` input. Will be superseded by `Module7_BPVariabilityEngine.java` |

---

## File Structure (New Files Only)

```
backend/shared-infrastructure/flink-processing/src/
├── main/java/com/cardiofit/flink/operators/
│   ├── Module7ARVComputer.java            ← Task 2: Static ARV/SD/CV computation
│   ├── Module7SurgeDetector.java          ← Task 3: Morning surge (sleep-trough method)
│   ├── Module7DipClassifier.java          ← Task 4: Nocturnal dipping classification
│   ├── Module7CrisisDetector.java         ← Task 5: Crisis bypass (SBP>180/DBP>120)
│   ├── Module7BPControlClassifier.java    ← Task 6: Control status + white-coat/masked
│   └── Module7_BPVariabilityEngine.java   ← Task 8: Main KeyedProcessFunction
├── test/java/com/cardiofit/flink/
│   ├── builders/
│   │   └── Module7TestBuilder.java        ← Task 1: Test data factory
│   └── operators/
│       ├── Module7ARVComputationTest.java  ← Task 2: ARV tests
│       ├── Module7SurgeDetectionTest.java  ← Task 3: Surge tests
│       ├── Module7DipClassificationTest.java ← Task 4: Dipping tests
│       ├── Module7CrisisDetectionTest.java ← Task 5: Crisis tests
│       ├── Module7BPControlTest.java       ← Task 6: Control tests
│       ├── Module7WhiteCoatMaskedTest.java ← Task 6: White-coat/masked tests
│       └── Module7IntegrationTest.java     ← Task 9: Full-flow integration
```

---

## Task 1: Module7TestBuilder — Test Data Factory

**Files:**
- Create: `src/test/java/com/cardiofit/flink/builders/Module7TestBuilder.java`

Provides factory methods for 9 canonical patient scenarios. All subsequent TDD tasks depend on this.

- [ ] **Step 1: Create Module7TestBuilder**

```java
package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import java.time.*;
import java.util.*;

/**
 * Test data factory for Module 7 BP Variability Engine tests.
 * Provides patient scenarios with realistic BP reading patterns.
 */
public class Module7TestBuilder {

    // ── Single BPReading builders ──

    public static BPReading reading(String patientId, double sbp, double dbp,
                                     long timestamp, String timeContext, String source) {
        BPReading r = new BPReading();
        r.setPatientId(patientId);
        r.setSystolic(sbp);
        r.setDiastolic(dbp);
        r.setTimestamp(timestamp);
        r.setTimeContext(timeContext);
        r.setSource(source);
        return r;
    }

    public static BPReading morningReading(String patientId, double sbp, double dbp, long timestamp) {
        return reading(patientId, sbp, dbp, timestamp, "MORNING", "HOME_CUFF");
    }

    public static BPReading eveningReading(String patientId, double sbp, double dbp, long timestamp) {
        return reading(patientId, sbp, dbp, timestamp, "EVENING", "HOME_CUFF");
    }

    public static BPReading clinicReading(String patientId, double sbp, double dbp, long timestamp) {
        return reading(patientId, sbp, dbp, timestamp, "MORNING", "CLINIC");
    }

    public static BPReading crisisReading(String patientId) {
        return reading(patientId, 195.0, 125.0,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
    }

    // ── PatientBPState builders (pre-populated with multiple days) ──

    /**
     * Well-controlled patient: 7 days of stable readings.
     * Mean SBP ~122, ARV ~3, should classify as CONTROLLED + LOW variability.
     */
    public static PatientBPState controlledPatient(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        double[] sbps = {120, 124, 121, 123, 119, 125, 122};
        double[] dbps = {76, 78, 75, 77, 74, 79, 76};
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            // Morning reading
            state.addReading(morningReading(patientId, sbps[6-day], dbps[6-day], ts));
            // Evening reading (slightly lower — normal dipping)
            state.addReading(eveningReading(patientId, sbps[6-day] - 8, dbps[6-day] - 5,
                ts + 12 * 60 * 60 * 1000L));
        }
        return state;
    }

    /**
     * High variability patient: 7 days of unstable readings.
     * ARV ~18, should classify as HIGH variability.
     */
    public static PatientBPState highVariabilityPatient(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        // Oscillating SBP: big swings day to day
        double[] sbps = {125, 155, 118, 160, 122, 152, 128};
        double[] dbps = {78, 95, 74, 98, 76, 92, 80};
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            state.addReading(morningReading(patientId, sbps[6-day], dbps[6-day], ts));
        }
        return state;
    }

    /**
     * Non-dipper patient: nocturnal SBP does not drop (< 10% reduction).
     * 7 days with daytime ~141, nocturnal ~138 (dip ratio ~3% — NON_DIPPER).
     */
    public static PatientBPState nonDipperPatient(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            // Daytime: ~142
            state.addReading(morningReading(patientId, 142, 88, ts));
            state.addReading(reading(patientId, 140, 86,
                ts + 6 * 60 * 60 * 1000L, "AFTERNOON", "HOME_CUFF"));
            // Nocturnal: ~138 (only ~3% drop — NON_DIPPER)
            state.addReading(reading(patientId, 138, 85,
                ts + 18 * 60 * 60 * 1000L, "NIGHT", "HOME_CUFF"));
        }
        return state;
    }

    /**
     * Reverse dipper: nocturnal SBP > daytime SBP.
     * Highest CV risk pattern. Daytime ~134, nocturnal ~145.
     */
    public static PatientBPState reverseDipperPatient(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            // Daytime: ~135
            state.addReading(morningReading(patientId, 135, 82, ts));
            state.addReading(reading(patientId, 133, 80,
                ts + 6 * 60 * 60 * 1000L, "AFTERNOON", "HOME_CUFF"));
            // Nocturnal: ~145 (reverse dip — nocturnal > daytime)
            state.addReading(reading(patientId, 145, 90,
                ts + 18 * 60 * 60 * 1000L, "NIGHT", "HOME_CUFF"));
        }
        return state;
    }

    /**
     * Morning surge patient: large morning-evening differential.
     * Morning SBP ~155, previous evening ~118 → surge ~37 (HIGH).
     */
    public static PatientBPState morningSurgePatient(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            // Evening: low
            state.addReading(eveningReading(patientId, 118, 72,
                ts - 6 * 60 * 60 * 1000L)); // previous evening
            // Morning: high surge
            state.addReading(morningReading(patientId, 155, 92, ts));
        }
        return state;
    }

    /**
     * White-coat hypertension suspect: clinic BP > home BP by > 15 mmHg.
     * Home ~123, clinic ~150 → delta ~27.
     */
    public static PatientBPState whiteCoatSuspect(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            // Home readings: normal
            state.addReading(morningReading(patientId, 125, 78, ts));
            state.addReading(eveningReading(patientId, 122, 76, ts + 12 * 60 * 60 * 1000L));
        }
        // Add clinic readings: elevated (> 15 mmHg above home)
        long clinicTs = now - 2 * 24 * 60 * 60 * 1000L;
        state.addReading(clinicReading(patientId, 148, 92, clinicTs));
        state.addReading(clinicReading(patientId, 152, 94, clinicTs + 300_000));
        return state;
    }

    /**
     * Masked hypertension suspect: home BP > clinic BP by > 15 mmHg.
     * Home ~146, clinic ~127 → delta ~-19.
     */
    public static PatientBPState maskedHtnSuspect(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            // Home readings: elevated
            state.addReading(morningReading(patientId, 148, 92, ts));
            state.addReading(eveningReading(patientId, 145, 90, ts + 12 * 60 * 60 * 1000L));
        }
        // Clinic readings: normal (patient relaxed in clinic)
        long clinicTs = now - 2 * 24 * 60 * 60 * 1000L;
        state.addReading(clinicReading(patientId, 128, 78, clinicTs));
        state.addReading(clinicReading(patientId, 126, 76, clinicTs + 300_000));
        return state;
    }

    /**
     * Stage 2 uncontrolled: 7-day average SBP ~158.
     */
    public static PatientBPState stage2Uncontrolled(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        double[] sbps = {155, 162, 158, 160, 154, 163, 156};
        double[] dbps = {95, 98, 96, 97, 94, 99, 95};
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            state.addReading(morningReading(patientId, sbps[6-day], dbps[6-day], ts));
        }
        return state;
    }

    /**
     * Insufficient data: only 2 days of readings.
     * ARV and dipping should return INSUFFICIENT_DATA.
     */
    public static PatientBPState insufficientData(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        state.addReading(morningReading(patientId, 130, 82, now));
        state.addReading(morningReading(patientId, 128, 80, now - 24 * 60 * 60 * 1000L));
        return state;
    }
}
```

- [ ] **Step 2: Verify test compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test-compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/builders/Module7TestBuilder.java
git commit -m "test(module7): add Module7TestBuilder with 9 canonical BP patient scenarios"
```

---

## Task 2: Module7ARVComputer — Average Real Variability + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module7ARVComputer.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module7ARVComputationTest.java`

ARV = mean of |dailyAvg_i+1 - dailyAvg_i| for consecutive days. Better than SD for capturing day-to-day fluctuation without outlier inflation.

- [ ] **Step 1: Write ARV computation tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module7ARVComputationTest {

    @Test
    void controlledPatient_arvBelow8() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-CTRL");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Double arv = Module7ARVComputer.computeARV(summaries);
        assertNotNull(arv, "7 days of data should produce non-null ARV");
        assertTrue(arv < 8.0, "Controlled patient ARV should be LOW (< 8), got: " + arv);
    }

    @Test
    void highVariabilityPatient_arvAbove16() {
        PatientBPState state = Module7TestBuilder.highVariabilityPatient("P-HIGH-VAR");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Double arv = Module7ARVComputer.computeARV(summaries);
        assertNotNull(arv);
        assertTrue(arv >= 16.0, "High variability patient ARV should be HIGH (>= 16), got: " + arv);
    }

    @Test
    void insufficientData_returnsNull() {
        PatientBPState state = Module7TestBuilder.insufficientData("P-INSUFF");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Double arv = Module7ARVComputer.computeARV(summaries);
        assertNull(arv, "Fewer than 3 days should produce null ARV");
    }

    @Test
    void exactlyThreeDays_computesARV() {
        PatientBPState state = new PatientBPState("P-3DAY");
        long now = System.currentTimeMillis();
        // Day 1: avg SBP = 120, Day 2: avg SBP = 130, Day 3: avg SBP = 125
        state.addReading(Module7TestBuilder.morningReading("P-3DAY", 120, 78, now - 2 * 86400000L));
        state.addReading(Module7TestBuilder.morningReading("P-3DAY", 130, 84, now - 1 * 86400000L));
        state.addReading(Module7TestBuilder.morningReading("P-3DAY", 125, 80, now));
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, now);
        Double arv = Module7ARVComputer.computeARV(summaries);
        assertNotNull(arv);
        // ARV = (|130-120| + |125-130|) / 2 = (10 + 5) / 2 = 7.5
        assertEquals(7.5, arv, 0.01, "ARV for [120, 130, 125] should be 7.5");
    }

    @Test
    void sdAndCV_computeCorrectly() {
        PatientBPState state = new PatientBPState("P-SD");
        long now = System.currentTimeMillis();
        double[] sbps = {120, 130, 125, 135, 120};
        for (int i = 0; i < 5; i++) {
            state.addReading(Module7TestBuilder.morningReading("P-SD",
                sbps[i], 80, now - (4-i) * 86400000L));
        }
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, now);
        double mean = Module7ARVComputer.computeMeanSBP(summaries);
        double sd = Module7ARVComputer.computeSD(summaries);
        double cv = Module7ARVComputer.computeCV(summaries);
        assertEquals(126.0, mean, 0.1, "Mean of [120,130,125,135,120] should be 126");
        assertTrue(sd > 0, "SD should be positive");
        assertEquals(sd / mean, cv, 0.001, "CV = SD / mean");
    }

    @Test
    void thirtyDayWindow_usesAllDays() {
        PatientBPState state = new PatientBPState("P-30D");
        long now = System.currentTimeMillis();
        for (int day = 19; day >= 0; day--) {
            double sbp = 130 + (day % 2 == 0 ? 5 : -5); // alternating 135, 125
            state.addReading(Module7TestBuilder.morningReading("P-30D",
                sbp, 80, now - day * 86400000L));
        }
        List<DailyBPSummary> summaries7 = state.getSummariesInWindow(7, now);
        List<DailyBPSummary> summaries30 = state.getSummariesInWindow(30, now);
        assertTrue(summaries30.size() > summaries7.size(),
            "30-day window should contain more days than 7-day");
        Double arv30 = Module7ARVComputer.computeARV(summaries30);
        assertNotNull(arv30);
        // Each consecutive pair differs by 10 → ARV = 10.0
        assertEquals(10.0, arv30, 0.1, "Alternating 135/125 should produce ARV ~10");
    }

    @Test
    void variabilityClassification_matchesARV() {
        assertEquals(VariabilityClassification.LOW,
            VariabilityClassification.fromARV(5.0));
        assertEquals(VariabilityClassification.MODERATE,
            VariabilityClassification.fromARV(10.0));
        assertEquals(VariabilityClassification.ELEVATED,
            VariabilityClassification.fromARV(14.0));
        assertEquals(VariabilityClassification.HIGH,
            VariabilityClassification.fromARV(18.0));
        assertEquals(VariabilityClassification.INSUFFICIENT_DATA,
            VariabilityClassification.fromARV(null));
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module7ARVComputationTest -q 2>&1 | tail -10`
Expected: FAIL — `Module7ARVComputer` class does not exist yet

- [ ] **Step 3: Implement Module7ARVComputer**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.DailyBPSummary;
import java.util.List;

/**
 * Static ARV (Average Real Variability) computation.
 * No Flink dependencies — fully unit-testable.
 *
 * ARV = mean of |dailyAvg_i+1 - dailyAvg_i| for consecutive days with data.
 * Days without readings are skipped — they don't contribute to ARV.
 *
 * Minimum data requirement:
 *   7-day ARV: at least 3 daily averages
 *   30-day ARV: at least 7 daily averages
 */
public final class Module7ARVComputer {

    private Module7ARVComputer() {} // static utility

    /**
     * Compute ARV from ordered daily summaries.
     * @param summaries ordered list (oldest first) of DailyBPSummary
     * @return ARV in mmHg, or null if insufficient data (< 3 days)
     */
    public static Double computeARV(List<DailyBPSummary> summaries) {
        if (summaries == null || summaries.size() < 3) return null;

        double sumAbsDiff = 0;
        int pairs = 0;
        double prevAvg = summaries.get(0).getAvgSBP();

        for (int i = 1; i < summaries.size(); i++) {
            double currentAvg = summaries.get(i).getAvgSBP();
            sumAbsDiff += Math.abs(currentAvg - prevAvg);
            pairs++;
            prevAvg = currentAvg;
        }

        return pairs > 0 ? sumAbsDiff / pairs : null;
    }

    /** Compute mean SBP across daily averages. */
    public static double computeMeanSBP(List<DailyBPSummary> summaries) {
        return summaries.stream()
            .mapToDouble(DailyBPSummary::getAvgSBP)
            .average()
            .orElse(0.0);
    }

    /** Compute mean DBP across daily averages. */
    public static double computeMeanDBP(List<DailyBPSummary> summaries) {
        return summaries.stream()
            .mapToDouble(DailyBPSummary::getAvgDBP)
            .average()
            .orElse(0.0);
    }

    /** Compute sample standard deviation of daily SBP averages. */
    public static double computeSD(List<DailyBPSummary> summaries) {
        if (summaries.size() < 2) return 0.0;
        double mean = computeMeanSBP(summaries);
        double sumSqDiff = summaries.stream()
            .mapToDouble(s -> {
                double diff = s.getAvgSBP() - mean;
                return diff * diff;
            })
            .sum();
        return Math.sqrt(sumSqDiff / (summaries.size() - 1)); // sample SD
    }

    /** Compute coefficient of variation (SD / mean). */
    public static double computeCV(List<DailyBPSummary> summaries) {
        double mean = computeMeanSBP(summaries);
        if (mean < 1e-9) return 0.0;
        return computeSD(summaries) / mean;
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module7ARVComputationTest -q 2>&1 | tail -15`
Expected: All 7 tests PASS

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module7ARVComputer.java \
  src/test/java/com/cardiofit/flink/operators/Module7ARVComputationTest.java
git commit -m "feat(module7): implement ARV computation with SD, CV, and variability classification"
```

---

## Task 3: Module7SurgeDetector — Morning Surge + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module7SurgeDetector.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module7SurgeDetectionTest.java`

Morning surge = morning_SBP_today - evening_SBP_yesterday (sleep-trough method, Kario et al.). More predictive than pre-waking method and feasible with twice-daily home readings.

- [ ] **Step 1: Write surge detection tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module7SurgeDetectionTest {

    @Test
    void morningSurgePatient_classifiesHigh() {
        PatientBPState state = Module7TestBuilder.morningSurgePatient("P-SURGE");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Double surge7dAvg = Module7SurgeDetector.compute7DayAvgSurge(summaries);
        assertNotNull(surge7dAvg);
        assertTrue(surge7dAvg >= 35.0 - 1e-9,
            "Morning surge ~37 should be >= 35, got: " + surge7dAvg);
        assertEquals(SurgeClassification.HIGH,
            SurgeClassification.fromSurge(surge7dAvg));
    }

    @Test
    void controlledPatient_surgeNormal() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-CTRL");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Double surge7dAvg = Module7SurgeDetector.compute7DayAvgSurge(summaries);
        // Controlled patient: morning ~122, evening ~114, surge ~8
        if (surge7dAvg != null) {
            assertTrue(surge7dAvg < 20.0,
                "Controlled patient surge should be NORMAL (< 20), got: " + surge7dAvg);
            assertEquals(SurgeClassification.NORMAL,
                SurgeClassification.fromSurge(surge7dAvg));
        }
    }

    @Test
    void todaySurge_computedFromMorningMinusPreviousEvening() {
        PatientBPState state = new PatientBPState("P-TODAY");
        long now = System.currentTimeMillis();
        long yesterday = now - 24 * 60 * 60 * 1000L;
        // Yesterday evening: SBP 115
        state.addReading(Module7TestBuilder.eveningReading("P-TODAY", 115, 72, yesterday + 12 * 3600000L));
        // Today morning: SBP 148
        state.addReading(Module7TestBuilder.morningReading("P-TODAY", 148, 88, now));

        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, now);
        Double todaySurge = Module7SurgeDetector.computeTodaySurge(summaries, now);
        assertNotNull(todaySurge);
        assertEquals(33.0, todaySurge, 1.0, "Surge should be ~148 - 115 = 33");
    }

    @Test
    void noEveningReading_surgeIsNull() {
        PatientBPState state = new PatientBPState("P-NO-EVE");
        long now = System.currentTimeMillis();
        state.addReading(Module7TestBuilder.morningReading("P-NO-EVE", 140, 85, now));
        state.addReading(Module7TestBuilder.morningReading("P-NO-EVE", 138, 84, now - 86400000L));

        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, now);
        Double todaySurge = Module7SurgeDetector.computeTodaySurge(summaries, now);
        assertNull(todaySurge, "No evening reading means surge cannot be computed");
    }

    @Test
    void surgeClassification_boundaries() {
        assertEquals(SurgeClassification.NORMAL, SurgeClassification.fromSurge(15.0));
        assertEquals(SurgeClassification.ELEVATED, SurgeClassification.fromSurge(25.0));
        assertEquals(SurgeClassification.HIGH, SurgeClassification.fromSurge(40.0));
        assertEquals(SurgeClassification.INSUFFICIENT_DATA, SurgeClassification.fromSurge(null));
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module7SurgeDetectionTest -q 2>&1 | tail -10`
Expected: FAIL — `Module7SurgeDetector` does not exist

- [ ] **Step 3: Implement Module7SurgeDetector**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.DailyBPSummary;
import java.time.Instant;
import java.time.ZoneOffset;
import java.util.List;

/**
 * Morning BP surge detection using the sleep-trough method.
 *
 * Surge = morning_SBP_today - evening_SBP_yesterday
 *
 * Per Kario et al. (J-HOP study), morning surge > 25 mmHg is an
 * independent stroke predictor. JSH 2025 "Asakatsu BP" targets
 * morning home BP < 130/80.
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module7SurgeDetector {

    private Module7SurgeDetector() {}

    /**
     * Compute today's morning surge (morning SBP - previous evening SBP).
     * @param summaries ordered daily summaries (oldest first)
     * @param referenceTime current event time
     * @return surge in mmHg, or null if insufficient data
     */
    public static Double computeTodaySurge(List<DailyBPSummary> summaries, long referenceTime) {
        if (summaries == null || summaries.size() < 2) return null;

        String todayKey = Instant.ofEpochMilli(referenceTime)
            .atZone(ZoneOffset.UTC).toLocalDate().toString();
        String yesterdayKey = Instant.ofEpochMilli(referenceTime)
            .atZone(ZoneOffset.UTC).toLocalDate().minusDays(1).toString();

        DailyBPSummary today = null;
        DailyBPSummary yesterday = null;

        for (DailyBPSummary s : summaries) {
            if (todayKey.equals(s.getDateKey())) today = s;
            if (yesterdayKey.equals(s.getDateKey())) yesterday = s;
        }

        if (today == null || today.getMorningAvgSBP() == null) return null;
        if (yesterday == null || yesterday.getEveningAvgSBP() == null) return null;

        return today.getMorningAvgSBP() - yesterday.getEveningAvgSBP();
    }

    /**
     * Compute 7-day average morning surge across days that have both readings.
     * @param summaries ordered daily summaries (oldest first)
     * @return 7-day average surge, or null if fewer than 3 valid pairs
     */
    public static Double compute7DayAvgSurge(List<DailyBPSummary> summaries) {
        if (summaries == null || summaries.size() < 2) return null;

        double sumSurge = 0;
        int validPairs = 0;

        for (int i = 1; i < summaries.size(); i++) {
            DailyBPSummary today = summaries.get(i);
            DailyBPSummary prevDay = summaries.get(i - 1);

            if (today.getMorningAvgSBP() != null && prevDay.getEveningAvgSBP() != null) {
                sumSurge += today.getMorningAvgSBP() - prevDay.getEveningAvgSBP();
                validPairs++;
            }
        }

        if (validPairs < 3) return null; // insufficient paired data
        return sumSurge / validPairs;
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module7SurgeDetectionTest -q 2>&1 | tail -10`
Expected: All 5 tests PASS

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module7SurgeDetector.java \
  src/test/java/com/cardiofit/flink/operators/Module7SurgeDetectionTest.java
git commit -m "feat(module7): implement morning surge detection with sleep-trough method"
```

---

## Task 4: Module7DipClassifier — Dipping Pattern + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module7DipClassifier.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module7DipClassificationTest.java`

Dip ratio = 1 - (nocturnal_mean / daytime_mean). Non-dippers have 2-3x higher CV event rates (MAPEC study). Requires 3+ days with both daytime AND nocturnal readings.

- [ ] **Step 1: Write dipping classification tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module7DipClassificationTest {

    @Test
    void controlledPatient_hasResult() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-DIP");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Module7DipClassifier.DipResult result = Module7DipClassifier.classify(summaries);
        // controlledPatient has morning + evening but no NIGHT readings
        // so dipping data may be insufficient
        assertNotNull(result);
    }

    @Test
    void nonDipperPatient_classifiesAsNonDipper() {
        PatientBPState state = Module7TestBuilder.nonDipperPatient("P-NONDIP");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Module7DipClassifier.DipResult result = Module7DipClassifier.classify(summaries);
        assertNotNull(result);
        assertNotNull(result.dipRatio());
        assertEquals(DipClassification.NON_DIPPER, result.classification(),
            "Dip ratio ~3% should be NON_DIPPER, got ratio: " + result.dipRatio());
    }

    @Test
    void reverseDipperPatient_classifiesAsReverseDipper() {
        PatientBPState state = Module7TestBuilder.reverseDipperPatient("P-REVDIP");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Module7DipClassifier.DipResult result = Module7DipClassifier.classify(summaries);
        assertNotNull(result);
        assertNotNull(result.dipRatio());
        assertTrue(result.dipRatio() < 0, "Reverse dipper should have negative dip ratio");
        assertEquals(DipClassification.REVERSE_DIPPER, result.classification());
    }

    @Test
    void insufficientData_returnsInsufficientClassification() {
        PatientBPState state = Module7TestBuilder.insufficientData("P-INSUFF");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Module7DipClassifier.DipResult result = Module7DipClassifier.classify(summaries);
        assertEquals(DipClassification.INSUFFICIENT_DATA, result.classification());
    }

    @Test
    void dipRatio_boundaries() {
        assertEquals(DipClassification.REVERSE_DIPPER, DipClassification.fromDipRatio(-0.05));
        assertEquals(DipClassification.NON_DIPPER, DipClassification.fromDipRatio(0.05));
        assertEquals(DipClassification.DIPPER, DipClassification.fromDipRatio(0.15));
        assertEquals(DipClassification.EXTREME_DIPPER, DipClassification.fromDipRatio(0.25));
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module7DipClassificationTest -q 2>&1 | tail -10`
Expected: FAIL

- [ ] **Step 3: Implement Module7DipClassifier**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.DailyBPSummary;
import com.cardiofit.flink.models.DipClassification;
import java.util.List;

/**
 * Nocturnal dipping pattern classifier.
 *
 * Dip ratio = 1 - (nocturnal_mean_SBP / daytime_mean_SBP)
 *
 * Requires at least 3 days with both daytime AND nocturnal readings.
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module7DipClassifier {

    private Module7DipClassifier() {}

    /** Immutable result record for dipping analysis. */
    public record DipResult(DipClassification classification, Double dipRatio,
                            Double daytimeMean, Double nocturnalMean, int validDays) {}

    /**
     * Classify dipping pattern from daily summaries.
     * @param summaries ordered list of DailyBPSummary (oldest first)
     * @return DipResult with classification and metrics
     */
    public static DipResult classify(List<DailyBPSummary> summaries) {
        if (summaries == null || summaries.isEmpty()) {
            return new DipResult(DipClassification.INSUFFICIENT_DATA, null, null, null, 0);
        }

        double sumDaytime = 0;
        double sumNocturnal = 0;
        int daytimeDays = 0;
        int nocturnalDays = 0;

        for (DailyBPSummary s : summaries) {
            if (s.getDaytimeAvgSBP() != null && s.getDaytimeCount() > 0) {
                sumDaytime += s.getDaytimeAvgSBP();
                daytimeDays++;
            }
            if (s.getNocturnalAvgSBP() != null && s.getNocturnalCount() > 0) {
                sumNocturnal += s.getNocturnalAvgSBP();
                nocturnalDays++;
            }
        }

        // Require at least 3 days with both daytime and nocturnal data
        if (daytimeDays < 3 || nocturnalDays < 3) {
            return new DipResult(DipClassification.INSUFFICIENT_DATA,
                null, null, null, Math.min(daytimeDays, nocturnalDays));
        }

        double daytimeMean = sumDaytime / daytimeDays;
        double nocturnalMean = sumNocturnal / nocturnalDays;

        if (daytimeMean < 1e-9) {
            return new DipResult(DipClassification.INSUFFICIENT_DATA,
                null, daytimeMean, nocturnalMean, Math.min(daytimeDays, nocturnalDays));
        }

        double dipRatio = 1.0 - (nocturnalMean / daytimeMean);
        DipClassification classification = DipClassification.fromDipRatio(dipRatio);

        return new DipResult(classification, dipRatio, daytimeMean, nocturnalMean,
            Math.min(daytimeDays, nocturnalDays));
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module7DipClassificationTest -q 2>&1 | tail -10`
Expected: All 5 tests PASS

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module7DipClassifier.java \
  src/test/java/com/cardiofit/flink/operators/Module7DipClassificationTest.java
git commit -m "feat(module7): implement nocturnal dipping classification with MAPEC-validated thresholds"
```

---

## Task 5: Module7CrisisDetector — Crisis Bypass + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module7CrisisDetector.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module7CrisisDetectionTest.java`

Crisis detection is the highest-priority safety path. SBP > 180 or DBP > 120 bypasses all windowed computation. Evaluated BEFORE any other logic. Strict `>` (not `>=`).

- [ ] **Step 1: Write crisis detection tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module7CrisisDetectionTest {

    @Test
    void sbpAbove180_isCrisis() {
        BPReading reading = Module7TestBuilder.reading("P001", 185, 95,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertTrue(Module7CrisisDetector.isCrisis(reading),
            "SBP 185 must trigger crisis");
    }

    @Test
    void dbpAbove120_isCrisis() {
        BPReading reading = Module7TestBuilder.reading("P001", 165, 125,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertTrue(Module7CrisisDetector.isCrisis(reading),
            "DBP 125 must trigger crisis");
    }

    @Test
    void bothAboveThreshold_isCrisis() {
        BPReading reading = Module7TestBuilder.crisisReading("P001");
        assertTrue(Module7CrisisDetector.isCrisis(reading));
    }

    @Test
    void normalReading_notCrisis() {
        BPReading reading = Module7TestBuilder.reading("P001", 140, 88,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertFalse(Module7CrisisDetector.isCrisis(reading),
            "SBP 140 / DBP 88 is not crisis");
    }

    @Test
    void exactlyAt180_notCrisis() {
        BPReading reading = Module7TestBuilder.reading("P001", 180, 110,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertFalse(Module7CrisisDetector.isCrisis(reading),
            "SBP exactly 180 is borderline — threshold is > 180, not >= 180");
    }

    @Test
    void exactlyAt120DBP_notCrisis() {
        BPReading reading = Module7TestBuilder.reading("P001", 170, 120,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertFalse(Module7CrisisDetector.isCrisis(reading),
            "DBP exactly 120 is borderline — threshold is > 120, not >= 120");
    }

    @Test
    void acuteSurge_sbpJump30InOneHour() {
        BPReading previous = Module7TestBuilder.reading("P001", 135, 82,
            System.currentTimeMillis() - 45 * 60 * 1000L, "MORNING", "HOME_CUFF");
        BPReading current = Module7TestBuilder.reading("P001", 170, 95,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertTrue(Module7CrisisDetector.isAcuteSurge(previous, current),
            "SBP jump from 135 to 170 (35 mmHg) in 45 min should be acute surge");
    }

    @Test
    void acuteSurge_sbpJump20_notSurge() {
        BPReading previous = Module7TestBuilder.reading("P001", 135, 82,
            System.currentTimeMillis() - 45 * 60 * 1000L, "MORNING", "HOME_CUFF");
        BPReading current = Module7TestBuilder.reading("P001", 152, 88,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertFalse(Module7CrisisDetector.isAcuteSurge(previous, current),
            "SBP jump of 17 mmHg should not be acute surge");
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module7CrisisDetectionTest -q 2>&1 | tail -10`
Expected: FAIL

- [ ] **Step 3: Implement Module7CrisisDetector**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.BPReading;

/**
 * Hypertensive crisis detection. Evaluated FIRST on every reading,
 * before any windowed computation.
 *
 * Crisis thresholds (per ACC/AHA):
 *   SBP > 180 mmHg OR DBP > 120 mmHg
 *
 * Acute surge detection:
 *   SBP increase > 30 mmHg within < 1 hour
 *
 * Both conditions emit directly to ingestion.safety-critical
 * for immediate notification-service delivery.
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module7CrisisDetector {

    private Module7CrisisDetector() {}

    /** Strict inequality: > 180 / > 120 (not >=). */
    public static boolean isCrisis(BPReading reading) {
        if (reading == null || reading.getSystolic() == null || reading.getDiastolic() == null) {
            return false;
        }
        return reading.getSystolic() > 180.0 || reading.getDiastolic() > 120.0;
    }

    /**
     * Acute surge: SBP increase > 30 mmHg within < 1 hour.
     * Requires a previous reading for comparison.
     */
    public static boolean isAcuteSurge(BPReading previous, BPReading current) {
        if (previous == null || current == null) return false;
        if (previous.getSystolic() == null || current.getSystolic() == null) return false;

        long timeDeltaMs = current.getTimestamp() - previous.getTimestamp();
        if (timeDeltaMs <= 0 || timeDeltaMs > 60 * 60 * 1000L) return false; // > 1 hour apart

        double sbpDelta = current.getSystolic() - previous.getSystolic();
        return sbpDelta > 30.0;
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module7CrisisDetectionTest -q 2>&1 | tail -10`
Expected: All 8 tests PASS

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module7CrisisDetector.java \
  src/test/java/com/cardiofit/flink/operators/Module7CrisisDetectionTest.java
git commit -m "feat(module7): implement crisis detection — SBP>180/DBP>120 bypass + acute surge"
```

---

## Task 6: Module7BPControlClassifier + White-Coat/Masked HTN + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module7BPControlClassifier.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module7BPControlTest.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module7WhiteCoatMaskedTest.java`

BP control uses 7-day avg against home thresholds (5 mmHg lower than clinic per JSH 2025). White-coat: clinic > home by >15 mmHg. Masked HTN: home > clinic by >15 mmHg. Both require ≥2 clinic + ≥5 home readings.

- [ ] **Step 1: Write BP control tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module7BPControlTest {

    @Test
    void controlledPatient_classifiesControlled() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-CTRL");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        BPControlStatus status = Module7BPControlClassifier.classifyControl(summaries);
        assertEquals(BPControlStatus.CONTROLLED, status,
            "7-day avg SBP ~122 should be CONTROLLED");
    }

    @Test
    void stage2Patient_classifiesStage2() {
        PatientBPState state = Module7TestBuilder.stage2Uncontrolled("P-STG2");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        BPControlStatus status = Module7BPControlClassifier.classifyControl(summaries);
        assertEquals(BPControlStatus.STAGE_2_UNCONTROLLED, status,
            "7-day avg SBP ~158 should be STAGE_2");
    }

    @Test
    void controlStatusBoundaries() {
        assertEquals(BPControlStatus.CONTROLLED, BPControlStatus.fromAverages(125, 75));
        assertEquals(BPControlStatus.ELEVATED, BPControlStatus.fromAverages(132, 78));
        assertEquals(BPControlStatus.STAGE_1_UNCONTROLLED, BPControlStatus.fromAverages(138, 82));
        assertEquals(BPControlStatus.STAGE_2_UNCONTROLLED, BPControlStatus.fromAverages(150, 95));
    }

    @Test
    void dbpAlone_canTriggerStage() {
        assertEquals(BPControlStatus.STAGE_1_UNCONTROLLED,
            BPControlStatus.fromAverages(128, 87),
            "DBP 87 alone should trigger STAGE_1 even with controlled SBP");
    }
}
```

- [ ] **Step 2: Write white-coat/masked HTN tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module7WhiteCoatMaskedTest {

    @Test
    void whiteCoatSuspect_detected() {
        PatientBPState state = Module7TestBuilder.whiteCoatSuspect("P-WC");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(30, System.currentTimeMillis());
        Module7BPControlClassifier.WhiteCoatResult result =
            Module7BPControlClassifier.detectWhiteCoatMasked(summaries);
        assertTrue(result.whiteCoatSuspect(),
            "Clinic ~150 vs home ~123 = delta ~27 should trigger white-coat suspect");
        assertFalse(result.maskedHtnSuspect());
    }

    @Test
    void maskedHtnSuspect_detected() {
        PatientBPState state = Module7TestBuilder.maskedHtnSuspect("P-MH");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(30, System.currentTimeMillis());
        Module7BPControlClassifier.WhiteCoatResult result =
            Module7BPControlClassifier.detectWhiteCoatMasked(summaries);
        assertTrue(result.maskedHtnSuspect(),
            "Home ~146 vs clinic ~127 should trigger masked HTN suspect");
        assertFalse(result.whiteCoatSuspect());
    }

    @Test
    void neitherCondition_whenNoDelta() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-CTRL");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(30, System.currentTimeMillis());
        Module7BPControlClassifier.WhiteCoatResult result =
            Module7BPControlClassifier.detectWhiteCoatMasked(summaries);
        assertFalse(result.whiteCoatSuspect());
        assertFalse(result.maskedHtnSuspect());
    }
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module7BPControlTest,Module7WhiteCoatMaskedTest" -q 2>&1 | tail -10`
Expected: FAIL

- [ ] **Step 4: Implement Module7BPControlClassifier**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.List;

/**
 * BP control status classification and white-coat/masked HTN detection.
 *
 * BP control uses 7-day average SBP and DBP against home BP thresholds
 * (5 mmHg lower than clinic thresholds per JSH 2025).
 *
 * White-coat detection: clinic_avg_SBP - home_avg_SBP > 15 mmHg.
 * Masked HTN detection: home_avg_SBP - clinic_avg_SBP > 15 mmHg.
 * Both require >= 2 clinic readings and >= 5 home readings in 30 days.
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module7BPControlClassifier {

    private Module7BPControlClassifier() {}

    private static final double WHITE_COAT_THRESHOLD = 15.0; // mmHg

    public record WhiteCoatResult(boolean whiteCoatSuspect, boolean maskedHtnSuspect,
                                   Double clinicHomeDelta) {}

    /**
     * Classify BP control status from 7-day daily summaries.
     */
    public static BPControlStatus classifyControl(List<DailyBPSummary> summaries) {
        if (summaries == null || summaries.isEmpty()) return BPControlStatus.ELEVATED; // conservative

        double avgSBP = Module7ARVComputer.computeMeanSBP(summaries);
        double avgDBP = Module7ARVComputer.computeMeanDBP(summaries);

        return BPControlStatus.fromAverages(avgSBP, avgDBP);
    }

    /**
     * Detect white-coat and masked hypertension from clinic vs home readings.
     * Requires >= 2 clinic readings and >= 5 home readings.
     */
    public static WhiteCoatResult detectWhiteCoatMasked(List<DailyBPSummary> summaries) {
        if (summaries == null || summaries.isEmpty()) {
            return new WhiteCoatResult(false, false, null);
        }

        double sumClinic = 0;
        int clinicReadings = 0;
        double sumHome = 0;
        int homeReadings = 0;

        for (DailyBPSummary s : summaries) {
            if (s.getClinicAvgSBP() != null && s.getClinicCount() > 0) {
                sumClinic += s.getClinicAvgSBP() * s.getClinicCount();
                clinicReadings += s.getClinicCount();
            }
            if (s.getHomeAvgSBP() != null && s.getHomeCount() > 0) {
                sumHome += s.getHomeAvgSBP() * s.getHomeCount();
                homeReadings += s.getHomeCount();
            }
        }

        if (clinicReadings < 2 || homeReadings < 5) {
            return new WhiteCoatResult(false, false, null);
        }

        double clinicMean = sumClinic / clinicReadings;
        double homeMean = sumHome / homeReadings;
        double delta = clinicMean - homeMean;

        boolean whiteCoat = delta > WHITE_COAT_THRESHOLD;
        boolean masked = delta < -WHITE_COAT_THRESHOLD;

        return new WhiteCoatResult(whiteCoat, masked, delta);
    }
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module7BPControlTest,Module7WhiteCoatMaskedTest" -q 2>&1 | tail -15`
Expected: All 7 tests PASS

- [ ] **Step 6: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module7BPControlClassifier.java \
  src/test/java/com/cardiofit/flink/operators/Module7BPControlTest.java \
  src/test/java/com/cardiofit/flink/operators/Module7WhiteCoatMaskedTest.java
git commit -m "feat(module7): implement BP control classification and white-coat/masked HTN detection"
```

---

## Task 7: Compile Check — All Static Analyzers

Verify everything compiles together before building the main operator.

- [ ] **Step 1: Run full compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 2: Run all Module 7 unit tests so far**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module7ARVComputationTest,Module7SurgeDetectionTest,Module7DipClassificationTest,Module7CrisisDetectionTest,Module7BPControlTest,Module7WhiteCoatMaskedTest" 2>&1 | tail -20`
Expected: All ~32 tests PASS

---

## Task 8: Module7_BPVariabilityEngine — Main Operator

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module7_BPVariabilityEngine.java`

This `KeyedProcessFunction` wires all 5 static analyzers. It maintains `PatientBPState` in Flink keyed state with 31-day TTL, evaluates crisis first, then computes metrics.

**Important:** The on-disk `BPVariabilityMetrics.java` uses String-typed fields. All enum results must be converted via `.name()` before setting on the metrics object. Field names follow the on-disk version (see mapping table in "Already Implemented" section above).

- [ ] **Step 1: Create the main operator**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
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

/**
 * Module 7: BP Variability Engine — main operator.
 *
 * Keyed by patientId. On each BPReading:
 *   1. Validate reading (Rule 1, Rule 3)
 *   2. Crisis check (SBP>180 / DBP>120) → immediate side output
 *   3. Acute surge check (SBP jump >30 in <1hr) → side output
 *   4. Skip non-clinical-grade readings (cuffless) for main computation
 *   5. Update 30-day rolling state
 *   6. Recompute all metrics (ARV, surge, dipping, control status, white-coat/masked)
 *   7. Emit BPVariabilityMetrics to main output
 *
 * State TTL: 31 days (one extra day to prevent eviction edge case).
 */
public class Module7_BPVariabilityEngine
        extends KeyedProcessFunction<String, BPReading, BPVariabilityMetrics> {

    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(Module7_BPVariabilityEngine.class);

    // Side-output tag for crisis readings
    public static final OutputTag<BPReading> CRISIS_TAG =
        new OutputTag<>("safety-critical", TypeInformation.of(BPReading.class));

    // Flink keyed state
    private transient ValueState<PatientBPState> bpState;
    private transient ValueState<BPReading> lastReadingState; // for acute surge detection

    @Override
    public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
        super.open(openContext);

        // Patient BP state with 31-day TTL
        ValueStateDescriptor<PatientBPState> stateDesc =
            new ValueStateDescriptor<>("patient-bp-state", PatientBPState.class);
        StateTtlConfig ttl = StateTtlConfig
            .newBuilder(Duration.ofDays(31))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();
        stateDesc.enableTimeToLive(ttl);
        bpState = getRuntimeContext().getState(stateDesc);

        // Last reading for acute surge detection (short TTL)
        ValueStateDescriptor<BPReading> lastDesc =
            new ValueStateDescriptor<>("last-bp-reading", BPReading.class);
        StateTtlConfig shortTtl = StateTtlConfig
            .newBuilder(Duration.ofHours(2))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();
        lastDesc.enableTimeToLive(shortTtl);
        lastReadingState = getRuntimeContext().getState(lastDesc);

        LOG.info("Module7_BPVariabilityEngine initialized");
    }

    @Override
    public void processElement(BPReading reading, Context ctx,
                                Collector<BPVariabilityMetrics> out) throws Exception {

        // 1. Validate
        if (!reading.isValid()) {
            LOG.warn("Module7: dropping invalid BP reading for patient {}. SBP={}, DBP={}",
                reading.getPatientId(), reading.getSystolic(), reading.getDiastolic());
            return;
        }

        // 2. Crisis detection — ALWAYS first, even for cuffless
        if (Module7CrisisDetector.isCrisis(reading)) {
            LOG.warn("Module7: CRISIS for patient {}. SBP={}, DBP={}",
                reading.getPatientId(), reading.getSystolic(), reading.getDiastolic());
            ctx.output(CRISIS_TAG, reading);
        }

        // 3. Acute surge detection
        BPReading lastReading = lastReadingState.value();
        if (lastReading != null && Module7CrisisDetector.isAcuteSurge(lastReading, reading)) {
            LOG.warn("Module7: ACUTE SURGE for patient {}. delta={}",
                reading.getPatientId(),
                reading.getSystolic() - lastReading.getSystolic());
            ctx.output(CRISIS_TAG, reading);
        }
        lastReadingState.update(reading);

        // 4. Skip non-clinical-grade for main computation
        BPSource source = reading.resolveSource();
        if (!source.isClinicalGrade()) {
            LOG.debug("Module7: skipping non-clinical-grade reading (source={}) for patient {}",
                source, reading.getPatientId());
            return;
        }

        // 5. Update rolling state
        PatientBPState state = bpState.value();
        if (state == null) {
            state = new PatientBPState(reading.getPatientId());
        }
        state.addReading(reading);

        // 6. Compute all metrics
        BPVariabilityMetrics metrics = computeMetrics(reading, state);

        // 7. Emit and update state
        out.collect(metrics);
        bpState.update(state);
    }

    private BPVariabilityMetrics computeMetrics(BPReading reading, PatientBPState state) {
        long now = reading.getTimestamp();
        BPVariabilityMetrics m = new BPVariabilityMetrics();

        // Identity
        m.setPatientId(reading.getPatientId());
        m.setCorrelationId(reading.getCorrelationId());
        m.setComputedAt(now);
        m.setContextDepth(state.getContextDepth());

        // Trigger
        m.setTriggerSBP(reading.getSystolic());
        m.setTriggerDBP(reading.getDiastolic());
        m.setTriggerSource(reading.resolveSource().name());
        m.setTriggerTimeContext(reading.resolveTimeContext().name());
        m.setTriggerTimestamp(reading.getTimestamp());

        // 7-day window
        List<DailyBPSummary> window7 = state.getSummariesInWindow(7, now);
        m.setDaysWithDataIn7d(window7.size());
        if (window7.size() >= 3) {
            m.setSbp7dAvg(Module7ARVComputer.computeMeanSBP(window7));
            m.setDbp7dAvg(Module7ARVComputer.computeMeanDBP(window7));
            m.setSdSbp7d(Module7ARVComputer.computeSD(window7));
            m.setCvSbp7d(Module7ARVComputer.computeCV(window7));
            m.setArvSbp7d(Module7ARVComputer.computeARV(window7));
        }

        // 30-day window
        List<DailyBPSummary> window30 = state.getSummariesInWindow(30, now);
        m.setDaysWithDataIn30d(window30.size());
        if (window30.size() >= 7) {
            m.setArvSbp30d(Module7ARVComputer.computeARV(window30));
            m.setSdSbp30d(Module7ARVComputer.computeSD(window30));
            m.setCvSbp30d(Module7ARVComputer.computeCV(window30));
        }

        // Variability classification (7-day ARV is primary)
        m.setVariabilityClassification7d(VariabilityClassification.fromARV(m.getArvSbp7d()).name());
        if (m.getArvSbp30d() != null) {
            m.setVariabilityClassification30d(VariabilityClassification.fromARV(m.getArvSbp30d()).name());
        }

        // BP control status
        m.setBpControlStatus(Module7BPControlClassifier.classifyControl(window7).name());

        // Morning surge
        m.setMorningSurgeToday(Module7SurgeDetector.computeTodaySurge(window7, now));
        m.setMorningSurge7dAvg(Module7SurgeDetector.compute7DayAvgSurge(window7));
        m.setSurgeClassification(SurgeClassification.fromSurge(m.getMorningSurge7dAvg()).name());

        // Dipping
        Module7DipClassifier.DipResult dipResult = Module7DipClassifier.classify(window7);
        m.setDipClassification(dipResult.classification().name());
        m.setDipRatio(dipResult.dipRatio());

        // White-coat / Masked HTN (use 30-day window for enough clinic visits)
        Module7BPControlClassifier.WhiteCoatResult wcResult =
            Module7BPControlClassifier.detectWhiteCoatMasked(window30);
        m.setWhiteCoatSuspected(wcResult.whiteCoatSuspect());
        m.setMaskedHTNSuspected(wcResult.maskedHtnSuspect());
        m.setClinicHomeGapSBP(wcResult.clinicHomeDelta());

        // Crisis
        m.setCrisisFlag(Module7CrisisDetector.isCrisis(reading));

        // Data quality
        m.setTotalReadingsInState(state.getTotalReadingsProcessed());

        return m;
    }
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module7_BPVariabilityEngine.java
git commit -m "feat(module7): implement BPVariabilityEngine KeyedProcessFunction with crisis bypass, ARV, surge, dipping, control"
```

---

## Task 9: Wire Module 7 into FlinkJobOrchestrator + Integration Tests

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java` (lines 89-91, line 391)
- Create: `src/test/java/com/cardiofit/flink/operators/Module7IntegrationTest.java`

- [ ] **Step 1: Read FlinkJobOrchestrator to confirm exact edit locations**

Run: `cd backend/shared-infrastructure/flink-processing && grep -n "Module7_BPVariability\|bp-variability\|module7" src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java`

Expected output:
```
89:            case "bp-variability":
90:            case "module7":
91:                Module7_BPVariability.createBPVariabilityPipeline(env);
391:            Module7_BPVariability.createBPVariabilityPipeline(env);
```

- [ ] **Step 2: Update the orchestrator import and switch case**

Replace the import:
```java
// Old:
import com.cardiofit.flink.operators.Module7_BPVariability;
// New:
import com.cardiofit.flink.operators.Module7_BPVariabilityEngine;
```

Replace lines 89-91 in the job-type switch:
```java
            case "bp-variability":
            case "module7":
            case "bp-variability-engine":
                launchBPVariabilityEngine(env);
                break;
```

Replace line 391 in the full-pipeline launcher:
```java
            LOG.info("Initializing Module 7: BP Variability Engine");
            launchBPVariabilityEngine(env);
```

Add the `launchBPVariabilityEngine` method (after existing launcher methods):
```java
    /**
     * Module 7: BP Variability Engine.
     * Consumes ingestion.vitals, keys by patientId, produces bp-variability-metrics
     * and safety-critical side output.
     */
    private static void launchBPVariabilityEngine(StreamExecutionEnvironment env) {
        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        // Kafka source: BPReading from ingestion.vitals
        KafkaSource<BPReading> source = KafkaSource
            .<BPReading>builder()
            .setBootstrapServers(bootstrap)
            .setTopics(KafkaTopics.INGESTION_VITALS.getTopicName())
            .setGroupId("flink-module7-bp-variability-v2")
            .setValueOnlyDeserializer(new BPReadingDeserializer())
            .build();

        SingleOutputStreamOperator<BPVariabilityMetrics> metrics = env
            .fromSource(source,
                WatermarkStrategy.<BPReading>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                    .withTimestampAssigner((r, ts) -> r.getTimestamp()),
                "Kafka Source: BP Readings")
            .keyBy(BPReading::getPatientId)
            .process(new Module7_BPVariabilityEngine())
            .uid("module7-bp-variability-engine")
            .name("Module 7: BP Variability Engine");

        // Main output → flink.bp-variability-metrics
        metrics.sinkTo(
            KafkaSink.<BPVariabilityMetrics>builder()
                .setBootstrapServers(bootstrap)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.FLINK_BP_VARIABILITY_METRICS.getTopicName())
                        .setValueSerializationSchema(new BPMetricsSerializer())
                        .build())
                .build());

        // Crisis side output → ingestion.safety-critical
        metrics.getSideOutput(Module7_BPVariabilityEngine.CRISIS_TAG).sinkTo(
            KafkaSink.<BPReading>builder()
                .setBootstrapServers(bootstrap)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.INGESTION_SAFETY_CRITICAL.getTopicName())
                        .setValueSerializationSchema(new BPReadingSerializer())
                        .build())
                .build());
    }
```

**Note:** `BPReadingDeserializer`, `BPReadingSerializer`, and `BPMetricsSerializer` are inner classes you need to add. They follow the same Jackson ObjectMapper pattern as the existing `Module7_BPVariability.BPMetricsSerializer`. Create them as static inner classes or add them to the orchestrator file.

```java
    static class BPReadingDeserializer implements DeserializationSchema<BPReading> {
        private transient ObjectMapper mapper;
        @Override public void open(InitializationContext ctx) {
            mapper = new ObjectMapper();
        }
        @Override public BPReading deserialize(byte[] bytes) throws java.io.IOException {
            return mapper.readValue(bytes, BPReading.class);
        }
        @Override public boolean isEndOfStream(BPReading r) { return false; }
        @Override public TypeInformation<BPReading> getProducedType() {
            return TypeInformation.of(BPReading.class);
        }
    }

    static class BPMetricsSerializer implements SerializationSchema<BPVariabilityMetrics> {
        private transient ObjectMapper mapper;
        @Override public void open(InitializationContext ctx) {
            mapper = new ObjectMapper();
        }
        @Override public byte[] serialize(BPVariabilityMetrics m) {
            try { return mapper.writeValueAsBytes(m); }
            catch (Exception e) { throw new RuntimeException("Serialize BPMetrics failed", e); }
        }
    }

    static class BPReadingSerializer implements SerializationSchema<BPReading> {
        private transient ObjectMapper mapper;
        @Override public void open(InitializationContext ctx) {
            mapper = new ObjectMapper();
        }
        @Override public byte[] serialize(BPReading r) {
            try { return mapper.writeValueAsBytes(r); }
            catch (Exception e) { throw new RuntimeException("Serialize BPReading failed", e); }
        }
    }
```

- [ ] **Step 3: Write integration test**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Integration tests for Module 7.
 * Validates full flow: reading → state → all 5 analyzers → classifications.
 */
class Module7IntegrationTest {

    @Test
    void controlledPatient_producesExpectedClassifications() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-CTRL");
        long now = System.currentTimeMillis();

        java.util.List<DailyBPSummary> window7 = state.getSummariesInWindow(7, now);
        assertTrue(window7.size() >= 3, "Should have at least 3 days of data");

        Double arv = Module7ARVComputer.computeARV(window7);
        assertNotNull(arv);
        assertEquals(VariabilityClassification.LOW, VariabilityClassification.fromARV(arv));

        BPControlStatus status = Module7BPControlClassifier.classifyControl(window7);
        assertEquals(BPControlStatus.CONTROLLED, status);
    }

    @Test
    void stage2WithHighVariability_producesExpectedOutput() {
        PatientBPState state = Module7TestBuilder.highVariabilityPatient("P-HV");
        long now = System.currentTimeMillis();

        java.util.List<DailyBPSummary> window7 = state.getSummariesInWindow(7, now);
        Double arv = Module7ARVComputer.computeARV(window7);
        assertNotNull(arv);
        assertEquals(VariabilityClassification.HIGH, VariabilityClassification.fromARV(arv),
            "ARV ~18 should be HIGH, got: " + arv);
    }

    @Test
    void crisisReading_detectedBeforeWindowed() {
        BPReading crisis = Module7TestBuilder.crisisReading("P-CRISIS");
        assertTrue(Module7CrisisDetector.isCrisis(crisis));
        assertTrue(crisis.isValid(), "Crisis reading should still be valid");
    }

    @Test
    void reverseDipper_fullPipeline() {
        PatientBPState state = Module7TestBuilder.reverseDipperPatient("P-REVDIP");
        long now = System.currentTimeMillis();

        java.util.List<DailyBPSummary> window7 = state.getSummariesInWindow(7, now);
        Module7DipClassifier.DipResult dip = Module7DipClassifier.classify(window7);
        assertEquals(DipClassification.REVERSE_DIPPER, dip.classification());

        BPControlStatus status = Module7BPControlClassifier.classifyControl(window7);
        assertNotEquals(BPControlStatus.CONTROLLED, status,
            "Reverse dipper with mean SBP ~138 should not be CONTROLLED");
    }

    @Test
    void whiteCoatAndMasked_cannotBothBeTrue() {
        PatientBPState wcState = Module7TestBuilder.whiteCoatSuspect("P-WC");
        java.util.List<DailyBPSummary> wcSummaries = wcState.getSummariesInWindow(30, System.currentTimeMillis());
        Module7BPControlClassifier.WhiteCoatResult wcResult =
            Module7BPControlClassifier.detectWhiteCoatMasked(wcSummaries);
        assertFalse(wcResult.whiteCoatSuspect() && wcResult.maskedHtnSuspect(),
            "Cannot be both white-coat AND masked");

        PatientBPState mhState = Module7TestBuilder.maskedHtnSuspect("P-MH");
        java.util.List<DailyBPSummary> mhSummaries = mhState.getSummariesInWindow(30, System.currentTimeMillis());
        Module7BPControlClassifier.WhiteCoatResult mhResult =
            Module7BPControlClassifier.detectWhiteCoatMasked(mhSummaries);
        assertFalse(mhResult.whiteCoatSuspect() && mhResult.maskedHtnSuspect(),
            "Cannot be both white-coat AND masked");
    }

    @Test
    void insufficientData_gracefulDegradation() {
        PatientBPState state = Module7TestBuilder.insufficientData("P-INSUFF");
        long now = System.currentTimeMillis();

        java.util.List<DailyBPSummary> window7 = state.getSummariesInWindow(7, now);
        Double arv = Module7ARVComputer.computeARV(window7);
        assertNull(arv, "Insufficient data should produce null ARV");
        assertEquals(VariabilityClassification.INSUFFICIENT_DATA,
            VariabilityClassification.fromARV(arv));

        Module7DipClassifier.DipResult dip = Module7DipClassifier.classify(window7);
        assertEquals(DipClassification.INSUFFICIENT_DATA, dip.classification());
    }
}
```

- [ ] **Step 4: Verify full compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 5: Run integration tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module7IntegrationTest -q 2>&1 | tail -15`
Expected: All 6 tests PASS

- [ ] **Step 6: Commit**

```bash
git add src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java \
  src/test/java/com/cardiofit/flink/operators/Module7IntegrationTest.java
git commit -m "feat(module7): wire BPVariabilityEngine into FlinkJobOrchestrator + integration tests"
```

---

## Task 10: Full Test Suite + Final Verification

- [ ] **Step 1: Run ALL Module 7 tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module7*" 2>&1 | tail -25`
Expected: All ~38 tests PASS across 7 test classes:
- Module7ARVComputationTest (7 tests)
- Module7SurgeDetectionTest (5 tests)
- Module7DipClassificationTest (5 tests)
- Module7CrisisDetectionTest (8 tests)
- Module7BPControlTest (4 tests)
- Module7WhiteCoatMaskedTest (3 tests)
- Module7IntegrationTest (6 tests)

- [ ] **Step 2: Run full project compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Verify no regressions in existing tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . 2>&1 | tail -25`
Expected: All existing tests still pass (Module 1-6 tests unaffected)

- [ ] **Step 4: Final commit if any fixes were needed**

```bash
git add -A
git commit -m "fix(module7): resolve any compilation or test issues from integration"
```

---

## Key Technical Decisions

| Decision | Rationale |
|----------|-----------|
| Static analyzers with zero Flink deps | Fully unit-testable without Flink harness. Mirrors Module 6 `Module6ActionClassifier` pattern. |
| Crisis detection BEFORE state update | Patient safety — SBP 195 must emit to safety-critical even if state is corrupted/absent. |
| 31-day state TTL (not 30) | Prevents edge case where reading on day 30 evicts data needed for 30-day ARV computation. |
| Skip cuffless from main computation | Architecture: "NOT clinical grade until validated." `BPSource.isClinicalGrade()` is the upgrade switch. |
| DailyBPSummary incremental aggregation | No windowed replay. Each reading updates running sums. Memory-efficient for 30 days. |
| White-coat uses 30-day window, control uses 7-day | White-coat needs enough clinic visits (infrequent). Control needs recent data for medication response. |
| Crisis threshold `>` not `>=` at 180/120 | Matches ACC/AHA hypertensive crisis definition (> 180/120). |
| Sleep-trough method for morning surge | Feasible with twice-daily readings. Pre-waking method deferred to cuffless extension. |
| Minimum 3 days for ARV, 3 paired days for dipping | Prevents noisy classifications from sparse data. Returns INSUFFICIENT_DATA below minimum. |
| String-typed enum fields in BPVariabilityMetrics | Kafka JSON serialization without custom serde. Downstream consumers parse strings. |

## Deferred (Not in Scope)

1. **Cuffless BP separate ARV** — reserved field in model, no computation yet
2. **Patient timezone handling** — `TimeContext.fromHour()` uses UTC; production needs patient-local timezone from KB-20
3. **Orthostatic hypotension** — `BPReading.position` captured but detection deferred
4. **KB-26 MHRI hemodynamic scoring** — happens in KB-26, not Module 7
5. **KB-22 deterioration event firing** — KB-22 consumer responsibility
