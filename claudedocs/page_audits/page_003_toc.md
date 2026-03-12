# Page 3 Audit — Table of Contents

| Field | Value |
|-------|-------|
| **Page** | 3 (PDF page S2) |
| **Content Type** | Table of Contents |
| **Extracted Spans** | 57 total (1 T1, 56 T2) |
| **Channels** | D (Table Decomp), F (NuExtract LLM) |
| **Risk** | No clinical content — ToC only |
| **Disagreements** | 0 |
| **Review Status** | PENDING: 57 |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — S19 count corrected, D-channel double-pass bug added, pipeline artifact flagged |

---

## 1. OVERALL VERDICT

**Extraction clinically correct?** → **NO**

This page is a Table of Contents and front-matter metadata. It contains no clinical guidance, no treatment thresholds, no monitoring instructions, and no safety rules.

All 57 extracted spans are structural/navigational text and should be classified as T3.

- 1 false-positive T1 span
- 56 false-positive T2 spans
- 0 correct T1 spans
- 0 correct T2 spans

---

## 2. PAGE-BY-PAGE FINDINGS

### Page 3 (PDF page S2)

**Source PDF content**: Table of Contents listing chapters, supplementary material, methods, biographic disclosures, acknowledgments, and references. Also contains a Kidney International copyright/disclaimer notice and volume metadata.

**Confirmed present on page** (examples):
- "Summary of recommendation statements and practice points"
- "Notice", "Abbreviations and acronyms"
- "Glycemic monitoring and targets in patients with diabetes and CKD"
- "Approaches to management of patients with diabetes and CKD"
- Page identifiers: S3, S7, S8, S9, S10, S11, S12, S13, S14, S16, S17, S19, S29, S55, S63, S75, S89, S97, S106, S115, S117
- "VOL 102 | ISSUE 5S | NOVEMBER 2022"
- "Patients", "Outcomes"

All verified as ToC labels and metadata. **Presence does not qualify them as extractable clinical content.**

**Errors / mis-tiered spans**:

| Category | Count | Current Tier | Correct Tier |
|----------|-------|-------------|-------------|
| "Summary of recommendation statements and practice points" | 1 | T1 | **T3** — ToC heading, not a clinical instruction |
| Page number identifiers (S3, S7, S8, ... S117) | 36 | T2 | **T3** — Navigational references only |
| Section headings (Notice, Abbreviations, chapter titles) | 14 | T2 | **T3** — Structural labels |
| "Outcomes" (repeated) | 3 | T2 | **T3** — ToC column header |
| "Patients" | 1 | T2 | **T3** — Single-word structural label |
| "VOL 102 \| ISSUE 5S \| NOVEMBER 2022" | 1 | T2 | **T3** — Publication metadata |

**Duplicate spans identified**:
- "S97" appears 4 times (spans #19, #20, #40, #41)
- "S19" appears 4 times (spans #13, #21, #34, #42) — two pairs from D-channel double-pass
- "S106" appears 2 times, "S115" appears 2 times, "S117" appears 2 times
- "Notice" appears 2 times
- "Outcomes" appears 3 times

**D-channel double-pass bug**: The D channel processed the ToC table twice, producing near-duplicate span sets:
- First pass: spans #3–21 (S106, S115, S117, S10–S17, S19, S29, S55, S63, S75, S89, S97×2, S19)
- Second pass: spans #22–45 (S3, S7–S9, [text], S10–S17, S19, S29, S55, S63, S75, S89, S97×2, S19, S106, S115, S117)
- The second pass contains the same page numbers in a different order, confirming re-entry into the same table block. This is the same bug documented on page 5.

**Pipeline artifact**: Span #57 contains `<!-- PAGE 3 -->` — an HTML comment from the pipeline's own page-boundary markers. This should never appear as a span. The audit's note of "VOL 102 | ISSUE 5S | NOVEMBER 2022" is correct but the artifact prefix was not flagged.

**Missing critical content**: None — ToC pages contain no extractable clinical facts.

---

## 3. TIER CORRECTIONS

### T1 → T3 (1 span — CRITICAL)

- `"Summary of recommendation statements and practice points"` (D) — Current: T1, Correct: T3 — ToC heading pointing to page S19; contains no drug, dose, threshold, timing, or safety rule

### T2 → T3 (all 56 remaining spans)

- `"S106"`, `"S115"`, `"S117"`, `"S10"`, `"S11"`, etc. (D) — Current: T2, Correct: T3 — Page number references
- `"Notice"`, `"Abbreviations and acronyms"` (D) — Current: T2, Correct: T3 — Section headings
- `"Glycemic monitoring and targets in patients with diabetes and CKD"` (D) — Current: T2, Correct: T3 — Chapter title in ToC, not a clinical instruction
- `"Approaches to management of patients with diabetes and CKD"` (D) — Current: T2, Correct: T3 — Chapter title in ToC
- `"Outcomes"` x3, `"Patients"` x1 (D) — Current: T2, Correct: T3 — ToC structural labels
- `"VOL 102 | ISSUE 5S | NOVEMBER 2022"` (F) — Current: T2, Correct: T3 — Publication metadata

---

## 4. CRITICAL SAFETY FINDINGS

**None.** This page contains:
- No stop/hold criteria
- No dose modification thresholds
- No monitoring requirements
- No eGFR cutoffs
- No contraindications

This is expected for a Table of Contents page.

---

## 5. L1_RECOVERY GOLD STANDARD SPANS

**None.** This page contains zero extractable clinical guidance. No L1_RECOVERY candidates.

---

## 6. COMPLETENESS SCORE

| Metric | Value |
|--------|-------|
| **True T1 content on page** | 0 (none expected) |
| **True T1 captured** | N/A |
| **False-positive T1** | 1/1 (100%) |
| **False-positive T2** | 56/56 (100%) |
| **Correct tier assignments** | 0/57 (0%) |
| **Noise ratio** | 100% — all 57 spans are non-clinical structural content |
| **Overall quality** | **POOR / FAIL** |

---

## 7. REVIEWER RECOMMENDATION

| Action | Details |
|--------|---------|
| **Decision** | **ESCALATE** — All 57 spans require re-tiering to T3 or bulk rejection |
| **T1 correction** | 1 span: "Summary of recommendation statements..." → T3 |
| **T2 correction** | All 56 spans → T3 |
| **Missing content** | None (no clinical content expected on ToC page) |
| **Root cause** | (1) D channel (Table Decomp) parsed the ToC as a clinical data table, generating 56 ghost spans from structural/navigational elements; (2) D channel double-pass bug — re-enters same table block producing duplicate span sets (spans #3-21 duplicated in #22-45); (3) F channel emits pipeline artifact `<!-- PAGE 3 -->` as span |
| **Pipeline recommendation** | (1) Flag pages 1–7 as front matter in preprocessing to skip extraction on non-clinical pages; (2) Investigate D-channel double-pass table parsing bug; (3) Filter `<!-- PAGE N -->` artifacts from span output |
