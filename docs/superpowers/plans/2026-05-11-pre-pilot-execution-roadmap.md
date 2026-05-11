# Pre-Pilot Execution Roadmap

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Plan type:** Master sequencing roadmap. Each task is a bite-sized decision point at the roadmap level. Code-level implementation detail (failing-test → implement → commit) lives in the sub-plans referenced from each task. Do not execute roadmap tasks out of order — every task except parallel-able authoring tasks depends on the previous one's gate.

**Goal:** Sequence every workstream from the current state (Phase 3 ethics gates on `feat/phase-3-tightened-ethics-gates`) through pilot Phase 1 launch (Months 1–8 of pilot timeline) as one canonical execution path. Resolve phase-numbering ambiguity by using step numbers throughout.

**Architecture:** Master roadmap referencing sub-plans. Three categories of work: (i) **code execution** via existing or to-be-written implementation plans; (ii) **specification authoring** for the four MVP-blocking spec docs not yet written; (iii) **operational deliverables** outside the codebase (Ethics Steering Committee charter, Pharmacist Advisory Group recruitment, external auditor engagement). The roadmap holds all three categories together so nothing is forgotten.

**Tech Stack:** Roadmap is markdown-only. Code work lands in Go (kb-32, ethics-monitoring, pharmacist-self-visibility, kb-33-to-be-built) and Postgres migrations. Spec authoring lands in `docs/superpowers/plans/` as markdown.

---

## Decision matrix — which Phase 4?

This roadmap supersedes the phase-numbering used elsewhere. Reference labels for collision avoidance:

| Old label | Canonical name in this roadmap | Where defined |
|---|---|---|
| "Phase 2-completion" | Step 3 | `2026-05-09-phase-2-completion.md` |
| "CAPE Weeks 18–24" / "Layer 2/3 Phase 3" | Step 10 (Build CAPE engine) | `CAPE_Implementation_Guidelines_v1_1.md` Part 16 |
| "Phase 4 of Vaidshala execution" / "carry-forward bucket" | Step 4 (CAPE substrate prerequisites) + Step 11 (post-pilot Layer 4 UX) | this roadmap + `2026-05-07-phase-4-rule-volume-and-layer-4-surfaces.md` (cross-check needed) |
| "CAPE Phase 4" (wearables, NLP, predictive) | Out of scope for this roadmap — gated on pre-Phase-4 architecture audit per Addendum §5.4 | future |
| "KB-29 Phase 4" (reasoning primitive extraction) | Out of scope for this roadmap | future |

---

## File Structure (artifacts this roadmap produces)

**New plans authored by this roadmap's tasks:**
- Create: `docs/superpowers/plans/2026-05-11-cape-substrate-prerequisites.md` (Task 4)
- Create: `docs/superpowers/plans/2026-05-11-s2-resident-workspace-spec.md` (Task 5)
- Create: `docs/superpowers/plans/2026-05-11-s3-gp-communication-hub-spec.md` (Task 6)
- Create: `docs/superpowers/plans/2026-05-11-rmmr-workflow-spec.md` (Task 7)
- Create: `docs/superpowers/plans/2026-05-11-cpd-ahpra-records-spec.md` (Task 8)
- Create: `docs/superpowers/plans/2026-05-11-kb-33-triage-engine-build.md` (Task 9)

**Existing plans referenced (not modified):**
- `docs/superpowers/plans/2026-05-09-phase-2-completion.md`
- `docs/superpowers/plans/2026-05-09-phase-3-ethics-monitoring-and-pre-pilot-gates.md`
- `docs/superpowers/plans/2026-05-07-phase-4-rule-volume-and-layer-4-surfaces.md` (cross-check needed in Task 4)

**Existing implementation guideline docs referenced:**
- `docs/superpowers/plans/CAPE_Implementation_Guidelines_v1_1.md`
- `docs/superpowers/plans/CAPE_v1_1_Architectural_Commitment_Addendum.md`
- `Ethical_Architecture_Implementation_Guidelines_v1.md` (repo root)
- `Pharmacist_Self_Visibility_Implementation_Guidelines_v1.md` (repo root)
- `backend/shared-infrastructure/knowledge-base-services/Recommendation_Craft_Implementation_Guidelines_v1.md`

---

## Step 1: Open and merge Phase 3 PR

**Files:**
- No code changes. Branch operations only.

**Branch state:**
- `feat/phase-3-tightened-ethics-gates` exists on origin, commit `266d000b` (CI invariance gates).
- 8 commits ahead of `main` at `c8d4215e`.

- [ ] **Step 1: Open PR**

```bash
cd /Volumes/Vaidshala/cardiofit
gh pr create --base main --head feat/phase-3-tightened-ethics-gates \
  --title "Phase 3 (tightened): ethics monitoring + pre-pilot gates" \
  --body "$(cat <<'EOF'
## Summary
- Standalone ethics-monitoring service with cron orchestrator over Phase 1c pattern detectors
- Demographic stratification pipeline (6 dimensions) feeding DetectBiasDisparity
- Capacity + restrictive-practice consent gate (Stage 3.5) in kb-32 ERM pipeline
- Layer 4 deep-audit GET /v1/explain/:decision_id endpoint with audit-trail returns
- CI invariance gates: frame-vs-content content_hash + override pathway availability

## Test plan
- [x] ethics-monitoring: 4 packages green under -race
- [x] kb-32-recommendation-craft: 22 packages green including integration / clinical_safety / metric_integrity
- [x] pharmacist-self-visibility: 8 packages green including new override-pathway gates
- [x] shared/v2_substrate/ethics/bias_stratification: 10 tests green under -race

## Carry-forwards documented
See docs/superpowers/plans/2026-05-11-pre-pilot-execution-roadmap.md
EOF
)"
```

- [ ] **Step 2: Verify PR opened, link saved**

Run:
```bash
gh pr list --head feat/phase-3-tightened-ethics-gates --json url,number
```

Expected: one open PR with URL and number printed.

- [ ] **Step 3: Merge with --no-ff (matching Phase 1+2 pattern)**

Once review approval lands (or immediately for solo-developer flow):
```bash
gh pr merge --merge --delete-branch=false <PR_NUMBER>
```

The `--delete-branch=false` preserves the branch on origin as a safety reference, matching the Phase 1+2 retention strategy where `feat/phase-1-trust-architecture-plans` was kept after `c8d4215e`.

- [ ] **Step 4: Verify main contains Phase 3**

```bash
git fetch origin
git -C /Volumes/Vaidshala/cardiofit log --oneline origin/main | head -10
```

Expected: top of log shows merge commit incorporating commits `98e30cb7` through `266d000b`.

**Gate:** Step 2 cannot start until origin/main contains the Phase 3 commits. This locks the baseline for Phase 2-completion.

---

## Step 2: Set up Phase 2-completion branch off updated main

**Files:**
- No code changes. Branch operation only.

- [ ] **Step 1: Sync and branch**

```bash
cd /Volumes/Vaidshala/cardiofit
git checkout main
git pull origin main
git checkout -b feat/phase-2-completion-postgres-wiring
```

- [ ] **Step 2: Verify branch state**

```bash
git status --short
git log --oneline -3
```

Expected: clean working tree (modulo the pre-existing untracked `Ethical_Architecture_Implementation_Guidelines_v1.md` and friends); top commit is the Phase 3 merge.

---

## Step 3: Execute Phase 2-completion plan

**Sub-plan:** `docs/superpowers/plans/2026-05-09-phase-2-completion.md`

This is an existing 8-task plan. Execute via subagent-driven-development.

**Task summary** (verify against the actual plan):
1. PostgresSubstrateClient replaces inMemorySubstrateClient placeholder
2. SubstrateBackedScorer replaces DefaultAppropriatenessSource always-passes-gate placeholder
3. PostgresCitationRegistry + Stage 7 EvidenceTrace emission
4. Override taxonomy vocabulary alignment with Guidelines Part 5 3-letter codes (requires clinical informatics sign-off)
5. Prescriber framing opt-out HTTP endpoint
6. PDP middleware mounting on /v1/craft/draft and /v1/craft/override routes
7. End-to-end integration test with all-Postgres-backed dependencies
8. (Folded in from Phase 3 carry-forwards) `WithCapacityGate` constructor refactor + structured `HoldCode` field

- [ ] **Step 1: Read the sub-plan top-to-bottom**

```bash
cat docs/superpowers/plans/2026-05-09-phase-2-completion.md | head -50
```

Expected: plan header, goal, architecture, 8 tasks listed.

- [ ] **Step 2: Surface Task 4's external dependency before starting code**

Task 4 (override taxonomy alignment) requires clinical informatics sign-off. Surface to the team **before** starting code work on Tasks 1–3, because if Task 4 reveals that the taxonomy itself needs revision, it may invalidate test fixtures elsewhere. Action: email the clinical informatics lead a one-page summary of the snake_case ↔ 3-letter-code mismatch and request sign-off on the proposed alignment.

- [ ] **Step 3: Execute Tasks 1–8 via subagent-driven-development**

Dispatch one implementer subagent per task. Two-stage review (spec compliance → code quality) after each. Same pattern as Phase 3 execution.

- [ ] **Step 4: Verify pre-pilot acceptance gate (Phase 2-completion half)**

Per `2026-05-09-phase-3-ethics-monitoring-and-pre-pilot-gates.md` §"Pre-pilot acceptance gate (consolidated)":

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-32-recommendation-craft
go test -race ./tests/integration/... ./tests/clinical_safety/... ./tests/metric_integrity/...
```

Expected: all green. The integration test from Task 7 must exist and run against a real Postgres (skip-on-missing-DSN pattern preserved).

- [ ] **Step 5: Open PR and merge**

```bash
gh pr create --base main --head feat/phase-2-completion-postgres-wiring \
  --title "Phase 2-completion: Postgres substrate wiring + override taxonomy alignment" \
  --body "Closes pre-pilot acceptance gate items 1-6 from 2026-05-09-phase-3-...md. See plan 2026-05-09-phase-2-completion.md for task-level detail."
gh pr merge --merge --delete-branch=false <PR_NUMBER>
```

**Gate:** Step 4 of this roadmap cannot start until Phase 2-completion is merged. CAPE substrate prerequisites will modify some of the same files (decision_metadata, citations registry) and conflict resolution is cheaper after Phase 2-completion lands.

---

## Step 4: Author CAPE substrate prerequisites sub-plan

**Files:**
- Create: `docs/superpowers/plans/2026-05-11-cape-substrate-prerequisites.md`
- Cross-check: `docs/superpowers/plans/2026-05-07-phase-4-rule-volume-and-layer-4-surfaces.md` for overlap (auditor previously flagged this file may cover related ground)

- [ ] **Step 1: Read existing Phase 4 plan for overlap**

```bash
wc -l docs/superpowers/plans/2026-05-07-phase-4-rule-volume-and-layer-4-surfaces.md
head -80 docs/superpowers/plans/2026-05-07-phase-4-rule-volume-and-layer-4-surfaces.md
```

Decide one of:
- If existing plan covers Failed Intervention History, Instability Chronology, PRN velocity, RecommendationID field: **revise** that plan rather than write a new one.
- If existing plan covers a different scope (e.g., rule volume expansion, Layer 4 frontend surfaces): **write a new plan** as below and add a SUPERSEDED-style note to the existing plan if its scope has shifted.

- [ ] **Step 2: Author the CAPE substrate prerequisites plan**

Write `docs/superpowers/plans/2026-05-11-cape-substrate-prerequisites.md` covering these 5 substrate additions in this order:

**Sub-Task A: `decision_metadata.Metadata.RecommendationID` field (smallest, unblocks /v1/explain citations)**
- Add `RecommendationID *uuid.UUID` field to `shared/v2_substrate/ethics/decision_metadata/recorder.go`
- Migration to add the column (next available migration number after Phase 2-completion lands — verify the sequence)
- Update `InMemoryStore` round-trip tests
- Update `kb-32-recommendation-craft/internal/api/explain_handlers.go` to call `citationReg.ListCitations(ctx, md.RecommendationID.String())` when `RecommendationID != nil`
- Activate the previously-vacuous `TestExplain_RegistryError_DegradesGracefully` test
- Update `TestExplain_KnownDecision_Returns200WithFullPayload` to seed a non-empty citations list

**Sub-Task B: Failed Intervention History substrate**
- New package: `shared/v2_substrate/intervention_history/`
- Types: `FailedInterventionRecord` per CAPE v1.1 Part 4.3
- Store interface + `InMemoryStore` + `PostgresStore`
- Querier: `IsVetoActive(resident, interventionType) (bool, *Record)`, `History(resident, interventionType)`
- Migration for `failed_intervention_records` table with columns matching the Go struct
- Integration with kb-32 override capture: when an override is recorded with reversal-class reason codes (per Guidelines Part 5), automatically populate a `FailedInterventionRecord` with default 12-month retry-eligibility

**Sub-Task C: PRN administration velocity primitives**
- Determine package location: either extend `shared/v2_substrate/models/medication_use.go` or create `shared/v2_substrate/prn_velocity/`. Check what exists first.
- Function: `ComputeVelocityRatio(recent30dCount int, baseline90dAvg float64) float64`
- Function: `VelocityToSeverity(ratio float64) int` returning 1–5 per CAPE Part 3.2 thresholds (>4.0=5, >2.5=4, >1.5=3, >1.0=2, else 1)
- Reader interface: `PRNHistory.CountByClass(ctx, residentID, drugClass, window time.Duration) (int, error)`
- Tests covering each severity threshold + boundary cases

**Sub-Task D: Instability Chronology primitive**
- New package: `shared/v2_substrate/chronology/`
- Types: `ChronologyEvent`, `TemporalPattern`, `InstabilityChronology` per Addendum §3.1
- Assembly function: `AssembleChronology(ctx, residentID, window time.Duration) (InstabilityChronology, error)` that pulls events from substrate state machines and orders by timestamp
- Pattern recognition starter: 2 named clinical patterns to begin with — `VolumeContractionCascade` (diuretic change → intake decline → orthostatic instability → near-fall) per Addendum §3.2 example, and `SedationCascade` (PRN benzodiazepine escalation → sedation drift → fall) per CAPE Part 3.2
- Clinical pattern library is senior consultant pharmacist authoring work — only structural pattern matching ships in this sub-task; the full library lands in a separate authoring workstream

**Sub-Task E: ObservationLayer API skeleton (deferred — flag explicitly)**
- The Addendum §2.3 specifies a multi-surface ObservationLayer gRPC service.
- This sub-task only writes the **proto IDL** and an in-process Go interface; the gRPC server build is deferred to Step 10 (kb-33 build) or later because no consumer surface exists yet to validate the contract against.

- [ ] **Step 3: Commit the new plan**

```bash
git add docs/superpowers/plans/2026-05-11-cape-substrate-prerequisites.md
git commit -m "docs(plans): CAPE substrate prerequisites plan (Failed Intervention History, PRN velocity, Chronology, RecommendationID field)"
git push origin main  # plans land directly on main, not on feature branch
```

(Alternatively: open a small docs-only PR if your team's convention requires it. Plans are markdown — your call.)

---

## Step 5: Author S2 Resident Workspace specification

**Files:**
- Create: `docs/superpowers/plans/2026-05-11-s2-resident-workspace-spec.md`

This is **specification authoring**, not code. The deliverable is an implementation guidelines document modelled on `CAPE_Implementation_Guidelines_v1_1.md` in structure — Parts 0–18 with explicit operational tests, anti-patterns, and risk register.

- [ ] **Step 1: Read the rendering specification's S2 section**

```bash
# Find the Decision Packet Rendering doc
find /Volumes/Vaidshala/cardiofit -name "Decision_Packet_Rendering*" -type f 2>/dev/null
# Read its S2 section (likely Part 2 or similar)
```

Expected: the existing Decision Packet Rendering guidelines specify S2 rendering rules at a high level. The S2 implementation guidelines doc fills in the operational detail.

- [ ] **Step 2: Author the spec covering these sections**

Required sections (one sitting):
- Part 0: Operational test ("does this design improve the engine's answer to the pharmacist's per-resident-review question?")
- Part 1: Design philosophy (8 principles, matching CAPE structure)
- Part 2: Surface architecture — how S2 consumes from ObservationLayer (per Addendum §2.3), how it integrates with worklist entries (Step 9's kb-33 service)
- Part 3: Data composition — substrate observations, trajectories, pending recommendations, restraint signals, failed intervention history, goals-of-care state
- Part 4: Drill-through patterns — one-click to substrate evidence, one-click to kb-32 craft engine, one-click to complex resident workspace (if active)
- Part 5: Brevity budget and visual hierarchy
- Part 6: Failed Intervention History rendering (consumes Step 4 Sub-Task B)
- Part 7: Pharmacist controls
- Part 8: Integration with other surfaces (S1 worklist, S3 GP comms, S5 standard 5)
- Part 9: Complex resident workspace activation criteria + transition
- Part 10: Performance metrics (per-resident-review time, drill-through usage, time-to-action)
- Part 11: Anti-patterns
- Part 12: Risks and mitigations
- Part 13: Implementation sequencing (Weeks within the overall roadmap)

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/plans/2026-05-11-s2-resident-workspace-spec.md
git commit -m "docs(spec): S2 Resident Workspace Implementation Guidelines v1.0"
git push origin main
```

**Parallelisation note:** Steps 5, 6, 7, 8 are all spec-authoring tasks. They have no dependencies on each other (each only depends on existing guidelines docs). One author can sequence them; a team with multiple authors can parallelise.

---

## Step 6: Author S3 GP Communication Hub specification

**Files:**
- Create: `docs/superpowers/plans/2026-05-11-s3-gp-communication-hub-spec.md`

- [ ] **Step 1: Author the spec**

Same structure as Step 5 (Parts 0–13). Specific S3 content:
- Asynchronous-first delivery (no synchronous-call expectation)
- Audience adaptation for GP audience (consumes from kb-32 craft engine's per-GP framing observer)
- Mobile-primary form factor — explicit constraints on token-budget and interaction patterns
- Response capture workflow: accept / modify / decline with structured reason codes (consume from kb-32 override taxonomy aligned in Step 3 Task 4)
- Outcome observation linking — closing the recommendation lifecycle loop
- ACOP §15B / §10 messaging compliance considerations
- Integration with EvidenceTrace for audit defensibility
- Performance metrics: GP response rate, time-to-response, modification rate

- [ ] **Step 2: Commit**

```bash
git add docs/superpowers/plans/2026-05-11-s3-gp-communication-hub-spec.md
git commit -m "docs(spec): S3 GP Communication Hub Implementation Guidelines v1.0"
git push origin main
```

---

## Step 7: Author RMMR Workflow specification

**Files:**
- Create: `docs/superpowers/plans/2026-05-11-rmmr-workflow-spec.md`

- [ ] **Step 1: Author the spec**

Same structure (Parts 0–13). Specific RMMR content:
- RMMR program definition (annual + 3-month follow-up + 6-month follow-up cadence per ACOP rules)
- Pre-RMMR data assembly: which substrate observations get bundled, in what order
- RMMR structured output format: the deliverable the pharmacist produces, the GP receives, and the ACQSC audit can review
- Scheduling integration with kb-19 protocol orchestrator
- ACOP funding mechanics: claim generation, supporting evidence pack
- Integration with kb-32 craft engine (each RMMR can spawn multiple recommendations)
- Integration with S5 Standard 5 evidence panel
- Performance metrics: RMMR completion rate, follow-up adherence, recommendations-per-RMMR ratio
- Anti-patterns specific to RMMR (e.g., template-completion-as-clinical-work)

- [ ] **Step 2: Commit**

```bash
git add docs/superpowers/plans/2026-05-11-rmmr-workflow-spec.md
git commit -m "docs(spec): RMMR Workflow Implementation Guidelines v1.0"
git push origin main
```

---

## Step 8: Author CPD + AHPRA Records specification (or skip)

**Files:**
- Create: `docs/superpowers/plans/2026-05-11-cpd-ahpra-records-spec.md`

**Decision point:** the previous gap analysis flagged this as potentially skippable if the existing code-level plan suffices. Make the call before authoring.

- [ ] **Step 1: Decide whether to author**

Read the existing CPD code coverage:
```bash
find /Volumes/Vaidshala/cardiofit -path "*cpd*" -o -path "*ahpra*" 2>/dev/null | head -10
grep -rn "CPD\|AHPRA" docs/superpowers/plans/2026-05-07-phase-1b-self-visibility-surfaces.md | head -10
```

If the code coverage (Phase 1b Tasks 14 + Phase 1b-completion Task 6 per prior summaries) implements the CPD tagger, AHPRA record generator, and reflective writing surface to a usable degree, **skip this step** and proceed to Step 9. Document the skip:
```bash
echo "Step 8 skipped per <date>: CPD/AHPRA code-level plan deemed sufficient. 30 June 2026 cliff feature requirement met by existing implementation. Authoring deferred to post-pilot if pilot evidence indicates spec gaps." >> docs/superpowers/plans/2026-05-11-pre-pilot-execution-roadmap.md
```

If the code coverage has gaps (e.g., the reflective writing surface isn't built, or AHPRA-format export is incomplete), author the spec covering Parts 0–13 with content matching v3.0 §10 + §12.

- [ ] **Step 2: Commit (if authored)**

```bash
git add docs/superpowers/plans/2026-05-11-cpd-ahpra-records-spec.md
git commit -m "docs(spec): CPD + AHPRA Records Implementation Guidelines v1.0"
git push origin main
```

---

## Step 9: Execute CAPE substrate prerequisites sub-plan

**Sub-plan:** `docs/superpowers/plans/2026-05-11-cape-substrate-prerequisites.md` (authored in Step 4)

Execute via subagent-driven-development, one sub-task at a time, with two-stage review after each.

- [ ] **Step 1: Create feature branch**

```bash
cd /Volumes/Vaidshala/cardiofit
git checkout main
git pull origin main
git checkout -b feat/cape-substrate-prerequisites
```

- [ ] **Step 2: Execute Sub-Task A — Metadata.RecommendationID field**

This is the smallest sub-task and unblocks `/v1/explain` citations immediately. Land first.

Dispatch implementer per sub-plan Sub-Task A. Verify: `go test -race ./internal/api/...` in kb-32 passes; `TestExplain_RegistryError_DegradesGracefully` now actually exercises the citation lookup branch (it was previously vacuous per the Phase 3 Task 4 quality review).

- [ ] **Step 3: Execute Sub-Task B — Failed Intervention History**

Dispatch implementer. New package `shared/v2_substrate/intervention_history/`. Verify Postgres migration applies cleanly and integration with kb-32 override capture writes records.

- [ ] **Step 4: Execute Sub-Task C — PRN velocity primitives**

Dispatch implementer. Verify the 5 severity thresholds match CAPE Part 3.2 exactly.

- [ ] **Step 5: Execute Sub-Task D — Instability Chronology**

Dispatch implementer. Verify the two starter patterns (VolumeContractionCascade, SedationCascade) match the structural definitions in Addendum §3.2 and CAPE Part 3.2.

- [ ] **Step 6: Execute Sub-Task E — ObservationLayer proto IDL only**

Dispatch implementer. Only the `.proto` file + generated Go stubs + an in-process Go interface; no gRPC server. Verify the interface compiles cleanly when imported by a downstream package (e.g., kb-32).

- [ ] **Step 7: PR and merge**

```bash
gh pr create --base main --head feat/cape-substrate-prerequisites \
  --title "CAPE substrate prerequisites: RecommendationID + Failed Intervention History + PRN velocity + Chronology" \
  --body "Closes Step 9 of 2026-05-11-pre-pilot-execution-roadmap.md. Per CAPE v1.1 Parts 3.2 + 4.3 + 7.3 and Addendum §3."
gh pr merge --merge --delete-branch=false <PR_NUMBER>
```

**Gate:** Step 10 (kb-33 build) cannot start until this merges. kb-33 imports all four substrate primitives.

---

## Step 10: Author and execute kb-33-triage-engine implementation plan

**Spec:** `docs/superpowers/plans/CAPE_Implementation_Guidelines_v1_1.md` (the v1.1 guidelines document itself is the spec).
**Sub-plan to author:** `docs/superpowers/plans/2026-05-11-kb-33-triage-engine-build.md`

This is the largest workstream on the roadmap — CAPE v1.1 Part 16 sizes it at 7 weeks of implementation (the document's "Weeks 18–24" block).

- [ ] **Step 1: Author the kb-33 build plan**

Translate CAPE v1.1 Part 16 (Weeks 18–24) into a subagent-driven-development plan with one task per week's deliverable:

| Plan task | CAPE Part 16 reference | Substrate dependency |
|---|---|---|
| Task 1: Service scaffold | Week 18 | Phase 3-tightened ethics-monitoring (precedent service pattern) |
| Task 2: Storage + event subscriptions | Week 18 | Postgres + Layer 2 substrate event bus |
| Task 3: Five-layer scoring engine | Week 18 | None (pure logic) |
| Task 4: Failed Intervention History wiring | Week 18 | Step 9 Sub-Task B |
| Task 5: Signal class definitions across 5 layers | Week 19 | None (pure data) |
| Task 6: PRN escalation velocity detection | Week 19 | Step 9 Sub-Task C |
| Task 7: Sedation drift pattern detection | Week 19 | Substrate state machines |
| Task 8: Detection logic per signal class | Week 19 | Phase 1c pattern_detection |
| Task 9: Substrate reference integration | Week 19 | EvidenceTrace |
| Task 10: Velocity + acceleration primitives | Week 20 | (extend Step 9 Sub-Task C if needed) |
| Task 11: Z-score outlier detection | Week 20 | None (pure stats) |
| Task 12: Multi-parameter composition | Week 20 | Layer 1+2 detections |
| Task 13: Layer 4 (Intervention Opportunity) scoring | Week 20 | Step 9 Sub-Task B (veto factors) |
| Task 14: Veto factor logic | Week 20 | Step 9 Sub-Task B + restraint signaler |
| Task 15: Worklist sizing logic with time estimation | Week 21 | Estimated review time function per CAPE Part 5.2 |
| Task 16: Pharmacist control handlers (Open, Defer, Mark Considered, Promote, Override) | Week 21 | None |
| Task 17: EvidenceTrace integration | Week 21 | Phase 0.1 EvidenceTrace |
| Task 18: Failed intervention documentation flow | Week 21 | Step 9 Sub-Task B |
| Task 19: Calibration learning scaffold | Week 21 | None (scaffold only — full capability is post-pilot) |
| Task 20: Pattern detector implementations | Week 22 | Cross-resident substrate queries |
| Task 21: Coordination support | Week 22 | Facility-level aggregations |
| Task 22: "Everything looks off" mode | Week 22 | Step 9 Sub-Task D (Instability Chronology for facility-level rendering) |
| Task 23: Reasoning library content integration | Week 23 | Senior consultant pharmacist authoring (operational, not code) |
| Task 24: Progressive disclosure rendering | Week 23 | Layer-score primitives |
| Task 25: 10 performance metrics per CAPE Part 10 | Week 23 | Metric storage |
| Task 26: ERM quarterly review integration | Week 23 | Ethics-monitoring service (Phase 3-tightened Task 1) |
| Task 27: Cross-component integration tests | Week 24 | All previous tasks |
| Task 28: Single-score collapse rejection tests (CAPE Part 14 Cat 8) | Week 24 | UI rendering layer (mock acceptable) |
| Task 29: Engine boundary tests (CAPE Part 14 Cat 9) | Week 24 | All output paths |
| Task 30: External clinical informatics UX review | Week 24 | Operational (external, not code) |
| Task 31: Pilot pharmacist user testing (3 pharmacists × 1 week) | Week 24 | Operational |
| Task 32: Calibration baseline tuning | Week 24 | Performance metrics from Task 31 |

Each task gets bite-sized TDD detail per the writing-plans skill rules.

- [ ] **Step 2: Commit the kb-33 build plan**

```bash
git add docs/superpowers/plans/2026-05-11-kb-33-triage-engine-build.md
git commit -m "docs(plans): kb-33-triage-engine (CAPE) implementation plan"
git push origin main
```

- [ ] **Step 3: Create kb-33 build branch and execute**

```bash
git checkout -b feat/kb-33-triage-engine-build
```

Execute via subagent-driven-development. **This is 7 weeks of work.** Use checkpoint reviews per CAPE Part 16's weekly granularity. Tasks 30 and 31 (external review, pilot pharmacist testing) are operational deliverables and may require pausing the code stream while they run.

- [ ] **Step 4: PR and merge**

When all 32 tasks complete and CAPE Part 10 performance metrics are baselined:
```bash
gh pr create --base main --head feat/kb-33-triage-engine-build \
  --title "kb-33-triage-engine (CAPE v1.1): clinical attention prioritisation engine" \
  --body "Implements CAPE v1.1 Part 16 Weeks 18-24. Five-layer scoring, signal taxonomy, capacity calibration, pharmacist controls, facility pattern detection, performance metrics."
gh pr merge --merge --delete-branch=false <PR_NUMBER>
```

**Gate:** Step 11 (pilot launch) cannot start until kb-33 merges AND CAPE Part 10 performance baseline is established AND operational deliverables in Step 11 are in place.

---

## Step 11: Pre-pilot readiness gate verification

**Files:**
- No code changes. Verification checklist.

This is the consolidated pre-pilot acceptance gate from `2026-05-09-phase-3-ethics-monitoring-and-pre-pilot-gates.md` §"Pre-pilot acceptance gate (consolidated)" PLUS the operational deliverables.

- [ ] **Step 1: Code gate — Phase 2-completion items**

Verify, by reading the relevant code and running test suites:
- [ ] PostgresSubstrateClient replaces inMemorySubstrateClient placeholder
- [ ] SubstrateBackedScorer replaces DefaultAppropriatenessSource placeholder
- [ ] PostgresCitationRegistry + Stage 7 EvidenceTrace emission live
- [ ] Override taxonomy vocabulary aligned with Guidelines Part 5 (clinical informatics sign-off recorded)
- [ ] PDP middleware mounted on `/v1/craft/draft` and `/v1/craft/override`
- [ ] End-to-end integration test with all-Postgres-backed dependencies green

- [ ] **Step 2: Code gate — Phase 3-tightened items**

Verify:
- [ ] Ethics-monitoring service running daily/weekly/monthly EBA jobs against pattern_detection primitives
- [ ] Demographic stratification pipeline producing pre-computed bias_stratification_results
- [ ] Capacity + consent gate operational in kb-32 pipeline; restrictive-practice recommendations are consent-gated
- [ ] Layer 4 `/v1/explain` endpoint returning complete audit trail for every decision
- [ ] CI invariance gates passing (frame-vs-content + override pathway availability tests green)

- [ ] **Step 3: Code gate — CAPE prerequisites + engine**

Verify:
- [ ] `decision_metadata.Metadata.RecommendationID` field populated end-to-end
- [ ] Failed Intervention History substrate integrated with override capture
- [ ] PRN velocity primitives consumed by CAPE Layer 1 detection
- [ ] Instability Chronology assembly working for sample residents
- [ ] kb-33-triage-engine serving worklists with five-layer scoring

- [ ] **Step 4: Operational gate — Ethics + governance deliverables**

Per `Ethical_Architecture_Implementation_Guidelines_v1.md` and Phase 3-tightened §"Plus operational deliverables":
- [ ] Ethics Steering Committee constituted with first meeting completed
- [ ] Pharmacist Advisory Group recruited with first quarterly meeting scheduled
- [ ] External ethics auditor identified and engaged
- [ ] Incident response runbook published with on-call rotation in place
- [ ] Plain-language transparency summaries authored and reviewed (Ethical Architecture §13.4)
- [ ] Aboriginal community engagement protocol established (Ethical Architecture §7.5)

- [ ] **Step 5: Operational gate — Commercial readiness**

- [ ] Anchor pharmacy chain pilot agreement signed
- [ ] ACOP funding flow established (Tier 1 funding pattern)
- [ ] RACH pilot site agreements signed (for Phase 2 co-deployment per Step 12)
- [ ] AHPRA / APC RPL evidence pack generation tested with one pilot pharmacist before 30 June 2026

- [ ] **Step 6: Document gate completion**

```bash
cat >> docs/superpowers/plans/2026-05-11-pre-pilot-execution-roadmap.md <<EOF

## Pre-pilot gate verification record

Verified on: <DATE>
Verifier: <NAME>

Code gates: <PASS/FAIL with notes>
Operational gates: <PASS/FAIL with notes>
Commercial readiness: <PASS/FAIL with notes>

Overall: <READY FOR PILOT LAUNCH / NOT READY (with blocker list)>
EOF
git commit -am "docs(roadmap): pre-pilot gate verification record"
git push origin main
```

**Gate:** Step 12 (pilot launch) starts when this verification records "READY FOR PILOT LAUNCH."

---

## Step 12: Pilot Phase 1 launch + parallel Tier 2 spec authoring

**Files:**
- No code changes for launch itself. Tier 2 spec authoring tasks listed below.

Pilot Phase 1 per `Pilot Design` doc: Months 1–8, pharmacist worklist surface, anchor pharmacy chain enterprise tier buyer, ACOP funding flow, CAPE substrate accumulation, RIR baseline + target of +12 percentage points.

- [ ] **Step 1: Pilot launch operational checklist**

Outside this roadmap's code scope. Run the pilot-design playbook.

- [ ] **Step 2: During Months 1–8, author Tier 2 spec docs**

These follow pilot evidence accumulation per the prior gap analysis. One sitting each:
- [ ] `docs/superpowers/plans/2026-MM-DD-s5-standard-5-evidence-panel-spec.md`
- [ ] `docs/superpowers/plans/2026-MM-DD-instability-chronology-rendering-spec.md` (the full implementation spec on top of Step 9 Sub-Task D's structural foundation)
- [ ] `docs/superpowers/plans/2026-MM-DD-complex-resident-workspace-spec.md` (depends on senior consultant pharmacist input on concern vectors)

- [ ] **Step 3: Months 6–12 overlapping — author and execute Phase 2 pilot (RACH operational view)**

Per CAPE v1.1 Addendum §4.1:
- [ ] Author `docs/superpowers/plans/2026-MM-DD-rach-operational-view-spec.md`
- [ ] Author `docs/superpowers/plans/2026-MM-DD-rach-operational-view-build.md`
- [ ] Execute the build plan

This is the second consumer of the ObservationLayer API skeletoned in Step 9 Sub-Task E. Implementing it forces the ObservationLayer API to be fully built — that's the moment the multi-surface architecture becomes real rather than aspirational.

---

## Out of scope for this roadmap (post-pilot, Year 2+)

These workstreams are explicitly deferred. Documented here so they're not forgotten:

- **CAPE adaptive capabilities** (wearables, NLP, predictive trajectory, per-pharmacist calibration learning) — gated on pre-Phase-4 architecture audit per Addendum §5.4
- **KB-29 reasoning primitive extraction** — gated on accumulated template usage data
- **Family Communication Scaffolding surface** — follows RACH surface stabilisation
- **S7 Audit/Regulator interface** — follows Phase 2 substrate accumulation
- **S6 Hospital Handoff surface** — follows transition signal taxonomy validation
- **Pharmacy Chain Enterprise Tier dashboard** — follows Phase 1 + Phase 2 substrate

---

## Self-review checklist (per writing-plans skill)

- [x] Spec coverage: every "Phase 4" / "Phase 3 CAPE" / "Layer 2/3 Phase 3" reference disambiguated to a roadmap step number
- [x] Placeholder scan: no "TBD" / "TODO" / "fill in details" — every step has either a concrete command, a concrete file path, or a clear authoring task with section list
- [x] Type consistency: step numbers stable throughout; sub-plan filenames stable throughout
- [x] Decomposition acknowledged: master roadmap with sub-plan references, not one monolithic plan
- [x] Gates documented: every step that blocks the next states the gate condition explicitly
- [x] Phase-numbering collisions resolved: decision matrix at top maps every legacy phase label to a step number
