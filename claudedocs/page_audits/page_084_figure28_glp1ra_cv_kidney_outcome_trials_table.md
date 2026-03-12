# Page 84 Audit — Figure 28: GLP-1 RA Cardiovascular and Kidney Outcome Trials Comparison Table

| Field | Value |
|-------|-------|
| **Page** | 84 (PDF page S83) |
| **Content Type** | Figure 28: Comprehensive comparison table of GLP-1 RA cardiovascular and kidney outcome trials — LEADER (liraglutide, N=9340), SUSTAIN-6 (semaglutide, N=3297), HARMONY (albiglutide, N=9463), REWIND (dulaglutide, N=9901), ELIXA (lixisenatide, N=6068), EXSCEL (exenatide, N=14,752), AMPLITUDE-O (efpeglenatide, N=4076), AWARD-7 (dulaglutide, N=577, 100% CKD G3a-G4). Table columns: trial name, drug, sample size, follow-up, % CVD, % CKD, eGFR inclusion, baseline albuminuria, CV outcome definition/HR, kidney outcome definition/HR |
| **Extracted Spans** | 200 total (51 T1, 149 T2) |
| **Channels** | B (Drug Dictionary), C (Grammar/Regex), D (Table Decomp) |
| **Disagreements** | 0 |
| **Review Status** | CONFIRMED: 1, PENDING: 199 |
| **Risk** | Disagreement |
| **Cross-Check** | Count corrected UPWARD 160→200 (T1 52→51, T2 108→149); verified against raw extraction data |
| **Audit Date** | 2026-02-25 (revised) |

---

## Source PDF Content

**Figure 28 — Cardiovascular and Kidney Outcome Trials for GLP-1 RA:**

Full caption: "Figure 28 | Cardiovascular and kidney outcome trials for glucagon-like peptide-1 receptor agonists (GLP-1 RA). ACEi, angiotensin-converting enzyme inhibitor; ACR, albumin–creatinine ratio; ARB, angiotensin II receptor blocker; CI, confidence interval; CKD, chronic kidney disease; CrCl, creatinine clearance; CV, cardiovascular; CVD, cardiovascular disease; eGFR, estimated glomerular filtration rate (ml/min per 1.73 m²); G, glomerular filtration rate category; G3a–G4, estimated glomerular filtration rate 15–59 ml/min per 1.73 m²; HR, hazard ratio; KRT, kidney replacement therapy; MI, myocardial infarction; NA, not available; SCr, serum creatinine."

**Reconstructed Table from D Channel Extractions:**

| Trial | Drug | N | FU (yr) | % CVD | % CKD | eGFR Incl | Baseline Albuminuria | CV Outcome HR | Kidney Outcome | Kidney HR |
|-------|------|---|---------|-------|-------|-----------|---------------------|---------------|----------------|-----------|
| **LEADER** | Liraglutide | 9340 | 3.8 | 81.3 | 22.9 | ≥30 (incl 220 with eGFR <30) | — | HR 0.85 (0.77–0.93) | Composite severely increased albuminuria (ACR >300 mg/mg or >33.9 mg/mmol) | HR 0.78 (0.67–0.92) |
| **SUSTAIN-6** | Semaglutide | 3297 | 2.1 | 83 | 22.2 | ≥30 (0.9% had eGFR <30) | Median 28.3 mg/g [2.83 mg/mmol] | HR 0.75 (0.58–0.95) | New-onset macroalbuminuria HR 0.81 (0.66–0.99); Doubling SCr HR 1.16 | HR 0.64 (0.46–0.88) |
| **HARMONY** | Albiglutide | 9463 | 1.6 | — | — | ≥30 | — | HR 0.78 (0.67–0.92) | — | — |
| **REWIND** | Dulaglutide | 9901 | 3.2 | 31.5 | 26.9 | ≥15 | 19% moderately increased, 7% severely increased | HR 0.85 (0.57–1.11)? | New-onset severely increased albuminuria + doubling SCr | — |
| **ELIXA** | Lixisenatide | 6068 | 2.08 | — | — | ≥30 | — | HR 1.02 (0.89–1.17) | — | — |
| **EXSCEL** | Exenatide | 14,752 | 3.2 | 72.4–76.9 | — | ≥30 | — | HR 0.91 (—) | 40% eGFR decline, kidney replacement, or renal death | HR 0.87 (0.73–1.04) NS |
| **AMPLITUDE-O** | Efpeglenatide | 4076 | 1.81 | 89.6 | 31.6 | ≥30 | — | HR 0.95 (0.58–0.92)? | New severely increased albuminuria ACR >33.9 mg/ml, sustained eGFR fall 30%, or KRT | HR 0.68 (0.57–0.79) |
| **AWARD-7** | Dulaglutide | 577 | 1.36 | — | 100 G3a-G4 | — | 44% with moderately increased | — | eGFR decline -0.7 ml/min per 1.73 m² (dulaglutide 1.5 mg, 0.75 mg) | — |

**Note**: Some HR values appear garbled by D channel extraction (e.g., "HR: 0.91% CI: 0.99% CI: 0.78-0.95" is likely two concatenated HRs). Row-column alignment of extracted cells is approximate — see Critical Findings below.

---

## Key Spans Assessment

### Tier 1 Spans (52)

| Span Pattern | Channel | Count | Conf | Assessment |
|--------------|---------|-------|------|------------|
| "eGFR did not significantly decline (-0.7 ml/min per 1.73 m²) with dulaglutide 1.5 mg or dulaglutide 0.75 mg..." | D | 1 | 92% | **⚠️ T2** — AWARD-7 kidney outcome finding. Evidence data, not patient safety directive |
| "Most had eGFR ≥30, but did include 220 patients" | D | 1 | 92% | **→ T3** — LEADER study population description |
| "Composite of incident severely increased albuminuria (ACR >300 mg/mg or >33.9 mg/mmol), increase in ACR" | D | 1 | 92% | **⚠️ T2** — Kidney outcome definition (LEADER) |
| "HR: 0.85; CI: 0.77-0.93 Similar for eGFR >60 vs." | D | 1 | 92% | **⚠️ T2** — LEADER MACE HR with eGFR subgroup |
| GLP-1 RA ×5 | B | 5 | 100% | **→ T3** — Drug class abbreviation mentions (1 already CONFIRMED as T1 — mis-confirmation) |
| liraglutide ×5 | B | 5 | 100% | **→ T3** — Drug name from LEADER/context |
| semaglutide ×5 | B | 5 | 100% | **→ T3** — Drug name from SUSTAIN-6/context |
| dulaglutide ×5 | B | 5 | 100% | **→ T3** — Drug name from REWIND/AWARD-7 |
| exenatide ×3 | B | 3 | 100% | **→ T3** — Drug name from EXSCEL |
| ACEi ×2, ARB ×2 | B | 4 | 100% | **→ T3** — Drug class names from figure abbreviation list |
| insulin ×1 | B | 1 | 100% | **→ T3** — Drug name in context |
| eGFR thresholds (≤30, <60, ≥60, <71, ≥30) ×13 | C | 13 | 95% | **→ T3** — Standalone eGFR thresholds from trial inclusion criteria, not clinical directives |

**Summary: 0/52 T1 genuine patient safety content. 3 D-channel table cells → T2 (evidence data). 35 are drug names → T3. 13 are eGFR threshold fragments → T3. 1 CONFIRMED GLP-1 RA is a T3 mis-confirmation.**

### Tier 2 Spans (108)

| Span Pattern | Channel | Count | Conf | Assessment |
|--------------|---------|-------|------|------------|
| **HR values with CIs** (0.81, 1.16, 1.02, 0.87, 0.68, 0.91, 0.75, 0.85, 0.70, 0.95, 0.78, 0.64) | D | ~12 | 92% | **✅ T2 CORRECT** — Clinical evidence metrics from trial outcomes |
| **Kidney outcome definitions** (new-onset macroalbuminuria, composite severely increased albuminuria, 40% eGFR decline + KRT, sustained eGFR fall 30%) | D | ~6 | 92% | **✅ T2 CORRECT** — Outcome specification definitions |
| **"CV death, nonfatal MI, or nonfatal stroke"** ×6 | D | 6 | 92% | **⚠️ 1 correct T2, 5 duplicates** — MACE definition repeated per trial row |
| **"Kidney composite outcome: HR: 0.68; 95% CI: 0.57-0.79"** | D | 1 | 92% | **✅ T2 CORRECT** — AMPLITUDE-O kidney composite HR |
| **"19% with moderately increased albuminuria and 7% with severely increased albuminuria"** | D | 1 | 92% | **✅ T2 CORRECT** — REWIND baseline albuminuria distribution |
| **Baseline albuminuria data** (Median 28.3 mg/g [2.83 mg/mmol]) | D | 1 | 92% | **✅ T2 CORRECT** — SUSTAIN-6 baseline data |
| **"Kidney outcome (secondary end points)"** | D | 1 | 92% | **→ T3** — Column header label |
| **"CV outcome definition"** | D | 1 | 92% | **→ T3** — Column header label |
| **"Follow-up time (yr)"** | D | 1 | 92% | **→ T3** — Column header label |
| Trial names: SUSTAIN, AWARD-7, EXSCEL, ELIXA | D | 4 | 92% | **→ T3** — Table row labels |
| **"Lisakenatide"** | D | 1 | 92% | **🚨 OCR CORRUPTION** — Should be "Lixisenatide" (ELIXA trial drug) |
| Sample sizes: 6068, 9340, 3297, 9463, 9901, 3183, 4076 | D | 7 | 92% | **→ T3/NOISE** — Bare numbers (trial N) without row context |
| % CVD: 81.3, 31.5, 84.7, 89.6, 76.9, 72.4 | D | 6 | 92% | **→ T3/NOISE** — Bare percentages without column header |
| % CKD: 22.9, 22.2, 26.9, 31.6 | D | 4 | 92% | **→ T3/NOISE** — Bare percentages |
| Follow-up times: 3.8, 2.1, 3.2, 1.6, 3.2, 2.08, 1.36, 1.81 | D | ~5 | 92% | **→ T3/NOISE** — Bare numbers |
| eGFR inclusion: ≥30 ×5, ≥15, 100 ×4 | D | ~11 | 92% | **→ T3/NOISE** — Bare numbers/thresholds |
| CKD detail: "100 with CKD G3a-G4", "577" | D | 2 | 92% | **→ T3** — AWARD-7 CKD description fragments |
| eGFR bare ×14 | C | 14 | 85% | **→ NOISE** — Standalone "eGFR" abbreviation |
| HbA1c ×5 | C | 5 | 85% | **→ NOISE** — Standalone lab name |
| mg values (28.3, 2.83, 33.9, 339, 300, 1.5, 0.75) ×9 | C | 9 | 85% | **→ T3/NOISE** — Bare dosing/threshold values from ACR criteria |
| weekly/daily ×4 | C | 4 | 85% | **→ T3** — Dosing frequency words from trial context |
| "20.7 with eGFR 30 to 59 ml/min per 1.73 m², 2.4 with eGFR..." | D | 1 | 92% | **✅ T2 CORRECT** — CKD distribution data |

**Summary: ~22/108 T2 correctly tiered (HR values, kidney outcome definitions, baseline data). ~30 are bare numbers/noise. ~50 are fragments, labels, or duplicates → T3 or NOISE.**

---

## Critical Findings

### ✅ D Channel Fires on Figure 28 — Second Massive Table (78 Spans)

This is the second major D channel table extraction in the audit (after Figure 24 on page 77 with 162 spans). Figure 28 is a comprehensive GLP-1 RA trial comparison table with ~8 trials × ~12 columns = ~96 data cells. The D channel captured 78 of these cells.

**D channel performance comparison:**

| Page | Figure | D Spans | Useful T2 | Noise/T3 | Useful % |
|------|--------|---------|-----------|----------|----------|
| 77 | Figure 24 (SGLT2i trials) | 162 | ~20 | ~142 | ~12% |
| 80 | Figure 26 (Metformin formulations) | 8 | 6 | 2 | 75% |
| 84 | **Figure 28 (GLP-1 RA trials)** | **78** | **~22** | **~56** | **~28%** |

**Pattern confirmed**: D channel performs best on small, structured dosing tables (Figure 26 → 75% useful) but produces massive noise on large clinical trial comparison tables (Figures 24, 28) because it extracts individual cells without row/column context.

### 🚨 "Lisakenatide" — SECOND Drug Name OCR Corruption

The D channel extracted "Lisakenatide" instead of "Lixisenatide" (the ELIXA trial drug). This is the **second OCR drug name corruption** in the audit:

| Page | Corrupted | Correct | Drug |
|------|-----------|---------|------|
| 77 | "Slaglifcin" | Sotagliflozin | SGLT2i |
| **84** | **"Lisakenatide"** | **Lixisenatide** | **GLP-1 RA** |

Both corruptions occur in the D channel on large table pages, suggesting OCR quality degrades on complex table layouts. "Lisakenatide" is particularly dangerous because it's close enough to the real name to potentially pass automated drug name validation.

### ⚠️ HR Value Concatenation Artifacts

Several D channel extractions appear to concatenate two HR values from adjacent cells:
- "HR: 0.91% CI: 0.99% CI: 0.78-0.95" — likely two separate HRs from adjacent columns
- "HR: 0.85% CI: 0.57-1.11" and "HR: 0.70% CI: 0.57-1.11" — possibly column misalignment

This is a D channel table parsing artifact where cell boundaries are not cleanly detected in dense, multi-column tables.

### ⚠️ GLP-1 RA CONFIRMED as T1 — Mis-Confirmation

One "GLP-1 RA" span has been CONFIRMED as T1 by a reviewer. This is a **mis-confirmation** — "GLP-1 RA" as a standalone drug class abbreviation is T3 (informational), not T1 (patient safety). The drug class name alone, without a clinical assertion about safety/dosing/contraindication, does not constitute patient safety content.

### ❌ No F or E Channels on Table Page

F (NuExtract LLM) and E (GLiNER NER) did not fire on page 84. This is consistent with the pattern:
- **F fires on**: Evidence prose pages (pp65, 72, 79, 80, 81)
- **F silent on**: Table-heavy pages (pp77, 83, 84)
- **E barely fires anywhere** in the Chapter 4 audit

### ❌ No Row-Column Context for 78 D Cells

The D channel extracts "9340" but doesn't associate it with "LEADER" or "Sample Size". It extracts "81.3" but doesn't link it to "% Established CVD" or "LEADER". This makes the 40+ bare numbers essentially useless — a reviewer cannot determine what trial or column they belong to without manually cross-referencing the PDF.

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| LEADER kidney: composite albuminuria HR 0.78 (0.67-0.92) — **with trial name context** | **T2** | Key kidney outcome (partially captured but decontextualized) |
| SUSTAIN-6 kidney: HR 0.64 (0.46-0.88) new-onset macroalbuminuria — **with context** | **T2** | Strongest GLP-1 RA kidney signal |
| AMPLITUDE-O kidney: HR 0.68 (0.57-0.79) — captured but row context missing | **T2** | Kidney composite outcome |
| AWARD-7: Only CKD G3a-G4 trial (100% CKD population) | **T2** | Unique CKD-specific trial data |
| EXSCEL: kidney HR 0.87 (0.73-1.04) — NS result important for evidence balance | **T2** | Negative kidney finding (balances positive results) |
| ELIXA: CV safety confirmed, no kidney benefit | **T2** | Evidence completeness |
| Trial-level integrated rows (trial name + drug + N + outcomes together) | **T2** | Without row integration, individual cells are meaningless |
| CKD subgroup consistency across GLP-1 RA class for kidney outcomes | **T1** | Drug class kidney benefit summary |

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **Conditional ACCEPT** — D channel captures 78 table cells including ~22 useful HR values and kidney outcome definitions; but massive noise from decontextualized numbers; OCR corruption "Lisakenatide"; no F/E channels; 0/52 T1 genuine patient safety |
| **Tier corrections** | All 35 B drug names: T1 → T3; All 13 C eGFR thresholds: T1 → T3; 3 D evidence cells: T1 → T2; 1 CONFIRMED GLP-1 RA: should be T3 |
| **Missing T2** | Row-integrated trial summaries (trial + drug + outcome + HR together); AWARD-7 CKD-specific context |
| **OCR fix needed** | "Lisakenatide" → "Lixisenatide" |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~30% — D channel captures many table cells but without row/column context; HR values present but decontextualized; kidney outcome definitions partially captured |
| **Tier accuracy** | ~14% (0/52 T1 correct + ~22/108 T2 correct = ~22/160) |
| **Noise ratio** | ~65% — 35 drug names + 14 bare eGFR + 5 bare HbA1c + 40+ bare numbers + 6 MACE duplicates = ~100/160 |
| **Genuine T1 content** | 0 extracted (page is evidence table = T2/T3 content) |
| **Prior review** | 1/160 reviewed (1 GLP-1 RA CONFIRMED as T1 — mis-confirmation) |
| **Overall quality** | **MODERATE** — D channel extracts substantial table data (better ratio than p77), but bare numbers and OCR corruption limit utility; identical structural pattern to p77 Figure 24 |

---

## D Channel Table Performance (Audit-Wide Summary)

| Page | Figure | Type | D Spans | Useful | Noise | Useful % | OCR Issues |
|------|--------|------|---------|--------|-------|----------|------------|
| 77 | Figure 24 | SGLT2i trial comparison (large) | 162 | ~20 | ~142 | ~12% | "Slaglifcin" → Sotagliflozin |
| 80 | Figure 26 | Metformin formulations (small) | 8 | 6 | 2 | 75% | None |
| 83 | — | LEADER HR inline | 1 | 1 | 0 | 100% | None |
| **84** | **Figure 28** | **GLP-1 RA trial comparison (large)** | **78** | **~22** | **~56** | **~28%** | **"Lisakenatide" → Lixisenatide** |

**Pattern**: Small structured tables → excellent extraction (75-100%). Large multi-column trial comparison tables → high noise (70-88%) with OCR corruption risk on drug names.

---

## Raw PDF Cross-Check (2026-02-28)

### Methodology
User provided exact Figure 28 table data (all 9 trial rows with complete column data: Trial, Drug, N, %CVD, eGFR criteria, Follow-up, CV outcomes, Kidney outcomes). Cross-checked all ADDED and PENDING spans against exact PDF text.

### Duplicate ADDED Spans Rejected (8)

Parallel agents created near-identical ADDED spans for Figure 28 trial rows. Kept the more detailed version of each and rejected the subset/duplicate:

| # | Span ID Prefix | Content | Reason |
|---|---------------|---------|--------|
| 1 | Duplicate LEADER row summary | LEADER liraglutide trial summary | Subset of more detailed ADDED span |
| 2 | Duplicate SUSTAIN-6 row summary | SUSTAIN-6 semaglutide trial summary | Subset of more detailed ADDED span |
| 3 | Duplicate HARMONY row summary | HARMONY albiglutide trial summary | Subset of more detailed ADDED span |
| 4 | Duplicate REWIND row summary | REWIND dulaglutide trial summary | Subset of more detailed ADDED span |
| 5 | Duplicate ELIXA row summary | ELIXA lixisenatide trial summary | Subset of more detailed ADDED span |
| 6 | Duplicate EXSCEL row summary | EXSCEL exenatide trial summary | Subset of more detailed ADDED span |
| 7 | Duplicate AMPLITUDE-O row summary | AMPLITUDE-O efpeglenatide trial summary | Subset of more detailed ADDED span |
| 8 | Duplicate AWARD-7 row summary | AWARD-7 dulaglutide trial summary | Subset of more detailed ADDED span |

### Garbled PENDING D-Channel Spans Rejected (7)

D-channel table decomposition produced spans with systematic OCR errors — wrong HR values, "%" artifacts in CI notation, cross-contaminated trial data:

| # | Span ID Prefix | OCR Error | Correct Value | Impact |
|---|---------------|-----------|---------------|--------|
| 1 | D-channel HR concat | "HR: 0.75% CI" format | HR 0.75 (95% CI: ...) | % sign OCR artifact in CI notation |
| 2 | D-channel wrong HR | "HR: 0.95" for AMPLITUDE-O CV | HR 0.73 (0.58–0.94) | Wrong HR value entirely |
| 3 | D-channel ELIXA kidney | "P=0.84" for macroalbuminuria | P=0.04 | Clinically dangerous — changes statistical significance |
| 4 | D-channel cross-contam | REWIND showing "3.2 yr" | 5.4 yr follow-up | Cross-contaminated with EXSCEL data |
| 5 | D-channel wrong outcome | "ML" in outcome definition | "MI" (myocardial infarction) | OCR character substitution |
| 6 | D-channel concat HRs | Two HRs merged from adjacent columns | Separate CV and kidney HRs | Cell boundary detection failure |
| 7 | D-channel bare numbers | Decontextualized bare trial values | Need row/column context | Unusable without trial attribution |

### Missing Gaps Added (4)

| Gap ID | Content Added (exact PDF text) | Note | Target KB |
|--------|-------------------------------|------|-----------|
| **G84-A** | Oral semaglutide (PIONEER 6): N=3183, 84.7% established CVD, eGFR ≥60 or CrCl ≥30, 1.3 yr follow-up. CV: 3-point MACE HR 0.79 (0.57–1.11) NS. Kidney: not reported | Figure 28 — PIONEER 6 row entirely missing from all extracted spans | KB-1, KB-4 |
| **G84-B** | ELIXA kidney outcomes: New-onset macroalbuminuria HR 0.84 (0.71–0.99) P=0.04; % change in ACR from baseline significantly lower with lixisenatide | Corrects OCR error P=0.84→P=0.04 — result IS statistically significant for macroalbuminuria | KB-4, KB-16 |
| **G84-C** | REWIND kidney subgroup: consistent benefit across eGFR subgroups for kidney composite outcome (new-onset severely increased albuminuria, sustained ≥30% eGFR decline, or chronic KRT) | REWIND kidney outcome consistency across CKD subgroups — important for class-level kidney benefit evidence | KB-4 |
| **G84-D** | EXSCEL expanded kidney composite: 40% eGFR decline, kidney replacement therapy, or renal death — HR 0.85 (0.73–0.98) P=0.03; pre-specified exploratory analysis | EXSCEL kidney composite P=0.03 (significant in expanded analysis) — balances NS primary kidney finding | KB-4 |

### Remaining PENDING Spans Rejected (11)

All 11 remaining PENDING D-channel spans were decontextualized fragments already covered by ADDED row summaries, or contained OCR errors. Rejected to clear page to 0 PENDING:

| # | ID | Text | Reason |
|---|-----|------|--------|
| 1 | `f38aee6e` | "HR: 1.02; 95% CI: 0.89-1.17" | Bare ELIXA CV HR — covered in ADDED row |
| 2 | `cc46c3b5` | "Median 28.3 mg/g [2.83 mg/mmol]" | Bare SUSTAIN-6 baseline — covered in ADDED row |
| 3 | `c877e61b` | "New-onset severely increased albuminuria and doubling of SCr" | REWIND kidney definition — covered in G84-C |
| 4 | `89f2d151` | "20.7 with eGFR 30 to 59 ml/min per 1.73 ml²..." | CKD distribution fragment, LaTeX artifact — covered in ADDED |
| 5 | `73d2a609` | "19% with moderately increased albuminuria..." | REWIND baseline — covered in ADDED row |
| 6 | `20d1b39d` | "Composite of incident severely increased albuminuria (ACR >300...)" | LEADER kidney definition — covered in ADDED row |
| 7 | `a4d52717` | "...P=0.84..." | **OCR error P=0.84→P=0.04** — corrected in G84-B |
| 8 | `f4f8198f` | "New severely increased albuminuria ACR of >33.9 mg/ml..." | AMPLITUDE-O kidney definition — covered in ADDED row |
| 9 | `61e2f8bd` | "eGFR did not significantly decline (-0.7 ml/min per 1.73 m²)..." | AWARD-7 kidney outcome — covered in ADDED row |
| 10 | `63bef395` | "40% eGFR decline, kidney replacement, or renal death; HR: 0.87..." | EXSCEL kidney composite — covered in G84-D |
| 11 | `64324b60` | "CV death, nonfatal ML, or nonfatal stroke" | **OCR error "ML"→"MI"** — MACE definition covered in all ADDED rows |

### Post-Review State (Final)

| Metric | Before Cross-Check | After Cross-Check | Change |
|--------|-------------------|-------------------|--------|
| **Total spans** | 200 + agent ADDED | 181 total | — |
| **CONFIRMED** | 1 | 0 | -1 (GLP-1 RA mis-confirm cleared by agents) |
| **ADDED (P2-ready)** | 17 (from agents) | 13 | -4 net (-8 dupes + 4 gaps) |
| **PENDING** | 199 | **0** | -199 (all resolved) |
| **REJECTED** | ~185 (agents) | 168 | +26 (15 cross-check + 11 PENDING cleanup) |
| **Completeness** | ~80% (agent pass) | **~98%** (fully resolved) | +18% |

### Key Findings from Cross-Check

1. **PIONEER 6 (oral semaglutide) entirely missing**: The 9th trial row in Figure 28 was not captured by any extraction channel or agent. This is the only oral GLP-1 RA formulation — clinically significant distinction from injectable forms.

2. **ELIXA P-value OCR error is clinically dangerous**: The D-channel extracted P=0.84 (not significant) instead of P=0.04 (significant). This reverses the clinical conclusion — lixisenatide DOES show significant macroalbuminuria benefit. If this error propagated to CQL rules, it would incorrectly exclude ELIXA from kidney benefit evidence.

3. **EXSCEL P=0.03 in expanded analysis**: The primary EXSCEL kidney composite was NS (HR 0.87, 0.73–1.04), but the expanded pre-specified analysis showed significance (P=0.03). Both findings are important for balanced evidence assessment.

4. **D-channel OCR degradation on large tables confirmed**: 7/199 PENDING spans had clinically significant OCR errors — consistent with the pattern from page 77 (Figure 24) where similar garbling occurred on the SGLT2i trial comparison table.
