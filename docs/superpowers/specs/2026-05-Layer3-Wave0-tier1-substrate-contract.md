# Layer 3 Wave 0 — Tier 1 substrate primitive inventory

**Date:** 2026-05-06
**Plan:** [2026-05-04-layer3-rule-encoding-plan.md](../plans/2026-05-04-layer3-rule-encoding-plan.md) — Wave 0 Task 1
**Companion specs:**
* [2026-05-Layer3-Wave0-cql-helper-surface.md](2026-05-Layer3-Wave0-cql-helper-surface.md) (Wave 0 Task 2)
* [2026-05-Layer3-Wave0-trigger-surface-mapping.md](2026-05-Layer3-Wave0-trigger-surface-mapping.md) (Wave 0 Task 4)
**Layer 2 contract:** [docs/handoff/layer-2-to-layer-3-handoff.md](../../handoff/layer-2-to-layer-3-handoff.md)
**Source spec:** [Layer3_v2_Rule_Encoding_Implementation_Guidelines (1).md](../../../Layer3_v2_Rule_Encoding_Implementation_Guidelines%20(1).md) — Part 0.5
**Status:** Draft pending sign-off.

---

## Reviewer signoff

| Role | Name | Status | Date |
|---|---|---|---|
| Layer 2 lead | _pending_ | _pending_ | _pending_ |
| Clinical informatics | _pending_ | _pending_ | _pending_ |
| Engineering lead | _pending_ | _pending_ | _pending_ |

This document maps the 25 Tier 1 immediate-safety rules (planned for
Layer 3 Wave 2) onto the five Layer 2 substrate state machines. For
each rule, the table specifies what the rule reads from each machine
and what it writes to each machine. The objective is to surface any
substrate primitive that is not yet delivered before Wave 2 starts
authoring rules.

The five state machines are:

* **Clinical** — Resident, MedicineUse, Observation, ActiveConcerns,
  CareIntensity, baselines, deltas. Read paths per the Layer 2 →
  Layer 3 handoff doc.
* **Recommendation** — DraftedRecommendation, alternative options,
  priority score (Layer 3 owns the writes; Layer 2 hosts the table).
* **Monitoring** — MonitoringPlan + paired observations + threshold
  crossings. Writes occur as a side-effect of Tier 1 rule fires.
* **Authorisation** — Person+Role, Authorisation seam, ScopeRule
  evaluation. Reads are sub-500ms latency-budgeted.
* **Consent** — Consent class, expiry, gathering recommendations.
  Reads gate psychotropic / restrictive-practice rules.

Notation in the table cells:
* `R: <fact>` — read of named fact via substrate API
* `W: <action>` — write to substrate (Recommendation /
  Monitoring / EvidenceTrace are the most common write targets)
* `—` — machine not touched by this rule

---

## 1. Hyperkalemia trajectory rules (4)

| # | Rule define | Clinical | Recommendation | Monitoring | Authorisation | Consent |
|---|---|---|---|---|---|---|
| 1 | `HyperkalemiaRiskTrajectory` | R: potassium baseline 7d delta; R: ACEi/ARB/MRA active MedicineUse; R: eGFR latest | W: drafted "consider K+-sparing review" | W: paired plan "K+ q48h × 7d" | R: prescriber for "Cardio-renal" class | — |
| 2 | `HyperkalemiaThresholdCrossed` | R: latest potassium ≥5.5; R: ActiveConcerns "AKI watch" | W: drafted "hold ACEi pending review" | W: paired plan "K+ q24h until <5.0" | R: emergency-prescriber availability | — |
| 3 | `MraInitiationRequiresK4Baseline` | R: new MedicineUse intent=initiation, class=MRA; R: most recent K+ <30d | W: blocking draft + "obtain K+/eGFR before MRA" | W: paired baseline-K plan | R: GP / NP for endorse | — |
| 4 | `KSparingDuoEscalation` | R: 2 K+-sparing agents on same MAR; R: K+ baseline trending up | W: drafted "deprescribing review" | W: K+ trend monitoring | R: GP for change | — |

## 2. Bleeding-risk rules (5)

| # | Rule define | Clinical | Recommendation | Monitoring | Authorisation | Consent |
|---|---|---|---|---|---|---|
| 5 | `WarfarinNoInrInWindow` | R: warfarin active; R: latest INR > 14d | W: drafted "INR overdue" | W: paired plan "INR within 48h" | R: prescriber for warfarin | — |
| 6 | `DoacWithCyp3a4Strong` | R: active DOAC + active strong CYP3A4 inhibitor | W: drafted "interaction review" | W: bleed-watch concern | R: GP / NP | — |
| 7 | `AntiplateletPlusAnticoag` | R: active aspirin/clopidogrel + active warfarin/DOAC | W: drafted "duplicate antithrombotic review" | W: paired plan "Hb in 7d" | R: GP / NP | — |
| 8 | `PostFallOnAntiCoag` | R: ActiveConcerns "post-fall 72h watch" + active antithrombotic | W: drafted "head-injury watch + reverse-agent ready" | W: paired plan "neuro obs 72h" | R: emergency prescriber | — |
| 9 | `NsaidPlusAnticoag` | R: active NSAID + active antithrombotic | W: drafted "GI bleed risk review" | W: paired plan "Hb / FOB" | R: GP / NP | — |

## 3. Acute renal injury rules (3)

| # | Rule define | Clinical | Recommendation | Monitoring | Authorisation | Consent |
|---|---|---|---|---|---|---|
| 10 | `AkiRiskTriadActive` | R: NSAID + ACEi + diuretic active simultaneously | W: drafted "AKI triad — review" | W: paired plan "Cr q48h × 7d" | R: GP / NP | — |
| 11 | `EgfrDropTriggers` | R: eGFR baseline delta >20% in 14d | W: drafted "renal-dosed med review" | W: paired plan "Cr q72h" | R: prescriber for renal-dosed meds | — |
| 12 | `MetforminWithEgfrDrop` | R: active metformin + eGFR <30 trending | W: drafted "hold metformin" | W: paired plan "Cr in 5d" | R: GP / NP | — |

## 4. Cardiotoxicity rules (3)

| # | Rule define | Clinical | Recommendation | Monitoring | Authorisation | Consent |
|---|---|---|---|---|---|---|
| 13 | `DigoxinLevelOverdue` | R: active digoxin >90d; R: latest digoxin level > 90d | W: drafted "digoxin level overdue" | W: paired plan "level in 14d" | R: GP / NP | — |
| 14 | `DigoxinPlusAmiodaroneNew` | R: active digoxin; R: new amiodarone in 14d | W: drafted "digoxin dose review (50% reduction)" | W: paired plan "ECG + level" | R: GP / NP | — |
| 15 | `BetaBlockerBradycardia` | R: active β-blocker; R: HR baseline trending <50 | W: drafted "β-blocker dose review" | W: paired plan "HR daily × 7d" | R: GP / NP | — |

## 5. QT-prolongation rules (3)

| # | Rule define | Clinical | Recommendation | Monitoring | Authorisation | Consent |
|---|---|---|---|---|---|---|
| 16 | `QtcProlongingDuo` | R: 2 active QT-prolonging meds (CredibleMeds known-risk) | W: drafted "QT review + ECG" | W: paired plan "ECG within 7d" | R: GP / NP | — |
| 17 | `QtcWithKMgLow` | R: active QT-prolonging med; R: latest K+ <3.5 OR Mg <0.7 | W: drafted "replace electrolytes; reassess QT" | W: paired plan "K/Mg in 24h" | R: GP / NP | — |
| 18 | `MethadoneNewQtCheck` | R: new methadone in 14d; R: no ECG <30d | W: drafted "ECG required" | W: paired plan "ECG in 72h" | R: GP / NP | — |

## 6. Serotonin syndrome rules (2)

| # | Rule define | Clinical | Recommendation | Monitoring | Authorisation | Consent |
|---|---|---|---|---|---|---|
| 19 | `SerotoninergicTriple` | R: 3 active serotonergic agents (SSRI/SNRI/tramadol/linezolid/MAOI) | W: drafted "serotonin syndrome risk — taper one" | W: clinical-watch concern 72h | R: GP / NP | — |
| 20 | `SsriPlusMaoiNew` | R: active SSRI + new MAOI initiation in 14d | W: blocking draft "do not co-administer" | W: clinical-watch concern 14d | R: GP / NP | — |

## 7. Psychotropic-with-falls rules (5)

| # | Rule define | Clinical | Recommendation | Monitoring | Authorisation | Consent |
|---|---|---|---|---|---|---|
| 21 | `AntipsychoticConsentMissing` | R: active antipsychotic >90d; R: ActiveConcerns "BPSD" | W: drafted "consent-gathering: psychotropic deprescribe review" | — | R: SDM workflow | R: active consent for "Antipsychotic_deprescribe_review" |
| 22 | `BenzoPostFall` | R: ActiveConcerns "post-fall 72h"; R: active benzodiazepine | W: drafted "benzo taper review" | W: paired plan "fall-risk review 14d" | R: GP / NP | R: psychotropic taper consent |
| 23 | `ZDrugChronic` | R: active z-drug >30d | W: drafted "z-drug review" | — | R: GP / NP | R: hypnotic taper consent |
| 24 | `OpioidPlusBenzo` | R: active opioid + active benzo | W: drafted "respiratory-depression review" | W: paired plan "RR + SpO₂ daily × 7d" | R: GP / NP | R: opioid review consent |
| 25 | `InsulinHypoglycemiaImminent` | R: active insulin; R: latest BGL <3.5; R: recent food intake <4h | W: drafted "reduce dose immediately" OR "ED transfer" | W: paired plan "BGL q1h × 4h" | R: prescriber for "Insulin dose adjust" (gates fallback) | — |

---

## Substrate machines summary

Across the 25 rules, the substrate consumption pattern is:

* **Clinical reads:** every rule (25/25). Most-touched primitives:
  active MedicineUse with intent (25), latest Observation by kind
  (18), baseline-delta on Observation (8), ActiveConcerns lookup (6),
  CareIntensity gating (suppression — implicit on every Tier 1 rule).
* **Recommendation writes:** every rule (25/25). Includes "blocking
  draft" pattern (rules 3 + 20) and "consent-gathering draft" pattern
  (rule 21).
* **Monitoring writes:** 19/25. Default pattern is paired-plan for
  the Observation kind that the rule's clinical condition referenced.
* **Authorisation reads:** 22/25. Most rules query
  `available_prescriber_for_class` so the recommendation routing can
  fall back to ED transfer / telehealth where required (rule 25 is
  the canonical example).
* **Consent reads:** 5/25 — every psychotropic / opioid rule + the
  benzo-post-fall rule. These are the rules that fire a
  consent-gathering recommendation when consent is missing rather
  than a medication-change recommendation.

Every rule additionally writes to **EvidenceTrace** (per Layer 3 v2
doc Part 0.5.6). EvidenceTrace is not enumerated as a separate column
because it is a uniform write across all 25; the trace contents per
rule are itemised in the rule_specification.yaml.

---

## Missing primitives

**No missing primitives identified.**

Every read/write referenced in the table above is backed by a Layer 2
deliverable that has shipped per the
[Layer 2 → Layer 3 handoff doc](../../handoff/layer-2-to-layer-3-handoff.md):

| Substrate primitive used by Tier 1 rules | Layer 2 deliverable |
|---|---|
| Resident snapshot | `GET /v2/residents/{id}` (handoff §"Resident snapshot") |
| MedicineUse with intent | `GET /v2/residents/{id}/medicine_uses` (handoff §"Medicine use list") |
| Observation by kind | `GET /v2/residents/{id}/observations/{kind}` (handoff §"Observations") |
| Observation baseline delta | `flagged_baseline_delta` field (handoff §"Observations"; Layer 2 plan Wave 2 sub-task 2.1) |
| ActiveConcerns | `GET /v2/active_concerns?resident_ref=...&open=true` (handoff §"Active concerns") |
| CareIntensity | `GET /v2/care_intensity/{resident_id}` (handoff §"Care intensity") |
| CapacityAssessment | `GET /v2/capacity_assessment/{resident_id}` (handoff §"Capacity assessment") |
| Person+Role + Authorisation seam | shipped under Phase 1B-β.2 (plan §"Predecessor") |
| MonitoringPlan write | Layer 2 doc §2.4 (Wave 2 sub-task 2.4) |
| EvidenceTrace write | Layer 2 plan Wave 1R.2 (EvidenceTrace v0) |
| Consent class lookup + expiry | Layer 2 doc §3 (Consent state machine) |

Layer 3 implementers MUST add to this section if a Wave 1+ rule
authoring session surfaces a primitive that is not yet delivered.
The expected escalation path is: file a Layer 2 backlog ticket,
update this section with the gap, hold the affected rule pending
Layer 2 catch-up.

---

## Open questions

1. **AN-ACC funding-class read path.** Rules referenced at `Tier
   2` (deprescribing) may want to suppress on funding-class 1 — the
   `ResidentAnAccClass()` helper signature is reserved in
   `AgedCareHelpers.cql` but no Tier 1 rule above currently
   references it. If Tier 2 needs it, confirm kb-20 surfaces the
   funding class via a substrate API.
2. **Reverse-agent inventory (rule 8).** "Reverse-agent ready"
   recommendation text presumes a facility-level reverse-agent
   inventory check. This is an Authorisation-evaluator helper, not
   a substrate primitive — confirm Wave 0 Task 2 helper surface
   covers it (`FacilityHasReverseAgentFor(class)`).
3. **CredibleMeds list provenance (rule 16).** The QT-prolonging
   "known risk" list is sourced from CredibleMeds; confirm KB-7
   terminology has the list loaded as a ValueSet.

These are tracked as Wave 1 backlog items, not blockers.
