# S2 Layer 1 Frontend — Build Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development`. Steps use checkbox (`- [ ]`) syntax. Frontend tasks dispatch to `frontend-architect` (not `backend-architect`).

**Goal:** Implement the S2 Layer 1 frontend per the canonical S2 v1.0 Part 15 frontend component tree, consuming the 17 HTTP routes from `s2-aggregator` shipped in the S2 Layer 1 backend build (merged `bf0c1c6f`). Honors the same Addendum Part 6 authoring discipline as the backend: architectural framework + structural primitives + integration code only; Layer 2–5 visual content deferred to senior consultant pharmacist authoring + pilot evidence.

**Canonical source documents (READ BEFORE TOUCHING ANYTHING):**
- `docs/superpowers/plans/S2_Resident_Workspace_Implementation_Guidelines_v1.md` (1475 lines) — Parts 4 (view layout), 5–13 (per-panel rendering), 14 (form-factor adaptations + accessibility), 15 (frontend component tree).
- `docs/superpowers/plans/S2_Adaptive_Cognition_Architectural_Commitment_Addendum.md` (477 lines) — Part 4 (shared primitives), Part 6 (authoring discipline), Part 8.1 (layer-aware rendering — component hierarchy must support layer-aware composition without rebuild).

**Pre-flight reality (recon 2026-05-11):**
The Next.js app at `vaidshala/clinical-applications/ui/clinician/` has installed `node_modules/`, `.next/` build artifacts, and `.vercel/` deploy config — but no source code. Stack is locked in:

| Layer | Tech | Version |
|---|---|---|
| Framework | Next.js | 14.2.35 (App Router) |
| Runtime | React | 18.3.1 |
| Language | TypeScript | (present in node_modules) |
| Styling | Tailwind CSS | (present in node_modules) |
| Auth | @auth0/nextjs-auth0 | (present in node_modules) |
| Data fetching | @tanstack/react-query | (present in node_modules) |
| Type-checking + linting | @typescript-eslint, eslint-config-next | (present) |

Task 1 scaffolds the app from zero. No legacy source to migrate.

**Branch:** `feat/s2-layer-1-frontend` off `main` (currently at `bf0c1c6f` — S2 Layer 1 backend merged). One commit per task. Push to origin between tasks.

**Authoring discipline (load-bearing — same rules as backend):**
- Claude provides **layout structure, component skeletons, state plumbing, API integration, accessibility scaffolding, test harness**.
- Claude does **NOT** author **visual design** (colours, typography, exact spacing — those are design-lead work) or **clinical content** (Layer 1 panel content is data-driven from s2-aggregator; visual rendering composes that data; cognitive content for Layers 2–5 is deferred to senior pharmacist authoring per Addendum Part 6).
- Component implementations use **Tailwind utility classes** for spacing/layout primitives. Where v1.0 doesn't specify exact visual treatment, comment `// TODO(design lead): visual treatment` and use sensible defaults.
- All clinical claims rendered must trace to a substrate reference passed in from the s2-aggregator response (verification-not-belief principle from v1.0 Part 1 Principle 2 — frontend-side enforcement: every rendered claim component accepts a `substrateRefs` prop, and a runtime DEV-mode warning fires when it's empty).

---

## Architecture

```
                ┌──────────────────────────────────────┐
                │  s2-aggregator backend (port 8200)   │
                │  17 HTTP routes (POST + GET)         │
                └────────────┬─────────────────────────┘
                             │  HTTPS + Auth0 JWT
                             ▼
                ┌──────────────────────────────────────┐
                │  apiClient (fetch wrapper)           │
                │   - Auth0 access-token attachment    │
                │   - Error normalization              │
                │   - SubstrateRef pass-through        │
                └────────────┬─────────────────────────┘
                             │
                             ▼
                ┌──────────────────────────────────────┐
                │  TanStack Query hooks                │
                │   - useResidentWorkspace(id)         │
                │   - usePharmacistAction(...)         │
                │   - useDrillThrough(...)             │
                │   - useSession(...)                  │
                └────────────┬─────────────────────────┘
                             │
                             ▼
                ┌──────────────────────────────────────┐
                │  S2 page route                       │
                │  app/residents/[id]/page.tsx         │
                │  → renders <ResidentWorkspace />     │
                └────────────┬─────────────────────────┘
                             │
                             ▼
                ┌──────────────────────────────────────┐
                │  components/s2/ (14 React components)│
                │  per v1.0 Part 15 frontend tree      │
                └──────────────────────────────────────┘
```

---

## File structure (per v1.0 Part 15 + Next.js 14 App Router conventions)

```
vaidshala/clinical-applications/ui/clinician/
├── package.json                          # restored / recreated
├── tsconfig.json
├── next.config.mjs
├── tailwind.config.ts
├── postcss.config.mjs
├── eslint.config.mjs
├── .env.local.example                    # template
├── src/
│   ├── app/
│   │   ├── layout.tsx                    # root layout with providers
│   │   ├── page.tsx                      # landing → /residents
│   │   ├── providers.tsx                 # Auth0Provider + QueryClientProvider
│   │   ├── residents/
│   │   │   ├── page.tsx                  # S1 worklist (out of scope — placeholder)
│   │   │   └── [residentId]/
│   │   │       ├── page.tsx              # S2 entry point
│   │   │       └── layout.tsx            # workspace layout shell
│   │   └── api/                          # (none — all data from s2-aggregator)
│   ├── components/
│   │   └── s2/                           # per v1.0 Part 15
│   │       ├── ResidentWorkspace.tsx
│   │       ├── ResidentHeader.tsx
│   │       ├── CAPEContextBand.tsx
│   │       ├── NotificationContextBand.tsx
│   │       ├── TrajectoriesPanel.tsx
│   │       ├── TrajectoryChart.tsx
│   │       ├── PendingRecommendationsPanel.tsx
│   │       ├── RecommendationCard.tsx
│   │       ├── RestraintSignalsPanel.tsx
│   │       ├── RestraintSignalCard.tsx
│   │       ├── FailedInterventionHistory.tsx
│   │       ├── GoalsOfCarePanel.tsx
│   │       ├── CareIntensityPanel.tsx
│   │       ├── ComplexActivationOffer.tsx
│   │       ├── PharmacistNotes.tsx
│   │       ├── PharmacistActionsPanel.tsx
│   │       ├── AuditTrailFooter.tsx
│   │       └── primitives/               # shared low-level pieces
│   │           ├── SubstrateRefDrawer.tsx     # drill-through drawer
│   │           ├── ConfidenceChip.tsx
│   │           ├── SeverityBadge.tsx
│   │           ├── EmptyState.tsx
│   │           ├── SparseDataFlag.tsx
│   │           ├── NegativeEvidenceCard.tsx
│   │           └── ClaimWithRef.tsx     # the substrateRefs-mandatory wrapper
│   ├── lib/
│   │   ├── apiClient.ts                  # fetch wrapper + auth
│   │   ├── api/
│   │   │   ├── workspace.ts              # GetResidentWorkspace etc
│   │   │   ├── actions.ts                # 11 pharmacist actions
│   │   │   ├── drillThrough.ts
│   │   │   ├── audit.ts
│   │   │   └── session.ts
│   │   ├── hooks/
│   │   │   ├── useResidentWorkspace.ts
│   │   │   ├── usePharmacistAction.ts
│   │   │   ├── useDrillThrough.ts
│   │   │   └── useSession.ts
│   │   ├── types/
│   │   │   ├── Layer1View.ts             # mirror of backend types
│   │   │   ├── ActionRequest.ts
│   │   │   └── SubstrateRef.ts
│   │   ├── form_factor/
│   │   │   ├── useFormFactor.ts          # desktop|mobile detection hook
│   │   │   └── breakpoints.ts
│   │   └── a11y/
│   │       └── liveRegion.ts             # screen-reader announcements
│   └── styles/
│       └── globals.css
├── tests/
│   ├── unit/                             # Vitest
│   ├── integration/                      # Playwright component
│   └── e2e/                              # Playwright e2e (skip without S2_AGGREGATOR_URL)
└── public/
    └── (favicons, assets)
```

---

## Task sequencing

Same pattern as the backend plan: one commit per task, push between tasks, two-stage review.

1. **Scaffold the Next.js app** from the locked-in stack (recover from empty state)
2. **API client + TanStack Query hooks** for the 17 routes
3. **Type mirrors** (`Layer1View`, action requests, `SubstrateRef`) + the `ClaimWithRef` primitive enforcing verification-not-belief at the React level
4. **App shell + routing + Auth0 + S2 page entry**
5. **ResidentHeader + CAPEContextBand + NotificationContextBand**
6. **TrajectoriesPanel + TrajectoryChart** (the highest-density visual component)
7. **PendingRecommendationsPanel + RecommendationCard**
8. **RestraintSignalsPanel + RestraintSignalCard** (Phase 1 advisory-only rendering)
9. **FailedInterventionHistory + GoalsOfCarePanel + CareIntensityPanel**
10. **ComplexActivationOffer + PharmacistNotes**
11. **PharmacistActionsPanel** (the eleven actions UI; reasoning-capture modal)
12. **AuditTrailFooter + SubstrateRefDrawer** (drill-through UX)
13. **Form-factor adaptations + accessibility audit baseline**
14. **Unit + component tests** (Vitest + Playwright component testing)
15. **E2E test against running s2-aggregator** (Playwright; skip without `S2_AGGREGATOR_URL`)

---

### Task 1: Scaffold Next.js 14 app from empty state

**Goal:** Recover the source tree from the empty `vaidshala/clinical-applications/ui/clinician/` directory. Stack locked in by existing `node_modules/`.

**Files to create:**
- `package.json` — dependencies inferred from `node_modules/` top-level packages; runtime deps: `next@^14.2.35 react@^18.3.1 react-dom@^18.3.1 @auth0/nextjs-auth0 @tanstack/react-query`; dev deps: `typescript @types/node @types/react @types/react-dom tailwindcss postcss autoprefixer eslint eslint-config-next @typescript-eslint/eslint-plugin @typescript-eslint/parser vitest @testing-library/react @testing-library/jest-dom @playwright/test`. Verify each from `node_modules/<name>/package.json` "version" field before pinning.
- `tsconfig.json` — Next 14 App Router defaults: `target: ES2020, lib: [ES2022, DOM, DOM.Iterable], module: ESNext, moduleResolution: bundler, jsx: preserve, strict: true, noUncheckedIndexedAccess: true, paths: { "@/*": ["./src/*"] }, include: src/**/*, exclude: node_modules + .next`
- `next.config.mjs` — App Router; `experimental.serverActions: false` (we use s2-aggregator for all writes)
- `tailwind.config.ts` — content paths covering `./src/**/*.{ts,tsx}`; preset content; `darkMode: 'class'`
- `postcss.config.mjs` — tailwindcss + autoprefixer
- `eslint.config.mjs` — extends `next/core-web-vitals` + `@typescript-eslint/recommended`
- `.env.local.example` — `NEXT_PUBLIC_S2_AGGREGATOR_URL=http://localhost:8200`, `AUTH0_*` placeholders
- `.gitignore` — node_modules, .next, .vercel, .env.local, coverage
- `src/app/layout.tsx` — minimal root layout (html/body, font setup)
- `src/app/page.tsx` — placeholder "redirects to /residents" page
- `src/styles/globals.css` — Tailwind directives + base CSS reset
- `README.md` — `npm install && npm run dev → :3000`; env-var checklist

**Verification:**
- `npm install --no-fund --no-audit` (against existing node_modules) — succeeds or surfaces real missing peer deps
- `npm run build` succeeds (Next.js production build)
- `npm run lint` clean
- `npx tsc --noEmit` clean
- `npm run dev` boots; `curl localhost:3000` returns the placeholder page

- [ ] Step 1: extract version pins from existing `node_modules/<pkg>/package.json`
- [ ] Step 2: author `package.json` + run `npm install --no-fund --no-audit` (no-op if node_modules already satisfies)
- [ ] Step 3: author config files (ts, next, tailwind, postcss, eslint, gitignore, env example)
- [ ] Step 4: author minimal app shell (layout, page, globals.css)
- [ ] Step 5: `npm run build && npm run lint && npx tsc --noEmit`
- [ ] Step 6: commit

```
feat(s2-fe): scaffold Next.js 14 clinician app from empty state (Task 1)
```

---

### Task 2: API client + TanStack Query hooks

**Files:**
- `src/lib/apiClient.ts` — fetch wrapper:
  - Reads `NEXT_PUBLIC_S2_AGGREGATOR_URL`
  - Attaches `Authorization: Bearer <accessToken>` from Auth0 (use `getAccessToken()` from `@auth0/nextjs-auth0`)
  - Normalizes errors: `400 → APIError.kind='validation'`, `401 → APIError.kind='unauthenticated'`, `403 → APIError.kind='forbidden_pdp'` (PDP visibility class denial), `501 → APIError.kind='not_yet_implemented'` (layer-not-implemented sentinel), `502 → APIError.kind='upstream_unavailable'`, others → `APIError.kind='server'`
  - Returns parsed JSON or throws `APIError`
- `src/lib/api/workspace.ts` — typed wrappers: `getResidentWorkspace(req: WorkspaceRequest): Promise<Layer1View>`, `refreshResidentWorkspace(...)`
- `src/lib/api/actions.ts` — 11 typed action wrappers, one per action endpoint
- `src/lib/api/drillThrough.ts` — `getSubstrateObservation`, `getTrajectoryHistory`
- `src/lib/api/audit.ts` — `getS2AuditTrail`
- `src/lib/api/session.ts` — `startSession`, `endSession`
- `src/lib/hooks/useResidentWorkspace.ts` — TanStack Query hook with `queryKey: ['s2', 'workspace', residentId, entryPath]`; staleTime 30s
- `src/lib/hooks/usePharmacistAction.ts` — TanStack Mutation; invalidates `['s2', 'workspace', residentId]` on success
- `src/lib/hooks/useDrillThrough.ts` — TanStack Query for substrate observation lookups
- `src/lib/hooks/useSession.ts` — start/end session mutations

**Test:** Vitest unit tests for `apiClient` error normalization + each typed wrapper round-trip against MSW-mocked server.

- [ ] Steps 1–5: implement + tests + commit

---

### Task 3: Type mirrors + ClaimWithRef primitive

**Files:**
- `src/lib/types/Layer1View.ts` — mirrors backend `aggregation.Layer1View` (will need to construct from Tasks 3–7 backend outputs since Layer1View struct itself is currently empty per Task 9 composition gap; document)
- `src/lib/types/ActionRequest.ts` — eleven action types
- `src/lib/types/SubstrateRef.ts` — `{ source: string; id: string; description: string }`
- `src/lib/types/PharmacistAction.ts` — Action enum matching backend
- `src/components/s2/primitives/ClaimWithRef.tsx` — wrapper component that REQUIRES a `substrateRefs: SubstrateRef[]` prop; renders the wrapped claim; in DEV mode logs `console.warn` if `substrateRefs.length === 0`; in PROD mode renders silently but emits a `data-substrate-warn="empty"` attribute for QA tooling. This is the frontend-side enforcement of verification-not-belief.
- `src/components/s2/primitives/SubstrateRefDrawer.tsx` — drawer component triggered by clicking any ClaimWithRef; fetches via `useDrillThrough`; renders the observation row

**Test:** ClaimWithRef DEV warning fires on empty refs; renders without warning when refs supplied. Snapshot tests for drawer states (loading, error, populated, negative-evidence framing).

- [ ] Steps 1–5: implement + tests + commit

---

### Task 4: App shell + routing + Auth0 + S2 entry route

**Files:**
- `src/app/providers.tsx` — `<Auth0Provider>` + `<QueryClientProvider>` composition
- `src/app/layout.tsx` — updated to wrap with `<Providers>`
- `src/app/residents/page.tsx` — placeholder S1 list ("S1 worklist out of scope — see kb-33"); link to test residents
- `src/app/residents/[residentId]/layout.tsx` — workspace shell; reads `searchParams` for `entryPath` (`worklist | search | notification | cross_reference`)
- `src/app/residents/[residentId]/page.tsx` — S2 entry; calls `useResidentWorkspace`; renders loading/error/data states; mounts `<ResidentWorkspace />` from Task 5

- [ ] Steps 1–4: implement + commit

---

### Task 5: ResidentHeader + CAPEContextBand + NotificationContextBand

Three components per v1.0 Part 4.1 (top region).

- `ResidentHeader.tsx` — name, DOB, care intensity tag (uses kb-20 vocabulary: `active_treatment | rehabilitation | comfort_focused | palliative`), substrate row (eGFR/DBI/ACB/CFS), recent-events row (RecentFall72h, RecentAdmission72h, FrailtyStepIncrease30d), gate banner (RestrictivePracticeActive | CapacityLapse | FamilyDistress)
- `CAPEContextBand.tsx` — renders when `entryPath === 'worklist'`; shows `PrimarySignals[]` with severity badges + substrate refs; "you came here because of [signal A] AND [signal B]"
- `NotificationContextBand.tsx` — renders when `entryPath === 'notification'`; shows the notification reason text

Tests: rendering matrix for each entry path × care intensity × gate state.

- [ ] Steps 1–6: implement + tests + commit

---

### Task 6: TrajectoriesPanel + TrajectoryChart

Per v1.0 Part 5. The highest-density visual component.

- `TrajectoriesPanel.tsx` — grid of trajectory tiles; one per parameter from backend Task 3 (eGFR, DBI, ACB, CFS, weight, BP systolic/diastolic, PRN benzodiazepine/antipsychotic/analgesic)
- `TrajectoryChart.tsx` — small chart per parameter; shows: current value + velocity arrow + baseline-90d reference line + threshold flags. Sparse data (`sparseDataFlag=true`) renders "insufficient data for velocity" with the underlying observations still listed. Use a small charting library (recharts? lightweight d3?) — Tasks owner picks; document choice
- Multi-parameter compositions panel (from backend `MultiParameterComposition[]`): renders the composition label + contributing parameters as a small banner above the trajectory grid

Tests: sparse vs populated; multi-param composition rendering; threshold-flag visibility; click → SubstrateRefDrawer.

- [ ] Steps 1–7: implement + tests + commit

---

### Task 7: PendingRecommendationsPanel + RecommendationCard

Per v1.0 Part 6.

- `PendingRecommendationsPanel.tsx` — list of cards; empty state via `<EmptyState>` primitive; sort order STOP > MONITOR > DOSE_CHANGE > ADD
- `RecommendationCard.tsx` — type tag, urgency badge (red/amber/green), Layer 1 framing body, lifecycle state badge, 5-dim Assessment scores as `<ConfidenceChip>`, paired restraint signal badge (if present), hold reason (if present), three action buttons (Send to GP / Override / Defer) always visible per Phase 3 `override_pathway_test.go` invariant, citation drawer toggle

Tests: every render path × each lifecycle state; empty state renders ClaimWithRef anchoring "no pending recommendations"; sort order assertion.

- [ ] Steps 1–7: implement + tests + commit

---

### Task 8: RestraintSignalsPanel + RestraintSignalCard

Per v1.0 Part 7 (Phase 1 advisory-only).

- `RestraintSignalsPanel.tsx` — list of active restraint signals
- `RestraintSignalCard.tsx` — signal type, severity, paired recommendation, acknowledgment workflow modal; safety-critical bypass requires mandatory reasoning capture in modal; transition criteria displayed informationally even when satisfied (NO auto-suppression — v1.0 Part 7.3 invariant)

Tests: acknowledgment workflow happy path; bypass mandatory reasoning enforcement; transition-criteria-satisfied still shows signal.

- [ ] Steps 1–6: implement + tests + commit

---

### Task 9: FailedInterventionHistory + GoalsOfCarePanel + CareIntensityPanel

Per v1.0 Part 8 + Part 9.

- `FailedInterventionHistory.tsx` — list of FIR cards; gap-badge surface when `retrievalAvailable=false` (per Step 4 Task B documented gap); active vs expired veto visual distinction; inline link to vetoed recommendation
- `GoalsOfCarePanel.tsx` — current state + history; freshness flags (6mo soft, 12mo strong per backend Task 5)
- `CareIntensityPanel.tsx` — current tag + history; sparse-data flag
- Goals conflict surfacing: when `<RecommendationCard>` has an associated GoalsConflict, render a conflict ribbon on the card

Tests: gap-badge presence/absence; freshness threshold rendering; conflict ribbon visibility.

- [ ] Steps 1–6: implement + tests + commit

---

### Task 10: ComplexActivationOffer + PharmacistNotes

Per v1.0 Part 11.

- `ComplexActivationOffer.tsx` — appears when backend signals activation criteria met (CFS≥6 + ≥3 high-risk meds + concurrent trajectory declines); offers "Open Complex Resident Workspace?" — clicking calls `usePharmacistAction({action: 'open_complex_workspace'})`; per Phase 1 the backend returns 501 "layer 3 not yet implemented" sentinel; frontend renders a tasteful "Layer 3 not yet available — coming after senior pharmacist authoring of concern vectors" with a link to the Addendum
- `PharmacistNotes.tsx` — list of session notes; add-note modal with free-text input

Tests: offer rendering when criteria met; 501 sentinel handled gracefully; note add round-trip.

- [ ] Steps 1–5: implement + tests + commit

---

### Task 11: PharmacistActionsPanel — the eleven actions UI

Per v1.0 Part 12.

- `PharmacistActionsPanel.tsx` — dispatcher for the eleven actions; per-action modal:
  - `open` → no modal, fires `usePharmacistAction({action: 'open'})`
  - `modify` → reasoning modal (mandatory, min 10 chars)
  - `defer` → reasoning modal (optional)
  - `override` → reasoning modal + override taxonomy dropdown (dual-vocab snake_case + 3-letter codes from Phase 2-completion Task 5)
  - `mark_reviewed` → confirm modal
  - `flag_for_follow_up` → no modal
  - `add_note` → note text modal
  - `open_complex_workspace` → handled in Task 10
  - `drill_into_substrate` → opens SubstrateRefDrawer
  - `acknowledge_restraint_signal` → reasoning modal (optional)
  - `invoke_safety_critical_bypass` → reasoning modal (mandatory) + audit-priority badge

Tests: each action's modal validation; submit path; error rendering.

- [ ] Steps 1–13: implement (one sub-step per action) + tests + commit

---

### Task 12: AuditTrailFooter + SubstrateRefDrawer polish

Per v1.0 Part 13.5 (audit trail surfacing in S2) + Part 10 (drill-through).

- `AuditTrailFooter.tsx` — collapsible footer showing recent EvidenceTrace events scoped to this session × resident; reads from `GET /v1/s2/audit/:resident_id`
- `SubstrateRefDrawer.tsx` polish from Task 3 stub: full negative-evidence framing per v1.0 Part 10.4; substrate confidence visible; back-trail navigation

Tests: footer render; drawer state transitions; negative-evidence epistemic humility framing assertion.

- [ ] Steps 1–5: implement + tests + commit

---

### Task 13: Form-factor adaptations + accessibility audit

Per v1.0 Part 14.

- `src/lib/form_factor/useFormFactor.ts` — desktop/mobile detection hook via `window.matchMedia`
- `src/lib/form_factor/breakpoints.ts` — Tailwind breakpoint constants matching v1.0 Part 14.1–14.2 contracts
- Per-component responsive variants: mobile collapses dense panels (TrajectoriesPanel → single-column accordion; FailedInterventionHistory → top-3 with "more" expander)
- Mobile-specific limitations rendered explicitly per v1.0 Part 14.3 (Layer 5 deep investigation desktop-primary; trajectory chart degrades gracefully)
- `src/lib/a11y/liveRegion.ts` — screen-reader announcement helper for action results
- Audit pass: axe-core run + manual keyboard-only walkthrough per v1.0 Part 14.5
- Audit findings → `claudedocs/s2-fe-a11y-audit-2026-05-11.md`

- [ ] Steps 1–7: implement + audit + commit

---

### Task 14: Unit + component test coverage

Vitest unit tests across all components. Playwright component tests for the substrate-drawer + action-modal flows.

Coverage target: >80% line, >70% branch (modest given the visual surface). Verification-not-belief structural test parallels the backend Task 9 critical test:

```ts
test('every rendered Claim has substrateRefs', () => {
  const view = buildTestLayer1View();
  const { container } = render(<ResidentWorkspace view={view} />);
  const claims = container.querySelectorAll('[data-claim]');
  for (const claim of claims) {
    expect(claim.getAttribute('data-substrate-warn')).not.toBe('empty');
  }
});
```

- [ ] Steps 1–6: implement + commit

---

### Task 15: E2E test against running s2-aggregator

Playwright E2E. Skips when `S2_AGGREGATOR_URL` env var unset.

Scenarios mirror backend Task 10 E2E happy paths:
- Worklist entry → full workspace renders
- Override action → round-trip; recommendation re-fetches with override history populated
- Drill-through → SubstrateRefDrawer opens and renders the substrate observation

- [ ] Steps 1–5: implement + commit

---

## Pre-acceptance gate

Before declaring frontend Layer 1 complete:

1. ✅ `npm run build && npm run lint && npx tsc --noEmit` clean
2. ✅ Vitest unit tests pass with coverage targets met
3. ✅ Playwright component tests pass
4. ✅ Playwright E2E passes against staging s2-aggregator (operational, not just structural)
5. ✅ Verification-not-belief frontend assertion passes (every rendered Claim has substrateRefs)
6. ✅ axe-core accessibility audit: zero critical issues
7. ⏳ External clinical informatics UX review (operational gate — v1.0 Part 19 Week 14)
8. ⏳ 3-pharmacist pilot user testing × 1 week (operational gate)
9. ⏳ Design lead applies visual treatment to TODO(design lead) markers throughout

## Out of scope (still deferred)

- **Layer 2–5 visual content** — per Addendum Part 6, senior pharmacist authoring
- **S1 worklist** — kb-33 build (Step 5)
- **S3 GP Communication Hub frontend** — separate Tier 1 doc B (still to be authored)
- **Visual design system** — design lead's authoring work; Claude ships utility-class scaffolding + `TODO(design lead)` markers
- **Native mobile apps** — responsive web only
- **Internationalization** — English-only Phase 1
- **Real-time updates** — TanStack Query polling Phase 1; WebSocket / SSE deferred

Plan complete and saved.
