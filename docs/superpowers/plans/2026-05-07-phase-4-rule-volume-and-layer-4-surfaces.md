# Rule Volume + Layer 4 Surfaces Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the two largest calendar consumers in V1 closure: (a) raise the CQL rule library from 84 defines / 78 specs (22% of plan target) to ~250 by binding ADG 2025, completing STOPP/START coverage, and authoring class-specific Ramsey-baseline rules; (b) ship the Layer 4 surfaces that make the MVP exit criterion demonstrable to buyers — Worklist, Resident Workspace, GP Communication Hub, Standard 5 evidence panel, PHARMA-Care dashboard, and AN-ACC scaffolding (kb-28). These are the most calendar-heavy items per the implementation plan and the most parallelisable; rule authoring runs alongside frontend engineering.

**Architecture:** Two streams running in parallel. **Stream A — Rule volume:** clinical authors write `*.cql.spec.yaml` + `*.cql` files following the existing `cql-toolchain` patterns; the validator + CompatibilityChecker + governance promoter (already 244-test passing) handles the rest. **Stream B — Layer 4 surfaces:** Angular workspace under `frontend/` (or wherever the existing app shell lives) consumes the v2 substrate REST APIs through Phase 1's permission middleware. Five surfaces, each rendered against one or more PharmacistView/EmployerView/RACHView/RegulatorView slices.

**Tech Stack:** Stream A — CQL + cql-toolchain (Python pytest harness, already production-shaped). Stream B — Angular 17, Tailwind, Apollo client (per existing apollo-federation gateway), depends on Plans 0.1–0.5 + Phases 1–3.

---

## File Structure

**Stream A (rule volume):**
- `shared/cql-libraries/Tier2_Deprescribing_*.cql` — ~204 new defines across STOPP/START/Beers/AU PIMs/ADG categories
- `shared/cql-toolchain/specs/tier2_*/` — corresponding `*.spec.yaml` files
- `shared/cql-toolchain/fixtures/tier2_*/` — fixtures per spec

**Stream B (Layer 4):**
- `frontend/src/app/worklist/` — Worklist surface (Plan 0.1 Recommendation lifecycle UI)
- `frontend/src/app/resident-workspace/` — Resident Workspace (Plan 0.1+0.3 substrate state)
- `frontend/src/app/gp-hub/` — GP Communication Hub
- `frontend/src/app/standard5-panel/` — audit evidence panel
- `frontend/src/app/pharma-care-dashboard/` — PHARMA-Care five-domain indicators
- `frontend/src/app/an-acc-view/` — AN-ACC reassessment workflow scaffold
- `frontend/src/app/self-visibility/` — pharmacist's own-data dashboard (Phase 1 PharmacistView)

**New service:**
- `kb-28-an-acc-revenue-assurance/` — basic scaffold per existing kb-* pattern

---

## Stream A — Tier 2 rule volume (~14 weeks; one rule = ~half-day for an experienced author)

The pattern is well-established: each rule is a YAML spec + a CQL define + 3 fixtures. The cql-toolchain validator + CompatibilityChecker enforce structural correctness. The work is primarily clinical authoring, not engineering.

### Task A1: ADG 2025 binding (~48 rules, ~24 days)

**Prerequisite:** ADG 2025 licensing review complete (P0 commercial action per audit). 5 PDFs already landed; Pipeline-2 extraction runbook exists.

- [ ] **Step 1: Run ADG 2025 Pipeline-2 extraction**

```bash
cd shared/cql-toolchain
python pipeline2/extract.py --source ADG-2025 --output specs/tier2_adg2025/
```
Expected: ~48 spec stubs in `specs/tier2_adg2025/`.

- [ ] **Step 2-N: Author each rule following the established pattern**

For each spec stub, the clinical author:
1. Reviews the extracted recommendation
2. Hand-writes the CQL define using existing helper libraries (`SuppressionHelpers`, `MonitoringHelpers`, `Vaidshala.Substrate.*`)
3. Writes 3 fixtures: positive (rule fires), negative (rule does not fire), edge case (suppression engaged)
4. Runs the validator: `pytest specs/tier2_adg2025/test_<rule_id>.py -v`
5. Runs CompatibilityChecker: `python cql-toolchain/check_compatibility.py --rule <id>`
6. Commits: `git commit -m "feat(tier-2): ADG 2025 rec <id> — <short title>"`

The pattern is repetitive across 48 rules; expect ~24 working days of clinical authoring with engineering support for tricky CQL idioms.

- [ ] **Step Final: Run full toolchain regression**

```bash
cd shared/cql-toolchain
python -m pytest -v 2>&1 | tail -10
```
Expected: ≥292 tests pass (244 existing + 48 new × ~1 test each).

```bash
git commit -m "feat(tier-2): ADG 2025 binding complete (48 rules)"
```

### Task A2: STOPP/START full coverage closure (~80 rules already loaded as data; ~60 new CQL defines, ~30 days)

Per audit: STOPP v3 (80) + START v3 (40) "all loaded" but not all have CQL defines. Author the missing CQL definitions following the same per-rule pattern.

- [ ] **Step 1-N: For each STOPP/START criterion without a CQL define, author it**

```bash
cd shared/cql-toolchain
python lint_coverage.py --tier 2 --source STOPP --output missing_stopp.txt
# 60 rules listed; author each
git commit -m "feat(tier-2): STOPP/START coverage closure"
```

### Task A3: Class-specific Ramsey-baseline rules (~20 rules, ~10 days)

Per v3 §11 line 591. Targets: colecalciferol, calcium, PPIs, polypharmacy. Each authored against the corresponding ADG/STOPP/Beers source with explicit baseline-aware monitoring.

- [ ] **Step 1-N: Author each class-specific rule with explicit Ramsey baseline reference in the spec**

```bash
git commit -m "feat(tier-2): Ramsey-baseline class-specific rules"
```

### Task A4: Tier 2 exit verification

- [ ] **Step 1: Verify total rule count meets target**

```bash
cd shared/cql-toolchain
python coverage_report.py --tier 2
```
Expected: ≥225 Tier-2 rules.

- [ ] **Step 2: Run full integration suite**

```bash
python -m pytest -v 2>&1 | tail -20
```
Expected: All passing.

- [ ] **Step 3: Run override-rate analytics against synthetic fixture pack**

```bash
python override_rate_synthetic.py --target-pct 5
```
Expected: Synthetic override rate <5%.

```bash
git commit -m "milestone(tier-2): rule volume target met (225 rules)"
```

---

## Stream B — Layer 4 surfaces (~12-16 weeks)

Each surface follows the same construction pattern: data contract → API client → component skeleton → component features → permission middleware integration → end-to-end test → commit.

### Task B1: Worklist surface (~3 weeks)

**Files:**
- Create: `frontend/src/app/worklist/worklist.component.ts/.html/.scss`
- Create: `frontend/src/app/worklist/worklist.service.ts`
- Create: `frontend/src/app/worklist/worklist.component.spec.ts`

Renders pharmacist's Recommendation pipeline (Plan 0.1) with deferred-state callouts (Plan 0.1 escalator events surfaced as "needs attention" cards). Permission scope: pharmacist self-view (Phase 1).

- [ ] **Step 1: Define data contract**

```typescript
// worklist.types.ts
export interface WorklistItem {
  recommendationId: string;
  residentName: string;
  state: RecommendationState;
  type: 'stop' | 'monitor' | 'dose_change' | 'add';
  urgency: 'red' | 'amber' | 'green';
  title: string;
  reviewDueAt: Date | null; // populated for deferred items
}
```

- [ ] **Step 2: Write failing component test**

```typescript
describe('WorklistComponent', () => {
  it('renders deferred-overdue items with attention badge', async () => {
    const fixture = TestBed.createComponent(WorklistComponent);
    fixture.componentInstance.items = [{
      ...mockItem(),
      state: 'deferred',
      reviewDueAt: new Date(Date.now() - 24*60*60*1000),
    }];
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.attention-badge')).toBeTruthy();
  });
});
```

- [ ] **Step 3: Implement component**

- [ ] **Step 4: Wire permission middleware**

The Angular service calls `/views/pharmacist/own/pipeline` (Phase 1 endpoint). Backend permission middleware enforces `subject = self`.

- [ ] **Step 5: End-to-end test using Cypress/Playwright**

- [ ] **Step 6: Commit**

```bash
git commit -m "feat(frontend): Worklist surface renders Recommendation pipeline"
```

### Task B2: Resident Workspace (~3 weeks)

Renders for one resident: trajectory panel (baselines from Plan 0.3), MedicineUse list with intent/target/stop, active concerns, care_intensity tag, recent events. Permission scope: pharmacist + RACH view types depending on viewer role.

Same task structure as B1: data contract → test → component → permission wiring → e2e → commit.

```bash
git commit -m "feat(frontend): Resident Workspace surface"
```

### Task B3: GP Communication Hub (~3 weeks)

Closed-loop tracking, recommendation supersession (one recommendation supersedes a prior one when context changes), Smart Form / structured-input integration where prescriber system supports it. Renders the Phase 2 frame-vs-content separated representation: GP sees their framed view; the underlying clinical content invariant is auditable but not exposed in the UI.

```bash
git commit -m "feat(frontend): GP Communication Hub with Smart Form integration"
```

### Task B4: Standard 5 evidence panel (~2 weeks)

Audit-ready evidence pack generator + UI. Pulls EvidenceTrace lineage for any resident + medication and renders as a Standard 5-aligned evidence document. Output: PDF + structured FHIR Bundle for regulator-grade artefacts.

```bash
git commit -m "feat(frontend): Standard 5 evidence panel + PDF generator"
```

### Task B5: PHARMA-Care dashboard (~2 weeks)

Five-domain indicators per the framework (configurable per v3 §11 + Risk 11). Pulls from KB-13 indicator scaffold (already 11 AU QI Program indicators seeded; 5 PHARMA-Care placeholder rows). Permission scope: RACH + employer view-types.

```bash
git commit -m "feat(frontend): PHARMA-Care five-domain dashboard"
```

### Task B6: AN-ACC scaffold (kb-28) (~2 weeks)

New backend service + frontend view. Honest framing per v3 §12 line 691: workflow-support module surfacing medication-complexity change signals during AN-ACC reassessment, not a pharmacy-attributable revenue line.

- [ ] **Step 1: Scaffold kb-28 service**

```bash
mkdir -p backend/shared-infrastructure/knowledge-base-services/kb-28-an-acc-revenue-assurance/{cmd/server,internal/{api,signals},config}
cd kb-28-an-acc-revenue-assurance
go mod init kb28
```

- [ ] **Step 2-5: Implement medication-complexity change signal queries (drug count delta, anticholinergic delta, falls events delta), expose via REST, wire frontend, commit**

```bash
git commit -m "feat(kb-28): AN-ACC reassessment workflow scaffold"
```

### Task B7: Pharmacist self-visibility dashboard (~1 week; Phase 1 backend already exists)

Renders own RIR trajectory, recommendation pipeline, CPD-relevant cases (Phase 3 Task 5), per-GP acceptance patterns (Phase 2 Task 7 with toxicity guards on aggregation upward).

```bash
git commit -m "feat(frontend): Pharmacist self-visibility dashboard"
```

### Task B8: End-to-end MVP exit-criterion demo

Exercise: ACOP pharmacist logs in → Worklist shows top 5 priority residents → opens Resident Workspace for one → reviews medication timeline → uses craft engine (Phase 2) to draft recommendation → recommendation appears in GP Communication Hub → GP responds → RIR updates on self-visibility dashboard → Standard 5 evidence panel reflects the chain.

- [ ] **Step 1-5: Write Playwright e2e against staging deployment, run, commit**

```bash
git commit -m "test(e2e): MVP exit-criterion demo — 5-min review with full evidence chain"
```

---

## Final integration milestone

After both streams complete:

- [ ] **Run all CQL toolchain tests**

```bash
cd shared/cql-toolchain && python -m pytest -v 2>&1 | tail
```
Expected: All passing, ≥225 Tier-2 rules executable through Plan 0.5's HAPI runtime.

- [ ] **Run full backend integration suite**

```bash
cd backend/shared-infrastructure/knowledge-base-services
go test ./... -v 2>&1 | tail -20
```
Expected: All packages passing.

- [ ] **Run frontend e2e suite**

```bash
cd frontend && npx playwright test
```
Expected: MVP-exit-criterion demo passes.

- [ ] **Tag the V1 release**

```bash
git tag v1.0.0-rc1 -m "V1 release candidate: substrate + state machines + craft + ethical guards + Layer 4"
```

---

## Spec coverage

- [x] Tier-2 deprescribing rule volume from 21 → 225 — Stream A (Tasks A1-A4)
- [x] Worklist surface (MVP-3 render) — B1
- [x] Resident Workspace — B2
- [x] GP Communication Hub (V1-3) — B3
- [x] Standard 5 evidence panel (MVP-6) — B4
- [x] PHARMA-Care dashboard (V1-4) — B5
- [x] AN-ACC scaffold (V2-1) — B6
- [x] Pharmacist self-visibility dashboard (MVP-7) — B7
- [x] MVP exit-criterion end-to-end demonstrable — B8

Plan complete and saved.
