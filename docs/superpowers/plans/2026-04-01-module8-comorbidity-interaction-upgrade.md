# Module 8: Comorbidity Interaction Detector — Remaining Gaps Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the Module 8 Comorbidity Interaction Detector by implementing the CID alert model/enums, patient comorbidity state model, 17 CID rule evaluators across 3 severity tiers (5 HALT, 5 PAUSE, 7 SOFT_FLAG), the main KeyedProcessFunction operator, alert deduplication/lifecycle management, orchestrator wiring, and all tests. An existing V3 monolith `Module8_ComorbidityInteraction.java` will be superseded.

**Architecture:** Module 8 is a `KeyedProcessFunction<String, CanonicalEvent, CIDAlert>` keyed by `patientId`, consuming enriched patient events from `enriched-patient-events-v1`. Three static rule evaluator classes (zero Flink dependencies) compute clinically validated comorbidity interaction checks. The main operator wires them with Flink keyed state, HALT side-output to `ingestion.safety-critical`, and PAUSE/SOFT_FLAG main output to `alerts.comorbidity-interactions`. Alert deduplication uses a suppression key (rule_id + patient_id + hash(medications_involved)) with a 72-hour suppression window. All threshold comparisons use `>= threshold - 1e-9` for IEEE 754 safety.

**Tech Stack:** Java 17, Flink 2.1.0, Jackson 2.17, Kafka (Confluent Cloud), JUnit 5

---

## Pre-Conditions Verified (2026-04-01)

The following were confirmed against the actual codebase before finalizing this plan:

| Check | Status | Detail |
|---|---|---|
| Java version | **17** | `maven.compiler.source=17` in pom.xml — records supported |
| `CanonicalEvent.java` API | **Confirmed** | `getPatientId()`, `getEventType()` (returns `EventType` enum), `getPayload()` (Map), `getEventTimestamp()` (alias for `getEventTime()`), `getCorrelationId()` |
| `EventType` enum | **Has all needed types** | `VITAL_SIGN`, `LAB_RESULT`, `MEDICATION_ORDERED`, `MEDICATION_ADMINISTERED`, `MEDICATION_DISCONTINUED`, `PATIENT_REPORTED` all present |
| `KafkaTopics` constants | **Both exist** | `ALERTS_COMORBIDITY_INTERACTIONS("alerts.comorbidity-interactions")`, `INGESTION_SAFETY_CRITICAL("ingestion.safety-critical")` |
| Orchestrator wiring | **Case exists** | `case "comorbidity-interaction"` and `case "module8"` already wired to V3 `Module8_ComorbidityInteraction.createComorbidityPipeline(env)` — needs redirect to new engine |
| V3 Module 8 | **549 lines** | Uses V4 CID numbering (CID-06=Hyponatremia, not DD#7 CID-06=Thiazide Glucose). Missing CID-10, CID-11, CID-16. No dedup. Untyped Map input. |
| Existing test | **`Module8_CIDRuleTest.java`** | 224 lines, 17 tests using `CIDRuleEvaluatorTestHelper` — will NOT compile after V3 replacement. Must be deleted or rewritten. |
| `ComorbidityAlert.java` model | **V3 model exists** | Used by V3 operator. Plan introduces `CIDAlert.java` as replacement with suppression key + lifecycle fields. V3 model can be kept until all consumers migrate. |

### eventType Mapping Note

The plan's `updateStateFromEvent()` compares `eventType` as String (`"MEDICATION"`, `"LAB_RESULT"`). But `CanonicalEvent.getEventType()` returns the `EventType` **enum**, not a String. The implementation must use:
```java
EventType type = event.getEventType();
if (type == EventType.MEDICATION_ORDERED || type == EventType.MEDICATION_ADMINISTERED) { ... }
if (type == EventType.LAB_RESULT) { ... }
if (type == EventType.VITAL_SIGN) { ... }
if (type == EventType.PATIENT_REPORTED) { ... }
```

This is a **compile-time safety improvement** over the plan's String comparison — the enum ensures only valid event types are handled.

### Review Fixes Applied (Post-Review, 2026-04-01)

The following fixes were applied after code review. All are incorporated into the code blocks below.

| # | Issue | Fix | Affected Code |
|---|---|---|---|
| R1 | `updateStateFromEvent()` used String comparison — compile blocker | Rewritten to use `EventType` enum, handles `MEDICATION_ORDERED`, `MEDICATION_ADMINISTERED`, `MEDICATION_PRESCRIBED`, `MEDICATION_DISCONTINUED` separately | Task 7 operator |
| R2 | Rolling aggregates (SBP/FBG/weight averages) never populated | Added `rollingBuffers` map with `addToRollingBuffer()`, `getRollingAverage()`, `getValueApproxDaysAgo()`. Evaluators now pass `long eventTime` to computed methods. | Task 1 ComorbidityState, Task 7 operator |
| R3 | Symptom flags were permanently sticky | Added onset timestamps + 72h TTL via `expireStaleSymptoms(now)`. Symptom-resolved events clear flags explicitly. | Task 1 ComorbidityState, Task 7 operator |
| R4 | CID-01 used baseline eGFR decline, not 14-day acute window | Added `eGFR14dAgo` field + `getEGFRAcuteDeclinePercent14d()`. CID-01 now checks 14-day window per DD#7. | Task 1 state, Task 3 HALT evaluator |
| R5 | CID-02 didn't verify rising K+ trajectory | Added `previousPotassium` field. CID-02 now requires `currentK > previousK` (trajectory rising, not just above threshold). | Task 1 state, Task 3 HALT evaluator |
| R6 | CID-09 checked boolean, not 48h nausea persistence | Added `symptomNauseaOnsetTimestamp` + `isNauseaPersistent(now, 48h)`. CID-09 requires 48h duration per DD#7. | Task 1 state, Task 4 PAUSE evaluator |
| R7 | CID-10 had no medication-change guard | Added `lastMedicationChangeTimestamp` + `hadMedicationChangeWithin()`. CID-10 skips if meds changed within 14 days. | Task 1 state, Task 4 PAUSE evaluator |
| R8 | Suppression key used fragile no-delimiter concatenation | Changed to pipe-delimited: `"drug1|drug2"` before hashing. | Task 1 CIDAlert |
| R9 | Integration tests didn't test actual `processElement()` | Added `Module8ProcessElementTest` that verifies the full wiring including state update, rule evaluation, and side-output routing. | Task 8 |
| R10 | Orchestrator launcher code was missing | Added explicit `launchComorbidityEngine()` with dual-sink wiring (HALT side-output + main output). | Task 8 |

**Evaluator Signature Change:** All three evaluators now accept `long eventTime` parameter:
- `Module8HALTEvaluator.evaluate(state, eventTime)` — needed for `getWeightDelta7d(now)`, `getEGFRAcuteDeclinePercent14d()`
- `Module8PAUSEEvaluator.evaluate(state, eventTime)` — needed for `getFBGDelta14d(now)`, `isNauseaPersistent()`, `hadMedicationChangeWithin()`
- `Module8SOFTFLAGEvaluator.evaluate(state, sbpTarget)` — unchanged (no time-dependent fields)

**Test Builder Migration:** All test builders that previously called `setFbgSevenDayAvg()`, `setWeightSevenDaysAgo()`, etc. must use `addToRollingBuffer("fbg", value, timestamp)` with explicit timestamps. A helper `Module8TestBuilder.daysAgo(int days)` provides relative timestamps for convenience.

---

## CRITICAL: CID Rule Set Reconciliation

Two different CID rule numbering schemes exist across design documents. **This must be reconciled before coding.**

### DD#7 (Original Deep Dive — Authoritative Clinical Spec)

The Deep Dive #7 document contains full trigger conditions, clinical danger explanations, alert text templates, and recommended actions for all 17 rules. **This is the source of truth for implementation.**

| ID | Name | Severity | Trigger Summary |
|---|---|---|---|
| CID-01 | Triple Whammy AKI | HALT | ACEi/ARB + SGLT2i + diuretic + precipitant (weight drop >2kg/7d OR eGFR drop >15%/14d OR illness) |
| CID-02 | Hyperkalemia Cascade | HALT | ACEi/ARB max dose + finerenone + K+ >5.3 rising |
| CID-03 | Hypoglycemia Masking | HALT | Insulin/SU + beta-blocker + glucose <60 + no symptom report |
| CID-04 | Euglycemic DKA | HALT | SGLT2i + (nausea/vomiting OR keto diet OR insulin reduction in LADA) |
| CID-05 | Severe Hypotension Risk | HALT | ≥3 antihypertensives + SGLT2i + SBP <95 OR SBP drop >30 from 7d avg |
| CID-06 | Thiazide Glucose Worsening | PAUSE | Thiazide + FBG 7d avg increased >15 mg/dL in 14 days |
| CID-07 | ACEi + Sustained eGFR Decline | PAUSE | ACEi/ARB + eGFR drop >25% from baseline + >6 weeks since initiation |
| CID-08 | Statin Myopathy | PAUSE | Statin + new muscle pain/weakness (patient-reported) |
| CID-09 | GLP-1RA + Acute GI + Dehydration | PAUSE | GLP-1RA + persistent nausea/vomiting >48h + weight drop >1.5kg/7d |
| CID-10 | Concurrent Glucose AND BP Deterioration | PAUSE | Both FBG and SBP worsening trajectories without medication change |
| CID-11 | Genital Infection + SGLT2i | SOFT_FLAG | History of genital infection + SGLT2i prescribed/recommended |
| CID-12 | Polypharmacy Burden | SOFT_FLAG | ≥8 daily medications + new medication recommended |
| CID-13 | Elderly + Intensive BP Target | SOFT_FLAG | Age ≥75 + SBP target <130 + (eGFR <45 OR falls history OR orthostatic hypotension) |
| CID-14 | Metformin Near eGFR Threshold | SOFT_FLAG | Metformin + eGFR 30-35 + eGFR trajectory DECLINING |
| CID-15 | SGLT2i + NSAID Use | SOFT_FLAG | SGLT2i + NSAIDs in medication list |
| CID-16 | Salt-Sensitive + Sodium-Retaining Drug | SOFT_FLAG | HIGH_SALT_SENSITIVITY phenotype + sodium-retaining drug (fludrocortisone, corticosteroids) |
| CID-17 | SGLT2i + Fasting Period | SOFT_FLAG | SGLT2i + active fasting period (Ramadan, Navratri) + duration >16h |

### V4 Architecture Summary (Later Document — Different Numbering)

The V4 architecture overview renumbered PAUSE and SOFT_FLAG rules with different clinical scenarios:

| ID (V4) | Name (V4) | Severity | Conflict with DD#7? |
|---|---|---|---|
| CID-06 | Severe Hyponatremia | PAUSE | **YES** — DD#7 CID-06 is Thiazide Glucose Worsening |
| CID-07 | Recurrent Hypoglycemia | PAUSE | **YES** — DD#7 CID-07 is ACEi + eGFR Decline |
| CID-08 | Volume Depletion | PAUSE | **YES** — DD#7 CID-08 is Statin Myopathy |
| CID-09 | Heart Rate Masking | PAUSE | **YES** — DD#7 CID-09 is GLP-1RA + GI |
| CID-11 | Metformin Dose Cap | SOFT_FLAG | **YES** — DD#7 CID-11 is Genital Infection + SGLT2i |
| CID-12 | Statin-Fibrate Myopathy | SOFT_FLAG | **YES** — DD#7 CID-12 is Polypharmacy Burden |
| CID-13 | Expected eGFR Dip | SOFT_FLAG | **YES** — DD#7 CID-13 is Elderly + Intensive BP |
| CID-14 | Triple Antithrombotic | SOFT_FLAG | **YES** — DD#7 CID-14 is Metformin Near eGFR |
| CID-15 | NSAID-RASi Renal Risk | SOFT_FLAG | **YES** — DD#7 CID-15 is SGLT2i + NSAID |

### Resolution

**Follow DD#7 for CID-01 through CID-17.** The DD#7 document has complete trigger conditions, clinical rationale, alert text, and recommended actions — it is the authoritative clinical specification. The V4 architecture document was a later summary that reorganized rules for a high-level overview. The V4-specific rules (Severe Hyponatremia, Recurrent Hypoglycemia, Volume Depletion, Heart Rate Masking, Metformin Dose Cap, Expected eGFR Dip) should be incorporated as **additional rules CID-18 through CID-23** in a future phase, alongside the elderly polypharmacy extensions (CID-20 through CID-23 from the enhancement document).

**Phase 1 implementation: DD#7 CID-01 through CID-17 (this plan).**
**Phase 2 deferred: V4-specific rules + elderly extensions (CID-18+).**

---

## Already Implemented (Existing Infrastructure)

### Existing Files — Verify Before Starting

Unlike Module 7, Module 8 does NOT have a pre-built data foundation of model/enum files. The following models from Module 7 and the existing codebase may be relevant:

| File | Status | Notes |
|---|---|---|
| `Module8_ComorbidityInteraction.java` | **REPLACE** | V3 monolith — will be superseded by `Module8_ComorbidityEngine.java` |
| `KafkaTopics.java` | **KEEP** | Verify `ALERTS_COMORBIDITY_INTERACTIONS` and `INGESTION_SAFETY_CRITICAL` constants exist |
| `FlinkJobOrchestrator.java` | **MODIFY** | Add Module 8 case to the job-type switch |
| `CanonicalEvent.java` (Module 1/1b output) | **READ ONLY** | The input event model — verify field accessors |

### Input Contract: CanonicalEvent / EnrichedPatientContext

Module 8 consumes from `enriched-patient-events-v1`. Each event is a `CanonicalEvent` carrying:
- `patientId` (String) — keying field
- `eventType` (Enum: VITAL_SIGN, LAB_RESULT, MEDICATION, PATIENT_REPORTED, etc.)
- `payload` (Map<String, Object>) — contains clinical values
- `sourceSystem` (String)
- `eventTimestamp` (long)
- `correlationId` (String)

**Critical pre-task:** Read the on-disk `CanonicalEvent.java` to confirm exact field names and accessor methods. The payload extraction for medications, labs, and vitals will differ based on the actual on-disk schema.

---

## File Structure (New Files Only)

```
backend/shared-infrastructure/flink-processing/src/
├── main/java/com/cardiofit/flink/
│   ├── models/
│   │   ├── CIDSeverity.java               ← Task 1: HALT/PAUSE/SOFT_FLAG enum
│   │   ├── CIDRuleId.java                  ← Task 1: CID_01 through CID_17 enum
│   │   ├── CIDAlert.java                   ← Task 1: Alert output model (14 fields)
│   │   └── ComorbidityState.java           ← Task 1: Per-patient keyed state
│   └── operators/
│       ├── Module8HALTEvaluator.java        ← Task 3: CID-01 through CID-05
│       ├── Module8PAUSEEvaluator.java       ← Task 4: CID-06 through CID-10
│       ├── Module8SOFTFLAGEvaluator.java    ← Task 5: CID-11 through CID-17
│       ├── Module8SuppressionManager.java   ← Task 6: Dedup + lifecycle
│       └── Module8_ComorbidityEngine.java   ← Task 7: Main KeyedProcessFunction
└── test/java/com/cardiofit/flink/
    ├── builders/
    │   └── Module8TestBuilder.java          ← Task 2: Test data factory
    └── operators/
        ├── Module8HALTRulesTest.java         ← Task 3: HALT rule tests
        ├── Module8PAUSERulesTest.java        ← Task 4: PAUSE rule tests
        ├── Module8SOFTFLAGRulesTest.java     ← Task 5: SOFT_FLAG rule tests
        ├── Module8SuppressionTest.java       ← Task 6: Dedup tests
        └── Module8IntegrationTest.java       ← Task 8: Full-flow integration
```

---

## Task 0: Pre-Implementation Verification

Before writing any code, verify the on-disk models that Module 8 depends on.

- [ ] **Step 1: Verify CanonicalEvent API surface**

```bash
cd backend/shared-infrastructure/flink-processing
grep -n "public.*get\|public.*set\|public.*is" \
  src/main/java/com/cardiofit/flink/models/CanonicalEvent.java | head -40
```

Confirm these accessors exist (or their equivalents):
- `getPatientId()` → String
- `getEventType()` → String or Enum
- `getPayload()` → Map<String, Object> or JsonNode
- `getEventTimestamp()` → long
- `getCorrelationId()` → String
- `getSourceSystem()` → String

If the input model is NOT `CanonicalEvent` but `EnrichedPatientContext` (Module 2 output), adjust all references in this plan.

- [ ] **Step 2: Verify KafkaTopics constants**

```bash
grep -n "COMORBIDITY\|SAFETY_CRITICAL\|comorbidity\|safety-critical" \
  src/main/java/com/cardiofit/flink/utils/KafkaTopics.java
```

Expected: Constants for `alerts.comorbidity-interactions` and `ingestion.safety-critical`. If missing, add them.

- [ ] **Step 3: Verify existing V3 Module 8 to understand what is being superseded**

```bash
wc -l src/main/java/com/cardiofit/flink/operators/Module8_ComorbidityInteraction.java
head -50 src/main/java/com/cardiofit/flink/operators/Module8_ComorbidityInteraction.java
```

Document V3 approach (likely Map<String,Object> based, CSV-encoded state) for reference during migration.

- [ ] **Step 4: Verify Java source/target version supports records**

```bash
grep -n "<source>\|<target>\|<release>\|<maven.compiler" pom.xml | head -10
```

Expected: Java 17 (required for `record` syntax in CIDAlert and WhiteCoatResult-style records). If Java 11, use traditional classes with constructors.

---

## Task 1: Model and Enum Files

**Files to create:**
- `src/main/java/com/cardiofit/flink/models/CIDSeverity.java`
- `src/main/java/com/cardiofit/flink/models/CIDRuleId.java`
- `src/main/java/com/cardiofit/flink/models/CIDAlert.java`
- `src/main/java/com/cardiofit/flink/models/ComorbidityState.java`

- [ ] **Step 1: Create CIDSeverity enum**

```java
package com.cardiofit.flink.models;

/**
 * CID alert severity levels.
 *
 * HALT: Life-threatening. Immediate physician notification (<5 min SLA).
 *       Side-output to ingestion.safety-critical. All Decision Cards paused.
 *
 * PAUSE: Correction loop paused. Physician review within 48 hours.
 *        Main output to alerts.comorbidity-interactions.
 *
 * SOFT_FLAG: Warning attached to Decision Cards. No pause.
 *            Main output to alerts.comorbidity-interactions.
 */
public enum CIDSeverity {
    HALT,
    PAUSE,
    SOFT_FLAG;

    /**
     * Whether this severity level requires immediate safety-critical routing.
     */
    public boolean isSafetyCritical() {
        return this == HALT;
    }

    /**
     * Whether this severity level pauses the correction loop.
     */
    public boolean pausesCorrectionLoop() {
        return this == HALT || this == PAUSE;
    }
}
```

- [ ] **Step 2: Create CIDRuleId enum**

```java
package com.cardiofit.flink.models;

/**
 * Canonical CID rule identifiers.
 * Follows DD#7 authoritative numbering.
 *
 * HALT rules (CID-01 to CID-05): Life-threatening interactions.
 * PAUSE rules (CID-06 to CID-10): Requires physician review.
 * SOFT_FLAG rules (CID-11 to CID-17): Informational warnings.
 */
public enum CIDRuleId {
    // HALT
    CID_01("Triple Whammy AKI", CIDSeverity.HALT),
    CID_02("Hyperkalemia Cascade", CIDSeverity.HALT),
    CID_03("Hypoglycemia Masking", CIDSeverity.HALT),
    CID_04("Euglycemic DKA", CIDSeverity.HALT),
    CID_05("Severe Hypotension Risk", CIDSeverity.HALT),

    // PAUSE
    CID_06("Thiazide Glucose Worsening", CIDSeverity.PAUSE),
    CID_07("ACEi Sustained eGFR Decline", CIDSeverity.PAUSE),
    CID_08("Statin Myopathy", CIDSeverity.PAUSE),
    CID_09("GLP1RA GI Dehydration", CIDSeverity.PAUSE),
    CID_10("Concurrent Glucose BP Deterioration", CIDSeverity.PAUSE),

    // SOFT_FLAG
    CID_11("Genital Infection SGLT2i", CIDSeverity.SOFT_FLAG),
    CID_12("Polypharmacy Burden", CIDSeverity.SOFT_FLAG),
    CID_13("Elderly Intensive BP Target", CIDSeverity.SOFT_FLAG),
    CID_14("Metformin Near eGFR Threshold", CIDSeverity.SOFT_FLAG),
    CID_15("SGLT2i NSAID Use", CIDSeverity.SOFT_FLAG),
    CID_16("Salt Sensitive Sodium Retaining Drug", CIDSeverity.SOFT_FLAG),
    CID_17("SGLT2i Fasting Period", CIDSeverity.SOFT_FLAG);

    private final String displayName;
    private final CIDSeverity severity;

    CIDRuleId(String displayName, CIDSeverity severity) {
        this.displayName = displayName;
        this.severity = severity;
    }

    public String getDisplayName() { return displayName; }
    public CIDSeverity getSeverity() { return severity; }
}
```

- [ ] **Step 3: Create CIDAlert output model**

```java
package com.cardiofit.flink.models;

import java.util.List;
import java.util.UUID;

/**
 * CID Alert event — output model for Module 8.
 * 14 fields per DD#7 Section 4 alert event schema.
 *
 * All enum fields are String-typed for Kafka JSON serialization
 * (same pattern as BPVariabilityMetrics).
 */
public class CIDAlert implements java.io.Serializable {
    private static final long serialVersionUID = 1L;

    // Identity
    private String alertId;          // UUID v7
    private String patientId;

    // Rule
    private String ruleId;           // CID_01 through CID_17
    private String severity;         // HALT / PAUSE / SOFT_FLAG
    private String triggerSummary;   // Human-readable trigger description

    // Clinical context
    private List<String> medicationsInvolved;    // Drug names in the interaction
    private String labValuesInvolved;            // JSON: relevant labs at trigger time
    private String vitalsInvolved;               // JSON: relevant vitals
    private String recommendedAction;            // Deterministic recommendation from rule

    // Deduplication
    private String suppressionKey;   // ruleId + patientId + hash(medications)

    // Lifecycle
    private long createdAt;
    private Long resolvedAt;         // nullable — set when acknowledged/resolved
    private String resolution;       // nullable — PHYSICIAN_ACKNOWLEDGED / PHYSICIAN_ACTIONED / AUTO_RESOLVED / EXPIRED

    // Provenance
    private String correlationId;    // From triggering CanonicalEvent

    // --- Constructors ---
    public CIDAlert() {}

    public static CIDAlert create(CIDRuleId rule, String patientId,
                                   String triggerSummary, List<String> medications,
                                   String recommendedAction, String correlationId) {
        CIDAlert alert = new CIDAlert();
        alert.alertId = UUID.randomUUID().toString();
        alert.patientId = patientId;
        alert.ruleId = rule.name();
        alert.severity = rule.getSeverity().name();
        alert.triggerSummary = triggerSummary;
        alert.medicationsInvolved = medications;
        alert.recommendedAction = recommendedAction;
        alert.correlationId = correlationId;
        alert.createdAt = System.currentTimeMillis();

        // R8 fix: Suppression key uses pipe-delimited meds for deterministic hashing.
        // Without delimiter, "ACEI"+"SGLT2I" and "ACEISGLT"+"2I" hash identically.
        String medsHash = medications == null || medications.isEmpty() ? "none"
            : String.valueOf(medications.stream().sorted()
                .collect(java.util.stream.Collectors.joining("|")).hashCode());
        alert.suppressionKey = rule.name() + ":" + patientId + ":" + medsHash;

        return alert;
    }

    // --- All getters and setters ---
    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public String getRuleId() { return ruleId; }
    public void setRuleId(String ruleId) { this.ruleId = ruleId; }
    public String getSeverity() { return severity; }
    public void setSeverity(String severity) { this.severity = severity; }
    public String getTriggerSummary() { return triggerSummary; }
    public void setTriggerSummary(String triggerSummary) { this.triggerSummary = triggerSummary; }
    public List<String> getMedicationsInvolved() { return medicationsInvolved; }
    public void setMedicationsInvolved(List<String> medicationsInvolved) { this.medicationsInvolved = medicationsInvolved; }
    public String getLabValuesInvolved() { return labValuesInvolved; }
    public void setLabValuesInvolved(String labValuesInvolved) { this.labValuesInvolved = labValuesInvolved; }
    public String getVitalsInvolved() { return vitalsInvolved; }
    public void setVitalsInvolved(String vitalsInvolved) { this.vitalsInvolved = vitalsInvolved; }
    public String getRecommendedAction() { return recommendedAction; }
    public void setRecommendedAction(String recommendedAction) { this.recommendedAction = recommendedAction; }
    public String getSuppressionKey() { return suppressionKey; }
    public void setSuppressionKey(String suppressionKey) { this.suppressionKey = suppressionKey; }
    public long getCreatedAt() { return createdAt; }
    public void setCreatedAt(long createdAt) { this.createdAt = createdAt; }
    public Long getResolvedAt() { return resolvedAt; }
    public void setResolvedAt(Long resolvedAt) { this.resolvedAt = resolvedAt; }
    public String getResolution() { return resolution; }
    public void setResolution(String resolution) { this.resolution = resolution; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String correlationId) { this.correlationId = correlationId; }
}
```

- [ ] **Step 4: Create ComorbidityState — per-patient keyed state**

```java
package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.*;

/**
 * Per-patient comorbidity state maintained in Flink keyed state.
 *
 * Tracks: active medications, recent lab values, recent vitals,
 * patient demographics, and alert suppression history.
 *
 * This is the "memory" of Module 8 — each incoming event updates
 * the relevant slice of state, and all 17 rules evaluate against
 * the current snapshot.
 */
public class ComorbidityState implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;

    // --- Active Medications ---
    // Key: drug name (lowercase), Value: MedicationEntry
    private Map<String, MedicationEntry> activeMedications = new LinkedHashMap<>();

    // --- Recent Lab Values ---
    // Key: lab type (e.g., "eGFR", "potassium", "FBG", "sodium", "CK", "HbA1c")
    // Value: LabEntry with value + timestamp
    private Map<String, LabEntry> recentLabs = new LinkedHashMap<>();

    // --- Recent Vitals (point-in-time) ---
    private Double latestSBP;
    private Double latestDBP;
    private Double latestWeight;

    // --- Rolling Buffers (for computing averages and deltas) ---
    // Each buffer holds timestamped readings, pruned to 14-day window.
    // Key: metric name ("sbp", "fbg", "weight"), Value: list of TimestampedValue
    private Map<String, List<TimestampedValue>> rollingBuffers = new LinkedHashMap<>();

    // --- Medication Change Tracking ---
    private long lastMedicationChangeTimestamp;  // for CID-10 guard

    // --- Patient Demographics ---
    private Integer age;
    private Double latestGlucose;

    // --- Symptom Flags with Onset Timestamps ---
    // Symptoms are time-bounded: expire after TTL unless refreshed.
    // Prevents permanently sticky flags from historical reports.
    private boolean symptomReportedHypoglycemia;
    private long symptomHypoglycemiaTimestamp;     // onset time
    private boolean symptomReportedMusclePain;
    private long symptomMusclePainTimestamp;
    private boolean symptomReportedNauseaVomiting;
    private long symptomNauseaOnsetTimestamp;      // for CID-09 48h check
    private boolean symptomReportedKetoDiet;

    // --- Patient History Flags ---
    private boolean genitalInfectionHistory;       // for CID-11
    private boolean fallsHistory;                  // for CID-13
    private boolean orthostaticHypotension;        // for CID-13
    private String saltSensitivityPhenotype;        // HIGH / LOW / UNDETERMINED
    private boolean activeFastingPeriod;            // for CID-17
    private int activeFastingDurationHours;         // for CID-17

    // --- eGFR Trajectory ---
    private Double eGFRBaseline;         // pre-RASi initiation
    private Long eGFRBaselineTimestamp;
    private Double eGFRCurrent;
    private Long eGFRCurrentTimestamp;
    private Double eGFR14dAgo;           // for CID-01 acute 14-day decline

    // --- Potassium Trajectory ---
    private Double previousPotassium;    // for CID-02 rising check

    // --- Suppression History ---
    // Key: suppressionKey, Value: timestamp of last alert emission
    private Map<String, Long> suppressionHistory = new LinkedHashMap<>();

    // --- Timestamps ---
    private long lastUpdated;
    private long totalEventsProcessed;

    // --- Constructors ---
    public ComorbidityState() {}

    public ComorbidityState(String patientId) {
        this.patientId = patientId;
        this.lastUpdated = System.currentTimeMillis();
    }

    // --- Medication Helpers ---

    public void addMedication(String drugName, String drugClass, Double doseMg) {
        activeMedications.put(drugName.toLowerCase(),
            new MedicationEntry(drugName, drugClass, doseMg, System.currentTimeMillis()));
    }

    public void removeMedication(String drugName) {
        activeMedications.remove(drugName.toLowerCase());
    }

    public boolean hasDrugClass(String drugClass) {
        return activeMedications.values().stream()
            .anyMatch(m -> drugClass.equalsIgnoreCase(m.drugClass));
    }

    public boolean hasDrugClasses(String... drugClasses) {
        for (String dc : drugClasses) {
            if (!hasDrugClass(dc)) return false;
        }
        return true;
    }

    public boolean hasAnyDrugClass(String... drugClasses) {
        for (String dc : drugClasses) {
            if (hasDrugClass(dc)) return true;
        }
        return false;
    }

    public int countDrugClass(String drugClass) {
        return (int) activeMedications.values().stream()
            .filter(m -> drugClass.equalsIgnoreCase(m.drugClass))
            .count();
    }

    public int getActiveMedicationCount() {
        return activeMedications.size();
    }

    public List<String> getMedicationsByClass(String drugClass) {
        List<String> result = new ArrayList<>();
        for (MedicationEntry m : activeMedications.values()) {
            if (drugClass.equalsIgnoreCase(m.drugClass)) {
                result.add(m.drugName);
            }
        }
        return result;
    }

    // --- Lab Helpers ---

    public void updateLab(String labType, double value) {
        recentLabs.put(labType.toLowerCase(),
            new LabEntry(labType, value, System.currentTimeMillis()));
    }

    public Double getLabValue(String labType) {
        LabEntry entry = recentLabs.get(labType.toLowerCase());
        return entry != null ? entry.value : null;
    }

    public boolean hasLab(String labType) {
        return recentLabs.containsKey(labType.toLowerCase());
    }

    // --- Suppression ---

    public boolean isSuppressed(String suppressionKey, long currentTime) {
        Long lastEmission = suppressionHistory.get(suppressionKey);
        if (lastEmission == null) return false;
        return (currentTime - lastEmission) < 72 * 60 * 60 * 1000L; // 72 hours
    }

    public void recordSuppression(String suppressionKey, long currentTime) {
        suppressionHistory.put(suppressionKey, currentTime);
        // Evict entries older than 7 days to prevent unbounded growth
        suppressionHistory.entrySet().removeIf(
            e -> (currentTime - e.getValue()) > 7 * 24 * 60 * 60 * 1000L);
    }

    // --- Rolling Buffer Methods ---

    /**
     * Add a timestamped reading to a named rolling buffer.
     * Prunes entries older than 14 days on each insert.
     */
    public void addToRollingBuffer(String metric, double value, long timestamp) {
        List<TimestampedValue> buffer = rollingBuffers.computeIfAbsent(
            metric.toLowerCase(), k -> new ArrayList<>());
        buffer.add(new TimestampedValue(value, timestamp));
        // Prune >14 days
        long cutoff = timestamp - 14L * 86400000L;
        buffer.removeIf(tv -> tv.timestamp < cutoff);
    }

    /**
     * Compute average of readings in the last N days from a rolling buffer.
     */
    public Double getRollingAverage(String metric, long now, int days) {
        List<TimestampedValue> buffer = rollingBuffers.get(metric.toLowerCase());
        if (buffer == null || buffer.isEmpty()) return null;
        long cutoff = now - (long) days * 86400000L;
        double sum = 0; int count = 0;
        for (TimestampedValue tv : buffer) {
            if (tv.timestamp >= cutoff) { sum += tv.value; count++; }
        }
        return count > 0 ? sum / count : null;
    }

    /**
     * Get the oldest reading value in a buffer that falls within the [daysAgo-1, daysAgo+1] window.
     * Used for "value N days ago" comparisons (e.g., FBG 14 days ago).
     */
    public Double getValueApproxDaysAgo(String metric, long now, int daysAgo) {
        List<TimestampedValue> buffer = rollingBuffers.get(metric.toLowerCase());
        if (buffer == null || buffer.isEmpty()) return null;
        long target = now - (long) daysAgo * 86400000L;
        long window = 86400000L; // +/- 1 day tolerance
        TimestampedValue closest = null;
        long closestDelta = Long.MAX_VALUE;
        for (TimestampedValue tv : buffer) {
            long delta = Math.abs(tv.timestamp - target);
            if (delta <= window && delta < closestDelta) {
                closest = tv; closestDelta = delta;
            }
        }
        return closest != null ? closest.value : null;
    }

    // --- Computed Aggregates (derived from rolling buffers) ---

    /** SBP 7-day average. */
    public Double getSbpSevenDayAvg(long now) {
        return getRollingAverage("sbp", now, 7);
    }

    /** FBG 7-day average. */
    public Double getFbgSevenDayAvg(long now) {
        return getRollingAverage("fbg", now, 7);
    }

    /** Weight approximately 7 days ago (for delta). */
    public Double getWeightApprox7dAgo(long now) {
        return getValueApproxDaysAgo("weight", now, 7);
    }

    /** Weight delta: latest - 7d ago. Negative = weight loss. */
    public Double getWeightDelta7d(long now) {
        Double w7d = getWeightApprox7dAgo(now);
        if (latestWeight == null || w7d == null) return null;
        return latestWeight - w7d;
    }

    // --- FBG Delta (14 days) ---

    public Double getFBGDelta14d(long now) {
        Double fbg7d = getFbgSevenDayAvg(now);
        Double fbg14d = getValueApproxDaysAgo("fbg", now, 14);
        if (fbg7d == null || fbg14d == null) return null;
        return fbg7d - fbg14d;
    }

    // --- SBP Delta (14 days) ---

    public Double getSbpDelta14d(long now) {
        Double sbp7d = getSbpSevenDayAvg(now);
        Double sbp14d = getValueApproxDaysAgo("sbp", now, 14);
        if (sbp7d == null || sbp14d == null) return null;
        return sbp7d - sbp14d;
    }

    // --- eGFR 14-Day Acute Decline (for CID-01) ---

    public Double getEGFRAcuteDeclinePercent14d() {
        if (eGFR14dAgo == null || eGFRCurrent == null || eGFR14dAgo < 1e-9) return null;
        return ((eGFR14dAgo - eGFRCurrent) / eGFR14dAgo) * 100.0;
    }

    // --- Symptom Expiry ---

    private static final long SYMPTOM_TTL_MS = 72L * 60 * 60 * 1000; // 72h default

    /**
     * Expire symptom flags older than TTL. Called on each event.
     * Prevents permanently sticky flags from historical reports.
     */
    public void expireStaleSymptoms(long now) {
        if (symptomReportedHypoglycemia && symptomHypoglycemiaTimestamp > 0
                && (now - symptomHypoglycemiaTimestamp) > SYMPTOM_TTL_MS) {
            symptomReportedHypoglycemia = false;
        }
        if (symptomReportedMusclePain && symptomMusclePainTimestamp > 0
                && (now - symptomMusclePainTimestamp) > SYMPTOM_TTL_MS) {
            symptomReportedMusclePain = false;
        }
        if (symptomReportedNauseaVomiting && symptomNauseaOnsetTimestamp > 0
                && (now - symptomNauseaOnsetTimestamp) > SYMPTOM_TTL_MS) {
            symptomReportedNauseaVomiting = false;
        }
        // ketoDiet does not auto-expire — requires explicit RESOLVED event
    }

    /** Check if nausea has persisted for at least the given duration. */
    public boolean isNauseaPersistent(long now, long minDurationMs) {
        if (!symptomReportedNauseaVomiting || symptomNauseaOnsetTimestamp <= 0) return false;
        return (now - symptomNauseaOnsetTimestamp) >= minDurationMs;
    }

    /** Check if medication was changed within the given window. */
    public boolean hadMedicationChangeWithin(long now, long windowMs) {
        return lastMedicationChangeTimestamp > 0
            && (now - lastMedicationChangeTimestamp) < windowMs;
    }

    // --- eGFR Decline (from baseline, for CID-07) ---

    /**
     * Compute eGFR percentage decline from baseline.
     * @return percentage decline (positive = declining), or null if no baseline
     */
    public Double getEGFRDeclinePercent() {
        if (eGFRBaseline == null || eGFRCurrent == null || eGFRBaseline < 1e-9) return null;
        return ((eGFRBaseline - eGFRCurrent) / eGFRBaseline) * 100.0;
    }

    /**
     * Weeks since eGFR baseline was established.
     */
    public Double getWeeksSinceEGFRBaseline() {
        if (eGFRBaselineTimestamp == null || eGFRCurrentTimestamp == null) return null;
        long deltaMs = eGFRCurrentTimestamp - eGFRBaselineTimestamp;
        return deltaMs / (7.0 * 24 * 60 * 60 * 1000L);
    }

    // --- Count of antihypertensives ---

    public int countAntihypertensives() {
        int count = 0;
        String[] ahClasses = {"ACEI", "ARB", "CCB", "THIAZIDE", "LOOP_DIURETIC",
            "BETA_BLOCKER", "ALPHA_BLOCKER", "MINERALOCORTICOID_ANTAGONIST"};
        for (String cls : ahClasses) {
            count += countDrugClass(cls);
        }
        return count;
    }

    // --- Inner classes ---

    public static class TimestampedValue implements Serializable {
        private static final long serialVersionUID = 1L;
        public double value;
        public long timestamp;
        public TimestampedValue() {}
        public TimestampedValue(double value, long timestamp) {
            this.value = value; this.timestamp = timestamp;
        }
    }

    public static class MedicationEntry implements Serializable {
        private static final long serialVersionUID = 1L;
        public String drugName;
        public String drugClass;   // ACEI, ARB, SGLT2I, BETA_BLOCKER, INSULIN, SU, GLP1RA, STATIN, THIAZIDE, etc.
        public Double doseMg;
        public long addedTimestamp;

        public MedicationEntry() {}
        public MedicationEntry(String drugName, String drugClass, Double doseMg, long ts) {
            this.drugName = drugName;
            this.drugClass = drugClass;
            this.doseMg = doseMg;
            this.addedTimestamp = ts;
        }
    }

    public static class LabEntry implements Serializable {
        private static final long serialVersionUID = 1L;
        public String labType;
        public double value;
        public long timestamp;

        public LabEntry() {}
        public LabEntry(String labType, double value, long ts) {
            this.labType = labType;
            this.value = value;
            this.timestamp = ts;
        }
    }

    // --- Standard getters and setters for all fields ---
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Map<String, MedicationEntry> getActiveMedications() { return activeMedications; }
    public void setActiveMedications(Map<String, MedicationEntry> m) { this.activeMedications = m; }
    public Map<String, LabEntry> getRecentLabs() { return recentLabs; }
    public void setRecentLabs(Map<String, LabEntry> l) { this.recentLabs = l; }
    public Double getLatestSBP() { return latestSBP; }
    public void setLatestSBP(Double v) { this.latestSBP = v; }
    public Double getLatestDBP() { return latestDBP; }
    public void setLatestDBP(Double v) { this.latestDBP = v; }
    public Double getLatestWeight() { return latestWeight; }
    public void setLatestWeight(Double v) { this.latestWeight = v; }
    public long getLastMedicationChangeTimestamp() { return lastMedicationChangeTimestamp; }
    public void setLastMedicationChangeTimestamp(long v) { this.lastMedicationChangeTimestamp = v; }
    public Double getPreviousPotassium() { return previousPotassium; }
    public void setPreviousPotassium(Double v) { this.previousPotassium = v; }
    public Double getEGFR14dAgo() { return eGFR14dAgo; }
    public void setEGFR14dAgo(Double v) { this.eGFR14dAgo = v; }
    public long getSymptomNauseaOnsetTimestamp() { return symptomNauseaOnsetTimestamp; }
    public void setSymptomNauseaOnsetTimestamp(long v) { this.symptomNauseaOnsetTimestamp = v; }
    public long getSymptomHypoglycemiaTimestamp() { return symptomHypoglycemiaTimestamp; }
    public void setSymptomHypoglycemiaTimestamp(long v) { this.symptomHypoglycemiaTimestamp = v; }
    public long getSymptomMusclePainTimestamp() { return symptomMusclePainTimestamp; }
    public void setSymptomMusclePainTimestamp(long v) { this.symptomMusclePainTimestamp = v; }
    // Removed: fbgSevenDayAvg, fbgFourteenDaysAgo, sbpSevenDayAvg, sbpFourteenDaysAgo,
    // weightSevenDaysAgo — these are now computed dynamically from rolling buffers.
    // Use: getSbpSevenDayAvg(now), getFBGDelta14d(now), getWeightDelta7d(now), etc.
    public Integer getAge() { return age; }
    public void setAge(Integer age) { this.age = age; }
    public Double getLatestGlucose() { return latestGlucose; }
    public void setLatestGlucose(Double v) { this.latestGlucose = v; }
    public boolean isSymptomReportedHypoglycemia() { return symptomReportedHypoglycemia; }
    public void setSymptomReportedHypoglycemia(boolean v) { this.symptomReportedHypoglycemia = v; }
    public boolean isSymptomReportedMusclePain() { return symptomReportedMusclePain; }
    public void setSymptomReportedMusclePain(boolean v) { this.symptomReportedMusclePain = v; }
    public boolean isSymptomReportedNauseaVomiting() { return symptomReportedNauseaVomiting; }
    public void setSymptomReportedNauseaVomiting(boolean v) { this.symptomReportedNauseaVomiting = v; }
    public boolean isSymptomReportedKetoDiet() { return symptomReportedKetoDiet; }
    public void setSymptomReportedKetoDiet(boolean v) { this.symptomReportedKetoDiet = v; }
    public boolean isGenitalInfectionHistory() { return genitalInfectionHistory; }
    public void setGenitalInfectionHistory(boolean v) { this.genitalInfectionHistory = v; }
    public boolean isFallsHistory() { return fallsHistory; }
    public void setFallsHistory(boolean v) { this.fallsHistory = v; }
    public boolean isOrthostaticHypotension() { return orthostaticHypotension; }
    public void setOrthostaticHypotension(boolean v) { this.orthostaticHypotension = v; }
    public String getSaltSensitivityPhenotype() { return saltSensitivityPhenotype; }
    public void setSaltSensitivityPhenotype(String v) { this.saltSensitivityPhenotype = v; }
    public boolean isActiveFastingPeriod() { return activeFastingPeriod; }
    public void setActiveFastingPeriod(boolean v) { this.activeFastingPeriod = v; }
    public int getActiveFastingDurationHours() { return activeFastingDurationHours; }
    public void setActiveFastingDurationHours(int v) { this.activeFastingDurationHours = v; }
    public Double getEGFRBaseline() { return eGFRBaseline; }
    public void setEGFRBaseline(Double v) { this.eGFRBaseline = v; }
    public Long getEGFRBaselineTimestamp() { return eGFRBaselineTimestamp; }
    public void setEGFRBaselineTimestamp(Long v) { this.eGFRBaselineTimestamp = v; }
    public Double getEGFRCurrent() { return eGFRCurrent; }
    public void setEGFRCurrent(Double v) { this.eGFRCurrent = v; }
    public Long getEGFRCurrentTimestamp() { return eGFRCurrentTimestamp; }
    public void setEGFRCurrentTimestamp(Long v) { this.eGFRCurrentTimestamp = v; }
    public Map<String, Long> getSuppressionHistory() { return suppressionHistory; }
    public void setSuppressionHistory(Map<String, Long> m) { this.suppressionHistory = m; }
    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long v) { this.lastUpdated = v; }
    public long getTotalEventsProcessed() { return totalEventsProcessed; }
    public void setTotalEventsProcessed(long v) { this.totalEventsProcessed = v; }
}
```

- [ ] **Step 5: Verify compilation**

```bash
cd backend/shared-infrastructure/flink-processing && mvn test-compile -pl . -q 2>&1 | tail -5
```

- [ ] **Step 6: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/CID*.java \
  src/main/java/com/cardiofit/flink/models/ComorbidityState.java
git commit -m "feat(module8): add CIDSeverity, CIDRuleId, CIDAlert, ComorbidityState models"
```

---

## Task 2: Module8TestBuilder — Test Data Factory

**File:** `src/test/java/com/cardiofit/flink/builders/Module8TestBuilder.java`

Provides factory methods for canonical comorbidity scenarios. All subsequent TDD tasks depend on this.

- [ ] **Step 1: Create Module8TestBuilder**

```java
package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import java.util.*;

/**
 * Test data factory for Module 8 CID Engine tests.
 * Provides patient comorbidity state scenarios.
 *
 * Drug class conventions (must match Module8 evaluator constants):
 *   ACEI, ARB, SGLT2I, THIAZIDE, LOOP_DIURETIC, BETA_BLOCKER,
 *   INSULIN, SU (sulfonylurea), GLP1RA, STATIN, FINERENONE,
 *   CCB, NSAID, CORTICOSTEROID, FLUDROCORTISONE
 */
public class Module8TestBuilder {

    // ── ComorbidityState builders ──

    /** Helper: timestamp N days before now. */
    private static long daysAgo(int days) {
        return System.currentTimeMillis() - (long) days * 86_400_000L;
    }

    /**
     * CID-01 Triple Whammy: ACEi + SGLT2i + thiazide + eGFR 14-day decline >20%.
     */
    public static ComorbidityState tripleWhammyPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.addMedication("chlorthalidone", "THIAZIDE", 12.5);
        state.addToRollingBuffer("weight", 75.0, daysAgo(7));
        state.addToRollingBuffer("weight", 72.0, System.currentTimeMillis());
        state.setEGFR14dAgo(60.0);   // 14 days ago
        state.setEGFRCurrent(48.0);  // now → 20% decline
        return state;
    }

    /**
     * CID-01 variant: Same drugs but NO precipitant (no weight drop, no eGFR decline).
     * Should NOT fire CID-01.
     */
    public static ComorbidityState tripleWhammyNoPrecipitant(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.addMedication("chlorthalidone", "THIAZIDE", 12.5);
        state.addToRollingBuffer("weight", 74.5, daysAgo(7));
        state.addToRollingBuffer("weight", 74.0, System.currentTimeMillis());
        state.setEGFR14dAgo(60.0);
        state.setEGFRCurrent(58.0); // <5% decline — normal
        return state;
    }

    /**
     * CID-02 Hyperkalemia: ACEi max dose + finerenone + rising K+.
     */
    public static ComorbidityState hyperkalemiaPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("ramipril", "ACEI", 10.0); // max dose
        state.addMedication("finerenone", "FINERENONE", 20.0);
        state.updateLab("potassium", 5.5);
        return state;
    }

    /**
     * CID-03 Hypoglycemia Masking: insulin + beta-blocker + glucose <60.
     */
    public static ComorbidityState hypoMaskingPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("insulin glargine", "INSULIN", 30.0);
        state.addMedication("metoprolol", "BETA_BLOCKER", 100.0);
        state.setLatestGlucose(55.0); // <60 mg/dL
        state.setSymptomReportedHypoglycemia(false); // no symptoms reported
        return state;
    }

    /**
     * CID-03 variant: Same drugs + glucose <60 BUT patient reports symptoms.
     * Should NOT fire CID-03 (masking = asymptomatic only).
     */
    public static ComorbidityState hypoWithSymptoms(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("insulin glargine", "INSULIN", 30.0);
        state.addMedication("metoprolol", "BETA_BLOCKER", 100.0);
        state.setLatestGlucose(55.0);
        state.setSymptomReportedHypoglycemia(true); // symptoms reported — not masked
        return state;
    }

    /**
     * CID-04 Euglycemic DKA: SGLT2i + nausea/vomiting.
     */
    public static ComorbidityState euglycemicDKAPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("dapagliflozin", "SGLT2I", 10.0);
        state.setSymptomReportedNauseaVomiting(true);
        state.setLatestGlucose(140.0); // normal glucose — that's the danger
        return state;
    }

    /**
     * CID-05 Severe Hypotension: 3+ antihypertensives + SGLT2i + SBP <95.
     */
    public static ComorbidityState severeHypotensionPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("amlodipine", "CCB", 10.0);
        state.addMedication("chlorthalidone", "THIAZIDE", 12.5);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.setLatestSBP(92.0); // <95 mmHg
        return state;
    }

    /**
     * CID-06 Thiazide Glucose Worsening: thiazide + FBG increase >15 mg/dL in 14d.
     */
    public static ComorbidityState thiazideGlucosePatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("hydrochlorothiazide", "THIAZIDE", 25.0);
        // FBG 14 days ago: 110, recent: 128 → delta = +18 mg/dL
        state.addToRollingBuffer("fbg", 110.0, daysAgo(14));
        state.addToRollingBuffer("fbg", 128.0, System.currentTimeMillis());
        return state;
    }

    /**
     * CID-07 ACEi + eGFR Decline: ACEi/ARB + eGFR drop >25% + >6 weeks.
     */
    public static ComorbidityState eGFRDeclinePatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("telmisartan", "ARB", 80.0);
        state.setEGFRBaseline(65.0);
        state.setEGFRCurrent(45.0); // 30.8% decline
        state.setEGFRBaselineTimestamp(System.currentTimeMillis() - 56 * 86400000L); // 8 weeks ago
        state.setEGFRCurrentTimestamp(System.currentTimeMillis());
        return state;
    }

    /**
     * CID-08 Statin Myopathy: statin + muscle pain reported.
     */
    public static ComorbidityState statinMyopathyPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("atorvastatin", "STATIN", 40.0);
        state.setSymptomReportedMusclePain(true);
        return state;
    }

    /**
     * CID-09 GLP-1RA + GI + Dehydration: GLP-1RA + nausea + weight drop.
     */
    public static ComorbidityState glp1raGIPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("semaglutide", "GLP1RA", 0.5);
        state.setSymptomReportedNauseaVomiting(true);
        state.setSymptomNauseaOnsetTimestamp(daysAgo(3)); // 3 days ago → meets 48h threshold
        state.addToRollingBuffer("weight", 80.0, daysAgo(7));
        state.addToRollingBuffer("weight", 78.0, System.currentTimeMillis()); // -2kg in 7d
        return state;
    }

    /**
     * CID-10 Concurrent Deterioration: both glucose and BP worsening.
     */
    public static ComorbidityState concurrentDeteriorationPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("metformin", "METFORMIN", 1000.0);
        state.addMedication("amlodipine", "CCB", 5.0);
        // FBG: 135 → 165 (+30 mg/dL in 14d)
        state.addToRollingBuffer("fbg", 135.0, daysAgo(14));
        state.addToRollingBuffer("fbg", 165.0, System.currentTimeMillis());
        // SBP: 138 → 152 (+14 mmHg in 14d)
        state.addToRollingBuffer("sbp", 138.0, daysAgo(14));
        state.addToRollingBuffer("sbp", 152.0, System.currentTimeMillis());
        // No medication change in past 14 days (CID-10 guard)
        state.setLastMedicationChangeTimestamp(daysAgo(30));
        return state;
    }

    /**
     * CID-13 Elderly + Intensive BP: age 78, SBP target <130, eGFR <45.
     */
    public static ComorbidityState elderlyIntensiveBPPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.setAge(78);
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("amlodipine", "CCB", 10.0);
        state.setEGFRCurrent(42.0);
        state.setFallsHistory(true);
        // SBP target stored externally — evaluator receives it as parameter
        return state;
    }

    /**
     * CID-14 Metformin near eGFR threshold: eGFR 32, declining trajectory.
     */
    public static ComorbidityState metforminEGFRPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("metformin", "METFORMIN", 2000.0);
        state.setEGFRCurrent(32.0);
        state.setEGFRBaseline(45.0);
        state.setEGFRBaselineTimestamp(System.currentTimeMillis() - 180 * 86400000L);
        state.setEGFRCurrentTimestamp(System.currentTimeMillis());
        return state;
    }

    /**
     * CID-15 SGLT2i + NSAID.
     */
    public static ComorbidityState sglt2iNsaidPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.addMedication("ibuprofen", "NSAID", 400.0);
        return state;
    }

    /**
     * CID-12 Polypharmacy: 9 active medications.
     */
    public static ComorbidityState polypharmacyPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("metformin", "METFORMIN", 1000.0);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.addMedication("semaglutide", "GLP1RA", 1.0);
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("amlodipine", "CCB", 10.0);
        state.addMedication("atorvastatin", "STATIN", 40.0);
        state.addMedication("aspirin", "ANTIPLATELET", 75.0);
        state.addMedication("omeprazole", "PPI", 20.0);
        state.addMedication("levothyroxine", "THYROID", 100.0);
        return state;
    }

    /**
     * Safe patient: well-controlled, no interaction triggers.
     * On metformin + atorvastatin only. Normal labs.
     */
    public static ComorbidityState safePatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("metformin", "METFORMIN", 1000.0);
        state.addMedication("atorvastatin", "STATIN", 20.0);
        state.setAge(55);
        state.setLatestSBP(128.0);
        state.setLatestGlucose(108.0);
        state.setEGFRCurrent(82.0);
        state.updateLab("potassium", 4.2);
        // Stable FBG and SBP over 14 days — no worsening
        state.addToRollingBuffer("fbg", 110.0, daysAgo(14));
        state.addToRollingBuffer("fbg", 112.0, System.currentTimeMillis());
        state.addToRollingBuffer("sbp", 129.0, daysAgo(14));
        state.addToRollingBuffer("sbp", 130.0, System.currentTimeMillis());
        return state;
    }
}
```

- [ ] **Step 2: Verify test compilation**

```bash
cd backend/shared-infrastructure/flink-processing && mvn test-compile -pl . -q 2>&1 | tail -5
```

- [ ] **Step 3: Commit**

```bash
git add src/test/java/com/cardiofit/flink/builders/Module8TestBuilder.java
git commit -m "test(module8): add Module8TestBuilder with 16 canonical CID patient scenarios"
```

---

## Task 3: Module8HALTEvaluator — HALT Rules (CID-01 to CID-05) + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module8HALTEvaluator.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module8HALTRulesTest.java`

HALT rules are life-threatening. They must be evaluated FIRST on every event, and a single match causes immediate side-output to `ingestion.safety-critical`.

- [ ] **Step 1: Write HALT rule tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module8TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class Module8HALTRulesTest {

    private static final long NOW = System.currentTimeMillis();

    // ── CID-01: Triple Whammy AKI ──

    @Test
    void cid01_tripleWhammy_fires() {
        ComorbidityState state = Module8TestBuilder.tripleWhammyPatient("P-TW");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_01".equals(a.getRuleId())),
            "Triple Whammy with weight drop should fire CID-01");
    }

    @Test
    void cid01_tripleWhammy_noPrecipitant_doesNotFire() {
        ComorbidityState state = Module8TestBuilder.tripleWhammyNoPrecipitant("P-TW-NP");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_01".equals(a.getRuleId())),
            "Triple Whammy without precipitant should NOT fire CID-01");
    }

    @Test
    void cid01_missingDiuretic_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-TW-ND");
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        // No diuretic
        state.addToRollingBuffer("weight", 75.0, NOW - 7L * 86_400_000L);
        state.addToRollingBuffer("weight", 72.0, NOW);
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_01".equals(a.getRuleId())),
            "Only 2 of 3 nephrotoxic classes — CID-01 should not fire");
    }

    // ── CID-02: Hyperkalemia Cascade ──

    @Test
    void cid02_hyperkalemia_fires() {
        ComorbidityState state = Module8TestBuilder.hyperkalemiaPatient("P-HK");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_02".equals(a.getRuleId())),
            "ACEi + finerenone + K+ 5.5 should fire CID-02");
    }

    @Test
    void cid02_normalPotassium_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-HK-NK");
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("finerenone", "FINERENONE", 20.0);
        state.updateLab("potassium", 4.5); // normal K+
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_02".equals(a.getRuleId())),
            "Normal K+ should NOT fire CID-02");
    }

    // ── CID-03: Hypoglycemia Masking ──

    @Test
    void cid03_hypoMasking_fires() {
        ComorbidityState state = Module8TestBuilder.hypoMaskingPatient("P-HM");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_03".equals(a.getRuleId())),
            "Insulin + BB + glucose <60 + no symptoms should fire CID-03");
    }

    @Test
    void cid03_hypoWithSymptoms_doesNotFire() {
        ComorbidityState state = Module8TestBuilder.hypoWithSymptoms("P-HM-S");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_03".equals(a.getRuleId())),
            "Patient reporting symptoms = not masked = CID-03 should NOT fire");
    }

    // ── CID-04: Euglycemic DKA ──

    @Test
    void cid04_euglycemicDKA_fires() {
        ComorbidityState state = Module8TestBuilder.euglycemicDKAPatient("P-DKA");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_04".equals(a.getRuleId())),
            "SGLT2i + nausea/vomiting should fire CID-04");
    }

    @Test
    void cid04_sglt2iWithoutSymptoms_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-DKA-NS");
        state.addMedication("dapagliflozin", "SGLT2I", 10.0);
        state.setSymptomReportedNauseaVomiting(false);
        state.setSymptomReportedKetoDiet(false);
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_04".equals(a.getRuleId())),
            "SGLT2i without DKA triggers should NOT fire CID-04");
    }

    // ── CID-05: Severe Hypotension ──

    @Test
    void cid05_severeHypotension_fires() {
        ComorbidityState state = Module8TestBuilder.severeHypotensionPatient("P-HYPO");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_05".equals(a.getRuleId())),
            "3+ AH + SGLT2i + SBP <95 should fire CID-05");
    }

    @Test
    void cid05_normalBP_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-HYPO-N");
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("amlodipine", "CCB", 10.0);
        state.addMedication("chlorthalidone", "THIAZIDE", 12.5);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.setLatestSBP(125.0); // normal
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_05".equals(a.getRuleId())),
            "Normal SBP should NOT fire CID-05");
    }

    // ── All HALT rules are HALT severity ──

    @Test
    void allHALTAlerts_haveSeverityHALT() {
        ComorbidityState state = Module8TestBuilder.tripleWhammyPatient("P-SEV");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        for (CIDAlert alert : alerts) {
            assertEquals("HALT", alert.getSeverity(),
                "All HALT evaluator alerts must have HALT severity");
        }
    }

    // ── Safe patient fires no HALT rules ──

    @Test
    void safePatient_noHALTAlerts() {
        ComorbidityState state = Module8TestBuilder.safePatient("P-SAFE");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertTrue(alerts.isEmpty(),
            "Safe patient should trigger zero HALT alerts");
    }
}
```

- [ ] **Step 2: Run tests to verify they fail (Module8HALTEvaluator does not exist)**

```bash
cd backend/shared-infrastructure/flink-processing && mvn test -pl . \
  -Dtest=Module8HALTRulesTest -q 2>&1 | tail -10
```

- [ ] **Step 3: Implement Module8HALTEvaluator**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

/**
 * HALT-severity CID rule evaluator (CID-01 through CID-05).
 *
 * Life-threatening comorbidity interactions. Each HALT fires
 * immediately to ingestion.safety-critical with <5 min physician SLA.
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module8HALTEvaluator {
    private Module8HALTEvaluator() {}

    // Thresholds
    private static final double WEIGHT_DROP_THRESHOLD_KG = 2.0;     // CID-01
    private static final double EGFR_DROP_THRESHOLD_PCT = 15.0;     // CID-01
    private static final double POTASSIUM_THRESHOLD = 5.3;          // CID-02
    private static final double GLUCOSE_HYPO_THRESHOLD = 60.0;      // CID-03
    private static final double SBP_HYPOTENSION_THRESHOLD = 95.0;   // CID-05
    private static final double SBP_DROP_THRESHOLD = 30.0;          // CID-05
    private static final int MIN_ANTIHYPERTENSIVES = 3;             // CID-05

    /**
     * Evaluate all 5 HALT rules against current patient state.
     * @param state current ComorbidityState snapshot
     * @param eventTime event timestamp (for rolling buffer computations)
     * @return list of CIDAlerts (may be empty, may contain multiple)
     */
    public static List<CIDAlert> evaluate(ComorbidityState state, long eventTime) {
        List<CIDAlert> alerts = new ArrayList<>();
        if (state == null) return alerts;

        evaluateCID01(state, eventTime, alerts);
        evaluateCID02(state, alerts);
        evaluateCID03(state, alerts);
        evaluateCID04(state, alerts);
        evaluateCID05(state, eventTime, alerts);

        return alerts;
    }

    /**
     * CID-01: Triple Whammy AKI.
     * ACEi/ARB + SGLT2i + diuretic + precipitant.
     * DD#7: precipitant = weight drop >2kg/7d OR eGFR drop >15%/14d OR illness.
     */
    private static void evaluateCID01(ComorbidityState state, long eventTime, List<CIDAlert> alerts) {
        boolean hasRASi = state.hasAnyDrugClass("ACEI", "ARB");
        boolean hasSGLT2i = state.hasDrugClass("SGLT2I");
        boolean hasDiuretic = state.hasAnyDrugClass("THIAZIDE", "LOOP_DIURETIC");

        if (!hasRASi || !hasSGLT2i || !hasDiuretic) return;

        // Check precipitant — per DD#7 spec
        boolean weightDrop = false;
        Double weightDelta = state.getWeightDelta7d(eventTime);
        if (weightDelta != null && weightDelta < -WEIGHT_DROP_THRESHOLD_KG) {
            weightDrop = true;
        }

        // Use 14-day acute eGFR decline (not baseline) per DD#7
        boolean eGFRDrop = false;
        Double eGFRAcuteDecline = state.getEGFRAcuteDeclinePercent14d();
        if (eGFRAcuteDecline != null && eGFRAcuteDecline >= EGFR_DROP_THRESHOLD_PCT - 1e-9) {
            eGFRDrop = true;
        }

        boolean illness = state.isSymptomReportedNauseaVomiting();

        if (!weightDrop && !eGFRDrop && !illness) return;

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("ACEI"));
        meds.addAll(state.getMedicationsByClass("ARB"));
        meds.addAll(state.getMedicationsByClass("SGLT2I"));
        meds.addAll(state.getMedicationsByClass("THIAZIDE"));
        meds.addAll(state.getMedicationsByClass("LOOP_DIURETIC"));

        String precipitant = weightDrop ? "weight drop" : eGFRDrop ? "eGFR decline" : "illness/vomiting";
        String summary = String.format(
            "HALT: Triple whammy AKI risk. Patient on RASi + SGLT2i + diuretic with %s.",
            precipitant);
        String action = "Pause SGLT2i and diuretic. Urgent eGFR + creatinine within 48 hours. " +
            "If confirmed AKI: hold all three agents until renal function recovers.";

        alerts.add(CIDAlert.create(CIDRuleId.CID_01, state.getPatientId(),
            summary, meds, action, null));
    }

    /**
     * CID-02: Hyperkalemia Cascade.
     * ACEi/ARB + finerenone + K+ > 5.3 AND rising (trajectory, not just threshold).
     */
    private static void evaluateCID02(ComorbidityState state, List<CIDAlert> alerts) {
        boolean hasRASi = state.hasAnyDrugClass("ACEI", "ARB");
        boolean hasFinerenone = state.hasDrugClass("FINERENONE");

        if (!hasRASi || !hasFinerenone) return;

        Double potassium = state.getLabValue("potassium");
        if (potassium == null || potassium < POTASSIUM_THRESHOLD - 1e-9) return;

        // Must be RISING — stable elevated K+ is a different clinical scenario
        Double previousK = state.getPreviousPotassium();
        if (previousK == null || potassium <= previousK) return;

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("ACEI"));
        meds.addAll(state.getMedicationsByClass("ARB"));
        meds.addAll(state.getMedicationsByClass("FINERENONE"));

        String summary = String.format(
            "HALT: Hyperkalemia cascade. K+ %.1f on RASi + finerenone.", potassium);
        String action = "Hold finerenone immediately. Recheck K+ in 48-72 hours. " +
            "If K+ >5.5: hold ACEi/ARB dose. If K+ >6.0: emergency protocol.";

        alerts.add(CIDAlert.create(CIDRuleId.CID_02, state.getPatientId(),
            summary, meds, action, null));
    }

    /**
     * CID-03: Hypoglycemia Masking.
     * Insulin/SU + beta-blocker + glucose <60 + no symptom report.
     */
    private static void evaluateCID03(ComorbidityState state, List<CIDAlert> alerts) {
        boolean hasHypoAgent = state.hasAnyDrugClass("INSULIN", "SU");
        boolean hasBetaBlocker = state.hasDrugClass("BETA_BLOCKER");

        if (!hasHypoAgent || !hasBetaBlocker) return;

        Double glucose = state.getLatestGlucose();
        if (glucose == null || glucose >= GLUCOSE_HYPO_THRESHOLD) return;

        // Masking = no symptoms despite low glucose
        if (state.isSymptomReportedHypoglycemia()) return;

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("INSULIN"));
        meds.addAll(state.getMedicationsByClass("SU"));
        meds.addAll(state.getMedicationsByClass("BETA_BLOCKER"));

        String summary = String.format(
            "HALT: Masked hypoglycemia. Glucose %.0f mg/dL with no symptoms. " +
            "Beta-blocker masking adrenergic warning signs.", glucose);
        String action = "Deprescribe sulfonylurea if present. If insulin: reduce basal by 20%. " +
            "Consider switching beta-blocker to carvedilol. Educate on neuroglycopenic symptoms.";

        alerts.add(CIDAlert.create(CIDRuleId.CID_03, state.getPatientId(),
            summary, meds, action, null));
    }

    /**
     * CID-04: Euglycemic DKA.
     * SGLT2i + (nausea/vomiting OR keto diet OR insulin reduction in LADA).
     */
    private static void evaluateCID04(ComorbidityState state, List<CIDAlert> alerts) {
        boolean hasSGLT2i = state.hasDrugClass("SGLT2I");
        if (!hasSGLT2i) return;

        boolean hasTrigger = state.isSymptomReportedNauseaVomiting()
            || state.isSymptomReportedKetoDiet();
        // Note: insulin dose reduction detection requires comparing current vs previous dose.
        // Deferred to Phase 2 — requires medication change tracking in state.

        if (!hasTrigger) return;

        List<String> meds = state.getMedicationsByClass("SGLT2I");
        String trigger = state.isSymptomReportedNauseaVomiting() ? "nausea/vomiting" : "keto/low-carb diet";

        String summary = String.format(
            "HALT: Euglycemic DKA risk. Patient on SGLT2i with %s. " +
            "Glucose may be NORMAL despite ketoacidosis.", trigger);
        String action = "Hold SGLT2i immediately. Check blood ketones urgently. " +
            "If ketones elevated: emergency department. If no symptoms: hold 48h, resume after trigger resolves.";

        alerts.add(CIDAlert.create(CIDRuleId.CID_04, state.getPatientId(),
            summary, meds, action, null));
    }

    /**
     * CID-05: Severe Hypotension Risk.
     * >= 3 antihypertensives + SGLT2i + SBP < 95 (or SBP drop > 30 from 7d avg).
     */
    private static void evaluateCID05(ComorbidityState state, long eventTime, List<CIDAlert> alerts) {
        boolean hasSGLT2i = state.hasDrugClass("SGLT2I");
        int ahCount = state.countAntihypertensives();

        if (!hasSGLT2i || ahCount < MIN_ANTIHYPERTENSIVES) return;

        Double sbp = state.getLatestSBP();
        if (sbp == null) return;

        boolean sbpLow = sbp < SBP_HYPOTENSION_THRESHOLD;

        boolean sbpDrop = false;
        Double sbpAvg = state.getSbpSevenDayAvg(eventTime);
        if (sbpAvg != null && (sbpAvg - sbp) > SBP_DROP_THRESHOLD) {
            sbpDrop = true;
        }

        if (!sbpLow && !sbpDrop) return;

        List<String> meds = new ArrayList<>(state.getActiveMedications().keySet());

        String summary = String.format(
            "HALT: Severe hypotension risk. SBP %.0f on %d antihypertensives + SGLT2i.", sbp, ahCount);
        String action = "Review all antihypertensive doses. Reduce or hold most recently added. " +
            "Check orthostatic BP. Assess hydration. If SBP <85: urgent review.";

        alerts.add(CIDAlert.create(CIDRuleId.CID_05, state.getPatientId(),
            summary, meds, action, null));
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend/shared-infrastructure/flink-processing && mvn test -pl . \
  -Dtest=Module8HALTRulesTest -q 2>&1 | tail -15
```

Expected: All 12 tests PASS

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module8HALTEvaluator.java \
  src/test/java/com/cardiofit/flink/operators/Module8HALTRulesTest.java
git commit -m "feat(module8): implement HALT rules CID-01 through CID-05 with TDD tests"
```

---

## Task 4: Module8PAUSEEvaluator — PAUSE Rules (CID-06 to CID-10) + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module8PAUSEEvaluator.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module8PAUSERulesTest.java`

PAUSE rules pause the correction loop. Physician review within 48 hours. Not immediately life-threatening but requires intervention.

- [ ] **Step 1: Write PAUSE rule tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module8TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module8PAUSERulesTest {

    private static final long NOW = System.currentTimeMillis();

    @Test
    void cid06_thiazideGlucose_fires() {
        ComorbidityState state = Module8TestBuilder.thiazideGlucosePatient("P-TG");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_06".equals(a.getRuleId())),
            "Thiazide + FBG increase >15 mg/dL should fire CID-06");
    }

    @Test
    void cid06_thiazideSmallGlucoseChange_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-TG-S");
        state.addMedication("hydrochlorothiazide", "THIAZIDE", 25.0);
        // FBG: 112 → 118 = delta +6 — below 15 threshold
        state.addToRollingBuffer("fbg", 112.0, NOW - 14L * 86_400_000L);
        state.addToRollingBuffer("fbg", 118.0, NOW);
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_06".equals(a.getRuleId())));
    }

    @Test
    void cid07_eGFRDecline_fires() {
        ComorbidityState state = Module8TestBuilder.eGFRDeclinePatient("P-EGD");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_07".equals(a.getRuleId())),
            "ARB + eGFR drop >25% + >6 weeks should fire CID-07");
    }

    @Test
    void cid07_expectedDip_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-EGD-ED");
        state.addMedication("telmisartan", "ARB", 80.0);
        state.setEGFRBaseline(65.0);
        state.setEGFRCurrent(55.0); // 15.4% decline — within expected 10-25% dip
        state.setEGFRBaselineTimestamp(System.currentTimeMillis() - 21 * 86400000L); // 3 weeks — within dip window
        state.setEGFRCurrentTimestamp(System.currentTimeMillis());
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_07".equals(a.getRuleId())),
            "Expected eGFR dip within 6 weeks should NOT fire CID-07");
    }

    @Test
    void cid08_statinMyopathy_fires() {
        ComorbidityState state = Module8TestBuilder.statinMyopathyPatient("P-SM");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_08".equals(a.getRuleId())),
            "Statin + muscle pain should fire CID-08");
    }

    @Test
    void cid09_glp1raGI_fires() {
        ComorbidityState state = Module8TestBuilder.glp1raGIPatient("P-GI");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_09".equals(a.getRuleId())),
            "GLP-1RA + nausea + weight drop >1.5kg should fire CID-09");
    }

    @Test
    void cid10_concurrentDeterioration_fires() {
        ComorbidityState state = Module8TestBuilder.concurrentDeteriorationPatient("P-CD");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_10".equals(a.getRuleId())),
            "Both glucose AND BP worsening should fire CID-10");
    }

    @Test
    void cid10_onlyGlucoseWorsening_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-CD-G");
        // FBG worsening: 135 → 165
        state.addToRollingBuffer("fbg", 135.0, NOW - 14L * 86_400_000L);
        state.addToRollingBuffer("fbg", 165.0, NOW);
        // SBP stable/improving: 134 → 132 (no worsening)
        state.addToRollingBuffer("sbp", 134.0, NOW - 14L * 86_400_000L);
        state.addToRollingBuffer("sbp", 132.0, NOW);
        state.setLastMedicationChangeTimestamp(NOW - 30L * 86_400_000L);
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_10".equals(a.getRuleId())),
            "Only glucose worsening (no BP worsening) should NOT fire CID-10");
    }

    @Test
    void allPAUSEAlerts_haveSeverityPAUSE() {
        ComorbidityState state = Module8TestBuilder.thiazideGlucosePatient("P-SEV");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        for (CIDAlert alert : alerts) {
            assertEquals("PAUSE", alert.getSeverity());
        }
    }

    @Test
    void safePatient_noPAUSEAlerts() {
        ComorbidityState state = Module8TestBuilder.safePatient("P-SAFE");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertTrue(alerts.isEmpty());
    }
}
```

- [ ] **Step 2: Implement Module8PAUSEEvaluator**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.ArrayList;
import java.util.List;

/**
 * PAUSE-severity CID rule evaluator (CID-06 through CID-10).
 *
 * Correction loop paused. Physician review within 48 hours.
 * Not immediately life-threatening but requires intervention.
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module8PAUSEEvaluator {
    private Module8PAUSEEvaluator() {}

    private static final double FBG_DELTA_THRESHOLD = 15.0;        // CID-06 mg/dL in 14d
    private static final double EGFR_DECLINE_THRESHOLD = 25.0;     // CID-07 percentage
    private static final double EGFR_DIP_WINDOW_WEEKS = 6.0;       // CID-07
    private static final double WEIGHT_DROP_GI_THRESHOLD = 1.5;    // CID-09 kg in 7d
    private static final double GLUCOSE_WORSENING_THRESHOLD = 10.0; // CID-10 mg/dL
    private static final double SBP_WORSENING_THRESHOLD = 10.0;    // CID-10 mmHg

    public static List<CIDAlert> evaluate(ComorbidityState state, long eventTime) {
        List<CIDAlert> alerts = new ArrayList<>();
        if (state == null) return alerts;

        evaluateCID06(state, eventTime, alerts);
        evaluateCID07(state, alerts);
        evaluateCID08(state, alerts);
        evaluateCID09(state, eventTime, alerts);
        evaluateCID10(state, eventTime, alerts);

        return alerts;
    }

    /** CID-06: Thiazide + FBG increase > 15 mg/dL in 14 days. */
    private static void evaluateCID06(ComorbidityState state, long eventTime, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("THIAZIDE")) return;
        Double fbgDelta = state.getFBGDelta14d(eventTime);
        if (fbgDelta == null || fbgDelta < FBG_DELTA_THRESHOLD - 1e-9) return;

        List<String> meds = state.getMedicationsByClass("THIAZIDE");
        String summary = String.format(
            "PAUSE: Thiazide-associated glucose rise. FBG increased %.0f mg/dL in 14 days.", fbgDelta);
        String action = "Consider: CCB substitution, dose reduction, or SGLT2i addition.";
        alerts.add(CIDAlert.create(CIDRuleId.CID_06, state.getPatientId(), summary, meds, action, null));
    }

    /** CID-07: ACEi/ARB + eGFR drop > 25% + > 6 weeks since initiation. */
    private static void evaluateCID07(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasAnyDrugClass("ACEI", "ARB")) return;
        Double decline = state.getEGFRDeclinePercent();
        Double weeks = state.getWeeksSinceEGFRBaseline();
        if (decline == null || weeks == null) return;
        if (decline < EGFR_DECLINE_THRESHOLD - 1e-9) return;
        if (weeks < EGFR_DIP_WINDOW_WEEKS) return; // within expected dip window

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("ACEI"));
        meds.addAll(state.getMedicationsByClass("ARB"));

        String summary = String.format(
            "PAUSE: Sustained eGFR decline on RASi. eGFR %.0f vs baseline %.0f (%.0f%% decline). " +
            "Expected dip window (6 weeks) has passed.", state.getEGFRCurrent(), state.getEGFRBaseline(), decline);
        String action = "Renal ultrasound with Doppler. If stenosis: stop RASi, switch to CCB. " +
            "If no stenosis: reduce RASi dose by 50%, recheck eGFR in 4 weeks.";
        alerts.add(CIDAlert.create(CIDRuleId.CID_07, state.getPatientId(), summary, meds, action, null));
    }

    /** CID-08: Statin + new muscle symptoms. */
    private static void evaluateCID08(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("STATIN")) return;
        if (!state.isSymptomReportedMusclePain()) return;

        List<String> meds = state.getMedicationsByClass("STATIN");
        String summary = "PAUSE: Possible statin myopathy. Patient reports muscle pain/weakness.";
        String action = "Order CK. If CK <5x ULN: consider statin switch. If 5-10x: hold, recheck 2 weeks. " +
            "If >10x: discontinue, urgent rhabdomyolysis review.";
        alerts.add(CIDAlert.create(CIDRuleId.CID_08, state.getPatientId(), summary, meds, action, null));
    }

    /** CID-09: GLP-1RA + persistent GI (≥48h) + weight drop > 1.5 kg/7d. */
    private static void evaluateCID09(ComorbidityState state, long eventTime, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("GLP1RA")) return;
        // R6 fix: DD#7 requires nausea/vomiting persisting ≥48 hours, not just boolean flag
        if (!state.isNauseaPersistent(eventTime, 48L * 3600_000L)) return;
        Double weightDelta = state.getWeightDelta7d(eventTime);
        if (weightDelta == null || weightDelta > -WEIGHT_DROP_GI_THRESHOLD) return;

        List<String> meds = state.getMedicationsByClass("GLP1RA");
        String summary = String.format(
            "PAUSE: GLP-1RA GI intolerance with possible dehydration. Weight change: %.1f kg in 7 days.",
            weightDelta);
        String action = "Hold GLP-1RA dose escalation. Assess hydration. " +
            "If concurrent SGLT2i/diuretic: assess renal function.";
        alerts.add(CIDAlert.create(CIDRuleId.CID_09, state.getPatientId(), summary, meds, action, null));
    }

    /** CID-10: Concurrent glucose AND BP deterioration without medication change in past 14d. */
    private static void evaluateCID10(ComorbidityState state, long eventTime, List<CIDAlert> alerts) {
        // R7 fix: DD#7 says "without medication change" — skip if meds changed within 14 days
        if (state.hadMedicationChangeWithin(eventTime, 14L * 86_400_000L)) return;

        Double fbgDelta = state.getFBGDelta14d(eventTime);
        boolean glucoseWorsening = fbgDelta != null && fbgDelta >= GLUCOSE_WORSENING_THRESHOLD - 1e-9;

        Double sbpDelta = state.getSbpDelta14d(eventTime);
        boolean bpWorsening = sbpDelta != null && sbpDelta >= SBP_WORSENING_THRESHOLD - 1e-9;

        if (!glucoseWorsening || !bpWorsening) return;

        String summary = String.format(
            "PAUSE: Concurrent deterioration. FBG +%.0f mg/dL, SBP +%.0f mmHg over 14 days " +
            "without medication change.", fbgDelta, sbpDelta);
        String action = "Review adherence. Assess lifestyle factors (diet, sleep, stress). " +
            "Consider medication intensification across both domains.";
        alerts.add(CIDAlert.create(CIDRuleId.CID_10, state.getPatientId(), summary, List.of(), action, null));
    }
}
```

- [ ] **Step 3: Run tests, verify pass**

```bash
cd backend/shared-infrastructure/flink-processing && mvn test -pl . \
  -Dtest=Module8PAUSERulesTest -q 2>&1 | tail -15
```

Expected: All 10 tests PASS

- [ ] **Step 4: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module8PAUSEEvaluator.java \
  src/test/java/com/cardiofit/flink/operators/Module8PAUSERulesTest.java
git commit -m "feat(module8): implement PAUSE rules CID-06 through CID-10"
```

---

## Task 5: Module8SOFTFLAGEvaluator — SOFT_FLAG Rules (CID-11 to CID-17) + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module8SOFTFLAGEvaluator.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module8SOFTFLAGRulesTest.java`

SOFT_FLAG rules attach warnings to Decision Cards. No correction loop pause. Informational alerts that influence card generation in KB-23.

- [ ] **Step 1: Write SOFT_FLAG tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module8TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module8SOFTFLAGRulesTest {

    @Test
    void cid12_polypharmacy_fires() {
        ComorbidityState state = Module8TestBuilder.polypharmacyPatient("P-POLY");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(alerts.stream().anyMatch(a -> "CID_12".equals(a.getRuleId())),
            "9 medications should fire CID-12 polypharmacy warning");
    }

    @Test
    void cid12_fewMeds_doesNotFire() {
        ComorbidityState state = Module8TestBuilder.safePatient("P-FEW");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertFalse(alerts.stream().anyMatch(a -> "CID_12".equals(a.getRuleId())),
            "2 medications should NOT fire CID-12");
    }

    @Test
    void cid13_elderlyIntensiveBP_fires() {
        ComorbidityState state = Module8TestBuilder.elderlyIntensiveBPPatient("P-ELD");
        // SBP target would be provided as parameter (from KB-20 patient profile)
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, 125.0);
        assertTrue(alerts.stream().anyMatch(a -> "CID_13".equals(a.getRuleId())),
            "Age 78 + SBP target 125 + eGFR <45 + falls should fire CID-13");
    }

    @Test
    void cid13_youngPatient_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-YOUNG");
        state.setAge(55);
        state.setEGFRCurrent(42.0);
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, 125.0);
        assertFalse(alerts.stream().anyMatch(a -> "CID_13".equals(a.getRuleId())),
            "Age 55 should NOT fire CID-13 (threshold is >= 75)");
    }

    @Test
    void cid14_metforminEGFR_fires() {
        ComorbidityState state = Module8TestBuilder.metforminEGFRPatient("P-MET");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(alerts.stream().anyMatch(a -> "CID_14".equals(a.getRuleId())),
            "Metformin + eGFR 32 + declining should fire CID-14");
    }

    @Test
    void cid15_sglt2iNsaid_fires() {
        ComorbidityState state = Module8TestBuilder.sglt2iNsaidPatient("P-NSAID");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(alerts.stream().anyMatch(a -> "CID_15".equals(a.getRuleId())),
            "SGLT2i + NSAID should fire CID-15");
    }

    @Test
    void cid11_genitalInfection_fires() {
        ComorbidityState state = new ComorbidityState("P-GI");
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.setGenitalInfectionHistory(true);
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(alerts.stream().anyMatch(a -> "CID_11".equals(a.getRuleId())),
            "SGLT2i + genital infection history should fire CID-11");
    }

    @Test
    void cid17_sglt2iFasting_fires() {
        ComorbidityState state = new ComorbidityState("P-FAST");
        state.addMedication("dapagliflozin", "SGLT2I", 10.0);
        state.setActiveFastingPeriod(true);
        state.setActiveFastingDurationHours(18); // >16h
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(alerts.stream().anyMatch(a -> "CID_17".equals(a.getRuleId())),
            "SGLT2i + fasting >16h should fire CID-17");
    }

    @Test
    void cid17_shortFast_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-FAST-S");
        state.addMedication("dapagliflozin", "SGLT2I", 10.0);
        state.setActiveFastingPeriod(true);
        state.setActiveFastingDurationHours(12); // <16h
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertFalse(alerts.stream().anyMatch(a -> "CID_17".equals(a.getRuleId())),
            "Fasting <16h should NOT fire CID-17");
    }

    @Test
    void allSOFTFLAGs_haveSeveritySOFTFLAG() {
        ComorbidityState state = Module8TestBuilder.polypharmacyPatient("P-SEV");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        for (CIDAlert alert : alerts) {
            assertEquals("SOFT_FLAG", alert.getSeverity());
        }
    }

    @Test
    void safePatient_noSOFTFLAGs() {
        ComorbidityState state = Module8TestBuilder.safePatient("P-SAFE");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(alerts.isEmpty());
    }
}
```

- [ ] **Step 2: Implement Module8SOFTFLAGEvaluator**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.ArrayList;
import java.util.List;

/**
 * SOFT_FLAG-severity CID rule evaluator (CID-11 through CID-17).
 *
 * Warnings attached to Decision Cards. No correction loop pause.
 * Informational alerts that influence KB-23 card generation.
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module8SOFTFLAGEvaluator {
    private Module8SOFTFLAGEvaluator() {}

    private static final int POLYPHARMACY_THRESHOLD = 8;    // CID-12
    private static final int ELDERLY_AGE_THRESHOLD = 75;     // CID-13
    private static final double SBP_TARGET_INTENSIVE = 130.0; // CID-13
    private static final double EGFR_METFORMIN_LOW = 30.0;   // CID-14
    private static final double EGFR_METFORMIN_HIGH = 35.0;  // CID-14
    private static final int FASTING_DURATION_THRESHOLD = 16; // CID-17 hours

    /**
     * Evaluate all 7 SOFT_FLAG rules.
     * @param state current patient comorbidity state
     * @param sbpTargetMmHg patient's SBP target (from KB-20), nullable
     */
    public static List<CIDAlert> evaluate(ComorbidityState state, Double sbpTargetMmHg) {
        List<CIDAlert> alerts = new ArrayList<>();
        if (state == null) return alerts;

        evaluateCID11(state, alerts);
        evaluateCID12(state, alerts);
        evaluateCID13(state, sbpTargetMmHg, alerts);
        evaluateCID14(state, alerts);
        evaluateCID15(state, alerts);
        evaluateCID16(state, alerts);
        evaluateCID17(state, alerts);

        return alerts;
    }

    /** CID-11: Genital infection history + SGLT2i. */
    private static void evaluateCID11(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("SGLT2I")) return;
        if (!state.isGenitalInfectionHistory()) return;

        alerts.add(CIDAlert.create(CIDRuleId.CID_11, state.getPatientId(),
            "WARNING: Patient has genital infection history. SGLT2i may increase recurrence.",
            state.getMedicationsByClass("SGLT2I"),
            "Monitor closely. Consider prophylactic antifungal if recurrence occurs.", null));
    }

    /** CID-12: Polypharmacy burden >= 8 medications. */
    private static void evaluateCID12(ComorbidityState state, List<CIDAlert> alerts) {
        int count = state.getActiveMedicationCount();
        if (count < POLYPHARMACY_THRESHOLD) return;

        alerts.add(CIDAlert.create(CIDRuleId.CID_12, state.getPatientId(),
            String.format("WARNING: Polypharmacy burden high (%d daily medications). " +
                "Consider deprescribing review before adding.", count),
            new ArrayList<>(state.getActiveMedications().keySet()),
            "Review medication list. Assess adherence risk. Consider deprescribing.", null));
    }

    /** CID-13: Elderly + intensive BP target. */
    private static void evaluateCID13(ComorbidityState state, Double sbpTarget, List<CIDAlert> alerts) {
        if (state.getAge() == null || state.getAge() < ELDERLY_AGE_THRESHOLD) return;
        if (sbpTarget == null || sbpTarget >= SBP_TARGET_INTENSIVE) return;

        boolean hasRisk = false;
        if (state.getEGFRCurrent() != null && state.getEGFRCurrent() < 45.0) hasRisk = true;
        if (state.isFallsHistory()) hasRisk = true;
        if (state.isOrthostaticHypotension()) hasRisk = true;

        if (!hasRisk) return;

        alerts.add(CIDAlert.create(CIDRuleId.CID_13, state.getPatientId(),
            String.format("WARNING: Intensive SBP target (<%.0f) may increase adverse events " +
                "in this elderly patient (age %d). Consider relaxing to <140 mmHg.", sbpTarget, state.getAge()),
            List.of(),
            "Review BP target per ADA 2026 frailty guidance. Consider <140 mmHg.", null));
    }

    /** CID-14: Metformin + eGFR 30-35 + declining trajectory. */
    private static void evaluateCID14(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("METFORMIN")) return;
        Double eGFR = state.getEGFRCurrent();
        if (eGFR == null) return;
        if (eGFR < EGFR_METFORMIN_LOW || eGFR > EGFR_METFORMIN_HIGH) return;

        // Check declining trajectory
        Double decline = state.getEGFRDeclinePercent();
        if (decline == null || decline <= 0) return; // not declining

        alerts.add(CIDAlert.create(CIDRuleId.CID_14, state.getPatientId(),
            String.format("WARNING: eGFR approaching metformin threshold (30). Current: %.0f. " +
                "Plan for dose reduction at 30-45, discontinuation at <30.", eGFR),
            state.getMedicationsByClass("METFORMIN"),
            "Consider SGLT2i as glucose-lowering replacement if eGFR permits.", null));
    }

    /** CID-15: SGLT2i + NSAID use. */
    private static void evaluateCID15(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("SGLT2I") || !state.hasDrugClass("NSAID")) return;

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("SGLT2I"));
        meds.addAll(state.getMedicationsByClass("NSAID"));

        alerts.add(CIDAlert.create(CIDRuleId.CID_15, state.getPatientId(),
            "WARNING: NSAIDs + SGLT2i increase AKI risk and reduce SGLT2i renal benefit.",
            meds,
            "Advise NSAID avoidance. If analgesia needed: paracetamol preferred.", null));
    }

    /** CID-16: Salt-sensitive patient + sodium-retaining drug. */
    private static void evaluateCID16(ComorbidityState state, List<CIDAlert> alerts) {
        if (!"HIGH".equalsIgnoreCase(state.getSaltSensitivityPhenotype())) return;
        if (!state.hasAnyDrugClass("CORTICOSTEROID", "FLUDROCORTISONE")) return;

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("CORTICOSTEROID"));
        meds.addAll(state.getMedicationsByClass("FLUDROCORTISONE"));

        alerts.add(CIDAlert.create(CIDRuleId.CID_16, state.getPatientId(),
            "WARNING: Patient is salt-sensitive. Sodium-retaining drug may elevate BP.",
            meds,
            "Monitor BP closely after initiation.", null));
    }

    /** CID-17: SGLT2i + fasting period > 16 hours. */
    private static void evaluateCID17(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("SGLT2I")) return;
        if (!state.isActiveFastingPeriod()) return;
        if (state.getActiveFastingDurationHours() < FASTING_DURATION_THRESHOLD) return;

        alerts.add(CIDAlert.create(CIDRuleId.CID_17, state.getPatientId(),
            String.format("WARNING: Extended fasting (%dh) on SGLT2i increases DKA and dehydration risk.",
                state.getActiveFastingDurationHours()),
            state.getMedicationsByClass("SGLT2I"),
            "Advise adequate hydration during non-fasting hours. Consider holding SGLT2i during fasts >20h.", null));
    }
}
```

- [ ] **Step 3: Run tests, verify pass**

```bash
cd backend/shared-infrastructure/flink-processing && mvn test -pl . \
  -Dtest=Module8SOFTFLAGRulesTest -q 2>&1 | tail -15
```

Expected: All 11 tests PASS

- [ ] **Step 4: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module8SOFTFLAGEvaluator.java \
  src/test/java/com/cardiofit/flink/operators/Module8SOFTFLAGRulesTest.java
git commit -m "feat(module8): implement SOFT_FLAG rules CID-11 through CID-17"
```

---

## Task 6: Module8SuppressionManager — Alert Dedup + Lifecycle + Tests

**Files:**
- Create: `src/main/java/com/cardiofit/flink/operators/Module8SuppressionManager.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module8SuppressionTest.java`

Alert deduplication prevents alert fatigue. Same alert (same rule + patient + medication combination) is suppressed for 72 hours unless severity escalates. HALT alerts are NEVER suppressed.

- [ ] **Step 1: Write suppression tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module8SuppressionTest {

    @Test
    void firstAlert_neverSuppressed() {
        ComorbidityState state = new ComorbidityState("P-SUP");
        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_06, "P-SUP",
            "test", List.of("drug-a"), "action", null);
        boolean suppressed = Module8SuppressionManager.shouldSuppress(alert, state, System.currentTimeMillis());
        assertFalse(suppressed, "First alert for a rule should never be suppressed");
    }

    @Test
    void duplicateWithin72Hours_suppressed() {
        ComorbidityState state = new ComorbidityState("P-SUP");
        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_06, "P-SUP",
            "test", List.of("drug-a"), "action", null);

        long now = System.currentTimeMillis();
        // Record first emission
        state.recordSuppression(alert.getSuppressionKey(), now);

        // 24 hours later, same alert
        boolean suppressed = Module8SuppressionManager.shouldSuppress(
            alert, state, now + 24 * 60 * 60 * 1000L);
        assertTrue(suppressed, "Same alert within 72h should be suppressed");
    }

    @Test
    void duplicateAfter72Hours_notSuppressed() {
        ComorbidityState state = new ComorbidityState("P-SUP");
        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_06, "P-SUP",
            "test", List.of("drug-a"), "action", null);

        long now = System.currentTimeMillis();
        state.recordSuppression(alert.getSuppressionKey(), now);

        // 73 hours later
        boolean suppressed = Module8SuppressionManager.shouldSuppress(
            alert, state, now + 73 * 60 * 60 * 1000L);
        assertFalse(suppressed, "Same alert after 72h should NOT be suppressed");
    }

    @Test
    void haltAlerts_neverSuppressed() {
        ComorbidityState state = new ComorbidityState("P-HALT");
        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_01, "P-HALT",
            "test", List.of("drug-a"), "action", null);

        long now = System.currentTimeMillis();
        state.recordSuppression(alert.getSuppressionKey(), now);

        // Same HALT alert 1 hour later
        boolean suppressed = Module8SuppressionManager.shouldSuppress(
            alert, state, now + 60 * 60 * 1000L);
        assertFalse(suppressed, "HALT alerts should NEVER be suppressed");
    }

    @Test
    void differentMedications_notSuppressed() {
        ComorbidityState state = new ComorbidityState("P-DIFF");
        CIDAlert alert1 = CIDAlert.create(CIDRuleId.CID_15, "P-DIFF",
            "test", List.of("empagliflozin", "ibuprofen"), "action", null);
        CIDAlert alert2 = CIDAlert.create(CIDRuleId.CID_15, "P-DIFF",
            "test", List.of("empagliflozin", "naproxen"), "action", null);

        long now = System.currentTimeMillis();
        state.recordSuppression(alert1.getSuppressionKey(), now);

        // Same rule but different medication combo
        boolean suppressed = Module8SuppressionManager.shouldSuppress(
            alert2, state, now + 1000L);
        assertFalse(suppressed,
            "Same rule but different medication combination should NOT be suppressed");
    }
}
```

- [ ] **Step 2: Implement Module8SuppressionManager**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;

/**
 * Alert suppression and deduplication manager.
 *
 * Suppression rules:
 * - HALT alerts: NEVER suppressed (patient safety overrides fatigue prevention)
 * - PAUSE/SOFT_FLAG: Suppressed for 72 hours per suppression key
 * - Suppression key = ruleId + patientId + hash(sorted medications)
 * - Different medication combinations = different suppression keys
 * - Severity escalation (e.g., SOFT_FLAG → PAUSE for same combination)
 *   bypasses suppression
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module8SuppressionManager {
    private Module8SuppressionManager() {}

    /**
     * Determine if an alert should be suppressed.
     * @param alert the candidate alert
     * @param state current patient state (contains suppression history)
     * @param currentTime current event time
     * @return true if the alert should be suppressed (not emitted)
     */
    public static boolean shouldSuppress(CIDAlert alert, ComorbidityState state, long currentTime) {
        if (alert == null || state == null) return false;

        // HALT alerts are NEVER suppressed — patient safety override
        if ("HALT".equals(alert.getSeverity())) return false;

        return state.isSuppressed(alert.getSuppressionKey(), currentTime);
    }

    /**
     * Record that an alert was emitted (for future suppression checks).
     */
    public static void recordEmission(CIDAlert alert, ComorbidityState state, long currentTime) {
        if (alert == null || state == null) return;
        state.recordSuppression(alert.getSuppressionKey(), currentTime);
    }
}
```

- [ ] **Step 3: Run tests, verify pass**

```bash
cd backend/shared-infrastructure/flink-processing && mvn test -pl . \
  -Dtest=Module8SuppressionTest -q 2>&1 | tail -10
```

Expected: All 5 tests PASS

- [ ] **Step 4: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module8SuppressionManager.java \
  src/test/java/com/cardiofit/flink/operators/Module8SuppressionTest.java
git commit -m "feat(module8): implement alert suppression with 72h dedup and HALT bypass"
```

---

## Task 7: Module8_ComorbidityEngine — Main Operator

**File:** `src/main/java/com/cardiofit/flink/operators/Module8_ComorbidityEngine.java`

This `KeyedProcessFunction` wires all 3 rule evaluators + suppression manager. It maintains `ComorbidityState` in Flink keyed state with 31-day TTL, updates state from incoming events, evaluates all rules, applies suppression, and routes HALT to side-output.

**Important:** The state update logic must extract medication, lab, vital, and patient-reported data from the `CanonicalEvent` payload. The exact payload structure depends on the on-disk `CanonicalEvent.java` — verify before implementing.

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
import java.util.Map;

/**
 * Module 8: Comorbidity Interaction Detector — main operator.
 *
 * Keyed by patientId. On each CanonicalEvent:
 * 1. Update ComorbidityState from event payload (meds, labs, vitals, symptoms)
 * 2. Evaluate HALT rules (CID-01 to CID-05) → side output if match
 * 3. Evaluate PAUSE rules (CID-06 to CID-10) → main output if match
 * 4. Evaluate SOFT_FLAG rules (CID-11 to CID-17) → main output if match
 * 5. Apply suppression to PAUSE/SOFT_FLAG alerts
 * 6. Emit unsuppressed alerts
 *
 * HALT alerts bypass suppression entirely (patient safety).
 *
 * State TTL: 31 days.
 */
public class Module8_ComorbidityEngine
        extends KeyedProcessFunction<String, CanonicalEvent, CIDAlert> {

    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(Module8_ComorbidityEngine.class);

    // Side-output for HALT alerts → ingestion.safety-critical
    public static final OutputTag<CIDAlert> HALT_SAFETY_TAG =
        new OutputTag<>("safety-critical-cid", TypeInformation.of(CIDAlert.class));

    private transient ValueState<ComorbidityState> comorbidityState;

    @Override
    public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
        super.open(openContext);

        ValueStateDescriptor<ComorbidityState> stateDesc =
            new ValueStateDescriptor<>("comorbidity-state", ComorbidityState.class);
        StateTtlConfig ttl = StateTtlConfig
            .newBuilder(Duration.ofDays(31))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();
        stateDesc.enableTimeToLive(ttl);
        comorbidityState = getRuntimeContext().getState(stateDesc);

        LOG.info("Module8_ComorbidityEngine initialized");
    }

    @Override
    public void processElement(CanonicalEvent event, Context ctx,
                                Collector<CIDAlert> out) throws Exception {
        // 1. Get or create state
        ComorbidityState state = comorbidityState.value();
        if (state == null) {
            state = new ComorbidityState(event.getPatientId());
        }

        // 2. Update state from event
        updateStateFromEvent(state, event);

        long now = event.getEventTimestamp();

        // 3. Evaluate HALT rules (always first, never suppressed)
        List<CIDAlert> haltAlerts = Module8HALTEvaluator.evaluate(state, now);
        for (CIDAlert alert : haltAlerts) {
            alert.setCorrelationId(event.getCorrelationId());
            LOG.warn("Module8: HALT alert {} for patient {}. {}",
                alert.getRuleId(), event.getPatientId(), alert.getTriggerSummary());
            ctx.output(HALT_SAFETY_TAG, alert);
            // Also emit to main output for KB-23 card generation
            out.collect(alert);
            Module8SuppressionManager.recordEmission(alert, state, now);
        }

        // 4. Evaluate PAUSE rules
        List<CIDAlert> pauseAlerts = Module8PAUSEEvaluator.evaluate(state, now);
        for (CIDAlert alert : pauseAlerts) {
            alert.setCorrelationId(event.getCorrelationId());
            if (!Module8SuppressionManager.shouldSuppress(alert, state, now)) {
                LOG.info("Module8: PAUSE alert {} for patient {}",
                    alert.getRuleId(), event.getPatientId());
                out.collect(alert);
                Module8SuppressionManager.recordEmission(alert, state, now);
            }
        }

        // 5. Evaluate SOFT_FLAG rules
        // SBP target is not available from the event — would need KB-20 lookup.
        // For now, pass null (CID-13 will only fire if target is provided via state extension).
        Double sbpTarget = null; // TODO: integrate KB-20 patient target via broadcast state
        List<CIDAlert> softAlerts = Module8SOFTFLAGEvaluator.evaluate(state, sbpTarget);
        for (CIDAlert alert : softAlerts) {
            alert.setCorrelationId(event.getCorrelationId());
            if (!Module8SuppressionManager.shouldSuppress(alert, state, now)) {
                LOG.debug("Module8: SOFT_FLAG alert {} for patient {}",
                    alert.getRuleId(), event.getPatientId());
                out.collect(alert);
                Module8SuppressionManager.recordEmission(alert, state, now);
            }
        }

        // 6. Update state
        state.setLastUpdated(now);
        state.setTotalEventsProcessed(state.getTotalEventsProcessed() + 1);
        comorbidityState.update(state);
    }

    /**
     * Extract clinical data from CanonicalEvent payload and update ComorbidityState.
     *
     * IMPORTANT: The payload field extraction below is illustrative.
     * The actual field names depend on the on-disk CanonicalEvent and
     * the upstream enrichment pipeline (Module 1/1b → Module 2).
     * VERIFY AND ADJUST before deployment.
     */
    private void updateStateFromEvent(ComorbidityState state, CanonicalEvent event) {
        if (event == null || event.getPayload() == null) return;

        Map<String, Object> payload = event.getPayload();
        EventType eventType = event.getEventType();
        long eventTime = event.getEventTimestamp();

        try {
            // Medication events — handle all three medication lifecycle types
            if (eventType == EventType.MEDICATION_ORDERED
                    || eventType == EventType.MEDICATION_ADMINISTERED
                    || eventType == EventType.MEDICATION_PRESCRIBED) {
                String drugName = getStringField(payload, "drug_name");
                String drugClass = getStringField(payload, "drug_class");
                Double dose = getDoubleField(payload, "dose_mg");

                if (drugName != null && drugClass != null) {
                    state.addMedication(drugName, drugClass, dose);
                    state.setLastMedicationChangeTimestamp(eventTime); // for CID-10 guard
                }
            }
            if (eventType == EventType.MEDICATION_DISCONTINUED) {
                String drugName = getStringField(payload, "drug_name");
                if (drugName != null) {
                    state.removeMedication(drugName);
                    state.setLastMedicationChangeTimestamp(eventTime);
                }
            }

            // Lab results
            if (eventType == EventType.LAB_RESULT) {
                String labType = getStringField(payload, "lab_type");
                Double value = getDoubleField(payload, "value");
                if (labType != null && value != null) {
                    state.updateLab(labType, value);

                    // eGFR: track baseline + current + 14-day history for CID-01
                    if ("egfr".equalsIgnoreCase(labType)) {
                        if (state.getEGFRBaseline() == null) {
                            state.setEGFRBaseline(value);
                            state.setEGFRBaselineTimestamp(eventTime);
                        }
                        // Shift current → 14d-ago if current is >14 days old
                        Long prevTs = state.getEGFRCurrentTimestamp();
                        if (prevTs != null && (eventTime - prevTs) >= 14L * 86400000L) {
                            state.setEGFR14dAgo(state.getEGFRCurrent());
                        }
                        state.setEGFRCurrent(value);
                        state.setEGFRCurrentTimestamp(eventTime);
                    }

                    // Potassium: track previous for CID-02 rising trajectory
                    if ("potassium".equalsIgnoreCase(labType)) {
                        Double currentK = state.getLabValue("potassium");
                        if (currentK != null) {
                            state.setPreviousPotassium(currentK);
                        }
                    }

                    // Glucose/FBG
                    if ("glucose".equalsIgnoreCase(labType) || "fbg".equalsIgnoreCase(labType)) {
                        state.setLatestGlucose(value);
                        state.addToRollingBuffer("fbg", value, eventTime);
                    }
                }
            }

            // Vital signs — feed rolling buffers for averages
            if (eventType == EventType.VITAL_SIGN) {
                Double sbp = getDoubleField(payload, "systolic_bp");
                Double dbp = getDoubleField(payload, "diastolic_bp");
                Double weight = getDoubleField(payload, "weight");

                if (sbp != null) {
                    state.setLatestSBP(sbp);
                    state.addToRollingBuffer("sbp", sbp, eventTime);
                }
                if (dbp != null) state.setLatestDBP(dbp);
                if (weight != null) {
                    state.setLatestWeight(weight);
                    state.addToRollingBuffer("weight", weight, eventTime);
                }
            }

            // Patient-reported symptoms — record with onset timestamp for TTL
            if (eventType == EventType.PATIENT_REPORTED) {
                String symptom = getStringField(payload, "symptom_type");
                if ("HYPOGLYCEMIA".equalsIgnoreCase(symptom)) {
                    state.setSymptomReportedHypoglycemia(true);
                    state.setSymptomHypoglycemiaTimestamp(eventTime);
                }
                if ("MUSCLE_PAIN".equalsIgnoreCase(symptom) || "MYALGIA".equalsIgnoreCase(symptom)) {
                    state.setSymptomReportedMusclePain(true);
                    state.setSymptomMusclePainTimestamp(eventTime);
                }
                if ("NAUSEA".equalsIgnoreCase(symptom) || "VOMITING".equalsIgnoreCase(symptom)) {
                    state.setSymptomReportedNauseaVomiting(true);
                    state.setSymptomNauseaOnsetTimestamp(eventTime);
                }
                if ("KETO_DIET".equalsIgnoreCase(symptom) || "LOW_CARB".equalsIgnoreCase(symptom)) {
                    state.setSymptomReportedKetoDiet(true);
                }
                // Symptom resolution — allows clearing sticky flags
                if ("RESOLVED".equalsIgnoreCase(getStringField(payload, "status"))) {
                    clearSymptomFlag(state, symptom);
                }
            }

            // Demographics (from enrichment)
            Integer age = getIntField(payload, "age");
            if (age != null) state.setAge(age);

            // Expire stale symptom flags (TTL: 72h for CID-08, 48h-aware for CID-09)
            state.expireStaleSymptoms(eventTime);

        } catch (Exception e) {
            LOG.warn("Module8: failed to update state from event for patient {}. Error: {}",
                state.getPatientId(), e.getMessage());
        }
    }

    private static void clearSymptomFlag(ComorbidityState state, String symptom) {
        if (symptom == null) return;
        switch (symptom.toUpperCase()) {
            case "HYPOGLYCEMIA": state.setSymptomReportedHypoglycemia(false); break;
            case "MUSCLE_PAIN": case "MYALGIA": state.setSymptomReportedMusclePain(false); break;
            case "NAUSEA": case "VOMITING": state.setSymptomReportedNauseaVomiting(false); break;
            case "KETO_DIET": case "LOW_CARB": state.setSymptomReportedKetoDiet(false); break;
        }
    }

    // --- Payload field extraction helpers ---

    private static String getStringField(Map<String, Object> payload, String key) {
        Object val = payload.get(key);
        return val != null ? val.toString() : null;
    }

    private static Double getDoubleField(Map<String, Object> payload, String key) {
        Object val = payload.get(key);
        if (val instanceof Number) return ((Number) val).doubleValue();
        if (val instanceof String) {
            try { return Double.parseDouble((String) val); }
            catch (NumberFormatException e) { return null; }
        }
        return null;
    }

    private static Integer getIntField(Map<String, Object> payload, String key) {
        Object val = payload.get(key);
        if (val instanceof Number) return ((Number) val).intValue();
        if (val instanceof String) {
            try { return Integer.parseInt((String) val); }
            catch (NumberFormatException e) { return null; }
        }
        return null;
    }
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5
```

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module8_ComorbidityEngine.java
git commit -m "feat(module8): implement ComorbidityEngine KeyedProcessFunction with 17 CID rules"
```

---

## Task 8: Integration Tests + Orchestrator Wiring

**Files:**
- Modify: `FlinkJobOrchestrator.java`
- Create: `src/test/java/com/cardiofit/flink/operators/Module8IntegrationTest.java`

- [ ] **Step 1: Write integration tests**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module8TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module8IntegrationTest {

    private static final long NOW = System.currentTimeMillis();

    @Test
    void tripleWhammy_producesHALTAlert() {
        ComorbidityState state = Module8TestBuilder.tripleWhammyPatient("P-INT-TW");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.isEmpty());
        assertEquals("HALT", alerts.get(0).getSeverity());
        assertEquals("CID_01", alerts.get(0).getRuleId());
        assertNotNull(alerts.get(0).getAlertId());
        assertNotNull(alerts.get(0).getSuppressionKey());
    }

    @Test
    void safePatient_producesNoAlerts() {
        ComorbidityState state = Module8TestBuilder.safePatient("P-INT-SAFE");
        List<CIDAlert> halt = Module8HALTEvaluator.evaluate(state, NOW);
        List<CIDAlert> pause = Module8PAUSEEvaluator.evaluate(state, NOW);
        List<CIDAlert> soft = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(halt.isEmpty(), "Safe patient: no HALT");
        assertTrue(pause.isEmpty(), "Safe patient: no PAUSE");
        assertTrue(soft.isEmpty(), "Safe patient: no SOFT_FLAG");
    }

    @Test
    void multipleRulesCanFireSimultaneously() {
        // Patient on ACEi + SGLT2i + thiazide + NSAID + weight drop → CID-01 + CID-15
        ComorbidityState state = Module8TestBuilder.tripleWhammyPatient("P-MULTI");
        state.addMedication("ibuprofen", "NSAID", 400.0);

        List<CIDAlert> halt = Module8HALTEvaluator.evaluate(state, NOW);
        List<CIDAlert> soft = Module8SOFTFLAGEvaluator.evaluate(state, null);

        assertTrue(halt.stream().anyMatch(a -> "CID_01".equals(a.getRuleId())),
            "CID-01 should fire");
        assertTrue(soft.stream().anyMatch(a -> "CID_15".equals(a.getRuleId())),
            "CID-15 should also fire (SGLT2i + NSAID)");
    }

    @Test
    void suppressionPreventsRepeatedPAUSEAlerts() {
        ComorbidityState state = Module8TestBuilder.statinMyopathyPatient("P-SUP");

        List<CIDAlert> alerts1 = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertFalse(alerts1.isEmpty());
        CIDAlert first = alerts1.get(0);

        // Record emission
        Module8SuppressionManager.recordEmission(first, state, NOW);

        // Same state, 12 hours later
        List<CIDAlert> alerts2 = Module8PAUSEEvaluator.evaluate(state, NOW + 12 * 3600_000L);
        for (CIDAlert a : alerts2) {
            if ("CID_08".equals(a.getRuleId())) {
                assertTrue(Module8SuppressionManager.shouldSuppress(a, state, NOW + 12 * 3600_000L),
                    "CID-08 should be suppressed within 72h");
            }
        }
    }

    @Test
    void haltAndPauseCanCoexist() {
        // Patient triggers CID-04 (HALT: SGLT2i + nausea) and CID-09 (PAUSE: GLP-1RA + GI ≥48h)
        long nauseaOnset = NOW - 50L * 3600_000L; // 50 hours ago → meets 48h threshold
        ComorbidityState state = new ComorbidityState("P-COEXIST");
        state.addMedication("dapagliflozin", "SGLT2I", 10.0);
        state.addMedication("semaglutide", "GLP1RA", 1.0);
        state.setSymptomReportedNauseaVomiting(true);
        state.setSymptomNauseaOnsetTimestamp(nauseaOnset);
        state.addToRollingBuffer("weight", 80.0, NOW - 7L * 86_400_000L);
        state.addToRollingBuffer("weight", 78.0, NOW);

        List<CIDAlert> halt = Module8HALTEvaluator.evaluate(state, NOW);
        List<CIDAlert> pause = Module8PAUSEEvaluator.evaluate(state, NOW);

        assertTrue(halt.stream().anyMatch(a -> "CID_04".equals(a.getRuleId())),
            "CID-04 HALT should fire (SGLT2i + nausea)");
        assertTrue(pause.stream().anyMatch(a -> "CID_09".equals(a.getRuleId())),
            "CID-09 PAUSE should also fire (GLP-1RA + GI ≥48h + weight drop)");
    }

    @Test
    void alertContainsMedicationList() {
        ComorbidityState state = Module8TestBuilder.sglt2iNsaidPatient("P-MEDS");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        CIDAlert cid15 = alerts.stream()
            .filter(a -> "CID_15".equals(a.getRuleId()))
            .findFirst().orElse(null);
        assertNotNull(cid15);
        assertNotNull(cid15.getMedicationsInvolved());
        assertFalse(cid15.getMedicationsInvolved().isEmpty(),
            "Alert should list involved medications");
    }
}
```

- [ ] **Step 2: Update FlinkJobOrchestrator** (same pattern as Module 7)

Read orchestrator first:
```bash
grep -n "Module8\|module8\|comorbidity" \
  src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java
```

Add case to switch:
```java
case "comorbidity":
case "module8":
case "comorbidity-interaction":
    launchComorbidityEngine(env);
    break;
```

Add `launchComorbidityEngine()` method — follows dual-sink pattern from Module 6/7:

```java
/**
 * R10: Explicit launcher with dual-sink wiring.
 * Main output → alerts.comorbidity-interactions
 * HALT side-output → ingestion.safety-critical (patient safety fast-path)
 */
private static void launchComorbidityEngine(StreamExecutionEnvironment env) {
    LOG.info("Launching Module 8: Comorbidity Interaction Engine pipeline");

    String bootstrap = KafkaConfigLoader.getBootstrapServers();

    // Source: CanonicalEvent from enriched-patient-events-v1
    KafkaSource<CanonicalEvent> source = KafkaSource.<CanonicalEvent>builder()
        .setBootstrapServers(bootstrap)
        .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
        .setGroupId("flink-module8-comorbidity-engine-v2")
        .setStartingOffsets(OffsetsInitializer.latest())
        .setValueOnlyDeserializer(new CanonicalEventDeserializer())
        .build();

    SingleOutputStreamOperator<CIDAlert> alerts = env
        .fromSource(source,
            WatermarkStrategy.<CanonicalEvent>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                .withTimestampAssigner((e, ts) -> e.getEventTime()),
            "Kafka Source: Enriched Patient Events (Module 8)")
        .keyBy(CanonicalEvent::getPatientId)
        .process(new Module8_ComorbidityEngine())
        .uid("module8-comorbidity-engine")
        .name("Module 8: Comorbidity Interaction Engine");

    // Main output → alerts.comorbidity-interactions (all severities)
    alerts.sinkTo(
        KafkaSink.<CIDAlert>builder()
            .setBootstrapServers(bootstrap)
            .setRecordSerializer(
                KafkaRecordSerializationSchema.<CIDAlert>builder()
                    .setTopic(KafkaTopics.ALERTS_COMORBIDITY_INTERACTIONS.getTopicName())
                    .setValueSerializationSchema(new JsonSerializer<CIDAlert>())
                    .build())
            .build()
    ).name("Sink: Comorbidity Alerts");

    // HALT side-output → ingestion.safety-critical (fast-path, never suppressed)
    alerts.getSideOutput(Module8_ComorbidityEngine.HALT_SAFETY_TAG).sinkTo(
        KafkaSink.<CIDAlert>builder()
            .setBootstrapServers(bootstrap)
            .setRecordSerializer(
                KafkaRecordSerializationSchema.<CIDAlert>builder()
                    .setTopic(KafkaTopics.INGESTION_SAFETY_CRITICAL.getTopicName())
                    .setValueSerializationSchema(new JsonSerializer<CIDAlert>())
                    .build())
            .build()
    ).name("Sink: HALT Safety-Critical Alerts");

    LOG.info("Module 8 Comorbidity Engine pipeline configured: "
        + "source=[{}], sinks=[{}, {}]",
        KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName(),
        KafkaTopics.ALERTS_COMORBIDITY_INTERACTIONS.getTopicName(),
        KafkaTopics.INGESTION_SAFETY_CRITICAL.getTopicName());
}
```

> **Note:** `Module8_ComorbidityEngine.HALT_SAFETY_TAG` is defined as
> `public static final OutputTag<CIDAlert> HALT_SAFETY_TAG = new OutputTag<>("halt-safety-critical"){};`
> in the operator class (Task 7). The `CanonicalEventDeserializer` already exists in the orchestrator
> (used by Module 3). `JsonSerializer<CIDAlert>` uses Jackson — same pattern as Module 6.

- [ ] **Step 2b: Add Module8ProcessElementTest (R9)**

This test verifies the **full wiring** — state update from a `CanonicalEvent`, rule evaluation, and side-output routing — which pure evaluator tests can't cover.

Create: `src/test/java/com/cardiofit/flink/operators/Module8ProcessElementTest.java`

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.apache.flink.streaming.api.operators.KeyedProcessOperator;
import org.apache.flink.streaming.util.KeyedOneInputStreamOperatorTestHarness;
import org.apache.flink.api.common.typeinfo.Types;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * R9: Integration test that exercises the real KeyedProcessFunction.
 * Uses Flink's test harness to simulate processElement() calls
 * with proper keyed state, watermarks, and side-output collection.
 */
class Module8ProcessElementTest {

    private KeyedOneInputStreamOperatorTestHarness<String, CanonicalEvent, CIDAlert> harness;

    @BeforeEach
    void setUp() throws Exception {
        Module8_ComorbidityEngine engine = new Module8_ComorbidityEngine();
        harness = new KeyedOneInputStreamOperatorTestHarness<>(
            new KeyedProcessOperator<>(engine),
            CanonicalEvent::getPatientId,
            Types.STRING
        );
        harness.open();
    }

    @AfterEach
    void tearDown() throws Exception {
        if (harness != null) harness.close();
    }

    @Test
    @DisplayName("HALT alert emitted as both main output and side-output for triple whammy")
    void tripleWhammy_emitsHALTViaSideOutput() throws Exception {
        String patientId = "P-HARNESS-TW";
        long now = System.currentTimeMillis();

        // 1) Medication events to build state
        harness.processElement(buildMedEvent(patientId, "lisinopril", "ACEI", now - 5000), now - 5000);
        harness.processElement(buildMedEvent(patientId, "dapagliflozin", "SGLT2I", now - 4000), now - 4000);
        harness.processElement(buildMedEvent(patientId, "hydrochlorothiazide", "THIAZIDE", now - 3000), now - 3000);

        // 2) Lab event with eGFR drop >20% (baseline 62, current 45 = 27% drop)
        harness.processElement(buildLabEvent(patientId, "eGFR", 62.0, now - 15L * 86_400_000L), now - 15L * 86_400_000L);
        harness.processElement(buildLabEvent(patientId, "eGFR", 45.0, now), now);

        // 3) Check main output contains CID-01
        List<CIDAlert> mainOutput = harness.extractOutputValues();
        assertTrue(mainOutput.stream().anyMatch(a -> "CID_01".equals(a.getRuleId())),
            "CID-01 Triple Whammy should appear in main output");

        // 4) Check side-output contains same HALT alert
        var sideOutput = harness.getSideOutput(Module8_ComorbidityEngine.HALT_SAFETY_TAG);
        assertFalse(sideOutput.isEmpty(),
            "HALT side-output should have at least one record");
        assertTrue(sideOutput.stream().anyMatch(r -> "CID_01".equals(r.getValue().getRuleId())),
            "CID-01 should be routed to HALT safety-critical side-output");
    }

    @Test
    @DisplayName("Safe patient produces no alerts from processElement")
    void safePatient_noOutput() throws Exception {
        String patientId = "P-HARNESS-SAFE";
        long now = System.currentTimeMillis();

        // Single benign medication
        harness.processElement(buildMedEvent(patientId, "amlodipine", "CCB", now), now);

        // Normal vitals
        harness.processElement(buildVitalEvent(patientId, "sbp", 125.0, now), now);

        List<CIDAlert> output = harness.extractOutputValues();
        assertTrue(output.isEmpty(), "Safe patient should produce zero alerts");
    }

    @Test
    @DisplayName("State accumulates across multiple events for same patient")
    void stateAccumulates_acrossEvents() throws Exception {
        String patientId = "P-HARNESS-ACCUM";
        long now = System.currentTimeMillis();

        // First event: NSAID alone — no alert
        harness.processElement(buildMedEvent(patientId, "ibuprofen", "NSAID", now - 2000), now - 2000);
        assertTrue(harness.extractOutputValues().isEmpty(), "NSAID alone: no alert");

        // Second event: ARB — now NSAID + ARB triggers CID-15 SOFT_FLAG
        harness.processElement(buildMedEvent(patientId, "losartan", "ARB", now), now);
        List<CIDAlert> output = harness.extractOutputValues();
        assertTrue(output.stream().anyMatch(a -> "CID_15".equals(a.getRuleId())),
            "CID-15 should fire after ARB added to existing NSAID");
    }

    // --- Test event builders ---

    private CanonicalEvent buildMedEvent(String patientId, String drugName, String drugClass, long time) {
        return CanonicalEvent.builder()
            .id("evt-" + System.nanoTime())
            .patientId(patientId)
            .eventType(EventType.MEDICATION_ORDERED)
            .eventTime(time)
            .payload(Map.of("drugName", drugName, "drugClass", drugClass, "dose", 10.0))
            .build();
    }

    private CanonicalEvent buildLabEvent(String patientId, String labName, double value, long time) {
        return CanonicalEvent.builder()
            .id("evt-" + System.nanoTime())
            .patientId(patientId)
            .eventType(EventType.LAB_RESULT)
            .eventTime(time)
            .payload(Map.of("labName", labName, "value", value))
            .build();
    }

    private CanonicalEvent buildVitalEvent(String patientId, String vitalName, double value, long time) {
        return CanonicalEvent.builder()
            .id("evt-" + System.nanoTime())
            .patientId(patientId)
            .eventType(EventType.VITAL_SIGN)
            .eventTime(time)
            .payload(Map.of("vitalName", vitalName, "value", value))
            .build();
    }
}
```

> **Why this test matters (R9):** The evaluator unit tests (Tasks 3-5) test rule logic in isolation
> with hand-crafted `ComorbidityState`. But bugs can hide in the glue:
> - `updateStateFromEvent()` might not populate state fields correctly from `CanonicalEvent.payload`
> - The `OutputTag` side-output routing for HALT alerts could be misconfigured
> - State TTL configuration might prevent state accumulation
>
> `KeyedOneInputStreamOperatorTestHarness` exercises the real Flink operator lifecycle without
> needing a cluster — it's the standard Flink testing approach for `KeyedProcessFunction` operators.

- [ ] **Step 3: Run integration tests**

```bash
cd backend/shared-infrastructure/flink-processing && mvn test -pl . \
  -Dtest="Module8IntegrationTest,Module8ProcessElementTest" -q 2>&1 | tail -20
```

Expected: All 9 tests PASS (6 from IntegrationTest + 3 from ProcessElementTest)

- [ ] **Step 4: Commit**

```bash
git add src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java \
  src/test/java/com/cardiofit/flink/operators/Module8IntegrationTest.java \
  src/test/java/com/cardiofit/flink/operators/Module8ProcessElementTest.java
git commit -m "feat(module8): wire ComorbidityEngine into orchestrator + integration tests"
```

---

## Task 9: Full Test Suite + Final Verification

- [ ] **Step 1: Run ALL Module 8 tests**

```bash
cd backend/shared-infrastructure/flink-processing && mvn test -pl . \
  -Dtest="Module8*" 2>&1 | tail -25
```

Expected: All ~44 tests PASS across 5 test classes:
- Module8HALTRulesTest (12 tests)
- Module8PAUSERulesTest (10 tests)
- Module8SOFTFLAGRulesTest (11 tests)
- Module8SuppressionTest (5 tests)
- Module8IntegrationTest (6 tests)

- [ ] **Step 2: Run full compilation**

```bash
cd backend/shared-infrastructure/flink-processing && mvn compile -pl . -q 2>&1 | tail -5
```

- [ ] **Step 3: Verify no regressions**

```bash
cd backend/shared-infrastructure/flink-processing && mvn test -pl . 2>&1 | tail -25
```

---

## Key Technical Decisions

| Decision | Rationale |
|---|---|
| HALT evaluated first, never suppressed | Patient safety — Triple Whammy AKI must reach physician even during alert fatigue |
| Static evaluators with zero Flink deps | Fully unit-testable. Mirrors Module 7 pattern. |
| 72-hour suppression window | Balances alert fatigue vs. clinical safety. HALT exempt. |
| Suppression key includes medication hash | Same rule + different drug combo = new alert (different clinical scenario) |
| ComorbidityState stores meds by drug class | Enables class-level rule evaluation without drug name lookup tables |
| SOFT_FLAG accepts SBP target parameter | CID-13 needs KB-20's patient-specific target. Passed as parameter, not hardcoded. |
| CanonicalEvent payload extraction is illustrative | Actual field names depend on on-disk schema. MUST verify before deployment. |
| CID-10 uses 14-day glucose AND BP trajectory | Both must worsen simultaneously — single-domain deterioration is handled by domain-specific modules |
| CID-04 insulin dose reduction detection deferred | Requires comparing current vs. previous medication doses in state. Phase 2. |
| DD#7 rule numbering as authoritative | V4 architecture renumbered rules inconsistently. DD#7 has full clinical spec. |

## Deferred (Not in Scope)

1. **V4-specific rules** — Severe Hyponatremia, Recurrent Hypoglycemia, Volume Depletion, Heart Rate Masking, Expected eGFR Dip — to be added as CID-18+ in Phase 2
2. **Elderly polypharmacy extensions** — CID-20 (ACB ≥3), CID-21 (prescribing cascade), CID-22 (pill burden >10), CID-23 (duplicate class) — requires KB-5 ACB scoring integration
3. **STOPP/START 190 rules** — KB-24 YAML configuration, not Module 8 code. Separate implementation.
4. **CID-04 insulin dose reduction detection** — Requires medication change history tracking
5. **CID-13 SBP target from KB-20** — Requires broadcast state from KB-20 or enrichment in event payload
6. **CID-16 salt sensitivity phenotype** — Depends on Module 10b salt_sensitivity_beta computation
7. **CID-17 cultural calendar** — Depends on localization bundle (Deep Dive #0 Component 7)
8. **Alert lifecycle management** — ACKNOWLEDGED → ACTIONED → RESOLVED → EXPIRED state machine. Requires downstream consumer (notification-service) integration.
9. **Physician feedback rule evolution** — CONTRAINDICATION_KNOWN → new CID rule pipeline (DD#4 Section 8)
10. **Alert frequency analytics dashboard** — Per-rule fire rate, acknowledgment rate, false positive tracking
