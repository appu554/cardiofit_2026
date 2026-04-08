# CardioFit Flink Pipeline — Full Gap Analysis Report

**Date**: 2026-04-05
**Scope**: Modules 1–13, v3.0 Enhancements (E1–E7, A1–A5), Pre-Pilot Blockers
**Branch**: `feature/kb25-kb26-implementation`

---

## Section 1 — Module 13 Implementation Plan Corrections

Seven issues found in `docs/superpowers/plans/2026-04-05-module13-clinical-state-synchroniser.md` that will cause compile failures or silent runtime bugs.

### M13-1: TrajectoryAttribution.INSUFFICIENT does not exist

| Field | Value |
|-------|-------|
| **Severity** | COMPILE_ERROR |
| **Location** | Module13TestBuilder, StateChangeDetectorTest, IntegrationTest |
| **Root Cause** | Plan references `TrajectoryAttribution.INSUFFICIENT` but the actual enum value is `INTERVENTION_INSUFFICIENT` |
| **Fix** | Replace every occurrence of `.INSUFFICIENT` with `.INTERVENTION_INSUFFICIENT` |

### M13-2: CanonicalEvent.Builder.processingTime() does not exist

| Field | Value |
|-------|-------|
| **Severity** | COMPILE_ERROR |
| **Location** | Module13TestBuilder.baseEvent() |
| **Root Cause** | The Builder class has no `processingTime()` method — the field is set by the default constructor (`this.processingTime = System.currentTimeMillis()`) |
| **Fix** | Remove `.processingTime(Instant.now().toEpochMilli())` from the builder chain |

### M13-3: Wrong deserializer for 6 of 7 source topics (CONNECTED FAILURE)

| Field | Value |
|-------|-------|
| **Severity** | RUNTIME — silent data loss |
| **Location** | FlinkJobOrchestrator `launchClinicalStateSynchroniser()` |
| **Root Cause** | Plan uses `CanonicalEventDeserializer` for all 7 sources. Only `enriched-patient-events-v1` carries native CanonicalEvent JSON. The other 6 topics (CDS alerts, ML predictions, action decisions, ARV classifications, engagement scores, intervention deltas) carry domain-specific JSON that is **not** CanonicalEvent. |
| **Fix** | Use `JsonToCanonicalEventDeserializer(topicName)` for the 6 non-canonical topics. This wraps raw JSON into `CanonicalEvent.payload` and injects `_source_topic` as a discriminator. |

**Source mapping:**

| Topic | Carries | Deserializer |
|-------|---------|-------------|
| `enriched-patient-events-v1` | Native CanonicalEvent | `CanonicalEventDeserializer` |
| `flink.cds-alerts` | CDS alert JSON | `JsonToCanonicalEventDeserializer` |
| `flink.ml-predictions` | ML prediction JSON | `JsonToCanonicalEventDeserializer` |
| `flink.action-decisions` | Action decision JSON | `JsonToCanonicalEventDeserializer` |
| `flink.arv-classifications` | ARV result JSON | `JsonToCanonicalEventDeserializer` |
| `flink.engagement-scores` | Engagement JSON | `JsonToCanonicalEventDeserializer` |
| `flink.intervention-deltas` | Delta JSON | `JsonToCanonicalEventDeserializer` |

### M13-4: source_module routing discriminator does not exist (CONNECTED FAILURE)

| Field | Value |
|-------|-------|
| **Severity** | RUNTIME — NPE or mis-routing |
| **Location** | Module13_ClinicalStateSynchroniser.processElement() |
| **Root Cause** | Plan routes events by `payload.get("source_module")` but `JsonToCanonicalEventDeserializer` does not inject `source_module`. It injects `_source_topic` (the Kafka topic name). |
| **Fix** | Route by `_source_topic` using a topic-name → module mapping: |

```java
private static final Map<String, String> TOPIC_TO_MODULE = Map.of(
    "flink.cds-alerts",          "MODULE_3",
    "flink.ml-predictions",      "MODULE_5",
    "flink.action-decisions",    "MODULE_6",
    "flink.arv-classifications", "MODULE_7",
    "flink.engagement-scores",   "MODULE_9",
    "flink.intervention-deltas", "MODULE_12"
);
```

### M13-5: Test factories mask routing bugs (CONNECTED FAILURE)

| Field | Value |
|-------|-------|
| **Severity** | TEST_VALIDITY |
| **Location** | Module13TestBuilder factory methods |
| **Root Cause** | Test factories manually inject `source_module` into the payload, creating false confidence that routing works. Real events produced by `JsonToCanonicalEventDeserializer` have `_source_topic`, not `source_module`. |
| **Fix** | Test factories must construct events the same way the deserializer does — inject `_source_topic` with actual topic names, not `source_module` with module names. |

### M13-6: Missing KafkaTopics enum constant

| Field | Value |
|-------|-------|
| **Severity** | COMPILE_ERROR |
| **Location** | `KafkaTopics.java` |
| **Root Cause** | Module 13 output topic `clinical.state-change-events` is not registered in the KafkaTopics enum |
| **Fix** | Add `CLINICAL_STATE_CHANGE_EVENTS("clinical.state-change-events", 4, 90)` after `FLINK_INTERVENTION_DELTAS` at line 181, and add it to `isV4OutputTopic()` at lines 314–328 |

### M13-7: FlinkJobOrchestrator switch case missing

| Field | Value |
|-------|-------|
| **Severity** | RUNTIME — module won't launch |
| **Location** | `FlinkJobOrchestrator.java` lines 68–155 |
| **Root Cause** | No `case "module13":` in the module launch switch statement |
| **Fix** | Add case before `default:` calling `launchClinicalStateSynchroniser()` |

**Connected failure summary**: M13-3 + M13-4 + M13-5 form a single coherent bug. Fixing the deserializer (M13-3) forces fixing the discriminator (M13-4) which forces fixing the test factories (M13-5). They must be addressed together.

---

## Section 2 — V3.0 Document Accuracy Fixes

Six discrepancies between the v3.0 architecture document (`Vaidshala_V4_Flink_Architecture_v3.docx`) and the actual codebase.

### DOC-1: Test count undercount

| Field | Value |
|-------|-------|
| **Severity** | DOCUMENTATION |
| **Actual vs Claimed** | 328+ actual tests vs 252+ claimed |
| **Details** | Module 8: 70 actual (doc says 22), Module 9: 83 actual (doc says 18), Module 7: 47 actual (doc says 31), Module 10/10b: 43 actual (doc says 51), Module 11/11b: 47 actual (doc says 54) |
| **Fix** | Update test counts in v3.0 document to reflect actual numbers |

### DOC-2: MealTimeCategory enum missing EARLY/LATE/VERY_LATE

| Field | Value |
|-------|-------|
| **Severity** | CODE_GAP |
| **Details** | Enhancement A3 (Chrono-nutrition Windows) requires EARLY_MORNING, LATE_NIGHT, VERY_LATE variants. Current enum only has BREAKFAST, LUNCH, DINNER, SNACK. Document states A3 is "ready" but enum hasn't been extended. |
| **Fix** | Either extend MealTimeCategory or create a ChronoNutritionWindow enum. Document should mark A3 as PARTIAL. |

### DOC-3: Enhancement A1 phase placement contradiction

| Field | Value |
|-------|-------|
| **Severity** | PLANNING |
| **Details** | A1 (Personalized Targets) is placed in Phase 6 (lowest priority). But Gap G9 (velocity-vs-displacement scoring) is marked HIGH severity in Phase 5. G9 depends on A1 for personalized baselines. |
| **Fix** | Move A1 to Phase 5 or before G9 in the implementation sequence |

### DOC-4: GMI calculator missing from KB-8

| Field | Value |
|-------|-------|
| **Severity** | CODE_GAP |
| **Details** | Document references GMI (Glucose Management Indicator) calculation for Enhancement E5 (CKM Syndrome Scoring). No GMI calculator exists in KB-8 or any other KB service. |
| **Fix** | Implement GMI calculator in KB-8 or inline within Module 8 comorbidity engine |

### DOC-5: Module 7 ARV window sizes not configurable

| Field | Value |
|-------|-------|
| **Severity** | OPERATIONAL |
| **Details** | Document states ARV windows are configurable (7-day, 14-day, 30-day). Current implementation uses hardcoded 14-day window only. |
| **Fix** | Add Flink configuration parameters for ARV window sizes |

### DOC-6: Consumer group naming convention not enforced

| Field | Value |
|-------|-------|
| **Severity** | OPERATIONAL |
| **Details** | Document specifies convention `flink-{module}-{topic-function}-v{version}`. Several modules use ad-hoc naming. |
| **Fix** | Audit and standardize all consumer group IDs across modules |

---

## Section 3 — Flink Pipeline Gaps (Modules 1–12)

Eight gaps found across the existing pipeline that affect v4.0 readiness.

### PIPE-1: Module 1b DLQ lacks alerting

| Field | Value |
|-------|-------|
| **Severity** | HIGH |
| **Module** | 1b (Canonicalizer) |
| **Details** | Dead-letter queue exists but has no monitoring, no alerting, and no replay mechanism. Malformed events silently accumulate. |
| **Impact** | Data loss goes undetected until clinical audit |
| **Fix** | Add DLQ depth metric + Prometheus alert rule + replay script |

### PIPE-2: Module 2 enrichment — KB-20 fallback not implemented

| Field | Value |
|-------|-------|
| **Severity** | HIGH |
| **Module** | 2 (Patient Context Enrichment) |
| **Details** | When KB-20 (Patient Profile) is unreachable, Module 2 drops the event. No cached patient context fallback. |
| **Impact** | Transient KB-20 outage causes complete pipeline stall for affected patients |
| **Fix** | Add Flink ValueState cache of last-known patient context with 1-hour TTL |

### PIPE-3: Module 3 CDS — missing AHA 2023 CKM pathway

| Field | Value |
|-------|-------|
| **Severity** | MEDIUM |
| **Module** | 3 (Comprehensive CDS) |
| **Details** | CDS rules cover ACC/AHA hypertension and KDIGO CKD but lack the 2023 AHA CKM (Cardiovascular-Kidney-Metabolic) syndrome staging pathway |
| **Impact** | E5 (CKM Syndrome Scoring) enhancement cannot activate without Module 3 CKM rules |
| **Fix** | Add CKM staging rules to Module 3 rule engine |

### PIPE-4: Module 4 — window function race condition under backpressure

| Field | Value |
|-------|-------|
| **Severity** | MEDIUM |
| **Module** | 4 (Pattern Detection) |
| **Details** | Under Kafka backpressure, the tumbling window can fire with partial data if watermark advances past window close before all events arrive. Current watermark strategy uses `BoundedOutOfOrdernessTimestampAssigner` with 5-second tolerance, but clinical events can be delayed 30+ seconds. |
| **Impact** | Pattern detection produces false negatives during high-load periods |
| **Fix** | Increase watermark tolerance to 60 seconds or switch to allowed-lateness with side output for late events |

### PIPE-5: Module 5 ONNX model — no version pinning

| Field | Value |
|-------|-------|
| **Severity** | HIGH |
| **Module** | 5 (ML Inference) |
| **Details** | ONNX models are loaded from filesystem path without version tracking. Model replacement during deployment causes non-deterministic inference during rolling restart. |
| **Impact** | Mixed model versions during deployment produce inconsistent predictions |
| **Fix** | Add model version hash to ML prediction output; verify model hash on startup against expected version in config |

### PIPE-6: Module 6 — acknowledgment timeout hardcoded

| Field | Value |
|-------|-------|
| **Severity** | LOW |
| **Module** | 6 (Clinical Action Engine) |
| **Details** | Alert acknowledgment timeout is hardcoded to 30 minutes. Different alert severities should have different timeout windows (CRITICAL: 15 min, HIGH: 30 min, MEDIUM: 60 min). |
| **Impact** | Over-escalation for medium-severity alerts, under-escalation for critical alerts |
| **Fix** | Make timeout configurable per severity level via Flink properties |

### PIPE-7: Module 8 — comorbidity state not persisted to KB-20

| Field | Value |
|-------|-------|
| **Severity** | MEDIUM |
| **Module** | 8 (Comorbidity Interaction) |
| **Details** | ComorbidityState is maintained in Flink ValueState but not synchronized back to KB-20. Module 13 needs KB-20 to have current comorbidity state for its composite snapshot. |
| **Impact** | Module 13's patient snapshot will lack comorbidity context |
| **Fix** | Add side output from Module 8 to KB-20 state update topic |

### PIPE-8: Module 12 — intervention delta baseline drift

| Field | Value |
|-------|-------|
| **Severity** | MEDIUM |
| **Module** | 12 (Intervention Window Monitor) |
| **Details** | Baseline readings are captured at intervention start but never refreshed. For long-running interventions (30+ days), the baseline becomes stale, making delta computations unreliable. |
| **Impact** | Long-term intervention effectiveness tracking produces misleading deltas |
| **Fix** | Implement rolling baseline with 7-day lookback, refreshed weekly |

---

## Section 4 — Enhancement Readiness Matrix

Status of v3.0 enhancements (E1–E7) and advanced features (A1–A5) against actual codebase state.

### Core Enhancements (E1–E7)

| ID | Enhancement | Status | Blocking Issue |
|----|------------|--------|----------------|
| E1 | 90-day State TTL | **IMPLEMENTED** | Deployed in Modules 7, 8, 9, 12 with `OnReadAndWrite` + `NeverReturnExpired` |
| E2 | Meal-Response Correlation (10/10b) | **IMPLEMENTED** | Module 10b MealPatternAggregator + 10a correlation working |
| E3 | Activity-Response Correlation (11/11b) | **IMPLEMENTED** | Module 11 ActivityResponseCorrelator + 11b estimators deployed |
| E4 | Engagement Monitoring (Module 9) | **IMPLEMENTED** | Channel-aware scoring, trajectory analysis, drop detection |
| E5 | CKM Syndrome Scoring | **PARTIAL** | Missing: AHA 2023 CKM rules in Module 3 (PIPE-3), GMI calculator in KB-8 (DOC-4). Module 8 has comorbidity engine but not CKM-specific composite |
| E6 | Intervention Window Monitoring (Module 12) | **IMPLEMENTED** | Trajectory tracking, adherence assembly, delta computation working. Baseline drift issue (PIPE-8) is non-blocking for pilot |
| E7 | Clinical State Synchronization (Module 13) | **ABSENT** | Not yet implemented. Plan exists but has 7 corrections needed (Section 1) |

### Advanced Features (A1–A5)

| ID | Feature | Status | Blocking Issue |
|----|---------|--------|----------------|
| A1 | Personalized BP Targets | **ABSENT** | No per-patient target override in any module. All thresholds hardcoded. Needed for G9 (velocity-vs-displacement). Phase placement contradiction (DOC-3). |
| A2 | Sex-Aware BP Thresholds | **PARTIAL** | Module 11 has sex-aware thresholds (per recent commit). Modules 3, 7, 8 still use unisex thresholds. |
| A3 | Chrono-nutrition Windows | **PARTIAL** | MealTimeCategory missing EARLY/LATE/VERY_LATE variants (DOC-2). Module 10b aggregation exists but temporal granularity is limited to BREAKFAST/LUNCH/DINNER/SNACK. |
| A4 | Cross-Module Deduplication | **IMPLEMENTED** | Module 6 CrossModuleDedup operator deployed. Hash-based with TTL window. |
| A5 | Alert Fatigue Prevention | **IMPLEMENTED** | Module 6 suppression manager + acknowledgment tracking. Timeout hardcoded issue (PIPE-6) is non-blocking. |

### Readiness Summary

| Category | Implemented | Partial | Absent | Total |
|----------|------------|---------|--------|-------|
| Core (E1–E7) | 5 | 1 | 1 | 7 |
| Advanced (A1–A5) | 2 | 2 | 1 | 5 |
| **Total** | **7** | **3** | **2** | **12** |

---

## Section 5 — Pre-Pilot Deployment Blockers

### 5A — KB Service Dependencies

| ID | Service | Status | Issue |
|----|---------|--------|-------|
| KB-1 | KB-20 (Patient Profile) | **READY** | Service exists with FHIR sync, stratum engine, streaming routes. Must verify write-coalescing under Module 13 load. |
| KB-2 | KB-24 (Safety Constraint) | **READY** | Handlers, server, safety models deployed. Intake rules integrated with onboarding service. |
| KB-3 | KB-25 (Lifestyle Knowledge Graph) | **PARTIAL** | Dockerfile and go.mod exist. Internal graph logic and API completeness unverified. |
| KB-4 | KB-26 (Metabolic Digital Twin) | **PARTIAL** | Dockerfile and go.mod exist. Twin simulation logic and API completeness unverified. |
| KB-5 | KB-8 (GMI Calculator) | **ABSENT** | No GMI calculator. Blocks E5 CKM scoring. |

### 5B — FHIR & Regulatory

| ID | Requirement | Status | Issue |
|----|-------------|--------|-------|
| FR-1 | ABDM Consent Lifecycle | **IMPLEMENTED** | Consent handling exists in ingestion service and intake onboarding. |
| FR-2 | FHIR R4 Observation Compliance | **IMPLEMENTED** | All clinical services implement FHIR R4 resources with validation. |
| FR-3 | Audit Trail for Clinical Decisions | **PARTIAL** | Module 6 action decisions are logged to Kafka. No centralized audit store queryable by compliance team. |

### 5C — Safety Infrastructure

| ID | Requirement | Status | Issue |
|----|-------------|--------|-------|
| SI-4 | V-MCU 3-Channel Safety | **IMPLEMENTED** | Channel A (MCU gate), Channel B (physiology), Channel C (protocol) with 1oo3 veto arbiter. |
| SI-5 | KB-23 Decision Cards | **IMPLEMENTED** | Service running on port 8134. |
| SI-6 | Safety Gateway gRPC | **IMPLEMENTED** | Go-based safety gateway with gRPC communication to clinical reasoning. |
| SI-7 | Circuit Breaker (KB-20 Sink) | **PLANNED** | Module 13 plan includes KB20AsyncSinkFunction with 3-failure/30s-open/1000-buffer pattern. Not yet implemented. |
| SI-8 | DLQ Monitoring | **ABSENT** | Module 1b DLQ exists but has no alerting or replay (PIPE-1). |
| SI-9 | Model Version Pinning | **ABSENT** | Module 5 ONNX models lack version tracking (PIPE-5). |

### 5D — Infrastructure & Observability

| ID | Requirement | Status | Issue |
|----|-------------|--------|-------|
| INFRA-1 | Kafka Topic Provisioning | **PARTIAL** | Most topics exist. Module 13 output topic `clinical.state-change-events` not yet in KafkaTopics enum (M13-6). |
| INFRA-2 | Prometheus Metrics | **PARTIAL** | Some modules expose custom metrics. No standardized metric naming convention across modules. |
| INFRA-3 | Grafana Dashboard | **PARTIAL** | Dashboard exists at grafana.internal. Module 9–13 panels not yet added. |
| INFRA-4 | Consumer Group Audit | **ABSENT** | Consumer group naming convention not enforced (DOC-6). Rogue consumer groups could cause rebalancing storms during deployment. |

### Blocker Priority for Pilot

**Must-fix before pilot (P0):**
1. M13-3/4/5 — Connected deserializer/routing/test bug (compile + runtime failures)
2. M13-1/2 — Compile errors in Module 13
3. PIPE-1 — DLQ monitoring (clinical safety requirement)
4. PIPE-2 — KB-20 fallback (pipeline resilience)
5. PIPE-5 — Model version pinning (inference correctness)

**Should-fix before pilot (P1):**
6. M13-6/7 — KafkaTopics enum + orchestrator switch case
7. PIPE-7 — Module 8 comorbidity state to KB-20 (needed by Module 13)
8. INFRA-4 — Consumer group audit
9. FR-3 — Centralized audit store

**Can defer post-pilot (P2):**
10. A1 — Personalized BP targets
11. A3 — Chrono-nutrition extended windows
12. PIPE-4 — Watermark tolerance tuning
13. PIPE-8 — Baseline drift in Module 12
14. DOC-1 through DOC-6 — Documentation corrections

---

## Appendix — Cross-Reference Index

| Issue ID | Section | Depends On | Blocks |
|----------|---------|------------|--------|
| M13-1 | 1 | — | Module 13 compilation |
| M13-2 | 1 | — | Module 13 compilation |
| M13-3 | 1 | — | M13-4, M13-5 |
| M13-4 | 1 | M13-3 | M13-5 |
| M13-5 | 1 | M13-4 | Module 13 test validity |
| M13-6 | 1 | — | M13-7 |
| M13-7 | 1 | M13-6 | Module 13 launch |
| DOC-2 | 2 | — | A3 |
| DOC-3 | 2 | — | A1 phase scheduling |
| DOC-4 | 2 | — | E5 |
| PIPE-1 | 3 | — | SI-8 |
| PIPE-2 | 3 | — | Pipeline resilience |
| PIPE-3 | 3 | — | E5 |
| PIPE-5 | 3 | — | SI-9 |
| PIPE-7 | 3 | — | Module 13 composite snapshot |
| E5 | 4 | PIPE-3, DOC-4 | CKM scoring |
| E7 | 4 | M13-1–7 | Clinical state sync |
| A1 | 4 | DOC-3 | G9 velocity scoring |
| A3 | 4 | DOC-2 | Chrono-nutrition |
