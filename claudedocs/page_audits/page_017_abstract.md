# Page 17 Audit — Abstract

| Field | Value |
|-------|-------|
| **Page** | 17 (PDF page S16) |
| **Content Type** | Abstract — guideline summary |
| **Extracted Spans** | 4 (T1: 3, T2: 1) |
| **Channels** | B (Drug Dictionary), C (Grammar/Regex) |
| **Disagreements** | 0 |
| **Review Status** | PENDING: 4 |
| **Risk** | First page with potential clinical references — but only abstract-level |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — span count corrected (5→4), channels corrected (D,B,C→B,C), T1 count corrected (4→3), phantom D-channel span removed |

---

## Source PDF Content

Page 17 is the **Abstract**, providing a high-level summary of the guideline:
- Describes the scope: comprehensive care for patients with diabetes and CKD
- Mentions key drug classes covered: SGLT2 inhibitors, GLP-1 receptor agonists, metformin
- References use of GRADE evidence system
- Summarizes that the guideline covers glycemic monitoring, targets, lifestyle interventions, glucose-lowering therapies, and comprehensive management approaches
- Notes HbA1c targets and monitoring

---

## Extraction Analysis

### Channel Breakdown (4 spans)

| Channel | Spans | T1 | T2 | Notes |
|---------|-------|----|----|-------|
| **B (Drug Dictionary)** | #1–3 | 3 | 0 | Drug class names matched from abstract text |
| **C (Grammar/Regex)** | #4 | 0 | 1 | Lab test name regex match |

**Note:** Previous audit incorrectly listed D (Table Decomp) channel. Raw data confirms NO D-channel spans on this page.

### What Was Extracted (4 spans)

| # | Channel | Conf | Tier | Text | Assessment |
|---|---------|------|------|------|------------|
| 1 | B | 100% | T1 | GLP-1 receptor agonist | **FALSE POSITIVE** — drug class name mention in abstract, no dosing/threshold |
| 2 | B | 100% | T1 | metformin | **FALSE POSITIVE** — drug name mention in abstract, no safety context |
| 3 | B | 100% | T1 | SGLT2 inhibitor | **FALSE POSITIVE** — drug class name mention in abstract, no dosing/threshold |
| 4 | C | 85% | T2 | HbA1c | T3 candidate — lab test name mention in abstract |

### Tier Assignment Issues

| Current Tier | Count | Should Be | Reason |
|-------------|-------|-----------|--------|
| **T1** | 3 | **T3** | Drug names mentioned in abstract summary — no associated thresholds, doses, or contraindications |
| **T2** | 1 | **T3** | Lab test name in abstract context |

### Severity: MODERATE — UNDER-EXTRACTION + TIER MISASSIGNMENT

---

## Specific Problems

### Problem 1: Only Drug/Lab Names Extracted from Abstract
The abstract summarizes the entire guideline's scope, but only 4 spans were extracted — all just drug/lab names without any of the surrounding summary statements. The abstract text like "This guideline covers comprehensive care..." was not extracted.

### Problem 2: Drug Name Mentions ≠ T1 Safety Facts
"SGLT2 inhibitor", "GLP-1 receptor agonist", and "metformin" appearing in an abstract are not safety facts. A T1 span requires actionable safety information like "discontinue SGLT2i when eGFR <20" — not just the drug name in a summary paragraph.

### Problem 3: F-Channel Missing
The F (NuExtract LLM) channel did not extract any spans from this page, despite the abstract containing substantial narrative text about the guideline's scope. This represents an F-channel dropout on an important page.

### Problem 4: Significant Missing Content
The abstract contains key summary statements that were NOT extracted:
- Summary of recommendation strength methodology
- Scope of clinical topics covered (glycemic monitoring, lifestyle, therapies)
- Overall approach to diabetes and CKD management
- These would appropriately be T3 spans

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Page Decision** | **FLAG** — Only drug names extracted from a content-rich abstract |
| **T1 spans** | All 3 → T3 (drug name mentions without clinical context) |
| **T2 spans** | 1 → T3 (lab test name mention) |
| **Missing content** | Abstract summary statements not extracted (~T3 content) |
| **Root cause** | 1. B channel matches known drug names regardless of clinical context; 2. F channel completely absent from this page (dropout); 3. No D-channel content exists despite prior audit claim |

---

## Completeness Score

| Metric | Score |
|--------|-------|
| **Extraction completeness** | ~15% (only drug/lab names captured from a full abstract) |
| **Tier accuracy** | 0% (0/4 correctly tiered) |
| **False positive rate** | 100% for T1 (3/3 are just drug name mentions) |
| **Missing T1 content** | 0 (abstract doesn't contain T1-level safety facts) |
| **Missing T3 content** | ~10 spans worth of abstract summary text |
| **F-channel dropout** | Yes — no F-channel extraction on this content-rich page |
| **Overall page quality** | FAIL — under-extracted and mistiered |
