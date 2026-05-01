# Australian Aged Care KB Stack — Master README

> **Purpose**: don't lose track of where AU-specific data lives across the
> KB platform. Each KB service has its own database; this doc is the
> single index showing which AU dataset is loaded into which DB and how
> to refresh / verify / extend it.

**Last updated:** 2026-04-29
**Source plan:** [Layer 1 Australian Aged Care Implementation Guidelines](../../../Layer1_Australian_Aged_Care_Implementation_Guidelines.md)
**Gap audit:** [claudedocs/audits/Layer1_AU_AgedCare_Codebase_Gap_Audit.md](../../../claudedocs/audits/Layer1_AU_AgedCare_Codebase_Gap_Audit.md)

---

## At a glance — what's loaded right now

| KB | Data | Rows | Source | Status |
|---|---|---|---|---|
| **KB-7** terminology | SNOMED CT-AU concepts | 714,687 | NCTS RF2 30 Apr 2026 | ✅ live |
| **KB-7** terminology | SNOMED CT-AU descriptions (en-au) | 2,222,900 | NCTS RF2 30 Apr 2026 | ✅ live |
| **KB-7** terminology | SNOMED CT-AU relationships | 5,028,804 | NCTS RF2 30 Apr 2026 | ✅ live |
| **KB-7** terminology | AU refset memberships | 1,364,790 | NCTS RF2 30 Apr 2026 | ✅ live |
| **KB-7** terminology | AMT packs | 54,303 | NCTS AMT TSV 30 Apr 2026 | ✅ live |
| **KB-7** terminology | ICD-10-AM codes | 0 | IHACPA (commercial) | ⚠️ schema ready, license blocked |
| **KB-6** formulary | PBS items | 6,935 | PBS API CSV 1 Apr 2026 | ✅ live |
| **KB-6** formulary | PBS ATC codes (`kb6_pbs_rel_atc_codes`) | 7,891 | PBS API CSV 1 Apr 2026 | ✅ live |
| **KB-6** formulary | PBS item↔ATC linkages (`kb6_pbs_rel_item_atc`) | 6,803 | PBS API CSV 1 Apr 2026 | ✅ live |
| **KB-6** formulary | PBS prescribers (`kb6_pbs_rel_prescribers`) | 10,324 | PBS API CSV 1 Apr 2026 | ✅ live |
| **KB-6** formulary | PBS restrictions full text (`kb6_pbs_rel_restrictions`) | 3,553 | PBS API CSV 1 Apr 2026 | ✅ live |
| **KB-6** formulary | PBS item↔restriction linkages (`kb6_pbs_rel_item_restrictions`) | 23,073 | PBS API CSV 1 Apr 2026 | ✅ live |
| **KB-6** formulary | PBS prescribing texts (`kb6_pbs_rel_prescribing_texts`) | 9,007 | PBS API CSV 1 Apr 2026 | ✅ live |
| **KB-6** formulary | PBS item↔prescribing-text linkages (`kb6_pbs_rel_item_prescribing_texts`) | 12,915 | PBS API CSV 1 Apr 2026 | ✅ live |
| **KB-6** formulary | PBS criteria (`kb6_pbs_rel_criteria`) | 2,820 | PBS API CSV 1 Apr 2026 | ✅ live |
| **KB-6** formulary | PBS parameters (`kb6_pbs_rel_parameters`) | 3,501 | PBS API CSV 1 Apr 2026 | ✅ live |
| **KB-6** formulary | PBS criteria↔parameter linkages (`kb6_pbs_rel_criteria_parameters`) | 3,885 | PBS API CSV 1 Apr 2026 | ✅ live |
| **KB-6** formulary | PBS indications (`kb6_pbs_rel_indications`) | 646 | PBS API CSV 1 Apr 2026 | ✅ live |
| **KB-6** formulary | PBS programs (`kb6_pbs_rel_programs`) | 17 | PBS API CSV 1 Apr 2026 | ✅ live |
| **KB-3** guidelines | Pipeline 2 layered spans / sections / tree / corrections | 11,873 | KDIGO 2022 Pipeline 2 | ✅ live |
| **KB-1** drug rules | KDIGO L3 dosing facts (typed `drug_rules`) | 37 | KDIGO 2022 L3 | ✅ live |
| **KB-1** drug rules | KDIGO L3 staging | 64 | KDIGO 2022 L3 | ✅ live |
| **KB-4** patient safety | KDIGO L3 typed safety facts (`kb4_l3_safety_facts`) | 170 | KDIGO 2022 L3 | ✅ live |
| **KB-4** patient safety | KDIGO L3 staging | 64 | KDIGO 2022 L3 | ✅ live |
| **KB-16** lab interp | KDIGO L3 typed lab requirements (`kb16_l3_lab_requirements`) | 97 | KDIGO 2022 L3 | ✅ live |
| **KB-16** lab interp | KDIGO L3 staging | 63 | KDIGO 2022 L3 | ✅ live |
| **KB-20** patient profile | KDIGO L3 typed ADR profiles (`adverse_reaction_profiles`) | 112 | KDIGO 2022 L3 | ✅ live |
| **KB-20** patient profile | KDIGO L3 staging | 63 | KDIGO 2022 L3 | ✅ live |
| **KB-4** patient safety | STOPP v3 (drugs-to-stop, 10 sections) | 80 | O'Mahony 2023, Age and Ageing | ✅ live |
| **KB-4** patient safety | START v3 (drugs-to-start / PPO, 9 sections) | 40 | O'Mahony 2023 | ✅ live |
| **KB-4** patient safety | AGS Beers 2023 (US PIM, RxCUI + ATC + ACB scores) | 57 | AGS 2023 | ✅ live |
| **KB-4** patient safety | AU APINCHs high-alert medications | 33 | ACSQHC | ✅ live |
| **KB-4** patient safety | TGA black-box warnings (Australian) | 52 | TGA | ✅ live |
| **KB-4** patient safety | TGA pregnancy categories | 55 | TGA | ✅ live |
| **KB-4** patient safety | ACB anticholinergic burden scale | 56 | Boustani 2008 + extensions | ✅ live (Wave 4 partial) |
| **KB-4** patient safety | Australian PIMs 2024 (Wang IMJ) | 19 | Wiley DOI 10.1111/imj.16322 | ✅ live (criterion_set=AU_PIMS_2024, Delphi-curated, re-phrased) |
| **KB-4** patient safety | Drug Burden Index weights (DBI) | 0 | Hilmer 2007 + Monash CMUS | 🚫 **deferred — procurement blocked** (no JAMA supp / Monash CSV / Kouladjian 2014 obtained; will NOT synthesize weights — clinical safety risk) |

**Totals:** 9.4M SNOMED-AU rows + 6.9k PBS items + **84.4k PBS relational rows** + 11.9k pipeline spans + 416 typed clinical facts + 392 explicit-criteria rules = approx. **9.51M rows** across 5 separate KB DBs, all loaded fresh 28-29 April 2026.

---

## Where each AU dataset lives

Every KB service has its own Postgres DB on its own container. The table below is the address book.

| KB service | Container | Host port | DB | User | Password | Key AU tables |
|---|---|---|---|---|---|---|
| KB-7 terminology | `kb7-postgres` | **5457** | `kb_terminology` | `postgres` | `password` | `kb7_snomed_*`, `kb7_amt_pack`, `kb7_icd10am_*`, `kb7_achi_*` |
| KB-6 formulary | `kb6-postgres` | **5447** | `kb_formulary` | `kb6_admin` | `kb6_secure_password` | `kb6_pbs_items`, `kb6_pbs_authorities`, `kb6_pbs_indications`, `kb6_pbs_section_100`, `kb6_pbs_restrictions`, `kb6_pbs_prescriber_types`, `kb6_pbs_load_log` |
| KB-3 guidelines | `kb3-postgres` | **5435** | `kb3_guidelines` | `postgres` | `password` | `kb3_pipeline2_*` (jobs, raw_spans, merged_spans, sections, tree, rxnorm_corrections, validation_reports) |
| KB-1 drug-rules | `kb1-postgres` | **5481** | `kb1_drug_rules` | `kb1_user` | `kb1_password` | `kb_l3_staging`, `drug_rules` (filter `source_set_id LIKE 'L3:%'`) |
| KB-4 patient-safety | `kb4-patient-safety-postgres` | **5440** | `kb4_patient_safety` | `kb4_safety_user` | `kb4_safety_password` | `kb_l3_staging`, `kb4_l3_safety_facts`, `kb4_explicit_criteria` (STOPP/START/Beers) |
| KB-16 lab-interpretation | `kb16-postgres` | **5446** | `kb_lab_interpretation` | `kb16_user` | `kb_password` | `kb_l3_staging`, `kb16_l3_lab_requirements` |
| KB-20 patient-profile | `kb20-postgres` | **5436** | `kb_service_20` | `kb20_user` | `kb20_password` | `kb_l3_staging`, `adverse_reaction_profiles` (filter `source = 'L3_KDIGO'`) |

**Note**: container-to-container access uses port **5432** internally. Host-to-container uses the port shown above.

---

## Service endpoints (what's exposed)

| Service | Host process / container | Port | AU routes |
|---|---|---|---|
| **KB-7 terminology** | host process (binary `bin/kb7-server`) | **8094** | `/v1/au/health`, `/v1/au/concepts/:code`, `/v1/au/concepts/:code/{children,parents}`, `/v1/au/concepts/search?module=au`, `/v1/au/amt/{search,substance/:id,pack/:id}` |
| KB-6 formulary | `kb6-formulary` container | 8091 | (existing routes; PBS data queryable directly via Postgres for now) |
| KB-3 guidelines | `kb3-guidelines` container | 8083 | (existing routes; Pipeline 2 layers queryable via Postgres) |
| KB-1 drug-rules | `kb-drug-rules` container | 8081 | KDIGO L3 dosing data in `drug_rules` table (filter source_set_id) |
| KB-4 patient-safety | `kb4-patient-safety-service` container | 8088 | new `kb4_l3_safety_facts` table |
| KB-16 lab-interpretation | `kb-16-lab-interpretation` container | 8116 | new `kb16_l3_lab_requirements` table |
| KB-20 patient-profile | `kb20-service` container | 8131 | KDIGO ADR profiles in `adverse_reaction_profiles` (filter `source = 'L3_KDIGO'`) |

---

## Loader scripts — single source of truth

All AU loading is via Python scripts under [kb-7-terminology/scripts/](kb-7-terminology/scripts/) and [kb-6-formulary/scripts/](kb-6-formulary/scripts/). Each is **idempotent** (safe to re-run).

### KB-7 — SNOMED CT-AU + AMT (Wave 1)

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-7-terminology

# 1. Set NCTS OAuth2 credentials (gitignored)
echo "NCTS_CLIENT_ID=<your-id>"        > .env.ncts.local
echo "NCTS_CLIENT_SECRET=<your-secret>" >> .env.ncts.local

# 2. Download (free, NCTS account required)
set -a && source .env.ncts.local && set +a
python3 scripts/download_snomed_au_local.py --package-type SCT_RF2_SNAPSHOT
python3 scripts/download_snomed_au_local.py --package-type AMT_TSV

# 3. Load
python3 scripts/load_snomed_au_rf2.py
python3 scripts/load_amt.py

# 4. Delete creds
rm .env.ncts.local
```

### KB-7 — ICD-10-AM (Wave 1 Phase B1, gated)

```bash
# Awaits IHACPA license. When data file is on disk:
python3 scripts/load_icd10am.py \
  --edition "12th edition" \
  --tabular  data/icd10am/12th/icd10am_tabular.xml \
  --index    data/icd10am/12th/icd10am_index.csv \
  --achi-tab data/icd10am/12th/achi_tabular.xml \
  --achi-idx data/icd10am/12th/achi_index.csv
```
Details: [kb-7-terminology/scripts/README_ICD10AM.md](kb-7-terminology/scripts/README_ICD10AM.md)

### KB-6 — PBS Schedule (Wave 2)

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-6-formulary

# Free public download (no auth)
mkdir -p data/pbs
curl -fL -A 'Mozilla/5.0' \
  -o data/pbs/2026-04-01-PBS-API-CSV-files.zip \
  'https://www.pbs.gov.au/downloads/2026/04/2026-04-01-PBS-API-CSV-files.zip'
unzip -q -o data/pbs/2026-04-01-PBS-API-CSV-files.zip -d data/pbs/extracted/

python3 scripts/load_pbs.py \
  --csv data/pbs/extracted/tables_as_csv/items.csv \
  --schedule-date 2026-04-01

# Then load the FULL relational graph (12 CSVs, ~84k rows): authorities,
# restrictions text, indications, ATC codes, prescribers, criteria,
# parameters, programs. TRUNCATE+COPY semantics, idempotent monthly.
python3 scripts/load_pbs_relational.py --apply-migrations
```
Details: [kb-6-formulary/scripts/README_PBS.md](kb-6-formulary/scripts/README_PBS.md)

After both loaders run, KB-6 supports decision-support joins like:
```sql
-- "What authority text applies to PBS code X?"
SELECT i.drug_name, ir.benefit_type_code, r.li_html_text
FROM kb6_pbs_items i
JOIN kb6_pbs_rel_item_restrictions ir ON ir.pbs_code = i.pbs_code
JOIN kb6_pbs_rel_restrictions r ON r.res_code = ir.res_code
WHERE i.pbs_code = '10001J';
```

### KB-1/4/16/20 — KDIGO L3 fact pipeline (4-stage)

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-7-terminology

# Stage 1: load raw L3 JSONs into per-KB staging tables
python3 scripts/load_l3_facts_staging.py

# Stage 2: resolve drug-name -> RxCUI via RxNav-in-a-Box (localhost:4000)
python3 scripts/resolve_rxnorm_codes.py

# Stage 3: extract staged rows into typed clinical tables
python3 scripts/extract_l3_to_typed.py

# Stage 4: load Pipeline 2 layers (L1/L2/L2.5/L4) into KB-3
python3 scripts/load_pipeline2_layers.py
```

### KB-4 — Explicit criteria (Wave 3): STOPP v3 + START v3 + Beers 2023

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-4-patient-safety

# Source YAMLs already in repo (no download needed)
python3 scripts/load_explicit_criteria.py

# Per-set load
python3 scripts/load_explicit_criteria.py --set STOPP_V3
python3 scripts/load_explicit_criteria.py --set START_V3
python3 scripts/load_explicit_criteria.py --set BEERS_2023
```
Result: 177 rules in `kb4_explicit_criteria` (80 STOPP + 40 START + 57 Beers).

### Cross-KB validator (Wave 1 exit check)

Validates that every code-bearing column across consumer KBs (KB-1, KB-4) resolves in KB-7's authoritative reference tables. Optional `--rxnav` flag cross-classifies unresolved RxCUIs against RxNav-in-a-Box (localhost:4000) to distinguish retired/obsolete/remapped codes (fixable) from true phantoms (YAML typos).

```bash
cd kb-7-terminology
python3 scripts/validate_kb_codes.py                    # text report
python3 scripts/validate_kb_codes.py --json             # CI-friendly
python3 scripts/validate_kb_codes.py --rxnav --sample 25 # full classification
```

Exit codes: `0` if all PASS/WARN, `2` if any FAIL.

**Latest run (2026-04-29):** 3 FAILs — KB-4 RxNorm 78–82% resolution (mostly retired RxCUIs, remappable via RxNav), KB-4 ICD-10 only 2.3% (format mismatch: KB-7 holds ICD-10-CM, START_V3 uses WHO ICD-10). Gap report + remediation paths in [claudedocs/audits/2026-04-29_kb_cross_reference_gap_report.md](../../claudedocs/audits/2026-04-29_kb_cross_reference_gap_report.md).

---

## Verification — quick queries you can run anytime

### KB-7 SNOMED CT-AU + AMT

```bash
# AU-extension concepts only
PGPASSWORD=password docker exec -it kb7-postgres psql -U postgres -d kb_terminology -c "
SELECT count(*) FROM kb7_snomed_concept
WHERE module_id = 32506021000036107 AND active = 1
"  # => 169,906

# Find all packs of paracetamol via AMT
PGPASSWORD=password docker exec -it kb7-postgres psql -U postgres -d kb_terminology -c "
SELECT mp_pt, count(DISTINCT ctpp_sctid) AS num_packs
FROM kb7_amt_pack
WHERE lower(mp_pt) LIKE '%paracetamol%'
GROUP BY mp_pt ORDER BY num_packs DESC LIMIT 5
"
```

### KB-6 PBS

```bash
# Authority Required + Streamlined breakdown
PGPASSWORD=kb6_secure_password docker exec -it kb6-postgres psql -U kb6_admin -d kb_formulary -c "
SELECT schedule_section, count(*) FROM kb6_pbs_items GROUP BY 1 ORDER BY 2 DESC
"

# Find all metformin items
PGPASSWORD=kb6_secure_password docker exec -it kb6-postgres psql -U kb6_admin -d kb_formulary -c "
SELECT pbs_code, drug_name, schedule_section
FROM kb6_pbs_items
WHERE lower(drug_name) LIKE '%metformin%' AND is_section_100 = false
LIMIT 5
"
```

### KB-7 service HTTP

```bash
curl http://localhost:8094/v1/au/health
curl http://localhost:8094/v1/au/concepts/73211009                 # Diabetes mellitus
curl "http://localhost:8094/v1/au/amt/search?q=metformin&level=mp&limit=5"
curl "http://localhost:8094/v1/au/concepts/search?q=aboriginal&module=au&limit=5"
```

### KB-4 explicit criteria (Wave 3)

```bash
# Counts per criterion set
PGPASSWORD=kb4_safety_password docker exec -it kb4-patient-safety-postgres \
  psql -U kb4_safety_user -d kb4_patient_safety -c "
SELECT criterion_set, count(*) FROM kb4_explicit_criteria GROUP BY 1
"
# => STOPP_V3 80, START_V3 40, BEERS_2023 57

# All Beers AVOID drugs with high anticholinergic burden (ACB >= 3)
PGPASSWORD=kb4_safety_password docker exec -it kb4-patient-safety-postgres \
  psql -U kb4_safety_user -d kb4_patient_safety -c "
SELECT drug_name, recommendation, acb_score
FROM kb4_explicit_criteria
WHERE criterion_set='BEERS_2023' AND recommendation='AVOID' AND acb_score >= 3
ORDER BY drug_name LIMIT 10
"

# START criteria for atrial fibrillation (ICD-10 I48)
PGPASSWORD=kb4_safety_password docker exec -it kb4-patient-safety-postgres \
  psql -U kb4_safety_user -d kb4_patient_safety -c "
SELECT criterion_id, condition_text, recommended_drugs
FROM kb4_explicit_criteria
WHERE criterion_set='START_V3' AND 'I48' = ANY(condition_icd10)
"
```

---

## Backups

Both `kb_terminology` and `kb_formulary` should be backed up periodically. Pattern:

```bash
# KB-7 terminology
docker exec kb7-postgres pg_dump -U postgres -F c --compress=6 -d kb_terminology \
  > kb-7-terminology/data/backups/kb7_terminology_$(date +%Y%m%d_%H%M%S).dump

# KB-6 formulary
docker exec kb6-postgres pg_dump -U kb6_admin -F c --compress=6 -d kb_formulary \
  > kb-6-formulary/data/backups/kb_formulary_$(date +%Y%m%d_%H%M%S).dump
```

Existing backup: [kb-7-terminology/data/backups/kb7_terminology_20260429_131550.dump](kb-7-terminology/data/backups/) (342 MB compressed, taken 29 Apr 2026).

To restore:
```bash
docker exec -i kb7-postgres dropdb -U postgres kb_terminology
docker exec -i kb7-postgres createdb -U postgres kb_terminology
docker exec -i kb7-postgres pg_restore -U postgres -d kb_terminology --no-owner -j 4 \
  < kb-7-terminology/data/backups/latest.dump
```

---

## What's NOT yet loaded (per Wave plan)

| Wave | Scope | Status | Blocker |
|---|---|---|---|
| Wave 1 | ICD-10-AM / ACHI codes | ⚠️ infra ready | IHACPA commercial license |
| Wave 2 | PBS authorities, restrictions, indications, ATC codes, prescriber types, criteria, parameters, programs (12 relational CSVs, 84,435 rows) | ✅ **loaded** (commit `<this commit>`) | — |
| Wave 3 | STOPP v3 + START v3 (120 entries) | ✅ **loaded** (commit 5c0eda39) | — |
| Wave 3 | AU APINCHs (high-alert) + TGA blackbox + TGA pregnancy (140 entries) | ✅ **loaded** | — |
| Wave 3 | Australian PIMs 2024 (Wang IMJ) | ✅ **loaded** (19 entries, manually curated from PDF, criteria re-phrased) | — |
| Wave 3 | AGS Beers 2023 | ✅ **loaded** (57 entries) | — |
| Wave 4 | ACB Scale (Boustani-derived, 56 entries) | ✅ **loaded** | — |
| Wave 4 | Drug Burden Index (DBI) weights | 🚫 **deferred** — procurement blocked, NO synthetic weights | Need ONE of: Hilmer 2007 JAMA supp / Monash CMUS CSV / Kouladjian 2014 (DOI 10.2147/CIA.S66660 — open access). When obtained, schema scaffold per [runbook](kb-4-patient-safety/scripts/README_AU_PIMS_DBI_PROCUREMENT.md) §"Source 2". |
| Wave 4 | Anticholinergic Cognitive Burden (ACB) scores | ❌ not started | CSV load |
| Wave 5 | AMH Aged Care Companion | ❌ blocked | Commercial license |
| Wave 5 | eTG Geriatric | ❌ blocked | Commercial license |
| Wave 5 (alt) | **Australian Deprescribing Guideline 2025 (UWA)** — v2-introduced free alternative to AMH | ✅ **PDFs landed (5 documents, 16 MB)** — DRAFT in public consultation | deprescribing.com / UWA — see [wave6 MANIFEST](kb-3-guidelines/knowledge/au/wave6/MANIFEST.md) §"Australian Deprescribing Guideline 2025" |
| Wave 6 | Heart Foundation (11 PDFs), ADS-ADEA T2D algorithm 2025 (2 PDFs), KHA-CARI KDIGO commentaries (5 PDFs) | ⏳ **PDFs downloaded — Pipeline 2 extraction next** | 18 PDFs / ~16 MB landed at [kb-3-guidelines/knowledge/au/wave6/](kb-3-guidelines/knowledge/au/wave6/MANIFEST.md), gitignored |
| Wave 6 | RANZCP Clinical Practice Guidelines (9 PDFs, 30 MB) | ✅ **landed via Playwright** | BPSD + Mood Disorders + Acute Pain (ANZCA APMSE5) + Opioids (RACP) + Physical Health + Valproate (BAP) + ECT + Off-label + BZD PPGs at [ranzcp_psych/](kb-3-guidelines/knowledge/au/wave6/ranzcp_psych/PROCUREMENT.md). 7 items remain not-downloaded (rescinded or JS-rendered) |
| Wave 6 | ACSQHC Clinical Care Standards + Med Mgmt Transitions Stewardship Framework 2024 (9 PDFs, 27.4 MB) | ✅ **landed via Playwright** | 6 CCSs (AMS, Delirium, Hip Fracture, Opioid, Psychotropic-CDI, VTE) + 1 framework + 2 supporting guides at [acsqhc_ams/](kb-3-guidelines/knowledge/au/wave6/acsqhc_ams/). Direct curl returns 000 (TLS block); Playwright's Chromium TLS context bypasses it. Recipe at [PROCUREMENT.md](kb-3-guidelines/knowledge/au/wave6/acsqhc_ams/PROCUREMENT.md) |
| Wave 6 | NPS MedicineWise deprescribing algorithms | ⏳ legacy procurement needed | Site retired Q4 2023, content moved to ACSQHC — try web.archive.org per MANIFEST |
| Wave 6 | Standard 5 + Royal Commission | ⚠️ partial | Manual ingestion |
| **Wave 6 / TGA** | **TGA Product Information catalog (4,298 PIs discovered)** | ✅ **catalog discovered + 64 PIs downloaded (37 MB) for top metformin/atorvastatin/apixaban brands** | Scraper at [scripts/tga_pi_scraper.py](kb-3-guidelines/scripts/tga_pi_scraper.py); watchlist at [tga_pi/top_racf_drugs.yaml](kb-3-guidelines/knowledge/au/tga_pi/top_racf_drugs.yaml) — 1,460 PIs match top-RACF watchlist (34% of catalog), full download takes ~25 min |
| **Wave 6 / KB-13** | **PHARMA-Care 5-domain placeholder + AU QI Program 11 indicators** | ✅ **16 measures seeded in KB-13** | Migration `kb-13-quality-measures/migrations/002_au_indicators_seed.sql` — QI Program rows are active; PHARMA-Care rows are placeholder pending pilot publication |

---

## Branch state

All AU work lives on branch `feature/v4-clinical-gaps`. Recent commits (most recent first):

```
5ae00983  feat(kb6-au): real PBS Schedule load (6,935 items from April 2026 CSV bundle)
433d1d62  feat(kb6-au): PBS schema + loader (Wave 2)
13d305ee  feat(kb-services): Pipeline 2 multi-layer loader (L1/L2/L2.5/L4 -> KB-3)
85aaf572  feat(kb-services): L3 ADA fact pipeline -> typed clinical tables
070c5447  fix(kb-services): bring KB-8/9/13/16/23 to healthy startup
9b8394a1  feat(kb7-au): cross-KB code-resolution validator (Wave 1 exit check)
f2f80232  feat(kb7-au): /v1/au/* HTTP routes + service deploy on port 8095
eeb10808  feat(kb7-au): ICD-10-AM/ACHI loader infrastructure (Wave 1 Phase B1)
e545b923  feat(kb7-au): SNOMED CT-AU + AMT bulk-load infrastructure (Wave 1 KB-7)
```

---

## Refresh cadence

| Source | Cadence | Method |
|---|---|---|
| SNOMED CT-AU + AMT (NCTS) | Monthly | Re-run `download_snomed_au_local.py` + loader; UPSERT replaces |
| PBS Schedule | Monthly (1st of month) | Re-run `load_pbs.py` against new month's CSV |
| ICD-10-AM | Annual (typically July) | Re-run `load_icd10am.py` once IHACPA file is updated |
| KDIGO L3 facts | When Pipeline 2 reruns | Re-run staging + extract scripts |

The loader scripts are designed to support incremental refresh — UPSERT semantics on primary keys mean re-running the same source file is idempotent, and re-running with a newer source replaces existing rows in place.

---

## Cross-KB conventions

When data in one KB references data in another:

| From | Foreign key column | References | Cross-DB |
|---|---|---|---|
| `kb6_pbs_items` | `amt_mp_sctid`, `amt_mpuu_sctid`, `amt_tpp_sctid`, `amt_ctpp_sctid` | `kb7_amt_pack.*` | yes (soft FK, app-level join) |
| `kb6_pbs_items` | `rxnorm_code` | `concepts_rxnorm.code` (KB-7) or `drug_rules.rxnorm_code` (KB-1) | yes |
| `kb_l3_staging` (each KB) | `resolved_rxcui` | RxNav | external |
| KB-1/4/16/20 typed tables | `rxnorm_code` | `concepts_rxnorm` (KB-7) | yes |
| KB-4 `kb4_l3_safety_facts` | `condition_codes`, `snomed_codes` | KB-7 `kb7_snomed_concept` (when populated) | yes |

The platform uses **soft FKs** (uuid/int columns without database-level FK constraints) for cross-DB references because each KB DB is isolated. App code is responsible for the join. A future cross-KB validator could automate consistency checks (see `scripts/validate_kb_codes.py`).

---

## Where to find more

| Topic | Path |
|---|---|
| Layer 1 source spec | [Layer1_Australian_Aged_Care_Implementation_Guidelines.md](../../../Layer1_Australian_Aged_Care_Implementation_Guidelines.md) |
| Gap audit | [claudedocs/audits/Layer1_AU_AgedCare_Codebase_Gap_Audit.md](../../../claudedocs/audits/Layer1_AU_AgedCare_Codebase_Gap_Audit.md) |
| KB-7 SNOMED + AMT | this README + scripts in [kb-7-terminology/scripts/](kb-7-terminology/scripts/) |
| KB-7 ICD-10-AM | [kb-7-terminology/scripts/README_ICD10AM.md](kb-7-terminology/scripts/README_ICD10AM.md) |
| KB-6 PBS | [kb-6-formulary/scripts/README_PBS.md](kb-6-formulary/scripts/README_PBS.md) |
| Per-service docs | [`kb-N-name/README.md`](.) where it exists |
| Tier-5 AU CQL adapters (existing platform code) | [vaidshala/clinical-knowledge-core/tier-5-regional-adapters/AU/](../../../vaidshala/clinical-knowledge-core/tier-5-regional-adapters/AU/) |

---

## Operational playbook

### Starting from scratch (new dev environment)

1. Start KB containers: `docker compose -f docker-compose.kb-only.yml up -d` (or per-KB composes)
2. Apply migrations (each KB has its own `migrations/` dir)
3. Run loaders in dependency order:
   - KB-7 SNOMED + AMT (foundation)
   - KB-6 PBS (depends on AMT for cross-references)
   - KB-1/4/16/20 L3 staging + extract (depends on KB-7 for RxCUI lookups)
   - KB-3 Pipeline 2 layers (depends on nothing else)

### Diagnosing missing AU data

1. Check service health: `docker ps --filter 'name=kb' --filter 'health=unhealthy'`
2. Check row counts: queries above
3. Check load logs: each KB has a `*_load_log` table tracking each load run

### Adding a new AU data source

1. Create migration in the relevant KB's `migrations/` dir (next available number)
2. Create loader script in that KB's `scripts/` dir following the existing pattern
3. Add a row to the inventory table at the top of this README
4. Document in this README's "Where to find more" section

## V5 Feature Flags

Pipeline 1 supports additive V5 subsystems controlled by feature flags. All V5 features are **off by default** — V4 output is byte-identical when no V5 flags are set.

### Available flags

| Flag | Env var | Profile key | Subsystem |
|------|---------|-------------|-----------|
| Bbox Provenance | `V5_BBOX_PROVENANCE=1` | `v5_features.bbox_provenance: true` | Per-channel attribution + bbox in every merged span |

### Enabling via environment variable (RunPod / CLI)

```bash
export V5_BBOX_PROVENANCE=1
python3 data/run_pipeline_targeted.py --pipeline 1 --guideline heart_foundation_au_2025 --source acs-hcp-summary --l1 monkeyocr --target-kb all
```

### Enabling via guideline profile (YAML)

Add to your guideline profile YAML (e.g. `data/profiles/heart_foundation_au_2025.yaml`):

```yaml
v5_features:
  bbox_provenance: true
```

### Disabling all V5 features

```bash
export V5_DISABLE_ALL=1
```

### Verifying V5 output

After a run with `V5_BBOX_PROVENANCE=1`:

```bash
# Check bbox coverage metric
python3 data/v5_metrics.py data/output/v4/job_monkeyocr_*/

# Inspect merged spans JSON directly
python3 -c "
import json; spans = json.load(open('data/output/v4/job_monkeyocr_<TIMESTAMP>/merged_spans.json'))
v5 = [s for s in spans if s.get('channel_provenance')]
print(f'{len(v5)}/{len(spans)} spans have bbox provenance')
print('Sample:', json.dumps(v5[0]['channel_provenance'][0], indent=2))
"
```

### KB-0 GCP verification

After `push_to_kb0_gcp.py` with V5 enabled, migration 009 adds `provenance_v5 JSONB` to `l2_merged_spans`:

```sql
SELECT COUNT(*) FROM l2_merged_spans WHERE provenance_v5 IS NOT NULL;
```

See `data/RUNPOD_SMOKE_V5.md` for the full end-to-end smoke checklist.
