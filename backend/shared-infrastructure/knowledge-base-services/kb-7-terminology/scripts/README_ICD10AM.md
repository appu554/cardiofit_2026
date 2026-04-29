# ICD-10-AM / ACHI Loading — Operational Runbook

ICD-10-AM (Australian Modification of ICD-10) and ACHI (Australian
Classification of Health Interventions) are licensed by **IHACPA** /
**ACCD** — **not free**, separate institutional account from NCTS.

This directory contains everything needed to load these classifications
into KB-7 Postgres once you have access.

## Status as of 2026-04-29

- ✅ **Schema migrations** ready ([`017_icd10am_schema.sql`](../migrations/017_icd10am_schema.sql), [`018_icd10am_indexes.sql`](../migrations/018_icd10am_indexes.sql))
- ✅ **File-based loader** ready ([`load_icd10am.py`](load_icd10am.py)) — runs against any local files
- ⚠️ **API downloader** SKELETON ([`download_icd10am_local.py`](download_icd10am_local.py)) — URL pattern is speculative until IHACPA API documentation is obtained
- ❌ **No data loaded** — pending IHACPA license procurement and credential provisioning

## Two run paths

### Path A — Manual download from portal *(recommended; works today)*

The IHACPA portal at https://www.ihacpa.gov.au is the canonical
distribution point. After login:

1. Navigate to ICD-10-AM/ACHI/ACS download page
2. Download the latest edition (currently 12th edition, July 2024)
3. Extract the ZIP into:

   ```
   data/icd10am/12th/
     icd10am_tabular.xml      # ICD-10-AM disease codes (Tabular List)
     icd10am_index.csv        # Alphabetic Index
     achi_tabular.xml         # ACHI procedure codes (Tabular List)
     achi_index.csv           # ACHI Alphabetic Index
   ```

4. Run the loader:

   ```bash
   cd backend/shared-infrastructure/knowledge-base-services/kb-7-terminology
   python3 scripts/load_icd10am.py \
     --edition "12th edition" \
     --release-date 2024-07-01 \
     --tabular   data/icd10am/12th/icd10am_tabular.xml \
     --index     data/icd10am/12th/icd10am_index.csv \
     --achi-tab  data/icd10am/12th/achi_tabular.xml \
     --achi-idx  data/icd10am/12th/achi_index.csv
   ```

5. Validate parsing first with `--dry-run` if desired:

   ```bash
   python3 scripts/load_icd10am.py --tabular data/icd10am/12th/icd10am_tabular.xml --dry-run
   ```

### Path B — Programmatic download via API *(once IHACPA API is documented)*

1. Create a gitignored env file with your IHACPA credentials:

   ```
   # .env.ihacpa.local
   IHACPA_BASE_URL=https://api.ihacpa.gov.au
   IHACPA_INSTITUTION_ID=<your-institution-id>
   IHACPA_ACCESS_KEY=<your-api-key>
   # Optional mTLS:
   # IHACPA_CERT_PATH=/path/to/client.crt
   # IHACPA_KEY_PATH=/path/to/client.key
   ```

   The `.env.ihacpa.local` filename matches the existing gitignore
   pattern (`.env.*.local`) so it won't accidentally be committed.

2. Probe auth (read-only, no download):

   ```bash
   set -a && source .env.ihacpa.local && set +a
   python3 scripts/download_icd10am_local.py --probe
   ```

3. Run the download:

   ```bash
   python3 scripts/download_icd10am_local.py --edition 12th
   ```

4. Run the loader (same as Path A from step 4 onwards).

5. **Always delete the env file after use:**

   ```bash
   rm .env.ihacpa.local
   ```

## What gets loaded

| Postgres table | Source file | Typical row count |
|---|---|---|
| `kb7_icd10am_chapter` | tabular XML | ~22 |
| `kb7_icd10am_block` | tabular XML | ~280 |
| `kb7_icd10am_code` | tabular XML | ~22,000 |
| `kb7_icd10am_index` | index CSV | ~50,000 |
| `kb7_achi_block` | ACHI tabular XML | ~150 |
| `kb7_achi_code` | ACHI tabular XML | ~6,000 |
| `kb7_achi_index` | ACHI index CSV | ~30,000 |

(Counts approximate; actual values depend on edition.)

## Why XML format may need parser tweaks

The parser in `load_icd10am.py` assumes a "typical ACCD" XML structure:

```xml
<icd10am>
  <chapter number="1" title="..." range="A00-B99">
    <block code="A00-A09" title="...">
      <category code="A00" title="...">
        <code value="A00.0" desc="..."/>
      </category>
    </block>
  </chapter>
</icd10am>
```

If IHACPA's actual XML uses different element/attribute names (e.g.,
`<Chapter ChapterNumber="1">` instead of `<chapter number="1">`), the
parser will need light adjustment. The `--dry-run` mode prints row
counts after parsing — if counts are 0 when the file is non-empty,
the element/attribute names need to be remapped. Fix sites:

- `parse_icd10am_tabular()` — `chap.iter("chapter")`, `blk.iter("block")`, etc.
- `parse_achi_tabular()` — same pattern

## Safety

- `data/icd10am/` is covered by the existing gitignore rule for
  `kb-7-terminology/data/` — large XML/CSV files don't accidentally
  get committed.
- All credentials live only in `.env.ihacpa.local` (gitignored) and
  are deleted after a successful run, mirroring the NCTS pattern.
- The downloader will exit with a clear error if credentials are
  missing rather than silently doing nothing.

## Known unknowns

These will need real IHACPA access or documentation to resolve:

1. **Exact distribution file format.** The XML element/attribute names
   used in the parser are best-guess; the real distribution may use
   different naming conventions.
2. **API endpoint URL pattern.** The URL template
   `<base>/distribution/<edition>/<package>` in `download_icd10am_local.py`
   is speculative.
3. **Auth header convention.** The downloader tries Bearer +
   X-Institution-ID + X-Access-Key in parallel. Real IHACPA may use
   a single, different scheme.
4. **mTLS requirement.** The Go `IHACPAConfig` struct includes cert
   paths suggesting some IHACPA endpoints might require mutual TLS;
   needs confirmation.

When IHACPA access lands, run with `--probe` first to surface auth
behaviour and adjust accordingly. If the API path doesn't pan out,
Path A (manual portal download) is the production-acceptable fallback.
