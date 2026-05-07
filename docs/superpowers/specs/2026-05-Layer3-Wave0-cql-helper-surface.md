# Layer 3 Wave 0 — CQL helper surface specification

**Date:** 2026-05-06
**Plan:** [2026-05-04-layer3-rule-encoding-plan.md](../plans/2026-05-04-layer3-rule-encoding-plan.md) — Wave 0 Task 2
**Companion specs:**
* [2026-05-Layer3-Wave0-tier1-substrate-contract.md](2026-05-Layer3-Wave0-tier1-substrate-contract.md)
* [2026-05-Layer3-Wave0-trigger-surface-mapping.md](2026-05-Layer3-Wave0-trigger-surface-mapping.md)
**Status:** Draft pending sign-off.

---

## Purpose

Define the function signatures of every helper in the six CQL helper
files Layer 3 will ship in Wave 1. Each helper has:

* Function name + parameter signature with FHIR / Vaidshala types
* Return type
* Performance contract (sync vs async, latency target — default <50ms p95
  against Layer 2 substrate APIs)
* Example usage in a CQL `define`
* Backing substrate API per the [Layer 2 → Layer 3 handoff doc](../../handoff/layer-2-to-layer-3-handoff.md)

Default latency target is **<50ms p95**. Helpers that read aggregate
substrate facts (baselines) carry tighter <30ms targets per the
handoff SLOs. Helpers that fan out to the Authorisation evaluator
carry a <60ms target. Anything async is called out explicitly; the
default contract is sync.

---

## File 1 — `MedicationHelpers.cql`

| Helper | Signature | Returns | Latency | Substrate path |
|---|---|---|---|---|
| `IsActiveOnDate` | `(med MedicineUse, asOf DateTime)` | `Boolean` | <30ms | `medicine_uses` (handoff) |
| `HasActiveAtcPrefix` | `(residentRef String, atcPrefix String)` | `Boolean` | <50ms | `medicine_uses` filtered by ATC |
| `GetIntentClass` | `(med MedicineUse)` | `String` (one of `initiation`, `continuation`, `taper`, `prn`, `unknown`) | <30ms | `medicine_uses.intent.category` |
| `GetTargetType` | `(med MedicineUse)` | `String` | <30ms | `medicine_uses.target.type` |
| `IsPrescribingOmission` | `(criterionSet String)` | `Boolean` | <1ms (pure) | discriminator legend (kb-4 migration 007) |
| `IsHighRiskFromBeers` | `(med MedicineUse)` | `Boolean` | <50ms | join Beers ValueSet to active meds |
| `ActiveDoseFor` | `(residentRef String, rxcui String)` | `Quantity` | <50ms | `medicine_uses` |
| `DurationDaysActive` | `(med MedicineUse)` | `Integer` | <30ms | `medicine_uses.start_date` |
| `MorphineEquivalentMgPerDay` | `(residentRef String)` | `Decimal` | <50ms | aggregate over opioid `medicine_uses` |
| `AnticholinergicBurdenScore` | `(residentRef String)` | `Integer` | <50ms | KB-4 ACB rules joined to active meds |

**Example:**

```cql
define "Long-Term PPI Without Indication":
  exists(MedicineUse) M
    where MedicationHelpers."HasActiveAtcPrefix"(M.subject.reference, 'A02BC')
      and MedicationHelpers."DurationDaysActive"(M) > 56
      and not exists(Condition) C where C.code in "PPI Indication ValueSet"
```

---

## File 2 — `ClinicalStateHelpers.cql`

| Helper | Signature | Returns | Latency | Substrate path |
|---|---|---|---|---|
| `BaselineFor` | `(residentRef String, observationKind String)` | `Decimal` | <30ms | `baseline_state` (Layer 2 plan §2.1) |
| `DeltaFromBaseline` | `(residentRef String, observationKind String, lookbackDays Integer)` | `Decimal` | <60ms | `flagged_baseline_delta` field |
| `IsTrending` | `(residentRef String, observationKind String, direction String)` | `Boolean` | <60ms | trajectory detector (Layer 2 plan §2.1) |
| `VelocityFlag` | `(residentRef String, observationKind String)` | `Boolean` | <60ms | velocity-flag field on baseline |
| `ActiveConcernCount` | `(residentRef String)` | `Integer` | <60ms | `active_concerns?open=true` |
| `HasActiveConcernType` | `(residentRef String, concernType String)` | `Boolean` | <60ms | `active_concerns?type=...` |
| `CurrentCareIntensity` | `(residentRef String)` | `String` (one of `active_treatment`, `comfort_focused`, `palliative`) | <30ms | `care_intensity` |
| `CapacityAssessmentFor` | `(residentRef String, domain String)` | `String` (one of `intact`, `impaired`, `lacks_capacity`) | <50ms | `capacity_assessment` |
| `LatestObservationValue` | `(residentRef String, kind String)` | `Decimal` | <100ms | `observations/{kind}` |
| `LatestObservationAt` | `(residentRef String, kind String)` | `DateTime` | <100ms | `observations/{kind}` |
| `IsPalliative` | `(residentRef String)` | `Boolean` | <30ms | `care_intensity = 'palliative'` |

**Example:**

```cql
define "Hyperkalemia Trajectory Risk":
  ClinicalStateHelpers."DeltaFromBaseline"(Patient.id, 'potassium', 7) > 0.8
    and exists(MedicineUse) M
      where MedicationHelpers."HasActiveAtcPrefix"(M.subject.reference, 'C09')
```

---

## File 3 — `ConsentStateHelpers.cql`

| Helper | Signature | Returns | Latency | Substrate path |
|---|---|---|---|---|
| `HasActiveConsentForClass` | `(residentRef String, consentClass String)` | `Boolean` | <50ms | Consent state machine API |
| `ConsentExpiringWithin` | `(residentRef String, consentClass String, days Integer)` | `Boolean` | <50ms | Consent state machine API |
| `NeedsConsentRefresh` | `(residentRef String, consentClass String)` | `Boolean` | <50ms | Consent state machine API |
| `ConsentExpiryDate` | `(residentRef String, consentClass String)` | `DateTime` | <50ms | Consent state machine API |
| `SdmReferenceFor` | `(residentRef String)` | `Reference` | <30ms | Resident → SDM relationship |

**Example:**

```cql
define "Antipsychotic Without Active Deprescribe Consent":
  exists(MedicineUse) M
    where MedicationHelpers."HasActiveAtcPrefix"(M.subject.reference, 'N05A')
      and MedicationHelpers."DurationDaysActive"(M) > 90
      and not ConsentStateHelpers."HasActiveConsentForClass"(
            M.subject.reference, 'Antipsychotic_deprescribe_review')
```

---

## File 4 — `AuthorisationHelpers.cql`

All helpers in this file fan out to the Authorisation evaluator (port
8138, planned Wave 3). Latency budget is **<60ms p95** (a touch
higher than substrate reads because the evaluator may consult
ScopeRules + credential cache).

| Helper | Signature | Returns | Latency | Substrate path |
|---|---|---|---|---|
| `RoleCanPrescribeFor` | `(roleRef String, drugClass String)` | `Boolean` | <60ms | Auth evaluator — role × class lookup |
| `AuthorisationIsActive` | `(authorisationRef String, asOf DateTime)` | `Boolean` | <60ms | Auth evaluator — Authorisation seam |
| `ScopeRulePermits` | `(action String, jurisdiction String, asOf DateTime)` | `Boolean` | <60ms | ScopeRule engine (port 8139) |
| `AvailablePrescriberForClass` | `(facilityRef String, drugClass String)` | `List<Reference>` | <60ms | Auth evaluator — credential cache |
| `HasAvailablePrescriberForClass` | `(facilityRef String, drugClass String)` | `Boolean` | <60ms | sugar over `AvailablePrescriberForClass` |
| `FallbackRouting` | `(drugClass String, urgency String)` | `String` (one of `telehealth`, `ED_transfer`, `after_hours_GP`) | <30ms | static routing table |
| `FacilityHasReverseAgentFor` | `(facilityRef String, drugClass String)` | `Boolean` | <50ms | Auth evaluator inventory query |

**Example:**

```cql
define "Insulin Hypoglycemia — Action Routing":
  if AuthorisationHelpers."HasAvailablePrescriberForClass"(
       Patient.facility, 'Insulin_dose_adjustment')
  then 'Reduce_dose_immediate_prescriber_attention'
  else AuthorisationHelpers."FallbackRouting"('Insulin_dose_adjustment', 'urgent')
```

---

## File 5 — `MonitoringHelpers.cql`

| Helper | Signature | Returns | Latency | Substrate path |
|---|---|---|---|---|
| `ExpectedObservationCount` | `(planRef String, asOf DateTime)` | `Integer` | <50ms | MonitoringPlan API |
| `ObservationOverdueBy` | `(planRef String, kind String, asOf DateTime)` | `Decimal` (hours) | <50ms | MonitoringPlan + observations join |
| `MonitoringPlanIsActive` | `(planRef String, asOf DateTime)` | `Boolean` | <30ms | MonitoringPlan API |
| `ActivePlansFor` | `(residentRef String)` | `List<Reference>` | <50ms | MonitoringPlan API |
| `HasOpenPlanOfType` | `(residentRef String, planType String)` | `Boolean` | <50ms | MonitoringPlan API |
| `LastThresholdCrossingAt` | `(planRef String)` | `DateTime` | <50ms | MonitoringPlan API |

**Example:**

```cql
define "Warfarin INR Overdue":
  exists(MedicineUse) M
    where MedicationHelpers."HasActiveAtcPrefix"(M.subject.reference, 'B01AA')
      and MonitoringHelpers."ObservationOverdueBy"(
            ActivePlanForWarfarin(M.subject.reference), 'INR', Now()) > 24
```

---

## File 6 — `EvidenceTraceHelpers.cql`

The EvidenceTrace API is read-only from CQL's perspective; writes
happen as a side-effect of the rule fire (handled by the engine, not
the rule body). These helpers expose the EvidenceTrace v0 graph
queries (Layer 2 plan Wave 1R.2).

| Helper | Signature | Returns | Latency (p95) | Substrate path |
|---|---|---|---|---|
| `LineageOf` | `(traceNodeRef String, depth Integer)` | `List<Reference>` | <100ms | EvidenceTrace bidirectional edges |
| `ConsequencesOf` | `(traceNodeRef String, depth Integer)` | `List<Reference>` | <100ms | EvidenceTrace forward traversal |
| `ReasoningWindowSummary` | `(residentRef String, sinceWindow Period)` | `Tuple<{ rule_fires: Integer, recommendations_drafted: Integer, observations_used: Integer }>` | <200ms (async) | EvidenceTrace materialised view; mapped to Wave 5.2 query API |
| `RuleFiredFor` | `(residentRef String, ruleId String, sinceWindow Period)` | `List<DateTime>` | <100ms | EvidenceTrace by rule_id |
| `RecommendationOriginRule` | `(recommendationRef String)` | `String` (rule_id) | <50ms | EvidenceTrace inputs lookup |

`ReasoningWindowSummary` is async: the helper returns a Tuple stream
that the engine resolves before evaluating downstream defines. It is
the only async helper in this surface.

**Example:**

```cql
define "Recent Deprescribing Activity Window":
  EvidenceTraceHelpers."ReasoningWindowSummary"(
    Patient.id,
    Interval[Now() - 90 days, Now()]).rule_fires > 0
```

---

## Cross-cutting performance rollup

Per Layer 3 v2 doc Part 1 the rule-evaluation budget is **<500ms
p95** end-to-end. With ~10 helper invocations per typical Tier 1 rule
× <50ms p95 each = <500ms in the pessimistic case. The
Authorisation-evaluator helpers (~3 invocations per gated rule) +
ClinicalState (~3) + Medication (~3) + Consent (~1) helpers comfortably
fit. EvidenceTrace queries (`LineageOf`, `ConsequencesOf`,
`ReasoningWindowSummary`) are **not** in the hot path of rule
firing — they are used by Wave 5 reporting / display surfaces.

---

## Substrate API coverage check

Every helper above maps onto a Layer 2 deliverable that has shipped
or is scheduled in the published Layer 2 plan. The following mapping
table is the single source of truth for the Wave 0 Task 1 acceptance
clause "Layer 2 team confirms each helper has a backing substrate
API":

| Helper file | Layer 2 deliverable group |
|---|---|
| MedicationHelpers | `medicine_uses` API + KB-4 explicit-criteria join |
| ClinicalStateHelpers | `observations` + `baseline_state` + `active_concerns` + `care_intensity` + `capacity_assessment` |
| ConsentStateHelpers | Consent state machine (Layer 2 doc §3) |
| AuthorisationHelpers | Authorisation evaluator (Layer 3 plan Wave 3) + ScopeRule engine (Wave 4) — **note: AuthorisationHelpers consumers must defensively handle the case where the evaluator is not yet deployed; until Wave 3 they return conservative defaults** |
| MonitoringHelpers | MonitoringPlan API (Layer 2 doc §2.4) |
| EvidenceTraceHelpers | EvidenceTrace v0 (Layer 2 plan Wave 1R.2) + Wave 5 hardening |

---

## Open questions

1. **Authorisation helper defaults pre-Wave-3.** Until the
   Authorisation evaluator is deployed, what should
   `RoleCanPrescribeFor` and friends return? Options:
   (a) `null` (force the rule to suppress);
   (b) `true` (permissive — rules fire under the assumption that
       authorisation will be checked at submission time);
   (c) `false` (deny — every rule fires the fallback path).
   **Recommendation:** option (a) — null forces rules to handle the
   "evaluator down" case explicitly, avoiding silent over- or
   under-permissioning. Validator (Wave 1) can lint for unhandled
   nulls.
2. **Helper namespace collisions.** CQL `define` resolution is by
   library name + define name; with 6 helper libraries we are at
   risk of accidental collisions if a Wave-2 rule define has the
   same name as a helper. Convention: helper names use TitleCase
   without verbs ("BaselineFor"); rule defines use sentence-case
   with the rule's clinical action ("Long-Term PPI Without
   Indication"). Validator enforces.
3. **Async helper handling.** Only `ReasoningWindowSummary` is
   async; the engine wrapper needs to know not to apply the helper
   in the hot rule-firing path. Stage 1 validator (Wave 1) will
   warn if an async helper appears inside a Tier 1 rule.

These tracked as Wave 1 backlog.
