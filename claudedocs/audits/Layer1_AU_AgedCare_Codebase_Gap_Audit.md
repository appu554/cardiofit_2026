# CardioFit Layer 1 Australian Aged Care — Codebase Gap Audit

**Audit Date:** 2026-04-28
**Last Reconciled:** 2026-04-29 (Wave 1 / KB-7 file-level recon — see "Reconciliation Notes" at end)
**Source Spec:** [Layer1_Australian_Aged_Care_Implementation_Guidelines.md](../../Layer1_Australian_Aged_Care_Implementation_Guidelines.md)
**Branch:** feature/v4-clinical-gaps
**Audit Type:** Wave-by-wave gap analysis with confidence levels and effort estimates

---

## Executive Summary

The CardioFit codebase has **foundational infrastructure** for Layer 1 Australian aged care deployment, but **critical content gaps remain** across all six waves defined in the Layer 1 spec. Verdict by wave:

- **Wave 1 (Foundation Terminology):** PARTIAL — KB-7 AU infra in place (SNOMED-AU/AMT downloaders, Neo4j-AU, 35,344 LOINC codes loaded). Remaining gaps are *integration* (GCS→Postgres wire, IHACPA fetch for ICD-10-AM, monthly cron), not *data*. **Reconciliation note:** the "13 LOINC=UNKNOWN entries" claim from the source spec was investigated 2026-04-29 — the issue actually surfaces in **KB-16 extraction output**, not KB-7, and the real count is **1 entry**, not 13. See "KB-16 Data Note" below.
- **Wave 2 (AU Formulary + Drug Labels):** MISSING — KB-6 service skeleton exists; no PBS Schedule loader; no TGA PI/CMI scraper; constitutional DDI projection (2,527 unprojected) still pending.
- **Wave 3 (Explicit-Criteria Rules):** NOT STARTED — KB-4 schema is generic; no STOPP/START v3, Australian PIMs 2024, or Beers 2023 schemas/extraction; `PRESCRIBING_OMISSION` fact_type missing.
- **Wave 4 (Drug-Burden Scoring):** NOT STARTED — No DBI/ACB schema extensions; no CSV loaders.
- **Wave 5 (AMH/eTG):** NOT STARTED — Gated on commercial licensing; no `DEPRESCRIBING_PROTOCOL` or `FRAILTY_INTERACTION` schemas.
- **Wave 6 (Disease-specific AU):** PARTIAL — Tier-5 PBS + Threshold CQL adapters operational; AU market-config overrides exist; tier-4-guidelines/australia/ is empty placeholders; no Heart Foundation/ADS-ADEA/KHA-CARI/RANZCP extraction.

**Total remaining engineering effort:** ~25–34 days (aligns with spec's estimate)
**Critical path:** Wave 1 KB-7 population blocks everything downstream.
**Top commercial action:** Initiate AMH Aged Care Companion licensing **now** — it's a P0 commercial item with multi-week lead time that gates Wave 5.

---

## Wave 1 — Foundation Terminology (KB-7)

### EXISTS
- **SNOMED CT-AU downloader** — OAuth2-authenticated NCTS download (module `32506021000036107`), RF2 snapshot/full extraction with SHA256 verification, GCS staging. Operational.
- **AMT downloader** — Same NCTS pipeline, module `900062011000036103`. Operational.
- **Neo4j-AU instance** — Dedicated `bolt://localhost:7688` with ELK hierarchy materialized; concept lookup <50ms, subsumption <100ms.
- **KB-7 Terminology service** — Port 8092 (per CLAUDE.md), Go-based, PostgreSQL + Redis + Neo4j-AU. Spec-compliant operations exposed.
- **Australian market configs** — [backend/shared-infrastructure/market-configs/australia/](../../backend/shared-infrastructure/market-configs/australia/) with renal, CKM stage-4, CGM, formulary-accessibility overrides.

### PARTIAL
- **LOINC-AU subset** — KB-7 has **35,344 LOINC codes loaded** via [migrations/011_all_loinc_codes.sql](../../backend/shared-infrastructure/knowledge-base-services/kb-7-terminology/migrations/011_all_loinc_codes.sql). Status: ✅ effectively complete from a KB-7 perspective. *(The "13 LOINC=UNKNOWN" issue cited in the source spec was relocated to KB-16 after recon; see "KB-16 Data Note" below.)*
- **ICD-10-AM** — Loader code is complete (1,036 lines at [internal/regional/icd10am/loader.go](../../backend/shared-infrastructure/knowledge-base-services/kb-7-terminology/internal/regional/icd10am/loader.go)) with full XML/CSV/Text parsing, hierarchy inference, and Postgres insert logic. **Gap:** `IHACPAConfig` struct is defined but unused — no actual HTTP fetch from IHACPA. Needs OAuth/cert-auth client OR a CSV-upload fallback.
- **SNOMED-AU + AMT downloaders** — Operational (NCTS OAuth2 → RF2 snapshot → GCS), but the pipeline **stops at GCS**. Postgres loaders exist (`refset_loader.go`, `amt/loader.go`) but are not invoked by the GCP function. Requires Step Functions / GitHub Actions wiring.

### MISSING
- **Monthly NCTS refresh automation** — No cron/scheduler for SNOMED-AU/LOINC-AU/AMT/ICD-10-AM monthly refresh.
- **KB-7 code-resolution validation pass** — No verification that all existing KB-1/4/5/16/20 ADA-derived rows resolve via populated KB-7.

**Confidence:** 60% complete | **Remaining effort:** ~2 days

---

## Wave 2 — Australian Formulary + Drug Labels

### EXISTS
- **KB-6 Formulary service** — Go service on port 8091, GORM + PostgreSQL + Redis, governance framework. Service skeleton only.
- **Generic SPL Pipeline** — US FDA SPL extraction infrastructure exists in medication-service; reusable scaffold for TGA adaptation.

### MISSING
- **PBS Schedule monthly loader** — No code for PBS XML/CSV ingestion. Spec says: 1 day to build.
- **TGA PI/CMI scraper** — No Australian-specific scraper. Spec flags this as the most engineering-heavy piece of Layer 1 (2–3 days dev). TGA has no clean API; PDF extraction needed.
- **TGA PI extraction run** — Spec calls for top 100 RACF drugs (~$2–3 API).
- **CMI extraction for family-facing summaries** — Spec budgets ~$1 API.
- **Constitutional DDI projection** — 2,527 unprojected DDI definitions in KB-5 still pending per Layer 1 spec.

**Confidence:** 15% complete | **Remaining effort:** ~5 days + ~$3–4 API

---

## Wave 3 — Explicit-Criteria Rules

### EXISTS
- **KB-4 Safety schema** — [backend/shared-infrastructure/knowledge-base-services/shared/extraction/schemas/kb4_safety.py](../../backend/shared-infrastructure/knowledge-base-services/shared/extraction/schemas/kb4_safety.py): ContraindicationFact, WarningFact, ClinicalGovernance models. Generic; no AU-specific fact types.
- **KB-1 Dosing schema** — RenalAdjustment, HepaticAdjustment with eGFR/Child-Pugh ranges.
- **KB-5 Interactions schema** — [shared/extraction/schemas/kb5_interactions.py](../../backend/shared-infrastructure/knowledge-base-services/shared/extraction/schemas/kb5_interactions.py): DrugInteraction with severity/effect/management.
- **Fact extractor pipeline** — [shared/tools/guideline-atomiser/fact_extractor.py](../../backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/fact_extractor.py): Claude-based extraction routing to dosing/safety/monitoring/contextual/interactions.
- **CQL guideline registry** — [shared/cql/registry/cql_guideline_registry.yaml](../../backend/shared-infrastructure/knowledge-base-services/shared/cql/registry/cql_guideline_registry.yaml): maps CQL defines ↔ authority ↔ KB fields. Currently US-centric (ADA2026).

### MISSING
- **`KB4ExtractionResult_STOPP_START_v3` schema variant** — Needs `criterion_id`, `physiological_system`, `omission_or_inappropriate`, `evidence_strength_v3` fields.
- **`PRESCRIBING_OMISSION` fact_type** — START half of STOPP/START. New fact_type addition to KB-4.
- **`KB4ExtractionResult_AustralianPIMs_2024` schema variant** — Largely reuses STOPP/START schema with AU-specific section type and safer-alternative refs.
- **STOPP/START v3 extraction** — O'Mahony et al. 2023 paper + supplementary tables. 190 criteria. Pipeline 2 run (~$2–3).
- **Australian PIMs 2024 extraction** — Wang et al. 2024 IMJ. Pipeline 2 run (~$1–2).
- **Beers 2023 verification + enrichment** — OHDSI concept set may have IDs only without recommendation text. 0.5–1 day.
- **L6 loader governance audit trail for AU rules** — Loader infra exists for ADA work; needs AU rule onboarding.
- **CompatibilityChecker pass for tier-4-guidelines/au/** — No CQL defines authored for these criteria yet.

**Confidence:** 25% complete | **Remaining effort:** ~5 days + ~$4–6 API

---

## Wave 4 — Drug-Burden Scoring

### EXISTS
- **KB-20 Contextual schema** — [shared/extraction/schemas/kb20_contextual.py](../../backend/shared-infrastructure/knowledge-base-services/shared/extraction/schemas/kb20_contextual.py): AdverseReactionProfile, ContextualModifierFact (LAB_VALUE/COMORBIDITY/CONCOMITANT_DRUG), completeness grading. Suitable host for DBI/ACB additions.

### MISSING
- **`DBI_ANTICHOLINERGIC_WEIGHT` + `DBI_SEDATIVE_WEIGHT` fact types** in KB-20.
- **`ACB_SCORE` fact type** in KB-20.
- **DBI weight CSV loader** — Monash Australian-specific list. ~30 min work.
- **ACB score CSV loader** — Salahudeen 2015 + AU extensions. ~30 min work.
- **AMT code mapping verification** — Confirm DBI/ACB drug names resolve via KB-7.
- **Patient-level DBI/ACB computation in rule layer** — Layer 3 work but exit criterion is "patient-level scores computable at runtime."

**Confidence:** 10% complete | **Remaining effort:** ~1 day | $0 API

---

## Wave 5 — AMH/eTG Licensed Content (gated)

### EXISTS
- Nothing — fully gated on commercial licensing.

### MISSING
- **`DEPRESCRIBING_PROTOCOL` fact type** — Workflow schema (taper phases, monitoring, withdrawal symptoms, success criteria, fallback). 0.5 day.
- **`FRAILTY_INTERACTION` fact type** — drug + CFS frailty score + condition → modified recommendation. Distinct from age-based PIMs. 0.5 day.
- **`KB1ExtractionResult_DeprescribingProtocol` schema variant** — Phase-based taper schema.
- **AMH Aged Care Companion 2024 extraction** — ~70 chapters, Pipeline 2 (~$8–12 API, 2–3 days runtime). **Gated on AMH license.**
- **eTG Geriatric extraction** — Lower priority. **Gated on Therapeutic Guidelines license.**
- **CQL authoring for AMH-derived deprescribing protocols** — 3–5 days post-extraction.

### Commercial Action Required
- **P0:** Initiate AMH Aged Care Companion licensing conversation immediately. Multi-week lead time.
- **Secondary:** eTG Geriatric license (lower ROI per spec).

**Confidence:** 0% | **Remaining effort:** 8–12 days post-license + ~$13–20 API + license fees

---

## Wave 6 — Disease-Specific Australian Guidelines

### EXISTS
- **Tier-5 AU PBS adapter** — `AustraliaDrugAdapter.cql` (~500 lines): PBS classification (General/Restricted/Authority/Authority Streamlined/S100 HSD/S100 RAAHS/Chemo), TGA registration status, SUSMP scheduling (S2/S3/S4/S8 with state variations), CTG eligibility for Indigenous Australians, PBS Safety Net, biosimilar logic.
- **Tier-5 AU Threshold adapter** — `AustraliaThresholdAdapter.cql` (~500 lines): RACGP Red Book BMI/waist, Diabetes Australia/RACGP HbA1c targets, Heart Foundation BP/lipid targets, KHA CKD staging G1–G5, Indigenous-specific screening ages (35 vs 45), absolute CVD risk framework.
- **AU market-config overrides** — bp_context, renal, ckm_stage4, cgm, inertia, escalation, formulary_accessibility, care_transition.
- **Coverage index** — [vaidshala/clinical-knowledge-core/coverage-index.yaml](../../vaidshala/clinical-knowledge-core/coverage-index.yaml): tracks region/setting/domain; AU coverage % unpopulated.

### PARTIAL
- **Strengthened Quality Standards (Standard 5)** — Spec says KB-28 already references Aged Care Rules 2025 partially. Standard 5 evidence templates not explicit.
- **QI Program indicators** — KB-13 has measure scaffold; specific indicator definitions not loaded.

### MISSING
- **Heart Foundation CV guidelines extraction** — Pipeline 2, ~$1.
- **ADS-ADEA Australian diabetes guidelines extraction** — Pipeline 2, ~$1. (Note: existing 32-dossier ADA 2026 work is **US ADA**, not Australian ADS — these don't fully overlap on PBS-aligned medication sequencing.)
- **KHA-CARI renal guidelines extraction** — Pipeline 2, ~$1. (Largely KDIGO-aligned but with AU overlays.)
- **RANZCP psychotropic guidelines for older adults** — Critical for spec Rules 2 (sedatives + falls) and 8 (antipsychotics in dementia). Pipeline 2, ~$1.
- **ACSQHC AMS standards + Therapeutic Guidelines: Antibiotic** — Mixed (public + licensed). ~$1.
- **Royal Commission recommendations operationalized** — Recommendation 38 (ACOP basis) → KB-13 + KB-28 evidence dimension.
- **tier-4-guidelines/australia/ population** — `australia/racgp/` and `australia/tga/` directories are `.gitkeep`-only placeholders. No CQL defines authored.
- **PSA RMMR Guidelines + ACOP Tier 1/Tier 2 Rules** — Manual ingestion → KB-29 prep-pack templates.

**Confidence:** 50% complete (adapters strong; guideline content thin) | **Remaining effort:** ~5–6 days + ~$5–7 API

---

## Cross-Cutting Observations

### Strengths
1. **Tier-5 CQL adapters production-grade** — PBS and Threshold AU adapters are the strongest single Layer 1 asset in the repo.
2. **All KB services scaffolded** — KB-1/4/5/6/7/16/20/23 Go services exist with health, governance, schema versioning.
3. **Schema extensibility proven** — KB-4 and KB-20 already support governance metadata, conditional logic, completeness grading — easy to extend with new fact types.
4. **Extraction pipeline operational** — Claude-based fact extractor demonstrated on ADA 2026 (32 dossiers) and KDIGO 2024.
5. **KB-7 AU foundation real** — SNOMED-AU + AMT downloaders working; Neo4j-AU live.

### Critical Gaps
1. **AU PIM/safety rules not extracted** — STOPP/START v3, AU PIMs 2024, Beers 2023 are the rule backbone and none are loaded.
2. **Drug-burden scoring absent** — DBI/ACB power Rules 1, 2, 7, 14 of the initial 20-rule set; not even schema-ready.
3. **Deprescribing infrastructure absent** — No protocol workflows; gated on AMH license.
4. **AU formulary loading incomplete** — PBS loader and TGA scraper both missing; KB-6 cannot serve real AU drug data yet.
5. **Guideline extraction limited to US** — ADA 2026 + KDIGO 2024 are US/international; no AU-source extraction has run.
6. **QUM governance under-operationalized** — Standard 5 templates, QI Program indicators, Royal Commission recommendations not yet structured KB rows.

### Dependencies & Blockers
- **KB-7 → all** — Wave 1 LOINC-AU + ICD-10-AM resolution unblocks Waves 2–6.
- **NCTS account approval** — 1–2 weeks; apply at Wave 0.
- **Commercial licensing** — AMH and Therapeutic Guidelines have multi-week lead times.
- **TGA scraper** — Most fragile engineering piece; consider commercial MIMS/AusDI alternative.

---

## Risk Register

| Risk | Severity | Trigger | Mitigation |
|------|----------|---------|-----------|
| AMH/eTG licensing delay blocks Wave 5 | HIGH | Critical-path dependency | Start license conversation immediately; product launchable without Wave 5 |
| TGA PI/CMI scraping fragility | HIGH | TGA format changes | Allocate 2–3 days dev + ongoing maintenance; evaluate MIMS API as alternative |
| ~~LOINC-AU 13 unresolved entries~~ | ~~MEDIUM~~ | **Resolved 2026-04-29:** recon found only 1 entry, located in KB-16 (not KB-7). Trivial to fix — see KB-16 Data Note below. | n/a |
| STOPP/START v3 + AU PIMs copyright | MEDIUM | Wave 3 extraction | Verify with legal counsel; falling back to "reference + own restatement" is universally accepted |
| 2,527 DDI projection backlog | LOW | KB-5 completeness | Sequence within Wave 2; non-blocking for MVP |

---

## Status-By-Component Matrix

| Wave | Component | Status | Confidence | Effort | API $ |
|------|-----------|--------|------------|--------|-------|
| 1 | SNOMED-AU loader | ✅ EXISTS | 85% | — | $0 |
| 1 | AMT loader | ✅ EXISTS | 85% | — | $0 |
| 1 | LOINC-AU codes loaded in KB-7 | ✅ EXISTS | 95% | — | $0 |
| 1 | ICD-10-AM loader (code present, IHACPA fetch stubbed) | ⚠️ PARTIAL | 60% | 1–2d | $0 |
| 1 | Monthly NCTS refresh cron | ❌ MISSING | 0% | 0.5d | $0 |
| 1 | KB-7 code-resolution validation | ❌ MISSING | 0% | 1d | $0 |
| 2 | PBS Schedule loader | ❌ MISSING | 0% | 1d | $0 |
| 2 | TGA PI/CMI scraper | ❌ MISSING | 0% | 2–3d | $0 |
| 2 | TGA PI extraction (top 100 drugs) | ❌ MISSING | 0% | 1d | ~$2–3 |
| 2 | CMI extraction | ❌ MISSING | 0% | 1d | ~$1 |
| 2 | DDI projection (2,527 backlog) | ⚠️ PARTIAL | 0% | 0.5d | $0 |
| 3 | STOPP/START v3 schema variant | ❌ MISSING | 0% | 0.5d | $0 |
| 3 | `PRESCRIBING_OMISSION` fact_type | ❌ MISSING | 0% | (in above) | $0 |
| 3 | AU PIMs 2024 schema variant | ❌ MISSING | 0% | 0.25d | $0 |
| 3 | STOPP/START extraction | ❌ MISSING | 0% | 1d | ~$2–3 |
| 3 | AU PIMs extraction | ❌ MISSING | 0% | 0.5d | ~$1–2 |
| 3 | Beers 2023 verification + enrich | ⚠️ PARTIAL | 30% | 0.5–1d | ~$1 |
| 3 | L6 loader AU governance run | ❌ MISSING | 0% | 0.5d | $0 |
| 3 | CompatibilityChecker au/ pass | ❌ MISSING | 0% | 0.5d | $0 |
| 4 | DBI weight schema (KB-20) | ❌ MISSING | 0% | 0.25d | $0 |
| 4 | ACB score schema (KB-20) | ❌ MISSING | 0% | 0.25d | $0 |
| 4 | DBI CSV load (Monash) | ❌ MISSING | 0% | 0.25d | $0 |
| 4 | ACB CSV load (Salahudeen+AU) | ❌ MISSING | 0% | 0.25d | $0 |
| 4 | AMT code mapping verification | ❌ MISSING | 0% | 0.25d | $0 |
| 5 | `DEPRESCRIBING_PROTOCOL` fact_type | ❌ MISSING | 0% | 0.5d | $0 |
| 5 | `FRAILTY_INTERACTION` fact_type | ❌ MISSING | 0% | 0.5d | $0 |
| 5 | AMH Aged Care Companion extraction | ❌ MISSING (gated) | 0% | 2–3d | ~$8–12 |
| 5 | eTG Geriatric extraction | ❌ MISSING (gated) | 0% | 2–3d | ~$5–8 |
| 5 | AMH-derived CQL authoring | ❌ MISSING (gated) | 0% | 3–5d | $0 |
| 6 | Heart Foundation CV extraction | ❌ MISSING | 0% | 0.5d | ~$1 |
| 6 | ADS-ADEA diabetes (AU) extraction | ❌ MISSING | 0% | 0.5d | ~$1 |
| 6 | KHA-CARI renal extraction | ❌ MISSING | 0% | 0.5d | ~$1 |
| 6 | RANZCP psychotropic extraction | ❌ MISSING | 0% | 0.5d | ~$1 |
| 6 | ACSQHC AMS + Therapeutic Guidelines | ❌ MISSING | 0% | 0.5d | ~$1 |
| 6 | Standard 5 evidence requirements | ⚠️ PARTIAL | 20% | 1d | $0 |
| 6 | QI Program indicator definitions | ⚠️ PARTIAL | 20% | 1d | $0 |
| 6 | Royal Commission Rec. 38 → KB-13 | ❌ MISSING | 0% | 0.5d | $0 |
| 6 | tier-4-guidelines/australia/ population | ❌ EMPTY | 0% | (covered above) | $0 |
| 6 | PSA RMMR + ACOP Tier 1/2 → KB-29 | ❌ MISSING | 0% | 1d | $0 |

**Totals (excluding Wave 5 gated work):** ~17–21 days engineering, ~$11–17 API
**With Wave 5:** ~25–34 days engineering, ~$24–37 API + license fees

---

## Recommended Sequencing (next 7 weeks)

### Week 0 (immediately, parallel)
- Apply for NCTS account (1–2 week approval).
- Initiate AMH Aged Care Companion licensing conversation.
- Legal counsel review of STOPP/START v3 + AU PIMs 2024 copyright posture.

### Week 1 — Wave 1 completion
- ~~Resolve 13 LOINC-AU UNKNOWN entries~~ — *Reframed as KB-16 data fix (1 entry); see "KB-16 Data Note" below. ~10 minutes, not on Wave 1 critical path.*
- Implement IHACPA fetch for ICD-10-AM loader (or CSV-upload fallback).
- Build monthly NCTS refresh cron + complete Step Functions state machine for GCS→Postgres wire.
- Run KB-7 code-resolution validation pass against existing KB-1/4/5/16/20 rows.

### Week 2 — Wave 2
- Build PBS Schedule monthly loader.
- Build TGA PI/CMI scraper (adapt SPL pipeline).
- Run TGA PI extraction on top 100 RACF drugs + CMI.
- Sequence the 2,527 DDI constitutional projection.

### Weeks 3–4 — Waves 3 + 4 (parallel)
- Author STOPP/START v3 + AU PIMs schemas + `PRESCRIBING_OMISSION` fact_type.
- Run Pipeline 2 extraction on STOPP/START + AU PIMs.
- Verify Beers 2023 OHDSI concept set; enrich if recommendation text missing.
- Author DBI/ACB schema extensions; CSV-load Monash + Salahudeen lists.
- L6 loader + governance run for all extracted AU rules.
- CompatibilityChecker pass for au/ tier-4-guidelines.

### Weeks 5–7 — Wave 6 (parallel with Wave 5 if licensed)
- Pipeline 2 extraction: Heart Foundation, ADS-ADEA, KHA-CARI, RANZCP, ACSQHC AMS.
- Manual ingestion: Standard 5 evidence requirements, QI Program indicators, Royal Commission Rec. 38, PSA RMMR + ACOP Tier 1/2 rules.
- Populate `vaidshala/clinical-knowledge-core/tier-4-guidelines/australia/{racgp,tga}/` with CQL defines.

### Wave 5 (gated trigger)
- On AMH/eTG license execution: author `DEPRESCRIBING_PROTOCOL` + `FRAILTY_INTERACTION` schemas, run extraction, author CQL.

---

---

## KB-16 Data Note: LOINC=`<UNKNOWN>` Cleanup

**Status:** Small data fix, **not a Wave 1 / KB-7 task**.
**Service affected:** KB-16 Lab Interpretation (extraction output, not seed data or DB).
**Effort:** ~10 minutes.

### Background
The source Layer 1 spec (Source O — LOINC AU subset, ~line 402) referenced *"the LOINC=UNKNOWN problem flagged in earlier reviews — the 13 unresolved KB-16 entries"*. The first version of this audit (2026-04-28) carried that claim forward into Wave 1 / KB-7 work. A 2026-04-29 file-level recon found:

- **KB-7 LOINC coverage is fine:** 35,344 LOINC codes loaded via [migrations/011_all_loinc_codes.sql](../../backend/shared-infrastructure/knowledge-base-services/kb-7-terminology/migrations/011_all_loinc_codes.sql).
- **KB-16 seed CSVs are clean:** [loinc_labs.csv](../../backend/shared-infrastructure/knowledge-base-services/shared/cmd/phase1-ingest/data/loinc_labs.csv) (51 rows) and [loinc_labs_expanded.csv](../../backend/shared-infrastructure/knowledge-base-services/shared/cmd/phase1-ingest/data/loinc_labs_expanded.csv) (353 rows) have no UNKNOWN entries.
- **KB-16 DB schema rejects UNKNOWN:** [conditional_ranges.go:18](../../backend/shared-infrastructure/knowledge-base-services/kb-16-lab-interpretation/pkg/reference/conditional_ranges.go#L18) declares `loinc_code` as `not null`.
- **Only 1 actual UNKNOWN found**, in extraction-stage JSON output (not yet ingested):

### The single entry
**File:** [ccb_monitoring_targeted.json:10](../../backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/output/v4/job_monkeyocr_df538e50-0170-4ef8-862d-5b0a7c48e4ff/l3_output/ccb_monitoring_targeted.json#L10)
**Source:** KDIGO 2022 Diabetes in CKD, PP 1.2.1
**Drug class:** dihydropyridine calcium channel blocker
**Lab:** "blood pressure" — `loincCode: "<UNKNOWN>"`

Blood pressure is a vital sign rather than a true lab, so the LOINC mapping options are:
- **85354-9** — BP panel with all children optional (quick fix)
- **8480-6** + **8462-4** — Systolic + Diastolic split (faithful to KDIGO PP 1.2.1 systolic <120 target)

### Recommended action
- Apply chosen LOINC fix to the JSON file.
- Mark this audit item closed.
- *(Optional architectural follow-up, deferred:)* consider whether BP belongs in a vital-signs schema rather than KB-16 lab interpretation. Out of scope for current Layer 1 work.

### Why this matters for the audit
- The "13 unresolved entries" claim is overstated for the current codebase state. Wave 1 / KB-7 is **not blocked** by this issue.
- The *real* Wave 1 critical-path items are: ICD-10-AM IHACPA fetch, GCS→Postgres wiring, monthly refresh scheduler, cross-KB code-resolution validation.

---

## Reconciliation Notes

### 2026-04-29 — Wave 1 / KB-7 file-level recon

A focused recon of the KB-7 service produced these refinements vs. the 2026-04-28 audit:

| Original audit claim | Reconned reality |
|---|---|
| 13 LOINC=UNKNOWN entries blocking Wave 1 exit | Only 1 entry, located in KB-16 extraction output (not KB-7). See KB-16 Data Note above. |
| ICD-10-AM "MISSING" | 1,036-line loader exists and is functionally complete; only the IHACPA fetch is stubbed. |
| SNOMED-AU/AMT downloaders "operational" | Operational *to GCS only*. Postgres loaders exist but aren't invoked — pipeline gap, not missing components. |
| Monthly refresh cron "MISSING" | EventBridge rule exists in `aws/cloudformation/step-functions.yaml`; Step Functions state machine is incomplete. |
| KB-7 port "8092" (per CLAUDE.md) | Operations guide says 8087 — discrepancy needs resolving against `cmd/server/main.go`. |

**Net effect on plan:** Wave 1 effort estimate is unchanged (~2–3 days) but the *shape* of the work is different — it's mostly integration/wiring, not new code. The "13 LOINC UNKNOWN" item drops out as a phantom blocker.

---

*End of audit — generated 2026-04-28; reconciled 2026-04-29; against branch `feature/v4-clinical-gaps`.*
