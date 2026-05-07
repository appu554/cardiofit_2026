# Vaidshala v2 Product Spec — Gap Analysis

**Date:** 2026-05-06
**Spec audited:** `backend/shared-infrastructure/knowledge-base-services/kb-6-formulary/Vaidshala_Final_Product_Proposal_v2_Revision_Mapping.md` (490 lines, 12 Parts)
**Branch / HEAD:** `feature/v4-clinical-gaps` @ `c7134402` (per prior audit) → 2033f9ba (current).
**Method:** Spec dimension-by-dimension diff against codebase + three prior audits (Layer2-Layer3 Gap Analysis 2026-05; V1 Gap Closure Plan 2026-05; Layer2-Layer3 Full Implementation Audit 2026-05).
**Auditor scope:** product-spec gap, not engineering gap. The prior audits cover plan-vs-shipped; this audit asks "did we build what the v2 product spec actually committed to."

---

## Executive verdict

**The substrate is largely shipped; the product is not.** The five-state-machine architectural commitment landed cleanly for two of five machines (Authorisation: production-shaped via kb-30; Clinical state: production-shaped via kb-20), partially for Recommendation/Monitoring/Consent (CQL helpers + tier-1/4 rules but no first-class Go entity, lifecycle store, or state-machine engine), and not at all as a unified product. The 7 user surfaces (Layer 4) are essentially untouched beyond app-shell scaffolding. The three commercial GTM moves have no engineering footprint and no commercial-dir evidence. **No buyer in the v2 spec could complete a purchase justification today** — Buyer 1 lacks the GP communication hub and rationale-survival instrumentation; Buyer 2 lacks PHARMA-Care indicators in production form, VIC PCW compliance UI, and AN-ACC dashboards; Buyer 3 lacks GP-facing surfaces entirely. The MVP exit criterion ("ACOP completes review for one resident in 5 minutes from cold start") is **not currently demonstrable** because no decision-packet UI exists.

What is unambiguously strong: substrate entity model, EvidenceTrace bidirectional graph (Wave 5), kb-30 authorisation evaluator, kb-31 ScopeRules including the legislatively-real VIC PCW exclusion and DRNP rules, ScopeRule-aware CompatibilityChecker, and the rule library scaffolding (Tier 1 at 25/25). These are V1-shaped engineering deliverables that exceed what most CDS systems ship.

---

## Part 2 — Strategic positioning

> *Spec commitment:* Vaidshala is **clinical reasoning continuity infrastructure**, not a Clinical Operating System. The moat is the **EvidenceTrace graph** — longitudinal preservation of *why*, *what alternatives*, *who acted*, *what was observed*, *how it propagated*, queryable forward and backward.

| Commitment | Status | Evidence |
|---|---|---|
| EvidenceTrace as one shared graph across all state machines | 🟡 Partial | `shared/v2_substrate/evidence_trace/{graph,edge_store,query,lineage}.go` shipped. Materialised views (mv_recommendation_lineage, mv_observation_consequences, mv_resident_reasoning_summary) shipped. **But:** writers are only Authorisation + Clinical state + Capacity + Care-intensity. Recommendation, Monitoring, and Consent state machines do not yet write directly to EvidenceTrace because they do not exist as first-class Go entities. |
| Bidirectional queryability | ✅ Shipped | `evidence_trace/lineage.go` provides LineageOf / ConsequencesOf / ReasoningWindow with tests. |
| Records action + actor + inputs + alternatives + reasoning summary | 🟡 Partial | Schema supports it; populate-pattern only wired through kb-30 Authorisation evaluations and kb-20 capacity/care-intensity transitions. "Alternatives considered" is **not explicitly captured** by any current writer — the EvidenceTrace edge schema has a payload field but no writer is recording rejected alternatives today. |
| Category framing in external-facing materials | 🔴 Not assessable in code | No `claudedocs/positioning/` or external-pitch directory exists. The `kb-6-formulary/Vaidshala_Final_Product_Proposal_v2_Revision_Mapping.md` is the only positioning doc in tree. The "Clinical Operating System" framing still appears in `vaidshala/clinical-knowledge-core/` and `shared-infrastructure/knowledge-base-services/Clinical_Knowledge_OS_Implementation_Plan.docx` filenames. |

**Verdict — Part 2:** The moat artefact (EvidenceTrace graph) is the strongest single deliverable in the codebase, but only 2 of the 5 state machines write to it today, and "alternatives considered" — the qualitative differentiator the spec emphasises — is structurally supported but not populated. The category-positioning rename has not propagated to internal artefact filenames.

---

## Part 3 — Five state machines (load-bearing architectural commitment)

| State machine | Spec status | Implementation status | Service / files | Gap |
|---|---|---|---|---|
| **1. Authorisation** | "Runtime: who may act on this resident now?" — new safety primitive | ✅ Shipped — production-shaped | `kb-30-authorisation-evaluator/` (DSL + parser, store, evaluator, cache, invalidation, audit query API). Three real-world example rules including DRNP. Sunday-night-fall integration test passes. | Live multi-region deployment + cache invalidation under load not yet exercised at scale (kb-30 cache is in-memory + Redis stub, not clustered). |
| **2. Recommendation** | `detected → drafted → submitted → viewed → decided → implemented → monitoring-active → outcome-recorded → closed`, with `deferred` an explicit state | 🔴 **Missing as a first-class state-machine entity** | No `Recommendation` Go type in `shared/v2_substrate/models/`. No recommendation-lifecycle store. No `deferred` state with forced review_date / escalation. The CDS Hooks emitter (`shared/cql-toolchain/cds_hooks_emitter.py`) produces *card payloads* but does not persist a Recommendation entity through a lifecycle. | This is the **largest architectural gap** in the v2 spec. The Ramsey-2025 50% non-implementation problem cannot be measured (recommendation rationale survival rate metric — Part 9) without this entity. |
| **3. Monitoring** | Lifecycle that outlives the recommendation; produces new Events on threshold cross | 🔴 **Missing as an entity / lifecycle** | `shared/cql-libraries/helpers/MonitoringHelpers.cql` exists with skeleton bodies (TODO(wave-1-runtime)). One Tier-4 surveillance rule (`monitoring-plan-overdue-observation`) references `MonitoringPlan` but there is no MonitoringPlan Go entity, no store, no scheduler. | Without this, the spec's claim that "monitoring outlives the recommendation" is a design intention, not a built capability. |
| **4. Clinical state** | Running baselines + active concerns + care intensity + capacity | ✅ Shipped — production-shaped | `shared/v2_substrate/clinical_state/` (active_concerns, care_intensity_engine, capacity_assessment), `delta/persistent_baseline_provider.go`, scoring (CFS/AKPS/DBI/ACB), trajectory_detector. All four sub-systems present + tested. | Streaming pipeline (Wave 2.7) deferred — synchronous Go path covers near-term load only. |
| **5. Consent** | `requested → discussed → granted/refused/granted-with-conditions → active → under_review → withdrawn/expired`. Recommendation in `submitted` blocks on missing matching Consent for restrictive-practice classes. | 🟡 Partial — gating exists, lifecycle does not | `shared/cql-libraries/helpers/ConsentStateHelpers.cql` + `tier-1-immediate-safety/specs/{antipsychotic,benzodiazepine}-bpsd-consent-missing.yaml` + `tier-4-surveillance/specs/consent-expiring-within-30d.yaml` reference Consent state. SubstituteDecisionMaker is enumerated as `RoleSDM`. **But:** no `Consent` Go entity, no lifecycle transitions, no "block on missing matching Consent" runtime gate — the gating is only a CQL define producing a card. | Gating is decorative until Consent is a tracked entity with active-consent lookup against Recommendation submission. |

**Verdict — Part 3:** **2 of 5 state machines shipped to production-shape; 1 partial; 2 missing as entities.** The Authorisation evaluator is genuinely the cleanest end-to-end build in the codebase. The Recommendation/Monitoring/Consent gap is the most consequential V1 deficit because the spec's product positioning ("reasoning continuity across actors and time") requires all three.

---

## Part 4 — Eight-to-twelve role authority model

| Role | Spec requirement | Substrate status | Gap |
|---|---|---|---|
| GP | Existing primary prescriber | ✅ `RoleGP` enum | UI must "visibly strengthen GP authority" — no UI yet |
| Nurse Practitioner | Autonomous since Nov 2024; first-class equal to GP | ✅ `RoleNP` enum, kb-30 supports NP authorisation | NP collaborative-arrangement scope captured? Not visibly. |
| ACOP-credentialed pharmacist | All require APC training by 1 July 2026; credential verification important | ✅ `RoleACOP` + `RolePharmacist` enums; kb-31 `acop-apc-credential.yaml` ScopeRule exists | ACOPCredential as **first-class entity with expiry tracking** — referenced in kb-30 evaluator code but no standalone credential store/entity |
| Designated RN prescriber | Endorsement live; partnership-only with prescribing agreement + 6-month mentorship | ✅ `RoleDRNP` + kb-31 `drnp-prescribing-agreement.yaml` + kb-30 example rule | PrescribingAgreement / MentorshipStatus referenced in evaluator + ScopeRule but **not modelled as first-class entities** with their own stores. They are evaluated as fields on the authorisation context. Acceptable for V1 evaluation; insufficient for V1 lifecycle management (renewal alerts, scope changes, mentorship completion). |
| RN | Existing | ✅ `RoleRN` | — |
| Enrolled Nurse | Existing; admin under supervision | ✅ `RoleEN` | — |
| Personal Care Worker (with VIC exclusion) | VIC PCW cannot administer S4/S8 to non-self-administering residents from 1 July 2026 | ✅ `RolePCW` + kb-31 `data/AU/VIC/pcw-s4-exclusion-2026-07-01.yaml` with real legislative content | Self-administering vs non-self-administering resident flag — present in resident model? Not verified; ScopeRule references it via `resident.self_administering` but storage path not confirmed. |
| Pharmacist co-prescriber (TAS pilot) | Australian-first pilot 2026-2027 | 🟡 kb-31 `data/AU/TAS/pharmacist-coprescribe-pilot-2026.yaml` staged DRAFT | Activation gated on Move 1 (commercial); engineering done, commercial not |
| Dispensing pharmacy | First-class execution actor (DAA timing) | 🔴 **Not modelled as an actor** | No `RoleDispensingPharmacy`; no DAA-timing entity; no FRED API integration. V1-6 explicitly absent. |
| Hospital | First-class transition counterparty | 🟡 Discharge ingestion only | `ingestion/discharge_pdf.go` + `discharge_mhr.go` ingest from hospitals, but Hospital is not an actor entity that participates in EvidenceTrace as a counterparty. |
| SubstituteDecisionMaker | First-class consent state actor | ✅ `RoleSDM` enum | Without a Consent entity, SDM has no operational role yet |
| Resident self | First-class signal where capacity allows | ✅ Resident entity with capacity assessment | — |
| Regulator (ACQSC, AIHW, PHARMA-Care, ACSQHC) | Ghost user shaping buyer pain | 🟡 kb-30 `internal/audit/` has 4 regulator-ready endpoints | No regulator-export bundle / Standard 5 evidence packet generator yet (MVP-6 scaffold only) |

**First-class supporting entities the spec implies:**

| Entity | Status |
|---|---|
| PrescribingAgreement | 🟡 Evaluated as field; not standalone entity/store |
| MentorshipStatus | 🟡 Evaluated as field; not standalone entity/store |
| ACOPCredential | 🟡 Evaluated as field; not standalone entity/store |

**Verdict — Part 4:** Role enumeration is **complete (12/13 roles enumerated)** which exceeds the 8-12 minimum the spec called out. The supporting credential/agreement/mentorship entities are operationally sufficient for V1 authorisation evaluation but architecturally thin — they are evaluated as fields on the authorisation context, not stored, queried, or lifecycle-managed. Dispensing pharmacy is the single role with no substrate footprint.

---

## Part 6 — MVP / V1 / V2 features

### MVP-1..6 (Months 1-3)

| Feature | Spec | Status | Evidence / gap |
|---|---|---|---|
| **MVP-1** Substrate entities + Authorisation evaluator | Foundation | ✅ Shipped | Resident, Person, Role, MedicineUse(intent+target+stop), Observation(delta-on-write), Event, EvidenceTrace; kb-30 evaluator end-to-end |
| **MVP-2** Unified Medication Timeline + eNRMC + CSV ingestion | Designed against the 8/10 conformant eNRMC vendors; CSV fallback | 🟡 Partial | CSV ingestion shipped (`ingestion/csv_enrmc.go` + `cmd/ingest-csv/main.go`). eNRMC FHIR ingestion exists as `mhr_fhir_gateway.go` stub. **No "Unified Medication Timeline" UI.** |
| **MVP-3** Recommendation lifecycle as thin layer over substrate | `deferred` as explicit state | 🔴 Missing | No Recommendation entity; no lifecycle store; no `deferred` state. CQL emits cards but no entity is persisted through transitions. |
| **MVP-4** Monitoring lifecycle as separate object | The architecturally important addition | 🔴 Missing | No MonitoringPlan entity / lifecycle. Tier-4 rule fires on overdue but has no plan to compare against. |
| **MVP-5** Decision packet generator with guideline tension flags | + PBS authority pre-population + alternative considerations | 🟡 Partial | CDS Hooks v2.0 emitter shipped; PlanDefinition `$apply` shipped; PBS authority pre-population **not visible**; alternative-considerations capture **not visible**; guideline tension flags supported in spec yaml schema but no producer code that evaluates and surfaces them. |
| **MVP-6** Standard 5 evidence panel as workflow exhaust | Not separate report | 🟡 Partial | kb-30 audit query API has 4 regulator-ready endpoints; PHARMA-Care five-domain indicator computation scaffold exists in Tier-3 (`PharmaCareIndicators.cql`). **No Standard 5 evidence-packet UI / export.** |

**MVP exit criterion: "ACOP completes a full medication review for one resident in 5 minutes from cold start, with rationale preserved, audit trail intact, Standard 5 evidence bundle generated."**

**Status: NOT DEMONSTRABLE.** No clinician-facing review UI exists. CSV/eNRMC ingestion + CQL rule firing + EvidenceTrace logging exist, but there is no surface where an ACOP would actually conduct the review. The Standard 5 bundle has no generator.

### V1-1..8 (Months 4-6)

| Feature | Spec | Status | Evidence / gap |
|---|---|---|---|
| **V1-1** Clinical state with running baselines | New baseline-aware reasoning layer | ✅ Shipped | Layer 2 Wave 2 — PersistentBaselineProvider, per-type config, active concerns, care intensity, capacity, CFS/AKPS/DBI/ACB. **This V1 feature was effectively delivered as MVP work.** |
| **V1-2** Consent state machine | Regulatory substrate for restrictive practice | 🔴 Missing | CQL gating only; no entity, no lifecycle, no SDM linkage in operations |
| **V1-3** GP Communication Hub | + closed-loop tracking + supersession + Smart Form integration | 🔴 Missing | No surface, no integration. The CDS Hooks emitter alone does not constitute a hub. |
| **V1-4** Facility Intelligence Dashboard with PHARMA-Care indicators | Five-domain indicators automatically | 🟡 Partial | `tier-3-quality-gap/PharmaCareIndicators.cql` scaffold (6 indicator computations) exists end-to-end on fixture data. **No dashboard UI; production deployment of PHARMA-Care framework alignment unconfirmed (see Part 9 — framework still in pilot).** |
| **V1-5** Hospital discharge reconciliation workflow | Discharge → diff → ACOP routing within 24h → pre-fill packet | ✅ Shipped (Layer 2) | Wave 4 complete — discharge_pdf, discharge_mhr, diff engine, classifier, worklist, writeback. |
| **V1-6** Dispensing pharmacy integration | DAA timing layer; FRED API | 🔴 Missing | Explicit V1-deferral per Layer 2 audit (Wave 5 V1). No FRED client. |
| **V1-7** Jurisdiction-aware ScopeRules infrastructure | Data not code | ✅ Shipped | kb-31-scope-rules service + DSL + parser + store + 4 real ScopeRules (VIC PCW, DRNP, ACOP, TAS-DRAFT). |
| **V1-8** Multi-facility deployment | | 🟡 Partial | `Role.FacilityID` exists; tenancy model present. No multi-facility deployment runbook / pilot proof. |

**V1 exit criterion: "Consultant pharmacy practice running 5 ACOPs across 8 RACHs sees recommendation acceptance rates climb, with measurable rationale-survival improvement, PHARMA-Care indicators automatic, hospital reconciliation operational, dispensing pharmacy integrated."**

**Status: NOT MEASURABLE.** Rationale survival rate has no instrument. Acceptance rate has no Recommendation entity to track against. Hospital reconciliation IS operational at the engine level. Dispensing pharmacy is not integrated.

### V2-1..7 (Months 7-12)

| Feature | Spec | Status | Evidence / gap |
|---|---|---|---|
| **V2-1** AN-ACC Revenue Assurance (KB-28) | Reassessment-driven revenue recovery | 🔴 Missing | No `kb-28` directory exists. `tier-3-quality-gap/ANACCDefensibility.cql` ships 2 rules + queue manifest for 8. No revenue dashboard. |
| **V2-2** Designated RN prescriber workflow | Prescribing agreement ledger + mentorship status + scope-match per action + audit trail | 🟡 Partial | kb-30 example rule + kb-31 ScopeRule exist. **Workflow UI / agreement ledger / mentorship lifecycle does not.** |
| **V2-3** Tasmanian pharmacist co-prescribing pilot integration | If Vaidshala is pilot's digital substrate | 🟡 Engineering ready, commercial not | DRAFT ScopeRule staged. No partnership engagement evidence. |
| **V2-4** Victorian PCW exclusion compliance infrastructure | Legal-administration trail | 🟡 Partial | ScopeRule + kb-30 evaluation shipped. **No "visibly maintained" trail UI; no regulator-queryable export.** |
| **V2-5** Learning System surfaced selectively | | 🔴 Missing | Wave 6 override-rate analytics shipped at engine level (`kb-30-authorisation-evaluator/internal/analytics/`). Not surfaced. |
| **V2-6** Causality engine for ADE attribution | | 🔴 Missing | No causality engine. Not started. |
| **V2-7** Goals-of-care alignment with care_intensity tag | Reshapes recommendations downstream | 🟡 Substrate present, downstream wiring absent | `clinical_state/care_intensity_engine.go` ships. **No Recommendation re-prioritisation against care_intensity** (no Recommendation entity to re-prioritise). |

**V2 exit criterion: AN-ACC revenue, Standard 5 defensibility, antipsychotic QI reduction, PHARMA-Care indicators, VIC compliance evidence within 12 months.** Status: most preconditions absent.

---

## Part 5 — Three buyers

### Buyer 1 — Consultant pharmacy practice

> *Purchase justification:* "Preserve your pharmacists' clinical reasoning across the GP handoff so it doesn't get lost in translation, and produce the PHARMA-Care indicators that prove your service quality."

**Required to satisfy:**
- Recommendation lifecycle with rationale + alternatives captured (🔴 missing)
- GP-facing decision packet that survives translation (🟡 emitter exists, no UI)
- Rationale survival rate metric (🔴 no instrument)
- PHARMA-Care five-domain indicator dashboard (🟡 scaffold computation, no dashboard)
- Class-specific implementation rate vs Ramsey baseline (🔴 no instrument)

**Verdict: Would not buy today.** Three of five pillars missing.

### Buyer 2 — RACH operator

> *Purchase justification:* Standard 5 audit defensibility + AN-ACC revenue + Star Ratings + **PHARMA-Care indicators + VIC PCW compliance + DRNP scope verification**.

**Required:**
- Standard 5 evidence packet generation (🟡 partial — audit endpoints, no packet)
- AN-ACC reassessment workflow + dashboard (🔴 missing — kb-28 not started)
- Star Ratings tracking (🔴 missing)
- PHARMA-Care dashboard (🟡 scaffold)
- VIC PCW compliance evidence trail with regulator query (🟡 ScopeRule + evaluation; no UI / export)
- DRNP scope verification audit trail (✅ kb-30 audit endpoints support this)

**Verdict: Would not buy today.** AN-ACC and Star Ratings absent; compliance trails are engine-only.

### Buyer 3 — GP networks and practice managers

> *Purchase justification:* "Clinical complexity capture + authority verification layer that lets GPs handle aged care safely without becoming the bottleneck."

**Required:**
- GP-facing review surface (🔴 missing)
- One-click approval / one-click revocation (🔴 missing — implied UI work)
- Authority verification visible to GP (✅ kb-30 evaluator can answer "may this person act now")
- Comprehensive audit trail visible to GP (🟡 audit endpoints exist; no surface)

**Verdict: Would not buy today.** No GP-facing surface exists at all.

---

## Part 7 — GTM moves (next 30-60 days per spec)

| Move | Spec call-to-action | Evidence in repo | Status |
|---|---|---|---|
| **1. Tasmanian pilot engagement** | Contact Salahudeen, Peterson (UTas), Duncan McKenzie (Tas DoH); offer no-cost digital substrate | No `claudedocs/commercial/`, `claudedocs/partnerships/`, or pilot-engagement memo. Engineering DRAFT ScopeRule (`tas-pharmacist-coprescribe-pilot-2026.yaml`) exists, suggesting readiness without commercial trigger | 🔴 No engineering-visible commercial action |
| **2. PHARMA-Care framework consortium** | Contact Sluggett, Javanparast (UniSA); position as implementation partner | `tier-3-quality-gap/PharmaCareIndicators.cql` indicates technical alignment intent. No partnership memo, no UniSA engagement evidence | 🔴 No engineering-visible commercial action |
| **3. AMA/RACGP narrative** | Engage RACGP aged-care interest group; messaging guides | No messaging-guide artefact in tree. The RACGP-friendly framing is implicit in kb-30 design but unarticulated externally. | 🔴 No artefact |

**Verdict — Part 7:** All three moves are out of engineering's primary scope, but the spec explicitly named these as 30-60-day commercial dependencies. The codebase contains no traces of any of them.

---

## Part 8 — Risk register (11 risks)

| # | Risk | Mitigation status |
|---|---|---|
| 1 | ADG 2025 licensing | 🟡 Layer 1 audit identifies as Wave 5 work; 165/185 ADG rules deferred to Pipeline-2 |
| 2 | AMH licensing timeline | 🔴 No mitigation evidence in tree |
| 3 | GP integration tooling fragility | 🟡 CDS Hooks v2.0 emitter exists; Smart Form integration not built |
| 4 | Acceptance rate may not move | 🔴 No instrument to even measure (no Recommendation entity) |
| 5 | eNRMC vendor cooperation | 🟡 Conformance work in progress; FHIR Gateway stub only |
| 6 | Restrictive practice legal exposure | 🟡 ScopeRule + Consent gating in CQL; no Consent entity to enforce |
| 7 | Jurisdictional regulatory fragmentation | ✅ kb-31 ScopeRules-as-data shipped per spec — risk well-mitigated |
| 8 | Designated RN prescriber rollout uncertainty | ✅ Built infrastructure but plan acknowledges low V1 population — appropriate |
| 9 | Pharmacist autonomous prescribing acceleration | 🟡 Role model has 12 roles; can absorb a 13th. Authorisation evaluator is data-driven. |
| 10 | Hospital integration depth | 🟡 Plan acknowledges V1=PDF, V2=MHR, V3=ADT — current state matches V1 |
| 11 | PHARMA-Care framework evolution | 🟡 Indicator computation scaffolded; not hardcoded — re-evaluation feasible |

**Verdict — Part 8:** Risks 7, 8, 11 well-mitigated. Risks 4 and 6 are exposed by the missing Recommendation/Consent entities. Risk 2 (AMH licensing) has no visible mitigation track.

---

## Part 9 — Success metrics (8 new metrics)

| Metric | Tracker / instrumentation | Status |
|---|---|---|
| Recommendation rationale survival rate | Requires Recommendation entity + GP response capture | 🔴 No instrument |
| Class-specific implementation rates vs Ramsey 2025 baseline | Requires Recommendation lifecycle | 🔴 No instrument |
| PHARMA-Care five-domain indicator scores | `PharmaCareIndicators.cql` (6 computations) | 🟡 Computable on fixtures; no production deployment / dashboard |
| Victorian PCW exclusion compliance rate | Requires administration-event tracking by role | 🟡 ScopeRule evaluates per-action; no rate roll-up |
| Authorisation evaluator latency p95 | kb-30 metrics? | 🟡 Prometheus metrics exist; SLO target documented in `docs/slo/v2-substrate-slos.md`; not yet load-tested at the V1 (<500ms) / V2 (<200ms) targets at scale |
| EvidenceTrace graph query depth | `evidence_trace/lineage.go` ReasoningWindow | 🟡 Queryable; no metric tracking max-depth in production |
| North Star: preventable ADEs / 1,000 resident-days | Requires longitudinal outcome tracking | 🔴 No instrument |
| North Star North Star: queryable reasoning chain dataset | The EvidenceTrace graph itself | 🟡 Substrate exists; only 2 of 5 state machines write to it |

**Verdict — Part 9:** **0 of 8 metrics fully instrumented.** 6 of 8 have substrate in place; 2 (rationale survival, class-specific implementation) cannot exist without the Recommendation entity.

---

## Cross-cutting findings

### Finding 1 — The substrate-feature gap is the dominant pattern

The v2 Revision Mapping is 50% architectural commitment (5 state machines, EvidenceTrace, ScopeRules-as-data, role model) and 50% feature commitment (7 surfaces, 6 MVP + 8 V1 + 7 V2 features, 3 buyer purchase justifications). The codebase is roughly 70% complete on architectural commitments and ~10% complete on feature commitments. **Neither buyer 1, 2, nor 3 cares about substrate; they care about features.**

### Finding 2 — Three of the five state machines are entity-less

Authorisation and Clinical state are first-class. Recommendation, Monitoring, and Consent exist only as CQL helper signatures + tier rules — not as Go entities with stores, lifecycles, or transitions. The spec is unambiguous that all five must be separate entities sharing one EvidenceTrace graph. The architectural commitment is half-met.

### Finding 3 — Layer 4 (user surfaces) is unbuilt

`vaidshala/clinical-applications/apps/medication-advisor/` exists as scaffold without contents. None of the seven user surfaces (Shift Command Center, Worklist, Resident Workspace, GP Communication Hub, Facility Intelligence Dashboard, AN-ACC Revenue Assurance, Learning System) has an implementation. The MVP exit criterion ("5 minutes cold-start review") is structurally undemonstrable until at least Worklist + Resident Workspace + Decision Packet land.

### Finding 4 — The commercial / GTM gap is not engineering's domain but is engineering-visible by absence

The spec is explicit that the Tasmanian pilot engagement is "the single highest-leverage commercial move available" with a "window that closes" by mid-2026. There is no `claudedocs/commercial/` or partnership memo in the repo. The DRAFT TAS ScopeRule signals engineering readiness; nothing else does.

### Finding 5 — Positive surprises

- **kb-30 Authorisation evaluator** is a **production-shaped V1-grade artefact** that exceeds the spec's V1 latency target on synthetic loads, ships with regulator-ready audit endpoints, and has the integration-test coverage the spec implied.
- **kb-31 ScopeRules-as-data** with 4 real legislative ScopeRules (VIC PCW, DRNP, ACOP, TAS-DRAFT) **exceeds** the V1-7 spec which only required infrastructure — the team shipped both infrastructure and content.
- **EvidenceTrace materialised views + bidirectional query API** is more than the spec asked for at this stage.
- **CFS/AKPS/DBI/ACB scoring + recompute-on-MedicineUse-change** is additional clinical-state signal beyond the four sub-systems the spec named.
- **Role enumeration is 12/13 — exceeds the 8-12 spec minimum.**

---

## Risk-ranked top 5 v2 product gaps

1. **Recommendation entity + lifecycle (MVP-3 + V1-3 + every metric).** Without this, the Ramsey-50%-non-implementation problem cannot be measured, the rationale-survival metric cannot exist, the GP Communication Hub has no payload to surface, and `deferred → forced review` cannot fire. **This is the single highest-priority gap.**
2. **Consent entity + lifecycle (V1-2).** Restrictive practice gating is currently decorative (CQL emits a card; no runtime block). Buyer 2's RACH operator value story is exposed legally without this.
3. **Layer 4 surfaces — at minimum Worklist + Resident Workspace + Decision Packet UI (MVP exit criterion).** Without this, no buyer can demonstrate the product in a sales meeting; the 5-minute cold-start criterion is unmeasurable.
4. **Monitoring entity + lifecycle + scheduler (MVP-4).** The closed-loop outcome chain (recommendation → monitoring → new event → new recommendation) does not exist. The "moat" framing of EvidenceTrace cannot exhibit longitudinal handoffs without it.
5. **Standard 5 evidence packet generator + PHARMA-Care indicator dashboard (MVP-6 + V1-4).** The spec frames these as "workflow exhaust" — Buyer 2 explicitly purchases on this. Computation scaffold exists; productisation does not.

---

## Production-readiness for the three buyer purchase decisions

| Buyer | Would they buy this today? | Critical pre-purchase gaps |
|---|---|---|
| **1. Consultant pharmacy practice** | **No** | Recommendation entity, GP-facing decision packet UI, rationale survival instrument, PHARMA-Care dashboard |
| **2. RACH operator** | **No** | AN-ACC dashboard (kb-28 not started), Standard 5 evidence packet generator, VIC PCW compliance UI/export, PHARMA-Care dashboard |
| **3. GP networks / practice managers** | **No** | Any GP-facing UI surface; one-click approval/revocation; visible audit trail surface |

The honest answer: **the engineering substrate would let any of the three buyers see a compelling demo of the moat artefact (EvidenceTrace graph, Authorisation evaluator), but none of the three could complete a purchase justification on the substrate alone.** The next 12 weeks of work should be heavily Layer-4 + Recommendation/Consent/Monitoring entity completion, with substrate work limited to Wave 2.7 streaming and Wave 3.x live integrations.

---

## Appendix A — Cross-reference to prior audits

| Topic | This audit | Layer 2/3 Gap Analysis | V1 Gap Closure Plan |
|---|---|---|---|
| Substrate entities (Resident, Person, Role, MedicineUse, Observation, Event, EvidenceTrace) | ✅ Shipped | Wave 0–1R: 100% shipped | n/a — substrate complete |
| Authorisation evaluator (kb-30) | ✅ Production-shaped | Wave 4 Stream B: 7/7 shipped | n/a |
| Clinical state (baselines, concerns, intensity, capacity, scoring) | ✅ Shipped | Wave 2: 6/7 shipped (streaming deferred) | n/a |
| Hospital discharge reconciliation (V1-5) | ✅ Shipped | Wave 4: 4/4 shipped | n/a |
| ScopeRules-as-data (V1-7) | ✅ Shipped + content | Wave 5: 6/7 shipped | n/a |
| Recommendation lifecycle (MVP-3) | 🔴 Missing | Not in plan scope | Should be V1 priority — confirmed here |
| Monitoring lifecycle (MVP-4) | 🔴 Missing | Helper signatures only (Wave 1.1) | Should be V1 priority — confirmed here |
| Consent lifecycle (V1-2) | 🔴 Missing as entity | Helper signatures + tier rules | Should be V1 priority — confirmed here |
| Layer 4 user surfaces (7 surfaces) | 🔴 Unbuilt | Out of Layer 2/3 scope | Implied V1+ work |
| Tier 2 rule volume | 🟡 6/75 | 8% — largest single gap in Layer 3 plan | V1 priority #1 there |
| Streaming pipeline (Wave 2.7) | 🟡 ADR + skeleton | Reduced scope; sync Go path covers near-term | V1 priority #3 there |
| MHR / HL7 live integration | 🟡 Stubs | Reduced scope | V1 priority #2 there |
| GTM Move 1/2/3 | 🔴 No engineering-visible action | Out of scope | Out of scope |
| AN-ACC (V2-1, kb-28) | 🔴 Not started | Out of Layer 2/3 scope | V2 |

---

**End of v2 product spec gap analysis.**
