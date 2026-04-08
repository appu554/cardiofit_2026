# Module 6: Clinical Action Engine Upgrade — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade Module 6 from a basic PatternEvent → ComposedAlert egress router into a full clinical action engine that consumes CDS events (Module 3), pattern events (Module 4), and ML predictions (Module 5), classifies them into HALT/PAUSE/SOFT_FLAG/ROUTINE tiers, deduplicates across modules, manages alert lifecycles with SLA escalation timers, routes notifications, emits FHIR write-back requests, and produces a complete audit trail.

**Architecture:** Module 6 unifies four input streams into a single `ClinicalEvent` wrapper, classifies urgency via a static `ClinicalActionClassifier`, deduplicates across modules using `CrossModuleDeduplicator`, manages alert state via `AlertLifecycleManager` with Flink processing-time timers, and distributes output to six Kafka sinks via side-output tags. A `KeyedCoProcessFunction` handles physician acknowledgment feedback.

**Tech Stack:** Java 17, Flink 2.1.0, Jackson 2.17, Kafka (Confluent Cloud), JUnit 5

---

## Existing Code Inventory

**Files that exist and will be REPLACED/REWRITTEN:**
- `operators/Module6_AlertComposition.java` — current basic PatternEvent → ComposedAlert router (will be superseded by `Module6_ClinicalActionEngine.java`)

**Files that exist and will be KEPT AS-IS:**
- `operators/Module6_EgressRouting.java` — hybrid multi-sink egress (separate concern, not touched)
- `operators/Module6_AnalyticsEngine.java` — analytics (separate concern, not touched)
- `models/ComposedAlert.java` — kept for backward compat with egress routing
- `models/CDSEvent.java`, `models/MLPrediction.java`, `models/PatternEvent.java` — input models, read-only
- `models/PatientContextState.java`, `models/RiskIndicators.java` — used by classifier
- `utils/KafkaTopics.java` — will be modified to add new topics
- `utils/KafkaConfigLoader.java` — used as-is

**Existing test patterns to follow:**
- `builders/Module4TestBuilder.java` — factory methods for patient scenarios
- `builders/Module5TestBuilder.java` — `sepsisPatientState()`, `akiPatientState()`, `drugLabPatientState()`, `stablePatientState()`

---

## File Structure

```
backend/shared-infrastructure/flink-processing/src/
├── main/java/com/cardiofit/flink/
│   ├── models/
│   │   ├── ActionTier.java                    ← NEW: enum HALT, PAUSE, SOFT_FLAG, ROUTINE
│   │   ├── AlertState.java                    ← NEW: enum ACTIVE, ACKNOWLEDGED, ACTIONED, AUTO_RESOLVED, ESCALATED, RESOLVED
│   │   ├── ClinicalEvent.java                 ← NEW: unified wrapper for CDS/Pattern/ML inputs
│   │   ├── ClinicalAlert.java                 ← NEW: full alert entity with lifecycle
│   │   ├── ClinicalAction.java                ← NEW: output action record
│   │   ├── NotificationRequest.java           ← NEW: notification output contract
│   │   ├── AuditRecord.java                   ← NEW: audit trail record
│   │   ├── FhirWriteRequest.java              ← NEW: FHIR write-back request
│   │   ├── PatientAlertState.java             ← NEW: per-patient alert state
│   │   └── AlertAcknowledgment.java           ← NEW: acknowledgment input
│   ├── operators/
│   │   ├── Module6ActionClassifier.java        ← NEW: static HALT/PAUSE/SOFT_FLAG logic
│   │   ├── Module6CrossModuleDedup.java        ← NEW: cross-module deduplication
│   │   ├── Module6_ClinicalActionEngine.java   ← NEW: main KeyedProcessFunction operator
│   │   └── Module6_AlertComposition.java       ← EXISTING: kept for backward compat
│   ├── routing/
│   │   └── NotificationRouter.java             ← NEW: channel selection by tier
│   └── lifecycle/
│       └── AlertLifecycleManager.java          ← NEW: state machine + auto-resolution
├── test/java/com/cardiofit/flink/
│   ├── builders/
│   │   └── Module6TestBuilder.java             ← NEW: test data factory
│   └── operators/
│       ├── Module6ActionClassifierTest.java     ← NEW
│       ├── Module6DeduplicationTest.java        ← NEW
│       ├── Module6AlertLifecycleTest.java       ← NEW
│       ├── Module6NotificationRoutingTest.java  ← NEW
│       ├── Module6AlertFatigueTest.java         ← NEW
│       └── Module6AuditTrailTest.java           ← NEW
```

---

## Task 1: Enums — ActionTier and AlertState

**Files:**
- Create: `src/main/java/com/cardiofit/flink/models/ActionTier.java`
- Create: `src/main/java/com/cardiofit/flink/models/AlertState.java`

- [ ] **Step 1: Create ActionTier enum**

```java
package com.cardiofit.flink.models;

/**
 * Three-tier clinical action severity model + routine.
 *
 * HALT  = Critical safety — SMS + FCM + phone fallback, 30 min SLA
 * PAUSE = Needs physician review — FCM push + email, 24 hr SLA
 * SOFT_FLAG = Advisory — attached to next Decision Card, no SLA
 * ROUTINE = No action required
 */
public enum ActionTier {
    HALT(1, 30 * 60 * 1000L),           // 30-minute SLA
    PAUSE(2, 24 * 60 * 60 * 1000L),     // 24-hour SLA
    SOFT_FLAG(3, -1L),                   // no SLA
    ROUTINE(4, -1L);                     // no SLA

    private final int priority;
    private final long slaMs;

    ActionTier(int priority, long slaMs) {
        this.priority = priority;
        this.slaMs = slaMs;
    }

    public int getPriority() { return priority; }
    public long getSlaMs() { return slaMs; }
    public boolean requiresNotification() { return this == HALT || this == PAUSE; }
    public boolean requiresEscalation() { return slaMs > 0; }
}
```

- [ ] **Step 2: Create AlertState enum**

```java
package com.cardiofit.flink.models;

/**
 * Alert lifecycle state machine.
 *
 * ACTIVE → ACKNOWLEDGED → ACTIONED → RESOLVED
 * ACTIVE → AUTO_RESOLVED
 * ACTIVE → ESCALATED → ACTIONED → RESOLVED
 */
public enum AlertState {
    ACTIVE,
    ACKNOWLEDGED,
    ACTIONED,
    AUTO_RESOLVED,
    ESCALATED,
    RESOLVED;

    public boolean isTerminal() {
        return this == AUTO_RESOLVED || this == RESOLVED;
    }

    public boolean canTransitionTo(AlertState next) {
        return switch (this) {
            case ACTIVE -> next == ACKNOWLEDGED || next == AUTO_RESOLVED || next == ESCALATED;
            case ACKNOWLEDGED -> next == ACTIONED || next == RESOLVED;
            case ESCALATED -> next == ACTIONED || next == ACKNOWLEDGED || next == RESOLVED;
            case ACTIONED -> next == RESOLVED;
            case AUTO_RESOLVED, RESOLVED -> false;
        };
    }
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/ActionTier.java src/main/java/com/cardiofit/flink/models/AlertState.java
git commit -m "feat(module6): add ActionTier and AlertState enums for clinical action classification"
```

---

## Task 2: ClinicalEvent — Unified Input Wrapper

**Files:**
- Create: `src/main/java/com/cardiofit/flink/models/ClinicalEvent.java`

This wrapper unifies the four input schemas (CDSEvent, PatternEvent, MLPrediction) so that downstream classification logic has a single type to work with.

- [ ] **Step 1: Write the failing test**

Create `src/test/java/com/cardiofit/flink/operators/Module6ActionClassifierTest.java` with a single compilation-check test:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module6ActionClassifierTest {

    @Test
    void clinicalEvent_fromCDSEvent_extractsNews2Score() {
        CDSEvent cds = new CDSEvent();
        PatientContextState state = new PatientContextState("P001");
        state.setNews2Score(13);
        state.setQsofaScore(2);
        cds.setPatientId("P001");
        cds.setPatientState(state);

        ClinicalEvent event = ClinicalEvent.fromCDS(cds);

        assertEquals("P001", event.getPatientId());
        assertEquals(13, event.getNews2Score());
        assertEquals(2, event.getQsofaScore());
        assertEquals(ClinicalEvent.Source.CDS, event.getSource());
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module6ActionClassifierTest#clinicalEvent_fromCDSEvent_extractsNews2Score -q 2>&1 | tail -10`
Expected: FAIL — `ClinicalEvent` class does not exist

- [ ] **Step 3: Create ClinicalEvent**

```java
package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * Unified wrapper for all Module 6 input event types.
 * Exactly one of cdsEvent/patternEvent/prediction is non-null.
 */
public class ClinicalEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum Source { CDS, PATTERN, ML_PREDICTION }

    private String patientId;
    private long eventTime;
    private Source source;
    private CDSEvent cdsEvent;
    private PatternEvent patternEvent;
    private MLPrediction prediction;

    private ClinicalEvent() {}

    public static ClinicalEvent fromCDS(CDSEvent cds) {
        ClinicalEvent e = new ClinicalEvent();
        e.patientId = cds.getPatientId();
        e.eventTime = cds.getEventTime();
        e.source = Source.CDS;
        e.cdsEvent = cds;
        return e;
    }

    public static ClinicalEvent fromPattern(PatternEvent pattern) {
        ClinicalEvent e = new ClinicalEvent();
        e.patientId = pattern.getPatientId();
        e.eventTime = pattern.getDetectionTime();
        e.source = Source.PATTERN;
        e.patternEvent = pattern;
        return e;
    }

    public static ClinicalEvent fromPrediction(MLPrediction pred) {
        ClinicalEvent e = new ClinicalEvent();
        e.patientId = pred.getPatientId();
        e.eventTime = pred.getPredictionTime();
        e.source = Source.ML_PREDICTION;
        e.prediction = pred;
        return e;
    }

    // ── Convenience accessors across all source types ──

    public int getNews2Score() {
        if (cdsEvent != null && cdsEvent.getPatientState() != null
                && cdsEvent.getPatientState().getNews2Score() != null) {
            return cdsEvent.getPatientState().getNews2Score();
        }
        return 0;
    }

    public int getQsofaScore() {
        if (cdsEvent != null && cdsEvent.getPatientState() != null
                && cdsEvent.getPatientState().getQsofaScore() != null) {
            return cdsEvent.getPatientState().getQsofaScore();
        }
        return 0;
    }

    public boolean hasSepsisIndicators() {
        if (cdsEvent == null || cdsEvent.getPatientState() == null) return false;
        RiskIndicators ri = cdsEvent.getPatientState().getRiskIndicators();
        if (ri == null) return false;
        return ri.isFever() || ri.isElevatedLactate() || ri.isTachycardia();
    }

    /**
     * Check for an active alert by type and optional severity.
     * Searches PatientContextState.activeAlerts (Set of SimpleAlert).
     */
    public boolean hasActiveAlert(String alertType, String severity) {
        if (cdsEvent == null || cdsEvent.getPatientState() == null) return false;
        Set<SimpleAlert> alerts = cdsEvent.getPatientState().getActiveAlerts();
        if (alerts == null) return false;
        for (SimpleAlert alert : alerts) {
            String type = alert.getAlertType() != null ? alert.getAlertType().toString() : "";
            String sev = alert.getSeverity() != null ? alert.getSeverity().toString() : "";
            if (type.contains(alertType)) {
                if (severity == null || sev.equalsIgnoreCase(severity)) return true;
            }
            // Also check message field for alert type strings like "HYPERKALEMIA_ALERT"
            String msg = alert.getMessage() != null ? alert.getMessage() : "";
            if (msg.contains(alertType)) {
                if (severity == null || sev.equalsIgnoreCase(severity) || msg.contains(severity)) return true;
            }
        }
        return false;
    }

    public boolean hasActiveAlert(String alertType) {
        return hasActiveAlert(alertType, null);
    }

    public String getAlertDetail(String alertType, String detailKey) {
        if (cdsEvent == null || cdsEvent.getPatientState() == null) return null;
        Set<SimpleAlert> alerts = cdsEvent.getPatientState().getActiveAlerts();
        if (alerts == null) return null;
        for (SimpleAlert alert : alerts) {
            String msg = alert.getMessage() != null ? alert.getMessage() : "";
            String type = alert.getAlertType() != null ? alert.getAlertType().toString() : "";
            if (type.contains(alertType) || msg.contains(alertType)) {
                Map<String, Object> ctx = alert.getContext();
                if (ctx != null && ctx.containsKey(detailKey)) {
                    return String.valueOf(ctx.get(detailKey));
                }
            }
        }
        return null;
    }

    public boolean hasPattern(String patternType) {
        return patternEvent != null && patternType.equals(patternEvent.getPatternType());
    }

    public PatternEvent getPattern(String patternType) {
        if (hasPattern(patternType)) return patternEvent;
        return null;
    }

    public boolean hasPatternWithSeverity(String severity) {
        return patternEvent != null && severity.equalsIgnoreCase(patternEvent.getSeverity());
    }

    public boolean hasPrediction(String category) {
        return prediction != null && category.equalsIgnoreCase(prediction.getPredictionCategory());
    }

    public MLPrediction getPrediction(String category) {
        if (hasPrediction(category)) return prediction;
        return null;
    }

    public boolean hasAnyPredictionAbove(double threshold) {
        if (prediction == null || prediction.getCalibratedScore() == null) return false;
        return prediction.getCalibratedScore() >= threshold - 1e-9;
    }

    /**
     * Derive the clinical category for dedup keying.
     * CDS events → derived from active alerts or scores.
     * Pattern events → patternType.
     * ML predictions → predictionCategory.
     */
    public String getClinicalCategory() {
        if (patternEvent != null) return patternEvent.getPatternType();
        if (prediction != null) return prediction.getPredictionCategory() != null
                ? prediction.getPredictionCategory().toUpperCase() : "ML_PREDICTION";
        // CDS: derive from dominant risk
        if (cdsEvent != null) {
            if (hasActiveAlert("HYPERKALEMIA")) return "HYPERKALEMIA";
            if (hasActiveAlert("ANTICOAGULATION")) return "ANTICOAGULATION";
            if (hasActiveAlert("AKI")) return "AKI";
            if (getQsofaScore() >= 2 || hasActiveAlert("SEPSIS")) return "SEPSIS";
            if (getNews2Score() >= 7) return "DETERIORATION";
            return "CDS_GENERAL";
        }
        return "UNKNOWN";
    }

    // ── Standard getters ──
    public String getPatientId() { return patientId; }
    public long getEventTime() { return eventTime; }
    public Source getSource() { return source; }
    public CDSEvent getCdsEvent() { return cdsEvent; }
    public PatternEvent getPatternEvent() { return patternEvent; }
    public MLPrediction getPrediction() { return prediction; }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module6ActionClassifierTest#clinicalEvent_fromCDSEvent_extractsNews2Score -q 2>&1 | tail -10`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/ClinicalEvent.java src/test/java/com/cardiofit/flink/operators/Module6ActionClassifierTest.java
git commit -m "feat(module6): add ClinicalEvent unified input wrapper with convenience accessors"
```

---

## Task 3: Module6TestBuilder — Test Data Factory

**Files:**
- Create: `src/test/java/com/cardiofit/flink/builders/Module6TestBuilder.java`

Provides factory methods for the five canonical patient scenarios used throughout Module 6 tests. Mirrors the patterns in `Module5TestBuilder` and `Module4TestBuilder`.

- [ ] **Step 1: Create Module6TestBuilder**

```java
package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;

import java.util.*;

/**
 * Test data factory for Module 6 Clinical Action Engine tests.
 * Provides patient scenarios validated against passing E2E pipeline data.
 */
public class Module6TestBuilder {

    // ── ClinicalEvent builders from CDSEvent ──

    /**
     * Sepsis patient: NEWS2=13, qSOFA=2, elevated lactate/WBC.
     * Expected classification: HALT
     */
    public static ClinicalEvent sepsisHaltEvent(String patientId) {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId(patientId);
        cds.setEventTime(System.currentTimeMillis());

        PatientContextState state = new PatientContextState(patientId);
        state.setNews2Score(13);
        state.setQsofaScore(2);
        state.getLatestVitals().put("heartrate", 130);
        state.getLatestVitals().put("systolicbloodpressure", 78);
        state.getLatestVitals().put("respiratoryrate", 30);
        state.getLatestVitals().put("temperature", 39.8);
        state.getLatestVitals().put("oxygensaturation", 87);

        LabResult lactate = new LabResult();
        lactate.setLabCode("2524-7");
        lactate.setValue(4.8);
        lactate.setUnit("mmol/L");
        state.getRecentLabs().put("2524-7", lactate);

        RiskIndicators ri = new RiskIndicators();
        ri.setTachycardia(true);
        ri.setHypotension(true);
        ri.setFever(true);
        ri.setElevatedLactate(true);
        state.setRiskIndicators(ri);

        cds.setPatientState(state);
        return ClinicalEvent.fromCDS(cds);
    }

    /**
     * Hyperkalemia patient: NEWS2=2, qSOFA=0, K+=6.5.
     * Lab emergency invisible to vitals scoring.
     * Expected classification: HALT (lab-derived)
     */
    public static ClinicalEvent hyperkalemiaHaltEvent(String patientId) {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId(patientId);
        cds.setEventTime(System.currentTimeMillis());

        PatientContextState state = new PatientContextState(patientId);
        state.setNews2Score(2);
        state.setQsofaScore(0);
        state.getLatestVitals().put("heartrate", 82);
        state.getLatestVitals().put("systolicbloodpressure", 128);

        LabResult potassium = new LabResult();
        potassium.setLabCode("2823-3");
        potassium.setValue(6.5);
        potassium.setUnit("mmol/L");
        state.getRecentLabs().put("2823-3", potassium);

        // Simulate activeAlert for HYPERKALEMIA
        SimpleAlert kAlert = new SimpleAlert();
        kAlert.setAlertType(AlertType.LAB_CRITICAL_VALUE);
        kAlert.setSeverity(AlertSeverity.CRITICAL);
        kAlert.setMessage("HYPERKALEMIA_ALERT CRITICAL K+ 6.5");
        kAlert.setPatientId(patientId);
        kAlert.setTimestamp(System.currentTimeMillis());
        state.addAlert(kAlert);

        state.setRiskIndicators(new RiskIndicators());
        cds.setPatientState(state);
        return ClinicalEvent.fromCDS(cds);
    }

    /**
     * AKI Stage 3 patient: NEWS2=3, qSOFA=0, Cr=4.2.
     * Expected classification: HALT
     */
    public static ClinicalEvent akiStage3HaltEvent(String patientId) {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId(patientId);
        cds.setEventTime(System.currentTimeMillis());

        PatientContextState state = new PatientContextState(patientId);
        state.setNews2Score(3);
        state.setQsofaScore(0);

        LabResult creatinine = new LabResult();
        creatinine.setLabCode("2160-0");
        creatinine.setValue(4.2);
        creatinine.setUnit("mg/dL");
        state.getRecentLabs().put("2160-0", creatinine);

        SimpleAlert akiAlert = new SimpleAlert();
        akiAlert.setAlertType(AlertType.LAB_CRITICAL_VALUE);
        akiAlert.setSeverity(AlertSeverity.CRITICAL);
        akiAlert.setMessage("AKI_RISK STAGE_3");
        akiAlert.setPatientId(patientId);
        akiAlert.setTimestamp(System.currentTimeMillis());
        Map<String, Object> ctx = new HashMap<>();
        ctx.put("stage", "STAGE_3");
        akiAlert.setContext(ctx);
        state.addAlert(akiAlert);

        RiskIndicators ri = new RiskIndicators();
        ri.setElevatedCreatinine(true);
        state.setRiskIndicators(ri);

        cds.setPatientState(state);
        return ClinicalEvent.fromCDS(cds);
    }

    /**
     * Moderate deterioration: NEWS2=7, qSOFA=1.
     * Expected classification: PAUSE
     */
    public static ClinicalEvent moderateDeteriorationPauseEvent(String patientId) {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId(patientId);
        cds.setEventTime(System.currentTimeMillis());

        PatientContextState state = new PatientContextState(patientId);
        state.setNews2Score(7);
        state.setQsofaScore(1);
        state.getLatestVitals().put("heartrate", 105);
        state.getLatestVitals().put("systolicbloodpressure", 95);
        state.getLatestVitals().put("respiratoryrate", 22);
        state.getLatestVitals().put("temperature", 38.3);
        state.getLatestVitals().put("oxygensaturation", 93);

        state.setRiskIndicators(new RiskIndicators());
        cds.setPatientState(state);
        return ClinicalEvent.fromCDS(cds);
    }

    /**
     * Mildly elevated scores: NEWS2=5, qSOFA=0.
     * Expected classification: SOFT_FLAG
     */
    public static ClinicalEvent mildElevationSoftFlagEvent(String patientId) {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId(patientId);
        cds.setEventTime(System.currentTimeMillis());

        PatientContextState state = new PatientContextState(patientId);
        state.setNews2Score(5);
        state.setQsofaScore(0);

        state.setRiskIndicators(new RiskIndicators());
        cds.setPatientState(state);
        return ClinicalEvent.fromCDS(cds);
    }

    /**
     * Stable patient: NEWS2=1, qSOFA=0, normal labs.
     * Expected classification: ROUTINE
     */
    public static ClinicalEvent stableRoutineEvent(String patientId) {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId(patientId);
        cds.setEventTime(System.currentTimeMillis());

        PatientContextState state = new PatientContextState(patientId);
        state.setNews2Score(1);
        state.setQsofaScore(0);
        state.getLatestVitals().put("heartrate", 72);
        state.getLatestVitals().put("systolicbloodpressure", 122);

        state.setRiskIndicators(new RiskIndicators());
        cds.setPatientState(state);
        return ClinicalEvent.fromCDS(cds);
    }

    // ── ClinicalEvent builders from PatternEvent ──

    public static ClinicalEvent criticalDeteriorationPatternEvent(String patientId) {
        PatternEvent pe = PatternEvent.builder()
            .patternType("CLINICAL_DETERIORATION")
            .patientId(patientId)
            .severity("CRITICAL")
            .confidence(0.92)
            .detectionTime(System.currentTimeMillis())
            .recommendedActions(List.of("IMMEDIATE_ASSESSMENT_REQUIRED", "ESCALATE_TO_PHYSICIAN"))
            .build();
        pe.addTag("SEVERITY_ESCALATION");
        return ClinicalEvent.fromPattern(pe);
    }

    public static ClinicalEvent highSeverityPatternEvent(String patientId, String patternType) {
        PatternEvent pe = PatternEvent.builder()
            .patternType(patternType)
            .patientId(patientId)
            .severity("HIGH")
            .confidence(0.85)
            .detectionTime(System.currentTimeMillis())
            .build();
        return ClinicalEvent.fromPattern(pe);
    }

    public static ClinicalEvent moderatePatternEvent(String patientId, String patternType) {
        PatternEvent pe = PatternEvent.builder()
            .patternType(patternType)
            .patientId(patientId)
            .severity("MODERATE")
            .confidence(0.70)
            .detectionTime(System.currentTimeMillis())
            .build();
        return ClinicalEvent.fromPattern(pe);
    }

    // ── ClinicalEvent builders from MLPrediction ──

    public static ClinicalEvent sepsisHighRiskPrediction(String patientId, double calibratedScore) {
        MLPrediction pred = MLPrediction.builder()
            .patientId(patientId)
            .predictionCategory("sepsis")
            .calibratedScore(calibratedScore)
            .riskLevel(calibratedScore >= 0.60 ? "CRITICAL" : calibratedScore >= 0.35 ? "HIGH" : "MODERATE")
            .contextDepth("ESTABLISHED")
            .triggerSource("CDS_EVENT")
            .predictionTime(System.currentTimeMillis())
            .build();
        return ClinicalEvent.fromPrediction(pred);
    }

    public static ClinicalEvent deteriorationPrediction(String patientId, double calibratedScore) {
        MLPrediction pred = MLPrediction.builder()
            .patientId(patientId)
            .predictionCategory("deterioration")
            .calibratedScore(calibratedScore)
            .riskLevel(calibratedScore >= 0.45 ? "HIGH" : "MODERATE")
            .contextDepth("ESTABLISHED")
            .triggerSource("CDS_EVENT")
            .predictionTime(System.currentTimeMillis())
            .build();
        return ClinicalEvent.fromPrediction(pred);
    }

    public static ClinicalEvent lowRiskPrediction(String patientId) {
        MLPrediction pred = MLPrediction.builder()
            .patientId(patientId)
            .predictionCategory("readmission")
            .calibratedScore(0.15)
            .riskLevel("LOW")
            .contextDepth("ESTABLISHED")
            .predictionTime(System.currentTimeMillis())
            .build();
        return ClinicalEvent.fromPrediction(pred);
    }
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test-compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/builders/Module6TestBuilder.java
git commit -m "test(module6): add Module6TestBuilder with canonical patient scenarios"
```

---

## Task 4: ClinicalActionClassifier — Static Classification Logic

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module6ActionClassifier.java`
- Modify: `src/test/java/com/cardiofit/flink/operators/Module6ActionClassifierTest.java`

The classifier is intentionally a static utility class — no Flink runtime dependency, fully unit-testable. This is where patient safety logic lives.

- [ ] **Step 1: Write comprehensive classification tests**

Append to `Module6ActionClassifierTest.java`:

```java
    // ══ HALT conditions ══

    @Test
    void sepsisPatient_news2Above10_classifiesHalt() {
        ClinicalEvent event = Module6TestBuilder.sepsisHaltEvent("P-SEPSIS");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.HALT, tier, "NEWS2=13 must produce HALT");
    }

    @Test
    void hyperkalemiaPatient_news2Normal_classifiesHalt() {
        ClinicalEvent event = Module6TestBuilder.hyperkalemiaHaltEvent("P-HYPER-K");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.HALT, tier, "K+ 6.5 with CRITICAL alert must produce HALT even with low NEWS2");
    }

    @Test
    void akiStage3_classifiesHalt() {
        ClinicalEvent event = Module6TestBuilder.akiStage3HaltEvent("P-AKI");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.HALT, tier, "AKI Stage 3 must produce HALT");
    }

    @Test
    void sepsisMLPrediction_above060_classifiesHalt() {
        ClinicalEvent event = Module6TestBuilder.sepsisHighRiskPrediction("P-ML-SEPSIS", 0.72);
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.HALT, tier, "Sepsis calibrated score 0.72 must produce HALT");
    }

    @Test
    void sepsisMLPrediction_exactlyAt060_classifiesHalt() {
        ClinicalEvent event = Module6TestBuilder.sepsisHighRiskPrediction("P-ML-BOUNDARY", 0.60);
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.HALT, tier, "Sepsis calibrated score exactly 0.60 must produce HALT (epsilon)");
    }

    @Test
    void criticalDeteriorationPattern_classifiesHalt() {
        ClinicalEvent event = Module6TestBuilder.criticalDeteriorationPatternEvent("P-DET-CRIT");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.HALT, tier, "CLINICAL_DETERIORATION CRITICAL must produce HALT");
    }

    // ══ PAUSE conditions ══

    @Test
    void moderateDeterioration_news2Is7_classifiesPause() {
        ClinicalEvent event = Module6TestBuilder.moderateDeteriorationPauseEvent("P-MOD");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.PAUSE, tier, "NEWS2=7 must produce PAUSE");
    }

    @Test
    void sepsisMLPrediction_above035_below060_classifiesPause() {
        ClinicalEvent event = Module6TestBuilder.sepsisHighRiskPrediction("P-ML-MOD", 0.45);
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.PAUSE, tier, "Sepsis calibrated score 0.45 must produce PAUSE");
    }

    @Test
    void deteriorationPrediction_above045_classifiesPause() {
        ClinicalEvent event = Module6TestBuilder.deteriorationPrediction("P-DET-ML", 0.50);
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.PAUSE, tier, "Deterioration calibrated score 0.50 must produce PAUSE");
    }

    @Test
    void highSeverityPattern_classifiesPause() {
        ClinicalEvent event = Module6TestBuilder.highSeverityPatternEvent("P-PAT-HIGH", "VITAL_SIGNS_TREND");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.PAUSE, tier, "HIGH severity pattern must produce PAUSE");
    }

    // ══ SOFT_FLAG conditions ══

    @Test
    void mildElevation_news2Is5_classifiesSoftFlag() {
        ClinicalEvent event = Module6TestBuilder.mildElevationSoftFlagEvent("P-MILD");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.SOFT_FLAG, tier, "NEWS2=5 must produce SOFT_FLAG");
    }

    @Test
    void moderatePattern_classifiesSoftFlag() {
        ClinicalEvent event = Module6TestBuilder.moderatePatternEvent("P-PAT-MOD", "TREND_ANALYSIS");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.SOFT_FLAG, tier, "MODERATE severity pattern must produce SOFT_FLAG");
    }

    @Test
    void anyPredictionAbove025_classifiesSoftFlag() {
        ClinicalEvent event = Module6TestBuilder.deteriorationPrediction("P-DET-LOW", 0.30);
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.SOFT_FLAG, tier, "Deterioration calibrated score 0.30 must produce SOFT_FLAG");
    }

    // ══ ROUTINE conditions ══

    @Test
    void stablePatient_classifiesRoutine() {
        ClinicalEvent event = Module6TestBuilder.stableRoutineEvent("P-STABLE");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.ROUTINE, tier, "Stable patient must produce ROUTINE");
    }

    @Test
    void lowRiskPrediction_classifiesRoutine() {
        ClinicalEvent event = Module6TestBuilder.lowRiskPrediction("P-LOW-ML");
        ActionTier tier = Module6ActionClassifier.classify(event);
        assertEquals(ActionTier.ROUTINE, tier, "Low risk prediction (0.15) must produce ROUTINE");
    }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module6ActionClassifierTest -q 2>&1 | tail -10`
Expected: FAIL — `Module6ActionClassifier` class does not exist

- [ ] **Step 3: Implement Module6ActionClassifier**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;

/**
 * Static clinical action classifier. Determines the urgency tier for a clinical event.
 *
 * Evaluation order matters: HALT conditions checked first (most dangerous),
 * then PAUSE, then SOFT_FLAG. First match wins.
 *
 * This class has NO Flink dependencies — fully unit-testable.
 */
public final class Module6ActionClassifier {

    private Module6ActionClassifier() {} // static utility

    public static ActionTier classify(ClinicalEvent event) {

        // ══ HALT conditions (immediate danger) ══

        // Vitals-based
        if (event.getNews2Score() >= 10) return ActionTier.HALT;
        if (event.getQsofaScore() >= 2 && event.hasSepsisIndicators()) return ActionTier.HALT;

        // Lab emergencies (from active alerts — invisible to vitals scoring)
        if (event.hasActiveAlert("HYPERKALEMIA", "CRITICAL")) return ActionTier.HALT;
        if (event.hasActiveAlert("ANTICOAGULATION_RISK", "CRITICAL")) return ActionTier.HALT;
        if (event.hasActiveAlert("AKI_RISK")
                && "STAGE_3".equals(event.getAlertDetail("AKI_RISK", "stage"))) return ActionTier.HALT;

        // ML predictions at critical threshold (epsilon for IEEE 754)
        if (event.hasPrediction("sepsis")
                && event.getPrediction("sepsis").getCalibratedScore() != null
                && event.getPrediction("sepsis").getCalibratedScore() >= 0.60 - 1e-9) return ActionTier.HALT;

        // Pattern escalation
        if (event.hasPattern("CLINICAL_DETERIORATION")
                && "CRITICAL".equalsIgnoreCase(event.getPattern("CLINICAL_DETERIORATION").getSeverity()))
            return ActionTier.HALT;

        // ══ PAUSE conditions (needs physician review) ══

        if (event.getNews2Score() >= 7) return ActionTier.PAUSE;
        if (event.getQsofaScore() >= 1) return ActionTier.PAUSE;

        if (event.hasActiveAlert("AKI_RISK", "HIGH")) return ActionTier.PAUSE;
        if (event.hasActiveAlert("ANTICOAGULATION_RISK", "HIGH")) return ActionTier.PAUSE;
        if (event.hasActiveAlert("BLEEDING_RISK", "HIGH")) return ActionTier.PAUSE;

        if (event.hasPrediction("deterioration")
                && event.getPrediction("deterioration").getCalibratedScore() != null
                && event.getPrediction("deterioration").getCalibratedScore() >= 0.45 - 1e-9) return ActionTier.PAUSE;
        if (event.hasPrediction("sepsis")
                && event.getPrediction("sepsis").getCalibratedScore() != null
                && event.getPrediction("sepsis").getCalibratedScore() >= 0.35 - 1e-9) return ActionTier.PAUSE;

        if (event.hasPatternWithSeverity("HIGH")) return ActionTier.PAUSE;

        // ══ SOFT_FLAG conditions (advisory) ══

        if (event.getNews2Score() >= 5) return ActionTier.SOFT_FLAG;
        if (event.hasActiveAlert("AKI_RISK", "MODERATE")) return ActionTier.SOFT_FLAG;
        if (event.hasAnyPredictionAbove(0.25)) return ActionTier.SOFT_FLAG;
        if (event.hasPatternWithSeverity("MODERATE")) return ActionTier.SOFT_FLAG;

        return ActionTier.ROUTINE;
    }
}
```

- [ ] **Step 4: Run tests to verify they all pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module6ActionClassifierTest -q 2>&1 | tail -15`
Expected: All 15 tests PASS

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module6ActionClassifier.java src/test/java/com/cardiofit/flink/operators/Module6ActionClassifierTest.java
git commit -m "feat(module6): implement ClinicalActionClassifier with HALT/PAUSE/SOFT_FLAG/ROUTINE classification"
```

---

## Task 5: CrossModuleDeduplicator + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module6CrossModuleDedup.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module6DeduplicationTest.java`

- [ ] **Step 1: Write deduplication tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module6DeduplicationTest {

    private Module6CrossModuleDedup dedup;

    @BeforeEach
    void setUp() {
        dedup = new Module6CrossModuleDedup();
    }

    @Test
    void firstAlert_alwaysEmitted() {
        long now = System.currentTimeMillis();
        assertTrue(dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now));
    }

    @Test
    void duplicateWithinHaltWindow_suppressed() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now);
        assertFalse(dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now + 60_000),
            "Same HALT within 5 min should be suppressed");
    }

    @Test
    void duplicateAfterHaltWindow_emitted() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now);
        assertTrue(dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now + 6 * 60_000),
            "Same HALT after 5 min window should emit");
    }

    @Test
    void duplicateWithinPauseWindow_suppressed() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.PAUSE, "AKI", now);
        assertFalse(dedup.shouldEmit("P001", ActionTier.PAUSE, "AKI", now + 10 * 60_000),
            "Same PAUSE within 30 min should be suppressed");
    }

    @Test
    void differentClinicalCategory_notSuppressed() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now);
        assertTrue(dedup.shouldEmit("P001", ActionTier.HALT, "HYPERKALEMIA", now + 1000),
            "Different clinical category should not be suppressed");
    }

    @Test
    void differentPatient_notSuppressed() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now);
        assertTrue(dedup.shouldEmit("P002", ActionTier.HALT, "SEPSIS", now + 1000),
            "Different patient should not be suppressed");
    }

    @Test
    void softFlagWindow_60minutes() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.SOFT_FLAG, "TREND", now);
        assertFalse(dedup.shouldEmit("P001", ActionTier.SOFT_FLAG, "TREND", now + 30 * 60_000),
            "Same SOFT_FLAG within 60 min should be suppressed");
        assertTrue(dedup.shouldEmit("P001", ActionTier.SOFT_FLAG, "TREND", now + 61 * 60_000),
            "Same SOFT_FLAG after 60 min should emit");
    }

    @Test
    void routineEvents_neverSuppressed() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.ROUTINE, "CDS_GENERAL", now);
        assertTrue(dedup.shouldEmit("P001", ActionTier.ROUTINE, "CDS_GENERAL", now + 1000),
            "ROUTINE events should never be suppressed");
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module6DeduplicationTest -q 2>&1 | tail -10`
Expected: FAIL — `Module6CrossModuleDedup` does not exist

- [ ] **Step 3: Implement CrossModuleDeduplicator**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.ActionTier;

import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * Cross-module alert deduplication.
 *
 * Different from Module 4's within-pattern dedup. Module 6 deduplicates ACROSS modules —
 * three modules detecting the same clinical situation produce ONE alert.
 *
 * Dedup windows are tier-specific:
 * - HALT:      5 minutes  (critical alerts need fast re-evaluation)
 * - PAUSE:     30 minutes (physician review window)
 * - SOFT_FLAG: 60 minutes (advisory, lower noise)
 * - ROUTINE:   no dedup   (always pass through)
 */
public class Module6CrossModuleDedup implements Serializable {
    private static final long serialVersionUID = 1L;

    private static final long HALT_DEDUP_WINDOW_MS = 5 * 60 * 1000L;
    private static final long PAUSE_DEDUP_WINDOW_MS = 30 * 60 * 1000L;
    private static final long SOFT_FLAG_DEDUP_WINDOW_MS = 60 * 60 * 1000L;

    private final Map<String, Long> recentAlertState = new HashMap<>();

    public boolean shouldEmit(String patientId, ActionTier tier,
                              String clinicalCategory, long eventTime) {

        if (tier == ActionTier.ROUTINE) return true;

        String dedupKey = patientId + ":" + tier + ":" + clinicalCategory;
        Long lastEmitted = recentAlertState.get(dedupKey);

        long window = switch (tier) {
            case HALT -> HALT_DEDUP_WINDOW_MS;
            case PAUSE -> PAUSE_DEDUP_WINDOW_MS;
            case SOFT_FLAG -> SOFT_FLAG_DEDUP_WINDOW_MS;
            default -> Long.MAX_VALUE;
        };

        if (lastEmitted != null && (eventTime - lastEmitted) < window) {
            return false; // within dedup window — suppress
        }

        recentAlertState.put(dedupKey, eventTime);
        return true;
    }

    /** Remove entries older than the maximum window to bound memory. */
    public void pruneExpired(long currentTime) {
        recentAlertState.entrySet().removeIf(entry ->
            (currentTime - entry.getValue()) > SOFT_FLAG_DEDUP_WINDOW_MS);
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module6DeduplicationTest -q 2>&1 | tail -10`
Expected: All 8 tests PASS

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module6CrossModuleDedup.java src/test/java/com/cardiofit/flink/operators/Module6DeduplicationTest.java
git commit -m "feat(module6): implement CrossModuleDeduplicator with tier-specific windows"
```

---

## Task 6: Output Models — ClinicalAlert, ClinicalAction, NotificationRequest, AuditRecord, FhirWriteRequest, PatientAlertState, AlertAcknowledgment

**Files:**
- Create: 7 new model files in `src/main/java/com/cardiofit/flink/models/`

These are data classes — no logic, just structure. All implement `Serializable`.

- [ ] **Step 1: Create ClinicalAlert**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.*;

/**
 * Full alert entity with lifecycle tracking.
 * Replaces ComposedAlert for Module 6 upgrade output.
 */
public class ClinicalAlert implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("alert_id") private String alertId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("encounter_id") private String encounterId;

    // Classification
    @JsonProperty("tier") private ActionTier tier;
    @JsonProperty("clinical_category") private String clinicalCategory;

    // Clinical content
    @JsonProperty("title") private String title;
    @JsonProperty("body") private String body;
    @JsonProperty("recommended_actions") private List<String> recommendedActions = new ArrayList<>();
    @JsonProperty("clinical_context") private Map<String, Object> clinicalContext = new HashMap<>();
    @JsonProperty("ml_predictions") private Map<String, Double> mlPredictions = new HashMap<>();

    // Source provenance
    @JsonProperty("source_module") private String sourceModule;
    @JsonProperty("trigger_event_id") private String triggerEventId;
    @JsonProperty("correlation_id") private String correlationId;
    @JsonProperty("contributing_sources") private List<String> contributingSources = new ArrayList<>();

    // Lifecycle
    @JsonProperty("state") private AlertState state = AlertState.ACTIVE;
    @JsonProperty("created_at") private long createdAt = System.currentTimeMillis();
    @JsonProperty("acknowledged_at") private Long acknowledgedAt;
    @JsonProperty("actioned_at") private Long actionedAt;
    @JsonProperty("resolved_at") private Long resolvedAt;
    @JsonProperty("escalated_at") private Long escalatedAt;
    @JsonProperty("acknowledged_by") private String acknowledgedBy;
    @JsonProperty("action_description") private String actionDescription;

    // SLA
    @JsonProperty("sla_deadline_ms") private long slaDeadlineMs;
    @JsonProperty("escalation_level") private int escalationLevel;
    @JsonProperty("assigned_to") private String assignedTo;

    public ClinicalAlert() {}

    // Getters and setters — all fields
    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public String getEncounterId() { return encounterId; }
    public void setEncounterId(String encounterId) { this.encounterId = encounterId; }
    public ActionTier getTier() { return tier; }
    public void setTier(ActionTier tier) { this.tier = tier; }
    public String getClinicalCategory() { return clinicalCategory; }
    public void setClinicalCategory(String clinicalCategory) { this.clinicalCategory = clinicalCategory; }
    public String getTitle() { return title; }
    public void setTitle(String title) { this.title = title; }
    public String getBody() { return body; }
    public void setBody(String body) { this.body = body; }
    public List<String> getRecommendedActions() { return recommendedActions; }
    public void setRecommendedActions(List<String> recommendedActions) { this.recommendedActions = recommendedActions; }
    public Map<String, Object> getClinicalContext() { return clinicalContext; }
    public void setClinicalContext(Map<String, Object> clinicalContext) { this.clinicalContext = clinicalContext; }
    public Map<String, Double> getMlPredictions() { return mlPredictions; }
    public void setMlPredictions(Map<String, Double> mlPredictions) { this.mlPredictions = mlPredictions; }
    public String getSourceModule() { return sourceModule; }
    public void setSourceModule(String sourceModule) { this.sourceModule = sourceModule; }
    public String getTriggerEventId() { return triggerEventId; }
    public void setTriggerEventId(String triggerEventId) { this.triggerEventId = triggerEventId; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String correlationId) { this.correlationId = correlationId; }
    public List<String> getContributingSources() { return contributingSources; }
    public void setContributingSources(List<String> contributingSources) { this.contributingSources = contributingSources; }
    public AlertState getState() { return state; }
    public void setState(AlertState state) { this.state = state; }
    public long getCreatedAt() { return createdAt; }
    public void setCreatedAt(long createdAt) { this.createdAt = createdAt; }
    public Long getAcknowledgedAt() { return acknowledgedAt; }
    public void setAcknowledgedAt(Long acknowledgedAt) { this.acknowledgedAt = acknowledgedAt; }
    public Long getActionedAt() { return actionedAt; }
    public void setActionedAt(Long actionedAt) { this.actionedAt = actionedAt; }
    public Long getResolvedAt() { return resolvedAt; }
    public void setResolvedAt(Long resolvedAt) { this.resolvedAt = resolvedAt; }
    public Long getEscalatedAt() { return escalatedAt; }
    public void setEscalatedAt(Long escalatedAt) { this.escalatedAt = escalatedAt; }
    public String getAcknowledgedBy() { return acknowledgedBy; }
    public void setAcknowledgedBy(String acknowledgedBy) { this.acknowledgedBy = acknowledgedBy; }
    public String getActionDescription() { return actionDescription; }
    public void setActionDescription(String actionDescription) { this.actionDescription = actionDescription; }
    public long getSlaDeadlineMs() { return slaDeadlineMs; }
    public void setSlaDeadlineMs(long slaDeadlineMs) { this.slaDeadlineMs = slaDeadlineMs; }
    public int getEscalationLevel() { return escalationLevel; }
    public void setEscalationLevel(int escalationLevel) { this.escalationLevel = escalationLevel; }
    public String getAssignedTo() { return assignedTo; }
    public void setAssignedTo(String assignedTo) { this.assignedTo = assignedTo; }
}
```

- [ ] **Step 2: Create ClinicalAction**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Output action record emitted by Module 6.
 * Main collector output of the ClinicalActionEngine.
 */
public class ClinicalAction implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("action_id") private String actionId;
    @JsonProperty("action_type") private String actionType; // NEW_ALERT, ESCALATION, AUTO_RESOLVED, ACKNOWLEDGMENT
    @JsonProperty("alert") private ClinicalAlert alert;
    @JsonProperty("timestamp") private long timestamp = System.currentTimeMillis();

    public ClinicalAction() {}

    public static ClinicalAction newAlert(ClinicalAlert alert) {
        ClinicalAction a = new ClinicalAction();
        a.actionId = java.util.UUID.randomUUID().toString();
        a.actionType = "NEW_ALERT";
        a.alert = alert;
        return a;
    }

    public static ClinicalAction escalation(ClinicalAlert alert, String escalateTo) {
        ClinicalAction a = new ClinicalAction();
        a.actionId = java.util.UUID.randomUUID().toString();
        a.actionType = "ESCALATION";
        a.alert = alert;
        return a;
    }

    public static ClinicalAction autoResolved(ClinicalAlert alert) {
        ClinicalAction a = new ClinicalAction();
        a.actionId = java.util.UUID.randomUUID().toString();
        a.actionType = "AUTO_RESOLVED";
        a.alert = alert;
        return a;
    }

    public String getActionId() { return actionId; }
    public void setActionId(String actionId) { this.actionId = actionId; }
    public String getActionType() { return actionType; }
    public void setActionType(String actionType) { this.actionType = actionType; }
    public ClinicalAlert getAlert() { return alert; }
    public void setAlert(ClinicalAlert alert) { this.alert = alert; }
    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }
}
```

- [ ] **Step 3: Create NotificationRequest**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * Structured notification request emitted to clinical-notifications.v1.
 * Module 6 does NOT send notifications directly — a separate service handles delivery.
 */
public class NotificationRequest implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum Channel { SMS, FCM_PUSH, EMAIL, PHONE_FALLBACK, DASHBOARD_ONLY }

    @JsonProperty("notification_id") private String notificationId;
    @JsonProperty("alert_id") private String alertId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("channel") private Channel channel;
    @JsonProperty("tier") private ActionTier tier;
    @JsonProperty("title") private String title;
    @JsonProperty("body") private String body;
    @JsonProperty("data") private Map<String, String> data = new HashMap<>();
    @JsonProperty("created_at") private long createdAt = System.currentTimeMillis();
    @JsonProperty("priority") private int priority;
    @JsonProperty("requires_acknowledgment") private boolean requiresAcknowledgment;

    public NotificationRequest() {}

    // Getters and setters
    public String getNotificationId() { return notificationId; }
    public void setNotificationId(String notificationId) { this.notificationId = notificationId; }
    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Channel getChannel() { return channel; }
    public void setChannel(Channel channel) { this.channel = channel; }
    public ActionTier getTier() { return tier; }
    public void setTier(ActionTier tier) { this.tier = tier; }
    public String getTitle() { return title; }
    public void setTitle(String title) { this.title = title; }
    public String getBody() { return body; }
    public void setBody(String body) { this.body = body; }
    public Map<String, String> getData() { return data; }
    public void setData(Map<String, String> data) { this.data = data; }
    public long getCreatedAt() { return createdAt; }
    public void setCreatedAt(long createdAt) { this.createdAt = createdAt; }
    public int getPriority() { return priority; }
    public void setPriority(int priority) { this.priority = priority; }
    public boolean isRequiresAcknowledgment() { return requiresAcknowledgment; }
    public void setRequiresAcknowledgment(boolean requiresAcknowledgment) { this.requiresAcknowledgment = requiresAcknowledgment; }
}
```

- [ ] **Step 4: Create AuditRecord**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * Audit trail record. Every clinical decision, alert, notification, and action
 * must be auditable. Healthcare regulations require 7-year retention.
 * Emitted to prod.ehr.audit.logs (retention: 2555 days / 7 years).
 */
public class AuditRecord implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("audit_id") private String auditId;
    @JsonProperty("timestamp") private long timestamp = System.currentTimeMillis();
    @JsonProperty("event_type") private String eventType; // ALERT_CREATED, ALERT_ACKNOWLEDGED, ALERT_ESCALATED, etc.
    @JsonProperty("event_description") private String eventDescription;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("source_module") private String sourceModule;
    @JsonProperty("tier") private ActionTier tier;
    @JsonProperty("clinical_category") private String clinicalCategory;
    @JsonProperty("clinical_data") private Map<String, Object> clinicalData = new HashMap<>();
    @JsonProperty("correlation_id") private String correlationId;
    @JsonProperty("alert_id") private String alertId;
    @JsonProperty("model_version") private String modelVersion;
    @JsonProperty("input_snapshot") private Map<String, Object> inputSnapshot = new HashMap<>();

    public AuditRecord() {}

    public static AuditRecord alertCreated(ClinicalAlert alert, ClinicalEvent sourceEvent) {
        AuditRecord r = new AuditRecord();
        r.auditId = java.util.UUID.randomUUID().toString();
        r.eventType = "ALERT_CREATED";
        r.eventDescription = "Clinical alert created: " + alert.getTier() + " " + alert.getClinicalCategory();
        r.patientId = alert.getPatientId();
        r.sourceModule = alert.getSourceModule();
        r.tier = alert.getTier();
        r.clinicalCategory = alert.getClinicalCategory();
        r.alertId = alert.getAlertId();
        r.correlationId = alert.getCorrelationId();
        return r;
    }

    // Getters and setters
    public String getAuditId() { return auditId; }
    public void setAuditId(String auditId) { this.auditId = auditId; }
    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }
    public String getEventType() { return eventType; }
    public void setEventType(String eventType) { this.eventType = eventType; }
    public String getEventDescription() { return eventDescription; }
    public void setEventDescription(String eventDescription) { this.eventDescription = eventDescription; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public String getSourceModule() { return sourceModule; }
    public void setSourceModule(String sourceModule) { this.sourceModule = sourceModule; }
    public ActionTier getTier() { return tier; }
    public void setTier(ActionTier tier) { this.tier = tier; }
    public String getClinicalCategory() { return clinicalCategory; }
    public void setClinicalCategory(String clinicalCategory) { this.clinicalCategory = clinicalCategory; }
    public Map<String, Object> getClinicalData() { return clinicalData; }
    public void setClinicalData(Map<String, Object> clinicalData) { this.clinicalData = clinicalData; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String correlationId) { this.correlationId = correlationId; }
    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }
    public String getModelVersion() { return modelVersion; }
    public void setModelVersion(String modelVersion) { this.modelVersion = modelVersion; }
    public Map<String, Object> getInputSnapshot() { return inputSnapshot; }
    public void setInputSnapshot(Map<String, Object> inputSnapshot) { this.inputSnapshot = inputSnapshot; }
}
```

- [ ] **Step 5: Create FhirWriteRequest**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Structured FHIR write-back request emitted to fhir-writeback topic.
 * A separate FHIR Writer Service handles delivery — no HTTP calls inside Flink.
 */
public class FhirWriteRequest implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum ResourceType { OBSERVATION, RISK_ASSESSMENT, DETECTED_ISSUE, CLINICAL_IMPRESSION, FLAG, COMMUNICATION_REQUEST }
    public enum WritePriority { CRITICAL, NORMAL, LOW }

    @JsonProperty("request_id") private String requestId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("resource_type") private ResourceType resourceType;
    @JsonProperty("fhir_resource_json") private String fhirResourceJson;
    @JsonProperty("priority") private WritePriority priority;
    @JsonProperty("created_at") private long createdAt = System.currentTimeMillis();
    @JsonProperty("max_retries") private int maxRetries = 3;

    public FhirWriteRequest() {}

    // Getters and setters
    public String getRequestId() { return requestId; }
    public void setRequestId(String requestId) { this.requestId = requestId; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public ResourceType getResourceType() { return resourceType; }
    public void setResourceType(ResourceType resourceType) { this.resourceType = resourceType; }
    public String getFhirResourceJson() { return fhirResourceJson; }
    public void setFhirResourceJson(String fhirResourceJson) { this.fhirResourceJson = fhirResourceJson; }
    public WritePriority getPriority() { return priority; }
    public void setPriority(WritePriority priority) { this.priority = priority; }
    public long getCreatedAt() { return createdAt; }
    public void setCreatedAt(long createdAt) { this.createdAt = createdAt; }
    public int getMaxRetries() { return maxRetries; }
    public void setMaxRetries(int maxRetries) { this.maxRetries = maxRetries; }
}
```

- [ ] **Step 6: Create PatientAlertState and AlertAcknowledgment**

`PatientAlertState.java`:
```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * Per-patient alert state maintained in Flink keyed state.
 * Tracks active alerts, dedup history, and fatigue metrics.
 */
public class PatientAlertState implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("active_alerts") private Map<String, ClinicalAlert> activeAlerts = new HashMap<>();
    @JsonProperty("alerts_in_last_24h") private int alertsInLast24Hours;
    @JsonProperty("alert_window_start") private long alertWindowStart;

    public PatientAlertState() { this.alertWindowStart = System.currentTimeMillis(); }
    public PatientAlertState(String patientId) { this(); this.patientId = patientId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Map<String, ClinicalAlert> getActiveAlerts() { return activeAlerts; }
    public void setActiveAlerts(Map<String, ClinicalAlert> activeAlerts) { this.activeAlerts = activeAlerts; }
    public int getAlertsInLast24Hours() { return alertsInLast24Hours; }
    public void setAlertsInLast24Hours(int alertsInLast24Hours) { this.alertsInLast24Hours = alertsInLast24Hours; }
    public long getAlertWindowStart() { return alertWindowStart; }
    public void setAlertWindowStart(long alertWindowStart) { this.alertWindowStart = alertWindowStart; }
}
```

`AlertAcknowledgment.java`:
```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Physician acknowledgment flowing back via alert-acknowledgments.v1.
 */
public class AlertAcknowledgment implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum AckAction { ACKNOWLEDGE, ACTION_TAKEN, DISMISS }

    @JsonProperty("alert_id") private String alertId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("clinical_category") private String clinicalCategory;
    @JsonProperty("action") private AckAction action;
    @JsonProperty("practitioner_id") private String practitionerId;
    @JsonProperty("action_description") private String actionDescription;
    @JsonProperty("timestamp") private long timestamp;

    public AlertAcknowledgment() {}

    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public String getClinicalCategory() { return clinicalCategory; }
    public void setClinicalCategory(String clinicalCategory) { this.clinicalCategory = clinicalCategory; }
    public AckAction getAction() { return action; }
    public void setAction(AckAction action) { this.action = action; }
    public String getPractitionerId() { return practitionerId; }
    public void setPractitionerId(String practitionerId) { this.practitionerId = practitionerId; }
    public String getActionDescription() { return actionDescription; }
    public void setActionDescription(String actionDescription) { this.actionDescription = actionDescription; }
    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }
}
```

- [ ] **Step 7: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 8: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/ClinicalAlert.java \
  src/main/java/com/cardiofit/flink/models/ClinicalAction.java \
  src/main/java/com/cardiofit/flink/models/NotificationRequest.java \
  src/main/java/com/cardiofit/flink/models/AuditRecord.java \
  src/main/java/com/cardiofit/flink/models/FhirWriteRequest.java \
  src/main/java/com/cardiofit/flink/models/PatientAlertState.java \
  src/main/java/com/cardiofit/flink/models/AlertAcknowledgment.java
git commit -m "feat(module6): add output models — ClinicalAlert, ClinicalAction, NotificationRequest, AuditRecord, FhirWriteRequest, PatientAlertState, AlertAcknowledgment"
```

---

## Task 7: NotificationRouter + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/routing/NotificationRouter.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module6NotificationRoutingTest.java`

- [ ] **Step 1: Write notification routing tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.routing.NotificationRouter;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module6NotificationRoutingTest {

    @Test
    void haltAlert_getsSmsAndFcmAndPhoneFallback() {
        ClinicalAlert alert = new ClinicalAlert();
        alert.setTier(ActionTier.HALT);
        List<NotificationRequest.Channel> channels = NotificationRouter.getChannels(alert);
        assertEquals(3, channels.size());
        assertTrue(channels.contains(NotificationRequest.Channel.SMS));
        assertTrue(channels.contains(NotificationRequest.Channel.FCM_PUSH));
        assertTrue(channels.contains(NotificationRequest.Channel.PHONE_FALLBACK));
    }

    @Test
    void pauseAlert_getsFcmAndEmail() {
        ClinicalAlert alert = new ClinicalAlert();
        alert.setTier(ActionTier.PAUSE);
        List<NotificationRequest.Channel> channels = NotificationRouter.getChannels(alert);
        assertEquals(2, channels.size());
        assertTrue(channels.contains(NotificationRequest.Channel.FCM_PUSH));
        assertTrue(channels.contains(NotificationRequest.Channel.EMAIL));
    }

    @Test
    void softFlagAlert_getsDashboardOnly() {
        ClinicalAlert alert = new ClinicalAlert();
        alert.setTier(ActionTier.SOFT_FLAG);
        List<NotificationRequest.Channel> channels = NotificationRouter.getChannels(alert);
        assertEquals(1, channels.size());
        assertTrue(channels.contains(NotificationRequest.Channel.DASHBOARD_ONLY));
    }

    @Test
    void routineAlert_getsDashboardOnly() {
        ClinicalAlert alert = new ClinicalAlert();
        alert.setTier(ActionTier.ROUTINE);
        List<NotificationRequest.Channel> channels = NotificationRouter.getChannels(alert);
        assertEquals(1, channels.size());
        assertTrue(channels.contains(NotificationRequest.Channel.DASHBOARD_ONLY));
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module6NotificationRoutingTest -q 2>&1 | tail -10`
Expected: FAIL

- [ ] **Step 3: Implement NotificationRouter**

```java
package com.cardiofit.flink.routing;

import com.cardiofit.flink.models.*;

import java.util.List;

/**
 * Notification channel selection based on alert tier.
 * Module 6 emits NotificationRequests — a separate service handles delivery.
 */
public final class NotificationRouter {

    private NotificationRouter() {}

    public static List<NotificationRequest.Channel> getChannels(ClinicalAlert alert) {
        return switch (alert.getTier()) {
            case HALT -> List.of(
                NotificationRequest.Channel.SMS,
                NotificationRequest.Channel.FCM_PUSH,
                NotificationRequest.Channel.PHONE_FALLBACK
            );
            case PAUSE -> List.of(
                NotificationRequest.Channel.FCM_PUSH,
                NotificationRequest.Channel.EMAIL
            );
            case SOFT_FLAG -> List.of(NotificationRequest.Channel.DASHBOARD_ONLY);
            case ROUTINE -> List.of(NotificationRequest.Channel.DASHBOARD_ONLY);
        };
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module6NotificationRoutingTest -q 2>&1 | tail -10`
Expected: All 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/routing/NotificationRouter.java src/test/java/com/cardiofit/flink/operators/Module6NotificationRoutingTest.java
git commit -m "feat(module6): implement NotificationRouter with tier-based channel selection"
```

---

## Task 8: AlertLifecycleManager + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/lifecycle/AlertLifecycleManager.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module6AlertLifecycleTest.java`

- [ ] **Step 1: Write alert lifecycle tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.lifecycle.AlertLifecycleManager;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module6AlertLifecycleTest {

    @Test
    void newAlert_startsAsActive() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        assertEquals(AlertState.ACTIVE, alert.getState());
        assertNotNull(alert.getAlertId());
        assertTrue(alert.getSlaDeadlineMs() > 0, "HALT must have SLA deadline");
    }

    @Test
    void haltSla_is30minutes() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        long expectedDeadline = alert.getCreatedAt() + (30 * 60 * 1000L);
        assertEquals(expectedDeadline, alert.getSlaDeadlineMs());
    }

    @Test
    void pauseSla_is24hours() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.PAUSE, "AKI", "MODULE_3_CDS");
        long expectedDeadline = alert.getCreatedAt() + (24 * 60 * 60 * 1000L);
        assertEquals(expectedDeadline, alert.getSlaDeadlineMs());
    }

    @Test
    void softFlagSla_isNegativeOne() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.SOFT_FLAG, "TREND", "MODULE_4_CEP");
        assertEquals(-1L, alert.getSlaDeadlineMs());
    }

    @Test
    void alertFatigue_capsAt10Per24Hours() {
        PatientAlertState state = new PatientAlertState("P001");
        for (int i = 0; i < 10; i++) {
            assertFalse(AlertLifecycleManager.checkAlertFatigue(state),
                "Alert " + (i+1) + " should not be fatigued");
        }
        assertTrue(AlertLifecycleManager.checkAlertFatigue(state),
            "Alert 11 should trigger fatigue protection");
    }

    @Test
    void alertFatigue_resetsAfter24Hours() {
        PatientAlertState state = new PatientAlertState("P001");
        state.setAlertsInLast24Hours(10);
        state.setAlertWindowStart(System.currentTimeMillis() - 25 * 60 * 60 * 1000L); // 25h ago
        assertFalse(AlertLifecycleManager.checkAlertFatigue(state),
            "Fatigue should reset after 24h window expires");
    }

    @Test
    void escalation_level1_goesToCareCoordinator() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        AlertLifecycleManager.escalate(alert);
        assertEquals(AlertState.ESCALATED, alert.getState());
        assertEquals(1, alert.getEscalationLevel());
        assertEquals("CARE_COORDINATOR", alert.getAssignedTo());
    }

    @Test
    void escalation_level2_goesToClinicalSupervisor() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        AlertLifecycleManager.escalate(alert); // level 1
        AlertLifecycleManager.escalate(alert); // level 2
        assertEquals(2, alert.getEscalationLevel());
        assertEquals("CLINICAL_SUPERVISOR", alert.getAssignedTo());
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module6AlertLifecycleTest -q 2>&1 | tail -10`
Expected: FAIL

- [ ] **Step 3: Implement AlertLifecycleManager**

```java
package com.cardiofit.flink.lifecycle;

import com.cardiofit.flink.models.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.UUID;

/**
 * Alert lifecycle operations: creation, SLA management, escalation, fatigue protection.
 * Stateless utility — state is managed by the Flink operator.
 */
public final class AlertLifecycleManager {
    private static final Logger LOG = LoggerFactory.getLogger(AlertLifecycleManager.class);
    private static final int MAX_ALERTS_PER_24H = 10;

    private AlertLifecycleManager() {}

    public static ClinicalAlert createAlert(String patientId, ActionTier tier,
                                             String clinicalCategory, String sourceModule) {
        ClinicalAlert alert = new ClinicalAlert();
        alert.setAlertId(UUID.randomUUID().toString());
        alert.setPatientId(patientId);
        alert.setTier(tier);
        alert.setClinicalCategory(clinicalCategory);
        alert.setSourceModule(sourceModule);
        alert.setState(AlertState.ACTIVE);
        alert.setCreatedAt(System.currentTimeMillis());

        // SLA deadline
        long sla = tier.getSlaMs();
        alert.setSlaDeadlineMs(sla > 0 ? alert.getCreatedAt() + sla : -1L);

        return alert;
    }

    /**
     * Check alert fatigue: cap at 10 alerts per 24-hour window per patient.
     * @return true if fatigued (should suppress), false if OK to emit
     */
    public static boolean checkAlertFatigue(PatientAlertState state) {
        long now = System.currentTimeMillis();
        if (now - state.getAlertWindowStart() > 24 * 60 * 60 * 1000L) {
            state.setAlertsInLast24Hours(0);
            state.setAlertWindowStart(now);
        }
        if (state.getAlertsInLast24Hours() >= MAX_ALERTS_PER_24H) {
            LOG.warn("Alert fatigue threshold for patient {} — {} alerts in 24h. Suppressing.",
                state.getPatientId(), state.getAlertsInLast24Hours());
            return true;
        }
        state.setAlertsInLast24Hours(state.getAlertsInLast24Hours() + 1);
        return false;
    }

    public static void escalate(ClinicalAlert alert) {
        alert.setState(AlertState.ESCALATED);
        alert.setEscalatedAt(System.currentTimeMillis());
        alert.setEscalationLevel(alert.getEscalationLevel() + 1);

        String escalateTo = switch (alert.getEscalationLevel()) {
            case 1 -> "CARE_COORDINATOR";
            case 2 -> "CLINICAL_SUPERVISOR";
            default -> "DEPARTMENT_HEAD";
        };
        alert.setAssignedTo(escalateTo);
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module6AlertLifecycleTest -q 2>&1 | tail -10`
Expected: All 8 tests PASS

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/lifecycle/AlertLifecycleManager.java src/test/java/com/cardiofit/flink/operators/Module6AlertLifecycleTest.java
git commit -m "feat(module6): implement AlertLifecycleManager with SLA, fatigue cap, and escalation"
```

---

## Task 9: Kafka Topics — Add Module 6 Topics

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/utils/KafkaTopics.java`

- [ ] **Step 1: Add Module 6 topics to KafkaTopics enum**

Add these entries to the enum (after existing entries, before the closing semicolon):

```java
    // ── Module 6: Clinical Action Engine ──
    CLINICAL_NOTIFICATIONS("clinical-notifications.v1", 4, 7),
    CLINICAL_AUDIT("clinical-audit.v1", 4, 2555),          // 7-year retention
    CLINICAL_ACTIONS("clinical-actions.v1", 4, 30),
    FHIR_WRITEBACK("fhir-writeback.v1", 4, 30),
    ALERT_STATE_UPDATES("alert-state-updates.v1", 4, 30),
    ALERT_ACKNOWLEDGMENTS("alert-acknowledgments.v1", 4, 30),
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/utils/KafkaTopics.java
git commit -m "feat(module6): add Kafka topics for notifications, audit, actions, FHIR writeback, alert state"
```

---

## Task 10: Module6_ClinicalActionEngine — Main Operator

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module6_ClinicalActionEngine.java`

This is the main `KeyedProcessFunction` that wires together the classifier, deduplicator, lifecycle manager, notification router, and side outputs. It also registers Flink processing-time timers for SLA escalation.

- [ ] **Step 1: Create the main operator**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.lifecycle.AlertLifecycleManager;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.routing.NotificationRouter;
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
import java.util.UUID;

/**
 * Module 6: Clinical Action Engine — main operator.
 *
 * Consumes unified ClinicalEvents (from CDS, Pattern, ML sources),
 * classifies into HALT/PAUSE/SOFT_FLAG/ROUTINE, deduplicates across modules,
 * manages alert lifecycle with SLA escalation timers, and distributes
 * output to multiple Kafka sinks via side-output tags.
 */
public class Module6_ClinicalActionEngine
        extends KeyedProcessFunction<String, ClinicalEvent, ClinicalAction> {

    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(Module6_ClinicalActionEngine.class);

    // ── Side-output tags ──
    public static final OutputTag<NotificationRequest> NOTIFICATION_TAG =
        new OutputTag<>("notifications", TypeInformation.of(NotificationRequest.class));
    public static final OutputTag<AuditRecord> AUDIT_TAG =
        new OutputTag<>("audit", TypeInformation.of(AuditRecord.class));
    public static final OutputTag<FhirWriteRequest> FHIR_TAG =
        new OutputTag<>("fhir-writeback", TypeInformation.of(FhirWriteRequest.class));

    // ── State ──
    private transient ValueState<PatientAlertState> alertState;
    private transient ValueState<Module6CrossModuleDedup> dedupState;

    @Override
    public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
        super.open(openContext);

        // Patient alert state with 7-day TTL
        ValueStateDescriptor<PatientAlertState> alertDescriptor =
            new ValueStateDescriptor<>("patient-alert-state", PatientAlertState.class);
        StateTtlConfig ttlConfig = StateTtlConfig
            .newBuilder(Duration.ofDays(7))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();
        alertDescriptor.enableTimeToLive(ttlConfig);
        alertState = getRuntimeContext().getState(alertDescriptor);

        // Dedup state with 7-day TTL
        ValueStateDescriptor<Module6CrossModuleDedup> dedupDescriptor =
            new ValueStateDescriptor<>("dedup-state", Module6CrossModuleDedup.class);
        dedupDescriptor.enableTimeToLive(ttlConfig);
        dedupState = getRuntimeContext().getState(dedupDescriptor);

        LOG.info("Module6_ClinicalActionEngine initialized");
    }

    @Override
    public void processElement(ClinicalEvent event, Context ctx,
                                Collector<ClinicalAction> out) throws Exception {

        String patientId = event.getPatientId();
        if (patientId == null || patientId.isEmpty()) {
            LOG.warn("Dropping event with null patientId");
            return;
        }

        // 1. Classify
        ActionTier tier = Module6ActionClassifier.classify(event);
        if (tier == ActionTier.ROUTINE) return; // no action needed

        // 2. Initialize state
        PatientAlertState patState = alertState.value();
        if (patState == null) patState = new PatientAlertState(patientId);

        Module6CrossModuleDedup dedup = dedupState.value();
        if (dedup == null) dedup = new Module6CrossModuleDedup();

        // 3. Alert fatigue check
        if (AlertLifecycleManager.checkAlertFatigue(patState)) {
            alertState.update(patState);
            return; // suppress — fatigue threshold hit
        }

        // 4. Cross-module dedup
        String clinicalCategory = event.getClinicalCategory();
        if (!dedup.shouldEmit(patientId, tier, clinicalCategory, event.getEventTime())) {
            alertState.update(patState);
            dedupState.update(dedup);
            return; // suppressed by dedup
        }

        // 5. Create alert
        String sourceModule = switch (event.getSource()) {
            case CDS -> "MODULE_3_CDS";
            case PATTERN -> "MODULE_4_CEP";
            case ML_PREDICTION -> "MODULE_5_ML";
        };
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            patientId, tier, clinicalCategory, sourceModule);

        // 6. Register escalation timer if needed
        if (tier.requiresEscalation()) {
            ctx.timerService().registerProcessingTimeTimer(alert.getSlaDeadlineMs());
        }

        // 7. Store in active alerts
        patState.getActiveAlerts().put(clinicalCategory, alert);

        // 8. Emit main output
        out.collect(ClinicalAction.newAlert(alert));

        // 9. Emit notifications via side output
        List<NotificationRequest.Channel> channels = NotificationRouter.getChannels(alert);
        for (NotificationRequest.Channel channel : channels) {
            NotificationRequest notif = new NotificationRequest();
            notif.setNotificationId(UUID.randomUUID().toString());
            notif.setAlertId(alert.getAlertId());
            notif.setPatientId(patientId);
            notif.setChannel(channel);
            notif.setTier(tier);
            notif.setTitle(tier + ": " + clinicalCategory);
            notif.setRequiresAcknowledgment(tier == ActionTier.HALT);
            ctx.output(NOTIFICATION_TAG, notif);
        }

        // 10. Emit audit record
        AuditRecord audit = AuditRecord.alertCreated(alert, event);
        ctx.output(AUDIT_TAG, audit);

        // 11. Update state
        alertState.update(patState);
        dedupState.update(dedup);

        LOG.info("Module6 {} alert: patient={}, category={}, source={}",
            tier, patientId, clinicalCategory, sourceModule);
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx,
                        Collector<ClinicalAction> out) throws Exception {

        PatientAlertState patState = alertState.value();
        if (patState == null) return;

        // Find the alert whose SLA deadline matches this timer
        for (ClinicalAlert alert : patState.getActiveAlerts().values()) {
            if (alert.getState() == AlertState.ACTIVE
                    && alert.getSlaDeadlineMs() == timestamp) {

                AlertLifecycleManager.escalate(alert);
                out.collect(ClinicalAction.escalation(alert, alert.getAssignedTo()));

                // Emit audit for escalation
                AuditRecord audit = new AuditRecord();
                audit.setAuditId(UUID.randomUUID().toString());
                audit.setEventType("ALERT_ESCALATED");
                audit.setPatientId(alert.getPatientId());
                audit.setAlertId(alert.getAlertId());
                audit.setTier(alert.getTier());
                audit.setClinicalCategory(alert.getClinicalCategory());
                ctx.output(AUDIT_TAG, audit);

                // Register next escalation if under max level
                if (alert.getEscalationLevel() < 3 && alert.getTier().requiresEscalation()) {
                    long nextEscalation = switch (alert.getTier()) {
                        case HALT -> timestamp + (90 * 60 * 1000L);     // +90 min
                        case PAUSE -> timestamp + (48 * 60 * 60 * 1000L); // +48 hr
                        default -> -1L;
                    };
                    if (nextEscalation > 0) {
                        ctx.timerService().registerProcessingTimeTimer(nextEscalation);
                        alert.setSlaDeadlineMs(nextEscalation);
                    }
                }
                break;
            }
        }
        alertState.update(patState);
    }
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module6_ClinicalActionEngine.java
git commit -m "feat(module6): implement ClinicalActionEngine KeyedProcessFunction with classification, dedup, lifecycle, notification, audit side-outputs"
```

---

## Task 11: Alert Fatigue Test

**Files:**
- Create: `src/test/java/com/cardiofit/flink/operators/Module6AlertFatigueTest.java`

- [ ] **Step 1: Write fatigue tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.lifecycle.AlertLifecycleManager;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module6AlertFatigueTest {

    @Test
    void alertsUpTo10_notFatigued() {
        PatientAlertState state = new PatientAlertState("P001");
        for (int i = 0; i < 10; i++) {
            assertFalse(AlertLifecycleManager.checkAlertFatigue(state));
        }
        assertEquals(10, state.getAlertsInLast24Hours());
    }

    @Test
    void alert11_isFatigued() {
        PatientAlertState state = new PatientAlertState("P001");
        for (int i = 0; i < 10; i++) {
            AlertLifecycleManager.checkAlertFatigue(state);
        }
        assertTrue(AlertLifecycleManager.checkAlertFatigue(state));
    }

    @Test
    void windowReset_after24Hours() {
        PatientAlertState state = new PatientAlertState("P001");
        state.setAlertsInLast24Hours(10);
        state.setAlertWindowStart(System.currentTimeMillis() - 25 * 60 * 60 * 1000L);
        assertFalse(AlertLifecycleManager.checkAlertFatigue(state));
        assertEquals(1, state.getAlertsInLast24Hours());
    }

    @Test
    void counterIncrements_correctly() {
        PatientAlertState state = new PatientAlertState("P001");
        AlertLifecycleManager.checkAlertFatigue(state);
        assertEquals(1, state.getAlertsInLast24Hours());
        AlertLifecycleManager.checkAlertFatigue(state);
        assertEquals(2, state.getAlertsInLast24Hours());
    }
}
```

- [ ] **Step 2: Run tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module6AlertFatigueTest -q 2>&1 | tail -10`
Expected: All 4 tests PASS (implementation already exists from Task 8)

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/operators/Module6AlertFatigueTest.java
git commit -m "test(module6): add alert fatigue protection tests — 10/24hr cap with window reset"
```

---

## Task 12: Audit Trail Test

**Files:**
- Create: `src/test/java/com/cardiofit/flink/operators/Module6AuditTrailTest.java`

- [ ] **Step 1: Write audit trail tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module6TestBuilder;
import com.cardiofit.flink.lifecycle.AlertLifecycleManager;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module6AuditTrailTest {

    @Test
    void alertCreated_auditHasAllRequiredFields() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        ClinicalEvent event = Module6TestBuilder.sepsisHaltEvent("P001");

        AuditRecord audit = AuditRecord.alertCreated(alert, event);

        assertNotNull(audit.getAuditId(), "auditId required");
        assertEquals("ALERT_CREATED", audit.getEventType());
        assertEquals("P001", audit.getPatientId());
        assertEquals(ActionTier.HALT, audit.getTier());
        assertEquals("SEPSIS", audit.getClinicalCategory());
        assertEquals(alert.getAlertId(), audit.getAlertId());
        assertTrue(audit.getTimestamp() > 0, "timestamp required");
    }

    @Test
    void auditRecord_containsSourceModule() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.PAUSE, "AKI", "MODULE_5_ML");
        ClinicalEvent event = Module6TestBuilder.deteriorationPrediction("P001", 0.50);

        AuditRecord audit = AuditRecord.alertCreated(alert, event);

        assertEquals("MODULE_5_ML", audit.getSourceModule());
    }

    @Test
    void auditRecord_provenance_isNotNull() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "HYPERKALEMIA", "MODULE_3_CDS");
        ClinicalEvent event = Module6TestBuilder.hyperkalemiaHaltEvent("P001");

        AuditRecord audit = AuditRecord.alertCreated(alert, event);

        assertNotNull(audit.getClinicalData(), "clinicalData map must not be null");
        assertNotNull(audit.getInputSnapshot(), "inputSnapshot must not be null");
    }
}
```

- [ ] **Step 2: Run tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module6AuditTrailTest -q 2>&1 | tail -10`
Expected: All 3 tests PASS

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/operators/Module6AuditTrailTest.java
git commit -m "test(module6): add audit trail tests — field completeness and provenance"
```

---

## Task 13: Wire Module 6 into FlinkJobOrchestrator

**Files:**
- Modify: `src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java`

- [ ] **Step 1: Read the existing orchestrator to find the egress-routing case**

Run: `cd backend/shared-infrastructure/flink-processing && grep -n "egress-routing\|module6\|Module6" src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java | head -20`

- [ ] **Step 2: Add a `clinical-action-engine` job type**

Add a new case in the job-type switch that creates a CDS source, pattern source, ML prediction source, unions them into `ClinicalEvent`, keys by patientId, and processes through `Module6_ClinicalActionEngine`. Connect the side outputs to Kafka sinks using the new topic constants.

The specific code depends on the orchestrator's existing patterns — follow the same structure as the existing `egress-routing` and `ml-inference` cases.

- [ ] **Step 3: Verify compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java
git commit -m "feat(module6): wire ClinicalActionEngine into FlinkJobOrchestrator as clinical-action-engine job"
```

---

## Task 14: Run All Module 6 Tests Together

- [ ] **Step 1: Run all Module 6 tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest="Module6*" -q 2>&1 | tail -20`
Expected: All tests PASS (~34 tests across 6 test classes)

- [ ] **Step 2: Run full project compilation**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5`
Expected: BUILD SUCCESS

- [ ] **Step 3: Final commit if any fixes were needed**

```bash
git add -A
git commit -m "fix(module6): resolve any compilation or test issues from integration"
```

---

## Implementation Notes

### What This Plan Does NOT Include (Deferred to Future Work)

1. **Auto-resolution logic** (Section 4.4 of guidelines) — Requires incoming CDS events to be checked against active alert state to detect when conditions resolve. Add after the main engine is stable.

2. **Acknowledgment feedback loop** (Section 11) — The `KeyedCoProcessFunction` that merges `ClinicalEvent + AlertAcknowledgment` streams. Add after the single-stream engine is validated.

3. **FHIR resource JSON generation** (Section 7) — The `FhirWriteRequest` model exists, but generating actual FHIR R4 JSON (RiskAssessment, DetectedIssue, etc.) requires FHIR library integration. Deferred.

4. **Neo4j graph update side-output** — The `EHR_GRAPH_MUTATIONS` topic exists but is currently DISABLED in egress routing. Deferred.

5. **Integration with Module6_EgressRouting** — The existing egress router handles the hybrid 6-sink architecture. Once the clinical action engine is stable, its `ClinicalAction` output can feed into the egress router.

### Key Technical Decisions

| Decision | Rationale |
|----------|-----------|
| Separate `Module6_ClinicalActionEngine` from existing `Module6_AlertComposition` | Backward compat — existing egress routing still works during migration |
| Static `Module6ActionClassifier` with no Flink deps | Fully unit-testable, no harness needed |
| `ClinicalEvent.fromCDS/fromPattern/fromPrediction` factory methods | Type-safe wrapper, impossible to create ambiguous events |
| Dedup windows: HALT=5min, PAUSE=30min, SOFT_FLAG=60min | Per guidelines Section 3.3, tuned for clinical workflow |
| Alert fatigue cap at 10/24hr | Per guidelines Section 10 — safety mechanism, not a tunable knob |
| Processing-time timers for SLA escalation | Fire even when no new events arrive — required for HALT 30-min escalation |
