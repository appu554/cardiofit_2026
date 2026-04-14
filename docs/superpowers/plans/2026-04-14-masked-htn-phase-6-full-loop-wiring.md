# Masked HTN / Clinical Intelligence Platform — Phase 6: Full-Loop Wiring

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan sub-project-by-sub-project. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Take the clinical intelligence detectors that already exist in the KB services (inertia, renal dose gating, domain decomposition, CGM analytics, CKM substaging) and wire them into autonomous operational loops. After Phase 6, no clinical feature stops at "the detector exists" or "a clinician can hit an HTTP endpoint" — every feature has autonomous triggering (batch or event-driven), event publication to KB-19, stability dampening where appropriate, and Prometheus metrics.

**Architecture:** Phase 6 is **not** a single feature. It is **6 independent sub-projects**, each wiring one detector into an operational loop. Each sub-project has its own cadence (weekly batch, event-driven, monthly batch, daily batch), its own prerequisite state, and its own verification questions. Sub-projects can ship in any order except where explicitly noted. The Phase 5 `BatchJob.ShouldRun` abstraction hosts the batch-cadence consumers; existing Kafka topics (`clinical.state-changes.v1`, `clinical.priority-events.v1`) host the event-driven consumers.

**Pre-requisite:** Phase 5 shipped. `feature/v4-clinical-gaps` is at HEAD `ed43b6ef` with all P5-1 through P5-5 work committed and pushed to origin. KB-20, KB-23, KB-26 all build and test green.

**Tech Stack:** Go 1.25 (Gin, GORM, Zap, Prometheus, segmentio/kafka-go), PostgreSQL 15, existing `clinical.state-changes.v1` and `clinical.priority-events.v1` Kafka topics.

---

## Status Snapshot — 2026-04-14

- [ ] **P6-1 — Therapeutic Inertia Weekly Batch:** Not started. **Requires KB-20 intervention timeline service build-out** (~1 day) before the KB-26 weekly batch orchestrator can be built (~2 days). Total ~3 days. Priority: highest clinical impact.
- [ ] **P6-2 — Reactive Renal Dose Gating (Kafka consumer):** Not started. `RenalDoseGate.EvaluatePatient` exists. Needs a KB-23 consumer on the existing `clinical.state-changes.v1` topic that filters for eGFR lab events. Total ~2 days.
- [ ] **P6-3 — Domain Decomposition Wiring:** Partially wired already — `TrajectoryEngine.Compute` + `DomainTrajectoryComputedEvent` publisher already exist. Remaining work: ensure it's invoked from the MHRI recomputation path, add stability check for concordant deterioration detection, confirm Prometheus metrics. Total ~1 day.
- [ ] **P6-4 — CGM 14-Day Batch + Period Report Computation:** Not started. **Scope larger than originally estimated** — the CGM period report computation itself (raw readings → TIR/TBR/TAR/CV/GMI/GRI/AGP) does not yet exist. Only the post-computed glucose domain score exists. This sub-project ships the computation pipeline AND the batch trigger. Total ~5 days. Consider scope contraction: ship the trigger + TIR/CV computation only, defer full AGP to Phase 7.
- [ ] **P6-5 — Renal Anticipatory Monthly Batch:** Not started. `FindApproachingThresholds` + `DetectStaleEGFR` exist. Straightforward batch wrapping. Total ~1 day.
- [ ] **P6-6 — CKM Substaging Event-Driven Trigger:** Not started. `ClassifyCKMStage` exists in KB-20. `MandatoryMedChecker` exists in KB-23 (cross-service — see Decision 9). KB-20 publishes `CKM_STAGE_TRANSITION` to existing `clinical.priority-events.v1` topic; KB-23's existing `PrioritySignalConsumer` is extended to handle the event and trigger `MandatoryMedChecker` on 4c transitions. Total ~2 days (1.5d KB-20 handler + 0.5d KB-23 consumer extension).

**Total remaining effort: ~14 days** across 6 sub-projects. Realistic session pacing: 2 sub-projects per session, ~3-4 sessions.

---

## Locked Decisions

These are **not** open questions. They are fixed constraints derived from the Phase 5 recon and from the existing codebase shape discovered during Phase 6 planning. Future sessions implementing this plan should treat them as given.

### Decision 1: `InertiaWeeklyBatch.Run` stays in KB-26, not KB-23

The Phase 5 stub placed `InertiaWeeklyBatch` in KB-26. The Phase 6 inertia orchestrator — which must call `ComputeGlycaemicTargetStatus` + `ComputeHemodynamicTargetStatus` + `ComputeRenalTargetStatus` (all in [kb-26/internal/services/target_status.go](backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/target_status.go)) and fetch the patient's intervention timeline from KB-20 — benefits from being co-located with the target-status computations. Putting it in KB-26 means **no HTTP hop for the glycaemic/hemodynamic/renal target status computations**; only the KB-20 intervention timeline fetch is remote. This matches the pattern `BPContextOrchestrator` established in Phase 2.

**No new KB-23 `/api/v1/inertia/scan/:patientId` endpoint is needed.** The batch orchestrator calls `DetectInertia(input)` directly (`DetectInertia` is a pure function in `kb-23-decision-cards` but KB-26 can't import KB-23 internal packages). Two options: (a) move `DetectInertia` + `PatientInertiaReport` to a shared package under `shared-infrastructure/clinical-intelligence`, or (b) add a KB-23 HTTP endpoint and have KB-26 call it. **Option (a) is the correct long-term architecture** because the detector is pure and multiple services will want to call it. Sub-project P6-1 ships option (a).

### Decision 2: KB-20 intervention timeline ships as a new service + HTTP endpoint in P6-1, not as a separate prerequisite plan

The Phase 5 plan deferred "build the KB-20 intervention timeline service" as a Phase 6 dependency. Phase 6 P6-1 owns that build — it's a ~1-day addition to the sub-project, not a separate sub-project. The reason: the inertia batch is the only current consumer, so building the service without a consumer would be speculative work.

### Decision 3: Reactive renal gate consumes the existing `clinical.state-changes.v1` topic, no new topic is created

The Phase 5 recon found KB-26 already has a `SignalConsumer` at [kb-26/internal/services/signal_consumer.go](backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/signal_consumer.go) that consumes `clinical.state-changes.v1`. P6-2 adds a **new consumer in KB-23** (mirroring the `PrioritySignalConsumer` pattern at [kb-23/internal/services/priority_signal_consumer.go](backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/priority_signal_consumer.go)) that subscribes to the same topic and filters messages by `event_type == "LAB_RESULT"` + `lab_type == "eGFR"`. No new topic. No changes to KB-20's producer.

### Decision 4: Domain decomposition Phase 6 work is verification + missing-hook wiring, not implementation

`TrajectoryEngine.Compute` and `DomainTrajectoryComputedEvent` both exist in KB-26 as of the Module 13 trajectory work (seen during Phase 5 recon). P6-3 is **"verify the decomposition runs on every MHRI recomputation, add the stability check, add the Prometheus metrics if missing, and confirm the event is published."** If everything is already wired end-to-end, P6-3 is a verification task with no code changes. If any hook is missing, P6-3 adds that hook only. This is why P6-3's effort dropped from the original 1-2 days to ~1 day maximum.

### Decision 5: CGM period report computation ships as a minimal viable slice, not full AGP, in P6-4

The original Phase 6 plan assumed `GenerateCGMReport` exists. It doesn't. Building a full AGP (Ambulatory Glucose Profile) computation — 14-day overlay percentile plots, glycemic pattern classification, hypoglycemia risk zones — is a major clinical analytics workstream that's ~5 days by itself. P6-4 scopes this down: ship **TIR + TBR + TAR + CV + GMI computation + the 14-day trigger gate**, wire to the existing `ComputeGlucoseDomainScore` path, publish `CGM_REPORT_AVAILABLE` events. **Defer the full AGP percentile overlay to Phase 7.** This keeps P6-4 at ~3 days.

### Decision 6: CKM substaging event trigger lives inside KB-20, not as a downstream consumer

KB-20 already has clinical event processing — the FHIR sync worker at [kb-20/internal/fhir/fhir_sync_worker.go](backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/fhir/fhir_sync_worker.go) handles `MedicationRequest` events, and Phase 5 P5-2 added the `stampMedicationChange` helper pattern for this. P6-6 extends that pattern: when the FHIR sync worker processes a new `Condition` (MI, HF, stroke) or `Observation` (echo EF, CAC score, NT-proBNP), it immediately calls `ClassifyCKMStage(input)` in-process and compares to the stored stage. If the stage changed, publish `CKM_STAGE_TRANSITION` to KB-19 and update `PatientProfile.CKMStageV2`. No new consumer. No new topic. The trigger fires where the data lands.

### Decision 7: Every sub-project ends with Verification Questions

Carried over from Phase 5 as the permanent standard. The implementer's completion report must answer each with **yes / no / evidence** — not narrative. This prevents the Phase 4-era failure mode where "the stability engine shipped" hid two unshipped properties.

### Decision 8: Phase 6 does NOT ship Gap 9-13 work (FHIR outbound, explainability chain, audit event sourcing, circuit breaking, formulary accessibility)

These are documented in the Phase 5 review as post-Phase-6 work. Phase 6 is the **integration phase** — wiring existing detectors into autonomous operation. Gaps 9-13 are compliance / UX / scale concerns that belong to Phase 7+.

### Decision 9: CKM 4c transitions trigger MandatoryMedChecker via event-driven cross-service hop, NOT in-process call

Late recon confirmed `MandatoryMedChecker` exists at [kb-23-decision-cards/internal/services/mandatory_med_checker.go](backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/mandatory_med_checker.go) — but it lives in **KB-23**, not KB-20. P6-6's CKM event handler runs in-process inside KB-20's FHIR sync worker, which cannot import KB-23 internal packages (cross-service boundary).

The integration uses event-driven cross-service handoff, mirroring the P6-2 pattern:

1. KB-20's CKM event handler classifies the new stage in-process via `ClassifyCKMStage`
2. If the stage changed, KB-20 publishes `CKM_STAGE_TRANSITION` to **`clinical.priority-events.v1`** (the existing topic KB-23 already consumes via `PrioritySignalConsumer`) with payload `{patient_id, from_stage, to_stage, transition_date}`
3. KB-23's existing `PrioritySignalConsumer` is extended to handle the `CKM_STAGE_TRANSITION` event type. When it receives one with `to_stage == "4c"`, it synchronously invokes `MandatoryMedChecker.CheckPatient(patientID)` and generates IMMEDIATE cards for any missing mandatory medications
4. Other CKM transitions (e.g. 3a → 3b) are still published but trigger no special action — downstream consumers may use them for dashboards, audit trails, or future sub-projects

This pattern is consistent with the platform direction: **event-driven cross-service triggers over HTTP RPCs**. It reuses the topic that's already running, requires no new endpoint on KB-23, and keeps the services loosely coupled so each can be deployed independently.

---

## File Structure Overview

### KB-20 changes (P6-1 + P6-6)

| Action | File | Sub-project |
|---|---|---|
| Create | `kb-20-patient-profile/internal/services/intervention_timeline_service.go` | P6-1 — builds timeline from medication_states + interventions |
| Create | `kb-20-patient-profile/internal/services/intervention_timeline_service_test.go` | P6-1 |
| Create | `kb-20-patient-profile/internal/api/intervention_handlers.go` | P6-1 — `GET /api/v1/patient/:id/intervention-timeline` |
| Modify | `kb-20-patient-profile/internal/api/routes.go` | P6-1 |
| Modify | `kb-20-patient-profile/internal/fhir/fhir_sync_worker.go` | P6-6 — add `stampCKMStageRecomputation` hook on Condition/Observation events |
| Create | `kb-20-patient-profile/internal/services/ckm_event_handler.go` | P6-6 — runs `ClassifyCKMStage` + publishes `CKM_STAGE_TRANSITION` |
| Create | `kb-20-patient-profile/internal/services/ckm_event_handler_test.go` | P6-6 |

### KB-23 changes (P6-2 + shared inertia move)

| Action | File | Sub-project |
|---|---|---|
| Create | `kb-23-decision-cards/internal/services/renal_reactive_consumer.go` | P6-2 — Kafka consumer on `clinical.state-changes.v1` filtering for eGFR labs |
| Create | `kb-23-decision-cards/internal/services/renal_reactive_consumer_test.go` | P6-2 |
| Modify | `kb-23-decision-cards/main.go` | P6-2 — wire the new consumer at startup |
| Move | `kb-23-decision-cards/internal/services/inertia_detector.go` → `backend/shared-infrastructure/clinical-intelligence/inertia/` | P6-1 — shared package so KB-26 can import |
| Move | `kb-23-decision-cards/internal/services/inertia_card_generator.go` → same shared location | P6-1 |
| Modify | `kb-23-decision-cards/internal/services/*_test.go` that imports inertia | P6-1 — fix import paths |

### KB-26 changes (P6-1 + P6-3 + P6-4 + P6-5)

| Action | File | Sub-project |
|---|---|---|
| Create | `kb-26-metabolic-digital-twin/internal/services/inertia_orchestrator.go` | P6-1 — fetches targets + timeline, runs DetectInertia, dampens, persists, publishes |
| Create | `kb-26-metabolic-digital-twin/internal/services/inertia_orchestrator_test.go` | P6-1 |
| Create | `kb-26-metabolic-digital-twin/internal/services/inertia_repository.go` | P6-1 — persists weekly verdict history |
| Create | `kb-26-metabolic-digital-twin/migrations/009_inertia_verdict_history.sql` | P6-1 |
| Modify | `kb-26-metabolic-digital-twin/internal/services/inertia_weekly_batch.go` | P6-1 — replace heartbeat with orchestrator invocation |
| Modify | `kb-26-metabolic-digital-twin/internal/clients/kb20_client.go` | P6-1 — add `FetchInterventionTimeline` method |
| Verify | `kb-26-metabolic-digital-twin/internal/services/trajectory_engine.go` | P6-3 — confirm Compute runs on MHRI recompute path |
| Verify | `kb-26-metabolic-digital-twin/internal/services/trajectory_publisher.go` | P6-3 — confirm event publication wired |
| Modify (if needed) | `kb-26-metabolic-digital-twin/internal/services/mri_service.go` (or similar) | P6-3 — add the decomposition call if missing |
| Create | `kb-26-metabolic-digital-twin/internal/services/cgm_period_report.go` | P6-4 — TIR/TBR/TAR/CV/GMI computation from raw readings |
| Create | `kb-26-metabolic-digital-twin/internal/services/cgm_period_report_test.go` | P6-4 |
| Create | `kb-26-metabolic-digital-twin/internal/services/cgm_daily_batch.go` | P6-4 — daily BatchJob checking "≥14 days since last report?" |
| Create | `kb-26-metabolic-digital-twin/internal/services/cgm_daily_batch_test.go` | P6-4 |
| Create | `kb-26-metabolic-digital-twin/migrations/010_cgm_period_report.sql` | P6-4 |
| Modify | `kb-26-metabolic-digital-twin/internal/models/patient_profile_mirror.go` (or wherever KB-20 profile is mirrored) | P6-4 — add `cgm_active` field |
| Create | `kb-26-metabolic-digital-twin/internal/services/renal_anticipatory_batch.go` | P6-5 — monthly BatchJob wrapping FindApproachingThresholds |
| Create | `kb-26-metabolic-digital-twin/internal/services/renal_anticipatory_batch_test.go` | P6-5 |
| Modify | `kb-26-metabolic-digital-twin/main.go` | P6-1 + P6-4 + P6-5 — register new BatchJobs at startup |

### Shared infrastructure changes (P6-1 package move)

| Action | File | Sub-project |
|---|---|---|
| Create | `backend/shared-infrastructure/clinical-intelligence/inertia/detector.go` (moved from kb-23) | P6-1 |
| Create | `backend/shared-infrastructure/clinical-intelligence/inertia/card_generator.go` (moved from kb-23) | P6-1 |
| Create | `backend/shared-infrastructure/clinical-intelligence/inertia/go.mod` | P6-1 — standalone Go module |
| Create | `backend/shared-infrastructure/clinical-intelligence/inertia/README.md` | P6-1 |

### Prometheus metrics (all sub-projects add to existing collector)

| Metric | Sub-project |
|---|---|
| `inertia_detected_total{domain,severity}` | P6-1 |
| `inertia_batch_duration_seconds` | P6-1 |
| `inertia_batch_patients_total{outcome}` | P6-1 |
| `renal_contraindication_detected_total{drug_class}` | P6-2 |
| `renal_reactive_latency_seconds` (event → card) | P6-2 |
| `domain_trajectory_events_total{event_type}` | P6-3 |
| `cgm_period_reports_generated_total` | P6-4 |
| `cgm_tir_distribution{bucket}` | P6-4 |
| `renal_anticipatory_alerts_total{drug_class,horizon}` | P6-5 |
| `stale_egfr_total{ckd_stage}` | P6-5 |
| `ckm_stage_transitions_total{from,to}` | P6-6 |

**Total: ~32 files created, ~10 modified, 2 migrations, 11 new Prometheus metrics across 3 services + 1 new shared package.**

---

# Sub-project P6-1: Therapeutic Inertia Weekly Batch

**Priority:** Highest clinical impact of Phase 6. Inertia is invisible without periodic reassessment.
**Effort:** ~3 days (1 day KB-20 intervention timeline + 0.5 day shared package + 1.5 day KB-26 orchestrator + 0.5 day batch wiring/tests).
**Depends on:** Phase 5 `BatchJob.ShouldRun` abstraction (shipped in `728ce373`).

This sub-project replaces the P5-3 `InertiaWeeklyBatch` heartbeat with a real weekly inertia scan that fans out across all active patients, assembles the detector input from KB-20 + KB-26 data, runs `DetectInertia`, dampens flapping verdicts, persists the verdict history, and publishes events to KB-19.

## Task P6-1.1: Build KB-20 intervention timeline service + HTTP endpoint

**Files:**
- Create: `kb-20-patient-profile/internal/services/intervention_timeline_service.go`
- Create: `kb-20-patient-profile/internal/services/intervention_timeline_service_test.go`
- Create: `kb-20-patient-profile/internal/api/intervention_handlers.go`
- Modify: `kb-20-patient-profile/internal/api/routes.go`

The existing `intervention_timeline.go` in KB-20 contains only data types and the drug-class-to-domain mapping. This task builds the actual service that queries `medication_states` + `interventions` tables, groups by clinical domain (GLYCAEMIC / HEMODYNAMIC / RENAL / LIPID), and returns the latest action per domain.

- [ ] **Step 1:** Read existing `intervention_timeline.go` and confirm `LatestDomainAction` struct shape. Decide on the repository query: most recent medication change or clinical action per `(patient_id, domain)` tuple within a 90-day lookback.
- [ ] **Step 2:** Write failing test for `BuildTimeline(patientID) (*InterventionTimelineResult, error)` — seed 5 medication_state rows across 3 domains with different timestamps, assert the latest per domain is returned.
- [ ] **Step 3:** Implement `BuildTimeline`. Query `medication_states WHERE patient_id = ? AND start_date > ?`, iterate results, map drug class to domain via existing `MapDrugClassToDomain`, retain latest per domain.
- [ ] **Step 4:** Add `GET /api/v1/patient/:id/intervention-timeline` handler that wraps the service call and returns JSON.
- [ ] **Step 5:** Register the route in `routes.go`.
- [ ] **Step 6:** Integration test hitting the handler via `httptest.NewRecorder`.
- [ ] **Step 7:** Run `go test ./...` in KB-20. Commit with message `feat(kb20): intervention timeline service + GET /patient/:id/intervention-timeline (Phase 6 P6-1)`.

## Task P6-1.2: Move inertia detector to shared-infrastructure/clinical-intelligence package

**Files:**
- Move: `kb-23-decision-cards/internal/services/inertia_detector.go` → `backend/shared-infrastructure/clinical-intelligence/inertia/detector.go`
- Move: `kb-23-decision-cards/internal/services/inertia_card_generator.go` → same directory `card_generator.go`
- Create: `backend/shared-infrastructure/clinical-intelligence/inertia/go.mod`
- Modify: all KB-23 import sites
- Modify: `kb-23-decision-cards/go.mod` — add `replace` directive pointing to the shared package

This enables KB-26's orchestrator to import `inertia.DetectInertia` without cross-service imports of KB-23 internals. KB-23 continues to use the detector via the shared package.

- [ ] **Step 1:** Read both files; confirm they have no KB-23-specific imports other than `kb-23-decision-cards/internal/models`.
- [ ] **Step 2:** Decide on model ownership — `PatientInertiaReport` and related types currently live in `kb-23-decision-cards/internal/models/therapeutic_inertia.go`. Move these to `shared-infrastructure/clinical-intelligence/inertia/models.go` as well.
- [ ] **Step 3:** Create the new shared package with its own `go.mod` (module name `clinical-intelligence/inertia`). Keep dependencies minimal — only stdlib + time.
- [ ] **Step 4:** Move files; fix package declarations.
- [ ] **Step 5:** Add `replace` directives in KB-23 and (eventually) KB-26 `go.mod` files to resolve the shared package from local disk.
- [ ] **Step 6:** Update all KB-23 callers of the moved functions — the existing `InertiaCardBuilder`, `FourPillarEvaluator`, tests.
- [ ] **Step 7:** `go build ./... && go test ./...` in KB-23. No regressions.
- [ ] **Step 8:** Commit: `refactor(inertia): move detector + card generator to shared-infrastructure/clinical-intelligence (Phase 6 P6-1 prep)`.

## Task P6-1.3: Build KB-26 InertiaOrchestrator

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/inertia_orchestrator.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/inertia_orchestrator_test.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/inertia_repository.go`
- Create: `kb-26-metabolic-digital-twin/migrations/009_inertia_verdict_history.sql`
- Modify: `kb-26-metabolic-digital-twin/internal/clients/kb20_client.go` — add `FetchInterventionTimeline` method

The orchestrator is the KB-26 analogue of `BPContextOrchestrator` — for each patient it fetches inputs, runs the detector, dampens flapping, persists history, publishes events.

- [ ] **Step 1:** Add `FetchInterventionTimeline(ctx, patientID) (*InterventionTimelineDTO, error)` to `kb20_client.go`. Mirror the `FetchBPReadings` pattern.
- [ ] **Step 2:** Create the `inertia_verdict_history` migration — columns: `id`, `patient_id`, `week_start_date`, `verdicts_jsonb`, `dual_domain_detected`, `created_at`; unique index on `(patient_id, week_start_date)`.
- [ ] **Step 3:** Create `inertia_repository.go` with `SaveVerdict(snapshot)` and `FetchLatest(patientID)` methods mirroring `bp_context_repository.go`.
- [ ] **Step 4:** Write failing orchestrator test: stub KB-20 client returning a timeline with recent glycaemic action but no hemodynamic action, seed patient profile in KB-26 test DB with glycaemic target unmet + hemodynamic target unmet for 12 weeks, assert `InertiaOrchestrator.EvaluatePatient(ctx, patientID)` returns a report with hemodynamic inertia detected and persists a row to `inertia_verdict_history`.
- [ ] **Step 5:** Implement `InertiaOrchestrator.EvaluatePatient(ctx, patientID)`:
  1. Fetch patient profile from repo
  2. Compute `GlycaemicTargetStatus`, `HemodynamicTargetStatus`, `RenalTargetStatus` inline via the existing functions in `target_status.go`
  3. Fetch intervention timeline from KB-20
  4. Assemble `InertiaDetectorInput`
  5. Call `inertia.DetectInertia(input)` (now in shared package)
  6. Fetch previous week's verdict, apply stability check (simple rule: if current week's verdict differs from previous week's AND the patient's target status changed, accept; otherwise dampen by retaining previous verdict)
  7. Persist verdict snapshot
  8. If new inertia detected, publish `INERTIA_DETECTED` to KB-19
  9. If dual-domain, publish `DUAL_DOMAIN_INERTIA_DETECTED`
- [ ] **Step 6:** Build + test. Commit: `feat(kb26): InertiaOrchestrator — weekly detector wiring for all active patients (Phase 6 P6-1)`.

## Task P6-1.4: Wire InertiaOrchestrator into InertiaWeeklyBatch

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/services/inertia_weekly_batch.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/inertia_weekly_batch_test.go`
- Modify: `kb-26-metabolic-digital-twin/main.go`

Replace the P5-3 heartbeat `log.Info("inertia weekly heartbeat", ...)` with a real loop that calls `orchestrator.EvaluatePatient(ctx, patientID)` for every active patient.

- [ ] **Step 1:** Update `InertiaWeeklyBatch` constructor to accept an `InertiaOrchestrator` dependency.
- [ ] **Step 2:** Replace `Run`'s log-only body with a fan-out loop over `ListActivePatientIDs`, calling the orchestrator per patient. Bounded concurrency (configurable, default 4) mirroring `BPContextDailyBatch`.
- [ ] **Step 3:** Update the P5-3 heartbeat tests — the new tests should verify the orchestrator stub is called once per patient.
- [ ] **Step 4:** Wire the real orchestrator in `main.go` at startup (after KB-20 client is available).
- [ ] **Step 5:** Build + test. Commit: `feat(kb26): wire InertiaOrchestrator to InertiaWeeklyBatch (Phase 6 P6-1)`.

## P6-1 Verification Questions

1. Does `KB20Client.FetchInterventionTimeline` return a non-nil timeline for a patient with recent medication changes?
2. Does `InertiaOrchestrator.EvaluatePatient` detect inertia for a patient with unmet targets + no recent intervention in that domain?
3. Does the stability check prevent a one-week flip-flop from being published as INERTIA_DETECTED?
4. Does `inertia_verdict_history` contain one row per patient per week after a batch run?
5. Does `InertiaWeeklyBatch.Run` fan out across all active patients at the configured concurrency?
6. Do the Prometheus metrics `inertia_detected_total` and `inertia_batch_duration_seconds` increment during a test batch run?
7. Are KB-20, KB-23, KB-26 full test suites green?

---

# Sub-project P6-2: Reactive Renal Dose Gating (Kafka Consumer)

**Priority:** Patient safety. Contraindications detected within minutes of an eGFR lab result landing, not at a weekly batch.
**Effort:** ~2 days.
**Depends on:** existing `clinical.state-changes.v1` Kafka topic, existing `RenalDoseGate.EvaluatePatient` in [kb-23/internal/services/renal_dose_gate.go](backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/renal_dose_gate.go).

When a new eGFR lab result is published to `clinical.state-changes.v1` by KB-20's FHIR sync worker, a new consumer in KB-23 filters for eGFR events, fetches the patient's active medications, runs `RenalDoseGate.EvaluatePatient`, and generates `RENAL_CONTRAINDICATION` cards for any flagged medications.

## Task P6-2.1: Verify KB-20 publishes LAB_RESULT events with eGFR details

**Files:**
- Read only: `kb-20-patient-profile/internal/services/kafka_outbox_relay.go`
- Read only: `kb-20-patient-profile/internal/services/lab_service.go`

- [ ] **Step 1:** Confirm that when a new `LabEntry` with `LabType="eGFR"` is created, KB-20 writes an outbox row that the relay publishes to `clinical.state-changes.v1`. If not, adding that producer side is a prerequisite task for P6-2.
- [ ] **Step 2:** Confirm the payload shape — must include `patient_id`, `lab_type`, `value`, `measured_at`, `event_type`. If the outbox doesn't emit `LAB_RESULT` events yet, add them before proceeding.
- [ ] **Step 3:** Document findings in the task notes. If KB-20 needs producer-side work, this becomes Task P6-2.1a.

## Task P6-2.2: Build KB-23 RenalReactiveConsumer

**Files:**
- Create: `kb-23-decision-cards/internal/services/renal_reactive_consumer.go`
- Create: `kb-23-decision-cards/internal/services/renal_reactive_consumer_test.go`

- [ ] **Step 1:** Copy the structural pattern from `priority_signal_consumer.go`. The new consumer subscribes to `clinical.state-changes.v1`, group ID `kb-23-renal-reactive`, with the same Zap logger + kafka.Reader setup.
- [ ] **Step 2:** Handler function: parse message → check `event_type == "LAB_RESULT"` and `lab_type == "eGFR"` → extract `patient_id` + `value` + `measured_at`.
- [ ] **Step 3:** For each matching event:
  1. Fetch patient's active medications from KB-20 via existing `kb20Client`
  2. Construct `models.RenalStatus` with `eGFRValue` + `measured_at`
  3. Call `renalDoseGate.EvaluatePatient(patientID, renalStatus, meds)`
  4. For each `CONTRAINDICATED` result, generate a `RENAL_CONTRAINDICATION` card via the existing card builder
  5. For each `DOSE_ADJUST_REQUIRED` result, generate a `RENAL_DOSE_REDUCE` card
  6. Persist cards via `DecisionCardRepository.Create`
  7. Publish `RENAL_CONTRAINDICATION_DETECTED` event to KB-19 via existing `KB19Publisher`
- [ ] **Step 4:** Write a test that uses a fake kafka.Reader (or mocked interface) to deliver a LAB_RESULT message with eGFR=25 for a patient on metformin, assert a CONTRAINDICATED card is created and KB-19 event published.
- [ ] **Step 5:** Wire the consumer in `kb-23-decision-cards/main.go` at startup alongside `PrioritySignalConsumer`.
- [ ] **Step 6:** Build + test + commit: `feat(kb23): RenalReactiveConsumer — auto-gate medications on new eGFR labs (Phase 6 P6-2)`.

## P6-2 Verification Questions

1. Does KB-20's outbox publish `LAB_RESULT` events with eGFR details on `clinical.state-changes.v1`?
2. Does the new KB-23 consumer subscribe successfully at startup?
3. Does an eGFR=25 event for a metformin patient produce a CONTRAINDICATED card within 10 seconds (measured by `renal_reactive_latency_seconds` metric)?
4. Does an eGFR=55 event (normal) produce no card?
5. Is the `renal_contraindication_detected_total{drug_class}` counter incremented with the correct label?
6. Does shutdown drain the consumer cleanly (via the existing graceful shutdown pattern)?

---

# Sub-project P6-3: Domain Decomposition Wiring Verification

**Priority:** Low effort, high integration value. Connects the Module 13 velocity gap.
**Effort:** ~1 day (likely less — partially wired already).
**Depends on:** existing `TrajectoryEngine.Compute` in [kb-26/internal/services/trajectory_engine.go](backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/trajectory_engine.go), existing `DomainTrajectoryComputedEvent` in `trajectory_publisher.go`.

P6-3 is a **verification-first sub-project**. The recon found that `TrajectoryEngine.Compute` and `DomainTrajectoryComputedEvent` both exist. The open question is whether they're invoked from the MHRI recomputation path or only from a standalone handler. If already wired, P6-3 is zero code. If missing, P6-3 adds one hook.

## Task P6-3.1: Verify the MHRI → trajectory invocation path

**Files:**
- Read only: `kb-26-metabolic-digital-twin/internal/services/mri_service.go` (or wherever MHRI computation lives)
- Read only: `kb-26-metabolic-digital-twin/internal/services/trajectory_engine.go`

- [ ] **Step 1:** Find the function that computes MHRI scores (grep for `ComputeMRI` / `MRIService` / similar).
- [ ] **Step 2:** Check whether that function calls `TrajectoryEngine.Compute` after computing the composite score.
- [ ] **Step 3:** Check whether the result is published via `DomainTrajectoryComputedEvent`.
- [ ] **Step 4:** Outcome branches:
  - **A: Already wired end-to-end.** Document in task notes, jump to Task P6-3.2 for metrics verification.
  - **B: Computed but not published.** Add the event publication call.
  - **C: Not computed at all.** Add both the `TrajectoryEngine.Compute` call and the publish.

## Task P6-3.2: Add/verify Prometheus metrics

**Files:**
- Modify (if missing): metrics collector file in KB-26

- [ ] **Step 1:** Check whether `domain_trajectory_events_total{event_type}` exists as a metric.
- [ ] **Step 2:** If missing, add it to the metrics collector with labels for `CONCORDANT_DETERIORATION`, `DOMAIN_DIVERGENCE`, `BEHAVIORAL_LEADING_INDICATOR`.
- [ ] **Step 3:** Wire the counter increment in the trajectory event publication path.
- [ ] **Step 4:** Write a test asserting the counter increments when the publication path is exercised.

## Task P6-3.3: Add stability check for concordant deterioration events

**Files:**
- Modify: trajectory publication path in KB-26

The stability concern here is: a single week's MHRI computation might show concordant deterioration that resolves the following week as data revisions land. An event should fire only when the deterioration has been sustained for ≥2 consecutive computations. This mirrors the P5-1 dwell pattern but for a different signal.

- [ ] **Step 1:** Decide: use the existing `pkg/stability` engine with a new policy, or implement a minimal "previous computation match" check inline?
- [ ] **Step 2:** Implement the chosen approach.
- [ ] **Step 3:** Test: simulate two consecutive computations with concordant deterioration → event fires on second; simulate a flap (deterioration → resolved → deterioration) → event does not fire on the second deterioration.
- [ ] **Step 4:** Commit: `feat(kb26): domain trajectory events with sustained-deterioration stability check (Phase 6 P6-3)`.

## P6-3 Verification Questions

1. Does `TrajectoryEngine.Compute` run on every MHRI recomputation?
2. Is `DomainTrajectoryComputedEvent` published to KB-19 after every computation?
3. Does `domain_trajectory_events_total` metric exist and increment correctly?
4. Does a single-week concordant deterioration followed by resolution NOT publish `CONCORDANT_DETERIORATION` (sustained check works)?
5. Does two-week sustained concordant deterioration publish the event on week 2?

---

# Sub-project P6-4: CGM 14-Day Period Report + Daily Batch

**Priority:** Completes the CGM data loop. Largest sub-project by effort because the period report computation itself must be built.
**Effort:** ~3 days (scope-contracted from original 3-4 to stay minimal; full AGP deferred to Phase 7).
**Depends on:** existing CGM reading ingestion (via Module 3 or direct device upload — needs recon), existing `ComputeGlucoseDomainScore` in `cgm_analytics.go`.

Per Locked Decision 5, P6-4 ships only the minimal-viable slice: TIR / TBR / TAR / CV / GMI computation + the 14-day trigger gate + glucose domain score update. Full AGP percentile overlays are Phase 7.

## Task P6-4.1: Discovery — where do raw CGM readings live?

**Files:**
- Investigation only.

- [ ] **Step 1:** Grep for `cgm_readings`, `CGMReading`, `glucose_reading` tables and models across KB-20 and KB-26. Determine which service owns the raw reading store and how to query it.
- [ ] **Step 2:** Confirm or refute: do `patient_profile.cgm_active` or `has_cgm` fields exist anywhere?
- [ ] **Step 3:** Check if `Module 3` Flink CGM operator exists and whether it writes period summaries we can reuse.
- [ ] **Step 4:** Document outcome. Two branches:
  - **A: Readings exist with clean query path.** Proceed with P6-4.2.
  - **B: No raw reading store.** This sub-project is blocked — deferred to when the ingestion ships.

## Task P6-4.2: Build CGMPeriodReport computation

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/cgm_period_report.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/cgm_period_report_test.go`
- Create: `kb-26-metabolic-digital-twin/migrations/010_cgm_period_report.sql`

- [ ] **Step 1:** Schema for `cgm_period_reports`: `patient_id`, `period_start`, `period_end`, `reading_count`, `tir_pct`, `tbr_pct`, `tar_pct`, `cv_pct`, `gmi`, `gri`, `created_at`.
- [ ] **Step 2:** Write failing tests for `ComputePeriodReport(readings []GlucoseReading, periodStart, periodEnd time.Time) CGMPeriodReport`. Cases:
  - All readings in target range → TIR=100, TBR=0, TAR=0
  - Mix of high/low/in-range → correct percentages
  - Insufficient data (<70% capture) → `SufficientData: false`
  - Empty readings → zero report with `SufficientData: false`
- [ ] **Step 3:** Implement the computation. Formulas per ADA 2023:
  - TIR = % readings in [70, 180] mg/dL
  - TBR = % readings <70 mg/dL
  - TAR = % readings >180 mg/dL
  - CV = (stdev / mean) * 100
  - GMI = 3.31 + 0.02392 * mean_glucose_mg_dL
  - GRI = placeholder — use simple formula or defer
- [ ] **Step 4:** Tests pass. Commit: `feat(kb26): CGMPeriodReport computation — TIR/TBR/TAR/CV/GMI (Phase 6 P6-4)`.

## Task P6-4.3: Build CGMDailyBatch job

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/cgm_daily_batch.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/cgm_daily_batch_test.go`
- Modify: `kb-26-metabolic-digital-twin/main.go`

- [ ] **Step 1:** Implement `CGMDailyBatch` implementing the `BatchJob` interface from P5-3. `ShouldRun` returns true when `now.Hour() == 1` (01:00 UTC default).
- [ ] **Step 2:** `Run`: iterate active CGM patients, for each check "days since last period report ≥ 14?" via repository query. If yes:
  1. Fetch readings from the last 14 days
  2. Call `ComputePeriodReport`
  3. Persist to `cgm_period_reports`
  4. Feed the result into `ComputeGlucoseDomainScore` to update the MHRI glucose domain
  5. Publish `CGM_REPORT_AVAILABLE` to KB-19
  6. Stability check: if previous report's GRI zone differs and the new one is worse, publish `CGM_DETERIORATION_DETECTED`
- [ ] **Step 3:** Test with a stub repository returning 5 active CGM patients, 2 of whom have ≥14 days since last report. Assert those 2 produce new report rows.
- [ ] **Step 4:** Register the job in `main.go`. Commit: `feat(kb26): CGMDailyBatch — 14-day period report trigger (Phase 6 P6-4)`.

## P6-4 Verification Questions

1. Does `ComputePeriodReport` return correct TIR/TBR/TAR/CV/GMI for a known test fixture?
2. Does `CGMDailyBatch.ShouldRun` fire at 01:00 UTC and nowhere else?
3. Does `CGMDailyBatch.Run` skip patients whose last report is <14 days old?
4. Does a new period report update the patient's glucose domain score in KB-26 state?
5. Does `CGM_REPORT_AVAILABLE` fire on new report, and `CGM_DETERIORATION_DETECTED` fire only on worsened GRI zone?

---

# Sub-project P6-5: Renal Anticipatory Monthly Batch

**Priority:** Lower urgency than the reactive gate (P6-2). Proactive clinical planning for patients approaching renal thresholds.
**Effort:** ~1 day.
**Depends on:** existing `FindApproachingThresholds` in [kb-23/internal/services/renal_anticipatory.go](backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/renal_anticipatory.go), existing `DetectStaleEGFR` in `stale_egfr_detector.go`.

A monthly batch that finds patients whose projected eGFR will cross a contraindication threshold within the next 6-12 months. Lower urgency than the reactive gate but important for planning.

## Task P6-5.1: Build RenalAnticipatoryBatch job

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/renal_anticipatory_batch.go` (or in KB-23 if simpler given where `FindApproachingThresholds` lives)
- Create: `kb-26-metabolic-digital-twin/internal/services/renal_anticipatory_batch_test.go`
- Modify: `main.go` of the chosen service

- [ ] **Step 1:** Decide service placement. Since `FindApproachingThresholds` is in KB-23 and needs access to KB-23's medication state, the batch lives in KB-23 and registers with a KB-23-local scheduler. **Phase 6 Decision**: this is the first KB-23 batch consumer. Create a KB-23 scheduler (copying the Phase 5 KB-26 abstraction pattern) OR call KB-23 from a KB-26 monthly batch via HTTP. **Recommended: copy the BatchScheduler pattern into KB-23** — same pattern, new host service.
- [ ] **Step 2:** Implement `RenalAnticipatoryBatch` with `ShouldRun` returning true when `now.Day() == 1 && now.Hour() == 4` (1st of month, 04:00 UTC).
- [ ] **Step 3:** `Run`: iterate patients on renal-sensitive medications, call `FindApproachingThresholds(patientID, 12*30*24*time.Hour)`, call `DetectStaleEGFR(patientID)`, publish `RENAL_THRESHOLD_APPROACHING` and `STALE_EGFR` events.
- [ ] **Step 4:** Test + commit.

## P6-5 Verification Questions

1. Does `RenalAnticipatoryBatch.ShouldRun` fire on the 1st of each month at 04:00 UTC?
2. Does a patient with eGFR trending downward produce a `RENAL_THRESHOLD_APPROACHING` event?
3. Does a patient with overdue eGFR labs produce a `STALE_EGFR` event?
4. Do the Prometheus counters increment correctly?

---

# Sub-project P6-6: CKM Substaging Event-Driven Trigger

**Priority:** Low volume but important when events occur.
**Effort:** ~2 days (1.5d KB-20 handler + 0.5d KB-23 consumer extension per Decision 9).
**Depends on:** existing `ClassifyCKMStage` in [kb-20/internal/services/ckm_classifier.go](backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/ckm_classifier.go), existing `MandatoryMedChecker` in [kb-23/internal/services/mandatory_med_checker.go](backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/mandatory_med_checker.go), existing `PrioritySignalConsumer` in KB-23, existing FHIR sync worker pattern from P5-2.

When KB-20 receives a new `Condition` (MI, HF, stroke) or `Observation` (echo EF, CAC, NT-proBNP), immediately reclassify CKM stage in-process. The classification + event publication happens in KB-20; the downstream `MandatoryMedChecker` invocation on 4c transitions happens in KB-23 via the event consumer (per Decision 9).

## Task P6-6.1: Build KB-20 CKM event handler + publish CKM_STAGE_TRANSITION

**Files:**
- Create: `kb-20-patient-profile/internal/services/ckm_event_handler.go`
- Create: `kb-20-patient-profile/internal/services/ckm_event_handler_test.go`
- Modify: `kb-20-patient-profile/internal/fhir/fhir_sync_worker.go` — add `recomputeCKMStage` hook on relevant Condition/Observation events
- Modify: `kb-20-patient-profile/internal/services/kafka_outbox_relay.go` (or wherever event types are defined) — add `CKM_STAGE_TRANSITION` event type if not present

- [ ] **Step 1:** Implement `CKMEventHandler.HandleCondition(condition)` and `HandleObservation(observation)` that:
  1. Build `CKMClassifierInput` from patient profile + new event data
  2. Call `ClassifyCKMStage(input)` — confirmed exists at `ckm_classifier.go:43`
  3. Compare to current `PatientProfile.CKMStageV2`
  4. If unchanged: no-op (early return, no event published — this prevents false-positive event spam)
  5. If changed: update `PatientProfile.CKMStageV2`, persist, then publish a `CKM_STAGE_TRANSITION` event to the existing `clinical.priority-events.v1` topic via the kafka_outbox_relay. Payload must include `{patient_id, from_stage, to_stage, transition_date, triggered_by_event_type, triggered_by_event_id}`
- [ ] **Step 2:** In `fhir_sync_worker.go`, after persisting a new Condition or Observation that could affect CKM staging (MI, HF, stroke conditions; echo EF, CAC, NT-proBNP observations), call `ckmEventHandler.HandleCondition(...)` or `HandleObservation(...)`. Mirror the placement of the P5-2 `stampMedicationChange` call.
- [ ] **Step 3:** Test with a patient profile seeded at CKM 3b. Inject a new HF Condition. Assert `ClassifyCKMStage` returns `4a` (or whatever the input warrants), the profile updates, and the outbox row contains a `CKM_STAGE_TRANSITION` event with `from_stage="3b" to_stage="4a"`.
- [ ] **Step 4:** Test with a patient at CKM 3b receiving a Condition that doesn't change the stage — assert no event is published.
- [ ] **Step 5:** Build + test KB-20. Commit: `feat(kb20): CKM event handler — classify on new Condition/Observation, publish CKM_STAGE_TRANSITION (Phase 6 P6-6)`.

## Task P6-6.2: Extend KB-23 PrioritySignalConsumer to handle CKM_STAGE_TRANSITION

**Files:**
- Modify: `kb-23-decision-cards/internal/services/priority_signal_consumer.go` — add `CKM_STAGE_TRANSITION` event-type handler
- Modify: `kb-23-decision-cards/internal/services/priority_signal_consumer_test.go` (or create if missing)
- Modify: `kb-23-decision-cards/main.go` — wire `MandatoryMedChecker` into the consumer's dependencies if not already present

The KB-23 consumer already subscribes to `clinical.priority-events.v1`. This task extends its event-type dispatch to recognize `CKM_STAGE_TRANSITION` and act on `to_stage == "4c"` transitions specifically.

- [ ] **Step 1:** Read the existing `priority_signal_consumer.go` to understand the dispatch pattern (the consumer currently routes based on `PriorityRouteAction`). Decide: add a new `RouteAction` for `CKM_4C_MANDATORY_MEDS_CHECK`, or pattern-match on event type within the existing handler.
- [ ] **Step 2:** Write a failing test that delivers a stub `CKM_STAGE_TRANSITION` message with `to_stage="4c"` to the consumer, asserts `MandatoryMedChecker.CheckPatient(patientID)` is called once, and any IMMEDIATE cards returned by the checker are persisted via `DecisionCardRepository`.
- [ ] **Step 3:** Implement the dispatch. On a `CKM_STAGE_TRANSITION` event with `to_stage="4c"`:
  1. Call `MandatoryMedChecker.CheckPatient(patientID)`
  2. For each missing mandatory medication, build an IMMEDIATE card and persist via the existing card repository
  3. Increment `ckm_4c_mandatory_med_alerts_total{drug_class}` Prometheus counter
- [ ] **Step 4:** Test the negative case — a `CKM_STAGE_TRANSITION` with `to_stage="3b"` produces no `MandatoryMedChecker` invocation and no cards.
- [ ] **Step 5:** Wire the `MandatoryMedChecker` instance into the consumer constructor in `main.go` if not already wired.
- [ ] **Step 6:** Build + test KB-23. Commit: `feat(kb23): PrioritySignalConsumer handles CKM_STAGE_TRANSITION → MandatoryMedChecker on 4c (Phase 6 P6-6)`.

## P6-6 Verification Questions

1. Does a new MI Condition trigger `ClassifyCKMStage` + publish `CKM_STAGE_TRANSITION` to `clinical.priority-events.v1`?
2. Does a new echo with reduced EF trigger stage recomputation?
3. Does an unchanged stage produce NO event (no false positives)?
4. Does a `CKM_STAGE_TRANSITION` event with `to_stage="4c"` cause the KB-23 consumer to invoke `MandatoryMedChecker.CheckPatient(patientID)`?
5. Does a `CKM_STAGE_TRANSITION` event with `to_stage="3b"` produce no `MandatoryMedChecker` invocation?
6. Are the Prometheus counters `ckm_stage_transitions_total{from,to}` (KB-20) and `ckm_4c_mandatory_med_alerts_total{drug_class}` (KB-23) incremented correctly?
7. Are KB-20 and KB-23 full test suites green?

---

# Execution Order Recommendation

Sub-projects can ship in any order except:
- **P6-1 Task P6-1.2 (shared package move)** should land before Task P6-1.3 because P6-1.3 depends on the shared package.
- **P6-2 Task P6-2.1 (KB-20 producer verification)** should land before P6-2.2 because the consumer depends on the producer existing.
- **P6-3 Task P6-3.1 (verification task)** should land before any P6-3 code changes because it determines whether code changes are needed at all.

Recommended sequence for a multi-session Phase 6 execution:

**Session 1 (half day):** P6-3 (verification + any missing hooks) — lowest effort, highest "clean win" ratio. Get a fast one on the board before tackling the big sub-projects.

**Session 2 (half day):** P6-5 (renal anticipatory monthly batch) — clean wrap of existing `FindApproachingThresholds`. Proves the KB-23 BatchScheduler pattern extraction from Phase 5 P5-3.

**Session 3 (half day):** P6-6 (CKM event handler) — also clean wrap, in KB-20.

**Session 4-5 (2 days):** P6-1 (inertia weekly batch end-to-end) — the KB-20 timeline service + shared package move + KB-26 orchestrator + batch wiring. This is the biggest sub-project and the one that finally replaces the P5-3 heartbeat with real work.

**Session 6-7 (2 days):** P6-2 (reactive renal consumer) — Kafka consumer pattern, depends on P6-2.1 recon outcome.

**Session 8-9 (2-3 days):** P6-4 (CGM period report + batch) — the largest and least clearly specified. Do last so earlier sub-projects validate the pattern before we tackle the most new computation.

**Total: ~9-11 sessions for all of Phase 6.** This is realistic given Phase 5 took 2 sessions for 4 sub-projects at similar individual complexity.

---

# What Phase 6 Delivers

After every sub-project ships:

- **Inertia detection runs autonomously every week**, for every active patient, without a clinician asking. The P5-3 heartbeat becomes a real scan.
- **Renal contraindications are detected within minutes** of a new eGFR lab landing, not at a weekly batch. Metformin at eGFR <30 flags the same hour.
- **Domain trajectory decomposition** publishes concordant deterioration and divergence events so downstream consumers (protocol orchestrator, composite cards, dashboards) see per-domain velocities, not just the MHRI composite.
- **CGM analytics** produces period reports every 14 days per patient, updates the glucose domain score, feeds the MHRI recomputation loop.
- **Anticipatory renal alerts** fire monthly for patients approaching thresholds — clinicians plan ahead instead of reacting at crossing.
- **CKM stage transitions** reclassify in-process when clinical events land, publishing `CKM_STAGE_TRANSITION` events and triggering mandatory-medication checks on 4c transitions.

Every feature follows the same operational pattern established in Phase 5:

```
Clinical Signal → Detection → Classification → Card Generation
                                                      ↓
                                              Event Publication
                                                      ↓
                                              Stability Dampening (where applicable)
                                                      ↓
                                              Operational Monitoring
                                                      ↓
                                              Batch/Event Reassessment
```

No clinical feature stops at card generation. The physician experience becomes "the system proactively contacts me when something changes, and I trust the change is real because stability dampening filtered the noise."

---

# What Phase 6 Does NOT Deliver

Explicit deferrals:

- **Full AGP (Ambulatory Glucose Profile) percentile overlays** — deferred to Phase 7 per Locked Decision 5. P6-4 ships TIR/TBR/TAR/CV/GMI only.
- **CGM sustained-hypo streaming alerts** — belongs to the Module 3 Flink workstream, not Phase 6. Batch handles slow-moving metrics; stream handles fast-moving alerts.
- **Gap 9 (FHIR outbound)** — regulatory compliance blocker for Australia MHR / India ABDM. Separate workstream.
- **Gap 10 (Unified explainability chain)** — partially addressed by the inertia evidence builder; needs a unified `rationale` field on every card type. Phase 7.
- **Gap 11 (Clinical audit event sourcing)** — append-only event log with hash-chaining for TGA/CDSCO. Phase 7.
- **Gap 12 (Circuit breaking)** — backpressure between services when KB-20 is slow. Phase 7 infrastructure hardening.
- **Gap 13 (Formulary accessibility filtering)** — market-specific drug availability. Phase 7.

---

# Plan Summary

| Sub-project | Cadence | Trigger | Effort | Files | New Tests |
|---|---|---|---|---|---|
| P6-1 Therapeutic Inertia | Weekly (Sunday 03:00 UTC) | BatchJob (KB-26) | 3d | ~10 | ~15 |
| P6-2 Reactive Renal Gate | Per eGFR lab event | Kafka consumer (KB-23) | 2d | ~3 | ~6 |
| P6-3 Domain Decomposition | Per MHRI recompute | Inline (KB-26) | 1d | ~2 | ~4 |
| P6-4 CGM Period Report + Batch | Per-patient 14-day | Daily BatchJob (KB-26) | 3d | ~5 | ~10 |
| P6-5 Renal Anticipatory | Monthly (1st 04:00 UTC) | BatchJob (KB-23) | 1d | ~3 | ~5 |
| P6-6 CKM Event Handler + KB-23 consumer extension | Per Condition/Observation + KB-23 priority event | In-process (KB-20) + Kafka consumer extension (KB-23) | 2d | ~5 | ~8 |
| **Total** | — | — | **12d** | **~28** | **~48** |

Phase 6 brings the platform from "detectors exist, clinicians can query them" to "detectors run autonomously, publish events, dampen flapping, expose metrics." It is the integration phase that unlocks Phase 7's compliance and UX work.
