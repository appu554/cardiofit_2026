# Layer 3 v2 — Rule Encoding for ACOP over the Five-State-Machine Substrate

**Version:** 2.0 — substantial revision of v1.0
**Date:** April 2026
**Status:** Implementation guidelines for the rule encoding layer that operates over the five-state-machine substrate

**Companion documents:**
- *Vaidshala Final Product Proposal v2.0 Revision Mapping* (the strategic context)
- *Layer 1 v2 Australian Aged Care Implementation Guidelines* (the sources)
- *Layer 3 Rule Encoding Implementation Guidelines v1.0* (being revised)
- *Layer 2 Implementation Guidelines* (forthcoming — the substrate this document depends on)

**Audience:** CQL authors, clinical informatics, engineering leads building the Authorisation evaluator and rule infrastructure.

**Author:** Claude (Anthropic), grounded in 2024-26 CDS literature, the FHIR Clinical Reasoning Module v6.0.0-ballot, CDS Hooks v2.0, the verified Australian regulatory landscape, and the McCoy alert appropriateness framework.

---

## Part 0 — What changed and what didn't

### 0.1 What still holds from Layer 3 v1.0

The substantive Layer 3 v1.0 content holds up well and continues unchanged:

- **The five-stage authoring pipeline** (clinical translation → CQL drafting → two-gate validation → clinical test cases → governance promotion). The PPI deprescribing worked example is still the canonical template.
- **The four-class suppression model** (eligibility / recently-actioned / cohort / workflow). The defer-vs-suppress-vs-fire trichotomy. The override-reason taxonomy. The volume budget (≤5 alerts/resident/day for routine review).
- **The four-bucket priority taxonomy** (immediate safety / deprescribing / quality gap / surveillance). Within-tier scoring. Cluster-first surfacing.
- **The CompatibilityChecker bidirectional contract** with multi-source authority resolution, cross-rule consistency, suppression coverage audit, test-case compatibility.
- **The 7-day SLA target** from authoritative source publication to deployed CQL define.

These remain the methodology for rule encoding. The book is still the book.

### 0.2 What v2.0 adds

Three things v1.0 didn't have:

**Part 0.5 — The Five-State-Machine Substrate.** Rule firing in v2.0 happens over a substrate that v1.0 didn't address. Rules now must reference and respect five state machines (Authorisation, Recommendation, Monitoring, Clinical state, Consent), not just the patient's medication and condition lists. This changes how rules are authored, how suppressions work, how priority is computed, and how the firing event propagates downstream.

**Part 4.5 — The Authorisation evaluator as a separate subsystem.** The Authorisation state machine cannot be encoded as CQL rules alongside clinical rules. It's a separate runtime subsystem with its own rule format, latency budget, cache strategy, and audit query API. CQL rules consume from the Authorisation evaluator at firing time but don't implement it.

**Part 5.5 — ScopeRules-as-data architecture.** Jurisdiction-aware regulatory rules (Victorian PCW exclusion, designated RN prescriber prescribing agreements, Tasmanian pharmacist co-prescriber pilot scope, ACOP credential requirements) cannot be hardcoded in CQL. They live in a separate jurisdiction-aware rules engine that the Authorisation evaluator queries.

### 0.3 What got revised

Two pieces of v1.0 needed updating:

**The four-bucket priority taxonomy** now needs a fifth dimension: **Authorisation gating.** A rule that fires for a recommendation requiring a Schedule 8 prescription has a different priority — and a different display — depending on whether any authorised prescriber is currently available at the facility. This isn't a new tier; it's a modifier within the existing tiers.

**The 7-day SLA** now also covers regulatory rule updates, not just clinical guideline updates. When the Aged Care Rules 2025 change, when a state's PCW exclusion legislation passes, when a designated RN prescriber's prescribing agreement is amended — these are also "source publications" that must propagate to deployed rules within 7 days. This makes Layer 3 v2 work significantly broader in scope.

---

## Part 0.5 — The Five-State-Machine Substrate

Layer 3 v2 cannot be discussed in isolation from the substrate it operates over. Before any rule authoring methodology applies, the team must understand how rules interact with the five state machines. This section is essential reading.

### 0.5.1 The substrate, recapped

```
┌─────────────────────────────────────────────────────────────────────┐
│ Five interlocking state machines, sharing one EvidenceTrace graph:   │
│                                                                      │
│  1. Authorisation — runtime: who may act on this resident now?       │
│  2. Recommendation — proposed clinical action lifecycle              │
│  3. Monitoring — observation obligations with stop criteria          │
│  4. Clinical state — slowly-evolving baseline + active concerns      │
│  5. Consent — regulatory substrate for restrictive practice          │
│                                                                      │
│ Substrate entities: Resident, Person+Role, MedicineUse (intent +     │
│ target + stop criteria), Observation (with delta-from-baseline),     │
│ Event, EvidenceTrace                                                 │
└─────────────────────────────────────────────────────────────────────┘
```

CQL rules live alongside this substrate. They don't implement any of the state machines themselves; they read from and write to them.

### 0.5.2 What CQL rules read from the substrate

When a CQL rule evaluates, it has access to:

**From Clinical state machine:**
- Per-observation-type baseline values with confidence intervals
- Delta-from-baseline values (computed on write, available on read)
- Active concerns (open clinical questions)
- Care intensity tag (palliative / comfort / active treatment / rehabilitation)
- Cognitive capacity status (with date of last assessment)

**From Recommendation state machine:**
- Active recommendations for this resident (state, age, evidence)
- Recently-closed recommendations (for suppression-by-recent-action)
- Deferred recommendations awaiting prescriber review

**From Monitoring state machine:**
- Active monitoring plans for this resident
- Observation expectations (what's due, what's overdue)
- Abnormal monitoring findings (which may be a trigger for new rules)

**From Consent state machine:**
- Active consents for medication classes
- Consent expiry dates
- Withdrawn or under-review consents
- Substitute decision-maker scope

**From Authorisation state machine (read-only):**
- Currently-authorised prescribers at this facility
- Roles available for action
- Jurisdictional ScopeRules in effect

### 0.5.3 What CQL rules write to the substrate

When a CQL rule fires (after suppression evaluation), it writes:

**To the Recommendation state machine:** a new Recommendation in `drafted` state, with full evidence chain, proposed action, proposed monitoring plan reference, proposed consent requirement reference (if applicable).

**To the EvidenceTrace graph:** a state transition entry recording the rule fire, the actor (the rule itself, with rule_id and version), the inputs that satisfied the rule conditions, the suppressions that were evaluated, and the alternative options considered.

**To the Monitoring state machine (when rule includes a monitoring proposal):** a paired draft MonitoringPlan with `expected_observations`, `monitoring_window`, and `stop_criteria` defined.

**Critically:** CQL rules do NOT write to the Authorisation, Clinical state, or Consent state machines directly. Those state machines have their own input mechanisms (substrate-level events, system actions, manual entries by authorised actors).

### 0.5.4 What this means for rule authoring

Three concrete implications for how rules are written:

**Implication 1: Rules must reference Clinical state baselines, not raw observations.**

v1.0 rules could fire on snapshot conditions: "if K+ > 5.5 AND on ACEi/ARB AND on K-sparing diuretic." That's correct as far as it goes, but it misses the temporal dimension that's the actual clinical signal.

v2.0 rules should reference baseline-adjusted state: "if K+ delta-from-baseline > 0.8 mEq/L over 7 days OR K+ absolute > 5.5 AND on ACEi/ARB AND on K-sparing diuretic AND no recent dose-change-driven explanation."

The Clinical state machine produces these deltas. Rules consume them. **This is what turns rule firing from "you're abnormal" to "you're getting worse" — which is the difference between alert noise and clinical signal.**

```cql
// v1.0 style — snapshot reasoning
define "Hyperkalemia Risk Snapshot":
  Latest("Potassium").value > 5.5
    and Patient.medications contains "ACEi/ARB"
    and Patient.medications contains "K-sparing diuretic"

// v2.0 style — baseline-aware reasoning
define "Hyperkalemia Risk Trajectory":
  (
    ClinicalState.PotassiumDeltaFromBaseline7Days > 0.8
      or Latest("Potassium").value > 5.5
  )
    and Patient.medications contains "ACEi/ARB"
    and Patient.medications contains "K-sparing diuretic"
    and not Patient.recent_K_affecting_dose_change_within_days(14)
```

The second form fires earlier (catching trajectory before threshold breach), suppresses correctly (recent dose change is an explanation, not a separate concern), and produces a recommendation that's actionable rather than reactive.

**Implication 2: Rules must check Consent state for psychotropic and restrictive-practice classes.**

Some recommendation classes can't proceed without an active matching Consent. Rules in these classes must check Consent state at firing time, not at recommendation submission time.

```cql
define "Antipsychotic Long Term Without BPSD Documentation":
  exists(Patient.medications with class = "Antipsychotic" and duration_days > 90)
    and not exists(Patient.conditions with code in "BPSD Severe ValueSet" and recent_within_days(180))
    and not Patient.has_palliative_goals_of_care
    and ConsentState.has_active_consent_for_class("Antipsychotic", "deprescribe_review")
    // ↑ this check is new in v2.0; the rule cannot meaningfully fire unless deprescribe-review consent is active
```

**Where consent is missing**, the rule should fire a *consent-gathering Recommendation* rather than a *medication-change Recommendation*. The action proposed is "discuss antipsychotic deprescribing with SDM and obtain consent" rather than "cease risperidone." This is a structurally different recommendation handled by the Consent state machine workflow.

**Implication 3: Rules must check Authorisation availability for high-tier actions.**

A rule that fires for a Schedule 8 prescription in a Tier 1 immediate-safety scenario must know whether any authorised Schedule 8 prescriber is currently available at the facility. If not, the recommendation needs different routing — possibly to telehealth, possibly to an after-hours service, possibly to ED transfer.

```cql
// Rule: insulin overdose risk requiring immediate dose adjustment
define "Insulin Risk Imminent Hypoglycemia":
  Patient.is_on_insulin
    and Latest("BGL").value < 3.5
    and Patient.recent_food_intake_hours < 4
    // ... clinical conditions ...

// Rule action depends on Authorisation availability
define "Recommended Action":
  if AuthorisationState.has_available_prescriber_for_class("Insulin", "dose_adjustment")
  then "Reduce_dose_immediate_prescriber_attention"
  else "ED_transfer_protocol"
```

This is the architecturally important point: **Layer 3 rules don't decide who acts; they propose what should happen and let Authorisation routing determine the path.**

### 0.5.5 The trigger surface change

In v1.0, rules fired on changes to medication or condition lists. In v2.0, the trigger surface expands:

| Trigger event source | Examples | State machine origin |
|---|---|---|
| Medication change | New prescription, dose change, cessation | Recommendation → eNRMC |
| Condition change | New diagnosis, condition resolution | eNRMC, GP notes, MHR |
| Observation update | Lab result, vital sign, weight | Pathology, RN observation, eMAR |
| **Baseline delta** | Sedation 4/7 vs baseline 0/7, eGFR drop >20% in 14 days | **Clinical state machine** |
| **Active concern resolution** | "Watching for delayed head injury 72h" expires | **Clinical state machine** |
| **Monitoring threshold crossed** | "K+ trending up" hits 5.5 | **Monitoring state machine** |
| **Consent expiry approaching** | Antipsychotic consent expires in 14 days | **Consent state machine** |
| **Authorisation expiry approaching** | ACOP credential expires in 30 days | **Authorisation state machine** |
| Care intensity transition | Active treatment → palliative | **Clinical state machine** |
| Care transition | Hospital discharge, RACF admission | Event |

The expanded trigger surface means rules can be authored to fire on patterns that v1.0 couldn't capture: trajectory-based firings, lifecycle-aware firings, regulatory-deadline firings. This is the substrate dividend.

### 0.5.6 The EvidenceTrace requirement

Every rule fire writes to the EvidenceTrace graph. v1.0 specified this as part of governance audit. v2.0 makes it more architecturally important.

The EvidenceTrace entry for a rule fire must record:
- **action**: rule_id, rule_version, fire_timestamp
- **actor**: the rule itself (system actor, with full attribution chain to authoring pharmacist, medical director sign-off, source authority)
- **inputs**: links to all Observations, Conditions, MedicineUses that satisfied conditions
- **suppressions evaluated**: which suppressions were checked (whether they fired or not)
- **alternative options considered**: if the rule has alternative recommendations (e.g., taper vs abrupt cessation), which was selected and why
- **clinical state context**: the resident's care intensity, active concerns, baseline values at fire time
- **authorisation context**: which prescribers were available at fire time (for routing)
- **consent context**: relevant active consents at fire time

This is the longitudinal data that becomes the moat. **Treat the EvidenceTrace as a first-class output of every rule fire, not as audit logging.**

---

## Part 1 — CQL define authoring methodology (mostly unchanged from v1.0)

The five-stage pipeline from v1.0 holds. Only Stage 1 (clinical translation) and Stage 4 (clinical test cases) need v2.0 additions.

### 1.1 Stage 1 changes — `rule_specification.yaml` extensions

The yaml schema from v1.0 should be extended with:

**State machine references:**
```yaml
state_machine_references:
  reads_from:
    - state_machine: clinical_state
      facts: [potassium_baseline_7d_delta, care_intensity, palliative_status]
    - state_machine: consent
      facts: [active_consents_for_class("Antipsychotic")]
    - state_machine: authorisation
      facts: [available_prescribers_for_class("Schedule_4")]
  writes_to:
    - state_machine: recommendation
      action: create_drafted_recommendation
    - state_machine: monitoring
      action: propose_paired_monitoring_plan
    - state_machine: evidence_trace
      action: record_full_fire_context

trigger_sources:
  - source: medication_change
    pattern: any_change
  - source: clinical_state_baseline_delta
    measure: potassium_delta_7d
    threshold: ">0.8"
  - source: monitoring_threshold_crossed
    plan_type: post_fall_observation
    threshold_type: any_abnormal
```

**Authorisation gating (where applicable):**
```yaml
authorisation_gating:
  required_prescriber_class:
    - "Schedule_4_prescriber"
  fallback_routing_if_unavailable:
    - "telehealth"
    - "ED_transfer_protocol"
```

**Consent gating (where applicable):**
```yaml
consent_gating:
  required_consent_class: "Antipsychotic_deprescribe_review"
  consent_missing_action: "create_consent_gathering_recommendation"
  consent_missing_recommendation_template: "discuss_psychotropic_with_SDM"
```

These extensions make the rule_specification a complete clinical-and-regulatory contract, not just a clinical translation.

### 1.2 Stage 4 changes — clinical test cases

Test cases must now cover the substrate interactions, not just snapshot conditions. Per rule, in addition to v1.0's test classes (positive case, suppression cases, boundary cases, missing-data cases), v2.0 requires:

- **Baseline-aware fire test:** synthetic patient where snapshot conditions don't satisfy the rule, but baseline-delta conditions do. Rule should fire.
- **Authorisation-routing test:** synthetic facility where no authorised prescriber for required class is available. Rule should fire with fallback routing.
- **Consent-gating test:** synthetic patient where consent is missing for the relevant class. Rule should fire a consent-gathering Recommendation, not a medication-change Recommendation.
- **Care-intensity test:** synthetic patient with palliative care intensity. Rule should suppress (or modify) based on care intensity.
- **EvidenceTrace test:** verify that the rule fire writes a complete EvidenceTrace entry with all required fields.

For 200+ rules × ~10 tests per rule = 2,000+ test cases. This is a real investment but is what makes the rule library defensible at scale.

---

## Part 2 — Suppression model (extended for substrate)

The four-class suppression model from v1.0 (eligibility / recently-actioned / cohort / workflow) holds. v2.0 adds two new suppression sources from the substrate.

### 2.1 Class 5 — Substrate-state suppression

Some suppressions are determined by Clinical state, not by patient eligibility characteristics. Examples:

- **Acute concern open**: rule "PPI long-term without indication" should suppress (not just defer) if Clinical state has open active concern "post-acute illness, watching for medication-related GI bleeding for 14 days." The suppression is automatic once the concern resolves.
- **Recent baseline shift**: rule "ACEi at full dose" should defer if Clinical state shows eGFR baseline shifting downward in last 30 days — a new ACEi recommendation might be valid in 60 days but not now.
- **Care intensity mismatch**: rule "statin in primary prevention" should suppress if Clinical state has care intensity = palliative.

These suppressions are computed by the Clinical state machine and surface to rule firing as boolean flags. Rules check them like any other suppression.

### 2.2 Class 6 — Authorisation-context suppression

Some rules shouldn't fire if there's no realistic authorisation pathway. Examples:

- **Schedule 8 dose adjustment** in a facility with no Schedule 8 prescriber available within 6 hours, AND the resident is clinically stable: the rule fires informationally but suppresses the actionable recommendation, with a flag for "no immediate prescriber available."
- **Designated RN prescriber recommendation** in a facility with no endorsed RN prescriber: the rule still fires, but routes to GP or NP rather than to the (non-existent) RN prescriber.

The Authorisation evaluator (Part 4.5) provides these flags. Rules that gate on them remain authoring-friendly because the gating is declarative, not procedural.

### 2.3 The volume budget under substrate

The ≤5 alerts/resident/day for routine review remains. But under substrate-aware rule authoring, the *kinds* of alerts change:

- More **trajectory-based** alerts (caught before threshold breach), fewer **threshold-breach** alerts
- More **consent-gathering** recommendations (separate workflow), fewer **medication-change** recommendations blocked at submission time
- More **lifecycle-aware** alerts (consent expiring, monitoring overdue), more proactive workflow

The volume budget operates on the *display surface*, not the rule firing surface. A rule may fire informationally without surfacing to the user; only actionable items count toward the budget.

---

## Part 3 — Priority ranking (extended with Authorisation modifier)

The four-bucket urgency taxonomy (immediate safety / deprescribing / quality gap / surveillance) holds. v2.0 adds an **Authorisation modifier** within each tier.

### 3.1 The Authorisation modifier

Every priority score gets adjusted by Authorisation context:

```
Final priority score = base_score 
                     + clinical_severity_modifier
                     + resident_vulnerability_modifier
                     + intervention_yield_modifier
                     + AUTHORISATION_AVAILABILITY_MODIFIER  // new in v2.0
                     - recently_addressed_decay
```

The Authorisation availability modifier:
- **+10** if no authorised prescriber for required class is available within 6 hours (rule rises in priority because the routing path is harder)
- **+5** if authorised prescriber is available but not the patient's primary GP (handoff complexity)
- **0** baseline (authorised primary prescriber available)
- **-5** if multiple authorised prescribers are available for the resident (handoff is straightforward)

This is small in magnitude but matters for prioritising the worklist when many rules fire across many residents.

### 3.2 The "deferred awaiting consent" priority

A new priority class within Tier 2 deprescribing: recommendations awaiting active consent. These have a different display:

- They show in the worklist as "consent gathering needed" rather than "deprescribing recommendation"
- They route to the SDM/family workflow rather than the GP workflow
- Their urgency is determined by the *clinical risk of inaction* (how risky is continued antipsychotic use?) plus *consent timeline* (how soon does this need to happen?)

This is structurally different from a standard recommendation and the priority score reflects that.

### 3.3 GP-perspective re-ranking under substrate

When recommendations surface to the prescriber via Smart Form / decision packet, the substrate gives the GP additional context:

- **Resident's care intensity** — same recommendation has different framing for active treatment vs palliative
- **Baseline trajectory** — "this medication's planned target was BP <140/90; current trajectory shows we're now at 110/65 with 3 falls in last month"
- **Authorisation history** — "this is the third deprescribing recommendation in 90 days, GP previously deferred twice"
- **Consent status** — "SDM has indicated openness to deprescribing review, consent active"

These are the "cognitive shortcuts" the GP needs to make a decision quickly. They come from the substrate, not from new rule logic.

---

## Part 4 — CompatibilityChecker (extended for substrate)

The bidirectional contract from v1.0 (Event A: CQL bundle changes; Event B: L3 facts change) holds. v2.0 adds two new event sources.

### 4.1 Event C — Substrate schema changes

When the Clinical state machine adds a new baseline type, when the Consent state machine adds a new class, when the Monitoring state machine adds a new threshold rule type — these are substrate schema changes that potentially invalidate rule libraries.

**Workflow:**
1. Substrate change is proposed via the Layer 2 governance process
2. CompatibilityChecker evaluates: which CQL defines reference the changing substrate elements?
3. Per-define `compatibility_status` updates: STALE if substrate change breaks rule logic, COMPATIBLE if not
4. STALE defines block rule library deployment until updated

### 4.2 Event D — Regulatory ScopeRule changes

When the Aged Care Rules 2025 change, when a state's PCW exclusion legislation passes, when a designated RN prescriber's prescribing agreement is amended — these are regulatory ScopeRule changes that potentially affect rule firing.

**Workflow:**
1. ScopeRule change is published (parsed from Layer 1 sources)
2. CompatibilityChecker evaluates: which CQL defines have authorisation_gating that depends on the changing ScopeRule?
3. Per-define `compatibility_status` updates as above
4. Affected defines route to Layer 3 governance for review

### 4.3 The 7-day SLA, revised for v2.0

v1.0 specified 7 days from authoritative source publication to deployed CQL define. v2.0 broadens the SLA to cover:

- Clinical guideline changes (ADG, STOPP/START, Beers)
- Regulatory rule changes (Aged Care Rules 2025, ScopeRule changes)
- Substrate schema changes (Clinical state baseline types, Consent classes, Monitoring threshold types)
- Source authority version pins (when AMT, SNOMED-CT-AU, LOINC AU update)

**Realistic timeline (revised):**
- Day 1: Source detected, L0–L6 extraction triggered
- Day 2: Facts land in DRAFT, governance review begins
- Day 3: Facts promoted to ACTIVE; substrate schemas update if affected
- Day 4: Affected CQL defines marked for review; affected ScopeRules deployed
- Day 5-6: rule_specification.yaml updates, CQL regeneration, two-gate validation, test case updates, governance review
- Day 7: New defines ACTIVE, engines reloaded; ScopeRules refreshed in Authorisation evaluator

**For regulatory ScopeRule changes specifically**, the SLA may be tighter — Victorian PCW exclusion enforcement begins 29 September 2026 with no grace period after that. Rules that depend on it must be deployed by 1 July 2026 for the grace period.

---

## Part 4.5 — The Authorisation evaluator as a separate subsystem

This is a major v2.0 addition. The Authorisation state machine is not a CQL rule library — it's a separate runtime subsystem. Layer 3 v2 specifies its design here because it's load-bearing for rule firing.

### 4.5.1 Why it can't be CQL

CQL is designed for clinical knowledge artifacts that evaluate against a patient's clinical record. Authorisation queries don't fit this model:

- They're per-action, not per-patient (every action attempt fires an evaluation)
- They require sub-second latency (clinicians at the bedside cannot wait)
- They need cache aggressiveness with correctness invalidation
- They produce structured authorisation decisions with audit trails, not clinical recommendations

The Authorisation evaluator is a separate service. CQL rules consume from it (via helper functions like `AuthorisationState.has_available_prescriber_for_class("Schedule_4")`) but don't implement it.

### 4.5.2 The rule format

Authorisation rules are declarative, jurisdiction-aware, and time-aware. They live in a structured rules engine (not CQL) with the following format:

```yaml
authorisation_rule:
  rule_id: "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01"
  jurisdiction: "AU/VIC"
  effective_period:
    start_date: "2026-07-01"
    end_date: null  # in force until amended
    grace_period_days: 90
  
  applies_to:
    role: "PCW"
    action_class: "administer"
    medication_schedule: ["S4", "S8", "S9"]
    medication_class_includes:
      - "antibiotics"
      - "opioid_analgesics"
      - "benzodiazepines"
      - "drugs_of_dependence"
    resident_self_administering: false
  
  evaluation:
    decision: "denied"
    reason: "Victorian Drugs, Poisons and Controlled Substances Amendment 
             (Medication Administration in Residential Aged Care) Act 2025"
    fallback_required: true
    fallback_eligible_roles:
      - "RN"
      - "EN" 
      - "Pharmacist"
      - "Medical_practitioner"
  
  audit:
    legislative_reference: "Drugs, Poisons and Controlled Substances Amendment 
                            (Medication Administration in Residential Aged 
                            Care) Act 2025"
    recordkeeping_required: true
    recordkeeping_period_years: 7
```

```yaml
authorisation_rule:
  rule_id: "AUS-NMBA-DRNP-PRESCRIBING-AGREEMENT-2025-09-30"
  jurisdiction: "AU"
  effective_period:
    start_date: "2025-09-30"
    end_date: null
  
  applies_to:
    role: "designated_RN_prescriber"
    action_class: "prescribe"
  
  evaluation:
    decision: "granted_with_conditions"
    conditions:
      - condition: "valid_prescribing_agreement_in_place"
        check: "PrescribingAgreement.exists_for_person_AND_resident_AND_medication_class"
      - condition: "mentorship_complete_or_active"
        check: "MentorshipStatus IN ['active', 'complete']"
      - condition: "medication_class_in_agreement_scope"
        check: "PrescribingAgreement.scope_includes(medication_class)"
      - condition: "endorsement_current"
        check: "Credential.endorsement_valid_at_action_time"
      - condition: "scope_match"
        check: "Action.scope_matches(PrescribingAgreement.scope)"
    
    if_any_condition_fails:
      decision: "denied"
      reason: "Designated RN prescribing requirements not met"
  
  audit:
    legislative_reference: "NMBA Registration Standard: Endorsement for 
                            scheduled medicines – designated registered 
                            nurse prescriber, effective 30 September 2025"
    recordkeeping_required: true
```

Rules are stored in a database (not in CQL files), versioned per jurisdiction, with effective_period for time-aware activation.

### 4.5.3 Cache invalidation strategy

Authorisation queries must complete in <500ms p95 for V1, target <200ms for V2. This requires aggressive caching with correct invalidation.

**Cache key:** `(jurisdiction, role, action_class, medication_class, resident_id, fire_date)` — fully qualified by jurisdictional and temporal context.

**Cache TTL:** Per-rule, depending on what data the rule depends on:
- Static jurisdictional rules: 24 hours (e.g., Victorian PCW exclusion is effective for years)
- Per-credential rules: 1 hour (credentials expire and get renewed)
- Per-prescribing-agreement rules: 15 minutes (agreements get amended more frequently)
- Per-resident consent rules: 5 minutes (consents can be withdrawn quickly)

**Invalidation triggers:**
- Credential update → invalidate all entries for that person
- PrescribingAgreement update → invalidate all entries for that agreement's scope
- Consent update → invalidate all entries for that resident's consent class
- Substrate Resident update → invalidate all entries for that resident
- ScopeRule deployment → invalidate all entries for affected jurisdictions

**Cache warming:** for facilities with predictable activity patterns (e.g., every-Saturday DAA pack review), warm cache the day before for the next day's expected queries.

### 4.5.4 The audit query API

Beyond runtime evaluation, the Authorisation evaluator exposes an audit query API for regulators, RACH operators, and internal compliance:

**Sample queries:**

- "Show me every Schedule 4 administration on resident R during Q3 2026 and the authorisation that justified each one."
- "Show me every dose-change prescription by RN P during 2026-2027, with the prescribing agreement scope that authorised each."
- "Show me every PCW administration at facility F since 1 July 2026 and the corresponding authorisation rule that permitted it."
- "Show me every consent-gating event on resident R, with timestamps and SDM identity."

These queries must be answerable in seconds, not hours. They produce structured output (CSV, JSON, FHIR Bundle) that supports regulatory audit defensibility.

**Implementation approach:** the EvidenceTrace graph captures every authorisation evaluation. The audit query API is a structured query layer over the EvidenceTrace, optimized for regulatory reporting.

### 4.5.5 The Sunday-night-fall reconsidered

Recall the Sunday-night-fall walkthrough from the second synthesis document. The Authorisation evaluator is the gate that fires at every step:

**2147 Sunday — PCW Sarah finds Mary on the floor.**
PCW logs Event of class `clinical_observation` (subclass `fall`). Authorisation check: PCWs may log Events of class `clinical_observation`. Granted. Latency: <50ms.

**2152 Sunday — RN Jamie arrives.**
RN performs post-fall assessment. Authorisation check: RNs may write Observations of clinical assessment class. Granted. Latency: <50ms.

**Monday 0830 — ACOP pharmacist Priya logs in.**
Pharmacist views Mary's profile and queue. Authorisation check: ACOP-credentialed pharmacist with current APC training credential may view resident profiles and submit drafted Recommendations. Granted. Latency: <100ms.

**Tuesday — Dr Chen (GP) approves Recommendation.**
GP modifies and approves recommendation. Authorisation check: Dr Chen has GP registration, has Mary on his patient list, his prescribing authority covers temazepam taper. Granted. Latency: <200ms (this is the most complex check, involving credential lookup, patient list lookup, scope match).

**MonitoringPlan activates, RN observations begin.**
Authorisation check at every observation: RN may log monitoring observations per active MonitoringPlan. Granted. Latency: <50ms.

Total Authorisation evaluations across the workflow: 7 in this single fall scenario. **The latency budget exists at every one of these.** This is why the Authorisation evaluator must be a separate optimized subsystem, not a slow query layer.

### 4.5.6 Sequencing the build

Building the Authorisation evaluator is roughly 6-8 weeks of focused engineering work:

| Week | Work |
|---|---|
| 1 | Rule format design, schema design |
| 2 | Initial rule database implementation (storage, versioning, query) |
| 3 | Runtime evaluator with basic rules |
| 4 | Cache layer with TTL strategy |
| 5 | Cache invalidation triggers and integration |
| 6 | Audit query API |
| 7 | Performance testing, latency optimisation |
| 8 | Integration testing with CQL helpers and rule library |

This is V1 work, not MVP. **MVP can run with simple RBAC** — a basic role-permission matrix without runtime evaluation. The Authorisation evaluator is the V1 substrate that enables designated RN prescribers, Tasmanian pilot, Victorian PCW exclusion, and the audit query API.

---

## Part 5 — The implementation roadmap, revised

The six-wave roadmap from v1.0 holds in shape but expands in scope.

### 5.1 Wave 0 — Substrate scoping (Weeks 1-2, NEW in v2.0)

Before any Layer 3 wave begins, the team must scope (not build, scope) the five-state-machine substrate. Specifically:

- Identify which Layer 2 substrate entities are needed by Wave 2 rules (Tier 1 immediate safety)
- Identify which Clinical state baseline types are needed (potassium, eGFR, weight, sedation, etc.)
- Identify which Consent classes will be referenced by rules
- Identify which Authorisation gating will be needed
- Identify which substrate APIs the CQL helpers will need

This scoping informs Layer 2 implementation. **It's two weeks of cross-team work and it should happen before Layer 3 Wave 1 starts.**

### 5.2 Wave 1 — Authoring infrastructure (Weeks 3-4, was Weeks 1-2 in v1.0)

Same as v1.0 but with substrate-aware extensions:
- rule_specification.yaml schema extended with state_machine_references, authorisation_gating, consent_gating
- Anchor CQL defines updated to demonstrate substrate consumption (e.g., the "Hyperkalemia Risk Trajectory" example above)
- MedicationHelpers.cql and AgedCareHelpers.cql include substrate query helpers
- CompatibilityChecker extended with Event C and Event D handlers

### 5.3 Wave 2 — Tier 1 immediate-safety rules (Weeks 5-8)

Same scope as v1.0 (~25 rules), but now authored over the substrate. Each rule references baseline state where applicable, gates on authorisation where applicable, gates on consent where applicable. The PPI deprescribing example has been the v1.0 template; v2.0 needs new templates for:
- A baseline-aware trajectory rule (hyperkalemia trajectory above)
- A consent-gating rule (antipsychotic deprescribing review)
- An authorisation-gating rule (Schedule 8 dose adjustment)

### 5.4 Wave 3 — Tier 2 deprescribing rules (Weeks 9-16)

Same scope as v1.0 (~75 rules). Substrate consumption applies throughout. Effort estimate increases slightly (~1.7 days per rule vs v1.0's 1.5) due to substrate references.

### 5.5 Wave 4 — Tier 3 quality gap rules + Authorisation evaluator build (Weeks 17-22)

Tier 3 rules (~50) per v1.0 schedule. Concurrently, the Authorisation evaluator is built (Part 4.5.6 above). By end of Wave 4, the Authorisation evaluator is operational and rules can route on availability.

### 5.6 Wave 5 — Tier 4 surveillance rules + ScopeRules deployment (Weeks 23-25)

Tier 4 rules (~50) per v1.0 schedule. ScopeRules for Aged Care Rules 2025, Victorian PCW exclusion, designated RN prescriber prescribing agreements deployed.

### 5.7 Wave 6 — Continuous tuning (ongoing from Week 26)

Same as v1.0:
- Override-reason analysis (weekly)
- Coverage audit (monthly)
- Source update tracking (continuous, with broadened scope per Section 4.3)
- GP behaviour model refinement (monthly)
- New rule additions from coronial/ACQSC findings

### 5.8 Total Layer 3 effort summary, revised

| Wave | Effort | Output |
|---|---|---|
| 0. Substrate scoping (cross-team) | 2 weeks | Substrate requirements list |
| 1. Authoring infrastructure | 2 weeks | Toolchain + helpers + anchor defines |
| 2. Tier 1 immediate safety | 4 weeks | ~25 rules live |
| 3. Tier 2 deprescribing | 8 weeks | ~75 rules live |
| 4. Tier 3 quality gaps + Authorisation evaluator | 6 weeks | ~50 rules live + Auth evaluator operational |
| 5. Tier 4 surveillance + ScopeRules | 3 weeks | ~50 rules live + ScopeRules deployed |
| 6. Continuous tuning | Ongoing | Rule library stays current |
| **Total to MVP** | **~25 weeks** | **~200 rules + substrate-aware infrastructure** |

This is roughly 13 weeks longer than v1.0's estimate (12 weeks). The extra time is the Authorisation evaluator build, ScopeRules deployment, and substrate-aware authoring overhead. **This is real and should not be underestimated.**

---

## Part 5.5 — ScopeRules-as-data architecture

ScopeRules cannot be hardcoded in CQL. They must be data, not code. Here's why and how.

### 5.5.1 Why data not code

**Reason 1 — Frequency of change.** Australian medication regulation changes faster than database schemas. Victorian PCW exclusion is the first of many likely state-level changes. NSW, QLD, SA branches of ANMF have all advocated for similar restrictions. If ScopeRules are hardcoded in CQL, every state-level change becomes an engineering project.

**Reason 2 — Cross-cutting consumption.** ScopeRules are consumed by the Authorisation evaluator, the rule firing layer, the user surface (which legal actions are visible to which user), the audit trail (what actions were performed under what authority), and the regulator API. Hardcoding in CQL means duplication or fragility.

**Reason 3 — Auditability.** Regulators need to verify that the platform's authorisation logic correctly implements the legislation. ScopeRules-as-data, parsed from authoritative sources with attribution, are auditable. Hardcoded CQL is not.

### 5.5.2 The data structure

ScopeRules live in a database (not in CQL files), with the following schema:

```yaml
scope_rule:
  id: "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01"
  jurisdiction: "AU/VIC"
  category: "medication_administration_scope_restriction"
  effective_period:
    start_date: "2026-07-01"
    end_date: null
    grace_period_days: 90
  
  applies_to:
    role: "PCW"
    action_class: "administer"
    medication_schedule: ["S4", "S8", "S9"]
    medication_class_includes:
      - "antibiotics"
      - "opioid_analgesics"
      - "benzodiazepines"
      - "drugs_of_dependence"
    resident_self_administering: false
  
  evaluation:
    decision: "denied"
    fallback_required: true
    fallback_eligible_roles: ["RN", "EN", "Pharmacist", "Medical_practitioner"]
  
  source:
    legislative_reference: "Drugs, Poisons and Controlled Substances Amendment 
                            (Medication Administration in Residential Aged 
                            Care) Act 2025"
    source_id: "DPCS-2025-VIC"
    source_version: "1.0"
    source_url: "https://www.legislation.vic.gov.au/in-force/acts/..."
  
  audit:
    recordkeeping_required: true
    recordkeeping_period_years: 7
```

### 5.5.3 Where ScopeRules are parsed from

Layer 1 v2 sources Category C (Regulatory and Authority Sources) feed ScopeRules:
- Aged Care Act 2024 + Strengthened Quality Standards → governance and clinical care ScopeRules
- Restrictive Practice regulations → consent and behaviour-support ScopeRules
- Victorian PCW exclusion legislation → Victorian-specific medication administration ScopeRules
- NMBA designated RN prescriber registration standard → RN prescribing scope ScopeRules
- Tasmanian pharmacist co-prescribing pilot → Tasmanian-specific ScopeRules
- ACOP credentialing requirements → ACOP credential ScopeRules
- Pharmacy Board autonomous prescribing (when activated) → pharmacist autonomous prescribing ScopeRules

Each is a different source authority with its own update cadence and authority tier. The Source Registry from Layer 1 v2 governs parsing and versioning.

### 5.5.4 The runtime evaluation pattern

When the Authorisation evaluator receives a query, it:

1. Looks up applicable ScopeRules for the (jurisdiction, role, action_class, medication_schedule, resident_id, action_date) tuple
2. Filters by effective_period (rules outside their effective dates are excluded)
3. Evaluates each applicable rule against the action context
4. Combines results: if any rule denies, action is denied; if all rules grant or grant-with-conditions, action is granted with the most restrictive condition set
5. Returns structured decision with full audit trail

This is sub-100ms typical, sub-500ms p95 with cold cache. The cache strategy (Section 4.5.3) keeps it fast under load.

### 5.5.5 Multi-jurisdiction expansion path

When other states pass PCW exclusion legislation:
1. Parse the new legislation into a ScopeRule
2. Deploy to the Authorisation evaluator
3. Affected facilities (in that jurisdiction) get the new ScopeRule applied automatically

No engineering work required. This is the architectural payoff of data-not-code.

---

## Part 6 — What can go wrong, and what to defend against

The six failure modes from v1.0 hold. v2.0 adds three:

**Failure 7: Substrate concurrency bugs.** The five state machines share one EvidenceTrace graph. Two state machines transitioning simultaneously must produce coherent linked entries. Without transactional integrity, the EvidenceTrace becomes inconsistent and the moat is lost.

**Defended by:** event sourcing pattern with eventual consistency tolerance; transactional boundaries scoped to per-state-machine writes with idempotent EvidenceTrace appends.

**Failure 8: Authorisation evaluator latency creep.** Every action attempt fires an evaluation. As rules accumulate, latency tends to grow. p95 budget breach kills the bedside experience.

**Defended by:** strict latency monitoring (alerts on p95 >300ms); cache strategy as designed; rule deduplication; rule precompilation where possible.

**Failure 9: ScopeRule misencoding.** A misencoded ScopeRule could deny legitimate actions or grant illegitimate ones. Both are catastrophic.

**Defended by:** dual-review governance for all ScopeRule changes (clinical pharmacist + medical director + legal review for jurisdictional rules); regression test suite covering all currently-deployed rules; staged rollout (silent mode first, then enforced).

---

## Part 7 — Three sharp recommendations

If your team can do only three things from this document, do these:

**1. Commit to Wave 0 substrate scoping before Wave 1 begins.**

The two-week cross-team scoping investment prevents months of rework downstream. Skipping it produces rule libraries that don't compose with the substrate, helpers that don't match what's available, test cases that don't reflect production behaviour. Two weeks of scoping is cheap; two months of debugging is not.

**2. Build the Authorisation evaluator as a separate optimised subsystem from V1, not as a CQL extension.**

The temptation to extend CQL with authorisation helpers and treat it all as one rule layer is strong because it simplifies tooling. Resist it. The Authorisation evaluator has different latency requirements, different audit requirements, different caching needs, different schema-evolution patterns. Build it separately, integrate via clean interfaces, and let each layer be optimal for its purpose.

**3. Treat ScopeRules as data from the first jurisdiction, not "we'll generalise later."**

Hardcoding Victorian PCW exclusion logic now and "generalising when other states pass legislation" is a trap. The other states will pass their legislation; the cost of retrofitting jurisdiction-aware rules to hardcoded logic is much higher than designing for it from the start. Same applies to designated RN prescribing scope, Tasmanian pilot scope, and ACOP credentialing.

---

## Part 8 — Closing

Three things to register before this document leaves the room.

**One:** Layer 3 v2 is structurally larger than Layer 3 v1.0. The substrate awareness, the Authorisation evaluator, the ScopeRules-as-data architecture — these are real additions, not cosmetic. The team should expect roughly 25 weeks to MVP coverage at ~200 rules + substrate-aware infrastructure, vs v1.0's 12-week estimate at ~200 rules. The extra time is the substrate dividend; without it, the rule library doesn't deliver the v2.0 product.

**Two:** the Authorisation evaluator is the new safety primitive that almost nobody in the Australian aged care market is building. The rule format, the latency budget, the cache strategy, the audit query API — these are non-trivial engineering investments. They're also what makes the platform regulatorily defensible for the regime arriving in 2026-2027 (Victorian PCW exclusion, designated RN prescribers, Tasmanian pharmacist co-prescribing). Don't underestimate this work; don't skip it; don't try to do it as a CQL extension.

**Three:** Layer 2 implementation guidelines are now blocking work for V1. The Clinical state machine, the running baselines with delta-on-write, the substrate entities with consistent identity — these are Layer 2 deliverables that Layer 3 v2 depends on. **Without Layer 2, Layer 3 v2 cannot be authored against the substrate.** I recommend writing Layer 2 implementation guidelines next, before any Layer 3 Wave 0 work begins.

What the platform becomes when Layer 3 v2 works: the canonical Australian aged care medication rule library, substrate-aware, jurisdiction-aware, audit-defensible, with a 7-day SLA from authoritative source publication to deployed rule, integrated via FHIR Clinical Reasoning and CDS Hooks v2.0 standards. Most competitors are doing snapshot reasoning, hardcoded jurisdiction logic, manual rule authoring with no source attribution, and quarterly audit reporting. The team is building the rule layer for clinical reasoning continuity infrastructure — and the Authorisation evaluator beneath it is the safety primitive nobody else has.

In the next document I'll write Layer 2 implementation guidelines covering the substrate that Layer 3 v2 depends on.

— Claude
