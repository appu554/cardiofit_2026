# ACSQHC Clinical Care Standards — Procurement Runbook

**Status (2026-04-30 evening):** ✅ **9 PDFs (27.4 MB) downloaded** via Playwright-driven browser fetch. Direct curl/wget still returns 000 from this dev env (TLS/CDN/firewall block), but Playwright's Chromium context completes the TLS handshake successfully and fetch() in browser context bypasses the block.

The procurement path is now:
- **Code path:** Playwright `browser_navigate` to find PDF URLs, then `browser_evaluate` with `fetch()` + base64 encoding to retrieve, decode locally to disk.
- **Manual fallback:** Original browser-driven path documented below remains valid for users without Playwright.

## What to download

The Australian Commission on Safety and Quality in Health Care (ACSQHC) publishes "Clinical Care Standards" — concise definitions of the care a person should receive for a specific condition. For aged-care decision support, the highest-priority standards are:

### Highest priority (aged-care relevant)

1. **Antimicrobial Stewardship Clinical Care Standard** (2nd ed, 2020)
   - Source: `https://www.safetyandquality.gov.au/standards/clinical-care-standards/antimicrobial-stewardship-clinical-care-standard`
   - Why: antibiotic prescribing in aged care is a top-3 PIM area
   - Maps to: KB-4 (PIM rules), KB-13 (QI Program AMS sub-indicators)

2. **Delirium Clinical Care Standard** (2021)
   - Source: `.../clinical-care-standards/delirium-clinical-care-standard`
   - Why: ~30% prevalence in hospitalised aged-care residents; tied to antipsychotic/anticholinergic over-use
   - Maps to: KB-4, KB-22 (HPI engine — delirium screening triggers)

3. **Hip Fracture Care Clinical Care Standard** (2nd ed, 2023)
   - Source: `.../clinical-care-standards/hip-fracture-care-clinical-care-standard`
   - Why: high mortality fall outcome in aged care; ties to bone-health prescribing (alendronate, denosumab) + anticoagulant management
   - Maps to: KB-1 (peri-operative dosing), KB-4, KB-13

4. **Cognitive Impairment in Hospital Care Clinical Care Standard** (2014, refreshed)
   - Source: `.../clinical-care-standards/cognitive-impairment-clinical-care-standard`
   - Why: dementia/MCI screening in transitions; ties to antipsychotic Quality Indicator
   - Maps to: KB-4, KB-22

5. **Medication Management at Transitions of Care Stewardship Framework 2024** ⭐ NEW
   - Source: `https://www.safetyandquality.gov.au/our-work/medication-safety/medication-management-transitions-care`
   - Why: v2 Revision Mapping document explicitly names this. Defines hospital→aged-care transition stewardship that ACOP services will be measured against.
   - Maps to: KB-4 transitions rules, KB-13 (PHARMA-Care Domain 3 indicator), KB-2 (Clinical state baseline at transitions)

### Lower priority (skip for now)

- Heavy Menstrual Bleeding (n/a for aged care)
- Acute Coronary Syndromes (already covered by Heart Foundation guidelines we have)
- Stroke (lower aged-care RACF relevance vs hospital)

## Manual download steps

1. **Open in a normal browser** (Chrome/Firefox/Safari):
   ```
   https://www.safetyandquality.gov.au/standards/clinical-care-standards
   ```
2. For each standard listed above, navigate to its page and look for download links labelled:
   - "Clinical Care Standard" — the **main standard PDF** (~30-50 pages)
   - "Indicators" or "Indicator Specification" — measurement specs
   - Sometimes "Information for clinicians" — supplementary guidance

3. Save downloads to:
   ```
   backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/wave6/acsqhc_ams/
       AMS-Clinical-Care-Standard-2020.pdf
       AMS-Indicator-Specification-2020.pdf
       Delirium-Clinical-Care-Standard-2021.pdf
       Hip-Fracture-Care-Clinical-Care-Standard-2023.pdf
       Cognitive-Impairment-Clinical-Care-Standard.pdf
       Medication-Management-Transitions-Stewardship-Framework-2024.pdf
   ```

4. The wave6/.gitignore already excludes `*.pdf` so no commit-related cleanup needed.

## After PDFs land

1. Update [wave6 MANIFEST.md](../MANIFEST.md) — flip ACSQHC row from "⏳ manual procurement needed" to ✅ landed.
2. Run Pipeline 1 V4 against the 6 PDFs (they'll feed KB-4 + KB-13).
3. Re-run cross-KB validator to confirm new content lands in KB-13 measures table.

## Why we couldn't auto-download

Several attempts from this dev environment (`curl` + UA + TLS variants):
- `https://www.safetyandquality.gov.au/` → status 000 (connection failure)
- `web.archive.org` cached versions of those pages → status 000 (same TLS/CDN issue)
- HTTP/1.1 explicit retry → status 000

The most likely cause is a firewall/CDN configuration that drops requests from this network's egress IP range. Manual browser download from a normal client (with cookies, JS, full TLS handshake) is the workaround.

## Procurement status (2026-04-30 19:25 IST — POST-PLAYWRIGHT)

| Document | Path | Size | Status |
|----------|------|-----:|--------|
| Antimicrobial Stewardship Clinical Care Standard 2020 | `ccs/AMS-Clinical-Care-Standard-2020.pdf` | 2.87 MB | ✅ landed |
| Delirium Clinical Care Standard 2021 | `ccs/Delirium-Clinical-Care-Standard-2021.pdf` | 1.42 MB | ✅ landed |
| Hip Fracture Clinical Care Standard 2023 | `ccs/Hip-Fracture-Clinical-Care-Standard-2023.pdf` | 3.07 MB | ✅ landed |
| Opioid Analgesic Stewardship in Acute Pain CCS | `ccs/Opioid-Analgesic-Stewardship-in-Acute-Pain-CCS.pdf` | 1.20 MB | ✅ landed |
| **Psychotropic Medicines in Cognitive Disability or Impairment CCS** ⭐ | `ccs/Psychotropic-Medicines-in-Cognitive-Disability-Impairment-CCS.pdf` | 2.35 MB | ✅ landed (replaces earlier "Cognitive Impairment in Hospital Care" reference — better aged-care fit, directly maps to KB-13 AU-QI-06 antipsychotic indicator) |
| Venous Thromboembolism Prevention CCS 2020 | `ccs/VTE-Prevention-Clinical-Care-Standard-2020.pdf` | 7.11 MB | ✅ landed |
| **Medication Management at Transitions of Care Stewardship Framework 2024** ⭐ | `framework/Medication-Management-Transitions-Care-Stewardship-Framework-2024.pdf` | 5.81 MB | ✅ landed (the v2-introduced piece) |
| Cognitive Impairment User Guide 2019 (supporting) | `supporting/Cognitive-Impairment-User-Guide-2019.pdf` | 2.94 MB | ✅ landed |
| **National Medication Management Plan User Guide 2021** | `supporting/National-Medication-Management-Plan-User-Guide-2021.pdf` | 1.85 MB | ✅ landed (bonus — structured medication-management template referenced by Psychotropic CCS) |
| **Total** | 3 subdirs | **27.4 MB** | **9 / 9 PDFs ✅** |

## How the Playwright fallback worked (recipe for future ACSQHC procurement)

```python
# Stage A: discover PDFs from CCS overview page
browser_navigate("https://www.safetyandquality.gov.au/clinical-care-standards")
# Each individual standard's page exposes a "Download" link to a PDF under
# /sites/default/files/resources/attachments/<filename>.pdf

# Stage B: per-PDF fetch via browser context (bypasses TCP/curl block)
browser_evaluate(async () => {
    const r = await fetch(pdf_url);
    const buf = await r.arrayBuffer();
    // chunked btoa to avoid stack overflow on large PDFs
    let bin = '';
    const chunk = 0x8000;
    for (let i = 0; i < bytes.byteLength; i += chunk)
        bin += String.fromCharCode.apply(null, bytes.subarray(i, i + chunk));
    return btoa(bin);
});

# Stage C: decode base64 → write to disk via Python
python3 -c "
import json, base64
data = json.load(open('.playwright-mcp/...json'))
open(dest, 'wb').write(base64.b64decode(data))
"
```

Same recipe should work for any other AU regulator site that's reachable in browser but blocked by curl from this dev env.
