# RANZCP Clinical Practice Guidelines — Procurement Notes

**Procured 2026-04-30 evening via Playwright** (curl/wget blocked from this dev env, identical pattern to ACSQHC).

## What we got (9 PDFs / 30 MB)

| Filename | Size | Source | What it is |
|----------|-----:|--------|------------|
| `RANZCP-BPSD-Assessment-Management-NSWHealth-2022.pdf` | 1.4 MB | NSW Health (RANZCP-endorsed) | **Highest priority for aged care** — assessment + management of behavioural & psychological symptoms of dementia |
| `RANZCP-Mood-Disorders-CPG.pdf` | 4.0 MB | RANZCP `/getmedia/` | Joint Mood Disorders CPG (depression + bipolar) |
| `RANZCP-Pain-Mgmt-Acute-ANZCA-APMSE5-2020.pdf` | 22.9 MB | ANZCA (RANZCP-endorsed) | Acute Pain Management: Scientific Evidence 5th edition (2020) |
| `RANZCP-Prescription-Opioids-RACP-Policy.pdf` | 1.9 MB | RACP (RANZCP-endorsed) | Prescription opioid policy |
| `RANZCP-Physical-Health.pdf` | 619 KB | RANZCP `/getmedia/` | Physical health of psychiatric patients |
| `RANZCP-Valproate-Healthcare-BAP-PS04-2018.pdf` | 270 KB | British Assoc for Psychopharmacology (RANZCP-endorsed) | Valproate use position statement PS04-18 |
| `RANZCP-ECT-Professional-Practice.pdf` | 197 KB | RANZCP `/getmedia/` | Electroconvulsive therapy professional practice |
| `RANZCP-Off-Label-Prescribing-Psych.pdf` | 185 KB | RANZCP `/getmedia/` | PPG-4 Off-label prescribing in psychiatry |
| `RANZCP-Benzodiazepines-Psychiatric-Practice.pdf` | 140 KB | RANZCP `/getmedia/` | PPG-5 Use of benzodiazepines in psychiatric practice |

## Items NOT downloaded (and why)

| Slug | Reason |
|------|--------|
| `antipsychotics-and-dementia` | **RANZCP rescinded the formal PPG-10 guideline pending review** — text-only landing page on RANZCP, no PDF. Use BPSD handbook (NSW Health) above instead. |
| `psychiatric-service-delivery-for-older-people-with-mental-disorders-and-dementia` | Page is JS-rendered; static HTML didn't expose a PDF link. May be text-only position statement. |
| `de-prescribing-cholinesterase-inhibitors-and-memantine` | Same — JS-rendered, no PDF link in static HTML. **High aged-care value** if recoverable; manual browse may yield. |
| `dementia` | Endorsed external guidance — RANZCP just links to U Sydney CDPC's dementia guidelines portal (HTML, not single PDF) |
| `care-of-sedated-abd-patients` | JS-rendered |
| `post-traumatic-stress-disorder-endorsed-guidelines` | JS-rendered |
| `pharmacogenetics-in-healthcare` | JS-rendered |

## How the Playwright procurement worked

```
curl/wget       → 000 (network/CORS blocked from this dev env)
Playwright nav  → loads page, JS renders content
fetch() in page context (same-origin) → returns bytes for PDFs hosted on RANZCP
fetch() cross-origin → fails CORS

Workaround for cross-origin PDFs (NSW Health, BAP, ANZCA, RACP):
  1. browser_navigate to the PDF URL directly (Chromium handles disclaimer/auth)
  2. Once on the PDF's origin, fetch(window.location.href) is same-origin
  3. arrayBuffer → base64 → return → Python decode to disk

For PDFs that auto-trigger Chromium's download manager (e.g., ANZCA's 22 MB
APMSE5), Playwright deposits the file directly in `.playwright-mcp/` —
move it into place after navigation completes.
```

## Outstanding aged-care priorities still missing

The 7 not-downloaded items above include 3 high-value-if-recoverable aged-care items:
1. **Psychiatric service delivery for older people with mental disorders and dementia** — directly aged-care-targeted
2. **De-prescribing cholinesterase inhibitors and memantine** — aged-care deprescribing
3. **Care of sedated ABD patients** — psychotropic-stewardship-adjacent

For these, a manual browse with download via Chromium UI (instead of programmatic fetch) is likely needed. The pages may use JS-loaded download widgets that the static-HTML scrape doesn't catch.

## Mapping to ACOP rule layer

| RANZCP source | KB consumed | Rule type |
|---------------|-------------|-----------|
| BPSD handbook | KB-4 | CONTRAINDICATION (antipsychotic for BPSD without trial of non-pharm), PRESCRIBING_OMISSION (referral to old-age psychiatry) |
| Mood Disorders CPG | KB-1, KB-4 | DOSING (titration), CONTRAINDICATION (TCA in elderly) |
| Acute Pain Management | KB-1, KB-4, KB-5 | DOSING (opioid + paracetamol step-up), DRUG_DRUG_INTERACTION (opioid + benzodiazepine), CONTRAINDICATION |
| Prescription opioid policy | KB-4 | CONTRAINDICATION (chronic non-malignant pain → opioid avoidance), PRESCRIBING_OMISSION (deprescribing taper) |
| Valproate position | KB-4 | CONTRAINDICATION (women of childbearing potential — N/A for aged care; relevant for liver/cognitive monitoring) |
| Benzodiazepine PPG | KB-4 (overlay onto STOPP D6/D8 already loaded) | reinforces existing STOPP rules |
| Off-label prescribing | KB-4 | governance metadata for any rule citing off-label use |
| Physical health | KB-16 | CLINICAL_DECISION_LIMIT (metabolic monitoring for SGAs) |
| ECT | (not Layer 1 — non-drug therapy) | reference only |

## Maps to existing KB-13 indicators

- BPSD + Antipsychotic policy → directly inform **AU-QI-06-MEDICATION-ANTIPSYCHOTIC** in KB-13
- Benzodiazepines PPG → reinforces **AU-QI-05-MEDICATION-POLYPHARMACY** logic
- Prescription opioids → contributes to potential future opioid-related QI indicator
