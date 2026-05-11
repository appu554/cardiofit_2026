# Pharmacist Self-Visibility — Implementation Guidelines v1.0

**Date:** May 2026
**Service:** Pharmacist Self-Visibility module within `kb-32-recommendation-craft` and `shared/v2_substrate/permissions`
**Implementation phase:** Phase 1 of Layer 2/3 plan (Weeks 5–10), with extensions for development-not-evaluation framing, contestation operational layer, and cross-employer portability
**Builds on:** *Vaidshala v3.0 Product Proposal* §5 (Buyer 4 — pharmacists themselves), §8 (expertise gap closure), §9 (ethical architecture), §10 (workforce development); *Recommendation Craft Implementation Guidelines v1.0* (cross-service collaboration); *Layer 2 & 3 Implementation Plan* (7 May 2026) §1.1–1.4 (trust architecture); *Pilot Design v1.0* (freemium architecture)

**Reading order:** Engineering and product leads read Parts 1–4 (philosophy, boundary architecture, dashboard surfaces). Clinical leads read Parts 4–7 (KPI specifications, development framing, RPL/CPD pathways) for clinical sign-off. Legal and ethics leads read Parts 8–10 (consent, contestation, aggregation rules, portability). Implementers read Parts 11–14 (file structure, contracts, tests, sequencing).

---

## Part 0 — Honest framing: distinctive positioning

The pharmacist self-visibility module sits in a structurally underdeveloped design space. Most clinical performance dashboards are designed for team or management consumption (the Wong 2022 scoping review of clinical indicator dashboards found 67% designed for team use, only 22% for individual clinicians). The empirical literature on individual-clinician self-monitoring tools is thin, and the literature that does exist (Ivers 2025 Cochrane update) studies supervisor-delivered audit and feedback rather than self-delivered professional development.

This means the module cannot rely on conventional dashboard patterns lifted from population health management or quality improvement contexts. Those contexts assume a manager-clinician relationship; the pharmacist self-visibility module assumes a professional self-relationship.

What's established and we adopt with discipline:

- **Audit and feedback theory (Ivers 2012, 2025; Jamtvedt 2003).** A&F works when professionals are not performing well to begin with, when delivered repeatedly with verbal and written components, and with clear targets and action plans.
- **Theoretical Domains Framework (TDF) — Cane 2012.** Behaviour change interventions targeting knowledge, motivation, goals, and social influences are well-studied.
- **CPD portfolio-based models (Australia, Ireland, New Zealand).** Reflective practice through portfolios is the dominant pharmacy CPD model.
- **Algorithmic management occupational health frameworks (Bowdler 2026; Vignola 2023).** Worker-protective design for algorithmic systems is becoming a regulatory expectation.

What's distinctive and we lead with:

- **Professional identity and emotion as design targets.** The e-A&F systematic review (van Dijk 2017) explicitly found these TDF domains were not targeted by any studies. This is a structural gap where Vaidshala leads — designing for the pharmacist's professional identity formation and emotional sustainability of clinical work, not just knowledge and motivation.
- **Trajectory and ceiling framing instead of peer comparison.** The Salahudeen family physician dashboard study found above-average performers stopped improving when shown peer comparison. The module uses trajectory (own improvement over time) and ceiling (distance to anonymised best-in-class) instead.
- **Anti-surveillance architecture as first principle.** The visibility boundary architecture (Part 2) is enforced at the data-structure level, not just at the UI level. This means surveillance is not just discouraged; it is architecturally prevented.
- **Self-visibility before aggregation.** The pharmacist sees their own data first, with explicit time and contextual gates before any upward aggregation. This is rare in workplace performance systems.
- **Contestation as architecture.** Any KPI that could feed an employment decision has a structured contestation pathway built into the substrate, not bolted on as policy.
- **Cross-employer portability.** The pharmacist's record persists across employer transitions. The module treats the pharmacist as the data subject, not the employer's asset. This is a structural alignment with bottom-up adoption motion that no enterprise-first dashboard can replicate.

These six elements are what the module leads with. The dashboard surfaces (Part 3) are necessary scaffolding; the trust and development architecture is the moat.

---

## Part 1 — Design philosophy

Five principles, each grounded in empirical evidence, shape every implementation decision.

**Principle 1: Self-visibility before aggregation.** The pharmacist sees their own data first. Aggregation upward to employer requires consent, time gating, or contractual basis. The temporal order matters: a pharmacist who learns of their own performance pattern from their employer — rather than from their own dashboard — has experienced surveillance, regardless of how the data flowed. The architecture prevents this temporal inversion.

**Principle 2: Development, not evaluation.** The module's primary purpose is professional development, not performance evaluation. Language, framing, and metric construction adhere to this distinction throughout. Evaluative metrics (those feeding employment decisions) are explicitly bounded, separately surfaced, and contestable. Most of the dashboard is developmental and never feeds employer decisions.

**Principle 3: Psychological safety as architectural commitment.** Per the 2026 healthcare workplace research, psychological safety is foundational to trust in clinical environments. The module operationalises this through: no surprise visibility, restraint from evaluative judgment, contestation as right not exception, the explicit "this is observation, not evaluation" framing in UI text, and design choices that surface concerns the pharmacist can act on rather than concerns the platform has formed about the pharmacist.

**Principle 4: Trajectory and ceiling, not peer comparison.** Empirical evidence (Salahudeen et al., family physician dashboard study) shows above-average performers stop improving when shown peer comparison. The module uses *trajectory* (the pharmacist's own improvement over time) as the primary motivational frame, and *ceiling* (anonymised best-in-class benchmark) as the aspirational frame. Peer comparison is used sparingly, anonymised, and only where empirically justified.

**Principle 5: Portability across employers.** The pharmacist's clinical work, recommendation outcomes, CPD log, and career portfolio persist across employment transitions. The platform treats the pharmacist as the data subject. This is both an ethical commitment and a strategic feature — pharmacists who can take their record with them have stronger professional autonomy, and this autonomy is what makes the bottom-up adoption motion durable.

---

## Part 2 — The visibility boundary architecture

The module's most important architectural specification is which data is visible to whom, under what conditions, and through what channels. This is enforced at the data structure level, not just at UI rendering.

### 2.1 Five visibility classes

Every data element in the module is classified into one of five visibility classes:

| Class | Visible to pharmacist | Visible to employer | Visible to RACH | Visible to regulator |
|---|---|---|---|---|
| **Pharmacist-Only-Always (POA)** | Yes | No, ever | No | No |
| **Pharmacist-Default-Private (PDP)** | Yes | Only with explicit pharmacist consent | No | Anonymised aggregate only |
| **Pharmacist-First-Then-Aggregated (PFA)** | Yes (pharmacist sees first) | Aggregated after time gate | Aggregated after time gate | Aggregated after time gate |
| **Workflow-Operational (WO)** | Yes | Yes (for workflow function) | Yes (for workflow function) | Yes (for audit) |
| **Audit-Defensible (AD)** | Yes | Yes | Yes | Yes (full detail under formal data-sharing agreement) |

### 2.2 What lives in each class

**Pharmacist-Only-Always (POA):**
- Reflective writing entries
- Personal CPD reflections (the reflection itself, not the activity log)
- Self-assessed confidence scores on clinical scenarios
- Personal goal-setting against the CPD plan
- Notes on professional development direction
- Career portfolio narrative (the pharmacist's own descriptions)

**Pharmacist-Default-Private (PDP):**
- Per-GP framing observations from own work (only the pharmacist sees per-GP patterns)
- Recommendation appropriateness scores on own recommendations
- Time-allocation breakdown at the personal level
- Restraint signal override patterns
- Per-resident reasoning chains the pharmacist authored

**Pharmacist-First-Then-Aggregated (PFA):**
- Recommendation Implementation Rate (RIR) — pharmacist sees own first; aggregated to employer after 90-day rolling window with explicit contractual notice
- Class-specific implementation rates vs Ramsey 2025 baseline — same pattern
- Context-assembly time
- Recommendations made per session
- CPD activity completion rate

**Workflow-Operational (WO):**
- Active recommendations in lifecycle (visible to anyone with workflow role on this resident)
- Active monitoring obligations
- Pharmacist availability and roster status
- Inter-pharmacist handoff records

**Audit-Defensible (AD):**
- EvidenceTrace records (full audit trail)
- Algorithmic-vs-human distinction logs
- Contestation records and outcomes
- Source citation versioning
- Substrate state at every recommendation fire time

### 2.3 Visibility class enforcement

The visibility class is a first-class field on every data structure. The query layer enforces visibility at the data-fetch boundary, not at the UI layer. This means:

- A misconfigured UI cannot accidentally expose POA or PDP data
- An attacker compromising the UI cannot bypass visibility enforcement
- Audit logs record every access attempt, including those denied

```go
// /shared/v2_substrate/visibility/class.go
type VisibilityClass int

const (
    POA VisibilityClass = iota  // Pharmacist-Only-Always
    PDP                         // Pharmacist-Default-Private
    PFA                         // Pharmacist-First-Then-Aggregated
    WO                          // Workflow-Operational
    AD                          // Audit-Defensible
)

type VisibilityRule struct {
    DataElement     string
    Class           VisibilityClass
    PharmacistID    string  // subject pharmacist
    AggregationGate *AggregationGate  // for PFA only
    ConsentRecord   *ConsentRecord    // for PDP, where applicable
}

type AggregationGate struct {
    MinObservations  int           // e.g., 30 recommendations before aggregating RIR
    TimeWindow       time.Duration // e.g., 90 days rolling
    ContractualBasis string        // e.g., "enterprise tier deployment contract clause 4.2"
    ExplicitNotice   bool          // notice given to pharmacist before aggregation began
}
```

### 2.4 The temporal-order commitment

A specific commitment that distinguishes self-visibility from surveillance: **the pharmacist sees their own data before any aggregation occurs.** This is enforced operationally:

- For PFA-class data: the pharmacist's dashboard shows the metric at observation time. Employer aggregation is delayed by a defined window (default 30 days) so the pharmacist always sees current data first.
- For PDP-class data: the employer never sees the data without explicit pharmacist consent. Consent is per-data-element, time-bounded, revocable.
- For POA-class data: there is no aggregation pathway. The data structurally cannot reach the employer.

The temporal-order commitment is what prevents the surveillance experience even when the underlying data flows are similar to enterprise performance management.

### 2.5 Pharmacist-controlled exports

The pharmacist can export their own data at any time, in standard formats (PDF, JSON, CSV). Exports include all POA, PDP, and pharmacist's own PFA data. This supports:

- RPL evidence pack generation
- CPD record submission to AHPRA
- Career portfolio download
- Cross-employer migration when pharmacist changes jobs
- Personal record-keeping

The export capability is independent of employer contract — a pharmacist on free tier exports their work record at any time, regardless of employer adoption status.

---

## Part 3 — Dashboard surfaces

The pharmacist self-visibility module renders six concrete dashboard surfaces. Each has explicit purpose, content, and design constraints grounded in the empirical evidence.

### 3.1 Today's Worklist

**Purpose:** Risk-stratified daily queue. The pharmacist's primary operational interface.

**Content:**
- Residents requiring attention today, prioritised by composite risk score (recent fall + recent admission + new high-risk medication + overdue monitoring + family concern)
- Each entry shows resident identifier, top 1–3 reasons surfaced, estimated time-to-action
- Restraint signals visible alongside action prompts (per craft engine §10)

**Design constraints:**
- The worklist is **today's work**, not evaluation. No metric scoring on this surface.
- Items completed disappear from the worklist into the historical record
- Items deferred reappear at their forced review date with an annotation showing the prior deferral reason

**Underlying visibility class:** WO (workflow-operational; visible to pharmacy operations but not aggregated as performance data)

### 3.2 My Recommendations

**Purpose:** The pharmacist's own recommendation lifecycle view. Not employer-visible.

**Content:**
- Recommendations the pharmacist authored, in current state (drafted, submitted, viewed, decided, implemented, monitoring-active, outcome-recorded, closed, rejected, deferred)
- For each, the full Layer 1/2/3/4 packet (per craft engine §2)
- Override reasons captured for rejected recommendations (per craft engine §5)
- Outcomes for implemented recommendations (per monitoring lifecycle)

**Design constraints:**
- This is the pharmacist's **clinical work record**. It is professional development infrastructure, not performance scorecard.
- Rejected recommendations are framed as learning opportunities, not failures. UI text uses "What we can learn from this" rather than "Why this was declined."
- Patterns across recommendations (e.g., "your deprescribing recommendations have higher acceptance when you include monitoring plans up front") surface as gentle observations, not directives.

**Underlying visibility class:** PDP (pharmacist's own work; employer sees only with explicit consent)

### 3.3 My GP Relationships

**Purpose:** Per-GP framing patterns observed from own work. Used to inform recommendation craft, not to evaluate prescribers or pharmacist relationship management.

**Content:**
- For each GP the pharmacist works with, observed acceptance patterns in the pharmacist's own recommendations (recommendation type, evidence source, channel, framing style)
- Suggested framing observations for upcoming recommendations to that GP
- Recent decision history

**Design constraints (critical):**
- **No GP scorecards.** The UI never displays "Dr Smith has 42% acceptance rate." It displays "recommendations to Dr Smith have landed better when X."
- **No GP rankings.** Prescribers are not compared, ranked, or characterised.
- **Aggregate-only across pharmacists for prescriber-side observations.** A given prescriber's pattern across all pharmacists is anonymised; a single pharmacist cannot be identified as the source of any specific observation.
- **GP opt-out respected.** If a GP has opted out of pattern tracking, the dashboard shows default framing only.

**Underlying visibility class:** PDP (pharmacist's own observations; never aggregated to employer)

### 3.4 My Clinical Reasoning Patterns

**Purpose:** Surface the pharmacist's own reasoning patterns over time, supporting professional development.

**Content:**
- Recommendation type distribution (deprescribing vs dose change vs monitoring vs add) over time
- Restraint signal override patterns (when did the pharmacist proceed despite restraint, with what reasoning)
- Appropriateness score distributions on own recommendations
- Class-specific implementation rates vs Ramsey 2025 baseline (own trajectory)
- Time-allocation breakdown (PiRACF protocol: reviews / communication / administration / education)

**Design constraints:**
- **Trajectory framing first.** The primary visualisation is the pharmacist's own improvement over time, not comparison.
- **Ceiling framing as aspiration.** Anonymised best-in-class benchmarks shown as "what excellent ACOPs achieve" rather than peer ranking. This addresses the family physician dashboard finding.
- **Reflective writing prompts.** Pattern surfacing is paired with reflective prompts: "You've been recommending more dose adjustments and fewer deprescribings recently. What's driving that?" The prompt invites reflection, not justification.

**Underlying visibility class:** PFA for trajectory metrics (aggregated to employer with time gate); POA for reflective writing entries

### 3.5 My CPD Progression

**Purpose:** AHPRA CPD compliance and personal development tracking.

**Content:**
- CPD activity log auto-tagged from clinical work (per v3.0 §10)
- AHPRA-required hours by activity category
- Reflective writing entries on CPD activities
- CPD plan and goal progression
- Submission-ready CPD record export

**Design constraints:**
- **CPD activity is auto-detected** from the pharmacist's clinical work, but the pharmacist confirms each activity before it counts. No silent attribution.
- **Reflective writing is POA always.** The pharmacist's own reflections on cases never flow to employer or any other party.
- **AHPRA submission is pharmacist-controlled.** The pharmacist generates and submits records; the platform does not submit on their behalf.

**Underlying visibility class:** Mostly POA (reflections); WO for activity log (so employer can see CPD compliance status, but not the reflective content)

### 3.6 My Career Portfolio

**Purpose:** Longitudinal record of clinical work, supporting career advancement, RPL applications, and inter-employer portability.

**Content:**
- Summary of clinical scenarios handled (anonymised, structured)
- Recommendation outcomes (anonymised aggregate)
- CPD completion record
- Credentialing status (ACOP, MMR, specialty)
- RPL-evidence pack generation
- Self-authored career narrative
- Endorsements (peer, employer-volunteered)

**Design constraints:**
- **Pharmacist authors the portfolio narrative.** The platform provides data; the pharmacist tells their own story.
- **Anonymisation strict.** Resident identifiers are scrubbed. Employer identifiers may be included with consent.
- **Cross-employer persistence.** When the pharmacist changes employers, the portfolio persists. The new employer does not automatically see the prior employer's data unless the pharmacist consents.
- **Export as standard.** The portfolio exports as PDF for traditional applications and structured JSON for digital workflows.

**Underlying visibility class:** Pharmacist controls visibility; default POA for narrative, exports controlled by pharmacist

---

## Part 4 — KPI specifications

Each KPI surfaced in the dashboard has explicit specification. Format: definition, computation, surface, visibility class, aggregation rules, contestation pathway.

### 4.1 Recommendation Implementation Rate (RIR)

**Definition:** Percentage of pharmacist's submitted recommendations that reach `implemented` state within agreed window (default 30 days), with appropriate scope and substrate observability.

**Computation:**
```
RIR = (count of recommendations in implemented state OR beyond)
      / (count of recommendations in submitted state OR beyond, age > 30 days)
```

Class-specific RIR computed similarly with class filter. Computed in rolling 90-day window.

**Surface:** My Clinical Reasoning Patterns; trajectory chart over time; class-specific breakout against Ramsey 2025 baseline.

**Visibility class:** PFA. Pharmacist sees own RIR continuously updated. Employer aggregation begins after 30-day delay, with quarterly rolling window, with contractual notice clause in enterprise tier deployment.

**Aggregation rules:**
- Minimum 30 recommendations before any aggregation (statistical adequacy)
- Aggregation always rolling window, never point-in-time
- Aggregated values are pharmacist-anonymous within their employer (employer sees their network's RIR distribution, not per-pharmacist)
- Per-pharmacist RIR available to employer only with explicit pharmacist consent

**Contestation pathway:** Pharmacist can contest:
- Specific recommendation classification (e.g., "this should not have counted because the GP indicated they wanted clarification, not rejection")
- Window definition (e.g., "the 30-day window doesn't suit this resident's situation")
- Aggregation methodology
Contestation surfaces in pharmacist dashboard and in any employer view of the metric.

### 4.2 Class-specific implementation rates vs Ramsey 2025

**Definition:** Implementation rate for specific recommendation classes, measured against the Ramsey et al. 2025 national baseline (colecalciferol 37%, calcium 36%, PPI 43%, cessation overall 51%, dose reduction 49%).

**Computation:** Class-filtered RIR in rolling 90-day window with minimum-observation threshold (10 per class).

**Surface:** My Clinical Reasoning Patterns; comparison against Ramsey baseline shown as ceiling not peer rank.

**Visibility class:** PFA, same rules as RIR.

**Contestation pathway:** Same as RIR.

### 4.3 Context-assembly time

**Definition:** Time from "case opened" to "first clinical decision logged" per resident review.

**Computation:** Median across rolling 30 reviews. Excludes administrative gaps (interruptions logged by pharmacist).

**Surface:** My Clinical Reasoning Patterns; trajectory chart over time.

**Visibility class:** PFA. Aggregate context-assembly time visible to employer for workforce planning, not for individual evaluation.

**Contestation pathway:** Pharmacist can flag a review where time was atypically long for clinical reasons (complex resident, family meeting, etc.) and have it excluded from the metric.

### 4.4 Recommendation appropriateness score

**Definition:** Average appropriateness score across the pharmacist's own recommendations (per craft engine §9 five-dimension rubric).

**Computation:** Mean of overall appropriateness scores, rolling 90 days.

**Surface:** My Clinical Reasoning Patterns; trajectory only; never compared to peers.

**Visibility class:** PDP. The pharmacist sees their own appropriateness trajectory. Employer never sees per-pharmacist appropriateness — only aggregated at network level.

**Why PDP not PFA:** Appropriateness score is too closely tied to clinical judgment to be aggregated to employer at individual level without risking interference with clinical autonomy.

**Contestation pathway:** Per-recommendation contestation; aggregate contestation only through formal review.

### 4.5 Restraint override pattern

**Definition:** Frequency and reasons for proceeding despite restraint signals.

**Computation:** Count of restraint overrides per recommendation type per quarter.

**Surface:** My Clinical Reasoning Patterns; reflective prompt accompanies high-frequency overrides ("You've been overriding restraint signals on antipsychotic deprescribing recently. What's the clinical reasoning?")

**Visibility class:** POA. The pharmacist alone sees their restraint override pattern. This is reflective, not evaluative.

**Why POA:** Restraint override pattern is intimate professional judgment territory. Surveilling this would compromise clinical autonomy.

**Contestation pathway:** Not applicable; not employer-visible.

### 4.6 CPD activity completion

**Definition:** AHPRA-required CPD hours completed by category, with reflective entries.

**Computation:** Auto-tagged activities pharmacist confirmed, summed by AHPRA category.

**Surface:** My CPD Progression dashboard.

**Visibility class:** WO for completion status (employer can see compliance); POA for reflective content.

**Contestation pathway:** Pharmacist can re-categorise activities; can challenge auto-tagging.

### 4.7 Career portfolio metrics

**Definition:** Longitudinal scenario coverage, recommendation outcomes, credential progress.

**Computation:** Anonymised aggregate over career to date.

**Surface:** My Career Portfolio.

**Visibility class:** Pharmacist-controlled; export-only by default.

---

## Part 5 — Development-not-evaluation framing

The module's primary purpose is professional development, and the design must operationalise this distinction throughout. This section specifies the design choices that hold the development-not-evaluation framing.

### 5.1 Reflective writing prompts

Aligned with the Pharmacy Times finding that reflective diary maintenance improves both emotional intelligence and clinical practice in pharmacists.

**Prompt design:**
- Open-ended questions, not yes/no
- Past-tense and present-tense, not future-imperative ("What worked?" not "What will you change?")
- Pattern-revealing without judgment ("You've been recommending more deprescribing recently — what's behind that?")
- Time-bounded and specific (weekly prompts on recent work, not abstract)
- Privacy-preserving (no shared visibility ever)

**Example prompts (rotating monthly):**
- *"This month you authored 23 recommendations. Which one are you proudest of, and why?"*
- *"You overrode the restraint signal on three antipsychotic recommendations this quarter. What clinical reasoning supported that?"*
- *"Your context-assembly time has dropped from 22 to 9 minutes. What's making that possible?"*
- *"You've been working with Dr Smith for 6 months now. What have you learned about what lands well?"*

**Implementation:** Reflective entries are POA always. Pharmacist can review own past entries; cannot share without explicit export action. Entries can be referenced in CPD record submissions if pharmacist chooses.

### 5.2 Pattern surfacing without judgment

When the platform identifies a pattern in the pharmacist's work, the framing matters.

**Anti-pattern:** *"Your acceptance rate is below your peers."*

**Aligned pattern:** *"Your acceptance rate has dropped 8 percentage points over the last quarter. Several deprescribing recommendations have been declined recently. Want to look at which ones?"*

The aligned framing surfaces the observation, supplies actionable detail, and invites engagement. It does not compare, evaluate, or judge.

### 5.3 Trajectory and ceiling framing

Per the Salahudeen family physician dashboard finding, peer comparison demotivates above-average performers. The module replaces peer comparison with two alternative frames:

**Trajectory:** The pharmacist's own improvement over time. Always primary. *"Your context-assembly time has improved 40% in 6 months."*

**Ceiling:** Anonymised best-in-class benchmark. Used as aspiration, not ranking. *"Best-in-class ACOPs achieve median context-assembly under 6 minutes. You're at 8 minutes."*

Peer percentile is **not surfaced** in the pharmacist's own dashboard. (It may be relevant in employer aggregated views with consent.)

### 5.4 Above-average-performer engagement

Specifically designed to address the family physician finding that above-average performers stop improving.

**For above-average performers, the dashboard:**
- De-emphasises the metric where they're above average (no "you're still 23% above peer")
- Surfaces gaps relative to ceiling (not relative to peers)
- Highlights areas where their patterns suggest unexplored development
- Offers reflective prompts on what's working that could be shared with colleagues (peer teaching opportunity)

This shifts the motivational frame from "stay above peers" to "continue growing."

### 5.5 Below-baseline-performer support

Pharmacists whose metrics fall below relevant baselines need specific support, not just visibility.

**For below-baseline-metric situations:**
- Visibility is surfaced gently, with context (acknowledges complexity of caseload, recent transitions, etc.)
- Reflective prompts focus on what's contributing, not what's wrong
- Action options are offered (peer consultation, rule of three review, training resource)
- No employer flag is raised purely on metric deviation; only sustained pattern with documented support attempts

This holds the development framing even where evaluation pressure might naturally creep in.

### 5.6 The explicit framing in UI text

The dashboard's standing UI text reinforces the development framing:

- Header: *"Your professional development dashboard"*
- Sub-header on metric panels: *"For your own development. Patterns you choose to share with your employer remain in your control."*
- Footer: *"This is observation, not evaluation."*
- Privacy notice link: *"What your employer can and cannot see."*

These small UI choices set expectations and operationalise the trust architecture.

---

## Part 6 — Algorithmic-vs-human distinction in self-view

When the platform surfaces an observation, the pharmacist must be able to distinguish between:
- **Platform-suggested patterns** (the algorithm noticed something)
- **Substrate facts** (the data shows X)
- **Pharmacist-recorded reflections** (you noted Y)
- **Hybrid observations** (the algorithm noticed, you confirmed)

This distinction is foundational to the v3.0 §9 Principle 4 commitment (algorithmic-vs-human distinction in audit trail) and to the algorithmic management transparency requirements emerging in healthcare regulation.

### 6.1 Visual distinction in UI

Each surface element carries a visual marker indicating its source:

- 🔵 **Substrate fact** — direct data computation (e.g., RIR trajectory)
- 🟡 **Platform suggestion** — algorithmic pattern detection (e.g., "your deprescribing acceptance is changing")
- 🟢 **Pharmacist reflection** — pharmacist's own recorded entry
- 🟣 **Hybrid** — algorithmic observation confirmed by pharmacist

Hover or click on each marker reveals the source attribution and timestamp.

### 6.2 Substrate fact handling

Substrate facts are computed directly from EvidenceTrace queries. They are reproducible and verifiable. The UI links each fact to the underlying substrate query (per craft engine four-layer architecture).

### 6.3 Platform suggestion handling

Platform suggestions are flagged explicitly as such. The UI text uses suggestion language ("we noticed", "this pattern suggests") never assertion language ("your problem is", "you should"). Pharmacist can dismiss a platform suggestion (it disappears from current view but is logged).

The MIN_OBSERVATIONS_THRESHOLD applies — platform suggestions on patterns observed across fewer than 30 instances are not surfaced.

### 6.4 Pharmacist reflection handling

Pharmacist reflections are POA. The platform never re-surfaces reflective content algorithmically (no pattern detection on reflections). This protects the safe-space character of reflective writing.

### 6.5 Hybrid observation handling

When a platform suggestion has been confirmed by the pharmacist (e.g., they wrote a reflective entry that aligns with the surfaced pattern), the observation transitions to hybrid status. This is the most actionable category — both the algorithm and the pharmacist have aligned on the observation.

Hybrid observations may inform the pharmacist's CPD plan, may be shared in peer consultation, and may be referenced in employer conversations *if the pharmacist chooses*.

### 6.6 Audit trail

Every observation surfaced to the pharmacist is logged in EvidenceTrace with:
- `observation_class`: substrate-fact / platform-suggestion / pharmacist-reflection / hybrid
- `algorithmic_origin_id`: which pattern detector / rule fired (if applicable)
- `human_actor_id`: which pharmacist confirmed (if applicable)
- `timestamp`
- `pharmacist_response`: confirmed / dismissed / no-action / pending
- `subsequent_action`: e.g., "fed into CPD plan", "discussed with peer"

---

## Part 7 — RPL evidence and CPD export pathways

The pharmacist self-visibility module is the substrate for two regulator-facing exports: RPL evidence packs (for ACOP credentialing) and AHPRA CPD records.

### 7.1 RPL evidence pack generation

Per v3.0 §10, the 30 June 2026 credentialing cliff makes RPL evidence a time-bound commercial wedge. The pack generator extracts structured evidence from the pharmacist's longitudinal substrate work.

**Pack contents (aligned with APC RPL framework):**
- Competency dimension 1: Clinical assessment — case examples with anonymised resident detail
- Competency dimension 2: Medication review — recommendation lifecycle examples with outcomes
- Competency dimension 3: Communication — GP collaboration examples
- Competency dimension 4: Quality use of medicines — facility-level work examples
- Competency dimension 5: Professional practice — CPD reflections, peer consultation examples

Each evidence item:
- Pulled from EvidenceTrace with original timestamps
- Anonymised at resident, GP, and facility level (configurable for cases where identification is consented)
- Pharmacist-curated (the pharmacist selects which examples to include)
- Reflective annotation by the pharmacist accompanies each example
- Exportable as structured PDF aligned with APC submission templates

**Generation flow:**
1. Pharmacist initiates RPL pack generation
2. Platform surfaces candidate evidence items per competency dimension (substrate-driven)
3. Pharmacist reviews, selects, and annotates
4. Platform formats as APC-aligned PDF
5. Pharmacist downloads; platform retains no submission record

### 7.2 AHPRA CPD record export

AHPRA requires 40 hours of CPD per registration cycle, of which a portion must be reflective.

**Export contents:**
- Activities completed by AHPRA category
- Reflective entries linked to activities
- Self-assessed learning outcomes
- Submission-ready record format

**Generation flow:**
1. Pharmacist reviews CPD log monthly or quarterly
2. Auto-tagged activities are confirmed or reclassified
3. Reflective entries are written (POA)
4. At submission time, pharmacist exports record to AHPRA portal
5. Platform retains record but does not submit on pharmacist's behalf

### 7.3 Cross-registration support

For pharmacists holding multiple credentials (ACOP + MMR + specialty), the export pathway supports per-credential filtering of evidence and activity records.

---

## Part 8 — Consent and contestation operational layer

The trust architecture (v3.0 §9) establishes contestation as architectural commitment. This section specifies the operational layer.

### 8.1 Consent model

Consent in the module operates at three granularities:

**Element-level consent:** Pharmacist consents to specific data elements being aggregated upward (e.g., RIR aggregated to employer with 30-day window). Default is no consent except where contractually established at enterprise tier deployment.

**Purpose-bounded consent:** Each consent specifies the purpose (workforce planning, contract retention defence, regulatory compliance evidence). Consent for one purpose does not extend to others.

**Time-bounded consent:** Consent has explicit expiration. Default 12 months. Renewal requires affirmative re-consent.

**Revocable consent:** Pharmacist can revoke any consent at any time. Revocation is forward-looking (data already aggregated remains, but no further aggregation occurs without renewed consent).

```go
// /shared/v2_substrate/permissions/consent.go
type ConsentRecord struct {
    ID                 uuid.UUID
    PharmacistID       string
    DataElement        string  // e.g., "rir_class_specific"
    AggregationTarget  string  // e.g., "employer_pharmacy_xyz"
    Purpose            string  // bounded list
    GrantedAt          time.Time
    ExpiresAt          time.Time
    RevokedAt          *time.Time
    RevocationReason   *string
}
```

### 8.2 Contestation pathway

Any KPI surfaced to a pharmacist that could feed an employment decision has a contestation pathway. Pharmacist contests by:

1. Clicking "Contest this metric" on the dashboard
2. Selecting contestation type (specific recommendation classification, window definition, methodology, factual error)
3. Providing free-text reasoning
4. Optionally requesting peer or independent review

The contestation creates a formal record visible to both pharmacist and (if the metric was employer-aggregated) the employer. The contested metric is flagged in employer view until resolved.

```go
// /shared/v2_substrate/permissions/contestation.go
type Contestation struct {
    ID               uuid.UUID
    PharmacistID     string
    MetricID         string
    ContestationType string  // "classification" / "window" / "methodology" / "factual"
    Reasoning        string
    SubmittedAt      time.Time
    Status           string  // "pending" / "under_review" / "resolved_in_pharmacist_favor" / "resolved_methodology_revised" / "resolved_no_change" / "withdrawn"
    Reviewer         *string // peer pharmacist or independent reviewer
    ReviewedAt       *time.Time
    Resolution       *string
    EmployerNotified bool
}
```

### 8.3 Algorithmic-determination protection

Per v3.0 Risk 12 (algorithmic performance management legal exposure), the module commits:

- No algorithmic determination is the sole basis for an adverse employment decision
- Every metric feeding employer view is contestable
- Contestation pauses any employer action based on the contested metric
- Independent (non-employer-affiliated) review is available for high-stakes contestations

This commitment is operationalised through contractual clauses in enterprise tier deployment (the contract explicitly prohibits sole-algorithmic-basis adverse decisions) and through technical architecture (the contestation flag travels with the metric in employer view, prompting pause).

### 8.4 Resolution pathways

Contestations resolve through one of four outcomes:

- **Resolved in pharmacist's favour:** Metric is corrected, retroactive correction applied to employer view if applicable
- **Resolved with methodology revision:** Contestation revealed methodology problem; methodology revised, applied prospectively
- **Resolved no change:** Contestation reviewed, original metric upheld with documented reasoning
- **Withdrawn:** Pharmacist withdraws contestation

All outcomes are logged in EvidenceTrace.

---

## Part 9 — Employer-aggregation rules

When an employer pharmacy adopts the enterprise tier, the consent and aggregation rules are operationalised. This section specifies what the employer sees, when, and under what conditions.

### 9.1 The bright-line commitments

Three commitments hold without exception:

**Commitment 1:** No POA data ever reaches the employer.
**Commitment 2:** No PDP data reaches the employer without explicit, time-bounded, purpose-bounded, revocable pharmacist consent.
**Commitment 3:** No PFA data reaches the employer at individual-pharmacist resolution without explicit consent. Aggregate views always available with appropriate contractual basis.

### 9.2 Default employer views

Without per-pharmacist consent, the employer sees aggregated network-level views:

- Network RIR distribution (anonymised across pharmacists)
- Network class-specific implementation rates
- Network context-assembly time distribution
- Network CPD compliance status (anonymised)
- Network credentialing status

These support the employer's legitimate workforce planning and contract retention needs without identifying individual pharmacists.

### 9.3 Per-pharmacist views with consent

With explicit pharmacist consent, the employer can see:

- Individual RIR (gated by 30-day delay)
- Individual class-specific rates
- Individual context-assembly time
- Individual recommendation appropriateness (rare; PDP class)

Consent is purpose-bounded — typically for performance review, contract retention, or peer development conversations.

### 9.4 What the employer never sees

- Reflective writing entries (POA)
- Per-recommendation appropriateness scores at individual pharmacist level (without consent)
- Restraint override patterns (POA)
- Per-GP framing patterns (PDP, never aggregated)
- Career portfolio narrative (POA)

### 9.5 The temporal-order enforcement

Even where employer aggregation is consented, the temporal-order commitment holds: the pharmacist sees their own data in their own dashboard before the same data is aggregated to the employer view. The employer view runs on a 30-day delay relative to the pharmacist's view by default.

### 9.6 Network-level vs facility-level aggregation

Employer aggregation occurs at the pharmacy practice network level by default. Facility-level aggregation (recommendations to a specific RACH) requires additional consent because facility-level work patterns can sometimes identify individual pharmacists in small deployments.

### 9.7 Surveillance pattern detection

The platform monitors for emergent surveillance patterns and flags them. Examples:

- Employer queries on individual pharmacists exceeding 95th percentile (suggests targeted scrutiny)
- Employer queries timed to coincide with employment review cycles
- Aggregation queries that effectively re-identify pharmacists in small subsets

When detected, the platform: notifies the pharmacist, flags the query in employer view (transparency about scrutiny), and reviews the contractual basis for the aggregation pattern.

---

## Part 10 — Cross-employer portability

The pharmacist's professional record persists across employer transitions. This is both an ethical commitment and a strategic feature supporting the bottom-up adoption motion.

### 10.1 The portability commitment

When a pharmacist changes employers, the platform:

- Preserves all POA, PDP, and the pharmacist's own PFA data
- Preserves career portfolio, CPD record, RPL evidence pack capability
- Migrates the pharmacist's account to the new employer context (or maintains free-tier account if new employer is not on platform)
- Retains workflow continuity for residents the pharmacist may continue working with under different arrangements

### 10.2 What does not transfer

- Active recommendations on residents at the prior employer (those remain with the prior employer's deployment)
- Workflow-operational data tied to the prior employer's contract
- Aggregated views the prior employer had access to (the prior employer's aggregated view includes the now-departed pharmacist's contribution as historical record)

### 10.3 The new employer's view

When a pharmacist arrives at a new employer with prior platform history:

- The new employer does not automatically see the prior employer's PDP or PFA data
- The pharmacist may consent to share prior career portfolio elements (commonly done for performance review or onboarding)
- The pharmacist's CPD compliance record is visible (WO class)
- The pharmacist's credentialing status is visible

### 10.4 Free-tier persistence

A pharmacist who leaves an enterprise-tier employer for an employer not on the platform reverts to free-tier status. They retain:

- Their personal career portfolio
- Their CPD record (capped to free-tier rate going forward)
- Their RPL evidence pack capability
- Their reflective writing entries

They lose access to:
- eNRMC integration (no employer deployment)
- Multi-pharmacist views (irrelevant)
- Facility-level dashboards (irrelevant)
- Recommendation count above 15 per month (free-tier limit)

This continuity is what makes the bottom-up adoption motion durable — the pharmacist's investment in the platform is portable.

### 10.5 Account closure

If a pharmacist closes their account entirely, the platform:

- Exports all pharmacist-controlled data to the pharmacist
- Retains audit-defensible records (AD class) per regulatory requirements
- Anonymises any aggregated data the pharmacist contributed to (their contribution remains in aggregate; their identity is removed)
- Provides written confirmation of closure with retention schedule

---

## Part 11 — File and code organisation

```
shared/v2_substrate/
├── permissions/
│   ├── visibility/
│   │   ├── class.go            # 5 visibility classes
│   │   ├── enforcer.go         # Query-layer enforcement
│   │   └── tests/
│   ├── consent/
│   │   ├── record.go           # Consent records
│   │   ├── purpose_bounds.go   # Purpose binding
│   │   └── tests/
│   ├── contestation/
│   │   ├── pathway.go          # Contestation workflow
│   │   ├── reviewer.go         # Independent reviewer integration
│   │   └── tests/
│   └── aggregation/
│       ├── temporal_gate.go    # 30-day delay enforcement
│       ├── network_agg.go      # Network-level aggregation
│       └── tests/

backend/services/pharmacist-self-visibility/
├── cmd/server/main.go
├── internal/
│   ├── api/
│   │   ├── grpc.go
│   │   └── http.go
│   ├── dashboards/
│   │   ├── worklist.go         # Surface 1
│   │   ├── recommendations.go  # Surface 2
│   │   ├── gp_relationships.go # Surface 3
│   │   ├── reasoning.go        # Surface 4
│   │   ├── cpd.go              # Surface 5
│   │   └── portfolio.go        # Surface 6
│   ├── kpis/
│   │   ├── rir.go              # Recommendation Implementation Rate
│   │   ├── class_specific.go   # Class-specific rates
│   │   ├── context_time.go     # Context-assembly time
│   │   ├── appropriateness.go  # Appropriateness scoring
│   │   └── trajectory.go       # Trajectory computation
│   ├── reflection/
│   │   ├── prompts.go          # Reflective prompt generation
│   │   ├── entries.go          # Entry storage and retrieval
│   │   └── tests/
│   ├── algorithmic_distinction/
│   │   ├── classifier.go       # Substrate-fact / suggestion / reflection / hybrid
│   │   ├── markers.go          # UI marker rendering
│   │   └── tests/
│   ├── exports/
│   │   ├── rpl_pack.go         # RPL evidence pack
│   │   ├── cpd_record.go       # AHPRA CPD record
│   │   └── portfolio.go        # Career portfolio
│   ├── portability/
│   │   ├── transition.go       # Employer transition handling
│   │   ├── account_closure.go
│   │   └── tests/
│   └── store/
│       ├── postgres/
│       │   ├── kpi.go
│       │   ├── reflection.go
│       │   ├── consent.go
│       │   ├── contestation.go
│       │   └── migrations/
│       └── redis/
└── README.md
```

---

## Part 12 — API contracts

```protobuf
service PharmacistSelfVisibilityService {
    // Get the pharmacist's own dashboard data for a specific surface
    rpc GetDashboard(DashboardRequest) returns (Dashboard);

    // Submit a reflective writing entry (POA)
    rpc SubmitReflection(ReflectionEntry) returns (Acknowledgment);

    // Get reflective prompts for the current period
    rpc GetReflectivePrompts(PharmacistID) returns (PromptList);

    // Generate RPL evidence pack
    rpc GenerateRPLPack(RPLRequest) returns (EvidencePack);

    // Generate AHPRA CPD record
    rpc GenerateCPDRecord(CPDRequest) returns (CPDRecord);

    // Submit a contestation
    rpc SubmitContestation(ContestationRequest) returns (Contestation);

    // Grant or revoke consent for data aggregation
    rpc ManageConsent(ConsentRequest) returns (ConsentRecord);

    // Initiate employer transition (cross-employer portability)
    rpc InitiateTransition(TransitionRequest) returns (TransitionPlan);
}
```

Visibility-class annotations on every API field — gRPC interceptors enforce class restrictions before any data return.

---

## Part 13 — Testing approach

Six test categories, including new categories specific to trust architecture.

### Category 1: Standard unit and integration tests

Coverage target ≥85% for unit tests; full integration tests for KPI computation pipeline.

### Category 2: Privacy boundary tests

Specific tests asserting that POA, PDP, and PFA visibility classes are correctly enforced at the data layer.

- POA data never returnable to non-pharmacist requester
- PDP data returnable to employer only with valid consent record
- PFA data returnable in aggregate only after time-gate
- Visibility class enforcement happens at query layer, not UI layer

### Category 3: Contestation pathway tests

End-to-end contestation workflow tests.

- Contestation submission creates record visible to pharmacist and (if metric was aggregated) employer
- Contested metric flagged in employer view until resolution
- Independent reviewer can be assigned
- Resolution outcomes correctly applied (correction, methodology revision, etc.)

### Category 4: Anti-surveillance tests

Specific tests asserting the surveillance prevention commitments.

- **Temporal-order test:** pharmacist always sees data in own dashboard before same data appears in employer view
- **Re-identification test:** small-subset aggregations checked for re-identification risk; flagged appropriately
- **Reflective-entry isolation test:** pattern detection algorithms cannot reach reflective writing data
- **Surveillance pattern detection:** queries matching surveillance heuristics generate alerts

### Category 5: Development-not-evaluation framing tests

UI text and pattern surfacing tests asserting the framing commitments.

- Above-average performers do not see "you're better than peers" framing
- Below-baseline performers see context-aware support, not metric judgment
- Reflective prompts are open-ended, not yes/no
- Pattern surfacing uses observation language, not judgment language

### Category 6: Portability tests

Cross-employer transition tests.

- All POA, PDP, and pharmacist's own PFA data preserved across transition
- Prior employer's aggregated views retain pharmacist's historical contribution as anonymous aggregate
- New employer does not automatically receive prior PDP data
- Free-tier reversion preserves career portfolio, CPD record, RPL pack capability

---

## Part 14 — Performance budgets

| Surface | p95 budget | Hard cap |
|---|---|---|
| Today's Worklist | 800ms | 2000ms |
| My Recommendations | 600ms | 1500ms |
| My GP Relationships | 1000ms | 2500ms |
| My Clinical Reasoning Patterns | 1500ms | 3000ms |
| My CPD Progression | 600ms | 1500ms |
| My Career Portfolio | 1000ms | 2500ms |
| RPL pack generation | 5000ms | 10000ms |
| CPD record generation | 2000ms | 5000ms |
| Visibility class enforcement | 50ms | 200ms |
| Contestation submission | 500ms | 1500ms |

Performance is more relaxed than craft engine because dashboards are less time-critical than recommendation flow. Visibility class enforcement is the tight budget — it's on every query and must not be the bottleneck.

---

## Part 15 — Implementation sequencing

Aligned with Phase 1 of the implementation plan (Weeks 5–10) plus extensions.

### Week 5–6: Visibility class architecture (foundation)

- Five-class enumeration
- ViewPermission engine integration
- Query-layer enforcement
- Algorithmic-vs-human distinction in EvidenceTrace
- Privacy boundary tests (Category 2)

### Week 7–8: Core dashboards

- Today's Worklist (Surface 1) — depends on craft engine recommendation lifecycle
- My Recommendations (Surface 2)
- My Clinical Reasoning Patterns (Surface 4) — basic trajectory views
- KPI computation pipeline (RIR, class-specific, context-assembly)

### Week 9: Trust architecture operationalisation

- Consent model implementation
- Contestation pathway
- Temporal-order enforcement
- Anti-surveillance tests (Category 4)
- Contestation pathway tests (Category 3)

### Week 10: Reflective and developmental layer

- My GP Relationships (Surface 3) — depends on per-GP framing learning from craft engine
- My CPD Progression (Surface 5)
- Reflective writing prompts and entry storage
- Above-average and below-baseline framing logic
- Development-not-evaluation framing tests (Category 5)

### Buffer Week 11: Portfolio and exports

- My Career Portfolio (Surface 6)
- RPL evidence pack generation
- CPD record export
- Cross-employer portability foundation
- Portability tests (Category 6)

### Estimated team

- 2 backend engineers (full-time across 7 weeks)
- 1 frontend engineer specifically for dashboard surfaces (parallel, 7 weeks)
- 1 clinical informatics lead (part-time, for prompt design and metric specification)
- 1 legal/ethics consultant (part-time, for consent model and contestation pathway review)
- External clinical ethics review at week 11 (3 days)

---

## Part 16 — Risks and mitigations

**Risk 1: Visibility class enforcement bypass.** A misconfigured query or UI fix could expose POA data. Mitigation: enforcement at query layer (not UI), comprehensive privacy boundary tests, code review checklist requires explicit class annotation on all data structures, quarterly external audit.

**Risk 2: Surveillance perception despite architecture.** Pharmacists may perceive the dashboard as surveillance regardless of architectural commitments if framing or UI choices feel evaluative. Mitigation: usability testing during pilot specifically on surveillance perception, ongoing pharmacist advisory input, willingness to revise framing based on feedback.

**Risk 3: Above-average performer complacency despite framing.** The trajectory and ceiling framings may not be sufficient to overcome the family physician finding. Mitigation: pilot data on engagement of above-average performers, additional development opportunities surfaced (peer teaching, advanced credential pathways), willingness to iterate.

**Risk 4: Contestation pathway under-utilised.** Pharmacists may not contest metrics they should contest, perceiving contestation as career-risky. Mitigation: contestation visibility commitments (employer cannot retaliate; contestation is a normal feature, not exceptional), example contestations surfaced in onboarding to normalise the pathway.

**Risk 5: Contestation pathway over-utilised.** Frivolous contestations could overwhelm review capacity. Mitigation: peer review for first-tier contestations, escalation only for substantial disputes, contestation-volume monitoring.

**Risk 6: Reflective writing entries breach POA through mistakes.** A bug or a misconfigured export could expose reflective entries. Mitigation: POA enforcement at query layer with separate audit trail; reflective entries excluded from any pattern detection; strict export controls.

**Risk 7: Cross-employer portability misused for credential laundering.** A pharmacist could attempt to claim a prior employer's high-quality work as their own portfolio. Mitigation: portfolio narratives require pharmacist authorship; AD-class records (unforgeable) underpin claims; employer-signed endorsements distinguish self-claims from validated work.

**Risk 8: Algorithmic-vs-human distinction ambiguous in edge cases.** Some observations are genuinely hybrid; classification may be unclear. Mitigation: default to "hybrid" classification; pharmacist confirmation required to convert hybrid to substrate-fact in their own view; audit trail records ambiguity.

**Risk 9: Free-tier reversion frustrates departing pharmacists.** A pharmacist losing eNRMC integration when changing employers may experience the platform negatively. Mitigation: clear communication at signup that integration is enterprise-tier-dependent; free-tier capability genuinely valuable on its own; smooth transition workflow.

**Risk 10: Legal exposure from algorithmic determination.** Despite architectural commitments, an employer using platform metrics in adverse employment decisions could create legal liability. Mitigation: contractual clauses in enterprise tier prohibiting sole-algorithmic-basis adverse decisions; contestation pathway with independent review; legal review before any KPI methodology changes.

**Risk 11: Aggregate views enabling re-identification.** In small deployments (e.g., a 3-pharmacist practice), aggregate metrics may effectively identify individuals. Mitigation: minimum-pharmacist-count threshold for aggregation (default 5); facility-level aggregation requires additional consent in small deployments.

**Risk 12: Reflective prompts interpreted as evaluation.** A poorly worded prompt could feel like critique. Mitigation: prompt library reviewed by clinical psychologist with pharmacy experience; pharmacist can dismiss any prompt; pilot feedback on prompt quality.

---

## Part 17 — Closing

Three observations as we close v1.0 of these guidelines.

**One:** The pharmacist self-visibility module is the trust architecture's first proving ground. Every commitment in v3.0 §9 — frame adapts content invariant, acceptance follows appropriateness, restraint as clinical answer, pharmacist autonomy preserved, self-visibility before aggregation, reviewability of platform itself, GP authority strengthened — is operationalised here in a form pharmacists experience daily. If these commitments are perceived as real by pharmacists in the pilot, the bottom-up adoption motion has integrity. If perceived as marketing language, the motion collapses.

**Two:** The empirical literature has a structural gap exactly where this module sits. Audit and feedback (Ivers 2025) is studied with supervisor-delivered feedback; individual self-monitoring tools that build professional confidence rather than triggering surveillance are under-studied. The TDF domains of professional identity and emotion are untargeted by prior interventions. This means the module is partly research as well as product — its design choices contribute to evidence about what works for clinician self-development, particularly for pharmacy. The pilot's qualitative methods (Pilot Design v1.0 §4.3) should specifically capture findings on these dimensions for academic publication.

**Three:** The portability commitment is the single most distinctive architectural choice. Most workplace performance systems treat the worker as the employer's data subject; the pharmacist self-visibility module treats the pharmacist as the data subject and the employer as a contractually-permitted observer. This is unusual in workplace performance management generally, and unusual in healthcare technology specifically. It is what makes the bottom-up adoption motion durable, what aligns the platform with the pharmacist's professional identity, and what creates the structural defensibility that competitor platforms deploying enterprise-first cannot replicate.

What this document does not yet specify, and what should be subsequent work:

- **Frontend implementation guidelines for each of the six dashboard surfaces** — UI patterns, interaction flows, visual design system aligned with the development-not-evaluation framing
- **Pharmacy employer view design** — what the employer sees when pharmacists have consented, with the aggregation rules and temporal-order commitments operationalised
- **PHN-style regional observability layer** — when PHARMA-Care framework matures and PHN as Buyer 5 emerges (per v3.0 §5)
- **Inter-pharmacist peer consultation features** — pharmacist-to-pharmacist learning loops, with appropriate privacy preservation

Each merits its own implementation guideline of comparable rigour. The pharmacist self-visibility module is the foundation those subsequent surfaces consume from.

— Claude
