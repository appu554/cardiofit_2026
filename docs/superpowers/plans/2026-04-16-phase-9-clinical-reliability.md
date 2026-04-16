# Phase 9: Clinical Reliability — Eliminating False Positives and Surveillance Gaps

**Goal:** Take the 15-of-20 patient win rate from Phase 7-8 and close the 3 failure modes that produce false positives or miss patients entirely. After Phase 9, the system tells a nurse who to call today with fewer false alarms and catches patients who fell off the monitoring radar.

**Pre-requisite:** Phase 8 P8-1 through P8-6 shipped and pushed. Branch at `35180b95`. All three KB services build and test green.

**Prioritization frame:** The 20-patient thought experiment identified three patients we'd get wrong:
- **Patient 17** — non-adherent patient gets a false-positive inertia card (should get adherence counselling instead)
- **Patient 18** — patient who stopped monitoring is invisible to masked HTN detection
- **Patient 19** — frail polypharmacy patient gets more cards, not better cards (should get deprescribing consideration)

Phase 9 Session 1 fixes Patient 17 and Patient 18 (both shippable in one session) plus three hardening items that production traffic will need. Patient 19 (frailty/deprescribing modifier) is Phase 9 Session 2+ scope because it touches the MCU gate evaluation logic and needs clinical review before implementation.

---

## Status Snapshot — 2026-04-16

**Session 1 scope (this session):**

- [ ] **P9-A — Adherence-Exclusion Branch on Inertia Detector:** When `EngagementStatus == "DISENGAGED"` or `EngagementComposite < 0.4`, suppress the inertia verdict and generate an `ADHERENCE_GAP` card instead. Uses existing fields on PatientContext (P8-2 wired them). ~1-2 hours.
- [ ] **P9-B — Monitoring Engagement Detector:** New batch job or event-driven check that flags "patient was generating home BP readings and stopped >14 days ago." New `MONITORING_LAPSED` card template. ~2-3 hours.
- [ ] **P9-C — InertiaVerdictHistory Postgres Store:** Replace the in-memory store with a persistent table so dampening survives service restart. Additive — no orchestrator/batch changes. ~1-2 hours.
- [ ] **P9-D — CGM Period Report Deduplication:** UNIQUE constraint on `(patient_id, period_end)` + upsert logic to handle at-least-once Kafka delivery. ~30 minutes.
- [ ] **P9-E — Bounded-Concurrency Fan-Out for Batch Jobs:** Worker pool with `semaphore.Weighted` for the inertia weekly batch and renal anticipatory batch. Default concurrency 4. ~1 hour.

**Deferred to Session 2+ (documented, not blocked):**

- [ ] P9-F — Frailty / Deprescribing Context Modifier (Patient 19 fix). Needs clinical review for threshold definition.
- [ ] P9-G — FHIR Condition Sync for CKM Recomputation. Needs upstream FHIR infra changes in fhir_sync_worker.
- [ ] P9-H — Gap 9 FHIR Outbound (Australia MHR / India ABDM market access). Different workstream — compliance, not clinical intelligence.
- [ ] P9-I — Gap 10 Unified Explainability Chain. Card enhancement, not new detection.
- [ ] P9-J — Gap 11 Clinical Audit Event Sourcing. Compliance infrastructure.
- [ ] P9-K — Gap 12 Circuit Breaking. Service reliability hardening.
- [ ] P9-L — Gap 13 Formulary Accessibility Filtering. Market-specific drug availability.
- [ ] P9-M — Full AGP Percentile Overlays for CGM. Deferred since Phase 6.
- [ ] P9-N — Integration tests for GetCurrentMRI / GetMRIHistory. ~30 minutes.
- [ ] P9-O — CONTRIBUTING.md testing standards + sqlite portability helper.

---

## Locked Decisions

### Decision 1: Adherence-exclusion is a gate on the detector, not a filter on the card builder
The inertia detector's `Evaluate` method checks `EngagementStatus` BEFORE running `DetectInertia`, not after. If the patient is disengaged, the detector produces an `ADHERENCE_GAP` verdict (different domain, different card template) instead of an inertia verdict. This prevents false-positive inertia cards at the source rather than filtering them at the card layer. The MCU gate manager does not need to change.

### Decision 2: Monitoring engagement uses a simple "days since last home BP reading" heuristic
No ML, no engagement phenotype classification. Pure threshold: if the patient had ≥7 home BP readings in the 28 days before the gap, and then 0 readings in the last 14 days, they've lapsed. The threshold is conservative — it catches patients who were actively monitoring and stopped, not patients who never monitored. A future Phase 10 enhancement can use the KB-21 engagement composite for a richer model.

### Decision 3: InertiaVerdictHistory Postgres table schema matches the in-memory interface
`UNIQUE(patient_id, week_start_date)` with `ON CONFLICT DO UPDATE`. The `InertiaVerdictHistory` interface stays unchanged — only the implementation swaps from `inMemoryInertiaHistory` to a new `postgresInertiaHistory`. Zero changes to the orchestrator or batch code.

### Decision 4: CGM dedup uses Postgres upsert, not application-level dedup
`ON CONFLICT (patient_id, period_end) DO UPDATE SET ...` on the `cgm_period_reports` table. Application code stays as a simple `db.Create(report)` — the database handles the dedup. This is cleaner than checking-before-writing because it's atomic and handles concurrent consumers.

### Decision 5: Bounded concurrency uses `sync.WaitGroup` + channel-based semaphore, not `semaphore.Weighted`
`semaphore.Weighted` requires `golang.org/x/sync` which may not be in go.mod. A channel-based semaphore (`make(chan struct{}, 4)`) is zero-dependency and idiomatic Go. Each patient evaluation sends into the channel before starting and receives after completing.

---

## Metrics (Decision 9 pattern from Phase 7)

| Sub-project | Metrics |
|---|---|
| P9-A | `kb23_adherence_gap_detected_total{domain}`, `kb23_inertia_suppressed_by_adherence_total` |
| P9-B | `kb23_monitoring_lapsed_total{monitor_type}`, `kb23_monitoring_lapsed_batch_duration_seconds` |
| P9-C | (no new metrics — existing inertia metrics unchanged) |
| P9-D | `kb26_cgm_period_report_dedup_total` |
| P9-E | (no new metrics — existing batch duration histograms capture the speedup) |

---

## File Touchpoints

### KB-23 changes (P9-A + P9-B + P9-C + P9-E)

| Action | File | Sub-project |
|---|---|---|
| Modify | `kb-23/internal/services/inertia_orchestrator.go` | P9-A — add adherence check before DetectInertia |
| Create | `kb-23/templates/inertia/adherence_gap.yaml` | P9-A — card template |
| Create | `kb-23/internal/services/monitoring_engagement_detector.go` | P9-B — lapsed monitoring detection |
| Create | `kb-23/internal/services/monitoring_engagement_detector_test.go` | P9-B |
| Create | `kb-23/templates/monitoring/monitoring_lapsed.yaml` | P9-B — card template |
| Modify | `kb-23/internal/services/inertia_weekly_batch.go` | P9-B — register monitoring check in batch OR separate batch job |
| Create | `kb-23/internal/services/postgres_inertia_history.go` | P9-C — persistent store |
| Create | `kb-23/internal/services/postgres_inertia_history_test.go` | P9-C |
| Modify | `kb-23/main.go` | P9-C — swap in-memory for Postgres; P9-E — add concurrency to batches |
| Modify | `kb-23/internal/services/inertia_weekly_batch.go` | P9-E — add worker pool |
| Modify | `kb-23/internal/services/renal_anticipatory_batch.go` | P9-E — add worker pool |
| Modify | `kb-23/internal/metrics/collector.go` | P9-A + P9-B — new counters |

### KB-26 changes (P9-D)

| Action | File | Sub-project |
|---|---|---|
| Modify | `kb-26/internal/models/cgm_metrics.go` | P9-D — add unique index annotation |
| Modify | `kb-26/internal/services/cgm_period_report_repository.go` | P9-D — upsert logic |

---

# Sub-project P9-A: Adherence-Exclusion Branch on Inertia Detector

**Priority:** Fixes the Patient 17 false-positive. Highest clinical leverage in Phase 9 because it prevents the system from recommending therapy escalation when the real problem is adherence.

**Depends on:** P8-2 (EngagementStatus + EngagementComposite fields on PatientContext wire contract — already shipped).

## Task P9-A.1: Add adherence gate to InertiaOrchestrator.Evaluate

**Files:**
- Modify: `kb-23-decision-cards/internal/services/inertia_orchestrator.go`
- Modify: `kb-23-decision-cards/internal/services/inertia_input_assembler.go`

- [ ] **Step 1:** In `InertiaOrchestrator.Evaluate`, before calling `DetectInertia(input)`, check if the patient is disengaged. The assembler already populates `PatientContext` from KB-20's summary-context which carries `EngagementStatus` and `EngagementComposite`. Thread these through `InertiaDetectorInput` as new optional fields.
- [ ] **Step 2:** When `EngagementStatus == "DISENGAGED"` OR `EngagementComposite != nil && *EngagementComposite < 0.4`, skip `DetectInertia` and instead generate an `ADHERENCE_GAP` verdict with `Domain` matching the uncontrolled domain(s) and `DetectedPattern: "ADHERENCE_GAP"`.
- [ ] **Step 3:** Persist the `ADHERENCE_GAP` card via the same card persistence path used by `INERTIA_DETECTED`, using the new `adherence_gap.yaml` template.

## Task P9-A.2: Build ADHERENCE_GAP YAML template

**Files:**
- Create: `kb-23-decision-cards/templates/inertia/adherence_gap.yaml`

- [ ] **Step 1:** `template_id: "dc-adherence-gap-v1"`, `node_id: "CROSS_NODE"`, `differential_id: "ADHERENCE_GAP"`, `mcu_gate_default: "SAFE"` (advisory only — do not modify therapy, recommend adherence counselling).
- [ ] **Step 2:** Clinician fragment: "Target unmet but engagement phenotype is DISENGAGED — likely adherence-driven, not therapy-failure-driven. Recommend structured adherence assessment before therapy escalation."
- [ ] **Step 3:** Patient fragment: explain adherence support resources.

## Task P9-A.3: Tests + metrics + commit

- [ ] **Step 1:** Test that a disengaged patient with unmet glycaemic target produces `ADHERENCE_GAP` instead of `INERTIA_DETECTED`.
- [ ] **Step 2:** Test that an engaged patient still produces `INERTIA_DETECTED` (regression guard).
- [ ] **Step 3:** Add `kb23_adherence_gap_detected_total{domain}` + `kb23_inertia_suppressed_by_adherence_total` metrics.
- [ ] **Step 4:** Commit: `feat(kb23): add adherence-exclusion branch to inertia detector (Phase 9 P9-A)`.

## P9-A Verification Questions

1. Does a patient with `EngagementStatus=DISENGAGED` + HbA1c 8.5 produce an `ADHERENCE_GAP` card instead of an `INERTIA_DETECTED` card? (yes / test)
2. Does a patient with `EngagementStatus=ENGAGED` + HbA1c 8.5 still produce an `INERTIA_DETECTED` card? (yes / test — regression guard)
3. Does the adherence check use a threshold of 0.4 on `EngagementComposite` when `EngagementStatus` is empty? (yes / test)

---

# Sub-project P9-B: Monitoring Engagement Detector

**Priority:** Fixes the Patient 18 blind spot. The system can't detect masked HTN without home BP data, but it CAN detect that data has stopped arriving.

## Task P9-B.1: Build MonitoringEngagementDetector

**Files:**
- Create: `kb-23-decision-cards/internal/services/monitoring_engagement_detector.go`
- Create: `kb-23-decision-cards/internal/services/monitoring_engagement_detector_test.go`

- [ ] **Step 1:** `DetectMonitoringLapse(patientID, monitorType, lastReadingAt, readingCountLast28d, now) → MonitoringLapseResult`. Pure function, no DB dependency.
- [ ] **Step 2:** Logic: if `readingCountLast28d >= 7` AND `now.Sub(lastReadingAt) > 14 * 24 * time.Hour` → lapsed. Otherwise → active.
- [ ] **Step 3:** Result struct: `{IsLapsed bool, DaysSinceLastReading int, PreviousMonitoringRate int, MonitorType string}`.

## Task P9-B.2: Build MONITORING_LAPSED YAML template

- [ ] **Step 1:** `template_id: "dc-monitoring-lapsed-v1"`, `mcu_gate_default: "SAFE"` (advisory).
- [ ] **Step 2:** Clinician fragment: "Patient was actively monitoring home BP ({{.PreviousRate}} readings/28d) and has stopped — last reading {{.DaysSince}} days ago. Clinical context may have changed unobserved."
- [ ] **Step 3:** Patient fragment: encourage return to monitoring.

## Task P9-B.3: Wire into weekly batch or separate batch job

- [ ] **Step 1:** Option A: add monitoring check to the existing inertia weekly batch (Sunday 03:00 UTC, same patient population). Option B: separate batch job with its own ShouldRun cadence. Decide during implementation — Option A is simpler unless the monitoring check needs a different patient population than the inertia population.
- [ ] **Step 2:** For each patient, fetch last home BP reading date + 28-day count from KB-20 (reuse FetchRenalStatus or add a dedicated endpoint). Call `DetectMonitoringLapse`. If lapsed → persist card.

## P9-B Verification Questions

1. Does a patient with 10 home BP readings in the last 28 days and no reading in the last 14 days produce a `MONITORING_LAPSED` card? (yes / test)
2. Does a patient who never monitored (0 readings in 28 days, 0 recently) NOT produce a card? (yes / test — can't lapse from a state you were never in)
3. Does a patient who is still actively monitoring (latest reading 2 days ago) NOT produce a card? (yes / test)

---

# Sub-project P9-C: InertiaVerdictHistory Postgres Store

**Priority:** Production hardening. The in-memory store resets on restart. Weekly batch means one post-deployment run without dampening. Persistent store eliminates this.

## Task P9-C.1: Build the Postgres implementation

- [ ] **Step 1:** New GORM model `InertiaVerdictRow` with `UNIQUE(patient_id, week_start_date)`.
- [ ] **Step 2:** `SaveVerdict` uses `ON CONFLICT (patient_id, week_start_date) DO UPDATE SET verdicts_json = ..., updated_at = NOW()`.
- [ ] **Step 3:** `FetchLatest` queries `WHERE patient_id = ? ORDER BY week_start_date DESC LIMIT 1`.
- [ ] **Step 4:** Implement `InertiaVerdictHistory` interface — swap is one line in `main.go`.

## P9-C Verification Questions

1. Does `SaveVerdict` upsert (not duplicate) when called twice for the same patient + week? (yes / test)
2. Does `FetchLatest` return the most recent week's verdict? (yes / test)
3. Does `main.go` swap `NewInMemoryInertiaHistory()` for the Postgres implementation? (yes / code review)

---

# Sub-project P9-D: CGM Period Report Deduplication

**Priority:** Production hardening. At-least-once Kafka delivery means duplicate events WILL arrive. Without dedup, the cgm_period_reports table grows linearly with duplicate deliveries.

## Task P9-D.1: Add unique constraint + upsert

- [ ] **Step 1:** Add `gorm:"uniqueIndex:idx_cgm_patient_period_end"` tags to `PatientID` + `PeriodEnd` on `CGMPeriodReport`.
- [ ] **Step 2:** Update `SavePeriodReport` to use `db.Clauses(clause.OnConflict{...}).Create(report)` for upsert.
- [ ] **Step 3:** Add `kb26_cgm_period_report_dedup_total` counter that increments on conflict.

## P9-D Verification Questions

1. Does a second insert with the same `(patient_id, period_end)` update the existing row instead of creating a duplicate? (yes / test)
2. Does the dedup counter increment on conflict? (yes / test)

---

# Sub-project P9-E: Bounded-Concurrency Fan-Out

**Priority:** Production hardening for large patient cohorts. Sequential iteration over 10K patients takes seconds today; with cross-service HTTP calls per patient it could take minutes.

## Task P9-E.1: Add worker pool to InertiaWeeklyBatch.Run

- [ ] **Step 1:** Channel-based semaphore: `sem := make(chan struct{}, concurrency)`. Default `concurrency = 4`.
- [ ] **Step 2:** Each patient evaluation sends into `sem` before starting, receives after completing. `sync.WaitGroup` for shutdown.
- [ ] **Step 3:** Same pattern for `RenalAnticipatoryBatch.Run`.

## P9-E Verification Questions

1. Does the batch complete faster with concurrency=4 vs concurrency=1 when evaluating ≥10 patients? (yes / benchmark or log)
2. Do per-patient errors still isolate correctly under concurrent execution? (yes / test)

---

## Execution Order

1. **P9-A** (~1-2h) — highest clinical leverage, fixes Patient 17
2. **P9-B** (~2-3h) — fixes Patient 18, second-highest clinical leverage
3. **P9-C** (~1-2h) — production hardening, eliminates dampening reset
4. **P9-D** (~30m) — production hardening, eliminates CGM duplicates
5. **P9-E** (~1h) — production hardening, enables large-cohort scaling

**Total: ~6-9 hours across 5 sub-projects in a single session.**

---

## What Phase 9 Session 1 does NOT cover (and why)

| Item | Why deferred |
|---|---|
| P9-F Frailty/deprescribing modifier | Needs clinical review for threshold definitions; touches MCU gate evaluation logic |
| P9-G FHIR Condition sync | Needs upstream FHIR infra changes |
| Gaps 9-13 (FHIR outbound, explainability, audit, circuit breaking, formulary) | Different workstream — compliance/platform, not clinical intelligence |
| Full AGP percentile overlays | Clinical value lower than P9-A/B; deferred since Phase 6 |
| Integration tests for GetCurrentMRI/GetMRIHistory | ~30 minutes but not in the Phase 7-8 critical path |

---

## Effort Summary

| Sub-project | Toolset | Upper bound | Expected actual | Files | New tests |
|---|---|---|---|---|---|
| P9-A Adherence-Exclusion | Go/KB | 0.5d | 1-2h | ~4 | ~4 |
| P9-B Monitoring Engagement | Go/KB | 1d | 2-3h | ~6 | ~6 |
| P9-C Verdict History Postgres | Go/KB | 0.5d | 1-2h | ~3 | ~4 |
| P9-D CGM Dedup | Go/KB | 2h | 30m | ~2 | ~2 |
| P9-E Bounded Concurrency | Go/KB | 0.5d | 1h | ~3 | ~2 |
