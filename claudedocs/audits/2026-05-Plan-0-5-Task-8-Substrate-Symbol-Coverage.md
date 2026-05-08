# Plan 0.5 Task 8 — Substrate Symbol Coverage Audit

**Purpose**: Closes Plan 0.5 Task 8 honestly by documenting the real state of Substrate.* symbol
coverage between CQL helpers and the Java runtime, retagging misleading `TODO(wave-1-runtime)`
markers, and confirming no validator regression.

**Date**: 2026-05-08

**Author note — plan-vs-reality reframing**: The original Task 8 example in the plan assumed helper
bodies contained hollow stubs (`define X: ... false`) waiting to be wired to `Substrate.*` calls.
The actual codebase has fully-filled bodies that already call `Substrate.*`. The 17 `TODO` markers
that existed were file-header comments, not body markers. This audit corrects the terminology and
surfaces the real remaining gap: the Java runtime is missing ~38 of the ~43 symbols the helpers
reference.

---

## Section A: What Was Originally Proposed

The plan's Task 8 example read (paraphrased):

> "The 8 production helper files have hollow bodies (`define X: ... false`) with
> `TODO(wave-1-runtime)` body markers. Task 8 wires these to `Substrate.*` calls and removes the
> markers."

This shaped the expectation that helper bodies were scaffolding placeholders. That expectation was
incorrect — see Section B.

---

## Section B: Reality of the Codebase

All 8 production helper files already contain real `Substrate.*` calls in their function bodies.
For example, `MonitoringHelpers.cql`:

```cql
define function "ExpectedObservationCount"(planRef String, asOf DateTime):
  Substrate.MonitoringPlanExpectedCount(planRef, asOf)
```

The `TODO(wave-1-runtime)` markers appeared only in **file-header block comments** (or top-of-file
`//` comments in test files), not inside any function body. They were notes about when the
runtime integration would allow live execution — not body placeholders.

Each of the 8 production files had exactly one such header marker; each of the 7 matching `_test.cql`
files also had one (15 markers total, plus 2 multi-line variants in `MedicationHelpers.cql` and
`MedicationHelpers_test.cql` = 17 logical marker instances across all files). All have been
retagged to `TODO(plan-0.5-followup)` — see Section E.

---

## Section C: Symbol Coverage Matrix

43 distinct `Substrate.*` symbols were extracted from the 8 production helper files (excluding
`_test.cql` files per task scope). 6 symbols are registered in
`SubstrateExternalFunctions.java` (counting `recentObservations` which maps to `LatestObservationAt`/
`LatestObservationValue` semantically; see note below table).

Extraction command used:
```bash
grep -h "Substrate\." backend/shared-infrastructure/knowledge-base-services/shared/cql-libraries/helpers/*.cql \
  | grep -v "^[ ]*\*\|^//\|_test" \
  | sed 's/.*Substrate\.\([A-Za-z]*\).*/\1/' | sort -u
```

Java registration source: `kb-cql-runtime/src/main/java/au/vaidshala/cqlruntime/external/SubstrateExternalFunctions.java`
— public methods: `runningBaseline`, `baselineConfidence`, `activeConcerns`, `careIntensity`,
`medicineUse`, `recentObservations`.

| Substrate symbol | Used by helpers | Registered in `SubstrateExternalFunctions.java` |
|---|---|---|
| `ActiveConcernOpenedWithin` | ClinicalStateHelpers | ❌ |
| `AtcStartsWith` | MedicationHelpers | ❌ |
| `AuthRoleCanPrescribe` | AuthorisationHelpers | ❌ |
| `AuthorisationActiveAt` | AuthorisationHelpers | ❌ |
| `AvailablePrescribers` | AuthorisationHelpers | ❌ |
| `BaselineFor` | ClinicalStateHelpers | ❌ (mapped via `runningBaseline` variant) |
| `BaselineVelocityFlag` | ClinicalStateHelpers | ❌ |
| `CapacityAssessment` | ClinicalStateHelpers | ❌ |
| `CareIntensity` | ClinicalStateHelpers | ✅ (`careIntensity`) |
| `DischargeReconciliationCompleted` | QualityGapHelpers | ❌ |
| `EventCountOfTypeWithin` | QualityGapHelpers | ❌ |
| `EvidenceTraceConsequences` | EvidenceTraceHelpers | ❌ |
| `EvidenceTraceLineage` | EvidenceTraceHelpers | ❌ |
| `EvidenceTraceRecommendationOrigin` | EvidenceTraceHelpers | ❌ |
| `EvidenceTraceRuleFires` | EvidenceTraceHelpers | ❌ |
| `EvidenceTraceWindowSummary` | EvidenceTraceHelpers | ❌ |
| `FacilityHasReverseAgent` | MedicationHelpers | ❌ |
| `FlaggedBaselineDelta` | ClinicalStateHelpers | ❌ |
| `HasEventOfTypeWithin` | QualityGapHelpers | ❌ |
| `LatestEventIdOfType` | ClinicalStateHelpers | ❌ |
| `LatestEventOfTypeOlderThan` | ClinicalStateHelpers | ❌ |
| `LatestObservationAt` | ClinicalStateHelpers | ❌ (partial — `recentObservations` covers list, not single) |
| `LatestObservationValue` | ClinicalStateHelpers | ❌ (same as above) |
| `ListActiveConcerns` | ClinicalStateHelpers | ❌ (`activeConcerns` covers, but different CQL name) |
| `ListConsents` | ConsentStateHelpers | ❌ |
| `ListMedicineUsesByResident` | MedicationHelpers | ❌ (`medicineUse` is close but different CQL name) |
| `ListMonitoringPlans` | MonitoringHelpers | ❌ |
| `MedicineUseIntentCategory` | MedicationHelpers | ❌ |
| `MedicineUseIsActive` | MedicationHelpers | ❌ |
| `MedicineUseMeddMg` | MedicationHelpers | ❌ |
| `MedicineUseRxcui` | MedicationHelpers | ❌ |
| `MedicineUseStart` | MedicationHelpers | ❌ |
| `MedicineUseTargetType` | MedicationHelpers | ❌ |
| `MonitoringLastThresholdCrossing` | MonitoringHelpers | ❌ |
| `MonitoringObservationOverdueHours` | MonitoringHelpers | ❌ |
| `MonitoringPlanActiveAt` | MonitoringHelpers | ❌ |
| `MonitoringPlanExpectedCount` | MonitoringHelpers | ❌ |
| `RecommendationActionedWithin` | SuppressionHelpers | ❌ |
| `RecommendationDeferredWithin` | SuppressionHelpers | ❌ |
| `RecommendationsActionedByPrefix` | SuppressionHelpers | ❌ |
| `ScopeRulePermits` | AuthorisationHelpers | ❌ |
| `SdmFor` | ClinicalStateHelpers | ❌ |
| `TrajectoryDirection` | ClinicalStateHelpers | ❌ |

**Summary**: 43 distinct symbols used in production helpers. 1 registered by exact CQL name
(`CareIntensity` → `careIntensity`). 5 additional Java methods exist (`runningBaseline`,
`baselineConfidence`, `activeConcerns`, `medicineUse`, `recentObservations`) that correspond
loosely to CQL-side names but are not registered under the exact symbol names the helpers call
(e.g., helpers call `Substrate.BaselineFor`, Java registers `runningBaseline`). Name-mapping
resolution is a Task 5 concern. For conservative coverage counting: **1 confirmed match, 42
unregistered**.

---

## Section D: Why This Matters

The CQL toolchain validator accepts the helpers as structurally valid because external function
signatures are declared in the `Vaidshala.Substrate` CQL library — the static type-checker sees
the declaration and passes. At runtime, once `cqf-fhir-cr` is wired via `$evaluate-rule` (Plan 0.5
Task 5 followup), the HAPI engine will call the Java `ExternalFunctionProvider` for each
`Substrate.*` invocation. Any symbol not registered will throw `UnknownExternalFunction` at
evaluation time, causing the rule to error rather than produce a recommendation. With 42 of 43
symbols unregistered, effectively every helper-dependent rule would fail at runtime today. This
does not affect static validation (which is what Plan 0.5 covers) but it is the critical blocker
for live rule evaluation.

---

## Section E: Closing Plan 0.5 Task 8

### Action 1: Audit produced
This document.

### Action 2: Validator regression check

After retagging all 15 `TODO(wave-1-runtime)` markers (8 production files, 7 test files), the
CQL toolchain validator was re-run:

```
cd backend/shared-infrastructure/knowledge-base-services/shared/cql-toolchain
python3 -m pytest -v 2>&1 | tail -5
```

Output (tail):
```
tests/test_wave_extension_2026_05_batch.py::test_wave_extension_all_emit_valid_cds_hooks PASSED [99%]
tests/test_wave_extension_2026_05_batch.py::test_wave_extension_real_published_citations_for_published_tier2 PASSED [100%]

============================= 244 passed in 4.14s ==============================
```

**Result: 244 passed, 0 failures. No regression.**

### Action 3: Marker retagging

All `TODO(wave-1-runtime)` markers in the helpers directory have been retagged to
`TODO(plan-0.5-followup)` with the standard two-line explanation referencing Task 5 and this
audit. Files modified:

Production (block-comment `*` style):
- `AuthorisationHelpers.cql`
- `ClinicalStateHelpers.cql`
- `ConsentStateHelpers.cql`
- `EvidenceTraceHelpers.cql`
- `MedicationHelpers.cql`
- `MonitoringHelpers.cql`
- `QualityGapHelpers.cql`
- `SuppressionHelpers.cql`

Test (inline `//` style):
- `AuthorisationHelpers_test.cql`
- `ClinicalStateHelpers_test.cql`
- `ConsentStateHelpers_test.cql`
- `EvidenceTraceHelpers_test.cql`
- `MedicationHelpers_test.cql`
- `MonitoringHelpers_test.cql`
- `SuppressionHelpers_test.cql`

(`AgedCareHelpers.cql` had no `TODO(wave-1-runtime)` markers — confirmed by grep.)

---

## Section F: Carried Forward to Next Plan

The 42 unregistered `Substrate.*` symbols represent the core Wave-1 runtime work. It splits into
two parallel tracks:

1. **Java side**: Implement ~42 more methods in `SubstrateExternalFunctions.java` (or split across
   additional `@ExternalFunctionProvider` classes), each mapping a CQL symbol to a kb-20 REST
   endpoint.

2. **Go side**: Implement the corresponding ~42 REST endpoints on kb-20
   (`/v2/runtime/<symbol-slug>`) that the Java substrate client will call.

This work is scoped to **Phase 1 of v3 (next plan after Plan 0.5)**. The symbol list in Section C
serves as the work item backlog for that plan. Priority order suggested: `ClinicalState*` symbols
first (highest rule-surface coverage), then `Medication*`, then `Monitoring*`, then `EvidenceTrace*`,
then `Authorisation*`/`Suppression*`/`Consent*`.

Until that work lands, all helper-backed rules pass static validation but will throw
`UnknownExternalFunction` at live evaluation time.
