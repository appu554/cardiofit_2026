# Page 40 Audit — Figure 5: SGLT2i Cardiovascular and Kidney Outcome Trials Table

| Field | Value |
|-------|-------|
| **Page** | 40 (PDF page S39) |
| **Content Type** | Figure 5: SGLT2i trial comparison table (EMPA-REG, CANVAS, CREDENCE, DAPA-CKD, EMPA-KIDNEY) |
| **Extracted Spans** | 104 total (99 pipeline + 5 REVIEWER) |
| **Channels** | C, D, F, REVIEWER |
| **Disagreements** | 0 |
| **Review Status** | ✅ **ACCEPTED** — 22 confirmed + 77 rejected + 5 REVIEWER added = 104/104 reviewed |
| **Risk** | Clean |
| **Audit Date** | 2026-02-27 (execution complete, raw PDF cross-check complete) |
| **Cross-Check** | Verified against raw API data (99 spans) AND raw PDF text. Drug dosing spans cross-checked against PDF Figure 5 table — all match. HR/CI outcome spans verified against PDF trial data rows. Raw PDF cross-check identified 5 gaps: missing enrollment criteria, truncated endpoints, OCR error, missing outcome definitions, missing population data. All 5 added as REVIEWER facts. |

---

## Source PDF Content

**Figure 5 — SGLT2i Trial Comparison Table (EVIDENCE TABLE):**

| Trial | Drug/Dose | N | eGFR Criteria | Key Kidney Result | Key CV Result |
|-------|-----------|---|---------------|-------------------|---------------|
| EMPA-REG | Empagliflozin 10/25 mg daily | 7020 | ≥30 | Nephropathy HR 0.61 (0.53-0.70) | MACE HR 0.86 (0.74-0.99); HF HR 0.65 (0.50-0.85) |
| CANVAS | Canagliflozin 100/300 mg daily | 10,142 | ≥30 | Composite kidney HR 0.53 (0.33-0.84) | MACE HR 0.86 (0.75-0.97); HF HR 0.67 (0.52-0.87) |
| CREDENCE | Canagliflozin 100 mg daily | 4401 | 30-90 | Primary kidney HR 0.70 (0.59-0.82) | CV death/MI/stroke HR 0.80 (0.67-0.95); HF HR 0.61 (0.47-0.80) |
| DAPA-CKD | Dapagliflozin 10 mg daily | 4304 | 25-75 | Primary outcome HR 0.61 (0.51-0.72) | CV death/HF HR 0.71 (0.55-0.92) |
| EMPA-KIDNEY | Empagliflozin 10 mg daily | 6609 | ≥20-<45 or ≥45-<90 | Stopped early for efficacy | Not reported |

---

## Execution Results

### API Confirmations (22/22 succeeded)

| # | Span Text (truncated) | Tier | Channel | Reason |
|---|----------------------|------|---------|--------|
| 1 | Incident/worsening nephropathy: 12.7% vs. 18.8%... HR 0.61 (0.53-0.70) | T1 | D 92% | EMPA-REG nephropathy outcome with HR |
| 2 | Composite doubling in SC, kidney failure, or death from kidney causes | T2 | D 92% | CREDENCE composite endpoint definition |
| 3 | Canagliflozin 100 mg, 300 mg once daily | T1 | D 92% | CANVAS drug dosing |
| 4 | Empagliflozin 10 mg, 25 mg once daily | T1 | D 92% | EMPA-REG drug dosing |
| 5 | Canagliflozin 100 mg once daily | T1 | D 92% | CREDENCE drug dosing |
| 6 | Dapagliflozin 10 mg once daily | T1 | D 92% | DAPA-CKD drug dosing |
| 7 | Empagliflozin 10 mg once daily | T1 | D 92% | EMPA-KIDNEY drug dosing |
| 8 | ACR 200-5000 mg/g [20-500 mg/mmol] ACR Median... | T2 | D 92% | CREDENCE ACR enrollment criteria |
| 9 | Composite kidney: 1.5 vs. 2.8 per 1000 patient-years... HR 0.53 (0.33-0.84) | T1 | D 92% | CANVAS composite kidney outcome |
| 10 | First occurrence of a composite of kidney disease progression... | T2 | D 92% | CREDENCE primary endpoint definition |
| 11 | First occurrence of a ≥50% decline in eGFR... (DAPA-CKD) | T2 | D 92% | DAPA-CKD primary endpoint definition |
| 12 | First occurrence of a ≥50% decline in eGFR... (EMPA-KIDNEY) | T2 | D 92% | EMPA-KIDNEY primary endpoint definition |
| 13 | CV death, MI, stroke: HR: 0.80, 95% CI: 0.67-0.95; HF: HR: 0.61 (0.47-0.80) | T2 | D 92% | CREDENCE CV outcome HRs |
| 14 | Secondary composite of CV death or HF: HR: 0.71; 95% CI: 0.55-0.92 | T2 | D 92% | DAPA-CKD CV outcome HR |
| 15 | MACE: HR: 0.86; 95% CI: 0.74-0.99; HF: HR: 0.65 (0.50-0.85) | T2 | D 92% | EMPA-REG MACE + HF HRs |
| 16 | MACE: HR: 0.86; 95% CI: 0.75-0.97; HF: HR: 0.67 (0.52-0.87) | T2 | D 92% | CANVAS MACE + HF HRs |
| 17 | Composite of kidney failure, doubling of SC, or death from kidney or CV causes | T2 | D 92% | CREDENCE kidney composite definition |
| 18 | Composite of kidney failure, doubling SC, or death from kidney or CV causes | T2 | D 92% | CREDENCE kidney composite (variant) |
| 19 | Primary outcome: HR: 0.61; 95% CI: 0.51-0.72 | T2 | D 92% | DAPA-CKD primary outcome HR |
| 20 | [Trial stopped early due to positive results] | T2 | D 92% | EMPA-KIDNEY early termination |
| 21 | Primary kidney: HR: 0.70; 95% CI: 0.59-0.82 | T2 | D 92% | CREDENCE primary kidney HR |
| 22 | Criteria: ACR >100-5000 mg/g [30-500 mg/mmol] Median ACR 927 mg/g | T1 | D 92% | CREDENCE ACR enrollment criteria |

### API Rejections (77/77 succeeded)

| Category | Count | Channel | Reason |
|----------|-------|---------|--------|
| **"Not reported"** | 10 | D 92% | Empty table cells — no clinical content |
| **Decontextualized numbers** (sample sizes, %, follow-up years: 7020, 10142, 4401, 4304, 6609, 37.4, 37.5, 2.6, 2.4, 3.1, etc.) | ~20 | D 92% | Standalone numbers without drug/outcome context |
| **Table headers** ("Primary outcome", "CV outcome results", "Kidney outcome results", "CARDIOVASCULAR TRIALS", "KIDNEY TRIALS", "Follow-up (yr)", "Total of participants", "Primary outcome(s)") | ~15 | D 92% | Column/section headers without data values |
| **"% with eGFR"**, **"% with CVD"** (column headers) | 6 | D 92% | Table column header fragments |
| **"No criteria"**, **"No information"** | 4 | D 92% | Empty criteria cells |
| **Decontextualized eGFR thresholds** ("eGFR ≥45" ×2, "eGFR ≥40" ×3) | 5 | C 95% | Trial enrollment eGFR values without trial name/drug context |
| **Decontextualized ranges** ("30-90", "25-75", "220-") | 3 | D 92% | eGFR ranges without context |
| **Standalone percentages** ("76" ×5, "20", "27") | 7 | D 92% | "% with eGFR/CVD" values without column context |
| **Corrupted OCR text** ("First occurrence of a 2% decline in eGFR...") | 1 | D 92% | 50% misread as 2% — corrupted duplicate of confirmed span |
| **Truncated/duplicate headers** ("eGFR criteria for enrollment..." ×3, "Mean eGFR at enrollment..." ×3, "CV death or hospitalization for HF" ×2) | 8 | D 92% | Table column headers without data |
| **"CANVAS"** (trial name) | 1 | D 92% | Trial name without results or context |
| **"No criteria Median ACR 12.3 mg/g..."** | 1 | D 92% | Mixed empty cell + uncontextualized ACR |
| **"Expected ≥3"** | 1 | D 92% | Fragment — follow-up duration without trial context |
| `<!-- PAGE 40 -->` | 1 | F 90% | Pipeline HTML artifact |

### REVIEWER Facts Added (5/5 succeeded) — Raw PDF Cross-Check

Initial review found 0 gaps (narrative context covered by Page 39). However, a **raw PDF verbatim cross-check** against confirmed spans identified 5 gaps in the tabular data itself:

| # | REVIEWER Fact Text | Note | Gap Type |
|---|-------------------|------|----------|
| 1 | EMPA-KIDNEY ACR enrollment criteria: eGFR ≥45–<90: ACR ≥200 mg/g [≥20 mg/mmol] (or PCR ≥300 mg/g [≥30 mg/mmol]). No ACR criteria for eGFR ≥40–<45. Median ACR 412 mg/g [41.2 mg/mmol]. | P40 Figure 5: EMPA-KIDNEY has stratified ACR enrollment criteria by eGFR band — entirely missing from pipeline extractions. KB-1 dosing. | Missing enrollment criteria |
| 2 | EMPA-KIDNEY primary outcome: First occurrence of a composite of kidney disease progression (kidney failure, sustained decline in eGFR to <10 ml/min/1.73 m², sustained decline in eGFR ≥40%, or renal death) or CV death. | P40 Figure 5: Confirmed span #12 truncated — missing <10 ml/min and ≥40% thresholds. Complete definition needed for KB-4 safety rules. | Truncated endpoint |
| 3 | EMPA-REG kidney outcome: Incident or worsening nephropathy (progression to severely increased albuminuria, doubling of SCr, initiation of KRT, or renal death) and incident albuminuria. | P40 Figure 5: EMPA-REG kidney outcome definition — 'nephropathy' never defined in confirmed spans. KB-4 safety. | Missing definition |
| 4 | CREDENCE ACR enrollment criteria (corrected): Criteria: ACR >300–5000 mg/g [>30–500 mg/mmol]. Median ACR 927 mg/g [92.7 mg/mmol]. | P40 Figure 5: Confirmed span #22 reads '>100-5000' but PDF says '>300-5000'. OCR error — cannot undo confirm, adding corrected version. KB-1 dosing. | OCR error correction |
| 5 | EMPA-REG ACR population: No criteria. ACR <30 mg/g [<3 mg/mmol] in 60%; 30–300 mg/g [3–30 mg/mmol] in 30%; >300 mg/g [>30 mg/mmol] in 10%. | P40 Figure 5: EMPA-REG enrolled mostly normoalbuminuric patients (60%) with no ACR enrollment criteria — important prescribing context for KB-1 dosing rules. | Missing population data |

**Note on initial "no gaps" assessment**: The first pass correctly identified that Page 39 REVIEWER facts cover the narrative context. The raw PDF cross-check found gaps *within the confirmed tabular spans themselves* — truncated text, OCR errors, and missing enrollment criteria columns that the pipeline decomposed into noise.

---

## Key Spans Assessment (Post-Review)

### Tier 1 Spans (13 total: 8 CONFIRMED, 5 REJECTED)

| Category | Count | Review Action |
|----------|-------|---------------|
| **Drug dosing**: Empagliflozin 10/25 mg, Canagliflozin 100/300 mg, Canagliflozin 100 mg, Dapagliflozin 10 mg, Empagliflozin 10 mg | 5 | **✅ CONFIRMED** — Drug + dose + frequency from trial protocols |
| **Outcome with HR**: Nephropathy HR 0.61, Composite kidney HR 0.53 | 2 | **✅ CONFIRMED** — Trial outcome data with drug context (accepted as T1 per pipeline classification) |
| **ACR criteria**: "ACR >100-5000 mg/g..." | 1 | **✅ CONFIRMED** — CREDENCE enrollment criterion with values |
| **eGFR thresholds**: "eGFR ≥45" ×2, "eGFR ≥40" ×3 | 5 | **❌ REJECTED** — Decontextualized trial enrollment eGFR values without drug/trial context |

### Tier 2 Spans (86 total: 14 CONFIRMED, 72 REJECTED)

| Category | Count | Review Action |
|----------|-------|---------------|
| **HR/CI values with context** (MACE, HF, kidney, CV outcomes) | 8 | **✅ CONFIRMED** — Trial outcome data with HR + 95% CI |
| **Composite endpoint descriptions** | 4 | **✅ CONFIRMED** — Outcome definitions (kidney failure, eGFR decline, death) |
| **ACR enrollment criteria with values** | 1 | **✅ CONFIRMED** — CREDENCE ACR criteria |
| **Trial stopped early** | 1 | **✅ CONFIRMED** — EMPA-KIDNEY efficacy signal |
| **Noise spans** (numbers, headers, empty cells, artifacts) | 72 | **❌ REJECTED** — See rejection categories above |

---

## Critical Findings

### ✅ D Channel Captures Drug Dosing from Table (Good)
The D channel successfully extracted 5 drug + dose + frequency spans from the trial table (e.g., "Empagliflozin 10 mg, 25 mg once daily"). These are correctly classified as T1 — they represent specific SGLT2i dosing protocols used in major trials and inform prescribing.

### ✅ Genuine Trial Outcome Data Captured (Good)
The D channel captured key hazard ratios with confidence intervals for all 5 trials:
- EMPA-REG: Nephropathy HR 0.61 (0.53-0.70), MACE HR 0.86 (0.74-0.99), HF HR 0.65 (0.50-0.85)
- CANVAS: Composite kidney HR 0.53 (0.33-0.84), MACE HR 0.86 (0.75-0.97), HF HR 0.67 (0.52-0.87)
- CREDENCE: Primary kidney HR 0.70 (0.59-0.82), CV death/MI/stroke HR 0.80 (0.67-0.95), HF HR 0.61 (0.47-0.80)
- DAPA-CKD: Primary outcome HR 0.61 (0.51-0.72), CV death/HF HR 0.71 (0.55-0.92)
- EMPA-KIDNEY: Trial stopped early for efficacy

### ⚠️ Massive Table Decomposition Noise (77 rejected)
77 of 99 spans (78%) were noise: decontextualized numbers, empty cells, table headers, and column fragments. This is characteristic of D channel table decomposition on evidence summary tables.

### ⚠️ Corrupted OCR Span Detected
One span read "First occurrence of a 2% decline in eGFR" — the "50%" was misread as "2%". This was correctly rejected as corrupted.

### ⚠️ Raw PDF Cross-Check Found 5 Tabular Gaps
Initial review found no narrative gaps (Page 39 covers context). However, raw PDF verbatim cross-check revealed:
- **EMPA-KIDNEY ACR**: Complex stratified enrollment criteria entirely missing from pipeline
- **EMPA-KIDNEY endpoint**: Confirmed span truncated — missing <10 ml/min and ≥40% thresholds
- **EMPA-REG outcome definition**: "Nephropathy" never defined in confirmed spans
- **CREDENCE ACR OCR error**: Confirmed span reads ">100-5000" but PDF says ">300-5000"
- **EMPA-REG population**: 60% normoalbuminuric — critical KB-1 prescribing context missing

All 5 gaps added as REVIEWER facts. Cross-page narrative coverage via Page 39 remains intact.

---

## Final Disposition

| Action | Details |
|--------|---------|
| **Decision** | **ACCEPTED** |
| **Total Extractions** | 104 (99 pipeline + 5 REVIEWER) |
| **Confirmed** | 22 (5 drug dosing T1, 2 outcome HR T1, 1 ACR T1, 8 HR/CI T2, 4 endpoint descriptions T2, 1 ACR T2, 1 trial stopped T2) |
| **Rejected** | 77 (10 "Not reported", ~20 numbers, ~15 headers, 6 column headers, 5 eGFR fragments, 4 empty criteria, 7 standalone %, 3 ranges, 1 corrupted OCR, 8 duplicate headers, 1 trial name, 1 mixed cell, 1 follow-up fragment, 1 pipeline artifact) |
| **REVIEWER Added** | 5 (1 missing enrollment criteria, 1 truncated endpoint, 1 missing outcome definition, 1 OCR error correction, 1 missing population data) |
| **Page Completeness** | **~95%** — All drug dosing + trial outcome HRs + enrollment criteria + endpoint definitions + population data captured. Minor gaps: individual trial demographics (age, diabetes duration) not in span form |

---

## Completeness Score (Post-Review)

| Metric | Pre-Review | Post-Review |
|--------|------------|-------------|
| **Extraction completeness** | ~45% (estimated) | **~95%** — Drug dosing + trial outcome HRs + enrollment criteria + endpoint definitions + population data. Raw PDF cross-check closed 5 gaps. |
| **Tier accuracy** | ~20% | **26%** (27 genuine / 104 total) — low due to massive D channel table noise |
| **False positive rate** | ~80% | **74%** (77/104 rejected) — characteristic of evidence table pages |
| **Genuine content** | Unknown | **27 total** — 22 confirmed pipeline spans + 5 REVIEWER additions |
| **Overall quality** | **FAIR** | **VERY GOOD** — Comprehensive evidence table coverage after raw PDF cross-check. D channel captures drug dosing and HR outcomes well; REVIEWER facts fill enrollment criteria, endpoint definitions, OCR corrections, and population data. Page 39 REVIEWER facts provide complementary narrative context. |
