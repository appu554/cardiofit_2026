# Page 5 Audit — Table of Contents (Figures cont. + Supplementary Material)

| Field | Value |
|-------|-------|
| **Page** | 5 (PDF page S4) |
| **Content Type** | Table of Contents — Figures continuation + Supplementary Material listing |
| **Extracted Spans** | 62 (T1: 3, T2: 59) |
| **Channels** | D (Table Decomp), F (NuExtract LLM), C (Grammar/Regex) |
| **Risk** | No clinical content — ToC only |
| **Disagreements** | 0 |
| **Review Status** | PENDING: 62 |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans and external audit — span count corrected from 57→62 |

---

## Source PDF Content

Page 5 continues the Table of Contents from page 4:

**FIGURES (cont.):**
- Figure 29. Dosing for GLP-1 RA and dose modification for CKD (S27)
- Figure 30–36: Self-management, meta-analysis, integrated care, chronic care model, team-based care, search yield (S91–S102)

**SUPPLEMENTARY MATERIAL:**
- Appendix A: Search strategies (Table S1)
- Appendix B: Concurrence with IOM standards (Tables S2–S3)
- Appendix C: Data supplement — Summary of Findings (SoF) tables:
  - Table S4–S10: Drug class SoF tables (ACEi, ARB, SGLT2i, MRA, steroidal MRA, nonsteroidal MRA, smoking cessation)
  - Table S11–S13: Tight glycemic control at various HbA1c targets
  - Table S14–S15: Alternative biomarkers and CGM/SMBG
  - Table S16: Low-protein diet
  - Table S17–S20: Salt intake (T1D, T2D, habitual low/high salt)
  - Table S21–S22: Exercise interventions
  - Table S23: GLP-1 RA outcomes

---

## Extraction Analysis

### What Was Extracted (62 spans — verified against raw span table)

| Category | Count | Channel | Current Tier | Issue |
|----------|-------|---------|-------------|-------|
| **T1 false positives** | | | | |
| Lo et al. Cochrane study citation | 2 | D | **T1** | Bibliographic reference — should be T3 |
| "Table S23. Soft: GLP-1 RA versus placebo" | 1 | D | **T1** | SoF table title in ToC — should be T3 |
| **T2 false positives** | | | | |
| SoF table titles (Table S11–S23, incl. duplicates) | 10 | D | T2 | ToC references, not actual SoF data. S11/S12/S13 each extracted twice (#4/#57, #5/#56, #6/#53). Table S20 at #50. |
| Figure captions (Figure 30, 34, 35) | 3 | **F only** | T2 | F-channel only (#59-61) — D channel did NOT extract standalone figure captions on this page |
| Page number strings (S27, S28, S91, S92, S94, S95, S102, S11.–S23.) | ~34 | D | T2 | Navigation page numbers — D channel extracted the page number block **twice** (double-pass bug: #10-22 and #39-52) |
| Abbreviation expansions | 3 | D | T2 | #54 "glomerular filtration rate", #55 "estimated glomerular filtration rate", #62 "hemoglobin" (C channel) |
| Pipeline artifact | 1 | F | T2 | "<!-- PAGE 5 -->" (#58) — should never be a span |
| Other (comparator labels, metadata) | 5 | D | T2 | "Standard of care" / "Placebo/standard of care" phrases, "Supplementary File (PDF)" |

### Tier Assignment Issues

| Current Tier | Count | Should Be | Reason |
|-------------|-------|-----------|--------|
| **T1** | 3 | **T3** | Study citation + SoF table title from ToC — no actual safety data |
| **T2** | 59 | **T3 or REJECT** | Figure/table ToC references, page numbers, labels — no clinical accuracy content |
| **T3** | 0 | — | — |

### Severity: CRITICAL TIER MISASSIGNMENT

All 62 spans incorrectly tiered. This is page 3 of the Table of Contents.

---

## Specific Problems

### Problem 1: Study Citations Classified as T1
Two spans containing "Lo et al. Insulin and glucose-lowering agents for treating people with diabetes and chronic kidney disease. Cochrane Dat..." were tagged T1. This is a **bibliographic reference** in a ToC, not a clinical safety fact.

### Problem 2: SoF Table Title as T1
"Table S23. Soft: GLP-1 RA versus placebo/standard of care" — this is a supplementary material table title, not actual clinical trial outcome data. The actual safety-relevant data is inside the table (on a different page), not in its ToC entry.

### Problem 3: Pipeline Artifact Extracted
"<!-- PAGE 5 -->" is an HTML comment from the pipeline's own processing. This should never appear as a span.

### Problem 4: Character Encoding Loss (≤ symbol dropped)
SoF table titles show "HbA1c  7%", "HbA1c  6.5%", "HbA1c  6%" — the **≤** comparison operator was stripped during extraction, leaving double-spaces. On this ToC page the impact is nil (all T3 anyway), but the same encoding bug on clinical content pages could render threshold spans ambiguous (e.g., "eGFR  20" losing whether it was ≥20 or <20). **Flag as pipeline-wide concern.**

### Problem 5: D-Channel Double-Pass Extraction
The D channel processed the supplementary material ToC table **twice**, producing near-complete duplicate sets:
- Page numbers S11.–S23. appear at spans #10-22 **and again** at #39-52
- SoF table titles duplicated: Table S11 (#6 and #57), Table S12 (#5 and #56), Table S13 (#4 and #53)
- Page numbers S27, S91, S92, S94, S95, S28, S102 each appear twice across the two passes

This accounts for the 5 "missing" spans (62 actual vs. 57 previously reported) and indicates a **double-pass bug in the D channel's table parsing** — the ToC structure caused the parser to re-enter the same table block.

### Problem 6: F-Channel Figure Captions (Not Cross-Channel Duplicates)
Figures 30, 34, and 35 were extracted by the **F channel only** (spans #59-61). Previously reported as D+F cross-channel duplicates — this was incorrect. The D channel does not produce standalone figure caption spans on this page; it extracts SoF table titles instead.

---

## Critical Safety Findings

**None.** This page contains:
- No drug dosing instructions
- No eGFR thresholds
- No lab cutoffs
- No stop/hold rules
- No monitoring timelines
- No contraindications

SoF table titles reference clinical trial outcomes (e.g., "GLP-1 RA versus placebo") but these are navigation pointers to data elsewhere, not the clinical facts themselves.

---

## L1_RECOVERY Gold Standard Spans

**None.** This page contains zero extractable clinical guidance. No L1_RECOVERY candidates.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Page Decision** | **ESCALATE** — All 62 spans require re-tiering to T3 or bulk rejection |
| **T1 spans** | All 3 → T3 (2 study citations + 1 SoF table title) |
| **T2 spans** | All 59 → T3 or REJECT |
| **Missing content** | None — no clinical facts exist on this ToC page |
| **Root cause** | (1) Same as pages 3-4: D channel (Table Decomp) treats ToC as clinical data table; (2) D channel double-pass bug — re-enters same table block producing duplicate span sets; (3) F channel emits pipeline artifact `<!-- PAGE 5 -->` as span; (4) Character encoding loss strips ≤ operator |
| **Pipeline recommendation** | Flag pages 1–7 as front matter in preprocessing to skip extraction on non-clinical pages. Investigate D-channel double-pass table parsing bug. Add ≤/≥/< symbol preservation check. |

---

## Completeness Score

| Metric | Score |
|--------|-------|
| **True T1 content on page** | 0 (none expected) |
| **True T1 captured** | N/A |
| **False-positive T1** | 3/3 (100%) |
| **False-positive T2** | 59/59 (100%) |
| **Correct tier assignments** | 0/62 (0%) |
| **Noise ratio** | 100% — all 62 spans are non-clinical ToC content |
| **Missing T1 content** | 0 (none expected) |
| **Missing T2 content** | 0 (none expected) |
| **Overall page quality** | **POOR / FAIL** — requires bulk rejection |
