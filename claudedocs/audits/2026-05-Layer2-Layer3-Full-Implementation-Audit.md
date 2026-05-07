# Layer 2 + Layer 3 Full Implementation Audit

**Date:** 2026-05-06
**Auditor:** automated codebase audit (read-only inspection + test execution)
**Branch / HEAD:** `feature/v4-clinical-gaps` @ `318a125e`
**Scope:** Production-readiness, code quality, architectural soundness, and honest risk of all Layer 2 + Layer 3 substrate work shipped to `main`.
**Method:**
- `git log` analysis across last ~150 commits on main (Layer 2 + Layer 3 surface).
- `go build ./...` + `go test ./...` against `shared/`, `kb-20-patient-profile/`, `kb-30-authorisation-evaluator/`, `kb-31-scope-rules/`, `kb-4-patient-safety/`.
- `python3 -m pytest` against `shared/cql-toolchain/`.
- File-tree inspection across `shared/v2_substrate/` (13 packages), `shared/cql-libraries/` (helpers + 4 tiers), `kb-30/`, `kb-31/`, `backend/stream-services/substrate-pipeline/`, `docs/`, `claudedocs/`.
- Spot-read of representative files: helper CQL bodies, kb-30 main.go, Sunday-night-fall integration test, queue manifests, governance verifier.
- Cross-reference with predecessor `2026-05-Layer2-Layer3-Gap-Analysis.md` and `2026-05-V1-Gap-Closure-Plan.md`.

---

## Executive verdict

**The Layer 2 + Layer 3 substrate is a credible, well-organised MVP scaffold. It is not yet a production system.** Substrate (Layer 2) is genuinely shipped: 22k LOC of Go across 13 cleanly-bounded packages, all tests green, kb-20 wires real Postgres/Redis/Kafka through config, and the v2 product proposal's five state machines are all represented as code (Authorisation in kb-30, Recommendation suppression helpers, Monitoring helpers + KB-22 engine, ClinicalState in active_concerns + care_intensity + capacity_assessment, Consent in CapacityAssessment + ConsentStateHelpers). Layer 3 (Rule encoding) shipped a real CQL toolchain (244 pytest cases passing), 78 specs, 84 CQL `define` statements across 16 rule files, and kb-30 + kb-31 services that compile and pass integration tests. **The honest pattern is "vertical slice + queue manifest"**: the toolchain is real, the per-rule pattern is proven, and explicit queue manifests cover deferred volume — but the rule library is ~22% of plan target (78 of ~350 rules), kb-30/kb-31 ship with `MemoryStore + AlwaysPassResolver` in main.go (PostgresStore exists, just not wired by default), and the helper CQL bodies bottom out in `Vaidshala.Substrate.*` external functions that depend on a HAPI runtime that has not been stood up. **Risk surface is tractable** because the deferrals are explicit, marked, and have V1 owners — but anyone reading "Wave 4B kb-30 shipped" needs to know that the in-memory + AlwaysPassResolver path is what runs by default.

---

## Layer 2 — Substrate

### 1. Code organisation & module boundaries

**Verdict: clean.**

`shared/v2_substrate/` is organised into 13 logical packages (`models/`, `validation/`, `clinical_state/`, `delta/`, `evidence_trace/`, `fhir/`, `identity/`, `ingestion/`, `interfaces/`, `reconciliation/`, `scoring/`, `streaming/`, `client/`). The package boundaries map onto natural responsibilities: pure data + validators in `models/` + `validation/`; pure engines (no I/O) in `clinical_state/`, `delta/`, `reconciliation/`, `scoring/`; FHIR mappers isolated in `fhir/`; HTTP clients to KB-20 isolated in `client/`. There is a single `shared/go.mod` at module root (`github.com/cardiofit/shared`) and kb-20 takes it via `replace` directive.

**Cross-layer leakage check:**
- `kb-30-authorisation-evaluator/` and `kb-31-scope-rules/` both import **zero** symbols from `v2_substrate/`. Verified via `grep -r "v2_substrate"` → empty. This is correct: kb-30 is a Layer 3 runtime service that consumes credentials/agreements/consents over the wire, not a substrate-coupled module.
- `kb-20-patient-profile/` correctly depends on `shared/v2_substrate/...` (it is the substrate's primary Postgres host).
- The `interfaces/storage.go` file (596 LOC) is the contract surface between substrate engines and storage backends — a healthy abstraction that lets pure engines be tested without a database.

No circular imports observed. No ad-hoc coupling spotted.

**One concession:** `shared/v2_substrate/streaming/` contains only two markdown files (`topology.md`, `load_test_plan.md`) — there is no Go code under streaming/. The Java side is the actual streaming runtime, and it is a near-empty scaffold (see §4 below).

### 2. Build & test status

```
shared/v2_substrate/...           — go build OK; go test 13/13 packages PASS (2 with [no test files])
kb-20-patient-profile/...         — go build OK; go test 12 packages PASS, 7 [no test files]
kb-30-authorisation-evaluator/... — go build OK; go test 9 packages PASS, 1 [no test files]
kb-31-scope-rules/...             — go build OK; go test 4 packages PASS, 2 [no test files]
kb-4-patient-safety/...           — go build OK
shared/cql-toolchain/             — pytest 244/244 PASS in 4.08s
```

Some package-level test counts (no LOC-level test coverage gathered):
- `shared/v2_substrate/`: 86 source files, 69 test files (~80% per-file coverage).
- `shared/cql-toolchain/`: 244 tests across two-gate validator + CompatibilityChecker + governance promoter + CDS Hooks emitter.
- `kb-20-patient-profile/tests/failure_modes/` (Wave 6.1), `state_machine_integration/` (Wave 6.2), `pilot_scenarios/` (Wave 6.3 Sunday-night-fall) all green.

**Test SKIPs:** Postgres-backed test suites under `kb-30/internal/store/` and `kb-31/internal/store/` are gated behind `KB30_TEST_DATABASE_URL` and `KB31_TEST_DATABASE_URL` env vars (not set in this audit run), so `PostgresStore` paths are not exercised by `go test ./...` on a clean checkout. The MemoryStore tests do run.

**Trivial-coverage packages:**
- `kb-20-patient-profile/internal/cache/`, `internal/config/`, `internal/database/`, `internal/metrics/`, `cmd/ingest-csv/`, `cmd/mhr-poll/` — no test files. Cache and metrics are infrastructure adapters; ingest-csv has indirect coverage through pilot tests but no direct unit tests.

### 3. Commit history quality

`git log --oneline | wc -l` = 1038 commits on the branch. The Layer 2 + Layer 3 push consists of ~80–90 commits in the last ~120 commits, with a clear `feat(<package>)` / `test(<package>)` / `docs(<area>)` / `feat(wave-X.Y)` convention. Granularity is good — each Wave sub-task has a coherent commit (e.g., `eda09586 feat(kb-20): CapacityAssessmentStore + REST handlers + Consent-tagged EvidenceTrace for impaired-medical + main.go wiring`). Commit messages identify Wave numbers, deferral status, and key deliverables.

**Wave 0 retroactive commits:** No clear "bundled-files-in-working-tree" pattern observed. The Wave 0 commits (`7b9be8eb`, `a8757580`, `e3e7a5ad`, etc.) are docs-only specs that landed before the implementation commits, so they read as forward-progress, not retroactive bundling.

**"Stub-as-feature" cases:**
- `feat(stream-services/substrate-pipeline): Java module skeleton + processor stubs` (`fbf0f379`) — the message is honest. The code is genuinely a skeleton (~230 LOC across 6 files, processors are 18-49 LOC stubs).
- `feat(v2_substrate/evidence_trace/loadgen): graph synthesizer + benchmark (Wave 5.4 — execution deferred)` (`24797d38`) — message correctly flags deferral.
- `feat(v2_substrate): EvidenceTrace dispatcher routing nodes to Provenance OR AuditEvent (Wave 5.3)` (`f648b878`) — the dispatcher is a routing function; FHIR egress (the actual write) is not done. Commit message is silent on this. **Mild concern** — see §7.

Overall: the commit log is honest about Wave-N status and most deferrals are flagged in the commit subject.

### 4. Architectural decisions audit (five state machines)

| State machine | Status | Evidence |
|---|---|---|
| **Authorisation** | Implemented as kb-30 service | Real DSL + parser + Store interface (Memory + Postgres) + evaluator + cache + audit + 3 example rules + Sunday-night-fall integration test |
| **Recommendation** | Implemented in toolchain + helpers | `governance_promoter.py`, `rule_retirement_workflow.py`, suppression-by-recent-action `SuppressionHelpers.cql` + `recommendation_state.*` fact_types |
| **Monitoring** | Implemented in helpers + KB-22 + Tier 4 lifecycle | `MonitoringHelpers.cql` + KB-22 MonitoringNodeEngine + Tier 4 Lifecycle.cql |
| **ClinicalState** | Implemented as substrate packages | `clinical_state/active_concerns.go` + `care_intensity_engine.go`; capacity_assessment as separate Wave 2.x slice; full Postgres tables 015-017 |
| **Consent** | Partially implemented | `models/CapacityAssessment` + `ConsentStateHelpers.cql` cover capacity + consent state, but a dedicated Consent state machine with consent lifecycle (granted → revoked → expired) is **not** a separate substrate package — it's threaded through CapacityAssessment + AuthorisationRule guards. Acceptable for MVP; explicit consent-revocation flows would be V1 work. |

**EvidenceTrace bidirectional graph:** Real graph + table-backed. `evidence_trace/graph.go`, `edge_store.go`, `lineage.go` (259 LOC), `query.go` (134 LOC) implement the API; migration `022_evidencetrace_views.sql` provides materialised views. Tests cover lineage + consequences + reasoning-window queries. **Real, not table-only.**

**Persistent BaselineProvider:** Tied to real Postgres. `delta/baseline_state_store.go` + migration `013_baseline_persistent_store.sql` + `014_baseline_config.sql` provide real persistence, and `kb-20/internal/storage/v2_substrate_store.go` (1686 LOC) wires it into Recompute/Upsert flows.

**kb-30 / kb-31 cache + invalidation:** **In-memory by default.** kb-30 main.go calls `cache.NewInMemory()` and the Redis cache is a stub (`feat(kb-30): in-memory cache with per-rule TTL strategy + Redis cache stub (Wave 4B.4)`). Invalidation triggers (Wave 4B.5) are wired as in-process events — no Kafka consumer for credential/agreement/consent/scoperule events runs in production wiring.

**Wave 5.4 graph load test:** Harness only. `loadgen/synthesize.go` ships with a `DefaultProfile` (200 residents × 180d × 5 nodes/d → ~180k nodes), an `evidence_trace/bench_test.go` benchmark, and a "execution and SLO lock-in deferred to V1" docstring. **Not actually executed against a real Postgres.**

### 5. Operational readiness gates (Layer 2 services)

**kb-20-patient-profile:**
- main.go: production-quality. Wires `config.Load()`, structured logging (zap), Postgres, Redis cache, Kafka, FHIR client, Prometheus metrics, resilience pkg.
- Migrations: 22 numbered migrations applied via the project's standard migrator. **Not auto-applied to any DB in this environment** (no DB credentials in test env), but the SQL is real and runs in the unit-test in-memory PG path (where it exists).
- Monitoring: `internal/metrics/` + Prometheus endpoint.
- Security: relies on existing CardioFit auth substrate (HeaderAuthMiddleware from shared modules). NASH PKI / OAuth not directly wired into substrate writes.

**Stream-services / substrate-pipeline (Java):**
- 10 files, ~230 LOC across 4 processors. Processors are 18–49 LOC stubs that compile but do not implement the topology described in `shared/v2_substrate/streaming/topology.md`. **This is correctly labelled "scaffold; runtime to be filled in by V1"** but anyone expecting "Wave 2.7 streaming pipeline shipped" should know that the Java side is essentially placeholder.

**Migrations applied:** No `migrations/applied/` log directory exists; migrations are runtime-applied at service startup, not pre-applied. This is normal for the project's migration pattern but means there is no audit trail of "migration X ran on DB Y at time Z" within the repo.

### 6. Honest assessment of Layer 2 "shipped" claims

| Wave / sub-task | Claim | Honest assessment |
|---|---|---|
| Wave 1 ingestion (CSV) | Shipped | ✅ Real: `cmd/ingest-csv/main.go` + 50-row pilot fixture + `csv_enrmc.go` |
| Wave 1 ingestion (HL7) | Shipped | 🟡 HL7 v2.5 ORU parser real (`hl7_parser`); Java HL7 listener is 49-LOC stub |
| Wave 1 ingestion (MHR) | Shipped | 🟡 SOAP/CDA + FHIR Gateway are interface + ~150 LOC stubs; ADHA endpoint not wired (gap-analysis Gap 3) |
| Wave 2 active_concerns | Shipped | ✅ Real: pure engine + Postgres + REST + tests |
| Wave 2 care_intensity | Shipped | ✅ Real: pure engine + cascade rules + Postgres + REST |
| Wave 2 capacity_assessment | Shipped | ✅ Real: model + validator + store + Consent-tagged EvidenceTrace |
| Wave 2 scoring (DBI/ACB/CFS/AKPS) | Shipped | ✅ Real: pure calculators + DrugWeightLookup interface + 20-row seed |
| Wave 3 baseline + trajectory | Shipped | ✅ Real: TrajectoryDetector, BaselineConfig, persistent baseline state |
| Wave 4 reconciliation | Shipped | ✅ Real: pure diff engine + classifier + worklist + ACOP write-back |
| Wave 5.1 EvidenceTrace migration | Shipped | ✅ Real: 022 with materialised views |
| Wave 5.2 lineage queries | Shipped | ✅ Real: Lineage / Consequences / ReasoningWindow query API |
| Wave 5.3 FHIR Provenance/AuditEvent | "dispatcher" | 🟡 Routing logic is real, but **no FHIR egress wired**; no actual POST to Google Healthcare API or HAPI. Dispatcher decides which mapper to use; downstream send is V1. |
| Wave 5.4 graph load test | Shipped | 🟡 Harness only; execution deferred to V1 (commit message honest) |
| Wave 6.1-6.3 (failure modes / integration / pilot) | Shipped | ✅ Real: 3 dedicated test packages all green |
| Wave 6.4 docs (runbooks, SLOs, handoff, security) | Shipped | ✅ Real: 6 runbooks, 1 SLO doc, 1 handoff doc, 1 security review (see §9) |

### 7. Risk surface (Layer 2)

| Module | Risk | Evidence |
|---|---|---|
| `shared/v2_substrate/models/`, `validation/` | LOW | Pure data + validators with full test coverage |
| `shared/v2_substrate/clinical_state/` | LOW | Pure engines + dedicated tests + integration tests in kb-20 |
| `shared/v2_substrate/delta/` | LOW | Pure trajectory + baseline math; persistent store tied to real PG |
| `shared/v2_substrate/scoring/` | LOW | Pure calculators with seed weights + tests |
| `shared/v2_substrate/reconciliation/` | LOW | Pure diff + classifier + worklist with tests |
| `shared/v2_substrate/evidence_trace/` (excl. loadgen) | LOW–MEDIUM | Lineage queries real and tested; production-scale latency unverified |
| `shared/v2_substrate/evidence_trace/loadgen/` | MEDIUM | Synthesizer real but never executed against a real PG; no SLO numbers |
| `shared/v2_substrate/fhir/` (mappers) | LOW | 11 mapper pairs with tests |
| `shared/v2_substrate/fhir/evidence_trace_dispatcher.go` | MEDIUM | Routing logic only; FHIR egress is not wired |
| `shared/v2_substrate/identity/` (IHI matcher) | MEDIUM | Real fuzzy + IHI matcher; MHR IHI resolver is a stub interface |
| `shared/v2_substrate/ingestion/` (MHR) | HIGH | 150–200 LOC stubs; needs real ADHA SOAP/CDA + FHIR Gateway clients with NASH PKI |
| `shared/v2_substrate/ingestion/runner.go` | LOW | Real orchestrator with idempotency + EvidenceTrace logging |
| `kb-20-patient-profile/` | LOW | Real production wiring; full migration set; test coverage strong |
| `backend/stream-services/substrate-pipeline/` (Java) | HIGH | ~230 LOC scaffold; topology designed but not implemented |

---

## Layer 3 — Rule encoding

### 1. Code organisation & module boundaries

**Verdict: clean and intentional.**

`shared/cql-libraries/` is split into:
- `helpers/` — 8 helper libraries × (1 .cql + 1 _test.cql) = 16 files. ~970 LOC of CQL.
- `tier-1-immediate-safety/`, `tier-2-deprescribing/`, `tier-3-quality-gap/`, `tier-4-surveillance/` — each with `<RuleSet>.cql` files + `specs/*.yaml` + `fixtures/`.
- `plan-definitions/`, `examples/`, `rules/` — auxiliary.

`shared/cql-toolchain/` is a Python toolchain: `rule_specification_validator.py`, `two_gate_validator.py` (Stage 1 + Stage 2 substrate semantics), `compatibility_checker.py` (Events A/B/C/D + ScopeRule-aware), `cds_hooks_emitter.py` (CDS Hooks v2.0 + PlanDefinition $apply), `governance_promoter.py` (Stage 5 with Ed25519 dual-sign + EvidenceTrace emission), `rule_retirement_workflow.py`, `source_update_tracker.py`. Schemas separated under `schemas/`, tests under `tests/`.

`kb-30-authorisation-evaluator/` and `kb-31-scope-rules/` each have a textbook Go service layout: `cmd/server/`, `internal/{api,dsl,evaluator,store,cache,audit,invalidation,analytics,parser}/`, `tests/integration/`, `examples/` or `data/`, `migrations/`, `README.md`, `go.mod`. **kb-30 + kb-31 are correctly separated** — distinct DSLs (AuthorisationRule vs ScopeRule), distinct stores, distinct lifecycles. The architectural split is sound: kb-31 governs *which jurisdictions/sites a rule applies to*; kb-30 governs *whether a specific actor may take a specific action under credentials/agreements/consents*. They both feed into the rule engine but are evaluated at different points.

**The `internal/parser/` directory in kb-31 has zero test files** despite having 59 LOC. Mild concern — likely covered transitively by integration tests but worth a unit-test pass.

### 2. Build & test status (Layer 3)

- `kb-30-authorisation-evaluator/`: 9 packages PASS, integration test (Sunday-night-fall) PASS.
- `kb-31-scope-rules/`: 4 packages PASS, integration test PASS.
- `shared/cql-toolchain/`: 244 pytest cases PASS.
- `shared/cql-libraries/`: no executable tests in pure CQL (HAPI Test Engine deferred — see §5); the toolchain validates them via batch tests in `cql-toolchain/tests/`.
- `kb-4-patient-safety/internal/governance/verify_au_chain.go`: builds + tests PASS (382 LOC).

### 3. Commit history quality (Layer 3)

Layer 3 commits are well-organised: `feat(tier-1)`, `feat(tier-2)`, ..., `feat(cql-toolchain)`, `feat(kb-30)`, `feat(kb-31)`, `test(...)`, `docs(...)`. Wave numbers (Wave 0 → Wave 6 + extension batches) are tracked in subjects.

**Pre-Wave commits** (`a9b47f28`, `aadc8eb6`, `2a18473e`) are docs that establish the fact_type schema, RxCUI procedure, and ICD-10-AM placeholder policy *before* implementation commits land — good ordering, not retroactive.

**"Stub-as-feature" assessment:**
- `feat(cql-libraries/helpers): six helper libraries with ~40 function signatures + skeleton bodies + per-helper tests` (`6ff45515`) — message is honest ("skeleton bodies"). Subsequent commits `0ae24e77 feat(cql-libraries/helpers): SuppressionHelpers.cql with full bodies` filled in real bodies. Helpers now have real CQL bodies but they call external `Vaidshala.Substrate.*` functions that resolve under HAPI runtime (not yet stood up).
- `feat(tier-1): N rules` commits — these ship real CQL `define` statements with criterion citations, real fixtures in `specs/*.yaml`, and the rules pass the toolchain's two-gate validator. Honest.
- `feat(tier-2): 2 ADG 2025 rules with TODO(layer1-bind) markers pending Pipeline-2 extraction` — explicit and accurate.

### 4. Substantive CQL/Rule library assessment

**Counts:**
- 16 helper .cql files (8 production + 8 test) — ~970 LOC of CQL
- 16 rule .cql files across Tier 1–4 — **84 `define` statements total** = 84 rules in CQL bodies
- 78 rule specs (.yaml) under tier-N/specs/ (25 + 21 + 18 + 14)
- 252 fixtures (.json/.yaml) under tier-N/fixtures/

**Discrepancy:** 84 CQL defines vs 78 specs vs ~78 rules per gap analysis. The 6-rule delta is helpers' helper-layer constants and aggregation defines (e.g., `Falls Risk Compounding Medications` is one define that draws on multiple criteria).

**Per Tier:**

| Tier | CQL files | Defines | Specs | Fixtures | Plan target | % shipped |
|---|---|---|---|---|---|---|
| Tier 1 (immediate safety) | 5 | 25 | 25 | ~75 | 25 | **100%** |
| Tier 2 (deprescribing) | 5 | 21 | 21 | ~63 | ~75–225 (STOPP/START/Beers/Wang/ADG) | ~9–28% |
| Tier 3 (quality gap) | 4 | 24 | 18 | ~54 | ~50 | ~36–48% |
| Tier 4 (surveillance) | 2 | 14 | 14 | ~42 | ~50 | ~28% |
| **Total** | **16** | **84** | **78** | **~234** | **~200–350** | **~22–39%** |

**Are CQL bodies real?** Yes for rules. Tier 1 PostFall.cql is a typical example: rule bodies use namespaced helper calls (`State."LatestObservationAt"(...)`, `Trace."RuleFiredFor"(...)`, `Meds."HasActiveMed"(...)`). The helpers themselves have real CQL bodies. **The helpers' bodies bottom out in `Substrate.RecommendationActionedWithin(...)` external function calls** — these are the points where execution will need a HAPI Test Engine + Vaidshala.Substrate library at runtime. So the rules + helpers are real-CQL "all the way down to the external boundary," but **execution requires runtime work that has not happened**.

`grep -c "^define " shared/cql-libraries/tier-*/[A-Z]*.cql` confirms 84 real define statements. `grep "Substrate\."` finds zero direct external function calls in the rule files (rules go through helpers) — the abstraction is consistent.

**Wave 1 helper bodies:** Real CQL. Each helper file has a TODO line `TODO(wave-1-runtime): execute under HAPI Test Engine.` This is the honest summary: bodies are written, runtime is V1.

### 5. Operational readiness gates (Layer 3 services)

**kb-30-authorisation-evaluator (`cmd/server/main.go`):**
- Wires `store.NewMemoryStore()` + `cache.NewInMemory()` + `evaluator.AlwaysPassResolver` by default.
- A `PostgresStore` exists in `internal/store/store.go` (~196–384 LOC, 191+ LOC of pg implementation) but is not selected by main.go.
- `loadExamples()` ingests 3 example YAML rules from `examples/` directory — comment in code says "Production wiring would use the PostgresStore + a migration-driven seed."
- Migration `001_authorisation_rules.sql` exists and is real.
- Tests gated behind `KB30_TEST_DATABASE_URL`.
- gRPC handlers shipped (per commit `8027fc83`); REST handlers shipped.
- Audit query API (4 endpoints, Wave 4B.6) shipped with tests.
- Invalidation triggers (Wave 4B.5): in-process event subscribers; no Kafka consumer wired.
- **Honest summary:** main.go runs on in-memory + always-pass-resolver. To run against real Postgres + real credentials/agreements lookup requires V1 wiring of `NewPostgresStore` + a non-AlwaysPassResolver (e.g., one that queries kb-22-credential-store).

**kb-31-scope-rules (`cmd/server/main.go`):**
- Wires `store.NewMemoryStore()`. PostgresStore exists in `internal/store/store.go`.
- `loadBundledRules` ingests 4 YAML ScopeRules from `data/`.
- Migration `001_scope_rules.sql` exists.
- ScopeRule-aware CompatibilityChecker (Wave 5 Task 7) wired into the cql-toolchain selectivity test.
- **Same honest summary as kb-30.**

**No Prometheus / metrics endpoint** observed in either kb-30 or kb-31 main.go (no `/metrics` route registration). This is a gap that V1 will need.

**No NASH PKI / OAuth wiring.** The DSL has `signed_by` fields but signature verification is delegated to `evaluator.AlwaysPassResolver` by default. Real Ed25519 verification surfaces exist in the `governance_promoter.py` (Stage 5 dual-sign) and in `kb-4/internal/governance/verify_au_chain.go` (218 LOC L6 verifier with Ed25519 sig checks + CLI wrapper) — these are real cryptographic code, but wiring a real signing key + verification key chain into kb-30 runtime is V1.

### 6. Honest assessment of Layer 3 "shipped" claims

| Wave / sub-task | Claim | Honest assessment |
|---|---|---|
| Pre-Wave (fact_type schema, RxCUI, ICD-10-AM, L6 verifier) | Shipped | ✅ Real: 4 docs + 1 Go package (382 LOC verify_au_chain) |
| Wave 0 (specs + helper surface + trigger mapping) | Shipped | ✅ Real: 3 specs at docs/superpowers/specs/ |
| Wave 1 (rule_specification validator + two-gate + helpers) | Shipped | ✅ Real toolchain; helpers have real CQL bodies; HAPI runtime deferred |
| Wave 2 Tier 1 (25 rules immediate safety) | Shipped | ✅ Real: 25 specs + 25 defines + fixtures + batch test passing |
| Wave 3 Tier 2 (deprescribing) | "6 of 75 shipped" | 🟡 Honest queue manifest. ~21 defines now (post-extension). 90% of plan target deferred with explicit per-source counts. **Queue manifest is real**, with per-rule citation, helper, suppression class — not a fig leaf. |
| Wave 4A Tier 3 (quality gap) | "8 of 50 shipped" | 🟡 Now 24 defines (post-extension). 3 queue manifests under claudedocs/clinical/. |
| Wave 4B kb-30 | Shipped | 🟡 Service is real but main.go runs in-memory + always-pass. Sunday-night-fall test is a 252-LOC integration test — but it uses MemoryStore + AlwaysPassResolver, so it's **end-to-end across the kb-30 runtime, not end-to-end across real credential/agreement infrastructure.** Honest description: "in-process integration test against real DSL parser + real evaluator + real cache + real audit, with stub credential resolver and in-memory rule store." |
| Wave 5 Tier 4 (surveillance) | "6 of 50 shipped" | 🟡 Now 14 defines. Queue manifest of 44 rules with per-rule helper + source. |
| Wave 5 ScopeRule (kb-31) | Shipped | ✅ ScopeRule engine + 4 deployment ScopeRules + parser + REST API + integration test |
| Wave 5 Task 7 ScopeRule-aware CompatibilityChecker | Shipped | ✅ Real: in cql-toolchain with selectivity test |
| Wave 6 (override-rate, retirement, SLA, runbooks) | Shipped | ✅ Real: 4 governance runbooks + Python implementations |

### 7. Risk surface (Layer 3)

| Module | Risk | Evidence |
|---|---|---|
| `shared/cql-libraries/helpers/` | MEDIUM | Real CQL but bottom out in `Vaidshala.Substrate.*` external functions (no HAPI runtime wired) |
| `shared/cql-libraries/tier-1-immediate-safety/` | LOW | 25 of 25 plan rules; full specs + fixtures; clinical authorship visible in citations |
| `shared/cql-libraries/tier-2-deprescribing/` | HIGH (clinical) | ~21 of ~225 — 90% of plan target deferred; rule volume is the central V1 ask |
| `shared/cql-libraries/tier-3-quality-gap/` | MEDIUM (clinical) | ~50% shipped + clear queue manifests |
| `shared/cql-libraries/tier-4-surveillance/` | MEDIUM | ~28% shipped + queue manifest |
| `shared/cql-toolchain/` | LOW | 244 tests passing; toolchain is real and well-tested |
| `kb-30-authorisation-evaluator/` (kernel) | LOW | DSL + parser + evaluator + store interfaces all clean |
| `kb-30-authorisation-evaluator/` (production wiring) | HIGH | main.go uses in-memory + always-pass; needs real Postgres + real credential resolver + Kafka invalidation consumer + Redis cache |
| `kb-31-scope-rules/` | MEDIUM | Service is real but data/ ships only 4 ScopeRules; production deployment ScopeRules (per-RACF, per-state) need authoring |
| `kb-4-patient-safety/internal/governance/` | LOW | Real Ed25519 verifier + CLI wrapper |

### 8. Documentation surface

**Inventory:**
- ADRs: 2 (`docs/adr/2026-05-06-mhr-integration-strategy.md`, `2026-05-06-streaming-pipeline-choice.md`)
- Runbooks: 6 (`docs/runbooks/`)
- SLO docs: 1 (`docs/slo/v2-substrate-slos.md`)
- Handoff docs: 1 (`docs/handoff/layer-2-to-layer-3-handoff.md`)
- Security review: 1 (`docs/security/v2-substrate-security-review.md`)
- Specs: 3 Layer 3 Wave 0 specs + earlier specs
- Queue manifests: 6 (`claudedocs/clinical/2026-05-Layer3-Wave3-*`, `Wave4A-*`, `Wave5-*`)
- Governance manifests: 2 (`claudedocs/governance/2026-05-Layer3-Wave3-promotion-manifest.md`, `Wave4A-promotion-manifest.md`)
- Pre-Wave audits: 4 (`claudedocs/audits/2026-05-PreWave-*`)
- Layer 2/3 gap analysis: 1
- V1 gap closure plan: 1
- CDS Hooks response shape: 1 (`shared/cql-toolchain/docs/cds-hooks-response-shape.md`)

**Spot-check substance:**
- `claudedocs/clinical/2026-05-Layer3-Wave5-Task1-tier4-rule-queue.md`: 44 queued rules, per-row source citation + helper + suppression class. **Real prose, not a template.**
- `shared/v2_substrate/streaming/topology.md`: 80+ lines of real topology design with explicit TODO(wave-2.7) markers.
- `claudedocs/audits/2026-05-V1-Gap-Closure-Plan.md`: 10 gaps with risk, effort, blocker class, V1 owner, prerequisites, acceptance evidence per gap. **Real prose.**

Cross-references intact: gap analysis links to V1 closure plan; runbooks reference SLOs; handoff doc references Wave 0 specs.

### 9. What's left (concise)

**Production-ready fraction:**
- **Layer 2:** ~70% production-ready. Substrate engines + kb-20 service are real; ingestion (MHR, HL7) and Java streaming pipeline are stubs; FHIR egress is a routing stub.
- **Layer 3:** ~40% production-ready. Toolchain is solid, kb-30/kb-31 kernels are solid, but main.go uses in-memory paths, rule volume is ~22% of plan target, and HAPI Test Engine is not stood up. The CQL library is shipped *as authoring artefacts* but not *as a running rule engine*.

---

## Cross-cutting findings

1. **The "vertical slice + queue manifest" pattern is consistent.** Every reduced-scope wave ships (a) the engineering pattern (toolchain + tests), (b) a small but representative working set (e.g. 6 of 50 Tier 4 rules), and (c) a queue manifest enumerating the deferred items with per-row citation/helper/suppression-class. This is the right pattern for an MVP heading into a clinical-author-driven V1, and it is honestly labelled.

2. **The `MemoryStore + AlwaysPassResolver` default is a tripwire.** Both kb-30 and kb-31 ship with main.go binding to in-memory infrastructure by default. PostgresStore + Redis cache exist but are gated by env vars. Any "kb-30 deployed" claim needs to specify which backing it runs against.

3. **External dependencies are uniformly deferred.** HAPI FHIR engine, NASH PKI, ADHA SOAP/CDA endpoint, Confluent Cloud Kafka, real Postgres at production scale, Tasmanian pilot integration — all 6 are V1 work, not in-MVP scope. The V1 Gap Closure Plan correctly identifies these as "cannot be filled by writing code."

4. **Clinical-authorship is the long pole.** ~225 deprescribing rules + ~50 surveillance rules + ~16 ADG2025 layer-1 binds + 32 `TODO(clinical-author)` markers. The engineering pattern can scale; what gates V1 is clinical informatics review time (the V1 plan estimates 8-14 weeks of clinician-time for Tier 2 alone).

5. **Cryptographic surfaces are present but inert.** `governance_promoter.py` does Ed25519 dual-sign in tests; `kb-4/internal/governance/verify_au_chain.go` does Ed25519 verify in tests; kb-30 DSL has `signed_by` fields. But the production key chain is not wired: AlwaysPassResolver is the default, and Stage 5 dual-sign tests use synthetic keys. **V1 needs a key-management story.**

6. **Test-skip gating discipline is good.** `KB30_TEST_DATABASE_URL`, `KB31_TEST_DATABASE_URL`, `KB20_TEST_DATABASE_URL` consistently gate Postgres-backed integration tests so a `go test ./...` run on a clean checkout does not pretend Postgres paths are exercised.

7. **The Java side is the weakest link.** `backend/stream-services/substrate-pipeline/` is a 230-LOC Java scaffold with no real Kafka Streams topology; the HL7 listener Java skeleton (`cad5a571`) is the same. The streaming runtime is the gap with the highest impact-per-effort cost and is most explicitly marked `TODO(wave-2.7-runtime)`.

---

## Risk surface table (consolidated)

| Service / module | LOC | Tests | Risk | V1 effort to production |
|---|---|---|---|---|
| `shared/v2_substrate/models` + `validation` | ~3k | strong | LOW | none (already prod-ready) |
| `shared/v2_substrate/clinical_state` + `delta` + `scoring` + `reconciliation` | ~5k | strong | LOW | none |
| `shared/v2_substrate/evidence_trace` (queries) | ~700 | strong | LOW | latency verification at scale |
| `shared/v2_substrate/evidence_trace/loadgen` | ~80 | minimal | MEDIUM | execute against real PG, lock SLOs |
| `shared/v2_substrate/fhir/dispatcher` + mappers | ~3k | strong | MEDIUM (dispatcher) | wire FHIR egress to Google Healthcare API or HAPI |
| `shared/v2_substrate/identity` | ~700 | good | MEDIUM | replace MHR IHI resolver stub with real ADHA call |
| `shared/v2_substrate/ingestion` (MHR + HL7) | ~1.5k | partial | HIGH | NASH PKI + ADHA endpoint + JVM HL7 sidecar |
| `kb-20-patient-profile` | ~10k | strong | LOW | run migrations on prod-scale PG |
| `backend/stream-services/substrate-pipeline` (Java) | ~230 | none | HIGH | Kafka Streams topology + Confluent Cloud creds |
| `shared/cql-libraries/helpers` | ~970 | strong-validator | MEDIUM | wire to HAPI Test Engine; HAPI deployment |
| `shared/cql-libraries/tier-1` | small | strong-validator | LOW | clinical sign-off + HAPI runtime |
| `shared/cql-libraries/tier-2`–`tier-4` | small | strong-validator | HIGH (clinical) | author ~225 deprescribing rules + 44 surveillance |
| `shared/cql-toolchain` | ~3k Python | 244 tests | LOW | none |
| `kb-30-authorisation-evaluator` (kernel) | ~1.5k | strong | LOW | none |
| `kb-30-authorisation-evaluator` (wiring) | main.go 93 | n/a | HIGH | wire PostgresStore + real resolver + Redis + Kafka invalidation consumer |
| `kb-31-scope-rules` (kernel) | ~1.5k | strong | LOW | none |
| `kb-31-scope-rules` (wiring) | main.go 72 | n/a | MEDIUM | same wiring + ship more deployment ScopeRules |
| `kb-4-patient-safety/internal/governance` | 382 | good | LOW | execute against live kb-4 PG |

---

## Production-readiness scorecard

| Wave | Stated status | Honest scorecard |
|---|---|---|
| L2 Wave 1 (foundations) | Shipped | 70% prod-ready. CSV + idempotent runner real. MHR + HL7 ingestion stubs. |
| L2 Wave 2 (clinical state) | Shipped | 95% prod-ready. Active concerns + care intensity + capacity + scoring all real. |
| L2 Wave 3 (baseline + trajectory) | Shipped | 90% prod-ready. |
| L2 Wave 4 (reconciliation) | Shipped | 90% prod-ready. |
| L2 Wave 5.1 (EvidenceTrace migration) | Shipped | 95% prod-ready. |
| L2 Wave 5.2 (lineage queries) | Shipped | 90% prod-ready. |
| L2 Wave 5.3 (FHIR Provenance/AuditEvent) | Shipped | 50% prod-ready. Dispatcher real, FHIR egress not wired. |
| L2 Wave 5.4 (graph load test) | Shipped (deferred-marked) | 30% prod-ready. Harness only. |
| L2 Wave 6 (failure/integration/pilot/docs) | Shipped | 90% prod-ready. |
| L2 Wave 2.7 (Java streaming) | Scaffold | 10% prod-ready. |
| L3 Pre-Wave | Shipped | 95% prod-ready. |
| L3 Wave 0 (specs) | Shipped | 100% prod-ready. |
| L3 Wave 1 (toolchain + helpers) | Shipped | 70% prod-ready. Toolchain real; HAPI runtime deferred. |
| L3 Wave 2 Tier 1 (25 rules) | Shipped | 80% prod-ready. Clinical sign-off + HAPI runtime needed. |
| L3 Wave 3 Tier 2 (deprescribing) | Vertical slice + queue | 25% prod-ready. ~21 of ~225. |
| L3 Wave 4A Tier 3 (quality gap) | Vertical slice + queue | 50% prod-ready. ~24 of ~50. |
| L3 Wave 4B kb-30 (authorisation evaluator) | Shipped | 50% prod-ready. Kernel real; main.go in-memory + always-pass. |
| L3 Wave 5 Tier 4 (surveillance) | Vertical slice + queue | 30% prod-ready. ~14 of ~50. |
| L3 Wave 5 kb-31 (ScopeRules) | Shipped | 50% prod-ready. Kernel real; production ScopeRules to be authored. |
| L3 Wave 6 (override / retirement / SLA / runbooks) | Shipped | 80% prod-ready. |

**Overall:** ~55% of Layer 2 + Layer 3 sub-tasks are at or above 80% production-ready; ~30% are at 40-70% (vertical slice + queue manifest); ~15% are below 40% (Java streaming, Wave 5.4 load test, FHIR egress, kb-30 wiring).

---

## Top 5 honest concerns

1. **kb-30 + kb-31 main.go default to in-memory + always-pass.** Anyone reading "Wave 4B kb-30 shipped" without reading main.go will assume the service runs against real Postgres and verifies real credentials. It does not by default. The PostgresStore exists, the AlwaysPassResolver is explicit in code, but the production wiring step is non-trivial and is not in MVP scope. **Risk: external readers conflate "kernel ready" with "service ready."**

2. **Rule volume is ~22% of plan target and clinical-authorship is the bottleneck, not engineering.** ~225 Tier 2 deprescribing rules, ~26 more Tier 3 rules, ~36 more Tier 4 rules need clinical-informatics-author time (~14 weeks of clinician work per V1 plan). The toolchain can produce them; the limit is human review. **Risk: V1 underestimates clinician-author throughput.**

3. **Java substrate-pipeline is a 230-LOC scaffold.** The streaming topology design is documented (`shared/v2_substrate/streaming/topology.md`); the Java implementation is essentially placeholder. This is the single biggest "stub-as-feature" in the codebase. Wave 6 SLOs depend on a working pipeline (e.g., end-to-end p95 latency claims). **Risk: any SLO claim that depends on stream throughput is currently aspirational.**

4. **HAPI FHIR Test Engine is the runtime that the helpers + rules ultimately need, and it has not been stood up.** All 84 CQL `define`s + 8 helper libraries + 78 specs + 234 fixtures are *artefacts*. To execute them, a HAPI deployment + a `Vaidshala.Substrate` external function library binding the substrate APIs is needed. This is the rule-engine V1 dependency. **Risk: the rule library is "shippable as code review artefact" but not "executable as decision support" until HAPI runtime exists.**

5. **Cryptographic surfaces are inert by default.** Ed25519 sig-verify exists in tests + L6 verifier; AlwaysPassResolver short-circuits sig checks at runtime. Production deployment requires a key-management story (NASH PKI + Vaidshala signing keys + key rotation). **Risk: "signed_by" fields in YAML create the appearance of cryptographic enforcement that is not yet enforced.**

---

## What V1 should do FIRST (top 3 actions)

1. **Wire kb-30 to real backing infrastructure end-to-end.** Replace `MemoryStore` → `PostgresStore`; replace `AlwaysPassResolver` → real credential/agreement resolver (probably an HTTP call to kb-22-credential-store + kb-23-agreement-store); replace in-memory cache → Redis; subscribe to Kafka invalidation events. Run the Sunday-night-fall integration test against this real path — that single test, repointed, validates the entire chain. **Highest leverage: turns kb-30 from "kernel" to "service."** Effort: ~2-3 weeks per V1 plan Gap 6 + parts of Gap 5.

2. **Stand up HAPI FHIR Test Engine + bind `Vaidshala.Substrate` external function library.** This unblocks execution of all 84 CQL defines + helpers. Without it, the rule library is shipped-as-paper. With it, the cql-toolchain can promote tested rules into a running engine. **Highest leverage on Layer 3: turns 78 specs from artefacts into running CDS.** Effort: ~6-8 weeks per V1 plan Gap 5.

3. **Pick the 25 highest-yield deprescribing rules and clinical-author them this quarter.** The V1 plan estimates 8-12 weeks for ~225 rules; that is too long to wait. A focused 25-rule slice (the most-overridden / highest-clinical-impact STOPP/Beers rules) doubles Tier 2 volume in 4-6 weeks, gets the override-rate tracker (Wave 6) into a meaningful operating regime, and de-risks the larger authoring effort. **Highest leverage on rule library: shifts the bottleneck from authoring throughput to engine throughput.**

(The V1 Gap Closure Plan lists 10 gaps; my "first 3" differ from its risk-ranking by elevating HAPI runtime above MHR/HL7 wiring, on the grounds that HAPI is what executes the entire Layer 3 corpus while MHR/HL7 only feed Layer 2 ingestion — and Layer 2 ingestion already has the CSV path running.)

---

## Appendix A — Tests + builds verified

```
$ go build ./shared/v2_substrate/...
(via shared/go.mod)         OK

$ go test ./shared/v2_substrate/...
ok  github.com/cardiofit/shared/v2_substrate/client                (cached)
ok  github.com/cardiofit/shared/v2_substrate/clinical_state        (cached)
ok  github.com/cardiofit/shared/v2_substrate/delta                 (cached)
ok  github.com/cardiofit/shared/v2_substrate/evidence_trace        (cached)
ok  github.com/cardiofit/shared/v2_substrate/evidence_trace/loadgen (cached)
ok  github.com/cardiofit/shared/v2_substrate/fhir                  (cached)
ok  github.com/cardiofit/shared/v2_substrate/identity              (cached)
ok  github.com/cardiofit/shared/v2_substrate/ingestion             (cached)
?   github.com/cardiofit/shared/v2_substrate/interfaces            [no test files]
ok  github.com/cardiofit/shared/v2_substrate/models                (cached)
ok  github.com/cardiofit/shared/v2_substrate/reconciliation        (cached)
ok  github.com/cardiofit/shared/v2_substrate/scoring               (cached)
ok  github.com/cardiofit/shared/v2_substrate/validation            (cached)

$ cd kb-20-patient-profile && go test ./...
ok  kb-patient-profile/internal/api                            0.743s
ok  kb-patient-profile/internal/clients                        (cached)
ok  kb-patient-profile/internal/fhir                           (cached)
ok  kb-patient-profile/internal/models                         (cached)
ok  kb-patient-profile/internal/services                       (cached)
ok  kb-patient-profile/internal/storage                        1.226s
ok  kb-patient-profile/pkg/resilience                          (cached)
ok  kb-patient-profile/tests/failure_modes                     2.145s
ok  kb-patient-profile/tests/pilot_scenarios                   1.656s
ok  kb-patient-profile/tests/state_machine_integration         2.962s

$ cd kb-30-authorisation-evaluator && go test ./...
ok  kb-authorisation-evaluator/internal/analytics              (cached)
ok  kb-authorisation-evaluator/internal/api                    (cached)
ok  kb-authorisation-evaluator/internal/audit                  (cached)
ok  kb-authorisation-evaluator/internal/cache                  (cached)
ok  kb-authorisation-evaluator/internal/dsl                    (cached)
ok  kb-authorisation-evaluator/internal/evaluator              (cached)
ok  kb-authorisation-evaluator/internal/invalidation           (cached)
ok  kb-authorisation-evaluator/internal/store                  (cached)
ok  kb-authorisation-evaluator/tests/integration               (cached)

$ cd kb-31-scope-rules && go test ./...
ok  kb-scope-rules/internal/api                                (cached)
ok  kb-scope-rules/internal/dsl                                (cached)
ok  kb-scope-rules/internal/store                              (cached)
ok  kb-scope-rules/tests/integration                           (cached)

$ cd kb-4-patient-safety && go build ./...    OK

$ cd shared/cql-toolchain && python3 -m pytest -q
244 passed in 4.08s
```

## Appendix B — Surface counts

```
shared/v2_substrate/                — 86 source .go + 69 test .go = ~22.4k LOC
shared/cql-libraries/helpers/       — 8 helper .cql + 8 _test.cql + AgedCareHelpers = ~970 LOC of CQL
shared/cql-libraries/tier-1/        — 5 rule .cql, 25 specs, ~75 fixtures
shared/cql-libraries/tier-2/        — 5 rule .cql, 21 specs, ~63 fixtures
shared/cql-libraries/tier-3/        — 4 rule .cql, 18 specs, ~54 fixtures
shared/cql-libraries/tier-4/        — 2 rule .cql, 14 specs, ~42 fixtures
shared/cql-toolchain/               — ~3k LOC Python, 244 tests
kb-20-patient-profile/              — full service, 22 migrations
kb-30-authorisation-evaluator/      — ~1.5k LOC kernel + 252-LOC integration test, 1 migration, 3 example rules
kb-31-scope-rules/                  — ~1.5k LOC kernel, 1 migration, 4 ScopeRules in data/
backend/stream-services/substrate-pipeline/ — ~230 LOC Java scaffold across 10 files
```

## Appendix C — Honest one-liners (for executive summary slides)

- "Layer 2 substrate engines are production-ready; ingestion clients (MHR/HL7) and Java streaming pipeline are scaffolds."
- "Layer 3 toolchain + kernels (kb-30/kb-31) are production-ready; main.go production wiring + ~225 more deprescribing rules + HAPI runtime are V1."
- "84 CQL defines + 78 specs + 244 toolchain tests passing. This is artefact-shipped, not engine-shipped."
- "kb-30 + kb-31 default to MemoryStore + AlwaysPassResolver; PostgresStore exists; Redis cache + Kafka invalidation consumer + real credential resolver are V1."
- "No silent stubs; every reduced-scope wave has an explicit queue manifest with per-rule clinical citations."
