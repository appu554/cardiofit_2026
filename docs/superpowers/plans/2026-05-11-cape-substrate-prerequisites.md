# CAPE Substrate Prerequisites — Pre-pilot Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Build the five substrate primitives that `kb-33-triage-engine` (CAPE engine — Roadmap Step 5) imports on day one. Without them, kb-33 cannot scaffold cleanly. Three of the five also unblock immediate craft-engine concerns (notably the previously-vacuous citation lookup on `/v1/explain`).

**Why a separate plan:** These primitives don't belong inside any one CAPE part — they're cross-cutting types, vocabularies, and IDL surfaces that multiple downstream services share. Bundling them into the kb-33 plan would conflate substrate authoring with engine implementation.

**Source documents:**
- `docs/superpowers/plans/CAPE_Implementation_Guidelines_v1_1.md` (1186 lines) — definitive for Primitives B, C
- `docs/superpowers/plans/CAPE_v1_1_Architectural_Commitment_Addendum.md` (623 lines) — definitive for Primitives D, E
- `internal/api/explain_handlers.go:56–64,150–159` (kb-32) — defines the gap that Primitive A closes; CAPE docs are silent on it

**Tech stack:** Go, Protobuf, Postgres. Depends on `feat/phase-2-completion` merged to `main` (commit `8b79947d` — done 2026-05-11).

**Branch:** `feat/cape-substrate-prerequisites` off `main`. One commit per task. Push to origin between tasks.

---

## Architecture

Each primitive lands in its canonical home:

| Primitive | Home package | Reason |
|---|---|---|
| A — `Metadata.RecommendationID` | `shared/v2_substrate/ethics/decision_metadata/` | Field added to existing shared struct |
| B — Failed Intervention History | `shared/v2_substrate/clinical/failed_interventions/` (new) | Shared between kb-32 (writer) and kb-33 (reader) |
| C — PRN administration velocity | `shared/v2_substrate/clinical/prn_velocity/` (new) | Shared signal primitive; CQL definition lives alongside Go computation |
| D — Instability Chronology | `shared/v2_substrate/clinical/instability_chronology/` (new) | Types + composition primitives only; computation deferred to kb-33 |
| E — ObservationLayer proto IDL | `proto/v1/observation_layer.proto` (new) | Wire-format contract; no server code yet |

---

## File Structure

**Modified files (Primitive A):**
- `shared/v2_substrate/ethics/decision_metadata/recorder.go` — add `RecommendationID uuid.UUID` field
- `shared/v2_substrate/ethics/decision_metadata/recorder_test.go` — test the field round-trips
- `kb-32-recommendation-craft/internal/api/explain_handlers.go` — wire the now-possible `ListCitations` lookup, remove the TODO block
- All call sites that construct `decision_metadata.Metadata` — populate the new field (one per writer site)

**New files (Primitives B/C/D):**
- `shared/v2_substrate/clinical/failed_interventions/types.go` + `_test.go`
- `shared/v2_substrate/clinical/failed_interventions/store.go` — `Store` interface + `InMemoryStore` + `PostgresStore`
- `shared/migrations/047_failed_interventions.sql` + rollback (note: shared dir, distinct from kb-32-local 047)
- `shared/v2_substrate/clinical/prn_velocity/types.go` + `_test.go`
- `shared/v2_substrate/clinical/prn_velocity/compute.go` — Go implementation of the CQL spec
- `shared/v2_substrate/clinical/prn_velocity/cql/prn_escalation_velocity.cql` — the CQL definition (verbatim from CAPE Part 3.2)
- `shared/v2_substrate/clinical/instability_chronology/types.go` + `_test.go`

**New files (Primitive E):**
- `proto/v1/observation_layer.proto` — service definition + all 11 RPC + message types
- `proto/v1/common.proto` — shared message types (ResidentID, TimeWindow, Severity, etc.) if not already defined
- `proto/buf.yaml` + `proto/buf.gen.yaml` — buf configuration (if buf isn't already configured at the repo level)

---

## Task ordering rationale

Tasks A → E can run mostly in parallel after Task A. The recommended sequence balances unblocking value vs. blast radius:

1. **A first** — smallest diff, unblocks `/v1/explain` immediately, validates the shared-package edit pattern.
2. **B second** — defines the type that override-capture will write to (kb-32 connects in the same commit).
3. **C third** — pure addition, no consumers wired yet.
4. **D fourth** — types only; computation deferred to kb-33.
5. **E last** — proto IDL imports types from B, C, D so it benefits from going last.

---

### Task A: `decision_metadata.Metadata.RecommendationID` field

**Files:**
- Modify: `shared/v2_substrate/ethics/decision_metadata/recorder.go`
- Modify: `shared/v2_substrate/ethics/decision_metadata/recorder_test.go`
- Modify: `kb-32-recommendation-craft/internal/api/explain_handlers.go` (remove TODO at lines 60–64 + 150–159; wire the real `ListCitations` call)
- Modify: ALL call sites that construct `decision_metadata.Metadata` — populate `RecommendationID` (one per writer). Find via `grep -rn "decision_metadata.Metadata{"`.

**The current gap (from explain_handlers.go:56–64 verbatim):**

> "Citations: the v2_substrate decision_metadata.Metadata struct does not carry a RecommendationID. Because citations.Registry.ListCitations is keyed by recommendation ID, this endpoint cannot currently surface fire-time citations. Adding the RecommendationID field on Metadata will let us wire the lookup without ..."

- [ ] **Step 1: Add field to `Metadata` struct**
  ```go
  type Metadata struct {
      DecisionID       uuid.UUID
      RecommendationID uuid.UUID  // NEW: links decision to the kb-32 recommendation it produced
      // ... existing fields ...
  }
  ```
  Zero value (`uuid.Nil`) means "no associated recommendation" — `/v1/explain` callers must handle this case.

- [ ] **Step 2: Add table-driven tests for round-trip serialization (recorder_test.go)**

- [ ] **Step 3: Find ALL writers**
  ```bash
  grep -rn "decision_metadata.Metadata{" backend/
  ```
  Populate `RecommendationID` at every construction site that has access to one. Sites without access leave it at `uuid.Nil` and document why in a one-line comment.

- [ ] **Step 4: Wire `/v1/explain` lookup**
  In `kb-32-recommendation-craft/internal/api/explain_handlers.go`, replace the TODO block (lines 56–64 + 150–159) with:
  ```go
  var cites []citations.RecommendationCitation
  if md.RecommendationID != uuid.Nil {
      cites, err = h.citationReg.ListCitations(ctx, md.RecommendationID.String())
      if err != nil {
          log.Printf("explain: citations.ListCitations failed for decision_id=%s: %v", decisionID, err)
          // Don't fail the whole explain response — surface citations as empty.
      }
  }
  // Attach cites to the response.
  ```

- [ ] **Step 5: Run `go test ./...` across kb-32 AND every other service that imports `decision_metadata`. Verify no regressions.**

- [ ] **Step 6: Commit**
  ```
  feat(substrate): add RecommendationID to decision_metadata.Metadata; wire /v1/explain citation lookup
  ```

---

### Task B: Failed Intervention History substrate

**Files:**
- Create: `shared/v2_substrate/clinical/failed_interventions/types.go` + `_test.go`
- Create: `shared/v2_substrate/clinical/failed_interventions/store.go` + `_test.go`
- Create: `shared/migrations/047_failed_interventions.sql` + rollback
- Modify: `kb-32-recommendation-craft/internal/overrides/store.go` — after a successful override capture with outcome ∈ {`reversed_due_to_*`, `goals_of_care_aligned`, `frailty_consideration`}, write a `FailedInterventionRecord` via the new store.

**Spec (verbatim CAPE Guidelines lines 632–660):**

```go
type FailedInterventionRecord struct {
    ResidentID         uuid.UUID
    InterventionType   string  // e.g., "antipsychotic_deprescribing"
    AttemptDate        time.Time
    Outcome            string  // "reversed_due_to_BPSD_recurrence", etc.
    DocumentedReason   string
    RetryEligibleDate  time.Time  // typically AttemptDate + 12 months
    DocumentedBy       PharmacistID  // uuid.UUID
}
```

CAPE §4.3 (lines 421–425) — Failed Intervention History is a **veto factor** in Layer 4 (intervention opportunity assessment). CAPE §7.3 (lines 654–657) — `IsVetoActive(residentID, interventionType)` checks history; returns true when an entry exists with `RetryEligibleDate > now()`.

- [ ] **Step 1: Define struct + Outcome enum**
  Outcomes enumerated as Go constants. `IsRetryEligible() bool` helper. `IsVetoActive(records []FailedInterventionRecord, interventionType string) bool` free function.

- [ ] **Step 2: Define `Store` interface + `InMemoryStore` + `PostgresStore`**
  `Record(ctx, FailedInterventionRecord) error`, `ListByResident(ctx, residentID uuid.UUID) ([]Record, error)`, `IsVetoActive(ctx, residentID uuid.UUID, interventionType string) (bool, error)` (delegates to free function on top of `ListByResident`).

- [ ] **Step 3: Migration 047 (SHARED migrations dir; collisions: none — kb-32-local 047 is `prescriber_framing_optout`, different dir)**
  Table `failed_intervention_records` with the 7 fields. Composite index on `(resident_id, retry_eligible_date)` for fast `IsVetoActive` queries.

- [ ] **Step 4: Wire kb-32 override store to write FailedInterventionRecord**
  In `internal/overrides/store.go` `Create()`, after the INSERT, if `OverrideReason.Outcome` is in the failure-outcome set, also write a `FailedInterventionRecord`. Map `OverrideReason.RecommendationID` → resident_id (look up via recommendations table), `RuleID` → `InterventionType` via a small classifier (e.g., `STOP_PSYCHOTROPIC_*` → `antipsychotic_deprescribing`).
  
  **Open question:** the rule-id → intervention-type classifier is currently underspecified. Implementer should define it as a small map in `failed_interventions/classifier.go` with the 5–8 most-common mappings; unmatched rules write `InterventionType=""` and the reader treats empty as "unclassified, no veto".

- [ ] **Step 5: Tests** — unit tests on `IsVetoActive` truth table; integration test (DSN-skipping) for PostgresStore round-trip; integration test that an override capture writes the record.

- [ ] **Step 6: Commit**
  ```
  feat(substrate): Failed Intervention History — types, stores, kb-32 override-capture hook
  ```

---

### Task C: PRN Administration Velocity primitive

**Files:**
- Create: `shared/v2_substrate/clinical/prn_velocity/types.go` + `_test.go`
- Create: `shared/v2_substrate/clinical/prn_velocity/compute.go` + `_test.go`
- Create: `shared/v2_substrate/clinical/prn_velocity/cql/prn_escalation_velocity.cql`

**Spec (verbatim CAPE Guidelines lines 271–290):**

```cql
define "PRN Escalation Velocity":
  let recent_30d = count of PRN administrations of class X in last 30 days
  let baseline_90d = average of PRN administrations of class X per 30 days,
                     prior 90 days
  let velocity_ratio = recent_30d / baseline_90d
  in
    case
      when velocity_ratio > 4.0 then 5  // 400%+ increase
      when velocity_ratio > 2.5 then 4  // 250%+ increase
      when velocity_ratio > 1.5 then 3  // 150%+ increase
      when velocity_ratio > 1.0 then 2  // any increase
      else 1
    end
```

Three signal classes (CAPE Guidelines lines 569–571): `PRN_benzodiazepine_escalation_velocity`, `PRN_antipsychotic_escalation_velocity`, `PRN_analgesic_escalation_velocity`.

- [ ] **Step 1: Define types**
  ```go
  type PRNClass string
  const (
      PRNBenzodiazepine PRNClass = "benzodiazepine"
      PRNAntipsychotic  PRNClass = "antipsychotic"
      PRNAnalgesic      PRNClass = "analgesic"
  )

  type VelocityResult struct {
      Class         PRNClass
      Recent30dCount int
      Baseline90dAvg float64
      VelocityRatio float64
      Severity      int  // 1..5
  }
  ```

- [ ] **Step 2: Implement `Compute(administrations []Administration, class PRNClass, now time.Time) VelocityResult`**
  Pure function. No DB access — caller supplies the administration slice (sourced from wherever PRN administrations are persisted). Edge cases: `baseline_90d == 0` → `VelocityRatio = +Inf` if `recent_30d > 0`, treated as Severity 5; both zero → Severity 1.

- [ ] **Step 3: Persist the CQL definition verbatim**
  `cql/prn_escalation_velocity.cql` — the exact CQL block above, with a header comment pointing back to CAPE Guidelines lines 271–290 as the source of truth. This file is for clinical informatics review; not executed by the Go code (we don't have a CQL runtime here yet).

- [ ] **Step 4: Tests** — full coverage of the 5 severity buckets + the divide-by-zero edge case + the "no administrations" edge case.

- [ ] **Step 5: Commit**
  ```
  feat(substrate): PRN administration velocity — three signal classes, Go compute + CQL reference
  ```

---

### Task D: Instability Chronology primitive (types only)

**Files:**
- Create: `shared/v2_substrate/clinical/instability_chronology/types.go` + `_test.go`

**Scope clarification:** CAPE Addendum §3 specifies Instability Chronology as **computed by the CAPE engine** (i.e., kb-33). This task ships **types only** — the structs that kb-33 will populate, plus serialization tests. No computation logic.

**Spec (verbatim Addendum lines 235–260):**

```go
type InstabilityChronology struct {
    ResidentID    ResidentID  // uuid.UUID
    TimeWindow    TimeWindow
    Events        []ChronologyEvent
    Patterns      []TemporalPattern
    Severity      Severity
    AudienceAdaptations map[AudienceClass]ChronologyRendering
}

type ChronologyEvent struct {
    Timestamp        time.Time
    EventType        string  // e.g., "medication_change", "intake_decline"
    PrimitiveType    InstabilityPrimitive  // canonical vocabulary
    Severity         Severity
    Description      string  // factual, audience-neutral
    SubstrateRefs    []SubstrateReference
    SuspectedCauses  []string
    RelatedEvents    []EventID  // uuid.UUID
}
```

- [ ] **Step 1: Define all referenced types**
  `TimeWindow`, `TemporalPattern`, `Severity`, `AudienceClass`, `ChronologyRendering`, `InstabilityPrimitive`, `SubstrateReference`, `EventID` — declare them all in `types.go`. Where the spec leaves the shape underspecified (e.g., `TemporalPattern` is named but not fully fielded), make the minimal viable struct and add a doc comment pointing to the spec line range.

- [ ] **Step 2: Define `InstabilityPrimitive` canonical vocabulary**
  CAPE Addendum lists examples: `medication_change`, `intake_decline`, `fall`, `confusion_onset`, `orthostatic_instability`, `sedation`. Codify these as Go constants. Add `IsValidInstabilityPrimitive(string) bool`.

- [ ] **Step 3: JSON round-trip tests + alphabetical-ordering invariant on Patterns**
  No computation tests — types only.

- [ ] **Step 4: Commit**
  ```
  feat(substrate): Instability Chronology — types + canonical vocabulary (computation deferred to kb-33)
  ```

---

### Task E: ObservationLayer proto IDL

**Files:**
- Create: `proto/v1/observation_layer.proto`
- Create: `proto/v1/common.proto` (or extend existing if found)
- Create or update: `proto/buf.yaml`, `proto/buf.gen.yaml` (if buf not already in the repo)

**Scope:** Wire-format contract only. **No** generated Go bindings checked in (those come with kb-33). **No** server implementation. **No** action-layer services (PharmacistWorklistService, RACHOperationalView, GovernanceWorkspace) — those are out of scope per CAPE Addendum line 190 and remain Phase 2.

**Service signature (verbatim CAPE Addendum lines 152–171, transliterated from Go-like syntax to protobuf):**

```protobuf
service ObservationLayer {
  rpc GetResidentScoring(ResidentRequest) returns (FiveLayerScoring);
  rpc GetSignalDetections(ResidentTimeWindowRequest) returns (SignalList);
  rpc GetInstabilityChronology(ResidentTimeWindowRequest) returns (Chronology);
  rpc GetTrajectoryPrimitives(ResidentRequest) returns (TrajectoryComposite);

  rpc GetSubstrateState(ResidentRequest) returns (SubstrateSnapshot);
  rpc GetEvidenceTrace(EvidenceQueryRequest) returns (EvidenceGraph);
  rpc GetFailedInterventionHistory(ResidentRequest) returns (FailedInterventionList);

  rpc GetFacilityPatterns(FacilityRequest) returns (FacilityPatternList);
  rpc GetResidentList(FacilityFilterRequest) returns (ResidentSummaryList);

  rpc GetFacilityInstabilityOverview(FacilityTimeWindowRequest) returns (FacilityInstability);
  rpc GetResidentCluster(FacilityClusterRequest) returns (ResidentClusterList);
}
```

- [ ] **Step 1: Author the .proto file**
  All 11 RPCs. For message types whose shape is fully specified in CAPE (Chronology, FailedInterventionRecord, ResidentID, TimeWindow, Severity), match the field set 1:1 against the Go types from Tasks B + D. For messages whose shape is unspecified (FiveLayerScoring, TrajectoryComposite, SubstrateSnapshot, EvidenceGraph, FacilityPattern, ResidentSummary, FacilityInstability, ResidentCluster), declare minimal viable messages with a doc comment `// TODO(kb-33): full shape pending CAPE Part X authoring`.

- [ ] **Step 2: Lint via `buf lint`**
  If buf isn't configured, set up minimal `buf.yaml` with default lint rules. If buf is already configured elsewhere in the repo, mirror that configuration.

- [ ] **Step 3: NO code generation in this commit**
  Code generation happens when kb-33 imports the proto. We're shipping the contract, not the binding.

- [ ] **Step 4: Commit**
  ```
  feat(substrate): ObservationLayer proto IDL — 11 RPC contract for kb-33 (server impl deferred)
  ```

---

## Pre-acceptance gate

Before declaring Step 4 complete and starting Step 5 (kb-33 build):

1. ✅ All five tasks committed and merged to `main`.
2. ✅ Full repo `go build ./...` and `go test ./...` green across every affected service (kb-32, plus anything that imports `decision_metadata` or the new `shared/v2_substrate/clinical/*` packages).
3. ✅ `proto/v1/observation_layer.proto` lints clean.
4. ✅ Migration `shared/047_failed_interventions.sql` applies forward and rolls back on a test Postgres.
5. ⚠️ Clinical informatics lead reviews the rule-id → intervention-type classifier map authored in Task B Step 4 (no sign-off required to merge, but flag for follow-up).

## Out of scope (still deferred)

- Full ObservationLayer gRPC server implementation (kb-33 — Roadmap Step 5)
- Action-layer services (PharmacistWorklistService, RACHOperationalView, GovernanceWorkspace) — Phase 2
- Instability Chronology computation logic — kb-33 Week 20
- PRN velocity wiring into kb-23 decision cards or kb-32 craft engine — Phase 2

Plan complete and saved.
