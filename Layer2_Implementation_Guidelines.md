# Layer 2 Implementation Guidelines — The Substrate for Clinical Reasoning Continuity

**Version:** 1.0
**Date:** April 2026
**Status:** Implementation guidelines for the substrate layer that Layer 3 v2 depends on, sitting between Layer 1 (sources) and Layer 3 (rules)

**Companion documents:**
- *Vaidshala Final Product Proposal v2.0 Revision Mapping*
- *Layer 1 v2 Australian Aged Care Implementation Guidelines* (the sources Layer 2 ingests from)
- *Layer 3 v2 Rule Encoding Implementation Guidelines* (the rules Layer 2 supplies state to)

**Audience:** Engineering leads, clinical informatics, data engineering, anyone building or operating the substrate.

**Author:** Claude (Anthropic), grounded in the verified Australian regulatory landscape, the FHIR Clinical Reasoning and Provenance modules, the published clinical instruments (CFS v2.0, AKPS, DBI, ACB), and the architectural principles established in the v2 product proposal.

---

## Part 0 — Why this document is the load-bearing piece

If Layer 1 is what we ingest and Layer 3 is what we reason with, Layer 2 is what makes the reasoning *continuous*. It's the layer where the substrate entities live, where the Clinical state machine maintains running baselines, where transition events propagate, where the EvidenceTrace graph is constructed.

Three reasons this document is more architecturally important than the others:

**One — without the substrate, rules fire on snapshots.** Snapshot reasoning is the cause of the 96% override rate that destroys CDS systems. Layer 3 v2's baseline-aware rule examples (`PotassiumDeltaFromBaseline7Days > 0.8`) are only possible if Layer 2 maintains those baselines. The clinical signal-to-noise ratio of the entire platform depends on getting Layer 2 right.

**Two — without the substrate, the EvidenceTrace graph cannot be built.** The whole moat — the queryable longitudinal record of who decided what based on what evidence — requires substrate entities with consistent identity, stable lineage, and bidirectional graph edges. This is not retrofittable. It has to be designed in from the start.

**Three — without the substrate, the Authorisation evaluator has nothing to evaluate against.** The runtime authorisation queries Layer 3 v2 specified ("does this RN have a current prescribing agreement covering this medication for this resident?") need real-time access to credentials, agreements, consents, and patient state. Those live in Layer 2.

This document specifies what Layer 2 has to do, how it has to do it, and what's hardest about doing it well. It is roughly 20 weeks of engineering work, sized for 3-4 dedicated engineers plus clinical informatics partnership. **Build this before V1 begins. Do not skip it. Do not defer it. Do not attempt to reverse-engineer it from rule library bugs in production.**

---

## Part 1 — The substrate entities, in detail

The five-state-machine substrate references six core entities. Each is more than a schema definition — each is a clinical claim about how aged care medication management actually works.

### 1.1 Resident

The subject of all clinical reasoning. Maps to FHIR `Patient` with extensions.

**Standard FHIR Patient elements:** identifiers, name, demographics, address, telecom.

**Aged-care-specific extensions** (Vaidshala-defined):

```yaml
resident_extensions:
  ihi: "Individual Healthcare Identifier (16-digit national identifier)"
  facility_id: "current RACF identifier"
  admission_date: "date admitted to current RACF"
  prior_admission_date: "for transfers between RACFs"
  
  cognitive_capacity:
    status: "intact | impaired | unable_to_assess"
    last_assessed: "date"
    assessor_role: "GP | geriatrician | aged_care_assessment_team"
    instrument: "MMSE | MoCA | clinical_judgement | other"
    score: "if instrument has numeric output"
    capacity_for: "list of capacity-specific findings (medical, financial, accommodation)"
  
  care_intensity:
    tag: "active_treatment | rehabilitation | comfort_focused | palliative"
    effective_date: "when this tag became current"
    documented_by: "GP | care_manager | family_meeting"
    review_due: "next scheduled care planning meeting"
  
  restrictive_practice_authorisation:
    active_authorisations: "list of currently-authorised restrictive practices"
    last_review: "date"
    next_review_due: "date"
  
  language_and_cultural:
    primary_language: "for consent and education materials"
    cultural_considerations: "structured + free text"
    interpreter_required: "boolean"
```

### 1.2 Person + Role

A *Person* is a human (clinician, family member, resident themselves, or system actor). A *Role* is what they're authorised to do at this facility, on this resident, today. Maps loosely to FHIR `Practitioner` + `PractitionerRole`, but with substantial extensions.

**Why Role is separate from Person:** the same Person can hold multiple Roles. An RN can also be a designated RN prescriber; can also be a family-member-SDM for a different resident at a different facility. A pharmacist can be both an ACOP-credentialed pharmacist and a community dispensing pharmacist. Roles, not Persons, carry scope and credentials.

**Critical Role types:**

```yaml
role_types:
  # Clinical roles
  - GP                              # General practitioner
  - nurse_practitioner              # Autonomous since Nov 2024
  - geriatrician                    # Specialist
  - acop_pharmacist                 # APC training required from July 2026
  - dispensing_pharmacist           # Community pharmacy
  - designated_RN_prescriber        # Endorsement live since 30 Sept 2025
  - tasmanian_pharmacist_co_prescriber  # Tasmanian pilot 2026-2027
  - registered_nurse
  - enrolled_nurse
  - personal_care_worker            # VIC restrictions from July 2026
  
  # Non-clinical roles
  - resident_self                   # Resident as actor
  - substitute_decision_maker       # SDM with scope and legal basis
  - family_supporter                # Without SDM authority
  - facility_care_manager           # DON or equivalent
  - facility_administrator          # CEO, regulator-facing
  
  # System actors
  - system_rule_engine              # CQL rule fires
  - system_authorisation_evaluator  # ScopeRule decisions
  - system_extraction_pipeline      # L0-L6 actions
```

**Role attributes:**

```yaml
role:
  person_ref: "Reference(Person)"
  role_type: "from list above"
  facility_scope: "list of facilities where this role applies"
  resident_scope: "list of residents where this role applies (if scoped)"
  
  validity_period:
    valid_from: "date"
    valid_to: "date or null"
    
  credentials:
    - credential_type: "ACOP_APC_training | NMBA_RN_prescriber_endorsement | GP_AHPRA | etc."
      credential_id: "identifier"
      valid_from: "date"
      valid_to: "date"
      evidence_url: "link to verifiable source"
      verification_status: "verified | pending | expired | revoked"
      verified_by: "Person reference"
      verified_date: "date"
  
  scope_constraints:
    medication_classes_authorised: "list (only for prescribing roles)"
    actions_authorised: "list (administer, prescribe, observe, recommend, consent_witness, etc.)"
    jurisdictional_scope: "AU | AU/VIC | AU/TAS | etc."
```

**The SubstituteDecisionMaker special case:** SDMs in Australia have varying legal authority depending on jurisdiction and basis of appointment. Statutory next-of-kin SDMs have different scope from tribunal-appointed guardians from enduring guardianship appointees. The platform models SDM as a Role with explicit `legal_basis` and `scope_constraints`:

```yaml
role:
  role_type: "substitute_decision_maker"
  scope_constraints:
    legal_basis: "statutory_next_of_kin | enduring_guardianship | tribunal_appointed | other"
    decision_scope: "list (medical, financial, lifestyle, restrictive_practice)"
    jurisdictional_scope: "varies by state (each state has different rules)"
  credentials:
    - credential_type: "guardianship_order"
      credential_id: "VCAT/QCAT/etc. order number"
      evidence_url: "link to scanned order"
```

### 1.3 MedicineUse

This is the single most important entity in the system. v1.0 of these proposals treated medication as essentially the eNRMC's MedicationRequest. v2.0 says: that's not enough.

**The structural insight:** the eNRMC carries *what* and *how much*. It does not carry *why*, *for how long*, *toward what target*, or *under what stop criteria*. That gap is what causes deprescribing recommendations to fail — the next prescriber can't tell whether the medication is achieving what it was started for, or whether it's still needed, because the *intent* was never recorded.

MedicineUse extends FHIR `MedicationRequest` with intent, target, and stop criteria as first-class fields:

```yaml
medicine_use:
  # Standard MedicationRequest fields
  medication: "AMT code (Australian Medicines Terminology)"
  dose: "structured dose"
  route: "structured route"
  frequency: "structured frequency"
  start_date: "date"
  
  # Standard but often-missing fields
  prescriber_ref: "Person reference"
  prescriber_role_at_time: "Role reference (snapshotted at prescription time)"
  authorising_jurisdiction: "AU | AU/VIC | AU/TAS"
  
  # Vaidshala-specific extensions
  intent:
    primary_indication: "structured (SNOMED CT-AU code)"
    indication_text: "free-text rationale"
    intent_class: "treatment | prophylaxis | symptom_control | replacement | comfort"
    expected_benefit_horizon_months: "integer (clinical estimate)"
    
  target:
    target_type: "BP_below | HbA1c_below | symptom_relief | none_specified"
    target_value: "structured (where applicable)"
    target_measurement: "Observation type to measure against target"
    
  stop_criteria:
    planned_duration_months: "integer (where applicable, e.g., antibiotic 7 days)"
    review_due_date: "date"
    automatic_stop_conditions: "list of structured conditions"
    deprescribing_trial_attempted_dates: "list (if any prior deprescribing attempts)"
    deprescribing_trial_outcomes: "list (success | failure | partial)"
    
  context:
    started_during_event_ref: "Event reference (if started during hospital stay, infection, etc.)"
    is_substrate_for_restrictive_practice: "boolean"
    restrictive_practice_authorisation_ref: "if applicable"
    
  governance:
    last_reviewed: "date"
    last_reviewed_by_ref: "Role reference"
    next_review_due: "date"
    review_outcome_history: "list of structured review entries"
```

**A worked example — Mary's risperidone:**

```yaml
medicine_use:
  medication: "AMT_RISPERIDONE_0.25MG"
  dose: "0.25 mg"
  route: "oral"
  frequency: "twice daily"
  start_date: "2026-02-15"
  prescriber_role_at_time: "GP_DR_CHEN_2026"
  
  intent:
    primary_indication: "SNOMED_BPSD_AGITATION"
    indication_text: "Severe agitation with verbal aggression toward care staff,
                      after documented 4-week non-pharmacological behaviour support
                      trial (sensory garden, music therapy, life history work)"
    intent_class: "symptom_control"
    expected_benefit_horizon_months: 3  # Reassess at 3 months per ADG 2025
    
  target:
    target_type: "symptom_relief"
    target_value: "agitation episodes < 1 per day"
    target_measurement: "behavioural_chart_agitation_episode_count"
    
  stop_criteria:
    planned_duration_months: 3  # Antipsychotic in dementia, per ADG 2025
    review_due_date: "2026-05-15"
    automatic_stop_conditions:
      - "Three consecutive days zero agitation episodes"
      - "Resident withdrawal of consent"
      - "SDM withdrawal of consent"
      - "Care intensity transition to palliative (then re-evaluate)"
      
  context:
    is_substrate_for_restrictive_practice: true
    restrictive_practice_authorisation_ref: "Authorisation_RP_2026_0042"
    
  governance:
    last_reviewed: "2026-02-15"
    last_reviewed_by_ref: "GP_DR_CHEN_2026"
    next_review_due: "2026-03-15"  # First monthly review
    review_outcome_history: []
```

**Why this matters:** when a Layer 3 rule fires recommending review of risperidone, the rule has access to *all of this*. It knows the indication, the target, the planned duration, the stop criteria, the consent state, the review history. The recommendation that surfaces to the GP doesn't say "cease risperidone"; it says "scheduled review due. Resident on risperidone for BPSD agitation, target was <1 episode/day. Behavioural chart shows zero episodes for past 14 days. Stop criteria met. Recommend deprescribing trial per planned protocol."

That's the difference between a workflow tool and clinical reasoning continuity infrastructure.

### 1.4 Observation

A clinical fact about the resident at a time. Maps to FHIR `Observation` with one critical extension: every observation has an associated *delta from baseline* computed on write.

**Standard FHIR Observation fields** apply. The Vaidshala-specific extension:

```yaml
observation_extensions:
  baseline_at_write_time:
    baseline_value: "the running baseline value at the time this observation was recorded"
    baseline_window_days: "the lookback window used (typically 14 or 30 days)"
    baseline_n_observations: "how many observations contributed to the baseline"
    baseline_confidence: "high | medium | low (function of n and variance)"
    
  delta:
    absolute_delta: "this_value - baseline_value"
    relative_delta_percent: "absolute_delta / baseline_value * 100"
    direction: "increasing | decreasing | unchanged"
    velocity: "rate of change over the baseline window"
    
  trajectory:
    is_trending: "boolean (true if 3+ consecutive observations in same direction)"
    consecutive_same_direction_count: "integer"
    
  context_at_write:
    care_intensity_at_time: "snapshot of resident.care_intensity at observation time"
    active_concerns_at_time: "list of active clinical concerns at the time"
    recent_medication_changes_within_days: "list of MedicineUse changes in past 14 days"
```

**Why compute on write:** if delta is computed on read, every read incurs the cost. If multiple rules read the same observation, the work gets duplicated. If the baseline shifts after the observation is written but before it's read, deltas become inconsistent across reads. Compute-on-write fixes all of this — the observation gets a stable, point-in-time delta that doesn't change retroactively.

**The cost of compute-on-write:** every observation insert triggers a baseline computation. For 200-bed facilities with 5-10 observations per resident per day = 1,000-2,000 observation inserts per facility per day. The baseline computation itself is cheap (running median over last 14-30 days), but at scale this requires careful indexing and probably a streaming pipeline (Apache Flink or Kafka Streams) rather than ad-hoc database triggers.

### 1.5 Event

Something happened. A fall, a hospital admission, an antibiotic course started, a behavioural incident, a refusal, a death. Events are first-class because they're the trigger surface for almost everything else.

**Event vs Observation:** Observations are clinical facts about the resident's state. Events are things that occurred and have legal, regulatory, or workflow significance. A fall is an Event (mandatory reporting under Quality Indicator Program); the post-fall blood pressure is an Observation.

**Event types:**

```yaml
event_types:
  # Clinical events
  - fall                    # QI Program reportable
  - pressure_injury         # QI Program reportable
  - behavioural_incident    # Restrictive practice trigger
  - medication_error        # Serious Incident Response Scheme
  - adverse_drug_event      # Causality assessment trigger
  
  # Care transitions
  - hospital_admission      # MHR-trackable
  - hospital_discharge      # Reconciliation trigger
  - GP_visit
  - specialist_visit
  - emergency_department_presentation
  - end_of_life_recognition # Care intensity transition
  - death
  
  # Administrative events
  - admission_to_facility
  - transfer_between_facilities
  - care_planning_meeting
  - family_meeting
  
  # System events (for EvidenceTrace)
  - rule_fire
  - recommendation_submitted
  - recommendation_decided
  - monitoring_plan_activated
  - consent_granted_or_withdrawn
  - credential_verified_or_expired
```

**Event attributes:**

```yaml
event:
  event_type: "from list above"
  occurred_at: "datetime"
  occurred_at_facility: "Facility reference"
  resident_ref: "Resident reference"
  
  reported_by_ref: "Role reference (who logged this Event)"
  witnessed_by_refs: "list of Role references"
  
  severity: "minor | moderate | major | sentinel (SIRS)"
  
  description_structured: "structured details per event type"
  description_free_text: "free text"
  
  related_observations: "list of Observation references generated by/around this event"
  related_medication_uses: "list of MedicineUse references implicated"
  
  triggered_state_changes:
    - state_machine: "Recommendation | Monitoring | Authorisation | Consent | ClinicalState"
      state_change: "structured state transition"
      
  reportable_under:
    - "QI Program"
    - "Serious Incident Response Scheme"
    - "Coroner"
    - "ACQSC complaint trigger"
    - "etc."
```

### 1.6 EvidenceTrace

The architectural moat. Not a log — a queryable graph.

**What v1.0 specified:** every state transition logs to KB-18 governance audit trail.

**What v2.0 needs:** every state transition writes a structured node to a graph that supports bidirectional queries. Forward: given a recommendation, what did it produce? Backward: given an outcome, what reasoning produced it?

**The dual-resource pattern (FHIR-aligned):**

The team should use the FHIR Provenance + AuditEvent dual-resource pattern, extended for clinical reasoning continuity:

- **Provenance** records *resource changes* — who modified which entity, when, with what authority, based on what inputs. This is the resource-history layer.
- **AuditEvent** records *system events* — security, access, query, login. This is the operational-logging layer.
- **EvidenceTrace** (Vaidshala-specific, sits on top) records *clinical reasoning chains* — the bidirectional graph linking observations to interpretations to recommendations to decisions to outcomes. This is the moat.

**EvidenceTrace node structure:**

```yaml
evidence_trace_node:
  id: "globally unique"
  state_machine: "Authorisation | Recommendation | Monitoring | ClinicalState | Consent"
  state_change_type: "transition (from_state -> to_state)"
  
  recorded_at: "datetime"
  occurred_at: "datetime (may differ from recorded_at)"
  
  actor:
    role_ref: "Role reference at the time of action"
    person_ref: "Person reference"
    authority_basis_ref: "Credential or PrescribingAgreement that authorised this action"
  
  inputs:
    - input_type: "Observation | MedicineUse | Event | Condition | Consent | ScopeRule | Rule | other"
      input_ref: "reference to the input entity"
      role_in_decision: "supportive | primary_evidence | secondary_evidence | counter_evidence"
  
  reasoning_summary:
    text: "structured rationale"
    rule_fires: "list of rule_ids that contributed"
    suppressions_evaluated: "list of suppression_ids that were checked"
    suppressions_fired: "list of suppression_ids that suppressed"
    alternatives_considered: "list of alternative actions considered but not taken"
    alternative_selection_rationale: "why this option vs alternatives"
  
  outputs:
    - output_type: "Recommendation | MonitoringPlan | RecommendationStateChange | etc."
      output_ref: "reference to the output entity"
  
  graph_edges:
    upstream_traces: "list of EvidenceTrace nodes that led to this one"
    downstream_traces: "list of EvidenceTrace nodes this one led to"
```

**The bidirectional graph requirement:**

Every node has explicit upstream and downstream edges. Querying forward (given X, what did it produce?) traverses downstream edges. Querying backward (given Y, what produced it?) traverses upstream edges. **This is the architectural commitment that v1.0 didn't make.**

The graph cannot live in a write-only audit log table — that supports only one-direction queries. It needs:
- A graph database (Neo4j, AWS Neptune) OR
- A relational store with explicit edge tables and graph-query indexes OR
- An event-sourcing pattern with materialized views for common query patterns

**My recommendation:** start with the relational-with-edge-tables approach. It's simpler operationally, and the query patterns are stable enough to make materialized views work. Move to graph database only if query patterns prove genuinely too complex (which I don't think they will).

---

## Part 2 — The Clinical state machine

The Clinical state machine is the slowly-evolving baseline of who this resident is, against which observations become signal. It's the substrate's most clinically important component.

### 2.1 What the Clinical state maintains

For each resident, the Clinical state machine maintains four kinds of structured data:

**(a) Per-observation-type running baselines.** For each Observation type that has a meaningful baseline (vital signs, weight, lab values, behavioural observations, mobility scores), the state machine maintains a running median over a defined window with confidence interval.

**(b) Functional baseline.** Mobility, cognition, behaviour, intake, sleep — composite functional status with longitudinal track.

**(c) Active concerns.** Open clinical questions with expected resolution windows. "Post-fall watching for delayed head injury 72h." "Antibiotic course day 4 of 7, watching for C. diff."

**(d) Care intensity tag.** Active treatment / rehabilitation / comfort focused / palliative — with effective date and review schedule.

### 2.2 Running baseline computation

**Algorithm (per observation type, per resident):**

```
On observation insert:
  1. Look up the past N observations of this type for this resident,
     where N = max(14 days, last 5 observations)
  2. Compute running median of these values
  3. Compute interquartile range as confidence proxy
  4. Set baseline_value = running_median
  5. Set baseline_confidence = 
        if n_observations >= 7 and IQR < 25% of median: high
        elif n_observations >= 4 and IQR < 50% of median: medium
        else: low
  6. Compute delta:
     absolute_delta = this_value - baseline_value
     relative_delta_percent = (this_value - baseline_value) / baseline_value * 100
  7. Detect trajectory:
     if last 3 observations all in same direction from baseline: trending
     else: stable
  8. Write Observation with baseline + delta + trajectory attached
```

**Special cases:**

- **First observations of a type for a resident** (no baseline yet): write with `baseline_confidence: insufficient_data`. Rules should not fire on first 3-5 observations of a type.
- **Observation type with high physiological variation** (e.g., blood pressure, which varies hour-to-hour): use rolling 30-day window and exclude observations within 30 minutes of recent activity (mobilisation, distress).
- **Observation type with acute-period contamination** (e.g., post-fall vitals): exclude observations within active concern windows where the concern is acute (24-72h post-fall, post-infection).

**Per-observation-type baseline configuration (sample):**

```yaml
baseline_configurations:
  - observation_type: "potassium"
    baseline_window_days: 14
    minimum_observations_for_high_confidence: 4
    exclude_during_active_concerns: ["AKI_watching", "IV_fluid_resuscitation"]
    
  - observation_type: "systolic_blood_pressure"
    baseline_window_days: 30
    minimum_observations_for_high_confidence: 21  # Daily readings
    exclude_during_active_concerns: ["acute_pain", "infection"]
    morning_only: true  # Avoid post-meal/post-activity confounding
    
  - observation_type: "weight"
    baseline_window_days: 90  # Slow-changing
    minimum_observations_for_high_confidence: 4
    
  - observation_type: "behavioural_agitation_episode_count"
    baseline_window_days: 14
    minimum_observations_for_high_confidence: 7  # Daily charting
    exclude_during_active_concerns: ["acute_infection_24h", "post_fall_24h"]
    
  - observation_type: "egfr"
    baseline_window_days: 90
    minimum_observations_for_high_confidence: 3
    flag_velocity: true  # Decline of >20% in 14 days is itself a signal
```

### 2.3 Active concerns

An active concern is an open clinical question that:
- Has a clear start trigger (event, observation, or recommendation)
- Has an expected resolution window (e.g., "watching for delayed head injury for 72 hours")
- Has an owner role (typically RN, ACOP pharmacist, or GP)
- May have specific monitoring requirements (linked MonitoringPlan)
- Resolves either by stop_criteria being met, by escalation to a new concern, or by expiry without resolution

**Active concern types (sample):**

```yaml
active_concern_types:
  - post_fall_72h
  - post_hospital_discharge_72h
  - antibiotic_course_active
  - new_psychotropic_titration_window
  - acute_infection_active
  - end_of_life_recognition_window
  - post_deprescribing_monitoring
  - pre_event_warning_window  # When trajectory crosses warning threshold
  - awaiting_consent_review
  - awaiting_specialist_input
```

**Why this matters for rules:** active concerns suppress some rules and trigger others. A rule "PPI long-term without indication" should suppress while `post_acute_GI_bleed_watching` is active. A rule "fall risk reassessment" should fire when `post_fall_72h` resolves without resolution (i.e., the 72h passes, monitoring expires, but no formal reassessment was logged).

### 2.4 Care intensity tag

This is the single most important piece of context for shaping clinical recommendations. The same recommendation has different framing depending on care intensity.

**The four care intensity tags:**

| Tag | Description | Implication for recommendations |
|---|---|---|
| `active_treatment` | Resident has active rehabilitation goals, expected longer life, full treatment intensity | Standard CDS rules apply; cardiovascular and cognitive considerations weighted normally |
| `rehabilitation` | Resident is recovering from acute event, focus on functional restoration | Time-limited intensity; specific recovery milestones; deprescribing of acute-event medications expected |
| `comfort_focused` | Frailty progressing, treatment intensity reduced but not palliative | Deprescribing of preventive medications appropriate; symptom focus; quality-of-life weighted heavily |
| `palliative` | End-of-life recognized, comfort care primary goal | Deprescribing of all non-symptom medications expected; new medications only for symptom control; restrictive-practice authorisations re-examined |

**The transition between tags is itself an Event** that propagates through the substrate:
- Active concerns may resolve or become invalid (rehabilitation goals don't apply in palliative)
- Existing recommendations may be re-evaluated (statin deprescribing becomes immediately appropriate)
- Existing monitoring plans may be revised (BP monitoring may become irrelevant)
- Consent state may need refresh (palliative care implies different conversation about continuing antipsychotics)

**Operational scoring tools:**

The care intensity tag is set by clinical judgement (typically GP + family meeting + care manager), but should be informed by structured prognostic and functional measures:

- **Clinical Frailty Scale (CFS) v2.0** (Rockwood, 2020 revision) — 9-point scale, validated for Australian aged care, externally validated for 90-day mortality prediction. CFS ≥7 is a flag for considering comfort-focused or palliative tagging.
- **Australia-Modified Karnofsky Performance Status (AKPS)** — Australian-developed (Abernethy 2005, Flinders), 11-level scale 0-100, blends KPS and TKPS for any care setting. AKPS ≤40 is a flag for palliative tagging.
- **Aged Care Assessment Tool (ACAT)** outputs — for funding-level assessment, but also captures functional status that informs care intensity.

**These are NOT used to automate care intensity transitions.** Care intensity is a clinical judgement, often made in conversation with family. The substrate captures the score, surfaces it to the clinical team, and provides decision support — but the tag transition is a human-authored Event.

### 2.5 Capacity assessment

Cognitive capacity is dynamic. The substrate captures it as a separate object (not a single resident attribute) because:
- It changes (capacity can be lost permanently or temporarily)
- It's domain-specific (medical decisions, financial decisions, accommodation decisions all have different capacity standards)
- It's date-stamped and has an assessor
- It interacts with consent state (a resident with capacity gives consent themselves; without capacity, the SDM does)

```yaml
capacity_assessment:
  resident_ref: "Resident reference"
  assessed_at: "datetime"
  assessor_role_ref: "Role reference"
  
  domain: "medical_decisions | financial | accommodation | restrictive_practice | medication_decisions"
  
  instrument: "MMSE | MoCA | clinical_judgement | other"
  score: "if instrument has numeric output"
  
  outcome: "intact | impaired | unable_to_assess"
  
  duration: "permanent | temporary | unable_to_determine"
  expected_review_date: "if temporary"
  
  rationale: "structured + free text"
  
  interactions_with_consent_state:
    - "if outcome=impaired, consent state for this domain shifts to SDM-authorised"
    - "if outcome=intact, resident-self-consent is the path"
```

---

## Part 3 — The patient-state plumbing pipeline

Layer 1 v2 specified the *sources* — eNRMC, MHR, hospital discharge summaries, dispensing pharmacy, nursing observations. This part specifies how those sources flow into the substrate.

### 3.1 The pipeline shape

```
SOURCES                          PIPELINE STAGES                    SUBSTRATE
                                                                    
eNRMC (FHIR/CSV) -------┐                                           
                        ├-> [Ingestion] -> [Normalisation] -> [Substrate Write]
MHR FHIR Gateway -------┤                       |                       |
                        ├-> [Ingestion]    [Coding map]                 ↓
Hospital discharge -----┤                  [Identity match]      MedicineUse
(PDF + MHR-structured)  │                  [Deduplication]       Observation (delta-on-write)
                        │                  [Conflict resolution]  Event
Pathology --------------┤                       |                       |
(MHR Sharing by         │                       ↓                       ↓
Default July 2026)      │                  [Validation]            Clinical state
                        │                                          machine update
Care management --------┤                                              |
system (Leecare/         │                                              ↓
AutumnCare/etc.)        │                                          EvidenceTrace
                        │                                          graph node
Dispensing pharmacy ----┤
(DAA timing)            │
                        │
Behavioural charts -----┘
```

### 3.2 Ingestion strategy by source

This is where the rubber meets the road. Each source has different characteristics:

**eNRMC ingestion:**

- **Phase 1 (MVP):** CSV export, nightly sync. Simple, fragile, supports any vendor.
- **Phase 2 (V1):** FHIR R4 API for top 2-3 conformant vendors (Telstra Health MedPoint, MIMS, ResMed Software). Real-time or near-real-time.
- **Phase 3 (V2):** HL7 v2 ORM/RDE messaging where available; broader vendor coverage.

**Critical clinical data quality concern:** eNRMCs often have empty `indication` fields (the *why* is not captured). Vaidshala's MedicineUse extension fields (intent, target, stop criteria) are populated through:
1. NLP on nursing progress notes (low confidence, requires confirmation)
2. ACOP pharmacist structured entry during medication review (high confidence)
3. Inheritance from prescribing context (e.g., medication started during a hospital admission inherits acute-illness intent)

**MVP cannot wait for full intent capture.** Rules that depend on intent fields should suppress when intent is unknown, not fire on assumption. As ACOP reviews accumulate, intent coverage improves.

**MHR FHIR Gateway ingestion:**

The Modernising My Health Record (Sharing by Default) Act 2025 makes pathology and diagnostic imaging mandatory upload from 1 July 2026. For Layer 2:

- **Pre-July 2026:** MHR coverage is voluntary, ~70% of consumers. Per-pathology-vendor HL7 integration is the fallback for non-MHR-using residents.
- **Post-July 2026:** MHR coverage approaches 100% of consenting consumers. Single integration through MHR FHIR Gateway provides pathology for all.

**Implementation reality:** MHR production is currently SOAP/CDA. The FHIR Gateway (ADHA published IG v1.4.0, R4) is the modern path but still maturing. Plan:
- **MVP:** SOAP/CDA via MHR B2B Gateway (existing standard, mature)
- **V1:** FHIR Gateway for new MHR features as ADHA matures it
- **V2:** Full FHIR R4 transition

**Hospital discharge ingestion:**

Three coding systems, three timestamp regimes, three signers. This is the hardest single integration challenge in Layer 2.

- **MVP:** PDF discharge summary upload + manual reconciliation interface for the pharmacist. The platform displays the discharge medication list alongside the pre-admission RACF eNRMC list, highlights changes, and lets the pharmacist confirm/modify.
- **V1:** MHR-pulled structured discharge documents (where available) + automated diff against eNRMC + change-flagging + automatic ACOP routing within 24h. NLP for indication extraction from discharge notes.
- **V2:** Direct hospital ADT feeds where partnerships allow. This is jurisdictional and politically complex.

**The reconciliation algorithm (V1):**

```
On hospital discharge event:
  1. Pull discharge medication list (from MHR or PDF)
  2. Pull pre-admission RACF eNRMC medication list
  3. Normalise both to AMT codes
  4. Compute diff:
     - new_medications: in discharge, not in pre-admission
     - ceased_medications: in pre-admission, not in discharge
     - dose_changes: same medication, different dose
     - unchanged: in both, same dose
  5. For each diff entry, classify:
     - acute_illness_temporary (likely to be ceased on RACF readmission)
     - new_chronic (intent to continue long-term)
     - reconciled_change (replacement or dose adjustment)
     - unclear (requires pharmacist review)
  6. Generate reconciliation worklist for ACOP pharmacist
  7. ACOP confirms/modifies, then writes to substrate
  8. Each confirmed change becomes a MedicineUse event with intent populated
```

**Dispensing pharmacy ingestion:**

The unfashionable hard work. Australian community pharmacy software is fragmented (FRED, Z Solutions, Minfos, LOTS, Aquarius). No clean modern API.

- **MVP:** Manual DAA packing schedule entry by ACOP pharmacist; cessation/change alerts to dispensing pharmacy via fax/email.
- **V1:** API integration with FRED (largest vendor); structured DAA packing schedule and dispensing events.
- **V2:** Broader vendor coverage; full DAA composition tracking.

**The DAA timing data structure:**

```yaml
daa_packing_schedule:
  resident_ref: "Resident reference"
  dispensing_pharmacy_ref: "Pharmacy reference"
  
  packing_cycle: "weekly | fortnightly"
  packing_day_of_week: "saturday | sunday | monday | etc."
  
  current_pack:
    pack_id: "identifier"
    packed_at: "datetime"
    medications: "list of MedicineUse references with doses"
    delivery_to_facility_at: "datetime"
    
  next_pack:
    scheduled_packing: "datetime"
    composition_finalised: "boolean (true if no further changes accepted)"
    change_cutoff: "datetime (last time changes can be incorporated)"
    
  cessation_alerts_pending:
    - medication_ref: "MedicineUse reference"
      cessation_authorised_at: "datetime"
      taking_effect: "next_pack | manual_unpack_required"
      latency_days: "calculated"
```

This data structure lets the platform answer: "GP approved cessation Monday at 0900. The current DAA was packed Sunday. Mary will continue receiving the medication for 5 days until the next pack on Saturday, unless the dispensing pharmacy or RN manually unpacks the current DAA."

**Behavioural chart ingestion:**

Required under restrictive practice regulations for residents on antipsychotics. Currently a paper-based or care-management-system-internal data structure in most facilities.

- **MVP:** Structured manual entry by RN at point of care; nightly sync to substrate.
- **V1:** API integration with care management vendors (Leecare, AutumnCare, Person Centred Software).
- **V2:** Integration with behavioural charting apps (where deployed).

The behavioural chart is critical for Clinical state machine baselines for residents on psychotropics — without it, "agitation episode count baseline" cannot be computed and the rule library degrades.

### 3.3 Identity matching

A non-trivial problem. The same resident is identified differently across sources:

- IHI (Individual Healthcare Identifier, 16-digit national) — the gold standard
- Medicare number — common but not universal
- DVA number — if applicable
- RACF internal ID — vendor-specific
- Hospital MRN — different per hospital
- Dispensing pharmacy ID — different per pharmacy
- GP system ID — different per practice

**The identity matching service:**

```
On any incoming identifier from a source:
  1. Check if identifier is already mapped to a Resident in the substrate
  2. If yes: route data to that Resident
  3. If no: 
     - Try to match by IHI (high confidence)
     - Try to match by Medicare + name + DOB (medium confidence)  
     - Try to match by name + DOB + facility (lower confidence)
     - If no match: create new Resident with this identifier as primary;
       flag for manual identity verification
  4. Persist the identifier-to-Resident mapping for future routing
  5. Log all identity matches/non-matches to EvidenceTrace
```

**Identity matching errors are clinically dangerous.** A misrouted pathology result lands on the wrong resident's chart; a missed match means the resident's data is fragmented. The platform must:
- Use IHI as primary key wherever available (Sharing by Default Act 2025 makes IHI universally available)
- Flag low-confidence matches for human review
- Maintain audit trail of all match decisions
- Support post-hoc match correction with full data re-routing

### 3.4 The streaming pipeline

The volume justifies a streaming architecture. For a 200-bed facility:
- ~5-10 observations per resident per day = 1,000-2,000 observation events daily
- ~2-5 medication events per resident per day = 400-1,000 medication events daily
- Pathology, MHR updates, hospital events: variable

Across multiple facilities at scale, the platform handles tens of thousands of events daily. Apache Flink or Kafka Streams is the right architecture:

```
Source connectors (CSV, FHIR, HL7, MHR Gateway)
   ↓
Kafka topic: raw_inbound_events
   ↓
Stream processor: identity_matching
   ↓
Kafka topic: identified_events
   ↓
Stream processor: normalisation (coding map to AMT/SNOMED/LOINC AU)
   ↓
Kafka topic: normalised_events
   ↓
Stream processor: substrate_writer
   - For Observations: compute delta-on-write using current baseline state
   - For MedicineUse: detect changes against current state
   - For Events: classify and trigger downstream state machine updates
   ↓
Substrate database (relational + graph extensions)
   ↓
Stream processor: clinical_state_updater
   - Update running baselines
   - Update active concerns
   - Detect trajectory changes
   ↓
Kafka topic: substrate_updates
   ↓
Stream processor: rule_trigger_evaluator (Layer 3)
```

Each stage can scale horizontally. The substrate database is the single source of truth for current state; Kafka topics provide the audit history.

---

## Part 4 — The integration with state machines

This part specifies how Layer 2 substrate interacts with each of the five state machines.

### 4.1 With the Authorisation evaluator (Layer 3 v2 Part 4.5)

**What Layer 2 provides:**
- Person + Role lookups by ID
- Credential lookups with current validity status
- PrescribingAgreement lookups with current scope and mentorship status
- Resident-specific consent state for the queried action class
- Jurisdictional context for the resident's current facility

**Integration pattern:** the Authorisation evaluator queries Layer 2's substrate via a fast read API (target <50ms p95). Cache invalidation triggered by substrate writes (credential expiry, agreement amendment, consent withdrawal).

### 4.2 With the Recommendation state machine

**What Layer 2 provides:**
- Trigger events (medication changes, observation thresholds, baseline deltas, monitoring abnormalities)
- Resident clinical state context for recommendation rendering (care intensity, active concerns, baseline trajectories)
- MedicineUse intent/target/stop criteria for evaluating whether existing medications meet their targets

**Integration pattern:** Layer 2 emits substrate-update events that Layer 3 rules subscribe to. Recommendations are written back to the substrate as state-machine entities; the Recommendation lifecycle is itself substrate state.

### 4.3 With the Monitoring state machine

**What Layer 2 provides:**
- Expected observations vs received observations (the delta tells the Monitoring state machine what's overdue)
- Trajectory detection (3+ consecutive observations in same direction triggers `abnormal` state)
- Threshold crossings (configurable per monitoring plan)
- Active concern lifecycle (when concern resolves, related monitoring plan resolves)

**Integration pattern:** Layer 2 events stream into Monitoring state machine evaluators. MonitoringPlans live in the substrate; their state transitions are substrate writes.

### 4.4 With the Clinical state machine

The Clinical state machine *is* a Layer 2 component. Section 2 above is its specification.

### 4.5 With the Consent state machine

**What Layer 2 provides:**
- SubstituteDecisionMaker Role lookups (with current scope and validity)
- Capacity assessments (current and historical, by domain)
- Existing Consent objects (with active class, scope, expiry)

**Integration pattern:** Consent state queries against the substrate. Consent transitions (granted, withdrawn, expired) are substrate writes that propagate to dependent Recommendations (which may need re-evaluation if consent changes affect them).

---

## Part 5 — Implementation sequencing

Layer 2 build is roughly 20 weeks of engineering work. Sequencing matters because state machines depend on substrate, and rules depend on state machines.

### 5.1 Wave 1 — Substrate entities and basic ingestion (Weeks 1-6)

Build the six core entities (Resident, Person+Role, MedicineUse, Observation, Event, EvidenceTrace) with PostgreSQL backing + edge tables for graph queries. Implement basic CSV ingestion from at least one pilot eNRMC vendor.

**Deliverables:**
- Substrate schema deployed
- CSV ingestion operational from one eNRMC vendor
- Identity matching service operational
- EvidenceTrace graph queryable in both directions

**Wave 1 exit criterion:** can ingest a week of CSV exports from one pilot facility, produce a queryable resident profile, and trace any state change forward and backward.

### 5.2 Wave 2 — Clinical state machine (Weeks 7-12)

Build the Clinical state machine: running baselines with delta-on-write, active concerns, care intensity tagging, capacity assessment.

**Deliverables:**
- Per-observation-type baseline computation operational
- Delta-on-write at scale (handle 1,000+ observations/day per facility)
- Active concern lifecycle managed
- Care intensity tag with transition handling
- Capacity assessment objects with domain-specific scope
- CFS v2.0, AKPS, DBI, ACB scoring integration

**Wave 2 exit criterion:** running baselines exist for pilot facility residents; rules can query "is this observation abnormal relative to baseline?"; care intensity transitions propagate correctly through downstream state machines.

### 5.3 Wave 3 — MHR + pathology integration (Weeks 13-16)

Build MHR FHIR Gateway integration (initially for pathology, with hooks for future MHR information types).

**Deliverables:**
- MHR FHIR Gateway connection operational (SOAP/CDA in MVP, FHIR Gateway in V1)
- Pathology auto-ingestion for residents with MHR
- Per-pathology-vendor HL7 fallback for non-MHR residents

**Wave 3 exit criterion:** pathology results land in substrate within hours of being uploaded to MHR; baselines incorporate pathology results; rules can query lab trajectory.

### 5.4 Wave 4 — Hospital discharge reconciliation (Weeks 17-20)

Build the hospital discharge reconciliation workflow.

**Deliverables:**
- Discharge summary ingestion (PDF in MVP, MHR-structured in V1)
- Pre-admission vs discharge medication diff
- Reconciliation worklist for ACOP pharmacist
- Auto-routing within 24 hours of discharge event
- Intent capture during reconciliation (acute_illness_temporary vs new_chronic)

**Wave 4 exit criterion:** when a pilot resident returns from hospital, the platform produces a reconciliation worklist within hours; pharmacist completes reconciliation; substrate captures full intent for changed medications.

### 5.5 Beyond MVP — Wave 5 onwards (V1 and V2)

- Wave 5 (V1): Dispensing pharmacy DAA timing integration (FRED API)
- Wave 6 (V1): Behavioural chart structured ingestion
- Wave 7 (V2): Direct hospital ADT feed integration
- Wave 8 (V2): NLP for free-text indication extraction at scale
- Wave 9 (V2): Multi-vendor eNRMC FHIR coverage

### 5.6 Total Layer 2 effort summary

| Wave | Effort | Output |
|---|---|---|
| 1. Substrate entities | 6 weeks | Substrate operational with one pilot facility |
| 2. Clinical state machine | 6 weeks | Running baselines, delta-on-write, care intensity, capacity |
| 3. MHR + pathology | 4 weeks | Pathology auto-ingestion; SOAP/CDA + fallback |
| 4. Hospital discharge reconciliation | 4 weeks | Reconciliation workflow operational |
| **Total to MVP** | **~20 weeks** | **Substrate ready for Layer 3 v2 rule firing** |

This is a real investment. It's also non-negotiable — Layer 3 v2 cannot work without it.

---

## Part 6 — What's hardest, and what to defend against

Six failure modes specific to Layer 2:

**Failure 1: Compute-on-write performance.** Every observation insert triggers baseline computation. At scale this gets expensive. **Defended by:** streaming architecture (Flink/Kafka), per-resident-per-observation-type baseline caching, careful indexing, eventual consistency where acceptable.

**Failure 2: Identity match errors.** Misrouted data is clinically dangerous. **Defended by:** IHI-primary matching (universally available post-July 2026), confidence-tiered matching, manual review queue for low-confidence matches, audit trail with re-routing capability.

**Failure 3: Intent field sparseness.** eNRMCs don't capture intent. Rules that depend on intent suppress instead of firing on assumption. **Defended by:** progressive intent capture (NLP, ACOP review entry, prescribing context inheritance), graceful degradation when intent unknown, rules that explicitly handle "intent_unknown" as a state.

**Failure 4: Baseline contamination by acute periods.** A pneumonia week can shift a resident's baseline upward; once recovered, the new baseline is wrong. **Defended by:** active concern exclusion (don't include observations during acute concern windows), re-baselining after concern resolution, clinical review for low-confidence baselines.

**Failure 5: Care intensity transition lag.** Resident transitions to palliative care but the substrate doesn't know for days. Rules continue firing on `active_treatment` assumptions. **Defended by:** explicit transition events triggered by care planning meetings, family meetings, or end-of-life recognition; structured prompts during routine ACOP reviews; cross-cutting alerts when CFS or AKPS scores cross thresholds.

**Failure 6: EvidenceTrace graph query performance.** Bidirectional graph queries can be slow at scale. **Defended by:** materialized views for common query patterns, graph indexes, eventual consistency for non-critical queries, dedicated read replicas for regulator audit queries.

---

## Part 7 — Three sharp recommendations

**One — invest in the Clinical state machine before everything else after substrate entities.** Running baselines + delta-on-write are the single architectural commitment that determines whether rules fire on signal or noise. The 96% override rate that destroys CDS systems is largely a snapshot-reasoning problem. The Clinical state machine is the cure.

**Two — model MedicineUse intent + target + stop criteria as first-class fields, not extensions.** v1.0 thinking treated medication as a MedicationRequest. v2.0 says: that's the eNRMC's job. Vaidshala's job is the *why* — the intent, target, stop criteria — that makes deprescribing decisions clinically defensible. Without this, recommendations are "cease X"; with it, recommendations are "X was started for indication Y with target Z, stop criteria W are met, propose discontinuation."

**Three — build the EvidenceTrace as a queryable graph from day 1, not as an audit log to be retrofitted.** The temptation will be strong to log state changes and "build the graph later." Resist it. Bidirectional querying changes the schema, the indexing, the storage, the replication strategy. Retrofitting is 5x the cost of building it correctly at the start.

---

## Part 8 — Closing

Three things to register before this document leaves the room.

**One:** Layer 2 is the load-bearing piece of the platform. If Layer 1 is the sources and Layer 3 is the rules, Layer 2 is the substrate that makes the rules clinically meaningful. Without baseline-aware reasoning, the rule library produces alert noise; with it, the rule library produces clinical signal. Without the EvidenceTrace graph, there's no moat; with it, the platform owns a longitudinal dataset that doesn't exist anywhere else in Australian aged care.

**Two:** the 20-week build estimate is honest. A team that thinks they can build Layer 2 in 8 weeks is underestimating the complexity, particularly the streaming pipeline and the EvidenceTrace graph. A team that defers Layer 2 to "after MVP" is going to retrofit it under pressure later, at significantly higher cost. The right move is to commit the 20 weeks now, in parallel with Layer 1 ingestion work where possible.

**Three:** the hardest piece is the cross-cutting integration — the substrate has to support all five state machines simultaneously, with consistent identity, transactional integrity for state-machine writes, and bidirectional graph queries that perform at scale. None of these is impossible, but the team should expect the integration testing and performance optimisation to take real engineering time. **Plan for 4 additional weeks of hardening after the four waves complete.** Total realistic Layer 2 timeline: 24 weeks to production-ready.

What the platform becomes when Layer 2 works: the substrate that turns observation noise into clinical signal, the queryable longitudinal record of clinical reasoning that becomes the moat, the runtime layer that the Authorisation evaluator and the Recommendation engine and the Monitoring state machine all sit on. Most competitors are building rule libraries on top of snapshot data. The team is building the substrate that makes the rule library defensible — and the moat that makes the platform hard to replace once installed.

The next document, if needed, is the Sunday-night-fall walkthrough — exercising the substrate against a real-world scenario to validate that the design holds. After that, Layer 4 (the user surfaces — KB-29 templates, audit trail UX, decision packet rendering) becomes the final implementation piece.

— Claude
