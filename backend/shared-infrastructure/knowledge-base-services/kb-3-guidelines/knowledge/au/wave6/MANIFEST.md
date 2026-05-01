# Wave 6 — Australian peak-body guidelines download manifest

**Last updated:** 2026-04-29

This directory holds source PDFs from Australian peak-body guidelines for
ACOP clinical decision support. PDFs themselves are gitignored (copyright
applies to publisher prose); only this manifest + structured YAMLs we
curate from these PDFs are committed.

Storage convention:
```
wave6/
├── heart_foundation/    # Heart Foundation Australia
├── ads_adea_diabetes/   # Australian Diabetes Society + ADEA
├── nps_medicinewise/    # NPS MedicineWise (legacy — content moved to ACSQHC)
├── kha_cari_renal/      # KHA-CARI / Kidney Health Australia
├── ranzcp_psych/        # Royal AU/NZ College of Psychiatrists
├── acsqhc_ams/          # ACSQHC (Antimicrobial Stewardship + post-NPS content)
└── adg_2025_uwa/        # Australian Deprescribing Guideline 2025 (UWA — v2-introduced)
```

---

## ✅ Downloaded (23 documents, ~32.4 MB total)

### Heart Foundation Australia (11 PDFs, ~13.5 MB)

ACS (Acute Coronary Syndromes) guideline package:
- `acs/ACS-Guideline.pdf` (5.3 MB) — full guideline
- `acs/ACS-Reference-Guide-combined.pdf` (0.3 MB) — quick-reference for clinicians
- `acs/ACS-Guideline-summary-for-healthcare-professionals.pdf` (0.13 MB)
- `acs/ACS-Summary-of-recommendations.pdf` (0.15 MB)
- `acs/ACS-Guideline-Supplementary-material-A.pdf` (1.0 MB)
- `acs/ACS-Guideline-Supplementary-material-B.pdf` (0.78 MB)

CVD risk + hypertension + cholesterol:
- `hypertension/Guideline-for-assessing-and-managing-CVD-risk_20230522.pdf` (4.8 MB) — **2023 CVD risk assessment guideline (key)**
- `hf/Hypertension_Guidelines-2016_Presentation_.pdf` (0.6 MB)
- `hf/2026_HF_Cholesterol_Action_Plan.pdf` (0.14 MB)
- `hf/1225_CVD_risk_assessemnt_algorithm_f.pdf` (0.22 MB) — algorithm
- `hf/0323-Lipid-lowering-chart_A4_2pp__f.pdf` (0.07 MB) — lipid-lowering chart

Source pages probed:
- `https://www.heartfoundation.org.au/for-professionals` — HCP portal
- `https://www.heartfoundation.org.au/for-professionals/acs-guideline` — ACS landing
- `https://www.heartfoundation.org.au/for-professionals/hypertension` — hypertension landing

### KHA-CARI / Kidney Health Australia (5 PDFs, ~1.7 MB)

Renal guideline summaries:
- `AKI_Summary.pdf` (0.1 MB) — KHA-CARI adaptation of KDIGO 2012 AKI guideline
- `Heart_failure_May_2013_final.pdf` (0.86 MB) — HF in CKD (older but still authoritative)

KDIGO commentaries (highest-value — overlay AU context on the KDIGO guidelines we already have in KB-3):
- `kdigo_commentaries/Wallace-2026-CARI-Diabetes-Commentary.pdf` (0.5 MB) — **Diabetes in CKD (2026)**
- `kdigo_commentaries/Roberts_2014_BP_Commentary.pdf` (0.15 MB) — BP management in CKD
- `kdigo_commentaries/KDIGO_lipid_commentary.pdf` (0.11 MB) — Lipid management in CKD

Source: `https://www.cariguidelines.org/`

### ADS-ADEA Diabetes (2 PDFs, ~1.1 MB)

- `T2D-Treatment-Algorithm-2025.pdf` (0.24 MB) — **Australian T2D Glycaemic Management Algorithm 2025** (the canonical AU diabetes algorithm)
- `ADS_Position_Statement_v2.4.pdf` (0.83 MB) — ADS position statement v2.4

Source: `https://www.diabetessociety.com.au/guidelines/`

### Australian Deprescribing Guideline 2025 (UWA, 5 documents, ~16 MB) — NEW v2-introduced

**Status: DRAFTS in active public consultation phase (2025).** v2 Revision Mapping document framed this as "freely available, RACGP+ANZSGM endorsed" — that describes the *expected post-consultation* state. Currently downloadable from the project's consultation site.

- `ADG-2025-DRAFT-deprescribing-guidelines.pdf` (6.3 MB) — **the actual guideline (185 recommendations target)**
- `ADG-2025-DRAFT-technical-report.pdf` (4.2 MB) — evidence base + GRADE assessments
- `ADG-2025-DRAFT-administrative-report.pdf` (1.1 MB) — process/administrative
- `ADG-2025-DRAFT-dissemination-plan.pdf` (1.1 MB) — implementation/dissemination plan
- `ADG-2025-Guideline-draft-v4.1.docx` (3.3 MB) — DOCX version of guideline (preferred for parsing — preserves table structure better than PDF)

Source pages:
- `https://deprescribing.com/` — project homepage (UWA-led, consultation form at `https://uwa.qualtrics.com/jfe/form/SV_3l3eObHpqhna8lM`)
- `https://deprescribing.com/about/documents-for-public-consultation.html` — full document set

PDFs hosted on Dropbox public-share URLs (CDN-stable while consultation runs).

Why it matters per v2 Revision Mapping: ADG 2025 replaces what Wave-5 AMH Aged Care Companion was meant to provide for the deprescribing layer. Same legal posture as Wang 2024 — facts uncopyrightable, prose re-phrased before loading.

**Caveats for curation:**
- These are DRAFTS — final version may differ. Plan a re-fetch when consultation closes.
- 185 recommendations is the *target*; actual count in current draft to be verified after extraction.
- Endorsement status from RACGP/ANZSGM is *anticipated*, not confirmed in the draft documents.

---

## ⏳ Still needed — manual download required

### ACSQHC Clinical Care Standards (9 PDFs, ~27.4 MB) ✅ landed via Playwright (2026-04-30 evening)

The Australian Commission on Safety and Quality in Health Care (ACSQHC) publishes Clinical Care Standards (CCS) — concise definitions of the care a person should receive for a specific condition. Direct curl/wget returned 000 from this dev env (TLS/CDN block) but Playwright's Chromium TLS handshake completed successfully.

Clinical Care Standards (6 aged-care priority):
- `ccs/AMS-Clinical-Care-Standard-2020.pdf` (2.87 MB) — Antimicrobial Stewardship
- `ccs/Delirium-Clinical-Care-Standard-2021.pdf` (1.42 MB) — delirium screening/management
- `ccs/Hip-Fracture-Clinical-Care-Standard-2023.pdf` (3.07 MB) — fall outcome management
- `ccs/Opioid-Analgesic-Stewardship-in-Acute-Pain-CCS.pdf` (1.20 MB) — opioid stewardship
- `ccs/Psychotropic-Medicines-in-Cognitive-Disability-Impairment-CCS.pdf` (2.35 MB) — **directly maps to KB-13 AU-QI-06 antipsychotic indicator**
- `ccs/VTE-Prevention-Clinical-Care-Standard-2020.pdf` (7.11 MB)

Stewardship framework (the v2-introduced piece):
- `framework/Medication-Management-Transitions-Care-Stewardship-Framework-2024.pdf` (5.81 MB) ⭐

Supporting documents:
- `supporting/Cognitive-Impairment-User-Guide-2019.pdf` (2.94 MB)
- `supporting/National-Medication-Management-Plan-User-Guide-2021.pdf` (1.85 MB) — bonus structured medication-management template

Procurement recipe documented at `acsqhc_ams/PROCUREMENT.md` (Playwright fetch + base64 + Python decode pattern; reusable for any AU regulator site that's reachable in browser but blocked by curl from this dev env).

For NPS MedicineWise legacy deprescribing algorithms (the highest-value
content given Wave 5 is commercial-blocked):
1. The legacy NPS site redirects, but archived versions may still be on
   `web.archive.org` — search for "NPS MedicineWise deprescribing
   algorithm" to find PDF copies of:
   - PPI deprescribing algorithm
   - Benzodiazepine deprescribing algorithm
   - Antipsychotic deprescribing in dementia algorithm
   - Statin deprescribing in advanced age algorithm
2. Save to: `nps_medicinewise/`

### RANZCP Clinical Practice Guidelines (9 PDFs, ~30 MB) ✅ landed via Playwright (2026-04-30 evening)

Identical procurement pattern to ACSQHC: curl blocked, Playwright works. Detailed runbook at [ranzcp_psych/PROCUREMENT.md](ranzcp_psych/PROCUREMENT.md).

Highest-priority for aged care:
- `RANZCP-BPSD-Assessment-Management-NSWHealth-2022.pdf` (1.4 MB) — **antipsychotic-for-BPSD assessment + management**, RANZCP-endorsed NSW Health handbook
- `RANZCP-Mood-Disorders-CPG.pdf` (4.0 MB) — joint depression + bipolar CPG
- `RANZCP-Pain-Mgmt-Acute-ANZCA-APMSE5-2020.pdf` (22.9 MB) — Acute Pain Management Scientific Evidence 5th ed
- `RANZCP-Prescription-Opioids-RACP-Policy.pdf` (1.9 MB)
- `RANZCP-Physical-Health.pdf` (619 KB)
- `RANZCP-Valproate-Healthcare-BAP-PS04-2018.pdf` (270 KB)
- `RANZCP-ECT-Professional-Practice.pdf` (197 KB)
- `RANZCP-Off-Label-Prescribing-Psych.pdf` (185 KB) — PPG-4
- `RANZCP-Benzodiazepines-Psychiatric-Practice.pdf` (140 KB) — PPG-5

Not downloaded (rescinded or JS-rendered without static PDF link):
- Antipsychotics and dementia (PPG-10 — **rescinded pending review**)
- Older people psychiatric service delivery
- Deprescribing cholinesterase inhibitors + memantine
- Dementia (endorsed external — U Sydney CDPC HTML portal, not a single PDF)
- Care of sedated ABD patients · PTSD · Pharmacogenetics

### RANZCP — Royal AU/NZ College of Psychiatrists (legacy section)

**Status:** RANZCP's clinical-guidelines library page is dominated by
policy submissions (carers strategy, crimes bills, pre-budget submissions)
rather than CPGs. Their actual Clinical Practice Guidelines are
typically published in the *Australian and New Zealand Journal of
Psychiatry* (ANZJP) on SAGE Publications.

**To procure manually:**

1. Browse to: `https://www.ranzcp.org/clinical-guidelines-publications/clinical-guidelines-publications-library`
2. Use the search to find:
   - **RANZCP CPG for the Management of Schizophrenia and Related Disorders** — published in ANZJP, free version typically downloadable
   - **RANZCP CPG for Mood Disorders (Bipolar + MDD)**
   - **BPSD (Behavioural and Psychological Symptoms of Dementia) Clinical Practice Guideline** — **HIGHEST VALUE for aged care** (Royal Commission specifically called out antipsychotic over-prescribing)
3. Save to: `ranzcp_psych/`

Alternative: search SAGE journals at `journals.sagepub.com/home/anp` and
filter by "Clinical Practice Guidelines". You have institutional access
via the Wiley path used for Wang 2024.

---

## Curation pipeline (after PDFs land)

For each downloaded PDF, the workflow is:

1. **Run Pipeline 2 extraction** (same multi-stage process used for KDIGO 2022):
   - L1: raw spans extraction
   - L2: merged span deduplication
   - L2.5: section identification
   - L3: structured fact extraction (drug rules, safety facts, lab requirements, ADR profiles)
   - L4: corrections pass

2. **Stage to KB-3** via `kb-7-terminology/scripts/load_pipeline2_layers.py`

3. **Extract typed facts** to KB-1 / KB-4 / KB-16 / KB-20 via
   `kb-7-terminology/scripts/extract_l3_to_typed.py`

4. **Verify** via cross-KB validator
   (`kb-7-terminology/scripts/validate_kb_codes.py --rxnav`)

5. **Update README_AU.md** Wave 6 inventory

### Recommended priority order for Pipeline 2 runs

| # | Source | Document | Aged-care impact | Est. rules |
|---|--------|----------|------------------|-----------:|
| 1 | Heart Foundation | ACS Guideline | High (CV is #1 cause of death) | 60-90 |
| 2 | ADS-ADEA | T2D Glycaemic Management Algorithm 2025 | High (~25-30% diabetes prevalence) | 40-60 |
| 3 | KHA-CARI | Wallace 2026 Diabetes-in-CKD Commentary | High (CKD+T2D combo) | 20-30 |
| 4 | Heart Foundation | CVD Risk Guideline 2023 | Medium-high | 30-50 |
| 5 | KHA-CARI | KDIGO BP commentary | Medium | 15-25 |
| 6 | KHA-CARI | KDIGO Lipid commentary | Medium | 15-20 |
| 7 | Heart Foundation | Cholesterol Action Plan 2026 | Medium | 10-20 |
| 8 | KHA-CARI | AKI Summary | Medium | 15-25 |

**Total estimated rules from current 18 PDFs: ~200-320.**

---

## Copyright posture

Per the Layer 1 audit risk register:

> "Facts are uncopyrightable; the editorial prose around them is copyrighted.
> Re-phrase before loading."

All 18 PDFs in this directory are **gitignored** (`.gitignore` excludes
`*.pdf`, `*.docx`, `*.xlsx`, `*.zip`, `*.epub`, `*.mobi`). The structured
YAMLs we curate from them — containing the *facts* (drug → dose → condition)
re-phrased in our own words — are the only artefacts committed to the repo.

Most of these sources are openly downloadable (no auth required, no paywall),
but copyright still applies to the prose. The re-phrase posture is the
same as for Wang 2024 (`kb-4-patient-safety/knowledge/au/pims_wang_2024/`).
