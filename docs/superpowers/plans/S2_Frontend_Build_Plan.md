# S2 Frontend Build Plan

**Version:** 1.0
**Date:** May 2026
**Status:** Engineering build plan for the S2 Resident Workspace frontend, calibrated to two senior React engineers (10-year average experience).

**Companion documents:**
- *S2 Resident Workspace Implementation Guidelines v1.0* — the canonical specification this plan executes against
- *S2 Adaptive Cognition Architectural Commitment Addendum* — substrate-primitive-inheritance discipline and Phase 1 scope
- *Architectural Commitment Addendum — Artefact-Per-Surface with Role-Views* — the recent reframe; S2 is pharmacist role-view into Workflow 1 + 2
- *Frictionless Citation Design Implementation Proposal + Supplemental* — citation discipline this plan inherits
- *CAPE v1.1 Implementation Guidelines* — CAPE context band integration point

**Backend reference:** S2 Layer 1 backend complete per status document (10 May). 17 HTTP routes operational, 226 tests passing, Phase 1c + Phase 2 merged, Phase 3 tightened ethics gates ready to merge. Frontend builds against committed backend contracts.

**Author's note:** This plan is calibrated to two senior React engineers. It assumes 10 years average experience means familiarity with modern React patterns (hooks, suspense, server components), TypeScript at intermediate-to-advanced level, accessibility expertise, and ability to make UI architecture decisions without close supervision. The plan specifies *what* needs to be built and the *constraints* that bind it; engineers will make implementation decisions within those constraints.

If the team composition changes (additional engineers, less experienced engineers, fewer engineering hours per week), the calendar estimates need recalibration. Effort estimates are given as engineer-weeks so they scale.

---

## Part 1 — What this plan covers and what it doesn't

### 1.1 In scope

- All 14 React components named in v1.0 Part 15 file structure
- Shared design system primitives upstream of those 14 components
- Mobile form-factor adaptation across all components (in MVP per team commitment)
- Backend wiring: integration with the 17 operational HTTP routes
- Form-factor adapter backend stubs (per v1.0 Part 14, currently unimplemented)
- Testing approach matching v1.0 Part 17 six-category structure
- Gate criteria for each phase
- Accessibility commitments (WCAG 2.1 AA minimum)
- Performance budget enforcement at the frontend layer

### 1.2 Out of scope

These are deferred per the Adaptive Cognition Addendum or per scope discipline:

- Layer 2-5 cognitive content (deferred to senior consultant pharmacist authoring + pilot evidence)
- v1.1 unified rewrite (deferred until Tier 1 complete or senior pharmacist input arrives)
- gRPC server implementation (proto IDL exists; codegen tooling decision pending; REST sufficient for pilot)
- ERM integration scaffold (`internal/erm_integration/ethical_review.go`) — backend Phase 2-completion work
- Redis view cache (`internal/store/redis/s2_view_cache.go`) — backend Phase 2-completion work
- Production adapters (shared permissions middleware, shared ethics log, kb-32 HTTP override forwarder) — backend Phase 2-completion work
- External clinical informatics UX review (operational, scheduled Week 14 gate)
- Pilot pharmacist user testing (3 pharmacists × 1 week, scheduled Week 14 gate)
- Future role-views (RN, GP, EN, PCW) — separate plans

### 1.3 The v1.0-scoped commitment

This plan builds components as **pharmacist role-view into Workflow 1 (Resident clinical work) and Workflow 2 (Recommendation lifecycle)**, not as generic workflow primitives.

Practical consequence: components are pharmacist-shaped. PendingRecommendationsPanel renders recommendations with pharmacist affordances. RestraintSignalsPanel renders restraint signals with pharmacist context.

When v1.1 unified rewrite arrives, shared primitives get extracted. The cost of that future extraction is bounded and acceptable per the Addendum's deferral rule.

Every component specification in Part 5 surfaces the boundary between *clinical data* (workflow-level, will be shared) and *role-specific presentation/affordances* (pharmacist-scoped, will not). The boundary discipline is what makes future extraction tractable.

### 1.4 The mobile-in-MVP commitment

Mobile form-factor is in MVP scope. This adds 30-50% to frontend engineering effort and forces decisions about touch interaction patterns equivalent to desktop hover patterns.

The plan specifies mobile-specific behaviour per component in Part 5 and consolidates mobile architectural decisions in Part 6.

**Decision worth revisiting after first pilot pharmacist conversation:** if pilot pharmacists confirm they work primarily at workstations during structured review sessions and use mobile only for casual reference between residents, mobile can defer to V1 and recover meaningful timeline. This plan ships with mobile in MVP per team commitment.

---

## Part 2 — Engineering preconditions

### 2.1 What the backend provides today

Per the status document (10 May 2026):

**Operational HTTP routes:** 17 routes for S2 data shapes — trajectories, pending recommendations, restraint signals, failed interventions, goals-of-care, care intensity, pharmacist actions, audit trail, complex workspace activation offer, pharmacist notes.

**Backend test coverage:** 226 tests across Phase 1a-1c and Phase 2.

**Deferred backend work blocking frontend integration:**

- `internal/store/redis/s2_view_cache.go` — frontend can proceed with backend reading from Postgres directly; cache addition later doesn't affect frontend code
- `internal/form_factor/desktop.go` + `mobile.go` — frontend needs a backend-resolved form-factor signal per request; current stubs return desktop unconditionally. **Frontend can proceed assuming desktop signal for MVP, but mobile form-factor support requires this backend work to land before mobile-specific behaviour is testable end-to-end.**
- `internal/erm_integration/ethical_review.go` — required by appropriateness gate; backend currently stub. Frontend can render against placeholder responses but production deployment requires this.

### 2.2 Backend wiring still needed (Claude-can-do, but backend work)

Per the status document's Section A:

1. gRPC server binding (codegen tooling decision pending) — frontend will use REST; gRPC is V1 concern
2. Form-factor adapter backend (`desktop.go` + `mobile.go`) — **blocking mobile MVP**
3. Redis view cache — not blocking
4. ERM integration scaffold — blocking production deployment, not blocking MVP frontend development
5. Production adapters (permissions middleware, ethics log, kb-32 override forwarder) — blocking production deployment
6. Empty-pending-recs SubstrateRef anchor — composition debt, frontend can render around it
7. isPsychotropicRuleID heuristic replacement — frontend reads the RuleID directly; backend cleanup

**Recommendation:** Backend team prioritises form-factor adapter (`desktop.go` + `mobile.go`) immediately so frontend mobile work isn't blocked at Phase 1 end. ERM integration scaffold and production adapters can land during frontend Phase 2-4 work.

### 2.3 What this plan assumes about backend stability

The plan assumes backend HTTP contracts are stable. If backend Phase 2-completion (the 8 tasks in `2026-05-09-phase-2-completion.md`) introduces breaking changes to response shapes, the frontend plan needs adjustment. Coordinate backend Phase 2-completion task 4 (override taxonomy vocabulary alignment) with frontend Phase 3 (when restraint and override surfaces ship) to avoid mid-development contract churn.

---

## Part 3 — Shared design system primitives

These are built first, before any of the 14 components. Building them once correctly is what makes the 14 components consistent and what makes mobile responsive design tractable.

### 3.1 The eight primitives

**Primitive 1 — Trajectory Chart Primitive (`<TrajectoryChart>`)**

Renders a clinical observation over time with baseline overlay, delta-from-baseline annotation, and trajectory direction indicator. Used by TrajectoriesPanel and TrajectoryChart components.

Constraints:
- Accepts substrate observation series with baseline, delta, trajectory metadata (from backend `/v1/s2/trajectories/:resident_id` route)
- Hover (desktop) / tap (mobile) reveals point-in-time details with citation surface
- Renders identical clinical data regardless of form-factor; visual density adapts
- Performance: <100ms render for typical 90-day daily observation series
- Accessibility: keyboard navigable; screen reader exposes "current value X, baseline Y, deviation Z%"

**Primitive 2 — Evidence Chain Renderer (`<EvidenceChain>`)**

Renders the substrate EvidenceTrace as role-appropriate clinical reasoning chain. Used by RecommendationCard, RestraintSignalCard, FailedInterventionHistory, and the Deep Audit layer of the citation surface.

Constraints:
- Pharmacist-appropriate depth per Frictionless Citation proposal (full clinical context with reasoning preserved across prior actors)
- Substrate-generated text per Frictionless Citation non-negotiable 1, not template strings
- Renders multi-actor reasoning preservation (per Walkthrough 2 DP-41) without overwriting toward consensus
- Hover (desktop) / tap (mobile) on cited evidence highlights specific data points in resident timeline
- Performance: <100ms expand to Layer 2 Reasoning, <500ms expand to Layer 3 Provenance

**Primitive 3 — Citation Surface (`<CitationSurface>`)**

Implements the four-layer disclosure architecture from Frictionless Citation: Signal → Reasoning → Provenance → Deep Audit. Used across all components that surface clinical recommendations or interpretations.

Constraints:
- Layer 1 Signal always visible; Layers 2-4 progressively disclosed
- Specific source citations (not generic "based on guidelines")
- Hover-to-highlight pattern (desktop) for time-linked provenance per Frictionless Citation
- Tap-to-pin-then-tap-to-unpin equivalent (mobile)
- Override-reason taxonomy capture (Frictionless Citation Supplemental Part 1)
- Citation versioning display (Frictionless Citation Supplemental Part 2): fire-time citation in audit views; current canonical citation in active clinical work; review banner for substantively superseded citations
- Performance budget enforced at primitive level

**Primitive 4 — Pharmacist Action Panel (`<PharmacistActionsPanel>`)**

The 11 pharmacist actions from v1.0 Part 12. Used by RecommendationCard, RestraintSignalCard, and standalone panel.

Constraints:
- Actions surface conditionally based on current state (e.g., "refine" only available on drafted recommendations)
- Action confirmation flow per action with override-reason capture where appropriate
- Form-factor: full action set visible on desktop; primary actions visible on mobile with secondary actions in overflow menu
- Authorisation check per action against backend (the expanded Authorisation evaluator)
- Audit trail capture for every action

**Primitive 5 — Urgency Tier Indicator (`<UrgencyTier>`)**

Renders Tier 1/2/3/4 priority taxonomy from Layer 3 v2. Used by CAPEContextBand, PendingRecommendationsPanel, RestraintSignalsPanel.

Constraints:
- Visual encoding distinct from colour alone (per accessibility — colour-blindness considered)
- Tooltip on hover (desktop) / tap-to-expand (mobile) shows tier rationale per Frictionless Citation
- CAPE prioritisation logic respected (CAPE v1.1 commitment); tier alone does not determine display order, CAPE composite score does

**Primitive 6 — Audit Trail Footer (`<AuditTrailFooter>`)**

Per-component audit trail showing what other actors have done. Used as embedded footer in RecommendationCard, RestraintSignalCard, and per-zone footers in the workspace.

Constraints:
- Role-appropriate visibility per the Architectural Commitment Addendum visibility classification (the pharmacist sees their own actions in detail; sees other actors' actions at audit-trail depth)
- Click-through to Deep Audit (Layer 4 of citation surface)
- Chronological display with multi-actor preservation
- Performance: <100ms render of last 5 trail entries; lazy-load older entries

**Primitive 7 — Substrate Observation Drill-through (`<ObservationDrillThrough>`)**

The "click trajectory → see source observations" pattern from v1.0 Part 10. Used by TrajectoriesPanel.

Constraints:
- Modal overlay (desktop) / bottom-sheet (mobile)
- Renders observation source, timestamp, baseline reference, captured-by actor
- Negative-evidence display where applicable (per Frictionless Citation Supplemental Part 3)
- Closable with keyboard (Esc) and screen-reader announcement

**Primitive 8 — Form-Factor Adapter (`<FormFactor>`)**

Top-level wrapper resolving desktop vs mobile presentation. Consumes backend form-factor signal once available; falls back to user-agent + viewport-width heuristic until backend signal lands.

Constraints:
- Resolves once per session, not per render
- Overridable via user preference (some pharmacists prefer desktop layout on tablet)
- Coordinates with backend form-factor adapter once that lands

### 3.2 Why these are built first

These eight primitives have at least three consumers each among the 14 components. Building them first prevents:

- Inconsistent rendering of clinical data across components
- Mobile form-factor decisions made independently per component
- Citation discipline drift across components
- Audit trail rendering inconsistency
- Performance budget violations from component-specific implementations

Building them first costs 4-6 engineer-weeks (2 engineers × 2-3 weeks) before any of the 14 components ship. **This is the right investment.** Without it, mobile form-factor + citation discipline + audit trail consistency will be retrofitted across all 14 components later at much higher cost.

---

## Part 4 — Component priority order

### 4.1 The four phases

```
PHASE 0 — Shared primitives (Weeks 1-2-3)
   ↓
PHASE 1 — Skeleton (Weeks 3-5)
   ↓
PHASE 2 — Core clinical functionality (Weeks 5-9)
   ↓
PHASE 3 — Clinical context and safety (Weeks 9-12)
   ↓
PHASE 4 — Continuity infrastructure (Weeks 12-14)
```

Phases overlap by 1 week to allow handoff and integration testing.

### 4.2 The dependency graph

Components depend on primitives and on each other. The build order respects dependencies:

```
Phase 0 — Primitives (TrajectoryChart, EvidenceChain, CitationSurface, 
          PharmacistActionsPanel, UrgencyTier, AuditTrailFooter,
          ObservationDrillThrough, FormFactor)
   ↓
Phase 1 — ResidentWorkspace (parent) + CAPEContextBand (header)
   ↓
Phase 2 — TrajectoriesPanel + TrajectoryChart + PendingRecommendationsPanel +
          RecommendationCard + PharmacistActionsPanel (consumed)
   ↓
Phase 3 — RestraintSignalsPanel + RestraintSignalCard + FailedInterventionHistory +
          GoalsOfCarePanel + CareIntensityPanel + ComplexActivationOffer
   ↓
Phase 4 — AuditTrailFooter (consumed) + PharmacistNotes
```

### 4.3 Parallel work for two engineers

With two senior engineers, Phase 0 primitives split as follows:
- Engineer A: TrajectoryChart, EvidenceChain, ObservationDrillThrough, FormFactor
- Engineer B: CitationSurface, PharmacistActionsPanel, UrgencyTier, AuditTrailFooter

Phase 1 has Engineer A on ResidentWorkspace shell and Engineer B on CAPEContextBand. Both engineers needed for integration.

Phase 2 splits as:
- Engineer A: TrajectoriesPanel + TrajectoryChart
- Engineer B: PendingRecommendationsPanel + RecommendationCard

Phase 3 splits as:
- Engineer A: RestraintSignalsPanel + RestraintSignalCard + FailedInterventionHistory
- Engineer B: GoalsOfCarePanel + CareIntensityPanel + ComplexActivationOffer

Phase 4 with both engineers on integration: AuditTrailFooter wiring across all components + PharmacistNotes.

### 4.4 Why this order

**Why CAPEContextBand in Phase 1 not Phase 2:** The CAPE context band is the entry point. Pharmacists see it before they see clinical content. Building it in Phase 1 with the workspace shell lets pilot users see the platform shape early, even if clinical functionality lands in Phase 2.

**Why TrajectoriesPanel before PendingRecommendationsPanel:** Trajectories establish clinical context; recommendations require context to be interpretable. Building trajectories first gives engineers the "harder" UX problem (time-series rendering with hover-to-highlight, mobile-equivalent interactions) before the relatively simpler card-based recommendation rendering.

**Why RestraintSignals in Phase 3 not Phase 2:** Restraint signals are advisory-only in Phase 1 per v1.0 Part 7. They're important but they don't gate pharmacist action. Recommendations and trajectories are the primary workflow. Restraint signals enrich, they don't drive.

**Why Goals-of-care + Care intensity in Phase 3:** These are contextual rendering primitives. They affect how recommendations are interpreted but don't generate new pharmacist actions. Building them after recommendations (so the action panel is stable) is cleaner than building them concurrently.

**Why AuditTrailFooter wiring in Phase 4:** The footer needs to render across all 13 other components. Building it in Phase 4 after the other components are stable means the footer's integration touches stable code, not in-flux code. This trades slight delay for substantially lower integration risk.

### 4.5 What "ship" means per phase

Each phase ships when:
- All components in the phase pass v1.0 Part 17 testing categories (functional, integration, accessibility, performance, snapshot, visual regression)
- Performance budgets enforced (per primitive and per component)
- Mobile form-factor verified on representative devices (per Part 6)
- Backend integration verified against the 17 HTTP routes
- Internal engineering review passed
- Phase gate criteria met (Part 9)

"Ship" does not mean "production deployment." Production deployment is gated on:
- Backend Phase 2-completion (Postgres-backed adapters, ERM scaffold, production middleware)
- External clinical informatics UX review (Week 14)
- Pilot pharmacist user testing (Week 14)

The frontend is build-complete after Phase 4 (Week 14). Production-deployable some weeks after that, gated on backend Phase 2-completion + external review.

---

## Part 5 — Per-component specifications

Each component specification has the same structure: parent zone in v1.0 Part 4 layout, backend route consumed, state requirements, props interface (signature only), mobile form-factor handling, accessibility commitment, testing approach, gate criteria.

### 5.1 ResidentWorkspace.tsx

**Purpose:** The parent component. Renders the 11-zone S2 layout. Coordinates child components.

**Parent zone:** None (top-level).

**Backend routes consumed:** `/v1/s2/residents/:resident_id/snapshot` (composite endpoint returning workspace metadata).

**State requirements:**
- Current resident_id (route parameter)
- Workspace mode (standard vs complex-activated, default standard)
- Form-factor (resolved from FormFactor primitive)
- Loading and error states per child zone (delegated to children)

**Props interface:** `{ residentId: string }`. Children read state from React Context.

**Mobile form-factor handling:** Layout switches from 11-zone grid (desktop) to scrollable single-column with collapsible zones (mobile). CAPEContextBand and ResidentHeader pinned at top; other zones lazy-loaded as user scrolls.

**Accessibility:** Skip-to-zone landmarks. Keyboard navigation between zones via tab order matching visual order. Screen reader announces zone changes.

**Testing:**
- Functional: routes to correct resident, mode switching
- Integration: zone composition, child state propagation
- Accessibility: landmark navigation, focus management
- Performance: <500ms initial render, <100ms zone-to-zone navigation

**Gate criteria for shipping:** All 11 zones placeholder-rendered, child zone composition verified, mobile single-column flow verified on 360px-wide viewport.

### 5.2 CAPEContextBand.tsx

**Purpose:** The header band showing CAPE prioritisation context — which residents are queued, why this resident now, what's the alternative.

**Parent zone:** Zone 1 (header band, full-width).

**Backend routes consumed:** `/v1/cape/context-band/:pharmacist_id` (CAPE substrate per current pharmacist).

**State requirements:**
- Current pharmacist context (session-scoped)
- CAPE queue with composite scores
- Why-this-resident-now rationale

**Props interface:** None directly; reads from ResidentWorkspace context.

**Mobile form-factor handling:** Collapses to single-line summary on mobile with expand-on-tap for full context. CAPE rationale text shortened to <80 characters on mobile; full text in expand.

**Accessibility:** Screen reader announces queue position and rationale on focus.

**Testing:**
- Functional: rationale rendering, queue position
- Integration: with CAPE backend, real composite scores
- Performance: <100ms render

**Gate criteria:** CAPE rationale renders from real backend response; queue position accurate; mobile expand-collapse works.

**Known gap:** Full CAPE signal display vocabulary is currently 6-entry stub per status document. Phase 1 ships against this stub; full vocabulary lands when senior pharmacist authoring completes.

### 5.3 TrajectoriesPanel.tsx + TrajectoryChart.tsx

**Purpose:** TrajectoriesPanel hosts multiple TrajectoryChart instances for this resident's clinically meaningful observations (BP, weight, eGFR, behavioural scores, etc.). TrajectoryChart renders one time-series with baseline and delta.

**Parent zone:** Zone 4 (clinical trajectories, ~50% workspace area on desktop).

**Backend route consumed:** `/v1/s2/trajectories/:resident_id` (returns array of trajectory series with baselines and deltas).

**State requirements:**
- Trajectory series data (lazy-loaded per series)
- Selected time-window (default 90 days, expandable to 1 year)
- Drill-through state (which observation point is being inspected)

**Props interface:** TrajectoriesPanel `{}`. TrajectoryChart `{ series: TrajectorySeries, onObservationClick: (id) => void }`.

**Mobile form-factor handling:** Trajectories stack vertically on mobile (one per row); each chart is full-width. Drill-through opens as bottom-sheet on mobile, modal on desktop. Hover-to-highlight becomes tap-to-pin on mobile.

**Accessibility:** Each TrajectoryChart screen-reader-exposes "[observation type], current value, baseline, deviation, trajectory direction". Keyboard navigation between data points (left/right arrows).

**Testing:**
- Functional: trajectory rendering, baseline overlay, delta annotation
- Integration: backend trajectory data, observation drill-through
- Accessibility: keyboard navigation, screen reader announcement
- Performance: <500ms render of 5-trajectory panel; <100ms hover-to-highlight

**Gate criteria:** Trajectories render from real backend data, mobile single-column verified, drill-through to observation source works, baseline confidence indicators visible.

### 5.4 PendingRecommendationsPanel.tsx + RecommendationCard.tsx

**Purpose:** Renders all pending recommendations for this resident. Each is a RecommendationCard.

**Parent zone:** Zone 5 (pending recommendations, ~30% workspace area on desktop).

**Backend route consumed:** `/v1/s2/recommendations/:resident_id/pending`.

**State requirements:**
- Pending recommendations list
- Per-card expansion state (Layer 1/2/3/4 disclosure)
- Per-card action state (drafted, refining, withholding, submitting)

**Props interface:** PendingRecommendationsPanel `{}`. RecommendationCard `{ recommendation: Recommendation, onAction: (action, reason?) => void }`.

**Mobile form-factor handling:** Cards stack full-width on mobile. Layer 1 Signal always visible; Layers 2-4 expand on tap (vs hover on desktop). Pharmacist actions in primary row (3 actions visible) with overflow menu for remaining 8.

**Accessibility:** Each card is a landmark. Action buttons keyboard-focusable with descriptive labels. Override reason capture screen-reader-accessible.

**Testing:**
- Functional: recommendation rendering, layer disclosure, pharmacist actions
- Integration: backend recommendation data, citation surface, action confirmation, override-reason capture
- Accessibility: landmark navigation, action button labels
- Performance: <100ms Layer 1 render; <500ms Layer 3 expand

**Gate criteria:** Recommendations render from real backend data, all 11 pharmacist actions functional, override-reason taxonomy capture works, citation surface (4-layer) functional.

**Known gap:** Empty-pending-recs SubstrateRef anchor (per status document Section A.7) — frontend handles gracefully with empty-state UI.

### 5.5 RestraintSignalsPanel.tsx + RestraintSignalCard.tsx

**Purpose:** Renders restraint signals for this resident. Phase 1 advisory-only per v1.0 Part 7.

**Parent zone:** Zone 6 (restraint signals, ~20% workspace area on desktop, may collapse to summary if no signals).

**Backend route consumed:** `/v1/s2/restraint-signals/:resident_id`.

**State requirements:**
- Restraint signals list (typically 0-3 active)
- Per-signal expansion state

**Props interface:** RestraintSignalsPanel `{}`. RestraintSignalCard `{ signal: RestraintSignal, onAcknowledge: () => void }`.

**Mobile form-factor handling:** Collapses to single-row summary on mobile with expand-to-detail-on-tap. Each signal expanded view full-width.

**Accessibility:** Signals are advisory; screen reader announces "advisory: [signal description]". Acknowledgment captured with audit trail.

**Testing:**
- Functional: signal rendering, acknowledgment capture
- Integration: backend restraint signal data, 9 restraint signal detectors (per Phase 2b Task 9)
- Performance: <100ms render

**Gate criteria:** All 9 restraint signal types render correctly; acknowledgment captured in audit trail; advisory-only nature clearly communicated.

### 5.6 FailedInterventionHistory.tsx

**Purpose:** Renders history of failed prior interventions for this resident — what was tried, why it failed, what was learned.

**Parent zone:** Zone 7 (failed intervention history, expandable side panel on desktop, separate tab on mobile).

**Backend route consumed:** `/v1/s2/failed-interventions/:resident_id`.

**State requirements:**
- Failed intervention list (sorted by recency)
- Per-intervention detail expansion state

**Props interface:** `{}`.

**Mobile form-factor handling:** Tab in mobile workspace navigation (not always-visible side panel).

**Accessibility:** List items keyboard-navigable. Each intervention's failure reason and learning rendered as structured text.

**Testing:**
- Functional: history rendering, detail expansion
- Integration: backend failed-intervention data, FIR retrieval gap documented in status

**Gate criteria:** Failed interventions render from real backend data; FIR retrieval gap (kb-32 RecommendationID→ResidentID JOIN-resolver per status Section C.19) does not block rendering — frontend handles partial data gracefully.

**Known gap:** kb-32 JOIN-resolver pending. Frontend renders what's available; full pattern detection deferred per status document.

### 5.7 GoalsOfCarePanel.tsx

**Purpose:** Renders resident's goals-of-care documentation. Affects interpretation of recommendations.

**Parent zone:** Zone 8 (goals of care, header above clinical content on desktop, separate section on mobile).

**Backend route consumed:** `/v1/s2/goals-of-care/:resident_id`.

**State requirements:**
- Current goals-of-care documentation
- Last review date and reviewer
- Documented preferences (specific to medication contexts)

**Props interface:** `{}`.

**Mobile form-factor handling:** Collapses to single-line summary with expand-on-tap.

**Accessibility:** Goals rendered as readable prose. Last-review date prominent for clinical context.

**Testing:**
- Functional: goals rendering, review date accuracy
- Integration: backend goals data

**Gate criteria:** Goals render correctly; last-review date accurate; mobile expand functional.

### 5.8 CareIntensityPanel.tsx

**Purpose:** Renders current care intensity tag (active treatment / rehabilitation / comfort focused / palliative) with transition context.

**Parent zone:** Zone 9 (care intensity, paired with goals-of-care).

**Backend route consumed:** `/v1/s2/care-intensity/:resident_id`.

**State requirements:**
- Current care intensity tag
- Effective date, documented by
- Recent transitions (if any)

**Props interface:** `{}`.

**Mobile form-factor handling:** Single-row indicator with tap-to-detail for transition history.

**Accessibility:** Tag screen-reader-exposed. Transition history accessible via keyboard.

**Testing:**
- Functional: tag rendering, transition history
- Integration: backend care-intensity data
- Performance: <100ms render

**Gate criteria:** Tag accurate; transition history visible; affects rule firing per substrate (verified in integration test).

### 5.9 ComplexActivationOffer.tsx

**Purpose:** Offers activation of complex workspace mode for residents whose clinical complexity warrants it.

**Parent zone:** Zone 10 (complex activation, dismissible banner above clinical content).

**Backend route consumed:** `/v1/s2/complex-activation/:resident_id` (returns offer with rationale or null if not warranted).

**State requirements:**
- Offer status (offered / accepted / dismissed)
- Dismissal persistence per session

**Props interface:** `{ onActivate: () => void }`.

**Mobile form-factor handling:** Top banner; dismissible.

**Accessibility:** Offer announced on render; dismissal button labeled.

**Testing:**
- Functional: offer rendering, activate path, dismiss path
- Integration: backend offer logic

**Gate criteria:** Offer renders when warranted; dismiss persists per session; activate handed to ResidentWorkspace for mode switch.

**Important note:** The activated Layer 3 view itself is **deferred** per Adaptive Cognition Addendum Part 6.1 (senior consultant pharmacist authoring required). This component only handles the offer; activating switches workspace mode but the activated content is placeholder until Tier 2.

### 5.10 PharmacistActionsPanel.tsx

**Purpose:** Standalone action panel for actions not tied to a specific recommendation (e.g., "log clinical observation", "add note", "submit complex review").

**Parent zone:** Zone 11 (pharmacist actions, footer area on desktop, separate tab on mobile).

**Backend routes consumed:** `/v1/s2/actions/log-observation`, `/v1/s2/actions/add-note`, etc. (per v1.0 Part 12's 11 actions).

**State requirements:**
- Available actions (some conditional on current context)
- Per-action form state (during capture)

**Props interface:** `{}`.

**Mobile form-factor handling:** Tab in mobile workspace navigation. Action selection then form rendering.

**Accessibility:** Each action labeled clearly. Forms accessible with proper field labels.

**Testing:**
- Functional: all 11 actions functional
- Integration: backend action capture, audit trail
- Accessibility: form labeling, keyboard navigation
- Override-reason capture per Frictionless Citation Supplemental Part 1

**Gate criteria:** All 11 actions functional; audit trail capture verified; override-reason taxonomy applied.

### 5.11 AuditTrailFooter.tsx

**Purpose:** Footer rendering audit trail per component. Cross-cutting across the workspace.

**Parent zone:** Embedded in each component as appropriate (RecommendationCard footer, RestraintSignalCard footer, workspace global footer).

**Backend routes consumed:** `/v1/s2/audit-trail/:context_id` (parameterized by what context the audit is for).

**State requirements:**
- Recent audit entries (default last 5)
- Expandable to full history

**Props interface:** `{ contextId: string, scope: 'recommendation' | 'observation' | 'workspace' }`.

**Mobile form-factor handling:** Collapsed footer summary; tap-to-expand for full history.

**Accessibility:** Audit entries as structured list; keyboard navigable.

**Testing:**
- Functional: trail rendering, expansion
- Integration: backend audit data, multi-actor preservation
- Performance: <100ms render of last 5 entries

**Gate criteria:** Audit trail renders multi-actor history; visibility classification respected (pharmacist sees their own actions in detail; sees others at audit-appropriate depth); click-through to Deep Audit functional.

### 5.12 PharmacistNotes.tsx

**Purpose:** Pharmacist's personal annotation on this resident — clinical reasoning notes, observations, follow-up reminders. Not shared with other actors by default.

**Parent zone:** Zone 12 (pharmacist notes — actually a separate panel, possibly v1.0 Part 12 not making this clear as zone vs action).

**Backend routes consumed:** `/v1/s2/pharmacist-notes/:resident_id` (GET, POST, PATCH).

**State requirements:**
- Notes list for this pharmacist on this resident
- Editor state during note authoring
- Auto-save state

**Props interface:** `{}`.

**Mobile form-factor handling:** Tab in mobile workspace navigation. Editor takes full screen on mobile.

**Accessibility:** Editor accessible. Notes list keyboard-navigable.

**Testing:**
- Functional: note authoring, editing, deletion
- Integration: backend note persistence, auto-save
- Trust architecture: notes are pharmacist-private by default (Frictionless Citation principle 5 / v3.0 non-negotiable 2)

**Gate criteria:** Notes private to pharmacist by default; auto-save functional; multi-session persistence verified.

---

## Part 6 — Form-factor handling

### 6.1 The architectural decisions

Mobile in MVP forces decisions across the frontend. The principles:

**Principle 1 — Mobile is not "responsive desktop."** Mobile is a separate interaction context with its own patterns. Desktop hover patterns become tap-to-pin patterns; desktop overlay patterns become bottom-sheet patterns; desktop side panels become bottom-tab navigation.

**Principle 2 — Touch interaction patterns are explicit.** Every component specifies its mobile behaviour, not just "responsive layout." Hover-to-highlight on TrajectoryChart becomes tap-to-pin with second-tap-to-unpin. Modal overlay becomes bottom-sheet. Tooltip becomes inline expansion.

**Principle 3 — Performance budget on mobile devices.** The Layer 1 Signal <100ms render, hover-to-highlight <100ms, Provenance expand <500ms targets hold on mid-range Android devices (Snapdragon 700-series equivalent), not just desktop browsers. This affects bundle size, lazy-loading, and client-side computation choices.

**Principle 4 — Touch target size.** All interactive elements meet WCAG 2.1 AA touch target size (44×44 minimum). Action buttons, observation drill-through, citation surface taps — all sized for finger interaction.

### 6.2 Specific touch interaction patterns

**Desktop hover-to-highlight → Mobile tap-to-pin:**

- Desktop: hovering over a citation in Layer 3 Provenance highlights data points in the resident timeline; releasing hover removes highlight
- Mobile: tapping the citation pins it; subsequent timeline points are highlighted; tapping again or tapping elsewhere unpins

**Desktop modal overlay → Mobile bottom-sheet:**

- Desktop: Deep Audit opens as overlay modal centered on screen
- Mobile: Deep Audit opens as bottom-sheet covering bottom 80% of screen; pull-down-to-dismiss

**Desktop side panel → Mobile bottom-tab navigation:**

- Desktop: FailedInterventionHistory in right side panel always visible
- Mobile: FailedInterventionHistory as bottom tab in workspace navigation, tab-to-show

**Desktop tooltip → Mobile inline expansion:**

- Desktop: hovering over UrgencyTier shows tier rationale in tooltip
- Mobile: tapping UrgencyTier expands inline with rationale; tap again collapses

### 6.3 Backend form-factor signal

The frontend resolves form-factor through three sources, in priority order:

1. **Backend form-factor signal** (from `internal/form_factor/desktop.go` + `mobile.go` per v1.0 Part 14 — currently stub). When available, this is authoritative.

2. **User preference** (some pharmacists prefer desktop layout on tablet). Stored per-pharmacist session.

3. **User-agent + viewport-width heuristic** (fallback until backend signal available). Mobile if viewport ≤768px AND touch-capable, otherwise desktop.

**Engineering note:** Phase 1 frontend uses the heuristic with backend signal integration planned for Phase 2 once backend `desktop.go` + `mobile.go` lands. Don't gate frontend work on backend form-factor adapter completion.

### 6.4 Devices to test against

The plan specifies minimum testing surface:

- **Desktop:** Chrome (Linux, macOS, Windows), Firefox, Safari, Edge — last two major versions of each
- **Tablet:** iPad Pro (1024×1366), iPad Mini (768×1024), Galaxy Tab S (800×1280)
- **Phone:** iPhone 13/14/15 (390×844), iPhone SE (375×667), Galaxy S22/S23 (360×800)

Pilot site engagement may surface specific devices pharmacists use — calibrate testing matrix accordingly when pilot sites confirmed.

### 6.5 Performance budget verification per device class

Performance budget enforced per device class:

- **Desktop:** Layer 1 <100ms, hover <100ms, Layer 2 <300ms, Layer 3 <500ms, Layer 4 <1000ms
- **Tablet:** Same as desktop (Apple Silicon iPads exceed; Galaxy Tab S meets)
- **Mid-range mobile:** Layer 1 <150ms, tap-equivalent <150ms, Layer 2 <500ms, Layer 3 <800ms, Layer 4 <1500ms (50% allowance over desktop on mid-range phones)

If mid-range mobile performance exceeds these budgets, the affected component is re-engineered before Phase ships.

---

## Part 7 — Backend wiring still needed

This part lists what the frontend needs from backend that the status document identifies as not-yet-built.

### 7.1 Blocking mobile MVP

**Form-factor adapter backend** (`internal/form_factor/desktop.go` + `mobile.go`)

Required for end-to-end mobile testing in Phase 1+. Frontend can proceed with viewport-width heuristic for Phase 0-1, but Phase 2+ component shipping requires backend signal accuracy.

**Recommended backend Phase 2-completion scheduling:** Land form-factor adapter in backend Phase 2-completion Task 1 or 2, so it's available before frontend Phase 2 ships.

### 7.2 Blocking production deployment (not blocking MVP frontend development)

**ERM integration scaffold** (`internal/erm_integration/ethical_review.go`)

Required for production appropriateness gate (per Phase 2a Task 4). Frontend can render against backend stub responses for MVP development. Production deployment requires this.

**Production adapters:**
- Shared permissions middleware
- Shared ethics log logger
- kb-32 HTTP override forwarder

Required for production. Frontend tested against mock responses during MVP development. Production deployment requires these in backend Phase 2-completion.

**Redis view cache** (`internal/store/redis/s2_view_cache.go`)

Performance optimization. Frontend works without it; performance budget meets desktop targets even without caching. Cache addition does not require frontend changes.

### 7.3 Technical debts in backend

**Empty-pending-recs SubstrateRef anchor** (per status Section A.7)

Frontend handles gracefully with empty-state UI. Backend cleanup not blocking frontend.

**isPsychotropicRuleID heuristic replacement** (per status Section A.8)

Frontend reads RuleID directly from backend response; backend heuristic replacement does not affect frontend.

**kb-32 RecommendationID→ResidentID JOIN-resolver** (per status Section C.19)

Affects FailedInterventionHistory completeness. Frontend renders what's available; full pattern detection deferred per status document.

### 7.4 Coordination with backend Phase 2-completion

The frontend plan assumes the backend team works on Phase 2-completion (the 8 tasks in `2026-05-09-phase-2-completion.md`) in parallel with frontend Phase 0-4.

**Critical coordination:** Phase 2-completion Task 4 (override taxonomy vocabulary alignment) should land before frontend Phase 3 (when restraint signals and pharmacist action panel ship), or both teams should agree on vocabulary in advance to avoid mid-development contract churn.

**Recommended sequencing:**
- Backend Phase 2-completion Task 4 (override taxonomy) by frontend Week 8
- Backend form-factor adapter (`desktop.go` + `mobile.go`) by frontend Week 4-5

If backend can't meet these dates, frontend phases re-plan accordingly.

---

## Part 8 — Testing approach

Matching v1.0 Part 17's six categories.

### 8.1 Functional testing

Per-component unit tests for rendering, state transitions, action handling. React Testing Library. Coverage target ≥90% statement coverage per component.

### 8.2 Integration testing

Per-phase integration tests. Frontend talks to backend through real HTTP (or recorded responses in CI). End-to-end flows tested.

### 8.3 Accessibility testing

Per-component accessibility verification. axe-core in test suite. Manual keyboard navigation testing. Screen reader testing (VoiceOver, NVDA) on representative components.

### 8.4 Performance testing

Performance budget enforced per component. Lighthouse CI for desktop. WebPageTest mid-range mobile profile. Performance regression blocks merge.

### 8.5 Snapshot and visual regression testing

Per-component snapshot tests. Visual regression on representative components via tooling decision (Percy / Chromatic / Loki — engineer choice).

### 8.6 Verification-not-belief testing (v1.0 Part 17.2)

Tests that verify substrate accuracy rather than UI compliance. E.g., trajectory chart renders the actual baseline value from substrate, not a templated/mocked baseline. Citation surface cites the actual citation_at_fire_time from substrate, not a current value.

These tests are slow (require real substrate fixtures) but they're what verifies the platform's "substrate-driven, not template-driven" commitment from Frictionless Citation non-negotiable 1.

### 8.7 Test scoping by phase

Each phase ships when:
- Functional + integration + accessibility tests pass at coverage target
- Performance tests within budget for that phase's components
- Snapshot tests committed
- Visual regression baseline captured
- Verification-not-belief tests pass for substrate-consuming components

Phase 0 (primitives) has higher testing density per primitive because primitives' correctness affects all 14 components.

---

## Part 9 — Gate criteria per phase

### 9.1 Phase 0 gate (primitives)

Ship when:
- 8 primitives complete with full test coverage
- TrajectoryChart renders baseline + delta correctly on representative data
- CitationSurface implements 4-layer disclosure with substrate-generated text
- EvidenceChain renders multi-actor reasoning preservation
- Hover-to-highlight functional on desktop primitives
- Tap-to-pin functional on mobile primitive equivalents
- Performance budget verified for primitives on all device classes
- Internal engineering review of primitive design

Failure to meet gate: Phase 1 doesn't start. Re-plan.

### 9.2 Phase 1 gate (skeleton)

Ship when:
- ResidentWorkspace renders 11-zone layout
- CAPEContextBand renders CAPE substrate
- Mobile single-column flow verified
- Navigation between zones works (desktop) / tab navigation works (mobile)
- Performance budget met for skeleton
- Pilot site can navigate the workspace shell even though clinical content is placeholder

Failure to meet gate: Phase 2 doesn't start. Re-plan.

### 9.3 Phase 2 gate (core clinical functionality)

Ship when:
- TrajectoriesPanel + TrajectoryChart render real backend data
- PendingRecommendationsPanel + RecommendationCard render real backend data
- All 11 pharmacist actions functional from RecommendationCard
- Override-reason taxonomy capture works
- Citation surface 4-layer disclosure functional on RecommendationCard
- Hover-to-highlight on TrajectoryChart with citation pinning on mobile
- Audit trail capture verified for all actions
- Performance budget met

Failure to meet gate: Phase 3 doesn't start. Re-plan.

### 9.4 Phase 3 gate (clinical context and safety)

Ship when:
- RestraintSignalsPanel + RestraintSignalCard render 9 restraint signal types correctly
- FailedInterventionHistory renders with graceful handling of FIR retrieval gap
- GoalsOfCarePanel + CareIntensityPanel render correctly
- ComplexActivationOffer renders and dismissal persists
- Mobile form-factor verified for all Phase 3 components

Failure to meet gate: Phase 4 doesn't start. Re-plan.

### 9.5 Phase 4 gate (continuity infrastructure)

Ship when:
- AuditTrailFooter integrated into all components needing it
- PharmacistNotes functional with auto-save and multi-session persistence
- All 14 components production-ready (modulo backend Phase 2-completion)
- Performance budget met across all components
- Full S2 workspace integration test passes
- Internal engineering review complete

Failure to meet gate: Pre-pilot review delayed. Re-plan.

### 9.6 Final gate (production deployment)

Ship when (independent of frontend phases):
- Backend Phase 2-completion complete (Postgres-backed adapters, ERM scaffold, production middleware)
- External clinical informatics UX review passed (v1.0 Part 19 Week 14 gate)
- Pilot pharmacist user testing complete (3 pharmacists × 1 week, v1.0 Part 19 Week 14)
- Pilot site engagement confirmed
- Production deployment runbook complete

Failure to meet gate: Production deployment delayed. Pilot delayed.

---

## Part 10 — Engineering effort estimates

For two senior engineers (10-year average React/TypeScript experience):

| Phase | Engineer-weeks total | Calendar weeks (2 engineers) | What |
|---|---|---|---|
| **Phase 0** | 4-6 weeks | 2-3 weeks | 8 shared primitives |
| **Phase 1** | 2-3 weeks | 1.5-2 weeks | Skeleton (ResidentWorkspace + CAPEContextBand) |
| **Phase 2** | 4-6 weeks | 2-3 weeks | TrajectoriesPanel + TrajectoryChart + PendingRecommendationsPanel + RecommendationCard |
| **Phase 3** | 4-6 weeks | 2-3 weeks | Restraint + Failed Interventions + Goals/Care + ComplexActivation |
| **Phase 4** | 2-3 weeks | 1-1.5 weeks | AuditTrailFooter + PharmacistNotes + final integration |
| **TOTAL (build)** | **16-24 weeks** | **8-12 weeks calendar** | **Frontend build complete** |

**Add for mobile in MVP:** 30-50% additional effort. Effort range becomes 21-36 engineer-weeks, calendar range 10-14 weeks.

**Add for testing depth:** Tests included in per-phase estimates. Visual regression tooling setup (~3-5 engineer-days, one-time, in Phase 0).

**Add for backend Phase 2-completion coordination:** No additional frontend time if backend team meets coordination dates in Part 7.4. Re-plan needed if backend slips.

**Contingency:** Add 15-25% to all estimates for inevitable surprises (component complexity higher than anticipated, accessibility testing surfacing issues, mobile-specific bugs).

**Conservative final estimate:** 25-44 engineer-weeks total, 12-16 calendar weeks for two engineers.

**Aggressive final estimate (everything goes well):** 21 engineer-weeks total, 10 calendar weeks.

**My recommendation for what to commit externally:** 12-14 calendar weeks to MVP frontend, gated on backend Phase 2-completion coordination and pilot site confirmation. If you commit to anything shorter, you're optimistic. If you commit to anything longer, you're not pushing the team.

---

## Part 11 — Risks and what to do about them

### 11.1 Frontend-specific risks

**Risk F1 — Mobile form-factor complexity exceeds estimate.**

Mobile hover-equivalent patterns (tap-to-pin, bottom-sheets, mobile-specific citation surface interactions) are easier to specify than to implement well. The first time tap-to-pin breaks on iOS Safari with VoiceOver, expect a 2-3 day investigation.

**Mitigation:** Engineering team builds tap-to-pin pattern in Phase 0 (TrajectoryChart) and stress-tests with VoiceOver before Phase 2 ships. Other mobile equivalents pattern-match to this verified implementation.

**Risk F2 — Citation surface 4-layer disclosure performance degrades on mobile.**

The Layer 1-2-3-4 progressive disclosure with mobile-specific bottom-sheet patterns has more moving parts than desktop. Performance budget on mid-range mobile becomes contested.

**Mitigation:** Performance testing on representative mid-range device starts Phase 0. If Layer 3 expand exceeds 800ms on mid-range mobile, lazy-load Layer 3 evidence inline rather than full bottom-sheet.

**Risk F3 — Backend Phase 2-completion slips, blocking production deployment.**

The frontend is build-complete at end of Phase 4. Production deployment requires backend Phase 2-completion (Postgres adapters, ERM scaffold, production middleware). If backend Phase 2-completion slips beyond Week 16, production deployment slips equivalently.

**Mitigation:** Backend Phase 2-completion is independently tracked. Frontend can ship production-ready against current backend state; backend slips delay production cut, not frontend build.

**Risk F4 — Accessibility surprises in Phase 4.**

Some accessibility issues only surface during integration testing or pilot pharmacist testing. Phase 4 may need re-work on Phase 0-3 components for accessibility issues that don't show in unit tests.

**Mitigation:** Accessibility testing in each phase, not just at end. Screen reader testing on representative components weekly during Phase 0-4. Pilot pharmacist with accessibility needs (if any) tested earlier rather than last.

### 11.2 Cross-cutting risks (from earlier conversation)

**Risk C1 — Visibility classification engineering scope.**

The Architectural Commitment Addendum committed to seven visibility classes. Frontend renders the role-appropriate views; backend enforces. If backend visibility classification engineering work is incomplete, frontend renders inconsistent views.

**Mitigation:** Backend visibility classification work tracked separately. Frontend Phase 4 (audit trail rendering) verifies visibility classification works in integration test.

**Risk C2 — Authorisation evaluator latency under load.**

Frontend assumes <500ms p95 V1 latency on authorisation queries. If backend Authorisation evaluator under load exceeds this, frontend pharmacist actions become slow.

**Mitigation:** Backend performance testing under load before frontend Phase 4. Frontend implements optimistic UI for actions (action visually completes immediately; rollback if backend rejects) to mitigate user-perceived latency.

### 11.3 What's not a risk this plan addresses

- Layer 2-5 cognitive content authoring — deferred to senior consultant pharmacist
- Complex Resident Workspace activated mode — Tier 2 work, not Phase 4
- gRPC server — V1, not MVP
- Future role-views (RN, GP, EN, PCW) — separate plans

---

## Part 12 — Closing

### 12.1 What this plan commits to

- 8 shared primitives shipped before any of the 14 components
- 14 components shipped in 4 phases over 10-14 calendar weeks for two senior React engineers
- Mobile form-factor in MVP, with explicit touch interaction patterns per component
- v1.0-scoped per Adaptive Cognition Addendum; v1.1 unified rewrite deferred
- Testing per v1.0 Part 17 six categories
- Performance budget enforced per device class
- Accessibility WCAG 2.1 AA minimum
- Gate criteria per phase prevents downstream rework
- Coordination with backend Phase 2-completion specified

### 12.2 What this plan does not commit to

- Production deployment (gated on backend Phase 2-completion + external review + pilot pharmacist testing)
- Layer 2-5 cognitive content (deferred to senior consultant pharmacist authoring)
- gRPC server (REST sufficient for pilot)
- Future role-views (separate plans needed)
- Complex Resident Workspace activated mode (Tier 2 work)

### 12.3 What the engineers actually do

The plan gives the engineering team:
- The 14 components to build
- 8 primitives to build first
- The 4-phase sequence
- Per-component specifications
- Mobile-specific behaviour per component
- Testing approach
- Gate criteria

The plan does not specify:
- Component file structure beyond v1.0 Part 15 (engineers' choice within v1.0 conventions)
- State management library (engineers' choice — React Context is fine for this scale; Redux/Zustand if engineers prefer)
- Styling approach (Tailwind, CSS modules, styled-components — engineers' choice)
- Testing library specifics beyond "React Testing Library + Jest" (engineers' choice on visual regression tool, mocking approach, fixture management)
- Code organisation within components (engineers' decisions on hooks, composition patterns, abstraction levels)

The engineers' 10-year average experience is what makes these implementation decisions appropriate to delegate. The plan specifies what and why; engineers specify how.

### 12.4 What I want to flag honestly before this plan leaves the room

**One — the 12-14 calendar week estimate is realistic but not generous.** If pilot site engagement reveals significantly different workflow needs than v1.0 assumed, the plan may need re-work. Build in a 1-week buffer between Phase 4 completion and external clinical informatics UX review to absorb pilot-driven changes.

**Two — backend Phase 2-completion is the real critical path to production deployment.** Even if frontend ships in 12 weeks, production deployment is gated on backend Phase 2-completion (~8 tasks, several weeks). Plan the overall production-deployment timeline as max(frontend, backend), not sum.

**Three — the team's two engineers should sync on architectural decisions weekly.** Two senior engineers can build divergent solutions to the same problem if they don't coordinate. Weekly architectural sync (1 hour, lightweight) prevents drift. This is engineering discipline, not specification.

The plan is sufficient to begin engineering work tomorrow. Phase 0 primitive build starts when engineering team is ready. Calibrate to actual team availability; the estimates assume full-time engineering at 35-40 hours per engineer per week.

— Claude
