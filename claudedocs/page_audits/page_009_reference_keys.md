# Page 9 Audit — Reference Keys (GRADE System)

| Field | Value |
|-------|-------|
| **Page** | 9 (PDF page S8) |
| **Content Type** | Reference Keys — GRADE evidence rating system and recommendation strength |
| **Extracted Spans** | 39 (T1: 4, T2: 35) |
| **Channels** | C (Grammar/Regex), D (Table Decomp), F (NuExtract LLM) |
| **Disagreements** | 0 |
| **Review Status** | PENDING: 39 |
| **Risk** | Methodological reference — no direct clinical recommendations |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — span count corrected (59→39), tier breakdown corrected (T1:7→4, T2:52→35), channels corrected (D→C,D,F) |

---

## Source PDF Content

Page 9 presents two reference tables:

**Table 3. Classification for certainty and quality of the evidence (GRADE):**
- Grade A: High — "We are confident that the true effect is close to the estimate"
- Grade B: Moderate — "True effect likely close to estimate but may be substantially different"
- Grade C: Low — "True effect may be substantially different"
- Grade D: Very low — "Estimate is very uncertain"

**Table 5. KDIGO nomenclature for recommendation strength:**
- Level 1 (Strong): "We recommend" — Most patients should receive recommended action
- Level 2 (Weak): "We suggest" — Different choices appropriate for different patients

**Implications for patients, clinicians, and policy:**
- Strong recommendations → can be used for policy/performance measures
- Weak recommendations → require substantial debate before policy implementation

---

## Extraction Analysis

### Channel Breakdown (39 spans)

| Channel | Spans | Tier | Notes |
|---------|-------|------|-------|
| **D (Table Decomp)** | #1–35 (35 spans) | 3 T1 + 32 T2 | GRADE table cells extracted individually |
| **F (NuExtract LLM)** | #4 (1 span) | 1 T1 | Pipeline artifact `<!-- PAGE 9 -->` heading |
| **C (Grammar/Regex)** | #36–39 (4 spans) | 4 T2 | Matched "Level 1" and "Level 2" as regex patterns |

### What Was Extracted (39 spans)

| Category | Count | Channel | Current Tier | Span #s |
|----------|-------|---------|-------------|---------|
| 'Level 1, strong "We recommend"' | 2 | D | T1 | #1, #2 (duplicate) |
| Policy implication statement | 1 | D | T1 | #3 |
| Pipeline artifact heading | 1 | F | T1 | #4 |
| "Grade" column header | 7 | D | T2 | #8–12, #23, #24 |
| Single-letter grade labels (A, B, c, D) | 10 | D | T2 | #14–22, #31 |
| "Meaning" column header | 3 | D | T2 | #5, #27, #28 |
| "Policy" column header | 3 | D | T2 | #6, #7, #29 |
| Evidence quality descriptions | 3 | D | T2 | #25, #30, #33 |
| Patient/clinician implication text | 2 | D | T2 | #26, #32 (whitespace variant duplicate) |
| "Very low" label | 2 | D | T2 | #34, #35 |
| "Level 1" / "Level 2" regex matches | 4 | C | T2 | #36–39 |

### T1 Span Detail

| # | Channel | Text | Assessment |
|---|---------|------|------------|
| 1 | D | Level 1, strong "We recommend" | **FALSE POSITIVE** — methodology label, not a clinical recommendation |
| 2 | D | Level 1, strong "We recommend" | **FALSE POSITIVE** — duplicate of #1 |
| 3 | D | The recommendation can be evaluated as a candidate for developing a policy or a performance measure. | **FALSE POSITIVE** — describes implication of strong recs generally |
| 4 | F | <!-- PAGE 9 --> NOMENCLATURE AND DESCRIPTION FOR RATING GUIDELINE RECOMMENDATIONS | **PIPELINE ARTIFACT** — HTML comment + section heading |

### Tier Assignment Issues

| Current Tier | Count | Should Be | Reason |
|-------------|-------|-----------|--------|
| **T1** | 4 | **T3** | Methodology labels (#1–3) and pipeline artifact (#4) — none are actual clinical recommendations |
| **T2** | 35 | **T3** | Column headers, grade letters, evidence descriptions — all describe GRADE methodology, not clinical actions |
| **T3** | 0 | — | — |

### Severity: MODERATE TIER MISASSIGNMENT

All 39 spans should be T3 (Informational — evidence grades, study methodology).

**Key distinction**: This page describes HOW recommendations are graded, not the recommendations themselves. The phrases "We recommend" and "We suggest" here are LABELS in a methodology table, not actual clinical directives.

---

## Specific Problems

### Problem 1: Methodology Labels Classified as T1
Spans #1–2 ('Level 1, strong "We recommend"') are labels in the GRADE explanation table describing the nomenclature system, not actual treatment recommendations. Span #3 describes a general implication of strong recommendations — it's process guidance, not a clinical directive.

### Problem 2: Pipeline Artifact as T1
Span #4 is an F-channel extraction of "<!-- PAGE 9 -->" HTML comment concatenated with the section heading. This is a pipeline processing marker that should never be classified as clinical content.

### Problem 3: D-Channel Partial Double-Pass
Grade letters appear in two overlapping sets: first pass (#14–17: A, B, c, D) and second pass (#18–22: c, A, B, c, D) with an extra 'c'. Column headers "Grade" (7×), "Meaning" (3×), "Policy" (3×) are extracted from multiple table positions.

### Problem 4: Single-Letter Extractions
Letters "A", "B", "c" (lowercase), "D" extracted as standalone spans (10 total). These are evidence grade letters meaningless without context. Note lowercase "c" at #16, #18, #21 — possible OCR inconsistency (should be uppercase "C").

### Problem 5: Whitespace-Variant Duplicate
Span #32 is a whitespace-mangled duplicate of #26 — same patient implication text with extra spaces and line breaks from PDF extraction.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Page Decision** | **FLAG** — All 39 spans are GRADE methodology, not clinical facts |
| **T1 spans** | All 4 should be re-tiered to T3 (methodology labels and pipeline artifact) |
| **T2 spans** | All 35 should be re-tiered to T3 (informational/methodological) |
| **Missing content** | None — GRADE system descriptions are informational only |
| **Root cause** | 1. D channel decomposes methodology tables into fragments without understanding they describe process, not clinical actions; 2. F channel extracts pipeline markers as content; 3. C channel regex matches "Level 1/2" without context that these are nomenclature labels |

---

## Completeness Score

| Metric | Score |
|--------|-------|
| **Extraction completeness** | N/A (no clinical recommendations on this page) |
| **Tier accuracy** | 0% (0/39 correctly tiered; all should be T3) |
| **False positive rate** | 100% for T1 (4/4 are methodology labels or artifacts) |
| **Missing T1 content** | 0 (none expected) |
| **Missing T2 content** | 0 (none expected) |
| **D-channel double-pass** | Confirmed — grade letters and column headers extracted in overlapping passes |
| **Pipeline artifacts** | 1 (span #4: `<!-- PAGE 9 -->` heading) |
| **Overall page quality** | FAIL — all tiers wrong, but content IS extracted |
