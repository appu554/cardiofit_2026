# KDIGO 2024 CKD Delta — Per-Page Extraction Audit Report

**Job ID**: `f172f6a9-7733-4352-a0aa-43707fdb46c8`
**PDF**: KDIGO-2024-CKD-Delta-53pages.pdf (53 pages)
**Pipeline Version**: 4.2.4
**Audit Date**: 2026-03-07
**Total Merged Spans**: 939 | **Pages with spans**: 41/53 | **Pages with zero spans**: 12

---

## Executive Summary

The extraction pipeline captures 41 of 53 pages (77%). However, the audit identified **10 missing Recommendations** and **32 missing Practice Points** — high-priority clinical content that should have been extracted. The 12 zero-span pages contain significant clinical content including 9 Recommendations and 21 Practice Points that are entirely absent from the extraction.

### Issue Categories

| Category | Count | Severity |
|----------|-------|----------|
| Zero-span pages with clinical content | 12 | CRITICAL |
| Missing Recommendations (across all pages) | 10 | HIGH |
| Missing Practice Points (across all pages) | 32 | HIGH |
| Missing dosing thresholds | ~45 instances | MEDIUM |
| NOISE spans that are legitimate table cells | 44 | LOW |

---

## Zero-Span Pages (12 pages)

These pages have substantial PDF content (2,019-6,231 chars) but zero merged spans. The root cause is that the normalized_text PAGE markers create offset boundaries, and these pages fall into gaps where Channel A's section parsing doesn't create sections that attract Channel B-G extraction.

### CRITICAL: Pages with Recommendations/Practice Points but zero spans

| Page | Content | Missing Items |
|------|---------|---------------|
| 1 | Ch4 Medication Management intro | PP 4.1.1, PP 4.1.2, PP 4.1.3 |
| 3 | Insulin barriers, erythropoietin | PP 4.1.4 |
| 5 | Drug dosing in CKD, mGFR | PP 4.3.1 |
| 12 | BP targets, RASi/SGLT2i | **Rec 3.6.1-3.6.4**, PP 3.6.1-3.6.7 (13 items!) |
| 14 | SGLT2i benefit, ketoacidosis risk | **Rec 1.3.1** |
| 21 | LDL-cholesterol, statin therapy | **Rec 3.15.2.1**, PP 3.15.1.4-3.15.1.5 |
| 24 | Intensive management benefit | PP 3.15.3.1 |
| 25 | Coronary revascularization | PP 3.16.1 |
| 26 | Thromboembolic events, AFib | **Rec 3.16.1** |
| 32 | Public health CKD detection | PP 1.1.1.2 |
| 53 | Gout management in CKD | **Rec 3.14.2**, PP 3.14.3, PP 3.14.4 |

### LOW: Pages with zero spans but minimal clinical content

| Page | Content | Assessment |
|------|---------|------------|
| 22 | Continuation text about bleeding risk | Only drug names, no structured recommendations — acceptable gap |

---

## Missing Clinical Content on Pages WITH Spans

These pages have some spans but are missing specific Recommendations or Practice Points.

| Page | Spans | Missing Items |
|------|-------|---------------|
| 6 | 15 | PP 4.3.2 |
| 7 | 15 | PP 4.3.1.1, PP 4.3.1.2 |
| 8 | 10 | PP 4.4.1.1, PP 4.4.1.2 + 5 dosing values + 5 thresholds |
| 10 | 1 | **Rec 3.3.1.1** (only 1 span on page with dense content), PP 3.3.1.4-3.3.1.5 |
| 11 | 14 | **Rec 3.4.1**, PP 3.4.1 |
| 16 | 29 | PP 3.11.1.1 + dosing values |
| 18 | 54 | PP 3.11.5.1 |
| 31 | 47 | PP 1.1.1.1 |
| 33 | 32 | **Rec 1.1.2.1** |
| 34 | 15 | PP 1.1.3.1 |
| 38 | 3 | PP 1.2.1.1 (only 3 spans on content-rich page) |
| 39 | 5 | **Rec 1.2.2.1** |
| 40 | 22 | PP 1.2.2.1 |
| 44 | 11 | PP 1.2.2.4, PP 1.2.2.5 |
| 45 | 7 | PP 1.2.2.6, PP 1.2.2.7 |
| 46 | 38 | PP 1.2.3.1, PP 1.2.3.2 |
| 48 | 71 | PP 3.10.1, PP 3.10.2 |
| 51 | 5 | **Rec 3.14.1** |
| 52 | 6 | PP 3.14.1 |

---

## Root Cause Analysis

### Why 12 pages have zero spans

Investigation reveals that raw spans from Channels B, C, E, F, G exist for pages 1, 3, 5 (pre-fix page numbers mapped there), but after the page_map fix, these spans correctly relocated to their true pages. The normalized_text for pages 1, 3, 5 etc. shows 0 chars — meaning the PAGE marker exists but no text content follows until the next PAGE marker.

**Root cause**: The normalized text from Channel 0 creates empty page slots when the L1 markdown has PAGE markers with content following on the next marker's section. This is a **text segmentation issue** — the content exists in the markdown but the PAGE markers don't bracket it correctly for per-page extraction.

### Why Recommendations/Practice Points are missing on pages WITH spans

Channel C (grammar/regex) is the primary channel that extracts Recommendation and Practice Point labels. The audit shows:
- Channel C extracts the pattern "Practice Point X.Y.Z" but sometimes misses the **body text** that follows
- The merger clusters nearby spans but may miss the full recommendation text when it spans multiple paragraphs
- Some PP labels span line breaks (e.g., "Practice\nPoint\n3.11.5.1") which may not match Channel C's regex

### NOISE Classification

All 44 NOISE spans are single-channel Channel D (table cell) spans with very short text:
- Standalone lab names: "Creatinine", "Hemoglobin", "Potassium" (8 instances)
- Standalone drug class names: "SGLT2i", "NSAIDs" (6 instances)
- Bare numbers from table cells: "0.11", "0.37" (5 instances)
- These are correctly classified as NOISE — they lack clinical context without the table structure

---

## Tier Assignment Analysis

| Tier | Count | Assessment |
|------|-------|------------|
| TIER_1 | 347 (37%) | Multi-channel corroborated clinical facts — correctly high confidence |
| TIER_2 | 548 (58%) | Single or dual channel — reasonable default for review |
| NOISE | 44 (5%) | All Channel D table cell fragments — correctly filtered |

**No misclassified TIER_1 spans found** — all TIER_1 spans have text length > 20 chars and multi-channel support.

---

## Page-by-Page Detail

### Pages 1-10

| Page | Chars | Spans | T1 | T2 | N | Disagr | Status |
|------|-------|-------|----|----|---|--------|--------|
| 1 | 5128 | 0 | 0 | 0 | 0 | 0 | CRITICAL: 3 Practice Points missing |
| 2 | 5044 | 23 | 0 | 23 | 0 | 0 | OK (table content) |
| 3 | 5079 | 0 | 0 | 0 | 0 | 0 | CRITICAL: PP 4.1.4 missing |
| 4 | 5673 | 33 | 33 | 0 | 0 | 33 | OK (high disagreement = dense multi-channel) |
| 5 | 4877 | 0 | 0 | 0 | 0 | 0 | CRITICAL: PP 4.3.1 missing |
| 6 | 5178 | 15 | 15 | 0 | 0 | 15 | PARTIAL: PP 4.3.2 missing |
| 7 | 5852 | 15 | 1 | 12 | 2 | 1 | PARTIAL: 2 PPs + contraindication text missing |
| 8 | 5907 | 10 | 0 | 10 | 0 | 0 | POOR: Only 10 spans, missing 5 doses + 5 thresholds + 2 PPs |
| 9 | 2493 | 19 | 19 | 0 | 0 | 18 | OK |
| 10 | 5921 | 1 | 1 | 0 | 0 | 1 | POOR: Only 1 span on dense clinical page |

### Pages 11-20

| Page | Chars | Spans | T1 | T2 | N | Disagr | Status |
|------|-------|-------|----|----|---|--------|--------|
| 11 | 5355 | 14 | 0 | 7 | 7 | 0 | PARTIAL: Rec 3.4.1 + PP 3.4.1 missing |
| 12 | 5237 | 0 | 0 | 0 | 0 | 0 | CRITICAL: 4 Recs + 9 PPs missing |
| 13 | 3917 | 41 | 41 | 0 | 0 | 41 | OK (dense table/figure content) |
| 14 | 6231 | 0 | 0 | 0 | 0 | 0 | CRITICAL: Rec 1.3.1 missing |
| 15 | 5808 | 9 | 9 | 0 | 0 | 9 | OK |
| 16 | 5707 | 29 | 1 | 23 | 5 | 1 | PARTIAL: PP + dosing missing |
| 17 | 3647 | 43 | 23 | 19 | 1 | 22 | MINOR: 1 dosing value missing |
| 18 | 4588 | 54 | 18 | 32 | 4 | 18 | PARTIAL: PP 3.11.5.1 missing |
| 19 | 4548 | 57 | 1 | 55 | 1 | 1 | MINOR: 1 dosing value missing |
| 20 | 3768 | 7 | 7 | 0 | 0 | 7 | PARTIAL: dosing + threshold missing |

### Pages 21-30

| Page | Chars | Spans | T1 | T2 | N | Disagr | Status |
|------|-------|-------|----|----|---|--------|--------|
| 21 | 5836 | 0 | 0 | 0 | 0 | 0 | CRITICAL: Rec 3.15.2.1 + 2 PPs missing |
| 22 | 2019 | 0 | 0 | 0 | 0 | 0 | LOW: Continuation text only |
| 23 | 5742 | 16 | 16 | 0 | 0 | 16 | MINOR: 1 dosing value |
| 24 | 6134 | 0 | 0 | 0 | 0 | 0 | CRITICAL: PP 3.15.3.1 missing |
| 25 | 5858 | 0 | 0 | 0 | 0 | 0 | CRITICAL: PP 3.16.1 missing |
| 26 | 5147 | 0 | 0 | 0 | 0 | 0 | CRITICAL: Rec 3.16.1 missing |
| 27 | 5542 | 8 | 8 | 0 | 0 | 8 | PARTIAL: dosing + threshold |
| 28 | 5163 | 2 | 2 | 0 | 0 | 2 | PARTIAL: threshold missing |
| 29 | 4334 | 6 | 6 | 0 | 0 | 6 | PARTIAL: dosing + drug names |
| 30 | 2249 | 19 | 1 | 17 | 1 | 1 | MINOR: 1 dosing value |

### Pages 31-40

| Page | Chars | Spans | T1 | T2 | N | Disagr | Status |
|------|-------|-------|----|----|---|--------|--------|
| 31 | 4416 | 47 | 0 | 46 | 1 | 0 | PARTIAL: PP 1.1.1.1 missing |
| 32 | 6211 | 0 | 0 | 0 | 0 | 0 | CRITICAL: PP 1.1.1.2 missing |
| 33 | 4114 | 32 | 0 | 31 | 1 | 0 | PARTIAL: Rec 1.1.2.1 missing |
| 34 | 5565 | 15 | 15 | 0 | 0 | 14 | PARTIAL: PP 1.1.3.1 + 3 PPs missing |
| 35 | 5409 | 22 | 22 | 0 | 0 | 22 | OK |
| 36 | 5528 | 17 | 2 | 15 | 0 | 2 | PARTIAL: dosing + contraindication |
| 37 | 4848 | 9 | 9 | 0 | 0 | 9 | OK |
| 38 | 5800 | 3 | 3 | 0 | 0 | 3 | POOR: Only 3 spans, PP 1.2.1.1 missing |
| 39 | 3452 | 5 | 5 | 0 | 0 | 5 | PARTIAL: Rec 1.2.2.1 missing |
| 40 | 5706 | 22 | 4 | 16 | 2 | 4 | PARTIAL: PP 1.2.2.1 missing |

### Pages 41-53

| Page | Chars | Spans | T1 | T2 | N | Disagr | Status |
|------|-------|-------|----|----|---|--------|--------|
| 41 | 4523 | 61 | 7 | 53 | 1 | 7 | OK (high span count) |
| 42 | 5503 | 18 | 5 | 13 | 0 | 5 | PARTIAL: PP missing |
| 43 | 6349 | 8 | 8 | 0 | 0 | 8 | PARTIAL: PP missing |
| 44 | 4579 | 11 | 11 | 0 | 0 | 11 | PARTIAL: 2 PPs + dosing + contraindication |
| 45 | 6037 | 7 | 7 | 0 | 0 | 7 | PARTIAL: 3 PPs + dosing |
| 46 | 5946 | 38 | 12 | 26 | 0 | 12 | PARTIAL: 2 PPs + dosing |
| 47 | 6249 | 8 | 8 | 0 | 0 | 8 | OK |
| 48 | 5036 | 71 | 5 | 60 | 6 | 5 | PARTIAL: PP 3.10.1 + 3.10.2 |
| 49 | 5743 | 59 | 7 | 46 | 6 | 7 | PARTIAL: dosing + thresholds |
| 50 | 5040 | 54 | 4 | 44 | 6 | 4 | PARTIAL: dosing + thresholds |
| 51 | 5117 | 5 | 5 | 0 | 0 | 5 | PARTIAL: Rec 3.14.1 missing |
| 52 | 5516 | 6 | 6 | 0 | 0 | 6 | PARTIAL: PP 3.14.1 + dosing |
| 53 | 5893 | 0 | 0 | 0 | 0 | 0 | CRITICAL: Rec 3.14.2 + 2 PPs missing |

---

## Recommended Fixes (Priority Order)

### P1 — Fix zero-span pages (12 pages, ~22 Recs/PPs)
The 12 zero-span pages are caused by text segmentation in the normalized text. The PAGE markers exist but content between them is empty. This requires investigating Channel 0 (normalizer) text flow — the content may be absorbed into adjacent pages' text blocks.

**Action**: Check if the 12 missing pages' content appears in the normalized_text under adjacent page offsets. If so, the issue is in `_build_page_map` boundary calculation.

### P2 — Improve Channel C Recommendation/PP extraction (32 PPs, 10 Recs)
Channel C's regex for Practice Points may not handle multi-line PP labels (e.g., "Practice\nPoint\n3.11.5.1"). Also, the extraction may find the label but miss the body text.

**Action**:
1. Update Channel C regex to handle line breaks within "Practice Point" and "Recommendation" labels
2. Extend extraction to capture 200+ chars after the label as the recommendation body

### P3 — Add dosing value extraction (45+ instances)
Dosing values like "25 mg/day", "0.5 mg/kg" appear in the PDF but aren't extracted. These are valuable for KB-1 drug rules.

**Action**: Enhance Channel C grammar patterns for dosing values, or add dedicated dosing extraction to Channel B.

### P4 — Address sparse extraction pages (pages 10, 38 with 1-3 spans)
Pages 10 and 38 have only 1-3 spans despite 5,800-5,900 chars of content. This suggests the normalized text for these pages is very short (content may have shifted to adjacent pages).

**Action**: Investigate page boundary accuracy for pages 10 and 38 in the page_map.

---

## Summary Metrics

| Metric | Value |
|--------|-------|
| Pages with full extraction (OK) | 9/53 (17%) |
| Pages with partial extraction | 20/53 (38%) |
| Pages with poor extraction (1-3 spans) | 2/53 (4%) |
| Pages with zero spans | 12/53 (23%) |
| Pages with minor issues only | 10/53 (19%) |
| Total missing Recommendations | 10 |
| Total missing Practice Points | 32 |
| NOISE correctly classified | 44/44 (100%) |
| TIER_1 accuracy | No misclassifications detected |
