# Page 8 Audit — KDIGO Executive Committee

| Field | Value |
|-------|-------|
| **Page** | 8 (PDF page S7) |
| **Content Type** | KDIGO Executive Committee listing |
| **Extracted Spans** | 5 |
| **Channels** | F (NuExtract LLM) |
| **Risk** | No clinical content — organizational listing |
| **Disagreements** | 0 |
| **Review Status** | PENDING: 5 |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | 2026-02-25 — counts verified against raw extraction data |

---

## Source PDF Content

Page 8 lists the KDIGO Executive Committee members:
- Garabed Eknoyan, MD / Norbert Lameire, MD, PhD — Founding Co-Chairs
- David C. Wheeler, MD, FRCP — Immediate Past Co-Chair
- Michel Jadoul, MD — KDIGO Co-Chair
- Wolfgang C. Winkelmayer, MD, MPH, ScD — KDIGO Co-Chair

This is purely organizational/administrative content.

---

## Extraction Analysis

### What Was Extracted (5 spans)

| Category | Count | Channel | Current Tier | Issue |
|----------|-------|---------|-------------|-------|
| "<!-- PAGE 8 --> KDIGO EXECUTIVE COMMITTEE" | 1 | F | T2 | Pipeline artifact + heading |
| Committee member names with titles | 4 | F | T2 | Person names and credentials |

### Tier Assignment Issues

| Current Tier | Count | Should Be | Reason |
|-------------|-------|-----------|--------|
| **T1** | 0 | — | — |
| **T2** | 5 | **T3** | Names and credentials — no clinical content |
| **T3** | 0 | — | — |

### Severity: MINOR (low span count, correct that no T1)

---

## Specific Problems

### Problem 1: Pipeline Artifact
"<!-- PAGE 8 -->" HTML comment appears at the start of the first span. This is a pipeline processing marker that should never be extracted.

### Problem 2: Non-Clinical Content as T2
Committee member names are organizational metadata, not monitoring intervals or lab thresholds (T2 criteria). Should be T3 at most.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Page Decision** | **FLAG** — All 5 spans are organizational content |
| **T2 spans** | All 5 should be re-tiered to T3 |
| **Missing content** | None expected |
| **Root cause** | F channel extracts any structured text; no clinical context filter |

---

## Completeness Score

| Metric | Score |
|--------|-------|
| **Extraction completeness** | N/A (no clinical content) |
| **Tier accuracy** | 0% (0/5 correctly tiered) |
| **False positive rate** | 100% (5/5 non-clinical) |
| **Overall page quality** | FAIL — but low impact (only 5 spans) |
