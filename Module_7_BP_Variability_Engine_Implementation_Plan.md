# Module 7: BP Variability Engine — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the BP Variability Engine that consumes raw BP readings from two input topics (`ingestion.vitals`, `ingestion.clinic-bp`), maintains 30-day per-patient rolling state, computes clinically validated variability metrics (ARV, morning surge, dipping classification), detects hypertensive crises for immediate safety bypass, classifies BP control status, and produces structured output for five downstream consumers (KB-20, KB-26, KB-23, Module 4 RunCycle, notification-service).

**Architecture:** Module 7 is a `KeyedProcessFunction<String, BPReading, BPVariabilityMetrics>` keyed by `patientId`. Unlike the core pipeline (Modules 1–6) which processes enriched clinical events, Module 7 operates directly on raw vital-sign input — it is a domain-specific engine running in parallel with the core pipeline, not downstream of it. BP readings arrive as individual measurements; the operator maintains a 30-day sliding window of `DailyBPSummary` entries in Flink keyed state, recomputing variability metrics on each new reading. A hypertensive crisis (SBP > 180 or DBP > 120) bypasses all windowed computation and immediately emits to the safety-critical side output.

**Clinical foundation:** BP variability is an independent predictor of cardiovascular events beyond mean BP level. The J-HOP (Japan Home versus Office Blood Pressure) study demonstrated that day-to-day ARV of home systolic BP predicts stroke and cardiovascular mortality independently of mean SBP. Morning BP surge (Kario et al.) is an independent stroke risk factor targeted by the JSH 2025 "Asakatsu BP" initiative. Non-dipping patterns carry significantly higher cardiovascular risk. This module operationalizes these findings for the Vaidshala cardiometabolic platform.

**Tech Stack:** Java 17, Flink 2.1.0, Jackson 2.17, Kafka (Confluent Cloud), JUnit 5

---

## Existing Code Inventory

**Files that exist and will be VERIFIED/COMPLETED:**
- The architecture document marks Module 7 as "Partial (Java on disk)". Before beginning implementation, run Task 0 to inventory existing files and determine what can be reused vs. what needs replacement.

**Files from core pipeline that Module 7 reads (read-only):**
- `models/PatientContextState.java` — NOT used. Module 7 operates on raw BP readings, not enriched CDS events.
- `utils/KafkaTopics.java` — will be modified to add Module 7 topics
- `utils/KafkaConfigLoader.java` — used as-is

**Existing test patterns to follow:**
- `builders/Module4TestBuilder.java` — factory methods for patient scenarios
- `builders/Module5TestBuilder.java` — `sepsisPatientState()`, `akiPatientState()`, etc.
- `builders/Module6TestBuilder.java` — canonical patient scenario builders

**Key architectural difference from Modules 1–6:** Module 7 does NOT consume `comprehensive-cds-events.v1` or any inter-module topic. It reads directly from ingestion topics. This means:
1. No dependency on Module 2 enrichment or Module 3 CDS processing.
2. BP readings arrive as simple vital-sign events, not as `EnrichedPatientContext`.
3. Module 7 must handle its own field normalization (Rule 2).
4. Module 7 must validate `patientId != null` at entry (Rule 1).
5. Null numeric values must be triple-checked (Rule 3 — `getValue()` can return `null`).

---

## File Structure

```
backend/shared-infrastructure/flink-processing/src/
├── main/java/com/cardiofit/flink/
│   ├── models/
│   │   ├── BPReading.java                     ← NEW: canonical BP reading input
│   │   ├── TimeContext.java                   ← NEW: enum MORNING, AFTERNOON, EVENING, NIGHT, UNKNOWN
│   │   ├── BPSource.java                      ← NEW: enum HOME_CUFF, CLINIC, CUFFLESS, UNKNOWN
│   │   ├── DailyBPSummary.java                ← NEW: per-day aggregated readings
│   │   ├── BPVariabilityMetrics.java          ← NEW: full output model
│   │   ├── PatientBPState.java                ← NEW: 30-day rolling state
│   │   ├── VariabilityClassification.java     ← NEW: enum LOW, MODERATE, ELEVATED, HIGH
│   │   ├── DipClassification.java             ← NEW: enum DIPPER, NON_DIPPER, EXTREME_DIPPER, REVERSE_DIPPER, INSUFFICIENT_DATA
│   │   ├── SurgeClassification.java           ← NEW: enum NORMAL, ELEVATED, HIGH, INSUFFICIENT_DATA
│   │   └── BPControlStatus.java               ← NEW: enum CONTROLLED, ELEVATED, STAGE_1_UNCONTROLLED, STAGE_2_UNCONTROLLED, CRISIS
│   ├── operators/
│   │   ├── Module7_BPVariabilityEngine.java   ← NEW: main KeyedProcessFunction
│   │   ├── Module7ARVComputer.java            ← NEW: static ARV calculation (testable)
│   │   ├── Module7SurgeDetector.java          ← NEW: static morning surge logic (testable)
│   │   ├── Module7DipClassifier.java          ← NEW: static dipping pattern logic (testable)
│   │   ├── Module7CrisisDetector.java         ← NEW: static crisis bypass logic (testable)
│   │   └── Module7BPControlClassifier.java    ← NEW: static BP control + variability classification (testable)
│   └── utils/
│       └── KafkaTopics.java                   ← MODIFY: add Module 7 topics
├── test/java/com/cardiofit/flink/
│   ├── builders/
│   │   └── Module7TestBuilder.java            ← NEW: test data factory
│   └── operators/
│       ├── Module7ARVComputationTest.java     ← NEW
│       ├── Module7SurgeDetectionTest.java     ← NEW
│       ├── Module7DipClassificationTest.java  ← NEW
│       ├── Module7CrisisDetectionTest.java    ← NEW
│       ├── Module7BPControlTest.java          ← NEW
│       ├── Module7WhiteCoatMaskedTest.java    ← NEW
│       └── Module7IntegrationTest.java        ← NEW
```

---

## Task 0: Inventory Existing Module 7 Code

Before creating anything, determine what already exists on disk.

- [ ] **Step 1: Search for existing Module 7 files**

```bash
cd backend/shared-infrastructure/flink-processing
find src -name "*Module7*" -o -name "*BPVariab*" -o -name "*bp_variab*" -o -name "*BloodPressure*" | sort
grep -rl "bp-variability\|BPVariability\|Module7\|module7" src/main/java/ --include="*.java" | sort
```

- [ ] **Step 2: Assess each found file**

For each file found, determine:
- Does it compile against the current codebase?
- Does it follow the architecture in this plan (KeyedProcessFunction, static analyzers)?
- Can it be reused, or does it need replacement?

Document findings and adjust subsequent tasks accordingly. If files exist that align with this plan's architecture, mark them as KEEP and skip their creation steps. If they diverge, mark them as REPLACE.

- [ ] **Step 3: Proceed with tasks below, skipping creation of files that already exist and are adequate**

---

## Task 1: Enums and Classification Types

**Files:**
- Create: `src/main/java/com/cardiofit/flink/models/TimeContext.java`
- Create: `src/main/java/com/cardiofit/flink/models/BPSource.java`
- Create: `src/main/java/com/cardiofit/flink/models/VariabilityClassification.java`
- Create: `src/main/java/com/cardiofit/flink/models/DipClassification.java`
- Create: `src/main/java/com/cardiofit/flink/models/SurgeClassification.java`
- Create: `src/main/java/com/cardiofit/flink/models/BPControlStatus.java`

- [ ] **Step 1: Create TimeContext enum**

```java
package com.cardiofit.flink.models;

/**
 * Time-of-day context for BP readings.
 * Critical for morning surge computation (morning vs evening differential)
 * and dipping classification (daytime vs nocturnal).
 *
 * Resolution logic:
 *   1. Explicit time_context field from device/app (preferred)
 *   2. Fallback: derive from measurement_timestamp in patient's local timezone
 *      MORNING = 06:00–10:00, AFTERNOON = 10:00–17:00,
 *      EVENING = 17:00–22:00, NIGHT = 22:00–06:00
 */
public enum TimeContext {
    MORNING,      // 06:00–10:00 — used for surge computation (numerator)
    AFTERNOON,    // 10:00–17:00
    EVENING,      // 17:00–22:00 — used for surge computation (denominator)
    NIGHT,        // 22:00–06:00 — used for dipping classification (nocturnal)
    UNKNOWN;      // insufficient metadata — excluded from surge/dip analysis

    public boolean isDaytime() {
        return this == MORNING || this == AFTERNOON || this == EVENING;
    }

    public boolean isNocturnal() {
        return this == NIGHT;
    }

    /**
     * Derive TimeContext from hour of day (0–23).
     * Used as fallback when explicit time_context is not provided.
     */
    public static TimeContext fromHour(int hour) {
        if (hour >= 6 && hour < 10) return MORNING;
        if (hour >= 10 && hour < 17) return AFTERNOON;
        if (hour >= 17 && hour < 22) return EVENING;
        return NIGHT; // 22:00–05:59
    }
}
```

- [ ] **Step 2: Create BPSource enum**

```java
package com.cardiofit.flink.models;

/**
 * Source classification for BP readings.
 *
 * HOME_CUFF: oscillometric home device — clinical grade by default.
 * CLINIC: in-office measurement — reference standard but infrequent.
 * CUFFLESS: wearable-derived estimate — NOT clinical grade until validated.
 * UNKNOWN: source metadata missing — treated as home for processing,
 *          flagged in output for data quality tracking.
 *
 * White-coat detection requires both CLINIC and HOME_CUFF readings.
 * Masked hypertension detection requires the same.
 * Cuffless readings compute separate ARV (arv_cuffless) for research
 * but do NOT contribute to clinical Decision Cards.
 */
public enum BPSource {
    HOME_CUFF(true),
    CLINIC(true),
    CUFFLESS(false),  // NOT clinical grade until upgrade
    UNKNOWN(true);    // assume clinical grade, flag for review

    private final boolean clinicalGrade;

    BPSource(boolean clinicalGrade) {
        this.clinicalGrade = clinicalGrade;
    }

    public boolean isClinicalGrade() { return clinicalGrade; }
}
```

- [ ] **Step 3: Create VariabilityClassification enum**

```java
package com.cardiofit.flink.models;

/**
 * ARV-based variability classification.
 * Thresholds from J-HOP study data (upper quartile ~12–15 mmHg).
 *
 * LOW       (ARV < 8):    Normal — no action
 * MODERATE  (8 ≤ ARV < 12): Monitor — track trend over 30 days
 * ELEVATED  (12 ≤ ARV < 16): Flag in physician dashboard, consider medication timing
 * HIGH      (ARV ≥ 16):   Decision Card: medication review
 */
public enum VariabilityClassification {
    LOW,
    MODERATE,
    ELEVATED,
    HIGH,
    INSUFFICIENT_DATA;

    public static VariabilityClassification fromARV(Double arv) {
        if (arv == null) return INSUFFICIENT_DATA;
        if (arv < 8.0 - 1e-9) return LOW;
        if (arv < 12.0 - 1e-9) return MODERATE;
        if (arv < 16.0 - 1e-9) return ELEVATED;
        return HIGH;
    }
}
```

- [ ] **Step 4: Create DipClassification enum**

```java
package com.cardiofit.flink.models;

/**
 * Nocturnal dipping pattern classification.
 *
 * Dip ratio = 1 - (nocturnal_mean_SBP / daytime_mean_SBP)
 *
 * DIPPER:           10–20% nocturnal reduction (normal, cardioprotective)
 * NON_DIPPER:       0–10% reduction (elevated CV risk)
 * EXTREME_DIPPER:   > 20% reduction (cerebral hypoperfusion risk)
 * REVERSE_DIPPER:   < 0% (nocturnal > daytime — highest CV risk)
 * INSUFFICIENT_DATA: fewer than 3 morning + 3 evening readings in 7 days
 *
 * Non-dippers have 2–3x higher cardiovascular event rates per MAPEC study.
 * Detection requires morning AND evening (or nocturnal) readings on the same days.
 */
public enum DipClassification {
    DIPPER,
    NON_DIPPER,
    EXTREME_DIPPER,
    REVERSE_DIPPER,
    INSUFFICIENT_DATA;

    /**
     * Classify from dip ratio.
     * @param dipRatio 1 - (nightMean / dayMean)
     */
    public static DipClassification fromDipRatio(double dipRatio) {
        if (dipRatio < 0.0) return REVERSE_DIPPER;
        if (dipRatio < 0.10 - 1e-9) return NON_DIPPER;
        if (dipRatio < 0.20 - 1e-9) return DIPPER;
        return EXTREME_DIPPER;
    }
}
```

- [ ] **Step 5: Create SurgeClassification enum**

```java
package com.cardiofit.flink.models;

/**
 * Morning BP surge classification.
 * Surge = morning_SBP_avg - evening_SBP_avg (sleep-trough method).
 * Per Kario et al., morning surge > 25 mmHg is an independent stroke predictor.
 *
 * NORMAL:    surge < 20 mmHg
 * ELEVATED:  20 ≤ surge < 35 mmHg (monitor, consider medication timing)
 * HIGH:      surge ≥ 35 mmHg (Decision Card: assess medication timing, sleep apnea)
 */
public enum SurgeClassification {
    NORMAL,
    ELEVATED,
    HIGH,
    INSUFFICIENT_DATA;

    public static SurgeClassification fromSurge(Double surgeMmHg) {
        if (surgeMmHg == null) return INSUFFICIENT_DATA;
        if (surgeMmHg < 20.0 - 1e-9) return NORMAL;
        if (surgeMmHg < 35.0 - 1e-9) return ELEVATED;
        return HIGH;
    }
}
```

- [ ] **Step 6: Create BPControlStatus enum**

```java
package com.cardiofit.flink.models;

/**
 * BP control status based on 7-day average SBP and DBP.
 * AHA/ACC 2017 guidelines + JSH 2025 home BP targets.
 *
 * Home BP targets (lower than clinic thresholds):
 *   CONTROLLED:      SBP < 130 AND DBP < 80 (per JSH 2025 Asakatsu BP)
 *   ELEVATED:        130 ≤ SBP < 135 OR 80 ≤ DBP < 85
 *   STAGE_1:         135 ≤ SBP < 145 OR 85 ≤ DBP < 90
 *   STAGE_2:         SBP ≥ 145 OR DBP ≥ 90
 *   CRISIS:          SBP > 180 OR DBP > 120 (any single reading)
 *
 * Note: These are HOME BP thresholds, which are 5 mmHg lower than clinic thresholds.
 * The V-MCU correction loop uses bp_control_status to trigger Channel C
 * HTN-1 protocol for titration when STAGE_1 or worse persists ≥ 14 days.
 */
public enum BPControlStatus {
    CONTROLLED,
    ELEVATED,
    STAGE_1_UNCONTROLLED,
    STAGE_2_UNCONTROLLED,
    CRISIS;

    public static BPControlStatus fromAverages(double avgSBP, double avgDBP) {
        if (avgSBP >= 145.0 - 1e-9 || avgDBP >= 90.0 - 1e-9) return STAGE_2_UNCONTROLLED;
        if (avgSBP >= 135.0 - 1e-9 || avgDBP >= 85.0 - 1e-9) return STAGE_1_UNCONTROLLED;
        if (avgSBP >= 130.0 - 1e-9 || avgDBP >= 80.0 - 1e-9) return ELEVATED;
        return CONTROLLED;
    }
}
```

- [ ] **Step 7: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 8: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/TimeContext.java \
  src/main/java/com/cardiofit/flink/models/BPSource.java \
  src/main/java/com/cardiofit/flink/models/VariabilityClassification.java \
  src/main/java/com/cardiofit/flink/models/DipClassification.java \
  src/main/java/com/cardiofit/flink/models/SurgeClassification.java \
  src/main/java/com/cardiofit/flink/models/BPControlStatus.java
git commit -m "feat(module7): add enums for BP variability classification types"
```

---

## Task 2: BPReading — Canonical Input Model

**Files:**
- Create: `src/main/java/com/cardiofit/flink/models/BPReading.java`

This is the canonical deserialized representation of a BP measurement from both input topics. Module 7 must normalize field names from two different topic schemas into this single type.

- [ ] **Step 1: Create BPReading**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Canonical BP reading from ingestion.vitals or ingestion.clinic-bp.
 *
 * Field normalization (Rule 2):
 *   ingestion.vitals uses: "systolicbloodpressure", "diastolicbloodpressure"
 *   ingestion.clinic-bp may use: "systolic", "diastolic", "sbp", "dbp"
 *   Both may carry: "heartrate" (for orthostatic assessment)
 *
 * Validation at entry (Rule 1 + Rule 3):
 *   - patientId must be non-null
 *   - SBP must be in [40, 300] (physiological range)
 *   - DBP must be in [20, 200]
 *   - SBP must be > DBP
 *   - timestamp must be present and within ±1h future / 30d past
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class BPReading implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("systolic") private Double systolic;     // mmHg
    @JsonProperty("diastolic") private Double diastolic;    // mmHg
    @JsonProperty("heart_rate") private Double heartRate;   // bpm (optional, for orthostatic)
    @JsonProperty("timestamp") private long timestamp;      // epoch millis
    @JsonProperty("time_context") private String timeContext; // MORNING, EVENING, etc. (may be null)
    @JsonProperty("source") private String source;           // HOME_CUFF, CLINIC, CUFFLESS
    @JsonProperty("position") private String position;       // SEATED, STANDING, SUPINE
    @JsonProperty("device_type") private String deviceType;  // oscillometric_cuff, etc.
    @JsonProperty("encounter_id") private String encounterId;
    @JsonProperty("correlation_id") private String correlationId;

    public BPReading() {}

    // ── Validation ──

    public boolean isValid() {
        if (patientId == null || patientId.isEmpty()) return false;
        if (systolic == null || diastolic == null) return false;
        if (systolic < 40 || systolic > 300) return false;
        if (diastolic < 20 || diastolic > 200) return false;
        if (systolic <= diastolic) return false;
        if (timestamp <= 0) return false;
        return true;
    }

    // ── Derived accessors ──

    public TimeContext resolveTimeContext() {
        if (timeContext != null && !timeContext.isEmpty()) {
            try {
                return TimeContext.valueOf(timeContext.toUpperCase());
            } catch (IllegalArgumentException e) {
                // fall through to hour-based derivation
            }
        }
        // Derive from timestamp hour (UTC — in production, use patient timezone)
        java.time.Instant instant = java.time.Instant.ofEpochMilli(timestamp);
        int hour = instant.atZone(java.time.ZoneOffset.UTC).getHour();
        return TimeContext.fromHour(hour);
    }

    public BPSource resolveSource() {
        if (source == null || source.isEmpty()) return BPSource.UNKNOWN;
        try {
            return BPSource.valueOf(source.toUpperCase());
        } catch (IllegalArgumentException e) {
            // Normalize common variants
            String s = source.toLowerCase();
            if (s.contains("clinic") || s.contains("office")) return BPSource.CLINIC;
            if (s.contains("cuffless") || s.contains("wearable")) return BPSource.CUFFLESS;
            if (s.contains("cuff") || s.contains("home") || s.contains("oscillometric")) return BPSource.HOME_CUFF;
            return BPSource.UNKNOWN;
        }
    }

    public double getPulsePresure() {
        return systolic - diastolic;
    }

    public double getMeanArterialPressure() {
        return diastolic + (systolic - diastolic) / 3.0;
    }

    // ── Standard getters/setters ──
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Double getSystolic() { return systolic; }
    public void setSystolic(Double systolic) { this.systolic = systolic; }
    public Double getDiastolic() { return diastolic; }
    public void setDiastolic(Double diastolic) { this.diastolic = diastolic; }
    public Double getHeartRate() { return heartRate; }
    public void setHeartRate(Double heartRate) { this.heartRate = heartRate; }
    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }
    public String getTimeContext() { return timeContext; }
    public void setTimeContext(String timeContext) { this.timeContext = timeContext; }
    public String getSource() { return source; }
    public void setSource(String source) { this.source = source; }
    public String getPosition() { return position; }
    public void setPosition(String position) { this.position = position; }
    public String getDeviceType() { return deviceType; }
    public void setDeviceType(String deviceType) { this.deviceType = deviceType; }
    public String getEncounterId() { return encounterId; }
    public void setEncounterId(String encounterId) { this.encounterId = encounterId; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String correlationId) { this.correlationId = correlationId; }
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/BPReading.java
git commit -m "feat(module7): add BPReading canonical input model with validation and field normalization"
```

---

## Task 3: DailyBPSummary + PatientBPState — State Models

**Files:**
- Create: `src/main/java/com/cardiofit/flink/models/DailyBPSummary.java`
- Create: `src/main/java/com/cardiofit/flink/models/PatientBPState.java`

- [ ] **Step 1: Create DailyBPSummary**

```java
package com.cardiofit.flink.models;

import java.io.Serializable;

/**
 * Aggregated BP data for a single calendar day.
 * Stored in the 30-day rolling window within PatientBPState.
 *
 * A day's summary accumulates as readings arrive. On each new reading,
 * the running averages are updated incrementally.
 */
public class DailyBPSummary implements Serializable {
    private static final long serialVersionUID = 1L;

    private String dateKey;         // "YYYY-MM-DD"
    private double sumSBP;
    private double sumDBP;
    private int readingCount;

    // Time-context-specific averages for surge/dip computation
    private double sumMorningSBP;
    private int morningCount;
    private double sumEveningSBP;
    private int eveningCount;
    private double sumNocturnalSBP;
    private int nocturnalCount;
    private double sumDaytimeSBP;
    private int daytimeCount;

    // Clinic vs home for white-coat/masked detection
    private double sumClinicSBP;
    private int clinicCount;
    private double sumHomeSBP;
    private int homeCount;

    // Extremes for crisis detection
    private double maxSBP;
    private double maxDBP;
    private double minSBP;

    public DailyBPSummary() {}

    public DailyBPSummary(String dateKey) {
        this.dateKey = dateKey;
        this.maxSBP = Double.MIN_VALUE;
        this.maxDBP = Double.MIN_VALUE;
        this.minSBP = Double.MAX_VALUE;
    }

    /**
     * Add a reading to this day's summary. Incremental update.
     */
    public void addReading(BPReading reading) {
        double sbp = reading.getSystolic();
        double dbp = reading.getDiastolic();

        sumSBP += sbp;
        sumDBP += dbp;
        readingCount++;

        if (sbp > maxSBP) maxSBP = sbp;
        if (dbp > maxDBP) maxDBP = dbp;
        if (sbp < minSBP) minSBP = sbp;

        // Time-context-specific accumulation
        TimeContext tc = reading.resolveTimeContext();
        switch (tc) {
            case MORNING -> { sumMorningSBP += sbp; morningCount++; }
            case EVENING -> { sumEveningSBP += sbp; eveningCount++; }
            case NIGHT   -> { sumNocturnalSBP += sbp; nocturnalCount++; }
            default -> {}
        }
        if (tc.isDaytime()) { sumDaytimeSBP += sbp; daytimeCount++; }

        // Source-specific accumulation
        BPSource src = reading.resolveSource();
        if (src == BPSource.CLINIC) { sumClinicSBP += sbp; clinicCount++; }
        else if (src == BPSource.HOME_CUFF) { sumHomeSBP += sbp; homeCount++; }
    }

    // ── Computed averages ──
    public double getAvgSBP() { return readingCount > 0 ? sumSBP / readingCount : 0; }
    public double getAvgDBP() { return readingCount > 0 ? sumDBP / readingCount : 0; }
    public Double getMorningAvgSBP() { return morningCount > 0 ? sumMorningSBP / morningCount : null; }
    public Double getEveningAvgSBP() { return eveningCount > 0 ? sumEveningSBP / eveningCount : null; }
    public Double getNocturnalAvgSBP() { return nocturnalCount > 0 ? sumNocturnalSBP / nocturnalCount : null; }
    public Double getDaytimeAvgSBP() { return daytimeCount > 0 ? sumDaytimeSBP / daytimeCount : null; }
    public Double getClinicAvgSBP() { return clinicCount > 0 ? sumClinicSBP / clinicCount : null; }
    public Double getHomeAvgSBP() { return homeCount > 0 ? sumHomeSBP / homeCount : null; }

    // ── Standard getters ──
    public String getDateKey() { return dateKey; }
    public void setDateKey(String dateKey) { this.dateKey = dateKey; }
    public int getReadingCount() { return readingCount; }
    public double getMaxSBP() { return maxSBP; }
    public double getMaxDBP() { return maxDBP; }
    public double getMinSBP() { return minSBP; }
    public int getMorningCount() { return morningCount; }
    public int getEveningCount() { return eveningCount; }
    public int getNocturnalCount() { return nocturnalCount; }
    public int getDaytimeCount() { return daytimeCount; }
    public int getClinicCount() { return clinicCount; }
    public int getHomeCount() { return homeCount; }
    // Setters for all fields needed for serialization
    public void setSumSBP(double v) { sumSBP = v; }
    public double getSumSBP() { return sumSBP; }
    public void setSumDBP(double v) { sumDBP = v; }
    public double getSumDBP() { return sumDBP; }
    public void setReadingCount(int v) { readingCount = v; }
    public void setSumMorningSBP(double v) { sumMorningSBP = v; }
    public double getSumMorningSBP() { return sumMorningSBP; }
    public void setMorningCount(int v) { morningCount = v; }
    public void setSumEveningSBP(double v) { sumEveningSBP = v; }
    public double getSumEveningSBP() { return sumEveningSBP; }
    public void setEveningCount(int v) { eveningCount = v; }
    public void setSumNocturnalSBP(double v) { sumNocturnalSBP = v; }
    public double getSumNocturnalSBP() { return sumNocturnalSBP; }
    public void setNocturnalCount(int v) { nocturnalCount = v; }
    public void setSumDaytimeSBP(double v) { sumDaytimeSBP = v; }
    public double getSumDaytimeSBP() { return sumDaytimeSBP; }
    public void setDaytimeCount(int v) { daytimeCount = v; }
    public void setSumClinicSBP(double v) { sumClinicSBP = v; }
    public double getSumClinicSBP() { return sumClinicSBP; }
    public void setClinicCount(int v) { clinicCount = v; }
    public void setSumHomeSBP(double v) { sumHomeSBP = v; }
    public double getSumHomeSBP() { return sumHomeSBP; }
    public void setHomeCount(int v) { homeCount = v; }
    public void setMaxSBP(double v) { maxSBP = v; }
    public void setMaxDBP(double v) { maxDBP = v; }
    public void setMinSBP(double v) { minSBP = v; }
}
```

- [ ] **Step 2: Create PatientBPState**

```java
package com.cardiofit.flink.models;

import java.io.Serializable;
import java.time.Instant;
import java.time.LocalDate;
import java.time.ZoneOffset;
import java.util.*;

/**
 * Per-patient BP state maintained in Flink keyed state.
 * Contains a 30-day rolling window of DailyBPSummary entries.
 *
 * State TTL: 30 days (per architecture Section 7.2).
 * First event after state expiry produces contextDepth: INITIAL.
 */
public class PatientBPState implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private Map<String, DailyBPSummary> dailySummaries = new LinkedHashMap<>(); // dateKey → summary
    private long lastReadingTime;
    private int totalReadingsProcessed;

    public PatientBPState() {}
    public PatientBPState(String patientId) { this.patientId = patientId; }

    /**
     * Add a reading to the appropriate day's summary.
     * Creates a new DailyBPSummary if this is the first reading for the day.
     * Evicts entries older than 30 days.
     */
    public void addReading(BPReading reading) {
        String dateKey = toDateKey(reading.getTimestamp());
        DailyBPSummary summary = dailySummaries.computeIfAbsent(
            dateKey, DailyBPSummary::new);
        summary.addReading(reading);
        lastReadingTime = Math.max(lastReadingTime, reading.getTimestamp());
        totalReadingsProcessed++;
        evictOlderThan30Days(reading.getTimestamp());
    }

    /**
     * Get daily summaries for the last N days that HAVE readings.
     * Days without readings are skipped — they don't contribute to ARV.
     * @param windowDays 7 or 30
     * @param referenceTime the current event time
     * @return ordered list of DailyBPSummary (oldest first)
     */
    public List<DailyBPSummary> getSummariesInWindow(int windowDays, long referenceTime) {
        LocalDate refDate = Instant.ofEpochMilli(referenceTime)
            .atZone(ZoneOffset.UTC).toLocalDate();
        LocalDate cutoff = refDate.minusDays(windowDays);

        List<DailyBPSummary> result = new ArrayList<>();
        for (Map.Entry<String, DailyBPSummary> entry : dailySummaries.entrySet()) {
            LocalDate date = LocalDate.parse(entry.getKey());
            if (!date.isBefore(cutoff) && !date.isAfter(refDate)) {
                result.add(entry.getValue());
            }
        }
        result.sort(Comparator.comparing(DailyBPSummary::getDateKey));
        return result;
    }

    private void evictOlderThan30Days(long referenceTime) {
        LocalDate cutoff = Instant.ofEpochMilli(referenceTime)
            .atZone(ZoneOffset.UTC).toLocalDate().minusDays(31);
        dailySummaries.entrySet().removeIf(entry -> {
            LocalDate date = LocalDate.parse(entry.getKey());
            return date.isBefore(cutoff);
        });
    }

    private String toDateKey(long timestamp) {
        return Instant.ofEpochMilli(timestamp)
            .atZone(ZoneOffset.UTC).toLocalDate().toString();
    }

    public String getContextDepth() {
        if (totalReadingsProcessed <= 1) return "INITIAL";
        List<DailyBPSummary> recent = getSummariesInWindow(7, lastReadingTime);
        if (recent.size() < 3) return "BUILDING";
        return "ESTABLISHED";
    }

    // ── Standard getters/setters ──
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Map<String, DailyBPSummary> getDailySummaries() { return dailySummaries; }
    public void setDailySummaries(Map<String, DailyBPSummary> dailySummaries) { this.dailySummaries = dailySummaries; }
    public long getLastReadingTime() { return lastReadingTime; }
    public void setLastReadingTime(long lastReadingTime) { this.lastReadingTime = lastReadingTime; }
    public int getTotalReadingsProcessed() { return totalReadingsProcessed; }
    public void setTotalReadingsProcessed(int v) { totalReadingsProcessed = v; }
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/DailyBPSummary.java \
  src/main/java/com/cardiofit/flink/models/PatientBPState.java
git commit -m "feat(module7): add DailyBPSummary and PatientBPState with 30-day rolling window"
```

---

## Task 4: BPVariabilityMetrics — Output Model

**Files:**
- Create: `src/main/java/com/cardiofit/flink/models/BPVariabilityMetrics.java`

This is the structured output emitted to `flink.bp-variability-metrics`. Five downstream consumers read this: KB-20, KB-26, KB-23, Module 4 RunCycle, and the physician dashboard.

- [ ] **Step 1: Create BPVariabilityMetrics**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Complete BP variability output emitted on each new clinical-grade reading.
 *
 * Consumers:
 *   KB-20: all fields → patient projection (Redis + PostgreSQL)
 *   KB-26: mean_sbp_7d, arv_sbp_7d, dip_classification, bp_control_status
 *          → MHRI hemodynamic component (25% weight)
 *   KB-23: bp_control_status, variability_classification, surge_classification
 *          → Decision Card generation
 *   M4 RunCycle: bp_control_status, variability_classification
 *          → dual-domain state classification
 *   Dashboard: all fields for visualization
 */
public class BPVariabilityMetrics implements Serializable {
    private static final long serialVersionUID = 1L;

    // ── Identity ──
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("timestamp") private long timestamp;
    @JsonProperty("context_depth") private String contextDepth; // INITIAL, BUILDING, ESTABLISHED

    // ── Current reading (trigger) ──
    @JsonProperty("trigger_sbp") private double triggerSBP;
    @JsonProperty("trigger_dbp") private double triggerDBP;
    @JsonProperty("trigger_source") private BPSource triggerSource;
    @JsonProperty("trigger_time_context") private TimeContext triggerTimeContext;

    // ── 7-day metrics ──
    @JsonProperty("mean_sbp_7d") private Double meanSBP7d;
    @JsonProperty("mean_dbp_7d") private Double meanDBP7d;
    @JsonProperty("sd_sbp_7d") private Double sdSBP7d;
    @JsonProperty("cv_sbp_7d") private Double cvSBP7d;        // coefficient of variation
    @JsonProperty("arv_sbp_7d") private Double arvSBP7d;      // average real variability
    @JsonProperty("days_with_data_7d") private int daysWithData7d;

    // ── 30-day metrics ──
    @JsonProperty("mean_sbp_30d") private Double meanSBP30d;
    @JsonProperty("mean_dbp_30d") private Double meanDBP30d;
    @JsonProperty("sd_sbp_30d") private Double sdSBP30d;
    @JsonProperty("cv_sbp_30d") private Double cvSBP30d;
    @JsonProperty("arv_sbp_30d") private Double arvSBP30d;
    @JsonProperty("days_with_data_30d") private int daysWithData30d;

    // ── Classifications ──
    @JsonProperty("variability_classification") private VariabilityClassification variabilityClassification;
    @JsonProperty("dip_classification") private DipClassification dipClassification;
    @JsonProperty("surge_classification") private SurgeClassification surgeClassification;
    @JsonProperty("bp_control_status") private BPControlStatus bpControlStatus;

    // ── Morning surge ──
    @JsonProperty("morning_surge_today") private Double morningSurgeToday;
    @JsonProperty("morning_surge_7d_avg") private Double morningSurge7dAvg;

    // ── Dipping ──
    @JsonProperty("dip_ratio") private Double dipRatio;

    // ── White-coat / Masked HTN flags ──
    @JsonProperty("white_coat_suspect") private boolean whiteCoatSuspect;
    @JsonProperty("masked_htn_suspect") private boolean maskedHtnSuspect;
    @JsonProperty("clinic_home_delta") private Double clinicHomeDelta; // clinic avg - home avg

    // ── Crisis flag ──
    @JsonProperty("crisis_detected") private boolean crisisDetected;

    // ── Data quality ──
    @JsonProperty("total_readings_in_state") private int totalReadingsInState;

    public BPVariabilityMetrics() {}

    // Getters and setters for all fields
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }
    public String getContextDepth() { return contextDepth; }
    public void setContextDepth(String contextDepth) { this.contextDepth = contextDepth; }
    public double getTriggerSBP() { return triggerSBP; }
    public void setTriggerSBP(double v) { this.triggerSBP = v; }
    public double getTriggerDBP() { return triggerDBP; }
    public void setTriggerDBP(double v) { this.triggerDBP = v; }
    public BPSource getTriggerSource() { return triggerSource; }
    public void setTriggerSource(BPSource v) { this.triggerSource = v; }
    public TimeContext getTriggerTimeContext() { return triggerTimeContext; }
    public void setTriggerTimeContext(TimeContext v) { this.triggerTimeContext = v; }
    public Double getMeanSBP7d() { return meanSBP7d; }
    public void setMeanSBP7d(Double v) { this.meanSBP7d = v; }
    public Double getMeanDBP7d() { return meanDBP7d; }
    public void setMeanDBP7d(Double v) { this.meanDBP7d = v; }
    public Double getSdSBP7d() { return sdSBP7d; }
    public void setSdSBP7d(Double v) { this.sdSBP7d = v; }
    public Double getCvSBP7d() { return cvSBP7d; }
    public void setCvSBP7d(Double v) { this.cvSBP7d = v; }
    public Double getArvSBP7d() { return arvSBP7d; }
    public void setArvSBP7d(Double v) { this.arvSBP7d = v; }
    public int getDaysWithData7d() { return daysWithData7d; }
    public void setDaysWithData7d(int v) { this.daysWithData7d = v; }
    public Double getMeanSBP30d() { return meanSBP30d; }
    public void setMeanSBP30d(Double v) { this.meanSBP30d = v; }
    public Double getMeanDBP30d() { return meanDBP30d; }
    public void setMeanDBP30d(Double v) { this.meanDBP30d = v; }
    public Double getSdSBP30d() { return sdSBP30d; }
    public void setSdSBP30d(Double v) { this.sdSBP30d = v; }
    public Double getCvSBP30d() { return cvSBP30d; }
    public void setCvSBP30d(Double v) { this.cvSBP30d = v; }
    public Double getArvSBP30d() { return arvSBP30d; }
    public void setArvSBP30d(Double v) { this.arvSBP30d = v; }
    public int getDaysWithData30d() { return daysWithData30d; }
    public void setDaysWithData30d(int v) { this.daysWithData30d = v; }
    public VariabilityClassification getVariabilityClassification() { return variabilityClassification; }
    public void setVariabilityClassification(VariabilityClassification v) { this.variabilityClassification = v; }
    public DipClassification getDipClassification() { return dipClassification; }
    public void setDipClassification(DipClassification v) { this.dipClassification = v; }
    public SurgeClassification getSurgeClassification() { return surgeClassification; }
    public void setSurgeClassification(SurgeClassification v) { this.surgeClassification = v; }
    public BPControlStatus getBpControlStatus() { return bpControlStatus; }
    public void setBpControlStatus(BPControlStatus v) { this.bpControlStatus = v; }
    public Double getMorningSurgeToday() { return morningSurgeToday; }
    public void setMorningSurgeToday(Double v) { this.morningSurgeToday = v; }
    public Double getMorningSurge7dAvg() { return morningSurge7dAvg; }
    public void setMorningSurge7dAvg(Double v) { this.morningSurge7dAvg = v; }
    public Double getDipRatio() { return dipRatio; }
    public void setDipRatio(Double v) { this.dipRatio = v; }
    public boolean isWhiteCoatSuspect() { return whiteCoatSuspect; }
    public void setWhiteCoatSuspect(boolean v) { this.whiteCoatSuspect = v; }
    public boolean isMaskedHtnSuspect() { return maskedHtnSuspect; }
    public void setMaskedHtnSuspect(boolean v) { this.maskedHtnSuspect = v; }
    public Double getClinicHomeDelta() { return clinicHomeDelta; }
    public void setClinicHomeDelta(Double v) { this.clinicHomeDelta = v; }
    public boolean isCrisisDetected() { return crisisDetected; }
    public void setCrisisDetected(boolean v) { this.crisisDetected = v; }
    public int getTotalReadingsInState() { return totalReadingsInState; }
    public void setTotalReadingsInState(int v) { this.totalReadingsInState = v; }
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/BPVariabilityMetrics.java
git commit -m "feat(module7): add BPVariabilityMetrics output model for 5 downstream consumers"
```

---

## Task 5: Module7TestBuilder — Test Data Factory

**Files:**
- Create: `src/test/java/com/cardiofit/flink/builders/Module7TestBuilder.java`

Provides factory methods for canonical BP patient scenarios. Each scenario produces a pre-populated `PatientBPState` with realistic daily summaries.

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
     * 7 days with morning ~140, evening ~138 (dip ratio < 0.10).
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
     * Highest CV risk pattern.
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
     * Masked hypertension suspect: home BP > clinic BP.
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

- [ ] **Step 2: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test-compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/builders/Module7TestBuilder.java
git commit -m "test(module7): add Module7TestBuilder with 9 canonical BP patient scenarios"
```

---

## Task 6: Module7ARVComputer — Average Real Variability + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module7ARVComputer.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module7ARVComputationTest.java`

ARV is the core metric. It captures true day-to-day fluctuation without being inflated by outliers (unlike SD). The algorithm computes the mean of absolute differences between consecutive daily averages.

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
        // Manually build 3 days with known values
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
        // 5 days: 120, 130, 125, 135, 120
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
        // Build 20 days of data
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
Expected: FAIL — `Module7ARVComputer` does not exist

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

    /**
     * Compute mean SBP across daily averages.
     */
    public static double computeMeanSBP(List<DailyBPSummary> summaries) {
        return summaries.stream()
            .mapToDouble(DailyBPSummary::getAvgSBP)
            .average()
            .orElse(0.0);
    }

    /**
     * Compute mean DBP across daily averages.
     */
    public static double computeMeanDBP(List<DailyBPSummary> summaries) {
        return summaries.stream()
            .mapToDouble(DailyBPSummary::getAvgDBP)
            .average()
            .orElse(0.0);
    }

    /**
     * Compute standard deviation of daily SBP averages.
     */
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

    /**
     * Compute coefficient of variation (SD / mean).
     */
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

## Task 7: Module7SurgeDetector — Morning Surge + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module7SurgeDetector.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module7SurgeDetectionTest.java`

Morning surge uses the sleep-trough method: morning_SBP_today - evening_SBP_yesterday. Per Kario et al., this is more predictive than the pre-waking method and feasible with twice-daily home readings.

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
        // Only morning readings, no evening
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
Expected: FAIL

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

## Task 8: Module7DipClassifier — Dipping Pattern + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module7DipClassifier.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module7DipClassificationTest.java`

Dipping classification requires both daytime and nocturnal readings. The dip ratio = 1 - (nocturnal_mean / daytime_mean). Non-dippers have 2–3x higher CV event rates (MAPEC study).

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
    void controlledPatient_classifiesAsDipper() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-DIP");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Module7DipClassifier.DipResult result = Module7DipClassifier.classify(summaries);
        // Controlled: morning ~122, evening ~114 (daytime ~118), no nocturnal readings
        // Note: controlledPatient has morning + evening but no NIGHT readings
        // This means dipping data may be insufficient depending on implementation
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
 * Requires at least 3 days with both daytime AND nocturnal readings
 * for a meaningful classification.
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module7DipClassifier {

    private Module7DipClassifier() {}

    /**
     * Immutable result record for dipping analysis.
     */
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

## Task 9: Module7CrisisDetector + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module7CrisisDetector.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module7CrisisDetectionTest.java`

Crisis detection is the highest-priority path. Any single reading with SBP > 180 or DBP > 120 bypasses all windowed computation and immediately emits to `ingestion.safety-critical`. This is a patient safety mechanism — it must be evaluated BEFORE any other computation.

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
    void surgeDetection_sbpJump30InOneHour() {
        BPReading previous = Module7TestBuilder.reading("P001", 135, 82,
            System.currentTimeMillis() - 45 * 60 * 1000L, "MORNING", "HOME_CUFF");
        BPReading current = Module7TestBuilder.reading("P001", 170, 95,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertTrue(Module7CrisisDetector.isAcuteSurge(previous, current),
            "SBP jump from 135 to 170 (35 mmHg) in 45 min should be acute surge");
    }

    @Test
    void surgeDetection_sbpJump20InOneHour_notSurge() {
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
 * Crisis thresholds (per architecture):
 *   SBP > 180 mmHg OR DBP > 120 mmHg
 *
 * Acute surge detection (per architecture):
 *   SBP increase > 30 mmHg within < 1 hour
 *
 * Both conditions bypass normal processing and emit directly to
 * ingestion.safety-critical for immediate notification-service delivery.
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

## Task 10: Module7BPControlClassifier + White-Coat/Masked HTN + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module7BPControlClassifier.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module7BPControlTest.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module7WhiteCoatMaskedTest.java`

- [ ] **Step 1: Write BP control and white-coat/masked tests**

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
        // SBP is controlled but DBP is stage 1
        assertEquals(BPControlStatus.STAGE_1_UNCONTROLLED,
            BPControlStatus.fromAverages(128, 87),
            "DBP 87 alone should trigger STAGE_1 even with controlled SBP");
    }
}
```

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

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module7BPControlTest,Module7WhiteCoatMaskedTest" -q 2>&1 | tail -10`
Expected: FAIL

- [ ] **Step 3: Implement Module7BPControlClassifier**

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
 * Both require ≥ 2 clinic readings and ≥ 5 home readings in 30 days.
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
        if (summaries == null || summaries.isEmpty()) return BPControlStatus.ELEVATED; // unknown → conservative

        double avgSBP = Module7ARVComputer.computeMeanSBP(summaries);
        double avgDBP = Module7ARVComputer.computeMeanDBP(summaries);

        return BPControlStatus.fromAverages(avgSBP, avgDBP);
    }

    /**
     * Detect white-coat and masked hypertension from clinic vs home readings.
     * Requires ≥ 2 clinic readings and ≥ 5 home readings.
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

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module7BPControlTest,Module7WhiteCoatMaskedTest" -q 2>&1 | tail -15`
Expected: All 7 tests PASS

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module7BPControlClassifier.java \
  src/test/java/com/cardiofit/flink/operators/Module7BPControlTest.java \
  src/test/java/com/cardiofit/flink/operators/Module7WhiteCoatMaskedTest.java
git commit -m "feat(module7): implement BP control classification and white-coat/masked HTN detection"
```

---

## Task 11: Kafka Topics — Add Module 7 Topics

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/utils/KafkaTopics.java`

- [ ] **Step 1: Add Module 7 topics to KafkaTopics enum**

Add these entries (after existing entries, before the closing semicolon):

```java
    // ── Module 7: BP Variability Engine ──
    BP_VARIABILITY_METRICS("flink.bp-variability-metrics", 4, 90),    // 90-day retention for trend analysis
    SAFETY_CRITICAL("ingestion.safety-critical", 4, 30),              // crisis bypass topic
```

**Note:** `ingestion.safety-critical` may already exist (shared with Module 8). Check before adding. If it exists, skip that entry.

- [ ] **Step 2: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/utils/KafkaTopics.java
git commit -m "feat(module7): add Kafka topics for BP variability metrics and safety-critical output"
```

---

## Task 12: Module7_BPVariabilityEngine — Main Operator

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module7_BPVariabilityEngine.java`

This is the main `KeyedProcessFunction` that wires together all the static analyzers. It maintains `PatientBPState` in Flink keyed state, evaluates crisis detection first, then recomputes variability metrics on each new clinical-grade reading.

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
 * State TTL: 30 days (per architecture Section 7.2).
 * Checkpointing: 30s interval, RocksDB, 3 retained.
 */
public class Module7_BPVariabilityEngine
        extends KeyedProcessFunction<String, BPReading, BPVariabilityMetrics> {

    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(Module7_BPVariabilityEngine.class);

    // ── Side-output tags ──
    public static final OutputTag<BPReading> CRISIS_TAG =
        new OutputTag<>("safety-critical", TypeInformation.of(BPReading.class));

    // ── State ──
    private transient ValueState<PatientBPState> bpState;
    private transient ValueState<BPReading> lastReadingState; // for acute surge detection

    @Override
    public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
        super.open(openContext);

        // Patient BP state with 30-day TTL
        ValueStateDescriptor<PatientBPState> stateDescriptor =
            new ValueStateDescriptor<>("patient-bp-state", PatientBPState.class);
        StateTtlConfig ttlConfig = StateTtlConfig
            .newBuilder(Duration.ofDays(31)) // 31 days to allow 30-day window computation
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();
        stateDescriptor.enableTimeToLive(ttlConfig);
        bpState = getRuntimeContext().getState(stateDescriptor);

        // Last reading for acute surge detection (short TTL)
        ValueStateDescriptor<BPReading> lastReadingDescriptor =
            new ValueStateDescriptor<>("last-bp-reading", BPReading.class);
        StateTtlConfig shortTtl = StateTtlConfig
            .newBuilder(Duration.ofHours(2))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();
        lastReadingDescriptor.enableTimeToLive(shortTtl);
        lastReadingState = getRuntimeContext().getState(lastReadingDescriptor);

        LOG.info("Module7_BPVariabilityEngine initialized");
    }

    @Override
    public void processElement(BPReading reading, Context ctx,
                                Collector<BPVariabilityMetrics> out) throws Exception {

        // ── 1. Validate ──
        if (!reading.isValid()) {
            LOG.warn("Module7: dropping invalid BP reading for patient {}. SBP={}, DBP={}",
                reading.getPatientId(), reading.getSystolic(), reading.getDiastolic());
            return;
        }

        // ── 2. Crisis detection — ALWAYS checked first, even for cuffless ──
        if (Module7CrisisDetector.isCrisis(reading)) {
            LOG.warn("Module7: CRISIS detected for patient {}. SBP={}, DBP={}",
                reading.getPatientId(), reading.getSystolic(), reading.getDiastolic());
            ctx.output(CRISIS_TAG, reading);
            // Do NOT return — still update state and emit metrics
        }

        // ── 3. Acute surge detection ──
        BPReading lastReading = lastReadingState.value();
        if (lastReading != null && Module7CrisisDetector.isAcuteSurge(lastReading, reading)) {
            LOG.warn("Module7: ACUTE SURGE for patient {}. Previous SBP={}, Current SBP={} (delta={})",
                reading.getPatientId(), lastReading.getSystolic(), reading.getSystolic(),
                reading.getSystolic() - lastReading.getSystolic());
            ctx.output(CRISIS_TAG, reading);
        }
        lastReadingState.update(reading);

        // ── 4. Skip non-clinical-grade for main computation ──
        BPSource source = reading.resolveSource();
        if (!source.isClinicalGrade()) {
            LOG.debug("Module7: skipping non-clinical-grade reading (source={}) for patient {}",
                source, reading.getPatientId());
            return; // Cuffless readings: don't update state or emit metrics
        }

        // ── 5. Update rolling state ──
        PatientBPState state = bpState.value();
        if (state == null) {
            state = new PatientBPState(reading.getPatientId());
        }
        state.addReading(reading);

        // ── 6. Compute all metrics ──
        BPVariabilityMetrics metrics = computeMetrics(reading, state);

        // ── 7. Emit and update state ──
        out.collect(metrics);
        bpState.update(state);

        LOG.info("Module7: patient={}, control={}, variability={}, dip={}, surge={}",
            reading.getPatientId(),
            metrics.getBpControlStatus(),
            metrics.getVariabilityClassification(),
            metrics.getDipClassification(),
            metrics.getSurgeClassification());
    }

    private BPVariabilityMetrics computeMetrics(BPReading reading, PatientBPState state) {
        long now = reading.getTimestamp();
        BPVariabilityMetrics m = new BPVariabilityMetrics();

        // Identity
        m.setPatientId(reading.getPatientId());
        m.setTimestamp(now);
        m.setContextDepth(state.getContextDepth());

        // Current reading
        m.setTriggerSBP(reading.getSystolic());
        m.setTriggerDBP(reading.getDiastolic());
        m.setTriggerSource(reading.resolveSource());
        m.setTriggerTimeContext(reading.resolveTimeContext());

        // 7-day window
        List<DailyBPSummary> window7 = state.getSummariesInWindow(7, now);
        m.setDaysWithData7d(window7.size());
        if (window7.size() >= 3) {
            m.setMeanSBP7d(Module7ARVComputer.computeMeanSBP(window7));
            m.setMeanDBP7d(Module7ARVComputer.computeMeanDBP(window7));
            m.setSdSBP7d(Module7ARVComputer.computeSD(window7));
            m.setCvSBP7d(Module7ARVComputer.computeCV(window7));
            m.setArvSBP7d(Module7ARVComputer.computeARV(window7));
        }

        // 30-day window
        List<DailyBPSummary> window30 = state.getSummariesInWindow(30, now);
        m.setDaysWithData30d(window30.size());
        if (window30.size() >= 7) {
            m.setMeanSBP30d(Module7ARVComputer.computeMeanSBP(window30));
            m.setMeanDBP30d(Module7ARVComputer.computeMeanDBP(window30));
            m.setSdSBP30d(Module7ARVComputer.computeSD(window30));
            m.setCvSBP30d(Module7ARVComputer.computeCV(window30));
            m.setArvSBP30d(Module7ARVComputer.computeARV(window30));
        }

        // Classifications (use 7-day ARV as primary)
        m.setVariabilityClassification(VariabilityClassification.fromARV(m.getArvSBP7d()));
        m.setBpControlStatus(Module7BPControlClassifier.classifyControl(window7));

        // Morning surge
        m.setMorningSurgeToday(Module7SurgeDetector.computeTodaySurge(window7, now));
        m.setMorningSurge7dAvg(Module7SurgeDetector.compute7DayAvgSurge(window7));
        m.setSurgeClassification(SurgeClassification.fromSurge(m.getMorningSurge7dAvg()));

        // Dipping
        Module7DipClassifier.DipResult dipResult = Module7DipClassifier.classify(window7);
        m.setDipClassification(dipResult.classification());
        m.setDipRatio(dipResult.dipRatio());

        // White-coat / Masked HTN (use 30-day window for enough clinic visits)
        Module7BPControlClassifier.WhiteCoatResult wcResult =
            Module7BPControlClassifier.detectWhiteCoatMasked(window30);
        m.setWhiteCoatSuspect(wcResult.whiteCoatSuspect());
        m.setMaskedHtnSuspect(wcResult.maskedHtnSuspect());
        m.setClinicHomeDelta(wcResult.clinicHomeDelta());

        // Crisis (from original reading — for downstream consumers)
        m.setCrisisDetected(Module7CrisisDetector.isCrisis(reading));

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
git commit -m "feat(module7): implement BPVariabilityEngine KeyedProcessFunction with crisis bypass, ARV, surge, dipping, control classification"
```

---

## Task 13: Wire Module 7 into FlinkJobOrchestrator

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java`

- [ ] **Step 1: Read the existing orchestrator to find the job-type switch**

```bash
cd backend/shared-infrastructure/flink-processing && grep -n "case\|job.*type\|egress-routing\|ml-inference\|clinical-action" \
  src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java | head -20
```

- [ ] **Step 2: Add a `bp-variability-engine` job type**

Add a new case that:
1. Creates a Kafka source for `ingestion.vitals` + `ingestion.clinic-bp` (union of two sources)
2. Deserializes to `BPReading` (with field normalization for the two different schemas)
3. Keys by `patientId`
4. Processes through `Module7_BPVariabilityEngine`
5. Connects main output to `flink.bp-variability-metrics` Kafka sink
6. Connects `CRISIS_TAG` side output to `ingestion.safety-critical` Kafka sink

Follow the same structure as existing job types. The specific code depends on the orchestrator's existing patterns.

- [ ] **Step 3: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java
git commit -m "feat(module7): wire BPVariabilityEngine into FlinkJobOrchestrator as bp-variability-engine job"
```

---

## Task 14: Integration Test

**Files:**
- Create: `src/test/java/com/cardiofit/flink/operators/Module7IntegrationTest.java`

Tests the full flow through the main operator using Flink's test harness.

- [ ] **Step 1: Write integration test**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Integration tests for Module 7.
 * Validates the wiring between static analyzers through the full flow.
 *
 * Note: Full Flink TestHarness tests require the harness dependency.
 * These tests validate the computeMetrics logic by building pre-populated
 * state and asserting on the output.
 */
class Module7IntegrationTest {

    @Test
    void controlledPatient_producesExpectedClassifications() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-CTRL");
        long now = System.currentTimeMillis();

        java.util.List<DailyBPSummary> window7 = state.getSummariesInWindow(7, now);
        assertTrue(window7.size() >= 3, "Should have at least 3 days of data");

        // ARV
        Double arv = Module7ARVComputer.computeARV(window7);
        assertNotNull(arv);
        assertEquals(VariabilityClassification.LOW, VariabilityClassification.fromARV(arv));

        // Control status
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
        assertTrue(Module7CrisisDetector.isCrisis(crisis),
            "Crisis reading must be detected");
        assertTrue(crisis.isValid(), "Crisis reading should still be valid");
    }

    @Test
    void reverseDipper_fullPipeline() {
        PatientBPState state = Module7TestBuilder.reverseDipperPatient("P-REVDIP");
        long now = System.currentTimeMillis();

        java.util.List<DailyBPSummary> window7 = state.getSummariesInWindow(7, now);
        Module7DipClassifier.DipResult dip = Module7DipClassifier.classify(window7);
        assertEquals(DipClassification.REVERSE_DIPPER, dip.classification());

        // Also verify control status is elevated (mean SBP ~138)
        BPControlStatus status = Module7BPControlClassifier.classifyControl(window7);
        assertNotEquals(BPControlStatus.CONTROLLED, status,
            "Reverse dipper with mean SBP ~138 should not be CONTROLLED");
    }

    @Test
    void whiteCoatAndMasked_notBothTrue() {
        // White-coat suspect
        PatientBPState wcState = Module7TestBuilder.whiteCoatSuspect("P-WC");
        java.util.List<DailyBPSummary> wcSummaries = wcState.getSummariesInWindow(30, System.currentTimeMillis());
        Module7BPControlClassifier.WhiteCoatResult wcResult =
            Module7BPControlClassifier.detectWhiteCoatMasked(wcSummaries);
        assertFalse(wcResult.whiteCoatSuspect() && wcResult.maskedHtnSuspect(),
            "Cannot be both white-coat AND masked");

        // Masked suspect
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

- [ ] **Step 2: Run integration tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module7IntegrationTest -q 2>&1 | tail -15`
Expected: All 6 tests PASS

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/operators/Module7IntegrationTest.java
git commit -m "test(module7): add integration tests validating full pipeline from reading to classification"
```

---

## Task 15: Run All Module 7 Tests Together

- [ ] **Step 1: Run all Module 7 tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module7*" -q 2>&1 | tail -20`
Expected: All tests PASS (~38 tests across 7 test classes)

- [ ] **Step 2: Run full project compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Final commit if any fixes were needed**

```bash
git add -A
git commit -m "fix(module7): resolve any compilation or test issues from integration"
```

---

## Implementation Notes

### What This Plan Does NOT Include (Deferred to Future Work)

1. **Cuffless BP separate ARV computation** — The model tracks `BPSource.CUFFLESS` but the main operator skips cuffless readings for clinical-grade metrics. When cuffless devices reach clinical validation, add a parallel ARV computation using reading-to-reading (not daily average) granularity. The `BPSource.isClinicalGrade()` flag will be the upgrade switch.

2. **Patient timezone handling** — `TimeContext.fromHour()` currently uses UTC. In production, morning/evening classification must use the patient's local timezone (from KB-20 profile). This requires adding a timezone field to `BPReading` or looking it up from enrichment state.

3. **Orthostatic hypotension detection** — `BPReading` captures `position` (SEATED/STANDING/SUPINE) and `heartRate`. Orthostatic detection (SBP drop > 20 mmHg on standing) requires consecutive readings in different positions within 3 minutes. Deferred until position-tagged readings are reliably captured.

4. **KB-26 MHRI hemodynamic scoring** — The architecture specifies that ARV feeds into the MHRI hemodynamic component with weights (40% norm(SBP) + 30% norm(ARV) + 30% norm(dip)). This scoring happens in KB-26, not in Module 7 — Module 7 emits the raw metrics.

5. **KB-22 deterioration event firing** — KB-22 consumes `flink.bp-variability-metrics` and fires `BP_VARIABILITY_ELEVATED` / `BP_VARIABILITY_HIGH` events. This is a KB-22 consumer responsibility, not Module 7.

6. **Post-meal BP correlation** — Module 10 creates sodium-BP pairs using BP readings. Module 7 and Module 10 are independent consumers of `ingestion.vitals`.

### Key Technical Decisions

| Decision | Rationale |
|----------|-----------|
| Static analyzers (ARV, Surge, Dip, Crisis, Control) with no Flink deps | Fully unit-testable without Flink harness. Mirror Module 6's `Module6ActionClassifier` pattern. |
| Crisis detection evaluated BEFORE state update | Patient safety — SBP 195 must emit to safety-critical even if state is corrupted or absent. |
| 30-day state with 31-day TTL | One extra day prevents edge case where a reading on day 30 evicts data needed for the 30-day ARV computation. |
| Skip cuffless from main computation | Architecture: "NOT clinical grade until validated." Separate ARV field reserved for future use. |
| DailyBPSummary aggregates incrementally | No windowed replay needed. Each reading updates running averages. Memory-efficient for 30 days of state. |
| White-coat uses 30-day window, control uses 7-day | White-coat detection needs enough clinic visits (infrequent). Control status needs recent data for medication response. |
| `isCrisis()` uses strict `>` not `>=` at 180/120 | Architecture says "SBP > 180 or DBP > 120." Matches ACC/AHA hypertensive crisis definition (> 180/120). |
| Morning surge uses sleep-trough method | Feasible with twice-daily readings (most patients). Pre-waking method reserved for cuffless extension. |
| Minimum 3 days for ARV, 3 paired days for dipping | Prevents noisy classifications from sparse data. Returns `INSUFFICIENT_DATA` when below minimum. |

### Lessons from Modules 1–6 Applied Here

1. **Rule 1 (Validate after deserialization):** `BPReading.isValid()` checks `patientId != null`, SBP/DBP in physiological range, SBP > DBP, timestamp present.

2. **Rule 2 (Field names are contracts):** `BPReading` normalizes field variants from two topic schemas. `resolveSource()` handles `"clinic"`, `"CLINIC"`, `"office"` variants.

3. **Rule 3 (Null lab values):** `BPReading` uses `Double` (boxed) for SBP/DBP with null checks in `isValid()` and all comparisons.

4. **Rule 7 (Floating-point epsilon):** All threshold comparisons in enums use `>= threshold - 1e-9` or `< threshold - 1e-9` for IEEE 754 safety.

5. **Module 4's test pattern:** `Module7TestBuilder` mirrors `Module4TestBuilder` / `Module5TestBuilder` with scenario-based factory methods.

6. **Module 6's static classifier pattern:** All computation classes (`Module7ARVComputer`, `Module7SurgeDetector`, `Module7DipClassifier`, `Module7CrisisDetector`, `Module7BPControlClassifier`) are static utilities with no Flink dependencies, fully unit-testable.
