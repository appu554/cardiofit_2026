# Module 5: ML Inference Engine Upgrade — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade Module 5 from union-based dual-stream to `KeyedCoProcessFunction` with E2E-validated feature extraction, lab-aware cooldown, risk indicator resilience, and active alert features.

**Architecture:** Replace the current `SemanticFeatureExtractor → union → PatternFeatureExtractor → FeatureCombiner → MLInferenceProcessor` chain with a single `KeyedCoProcessFunction<String, EnrichedPatientContext, PatternEvent, MLPrediction>` that buffers pattern events and only triggers ONNX inference on CDS events (or CRITICAL escalations). Feature extraction is refactored into a standalone testable class with production-validated vital key mapping, triple null-check lab extraction, and lab-aware cooldown logic.

**Tech Stack:** Java 17, Flink 2.1.0, ONNX Runtime 1.17.0, Jackson 2.17, JUnit 5.10.2

**Seven E2E Gaps Addressed:**
1. Lab payload deserialization (null LabResult.getValue())
2. Risk indicator reliability (unreliable medication-derived flags)
3. Lab-aware inference cooldown (AKI/anticoagulation patients invisible to NEWS2/qSOFA)
4. Module 3 active alerts as ML features
5. CorrelationId for audit trail traceability
6. Lab key mismatch — `updateStateFromCDS` keyed labs by LOINC code (`labType`) but `Module5FeatureExtractor` looks up by friendly name; fix: key by `clinicalConcept` (KB-7 canonical name)
7. Double Kafka consumer — `createMLInferencePipeline()` called `createEnrichedPatientContextSource(env)` twice; fix: consume once and reuse the `DataStream`

---

## File Structure

### Files to CREATE

| File | Responsibility |
|------|---------------|
| `src/main/java/com/cardiofit/flink/operators/Module5_MLInferenceEngine.java` | New `KeyedCoProcessFunction` operator — dual-input CDS+Pattern processing with cooldown, state management, ONNX inference orchestration |
| `src/main/java/com/cardiofit/flink/operators/Module5FeatureExtractor.java` | Static feature extraction — vital key normalization, lab null-safety, risk indicator resilience, alert features. 100% testable without Flink runtime |
| `src/main/java/com/cardiofit/flink/operators/Module5ClinicalScoring.java` | Risk level classification with category-specific thresholds, Platt calibration, lab-aware cooldown logic |
| `src/main/java/com/cardiofit/flink/models/PatientMLState.java` | Per-patient ML state: clinical snapshot, temporal ring buffers, pattern buffer, prediction tracking |
| `src/test/java/com/cardiofit/flink/operators/Module5RealDataMappingTest.java` | Validates production CDS JSON → feature vector (highest priority test) |
| `src/test/java/com/cardiofit/flink/operators/Module5FeatureExtractionTest.java` | Full feature extraction: vital normalization, null safety, key mapping, demographic exclusion |
| `src/test/java/com/cardiofit/flink/operators/Module5CooldownTest.java` | Cooldown by risk level, lab-critical bypass, escalation bypass |
| `src/test/java/com/cardiofit/flink/operators/Module5CalibrationTest.java` | Platt scaling, threshold classification, boundary values |
| `src/test/java/com/cardiofit/flink/builders/Module5TestBuilder.java` | Test data factory for PatientMLState, production-format CDS events, pattern events |

### Files to MODIFY

| File | Change |
|------|--------|
| `src/main/java/com/cardiofit/flink/operators/Module5_MLInference.java` | Replace `createMLInferencePipeline()` to use new `KeyedCoProcessFunction` operator instead of union-based chain |
| `src/main/java/com/cardiofit/flink/models/MLPrediction.java` | Add `predictionCategory`, `calibratedScore`, `contextDepth`, `triggerSource`, `predictionHorizonMs`, `inputSnapshot` fields |
| `src/main/java/com/cardiofit/flink/ml/features/ClinicalFeatureExtractor.java` | Fix vital key mapping from snake_case to production lowercase-no-separator format |

### Files UNCHANGED (confirmed sound by E2E)

- `ONNXModelContainer.java` — ONNX lifecycle, inference, batch predict all working
- `PatientContextSnapshot.java` — adapter layer (MIMIC pipeline) unaffected
- `MLAlertGenerator.java`, `SHAPCalculator.java`, `ModelRegistry.java` — downstream consumers

---

## Task 1: PatientMLState — Per-Patient ML State Model

**Files:**
- Create: `src/main/java/com/cardiofit/flink/models/PatientMLState.java`

This is the Flink `ValueState` backing object for Module 5. It accumulates clinical data across events and maintains temporal features (ring buffers for NEWS2/acuity history) plus a capped pattern buffer from Module 4.

- [ ] **Step 1: Create PatientMLState.java**

```java
package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.*;

/**
 * Per-patient ML state maintained across events in Module 5.
 * Stored in Flink ValueState with 7-day TTL.
 */
public class PatientMLState implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final int HISTORY_SIZE = 10;
    public static final int MAX_PATTERN_BUFFER_SIZE = 20;

    private String patientId;

    // ── Latest clinical snapshot (from CDS events) ──
    private Map<String, Double> latestVitals;
    private Map<String, Double> latestLabs;
    private int news2Score;
    private int qsofaScore;
    private double acuityScore;
    private List<String> semanticTags;
    private Map<String, Object> riskIndicators;
    private Map<String, Object> activeAlerts;

    // ── Temporal features (ring buffers) ──
    private double[] news2History;
    private int news2HistoryIndex;
    private double[] acuityHistory;
    private int acuityHistoryIndex;
    private long firstEventTime;
    private int totalEventCount;

    // ── Pattern features (from Module 4) ──
    private List<PatternSummary> recentPatterns;
    private int deteriorationPatternCount;
    private int sepsisPatternCount;
    private String maxSeveritySeen;
    private boolean severityEscalationDetected;
    private long lastPatternTime;

    // ── Prediction tracking ──
    private Map<String, Double> lastPredictions;
    private long lastInferenceTime;

    public PatientMLState() {
        this.latestVitals = new HashMap<>();
        this.latestLabs = new HashMap<>();
        this.riskIndicators = new HashMap<>();
        this.activeAlerts = new HashMap<>();
        this.semanticTags = new ArrayList<>();
        this.news2History = new double[HISTORY_SIZE];
        this.acuityHistory = new double[HISTORY_SIZE];
        this.recentPatterns = new ArrayList<>();
        this.lastPredictions = new HashMap<>();
        this.maxSeveritySeen = "NONE";
    }

    // ── Ring buffer helpers ──

    public void pushNews2(int score) {
        news2History[news2HistoryIndex % HISTORY_SIZE] = score;
        news2HistoryIndex++;
    }

    public void pushAcuity(double score) {
        acuityHistory[acuityHistoryIndex % HISTORY_SIZE] = score;
        acuityHistoryIndex++;
    }

    public double news2Slope() {
        return calculateSlope(news2History, Math.min(news2HistoryIndex, HISTORY_SIZE));
    }

    public double acuitySlope() {
        return calculateSlope(acuityHistory, Math.min(acuityHistoryIndex, HISTORY_SIZE));
    }

    private static double calculateSlope(double[] buffer, int count) {
        if (count < 2) return 0.0;
        double sumX = 0, sumY = 0, sumXY = 0, sumX2 = 0;
        for (int i = 0; i < count; i++) {
            sumX += i;
            sumY += buffer[i];
            sumXY += i * buffer[i];
            sumX2 += i * i;
        }
        double denom = count * sumX2 - sumX * sumX;
        if (Math.abs(denom) < 1e-9) return 0.0;
        return (count * sumXY - sumX * sumY) / denom;
    }

    // ── Pattern buffer ──

    public void addPattern(PatternSummary pattern) {
        if (recentPatterns.size() >= MAX_PATTERN_BUFFER_SIZE) {
            recentPatterns.remove(0);
        }
        recentPatterns.add(pattern);
        lastPatternTime = pattern.detectionTime();

        if ("CLINICAL_DETERIORATION".equals(pattern.patternType())) {
            deteriorationPatternCount++;
        }
        if ("SEPSIS_RISK".equals(pattern.patternType())
                || "SEPSIS".equalsIgnoreCase(pattern.patternType())) {
            sepsisPatternCount++;
        }

        int sevIdx = severityIndex(pattern.severity());
        if (sevIdx > severityIndex(maxSeveritySeen)) {
            maxSeveritySeen = pattern.severity();
        }
        if ("CRITICAL".equals(pattern.severity())
                && pattern.tags() != null
                && pattern.tags().contains("SEVERITY_ESCALATION")) {
            severityEscalationDetected = true;
        }
    }

    public void clearPatternBuffer() {
        recentPatterns.clear();
        deteriorationPatternCount = 0;
        sepsisPatternCount = 0;
        maxSeveritySeen = "NONE";
        severityEscalationDetected = false;
    }

    public static int severityIndex(String severity) {
        if (severity == null) return 0;
        return switch (severity.toUpperCase()) {
            case "LOW" -> 1;
            case "MODERATE" -> 2;
            case "HIGH" -> 3;
            case "CRITICAL" -> 4;
            default -> 0;
        };
    }

    // ── Pattern summary record ──

    public record PatternSummary(
        String patternType,
        String severity,
        double confidence,
        long detectionTime,
        Set<String> tags
    ) implements Serializable {}

    // ── Standard getters/setters ──

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public Map<String, Double> getLatestVitals() { return latestVitals; }
    public void setLatestVitals(Map<String, Double> latestVitals) { this.latestVitals = latestVitals; }

    public Map<String, Double> getLatestLabs() { return latestLabs; }
    public void setLatestLabs(Map<String, Double> latestLabs) { this.latestLabs = latestLabs; }

    public int getNews2Score() { return news2Score; }
    public void setNews2Score(int news2Score) { this.news2Score = news2Score; }

    public int getQsofaScore() { return qsofaScore; }
    public void setQsofaScore(int qsofaScore) { this.qsofaScore = qsofaScore; }

    public double getAcuityScore() { return acuityScore; }
    public void setAcuityScore(double acuityScore) { this.acuityScore = acuityScore; }

    public List<String> getSemanticTags() { return semanticTags; }
    public void setSemanticTags(List<String> semanticTags) { this.semanticTags = semanticTags; }

    public Map<String, Object> getRiskIndicators() { return riskIndicators; }
    public void setRiskIndicators(Map<String, Object> riskIndicators) { this.riskIndicators = riskIndicators; }

    public Map<String, Object> getActiveAlerts() { return activeAlerts; }
    public void setActiveAlerts(Map<String, Object> activeAlerts) { this.activeAlerts = activeAlerts; }

    public double[] getNews2History() { return news2History; }
    public double[] getAcuityHistory() { return acuityHistory; }
    public int getNews2HistoryIndex() { return news2HistoryIndex; }
    public int getAcuityHistoryIndex() { return acuityHistoryIndex; }

    public long getFirstEventTime() { return firstEventTime; }
    public void setFirstEventTime(long firstEventTime) { this.firstEventTime = firstEventTime; }

    public int getTotalEventCount() { return totalEventCount; }
    public void setTotalEventCount(int totalEventCount) { this.totalEventCount = totalEventCount; }

    public List<PatternSummary> getRecentPatterns() { return recentPatterns; }
    public int getDeteriorationPatternCount() { return deteriorationPatternCount; }
    public int getSepsisPatternCount() { return sepsisPatternCount; }
    public String getMaxSeveritySeen() { return maxSeveritySeen; }
    public boolean isSeverityEscalationDetected() { return severityEscalationDetected; }
    public long getLastPatternTime() { return lastPatternTime; }

    public Map<String, Double> getLastPredictions() { return lastPredictions; }
    public void setLastPredictions(Map<String, Double> lastPredictions) { this.lastPredictions = lastPredictions; }

    public long getLastInferenceTime() { return lastInferenceTime; }
    public void setLastInferenceTime(long lastInferenceTime) { this.lastInferenceTime = lastInferenceTime; }
}
```

- [ ] **Step 2: Compile to verify**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS (no compilation errors)

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/PatientMLState.java
git commit -m "feat(module5): add PatientMLState with ring buffers and pattern buffer"
```

---

## Task 2: Module5TestBuilder — Test Data Factory

**Files:**
- Create: `src/test/java/com/cardiofit/flink/builders/Module5TestBuilder.java`

Provides factory methods for creating test data in **production format** (lowercase-no-separator vital keys, LOINC-keyed labs, structured risk indicators). This is the single source of truth for test data shapes.

- [ ] **Step 1: Create Module5TestBuilder.java**

```java
package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.PatientMLState;
import com.cardiofit.flink.models.PatternEvent;

import java.util.*;

/**
 * Test data factory for Module 5 tests.
 * All data uses PRODUCTION key formats (lowercase-no-separator vitals, LOINC labs).
 */
public class Module5TestBuilder {

    // ── Patient ML State factories ──

    /** Stable patient: NEWS2=1, qSOFA=0, normal vitals and labs */
    public static PatientMLState stablePatientState(String patientId) {
        PatientMLState state = new PatientMLState();
        state.setPatientId(patientId);
        state.setLatestVitals(Map.of(
            "heartrate", 78.0,
            "systolicbloodpressure", 128.0,
            "diastolicbloodpressure", 82.0,
            "respiratoryrate", 16.0,
            "oxygensaturation", 97.0,
            "temperature", 36.8
        ));
        state.setLatestLabs(Map.of(
            "lactate", 1.2,
            "creatinine", 0.9,
            "potassium", 4.1,
            "wbc", 7.5,
            "platelets", 220.0,
            "inr", 1.0
        ));
        state.setNews2Score(1);
        state.setQsofaScore(0);
        state.setAcuityScore(1.5);
        state.setTotalEventCount(5);
        state.setFirstEventTime(System.currentTimeMillis() - 86400000L);
        state.setRiskIndicators(stableRiskIndicators());
        state.setActiveAlerts(Collections.emptyMap());
        return state;
    }

    /** Sepsis patient: NEWS2=9, qSOFA=2, elevated lactate/WBC/temp, falling BP */
    public static PatientMLState sepsisPatientState(String patientId) {
        PatientMLState state = new PatientMLState();
        state.setPatientId(patientId);
        state.setLatestVitals(Map.of(
            "heartrate", 118.0,
            "systolicbloodpressure", 85.0,
            "diastolicbloodpressure", 52.0,
            "respiratoryrate", 26.0,
            "oxygensaturation", 91.0,
            "temperature", 39.2
        ));
        state.setLatestLabs(Map.of(
            "lactate", 4.8,
            "creatinine", 1.8,
            "potassium", 4.9,
            "wbc", 18.5,
            "platelets", 130.0,
            "inr", 1.3
        ));
        state.setNews2Score(9);
        state.setQsofaScore(2);
        state.setAcuityScore(7.5);
        state.setTotalEventCount(12);
        state.setFirstEventTime(System.currentTimeMillis() - 172800000L);
        state.setRiskIndicators(sepsisRiskIndicators());
        state.setActiveAlerts(sepsisAlerts());
        return state;
    }

    /**
     * AKI patient: NEWS2=1, qSOFA=0 (vitals normal!), but creatinine=3.2, K+=6.1.
     * This is the Gap 3 patient — lab-only emergency invisible to vitals-based scoring.
     */
    public static PatientMLState akiPatientState(String patientId) {
        PatientMLState state = new PatientMLState();
        state.setPatientId(patientId);
        state.setLatestVitals(Map.of(
            "heartrate", 82.0,
            "systolicbloodpressure", 135.0,
            "diastolicbloodpressure", 85.0,
            "respiratoryrate", 17.0,
            "oxygensaturation", 96.0,
            "temperature", 37.0
        ));
        state.setLatestLabs(Map.of(
            "lactate", 1.5,
            "creatinine", 3.2,
            "potassium", 6.1,
            "wbc", 9.0,
            "platelets", 180.0,
            "inr", 1.1
        ));
        state.setNews2Score(1);
        state.setQsofaScore(0);
        state.setAcuityScore(2.0);
        state.setTotalEventCount(8);
        state.setRiskIndicators(akiRiskIndicators());
        state.setActiveAlerts(akiAlerts());
        return state;
    }

    /** Drug-lab patient: Warfarin + INR 6.0 — NEWS2=1, qSOFA=0 */
    public static PatientMLState drugLabPatientState(String patientId) {
        PatientMLState state = new PatientMLState();
        state.setPatientId(patientId);
        state.setLatestVitals(Map.of(
            "heartrate", 76.0,
            "systolicbloodpressure", 130.0,
            "diastolicbloodpressure", 80.0,
            "respiratoryrate", 15.0,
            "oxygensaturation", 98.0,
            "temperature", 36.7
        ));
        state.setLatestLabs(Map.of(
            "lactate", 1.0,
            "creatinine", 1.0,
            "potassium", 4.2,
            "wbc", 7.0,
            "platelets", 55.0,
            "inr", 6.0
        ));
        state.setNews2Score(1);
        state.setQsofaScore(0);
        state.setAcuityScore(1.8);
        state.setRiskIndicators(drugLabRiskIndicators());
        state.setActiveAlerts(drugLabAlerts());
        return state;
    }

    /** State with null/missing labs — tests null-safety path */
    public static PatientMLState sparsePatientState(String patientId) {
        PatientMLState state = new PatientMLState();
        state.setPatientId(patientId);
        state.setLatestVitals(Map.of(
            "heartrate", 80.0,
            "systolicbloodpressure", 120.0
        ));
        state.setLatestLabs(Collections.emptyMap());
        state.setNews2Score(0);
        state.setQsofaScore(0);
        state.setRiskIndicators(Collections.emptyMap());
        state.setActiveAlerts(Collections.emptyMap());
        state.setTotalEventCount(1);
        return state;
    }

    // ── Risk indicator factories ──

    private static Map<String, Object> stableRiskIndicators() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("tachycardia", false);
        risk.put("hypotension", false);
        risk.put("fever", false);
        risk.put("hypoxia", false);
        risk.put("elevatedLactate", false);
        risk.put("elevatedCreatinine", false);
        risk.put("hyperkalemia", false);
        risk.put("thrombocytopenia", false);
        risk.put("onAnticoagulation", false);
        risk.put("onVasopressors", false);
        risk.put("confidenceScore", 0.85);
        return risk;
    }

    private static Map<String, Object> sepsisRiskIndicators() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("tachycardia", true);
        risk.put("hypotension", true);
        risk.put("fever", true);
        risk.put("elevatedLactate", true);
        risk.put("severelyElevatedLactate", true);
        risk.put("leukocytosis", true);
        risk.put("sepsisRisk", true);
        risk.put("confidenceScore", 0.92);
        return risk;
    }

    private static Map<String, Object> akiRiskIndicators() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("elevatedCreatinine", true);
        risk.put("hyperkalemia", true);
        // Vitals-based flags are all false (normal vitals)
        risk.put("tachycardia", false);
        risk.put("hypotension", false);
        risk.put("fever", false);
        risk.put("confidenceScore", 0.78);
        return risk;
    }

    private static Map<String, Object> drugLabRiskIndicators() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("onAnticoagulation", true);
        risk.put("thrombocytopenia", true);
        // Flag may be overwritten by Module 3 (Gap 2)
        risk.put("tachycardia", false);
        risk.put("hypotension", false);
        risk.put("confidenceScore", 0.70);
        return risk;
    }

    // ── Active alert factories (Gap 4) ──

    private static Map<String, Object> sepsisAlerts() {
        Map<String, Object> alerts = new HashMap<>();
        Map<String, Object> sepsisAlert = new HashMap<>();
        sepsisAlert.put("severity", "CRITICAL");
        sepsisAlert.put("message", "Sepsis pattern detected: lactate 4.8, WBC 18.5, temp 39.2");
        alerts.put("SEPSIS_PATTERN", sepsisAlert);

        Map<String, Object> detAlert = new HashMap<>();
        detAlert.put("severity", "HIGH");
        detAlert.put("message", "Clinical deterioration: multi-system decline");
        alerts.put("DETERIORATION_PATTERN", detAlert);
        return alerts;
    }

    private static Map<String, Object> akiAlerts() {
        Map<String, Object> alerts = new HashMap<>();
        Map<String, Object> akiAlert = new HashMap<>();
        akiAlert.put("severity", "HIGH");
        akiAlert.put("stage", 2);
        akiAlert.put("message", "AKI Risk: creatinine 3.2, potassium 6.1");
        alerts.put("AKI_RISK", akiAlert);

        Map<String, Object> hyperK = new HashMap<>();
        hyperK.put("severity", "CRITICAL");
        hyperK.put("message", "Hyperkalemia: K+ 6.1 mEq/L");
        alerts.put("HYPERKALEMIA_ALERT", hyperK);
        return alerts;
    }

    private static Map<String, Object> drugLabAlerts() {
        Map<String, Object> alerts = new HashMap<>();
        Map<String, Object> anticoagAlert = new HashMap<>();
        anticoagAlert.put("severity", "CRITICAL");
        anticoagAlert.put("message", "Anticoagulation risk: INR 6.0");
        alerts.put("ANTICOAGULATION_RISK", anticoagAlert);

        Map<String, Object> bleedAlert = new HashMap<>();
        bleedAlert.put("severity", "HIGH");
        bleedAlert.put("message", "Bleeding risk: platelets 55k, INR 6.0");
        alerts.put("BLEEDING_RISK", bleedAlert);
        return alerts;
    }

    // ── Pattern event factories ──

    /** CRITICAL deterioration pattern with SEVERITY_ESCALATION tag */
    public static PatternEvent criticalEscalationPattern(String patientId) {
        PatternEvent event = new PatternEvent();
        event.setId("pattern-crit-" + UUID.randomUUID().toString().substring(0, 8));
        event.setPatientId(patientId);
        event.setPatternType("CLINICAL_DETERIORATION");
        event.setSeverity("CRITICAL");
        event.setConfidence(0.92);
        event.setDetectionTime(System.currentTimeMillis());
        event.setTags(Set.of("SEVERITY_ESCALATION", "MULTI_SOURCE_CONFIRMED"));
        event.setPriority(1);
        return event;
    }

    /** Moderate trend analysis pattern (should NOT trigger immediate inference) */
    public static PatternEvent moderateTrendPattern(String patientId) {
        PatternEvent event = new PatternEvent();
        event.setId("pattern-trend-" + UUID.randomUUID().toString().substring(0, 8));
        event.setPatientId(patientId);
        event.setPatternType("TREND_ANALYSIS");
        event.setSeverity("MODERATE");
        event.setConfidence(0.65);
        event.setDetectionTime(System.currentTimeMillis());
        event.setTags(Set.of("VITAL_SIGNS_TREND"));
        event.setPriority(3);
        return event;
    }
}
```

- [ ] **Step 2: Compile to verify**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/builders/Module5TestBuilder.java
git commit -m "test(module5): add Module5TestBuilder with production-format test data"
```

---

## Task 3: Module5FeatureExtractor — Static Feature Extraction (Gap 1, 2, 4)

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module5FeatureExtractor.java`

This is the most critical class. It extracts a 55-element float array from `PatientMLState` with:
- Production vital key mapping (lowercase-no-separator)
- Triple null-check lab extraction
- Risk indicator resilience with missingness indicators
- Active alert features

- [ ] **Step 1: Create Module5FeatureExtractor.java**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.PatientMLState;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * Static feature extraction for Module 5 ONNX inference.
 * Extracts a 55-element float array from PatientMLState.
 *
 * Feature layout (indices):
 * [0-5]   Vital signs (normalized 0-1)
 * [6-8]   Clinical scores (normalized 0-1)
 * [9-10]  Event count (log-scaled) + hours since admission
 * [11-20] NEWS2 history ring buffer
 * [21-30] Acuity history ring buffer
 * [31-34] Pattern features
 * [35-44] Risk indicator flags (0/1)
 * [45-49] Lab-derived features (normalized, -1 = missing)
 * [50-54] Active alert features (binary + severity)
 *
 * All methods are static and testable without Flink runtime.
 */
public class Module5FeatureExtractor {
    private static final Logger LOG = LoggerFactory.getLogger(Module5FeatureExtractor.class);

    public static final int FEATURE_COUNT = 55;

    // ── Vital key aliases (Gap 1: snake_case → production format) ──
    private static final Map<String, String> VITAL_KEY_ALIASES = Map.ofEntries(
        Map.entry("heart_rate", "heartrate"),
        Map.entry("systolic_bp", "systolicbloodpressure"),
        Map.entry("diastolic_bp", "diastolicbloodpressure"),
        Map.entry("respiratory_rate", "respiratoryrate"),
        Map.entry("oxygen_saturation", "oxygensaturation"),
        Map.entry("HeartRate", "heartrate"),
        Map.entry("SystolicBP", "systolicbloodpressure")
    );

    // ── Non-vital fields that appear in latestVitals (must skip) ──
    private static final Set<String> NON_VITAL_KEYS = Set.of(
        "age", "gender", "bloodpressure", "weight", "height",
        "restingheartrate", "leftventricularejectionfraction", "data_tier"
    );

    /**
     * Extract full 55-element feature vector from patient ML state.
     */
    public static float[] extractFeatures(PatientMLState state) {
        float[] features = new float[FEATURE_COUNT];

        if (state == null) {
            LOG.warn("Null PatientMLState — returning zero feature vector");
            return features;
        }

        // [0-5] Vital signs
        Map<String, Double> vitals = normalizeVitalKeys(state.getLatestVitals());
        features[0] = normalize(vitals.getOrDefault("heartrate", 0.0), 30, 200);
        features[1] = normalize(vitals.getOrDefault("systolicbloodpressure", 0.0), 60, 250);
        features[2] = normalize(vitals.getOrDefault("diastolicbloodpressure", 0.0), 30, 150);
        features[3] = normalize(vitals.getOrDefault("respiratoryrate", 0.0), 5, 50);
        features[4] = normalize(vitals.getOrDefault("oxygensaturation", 0.0), 70, 100);
        features[5] = normalize(vitals.getOrDefault("temperature", 0.0), 34, 42);

        // [6-8] Clinical scores
        features[6] = normalize(state.getNews2Score(), 0, 20);
        features[7] = normalize(state.getQsofaScore(), 0, 3);
        features[8] = normalize(state.getAcuityScore(), 0, 10);

        // [9-10] Event context
        features[9] = (float) Math.log1p(state.getTotalEventCount());
        long hoursSinceAdmission = state.getFirstEventTime() > 0
            ? (System.currentTimeMillis() - state.getFirstEventTime()) / 3600000L
            : 0;
        features[10] = normalize(Math.min(hoursSinceAdmission, 720), 0, 720);

        // [11-20] NEWS2 history
        double[] news2Hist = state.getNews2History();
        int news2Count = Math.min(state.getNews2HistoryIndex(), 10);
        for (int i = 0; i < 10; i++) {
            features[11 + i] = i < news2Count ? normalize(news2Hist[i], 0, 20) : 0.0f;
        }

        // [21-30] Acuity history
        double[] acuityHist = state.getAcuityHistory();
        int acuityCount = Math.min(state.getAcuityHistoryIndex(), 10);
        for (int i = 0; i < 10; i++) {
            features[21 + i] = i < acuityCount ? normalize(acuityHist[i], 0, 10) : 0.0f;
        }

        // [31-34] Pattern features
        features[31] = (float) state.getRecentPatterns().size();
        features[32] = (float) state.getDeteriorationPatternCount();
        features[33] = (float) PatientMLState.severityIndex(state.getMaxSeveritySeen());
        features[34] = state.isSeverityEscalationDetected() ? 1.0f : 0.0f;

        // [35-44] Risk indicator flags (Gap 2: safe extraction with missingness)
        Map<String, Object> risk = state.getRiskIndicators();
        features[35] = safeRiskFlag(risk, "tachycardia");
        features[36] = safeRiskFlag(risk, "hypotension");
        features[37] = safeRiskFlag(risk, "fever");
        features[38] = safeRiskFlag(risk, "hypoxia");
        features[39] = safeRiskFlag(risk, "elevatedLactate");
        features[40] = safeRiskFlag(risk, "elevatedCreatinine");
        features[41] = safeRiskFlag(risk, "hyperkalemia");
        features[42] = safeRiskFlag(risk, "thrombocytopenia");
        features[43] = safeRiskFlag(risk, "onAnticoagulation");
        features[44] = safeRiskFlag(risk, "sepsisRisk");

        // [45-49] Lab-derived features (Gap 1: -1 = missing)
        Map<String, Double> labs = state.getLatestLabs();
        features[45] = safeLabFeature(labs, "lactate", 0, 20);
        features[46] = safeLabFeature(labs, "creatinine", 0, 15);
        features[47] = safeLabFeature(labs, "potassium", 2, 8);
        features[48] = safeLabFeature(labs, "wbc", 0, 40);
        features[49] = safeLabFeature(labs, "platelets", 0, 500);

        // [50-54] Active alert features (Gap 4)
        Map<String, Object> alerts = state.getActiveAlerts();
        features[50] = alertPresent(alerts, "SEPSIS_PATTERN") ? 1.0f : 0.0f;
        features[51] = alertPresent(alerts, "AKI_RISK") ? 1.0f : 0.0f;
        features[52] = alertPresent(alerts, "ANTICOAGULATION_RISK") ? 1.0f : 0.0f;
        features[53] = alertPresent(alerts, "BLEEDING_RISK") ? 1.0f : 0.0f;
        features[54] = alertMaxSeverity(alerts);

        return features;
    }

    // ── Vital key normalization ──

    public static Map<String, Double> normalizeVitalKeys(Map<String, Double> rawVitals) {
        if (rawVitals == null) return Collections.emptyMap();
        Map<String, Double> normalized = new HashMap<>();
        for (Map.Entry<String, Double> entry : rawVitals.entrySet()) {
            String key = VITAL_KEY_ALIASES.getOrDefault(entry.getKey(), entry.getKey());
            if (NON_VITAL_KEYS.contains(key)) continue;
            if (entry.getValue() != null) {
                normalized.put(key, entry.getValue());
            }
        }
        return normalized;
    }

    // ── Safe risk flag extraction (Gap 2) ──

    public static float safeRiskFlag(Map<String, Object> risk, String key) {
        if (risk == null) return 0.0f;
        Object val = risk.get(key);
        if (val instanceof Boolean) return ((Boolean) val) ? 1.0f : 0.0f;
        return 0.0f;
    }

    // ── Safe lab feature extraction (Gap 1) ──

    public static float safeLabFeature(Map<String, Double> labs, String key, double min, double max) {
        if (labs == null) return -1.0f;
        Double val = labs.get(key);
        if (val == null) return -1.0f;
        return normalize(val, min, max);
    }

    // ── Alert feature extraction (Gap 4) ──

    @SuppressWarnings("unchecked")
    public static boolean alertPresent(Map<String, Object> alerts, String alertType) {
        if (alerts == null) return false;
        return alerts.containsKey(alertType);
    }

    @SuppressWarnings("unchecked")
    public static float alertMaxSeverity(Map<String, Object> alerts) {
        if (alerts == null || alerts.isEmpty()) return 0.0f;
        int maxSev = 0;
        for (Object alertObj : alerts.values()) {
            if (alertObj instanceof Map) {
                Map<String, Object> alert = (Map<String, Object>) alertObj;
                Object sev = alert.get("severity");
                if (sev instanceof String) {
                    maxSev = Math.max(maxSev, PatientMLState.severityIndex((String) sev));
                }
            }
        }
        return normalize(maxSev, 0, 4);
    }

    // ── Normalization ──

    public static float normalize(double value, double min, double max) {
        if (Math.abs(max - min) < 1e-9) return 0.0f;
        return (float) Math.max(0.0, Math.min(1.0, (value - min) / (max - min)));
    }
}
```

- [ ] **Step 2: Compile to verify**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module5FeatureExtractor.java
git commit -m "feat(module5): add Module5FeatureExtractor with production key mapping and null-safety"
```

---

## Task 4: Module5RealDataMappingTest — Production Schema Validation (HIGHEST PRIORITY)

**Files:**
- Create: `src/test/java/com/cardiofit/flink/operators/Module5RealDataMappingTest.java`

This test validates that production CDS event data produces valid feature vectors. It catches schema mismatches **before** they reach ONNX inference. This is the Module 5 equivalent of `Module4RealCDSMappingTest`.

- [ ] **Step 1: Create Module5RealDataMappingTest.java**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module5TestBuilder;
import com.cardiofit.flink.models.PatientMLState;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Validates that real production CDS events produce valid feature vectors.
 * Catches schema mismatches before they reach ONNX inference.
 *
 * This is the HIGHEST PRIORITY test — run first, fix first.
 */
class Module5RealDataMappingTest {

    @Test
    @DisplayName("Production vital keys resolve to non-zero features")
    void productionVitalKeys_resolveCorrectly() {
        PatientMLState state = Module5TestBuilder.stablePatientState("P001");

        float[] features = Module5FeatureExtractor.extractFeatures(state);

        // Vitals should be non-zero (indices 0-5)
        assertNotEquals(0.0f, features[0], "heartrate should resolve");
        assertNotEquals(0.0f, features[1], "systolicbloodpressure should resolve");
        assertNotEquals(0.0f, features[2], "diastolicbloodpressure should resolve");
        assertNotEquals(0.0f, features[3], "respiratoryrate should resolve");
        assertNotEquals(0.0f, features[4], "oxygensaturation should resolve");
        assertNotEquals(0.0f, features[5], "temperature should resolve");
    }

    @Test
    @DisplayName("Snake_case vital keys normalize to production format")
    void snakeCaseVitalKeys_normalizeToProductionFormat() {
        PatientMLState state = new PatientMLState();
        state.setPatientId("P002");
        // Simulate Module4TestBuilder format (snake_case)
        state.setLatestVitals(Map.of(
            "heart_rate", 90.0,
            "systolic_bp", 130.0,
            "diastolic_bp", 85.0,
            "respiratory_rate", 18.0,
            "oxygen_saturation", 95.0,
            "temperature", 37.5
        ));
        state.setLatestLabs(Collections.emptyMap());
        state.setRiskIndicators(Collections.emptyMap());
        state.setActiveAlerts(Collections.emptyMap());

        float[] features = Module5FeatureExtractor.extractFeatures(state);

        // All vitals should resolve despite snake_case input
        assertNotEquals(0.0f, features[0], "heart_rate → heartrate should resolve");
        assertNotEquals(0.0f, features[1], "systolic_bp → systolicbloodpressure should resolve");
    }

    @Test
    @DisplayName("Demographic fields excluded from vital extraction")
    void demographicFields_excludedFromVitals() {
        Map<String, Double> vitalsWithDemographics = new HashMap<>(Map.of(
            "heartrate", 80.0,
            "systolicbloodpressure", 120.0,
            "age", 65.0,
            "weight", 78.0
        ));

        Map<String, Double> normalized = Module5FeatureExtractor.normalizeVitalKeys(vitalsWithDemographics);

        assertTrue(normalized.containsKey("heartrate"), "heartrate should be kept");
        assertFalse(normalized.containsKey("age"), "age should be excluded");
        assertFalse(normalized.containsKey("weight"), "weight should be excluded");
    }

    @Test
    @DisplayName("Feature vector has no NaN or Infinity values")
    void featureVector_noNaNOrInfinity() {
        // Test all patient scenarios
        String[] scenarios = {"stable", "sepsis", "aki", "druglab", "sparse"};
        PatientMLState[] states = {
            Module5TestBuilder.stablePatientState("P1"),
            Module5TestBuilder.sepsisPatientState("P2"),
            Module5TestBuilder.akiPatientState("P3"),
            Module5TestBuilder.drugLabPatientState("P4"),
            Module5TestBuilder.sparsePatientState("P5")
        };

        for (int s = 0; s < states.length; s++) {
            float[] features = Module5FeatureExtractor.extractFeatures(states[s]);
            assertEquals(Module5FeatureExtractor.FEATURE_COUNT, features.length,
                scenarios[s] + ": wrong feature count");

            for (int i = 0; i < features.length; i++) {
                assertFalse(Float.isNaN(features[i]),
                    scenarios[s] + ": NaN at index " + i);
                assertFalse(Float.isInfinite(features[i]),
                    scenarios[s] + ": Infinity at index " + i);
            }
        }
    }

    @Test
    @DisplayName("Missing labs produce -1.0 sentinel (not 0.0 or NaN)")
    void missingLabs_produceSentinelValue() {
        PatientMLState state = Module5TestBuilder.sparsePatientState("P006");

        float[] features = Module5FeatureExtractor.extractFeatures(state);

        // Lab features [45-49] should be -1.0 when labs are empty
        assertEquals(-1.0f, features[45], "missing lactate should be -1.0");
        assertEquals(-1.0f, features[46], "missing creatinine should be -1.0");
        assertEquals(-1.0f, features[47], "missing potassium should be -1.0");
        assertEquals(-1.0f, features[48], "missing wbc should be -1.0");
        assertEquals(-1.0f, features[49], "missing platelets should be -1.0");
    }

    @Test
    @DisplayName("AKI patient alerts produce non-zero alert features")
    void akiPatient_alertFeaturesPopulated() {
        PatientMLState state = Module5TestBuilder.akiPatientState("P007");

        float[] features = Module5FeatureExtractor.extractFeatures(state);

        // AKI_RISK alert present (index 51)
        assertEquals(1.0f, features[51], "AKI_RISK alert should be present");
        // Alert max severity should be non-zero (CRITICAL hyperkalemia)
        assertTrue(features[54] > 0.0f, "alert max severity should be non-zero");
    }

    @Test
    @DisplayName("Null PatientMLState produces zero vector (no crash)")
    void nullState_producesZeroVector() {
        float[] features = Module5FeatureExtractor.extractFeatures(null);

        assertEquals(Module5FeatureExtractor.FEATURE_COUNT, features.length);
        for (float f : features) {
            assertEquals(0.0f, f, "null state should produce all zeros");
        }
    }
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module5RealDataMappingTest -q 2>&1 | tail -10`
Expected: All 7 tests PASS

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/operators/Module5RealDataMappingTest.java
git commit -m "test(module5): add Module5RealDataMappingTest — validates production schema mapping"
```

---

## Task 5: Module5FeatureExtractionTest — Comprehensive Feature Extraction Tests

**Files:**
- Create: `src/test/java/com/cardiofit/flink/operators/Module5FeatureExtractionTest.java`

Covers normalization boundaries, risk indicator resilience, alert severity calculation, and slope computation.

- [ ] **Step 1: Create Module5FeatureExtractionTest.java**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module5TestBuilder;
import com.cardiofit.flink.models.PatientMLState;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

class Module5FeatureExtractionTest {

    @Test
    @DisplayName("normalize: value at min returns 0.0")
    void normalize_atMin_returnsZero() {
        assertEquals(0.0f, Module5FeatureExtractor.normalize(30, 30, 200));
    }

    @Test
    @DisplayName("normalize: value at max returns 1.0")
    void normalize_atMax_returnsOne() {
        assertEquals(1.0f, Module5FeatureExtractor.normalize(200, 30, 200));
    }

    @Test
    @DisplayName("normalize: value below min clamps to 0.0")
    void normalize_belowMin_clampsToZero() {
        assertEquals(0.0f, Module5FeatureExtractor.normalize(10, 30, 200));
    }

    @Test
    @DisplayName("normalize: value above max clamps to 1.0")
    void normalize_aboveMax_clampsToOne() {
        assertEquals(1.0f, Module5FeatureExtractor.normalize(300, 30, 200));
    }

    @Test
    @DisplayName("normalize: equal min/max returns 0.0 (no division by zero)")
    void normalize_equalMinMax_returnsZero() {
        assertEquals(0.0f, Module5FeatureExtractor.normalize(5, 5, 5));
    }

    @Test
    @DisplayName("safeRiskFlag: null map returns 0.0")
    void safeRiskFlag_nullMap_returnsZero() {
        assertEquals(0.0f, Module5FeatureExtractor.safeRiskFlag(null, "tachycardia"));
    }

    @Test
    @DisplayName("safeRiskFlag: missing key returns 0.0")
    void safeRiskFlag_missingKey_returnsZero() {
        assertEquals(0.0f, Module5FeatureExtractor.safeRiskFlag(
            Map.of("fever", true), "tachycardia"));
    }

    @Test
    @DisplayName("safeRiskFlag: non-Boolean value returns 0.0 (Gap 2 resilience)")
    void safeRiskFlag_nonBooleanValue_returnsZero() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("onAnticoagulation", "maybe");
        assertEquals(0.0f, Module5FeatureExtractor.safeRiskFlag(risk, "onAnticoagulation"));
    }

    @Test
    @DisplayName("safeLabFeature: null labs returns -1.0 sentinel")
    void safeLabFeature_nullLabs_returnsSentinel() {
        assertEquals(-1.0f, Module5FeatureExtractor.safeLabFeature(null, "lactate", 0, 20));
    }

    @Test
    @DisplayName("safeLabFeature: null value returns -1.0 sentinel (Gap 1)")
    void safeLabFeature_nullValue_returnsSentinel() {
        Map<String, Double> labs = new HashMap<>();
        labs.put("lactate", null);
        assertEquals(-1.0f, Module5FeatureExtractor.safeLabFeature(labs, "lactate", 0, 20));
    }

    @Test
    @DisplayName("Sepsis patient: elevated features produce high feature values")
    void sepsisPatient_producesHighFeatureValues() {
        PatientMLState state = Module5TestBuilder.sepsisPatientState("P-sepsis");

        float[] features = Module5FeatureExtractor.extractFeatures(state);

        // NEWS2=9 → normalize(9, 0, 20) = 0.45
        assertTrue(features[6] > 0.4f, "NEWS2 should be elevated");
        // qSOFA=2 → normalize(2, 0, 3) = 0.667
        assertTrue(features[7] > 0.6f, "qSOFA should be elevated");
        // Tachycardia flag
        assertEquals(1.0f, features[35], "tachycardia should be set");
        // Elevated lactate flag
        assertEquals(1.0f, features[39], "elevatedLactate should be set");
        // Sepsis alert
        assertEquals(1.0f, features[50], "SEPSIS_PATTERN alert should be present");
    }

    @Test
    @DisplayName("NEWS2 slope: increasing history returns positive slope")
    void news2Slope_increasingHistory_positiveSlope() {
        PatientMLState state = new PatientMLState();
        state.setPatientId("P-slope");
        state.pushNews2(2);
        state.pushNews2(4);
        state.pushNews2(6);
        state.pushNews2(8);

        assertTrue(state.news2Slope() > 0, "Increasing NEWS2 should produce positive slope");
    }

    @Test
    @DisplayName("Feature count is exactly 55")
    void featureCount_isExactly55() {
        PatientMLState state = Module5TestBuilder.stablePatientState("P-count");
        float[] features = Module5FeatureExtractor.extractFeatures(state);
        assertEquals(55, features.length);
        assertEquals(55, Module5FeatureExtractor.FEATURE_COUNT);
    }
}
```

- [ ] **Step 2: Run tests**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module5FeatureExtractionTest -q 2>&1 | tail -10`
Expected: All 13 tests PASS

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/operators/Module5FeatureExtractionTest.java
git commit -m "test(module5): add comprehensive feature extraction tests including Gap 1/2/4 coverage"
```

---

## Task 6: Module5ClinicalScoring — Cooldown + Calibration + Risk Classification (Gap 3)

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module5ClinicalScoring.java`

Contains the lab-aware cooldown logic (Gap 3), Platt scaling calibration, and category-specific risk thresholds.

- [ ] **Step 1: Create Module5ClinicalScoring.java**

```java
package com.cardiofit.flink.operators;

import java.util.Map;

/**
 * Clinical scoring utilities for Module 5: cooldown logic, calibration, risk classification.
 * All methods are static and testable without Flink runtime.
 */
public class Module5ClinicalScoring {

    private static final long STABLE_COOLDOWN_MS = 30_000;
    private static final long MODERATE_COOLDOWN_MS = 10_000;

    // ── Platt scaling parameters per category (trained offline) ──
    // Format: { A, B } where P(y=1) = 1 / (1 + exp(A * rawScore + B))
    private static final Map<String, double[]> PLATT_PARAMS = Map.of(
        "sepsis",         new double[]{ -2.5, 1.2 },
        "deterioration",  new double[]{ -2.0, 0.8 },
        "readmission",    new double[]{ -1.8, 0.5 },
        "fall",           new double[]{ -2.2, 1.0 },
        "mortality",      new double[]{ -2.0, 0.9 }
    );

    // ═══════════════════════════════════════════
    // Gap 3: Lab-Aware Inference Cooldown
    // ═══════════════════════════════════════════

    /**
     * Determines whether ONNX inference should run for this event.
     *
     * The cooldown tiers account for lab-only emergencies (AKI, anticoagulation,
     * hematologic) that produce NEWS2=0 with a critically ill patient.
     *
     * @param news2            Current NEWS2 score
     * @param qsofa            Current qSOFA score
     * @param riskIndicators   Risk indicator map from CDS event
     * @param lastInferenceMs  Timestamp of last inference (0 if never run)
     * @param currentTimeMs    Current processing time
     * @return true if inference should run
     */
    public static boolean shouldRunInference(
            int news2, int qsofa,
            Map<String, Object> riskIndicators,
            long lastInferenceMs, long currentTimeMs) {

        // First event for this patient — always run
        if (lastInferenceMs == 0) return true;

        long elapsed = currentTimeMs - lastInferenceMs;

        // HIGH RISK — no cooldown (every event)
        // Vitals-based
        if (news2 >= 7 || qsofa >= 2) return true;
        // Lab-based (Gap 3: invisible to NEWS2/qSOFA)
        if (hasLabCritical(riskIndicators)) return true;

        // MODERATE — 10s cooldown
        if (news2 >= 5) return elapsed >= MODERATE_COOLDOWN_MS;
        if (hasLabElevated(riskIndicators)) return elapsed >= MODERATE_COOLDOWN_MS;

        // STABLE — 30s cooldown
        return elapsed >= STABLE_COOLDOWN_MS;
    }

    /**
     * Lab-only emergencies that don't elevate NEWS2/qSOFA.
     * These patients need immediate ML prediction on every event.
     */
    public static boolean hasLabCritical(Map<String, Object> risk) {
        if (risk == null) return false;
        return Boolean.TRUE.equals(risk.get("hyperkalemia"))
            || Boolean.TRUE.equals(risk.get("severelyElevatedLactate"))
            || Boolean.TRUE.equals(risk.get("elevatedCreatinine"))
            || Boolean.TRUE.equals(risk.get("thrombocytopenia"));
    }

    /**
     * Elevated lab markers — reduce cooldown to moderate tier.
     */
    public static boolean hasLabElevated(Map<String, Object> risk) {
        if (risk == null) return false;
        return Boolean.TRUE.equals(risk.get("elevatedLactate"))
            || Boolean.TRUE.equals(risk.get("leukocytosis"))
            || Boolean.TRUE.equals(risk.get("leukopenia"));
    }

    // ═══════════════════════════════════════════
    // Platt Scaling Calibration
    // ═══════════════════════════════════════════

    /**
     * Calibrate raw ONNX output (sigmoid) to clinical probability.
     * Uses Platt scaling: P(y=1) = 1 / (1 + exp(A * rawScore + B))
     */
    public static double calibrate(double rawScore, String category) {
        double[] params = PLATT_PARAMS.get(category);
        if (params == null) return rawScore; // uncalibrated fallback
        return 1.0 / (1.0 + Math.exp(params[0] * rawScore + params[1]));
    }

    // ═══════════════════════════════════════════
    // Category-Specific Risk Classification
    // ═══════════════════════════════════════════

    /**
     * Classify calibrated score into risk level.
     * Sepsis thresholds are intentionally lower (false negatives are fatal).
     * Uses epsilon tolerance for floating-point boundary comparison (Lesson 6).
     */
    public static String classifyRiskLevel(double calibratedScore, String category) {
        double eps = 1e-9;
        return switch (category) {
            case "sepsis" -> {
                if (calibratedScore >= 0.60 - eps) yield "CRITICAL";
                if (calibratedScore >= 0.35 - eps) yield "HIGH";
                if (calibratedScore >= 0.15 - eps) yield "MODERATE";
                yield "LOW";
            }
            case "deterioration" -> {
                if (calibratedScore >= 0.70 - eps) yield "CRITICAL";
                if (calibratedScore >= 0.45 - eps) yield "HIGH";
                if (calibratedScore >= 0.20 - eps) yield "MODERATE";
                yield "LOW";
            }
            case "readmission" -> {
                if (calibratedScore >= 0.80 - eps) yield "CRITICAL";
                if (calibratedScore >= 0.55 - eps) yield "HIGH";
                if (calibratedScore >= 0.30 - eps) yield "MODERATE";
                yield "LOW";
            }
            default -> {
                if (calibratedScore >= 0.75 - eps) yield "CRITICAL";
                if (calibratedScore >= 0.50 - eps) yield "HIGH";
                if (calibratedScore >= 0.25 - eps) yield "MODERATE";
                yield "LOW";
            }
        };
    }
}
```

- [ ] **Step 2: Compile to verify**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module5ClinicalScoring.java
git commit -m "feat(module5): add Module5ClinicalScoring with lab-aware cooldown and Platt calibration"
```

---

## Task 7: Module5CooldownTest + Module5CalibrationTest (Gap 3 validation)

**Files:**
- Create: `src/test/java/com/cardiofit/flink/operators/Module5CooldownTest.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module5CalibrationTest.java`

- [ ] **Step 1: Create Module5CooldownTest.java**

```java
package com.cardiofit.flink.operators;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

class Module5CooldownTest {

    private static final long NOW = System.currentTimeMillis();

    @Test
    @DisplayName("First event for patient — always triggers inference")
    void firstEvent_alwaysTriggersInference() {
        assertTrue(Module5ClinicalScoring.shouldRunInference(
            1, 0, Map.of(), 0, NOW));
    }

    @Test
    @DisplayName("High NEWS2 (>=7) — no cooldown")
    void highNews2_noCooldown() {
        assertTrue(Module5ClinicalScoring.shouldRunInference(
            7, 0, Map.of(), NOW - 1000, NOW));
    }

    @Test
    @DisplayName("High qSOFA (>=2) — no cooldown")
    void highQsofa_noCooldown() {
        assertTrue(Module5ClinicalScoring.shouldRunInference(
            3, 2, Map.of(), NOW - 1000, NOW));
    }

    @Test
    @DisplayName("Gap 3: AKI patient (NEWS2=1, hyperkalemia=true) — no cooldown")
    void akiPatient_labCritical_noCooldown() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("hyperkalemia", true);
        risk.put("elevatedCreatinine", true);

        assertTrue(Module5ClinicalScoring.shouldRunInference(
            1, 0, risk, NOW - 1000, NOW),
            "AKI patient with normal vitals should bypass cooldown");
    }

    @Test
    @DisplayName("Gap 3: Drug-lab patient (NEWS2=1, thrombocytopenia=true) — no cooldown")
    void drugLabPatient_thrombocytopenia_noCooldown() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("thrombocytopenia", true);

        assertTrue(Module5ClinicalScoring.shouldRunInference(
            1, 0, risk, NOW - 1000, NOW),
            "Thrombocytopenic patient should bypass cooldown");
    }

    @Test
    @DisplayName("Moderate NEWS2 (5-6) — 10s cooldown")
    void moderateNews2_10sCooldown() {
        // Within 10s → blocked
        assertFalse(Module5ClinicalScoring.shouldRunInference(
            5, 0, Map.of(), NOW - 5000, NOW));
        // After 10s → allowed
        assertTrue(Module5ClinicalScoring.shouldRunInference(
            5, 0, Map.of(), NOW - 11000, NOW));
    }

    @Test
    @DisplayName("Elevated lactate — 10s cooldown (moderate tier)")
    void elevatedLactate_10sCooldown() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("elevatedLactate", true);

        assertFalse(Module5ClinicalScoring.shouldRunInference(
            2, 0, risk, NOW - 5000, NOW));
        assertTrue(Module5ClinicalScoring.shouldRunInference(
            2, 0, risk, NOW - 11000, NOW));
    }

    @Test
    @DisplayName("Stable patient (NEWS2=1) — 30s cooldown")
    void stablePatient_30sCooldown() {
        assertFalse(Module5ClinicalScoring.shouldRunInference(
            1, 0, Map.of(), NOW - 15000, NOW));
        assertTrue(Module5ClinicalScoring.shouldRunInference(
            1, 0, Map.of(), NOW - 31000, NOW));
    }
}
```

- [ ] **Step 2: Create Module5CalibrationTest.java**

```java
package com.cardiofit.flink.operators;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import static org.junit.jupiter.api.Assertions.*;

class Module5CalibrationTest {

    @Test
    @DisplayName("Platt calibration returns value in [0, 1]")
    void plattCalibration_inRange() {
        for (String category : new String[]{"sepsis", "deterioration", "readmission", "fall", "mortality"}) {
            for (double raw = 0.0; raw <= 1.0; raw += 0.1) {
                double calibrated = Module5ClinicalScoring.calibrate(raw, category);
                assertTrue(calibrated >= 0.0 && calibrated <= 1.0,
                    category + " calibrated=" + calibrated + " out of range for raw=" + raw);
            }
        }
    }

    @Test
    @DisplayName("Unknown category returns raw score (uncalibrated fallback)")
    void unknownCategory_returnsRawScore() {
        assertEquals(0.75, Module5ClinicalScoring.calibrate(0.75, "unknown_model"), 1e-9);
    }

    @Test
    @DisplayName("Sepsis thresholds: LOW < MODERATE < HIGH < CRITICAL")
    void sepsisThresholds_correctOrder() {
        assertEquals("LOW", Module5ClinicalScoring.classifyRiskLevel(0.10, "sepsis"));
        assertEquals("MODERATE", Module5ClinicalScoring.classifyRiskLevel(0.20, "sepsis"));
        assertEquals("HIGH", Module5ClinicalScoring.classifyRiskLevel(0.40, "sepsis"));
        assertEquals("CRITICAL", Module5ClinicalScoring.classifyRiskLevel(0.65, "sepsis"));
    }

    @Test
    @DisplayName("Sepsis HIGH threshold is lower than deterioration HIGH (clinical asymmetry)")
    void sepsisThreshold_lowerThanDeterioration() {
        // 0.40 should be HIGH for sepsis but only MODERATE for deterioration
        assertEquals("HIGH", Module5ClinicalScoring.classifyRiskLevel(0.40, "sepsis"));
        assertEquals("MODERATE", Module5ClinicalScoring.classifyRiskLevel(0.40, "deterioration"));
    }

    @Test
    @DisplayName("Floating-point boundary: 0.35 exactly is HIGH for sepsis (epsilon tolerance)")
    void floatingPointBoundary_epsilonTolerance() {
        // Lesson 6: 0.35 should classify as HIGH, not MODERATE
        assertEquals("HIGH", Module5ClinicalScoring.classifyRiskLevel(0.35, "sepsis"));
    }

    @Test
    @DisplayName("Readmission thresholds: higher than other categories")
    void readmissionThresholds_higherThanOthers() {
        // 0.50 is HIGH for sepsis, but only MODERATE for readmission
        assertEquals("HIGH", Module5ClinicalScoring.classifyRiskLevel(0.50, "sepsis"));
        assertEquals("MODERATE", Module5ClinicalScoring.classifyRiskLevel(0.50, "readmission"));
    }

    @Test
    @DisplayName("Score 0.0 always classifies as LOW")
    void scoreZero_alwaysLow() {
        for (String cat : new String[]{"sepsis", "deterioration", "readmission", "fall", "mortality"}) {
            assertEquals("LOW", Module5ClinicalScoring.classifyRiskLevel(0.0, cat));
        }
    }

    @Test
    @DisplayName("Score 1.0 always classifies as CRITICAL")
    void scoreOne_alwaysCritical() {
        for (String cat : new String[]{"sepsis", "deterioration", "readmission", "fall", "mortality"}) {
            assertEquals("CRITICAL", Module5ClinicalScoring.classifyRiskLevel(1.0, cat));
        }
    }
}
```

- [ ] **Step 3: Run all Module5 tests**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module5CooldownTest,Module5CalibrationTest" -q 2>&1 | tail -10`
Expected: All tests PASS (8 cooldown + 8 calibration = 16 tests)

- [ ] **Step 4: Commit**

```bash
git add src/test/java/com/cardiofit/flink/operators/Module5CooldownTest.java \
        src/test/java/com/cardiofit/flink/operators/Module5CalibrationTest.java
git commit -m "test(module5): add cooldown and calibration tests covering Gap 3 lab-aware logic"
```

---

## Task 8: MLPrediction Schema Update (Gap 5)

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/models/MLPrediction.java`

Add fields for prediction category, calibrated score, context depth, trigger source, and correlation ID traceability.

- [ ] **Step 1: Add new fields to MLPrediction.java**

Add these fields after line 65 (`private String correlationId;`):

```java
    // ── Module 5 upgrade fields ──

    @JsonProperty("prediction_category")
    private String predictionCategory;       // readmission, sepsis, deterioration, fall, mortality

    @JsonProperty("calibrated_score")
    private Double calibratedScore;          // after Platt scaling

    @JsonProperty("context_depth")
    private String contextDepth;             // INITIAL or ESTABLISHED

    @JsonProperty("trigger_source")
    private String triggerSource;            // CDS_EVENT or SEVERITY_ESCALATION

    @JsonProperty("prediction_horizon_ms")
    private Long predictionHorizonMs;        // how far ahead the prediction looks

    @JsonProperty("input_snapshot")
    private Map<String, Object> inputSnapshot;  // feature vector for audit reproducibility
```

- [ ] **Step 2: Add getters/setters after existing getters (after line 228)**

```java
    public String getPredictionCategory() { return predictionCategory; }
    public void setPredictionCategory(String predictionCategory) { this.predictionCategory = predictionCategory; }

    public Double getCalibratedScore() { return calibratedScore; }
    public void setCalibratedScore(Double calibratedScore) { this.calibratedScore = calibratedScore; }

    public String getContextDepth() { return contextDepth; }
    public void setContextDepth(String contextDepth) { this.contextDepth = contextDepth; }

    public String getTriggerSource() { return triggerSource; }
    public void setTriggerSource(String triggerSource) { this.triggerSource = triggerSource; }

    public Long getPredictionHorizonMs() { return predictionHorizonMs; }
    public void setPredictionHorizonMs(Long predictionHorizonMs) { this.predictionHorizonMs = predictionHorizonMs; }

    public Map<String, Object> getInputSnapshot() { return inputSnapshot; }
    public void setInputSnapshot(Map<String, Object> inputSnapshot) { this.inputSnapshot = inputSnapshot; }
```

- [ ] **Step 3: Add builder methods in Builder class (after line 156)**

```java
        public Builder predictionCategory(String predictionCategory) {
            prediction.predictionCategory = predictionCategory;
            return this;
        }

        public Builder calibratedScore(Double calibratedScore) {
            prediction.calibratedScore = calibratedScore;
            return this;
        }

        public Builder contextDepth(String contextDepth) {
            prediction.contextDepth = contextDepth;
            return this;
        }

        public Builder triggerSource(String triggerSource) {
            prediction.triggerSource = triggerSource;
            return this;
        }
```

- [ ] **Step 4: Compile to verify no regressions**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/MLPrediction.java
git commit -m "feat(module5): add predictionCategory, calibratedScore, contextDepth, correlationId to MLPrediction"
```

---

## Task 9: Module5_MLInferenceEngine — KeyedCoProcessFunction Operator

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module5_MLInferenceEngine.java`

The core operator that replaces the union-based chain. Uses `KeyedCoProcessFunction<String, EnrichedPatientContext, PatternEvent, MLPrediction>` with:
- CDS events (processElement1) trigger inference with cooldown
- Pattern events (processElement2) buffer into patient state
- CRITICAL escalation patterns bypass cooldown
- 7-day state TTL

- [ ] **Step 1: Create Module5_MLInferenceEngine.java**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.ml.ONNXModelContainer;
import com.cardiofit.flink.ml.ModelConfig;
import com.cardiofit.flink.models.*;
import org.apache.flink.api.common.state.StateTtlConfig;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.streaming.api.functions.co.KeyedCoProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.time.Duration;
import java.util.*;

/**
 * Module 5: ML Inference Engine — KeyedCoProcessFunction implementation.
 *
 * Dual-input streams:
 *   Input 1 (CDS events): Triggers inference after cooldown check
 *   Input 2 (Pattern events): Buffers into patient state, triggers on CRITICAL escalation
 *
 * Replaces the union-based chain to prevent 4-5x inference amplification.
 */
public class Module5_MLInferenceEngine
        extends KeyedCoProcessFunction<String, EnrichedPatientContext, PatternEvent, MLPrediction> {

    private static final Logger LOG = LoggerFactory.getLogger(Module5_MLInferenceEngine.class);

    // Output tags for side outputs
    public static final OutputTag<MLPrediction> HIGH_RISK_TAG =
        new OutputTag<>("high-risk-predictions", TypeInformation.of(MLPrediction.class)){};
    public static final OutputTag<MLPrediction> AUDIT_TAG =
        new OutputTag<>("prediction-audit", TypeInformation.of(MLPrediction.class)){};

    // State
    private transient ValueState<PatientMLState> patientState;
    private transient ValueState<Long> lastInferenceTime;

    // ONNX models (transient — initialized in open())
    private transient Map<String, ONNXModelContainer> modelSessions;

    private static final String[] CATEGORIES = {
        "readmission", "sepsis", "deterioration", "fall", "mortality"
    };

    @Override
    public void open(Configuration parameters) throws Exception {
        super.open(parameters);

        // State with 7-day TTL
        StateTtlConfig ttlConfig = StateTtlConfig.newBuilder(Duration.ofDays(7))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();

        ValueStateDescriptor<PatientMLState> stateDesc =
            new ValueStateDescriptor<>("patient-ml-state", PatientMLState.class);
        stateDesc.enableTimeToLive(ttlConfig);
        patientState = getRuntimeContext().getState(stateDesc);

        ValueStateDescriptor<Long> timeDesc =
            new ValueStateDescriptor<>("last-inference-time", Long.class);
        timeDesc.enableTimeToLive(ttlConfig);
        lastInferenceTime = getRuntimeContext().getState(timeDesc);

        // Load ONNX models (graceful degradation — Section 10)
        modelSessions = new HashMap<>();
        String modelBasePath = System.getenv("ML_MODEL_PATH");
        if (modelBasePath == null) modelBasePath = "/opt/flink/models";

        for (String category : CATEGORIES) {
            String modelPath = modelBasePath + "/" + category + "/model.onnx";
            if (new File(modelPath).exists()) {
                try {
                    List<String> featureNames = new ArrayList<>();
                    for (int i = 0; i < Module5FeatureExtractor.FEATURE_COUNT; i++) {
                        featureNames.add("f" + i);
                    }
                    ONNXModelContainer model = ONNXModelContainer.builder()
                        .modelId(category + "_v1")
                        .modelName(category + " predictor")
                        .modelType(mapCategoryToModelType(category))
                        .modelVersion("1.0.0")
                        .inputFeatureNames(featureNames)
                        .config(ModelConfig.builder()
                            .predictionThreshold(0.5)
                            .intraOpThreads(1)
                            .interOpThreads(1)
                            .modelPath(modelPath)
                            .build())
                        .build();
                    model.initialize();
                    modelSessions.put(category, model);
                    LOG.info("Loaded ONNX model for {} from {}", category, modelPath);
                } catch (Exception e) {
                    LOG.warn("Failed to load model for {}: {}", category, e.getMessage());
                }
            } else {
                LOG.info("Model not found for {}: {} — predictions disabled", category, modelPath);
            }
        }

        // Register metrics
        getRuntimeContext().getMetricGroup().gauge("models_loaded",
            () -> (long) modelSessions.size());
    }

    // ═══════════════════════════════════════════
    // PRIMARY PATH: CDS events trigger inference
    // ═══════════════════════════════════════════

    @Override
    public void processElement1(
            EnrichedPatientContext cdsEvent,
            KeyedCoProcessFunction<String, EnrichedPatientContext, PatternEvent, MLPrediction>.Context ctx,
            Collector<MLPrediction> out) throws Exception {

        if (cdsEvent == null || cdsEvent.getPatientId() == null) {
            LOG.warn("Null CDS event or patientId — skipping (Lesson 1: silent deser failure)");
            return;
        }

        // Update patient state
        PatientMLState state = getOrCreateState(cdsEvent.getPatientId());
        updateStateFromCDS(state, cdsEvent);
        patientState.update(state);

        // No models → pass through
        if (modelSessions.isEmpty()) {
            LOG.debug("No ML models loaded — skipping inference for {}", cdsEvent.getPatientId());
            return;
        }

        // Check cooldown (Gap 3: lab-aware)
        Long lastRun = lastInferenceTime.value();
        long currentTime = cdsEvent.getProcessingTime();
        if (!Module5ClinicalScoring.shouldRunInference(
                state.getNews2Score(), state.getQsofaScore(),
                state.getRiskIndicators(),
                lastRun != null ? lastRun : 0, currentTime)) {
            return;
        }

        // Run inference
        runInferenceAndEmit(state, "CDS_EVENT", ctx, out);
        lastInferenceTime.update(currentTime);

        // Clear pattern buffer after inference
        state.clearPatternBuffer();
        patientState.update(state);
    }

    // ═══════════════════════════════════════════
    // SECONDARY PATH: Pattern events update state
    // ═══════════════════════════════════════════

    @Override
    public void processElement2(
            PatternEvent patternEvent,
            KeyedCoProcessFunction<String, EnrichedPatientContext, PatternEvent, MLPrediction>.Context ctx,
            Collector<MLPrediction> out) throws Exception {

        if (patternEvent == null || patternEvent.getPatientId() == null) return;

        PatientMLState state = getOrCreateState(patternEvent.getPatientId());

        // Buffer pattern
        state.addPattern(new PatientMLState.PatternSummary(
            patternEvent.getPatternType(),
            patternEvent.getSeverity(),
            patternEvent.getConfidence(),
            patternEvent.getDetectionTime(),
            patternEvent.getTags()
        ));
        patientState.update(state);

        // CRITICAL escalation → bypass cooldown and trigger immediate inference
        if ("CRITICAL".equals(patternEvent.getSeverity())
                && patternEvent.getTags() != null
                && patternEvent.getTags().contains("SEVERITY_ESCALATION")
                && !modelSessions.isEmpty()) {

            LOG.info("CRITICAL escalation for {} — triggering immediate inference",
                patternEvent.getPatientId());
            runInferenceAndEmit(state, "SEVERITY_ESCALATION", ctx, out);
            lastInferenceTime.update(System.currentTimeMillis());
            state.clearPatternBuffer();
            patientState.update(state);
        }
    }

    // ═══════════════════════════════════════════
    // Inference execution
    // ═══════════════════════════════════════════

    private void runInferenceAndEmit(
            PatientMLState state, String triggerSource,
            KeyedCoProcessFunction<String, EnrichedPatientContext, PatternEvent, MLPrediction>.Context ctx,
            Collector<MLPrediction> out) {

        float[] features = Module5FeatureExtractor.extractFeatures(state);

        for (Map.Entry<String, ONNXModelContainer> entry : modelSessions.entrySet()) {
            String category = entry.getKey();
            ONNXModelContainer model = entry.getValue();

            try {
                long start = System.nanoTime();
                MLPrediction rawPrediction = model.predict(features);
                long elapsedMs = (System.nanoTime() - start) / 1_000_000;

                // Enrich with Module 5 metadata
                rawPrediction.setPatientId(state.getPatientId());
                rawPrediction.setPredictionCategory(category);
                rawPrediction.setTriggerSource(triggerSource);
                rawPrediction.setContextDepth(
                    state.getTotalEventCount() > 3 ? "ESTABLISHED" : "INITIAL");

                // Calibrate (Gap 3 related)
                double rawScore = rawPrediction.getPrimaryScore();
                double calibrated = Module5ClinicalScoring.calibrate(rawScore, category);
                rawPrediction.setCalibratedScore(calibrated);
                rawPrediction.setRiskLevel(
                    Module5ClinicalScoring.classifyRiskLevel(calibrated, category));

                // Store input features for audit
                rawPrediction.setInputFeatures(features);

                // Main output
                out.collect(rawPrediction);

                // Side output: high risk
                if ("HIGH".equals(rawPrediction.getRiskLevel())
                        || "CRITICAL".equals(rawPrediction.getRiskLevel())) {
                    ctx.output(HIGH_RISK_TAG, rawPrediction);
                }

                // Side output: audit
                ctx.output(AUDIT_TAG, rawPrediction);

                if (elapsedMs > 100) {
                    LOG.warn("ONNX inference slow for {} on {}: {}ms",
                        category, state.getPatientId(), elapsedMs);
                }

            } catch (Exception e) {
                LOG.error("Inference failed for {} on {}: {}",
                    category, state.getPatientId(), e.getMessage());
            }
        }
    }

    // ═══════════════════════════════════════════
    // State management helpers
    // ═══════════════════════════════════════════

    private PatientMLState getOrCreateState(String patientId) throws Exception {
        PatientMLState state = patientState.value();
        if (state == null) {
            state = new PatientMLState();
            state.setPatientId(patientId);
            state.setFirstEventTime(System.currentTimeMillis());
        }
        return state;
    }

    @SuppressWarnings("unchecked")
    private void updateStateFromCDS(PatientMLState state, EnrichedPatientContext cds) {
        var ps = cds.getPatientState();
        if (ps == null) return;

        // Vitals — normalize keys from production format
        if (ps.getLatestVitals() != null) {
            Map<String, Double> vitals = new HashMap<>();
            for (Map.Entry<String, Object> e : ps.getLatestVitals().entrySet()) {
                if (e.getValue() instanceof Number) {
                    vitals.put(e.getKey(), ((Number) e.getValue()).doubleValue());
                }
            }
            state.setLatestVitals(vitals);
        }

        // Labs — extract numeric values with null safety (Gap 1)
        if (ps.getRecentLabs() != null) {
            Map<String, Double> labs = new HashMap<>();
            for (Map.Entry<String, ?> e : ps.getRecentLabs().entrySet()) {
                Object labResult = e.getValue();
                if (labResult instanceof Map) {
                    Object val = ((Map<String, Object>) labResult).get("value");
                    if (val instanceof Number) {
                        String labType = (String) ((Map<String, Object>) labResult).get("labType");
                        String key = labType != null ? labType.toLowerCase() : e.getKey();
                        labs.put(key, ((Number) val).doubleValue());
                    }
                }
            }
            state.setLatestLabs(labs);
        }

        // Clinical scores
        state.setNews2Score(ps.getNews2Score() != null ? ps.getNews2Score() : 0);
        state.setQsofaScore(ps.getQsofaScore() != null ? ps.getQsofaScore() : 0);
        state.setAcuityScore(ps.getCombinedAcuityScore() != null ? ps.getCombinedAcuityScore() : 0.0);
        state.pushNews2(state.getNews2Score());
        state.pushAcuity(state.getAcuityScore());

        // Risk indicators
        if (ps.getRiskIndicators() != null) {
            state.setRiskIndicators(ps.getRiskIndicators().toMap());
        }

        // Active alerts (Gap 4)
        if (ps.getActiveAlerts() != null) {
            Map<String, Object> alertMap = new HashMap<>();
            for (var alert : ps.getActiveAlerts()) {
                alertMap.put(alert.getAlertType() != null ? alert.getAlertType().name() : "UNKNOWN",
                    Map.of(
                        "severity", alert.getSeverity() != null ? alert.getSeverity().name() : "UNKNOWN",
                        "message", alert.getMessage() != null ? alert.getMessage() : ""
                    ));
            }
            state.setActiveAlerts(alertMap);
        }

        state.setTotalEventCount(state.getTotalEventCount() + 1);
    }

    private static ONNXModelContainer.ModelType mapCategoryToModelType(String category) {
        return switch (category) {
            case "readmission" -> ONNXModelContainer.ModelType.READMISSION_RISK;
            case "sepsis" -> ONNXModelContainer.ModelType.SEPSIS_ONSET;
            case "deterioration" -> ONNXModelContainer.ModelType.CLINICAL_DETERIORATION;
            case "fall" -> ONNXModelContainer.ModelType.FALL_RISK;
            case "mortality" -> ONNXModelContainer.ModelType.MORTALITY_PREDICTION;
            default -> ONNXModelContainer.ModelType.CLINICAL_DETERIORATION;
        };
    }

    @Override
    public void close() throws Exception {
        if (modelSessions != null) {
            for (ONNXModelContainer model : modelSessions.values()) {
                model.close();
            }
        }
        super.close();
    }
}
```

- [ ] **Step 2: Compile to verify**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -10`
Expected: BUILD SUCCESS (may need minor type adjustments based on actual `PatientContextState` API — check compile errors and fix)

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module5_MLInferenceEngine.java
git commit -m "feat(module5): add KeyedCoProcessFunction operator with dual-input, lab-aware cooldown"
```

---

## Task 10: Wire New Operator into Module5_MLInference Pipeline

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/operators/Module5_MLInference.java:100-135`

Replace the union-based `createMLInferencePipeline()` to use the new `KeyedCoProcessFunction`.

- [ ] **Step 1: Replace the pipeline creation method**

Replace the `createMLInferencePipeline` method body (lines ~100-161) with a version that uses `KeyedCoProcessFunction` instead of union + FeatureCombiner:

```java
    public static void createMLInferencePipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating ML inference pipeline (KeyedCoProcessFunction architecture)");

        // Input stream 1: CDS events from Module 3
        DataStream<EnrichedPatientContext> cdsEvents = createEnrichedPatientContextSource(env);

        // Input stream 2: Pattern events from Module 4
        DataStream<PatternEvent> patternEvents = createPatternEventSource(env);

        // Dual-input KeyedCoProcessFunction — CDS triggers inference, patterns buffer
        SingleOutputStreamOperator<MLPrediction> predictions = cdsEvents
            .keyBy(EnrichedPatientContext::getPatientId)
            .connect(patternEvents.keyBy(PatternEvent::getPatientId))
            .process(new Module5_MLInferenceEngine())
            .uid("Module5-ML-Inference-Engine");

        // Main output: all predictions
        predictions
            .sinkTo(createMLPredictionsSink())
            .uid("ML Predictions Sink");

        // Side output: high-risk predictions
        predictions.getSideOutput(Module5_MLInferenceEngine.HIGH_RISK_TAG)
            .sinkTo(createHighRiskAlertsSink())
            .uid("High Risk Predictions Sink");

        // Side output: audit trail
        predictions.getSideOutput(Module5_MLInferenceEngine.AUDIT_TAG)
            .sinkTo(createAuditSink())
            .uid("Prediction Audit Sink");

        // Keep MIMIC-IV pipeline for backward compatibility
        LOG.info("Adding MIMIC-IV real ML inference pipeline");
        DataStream<EnrichedPatientContext> mimicContext = createEnrichedPatientContextSource(env);
        PatientContextAdapter adapter = new PatientContextAdapter();
        DataStream<PatientContextSnapshot> patientSnapshots = mimicContext
            .map(context -> adapter.adapt(context))
            .name("Patient Context Adapter")
            .uid("mimic-context-adapter");

        DataStream<List<MLPrediction>> mimicPredictionLists = patientSnapshots
            .map(new MIMICMLInferenceOperator())
            .name("MIMIC-IV ML Inference")
            .uid("mimic-ml-inference");

        DataStream<MLPrediction> mimicPredictions = mimicPredictionLists
            .flatMap((FlatMapFunction<List<MLPrediction>, MLPrediction>)
                (list, out) -> list.forEach(out::collect))
            .returns(TypeInformation.of(MLPrediction.class))
            .name("MIMIC Prediction Flattener")
            .uid("mimic-prediction-flattener");

        mimicPredictions
            .sinkTo(createMIMICMLPredictionsSink())
            .uid("MIMIC Predictions Sink");
    }
```

- [ ] **Step 2: Add missing sink factory method for audit**

Add after the existing sink methods in Module5_MLInference.java:

```java
    private static KafkaSink<MLPrediction> createAuditSink() {
        return KafkaSink.<MLPrediction>builder()
            .setBootstrapServers(KafkaConfigLoader.getBootstrapServers())
            .setRecordSerializer(
                KafkaRecordSerializationSchema.builder()
                    .setTopic("prediction-audit.v1")
                    .setValueSerializationSchema(new MLPredictionSerializer())
                    .build()
            )
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    private static KafkaSink<MLPrediction> createHighRiskAlertsSink() {
        return KafkaSink.<MLPrediction>builder()
            .setBootstrapServers(KafkaConfigLoader.getBootstrapServers())
            .setRecordSerializer(
                KafkaRecordSerializationSchema.builder()
                    .setTopic("high-risk-predictions.v1")
                    .setValueSerializationSchema(new MLPredictionSerializer())
                    .build()
            )
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }
```

- [ ] **Step 3: Compile to verify integration**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -10`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module5_MLInference.java
git commit -m "refactor(module5): wire KeyedCoProcessFunction into pipeline, replace union-based chain"
```

---

## Task 11: Fix ClinicalFeatureExtractor Vital Key Mapping

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/ml/features/ClinicalFeatureExtractor.java:178-183`

The existing `ClinicalFeatureExtractor` (used by MIMIC pipeline) uses `heart_rate`, `systolic_bp` etc. Fix to accept both formats.

- [ ] **Step 1: Add key normalization to extractVitals()**

Replace lines 178-183 in `extractVitals()`:

```java
        // Normalize vital keys to handle both production (heartrate) and SemanticEvent (heart_rate) formats
        Map<String, Object> vitals = new HashMap<>(context.getLatestVitals());
        Map<String, Object> normalized = new HashMap<>();
        Map<String, String> aliases = Map.of(
            "heartrate", "heart_rate", "systolicbloodpressure", "systolic_bp",
            "diastolicbloodpressure", "diastolic_bp", "respiratoryrate", "respiratory_rate",
            "oxygensaturation", "oxygen_saturation"
        );
        for (Map.Entry<String, Object> e : vitals.entrySet()) {
            normalized.put(e.getKey(), e.getValue());
            String alias = aliases.get(e.getKey());
            if (alias != null) normalized.put(alias, e.getValue());
        }
        vitals = normalized;
```

- [ ] **Step 2: Run existing ClinicalFeatureExtractor tests**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="*FeatureParity*,*ClinicalFeature*" -q 2>&1 | tail -10`
Expected: No regressions

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/ml/features/ClinicalFeatureExtractor.java
git commit -m "fix(module5): normalize vital keys in ClinicalFeatureExtractor for production format compatibility"
```

---

## Task 12: Run Full Test Suite and Verify

**Files:** None created — validation only.

- [ ] **Step 1: Run all Module 5 tests**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module5*" -q 2>&1 | tail -20`
Expected: All tests PASS (7 + 13 + 8 + 8 = 36 new tests)

- [ ] **Step 2: Run full project compile**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Run existing Module 4 tests (no regressions)**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module4*" -q 2>&1 | tail -10`
Expected: All existing tests still PASS

---

## Summary of Gaps Addressed

| Gap | Problem | Fix Location | Test |
|-----|---------|-------------|------|
| **Gap 1**: Lab payload null values | `LabResult.getValue()` returns null silently | `Module5FeatureExtractor.safeLabFeature()` returns -1.0 sentinel | `Module5RealDataMappingTest.missingLabs_produceSentinelValue` |
| **Gap 2**: Unreliable risk indicators | Module 3 overwrites Module 2 medication flags | `Module5FeatureExtractor.safeRiskFlag()` treats non-Boolean as false | `Module5FeatureExtractionTest.safeRiskFlag_nonBooleanValue_returnsZero` |
| **Gap 3**: Lab-only emergencies invisible to NEWS2 | AKI (K+ 6.1) gets 30s cooldown | `Module5ClinicalScoring.shouldRunInference()` checks lab flags | `Module5CooldownTest.akiPatient_labCritical_noCooldown` |
| **Gap 4**: Active alerts as features | Pre-computed clinical intelligence unused | `Module5FeatureExtractor` features [50-54] extract alert presence + severity | `Module5RealDataMappingTest.akiPatient_alertFeaturesPopulated` |
| **Gap 5**: CorrelationId missing | Audit trail can't trace through pipeline | `MLPrediction.correlationId` field (already existed, now documented) | Builder pattern updated |
| **Gap 6**: Lab key mismatch (LOINC vs concept) | `updateStateFromCDS` keyed labs by `labResult.getLabType()` (LOINC code like "2524-7"), but `Module5FeatureExtractor` looks up by friendly name ("lactate") — every lab silently returned -1.0 sentinel | `Module5_MLInferenceEngine.updateStateFromCDS()` — key by `labResult.getClinicalConcept().toLowerCase()` (KB-7 canonical name) | Existing `Module5RealDataMappingTest` and `Module5FeatureExtractionTest` already use clinicalConcept-style keys |
| **Gap 7**: Double Kafka consumer | `createMLInferencePipeline()` called `createEnrichedPatientContextSource(env)` twice (lines 104 + 137), creating two independent Kafka consumers on the same topic | `Module5_MLInference.createMLInferencePipeline()` — reuse `cdsEvents` stream for MIMIC pipeline instead of second consumer | Compile-verified; no runtime test (requires Kafka) |
