# PBS (Australian Pharmaceutical Benefits Scheme) Loading — Operational Runbook

PBS data is **free and public domain** (Australian Government, CC-BY).
Distribution URLs migrate between releases — this directory contains
a **file-based loader** that parses any PBS XML or CSV extract you
have on disk. Wave 2 of the AU Aged Care Layer 1 plan.

## Status as of 2026-04-29

- ✅ **Schema migrations** ready ([`005_pbs_au_schema.sql`](../migrations/005_pbs_au_schema.sql), [`006_pbs_au_indexes.sql`](../migrations/006_pbs_au_indexes.sql)) — applied
- ✅ **Loader script** ready ([`load_pbs.py`](load_pbs.py)) — XML + CSV
- ✅ **Sample data** ([`sample_pbs_extract.xml`](sample_pbs_extract.xml)) — verified end-to-end
- ❌ **Live PBS data not yet loaded** — pending acquisition of monthly extract

## Tables (in `kb_formulary` on `kb6-postgres:5447`)

| Table | One row per | Cross-references |
|---|---|---|
| `kb6_pbs_items` | PBS item code (e.g. "1574H") | `amt_*_sctid` -> KB-7 AMT, `rxnorm_code` -> KB-1 |
| `kb6_pbs_authorities` | (item, authority requirement) | FK -> kb6_pbs_items |
| `kb6_pbs_restrictions` | (item, clinical criterion) | FK -> kb6_pbs_items |
| `kb6_pbs_prescriber_types` | (item, prescriber type) | FK -> kb6_pbs_items |
| `kb6_pbs_section_100` | item (1:1 — Section 100 detail) | FK -> kb6_pbs_items |
| `kb6_pbs_indications` | (item, approved indication) | FK -> kb6_pbs_items |
| `kb6_pbs_load_log` | load run | audit |

## Where to obtain real PBS data

The PBS Schedule is published monthly. **Public, no auth required**
for the file downloads (the new PBS Data API itself does require a
key, but the CSV bundle is freely available).

### **Recommended: PBS API CSV bundle** ✅

```
https://www.pbs.gov.au/downloads/<YYYY>/<MM>/<YYYY-MM-01>-PBS-API-CSV-files.zip
```

Example (April 2026):
```
https://www.pbs.gov.au/downloads/2026/04/2026-04-01-PBS-API-CSV-files.zip
```

This is a **4-5 MB ZIP** containing 22 normalized CSV tables:
`items.csv` (the primary table), `amt-items.csv` (AMT cross-reference),
`atc-codes.csv`, `criteria.csv`, `indications.csv`, `item-restriction-relationships.csv`,
plus several smaller relationship/lookup tables. **All free, no auth.**

### Other formats (XML deprecated 1 May 2026)

```
https://www.pbs.gov.au/downloads/<YYYY>/<MM>/<DATE>-XML-V3.zip            (deprecated)
https://www.pbs.gov.au/downloads/<YYYY>/<MM>/<DATE>-V2-down-converted.zip (legacy)
```

### PBS API (REST, requires key)

```
https://api.pbs.gov.au/api/v3/...
```
Returns 401 without an API key. Email HPP Support to register.

### Discovery / search routes

- https://www.pbs.gov.au/info/browse/download — official download page
- https://data.gov.au/data/dataset?q=pbs+schedule — open data CKAN
  (note: tends to host historical aggregates, not the current monthly schedule)

## Run

### Once you have a PBS XML or CSV file on disk

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-6-formulary

# Validate parse first (no DB write)
python3 scripts/load_pbs.py \
    --xml /path/to/pbs-extract.xml \
    --schedule-date 2026-04-01 \
    --dry-run

# Real load
python3 scripts/load_pbs.py \
    --xml /path/to/pbs-extract.xml \
    --schedule-date 2026-04-01

# CSV variant
python3 scripts/load_pbs.py \
    --csv /path/to/PBS_LIMITS.csv \
    --schedule-date 2026-04-01
```

### Sample (synthetic) data — for testing the loader

```bash
python3 scripts/load_pbs.py \
    --xml scripts/sample_pbs_extract.xml \
    --schedule-date 2026-04-01
```

The sample contains 5 items covering all 5 main schedule classifications
(General, Streamlined Authority, Section 100 HSD, Restricted, Palliative).

## Idempotency

The loader uses `INSERT ... ON CONFLICT (pbs_code) DO UPDATE` for the
items table; child tables (authorities, restrictions, prescriber_types,
indications, section_100) get DELETE-then-INSERT per item. Safe to
re-run the same file multiple times — produces stable end-state.

When loading a NEW month's extract, items that no longer exist remain
in the DB with their old `loaded_at`. To clean: filter by `loaded_at`
or `schedule_publish_date` after each run.

## Sample queries the KB-6 service can run after loading

```sql
-- All ACOP-relevant Authority Required items
SELECT pbs_code, drug_name, schedule_section
FROM kb6_pbs_items
WHERE is_authority_required = TRUE OR is_streamlined = TRUE;

-- Section 100 HSD drugs (special supply pathways for aged care)
SELECT i.pbs_code, i.drug_name, s.section_100_type, s.supply_pathway
FROM kb6_pbs_items i
JOIN kb6_pbs_section_100 s USING (pbs_code)
WHERE s.section_100_type = 'HSD';

-- Find all PBS items for a specific AMT MP (cross-KB)
SELECT pbs_code, drug_name, schedule_section
FROM kb6_pbs_items
WHERE amt_mp_sctid = 6809011000036107;  -- metformin

-- Authority codes + indications for a drug
SELECT i.drug_name, a.authority_code, ind.indication_text
FROM kb6_pbs_items i
LEFT JOIN kb6_pbs_authorities a USING (pbs_code)
LEFT JOIN kb6_pbs_indications ind USING (pbs_code)
WHERE lower(i.drug_name) LIKE '%empagliflozin%';
```

## XML format compatibility

The parser is tolerant of common element/attribute name variants used
in different PBS extract generations:

| Field | Recognised attribute names |
|---|---|
| Item code | `code`, `pbs_code`, `ItemCode`, `PBSCode`, `pbsCode` |
| Drug name | `drug_name`, `DrugName`, `name`, `Name`, `GenericName`, `li_drug_name` |
| Schedule section | `schedule_section`, `ScheduleSection`, `Schedule`, `Section`, `ScheduleSubsection`, `li_schedule_section` |
| Max quantity | `max_quantity`, `MaxQuantity`, `MaxQty`, `li_max_qty` |
| Form | `form`, `Form`, `DoseForm`, `li_form` |
| AMT codes | `amt_mp_sctid`, `AMTMpId`, `MP_SCTID` (etc. for MPUU/TPP/CTPP) |

For unfamiliar PBS XML schemas, run with `--dry-run` first to see how
many items the parser found. If 0, the element/attribute names need
adjustment in [`load_pbs.py`](load_pbs.py) `_attr_or_text()` calls.

## Known limitations

- **Auto-download**: not implemented. PBS download URLs migrate too
  often to make a hardcoded fetcher reliable.
- **Pricing data** (dispensed price, manufacturer co-payment): not
  loaded. Audit recommended skipping unless ACOP requires it.
- **Concession types / RPBS** (Repatriation Pharmaceutical Benefits
  Scheme): not loaded.
- **AMT cross-reference**: fields exist in schema; populated only for
  PBS items whose XML includes AMT codes (most post-2010 items).
- **Streamlined vs Authority Required**: `is_streamlined` and
  `is_authority_required` are MUTUALLY EXCLUSIVE in this schema. A
  Streamlined item has `is_authority_required = FALSE`. ACOP queries
  that need either should use `WHERE is_authority_required OR is_streamlined`.
