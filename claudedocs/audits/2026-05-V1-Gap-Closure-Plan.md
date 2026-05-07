# V1 Gap Closure Plan — Layer 2 + Layer 3 Substrate

**Date:** 2026-05-06
**Predecessor audit:** [2026-05-Layer2-Layer3-Gap-Analysis.md](./2026-05-Layer2-Layer3-Gap-Analysis.md)
**Scope:** Operational handoff for gaps that require external dependencies (credentials, infrastructure, clinical author input, real-data extraction). MVP-shippable code gaps were closed in commits `e748eef4..91975699` (33 additional rules + cds-hooks-response-shape.md).

---

## Why this document exists

The post-MVP gap analysis identified ~26% of Layer 2 + ~46% of Layer 3 sub-tasks as either reduced-scope or V1-deferred. A subsequent rule-volume expansion (commits `e748eef4..91975699`) shipped 33 more rules + closed the cds-hooks-response-shape.md hidden gap, raising shipped-rule totals from 45 to 78 across Tiers 1-4.

The remaining gaps cannot be filled by writing code. They require one of:

1. **External credentials** that no engineering subagent can produce (NASH PKI, ADHA endpoint access, Confluent Cloud credentials, real Postgres at production scale)
2. **Clinical author review** of every rule against pilot facility data
3. **Real document extraction** (Layer 1 Pipeline-2 ADG 2025 PDF processing)
4. **Java runtime work** (HAPI FHIR engine, Kafka Streams sidecar) that requires a JVM toolchain we have not stood up
5. **Commercial/regulatory gates** (Tasmanian pilot integration, ACQSC partnership)

This document specifies, for each remaining gap, **exactly what V1 needs to do**, who owns it, what the prerequisites are, and what acceptance evidence proves it's done.

---

## Gap closure priority (risk-ranked)

| # | Gap | Risk | Effort | Blocker class | V1 owner |
|---|---|---|---|---|---|
| 1 | Tier 2 deprescribing rule volume (78 → 200+ rules) | High clinical | 8-12 weeks | Clinical author | Clinical informatics + V1 author team |
| 2 | ADG 2025 Pipeline-2 extraction → bind 16 `TODO(layer1-bind)` markers | High clinical | 4 weeks | Layer 1 Pipeline-2 | Layer 1 ingestion team |
| 3 | MHR / HL7 production wiring (Wave 3.1-3.3) | High operational | 6-8 weeks | NASH PKI + ADHA endpoint | Integration + DevOps |
| 4 | Streaming pipeline runtime (Wave 2.7) | Medium operational | 4-6 weeks | Java + Confluent Cloud | Stream-services team |
| 5 | HAPI FHIR engine integration (Wave 1 Task 1) | High clinical | 6-8 weeks | Java runtime + HAPI deployment | Engineering + Clinical |
| 6 | Real Redis + Kafka wiring in kb-30 + kb-31 | Medium operational | 2-3 weeks | Production infra | DevOps |
| 7 | Production Postgres load test (Wave 5.4) | Medium operational | 1-2 weeks | Production-scale Postgres | DevOps + Performance |
| 8 | Clinical-author review of 32 `TODO(clinical-author)` markers | High clinical | 2-3 weeks | Clinical lead time | Clinical informatics |
| 9 | L6 governance verifier execution against live kb-4 PG | Low operational | 1 day | KB-4 PG credentials | Ops |
| 10 | Tasmanian pilot ScopeRule activation | Low commercial | 1 day post-gate | Vaidshala v2 Move 1 outcome | Commercial |

---

## Gap 1 — Tier 2 deprescribing rule volume

### Current state (post-expansion)

| Source | Published total | Shipped | Queued |
|---|---|---|---|
| STOPP v3 | 80 | 5 (B1, D5, F2, K1, J6) | 75 |
| START v3 | 40 | 5 (A1, B5, D2, E1, F4) | 35 |
| Beers 2023 | 57 | 5 (Table 2 anti-AH, K.1, K.7, H, G) | 52 |
| Wang 2024 AU-PIMs | 19 | 4 (§3, §1, §7, §11) | 15 |
| ADG 2025 (custom) | ~50 | 2 (PPI step-down, antipsychotic 12w-review) | ~48 |
| **Total** | **~246** | **21** | **~225** |

### V1 closure plan

**Prerequisites:**
- Clinical informatics lead time (~0.3 days/rule × 225 = ~68 days = ~14 weeks)
- Pilot facility data access for fixture creation (anonymised RACF cases)
- Per-rule acceptance gate: clinical author signoff + ≥10 real-RACF-case validation + override-rate prediction <70%

**Sequencing:**
- **Phase 1 (Weeks 1-4):** Author the remaining 75 STOPP v3 rules + 35 START v3 rules. Use the established CQL/spec/fixture pattern. Each rule batch of 10 promotes through governance together.
- **Phase 2 (Weeks 5-8):** Author the remaining 52 Beers 2023 + 15 Wang 2024 AU-PIMs rules.
- **Phase 3 (Weeks 9-12):** Author the ADG 2025-derived deprescribing rules (depends on Gap 2 — ADG Pipeline-2 extraction).

**Acceptance evidence:**
- 200+ rules ACTIVE in CompatibilityChecker
- KB-4 governance log shows dual-signature for each
- Pilot facility deprescribing-implementation rates show measurable lift toward Wave 3 targets (PPI >50%, cessation overall >55%, dose-reduction >55%)

**Estimated V1 effort:** 12 weeks engineering + ongoing clinical review

---

## Gap 2 — ADG 2025 Pipeline-2 extraction

### Current state

- 16 `TODO(layer1-bind)` markers across Wave 3 + Wave 4 tier-2/tier-3/tier-4 rules referencing ADG 2025 recommendations
- ADG 2025 PDFs landed in Layer 1 (per audit `Layer1_AU_AgedCare_Codebase_Gap_Audit_v2_2026-04-30.md` Wave 5 row) but NOT YET PROCESSED
- 4 ADG 2025 themed rules ship with placeholder `criterion_id` values (e.g. `ADG2025-PPI-001-PLACEHOLDER`); each will need its real `criterion_id` swapped in once Pipeline-2 outputs land

### V1 closure plan

**Prerequisites:**
- Layer 1 Pipeline-2 ingestion pipeline operational (this is Layer 1 plan work, not Layer 2/3)
- UWA licensing review for ADG 2025 commercial CDS use complete (per `claudedocs/audits/2026-05-PreWave-rxcui-validation-procedure.md`)

**Sequencing:**
- **Step 1:** Layer 1 team runs Pipeline-2 against ADG 2025 PDFs. Output: structured recommendation dataset with stable IDs.
- **Step 2:** Map placeholder IDs to real recommendation IDs. Run `grep -rn "TODO(layer1-bind)" backend/` to enumerate all 16 markers; replace each with the real ID.
- **Step 3:** Re-run governance promotion against the 4 ADG 2025 rules (re-sign with new content_sha).
- **Step 4:** Author the remaining ~48 ADG 2025 deprescribing rules per Phase 3 of Gap 1 above.

**Acceptance evidence:**
- `grep -r "TODO(layer1-bind)" backend/` returns 0 matches
- All 4 ADG 2025 rules' `criterion_id` field references a real, citable ADG 2025 recommendation ID
- KB-7 terminology binding confirms the recommendation IDs match the published ADG 2025 source

**Estimated V1 effort:** 4 weeks (mostly Layer 1 work + 1 week of Layer 3 rebinding)

---

## Gap 3 — MHR / HL7 production wiring (Wave 3.1–3.3)

### Current state

- `shared/v2_substrate/ingestion/mhr_soap_cda.go` — interface shipped; stub implementation returns `errors.New("mhr_soap_cda: production wiring deferred to V1")`
- `shared/v2_substrate/ingestion/mhr_fhir_gateway.go` — same pattern
- `shared/v2_substrate/ingestion/hl7_oru.go` — parser works against synthetic fixture; per-vendor adapter table is a stub
- `kb-20-patient-profile/cmd/mhr-poll/main.go` — CLI scaffold; calls stubbed client and panics with deferred-V1 message

### V1 closure plan

**Prerequisites:**
- ADHA registration as a B2B Gateway data sender (organisational onboarding; legal team)
- NASH PKI certificate procurement + secure storage (per `docs/security/v2-substrate-security-review.md` rotation policy)
- ADHA conformance test pack access (CDA + FHIR Gateway samples)
- 2-3 pilot pathology vendor integration agreements (HL7 endpoint URLs + auth)

**Sequencing per Wave 3.1:**
1. Wire `MHRSOAPClient` against ADHA B2B Gateway endpoint with NASH PKI mTLS
2. Replace synthetic CDA fixture with ADHA conformance pack samples
3. Run `cmd/mhr-poll` CLI against ADHA test environment for 1 facility
4. Idempotency verification: re-poll same documents produces 0 duplicates

**Sequencing per Wave 3.2:**
1. Wire `MHRFHIRClient` against ADHA FHIR Gateway IG v1.4.0 with OAuth2 + NASH
2. Add `dual` mode toggle to mhr-poll CLI (run both SOAP/CDA + FHIR Gateway concurrently for migration)
3. Verify both paths produce identical `ParsedObservation` lists for same source documents

**Sequencing per Wave 3.3:**
1. Per-vendor adapter authoring (start with the largest pilot pathology vendor)
2. MLLP listener Java module brought up under `backend/stream-services/substrate-pipeline/hl7_listener/`
3. Identity match at MLLP boundary; observation insert triggers baseline recompute

**Acceptance evidence:**
- 1 pilot resident's pathology results land in substrate within 4 hours of MHR upload
- Identity match for IHI-keyed records returns HIGH confidence
- Per-pathology-vendor HL7 fallback produces the same Observation shape as MHR path

**Estimated V1 effort:** 6-8 weeks (depends on ADHA onboarding latency)

---

## Gap 4 — Streaming pipeline runtime (Wave 2.7)

### Current state

- `backend/stream-services/substrate-pipeline/` — Java module skeleton
  - `pom.xml` declares Kafka Streams 3.7.0 + Confluent client deps
  - `SubstrateStreamApp.java` + 3 processor stubs with `// TODO(wave-2.7-runtime)` body markers (15 markers total)
  - `Dockerfile` + `application.properties` ship
- `docs/adr/2026-05-XX-streaming-pipeline-choice.md` — ADR proposed (Kafka Streams library mode chosen)
- `shared/v2_substrate/streaming/{topology.md,load_test_plan.md}` — doc-complete

### V1 closure plan

**Prerequisites:**
- Confluent Cloud Kafka cluster credentials + topic provisioning quota
- 4 Avro schemas for the substrate topics (raw_inbound_events, identified_events, normalised_events, substrate_updates) — schema registry strategy pending per ADR open question
- Java 17 build pipeline running on CI

**Sequencing (per `topology.md` TODO list):**
1. Provision the 4 Confluent Cloud topics (12 partitions, RF=3, snappy, min.insync.replicas=2)
2. Author concrete Avro schemas per topic
3. Implement IdentityMatchingProcessor (calls kb-20 `/v2/identity/match` over HTTP with retry/circuit-breaker)
4. Implement NormalisationProcessor (calls kb-7-terminology with cache)
5. Implement SubstrateWriterProcessor (calls kb-20 REST; preserves Go transactional ownership)
6. Wire Prometheus metrics per processor
7. Run load test per `load_test_plan.md`: 2,000 obs/day × 10 facilities, 24h sustained; target p95 e2e <5s

**Acceptance evidence:**
- All 15 `TODO(wave-2.7-runtime)` markers removed
- Load test S1 (steady-state) passes acceptance bars
- Substrate-pipeline service deployed alongside existing Stage 1 enricher in dev

**Estimated V1 effort:** 4-6 weeks

---

## Gap 5 — HAPI FHIR engine integration (Wave 1 Task 1 + CDS Hooks runtime)

### Current state

- 17 `TODO(wave-1-runtime)` markers across the 6 helper libraries
- All helper bodies reference `Vaidshala.Substrate.*` external functions; the substrate adapter library does not yet exist
- CDS Hooks emitter produces JSON; `cmd/server` for a runtime CDS Hooks service is not built
- `cds-hooks-response-shape.md` doc shipped (closes the docs gap; runtime gap remains)

### V1 closure plan

**Prerequisites:**
- HAPI FHIR Engine deployment (Java) OR Smile CDR commercial license
- Substrate adapter library (`Vaidshala.Substrate`) — Java module that mediates between HAPI's CQL evaluator and the Layer 2 substrate REST APIs (kb-20 + kb-30 + kb-31)
- CDS Hooks v2.0 service mounted on a public-facing endpoint with prescriber EHR integration

**Sequencing:**
1. Stand up HAPI FHIR Engine in dev (Docker)
2. Author the `Vaidshala.Substrate` adapter library — translates each helper's external function call into a kb-20 / kb-30 REST call, with caching
3. Run all 44 helper functions through HAPI's CQL Test Engine using the existing `_test.cql` files
4. Build CDS Hooks v2.0 service (Java or Go HTTP wrapper around HAPI) on port TBD
5. Wire to first pilot prescriber EHR (Best Practice / Medical Director Australia)

**Acceptance evidence:**
- All 17 `TODO(wave-1-runtime)` markers removed (or moved to less-blocking categories)
- Anchor rule end-to-end: PPI deprescribing rule fires under HAPI against fixture data, emits valid CDS Hooks v2.0 card, prescriber EHR receives + displays
- Latency: rule + Authorisation evaluator round-trip <500ms p95 against fixture data

**Estimated V1 effort:** 6-8 weeks (largest single integration in the V1 plan)

---

## Gap 6 — Real Redis + Kafka wiring in kb-30 + kb-31

### Current state

- `kb-30-authorisation-evaluator/internal/cache/cache.go` — `RedisCache` is a stub with `// TODO(layer3-v1): wire go-redis client`. `InMemoryCache` is fully wired and used by all tests + integration walkthrough + `cmd/server`.
- `kb-30-authorisation-evaluator/internal/invalidation/invalidator.go` — Kafka consumer skeleton; `// TODO(layer3-v1): wire confluent-kafka-go consumer`. `InvalidateOnEvent` method is fully implemented and tested with synthetic events.
- 4 `TODO(layer3-v1)` markers total

### V1 closure plan

**Prerequisites:**
- Production Redis instance (Confluent Cloud or AWS ElastiCache)
- Kafka consumer credentials for the `substrate_updates` topic (depends on Gap 4)

**Sequencing:**
1. Add `go-redis/redis/v9` dependency to kb-30 + kb-31 go.mod
2. Implement `RedisCache` against the in-memory `Cache` interface contract
3. Add `confluent-kafka-go` consumer to `internal/invalidation/`
4. Wire production credentials via environment vars
5. Synthetic credential expiry test: emit Kafka event, verify cache invalidation within 1s

**Acceptance evidence:**
- All 4 `TODO(layer3-v1)` markers removed
- Cache hit-rate >95% on simulated production load
- Cache invalidation latency p95 <1s after substrate change event

**Estimated V1 effort:** 2-3 weeks

---

## Gap 7 — Production Postgres load test (Wave 5.4)

### Current state

- `shared/v2_substrate/evidence_trace/loadgen/synthesize.go` — synthetic graph generator (180K nodes, 500K edges); shipped + tested
- `shared/v2_substrate/evidence_trace/bench_test.go` — Go benchmarks for forward/backward depth=5 traversal
- `shared/v2_substrate/evidence_trace/loadgen/README.md` — invocation instructions
- Production execution: deferred

### V1 closure plan

**Prerequisites:**
- Production-scale Postgres instance (managed; AWS RDS or similar)
- 6 months of synthetic activity data loaded (or pilot facility data)

**Sequencing:**
1. Run loadgen against the production Postgres
2. Execute pgbench-driven workload mixing recursive-CTE forward + backward traversals
3. Tune indexes per the actual EXPLAIN output
4. Re-execute Wave 5.4 acceptance bar: forward depth=5 p95 <200ms; backward depth=5 p95 <200ms
5. Materialised view refresh: <60s incremental, <10min full

**Acceptance evidence:**
- Wave 5.4 SLO targets met against 6-month-of-activity dataset
- Index tuning recommendations checked into `migrations/023_evidencetrace_index_tune.sql`

**Estimated V1 effort:** 1-2 weeks

---

## Gap 8 — Clinical-author review of `TODO(clinical-author)` markers

### Current state

- 32 `TODO(clinical-author)` markers across:
  - PHARMA-Care 5-domain framework citations (waiting on UniSA Sluggett v1 framework PDF)
  - AN-ACC defensibility rule clinical thresholds
  - 4 Tier 3 PHARMA-Care `criterion_id` values
  - Tier 4 surveillance threshold tunings (e.g. eGFR decline %, weight loss %)

### V1 closure plan

**Prerequisites:**
- Clinical informatics lead engagement (recurring, ~0.5 day/week)
- Access to UniSA-published PHARMA-Care v1 framework PDF (procurement)
- Pilot facility outcome data for threshold tuning

**Sequencing:**
1. Schedule weekly clinical-author review sessions (1h/week × 8 weeks)
2. Walk through each `TODO(clinical-author)` marker in priority order (Tier 1 first)
3. Either replace marker with real value OR mark as "intentionally permissive — clinical-team approved"
4. Re-promote affected rules through governance with new content_sha

**Acceptance evidence:**
- `grep -r "TODO(clinical-author)" backend/` returns 0 matches OR every remaining marker is accompanied by a `# clinical-team-approved-permissive` comment with date + reviewer
- KB-4 governance log shows re-signing for affected rules

**Estimated V1 effort:** 2-3 weeks elapsed (clinical bandwidth-bound)

---

## Gap 9 — L6 governance verifier execution against live kb-4 PG

### Current state

- `kb-4-patient-safety/internal/governance/verify_au_chain.go` — Go verifier with Ed25519 signature checks; 8 unit tests pass
- `kb-4-patient-safety/cmd/verify-au-chain/main.go` — CLI wrapper
- `claudedocs/audits/2026-05-PreWave-l6-governance-verification.md` — runbook documenting invocation

Live execution against the production kb-4 PG: not run (deferred per Pre-Wave Task 2 dispatch instruction).

### V1 closure plan

**Prerequisites:**
- KB-4 PG database credentials
- Platform Ed25519 signing public key

**Sequencing:**
1. Set `KB4_DATABASE_URL` env to live kb-4 PG
2. Set `KB4_SIGNING_PUBKEY_PATH` to platform pubkey file
3. Run `verify-au-chain` against all 8 criterion sets
4. Resolve any signature failures (re-sign affected rules through governance)

**Acceptance evidence:**
- `verify-au-chain --strict` exits 0 against all 8 criterion sets
- Audit log entry recorded with verification timestamp + verifier identity

**Estimated V1 effort:** 1 day (assuming no signature failures surface; +1 week per failure batch)

---

## Gap 10 — Tasmanian pilot ScopeRule activation

### Current state

- `kb-31-scope-rules/data/AU/TAS/pharmacist-coprescribe-pilot-2026.yaml` ships with `status: DRAFT` + `activation_gate: "Pilot integration confirmation pending Vaidshala v2 Move 1 outcome"`
- 5 layered enforcement points prevent DRAFT from surfacing as ACTIVE
- All other 3 ScopeRules (Vic PCW, NMBA DRNP, ACOP APC) are ACTIVE-ready

### V1 closure plan

**Prerequisites:**
- Vaidshala v2 Revision Mapping Move 1 outcome confirmed: Vaidshala selected as digital substrate for the Tasmanian pharmacist co-prescribing pilot
- Tasmanian Department of Health partnership agreement signed
- Pilot facility identified and onboarded

**Sequencing:**
1. Receive partnership confirmation
2. Edit `pharmacist-coprescribe-pilot-2026.yaml`: change `status: DRAFT` → `status: ACTIVE`
3. Remove `activation_gate` field (or mark resolved)
4. Re-deploy through kb-31's promotion pipeline
5. Synthetic test: Tasmanian pharmacist co-prescribing query against the rule returns granted

**Acceptance evidence:**
- ScopeRule status flipped to ACTIVE
- Authorisation evaluator returns granted for legitimate Tasmanian pharmacist co-prescribing queries
- Audit query API surfaces the new authorisation pathway

**Estimated V1 effort:** 1 day post-gate

---

## Cross-cutting V1 work not enumerated above

The following are part of normal V1 productionisation but worth noting:

- **Migration application to live DBs.** All 22 v2_substrate migrations (008_part1 through 022) and 2 kb-30/kb-31 migrations are checked in but not applied to any live database. V1 deployment runs them in order through the existing GORM migration runner.
- **Operational dashboards.** `monitoring/dashboards/` is referenced in the Wave 6 SLA doc but not built. V1 builds Grafana dashboards covering: baseline_state recompute lag, identity_review_queue depth, authorisation evaluator p95 latency, EvidenceTrace traversal latency, override-rate per rule.
- **Layer 4 work** (Vaidshala v2 Revision Mapping seven user surfaces) — explicitly out of scope for both Layer 2 and Layer 3 plans. V1 work depends on substrate (Layer 2) + rules (Layer 3) being live.

---

## Summary table — what V1 delivers per gap

| Gap | V1 deliverable | Effort | Owner |
|---|---|---|---|
| 1 | 200+ Tier 2 rules ACTIVE; pilot deprescribing rates measured | 12 weeks | Clinical informatics + V1 author team |
| 2 | 16 layer1-bind markers resolved; 4 ADG rules re-signed; ~48 ADG rules authored | 4 weeks | Layer 1 + Layer 3 teams |
| 3 | MHR + HL7 live wiring for ≥1 pilot facility | 6-8 weeks | Integration + DevOps |
| 4 | Streaming pipeline operational; load test S1 passes | 4-6 weeks | Stream-services team |
| 5 | HAPI engine + CDS Hooks runtime; first pilot prescriber EHR receives a card | 6-8 weeks | Engineering + Clinical |
| 6 | Redis + Kafka wired in kb-30 + kb-31 | 2-3 weeks | DevOps |
| 7 | Wave 5.4 graph load test passes against 6-month dataset | 1-2 weeks | DevOps + Performance |
| 8 | All 32 clinical-author markers resolved | 2-3 weeks elapsed | Clinical informatics |
| 9 | L6 verifier runs clean against live kb-4 PG | 1 day | Ops |
| 10 | Tasmanian ScopeRule ACTIVE | 1 day post-gate | Commercial → Engineering |

**Total V1 effort:** ~22-30 weeks elapsed (parallelisable; 12-16 weeks if streamed)

**Dependencies outside V1 control:** ADHA onboarding latency, UniSA framework PDF availability, pilot facility partnership timing, Tasmanian pilot integration decision, clinical informatics lead bandwidth.

---

## Appendix — Greppable deferral markers (post-rule-expansion)

Run these to enumerate remaining V1 work:

```
grep -r "TODO(layer1-bind)"          backend/  # Layer 1 Pipeline-2 binds (Gap 2)
grep -r "TODO(layer3-v1)"            backend/  # kb-30/kb-31 production wiring (Gap 6)
grep -r "TODO(clinical-author)"      backend/  # clinical lead review (Gap 8)
grep -r "TODO(wave-1-runtime)"       backend/  # HAPI engine integration (Gap 5)
grep -r "TODO(wave-2.7-runtime)"     backend/  # streaming pipeline runtime (Gap 4)
grep -r "TODO(kb-26-acute-repository)" backend/  # kb-26 baseline adapter (Layer 2 followup)
grep -r "TODO(kb-7-binding)"         backend/  # AMT/SNOMED binding when KB-7 publishes
```

---

## Closing

The MVP shipped a defensible substrate (Layer 2: 73% complete with 0% missing) and a working rule library + dual-service authority infrastructure (Layer 3: 54% complete with 3% missing). The remaining work is concentrated in 10 well-bounded gaps, every one of which has a documented owner, prerequisite, sequencing plan, and acceptance evidence.

Once V1 closes Gaps 1-10, Vaidshala has the architectural commitments the v2 product proposal called for: clinical reasoning continuity infrastructure with five interlocking state machines, an EvidenceTrace longitudinal moat, a substrate-aware rule library, and a runtime authorisation evaluator that meets sub-500ms p95 latency.

— V1 Gap Closure Plan, 2026-05-06
