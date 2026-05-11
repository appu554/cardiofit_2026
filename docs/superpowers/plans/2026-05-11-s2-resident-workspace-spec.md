# S2 Resident Workspace — Specification

> **Status:** Pre-pilot design specification. Authored before any S2 implementation work. Owned by clinical informatics; consumed by kb-33 worklist (Step 5) and the eventual S2 implementation plan.

**Audience:** The accredited consultant pharmacist conducting medication review at a residential aged care facility. Surface is *not* for residents, families, GPs, facility operators, or auditors directly — those are S3, S4, S5, and the audit query API respectively.

**Source-of-truth references:**
- `docs/superpowers/plans/CAPE_Implementation_Guidelines_v1_1.md` (structural template + audience-adaptation contract)
- `docs/superpowers/plans/CAPE_v1_1_Architectural_Commitment_Addendum.md` lines 232–345 (Chronology audience adaptation; S2 = `AudiencePharmacist`)
- `docs/superpowers/plans/2026-05-09-phase-2-completion.md` (kb-32 craft engine, override taxonomy, ethics gates that S2 must respect)
- `docs/superpowers/plans/2026-05-11-cape-substrate-prerequisites.md` (the substrate types S2 reads)

---

## Part 0 — Operational test

> A consultant pharmacist arrives at a resident's record after triaging from S1 worklist. Within 30 seconds, S2 must give them: (a) the resident's current clinical posture (eGFR, DBI, ACB, CFS, care intensity, capacity status), (b) the active recommendation set with urgency and appropriateness scores, (c) any capacity or restrictive-practice hold blocking action, (d) the most recent instability chronology and PRN velocity signal that explain *why* the resident surfaced in their worklist. From this surface they can drill to evidence, capture an override, or push a recommendation to the GP via S3 — without re-reading the underlying record. If the pharmacist cannot answer "why is this resident in front of me, and what would I do next?" inside 30 seconds, S2 has failed its operational test.

---

## Part 1 — Design philosophy

S2 exists because the S1 worklist is row-level — it ranks residents by signal severity but cannot show *why* without becoming the surface S2 is. The split is deliberate: S1 optimises for triage throughput, S2 optimises for single-resident decision quality.

Five principles shape every interaction:

1. **Substrate-grounded.** Every claim S2 renders traces to a substrate primitive in `shared/v2_substrate/` (ClinicalSnapshot fields, FailedInterventionRecord, VelocityResult, InstabilityChronology) or a craft-engine output (kb-32 Packet, Assessment, RecommendationCitation). No synthesised metrics, no UI-only computed values. If S2 displays a number, that number is queryable from a Go type.

2. **Audit-defensible.** Every recommendation surfaced carries its fire-time citation pin (per `citations.PinAtFireTime` and the `/v1/explain` endpoint wired in CAPE substrate Task A). Source amendments after fire time do not retroactively alter what S2 displays — the citation drawer always shows what the pharmacist actually saw.

3. **Audience-neutral facts + audience-adapted rendering.** Per CAPE Addendum §3, the substrate is audience-neutral; rendering is per-audience. S2 receives `InstabilityChronology` and renders via `AudienceAdaptations[AudiencePharmacist]` — narrative tuned for clinical reasoning, not family explanation or governance audit. The same chronology rendered for S3 or for the family surface would emphasise different events.

4. **Override pathway always available.** Phase 3 CI invariance gate (`override_pathway_test.go` — Phase 3 commit `266d000b`) asserts that every blocked recommendation must surface a clinician override route. S2 honours this: a STOP recommendation held by appropriateness gate or capacity hold MUST present an override pathway on the same view, not behind a confirmation modal, dialog tree, or hover-revealed control.

5. **Frame-vs-content invariance.** Per Phase 3 commit `266d000b` (`framing/invariance_test.go`), clinical content must not vary based on framing tone. S2's rendering of the recommendation Layer 1 body is invariant across pharmacist tone preferences; only the GP-facing framing in S3 adapts. Pharmacist sees content equivalent to what the GP sees, prior to S3's per-GP-observer rendering.

---

## Part 2 — Surface architecture

S2 is a single resident workspace, organised into eight panels. Layout is logical, not pixel-bound; final visual rendering happens in implementation.

```
┌─────────────────────────────────────────────────────────────────┐
│  (R) Resident header                                            │
├─────────────────────────────┬───────────────────────────────────┤
│  (A) Active recommendations │  (C) Capacity + restrictive-      │
│      panel                  │      practice consent banner      │
│      (kb-32 craft engine)   │      (Guidelines §6.4–6.6)        │
├─────────────────────────────┴───────────────────────────────────┤
│  (T) Instability Chronology — pharmacist rendering              │
│      (last 14d default, configurable; CAPE Addendum line 268)   │
├─────────────────────────────┬───────────────────────────────────┤
│  (V) PRN velocity signals   │  (F) Failed Intervention History  │
│      (3 classes)            │      (12-month default window)    │
├─────────────────────────────┼───────────────────────────────────┤
│  (O) Override history       │  (E) Evidence/citation drawer     │
│      (dual-vocab taxonomy)  │      (drawer; opens from A)       │
└─────────────────────────────┴───────────────────────────────────┘
```

### Panel R — Resident header

Identity, current clinical posture, and gate status. Reads from `kb32ctx.ClinicalSnapshot` via the PostgresSubstrateClient.

- **Identity row:** Display name (per facility's display preference), date of birth, primary care intensity tag (`active_treatment | rehabilitation | comfort_focused | palliative`, mapped via `internal/store/postgres/substrate_client.go` `translateCareIntensity`).
- **Substrate row:** eGFR, DBI score, ACB score, CFS score, AssessedAt timestamp. Each value is a small inline component; click opens a per-metric history sparkline drawer.
- **Recent-events row:** RecentFall72h, RecentAdmission72h, FrailtyStepIncrease30d. Boolean flags rendered as muted-when-false, emphasised-when-true.
- **Gate banner area:** when any of `RestrictivePracticeActive`, `CapacityLapse`, or `FamilyDistress` are true, a banner spans the panel with the gate name + Guidelines reference + link to the capacity assessment record.

### Panel A — Active recommendations

The reasonbeing the resident surfaced. Reads from kb-32 craft engine via `/v1/craft/draft` (when triggered by S2) or from a pre-cached recommendation set scoped to this resident.

Each recommendation row shows:
- Type tag: `STOP | MONITOR | DOSE_CHANGE | ADD` (matches `generator.Packet.Type` vocabulary)
- Urgency: `red | amber | green` (matches `urgency.Tag` output)
- Layer 1 framing body (one paragraph; from `Packet.Sections["layer_1"]`)
- Appropriateness assessment summary: 5-dimension scores `[ClinicalWarrant, EvidenceSolidity, AlternativesConsidered, RestraintConsidered, GoalsOfCareAlignment]` rendered as small score chips. Click opens the evidence drawer (E) with the scoring rationale.
- Hold reason (if any): rendered prominently when `PipelineResult.HoldReason != ""` — capacity hold, appropriateness hold, or restrictive-practice hold.
- Action row: `[Send to GP via S3] [Capture override] [View evidence] [Defer]`. The override and defer actions are always present, even when the recommendation is held — per Principle 4.

Ordering follows kb-32's `ordering.Order` rule priority (`STOP > MONITOR > DOSE_CHANGE > ADD`) with urgency as secondary sort.

### Panel C — Capacity + restrictive-practice consent banner

When relevant: capacity uncertain / unconfirmed within last 6 months, or `RestrictivePracticeActive=true`. Reads from `capacity.Gate` evaluation and the EBA register (Phase 3 commit `8d7d687b`, migration 045 EBA).

- Capacity status (last assessed, by whom, outcome). If > 6 months ago, displays "Reassessment due" with link to assessment workflow.
- Restrictive-practice consent record: practice type (chemical / physical / environmental / seclusion), consent on file (yes/no), consent expiry date.
- Per Guidelines §6.6: STOP-psychotropic recommendations on a resident with `RestrictivePracticeActive=true` AND missing/expired consent surface a *consent-required* gate badge on the relevant recommendation in panel A — the pharmacist must confirm or override before send-to-GP unlocks.

### Panel T — Instability Chronology

Per CAPE Addendum §3, the temporal narrative of multi-parameter events. Rendered via `AudienceAdaptations[AudiencePharmacist]`.

- Default window: last 14 days (CAPE Addendum line 268; configurable per facility — see Part 13 Q3).
- Event timeline: `ChronologyEvent` markers laid out chronologically, each with `PrimitiveType` (medication_change / intake_decline / fall / confusion_onset / orthostatic_instability / sedation), Severity, and a short Description.
- Recognised TemporalPattern overlays — e.g., a "volume-contraction cascade" pattern matched by the engine, with confidence score and reasoning text.
- Click on any event opens its `SubstrateRefs` set in panel E (evidence drawer).
- Until kb-33 ships the computation (Step 5 Week 20), this panel is a fallback view: chronology entries derived directly from raw substrate events (PRN administrations, lab results, fall flags) without pattern recognition. Mark fallback explicitly with a "pattern recognition pending" badge.

### Panel V — PRN velocity signals

Reads from `shared/v2_substrate/prn_velocity/compute.Compute` for each of the three Phase 1 classes.

- Three cards: benzodiazepine, antipsychotic, analgesic.
- Each card: Recent30dCount / Baseline90dAvg / VelocityRatio / Severity (1–5).
- Visual emphasis at Severity ≥ 4 (matches CAPE Guidelines line 668 threshold for elevation to the worklist).
- Click opens the underlying administration timeline in panel E.

### Panel F — Failed Intervention History

Reads from `shared/v2_substrate/failed_interventions/store.Store.ListByResident`.

- Each FIR record shows: InterventionType, AttemptDate, Outcome, RetryEligibleDate, DocumentedBy.
- Active vetoes (`RetryEligibleDate > now`) emphasised; expired vetoes muted but present.
- When a panel A recommendation would be vetoed by an active FIR (`IsVetoActive == true`), the recommendation row in panel A surfaces an inline FIR badge linking to the FIR record here.
- **Known gap (Step 4 Task B report):** `FailedInterventionRecord.ResidentID` is currently written as `uuid.Nil` because `OverrideReason` lacks a resident ID. Until the kb-32 override store is extended to resolve `RecommendationID → ResidentID` via a JOIN, FIR records cannot be retrieved by resident — they exist only by recommendation. This panel will be empty for pilot Phase 1 until the gap closes. Surface a "FIR retrieval pending" banner until then. Track as a kb-33 prerequisite.

### Panel O — Override history

Reads from `kb-32/internal/overrides/store.Store.ListByRule` (proximate) and the broader override capture history (per-resident).

- Each override row: ReasonCode (snake_case), ReasonCodeShort (3-letter Guidelines Part 5 code per Phase 2-completion Task 5), AppropriatenessFlag, Reasoning text, CapturedAt, CapturedBy.
- Filter by code, flag, or date range.
- Useful for the pharmacist to see "what have I told the engine about this resident before?" — informs override decisions on new recommendations.

### Panel E — Evidence/citation drawer

Slides out from the right on any "View evidence" click in panels A, T, V, or F.

- Recommendation citations: from `citations.Registry.ListCitations(ctx, RecommendationID)` (wired via Step 4 Task A — the `/v1/explain/:decision_id` endpoint is the canonical query).
- Per citation: SourceVersion, EffectiveFrom, EffectiveTo, ContentHash, Status (`active | amended | retracted | superseded`).
- Source amendments after fire time displayed with a "source amended after fire time" badge; the citation pin remains valid (audit-defensibility invariant from Phase 2b Task 6) but the pharmacist sees the divergence.
- Scoring rationale (when opened from a recommendation): the 5-dimension assessment with per-dimension drivers explained — sourced from `appropriateness.SubstrateBackedScorer` outputs (Phase 2-completion Task 2).
- Substrate refs (when opened from a chronology event or PRN signal): polymorphic `SubstrateReference` pointers resolved to display rows.

---

## Part 3 — Data composition

Each panel maps to specific Go types from the existing substrate. The table below is the binding contract — implementation must read from these types, not synthesise equivalent data.

| Panel | Primary read | Secondary reads |
|---|---|---|
| R Resident header | `kb32ctx.ClinicalSnapshot` (via `internal/store/postgres.PostgresSubstrateClient.SnapshotFor`) | `capacity.Assessment` for capacity status; EBA register row for consent state |
| A Active recommendations | `api.PipelineResult` (`Packet`, `ContentHash`, `LayerOutput`, `UrgencyTag`, `Assessment`, `Citations`, `HoldReason`) | `ordering.Order` for stable sort; `urgency.Tag` for red/amber/green |
| C Consent banner | EBA register row + `consent_extension.PracticeType` | capacity assessment timestamp |
| T Chronology | `instability_chronology.InstabilityChronology` (`Events`, `Patterns`, `Severity`, `AudienceAdaptations[AudiencePharmacist]`) | `SubstrateReference` resolution for click-through |
| V PRN velocity | `prn_velocity.VelocityResult` × 3 classes | `Administration` slice for the click-through timeline |
| F FIR | `failed_interventions.FailedInterventionRecord` slice | classifier output for inline-veto badging on panel A |
| O Override history | `overrides.OverrideReason` slice | `overrides.ValidReasonCodes` for filter UI |
| E Evidence drawer | `citations.RecommendationCitation` slice + `appropriateness.Assessment` | `SourceVersion` for amendment-status rendering |

Where data is not yet implemented (e.g., `FiveLayerScoring` is `TODO(kb-33)` per the ObservationLayer proto IDL), S2's fallback is the underlying substrate panels (V, F, T, O) — the pharmacist composes the scoring mentally. Mark these states explicitly so the pharmacist isn't surprised by missing aggregate views.

---

## Part 4 — Interaction model

### Flow 1: Reviewing a recommendation

1. Pharmacist arrives from S1 worklist with `resident_id` + (optionally) a primary `recommendation_id` from the triage signal.
2. S2 fetches the full snapshot (panel R) and recommendation set (panel A); chronology and signals pre-populate panels T, V, F.
3. Pharmacist clicks on the primary recommendation row.
4. Evidence drawer (E) opens with: Layer 1 / Layer 2 / Layer 3 framing, citation set, scoring rationale.
5. Pharmacist either: (a) accepts → triggers send-to-GP via S3, (b) overrides → flow 2, (c) defers → recommendation stays on the worklist with a deferral timestamp.

### Flow 2: Capturing an override

1. From panel A or the evidence drawer, pharmacist clicks "Capture override".
2. Modal opens with the dual-vocab taxonomy (per Phase 2-completion Task 5):
   - ReasonCode dropdown showing both snake_case and 3-letter codes (e.g., `goals_of_care_aligned (GCA)`)
   - AppropriatenessFlag dropdown
   - Reasoning free-text (required; min 20 characters to discourage rubber-stamping)
3. On submit: `POST /v1/craft/override/{recommendation_id}` with the override body.
4. Server-side: kb-32 override store writes the `OverrideReason`. Per Step 4 Task B, if the override's `ReasonCode` is in the reversal set (`goals_of_care_aligned | frailty_consideration | deprescribing_underway | family_consensus_pending | trial_period_active`) AND the rule ID classifier returns a known InterventionType, the override store auto-writes a `FailedInterventionRecord` with `RetryEligibleDate = now + 12 months`.
5. S2 refreshes panels O (override history) and F (FIR) to reflect the new state.

### Flow 3: Recording a failed intervention manually

The auto-write path (flow 2 step 4) is the primary mechanism. Manual entry is the fallback for:
- Failures observed outside the kb-32 recommendation loop (e.g., facility decided to reverse a deprescribing without an active recommendation)
- Historical failures discovered during initial onboarding

Flow:
1. From panel F, pharmacist clicks "Record failed intervention".
2. Modal: InterventionType (dropdown of classifier vocabulary), AttemptDate, Outcome (Outcome*Constants from `failed_interventions/types.go`), DocumentedReason (free-text), DocumentedBy (auto-filled with pharmacist identity).
3. Submit: writes `FailedInterventionRecord` via `failed_interventions.Store.Record`.
4. Panel F refreshes; panel A re-evaluates inline veto badges.

### Flow 4: Reviewing the chronology

1. Pharmacist clicks any event marker in panel T.
2. Evidence drawer (E) opens with the event's `SubstrateRefs` resolved to underlying rows (PRN administration, lab result, fall flag, etc.).
3. Pharmacist can navigate forward/backward chronologically without leaving the drawer.

---

## Part 5 — Controls and safety

### Hard stops

1. **Capacity-uncertain hold** on STOP/ADD recommendations: the recommendation surfaces in panel A but the "Send to GP" action is disabled until the capacity gate clears or the pharmacist explicitly overrides with a Guidelines-§6.4-compliant rationale. The override action is always present (Principle 4); only the "Send to GP" action is gated.

2. **Restrictive-practice consent missing** on STOP-psychotropic recommendations where `RestrictivePracticeActive=true`: same pattern — recommendation visible, override always available, "Send to GP" gated on consent confirmation or override-with-rationale.

3. **FIR active veto** on a re-attempted intervention: recommendation visible with inline FIR badge in panel A and link to the FIR record in panel F. "Send to GP" requires either (a) the FIR's `RetryEligibleDate` has passed, OR (b) explicit pharmacist override with rationale referencing the FIR record. Auto-override forbidden.

### Soft warnings

1. **Source amended after fire time**: when the citation drawer detects a citation whose underlying `SourceVersion.Status` is `amended | retracted | superseded`, the recommendation row shows a soft warning. The recommendation is still actionable (audit-defensibility invariant: the fire-time pin is canonical), but the pharmacist sees the divergence and can choose to re-run kb-32 against the current source.

2. **Stale snapshot**: `ClinicalSnapshot.AssessedAt` older than 30 days surfaces a "snapshot stale" badge in panel R. Doesn't block any action but warns the pharmacist that downstream signals may be based on outdated substrate.

3. **PRN velocity Severity 5 without chronology context**: when a PRN velocity card shows Severity 5 but the chronology has no events in the same window, surface a "context missing" badge. The signal is real but the engine couldn't compose context — pharmacist should investigate before recommending.

### Opt-out respect

The `prescriber_framing_optout` table (Phase 2-completion Task 6) governs S3 framing, not S2 rendering. However, S2 surfaces the opt-out state in panel A when the pharmacist hovers over the "Send to GP" action — the pharmacist sees that the framing will be "default" tone, not personalised. This is observation only; S2 does not provide an opt-out toggle (that's the GP's prerogative via the `POST /v1/framing/optout/{gp_id}` endpoint).

### Frame-vs-content invariance enforcement

S2 must NOT render different content based on resident demographics, facility type, or pharmacist preferences. The Phase 3 CI invariance test (`framing/invariance_test.go`, commit `266d000b`) is the contract. S2's per-pharmacist customisation is limited to:
- Panel layout density (compact / regular / spacious)
- Default chronology window length (per facility, not per resident)
- Sort preferences on panel A and O

None of these touch the clinical content of any recommendation.

---

## Part 6 — Integrations

### Upstream (S2 reads from)

- **kb-19 Protocol Orchestrator** (port 8103): conflict resolution between candidate recommendations. S2 reads the resolved set, not the raw candidates.
- **kb-20 Patient Profile** (port 8131): the canonical `ClinicalSnapshot` source. Mediated via kb-32's `PostgresSubstrateClient`.
- **kb-22 HPI Engine** (port 8132): HPI session output if available; surfaced in panel T as a context layer.
- **kb-23 Decision Cards** (port 8134): urgency tagging + MCU gate state. S2 reads the urgency tag for panel A ordering.
- **kb-32 Recommendation Craft** (the primary recommendation source): `/v1/craft/draft` for fresh evaluation, `/v1/explain/:decision_id` for audit drilldown.
- **shared/v2_substrate/** packages: failed_interventions, prn_velocity, instability_chronology, decision_metadata.

### Within-platform writes (S2 writes to)

- **kb-32 override store**: via `POST /v1/craft/override/{recommendation_id}` — captures `OverrideReason` and (via Step 4 Task B hook) triggers FIR auto-write when applicable.
- **kb-32 framing observations**: via `POST /v1/framing/observation` (not yet implemented; Phase 2 extension) — pharmacist's framing-tone observations feed back into `PerGPObserver` learning for S3.
- **EvidenceTrace** (indirect): kb-32's Stage 7 emission fires on detected→drafted; S2's "Send to GP" action triggers a new lifecycle event (`drafted→sent`) which is NOT YET wired into the EvidenceTrace emitter — flag for Phase 2 extension.

### External (Phase 2 — not Phase 1 pilot)

- **FHIR Communication resource emission** for S3 GP messages (S2 is the launch point but the integration lives in S3).
- **MyHR** upload of recommendation summaries.
- **Facility EHR** export of override decisions.

### Future (kb-33 dependency)

Once kb-33 ships the ObservationLayer gRPC server (per the proto IDL committed in Step 4 Task E), S2 may migrate from per-package reads to a single `GetResidentScoring` / `GetSignalDetections` / `GetInstabilityChronology` call. Migration is opt-in and not required for Phase 1 pilot.

---

## Part 7 — Metrics and observability

### Emitted by S2

- **View events**: `s2_resident_viewed` with resident_id (hashed for ethics monitoring), pharmacist_id, timestamp, duration. Fed into Phase 3 demographic stratification (`bias_stratification/stratifier.go`) to detect view-distribution disparities across resident cohorts.
- **Action events**: `s2_recommendation_accepted | overridden | deferred | sent` with recommendation_id, pharmacist_id, latency-from-view, timestamp.
- **Drawer-open events**: `s2_evidence_drawer_opened` — proxy for citation review behaviour. Used in audit defensibility metrics (did the pharmacist actually look at evidence before approving?).

### Inherited from kb-32

- All Stage 7 EvidenceTrace entries emitted by the pipeline are linked to the S2 view that produced them via `decision_metadata.Metadata.RecommendationID` (Step 4 Task A).
- All override captures emit ethics_log entries with `EntryType=decision`.

### Frame-vs-content invariance monitoring

The Phase 3 `bias_stratification` substrate stratifies emissions by configurable demographic dimensions. S2 must emit the same dimensions on its action events. The CI invariance gate (Phase 3 commit `266d000b`) covers frame-vs-content at the kb-32 level; S2 inherits this guarantee. A separate S2-level invariance test should assert that the same recommendation_id produces identical rendered content across all `s2_resident_viewed` events for different pharmacists.

### Performance metrics

- Panel A render time (target Part 10)
- Evidence drawer fetch latency (target Part 10)
- Override capture submit latency

---

## Part 8 — Anti-patterns

S2 MUST NOT:

1. **Render different recommendation content based on resident demographics.** Frame-vs-content invariance violation. The Phase 3 CI test is the contract.

2. **Display synthesised "risk scores" not traceable to a substrate primitive.** If a number appears, it must derive from a `shared/v2_substrate/` type or a kb-32 Assessment field. No UI-only computed metrics.

3. **Hide or modally-gate the override pathway** when a recommendation is held. Per Phase 3 `override_pathway_test.go`, override must be a primary action on the same view as the held recommendation, not behind a confirmation modal or hover-revealed control.

4. **Re-use a citation set across recommendation refreshes** when the underlying recommendation regenerated. Each pipeline run produces its own citation pin; S2 must show the pin for the recommendation currently displayed, not a cached prior version.

5. **Auto-override an FIR veto on behalf of the pharmacist.** Even if the engine could classify the override rationale automatically, the pharmacist must explicitly capture the override with reasoning text. Auto-override breaks the audit-defensibility chain.

6. **Display chronology events without their `SubstrateRefs`.** A chronology event without resolvable substrate references is not audit-defensible; suppress the event rather than render it without provenance.

7. **Surface a recommendation outside its valid `effective_from..effective_to` window** without an explicit "expired" badge. Source amendments are tracked but the pharmacist must see the temporal state.

8. **Customise panel layout per resident.** Per-pharmacist or per-facility layout customisation is allowed; per-resident customisation creates audit-defensibility holes (two pharmacists seeing different layouts for the same resident at the same time may capture different overrides).

---

## Part 9 — Risks and mitigations

| # | Risk | Probability | Impact | Mitigation | Residual |
|---|---|---|---|---|---|
| 1 | Pharmacist time pressure → evidence drawer skipping | High | Medium | Track `evidence_drawer_opened` rate per pharmacist; surface "drawer not opened" warning in audit reports. Don't gate "Send to GP" on drawer-opening (that creates click-theatre); instead audit it. | Pharmacists may game by opening the drawer without reading. Substrate-grounding (Principle 1) limits the damage — fake reviews still produce real EvidenceTrace entries that downstream audit can analyse. |
| 2 | Alert fatigue from high recommendation volume | High | Medium | Per-pharmacist override-frequency monitoring (existing kb-32 `Outcome=alert_fatigue` code). When alert_fatigue overrides exceed a threshold, S1 worklist re-prioritises. S2 surfaces the pharmacist's recent override pattern in panel O. | Fatigue can persist even with rate-limiting. Long-term mitigation is upstream — better signal-to-noise in kb-32 scoring. |
| 3 | FIR records empty due to `ResidentID = uuid.Nil` gap | Certain (Phase 1) | High for veto enforcement | Panel F displays "FIR retrieval pending" banner; Panel A inline veto badges only show for FIRs created within the current session. Resolve at kb-32 layer (extend override store to JOIN recommendations table) before kb-33 consumes. | Pilot Phase 1 operates with veto enforcement only for within-session FIRs. |
| 4 | Pharmacist disagreement with appropriateness scoring → low confidence in surface | Medium | High | Evidence drawer's scoring rationale must be readable and traceable; pharmacist feedback on individual dimension scores becomes a `uncertain_evidence` (UNE) override reason that gets fed back to `SubstrateBackedScorer` calibration. | Score calibration is an ongoing concern; not solvable at the S2 layer alone. |
| 5 | Audit failure on EvidenceTrace gaps | Low | Critical | Every recommendation displayed in panel A must have a corresponding `evidence_trace_entries` row. S2 startup self-check: queries a sample of recently-drafted recommendations and verifies trace presence; alerts on drift. Phase 3 ethics-monitoring service catches systemic gaps. | Single missing trace would be a regulatory finding; the self-check is a tripwire, not a guarantee. |

---

## Part 10 — Performance budget

| Metric | Target | Source-of-truth basis |
|---|---|---|
| Panel R + A initial render | < 1500 ms p95 | TODO(kb-33 integration) — currently propose conservatively based on kb-32 `/v1/craft/draft` latency (TODO measure) |
| Evidence drawer open | < 800 ms p95 | `/v1/explain/:decision_id` latency budget (TODO measure on staging) |
| Override submit | < 500 ms p95 | `POST /v1/craft/override/:recommendation_id` round-trip |
| Full surface load (8 panels) | < 2500 ms p95 | Composed budget; some panels parallelise |
| Chronology render (last 14d, typical resident) | < 1000 ms | Bounded by `InstabilityChronology` event count; CAPE Addendum doesn't specify |

All budgets are pre-implementation estimates and must be validated against pilot staging measurements. Mark as TODO(integration-time) until then.

---

## Part 11 — Accessibility

- **WCAG 2.1 AA** minimum across all panels.
- **Screen-reader navigation**: panels announce as landmarks; recommendation rows announce type, urgency, hold reason. Evidence drawer is a discoverable region (not a modal trap).
- **Keyboard navigation**: every action reachable via keyboard; override capture form is fully tabbable.
- **High-contrast mode**: red/amber/green urgency tags must remain distinguishable without colour (use shape + label, not colour alone).
- **Dense data displays**: panel V (PRN velocity) and panel F (FIR) tables must remain readable at 200% zoom.
- **Pharmacist context**: consultant pharmacists may use S2 on a laptop or tablet at the facility; touch-target sizes per WCAG 2.5.5 (44×44 CSS pixels minimum).

---

## Part 12 — Out of scope for this spec

- **S3 GP Communication Hub** — separate spec at `2026-05-11-s3-gp-communication-hub-spec.md`. S2 launches S3 sends but does not own the GP-facing surface.
- **S4 RACH Operational View** — facility operator surface, Phase 2 (not Phase 1 pilot).
- **S5 Standard 5 Evidence Panel** — facility auditor surface, Phase 2.
- **S6 Pharmacy Operator / Employer View** — pharmacy-chain analytics, Phase 2.
- **Patient/family direct portal access** — out of pilot scope entirely.
- **Cross-resident comparative views** — that's S1 worklist territory.
- **Billing surfaces (MBS RMMR item 903)** — see RMMR workflow spec.
- **Pharmacist-to-pharmacist handover** (covering colleague's residents) — Phase 2 extension.
- **Mobile-native S2** — responsive web is in scope; native iOS/Android apps are not.

---

## Part 13 — Open questions for clinical informatics lead

1. **Chronology window default.** CAPE Addendum line 268 uses "last 14 days" in a worked example. Is 14 days the canonical Phase 1 default, or should it be configurable per-facility? If configurable, what's the upper bound (90 days? 365 days?) before the panel becomes unusable?

2. **Per-dimension override rationale.** When a pharmacist overrides because the appropriateness scoring is wrong on a specific dimension (e.g., `EvidenceSolidity` scored 1 but pharmacist disagrees), should S2 capture per-dimension rationale, or is the existing `Reasoning` text field sufficient? Per-dimension capture feeds calibration loops but adds capture friction.

3. **Auto-default `Outcome` mapping.** Step 4 Task B mapped 5 reversal `ReasonCode` values to `Outcome*` constants. Does clinical informatics agree with the mapping (especially `frailty_consideration → reversed_due_to_frailty` and `goals_of_care_aligned → goals_of_care_aligned`)? Are there edge cases where the mapping should NOT auto-trigger FIR write?

4. **Stale snapshot threshold.** The 30-day default for the "snapshot stale" badge in panel R — is that clinically meaningful, or should it vary by signal? eGFR changes faster than CFS; one threshold may not fit both.

5. **Source amendment behaviour.** When a citation's `SourceVersion.Status` transitions to `amended` or `retracted` after fire time, should S2 (a) display the amendment banner only, (b) offer to re-run kb-32 against current source, (c) auto-flag the recommendation for pharmacist re-review? Audit-defensibility says (a); operational practicality may want (b) or (c).

6. **FIR retry-window override.** CAPE Guidelines line 643 says 12 months default for the retry-eligibility window. Should pilot Phase 1 use 12 months for all intervention types, or differentiate (e.g., antipsychotic-deprescribing 12mo, dose-reduction 6mo, statin-deprescribing 24mo)?

7. **Panel V minimum sample size.** The PRN velocity compute requires baseline data. What's the minimum baseline window before S2 should display Velocity Severity vs "insufficient data"? Currently the compute returns Severity 5 when baseline is 0 and recent > 0 — clinically that's correct for "emergent class use" but operationally noisy for new admissions.

8. **Capacity reassessment cadence.** Per Guidelines §6.4, capacity should be reassessed periodically. Is 6 months the canonical cadence? Should it vary by initial capacity outcome (e.g., resident with clear capacity reassessed annually; uncertain capacity quarterly)?

9. **Override "Reasoning" minimum length.** The flow 2 modal proposes a 20-character minimum on reasoning text to discourage rubber-stamping. Is that the right number? Should there be a maximum?

10. **Inline FIR badge behaviour on expired vetoes.** If a FIR record exists but `RetryEligibleDate` has passed (veto is expired), should panel A still surface the prior failure (educational), or remain silent (less noise)?

---

## Pre-acceptance gate for S2 implementation work

Before implementation begins, the following must be confirmed:

1. ✅ Step 4 substrate prerequisites merged to main (`f3422b23` — done 2026-05-11)
2. ⏳ Clinical informatics lead reviews Part 13 questions and provides Phase 1 defaults
3. ⏳ S3 GP Communication Hub spec authored (separate doc; S2 launches S3 sends and the spec interplay must be coherent)
4. ⏳ RMMR Workflow spec authored (RMMR triggers many S2 sessions; spec interplay must be coherent)
5. ⏳ kb-32 override store extension to resolve `RecommendationID → ResidentID` for FIR queries (the `uuid.Nil` gap from Step 4 Task B)
6. ⏳ EvidenceTrace lifecycle extension to emit `drafted→sent` events (S2 "Send to GP" action needs an audit trail)

Spec complete and saved.
