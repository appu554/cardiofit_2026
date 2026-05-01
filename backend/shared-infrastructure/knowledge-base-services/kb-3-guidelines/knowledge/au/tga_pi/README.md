# TGA Product Information (PI) — AU regulatory drug labels

**Layer 1 spec source:** [Source R, Layer 1 Implementation Guidelines §2.5](../../../../../../../Layer1_Australian_Aged_Care_Implementation_Guidelines.md)
**Status:** ✅ Scraper operational. Catalog discovered (4,298 PIs). Watchlist matches 1,460 (34%) of catalog.

## What this is

The Therapeutic Goods Administration (TGA) is Australia's drug regulator (FDA equivalent). They publish **Product Information** (PI) PDFs for every TGA-registered drug — the AU regulatory equivalent of US SPL or UK SmPC.

PI documents contain:
- Registered indications
- Contraindications
- Warnings + precautions
- Drug-drug interactions
- Adverse reactions (ADRs)
- Dosing + administration
- Pregnancy/lactation categorisation

These feed **KB-1** (dosing — primary AU baseline), **KB-4** (contraindications/warnings), **KB-5** (per-drug DDI sections), **KB-20** (ADR profiles).

## Why TGA was hard

The v1.0 Layer 1 spec flagged TGA as "the most engineering-heavy piece of Layer 1" because:

1. **No clean API** (unlike FDA DailyMed which has bulk SPL XML download)
2. **Lotus Notes backend** (eBS PICMI repository runs on IBM Lotus Notes/Domino)
3. **JS-driven license gate** for PDF access — disclaimer + cookie required
4. **Non-obvious URL params** — `q=` doesn't filter; `k=` does (revealed by reading `displayCategory()` JS in the form HTML)
5. **No site-wide search** — must browse by category letter

## How the scraper solves it

`scripts/tga_pi_scraper.py` reverse-engineers all four blockers:

| Stage | Endpoint | What it does |
|-------|----------|--------------|
| Discovery | `picmirepository.nsf/PICMI?OpenForm&t=PI&k=<LETTER>&r=/` | Returns up to ~285 PI rows for that letter; iterates 0-9 + A-Z = 36 calls = ~30 sec for full catalog |
| License gate | `picmirepository.nsf/pdf?OpenAgent&id=<PI_ID>` | Returns disclaimer HTML; scraper extracts hidden `Remote_Addr` field |
| Cookie compute | `(in code)` | Builds `PICMIIAccept=<UTC YYYYMMDD><RemoteAddr without dots>` matching the JS `IAccept()` function |
| Download | Same URL with `&d=<cookie_value>` + `Cookie: PICMIIAccept=<value>` | Returns the actual PDF (typically 30-150 pages, 50-700 KB) |

## Usage

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines

# Step 1 — full catalog discovery (~30 sec)
python3 scripts/tga_pi_scraper.py discover

# Step 2a — download by watchlist (recommended)
python3 scripts/tga_pi_scraper.py download --watchlist knowledge/au/tga_pi/top_racf_drugs.yaml

# Step 2b — download single PI by ID (for spot-checks)
python3 scripts/tga_pi_scraper.py download --pi-id CP-2018-PI-02461-1
```

## Storage

```
knowledge/au/tga_pi/
├── README.md                  # this file
├── .gitignore                 # excludes *.pdf, cache/
├── tga_pi_index.json          # discovery output (committed — useful catalog reference)
├── top_racf_drugs.yaml        # ~140 INN watchlist (committed)
├── docs/                      # downloaded PI PDFs (gitignored)
└── cache/                     # disclaimer-page cache (gitignored)
```

## Catalog stats (2026-04-30)

- **4,298 unique PI documents** in the TGA eBS PICMI repository
- 36 category letters crawled (0-9 + A-Z)
- 4,362 row-views deduplicated to 4,298 unique
- ~30 sec to refresh catalog via `discover`

### Sample drug coverage from full catalog

| INN | PI documents (multiple brands per drug) |
|-----|-----------------------------------------|
| metformin | 38 |
| atorvastatin | 20 |
| apixaban | 5 |
| pantoprazole | 29 |
| amlodipine | 45 |
| paracetamol | 61 |
| oxycodone | 17 |
| levothyroxine | 9 |

## Watchlist (`top_racf_drugs.yaml`)

~140 INN names spanning:
- Cardiovascular (29 drugs)
- Antiplatelet/Anticoagulant (8)
- Lipid-lowering (5)
- Diabetes (17)
- GI/PPI/antiemetic (10)
- Pain/opioid (12)
- Respiratory (9)
- Antidepressant/anxiolytic (15)
- Antipsychotic (5)
- Anti-dementia (4)
- Bone health (7)
- Antibiotic (8)
- Bladder/BPH (10)
- Other (~10)

**Watchlist matches 1,460 of 4,298 PIs (34%) of the AU catalog.**

## Copyright posture

- TGA PIs are **Crown copyright** (Australian Government — Commonwealth of Australia).
- Permitted use: clinical use, reference, and curation of *facts* (registered indications, contraindications, dose ranges, ADRs).
- Not permitted: redistribution of the verbatim PI prose.
- Mitigation: Same as Wang 2024 / KHA-CARI / ADG 2025 sources — PDFs gitignored; only structured YAMLs/JSONs containing **re-phrased** facts are committed.

## Pipeline 2 / extraction next steps

The downloaded PI PDFs feed Pipeline 2 (or the new V4 multi-channel Pipeline 1) for fact extraction. Each PI yields ~10-30 structured facts spanning:
- KB-1 dosing/RenalAdjustment/HepaticAdjustment
- KB-4 ContraindicationFact / WarningFact
- KB-5 DrugInteraction (severity/effect/management)
- KB-20 AdverseReactionProfile

Estimated extraction cost (per v1.0 spec):
- TGA PI top 100: ~$2-3 API
- TGA CMI (Consumer Medicine Information, family-facing) for same: ~$1 API

## Limitations + caveats

1. **Search semantics are fuzzy.** `k=M` returns trade names starting with M *and* trade names containing M somewhere. Watchlist matching is on `active_ingredients` field which is more reliable than trade name.
2. **Multi-brand drugs produce many PIs.** Metformin alone returns 38 PIs (one per brand); curation should prioritize generic/canonical PI per ingredient.
3. **Disclaimer cookie has TTL.** The cookie value embeds today's UTC date; downloads must complete on the same UTC day they started. The scraper handles this transparently (re-fetches the disclaimer per call).
4. **Be polite.** Default `--sleep 0.4` between requests. TGA eBS has no documented rate limit but a full top-RACF download is ~25 min at default pace; respect their infrastructure.
5. ~~CMI is not yet covered.~~ **Resolved 2026-04-30:** scraper extended with `--doc-type CMI`. CMI catalog (4,508 unique documents) discovered. See "CMI extension" section below.
6. **Updates not auto-detected.** TGA updates PIs/CMIs continuously; the index is a point-in-time snapshot. Re-run `discover` periodically to refresh.

---

## CMI extension (2026-04-30)

The scraper now supports both PI (Product Information — clinician-facing) and CMI (Consumer Medicine Information — patient/family-facing plain-language summary). v1.0 Layer 1 spec routes CMI to KB-20 family-education layer + Activity 7 of the ACOP workflow.

### Same access pattern, different `t=` parameter

CMI uses the identical eBS PICMI infrastructure:
- Listing: `t=CMI&k=<LETTER>` → returns `<a ...>CMI</a>` row links
- ID format: `CP-YYYY-CMI-NNNNN-V` (vs `CP-YYYY-PI-NNNNN-V`)
- Disclaimer-cookie flow: identical (same `Remote_Addr` field + `PICMIIAccept` cookie)
- PDFs: also typically 30-150 pages, 50-500 KB; written in plain language for residents/families

### Storage separation

```
knowledge/au/tga_pi/
├── tga_pi_index.json      # 4,298 PIs (clinician-facing)
├── tga_cmi_index.json     # 4,508 CMIs (patient-facing)
├── docs/                  # PI PDFs (gitignored)
└── cmi_docs/              # CMI PDFs (gitignored)
```

### Usage

```bash
# Discover CMI catalog (~30 sec for 36 letters)
python3 scripts/tga_pi_scraper.py discover --doc-type CMI

# Download single CMI by ID
python3 scripts/tga_pi_scraper.py download --doc-type CMI --cmi-id CP-2013-CMI-01689-1

# Download by watchlist (same INN watchlist works for both PI and CMI)
python3 scripts/tga_pi_scraper.py download --doc-type CMI \
    --watchlist knowledge/au/tga_pi/top_racf_drugs.yaml
```

### CMI catalog stats (2026-04-30)

| Metric | Value |
|--------|------:|
| Unique CMI documents | 4,508 |
| Letters crawled | 36 (0-9 + A-Z) |

| INN | PI count | CMI count |
|-----|---------:|----------:|
| metformin | 38 | 38 |
| atorvastatin | 20 | 19 |
| paracetamol | 61 | 66 |
| apixaban | 5 | 8 |
| amlodipine | 45 | 49 |

### Why CMI matters for ACOP

Per Layer 1 spec Activity 7 (resident/family medication education): "The ACOP types a drug name + a clinical state and gets a plain-language summary printable for residents/families."

CMI provides:
- TGA-registered patient-facing language (regulator-approved)
- Side-effect explanation in non-technical terms
- "What to do if you miss a dose" guidance
- Pregnancy/breastfeeding notes for the rare younger residents

These feed KB-20 family-education layer + KB-23 Decision Card "family-facing template" mode.

## Verification queries

After download, verify pipe-end functionality:

```python
import json
data = json.load(open("knowledge/au/tga_pi/tga_pi_index.json"))
print(f"Catalog: {data['total_unique_pis']} unique PIs")

import os
n = len([f for f in os.listdir("knowledge/au/tga_pi/docs") if f.endswith(".pdf")])
print(f"Local PDFs: {n}")
```

Also, confirm a sample PDF is a real PI not a disclaimer page:

```bash
file knowledge/au/tga_pi/docs/CP-2018-PI-02461-1__*.pdf
# Expected: PDF document, version 1.X, NN pages
```
