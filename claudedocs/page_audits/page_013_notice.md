# Page 13 Audit — Notice / Disclaimer

| Field | Value |
|-------|-------|
| **Page** | 13 (PDF page S12) |
| **Content Type** | Legal notice and disclaimer |
| **Extracted Spans** | 3 |
| **Channels** | F (NuExtract LLM), E (GLiNER NER), C (Grammar/Regex) |
| **Risk** | No clinical content — legal text |
| **Disagreements** | 0 |
| **Review Status** | PENDING: 3 |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | 2026-02-25 — counts verified against raw extraction data |

---

## Source PDF Content

Page 13 contains:
- **Section heading**: "SECTION I: USE OF THE CLINICAL PRACTICE GUIDELINE"
- Standard legal disclaimer about how the guideline is intended for use by healthcare professionals
- Disclaimer that individual patient decisions should be made by treating physicians
- Copyright and usage restrictions

---

## Extraction Analysis

### What Was Extracted (3 spans)

| # | Text | Channel | Current Tier | Should Be |
|---|------|---------|-------------|-----------|
| 1 | "<!-- PAGE 13 --> SECTION I: USE OF THE CLINICAL PRACTICE GUIDELINE" | F | T2 | T3 (heading + pipeline artifact) |
| 2 | "avoid" | E | T2 | REJECT (single word from legal text, no clinical context) |
| 3 | "annually" | C | T2 | T3 (temporal word from legal text, not a monitoring interval) |

### Tier Assignment Issues

| Current Tier | Count | Should Be | Reason |
|-------------|-------|-----------|--------|
| **T1** | 0 | — | Correct |
| **T2** | 3 | **T3 or REJECT** | Legal disclaimer text has no clinical monitoring or dosing content |

### Severity: LOW (only 3 spans)

---

## Specific Problems

### Problem 1: "avoid" as Standalone T2
The E (GLiNER NER) channel extracted the single word "avoid" — likely from the legal text advising how to use the guideline. Without clinical context (avoid which drug? in what condition?), this is meaningless.

### Problem 2: "annually" Without Context
The C (Grammar/Regex) channel matched "annually" as a temporal word. In the legal disclaimer context, this likely refers to guideline review frequency, not a clinical monitoring interval.

### Problem 3: Pipeline Artifact
"<!-- PAGE 13 -->" HTML comment in first span.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Page Decision** | **FLAG** — Legal disclaimer, no clinical content |
| **T2 spans** | All 3 → T3 or REJECT |
| **Missing content** | None expected |
| **Root cause** | E/C channels match individual words without semantic context |

---

## Completeness Score

| Metric | Score |
|--------|-------|
| **Extraction completeness** | N/A (no clinical content) |
| **Tier accuracy** | 0% (0/3 correctly tiered) |
| **False positive rate** | 100% |
| **Overall page quality** | FAIL — but minimal impact (3 spans) |
