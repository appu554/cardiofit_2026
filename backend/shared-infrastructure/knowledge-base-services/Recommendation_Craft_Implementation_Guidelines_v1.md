# Recommendation Craft Engine — Implementation Guidelines v1.0

**Date:** May 2026
**Service:** `kb-32-recommendation-craft`
**Implementation phase:** Phase 2 of Layer 2/3 plan (Weeks 8–16), with override-reason taxonomy, citation versioning, and negative-evidence patterns added in v1.0
**Builds on:** *Vaidshala v3.0 Product Proposal* §7 (Recommendation Craft Engine), *Layer 2 & 3 Implementation Plan* (7 May 2026) §2.1–2.6, and the citation-design design discussion (May 2026)

**Reading order:** Engineering and clinical informatics leads read Parts 1–10 (architecture and clinical specifications). Engineering implementers read Parts 11–14 (file structure, contracts, tests, sequencing). Clinical leads read Parts 5–7 (override taxonomy, citation versioning, negative evidence) for clinical sign-off. Product leads read Parts 1–3 and 8 for the framing architecture.

---

## Part 0 — Honest framing: what's established, what's distinctive

Before specifying the engine, name what is established practice (which we adopt) and what is genuinely distinctive (which we lead with). Branding established UX patterns with proprietary names backfires with sophisticated buyers — clinical informatics specialists and academic partners recognise the patterns.

### Established practice we implement well

**Progressive disclosure** — gradually revealing information rather than dumping it. Validated empirically (Muralidhar 2025 in *International Journal of Human-Computer Studies*) for AI clinical diagnosis systems. The four-layer Signal/Reasoning/Provenance/Audit architecture is a disciplined application of progressive disclosure to recommendation citation, not an invention.

**Hover-to-highlight citation** — the document we received called this "Traceability Anchors"; it's simply linking recommendation text to the underlying data points so the clinician can verify in place. Existing in production CDS systems for years. We implement it well; we don't claim it as new.

**Time-anchored citation display** — the document called this "Time-Linked Provenance"; it's just showing when each data point was captured alongside the citation. Standard in EHR audit trails. We implement it well; we don't claim it as new.

**Structured override reasons** — Wright/McCoy 2019 (*JAMIA*) established the empirical taxonomy from 10 US clinical sites. We adopt their categories as foundation and extend for ACOP context.

**Alert appropriateness framework** — McCoy 2012 (*JAMIA*) established the framework: false positive rate, override rate, non-adherence rate, response appropriateness rate. We adopt this directly.

### Genuinely distinctive elements we lead with

**Substrate-driven Signal generation.** The Signal layer in the recommendation packet isn't templated text; it's generated from queries against the EvidenceTrace graph and Clinical state machine baselines. "DBI increased 0.8→1.9 over 72h" is not authored — it's computed. Most CDS systems author signal text statically; ours is reactive to the substrate state.

**EvidenceTrace as bidirectional reasoning graph.** Citation usually means "show me where this came from" (forward query). The bidirectional graph supports the equally important inverse query: "given this outcome, what reasoning chain produced it?" That's the regulator-defensible audit substrate that distinguishes the platform from CDS tools that emit alerts and forget them.

**Multi-actor reasoning preservation across handoffs.** The recommendation packet preserves reasoning across the pharmacist → GP → nursing → resident handoff in a way the receiving party can verify and the audit can reconstruct. Most CDS systems collapse multi-actor reasoning at handoff (the GP gets a recommendation but loses the substrate trail). We don't.

**Negative-evidence citation as first-class.** Citing what was checked and *not found* is harder than citing what was found, but it's what makes deprescribing defensible. This is genuinely distinctive in CDS implementation.

**Citation versioning under source updates.** When ADG 2025 → ADG 2026 publishes, what happens to live recommendations? Most CDS systems silently update; we maintain effective-date semantics so the audit trail shows what evidence was current at fire time. This is regulatorily important and rarely implemented.

**Override-reason taxonomy with appropriateness pairing.** Override capture is established practice; pairing override reasons with appropriateness assessment (was this override clinically justified or does it indicate a rule problem?) and feeding into rule tuning is less common.

These five elements are what the platform leads with. Citation UI design (the four-layer architecture) is necessary scaffolding; reasoning continuity infrastructure is the moat.

---

## Part 1 — Design philosophy

Five principles shape every implementation decision in this document.

**Principle 1: Verification, not belief.** When a clinician acts on a platform recommendation, they are not trusting an AI; they are verifying a trajectory the system has organised for them. Every recommendation must support instant verification — no claim is asserted without queryable provenance. The clinician must be able to reach the underlying data point in two clicks or fewer.

**Principle 2: Peer-to-peer mirror.** The UI surfaces the recommendation in the order an expert pharmacist colleague would when discussing the case. Trajectory first ("DBI increased 0.8→1.9 over 72h"), then context ("falls history of three events in past 90 days"), then recommendation ("consider tapering quetiapine"). Test for any UI element: would an expert pharmacist surface this data in this order when explaining the case to a peer? If no, the UI is wrong.

**Principle 3: Frame adapts, content invariant.** The clinical recommendation is identical regardless of the receiving prescriber. Only the way it's communicated adapts. This distinction is auditable in EvidenceTrace at the data-structure level — clinical content and framing-adaptation are recorded separately.

**Principle 4: Restraint as a clinical answer.** Sometimes the right recommendation is no recommendation. The engine surfaces context arguing for non-intervention as well as for action. Maximising alerts is a CDS failure mode; well-calibrated CDS recommends restraint when restraint is right.

**Principle 5: Brevity with progressive depth.** Each surface has a specific information density. The clinician's first view is one screen with the headline; the second click reveals reasoning; the third click reveals citation; the fourth click is the full audit trail. Progressive disclosure is the discipline; the four-layer architecture is the implementation.

---

## Part 2 — The four-layer presentation architecture

The recommendation packet renders in four layers, with progressive depth. Each layer has explicit information density, performance budget, and UX commitments.

### Layer 1 — Signal

**Purpose:** Trigger the clinician's attention. State what changed, in clinical terms, with specific numbers.

**Format:** One sentence, ≤25 words. Present-tense or recent-perfect tense. Specific clinical instrument and quantified change.

**Example:** *"DBI increased 0.8→1.9 over 72h; sedation contributors now include quetiapine 25mg nocte (started 2026-04-15) and oxazepam PRN (escalated from 1×/week to 4×/week)."*

**Generation:** Substrate query against Clinical state machine baselines and recent MedicineUse events. Computed at recommendation-creation time, not authored. Re-computed on every state machine transition affecting the underlying data.

**Performance budget:** Render within 50ms of recommendation packet load.

**Anti-pattern:** Generic "patient at risk" statements. Replace with specific clinical instrument and quantified delta.

### Layer 2 — Reasoning

**Purpose:** Explain why the signal warrants attention. Connect substrate state to clinical implication.

**Format:** Two to four sentences, ≤100 words. Trajectory and context. Cite specific clinical data points with hover-to-reveal source attribution.

**Example:** *"Sedation increase coincides with two near-falls (16 April, 28 April) and family-reported daytime drowsiness. Anticholinergic burden rising on multiple sedating agents in a frail resident with eGFR 32 and recent UTI. Goals-of-care notes (12 March 2026) prioritise mobility preservation. Continued anticholinergic accumulation is likely to compound fall risk and may also worsen delirium recovery."*

**Generation:** Combination of substrate query + rule-fired reasoning. Each clinical claim in the text is anchored to a specific substrate node accessible via hover.

**Performance budget:** Render within 200ms of Layer 1 expansion.

**Anti-pattern:** Generic reasoning text. Each sentence must reference resident-specific substrate data.

### Layer 3 — Provenance

**Purpose:** Show evidence sources and version state. Support audit defensibility and clinical verification.

**Format:** Structured list of evidence anchors with source identification, effective date, and supersession status. Hover-to-reveal full source text where licensable.

**Example:**
```
Evidence sources:
• Australian Deprescribing Guideline 2025, recommendation 4.7
  (Effective 2025-09-01, current at recommendation fire time 2026-04-29)
• Beers Criteria 2023, antipsychotics in dementia (American Geriatrics Society)
  (Supplementary)
• RACGP Silver Book Part A: Deprescribing in older people
  (Effective 2024-01-15, current at recommendation fire time)
• Resident's documented goals-of-care: family meeting 2026-03-12 (AutumnCare 
  progress note ID 8847291)
```

**Generation:** Source Registry query at recommendation fire time. Sources are snapshot-versioned at the moment of fire so subsequent source updates do not retroactively modify the recommendation's evidence basis.

**Performance budget:** Render within 300ms of Layer 2 expansion.

**Versioning:** See Part 6.

### Layer 4 — Deep audit

**Purpose:** Full reasoning trace, suitable for clinical informatics review, regulator inquiry, or coronial investigation.

**Format:** Complete EvidenceTrace traversal. Every substrate node, every rule fire, every framing adaptation, every state transition, with timestamps and actor attribution.

**Generation:** EvidenceTrace bidirectional graph query, materialised view rendering.

**Performance budget:** Render within 1000ms of Layer 3 expansion (full audit query is heavier; acceptable latency higher).

**Visibility:** Default-hidden in clinical UI; surfaced on explicit click. Always available to platform's audit reviewers and to regulator queries under appropriate permissions.

### Cross-layer commitments

- Each citation is **clickable**; click resolves to the underlying substrate node
- Each substrate reference is **versioned**; the version state at recommendation fire time is preserved
- Each layer is **independently rendered**; failure in Layer 4 does not prevent Layer 1–3 display
- Layer transitions are **logged** in EvidenceTrace as user actions (clinician clicked to reveal Layer N at timestamp T) — this is itself audit-relevant data

---

## Part 3 — The recommendation lifecycle state machine

The Recommendation entity (Phase 0.1 of the implementation plan, Weeks 1–3) carries the recommendation through its lifecycle. The craft engine is responsible for the `detected` → `drafted` transition; downstream states are managed by the platform's communication and tracking layers.

### State definitions

| State | Description | Owning service |
|---|---|---|
| `detected` | Substrate query or rule fire produced a candidate recommendation. Not yet visible to pharmacist. | Substrate / Layer 3 rules |
| `drafted` | Craft engine assembled the packet (template, context, evidence, framing). Pharmacist sees in worklist. | kb-32 craft engine |
| `deferred` | Pharmacist reviewed but chose watchful wait. Forced review-date set. Re-surfaces if context changes or date elapses. | Recommendation lifecycle |
| `submitted` | Pharmacist sent to receiving prescriber (typically GP) via communication channel. | GP Communication Hub |
| `viewed` | Receiving prescriber opened the recommendation packet. | GP Communication Hub |
| `decided` | Receiving prescriber recorded a decision (accepted / rejected / clarification sought). | GP Communication Hub |
| `implemented` | Decision resulted in observable substrate change (chart edited, script issued, monitoring scheduled). | Substrate listener |
| `monitoring-active` | Implementation triggered monitoring obligations; observations expected. | Monitoring lifecycle |
| `outcome-recorded` | Monitoring observations completed, outcome assessable. | Monitoring lifecycle |
| `closed` | Recommendation lifecycle complete. Outcome documented. EvidenceTrace finalised. | Lifecycle service |
| `rejected` | Receiving prescriber declined. Override-reason captured. May re-surface after context change. | GP Communication Hub |
| `superseded` | A newer recommendation has replaced this one (e.g., refinement after labs landed). | Craft engine |
| `withdrawn` | Pharmacist withdrew before submission, or after submission with explanation. | Pharmacist action |

### Transition gates

- `detected` → `drafted`: requires craft engine to complete all stages (Part 4) successfully, including appropriateness check
- `drafted` → `deferred`: requires forced `review_date` to be set; pharmacist provides deferral reason (free-text + structured tags)
- `drafted` → `submitted`: requires authorisation evaluator (kb-30) to confirm receiving prescriber has scope to act on this recommendation type
- `decided` → `implemented`: substrate change must be observable within configurable window (default 7 days); otherwise `decided-not-implemented` substate
- `rejected` → captured override-reason: must select from structured taxonomy (Part 5)

### EvidenceTrace emissions

Every state transition emits an EvidenceTrace node with:
- `actor_class`: human / algorithmic / hybrid
- `actor_id`: pharmacist ID, prescriber ID, system ID
- `timestamp`
- `prior_state`, `new_state`
- `substrate_refs`: links to Resident, MedicineUse, Observation entities current at transition time
- `evidence_refs`: links to Source Registry entries cited
- `framing_adaptation_id` (if applicable; see Part 6 of v3.0 §9)

This is the audit-defensible substrate that supports regulator queries.

---

## Part 4 — Recommendation craft pipeline

The pipeline that takes a `detected` recommendation through to `drafted` runs in seven stages.

### Stage 1: Substrate query and clinical context assembly

**Input:** Recommendation candidate (from rule fire or substrate-driven detection) — minimally `resident_id`, `medication_use_id`, `recommendation_type`, `triggering_rule_id`.

**Output:** ClinicalContext object containing:
- `resident_demographics`: age, sex, frailty score (CFS / AKPS), care_intensity tag
- `current_labs`: most recent eGFR, electrolytes, glucose, FBE; computed delta from baseline
- `recent_events`: last 90 days of relevant events (falls, infections, hospitalisations, behavioural changes)
- `current_medications`: full active list with intent + target + stop_criteria from MedicineUse
- `relevant_history`: last 24 months of relevant clinical history
- `goals_of_care`: most recent care_intensity transition + family input where recorded
- `prior_interventions`: previous recommendations on same medication or related concerns
- `current_burden_scores`: DBI, ACB current and trajectory

**Performance budget:** ≤500ms substrate query end-to-end.

**File:** `kb-32-recommendation-craft/internal/context/assembler.go`

### Stage 2: Pattern detection and rule firing

**Input:** ClinicalContext + recommendation candidate.

**Output:** ReasoningChain object containing:
- `triggering_pattern`: the substrate pattern that fired (e.g., "anticholinergic burden trajectory rising in frail resident with recent fall")
- `applicable_rules`: list of CQL rule fires from kb-cql-runtime (HAPI engine)
- `restraint_signals`: substrate signals arguing against intervention (Part 10)
- `alternative_patterns`: other recommendation candidates that could fire from same context

**Performance budget:** ≤1000ms for full HAPI engine execution + restraint signal computation.

**File:** `kb-32-recommendation-craft/internal/reasoning/chain_builder.go`

### Stage 3: Recommendation generation

**Input:** ClinicalContext + ReasoningChain.

**Output:** Recommendation draft with structured sections — Issue, Clinical Context (Layer 2), Rationale, Evidence (Layer 3 references), Proposed Plan, Monitoring, Urgency.

**Generation pattern:**
- Issue: derived from `triggering_pattern` + specific clinical instruments (e.g., "DBI 1.9; rising")
- Context: pulled from ClinicalContext, narrative-rendered with Peer-to-Peer-Mirror ordering
- Rationale: connects substrate state to clinical implication; references applicable rules
- Evidence: Source Registry citations with effective dates
- Plan: dose change / cessation / monitoring directive in clinical-action language
- Monitoring: observation obligations with thresholds (links to Monitoring state machine when implemented)
- Urgency: red/amber/green per Part 4.4

**Performance budget:** ≤500ms for draft generation post-reasoning.

**File:** `kb-32-recommendation-craft/internal/generator/`

### Stage 4: Clinical appropriateness check

**Input:** Recommendation draft + full ClinicalContext + ReasoningChain.

**Output:** AppropriatenessAssessment with five-dimension scoring (Part 9).

**Behaviour:** If appropriateness score below threshold, recommendation is held in `detected` state with `appropriateness_concern` flag for clinical review. Threshold initially conservative (any single dimension below 3/5 holds for review); tunable based on pilot data.

**Performance budget:** ≤200ms.

**File:** `kb-32-recommendation-craft/internal/appropriateness/checker.go`

### Stage 5: Framing adaptation per audience

**Input:** Recommendation draft + receiving prescriber identity + per-GP framing observations (where available).

**Output:** Framed recommendation with the same clinical content, adapted communication style.

**Behaviour:**
- Clinical content (sections Issue, Rationale, Plan, Monitoring) is invariant
- Framing layer adapts: evidence source ordering, language tone, structural emphasis
- Adaptation is logged separately in EvidenceTrace per §7 of v3.0
- For first-encounter prescribers, default framing is used (no adaptation observable yet)

**Performance budget:** ≤200ms.

**File:** `kb-32-recommendation-craft/internal/framing/`

### Stage 6: Brevity formatting and progressive disclosure

**Input:** Framed recommendation.

**Output:** Render-ready packet with Layer 1/2/3/4 separation, length-budgeted.

**Behaviour:**
- Layer 1 enforced ≤25 words
- Layer 2 enforced ≤100 words
- Layer 3 structured list, no prose
- Layer 4 unbounded (full trace)
- Brevity violations are blocked (recommendation cannot transition to `drafted`)

**Performance budget:** ≤100ms.

**File:** `kb-32-recommendation-craft/internal/formatter/`

### Stage 7: Submission and tracking

**Input:** Render-ready packet.

**Output:** Recommendation entity transitioned to `drafted`, visible in pharmacist worklist.

**Behaviour:**
- Recommendation entity created in Postgres with full draft
- EvidenceTrace node emitted for `detected` → `drafted` transition
- Pharmacist worklist updated
- Monitoring lifecycle pre-staged (will activate on `implemented`)

**Performance budget:** ≤200ms.

**File:** `kb-32-recommendation-craft/internal/lifecycle/`

### End-to-end performance

Total pipeline: ≤2700ms for 95th percentile case. Worklist queue allows asynchronous craft engine execution; pharmacist sees recommendations as they're produced rather than waiting for batch.

---

## Part 5 — Override-reason taxonomy (NEW)

When a receiving prescriber rejects a recommendation, structured capture of *why* feeds three downstream uses: rule tuning, per-GP framing learning, and audit defensibility.

### Foundation: Wright/McCoy 2019

The Wright et al. 2019 *JAMIA* paper analysed 177 unique override reasons across 10 US clinical sites and consolidated them into 12 categories. Three categories accounted for 78% of all overrides: "will monitor or take precautions," "not clinically significant," and "benefit outweighs risk." We adopt the 12-category foundation and extend.

### The Vaidshala override-reason taxonomy

Twelve foundation categories (from Wright/McCoy 2019, adapted for Australian aged care context):

| Code | Category | Description |
|---|---|---|
| `WMP` | Will monitor or take precautions | Prescriber accepts the risk; will monitor; no medication change |
| `NCS` | Not clinically significant | Prescriber judges the recommended change unnecessary in this resident's context |
| `BOR` | Benefit outweighs risk | Prescriber judges current regimen's benefit greater than the recommendation's anticipated improvement |
| `ALR` | Already in plan | Change is already scheduled or planned |
| `TLR` | Tolerated long-term | Resident has tolerated current regimen for extended period; risk of change exceeds risk of continuation |
| `AOI` | Allergy or intolerance issue | Recommended alternative not suitable due to allergy or prior intolerance |
| `RWE` | Recent prescriber decision | A recent prescribing decision (often by specialist) takes precedence; defer to that decision |
| `IAI` | Inaccurate alert information | The recommendation is based on incorrect substrate data |
| `WDO` | Will discontinue another | Will discontinue a related medication instead of the one recommended |
| `NRA` | No reasonable alternative | Recommendation requires alternative not suitable for this resident |
| `IPA` | Inappropriate alert | Recommendation should not have fired in this clinical context |
| `OTH` | Other (free-text required) | Reason not captured by above categories |

### Vaidshala-specific extensions for aged care context

Eight additional categories specific to ACOP / aged care context:

| Code | Category | Description |
|---|---|---|
| `CGM` | Consent gating mismatch | Recommendation requires consent (e.g., restrictive practice) not yet obtained |
| `CIN` | Care intensity transition | Resident has transitioned to palliative or comfort care; recommendation no longer applies |
| `FNR` | Family not ready | Family decision-maker not yet ready for proposed change (deprescribing context) |
| `RNR` | Resident not ready | Resident expressed reluctance; shared decision-making in progress |
| `TXP` | Transition pending | Hospital admission, transfer, or discharge in progress; defer change |
| `SAD` | Stable on adjusted regimen | Recent dose adjustment in similar direction; allow stabilisation period |
| `MRP` | Multidisciplinary review pending | Case under MAC review; deferring individual change |
| `SDH` | Specialist deferred | Specialist (e.g., psychiatrist, geriatrician) review scheduled; defer to that review |

### Override-reason data structure

```go
// /shared/v2_substrate/models/override_reason.go
type OverrideReason struct {
    ID                 uuid.UUID
    RecommendationID   uuid.UUID
    PrescriberID       string  // HPI-I or local ID
    Category           string  // 3-letter code from taxonomy above
    Subcategory        *string // optional refinement
    FreeText           *string // required when Category == "OTH"; optional otherwise
    AppropriatenessFlag *string // see appropriateness pairing below
    CapturedAt         time.Time
    CapturedVia        string  // "structured_form" / "phone_recorded" / "free_text_extracted"
    SubstrateRefs      []SubstrateRef // links to clinical context at override time
}
```

### Appropriateness pairing (per McCoy 2012 framework)

Override capture alone is insufficient. The McCoy 2012 framework distinguishes appropriate overrides (clinically justified) from inappropriate overrides (suggesting alert problems or clinical disagreement). Vaidshala pairs each override with appropriateness assessment.

`AppropriatenessFlag` values:
- `appropriate` — override is clinically justified given the substrate state
- `appropriate_with_signal` — override is justified, but indicates a substrate signal we should learn from (e.g., "TLR — tolerated long-term" repeatedly across residents may indicate the underlying rule is too aggressive)
- `inappropriate_clinical` — override is clinically questionable; the recommendation likely should have been accepted
- `inappropriate_communication` — override may reflect framing failure rather than clinical disagreement
- `pending_review` — appropriateness not yet assessed

Appropriateness is assessed by:
- Automated scoring against substrate state at override time (e.g., if clinical context strongly supports the recommendation and override reason is "NCS", flag for review)
- Pharmacist sampled review of override episodes (per pilot evaluation methodology)
- Independent clinical reviewer for high-stakes recommendation classes (psychotropic, anticoagulant, renal dose adjustment)

### Override capture UX

The receiving prescriber sees a structured form alongside the recommendation:

```
┌─────────────────────────────────────────────────────┐
│ DECISION                                            │
│ ○ Accept and implement                              │
│ ○ Accept with modification (specify)                │
│ ○ Decline                                           │
│ ○ Need more information                             │
│                                                     │
│ If declining, please indicate why:                  │
│ ○ Will monitor or take precautions                  │
│ ○ Not clinically significant in this resident       │
│ ○ Benefit outweighs risk                            │
│ ○ Already in plan                                   │
│ ○ Tolerated long-term                               │
│ ○ Recent prescriber decision applies                │
│ ○ Specialist review scheduled                       │
│ ○ Other [free text required]                        │
│                                                     │
│ Optional: clinical note (any decision)              │
│ [ ___________________________________________ ]    │
└─────────────────────────────────────────────────────┘
```

For phone-mediated decisions (common in aged care), pharmacist captures override reason on prescriber's behalf with explicit attribution: *"per phone call with Dr Smith 2026-04-30 14:35, declined with reason: tolerated long-term."*

### Feedback into rule tuning

Override-reason data flows back into the platform per Wright/McCoy 2019 recommendation: "Allow the user to provide feedback directly to the team responsible for maintaining the local EHR alerting system and the drug interaction knowledge base, so the knowledge base can be continuously improved."

The feedback loop:

- Weekly aggregate report of override patterns per rule (which rules are most-overridden, with which reason)
- Monthly clinical review of high-override rules (>50% override rate triggers review)
- Quarterly rule-tuning decisions: tighten rule, loosen rule, add suppression condition, retire rule
- All decisions documented in rule retirement workflow (per Layer 3 v2 spec) with reasoning preserved

### Storage and analytics

```sql
-- Postgres migration: kb-32 override-reason store
CREATE TABLE recommendation_override_reasons (
    id UUID PRIMARY KEY,
    recommendation_id UUID NOT NULL REFERENCES recommendations(id),
    prescriber_id VARCHAR(64) NOT NULL,
    category VARCHAR(3) NOT NULL,
    subcategory TEXT,
    free_text TEXT,
    appropriateness_flag VARCHAR(32),
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    captured_via VARCHAR(32) NOT NULL,
    substrate_refs JSONB,
    
    -- Indexes for common queries
    INDEX idx_recommendation (recommendation_id),
    INDEX idx_prescriber (prescriber_id),
    INDEX idx_category (category),
    INDEX idx_captured_at (captured_at)
);

-- Materialised view for rule-tuning analytics
CREATE MATERIALIZED VIEW rule_override_patterns AS
SELECT
    r.triggering_rule_id,
    or_.category,
    or_.appropriateness_flag,
    COUNT(*) as override_count,
    COUNT(*) FILTER (WHERE or_.appropriateness_flag = 'inappropriate_clinical') as inappropriate_count,
    AVG(EXTRACT(EPOCH FROM (or_.captured_at - r.created_at))) as avg_decision_time_seconds
FROM recommendations r
JOIN recommendation_override_reasons or_ ON or_.recommendation_id = r.id
WHERE r.state = 'rejected'
GROUP BY r.triggering_rule_id, or_.category, or_.appropriateness_flag;
```

---

## Part 6 — Citation versioning architecture (NEW)

When the Australian Deprescribing Guideline 2025 is superseded by ADG 2026, what happens to recommendations previously citing ADG 2025? Most CDS systems silently update — a recommendation generated under one version of the evidence appears, on later inspection, to cite a version that didn't exist at fire time. This is regulatorily problematic and clinically misleading.

Vaidshala maintains effective-date semantics so the audit trail is internally consistent.

### The problem space

Three sub-problems:

**Sub-problem 1: Source supersession.** A guideline is updated. Recommendations citing the old version exist in various states (drafted, submitted, viewed, decided, implemented, monitoring-active, closed). What citation do new recommendations use? What do existing recommendations show?

**Sub-problem 2: Source amendment.** A guideline is corrected mid-version (e.g., dose calculation error in a recommendation table). Recommendations relying on the corrected calculation may be clinically wrong.

**Sub-problem 3: Source retraction.** A guideline is withdrawn entirely. Recommendations citing it have lost their evidence basis.

### The Source Registry pattern (extending Layer 1 v2)

Layer 1 v2 introduced the Source Registry with category, jurisdiction, effective period, authority tier, reproduction terms. We extend with version-specific operations.

```go
// /shared/v2_substrate/models/source_registry.go
type SourceVersion struct {
    SourceID         uuid.UUID
    VersionLabel     string    // e.g., "ADG 2025 v1.0", "ADG 2025 v1.1 (corrected 2025-11-30)"
    EffectiveFrom    time.Time
    EffectiveTo      *time.Time // nil if currently effective
    SupersededBy     *uuid.UUID // pointer to next version's SourceVersion
    SupersedeReason  *string    // "scheduled update" / "correction" / "retraction" / "withdrawn"
    ContentHash      string     // hash of source content; detects undeclared changes
    EmbeddedInRules  []string   // rule IDs depending on this version
    EvidenceContent  string     // cached extract of relevant content for fast citation render
}
```

### Recommendation-time citation snapshot

When a recommendation fires, the Source Registry resolves the *currently effective* version of each citable source and snapshots the citation onto the recommendation. The snapshot is immutable.

```go
type RecommendationCitation struct {
    SourceID            uuid.UUID
    SourceVersionID     uuid.UUID  // pinned at fire time
    VersionLabelAtFire  string     // human-readable
    EffectiveFromAtFire time.Time
    FireTimestamp       time.Time
    CitationContext     string     // which part of the source supports this recommendation
    SupersessionStatus  string     // "current" / "superseded" / "amended" / "retracted" — computed at render time
}
```

### Behaviour when source updates

When a SourceVersion is superseded, no existing recommendations are modified. Their `RecommendationCitation` records remain pinned to the version current at fire time. New recommendations fire against the new version.

The UI reflects the supersession status at render time. Three rendering modes:

**Active recommendation, source still current:**
```
Australian Deprescribing Guideline 2025 v1.0, recommendation 4.7
(Effective 2025-09-01, current)
```

**Active recommendation, source superseded between fire and view:**
```
Australian Deprescribing Guideline 2025 v1.0, recommendation 4.7
(Effective 2025-09-01, current at fire time 2026-04-29; 
superseded 2026-08-15 by ADG 2026 v1.0 — review recommended)
```

**Closed recommendation, historical view:**
```
Australian Deprescribing Guideline 2025 v1.0, recommendation 4.7
(Effective 2025-09-01, current at fire time 2026-04-29)
```

The "review recommended" prompt is important: it tells the clinician that newer evidence may modify the recommendation without retroactively rewriting history.

### Behaviour for high-impact source changes

Three categories of source change require active re-evaluation:

**Source amendment (correction):** All recommendations citing the amended version are flagged for clinical review. Worklist surfaces them. Pharmacist re-evaluates each; either confirms the recommendation is still valid (annotates), withdraws it, or supersedes it with a new recommendation citing the corrected version.

**Source retraction:** All recommendations citing the retracted source are surfaced as urgent for review. They cannot be acted upon by receiving prescribers without acknowledgment.

**Source supersession with material change:** The Source Registry may flag a supersession as "material change" (not just routine update). Recommendations citing the prior version are surfaced for review, similar to amendment workflow but lower urgency.

### Source supersession workflow

```go
// /shared/v2_substrate/source_registry/supersession_workflow.go
func SupersedeSource(prior SourceVersion, new SourceVersion, reason string) error {
    // 1. Mark prior version's EffectiveTo
    prior.EffectiveTo = &time.Now()
    prior.SupersededBy = &new.SourceID
    prior.SupersedeReason = &reason
    
    // 2. Identify affected recommendations
    affected := QueryRecommendationsByCitation(prior.SourceID, prior.VersionLabel)
    
    // 3. Tag each with supersession status
    for _, rec := range affected {
        if rec.State in [drafted, submitted, viewed, decided, implemented, monitoring-active] {
            rec.AddTag("source_superseded")
            if reason in ["correction", "retraction"] {
                rec.AddTag("urgent_review")
                EmitWorklistAlert(rec)
            }
        }
        // closed recommendations: tag historically, no worklist alert
    }
    
    // 4. Notify rule maintenance team
    NotifyRuleTeam(prior, new, len(affected))
    
    // 5. Audit log
    EmitEvidenceTrace(supersession_event)
    
    return nil
}
```

### Source content integrity

Each SourceVersion stores a `ContentHash`. Periodic verification (monthly) confirms the source content matches the hash. If mismatch detected:
- Source is flagged for review
- Active recommendations citing the source are flagged
- Rule maintenance team investigates whether the source publisher changed content silently

This catches the failure mode where a source publisher updates content without publishing a new version — the hash mismatch surfaces it.

### Storage

```sql
-- Postgres migration
CREATE TABLE source_versions (
    id UUID PRIMARY KEY,
    source_id UUID NOT NULL REFERENCES sources(id),
    version_label VARCHAR(64) NOT NULL,
    effective_from TIMESTAMPTZ NOT NULL,
    effective_to TIMESTAMPTZ,
    superseded_by UUID REFERENCES source_versions(id),
    supersede_reason TEXT,
    content_hash VARCHAR(64) NOT NULL,
    evidence_content TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    INDEX idx_source_id (source_id),
    INDEX idx_effective_period (effective_from, effective_to),
    INDEX idx_superseded_by (superseded_by)
);

CREATE TABLE recommendation_citations (
    id UUID PRIMARY KEY,
    recommendation_id UUID NOT NULL REFERENCES recommendations(id),
    source_version_id UUID NOT NULL REFERENCES source_versions(id),
    version_label_at_fire VARCHAR(64) NOT NULL,
    effective_from_at_fire TIMESTAMPTZ NOT NULL,
    fire_timestamp TIMESTAMPTZ NOT NULL,
    citation_context TEXT,
    
    INDEX idx_recommendation (recommendation_id),
    INDEX idx_source_version (source_version_id)
);
```

---

## Part 7 — Negative-evidence citation patterns (NEW)

A meaningful proportion of deprescribing recommendations fire because of *absent* findings, not present ones. "No documented indication for PPI"; "no recent reflux symptoms in progress notes"; "no GP-led indication review since 2024-08-15"; "no acute mental-state event in past 90 days." Citing what was checked and not found is harder than citing what was found, but it's what makes deprescribing recommendations defensible.

This is a genuinely distinctive area — most CDS implementations cite present findings well and absent findings poorly or not at all.

### Three classes of negative evidence

**Class 1: Absence of indication.** Medication is on the chart but no documented indication exists in the substrate. Common for legacy hospital starts (PPIs, statins, benzodiazepines).

Citation pattern: *"Indication for omeprazole 40mg daily not documented in eNRMC progress notes (last 24 months reviewed) or in handover documentation. No GP-led indication review identified since admission 2024-08-15."*

**Class 2: Absence of symptoms supporting continuation.** Medication targets a symptom; symptom not observed in recent monitoring period.

Citation pattern: *"Reflux symptoms not documented in nursing progress notes (last 90 days reviewed); no antacid PRN use recorded; resident reports no upper GI symptoms in last care plan review (2026-03-15)."*

**Class 3: Absence of monitoring or review.** Required monitoring or periodic review not performed within expected interval.

Citation pattern: *"Last lithium level recorded 2025-09-12 (208 days ago); aged care psychotropic review not documented in past 12 months; behavioural symptom assessment last completed 2025-11-04."*

### Substrate query patterns for absence

Querying for absence requires careful query construction — the substrate must support "I checked all of X and found no Y" rather than just "I found no Y."

**Pattern A: Bounded-window absence.**
```cql
// Find residents with PPI on chart and no documented reflux symptoms in past 90 days
define "PPIWithoutSymptoms":
  Resident R such that
    exists (R.MedicineUse mu where mu.medication.code in PPIs and mu.active = true) and
    not exists (
      R.Observation o
      where o.category = "GI symptom"
        and o.timestamp >= (Now() - 90 days)
    ) and
    QueryEvidence("nursing progress notes searched 90 days") and
    QueryEvidence("PRN antacid use record searched 90 days") and
    QueryEvidence("care plan symptom review searched 90 days")
```

The `QueryEvidence` calls are critical — they record what *was* searched, not just that nothing was found. This is what makes the negative-evidence citation defensible: the audit shows the substrate was queried for the symptom and the symptom was not present.

**Pattern B: Periodic-review absence.**
```cql
// Find lithium-treated residents whose last lithium level is older than threshold
define "LithiumMonitoringOverdue":
  Resident R such that
    exists (R.MedicineUse mu where mu.medication.code = "Lithium" and mu.active = true) and
    let lastLevel = MaxBy(R.Observation o where o.code = "Lithium level", o.timestamp) in
      lastLevel.timestamp < (Now() - 90 days)
      or lastLevel is null
```

**Pattern C: Indication-documentation absence.**
```cql
// Find chronic medications with no recorded indication
define "ChronicMedWithoutIndication":
  Resident R such that
    exists (R.MedicineUse mu where 
      mu.active = true 
      and mu.duration > 12 months
      and mu.intent.indication is null
    ) and
    not exists (
      R.ProgressNote pn 
      where pn.medication_review_documented = true
        and pn.timestamp > (mu.start_date)
    )
```

### UI presentation of "what was checked"

The Layer 2/3 rendering for negative-evidence recommendations explicitly shows the searches that were performed and their results. This is what makes the recommendation defensible.

**Example UI:**

```
┌─────────────────────────────────────────────────────────────────┐
│ RECOMMENDATION: Consider deprescribing omeprazole 40mg daily    │
│                                                                  │
│ TRIGGERING FACTORS — what was checked:                           │
│                                                                  │
│ ✓ eNRMC indication field for omeprazole: empty                  │
│ ✓ Progress notes searched (90 days): no reflux symptoms found   │
│ ✓ Progress notes searched (90 days): no PRN antacid use found   │
│ ✓ Care plan symptom review (2026-03-15): no upper GI symptoms   │
│ ✓ GP indication review search (24 months): not documented       │
│ ✓ Hospital discharge summary 2024-08-15: PPI started during     │
│   admission for "stress ulcer prophylaxis"                      │
│                                                                  │
│ Each item is clickable to verify the underlying record.          │
└─────────────────────────────────────────────────────────────────┘
```

The defensibility comes from the explicit "what was checked" — a clinician reviewing the recommendation can verify the platform did the work, and a regulator querying the audit trail sees the same evidence base.

### Performance considerations for negative evidence

Negative-evidence queries are expensive because they require exhaustive searches across substrate categories. Optimisations:

- Pre-computed materialised views for common absence patterns (no recent symptoms, no recent monitoring)
- Substrate-level "evidence-checked" flags emitted during normal observation ingestion (so we know what was looked for)
- Bounded query windows (90 days, 12 months) tunable by recommendation type

Performance budget for negative-evidence recommendation generation: ≤2000ms (higher than positive-evidence; acceptable given the recommendation classes typically affected).

### Storage

```sql
CREATE TABLE evidence_checks (
    id UUID PRIMARY KEY,
    recommendation_id UUID NOT NULL REFERENCES recommendations(id),
    check_type VARCHAR(32) NOT NULL,  -- "absence" / "presence"
    check_target TEXT NOT NULL,       -- "reflux symptoms" / "lithium level"
    query_window_start TIMESTAMPTZ,
    query_window_end TIMESTAMPTZ,
    substrate_categories_searched JSONB,  -- ["progress_notes", "care_plan", "prn_records"]
    result VARCHAR(16) NOT NULL,      -- "found" / "not_found" / "stale"
    result_details JSONB,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    INDEX idx_recommendation (recommendation_id),
    INDEX idx_check_type (check_type)
);
```

This table is the substrate of the negative-evidence citation UI. Every absence claim in a recommendation is backed by an `evidence_checks` row showing what was searched and when.

---

## Part 8 — Per-GP framing learning with ethical limits

Per v3.0 §7 and §9, the platform observes acceptance patterns per receiving prescriber and offers framing suggestions to the pharmacist preparing recommendations. Implementation must respect the ethical guardrails.

### What the platform observes

For each receiving prescriber over a rolling 12-month window:

- Recommendation type acceptance pattern (STOP / MONITOR / DOSE-CHANGE / ADD)
- Evidence source resonance (which sources, when cited, correlate with acceptance)
- Communication channel acceptance (structured email vs phone vs embedded eNRMC note)
- Framing-style acceptance (clinical-urgency-first vs patient-context-first vs monitoring-burden-first)
- Time-of-day / day-of-week acceptance variation

These are stored against the prescriber's identity but exposed only as gentle suggestions to the pharmacist preparing a recommendation.

### What the platform does not surface

- Per-prescriber acceptance percentages (e.g., "Dr Smith has 42% acceptance rate")
- Comparative rankings of prescribers
- Aggregated "this prescriber is difficult" labels
- Any data flowing to the pharmacy employer about specific prescriber behaviour

### Surfacing pattern

When the pharmacist is preparing a recommendation, the framing learning module renders a side-panel suggestion:

```
┌──────────────────────────────────────────────────────────┐
│ FRAMING NOTE                                              │
│                                                           │
│ Recommendations to Dr Smith have landed better when:     │
│ • Monitoring plan is included up front                    │
│ • RACGP source is cited rather than international         │
│ • Phone follow-up is offered for urgent items             │
│                                                           │
│ This is observation, not direction. Use your judgment.    │
└──────────────────────────────────────────────────────────┘
```

Note the explicit "this is observation, not direction" — the platform does not direct; it observes and offers.

### Toxicity guardrails

Three rules governing the framing learning module:

**Rule 1: Pharmacist-only visibility.** Framing observations are surfaced only in the pharmacist's view. They never flow to pharmacy employer dashboards, RACH dashboards, or chain analytics.

**Rule 2: Aggregate-only across pharmacists.** Patterns observed across many pharmacists for a given prescriber are aggregated (no per-pharmacist attribution). No single pharmacist's interactions are visible to the platform's framing learner; only aggregate patterns inform suggestions.

**Rule 3: Prescriber opt-out.** Receiving prescribers are notified that the platform observes acceptance patterns and offers framing suggestions to pharmacists. They may opt out of pattern tracking entirely. Opt-out leaves the platform with default-framing only for that prescriber.

### Implementation

```go
// /kb-32-recommendation-craft/internal/framing/learner.go
type FramingObservation struct {
    PrescriberID         string
    RecommendationType   string
    EvidenceSourcesCited []string
    Channel              string
    FramingStyle         string
    OutcomeAccepted      bool
    OutcomeWindow        time.Duration
    ObservedAt           time.Time
}

func (l *FramingLearner) ObserveOutcome(rec Recommendation, decision Decision) {
    // Only record if prescriber has not opted out
    if l.prescriberHasOptedOut(rec.PrescriberID) {
        return
    }
    // Record observation
    obs := FramingObservation{...}
    l.store.Insert(obs)
    // Re-compute aggregate patterns (async)
    l.scheduleAggregateRecomputation(rec.PrescriberID)
}

func (l *FramingLearner) GetSuggestions(prescriberID string, context CraftContext) FramingSuggestions {
    if l.prescriberHasOptedOut(prescriberID) {
        return DefaultFramingSuggestions(context)
    }
    if l.observationCount(prescriberID) < MIN_OBSERVATIONS_THRESHOLD {
        return DefaultFramingSuggestions(context)
    }
    return l.computePersonalizedSuggestions(prescriberID, context)
}
```

`MIN_OBSERVATIONS_THRESHOLD` is 30 — fewer than 30 observed interactions yields default framing. This prevents premature pattern fitting and protects prescribers from being characterised on small samples.

---

## Part 9 — Clinical appropriateness check

Per v3.0 §9 Principle 2, recommendation acceptance must be paired with appropriateness. The appropriateness check runs at Stage 4 of the craft pipeline (Part 4) and produces an AppropriatenessAssessment.

### Five-dimension rubric

For each recommendation, the appropriateness check scores five dimensions on a 1–5 scale:

| Dimension | Question | 1 (low) | 5 (high) |
|---|---|---|---|
| Clinical warrant | Is this recommendation clinically warranted given current substrate state? | Marginal indication | Strong, well-evidenced indication |
| Evidence solidity | Is the evidence base solid for this resident's profile? | Weak or extrapolated | Strong, directly applicable |
| Alternatives considered | Have alternative interventions been considered? | Single option offered | Multiple options weighed |
| Restraint considered | Has restraint been considered? | No restraint signals checked | Restraint signals explicit |
| Goals-of-care alignment | Does the recommendation align with the resident's goals? | Misaligned | Strongly aligned |

### Threshold behaviour

Recommendations scoring ≤2 on any single dimension are **held in `detected` state** with `appropriateness_concern` flag. They surface in the pharmacist's worklist for review before they can transition to `drafted`. The pharmacist may:
- Override the concern with documented reasoning (logged in EvidenceTrace as algorithmic-vs-human distinction)
- Withdraw the recommendation
- Modify the recommendation to address the concern

Thresholds are tunable based on pilot data. Initial conservative thresholds expected to relax as the platform's clinical fidelity demonstrates over the pilot period.

### Pairing with acceptance metrics

Appropriateness scores are recorded alongside acceptance outcomes. The pairing produces three patterns of concern:

**Pattern 1: High acceptance, low appropriateness.** Recommendations are landing despite weak clinical warrant. This is the "persuasive framing of marginal recommendations" failure mode (v3.0 Risk 15). The platform flags rules where this pattern exceeds 10% of acceptances for clinical review.

**Pattern 2: Low acceptance, high appropriateness.** Recommendations are clinically sound but consistently rejected. This is either a framing problem (improve craft) or a prescriber problem (relationship work needed). The platform surfaces these to the pharmacist's self-view but does not flag them as employer-relevant.

**Pattern 3: Suppression.** Recommendation-type frequency drops without corresponding context change. Pharmacists may be suppressing clinically necessary recommendations because they're predicted to be rejected (v3.0 Risk 13). Detected by:
- Comparing pharmacist's recommendation rate to peer ACOPs at similar facilities (anonymous peer comparison)
- Detecting whether triggering substrate states are still firing rules (rules fire but recommendations don't draft = suppression)

Detected suppression patterns are surfaced to the pharmacist's self-view first (prompt for reflection), and to the pharmacy employer only if persistent and severe (rule of three: surfaced after three months of pattern, only after pharmacist has had two opportunities to address).

### Implementation

```go
// /kb-32-recommendation-craft/internal/appropriateness/checker.go
type AppropriatenessAssessment struct {
    RecommendationID         uuid.UUID
    ClinicalWarrantScore     int  // 1-5
    EvidenceSoliditScore     int
    AlternativesScore        int
    RestraintScore           int
    GoalsAlignmentScore      int
    OverallFlag              string  // "ok" / "concern" / "block"
    HoldReasons              []string
    AssessedAt               time.Time
    AssessmentMethod         string  // "automated" / "human_review" / "hybrid"
}

func (c *AppropriatenessChecker) Assess(rec Recommendation, ctx ClinicalContext, chain ReasoningChain) AppropriatenessAssessment {
    assessment := AppropriatenessAssessment{...}
    
    assessment.ClinicalWarrantScore = c.scoreClinicalWarrant(rec, ctx, chain)
    assessment.EvidenceSoliditScore = c.scoreEvidenceSolidity(rec, ctx)
    assessment.AlternativesScore = c.scoreAlternatives(rec, chain)
    assessment.RestraintScore = c.scoreRestraint(rec, chain)
    assessment.GoalsAlignmentScore = c.scoreGoalsAlignment(rec, ctx)
    
    minScore := min(assessment.ClinicalWarrantScore, ...)
    if minScore <= 2 {
        assessment.OverallFlag = "block"
        assessment.HoldReasons = c.identifyHoldReasons(assessment)
    } else if minScore <= 3 {
        assessment.OverallFlag = "concern"
    } else {
        assessment.OverallFlag = "ok"
    }
    
    return assessment
}
```

---

## Part 10 — Restraint signals

Sometimes the right clinical answer is to *not* recommend. The restraint module surfaces context arguing for non-intervention.

### Substrate signals for restraint

The restraint module queries the substrate for nine signal types:

| Signal | Substrate query |
|---|---|
| Care intensity transition recent | care_intensity changed in past 30 days; new intensity = palliative or comfort care |
| Recent dose adjustment | MedicineUse for related medication adjusted in past 14 days |
| Family processing decline | progress notes contain family-distress markers in past 30 days |
| Resident not ready for change | care plan reflects shared-decision-making in progress |
| Approaching end of life | clinical signals of late-stage frailty; AKPS or CFS recent measurement supports |
| Hospital admission imminent | recent acute-event signals, admission likely in coming week |
| Specialist review scheduled | calendar references show specialist appointment in coming 30 days |
| Stable on long-term regimen | medication tolerated >2 years with no documented adverse signals |
| Multidisciplinary review pending | MAC agenda contains this resident in coming meeting |

### Surfacing restraint alongside action

When a restraint signal is present and a recommendation would otherwise fire, the craft engine surfaces both:

```
┌─────────────────────────────────────────────────────────────┐
│ RECOMMENDATION (action)                                      │
│ Consider tapering quetiapine 25mg nocte                     │
│ [full Layer 1/2/3 below]                                    │
│                                                              │
│ ─────────────────────────────────────────                   │
│                                                              │
│ RESTRAINT SIGNAL (consider deferring)                        │
│ Care intensity transitioned to "comfort care" 12 days ago.  │
│ Family meeting 8 March documented resident's stable comfort  │
│ on current regimen as a goal. Consider deferring this        │
│ recommendation; review at next MAC if regimen changes.       │
│                                                              │
│ ☐ Proceed with recommendation (override restraint, document │
│    reason)                                                   │
│ ☐ Defer to scheduled review date [select]                   │
│ ☐ Withdraw recommendation entirely                          │
└─────────────────────────────────────────────────────────────┘
```

The pharmacist makes the clinical decision. The platform surfaces both possibilities and records the choice with reasoning.

### Restraint override capture

When the pharmacist proceeds despite a restraint signal, the override reason is captured (similar structure to GP override, but pharmacist-side):

- Why was restraint judged inappropriate in this case?
- What clinical context overrides the restraint signal?

This data, like GP override-reasons, feeds into rule tuning. If a restraint signal is consistently overridden by pharmacists with similar reasoning, the signal may be too sensitive.

### Implementation

```go
// /kb-32-recommendation-craft/internal/restraint/signaler.go
type RestraintSignal struct {
    SignalType        string
    Severity          int  // 1-5
    SubstrateRef      SubstrateRef
    Description       string
    DefaultAction     string  // "defer" / "review" / "withdraw"
    DeferralPeriod    *time.Duration
    DeferralReviewAt  *time.Time
}

func (r *RestraintSignaler) DetectSignals(ctx ClinicalContext) []RestraintSignal {
    signals := []RestraintSignal{}
    for _, detector := range r.detectors {
        if signal := detector.Check(ctx); signal != nil {
            signals = append(signals, *signal)
        }
    }
    return signals
}
```

---

## Part 11 — File structure and code organisation

```
backend/services/kb-32-recommendation-craft/
├── cmd/
│   └── server/
│       └── main.go              # Service entry point
├── internal/
│   ├── api/
│   │   ├── grpc.go              # gRPC API for substrate consumers
│   │   ├── http.go              # HTTP API for UI
│   │   └── auth.go              # Authentication middleware
│   ├── context/
│   │   ├── assembler.go         # Stage 1: clinical context assembly
│   │   └── tests/
│   ├── reasoning/
│   │   ├── chain_builder.go     # Stage 2: pattern detection
│   │   ├── hapi_client.go       # CQL runtime integration
│   │   └── tests/
│   ├── generator/
│   │   ├── recommendation.go    # Stage 3: recommendation generation
│   │   ├── template.go          # Structured template enforcement
│   │   └── tests/
│   ├── appropriateness/
│   │   ├── checker.go           # Stage 4: clinical appropriateness check
│   │   ├── scorers/             # Five-dimension scoring
│   │   └── tests/
│   ├── framing/
│   │   ├── adapter.go           # Stage 5: per-audience framing
│   │   ├── learner.go           # Per-GP framing learning
│   │   ├── observer.go          # Outcome observation
│   │   └── tests/
│   ├── formatter/
│   │   ├── formatter.go         # Stage 6: brevity + progressive disclosure
│   │   ├── layers.go            # Four-layer rendering
│   │   └── tests/
│   ├── lifecycle/
│   │   ├── transitions.go       # Stage 7: state machine transitions
│   │   └── tests/
│   ├── citations/
│   │   ├── source_registry.go   # Source Registry client
│   │   ├── versioning.go        # Citation versioning logic
│   │   ├── snapshot.go          # Recommendation-time citation snapshot
│   │   └── tests/
│   ├── overrides/
│   │   ├── taxonomy.go          # 20-category override taxonomy
│   │   ├── capture.go           # Override capture API
│   │   ├── analytics.go         # Override pattern analytics
│   │   └── tests/
│   ├── negative_evidence/
│   │   ├── checker.go           # Negative-evidence query patterns
│   │   ├── citation.go          # "What was checked" UI rendering
│   │   ├── workflow.go          # Pre-computed materialised views
│   │   └── tests/
│   ├── restraint/
│   │   ├── signaler.go          # Restraint signal detection
│   │   ├── detectors/           # Per-signal-type detectors
│   │   └── tests/
│   └── store/
│       ├── postgres/            # Postgres repositories
│       │   ├── recommendation.go
│       │   ├── citation.go
│       │   ├── override.go
│       │   ├── evidence_check.go
│       │   └── migrations/
│       └── redis/               # Cache layer
│           └── cache.go
├── api/
│   ├── proto/                   # gRPC schemas
│   └── openapi/                 # HTTP schemas
├── deployments/
│   ├── docker-compose.yml
│   └── kubernetes/
├── scripts/
│   ├── seed_taxonomy.sh         # Seed override taxonomy
│   └── verify_source_hashes.sh  # Source content integrity check
└── README.md
```

---

## Part 12 — API contracts

### gRPC service

```protobuf
service RecommendationCraftService {
    // Generate a recommendation packet from a substrate-detected candidate
    rpc GenerateRecommendation(GenerateRequest) returns (Recommendation);
    
    // Re-render a recommendation for a specific audience (per-GP framing)
    rpc FramedForAudience(FramingRequest) returns (FramedRecommendation);
    
    // Capture an override-reason for a rejected recommendation
    rpc CaptureOverride(OverrideRequest) returns (OverrideAcknowledgment);
    
    // Get the appropriateness assessment for a recommendation
    rpc GetAppropriateness(uuid.UUID) returns (AppropriatenessAssessment);
    
    // Re-evaluate recommendations affected by source supersession
    rpc ReevaluateOnSourceUpdate(SourceUpdateNotification) returns (ReevaluationReport);
}
```

### HTTP endpoints (selected)

```
GET  /api/v1/recommendations/:id                # Full recommendation packet
GET  /api/v1/recommendations/:id/layer/:n       # Specific layer (1-4)
POST /api/v1/recommendations/:id/decision       # Submit decision + override
GET  /api/v1/recommendations/:id/citations      # All citations with version state
POST /api/v1/recommendations/:id/defer          # Defer with review date
GET  /api/v1/pharmacists/:id/framing-suggestions/:prescriber_id  # Pharmacist's view of framing patterns
```

---

## Part 13 — Testing approach

Five test categories, each with explicit coverage targets.

### Category 1: Unit tests

Standard unit testing of all internal packages. Coverage target ≥85%.

### Category 2: Integration tests

End-to-end pipeline tests with real Postgres + Redis + HAPI runtime stub.

- Sunday-night-fall scenario (DP-31 from Layer 2 walkthrough): fall observation lands → substrate detects medication-related risk → craft engine generates recommendation → all four layers render correctly → citation versioning correct → audit trail intact

### Category 3: Clinical safety tests

Specific test cases that must pass before production deployment.

- **Frame-vs-content invariance:** for any recommendation, framing for prescriber A and prescriber B produce identical clinical content (Sections Issue, Rationale, Plan, Monitoring); only language adapts. Asserted via content_hash equality.
- **Appropriateness blocking:** recommendations scoring ≤2 on any dimension cannot transition to `drafted` without explicit pharmacist override.
- **Anti-suppression test:** if a CQL rule fires consistently but recommendations are not drafted, anti-suppression detector flags within 30 days.
- **Negative-evidence completeness:** every negative-evidence claim in a recommendation has a backing `evidence_checks` row with documented substrate categories searched.
- **Citation versioning correctness:** recommendation generated under SourceVersion v1.0 displays correct version state when SourceVersion v1.1 supersedes v1.0; previous recommendation remains pinned to v1.0.

### Category 4: Metric integrity tests

Tests validating the appropriateness pairing.

- High-acceptance-low-appropriateness pattern detection: simulate 100 recommendations, 80 accepted, 30% with low appropriateness; assert pattern is flagged.
- Suppression pattern detection: simulate 50 rule-fires with only 20 reaching `drafted`; assert suppression flagged.

### Category 5: Performance tests

Latency assertions for each pipeline stage and overall.

| Stage | p95 budget | Hard cap |
|---|---|---|
| Stage 1 — Context assembly | 500ms | 1000ms |
| Stage 2 — Reasoning | 1000ms | 2000ms |
| Stage 3 — Generation | 500ms | 1000ms |
| Stage 4 — Appropriateness | 200ms | 500ms |
| Stage 5 — Framing | 200ms | 500ms |
| Stage 6 — Formatting | 100ms | 300ms |
| Stage 7 — Lifecycle | 200ms | 500ms |
| End-to-end | 2700ms | 5000ms |
| Layer 1 render | 50ms | 100ms |
| Layer 2 render | 200ms | 400ms |
| Layer 3 render | 300ms | 600ms |
| Layer 4 render | 1000ms | 2000ms |
| Negative-evidence query | 2000ms | 4000ms |

Hard caps enforced; pipelines exceeding hard cap return error and flag for performance review.

---

## Part 14 — Implementation sequencing

Aligned with Phase 2 of the implementation plan (Weeks 8–16) plus extensions for v1.0 additions.

### Week 8–9: Stage 1 + Stage 6 (foundation)

- ClinicalContext assembler
- Brevity formatter + four-layer rendering scaffold
- Postgres migrations for recommendation entity (depends on Phase 0.1 — Recommendation entity)

### Week 10–11: Stage 2 + Stage 7

- Reasoning chain builder + HAPI client
- Lifecycle transitions
- Recommendation worklist surfacing

### Week 12–13: Stage 3 + Stage 5 default framing

- Recommendation generator
- Default framing adapter (no per-GP learning yet)
- Initial integration testing

### Week 14: Citation versioning (NEW — extension to plan)

- Source Registry version-aware queries
- RecommendationCitation snapshot logic
- Source supersession workflow
- UI rendering of version state

### Week 15: Override taxonomy + Appropriateness check (NEW — extensions)

- Override taxonomy seeding (20 categories)
- Override capture API + UI
- Appropriateness checker (Stage 4)
- Metric integrity tests

### Week 16: Negative-evidence + Restraint + Per-GP framing (NEW — extensions)

- Negative-evidence query patterns
- Evidence-checks substrate
- Restraint signal detector
- Per-GP framing learner (with ethical guardrails)
- Toxicity guardrail tests

### Buffer week 17: Integration testing + clinical safety review

- Sunday-night-fall end-to-end test
- Frame-vs-content invariance assertion
- Anti-suppression detector validation
- Clinical safety review with senior ACOP-credentialed pharmacist (external)

### Estimated team

- 2 backend engineers (full-time across 9 weeks)
- 1 backend engineer specifically for citation versioning + Source Registry integration (4 weeks)
- 1 clinical informatics lead (part-time across 9 weeks for clinical fidelity review)
- 1 frontend engineer (parallel; not in this plan but required for Layer 1–4 rendering)
- External clinical safety reviewer (3 days at week 17)

---

## Part 15 — Risks and mitigations

**Risk 1: Stage 5 (per-GP framing) develops without sufficient observation data.** Mitigation: MIN_OBSERVATIONS_THRESHOLD=30 prevents premature pattern fitting. First 6 months of pilot will have largely default framing while observations accumulate.

**Risk 2: Source Registry supersession events are not captured promptly.** Mitigation: weekly automated source-content-hash verification + monthly manual review of major sources. Subscription to publisher update feeds where available.

**Risk 3: Negative-evidence queries are too slow at scale.** Mitigation: pre-computed materialised views + bounded query windows + acceptance of 2-second budget for negative-evidence cases. If still too slow, async generation for negative-evidence recommendations (acceptable given clinical context).

**Risk 4: Override taxonomy categories are insufficient for ACOP context.** Mitigation: pilot will surface gaps; "OTH" category captures unstructured reasons; quarterly taxonomy review during pilot adds categories as needed.

**Risk 5: Appropriateness blocking creates pharmacist friction.** Mitigation: initial conservative thresholds; clear override pathway with documented reasoning; thresholds tuned based on first 90 days of pilot data.

**Risk 6: Persuasive framing carrying marginal recommendations to acceptance.** Mitigation: appropriateness pairing detector (Pattern 1 in Part 9). Monthly clinical review of high-acceptance-low-appropriateness cases.

**Risk 7: Restraint signals over-fired (every recommendation has a restraint reason).** Mitigation: severity scoring; only severity ≥3 surfaces in UI; pharmacist override of low-severity signals is silent (no required reasoning).

**Risk 8: Per-GP framing learning being misperceived as surveillance by GP college (RACGP).** Mitigation: prescriber opt-out; aggregate-only patterns; transparent communication via Move 3 (RACGP engagement); platform's tone collaborative not analytical.

**Risk 9: Citation versioning UI confuses clinicians.** Mitigation: only surface version state when supersession is material; routine updates rendered as "current" without distraction; usability testing during pilot.

**Risk 10: Evidence-checks table grows unboundedly.** Mitigation: archival policy after 5 years for closed recommendations; hot-warm-cold storage tiering.

---

## Part 16 — Closing

Three observations as we close v1.0 of these guidelines.

**One:** The recommendation craft engine is where the platform's reasoning continuity infrastructure becomes visible to clinicians. Most CDS systems treat recommendation generation as the product; for Vaidshala, recommendation generation is the surface above the substrate. The substrate's bidirectional EvidenceTrace is what makes the four-layer presentation defensible; the four-layer presentation is what makes the substrate clinically usable. Neither works without the other.

**Two:** The three v1.0 extensions — override taxonomy, citation versioning, negative-evidence patterns — are not novel CDS features individually. Override capture exists in production EHRs (though rarely with structured taxonomy and appropriateness pairing). Citation versioning exists in research-grade systems (rarely in commercial CDS). Negative-evidence citation exists in academic deprescribing literature (rarely as a first-class platform feature). What's distinctive is implementing all three with substrate-driven generation, multi-actor reasoning preservation, and ethical guardrails as architectural commitments rather than retrofits. The platform's defensibility comes from this disciplined integration, not from individual feature novelty.

**Three:** The frame-vs-content principle (v3.0 §9 Principle 1, this document Part 4 Stage 5) is the single most important clinical-safety commitment in the engine. Audit-defensible separation of clinical content from framing adaptation is what prevents the platform from being accused of varying clinical advice by audience — a charge that would catastrophically damage the platform's regulatory standing. Implementation must preserve this separation at the data-structure level, not just in policy. The content_hash invariance test (Part 13 Category 3) is the line of defence; it must pass in every release.

The recommendation craft engine is technically tractable and clinically defensible. The risks are real but mitigatable. The 9-week implementation horizon (with extensions) is consistent with the implementation plan's Phase 2 envelope. The pilot from Month 5 onward will produce the operational data that tunes the thresholds and validates the approach.

What this document does not specify, and what should be the subject of subsequent work:

- **The Layer 4 surfaces UX implementation** (worklist, resident workspace, GP communication hub) — these consume the craft engine but require their own design discipline
- **The pharmacy-employer view design** (per v3.0 §9 trust architecture) — what an employer sees about pharmacist work, with appropriate consent and aggregation rules
- **The RACH operator view design** — facility-level dashboards consuming craft engine output as PHARMA-Care indicators
- **The regulator audit interface** — what a regulator querying the platform under formal data-sharing agreement experiences

Each of these merits its own implementation guideline document of similar rigour.

— Claude
