# Masked HTN Phase 5 — Stability Completion + Platform Extraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the two stability gaps surfaced by the Phase 4 verification review (raw-vs-stable persistence, medication-change override) and extract the Phase 2-4 batch + stability + event publication patterns from a single masked-HTN consumer into a platform abstraction proven by a second clinical consumer (therapeutic inertia weekly scan).

**Architecture:** Phase 5 is **not** a single coherent feature. It is four independently-shippable sub-projects (P5-1, P5-2, P5-3, P5-4) plus one follow-up (P5-5). Two of them (P5-1, the KB-26 half of P5-2) are **already complete** — committed in `c5f0a3fe`. This plan picks up where that commit left off: KB-20 population for P5-2, then the platform extraction in P5-3, then the conditional cluster mirror in P5-4, then the PK-aware windows in P5-5. Each sub-project has its own "depends on" line and its own verification questions.

**Pre-requisite:** Phase 4 + commit `c5f0a3fe` (`feat(kb26): stability engine raw-phenotype persistence + medication-change override (Phase 5 P5-1, P5-2)`) are present. KB-20, KB-23, and KB-26 build and test green.

**Tech Stack:** Go 1.25 (Gin, GORM, Zap, Prometheus), PostgreSQL 15.

---

## Status Snapshot — 2026-04-14 (end of Phase 5 execution session)

- [x] **P5-1 — Engine raw/stable persistence:** Committed in `c5f0a3fe`. `pkg/stability` has `Entry.Raw`, `History.RawMatchRate`, `Policy.MaxDwellOverrideRate`. Orchestrator captures `rawPhenotype` before dampening and persists both columns. Migration `008_bp_context_raw_phenotype.sql` ships the column. Production wired with `MaxDwellOverrideRate=0.7`. 8 engine tests + 2 orchestrator tests, green.
- [x] **P5-2 — Medication-change override (end-to-end):** KB-26 half in `c5f0a3fe`, KB-20 half in `dcaaee3d`. FHIR sync worker stamps `last_medication_change_at` on both ADD and UPDATE publish paths via `SyncWorker.stampMedicationChange`. Migration `006_patient_profile_med_change.sql`. `PatientProfile.LastMedicationChangeAt` auto-surfaces in the existing profile JSON via `PatientProfileResponse`. End-to-end orchestrator tests (`TestBPContextOrchestrator_P5_2_E2E_*`) pin that recent med changes bypass dwell and stale changes do not.
- [x] **P5-3 — Platform batch extraction:** Committed in `728ce373`. `BatchJob` interface gains `ShouldRun(ctx, now) bool`; `BatchScheduler.RunOnce` consults it before every `Run`. `BPContextDailyBatch.BatchHourUTC` field + `ShouldRun` gate fires only at the configured hour. New `InertiaWeeklyBatch` (second consumer) fires Mondays via `ShouldRun`; currently a heartbeat that lists active patients — real per-patient inertia scan deferred to Phase 6 because existing `DetectInertia`/`GenerateInertiaCards` are pure functions requiring a pre-assembled `InertiaDetectorInput` (multi-domain KB-20 data pulls not in P5 scope). Scheduler now ticks hourly (`StartLoop(1h)`); `computeNextScheduleInterval` dead code removed. Three-scenario integration test (Monday 02:00 / Tuesday 02:00 / Monday 09:00) proves the two consumers cohabit correctly.
- [~] **P5-4 — Cluster raw/stable parity:** **Deferred at discovery.** Exhaustive search for `ClusterAssignment`, `raw_cluster`, `stable_cluster`, `cluster_label`, `ClusterEngine`, `ClusterService`, or cluster stability history found **zero** matches in the KB codebase. The only `PhenotypeCluster` reference is a plain string field on `PatientProfile` — a column that stores a label, not a clustering engine. When phenotype clustering ships (Phase 6+), the parity work mirrors P5-1 verbatim: add `RawClusterLabel` field, capture-before-revert in the cluster service, wire `stability.Engine` with `MaxDwellOverrideRate=0.7`. Reference implementation for the future work: `bp_context_orchestrator.go` in commit `c5f0a3fe`.
- [x] **P5-5 — PK-aware override windows:** Committed in `ed93352a`. New `services/medication_steady_state.go` with a lookup table covering 14 antihypertensives across 5 classes (CCBs, ARBs, ACEi, beta blockers, diuretics). `SteadyStateWindow(class)` is case-insensitive, whitespace-tolerant, falls back to 7-day default for unknowns. `detectOverrideEvent` now calls the lookup instead of using a flat constant — metoprolol overrides for 2 days, amlodipine for 8. KB-20 `last_medication_change_class` column (migration `007`), worker populates both fields atomically, KB-26 client struct + detector read the class end-to-end. 4 lookup tests + 5 per-drug boundary tests + existing 5 override-detection tests all green.

**Phase 5 shipped: P5-1, P5-2 (end-to-end), P5-3, P5-5. Deferred: P5-4 (pending upstream clustering work).** 19 new tests across 5 commits (`c5f0a3fe`, `dcaaee3d`, `728ce373`, `ed93352a`, `e231dc5d`). All three affected services (KB-20, KB-23, KB-26) stay green.

---

## Locked Decisions

These are **not** open questions in this plan. They are fixed constraints derived from the Phase 4 verification review and the implementation choices already locked in by commit `c5f0a3fe`.

### Decision 1: KB-20 surfaces `last_medication_change_at` on the existing patient profile JSON, not via a new endpoint

KB-26 already pulls a `KB20PatientProfile` JSON object through `kb20_client.GET /api/v1/patient-profile/:id`. Adding the timestamp as one more nullable field on that response is the smallest possible shape change. **No new endpoint is created.** This avoids a second round-trip per classification and reuses KB-26's existing client code (the field is already declared on the Go struct as of `c5f0a3fe`).

### Decision 2: KB-20 derives the timestamp from the existing internal `EventMedicationChange` event, not from a new FHIR query

KB-20 already publishes `MedicationChangePayload` events from `internal/fhir/fhir_sync_worker.go` whenever a FHIR `MedicationRequest` lands. P5-2 hooks the same code path: when the worker decides to publish a `MEDICATION_CHANGE` event, it also writes the timestamp to a new `last_medication_change_at` column on the patient profile aggregate. **The signal source is reused, not duplicated.**

### Decision 3: The override window stays at the 7-day constant for P5-2; PK-aware windows are P5-5

P5-2 ships the simple constant. P5-5 replaces it with a drug-class lookup. This split keeps P5-2 a single-day task and lets P5-5 ship later without blocking the medication-change signal from reaching production. The `medicationChangeOverrideWindow` const in `bp_context_orchestrator.go` becomes a function call in P5-5 — no other call sites change.

### Decision 4: P5-3 adds `ShouldRun(ctx, now) bool` to the `BatchJob` interface, and the scheduler invokes it before every `Run`

The existing `BatchScheduler.RunOnce` calls `job.Run()` for every registered job on every tick. P5-3 inserts a `ShouldRun` gate so jobs with different cadences (daily vs weekly vs sensor-change) can co-exist. **The hook is owned by the job, not the scheduler** — the scheduler stays generic and never has to know the difference between cron-style triggers and event-driven ones. Existing callers of the masked-HTN batch job are unaffected: `BPContextDailyBatch` returns `true` from `ShouldRun` whenever `now.Hour() == BatchHourUTC`, preserving its current behaviour.

### Decision 5: The second batch consumer for P5-3 is the existing `inertia_card_generator`, not a new feature

The Phase 4 review proposed a second consumer to stress-test the abstraction. The cheapest real consumer is the inertia card generator at `kb-23-decision-cards/internal/services/inertia_card_generator.go`, which already exists but is only invoked synchronously from KB-23's HTTP handlers. P5-3 wraps it in a `WeeklyInertiaBatch` job that runs on Mondays, registers it with the scheduler in KB-26's `main.go`, and proves the scheduler honors a different cadence than BP context. **No new clinical logic is built.** If a third consumer is needed for the abstraction proof, that's a separate plan, not P5-3.

### Decision 6: P5-4 starts with a discovery task and is allowed to be a no-op

The Phase 4 review assumed `ClusterAssignmentRecord` exists. Discovery in this plan's first task confirms or refutes that assumption. If clustering code doesn't exist in the repo today, P5-4 becomes "documented deferral with the file path where the parity work will be added when clustering ships." **No speculative integration is built.** This avoids the failure mode of designing two parallel implementations that diverge later.

### Decision 7: Every sub-project ends with a Verification Questions block

This is the meta-recommendation that came out of the Phase 4 review. The implementer's completion report must answer each question with **yes / no / evidence** — not narrative. Phase 4 didn't have these and that's how the snapshot could say "stability shipped" with two gaps still open. The verification questions are an explicit contract.

---

## File Structure

### KB-20 changes (P5-2)

| Action | File | Sub-project |
|---|---|---|
| Modify | `kb-20-patient-profile/internal/models/patient_profile.go` | P5-2 — add `LastMedicationChangeAt *time.Time` field |
| Modify | `kb-20-patient-profile/internal/fhir/fhir_sync_worker.go` | P5-2 — write timestamp when publishing `EventMedicationChange` |
| Modify | `kb-20-patient-profile/internal/services/patient_service.go` | P5-2 — include the field in the profile-build query |
| Modify | `kb-20-patient-profile/internal/api/patient_handlers.go` | P5-2 — surface the field in the JSON response |
| Create | `kb-20-patient-profile/migrations/<next>_patient_profile_med_change.sql` | P5-2 — add `last_medication_change_at TIMESTAMPTZ` column |
| Modify | `kb-20-patient-profile/internal/services/patient_service_test.go` | P5-2 — assert the field is populated when a med change exists |

### KB-26 changes (P5-3, P5-5)

| Action | File | Sub-project |
|---|---|---|
| Modify | `kb-26-metabolic-digital-twin/internal/services/batch_scheduler.go` | P5-3 — add `ShouldRun` to `BatchJob` interface; gate `RunOnce` on it |
| Modify | `kb-26-metabolic-digital-twin/internal/services/batch_scheduler_test.go` | P5-3 — test that scheduler skips jobs whose `ShouldRun` returns false |
| Modify | `kb-26-metabolic-digital-twin/internal/services/bp_context_batch_job.go` | P5-3 — implement `ShouldRun` (returns `true` at the configured hour) |
| Create | `kb-26-metabolic-digital-twin/internal/services/inertia_weekly_batch.go` | P5-3 — wraps `inertia_card_generator` as a `BatchJob` running on Mondays |
| Create | `kb-26-metabolic-digital-twin/internal/services/inertia_weekly_batch_test.go` | P5-3 |
| Modify | `kb-26-metabolic-digital-twin/main.go` | P5-3 — register the new job |
| Modify | `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator.go` | P5-5 — replace const window with `lookupOverrideWindow(profile)` |
| Create | `kb-26-metabolic-digital-twin/internal/services/medication_steady_state.go` | P5-5 — drug-class → steady-state-duration lookup |
| Create | `kb-26-metabolic-digital-twin/internal/services/medication_steady_state_test.go` | P5-5 |

### KB-26 changes (P5-4, conditional on discovery)

| Action | File | Sub-project |
|---|---|---|
| Discovery | `find . -name '*cluster*' -type f` | P5-4 — confirm presence/absence of `ClusterAssignmentRecord` |
| Modify (if present) | `<discovered cluster model file>` | P5-4 — add `RawClusterLabel` field |
| Modify (if present) | `<discovered cluster persistence file>` | P5-4 — capture raw before stable revert |
| Modify (if present) | `<discovered cluster service file>` | P5-4 — wire `stability.Engine` with `MaxDwellOverrideRate` |

### KB-26 client changes (no new files for P5-2 KB-26 side — `c5f0a3fe` already shipped them)

### Documentation

| Action | File | Sub-project |
|---|---|---|
| Modify | `docs/runbooks/masked-htn-operations.md` | P5-2 — add a section on the medication-change override and the new field |
| Modify | `docs/superpowers/plans/2026-04-11-masked-htn-phase-4-quality-and-correctness.md` | Final task — mark P5 complete in the Phase 4 status snapshot |

**Total: 5 create, 8 modify across 2 services + 2 docs = 15 files (P5-4 may add 3 more if clustering exists).**

---

# Sub-project P5-2: KB-20 Population of `LastMedicationChangeAt`

**Priority:** Highest. Closes the override-detection blocker — without this, the engine has the override path but no signal source.
**Effort:** ~5 tasks, ~1 day.
**Depends on:** Commit `c5f0a3fe` (KB-26 detector + KB20PatientProfile field already in place).

This sub-project is the KB-20 server side of P5-2. The KB-26 client already reads `LastMedicationChangeAt` and the engine already honors the override; KB-20 just needs to populate the timestamp.

## Task P5-2.1: Read existing FHIR sync worker + patient profile model

**Files:**
- Read: `kb-20-patient-profile/internal/models/patient_profile.go`
- Read: `kb-20-patient-profile/internal/fhir/fhir_sync_worker.go`
- Read: `kb-20-patient-profile/internal/services/patient_service.go`
- Read: `kb-20-patient-profile/internal/api/patient_handlers.go`

- [ ] **Step 1: Read the four files end to end**

```bash
cat backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/patient_profile.go
cat backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/fhir/fhir_sync_worker.go
cat backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/patient_service.go
cat backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/patient_handlers.go
```

Note in your local task notes:
1. The struct that represents the patient profile (its Go name and GORM table name).
2. The existing line(s) where `EventMedicationChange` is published in `fhir_sync_worker.go` (around lines 208 and 219 per Phase 5 recon).
3. The function that builds the profile JSON returned by `GET /api/v1/patient-profile/:id`.
4. The JSON response struct that the handler returns to KB-26.

You will modify all four files in subsequent tasks. If the patient profile struct is split across multiple files (an aggregate, a JSON DTO, a GORM model), name all of them.

- [ ] **Step 2: Verify the JSON contract KB-26 expects**

```bash
grep -n "LastMedicationChangeAt\|last_medication_change_at" backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/clients/kb20_client.go
```

Expected: the field exists on `KB20PatientProfile` with `json:"last_medication_change_at,omitempty"` and Go type `*time.Time`. This is the contract KB-20 must produce. **Do not change KB-26's struct — KB-20 must match it.**

## Task P5-2.2: Database migration

**Files:**
- Create: `kb-20-patient-profile/migrations/<NN>_patient_profile_med_change.sql`

- [ ] **Step 1: Determine the next migration number**

```bash
ls backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/
```

Pick the next sequential number (e.g. if the latest is `007_*.sql`, your file is `008_patient_profile_med_change.sql`).

- [ ] **Step 2: Write the migration**

```sql
-- kb-20-patient-profile/migrations/<NN>_patient_profile_med_change.sql
-- Phase 5 P5-2: Add last_medication_change_at to the patient profile aggregate.
--
-- The KB-26 BP context orchestrator's stability engine bypasses dwell when a
-- recent medication change is detected on a patient. The signal source is
-- this column, populated by the FHIR sync worker whenever it publishes a
-- MEDICATION_CHANGE event. KB-26 reads the field via the existing patient
-- profile JSON endpoint — see KB20PatientProfile.LastMedicationChangeAt.
--
-- Backwards compat: column is nullable. KB-26 treats nil as "no override"
-- (safe default), so this migration can ship before the worker is updated.

ALTER TABLE <patient_profile_table>
    ADD COLUMN IF NOT EXISTS last_medication_change_at TIMESTAMPTZ;
```

Replace `<patient_profile_table>` with the GORM table name you confirmed in Task P5-2.1 Step 1 (likely `patient_profiles`). If you are unsure, run:

```bash
grep -n "TableName\|table_name\|gorm.*Model" backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/patient_profile.go
```

- [ ] **Step 3: Verify migration parses**

If the project has a migration runner test, run it. Otherwise:

```bash
psql -d kb_patient_profile_test -f backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/<NN>_patient_profile_med_change.sql
```

Expected: `ALTER TABLE` succeeds; column appears in `\d <patient_profile_table>`.

## Task P5-2.3: Add field to model + service write path

**Files:**
- Modify: `kb-20-patient-profile/internal/models/patient_profile.go`
- Modify: `kb-20-patient-profile/internal/services/patient_service.go`
- Modify: `kb-20-patient-profile/internal/services/patient_service_test.go`

- [ ] **Step 1: Write the failing test**

In `patient_service_test.go` append (adapt the helper names to whatever the existing tests use):

```go
func TestPatientService_PopulatesLastMedicationChangeAt(t *testing.T) {
    db := setupTestDB(t)
    svc := NewPatientService(db)

    patientID := "p-med-change"
    medChangeAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)

    // Seed: existing patient profile row with the new field set.
    profile := &models.PatientProfile{
        PatientID:              patientID,
        LastMedicationChangeAt: &medChangeAt,
    }
    if err := db.Create(profile).Error; err != nil {
        t.Fatalf("seed: %v", err)
    }

    got, err := svc.GetProfile(context.Background(), patientID)
    if err != nil {
        t.Fatalf("get: %v", err)
    }
    if got.LastMedicationChangeAt == nil {
        t.Fatal("expected LastMedicationChangeAt populated, got nil")
    }
    if !got.LastMedicationChangeAt.Equal(medChangeAt) {
        t.Errorf("expected %v, got %v", medChangeAt, got.LastMedicationChangeAt)
    }
}
```

- [ ] **Step 2: Verify test fails**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./internal/services/ -run TestPatientService_PopulatesLastMedicationChangeAt -v
```

Expected: compile error — `unknown field LastMedicationChangeAt in struct literal of type models.PatientProfile`.

- [ ] **Step 3: Add the field to the model**

In `patient_profile.go`, add to the `PatientProfile` struct:

```go
// Phase 5 P5-2: timestamp of the most recent antihypertensive medication
// change for this patient. Read by KB-26's stability engine to bypass the
// phenotype dwell window when a recent prescription/dose-change event would
// otherwise be suppressed. Nil = no signal recorded.
LastMedicationChangeAt *time.Time `gorm:"column:last_medication_change_at" json:"last_medication_change_at,omitempty"`
```

Add `"time"` to imports if not already present.

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/services/ -run TestPatientService_PopulatesLastMedicationChangeAt -v
```

Expected: PASS. If GORM's auto-select doesn't pick up the new column, ensure the service's query is `SELECT *` or includes the column explicitly.

- [ ] **Step 5: Commit (do not push)**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/patient_profile.go \
        backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/patient_service.go \
        backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/patient_service_test.go \
        backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/<NN>_patient_profile_med_change.sql
git commit -m "feat(kb20): add last_medication_change_at to patient profile (Phase 5 P5-2)

Schema + read path. The FHIR sync worker (next task) populates the
field whenever it publishes EventMedicationChange. Read by KB-26's
stability engine to bypass the phenotype dwell window — see
KB20PatientProfile.LastMedicationChangeAt and detectOverrideEvent in
bp_context_orchestrator.go (committed in c5f0a3fe)."
```

## Task P5-2.4: FHIR sync worker writes the timestamp

**Files:**
- Modify: `kb-20-patient-profile/internal/fhir/fhir_sync_worker.go`
- Modify: `kb-20-patient-profile/internal/fhir/fhir_sync_worker_test.go` (or whichever test file covers the worker)

- [ ] **Step 1: Locate the existing publish call sites**

```bash
grep -n "EventMedicationChange" backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/fhir/fhir_sync_worker.go
```

You should see two call sites (lines 208 and 219 per Phase 5 recon). Each publishes a `MedicationChangePayload`. The current code does **not** write the timestamp anywhere persistent; this task adds that.

- [ ] **Step 2: Write the failing test**

In the worker test file, append:

```go
func TestFHIRSyncWorker_WritesLastMedicationChangeAt_OnMedicationEvent(t *testing.T) {
    db := setupTestDB(t)
    bus := newStubEventBus()
    worker := NewFHIRSyncWorker(db, bus, zap.NewNop())

    patientID := "p-fhir-med"
    // Seed an existing patient profile.
    if err := db.Create(&models.PatientProfile{PatientID: patientID}).Error; err != nil {
        t.Fatal(err)
    }

    // Trigger the worker code path that publishes EventMedicationChange.
    // Use whatever helper the existing tests use to drive a sync — likely
    // syncPatientMedications or processBundle. Mirror the existing test
    // pattern in this file.
    worker.syncPatientMedications(context.Background(), patientID, sampleMedicationBundle)

    var got models.PatientProfile
    if err := db.Where("patient_id = ?", patientID).First(&got).Error; err != nil {
        t.Fatal(err)
    }
    if got.LastMedicationChangeAt == nil {
        t.Fatal("expected last_medication_change_at to be populated after med sync, got nil")
    }
    if time.Since(*got.LastMedicationChangeAt) > time.Minute {
        t.Errorf("expected timestamp ~now, got %v", got.LastMedicationChangeAt)
    }
}
```

If the existing worker test file uses different helper names (`setupTestDB`, `newStubEventBus`, `sampleMedicationBundle`), substitute them — read the existing tests in `fhir_sync_worker_test.go` first and reuse their fixtures verbatim.

- [ ] **Step 3: Verify test fails**

```bash
go test ./internal/fhir/ -run TestFHIRSyncWorker_WritesLastMedicationChangeAt_OnMedicationEvent -v
```

Expected: FAIL — the worker doesn't update the timestamp yet, so `LastMedicationChangeAt` is nil.

- [ ] **Step 4: Update both publish sites**

In `fhir_sync_worker.go` at the line where the worker calls `w.eventBus.Publish(models.EventMedicationChange, ...)` (around lines 208 and 219), insert immediately **before** the publish:

```go
// Phase 5 P5-2: stamp the patient profile with the change time so that
// KB-26's BP context stability engine can bypass the dwell window when
// a recent prescription/dose-change event would otherwise be suppressed.
now := time.Now().UTC()
if err := w.db.Model(&models.PatientProfile{}).
    Where("patient_id = ?", state.PatientID).
    Update("last_medication_change_at", now).Error; err != nil {
    w.log.Warn("failed to stamp last_medication_change_at",
        zap.String("patient_id", state.PatientID), zap.Error(err))
    // Non-fatal: the event still publishes, KB-26 just won't override.
}
```

Add `"time"` and `"go.uber.org/zap"` imports if not already present. Use `w.log` if the worker has a logger field; otherwise plumb one in.

Apply the same insertion at **both** publish sites (the line ~208 add/update path and the line ~219 remove path). A medication removal is also a "change" for override purposes — stopping a beta blocker can shift BP just as much as starting one.

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./internal/fhir/ -run TestFHIRSyncWorker_WritesLastMedicationChangeAt_OnMedicationEvent -v
```

Expected: PASS.

- [ ] **Step 6: Run the full KB-20 test suite for regression**

```bash
go test ./...
```

Expected: all previously-green packages still green.

- [ ] **Step 7: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/fhir/fhir_sync_worker.go \
        backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/fhir/fhir_sync_worker_test.go
git commit -m "feat(kb20): stamp last_medication_change_at on FHIR med sync (Phase 5 P5-2)

Worker write path. Both add/update and remove publish sites stamp the
timestamp before publishing EventMedicationChange. KB-26's stability
engine reads the field via the existing patient profile endpoint and
bypasses the phenotype dwell window for 7 days after the change."
```

## Task P5-2.5: Handler returns the field (verify, do not modify if struct is auto-marshalled)

**Files:**
- Verify only: `kb-20-patient-profile/internal/api/patient_handlers.go`
- Modify if needed: same file

- [ ] **Step 1: Inspect the handler's response shape**

```bash
grep -n "GetPatientProfile\|c.JSON.*profile\|GetProfile" backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/patient_handlers.go
```

Two cases:

1. **Handler returns `*models.PatientProfile` directly** (e.g. `c.JSON(200, profile)`). No code change needed — Go's JSON marshaller picks up the new field automatically via the `json:` tag added in Task P5-2.3.

2. **Handler returns a hand-rolled DTO** (e.g. `c.JSON(200, gin.H{"patient_id": ..., "sbp_14d_mean": ...})`). You must add the field explicitly: `"last_medication_change_at": profile.LastMedicationChangeAt`. The KB-26 client expects a `*time.Time` (`null` or RFC3339 string) — Go's default JSON marshaller produces both correctly.

- [ ] **Step 2: Write a handler-level test**

```go
func TestPatientHandlers_ProfileResponseIncludesLastMedicationChangeAt(t *testing.T) {
    db := setupTestDB(t)
    medChangeAt := time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC)
    db.Create(&models.PatientProfile{
        PatientID:              "p-handler",
        LastMedicationChangeAt: &medChangeAt,
    })
    server := NewServer(db, zap.NewNop())

    w := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/api/v1/patient-profile/p-handler", nil)
    server.Router.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
    }
    var got map[string]interface{}
    if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
        t.Fatal(err)
    }
    if got["last_medication_change_at"] == nil {
        t.Fatalf("response missing last_medication_change_at: %s", w.Body.String())
    }
}
```

- [ ] **Step 3: Run test**

```bash
go test ./internal/api/ -run TestPatientHandlers_ProfileResponseIncludesLastMedicationChangeAt -v
```

If the handler is auto-marshalling the struct (case 1 from Step 1), expect PASS immediately. If it's a hand-rolled DTO (case 2), expect FAIL — fix by adding the field as described in Step 1, then re-run.

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/patient_handlers.go \
        backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/patient_handlers_test.go
git commit -m "test(kb20): pin patient profile JSON contract for last_medication_change_at (P5-2)

Handler test ensures KB-26's KB20PatientProfile.LastMedicationChangeAt
field arrives populated whenever a med change has been recorded. Closes
the KB-20 half of P5-2 — the engine, KB-26 client, and orchestrator
sides all shipped in commit c5f0a3fe."
```

## Task P5-2.6: End-to-end integration test (KB-20 → KB-26 detector)

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator_p5_2_e2e_test.go`

- [ ] **Step 1: Write an integration test that runs KB-26's orchestrator against a stubbed KB-20 client returning a profile with a recent `LastMedicationChangeAt`**

```go
package services

import (
    "context"
    "testing"
    "time"

    "go.uber.org/zap"

    "kb-26-metabolic-digital-twin/internal/clients"
    "kb-26-metabolic-digital-twin/pkg/stability"
)

func TestBPContextOrchestrator_P5_2_E2E_MedChangeOverridesDwell(t *testing.T) {
    // Day 1: classify patient as MASKED_HTN (no med change yet).
    // Day 2: a med change occurred yesterday. The classifier flips to
    //        SUSTAINED_NORMOTENSION. Without the override the dwell would
    //        damp this transition. With the override it must accept.
    medChange := time.Now().UTC().Add(-24 * time.Hour)

    kb20 := &stubKB20Client{
        profile: &clients.KB20PatientProfile{
            PatientID:        "p-p5-2-e2e",
            SBP14dMean:       ptrFloat(148),
            DBP14dMean:       ptrFloat(92),
            ClinicSBPMean:    ptrFloat(128),
            ClinicDBPMean:    ptrFloat(78),
            ClinicReadings:   2,
            HomeReadings:     14,
            HomeDaysWithData: 7,
        },
    }
    kb21 := &stubKB21Client{}
    kb19 := &stubKB19Publisher{}
    policy := stability.Policy{
        MinDwell:             14 * 24 * time.Hour,
        MaxDwellOverrideRate: 0.7,
    }
    orch := newOrchestratorWithStabilityPolicy(t, kb20, kb21, kb19, policy)

    // Day 1: MASKED_HTN, no med change yet.
    if _, err := orch.Classify(context.Background(), "p-p5-2-e2e"); err != nil {
        t.Fatalf("day 1 classify: %v", err)
    }

    // Day 2: readings flip to normotension; med change recorded yesterday.
    kb20.profile.SBP14dMean = ptrFloat(120)
    kb20.profile.DBP14dMean = ptrFloat(75)
    kb20.profile.LastMedicationChangeAt = &medChange

    day2, err := orch.Classify(context.Background(), "p-p5-2-e2e")
    if err != nil {
        t.Fatalf("day 2 classify: %v", err)
    }

    // Without override the dwell would still hold MASKED_HTN. With override
    // the engine must accept SUSTAINED_NORMOTENSION.
    if string(day2.Phenotype) == "MASKED_HTN" {
        t.Errorf("expected override to bypass dwell and accept new phenotype, got %s", day2.Phenotype)
    }

    // And the snapshot's raw_phenotype should still match the new classifier output.
    saved, _ := orch.repo.FetchLatest("p-p5-2-e2e")
    if saved.RawPhenotype == "MASKED_HTN" {
        t.Errorf("expected raw_phenotype to reflect new classifier output, got %s", saved.RawPhenotype)
    }

    _ = zap.NewNop() // silence unused import if logger isn't passed
}
```

- [ ] **Step 2: Run the test**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./internal/services/ -run TestBPContextOrchestrator_P5_2_E2E_MedChangeOverridesDwell -v
```

Expected: PASS. The KB-26 detector + engine path is already in `c5f0a3fe`; this test is a regression guard that pins the end-to-end contract.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator_p5_2_e2e_test.go
git commit -m "test(kb26): P5-2 E2E — med change override bypasses dwell

Pins the contract: when KB20PatientProfile.LastMedicationChangeAt is
within the override window, the stability engine emits Override and
the orchestrator accepts the new phenotype regardless of dwell. Together
with the KB-20 population work in this branch, P5-2 is end-to-end real."
```

## P5-2 Verification Questions

The implementer's completion report **must** answer each with yes / no / evidence.

1. **Does KB-20's `GET /api/v1/patient-profile/:id` response include a `last_medication_change_at` field for a patient with a recorded medication change?**
   - Required evidence: `curl` against a running KB-20 instance (or a handler test) showing the field in the JSON.

2. **Does the FHIR sync worker stamp the timestamp on both add/update and remove medication paths?**
   - Required evidence: greps showing both call sites in `fhir_sync_worker.go` write the field.

3. **Does KB-26's `detectOverrideEvent` return `true` when the field is populated within the 7-day window?**
   - Required evidence: the existing unit tests in `bp_context_orchestrator_test.go` (`TestDetectOverrideEvent_*`) still pass.

4. **Does the engine's `Evaluate` return `DecisionOverride` end-to-end when the orchestrator runs against a profile with a recent med change?**
   - Required evidence: Task P5-2.6's integration test passes.

5. **Did you verify there are no regressions in the existing KB-20, KB-23, and KB-26 test suites?**
   - Required evidence: `go test ./...` output for all three services.

---

# Sub-project P5-3: Platform Batch Extraction

**Priority:** High. Proves the Phase 3 batch abstraction holds for more than one consumer before any third feature builds on it.
**Effort:** ~6 tasks, ~2 days.
**Depends on:** Phase 3 (BatchScheduler, BatchJob interface, BPContextDailyBatch). Existing inertia card generator at `kb-23-decision-cards/internal/services/inertia_card_generator.go`.

This sub-project adds a `ShouldRun` hook to the `BatchJob` interface so jobs with different cadences (daily vs weekly) can co-exist on one scheduler, then registers the existing inertia card generator as the second consumer.

## Task P5-3.1: Add `ShouldRun` to `BatchJob` interface (RED)

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/services/batch_scheduler_test.go`

- [ ] **Step 1: Read the current `BatchJob` interface and `RunOnce` flow**

```bash
grep -n "type BatchJob\|RunOnce\|func.*Register" backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/batch_scheduler.go
```

Note: the current `BatchJob` interface has `Name() string` and `Run(ctx) error`. P5-3 adds `ShouldRun(ctx, now) bool`.

- [ ] **Step 2: Write the failing test**

Append to `batch_scheduler_test.go`:

```go
type stubGatedJob struct {
    name        string
    shouldRun   bool
    runCalled   bool
}

func (s *stubGatedJob) Name() string                                  { return s.name }
func (s *stubGatedJob) ShouldRun(ctx context.Context, now time.Time) bool { return s.shouldRun }
func (s *stubGatedJob) Run(ctx context.Context) error                 { s.runCalled = true; return nil }

func TestBatchScheduler_RunOnce_SkipsJobsWhereShouldRunFalse(t *testing.T) {
    sched := NewBatchScheduler(zap.NewNop())
    runs := &stubGatedJob{name: "runs", shouldRun: true}
    skips := &stubGatedJob{name: "skips", shouldRun: false}
    sched.Register(runs)
    sched.Register(skips)

    if err := sched.RunOnce(context.Background()); err != nil {
        t.Fatalf("RunOnce: %v", err)
    }
    if !runs.runCalled {
        t.Error("expected ShouldRun=true job to execute")
    }
    if skips.runCalled {
        t.Error("expected ShouldRun=false job to be skipped")
    }
}
```

- [ ] **Step 3: Verify it fails**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./internal/services/ -run TestBatchScheduler_RunOnce_SkipsJobsWhereShouldRunFalse -v
```

Expected: compile error — `*stubGatedJob does not implement BatchJob (missing method ShouldRun)` (because `BatchJob` doesn't yet declare the method) **OR** a logic failure if you accidentally wrote it before adding the method to the interface. The compile error is the desired RED state.

## Task P5-3.2: Add `ShouldRun` to interface, implement in `BPContextDailyBatch` (GREEN)

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/services/batch_scheduler.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_batch_job.go`

- [ ] **Step 1: Add `ShouldRun` to the interface**

In `batch_scheduler.go`, modify the `BatchJob` interface:

```go
// BatchJob is the contract a job must implement to be hosted by the
// BatchScheduler. ShouldRun lets the scheduler skip jobs whose cadence
// hasn't elapsed (daily, weekly, sensor-change, etc.) without baking
// any cadence semantics into the scheduler itself — the scheduler
// stays generic and the job owns its own "is it time?" logic.
type BatchJob interface {
    Name() string
    ShouldRun(ctx context.Context, now time.Time) bool
    Run(ctx context.Context) error
}
```

In the same file, modify `RunOnce` to consult `ShouldRun` before invoking `Run`:

```go
func (s *BatchScheduler) RunOnce(ctx context.Context) error {
    s.runWg.Add(1)
    defer s.runWg.Done()

    s.mu.RLock()
    jobs := append([]BatchJob(nil), s.jobs...)
    s.mu.RUnlock()

    now := time.Now().UTC()
    for _, job := range jobs {
        if !job.ShouldRun(ctx, now) {
            s.log.Debug("batch job skipped", zap.String("job", job.Name()))
            continue
        }
        if err := job.Run(ctx); err != nil {
            s.log.Error("batch job failed", zap.String("job", job.Name()), zap.Error(err))
        }
    }
    return nil
}
```

(Adapt to whatever the existing `RunOnce` body looks like — preserve any logging, metrics, or error aggregation. The only behavioural change is the `ShouldRun` gate.)

- [ ] **Step 2: Implement `ShouldRun` on `BPContextDailyBatch`**

In `bp_context_batch_job.go`, add the method (preserve the existing `Run` method unchanged):

```go
// ShouldRun returns true when the current hour matches the configured
// BatchHourUTC. The scheduler can call this on every tick — only one tick
// per day will see the matching hour, which is the existing daily cadence.
func (j *BPContextDailyBatch) ShouldRun(ctx context.Context, now time.Time) bool {
    return now.Hour() == j.BatchHourUTC
}
```

If `BatchHourUTC` is not currently a field on `BPContextDailyBatch`, add it (it should already exist per Phase 3 P4 batch-hour work; if not, plumb it in from the constructor with a default of `2`).

- [ ] **Step 3: Verify the new test passes**

```bash
go test ./internal/services/ -run TestBatchScheduler_RunOnce_SkipsJobsWhereShouldRunFalse -v
```

Expected: PASS.

- [ ] **Step 4: Run the full batch scheduler test suite**

```bash
go test ./internal/services/ -run "BatchScheduler\|BPContextDailyBatch\|BPContextBatch" -v
```

Expected: all existing batch tests still pass. The most likely regression is a stub `BatchJob` implementation in another test file that doesn't yet implement `ShouldRun` — fix by adding `ShouldRun(ctx, now) bool { return true }` to those stubs.

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/batch_scheduler.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/batch_scheduler_test.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/bp_context_batch_job.go
git commit -m "feat(kb26): BatchJob.ShouldRun gate so jobs with different cadences co-exist (P5-3)

The scheduler now consults ShouldRun before every Run call. Jobs own
their own cadence — daily wall-clock for BPContextDailyBatch, weekly
for the inertia consumer (next task), sensor-change for any future
CGM consumer. The scheduler stays generic and never has to know the
difference between cron-style triggers and event-driven ones."
```

## Task P5-3.3: Wrap inertia card generator as a weekly `BatchJob`

**Files:**
- Read: `kb-23-decision-cards/internal/services/inertia_card_generator.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/inertia_weekly_batch.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/inertia_weekly_batch_test.go`

- [ ] **Step 1: Read the inertia card generator**

```bash
cat backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/inertia_card_generator.go
```

Note in your local task notes:
1. The generator's constructor signature.
2. The function that processes a single patient (e.g. `EvaluatePatient(ctx, patientID)` or similar).
3. Whether it depends on KB-23 internals (DB, etc.) — if so, the batch job will need an HTTP client, not a direct import.

**Important:** KB-26 cannot import KB-23 internal packages. The batch job must call inertia generation via an **HTTP client**, mirroring the `KB23Client` pattern from Phase 4 P9 (see `kb-26-metabolic-digital-twin/internal/clients/kb23_client.go`). If KB-23 doesn't yet expose an inertia-generation HTTP endpoint, **add one in this task** at `POST /api/v1/inertia/scan/:patientId` that wraps the generator.

- [ ] **Step 2: Add the HTTP endpoint to KB-23 if it doesn't exist**

If a route like `/api/v1/inertia/...` already exists, skip this step.

Otherwise, in `kb-23-decision-cards/internal/api/`:

Create `inertia_handlers.go`:

```go
package api

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

// handleInertiaScan triggers therapeutic inertia evaluation for one patient.
// Phase 5 P5-3: called by KB-26's weekly inertia batch job.
func (s *Server) handleInertiaScan(c *gin.Context) {
    patientID := c.Param("patientId")
    if patientID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "patient_id required"})
        return
    }
    if s.inertiaCardGenerator == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "inertia_generator_unavailable"})
        return
    }
    cards, err := s.inertiaCardGenerator.EvaluatePatient(c.Request.Context(), patientID)
    if err != nil {
        s.log.Error("inertia scan failed", zap.String("patient_id", patientID), zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "inertia_scan_failed"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"cards_generated": len(cards)})
}
```

Adapt `EvaluatePatient` to whatever the actual generator method is called. Register the route in `routes.go`:

```go
v1.POST("/inertia/scan/:patientId", s.handleInertiaScan)
```

Add `inertiaCardGenerator` field to the `Server` struct and initialise it in `InitServices`.

- [ ] **Step 3: Add the KB-23 client method to KB-26**

In `kb-26-metabolic-digital-twin/internal/clients/kb23_client.go`, add:

```go
// TriggerInertiaScan asks KB-23 to evaluate therapeutic inertia for one
// patient. Best-effort — failures are logged by the caller, not retried.
func (c *KB23Client) TriggerInertiaScan(ctx context.Context, patientID string) error {
    if patientID == "" {
        return fmt.Errorf("patient_id required")
    }
    url := fmt.Sprintf("%s/api/v1/inertia/scan/%s", c.baseURL, patientID)
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
    if err != nil {
        return fmt.Errorf("build KB-23 inertia request: %w", err)
    }
    resp, err := c.client.Do(req)
    if err != nil {
        c.log.Warn("KB-23 inertia trigger failed", zap.String("url", url), zap.Error(err))
        return fmt.Errorf("KB-23 POST: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 400 {
        respBody, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("KB-23 inertia returned status %d: %s", resp.StatusCode, string(respBody))
    }
    return nil
}
```

- [ ] **Step 4: Write the failing test for the weekly batch job**

Create `inertia_weekly_batch_test.go`:

```go
package services

import (
    "context"
    "testing"
    "time"

    "go.uber.org/zap"
)

type stubInertiaTrigger struct {
    calls []string
    err   error
}

func (s *stubInertiaTrigger) TriggerInertiaScan(ctx context.Context, patientID string) error {
    s.calls = append(s.calls, patientID)
    return s.err
}

func TestInertiaWeeklyBatch_ShouldRun_OnlyOnMonday(t *testing.T) {
    repo := &stubBPRepoForBatch{ids: []string{"p1"}}
    trigger := &stubInertiaTrigger{}
    job := NewInertiaWeeklyBatch(repo, trigger, zap.NewNop())

    // Monday → true
    monday := time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC) // 2026-04-13 is a Monday
    if !job.ShouldRun(context.Background(), monday) {
        t.Error("expected ShouldRun=true on Monday")
    }
    // Tuesday → false
    tuesday := time.Date(2026, 4, 14, 9, 0, 0, 0, time.UTC)
    if job.ShouldRun(context.Background(), tuesday) {
        t.Error("expected ShouldRun=false on Tuesday")
    }
}

func TestInertiaWeeklyBatch_Run_TriggersOncePerActivePatient(t *testing.T) {
    repo := &stubBPRepoForBatch{ids: []string{"p1", "p2", "p3"}}
    trigger := &stubInertiaTrigger{}
    job := NewInertiaWeeklyBatch(repo, trigger, zap.NewNop())

    if err := job.Run(context.Background()); err != nil {
        t.Fatalf("Run: %v", err)
    }
    if len(trigger.calls) != 3 {
        t.Errorf("expected 3 inertia trigger calls, got %d", len(trigger.calls))
    }
}
```

If `stubBPRepoForBatch` doesn't already exist, mirror it from the BP context batch test (it's just a struct with an `ids []string` field and a `ListActivePatientIDs(window)` method).

- [ ] **Step 5: Verify the test fails**

```bash
go test ./internal/services/ -run TestInertiaWeeklyBatch -v
```

Expected: compile error — `NewInertiaWeeklyBatch` and `InertiaWeeklyBatch` undefined.

- [ ] **Step 6: Implement `InertiaWeeklyBatch`**

Create `inertia_weekly_batch.go`:

```go
package services

import (
    "context"
    "time"

    "go.uber.org/zap"
)

// InertiaTrigger is the narrow interface the inertia batch job needs from
// the KB-23 client — defined here, not in the clients package, so tests
// can stub it without importing the real client.
type InertiaTrigger interface {
    TriggerInertiaScan(ctx context.Context, patientID string) error
}

// InertiaWeeklyBatch fans out therapeutic inertia evaluation across all
// active patients once per week. Phase 5 P5-3 — the second consumer of
// the BatchScheduler, used to prove the Phase 3 batch abstraction holds
// for cadences other than the daily BP context cadence.
type InertiaWeeklyBatch struct {
    repo    InertiaActivePatientLister
    trigger InertiaTrigger
    log     *zap.Logger
}

// InertiaActivePatientLister is the narrow repo dependency.
type InertiaActivePatientLister interface {
    ListActivePatientIDs(window time.Duration) ([]string, error)
}

// NewInertiaWeeklyBatch wires the dependencies.
func NewInertiaWeeklyBatch(repo InertiaActivePatientLister, trigger InertiaTrigger, log *zap.Logger) *InertiaWeeklyBatch {
    if log == nil {
        log = zap.NewNop()
    }
    return &InertiaWeeklyBatch{repo: repo, trigger: trigger, log: log}
}

// Name implements BatchJob.
func (j *InertiaWeeklyBatch) Name() string { return "inertia_weekly" }

// ShouldRun implements BatchJob — fires only on Mondays.
func (j *InertiaWeeklyBatch) ShouldRun(ctx context.Context, now time.Time) bool {
    return now.Weekday() == time.Monday
}

// Run implements BatchJob — fan-out across active patients.
func (j *InertiaWeeklyBatch) Run(ctx context.Context) error {
    ids, err := j.repo.ListActivePatientIDs(60 * 24 * time.Hour)
    if err != nil {
        return err
    }
    for _, id := range ids {
        if err := j.trigger.TriggerInertiaScan(ctx, id); err != nil {
            j.log.Warn("inertia trigger failed",
                zap.String("patient_id", id), zap.Error(err))
        }
    }
    j.log.Info("inertia weekly batch complete", zap.Int("patients", len(ids)))
    return nil
}
```

- [ ] **Step 7: Run the new tests**

```bash
go test ./internal/services/ -run TestInertiaWeeklyBatch -v
```

Expected: PASS for both.

- [ ] **Step 8: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/inertia_weekly_batch.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/inertia_weekly_batch_test.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/clients/kb23_client.go \
        backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/inertia_handlers.go \
        backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/routes.go
git commit -m "feat(kb26,kb23): InertiaWeeklyBatch — second BatchJob consumer (P5-3)

KB-23 exposes POST /api/v1/inertia/scan/:patientId wrapping the
existing inertia_card_generator. KB-26's KB23Client gains
TriggerInertiaScan. New InertiaWeeklyBatch implements the BatchJob
interface with Monday-only cadence. Stress-tests the Phase 3 batch
abstraction with two consumers of different cadence — proves that
ShouldRun owns 'is it time?' and the scheduler stays generic."
```

## Task P5-3.4: Register the new job in `main.go`

**Files:**
- Modify: `kb-26-metabolic-digital-twin/main.go`

- [ ] **Step 1: Wire the new job at startup**

In `main.go`, after the existing `bpBatchJob` registration:

```go
// Phase 5 P5-3: register the inertia weekly batch as the second
// consumer of the BatchScheduler, proving the abstraction holds for
// jobs with different cadences (BP context daily vs inertia weekly).
inertiaBatchJob := services.NewInertiaWeeklyBatch(bpContextRepo, kb23Client, logger)
batchScheduler.Register(inertiaBatchJob)
```

`bpContextRepo` already implements `ListActivePatientIDs` (Phase 3). `kb23Client` was wired in Phase 4 P9.

- [ ] **Step 2: Build to verify**

```bash
go build ./...
```

Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/main.go
git commit -m "feat(kb26): register InertiaWeeklyBatch with the scheduler at startup (P5-3)

Two batch consumers now share the scheduler: BPContextDailyBatch fires
at the configured hour, InertiaWeeklyBatch fires on Mondays. The
ShouldRun hook keeps them isolated."
```

## Task P5-3.5: Multi-job scheduler integration test

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/services/batch_scheduler_test.go`

- [ ] **Step 1: Add a test that registers two jobs with different `ShouldRun` and runs the scheduler at multiple wall-clock times**

```go
func TestBatchScheduler_TwoConsumers_FireOnDifferentCadences(t *testing.T) {
    sched := NewBatchScheduler(zap.NewNop())

    // Job A: fires when the hour is 2 (BP context)
    bpJob := &stubGatedJob{name: "bp_daily"}
    bpJob.shouldRunAt = func(now time.Time) bool { return now.Hour() == 2 }
    // Job B: fires on Mondays (inertia weekly)
    inertiaJob := &stubGatedJob{name: "inertia_weekly"}
    inertiaJob.shouldRunAt = func(now time.Time) bool { return now.Weekday() == time.Monday }

    sched.Register(bpJob)
    sched.Register(inertiaJob)

    // Scenario 1: Monday at 02:00 → both should fire.
    bpJob.runCalled, inertiaJob.runCalled = false, false
    bpJob.now, inertiaJob.now = mondayAt(2), mondayAt(2)
    _ = sched.RunOnce(context.Background())
    if !bpJob.runCalled {
        t.Error("BP job should run on Monday at 02:00")
    }
    if !inertiaJob.runCalled {
        t.Error("Inertia job should run on Monday at 02:00")
    }

    // Scenario 2: Tuesday at 02:00 → only BP should fire.
    bpJob.runCalled, inertiaJob.runCalled = false, false
    bpJob.now, inertiaJob.now = tuesdayAt(2), tuesdayAt(2)
    _ = sched.RunOnce(context.Background())
    if !bpJob.runCalled {
        t.Error("BP job should still run on Tuesday at 02:00")
    }
    if inertiaJob.runCalled {
        t.Error("Inertia job should NOT run on Tuesday")
    }

    // Scenario 3: Monday at 09:00 → only inertia should fire.
    bpJob.runCalled, inertiaJob.runCalled = false, false
    bpJob.now, inertiaJob.now = mondayAt(9), mondayAt(9)
    _ = sched.RunOnce(context.Background())
    if bpJob.runCalled {
        t.Error("BP job should NOT run at 09:00")
    }
    if !inertiaJob.runCalled {
        t.Error("Inertia job should still run on Monday at 09:00")
    }
}

func mondayAt(hour int) time.Time {
    return time.Date(2026, 4, 13, hour, 0, 0, 0, time.UTC) // 2026-04-13 is a Monday
}
func tuesdayAt(hour int) time.Time {
    return time.Date(2026, 4, 14, hour, 0, 0, 0, time.UTC)
}
```

For this test you'll need to enrich `stubGatedJob` from Task P5-3.1 with a `shouldRunAt func(time.Time) bool` field and a `now time.Time` field, and update `ShouldRun` to call the func with `s.now` (since `RunOnce` calls `ShouldRun` with the wall clock, not a stub time — for this test we need an injection point). An alternative is to plumb a clock through the scheduler; do whichever is simplest given the existing scheduler shape.

- [ ] **Step 2: Run**

```bash
go test ./internal/services/ -run TestBatchScheduler_TwoConsumers_FireOnDifferentCadences -v
```

Expected: PASS in all three scenarios.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/batch_scheduler_test.go
git commit -m "test(kb26): two-consumer scheduler integration — different cadences (P5-3)

Asserts the BatchScheduler hosts BP daily and inertia weekly jobs
without either knowing about the other, and that ShouldRun is the
sole arbiter of whether a given job fires on a given tick."
```

## P5-3 Verification Questions

1. **Does the `BatchJob` interface declare `ShouldRun(ctx, now) bool`?**
   - Required evidence: `grep` in `batch_scheduler.go`.

2. **Does `BatchScheduler.RunOnce` consult `ShouldRun` before invoking `Run` for each job?**
   - Required evidence: the `TestBatchScheduler_RunOnce_SkipsJobsWhereShouldRunFalse` test passes.

3. **Does `BPContextDailyBatch.ShouldRun` return `true` only when `now.Hour() == BatchHourUTC`?**
   - Required evidence: existing `BPContextDailyBatch` tests still pass; new test asserts the boundary.

4. **Does `InertiaWeeklyBatch.ShouldRun` return `true` only on Mondays?**
   - Required evidence: `TestInertiaWeeklyBatch_ShouldRun_OnlyOnMonday` passes.

5. **Does the scheduler correctly fire BP at 02:00 daily AND inertia at 02:00 Monday but neither at 02:00 Tuesday → BP only?**
   - Required evidence: `TestBatchScheduler_TwoConsumers_FireOnDifferentCadences` passes all three scenarios.

6. **Did the BatchJob abstraction need to change to support two consumers?**
   - Required evidence: a one-paragraph note in the completion report. If the answer is "no" the abstraction is solid; if "yes" the change should be documented because future consumers will want the same flexibility.

---

# Sub-project P5-4: Cluster raw/stable parity (conditional)

**Priority:** Low. Mirrors the P5-1 pattern in the phenotype-clustering subsystem **if it exists**.
**Effort:** 1 discovery task + (3 implementation tasks if applicable) = ~0.5 day to discover, ~1 day to implement if green-lighted.
**Depends on:** P5-1 (the BP context pattern is the reference implementation to mirror).

## Task P5-4.1: Discovery — does phenotype clustering exist?

**Files:**
- Investigation only.

- [ ] **Step 1: Search for clustering implementations**

```bash
grep -rln "ClusterAssignment\|cluster_label\|phenotype_cluster\|raw_cluster\|stable_cluster" \
    backend/shared-infrastructure/knowledge-base-services/
```

Three possible outcomes:

**Outcome A — No matches.** Phenotype clustering does not exist in the codebase. Skip P5-4.2/P5-4.3/P5-4.4 entirely. Add the following note to the completion report and stop:

> P5-4 deferred. Phenotype clustering does not exist in the repo as of `<commit-sha>`. When clustering ships, the parity work should mirror the BP context pattern: add `RawClusterLabel` field, persist before any stability revert, wire the `stability.Engine` with `MaxDwellOverrideRate=0.7`. Estimated effort once clustering exists: 1 day. Reference implementation: `bp_context_orchestrator.go` lines around the Phase 5 P5-1 commit `c5f0a3fe`.

**Outcome B — Matches exist but only in a plan/spec doc, not in code.** Same as Outcome A. The spec exists, the code doesn't.

**Outcome C — Matches in actual `.go` files under `internal/` or `pkg/`.** Clustering exists. Proceed to P5-4.2.

- [ ] **Step 2: Document findings**

Append to your task notes which outcome applies and the file paths discovered. The next tasks branch on this.

## Task P5-4.2: Add `RawClusterLabel` field to the cluster record (conditional on Outcome C)

**Files:**
- Modify: `<discovered cluster model file>`
- Modify: `<discovered cluster service file>`
- Create: `<service>/migrations/<NN>_cluster_raw_label.sql`

The exact files come from Task P5-4.1's discovery output. The implementation pattern mirrors P5-1 verbatim:

- [ ] **Step 1: Add `RawClusterLabel` field to the cluster model**

(Same pattern as `BPContextHistory.RawPhenotype` — nullable column, gorm tag, json tag.)

- [ ] **Step 2: Migration to add the column**

(Same pattern as `008_bp_context_raw_phenotype.sql`.)

- [ ] **Step 3: In the cluster service, capture `rawClusterLabel := result.Cluster` before any stability revert; persist both**

- [ ] **Step 4: Wire `stability.Engine` with `MaxDwellOverrideRate=0.7` in the cluster service constructor**

- [ ] **Step 5: Mirror the P5-1 tests:** `_RawEqualsStable_OnFirstAssignment`, `_RawDifferentFromStable_OnDampedTransition`

- [ ] **Step 6: Commit**

```bash
git commit -m "feat(<service>): cluster raw/stable parity, mirrors BP context P5-1 (Phase 5 P5-4)

Cluster history now persists both raw and stable cluster labels.
The stability engine is wired with MaxDwellOverrideRate=0.7 so the
dwell yields when the raw classifier consistently disagrees. Mirrors
the BP context pattern from commit c5f0a3fe."
```

## P5-4 Verification Questions

1. **Does phenotype clustering exist in the repo?** (yes / no)

If **no**: P5-4 is deferred and the verification block ends here.

If **yes**, also answer:

2. **Does the cluster model have both `Cluster` (stable) and `RawClusterLabel` fields?**
3. **Does the cluster service capture the raw label before any stability revert?**
4. **Are there tests that pin `raw == stable` on first assignment AND `raw != stable` on damped transition?**
5. **Did the existing cluster test suite stay green?**

---

# Sub-project P5-5: PK-aware override windows (follow-up)

**Priority:** Low. Polish on top of P5-2.
**Effort:** ~3 tasks, ~0.5 day.
**Depends on:** P5-2 complete, KB-20 surfacing the most recent drug class on the patient profile (a small KB-20 addition that this sub-project's first task adds).

The current `medicationChangeOverrideWindow` is a flat 7-day constant. Different antihypertensives reach steady state at different speeds — metoprolol 1-2 days, losartan 3-6 days, amlodipine 7-8 days. P5-5 replaces the constant with a per-drug-class lookup so the override window matches each drug's pharmacokinetics.

## Task P5-5.1: Add `LastMedicationChangeClass` field to KB-20 patient profile

**Files:**
- Modify: `kb-20-patient-profile/internal/models/patient_profile.go`
- Modify: `kb-20-patient-profile/internal/fhir/fhir_sync_worker.go`
- Modify: `kb-26-metabolic-digital-twin/internal/clients/kb20_client.go`

- [ ] **Step 1: Add the field on both ends**

KB-20 model:
```go
LastMedicationChangeClass string `gorm:"column:last_medication_change_class" json:"last_medication_change_class,omitempty"`
```

KB-26 client struct:
```go
LastMedicationChangeClass string `json:"last_medication_change_class,omitempty"`
```

- [ ] **Step 2: Worker writes the drug class alongside the timestamp**

In the same `Update(...)` call from P5-2 Task 4, add the drug class:

```go
Updates(map[string]interface{}{
    "last_medication_change_at":    now,
    "last_medication_change_class": drugClass, // from the FHIR MedicationRequest
})
```

- [ ] **Step 3: Migration**

```sql
ALTER TABLE patient_profiles
    ADD COLUMN IF NOT EXISTS last_medication_change_class VARCHAR(40);
```

- [ ] **Step 4: Test + commit**

## Task P5-5.2: Drug-class to steady-state lookup table

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/medication_steady_state.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/medication_steady_state_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestSteadyStateWindow_KnownDrugs(t *testing.T) {
    cases := []struct {
        class    string
        expected time.Duration
    }{
        {"AMLODIPINE", 8 * 24 * time.Hour},
        {"LOSARTAN", 6 * 24 * time.Hour},
        {"METOPROLOL", 2 * 24 * time.Hour},
        {"LISINOPRIL", 5 * 24 * time.Hour},
        {"UNKNOWN_DRUG", 7 * 24 * time.Hour}, // fallback to default
    }
    for _, tc := range cases {
        if got := SteadyStateWindow(tc.class); got != tc.expected {
            t.Errorf("%s: expected %v, got %v", tc.class, tc.expected, got)
        }
    }
}
```

- [ ] **Step 2: Implement the lookup**

```go
package services

import (
    "strings"
    "time"
)

// drugClassSteadyState maps an antihypertensive drug class to its
// pharmacologic time-to-steady-state — the duration after a dose change
// during which the BP response is still moving and the stability dwell
// should yield to the new clinical reality.
//
// Sources: ESH 2023, ISH 2020 product monographs, FDA labels.
// Defaults to 7 days for any class not in this table — the safe middle
// ground used by Phase 5 P5-2's flat constant.
var drugClassSteadyState = map[string]time.Duration{
    "AMLODIPINE":  8 * 24 * time.Hour,
    "FELODIPINE":  7 * 24 * time.Hour,
    "NIFEDIPINE":  3 * 24 * time.Hour,
    "LOSARTAN":    6 * 24 * time.Hour,
    "VALSARTAN":   4 * 24 * time.Hour,
    "TELMISARTAN": 7 * 24 * time.Hour,
    "LISINOPRIL":  5 * 24 * time.Hour,
    "RAMIPRIL":    5 * 24 * time.Hour,
    "ENALAPRIL":   4 * 24 * time.Hour,
    "METOPROLOL":  2 * 24 * time.Hour,
    "ATENOLOL":    2 * 24 * time.Hour,
    "BISOPROLOL":  3 * 24 * time.Hour,
    "HCTZ":        7 * 24 * time.Hour,
    "INDAPAMIDE":  7 * 24 * time.Hour,
}

// defaultSteadyStateWindow is used when the drug class is unknown or empty.
const defaultSteadyStateWindow = 7 * 24 * time.Hour

// SteadyStateWindow returns the pharmacologic time-to-steady-state for a
// drug class. Unknown classes return defaultSteadyStateWindow.
func SteadyStateWindow(drugClass string) time.Duration {
    key := strings.ToUpper(strings.TrimSpace(drugClass))
    if d, ok := drugClassSteadyState[key]; ok {
        return d
    }
    return defaultSteadyStateWindow
}
```

- [ ] **Step 3: Run + commit**

## Task P5-5.3: Replace the constant in `detectOverrideEvent`

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator_test.go`

- [ ] **Step 1: Update the detector**

```go
func detectOverrideEvent(profile *clients.KB20PatientProfile) bool {
    if profile == nil || profile.LastMedicationChangeAt == nil {
        return false
    }
    window := SteadyStateWindow(profile.LastMedicationChangeClass)
    elapsed := time.Since(*profile.LastMedicationChangeAt)
    return elapsed >= 0 && elapsed <= window
}
```

The `medicationChangeOverrideWindow` constant becomes dead code — delete it.

- [ ] **Step 2: Add tests covering the per-drug branches**

```go
func TestDetectOverrideEvent_AmlodipineUsesEightDayWindow(t *testing.T) {
    seven := time.Now().UTC().Add(-8*24*time.Hour + time.Hour) // 7d23h ago
    profile := &clients.KB20PatientProfile{
        LastMedicationChangeAt:    &seven,
        LastMedicationChangeClass: "AMLODIPINE",
    }
    if !detectOverrideEvent(profile) {
        t.Error("expected amlodipine override true at 7d23h ago (within 8d window)")
    }
}

func TestDetectOverrideEvent_MetoprololUsesTwoDayWindow(t *testing.T) {
    threeDays := time.Now().UTC().Add(-3 * 24 * time.Hour)
    profile := &clients.KB20PatientProfile{
        LastMedicationChangeAt:    &threeDays,
        LastMedicationChangeClass: "METOPROLOL",
    }
    if detectOverrideEvent(profile) {
        t.Error("expected metoprolol override false at 3d ago (outside 2d window)")
    }
}
```

- [ ] **Step 3: Run + commit**

## P5-5 Verification Questions

1. **Does `SteadyStateWindow` return the correct duration for AMLODIPINE, LOSARTAN, METOPROLOL, and LISINOPRIL?**
2. **Does `SteadyStateWindow` fall back to 7 days for unknown classes?**
3. **Does `detectOverrideEvent` consult `SteadyStateWindow` instead of the dead constant?**
4. **Are there per-drug tests that pin the boundary behaviour for at least one fast drug (metoprolol) and one slow drug (amlodipine)?**

---

# Tasks Deferred from Phase 5

| # | Item | Reason for deferral |
|---|---|---|
| **CGM 14-day batch consumer** | The Phase 4 review proposed CGM as a third stress-test of the abstraction. Deferred to Phase 6 because (a) two consumers is enough to prove the cadence-isolation property, and (b) CGM batches are sensor-event-driven, not wall-clock — they need a separate event-bus signal that is its own brainstorm. |
| **Cluster stability if clustering ships in Phase 5+** | P5-4's discovery task may find clustering doesn't exist. If it ships in a later phase, the parity work follows P5-1's pattern verbatim — no new design needed. |
| **Override duration metrics** | A Prometheus histogram of "how often did the override fire / how often did it fire on a true future transition" would let us validate the 0.7 threshold empirically. Documented but not in scope for Phase 5; lives in the operations runbook as a Phase 6 ask. |

---

## Plan Summary

| Sub-project | Tasks | New tests | Outcome |
|---|---|---|---|
| **P5-1** Engine raw/stable persistence | ✅ Done in `c5f0a3fe` | 8 + 2 = 10 | Dwell becomes a soft block that yields to consistent disagreement |
| **P5-2 KB-26 side** Override detector | ✅ Done in `c5f0a3fe` | 5 | Engine path ready, signal source pending |
| **P5-2 KB-20 side** Population | 5 | ~4 | Med-change signal flows end-to-end |
| **P5-3** Platform batch extraction | 5 | ~5 | BatchJob abstraction proven with 2 real consumers |
| **P5-4** Cluster parity | 1-4 (conditional) | 0-3 | Mirrors P5-1 if clustering exists; deferred otherwise |
| **P5-5** PK-aware windows | 3 | ~5 | Per-drug-class override windows replace the flat 7-day const |
| **Total remaining** | **14-17** | **~17-20** | Stability gaps closed, batch abstraction extracted |

## What Phase 5 Delivers (after every sub-project ships)

- KB-26's stability engine yields the dwell when the raw classifier output has been consistently agreeing with the proposed transition (P5-1, ✅ done)
- KB-26's stability engine bypasses the dwell when KB-20 reports a recent medication change, with a per-drug-class window (P5-2 + P5-5)
- KB-20's FHIR sync worker stamps `last_medication_change_at` and `last_medication_change_class` on the patient profile whenever a medication event lands (P5-2)
- The BatchScheduler hosts at least two clinical consumers with different cadences via a shared `ShouldRun` hook (P5-3)
- The therapeutic inertia card generator runs as a weekly batch instead of being invoked only on demand (P5-3)
- Phenotype clustering, if it exists, mirrors the BP context raw/stable persistence pattern (P5-4)

## What Phase 5 Does NOT Deliver

- **CGM 14-day batch consumer.** Deferred to Phase 6 — sensor-event cadences need their own design pass.
- **Override-effectiveness metrics.** Deferred to Phase 6 with a Prometheus histogram proposal in the runbook.
- **Cluster integration if clustering doesn't exist yet.** Documented in P5-4 as a deferred-pending-discovery sub-project.

## Execution Order Recommendation

The sub-projects can be implemented in any order, but the natural sequence is:

1. **P5-2 (KB-20 side)** — closes the stability override loop, biggest clinical correctness win remaining
2. **P5-3** — proves the batch abstraction before more consumers pile up on it
3. **P5-5** — polish on top of P5-2; small but high-quality
4. **P5-4** — discovery first; may be a no-op

P5-2 and P5-5 together represent "stability engine clinically complete." P5-3 represents "platform ready for next clinical feature." P5-4 represents "stability pattern uniformly applied."
