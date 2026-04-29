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
| **KB-3** guidelines | Pipeline 2 layered spans / sections / tree / corrections | 11,873 | KDIGO 2022 Pipeline 2 | ✅ live |
| **KB-1** drug rules | KDIGO L3 dosing facts (typed `drug_rules`) | 37 | KDIGO 2022 L3 | ✅ live |
| **KB-1** drug rules | KDIGO L3 staging | 64 | KDIGO 2022 L3 | ✅ live |
| **KB-4** patient safety | KDIGO L3 typed safety facts (`kb4_l3_safety_facts`) | 170 | KDIGO 2022 L3 | ✅ live |
| **KB-4** patient safety | KDIGO L3 staging | 64 | KDIGO 2022 L3 | ✅ live |
| **KB-16** lab interp | KDIGO L3 typed lab requirements (`kb16_l3_lab_requirements`) | 97 | KDIGO 2022 L3 | ✅ live |
| **KB-16** lab interp | KDIGO L3 staging | 63 | KDIGO 2022 L3 | ✅ live |
| **KB-20** patient profile | KDIGO L3 typed ADR profiles (`adverse_reaction_profiles`) | 112 | KDIGO 2022 L3 | ✅ live |
| **KB-20** patient profile | KDIGO L3 staging | 63 | KDIGO 2022 L3 | ✅ live |

**Totals:** 9.4M SNOMED-AU rows + 6.9k PBS items + 11.9k pipeline spans + 416 typed clinical facts = approx. **9.43M rows** across 5 separate KB DBs, all loaded fresh 28-29 April 2026.

---

## Where each AU dataset lives

Every KB service has its own Postgres DB on its own container. The table below is the address book.

| KB service | Container | Host port | DB | User | Password | Key AU tables |
|---|---|---|---|---|---|---|
| KB-7 terminology | `kb7-postgres` | **5457** | `kb_terminology` | `postgres` | `password` | `kb7_snomed_*`, `kb7_amt_pack`, `kb7_icd10am_*`, `kb7_achi_*` |
| KB-6 formulary | `kb6-postgres` | **5447** | `kb_formulary` | `kb6_admin` | `kb6_secure_password` | `kb6_pbs_items`, `kb6_pbs_authorities`, `kb6_pbs_indications`, `kb6_pbs_section_100`, `kb6_pbs_restrictions`, `kb6_pbs_prescriber_types`, `kb6_pbs_load_log` |
| KB-3 guidelines | `kb3-postgres` | **5435** | `kb3_guidelines` | `postgres` | `password` | `kb3_pipeline2_*` (jobs, raw_spans, merged_spans, sections, tree, rxnorm_corrections, validation_reports) |
| KB-1 drug-rules | `kb1-postgres` | **5481** | `kb1_drug_rules` | `kb1_user` | `kb1_password` | `kb_l3_staging`, `drug_rules` (filter `source_set_id LIKE 'L3:%'`) |
| KB-4 patient-safety | `kb4-patient-safety-postgres` | **5440** | `kb4_patient_safety` | `kb4_safety_user` | `kb4_safety_password` | `kb_l3_staging`, `kb4_l3_safety_facts` |
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
```
Details: [kb-6-formulary/scripts/README_PBS.md](kb-6-formulary/scripts/README_PBS.md)

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

### Cross-KB validator (Wave 1 exit check)

```bash
python3 scripts/validate_kb_codes.py
python3 scripts/validate_kb_codes.py --json   # CI-friendly
```

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
| Wave 2 | PBS amt-items, criteria, indications, atc-codes (other CSVs in the bundle) | ⚠️ infra ready | Just need additional load runs |
| Wave 3 | STOPP/START v3 (190 criteria) | ❌ not started | Pipeline 2 re-run on source |
| Wave 3 | Australian PIMs 2024 | ❌ not started | Pipeline 2 re-run |
| Wave 3 | AGS Beers 2023 | ❌ not started | OHDSI verification + enrichment |
| Wave 4 | Drug Burden Index (DBI) weights | ❌ not started | Monash CSV load |
| Wave 4 | Anticholinergic Cognitive Burden (ACB) scores | ❌ not started | CSV load |
| Wave 5 | AMH Aged Care Companion | ❌ blocked | Commercial license |
| Wave 5 | eTG Geriatric | ❌ blocked | Commercial license |
| Wave 6 | Heart Foundation, ADS-ADEA, KHA-CARI, RANZCP, ACSQHC AMS | ❌ not started | Pipeline 2 extraction |
| Wave 6 | Standard 5 + QI Program + Royal Commission | ⚠️ partial | Manual ingestion |

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
