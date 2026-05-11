# Surface 2 (S2) — Resident Workspace Implementation Guidelines v1.0

**Date:** May 2026
**Surface scope:** Pharmacist's per-resident clinical workspace (Surface 2 of the seven user surfaces specified in *Decision Packet Rendering Implementation Guidelines v1.0*)
**Implementation phase:** Phase 1 of pilot deployment (pre-pilot blocker; engineering team needs this specification before substrate-consuming UI work begins)
**Status:** Tier 1 document A — pre-pilot blocker per the four-document Tier 1 sequence

**Builds on:**
- *Vaidshala v3.0 Product Proposal* (pharmacist as primary user; recommendation craft as core service)
- *Decision Packet Rendering Implementation Guidelines v1.0* (S2 rendering rules at the surface-taxonomy level; per-surface implementation deferred to this document)
- *CAPE v1.1 Implementation Guidelines* (the worklist engine that surfaces residents to S1, whose entries route to S2)
- *CAPE v1.1 Architectural Commitment Addendum* (multi-surface consumption; instability chronology as first-class primitive)
- *Recommendation Craft Engine Implementation Guidelines v1.0* (kb-32 produces the recommendations that S2 surfaces; lifecycle states, restraint signal pairing, confidence dimensions)
- *KB-29 Templates Implementation Guidelines v1.0* (template-fired recommendations consumed by S2)
- *Ethical Architecture Implementation Guidelines v1.0* (Principle 4 pharmacist autonomy; §8 algorithmic management protections)
- *Pharmacist Self-Visibility Implementation Guidelines v1.0* (S2 content is PDP visibility class)
- *Template Authoring Style Guide v1.0* (epistemic framing discipline; confidence-weighted recommendations)
- *Substrate Query Feasibility Analysis v1* (substrate dependencies flagged: care intensity state machine, recommendation lifecycle, negative-evidence search)

**Reading order:**
- Engineering and product leads: Parts 0–4, 14–18 (framing, architecture, layout, performance, implementation)
- Clinical informatics leads: Parts 1, 4–11 (philosophy, layout, trajectory, recommendations, restraint, history, goals-of-care, drill-through, complex activation)
- Senior consultant pharmacists: Parts 0, 4–12 (what the pharmacist actually sees and does)
- Design leads: Parts 4, 13, 14 (layout, audit integration, form-factor adaptations)
- Risk and ethics leads: Parts 1, 12, 13, 19 (philosophy, pharmacist actions, audit, risks)

---

## Part 0 — Honest framing: what S2 is and isn't

S2 is the pharmacist's primary clinical workspace for any individual resident. It is the surface where most pharmacist clinical reasoning happens after triage — they arrive at S2 from the S1 worklist (CAPE-prioritised), from a search, from a notification, or from cross-reference within another resident's chart, and they conduct their per-resident review.

This is the second-highest-frequency pharmacist surface after S1 worklist. In typical pilot deployment, a pharmacist will open S2 6–15 times per session, spending 10–45 minutes per resident depending on complexity. The engineering team needs this specification before substrate-consuming UI work begins, because S2 design decisions made locally will constrain how the substrate, CAPE outputs, kb-32 recommendations, restraint signals, and audit trail integrate.

### What this document specifies

- The S2 architectural foundation: what S2 consumes, where it sits in the stack, what it produces
- The four entry paths to S2 and how each shapes initial rendering
- The S2 view layout: top-to-bottom structure with named components
- Trajectory rendering specification (velocity, percentage change, threshold flags, baseline comparison)
- Pending recommendations panel: lifecycle states, confidence dimensions, restraint pairing
- Phase 1 advisory-only restraint signal rendering
- Failed intervention history surfacing
- Goals-of-care and care intensity state rendering
- Substrate observation one-click drill-through (verification-not-belief)
- Activation pathway from standard S2 to S2-Complex Resident Workspace mode
- Pharmacist actions and capture (open, defer, mark considered, note, action)
- Audit trail integration per ethical architecture §8
- Form-factor adaptations (mobile vs desktop)
- File and code organisation, API contracts, testing approach, performance budgets, implementation sequencing, risks

### What this document does NOT specify

- The visual design system (colours, typography, spacing, icon library) — design lead's authoring work, not architectural specification
- Specific UI mockups — UI design work that should follow this architectural specification
- The S2-Complex Resident Workspace rendering layer in full — that is Tier 2 document E per the sequenced plan, building on this S2 foundation
- The recommendation craft logic — that is kb-32 craft engine document
- The triage logic that populates S1 — that is CAPE engine document
- Per-template clinical content — that is KB-29 templates work
- The audit trail UX details across all surfaces — a future cross-cutting document if scoped

### What's distinctive about S2

S2 is the surface where the platform's substrate-up architecture becomes operational for the pharmacist's everyday clinical work. The distinctive elements:

**Distinctive 1: CAPE context carry-through.** When the pharmacist arrives at S2 from a CAPE worklist entry, the primary signals that drove the CAPE prioritisation appear at the top of S2 — not re-derived, but carried through. The pharmacist sees: "you came here because of [acute event severity 5: fall with injury 3 days ago] AND [trajectory velocity 4: rapid eGFR decline]." This prevents cognitive context-loss when transitioning from the cross-resident worklist view to the per-resident workspace.

**Distinctive 2: Trajectory primary, snapshot secondary.** S2 renders clinical parameters as trajectories with velocity, baseline comparison, and threshold-crossing flags — not as snapshot values. This is the trajectory rendering discipline from CAPE applied at the workspace level. The pharmacist sees what's moving, not just what is.

**Distinctive 3: Substrate verification at one click.** Every claim in S2 is one click from the underlying substrate observation. The pharmacist verifies before acting. This is the verification-not-belief principle from the recommendation craft engine operating at the workspace level.

**Distinctive 4: Phase 1 advisory-only restraint rendering.** Restraint signals appear alongside the recommendations they pair with, not as suppressive overlays. The pharmacist sees both the action recommendation and the restraint context, makes the integrated judgment. This is the Phase 1 advisory-only commitment from the restraint template work.

**Distinctive 5: Complex workspace activation as mode, not separate surface.** When CAPE flags a resident as meeting complex-workspace activation criteria, S2 surfaces this as an offered mode within S2 ("Open Complex Resident Workspace?"). The pharmacist activates the complex mode for the cases where it earns its place. For routine residents, standard S2 is the complete experience.

These five elements are what S2 leads with. The pending recommendations panel, restraint signal rendering, audit trail integration are necessary scaffolding; the architectural discipline above is what separates S2 from generic CDS resident summary views.

---

## Part 1 — Design philosophy

Seven principles, each grounded in prior architectural commitments.

**Principle 1: Organised retrieval, not analysis.** S2 organises the resident's substrate state across domains for the pharmacist to integrate. S2 does not analyse, conclude, or pre-integrate. The integration is the pharmacist's clinical contribution. This is the framing from the prior complex workspace synthesis applied at the standard S2 level: information assembly is more useful than analysis for cases the pharmacist already understands.

**Principle 2: Verification-not-belief at every claim.** Every fact, trajectory, signal, recommendation reasoning step in S2 is one click from the underlying substrate observation. The pharmacist can verify before acting. This is operationalised at the rendering level: no claim appears in S2 that cannot be traced to a substrate entity, observation, rule firing, or recommendation lifecycle event. This is craft engine Part 1 Principle 1 applied to S2.

**Principle 3: Pharmacist autonomy preserved.** S2 surfaces information and offers actions; it never blocks the pharmacist from any clinical judgment. Recommendations can be accepted, modified, deferred, or overridden. Restraint signals are advisory in Phase 1. Complex workspace activation is offered, not imposed. The pharmacist's autonomy is the architectural floor. This is ethical architecture §1 Principle 4 at the workspace level.

**Principle 4: Multi-dimensional retention.** When CAPE delivers a resident to S2, the multi-dimensional scoring that drove prioritisation is retained and surfaced — not collapsed to a single number. The dimensions are visible. This is the same discipline applied in CAPE; S2 must not undo it by re-collapsing to a summary score.

**Principle 5: Trajectory primary, snapshot secondary.** Clinical parameters render as trajectories first, snapshots second. The pharmacist sees what's moving, how fast, against what baseline, crossing what thresholds. The static current value is context, not the headline. This is Reviewer 1's contribution from the complex workspace discussion adopted at the standard S2 level.

**Principle 6: Audience-bounded rendering.** S2 is pharmacist-only by default (PDP visibility class per pharmacist self-visibility module). Content visible in S2 is the pharmacist's clinical workspace material, not material visible to other audiences. Aggregated patterns of pharmacist S2 use are not surveilled for performance evaluation per ethical architecture §8.

**Principle 7: Form-factor honest.** S2 must work on mobile and desktop. The pharmacist's reality is a hybrid of desktop sessions (deeper review, RMMR drafting) and mobile sessions (quick check between facility visits, after-hours review). The specification must be honest about what works on each form factor and what degrades gracefully.

---

## Part 2 — Architecture: where S2 sits

S2 is a rendering surface, not a service. It consumes from substrate and adjacent services to produce the pharmacist's per-resident clinical view.

### 2.1 Architectural positioning

```
                    ┌─────────────────────────────────┐
                    │ Layer 2 Substrate               │
                    │ (Clinical, Operational,         │
                    │  Consent, Care Intensity,       │
                    │  Goals-of-Care state machines)  │
                    └────────────┬────────────────────┘
                                 │ queries: trajectory,
                                 │ recent observations,
                                 │ acute events, state
                                 │ machine transitions
                                 ▼
┌──────────────────┐   ┌─────────────────────────────┐   ┌──────────────────┐
│ kb-33 CAPE       │──▶│ S2 Aggregation Service      │◀──│ kb-32 Craft      │
│ (worklist entry  │   │ (s2-aggregator)             │   │ Engine           │
│  context carry-  │   │                             │   │ (pending recs,   │
│  through)        │   │ • CAPE context retention    │   │  lifecycle       │
└──────────────────┘   │ • Substrate organisation    │   │  states,         │
                       │ • Trajectory computation    │   │  confidence      │
┌──────────────────┐   │ • Pending recs assembly     │   │  dimensions)     │
│ kb-29 Templates  │──▶│ • Restraint signal pairing  │   └──────────────────┘
│ (template-fired  │   │ • Failed intervention       │
│  recommendations │   │   history retrieval         │   ┌──────────────────┐
│  surfaced as     │   │ • GoC + care intensity      │◀──│ Restraint signal │
│  cards)          │   │   rendering                 │   │ evaluations      │
└──────────────────┘   │ • Complex activation check  │   │ (kb-29 restraint │
                       │ • Audit trail integration   │   │  templates)      │
                       └─────────┬───────────────────┘   └──────────────────┘
                                 │
                                 │ aggregated S2 view
                                 │
                                 ▼
                       ┌─────────────────────────────┐
                       │ S2 Rendering Layer          │
                       │ (frontend components per    │
                       │  Decision Packet Rendering  │
                       │  Guidelines)                │
                       └─────────┬───────────────────┘
                                 │
                                 │ pharmacist actions
                                 │
                                 ▼
                       ┌─────────────────────────────┐
                       │ EvidenceTrace audit trail   │
                       │ (per ethical architecture   │
                       │  §8 algorithmic management  │
                       │  protections)               │
                       └─────────────────────────────┘
```

### 2.2 What S2 consumes

**From Layer 2 substrate:**
- Clinical state machine: trajectories (eGFR, weight, cognition, function, vitals), recent observations, acute events
- Operational state machine: care plan updates, family meetings, room changes, transfers, recent staff interactions
- Consent state machine: consent state for clinical observations, restrictive practice consent
- Care intensity state machine: current state, transition history (subject to Substrate Feasibility Analysis flag — substantial substrate build may be required)
- Goals-of-care entity: documented goals, last update, family-meeting linkage

**From kb-33 CAPE (when entry from worklist):**
- The worklist entry that brought the pharmacist here
- The dimension scores that drove prioritisation
- The primary signals (top 1–3) and secondary signals
- The instability chronology if available (per CAPE Addendum)
- Linked pending recommendations identified by CAPE

**From kb-32 Craft Engine:**
- Pending recommendations on this resident
- Recommendation lifecycle states (detected, drafted, submitted, viewed, decided, monitoring-active)
- Confidence dimensions (substrate confidence, clinical confidence) per Style Guide Part 8
- Recommendation aging
- Outcome observation status

**From kb-29 templates:**
- Template-fired recommendations on this resident
- Restraint signal templates active (paired with action recommendations)
- Failed intervention history (previous recommendations that were declined or unsuccessful)

**From scheduling and operational context:**
- Scheduled reviews on this resident
- Upcoming family meetings
- MAC committee dates
- Specialist appointments

### 2.3 What S2 produces

For each S2 view request:

```go
type S2View struct {
    ResidentID            uuid.UUID
    PharmacistID          PharmacistID
    GeneratedAt           time.Time
    EntryPath             EntryPath
    
    // CAPE context (when entered from worklist)
    CAPEContext           *CAPEContextBand
    
    // Resident header
    ResidentHeader        ResidentHeader
    
    // Body sections
    Trajectories          []TrajectoryView
    PendingRecommendations []PendingRecommendationCard
    RestraintSignals      []RestraintSignalCard
    FailedInterventionHistory []FailedInterventionEntry
    GoalsOfCare           GoalsOfCarePanel
    CareIntensity         CareIntensityPanel
    
    // Complex workspace offer
    ComplexActivationOffer *ComplexActivationOffer
    
    // Audit trail
    AuditTraceRef         uuid.UUID
}
```

### 2.4 Service boundary

**S2 does:**
- Aggregate substrate and adjacent service data into the per-resident view
- Render trajectories from substrate observation series
- Surface pending recommendations and lifecycle states
- Pair restraint signals with action recommendations
- Offer complex workspace activation when criteria met
- Capture pharmacist actions and route to appropriate services
- Maintain audit trail per algorithmic management protections

**S2 does NOT:**
- Generate recommendations (kb-32 craft engine)
- Compute triage priority (CAPE engine)
- Author template clinical content (KB-29 templates)
- Modify substrate observations (substrate-up data integrity)
- Aggregate to employer view (visibility class enforcement)
- Make clinical decisions (pharmacist's contribution)

This boundary is critical. S2 is the pharmacist's rendering surface, not a clinical reasoning engine. The clinical reasoning happens elsewhere; S2 surfaces the substrate, the recommendations, the context — and the pharmacist's reasoning integrates them.

---

## Part 3 — Entry paths to S2

S2 has four distinct entry paths. Each shapes the initial rendering state.

### 3.1 Entry from CAPE worklist (highest-frequency entry)

The pharmacist clicks a worklist entry in S1. The S2 view loads with **CAPE context carry-through** at the top:

- The primary signals that drove the CAPE prioritisation
- The dimension scores breakdown
- The instability chronology if computed
- Quick-action links to the specific substrate observations that triggered the signals

Default scroll/focus position: top of view (CAPE context visible without scroll on desktop; collapsible on mobile).

This entry path is the most common during normal session work. The CAPE context band prevents the pharmacist from re-deriving why this resident appeared.

### 3.2 Entry from search

The pharmacist searches for a specific resident by name, room, or other identifier. The S2 view loads in **standard organised view** without CAPE-driven emphasis:

- Resident header
- Trajectories (all parameters)
- Pending recommendations
- Restraint signals (if any)
- Goals of care and care intensity
- Failed intervention history (collapsible)

No CAPE context band, because the pharmacist's entry was not triage-driven.

This entry path is common for targeted lookup — the pharmacist needs to check specific information they remember about this resident.

### 3.3 Entry from notification

A notification (e.g., new pathology result, new acute event, GP response to recommendation) routes the pharmacist to S2 for this resident. The view loads with **notification context band** at the top:

- The notification that brought them here (e.g., "Pathology result received: lithium level 0.92 mmol/L")
- The specific substrate event linked
- Default scroll/focus position: the relevant body section (e.g., trajectory section showing the new result in context)

This entry path supports event-driven workflow.

### 3.4 Entry from cross-reference

The pharmacist is reviewing another resident, sees a reference to this resident (e.g., "consider similar to Mrs Chen — also on lithium"), clicks through. The view loads in **comparative mode**:

- Resident header
- Body sections rendered in a layout that supports rapid comparison
- Optional split-screen on desktop
- Mobile: tabbed alternation between residents

This entry path is less common but operationally useful for pharmacists managing multiple complex residents simultaneously.

### 3.5 Entry path metadata

S2 captures entry path in audit trail. This supports:
- Performance analysis (which entry paths are most common per pharmacist?)
- UX iteration (are notification-entry workflows efficient?)
- Pharmacist self-visibility (PDP visibility class)

Entry path metadata is not used for performance evaluation per ethical architecture §8.

---

## Part 4 — S2 view layout

S2 has a defined top-to-bottom structure. Components render in order; some are collapsible by default; mobile adaptations alter density and layout.

### 4.1 Top region

**Component 1: Resident header** (always visible)
- Name, age, room, facility
- Frailty tier (CFS), care intensity state, goals-of-care summary status
- Quick links: full medication list, full pathology, care plan, family contacts
- Last-reviewed-by-you indicator with timestamp

**Component 2: CAPE context band** (visible only when entry from worklist)
- Primary signals from CAPE that drove prioritisation
- Dimension scores breakdown (acute event, trajectory velocity, recommendation aging, monitoring overdue, operational signals, restraint countermand)
- Instability chronology link (if computed per CAPE Addendum)
- Quick-jump links to relevant body sections

**Component 3: Notification context band** (visible only when entry from notification)
- The triggering event (substrate observation, GP response, pathology result)
- Linked substrate reference
- Quick-jump to relevant body section

**Component 4: Complex workspace activation offer** (visible only when activation criteria met — see Part 11)
- Brief explanation of why this resident meets complex activation criteria
- "Open Complex Resident Workspace" action button
- "Dismiss for this session" option

### 4.2 Body region

**Component 5: Trajectories panel** (always visible, default expanded)
- Multi-parameter trajectory rendering per Part 5
- Click any trajectory to drill into observation history

**Component 6: Pending recommendations panel** (always visible, default expanded)
- Cards for each pending recommendation per Part 6
- Confidence dimensions visible
- Restraint signal pairing where applicable

**Component 7: Restraint signals panel** (visible when restraint signals active, default collapsed if no paired recommendations)
- Active restraint signals per Part 7
- Phase 1 advisory-only mode rendering

**Component 8: Goals-of-care and care intensity panel** (always visible, default expanded)
- Current goals-of-care status per Part 9
- Care intensity state and transition history
- Family meeting recency

**Component 9: Failed intervention history** (collapsible, default collapsed)
- Previous recommendations declined or unsuccessful per Part 8
- Reasoning captured at time of decline

### 4.3 Bottom region

**Component 10: Pharmacist notes** (always visible, default expanded if notes present)
- Pharmacist's own notes on this resident (PDP visibility)
- Add note action
- Note history with timestamps

**Component 11: Pharmacist actions panel** (always visible, fixed position on desktop)
- Mark as reviewed (closes the worklist entry)
- Defer to next session
- Flag for follow-up
- Open complex workspace (if criteria met)
- Add note

**Component 12: Audit trail footer** (collapsible, default collapsed)
- Audit trail summary (PDP visibility)
- Recent S2 activity on this resident
- Detailed audit log link (opens audit view)

### 4.4 Layout density

- Desktop: all components visible with appropriate density; CAPE context band 4–8% of viewport height
- Tablet: components stack; some panels collapse by default
- Mobile: progressive disclosure with sectioned navigation; CAPE context band sticky at top

Per Principle 7 (form-factor honest), some pharmacist workflows (RMMR drafting, detailed regimen review) work primarily on desktop. Mobile S2 is optimised for quick check, action acknowledgment, and notification response.

---

## Part 5 — Trajectory rendering specification

Trajectories are the single most operationally important rendering component in S2. They are how the pharmacist sees what's moving.

### 5.1 Trajectory components per parameter

Each trajectory renders with:

**Current value** with units and observation date:
```
eGFR: 41 mL/min/1.73m² (2026-04-15)
```

**Velocity** computed per appropriate time unit:
```
Velocity: -5.8 mL/min/year over rolling 12 months
         -8.4 mL/min/year over recent 90 days (accelerating)
```

**Trajectory chart** showing observation series:
- Last 12 months minimum; longer if data supports
- Trend line with confidence interval
- Threshold-crossing markers (e.g., when crossed CKD stage 3b boundary)
- Recent observations highlighted

**Baseline comparison** as percentage and absolute change:
```
Baseline (12-month rolling): 52 mL/min/1.73m²
Current: 41 mL/min/1.73m²
Change: -11 (-21%) since baseline
```

**Threshold flags** for clinically relevant boundaries:
- AKI criteria
- CKD stage boundaries
- Therapeutic range boundaries (for drug levels)
- Clinical significance thresholds per parameter

### 5.2 Parameters rendered as trajectories

The following parameters render with full trajectory specification when sufficient data exists:

**Renal:** eGFR, serum creatinine, urea
**Cognitive:** MMSE, MoCA, 4AT (when serial)
**Functional:** CFS, ADL scores, mobility assessments
**Weight and nutrition:** weight (kg), BMI, MNA-SF when serial
**Vital signs:** systolic/diastolic BP (when serial), heart rate, oxygen saturation
**Therapeutic drug levels:** lithium, digoxin, warfarin INR, phenytoin
**Laboratory:** TSH, sodium, potassium, full blood count parameters
**Behavioural:** BPSD pattern indicators (when structured)

### 5.3 Sparse data graceful degradation

Not every resident has rich data for every parameter. Graceful degradation:

- **Single observation:** render as point with date, no velocity, no baseline
- **2–3 observations:** render observations with first-to-last delta; no formal velocity
- **4+ observations:** render full trajectory with velocity and baseline
- **Stale data:** trajectories with no observation in past 12 months render as "stale; last observation [date]"
- **Missing data category:** explicitly noted ("eGFR: no observation on record")

The discipline is to never produce false-precision trajectories from sparse data. A two-point "trajectory" is a delta, not a velocity.

### 5.4 Multi-parameter composition

When multiple parameters are concerning simultaneously, S2 surfaces the composition:

```
Multi-parameter trajectory pattern detected:
• Renal function declining (-21% from baseline)
• Cognitive decline accelerating (MMSE 22→18 over 3mo)
• Weight loss significant (-7kg over 2 months)

[Open multi-parameter view] [Show composition reasoning]
```

The "Open multi-parameter view" action routes to S2-Complex mode for the integrated rendering. The "Show composition reasoning" surfaces the substrate signals that the composition is computed from.

### 5.5 Trajectory drill-through

Clicking any trajectory opens the **observation history view**:
- Full observation series for this parameter
- Each observation linked to source (eNRMC, pathology lab, nursing assessment, etc.)
- Substrate confidence per observation
- Editable annotations (pharmacist notes per observation)

This is the verification-not-belief discipline operationalised — the pharmacist verifies the trajectory by examining its underlying observations.

---

## Part 6 — Pending recommendations panel

The pending recommendations panel surfaces kb-32-generated recommendations on this resident with full lifecycle context.

### 6.1 Pending recommendation card structure

Each pending recommendation renders as a card with:

**Header:**
- Recommendation summary (Layer 1 signal text per template)
- Lifecycle state badge (detected, drafted, submitted, viewed, decided)
- Urgency tier indicator (green, amber, red)
- Confidence dimensions (substrate confidence + clinical confidence per Style Guide Part 8)
- Age (days since detection)

**Body:**
- Layer 2 reasoning text (≤100 words per template)
- Linked substrate signals (verifiable)
- Restraint signal pairing badge if applicable
- Linked monitoring obligations if applicable

**Actions:**
- Open full recommendation (routes to recommendation detail surface)
- Modify before sending to GP
- Defer recommendation
- Override (with reasoning capture)
- Add note

### 6.2 Lifecycle state rendering

The pharmacist sees lifecycle state with clear semantics:

- **Detected:** substrate signal triggered template; not yet drafted or surfaced to GP
- **Drafted:** recommendation packet drafted; pharmacist may review or modify before submitting
- **Submitted:** sent to GP; awaiting response
- **Viewed:** GP has opened the recommendation
- **Decided:** GP has acted (accepted, modified, declined); decision logged
- **Monitoring-active:** decision made; monitoring obligations now active

State transitions are visible. The pharmacist sees when each transition occurred and the elapsed time since the last transition.

### 6.3 Confidence dimensions rendering

Per Style Guide Part 8, recommendations carry substrate confidence and clinical confidence:

```
Substrate confidence: high (5 sources searched, recent data, all consistent)
Clinical confidence: high (no concurrent risk factors, no specialist review pending)
```

Phase 1 deployment surfaces high-substrate-confidence recommendations primarily. Medium-confidence recommendations may appear with "verify substrate" prompts. Low-confidence recommendations are typically suppressed from pharmacist surfacing in Phase 1.

### 6.4 Restraint signal pairing rendering

When a pending recommendation has an active restraint signal paired (per kb-29 restraint templates), the card surfaces both:

```
PENDING RECOMMENDATION: STOP omeprazole (no documented indication)
[Substrate confidence: high | Clinical confidence: high]

⚠ PAIRED RESTRAINT SIGNAL ACTIVE: Care intensity transition recent
Resident transitioned to comfort care 8 days ago.
Suggested deferral period: 14 days.

Phase 1 mode: ADVISORY ONLY. No automatic lifecycle suppression.
Pharmacist judgment integrates both contexts.

[Review and decide] [Defer until +14 days] [Proceed with recommendation]
```

This is the Phase 1 advisory-only commitment from the restraint template work made visible in S2. The pharmacist sees both the action recommendation and the restraint context; integration is the pharmacist's clinical contribution.

### 6.5 Empty state

When a resident has no pending recommendations:

```
No pending recommendations on this resident.
Last recommendation: [date, brief description, outcome]
Next scheduled review: [date, type]
```

This is not a noisy state — many residents on most days have no pending recommendations. The empty state should be clean.

---

## Part 7 — Restraint signals rendering (Phase 1 advisory-only)

Restraint signals deserve their own panel because they operate differently from action recommendations.

### 7.1 Active restraint signals

When restraint signals are active on a resident, the panel surfaces each with:

- Signal type (e.g., care_intensity_transition_recent, recent_pathology_collection_attempt, specialist_review_scheduled)
- Substrate trigger (the specific substrate event that activated the signal)
- Suggested deferral period (per kb-29 restraint template)
- Active duration (how long since signal activated)
- Linked recommendations affected (if any)
- Phase 1 advisory-only badge

### 7.2 Pharmacist acknowledgment workflow

Per the Phase 1 advisory-only commitment, pharmacist acknowledgment is required for restraint signal interaction:

```
RESTRAINT SIGNAL: care_intensity_transition_recent
Activated: 2026-04-29 (12 days ago)
Trigger: Care intensity transitioned active → comfort_care

This signal advises deferral of routine deprescribing and non-urgent dose changes
for 14 days following the transition.

Affected pending recommendations on this resident:
• STOP omeprazole (currently in detected state) — restraint pairing active
• DOSE_CHANGE metformin (currently submitted, awaiting GP response) — restraint pairing active

Phase 1 mode: ADVISORY ONLY
The platform does not automatically suppress these recommendations.
Pharmacist judgment integrates both contexts.

[Acknowledge — proceed with affected recommendations]
[Acknowledge — defer affected recommendations]
[Acknowledge — case-by-case judgment per recommendation]
```

Pharmacist acknowledgment is logged in EvidenceTrace per ethical architecture §8.

### 7.3 Transition criteria visibility

Per the restraint template v1.1 specification, the Phase 1 advisory-only mode transitions to lifecycle suppression when criteria met (≥3 months evidence, ≥85% pharmacist agreement, zero safety incidents). The panel surfaces the platform-level status:

```
Phase 1 advisory-only mode status:
• Pilot evidence accumulated: 6 weeks
• Pharmacist agreement rate: pending
• Safety incidents: zero
• Transition to lifecycle suppression: not yet authorised
```

This is informational. Individual pharmacists do not transition the mode; the platform-level Clinical Informatics Committee does.

### 7.4 Safety-critical bypass rendering

Per the restraint template v1.1, safety-critical categories bypass restraint signals with mandatory documentation. When a pharmacist invokes safety-critical bypass:

```
SAFETY-CRITICAL BYPASS REQUESTED
Category: [select from explicit list]
• Toxic drug levels
• Severe drug interactions
• Acute symptoms requiring immediate medication
• Documented allergies/adverse reactions
• Pathology results requiring urgent intervention
• Hospital handoff reconciliation discrepancy

Pharmacist reasoning (required):
[text capture]

This bypass will be logged in EvidenceTrace and reviewed in monthly audit sampling.

[Confirm bypass] [Cancel]
```

---

## Part 8 — Failed intervention history

The failed intervention history component surfaces previous recommendations on this resident that were declined or unsuccessful.

### 8.1 What "failed intervention" means

Three categories:

**Category 1: Declined by GP.** A recommendation was submitted to the GP and declined (with or without reasoning capture). The override taxonomy per craft engine §5 categorises the decline.

**Category 2: Withdrawn by pharmacist.** A recommendation was generated but pharmacist withdrew before GP submission (e.g., upon learning new context).

**Category 3: Implemented but unsuccessful.** A recommendation was implemented (e.g., deprescribing carried out), but the outcome was unfavourable (e.g., symptom recurrence required restart).

### 8.2 Failed intervention card structure

Each failed intervention renders as:

```
PREVIOUS: STOP atorvastatin (declined 2026-02-15 by Dr Smith)

Original substrate signals (verifiable):
• CFS 7 (severely frail)
• Goals-of-care: comfort focus
• No recent cardiovascular event

GP decline reasoning captured:
"Family requested continuation; values long-standing prevention."

Outcome since decline: atorvastatin continued; no adverse events documented.

[Open original recommendation packet] [Open GP decline detail]
```

### 8.3 Why this matters

The failed intervention history serves three purposes:

**Purpose 1: Prevent unproductive re-recommendation.** Per restraint signal patterns, recently declined recommendations should not auto-resurface. The pharmacist sees the previous decline and judges whether circumstances have changed.

**Purpose 2: Learn the prescriber.** Per craft engine §8 per-GP framing learning, repeated declines from a specific GP for specific reasons inform how recommendations should be framed for that prescriber.

**Purpose 3: Document the audit trail.** Regulatory review (Standard 5, ACQSC) may ask why a recommendation was not made or repeated. The failed intervention history makes the reasoning visible.

### 8.4 Time horizon

Default display: failed interventions within 24 months. The pharmacist can extend the horizon to view longer history.

### 8.5 Failed intervention pattern detection

When multiple failed interventions cluster (e.g., three deprescribing declines for the same medication class), S2 surfaces a pattern indicator:

```
PATTERN DETECTED: Multiple failed deprescribing attempts
3 PPI deprescribing recommendations declined over past 18 months.

Consider:
• Whether the indication has been re-evaluated
• Whether family/resident preference is now documented in care plan
• Whether continuation is clinically aligned with current care intensity
```

This pattern detection is itself a restraint-adjacent signal — repeated declines suggest the platform should not keep generating the same recommendation.

---

## Part 9 — Goals-of-care and care intensity rendering

Goals-of-care and care intensity are foundational context for every clinical decision in aged care. S2 surfaces them prominently.

### 9.1 Goals-of-care panel

The goals-of-care panel surfaces:

**Current documented goals:**
- Free-text summary as captured in care plan
- Last update date and author
- Family-meeting linkage if applicable

**Goals freshness:**
- "Updated [date]" with relative time
- Flag if >12 months since last review
- Flag if care intensity has transitioned since last goals update

**Linked goals decisions:**
- ACD status (Advance Care Directive in place since when, reviewed when)
- Resuscitation preferences if documented
- Hospital transfer preferences if documented

### 9.2 Care intensity panel

The care intensity panel surfaces:

**Current state:**
- Active, active_with_recent_transition, comfort_care, palliative, end_of_life
- State entry date
- Time in current state

**Transition history:**
- Previous states and dates
- Last transition date
- Restraint signal pairing if recent transition

**Substrate Feasibility Analysis flag:** Per the Substrate Query Feasibility Analysis v1, the care intensity state machine is rated HIGHER RISK — requires substantial substrate build before reliable consumption. S2 must handle graceful degradation when care intensity state is not yet populated for a resident:

```
Care intensity state: not yet documented
Substrate feasibility note: care intensity state machine implementation in progress.
Free-text care intensity inference from care plan: [if available]
Pharmacist can document care intensity manually: [action]
```

### 9.3 Family communication context

When family meetings have occurred recently or are scheduled:

```
RECENT FAMILY ENGAGEMENT
• Family meeting 2026-04-25 — care intensity transition discussion
  Attendees: daughter Jane, son Mark, RN, GP, pharmacist (you)
  Documented family priorities: "wants to be settled, doesn't want hospital"

• Family meeting scheduled 2026-05-25 — care plan review
```

This context informs every pending recommendation interpretation. A deprescribing recommendation on a resident whose family last week discussed comfort goals lands differently than one on a resident with no recent family contact.

### 9.4 Conflict surfacing

When pending recommendations conflict with documented goals or care intensity, S2 surfaces the conflict explicitly:

```
⚠ CONTEXTUAL CONFLICT
Pending recommendation: ADD vitamin D supplementation
Documented goals: comfort focus; minimise pill burden
Care intensity: comfort care (transitioned 8 days ago)

This recommendation may conflict with current goals.
Pharmacist judgment integrates substrate signal + goals context.
```

The conflict is surfaced, not resolved. The pharmacist resolves it.

---

## Part 10 — Substrate observation drill-through

The verification-not-belief principle operationalised at the rendering level: every claim in S2 is one click from the underlying substrate observation.

### 10.1 Drill-through pattern

Every renderable element in S2 has an underlying substrate reference. Clicking the element opens the **substrate observation view** for that reference:

```
SUBSTRATE OBSERVATION: eGFR
Value: 41 mL/min/1.73m²
Observed: 2026-04-15 14:23
Source: pathology_lab (Sullivan Nicolaides)
Specimen collection: 2026-04-15 07:45
Reference range: 60-90 mL/min/1.73m² (adult)

Substrate confidence: high (pathology lab integration; specimen tracked)
Last calibration of integration: 2026-04-01 (verified)

Linked clinical context:
• Specimen collected during routine RACF pathology round
• No medication held for specimen collection
• Hydration status documented day-of: adequate

[Close] [Add pharmacist note] [Flag observation]
```

### 10.2 Drill-through depth

The substrate observation view itself supports further drill-through:
- The pathology lab integration audit log
- The specimen collection record
- The clinical context observations at time of specimen collection
- Pharmacist or clinical notes attached to the observation

This depth supports investigation when needed without imposing it when not needed. Routine pharmacist work uses surface-level rendering; investigation drills deeper.

### 10.3 Substrate confidence visibility

Every observation carries substrate confidence (high, moderate, low). This is visible at drill-through and surfaced in trajectory rendering when confidence is not high:

```
eGFR: 41 mL/min/1.73m² (2026-04-15) [confidence: moderate]

Substrate confidence: moderate
Reason: pathology lab integration recently updated; verification of result format in progress.
```

The pharmacist sees that the underlying data may need verification and can choose to verify before acting on the trajectory.

### 10.4 Negative-evidence rendering

Per Style Guide Part 3 and the Substrate Query Feasibility Analysis flag on negative-evidence search (rated medium-higher risk), negative-evidence claims render with epistemic humility:

```
"No current indication identified in available records"
[not "no indication exists"]

Substrate evidence reviewed:
✓ eNRMC indication field: not populated
✓ Progress notes searched (24 months): no current indication identified
✓ Care plan reviewed: no current indication documented

Substrate confidence: high
Note: negative-evidence claim. Substrate searched what was available; some sources
(e.g., scanned discharge summaries) may not be searchable in current implementation.
```

This is the verification-not-belief discipline applied to negative claims — the pharmacist sees what was searched and what wasn't.

### 10.5 Drill-through audit

Each drill-through is logged in EvidenceTrace. This supports:
- Pharmacist self-visibility of their verification patterns (PDP visibility)
- Performance analysis (which observations are most-verified — substrate confidence improvement targets)
- Anti-pattern detection (e.g., are pharmacists never verifying? — might indicate over-reliance)

Per ethical architecture §8, verification patterns are not surveilled for performance evaluation.

---

## Part 11 — Complex workspace activation

When a resident meets complex workspace activation criteria, S2 offers activation. The complex workspace is a mode of S2, not a separate surface.

### 11.1 Activation criteria

Per the prior synthesis on Complex Resident Workspace, activation criteria are conservative:

**Required (any of):**
- CFS ≥6 (severely frail or worse)
- ≥3 active high-risk medications (medications with significant adverse-event profile in aged care)
- ≥2 trajectory declines across systems (concurrent decline in renal + cognitive + functional, etc.)
- Recent care intensity transition (within 14 days for destabilisation, 21 days for transition_to_comfort, 28 days for transition_to_palliative)
- Recent acute event documented (fall with injury, hospital transfer, rapid response within 14 days)

Activation is triggered by **any** of these criteria being present. The thresholds are deliberately conservative (under-activate, not over-activate) to prevent alert fatigue.

### 11.2 Activation offer rendering

When criteria are met, the complex workspace activation offer appears in the top region of S2:

```
COMPLEX WORKSPACE ACTIVATION AVAILABLE

This resident meets activation criteria:
• CFS 7 (severely frail)
• 4 active high-risk medications (lithium, perindopril, frusemide, citalopram)
• 2 trajectory declines (renal -21% from baseline; cognitive MMSE 22→18 in 3mo)
• Recent care intensity transition (comfort_care, 12 days ago)

Complex Resident Workspace provides multi-domain situation board,
concern vector surfacing, instability chronology rendering, and
"what experts typically check" memory aid.

[Open Complex Resident Workspace] [Continue with standard S2] [Dismiss for session]
```

### 11.3 Pharmacist choice

The pharmacist chooses activation. The platform does not enforce. Two reasons:

**Reason 1: Pharmacist autonomy.** The platform suggests; the pharmacist decides. Some pharmacists prefer to work in standard S2 even for complex residents, integrating the multi-domain context themselves.

**Reason 2: Calibration evidence.** Phase 1 deployment is the period when activation criteria calibration is being evaluated. Pharmacist choice patterns inform whether criteria are too aggressive or too conservative.

### 11.4 Dismissal behaviour

If the pharmacist dismisses for session, the offer does not re-appear during the current session for this resident. If the pharmacist dismisses across sessions, the offer re-appears at next session (criteria still met).

### 11.5 Activation logging

Per ethical architecture §8, complex workspace activation choices are logged. Patterns inform calibration but are not surveilled for performance evaluation.

### 11.6 Tier 2 document linkage

The Complex Resident Workspace mode itself is specified in Tier 2 document E (per the four-tier sequenced plan). This document specifies only the activation offer integration within standard S2. The complex mode rendering — concern vectors, situation board, instability chronology surface — is the Tier 2 work.

---

## Part 12 — Pharmacist actions and capture

S2 supports a defined set of pharmacist actions with explicit capture.

### 12.1 The action set

**Action 1: Open recommendation.** Routes to the full recommendation detail surface where the pharmacist can review, modify, submit to GP.

**Action 2: Modify recommendation.** Inline modification of recommendation before submission (e.g., adjust dose, change wording per GP framing preference).

**Action 3: Defer recommendation.** Mark recommendation for re-surface at specified date. Reasoning capture optional.

**Action 4: Override recommendation.** Decline to send recommendation to GP. Reasoning capture mandatory (per override taxonomy per craft engine §5).

**Action 5: Mark resident as reviewed.** Closes the worklist entry; updates CAPE state.

**Action 6: Flag for follow-up.** Adds resident to deferred list with re-surface date.

**Action 7: Add pharmacist note.** Captures PDP-visibility note on resident.

**Action 8: Open complex workspace.** Activates complex mode (when offered).

**Action 9: Drill into substrate observation.** Verifies specific claim.

**Action 10: Acknowledge restraint signal.** Phase 1 advisory-only acknowledgment.

**Action 11: Invoke safety-critical bypass.** With mandatory reasoning capture.

### 12.2 Action capture data model

```go
type PharmacistAction struct {
    ID                uuid.UUID
    PharmacistID      PharmacistID
    ResidentID        uuid.UUID
    EntryPath         EntryPath
    ActionType        ActionType
    LinkedEntities    []EntityReference  // recommendation, substrate observation, etc.
    Reasoning         *string             // optional or mandatory per action type
    Timestamp         time.Time
    SessionContext    SessionContext
    AuditTraceID      uuid.UUID
}
```

### 12.3 Mandatory vs optional reasoning capture

Reasoning capture is mandatory for:
- Override recommendation (per craft engine override taxonomy)
- Invoke safety-critical bypass (per restraint template safety-critical bypass requirements)

Reasoning capture is optional but encouraged for:
- Defer recommendation
- Flag for follow-up
- Add pharmacist note

Reasoning capture is not requested for:
- Open recommendation
- Drill into substrate observation
- Acknowledge restraint signal (the acknowledgment itself is the capture)

### 12.4 Session context

Each pharmacist action is captured in session context:

```go
type SessionContext struct {
    SessionID         uuid.UUID
    PharmacistID      PharmacistID
    FacilityID        FacilityID
    SessionStartTime  time.Time
    ResidentsReviewedInSession []uuid.UUID
    EntryPathToS2     EntryPath
    TimeOnResident    time.Duration
}
```

The time-on-resident metric supports calibration learning (how long pharmacists typically spend per resident at each complexity tier). Per ethical architecture §8, this is not surveilled for performance evaluation.

---

## Part 13 — Audit trail integration

Every S2 action and rendering decision is auditable per ethical architecture §8.

### 13.1 What gets audited

**S2 view rendering:**
- View loaded (when, what entry path, what content rendered)
- Substrate observations drilled through
- Components expanded or collapsed

**Pharmacist actions:**
- All 11 actions in Part 12.1
- Reasoning captured (where applicable)
- Modification details (where applicable)

**System events:**
- Substrate update triggers re-render
- Recommendation lifecycle state transitions
- Restraint signal activations and deactivations
- Complex workspace activation offers and pharmacist choices

### 13.2 Audit data structure

```go
type S2AuditEvent struct {
    ID                uuid.UUID
    EventType         string
    Timestamp         time.Time
    PharmacistID      PharmacistID
    ResidentID        uuid.UUID
    SessionID         uuid.UUID
    EntryPath         EntryPath
    ActionDetails     map[string]interface{}
    SubstrateRefs     []SubstrateReference
    EvidenceTraceID   uuid.UUID
}
```

### 13.3 Visibility class enforcement

Per pharmacist self-visibility module:
- S2 audit events are PDP visibility (pharmacist's own clinical workspace activity)
- Aggregated patterns (PWA — pharmacist work aggregate) require pharmacist consent for cross-pharmacist analysis
- Employer/manager visibility (PEV — pharmacist employer view) is restricted per visibility class enforcement
- Regulator visibility (REG) under formal data-sharing agreement only

### 13.4 Algorithmic management protections

Per ethical architecture §8:
- S2 audit data is NOT used for pharmacist performance evaluation
- S2 audit data is NOT shared with employer for performance monitoring
- Patterns inform calibration learning (Phase 4) and quality improvement (anonymised aggregate)
- Contestation pathway: pharmacist can challenge any S2 audit-derived inference

### 13.5 Audit trail surfacing in S2

The audit trail footer (Component 12 of layout) surfaces:
- Pharmacist's own recent S2 activity on this resident
- Audit summary (what's been audited, what visibility class)
- Link to detailed audit log view

This is meta-visibility — the pharmacist sees what the platform sees about their work.

---

## Part 14 — Form-factor adaptations

S2 must work on both desktop and mobile. Different workflows; different density.

### 14.1 Desktop S2

Primary form factor for:
- RMMR drafting sessions
- Detailed resident review during pharmacist's facility-day
- Multi-resident comparative review
- Complex workspace mode

Layout: full-width responsive with side panels for actions; trajectories render with full chart visibility; pending recommendations cards render with full detail visible.

### 14.2 Mobile S2

Primary form factor for:
- Quick checks between facility visits
- After-hours notification response
- Brief pharmacist-GP phone-call context lookup
- Pharmacy-back-at-base review of facility activity

Layout: progressive disclosure with sectioned navigation; CAPE context band sticky at top; trajectories render as simplified series with detail on tap; pending recommendation cards collapse to summary with expansion on tap.

### 14.3 Mobile-specific limitations

Some workflows degrade gracefully on mobile:
- Inline recommendation modification: limited to dose/timing fields; full modification requires desktop
- Multi-parameter trajectory composition: simplified rendering; detail on desktop
- Failed intervention pattern detection: visible but limited interaction; investigation on desktop
- Audit trail detail: summary only; full audit on desktop

These limitations are honest — mobile is for quick check and acknowledgment, not deep clinical work. Pharmacists who need deep work should be on desktop.

### 14.4 Offline behaviour

Limited offline support per pilot scope:
- Cached recent S2 views (last 5 residents viewed in session) available offline
- Pharmacist actions queue and sync when online
- Substrate observations rendered from cache with staleness indicators

Full offline support is post-pilot. Phase 1 deployment assumes connectivity except for brief drops.

### 14.5 Accessibility

S2 must support pharmacist accessibility requirements:
- Screen reader compatibility (WCAG 2.1 AA)
- Keyboard navigation throughout
- Font size adjustment without layout break
- Colour-blind-safe trajectory rendering (not relying on red/green alone)
- High-contrast mode

Accessibility audit at end of Phase 1 implementation per the pilot design.

---

## Part 15 — File and code organisation

```
backend/services/s2-aggregator/
├── cmd/
│   └── server/main.go
├── internal/
│   ├── api/
│   │   ├── grpc.go                # gRPC API
│   │   ├── http.go                # HTTP API for S2 view requests
│   │   └── events.go              # Event subscriptions
│   ├── store/
│   │   ├── postgres/
│   │   │   ├── s2_views.go
│   │   │   ├── pharmacist_actions.go
│   │   │   ├── audit_events.go
│   │   │   └── migrations/
│   │   └── redis/
│   │       └── s2_view_cache.go   # short TTL for current S2 views
│   ├── aggregation/
│   │   ├── view_builder.go        # builds S2View from inputs
│   │   ├── cape_context.go        # CAPE context carry-through
│   │   ├── trajectories.go        # multi-parameter trajectory rendering
│   │   ├── pending_recs.go        # pending recommendations panel
│   │   ├── restraint_signals.go   # Phase 1 advisory-only rendering
│   │   ├── failed_history.go      # failed intervention history
│   │   ├── goals_care_intensity.go # GoC and care intensity rendering
│   │   ├── complex_activation.go  # complex workspace activation check
│   │   └── tests/
│   ├── entry_paths/
│   │   ├── from_worklist.go       # CAPE worklist entry
│   │   ├── from_search.go         # search entry
│   │   ├── from_notification.go   # notification entry
│   │   ├── from_cross_reference.go # cross-reference entry
│   │   └── tests/
│   ├── drill_through/
│   │   ├── substrate_observation.go # substrate observation view
│   │   ├── trajectory_history.go   # observation series for parameter
│   │   ├── negative_evidence.go    # negative-evidence rendering
│   │   └── tests/
│   ├── actions/
│   │   ├── handlers.go             # the 11 pharmacist actions
│   │   ├── reasoning_capture.go    # mandatory/optional reasoning
│   │   ├── session_context.go      # session metadata
│   │   └── tests/
│   ├── audit/
│   │   ├── evidence_trace.go       # EvidenceTrace integration
│   │   ├── visibility_class.go     # visibility class enforcement
│   │   └── tests/
│   ├── form_factor/
│   │   ├── desktop.go              # desktop layout adapter
│   │   ├── mobile.go               # mobile layout adapter
│   │   └── tests/
│   └── erm_integration/
│       └── ethical_review.go       # quarterly ERM review integration

frontend/components/s2/
├── ResidentWorkspace.tsx           # main S2 component
├── CAPEContextBand.tsx
├── NotificationContextBand.tsx
├── ResidentHeader.tsx
├── TrajectoriesPanel.tsx
│   └── TrajectoryChart.tsx
├── PendingRecommendationsPanel.tsx
│   └── RecommendationCard.tsx
├── RestraintSignalsPanel.tsx
│   └── RestraintSignalCard.tsx
├── FailedInterventionHistory.tsx
├── GoalsOfCarePanel.tsx
├── CareIntensityPanel.tsx
├── ComplexActivationOffer.tsx
├── PharmacistNotes.tsx
├── PharmacistActionsPanel.tsx
├── AuditTrailFooter.tsx
└── tests/
```

---

## Part 16 — API contracts

```protobuf
service S2WorkspaceService {
    // View request
    rpc GetResidentWorkspace(WorkspaceRequest) returns (S2View);
    rpc RefreshResidentWorkspace(RefreshRequest) returns (S2View);
    
    // Drill-through
    rpc GetSubstrateObservation(ObservationRequest) returns (SubstrateObservation);
    rpc GetTrajectoryHistory(TrajectoryRequest) returns (TrajectoryHistory);
    
    // Pharmacist actions
    rpc OpenRecommendation(ActionRequest) returns (Acknowledgment);
    rpc ModifyRecommendation(ModificationRequest) returns (Acknowledgment);
    rpc DeferRecommendation(DeferRequest) returns (Acknowledgment);
    rpc OverrideRecommendation(OverrideRequest) returns (Acknowledgment);
    rpc MarkResidentReviewed(ReviewRequest) returns (Acknowledgment);
    rpc FlagForFollowUp(FollowUpRequest) returns (Acknowledgment);
    rpc AddPharmacistNote(NoteRequest) returns (Acknowledgment);
    rpc OpenComplexWorkspace(ActivationRequest) returns (S2ComplexView);
    rpc AcknowledgeRestraintSignal(RestraintAckRequest) returns (Acknowledgment);
    rpc InvokeSafetyCriticalBypass(BypassRequest) returns (Acknowledgment);
    
    // Audit
    rpc GetS2AuditTrail(AuditRequest) returns (AuditTrail);
    
    // Session
    rpc StartSession(SessionStartRequest) returns (SessionContext);
    rpc EndSession(SessionEndRequest) returns (SessionSummary);
}
```

---

## Part 17 — Testing approach

Six test categories specific to S2.

### Category 1: View assembly tests

- S2View constructed correctly from each entry path
- CAPE context carries through when entry path is worklist
- Substrate organisation across domains correct
- Complex activation criteria correctly evaluated
- Graceful degradation when substrate data sparse

### Category 2: Trajectory rendering tests

- Velocity computation correct for various data densities
- Sparse data degraded gracefully (no false-precision trajectories)
- Threshold-crossing flags trigger correctly
- Multi-parameter composition correct
- Baseline computation correct

### Category 3: Pending recommendation tests

- Confidence dimensions visible
- Lifecycle states render correctly
- Restraint signal pairing surfaces correctly
- Empty state renders cleanly
- Failed intervention history surfaces past declines

### Category 4: Restraint signal rendering tests

- Phase 1 advisory-only mode rendered correctly
- Pharmacist acknowledgment workflow functions
- Safety-critical bypass requires mandatory reasoning
- Transition criteria visibility informational only

### Category 5: Drill-through tests

- Every renderable element has substrate reference
- Drill-through opens substrate observation view correctly
- Substrate confidence visible
- Negative-evidence rendering uses epistemic humility framing

### Category 6: Audit trail tests

- All 11 actions logged correctly
- Visibility class enforcement correct
- Algorithmic management protections honoured (no surveillance use)
- Drill-through events logged

### Critical: Verification-not-belief test

Every claim in S2 must be traceable to substrate. This is a structural test:

```go
func TestEveryClaimHasSubstrateReference(t *testing.T) {
    view := buildTestS2View()
    
    claims := extractAllClaims(view)
    for _, claim := range claims {
        require.NotNil(t, claim.SubstrateRef,
            "Claim '%s' has no substrate reference — violates verification-not-belief discipline",
            claim.Text)
    }
}
```

---

## Part 18 — Performance budgets

| Operation | p95 budget | Hard cap |
|---|---|---|
| GetResidentWorkspace (cold) | 1500ms | 4000ms |
| GetResidentWorkspace (warm, cached) | 400ms | 1000ms |
| RefreshResidentWorkspace | 800ms | 2000ms |
| GetSubstrateObservation (drill-through) | 200ms | 500ms |
| GetTrajectoryHistory | 600ms | 1500ms |
| Open recommendation | 300ms | 800ms |
| Pharmacist action (defer, flag, note) | 200ms | 500ms |
| Override recommendation with reasoning | 500ms | 1200ms |
| Complex workspace activation check | 200ms | 500ms |

Performance budgets reflect that S2 view generation is the dominant operation. Pharmacists open multiple residents per session; each open must feel responsive. Drill-through must be near-instant to support the verification-not-belief workflow.

---

## Part 19 — Implementation sequencing

Phase 1 pre-pilot deployment, Weeks 8–14 of pilot preparation per the master implementation plan.

### Week 8: Foundation

- Service scaffold (s2-aggregator)
- Storage and event subscriptions
- Basic view builder with substrate query patterns
- Performance budgets in place

### Week 9: Entry paths and CAPE integration

- Four entry path handlers
- CAPE context carry-through implementation
- Notification context band
- Search and cross-reference entry

### Week 10: Trajectories and substrate drill-through

- Multi-parameter trajectory rendering
- Velocity and baseline computation
- Threshold-crossing flag detection
- Substrate observation view (drill-through)

### Week 11: Pending recommendations and restraint signals

- Pending recommendation cards
- Confidence dimensions surfacing
- Phase 1 advisory-only restraint rendering
- Safety-critical bypass workflow

### Week 12: Failed intervention history, goals-of-care, care intensity

- Failed intervention history surfacing
- Failed intervention pattern detection
- Goals-of-care panel
- Care intensity panel with substrate degradation handling

### Week 13: Complex activation, pharmacist actions, audit

- Complex workspace activation offer (Tier 2 implementation deferred but offer surfacing in place)
- All 11 pharmacist actions with reasoning capture
- EvidenceTrace integration
- Visibility class enforcement

### Week 14: Form factor adaptations, testing, external review

- Desktop and mobile rendering paths
- Accessibility audit baseline
- Cross-component integration testing
- External clinical informatics UX review
- Pilot pharmacist user testing (3 pharmacists, 1 week)

### Estimated team

- 2 backend engineers (full-time, 7 weeks)
- 2 frontend engineers (full-time, 7 weeks)
- 1 clinical informatics lead (part-time, 7 weeks for substrate query patterns, content review)
- 1 senior consultant pharmacist (part-time, 4 weeks for workflow validation)
- 1 UI designer (part-time, 6 weeks for visual design system applied to S2 components)
- External clinical informatics consultant (3-day intensive review at week 14)
- 3 pilot pharmacists for user testing (1 week at week 14)

---

## Part 20 — Risks and mitigations

**Risk 1: Information overload.** S2 surfaces too much; pharmacists scan rather than engage. Mitigation: progressive disclosure with default-collapsed panels for less-frequent content; CAPE context band focuses entry attention; complex workspace mode for genuinely complex cases.

**Risk 2: CAPE context lost in transition.** Pharmacist arrives at S2 from worklist but the context that drove prioritisation isn't visible. Mitigation: CAPE context band as first-class component; sticky position on mobile; quick-jump links to relevant body sections.

**Risk 3: Substrate feasibility gaps cause empty rendering.** Care intensity state machine, recommendation lifecycle, or negative-evidence search not yet built; S2 renders empty panels. Mitigation: graceful degradation in every panel (per Part 9.2 example); explicit substrate-gap surfacing; manual override workflows for pharmacist documentation.

**Risk 4: Phase 1 advisory-only restraint creates confusion.** Pharmacists confused about whether restraint signals are blocking or advisory. Mitigation: explicit "Phase 1 mode: ADVISORY ONLY" badge on every restraint signal; pharmacist acknowledgment workflow makes the advisory-only nature operational; transition criteria visibility informational.

**Risk 5: Complex workspace activation creates competing surfaces.** Pharmacists confused by multiple workspace modes. Mitigation: complex workspace is a *mode* of S2, not a separate surface; activation criteria conservative (under-activate); pharmacist choice; dismissal behaviour clear.

**Risk 6: Trajectory rendering misleads with sparse data.** A 2-point "trajectory" creates false confidence in a trend. Mitigation: graceful degradation per Part 5.3; explicit handling of sparse data; never produce false-precision velocity from 2 observations.

**Risk 7: Drill-through performance degrades trust.** Slow substrate observation view erodes verification-not-belief workflow. Mitigation: aggressive performance budget for drill-through (200ms p95); cache substrate observations; pre-fetch likely drill targets.

**Risk 8: Audit trail surveillance use.** Pharmacist concerns about S2 audit data being used for performance evaluation. Mitigation: explicit ethical architecture §8 protections; pharmacist self-visibility module governs visibility classes; contestation pathway visible; informational rendering of "what is and isn't surveilled."

**Risk 9: Mobile workflow degrades clinical safety.** Pharmacists doing critical work on mobile when desktop is appropriate. Mitigation: honest form-factor limitations (Part 14.3); critical workflows (RMMR, recommendation override with reasoning) require desktop or full-screen mobile; mobile is for quick check and acknowledgment.

**Risk 10: Failed intervention history becomes overwhelming.** Resident with 20+ previous declined recommendations buries current context. Mitigation: default time horizon 24 months; pattern detection consolidates repeats; collapsible by default.

**Risk 11: Pharmacist action capture friction.** Mandatory reasoning capture for override/bypass slows workflow. Mitigation: structured reason taxonomy (per override taxonomy per craft engine §5); quick-select common reasons; free-text optional; bypass reasoning template for common safety-critical categories.

**Risk 12: Contextual conflict surfacing missed.** Pharmacist doesn't notice when pending recommendation conflicts with goals-of-care. Mitigation: contextual conflict explicit surfacing (Part 9.4); conflict badge visible at recommendation card level; conflict reasoning visible.

---

## Part 21 — Closing

Three observations as we close v1.0 of this specification.

**One:** S2 is the pharmacist's primary clinical workspace, and the design decisions in this document determine whether the substrate-up architecture becomes operational or remains abstract. The discipline of CAPE context carry-through, verification-not-belief at every claim, multi-dimensional retention, trajectory primary, and Phase 1 advisory-only restraint rendering — these are not optional. They are the architectural commitments that distinguish S2 from generic CDS resident summary views and from the alert-fatigue tools the literature documents.

**Two:** S2 sits at the intersection of CAPE (which surfaces the resident), kb-32 (which produces the recommendations), kb-29 (which defines the templates), the substrate (which holds the clinical reality), and the pharmacist (who integrates everything). The integration burden is large; the design choice is to organise rather than analyse, to surface rather than collapse, to verify rather than assert. The pharmacist's clinical reasoning is the integration; S2's role is supporting that reasoning without performing it.

**Three:** What this document does not specify — the visual design system, the specific UI mockups, the complete S2-Complex rendering — is deferred to subsequent work for good reason. Architectural specifications constrain design decisions but do not perform them. The design lead's authoring work on the visual system follows this specification; the UI mockups follow the design system; the S2-Complex specification (Tier 2 document E) builds on the activation offer mechanism specified here.

What I'd flag for the team as the most consequential implementation choices:

- **The Substrate Feasibility Analysis flags must be honoured.** Care intensity state machine, recommendation lifecycle, and negative-evidence search are flagged as higher-risk substrate dependencies. S2's graceful degradation behaviour must work even when these substrates are incomplete. The risk is that S2 ships with elegant rendering of substrate that doesn't yet exist, then fails when deployed against real data gaps.

- **The CAPE integration is the single highest-value design element.** When pharmacists arrive at S2 from a worklist entry, they should feel that the platform remembers why they came. The CAPE context band is what makes this real. Implementation cannot skimp on this.

- **The verification-not-belief discipline must be testable.** Part 17 Category 6 specifies the structural test for every claim having a substrate reference. This test must run continuously in CI and any addition to S2 that violates it must fail the build. The discipline only holds if it is enforced architecturally.

- **The Phase 1 advisory-only restraint rendering is a pilot calibration commitment.** The transition criteria to lifecycle suppression (≥3 months evidence, ≥85% pharmacist agreement, zero safety incidents) live outside this document but are operationally visible through S2. S2 must surface the transition status without enabling individual pharmacists to bypass the calibration evidence requirement.

The architecture stack now stands at:

1. v3.0 strategic positioning
2. Pilot design
3. Recommendation craft engine
4. Pharmacist self-visibility
5. Ethical architecture
6. Decision packet rendering
7. KB-29 templates
8. KB-29 maturity roadmap
9. CAPE v1.1 (Clinical Attention Prioritisation Engine)
10. CAPE v1.1 architectural commitment addendum
11. Template Authoring Style Guide v1.0
12. Substrate Query Feasibility Analysis v1
13. **S2 Resident Workspace Implementation Guidelines v1.0** ← this document

Three v1.1 calibration templates (PPI, lithium, restraint) are positioned for senior consultant pharmacist clinical validation in parallel with engineering substrate verification.

Tier 1 remaining: S3 GP Communication Hub (document B), RMMR Workflow (document C), CPD and AHPRA Records (document D). Each is one sitting and follows the same architectural pattern established in this document.

The engineering team now has the specification needed to begin S2 implementation. The clinical informatics lead and senior consultant pharmacist now have the surface design they will validate during pilot deployment. The design lead now has the architectural constraints within which the visual design system applies.

— Claude

---

**End of S2 Resident Workspace Implementation Guidelines v1.0**
