# Page 6 Audit — Table of Contents (Supplementary Material cont.)

| Field | Value |
|-------|-------|
| **Page** | 6 (PDF page S5) |
| **Content Type** | Table of Contents — Supplementary Material continuation |
| **Extracted Spans** | 15 (T1: 5, T2: 10) |
| **Channels** | B (Drug Dictionary), C (Grammar/Regex), D (Table Decomp), F (NuExtract LLM) |
| **Disagreements** | 0 |
| **Review Status** | PENDING: 15 |
| **Risk** | No clinical content — ToC only |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — previous audit had CRITICAL span count error (44→15). Complete rewrite required. |

---

## 1. OVERALL VERDICT

**Extraction clinically correct?** → **NO**

This page is a Table of Contents continuation listing Supplementary Material. It contains no clinical guidance, no treatment thresholds, no monitoring instructions, and no safety rules.

All 15 extracted spans are non-clinical ToC content and should be classified as T3.

- 5 false-positive T1 spans
- 10 false-positive T2 spans
- 0 correct T1 spans
- 0 correct T2 spans

**Previous audit error**: The prior audit claimed 44 spans (T1:16, T2:28) with content describing Tables S24–S62. Raw span data shows only **15 spans**. The prior audit's extraction analysis (DPP-4 inhibitors ×8, SoF table duplication ×15, etc.) does not match the actual extraction output. Complete rewrite was necessary.

---

## 2. PAGE-BY-PAGE FINDINGS

### Page 6 (PDF page S5)

**Source PDF content**: Supplementary Material listing continuation. Lists SoF tables for drug comparisons, protein diet, sodium intake, physical activity, and additional outcomes.

**What was actually extracted (15 spans)**:

| # | Channel | Tier | Span Text | Assessment |
|---|---------|------|-----------|------------|
| 1 | B | T1 | eplerenone | Drug name from ToC → **T3** |
| 2 | B | T1 | eplerenone | Duplicate → **T3** |
| 3 | B | T1 | eplerenone | Duplicate → **T3** |
| 4 | D | T1 | Table S91. SoF table: Kidney transplant recipients...glitazone and insulin versus placebo and insulin | SoF table title from ToC → **T3** |
| 5 | D | T1 | Table S89. SoF table: Kidney transplant recipients...DPP-4 inhibitor versus placebo | SoF table title from ToC → **T3** |
| 6 | B | T2 | RAS inhibitor | Drug class name from ToC → **T3** |
| 7 | B | T2 | RAS inhibitor | Duplicate → **T3** |
| 8 | B | T2 | Calcium channel blocker | Drug class name from ToC → **T3** |
| 9 | D | T2 | Moderate | Evidence quality label fragment → **T3** |
| 10 | D | T2 | Moderate | Duplicate → **T3** |
| 11 | F | T2 | `<!-- PAGE 6 -->` | **Pipeline artifact** — should never be a span → **REJECT** |
| 12 | C | T2 | Potassium | Lab name from ToC → **T3** |
| 13 | C | T2 | serum creatinine | Lab name from ToC → **T3** |
| 14 | C | T2 | potassium | Duplicate → **T3** |
| 15 | C | T2 | 0.8 g/kg | Protein threshold from SoF table title context only → **T3** |

---

## 3. TIER CORRECTIONS

### T1 → T3 (5 spans)

- `"eplerenone"` ×3 (B, spans #1–3) — Drug name only, no dosing/threshold/safety context
- `"Table S91..."` (D, span #4) — SoF table title in ToC, not actual trial data
- `"Table S89..."` (D, span #5) — SoF table title in ToC, not actual trial data

### T2 → T3 or REJECT (10 spans)

- `"RAS inhibitor"` ×2 (B, spans #6–7) — Drug class name only
- `"Calcium channel blocker"` (B, span #8) — Drug class name only
- `"Moderate"` ×2 (D, spans #9–10) — Evidence quality label fragment
- `"<!-- PAGE 6 -->"` (F, span #11) — Pipeline artifact → **REJECT**
- `"Potassium"` / `"potassium"` (C, spans #12, #14) — Lab name without threshold
- `"serum creatinine"` (C, span #13) — Lab name without threshold
- `"0.8 g/kg"` (C, span #15) — Protein threshold from SoF table title; clinically meaningful value but ToC context only

---

## 4. SPECIFIC PROBLEMS

### Problem 1: Pipeline Artifact
Span #11 contains `<!-- PAGE 6 -->` — an HTML comment from the pipeline's own page-boundary markers. This should never appear as a span. Same bug found on pages 3 and 5.

### Problem 2: "0.8 g/kg" Protein Threshold in ToC Context
The C channel extracted "0.8 g/kg" which IS a clinically meaningful protein restriction threshold. However, in this ToC context, it's from the title of a SoF table, not the actual clinical recommendation. The real clinical data is inside the table on another page.

### Problem 3: SoF Table Numbers vs. Page Content
The D channel extracted Tables S89 and S91 (kidney transplant recipients), which according to the page-by-page ToC layout belong later in the supplementary material. This may indicate the D channel is extracting content across page boundaries, or the page content extends further than originally described.

---

## 5. CRITICAL SAFETY FINDINGS

**None.** This page contains:
- No stop/hold criteria
- No dose modification thresholds
- No monitoring requirements
- No eGFR cutoffs
- No contraindications

---

## 6. L1_RECOVERY GOLD STANDARD SPANS

**None.** This page contains zero extractable clinical guidance. No L1_RECOVERY candidates.

---

## 7. COMPLETENESS SCORE

| Metric | Value |
|--------|-------|
| **True T1 content on page** | 0 (none expected) |
| **True T1 captured** | N/A |
| **False-positive T1** | 5/5 (100%) |
| **False-positive T2** | 10/10 (100%) |
| **Correct tier assignments** | 0/15 (0%) |
| **Noise ratio** | 100% — all 15 spans are non-clinical ToC content |
| **Pipeline artifacts** | 1 (`<!-- PAGE 6 -->`) |
| **Overall quality** | **POOR / FAIL** — requires bulk rejection |

---

## 8. REVIEWER RECOMMENDATION

| Action | Details |
|--------|---------|
| **Decision** | **ESCALATE** — All 15 spans require re-tiering to T3 or bulk rejection |
| **T1 correction** | All 5 → T3 (3 eplerenone drug names + 2 SoF table titles) |
| **T2 correction** | All 10 → T3 or REJECT (pipeline artifact, drug class names, lab names, evidence labels) |
| **Missing content** | None (no clinical content expected on ToC page) |
| **Root cause** | (1) B channel matches drug names without clinical context; (2) D channel treats ToC as data table; (3) F channel emits pipeline artifact `<!-- PAGE 6 -->` as span; (4) No front-matter page detection in pipeline |
| **Pipeline recommendation** | Flag pages 1–7 as front matter in preprocessing to skip extraction on non-clinical pages |
