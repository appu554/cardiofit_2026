# Layer 3 v2 Rule Encoding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Layer 3 v2 rule library — substrate-aware CQL defines + Authorisation evaluator subsystem + ScopeRules-as-data engine + CDS Hooks integration — that turns the Layer 2 substrate into clinical recommendations with sub-500ms p95 evaluation and < 5% override rate target.

**Architecture:** Three independent subsystems sharing the substrate: (1) **CQL rule library** (substrate-aware Tier 1-4 rules using FHIR Clinical Reasoning Module — PlanDefinition + Library + ActivityDefinition + RequestOrchestration with `$apply` operation, exposed via CDS Hooks v2.0 `order-select` and `order-sign`); (2) **Authorisation evaluator** (separate runtime service, sub-500ms p95, jurisdiction-aware ScopeRule engine with credential/agreement/consent caching); (3) **ScopeRules-as-data engine** (jurisdiction + temporal rules in YAML/JSON not CQL — Victorian PCW exclusion July 2026, designated RN prescriber agreements from mid-2026, Tasmanian pharmacist co-prescribing pilot 2026-2027).

**Tech Stack:** CQL (Clinical Quality Language) for rule expression, HAPI FHIR engine OR Smile CDR for `$apply`, CDS Hooks v2.0 client/service, PostgreSQL 16 for ScopeRule data + caches, Redis for Authorisation evaluator hot path, Go 1.22+ for the Authorisation evaluator + ScopeRule engine, Python for the CQL authoring + testing toolchain (mirrors existing kb-3-guidelines and Layer 1 patterns).

**Prerequisites (per Layer 3 v2 doc Part 0.5 + Part 5):**
- Layer 1 v2 ~75% complete; remaining gaps that block Layer 3: ICD-10 schema normalisation (resolved Tier 1.1), RxCUI resolution to 100% (resolved Tier 2), fact_type schema decision (`PRESCRIBING_OMISSION` discriminator vs tag — must be resolved before Wave 1), L6 governance audit trail verified for AU rules (must be resolved before Wave 2 production deploy)
- Layer 2 substrate per `/Volumes/Vaidshala/cardiofit/docs/superpowers/plans/2026-05-04-layer2-substrate-plan.md` — Wave 2 (Clinical state machine running baselines) is hard prerequisite for Wave 2+ rules of this plan; Wave 4 (hospital discharge reconciliation) is hard prerequisite for Wave 5 of this plan

**Predecessor:** Phase 1B-β.2 substrate entities (Observation with delta-on-write, MedicineUse with intent/target/stop, Authorisation seam via Person+Role) shipped on `main` at HEAD `ee6c5c1d`.

---

## File Structure

```
backend/shared-infrastructure/knowledge-base-services/
├── shared/
│   ├── cql-libraries/                          # NEW — root of CQL rule library
│   │   ├── helpers/
│   │   │   ├── MedicationHelpers.cql           # AMT/RxCUI lookups, dose math
│   │   │   ├── AgedCareHelpers.cql             # ACOP/ACQSC value sets, AN-ACC class
│   │   │   ├── ClinicalStateHelpers.cql        # baseline/delta/trajectory queries
│   │   │   ├── ConsentStateHelpers.cql         # consent class lookup, expiry
│   │   │   ├── AuthorisationHelpers.cql        # available_prescriber_for_class()
│   │   │   ├── MonitoringHelpers.cql           # active monitoring plan checks
│   │   │   └── EvidenceTraceHelpers.cql        # write-side hooks for trace context
│   │   ├── tier-1-immediate-safety/            # ~25 rules
│   │   │   ├── HyperkalemiaRiskTrajectory.cql
│   │   │   ├── PostFallReassessment.cql
│   │   │   ├── InsulinHypoglycemiaImminent.cql
│   │   │   ├── AntipsychoticConsentMissing.cql
│   │   │   └── ...
│   │   ├── tier-2-deprescribing/               # ~75 rules (ADG 2025, STOPP/START, Beers, Wang)
│   │   │   ├── PPILongTermNoIndication.cql
│   │   │   ├── AntipsychoticBPSDReview.cql
│   │   │   ├── BenzodiazepineTaper.cql
│   │   │   ├── StatinPalliativeDeprescribe.cql
│   │   │   └── ...
│   │   ├── tier-3-quality-gap/                 # ~50 rules
│   │   ├── tier-4-surveillance/                # ~50 rules
│   │   ├── value-sets/                         # FHIR ValueSet bundles (AMT/SNOMED-CT-AU/RxNorm)
│   │   ├── plan-definitions/                   # FHIR PlanDefinition + ActivityDefinition + Library
│   │   └── tests/                              # Synthea + scripted patient fixtures
│   │       ├── fixtures/
│   │       └── per-rule/
│   ├── cql-toolchain/                          # NEW — Python authoring toolchain
│   │   ├── rule_specification_validator.py     # YAML schema + substrate-ref linter
│   │   ├── two_gate_validator.py               # Stage 2: snapshot + substrate gate
│   │   ├── compatibility_checker.py            # Events A/B/C/D
│   │   ├── cds_hooks_emitter.py                # CDS Hooks v2.0 response builder
│   │   └── governance_promoter.py              # Stage 5 promotion + signing
│   └── v2_substrate/                           # EXISTING — read-side contract for rules
├── kb-3-guidelines/                            # EXISTING — extended for ScopeRule sourcing
├── kb-22-hpi-engine/                           # EXISTING — emits triggers into rule firing
├── kb-30-authorisation-evaluator/              # NEW — Go service, port 8138 (proposed)
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── api/                                # gRPC + REST surface for CQL helpers
│   │   ├── cache/                              # Redis-backed TTL strategy
│   │   ├── dsl/                                # AuthorisationRule parser + evaluator
│   │   ├── invalidation/                       # credential/agreement/consent triggers
│   │   ├── audit/                              # regulator-queryable audit query API
│   │   └── metrics/                            # p95 latency, cache hit rate
│   ├── migrations/
│   └── tests/
└── kb-31-scope-rules/                          # NEW — ScopeRule data dir + engine, port 8139
    ├── cmd/server/main.go
    ├── data/                                   # YAML/JSON ScopeRule files (versioned)
    │   ├── AU/national/                        # ACOP, NMBA, Aged Care Rules 2025
    │   ├── AU/VIC/                             # PCW exclusion 2026-07-01
    │   ├── AU/TAS/                             # Pharmacist co-prescribing pilot
    │   └── AU/<state>/                         # future expansion
    ├── internal/
    │   ├── parser/                             # ingest from kb-3 sources
    │   ├── store/                              # versioned PostgreSQL store
    │   └── evaluator/                          # applicable-rule lookup + grant/deny combine
    └── migrations/
```

---

## Pre-Wave: Layer 1 audit blocker closure (1 week)

**Goal:** Close the four Layer 1 v2 audit blockers that prevent Layer 3 rules from being authored, validated, or governance-signed at production grade.

### Pre-Wave Task 1 — fact_type schema decision for `PRESCRIBING_OMISSION`

- **Goal:** Decide whether START rules (40 rows) remain stored as `criterion_set='START_V3'` discriminator, or are re-tagged with explicit `fact_type='PRESCRIBING_OMISSION'`. Either choice is fine; ambiguity blocks rule authoring because the CQL `define` predicate must reference one or the other.
- **Files:** `/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-4-patient-safety/migrations/007_fact_type_resolution.sql` (new), audit decision memo at `claudedocs/audits/2026-05-PreWave-fact-type-decision.md`.
- **Acceptance:** A single SQL query distinguishes START rules from STOPP rules deterministically, and the `MedicationHelpers.cql` `IsPrescribingOmission()` helper returns the correct boolean for all 40 START rows + a sample of 10 STOPP rows.
- **Effort estimate:** 1 day (decision) + 1 day (migration + backfill).
- **Dependencies:** None.

### Pre-Wave Task 2 — L6 governance audit trail verified for AU rules

- **Goal:** Verify the KB-4 Ed25519 signing chain for all 392 AU rules is intact, dual-approval entries exist for ACB/Wang/AU APINCHs (the three new AU rule sets), and the audit trail is queryable through the existing KB-4 governance endpoints.
- **Files:** `kb-4-patient-safety/internal/governance/verify_au_chain.go`, `claudedocs/audits/2026-05-PreWave-l6-governance-verification.md`.
- **Acceptance:** Verification script exits 0 against all 8 criterion_sets; one signature failure surfaces as exit 1 with criterion_set + rule_id reported.
- **Effort estimate:** 2 days.
- **Dependencies:** None.

### Pre-Wave Task 3 — Final RxCUI gap closure

- **Goal:** Close the residual RxCUI mapping gaps surfaced in the 2026-04-30 cross-KB validation (82 mutations applied, but new ADG 2025 + Wang 2024 entries may add fresh gaps).
- **Files:** `kb-7-terminology/scripts/validate_kb_codes.py --rxnav --strict`.
- **Acceptance:** Validator exits 0; no `criterion_set` row references an unresolved RxCUI.
- **Effort estimate:** 1 day.
- **Dependencies:** Pre-Wave Task 1 (fact_type clarity required to scope which rows are validated).

### Pre-Wave Task 4 — ICD-10-AM placeholder strategy

- **Goal:** ICD-10-AM remains license-blocked. Tier 1/2 rules using ICD codes must use ICD-10 (international, free) with documented clinical equivalence, not ICD-10-AM. Document the mapping policy in CQL helpers so authors don't inadvertently reference ICD-10-AM.
- **Files:** `shared/cql-libraries/helpers/AgedCareHelpers.cql` (header comment + `IcdCodeIsClinicallyEquivalent()` helper).
- **Acceptance:** Helper compiles; rule_specification_validator.py rejects any rule referencing an ICD-10-AM code with a clear error.
- **Effort estimate:** 0.5 day.
- **Dependencies:** None.

**Pre-Wave exit criterion:** All four blockers closed; cross-KB validator exits 0; KB-4 L6 governance verifier exits 0; rule_specification_validator.py rejects fact_type/ICD-10-AM mistakes.

---

## Wave 0 — Substrate scoping (Weeks 1-2)

**Goal:** Identify exactly which Layer 2 substrate primitives Tier 1 rules need; document the substrate-rule contract; produce an authoring spec for substrate-aware CQL libraries. This is cross-team scoping work — outputs are docs, not code. Skipping it produces months of rework downstream (Layer 3 v2 doc Part 7 recommendation 1).

### Wave 0 Task 1 — Tier 1 substrate primitive inventory

- **Goal:** For each of the ~25 Tier 1 immediate-safety rules planned in Wave 2, enumerate the Clinical-state baselines, Consent classes, Authorisation classes, Monitoring plan types, and Event types each rule reads or writes.
- **Files:** `docs/superpowers/specs/2026-05-Layer3-Wave0-tier1-substrate-contract.md`.
- **Acceptance:** Table of 25 rules × 5 substrate machines with explicit fact references; reviewed and signed off by Layer 2 lead, clinical informatics, and engineering lead. Layer 2 plan Wave 2 backlog updated with any missing primitives.
- **Effort estimate:** 5 days.
- **Dependencies:** Layer 2 plan Wave 1 (substrate entities) entity model finalised. Pre-Wave complete.

### Wave 0 Task 2 — CQL helper surface specification

- **Goal:** Define the function signatures of every CQL helper in `MedicationHelpers.cql`, `ClinicalStateHelpers.cql`, `ConsentStateHelpers.cql`, `AuthorisationHelpers.cql`, `MonitoringHelpers.cql`, and `EvidenceTraceHelpers.cql`. Specify return types, parameter types, performance contracts (sync vs async, latency budget per call).
- **Files:** `docs/superpowers/specs/2026-05-Layer3-Wave0-cql-helper-surface.md`.
- **Acceptance:** Helper spec covers every fact type identified in Wave 0 Task 1; Layer 2 team confirms each helper has a backing substrate API.
- **Effort estimate:** 4 days.
- **Dependencies:** Wave 0 Task 1.

### Wave 0 Task 3 — rule_specification.yaml v2 schema

- **Goal:** Extend the v1.0 `rule_specification.yaml` schema with `state_machine_references`, `authorisation_gating`, `consent_gating`, `trigger_sources` per Layer 3 v2 doc Part 1.1.
- **Files:** `shared/cql-toolchain/schemas/rule_specification.v2.json` (JSON Schema), example specs at `shared/cql-libraries/examples/`.
- **Acceptance:** Three example rule_specifications validate against schema (PPI deprescribing, hyperkalemia trajectory, antipsychotic consent gating).
- **Effort estimate:** 3 days.
- **Dependencies:** Wave 0 Task 2.

### Wave 0 Task 4 — Trigger surface mapping

- **Goal:** Document every trigger event source from Layer 3 v2 doc Part 0.5.5 (medication change, condition change, observation update, baseline delta, active concern resolution, monitoring threshold crossed, consent expiry approaching, authorisation expiry approaching, care intensity transition, care transition) and identify which Layer 2 plan deliverable produces each.
- **Files:** `docs/superpowers/specs/2026-05-Layer3-Wave0-trigger-surface-mapping.md`.
- **Acceptance:** Each trigger has a named Layer 2 producer; gaps surfaced as Layer 2 plan backlog items.
- **Effort estimate:** 2 days.
- **Dependencies:** Wave 0 Task 1.

**Wave 0 exit criterion:** Three documents (substrate contract, helper surface, trigger mapping) signed off; rule_specification.v2.json validates the example specs; Layer 2 plan has any required additions opened as backlog.

---

## Wave 1 — Authoring infrastructure (Weeks 3-4)

**Goal:** Stand up the CQL library scaffolding, helpers, two-gate validation pipeline, and governance promoter — the toolchain that lets Wave 2+ author rules. No clinical rules in this Wave; only the infrastructure to author them.

### Wave 1 Task 1 — Helper library implementation

- **Goal:** Implement the six CQL helper files specified in Wave 0 Task 2 against the live Layer 2 substrate APIs.
- **Files:** `shared/cql-libraries/helpers/*.cql` (six files).
- **Acceptance:** Each helper has a unit test asserting correct values for at least 3 fixture residents from the Layer 2 plan Wave 1 fixtures. CQL compilation succeeds with HAPI FHIR engine. p95 latency per helper call < 50ms against fixture data.
- **Effort estimate:** 6 days.
- **Dependencies:** Wave 0 complete; Layer 2 plan Wave 1 substrate entities deployed in dev.

### Wave 1 Task 2 — rule_specification validator + two-gate pipeline

- **Goal:** Implement Stage 1 (clinical translation lint) and Stage 2 (two-gate validation: snapshot semantics gate + substrate semantics gate) per Layer 3 v2 doc Part 1.
- **Files:** `shared/cql-toolchain/rule_specification_validator.py`, `shared/cql-toolchain/two_gate_validator.py`.
- **Acceptance:** Validator catches the four classic authoring errors (missing trigger source, dangling fact reference, missing test case class, missing authorisation_gating for Schedule-8 actions). Test fixtures cover positive/negative cases.
- **Effort estimate:** 5 days.
- **Dependencies:** Wave 1 Task 1.

### Wave 1 Task 3 — CompatibilityChecker extensions (Events A/B/C/D)

- **Goal:** Extend the v1.0 CompatibilityChecker to handle Event C (substrate schema changes) and Event D (regulatory ScopeRule changes) per Layer 3 v2 doc Part 4.
- **Files:** `shared/cql-toolchain/compatibility_checker.py`.
- **Acceptance:** Synthetic substrate change (e.g., new Clinical-state baseline type) marks 3 example defines as STALE; synthetic ScopeRule deployment marks 1 authorisation-gated define as STALE.
- **Effort estimate:** 4 days.
- **Dependencies:** Wave 1 Task 2.

### Wave 1 Task 4 — CDS Hooks v2.0 emitter + PlanDefinition `$apply`

- **Goal:** Implement the FHIR Clinical Reasoning `$apply` invocation pattern (PlanDefinition + Library + ActivityDefinition + RequestOrchestration) and the CDS Hooks v2.0 response builder for `order-select` and `order-sign`.
- **Files:** `shared/cql-toolchain/cds_hooks_emitter.py`, sample PlanDefinition/Library/ActivityDefinition fixtures at `shared/cql-libraries/plan-definitions/example-ppi-deprescribe.json`.
- **Acceptance:** Round-trip test: rule fires → PlanDefinition `$apply` produces RequestOrchestration → CDS Hooks v2.0 response includes the `card` with link to FHIR resource. Response shape conforms to CDS Hooks v2.0 spec.
- **Effort estimate:** 5 days.
- **Dependencies:** Wave 1 Task 1.

### Wave 1 Task 5 — Governance promoter (Stage 5)

- **Goal:** Stage-5 promotion workflow with dual signing (clinical reviewer + medical director), Ed25519 signature, EvidenceTrace `rule_publication` node emission.
- **Files:** `shared/cql-toolchain/governance_promoter.py`.
- **Acceptance:** Promotes a sample rule end-to-end; KB-4 governance log shows dual signature; CompatibilityChecker subsequently sees the rule as ACTIVE.
- **Effort estimate:** 3 days.
- **Dependencies:** Wave 1 Task 3, Pre-Wave Task 2.

**Wave 1 exit criterion:** Three anchor defines (PPI deprescribing — substrate-aware version of v1.0 canonical example; Hyperkalemia Trajectory; Antipsychotic Consent Gathering) compile, validate, sign, and emit valid CDS Hooks v2.0 responses against fixture residents.

---

## Wave 2 — Tier 1 immediate-safety rules (Weeks 5-8)

**Goal:** Author ~25 Tier 1 immediate-safety rules over the substrate.

**Hard prerequisite:** Layer 2 plan Wave 2 (Clinical state machine running baselines) complete; without this, baseline-aware rules cannot be authored.

### Wave 2 Task 1 — Baseline-aware electrolyte/renal trajectory rules (~6 rules)

- **Goal:** Author HyperkalemiaRiskTrajectory, HyponatremiaRiskTrajectory, eGFRDeclineVelocity, AKIPostNSAID, AKIPostContrast, RhabdomyolysisStatinPostFall.
- **Files:** `shared/cql-libraries/tier-1-immediate-safety/*.cql` (six rules) + per-rule rule_specification + per-rule test fixtures.
- **Acceptance:** Each rule fires correctly against ≥10 historical RACF cases (anonymised) with 0 false positives in the matched-suppression cases. p95 evaluation latency < 200ms.
- **Effort estimate:** 8 days (~1.3 days/rule × 6).
- **Dependencies:** Layer 2 plan Wave 2 (running baselines on potassium, sodium, eGFR).

### Wave 2 Task 2 — Falls + post-fall rules (~5 rules)

- **Goal:** Author PostFallReassessment, PostFallMedicationReview, PostFallHeadInjuryWatch72h, FallsRiskCompoundingMedications, FallsRiskOrthostaticHypotension.
- **Files:** `shared/cql-libraries/tier-1-immediate-safety/PostFall*.cql` (five rules).
- **Acceptance:** Sunday-night-fall walkthrough scenario (Layer 3 v2 doc Part 4.5.5) produces correct firing sequence. Active-concern (`post_fall_72h`) integration verified end-to-end.
- **Effort estimate:** 7 days.
- **Dependencies:** Layer 2 plan Wave 3 (Monitoring state machine), Layer 2 plan Wave 5 (Active concerns in Clinical state).

### Wave 2 Task 3 — Restrictive practice + consent-gated rules (~5 rules)

- **Goal:** Author AntipsychoticBPSDConsentMissing, BenzodiazepineBPSDConsentMissing, RestrictivePracticeAuthorisationExpiry, AntipsychoticBPSDDocumentationGap, ChemicalRestraintCriteria.
- **Files:** `shared/cql-libraries/tier-1-immediate-safety/Antipsychotic*.cql`, `Benzodiazepine*.cql`, `RestrictivePractice*.cql`.
- **Acceptance:** Each rule fires consent-gathering Recommendation (not medication-change Recommendation) when consent is missing; ConsentState helper integration verified.
- **Effort estimate:** 7 days.
- **Dependencies:** Layer 2 plan Wave 6 (Consent state machine).

### Wave 2 Task 4 — Insulin/hypoglycemia + Schedule 8 routing rules (~5 rules)

- **Goal:** Author InsulinHypoglycemiaImminent, InsulinDoseAdjustmentRequiringS8, OpioidEscalationS8AuthorisationCheck, OpioidNaiveStartS8AuthorisationCheck, OxycodoneFentanylTitration.
- **Files:** `shared/cql-libraries/tier-1-immediate-safety/Insulin*.cql`, `Opioid*.cql`.
- **Acceptance:** Authorisation-routing test (Layer 3 v2 doc Part 1.2) passes — when no S8 prescriber available, rule fires with `ED_transfer_protocol` fallback. Latency check: rule + Authorisation evaluator round-trip < 500ms p95.
- **Effort estimate:** 7 days.
- **Dependencies:** Layer 2 plan Wave 1 (Authorisation seam read API). Authorisation evaluator NOT required at this stage — rules use the Layer 2 Authorisation seam directly with stubbed ScopeRules; full Authorisation evaluator arrives in Wave 4.

### Wave 2 Task 5 — Care-intensity-aware rules (~4 rules)

- **Goal:** Author StatinPalliativePrimaryPrevention, BisphosphonatePalliative, AntihypertensiveOverTreatedPalliative, AntiplateletPalliative.
- **Files:** `shared/cql-libraries/tier-1-immediate-safety/*Palliative*.cql`.
- **Acceptance:** Care-intensity-test passes — same patient scenario fires differently across active_treatment / comfort_focused / palliative tags.
- **Effort estimate:** 5 days.
- **Dependencies:** Layer 2 plan Wave 2 (Clinical state care_intensity tag).

**Wave 2 exit criterion:** ~25 Tier 1 rules ACTIVE; suppression-class 5 (substrate-state) and 6 (authorisation-context) verified for at least 5 rules each; volume-budget check (≤5 actionable alerts/resident/day) confirmed in pilot data.

---

## Wave 3 — Tier 2 deprescribing rules (Weeks 9-16)

**Goal:** Author ~75 Tier 2 deprescribing rules from STOPP v3, START v3, Beers 2023, Australian PIMs (Wang 2024), and the Australian Deprescribing Guideline 2025.

**Hard prerequisite:** Layer 2 plan Wave 2 complete (running baselines); Layer 2 plan Wave 4 partially required (hospital discharge reconciliation feeds intent fields that Tier 2 deprescribing rules consume — but rules can suppress when intent unknown, so Wave 4 is not a hard gate).

### Wave 3 Task 1 — ADG 2025 mapping spreadsheet + value sets

- **Goal:** Map all 185 ADG 2025 recommendations to candidate CQL defines; produce an FHIR ValueSet bundle from AMT codes for each medication class targeted.
- **Files:** `claudedocs/clinical/2026-05-Layer3-ADG2025-mapping.csv`, `shared/cql-libraries/value-sets/adg2025-*.json`.
- **Acceptance:** All 185 recommendations classified as (a) authored Wave 3 rule, (b) deferred to Wave 4 quality-gap, or (c) explicitly out-of-scope with rationale.
- **Effort estimate:** 5 days.
- **Dependencies:** ADG 2025 Pipeline-2 extraction complete (Layer 1 audit Wave 5 free-alternative path).

### Wave 3 Task 2 — STOPP/START/Beers/Wang authoring (~50 rules)

- **Goal:** Author CQL defines for the 50 highest-yield rules from the union of STOPP v3 (80 rows), START v3 (40 rows), Beers 2023 (57 rows), Wang 2024 (19 rows). Yield prioritised per the Ramsey 2025 implementation-rate baselines (PPI 43%, cessation 51%, dose-reduction 49%).
- **Files:** `shared/cql-libraries/tier-2-deprescribing/*.cql` (~50 rules).
- **Acceptance:** Each rule passes: snapshot+substrate gate; baseline-aware fire test; consent-gating test (where applicable); care-intensity test; EvidenceTrace test. Class-specific pilot result targets: colecalciferol >40%, calcium >40%, PPI >50%, cessation overall >55%, dose-reduction >55%.
- **Effort estimate:** 30 days (1.7 weeks per 5 rules ≈ 0.3 days/rule including spec/CQL/tests/governance).
- **Dependencies:** Wave 3 Task 1.

### Wave 3 Task 3 — ADG 2025-specific deprescribing rules (~20 rules)

- **Goal:** Author the ADG 2025 recommendations not covered by STOPP/START/Beers/Wang — the Australian-specific deprescribing pathways (e.g., antipsychotic in BPSD per ADG 2025 protocols, PPI step-down per ADG 2025).
- **Files:** `shared/cql-libraries/tier-2-deprescribing/ADG2025-*.cql` (~20 rules).
- **Acceptance:** Each rule cites ADG 2025 recommendation ID in EvidenceTrace `reasoning_summary`. CompatibilityChecker pass for `tier-2-deprescribing/au/`.
- **Effort estimate:** 6 days.
- **Dependencies:** Wave 3 Task 1.

### Wave 3 Task 4 — Suppression-by-recent-action library

- **Goal:** Implement the recently-actioned suppression class (Class 2) with the `recent_within_days()` patterns from Wang and STOPP, integrated with the Recommendation state machine's recently-closed query.
- **Files:** `shared/cql-libraries/helpers/SuppressionHelpers.cql`, integration tests.
- **Acceptance:** A rule that fires once and is closed (decided / refused / superseded) does not re-fire within the suppression window for any of the 50 Wave 3 rules.
- **Effort estimate:** 3 days.
- **Dependencies:** Wave 3 Task 2 partial (need ≥10 rules to test).

### Wave 3 Task 5 — Tier 2 governance + production deploy

- **Goal:** Promote Wave 3 rules through governance; deploy to engines; CompatibilityChecker pass.
- **Files:** governance audit log entries; deployment manifests.
- **Acceptance:** All ~75 rules ACTIVE; KB-4 L6 governance verifier exits 0; production engines reload without latency regression (p95 evaluation latency stays < 500ms across Tier 1+2 combined).
- **Effort estimate:** 4 days.
- **Dependencies:** Wave 3 Tasks 1-4, Pre-Wave Task 2.

**Wave 3 exit criterion:** ~75 Tier 2 rules ACTIVE; total rule library 100 rules; cumulative override-rate analytics (Wave 6 prep) collecting data; pilot facility deprescribing-implementation rates show measurable lift toward Wave 3 targets.

---

## Wave 4 — Tier 3 quality gap rules + Authorisation evaluator build (Weeks 17-22)

**Goal:** Two parallel streams. Stream A authors ~50 Tier 3 quality-gap rules. Stream B builds the Authorisation evaluator subsystem (Layer 3 v2 doc Part 4.5.6 — 6-8 weeks of focused engineering).

### Stream A — Tier 3 quality-gap rules (~50 rules)

#### Wave 4A Task 1 — Quality indicator + Standard 5 evidence rules (~25 rules)

- **Goal:** Author rules that produce PHARMA-Care indicators and Standard 5 evidence as workflow exhaust (Vaidshala v2 Revision Mapping Part 9 success metrics). Examples: AntipsychoticPrevalenceWithoutBPSD, PolypharmacyTrigger10plus, AnticholinergicBurdenACBabove3, FallsCompoundingMedicationCheck, BPSDFirstLineNonPharm.
- **Files:** `shared/cql-libraries/tier-3-quality-gap/*.cql` (~25 rules).
- **Acceptance:** PHARMA-Care 5-domain indicator computation runs end-to-end against pilot facility data; output schema matches PHARMA-Care framework v1.
- **Effort estimate:** 12 days.
- **Dependencies:** Layer 2 plan Wave 7 (PHARMA-Care indicator computation).

#### Wave 4A Task 2 — Care-transition quality-gap rules (~15 rules)

- **Goal:** Hospital discharge reconciliation gaps, RACF admission medication review gaps, post-RMMR follow-up gaps. Examples: HospitalDischargeMedicationsNotReconciled72h, RMMRRecommendationOverdue6mo, NewRACFAdmissionMedicationReviewMissing.
- **Files:** `shared/cql-libraries/tier-3-quality-gap/Transition*.cql` (~15 rules).
- **Acceptance:** Each rule integrates with Event types `hospital_discharge`, `admission_to_facility`, `care_planning_meeting`. Pilot data shows the rules surface real overdue reviews.
- **Effort estimate:** 8 days.
- **Dependencies:** Layer 2 plan Wave 4 (hospital discharge reconciliation pipeline).

#### Wave 4A Task 3 — AN-ACC defensibility rules (~10 rules)

- **Goal:** Rules that surface evidence supporting AN-ACC class reassessment and revenue assurance. Examples: ACFIClassRecentClinicalChange, AN-ACCFunctionalDeclineEvidence, AN-ACCBehaviouralEvidence.
- **Files:** `shared/cql-libraries/tier-3-quality-gap/AN-ACC*.cql` (~10 rules).
- **Acceptance:** Pilot RACH operator confirms the surfaced evidence packets meet AIHW assessor expectations.
- **Effort estimate:** 6 days.
- **Dependencies:** Wave 4A Task 1.

### Stream B — Authorisation evaluator subsystem build (6-8 weeks)

#### Wave 4B Task 1 — AuthorisationRule DSL + parser (Week 17)

- **Goal:** Implement the AuthorisationRule format (Layer 3 v2 doc Part 4.5.2) as a parser + schema validator. The DSL is YAML at rest, parsed to a typed AST in Go. Schema below is contractual.
- **Files:** `kb-30-authorisation-evaluator/internal/dsl/schema.go`, `internal/dsl/parser.go`, `internal/dsl/parser_test.go`.
- **Contractual data structure:**
  ```yaml
  authorisation_rule:
    rule_id: string                   # e.g. "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01"
    jurisdiction: string              # ISO-style: "AU" | "AU/VIC" | "AU/TAS" | ...
    effective_period:
      start_date: ISO8601
      end_date: ISO8601 | null
      grace_period_days: integer | null
    applies_to:
      role: string                    # role_type from Layer 2 Person+Role
      action_class: enum              # administer | prescribe | observe | recommend | consent_witness
      medication_schedule: [string]   # ["S4","S8","S9"] etc
      medication_class_includes: [string]
      resident_self_administering: boolean
    evaluation:
      decision: enum                  # granted | granted_with_conditions | denied
      reason: string
      conditions: [{condition, check}]
      fallback_required: boolean
      fallback_eligible_roles: [string]
      if_any_condition_fails:
        decision: enum
        reason: string
    audit:
      legislative_reference: string
      source_id: string
      source_version: string
      recordkeeping_required: boolean
      recordkeeping_period_years: integer
  ```
- **Acceptance:** Parser round-trips all three example rules from Layer 3 v2 doc Part 4.5.2 (Victorian PCW, designated RN, ACOP credential). Schema validator rejects malformed rules with line/column errors.
- **Effort estimate:** 5 days.
- **Dependencies:** None (Stream B is independent of Stream A).

#### Wave 4B Task 2 — Rule store + versioning (Week 18)

- **Goal:** PostgreSQL-backed AuthorisationRule store with per-jurisdiction versioning, effective_period query, lineage tracking.
- **Files:** `kb-30-authorisation-evaluator/migrations/`, `internal/store/`.
- **Acceptance:** Versioned write/read for ≥3 rules across 2 jurisdictions; lineage chain queryable.
- **Effort estimate:** 5 days.
- **Dependencies:** Wave 4B Task 1.

#### Wave 4B Task 3 — Runtime evaluator (Week 19)

- **Goal:** Given (jurisdiction, role, action_class, medication_class, resident_id, action_date), look up applicable rules, filter by effective_period, evaluate each, combine results per Layer 3 v2 doc Part 5.5.4. Expose via gRPC (Go-native callers) and REST (CQL helpers via Python toolchain).
- **Files:** `kb-30-authorisation-evaluator/internal/evaluator/`, `cmd/server/main.go`.
- **Acceptance:** End-to-end evaluation of the three example rules against fixture actions. Cold-cache p95 < 500ms.
- **Effort estimate:** 5 days.
- **Latency budget:** Cold p95 < 500ms; warm p95 < 200ms (post-cache).
- **Dependencies:** Wave 4B Task 2.

#### Wave 4B Task 4 — Cache layer with TTL strategy (Week 20)

- **Goal:** Redis-backed cache per Layer 3 v2 doc Part 4.5.3 — key = `(jurisdiction, role, action_class, medication_class, resident_id, fire_date)`, per-rule TTL (24h static, 1h credential, 15min agreement, 5min consent).
- **Files:** `kb-30-authorisation-evaluator/internal/cache/`.
- **Acceptance:** Warm p95 < 200ms; cache hit rate > 95% on simulated steady-state load.
- **Effort estimate:** 5 days.
- **Latency budget:** Warm p95 < 200ms.
- **Dependencies:** Wave 4B Task 3.

#### Wave 4B Task 5 — Invalidation triggers (Week 21)

- **Goal:** Wire credential / PrescribingAgreement / Consent / substrate-Resident / ScopeRule update events to cache invalidation per Part 4.5.3.
- **Files:** `kb-30-authorisation-evaluator/internal/invalidation/`, Kafka consumer for substrate change events.
- **Acceptance:** Synthetic credential expiry invalidates expected cache entries within 1s; no stale grants remain queryable.
- **Effort estimate:** 4 days.
- **Dependencies:** Wave 4B Task 4, Layer 2 plan Wave 1 (substrate change events on Kafka).

#### Wave 4B Task 6 — Audit query API (Week 22, half)

- **Goal:** Implement the four sample regulator queries from Layer 3 v2 doc Part 4.5.4 over the EvidenceTrace graph; expose via REST with FHIR Bundle / CSV / JSON output.
- **Files:** `kb-30-authorisation-evaluator/internal/audit/`, sample query fixtures.
- **Acceptance:** Each of the four sample queries returns correct structured output in <5s against pilot-volume data; output is regulator-defensible (full chain of authority, timestamps, EvidenceTrace IDs).
- **Effort estimate:** 4 days.
- **Dependencies:** Wave 4B Task 3, Layer 2 plan Wave 1 (EvidenceTrace graph).

#### Wave 4B Task 7 — Sunday-night-fall integration walkthrough (Week 22, half)

- **Goal:** End-to-end integration test of the Sunday-night-fall scenario (Layer 3 v2 doc Part 4.5.5) — 7 Authorisation evaluations across the workflow (PCW Event log, RN observation, ACOP profile view, GP recommendation approval, RN monitoring observations × 3).
- **Files:** `kb-30-authorisation-evaluator/tests/integration/sunday_night_fall_test.go`.
- **Acceptance:** All 7 evaluations succeed within their stated latency budgets (PCW Event <50ms, RN observation <50ms, ACOP profile <100ms, GP approval <200ms — the most complex, RN monitoring <50ms each). Total scenario p95 evaluation latency < 500ms per call. EvidenceTrace contains the full chain.
- **Effort estimate:** 3 days.
- **Latency budget:** Per the Layer 3 v2 spec table.
- **Dependencies:** Wave 4B Tasks 1-6.

**Wave 4 exit criterion:** Stream A — ~50 Tier 3 rules ACTIVE; PHARMA-Care indicators producible. Stream B — Authorisation evaluator operational at sub-500ms p95; Sunday-night-fall walkthrough passes; audit query API regulator-ready. Total rule library 150 rules.

---

## Wave 5 — Tier 4 surveillance rules + ScopeRules-as-data deployment (Weeks 23-25)

**Goal:** Author ~50 Tier 4 surveillance rules and deploy the ScopeRules-as-data engine (Layer 3 v2 doc Part 5.5).

**Hard prerequisite:** Layer 2 plan Wave 4 complete (hospital discharge reconciliation needed for surveillance rules around hospital→RACF transitions).

### Wave 5 Task 1 — Tier 4 surveillance rule authoring (~50 rules)

- **Goal:** Trajectory-based surveillance, lifecycle-aware deadlines, and outcome-monitoring rules. Examples: AntipsychoticReviewOverdue3mo, ConsentExpiringWithin30days, MonitoringPlanOverdueObservation, eGFRTrajectoryDecline90d, WeightLossTrajectory90d, BehaviouralEpisodeFrequencyChange.
- **Files:** `shared/cql-libraries/tier-4-surveillance/*.cql` (~50 rules).
- **Acceptance:** Each rule fires informationally (does not consume display surface) unless the trajectory crosses a clinical threshold; suppression-class 5 (substrate-state) governs surfacing.
- **Effort estimate:** 12 days.
- **Dependencies:** Layer 2 plan Wave 2 (running baselines), Wave 3 (Monitoring state machine).

### Wave 5 Task 2 — kb-31-scope-rules service scaffolding

- **Goal:** Stand up kb-31-scope-rules Go service (port 8139) with PostgreSQL store, REST + gRPC API, parser for ScopeRule YAML.
- **Files:** `kb-31-scope-rules/cmd/server/main.go`, `kb-31-scope-rules/internal/store/`, `internal/parser/`, migrations.
- **Acceptance:** Service starts; `/health` green; one ScopeRule round-trips from YAML through store to evaluator.
- **Effort estimate:** 4 days.
- **Dependencies:** Wave 4B Task 1 (DSL — ScopeRule shares schema with AuthorisationRule data structure).

### Wave 5 Task 3 — Victorian PCW exclusion ScopeRule deployment

- **Goal:** Author and deploy `AUS-VIC-PCW-S4-EXCLUSION-2026-07-01` ScopeRule per the contractual schema below. Deploy in shadow mode 1 May 2026; activate (enforced) by 1 July 2026 (start of grace period); enforcement begins 29 September 2026.
- **Contractual ScopeRule data structure** (per Layer 3 v2 doc Part 5.5.2):
  ```yaml
  scope_rule:
    id: string
    jurisdiction: string
    category: string                  # e.g. medication_administration_scope_restriction
    effective_period:
      start_date: ISO8601
      end_date: ISO8601 | null
      grace_period_days: integer | null
    applies_to:                       # see AuthorisationRule for sub-fields
      role: string
      action_class: enum
      medication_schedule: [string]
      medication_class_includes: [string]
      resident_self_administering: boolean
    evaluation:
      decision: enum
      fallback_required: boolean
      fallback_eligible_roles: [string]
    source:
      legislative_reference: string
      source_id: string
      source_version: string
      source_url: string
    audit:
      recordkeeping_required: boolean
      recordkeeping_period_years: integer
  ```
- **Files:** `kb-31-scope-rules/data/AU/VIC/pcw-s4-exclusion-2026-07-01.yaml`.
- **Acceptance:** Victorian pilot facility runs in shadow mode for ≥30 days with no false-deny on legitimate RN administrations; activation flips to enforced on 1 July 2026; enforcement test on 29 September 2026 successful. Compliance metric (Vaidshala v2 Revision Mapping success metric "Victorian PCW exclusion compliance rate") reaches 100% by end of grace period.
- **Effort estimate:** 4 days authoring + governance + shadow-mode monitoring.
- **Dependencies:** Wave 5 Task 2; Layer 1 audit Wave 6 (legislation source extracted).

### Wave 5 Task 4 — Designated RN prescriber ScopeRule deployment

- **Goal:** Author and deploy `AUS-NMBA-DRNP-PRESCRIBING-AGREEMENT-2025-09-30` ScopeRule. Live for first endorsed RN prescribers from mid-2026.
- **Files:** `kb-31-scope-rules/data/AU/national/drnp-prescribing-agreement.yaml`.
- **Acceptance:** Synthetic designated RN prescriber + valid prescribing agreement → grant decision; missing mentorship → conditional deny with reason; agreement scope mismatch → deny with reason.
- **Effort estimate:** 3 days.
- **Dependencies:** Wave 5 Task 2.

### Wave 5 Task 5 — Tasmanian pharmacist co-prescribing pilot ScopeRule (deferred to V2 if pilot timing slips)

- **Goal:** Author `AUS-TAS-PHARMACIST-COPRESCRIBE-PILOT-2026` ScopeRule scaffolding. If commercial Move 1 (Vaidshala v2 Revision Mapping Part 7) lands the platform as the digital substrate for the pilot, deploy live. Otherwise, defer activation to V2 with the rule structure ready.
- **Files:** `kb-31-scope-rules/data/AU/TAS/pharmacist-coprescribe-pilot-2026.yaml`.
- **Acceptance:** Either deploy live (pilot integration confirmed) or stage as DRAFT with clear activation gate documented.
- **Effort estimate:** 3 days.
- **Dependencies:** Wave 5 Task 2; Vaidshala v2 Revision Mapping Move 1 outcome.

### Wave 5 Task 6 — ACOP credential ScopeRules

- **Goal:** Author ACOP APC training credential ScopeRule + verification flow (mandatory from 1 July 2026 per $350M program).
- **Files:** `kb-31-scope-rules/data/AU/national/acop-apc-credential.yaml`.
- **Acceptance:** ACOP pharmacist with current APC credential → grant; expired or missing credential → deny with verification path surfaced.
- **Effort estimate:** 2 days.
- **Dependencies:** Wave 5 Task 2.

### Wave 5 Task 7 — ScopeRule-aware CompatibilityChecker integration

- **Goal:** When a ScopeRule changes (Event D), CompatibilityChecker marks affected CQL defines STALE and routes to governance review per Layer 3 v2 doc Part 4.2.
- **Files:** `shared/cql-toolchain/compatibility_checker.py` (extended).
- **Acceptance:** Synthetic ScopeRule change marks ≥3 expected defines STALE; no false positives.
- **Effort estimate:** 3 days.
- **Dependencies:** Wave 1 Task 3, Wave 5 Tasks 3-6.

**Wave 5 exit criterion:** ~50 Tier 4 rules ACTIVE; total rule library ~200 rules; ScopeRule engine in production with ≥4 jurisdictional rules; Victorian PCW exclusion enforced from 29 September 2026 with audit-defensible compliance trail.

---

## Wave 6 — Continuous tuning (ongoing from Week 26)

**Goal:** Tune the rule library against production override-rate data; retire unused rules; track source-update SLA; add rules from coronial / ACQSC findings.

### Wave 6 Task 1 — Override-rate analytics + rule retirement

- **Goal:** Weekly review of override reasons, override rate per rule, fire-rate-vs-action-rate ratio. Retire rules whose override rate exceeds 70% over 30 days unless clinical lead overrides.
- **Files:** `kb-30-authorisation-evaluator/internal/audit/`, analytics dashboards.
- **Acceptance:** Override-rate target < 5% library-wide reached by Week 40 (pilot data dependent).
- **Effort estimate:** Ongoing (1 day/week).
- **Dependencies:** Wave 5 complete.

### Wave 6 Task 2 — Suppression-class 5 + 6 tuning

- **Goal:** Tune Class 5 (substrate-state) and Class 6 (authorisation-context) suppressions per Layer 3 v2 doc Part 2.1-2.2.
- **Files:** `shared/cql-libraries/helpers/SuppressionHelpers.cql`.
- **Acceptance:** Volume budget (≤5 actionable alerts/resident/day) holds across pilot data; suppression effectiveness tracked.
- **Effort estimate:** Ongoing.
- **Dependencies:** Wave 6 Task 1.

### Wave 6 Task 3 — Source-update 7-day SLA infrastructure

- **Goal:** Implement the 7-day SLA for clinical guideline + regulatory ScopeRule + substrate schema + source-authority version-pin updates per Layer 3 v2 doc Part 4.3.
- **Files:** `shared/cql-toolchain/source_update_tracker.py`, on-call rotation docs.
- **Acceptance:** Each of the four source classes has a tested propagation path within 7 days; ScopeRule-specific SLA tighter (Victorian PCW enforcement deadline).
- **Effort estimate:** 5 days initial + ongoing.
- **Dependencies:** Wave 5 complete.

### Wave 6 Task 4 — Coronial / ACQSC finding intake

- **Goal:** Workflow for adding rules from coronial inquests and ACQSC complaint findings.
- **Files:** `claudedocs/governance/2026-XX-coronial-finding-rule-intake.md` template.
- **Acceptance:** Process documented; first finding intake completed.
- **Effort estimate:** Ongoing.
- **Dependencies:** Wave 6 Task 1.

---

## Cross-cutting deliverables

### Cross-cutting Task 1 — CompatibilityChecker bidirectional contract

- **Goal:** Maintain Event A (CQL bundle changes), Event B (L3 facts change), Event C (substrate schema changes), Event D (regulatory ScopeRule changes) per Layer 3 v2 doc Part 4.
- **Files:** `shared/cql-toolchain/compatibility_checker.py` (final form).
- **Acceptance:** All four event classes route to the correct handler; STALE marking blocks deployment until governance review.
- **Effort estimate:** Touched in Wave 1 Task 3 + Wave 5 Task 7; final hardening Week 26.
- **Dependencies:** Waves 1-5.

### Cross-cutting Task 2 — 7-day SLA infrastructure

- **Goal:** Day-1 detection through Day-7 deploy pipeline per Layer 3 v2 doc Part 4.3.
- **Files:** Wave 6 Task 3 deliverables.
- **Acceptance:** End-to-end test from synthetic source update to deployed CQL define inside 7 days.
- **Dependencies:** Wave 6 Task 3.

### Cross-cutting Task 3 — Worked rule example library

- **Goal:** Maintain a curated set of canonical worked examples spanning the Tiers, used as authoring templates and onboarding material.
- **Files:** `shared/cql-libraries/examples/`.
- **Worked examples (one per Tier + Authorisation integration pattern):**

#### Example A — Tier 1 baseline-aware trajectory rule

```cql
library HyperkalemiaRiskTrajectory version '2.0.0'
using FHIR version '4.0.1'
include MedicationHelpers called Med
include ClinicalStateHelpers called CS
include ConsentStateHelpers called Consent
include AuthorisationHelpers called Auth

context Patient

define "Hyperkalemia Risk Trajectory":
  (
    CS.PotassiumDeltaFromBaseline(7) > 0.8
      or Med.LatestObservationValue('potassium') > 5.5
  )
    and Med.IsOnClass('ACEi/ARB')
    and Med.IsOnClass('K-sparing diuretic')
    and not Med.RecentDoseChangeAffectingPotassium(14)
    and not CS.HasActiveConcern('AKI_watching')

define "Care Intensity Modifier":
  if CS.CareIntensity() = 'palliative'
  then 'suppress'
  else 'fire'

define "Recommended Action":
  if Auth.HasAvailablePrescriberForClass('Schedule_4', 'dose_adjustment')
  then 'reduce_ACEi_dose_with_repeat_K_72h'
  else 'telehealth_or_ED_transfer_protocol'
```

#### Example B — Tier 2 consent-gated deprescribing rule

```cql
library AntipsychoticBPSDDeprescribeReview version '2.0.0'
using FHIR version '4.0.1'
include MedicationHelpers called Med
include ConsentStateHelpers called Consent
include ClinicalStateHelpers called CS

context Patient

define "Antipsychotic Long Term Without BPSD Documentation":
  exists(Med.MedicineUsesByClass('Antipsychotic') M
         where M.duration_days > 90)
    and not exists([Condition: "BPSD Severe ValueSet"] C
                   where CS.WithinDays(C.recordedDate, 180))
    and CS.CareIntensity() != 'palliative'

define "Recommendation Type":
  if Consent.HasActiveConsent('Antipsychotic', 'deprescribe_review')
  then 'medication_change_recommendation'
  else 'consent_gathering_recommendation'

define "Consent Gathering Template":
  if "Recommendation Type" = 'consent_gathering_recommendation'
  then 'discuss_psychotropic_with_SDM_template'
  else null
```

#### Example C — Tier 3 quality-gap PHARMA-Care rule

```cql
library PHARMACareDomain3PolypharmacyTrigger version '2.0.0'
using FHIR version '4.0.1'
include MedicationHelpers called Med
include ClinicalStateHelpers called CS

context Patient

define "Polypharmacy 10+ With Quality Gap":
  Med.ActiveMedicineUseCount() >= 10
    and not exists(Med.MedicationReviewsWithinDays(180))
    and CS.CareIntensity() != 'palliative'

define "PHARMA-Care Indicator":
  Tuple {
    domain: 'D3_QualityUseOfMedicines',
    indicator: 'polypharmacy_unreviewed',
    fired_on: Now()
  }
```

#### Example D — Tier 4 surveillance trajectory rule

```cql
library eGFRDeclineSurveillance version '2.0.0'
using FHIR version '4.0.1'
include ClinicalStateHelpers called CS
include MedicationHelpers called Med

context Patient

define "eGFR 90 Day Decline > 20%":
  CS.RelativeDeltaFromBaseline('egfr', 90) <= -20
    and CS.BaselineConfidence('egfr') in {'high','medium'}

define "Surveillance Action":
  if CS.HasActiveConcern('AKI_watching') then 'suppress'
  else if Med.IsOnClass('nephrotoxic_compound') then 'fire_actionable'
  else 'fire_informational'
```

#### Example E — Authorisation evaluator integration pattern

```cql
// Inside any rule needing Schedule 8 routing
define "Action Routing":
  case
    when Auth.HasAvailablePrescriberForClass('Schedule_8','dose_adjustment')
      then 'route_to_authorised_prescriber'
    when Auth.HasAvailablePrescriberForClass('Schedule_8','dose_adjustment','telehealth')
      then 'route_to_telehealth_prescriber'
    else 'ED_transfer_protocol'
  end
```

### Cross-cutting Task 4 — Sample CDS Hooks v2.0 response shape

- **Goal:** Document the contractual CDS Hooks `order-select` / `order-sign` response emitted by the rule library.
- **Files:** `shared/cql-toolchain/docs/cds-hooks-response-shape.md`.
- **Sample response:**

```json
{
  "cards": [
    {
      "summary": "Antipsychotic >90 days without recent BPSD documentation",
      "indicator": "warning",
      "source": {
        "label": "Vaidshala Tier 2 — ADG 2025",
        "url": "https://deprescribing.com/...",
        "topic": { "code": "antipsychotic_review", "display": "Antipsychotic deprescribing review" }
      },
      "detail": "Resident on risperidone 0.25 mg BID since 2026-02-15. Behavioural chart shows zero agitation episodes for 14 days. ADG 2025 recommendation 47.",
      "suggestions": [
        {
          "label": "Generate consent-gathering recommendation",
          "actions": [
            {
              "type": "create",
              "description": "Discuss antipsychotic deprescribing with SDM",
              "resource": { "resourceType": "RequestOrchestration", "id": "..." }
            }
          ]
        }
      ],
      "links": [
        { "label": "ADG 2025 protocol", "url": "https://...", "type": "absolute" },
        { "label": "Resident Workspace", "url": "vaidshala://resident/...", "type": "smart" }
      ]
    }
  ]
}
```

---

## Acceptance Criteria Coverage Map (Layer 3 v2 doc → Waves)

| Layer 3 v2 doc Part | Wave / Task |
|---|---|
| Part 0 (what changed) | Plan header context |
| Part 0.5.1-0.5.6 (substrate substrate) | Wave 0 (scoping); Wave 1 Task 1 (helpers); Waves 2-5 (consumption) |
| Part 1.1 (rule_specification.yaml extensions) | Wave 0 Task 3 |
| Part 1.2 (Stage 4 test cases) | Wave 1 Task 2; embedded in every rule task in Waves 2-5 |
| Part 2.1 (Class 5 substrate-state suppression) | Wave 2; Wave 6 Task 2 |
| Part 2.2 (Class 6 authorisation-context suppression) | Wave 4 Stream B; Wave 6 Task 2 |
| Part 2.3 (volume budget under substrate) | Wave 6 Task 1 |
| Part 3.1 (Authorisation availability modifier) | Wave 4 Stream B Task 3 |
| Part 3.2 (deferred awaiting consent priority) | Wave 2 Task 3; Wave 3 Task 4 |
| Part 3.3 (GP-perspective re-ranking) | Wave 1 Task 4 (CDS Hooks emitter); Layer 2 plan Wave 5 (decision packet) |
| Part 4.1-4.3 (CompatibilityChecker A/B/C/D + 7-day SLA) | Wave 1 Task 3; Wave 5 Task 7; Wave 6 Task 3 |
| Part 4.5.1 (why not CQL) | Plan architecture preamble |
| Part 4.5.2 (rule format DSL) | Wave 4B Task 1 (contractual schema in plan) |
| Part 4.5.3 (cache strategy) | Wave 4B Task 4 |
| Part 4.5.4 (audit query API) | Wave 4B Task 6 |
| Part 4.5.5 (Sunday-night-fall) | Wave 4B Task 7 |
| Part 4.5.6 (build sequencing) | Wave 4 Stream B (full schedule) |
| Part 5.1-5.7 (six-wave roadmap) | Waves 0-6 (one-to-one) |
| Part 5.8 (effort summary) | Plan total: 25 weeks to MVP per Wave headers |
| Part 5.5.1 (why ScopeRules-as-data) | Plan architecture preamble |
| Part 5.5.2 (data structure) | Wave 5 Task 3 (contractual schema in plan) |
| Part 5.5.3 (where parsed from) | Wave 5 Tasks 3-6 (sourced from Layer 1 v2 Category C) |
| Part 5.5.4 (runtime evaluation) | Wave 4B Task 3; Wave 5 Task 2 |
| Part 5.5.5 (multi-jurisdiction expansion) | Wave 5 Tasks 3-6 (architecture supports new jurisdictions as data deploys) |
| Part 6 (failure modes) | Risk register below |
| Part 7 (three sharp recommendations) | Plan structure honours all three |
| Part 8 (closing) | Plan timeline honours 25-week budget |

**No part of the doc is dropped.** Sections that are pure exposition (Part 0, Part 0.2, Part 0.3, Part 7, Part 8) are reflected in plan structure rather than mapped to a Wave.

---

## Risk register

### Risks inherited from Layer 3 v1.0

The six v1.0 risks (ADG licensing, AMH timeline, GP integration tooling fragility, acceptance-rate ceiling, eNRMC vendor cooperation, restrictive-practice legal exposure) carry forward unchanged.

### Risks 7-11 from Vaidshala v2 Revision Mapping Part 8

**Risk 7 — Jurisdictional regulatory fragmentation.**
- Mitigation: ScopeRules-as-data (Wave 5) — every state-level change becomes a data deploy, not an engineering project.

**Risk 8 — Designated RN prescriber rollout uncertainty.**
- Mitigation: Wave 5 Task 4 deploys ScopeRule scaffolding; do not assume meaningful population in V1.

**Risk 9 — Pharmacist autonomous prescribing acceleration.**
- Mitigation: AuthorisationRule DSL is role-extensible; new role types deploy as data (Wave 4B Task 1 schema supports it).

**Risk 10 — Hospital integration depth.**
- Mitigation: Wave 4A Task 2 (transition rules) authored against PDF-upload-grade data first; richer integrations land in Layer 2 plan Wave 4 / V2.

**Risk 11 — PHARMA-Care framework evolution.**
- Mitigation: Wave 4A Task 1 produces indicators with configurable computation; quarterly re-evaluation cadence built into Wave 6.

### Layer 3-specific risks

**Risk 12 — CQL authoring tooling immaturity in Australia.**
- HAPI FHIR engine is the most mature open-source CQL execution; Smile CDR is commercial. AU-specific value-set tooling (AMT, SNOMED-CT-AU, RxNorm cross-walk) is thin.
- Mitigation: Wave 1 Task 1 includes cross-walk helpers; Wave 1 Task 2 includes substrate-aware lint; budget +20% on Wave 2 first-rule authoring as tooling shake-down.

**Risk 13 — Sub-500ms p95 may slip on first deploy.**
- Cold-cache evaluation under realistic load is hard to estimate before Wave 4B Task 4 lands.
- Mitigation: latency monitoring from Wave 4B Task 3 onward; cache strategy designed for warm-path < 200ms; accept p95 < 500ms in V1 per Vaidshala v2 success metric, target < 200ms in V2.

**Risk 14 — Override-rate target may not hit < 5% in V1.**
- v1.0 doc said this is a stretch goal; pilot data may show 8-12% in V1.
- Mitigation: Wave 6 Task 1 weekly tuning; rule retirement gate at 70% override over 30 days; target < 5% by Week 40, not Week 25.

**Risk 15 — Layer 2 hard-prerequisite Waves slip.**
- Wave 2 of this plan needs Layer 2 plan Wave 2 (running baselines); Wave 5 needs Layer 2 plan Wave 4 (hospital discharge reconciliation).
- Mitigation: Wave 0 cross-team scoping locks the Layer 2 contract; weekly cross-team standup Layer 2 ↔ Layer 3 from Week 1.

**Risk 16 — ScopeRule misencoding (Layer 3 v2 doc Failure 9).**
- A misencoded ScopeRule could deny legitimate actions (clinical harm) or grant illegitimate ones (regulatory exposure).
- Mitigation: dual-review governance for all ScopeRule changes (clinical pharmacist + medical director + legal review for jurisdictional rules); regression test suite covering all currently-deployed rules; staged rollout (silent mode first, then enforced) — Wave 5 Task 3 builds in 30-day shadow mode for Victorian PCW exclusion.

**Risk 17 — Substrate concurrency bugs (Layer 3 v2 doc Failure 7).**
- Two state machines transitioning simultaneously must produce coherent linked EvidenceTrace entries.
- Mitigation: Layer 2 plan responsibility, but Layer 3 EvidenceTrace assertions in every rule test catch downstream symptoms.

**Risk 18 — Authorisation evaluator latency creep (Layer 3 v2 doc Failure 8).**
- As rules accumulate, latency grows.
- Mitigation: strict latency monitoring (Wave 4B Task 3 onward); alert on p95 > 300ms; rule deduplication; rule precompilation where possible.

---

## Open dependencies (Layer 1 + Layer 2 hard gates)

### Layer 1 audit gaps blocking Layer 3 Waves

| Layer 1 gap | Blocks | Resolution |
|---|---|---|
| `PRESCRIBING_OMISSION` fact_type schema decision | Wave 1 Task 2 (validator), Wave 3 Task 2 (START rules) | Pre-Wave Task 1 |
| L6 governance audit trail verification for AU rules | Wave 2 Task production deploy, Wave 3 Task 5 governance | Pre-Wave Task 2 |
| Residual RxCUI gaps in ADG 2025 / Wang 2024 | Wave 3 Task 1 (ADG mapping), Wave 3 Task 2 (rule authoring) | Pre-Wave Task 3 |
| ICD-10-AM license block | Any rule referencing ICD-10-AM (none planned, but enforced via Pre-Wave Task 4) | Pre-Wave Task 4 documents policy |
| ADG 2025 Pipeline-2 extraction | Wave 3 Task 1 mapping spreadsheet | Layer 1 audit Wave 5 free-alternative path; PDFs landed 2026-04-30 |
| TGA PI/CMI extraction | Family-facing CMI surfacing (Tier 3-4 quality-gap rules referencing CMI) | Layer 1 audit Wave 2 remaining work; not a hard blocker for Tier 1-2 |
| Wave 6 source PDFs (RANZCP, ACSQHC, NPS) | Tier 3-4 specialty rules | Manual procurement; not a hard blocker for Tier 1-3 deprescribing |

### Layer 2 plan Waves required as hard prerequisites

| Layer 3 Wave | Layer 2 plan Wave hard-required | What it gives us |
|---|---|---|
| Wave 0 | Layer 2 plan Wave 1 | Substrate entity model finalised |
| Wave 1 | Layer 2 plan Wave 1 | Substrate read-side APIs deployed in dev |
| Wave 2 | Layer 2 plan Wave 2 | Clinical state machine with running baselines (potassium, eGFR, sodium, weight, behavioural episodes) |
| Wave 2 Task 2 | Layer 2 plan Wave 3 | Monitoring state machine (active monitoring plans, observation expectations) |
| Wave 2 Task 2 | Layer 2 plan Wave 5 | Active concerns in Clinical state |
| Wave 2 Task 3 | Layer 2 plan Wave 6 | Consent state machine |
| Wave 2 Task 4 | Layer 2 plan Wave 1 | Authorisation seam read API (Person+Role with credentials) |
| Wave 2 Task 5 | Layer 2 plan Wave 2 | Care intensity tag in Clinical state |
| Wave 3 | Layer 2 plan Wave 2 (mandatory), Wave 4 (soft — discharge-derived intent) | Running baselines + intent population from discharge |
| Wave 4A Task 1 | Layer 2 plan Wave 7 | PHARMA-Care indicator computation surface |
| Wave 4A Task 2 | Layer 2 plan Wave 4 | Hospital discharge reconciliation pipeline |
| Wave 4B Task 5 | Layer 2 plan Wave 1 | Substrate change events on Kafka |
| Wave 4B Task 6 | Layer 2 plan Wave 1 | EvidenceTrace graph queryable |
| Wave 5 Task 1 | Layer 2 plan Wave 2 + Wave 3 | Trajectory + monitoring overdue surveillance signals |
| Wave 5 Task 3 | Layer 1 audit Wave 6 | Victorian PCW legislation extracted as ScopeRule source |

**Cross-team gate:** Weekly Layer 2 ↔ Layer 3 standup from Week 1 of Wave 0; explicit hand-off acceptance test at every Layer 2 Wave exit before the matching Layer 3 Wave starts.

---

## Total effort summary

| Wave | Weeks | Output |
|---|---|---|
| Pre-Wave | 1 | Layer 1 audit blockers closed |
| Wave 0 | 2 | Substrate scoping docs |
| Wave 1 | 2 | Authoring infrastructure + helpers + 3 anchor defines |
| Wave 2 | 4 | ~25 Tier 1 rules ACTIVE |
| Wave 3 | 8 | ~75 Tier 2 rules ACTIVE |
| Wave 4 | 6 | ~50 Tier 3 rules ACTIVE + Authorisation evaluator operational |
| Wave 5 | 3 | ~50 Tier 4 rules ACTIVE + ScopeRules deployed (incl. Victorian PCW exclusion) |
| Wave 6 | Ongoing from Week 26 | Continuous tuning, override-rate to < 5% by Week 40 |
| **Total to MVP coverage** | **26 weeks** (1 Pre-Wave + 25 Waves) | **~200 rules + Authorisation evaluator + ScopeRules engine + 7-day SLA infrastructure** |

This honours the Layer 3 v2 doc Part 5.8 budget (25 weeks) plus the 1-week Pre-Wave for Layer 1 blocker closure. The estimate is honest; the Authorisation evaluator + ScopeRules engine are not compressible without quality compromise.
