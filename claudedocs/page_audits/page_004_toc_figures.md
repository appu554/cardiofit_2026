# Page 4 Audit — Table of Contents (Tables & Figures)

| Field | Value |
|-------|-------|
| **Page** | 4 (PDF page S3) |
| **Content Type** | Table of Contents — Tables & Figures listing |
| **Extracted Spans** | 122 total (24 T1, 98 T2) |
| **Channels** | B (Drug Dictionary), C (Grammar/Regex), D (Table Decomp), E (GLiNER NER) |
| **Risk** | No clinical content — ToC only |
| **Disagreements** | 0 |
| **Review Status** | CONFIRMED: 1, PENDING: 121 |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — D-channel double-pass bug documented |

---

## 1. OVERALL VERDICT

**Extraction clinically correct?** → **NO**

This page is purely a Table of Contents listing tables and figures. It contains:
- No dosing thresholds
- No monitoring intervals
- No drug initiation/stop rules
- No lab cutoffs
- No contraindications

All extracted spans classified as T1 or T2 are false positives. The extraction demonstrates systematic misclassification — drug names and figure captions inappropriately promoted to T1.

---

## 2. PAGE-BY-PAGE FINDINGS

### Page 4 (PDF page S3)

**Source PDF content**: Tables & Figures section of the Table of Contents. Lists 6 tables (Table 1–6, pages S97–S104) and 28+ figures (Figure 1–32, pages S19–S83). Also contains abbreviation expansions and the journal footer.

**Confirmed present on page** (correctly located but incorrectly tiered):
- Drug names: "ACEi", "ARB", "ARBs", "SGLT2i", "finerenone", "metformin", "GLP-1 RA"
- Drug class expansions: "GLP-1 receptor agonists"
- Figure captions: "Figure 27. Suggested approach in dosing metformin...", "Figure 9. Serum potassium monitoring during treatment with finerenone", "Figure 7. SGLT2i with established kidney and cardiovascular benefits..."
- Page references: S103, S104, S20, S34, etc.
- Structural labels: "FIGURES", "Clinical question"
- Entity fragments: "serum creatinine", "potassium", "HbA1c", "sodium", "hemoglobin"

All are ToC figure titles, page references, abbreviation expansions, or navigation labels. **None contain actionable clinical guidance.**

**Errors / mis-tiered spans**:

### False-positive T1 spans (24 spans — ALL incorrect)

| Category | Count | Spans | Correct Tier |
|----------|-------|-------|-------------|
| B-channel drug names | 14 | ACEi x2, ARBs x1, ARB x1, SGLT2i x4, finerenone x2, metformin x3, GLP-1 RA x1 | **T3** — Drug names without dosing context |
| D-channel drug class names | 4 | "GLP-1 receptor agonists" x4 | **T3** — Drug class repeated from ToC entries |
| D-channel figure/table captions | 6 | Figure 7, Figure 8, Figure 18, Figure 27, Figure 28, Table 5 | **T3** — ToC references, not the actual figure content |

### False-positive T2 spans (98 spans — ALL incorrect)

| Category | Count | Correct Tier |
|----------|-------|-------------|
| Page number identifiers (S103, S104, S20, S34, etc.) | ~58 | **T3** — Navigation references only |
| "Clinical question" (repeated) | 3 | **T3** — Generic label from PICOM table structure |
| "FIGURES" heading | 2 | **T3** — Structural heading |
| Abbreviation expansions (glucagon-like peptide-1 receptor agonist(s), glycated hemoglobin, dipeptidyl peptidase-4, continuous glucose monitoring, etc.) | 7 | **T3** — Dictionary definitions, not clinical guidance |
| Figure/table captions (Figure 12, Figure 19, Figure 23, Figure 32, Table 2) | 5 | **T3** — ToC references |
| Lab/substance entity fragments (sodium x6, potassium x2, hemoglobin x3, HbA1c x3, serum creatinine x1) | 15 | **T3** — Single-word NER matches from abbreviation text |
| Other (metabolic equivalent, self-monitoring of blood glucose, type 2 diabetes) | 8 | **T3** — Entity fragments and labels |

**Notable**: Span #1 ("ACEi", B channel) has status CONFIRMED — a prior reviewer confirmed a false-positive T1 classification.

**Duplicate spans identified**:
- Page numbers heavily duplicated: S103 x4, S104 x5, S25 x4, S98 x2, S20 x2, S34 x2, etc.
- "GLP-1 receptor agonists" x4 (spans #15, #16, #19, #20)
- "FIGURES" x2 (spans #58, #101)
- "Clinical question" x3 (spans #94, #95, #96)

**D-channel double-pass bug**: The D channel processed the page number table twice, producing near-duplicate span sets:
- First pass: spans #25–57 (S103×2, S104×3, S98, S20, S34, S21, S39, S22, S47, S49, S52, S56, S57, S58, S59, S23, S64–S68, S71–S73, S25×2, S76, S79, S26, S83, FIGURES)
- Second pass: spans #59–93 (S98, S20, S34, S21, S39, S22, S47, S49, S52, S56, S57, [text], S58, S59, S23, S64–S68, S71–S73, Figure 12, S73, S103×2, S25×2, S76, S103, S25, S79, S104×3)
- This accounts for the high duplicate count (S103×4, S104×5, S25×4) and is the same D-channel re-entry bug found on pages 3 and 5.

**Missing critical content**: None — ToC pages contain no extractable clinical facts.

---

## 3. TIER CORRECTIONS

### T1 → T3 (all 24 spans)

**B-channel drug names (14 spans)**:
- `"ACEi"` x2 (B, spans #1, #3) — Current: T1, Correct: T3 — Drug abbreviation only
- `"ARBs"` (B, span #2) — Current: T1, Correct: T3 — Drug class name only
- `"ARB"` (B, span #4) — Current: T1, Correct: T3 — Drug class name only
- `"SGLT2i"` x4 (B, spans #5-7, #10) — Current: T1, Correct: T3 — Drug class name only
- `"finerenone"` x2 (B, spans #8, #9) — Current: T1, Correct: T3 — Drug name only
- `"metformin"` x3 (B, spans #11-13) — Current: T1, Correct: T3 — Drug name only
- `"GLP-1 RA"` (B, span #14) — Current: T1, Correct: T3 — Drug class abbreviation only

**D-channel drug class + captions (10 spans)**:
- `"GLP-1 receptor agonists"` x4 (D, spans #15, #16, #19, #20) — Current: T1, Correct: T3 — Repeated drug class name from ToC
- `"Figure 7. Sodium-glucose cotransporter-2 inhibitors..."` (D, span #17) — Current: T1, Correct: T3 — ToC figure caption
- `"Table 5. KDIGO nomenclature..."` (D, span #18) — Current: T1, Correct: T3 — ToC table caption
- `"Figure 8. Cardiovascular...outcome trials for finerenone"` (D, span #21) — Current: T1, Correct: T3 — ToC figure caption
- `"Figure 27. Suggested approach in dosing metformin..."` (D, span #22) — Current: T1, Correct: T3 — ToC figure caption
- `"Figure 18. Effects of decreased sodium intake..."` (D, span #23) — Current: T1, Correct: T3 — ToC figure caption
- `"Figure 28. Cardiovascular and kidney outcome trials for GLP-1 RA"` (D, span #24) — Current: T1, Correct: T3 — ToC figure caption

### T2 → T3 (all 98 spans)

All 98 T2 spans are non-clinical: page numbers, structural labels, abbreviation expansions, figure captions, and single-word entity fragments. See categories in Section 2 above.

---

## 4. CRITICAL SAFETY FINDINGS

**None.** This page contains:
- No stop/hold criteria
- No dose modification thresholds
- No monitoring requirements
- No eGFR cutoffs
- No potassium cutoffs
- No contraindications

Figure captions reference safety-relevant content (e.g., "potassium monitoring during finerenone", "dosing metformin based on kidney function") but the captions on a ToC page are pointers, not the clinical facts themselves.

---

## 5. L1_RECOVERY GOLD STANDARD SPANS

**None.** This page contains zero extractable clinical guidance. No L1_RECOVERY candidates.

---

## 6. COMPLETENESS SCORE

| Metric | Value |
|--------|-------|
| **True T1 content on page** | 0 (none expected) |
| **True T1 captured** | N/A |
| **False-positive T1** | 24/24 (100%) |
| **False-positive T2** | 98/98 (100%) |
| **Correct tier assignments** | 0/122 (0%) |
| **Noise ratio** | 100% — all 122 spans are non-clinical ToC content |
| **Overall quality** | **POOR / FAIL** |

---

## 7. REVIEWER RECOMMENDATION

| Action | Details |
|--------|---------|
| **Decision** | **ESCALATE** — All 122 spans require re-tiering to T3 or bulk rejection |
| **T1 correction** | All 24 spans → T3 (14 B-channel drug names + 4 D-channel drug class names + 6 D-channel figure/table captions) |
| **T2 correction** | All 98 spans → T3 |
| **Missing content** | None (no clinical content expected on ToC page) |
| **Prior review note** | Span #1 ("ACEi") was CONFIRMED as T1 by a prior reviewer — this confirmation is incorrect and should be overridden |
| **Root cause** | (1) B channel (Drug Dictionary) matches drug names in figure captions without requiring clinical context; (2) D channel (Table Decomp) treats ToC as a data table generating ghost spans; (3) D channel double-pass bug — re-enters same table block producing duplicate span sets (spans #25-57 duplicated in #59-93); (4) No front-matter page detection in pipeline |
| **Pipeline recommendation** | (1) Flag pages 1–7 as front matter in preprocessing to skip extraction on non-clinical pages; (2) Investigate D-channel double-pass table parsing bug (confirmed on pages 3, 4, 5) |
