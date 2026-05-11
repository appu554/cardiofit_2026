# ⚠️ SUPERSEDED — DO NOT USE

> **This document is superseded and was authored in error.** It was written by a parallel agent (claude-opus-4-7, session 2026-05-11) that did not read the canonical architecture stack (v3.0 Product Proposal, Ethical Architecture Implementation Guidelines, Pharmacist Self-Visibility Implementation Guidelines, Recommendation Craft Engine Implementation Guidelines, S2 v1.0 + Adaptive Cognition Addendum, KB-29 Templates, Decision Packet Rendering Guidelines, Style Guide, Substrate Query Feasibility Analysis).
>
> **Why this document is wrong:** It authors a 4-view "envelope" architecture for the GP communication surface without inheriting the substrate-primitive-inheritance discipline committed in S2 Adaptive Cognition Addendum Part 4. The canonical discipline is that the shared primitives (recommendation lifecycle, restraint pairing, drill-through, audit, goals-of-care, pharmacist actions) are specified once across the platform and inherited by each surface that consumes them. Each surface specialises for its audience but does not redefine the primitives. This document specialises without inheriting — it invents framing-tone fields, transport candidates, response capture vocabulary, and ethics monitoring hooks rather than referencing how the canonical stack already specifies those primitives.
>
> **A canonical S3 GP Communication Hub Implementation Guidelines v1.0 is the proper authoring path.** It will inherit the substrate-primitive-inheritance discipline from S2 Addendum Part 4 and specialise for the GP audience (asynchronous decision capture, per-GP framing learning, mobile-primary form factor) — not reinvent the primitives. This document does not.
>
> **Do not implement against this document. Do not cite this document. Do not extend this document.**
>
> ---
>
> _Original content preserved below for audit trail only._

---

# S3 GP Communication Hub — Specification (SUPERSEDED)

> **Status:** SUPERSEDED. Authored without reading canonical stack. Do not use.

**Audience:** The general practitioner (GP) who prescribed the medication and is responsible for accepting / declining / deferring the pharmacist's recommendation. **Secondary:** the pilot pharmacist sending the message (sees the other side of the same envelope). **Tertiary:** the audit trail (every send/receive/response recorded for EvidenceTrace + AHPRA defensibility).

**Source-of-truth references:**
- `docs/superpowers/plans/CAPE_Implementation_Guidelines_v1_1.md` (structural template)
- `docs/superpowers/plans/2026-05-09-phase-2-completion.md` (kb-32 craft engine, framing substrate, opt-out endpoint)
- `kb-32/internal/framing/per_gp_observer.go` and `optout_store.go` (the core machinery S3 envelopes)
- `kb-32/internal/citations/` (fire-time pinning that S3 surfaces to GPs)
- `docs/superpowers/plans/2026-05-11-s2-resident-workspace-spec.md` (S3's launch point)
- Phase 3 commit `266d000b` — `framing/invariance_test.go` and `override_pathway_test.go` (the invariance contracts S3 must preserve)

---

## Part 0 — Operational test

> A GP receives a STOP-psychotropic recommendation from a consultant pharmacist via S3 while doing afternoon rounds. Within 90 seconds, S3 must give them: (a) the resident in context (one paragraph of clinical framing), (b) the recommendation and the substrate that justifies it (citation drawer), (c) a structured response capture with accept / decline-with-reason / defer-with-timeframe — without requiring login to a separate system or printing a fax. If the GP cannot answer "do I act on this, and if not why" inside 90 seconds, S3 has failed its operational test.

---

## Part 1 — Design philosophy

S3 is the cross-actor envelope between the consultant pharmacist (who reviewed) and the GP (who prescribes). Without S3, recommendations stay trapped in the pharmacist's worklist; with S3, they reach the prescribing decision-maker with provenance intact.

Five principles:

1. **Substrate-grounded framing.** The message's tone (clinical / collegial / brief / detailed) is data-driven from `PerGPObserver.Suggest`, not heuristic. Below the 30-observation floor (CAPE pre-pilot threshold), tone defaults to `"default"`. Opted-out GPs always get `"default"` regardless of observation count.

2. **Opt-out is irrevocable in spirit.** The `prescriber_framing_optout` table (Phase 2-completion Task 6) allows a GP to opt out of framing learning. Per Phase 2 Task 6, re-registering after a revoke reactivates opt-out idempotently. Re-registering does NOT reset prior observations to zero (per CAPE pre-pilot ethics commitment) — the engine retains them but does not consume them while opt-out is active. If the GP revokes opt-out and then re-registers, learning resumes from the existing observation set, not from zero.

3. **Evidence pinned at fire time is presented at decision time.** Every recommendation transmitted carries its `RecommendationCitation` set, pinned via `citations.PinAtFireTime`. Source amendments after fire time do not retroactively change what the GP saw at decision time — the audit-defensibility invariant from Phase 2b Task 6.

4. **GP response is structured, not free-text-only.** The response capture maps to the kb-32 dual-vocab override taxonomy (Phase 2-completion Task 5). A free-text "additional context" field is available, but the primary capture is a code (`PPF` / `CJG` / `GCA` / etc.). Structured capture feeds calibration loops; unstructured-only response is an anti-pattern.

5. **Frame-vs-content invariance.** Per `framing/invariance_test.go` (Phase 3 commit `266d000b`), the clinical content is invariant across framing tones. S3's tone variation MUST NOT alter recommendation Layer 1 body. Layer 2 (clinical context) and Layer 3 (evidence) are pure substrate — also invariant. Only the *envelope* (subject line, greeting, paragraph ordering, closing) varies by tone.

---

## Part 2 — Surface architecture

S3 has four primary views:

```
┌──────────────────────────────────────────────────────────────┐
│  (I) Inbound list — GP's queue of pending recommendations    │
├──────────────────────────────────────────────────────────────┤
│  (M) Single-message view — recommendation + response capture │
├──────────────────────────────────────────────────────────────┤
│  (B) Outbound list — pharmacist's view of sent recommendations│
├──────────────────────────────────────────────────────────────┤
│  (H) History view — accept/decline/defer record over time    │
└──────────────────────────────────────────────────────────────┘
```

### View I — Inbound list (GP-facing)

Queue of pending recommendations addressed to this GP.

- Rows ordered by urgency (`red > amber > green` per kb-32 `urgency.Tag`) then send timestamp.
- Each row: resident display name, recommendation type tag (STOP / MONITOR / DOSE_CHANGE / ADD), one-line Layer 1 framing excerpt, send timestamp, pharmacist name, response state (`pending | viewed | responded`).
- Filters: by resident, by recommendation type, by urgency, by response state.
- Bulk-action: mark all viewed (does not capture responses; just acknowledges queue).

### View M — Single-message view (GP-facing)

The decision surface. One recommendation at a time.

Composed sections:
- **Envelope header**: pharmacist name + accreditation, send timestamp, response deadline (if applicable per deferral policy — see Part 13).
- **Resident context** (Layer 2 framing from `Packet.Sections["layer_2"]`): clinical summary, current regimen, recent events. Audience-adapted for GP — emphasises prescribing-relevant fields.
- **Recommendation body** (Layer 1 framing from `Packet.Sections["layer_1"]`): the actual proposed change in one paragraph. Frame-invariant across tones.
- **Substrate rationale** (Layer 3 framing from `Packet.Sections["layer_3"]`): why the recommendation fired — references to ClinicalSnapshot, PRN velocity signal, InstabilityChronology event chain, FailedInterventionRecord vetoes (if any). Citation references inline.
- **Citation drawer**: opens from any inline citation reference. Shows `RecommendationCitation` set with SourceVersion, ContentHash, Status. Source amendments after fire time displayed as a soft warning badge.
- **Response capture form**:
  - `Accept` → primary CTA
  - `Decline` → opens dropdown with override taxonomy codes; required Reasoning text (min 10 chars)
  - `Defer` → opens timeframe picker (24h / 72h / 7d / 14d / custom); optional Reasoning
  - Override-pathway-availability invariant: all three response options are present on every recommendation, regardless of urgency or hold state (Phase 3 commit `266d000b`)
- **Additional context** field: free-text, optional, captured but does not replace structured response code.

### View B — Outbound list (pharmacist-facing)

Pharmacist sees what they've sent and where each recommendation stands.

- Rows ordered by send timestamp descending, with filter to "pending response" / "responded" / "expired (no response within deadline)".
- Each row: resident, recommendation type, GP recipient, send timestamp, current response state, framing tone used.
- Click opens the same view M with the response state visible.

### View H — History view (both audiences)

GP-side: rolling history of all recommendations received from any pharmacist, with response captured per row. Filter by pharmacist, date range, response code.

Pharmacist-side: rolling history of all recommendations sent, with response received per row. Filter by GP, date range, response code, framing tone.

History view supports practice-pattern analysis (per Part 7 metrics) without exposing raw demographic data.

---

## Part 3 — Data composition

| View / section | Primary read | Secondary reads |
|---|---|---|
| I inbound list | `s3_messages` (new — Phase 2 implementation table; see Part 6 integrations) filtered by `gp_id` | `urgency.Tag` for ordering |
| M envelope header | `s3_messages.pharmacist_id` + `s3_messages.send_at` | pharmacist identity from kb-30 PDP store |
| M Layer 1/2/3 framing | `generator.Packet.Sections["layer_1" \| "layer_2" \| "layer_3"]` | `framing.PerGPObserver.Suggest(ctx, gpID)` for tone envelope (NOT for content) |
| M citation drawer | `citations.Registry.ListCitations(ctx, RecommendationID)` | `SourceVersion` for amendment-status |
| M response capture | writes `OverrideReason` to kb-32 override store on decline; writes `s3_responses` row | dual-vocab `overrides.ToShortCode` / `ToReasonCode` for code rendering |
| B outbound list | `s3_messages` filtered by `pharmacist_id` | `s3_responses` JOIN for response state |
| H history (GP) | `s3_messages` + `s3_responses` JOIN, filtered by `gp_id` | none |
| H history (pharmacist) | `s3_messages` + `s3_responses` JOIN, filtered by `pharmacist_id` | none |

### New tables needed (Phase 2 implementation, NOT this spec)

- `s3_messages`: `id UUID PK, recommendation_id UUID, pharmacist_id UUID, gp_id UUID, transport TEXT, send_at TIMESTAMPTZ, framing_tone TEXT, content_hash TEXT, response_deadline TIMESTAMPTZ NULLABLE`
- `s3_responses`: `id UUID PK, message_id UUID FK, gp_id UUID, response_code TEXT, response_code_short TEXT, reasoning TEXT, defer_until TIMESTAMPTZ NULLABLE, captured_at TIMESTAMPTZ`

Both tables become migration candidates when S3 implementation lands; out of scope for this spec.

---

## Part 4 — Interaction model

### Flow 1: Pharmacist sends a recommendation

1. From S2 panel A, pharmacist selects a recommendation and clicks "Send to GP via S3".
2. S2 must have cleared capacity / restrictive-practice / FIR-veto holds first (per S2 Part 5). If a hold is active, "Send to GP" is disabled with the gate reason surfaced; override-with-rationale is the only path.
3. S3 composes the envelope:
   - GP resolved from the resident's primary prescriber (kb-20 patient profile)
   - Framing tone via `PerGPObserver.Suggest(ctx, gpID)` — returns the GP's learned tone OR `"default"` if opted out OR below 30-observation floor
   - Citations frozen at fire time via the pin set on the recommendation
   - Content hash computed across Layer 1 + Layer 2 + Layer 3 + citations (deterministic; matches the kb-32 Stage 5 hash for audit-trail coherence)
4. Transport: per Phase 2 selection (see Part 6). For Phase 1 pilot, default transport is **in-platform secure message** (S3 GP-facing surface accessed via the pilot portal). FHIR `Communication` resource emission and email/fax fallbacks are TODO(integration partner selection).
5. `s3_messages` row written with `send_at = now()`. EvidenceTrace receives a `drafted→sent` lifecycle event (TODO(kb-32-extension) — Stage 7 currently emits only on detected→drafted).
6. Pharmacist sees the message in view B (outbound) with state `pending`.

### Flow 2: GP receives + responds

1. GP logs into the pilot portal (or follows a deep link from notification email; see Part 6 transport options).
2. View I (inbound) shows queued recommendations.
3. GP clicks a row; view M opens.
4. GP reviews Layer 1 / Layer 2 / Layer 3 + citation drawer.
5. GP selects one of: Accept / Decline / Defer.
   - **Accept**: response captured with `response_code = "accepted"` (a synthetic code, NOT one of the kb-32 override codes — "accepted" is a positive outcome). `s3_responses` row written. The recommendation in kb-32 transitions to `drafted→accepted` (Stage 8 — TODO(kb-32-extension)). Pharmacist sees `responded:accepted` in view B.
   - **Decline**: response capture form opens with override taxonomy dropdown (dual-vocab snake_case + 3-letter codes). GP picks a code (e.g., `clinical_judgment / CJG`), enters Reasoning text (min 10 chars). On submit, `s3_responses` row written; kb-32 `OverrideReason` written via `POST /v1/craft/override/:recommendation_id` (with `CapturedBy` set to GP identity, not pharmacist — distinction matters for audit). The kb-32 override store's FIR auto-write hook (Step 4 Task B) fires when applicable.
   - **Defer**: response capture form opens with timeframe picker. On submit, `s3_responses` row written with `defer_until` set. Recommendation re-surfaces in view I after the deferral expires (auto-requeued). Pharmacist sees the deferral in view B.
6. Whichever path: response is irrevocable from the GP's side — to change a response, GP must contact the pharmacist out-of-band. (Rationale: response captures decision provenance; permitting silent edits breaks audit defensibility.)

### Flow 3: GP opts out of framing learning

1. From any view (M or H), GP clicks "Use default tone for all my recommendations".
2. Confirmation dialog explains: opt-out is per-GP-account; takes effect immediately; can be revoked later; prior observations are retained but not consumed during opt-out (per Principle 2).
3. `POST /v1/framing/optout/:gp_id` (Phase 2-completion Task 6 endpoint).
4. All future S3 envelopes addressed to this GP use `framing_tone = "default"` regardless of `PerGPObserver.Suggest` output.
5. To revoke: `DELETE /v1/framing/optout/:gp_id`. Learning resumes from the retained observation set (does not zero out).

### Flow 4: No-response escalation

1. After response deadline expires without GP response, recommendation surfaces in pharmacist's view B as `expired (no response)`.
2. Pharmacist can: (a) re-send via S3 (creates a new `s3_messages` row referencing the same `recommendation_id`), (b) escalate to senior pharmacist / RMMR re-queue, (c) write a documentation override (`Outcome=workflow_constraint` / `WFC`) and close out.
3. No-response does NOT auto-decline. The recommendation remains in `drafted` state with a "GP did not respond by deadline" annotation. Audit-defensibility: missing responses are visible, not hidden.

### Flow 5: GP-side history practice-pattern review

1. From view H, GP filters their accept-rate by pharmacist or recommendation type.
2. Aggregate statistics rendered as substrate-grounded counts (not synthesised metrics).
3. No PHI leakage: GP sees their own response patterns only; cannot see other GPs' patterns through this view.

---

## Part 5 — Controls and safety

### Frame-vs-content invariance (HARD STOP)

The Phase 3 CI test (`framing/invariance_test.go`, commit `266d000b`) asserts content equivalence across paraphrased framings. S3 must preserve this:

- `framing_tone` may vary the envelope (greeting, ordering, closing) but NOT the Layer 1 / 2 / 3 content
- A pre-send validation must compare `content_hash(layer_1 + layer_2 + layer_3 + citations)` against the kb-32 recommendation's original content hash; mismatch fails the send
- Phase 3 ethics-monitoring service should run a periodic invariance check across `s3_messages` table: same `recommendation_id` sent to different GPs must have identical `content_hash` (only `framing_tone` differs)

### Capacity + restrictive-practice gate (HARD STOP)

Recommendations triggered by Stage 3.5 capacity hold or §6.6 restrictive-practice consent gate MUST NOT be sendable via S3 until the hold is cleared by the pharmacist in S2. S3 surface must:
- Refuse to compose an envelope if the upstream recommendation has `HoldReason != ""`
- On send-attempt for held recommendation, return error to pharmacist with the hold reason surfaced

### Override pathway availability (HARD STOP)

Per Phase 3 `override_pathway_test.go`, every blocked recommendation must surface a clinician override route. S3 honours this on the GP side:
- The Decline action is always present on view M, regardless of urgency / hold state / framing tone
- Decline is NOT behind a confirmation modal, dialog tree, or hover-revealed control
- Decline opens the same form for STOP / MONITOR / DOSE_CHANGE / ADD recommendations

### Opt-out enforcement (HARD STOP)

Before any S3 envelope is composed:
1. Call `optout_store.IsOptedOut(ctx, gpID)`.
2. If `true`: force `framing_tone = "default"`, log the opt-out enforcement event for ethics monitoring.
3. If `false`: call `PerGPObserver.Suggest` for the tone.

Violation (sending non-default tone to opted-out GP) is a P0 ethics incident.

### Demographic stratification (SOFT WARNING)

Phase 3's `bias_stratification` substrate stratifies emissions by demographic dimensions. S3 must emit per-message stratification fields (resident demographic class, GP demographic class) on `s3_messages` insert. Ethics monitoring runs `DetectBiasDisparity` against:
- Recommendation send rate stratified by resident demographic
- GP decline rate stratified by resident demographic
- Framing tone distribution stratified by GP demographic

Disparities above the Phase 3 threshold trigger ethics-monitoring alerts.

### Soft warnings

1. **Source amended after fire time**: citation drawer surfaces the amendment status; recommendation remains valid (pin invariant).
2. **GP response received but pharmacist already re-sent**: race condition — surface to pharmacist that response arrived for prior version; pharmacist decides whether to honour or supersede.
3. **Defer-until in the past**: deferred recommendation that should have auto-requeued but didn't (operational bug); surface with timestamp diff.

---

## Part 6 — Integrations

### Upstream (S3 reads from)

- **S2 Resident Workspace** (launch point): pharmacist initiates send-to-GP from S2 panel A. S3 receives `recommendation_id` + `pharmacist_id` and composes the envelope.
- **kb-32 craft engine**: `Packet`, `Assessment`, `Citations` for the message body and audit trail.
- **kb-32 framing substrate**: `PerGPObserver.Suggest` for tone, `OptOutStore.IsOptedOut` for opt-out enforcement.
- **kb-32 citations registry**: `RecommendationCitation` set for the citation drawer; `SourceVersion` for amendment status.
- **kb-20 patient profile**: primary prescriber GP lookup for resident.

### Within-platform writes (S3 writes to)

- **`s3_messages` table** (new — Phase 2 migration): one row per envelope sent.
- **`s3_responses` table** (new — Phase 2 migration): one row per GP response.
- **kb-32 override store** (on decline path): writes `OverrideReason` via `POST /v1/craft/override/:recommendation_id`. The FIR auto-write hook (Step 4 Task B) fires if the override's `ReasonCode` is in the reversal set AND the rule classifies.
- **ethics_log** (indirect): every send and every response emits a `decision`-type entry.
- **EvidenceTrace** (TODO(kb-32-extension)): lifecycle events `drafted→sent` and `sent→responded` must be added to the Stage 7 emitter — currently emits only on detected→drafted.

### Transport candidates (Phase 1 pilot — pick one)

For Phase 1 pilot, transport is a single integration partner decision. Candidates:

| Transport | Pros | Cons | Phase 1 readiness |
|---|---|---|---|
| **In-platform secure message** (default) | Audit trail integrated; no third-party dependency; GP must log in to pilot portal | Friction: GP must remember to check the portal | Highest — no external dependencies |
| **FHIR `Communication` resource** | Standards-compliant; integrates with GP EHR if FHIR-capable | Most GPs in target pilot (RACGP small-practice clinicians) don't have FHIR-capable EHRs | Low — most pilot GPs not FHIR-ready |
| **Secure email with deep link to portal** | Familiar workflow; GP gets notification + clicks to portal | Email delivery is best-effort; not auditable end-to-end | Medium — needs email service + delivery monitoring |
| **MyHR (My Health Record) upload** | National infrastructure; GP can see in MyHR | MyHR write access requires patient consent + provider registration; pilot scope may be too narrow | Low for pilot; consider Phase 2 |
| **Fax** (regulatory edge case) | RACGP small practices still receive fax; audit trail is the fax receipt | Operationally archaic; OCR/parsing required for response capture | Unsuitable — abandon |

**TODO(integration partner selection):** before S3 implementation begins, clinical informatics + pilot operations must confirm transport. Default for spec is in-platform secure message; FHIR / email / MyHR are deferrable to Phase 2.

### Future (post-pilot)

- **kb-33 ObservationLayer gRPC**: once available (Step 5), S3 may use `GetEvidenceTrace` to render audit drilldowns directly rather than querying kb-32 endpoints.
- **GP EHR integration**: FHIR / HL7 v2 / proprietary EHR APIs for direct in-EHR display.

---

## Part 7 — Metrics and observability

### S3-emitted events

- `s3_message_sent` — per send, with framing_tone, content_hash, GP / pharmacist demographic stratification
- `s3_message_viewed` — first GP open of view M for a message (proxy for engagement)
- `s3_response_captured` — per response, with response_code, response_code_short, latency-from-send
- `s3_optout_registered` — per opt-out event
- `s3_invariance_violation` — pre-send content-hash check failure (should be zero; alert immediately)

### Aggregations for pilot success metrics

- **Per-GP response rate**: % of `s3_message_sent` events with a corresponding `s3_response_captured` within deadline
- **Time-to-respond**: latency distribution from send to response per GP
- **Decline-with-reason histogram**: distribution of response codes per recommendation type
- **Framing-tone effectiveness**: accept rate stratified by tone (gated by 30-observation floor + opted-out exclusion)
- **Opt-out rate**: % of pilot GPs registered as opted-out
- **Invariance violations**: count of pre-send content-hash check failures (target: zero)

### Phase 3 ethics-monitoring integration

`DetectBiasDisparity` runs against:
- Send rate stratified by resident demographic dimension
- Decline rate stratified by resident demographic dimension
- Framing tone distribution stratified by GP demographic dimension

Disparities above the Phase 3 threshold trigger ethics-monitoring alerts (per `ethics-monitoring/internal/cron/orchestrator.go`).

---

## Part 8 — Anti-patterns

S3 MUST NOT:

1. **Adjust clinical content based on prior GP decline rate.** Frame-vs-content invariance violation. The envelope tone may vary; the Layer 1/2/3 content must not.

2. **Send a recommendation without citation pinning.** Audit-defensibility violation. The pre-send composition step must verify `citations.PinAtFireTime` has run.

3. **Default to a non-default framing tone for an opted-out GP.** Opt-out enforcement is a HARD STOP (Part 5). Violation is a P0 ethics incident.

4. **Auto-resend a declined recommendation without pharmacist re-review.** Gaming the override loop. Re-send is a pharmacist action, not an automatic retry.

5. **Bury the decline pathway behind a confirmation modal.** Phase 3 override-pathway-availability invariant (`override_pathway_test.go`).

6. **Capture GP responses as free-text only.** Structured capture (override taxonomy code) is required; free-text "additional context" is supplementary.

7. **Allow GPs to silently edit responses.** Response is irrevocable. Audit-defensibility: decision provenance must be immutable. Out-of-band correction via pharmacist contact is the only path.

8. **Display different content to different GPs receiving the same recommendation.** S3 invariance check (Part 5): same `recommendation_id` → identical `content_hash` across all `s3_messages` rows.

9. **Use email/SMS/notification body as the primary content vehicle.** Notification is just a pointer; the audit-defensible content lives in the portal view M with citation drawer attached.

10. **Mix audit data with display data.** EvidenceTrace lifecycle events are the audit substrate; S3 view rendering reads from `s3_messages` + `s3_responses` separately. Never derive audit claims from view-render output.

---

## Part 9 — Risks and mitigations

| # | Risk | Probability | Impact | Mitigation | Residual |
|---|---|---|---|---|---|
| 1 | GP non-response (recommendations queue indefinitely) | High | Medium | Response deadline + auto-escalation to senior pharmacist + RMMR re-queue (Flow 4). Per-GP no-response rate monitored; pilot success metric. | Some GPs may chronically not respond; structural pharmacy workflow problem, not S3-solvable. |
| 2 | Framing tone drift (learning produces tone that's "too informal" or "too formal") | Medium | Medium | Per-GP observation-count audit; 30-observation floor before non-default tone; opt-out always available. Periodic tone-effectiveness analysis (Part 7 metrics). | Tone is data-driven; drift is detectable but not preventable. Opt-out is the safety valve. |
| 3 | Decline-reason taxonomy gaming (GPs always pick a low-friction code) | Medium | High | Cross-validate decline codes against substrate state — Phase 3 ethics monitoring catches systemic mismatches (e.g., always-`alert_fatigue` declines on critical-urgency STOPs). Per-GP code-distribution histograms surface anomalies. | Gaming detectable but not prevented by S3 alone; requires pharmacy workflow intervention. |
| 4 | GP message overload / alert fatigue | High | Medium | Per-GP rate limiting (TODO: thresholds in Part 13). Batched non-urgent recommendations (multiple MONITOR/DOSE_CHANGE in single envelope rather than separate sends). Urgency-tag filtering on view I default sort. | Pilot-scale GPs may still feel overwhelmed; structural — adjust upstream signal threshold in kb-32. |
| 5 | Inter-channel inconsistency (S3 portal vs printed RMMR report vs facility EHR show different content for same recommendation) | Medium | High | Single content hash across channels. Every render path must compute and verify the hash. RMMR report (separate spec) explicitly embeds the hash; facility EHR export embeds the hash. | Operational — requires every downstream channel to honour the contract. |
| 6 | Transport unreliability (in-platform notification missed; email delivered to spam) | Medium | Medium | Multi-channel notification fallback: portal + email + (optional) SMS. Per-GP delivery telemetry. Pharmacist sees "GP has not viewed" state in view B. | Pre-pilot — pick one transport, measure, iterate. |
| 7 | Audit-trail gap on `drafted→sent` and `sent→responded` lifecycle events | Certain (Phase 1) | High | TODO(kb-32-extension) flagged in Part 6. Stage 7 EvidenceTrace must be extended; until then, S3-emitted events (`s3_message_sent` etc.) are the proximate audit source — must be queryable for `/v1/explain`. | Pilot Phase 1 may run without full EvidenceTrace lifecycle until extension lands. Document the gap in pilot risk register. |

---

## Part 10 — Performance budget

| Metric | Target | Basis |
|---|---|---|
| View M render (single message) | < 1500 ms p95 | Composed budget: `Packet` fetch + framing tone lookup + citation list |
| Citation drawer open | < 800 ms p95 | `ListCitations` round-trip |
| Response submit | < 1000 ms p95 | `OverrideReason` write + `s3_responses` insert + EvidenceTrace emit |
| View I list (50 messages) | < 1000 ms p95 | Indexed query on `s3_messages.gp_id` |
| Pre-send invariance check | < 200 ms | In-memory content-hash computation against kb-32 cached hash |
| Opt-out registration | < 500 ms | `POST /v1/framing/optout/:gp_id` round-trip |

All TODO(integration-time validation). Pre-pilot estimates; revise against staging measurements.

---

## Part 11 — Accessibility

- **WCAG 2.1 AA** minimum.
- **GP mobile context**: many GPs review messages on phone or tablet during clinic / ward rounds. View M must be fully usable at 320px width.
- **Screen-reader support**: response capture form is fully labelled; override taxonomy dropdown options have aria-describedby pointing to the reason-code description.
- **Keyboard navigation**: every action reachable via keyboard; response form submittable without mouse.
- **Notification accessibility**: if notification email is used (transport candidate), email must be plain-text-readable with the portal link clearly marked.
- **Language**: pilot Phase 1 is English-only; non-English support is Phase 2.

---

## Part 12 — Out of scope for this spec

- **S2 Resident Workspace** — separate spec (`2026-05-11-s2-resident-workspace-spec.md`). S3 is launched from S2; S3 does not own resident drill-down.
- **S4 RACH Operational View** — facility operator surface, Phase 2.
- **S5 Standard 5 Evidence Panel** — facility auditor surface, Phase 2.
- **Direct patient/family communication** — out of pilot scope.
- **Billing surfaces (MBS RMMR items)** — see RMMR workflow spec.
- **Prescribing-side integration** — S3 reads what pharmacists send and what GPs respond. Writing prescriptions back to GP EHR is Phase 2 (or Phase 3) territory.
- **Multi-pharmacist consults on a single recommendation** — Phase 2.
- **GP-to-pharmacist initiated messages** (GP wants pharmacist input on a non-CardioFit-originated question) — Phase 2.

---

## Part 13 — Open questions for clinical informatics + GP advisory group

1. **Transport selection.** Default for spec is in-platform secure message. Should pilot Phase 1 also offer secure email with deep link as a backup notification channel? FHIR Communication and MyHR upload deferred to Phase 2 — confirm.

2. **Response deadline policy.** Should there be a default response deadline (e.g., 72h for amber, 24h for red, 14d for green)? Or no deadline (recommendations queue indefinitely)? Deadline drives Flow 4 escalation behaviour.

3. **Defer-with-timeframe options.** Current proposal: 24h / 72h / 7d / 14d / custom. Are these the right options? Should there be an upper bound on defer length (e.g., no defer beyond 30d — forces re-review)?

4. **Reasoning minimum length.** Proposed 10 chars on decline (vs 20 on S2 override). Different audience, different friction tolerance — confirm.

5. **No-response handling.** Currently: surfaces as `expired` in pharmacist's view B; recommendation stays drafted. Should it auto-escalate to senior pharmacist after a second deadline (e.g., 7d after initial deadline)? Should it auto-convert to a documentation override?

6. **Re-send semantics.** Pharmacist re-sends a no-response recommendation. Is it a new `s3_messages` row (preserves history) or an update to the existing row (cleaner queue)? Spec proposes new row; confirm.

7. **GP-side accept code.** When GP clicks Accept, the response code is the synthetic `"accepted"`. Should there be sub-codes (`accepted_will_implement_now` / `accepted_will_implement_at_next_visit`)? Adds friction; consider for Phase 2.

8. **Decline + Reasoning enforcement on Defer.** Currently Reasoning is optional on Defer (it's required on Decline). Should Defer also require Reasoning (to capture clinical context)?

9. **Practice-pattern leakage.** View H aggregates a GP's response patterns. Should this be visible to other GPs in the same practice (collegial learning)? To senior pharmacists (oversight)? To the GP's auditor (regulatory)? Privacy + utility trade-offs.

10. **Framing tone vocabulary.** `PerGPObserver` learns from observation outcomes. What's the canonical Phase 1 vocabulary for framing tones? (`brief | detailed | collegial | clinical | default` — confirm or revise.)

11. **Opt-out granularity.** Currently per-GP-account. Should it be per-recommendation-type (e.g., opt out of STOP framings only)? Phase 2.

12. **Per-GP rate limit thresholds.** Per Risk #4. What's the canonical max recommendations-per-day to a single GP? Per-week? Per-resident?

---

## Pre-acceptance gate for S3 implementation work

Before S3 implementation begins:

1. ✅ Step 4 substrate prerequisites merged to main (`f3422b23` — done 2026-05-11)
2. ⏳ Clinical informatics + GP advisory group review Part 13 questions
3. ⏳ Transport partner selected (default: in-platform secure message)
4. ⏳ S2 Resident Workspace spec authored (done — `2026-05-11-s2-resident-workspace-spec.md`)
5. ⏳ kb-32 override store extension: `drafted→sent` and `sent→responded` lifecycle events added to Stage 7 EvidenceTrace emitter
6. ⏳ GP advisory group constituted (per pre-pilot operational gate from roadmap)
7. ⏳ Pre-pilot ethics audit covers the invariance gate that S3 must preserve

Spec complete and saved.
