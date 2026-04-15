# Masked HTN / Clinical Intelligence Platform — Phase 7: Activation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan sub-project-by-sub-project. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Take the six Phase 6 heartbeat sub-projects and activate them end-to-end — wire real data sources, persist outputs as proper DecisionCards (via YAML templates + the confidence-tier + MCU-gate pipeline), emit Prometheus metrics, and publish events to downstream consumers. After Phase 7, no Phase 6 sub-project is in "heartbeat mode" anymore — either it's shipping real production effects or it's explicitly documented as blocked with a named upstream dependency.

**Architecture:** Phase 7 is **not** new feature work. It's the consolidation phase that turns Phase 6's proven abstractions into production-running clinical loops. Five of the six sub-projects (P7-A through P7-D and P7-F) are Go/KB-service wiring — those follow the Phase 6 effort pattern and will deflate 3-5× from their upper-bound estimates. The sixth (P7-E) is the Module 3 CGM streaming pipeline — Java/Flink work in an existing cluster that already hosts 13 modules, following the `HpiCalibrationStreamJob.java` template for a new streaming job that wires existing `CGMReadingBuffer` + `Module3_CGMAnalytics.computeMetrics` components into a running pipeline. **Effort estimates for P7-E should be treated as predictions, not upper bounds** — Flink engineering has irreducible complexity (watermark strategies, checkpoint semantics, serialization, mini-cluster tests) that doesn't deflate the way KB service wiring does.

**Pre-requisite:** Phase 6 shipped. `feature/v4-clinical-gaps` is at HEAD `9c92ba8a` with P6-1 through P6-6 committed and pushed to origin. KB-20, KB-23, KB-26 all build and test green. The Flink cluster is running with modules 1-13 already deployed.

**Tech Stack:** Go 1.25 (existing KB services), Java + Apache Flink (existing `flink-processing` Maven project), PostgreSQL 15, existing `clinical.state-changes.v1` + `clinical.priority-events.v1` Kafka topics, new `clinical.cgm-analytics.v1` topic for P7-E.

---

## Status Snapshot — 2026-04-15

- [ ] **P7-A — Reactive Renal Dose Gating Activation:** Card persistence (`RENAL_CONTRAINDICATION` + `RENAL_DOSE_REDUCE` templates) + metrics + KB-20 sub-recon on `deriveEGFR` outbox path. Upper bound ~0.5 day, expect ~1-2 hours.
- [ ] **P7-B — CKM 4c Mandatory Med Checker Activation:** Card persistence (`CKM_4C_MANDATORY_MEDICATION` template) + metrics + **resolve the unresolved upstream gap: nothing currently invokes `CKMTransitionPublisher.PublishStageTransition` in production**. Upper bound ~1 day, expect ~2-3 hours.
- [ ] **P7-C — Renal Anticipatory Monthly Activation:** Real `RenalAnticipatoryOrchestrator` + `RenalActivePatientLister` implementation + `RENAL_THRESHOLD_APPROACHING` + `STALE_EGFR` templates + metrics. Upper bound ~1 day, expect ~2-4 hours.
- [ ] **P7-D — Inertia Weekly Batch Activation:** Largest KB service sub-project. KB-20 intervention timeline service + HTTP endpoint, KB-26 target-status HTTP exposure, KB-23 `InertiaInputAssembler` + `InertiaActivePatientLister`, `INERTIA_DETECTED` + `DUAL_DOMAIN_INERTIA_DETECTED` templates, `inertia_verdict_history` table + repository, stability dampening, metrics. Upper bound ~2 days, expect ~4-6 hours.
- [ ] **P7-E — Module 3 CGM Streaming Pipeline:** Java/Flink sub-project, different toolset. New `Module3_CGMStreamJob.java` wiring `ingestion.cgm-raw` → `CGMReadingBuffer` → `Module3_CGMAnalytics.computeMetrics` → new `clinical.cgm-analytics.v1` topic, plus KB-26 Go consumer persisting `CGMAnalyticsEvent` → existing `cgm_period_reports` table. **Estimate ~3 days is a prediction, NOT an upper bound** — Flink work does not deflate like KB service wiring.
- [ ] **P7-F — Domain Trajectory Kafka Publisher Swap:** Replace `NoopTrajectoryPublisher` with `KafkaTrajectoryPublisher` in KB-26 `main.go`. 15-minute config change. Trivial cleanup that was flagged in Phase 6 P6-3 as `TODO(kb26-kafka)`.

**Total remaining effort: ~5 days upper bound across 6 sub-projects.** P7-A through P7-D + P7-F should ship in a single session (~1-2 hours each with deflation). P7-E is a dedicated multi-session sub-project on its own tempo.

---

## Locked Decisions

These are **not** open questions. They are fixed constraints derived from the Phase 6 retrospective review and the P7-E CGM recon report.

### Decision 1: Phase 7 ships real DecisionCards, not log-only stubs

Every Phase 6 sub-project deferred card persistence with the rationale "DecisionCard is template-driven." Phase 7 reverses that by building the templates. Each activation includes one or more YAML card templates in `kb-23-decision-cards/templates/` following the existing template structure (`template_id`, `node_id`, `differential_id`, `version`, `clinical_reviewer`, `mcu_gate_default`, `card_source`, `card_type`, `trigger_event`, `trigger_condition`, `sla_hours`, `priority`, `confidence_tier`, `recommendations`, `fragments`), then invokes the card builder via `TemplateLoader.Get(templateID)` + `CardBuilder.Build(...)` pattern from the existing masked HTN templates (Phase 4 P7) as the reference.

**Implication**: if a sub-project can't ship a proper YAML template because the template shape doesn't fit the detection output, the sub-project is **blocked**, not downgraded to log-stub mode. Log-stub was Phase 6's acceptable deferral; Phase 7 is the honest reckoning.

### Decision 2: Go `ComputePeriodReport` is marked provisional, NOT deleted

Per the Phase 6 retrospective review's refinement: the P6-4 Go files (`cgm_period_report.go`, `cgm_daily_batch.go`) stay on disk as a fallback + executable specification for the Java Module 3 pipeline. Each file gets a header comment:

```
// PROVISIONAL: This duplicates Module3_CGMAnalytics.computeMetrics (Java).
// Canonical source is the Module 3 Flink streaming pipeline (P7-E).
// When the Flink pipeline is confirmed producing CGMAnalyticsEvent on
// clinical.cgm-analytics.v1, delete this file and consume from the
// topic via KB-26's existing consumer infrastructure.
// DO NOT extend this file — deepen the Java Module 3 side instead.
```

Deletion is a P7-E final task after the Flink pipeline is verified producing events end-to-end.

### Decision 3: P7-E uses HpiCalibrationStreamJob as the structural template

The existing `flink-processing/src/main/java/com/cardiofit/flink/operators/HpiCalibrationStreamJob.java` is a production streaming job with Kafka source, event-time watermarks, keyed state, sliding windows, and Kafka sink. P7-E's `Module3_CGMStreamJob.java` copies this structure exactly, substituting:
- Source topic: `ingestion.cgm-raw` (already defined in `runtime-layer/config/kafka-topics.yaml:306`)
- Keyed state: existing `CGMReadingBuffer.java`
- Window: sliding 14-day window with 24-hour slide (so each day's fire produces a fresh 14-day rolling report)
- Compute: existing `Module3_CGMAnalytics.computeMetrics(readings, 14)`
- Sink topic: **new** `clinical.cgm-analytics.v1` (to be added to `kafka-topics.yaml`)
- Output shape: existing `CGMAnalyticsEvent.java`

**No new Module 3 logic is written.** P7-E is pure composition of existing components into a running job.

### Decision 4: KB-26 is the Postgres sink for Module 3 CGM output

The `cgm_period_reports` Postgres table lives in KB-26's migrations (`005_cgm_tables.sql`). Rather than giving Flink direct Postgres write access, Module 3 publishes `CGMAnalyticsEvent` to the new `clinical.cgm-analytics.v1` Kafka topic, and KB-26 adds a new consumer that persists each event into the existing table. This keeps Flink stateless from KB-26's perspective and preserves the "Kafka is the single integration bus" pattern that P6-2 and P6-6 established.

### Decision 5: P7-B resolves the CKM trigger gap inside KB-20, not as a cross-service task

The Phase 6 retrospective flagged that `CKMTransitionPublisher.PublishStageTransition` has no caller in production. P7-B adds the caller inside KB-20's FHIR sync worker: when a new `Observation` with `LabType = "LVEF"` or `LabType = "NT_PRO_BNP"` or `LabType = "CAC_SCORE"` is synced, the worker invokes a `CKMRecomputationService.RecomputeAndPublish(patientID, triggeredByObservationID)` helper that assembles `CKMClassifierInput` from the patient profile + the new observation, calls `ClassifyCKMStage`, and hands the result to `CKMTransitionPublisher`. **No new FHIR Condition sync is built** — only Observation-driven recomputation, which is tractable within the existing `syncObservations` path. Condition-driven recomputation remains a Phase 8 follow-up when FHIR Condition sync ships.

### Decision 6: P7-D's KB-20 intervention timeline service is a new HTTP endpoint, not a shared package

The Phase 6 plan flirted with moving the inertia detector to a shared-infrastructure package. P6-1 abandoned that by moving the batch to KB-23 instead. Phase 7 P7-D stays in KB-23 for the orchestrator and adds a **new KB-20 HTTP endpoint** `GET /api/v1/patient/:id/intervention-timeline` that returns `{domains: {GLYCAEMIC: {latest_drug_class, latest_action_date, days_since}, HEMODYNAMIC: {...}, RENAL: {...}}}`. KB-23's `InertiaInputAssembler` calls it via the existing `KB20Client` (extended with a `FetchInterventionTimeline` method). No cross-service package moves.

### Decision 7: P7-D's KB-26 target-status exposure is a new HTTP endpoint, not an orchestrator-side import

The target-status functions (`ComputeGlycaemicTargetStatus`, `ComputeHemodynamicTargetStatus`, `ComputeRenalTargetStatus`) live in KB-26's `target_status.go`. P7-D exposes them via a new `GET /api/v1/patient/:id/target-status` endpoint that returns `{glycaemic: {met, days_since_met, ...}, hemodynamic: {...}, renal: {...}}`. KB-23's `InertiaInputAssembler` calls both KB-20 (timeline) and KB-26 (target status) per patient. One round trip per service per patient; the batch is bounded-concurrency (default 4) to keep total RTT manageable.

### Decision 8: Stability dampening for P7-D reuses the pkg/stability engine from Phase 5 P5-1

The inertia verdict stability check ("don't flap an INERTIA_DETECTED event on and off between weeks") uses the existing `stability.Engine` from KB-26's `pkg/stability`. Since KB-23 doesn't currently import KB-26's internal `pkg`, P7-D either (a) copies the stability package into KB-23 as another intentional duplicate (like the BatchScheduler copy from P6-5), or (b) moves `pkg/stability` into `shared-infrastructure/clinical-intelligence/stability` as a real shared package. **Decision: copy it.** Rationale: the stability engine is 160 lines, the duplication is intentional, and shared-infrastructure package moves repeatedly prove to be disproportionate work. When a third consumer appears, promote to shared.

### Decision 9: Prometheus metrics ship alongside card persistence, not in a separate follow-up

Every Phase 7 sub-project registers its metrics in the same commit that wires card persistence. No separate metrics commit. The registered metrics per sub-project:

| Sub-project | Metrics |
|---|---|
| P7-A | `renal_contraindication_detected_total{drug_class}`, `renal_dose_reduce_detected_total{drug_class}`, `renal_reactive_latency_seconds` |
| P7-B | `ckm_stage_transitions_total{from,to}`, `ckm_4c_mandatory_med_alerts_total{drug_class}`, `ckm_recomputation_total{trigger}` |
| P7-C | `renal_anticipatory_alerts_total{drug_class,horizon}`, `stale_egfr_total{ckd_stage}`, `renal_anticipatory_batch_duration_seconds` |
| P7-D | `inertia_detected_total{domain,severity}`, `inertia_batch_duration_seconds`, `inertia_batch_patients_total{outcome}` |
| P7-E | `cgm_period_reports_persisted_total`, `cgm_tir_distribution_bucket`, `cgm_gri_zone_total{zone}`, `cgm_consumer_lag_seconds` |
| P7-F | (no new metrics — `kb26_trajectory_*` already exist from P6-3) |

### Decision 10: Every sub-project ends with Verification Questions — permanent standard

Carried forward from Phase 5 + Phase 6 as the mandatory pattern. Implementer's completion report answers each with yes / no / partial / N/A plus evidence. No narrative. The effort estimates are informational; the verification questions are the correctness contract.

---

## File Structure Overview

### KB-20 changes (P7-B + P7-D)

| Action | File | Sub-project |
|---|---|---|
| Create | `kb-20-patient-profile/internal/services/ckm_recomputation_service.go` | P7-B — assembles CKMClassifierInput + calls publisher |
| Create | `kb-20-patient-profile/internal/services/ckm_recomputation_service_test.go` | P7-B |
| Modify | `kb-20-patient-profile/internal/fhir/fhir_sync_worker.go` | P7-B — invoke CKM recomputation on LVEF/NTproBNP/CAC observations |
| Create | `kb-20-patient-profile/internal/services/intervention_timeline_service.go` | P7-D — query medication_states → per-domain latest action |
| Create | `kb-20-patient-profile/internal/services/intervention_timeline_service_test.go` | P7-D |
| Create | `kb-20-patient-profile/internal/api/intervention_handlers.go` | P7-D — `GET /api/v1/patient/:id/intervention-timeline` |
| Modify | `kb-20-patient-profile/internal/api/routes.go` | P7-D — register route |

### KB-26 changes (P7-D + P7-E + P7-F)

| Action | File | Sub-project |
|---|---|---|
| Create | `kb-26-metabolic-digital-twin/internal/api/target_status_handlers.go` | P7-D — `GET /api/v1/patient/:id/target-status` |
| Modify | `kb-26-metabolic-digital-twin/internal/api/routes.go` | P7-D — register route |
| Create | `kb-26-metabolic-digital-twin/internal/services/cgm_analytics_consumer.go` | P7-E — Kafka consumer on `clinical.cgm-analytics.v1` |
| Create | `kb-26-metabolic-digital-twin/internal/services/cgm_analytics_consumer_test.go` | P7-E |
| Create | `kb-26-metabolic-digital-twin/internal/services/cgm_period_report_repository.go` | P7-E — writes to existing `cgm_period_reports` table |
| Modify | `kb-26-metabolic-digital-twin/internal/services/cgm_period_report.go` | P7-E — add PROVISIONAL header comment |
| Modify | `kb-26-metabolic-digital-twin/internal/services/cgm_daily_batch.go` | P7-E — add PROVISIONAL header comment |
| Modify | `kb-26-metabolic-digital-twin/main.go` | P7-E — start CGM analytics consumer; P7-F — swap noop publisher for Kafka publisher |

### KB-23 changes (P7-A + P7-B + P7-C + P7-D)

| Action | File | Sub-project |
|---|---|---|
| Create | `kb-23-decision-cards/templates/renal/renal_contraindication.yaml` | P7-A — card template |
| Create | `kb-23-decision-cards/templates/renal/renal_dose_reduce.yaml` | P7-A — card template |
| Modify | `kb-23-decision-cards/internal/services/priority_signal_handler.go` | P7-A — replace log with card builder invocation in `handleRenalGate` |
| Create | `kb-23-decision-cards/templates/ckm/ckm_4c_mandatory_medication.yaml` | P7-B — card template |
| Modify | `kb-23-decision-cards/internal/services/priority_signal_handler.go` | P7-B — replace log with card builder invocation in `handleCKMTransition` |
| Create | `kb-23-decision-cards/internal/services/renal_anticipatory_orchestrator.go` | P7-C — real orchestrator calling FindApproachingThresholds |
| Create | `kb-23-decision-cards/internal/services/renal_anticipatory_orchestrator_test.go` | P7-C |
| Create | `kb-23-decision-cards/templates/renal/renal_threshold_approaching.yaml` | P7-C — card template |
| Create | `kb-23-decision-cards/templates/renal/stale_egfr.yaml` | P7-C — card template |
| Modify | `kb-23-decision-cards/internal/services/renal_anticipatory_batch.go` | P7-C — wire orchestrator + real repo |
| Create | `kb-23-decision-cards/internal/services/inertia_input_assembler.go` | P7-D — fetches KB-20 timeline + KB-26 target status |
| Create | `kb-23-decision-cards/internal/services/inertia_input_assembler_test.go` | P7-D |
| Create | `kb-23-decision-cards/internal/services/inertia_repository.go` | P7-D — inertia_verdict_history persistence |
| Create | `kb-23-decision-cards/migrations/XX_inertia_verdict_history.sql` | P7-D |
| Modify | `kb-23-decision-cards/internal/services/inertia_orchestrator.go` | P7-D — add stability dampening + persistence |
| Create | `kb-23-decision-cards/pkg/stability/engine.go` | P7-D — copy from KB-26 pkg/stability |
| Create | `kb-23-decision-cards/pkg/stability/policies.go` | P7-D — copy from KB-26 pkg/stability |
| Create | `kb-23-decision-cards/pkg/stability/engine_test.go` | P7-D — copy tests |
| Create | `kb-23-decision-cards/templates/inertia/inertia_detected.yaml` | P7-D — card template |
| Create | `kb-23-decision-cards/templates/inertia/dual_domain_inertia_detected.yaml` | P7-D — card template |
| Modify | `kb-23-decision-cards/internal/services/inertia_weekly_batch.go` | P7-D — wire assembler + orchestrator |
| Modify | `kb-23-decision-cards/internal/services/kb20_client.go` | P7-D — add FetchInterventionTimeline method |
| Create | `kb-23-decision-cards/internal/services/kb26_client.go` | P7-D — new client for target status (if not already exists) |

### Flink changes (P7-E)

| Action | File | Sub-project |
|---|---|---|
| Create | `flink-processing/src/main/java/com/cardiofit/flink/operators/Module3_CGMStreamJob.java` | P7-E — streaming job composing CGMReadingBuffer + Module3_CGMAnalytics into a running pipeline |
| Create | `flink-processing/src/test/java/com/cardiofit/flink/operators/Module3_CGMStreamJobTest.java` | P7-E — mini-cluster test |
| Modify | `runtime-layer/config/kafka-topics.yaml` | P7-E — add `clinical.cgm-analytics.v1` topic |
| Modify | `flink-processing/pom.xml` | P7-E — verify kafka-client dependencies are already present (they should be, used by HpiCalibrationStreamJob) |

### Metrics (shared collector updates, per Decision 9)

| Action | File | Sub-project |
|---|---|---|
| Modify | `kb-23-decision-cards/internal/metrics/collector.go` | P7-A + P7-B + P7-C + P7-D — add new counters |
| Modify | `kb-26-metabolic-digital-twin/internal/metrics/collector.go` | P7-E — add CGM consumer metrics |

**Total: ~26 create, ~9 modify across 4 services + 1 Flink module. ~60 new tests estimated.**

---

# Sub-project P7-A: Reactive Renal Dose Gating Activation

**Priority:** Patient safety — fastest win. Card persistence + metrics + upstream verification.
**Effort upper bound:** ~0.5 day of KB wiring. Expected actual: ~1-2 hours (deflate 3-5× per retrospective).
**Depends on:** Phase 6 P6-2 (reactive renal consumer + router dispatch already shipped).

## Task P7-A.1: Sub-recon on `deriveEGFR` → outbox → Kafka path

**Files:** read-only investigation.

- [ ] **Step 1:** Confirm that `lab_service.go:deriveEGFR` actually fires for FHIR-synced creatinine labs, not just directly-inserted ones. Trace: `syncObservations` → `createLabEntry` → `deriveEGFR`. Verify the Phase 6 P6-2 `eventBus.PublishTx(EventLabResult, LabType: "EGFR", ...)` call is reached on the FHIR-sync code path.
- [ ] **Step 2:** Check the outbox relay actually routes `EventLabResult` with `LabType="EGFR"` through `mapLabResult` → `signals.SignalEGFRLab` → `clinical.priority-events.v1`. The mapping was added in P6-2 but no end-to-end trace was verified.
- [ ] **Step 3:** If either step fails, surface the gap before writing card templates. The card work is wasted if the event never fires.

## Task P7-A.2: Build RENAL_CONTRAINDICATION + RENAL_DOSE_REDUCE YAML templates

**Files:**
- Create: `kb-23-decision-cards/templates/renal/renal_contraindication.yaml`
- Create: `kb-23-decision-cards/templates/renal/renal_dose_reduce.yaml`

- [ ] **Step 1:** Read an existing renal template reference — the Phase 4 masked HTN templates in `templates/bp_context/` are the closest shape. Copy the structure (`template_id`, `node_id`, `differential_id`, `version`, `clinical_reviewer`, `mcu_gate_default`, `card_source`, `card_type`, `trigger_event`, `trigger_condition`, `sla_hours`, `priority`, `confidence_tier`, `confidence_thresholds`, `recommendations`, `fragments`).
- [ ] **Step 2:** Build `renal_contraindication.yaml`:
  - `card_type: "RENAL_CONTRAINDICATION"`
  - `priority: "IMMEDIATE"`
  - `confidence_tier: "GUIDELINE_CONCORDANT"`
  - `mcu_gate_default: "BLOCK"`
  - `trigger_event: "EGFR_LAB"`
  - `trigger_condition: "renal_dose_gate.verdict == CONTRAINDICATED"`
  - 2 recommendations: immediate discontinuation + clinical review within 24h
  - Clinician fragment with templated drug class + eGFR value + threshold
  - Patient fragment with lay language
- [ ] **Step 3:** Build `renal_dose_reduce.yaml`:
  - `card_type: "RENAL_DOSE_REDUCE"`
  - `priority: "URGENT"`
  - `mcu_gate_default: "MODIFY"` (requires `dose_adjustment_notes`)
  - `trigger_condition: "renal_dose_gate.verdict == DOSE_REDUCE"`
  - 2 recommendations: dose reduction per formulary + monitoring frequency adjustment
- [ ] **Step 4:** Hot-reload templates locally via `/internal/templates/reload` and confirm the loader picks them up.

## Task P7-A.3: Wire card builder into `handleRenalGate`

**Files:**
- Modify: `kb-23-decision-cards/internal/services/priority_signal_handler.go`
- Modify: `kb-23-decision-cards/internal/metrics/collector.go`

- [ ] **Step 1:** Write failing test. Extend `priority_signal_handler_renal_test.go` with a case that provides a non-nil `db` + non-nil `kb20Client` + non-nil `renalDoseGate` and asserts that a contraindicated result produces a `DecisionCard` row with `CardType="RENAL_CONTRAINDICATION"`, `MCUGate="BLOCK"`, `Status=Active`, and `ClinicianSummary` containing the drug class + eGFR value.
- [ ] **Step 2:** Add `IncRenalContraindicationDetected(drugClass string)` + `IncRenalDoseReduceDetected(drugClass string)` + `ObserveRenalReactiveLatency(seconds float64)` to the metrics collector.
- [ ] **Step 3:** In `handleRenalGate`, replace the `h.log.Info("renal gate: gating gaps detected", ...)` block with:
  1. For each contraindicated medication: load the `RENAL_CONTRAINDICATION` template via `templateLoader.Get(...)`, build a `DecisionCard` via the existing `CardBuilder.Build(...)` pattern, persist via `db.Create`, increment the counter.
  2. For each dose-reduce medication: same flow with `RENAL_DOSE_REDUCE` template.
  3. Observe reactive latency from event timestamp (`env.Timestamp`) to now.
- [ ] **Step 4:** Run the new test. Run all renal handler tests. Verify no regressions.

## Task P7-A.4: Commit

- [ ] **Step 1:** `go test ./... ` in KB-23.
- [ ] **Step 2:** Commit: `feat(kb23): activate reactive renal dose gating with card persistence + metrics (Phase 7 P7-A)`.
- [ ] **Step 3:** Push.

## P7-A Verification Questions

1. Does `deriveEGFR` actually fire the `EGFR_LAB` event on the FHIR-sync code path? (yes / no / evidence: tracer log or integration test)
2. Does the outbox relay publish the event to `clinical.priority-events.v1`? (yes / no / evidence)
3. Does `handleRenalGate` persist a `RENAL_CONTRAINDICATION` DecisionCard with `MCUGate="BLOCK"` for an eGFR=25 / metformin patient? (yes / no / test)
4. Does `renal_reactive_latency_seconds` histogram populate on each event? (yes / no / evidence)
5. Full KB-23 + KB-20 + KB-26 test suites green? (yes / no)

---

# Sub-project P7-B: CKM 4c Activation + Resolve Trigger Gap

**Priority:** Closes the unresolved upstream gap flagged in the Phase 6 retrospective. `CKMTransitionPublisher.PublishStageTransition` exists but has no caller in production.
**Effort upper bound:** ~1 day. Expected actual: ~2-3 hours.
**Depends on:** Phase 6 P6-6 (KB-23 consumer dispatch already shipped).

## Task P7-B.1: Build `CKMRecomputationService` in KB-20

**Files:**
- Create: `kb-20-patient-profile/internal/services/ckm_recomputation_service.go`
- Create: `kb-20-patient-profile/internal/services/ckm_recomputation_service_test.go`

- [ ] **Step 1:** Write a failing test. Seed a `PatientProfile` at `CKMStageV2 = "3b"`. Call `service.RecomputeAndPublish(patientID, "obs-12345")`. Assert that `ClassifyCKMStage` runs, the stage changes (or not, per seeded data), and if changed, `CKMTransitionPublisher.PublishStageTransition` is called with the right arguments.
- [ ] **Step 2:** Implement the service:
  1. Fetch `PatientProfile` from DB
  2. Assemble `CKMClassifierInput` from the profile's existing fields (Age, Sex, BMI, HasDiabetes, HasHTN, HbA1c, EGFR, ASCVDEvents, HasHeartFailure, LVEF, NYHAClass, etc.) — **use only fields already present on PatientProfile**; any missing field is nil and `ClassifyCKMStage` handles nil gracefully
  3. Call `ClassifyCKMStage(input)` → `CKMStageResult`
  4. Call `publisher.PublishStageTransition(patientID, result.Stage, result.StagingRationale, result.Metadata.HFClassification, "OBSERVATION", observationID)`
  5. Return the transition result + any error
- [ ] **Step 3:** Test the no-change case: unchanged stage returns `(false, nil)` without calling the publisher.
- [ ] **Step 4:** Commit.

## Task P7-B.2: Wire `CKMRecomputationService` into the FHIR sync worker

**Files:**
- Modify: `kb-20-patient-profile/internal/fhir/fhir_sync_worker.go`

- [ ] **Step 1:** Find the `syncObservations` function + its persist path (`createLabEntry` or similar).
- [ ] **Step 2:** After a new `LabEntry` is persisted with `LabType` in `{"LVEF", "NT_PRO_BNP", "CAC_SCORE"}`, invoke `ckmRecomputationService.RecomputeAndPublish(patientID, labEntry.FHIRObservationID)`. Log + swallow errors — recomputation failure does not block lab ingestion.
- [ ] **Step 3:** Wire the service into the worker constructor (mirror the P5-2 `stampMedicationChange` wiring pattern).
- [ ] **Step 4:** Commit.

## Task P7-B.3: Build CKM_4C_MANDATORY_MEDICATION YAML template

**Files:**
- Create: `kb-23-decision-cards/templates/ckm/ckm_4c_mandatory_medication.yaml`

- [ ] **Step 1:** Template shape:
  - `card_type: "CKM_4C_MANDATORY_MEDICATION"`
  - `priority: "IMMEDIATE"`
  - `mcu_gate_default: "OBSERVE"` (detection is guideline-concordant, action requires clinician)
  - `confidence_tier: "GUIDELINE_CONCORDANT"`
  - `trigger_event: "CKM_STAGE_TRANSITION"`
  - `trigger_condition: "to_stage == '4c' && missing_mandatory_class != ''"`
  - 3 recommendations: initiate missing drug class, cardiology referral, monitoring at 2 weeks
  - Fragments: clinician template with missing drug class + HF type + source trial; patient template in lay language
- [ ] **Step 2:** Hot-reload + verify template loads.

## Task P7-B.4: Wire card builder into `handleCKMTransition`

**Files:**
- Modify: `kb-23-decision-cards/internal/services/priority_signal_handler.go`
- Modify: `kb-23-decision-cards/internal/metrics/collector.go`

- [ ] **Step 1:** Write failing test in `priority_signal_handler_ckm_test.go` that provides non-nil deps and asserts that a `to_stage="4c"` transition with gaps produces one `DecisionCard` row per missing drug class.
- [ ] **Step 2:** Add `IncCKMStageTransition(from, to string)` + `IncCKM4CMandatoryMedAlerts(drugClass string)` + `IncCKMRecomputation(trigger string)` to the metrics collector.
- [ ] **Step 3:** In `handleCKMTransition`, after the `CheckMandatory` call, if gaps exist: load `CKM_4C_MANDATORY_MEDICATION` template via loader, build + persist a card per gap, increment the counter. Also increment `ckm_stage_transitions_total{from,to}` for every event (not just 4c) so the dashboard sees all transitions.
- [ ] **Step 4:** Run tests, commit.

## P7-B Verification Questions

1. Does a new LVEF observation on a patient with EF=30% trigger `CKMRecomputationService.RecomputeAndPublish`? (yes / no / test)
2. Does `PublishStageTransition` write a `CKM_STAGE_TRANSITION` event to the outbox when the stage changes? (yes / no / test)
3. Does the outbox relay publish the event to `clinical.priority-events.v1`? (yes / no / evidence — already wired in P6-6)
4. Does the KB-23 consumer receive the event and invoke `MandatoryMedChecker`? (yes / no — already wired in P6-6)
5. Does a 4c transition with a missing STATIN produce a `CKM_4C_MANDATORY_MEDICATION` card with `Priority=IMMEDIATE`? (yes / no / test)
6. Does an unchanged stage produce NO event and NO card? (yes / no / test)
7. All three Prometheus counters increment correctly? (yes / no / evidence)

---

# Sub-project P7-C: Renal Anticipatory Monthly Activation

**Priority:** Proactive clinical planning. Fills the monthly runtime with real work.
**Effort upper bound:** ~1 day. Expected actual: ~2-4 hours.
**Depends on:** Phase 6 P6-5 (KB-23 BatchScheduler + `RenalAnticipatoryBatch` heartbeat shell already shipped).

## Task P7-C.1: Build `RenalAnticipatoryOrchestrator`

**Files:**
- Create: `kb-23-decision-cards/internal/services/renal_anticipatory_orchestrator.go`
- Create: `kb-23-decision-cards/internal/services/renal_anticipatory_orchestrator_test.go`

- [ ] **Step 1:** Write failing test. Stub a patient with eGFR=45, slope=-6 mL/min/1.73m²/year, on metformin. Assert that `orchestrator.EvaluatePatient(patientID)` calls `FindApproachingThresholds` + `DetectStaleEGFR`, returns a result with one `RENAL_THRESHOLD_APPROACHING` alert for metformin (projected crossing at 30 in ~30 months), and publishes events via KB-19.
- [ ] **Step 2:** Implement:
  1. Fetch patient's eGFR + slope from KB-20 (or KB-26 — decide during implementation; likely KB-20 since eGFR trajectories live there)
  2. Fetch patient's active medications from KB-20 via existing `KB20Client`
  3. Construct `[]ActiveMedication` from the drug class list
  4. Call `FindApproachingThresholds(formulary, egfr, slope, meds)`
  5. Call `DetectStaleEGFR(renalStatus, cfg, onRenalSensitiveMed)`
  6. Return a combined result
- [ ] **Step 3:** Commit.

## Task P7-C.2: Build `RENAL_THRESHOLD_APPROACHING` + `STALE_EGFR` YAML templates

**Files:**
- Create: `kb-23-decision-cards/templates/renal/renal_threshold_approaching.yaml`
- Create: `kb-23-decision-cards/templates/renal/stale_egfr.yaml`

- [ ] **Step 1:** `renal_threshold_approaching.yaml`:
  - `card_type: "RENAL_THRESHOLD_APPROACHING"`
  - `priority: "ROUTINE"` (anticipatory, not acute)
  - `mcu_gate_default: "ADVISORY"` (no action required, planning only)
  - `trigger_event: "RENAL_ANTICIPATORY_ALERT"`
  - 2 recommendations: plan substitute medication + schedule repeat eGFR
  - Fragments with templated drug class + months-to-threshold + substitute class
- [ ] **Step 2:** `stale_egfr.yaml`:
  - `card_type: "STALE_EGFR"`
  - `priority: "ROUTINE"`
  - `mcu_gate_default: "OBSERVE"`
  - `trigger_event: "STALE_EGFR_DETECTED"`
  - 1 recommendation: order eGFR labs ASAP

## Task P7-C.3: Build `RenalActivePatientLister` backed by KB-20

**Files:**
- Modify: `kb-23-decision-cards/internal/services/renal_anticipatory_batch.go`
- Modify: `kb-23-decision-cards/internal/services/kb20_client.go`

- [ ] **Step 1:** Add `FetchRenalActivePatientIDs(ctx) ([]string, error)` to `KB20Client`. Calls a new KB-20 endpoint `GET /api/v1/patients/renal-active` that returns patients with at least one renal-sensitive medication (ACEi/ARB/SGLT2/metformin/etc). **NOTE**: this requires a small new KB-20 endpoint; add it here as a sub-task if not already present.
- [ ] **Step 2:** Update `NewRenalAnticipatoryBatch(repo, orchestrator, log)` signature to take the orchestrator dependency.
- [ ] **Step 3:** In `Run`, iterate active patients and call `orchestrator.EvaluatePatient(patientID)` per patient, bounded concurrency default 4. Per-patient errors logged + isolated.
- [ ] **Step 4:** For each returned alert, load the appropriate template + build + persist a card.

## Task P7-C.4: Add metrics + wire in main.go + commit

**Files:**
- Modify: `kb-23-decision-cards/internal/metrics/collector.go`
- Modify: `kb-23-decision-cards/main.go`

- [ ] **Step 1:** Add metrics per Decision 9.
- [ ] **Step 2:** Replace `NewRenalAnticipatoryBatch(nil, logger)` in main.go with the wired version.
- [ ] **Step 3:** Full test sweep + commit.

## P7-C Verification Questions

1. Does `KB20Client.FetchRenalActivePatientIDs` return patients on renal-sensitive medications? (yes / no / evidence)
2. Does `RenalAnticipatoryOrchestrator.EvaluatePatient` call `FindApproachingThresholds` + `DetectStaleEGFR`? (yes / no / test)
3. Does a patient with eGFR=45 + slope=-6/year produce a `RENAL_THRESHOLD_APPROACHING` card for metformin? (yes / no / test)
4. Does a patient with 9-month-old eGFR produce a `STALE_EGFR` card? (yes / no / test)
5. Do all three Prometheus counters increment correctly? (yes / no / evidence)
6. Does the batch still fire only on the 1st of the month at 04:00 UTC? (yes / no — shouldn't regress from P6-5)

---

# Sub-project P7-D: Inertia Weekly Batch Activation

**Priority:** Highest clinical value KB-service activation. Inertia detection runs autonomously for the first time.
**Effort upper bound:** ~2 days. Expected actual: ~4-6 hours.
**Depends on:** Phase 6 P6-1 (KB-23 InertiaOrchestrator + InertiaWeeklyBatch heartbeat already shipped). Phase 6 P6-5 (KB-23 BatchScheduler).

## Task P7-D.1: Build KB-20 intervention timeline service + HTTP endpoint

**Files:**
- Create: `kb-20-patient-profile/internal/services/intervention_timeline_service.go`
- Create: `kb-20-patient-profile/internal/services/intervention_timeline_service_test.go`
- Create: `kb-20-patient-profile/internal/api/intervention_handlers.go`
- Modify: `kb-20-patient-profile/internal/api/routes.go`

- [ ] **Step 1:** Service `BuildTimeline(patientID) (*InterventionTimelineResult, error)`:
  1. Query `medication_states WHERE patient_id = ? AND start_date > now() - 90 days`
  2. Group by drug class → map drug class to clinical domain via the existing `MapDrugClassToDomain`
  3. Keep only the latest action per domain
  4. Return `{GLYCAEMIC: LatestDomainAction, HEMODYNAMIC: ..., RENAL: ..., LIPID: ...}`
- [ ] **Step 2:** Write test with 5 medication_state rows across 3 domains, assert the latest per domain is returned.
- [ ] **Step 3:** HTTP handler `GET /api/v1/patient/:id/intervention-timeline` returning the service output as JSON.
- [ ] **Step 4:** Commit.

## Task P7-D.2: Build KB-26 target-status HTTP endpoint

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/api/target_status_handlers.go`
- Modify: `kb-26-metabolic-digital-twin/internal/api/routes.go`

- [ ] **Step 1:** `GET /api/v1/patient/:id/target-status` handler:
  1. Fetch patient profile from repo (or accept domain state via request body — decide during impl)
  2. Call `ComputeGlycaemicTargetStatus` + `ComputeHemodynamicTargetStatus` + `ComputeRenalTargetStatus` inline
  3. Return `{glycaemic: DomainTargetStatusResult, hemodynamic: ..., renal: ...}`
- [ ] **Step 2:** Write a handler-level test via `httptest`.
- [ ] **Step 3:** Commit.

## Task P7-D.3: Copy pkg/stability into KB-23

**Files:**
- Create: `kb-23-decision-cards/pkg/stability/engine.go` (copy from KB-26)
- Create: `kb-23-decision-cards/pkg/stability/policies.go` (copy from KB-26)
- Create: `kb-23-decision-cards/pkg/stability/engine_test.go` (copy from KB-26)

Per Decision 8, this is intentional duplication until a third consumer appears. Header comment documents the duplication and points at the canonical KB-26 version.

- [ ] **Step 1:** Copy the three files verbatim from `kb-26-metabolic-digital-twin/pkg/stability/`, update the package path comments.
- [ ] **Step 2:** Run the copied tests — they should pass verbatim.
- [ ] **Step 3:** Commit.

## Task P7-D.4: Build `InertiaInputAssembler`

**Files:**
- Create: `kb-23-decision-cards/internal/services/inertia_input_assembler.go`
- Create: `kb-23-decision-cards/internal/services/inertia_input_assembler_test.go`
- Modify: `kb-23-decision-cards/internal/services/kb20_client.go`
- Create: `kb-23-decision-cards/internal/services/kb26_client.go`

- [ ] **Step 1:** Add `FetchInterventionTimeline(ctx, patientID)` to `KB20Client`.
- [ ] **Step 2:** Create new `KB26Client` with `FetchTargetStatus(ctx, patientID)` method.
- [ ] **Step 3:** Write failing assembler test.
- [ ] **Step 4:** Implement `InertiaInputAssembler.AssembleInertiaInput(ctx, patientID) (InertiaDetectorInput, error)`:
  1. Fetch target status from KB-26
  2. Fetch intervention timeline from KB-20
  3. Fetch active medications from KB-20 (existing `FetchSummaryContext`)
  4. Build `DomainInertiaInput` for each domain where target is unmet + last intervention is > `gracePeriodDays` ago
  5. Populate `PostEvent`, `RenalProgression`, `Ceiling` sub-inputs if the upstream data supports it (nil otherwise — detector handles nil gracefully)
  6. Return the assembled input
- [ ] **Step 5:** Test with synthetic target-status + timeline fixtures.

## Task P7-D.5: Build inertia_verdict_history repository + migration

**Files:**
- Create: `kb-23-decision-cards/migrations/XX_inertia_verdict_history.sql`
- Create: `kb-23-decision-cards/internal/services/inertia_repository.go`

- [ ] **Step 1:** Migration:
  ```sql
  CREATE TABLE inertia_verdict_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    week_start_date DATE NOT NULL,
    verdicts_jsonb JSONB NOT NULL,
    dual_domain_detected BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(patient_id, week_start_date)
  );
  CREATE INDEX idx_inertia_patient ON inertia_verdict_history(patient_id, week_start_date DESC);
  ```
- [ ] **Step 2:** Repository with `SaveVerdict` (upsert) + `FetchLatest(patientID)`.
- [ ] **Step 3:** Tests.

## Task P7-D.6: Add stability dampening to `InertiaOrchestrator`

**Files:**
- Modify: `kb-23-decision-cards/internal/services/inertia_orchestrator.go`

- [ ] **Step 1:** Update `InertiaOrchestrator.Evaluate` to:
  1. Run `DetectInertia(input)` as today
  2. Fetch previous week's verdict from repository
  3. Apply stability engine: if current verdict differs from previous AND the raw target status changed, accept; otherwise dampen to previous verdict
  4. Persist the (possibly-dampened) verdict via `SaveVerdict`
  5. Return the final report
- [ ] **Step 2:** Test the flip-flop case: raw verdict alternates between weeks but target status unchanged → dampening holds the previous verdict.

## Task P7-D.7: Build `INERTIA_DETECTED` + `DUAL_DOMAIN_INERTIA_DETECTED` templates + wire card builder

**Files:**
- Create: `kb-23-decision-cards/templates/inertia/inertia_detected.yaml`
- Create: `kb-23-decision-cards/templates/inertia/dual_domain_inertia_detected.yaml`
- Modify: `kb-23-decision-cards/internal/services/inertia_orchestrator.go`

- [ ] **Step 1:** Templates follow the same shape as renal templates (P7-A/C). Severity-mapped urgency per the existing `mapSeverityToUrgency` function.
- [ ] **Step 2:** In `InertiaOrchestrator.Evaluate`, after persistence, build + persist one `DecisionCard` per detected verdict via the existing card builder. Dual-domain produces one extra card with `CardType="DUAL_DOMAIN_INERTIA_DETECTED"`.

## Task P7-D.8: Wire everything into `InertiaWeeklyBatch` + main.go

**Files:**
- Modify: `kb-23-decision-cards/internal/services/inertia_weekly_batch.go`
- Modify: `kb-23-decision-cards/main.go`

- [ ] **Step 1:** Replace the nil repo + nil assembler + nil orchestrator in main.go with real wiring:
  ```go
  kb20c := services.NewKB20Client(...)  // already exists
  kb26c := services.NewKB26Client(...)  // new from Task P7-D.4
  assembler := services.NewInertiaInputAssembler(kb20c, kb26c, logger)
  inertiaRepo := services.NewInertiaRepository(db)
  orchestrator := services.NewInertiaOrchestrator(inertiaRepo, logger)
  // Build a real RenalActivePatientLister wrapper that calls KB-20's
  // generic active-patient endpoint (reuse the one P7-C adds, or add a
  // sibling method)
  inertiaJob := services.NewInertiaWeeklyBatch(activeLister, assembler, orchestrator, logger)
  ```
- [ ] **Step 2:** Full KB-23 + KB-20 + KB-26 test sweep.
- [ ] **Step 3:** Commit.

## P7-D Verification Questions

1. Does `KB20Client.FetchInterventionTimeline` return a non-nil timeline for a patient with recent medication changes? (yes / no / test)
2. Does `KB26Client.FetchTargetStatus` return target status per domain? (yes / no / test)
3. Does `InertiaInputAssembler.AssembleInertiaInput` produce a valid `InertiaDetectorInput` end-to-end? (yes / no / test)
4. Does the stability check prevent a one-week flip-flop from producing an `INERTIA_DETECTED` card? (yes / no / test)
5. Does `inertia_verdict_history` contain one row per patient per week after a batch run? (yes / no / evidence)
6. Does the batch fan out across active patients at bounded concurrency? (yes / no / evidence)
7. Do the metrics increment correctly? (yes / no / evidence)
8. Are KB-20 + KB-23 + KB-26 full test suites green? (yes / no)

---

# Sub-project P7-E: Module 3 CGM Streaming Pipeline

**Priority:** The only Phase 7 sub-project that produces a new running Flink job. Dedicated session(s), different toolset.
**Effort estimate:** ~3 days. **NOT deflated from upper bound** — Flink work has irreducible complexity.
**Depends on:** Existing Flink infrastructure (13 modules, `HpiCalibrationStreamJob.java` template), existing `CGMReadingBuffer` + `Module3_CGMAnalytics` + `CGMAnalyticsEvent` components.

## Task P7-E.1: Recon the HpiCalibrationStreamJob template

**Files:** read-only investigation.

- [ ] **Step 1:** Read `flink-processing/src/main/java/com/cardiofit/flink/operators/HpiCalibrationStreamJob.java` end to end.
- [ ] **Step 2:** Note the shape: `StreamExecutionEnvironment` setup, `KafkaSource.builder()...build()`, `WatermarkStrategy.forBoundedOutOfOrderness(...)`, `.keyBy(...)`, `.window(SlidingEventTimeWindows...)`, `.process(new KeyedProcessFunction(...))`, `.addSink(KafkaSink...)`. This is the structural template for `Module3_CGMStreamJob.java`.
- [ ] **Step 3:** Note the Kafka source configuration (brokers from env, topic from config, consumer group ID convention, offset init strategy).
- [ ] **Step 4:** Note the sink configuration (topic, serialization schema, delivery semantics).
- [ ] **Step 5:** Document any idiosyncrasies (checkpoint config, state backend, parallelism hints).

## Task P7-E.2: Add `clinical.cgm-analytics.v1` topic to config

**Files:**
- Modify: `runtime-layer/config/kafka-topics.yaml`

- [ ] **Step 1:** Add the topic entry alongside the existing `cgm_raw` definition:
  ```yaml
  cgm_analytics: "clinical.cgm-analytics.v1"
  ```
- [ ] **Step 2:** Add the topic to the topic-creation automation if one exists (`create-topics.sh` or equivalent — search for the ops script).

## Task P7-E.3: Build `Module3_CGMStreamJob.java`

**Files:**
- Create: `flink-processing/src/main/java/com/cardiofit/flink/operators/Module3_CGMStreamJob.java`

- [ ] **Step 1:** Copy the structural shape from `HpiCalibrationStreamJob.java`. Substitutions:
  - Class name: `Module3_CGMStreamJob`
  - Source topic: `ingestion.cgm-raw`
  - Source deserializer: parses CGM reading JSON/Avro into a Java record with `patient_id`, `timestamp_ms`, `glucose_mg_dl`
  - Watermark: `WatermarkStrategy.forBoundedOutOfOrderness(Duration.ofMinutes(5))` (CGM readings can arrive slightly out of order)
  - Key selector: `.keyBy(r -> r.patientId)`
  - Window: `.window(SlidingEventTimeWindows.of(Time.days(14), Time.days(1)))` — 14-day window sliding every 24 hours
  - Process function: `new CGMWindowProcessor()` — inside its `process(key, context, elements, out)`:
    1. Collect all `glucoseMgDl` values from `elements` into a `List<Double>`
    2. Compute `windowDays = 14`
    3. Call `Module3_CGMAnalytics.computeMetrics(readings, windowDays)` → `CGMAnalyticsEvent`
    4. Set `event.patientId = key`, `event.computedAt = context.currentProcessingTime()`, etc.
    5. `out.collect(event)`
  - Sink: `KafkaSink` writing to `clinical.cgm-analytics.v1`, serializer `CGMAnalyticsEventSerializer` (Jackson ObjectMapper → bytes)
- [ ] **Step 2:** Main method (`public static void main(String[] args)`) that reads brokers + topic names from env vars, configures the env, builds the pipeline, calls `env.execute("Module3 CGM Analytics")`.

## Task P7-E.4: Mini-cluster integration test

**Files:**
- Create: `flink-processing/src/test/java/com/cardiofit/flink/operators/Module3_CGMStreamJobTest.java`

- [ ] **Step 1:** Use Flink's `MiniClusterWithClientResource` (or the project's existing test pattern — check `HpiCalibrationStreamJobTest` if it exists for the reference).
- [ ] **Step 2:** Seed the mini-cluster with fake Kafka source containing 1344 readings at 140 mg/dL for patient "p1" across a 14-day window.
- [ ] **Step 3:** Run the job for one window emission.
- [ ] **Step 4:** Assert the sink received one `CGMAnalyticsEvent` with `patientId="p1"`, `tirPct~100`, `meanGlucose~140`, `griZone="A"`.

## Task P7-E.5: Build KB-26 CGM analytics consumer

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/cgm_analytics_consumer.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/cgm_analytics_consumer_test.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/cgm_period_report_repository.go`

- [ ] **Step 1:** Copy the shape of existing `signal_consumer.go` — Kafka reader on `clinical.cgm-analytics.v1`, group ID `kb26-cgm-analytics`, deserializes JSON → Go struct mirroring `CGMAnalyticsEvent`, dispatches to a handler.
- [ ] **Step 2:** Handler persists each event into `cgm_period_reports` table via the new repository.
- [ ] **Step 3:** Repository `SavePeriodReport(report *models.CGMPeriodReport) error` using GORM `Create`.
- [ ] **Step 4:** Test with a stub consumer receiving a synthetic `CGMAnalyticsEvent` — assert the Postgres row is written.
- [ ] **Step 5:** Add metrics per Decision 9.

## Task P7-E.6: Wire consumer into KB-26 main.go + mark Go duplicates provisional

**Files:**
- Modify: `kb-26-metabolic-digital-twin/main.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/cgm_period_report.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/cgm_daily_batch.go`

- [ ] **Step 1:** Start the CGM analytics consumer in main.go alongside the existing `SignalConsumer`.
- [ ] **Step 2:** Add the PROVISIONAL header comment (per Decision 2) to both Go files.
- [ ] **Step 3:** Full KB-26 test sweep.

## P7-E Verification Questions

1. Does `Module3_CGMStreamJob.java` compile cleanly under the existing Maven build? (yes / no / evidence)
2. Does the mini-cluster integration test pass? (yes / no / test)
3. Does the job produce one `CGMAnalyticsEvent` per 14-day window slide for a seeded patient? (yes / no / test)
4. Does KB-26's new consumer deserialize and persist the event into `cgm_period_reports`? (yes / no / test)
5. Does a `TIR=100` synthetic fixture produce a persisted row with `tir_pct=100`, `gri_zone="A"`? (yes / no / test)
6. Are the Go `cgm_period_report.go` + `cgm_daily_batch.go` files marked provisional with the canonical-source comment? (yes / no / grep)
7. Does the new `clinical.cgm-analytics.v1` topic appear in the Kafka topic config? (yes / no / grep)
8. Do the CGM consumer metrics increment on each event? (yes / no / evidence)

---

# Sub-project P7-F: Domain Trajectory Kafka Publisher Swap

**Priority:** Cleanup. Closes the Phase 6 P6-3 `TODO(kb26-kafka)` comment.
**Effort upper bound:** 15 minutes. Expected actual: 15 minutes.
**Depends on:** Phase 6 P6-3 (domain trajectory wiring complete, currently publishes to `NoopTrajectoryPublisher`).

## Task P7-F.1: Replace `NoopTrajectoryPublisher` with `KafkaTrajectoryPublisher`

**Files:**
- Modify: `kb-26-metabolic-digital-twin/main.go`
- Modify: `kb-26-metabolic-digital-twin/internal/api/server.go`

- [ ] **Step 1:** Find the existing `NoopTrajectoryPublisher{}` wiring (server.go:88 per P6-3 recon).
- [ ] **Step 2:** Replace with:
  ```go
  brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
  topic := "clinical.domain-trajectory.v1"  // or reuse an existing topic if one is designated
  trajectoryPublisher := services.NewKafkaTrajectoryPublisher(brokers, topic, logger)
  ```
  (Gracefully fall back to Noop if `KAFKA_BROKERS` is empty, so local dev without Kafka still works.)
- [ ] **Step 3:** Verify the topic is declared in `runtime-layer/config/kafka-topics.yaml` (add if missing).
- [ ] **Step 4:** Build + test + commit.

## P7-F Verification Questions

1. Does `TrajectoryEngine.Compute` now publish to the real Kafka topic in production? (yes / no / grep)
2. Does local dev without `KAFKA_BROKERS` still fall back to Noop without crashing? (yes / no / test)
3. Does the existing P6-3 metric `domain_trajectory_persist_total` still increment correctly? (yes / no — no regression)

---

## Execution Order Recommendation

Sub-projects can ship in any order except where explicitly noted:

**Session 1 (Go/KB services — expected ~2-4 hours total):**
1. **P7-F** (15 min) — warmup clean win
2. **P7-A** (1-2h) — card persistence pattern established
3. **P7-B** (2-3h) — closes the unresolved P6-6 gap
4. **P7-C** (2-4h) — second application of the card persistence pattern

**Session 2 (Go/KB services, continuing):**
5. **P7-D** (4-6h) — largest KB sub-project; benefits from having P7-A/B/C's card persistence pattern proven first

**Session 3+ (Java/Flink — dedicated session(s), different toolset):**
6. **P7-E** (~3 days) — Flink pipeline + KB-26 consumer

**Total: ~1.5-2 sessions for P7-A through P7-D + P7-F. P7-E is its own 1-3 session track.**

---

## What Phase 7 Delivers

After every sub-project ships:

- **Reactive renal contraindications produce real IMMEDIATE cards** when a new eGFR lands — metformin flagged the same hour for a patient whose eGFR just dropped below 30.
- **CKM 4c transitions are triggered by real clinical events** (new echo with reduced EF, new NT-proBNP, new CAC score) AND produce mandatory-med IMMEDIATE cards via the KB-23 consumer built in P6-6.
- **Monthly renal anticipatory alerts** fire for patients approaching thresholds, persisted as ROUTINE cards with planning recommendations.
- **Inertia detection runs weekly** for every active patient, with real target-status + intervention-timeline data, stability-dampened verdicts, and persisted cards.
- **Module 3 CGM analytics pipeline** produces CGMAnalyticsEvents on the new topic, persisted into the existing `cgm_period_reports` table, updating glucose domain scores.
- **Domain trajectory events** publish to real Kafka instead of the noop.

Every sub-project follows the established pattern: clinical signal → detection → classification → card generation → event publication → operational monitoring → batch/event reassessment. No feature is in heartbeat mode anymore.

---

## What Phase 7 Does NOT Deliver

Explicit deferrals:

- **FHIR Condition sync** (MI, HF, stroke Conditions triggering CKM recomputation). P7-B handles Observation-driven recomputation only. Condition-driven is a Phase 8 follow-up.
- **Full AGP percentile overlays** for CGM — still deferred per Phase 6 Locked Decision 5. Phase 8.
- **Gap 9 (FHIR outbound)** — regulatory compliance blocker for Australia MHR / India ABDM. Separate workstream.
- **Gap 10 (Unified explainability chain)** — unified `rationale` field on every card type. Phase 8.
- **Gap 11 (Clinical audit event sourcing)** — append-only event log with hash-chaining. Phase 8.
- **Gap 12 (Circuit breaking)** — backpressure between services. Phase 8 infrastructure hardening.
- **Gap 13 (Formulary accessibility filtering)** — market-specific drug availability. Phase 8.
- **`pkg/stability` extraction to shared-infrastructure** — P7-D copies it into KB-23 as intentional duplication. When a third consumer appears, promote to shared (Phase 8+).

---

## Plan Summary

| Sub-project | Toolset | Upper bound | Expected actual | Files | New tests |
|---|---|---|---|---|---|
| P7-A Reactive Renal Activation | Go/KB | 0.5d | 1-2h | ~4 | ~4 |
| P7-B CKM 4c + trigger gap | Go/KB | 1d | 2-3h | ~6 | ~8 |
| P7-C Renal Anticipatory Activation | Go/KB | 1d | 2-4h | ~7 | ~8 |
| P7-D Inertia Weekly Activation | Go/KB | 2d | 4-6h | ~12 | ~15 |
| P7-E Module 3 CGM Streaming | Java/Flink + Go/KB | 3d | **3d (no deflation)** | ~6 | ~8 |
| P7-F Trajectory Publisher Swap | Go/KB | 15m | 15m | ~2 | 0 |
| **Total** | — | **~8d upper** | **~1.5 sessions Go + 1-3 sessions Flink** | **~37** | **~43** |

Phase 7 brings the platform from "every Phase 6 feature has a proven abstraction but runs in heartbeat mode" to "every Phase 6 feature is active in production with real cards, real metrics, and real data flows." The CGM streaming sub-project is the one that breaks the pattern and needs its own dedicated session tempo. The other five sub-projects should ship cleanly in one or two Go-focused sessions, following the same verification-question contract that shipped Phase 6 in a single session at 20× the estimated velocity.
