# RMMR Workflow — Specification

> **Status:** Pre-pilot design specification. Authored before any RMMR workflow implementation. Owned by clinical informatics + regulatory advisor; consumed by S2/S3 spec interplay and the eventual RMMR implementation plan.

**Scope:** Residential Medication Management Review (RMMR) workflow support on the CardioFit platform — scheduling, data inputs, session conduct, report generation, GP communication, follow-up cycle, and AHPRA / ACOP / MBS-compliant audit trails.

**Audiences:**
- **Primary:** the accredited consultant pharmacist conducting the RMMR
- **Secondary:** aged-care facility nurse / clinical lead who triggers or schedules RMMRs
- **Tertiary:** the auditor — AHPRA, Aged Care Quality and Safety Commission (ACOP), Medicare RMMR-billing auditor

**Source-of-truth references:**
- `docs/superpowers/plans/CAPE_Implementation_Guidelines_v1_1.md` (structural template)
- `docs/superpowers/plans/2026-05-11-s2-resident-workspace-spec.md` (drill-down surface during sessions)
- `docs/superpowers/plans/2026-05-11-s3-gp-communication-hub-spec.md` (post-session recommendation envelope)
- `docs/superpowers/plans/2026-05-11-cape-substrate-prerequisites.md` (FIR, PRN velocity, InstabilityChronology types that feed RMMR digests)
- Phase 3 commits `8d7d687b` + `266d000b` (capacity gate + invariance contracts)

**Regulatory disclaimer:** Section numbers cited below are best-effort references to regulatory frameworks. Where the specific section or MBS item number is uncertain or version-dependent, the citation is marked `TODO(regulatory citation)`. The regulatory advisor (when constituted per pre-pilot operational gates) must validate every citation before pilot launch.

---

## Part 0 — Operational test

> A consultant pharmacist arrives at a residential aged care facility on Tuesday morning to conduct RMMR sessions on 6 residents booked for review. Within 60 minutes per resident, the workflow must enable: (a) pre-session digest review (substrate-grounded, citation-anchored), (b) face-to-face or telehealth session conduct with structured walkthrough, (c) capacity assessment if >6 months overdue, (d) recommendation capture with override taxonomy when applicable, (e) FIR-veto handling for any re-attempted intervention, (f) RMMR report finalization with content hash, citation pin set, and pharmacist signature, (g) MBS-billable audit trail emitted. If the pharmacist cannot reach finalization with substrate-defensible artifacts inside 60 minutes per typical resident, the workflow has failed its operational test.

---

## Part 1 — Design philosophy

RMMR is the regulated cadence at which Australian aged care residents receive structured medication review. Per the Aged Care Quality Standards, every resident must have an RMMR within a defined window of admission and recurring thereafter; per MBS, the consultant pharmacist's RMMR is billable as an item (TODO: confirm 903 vs 900 vs current MBS schedule version). The platform's job is to make the substrate-defensible workflow easier than the substrate-skipping workflow.

Five principles:

1. **RMMR is regulated and billable — every artifact must be auditor-defensible.** Medicare audit posture: the RMMR report must trace every recommendation to substrate, every substrate value to a source, every source to a version pin. EvidenceTrace + citation pinning + content hash on the finalized report = the auditor's reconstruction surface.

2. **Substrate-grounded by gate, not by trust.** The workflow gates "report-finalize" on substrate review. A pharmacist cannot finalize a report without having opened the pre-session digest (substrate panels reviewed). Audit defensibility >> click-theatre concerns: gate doesn't prove cognitive review, but its absence proves negligence.

3. **Cross-session continuity is first-class.** This RMMR sees the prior RMMR's recommendations, the GP responses captured in S3, the FIR records accumulated since the prior session, the chronology since last review. A fresh RMMR is not a fresh start; it is the next node in the resident's medication-review timeline.

4. **Deprescribing is first-class.** Australian aged care prescribing literature consistently shows that the highest-impact RMMR output is a STOP recommendation (per multiple peer-reviewed evaluations of RMMR effectiveness — TODO: cite specific Australian studies). The workflow's affordances make deprescribing recommendations as easy to compose as additive recommendations; ADD recommendations have no special workflow privilege over STOP.

5. **Capacity is a pre-condition, not an afterthought.** Per Guidelines §6.4–6.6, capacity must be assessed (or reassessed if >6 months) before STOP / ADD recommendations on cognitively-impacted residents can be finalized. The workflow surfaces capacity expiry in the pre-session digest and gates "report-finalize" on resolution.

---

## Part 2 — Workflow architecture

RMMR is a stateful workflow with six stages. Each stage has explicit entry / exit conditions and produces specific artifacts.

```
┌─────────────────────────────────────────────────────────┐
│ Stage 1: TRIGGER                                        │
│  - Annual recurring (12 months since prior RMMR)        │
│  - Initial admission (within 30 days)                   │
│  - Care-intensity escalation                            │
│  - PRN velocity Severity ≥4 (Phase 1 trigger)           │
│  - Manual nurse/facility request                        │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│ Stage 2: PRE-SESSION PREPARATION                        │
│  - Digest compiled: snapshot, active recs, FIR records, │
│    PRN velocity, InstabilityChronology, prior-RMMR      │
│    artifacts, GP responses since last RMMR              │
│  - Capacity expiry flag surfaced if >6mo                │
│  - Restrictive-practice consent state surfaced          │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│ Stage 3: SESSION CONDUCT                                │
│  - Face-to-face or telehealth with resident (if able)   │
│  - Structured walkthrough: regimen review, signal       │
│    investigation, chronology context                    │
│  - Recommendation capture: STOP/MONITOR/DOSE_CHANGE/ADD │
│  - Override capture for declined kb-32 recommendations  │
│  - FIR-veto handling per Step 4 Task B                  │
│  - Capacity reassessment if gated                       │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│ Stage 4: REPORT GENERATION                              │
│  - RMMR report artifact composed                        │
│  - Content hash computed over recommendation set +      │
│    citations + override rationale                       │
│  - Pharmacist identity + accreditation embedded         │
│  - EvidenceTrace `rmmr_session_finalized` event emitted │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│ Stage 5: GP COMMUNICATION                               │
│  - Recommendations dispatched via S3 GP Communication   │
│    Hub per recommendation (not bundled — each is its    │
│    own envelope for response capture granularity)       │
│  - Pharmacist sees aggregate response state on outbound │
│    view                                                 │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│ Stage 6: FOLLOW-UP CYCLE                                │
│  - 3-month interim review (optional, surfaces in S1     │
│    worklist)                                            │
│  - 12-month re-RMMR auto-scheduled                      │
│  - Interim PRN velocity triggers may pull resident back │
│    into Stage 1 before 12 months                        │
└─────────────────────────────────────────────────────────┘
```

### Workflow state machine

`rmmr_session` table (Phase 2 migration) tracks state per session:

```
TRIGGERED → PREPARED → IN_SESSION → REPORT_DRAFT → REPORT_FINALIZED → GP_DISPATCHED → CLOSED
                                       ↓
                                   ABANDONED (e.g. resident transferred mid-session)
```

State transitions are auditable; each emits an `ethics_log` entry with `EntryType=decision`.

---

## Part 3 — Data composition

| Stage | Reads | Writes |
|---|---|---|
| 1 Trigger | `rmmr_session` (last-session timestamp) + `prn_velocity.VelocityResult` (signal) + kb-20 admission table + manual trigger inputs | `rmmr_session` row with state=TRIGGERED |
| 2 Pre-session digest | `kb32ctx.ClinicalSnapshot` + kb-32 recommendation set (`/v1/craft/draft` fresh evaluation) + `failed_interventions.Store.ListByResident` + `prn_velocity.Compute` × 3 classes + `instability_chronology.InstabilityChronology` + prior `rmmr_session` artifact + S3 response history (`s3_responses` table — Phase 2) | `rmmr_session.digest_compiled_at` |
| 3 Session conduct | digest from Stage 2 + S2 workspace (drill-downs) | `OverrideReason` rows; `FailedInterventionRecord` (auto-write per Step 4 Task B); capacity assessment rows; recommendation accept/decline state on `rmmr_session_recommendations` |
| 4 Report generation | finalized recommendation set + citation pins + override rationale + capacity assessment outcome | `rmmr_session_reports` row with content hash, pharmacist identity, AHPRA registration number, finalize timestamp; EvidenceTrace `rmmr_session_finalized` event |
| 5 GP communication | `rmmr_session_recommendations` filtered to those needing GP action | `s3_messages` rows (one per recommendation) |
| 6 Follow-up | none (scheduling-only) | `rmmr_session` (next session scheduled) |

### New tables (Phase 2 implementation, not this spec)

- `rmmr_session`: per-session state machine
- `rmmr_session_recommendations`: per-recommendation outcome within a session (accept / decline / defer / capture-only)
- `rmmr_session_reports`: finalized report artifact with content hash, pharmacist signature reference, generated PDF/JSON
- `rmmr_capacity_assessments`: capacity reassessments performed during RMMR (separate from kb-20 capacity_assessments because RMMR-context capacity reassessment has additional audit fields)

---

## Part 4 — Interaction model

### Flow 1: Annual recurring RMMR

1. Stage 1 trigger: 12 months since prior `rmmr_session.finalized_at` for resident.
2. Resident appears in pharmacist's RMMR queue (a view separate from S1 worklist — RMMR queue is scheduled cadence; S1 is signal-triggered).
3. Pharmacist clicks resident → Stage 2 digest compiles.
4. Digest review (Stage 2) — pharmacist sees:
   - Substrate panels (snapshot, recommendations, FIR, PRN velocity, chronology)
   - Prior RMMR report summary (last 12 months of accepted / declined recommendations)
   - Open S3 message states (any deferred recommendations still pending GP response)
   - Capacity-expiry flag if >6mo since last assessment
5. Pharmacist conducts Stage 3 session (face-to-face or telehealth, depending on facility) — typically 45–60 minutes for a complex resident.
6. Stage 4: pharmacist clicks "Finalize report". Workflow checks:
   - Capacity gate cleared (or override-with-rationale captured per Guidelines §6.4)
   - FIR vetoes resolved (each active veto either expired, override-captured, or recommendation skipped)
   - Pre-session digest opened (substrate review acknowledged — Principle 2 gate)
   - Restrictive-practice consent confirmed where applicable
   - All capture forms have required Reasoning fields populated
7. Content hash computed; report row written; EvidenceTrace event emitted; pharmacist identity + AHPRA registration embedded.
8. Stage 5: per-recommendation S3 dispatches (Flow 1 of S3 spec).
9. Stage 6: next RMMR auto-scheduled 12 months out.

### Flow 2: Triggered RMMR (PRN velocity escalation)

1. Stage 1 trigger: `VelocityResult.Severity >= 4` for any of the three PRN classes (CAPE Guidelines line 668 threshold).
2. Resident surfaces in RMMR queue with explicit trigger reason: "PRN benzodiazepine escalation Severity 5 in last 30 days".
3. Pre-session digest (Stage 2) prominently surfaces the triggering signal — chronology pre-scrolled to the velocity window.
4. Stage 3 session: shorter typical duration (30 minutes) — focused review around the trigger rather than full regimen.
5. Stage 4 finalize: same gates, plus a new gate — "trigger signal addressed in recommendation set OR override-with-rationale" (prevents finalizing a triggered RMMR that ignored its trigger).
6. Stage 5 + 6 as per Flow 1.

### Flow 3: Initial-admission RMMR

1. Stage 1 trigger: new resident admitted; auto-queues within 30 days per Aged Care Quality Standard (TODO: confirm standard reference + window).
2. Pre-session digest may be sparse (limited prior substrate); workflow surfaces "initial admission — limited substrate available" badge.
3. Stage 3 session emphasises baseline establishment over change recommendations.
4. Stage 4 finalize: report explicitly marked `initial_admission_rmmr=true`; downstream review cycles use this as the baseline anchor.
5. Stage 6: next RMMR scheduled 12 months out (or earlier if clinical state warrants).

### Flow 4: Capacity reassessment within RMMR

When capacity-expiry gate is active:

1. Stage 3 workflow forces a capacity-assessment sub-flow before STOP/ADD recommendations can be captured.
2. Sub-flow: pharmacist assesses capacity per facility's clinical protocol; outcome captured in `rmmr_capacity_assessments` with timestamp, outcome (`clear | uncertain | impaired`), and rationale.
3. If outcome is `uncertain` or `impaired`, restrictive-practice consent state is re-checked. STOP-psychotropic recommendations gate on consent per Guidelines §6.6.
4. Outcome propagates back to kb-20 `capacity_assessments` table so future sessions see the updated state.

### Flow 5: FIR veto handling during session

When a candidate kb-32 recommendation is veto-blocked by an active FIR:

1. Recommendation surfaces in S2 panel A with inline FIR badge (per S2 Part 4 Flow 1).
2. Pharmacist sees the prior failure context in S2 panel F.
3. Three paths:
   - **Skip**: recommendation marked `skipped_fir_veto` in `rmmr_session_recommendations`; no further action.
   - **Override-with-rationale**: pharmacist captures an override referencing the FIR record explicitly (Reasoning text must mention the FIR id or AttemptDate); recommendation proceeds to dispatch.
   - **Wait**: recommendation noted as "FIR retry-eligible-date is {timestamp}; recheck at next interim or RMMR"; not dispatched this session.
4. All three paths are auditable; auto-override is forbidden (Principle 1 + S2 anti-pattern #5).

### Flow 6: Session abandonment

Resident transferred / deceased / declines RMMR mid-session:

1. Pharmacist clicks "Abandon session" with reason code.
2. `rmmr_session.state = ABANDONED`; any captured-but-not-finalized recommendations roll back.
3. Audit-trail event emitted; MBS billing implication captured (abandoned sessions may not be billable — TODO: confirm MBS rules).

---

## Part 5 — Controls and safety

### Hard stops

1. **Capacity expiry** on STOP / ADD recommendations: report finalize gated until capacity gate cleared (Guidelines §6.4–6.6). Override-with-rationale always available per Phase 3 override-pathway invariant.

2. **FIR active veto** on re-attempted intervention: report finalize gated until veto resolved per Flow 5 (one of three paths). Auto-override forbidden.

3. **Restrictive-practice consent missing** on STOP-psychotropic for `RestrictivePracticeActive=true` residents: report finalize gated until consent confirmed or override-with-rationale captured (Guidelines §6.6).

4. **Frame-vs-content invariance violation**: the RMMR report's recommendation content must match the kb-32 `content_hash` for each included recommendation. Pre-finalize check; mismatch fails finalize.

5. **MBS audit-trail completeness**: report finalize requires every captured recommendation to have:
   - A linked kb-32 `recommendation_id` (or explicit "captured-without-engine" annotation for manual additions)
   - A citation pin set (empty set permitted only for manual additions)
   - A capture-time pharmacist identity matching the session's `pharmacist_id`
   - A finalize-time content hash
   Missing any of these fails finalize.

### Soft warnings

1. **Pre-session digest not opened**: warns pharmacist that gate enforcement requires digest review. Doesn't block — but warning surfaces in audit report.

2. **Recommendation captured without override taxonomy code** (free-text-only): warns pharmacist that structured capture is the audit-preferred path. Submission allowed (free-text not always reducible to a taxonomy code) but flagged in report.

3. **Session duration outlier**: if a session is finalized in <10 minutes or >120 minutes for a typical resident, flag for senior pharmacist review. Not a fraud signal alone but a quality-of-review proxy.

4. **GP response pending from prior RMMR**: digest surfaces unresolved S3 messages from the prior RMMR; pharmacist should address these (re-send, abandon, escalate) before composing new recommendations on the same resident.

### Soft warning escalation

If the same pharmacist repeatedly produces sessions with:
- >X% override rate without Reasoning meeting clinical-substantive criteria
- >Y% sessions finalized without digest review
- >Z% sessions abandoned

ethics-monitoring service flags for senior pharmacist + AHPRA audit review (per Phase 3 ethics-monitoring orchestrator).

---

## Part 6 — Integrations

### Upstream (RMMR reads from)

- **kb-19 Protocol Orchestrator** (port 8103): protocol conflict resolution for candidate recommendations.
- **kb-20 Patient Profile** (port 8131): `ClinicalSnapshot`, capacity_assessments, admission dates.
- **kb-22 HPI Engine** (port 8132): HPI history feeds chronology context.
- **kb-23 Decision Cards** (port 8134): urgency tagging; MCU gate state.
- **kb-32 Recommendation Craft**: fresh recommendation evaluation per session.
- **shared/v2_substrate/**: failed_interventions, prn_velocity, instability_chronology.

### Within-platform writes

- **rmmr_session table** (new): session state machine.
- **rmmr_session_recommendations** (new): per-recommendation outcome.
- **rmmr_session_reports** (new): finalized report.
- **rmmr_capacity_assessments** (new): RMMR-context capacity reassessments.
- **kb-32 override store**: `OverrideReason` writes on decline path (same as S3).
- **failed_interventions** store: auto-write per Step 4 Task B classifier hook.
- **kb-20 capacity_assessments**: propagation from RMMR-context reassessment.
- **EvidenceTrace**: `rmmr_session_finalized` event (TODO(kb-32-extension) — Stage 7 emitter must extend).
- **ethics_log**: state-machine transitions emit `decision`-type entries.

### Cross-surface integration

- **S2 Resident Workspace**: launched during Stage 3 for per-resident drill-down. Same surface, RMMR-session context layered on top.
- **S3 GP Communication Hub**: Stage 5 dispatches use S3 envelopes. Each recommendation = one S3 message (not bundled).
- **S5 Standard 5 Evidence Panel** (Phase 2): exposes RMMR completion to facility auditors. RMMR finalize timestamp + report content hash surfaced.
- **Pharmacy Self-Visibility** dashboards (Phase 1 codebase already exists per session-start status): pharmacist-side analytics including per-session metrics.

### External (Phase 1 pilot)

- **MBS billing system** (out-of-band for pilot Phase 1): pharmacist exports billing data manually; audit-trail-defensible substrate is the export artifact. Direct MBS integration is Phase 2.
- **Facility EHR**: report export (PDF + JSON) shipped to facility EHR by pharmacist post-session. Direct EHR integration via FHIR is Phase 2.
- **MyHR** (My Health Record): upload of RMMR summary is Phase 2.

### Future (kb-33)

Once kb-33 ObservationLayer gRPC ships (Step 5), RMMR digests may migrate from per-package reads to `GetResidentScoring` + `GetSignalDetections` + `GetInstabilityChronology` + `GetFailedInterventionHistory` calls. Migration opt-in; not required for pilot.

---

## Part 7 — Metrics and observability

### Operational metrics

- **RMMR completion rate per facility**: (sessions finalized / sessions scheduled) per month. Cohort-stratified per Phase 3 demographic stratification.
- **Time-per-resident in session**: distribution (p50, p95, p99). Outliers flagged per Part 5 soft warning #3.
- **Recommendation-acceptance rate post-session**: % of S3-dispatched recommendations with GP acceptance within deadline. Cross-RMMR comparison surfaces practice patterns.
- **Capacity-gate-clearance time**: median time from gate-active to gate-cleared (proxy for workflow friction).
- **FIR-veto-override frequency**: % of FIR-vetoed recommendations that the pharmacist overrides with rationale. Over-overriding (>X% — TODO threshold) flags for senior review.

### Audit-defensibility metrics

- **MBS audit-readiness rate**: % of finalized reports with all five Stage 4 audit-trail completeness gates passed. Target: 100%.
- **Citation-pin coverage**: % of report recommendations with non-empty citation pin sets. Target: 100% for engine-originated; ≥X% for manual additions (TODO threshold).
- **EvidenceTrace `rmmr_session_finalized` event emission rate**: % of finalized reports with the lifecycle event. Target: 100%. Gap = audit-trail breach.

### Phase 3 ethics monitoring

`DetectBiasDisparity` runs against:
- RMMR scheduling frequency stratified by resident demographic
- Recommendation generation rate stratified by resident demographic
- Capacity-assessment outcome distribution stratified by resident demographic
- Session duration stratified by resident demographic

Disparities above threshold trigger ethics-monitoring alerts.

### Frame-vs-content invariance monitoring

RMMR report content hash MUST equal kb-32 recommendation content hash for each included recommendation. Periodic invariance check across `rmmr_session_reports`: alert on mismatch.

---

## Part 8 — Anti-patterns

RMMR workflow MUST NOT:

1. **Allow report finalize without capacity gate clearance** for STOP/ADD on cognitively-impacted residents. Guidelines §6.4 violation; AHPRA defensibility risk.

2. **Auto-override FIR vetoes without rationale capture.** Audit-defensibility violation; deprescribing-failure pattern goes invisible to downstream sessions.

3. **Generate the RMMR report before the session is conducted.** Substrate-grounding violation — recommendations must trace to session-time decisions, not pre-session digest extrapolation.

4. **Re-use a prior RMMR report verbatim for an annual recurring.** Cross-session-continuity violation — resident's state changed; each report is a snapshot, not a template.

5. **Recommend a STOP based solely on a single PRN velocity reading without InstabilityChronology context.** Substrate-grounding violation — velocity is a signal, not a sufficient cause.

6. **Bill MBS item 903 (or current equivalent) without supplying the auditable substrate trail.** Medicare audit fraud risk. The workflow must prevent finalize-without-trail; billing without finalize is operationally impossible.

7. **Allow free-text-only recommendation capture for STOPs of psychotropics on RestrictivePracticeActive residents.** Guidelines §6.6 violation; structured capture with consent-state field is required.

8. **Auto-finalize an in-progress session on timeout.** Sessions are explicitly finalized or explicitly abandoned. Timeout-finalize creates ghost reports.

9. **Surface pharmacist's prior decline / override patterns to other pharmacists during their sessions.** Pattern leakage breaks peer learning context; only senior pharmacist + audit views see cross-pharmacist patterns.

10. **Generate the report content hash before pharmacist signature.** Hash must include the pharmacist identity assertion; if signature isn't captured, hash is incomplete.

---

## Part 9 — Risks and mitigations

| # | Risk | Probability | Impact | Mitigation | Residual |
|---|---|---|---|---|---|
| 1 | Pharmacist time pressure leading to substrate-skipping | High | High | Pre-session digest gate (Principle 2): finalize requires digest opened. Audit logs record gate enforcement. Surface session duration outliers per Part 5. | Click-theatre risk: pharmacist opens digest without reading. EvidenceTrace catches absence of evidence drawer opens; can't catch shallow reading. |
| 2 | Capacity-assessment latency blocking the session | Medium | Medium | Pre-session digest flags upcoming capacity expiry (proactive); Flow 4 sub-flow streamlines reassessment. | Some facilities may not have capacity-assessment-trained staff on RMMR day; workflow blocks finalize — operationally painful but defensible. |
| 3 | Facility resistance to deprescribing recommendations | Medium | High | Cite ACOP Standard 5 explicitly in S3 GP messages; surface FIR + InstabilityChronology in report rationale; Pharmacy Self-Visibility dashboard surfaces facility-level deprescribing rates for senior review. | Facility-level cultural change required; not S3- or RMMR-solvable alone. |
| 4 | Audit failure on MBS billing | Low | Critical | Every billed RMMR has Stage 4 audit-trail completeness gates passed; quarterly internal audit via Phase 3 ethics-monitoring service; report content hash externally verifiable. | Auditor disagreement with substrate interpretation possible; defensibility is necessary, not sufficient. |
| 5 | Cross-tool inconsistency (RMMR report vs facility EHR vs printed handout) | Medium | High | Content hash on report artifact; export embeds hash; downstream consumers verify. | Operational — every downstream consumer must honour the verify contract. |
| 6 | FIR-veto enforcement gap (uuid.Nil ResidentID issue from Step 4 Task B) | Certain (Phase 1) | High for cross-session FIR | Pilot Phase 1 operates with within-session FIR enforcement only. Cross-session FIR retrieval requires kb-32 override store extension (resolve RecommendationID → ResidentID); track as Phase 2 must-fix. | Until extension lands, multi-session deprescribing-failure patterns may go undetected. Surface in pilot risk register. |
| 7 | EvidenceTrace lifecycle gap on `rmmr_session_finalized` event | Certain (Phase 1) | High for audit defensibility | TODO(kb-32-extension); until lands, `rmmr_session_reports` row + ethics_log entry serve as proximate audit substrate. | Pilot may operate without full Stage 7 lifecycle coverage on RMMR events; document the gap. |
| 8 | Initial-admission RMMR window slippage | Medium | Medium | Stage 1 auto-trigger on admission; pharmacist queue surfaces approaching window expirations. | Pilot operations must staff to meet 30-day window; staffing risk, not platform risk. |

---

## Part 10 — Performance budget

| Metric | Target | Basis |
|---|---|---|
| Pre-session digest compile (1 resident) | < 3000 ms p95 | Composed: snapshot + recs + FIR + velocity × 3 + chronology + prior session |
| Pre-session digest compile (8 residents batch) | < 10000 ms p95 | Per-resident operations parallelisable; budget reflects degradation |
| Report finalize latency | < 5000 ms p95 | Content hash + write + EvidenceTrace emit + S3 dispatches queued |
| Capacity reassessment sub-flow submit | < 2000 ms p95 | Write `rmmr_capacity_assessments` + propagate to kb-20 |
| Session-to-S3-dispatch latency | < 30 seconds (target — TODO confirm UX) | Stage 5 enqueues S3 messages |
| Report export (PDF) | < 8000 ms p95 | Per-resident; large chronology may elongate |

All TODO(integration-time validation). Pre-pilot estimates.

---

## Part 11 — Accessibility

- **WCAG 2.1 AA** minimum across all RMMR surfaces.
- **Telehealth session context**: when conducted via telehealth, RMMR surface must operate alongside a video call without crowding the screen. Split-screen design supported.
- **Multi-resident batch view**: pharmacist may have 8 residents queued; bulk navigation must be keyboard-accessible.
- **Report PDF accessibility**: exported PDFs must meet PDF/UA accessibility standard (alt text on charts; reading order preserved).
- **Mobile usage**: pharmacist may use tablet at facility; touch targets per WCAG 2.5.5.

---

## Part 12 — Out of scope for this spec

- **S2 Resident Workspace** — separate spec; RMMR Stage 3 uses S2 surface but doesn't own it.
- **S3 GP Communication Hub** — separate spec; RMMR Stage 5 uses S3 surface but doesn't own it.
- **Facility-operator dashboards (S4)** — Phase 2.
- **Pharmacy-employer / aggregate dashboards (S6)** — Phase 2.
- **State-government reporting** — out of pilot scope.
- **Direct patient/family portal access** — out of pilot scope.
- **MBS billing automation (direct claim submission)** — Phase 2 or Phase 3.
- **MyHR upload of RMMR summary** — Phase 2.
- **Direct FHIR R4 export to facility EHR** — Phase 2.
- **HMR (Home Medicines Review) workflow** — RMMR is the residential variant; HMR is the community-pharmacy variant. Not covered here.
- **Case conference (MBS item 900 variant if distinct from 903)** — RMMR is the primary review; case conference is a multi-party variant. Not covered here unless confirmed in scope per Part 13.

---

## Part 13 — Open questions for clinical informatics + regulatory advisor

**Regulatory citations to confirm (highest priority — load-bearing for pilot):**

1. **MBS item number for the RMMR.** Current spec assumes item 903; advisor must confirm against current MBS schedule version. Also confirm whether item 900 (case conference) is in pilot scope.

2. **Aged Care Quality Standard reference for the initial-admission window.** Spec assumes 30 days; advisor must confirm the standard reference and the actual window.

3. **Pharmacy Board of Australia accreditation citation.** Where in the AHPRA documentation is the consultant pharmacist's accreditation requirement specified? Report header must cite this; spec marks `TODO(regulatory citation)`.

4. **Quality Standards 5 + 8 wording.** Anti-pattern #1 cites ACOP Standard 5; report rationale referencing it must use the canonical wording. Confirm.

5. **Australian deprescribing literature citations.** Principle 4 references "multiple peer-reviewed evaluations of RMMR effectiveness". Authoritative citations needed for pre-pilot regulatory dossier.

**Design decisions (medium priority):**

6. **Capacity reassessment cadence.** Spec assumes 6 months. Should it vary by initial capacity outcome (clear: annual; uncertain: quarterly; impaired: monthly)?

7. **FIR retry window per intervention type.** CAPE Guidelines line 643 defaults 12 months. Should this differ for antipsychotic-deprescribing (recovery patterns) vs dose-reduction (faster failure modes)?

8. **Session duration outlier thresholds.** Spec proposes <10min OR >120min as outliers. Confirm bands per facility complexity (8-bed boutique vs 180-bed metropolitan facility may have very different distributions).

9. **Override rate threshold for senior-pharmacist flag.** Part 5 soft warning escalation cites X%. Define X.

10. **Report PDF embedding decisions.** Should the full InstabilityChronology be embedded, or just a 14-day window per CAPE Addendum line 268? Embedded as image or as structured data?

11. **AHPRA registration number display.** Header or footer? Watermark? Embedded in the content hash or attested-by separately?

12. **Pharmacist signature mechanism.** Digital signature (Ed25519 — kb-services already use Ed25519 per CLAUDE.md governance pattern) or attestation-by-login? If digital signature, where are private keys managed?

13. **Triggered RMMR collision with annual cadence.** If a PRN velocity Severity 5 triggers an RMMR 11 months after the prior, does the annual cadence reset to "this session + 12mo" or stay on the original cycle?

14. **Abandoned session billability.** MBS rules for abandoned sessions — billable, partial, not billable? Affects Stage 6 Flow 6 economics.

15. **Report retention period.** AHPRA retention requirements for medication-review documentation. Confirm and propagate to `rmmr_session_reports` retention policy.

16. **Cross-session continuity scope.** Should this RMMR have read access to ALL prior RMMRs for this resident, or only the most recent? Privacy vs continuity trade-off.

17. **Telehealth RMMR validity.** MBS rules for telehealth-conducted RMMRs — billable on parity with face-to-face, or different items / different rates?

---

## Pre-acceptance gate for RMMR implementation work

Before implementation begins:

1. ✅ Step 4 substrate prerequisites merged to main (`f3422b23` — done 2026-05-11)
2. ⏳ Regulatory advisor constituted + confirms Part 13 questions 1–5 (regulatory citations)
3. ⏳ Clinical informatics lead resolves Part 13 questions 6–17 (design decisions)
4. ⏳ S2 + S3 specs authored (done — sibling files in same plans directory)
5. ⏳ kb-32 override store extension: resolve `RecommendationID → ResidentID` for cross-session FIR enforcement (Risk #6)
6. ⏳ EvidenceTrace Stage 7 extension: `rmmr_session_finalized` event (Risk #7)
7. ⏳ Pilot operations staffing model includes capacity reassessment readiness (Risk #2)
8. ⏳ Pharmacist Advisory Group constituted (per pre-pilot operational gate from roadmap)

Spec complete and saved.
