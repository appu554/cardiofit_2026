# Page 7 Audit — Table of Contents (Supplementary Material cont. — SoF Tables S63–S94)

| Field | Value |
|-------|-------|
| **Page** | 7 (PDF page S6) |
| **Content Type** | Table of Contents — Supplementary Material continuation (SoF Tables S63–S94) |
| **Extracted Spans** | 122 (T1: 96, T2: 26) |
| **Channels** | B (Drug Dictionary), C (Grammar/Regex), D (Table Decomp), F (NuExtract LLM) |
| **Disagreements** | 0 |
| **Review Status** | CONFIRMED: 1, PENDING: 121 |
| **Risk** | No clinical content — ToC only |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — span count corrected (139→122), tier breakdown corrected (T1:110→96, T2:29→26), drug name counts corrected, false 0.25 U/kg/d span removed |

---

## 1. OVERALL VERDICT

**Extraction clinically correct?** → **NO**

This page is the final ToC / Supplementary Material page listing SoF tables S63–S94. It contains no clinical guidance, no treatment thresholds, no monitoring instructions, and no safety rules.

All 122 extracted spans are non-clinical ToC content and should be classified as T3.

- 96 false-positive T1 spans (86 B-channel drug names + 10 D-channel SoF table titles)
- 26 false-positive T2 spans
- 0 correct T1 spans
- 0 correct T2 spans

**Notable**: Page 7 has the highest T1 count (96) of any front matter page because the B channel matched every drug name appearing in SoF table titles. Span #82 ("DPP-4 inhibitor") has status CONFIRMED — a prior reviewer confirmed a false-positive T1 classification.

---

## 2. SOURCE PDF CONTENT

Page 7 is the final supplementary material ToC page, listing SoF (Summary of Findings) tables S63–S94:

- Table S63–S65: Aspirin/antiplatelet agent outcomes
- Table S66–S67: Sulfonylurea outcomes (vs metformin, vs alpha-glucosidase inhibitor)
- Table S68–S72: Glitazone/aleglitazar outcomes
- Table S73–S78: Insulin regimen comparisons (glulisine/glargine dose comparisons, degludec combinations)
- Table S79–S80: SGLT2i vs gliclazide, SGLT2i vs GLP-1 RA
- Table S81: GLP-1 RA and insulin vs insulin
- Table S82–S86: Liraglutide vs sitagliptin/linagliptin, sitagliptin vs linagliptin, omarigliptin vs linagliptin
- Table S87: Glitazone vs placebo (G3a–G5)
- Table S88: Kidney transplant recipients — intensive glucose control
- Table S89–S94: Self-management, care models, prompting systems, community-based care

Also contains Appendix F: Figure 36 Search yield diagram, and abbreviation expansions.

---

## 3. EXTRACTION ANALYSIS

### B-Channel Drug Names (86 spans — ALL T1, ALL false positives)

| Drug Name | Count | Span #s |
|-----------|-------|---------|
| insulin | 27 | #29–36, #49–51, #53–54, #57, #59, #61–63, #68–69, #77–78, #80–81, #84–86 |
| ACEi / ACEI | 10 | #2, #9, #12, #14, #19–20, #22, #24, #26, #28 |
| ARB | 9 | #3, #8, #10–11, #13, #15, #23, #25, #27 |
| liraglutide | 7 | #52, #55–56, #58, #60, #70, #72 |
| SGLT2i | 6 | #4, #16–18, #64–65 |
| thiazolidinedione | 4 | #37–38, #40–41 |
| MRA | 4 | #5–7, #21 |
| sitagliptin / Sitagliptin | 4 | #45, #47, #71, #74 |
| linagliptin / Linagliptin | 4 | #73, #75–76, #79 |
| GLP-1 RA | 3 | #1, #66–67 |
| sulfonylurea | 3 | #39, #42, #44 |
| DPP-4 inhibitor | 2 | #82 (CONFIRMED), #83 |
| metformin | 1 | #43 |
| glipizide | 1 | #46 |
| pioglitazone | 1 | #48 |

### D-Channel SoF Table Titles (10 spans — ALL T1, ALL false positives)

| Span # | Table | Content |
|--------|-------|---------|
| #87 | S88 | Kidney transplant recipients — intensive insulin therapy |
| #88 | S80 | SGLT2i versus GLP-1 RA |
| #89 | S84 | Sitagliptin versus linagliptin |
| #90 | S86 | Omarigliptin versus linagliptin |
| #91 | S78 | Insulin degludec versus insulin glargine (CKD G3a-G5) |
| #92 | S67 | Sulfonylurea versus alpha-glucosidase inhibitor (CKD G1-G2) |
| #93 | S86 | Omarigliptin versus linagliptin (duplicate of #90) |
| #94 | S66 | Sulfonylurea versus metformin (CKD G1-G2) |
| #95 | S72 | Aleglitazar versus pioglitazone (CKD G3a-G5) |
| #96 | S76 | Insulin degludec and liraglutide versus placebo (CKD G3a-G5) |

### D-Channel Labels (4 spans — T2)

- `"care"` ×2 (#97, #99) — Truncated text fragment
- Table S92 (#98) — Self-management SoF table title
- Table S93 (#100) — Community-based care SoF table title

### F-Channel (1 span — T2)

- Figure 36 Search yield diagram (#101)

### C-Channel Lab/Dose Fragments (21 spans — T2)

| Category | Count | Span #s |
|----------|-------|---------|
| HbA1c | 6 | #103–108 |
| Numeric doses (8.4 g, 18.6 g, 33.6 g, 1.5 mg, 325 mg, 2 g, 1 g) | 12 | #109–119, #121–122 |
| eGFR | 1 | #102 |
| sodium | 1 | #120 |
| 1 g (duplicate) | 1 | #122 |

---

## 4. TIER CORRECTIONS

### T1 → T3 (all 96 spans)

- 86 B-channel drug names: standalone drug names from SoF table title text — no dosing, thresholds, or safety context
- 10 D-channel SoF table titles: ToC references to supplementary tables — not the actual trial data

### T2 → T3 or REJECT (all 26 spans)

- 4 D-channel labels: truncated text fragments and SoF table titles
- 1 F-channel: Figure 36 reference
- 21 C-channel: lab test abbreviations and numeric doses from SoF table title fragments (e.g., "8.4 g", "18.6 g" are dosing values from insulin regimen comparison table titles, not clinical recommendations)

---

## 5. SPECIFIC PROBLEMS

### Problem 1: Massive T1 Inflation from Drug Dictionary
96 of 122 spans are T1 — the highest false-positive T1 count in the entire front matter. The B channel matched every drug name in SoF table titles: insulin (27×), ACEi/ACEI (10×), ARB (9×), liraglutide (7×), SGLT2i (6×), and 10 other drug names. None have clinical context.

### Problem 2: SoF Table Titles as Clinical Facts
D-channel SoF table titles like "Table S80. SoF table: SGLT2i versus GLP-1 RA" were tagged T1. These describe comparison studies but contain no outcome data, no thresholds, and no safety findings.

### Problem 3: Numeric Dose Fragments from Table Titles
The C channel extracted numeric doses like "8.4 g", "18.6 g", "33.6 g", "1.5 mg", "325 mg", "2 g", "1 g" — these appear to be from SoF table title fragments describing drug comparisons (e.g., insulin dose ranges, aspirin doses). They look clinically relevant but are ToC pointers, not dosing recommendations.

### Problem 4: Duplicate SoF Table Title
Table S86 (Omarigliptin versus linagliptin) appears twice as T1 spans (#90, #93) — D-channel duplicate extraction.

### Problem 5: CONFIRMED False Positive
Span #82 ("DPP-4 inhibitor") has status CONFIRMED — a prior reviewer confirmed a false-positive T1 classification. This confirmation is incorrect and should be overridden (same issue as page 4 span #1 "ACEi").

---

## 6. CRITICAL SAFETY FINDINGS

**None.** This page contains:
- No stop/hold criteria
- No dose modification thresholds
- No monitoring requirements
- No eGFR cutoffs
- No contraindications

---

## 7. L1_RECOVERY GOLD STANDARD SPANS

**None.** This page contains zero extractable clinical guidance. No L1_RECOVERY candidates.

---

## 8. COMPLETENESS SCORE

| Metric | Value |
|--------|-------|
| **True T1 content on page** | 0 (none expected) |
| **True T1 captured** | N/A |
| **False-positive T1** | 96/96 (100%) |
| **False-positive T2** | 26/26 (100%) |
| **Correct tier assignments** | 0/122 (0%) |
| **Noise ratio** | 100% — all 122 spans are non-clinical ToC content |
| **Prior review errors** | 1 (span #82 CONFIRMED as false-positive T1) |
| **Overall quality** | **POOR / FAIL** — requires bulk rejection |

---

## 9. REVIEWER RECOMMENDATION

| Action | Details |
|--------|---------|
| **Decision** | **ESCALATE** — All 122 spans require re-tiering to T3 or bulk rejection |
| **T1 correction** | All 96 → T3 (86 B-channel drug names + 10 D-channel SoF table titles) |
| **T2 correction** | All 26 → T3 (lab names, numeric doses, labels, figure reference) |
| **Prior review override** | Span #82 CONFIRMED status should be overridden — "DPP-4 inhibitor" in ToC is not a clinical safety fact |
| **Missing content** | None (no clinical content expected on ToC page) |
| **Root cause** | (1) B channel matches drug names without clinical context — produces 86 false T1 from a single ToC page; (2) D channel treats SoF table titles as clinical data; (3) C channel extracts numeric doses from table title fragments without ToC awareness |
| **Pipeline recommendation** | (1) Flag pages 1–7 as front matter in preprocessing to skip extraction on non-clinical pages; (2) B channel should require co-occurrence with threshold/action terms before assigning T1 |
