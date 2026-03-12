# Page 93 Audit — Chapter 5: Figure 32 Forest Plots (338 Spans — Highest Noise Page)

## Page Identity
- **PDF page**: S92 (Chapter 5 — www.kidney-international.org)
- **Content**: Figure 32 — 6-panel forest plot (SBP, DBP, eGFR, HbA1c, SM activity, HRQOL) from Zimbudzi et al. 2018 systematic review
- **Clinical tier**: T3 (Informational — meta-analysis statistical visualization)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | **450** (highest in entire audit) |
| T1 (Patient Safety) | 0 |
| T2 (Clinical Accuracy) | 450 |
| T3 (Informational) | 0 |
| Channels present | C (Grammar/Regex), D (Table Decomp) |
| Genuinely correct T2 | 0 |
| Tier accuracy | 0% (0/450) |
| Disagreements | 0 |
| Review Status | FINAL: 7 ADDED, 0 PENDING, 338 REJECTED |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — count corrected UPWARD 338→450 |
| Raw PDF Cross-Check | 2026-02-28 — 0 agent spans kept, 338 rejected (all D/C channel noise), 7 gaps added (G93-A–G93-G) |

## Channel Breakdown
| Channel | Count | Confidence | Content |
|---------|-------|------------|---------|
| D (Table Decomp) | 334 | 92% | Forest plot cell data (numbers, labels, statistics) |
| C (Grammar/Regex) | 4 | 85% | Lab parameter names from figure axes |

## Span Content Analysis

### D Channel — Forest Plot Cell Decomposition (334 spans)
The D channel treated Figure 32's 6-panel forest plot as a series of tables and extracted every individual cell. The top extracted terms reveal the pattern:

| Extracted Text | Count | Source |
|---------------|-------|--------|
| "Total" | 24× | Row label in each forest plot panel |
| "Mean" | 24× | Column header in each panel |
| "Control" | 12× | Comparator arm label |
| "Weight" | 12× | Weight column header |
| "Study or subgroup" | 7× | Row header |
| "Total (95% CI)" | 6× | Summary row label |
| Individual numbers (39, 41, 50, 26, 81, 82, 38, 70, etc.) | ~100× | Sample sizes, means, SDs |
| CI ranges (-2.30, -2.70, -0.30, 7.4, etc.) | ~80× | Effect sizes and CIs |
| Heterogeneity statistics | ~6× | I², Tau², chi-squared values |
| Study names (Scherpibier-de-Haan 2013, Chan 2009, Williams 2012) | ~13× | Author-year citations |
| Percentages (15.0%, etc.) | ~10× | Weight percentages |

**All 334 D channel spans should be T3** — these are meta-analysis statistical results from a figure, not clinical thresholds, dosing limits, or safety parameters.

### C Channel — Lab Parameter Names (4 spans)
| # | Text | Conf | Correct Tier | Issue |
|---|------|------|-------------|-------|
| 1 | "HbA1c" | 85% | T3 | Figure axis label |
| 2 | "eGFR" | 85% | T3 | Figure axis label |
| 3 | "hemoglobin" | 85% | T3 | Figure axis label (glycated hemoglobin) |
| 4 | "HbA1c" | 85% | T3 | Figure axis label (duplicate) |

## PDF Source Content Analysis

### Content Present on Page
The entire page is **Figure 32** — a 6-panel forest plot showing outcomes for self-management education programs in diabetes+CKD:
- (a) SBP
- (b) DBP
- (c) eGFR
- (d) HbA1c (%)
- (e) Self-management activity
- (f) Health-related quality of life

Source: Zimbudzi E, Lo C, Misso ML, et al. "Effectiveness of self-management support interventions for people with comorbid diabetes and chronic kidney disease: a systematic review and meta-analysis." Syst Rev. 2018;7:84.

Studies included: Scherpibier-de-Haan 2013, Chan 2009, Williams 2012, and others (approximately 5-7 studies per panel).

### What Should Have Been Extracted
| Content | Correct Tier | Extracted? |
|---------|-------------|------------|
| Figure 32 caption/title | T3 | YES (in PDF text) |
| Overall effect sizes per panel | T3 | Partially (individual CI numbers scattered across 334 spans) |
| Study citations | T3 | Partially (author names extracted as individual spans) |
| Heterogeneity statistics | T3 | YES (but as separate T2 spans) |

## Critical Pipeline Issue: D Channel Figure Explosion

### Problem
Page 93 demonstrates the most severe D channel noise pattern in the audit. The Table Decomp channel:
1. **Cannot distinguish figures from tables** — treated a 6-panel forest plot as 6 tables
2. **Extracted every cell individually** — each number, label, and statistic became a separate span
3. **No deduplication** — "Total" appears 24 times, "Mean" 24 times, "Control" 12 times
4. **No semantic grouping** — individual numbers like "39", "41", "50" are meaningless without their row/column context

### Impact
- **338 spans for a single figure page** — this is 6.8% of all spans (4978 total) for one page
- **100% noise** — every single span is either a decontextualized number or a repeated label
- **Reviewer burden** — if a pharmacist reviewed this page, they'd need to evaluate 338 individual meaningless spans

### Recommendation
D channel needs a **figure detection heuristic** to skip forest plots, bar charts, and other statistical visualizations. Possible signals:
- Repeated structural labels ("Study or subgroup", "Total", "Weight", "Mean")
- High density of numerical cells with no drug names or clinical terms
- Forest plot markers ("Favors intervention", "Favors control", "I²", "Tau²")

## Cross-Page Patterns

### Comparison with Page 92
Pages 92-93 are companion pages (Figure 31 on p92, Figure 32 on p93). But:
- Page 92: 9 D-channel CI spans + 3 F + 1 C = 13 total
- Page 93: 334 D-channel cell spans + 4 C = 338 total
- The 26× difference is because page 93's figure had its data rendered as accessible table-like structures in the PDF, while page 92's figure did not

### Worst Pages by Span Count (Audit-Wide)
1. **Page 93: 338 spans** — Figure 32 forest plots (THIS PAGE)
2. Page 45: 364 spans — (from earlier audit)
3. Page 88: 202 spans — GLP-1 RA dosing + Figure 29
4. Page 81: 210 spans — Chapter 4 opening
5. Page 84: 160 spans — Metformin table

## Quality Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| Extraction completeness | N/A | Figure page — no clinical text to extract |
| Tier accuracy | 0% | All 338 spans should be T3, classified as T2 |
| Clinical safety risk | NONE | Meta-analysis figure, zero patient safety content |
| Channel diversity | LOW | 99% D channel |
| Noise level | **EXTREME** | 338/338 spans are noise (100%) |
| Pipeline impact | HIGH | Single page accounts for 6.8% of all job spans |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. Entire page is a statistical figure. The 338 noisy spans are a D channel pipeline issue, not a clinical review concern.

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Review State
- **Total spans**: 338
- **ADDED**: 0
- **PENDING**: 0
- **REJECTED**: 338 (all pre-existing agent spans were D channel cell fragments from forest plot)

### Agent Spans Rejected (338)
All 338 original extraction spans were decontextualized forest plot cell data from D channel (334 spans: individual numbers, labels, column headers) and C channel (4 spans: bare lab parameter names "HbA1c" ×2, "eGFR", "hemoglobin"). The D channel treated the 6-panel forest plot as tables, extracting every cell individually — this is the worst-case D channel noise pattern in the entire audit.

### Gaps Added (7) — Pooled Meta-Analysis Results

| # | ID | Gap Text | Note |
|---|-----|----------|------|
| G93-A | — | Figure 32 \| Forest plots showing outcomes for people with diabetes and chronic kidney disease (CKD) undergoing self-management (SM) education programs. (a) Systolic blood pressure (SBP), (b) diastolic blood pressure (DBP), (c) estimated glomerular filtration rate (eGFR), (d) glycated hemoglobin (HbA1c; %), (e) SM activity, and (f) health-related quality of life (HRQOL). CI, confidence interval; df, degrees of freedom; IV, inverse variance. Reproduced from Zimbudzi E, Lo C, Misso ML, et al. Syst Rev. 2018;7:84. | Figure 32 caption + source |
| G93-B | — | SBP: Total (95% CI) n=239 vs n=237, MD –4.26 [–7.81, –0.70]; I²=0% | **Significant** — SM reduces SBP |
| G93-C | — | DBP: Total (95% CI) n=169 vs n=167, MD –2.70 [–6.19, 0.78]; I²=41% | Non-significant for DBP |
| G93-D | — | eGFR: Total (95% CI) n=159 vs n=149, MD 0.59 [–4.12, 5.29]; I²=60% | Non-significant for eGFR |
| G93-E | — | HbA1c: Total (95% CI) n=311 vs n=284, MD –0.46 [–0.83, –0.09]; I²=80% | **Significant** — SM reduces HbA1c, high heterogeneity |
| G93-F | — | SM activity: Total (95% CI) n=130 vs n=129, SMD 0.56 [0.15, 0.97]; I²=63% | **Significant** — SM improves self-management activity |
| G93-G | — | HRQOL: Total (95% CI) n=73 vs n=65, SMD –0.03 [–0.36, 0.31]; I²=0% | Non-significant for HRQOL |

### Post-Review State
- **Total spans**: 345
- **ADDED**: 7 (all gap additions — 0 agent spans kept)
- **PENDING**: 0
- **REJECTED**: 338 (all original agent noise)
- **P2-ready facts**: 7

### Clinical Significance Summary
Of 6 outcomes measured for SM interventions in diabetes+CKD (Zimbudzi 2018):
- **3 significant**: SBP (–4.26 mmHg), HbA1c (–0.46%), SM activity (SMD 0.56)
- **3 non-significant**: DBP, eGFR, HRQOL
- **Heterogeneity concern**: HbA1c (I²=80%) and SM activity (I²=63%) show substantial heterogeneity

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G93-A | KB-7 (Terminology) | Figure source attribution, outcome definitions |
| G93-B–G93-D | KB-16 (Monitoring) | SBP/DBP/eGFR monitoring outcomes from SM interventions |
| G93-E | KB-16 (Monitoring) | HbA1c monitoring outcome from SM interventions |
| G93-F–G93-G | KB-4 (Safety) | SM activity and HRQOL patient-reported outcomes |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI → file-saved snapshot → Python extraction (338 spans too large for direct snapshot)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-02-28 (claude-auditor via API)
- **Page in sequence**: 93 of 126
