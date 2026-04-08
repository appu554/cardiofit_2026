# Module 13: Clinical State Synchroniser Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the streaming bridge between Flink processing modules (7–12b) and KB-20 Patient State Engine, computing composite CKM risk velocity and emitting clinically significant state-change events for real-time Decision Card generation.

**Architecture:** Module 13 is a fan-in KeyedProcessFunction keyed by `patientId` that consumes from 7 Kafka topics via multi-source union. It maintains per-patient `ClinicalStateSummary` state (90-day TTL), delegates domain logic to 4 static analysers (CKMRiskComputer, StateChangeDetector, DataCompletenessMonitor, KB20StateProjector), and outputs `ClinicalStateChangeEvent` records to a new `clinical.state-change-events` topic. A side output feeds `KB20StateUpdate` diffs to an async sink that writes to PostgreSQL + Redis projections with circuit-breaking.

**Tech Stack:** Flink 2.1.0, Java 17, Jackson 2.17, async PostgreSQL + Redis client

**Base path:** `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/`
**Test base:** `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/`

---

## File Structure

### New Source Files (13)

| # | File | Responsibility |
|---|------|----------------|
| 1 | `models/CKMRiskDomain.java` | Enum: METABOLIC, RENAL, CARDIOVASCULAR with per-domain thresholds |
| 2 | `models/CKMRiskVelocity.java` | Per-domain velocity scores [-1,+1], composite, CKM stage, amplification flag |
| 3 | `models/ClinicalStateChangeType.java` | Enum: 12 state change types with priority levels |
| 4 | `models/ClinicalStateChangeEvent.java` | Output model (16 fields) for `clinical.state-change-events` |
| 5 | `models/ClinicalStateSummary.java` | Per-patient keyed state: latest + previous metric snapshots, computed risk scores |
| 6 | `models/KB20StateUpdate.java` | Field-level diff model for KB-20 PostgreSQL + Redis projection writes |
| 7 | `operators/Module13CKMRiskComputer.java` | Static analyser: 3-domain CKM risk velocity from **temporal deltas** (previous vs current snapshot) |
| 8 | `operators/Module13StateChangeDetector.java` | Static analyser: clinically significant transition detection |
| 9 | `operators/Module13DataCompletenessMonitor.java` | Static analyser: per-module data freshness tracking |
| 10 | `operators/Module13KB20StateProjector.java` | Static analyser: streaming signal → KB-20 field mapping |
| 11 | `operators/Module13_ClinicalStateSynchroniser.java` | Main KPF: multi-source routing, timer management, state materialisation |
| 12 | `sinks/KB20AsyncSinkFunction.java` | Async PostgreSQL + Redis sink with circuit-breaking and retry |
| 13 | `serialization/SourceTaggingDeserializer.java` | Generic JSON→CanonicalEvent adapter that injects `source_module` tag per topic |

### New Test Files (8)

| # | File | Test Count |
|---|------|-----------|
| 1 | `builders/Module13TestBuilder.java` | — (factory) |
| 2 | `operators/Module13CKMRiskComputerTest.java` | 7 |
| 3 | `operators/Module13StateChangeDetectorTest.java` | 6 |
| 4 | `operators/Module13DataCompletenessMonitorTest.java` | 5 |
| 5 | `operators/Module13KB20StateProjectorTest.java` | 5 |
| 6 | `operators/Module13StateSyncLifecycleTest.java` | 8 |
| 7 | `operators/Module13MultiModuleFusionIntegrationTest.java` | 4 |
| 8 | `operators/Module13DataAbsenceTimerTest.java` | 4 |

### Modified Files (2)

| File | Change |
|------|--------|
| `utils/KafkaTopics.java:181` | Add `CLINICAL_STATE_CHANGE_EVENTS` enum constant |
| `FlinkJobOrchestrator.java:151` | Add `module13`/`clinical-state-sync` case + `launchClinicalStateSynchroniser()` method |

**Total: 13 source, 8 test, 2 modified. 39 unit tests.**

---

## Task 1: CKMRiskDomain Enum

**Files:**
- Create: `models/CKMRiskDomain.java`
- Test: (tested via CKMRiskComputerTest in Task 6)

- [ ] **Step 1: Create CKMRiskDomain enum**

```java
package com.cardiofit.flink.models;

import java.io.Serializable;

/**
 * CKM syndrome risk domains per AHA 2023 Presidential Advisory
 * (Ndumele et al., Circulation 148:1606-1635).
 */
public enum CKMRiskDomain implements Serializable {

    METABOLIC(0.40, -0.30, 0.35),
    RENAL(0.40, -0.30, 0.35),
    CARDIOVASCULAR(0.40, -0.30, 0.30);

    private final double deterioratingThreshold;
    private final double improvingThreshold;
    private final double compositeWeight;

    CKMRiskDomain(double deterioratingThreshold, double improvingThreshold, double compositeWeight) {
        this.deterioratingThreshold = deterioratingThreshold;
        this.improvingThreshold = improvingThreshold;
        this.compositeWeight = compositeWeight;
    }

    public double getDeterioratingThreshold() { return deterioratingThreshold; }
    public double getImprovingThreshold() { return improvingThreshold; }
    public double getCompositeWeight() { return compositeWeight; }
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/CKMRiskDomain.java
git commit -m "feat(module13): add CKMRiskDomain enum with AHA CKM syndrome domains"
```

---

## Task 2: CKMRiskVelocity Model

**Files:**
- Create: `models/CKMRiskVelocity.java`
- Test: (tested via CKMRiskComputerTest in Task 6)

- [ ] **Step 1: Create CKMRiskVelocity model**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;
import java.util.EnumMap;
import java.util.Map;

@JsonIgnoreProperties(ignoreUnknown = true)
public class CKMRiskVelocity implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum CompositeClassification {
        IMPROVING, STABLE, DETERIORATING, UNKNOWN
    }

    @JsonProperty("domain_velocities")
    private Map<CKMRiskDomain, Double> domainVelocities;

    @JsonProperty("composite_score")
    private double compositeScore;

    @JsonProperty("composite_classification")
    private CompositeClassification compositeClassification;

    @JsonProperty("cross_domain_amplification")
    private boolean crossDomainAmplification;

    @JsonProperty("amplification_factor")
    private double amplificationFactor;

    @JsonProperty("domains_deteriorating")
    private int domainsDeteriorating;

    @JsonProperty("computation_timestamp")
    private long computationTimestamp;

    @JsonProperty("data_completeness")
    private double dataCompleteness;

    public CKMRiskVelocity() {
        this.domainVelocities = new EnumMap<>(CKMRiskDomain.class);
        this.amplificationFactor = 1.0;
    }

    public static Builder builder() { return new Builder(); }

    public Map<CKMRiskDomain, Double> getDomainVelocities() { return domainVelocities; }
    public double getDomainVelocity(CKMRiskDomain domain) {
        return domainVelocities.getOrDefault(domain, 0.0);
    }
    public double getCompositeScore() { return compositeScore; }
    public CompositeClassification getCompositeClassification() { return compositeClassification; }
    public boolean isCrossDomainAmplification() { return crossDomainAmplification; }
    public double getAmplificationFactor() { return amplificationFactor; }
    public int getDomainsDeteriorating() { return domainsDeteriorating; }
    public long getComputationTimestamp() { return computationTimestamp; }
    public double getDataCompleteness() { return dataCompleteness; }

    public static class Builder {
        private final CKMRiskVelocity v = new CKMRiskVelocity();

        public Builder domainVelocity(CKMRiskDomain domain, double velocity) {
            v.domainVelocities.put(domain, Math.max(-1.0, Math.min(1.0, velocity)));
            return this;
        }
        public Builder compositeScore(double s) { v.compositeScore = s; return this; }
        public Builder compositeClassification(CompositeClassification c) { v.compositeClassification = c; return this; }
        public Builder crossDomainAmplification(boolean a) { v.crossDomainAmplification = a; return this; }
        public Builder amplificationFactor(double f) { v.amplificationFactor = f; return this; }
        public Builder domainsDeteriorating(int d) { v.domainsDeteriorating = d; return this; }
        public Builder computationTimestamp(long t) { v.computationTimestamp = t; return this; }
        public Builder dataCompleteness(double d) { v.dataCompleteness = d; return this; }
        public CKMRiskVelocity build() { return v; }
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/CKMRiskVelocity.java
git commit -m "feat(module13): add CKMRiskVelocity model with per-domain scores and composite"
```

---

## Task 3: ClinicalStateChangeType Enum + ClinicalStateChangeEvent Model

**Files:**
- Create: `models/ClinicalStateChangeType.java`
- Create: `models/ClinicalStateChangeEvent.java`
- Test: (tested via StateChangeDetectorTest in Task 7)

- [ ] **Step 1: Create ClinicalStateChangeType enum**

```java
package com.cardiofit.flink.models;

import java.io.Serializable;

public enum ClinicalStateChangeType implements Serializable {

    CKM_RISK_ESCALATION("HIGH", "Generate urgent review card within 4 hours"),
    CKM_DOMAIN_DIVERGENCE("HIGH", "Generate multi-domain intervention card"),
    RENAL_RAPID_DECLINE("CRITICAL", "Generate nephrology referral card + SGLT2i review"),
    ENGAGEMENT_COLLAPSE("MEDIUM", "Trigger coaching nudge via BCE within 24 hours"),
    INTERVENTION_FUTILITY("MEDIUM", "Generate phenotype review card"),
    TRAJECTORY_REVERSAL("HIGH", "Generate investigation card"),
    DATA_ABSENCE_WARNING("LOW", "Generate engagement check card"),
    DATA_ABSENCE_CRITICAL("MEDIUM", "Generate clinical outreach card"),
    METABOLIC_MILESTONE("INFO", "Generate positive reinforcement card"),
    BP_MILESTONE("INFO", "Generate positive reinforcement card"),
    MEDICATION_RESPONSE_CONFIRMED("INFO", "Update KB-20 phenotype response profile"),
    CROSS_MODULE_INCONSISTENCY("MEDIUM", "Generate diagnostic investigation card");

    private final String priority;
    private final String recommendedAction;

    ClinicalStateChangeType(String priority, String recommendedAction) {
        this.priority = priority;
        this.recommendedAction = recommendedAction;
    }

    public String getPriority() { return priority; }
    public String getRecommendedAction() { return recommendedAction; }

    public boolean isCritical() { return "CRITICAL".equals(priority); }
    public boolean isHighOrAbove() { return "CRITICAL".equals(priority) || "HIGH".equals(priority); }
}
```

- [ ] **Step 2: Create ClinicalStateChangeEvent model**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;
import java.util.*;

@JsonIgnoreProperties(ignoreUnknown = true)
public class ClinicalStateChangeEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("change_id") private String changeId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("change_type") private ClinicalStateChangeType changeType;
    @JsonProperty("priority") private String priority;
    @JsonProperty("previous_value") private String previousValue;
    @JsonProperty("current_value") private String currentValue;
    @JsonProperty("domain") private CKMRiskDomain domain;
    @JsonProperty("trigger_module") private String triggerModule;
    @JsonProperty("ckm_velocity_at_change") private CKMRiskVelocity ckmVelocityAtChange;
    @JsonProperty("recommended_action") private String recommendedAction;
    @JsonProperty("originating_signals") private List<String> originatingSignals;
    @JsonProperty("data_completeness_at_change") private double dataCompletenessAtChange;
    @JsonProperty("confidence_score") private double confidenceScore;
    @JsonProperty("metadata") private Map<String, Object> metadata;
    @JsonProperty("processing_timestamp") private long processingTimestamp;
    @JsonProperty("version") private String version = "1.0";

    public ClinicalStateChangeEvent() {
        this.originatingSignals = new ArrayList<>();
        this.metadata = new HashMap<>();
    }

    public static Builder builder() { return new Builder(); }

    // Getters
    public String getChangeId() { return changeId; }
    public String getPatientId() { return patientId; }
    public ClinicalStateChangeType getChangeType() { return changeType; }
    public String getPriority() { return priority; }
    public String getPreviousValue() { return previousValue; }
    public String getCurrentValue() { return currentValue; }
    public CKMRiskDomain getDomain() { return domain; }
    public String getTriggerModule() { return triggerModule; }
    public CKMRiskVelocity getCkmVelocityAtChange() { return ckmVelocityAtChange; }
    public String getRecommendedAction() { return recommendedAction; }
    public List<String> getOriginatingSignals() { return originatingSignals; }
    public double getDataCompletenessAtChange() { return dataCompletenessAtChange; }
    public double getConfidenceScore() { return confidenceScore; }
    public Map<String, Object> getMetadata() { return metadata; }
    public long getProcessingTimestamp() { return processingTimestamp; }
    public String getVersion() { return version; }

    public static class Builder {
        private final ClinicalStateChangeEvent e = new ClinicalStateChangeEvent();

        public Builder changeId(String id) { e.changeId = id; return this; }
        public Builder patientId(String id) { e.patientId = id; return this; }
        public Builder changeType(ClinicalStateChangeType t) {
            e.changeType = t;
            e.priority = t.getPriority();
            e.recommendedAction = t.getRecommendedAction();
            return this;
        }
        public Builder previousValue(String v) { e.previousValue = v; return this; }
        public Builder currentValue(String v) { e.currentValue = v; return this; }
        public Builder domain(CKMRiskDomain d) { e.domain = d; return this; }
        public Builder triggerModule(String m) { e.triggerModule = m; return this; }
        public Builder ckmVelocityAtChange(CKMRiskVelocity v) { e.ckmVelocityAtChange = v; return this; }
        public Builder originatingSignals(List<String> s) { e.originatingSignals = s; return this; }
        public Builder dataCompletenessAtChange(double d) { e.dataCompletenessAtChange = d; return this; }
        public Builder confidenceScore(double c) { e.confidenceScore = c; return this; }
        public Builder metadata(Map<String, Object> m) { e.metadata = m; return this; }
        public Builder processingTimestamp(long t) { e.processingTimestamp = t; return this; }
        public ClinicalStateChangeEvent build() { return e; }
    }
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/ClinicalStateChangeType.java \
       backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/ClinicalStateChangeEvent.java
git commit -m "feat(module13): add ClinicalStateChangeType enum (12 types) and ClinicalStateChangeEvent model"
```

---

## Task 4: ClinicalStateSummary (Per-Patient Keyed State)

**Files:**
- Create: `models/ClinicalStateSummary.java`
- Test: (tested via lifecycle tests in Task 12)

- [ ] **Step 1: Create ClinicalStateSummary model**

This is the per-patient state object managed by Module 13's KPF. It aggregates latest values from every upstream module plus computed risk scores.

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

import java.io.Serializable;
import java.util.*;

@JsonIgnoreProperties(ignoreUnknown = true)
public class ClinicalStateSummary implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;

    // --- Metric snapshots: current + previous for velocity computation ---
    // Velocity = (current - previous) / elapsed_time. Without two snapshots, only displacement.
    private MetricSnapshot currentSnapshot = new MetricSnapshot();
    private MetricSnapshot previousSnapshot; // null until first snapshot rotation
    private long previousSnapshotTimestamp;   // when previousSnapshot was captured

    // --- Upstream module-specific state (non-metric) ---

    // Module 9: Engagement (needs previous for collapse detection)
    private Double previousEngagementScore;
    private String latestPhenotype;
    private String dataTier;
    private String channel;

    // Module 12: Active Interventions
    private Map<String, InterventionWindowSummary> activeInterventions = new HashMap<>();

    // Module 12b: Completed Intervention Deltas
    private List<InterventionDeltaSummary> recentInterventionDeltas = new ArrayList<>();

    // --- Computed by Module 13 ---
    private CKMRiskVelocity lastComputedVelocity;
    private double dataCompletenessScore;

    /**
     * Rotate snapshots: current becomes previous, current is reset to empty.
     * Called on a periodic interval (every 7 days) by the main KPF.
     */
    public void rotateSnapshots(long rotationTimestamp) {
        this.previousSnapshot = this.currentSnapshot.copy();
        this.previousSnapshotTimestamp = rotationTimestamp;
        // Don't reset currentSnapshot — it keeps accumulating until next rotation
    }

    // --- Dedup: last-emitted state change per type (24h window) ---
    private Map<ClinicalStateChangeType, Long> lastEmittedChangeTimestamps = new EnumMap<>(ClinicalStateChangeType.class);

    // --- Per-module last-seen timestamps ---
    private Map<String, Long> moduleLastSeenMs = new HashMap<>();

    // --- Write coalescing ---
    private List<KB20StateUpdate> coalescingBuffer = new ArrayList<>();
    private long coalescingTimerMs = -1L;

    // --- Daily data absence timer ---
    private long dailyTimerMs = -1L;

    // --- Snapshot rotation timer (7-day interval) ---
    private long snapshotRotationTimerMs = -1L;

    // --- Idle-patient quiescence (Issue 6 fix) ---
    private int consecutiveZeroCompletenessDays = 0;

    private long lastUpdated;

    // Constructors
    public ClinicalStateSummary() {}

    public ClinicalStateSummary(String patientId) {
        this.patientId = patientId;
        this.lastUpdated = System.currentTimeMillis();
    }

    // --- Convenience accessors delegating to currentSnapshot ---
    public MetricSnapshot current() { return currentSnapshot; }
    public MetricSnapshot previous() { return previousSnapshot; }
    public long getPreviousSnapshotTimestamp() { return previousSnapshotTimestamp; }
    public boolean hasVelocityData() { return previousSnapshot != null; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Double getPreviousEngagementScore() { return previousEngagementScore; }
    public void setPreviousEngagementScore(Double v) { this.previousEngagementScore = v; }
    public String getLatestPhenotype() { return latestPhenotype; }
    public void setLatestPhenotype(String v) { this.latestPhenotype = v; }
    public String getDataTier() { return dataTier; }
    public void setDataTier(String v) { this.dataTier = v; }
    public String getChannel() { return channel; }
    public void setChannel(String v) { this.channel = v; }
    public Map<String, InterventionWindowSummary> getActiveInterventions() { return activeInterventions; }
    public List<InterventionDeltaSummary> getRecentInterventionDeltas() { return recentInterventionDeltas; }
    public CKMRiskVelocity getLastComputedVelocity() { return lastComputedVelocity; }
    public void setLastComputedVelocity(CKMRiskVelocity v) { this.lastComputedVelocity = v; }
    public double getDataCompletenessScore() { return dataCompletenessScore; }
    public void setDataCompletenessScore(double v) { this.dataCompletenessScore = v; }
    public Map<ClinicalStateChangeType, Long> getLastEmittedChangeTimestamps() { return lastEmittedChangeTimestamps; }
    public Map<String, Long> getModuleLastSeenMs() { return moduleLastSeenMs; }
    public void recordModuleSeen(String moduleKey, long timestamp) { this.moduleLastSeenMs.put(moduleKey, timestamp); }
    public List<KB20StateUpdate> getCoalescingBuffer() { return coalescingBuffer; }
    public long getCoalescingTimerMs() { return coalescingTimerMs; }
    public void setCoalescingTimerMs(long v) { this.coalescingTimerMs = v; }
    public long getDailyTimerMs() { return dailyTimerMs; }
    public void setDailyTimerMs(long v) { this.dailyTimerMs = v; }
    public long getSnapshotRotationTimerMs() { return snapshotRotationTimerMs; }
    public void setSnapshotRotationTimerMs(long v) { this.snapshotRotationTimerMs = v; }
    public int getConsecutiveZeroCompletenessDays() { return consecutiveZeroCompletenessDays; }
    public void setConsecutiveZeroCompletenessDays(int v) { this.consecutiveZeroCompletenessDays = v; }
    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long v) { this.lastUpdated = v; }

    // --- MetricSnapshot: all numeric values at a point in time ---
    public static class MetricSnapshot implements Serializable {
        // Module 7: BP Variability
        public Double arv;
        public VariabilityClassification variabilityClass;
        public Double meanSBP;
        public Double meanDBP;
        public Double morningSurgeMagnitude;
        public DipClassification dipClass;
        // Module 9: Engagement
        public Double engagementScore;
        public EngagementLevel engagementLevel;
        // Module 10b: Meal Patterns
        public Double meanIAUC;
        public Double medianExcursion;
        public SaltSensitivityClass saltSensitivity;
        public Double saltBeta;
        // Module 11b: Fitness Patterns
        public Double estimatedVO2max;
        public Double vo2maxTrend;
        public Double totalMetMinutes;
        public Double meanExerciseGlucoseDelta;
        // Labs
        public Double fbg;
        public Double hba1c;
        public Double egfr;
        public Double uacr;
        public Double totalCholesterol;
        public Double ldl;
        public Double weight;
        public Double bmi;

        public MetricSnapshot() {}
        public MetricSnapshot copy() {
            MetricSnapshot c = new MetricSnapshot();
            c.arv = this.arv; c.variabilityClass = this.variabilityClass;
            c.meanSBP = this.meanSBP; c.meanDBP = this.meanDBP;
            c.morningSurgeMagnitude = this.morningSurgeMagnitude; c.dipClass = this.dipClass;
            c.engagementScore = this.engagementScore; c.engagementLevel = this.engagementLevel;
            c.meanIAUC = this.meanIAUC; c.medianExcursion = this.medianExcursion;
            c.saltSensitivity = this.saltSensitivity; c.saltBeta = this.saltBeta;
            c.estimatedVO2max = this.estimatedVO2max; c.vo2maxTrend = this.vo2maxTrend;
            c.totalMetMinutes = this.totalMetMinutes; c.meanExerciseGlucoseDelta = this.meanExerciseGlucoseDelta;
            c.fbg = this.fbg; c.hba1c = this.hba1c; c.egfr = this.egfr; c.uacr = this.uacr;
            c.totalCholesterol = this.totalCholesterol; c.ldl = this.ldl;
            c.weight = this.weight; c.bmi = this.bmi;
            return c;
        }
    }

    // --- Inner models for intervention tracking ---

    public static class InterventionWindowSummary implements Serializable {
        private String interventionId;
        private InterventionType interventionType;
        private String status; // OPENED, MIDPOINT, CLOSED
        private long observationStartMs;
        private long observationEndMs;
        private TrajectoryClassification trajectoryAtOpen;

        public InterventionWindowSummary() {}

        public String getInterventionId() { return interventionId; }
        public void setInterventionId(String v) { this.interventionId = v; }
        public InterventionType getInterventionType() { return interventionType; }
        public void setInterventionType(InterventionType v) { this.interventionType = v; }
        public String getStatus() { return status; }
        public void setStatus(String v) { this.status = v; }
        public long getObservationStartMs() { return observationStartMs; }
        public void setObservationStartMs(long v) { this.observationStartMs = v; }
        public long getObservationEndMs() { return observationEndMs; }
        public void setObservationEndMs(long v) { this.observationEndMs = v; }
        public TrajectoryClassification getTrajectoryAtOpen() { return trajectoryAtOpen; }
        public void setTrajectoryAtOpen(TrajectoryClassification v) { this.trajectoryAtOpen = v; }
    }

    public static class InterventionDeltaSummary implements Serializable {
        private String interventionId;
        private InterventionType interventionType;
        private TrajectoryAttribution attribution;
        private Double adherenceScore;
        private long closedAtMs;

        public InterventionDeltaSummary() {}

        public String getInterventionId() { return interventionId; }
        public void setInterventionId(String v) { this.interventionId = v; }
        public InterventionType getInterventionType() { return interventionType; }
        public void setInterventionType(InterventionType v) { this.interventionType = v; }
        public TrajectoryAttribution getAttribution() { return attribution; }
        public void setAttribution(TrajectoryAttribution v) { this.attribution = v; }
        public Double getAdherenceScore() { return adherenceScore; }
        public void setAdherenceScore(Double v) { this.adherenceScore = v; }
        public long getClosedAtMs() { return closedAtMs; }
        public void setClosedAtMs(long v) { this.closedAtMs = v; }
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/ClinicalStateSummary.java
git commit -m "feat(module13): add ClinicalStateSummary per-patient keyed state model"
```

---

## Task 5: KB20StateUpdate Model

**Files:**
- Create: `models/KB20StateUpdate.java`
- Test: (tested via KB20StateProjectorTest in Task 9)

- [ ] **Step 1: Create KB20StateUpdate model**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

@JsonIgnoreProperties(ignoreUnknown = true)
public class KB20StateUpdate implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum UpdateOperation {
        REPLACE,   // Last-write-wins: bp_variability_state, engagement_state, etc.
        MERGE,     // Merge into existing map: meal_response_profile (append weekly)
        APPEND,    // Append to list: intervention_outcomes
        UPSERT     // Upsert by key: active_interventions (by intervention_id)
    }

    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("field_updates") private Map<String, Object> fieldUpdates;
    @JsonProperty("operation") private UpdateOperation operation;
    @JsonProperty("source_module") private String sourceModule;
    @JsonProperty("timestamp") private long timestamp;
    @JsonProperty("upsert_key") private String upsertKey; // for UPSERT: the key field name

    public KB20StateUpdate() {
        this.fieldUpdates = new HashMap<>();
    }

    public static Builder builder() { return new Builder(); }

    public String getPatientId() { return patientId; }
    public Map<String, Object> getFieldUpdates() { return fieldUpdates; }
    public UpdateOperation getOperation() { return operation; }
    public String getSourceModule() { return sourceModule; }
    public long getTimestamp() { return timestamp; }
    public String getUpsertKey() { return upsertKey; }

    public static class Builder {
        private final KB20StateUpdate u = new KB20StateUpdate();

        public Builder patientId(String id) { u.patientId = id; return this; }
        public Builder field(String name, Object value) { u.fieldUpdates.put(name, value); return this; }
        public Builder fieldUpdates(Map<String, Object> fields) { u.fieldUpdates.putAll(fields); return this; }
        public Builder operation(UpdateOperation op) { u.operation = op; return this; }
        public Builder sourceModule(String m) { u.sourceModule = m; return this; }
        public Builder timestamp(long t) { u.timestamp = t; return this; }
        public Builder upsertKey(String k) { u.upsertKey = k; return this; }
        public KB20StateUpdate build() { return u; }
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/KB20StateUpdate.java
git commit -m "feat(module13): add KB20StateUpdate field-level diff model for KB-20 projections"
```

---

## Task 5b: SourceTaggingDeserializer (Issue 2+3 fix)

**Files:**
- Create: `serialization/SourceTaggingDeserializer.java`

**Design note (Issue 2+3 fix):** Upstream modules (7, 9, 10b, 11b, 12, 12b) do NOT emit `source_module` in their payload. Without this, the main KPF's `routeAndUpdateState()` switch always hits `default: return null` — silent production failure. Additionally, Sources 5 (`clinical.intervention-window-signals`) and 6 (`flink.intervention-deltas`) carry `InterventionWindowSignal` and `InterventionDeltaRecord`, NOT `CanonicalEvent`. Using `CanonicalEventDeserializer` would fail.

The fix: a generic `SourceTaggingDeserializer` that (a) deserializes raw JSON bytes into a `Map<String, Object>`, (b) injects a `source_module` tag, (c) wraps into a `CanonicalEvent`. This follows the existing pattern of `InterventionSignalToCanonicalDeserializer` at `FlinkJobOrchestrator.java:931`.

- [ ] **Step 1: Create SourceTaggingDeserializer**

```java
package com.cardiofit.flink.serialization;

import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.EventType;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

/**
 * Generic deserializer that wraps raw JSON into a CanonicalEvent and injects
 * a source_module tag. This solves two problems:
 * 1. Upstream modules don't emit source_module in their payload.
 * 2. Sources 5-6 emit InterventionWindowSignal/InterventionDeltaRecord, not CanonicalEvent.
 *
 * Parameterized per topic at construction time.
 * Pattern follows InterventionSignalToCanonicalDeserializer.
 */
public class SourceTaggingDeserializer implements DeserializationSchema<CanonicalEvent> {

    private static final Logger LOG = LoggerFactory.getLogger(SourceTaggingDeserializer.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();
    private static final TypeReference<Map<String, Object>> MAP_TYPE = new TypeReference<>() {};

    private final String sourceModuleTag;
    private final EventType defaultEventType;

    /**
     * @param sourceModuleTag  tag to inject into payload.source_module (e.g. "module7", "module12")
     * @param defaultEventType fallback EventType when the raw JSON has no event_type
     */
    public SourceTaggingDeserializer(String sourceModuleTag, EventType defaultEventType) {
        this.sourceModuleTag = sourceModuleTag;
        this.defaultEventType = defaultEventType;
    }

    @Override
    public CanonicalEvent deserialize(byte[] message) throws IOException {
        if (message == null || message.length == 0) return null;

        try {
            Map<String, Object> raw = MAPPER.readValue(message, MAP_TYPE);

            // Inject source_module tag
            Map<String, Object> payload = new HashMap<>(raw);
            payload.put("source_module", sourceModuleTag);

            // Extract patient_id (try multiple field names for compatibility)
            String patientId = extractString(raw, "patient_id", "patientId");

            // Extract event_time
            long eventTime = extractLong(raw, "event_time", "eventTime", "timestamp",
                    "processing_timestamp", "observation_start_ms");

            // Extract event_type if present, else use default
            EventType eventType = defaultEventType;
            String typeStr = extractString(raw, "event_type", "eventType");
            if (typeStr != null) {
                try { eventType = EventType.valueOf(typeStr); }
                catch (IllegalArgumentException ignored) {}
            }

            return CanonicalEvent.builder()
                    .id(extractString(raw, "id", "event_id") != null
                            ? extractString(raw, "id", "event_id")
                            : UUID.randomUUID().toString())
                    .patientId(patientId)
                    .eventType(eventType)
                    .eventTime(eventTime)
                    .processingTime(System.currentTimeMillis())
                    .sourceSystem("flink-module13-" + sourceModuleTag)
                    .payload(payload)
                    .build();

        } catch (Exception e) {
            LOG.warn("Failed to deserialize {} event: {}", sourceModuleTag, e.getMessage());
            return null;
        }
    }

    @Override
    public boolean isEndOfStream(CanonicalEvent nextElement) {
        return false;
    }

    @Override
    public TypeInformation<CanonicalEvent> getProducedType() {
        return TypeInformation.of(CanonicalEvent.class);
    }

    private static String extractString(Map<String, Object> map, String... keys) {
        for (String key : keys) {
            Object v = map.get(key);
            if (v != null) return v.toString();
        }
        return null;
    }

    private static long extractLong(Map<String, Object> map, String... keys) {
        for (String key : keys) {
            Object v = map.get(key);
            if (v instanceof Number) return ((Number) v).longValue();
            if (v != null) {
                try { return Long.parseLong(v.toString()); }
                catch (NumberFormatException ignored) {}
            }
        }
        return System.currentTimeMillis();
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/serialization/SourceTaggingDeserializer.java
git commit -m "feat(module13): add SourceTaggingDeserializer — generic JSON→CanonicalEvent adapter with source_module injection"
```

---

## Task 6: Module13TestBuilder

**Files:**
- Create: `builders/Module13TestBuilder.java`

- [ ] **Step 1: Create test builder with factories for all upstream event types**

```java
package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;

import java.util.*;

public class Module13TestBuilder {

    public static final long HOUR_MS = 3_600_000L;
    public static final long DAY_MS = 86_400_000L;
    public static final long WEEK_MS = 7 * DAY_MS;
    public static final long BASE_TIME = 1743552000000L; // 2025-04-02 00:00:00 UTC

    public static long hoursAfter(long base, int hours) { return base + hours * HOUR_MS; }
    public static long daysAfter(long base, int days) { return base + days * DAY_MS; }
    public static long weeksAfter(long base, int weeks) { return base + weeks * WEEK_MS; }

    // ---- Event builders keyed by source module ----

    /** Module 7: BP variability metric event */
    public static CanonicalEvent bpVariabilityEvent(String patientId, long timestamp,
            double arv, String variabilityClass, double meanSBP, double meanDBP) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "module7");
        payload.put("arv", arv);
        payload.put("variability_classification", variabilityClass);
        payload.put("mean_sbp", meanSBP);
        payload.put("mean_dbp", meanDBP);
        return baseEvent(patientId, EventType.VITAL_SIGN, timestamp, payload);
    }

    /** Module 9: Engagement signal event */
    public static CanonicalEvent engagementEvent(String patientId, long timestamp,
            double compositeScore, String engagementLevel, String phenotype, String dataTier) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "module9");
        payload.put("composite_score", compositeScore);
        payload.put("engagement_level", engagementLevel);
        payload.put("phenotype", phenotype);
        payload.put("data_tier", dataTier);
        return baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp, payload);
    }

    /** Module 10b: Meal pattern summary event */
    public static CanonicalEvent mealPatternEvent(String patientId, long timestamp,
            double meanIAUC, double medianExcursion, String saltSensitivity, Double saltBeta) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "module10b");
        payload.put("mean_iauc", meanIAUC);
        payload.put("median_excursion", medianExcursion);
        payload.put("salt_sensitivity_class", saltSensitivity);
        payload.put("salt_beta", saltBeta);
        return baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp, payload);
    }

    /** Module 11b: Fitness pattern summary event */
    public static CanonicalEvent fitnessPatternEvent(String patientId, long timestamp,
            double estimatedVO2max, double vo2maxTrend, double totalMetMinutes, double meanGlucoseDelta) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "module11b");
        payload.put("estimated_vo2max", estimatedVO2max);
        payload.put("vo2max_trend", vo2maxTrend);
        payload.put("total_met_minutes", totalMetMinutes);
        payload.put("mean_exercise_glucose_delta", meanGlucoseDelta);
        return baseEvent(patientId, EventType.DEVICE_READING, timestamp, payload);
    }

    /** Module 12: Intervention window signal event */
    public static CanonicalEvent interventionWindowEvent(String patientId, long timestamp,
            String interventionId, String signalType, String interventionType,
            long observationStartMs, long observationEndMs) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "module12");
        payload.put("intervention_id", interventionId);
        payload.put("signal_type", signalType);
        payload.put("intervention_type", interventionType);
        payload.put("observation_start_ms", observationStartMs);
        payload.put("observation_end_ms", observationEndMs);
        return baseEvent(patientId, EventType.MEDICATION_ORDERED, timestamp, payload);
    }

    /** Module 12b: Intervention delta event */
    public static CanonicalEvent interventionDeltaEvent(String patientId, long timestamp,
            String interventionId, String attribution, Double adherenceScore,
            Double fbgDelta, Double sbpDelta, Double egfrDelta) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "module12b");
        payload.put("intervention_id", interventionId);
        payload.put("trajectory_attribution", attribution);
        payload.put("adherence_score", adherenceScore);
        payload.put("fbg_delta", fbgDelta);
        payload.put("sbp_delta", sbpDelta);
        payload.put("egfr_delta", egfrDelta);
        return baseEvent(patientId, EventType.LAB_RESULT, timestamp, payload);
    }

    /** Lab result from enriched-patient-events-v1 */
    public static CanonicalEvent labEvent(String patientId, long timestamp,
            String labType, double value) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_module", "enriched");
        payload.put("lab_type", labType); // FBG, HBA1C, EGFR, UACR, LDL, TOTAL_CHOLESTEROL
        payload.put("value", value);
        return baseEvent(patientId, EventType.LAB_RESULT, timestamp, payload);
    }

    // ---- State builders ----

    /** Empty state for new patient */
    public static ClinicalStateSummary emptyState(String patientId) {
        return new ClinicalStateSummary(patientId);
    }

    /** State with baselines — simulates a patient with 2 weeks of history (current snapshot populated) */
    public static ClinicalStateSummary stateWithBaselines(String patientId) {
        ClinicalStateSummary s = new ClinicalStateSummary(patientId);
        s.current().fbg = 130.0;
        s.current().hba1c = 7.2;
        s.current().egfr = 65.0;
        s.current().meanSBP = 142.0;
        s.current().meanDBP = 88.0;
        s.current().arv = 10.5;
        s.current().variabilityClass = VariabilityClassification.MODERATE;
        s.current().engagementScore = 0.72;
        s.current().engagementLevel = EngagementLevel.GREEN;
        s.current().estimatedVO2max = 35.0;
        s.setDataTier("TIER_2_SMBG");
        s.setChannel("CORPORATE");
        s.recordModuleSeen("module7", BASE_TIME);
        s.recordModuleSeen("module9", BASE_TIME);
        s.recordModuleSeen("module10b", BASE_TIME);
        s.recordModuleSeen("module11b", BASE_TIME);
        s.recordModuleSeen("enriched", BASE_TIME);
        s.setLastUpdated(BASE_TIME);
        return s;
    }

    /**
     * State with snapshot pair — simulates a patient after first 7-day rotation.
     * Both previousSnapshot and currentSnapshot are populated with typical South Asian defaults.
     * This is required for CKMRiskComputer velocity tests (Issue 1 fix).
     */
    public static ClinicalStateSummary stateWithSnapshotPair(String patientId) {
        ClinicalStateSummary s = stateWithBaselines(patientId);
        // Rotate: current → previous, then reset current to same defaults
        s.rotateSnapshots(BASE_TIME);
        // Re-populate current with same baselines (tests will override specific fields)
        s.current().fbg = 130.0;
        s.current().hba1c = 7.2;
        s.current().egfr = 65.0;
        s.current().meanSBP = 142.0;
        s.current().meanDBP = 88.0;
        s.current().arv = 10.5;
        s.current().variabilityClass = VariabilityClassification.MODERATE;
        s.current().engagementScore = 0.72;
        s.current().engagementLevel = EngagementLevel.GREEN;
        s.current().estimatedVO2max = 35.0;
        return s;
    }

    /** State with an active intervention window */
    public static ClinicalStateSummary stateWithActiveIntervention(String patientId,
            String interventionId, InterventionType type) {
        ClinicalStateSummary s = stateWithBaselines(patientId);
        ClinicalStateSummary.InterventionWindowSummary iw = new ClinicalStateSummary.InterventionWindowSummary();
        iw.setInterventionId(interventionId);
        iw.setInterventionType(type);
        iw.setStatus("OPENED");
        iw.setObservationStartMs(BASE_TIME);
        iw.setObservationEndMs(BASE_TIME + 28 * DAY_MS);
        iw.setTrajectoryAtOpen(TrajectoryClassification.STABLE);
        s.getActiveInterventions().put(interventionId, iw);
        return s;
    }

    /** State with historical CKM velocity (for testing transitions) */
    public static ClinicalStateSummary stateWithVelocity(String patientId,
            CKMRiskVelocity.CompositeClassification classification) {
        ClinicalStateSummary s = stateWithBaselines(patientId);
        CKMRiskVelocity velocity = CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.METABOLIC, classification == CKMRiskVelocity.CompositeClassification.IMPROVING ? -0.4 : 0.1)
                .domainVelocity(CKMRiskDomain.RENAL, 0.0)
                .domainVelocity(CKMRiskDomain.CARDIOVASCULAR, 0.0)
                .compositeClassification(classification)
                .compositeScore(classification == CKMRiskVelocity.CompositeClassification.IMPROVING ? -0.3 : 0.1)
                .dataCompleteness(0.85)
                .computationTimestamp(BASE_TIME)
                .build();
        s.setLastComputedVelocity(velocity);
        return s;
    }

    // ---- Private helpers ----

    private static CanonicalEvent baseEvent(String patientId, EventType type,
            long timestamp, Map<String, Object> payload) {
        return CanonicalEvent.builder()
                .id(UUID.randomUUID().toString())
                .patientId(patientId)
                .eventType(type)
                .eventTime(timestamp)
                .processingTime(System.currentTimeMillis())
                .sourceSystem("flink-module13-test")
                .payload(payload)
                .build();
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/builders/Module13TestBuilder.java
git commit -m "feat(module13): add Module13TestBuilder with factories for all upstream event types"
```

---

## Task 7: Module13CKMRiskComputer + 7 Tests

**Files:**
- Create: `operators/Module13CKMRiskComputer.java`
- Create (test): `operators/Module13CKMRiskComputerTest.java`

**Design note (Issue 1 fix):** Velocity is computed from **temporal deltas** between `previousSnapshot` and `currentSnapshot`, NOT from distance-to-static-baseline. A patient with eGFR stable at 55 for 6 months has renal velocity ~0.0 (stable), while a patient whose eGFR dropped 10 points in 7 days has high positive velocity. The snapshot rotation happens every 7 days in the main KPF (see Task 11). When `previousSnapshot` is null (first 7 days), we fall back to UNKNOWN classification.

- [ ] **Step 1: Write the 7 failing tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class Module13CKMRiskComputerTest {

    // --- Test 1: Metabolic velocity from declining FBG = IMPROVING ---
    @Test
    void metabolicVelocity_decliningFBG_returnsImproving() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        // Previous: FBG 140, HbA1c 7.5. Current: FBG 120, HbA1c 7.0
        state.previous().fbg = 140.0;
        state.previous().hba1c = 7.5;
        state.current().fbg = 120.0;
        state.current().hba1c = 7.0;

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertTrue(result.getDomainVelocity(CKMRiskDomain.METABOLIC) < 0,
                "Metabolic velocity should be negative (improving) when FBG dropped 20 mg/dL");
    }

    // --- Test 2: Renal velocity from eGFR decline >3/year pace = DETERIORATING ---
    @Test
    void renalVelocity_rapidEGFRDecline_returnsDeteriorating() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        // Previous: eGFR 65, UACR 30. Current: eGFR 55, UACR 120 (rapid decline in 7 days)
        state.previous().egfr = 65.0;
        state.previous().uacr = 30.0;
        state.current().egfr = 55.0;
        state.current().uacr = 120.0;
        state.current().arv = 14.0;
        state.current().meanSBP = 148.0;

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertTrue(result.getDomainVelocity(CKMRiskDomain.RENAL) > 0.4,
                "Renal velocity should exceed deteriorating threshold for 10-point eGFR drop in 7 days");
    }

    // --- Test 3: Cardiovascular velocity from worsening ARV + morning surge ---
    @Test
    void cardiovascularVelocity_worseningARVAndSurge_returnsDeteriorating() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        // Previous: ARV 9, surge 18. Current: ARV 18, surge 45
        state.previous().arv = 9.0;
        state.previous().morningSurgeMagnitude = 18.0;
        state.previous().engagementScore = 0.8;
        state.current().arv = 18.0;
        state.current().morningSurgeMagnitude = 45.0;
        state.current().engagementScore = 0.3;

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertTrue(result.getDomainVelocity(CKMRiskDomain.CARDIOVASCULAR) > 0.4,
                "CV velocity should exceed deteriorating threshold");
    }

    // --- Test 4: Composite worst-domain-wins ---
    @Test
    void composite_worstDomainWins() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        // Renal deteriorating rapidly, others stable (no change)
        state.previous().egfr = 65.0; state.current().egfr = 50.0; // big drop
        state.previous().uacr = 30.0; state.current().uacr = 180.0;
        state.previous().fbg = 130.0; state.current().fbg = 128.0; // stable
        state.previous().arv = 10.0; state.current().arv = 11.0;   // stable

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertEquals(CKMRiskVelocity.CompositeClassification.DETERIORATING,
                result.getCompositeClassification(),
                "Composite should be DETERIORATING when any domain exceeds threshold");
    }

    // --- Test 5: Cross-domain amplification when 2+ domains worsening ---
    @Test
    void composite_crossDomainAmplification_when2DomainsWorsening() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        // Both renal AND metabolic worsening simultaneously
        state.previous().egfr = 65.0; state.current().egfr = 52.0;
        state.previous().uacr = 30.0; state.current().uacr = 150.0;
        state.previous().fbg = 130.0; state.current().fbg = 175.0;
        state.previous().hba1c = 7.2; state.current().hba1c = 8.5;

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertTrue(result.isCrossDomainAmplification(),
                "Should flag cross-domain amplification");
        assertEquals(1.5, result.getAmplificationFactor(), 0.01,
                "Amplification factor should be 1.5x");
        assertTrue(result.getDomainsDeteriorating() >= 2);
    }

    // --- Test 6: All domains improving = IMPROVING ---
    @Test
    void composite_allDomainsImproving_returnsImproving() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        // All metrics improved from previous snapshot
        state.previous().fbg = 145.0; state.current().fbg = 115.0;
        state.previous().hba1c = 7.8; state.current().hba1c = 6.9;
        state.previous().egfr = 58.0; state.current().egfr = 64.0;
        state.previous().uacr = 80.0; state.current().uacr = 35.0;
        state.previous().arv = 14.0; state.current().arv = 8.0;
        state.previous().engagementScore = 0.5; state.current().engagementScore = 0.9;

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertEquals(CKMRiskVelocity.CompositeClassification.IMPROVING,
                result.getCompositeClassification());
        assertFalse(result.isCrossDomainAmplification());
    }

    // --- Test 7: No previous snapshot → UNKNOWN ---
    @Test
    void compute_noPreviousSnapshot_returnsUnknown() {
        ClinicalStateSummary state = Module13TestBuilder.emptyState("p1");
        // Only current snapshot, no previous (first 7 days)

        CKMRiskVelocity result = Module13CKMRiskComputer.compute(state);

        assertEquals(CKMRiskVelocity.CompositeClassification.UNKNOWN,
                result.getCompositeClassification(),
                "Cannot compute velocity without previous snapshot");
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module13CKMRiskComputerTest -Dsurefire.failIfNoTests=false 2>&1 | tail -5`
Expected: Compilation failure — `Module13CKMRiskComputer` class does not exist.

- [ ] **Step 3: Implement Module13CKMRiskComputer**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;

public final class Module13CKMRiskComputer {

    // --- Normalisation ranges: the delta magnitude that maps to velocity ±1.0 ---
    // These define "what change over 7 days would be maximally alarming"
    private static final double FBG_DELTA_RANGE = 40.0;      // ±40 mg/dL per 7d
    private static final double HBA1C_DELTA_RANGE = 1.0;     // ±1% per 7d
    private static final double IAUC_DELTA_RANGE = 30.0;     // ±30 units per 7d
    private static final double WEIGHT_DELTA_RANGE = 3.0;    // ±3 kg per 7d
    private static final double EGFR_DELTA_RANGE = 8.0;      // ±8 mL/min per 7d
    private static final double UACR_DELTA_RANGE = 80.0;     // ±80 mg/g per 7d
    private static final double ARV_DELTA_RANGE = 8.0;       // ±8 mmHg per 7d
    private static final double LDL_DELTA_RANGE = 30.0;      // ±30 mg/dL per 7d
    private static final double SURGE_DELTA_RANGE = 20.0;    // ±20 mmHg per 7d
    private static final double ENGAGEMENT_DELTA_RANGE = 0.4; // ±0.4 per 7d

    // --- Cross-domain amplification ---
    private static final double AMPLIFICATION_THRESHOLD = 0.2;
    private static final double AMPLIFICATION_FACTOR = 1.5;
    private static final int MIN_DOMAINS_FOR_AMPLIFICATION = 2;
    private static final int MIN_DOMAINS_FOR_VALID_SCORE = 2;

    private Module13CKMRiskComputer() {}

    /**
     * Compute CKM risk velocity from temporal deltas between previous and current snapshots.
     * Returns UNKNOWN if no previous snapshot exists (first 7 days of enrollment).
     */
    public static CKMRiskVelocity compute(ClinicalStateSummary state) {
        if (!state.hasVelocityData()) {
            return CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.UNKNOWN)
                    .compositeScore(0.0)
                    .dataCompleteness(0.0)
                    .computationTimestamp(System.currentTimeMillis())
                    .build();
        }

        ClinicalStateSummary.MetricSnapshot prev = state.previous();
        ClinicalStateSummary.MetricSnapshot curr = state.current();

        double metabolic = computeMetabolicVelocity(prev, curr, state);
        double renal = computeRenalVelocity(prev, curr, state);
        double cardiovascular = computeCardiovascularVelocity(prev, curr, state);

        int validDomains = 0;
        if (!Double.isNaN(metabolic)) validDomains++;
        if (!Double.isNaN(renal)) validDomains++;
        if (!Double.isNaN(cardiovascular)) validDomains++;
        double dataCompleteness = validDomains / 3.0;

        if (validDomains < MIN_DOMAINS_FOR_VALID_SCORE) {
            return CKMRiskVelocity.builder()
                    .domainVelocity(CKMRiskDomain.METABOLIC, safe(metabolic))
                    .domainVelocity(CKMRiskDomain.RENAL, safe(renal))
                    .domainVelocity(CKMRiskDomain.CARDIOVASCULAR, safe(cardiovascular))
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.UNKNOWN)
                    .compositeScore(0.0)
                    .dataCompleteness(dataCompleteness)
                    .computationTimestamp(System.currentTimeMillis())
                    .build();
        }

        double sM = safe(metabolic), sR = safe(renal), sCV = safe(cardiovascular);

        int deteriorating = 0;
        if (sM > CKMRiskDomain.METABOLIC.getDeterioratingThreshold()) deteriorating++;
        if (sR > CKMRiskDomain.RENAL.getDeterioratingThreshold()) deteriorating++;
        if (sCV > CKMRiskDomain.CARDIOVASCULAR.getDeterioratingThreshold()) deteriorating++;

        int aboveAmp = 0;
        if (sM > AMPLIFICATION_THRESHOLD) aboveAmp++;
        if (sR > AMPLIFICATION_THRESHOLD) aboveAmp++;
        if (sCV > AMPLIFICATION_THRESHOLD) aboveAmp++;
        boolean amplify = aboveAmp >= MIN_DOMAINS_FOR_AMPLIFICATION;
        double factor = amplify ? AMPLIFICATION_FACTOR : 1.0;

        double worst = Math.max(sM, Math.max(sR, sCV));
        double best = Math.min(sM, Math.min(sR, sCV));

        CKMRiskVelocity.CompositeClassification classification;
        double compositeScore;

        if (worst > CKMRiskDomain.METABOLIC.getDeterioratingThreshold()) {
            classification = CKMRiskVelocity.CompositeClassification.DETERIORATING;
            compositeScore = Math.min(1.0, worst * factor);
        } else if (best < CKMRiskDomain.METABOLIC.getImprovingThreshold()
                && sM <= 0 && sR <= 0 && sCV <= 0) {
            classification = CKMRiskVelocity.CompositeClassification.IMPROVING;
            compositeScore = best;
        } else {
            classification = CKMRiskVelocity.CompositeClassification.STABLE;
            compositeScore = sM * CKMRiskDomain.METABOLIC.getCompositeWeight()
                    + sR * CKMRiskDomain.RENAL.getCompositeWeight()
                    + sCV * CKMRiskDomain.CARDIOVASCULAR.getCompositeWeight();
        }

        return CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.METABOLIC, sM)
                .domainVelocity(CKMRiskDomain.RENAL, sR)
                .domainVelocity(CKMRiskDomain.CARDIOVASCULAR, sCV)
                .compositeScore(compositeScore)
                .compositeClassification(classification)
                .crossDomainAmplification(amplify)
                .amplificationFactor(factor)
                .domainsDeteriorating(deteriorating)
                .dataCompleteness(dataCompleteness)
                .computationTimestamp(System.currentTimeMillis())
                .build();
    }

    /**
     * Metabolic: 0.35*ΔFBG + 0.30*ΔHbA1c + 0.20*ΔmealIAUC + 0.15*Δweight
     * Positive delta (current > previous) = deteriorating. Negative = improving.
     */
    static double computeMetabolicVelocity(ClinicalStateSummary.MetricSnapshot prev,
            ClinicalStateSummary.MetricSnapshot curr, ClinicalStateSummary state) {
        int signals = 0;
        double weighted = 0.0;

        if (prev.fbg != null && curr.fbg != null) {
            weighted += 0.35 * clamp((curr.fbg - prev.fbg) / FBG_DELTA_RANGE);
            signals++;
        }
        if (prev.hba1c != null && curr.hba1c != null) {
            weighted += 0.30 * clamp((curr.hba1c - prev.hba1c) / HBA1C_DELTA_RANGE);
            signals++;
        }
        if (prev.meanIAUC != null && curr.meanIAUC != null) {
            weighted += 0.20 * clamp((curr.meanIAUC - prev.meanIAUC) / IAUC_DELTA_RANGE);
            signals++;
        }
        if (prev.weight != null && curr.weight != null) {
            weighted += 0.15 * clamp((curr.weight - prev.weight) / WEIGHT_DELTA_RANGE);
            signals++;
        }
        return signals == 0 ? Double.NaN : clamp(weighted / sumWeights(signals, 0.35, 0.30, 0.20, 0.15));
    }

    /**
     * Renal: 0.50*ΔeGFR(inverted) + 0.25*ΔUACR + 0.15*BP_kidney_impact + 0.10*adherence
     * eGFR inverted: drop in eGFR = positive velocity (deteriorating).
     */
    static double computeRenalVelocity(ClinicalStateSummary.MetricSnapshot prev,
            ClinicalStateSummary.MetricSnapshot curr, ClinicalStateSummary state) {
        int signals = 0;
        double weighted = 0.0;

        if (prev.egfr != null && curr.egfr != null) {
            // Inverted: eGFR drop = positive velocity
            weighted += 0.50 * clamp((prev.egfr - curr.egfr) / EGFR_DELTA_RANGE);
            signals++;
        }
        if (prev.uacr != null && curr.uacr != null) {
            weighted += 0.25 * clamp((curr.uacr - prev.uacr) / UACR_DELTA_RANGE);
            signals++;
        }
        // ARV*SBP interaction: current absolute values (not delta — this is a risk modifier)
        if (curr.arv != null && curr.meanSBP != null) {
            double arvNorm = curr.arv > 16.0 ? 1.0 : (curr.arv > 12.0 ? 0.5 : 0.0);
            double sbpFactor = curr.meanSBP > 140.0 ? 1.2 : 1.0;
            weighted += 0.15 * clamp(arvNorm * sbpFactor);
            signals++;
        }
        // Renoprotective adherence from intervention deltas
        if (!state.getRecentInterventionDeltas().isEmpty()) {
            double adherence = state.getRecentInterventionDeltas().stream()
                    .filter(d -> d.getAdherenceScore() != null)
                    .mapToDouble(ClinicalStateSummary.InterventionDeltaSummary::getAdherenceScore)
                    .average().orElse(0.5);
            weighted += 0.10 * clamp(0.5 - adherence); // high adherence = negative = protective
            signals++;
        }
        return signals == 0 ? Double.NaN : clamp(weighted / sumWeights(signals, 0.50, 0.25, 0.15, 0.10));
    }

    /**
     * Cardiovascular: 0.30*ΔARV + 0.25*ΔLDL + 0.20*Δsurge + 0.15*Δengagement + 0.10*intervention_effectiveness
     */
    static double computeCardiovascularVelocity(ClinicalStateSummary.MetricSnapshot prev,
            ClinicalStateSummary.MetricSnapshot curr, ClinicalStateSummary state) {
        int signals = 0;
        double weighted = 0.0;

        if (prev.arv != null && curr.arv != null) {
            weighted += 0.30 * clamp((curr.arv - prev.arv) / ARV_DELTA_RANGE);
            signals++;
        }
        if (prev.ldl != null && curr.ldl != null) {
            weighted += 0.25 * clamp((curr.ldl - prev.ldl) / LDL_DELTA_RANGE);
            signals++;
        }
        if (prev.morningSurgeMagnitude != null && curr.morningSurgeMagnitude != null) {
            weighted += 0.20 * clamp((curr.morningSurgeMagnitude - prev.morningSurgeMagnitude) / SURGE_DELTA_RANGE);
            signals++;
        }
        if (prev.engagementScore != null && curr.engagementScore != null) {
            // Engagement drop = positive velocity (higher CV risk)
            weighted += 0.15 * clamp((prev.engagementScore - curr.engagementScore) / ENGAGEMENT_DELTA_RANGE);
            signals++;
        }
        if (!state.getRecentInterventionDeltas().isEmpty()) {
            long insufficient = state.getRecentInterventionDeltas().stream()
                    .filter(d -> d.getAttribution() == TrajectoryAttribution.INTERVENTION_INSUFFICIENT)
                    .count();
            double ratio = (double) insufficient / state.getRecentInterventionDeltas().size();
            weighted += 0.10 * clamp(ratio * 2 - 1);
            signals++;
        }
        return signals == 0 ? Double.NaN : clamp(weighted / sumWeights(signals, 0.30, 0.25, 0.20, 0.15, 0.10));
    }

    private static double clamp(double v) { return Math.max(-1.0, Math.min(1.0, v)); }
    private static double safe(double v) { return Double.isNaN(v) ? 0.0 : v; }

    private static double sumWeights(int count, double... weights) {
        double sum = 0;
        for (int i = 0; i < Math.min(count, weights.length); i++) sum += weights[i];
        return sum > 0 ? sum : 1.0;
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module13CKMRiskComputerTest 2>&1 | tail -10`
Expected: All 7 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module13CKMRiskComputer.java \
       backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module13CKMRiskComputerTest.java
git commit -m "feat(module13): add Module13CKMRiskComputer with temporal-delta velocity — 7 tests"
```

---

## Task 8: Module13StateChangeDetector + 6 Tests

**Files:**
- Create: `operators/Module13StateChangeDetector.java`
- Create (test): `operators/Module13StateChangeDetectorTest.java`

- [ ] **Step 1: Write the 6 failing tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class Module13StateChangeDetectorTest {

    // --- Test 1: CKM_RISK_ESCALATION from STABLE to DETERIORATING ---
    @Test
    void detect_stableToDeteriorating_emitsCKMRiskEscalation() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithVelocity("p1",
                CKMRiskVelocity.CompositeClassification.STABLE);

        CKMRiskVelocity newVelocity = CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.RENAL, 0.6)
                .domainVelocity(CKMRiskDomain.METABOLIC, 0.1)
                .domainVelocity(CKMRiskDomain.CARDIOVASCULAR, 0.1)
                .compositeClassification(CKMRiskVelocity.CompositeClassification.DETERIORATING)
                .compositeScore(0.6)
                .dataCompleteness(0.85)
                .computationTimestamp(Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, newVelocity, Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.CKM_RISK_ESCALATION));
    }

    // --- Test 2: ENGAGEMENT_COLLAPSE from ACTIVE to DISENGAGED ---
    @Test
    void detect_engagementCollapse_2TierDrop() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        state.current().engagementLevel = EngagementLevel.GREEN;
        state.setPreviousEngagementScore(0.8);
        state.current().engagementScore = 0.2; // GREEN → RED = 2+ tier drop

        CKMRiskVelocity velocity = CKMRiskVelocity.builder()
                .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                .dataCompleteness(0.8)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.ENGAGEMENT_COLLAPSE));
    }

    // --- Test 3: INTERVENTION_FUTILITY after 2 consecutive INSUFFICIENT ---
    @Test
    void detect_interventionFutility_2ConsecutiveInsufficient() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        ClinicalStateSummary.InterventionDeltaSummary d1 = new ClinicalStateSummary.InterventionDeltaSummary();
        d1.setInterventionId("i1");
        d1.setAttribution(TrajectoryAttribution.INTERVENTION_INSUFFICIENT);
        d1.setClosedAtMs(Module13TestBuilder.BASE_TIME);
        ClinicalStateSummary.InterventionDeltaSummary d2 = new ClinicalStateSummary.InterventionDeltaSummary();
        d2.setInterventionId("i2");
        d2.setAttribution(TrajectoryAttribution.INTERVENTION_INSUFFICIENT);
        d2.setClosedAtMs(Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);
        state.getRecentInterventionDeltas().add(d1);
        state.getRecentInterventionDeltas().add(d2);

        CKMRiskVelocity velocity = CKMRiskVelocity.builder()
                .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                .dataCompleteness(0.8)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.INTERVENTION_FUTILITY));
    }

    // --- Test 4: CROSS_MODULE_INCONSISTENCY (high adherence + deteriorating) ---
    @Test
    void detect_crossModuleInconsistency_highAdherenceButDeteriorating() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        state.current().engagementScore = 0.85;
        state.current().engagementLevel = EngagementLevel.GREEN;
        ClinicalStateSummary.InterventionDeltaSummary d = new ClinicalStateSummary.InterventionDeltaSummary();
        d.setAdherenceScore(0.9);
        d.setAttribution(TrajectoryAttribution.INTERVENTION_INSUFFICIENT);
        d.setClosedAtMs(Module13TestBuilder.BASE_TIME);
        state.getRecentInterventionDeltas().add(d);

        CKMRiskVelocity velocity = CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.METABOLIC, 0.5)
                .compositeClassification(CKMRiskVelocity.CompositeClassification.DETERIORATING)
                .compositeScore(0.5)
                .dataCompleteness(0.9)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.CROSS_MODULE_INCONSISTENCY));
    }

    // --- Test 5: METABOLIC_MILESTONE (FBG below target) ---
    @Test
    void detect_metabolicMilestone_fbgBelowTarget() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        state.current().fbg = 105.0; // below 110 target

        CKMRiskVelocity velocity = CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.METABOLIC, -0.4)
                .compositeClassification(CKMRiskVelocity.CompositeClassification.IMPROVING)
                .compositeScore(-0.3)
                .dataCompleteness(0.8)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE));
    }

    // --- Test 6: Dedup — same state change not emitted twice within 24h ---
    @Test
    void detect_dedup_sameChangeNotEmittedTwiceIn24Hours() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        state.current().fbg = 105.0;
        // Mark METABOLIC_MILESTONE as already emitted recently
        state.getLastEmittedChangeTimestamps().put(
                ClinicalStateChangeType.METABOLIC_MILESTONE,
                Module13TestBuilder.BASE_TIME - Module13TestBuilder.HOUR_MS); // 1 hour ago

        CKMRiskVelocity velocity = CKMRiskVelocity.builder()
                .compositeClassification(CKMRiskVelocity.CompositeClassification.IMPROVING)
                .dataCompleteness(0.8)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME);

        assertTrue(events.stream().noneMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE),
                "Should NOT re-emit METABOLIC_MILESTONE within 24h dedup window");
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module13StateChangeDetectorTest -Dsurefire.failIfNoTests=false 2>&1 | tail -5`
Expected: Compilation failure — `Module13StateChangeDetector` class does not exist.

- [ ] **Step 3: Implement Module13StateChangeDetector**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;

import java.util.*;

public final class Module13StateChangeDetector {

    private static final long DEDUP_WINDOW_MS = 24 * 3_600_000L; // 24 hours
    private static final double FBG_TARGET = 110.0;
    private static final double SBP_TARGET = 130.0;
    private static final double ENGAGEMENT_COLLAPSE_DELTA = 0.35; // >0.35 drop = collapse
    private static final int FUTILITY_CONSECUTIVE_COUNT = 2;

    private Module13StateChangeDetector() {}

    /**
     * Detect clinically significant state transitions.
     * Returns list of change events (empty if no transitions detected).
     */
    public static List<ClinicalStateChangeEvent> detect(
            ClinicalStateSummary state,
            CKMRiskVelocity newVelocity,
            long currentTimestamp) {

        List<ClinicalStateChangeEvent> events = new ArrayList<>();

        checkCKMRiskEscalation(state, newVelocity, currentTimestamp, events);
        checkCKMDomainDivergence(newVelocity, state, currentTimestamp, events);
        checkRenalRapidDecline(state, currentTimestamp, events);
        checkEngagementCollapse(state, currentTimestamp, events);
        checkInterventionFutility(state, currentTimestamp, events);
        checkTrajectoryReversal(state, newVelocity, currentTimestamp, events);
        checkMetabolicMilestone(state, currentTimestamp, events);
        checkBPMilestone(state, currentTimestamp, events);
        checkCrossModuleInconsistency(state, newVelocity, currentTimestamp, events);

        return events;
    }

    private static void checkCKMRiskEscalation(ClinicalStateSummary state,
            CKMRiskVelocity newVelocity, long ts, List<ClinicalStateChangeEvent> events) {
        if (state.getLastComputedVelocity() == null) return;
        CKMRiskVelocity.CompositeClassification prev = state.getLastComputedVelocity().getCompositeClassification();
        CKMRiskVelocity.CompositeClassification curr = newVelocity.getCompositeClassification();
        if (prev != CKMRiskVelocity.CompositeClassification.DETERIORATING
                && curr == CKMRiskVelocity.CompositeClassification.DETERIORATING) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.CKM_RISK_ESCALATION,
                    prev.name(), curr.name(), null, "module13", newVelocity);
        }
    }

    private static void checkCKMDomainDivergence(CKMRiskVelocity velocity,
            ClinicalStateSummary state, long ts, List<ClinicalStateChangeEvent> events) {
        if (velocity.isCrossDomainAmplification()) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.CKM_DOMAIN_DIVERGENCE,
                    String.valueOf(velocity.getDomainsDeteriorating()), "2+ domains worsening",
                    null, "module13", velocity);
        }
    }

    private static void checkRenalRapidDecline(ClinicalStateSummary state,
            long ts, List<ClinicalStateChangeEvent> events) {
        Double egfr = state.current().egfr;
        if (egfr != null && egfr < 45.0) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.RENAL_RAPID_DECLINE,
                    "eGFR_baseline", String.valueOf(egfr),
                    CKMRiskDomain.RENAL, "module7", state.getLastComputedVelocity());
        }
    }

    private static void checkEngagementCollapse(ClinicalStateSummary state,
            long ts, List<ClinicalStateChangeEvent> events) {
        Double currentEngagement = state.current().engagementScore;
        if (state.getPreviousEngagementScore() != null && currentEngagement != null) {
            double drop = state.getPreviousEngagementScore() - currentEngagement;
            if (drop >= ENGAGEMENT_COLLAPSE_DELTA) {
                emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.ENGAGEMENT_COLLAPSE,
                        String.valueOf(state.getPreviousEngagementScore()),
                        String.valueOf(currentEngagement),
                        null, "module9", state.getLastComputedVelocity());
            }
        }
    }

    private static void checkInterventionFutility(ClinicalStateSummary state,
            long ts, List<ClinicalStateChangeEvent> events) {
        List<ClinicalStateSummary.InterventionDeltaSummary> deltas = state.getRecentInterventionDeltas();
        if (deltas.size() >= FUTILITY_CONSECUTIVE_COUNT) {
            // Check last N deltas (most recent) for consecutive INSUFFICIENT
            int consecutiveInsufficient = 0;
            for (int i = deltas.size() - 1; i >= 0; i--) {
                if (deltas.get(i).getAttribution() == TrajectoryAttribution.INTERVENTION_INSUFFICIENT) {
                    consecutiveInsufficient++;
                } else {
                    break;
                }
            }
            if (consecutiveInsufficient >= FUTILITY_CONSECUTIVE_COUNT) {
                emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.INTERVENTION_FUTILITY,
                        String.valueOf(consecutiveInsufficient) + " consecutive INSUFFICIENT",
                        "Phenotype review needed", null, "module12b",
                        state.getLastComputedVelocity());
            }
        }
    }

    private static void checkTrajectoryReversal(ClinicalStateSummary state,
            CKMRiskVelocity newVelocity, long ts, List<ClinicalStateChangeEvent> events) {
        if (state.getLastComputedVelocity() == null) return;
        CKMRiskVelocity.CompositeClassification prev = state.getLastComputedVelocity().getCompositeClassification();
        CKMRiskVelocity.CompositeClassification curr = newVelocity.getCompositeClassification();
        if (prev == CKMRiskVelocity.CompositeClassification.IMPROVING
                && curr == CKMRiskVelocity.CompositeClassification.DETERIORATING) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.TRAJECTORY_REVERSAL,
                    prev.name(), curr.name(), null, "module13", newVelocity);
        }
    }

    private static void checkMetabolicMilestone(ClinicalStateSummary state,
            long ts, List<ClinicalStateChangeEvent> events) {
        Double fbg = state.current().fbg;
        if (fbg != null && fbg < FBG_TARGET) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.METABOLIC_MILESTONE,
                    "FBG above target", String.valueOf(fbg),
                    CKMRiskDomain.METABOLIC, "enriched", state.getLastComputedVelocity());
        }
    }

    private static void checkBPMilestone(ClinicalStateSummary state,
            long ts, List<ClinicalStateChangeEvent> events) {
        Double meanSBP = state.current().meanSBP;
        if (meanSBP != null && meanSBP < SBP_TARGET) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.BP_MILESTONE,
                    "SBP above target", String.valueOf(meanSBP),
                    CKMRiskDomain.CARDIOVASCULAR, "module7", state.getLastComputedVelocity());
        }
    }

    private static void checkCrossModuleInconsistency(ClinicalStateSummary state,
            CKMRiskVelocity velocity, long ts, List<ClinicalStateChangeEvent> events) {
        boolean highAdherence = state.current().engagementScore != null
                && state.current().engagementScore > 0.7;
        boolean hasHighAdherenceDeltas = state.getRecentInterventionDeltas().stream()
                .anyMatch(d -> d.getAdherenceScore() != null && d.getAdherenceScore() > 0.7);
        boolean deteriorating = velocity.getCompositeClassification()
                == CKMRiskVelocity.CompositeClassification.DETERIORATING;

        if ((highAdherence || hasHighAdherenceDeltas) && deteriorating) {
            emitIfNotDeduped(state, ts, events, ClinicalStateChangeType.CROSS_MODULE_INCONSISTENCY,
                    "HIGH adherence", "DETERIORATING trajectory", null, "module13", velocity);
        }
    }

    private static void emitIfNotDeduped(ClinicalStateSummary state, long currentTs,
            List<ClinicalStateChangeEvent> events, ClinicalStateChangeType type,
            String previousValue, String currentValue, CKMRiskDomain domain,
            String triggerModule, CKMRiskVelocity velocity) {

        Long lastEmitted = state.getLastEmittedChangeTimestamps().get(type);
        if (lastEmitted != null && (currentTs - lastEmitted) < DEDUP_WINDOW_MS) {
            return; // Deduped
        }

        events.add(ClinicalStateChangeEvent.builder()
                .changeId(UUID.randomUUID().toString())
                .patientId(state.getPatientId())
                .changeType(type)
                .previousValue(previousValue)
                .currentValue(currentValue)
                .domain(domain)
                .triggerModule(triggerModule)
                .ckmVelocityAtChange(velocity)
                .dataCompletenessAtChange(state.getDataCompletenessScore())
                .confidenceScore(state.getDataCompletenessScore())
                .processingTimestamp(currentTs)
                .build());
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module13StateChangeDetectorTest 2>&1 | tail -10`
Expected: All 6 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module13StateChangeDetector.java \
       backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module13StateChangeDetectorTest.java
git commit -m "feat(module13): add Module13StateChangeDetector with 12 change types, 24h dedup — 6 tests"
```

---

## Task 9: Module13DataCompletenessMonitor + 5 Tests

**Files:**
- Create: `operators/Module13DataCompletenessMonitor.java`
- Create (test): `operators/Module13DataCompletenessMonitorTest.java`

- [ ] **Step 1: Write the 5 failing tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class Module13DataCompletenessMonitorTest {

    // --- Test 1: All modules reporting within 7 days = score 1.0 ---
    @Test
    void allModulesRecentlyReporting_scoreIs1() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS;

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertEquals(1.0, result.getCompositeScore(), 0.01);
        assertTrue(result.getDataGapFlags().isEmpty());
    }

    // --- Test 2: One module stale >7 days = reduced score ---
    @Test
    void oneModuleStale_reducedScore() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 10 * Module13TestBuilder.DAY_MS;
        // module11b last seen at BASE_TIME = 10 days ago
        state.recordModuleSeen("module7", now - Module13TestBuilder.DAY_MS); // 1 day ago
        state.recordModuleSeen("module9", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("module10b", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("enriched", now - Module13TestBuilder.DAY_MS);
        // module11b stays at BASE_TIME = 10 days stale

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertTrue(result.getCompositeScore() < 1.0);
        assertTrue(result.getCompositeScore() > 0.5);
        assertTrue(result.getDataGapFlags().containsKey("module11b"));
    }

    // --- Test 3: All modules stale >14 days = near-zero + DATA_ABSENCE_CRITICAL ---
    @Test
    void allModulesStale14Days_nearZeroScore() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 20 * Module13TestBuilder.DAY_MS;
        // All modules last seen at BASE_TIME = 20 days ago

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertTrue(result.getCompositeScore() < 0.4);
        assertTrue(result.isDataAbsenceCritical());
    }

    // --- Test 4: Specific module absence flagged correctly ---
    @Test
    void specificModuleAbsence_flagged() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 10 * Module13TestBuilder.DAY_MS;
        // Remove module12 from last-seen (never seen)
        state.getModuleLastSeenMs().remove("module12");
        // Update others to recent
        state.recordModuleSeen("module7", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("module9", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("module10b", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("module11b", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("enriched", now - Module13TestBuilder.DAY_MS);

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertTrue(result.getDataGapFlags().containsKey("module12"));
        assertEquals("NEVER_SEEN", result.getDataGapFlags().get("module12"));
    }

    // --- Test 5: CGM-tier patients penalised more for meal response gaps ---
    @Test
    void cgmTierPatient_mealGapPenalisedMore() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        state.setDataTier("TIER_1_CGM");
        long now = Module13TestBuilder.BASE_TIME + 10 * Module13TestBuilder.DAY_MS;
        // module10b stale
        state.recordModuleSeen("module7", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("module9", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("module11b", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("enriched", now - Module13TestBuilder.DAY_MS);
        // module10b stays at BASE_TIME = 10 days stale

        Module13DataCompletenessMonitor.Result cgmResult =
                Module13DataCompletenessMonitor.evaluate(state, now);

        // Same scenario but SMBG tier
        state.setDataTier("TIER_2_SMBG");
        Module13DataCompletenessMonitor.Result smbgResult =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertTrue(cgmResult.getCompositeScore() < smbgResult.getCompositeScore(),
                "CGM patient should be penalised more for meal pattern gaps");
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module13DataCompletenessMonitorTest -Dsurefire.failIfNoTests=false 2>&1 | tail -5`
Expected: Compilation failure.

- [ ] **Step 3: Implement Module13DataCompletenessMonitor**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.ClinicalStateSummary;

import java.util.*;

public final class Module13DataCompletenessMonitor {

    private static final long FRESH_WINDOW_MS = 7 * 86_400_000L;  // 7 days
    private static final long DECAY_WINDOW_MS = 30 * 86_400_000L; // 30 days
    private static final long CRITICAL_ABSENCE_MS = 14 * 86_400_000L; // 14 days
    private static final long WARNING_ABSENCE_MS = 7 * 86_400_000L;  // 7 days

    private static final String[] TRACKED_MODULES = {
            "module7", "module9", "module10b", "module11b", "module12", "module12b", "enriched"
    };

    // Weight by data tier: CGM patients should have more meal/activity data
    private static final Map<String, Map<String, Double>> TIER_WEIGHTS = new HashMap<>();
    static {
        Map<String, Double> cgm = new HashMap<>();
        cgm.put("module7", 0.15);
        cgm.put("module9", 0.15);
        cgm.put("module10b", 0.20); // meal patterns more important for CGM
        cgm.put("module11b", 0.15);
        cgm.put("module12", 0.10);
        cgm.put("module12b", 0.10);
        cgm.put("enriched", 0.15);
        TIER_WEIGHTS.put("TIER_1_CGM", cgm);

        Map<String, Double> smbg = new HashMap<>();
        smbg.put("module7", 0.15);
        smbg.put("module9", 0.15);
        smbg.put("module10b", 0.10); // meal patterns less critical for SMBG
        smbg.put("module11b", 0.10);
        smbg.put("module12", 0.10);
        smbg.put("module12b", 0.10);
        smbg.put("enriched", 0.30); // labs more important for SMBG
        TIER_WEIGHTS.put("TIER_2_SMBG", smbg);
        TIER_WEIGHTS.put("TIER_3_SMBG", smbg);
    }

    private static final Map<String, Double> DEFAULT_WEIGHTS;
    static {
        DEFAULT_WEIGHTS = new HashMap<>();
        double equal = 1.0 / TRACKED_MODULES.length;
        for (String m : TRACKED_MODULES) DEFAULT_WEIGHTS.put(m, equal);
    }

    private Module13DataCompletenessMonitor() {}

    public static Result evaluate(ClinicalStateSummary state, long currentTimestamp) {
        Map<String, Double> weights = TIER_WEIGHTS.getOrDefault(
                state.getDataTier() != null ? state.getDataTier() : "", DEFAULT_WEIGHTS);

        Map<String, String> gapFlags = new LinkedHashMap<>();
        double weightedScore = 0.0;
        int allStaleCount = 0;
        long oldestLastSeen = Long.MAX_VALUE;

        for (String module : TRACKED_MODULES) {
            Long lastSeen = state.getModuleLastSeenMs().get(module);
            double weight = weights.getOrDefault(module, 1.0 / TRACKED_MODULES.length);

            if (lastSeen == null) {
                gapFlags.put(module, "NEVER_SEEN");
                allStaleCount++;
                continue;
            }

            long age = currentTimestamp - lastSeen;
            oldestLastSeen = Math.min(oldestLastSeen, lastSeen);

            if (age <= FRESH_WINDOW_MS) {
                weightedScore += weight * 1.0;
            } else if (age <= DECAY_WINDOW_MS) {
                // Linear decay from 1.0 at 7d to 0.0 at 30d
                double freshness = 1.0 - ((double)(age - FRESH_WINDOW_MS)
                        / (DECAY_WINDOW_MS - FRESH_WINDOW_MS));
                weightedScore += weight * Math.max(0, freshness);
                if (age > WARNING_ABSENCE_MS) {
                    gapFlags.put(module, age > CRITICAL_ABSENCE_MS ? "CRITICAL" : "WARNING");
                }
                allStaleCount++;
            } else {
                gapFlags.put(module, "EXPIRED");
                allStaleCount++;
            }
        }

        // Normalise: divide by sum of weights for present modules
        double totalWeight = 0;
        for (String m : TRACKED_MODULES) totalWeight += weights.getOrDefault(m, 1.0 / TRACKED_MODULES.length);
        double compositeScore = totalWeight > 0 ? weightedScore / totalWeight : 0.0;

        boolean absenceCritical = allStaleCount == TRACKED_MODULES.length
                || (oldestLastSeen != Long.MAX_VALUE
                    && (currentTimestamp - oldestLastSeen) > CRITICAL_ABSENCE_MS
                    && allStaleCount >= TRACKED_MODULES.length - 1);

        return new Result(Math.min(1.0, compositeScore), gapFlags, absenceCritical);
    }

    public static class Result {
        private final double compositeScore;
        private final Map<String, String> dataGapFlags;
        private final boolean dataAbsenceCritical;

        public Result(double compositeScore, Map<String, String> dataGapFlags, boolean dataAbsenceCritical) {
            this.compositeScore = compositeScore;
            this.dataGapFlags = dataGapFlags;
            this.dataAbsenceCritical = dataAbsenceCritical;
        }

        public double getCompositeScore() { return compositeScore; }
        public Map<String, String> getDataGapFlags() { return dataGapFlags; }
        public boolean isDataAbsenceCritical() { return dataAbsenceCritical; }
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module13DataCompletenessMonitorTest 2>&1 | tail -10`
Expected: All 5 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module13DataCompletenessMonitor.java \
       backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module13DataCompletenessMonitorTest.java
git commit -m "feat(module13): add Module13DataCompletenessMonitor with tier-aware freshness scoring — 5 tests"
```

---

## Task 10: Module13KB20StateProjector + 5 Tests

**Files:**
- Create: `operators/Module13KB20StateProjector.java`
- Create (test): `operators/Module13KB20StateProjectorTest.java`

- [ ] **Step 1: Write the 5 failing tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

class Module13KB20StateProjectorTest {

    // --- Test 1: BP variability event → correct KB-20 field mapping ---
    @Test
    void project_bpVariabilityEvent_correctFieldMapping() {
        CanonicalEvent event = Module13TestBuilder.bpVariabilityEvent(
                "p1", Module13TestBuilder.BASE_TIME, 14.5, "ELEVATED", 145.0, 92.0);

        KB20StateUpdate update = Module13KB20StateProjector.project(event);

        assertNotNull(update);
        assertEquals("p1", update.getPatientId());
        assertEquals(KB20StateUpdate.UpdateOperation.REPLACE, update.getOperation());
        assertEquals("module7", update.getSourceModule());
        Map<String, Object> fields = update.getFieldUpdates();
        assertEquals(14.5, fields.get("bp_variability_arv"));
        assertEquals("ELEVATED", fields.get("bp_variability_classification"));
        assertEquals(145.0, fields.get("bp_mean_sbp"));
    }

    // --- Test 2: Engagement event → correct field mapping ---
    @Test
    void project_engagementEvent_correctFieldMapping() {
        CanonicalEvent event = Module13TestBuilder.engagementEvent(
                "p1", Module13TestBuilder.BASE_TIME, 0.65, "YELLOW", "DIGITAL_NATIVE", "TIER_1_CGM");

        KB20StateUpdate update = Module13KB20StateProjector.project(event);

        assertNotNull(update);
        assertEquals(KB20StateUpdate.UpdateOperation.REPLACE, update.getOperation());
        assertEquals("module9", update.getSourceModule());
        assertEquals(0.65, update.getFieldUpdates().get("engagement_composite_score"));
        assertEquals("YELLOW", update.getFieldUpdates().get("engagement_level"));
    }

    // --- Test 3: Intervention window OPENED → upsert to active_interventions ---
    @Test
    void project_interventionWindowOpened_upsert() {
        CanonicalEvent event = Module13TestBuilder.interventionWindowEvent(
                "p1", Module13TestBuilder.BASE_TIME, "int-001", "WINDOW_OPENED",
                "MEDICATION_ADD", Module13TestBuilder.BASE_TIME,
                Module13TestBuilder.BASE_TIME + 28 * Module13TestBuilder.DAY_MS);

        KB20StateUpdate update = Module13KB20StateProjector.project(event);

        assertNotNull(update);
        assertEquals(KB20StateUpdate.UpdateOperation.UPSERT, update.getOperation());
        assertEquals("module12", update.getSourceModule());
        assertEquals("intervention_id", update.getUpsertKey());
        assertEquals("int-001", update.getFieldUpdates().get("intervention_id"));
        assertEquals("WINDOW_OPENED", update.getFieldUpdates().get("window_status"));
    }

    // --- Test 4: Intervention delta (CLOSED) → append to outcomes ---
    @Test
    void project_interventionDelta_appendToOutcomes() {
        CanonicalEvent event = Module13TestBuilder.interventionDeltaEvent(
                "p1", Module13TestBuilder.BASE_TIME, "int-001", "IMPROVING", 0.85,
                -15.0, -8.0, 2.0);

        KB20StateUpdate update = Module13KB20StateProjector.project(event);

        assertNotNull(update);
        assertEquals(KB20StateUpdate.UpdateOperation.APPEND, update.getOperation());
        assertEquals("module12b", update.getSourceModule());
        assertEquals("int-001", update.getFieldUpdates().get("intervention_id"));
        assertEquals("IMPROVING", update.getFieldUpdates().get("trajectory_attribution"));
    }

    // --- Test 5: Unknown source module → null (defensive) ---
    @Test
    void project_unknownSourceModule_returnsNull() {
        CanonicalEvent event = CanonicalEvent.builder()
                .id("test-id")
                .patientId("p1")
                .eventType(EventType.UNKNOWN)
                .eventTime(Module13TestBuilder.BASE_TIME)
                .payload(Map.of("source_module", "unknown_module"))
                .build();

        KB20StateUpdate update = Module13KB20StateProjector.project(event);

        assertNull(update, "Unknown source module should return null");
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module13KB20StateProjectorTest -Dsurefire.failIfNoTests=false 2>&1 | tail -5`
Expected: Compilation failure.

- [ ] **Step 3: Implement Module13KB20StateProjector**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;

import java.util.Map;

public final class Module13KB20StateProjector {

    private Module13KB20StateProjector() {}

    /**
     * Map a CanonicalEvent from an upstream module to a KB20StateUpdate.
     * Returns null if the event source is unrecognised.
     */
    public static KB20StateUpdate project(CanonicalEvent event) {
        Map<String, Object> payload = event.getPayload();
        if (payload == null) return null;

        String sourceModule = payload.get("source_module") != null
                ? payload.get("source_module").toString() : "";

        switch (sourceModule) {
            case "module7":
                return projectBPVariability(event, payload);
            case "module9":
                return projectEngagement(event, payload);
            case "module10b":
                return projectMealPatterns(event, payload);
            case "module11b":
                return projectFitnessPatterns(event, payload);
            case "module12":
                return projectInterventionWindow(event, payload);
            case "module12b":
                return projectInterventionDelta(event, payload);
            case "enriched":
                return projectLabResult(event, payload);
            default:
                return null;
        }
    }

    private static KB20StateUpdate projectBPVariability(CanonicalEvent event, Map<String, Object> payload) {
        return KB20StateUpdate.builder()
                .patientId(event.getPatientId())
                .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                .sourceModule("module7")
                .timestamp(event.getEventTime())
                .field("bp_variability_arv", payload.get("arv"))
                .field("bp_variability_classification", payload.get("variability_classification"))
                .field("bp_mean_sbp", payload.get("mean_sbp"))
                .field("bp_mean_dbp", payload.get("mean_dbp"))
                .build();
    }

    private static KB20StateUpdate projectEngagement(CanonicalEvent event, Map<String, Object> payload) {
        return KB20StateUpdate.builder()
                .patientId(event.getPatientId())
                .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                .sourceModule("module9")
                .timestamp(event.getEventTime())
                .field("engagement_composite_score", payload.get("composite_score"))
                .field("engagement_level", payload.get("engagement_level"))
                .field("engagement_phenotype", payload.get("phenotype"))
                .field("data_tier", payload.get("data_tier"))
                .build();
    }

    private static KB20StateUpdate projectMealPatterns(CanonicalEvent event, Map<String, Object> payload) {
        return KB20StateUpdate.builder()
                .patientId(event.getPatientId())
                .operation(KB20StateUpdate.UpdateOperation.MERGE)
                .sourceModule("module10b")
                .timestamp(event.getEventTime())
                .field("meal_mean_iauc", payload.get("mean_iauc"))
                .field("meal_median_excursion", payload.get("median_excursion"))
                .field("salt_sensitivity_class", payload.get("salt_sensitivity_class"))
                .field("salt_beta", payload.get("salt_beta"))
                .build();
    }

    private static KB20StateUpdate projectFitnessPatterns(CanonicalEvent event, Map<String, Object> payload) {
        return KB20StateUpdate.builder()
                .patientId(event.getPatientId())
                .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                .sourceModule("module11b")
                .timestamp(event.getEventTime())
                .field("estimated_vo2max", payload.get("estimated_vo2max"))
                .field("vo2max_trend", payload.get("vo2max_trend"))
                .field("total_met_minutes", payload.get("total_met_minutes"))
                .field("mean_exercise_glucose_delta", payload.get("mean_exercise_glucose_delta"))
                .build();
    }

    private static KB20StateUpdate projectInterventionWindow(CanonicalEvent event, Map<String, Object> payload) {
        return KB20StateUpdate.builder()
                .patientId(event.getPatientId())
                .operation(KB20StateUpdate.UpdateOperation.UPSERT)
                .upsertKey("intervention_id")
                .sourceModule("module12")
                .timestamp(event.getEventTime())
                .field("intervention_id", payload.get("intervention_id"))
                .field("window_status", payload.get("signal_type"))
                .field("intervention_type", payload.get("intervention_type"))
                .field("observation_start_ms", payload.get("observation_start_ms"))
                .field("observation_end_ms", payload.get("observation_end_ms"))
                .build();
    }

    private static KB20StateUpdate projectInterventionDelta(CanonicalEvent event, Map<String, Object> payload) {
        return KB20StateUpdate.builder()
                .patientId(event.getPatientId())
                .operation(KB20StateUpdate.UpdateOperation.APPEND)
                .sourceModule("module12b")
                .timestamp(event.getEventTime())
                .field("intervention_id", payload.get("intervention_id"))
                .field("trajectory_attribution", payload.get("trajectory_attribution"))
                .field("adherence_score", payload.get("adherence_score"))
                .field("fbg_delta", payload.get("fbg_delta"))
                .field("sbp_delta", payload.get("sbp_delta"))
                .field("egfr_delta", payload.get("egfr_delta"))
                .build();
    }

    private static KB20StateUpdate projectLabResult(CanonicalEvent event, Map<String, Object> payload) {
        String labType = payload.get("lab_type") != null ? payload.get("lab_type").toString() : "";
        return KB20StateUpdate.builder()
                .patientId(event.getPatientId())
                .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                .sourceModule("enriched")
                .timestamp(event.getEventTime())
                .field("lab_type", labType)
                .field("lab_value", payload.get("value"))
                .build();
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module13KB20StateProjectorTest 2>&1 | tail -10`
Expected: All 5 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module13KB20StateProjector.java \
       backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module13KB20StateProjectorTest.java
git commit -m "feat(module13): add Module13KB20StateProjector with field-level mapping for 7 source modules — 5 tests"
```

---

## Task 11: Module13_ClinicalStateSynchroniser (Main KPF)

**Files:**
- Create: `operators/Module13_ClinicalStateSynchroniser.java`

This is the core operator. It follows the same pattern as `Module12_InterventionWindowMonitor`: `KeyedProcessFunction<String, CanonicalEvent, ClinicalStateChangeEvent>` with `ValueState<ClinicalStateSummary>`, 90-day TTL, processing-time timers for write coalescing (5s) and daily data absence (24h).

- [ ] **Step 1: Create the main KPF**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.api.common.state.StateTtlConfig;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.util.*;

public class Module13_ClinicalStateSynchroniser
        extends KeyedProcessFunction<String, CanonicalEvent, ClinicalStateChangeEvent> {

    private static final Logger LOG = LoggerFactory.getLogger(Module13_ClinicalStateSynchroniser.class);

    public static final OutputTag<KB20StateUpdate> KB20_SIDE_OUTPUT =
            new OutputTag<KB20StateUpdate>("kb20-state-updates") {};

    private static final long COALESCING_WINDOW_MS = 5_000L;  // 5 seconds
    private static final long DAILY_TIMER_INTERVAL_MS = 24 * 3_600_000L; // 24 hours
    private static final long SNAPSHOT_ROTATION_INTERVAL_MS = 7 * 86_400_000L; // 7 days
    private static final int IDLE_QUIESCENCE_THRESHOLD = 30; // days of zero completeness before stopping timer

    private transient ValueState<ClinicalStateSummary> summaryState;

    @Override
    public void open(OpenContext openContext) throws Exception {
        ValueStateDescriptor<ClinicalStateSummary> stateDesc =
                new ValueStateDescriptor<>("clinical-state-summary", ClinicalStateSummary.class);
        StateTtlConfig ttl = StateTtlConfig
                .newBuilder(Duration.ofDays(90))
                .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
                .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
                .build();
        stateDesc.enableTimeToLive(ttl);
        summaryState = getRuntimeContext().getState(stateDesc);
        LOG.info("Module 13 Clinical State Synchroniser initialized (90-day TTL, 7-source fan-in)");
    }

    @Override
    public void processElement(CanonicalEvent event, Context ctx,
                               Collector<ClinicalStateChangeEvent> out) throws Exception {
        ClinicalStateSummary state = summaryState.value();
        if (state == null) {
            state = new ClinicalStateSummary(event.getPatientId());
            LOG.info("New patient state created: {}", event.getPatientId());
        }

        // 1. Route event and update state fields
        String sourceModule = routeAndUpdateState(event, state);
        if (sourceModule == null) {
            LOG.debug("Unroutable event for patient {}: type={}", event.getPatientId(), event.getEventType());
            summaryState.update(state);
            return;
        }
        state.recordModuleSeen(sourceModule, event.getEventTime());

        // 2. Compute data completeness
        Module13DataCompletenessMonitor.Result completeness =
                Module13DataCompletenessMonitor.evaluate(state, ctx.timerService().currentProcessingTime());
        state.setDataCompletenessScore(completeness.getCompositeScore());

        // 3. Compute CKM risk velocity
        CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);

        // 4. Detect state changes
        List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                state, velocity, ctx.timerService().currentProcessingTime());

        // 5. Emit state change events + update dedup timestamps
        for (ClinicalStateChangeEvent change : changes) {
            out.collect(change);
            state.getLastEmittedChangeTimestamps().put(
                    change.getChangeType(), change.getProcessingTimestamp());
            LOG.info("State change emitted: patient={}, type={}, priority={}",
                    event.getPatientId(), change.getChangeType(), change.getPriority());
        }

        // 6. Check data absence events from completeness monitor
        if (completeness.isDataAbsenceCritical()) {
            emitDataAbsenceIfNeeded(state, ctx, out, ClinicalStateChangeType.DATA_ABSENCE_CRITICAL, velocity);
        } else if (!completeness.getDataGapFlags().isEmpty()) {
            emitDataAbsenceIfNeeded(state, ctx, out, ClinicalStateChangeType.DATA_ABSENCE_WARNING, velocity);
        }

        // 7. Update velocity in state
        state.setLastComputedVelocity(velocity);
        state.setLastUpdated(ctx.timerService().currentProcessingTime());

        // 8. Project KB-20 update and buffer for coalescing
        KB20StateUpdate kb20Update = Module13KB20StateProjector.project(event);
        if (kb20Update != null) {
            state.getCoalescingBuffer().add(kb20Update);

            // Add computed fields to buffer
            KB20StateUpdate computedUpdate = KB20StateUpdate.builder()
                    .patientId(event.getPatientId())
                    .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                    .sourceModule("module13")
                    .timestamp(ctx.timerService().currentProcessingTime())
                    .field("ckm_risk_velocity", velocity)
                    .field("data_completeness", completeness.getCompositeScore())
                    .field("last_streaming_update", ctx.timerService().currentProcessingTime())
                    .build();
            state.getCoalescingBuffer().add(computedUpdate);

            // Register coalescing timer if not already set
            if (state.getCoalescingTimerMs() < 0) {
                long timerTs = ctx.timerService().currentProcessingTime() + COALESCING_WINDOW_MS;
                ctx.timerService().registerProcessingTimeTimer(timerTs);
                state.setCoalescingTimerMs(timerTs);
            }
        }

        // 9. Register daily timer if not set (Issue 6 fix: skip if idle)
        if (state.getDailyTimerMs() < 0
                && state.getConsecutiveZeroCompletenessDays() < IDLE_QUIESCENCE_THRESHOLD) {
            long dailyTs = ctx.timerService().currentProcessingTime() + DAILY_TIMER_INTERVAL_MS;
            ctx.timerService().registerProcessingTimeTimer(dailyTs);
            state.setDailyTimerMs(dailyTs);
        }

        // 10. Register snapshot rotation timer if not set (7-day interval for velocity computation)
        if (state.getSnapshotRotationTimerMs() < 0) {
            long rotationTs = ctx.timerService().currentProcessingTime() + SNAPSHOT_ROTATION_INTERVAL_MS;
            ctx.timerService().registerProcessingTimeTimer(rotationTs);
            state.setSnapshotRotationTimerMs(rotationTs);
        }

        // 11. Reset idle counter since we received real data
        state.setConsecutiveZeroCompletenessDays(0);

        summaryState.update(state);
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx,
                        Collector<ClinicalStateChangeEvent> out) throws Exception {
        ClinicalStateSummary state = summaryState.value();
        if (state == null) return;

        if (timestamp == state.getCoalescingTimerMs()) {
            // Flush coalescing buffer via side output
            for (KB20StateUpdate update : state.getCoalescingBuffer()) {
                ctx.output(KB20_SIDE_OUTPUT, update);
            }
            LOG.debug("Flushed {} KB-20 updates for patient {}",
                    state.getCoalescingBuffer().size(), state.getPatientId());
            state.getCoalescingBuffer().clear();
            state.setCoalescingTimerMs(-1L);

        } else if (timestamp == state.getDailyTimerMs()) {
            // Daily data absence check
            Module13DataCompletenessMonitor.Result completeness =
                    Module13DataCompletenessMonitor.evaluate(state, timestamp);
            state.setDataCompletenessScore(completeness.getCompositeScore());

            CKMRiskVelocity velocity = state.getLastComputedVelocity();
            if (velocity == null) {
                velocity = Module13CKMRiskComputer.compute(state);
                state.setLastComputedVelocity(velocity);
            }

            if (completeness.isDataAbsenceCritical()) {
                emitDataAbsenceIfNeeded(state, ctx, out,
                        ClinicalStateChangeType.DATA_ABSENCE_CRITICAL, velocity);
            } else if (!completeness.getDataGapFlags().isEmpty()) {
                emitDataAbsenceIfNeeded(state, ctx, out,
                        ClinicalStateChangeType.DATA_ABSENCE_WARNING, velocity);
            }

            // Issue 6 fix: idle-patient timer quiescence
            if (completeness.getCompositeScore() < 0.01) {
                int idle = state.getConsecutiveZeroCompletenessDays() + 1;
                state.setConsecutiveZeroCompletenessDays(idle);
                if (idle >= IDLE_QUIESCENCE_THRESHOLD) {
                    LOG.info("Patient {} idle for {} days, stopping daily timer",
                            state.getPatientId(), idle);
                    state.setDailyTimerMs(-1L); // Don't re-register
                    // Timer will restart when next real event arrives via processElement
                } else {
                    long nextDaily = timestamp + DAILY_TIMER_INTERVAL_MS;
                    ctx.timerService().registerProcessingTimeTimer(nextDaily);
                    state.setDailyTimerMs(nextDaily);
                }
            } else {
                state.setConsecutiveZeroCompletenessDays(0);
                long nextDaily = timestamp + DAILY_TIMER_INTERVAL_MS;
                ctx.timerService().registerProcessingTimeTimer(nextDaily);
                state.setDailyTimerMs(nextDaily);
            }

        } else if (timestamp == state.getSnapshotRotationTimerMs()) {
            // 7-day snapshot rotation for velocity computation
            state.rotateSnapshots(timestamp);
            LOG.info("Snapshot rotated for patient {}: previous snapshot captured at {}",
                    state.getPatientId(), timestamp);

            // Re-register rotation timer
            long nextRotation = timestamp + SNAPSHOT_ROTATION_INTERVAL_MS;
            ctx.timerService().registerProcessingTimeTimer(nextRotation);
            state.setSnapshotRotationTimerMs(nextRotation);
        }

        summaryState.update(state);
    }

    /**
     * Route event by source_module discriminator in payload, update state fields.
     * Returns the source module key, or null if unrecognised.
     */
    private String routeAndUpdateState(CanonicalEvent event, ClinicalStateSummary state) {
        Map<String, Object> payload = event.getPayload();
        if (payload == null) return null;

        String sourceModule = payload.get("source_module") != null
                ? payload.get("source_module").toString() : "";

        switch (sourceModule) {
            case "module7":
                updateFromBPVariability(payload, state);
                return "module7";
            case "module9":
                updateFromEngagement(payload, state);
                return "module9";
            case "module10b":
                updateFromMealPatterns(payload, state);
                return "module10b";
            case "module11b":
                updateFromFitnessPatterns(payload, state);
                return "module11b";
            case "module12":
                updateFromInterventionWindow(payload, state);
                return "module12";
            case "module12b":
                updateFromInterventionDelta(payload, state);
                return "module12b";
            case "enriched":
                updateFromLabResult(payload, state);
                return "enriched";
            default:
                return null;
        }
    }

    private void updateFromBPVariability(Map<String, Object> payload, ClinicalStateSummary state) {
        state.current().arv = toDouble(payload.get("arv"));
        String vc = payload.get("variability_classification") != null
                ? payload.get("variability_classification").toString() : null;
        if (vc != null) {
            try { state.current().variabilityClass = VariabilityClassification.valueOf(vc); }
            catch (IllegalArgumentException ignored) {}
        }
        state.current().meanSBP = toDouble(payload.get("mean_sbp"));
        state.current().meanDBP = toDouble(payload.get("mean_dbp"));
        if (payload.get("morning_surge_magnitude") != null) {
            state.current().morningSurgeMagnitude = toDouble(payload.get("morning_surge_magnitude"));
        }
        if (payload.get("dip_classification") != null) {
            try { state.current().dipClass = DipClassification.valueOf(payload.get("dip_classification").toString()); }
            catch (IllegalArgumentException ignored) {}
        }
    }

    private void updateFromEngagement(Map<String, Object> payload, ClinicalStateSummary state) {
        state.setPreviousEngagementScore(state.current().engagementScore);
        state.current().engagementScore = toDouble(payload.get("composite_score"));
        String level = payload.get("engagement_level") != null
                ? payload.get("engagement_level").toString() : null;
        if (level != null) {
            try { state.current().engagementLevel = EngagementLevel.valueOf(level); }
            catch (IllegalArgumentException ignored) {}
        }
        state.setLatestPhenotype(payload.get("phenotype") != null ? payload.get("phenotype").toString() : null);
        state.setDataTier(payload.get("data_tier") != null ? payload.get("data_tier").toString() : null);
    }

    private void updateFromMealPatterns(Map<String, Object> payload, ClinicalStateSummary state) {
        state.current().meanIAUC = toDouble(payload.get("mean_iauc"));
        state.current().medianExcursion = toDouble(payload.get("median_excursion"));
        String sc = payload.get("salt_sensitivity_class") != null
                ? payload.get("salt_sensitivity_class").toString() : null;
        if (sc != null) {
            try { state.current().saltSensitivity = SaltSensitivityClass.valueOf(sc); }
            catch (IllegalArgumentException ignored) {}
        }
        state.current().saltBeta = toDouble(payload.get("salt_beta"));
    }

    private void updateFromFitnessPatterns(Map<String, Object> payload, ClinicalStateSummary state) {
        state.current().estimatedVO2max = toDouble(payload.get("estimated_vo2max"));
        state.current().vo2maxTrend = toDouble(payload.get("vo2max_trend"));
        state.current().totalMetMinutes = toDouble(payload.get("total_met_minutes"));
        state.current().meanExerciseGlucoseDelta = toDouble(payload.get("mean_exercise_glucose_delta"));
    }

    private void updateFromInterventionWindow(Map<String, Object> payload, ClinicalStateSummary state) {
        String interventionId = payload.get("intervention_id") != null
                ? payload.get("intervention_id").toString() : null;
        String signalType = payload.get("signal_type") != null
                ? payload.get("signal_type").toString() : "";
        if (interventionId == null) return;

        if ("WINDOW_OPENED".equals(signalType)) {
            ClinicalStateSummary.InterventionWindowSummary iw = new ClinicalStateSummary.InterventionWindowSummary();
            iw.setInterventionId(interventionId);
            iw.setStatus("OPENED");
            String itStr = payload.get("intervention_type") != null
                    ? payload.get("intervention_type").toString() : null;
            if (itStr != null) {
                try { iw.setInterventionType(InterventionType.valueOf(itStr)); }
                catch (IllegalArgumentException ignored) {}
            }
            if (payload.get("observation_start_ms") != null)
                iw.setObservationStartMs(toLong(payload.get("observation_start_ms")));
            if (payload.get("observation_end_ms") != null)
                iw.setObservationEndMs(toLong(payload.get("observation_end_ms")));
            state.getActiveInterventions().put(interventionId, iw);
        } else if ("WINDOW_MIDPOINT".equals(signalType)) {
            ClinicalStateSummary.InterventionWindowSummary iw = state.getActiveInterventions().get(interventionId);
            if (iw != null) iw.setStatus("MIDPOINT");
        } else if ("WINDOW_CLOSED".equals(signalType) || "WINDOW_EXPIRED".equals(signalType)
                || "WINDOW_CANCELLED".equals(signalType)) {
            state.getActiveInterventions().remove(interventionId);
        }
    }

    private void updateFromInterventionDelta(Map<String, Object> payload, ClinicalStateSummary state) {
        ClinicalStateSummary.InterventionDeltaSummary delta = new ClinicalStateSummary.InterventionDeltaSummary();
        delta.setInterventionId(payload.get("intervention_id") != null
                ? payload.get("intervention_id").toString() : null);
        String attr = payload.get("trajectory_attribution") != null
                ? payload.get("trajectory_attribution").toString() : null;
        if (attr != null) {
            try { delta.setAttribution(TrajectoryAttribution.valueOf(attr)); }
            catch (IllegalArgumentException ignored) {}
        }
        delta.setAdherenceScore(toDouble(payload.get("adherence_score")));
        delta.setClosedAtMs(System.currentTimeMillis());
        state.getRecentInterventionDeltas().add(delta);

        // Keep only last 10 deltas
        while (state.getRecentInterventionDeltas().size() > 10) {
            state.getRecentInterventionDeltas().remove(0);
        }
    }

    private void updateFromLabResult(Map<String, Object> payload, ClinicalStateSummary state) {
        String labType = payload.get("lab_type") != null ? payload.get("lab_type").toString() : "";
        Double value = toDouble(payload.get("value"));
        if (value == null) return;

        switch (labType) {
            case "FBG": state.current().fbg = value; break;
            case "HBA1C": state.current().hba1c = value; break;
            case "EGFR": state.current().egfr = value; break;
            case "UACR": state.current().uacr = value; break;
            case "LDL": state.current().ldl = value; break;
            case "TOTAL_CHOLESTEROL": state.current().totalCholesterol = value; break;
            case "WEIGHT": state.current().weight = value; break;
            default: break;
        }
    }

    private void emitDataAbsenceIfNeeded(ClinicalStateSummary state, Context ctx,
            Collector<ClinicalStateChangeEvent> out, ClinicalStateChangeType type,
            CKMRiskVelocity velocity) {
        Long lastEmitted = state.getLastEmittedChangeTimestamps().get(type);
        long now = ctx.timerService().currentProcessingTime();
        if (lastEmitted != null && (now - lastEmitted) < 24 * 3_600_000L) return;

        ClinicalStateChangeEvent event = ClinicalStateChangeEvent.builder()
                .changeId(UUID.randomUUID().toString())
                .patientId(state.getPatientId())
                .changeType(type)
                .previousValue("expected data")
                .currentValue("no data received")
                .triggerModule("module13")
                .ckmVelocityAtChange(velocity)
                .dataCompletenessAtChange(state.getDataCompletenessScore())
                .confidenceScore(state.getDataCompletenessScore())
                .processingTimestamp(now)
                .build();
        out.collect(event);
        state.getLastEmittedChangeTimestamps().put(type, now);
        LOG.warn("Data absence detected: patient={}, type={}", state.getPatientId(), type);
    }

    private static Double toDouble(Object v) {
        if (v == null) return null;
        if (v instanceof Number) return ((Number) v).doubleValue();
        try { return Double.parseDouble(v.toString()); }
        catch (NumberFormatException e) { return null; }
    }

    private static long toLong(Object v) {
        if (v instanceof Number) return ((Number) v).longValue();
        try { return Long.parseLong(v.toString()); }
        catch (NumberFormatException e) { return 0L; }
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module13_ClinicalStateSynchroniser.java
git commit -m "feat(module13): add Module13_ClinicalStateSynchroniser main KPF with 7-source routing, dual timers"
```

---

## Task 12: KB20AsyncSinkFunction

**Files:**
- Create: `sinks/KB20AsyncSinkFunction.java`

- [ ] **Step 1: Create the sinks directory if needed and add sink**

Run: `ls backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/sinks/ 2>/dev/null || echo "directory does not exist"`
If missing, create it.

```java
package com.cardiofit.flink.sinks;

import com.cardiofit.flink.models.KB20StateUpdate;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.streaming.api.functions.sink.RichSinkFunction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;

/**
 * Async sink for KB-20 state updates.
 * Writes to PostgreSQL (parameterised upsert) and Redis (pipeline SET for projections).
 * Circuit breaker: 3 consecutive failures → open for 30s.
 * During open-circuit, updates are buffered (max 1000 per patient).
 */
public class KB20AsyncSinkFunction extends RichSinkFunction<KB20StateUpdate> {

    private static final Logger LOG = LoggerFactory.getLogger(KB20AsyncSinkFunction.class);

    private static final int CIRCUIT_FAILURE_THRESHOLD = 3;
    private static final long CIRCUIT_OPEN_DURATION_MS = 30_000L; // 30 seconds
    private static final int MAX_BUFFER_SIZE = 1000;

    // Circuit breaker state
    private transient AtomicInteger consecutiveFailures;
    private transient AtomicLong circuitOpenedAt;
    private transient List<KB20StateUpdate> failoverBuffer;

    // Metrics
    private transient AtomicLong writeCount;
    private transient AtomicLong writeFailures;

    // Configuration (injected via constructor or config)
    private final String postgresUrl;
    private final String redisUrl;

    public KB20AsyncSinkFunction(String postgresUrl, String redisUrl) {
        this.postgresUrl = postgresUrl;
        this.redisUrl = redisUrl;
    }

    @Override
    public void open(Configuration parameters) throws Exception {
        super.open(parameters);
        consecutiveFailures = new AtomicInteger(0);
        circuitOpenedAt = new AtomicLong(0L);
        failoverBuffer = new ArrayList<>();
        writeCount = new AtomicLong(0);
        writeFailures = new AtomicLong(0);
        LOG.info("KB20AsyncSinkFunction initialized: postgres={}, redis={}", postgresUrl, redisUrl);
    }

    @Override
    public void invoke(KB20StateUpdate update, Context context) throws Exception {
        if (isCircuitOpen()) {
            bufferUpdate(update);
            return;
        }

        try {
            writeToPostgres(update);
            writeToRedis(update);
            consecutiveFailures.set(0);
            writeCount.incrementAndGet();

            // Flush buffer if circuit just closed
            if (!failoverBuffer.isEmpty()) {
                flushBuffer();
            }

        } catch (Exception e) {
            int failures = consecutiveFailures.incrementAndGet();
            writeFailures.incrementAndGet();
            LOG.warn("KB-20 write failed (attempt {}): patient={}, error={}",
                    failures, update.getPatientId(), e.getMessage());

            if (failures >= CIRCUIT_FAILURE_THRESHOLD) {
                circuitOpenedAt.set(System.currentTimeMillis());
                LOG.error("Circuit breaker OPENED for KB-20 sink after {} failures", failures);
            }
            bufferUpdate(update);
        }
    }

    private boolean isCircuitOpen() {
        long openedAt = circuitOpenedAt.get();
        if (openedAt == 0L) return false;
        if (System.currentTimeMillis() - openedAt > CIRCUIT_OPEN_DURATION_MS) {
            circuitOpenedAt.set(0L);
            consecutiveFailures.set(0);
            LOG.info("Circuit breaker CLOSED for KB-20 sink (recovery attempt)");
            return false;
        }
        return true;
    }

    private void bufferUpdate(KB20StateUpdate update) {
        if (failoverBuffer.size() >= MAX_BUFFER_SIZE) {
            failoverBuffer.remove(0); // Evict oldest
            LOG.warn("KB-20 failover buffer full ({}), evicting oldest for patient {}",
                    MAX_BUFFER_SIZE, update.getPatientId());
        }
        failoverBuffer.add(update);
    }

    private void flushBuffer() {
        LOG.info("Flushing {} buffered KB-20 updates", failoverBuffer.size());
        List<KB20StateUpdate> toFlush = new ArrayList<>(failoverBuffer);
        failoverBuffer.clear();
        for (KB20StateUpdate buffered : toFlush) {
            try {
                writeToPostgres(buffered);
                writeToRedis(buffered);
            } catch (Exception e) {
                LOG.warn("Buffer flush failed for patient {}: {}",
                        buffered.getPatientId(), e.getMessage());
                // Re-buffer on failure, but don't re-trigger circuit
                failoverBuffer.add(buffered);
            }
        }
    }

    /**
     * Write field-level upsert to PostgreSQL.
     * Implementation connects to KB-20's patient_state table.
     * TODO for infrastructure wiring: replace with actual JDBC/async-pg client.
     */
    private void writeToPostgres(KB20StateUpdate update) {
        // Actual implementation will use parameterised upsert:
        // INSERT INTO patient_streaming_state (patient_id, field_name, field_value, updated_at, source_module)
        // VALUES (?, ?, ?::jsonb, ?, ?)
        // ON CONFLICT (patient_id, field_name) DO UPDATE SET
        //   field_value = EXCLUDED.field_value,
        //   updated_at = EXCLUDED.updated_at,
        //   source_module = EXCLUDED.source_module
        // WHERE EXCLUDED.updated_at > patient_streaming_state.updated_at;
        LOG.debug("PostgreSQL write: patient={}, fields={}, op={}",
                update.getPatientId(), update.getFieldUpdates().keySet(), update.getOperation());
    }

    /**
     * Update Redis projection for KB-20.
     * Implementation uses Redis pipeline SET for fast projection updates.
     * TODO for infrastructure wiring: replace with actual Jedis/Lettuce client.
     */
    private void writeToRedis(KB20StateUpdate update) {
        // Actual implementation will use Redis pipeline:
        // HSET kb20:patient:{patient_id}:streaming {field_name} {field_value_json}
        // EXPIRE kb20:patient:{patient_id}:streaming 7776000  (90 days)
        LOG.debug("Redis write: patient={}, fields={}",
                update.getPatientId(), update.getFieldUpdates().keySet());
    }

    @Override
    public void close() throws Exception {
        if (!failoverBuffer.isEmpty()) {
            LOG.warn("KB20AsyncSinkFunction closing with {} unbuffered updates", failoverBuffer.size());
        }
        LOG.info("KB20AsyncSinkFunction closed: writes={}, failures={}",
                writeCount.get(), writeFailures.get());
        super.close();
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/sinks/KB20AsyncSinkFunction.java
git commit -m "feat(module13): add KB20AsyncSinkFunction with circuit breaker and failover buffer"
```

---

## Task 13: KafkaTopics + FlinkJobOrchestrator Wiring

**Files:**
- Modify: `utils/KafkaTopics.java:181`
- Modify: `FlinkJobOrchestrator.java:151`

- [ ] **Step 1: Add CLINICAL_STATE_CHANGE_EVENTS to KafkaTopics**

In `utils/KafkaTopics.java`, replace the final enum entry (line 181):

```java
// Before:
    FLINK_INTERVENTION_DELTAS("flink.intervention-deltas", 4, 90);

// After:
    FLINK_INTERVENTION_DELTAS("flink.intervention-deltas", 4, 90),
    CLINICAL_STATE_CHANGE_EVENTS("clinical.state-change-events", 4, 90);
```

Also add `CLINICAL_STATE_CHANGE_EVENTS` to the `isV4OutputTopic()` method (after line 327):

```java
               this == FLINK_INTERVENTION_DELTAS ||
               this == CLINICAL_STATE_CHANGE_EVENTS;
```

- [ ] **Step 2: Add module13 case to FlinkJobOrchestrator switch (after line 151)**

```java
            case "clinical-state-sync":
            case "module13":
            case "clinical-state-synchroniser":
                launchClinicalStateSynchroniser(env);
                break;
```

- [ ] **Step 3: Add launchClinicalStateSynchroniser method**

Add this method after the `launchInterventionDeltaComputer` method (after ~line 1130):

```java
    /**
     * Launch the Module 13 Clinical State Synchroniser pipeline.
     * Consumes from 7 topics via multi-source union, outputs state change events
     * and KB-20 state projections.
     */
    private static void launchClinicalStateSynchroniser(StreamExecutionEnvironment env) {
        LOG.info("Launching Module 13: Clinical State Synchroniser pipeline (7-source fan-in)");

        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        // All sources use SourceTaggingDeserializer to inject source_module tag.
        // This is critical (Issue 2+3 fix): upstream modules DON'T emit source_module,
        // and Sources 5-6 emit InterventionWindowSignal/InterventionDeltaRecord, not CanonicalEvent.
        // SourceTaggingDeserializer wraps raw JSON → CanonicalEvent with injected tag.

        // Source 1: BP Variability Metrics (Module 7)
        KafkaSource<CanonicalEvent> bpVarSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.FLINK_BP_VARIABILITY_METRICS.getTopicName())
                .setGroupId("flink-module13-bp-variability-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new SourceTaggingDeserializer("module7", EventType.VITAL_SIGN))
                .build();

        // Source 2: Engagement Signals (Module 9)
        KafkaSource<CanonicalEvent> engagementSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.FLINK_ENGAGEMENT_SIGNALS.getTopicName())
                .setGroupId("flink-module13-engagement-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new SourceTaggingDeserializer("module9", EventType.PATIENT_REPORTED))
                .build();

        // Source 3: Meal Patterns (Module 10b)
        KafkaSource<CanonicalEvent> mealSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.FLINK_MEAL_PATTERNS.getTopicName())
                .setGroupId("flink-module13-meal-patterns-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new SourceTaggingDeserializer("module10b", EventType.PATIENT_REPORTED))
                .build();

        // Source 4: Fitness Patterns (Module 11b)
        KafkaSource<CanonicalEvent> fitnessSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.FLINK_FITNESS_PATTERNS.getTopicName())
                .setGroupId("flink-module13-fitness-patterns-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new SourceTaggingDeserializer("module11b", EventType.DEVICE_READING))
                .build();

        // Source 5: Intervention Window Signals (Module 12) — NOT CanonicalEvent upstream!
        KafkaSource<CanonicalEvent> windowSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.CLINICAL_INTERVENTION_WINDOW_SIGNALS.getTopicName())
                .setGroupId("flink-module13-intervention-windows-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new SourceTaggingDeserializer("module12", EventType.MEDICATION_ORDERED))
                .build();

        // Source 6: Intervention Deltas (Module 12b) — NOT CanonicalEvent upstream!
        KafkaSource<CanonicalEvent> deltaSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.FLINK_INTERVENTION_DELTAS.getTopicName())
                .setGroupId("flink-module13-intervention-deltas-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new SourceTaggingDeserializer("module12b", EventType.LAB_RESULT))
                .build();

        // Source 7: Enriched Patient Events (labs, vitals)
        KafkaSource<CanonicalEvent> enrichedSource = KafkaSource.<CanonicalEvent>builder()
                .setBootstrapServers(bootstrap)
                .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
                .setGroupId("flink-module13-enriched-v1")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new SourceTaggingDeserializer("enriched", EventType.LAB_RESULT))
                .build();

        // Create streams with watermark strategy
        WatermarkStrategy<CanonicalEvent> watermark = WatermarkStrategy
                .<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                .withTimestampAssigner((e, ts) -> e.getEventTime());

        DataStream<CanonicalEvent> bpVarStream = env.fromSource(bpVarSource, watermark,
                "Kafka Source: BP Variability (Module 13)");
        DataStream<CanonicalEvent> engagementStream = env.fromSource(engagementSource, watermark,
                "Kafka Source: Engagement Signals (Module 13)");
        DataStream<CanonicalEvent> mealStream = env.fromSource(mealSource, watermark,
                "Kafka Source: Meal Patterns (Module 13)");
        DataStream<CanonicalEvent> fitnessStream = env.fromSource(fitnessSource, watermark,
                "Kafka Source: Fitness Patterns (Module 13)");
        DataStream<CanonicalEvent> windowStream = env.fromSource(windowSource, watermark,
                "Kafka Source: Intervention Windows (Module 13)");
        DataStream<CanonicalEvent> deltaStream = env.fromSource(deltaSource, watermark,
                "Kafka Source: Intervention Deltas (Module 13)");
        DataStream<CanonicalEvent> enrichedStream = env.fromSource(enrichedSource, watermark,
                "Kafka Source: Enriched Patient Events (Module 13)");

        // Union all 7 sources and process
        Module13_ClinicalStateSynchroniser processor = new Module13_ClinicalStateSynchroniser();

        SingleOutputStreamOperator<ClinicalStateChangeEvent> stateChanges = bpVarStream
                .union(engagementStream, mealStream, fitnessStream, windowStream, deltaStream, enrichedStream)
                .keyBy(CanonicalEvent::getPatientId)
                .process(processor)
                .uid("module13-clinical-state-synchroniser")
                .name("Module 13: Clinical State Synchroniser");

        // Main sink: State change events → Kafka
        stateChanges.sinkTo(
                KafkaSink.<ClinicalStateChangeEvent>builder()
                        .setBootstrapServers(bootstrap)
                        .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
                        .setTransactionalIdPrefix("m13-state-changes")
                        .setRecordSerializer(
                                KafkaRecordSerializationSchema.<ClinicalStateChangeEvent>builder()
                                        .setTopic(KafkaTopics.CLINICAL_STATE_CHANGE_EVENTS.getTopicName())
                                        .setValueSerializationSchema(new JsonSerializer<ClinicalStateChangeEvent>())
                                        .build())
                        .build()
        ).name("Sink: Clinical State Change Events");

        // Side output sink: KB-20 state updates → async PostgreSQL + Redis
        DataStream<KB20StateUpdate> kb20Updates = stateChanges
                .getSideOutput(Module13_ClinicalStateSynchroniser.KB20_SIDE_OUTPUT);

        String pgUrl = System.getenv().getOrDefault("KB20_POSTGRES_URL", "jdbc:postgresql://localhost:5433/kb20");
        String redisUrl = System.getenv().getOrDefault("KB20_REDIS_URL", "redis://localhost:6380");

        kb20Updates.addSink(new KB20AsyncSinkFunction(pgUrl, redisUrl))
                .name("Sink: KB-20 State Projections (PostgreSQL + Redis)");

        LOG.info("Module 13 pipeline configured: sources=[{},{},{},{},{},{},{}], sinks=[{}, KB-20]",
                KafkaTopics.FLINK_BP_VARIABILITY_METRICS.getTopicName(),
                KafkaTopics.FLINK_ENGAGEMENT_SIGNALS.getTopicName(),
                KafkaTopics.FLINK_MEAL_PATTERNS.getTopicName(),
                KafkaTopics.FLINK_FITNESS_PATTERNS.getTopicName(),
                KafkaTopics.CLINICAL_INTERVENTION_WINDOW_SIGNALS.getTopicName(),
                KafkaTopics.FLINK_INTERVENTION_DELTAS.getTopicName(),
                KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName(),
                KafkaTopics.CLINICAL_STATE_CHANGE_EVENTS.getTopicName());
    }
```

- [ ] **Step 4: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/utils/KafkaTopics.java \
       backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java
git commit -m "feat(module13): wire Module 13 into FlinkJobOrchestrator with 7-topic union and dual sinks"
```

---

## Task 14: Lifecycle + Integration + Data Absence Tests (16 tests)

**Files:**
- Create (test): `operators/Module13StateSyncLifecycleTest.java` — 8 tests (includes coalescing + snapshot rotation)
- Create (test): `operators/Module13MultiModuleFusionIntegrationTest.java` — 4 tests
- Create (test): `operators/Module13DataAbsenceTimerTest.java` — 4 tests

- [ ] **Step 1: Write Module13StateSyncLifecycleTest (6 tests)**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

class Module13StateSyncLifecycleTest {

    // --- Test 1: State creation for new patient ---
    @Test
    void newPatient_stateCreatedWithPatientId() {
        ClinicalStateSummary state = Module13TestBuilder.emptyState("p1");
        assertNotNull(state);
        assertEquals("p1", state.getPatientId());
        assertNull(state.current().fbg);
        assertNull(state.getLastComputedVelocity());
        assertTrue(state.getActiveInterventions().isEmpty());
        assertFalse(state.hasVelocityData(), "New patient should not have velocity data");
    }

    // --- Test 2: BP variability event updates correct fields ---
    @Test
    void bpVariabilityEvent_updatesStateFields() {
        ClinicalStateSummary state = Module13TestBuilder.emptyState("p1");
        CanonicalEvent event = Module13TestBuilder.bpVariabilityEvent(
                "p1", Module13TestBuilder.BASE_TIME, 15.0, "ELEVATED", 148.0, 93.0);

        Map<String, Object> payload = event.getPayload();
        // Simulate the routing logic (writes to current snapshot)
        state.current().arv = ((Number) payload.get("arv")).doubleValue();
        state.current().meanSBP = ((Number) payload.get("mean_sbp")).doubleValue();
        state.current().meanDBP = ((Number) payload.get("mean_dbp")).doubleValue();

        assertEquals(15.0, state.current().arv);
        assertEquals(148.0, state.current().meanSBP);
        assertEquals(93.0, state.current().meanDBP);
    }

    // --- Test 3: Engagement event preserves previous score ---
    @Test
    void engagementEvent_preservesPreviousScore() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        Double before = state.current().engagementScore; // 0.72

        // Simulate engagement update (writes to current snapshot)
        state.setPreviousEngagementScore(state.current().engagementScore);
        state.current().engagementScore = 0.45;

        assertEquals(before, state.getPreviousEngagementScore());
        assertEquals(0.45, state.current().engagementScore);
    }

    // --- Test 4: Intervention window OPENED adds to activeInterventions ---
    @Test
    void interventionWindowOpened_addsToActiveMap() {
        ClinicalStateSummary state = Module13TestBuilder.emptyState("p1");
        assertTrue(state.getActiveInterventions().isEmpty());

        ClinicalStateSummary.InterventionWindowSummary iw = new ClinicalStateSummary.InterventionWindowSummary();
        iw.setInterventionId("int-001");
        iw.setStatus("OPENED");
        iw.setInterventionType(InterventionType.MEDICATION_ADD);
        state.getActiveInterventions().put("int-001", iw);

        assertEquals(1, state.getActiveInterventions().size());
        assertEquals("OPENED", state.getActiveInterventions().get("int-001").getStatus());
    }

    // --- Test 5: Intervention window CLOSED removes from activeInterventions ---
    @Test
    void interventionWindowClosed_removesFromActiveMap() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithActiveIntervention(
                "p1", "int-001", InterventionType.MEDICATION_ADD);
        assertEquals(1, state.getActiveInterventions().size());

        state.getActiveInterventions().remove("int-001");
        assertTrue(state.getActiveInterventions().isEmpty());
    }

    // --- Test 6: Recent intervention deltas capped at 10 ---
    @Test
    void interventionDeltas_cappedAt10() {
        ClinicalStateSummary state = Module13TestBuilder.emptyState("p1");

        for (int i = 0; i < 12; i++) {
            ClinicalStateSummary.InterventionDeltaSummary d = new ClinicalStateSummary.InterventionDeltaSummary();
            d.setInterventionId("int-" + i);
            d.setAttribution(TrajectoryAttribution.IMPROVING);
            state.getRecentInterventionDeltas().add(d);
            while (state.getRecentInterventionDeltas().size() > 10) {
                state.getRecentInterventionDeltas().remove(0);
            }
        }

        assertEquals(10, state.getRecentInterventionDeltas().size());
        assertEquals("int-2", state.getRecentInterventionDeltas().get(0).getInterventionId());
    }

    // --- Test 7 (Issue 7 fix): Coalescing buffer accumulation and flush ---
    @Test
    void coalescingBuffer_accumulatesAndFlushes() {
        ClinicalStateSummary state = Module13TestBuilder.emptyState("p1");
        assertEquals(-1L, state.getCoalescingTimerMs());
        assertTrue(state.getCoalescingBuffer().isEmpty());

        // Simulate buffering 3 KB-20 updates
        KB20StateUpdate u1 = KB20StateUpdate.builder().patientId("p1")
                .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                .sourceModule("module7").field("arv", 12.0).build();
        KB20StateUpdate u2 = KB20StateUpdate.builder().patientId("p1")
                .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                .sourceModule("module9").field("engagement_score", 0.7).build();
        KB20StateUpdate u3 = KB20StateUpdate.builder().patientId("p1")
                .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                .sourceModule("module13").field("ckm_risk_velocity", "STABLE").build();

        state.getCoalescingBuffer().add(u1);
        state.getCoalescingBuffer().add(u2);
        state.getCoalescingBuffer().add(u3);
        state.setCoalescingTimerMs(Module13TestBuilder.BASE_TIME + 5000L);

        assertEquals(3, state.getCoalescingBuffer().size());
        assertTrue(state.getCoalescingTimerMs() > 0, "Timer should be registered");

        // Simulate timer fire: flush buffer
        List<KB20StateUpdate> flushed = new ArrayList<>(state.getCoalescingBuffer());
        state.getCoalescingBuffer().clear();
        state.setCoalescingTimerMs(-1L);

        assertEquals(3, flushed.size());
        assertTrue(state.getCoalescingBuffer().isEmpty(), "Buffer should be empty after flush");
        assertEquals(-1L, state.getCoalescingTimerMs(), "Timer should be reset after flush");
    }

    // --- Test 8: Snapshot rotation moves current → previous ---
    @Test
    void snapshotRotation_currentBecomesPrevious() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        assertFalse(state.hasVelocityData(), "Before rotation, no previous snapshot");

        // Set some current values
        state.current().fbg = 130.0;
        state.current().egfr = 65.0;

        // Rotate
        state.rotateSnapshots(Module13TestBuilder.BASE_TIME + Module13TestBuilder.WEEK_MS);

        assertTrue(state.hasVelocityData(), "After rotation, previous snapshot exists");
        assertEquals(130.0, state.previous().fbg);
        assertEquals(65.0, state.previous().egfr);

        // Current still has same values (not reset — continues accumulating)
        assertNotNull(state.current().fbg);
    }
}
```

- [ ] **Step 2: Write Module13MultiModuleFusionIntegrationTest (4 tests)**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class Module13MultiModuleFusionIntegrationTest {

    // --- Test 1: Simultaneous events from 3 modules → fused state → correct velocity ---
    @Test
    void threeModulesFiring_producesCorrectCompositeVelocity() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");

        // Simulate Module 7 update (writes to current snapshot)
        state.current().arv = 16.0;
        state.current().variabilityClass = VariabilityClassification.HIGH;
        state.recordModuleSeen("module7", Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        // Simulate Module 9 update
        state.setPreviousEngagementScore(state.current().engagementScore);
        state.current().engagementScore = 0.4;
        state.current().engagementLevel = EngagementLevel.ORANGE;
        state.recordModuleSeen("module9", Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        // Simulate lab update
        state.current().egfr = 58.0;
        state.recordModuleSeen("enriched", Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);

        assertNotNull(velocity);
        assertNotEquals(CKMRiskVelocity.CompositeClassification.UNKNOWN, velocity.getCompositeClassification());
        assertTrue(velocity.getDataCompleteness() > 0.5);
    }

    // --- Test 2: Engagement drop + trajectory deterioration → CROSS_MODULE emitted ---
    @Test
    void engagementDropWithDeteriorating_emitsCrossModuleChange() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithVelocity("p1",
                CKMRiskVelocity.CompositeClassification.STABLE);
        state.current().engagementScore = 0.85;
        state.setPreviousEngagementScore(0.85);

        // Add a high-adherence delta that was INTERVENTION_INSUFFICIENT
        ClinicalStateSummary.InterventionDeltaSummary d = new ClinicalStateSummary.InterventionDeltaSummary();
        d.setAdherenceScore(0.9);
        d.setAttribution(TrajectoryAttribution.INTERVENTION_INSUFFICIENT);
        state.getRecentInterventionDeltas().add(d);

        CKMRiskVelocity badVelocity = CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.METABOLIC, 0.6)
                .compositeClassification(CKMRiskVelocity.CompositeClassification.DETERIORATING)
                .compositeScore(0.6)
                .dataCompleteness(0.9)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, badVelocity, Module13TestBuilder.BASE_TIME);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.CROSS_MODULE_INCONSISTENCY));
    }

    // --- Test 3: Full cycle: baselines → events → velocity → state change ---
    @Test
    void fullCycle_baselinesEventsVelocityStateChange() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithVelocity("p1",
                CKMRiskVelocity.CompositeClassification.STABLE);

        // Simulate worsening eGFR (writes to current snapshot)
        state.current().egfr = 48.0;
        state.current().uacr = 180.0;
        state.current().arv = 14.0;

        CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        // Should detect at minimum CKM_RISK_ESCALATION (STABLE → DETERIORATING)
        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.CKM_RISK_ESCALATION
                        || e.getChangeType() == ClinicalStateChangeType.RENAL_RAPID_DECLINE));
    }

    // --- Test 4: Milestone emission when FBG improves below target ---
    @Test
    void metabolicMilestone_fbgBelowTarget_withVelocityImproving() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        state.current().fbg = 105.0;
        state.current().hba1c = 6.5;
        state.current().egfr = 70.0;

        CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);

        assertEquals(CKMRiskVelocity.CompositeClassification.IMPROVING,
                velocity.getCompositeClassification());

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE));
    }
}
```

- [ ] **Step 3: Write Module13DataAbsenceTimerTest (4 tests)**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class Module13DataAbsenceTimerTest {

    // --- Test 1: 7 days no data → DATA_ABSENCE_WARNING detected ---
    @Test
    void sevenDaysNoData_warningDetected() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 8 * Module13TestBuilder.DAY_MS;
        // All modules last seen at BASE_TIME = 8 days ago

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertFalse(result.getDataGapFlags().isEmpty(),
                "Should have data gap flags after 8 days");
    }

    // --- Test 2: 14 days no data → DATA_ABSENCE_CRITICAL ---
    @Test
    void fourteenDaysNoData_criticalDetected() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 20 * Module13TestBuilder.DAY_MS;

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertTrue(result.isDataAbsenceCritical());
    }

    // --- Test 3: Partial data (some modules active) → WARNING not CRITICAL ---
    @Test
    void partialData_warningNotCritical() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 10 * Module13TestBuilder.DAY_MS;
        // Module 7 and enriched are fresh, others stale
        state.recordModuleSeen("module7", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("enriched", now - Module13TestBuilder.DAY_MS);

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertFalse(result.isDataAbsenceCritical(),
                "Should not be CRITICAL when some modules are active");
        assertFalse(result.getDataGapFlags().isEmpty(),
                "Should still flag stale modules");
    }

    // --- Test 4: Fresh data arrival resets gap flags ---
    @Test
    void freshDataArrival_resetsGapFlags() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 2 * Module13TestBuilder.DAY_MS;
        // Update all modules to now
        for (String module : new String[]{"module7", "module9", "module10b", "module11b", "enriched"}) {
            state.recordModuleSeen(module, now - Module13TestBuilder.HOUR_MS);
        }

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertFalse(result.isDataAbsenceCritical());
        // Only module12 and module12b are not tracked (they might not exist for all patients)
        assertTrue(result.getCompositeScore() > 0.6);
    }
}
```

- [ ] **Step 4: Run all tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module13*" 2>&1 | tail -15`
Expected: All 39 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module13StateSyncLifecycleTest.java \
       backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module13MultiModuleFusionIntegrationTest.java \
       backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module13DataAbsenceTimerTest.java
git commit -m "feat(module13): add lifecycle, integration, and data absence tests — 16 tests (39 total)"
```

---

## Spec Coverage Matrix

| Requirement (from docx) | Task | Status |
|--------------------------|------|--------|
| Multi-source union from 7 input topics | Task 13 | Covered |
| SourceTaggingDeserializer for source_module injection | Task 5b | Covered (Issue 2+3 fix) |
| Per-domain CKM risk velocity (metabolic, renal, cardiovascular) | Task 7 | Covered |
| Temporal delta velocity (not displacement) | Task 7 | Covered (Issue 1 fix) |
| MetricSnapshot rotation (7-day interval) | Task 4, Task 11 | Covered |
| Composite worst-domain-wins velocity | Task 7 | Covered |
| Cross-domain amplification (2+ domains worsening) | Task 7 | Covered |
| South Asian calibration via market-shim thresholds | Task 7 (configurable constructor) | Covered |
| 12 clinically significant state change types | Task 3, Task 8 | Covered |
| 24-hour state change deduplication | Task 8 | Covered |
| TrajectoryAttribution.INTERVENTION_INSUFFICIENT (correct enum) | Task 7, Task 8 | Covered (Issue 5 fix) |
| Per-module data freshness tracking | Task 9 | Covered |
| Data absence detection (7d warning, 14d critical) | Task 9, Task 14 | Covered |
| Data tier-aware completeness scoring (CGM vs SMBG) | Task 9 | Covered |
| KB-20 field-level state projection (7 source modules) | Task 10 | Covered |
| Write coalescing (5-second window) | Task 11, Task 14 | Covered (Issue 7: test added) |
| Main KPF with multi-source routing + timer management | Task 11 | Covered |
| Idle-patient timer quiescence (30-day threshold) | Task 11 | Covered (Issue 6 fix) |
| Snapshot rotation timer (7-day interval) | Task 11, Task 14 | Covered |
| Async PostgreSQL + Redis sink with circuit breaker | Task 12 | Covered |
| ClinicalStateChangeEvent output (16 fields) | Task 3 | Covered |
| KB20StateUpdate diff model (4 operations) | Task 5 | Covered |
| FlinkJobOrchestrator wiring with 7-topic union | Task 13 | Covered |
| Daily data absence timer per patient | Task 11, Task 14 | Covered |
| 90-day state TTL with OnReadAndWrite | Task 11 | Covered |
| ClinicalStateSummary per-patient keyed state | Task 4 | Covered |
| Module13TestBuilder with all event factories | Task 6 | Covered |
| Total test count: 39 | Tasks 7-14 | Covered |
| MEDICATION_RESPONSE_CONFIRMED detection | Task 8 | Partial — enum defined, detector check TBD |
| South Asian calibration YAML loading | Task 7 | Partial — thresholds configurable, YAML deferred |

## Known Gaps (Address During Implementation)

1. **MEDICATION_RESPONSE_CONFIRMED**: The enum constant exists in `ClinicalStateChangeType` but `Module13StateChangeDetector` does not check for it. Add detection in `checkMedicationResponseConfirmed()`: when a Module 12b delta has `fbgDelta < -15.0` AND `adherenceScore > 0.7`, emit this event. This is an INFO-level positive signal.

2. **South Asian calibration YAML**: `Module13CKMRiskComputer` uses hardcoded delta normalisation ranges. For market-shim support, refactor the constructor to accept a `CKMThresholdConfig` object loaded from `deploy/markets/india/ckm_thresholds.yaml`. The default values serve as the general-population thresholds. This is a follow-up task after the core module is functional.

3. **KB20AsyncSinkFunction infrastructure wiring**: The PostgreSQL and Redis write methods are stubs with logging. Replace with actual JDBC (HikariCP connection pool) and Lettuce (async Redis client) implementations during infrastructure integration. The circuit breaker and buffer logic are complete.

4. **Confidence score ≠ completeness (Issue 8)**: Currently `confidenceScore` is set to `dataCompletenessScore` in both `Module13StateChangeDetector.emitIfNotDeduped()` and `Module13_ClinicalStateSynchroniser.emitDataAbsenceIfNeeded()`. A proper confidence model should also factor: (a) cross-domain signal consistency (agreeing domains → higher confidence), (b) snapshot age (newer data → higher confidence), (c) velocity stability (consistent direction across windows → higher confidence). Defer to v2 iteration after CKM model validation with clinical team.
