# S2 Layer 1 — Baseline Rendering Build Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Implement S2 Layer 1 (Baseline Rendering) per the canonical specification stack. Ship the `s2-aggregator` backend service with a layer-aware view-builder pattern, the integration to the substrate prerequisites already shipped in Steps 1–4 of this roadmap, and the EvidenceTrace audit infrastructure including the `EscalationEvent` hook stubbed for Phase 1 log-only operation.

**Canonical source documents (READ BEFORE TOUCHING ANYTHING):**
- `docs/superpowers/plans/S2_Resident_Workspace_Implementation_Guidelines_v1.md` (1475 lines) — the authoritative v1.0 spec. Parts 0–4 and 14–18 are the engineering-relevant sections per its own reading order. Parts 5–13 specify the substrate contracts each panel renders.
- `docs/superpowers/plans/S2_Adaptive_Cognition_Architectural_Commitment_Addendum.md` (477 lines) — commits S2 to a single adaptive cognition workspace at five depth layers. Layer 1 is what this plan implements. Layers 2–5 are stubbed as architectural slots (Addendum Part 8.1 specifies the `S2ViewBuilder` interface shape with `BuildLayer2..5` stub methods).

**Why a separate plan:** S2 Layer 1 implementation is the natural next workstream after the Step 1–4 substrate consolidation. Bundling it into Step 5 (kb-33 build) would conflate the worklist engine with the workspace surface — different audiences, different architectures, different teams. This plan tracks Layer 1 explicitly so the engineering team has a build sequence aligned to v1.0 Part 19 (originally framed as Weeks 8–14, here translated to task-based sequencing).

**Authoring discipline (load-bearing — re-read Addendum Part 6 before each task):**
- **Claude provides architectural framework, structural primitives, integration code.** Layer 1 baseline rendering is fully specified in v1.0 — that specification is the engineering target, not Claude speculation.
- **Claude does NOT author cognitive content.** No concern vectors, no "what experts typically check" memory aid content, no escalation trigger thresholds beyond what v1.0 explicitly states. Layer 2–5 content is reserved for senior consultant pharmacist authoring + pilot evidence per Addendum Part 6.1.
- **Where the plan implies a clinical decision** (e.g., the specific failed-intervention-pattern-detection rules in v1.0 Part 8.5), agent should implement the structural framework and surface the clinical content as a `TODO(senior consultant pharmacist authoring)` slot in the code — same pattern as the proto IDL's `TODO(kb-33)` placeholders from Step 4 Task E.

**Architecture (per v1.0 Part 2):**

```
                    ┌─────────────────────────────────┐
                    │ Layer 2 Substrate               │
                    │ (kb-20 patient profile,         │
                    │  Phase 2-completion stores,     │
                    │  Step 4 substrate types)        │
                    └────────────┬────────────────────┘
                                 │
                                 ▼
┌──────────────────┐   ┌─────────────────────────────┐   ┌──────────────────┐
│ kb-33 CAPE       │──▶│ s2-aggregator (NEW)         │◀──│ kb-32 Craft      │
│ (worklist entry) │   │  • Layer-aware ViewBuilder  │   │ Engine           │
│ (TODO Step 5;    │   │  • Entry path handlers (4)  │   │ (kb-32 already   │
│  stub for now)   │   │  • Aggregation modules (7)  │   │  shipped)        │
└──────────────────┘   │  • Drill-through handlers   │   └──────────────────┘
                       │  • Pharmacist action capture│
┌──────────────────┐   │  • EvidenceTrace + audit    │   ┌──────────────────┐
│ kb-29 Templates  │──▶│  • EscalationEvent stub     │◀──│ Restraint signal │
│ (not yet built;  │   │                             │   │ evaluations      │
│  stub for now)   │   └─────────┬───────────────────┘   │ (kb-29; stub)    │
└──────────────────┘             │                       └──────────────────┘
                                 ▼
                       ┌─────────────────────────────┐
                       │ Frontend rendering layer    │
                       │ (vaidshala/clinical-        │
                       │  applications/ui/clinician/ │
                       │  — separate downstream plan)│
                       └─────────────────────────────┘
```

**Tech stack:** Go, gRPC, HTTP (Gin), Postgres, Redis (short-TTL view cache). Follows the kb-services repo conventions (cmd/server/main.go + internal/api + internal/store/postgres/migrations).

**Scope boundaries:**
- IN SCOPE: backend `s2-aggregator` service per v1.0 Part 15 directory layout
- IN SCOPE: API contracts per v1.0 Part 16 (gRPC + HTTP)
- IN SCOPE: integration with Step 1–4 substrate (PostgresSubstrateClient, FailedInterventionRecord, prn_velocity, instability_chronology types, citations registry, override store, opt-out store, decision_metadata.Metadata.RecommendationID)
- IN SCOPE: layer-aware view builder with Layer 2–5 stubs returning "not yet implemented"
- IN SCOPE: EscalationEvent audit hook (log-only Phase 1 per Addendum Part 5.5)
- OUT OF SCOPE: frontend rendering components (separate plan: `2026-05-11-s2-layer-1-frontend-build.md`)
- OUT OF SCOPE: Layer 2–5 content (Addendum Part 6 — deferred to senior pharmacist + pilot evidence)
- OUT OF SCOPE: kb-33 CAPE worklist integration (Step 5 — stub the integration point for now)
- OUT OF SCOPE: kb-29 templates wiring (kb-29 not yet built — stub the read path)

**Branch:** `feat/s2-layer-1-build` off `main` (currently at `c7be8701`). One commit per task. Push to origin between tasks.

---

## File structure

**New service:** `backend/services/s2-aggregator/` per v1.0 Part 15 directory tree:

```
backend/services/s2-aggregator/
├── cmd/server/main.go
├── internal/
│   ├── api/                       # gRPC + HTTP + event subscriptions
│   ├── store/postgres/            # views, actions, audit_events, migrations
│   ├── store/redis/               # short-TTL view cache
│   ├── aggregation/               # view builder + 7 panel modules + complex activation
│   ├── entry_paths/               # 4 entry path handlers
│   ├── drill_through/             # substrate observation, trajectory history, negative-evidence
│   ├── actions/                   # 11 pharmacist actions + reasoning + session
│   ├── audit/                     # EvidenceTrace, visibility class, EscalationEvent
│   ├── form_factor/               # desktop / mobile (data-shape adapter; rendering is frontend)
│   └── erm_integration/           # quarterly ERM review integration
├── proto/                         # gRPC IDL for S2WorkspaceService
├── migrations/                    # s2-aggregator-local
└── tests/integration/             # E2E
```

---

## Task sequencing rationale

Tasks map v1.0 Part 19 Weeks 8–14 onto task units, plus structural framework tasks for the Addendum's layer-aware pattern:

1. **Service scaffold + layer-aware ViewBuilder interface** (foundation; mirrors v1.0 Week 8)
2. **Four entry paths + CAPE context band stub** (v1.0 Week 9)
3. **Trajectory aggregation + substrate drill-through** (v1.0 Week 10; consumes kb-20 + prn_velocity)
4. **Pending recommendations panel + restraint signal pairing** (v1.0 Week 11; consumes kb-32 + citations registry from Step 2)
5. **Failed intervention history + goals-of-care + care intensity** (v1.0 Week 12; consumes failed_interventions from Step 4)
6. **Eleven pharmacist actions + reasoning capture + session context** (v1.0 Week 13; writes to existing override store + audit_events)
7. **EvidenceTrace integration + visibility class + EscalationEvent stub** (v1.0 Week 13 audit; integrates Phase 1c ethics_log)
8. **gRPC + HTTP API surface** (v1.0 Part 16; binds everything together)
9. **Verification-not-belief structural test + 6 test categories** (v1.0 Part 17; cross-cutting)
10. **End-to-end integration test against Step 1–4 substrate** (DSN-skip pattern; matches Phase 2-completion Task 8)

---

### Task 1: Service scaffold + layer-aware ViewBuilder interface

**Files:**
- Create: `backend/services/s2-aggregator/` directory tree per v1.0 Part 15
- Create: `cmd/server/main.go` — service boot, config, graceful shutdown, health endpoint
- Create: `internal/aggregation/view_builder.go` — `S2ViewBuilder` interface per Addendum Part 8.1 with Layer 1 implemented + Layer 2–5 stubs returning `errors.New("layer N not yet implemented — Addendum Part 6")`
- Create: `internal/aggregation/types.go` — `S2View`, `Layer1View`, `WorkspaceRequest`, `EscalationEvent`, supporting types
- Create: `internal/store/postgres/migrations/001_s2_views.sql` — minimal tables for view cache (later tasks add tables)
- Create: `go.mod` — module path `github.com/cardiofit/s2-aggregator`

**Interface contract per Addendum Part 8.1:**

```go
type S2ViewBuilder interface {
    BuildLayer1Baseline(ctx context.Context, req WorkspaceRequest) (Layer1View, error)
    BuildLayer2Escalated(ctx context.Context, req WorkspaceRequest) (Layer2View, error)  // stub
    BuildLayer3Complex(ctx context.Context, req WorkspaceRequest) (Layer3View, error)    // stub
    BuildLayer4SituationBoard(ctx context.Context, req WorkspaceRequest) (Layer4View, error)  // stub
    BuildLayer5Investigation(ctx context.Context, req WorkspaceRequest) (Layer5View, error)   // stub
    EscalateToLayer(ctx context.Context, currentLayer int, targetLayer int, req WorkspaceRequest) (View, error)
    LogEscalation(ctx context.Context, escalation EscalationEvent) error
}
```

- [ ] Step 1: scaffold directory + go.mod
- [ ] Step 2: implement `S2ViewBuilder` interface with Layer 1 returning a zero-value `Layer1View` initially; Layers 2–5 return sentinel errors
- [ ] Step 3: unit test that asserts interface conformance + sentinel errors for Layers 2–5
- [ ] Step 4: `go build ./...` clean; `go vet ./...` clean
- [ ] Step 5: commit
  ```
  feat(s2): s2-aggregator service scaffold + layer-aware ViewBuilder interface

  Implements S2 Layer 1 build Task 1 — service scaffolding per v1.0 Part 15
  directory tree, plus the S2ViewBuilder interface per Addendum Part 8.1.
  Layer 1 returns zero-value Layer1View; Layers 2–5 return sentinel errors
  matching the Addendum's content-deferral discipline (Part 6).
  ```

---

### Task 2: Four entry paths + CAPE context band stub

**Files:**
- Create: `internal/entry_paths/from_worklist.go` + `_test.go` — CAPE worklist entry handler (kb-33 not yet built; stub the CAPE context shape)
- Create: `internal/entry_paths/from_search.go` + `_test.go`
- Create: `internal/entry_paths/from_notification.go` + `_test.go`
- Create: `internal/entry_paths/from_cross_reference.go` + `_test.go`
- Create: `internal/aggregation/cape_context.go` + `_test.go` — CAPE context band rendering per v1.0 Part 4.1 (CAPE-driven entry surfaces the primary signals that drove prioritisation at the top of S2)

**Entry path discipline per v1.0 Part 3:**
- Each handler returns a `EntryPathMetadata` struct (per v1.0 Part 3.5) that includes which path triggered, what context to carry through (CAPE signals vs notification vs search query vs cross-reference origin)
- `WorkspaceRequest` accepts `EntryPath` enum + `EntryContext` polymorphic struct

- [ ] Step 1: entry path enum + `EntryContext` polymorphic type
- [ ] Step 2: four entry-path handlers; CAPE one accepts a stub `CAPEContext` shape (the real shape lives in kb-33 — mark with `TODO(kb-33 integration)`)
- [ ] Step 3: `cape_context.go` renders the band per v1.0 lines 250–262 (signals that drove prioritisation, retained, not re-derived)
- [ ] Step 4: tests for each handler + CAPE band rendering
- [ ] Step 5: build + vet clean
- [ ] Step 6: commit

---

### Task 3: Trajectory aggregation + substrate drill-through

**Files:**
- Create: `internal/aggregation/trajectories.go` + `_test.go` — multi-parameter trajectory rendering per v1.0 Part 5
- Create: `internal/drill_through/substrate_observation.go` + `_test.go`
- Create: `internal/drill_through/trajectory_history.go` + `_test.go`
- Create: `internal/drill_through/negative_evidence.go` + `_test.go`

**Substrate sources (already shipped in Steps 1–4):**
- `kb-20` clinical snapshot via `kb-32/internal/store/postgres.PostgresSubstrateClient` (Phase 2-completion Task 1) — egfr, dbi, acb, cfs, care_intensity
- `prn_velocity.Compute` (Step 4 Task C) — administration velocity for three classes
- `instability_chronology` types (Step 4 Task D) — types only; computation deferred to kb-33

**Trajectory rendering per v1.0 Part 5:**
- Each parameter: current value, velocity (rate of change), baseline (90d), threshold flags (clinically meaningful crossings), sparse-data degradation (per v1.0 Part 5.3 — no false-precision trajectories when <3 observations)
- Multi-parameter composition per v1.0 Part 5.4 — concurrent trajectory shifts surface together (eGFR declining + DBI increasing = composite "anticholinergic burden + renal decline" signal)
- Drill-through per v1.0 Part 10.1 — every claim one click from substrate observation; substrate confidence visible

**Verification-not-belief discipline:** Each trajectory + each drill-through carries an explicit `SubstrateRef` pointing to the underlying observation row. Tests enforce this structurally per v1.0 Part 17 Critical test.

- [ ] Step 1: trajectory types (velocity, baseline, threshold, SparseDataFlag)
- [ ] Step 2: per-parameter trajectory computation (eGFR, DBI, ACB, CFS, weight, BP, PRN-velocity per class)
- [ ] Step 3: sparse-data graceful degradation tests (<3 observations → no velocity, render "insufficient data")
- [ ] Step 4: multi-parameter composition per v1.0 Part 5.4
- [ ] Step 5: drill-through handlers (substrate observation, trajectory history series, negative-evidence rendering per v1.0 Part 10.4)
- [ ] Step 6: critical structural test from v1.0 Part 17 — every trajectory claim has SubstrateRef
- [ ] Step 7: build/vet/test clean
- [ ] Step 8: commit

---

### Task 4: Pending recommendations panel + restraint signal pairing

**Files:**
- Create: `internal/aggregation/pending_recs.go` + `_test.go` — pending recommendation cards per v1.0 Part 6
- Create: `internal/aggregation/restraint_signals.go` + `_test.go` — Phase 1 advisory-only restraint rendering per v1.0 Part 7

**Substrate sources (already shipped):**
- kb-32 `Packet` + `PipelineResult` (recommendations + lifecycle state + Assessment)
- kb-32 `citations.Registry.ListCitations` (fire-time pinned citations via the wire-up in Step 4 Task A — `decision_metadata.Metadata.RecommendationID` is the lookup key)
- kb-32 `OverrideReason` history (Phase 2-completion Task 5 dual-vocab taxonomy)

**Card structure per v1.0 Part 6.1:**
- Type tag, urgency, Layer 1/2/3 framing body, 5-dim Assessment scores, restraint signal pairing badge if applicable, hold reason if applicable, citation drawer link
- Lifecycle state rendering per v1.0 Part 6.2 (detected, drafted, submitted, viewed, decided, monitoring-active)
- Confidence dimensions per v1.0 Part 6.3 (substrate confidence + clinical confidence — two-axis)

**Restraint signal pairing per v1.0 Part 7:**
- Phase 1 advisory-only mode: restraint signals appear alongside the recommendation they pair with, not as suppressive overlays
- Acknowledgment workflow per v1.0 Part 7.2 (pharmacist explicitly acknowledges; acknowledgment captured to audit)
- Safety-critical bypass per v1.0 Part 7.4 with mandatory reasoning capture
- Transition criteria visibility per v1.0 Part 7.3 — informational only, no auto-suppression

- [ ] Step 1: pending recommendation card struct + lifecycle state enum
- [ ] Step 2: confidence dimensions surfacing (read from kb-32 Assessment)
- [ ] Step 3: restraint signal pairing logic
- [ ] Step 4: acknowledgment workflow + audit hook
- [ ] Step 5: safety-critical bypass with mandatory reasoning
- [ ] Step 6: empty-state rendering per v1.0 Part 6.5
- [ ] Step 7: tests + structural verification-not-belief check
- [ ] Step 8: commit

---

### Task 5: Failed intervention history + goals-of-care + care intensity

**Files:**
- Create: `internal/aggregation/failed_history.go` + `_test.go` — per v1.0 Part 8
- Create: `internal/aggregation/goals_care_intensity.go` + `_test.go` — per v1.0 Part 9

**Substrate sources (already shipped):**
- `failed_interventions.Store.ListByResident` (Step 4 Task B) — note known gap: `ResidentID=uuid.Nil` issue from Task B. Layer 1 implementation surfaces this gap with "FIR retrieval incomplete — kb-32 RecommendationID→ResidentID resolver pending" badge per the gap surfaced in superseded-spec Risk #3
- kb-20 goals-of-care state machine (already in kb-20 migrations)
- kb-20 care_intensity_history table

**Failed intervention pattern detection per v1.0 Part 8.5:**
- Pattern detection logic is **deferred to senior consultant pharmacist authoring**. Task ships the structural framework: a `PatternDetector` interface with a default no-op implementation. Senior pharmacist content fills in the pattern rules later.
- Code comment must say: `// TODO(senior consultant pharmacist authoring per S2 Addendum Part 6.1)`

**Goals-of-care conflict surfacing per v1.0 Part 9.4:**
- When a pending recommendation conflicts with documented goals (e.g., ADD aggressive intervention on `comfort_focused` resident), surface the conflict on the recommendation card
- Reads `Packet.Type` + `ClinicalSnapshot.CareIntensity` + GoC state machine current entry; rule logic is mechanical (already encoded in `SubstrateBackedScorer.scoreGoalsOfCareAlignment` from Phase 2-completion Task 2)

- [ ] Step 1: FailedInterventionRecord card structure
- [ ] Step 2: FIR retrieval with gap-handling banner
- [ ] Step 3: `PatternDetector` interface + no-op default + senior-pharmacist-authoring TODO
- [ ] Step 4: goals-of-care panel + transition history rendering
- [ ] Step 5: care intensity panel with sparse-data degradation
- [ ] Step 6: conflict surfacing logic
- [ ] Step 7: tests
- [ ] Step 8: commit

---

### Task 6: Eleven pharmacist actions + reasoning + session

**Files:**
- Create: `internal/actions/handlers.go` + `_test.go` — eleven action handlers per v1.0 Part 12.1
- Create: `internal/actions/reasoning_capture.go` + `_test.go` — mandatory vs optional reasoning per v1.0 Part 12.3
- Create: `internal/actions/session_context.go` + `_test.go` — session metadata per v1.0 Part 12.4

**The eleven actions (from v1.0 Part 12.1):**
1. open (recommendation viewed; lifecycle transition)
2. modify (recommendation edited; mandatory reasoning)
3. defer (deferred to later; reasoning optional)
4. override (declined; mandatory reasoning + override taxonomy code from Phase 2-completion Task 5)
5. mark reviewed (resident marked as reviewed for the session)
6. flag for follow-up
7. add note (free-text pharmacist note)
8. open complex workspace (Layer escalation — wires to `EscalateToLayer` from Task 1)
9. drill into substrate (one-click verification)
10. acknowledge restraint signal
11. invoke safety-critical bypass (mandatory reasoning; audit-prioritised)

**Reasoning capture rules per v1.0 Part 12.3:**
- Mandatory: modify, override, invoke safety-critical bypass
- Optional: defer, add note
- Not applicable: open, mark reviewed, flag for follow-up, drill into substrate, acknowledge restraint, open complex workspace

**Write paths:**
- override → kb-32 override store via `POST /v1/craft/override/:recommendation_id` (Phase 2-completion existing endpoint) — triggers Step 4 Task B FIR auto-write hook
- All actions → s2-aggregator `pharmacist_actions` table (new migration in this task)
- All actions → audit hook in Task 7

- [ ] Step 1: action enum + ActionRequest struct
- [ ] Step 2: reasoning capture with mandatory/optional rules enforced
- [ ] Step 3: eleven handlers
- [ ] Step 4: integration with kb-32 override store on override action
- [ ] Step 5: session context + StartSession/EndSession lifecycle
- [ ] Step 6: migration for pharmacist_actions table
- [ ] Step 7: tests
- [ ] Step 8: commit

---

### Task 7: EvidenceTrace integration + visibility class + EscalationEvent stub

**Files:**
- Create: `internal/audit/evidence_trace.go` + `_test.go` — integrates Phase 1c `ethics_log.Logger`
- Create: `internal/audit/visibility_class.go` + `_test.go` — PDP enforcement per v1.0 Part 13.3
- Create: `internal/audit/escalation_event.go` + `_test.go` — `EscalationEvent` per Addendum Part 8.1 (LOG-ONLY in Phase 1)
- Create: `migrations/002_s2_audit_events.sql` — local audit_events table

**What gets audited per v1.0 Part 13.1:**
- View rendering events (every workspace open)
- All 11 pharmacist actions
- All drill-through events
- System events (recommendation lifecycle transitions visible in S2)
- Cognitive escalation events (Addendum Part 8.1 — Phase 1 logged for audit only)

**Visibility class enforcement per v1.0 Part 13.3:**
- S2 content is PDP visibility class
- Aggregated patterns of S2 use NOT surveilled for performance evaluation (ethical architecture §8 + Addendum Part 5.2)
- Per-pharmacist self-visibility view: pharmacist sees their own patterns (PDP) — no cross-pharmacist visibility

**EscalationEvent structure per Addendum Part 8.1:**

```go
type EscalationEvent struct {
    PharmacistID  uuid.UUID
    ResidentID    uuid.UUID
    SessionID     uuid.UUID
    FromLayer     int
    ToLayer       int
    TriggeredBy   EscalationTrigger // automatic | pharmacist_initiated
    Timestamp     time.Time
    AuditTraceID  uuid.UUID
}
```

Phase 1 commitment per Addendum Part 5.5: **logged for audit only**. No platform behaviour is driven by escalation patterns until ESC approval + 12mo evidence + external clinical informatics review + pharmacist self-visibility operational.

- [ ] Step 1: audit_events migration
- [ ] Step 2: `EvidenceTrace` writer integrating with shared `ethics_log.Logger`
- [ ] Step 3: visibility class enforcement middleware
- [ ] Step 4: `EscalationEvent` capture (log-only in Phase 1; no behavioural read path)
- [ ] Step 5: tests + structural test that asserts NO surveillance use of escalation patterns (cross-pharmacist aggregation must return 403)
- [ ] Step 6: commit

---

### Task 8: gRPC + HTTP API surface

**Files:**
- Create: `proto/v1/s2_workspace.proto` — `S2WorkspaceService` per v1.0 Part 16 (15 RPCs)
- Create: `internal/api/grpc.go` + `_test.go` — gRPC server bindings
- Create: `internal/api/http.go` + `_test.go` — HTTP gateway for browser clients
- Create: `internal/api/events.go` + `_test.go` — event subscriptions (recommendation lifecycle changes)

**The 15 RPCs from v1.0 Part 16:**
- GetResidentWorkspace, RefreshResidentWorkspace
- GetSubstrateObservation, GetTrajectoryHistory
- OpenRecommendation, ModifyRecommendation, DeferRecommendation, OverrideRecommendation
- MarkResidentReviewed, FlagForFollowUp, AddPharmacistNote
- OpenComplexWorkspace, AcknowledgeRestraintSignal, InvokeSafetyCriticalBypass
- GetS2AuditTrail
- StartSession, EndSession

**Permissions middleware wiring:**
- All routes wrap with the `KB32_PERMISSIONS_ENFORCED` gating pattern from Phase 2-completion Task 7 (renamed `S2_PERMISSIONS_ENFORCED` for service-local env var)
- Same `ginPermMW` adapter pattern (or its gRPC interceptor equivalent)
- Visibility class: PDP for all routes

- [ ] Step 1: proto IDL with 15 RPCs + supporting messages
- [ ] Step 2: gRPC server bindings
- [ ] Step 3: HTTP gateway (Gin handlers calling into the same service)
- [ ] Step 4: event subscription handler for recommendation lifecycle changes
- [ ] Step 5: permissions middleware wiring
- [ ] Step 6: tests including a structural API-completeness test (all 15 RPCs present, every endpoint behind PDP guard)
- [ ] Step 7: commit

---

### Task 9: Verification-not-belief structural test + 6 test categories

**Files:**
- Create: `tests/structural/verification_not_belief_test.go` — the critical test from v1.0 Part 17

**The critical test from v1.0 Part 17 lines 1298–1313:**

```go
func TestEveryClaimHasSubstrateReference(t *testing.T) {
    view := buildTestS2View()
    claims := extractAllClaims(view)
    for _, claim := range claims {
        require.NotNil(t, claim.SubstrateRef,
            "Claim '%s' has no substrate reference — violates verification-not-belief discipline", claim.Text)
    }
}
```

This is structural — it asserts the invariant from v1.0 Principle 2 across every render path. Every claim in every panel of Layer 1 must carry a substrate reference.

**Six test categories per v1.0 Part 17.1–17.6:**
- View assembly tests
- Trajectory rendering tests
- Pending recommendation tests
- Restraint signal rendering tests
- Drill-through tests
- Audit trail tests

Each gets a dedicated `_test.go` file with the assertions listed in v1.0 Part 17.

- [ ] Step 1: claim extraction helper
- [ ] Step 2: verification-not-belief structural test
- [ ] Step 3: six test category files
- [ ] Step 4: cross-cutting test: view assembly from each of 4 entry paths produces structurally valid Layer1View
- [ ] Step 5: commit

---

### Task 10: End-to-end integration test against Step 1–4 substrate

**Files:**
- Create: `tests/integration/end_to_end_s2_layer_1_test.go` — DSN-skip pattern mirroring Phase 2-completion Task 8

**What gets exercised:**
- Seed a kb-20 ClinicalSnapshot (via the kb-32 PostgresSubstrateClient pattern from Step 4 Task 1's E2E test)
- Seed a kb-32 recommendation through the craft pipeline (Phase 2-completion Task 8 pattern)
- Seed a Failed Intervention Record (Step 4 Task B store)
- Build the Layer1View
- Assert: all panels populate; CAPE context band carries entry-path metadata; trajectories render with substrate refs; pending recommendation card shows citation pin set; FIR panel surfaces the record; override pathway always reachable; audit_events row written; EscalationEvent NOT written (Layer 1 only — no escalation in this test)

Skips cleanly when `VAIDSHALA_TEST_DSN` unset.

- [ ] Step 1: seed helper integrating substrate from Steps 1–4
- [ ] Step 2: E2E happy path
- [ ] Step 3: E2E with override action (writes through to kb-32 override store + FIR auto-write per Step 4 Task B)
- [ ] Step 4: E2E with drill-through (asserts substrate observation handler returns the seeded row)
- [ ] Step 5: commit
  ```
  test(s2): end-to-end Layer 1 against Step 1–4 substrate

  Implements S2 Layer 1 build Task 10 — wires PostgresSubstrateClient,
  citations.PostgresRegistry, failed_interventions.PostgresStore,
  kb-32 override store, and the s2-aggregator Layer1View end-to-end.
  Skips without VAIDSHALA_TEST_DSN.

  Closes the S2 Layer 1 build plan (Tasks 1–10).
  ```

---

## Pre-acceptance gate (per v1.0 Part 19 Week 14 + Addendum Part 8)

Before declaring S2 Layer 1 build complete and starting Layer 2–5 content authoring (which is NOT Claude's work):

1. ✅ Service compiles, all tests pass, vet clean across `s2-aggregator`
2. ✅ Verification-not-belief structural test passes
3. ✅ E2E integration test passes against staging Postgres (operational, not just structural)
4. ✅ Permissions middleware enforced behind `S2_PERMISSIONS_ENFORCED=true` flag in production deploy
5. ✅ EscalationEvent capture verified log-only (no behavioural read paths exist)
6. ⏳ External clinical informatics UX review (operational gate, post-code)
7. ⏳ Pilot pharmacist user testing (3 pharmacists × 1 week, per v1.0 Week 14)

## Out of scope (still deferred)

- **Frontend rendering components** — separate plan `2026-05-11-s2-layer-1-frontend-build.md` (Next.js components at `vaidshala/clinical-applications/ui/clinician/`)
- **Layer 2–5 cognitive content** — Addendum Part 6.1; senior consultant pharmacist authoring + pilot evidence
- **Failed intervention pattern detection rules** — v1.0 Part 8.5; senior pharmacist authoring
- **kb-33 CAPE worklist real integration** — Step 5 build (stubbed in Task 2)
- **kb-29 templates wiring** — kb-29 not yet built (stubbed in Task 4)
- **Quarterly ERM review automation** — v1.0 Part 15 names `erm_integration/`; minimal stub in Task 7, full workflow deferred

Plan complete and saved.
