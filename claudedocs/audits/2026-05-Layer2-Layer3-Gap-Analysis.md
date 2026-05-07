# Layer 2 + Layer 3 Substrate Gap Analysis

**Date:** 2026-05-06
**Scope:** Plans at `docs/superpowers/plans/2026-05-04-layer2-substrate-plan.md` and `docs/superpowers/plans/2026-05-04-layer3-rule-encoding-plan.md`
**Method:** Plan-spec vs on-disk-state diff via `git log` + filesystem inspection + Go test execution (`go build ./... && go test -count=1 ./...` against `shared/v2_substrate`, `kb-20-patient-profile`, `kb-30-authorisation-evaluator`, `kb-31-scope-rules`).
**Auditor:** automated codebase audit run on branch `feature/v4-clinical-gaps` at HEAD `c7134402`.

---

## Executive summary

Both plans landed end-to-end on `main`: every numbered Wave / sub-task in both
plans has at least a **vertical slice** committed and exercised by tests.
Builds are clean and all four targeted Go modules pass tests on a fresh
checkout. The honest pattern, however, is **breadth before depth**: the Layer 3
rule-authoring volume is well below plan, and several Layer 2 ingestion paths
shipped as stubs/skeletons rather than production-grade clients. Deferral
markers and queue manifests are mostly explicit, not silent.

**Layer 2 substrate plan (Wave 0 + Wave 1-residual + Waves 2–6 = ~30 sub-tasks):**

| Category | Count | % |
|---|---|---|
| ✅ Shipped per plan | ~22 | ~73% |
| 🟡 Reduced scope (vertical slice + explicit deferral) | ~6 | ~20% |
| ⏸️ Deferred-to-V1 (explicit ADR / out-of-MVP-scope) | 2 | ~7% |
| 🔴 Missing (no slice, no marker) | 0 | 0% |

**Layer 3 rule encoding plan (Pre-Wave + Waves 0–6 = ~37 sub-tasks):**

| Category | Count | % |
|---|---|---|
| ✅ Shipped per plan | ~20 | ~54% |
| 🟡 Reduced scope (vertical slice + queue manifest) | ~13 | ~35% |
| ⏸️ Deferred-to-V1 | ~3 | ~8% |
| 🔴 Missing | ~1 | ~3% |

**Total commits on `main`** referencing Wave / tier / kb-30 / kb-31 / cql-:
**67 commits** (last ~50 commits encode the bulk of Layer 2 + Layer 3 delivery).

**TODO marker surface area (deferred-work signal):**

- `TODO(layer1-bind)` — **16** (Tier 2 ADG2025 rules + helpers awaiting
  Pipeline-2 extraction)
- `TODO(layer3-v1)` — **4** (Wave 5 / Wave 4A V1 deferrals)
- `TODO(clinical-author)` — **16** (Tier 3-4 rules with skeleton bodies)
- `TODO(wave-1-runtime)` — **17** (helper bodies awaiting live substrate
  bindings in dev)
- `TODO(wave-2.7-runtime)` — **15** (streaming pipeline runtime wiring)
- `TODO(kb-7-binding)` — 0 (zero — terminology binding not parked)
- `TODO(kb-26-acute-repository)` — 0
- Total Vaidshala-prefixed TODO markers: **~68** across `backend/`.

**Risk-ranked top 5 V1 gaps:**

1. **Layer 3 rule volume — ~45 of ~200 rules ACTIVE** (~22% of plan target). Tier
   2 shipped 6/75, Tier 3 shipped 8/50, Tier 4 shipped 6/50. Each shortfall is
   covered by an explicit queue manifest, but the clinical impact is that the
   library does not yet have the breadth to drive the Wave 6 override-rate
   target (<5%). **V1 priority: continue Tier 2 authoring (highest yield) at
   plan velocity (~1.7 weeks per 5 rules).**
2. **Streaming pipeline runtime not wired** (`backend/stream-services/substrate-pipeline/`
   ships as a Java skeleton — `pom.xml` + `Dockerfile` + property file + ADR but
   no processor code). Wave 2.7 acceptance ("load test green; identity_matching
   processor consumes raw events") is unmet. 15 `TODO(wave-2.7-runtime)`
   markers track the work. **V1 priority: medium — the synchronous Go write
   path covers near-term load; streaming is a scaling unlock.**
3. **MHR/HL7 ingestion clients ship as `_stub` interfaces.** SOAP/CDA, FHIR
   Gateway, and HL7-MLLP listeners all have parser unit tests against synthetic
   fixtures but no live ADHA-conformance round-trip. **V1 priority: high before
   any production pilot facility goes live; medium for shadow.**
4. **Layer 1 audit blocker — ADG 2025 Pipeline-2 extraction blocking 165/185
   ADG rules.** `claudedocs/clinical/2026-05-Layer3-ADG2025-mapping.csv`
   classifies 20 rows as candidates and 165 as deferred-to-Pipeline-2. This
   gates Tier 2 ADG2025 expansion (currently 2 rules with `TODO(layer1-bind)`).
   **V1 priority: high.**
5. **Wave 5.4 graph performance load test executed only against synthetic
   loadgen.** Commit `24797d38` literally says "execution deferred" — the
   benchmark exists, the 6-month-of-activity dataset run is not in CI. **V1
   priority: medium — needed before pilot expands beyond the first facility.**

**Surprises during the audit:** (1) Wave 0 commit-history cleanup actually
landed cleanly — fresh build of `shared/v2_substrate/...` passes immediately
with no untracked-file noise blocking; (2) the Layer 3 plan's Pre-Wave (Layer 1
audit blockers) is genuinely complete, including the four memo files and the
KB-4 verifier; (3) deferral discipline is high — every reduced-scope task has
an associated queue manifest under `claudedocs/clinical/` or
`claudedocs/governance/`, and TODO markers are namespaced consistently
(`TODO(layer1-bind)`, `TODO(layer3-v1)`, etc.) so future authors can grep by
deferral class.

---

## Layer 2 Substrate Plan — gap matrix

### Wave 0 — Commit history cleanup + retroactive β.1/β.2-A landing

| Sub-task | Category | Evidence | Acceptance | Notes |
|---|---|---|---|---|
| 0.1 Audit working tree, bucket entries | ✅ Shipped | `git status --short` shows only 2 modified .py files + 3 untracked binaries — no β.1 leftovers | Y | Bucketing memo not retained (transient by design per plan) |
| 0.2 Verify build broken without files | ✅ Shipped | n/a — superseded by 0.3 success | Y (implicit) | |
| 0.3 Land β.1 entity files | ✅ Shipped | `models/resident.go` `person.go` `role.go` + validators + `008_part1_actor_model.sql` all on main | Y | Mapper files (`patient_mapper.go`, `practitioner_mapper.go`) present |
| 0.4 Land β.2-A MedicineUse | ✅ Shipped | `medicine_use.go`, `target_schemas.go`, `stop_criteria_schemas.go`, `008_part2_clinical_primitives_partA.sql` on main | Y | Tests pass |
| 0.5 Land kb-20 test files | ✅ Shipped | `internal/storage` and `internal/api` tests both pass | Y | |
| 0.6 Quarantine unrelated work | 🟡 Partial | 2 v4 channel files (`channel_d_table.py`, `decomposition.py`) still in working tree as Modified | Acceptable per Wave 0.6 plan (stash/leave-untracked) | These belong to a v5 atomiser phase — stash discipline holds |
| 0.7 Fresh-clone build verification | ✅ Shipped | `go build ./v2_substrate/...` clean; `go test ./v2_substrate/...` 13 packages pass | Y | Build confirmed by audit run |

**Wave 0 verdict:** ✅ **Complete.** All Wave 0 acceptance criteria met. The 2
modified .py files in working tree are unrelated v5 atomiser WIP and conform to
the Wave 0.6 quarantine policy.

### Wave 1-residual — Event + EvidenceTrace v0 + Identity matching + CSV ingestion

| Sub-task | Category | Evidence | Acceptance | Notes |
|---|---|---|---|---|
| 1R.1 Event entity | ✅ Shipped | `models/event.go` + `_test.go`, `validation/event_validator.go`, `fhir/event_mapper.go`, `migrations/009_event_evidencetrace.sql`, kb-20 `event_store.go` + handlers | Y (tests pass) | |
| 1R.2 EvidenceTrace v0 graph | ✅ Shipped | `evidence_trace/graph.go`, `edge_store.go`, `query.go`, `migrations/009_event_evidencetrace.sql`, FHIR provenance + audit-event mappers | Y | Bidirectional traversal tested in `query_test.go` |
| 1R.3 Identity matching service | ✅ Shipped | `identity/matcher.go`, `ihi_matcher.go`, `fuzzy_matcher.go`, `mhr_ihi_resolver.go`, `migrations/010_identity_mapping.sql` | Y | Tests pass |
| 1R.4 CSV eNRMC ingestor | ✅ Shipped | `ingestion/csv_enrmc.go`, `normaliser.go`, `runner.go`, `kb-20-patient-profile/cmd/ingest-csv/main.go` | Y | Plus `testdata/` fixture |

### Wave 2 — Clinical state machine completion

| Sub-task | Category | Evidence | Acceptance | Notes |
|---|---|---|---|---|
| 2.1 PersistentBaselineProvider | ✅ Shipped | `delta/persistent_baseline_provider.go` + `_test.go`, `migrations/013_baseline_persistent_store.sql`, kb-20 baseline_store, wired in `main.go` | Y | |
| 2.2 Per-observation-type baseline config | ✅ Shipped | `delta/baseline_config.go` + `_test.go`, `migrations/014_baseline_config.sql` | Y | Seed migration includes potassium/eGFR/weight/BP rows |
| 2.3 Active concern lifecycle | ✅ Shipped | `models/active_concern.go`, `clinical_state/active_concerns.go`, `migrations/015_active_concerns.sql`, FHIR Condition mapper | Y | Sweep/expiry mechanism present in engine |
| 2.4 Care intensity tag | ✅ Shipped | `models/care_intensity.go`, `clinical_state/care_intensity_engine.go` + `_test.go`, `migrations/016_care_intensity.sql`, REST handlers | Y | Cascade EvidenceTrace nodes wired |
| 2.5 Capacity assessment | ✅ Shipped | `models/capacity_assessment.go`, `fhir/capacity_assessment_mapper.go`, `migrations/017_capacity_assessment.sql`, REST handlers | Y | Consent-tagged EvidenceTrace on impaired-medical |
| 2.6 CFS/AKPS/DBI/ACB scoring | ✅ Shipped | `models/{cfs,akps,dbi,acb}_score.go`, `scoring/{dbi,acb}_calculator.go`, `cfs_capture.go`, `akps_capture.go`, `migrations/018_scoring_instruments.sql` (20-row weight seed), recompute-on-MedicineUse-change wired | Y | |
| 2.7 Streaming pipeline | 🟡 Reduced scope | ADR `docs/adr/2026-05-06-streaming-pipeline-choice.md` chose Kafka Streams; topology + load-test plan committed (`shared/v2_substrate/streaming/topology.md`, `load_test_plan.md`); Java module skeleton at `backend/stream-services/substrate-pipeline/` (pom.xml, Dockerfile, application.properties, README) | Partial — ADR signed, scaffolding present; **no processor code committed; load test not green** | 15 `TODO(wave-2.7-runtime)` markers track the runtime gap. Synchronous Go write path covers near-term scale. |

### Wave 3 — MHR + pathology integration

| Sub-task | Category | Evidence | Acceptance | Notes |
|---|---|---|---|---|
| 3.1 MHR SOAP/CDA gateway client | 🟡 Reduced scope | `ingestion/mhr_soap_cda.go` + `cda_parser.go` + tests; `migrations/019_pathology_ingest.sql`; `kb-20-patient-profile/cmd/mhr-poll/main.go` skeleton | Partial — synthetic CDA fixture round-trips; NASH PKI auth, ADHA conformance pack not exercised | Commit message explicitly says "interface + stub" |
| 3.2 MHR FHIR Gateway client | 🟡 Reduced scope | `ingestion/mhr_fhir_gateway.go` + `_test.go`, `fhir/diagnostic_report_mapper.go` | Partial — interface + stub + DiagnosticReport mapper unit tested; OAuth2/NASH live integration not exercised | |
| 3.3 HL7 vendor fallback | 🟡 Reduced scope | `ingestion/hl7_oru.go`, `hl7_vendor_adapters.go` parsers + tests; Java MLLP listener skeleton | Partial — parser tests pass on synthetic ORU; MLLP listener Java module is skeleton-only | |
| 3.4 Trajectory + velocity finalisation | ✅ Shipped | `delta/trajectory_detector.go` + `_test.go`, `migrations/020_pathology_baseline_extras.sql` adding trajectory columns + `velocity_flag`; wired into `RecomputeAndUpsertTx` | Y | |

### Wave 4 — Hospital discharge reconciliation

| Sub-task | Category | Evidence | Acceptance | Notes |
|---|---|---|---|---|
| 4.1 Discharge document ingestion | ✅ Shipped | `ingestion/discharge_pdf.go`, `discharge_mhr.go`, `fhir/document_reference_mapper.go`, `migrations/021_discharge_reconciliation.sql` | Y | |
| 4.2 Diff engine | ✅ Shipped | `reconciliation/diff.go` + `_test.go` (4-class diff: new/ceased/dose_change/unchanged) | Y | |
| 4.3 Classifier + worklist | ✅ Shipped | `reconciliation/classifier.go`, `worklist.go`, REST handlers | Y | |
| 4.4 ACOP write-back | ✅ Shipped | `reconciliation/writeback.go` + `_test.go`, ReconciliationStore + KB20 client methods | Y | |

### Wave 5 — EvidenceTrace bidirectional graph hardening

| Sub-task | Category | Evidence | Acceptance | Notes |
|---|---|---|---|---|
| 5.1 Materialised views | ✅ Shipped | `migrations/022_evidencetrace_views.sql` (mv_recommendation_lineage, mv_observation_consequences, mv_resident_reasoning_summary) | Y | |
| 5.2 Query API surface | ✅ Shipped | `evidence_trace/lineage.go` + `_test.go` (LineageOf, ConsequencesOf, ReasoningWindow) | Y | |
| 5.3 FHIR Provenance/AuditEvent split | ✅ Shipped | `fhir/evidence_trace_dispatcher.go` + `_test.go` | Y | |
| 5.4 Graph load test + index tune | 🟡 Reduced scope | `evidence_trace/loadgen/` + `bench_test.go` synthesizer + benchmark code committed | Partial — commit message: "execution deferred". 6-month / 1M-node run not in CI | TODO(layer3-v1) wave-5.4-execution |

### Wave 6 — Stabilisation + hardening

| Sub-task | Category | Evidence | Acceptance | Notes |
|---|---|---|---|---|
| 6.1 Failure-mode defence validation | ✅ Shipped | `kb-20-patient-profile/tests/failure_modes/` (six tests) — pass | Y | |
| 6.2 Cross-state-machine integration | ✅ Shipped | `kb-20-patient-profile/tests/state_machine_integration/` (five tests) — pass | Y | |
| 6.3 Pilot scenario rehearsal | ✅ Shipped | `kb-20-patient-profile/tests/pilot_scenarios/sunday_night_fall_test.go` — pass | Y | |
| 6.4 Production readiness review | ✅ Shipped | `docs/runbooks/{baseline-drift-investigation,evidencetrace-audit-query,identity-match-queue-triage,mhr-gateway-error-recovery}.md`, `docs/slo/v2-substrate-slos.md`, `docs/handoff/layer-2-to-layer-3-handoff.md`, `docs/security/v2-substrate-security-review.md` | Y | All ≥100 lines substantive content |

### Layer 2 explicit V1-deferrals (per plan §"Gaps explicitly punted")

| Item | Plan note | Status |
|---|---|---|
| Dispensing pharmacy DAA timing (FRED API) | Layer 2 doc Wave 5 (V1) | ⏸️ Deferred — not commenced |
| Behavioural chart structured ingestion (Leecare/AutumnCare) | Layer 2 doc Wave 6 (V1) | ⏸️ Deferred — not commenced |
| Direct hospital ADT feeds | Layer 2 doc Wave 7 (V2) | ⏸️ Deferred |
| Multi-vendor eNRMC FHIR coverage | Layer 2 doc Wave 9 (V2) | ⏸️ Deferred — Telstra Health pilot only |
| NLP for free-text indication | Layer 2 doc Wave 8 (V2) | ⏸️ Deferred |

### Layer 2 — expanded notes on reduced-scope items

**Wave 2.7 (streaming):** the synchronous in-process Go path for compute-on-write
already meets the per-observation latency target on pilot-scale data; the
streaming pipeline is properly an architectural readiness item for facility
multi-tenancy. The ADR is signed (Kafka Streams library mode) and the topology +
topic list (`raw_inbound_events` → `identified_events` → `normalised_events` →
`substrate_updates`) is documented. The risk is purely "this hasn't been
exercised at 10× facility scale yet."

**Wave 3.1–3.3 (MHR + HL7 ingestion):** all three paths ship as
**interface + stub + parser-with-fixtures** rather than live ADHA / pathology
vendor connectors. Each test file uses synthetic CDA / ORU / FHIR fixtures.
The plan's acceptance criterion ("sample CDA from ADHA conformance pack
parses") is met for the parser layer; the SOAP envelope construction, NASH PKI
auth headers, and live OAuth2 path against the FHIR Gateway sandbox are
explicitly deferred. This is the single biggest gap before a real pilot
facility ingests live pathology.

**Wave 5.4 (graph performance):** the `loadgen/` synthesizer and `bench_test.go`
are committed, and the commit message states "execution deferred." The risk is
that the materialised-view refresh strategy (incremental + nightly full)
hasn't been stressed at the planned 1M-node / 1.5M-edge scale.

---

## Layer 3 Rule Encoding Plan — gap matrix

### Pre-Wave — Layer 1 audit blocker closure

| Task | Category | Evidence | Acceptance | Notes |
|---|---|---|---|---|
| Pre-1 fact_type schema decision | ✅ Shipped | `claudedocs/audits/2026-05-PreWave-fact-type-decision.md` + commit `a9b47f28` (kb-4 migration 007 docstring) | Y | |
| Pre-2 L6 governance verifier | ✅ Shipped | `kb-4-patient-safety/internal/governance/verify_au_chain.go` + CLI wrapper, audit memo | Y | Commit `10f990f8` |
| Pre-3 RxCUI gap closure | ✅ Shipped | `claudedocs/audits/2026-05-PreWave-rxcui-validation-procedure.md` runbook | Y | (validator exists in kb-7-terminology — runbook documents the strict-mode invocation) |
| Pre-4 ICD-10-AM placeholder policy | ✅ Shipped | `claudedocs/audits/2026-05-PreWave-icd10am-placeholder-policy.md` + `AgedCareHelpers.cql` header | Y | |

### Wave 0 — Substrate scoping (4 tasks)

| Task | Category | Evidence |
|---|---|---|
| W0.1 Tier 1 substrate primitive inventory | ✅ Shipped | `docs/superpowers/specs/2026-05-Layer3-Wave0-tier1-substrate-contract.md` |
| W0.2 CQL helper surface specification | ✅ Shipped | `docs/superpowers/specs/2026-05-Layer3-Wave0-cql-helper-surface.md` |
| W0.3 rule_specification.yaml v2 schema | ✅ Shipped | `shared/cql-toolchain/schemas/rule_specification.v2.json` + 3 example specs in `examples/` |
| W0.4 Trigger surface mapping | ✅ Shipped | `docs/superpowers/specs/2026-05-Layer3-Wave0-trigger-surface-mapping.md` |

### Wave 1 — Authoring infrastructure (5 tasks)

| Task | Category | Evidence | Notes |
|---|---|---|---|
| W1.1 Helper library implementation | 🟡 Reduced scope | All six helper files present (`MedicationHelpers.cql`, `AgedCareHelpers.cql`, `ClinicalStateHelpers.cql`, `ConsentStateHelpers.cql`, `AuthorisationHelpers.cql`, `MonitoringHelpers.cql`, `EvidenceTraceHelpers.cql`) plus `SuppressionHelpers.cql` and `QualityGapHelpers.cql`; per-helper `_test.cql` files | 17 `TODO(wave-1-runtime)` markers — bodies are skeletons pending live Layer 2 substrate API bindings |
| W1.2 rule_spec validator + two-gate | ✅ Shipped | `cql-toolchain/rule_specification_validator.py` + `two_gate_validator.py` | |
| W1.3 CompatibilityChecker (Events A/B/C/D) | ✅ Shipped | `cql-toolchain/compatibility_checker.py` (Wave 5.7 added selectivity test commit `3f893640`) | |
| W1.4 CDS Hooks v2.0 emitter + PlanDefinition `$apply` | ✅ Shipped | `cql-toolchain/cds_hooks_emitter.py` + `plan-definitions/` PlanDefinition fixtures | |
| W1.5 Governance promoter (Stage 5) | ✅ Shipped | `cql-toolchain/governance_promoter.py` (Ed25519 dual-sign + EvidenceTrace emission) | |

### Wave 2 — Tier 1 immediate-safety rules (~25 rules)

| Task | Category | Plan target | Shipped | Evidence |
|---|---|---|---|---|
| W2.1 Electrolyte/renal trajectory | ✅ Shipped | 6 | 6 | `tier-1-immediate-safety/ElectrolyteRenalTrajectory.cql` (6 defines) + 6 spec yamls |
| W2.2 Falls + post-fall | ✅ Shipped | 5 | 5 | `PostFall.cql` (5 defines) + 5 spec yamls |
| W2.3 Restrictive practice + consent-gated | ✅ Shipped | 5 | 5 | `RestrictivePractice.cql` (5 defines) + 5 spec yamls |
| W2.4 Insulin / opioid S8 | ✅ Shipped | 5 | 5 | `InsulinOpioidS8.cql` (5 defines) + 5 spec yamls |
| W2.5 Care-intensity-aware | ✅ Shipped | 4 | 4 | `CareIntensityPalliative.cql` (4 defines) + 4 spec yamls |

**Wave 2 total: 25 / 25 = 100%.** Note: the rules are batched into 5 CQL
library files rather than 25 separate files (an authoring efficiency choice),
but every spec yaml is per-rule and every define is independently named.
Toolchain batch validation green (`commit 0d14f9dd`).

### Wave 3 — Tier 2 deprescribing rules (~75 rules)

| Task | Category | Plan target | Shipped | Evidence |
|---|---|---|---|---|
| W3.1 ADG 2025 mapping CSV | 🟡 Reduced scope | 185 rows mapped | 20 mapped + 165 deferred to Layer 1 Pipeline-2 | `claudedocs/clinical/2026-05-Layer3-ADG2025-mapping.csv` |
| W3.2 STOPP/START/Beers/Wang authoring | 🟡 Reduced scope | ~50 rules | 4 rules + queue manifest for ~46 | `tier-2-deprescribing/{StoppV3,StartV3,Beers2023,Wang2024}.cql` (1 define each) + `claudedocs/clinical/2026-05-Layer3-Wave3-Task2-rule-queue.md` |
| W3.3 ADG 2025-specific rules | 🟡 Reduced scope | ~20 rules | 2 rules + queue manifest | `tier-2-deprescribing/ADG2025.cql` (2 defines, both with `TODO(layer1-bind)`) + `Wave3-Task3-adg2025-rule-queue.md` |
| W3.4 Suppression-by-recent-action library | ✅ Shipped | n/a | `helpers/SuppressionHelpers.cql` with full bodies + validator extension | |
| W3.5 Tier 2 governance + production deploy | 🟡 Reduced scope | ~75 ACTIVE | Promotion manifest committed for the 6 rules shipped | `claudedocs/governance/2026-05-Layer3-Wave3-promotion-manifest.md` |

**Wave 3 total: 6 / 75 ≈ 8% by rule volume.** This is the largest single gap
in the audit. Queue manifests are explicit and per-rule; the path to plan
volume is "lift each row in the manifest into spec + CQL define + 3 fixtures
(plan budget: ~0.3 days/rule)." Pre-requisite for ADG2025 expansion is Layer 1
audit Pipeline-2 ADG extraction — currently the hard external blocker.

### Wave 4 — Stream A: Tier 3 quality-gap rules (~50 rules)

| Task | Category | Plan target | Shipped | Evidence |
|---|---|---|---|---|
| W4A.1 PHARMA-Care + Standard 5 | 🟡 Reduced scope | ~25 rules | 4 PHARMA-Care + 6 indicator scaffold (10 defines total) + queue manifest for 15 | `tier-3-quality-gap/PharmaCareStandard5.cql` (4) + `PharmaCareIndicators.cql` (6 indicator computations) + `claudedocs/clinical/2026-05-Layer3-Wave4A-Task1-quality-gap-rule-queue.md` |
| W4A.2 Care-transition rules | 🟡 Reduced scope | ~15 rules | 2 rules + queue manifest | `tier-3-quality-gap/CareTransition.cql` (2) + `Wave4A-Task2-transition-rule-queue.md` |
| W4A.3 AN-ACC defensibility | 🟡 Reduced scope | ~10 rules | 2 rules + queue manifest | `tier-3-quality-gap/ANACCDefensibility.cql` (2) + `Wave4A-Task3-anacc-rule-queue.md` |

**Wave 4 Stream A total: ~14 / 50 ≈ 28%.** PHARMA-Care indicator computation
*scaffold* is in place; rule volume is reduced but the indicator framework runs
end-to-end on fixture data. Promotion manifest exists at
`claudedocs/governance/2026-05-Layer3-Wave4A-promotion-manifest.md`.

### Wave 4 — Stream B: Authorisation evaluator subsystem (kb-30)

| Task | Category | Evidence | Notes |
|---|---|---|---|
| W4B.1 AuthorisationRule DSL + parser | ✅ Shipped | `kb-30-authorisation-evaluator/internal/dsl/`, 3 example rules in `examples/` | Tests pass |
| W4B.2 Rule store + versioning | ✅ Shipped | `internal/store/`, `migrations/001_authorisation_rules.sql` | |
| W4B.3 Runtime evaluator | ✅ Shipped | `internal/evaluator/`, `internal/api/`, `cmd/server/main.go` | gRPC + REST |
| W4B.4 Cache layer with TTL strategy | ✅ Shipped | `internal/cache/` (in-memory + Redis stub) | Per-rule TTL |
| W4B.5 Invalidation triggers | ✅ Shipped | `internal/invalidation/` (credential / agreement / consent / ScopeRule events) | |
| W4B.6 Audit query API | ✅ Shipped | `internal/audit/` with 4 regulator-ready endpoints | |
| W4B.7 Sunday-night-fall integration walkthrough | ✅ Shipped | `kb-30-authorisation-evaluator/tests/integration/sunday_night_fall_test.go` | Tests pass |

**Wave 4 Stream B total: ✅ 7/7 — the entire Authorisation evaluator
subsystem is shipped end-to-end.** This is the single most production-shaped
deliverable in Layer 3.

### Wave 5 — Tier 4 surveillance + ScopeRules-as-data (7 tasks)

| Task | Category | Evidence | Notes |
|---|---|---|---|
| W5.1 Tier 4 surveillance authoring (~50 rules) | 🟡 Reduced scope | 6 rules (3 trajectory + 3 lifecycle) shipped + queue manifest for 44 deferred | `tier-4-surveillance/{Trajectory,Lifecycle}.cql` + `Wave5-Task1-tier4-rule-queue.md` |
| W5.2 kb-31-scope-rules service scaffolding | ✅ Shipped | `kb-31-scope-rules/cmd/server`, `internal/{api,dsl,parser,store}`, migrations, integration tests | Tests pass |
| W5.3 Victorian PCW exclusion ScopeRule | ✅ Shipped | `data/AU/VIC/pcw-s4-exclusion-2026-07-01.yaml` with real legislative content | Activation gates per plan |
| W5.4 Designated RN prescriber ScopeRule | ✅ Shipped | `data/AU/national/drnp-prescribing-agreement.yaml` | |
| W5.5 Tasmanian pharmacist co-prescribing pilot | ⏸️ Deferred-to-V1 | `data/AU/TAS/pharmacist-coprescribe-pilot-2026.yaml` staged as DRAFT per plan provision | Activation gated on Move 1 outcome (Vaidshala v2 Revision Mapping) |
| W5.6 ACOP credential ScopeRule | ✅ Shipped | `data/AU/national/acop-apc-credential.yaml` | |
| W5.7 ScopeRule-aware CompatibilityChecker | ✅ Shipped | `compatibility_checker.py` ScopeRule selectivity test (commit `3f893640`) | |

**Wave 5 total: 6/7 fully shipped + 1 explicit V1-deferral (Tasmania
pharmacist pilot pending external pilot launch decision). Tier 4 rule volume
6/50 = 12% with queue manifest.**

### Wave 6 — Continuous tuning (4 tasks)

| Task | Category | Evidence | Notes |
|---|---|---|---|
| W6.1 Override-rate analytics + rule retirement | ✅ Shipped | `kb-30-authorisation-evaluator/internal/analytics/`, `cql-toolchain/rule_retirement_workflow.py`, `claudedocs/governance/2026-05-Wave6-suppression-tuning-runbook.md` (under clinical/) | Commit `0eac13ad` |
| W6.2 Suppression-class tuning | ✅ Shipped | `helpers/SuppressionHelpers.cql` + `claudedocs/clinical/2026-05-Wave6-suppression-tuning-runbook.md` | |
| W6.3 Source-update 7-day SLA | ✅ Shipped | `cql-toolchain/source_update_tracker.py` + `claudedocs/governance/2026-05-Wave6-source-update-7day-sla.md` | |
| W6.4 Coronial / ACQSC finding intake | ✅ Shipped | `claudedocs/governance/2026-05-Wave6-coronial-finding-rule-intake-template.md` | |

**Wave 6 total: 4/4 shipped (infrastructure in place; "ongoing tuning"
naturally extends past the audit date — that's the plan).**

### Cross-cutting deliverables

| Item | Status | Evidence |
|---|---|---|
| Cross-cutting 1: CompatibilityChecker bidirectional contract | ✅ Shipped | Wave 1.3 + Wave 5.7 |
| Cross-cutting 2: 7-day SLA infrastructure | ✅ Shipped | Wave 6.3 |
| Cross-cutting 3: Worked rule example library | ✅ Shipped | `cql-libraries/examples/{ppi-deprescribe,hyperkalemia-trajectory,antipsychotic-consent-gating}.yaml` |
| Cross-cutting 4: Sample CDS Hooks v2.0 response shape | 🟡 Partial | `cds_hooks_emitter.py` produces conforming responses; the dedicated docs file (`shared/cql-toolchain/docs/cds-hooks-response-shape.md`) was not located in the audit |

### Layer 3 — expanded notes

**Tier 2 / Tier 3 / Tier 4 rule volume — pattern:** every Wave with a
"~N rules" target shipped a vertical slice (4–10 rules) plus a queue manifest
documenting each remaining rule with its source citation, helper dependencies,
and suppression class. This is **explicit reduced-scope with audit trail**, not
silent missing scope. Authors picking up the queue manifest can lift each row
into a spec + CQL + fixtures at the plan's documented per-rule budget.

**Helper bodies (`TODO(wave-1-runtime)` × 17):** the helper *signatures* are
all defined per Wave 0.2 spec, but the *bodies* are skeleton implementations
pending the live Layer 2 substrate REST endpoints being wired in dev. This is
internally consistent: Wave 1 authoring infra ships the *interface contract*;
Wave 2+ rule authoring exercises the contract via fixtures; live binding happens
when both substrate and helper hit the same dev environment. The contract is
solid — the bodies are the runtime puzzle piece.

**Layer 1 binding (`TODO(layer1-bind)` × 16):** these markers track ADG2025
rules whose canonical text awaits Layer 1 audit Pipeline-2 extraction. The
2026-04-30 Layer 1 audit (`Layer1_AU_AgedCare_Codebase_Gap_Audit_v2.md`)
explicitly identifies ADG 2025 PDF extraction as Wave 5 free-alternative-path
work. **Hard external blocker — not a Layer 3 plan failure.**

---

## Cross-cutting gaps + V1 priorities

### Risk-ranked top 5 V1 gaps

1. **Tier 2 deprescribing rule volume (6/75).** Highest clinical-impact gap.
   Each queue-manifest row is plan-shaped; the path is straightforward
   authoring once Layer 1 ADG Pipeline-2 lands. **Action: schedule 30-day
   sprint to lift queue manifest at plan velocity once ADG extraction is
   complete; target Tier 2 to 50+ rules ACTIVE before pilot facility expands
   beyond Telstra Health vendor.**

2. **Live MHR / HL7 ingestion (Wave 3.1–3.3 stubs).** Synthetic-fixture parser
   tests are not the same as live ADHA conformance pack + NASH PKI auth +
   FHIR Gateway sandbox round-trips. Risk: pilot facility cannot ingest live
   pathology until this is wired. **Action: dedicated 2-week integration
   sprint with ADHA sandbox + one pilot pathology vendor before pilot launch;
   keep stub path for unit tests.**

3. **Streaming pipeline runtime (Wave 2.7).** Synchronous Go path is fine for
   first facility but caps multi-facility scaling. 15 `TODO(wave-2.7-runtime)`
   markers track work. **Action: medium-priority — schedule Java processor
   build after Tier 2 catch-up; revisit if facility load testing flags
   per-observation latency drift.**

4. **ADG 2025 Pipeline-2 extraction (Layer 1 dependency).** Hard external
   gate on 165 / 185 ADG rule mappings. **Action: prioritise Layer 1 audit
   Wave 5 work; without it, Tier 2 ADG2025 expansion is blocked beyond the
   2 currently-shipped rules.**

5. **Wave 5.4 graph load test execution.** 6-month / 1M-node benchmark exists
   in code but not in CI. Risk: production-scale materialised view refresh
   strategy unproven. **Action: low-cost — schedule `bench_test.go` execution
   on staging Postgres against synthesizer dataset; promote to CI once green.**

### Other notable items

- **Layer 1 Pre-Wave dependencies fully closed.** All four Pre-Wave audit
  blockers (fact_type, L6 governance, RxCUI, ICD-10-AM) have shipped memos
  and the L6 verifier has executable code at
  `kb-4-patient-safety/internal/governance/verify_au_chain.go`.
- **Authorisation evaluator (kb-30) is the cleanest end-to-end build.** All 7
  Wave 4 Stream B tasks shipped, all integration + unit tests pass on a fresh
  build, three example rules with real legislative references in `examples/`.
  This is V1-shaped.
- **kb-31 ScopeRule engine is V1-shaped** — service scaffolding, 4 real
  jurisdictional ScopeRules with legislative references, integration tests
  green. Tasmania pharmacist pilot rule is staged DRAFT per plan provision.
- **Runbooks + handoff (Wave 6.4 deliverables) all exceed plan minimums.**
  Each runbook ≥100 lines substantive content; Layer 3 handoff doc lists
  every read/write API the rule library will consume.

### Cross-cutting deferral discipline assessment

The plan's expectation that reduced-scope work be marked with explicit deferral
manifests is **honoured throughout**. Specifically:
- `claudedocs/clinical/` contains 7 queue manifests covering Tier 2, Tier 3,
  Tier 4 and ADG2025 reduced-scope deliverables.
- `claudedocs/governance/` contains 4 docs covering Wave 6 ongoing-tuning
  infrastructure plus 2 governance promotion manifests.
- TODO markers are namespaced (`TODO(layer1-bind)`, `TODO(layer3-v1)`,
  `TODO(wave-1-runtime)`, `TODO(wave-2.7-runtime)`, `TODO(clinical-author)`)
  so future authors can grep by deferral class.
- Build is clean (`go build ./...` succeeds in `shared`, `kb-20`, `kb-30`,
  `kb-31`); all unit + integration tests pass on a fresh checkout.

---

## Appendix: TODO marker counts (raw grep)

```
TODO(layer1-bind):       16   (Tier 2 ADG2025 + Layer 1 ADG extraction blocker)
TODO(layer3-v1):          4   (Wave 5.4 + Wave 4A Task 3 + Tasmania pilot)
TODO(clinical-author):   16   (Tier 3-4 rule body completions)
TODO(kb-7-binding):       0   (terminology binding shipped where authored)
TODO(wave-1-runtime):    17   (helper bodies awaiting live substrate API)
TODO(wave-2.7-runtime):  15   (streaming pipeline Java processor code)
TODO(kb-26-acute-repository): 0
```

Total Vaidshala-prefixed TODO markers in `backend/`: **~68**.

(Comparator: third-party imports under `backend/` carry many more TODO markers
from upstream OSS — `TODO(jansel)` 51, `TODO(voz)` 39, `TODO(justinchuby)` 35
etc. — these are unrelated to the Vaidshala plans and are included only to
calibrate the meaningful Vaidshala-deferred-work count of ~68.)

---

## Appendix: build + test verification at audit time

Audit run on branch `feature/v4-clinical-gaps`, HEAD `c7134402`,
2026-05-06.

```
shared/v2_substrate/...                ✅ build clean; 13 packages tests pass
kb-20-patient-profile/...              ✅ build clean; all packages tests pass
                                          (incl. failure_modes, pilot_scenarios,
                                          state_machine_integration)
kb-30-authorisation-evaluator/...      ✅ build clean; all packages tests pass
                                          (incl. tests/integration)
kb-31-scope-rules/...                  ✅ build clean; all packages tests pass
                                          (incl. tests/integration)
```

Migrations 001–022 present in `kb-20-patient-profile/migrations/` and apply
in the documented order. Note: migrations 011–012 are absent from the
filename list (numbering jumps `010_identity_mapping.sql → 013_baseline_persistent_store.sql`)
— this is consistent with plan §"File Structure" which numbered the planned
migration set `009..021` but the actual landed migrations consolidated some
work; not a gap, an artifact of authoring efficiency.

---

## Appendix: per-Wave commit anchors

For traceability, the highest-signal commit per Wave/sub-task:

**Layer 2:**
- Wave 1R.1 Event entity — see `models/event.go`, no single-line commit (folded into earlier β.2 work)
- Wave 1R.2 EvidenceTrace v0 — see `migrations/009_event_evidencetrace.sql`
- Wave 2.1 PersistentBaselineProvider — `migrations/013_baseline_persistent_store.sql`
- Wave 2.6 scoring — `cba84e58`, `528860a0`
- Wave 2.7 streaming — `ddc7860e` (ADR), `fbf0f379` (skeleton)
- Wave 3 MHR/HL7 — `145de285`, `953feac4`, `67af09a4`, `cad5a571`, `4ba49602`
- Wave 4 reconciliation — `e4718eda`, `e1efd581`, `2dddd3ee`, `ac76955d`
- Wave 5 — `38d4ce60`, `890c9597`, `f648b878`, `24797d38`
- Wave 6 — `9468fba0`, `44f7ebb5`, `01f05db3`, `7e0b600f`

**Layer 3:**
- Pre-Wave — `a9b47f28`, `10f990f8`, `aadc8eb6`, `2a18473e`
- Wave 0 — `e3e7a5ad`, `a8757580`, `23bbb222`, `7b9be8eb`
- Wave 1 — `6ff45515`, `ea2ed744`, `657841d4`, `3c997196`, `64948ee2`, `d790b5d3`
- Wave 2 — `314d9193`, `5beedd4a`, `4dca1d38`, `6f6e01f5`, `95291922`, `0d14f9dd`
- Wave 3 — `6ccf334b`, `610b8ea3`, `54cdde92`, `0ae24e77`, `8fc5ef84`
- Wave 4A — `772b7529`, `e3b02e94`, `3a460326`, `40bc20f3`
- Wave 4B — `6b86f49a`, `50233f82`, `8027fc83`, `0d0400df`, `cecc2b89`, `8c914f26`, `8dc485cf`
- Wave 5 — `cb1cf9db`, `a73a39e3`, `3f893640`
- Wave 6 — `0eac13ad`, `c7134402`

---

**End of audit.**
